# Grove Documentation

**Date:** 2026-04-03
**Status:** Approved
**Scope:** Full project documentation — README, syntax reference, API reference, filter catalog, examples

## Motivation

Grove has no user-facing documentation. There is no README, no syntax guide, no API reference. The only written material is internal specs and CLAUDE.md. Anyone discovering the project on GitHub has nothing to read. This spec defines a complete documentation set that serves both template authors (writing `.grov` files) and Go developers (integrating Grove into applications).

## Delivery

All documentation is Markdown files viewable directly on GitHub. No static site generator, no build step.

## Files

### `README.md` (project root)

The GitHub landing page. Brief and punchy — gets someone from zero to "I understand what this is" in 30 seconds.

**Sections:**

1. **One-line tagline** — "A bytecode-compiled template engine for Go with components, inheritance, and web primitives."
2. **Install** — `go get grove`
3. **Quick example** — Minimal Go program: create engine, render inline template with a variable and filter, print output. ~10 lines.
4. **Feature highlights** — Bullet list:
   - Bytecode compilation (fast, goroutine-safe)
   - Template inheritance with `extends`, `block`, `super()`
   - Components with props, slots, and fills
   - 40+ built-in filters
   - Web primitives: asset collection, meta tags, content hoisting
   - Auto-escaping (on by default)
   - Sandboxing (tag/filter whitelists, loop limits)
5. **Documentation link** — Points to `docs/`

No badges, no lengthy explanations, no comparison tables.

### `docs/index.md`

Documentation home page. Short overview paragraph, then a linked table of contents to all doc pages with one-line descriptions. Navigation hub only — no substantive content duplicated here.

### `docs/getting-started.md`

Step-by-step guide for Go developers to integrate Grove.

**Sections:**

1. **Installation** — `go get grove`, import path `grove/pkg/grove`
2. **Inline templates** — Create engine with `grove.New()`, call `RenderTemplate()` with a string template and `grove.Data{}`. Show output.
3. **File-based templates** — Set up `FileSystemStore`, pass to engine with `WithStore()`, call `Render()` by template name.
4. **In-memory templates** — Create `MemoryStore`, add templates with `Set()`, render by name.
5. **Passing data** — `grove.Data` map, nested maps, slices, Go structs via `Resolvable` interface. Show a complete `Resolvable` example (the Tag struct from the blog example is a good model).
6. **Global variables** — `SetGlobal()` for data available across all renders.
7. **Engine options** — Brief table of all options: `WithStore`, `WithStrictVariables`, `WithCacheSize`, `WithSandbox`.
8. **Error handling** — `ParseError` vs `RuntimeError`, what fields they expose.

Each section has a runnable code example.

### `docs/template-syntax.md`

Complete syntax reference for template authors. This is the page people will bookmark.

**Sections:**

1. **Delimiters** — `{{ }}` for output, `{% %}` for tags, `{# #}` for comments. Whitespace control with `-` trim markers: `{{- -}}`, `{%- -%}`.
2. **Variables** — `{{ name }}`, `{{ user.name }}`, `{{ items[0] }}`. Dot access, bracket access, nested paths.
3. **Expressions** — Full operator table (highest to lowest precedence):
   - `.`, `[]`, `()` — access, index, call
   - `|` — filter pipe
   - `not` — negation
   - `*`, `/`, `%` — multiplicative
   - `+`, `-`, `~` — additive, string concatenation
   - `<`, `<=`, `>`, `>=`, `==`, `!=` — comparison
   - `and` — logical and
   - `or` — logical or
   - `? :` — ternary
4. **Ternary expressions** — `condition ? truthy : falsy`. Right-associative nesting. Filters bind tighter than `?`.
5. **List literals** — `[1, 2, 3]`, nested lists, trailing comma allowed. Access via `list[0]`.
6. **Map literals** — `{ key: "value" }`, unquoted identifier keys only, nested maps, trailing comma allowed. Access via `map.key` or `map["key"]`. Insertion order preserved.
7. **Filters** — Pipe syntax `{{ value | filter }}`, chaining `{{ value | filter1 | filter2 }}`, arguments `{{ value | filter(arg1, arg2) }}`. Link to filters.md for the full catalog.
8. **if / elif / else** — Conditional blocks. Truthy/falsy rules (empty string, 0, nil, empty list/map are falsy).
9. **for loops** — `{% for item in list %}`, `{% for key, val in map %}`, `{% empty %}` fallback. Loop variables: `loop.index`, `loop.index0`, `loop.first`, `loop.last`, `loop.length`, `loop.depth`, `loop.parent`.
10. **range** — `range(stop)`, `range(start, stop)`, `range(start, stop, step)`.
11. **set** — `{% set name = expression %}` for single variable assignment.
12. **let** — `{% let %}...{% endlet %}` for multi-variable assignment. Bare `name = expression` per line. `if/elif/else/end` conditionals inside (no delimiters). Variables written to outer scope. Multi-line expressions supported (map literals spanning lines). Full example showing conditional variable assignment.
13. **capture** — `{% capture name %}...{% endcapture %}`. Renders body into a variable.
14. **raw** — `{% raw %}...{% endraw %}`. Content inside is not parsed.
15. **Comments** — `{# comment #}`. Not rendered in output.

