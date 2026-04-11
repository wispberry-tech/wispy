package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	grove "github.com/wispberry-tech/grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// --- Types ---

type Section struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

func (s Section) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return s.Name, true
	case "slug":
		return s.Slug, true
	case "description":
		return s.Description, true
	case "order":
		return s.Order, true
	}
	return nil, false
}

type DocPage struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	SectionSlug string `json:"section_slug"`
	Order       int    `json:"order"`
	Body        string `json:"body"`
}

func (d DocPage) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return d.Title, true
	case "slug":
		return d.Slug, true
	case "section_slug":
		return d.SectionSlug, true
	case "section":
		if s, ok := sectionMap[d.SectionSlug]; ok {
			return s, true
		}
		return nil, false
	case "order":
		return d.Order, true
	case "body":
		return d.Body, true
	}
	return nil, false
}

type FilterEntry struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	ExampleInput  string `json:"example_input"`
	ExampleOutput string `json:"example_output"`
}

func (f FilterEntry) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return f.Name, true
	case "description":
		return f.Description, true
	case "category":
		return f.Category, true
	case "example_input":
		return f.ExampleInput, true
	case "example_output":
		return f.ExampleOutput, true
	}
	return nil, false
}

// --- Data ---

var (
	sections     []Section
	sectionMap   map[string]Section
	pages        []DocPage
	pageMap      map[string]DocPage // keyed by slug
	orderedPages []DocPage          // all pages in section-then-page order
	filterList   []FilterEntry
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
	loadJSON(baseDir, "sections.json", &sections)
	loadJSON(baseDir, "pages.json", &pages)
	loadJSON(baseDir, "filters.json", &filterList)

	sectionMap = make(map[string]Section)
	for _, s := range sections {
		sectionMap[s.Slug] = s
	}

	pageMap = make(map[string]DocPage)
	for _, p := range pages {
		pageMap[p.Slug] = p
	}

	// Build ordered page list: sort by section order then page order.
	orderedPages = make([]DocPage, len(pages))
	copy(orderedPages, pages)
	// Simple insertion sort by (section.order, page.order)
	for i := 1; i < len(orderedPages); i++ {
		for j := i; j > 0; j-- {
			si := sectionMap[orderedPages[j].SectionSlug].Order
			sj := sectionMap[orderedPages[j-1].SectionSlug].Order
			if si < sj || (si == sj && orderedPages[j].Order < orderedPages[j-1].Order) {
				orderedPages[j], orderedPages[j-1] = orderedPages[j-1], orderedPages[j]
			}
		}
	}
}

// --- Helpers ---

func sectionsToAny() []any {
	out := make([]any, len(sections))
	for i, s := range sections {
		out[i] = s
	}
	return out
}

func pagesToAny(pp []DocPage) []any {
	out := make([]any, len(pp))
	for i, p := range pp {
		out[i] = p
	}
	return out
}

func filtersToAny(ff []FilterEntry) []any {
	out := make([]any, len(ff))
	for i, f := range ff {
		out[i] = f
	}
	return out
}

func sectionPages(sectionSlug string) []DocPage {
	var out []DocPage
	for _, p := range orderedPages {
		if p.SectionSlug == sectionSlug {
			out = append(out, p)
		}
	}
	return out
}

func prevNextPages(slug string) (prev, next map[string]any) {
	for i, p := range orderedPages {
		if p.Slug == slug {
			if i > 0 {
				pp := orderedPages[i-1]
				prev = map[string]any{
					"title":        pp.Title,
					"slug":         pp.Slug,
					"section_slug": pp.SectionSlug,
				}
			}
			if i < len(orderedPages)-1 {
				np := orderedPages[i+1]
				next = map[string]any{
					"title":        np.Title,
					"slug":         np.Slug,
					"section_slug": np.SectionSlug,
				}
			}
			break
		}
	}
	return
}

