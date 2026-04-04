# Blog Example Overhaul — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the blog example from a shallow 3-post demo into a realistic multi-author tech blog with tags, authors, filtering, and working navigation.

**Architecture:** Data loaded from JSON files at startup. Go structs with GroveResolve for template binding. Chi router with 6 routes. Templates use components with slots, filter chains, and template inheritance. All navigation links (tags, authors, breadcrumbs) connect to real pages.

**Tech Stack:** Go 1.24, Grove template engine, Chi router, JSON data files

**Spec:** `docs/superpowers/specs/2026-04-04-examples-expansion-design.md` — Example 1: Blog

---

### Task 1: Create JSON data files

**Files:**
- Create: `examples/blog/data/authors.json`
- Create: `examples/blog/data/tags.json`
- Create: `examples/blog/data/posts.json`

- [ ] **Step 1: Create the data directory**

```bash
mkdir -p examples/blog/data
```

- [ ] **Step 2: Write authors.json**

```json
[
  {
    "name": "Jane Chen",
    "slug": "jane-chen",
    "bio": "Senior Go developer and Grove core contributor. Writes about language internals, performance, and best practices.",
    "avatar_url": "/static/avatars/jane.jpg",
    "role": "Core Team"
  },
  {
    "name": "Marcus Rivera",
    "slug": "marcus-rivera",
    "bio": "Frontend engineer who fell in love with server-side templating. Focuses on components, design systems, and developer experience.",
    "avatar_url": "/static/avatars/marcus.jpg",
    "role": "Contributor"
  },
  {
    "name": "Priya Sharma",
    "slug": "priya-sharma",
    "bio": "DevOps engineer and technical writer. Covers deployment, tooling, and release engineering for the Grove ecosystem.",
    "avatar_url": "/static/avatars/priya.jpg",
    "role": "Maintainer"
  }
]
```

- [ ] **Step 3: Write tags.json**

```json
[
  {"name": "Go", "slug": "go", "color": "blue"},
  {"name": "Templates", "slug": "templates", "color": "purple"},
  {"name": "Web Dev", "slug": "web-dev", "color": "green"},
  {"name": "Components", "slug": "components", "color": "orange"},
  {"name": "Performance", "slug": "performance", "color": "red"},
  {"name": "Getting Started", "slug": "getting-started", "color": "teal"},
  {"name": "Releases", "slug": "releases", "color": "gray"}
]
```

- [ ] **Step 4: Write posts.json**

Create 12 posts. Each post has title, slug, author_slug, date (YYYY-MM-DD), tags (array of tag slugs), summary (1 sentence), body (2-3 paragraphs of HTML about the topic — use `<p>` tags), and draft (bool). Two posts should have `"draft": true`, the rest `"draft": false`.

The posts should reference real Grove features (template inheritance, components, filters, etc.) and be spread across all 3 authors and most tags. Dates should span March–April 2026.

