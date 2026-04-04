# Store Example Overhaul — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the store example from a 4-product demo into a realistic outdoor gear catalog with category browsing, filtering/sorting, search, and a cookie-based cart.

**Architecture:** Data loaded from JSON files. Products have categories, ratings, sizes, colors. Routes support query param filtering (`?category=`, `?sort=`, `?min_price=`, `?max_price=`) and text search (`?q=`). Cart stored in a cookie as JSON. Macros handle pricing display, star ratings, and filter UI. Custom Go-registered filters for currency formatting.

**Tech Stack:** Go 1.24, Grove template engine, Chi router, JSON data files

**Spec:** `docs/superpowers/specs/2026-04-04-examples-expansion-design.md` — Example 2: Store

---

### Task 1: Create JSON data files

**Files:**
- Create: `examples/store/data/categories.json`
- Create: `examples/store/data/products.json`

- [ ] **Step 1: Create the data directory**

```bash
mkdir -p examples/store/data
```

- [ ] **Step 2: Write categories.json**

```json
[
  {"name": "Camping", "slug": "camping", "description": "Tents, sleeping bags, and everything you need for a night under the stars."},
  {"name": "Hiking", "slug": "hiking", "description": "Boots, packs, and gear for trails of every difficulty."},
  {"name": "Cycling", "slug": "cycling", "description": "Bikes, helmets, lights, and accessories for road and trail."},
  {"name": "Running", "slug": "running", "description": "Shoes, apparel, and tech for runners of all levels."},
  {"name": "Climbing", "slug": "climbing", "description": "Harnesses, ropes, shoes, and protection for rock and ice."}
]
```

- [ ] **Step 3: Write products.json**

Create 20-25 products spread across all 5 categories. Each product has: name, slug, price (int, cents), sale_price (int, cents — 0 if not on sale), description (1 sentence), body (1-2 paragraphs HTML), image_url (placeholder like "/static/images/product-slug.jpg"), category_slug, rating (float 1.0-5.0), review_count (int), colors (string array), sizes (string array), in_stock (bool), featured (bool). Mark 4-5 products as featured, 2-3 as on sale, 1-2 as out of stock.

Example entries (create 20+ following this pattern):

