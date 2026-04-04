<p align="center">
  <img src="branding/grove-full-logo.png" alt="Wispy Grove" width="400">
</p>

<p align="center">
  A bytecode-compiled template engine for Go with components, inheritance, and web primitives.
</p>

## Install

```bash
go get grove
```

## Quick Example

```go
package main

import (
	"context"
	"fmt"
	"grove/pkg/grove"
)

func main() {
	eng := grove.New()
	result, err := eng.RenderTemplate(
		context.Background(),
		`Hello, {{ name | title }}!`,
		grove.Data{"name": "world"},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Body) // Hello, World!
}
```

## Template Language

Grove templates mix HTML with expressions, control flow, and components:

```jinja2
{% extends "base.html" %}

{% block content %}
{% asset "/static/page.css" type="stylesheet" %}
{% meta name="description" content="Latest posts" %}

<h1>{{ title | upper }}</h1>

{% for post in posts %}
  {% component "components/card.html" title=post.title date=post.date %}
    {% fill "tags" %}
      {% for tag in post.tags %}
        <span class="{{ tag.draft ? "muted" : "active" }}">{{ tag.name }}</span>
      {% endfor %}
    {% endfill %}
  {% endcomponent %}
{% empty %}
  <p>No posts yet.</p>
{% endfor %}
{% endblock %}
```

### Syntax at a Glance

| Category | Tags |
|----------|------|
| **Output** | `{{ expr }}`, `{{ value \| filter }}`, `{{ cond ? a : b }}` |
| **Control flow** | `if` / `elif` / `else`, `for` / `empty`, `range` |
| **Assignment** | `set`, `let` (multi-variable with conditionals), `capture` |
| **Composition** | `include`, `render`, `macro` / `call` / `import` |
| **Inheritance** | `extends`, `block`, `super()` |
| **Components** | `component`, `props`, `slot`, `fill` |
| **Web primitives** | `asset`, `meta`, `hoist` |
| **Data literals** | `[1, 2, 3]`, `{key: "value"}` |
| **Other** | `{# comment #}`, `{% raw %}`, whitespace control (`{%- -%}`) |

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
| **Template inheritance** | `extends`, `block`, and `super()` for layered layouts with unlimited depth. |
| **Components** | Reusable templates with `props`, `slot`, and `fill`. Fills see the caller's scope, not the component's. |
| **Macros** | `macro`, `call`, `caller()`, and `import` for reusable template functions. |
| **Web primitives** | `asset`, `meta`, and `hoist` tags collect resources during rendering. `RenderResult` returns them for assembly into the final HTML response. |
| **Auto-escaping** | HTML output is escaped by default. `safe` filter and `raw` blocks bypass it for trusted content. |
| **Sandboxing** | Restrict allowed tags, filters, and loop iterations per engine. |

## Documentation

Full documentation is in the [`docs/`](docs/index.md) directory:

- [Getting Started](docs/getting-started.md) — install, configure, render your first template
- [Template Syntax](docs/template-syntax.md) — variables, expressions, control flow, assignment
- [Template Inheritance](docs/template-inheritance.md) — extends, block, super()
- [Components](docs/components.md) — props, slots, fills
- [Macros & Includes](docs/macros-and-includes.md) — macro, include, render, import
- [Filters](docs/filters.md) — all 42 built-in filters
- [Web Primitives](docs/web-primitives.md) — asset, meta, hoist, RenderResult
- [API Reference](docs/api-reference.md) — Go types, methods, and configuration
- [Examples](docs/examples.md) — blog app walkthrough