### `docs/template-inheritance.md`

Template inheritance system.

**Sections:**

1. **Overview** — Base templates define structure with blocks; child templates extend and override blocks.
2. **extends** — `{% extends "base.grov" %}`. Must be the first tag. Child template body is discarded except for block overrides.
3. **block** — `{% block name %}default content{% endblock %}`. Defines an override point in the base template.
4. **Overriding blocks** — Child template provides `{% block name %}new content{% endblock %}` to replace parent content.
5. **super()** — `{{ super() }}` inside a block includes the parent's version. Show a layered example: grandparent → parent → child.
6. **Multi-level inheritance** — Demonstrate three-level chain with blocks at each level.

### `docs/components.md`

The component system.

**Sections:**

1. **Overview** — Components are template files with a declared interface (props) and content injection points (slots).
2. **Using a component** — `{% component "path.grov" key=value %}...{% endcomponent %}`. Space-separated key=value params.
3. **Defining props** — `{% props name, role, active=true %}`. Required vs optional (has default). Error on missing required prop.
4. **Default slot** — Content between `{% component %}` and `{% endcomponent %}` goes to the default `{% slot %}`.
5. **Named slots** — `{% slot "name" %}fallback{% endslot %}` in the component. `{% fill "name" %}content{% endfill %}` from the caller.
6. **Scope rules** — Props are available inside the component. Fills see the **caller's** scope, not the component's. This is the key design decision — explain clearly with an example.
7. **Nested components** — Components can use other components.

### `docs/macros-and-includes.md`

Composition tools.

**Sections:**

1. **include** — `{% include "template.grov" key=value %}`. Included template shares the caller's scope plus any passed params.
2. **render** — `{% render "template.grov" key=value %}`. Isolated scope — only passed params are visible.
3. **include vs render** — When to use each. `include` for partials that need page context. `render` for self-contained fragments.
4. **macro** — `{% macro name(arg, kwarg="default") %}...{% endmacro %}`. Defines a reusable template function. Macros have their own scope.
5. **Calling macros** — Positional args, keyword args. `{{ name(arg) }}` or `{% call name(arg) %}body{% endcall %}`.
6. **caller()** — `{{ caller() }}` inside a macro renders the body from `{% call %}`.
7. **import** — `{% import "macros.grov" as m %}`. Access imported macros via `{{ m.name(args) }}`.

### `docs/filters.md`

Complete filter catalog. Every built-in filter documented with signature, description, and example.

**Structure:**

Organized by category with a table of contents at the top. Each filter entry:

```
#### `filter_name`

`value | filter_name` or `value | filter_name(arg1, arg2)`

Description.

**Example:**
Input → Output
```

**Categories and filters:**

**String (14):**
- `upper`, `lower`, `title`, `capitalize` — case transforms
- `trim`, `lstrip`, `rstrip` — whitespace stripping
- `replace(old, new)` or `replace(old, new, count)` — substring replacement
- `truncate(length, suffix)` — truncate with ellipsis (defaults: 255, "...")
- `center(width, fill)`, `ljust(width, fill)`, `rjust(width, fill)` — padding (default fill: space)
- `split(sep)` — split string into list (default sep: space)
- `wordcount` — count words

**Collection (15):**
- `length` — length of string, list, or map
- `first`, `last` — first/last element
- `join(sep)` — join list to string (default sep: empty string)
- `sort` — lexicographic sort
- `reverse` — reverse order
- `unique` — remove duplicates
- `min`, `max` — numeric min/max of list
- `sum` — numeric sum of list
- `map(attr)` — extract attribute from each item in list
- `batch(size)` — group list into batches of size
- `flatten` — flatten nested lists one level
- `keys`, `values` — extract map keys/values as list

**Numeric (6):**
- `abs` — absolute value
- `round(precision)` — round to precision (default: 0)
- `ceil`, `floor` — ceiling/floor
- `int` — convert to integer
- `float` — convert to float

**Logic / Type (3):**
- `default(fallback)` — use fallback if value is falsy
- `string` — convert to string
- `bool` — convert to boolean

**HTML (3):**
- `escape` — HTML-escape (`<` → `&lt;`, etc.)
- `striptags` — strip HTML tags
- `nl2br` — convert newlines to `<br>` tags

**Special (1):**
- `safe` — mark value as trusted HTML, bypass auto-escaping

**Custom filters section** at the bottom:
- How to register with `RegisterFilter()`
- `FilterFn` signature: `func(v Value, args []Value) (Value, error)`
- `FilterFunc` wrapper with `FilterOutputsHTML()` option
- Complete example: a custom `markdown` filter

### `docs/web-primitives.md`