func filterFilters(query, category string) []FilterEntry {
	var out []FilterEntry
	q := strings.ToLower(query)
	for _, f := range filterList {
		if category != "" && !strings.EqualFold(f.Category, category) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(f.Name), q) && !strings.Contains(strings.ToLower(f.Description), q) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func filterCategories() []any {
	seen := make(map[string]bool)
	var out []any
	for _, f := range filterList {
		if !seen[f.Category] {
			seen[f.Category] = true
			out = append(out, f.Category)
		}
	}
	return out
}

// --- Handlers ---

func landingHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := eng.Render(r.Context(), "landing.grov", grove.Data{
			"sections":  sectionsToAny(),
			"all_pages": pagesToAny(orderedPages),
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func sectionHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "section")
		sec, ok := sectionMap[slug]
		if !ok {
			http.NotFound(w, r)
			return
		}
		sp := sectionPages(slug)
		result, err := eng.Render(r.Context(), "section.grov", grove.Data{
			"section":       sec,
			"current_slug":  "",
			"section_pages": pagesToAny(sp),
			"sections":      sectionsToAny(),
			"all_pages":     pagesToAny(orderedPages),
			"breadcrumbs": []any{
				map[string]any{"label": "Docs", "href": "/"},
				map[string]any{"label": sec.Name, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func pageHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sectionSlug := chi.URLParam(r, "section")
		pageSlug := chi.URLParam(r, "page")

		page, ok := pageMap[pageSlug]
		if !ok || page.SectionSlug != sectionSlug {
			http.NotFound(w, r)
			return
		}

		sec := sectionMap[sectionSlug]
		prev, next := prevNextPages(pageSlug)

		// Try specific template, fall back to _default.grov
		templateName := "pages/" + pageSlug + ".grov"
		if _, err := eng.LoadTemplate(templateName); err != nil {
			templateName = "pages/_default.grov"
		}

		result, err := eng.Render(r.Context(), templateName, grove.Data{
			"page":          page,
			"current_slug":  page.Slug,
			"section":       sec,
			"section_slug":  sectionSlug,
			"sections":      sectionsToAny(),
			"all_pages":     pagesToAny(orderedPages),
			"prev":          prev,
			"next":          next,
			"breadcrumbs": []any{
				map[string]any{"label": "Docs", "href": "/"},
				map[string]any{"label": sec.Name, "href": "/docs/" + sectionSlug},
				map[string]any{"label": page.Title, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func filterRefHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		cat := r.URL.Query().Get("category")
		filtered := filterFilters(q, cat)

		result, err := eng.Render(r.Context(), "pages/filters.grov", grove.Data{
			"filters":           filtersToAny(filtered),
			"filter_categories": filterCategories(),
			"query":             q,
			"active_category":   cat,
			"result_count":      len(filtered),
			"sections":          sectionsToAny(),
			"all_pages":         pagesToAny(orderedPages),
			"breadcrumbs": []any{
				map[string]any{"label": "Docs", "href": "/"},
				map[string]any{"label": "Template Syntax", "href": "/docs/template-syntax"},
				map[string]any{"label": "Filter Reference", "href": ""},
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
	body = strings.Replace(body, "<!-- HEAD_HOISTED -->", result.GetHoisted("head"), 1)
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
	fsStore := grove.NewFileSystemStore(templateDir)
	eng := grove.New(
		grove.WithStore(fsStore),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags: []string{
				"set", "import", "slot", "asset", "meta",
				"#if", "#each", "#fill", "#slot", "#capture",
				"#hoist", "#let", "#verbatim",
				"Component",
			},
			AllowedFilters: []string{
				"upper", "lower", "title", "capitalize", "default", "truncate", "length",
				"join", "split", "replace", "trim", "lstrip", "rstrip", "nl2br", "safe",
				"floor", "ceil", "abs", "round", "int", "float",
				"first", "last", "sort", "reverse", "unique", "min", "max", "sum",
				"map", "batch", "flatten", "keys", "values",
				"escape", "striptags", "string", "bool", "wordcount",
				"center", "ljust", "rjust",
			},
			MaxLoopIter: 500,
		}),
	)
	eng.SetGlobal("site_name", "Grove Docs")
	eng.SetGlobal("current_year", "2026")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", landingHandler(eng))
	r.Get("/docs/filters", filterRefHandler(eng))
	r.Get("/docs/{section}", sectionHandler(eng))
	r.Get("/docs/{section}/{page}", pageHandler(eng))

	staticDir := filepath.Join(baseDir, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Serve colocated CSS and JS from component directories
	r.Handle("/css/*", http.StripPrefix("/css/", filteredFileServer(templateDir, ".css")))
	r.Handle("/js/*", http.StripPrefix("/js/", filteredFileServer(templateDir, ".js")))

	fmt.Println("Grove Documentation listening on http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", r))
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
	_ interface{ GroveResolve(string) (any, bool) } = Section{}
	_ interface{ GroveResolve(string) (any, bool) } = DocPage{}
	_ interface{ GroveResolve(string) (any, bool) } = FilterEntry{}
)
