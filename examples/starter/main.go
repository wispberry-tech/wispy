// Package main runs the Grove starter example — a minimal showcase site
// that demonstrates core Grove features: components, asset pipeline,
// templates, and domain-driven rendering.
//
// This file is ordered for clarity:
//  1. Domain types + GroveResolve
//  2. Engine bootstrap
//  3. HTTP handlers
//  4. Response assembly (writeResult)
//  5. Routes + main
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	grove "github.com/wispberry-tech/grove/pkg/grove"
	"github.com/wispberry-tech/grove/pkg/grove/assets"
	"github.com/wispberry-tech/grove/pkg/grove/assets/esm"
	"github.com/wispberry-tech/grove/pkg/grove/assets/minify"
)

// ─── Domain types ───────────────────────────────────────────────────────────

const (
	repoURL = "https://github.com/wispberry-tech/grove"
	docsURL = repoURL + "/tree/main/docs"
	pkgURL  = "https://pkg.go.dev/github.com/wispberry-tech/grove"
)

// Highlight is a short value-prop on the homepage that links to a real
// artefact (repo path, doc file, or pkg.go.dev page).
type Highlight struct {
	Title       string
	Description string
	Href        string
}

func (h Highlight) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return h.Title, true
	case "description":
		return h.Description, true
	case "href":
		return h.Href, true
	}
	return nil, false
}

// DocLink is a single entry on the /docs page pointing at a Markdown file
// in the repository's docs/ directory.
type DocLink struct {
	Title       string
	Description string
	Href        string
}

func (d DocLink) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return d.Title, true
	case "description":
		return d.Description, true
	case "href":
		return d.Href, true
	}
	return nil, false
}

// ─── Content registries ─────────────────────────────────────────────────────

// homeHighlights — three concise value props shown on the landing page.
// TODO(human): tune the copy below. Keep each description to one short
// sentence; you can keep, rewrite, or reorder entries. Titles ≤ 4 words.
var homeHighlights = []Highlight{
	{
		Title:       "Bytecode VM, shared safely",
		Description: "Templates compile once to immutable bytecode and run on a pooled VM across every goroutine.",
		Href:        repoURL + "/blob/main/docs/vm.md",
	},
	{
		Title:       "Components with slots & fills",
		Description: "Reusable `.grov` files with props, named slots, and default fallbacks — compose pages without string glue.",
		Href:        repoURL + "/blob/main/docs/components.md",
	},
	{
		Title:       "Colocated asset pipeline",
		Description: "Component CSS and JS sit next to the template, get content-hashed, and inject themselves into the page.",
		Href:        repoURL + "/blob/main/docs/asset-pipeline.md",
	},
}

// docLinks — cards shown on /docs. Each points at a real file in docs/.
var docLinks = []DocLink{
	{Title: "Getting Started", Description: "Install the module, render your first template, wire up an HTTP handler.", Href: repoURL + "/blob/main/docs/getting-started.md"},
	{Title: "Template Syntax", Description: "Variables, filters, expressions, conditionals, loops, captures, and literals.", Href: repoURL + "/blob/main/docs/template-syntax.md"},
	{Title: "Components", Description: "Define components, pass props, compose with slots and fills.", Href: repoURL + "/blob/main/docs/components.md"},
	{Title: "Filters", Description: "The 40+ built-in filters — strings, collections, numbers, HTML.", Href: repoURL + "/blob/main/docs/filters.md"},
	{Title: "Web Primitives", Description: "Meta tags, asset tags, hoisted head content, verbatim blocks.", Href: repoURL + "/blob/main/docs/web-primitives.md"},
	{Title: "Asset Pipeline", Description: "Hash, minify, and inject component CSS/JS. Configure the builder.", Href: repoURL + "/blob/main/docs/asset-pipeline.md"},
	{Title: "Template Inheritance", Description: "Base layouts, extensions, and block overrides.", Href: repoURL + "/blob/main/docs/template-inheritance.md"},
	{Title: "Macros & Includes", Description: "Reusable snippets and cross-template helpers.", Href: repoURL + "/blob/main/docs/macros-and-includes.md"},
	{Title: "Examples", Description: "Reference apps showcasing real-world Grove usage.", Href: repoURL + "/blob/main/docs/examples.md"},
	{Title: "API Reference", Description: "The public Go API: Engine, options, stores, filters, assets.", Href: repoURL + "/blob/main/docs/api-reference.md"},
}

func highlightsToAny(list []Highlight) []any {
	out := make([]any, len(list))
	for i := range list {
		out[i] = list[i]
	}
	return out
}

