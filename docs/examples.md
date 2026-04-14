# Examples

The `examples/` directory contains four complete apps. See
[`examples/README.md`](../examples/README.md) for the high-level tour and
per-example README files for design notes.

| Example | Port | Uses asset pipeline | Notable features |
|---------|------|---------------------|------------------|
| [blog](../examples/blog)   | 3000 | ✅ | Component composition, slots, `safe`-filtered HTML body |
| [store](../examples/store) | 3001 | ✅ | Custom `currency` filter, filtering/sort via query params |
| [docs](../examples/docs)   | 3002 | ✅ | Sandbox config, sidebar + breadcrumbs, macros |
| [email](../examples/email) | 3003 | ❌ (inline styles) | `{% #hoist %}` preheaders, MSO-safe HTML |

```bash
go run ./examples/blog
go run ./examples/store
go run ./examples/docs
go run ./examples/email
```

---

## Blog (reference)

`examples/blog/` is the canonical Grove integration. It's the smallest example
with every feature wired up.

### Project structure

```
examples/blog/
├── main.go
├── data/                           JSON fixtures: authors, tags, posts
├── dist/                           Generated: hashed CSS/JS + manifest.json
├── static/
│   ├── base.css                    Hand-managed global styles
│   └── tokens.css
└── templates/
    ├── base.grov                   Root layout with slots
    ├── index.grov                  Homepage
    ├── post.grov                   Single post page
    ├── composites/
    │   ├── card/{card.grov,card.css}
    │   ├── nav/{nav.grov,nav.css,nav.js}
    │   ├── author-card/...
    │   └── breadcrumbs/...
    └── primitives/
        ├── footer/{footer.grov,footer.css}
        ├── tag-badge/{tag-badge.grov,tag-badge.css}
        └── button/{button.grov,button.css,button.js}
```

Each component co-locates its CSS / JS. The builder (see below) walks
`templates/`, hashes + minifies every `.css` / `.js`, and writes them to
`dist/` with a manifest.

### The Go application

```go
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
    log.Fatalf("asset build failed: %v", err)
}

store := grove.NewFileSystemStore(templateDir)
eng := grove.New(
    grove.WithStore(store),
    grove.WithAssetResolver(manifest.Resolve),
)

// HTTP routing
distPattern, distHandler := builder.Route()
r.Handle(distPattern+"*", distHandler)
```

`WithAssetResolver(manifest.Resolve)` means every logical
`{% asset "composites/nav/nav.css" %}` in the templates is rewritten to
`/dist/composites/nav/nav.<hash>.css` at render time. `builder.Route()`
serves those files with `Cache-Control: immutable`. See
[Asset Pipeline](asset-pipeline.md) for the full API.

### Templates

`base.grov` keeps the hand-managed global as a URL-style (passthrough)
asset and composes nav/footer components:

```html
{% asset "/static/base.css" type="stylesheet" priority=10 %}
{% import Nav from "composites/nav" %}
{% import Footer from "primitives/footer" %}
<!DOCTYPE html>
<html lang="en">
<head>
  <title>{% #slot "title" %}Grove Blog{% /slot %}</title>
  <!-- HEAD_ASSETS -->
  <!-- HEAD_META -->
  <!-- HEAD_HOISTED -->
</head>
<body>
  <Nav site_name={site_name} />
  <main class="container">{% slot "content" %}</main>
  <Footer year={current_year} />
  <!-- FOOT_ASSETS -->
</body>
</html>
```

Component assets look like:

```html
{# composites/nav/nav.grov #}
{% asset "composites/nav/nav.css" type="stylesheet" %}
{% asset "composites/nav/nav.js"  type="script" %}
<nav class="nav">...</nav>
```

Placeholder comments in the base layout are replaced by the Go response
assembler using `result.HeadHTML()`, `result.FootHTML()`, `result.Meta`,
and `result.GetHoisted("head")` — see `main.go:writeResult`.

---

## Store

`examples/store/` adds:

- A custom filter registered from Go:
  ```go
  eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, _ []grove.Value) (grove.Value, error) {
      cents, _ := v.ToInt64()
      return grove.StringValue(fmt.Sprintf("$%d.%02d", cents/100, cents%100)), nil
  }))
  ```
  Templates then use `{% product.price | currency %}`.
- Cookie-based cart state (`cartHandler`, `cartAddHandler`).
- Query-string filtering + sorting in `productsHandler`.
- Same asset pipeline wiring as blog.

---

## Docs

`examples/docs/` demonstrates:

- `grove.WithSandbox(...)` restricting allowed tags / filters and capping
  loop iterations. Any template that oversteps errors at render time.
- Deep component nesting (`Base` → `DocsLayout` → page) with sidebar,
  breadcrumbs, and prev/next partials.
- Colocated macros (`macros/admonitions.grov`, `macros/code-example.grov`)
  with their own CSS — picked up by the asset builder just like
  composites/primitives.

The sandbox config must include `"asset"` in `AllowedTags` for the
pipeline to work; the example shows the full whitelist.

---

## Email

`examples/email/` is the one example that **does not** use the asset
pipeline. Email clients (especially Outlook) require inline styles, so
component `{% asset %}` tags would be useless. Instead the example leans
on `{% #hoist "preheader" %}`, captured blocks, and table-based layouts
with MSO conditional comments. See its README for the full feature list.

---

## Running for development

For template hot-reload, use `entr` or similar:

```bash
ls examples/blog/templates/**/*.grov | entr -r go run ./examples/blog
```

For asset hot-rebuild, swap `builder.Build()` for
`builder.Watch(ctx, handlers)` — it polls at 500 ms, debounces 100 ms,
and calls `engine.SetAssetResolver` on each rebuild so new hashes take
effect immediately. See [Asset Pipeline → Watch mode](asset-pipeline.md#watch-mode-development).