```json
[
  {
    "name": "Alpine Pro Tent",
    "slug": "alpine-pro-tent",
    "price": 34999,
    "sale_price": 27999,
    "description": "Lightweight 2-person tent rated for 4-season use.",
    "body": "<p>The Alpine Pro is our most versatile tent, built for backpackers who don't want to compromise on weather protection. Double-wall construction with a full-coverage rainfly keeps you dry in any conditions, while the lightweight aluminum pole system makes setup quick even in wind.</p><p>At just 4.2 lbs trail weight, the Alpine Pro won't weigh you down on long approaches. The two vestibules provide gear storage, and the interior mesh panels keep condensation under control. Rated to withstand 50mph winds and heavy snowfall.</p>",
    "image_url": "/static/images/alpine-pro-tent.jpg",
    "category_slug": "camping",
    "rating": 4.7,
    "review_count": 234,
    "colors": ["Forest Green", "Storm Gray"],
    "sizes": ["2-Person", "3-Person"],
    "in_stock": true,
    "featured": true
  },
  {
    "name": "Trailrunner GTX Boots",
    "slug": "trailrunner-gtx-boots",
    "price": 18999,
    "sale_price": 0,
    "description": "Waterproof hiking boots with Vibram outsoles for all-terrain grip.",
    "body": "<p>Built for serious hikers who need reliable footwear in any conditions. The Trailrunner GTX features a waterproof-breathable membrane that keeps your feet dry without overheating. The Vibram Megagrip outsole provides exceptional traction on wet rock, loose gravel, and muddy trails.</p><p>The EVA midsole delivers cushioning for long days, while the reinforced toe cap protects against rocks and roots. Available in regular and wide widths to ensure a perfect fit.</p>",
    "image_url": "/static/images/trailrunner-gtx-boots.jpg",
    "category_slug": "hiking",
    "rating": 4.5,
    "review_count": 187,
    "colors": ["Brown", "Black", "Olive"],
    "sizes": ["7", "8", "9", "10", "11", "12", "13"],
    "in_stock": true,
    "featured": true
  },
  {
    "name": "Velocity Carbon Road Bike",
    "slug": "velocity-carbon-road-bike",
    "price": 249999,
    "sale_price": 0,
    "description": "Full carbon frame road bike with Shimano 105 groupset.",
    "body": "<p>The Velocity Carbon delivers race-level performance at an accessible price point. The full carbon monocoque frame weighs just 980g and features our aero tube profiles for reduced drag at speed. The Shimano 105 R7000 groupset provides precise, reliable shifting across the 22-speed range.</p><p>Equipped with tubeless-ready carbon wheels and 28mm tires for a balance of speed and comfort. Internal cable routing keeps the cockpit clean, and the threaded bottom bracket makes maintenance straightforward.</p>",
    "image_url": "/static/images/velocity-carbon-road-bike.jpg",
    "category_slug": "cycling",
    "rating": 4.8,
    "review_count": 56,
    "colors": ["Matte Black", "Racing Red"],
    "sizes": ["48cm", "51cm", "54cm", "56cm", "58cm"],
    "in_stock": true,
    "featured": true
  },
  {
    "name": "Cloudstrike Running Shoes",
    "slug": "cloudstrike-running-shoes",
    "price": 15999,
    "sale_price": 12799,
    "description": "Lightweight daily trainer with responsive foam and breathable mesh upper.",
    "body": "<p>The Cloudstrike is designed for runners who want a versatile daily trainer that can handle everything from easy recovery runs to tempo workouts. The nitrogen-infused foam midsole delivers a snappy, responsive ride without sacrificing cushioning on longer efforts.</p><p>The engineered mesh upper provides targeted breathability and a secure midfoot wrap. A rubber outsole with strategic placement offers durability where you need it while keeping weight to a minimum at 8.5oz.</p>",
    "image_url": "/static/images/cloudstrike-running-shoes.jpg",
    "category_slug": "running",
    "rating": 4.3,
    "review_count": 312,
    "colors": ["White/Blue", "Black/Volt", "Gray/Orange"],
    "sizes": ["7", "8", "8.5", "9", "9.5", "10", "10.5", "11", "12"],
    "in_stock": true,
    "featured": false
  },
  {
    "name": "Apex Climbing Harness",
    "slug": "apex-climbing-harness",
    "price": 8999,
    "sale_price": 0,
    "description": "Lightweight sport climbing harness with gear loops and adjustable leg loops.",
    "body": "<p>The Apex harness is built for sport climbers who value comfort on long routes. The FrameFit construction uses a laminated foam frame that distributes weight evenly across your waist and legs, eliminating pressure points during extended hangs.</p><p>Four molded gear loops provide easy access to your rack, and the speed-adjust buckle system lets you dial in the fit in seconds. Tie-in points are reinforced with abrasion-resistant fabric for extended durability.</p>",
    "image_url": "/static/images/apex-climbing-harness.jpg",
    "category_slug": "climbing",
    "rating": 4.6,
    "review_count": 98,
    "colors": ["Orange", "Blue"],
    "sizes": ["S", "M", "L", "XL"],
    "in_stock": true,
    "featured": true
  }
]
```

Create 15-20 more products following this pattern, spread evenly across all 5 categories. Include a few that are out of stock (`"in_stock": false`) and a few more on sale. Keep prices realistic for outdoor gear.

- [ ] **Step 4: Commit data files**

```bash
git add examples/store/data/
git commit -m "store: Add JSON data files for categories and products"
```

---

### Task 2: Rewrite main.go

**Files:**
- Modify: `examples/store/main.go`

- [ ] **Step 1: Write the complete main.go**

Replace the entire file. The new main.go includes:
- `Category` and `Product` structs with JSON tags and `GroveResolve`
- `CartItem` struct (product slug + quantity) for cookie storage, and a resolved `CartEntry` for templates
- JSON data loading from `data/` directory
- Cookie-based cart: `getCart(r)` reads cart from cookie, `setCart(w, cart)` writes it
- Custom `currency` filter: cents → `"$X.XX"`
- Handlers: index, products (with filtering/sorting), category, product, cart, cart/add, cart/remove, search
- `writeResult` helper for assembling HTML from RenderResult

```go
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

	grove "grove/pkg/grove"

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
			"product":  product,
			"related":  productsToAny(related),
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
	eng.SetGlobal("site_name", "Grove Store")
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

	fmt.Println("Grove Store listening on http://localhost:3001")
	log.Fatal(http.ListenAndServe(":3001", r))
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Product{}
	_ interface{ GroveResolve(string) (any, bool) } = Category{}
	_ interface{ GroveResolve(string) (any, bool) } = CartEntry{}
)
```