func docLinksToAny(list []DocLink) []any {
	out := make([]any, len(list))
	for i := range list {
		out[i] = list[i]
	}
	return out
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func homeHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := grove.Data{
			"highlights": highlightsToAny(homeHighlights),
			"repo_url":   repoURL,
			"docs_url":   docsURL,
		}
		renderPage(w, r, eng, "pages/home", data)
	}
}

func docsHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := grove.Data{
			"doc_links": docLinksToAny(docLinks),
			"repo_url":  repoURL,
		}
		renderPage(w, r, eng, "pages/docs", data)
	}
}

func notFoundHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		renderPage(w, r, eng, "pages/404", grove.Data{})
	}
}

// ─── Response assembly ──────────────────────────────────────────────────────

func renderPage(w http.ResponseWriter, r *http.Request, eng *grove.Engine, name string, data grove.Data) {
	result, err := eng.Render(r.Context(), name, data)
	if err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeResult(w, result)
}

// importmapHTML is the `<script type="importmap">...</script>` block built
// once at startup from the asset manifest. writeResult injects it into the
// <head> so module scripts can use bare specifiers across fingerprinted files.
var importmapHTML string

func writeResult(w http.ResponseWriter, result grove.RenderResult) {
	body := result.Body

	body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)
	body = strings.Replace(body, "<!-- HEAD_IMPORTMAP -->", importmapHTML, 1)
	body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)
	body = strings.Replace(body, "<!-- HEAD_META -->", renderMeta(result.Meta), 1)
	body = strings.Replace(body, "<!-- HEAD_HOIST -->", result.GetHoisted("head"), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

func renderMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		attr := "name"
		if strings.HasPrefix(k, "og:") || strings.HasPrefix(k, "twitter:") {
			attr = "property"
		}
		fmt.Fprintf(&sb, `  <meta %s="%s" content="%s">`+"\n", attr, k, meta[k])
	}
	return sb.String()
}

// ─── Engine setup ───────────────────────────────────────────────────────────

func buildEngine(baseDir string) (*grove.Engine, *assets.Builder, string, error) {
	templateDir := filepath.Join(baseDir, "templates")
	distDir := filepath.Join(baseDir, "dist")

	builder := assets.NewWithDefaults(assets.Config{
		SourceDir:      templateDir,
		OutputDir:      distDir,
		URLPrefix:      "/dist",
		CSSTransformer: minify.New(),
		JSTransformer:  minify.New(),
		ManifestPath:   filepath.Join(distDir, "manifest.json"),
	})

	manifest, err := builder.Build()
	if err != nil {
		return nil, nil, "", fmt.Errorf("asset build: %w", err)
	}

	// Build a browser importmap from the manifest so module scripts can use
	// bare specifiers (e.g. `import { copyText } from "components/.../clipboard"`)
	// and still resolve to fingerprinted URLs. See docs/spec/esm-support.md.
	importmap := esm.Importmap(manifest, esm.Options{StripJSExt: true})

	eng := grove.New(
		grove.WithStore(grove.NewFileSystemStore(templateDir)),
		grove.WithAssetResolver(manifest.Resolve),
		grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 5000}),
	)

	eng.SetGlobal("site_name", "Grove")
	eng.SetGlobal("year", "2026")
	eng.SetGlobal("repo_url", repoURL)
	eng.SetGlobal("docs_url", docsURL)
	eng.SetGlobal("pkg_url", pkgURL)
	eng.SetGlobal("issues_url", repoURL+"/issues")

	// truncate: returns the first N words of a string + "…"
	eng.RegisterFilter("truncate", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		s := v.String()
		limit := 20
		if len(args) > 0 {
			n, _ := args[0].ToInt64()
			limit = int(n)
		}
		words := strings.Fields(s)
		if len(words) <= limit {
			return grove.StringValue(s), nil
		}
		return grove.StringValue(strings.Join(words[:limit], " ") + "…"), nil
	}))

	return eng, builder, importmap, nil
}

// ─── Wire-up ────────────────────────────────────────────────────────────────

func routes(eng *grove.Engine, builder *assets.Builder, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", homeHandler(eng))
	r.Get("/docs", docsHandler(eng))

	// Static assets (not pipeline-hashed): globals CSS, SVGs, etc.
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Pipeline-hashed assets.
	distPattern, distHandler := builder.Route()
	r.Handle(distPattern+"*", distHandler)

	// 404 fallback.
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		notFoundHandler(eng)(w, req)
	})

	return r
}

func getPort() string {
	port := "3000"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return port
}

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)

	eng, builder, imap, err := buildEngine(baseDir)
	if err != nil {
		log.Fatal(err)
	}
	importmapHTML = imap

	h := routes(eng, builder, filepath.Join(baseDir, "static"))

	port := getPort()
	fmt.Printf("Grove starter listening on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, h))
}
