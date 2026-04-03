# Web Primitives

Grove templates can declare CSS/JS assets, meta tags, and hoisted content. These are collected during rendering — including across nested includes, components, and inherited templates — and returned in `RenderResult` for the application to assemble into the final HTML response.

## asset

Declare a stylesheet or script dependency:

```jinja2
{% asset "/static/style.css" type="stylesheet" %}
{% asset "/static/app.js" type="script" %}
```

With priority and HTML attributes:

```jinja2
{% asset "/static/main.css" type="stylesheet" priority=10 %}
{% asset "/static/analytics.js" type="script" defer=true async=true %}
```

**Rules:**
- `type` is required — typically `"stylesheet"` or `"script"`
- `priority` controls sort order (higher = earlier within its type group). Default: 0
- Additional attributes (`defer`, `async`, `crossorigin`, etc.) are passed through as HTML attributes
- Boolean attributes use `attr=true` — rendered as bare attributes (e.g., `defer`)
- Assets are deduplicated by `Src` — declaring the same URL twice results in one entry
- Assets declared in components and includes bubble up to the top-level `RenderResult`

`asset` requires a template store — it does not work with inline `RenderTemplate`.

## meta

Declare document metadata:

```jinja2
{% meta name="description" content="A great page" %}
{% meta property="og:title" content="My Page" %}
{% meta property="og:image" content="https://example.com/image.png" %}
```

**Rules:**
- `name` or `property` serves as the key
- `content` is the value
- Last-write-wins for duplicate keys — a `Warning` is added to `RenderResult.Warnings` on collision
- Meta tags from components and includes bubble up

## hoist

Capture rendered content and collect it into a named target instead of outputting it inline:

```jinja2
{% hoist target="head" %}
  <link rel="preload" href="/font.woff2" as="font" crossorigin>
{% endhoist %}

{% hoist target="head" %}
  <style>.hero { background: blue; }</style>
{% endhoist %}
```

**Rules:**
- `target` names the collection bucket (any string)
- Multiple hoists to the same target are concatenated in order
- Hoisted content is removed from `Body` and collected in `RenderResult.Hoisted`
- Hoisted content from components and includes bubbles up

## RenderResult

When you call `Render` or `RenderTemplate`, Grove returns a `RenderResult`:

```go
type RenderResult struct {
    Body     string              // Rendered HTML output
    Assets   []Asset             // Collected assets, deduplicated by Src
    Meta     map[string]string   // Collected meta tags (last-write-wins)
    Hoisted  map[string][]string // target → ordered fragments
    Warnings []Warning           // Non-fatal warnings (e.g., meta key collision)
}
```

### Helper methods

**`HeadHTML()`** — returns `<link rel="stylesheet">` tags for all stylesheet assets, sorted by descending priority:

```go
result.HeadHTML()
// <link rel="stylesheet" href="/static/main.css">
// <link rel="stylesheet" href="/static/theme.css">
```

**`FootHTML()`** — returns `<script>` tags for all script assets, sorted by descending priority:

```go
result.FootHTML()
// <script src="/static/app.js" defer></script>
```

**`GetHoisted(target)`** — returns concatenated content for a hoist target:

```go
result.GetHoisted("head")
// <link rel="preload" href="/font.woff2" as="font" crossorigin>
// <style>.hero { background: blue; }</style>
```

## Integration Pattern

A typical web application renders a template and then injects collected assets and meta into the response. Here's the complete pattern:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    result, err := eng.Render(r.Context(), "page.html", grove.Data{
        "title": "My Page",
    })
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    body := result.Body

    // Inject stylesheet assets into <head>
    body = strings.Replace(body, "<!-- HEAD_ASSETS -->", result.HeadHTML(), 1)

    // Inject script assets before </body>
    body = strings.Replace(body, "<!-- FOOT_ASSETS -->", result.FootHTML(), 1)

    // Build and inject meta tags
    var meta strings.Builder
    for name, content := range result.Meta {
        if strings.HasPrefix(name, "og:") {
            meta.WriteString(fmt.Sprintf(`  <meta property="%s" content="%s">`+"\n", name, content))
        } else {
            meta.WriteString(fmt.Sprintf(`  <meta name="%s" content="%s">`+"\n", name, content))
        }
    }
    body = strings.Replace(body, "<!-- HEAD_META -->", meta.String(), 1)

    // Inject hoisted content
    body = strings.Replace(body, "<!-- HEAD_HOISTED -->", result.GetHoisted("head"), 1)

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, body)
}
```

The base template uses placeholder comments that get replaced:

```jinja2
<head>
  <title>{% block title %}My Site{% endblock %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  {% block content %}{% endblock %}
  <!-- FOOT_ASSETS -->
</body>
```

This pattern keeps template authors and application developers in their own domains — templates declare what they need, and the Go layer assembles it.

## Auto-Escaping

All `{{ }}` output is HTML-escaped by default. This prevents XSS from untrusted data:

```jinja2
{% set input = "<script>alert('xss')</script>" %}
{{ input }}
{# Output: &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; #}
```

To output trusted HTML, use the `safe` filter:

```jinja2
{{ trusted_html | safe }}
```

Or use `{% raw %}` blocks to output template syntax literally (no parsing or escaping):

```jinja2
{% raw %}{{ not parsed }}{% endraw %}
```

From Go code, use `SafeHTMLValue` to mark a value as pre-trusted:

```go
data := grove.Data{
    "content": grove.SafeHTMLValue("<p>Trusted HTML</p>"),
}
```

**Only mark content as safe when you trust the source.** Auto-escaping exists to protect against XSS — bypassing it with untrusted data creates vulnerabilities.
