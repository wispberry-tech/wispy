# Web Primitives

Grove templates can declare CSS/JS assets, meta tags, and hoisted content. These are collected during rendering — including across nested imports, components, and layout composition — and returned in `RenderResult` for the application to assemble into the final HTML response.

## asset

Declare a stylesheet or script dependency:

```html
{% asset "/static/style.css" type="stylesheet" %}
{% asset "/static/app.js" type="script" %}
```

With priority and HTML attributes:

```html
{% asset "/static/main.css" type="stylesheet" priority="10" %}
{% asset "/static/analytics.js" type="script" defer async %}
```

**Rules:**
- `src` and `type` are required — type is typically `"stylesheet"` or `"script"`
- `priority` controls sort order (higher = earlier within its type group). Default: 0
- Additional attributes (`defer`, `async`, `crossorigin`, etc.) are passed through as HTML attributes
- Bare attributes (like `defer`) are treated as boolean attributes
- Assets are deduplicated by `src` — declaring the same URL twice results in one entry
- Assets declared in components bubble up to the top-level `RenderResult`

### Per-Component Assets

Components declare their own CSS and JS dependencies. When a page renders components that use other components, all assets bubble up and are deduplicated in `RenderResult`:

```html
{# composites/nav/nav.grov #}
{% asset "composites/nav/nav.css" type="stylesheet" %}
{% asset "composites/nav/nav.js" type="script" %}
<nav class="nav">...</nav>
```

These are **logical names** — paths relative to your template root. Global
styles that live outside the template tree can still use URL-style paths; they
pass through unchanged when no manifest entry matches:

```html
{# base.grov #}
{% asset "/static/base.css" type="stylesheet" priority=10 %}
...
```

Component assets use the default priority (0), which means `HeadHTML()` outputs `base.css` before component stylesheets — preserving the correct cascade order.

### Asset resolution

The engine resolves every `{% asset %}` `src` through a pluggable
`AssetResolver` function before storing the asset on `RenderResult`:

```go
type AssetResolver func(logicalName string) (string, bool)
```

With no resolver configured (the default), the `src` is stored verbatim.
Configure one with `grove.WithAssetResolver(r)` at engine construction or
`engine.SetAssetResolver(r)` at runtime (atomic; safe against concurrent
renders).

The typical resolver comes from the optional asset pipeline in
`pkg/grove/assets`, which builds content-hashed output and produces a
`Manifest` whose `.Resolve` method satisfies `AssetResolver`:

```go
builder := assets.NewWithDefaults(assets.Config{
    SourceDir: "templates",
    OutputDir: "dist",
    URLPrefix: "/dist",
})
manifest, _ := builder.Build()

eng := grove.New(
    grove.WithStore(grove.NewFileSystemStore("templates")),
    grove.WithAssetResolver(manifest.Resolve),
)

pattern, handler := builder.Route()
mux.Handle(pattern+"*", handler)
```

With the pipeline wired, `composites/nav/nav.css` in the template above is
rewritten to `/dist/composites/nav/nav.<hash>.css` at render time. See the
[Asset Pipeline](asset-pipeline.md) page for watch mode, pruning, custom
transformers, and the HTTP handler API.

If you aren't using the pipeline, serve your static files however you like —
Grove's `{% asset %}` just records strings; making them load is your
application's job.

## meta

Declare document metadata:

```html
{% meta name="description" content="A great page" %}
{% meta property="og:title" content="My Page" %}
{% meta property="og:image" content="https://example.com/image.png" %}
```

**Rules:**
- `name` or `property` serves as the key
- `content` is the value
- Last-write-wins for duplicate keys — a `Warning` is added to `RenderResult.Warnings` on collision
- Meta tags from components bubble up

## hoist

Capture rendered content and collect it into a named target instead of outputting it inline:

```html
{% #hoist "head" %}
  <link rel="preload" href="/font.woff2" as="font" crossorigin>
{% /hoist %}

{% #hoist "head" %}
  <style>.hero { background: blue; }</style>
{% /hoist %}
```

**Rules:**
- `target` names the collection bucket (any string)
- Multiple hoists to the same target are concatenated in order
- Hoisted content is removed from `Body` and collected in `RenderResult.Hoisted`
- Hoisted content from components bubbles up

## verbatim

Output Grove syntax literally without parsing:

```html
{% #verbatim %}
  {% this is not parsed %}
  {# neither is this #}
{% /verbatim %}
```

Everything between `{% #verbatim %}` and `{% /verbatim %}` is emitted as raw text.

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

A typical web application renders a template and then injects collected assets and meta into the response:

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

The base layout component uses placeholder comments that get replaced:

```html
{# base.grov #}
{% asset "/static/base.css" type="stylesheet" priority=10 %}
<head>
  <title>{% #slot "title" %}My Site{% /slot %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  {% slot "content" %}
  <!-- FOOT_ASSETS -->
</body>
```

This pattern keeps template authors and application developers in their own domains — templates declare what they need, and the Go layer assembles it. Global styles load via the base layout, while component-specific styles are co-located with each component and collected automatically during rendering.

## Auto-Escaping

All `{% %}` output is HTML-escaped by default. This prevents XSS from untrusted data:

```html
{% set input = "<script>alert('xss')</script>" %}
{% input %}
{# Output: &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; #}
```

To output trusted HTML, use the `safe` filter:

```html
{% trusted_html | safe %}
```

Or use `{% #verbatim %}` blocks to output template syntax literally (no parsing or escaping):

```html
{% #verbatim %}{% not parsed %}{% /verbatim %}
```

From Go code, use `SafeHTMLValue` to mark a value as pre-trusted:

```go
data := grove.Data{
    "content": grove.SafeHTMLValue("<p>Trusted HTML</p>"),
}
```

**Only mark content as safe when you trust the source.** Auto-escaping exists to protect against XSS — bypassing it with untrusted data creates vulnerabilities.
