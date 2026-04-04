package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	grove "grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// DocPage represents a single documentation page.
type DocPage struct {
	Title   string
	Section string
	Slug    string
	Body    string
}

func (d DocPage) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return d.Title, true
	case "section":
		return d.Section, true
	case "slug":
		return d.Slug, true
	case "body":
		return d.Body, true
	}
	return nil, false
}

var sections = []string{"Getting Started", "Templates"}

var pages = []DocPage{
	{
		Title:   "Installation",
		Section: "Getting Started",
		Slug:    "installation",
		Body:    "Install Grove by adding it as a Go module dependency:\n\n<pre><code>go get grove</code></pre>\n\nGrove requires Go 1.24 or later. It has zero runtime dependencies — the only external package is testify, used for tests.",
	},
	{
		Title:   "Quick Start",
		Section: "Getting Started",
		Slug:    "quick-start",
		Body:    "Create an engine, add a template, and render it:\n\n<pre><code>store := grove.NewMemoryStore()\nstore.Set(\"hello.grov\", \"Hello, {{ name }}!\")\neng := grove.New(grove.WithStore(store))\nresult, _ := eng.Render(ctx, \"hello.grov\", grove.Data{\"name\": \"world\"})\nfmt.Println(result.Body) // Hello, world!</code></pre>",
	},
	{
		Title:   "Variables & Filters",
		Section: "Templates",
		Slug:    "variables-and-filters",
		Body:    "Output a variable with double curly braces: <code>{{ name }}</code>. Apply filters with the pipe operator: <code>{{ name | upper }}</code>.\n\nGrove includes 40+ built-in filters for strings, collections, numbers, and HTML. Chain multiple filters: <code>{{ title | lower | truncate(50) }}</code>.",
	},
	{
		Title:   "Control Flow",
		Section: "Templates",
		Slug:    "control-flow",
		Body:    "Use <code>if</code>, <code>elif</code>, and <code>else</code> for conditionals:\n\n<pre><code>{% if user.admin %}\n  Admin panel\n{% elif user.moderator %}\n  Mod tools\n{% else %}\n  Standard view\n{% endif %}</code></pre>\n\nLoop with <code>for</code> and handle empty lists with <code>empty</code>:\n\n<pre><code>{% for item in items %}\n  {{ item.name }}\n{% empty %}\n  No items found.\n{% endfor %}</code></pre>",
	},
	{
		Title:   "Template Inheritance",
		Section: "Templates",
		Slug:    "template-inheritance",
		Body:    "Define a base layout with <code>block</code> tags, then extend it in child templates. Child templates override blocks; use <code>super()</code> to include the parent's content.\n\nGrove supports unlimited inheritance depth — a child can extend a parent that extends a grandparent.",
	},
}

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(filepath.Dir(thisFile), "templates")

	fsStore := grove.NewFileSystemStore(templateDir)
	eng := grove.New(
		grove.WithStore(fsStore),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags:    []string{"if", "elif", "else", "for", "empty", "set", "let", "block", "extends", "include", "render", "import", "component", "slot", "fill", "props", "macro", "call", "capture", "range", "asset", "meta", "hoist"},
			AllowedFilters: []string{"upper", "lower", "title", "default", "truncate", "length", "join", "split", "replace", "trim", "nl2br", "safe", "floor", "ceil", "abs", "date"},
			MaxLoopIter:    500,
		}),
	)
	eng.SetGlobal("site_name", "Grove Docs")
	eng.SetGlobal("current_year", "2026")

	// Build sections data for sidebar.
	sectionsAny := make([]any, len(sections))
	for i, s := range sections {
		sectionsAny[i] = s
	}
	eng.SetGlobal("sections", sectionsAny)

	pagesAny := make([]any, len(pages))
	for i, p := range pages {
		pagesAny[i] = p
	}
	eng.SetGlobal("all_pages", pagesAny)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/getting-started/installation", http.StatusFound)
	})
	r.Get("/docs/{section}/{page}", pageHandler(eng))

	fmt.Println("Grove Docs listening on http://localhost:3003")
	log.Fatal(http.ListenAndServe(":3003", r))
}

func pageHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "page")
		var found *DocPage
		for i := range pages {
			if pages[i].Slug == slug {
				found = &pages[i]
				break
			}
		}
		if found == nil {
			http.NotFound(w, r)
			return
		}

		// Find prev/next pages.
		var prev, next map[string]any
		for i, p := range pages {
			if p.Slug == slug {
				if i > 0 {
					pp := pages[i-1]
					prev = map[string]any{
						"title":   pp.Title,
						"section": pp.Section,
						"slug":    pp.Slug,
					}
				}
				if i < len(pages)-1 {
					np := pages[i+1]
					next = map[string]any{
						"title":   np.Title,
						"section": np.Section,
						"slug":    np.Slug,
					}
				}
				break
			}
		}

		sectionSlug := strings.ReplaceAll(strings.ToLower(found.Section), " ", "-")
		templateName := "pages/" + found.Slug + ".grov"

		// Check if a specific page template exists; fall back to generic.
		_, err := eng.LoadTemplate(templateName)
		if err != nil {
			templateName = "pages/_default.grov"
		}

		result, err := eng.Render(r.Context(), templateName, grove.Data{
			"page":         *found,
			"section_slug": sectionSlug,
			"prev":         prev,
			"next":         next,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

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

var _ interface{ GroveResolve(string) (any, bool) } = DocPage{}
