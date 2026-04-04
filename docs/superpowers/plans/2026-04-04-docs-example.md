# Docs Example Overhaul — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the docs example into a real documentation site that accurately documents Grove's template syntax, with proper sidebar navigation, breadcrumbs, section pages, prev/next pagination, and a searchable filter reference.

**Architecture:** Data loaded from JSON files (sections, pages with real Grove documentation, filter reference data). Templates use 3-level inheritance with super(), render with explicit params for sidebar, and imported macro libraries. Engine configured with sandbox mode. All navigation derived from actual page hierarchy.

**Tech Stack:** Go 1.24, Grove template engine, Chi router, JSON data files

**Spec:** `docs/superpowers/specs/2026-04-04-examples-expansion-design.md` — Example 3: Docs

**Important:** The doc page content must be accurate Grove documentation. Read the actual Grove source code (particularly `pkg/grove/`, `internal/filters/`, `internal/compiler/`, `internal/vm/`) to verify syntax and feature descriptions before writing page content.

---

### Task 1: Create JSON data files

**Files:**
- Create: `examples/docs/data/sections.json`
- Create: `examples/docs/data/pages.json`
- Create: `examples/docs/data/filters.json`

- [ ] **Step 1: Create the data directory**

```bash
mkdir -p examples/docs/data
```

- [ ] **Step 2: Write sections.json**

```json
[
  {"name": "Getting Started", "slug": "getting-started", "description": "Install Grove and render your first template.", "order": 1},
  {"name": "Template Syntax", "slug": "template-syntax", "description": "Variables, expressions, filters, and control flow.", "order": 2},
  {"name": "Tags", "slug": "tags", "description": "Template inheritance, includes, components, and macros.", "order": 3},
  {"name": "Advanced", "slug": "advanced", "description": "Asset management, sandboxing, custom filters, and Go integration.", "order": 4}
]
```

- [ ] **Step 3: Write pages.json**

Create 13 documentation pages with accurate Grove content. Each page has: title, slug, section_slug, order (within section), body (HTML with code examples using `<pre><code>` blocks).

**IMPORTANT:** Before writing each page body, read the actual Grove source code to verify syntax. For example, check `internal/compiler/` and `internal/parser/` for exact tag syntax, `internal/filters/` for available filters, `pkg/grove/engine.go` for the Go API.

The pages and their content topics:

**Getting Started section (order 1):**
1. `installation` — `go get`, module setup, basic engine creation code
2. `quick-start` — First template, passing data from Go, rendering
3. `template-basics` — Variable interpolation `{{ }}`, comments `{# #}`, whitespace

**Template Syntax section (order 2):**
4. `variables-and-expressions` — Interpolation, arithmetic (`+`, `-`, `*`, `/`, `%`), comparisons (`==`, `!=`, `<`, `>`), logical (`and`, `or`, `not`), ternary `? :`, string concat `~`
5. `filters` — Pipe syntax, chaining, common filters with examples (link to filter reference)
6. `control-flow` — `if`/`elif`/`else`, `for`/`empty`, `range()`, `set`, `let` blocks

**Tags section (order 3):**
7. `template-inheritance` — `extends`, `block`/`endblock`, `super()`, multi-level inheritance
8. `includes-and-partials` — `include` (inherits scope), `render` (isolated scope with params)
9. `components` — `component`/`endcomponent`, `props`, `slot`/`endslot`, `fill`/`endfill`
10. `macros` — `macro`/`endmacro`, `import ... as`, calling macros

**Advanced section (order 4):**
11. `asset-management` — `asset` tag (type, priority), `meta` tag, `hoist` tag, RenderResult.HeadHTML/FootHTML
12. `sandboxing` — `WithSandbox`, `SandboxConfig` (AllowedTags, AllowedFilters, MaxLoopIter)
13. `go-integration` — `GroveResolve` interface, Engine API, `RegisterFilter`, `Data` type, stores

