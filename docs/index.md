# Grove Documentation

Grove is a bytecode-compiled template engine for Go with an HTML-centric syntax. Templates use `{% %}` for server operations (control flow, assignment, composition) and `<PascalCase>` elements for component invocations. The engine is safe for concurrent use — compiled bytecode is immutable and shared across goroutines, and VM instances are pooled.

## Contents

| Page | Description |
|------|-------------|
| [Getting Started](getting-started.md) | Install Grove, configure an engine, render your first template |
| [Template Syntax](template-syntax.md) | Expressions, operators, control flow (`{% #if %}`, `{% #each %}`), assignment, literals |
| [Components](components.md) | File-per-component, `{% import %}`, slots, fills, scoped slots, dynamic components |
| [Filters](filters.md) | All 42 built-in filters — string, collection, numeric, HTML, type conversion |
| [Web Primitives](web-primitives.md) | `{% asset %}`, `{% meta %}`, `{% #hoist %}`, `{% #verbatim %}` and `RenderResult` integration |
| [Asset Pipeline](asset-pipeline.md) | `pkg/grove/assets` — Builder, Manifest, `AssetResolver`, minify sub-package, watch mode, HTTP handler |
| [API Reference](api-reference.md) | Go types, methods, options, stores, custom filters, error types |
| [Examples](examples.md) | Walkthroughs of the blog, store, docs, and email example apps |