```json
[
  {
    "title": "Getting Started with Grove",
    "slug": "getting-started-with-grove",
    "author_slug": "jane-chen",
    "date": "2026-03-10",
    "tags": ["go", "getting-started"],
    "summary": "A beginner's guide to setting up Grove and rendering your first template.",
    "body": "<p>Grove is a template engine built from scratch in Go. It compiles templates to bytecode and runs them on a lightweight virtual machine, giving you both safety and speed. Unlike text/template, Grove is designed for building web applications with first-class support for components, inheritance, and asset management.</p><p>To get started, add Grove to your Go module with <code>go get grove</code>. Create an engine, point it at a template directory, and call <code>Render</code> with your data. That's it — Grove handles compilation, caching, and execution automatically.</p><p>In this post we'll walk through a minimal example: a single template that displays a greeting, then build up to a full page with a layout, components, and dynamic data from Go structs.</p>",
    "draft": false
  },
  {
    "title": "Building Reusable Components",
    "slug": "building-reusable-components",
    "author_slug": "marcus-rivera",
    "date": "2026-03-13",
    "tags": ["components", "templates"],
    "summary": "How to create component libraries with props, slots, and fills in Grove.",
    "body": "<p>Components are the building blocks of any modern UI. In Grove, a component is a template file that declares its interface using <code>{% props %}</code> and <code>{% slot %}</code> tags. The caller passes data through props and injects content through fills.</p><p>What makes Grove components powerful is scope isolation. Props create a clean boundary — the component only sees what you explicitly pass. But fills see the <em>caller's</em> scope, so you can use your page data inside fill blocks without threading everything through props.</p><p>Start by creating a <code>components/</code> directory in your templates folder. Each component file declares its props at the top, provides sensible defaults, and uses slots for flexible content areas. You'll be surprised how quickly a small library of cards, buttons, and alerts speeds up your development.</p>",
    "draft": false
  },
  {
    "title": "Template Inheritance in Depth",
    "slug": "template-inheritance-in-depth",
    "author_slug": "jane-chen",
    "date": "2026-03-16",
    "tags": ["templates", "getting-started"],
    "summary": "Master multi-level template inheritance with extends, block, and super().",
    "body": "<p>Template inheritance lets you define a base layout once and override specific sections in child templates. Grove supports unlimited inheritance depth — a child can extend a parent that extends a grandparent, each layer adding or replacing blocks as needed.</p><p>Blocks are the override points. Define them in the base template with default content, then override them in child templates. Need the parent's content too? Call <code>super()</code> to include it alongside your additions. This is especially useful for navigation blocks where you want to extend rather than replace.</p><p>A common pattern is three levels: a site-wide base with the HTML shell, a section layout that adds a sidebar or breadcrumbs, and individual pages that fill in the content. Each level only concerns itself with its own additions.</p>",
    "draft": false
  },
  {
    "title": "Grove v1.0 Released",
    "slug": "grove-v1-released",
    "author_slug": "priya-sharma",
    "date": "2026-03-19",
    "tags": ["releases"],
    "summary": "Announcing the stable release of Grove with a complete feature set for production use.",
    "body": "<p>After months of development and testing, Grove v1.0 is now available. This release marks the template engine as production-ready, with a stable API, comprehensive test coverage, and documented behavior for every feature.</p><p>The v1.0 release includes all core features: variables, filters, control flow, template inheritance, components with slots, macros, asset management, meta tags, hoisting, and sandboxing. The bytecode compiler and VM have been optimized for performance, and the engine is safe for concurrent use.</p><p>We've also published four example applications — a blog, an e-commerce store, a documentation site, and an email renderer — that demonstrate every major feature in context. Check them out in the <code>examples/</code> directory.</p>",
    "draft": false
  },
  {
    "title": "Mastering Filters",
    "slug": "mastering-filters",
    "author_slug": "marcus-rivera",
    "date": "2026-03-22",
    "tags": ["templates", "getting-started"],
    "summary": "A tour of Grove's 40+ built-in filters and how to chain them effectively.",
    "body": "<p>Filters transform values in your templates using the pipe syntax: <code>{{ name | upper }}</code>. Grove ships with over 40 built-in filters covering strings, collections, numbers, HTML, and dates. You can chain multiple filters together, and each one passes its output to the next.</p><p>String filters like <code>upper</code>, <code>lower</code>, <code>title</code>, <code>trim</code>, and <code>truncate</code> handle common text formatting. Collection filters like <code>length</code>, <code>join</code>, <code>first</code>, <code>last</code>, and <code>sort</code> work on arrays and maps. The <code>default</code> filter provides fallback values for nil or empty variables.</p><p>For advanced use cases, you can register custom filters in Go. A custom filter is a function that takes a value and optional arguments, returning a transformed value. This is how the store example implements its <code>currency</code> filter for formatting prices.</p>",
    "draft": false
  },
  {
    "title": "Performance Tuning Your Templates",
    "slug": "performance-tuning",
    "author_slug": "jane-chen",
    "date": "2026-03-25",
    "tags": ["performance", "go"],
    "summary": "How Grove's bytecode compiler and VM pool deliver fast template rendering.",
    "body": "<p>Grove compiles templates to bytecode once, then executes them on a stack-based virtual machine. Compiled bytecode is immutable and shared across goroutines, so there's no compilation overhead on subsequent renders. The engine maintains an LRU cache (default 512 entries) of compiled templates.</p><p>VM instances are pooled via <code>sync.Pool</code>, which means rendering doesn't allocate a new VM for every request. Combined with the bytecode cache, this gives Grove predictable performance characteristics under load — no GC pressure spikes from template compilation.</p><p>If you're seeing slower renders than expected, check your template structure. Deeply nested component trees and long filter chains add overhead. Consider using <code>capture</code> to pre-render expensive sections, and use <code>include</code> instead of <code>component</code> when you don't need prop isolation.</p>",
    "draft": false
  },
  {
    "title": "Component Slots and Fills",
    "slug": "component-slots-and-fills",
    "author_slug": "marcus-rivera",
    "date": "2026-03-28",
    "tags": ["components", "web-dev"],
    "summary": "Deep dive into named slots, default content, and fill scope rules.",
    "body": "<p>Slots are the content injection points in a Grove component. The default slot captures everything between the opening and closing component tags. Named slots let you inject content into specific regions — a card component might have slots for <code>header</code>, <code>body</code>, and <code>footer</code>.</p><p>Every slot can have default content that renders when the caller doesn't provide a fill. This is useful for optional sections — a card's <code>tags</code> slot might default to empty, while its <code>body</code> slot shows placeholder text.</p><p>The most important rule to remember: fills see the caller's scope, not the component's scope. This means you can reference your page variables inside a fill block without passing them as props. It's what makes Grove components practical for real-world use — you don't have to thread every variable through the prop chain.</p>",
    "draft": false
  },
  {
    "title": "Deploying Grove in Production",
    "slug": "deploying-grove-in-production",
    "author_slug": "priya-sharma",
    "date": "2026-03-31",
    "tags": ["go", "web-dev"],
    "summary": "Practical advice for running Grove-powered applications in production.",
    "body": "<p>Deploying a Grove application is no different from deploying any Go HTTP server. Compile your binary, include your template and static directories, and run it behind a reverse proxy. Grove has no runtime dependencies beyond the Go standard library.</p><p>For template management, the <code>FileSystemStore</code> loads templates from disk on first access and the engine caches compiled bytecode. In production, templates rarely change, so the cache stays warm. If you need to reload templates without restarting, you can call the engine's cache invalidation methods.</p><p>One thing to watch: if you're serving user-provided templates (like a CMS), always use sandboxing. The <code>WithSandbox</code> option lets you whitelist specific tags and filters, and set a maximum loop iteration count. This prevents template authors from accessing dangerous features or creating infinite loops.</p>",
    "draft": false
  },
  {
    "title": "Macro Libraries for Common Patterns",
    "slug": "macro-libraries",
    "author_slug": "jane-chen",
    "date": "2026-04-01",
    "tags": ["templates", "components"],
    "summary": "Organize reusable template logic into importable macro libraries.",
    "body": "<p>Macros are reusable template functions defined with <code>{% macro name(args) %}</code>. Unlike components, macros don't have their own scope or slot system — they're lightweight functions that take arguments and return rendered content. Think of them as template-level utility functions.</p><p>The real power comes from <code>{% import %}</code>. You can define a library of macros in a single file and import them into any template. The store example uses this for pricing display — a <code>pricing.grov</code> file exports macros for formatted prices, star ratings, and discount badges.</p><p>A good rule of thumb: use components when you need scope isolation and content slots (cards, alerts, modals). Use macros when you need a simple function that takes values and returns HTML (formatted prices, icon helpers, badge generators). Both have their place in a well-organized template codebase.</p>",
    "draft": false
  },
  {
    "title": "Asset Management with Grove",
    "slug": "asset-management",
    "author_slug": "priya-sharma",
    "date": "2026-04-02",
    "tags": ["web-dev", "templates"],
    "summary": "How Grove collects and organizes CSS, JS, and meta tags across nested templates.",
    "body": "<p>Grove's asset system solves a common web templating problem: components deep in the render tree need to declare their CSS and JS dependencies, but those assets must appear in the HTML <code>&lt;head&gt;</code> or before <code>&lt;/body&gt;</code>. The <code>{% asset %}</code> tag collects declarations during rendering, and the engine returns them in the <code>RenderResult</code>.</p><p>Assets have types (<code>stylesheet</code>, <code>script</code>, <code>preload</code>) and priority levels. Higher priority assets appear first within their type group. The <code>RenderResult.HeadHTML()</code> method generates link tags for stylesheets, while <code>FootHTML()</code> generates script tags. Your application replaces placeholder comments in the base template with these generated strings.</p><p>The same pattern works for meta tags (<code>{% meta %}</code>) and hoisted content (<code>{% hoist %}</code>). A blog post template can set its own Open Graph tags, and a component can hoist inline styles to the head — all collected automatically during rendering.</p>",
    "draft": false
  },
  {
    "title": "The GroveResolve Interface",
    "slug": "grove-resolve-interface",
    "author_slug": "jane-chen",
    "date": "2026-04-03",
    "tags": ["go", "getting-started"],
    "summary": "How to expose Go struct fields to templates using the GroveResolve method.",
    "body": "<p>When you pass a Go struct to a template, Grove needs a way to access its fields by name. The <code>GroveResolve</code> method provides this bridge. Any type that implements <code>GroveResolve(key string) (any, bool)</code> can be used directly in templates — access fields with dot notation like <code>{{ post.title }}</code>.</p><p>The method is a simple switch statement that maps string keys to struct fields. Return the value and <code>true</code> for known keys, or <code>nil, false</code> for unknown ones. You can also expose computed properties that don't correspond to a struct field — the store example exposes <code>on_sale</code> as <code>p.SalePrice > 0</code>.</p><p>For slices of custom types, you need to convert them to <code>[]any</code> before returning. This is because Grove's VM works with interface values. A helper function or a conversion in the GroveResolve method handles this cleanly.</p>",
    "draft": true
  },
  {
    "title": "What's Coming in Grove v2.0",
    "slug": "grove-v2-preview",
    "author_slug": "priya-sharma",
    "date": "2026-04-04",
    "tags": ["releases"],
    "summary": "A preview of upcoming features planned for the next major release.",
    "body": "<p>Grove v2.0 is in early planning and we're excited to share the roadmap. The focus is on developer experience: better error messages with source location context, a template language server for editor integration, and a development mode with automatic reload.</p><p>On the engine side, we're exploring partial template compilation for faster startup, streaming rendering for large pages, and a plugin system for custom tags. The VM will also get optimizations for common patterns like filter chains and nested component rendering.</p><p>We'd love your input on priorities. Join the discussion on the Grove GitHub repository or reach out on the community forum. The v2.0 milestone tracks all planned work.</p>",
    "draft": true
  }
]
```

