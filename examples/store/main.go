package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	grove "github.com/wispberry-tech/grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// --- Types ---

type Category struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (c Category) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return c.Name, true
	case "slug":
		return c.Slug, true
	case "description":
		return c.Description, true
	}
	return nil, false
}

type Product struct {
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Price        int      `json:"price"`
	SalePrice    int      `json:"sale_price"`
	Description  string   `json:"description"`
	Body         string   `json:"body"`
	ImageURL     string   `json:"image_url"`
	CategorySlug string   `json:"category_slug"`
	Rating       float64  `json:"rating"`
	ReviewCount  int      `json:"review_count"`
	Colors       []string `json:"colors"`
	Sizes        []string `json:"sizes"`
	InStock      bool     `json:"in_stock"`
	Featured     bool     `json:"featured"`
}

func (p Product) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return p.Name, true
	case "slug":
		return p.Slug, true
	case "price":
		return p.Price, true
	case "sale_price":
		return p.SalePrice, true
	case "on_sale":
		return p.SalePrice > 0, true
	case "effective_price":
		if p.SalePrice > 0 {
			return p.SalePrice, true
		}
		return p.Price, true
	case "description":
		return p.Description, true
	case "body":
		return p.Body, true
	case "image_url":
		return p.ImageURL, true
	case "category_slug":
		return p.CategorySlug, true
	case "category":
		if c, ok := categoryMap[p.CategorySlug]; ok {
			return c, true
		}
		return nil, false
	case "rating":
		return p.Rating, true
	case "review_count":
		return p.ReviewCount, true
	case "colors":
		out := make([]any, len(p.Colors))
		for i, c := range p.Colors {
			out[i] = c
		}
		return out, true
	case "sizes":
		out := make([]any, len(p.Sizes))
		for i, s := range p.Sizes {
			out[i] = s
		}
		return out, true
	case "in_stock":
		return p.InStock, true
	case "featured":
		return p.Featured, true
	}
	return nil, false
}

// CartItem is what we store in the cookie.
type CartItem struct {
	ProductSlug string `json:"product_slug"`
	Quantity    int    `json:"quantity"`
}

// CartEntry is the resolved version passed to templates.
type CartEntry struct {
	Product  Product
	Quantity int
}

func (e CartEntry) GroveResolve(key string) (any, bool) {
	switch key {
	case "product":
		return e.Product, true
	case "quantity":
		return e.Quantity, true
	case "line_total":
		price := e.Product.Price
		if e.Product.SalePrice > 0 {
			price = e.Product.SalePrice
		}
		return price * e.Quantity, true
	}
	return nil, false
}

// --- Data ---

var (
	categories  []Category
	categoryMap map[string]Category
	products    []Product
	productMap  map[string]Product
)

func loadJSON(baseDir, filename string, v any) {
	data, err := os.ReadFile(filepath.Join(baseDir, "data", filename))
	if err != nil {
		log.Fatalf("Failed to load %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		log.Fatalf("Failed to parse %s: %v", filename, err)
	}
}

func loadData(baseDir string) {
	loadJSON(baseDir, "categories.json", &categories)
	loadJSON(baseDir, "products.json", &products)

	categoryMap = make(map[string]Category)
	for _, c := range categories {
		categoryMap[c.Slug] = c
	}
	productMap = make(map[string]Product)
	for _, p := range products {
		productMap[p.Slug] = p
	}
}

// --- Cart (cookie-based) ---

const cartCookieName = "grove_cart"

func getCart(r *http.Request) []CartItem {
	cookie, err := r.Cookie(cartCookieName)
	if err != nil {
		return nil
	}
	decoded, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil
	}
	var items []CartItem
	if err := json.Unmarshal([]byte(decoded), &items); err != nil {
		return nil
	}
	return items
}

func setCart(w http.ResponseWriter, items []CartItem) {
	data, _ := json.Marshal(items)
	http.SetCookie(w, &http.Cookie{
		Name:  cartCookieName,
		Value: url.QueryEscape(string(data)),
		Path:  "/",
	})
}

func cartCount(r *http.Request) int {
	count := 0
	for _, item := range getCart(r) {
		count += item.Quantity
	}
	return count
}

