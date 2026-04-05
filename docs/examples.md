# Examples

## Blog Application

The `examples/blog/` directory contains a complete web application demonstrating Grove's features. It's a blog with posts, tags, components, template inheritance, and asset collection.

### Project Structure

```
examples/blog/
  main.go                           # Go web app (chi router)
  templates/
    base.grov                       # Root layout — nav, main, footer, asset placeholders
    index.grov                      # Homepage — extends base, lists posts
    post.grov                       # Post page — extends base, shows single post
    post-list.grov                  # Post list partial
    author.grov                     # Author page
    tag-list.grov                   # Tag list partial
    composites/
      card/card.grov               # Post card — props: title, summary, href, date; slot: tags
      nav/nav.grov                 # Navigation bar — props: site_name
      author-card/author-card.grov # Author card component
      breadcrumbs/breadcrumbs.grov # Breadcrumb navigation
    primitives/
      footer/footer.grov           # Footer — props: year
      tag-badge/tag-badge.grov     # Color tag badge — props: label, color
      button/button.grov           # Button link — props: label, href, variant
    pages/
```

### The Go Application

`main.go` sets up a Grove engine with a filesystem store and global variables:

```go
store := grove.NewFileSystemStore(templateDir)
eng := grove.New(grove.WithStore(store))
eng.SetGlobal("site_name", "Blog")
eng.SetGlobal("current_year", "2026")
```

Posts are Go structs implementing `Resolvable` to expose fields to templates:

```go
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
    // ... other fields
    }
    return nil, false
}
```

Handlers render templates and assemble the response by replacing placeholder comments with collected assets and meta:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    result, _ := eng.Render(r.Context(), "index.grov", grove.Data{
        "posts": postsAny,
    })

    body := result.Body
    body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)
    body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)
    // ... meta tags, hoisted content
    w.Write([]byte(body))
}
```

### Base Layout

`base.grov` defines the HTML skeleton with blocks and asset placeholders:

```jinja2
{% asset "/static/style.css" type="stylesheet" priority=10 %}
<!DOCTYPE html>
<html lang="en">
<head>
  <title>{% block title %}Grove Blog{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  {% component "composites/nav" site_name=site_name %}{% endcomponent %}
  <main class="container">{% block content %}{% endblock %}</main>
  {% component "primitives/footer" year=current_year %}{% endcomponent %}
  <!-- FOOT_ASSETS -->
</body>
</html>
```

Every page inherits this layout. The base template declares a global stylesheet asset, uses components for nav and footer, and provides placeholder comments that the Go layer replaces.

### Page Templates

`index.grov` extends the base and iterates over posts using the card component:

```jinja2
{% extends "base.grov" %}
{% block title %}Home — Grove Blog{% endblock %}

{% block content %}
{% meta name="description" content="A tech blog built with the Grove template engine" %}

{% for post in posts %}
  {% component "composites/card" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date %}
    {% fill "tags" %}
      {% for tag in post.tags %}
        {% component "primitives/tag-badge" label=tag.name color=tag.color %}{% endcomponent %}
      {% endfor %}
    {% endfill %}
  {% endcomponent %}
{% endfor %}
{% endblock %}
```

This demonstrates nested components (tag inside card), slot fills, expression concatenation (`"/post/" ~ post.slug`), and meta tag declaration.

### Components

**card.grov** — shows props with defaults and a named slot:

```jinja2
{% props title, summary, href="#", date="" %}
<article>
  <h2><a href="{{ href }}">{{ title }}</a></h2>
  {% if date %}<time>{{ date }}</time>{% endif %}
  <p>{{ summary | truncate(120) }}</p>
  <div>{% slot "tags" %}{% endslot %}</div>
</article>
```

**alert.grov** — shows the `let` block with conditional variable assignment:

```jinja2
{% props type="info" %}
{% let %}
  bg = "#d1ecf1"
  fg = "#0c5460"
  icon = "i"

  if type == "warning"
    bg = "#fff3cd"
    fg = "#856404"
    icon = "!"
  elif type == "error"
    bg = "#f8d7da"
    fg = "#721c24"
    icon = "x"
  end
{% endlet %}
<div style="background: {{ bg }}; color: {{ fg }}">
  <span>{{ icon }}</span>
  <div>{% slot %}{% endslot %}</div>
</div>
```

**button.grov** — shows ternary expressions:

```jinja2
{% props label, href="/", variant="primary" %}
{% if variant == "primary" %}
  {% set bg = "#e94560" %}{% set fg = "#fff" %}
{% elif variant == "outline" %}
  {% set bg = "transparent" %}{% set fg = "#e94560" %}
{% else %}
  {% set bg = "#6c757d" %}{% set fg = "#fff" %}
{% endif %}

<a href="{{ href }}" style="background: {{ bg }}; color: {{ fg }}; border-color: {{ variant != "outline" ? bg : "#e94560" }}">{{ label }}</a>
```

### Running It

```bash
cd examples/blog
go run main.go
```

Open `http://localhost:3000` to see:
- **Home page** — list of post cards with tags
- **Post pages** — individual posts with draft warnings (alert component)
- **Component library** (`/styleguide`) — showcase of all components with variations