- [ ] **Step 2: Verify it compiles**

```bash
cd examples/store && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add examples/store/main.go
git commit -m "store: Rewrite main.go with data loading, filtering, cart, and 8 route handlers"
```

---

### Task 3: Create macro templates

**Files:**
- Modify: `examples/store/templates/macros/pricing.grov`
- Create: `examples/store/templates/macros/filters.grov`

- [ ] **Step 1: Rewrite pricing.grov**

```
{% macro price(amount, sale_amount) %}
  {% if sale_amount > 0 %}
    <span class="price-strike">{{ amount | currency }}</span>
    <span class="price-sale">{{ sale_amount | currency }}</span>
  {% else %}
    <span class="price-regular">{{ amount | currency }}</span>
  {% endif %}
{% endmacro %}

{% macro star_rating(rating, count) %}
  {% set full = rating | floor %}
  {% set half = rating - full >= 0.5 ? 1 : 0 %}
  <span class="stars">
    {% for i in range(1, full) %}&#9733;{% endfor %}{% if half %}&#9734;{% endif %}
  </span>
  <span class="review-count">({{ count }} {{ count == 1 ? "review" : "reviews" }})</span>
{% endmacro %}

{% macro discount_badge(price, sale_price) %}
  {% if sale_price > 0 %}
    {% let %}
      pct = ((price - sale_price) * 100) / price
    {% endlet %}
    <span class="badge badge-sale">{{ pct | floor }}% off</span>
  {% endif %}
{% endmacro %}

{% macro stock_badge(in_stock) %}
  {% if in_stock %}
    <span class="badge badge-in-stock">In Stock</span>
  {% else %}
    <span class="badge badge-out-of-stock">Out of Stock</span>
  {% endif %}
{% endmacro %}
```

- [ ] **Step 2: Write filters.grov**

```
{% macro sort_dropdown(current_sort) %}
<div class="sort-controls">
  <label>Sort by:</label>
  <select onchange="window.location.search = '?sort=' + this.value">
    <option value="" {{ not current_sort ? "selected" : "" }}>Default</option>
    <option value="price-asc" {{ current_sort == "price-asc" ? "selected" : "" }}>Price: Low to High</option>
    <option value="price-desc" {{ current_sort == "price-desc" ? "selected" : "" }}>Price: High to Low</option>
    <option value="rating" {{ current_sort == "rating" ? "selected" : "" }}>Top Rated</option>
    <option value="name" {{ current_sort == "name" ? "selected" : "" }}>Name A-Z</option>
  </select>
</div>
{% endmacro %}

{% macro category_filter(categories, active_category) %}
<div class="category-filter">
  <h3>Categories</h3>
  <ul class="filter-list">
    <li><a href="/products" class="{{ not active_category ? "filter-active" : "" }}">All</a></li>
    {% for cat in categories %}
      <li>
        <a href="/products?category={{ cat.slug }}" class="{{ active_category == cat.slug ? "filter-active" : "" }}">
          {{ cat.name }}
        </a>
      </li>
    {% endfor %}
  </ul>
</div>
{% endmacro %}
```

- [ ] **Step 3: Commit**

```bash
git add examples/store/templates/macros/
git commit -m "store: Add pricing and filter macro libraries"
```

---

### Task 4: Create component and page templates

**Files:**
- Modify: `examples/store/templates/base.grov`
- Modify: `examples/store/templates/index.grov`
- Modify: `examples/store/templates/product.grov`
- Modify: `examples/store/templates/cart.grov`
- Modify: `examples/store/templates/components/product-card.grov`
- Create: `examples/store/templates/product-list.grov`
- Create: `examples/store/templates/category.grov`
- Create: `examples/store/templates/search.grov`

- [ ] **Step 1: Rewrite base.grov**

```
{% asset "/static/style.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}Grove Store{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
</head>
<body>
  <nav class="nav">
    <a href="/" class="nav-brand">{{ site_name }}</a>
    <div class="nav-links">
      <a href="/products" class="nav-link">All Products</a>
      {% block nav_categories %}{% endblock %}
      <a href="/cart" class="nav-link nav-cart">Cart{% if cart_count %} ({{ cart_count }}){% endif %}</a>
    </div>
    <form action="/search" method="get" class="nav-search">
      <input type="text" name="q" placeholder="Search products..." class="search-input">
    </form>
  </nav>
  <main class="container">
    {% block content %}{% endblock %}
  </main>
  <footer class="footer">
    <p>&copy; {{ current_year }} Grove Store. Powered by the Grove template engine.</p>
  </footer>
  <!-- FOOT_ASSETS -->
</body>
</html>
```