- [ ] **Step 5: Commit data files**

```bash
git add examples/blog/data/
git commit -m "blog: Add JSON data files for authors, tags, and posts"
```

---

### Task 2: Rewrite main.go — types and data loading

**Files:**
- Modify: `examples/blog/main.go`

- [ ] **Step 1: Write the complete main.go**

Replace the entire `examples/blog/main.go` with the following. This includes types, data loading, all route handlers, and the writeResult helper.

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

type Author struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}

func (a Author) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return a.Name, true
	case "slug":
		return a.Slug, true
	case "bio":
		return a.Bio, true
	case "avatar_url":
		return a.AvatarURL, true
	case "role":
		return a.Role, true
	}
	return nil, false
}

type Tag struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

func (t Tag) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return t.Name, true
	case "slug":
		return t.Slug, true
	case "color":
		return t.Color, true
	}
	return nil, false
}

type PostJSON struct {
	Title      string   `json:"title"`
	Slug       string   `json:"slug"`
	AuthorSlug string   `json:"author_slug"`
	Date       string   `json:"date"`
	Tags       []string `json:"tags"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	Draft      bool     `json:"draft"`
}

type Post struct {
	Title   string
	Slug    string
	Author  Author
	Date    string
	Tags    []Tag
	Summary string
	Body    string
	Draft   bool
}

func (p Post) GroveResolve(key string) (any, bool) {
	switch key {
	case "title":
		return p.Title, true
	case "slug":
		return p.Slug, true
	case "author":
		return p.Author, true
	case "date":
		return p.Date, true
	case "tags":
		tags := make([]any, len(p.Tags))
		for i, t := range p.Tags {
			tags[i] = t
		}
		return tags, true
	case "summary":
		return p.Summary, true
	case "body":
		return p.Body, true
	case "draft":
		return p.Draft, true
	}
	return nil, false
}

// --- Data loading ---

func loadJSON(baseDir, filename string, v any) {
	data, err := os.ReadFile(filepath.Join(baseDir, "data", filename))
	if err != nil {
		log.Fatalf("Failed to load %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		log.Fatalf("Failed to parse %s: %v", filename, err)
	}
}

var (
	authors []Author
	tags    []Tag
	posts   []Post
)

func loadData(baseDir string) {
	loadJSON(baseDir, "authors.json", &authors)
	loadJSON(baseDir, "tags.json", &tags)

	var raw []PostJSON
	loadJSON(baseDir, "posts.json", &raw)

	authorMap := make(map[string]Author)
	for _, a := range authors {
		authorMap[a.Slug] = a
	}
	tagMap := make(map[string]Tag)
	for _, t := range tags {
		tagMap[t.Slug] = t
	}

	posts = make([]Post, 0, len(raw))
	for _, r := range raw {
		p := Post{
			Title:   r.Title,
			Slug:    r.Slug,
			Author:  authorMap[r.AuthorSlug],
			Date:    r.Date,
			Summary: r.Summary,
			Body:    r.Body,
			Draft:   r.Draft,
		}
		for _, ts := range r.Tags {
			if t, ok := tagMap[ts]; ok {
				p.Tags = append(p.Tags, t)
			}
		}
		posts = append(posts, p)
	}
}

// --- Helpers ---

func publishedPosts() []Post {
	var out []Post
	for _, p := range posts {
		if !p.Draft {
			out = append(out, p)
		}
	}
	return out
}

func postsToAny(pp []Post) []any {
	out := make([]any, len(pp))
	for i, p := range pp {
		out[i] = p
	}
	return out
}

func findPostBySlug(slug string) *Post {
	for i := range posts {
		if posts[i].Slug == slug {
			return &posts[i]
		}
	}
	return nil
}

func findAuthorBySlug(slug string) *Author {
	for i := range authors {
		if authors[i].Slug == slug {
			return &authors[i]
		}
	}
	return nil
}

func findTagBySlug(slug string) *Tag {
	for i := range tags {
		if tags[i].Slug == slug {
			return &tags[i]
		}
	}
	return nil
}

func filterByTag(pp []Post, tagSlug string) []Post {
	var out []Post
	for _, p := range pp {
		for _, t := range p.Tags {
			if t.Slug == tagSlug {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func filterByAuthor(pp []Post, authorSlug string) []Post {
	var out []Post
	for _, p := range pp {
		if p.Author.Slug == authorSlug {
			out = append(out, p)
		}
	}
	return out
}

func relatedPosts(post Post, limit int) []Post {
	tagSet := make(map[string]bool)
	for _, t := range post.Tags {
		tagSet[t.Slug] = true
	}
	var out []Post
	for _, p := range publishedPosts() {
		if p.Slug == post.Slug {
			continue
		}
		for _, t := range p.Tags {
			if tagSet[t.Slug] {
				out = append(out, p)
				break
			}
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func tagPostCounts() []any {
	pub := publishedPosts()
	out := make([]any, 0, len(tags))
	for _, t := range tags {
		count := 0
		for _, p := range pub {
			for _, pt := range p.Tags {
				if pt.Slug == t.Slug {
					count++
					break
				}
			}
		}
		out = append(out, map[string]any{
			"tag":   t,
			"count": count,
		})
	}
	return out
}

// --- Handlers ---

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pub := publishedPosts()
		result, err := eng.Render(r.Context(), "index.grov", grove.Data{
			"posts": postsToAny(pub),
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
		post := findPostBySlug(slug)
		if post == nil {
			http.NotFound(w, r)
			return
		}
		related := relatedPosts(*post, 3)
		result, err := eng.Render(r.Context(), "post.grov", grove.Data{
			"post":          *post,
			"related_posts": postsToAny(related),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": post.Title, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func postsHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filtered := publishedPosts()
		title := "All Posts"
		var breadcrumbs []any

		tagSlug := r.URL.Query().Get("tag")
		authorSlug := r.URL.Query().Get("author")

		if tagSlug != "" {
			filtered = filterByTag(filtered, tagSlug)
			if t := findTagBySlug(tagSlug); t != nil {
				title = "Posts tagged \"" + t.Name + "\""
			}
		}
		if authorSlug != "" {
			filtered = filterByAuthor(filtered, authorSlug)
			if a := findAuthorBySlug(authorSlug); a != nil {
				title = "Posts by " + a.Name
			}
		}

		breadcrumbs = []any{
			map[string]any{"label": "Home", "href": "/"},
			map[string]any{"label": title, "href": ""},
		}

		result, err := eng.Render(r.Context(), "post-list.grov", grove.Data{
			"posts":       postsToAny(filtered),
			"title":       title,
			"breadcrumbs": breadcrumbs,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func tagListHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := eng.Render(r.Context(), "tag-list.grov", grove.Data{
			"tag_counts": tagPostCounts(),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Tags", "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func tagHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		tag := findTagBySlug(slug)
		if tag == nil {
			http.NotFound(w, r)
			return
		}
		filtered := filterByTag(publishedPosts(), slug)
		result, err := eng.Render(r.Context(), "post-list.grov", grove.Data{
			"posts": postsToAny(filtered),
			"title": "Posts tagged \"" + tag.Name + "\"",
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": "Tags", "href": "/tags"},
				map[string]any{"label": tag.Name, "href": ""},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeResult(w, result)
	}
}

func authorHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		author := findAuthorBySlug(slug)
		if author == nil {
			http.NotFound(w, r)
			return
		}
		filtered := filterByAuthor(publishedPosts(), slug)
		result, err := eng.Render(r.Context(), "author.grov", grove.Data{
			"author": *author,
			"posts":  postsToAny(filtered),
			"breadcrumbs": []any{
				map[string]any{"label": "Home", "href": "/"},
				map[string]any{"label": author.Name, "href": ""},
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
	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("site_name", "Grove Blog")
	eng.SetGlobal("current_year", "2026")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/post/{slug}", postHandler(eng))
	r.Get("/posts", postsHandler(eng))
	r.Get("/tags", tagListHandler(eng))
	r.Get("/tag/{slug}", tagHandler(eng))
	r.Get("/author/{slug}", authorHandler(eng))

	staticDir := filepath.Join(baseDir, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	fmt.Println("Grove Blog listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Post{}
	_ interface{ GroveResolve(string) (any, bool) } = Author{}
	_ interface{ GroveResolve(string) (any, bool) } = Tag{}
)
```

- [ ] **Step 2: Verify it compiles**

```bash
cd examples/blog && go build ./...
```

Expected: Compilation succeeds (templates don't exist yet but Go code compiles).

- [ ] **Step 3: Commit**

```bash
git add examples/blog/main.go
git commit -m "blog: Rewrite main.go with data loading, types, and 6 route handlers"
```

---

### Task 3: Create component templates

**Files:**
- Create: `examples/blog/templates/components/nav.grov`
- Create: `examples/blog/templates/components/footer.grov`
- Create: `examples/blog/templates/components/card.grov`
- Create: `examples/blog/templates/components/tag-badge.grov`
- Create: `examples/blog/templates/components/author-card.grov`
- Create: `examples/blog/templates/components/breadcrumbs.grov`

- [ ] **Step 1: Write nav.grov**

```
{% props site_name %}
<nav class="nav">
  <a href="/" class="nav-brand">{{ site_name }}</a>
  <div class="nav-links">
    <a href="/" class="nav-link">Home</a>
    <a href="/tags" class="nav-link">Tags</a>
    {% slot %}{% endslot %}
  </div>
</nav>
```

- [ ] **Step 2: Write footer.grov**

```
{% props year %}
<footer class="footer">
  <p>&copy; {{ year }} Grove Blog. Built with the <a href="https://github.com/grove">Grove</a> template engine.</p>
</footer>
```

- [ ] **Step 3: Write card.grov**

This is the primary component demonstrating props, slots, and fills.

```
{% props title, summary, href="#", date="", author_name="", author_slug="" %}
<article class="card">
  <h2 class="card-title">
    <a href="{{ href }}">{{ title }}</a>
  </h2>
  <div class="card-meta">
    {% if date %}<time>{{ date }}</time>{% endif %}
    {% if author_name %}
      <span class="card-author">by <a href="/author/{{ author_slug }}">{{ author_name }}</a></span>
    {% endif %}
  </div>
  <p class="card-summary">{{ summary | truncate(150) }}</p>
  <div class="card-tags">
    {% slot "tags" %}{% endslot %}
  </div>
</article>
```

- [ ] **Step 4: Write tag-badge.grov**

```
{% props label, color="gray", slug="" %}
{% if slug %}
  <a href="/tag/{{ slug }}" class="tag tag-{{ color }}">{{ label }}</a>
{% else %}
  <span class="tag tag-{{ color }}">{{ label }}</span>
{% endif %}
```

- [ ] **Step 5: Write author-card.grov**

```
{% props name, slug, bio="", avatar_url="", role="" %}
<div class="author-card">
  {% if avatar_url %}
    <div class="author-avatar">
      <img src="{{ avatar_url }}" alt="{{ name }}">
    </div>
  {% endif %}
  <div class="author-info">
    <h3><a href="/author/{{ slug }}">{{ name }}</a></h3>
    {% if role %}<span class="author-role">{{ role }}</span>{% endif %}
    {% if bio %}<p class="author-bio">{{ bio }}</p>{% endif %}
    {% slot %}{% endslot %}
  </div>
</div>
```

- [ ] **Step 6: Write breadcrumbs.grov**

This component uses `{% include %}` (not `{% component %}`), so it inherits the calling scope and reads `breadcrumbs` directly — no props needed.

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

- [ ] **Step 7: Remove old component files that are no longer needed**

Delete `templates/components/button.grov`, `templates/components/alert.grov`, and `templates/pages/styleguide.grov` — these are not part of the new blog design.

```bash
rm -f examples/blog/templates/components/button.grov
rm -f examples/blog/templates/components/alert.grov
rm -f examples/blog/templates/pages/styleguide.grov
```

- [ ] **Step 8: Commit**

```bash
git add examples/blog/templates/components/
git add examples/blog/templates/pages/
git commit -m "blog: Add component templates — card, tag-badge, author-card, breadcrumbs, nav, footer"
```

---

### Task 4: Create page templates

**Files:**
- Modify: `examples/blog/templates/base.grov`
- Modify: `examples/blog/templates/index.grov`
- Modify: `examples/blog/templates/post.grov`
- Create: `examples/blog/templates/post-list.grov`
- Create: `examples/blog/templates/tag-list.grov`
- Create: `examples/blog/templates/author.grov`

- [ ] **Step 1: Rewrite base.grov**

```
{% asset "/static/style.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{% block title %}Grove Blog{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  {% component "components/nav.grov" site_name=site_name %}{% endcomponent %}
  <main class="container">
    {% block content %}{% endblock %}
  </main>
  {% component "components/footer.grov" year=current_year %}{% endcomponent %}
  <!-- FOOT_ASSETS -->
</body>
</html>
```

- [ ] **Step 2: Rewrite index.grov**

```
{% extends "base.grov" %}

{% block title %}Home &mdash; Grove Blog{% endblock %}

{% block content %}
{% meta name="description" content="A tech blog built with the Grove template engine" %}

<h1>Latest Posts</h1>
<div class="post-grid">
  {% for post in posts %}
    {% component "components/card.grov" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date author_name=post.author.name author_slug=post.author.slug %}
      {% fill "tags" %}
        {% for tag in post.tags %}
          {% component "components/tag-badge.grov" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
        {% endfor %}
      {% endfill %}
    {% endcomponent %}
  {% empty %}
    <p class="empty-state">No posts yet.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 3: Rewrite post.grov**

```
{% extends "base.grov" %}

{% block title %}{{ post.title }} &mdash; Grove Blog{% endblock %}

{% block content %}
{% meta name="description" content=post.summary %}
{% meta property="og:title" content=post.title %}

{% include "components/breadcrumbs.grov" %}

{% if post.draft %}
  <div class="notice notice-warning">
    This post is a <strong>draft</strong> and is not yet published.
  </div>
{% endif %}

<article class="article">
  <header class="article-header">
    <h1>{{ post.title }}</h1>
    <div class="article-meta">
      <time>{{ post.date }}</time>
      <span class="article-author">
        by <a href="/author/{{ post.author.slug }}">{{ post.author.name }}</a>
      </span>
    </div>
    <div class="tag-list">
      {% for tag in post.tags %}
        {% component "components/tag-badge.grov" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
      {% endfor %}
    </div>
  </header>

  <div class="article-body">
    {{ post.body | safe }}
  </div>
</article>

{% component "components/author-card.grov" name=post.author.name slug=post.author.slug bio=post.author.bio avatar_url=post.author.avatar_url role=post.author.role %}{% endcomponent %}

{% if related_posts | length > 0 %}
  <section class="related-posts">
    <h2>Related Posts</h2>
    <div class="post-grid">
      {% for rp in related_posts %}
        {% component "components/card.grov" title=rp.title summary=rp.summary href="/post/" ~ rp.slug date=rp.date %}
          {% fill "tags" %}
            {% for tag in rp.tags %}
              {% component "components/tag-badge.grov" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
            {% endfor %}
          {% endfill %}
        {% endcomponent %}
      {% endfor %}
    </div>
  </section>
{% endif %}

<div class="back-link">
  <a href="/">&larr; Back to all posts</a>
</div>
{% endblock %}
```

- [ ] **Step 4: Write post-list.grov**

This template is shared by `/posts`, `/tag/{slug}`, and `/author/{slug}` routes — any filtered post listing.

```
{% extends "base.grov" %}

{% block title %}{{ title }} &mdash; Grove Blog{% endblock %}

{% block content %}
{% include "components/breadcrumbs.grov" %}

<h1>{{ title }}</h1>
<div class="post-grid">
  {% for post in posts %}
    {% component "components/card.grov" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date author_name=post.author.name author_slug=post.author.slug %}
      {% fill "tags" %}
        {% for tag in post.tags %}
          {% component "components/tag-badge.grov" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
        {% endfor %}
      {% endfill %}
    {% endcomponent %}
  {% empty %}
    <p class="empty-state">No posts found.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 5: Write tag-list.grov**

```
{% extends "base.grov" %}

{% block title %}Tags &mdash; Grove Blog{% endblock %}

{% block content %}
{% include "components/breadcrumbs.grov" %}

<h1>Tags</h1>
<div class="tag-grid">
  {% for item in tag_counts %}
    <a href="/tag/{{ item.tag.slug }}" class="tag-card tag-card-{{ item.tag.color }}">
      <span class="tag-card-name">{{ item.tag.name }}</span>
      <span class="tag-card-count">{{ item.count }} {{ item.count == 1 ? "post" : "posts" }}</span>
    </a>
  {% empty %}
    <p class="empty-state">No tags yet.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 6: Write author.grov**

```
{% extends "base.grov" %}

{% block title %}{{ author.name }} &mdash; Grove Blog{% endblock %}

{% block content %}
{% include "components/breadcrumbs.grov" %}

{% component "components/author-card.grov" name=author.name slug=author.slug bio=author.bio avatar_url=author.avatar_url role=author.role %}{% endcomponent %}

<h2>Posts by {{ author.name }}</h2>
<div class="post-grid">
  {% for post in posts %}
    {% component "components/card.grov" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date %}
      {% fill "tags" %}
        {% for tag in post.tags %}
          {% component "components/tag-badge.grov" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
        {% endfor %}
      {% endfill %}
    {% endcomponent %}
  {% empty %}
    <p class="empty-state">No posts by this author yet.</p>
  {% endfor %}
</div>
{% endblock %}
```

- [ ] **Step 7: Commit**

```bash
git add examples/blog/templates/
git commit -m "blog: Add page templates — index, post, post-list, tag-list, author with breadcrumbs"
```

---

### Task 5: Update stylesheet

**Files:**
- Modify: `examples/blog/static/style.css`

- [ ] **Step 1: Replace the stylesheet**

Replace `examples/blog/static/style.css` with a comprehensive stylesheet that supports all the new templates. The stylesheet should include:

- CSS custom properties for colors (brand green, text, background, borders, tag colors for each: blue, purple, green, orange, red, teal, gray)
- Base reset and typography (system font stack, line heights, link colors)
- Layout: `.container` max-width 900px centered, `.nav` with brand + links, `.footer`
- `.post-grid` — CSS grid, 1 column on mobile, 2 columns on wider screens
- `.card` — border, padding, hover shadow, `.card-title`, `.card-meta`, `.card-summary`, `.card-tags`
- `.tag` — inline pill badge, color variants for each tag color (`.tag-blue`, `.tag-purple`, etc.)
- `.tag-grid` — grid of tag cards for `/tags` page
- `.tag-card` — larger clickable tag card with name and post count
- `.breadcrumb` — inline breadcrumb trail with separator
- `.author-card` — flex layout with avatar, name, role, bio
- `.article` — full post view with `.article-header`, `.article-meta`, `.article-body` (readable line lengths, paragraph spacing)
- `.notice` — warning/info notice boxes (`.notice-warning`)
- `.related-posts` — section with heading
- `.empty-state` — styled "no results" message
- `.back-link` — return navigation link
- Responsive: adjust grid columns and spacing at smaller breakpoints

Use the existing blog's CSS as a style reference (brand green `#2E6740`, clean/minimal aesthetic). Keep the same design language but add the new classes.

- [ ] **Step 2: Commit**

```bash
git add examples/blog/static/style.css
git commit -m "blog: Update stylesheet for new templates and components"
```

---

### Task 6: Build and verify

- [ ] **Step 1: Build the blog example**

```bash
cd examples/blog && go build ./...
```

Expected: No compilation errors.

- [ ] **Step 2: Run the blog and verify all routes**

```bash
cd examples/blog && go run main.go &
sleep 2
curl -s http://localhost:3000/ | head -20
curl -s http://localhost:3000/post/getting-started-with-grove | head -20
curl -s http://localhost:3000/posts?tag=go | head -20
curl -s http://localhost:3000/tags | head -20
curl -s http://localhost:3000/tag/go | head -20
curl -s http://localhost:3000/author/jane-chen | head -20
kill %1
```

Expected: All routes return HTML without errors. Each page should have proper title, navigation, breadcrumbs, and content.

- [ ] **Step 3: Verify all navigation links are valid**

Check that:
- Tag badges on posts link to `/tag/{slug}` with real tag slugs
- Author links go to `/author/{slug}` with real author slugs
- Breadcrumbs link to real pages
- "Related posts" appear on post pages

- [ ] **Step 4: Final commit if any fixes were needed**

```bash
git add examples/blog/
git commit -m "blog: Fix any issues found during verification"
```
