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
				map[string]any{"label": "Authors", "href": "/posts"},
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

	// Serve colocated JS from component directories
	r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir(templateDir))))

	fmt.Println("Grove Blog listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", r))
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = Post{}
	_ interface{ GroveResolve(string) (any, bool) } = Author{}
	_ interface{ GroveResolve(string) (any, bool) } = Tag{}
)