- [ ] **Step 2: Rewrite index.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}{{ site_name }} &mdash; Outdoor Gear for Every Adventure{% endblock %}

{% block content %}
{% meta name="description" content="Shop outdoor gear for camping, hiking, cycling, running, and climbing." %}

<section class="hero">
  <h1>Gear Up for Your Next Adventure</h1>
  <p>Quality outdoor equipment for every pursuit.</p>
</section>

<section class="section">
  <h2>Featured Products</h2>
  <div class="product-grid">
    {% for product in featured %}
      {% capture price_html %}
        {{ pricing.price(product.price, product.sale_price) }}
      {% endcapture %}
      {% component "components/product-card.grov" name=product.name slug=product.slug image_url=product.image_url price_display=price_html rating=product.rating review_count=product.review_count %}
        {% fill "badge" %}
          {% if product.on_sale %}
            {{ pricing.discount_badge(product.price, product.sale_price) }}
          {% endif %}
        {% endfill %}
      {% endcomponent %}
    {% endfor %}
  </div>
</section>

<section class="section">
  <h2>Shop by Category</h2>
  <div class="category-grid">
    {% for cat in categories %}
      <a href="/category/{{ cat.slug }}" class="category-card">
        <h3>{{ cat.name }}</h3>
        <p>{{ cat.description | truncate(80) }}</p>
      </a>
    {% endfor %}
  </div>
</section>
{% endblock %}
```

- [ ] **Step 3: Write product-list.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}
{% import "macros/filters.grov" as flt %}

{% block title %}Products &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content="Browse our full product catalog" %}

{% include "components/breadcrumbs.grov" %}

<h1>Products</h1>
<p class="result-count">{{ result_count }} {{ result_count == 1 ? "product" : "products" }}</p>

<div class="product-page-layout">
  <aside class="sidebar">
    {{ flt.category_filter(categories, active_filters.category) }}
  </aside>

  <div class="product-content">
    {{ flt.sort_dropdown(active_filters.sort) }}

    <div class="product-grid">
      {% for product in products %}
        {% capture price_html %}
          {{ pricing.price(product.price, product.sale_price) }}
        {% endcapture %}
        {% component "components/product-card.grov" name=product.name slug=product.slug image_url=product.image_url price_display=price_html rating=product.rating review_count=product.review_count %}
          {% fill "badge" %}
            {% if product.on_sale %}
              {{ pricing.discount_badge(product.price, product.sale_price) }}
            {% endif %}
            {% if not product.in_stock %}
              {{ pricing.stock_badge(product.in_stock) }}
            {% endif %}
          {% endfill %}
        {% endcomponent %}
      {% empty %}
        <p class="empty-state">No products match your filters.</p>
      {% endfor %}
    </div>
  </div>
</div>
{% endblock %}
```

- [ ] **Step 4: Write category.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}{{ category.name }} &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content=category.description %}

{% include "components/breadcrumbs.grov" %}

<div class="category-header">
  <h1>{{ category.name }}</h1>
  <p>{{ category.description }}</p>
</div>

<div class="product-grid">
  {% for product in products %}
    {% capture price_html %}
      {{ pricing.price(product.price, product.sale_price) }}
    {% endcapture %}
    {% component "components/product-card.grov" name=product.name slug=product.slug image_url=product.image_url price_display=price_html rating=product.rating review_count=product.review_count %}
      {% fill "badge" %}
        {% if product.on_sale %}
          {{ pricing.discount_badge(product.price, product.sale_price) }}
        {% endif %}
      {% endfill %}
    {% endcomponent %}
  {% empty %}
    <p class="empty-state">No products in this category yet.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 5: Rewrite product.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}{{ product.name }} &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content=product.description %}
{% meta property="og:title" content=product.name %}

{% include "components/breadcrumbs.grov" %}

