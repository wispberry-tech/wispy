<p align="center">
  <img src="branding/grove-full-logo@2x.png" alt="Wispy Grove" width="400">
</p>

<p align="center">
  A bytecode-compiled template engine for Go with an HTML-centric syntax, components, and web primitives.
</p>

## Install

```bash
go get github.com/wispberry-tech/grove
```

## Quick Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/wispberry-tech/grove/pkg/grove"
)

func main() {
	eng := grove.New()
	result, err := eng.RenderTemplate(
		context.Background(),
		`Hello, {% name | title %}!`,
		grove.Data{"name": "world"},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Body) // Hello, World!
}
```

## Template Language

Grove templates use a single `{% %}` delimiter for server operations (control flow, assignment, composition) and PascalCase elements for components:

```html
{% import Base from "layouts/base" %}
{% import Card from "composites/card" %}

<Base>
  {% #fill "content" %}
    {% asset "composites/card/card.css" type="stylesheet" %}
    {% meta name="description" content="Latest posts" %}

    <h1>{% title | upper %}</h1>

    {% #each posts as post %}
      <Card title={post.title} date={post.date}>
        {% #fill "tags" %}
          {% #each post.tags as tag %}
            <span class="{% tag.draft ? "muted" : "active" %}">{% tag.name %}</span>
          {% /each %}
        {% /fill %}
      </Card>
    {% :empty %}
      <p>No posts yet.</p>
    {% /each %}
  {% /fill %}
</Base>
```

### Components

Components are `.grov` files, imported with `{% import %}`, and composed using slots and fills. Each component co-locates its own CSS and JS assets alongside the `.grov` file — assets declared inside a component automatically bubble up through `RenderResult` so the host page can collect them.

**File structure** — components live in named directories with their assets alongside them:

```
templates/
├── base.grov                          # layout component
├── index.grov                         # page (imports components)
├── composites/
│   └── card/
│       ├── card.grov                  # component template
│       └── card.css                   # co-located styles
└── primitives/
    └── button/
        ├── button.grov                # component template
        ├── button.css                 # co-located styles
        └── button.js                  # co-located script
```

**Defining a component** (`primitives/button/button.grov`) — the file body IS the component:

```html
{% asset "primitives/button/button.css" type="stylesheet" %}
{% asset "primitives/button/button.js" type="script" %}

{% #if type == "link" %}
  <a href="{% href %}" class="btn btn-{% variant %}" data-button>{% label %}</a>
{% :else %}
  <button type="{% type %}" class="btn btn-{% variant %}" data-button>{% label %}</button>
{% /if %}
```

Any attribute passed when invoking the component becomes a template variable. The `{% asset %}` tags declare that this component needs its own CSS and JS — no matter how deeply nested the component is, those assets bubble up to the final `RenderResult`.

**A component with named slots** (`composites/card/card.grov`):

```html
{% asset "composites/card/card.css" type="stylesheet" %}
<article class="card">
  <h2 class="card-title"><a href="{% href %}">{% title %}</a></h2>
  <p class="card-summary">{% summary | truncate(150) %}</p>
  <div class="card-tags">
    {% #slot "tags" %}{% /slot %}
  </div>
</article>
```

**Importing and using components** from a page template:

```html
{% import Base from "base" %}
{% import Card from "composites/card" %}
{% import TagBadge from "primitives/tag-badge" %}

<Base>
  {% #fill "content" %}
    <h1>{% title %}</h1>

    {% #each posts as post %}
      <Card title={post.title} summary={post.summary} href={"/post/" ~ post.slug}>
        {% #fill "tags" %}
          {% #each post.tags as tag %}
            <TagBadge label={tag.name} color={tag.color} />
          {% /each %}
        {% /fill %}
      </Card>
    {% :empty %}
      <p>No posts yet.</p>
    {% /each %}
  {% /fill %}
</Base>
```

Props are passed with `{expr}` syntax. Fills inject content into named slots — and fills always see the **caller's** scope, not the component's. Every `{% asset %}` declared by `Card`, `TagBadge`, `Base`, or any other component in the tree is deduplicated and available on the `RenderResult` via `result.HeadHTML()` (stylesheets) and `result.FootHTML()` (scripts).

### Asset Pipeline (optional)

The `{% asset %}` tags above use *logical names* — relative paths that the
engine rewrites through a pluggable resolver. Wire up `pkg/grove/assets` to
get content-hashed, minified URLs with one manifest-driven mapping:

```go
import (
    "github.com/wispberry-tech/grove/pkg/grove"
    "github.com/wispberry-tech/grove/pkg/grove/assets"
    "github.com/wispberry-tech/grove/pkg/grove/assets/minify"
)

builder := assets.NewWithDefaults(assets.Config{
    SourceDir:      "templates",
    OutputDir:      "dist",
    URLPrefix:      "/dist",
    CSSTransformer: minify.New(),
    JSTransformer:  minify.New(),
    ManifestPath:   "dist/manifest.json",
})
manifest, _ := builder.Build()

eng := grove.New(
    grove.WithStore(grove.NewFileSystemStore("templates")),
    grove.WithAssetResolver(manifest.Resolve),
)

// Serve hashed files with Cache-Control: immutable.
pattern, handler := builder.Route()
mux.Handle(pattern+"*", handler)
```

With the resolver wired, `{% asset "primitives/button/button.css" %}` renders
as `/dist/primitives/button/button.<hash>.css`. Drop the resolver and the
same tag passes through unchanged — the pipeline is fully opt-in and adds
no overhead when absent. See [Asset Pipeline](docs/asset-pipeline.md) for
watch mode, pruning, custom transformers, and the HTTP handler API.

### Syntax at a Glance

| Category | Syntax |
|----------|--------|
| **Output** | `{% expr %}`, `{% value \| filter %}`, `{% cond ? a : b %}` |
| **Control flow** | `{% #if %}`/`{% :else if %}`/`{% :else %}`/`{% /if %}`, `{% #each %}`/`{% :empty %}`/`{% /each %}` |
| **Assignment** | `{% set %}`, `{% #let %}`/`{% /let %}` (multi-variable), `{% #capture %}`/`{% /capture %}` |
| **Components** | `<Component>`, `{% import %}`, `{% slot %}`, `{% #fill %}`/`{% /fill %}` |
| **Web primitives** | `{% asset %}`, `{% meta %}`, `{% #hoist %}`/`{% /hoist %}` |
| **Data literals** | `[1, 2, 3]`, `{key: "value"}` |
| **Other** | `{# comment #}`, `{% #verbatim %}`/`{% /verbatim %}`, whitespace control (`{%- -%}`) |

### Built-in Filters

42 filters across 5 categories:

| Category | Filters |
|----------|---------|
| **String** | `upper` `lower` `title` `capitalize` `trim` `lstrip` `rstrip` `replace` `truncate` `center` `ljust` `rjust` `split` `wordcount` |
| **Collection** | `length` `first` `last` `join` `sort` `reverse` `unique` `min` `max` `sum` `map` `batch` `flatten` `keys` `values` |
| **Numeric** | `abs` `round` `ceil` `floor` `int` `float` |
| **Logic / Type** | `default` `string` `bool` |
| **HTML** | `escape` `safe` `striptags` `nl2br` |

## Features

| Feature | Description |
|---------|-------------|
| **Bytecode compilation** | Templates compile to bytecode and run on a stack-based VM. Compiled bytecode is immutable and shared across goroutines. |
| **Components** | File-per-component: each `.grov` file is a component. Invoke with `<ComponentName>`, import with `{% import %}`. Slots/fills for composition. Scoped slots pass data back to callers. |
| **Layouts** | Layouts are components with named slots — no special inheritance system. Compose layouts to any depth. |
| **Imports** | `{% import %}` brings components into scope. |
| **Web primitives** | `{% asset %}`, `{% meta %}`, and `{% #hoist %}` collect resources during rendering. Components declare their own CSS/JS assets, which bubble up through composition. `RenderResult` returns them for assembly into the final HTML response. |
| **Asset pipeline** | Optional `pkg/grove/assets` package builds a content-hashed, minified `dist/` from colocated CSS/JS, exposes a `Manifest`, and plugs into the engine via `WithAssetResolver`. Includes an HTTP handler with immutable caching, watch mode, and an optional `minify` sub-package. |
| **Auto-escaping** | HTML output is escaped by default. `safe` filter and `{% #verbatim %}` blocks bypass it for trusted content. |
| **Sandboxing** | Restrict allowed tags, filters, and loop iterations per engine. |

## Documentation

Full documentation is in the [`docs/`](docs/index.md) directory:

- [Getting Started](docs/getting-started.md) — install, configure, render your first template
- [Template Syntax](docs/template-syntax.md) — variables, expressions, control flow, assignment
- [Components](docs/components.md) — definitions, imports, props, slots, fills, layouts
- [Filters](docs/filters.md) — all 42 built-in filters
- [Web Primitives](docs/web-primitives.md) — ImportAsset, SetMeta, Hoist, RenderResult
- [Asset Pipeline](docs/asset-pipeline.md) — Builder, Manifest, resolver, minify, watch mode
- [API Reference](docs/api-reference.md) — Go types, methods, and configuration
- [Examples](docs/examples.md) — blog/store/docs/email walkthroughs