Grove's distinguishing feature: web-aware template primitives.

**Sections:**

1. **Overview** — Templates can declare assets, meta tags, and hoisted content. These are collected during rendering (including across nested includes/components) and returned in `RenderResult` for the application to place in the final HTML.
2. **asset** — `{% asset "path" type="stylesheet" %}`. Attributes: `type` (required), `priority` (higher = earlier), arbitrary HTML attrs (`defer`, `async`, `crossorigin`, etc.). Boolean attrs use bare key. Assets deduplicated by `Src`.
3. **meta** — `{% meta name="key" content="value" %}`. Last-write-wins for duplicate keys. Warns on collision.
4. **hoist** — `{% hoist target="name" %}content{% endhoist %}`. Hoists rendered content to a named target. Multiple hoists to same target are concatenated in order.
5. **RenderResult** — What the Go code receives:
   - `Body` — rendered HTML
   - `Assets` — collected `[]Asset`, deduplicated
   - `Meta` — collected `map[string]string`
   - `Hoisted` — collected `map[string][]string`
   - `Warnings` — `[]Warning`
6. **Helper methods** — `HeadHTML()`, `FootHTML()`, `GetHoisted(target)`. What they return and when to use them.
7. **Integration pattern** — Complete Go example showing how to assemble a full HTML response: render template, inject `HeadHTML()` into `<head>`, inject `FootHTML()` before `</body>`, inject meta tags, inject hoisted content. Based on the blog example's `writeResult` function.
8. **Auto-escaping** — On by default. `safe` filter bypasses it. `SafeHTMLValue()` from Go code bypasses it. Explain the security model briefly.

### `docs/api-reference.md`

Complete Go API reference. One page since the public surface is compact.

**Sections:**

1. **Engine** — `New(opts ...Option) *Engine`. Constructor, options, concurrency safety.
2. **Rendering methods:**
   - `RenderTemplate(ctx, source, data) (RenderResult, error)` — inline template string
   - `Render(ctx, name, data) (RenderResult, error)` — named template from store
   - `RenderTo(ctx, name, data, w) error` — stream to writer (no RenderResult)
   - `LoadTemplate(name) (*compiler.Bytecode, error)` — compile and cache
3. **Engine configuration:**
   - `SetGlobal(key, value)` — register global variable
   - `RegisterFilter(name, fn)` — register custom filter
4. **Options** — `WithStore`, `WithStrictVariables`, `WithCacheSize`, `WithSandbox`
5. **SandboxConfig** — `AllowedTags`, `AllowedFilters`, `MaxLoopIter`
6. **Data** — `grove.Data` alias for `map[string]any`
7. **Resolvable** — Interface for Go types to expose fields to templates. Method signature: `WispyResolve(key string) (any, bool)` (note: the method name is a legacy holdover from before the project rename). Complete example.
8. **Stores:**
   - `MemoryStore` — `NewMemoryStore()`, `Set(name, content)`
   - `FileSystemStore` — `NewFileSystemStore(root)`, path security (rejects `..` and absolute paths)
9. **RenderResult** — Fields and methods (brief, links to web-primitives.md for detail)
10. **Filter types** — `FilterFn`, `FilterDef`, `FilterFunc()`, `FilterOutputsHTML()`, `FilterSet`
11. **Value types** — `Value`, `Nil`, `StringValue()`, `SafeHTMLValue()`, `ArgInt()`
12. **Error types** — `ParseError` (Template, Line, Column fields), `RuntimeError`

### `docs/examples.md`

Walkthrough of the blog example app in `examples/blog/`.

**Sections:**

1. **Overview** — What the blog app demonstrates: template inheritance, components, asset collection, Resolvable types, full request/response cycle.
2. **Project structure** — File tree with one-line descriptions of each template.
3. **The Go application** — How `main.go` sets up the engine, store, globals, routes, and assembles responses with `writeResult()`.
4. **Base layout** — Walk through `base.grov`: blocks, asset placeholders, meta placeholders.
5. **Page templates** — How `index.grov` and `post.grov` extend the base and use components.
6. **Components** — Walk through `card.grov` (props + slots) and `alert.grov` (let block + conditional styling).
7. **Running it** — `go run examples/blog/main.go`, what to expect at `localhost:3000`.

## Style Guidelines

- **Voice:** Direct, second-person ("you"). No marketing language.
- **Code examples:** Every concept has a code example. Template examples use `.grov` syntax highlighting (falls back to `jinja2` or plain text on GitHub). Go examples use `go`.
- **Cross-references:** Link between docs pages where concepts connect. Don't duplicate explanations — link to the authoritative page.
- **Length:** Each page should be readable in one sitting. The filter catalog will be the longest page; that's fine since it's a reference.

## What Is NOT In Scope

- Hosted documentation site (Hugo, Docusaurus, etc.)
- API docs generated from godoc comments
- Changelog or migration guide
- Contributing guide
- Benchmarks documentation (already exists at `benchmarks/README.md`)