<div class="product-detail">
  <div class="product-image">
    <img src="{{ product.image_url }}" alt="{{ product.name }}">
  </div>
  <div class="product-info">
    <h1>{{ product.name }}</h1>

    <div class="product-rating">
      {{ pricing.star_rating(product.rating, product.review_count) }}
    </div>

    <div class="product-price">
      {{ pricing.price(product.price, product.sale_price) }}
      {{ pricing.discount_badge(product.price, product.sale_price) }}
    </div>

    {% if product.on_sale %}
      {% let %}
        savings = product.price - product.sale_price
      {% endlet %}
      <p class="savings">You save {{ savings | currency }}!</p>
    {% endif %}

    <p class="product-description">{{ product.description }}</p>

    {% if product.colors | length > 0 %}
      <div class="product-option">
        <strong>Color:</strong>
        <div class="option-list">
          {% for color in product.colors %}
            <span class="option-pill">{{ color }}</span>
          {% endfor %}
        </div>
      </div>
    {% endif %}

    {% if product.sizes | length > 0 %}
      <div class="product-option">
        <strong>Size:</strong>
        <div class="option-list">
          {% for size in product.sizes %}
            <span class="option-pill">{{ size }}</span>
          {% endfor %}
        </div>
      </div>
    {% endif %}

    {% if product.in_stock %}
      <div class="product-actions">
        <label>Qty:</label>
        <select id="qty-select">
          {% for n in range(1, 10) %}
            <option value="{{ n }}">{{ n }}</option>
          {% endfor %}
        </select>
        <a href="/cart/add?product={{ product.slug }}&qty=1" class="btn btn-primary">Add to Cart</a>
      </div>
    {% else %}
      <p class="out-of-stock">Out of Stock</p>
    {% endif %}
  </div>
</div>

<div class="product-body">
  {{ product.body | safe }}
</div>

{% if related | length > 0 %}
  <section class="related-products">
    <h2>Related Products</h2>
    <div class="product-grid">
      {% for rp in related %}
        {% capture rp_price %}
          {{ pricing.price(rp.price, rp.sale_price) }}
        {% endcapture %}
        {% component "components/product-card.grov" name=rp.name slug=rp.slug image_url=rp.image_url price_display=rp_price rating=rp.rating review_count=rp.review_count %}{% endcomponent %}
      {% endfor %}
    </div>
  </section>
{% endif %}
{% endblock %}
```

- [ ] **Step 6: Rewrite cart.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}Cart &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content="Your shopping cart" %}

{% include "components/breadcrumbs.grov" %}

<h1>Shopping Cart</h1>

{% if items | length > 0 %}
  <div class="table-wrap">
    <table class="table">
      <thead>
        <tr>
          <th>Product</th>
          <th>Price</th>
          <th>Qty</th>
          <th style="text-align: right;">Total</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {% for item in items %}
          <tr>
            <td><a href="/product/{{ item.product.slug }}">{{ item.product.name }}</a></td>
            <td>{{ pricing.price(item.product.price, item.product.sale_price) }}</td>
            <td>{{ item.quantity }}</td>
            <td style="text-align: right;">{{ item.line_total | currency }}</td>
            <td><a href="/cart/remove?product={{ item.product.slug }}" class="remove-link">Remove</a></td>
          </tr>
        {% endfor %}
      </tbody>
    </table>
  </div>

  {% let %}
    subtotal = 0
  {% endlet %}
  {% for item in items %}
    {% set subtotal = subtotal + item.line_total %}
  {% endfor %}

  <div class="cart-summary">
    <div class="cart-row">
      <span>Subtotal</span>
      <span>{{ subtotal | currency }}</span>
    </div>
    <div class="cart-row text-muted">
      <span>Shipping</span>
      <span>{{ subtotal >= 5000 ? "Free" : "$4.99" }}</span>
    </div>
    <hr>
    {% set total = subtotal >= 5000 ? subtotal : subtotal + 499 %}
    <div class="cart-row cart-total">
      <span>Total</span>
      <span>{{ total | currency }}</span>
    </div>
  </div>
{% else %}
  <p class="empty-state">Your cart is empty.</p>
  <a href="/products" class="btn btn-primary">Continue Shopping</a>
{% endif %}
{% endblock %}
```

- [ ] **Step 7: Write search.grov**