Example page entry:

```json
{
  "title": "Installation",
  "slug": "installation",
  "section_slug": "getting-started",
  "order": 1,
  "body": "<h2>Install Grove</h2><p>Add Grove to your Go module:</p><pre><code>go get grove</code></pre><p>Grove requires Go 1.24 or later. It has no runtime dependencies beyond the Go standard library.</p><h2>Create an Engine</h2><p>The engine is the main entry point for compiling and rendering templates:</p><pre><code>package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\tgrove \"grove/pkg/grove\"\n)\n\nfunc main() {\n\tstore := grove.NewFileSystemStore(\"./templates\")\n\teng := grove.New(grove.WithStore(store))\n\n\tresult, err := eng.Render(context.Background(), \"hello.grov\", grove.Data{\n\t\t\"name\": \"World\",\n\t})\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tfmt.Println(result.Body)\n}</code></pre><p>The <code>FileSystemStore</code> loads templates from the specified directory. The engine compiles templates to bytecode on first use and caches them (default 512 entries).</p>"
}
```

Write all 13 pages following this pattern. Each page body should be 2-5 paragraphs of HTML with code examples where appropriate. Code examples should use actual Grove syntax (verify against source).

- [ ] **Step 4: Write filters.json**

This file documents all built-in filters. Read `/home/theo/Work/grove/internal/filters/` to get the actual list. Each entry has: name, description, category (String, Collection, Numeric, HTML, Date), example_input, example_output.

```json
[
  {"name": "upper", "description": "Converts string to uppercase.", "category": "String", "example_input": "hello", "example_output": "HELLO"},
  {"name": "lower", "description": "Converts string to lowercase.", "category": "String", "example_input": "HELLO", "example_output": "hello"},
  {"name": "title", "description": "Capitalizes first letter of each word.", "category": "String", "example_input": "hello world", "example_output": "Hello World"},
  {"name": "trim", "description": "Removes leading and trailing whitespace.", "category": "String", "example_input": "  hello  ", "example_output": "hello"},
  {"name": "truncate", "description": "Truncates string to specified length, appending '...'.", "category": "String", "example_input": "hello world (truncate(5))", "example_output": "he..."},
  {"name": "replace", "description": "Replaces occurrences of a substring.", "category": "String", "example_input": "hello world (replace('world','grove'))", "example_output": "hello grove"},
  {"name": "split", "description": "Splits string into array by delimiter.", "category": "String", "example_input": "a,b,c (split(','))", "example_output": "[a, b, c]"},
  {"name": "length", "description": "Returns length of string or collection.", "category": "Collection", "example_input": "[1, 2, 3]", "example_output": "3"},
  {"name": "join", "description": "Joins array elements with separator.", "category": "Collection", "example_input": "[a, b, c] (join(', '))", "example_output": "a, b, c"},
  {"name": "first", "description": "Returns first element of array.", "category": "Collection", "example_input": "[1, 2, 3]", "example_output": "1"},
  {"name": "last", "description": "Returns last element of array.", "category": "Collection", "example_input": "[1, 2, 3]", "example_output": "3"},
  {"name": "sort", "description": "Sorts array elements.", "category": "Collection", "example_input": "[3, 1, 2]", "example_output": "[1, 2, 3]"},
  {"name": "reverse", "description": "Reverses array or string.", "category": "Collection", "example_input": "[1, 2, 3]", "example_output": "[3, 2, 1]"},
  {"name": "floor", "description": "Rounds number down to nearest integer.", "category": "Numeric", "example_input": "4.7", "example_output": "4"},
  {"name": "ceil", "description": "Rounds number up to nearest integer.", "category": "Numeric", "example_input": "4.2", "example_output": "5"},
  {"name": "abs", "description": "Returns absolute value.", "category": "Numeric", "example_input": "-5", "example_output": "5"},
  {"name": "nl2br", "description": "Converts newlines to <br> tags.", "category": "HTML", "example_input": "line1\\nline2", "example_output": "line1<br>line2"},
  {"name": "safe", "description": "Marks content as safe HTML (skips auto-escaping).", "category": "HTML", "example_input": "<b>bold</b>", "example_output": "<b>bold</b>"},
  {"name": "default", "description": "Returns fallback value if input is nil or empty.", "category": "Collection", "example_input": "nil (default('N/A'))", "example_output": "N/A"},
  {"name": "date", "description": "Formats a date string.", "category": "Date", "example_input": "2026-04-04 (date('Jan 2, 2006'))", "example_output": "Apr 4, 2026"}
]
```

