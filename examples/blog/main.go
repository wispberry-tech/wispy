package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Tag represents a blog post category.
type Tag struct {
	Name  string
	Color string
}

func (t Tag) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return t.Name, true
	case "color":
		return t.Color, true
	}
	return nil, false
}

// Post represents a blog post.
type Post struct {
	Title   string
	Slug    string
	Summary string
	Body    string
	Date    string
	Draft   bool
	Tags    []Tag
}

func (p Post) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return p.Title, true
	case "slug":
		return p.Slug, true
	case "summary":
		return p.Summary, true
	case "body":
		return p.Body, true
	case "date":
		return p.Date, true
	case "draft":
		return p.Draft, true
	case "tags":
		tags := make([]any, len(p.Tags))
		for i, t := range p.Tags {
			tags[i] = t
		}
		return tags, true
	}
	return nil, false
}

var posts = []Post{
	{
		Title:   "Hello, Grove!",
		Slug:    "hello-grove",
		Summary: "An introduction to the Grove template engine — a fast, safe, and expressive templating system for Go web applications.",
		Body:    "Grove is a template engine built from scratch in Go. It compiles templates to bytecode and runs them on a lightweight VM, making it both fast and safe.\n\nGrove supports all the features you'd expect from a modern template engine: variables, filters, control flow, loops, macros, components with slots, template inheritance, and more.\n\nWhat makes Grove special is its web-aware primitives. Templates can declare CSS and JS assets, set meta tags, and hoist content to specific page regions — all collected during rendering and assembled by the application layer.",
		Date:    "April 1, 2026",
		Draft:   false,
		Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Tutorial", Color: "blue"}},
	},
	{
		Title:   "Building Components",
		Slug:    "building-components",
		Summary: "Learn how to build reusable UI components with props, slots, and fills in Grove templates.",
		Body:    "Components are the building blocks of any modern UI. In Grove, a component is just a template file that declares its interface with props and slots.\n\nProps define the data a component accepts. You declare them at the top of a component file with the props tag. Each prop can have a default value, and Grove will raise an error if a required prop is missing.\n\nSlots let the caller inject content into specific regions of the component. The default slot captures the component body, while named slots give callers fine-grained control.\n\nHere's what makes Grove components powerful: fills see the caller's scope, not the component's. This means you can use your page data inside a fill block without threading it through props.",
		Date:    "March 28, 2026",
		Draft:   false,
		Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Components", Color: "green"}},
	},
	{
		Title:   "Template Inheritance Deep Dive",
		Slug:    "inheritance-deep-dive",
		Summary: "A deep dive into multi-level template inheritance, block overrides, and the super() function.",
		Body:    "Template inheritance lets you define a base layout once and override specific sections in child templates. Grove supports unlimited inheritance depth — a child can extend a parent that extends a grandparent.\n\nBlocks are the override points. Define a block in the base template with default content, then override it in child templates. Need the parent's content too? Call super() to include it.\n\nThis is a draft post — you should see a warning banner above!",
		Date:    "March 25, 2026",
		Draft:   true,
		Tags:    []Tag{{Name: "Grove", Color: "purple"}, {Name: "Advanced", Color: "red"}},
	},
}

func main() {
	// Resolve template directory relative to this source file.
	_, thisFile, _, _ := runtime.Caller(0)
	templateDir := filepath.Join(filepath.Dir(thisFile), "templates")

	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Grove Blog")
	eng.SetGlobal("current_year", "2026")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/post/{slug}", postHandler(eng))
	r.Get("/styleguide", styleguideHandler(eng))

	fmt.Println("Grove Blog listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Convert posts to []any for template iteration.
		postsAny := make([]any, len(posts))
		for i, p := range posts {
			postsAny[i] = p
		}

		result, err := eng.Render(r.Context(), "index.html", grove.Data{
			"posts": postsAny,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func postHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		var found *Post
		for i := range posts {
			if posts[i].Slug == slug {
				found = &posts[i]
				break
			}
		}
		if found == nil {
			http.NotFound(w, r)
			return
		}

		result, err := eng.Render(r.Context(), "post.html", grove.Data{
			"post": *found,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func styleguideHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := eng.Render(r.Context(), "pages/styleguide.html", grove.Data{
			"tag_colors": []any{"blue", "green", "red", "purple", "orange", "gray"},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

// writeResult assembles the final HTML response by injecting collected assets,
// meta tags, and hoisted content into the placeholder markers in the rendered body.
func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body

	// Inject stylesheet assets into <head>.
	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

	// Build meta tags.
	var meta strings.Builder
	for name, content := range result.Meta {
		if strings.HasPrefix(name, "og:") || strings.HasPrefix(name, "property:") {
			meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
		} else {
			meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
		}
	}
	body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)

	// Inject hoisted head content.
	body = strings.Replace(body, "<!-- HEAD_HOISTED -->", result.GetHoisted("head"), 1)

	// Inject script assets before </body>.
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

// Ensure types implement the Resolvable interface at compile time.
var (
	_ interface{ GroveResolve(string) (any, bool) } = Post{}
	_ interface{ GroveResolve(string) (any, bool) } = Tag{}
)

