# Grove v2 — HTML-Native Template Syntax Specification

**Status:** Draft — iterating
**Predecessor:** `master-spec.md` (Jinja2-style syntax)
**Scope:** Syntax only — the render pipeline, bytecode, VM, value system, scope chain, type coercion, caching, concurrency model, and public Go API are unchanged.

---

## Table of Contents

1. [Design Philosophy](#1-design-philosophy)
2. [Syntax Overview](#2-syntax-overview)
3. [Interpolation & Expressions](#3-interpolation--expressions)
4. [Filters](#4-filters)
5. [Control Flow Elements](#5-control-flow-elements)
6. [Assignment & Variable Binding](#6-assignment--variable-binding)
7. [Capture](#7-capture)
8. [Template Inheritance](#8-template-inheritance)
9. [Components & Slots](#9-components--slots)
10. [Inline Component Definitions](#10-inline-component-definitions)
11. [Web Primitives](#11-web-primitives)
12. [Comments & Verbatim](#12-comments--verbatim)
13. [Whitespace Control](#13-whitespace-control)
14. [Reserved Element Names](#14-reserved-element-names)
15. [Attribute Expression Syntax](#15-attribute-expression-syntax)
16. [Component Name Resolution](#16-component-name-resolution)
17. [Error Model](#17-error-model)
18. [Rendering Guarantees](#18-rendering-guarantees)
19. [Complete Examples](#19-complete-examples)

---

## 1. Design Philosophy

### What Changed

The v1 syntax used Jinja2-style delimiters (`{% tag %}`) for all structural constructs. The v2 syntax replaces block-level constructs — things that interact with HTML structure — with PascalCase HTML elements, while retaining `{% %}` delimiters for pure programming constructs (assignment, declarations).

### Why

- **HTML familiarity** — developers working in `.grov` files are writing HTML. The structural constructs should look and feel like HTML.
- **Tooling** — HTML-shaped syntax gets better editor support (folding, matching, highlighting) out of the box.
- **Component ergonomics** — user-defined components (`<Card>`, `<Modal>`) are first-class HTML elements, not string references in `{% component "card" %}`.
- **Unified composition** — components are the single reuse mechanism, replacing v1's separate `include`, `render`, `component`, and `macro` systems.
- **Inspired by** Svelte (element-based control flow, `{expr}` attribute expressions) and Vue (HTML-first templates, `{{ }}` interpolation).

### The Split Rule

> **HTML elements** = things that are HTML or define HTML structure (control flow, content regions, assets, meta, components)
> **`{% %}` tags** = programming constructs (assignment, function definitions, declarations)

| System | Used For |
|--------|----------|
| `{{ expr }}` | Output interpolation |
| `<Element>` | Control flow, inheritance, blocks, capture, hoist, verbatim, slots, fills, defines, web primitives, user components |
| `{% tag %}` | `set`, `let`/`endlet`, `props` |
| `{# comment #}` | Template comments |

---

## 2. Syntax Overview

### At a Glance

```html
<Extends src="layouts/base">
  <Block name="content">
    <h1>{{ page.title }}</h1>

    <If test={user.loggedIn}>
      <p>Welcome back, {{ user.name | capitalize }}!</p>
    <Else>
      <p>Please <a href="/login">log in</a>.</p>
    </If>

    <For each={posts} as="post">
      <article>
        <h2>{{ post.title }}</h2>
        <p>{{ post.body | truncate(200) }}</p>
      </article>
    <Empty>
      <p>No posts yet.</p>
    </For>

    {% set total = posts | length %}
    <p>{{ total }} post{{ total != 1 ? "s" : "" }}.</p>
  </Block>
</Extends>
```

---

## 3. Interpolation & Expressions

### Output

```html
{{ expression }}
```

Evaluates the expression, HTML-escapes the result, and writes it to the output buffer. Values of type `SafeHTML` bypass escaping.

### Expression Syntax

The full expression language is unchanged from v1:

```html
{{ user.name }}                       {# attribute access #}
{{ items[0].title }}                  {# index + attribute #}
{{ config["debug"] }}                 {# string key index #}
{{ count + 1 }}                       {# arithmetic #}
{{ "Hello, " ~ user.name }}           {# string concatenation #}
{{ price * 1.2 | round(2) }}          {# expression + filter #}
{{ active ? name : "Guest" }}         {# ternary #}
{{ not user.banned }}                  {# negation #}
{{ a > b and c != d }}                {# logical operators #}
```

### Whitespace-Trimming Output

```html
{{- expr -}}    {# strips whitespace before and after #}
{{- expr }}     {# strips whitespace before only #}
{{ expr -}}     {# strips whitespace after only #}
```

### Operator Precedence

| Level | Operators | Description |
|-------|-----------|-------------|
| 1 | `.`, `[]`, `()` | Attribute access, index, function call |
| 2 | `\|` | Filter pipe |
| 3 | `not`, `-` (unary) | Negation |
| 4 | `*`, `/`, `%` | Multiplicative |
| 5 | `+`, `-`, `~` | Additive, string concatenation |
| 6 | `<`, `<=`, `>`, `>=`, `==`, `!=` | Comparison |
| 7 | `and` | Logical AND |
| 8 | `or` | Logical OR |
| 9 | `? :` | Ternary conditional |

### Data Literals

```html
{% set colors = ["red", "green", "blue"] %}
{% set matrix = [[1, 2], [3, 4]] %}
{% set theme = { bg: "#fff", fg: "#333" } %}
{% set nested = { card: { padding: "1rem" } } %}
```

- Lists: `[expr, ...]` — comma-separated, trailing comma allowed
- Maps: `{ key: expr, ... }` — keys are unquoted identifiers, ordered by insertion
- No computed keys, no spread/merge operators

---

## 4. Filters

### Pipe Syntax

Filters are applied inside `{{ }}` expressions using the pipe operator:

```html
{{ name | upper }}
{{ bio | truncate(120, "...") }}
{{ items | sort | reverse | first }}
{{ price | round(2) }}
{{ user_input | safe }}
```

### Filter Reference

**String:** `upper`, `lower`, `title`, `capitalize`, `trim`, `lstrip`, `rstrip`, `replace(old, new)`, `truncate(n, suffix)`, `center(w)`, `ljust(w)`, `rjust(w)`, `split(sep)`, `wordcount`

**Collection:** `length`, `first`, `last`, `join(sep)`, `sort`, `reverse`, `unique`, `min`, `max`, `sum`, `map(attr)`, `batch(size)`, `flatten`, `keys`, `values`

**Numeric:** `abs`, `round(n)`, `ceil`, `floor`, `int`, `float`

**Type/Logic:** `default(fallback)`, `string`, `bool`

**HTML:** `escape`, `striptags`, `nl2br`

**Special:** `safe` — marks string as trusted HTML, the only escape hatch for auto-escaping

### Custom Filter Registration

Filters are registered via the Go API:

```go
eng.RegisterFilter("shout", func(v grove.Value, args []grove.Value) (grove.Value, error) {
    return grove.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
})
```

---

## 5. Control Flow Elements

Control flow constructs wrap HTML content and use PascalCase HTML element syntax.

### If / ElseIf / Else

```html
<If test={expression}>
  ...
</If>

<If test={expression}>
  ...
<ElseIf test={other_expression}>
  ...
<Else>
  ...
</If>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `test` | Yes | Expression to evaluate for truthiness |

**Rules:**
- `<ElseIf>` requires a `test` attribute — multiple `<ElseIf>` branches allowed
- `<Else>` takes no attributes
- `<ElseIf>` and `<Else>` are **branch separators** — they do not have closing tags (see [Branch Separators](#branch-separators))
- The entire chain is closed by a single `</If>`
- `<ElseIf>` and `<Else>` appearing outside an `<If>` is a parse error

**Examples:**

```html
{# Simple conditional #}
<If test={user.isAdmin}>
  <span class="badge">Admin</span>
</If>

{# Full chain #}
<If test={status == "active"}>
  <span class="green">Active</span>
<ElseIf test={status == "pending"}>
  <span class="yellow">Pending</span>
<ElseIf test={status == "suspended"}>
  <span class="red">Suspended</span>
<Else>
  <span class="gray">Unknown</span>
</If>
```

### For / Empty

```html
<For each={iterable} as="item">
  ...
</For>

<For each={iterable} as="item">
  ...
<Empty>
  ...
</For>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `each` | Yes | Expression evaluating to an iterable (list or map) |
| `as` | Yes | Variable name (string) to bind each element to |
| `key` | No | Variable name (string) for the key (map) or index (list) |

Note: `as` and `key` are **variable binding names**, not expressions — they are always string literals (`as="item"`, not `as={item}`). This is analogous to declaring a variable name, not referencing one.

**Rules:**
- `<Empty>` is a **branch separator** — no closing tag (see [Branch Separators](#branch-separators))
- The entire block is closed by `</For>`
- `<Empty>` appearing outside a `<For>` is a parse error

**Examples:**

```html
{# Simple iteration #}
<For each={items} as="item">
  <li>{{ item.name }}</li>
</For>

{# With empty fallback #}
<For each={results} as="result">
  <div class="result">{{ result.title }}</div>
<Empty>
  <p>No results found.</p>
</For>

{# Two-variable form (index + value) #}
<For each={items} as="item" key="i">
  <li>{{ i + 1 }}. {{ item.name }}</li>
</For>

{# Map iteration (key + value) #}
<For each={settings} as="value" key="name">
  <dt>{{ name }}</dt>
  <dd>{{ value }}</dd>
</For>

{# Range iteration #}
<For each={range(1, 11)} as="i">
  <li>Item {{ i }}</li>
</For>

{# Nested loops with loop variable #}
<For each={categories} as="cat">
  <h2>{{ cat.name }}</h2>
  <For each={cat.items} as="item">
    <p>{{ loop.parent.index }}.{{ loop.index }}: {{ item }}</p>
  </For>
</For>
```

### Loop Variable

Available inside `<For>` body:

| Variable | Description |
|----------|-------------|
| `loop.index` | 1-based position |
| `loop.index0` | 0-based position |
| `loop.first` | `true` on first iteration |
| `loop.last` | `true` on last iteration |
| `loop.length` | Total items in the collection |
| `loop.depth` | 1 for outer, 2 for first nested, etc. |
| `loop.parent` | Parent loop's `loop` object (nil if outermost) |

**Inside `<Empty>`:** The `loop` variable is available with `loop.length == 0`. All positional fields (`index`, `first`, `last`) are undefined. Only `loop.length`, `loop.depth`, and `loop.parent` are meaningful.

### Range Function

- `range(stop)` — `[0, stop)`
- `range(start, stop)` — `[start, stop)` (end-exclusive)
- `range(start, stop, step)` — stepped sequence

### Branch Separators

`<ElseIf>`, `<Else>`, and `<Empty>` are **branch separators**, not independent elements. They:

- Do **not** have closing tags (no `</Else>`, `</ElseIf>`, `</Empty>`)
- Must appear inside their parent element (`<If>` or `<For>`)
- Divide the parent's content into branches
- Are terminated by the next branch separator or the parent's closing tag

```html
{# Correct #}
<If test={x}>A<ElseIf test={y}>B<Else>C</If>

{# Wrong — no </Else> tag exists #}
<If test={x}>A<Else>B</Else></If>
```

---

## 6. Assignment & Variable Binding

Assignment constructs are programming statements — they use `{% %}` tags.

### Set

```
{% set name = expression %}
```

Single variable assignment. Writes to the current scope.

```html
{% set title = "Welcome" %}
{% set total = items | length %}
{% set full_name = first ~ " " ~ last %}
{% set colors = ["red", "green", "blue"] %}
```

### Let (Multi-Variable Block)

```
{% let %}
  name = expression
  name = expression
  if condition
    name = expression
  elif condition
    name = expression
  else
    name = expression
  end
{% endlet %}
```

Block assignment with a mini-DSL for computing multiple related variables.

**Rules:**
- Bare `name = expression` per line (no delimiters inside the block)
- Full expression syntax on right-hand side
- `if/elif/else/end` conditionals (note: `end` not `endif` — this is a deliberate simplification for the assignment DSL, which has no nesting and no HTML content)
- All assigned variables are written to the outer scope
- No HTML output inside the block

```html
{% let %}
  bg = "#d1ecf1"
  border = "#bee5eb"
  fg = "#0c5460"

  if type == "warning"
    bg = "#fff3cd"
    fg = "#856404"
  elif type == "error"
    bg = "#f8d7da"
    fg = "#721c24"
  end
{% endlet %}

<div style="background: {{ bg }}; color: {{ fg }}; border: 1px solid {{ border }}">
  {{ message }}
</div>
```

---

## 7. Capture

Capture redirects rendered output into a variable. It wraps HTML content and uses element syntax.

```html
<Capture name="nav">
  <For each={menu} as="item">
    <a href="{{ item.url }}">{{ item.label }}</a>
  </For>
</Capture>

{# Use the captured content later #}
<nav>{{ nav }}</nav>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Variable name to store the captured output |

The captured value is a `SafeHTML` string stored in the current scope. Content inside the capture block is auto-escaped during rendering (so `{{ user_input }}` is escaped), then the assembled result is marked as `SafeHTML` to prevent double-escaping when output later.

---

## 8. Template Inheritance

Template inheritance uses the `<Extends>` wrapping element. A child template wraps all of its block overrides inside `<Extends>`, making the parent-child relationship explicit and self-describing.

### Extends

```html
<Extends src="path/to/parent">
  <Block name="block_name">override content</Block>
</Extends>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | Yes | Path to the parent layout template (resolved via store) |

**Rules:**
- `<Extends>` must be the root element of the template — nothing outside it (except comments and whitespace)
- Only `<Block>` elements are allowed as direct children of `<Extends>`
- Content outside `<Block>` elements inside `<Extends>` is a parse error
- Inheritance is static and unconditional — you cannot conditionally extend different parents. This is a deliberate design choice for predictability.

### Block

```html
<Block name="block_name">
  content
</Block>
```

Defines a named block. In parent templates, blocks define default content. In child templates (inside `<Extends>`), blocks override the parent's version.

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Block identifier — must match between parent and child |

A `<Block>` appearing outside both a parent layout and an `<Extends>` wrapper is a parse error.

### Super

`super()` renders the parent's version of the current block. Called as a function inside `{{ }}`:

```html
{{ super() }}
```

### Full Inheritance Example

**Parent layout** — `layouts/base.grov`:
```html
<!DOCTYPE html>
<html>
<head>
  <title><Block name="title">My Site</Block></title>
  <Block name="head"></Block>
</head>
<body>
  <header>
    <Block name="header">
      <Nav />
    </Block>
  </header>

  <main>
    <Block name="content"></Block>
  </main>

  <footer>
    <Block name="footer">
      <p>&copy; 2026 My Site</p>
    </Block>
  </footer>
</body>
</html>
```

**Child template** — `pages/about.grov`:
```html
<Extends src="layouts/base">
  <Block name="title">About — {{ super() }}</Block>

  <Block name="head">
    <ImportAsset src="/css/about.css" type="stylesheet" />
  </Block>

  <Block name="content">
    <h1>About Us</h1>
    <p>Welcome to our site.</p>
  </Block>
</Extends>
```

### Multi-Level Inheritance

Chained inheritance works naturally. Each `<Extends>` layer accumulates block overrides before delegating to its parent. `super()` walks one level up the chain.

**Grandchild** — `pages/about-team.grov`:
```html
<Extends src="pages/about">
  <Block name="content">
    <h1>Our Team</h1>
    {{ super() }}
    <TeamGrid members={team} />
  </Block>
</Extends>
```

---

## 9. Components & Slots

Components are the primary composition mechanism in Grove v2. They replace v1's `{% include %}`, `{% render %}`, `{% component %}`, and `{% macro %}` with a single unified model: **PascalCase HTML elements.**

### Why Components Replace Include, Render, and Macros

In v1, there were four overlapping ways to compose templates:

| v1 Construct | Scope | Props Validation | Content Passing |
|-------------|-------|-----------------|-----------------|
| `{% include "x" %}` | Shared (caller's scope) | No | No |
| `{% render "x" %}` | Isolated | No | No |
| `{% component "x" %}` | Isolated | Yes (`{% props %}`) | Named slots |
| `{% macro name() %}` | Isolated | Positional + named args | Single `caller()` body |

In v2, **all four are replaced by components.** Every `.grov` file can be invoked as a PascalCase element. Components always use isolated scope — data is passed explicitly via props. This is simpler, more explicit, and aligns with the HTML-native design.

- `{% include "partials/nav" %}` → `<Nav />`
- `{% include "partials/nav" section="about" %}` → `<Nav section="about" />`
- `{% render "components/card" title="x" %}` → `<Card title="x" />`
- `{% component "card" title="x" %}...{% endcomponent %}` → `<Card title="x">...</Card>`
- `{% macro input(name) %}...{% endmacro %}` → `<Define name="Input">...</Define>` (see [Section 10](#10-inline-component-definitions))

Components without `{% props %}` accept any props (permissive mode), which covers the simple partial use case. Components with `{% props %}` get full validation.

### Component Definition

A component is a `.grov` file that declares its accepted props and defines named slots for caller content.

**`components/card.grov`:**
```html
{% props title, variant="default", elevated=false %}

<div class="card card--{{ variant }}{{ elevated ? " card--elevated" : "" }}">
  <h2>{{ title }}</h2>

  <Slot name="actions" />

  <div class="body">
    <Slot />
  </div>

  <footer>
    <Slot name="footer">
      <p>Default footer content</p>
    </Slot>
  </footer>
</div>
```

### Component Usage

Components are invoked as PascalCase HTML elements:

```html
<Card title="Orders" variant="primary">
  <p>This content feeds the default slot.</p>

  <Fill slot="actions">
    <button>View All</button>
    <button>Export</button>
  </Fill>

  <Fill slot="footer">
    <p>Custom footer for this card.</p>
  </Fill>
</Card>
```

### Props

```
{% props name, key="default", ... %}
```

Declared inside a component template (or `<Define>` block).

**Rules:**
- Parameters without defaults are required — missing props produce a `RuntimeError`
- Unknown props (not declared) produce a `RuntimeError`
- If no `{% props %}` declaration exists, all passed props are accepted (permissive mode)
- Props are available as variables in the component's scope

**Attribute values on the component element:**

| Attribute syntax | Meaning |
|-----------------|---------|
| `title="Orders"` | String literal prop |
| `count={items \| length}` | Expression prop |
| `elevated` | Boolean `true` (bare attribute, like HTML) |
| `elevated={false}` | Boolean `false` (explicit expression) |

### Slots (Definition Side)

Slots are defined inside component templates using the `<Slot>` element:

```html
<Slot />                                          {# default (unnamed) slot #}
<Slot name="actions" />                           {# named slot, no fallback #}
<Slot name="footer">Default footer</Slot>         {# named slot with fallback #}
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | No | Slot identifier. Omit for the default slot. |

**Rules:**
- A component may have at most one default (unnamed) slot
- Named slots are identified by string
- Fallback content renders when the caller does not provide a `<Fill>` for that slot
- Fallback content is rendered in the component's scope (has access to props)
- `<Slot>` appearing outside a component or `<Define>` is a parse error

### Fills (Usage Side)

Fills are provided at the component call site using the `<Fill>` element:

```html
<Fill slot="actions">
  <button>Go</button>
</Fill>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `slot` | Yes | Name of the slot to fill |

**Rules:**
- Content between `<Fill>` and `</Fill>` is rendered in the **caller's scope** (not the component's)
- Content outside any `<Fill>` block feeds the default slot
- Fills for slots that don't exist in the component are silently ignored
- `<Fill>` appearing outside a component invocation is a parse error

### Prop Shorthand

When a variable name matches the prop name, use shorthand to avoid repetition:

```html
{# Verbose #}
<Card title={title} count={count} active={active} />

{# Shorthand — variable name matches prop name #}
<Card {title} {count} {active} />
```

Shorthand `{name}` is equivalent to `name={name}`. It works on both component props and reserved element attributes:

```html
{% set items = data.items %}
<For each={items} as="item">...</For>

{# Equivalent shorthand #}
<For each={items} as="item">...</For>  {# 'each' doesn't benefit here #}
```

Shorthand is most useful for prop-heavy component invocations.

### Spread Props

Pass all entries of a map as individual props using spread syntax:

```html
{% set options = { title: "Hello", variant: "primary", elevated: true } %}

{# Without spread #}
<Card title={options.title} variant={options.variant} elevated={options.elevated} />

{# With spread #}
<Card {...options} />
```

**Rules:**
- The spread expression must evaluate to a map
- Spread props are applied in order — later attributes override earlier ones
- Explicit attributes override spread values:
  ```html
  {# variant="ghost" wins over options.variant #}
  <Card {...options} variant="ghost" />
  ```
- Multiple spreads are allowed: `<Card {...defaults} {...overrides} />`

### Built-in `props` Variable

Inside a component, the `props` variable holds all received props as a map. This enables prop forwarding in wrapper components:

```html
{# components/fancy-card.grov — wraps Card with extra styling #}
{% props highlighted=false %}

<div class="{{ highlighted ? 'highlight' : '' }}">
  <Card {...props} />
</div>
```

**Rules:**
- `props` is always available inside a component, even in permissive mode (no `{% props %}` declaration)
- In strict mode (with `{% props %}`), `props` contains only the declared props with their resolved values (defaults applied)
- `props` is read-only — modifying it has no effect
- `props` does not include `highlighted` in the example above if it's consumed by the wrapper's own `{% props %}` declaration. Only "pass-through" props (not declared by the component) appear in `props` when using permissive mode.

### Scoped Slots (Slot Props)

Slots can pass data *back* to the fill, enabling "renderless" components — components that provide logic/data while the caller controls the HTML.

**Definition side** — pass data via attributes on `<Slot>`:

```html
{# components/fetch-data.grov #}
{% props url %}
{# ... imagine fetching logic populating result, isLoading, error ... #}
<Slot data={result} loading={isLoading} error={error} />
```

**Usage side** — receive slot props via `let:name` attributes on `<Fill>`:

```html
<FetchData url="/api/users">
  <Fill slot="default" let:data let:loading let:error>
    <If test={loading}>
      <Spinner />
    <ElseIf test={error}>
      <p class="error">{{ error }}</p>
    <Else>
      <UserList users={data} />
    </If>
  </Fill>
</FetchData>
```

**`let:name` syntax:**

| Syntax | Meaning |
|--------|---------|
| `let:data` | Bind slot prop `data` to variable `data` in fill scope |
| `let:data="users"` | Bind slot prop `data` to variable `users` in fill scope (rename) |

**Rules:**
- Slot props are passed as attributes on `<Slot>`: `<Slot data={value} />`
- Fill receives them via `let:name` attributes on `<Fill>`
- `let:` bindings are available only inside the `<Fill>` body
- Slot props on the default slot work with default slot content (no `<Fill>` needed):
  ```html
  <FetchData url="/api/users" let:data let:loading>
    {# default slot content with slot props #}
    <UserList users={data} />
  </FetchData>
  ```
  When `let:` attributes appear on the component element itself, they bind slot props from the default slot.
- Named slots use `let:` on the `<Fill>` element
- Unused slot props are silently ignored

### Dynamic Components

The `<Component>` reserved element renders a component chosen at runtime:

```html
<Component is={widgetType} title="Hello" data={widgetData} />

<Component is="Card" title="Static name also works" />
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `is` | Yes | Component name — string literal or expression evaluating to a PascalCase name |

**Rules:**
- All attributes other than `is` are passed as props to the resolved component
- The `is` value must be a valid PascalCase component name (reserved names are not allowed)
- If the name doesn't resolve to a component, it is a `RuntimeError` (not parse-time, since the name is dynamic)
- Slots and fills work normally inside `<Component>`:
  ```html
  <Component is={cardType} title="Orders">
    <p>Default slot content.</p>
    <Fill slot="actions"><button>Go</button></Fill>
  </Component>
  ```
- Spread props work: `<Component is={type} {...options} />`

### Slot Forwarding

A wrapper component can forward its slots to a child component. No special syntax is needed — `<Slot>` inside a `<Fill>` naturally forwards:

```html
{# components/fancy-card.grov — wraps Card #}
{% props title, highlighted=false %}

<div class="{{ highlighted ? 'highlight' : '' }}">
  <Card title={title}>
    {# Forward default slot #}
    <Slot />

    {# Forward named slots #}
    <Fill slot="actions">
      <Slot name="actions" />
    </Fill>

    <Fill slot="footer">
      <Slot name="footer">
        {# Provide a default if caller doesn't fill "footer" #}
        <p>Fancy default footer</p>
      </Slot>
    </Fill>
  </Card>
</div>
```

**Usage — slots pass through transparently:**
```html
<FancyCard title="Orders" highlighted>
  <p>This reaches Card's default slot via forwarding.</p>

  <Fill slot="actions">
    <button>This reaches Card's actions slot.</button>
  </Fill>
</FancyCard>
```

**How it works:**
- `<Slot />` inside FancyCard's body renders whatever the caller passed to FancyCard's default slot
- That rendered content is placed inside Card's body (feeding Card's default slot)
- `<Slot name="actions" />` renders what the caller passed to FancyCard's "actions" slot
- That content is wrapped in `<Fill slot="actions">`, feeding Card's "actions" slot

Scoped slot props can also be forwarded:

```html
<Fill slot="items">
  <Slot name="items" let:item>
    {# Re-expose the slot prop #}
  </Slot>
</Fill>
```

### Fragment Support

Component templates may have multiple root elements. There is no requirement for a single root:

```html
{# components/table-row.grov — valid, multiple root elements #}
{% props name, value %}

<dt>{{ name }}</dt>
<dd>{{ value }}</dd>
```

**Rules:**
- File-based components and `<Define>` blocks can have any number of root elements
- Text nodes, HTML elements, and Grove elements can all be roots
- This mirrors how modern frameworks (React, Vue 3, Svelte) handle fragments

### Self-Closing Components

Components with no children (no default slot content, no fills) use self-closing syntax:

```html
<Icon name="star" size={16} />
<Divider />
<Spacer height={24} />
```

### Nested Components

Components can be nested naturally:

```html
<Card title="User Profile">
  <Avatar src={user.photo} size="large" />

  <DataList>
    <For each={user.fields} as="field">
      <DataItem label={field.name} value={field.value} />
    </For>
  </DataList>

  <Fill slot="actions">
    <Button variant="primary">Edit</Button>
    <Button variant="ghost">Cancel</Button>
  </Fill>
</Card>
```

---

## 10. Inline Component Definitions

The `<Define>` element creates a component inline — in the same file where it's used — without requiring a separate `.grov` file. This is ideal for small helpers that don't warrant their own file.

### Syntax

```html
<Define name="ComponentName">
  {% props param1, param2="default" %}
  {# component body — same as a file-based component #}
</Define>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | PascalCase component name — becomes available as an element |

### Rules

- The `name` must be PascalCase (like all component names)
- The defined component is available from the point of definition onward in the current template
- Multiple `<Define>` blocks can exist in one template
- `<Define>` can be used in any template: pages, layouts, and component files
- `<Define>` inside `<Define>` is not allowed (no nested definitions)
- The `name` must not conflict with [reserved element names](#14-reserved-element-names)
- If the `name` conflicts with a file-based component (same name resolves via the store), it is a **parse error** — name conflicts must be resolved explicitly. This prevents subtle bugs where an inline definition silently shadows a file-based component.
- The body supports everything a file-based component does: `{% props %}`, `<Slot>`, `<Fill>`, control flow, other component invocations

### Examples

**Simple helper:**
```html
<Define name="Icon">
  {% props name, size=16 %}
  <svg class="icon icon-{{ name }}" width="{{ size }}" height="{{ size }}">
    <use href="/icons.svg#{{ name }}" />
  </svg>
</Define>

<Icon name="check" size={24} />
<Icon name="star" />
```

**With slots:**
```html
<Define name="Card">
  {% props title, variant="default" %}
  <div class="card card--{{ variant }}">
    <h2>{{ title }}</h2>
    <Slot />
    <footer>
      <Slot name="footer">Default footer</Slot>
    </footer>
  </div>
</Define>

<Card title="News" variant="primary">
  <p>Card body content.</p>
  <Fill slot="footer">
    <p>Custom footer.</p>
  </Fill>
</Card>
```

**Multiple defines in one template:**
```html
<Define name="Badge">
  {% props label, color="gray" %}
  <span class="badge badge--{{ color }}">{{ label }}</span>
</Define>

<Define name="UserRow">
  {% props user %}
  <tr>
    <td>{{ user.name }}</td>
    <td>{{ user.email }}</td>
    <td>
      <Badge label={user.role} color={user.role == "admin" ? "red" : "blue"} />
    </td>
  </tr>
</Define>

<table>
  <For each={users} as="user">
    <UserRow user={user} />
  </For>
</table>
```

**Inside a component file** — helpers scoped to a component:

**`components/dashboard.grov`:**
```html
{% props widgets %}

<Define name="WidgetFrame">
  {% props title, icon %}
  <div class="widget">
    <h3><Icon name={icon} /> {{ title }}</h3>
    <Slot />
  </div>
</Define>

<div class="dashboard">
  <For each={widgets} as="w">
    <WidgetFrame title={w.name} icon={w.icon}>
      <p>{{ w.summary }}</p>
    </WidgetFrame>
  </For>
</div>
```

### Expression-Level Use via Capture

Unlike v1 macros, components cannot be called inside expressions. If you need a component's output as a string value (e.g., to embed in an attribute), use `<Capture>`:

```html
<Capture name="star_icon"><Icon name="star" size={12} /></Capture>
<div title="{{ star_icon }}">Rated</div>
```

---

## 11. Web Primitives

### ImportAsset

Collects asset references into `RenderResult.Assets`.

```html
<ImportAsset src="/css/app.css" type="stylesheet" />
<ImportAsset src="/css/about.css" type="stylesheet" priority={10} />
<ImportAsset src="/js/app.js" type="script" defer />
<ImportAsset src="/js/analytics.js" type="script" async />
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | Yes | Asset path |
| `type` | Yes | Asset type: `"stylesheet"` or `"script"` |
| `priority` | No | Integer — controls ordering within type groups (higher = earlier). Default: 0 |
| `defer`, `async` | No | Boolean flags (bare attributes) — passed through to rendered HTML tags |

**Rules:**
- Self-closing element (`/>`)
- Deduplicated by `src` — identical declarations silently dropped
- Additional bare attributes are treated as boolean flags and included in rendered output

**Go output helpers:**
```go
result.HeadHTML()  // <link rel="stylesheet"> tags for stylesheet assets
result.FootHTML()  // <script> tags for script assets
```

### SetMeta

Collects metadata into `RenderResult.Meta`.

```html
<SetMeta name="description" content="A page about Grove." />
<SetMeta property="og:title" content="Grove Engine" />
<SetMeta property="og:image" content="{{ page.image }}" />
```

**Rules:**
- Self-closing element (`/>`)
- Stored in `map[string]string` — last-write-wins semantics
- On key collision, a warning is appended to `RenderResult.Warnings`

### Hoist

Renders body content and appends it to `RenderResult.Hoisted[target]`.

```html
<Hoist target="head">
  <style>
    .about-hero { background: url("{{ hero_image }}"); }
  </style>
</Hoist>

<Hoist target="analytics">
  <script>trackPage("{{ page.title }}");</script>
</Hoist>
```

**Attributes:**

| Attribute | Required | Description |
|-----------|----------|-------------|
| `target` | Yes | User-defined string key for grouping hoisted content |

Retrieved in Go via `result.GetHoisted("analytics")`.

---

## 12. Comments & Verbatim

### Comments

```
{# This is a comment — stripped at parse time, zero runtime cost #}
```

HTML comments (`<!-- -->`) pass through to the output as-is.

### Verbatim

```html
<Verbatim>
  {{ this is not evaluated }}
  {% neither is this %}
  <If test={true}>This is literal text, not a condition.</If>
</Verbatim>
```

Everything inside `<Verbatim>` is emitted verbatim — no expression evaluation, no element parsing. Use this for documenting template syntax, or embedding code samples that use `{{ }}` or `<PascalCase>` patterns.

**Note:** To include `</Verbatim>` as literal text inside a verbatim block, there is no escape mechanism — restructure the content or split into multiple verbatim blocks.

---

## 13. Whitespace Control

### On `{% %}` Tags

The `-` modifier strips adjacent whitespace:

```
{%- set x = 1 -%}    {# strips whitespace before and after #}
```

### On `{{ }}` Output

```
{{- expr -}}          {# strips whitespace before and after output #}
```

### On HTML Elements

HTML elements do not have a whitespace-stripping modifier. The content inside HTML elements is the output — whitespace is controlled by the HTML you write. If you need tight whitespace control around an element boundary, use `{{- -}}` for adjacent output expressions or structure your HTML accordingly.

---

## 14. Reserved Element Names

The following PascalCase names are reserved by Grove and cannot be used as component names or `<Define>` names:

| Name | Type | Closing Tag | Purpose |
|------|------|-------------|---------|
| `Extends` | Block | `</Extends>` | Template inheritance — wraps block overrides |
| `Block` | Block | `</Block>` | Inheritance block definition/override |
| `If` | Block | `</If>` | Conditional — opens an if/elseif/else chain |
| `ElseIf` | Branch separator | None | Conditional branch (inside `<If>`) |
| `Else` | Branch separator | None | Default branch (inside `<If>`) |
| `For` | Block | `</For>` | Loop iteration |
| `Empty` | Branch separator | None | Empty-collection branch (inside `<For>`) |
| `Capture` | Block | `</Capture>` | Output capture into a variable |
| `Hoist` | Block | `</Hoist>` | Content hoisting into render result |
| `Verbatim` | Block | `</Verbatim>` | Literal output (no parsing) |
| `Define` | Block | `</Define>` | Inline component definition |
| `Slot` | Block or self-closing | `</Slot>` or `/>` | Slot definition (in component) |
| `Fill` | Block | `</Fill>` | Slot fill (at call site) |
| `Component` | Block or self-closing | `</Component>` or `/>` | Dynamic component (runtime name resolution) |
| `SetMeta` | Self-closing | `/>` | Metadata collection |
| `ImportAsset` | Self-closing | `/>` | Asset collection |

### Lowercase is HTML

Any element starting with a lowercase letter (or containing a hyphen, like `<my-widget>`) is treated as plain HTML and passed through to the output. However, expression attributes and interpolation are supported on HTML elements (see [Section 15](#15-attribute-expression-syntax)).

---

## 15. Attribute Expression Syntax

### The Rule

Expression attributes (`{expr}`) and string interpolation (`{{ }}`) work on **all elements** — reserved Grove elements, component elements, and plain HTML elements:

| Syntax | Type | Example |
|--------|------|---------|
| `attr="string"` | String literal | `name="sidebar"` |
| `attr={expression}` | Expression (evaluated at render time) | `test={user.active}` |
| `attr` | Boolean `true` (bare attribute) | `elevated` |

### String Literals vs Expressions

```html
{# String literal — the value is the literal text "hello" #}
<Card title="hello">

{# Expression — the value is the result of evaluating the variable `title` #}
<Card title={title}>

{# Expression — string concatenation at render time #}
<Card title={"Page: " ~ page.name}>

{# Expression with filter #}
<Card title={page.name | capitalize}>
```

### Interpolation Inside String Attributes

String attributes on components and reserved elements support `{{ }}` interpolation:

```html
<Card title="Hello, {{ user.name }}!">
<Card class="card card--{{ variant }}">
```

This is syntactic sugar — the attribute is parsed as a template string and compiled to a concatenation expression.

### Boolean Attributes

Bare attributes (no value) are treated as boolean `true`:

```html
<Card elevated>              {# elevated = true #}
<Card elevated={false}>      {# elevated = false #}
<Card elevated={show_card}>  {# elevated = value of show_card #}
```

### Expression Attributes on HTML Elements

Plain HTML elements support `{expr}` attributes and `{{ }}` interpolation in string attributes:

```html
<button disabled={isLoading} class="btn btn--{{ variant }}">
  {{ label }}
</button>

<input type="text" value={currentValue} placeholder="Enter {{ fieldName }}">

<div id="section-{{ index }}" data-active={isActive}>
  ...
</div>

<a href="/users/{{ user.id }}" class={linkClass}>
  {{ user.name }}
</a>
```

**Falsy attribute behavior** — when an expression attribute evaluates to `false` or `nil`, the attribute is **omitted entirely** from the output. This enables conditional HTML attributes:

```html
{# isLoading = true → renders: <button disabled> #}
{# isLoading = false → renders: <button> #}
<button disabled={isLoading}>Submit</button>

{# className = "highlight" → renders: <div class="highlight"> #}
{# className = nil → renders: <div> #}
<div class={className}>...</div>
```

**Truthiness rules for attribute output:**

| Expression result | Rendered output |
|------------------|-----------------|
| `false` | Attribute omitted |
| `nil` | Attribute omitted |
| `""` (empty string) | Attribute omitted |
| `true` | Bare attribute: `disabled` |
| `"value"` (non-empty string) | `attr="value"` |
| Number | `attr="123"` (converted to string) |

**Shorthand and spread on HTML elements:**

Prop shorthand and spread syntax also work on HTML elements:

```html
{% set id = "main" %}
{% set attrs = { class: "container", role: "main" } %}

<div {id} {...attrs}>
  ...
</div>
{# Renders: <div id="main" class="container" role="main"> #}
```

---

## 16. Component Name Resolution

### PascalCase to Path Conversion

Component element names are converted to file paths using the following rules:

1. **PascalCase → kebab-case**: Each uppercase letter (except the first) is preceded by a hyphen. Consecutive uppercase letters are treated as an acronym — all but the last become a single group.
   - `Card` → `card`
   - `UserProfile` → `user-profile`
   - `APIKey` → `api-key`
   - `HTMLParser` → `html-parser`

2. **Store resolution**: The kebab-case name is resolved through the configured store using its fallback rules:
   - Exact match: `<root>/card`
   - Extension: `<root>/card.grov`
   - Directory: `<root>/card/card.grov`

3. **Component directories**: The engine can be configured with component search paths (e.g., `components/`, `partials/`). Resolution checks each directory in order.

### Name Conflict Rules

- If a `<Define>` name matches a file-based component name that would resolve via the store, it is a **parse error**
- If two `<Define>` blocks in the same file use the same name, it is a **parse error**
- Reserved element names cannot be used for components or defines

### Resolution Order

1. Check reserved element names → dispatch to built-in element parser
2. Check `<Define>` definitions in current template (most recent first)
3. Check store resolution (file-based components)
4. If none found → parse error: `unknown component "ComponentName"`

---

## 17. Error Model

All template errors are classified as either **parse errors** (compile-time) or **runtime errors** (during rendering). Parse errors prevent compilation — the template never executes. Runtime errors occur during rendering and halt the current render pass.

### Parse Errors

Detected during lexing, parsing, or compilation. The template is rejected before any bytecode is generated.

| Situation | Error |
|-----------|-------|
| Malformed expression syntax inside `{expr}` or `{{ }}` | `ParseError` |
| Unclosed element — missing `</If>`, `</For>`, etc. | `ParseError` |
| `<ElseIf>` or `<Else>` outside an `<If>` | `ParseError` |
| `<Empty>` outside a `<For>` | `ParseError` |
| `<Block>` outside `<Extends>` or a parent layout | `ParseError` |
| Content outside `<Block>` inside `<Extends>` | `ParseError` |
| `<Slot>` outside a component or `<Define>` | `ParseError` |
| `<Fill>` outside a component invocation | `ParseError` |
| `<Define>` with non-PascalCase name | `ParseError` |
| `<Define>` name conflicts with a reserved element | `ParseError` |
| `<Define>` name conflicts with a file-based component | `ParseError` |
| Duplicate `<Define>` name in the same template | `ParseError` |
| `<Define>` nested inside another `<Define>` | `ParseError` |
| Unknown PascalCase element (no define or file match) | `ParseError` |
| Unclosed `{{ }}`, `{% %}`, or `{# #}` | `ParseError` |
| Sandbox `AllowedTags` violation | `ParseError` |
| Sandbox `AllowedFilters` violation | `ParseError` |

### Runtime Errors

Detected during template execution.

| Situation | Error |
|-----------|-------|
| Missing required prop (no default, not passed by caller) | `RuntimeError` |
| Unknown prop passed to strict component (has `{% props %}`) | `RuntimeError` |
| `super()` called outside a block | `RuntimeError` |
| Undefined variable access (strict mode only) | `RuntimeError` |
| Division by zero | `RuntimeError` |
| Sandbox `MaxLoopIter` exceeded | `RuntimeError` |
| Dynamic `<Component is={name}>` — name doesn't resolve | `RuntimeError` |
| Template not found in store (for file-based components) | `RuntimeError` |
| Type error in filter arguments | `RuntimeError` |
| Index out of bounds on a list (returns nil — not an error unless strict mode) | Nil / `RuntimeError` (strict) |

### Error Format

```
template.grov:42:7: unexpected token "}"
template.grov:15: missing required prop "title"
line 3:12: unclosed expression     (inline templates)
```

Both `ParseError` and `RuntimeError` implement Go's `error` interface and can be unwrapped with `errors.As`.

---

## 18. Rendering Guarantees

### Execution Model

- **Synchronous** — all rendering is synchronous. Components, slots, and fills are rendered inline during the parent's render pass. There is no async/await or deferred rendering.
- **Single-pass** — each template is rendered in a single pass through its bytecode. No multi-pass resolution.
- **Deterministic** — given the same input data, a template always produces the same output.

### Component Rendering

- A component's body is fully rendered before its output is inserted into the parent's output buffer.
- Slot fallback content is rendered **lazily** — only when the caller does not provide a `<Fill>` for that slot. If a fill is provided, the fallback is never rendered.
- `<Fill>` content is rendered in the caller's scope at the point where the corresponding `<Slot>` appears in the component's body.
- `<Define>` bodies are **not rendered at definition time** — they are compiled and registered. Rendering happens only when the component is invoked.

### Scope Isolation

- Components always execute in an **isolated scope** — they see only their declared props, engine globals, and variables set within the component itself.
- `<Fill>` content renders in the **caller's scope** — it can access the caller's variables but not the component's props.
- Scoped slot props (`let:name`) create bindings in the fill's scope, bridging data from the component to the caller.
- `<For>` creates a child scope for each iteration — loop variables and `{% set %}` inside the loop do not leak to the outer scope.
- `<If>` does **not** create a new scope — `{% set %}` inside an `<If>` writes to the enclosing scope.

### Fragment Rendering

- Components with multiple root elements render all roots in document order.
- The output is a concatenation of all root element outputs.
- There is no implicit wrapper element.

---

## 19. Complete Examples

### Example 1: Blog Post List

```html
<Extends src="layouts/base">
  <Block name="title">{{ section | title }} — Blog</Block>

  <Block name="head">
    <ImportAsset src="/css/blog.css" type="stylesheet" />
    <SetMeta property="og:title" content="Blog — {{ section }}" />
  </Block>

  <Block name="content">
    <h1>{{ section | title }}</h1>

    {% set featured = posts | first %}

    <If test={featured}>
      <article class="featured">
        <h2>{{ featured.title }}</h2>
        <p>{{ featured.excerpt | truncate(300) }}</p>
      </article>
    </If>

    <For each={posts} as="post">
      <If test={not loop.first}>
        <article>
          <h2>
            <a href="{{ post.url }}">{{ post.title }}</a>
          </h2>
          <p>{{ post.excerpt | truncate(150) }}</p>
          <time>{{ post.date }}</time>
        </article>
      </If>
    <Empty>
      <p>No posts in this section.</p>
    </For>
  </Block>
</Extends>
```

### Example 2: Component Definition & Usage

**`components/alert.grov`:**
```html
{% props message, type="info", dismissible=false %}

{% let %}
  bg = "#d1ecf1"
  fg = "#0c5460"
  icon = "info"

  if type == "warning"
    bg = "#fff3cd"
    fg = "#856404"
    icon = "alert"
  elif type == "error"
    bg = "#f8d7da"
    fg = "#721c24"
    icon = "error"
  end
{% endlet %}

<div class="alert" style="background: {{ bg }}; color: {{ fg }}" role="alert">
  <Icon name={icon} size={20} />
  <span>{{ message }}</span>

  <Slot name="actions" />

  <If test={dismissible}>
    <button class="close" aria-label="Dismiss">&times;</button>
  </If>
</div>
```

**Usage:**
```html
<Alert message="Your order has been placed!" type="info">
  <Fill slot="actions">
    <a href="/orders">View Order</a>
  </Fill>
</Alert>

<Alert message="Something went wrong." type="error" dismissible />
```

### Example 3: Inline Defines with Reuse

```html
<Define name="Field">
  {% props name, label, type="text", required=false, error="" %}
  <div class="field{{ error ? ' field--error' : '' }}">
    <label for="{{ name }}">
      {{ label }}
      <If test={required}>
        <span class="required">*</span>
      </If>
    </label>
    <input id="{{ name }}" name="{{ name }}" type="{{ type }}"
           {{ required ? "required" : "" }}>
    <If test={error}>
      <p class="error">{{ error }}</p>
    </If>
  </div>
</Define>

<Define name="SubmitButton">
  {% props text="Submit", variant="primary" %}
  <button type="submit" class="btn btn--{{ variant }}">{{ text }}</button>
</Define>

<form method="post" action="/register">
  <Field name="name" label="Full Name" required />
  <Field name="email" label="Email" type="email" required error={errors.email} />
  <Field name="bio" label="Biography" />
  <SubmitButton text="Create Account" />
</form>
```

### Example 4: Nested Components with Slots

```html
<Extends src="layouts/base">
  <Block name="content">
    <Dashboard>
      <For each={widgets} as="widget">
        <Card title={widget.name} variant={widget.style}>
          <If test={widget.type == "chart"}>
            <Chart data={widget.data} />
          <ElseIf test={widget.type == "table"}>
            <DataTable rows={widget.rows} columns={widget.cols} />
          <Else>
            <p>{{ widget.content }}</p>
          </If>

          <Fill slot="actions">
            <Button size="small">Refresh</Button>
          </Fill>
        </Card>
      </For>
    </Dashboard>
  </Block>
</Extends>
```

### Example 5: Full Page with Hoisting

```html
<Extends src="layouts/base">
  <Block name="title">{{ product.name }} — Store</Block>

  <Block name="content">
    <ImportAsset src="/css/product.css" type="stylesheet" />
    <ImportAsset src="/js/product.js" type="script" defer />
    <SetMeta name="description" content="{{ product.summary | truncate(160) }}" />
    <SetMeta property="og:image" content="{{ product.image }}" />

    <Hoist target="structured-data">
      <script type="application/ld+json">
      {
        "@context": "https://schema.org",
        "@type": "Product",
        "name": "{{ product.name | escape }}"
      }
      </script>
    </Hoist>

    <article class="product">
      <h1>{{ product.name }}</h1>

      <Gallery images={product.images} />

      <div class="details">
        {{ product.description | safe }}
      </div>

      <p class="price">{{ product.price | round(2) }}</p>

      <AddToCart product={product} />
    </article>
  </Block>
</Extends>
```

### Example 6: Simple Partial (Component without Props)

**`partials/nav.grov`:**
```html
<nav class="main-nav">
  <a href="/">Home</a>
  <a href="/about">About</a>
  <a href="/blog">Blog</a>
</nav>
```

**Usage:**
```html
<Nav />
```

Components without `{% props %}` accept any passed attributes in permissive mode:

```html
<Nav activeSection="about" />
```

### Example 7: Scoped Slots (Renderless Data Provider)

**`components/data-list.grov`:**
```html
{% props items, sortBy="name" %}

{% set sorted = items | sort %}

<ul class="data-list">
  <For each={sorted} as="item">
    <li>
      <Slot item={item} index={loop.index} total={sorted | length} />
    </li>
  <Empty>
    <li class="empty">
      <Slot name="empty">No items.</Slot>
    </li>
  </For>
</ul>
```

**Usage — caller controls how each item is rendered:**
```html
<DataList items={users} sortBy="name" let:item let:index>
  <strong>{{ index }}.</strong> {{ item.name }} — {{ item.email }}
</DataList>

{# With custom empty state #}
<DataList items={results} let:item>
  <a href={item.url}>{{ item.title }}</a>

  <Fill slot="empty">
    <p>No results found. <a href="/search">Try again</a></p>
  </Fill>
</DataList>
```

### Example 8: Dynamic Components

```html
<Extends src="layouts/base">
  <Block name="content">
    <For each={sections} as="section">
      <Component is={section.component} {...section.props}>
        <If test={section.content}>
          {{ section.content | safe }}
        </If>
      </Component>
    </For>
  </Block>
</Extends>
```

### Example 9: Expression Attributes on HTML Elements

```html
<Define name="FormField">
  {% props name, label, type="text", required=false, error="", value="" %}

  <div class={error ? "field field--error" : "field"}>
    <label for={name}>
      {{ label }}
      <If test={required}>
        <span class="required">*</span>
      </If>
    </label>
    <input
      id={name}
      name={name}
      type={type}
      value={value}
      required={required}
      aria-invalid={error ? "true" : nil}
      aria-describedby={error ? name ~ "-error" : nil}
    >
    <If test={error}>
      <p id="{{ name }}-error" class="error">{{ error }}</p>
    </If>
  </div>
</Define>
```

### Example 10: Prop Shorthand and Spread

```html
{% set title = "Dashboard" %}
{% set variant = "primary" %}
{% set cardProps = { elevated: true, collapsible: true } %}

{# All equivalent #}
<Card title={title} variant={variant} elevated={true} collapsible={true} />
<Card {title} {variant} {...cardProps} />
```

---

## Appendix A: Migration Summary (v1 → v2)

| v1 Syntax | v2 Syntax | Change Type |
|-----------|-----------|-------------|
| `{{ expr }}` | `{{ expr }}` | Unchanged |
| `{# comment #}` | `{# comment #}` | Unchanged |
| `{% if c %}...{% elif %}...{% else %}...{% endif %}` | `<If test={c}>...<ElseIf test={}>...<Else>...</If>` | → Element |
| `{% for x in items %}...{% empty %}...{% endfor %}` | `<For each={items} as="x">...<Empty>...</For>` | → Element |
| `{% set x = expr %}` | `{% set x = expr %}` | Unchanged |
| `{% let %}...{% endlet %}` | `{% let %}...{% endlet %}` | Unchanged |
| `{% capture x %}...{% endcapture %}` | `<Capture name="x">...</Capture>` | → Element |
| `{% raw %}...{% endraw %}` | `<Verbatim>...</Verbatim>` | → Element (renamed) |
| `{% include "x" %}` | `<X />` | → Component (removed) |
| `{% include "x" key=val %}` | `<X key=val />` | → Component (removed) |
| `{% render "x" key=val %}` | `<X key=val />` | → Component (removed) |
| `{% extends "x" %}` + `{% block %}` | `<Extends src="x"><Block>...</Block></Extends>` | → Wrapping element |
| `{% block name %}...{% endblock %}` | `<Block name="name">...</Block>` | → Element |
| `{% macro f(args) %}...{% endmacro %}` | `<Define name="F">{% props args %}...</Define>` | → Element (replaced) |
| `{% call f(args) %}...{% endcall %}` | `<F>...</F>` (default slot) | → Component call |
| `{{ f(args) }}` (macro call) | `<F args />` or `<Capture>` + component | → Component |
| `{% import "x" as y %}` | *(removed — use file-based components)* | Removed |
| `caller()` | `<Slot />` (default slot) | Replaced |
| `{% component "x" prop=val %}...{% endcomponent %}` | `<X prop=val>...</X>` | → PascalCase element |
| `{% props ... %}` | `{% props ... %}` | Unchanged |
| `{% slot %}...{% endslot %}` | `<Slot>...</Slot>` or `<Slot />` | → Element |
| `{% fill "x" %}...{% endfill %}` | `<Fill slot="x">...</Fill>` | → Element |
| `{% asset ... %}` | `<ImportAsset src="..." ... />` | → Element (renamed) |
| `{% meta ... %}` | `<SetMeta ... />` | → Element (renamed) |
| `{% hoist "x" %}...{% endhoist %}` | `<Hoist target="x">...</Hoist>` | → Element |

### Removed Constructs

| v1 Construct | Replacement | Rationale |
|-------------|-------------|-----------|
| `{% include "x" %}` | `<X />` (component) | Components with permissive mode (no `{% props %}`) cover the include use case. Scope is always isolated — pass data explicitly via props. |
| `{% render "x" %}` | `<X />` (component) | Identical to component invocation. Was already isolated scope in v1. |
| `{% macro %}` | `<Define>` | Inline component definitions replace macros. Components support slots (more powerful than `caller()`), named props, and the same isolation. |
| `{% call %}` | Component with default slot | Passing a body to a macro is just providing default slot content to a component. |
| `{% import %}` | File-based components | Cross-file reuse is handled by component resolution. Namespace grouping is replaced by directory structure. |
| `caller()` | `<Slot />` | The default slot is the direct equivalent of `caller()`. |

---

## Appendix B: Grammar Changes (Lexer/Parser Impact)

### New Token Types Required

```
TK_ELEMENT_OPEN       <          (when followed by PascalCase identifier)
TK_ELEMENT_CLOSE      </         (when followed by PascalCase identifier)
TK_ELEMENT_END        >
TK_SELF_CLOSE         />
TK_EXPR_OPEN          {          (inside element attributes)
TK_EXPR_CLOSE         }          (inside element attributes)
TK_ATTR_ASSIGN        =          (inside element attributes)
TK_ELEMENT_NAME       PascalCase identifier (If, For, Card, etc.)
TK_SPREAD             {...       (spread operator in attributes)
TK_SHORTHAND          {name}     (prop shorthand in attributes)
TK_LET_BINDING        let:       (scoped slot prop binding prefix)
```

### Lexer State Machine Changes

The lexer needs new states for parsing HTML-style elements and HTML element attributes:

1. **Text state** (existing) — emit `TK_TEXT` until hitting `{{`, `{%`, `{#`, or `<` followed by PascalCase
2. **Element state** (new) — triggered by `<` + PascalCase name. Parses attributes as `name="value"`, `name={expr}`, `{name}` (shorthand), `{...expr}` (spread), or `let:name` pairs. Exits on `>` or `/>`
3. **Element expression state** (new) — inside `{...}` within an element attribute. Uses the existing expression tokenizer.
4. **HTML attribute state** (new) — triggered by `{` or `{{ }}` inside a lowercase HTML element's attributes. Allows expression attributes and interpolation on plain HTML elements while passing the element tag through as output.

### Parser Changes

The parser gains new entry points for element parsing:

- `parseElement()` — dispatches on element name to specific parsers (reserved names) or component parser
- `parseExtendsElement()` — parses `<Extends>` with `<Block>` children until `</Extends>`
- `parseIfElement()` — parses `<If>` with nested `<ElseIf>` / `<Else>` branches until `</If>`
- `parseForElement()` — parses `<For>` with optional `<Empty>` branch until `</For>`
- `parseBlockElement()` — parses `<Block>` content until `</Block>`
- `parseCaptureElement()` — parses `<Capture>` content until `</Capture>`
- `parseHoistElement()` — parses `<Hoist>` content until `</Hoist>`
- `parseVerbatimElement()` — collects raw text until `</Verbatim>`
- `parseDefineElement()` — parses `<Define>` body (component definition) until `</Define>`
- `parseSlotElement()` — parses `<Slot>` (self-closing or with fallback) until `</Slot>` or `/>` 
- `parseFillElement()` — parses `<Fill>` content until `</Fill>`
- `parseComponentElement()` — parses component with props and fill/slot children until `</Name>`
- `parseDynamicComponent()` — parses `<Component is={expr}>` with runtime name resolution
- `parseHTMLElement()` — parses lowercase HTML elements, evaluating expression attributes and interpolation

### AST Impact

The AST node types are simplified:

**Removed nodes:**
- `IncludeNode` — replaced by `ComponentNode`
- `RenderNode` — replaced by `ComponentNode`
- `MacroNode` — replaced by `DefineNode` (new)
- `CallNode` (call with caller body) — replaced by `ComponentNode` with default slot
- `ImportNode` — removed entirely
- `MacroCallExpr` — removed (no expression-level component calls)
- `FuncCallNode` — only `range()` and `super()` remain as built-in functions

**New nodes:**
- `DefineNode` — `Name string`, `Body []Node`, `Line` — inline component definition
- `DynamicComponentNode` — `NameExpr Node`, `Props`, `Fills`, `Line` — `<Component is={expr}>` runtime dispatch
- `SpreadNode` — `Expr Node`, `Line` — `{...expr}` spread in attribute list
- `HTMLElementNode` — `Tag string`, `Attrs []HTMLAttr`, `Body []Node`, `Line` — HTML element with expression attributes (only generated when element has `{expr}` attributes; otherwise emitted as raw text)

**Modified nodes:**
- `SlotNode` — now parsed from `<Slot>` element syntax; gains `Props []NamedArgNode` for scoped slot props
- `FillNode` — now parsed from `<Fill>` element syntax; gains `LetBindings []LetBinding` for scoped slot prop reception
- `ComponentNode` — gains `SpreadProps []SpreadNode` and `LetBindings` (for default slot scoped props)
- `ExtendsNode` — now wraps child `BlockNode` children instead of being a standalone declaration

### Removed Opcodes

- `OP_MACRO_DEF`, `OP_MACRO_DEF_PUSH`, `OP_CALL_MACRO_VAL`, `OP_CALL_MACRO_CALL`, `OP_CALL_CALLER`
- `OP_INCLUDE`, `OP_RENDER`, `OP_IMPORT`

### New Opcodes

- `OP_DEFINE` — registers an inline component definition in the current scope
- `OP_COMPONENT_DYNAMIC` — like `OP_COMPONENT` but resolves component name from stack at runtime
- `OP_SPREAD_PROPS` — pop map value, merge entries into component prop set
- `OP_HTML_ATTR` — pop expression value, conditionally emit HTML attribute (omit if falsy)

The compiler and VM are otherwise **unchanged** — `<Define>` components compile to the same bytecode structures as file-based components.