**IMPORTANT:** Read `/home/theo/Work/grove/internal/filters/` to find ALL built-in filters and add any that are missing from this list. The spec says there are 40+ filters — include them all.

- [ ] **Step 5: Commit data files**

```bash
git add examples/docs/data/
git commit -m "docs: Add JSON data files with accurate Grove documentation content"
```

---

### Task 2: Rewrite main.go

**Files:**
- Modify: `examples/docs/main.go`

- [ ] **Step 1: Write the complete main.go**

Replace the entire file. The new main.go includes:
- `Section`, `DocPage`, `FilterEntry` structs with GroveResolve
- JSON data loading with section-page relationship resolution
- Ordered page list with prev/next computation
- Sandbox configuration
- Handlers: landing page, section index, doc page, filter reference (with query params)
- Template selection: specific template if available, else `_default.grov`

```go
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

	grove "grove/pkg/grove"

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
	sections   []Section
	sectionMap map[string]Section
	pages      []DocPage
	pageMap    map[string]DocPage // keyed by slug
	orderedPages []DocPage       // all pages in section-then-page order
	filters    []FilterEntry
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
	loadJSON(baseDir, "filters.json", &filters)

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
	for _, f := range filters {
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
	for _, f := range filters {
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
			"section":      sec,
			"section_pages": pagesToAny(sp),
			"sections":     sectionsToAny(),
			"all_pages":    pagesToAny(orderedPages),
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
			"page":         page,
			"section":      sec,
			"section_slug": sectionSlug,
			"sections":     sectionsToAny(),
			"all_pages":    pagesToAny(orderedPages),
			"prev":         prev,
			"next":         next,
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
				"if", "elif", "else", "for", "empty", "set", "let",
				"block", "extends", "include", "render", "import",
				"component", "slot", "fill", "props",
				"macro", "call", "capture", "range",
				"asset", "meta", "hoist",
			},
			AllowedFilters: []string{
				"upper", "lower", "title", "default", "truncate", "length",
				"join", "split", "replace", "trim", "nl2br", "safe",
				"floor", "ceil", "abs", "date", "first", "last",
				"sort", "reverse",
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

	fmt.Println("Grove Docs listening on http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", r))
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Section{}
	_ interface{ GroveResolve(string) (any, bool) } = DocPage{}
	_ interface{ GroveResolve(string) (any, bool) } = FilterEntry{}
)
```

- [ ] **Step 2: Verify it compiles**

```bash
cd examples/docs && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add examples/docs/main.go
git commit -m "docs: Rewrite main.go with data loading, sandbox config, and section/page handlers"
```

---

### Task 3: Create templates

**Files:**
- Modify: `examples/docs/templates/base.grov`
- Modify: `examples/docs/templates/docs-layout.grov`
- Create: `examples/docs/templates/landing.grov`
- Create: `examples/docs/templates/section.grov`
- Modify: `examples/docs/templates/pages/_default.grov`
- Create: `examples/docs/templates/pages/filters.grov`
- Create: `examples/docs/templates/pages/template-inheritance.grov`
- Modify: `examples/docs/templates/partials/sidebar.grov`
- Create: `examples/docs/templates/partials/breadcrumbs.grov`
- Create: `examples/docs/templates/partials/prev-next.grov`
- Modify: `examples/docs/templates/macros/admonitions.grov`
- Create: `examples/docs/templates/macros/code-example.grov`

