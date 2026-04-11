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
    {% asset "/css/composites/card/card.css" type="stylesheet" %}
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
| **Components** | `<Component>` definitions with props, `{% slot %}`, and `{% #fill %}`. Fills see the caller's scope, not the component's. Scoped slots pass data back to callers. |
| **Layouts** | Layouts are components with named slots — no special inheritance system. Compose layouts to any depth. |
| **Imports** | `{% import %}` brings components into scope. |
| **Web primitives** | `{% asset %}`, `{% meta %}`, and `{% #hoist %}` collect resources during rendering. Components declare their own CSS/JS assets, which bubble up through composition. `RenderResult` returns them for assembly into the final HTML response. |
| **Auto-escaping** | HTML output is escaped by default. `safe` filter and `{% #verbatim %}` blocks bypass it for trusted content. |
| **Sandboxing** | Restrict allowed tags, filters, and loop iterations per engine. |

## Documentation

Full documentation is in the [`docs/`](docs/index.md) directory:

- [Getting Started](docs/getting-started.md) — install, configure, render your first template
- [Template Syntax](docs/template-syntax.md) — variables, expressions, control flow, assignment
- [Components](docs/components.md) — definitions, imports, props, slots, fills, layouts
- [Filters](docs/filters.md) — all 42 built-in filters
- [Web Primitives](docs/web-primitives.md) — ImportAsset, SetMeta, Hoist, RenderResult
- [API Reference](docs/api-reference.md) — Go types, methods, and configuration
- [Examples](docs/examples.md) — blog app walkthrough
