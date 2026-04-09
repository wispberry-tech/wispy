package benchmarks

// Template strings for each engine and scenario.
// Each engine has its own syntax; the templates are semantically equivalent.

// --- Engine name constants ---

const (
	EngGrove        = "Grove"
	EngHTMLTemplate = "HTMLTemplate"
	EngTextTemplate = "TextTemplate"
	EngPongo2       = "Pongo2"
	EngJet          = "Jet"
	EngLiquid       = "Liquid"
)

// --- Simple: variable interpolation ---

var SimpleTemplates = map[string]string{
	EngGrove:        `Hello, {% name %}! You have {% count %} messages.`,
	EngHTMLTemplate: `Hello, {{.Name}}! You have {{.Count}} messages.`,
	EngTextTemplate: `Hello, {{.Name}}! You have {{.Count}} messages.`,
	EngPongo2:       `Hello, {{ name }}! You have {{ count }} messages.`,
	EngJet:          `Hello, {{ .Name }}! You have {{ .Count }} messages.`,
	EngLiquid:       `Hello, {{ name }}! You have {{ count }} messages.`,
}

// --- Loop: iterate over a slice ---

var LoopTemplates = map[string]string{
	EngGrove: `<ul><For each={items} as="item"><li>{% item %}</li></For></ul>`,
	EngHTMLTemplate: `<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>`,
	EngTextTemplate: `<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>`,
	EngPongo2: `<ul>{% for item in items %}<li>{{ item }}</li>{% endfor %}</ul>`,
	EngJet:    `<ul>{{range _, item := .Items}}<li>{{item}}</li>{{end}}</ul>`,
	EngLiquid: `<ul>{% for item in items %}<li>{{ item }}</li>{% endfor %}</ul>`,
}

// --- Conditional: if/elif/else ---

var ConditionalTemplates = map[string]string{
	EngGrove:        `<If test={role == "admin"}>Admin Panel<ElseIf test={role == "mod"} />Mod Tools<Else />User Dashboard</If>`,
	EngHTMLTemplate: `{{if eq .Role "admin"}}Admin Panel{{else if eq .Role "mod"}}Mod Tools{{else}}User Dashboard{{end}}`,
	EngTextTemplate: `{{if eq .Role "admin"}}Admin Panel{{else if eq .Role "mod"}}Mod Tools{{else}}User Dashboard{{end}}`,
	EngPongo2:       `{% if role == "admin" %}Admin Panel{% elif role == "mod" %}Mod Tools{% else %}User Dashboard{% endif %}`,
	EngJet:          `{{if .Role == "admin"}}Admin Panel{{else if .Role == "mod"}}Mod Tools{{else}}User Dashboard{{end}}`,
	EngLiquid:       `{% if role == "admin" %}Admin Panel{% elsif role == "mod" %}Mod Tools{% else %}User Dashboard{% endif %}`,
}

// --- Complex: blog post listing ---