- [ ] **Step 1: Rewrite base.grov**

```
{% asset "/static/docs.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}{{ site_name }}{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  {% block nav %}
  <nav class="nav">
    <a href="/" class="nav-brand">{{ site_name }}</a>
    <div class="nav-links">
      {% for section in sections %}
        <a href="/docs/{{ section.slug }}" class="nav-link">{{ section.name }}</a>
      {% endfor %}
      <a href="/docs/filters" class="nav-link">Filter Reference</a>
    </div>
  </nav>
  {% endblock %}
  {% block layout %}
  <main class="container">
    {% block content %}{% endblock %}
  </main>
  {% endblock %}
  <footer class="footer">
    <p>&copy; {{ current_year }} {{ site_name }}. Built with the Grove template engine.</p>
  </footer>
  <!-- FOOT_ASSETS -->
</body>
</html>
```

- [ ] **Step 2: Rewrite docs-layout.grov**

This is the second level of inheritance. Adds sidebar and breadcrumbs, uses `super()` to extend the nav block.

```
{% extends "base.grov" %}

{% block nav %}
  {{ super() }}
  {% render "partials/breadcrumbs.grov" breadcrumbs=breadcrumbs %}
{% endblock %}

{% block layout %}
<div class="docs-layout">
  <aside class="docs-sidebar">
    {% render "partials/sidebar.grov" sections=sections all_pages=all_pages current_slug=page.slug %}
  </aside>
  <main class="docs-content">
    {% block content %}{% endblock %}
  </main>
</div>
{% endblock %}
```

- [ ] **Step 3: Write landing.grov**

```
{% extends "base.grov" %}

{% block title %}{{ site_name }} &mdash; Grove Template Engine Documentation{% endblock %}

{% block content %}
{% meta name="description" content="Documentation for the Grove template engine for Go." %}

<div class="landing">
  <h1>Grove Documentation</h1>
  <p class="landing-subtitle">A bytecode-compiled template engine for Go with components, inheritance, and web primitives.</p>

  <div class="section-grid">
    {% for section in sections %}
      <a href="/docs/{{ section.slug }}" class="section-card">
        <h2>{{ section.name }}</h2>
        <p>{{ section.description }}</p>
      </a>
    {% endfor %}
  </div>
</div>
{% endblock %}
```

- [ ] **Step 4: Write section.grov**

```
{% extends "docs-layout.grov" %}

{% block title %}{{ section.name }} &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content=section.description %}

<h1>{{ section.name }}</h1>
<p class="section-description">{{ section.description }}</p>

<div class="page-list">
  {% for pg in section_pages %}
    <a href="/docs/{{ pg.section_slug }}/{{ pg.slug }}" class="page-list-item">
      <h3>{{ pg.title }}</h3>
    </a>
  {% empty %}
    <p>No pages in this section yet.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 5: Rewrite pages/_default.grov**

This is the generic doc page template — third level of inheritance.

```
{% extends "docs-layout.grov" %}
{% import "macros/admonitions.grov" as adm %}

{% block title %}{{ page.title }} &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content=page.title ~ " — Grove template engine documentation" %}

<article class="doc-page">
  <h1>{{ page.title }}</h1>
  <div class="doc-body">
    {{ page.body | safe }}
  </div>
</article>

{% render "partials/prev-next.grov" prev=prev next=next %}
{% endblock %}
```

- [ ] **Step 6: Write pages/template-inheritance.grov**

This is the meta page — it explains the very features rendering it.

```
{% extends "docs-layout.grov" %}
{% import "macros/admonitions.grov" as adm %}

{% block title %}{{ page.title }} &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content="How template inheritance works in Grove" %}