func resolveCart(items []CartItem) []CartEntry {
	var entries []CartEntry
	for _, item := range items {
		if p, ok := productMap[item.ProductSlug]; ok {
			entries = append(entries, CartEntry{Product: p, Quantity: item.Quantity})
		}
	}
	return entries
}

// --- Helpers ---

func productsToAny(pp []Product) []any {
	out := make([]any, len(pp))
	for i, p := range pp {
		out[i] = p
	}
	return out
}

func categoriesToAny() []any {
	out := make([]any, len(categories))
	for i, c := range categories {
		out[i] = c
	}
	return out
}

func cartEntriesToAny(entries []CartEntry) []any {
	out := make([]any, len(entries))
	for i, e := range entries {
		out[i] = e
	}
	return out
}

func featuredProducts() []Product {
	var out []Product
	for _, p := range products {
		if p.Featured && p.InStock {
			out = append(out, p)
		}
	}
	return out
}

func filterProducts(r *http.Request) ([]Product, map[string]string) {
	filtered := make([]Product, len(products))
	copy(filtered, products)
	activeFilters := make(map[string]string)

	if cat := r.URL.Query().Get("category"); cat != "" {
		activeFilters["category"] = cat
		var out []Product
		for _, p := range filtered {
			if p.CategorySlug == cat {
				out = append(out, p)
			}
		}
		filtered = out
	}

	if minStr := r.URL.Query().Get("min_price"); minStr != "" {
		if min, err := strconv.Atoi(minStr); err == nil {
			activeFilters["min_price"] = minStr
			var out []Product
			for _, p := range filtered {
				price := p.Price
				if p.SalePrice > 0 {
					price = p.SalePrice
				}
				if price >= min {
					out = append(out, p)
				}
			}
			filtered = out
		}
	}

	if maxStr := r.URL.Query().Get("max_price"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil {
			activeFilters["max_price"] = maxStr
			var out []Product
			for _, p := range filtered {
				price := p.Price
				if p.SalePrice > 0 {
					price = p.SalePrice
				}
				if price <= max {
					out = append(out, p)
				}
			}
			filtered = out
		}
	}

	if sortBy := r.URL.Query().Get("sort"); sortBy != "" {
		activeFilters["sort"] = sortBy
		switch sortBy {
		case "price-asc":
			sort.Slice(filtered, func(i, j int) bool {
				pi, pj := filtered[i].Price, filtered[j].Price
				if filtered[i].SalePrice > 0 {
					pi = filtered[i].SalePrice
				}
				if filtered[j].SalePrice > 0 {
					pj = filtered[j].SalePrice
				}
				return pi < pj
			})
		case "price-desc":
			sort.Slice(filtered, func(i, j int) bool {
				pi, pj := filtered[i].Price, filtered[j].Price
				if filtered[i].SalePrice > 0 {
					pi = filtered[i].SalePrice
				}
				if filtered[j].SalePrice > 0 {
					pj = filtered[j].SalePrice
				}
				return pi > pj
			})
		case "rating":
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].Rating > filtered[j].Rating
			})
		case "name":
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].Name < filtered[j].Name
			})
		}
	}

	return filtered, activeFilters
}

func searchProducts(query string) []Product {
	q := strings.ToLower(query)
	var out []Product
	for _, p := range products {
		if strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Description), q) {
			out = append(out, p)
		}
	}
	return out
}