```
{% extends "base.grov" %}
{% import "macros/pricing.grov" as pricing %}

{% block title %}Search: "{{ query }}" &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% include "components/breadcrumbs.grov" %}

<h1>Search Results</h1>
{% if query %}
  <p class="result-count">{{ result_count }} {{ result_count == 1 ? "result" : "results" }} for "{{ query }}"</p>
{% endif %}

<div class="product-grid">
  {% for product in products %}
    {% capture price_html %}
      {{ pricing.price(product.price, product.sale_price) }}
    {% endcapture %}
    {% component "components/product-card.grov" name=product.name slug=product.slug image_url=product.image_url price_display=price_html rating=product.rating review_count=product.review_count %}{% endcomponent %}
  {% empty %}
    <p class="empty-state">No products found. Try a different search term.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 8: Rewrite product-card.grov**

```
{% props name, slug, image_url, price_display, rating=0, review_count=0 %}
<div class="product-card">
  <a href="/product/{{ slug }}" class="product-card-image">
    <img src="{{ image_url }}" alt="{{ name }}">
  </a>
  <div class="product-card-body">
    <h3><a href="/product/{{ slug }}">{{ name }}</a></h3>
    <div class="product-card-price">{{ price_display | safe }}</div>
    {% if rating > 0 %}
      <div class="product-card-rating">
        {% set full = rating | floor %}
        {% for i in range(1, full) %}&#9733;{% endfor %}
        <span class="text-muted">({{ review_count }})</span>
      </div>
    {% endif %}
    {% slot "badge" %}{% endslot %}
  </div>
</div>
```

- [ ] **Step 9: Create breadcrumbs.grov component**

Create `examples/store/templates/components/breadcrumbs.grov`:

```
<nav class="breadcrumb">
  {% for crumb in breadcrumbs %}
    {% if crumb.href %}
      <a href="{{ crumb.href }}">{{ crumb.label }}</a>
      <span class="breadcrumb-sep">/</span>
    {% else %}
      <span class="breadcrumb-current">{{ crumb.label }}</span>
    {% endif %}
  {% endfor %}
</nav>
```

- [ ] **Step 10: Commit**

```bash
git add examples/store/templates/
git commit -m "store: Add all page and component templates with filtering, cart, and search"
```

---

### Task 5: Update stylesheet

**Files:**
- Modify: `examples/store/static/style.css`

- [ ] **Step 1: Replace the stylesheet**

Replace `examples/store/static/style.css` with a comprehensive stylesheet. Beyond what already exists, add styles for:

- `.hero` — homepage hero banner
- `.category-grid` — grid of category cards
- `.category-card` — clickable category card with hover effect
- `.category-header` — category page header with description
- `.product-page-layout` — two-column layout with sidebar + product grid
- `.sidebar` — left sidebar for filters
- `.filter-list` — list of filter links with `.filter-active` state
- `.sort-controls` — sort dropdown container
- `.product-detail` — two-column product detail (image + info)
- `.product-option` — color/size selector row
- `.option-pill` — selectable option pill
- `.product-actions` — add to cart controls
- `.savings` — green savings text
- `.out-of-stock` — red out of stock text
- `.search-input` — search field in nav
- `.result-count` — "N products" text
- `.remove-link` — red remove link in cart
- `.cart-total` — bold total row
- `.related-products` — related products section
- `.breadcrumb` — breadcrumb navigation
- `.empty-state` — "no results" message

Keep the existing Grove Store brand colors (green `#2E6740`) and design language. Use CSS custom properties.

- [ ] **Step 2: Commit**

```bash
git add examples/store/static/style.css
git commit -m "store: Update stylesheet for new templates and layouts"
```

---

### Task 6: Build and verify

- [ ] **Step 1: Build**

```bash
cd examples/store && go build ./...
```

- [ ] **Step 2: Run and verify routes**

```bash
cd examples/store && go run main.go &
sleep 2
curl -s http://localhost:3001/ | head -20
curl -s http://localhost:3001/products | head -20
curl -s http://localhost:3001/products?category=hiking&sort=price-asc | head -20
curl -s http://localhost:3001/category/camping | head -20
curl -s http://localhost:3001/product/alpine-pro-tent | head -20
curl -s http://localhost:3001/cart | head -20
curl -s http://localhost:3001/search?q=tent | head -20
kill %1
```

Expected: All routes return HTML without errors.

- [ ] **Step 3: Verify cart add/remove**

```bash
cd examples/store && go run main.go &
sleep 2
# Add a product to cart — should redirect and set cookie
curl -v http://localhost:3001/cart/add?product=alpine-pro-tent&qty=2 2>&1 | grep -i "set-cookie\|location"
kill %1
```

Expected: Response includes a `Set-Cookie` header with cart data and a `Location` redirect.

- [ ] **Step 4: Final commit if any fixes needed**

```bash
git add examples/store/
git commit -m "store: Fix any issues found during verification"
```