<article class="doc-page">
  <h1>{{ page.title }}</h1>

  {{ adm.tip("This page is itself rendered using 3-level template inheritance. The base template defines the HTML shell, docs-layout adds the sidebar, and this page fills in the content. View the template source to see it in action.") }}

  <div class="doc-body">
    {{ page.body | safe }}
  </div>
</article>

{% render "partials/prev-next.grov" prev=prev next=next %}
{% endblock %}
```

- [ ] **Step 7: Write pages/filters.grov**

The filter reference page with search and category filtering.

```
{% extends "docs-layout.grov" %}

{% block title %}Filter Reference &mdash; {{ site_name }}{% endblock %}

{% block content %}
{% meta name="description" content="Complete reference of all built-in Grove template filters" %}

<h1>Filter Reference</h1>
<p>{{ result_count }} filters{% if query %} matching "{{ query }}"{% endif %}{% if active_category %} in {{ active_category }}{% endif %}</p>

<form action="/docs/filters" method="get" class="filter-search">
  <input type="text" name="q" value="{{ query | default("") }}" placeholder="Search filters..." class="search-input">
  <select name="category" onchange="this.form.submit()">
    <option value="">All Categories</option>
    {% for cat in filter_categories %}
      <option value="{{ cat }}" {{ active_category == cat ? "selected" : "" }}>{{ cat }}</option>
    {% endfor %}
  </select>
</form>

<div class="filter-table-wrap">
  <table class="filter-table">
    <thead>
      <tr>
        <th>Filter</th>
        <th>Category</th>
        <th>Description</th>
        <th>Example</th>
      </tr>
    </thead>
    <tbody>
      {% for f in filters %}
        <tr>
          <td><code>{{ f.name }}</code></td>
          <td><span class="filter-category">{{ f.category }}</span></td>
          <td>{{ f.description }}</td>
          <td><code>{{ f.example_input }}</code> &rarr; <code>{{ f.example_output }}</code></td>
        </tr>
      {% empty %}
        <tr>
          <td colspan="4">No filters match your search.</td>
        </tr>
      {% endfor %}
    </tbody>
  </table>
</div>
{% endblock %}
```

- [ ] **Step 8: Rewrite partials/sidebar.grov**

Receives data via `{% render %}` with explicit params (isolated scope).

```
<nav class="sidebar-nav">
  {% for section in sections %}
    {% set section_slug = section.slug %}
    <div class="sidebar-section">
      <h3 class="sidebar-section-title">
        <a href="/docs/{{ section_slug }}">{{ section.name }}</a>
      </h3>
      <ul class="sidebar-page-list">
        {% for pg in all_pages %}
          {% if pg.section_slug == section_slug %}
            {% set href = "/docs/" ~ section_slug ~ "/" ~ pg.slug %}
            {% if pg.slug == current_slug %}
              <li><a href="{{ href }}" class="sidebar-active">{{ pg.title }}</a></li>
            {% else %}
              <li><a href="{{ href }}">{{ pg.title }}</a></li>
            {% endif %}
          {% endif %}
        {% endfor %}
      </ul>
    </div>
  {% endfor %}
</nav>
```

- [ ] **Step 9: Write partials/breadcrumbs.grov**

```
<div class="breadcrumb-bar">
  {% for crumb in breadcrumbs %}
    {% if crumb.href %}
      <a href="{{ crumb.href }}">{{ crumb.label }}</a>
      <span class="breadcrumb-sep">/</span>
    {% else %}
      <span class="breadcrumb-current">{{ crumb.label }}</span>
    {% endif %}
  {% endfor %}
</div>
```

- [ ] **Step 10: Write partials/prev-next.grov**

```
<nav class="prev-next">
  {% if prev %}
    <a href="/docs/{{ prev.section_slug }}/{{ prev.slug }}" class="prev-next-link prev-link">
      &larr; {{ prev.title }}
    </a>
  {% else %}
    <span></span>
  {% endif %}
  {% if next %}
    <a href="/docs/{{ next.section_slug }}/{{ next.slug }}" class="prev-next-link next-link">
      {{ next.title }} &rarr;
    </a>
  {% else %}
    <span></span>
  {% endif %}