var ComplexTemplates = map[string]string{
	EngGrove: `<div class="posts">
<For each={posts} as="post">
<article class="post"><If test={post.featured}> featured</If>">
  <h2>{% post.title %}</h2>
  <span class="meta">By {% post.author %} on {% post.date %}</span>
  <p>{% post.excerpt %}</p>
  <If test={post.tags}><div class="tags"><For each={post.tags} as="tag"><span class="tag">{% tag %}</span></For></div></If>
</article>
</For>
</div>`,

	EngHTMLTemplate: `<div class="posts">
{{range .Posts}}
<article class="post{{if .Featured}} featured{{end}}">
  <h2>{{.Title}}</h2>
  <span class="meta">By {{.Author}} on {{.Date}}</span>
  <p>{{.Excerpt}}</p>
  {{if .Tags}}<div class="tags">{{range .Tags}}<span class="tag">{{.}}</span>{{end}}</div>{{end}}
</article>
{{end}}
</div>`,

	EngTextTemplate: `<div class="posts">
{{range .Posts}}
<article class="post{{if .Featured}} featured{{end}}">
  <h2>{{.Title}}</h2>
  <span class="meta">By {{.Author}} on {{.Date}}</span>
  <p>{{.Excerpt}}</p>
  {{if .Tags}}<div class="tags">{{range .Tags}}<span class="tag">{{.}}</span>{{end}}</div>{{end}}
</article>
{{end}}
</div>`,

	EngPongo2: `<div class="posts">
{% for post in posts %}
<article class="post{% if post.featured %} featured{% endif %}">
  <h2>{{ post.title }}</h2>
  <span class="meta">By {{ post.author }} on {{ post.date }}</span>
  <p>{{ post.excerpt }}</p>
  {% if post.tags %}<div class="tags">{% for tag in post.tags %}<span class="tag">{{ tag }}</span>{% endfor %}</div>{% endif %}
</article>
{% endfor %}
</div>`,

	EngJet: `<div class="posts">
{{range _, post := .Posts}}
<article class="post{{if post.Featured}} featured{{end}}">
  <h2>{{post.Title}}</h2>
  <span class="meta">By {{post.Author}} on {{post.Date}}</span>
  <p>{{post.Excerpt}}</p>
  {{if post.Tags}}<div class="tags">{{range _, tag := post.Tags}}<span class="tag">{{tag}}</span>{{end}}</div>{{end}}
</article>
{{end}}
</div>`,

	EngLiquid: `<div class="posts">
{% for post in posts %}
<article class="post{% if post.featured %} featured{% endif %}">
  <h2>{{ post.title }}</h2>
  <span class="meta">By {{ post.author }} on {{ post.date }}</span>
  <p>{{ post.excerpt }}</p>
  {% if post.tags %}<div class="tags">{% for tag in post.tags %}<span class="tag">{{ tag }}</span>{% endfor %}</div>{% endif %}
</article>
{% endfor %}
</div>`,

}

// --- Data types ---

// BlogPost is used by stdlib and Jet (struct-based field access).
type BlogPost struct {
	Title    string
	Author   string
	Date     string
	Featured bool
	Tags     []string
	Excerpt  string
}

// SimpleData for stdlib/Jet templates.
type SimpleData struct {
	Name  string
	Count int
}

// LoopData for stdlib/Jet templates.
type LoopData struct {
	Items []string
}

// ConditionalData for stdlib/Jet templates.
type ConditionalData struct {
	Role string
}

// ComplexData for stdlib/Jet templates.
type ComplexData struct {
	Posts []BlogPost
}

// --- Data constructors ---

func NewSimpleMap() map[string]any {
	return map[string]any{"name": "World", "count": 42}
}

func NewSimpleStruct() SimpleData {
	return SimpleData{Name: "World", Count: 42}
}

func NewLoopMap() map[string]any {
	items := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India", "Juliet"}
	return map[string]any{"items": items}
}

func NewLoopStruct() LoopData {
	return LoopData{Items: []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India", "Juliet"}}
}

func NewConditionalMap() map[string]any {
	return map[string]any{"role": "mod"}
}

func NewConditionalStruct() ConditionalData {
	return ConditionalData{Role: "mod"}
}

func newBlogPosts() []BlogPost {
	posts := make([]BlogPost, 10)
	titles := []string{"Getting Started", "Advanced Tips", "Deep Dive", "Best Practices", "Performance Guide",
		"Security 101", "Testing Strategy", "Deploy Guide", "Monitoring", "Retrospective"}
	authors := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Hank", "Iris", "Jack"}
	for i := range posts {
		posts[i] = BlogPost{
			Title:    titles[i],
			Author:   authors[i],
			Date:     "2026-01-15",
			Featured: i%3 == 0,
			Excerpt:  "This is a brief excerpt of the blog post content that gives readers a preview.",
			Tags:     []string{"go", "web", "templates"},
		}
	}
	return posts
}

func NewComplexMap() map[string]any {
	posts := newBlogPosts()
	// Convert to []map[string]any for Grove/Pongo2.
	pmaps := make([]map[string]any, len(posts))
	for i, p := range posts {
		pmaps[i] = map[string]any{
			"title":    p.Title,
			"author":   p.Author,
			"date":     p.Date,
			"featured": p.Featured,
			"excerpt":  p.Excerpt,
			"tags":     p.Tags,
		}
	}
	return map[string]any{"posts": pmaps}
}

func NewComplexStruct() ComplexData {
	return ComplexData{Posts: newBlogPosts()}
}