func relatedProducts(product Product, limit int) []Product {
	var out []Product
	for _, p := range products {
		if p.Slug == product.Slug || !p.InStock {
			continue
		}
		if p.CategorySlug == product.CategorySlug {
			out = append(out, p)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

// --- Handlers ---

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := eng.Render(r.Context(), "index.grov", grove.Data{
			"featured":   productsToAny(featuredProducts()),
			"categories": categoriesToAny(),
			"cart_count": cartCount(r),
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func productsHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filtered, activeFilters := filterProducts(r)

		filtersAny := make(map[string]any)
		for k, v := range activeFilters {
			filtersAny[k] = v
		}

		result, err := eng.Render(r.Context(), "product-list.grov", grove.Data{
			"products":       productsToAny(filtered),
			"categories":     categoriesToAny(),
			"active_filters": filtersAny,
			"result_count":   len(filtered),
			"cart_count":     cartCount(r),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Products", "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func categoryHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		cat, ok := categoryMap[slug]
		if !ok {
			http.NotFound(w, r)
			return
		}
		var catProducts []Product
		for _, p := range products {
			if p.CategorySlug == slug {
				catProducts = append(catProducts, p)
			}
		}
		result, err := eng.Render(r.Context(), "category.grov", grove.Data{
			"category":   cat,
			"products":   productsToAny(catProducts),
			"cart_count": cartCount(r),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": cat.Name, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func productHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		product, ok := productMap[slug]
		if !ok {
			http.NotFound(w, r)
			return
		}
		cat := categoryMap[product.CategorySlug]
		related := relatedProducts(product, 4)

		result, err := eng.Render(r.Context(), "product.grov", grove.Data{
			"product":    product,
			"related":    productsToAny(related),
			"cart_count": cartCount(r),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": cat.Name, "href": "/category/" + cat.Slug},
				map[string]any{"label": product.Name, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func cartHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := resolveCart(getCart(r))
		result, err := eng.Render(r.Context(), "cart.grov", grove.Data{
			"items":      cartEntriesToAny(entries),
			"cart_count": cartCount(r),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Cart", "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func cartAddHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("product")
	qty, _ := strconv.Atoi(r.URL.Query().Get("qty"))
	if qty < 1 {
		qty = 1
	}
	if _, ok := productMap[slug]; !ok {
		http.NotFound(w, r)
		return
	}

	cart := getCart(r)
	found := false
	for i := range cart {
		if cart[i].ProductSlug == slug {
			cart[i].Quantity += qty
			found = true
			break
		}
	}
	if !found {
		cart = append(cart, CartItem{ProductSlug: slug, Quantity: qty})
	}
	setCart(w, cart)

	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/cart"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func cartRemoveHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("product")
	cart := getCart(r)
	var updated []CartItem
	for _, item := range cart {
		if item.ProductSlug != slug {
			updated = append(updated, item)
		}
	}
	setCart(w, updated)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func searchHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		results := searchProducts(q)
		result, err := eng.Render(r.Context(), "search.grov", grove.Data{
			"query":        q,
			"products":     productsToAny(results),
			"result_count": len(results),
			"cart_count":   cartCount(r),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Search", "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

// --- Response assembly ---

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body
	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

	var meta strings.Builder
	for name, content := range result.Meta {
		if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "property:") {
			meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
		} else {
			meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
		}
	}
	body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

// --- Main ---

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)

	loadData(baseDir)

	templateDir := filepath.Join(baseDir, "templates")
	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Coldfront Supply Co.")
	eng.SetGlobal("current_year", "2026")

	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		cents, _ := v.ToInt64()
		dollars := cents / 100
		remainder := int(math.Abs(float64(cents % 100)))
		return grove.StringValue(fmt.Sprintf("$%d.%02d", dollars, remainder)), nil
	}))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/products", productsHandler(eng))
	r.Get("/category/{slug}", categoryHandler(eng))
	r.Get("/product/{slug}", productHandler(eng))
	r.Get("/cart", cartHandler(eng))
	r.Get("/cart/add", cartAddHandler)
	r.Get("/cart/remove", cartRemoveHandler)
	r.Get("/search", searchHandler(eng))

	staticDir := filepath.Join(baseDir, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Serve colocated CSS and JS from component directories
	r.Handle("/css/*", http.StripPrefix("/css/", filteredFileServer(templateDir, ".css")))
	r.Handle("/js/*", http.StripPrefix("/js/", filteredFileServer(templateDir, ".js")))

	fmt.Println("Coldfront Supply Co. listening on http://localhost:3001")
	log.Fatal(http.ListenAndServe(":3001", r))
}

// filteredFileServer serves only files matching the given extension from dir.
// All other requests get a 404, preventing template source files from being served.
func filteredFileServer(dir, ext string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ext) {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Product{}
	_ interface{ GroveResolve(string) (any, bool) } = Category{}
	_ interface{ GroveResolve(string) (any, bool) } = CartEntry{}
)