</nav>
```

- [ ] **Step 11: Keep macros/admonitions.grov as-is** (the existing macros are fine)

Verify the file exists and has `note`, `warning`, and `tip` macros. No changes needed.

- [ ] **Step 12: Write macros/code-example.grov**

```
{% macro code(label, content) %}
<div class="code-example">
  {% if label %}
    <div class="code-label">{{ label }}</div>
  {% endif %}
  <pre><code>{{ content }}</code></pre>
</div>
{% endmacro %}
```

- [ ] **Step 13: Delete old pages/variables-and-filters.grov** (replaced by the new filters.grov route)

```bash
rm -f examples/docs/templates/pages/variables-and-filters.grov
```

- [ ] **Step 14: Commit**

```bash
git add examples/docs/templates/
git commit -m "docs: Add all templates — 3-level inheritance, sidebar, breadcrumbs, filter reference"
```

---

### Task 4: Update stylesheet

**Files:**
- Modify: `examples/docs/static/docs.css`

- [ ] **Step 1: Update the stylesheet**

Add styles for the new templates beyond what already exists:

- `.landing` — centered landing page with title and subtitle
- `.section-grid` — grid of section cards on landing page
- `.section-card` — clickable section card with hover
- `.section-description` — section index description text
- `.page-list` — list of pages in a section
- `.page-list-item` — clickable page link
- `.docs-layout` — two-column flex layout (sidebar + content)
- `.docs-sidebar` — fixed-width sidebar
- `.docs-content` — main content area
- `.sidebar-nav` — sidebar navigation structure
- `.sidebar-section` — collapsible section group
- `.sidebar-section-title` — section heading
- `.sidebar-page-list` — list of page links
- `.sidebar-active` — highlighted current page
- `.breadcrumb-bar` — breadcrumb navigation below nav
- `.doc-page` — article container
- `.doc-body` — documentation content with styled `pre`, `code`, `h2`, `h3`, `p`
- `.prev-next` — flex layout for prev/next links
- `.prev-next-link` — styled navigation arrows
- `.filter-search` — search form on filter reference page
- `.filter-table` — styled table for filter reference
- `.filter-category` — category badge in table
- `.code-example` — styled code block with label

Keep the existing Grove brand colors and design language.

- [ ] **Step 2: Commit**

```bash
git add examples/docs/static/docs.css
git commit -m "docs: Update stylesheet for new templates and layouts"
```

---

### Task 5: Build and verify

- [ ] **Step 1: Build**

```bash
cd examples/docs && go build ./...
```

- [ ] **Step 2: Run and verify routes**

```bash
cd examples/docs && go run main.go &
sleep 2
curl -s http://localhost:3002/ | head -20
curl -s http://localhost:3002/docs/getting-started | head -20
curl -s http://localhost:3002/docs/getting-started/installation | head -20
curl -s http://localhost:3002/docs/tags/template-inheritance | head -20
curl -s http://localhost:3002/docs/filters | head -20
curl -s http://localhost:3002/docs/filters?q=upper&category=String | head -20
kill %1
```

Expected: All routes return HTML. Landing page shows section cards. Doc pages have sidebar, breadcrumbs, prev/next navigation. Filter reference shows a searchable table.

- [ ] **Step 3: Verify sidebar navigation**

Check that the sidebar correctly highlights the current page and all links resolve to real pages.

- [ ] **Step 4: Verify breadcrumbs**

Check that breadcrumbs show Docs → Section → Page and each crumb links to a real page.

- [ ] **Step 5: Final commit if any fixes needed**

```bash
git add examples/docs/
git commit -m "docs: Fix any issues found during verification"
```
