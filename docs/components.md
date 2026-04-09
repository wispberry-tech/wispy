# Components

Components are reusable templates with a declared interface. They accept data through **props** and allow callers to inject content through **slots**. In Grove, components replace macros, includes, and template inheritance — one composition model for everything.

## Defining a Component

Wrap a template in `<Component>` to define a named, reusable unit:

```html
{# button.grov #}
<Component name="Button" label href="/" variant="primary">
  <a href="{% href %}" class="btn btn-{% variant %}">{% label %}</a>
</Component>
```

- `name` is required — it's the name callers use after importing
- Props are declared as attributes: bare names are required (`label`), names with values have defaults (`href="/"`)
- The component body is the template rendered when the component is called

### Props

```html
<Component name="Card" title summary>
  <article>
    <h2>{% title %}</h2>
    <p>{% summary %}</p>
  </article>
</Component>
```

- Props without defaults (like `title`, `summary`) are required — omitting them causes a `RuntimeError`
- Props with defaults (like `variant="primary"`) are optional
- Passing an unknown prop causes a `RuntimeError`
- Components have **isolated scope** — they cannot see the caller's variables, only their declared props

## Importing Components

Use `{% import %}` to bring components into scope before using them:

```html
{# page.grov #}
{% import Button from "button" %}

<Button label="Click me" href="/action" />
```

- The path is the template name **without** the `.grov` extension
- The name after `import` must match the `name` attribute on `<Component>` in that file

### Import variants

**Multiple components from one file:**

```html
{% import Card, Badge, Button from "ui" %}
```

**Wildcard — import all components:**

```html
{% import * from "ui" %}
```

**Alias — rename locally:**

```html
{% import Card as InfoCard from "cards" %}
<InfoCard title="Details" />
```

**Namespaced wildcard:**

```html
{% import * as UI from "ui" %}
<UI.Card title="X" />
<UI.Badge label="Y" />
```

### Multi-component files

A single file can define multiple components:

```html
{# ui.grov #}
<Component name="Card" title>
  <div class="card">{% title %}</div>
</Component>

<Component name="Badge" label>
  <span class="badge">{% label %}</span>
</Component>

<Component name="Button" text>
  <button>{% text %}</button>
</Component>
```

## Slots

Slots let callers inject content into specific points of a component.

### Default slot

```html
{# box.grov #}
<Component name="Box">
  <div class="box">
    {% #slot %}No content provided{% /slot %}
  </div>
</Component>
```

```html
{# Using it: #}
{% import Box from "box" %}
<Box>
  <p>This replaces "No content provided"</p>
</Box>
```

The content inside `{% #slot %}...{% /slot %}` is fallback content, rendered when the caller doesn't provide any.

### Named slots

Components can define multiple injection points:

```html
{# card.grov #}
<Component name="Card" title summary>
  <article>
    <h2>{% title %}</h2>
    <p>{% summary %}</p>
    <div class="tags">
      {% slot "tags" %}
    </div>
    <div class="actions">
      {% #slot "actions" %}<a href="#">Read more</a>{% /slot %}
    </div>
  </article>
</Component>
```

Callers fill named slots with `{% #fill %}`:

```html
{% import Card from "card" %}
<Card title="My Post" summary="A summary">
  {% #fill "tags" %}
    <span class="tag">Go</span>
    <span class="tag">Templates</span>
  {% /fill %}
  {% #fill "actions" %}
    <a href="/post/1">Read</a>
    <a href="/post/1/edit">Edit</a>
  {% /fill %}
</Card>
```

Unfilled named slots render their fallback content.

### Scoped slots

Slots can pass data back to the caller using `data={expr}`:

```html
{# list.grov #}
<Component name="List" items>
  <ul>
    {% #each items as item %}
      <li>{% slot "item" data={item} %}</li>
    {% /each %}
  </ul>
</Component>
```

The caller accesses the slot data with `let:data`:

```html
{% import List from "list" %}
<List items={users}>
  {% #fill "item" let:data %}
    <strong>{% data.name %}</strong>
  {% /fill %}
</List>
```

## Scope Rules

- **Props** are available inside the component template. The component cannot see the caller's variables.
- **Fills see the caller's scope**, not the component's. This means you can use your page data inside a `{% #fill %}` block without threading it through props.

```html
{# page.grov — caller's scope has "posts" #}
{% import Card from "card" %}
<Card title="Recent" summary="Latest posts">
  {% #fill "tags" %}
    {# This sees "posts" from the page, not from the card component #}
    {% #each posts as post %}
      <span>{% post.title %}</span>
    {% /each %}
  {% /fill %}
</Card>
```

## Layouts via Components

Template inheritance (`extends`/`block`) is replaced by component composition. Define a layout as a component with named slots:

```html
{# base.grov #}
<Component name="Base">
  <!DOCTYPE html>
  <html>
  <head>
    <title>{% #slot "title" %}My Site{% /slot %}</title>
  </head>
  <body>
    <nav>...</nav>
    <main>{% slot "content" %}</main>
    <footer>{% #slot "footer" %}&copy; 2026 My Site{% /slot %}</footer>
  </body>
  </html>
</Component>
```

Pages import and fill the layout slots:

```html
{# home.grov #}
{% import Base from "base" %}
<Base>
  {% #fill "title" %}Home — My Site{% /fill %}
  {% #fill "content" %}
    <h1>Welcome</h1>
    <p>This fills the content slot.</p>
  {% /fill %}
</Base>
```

## Nesting Components

Components can use other components:

```html
{# post-list.grov #}
<Component name="PostList" posts>
  {% import Card from "card" %}
  {% import TagBadge from "primitives/tag-badge" %}
  {% #each posts as post %}
    <Card title={post.title} summary={post.summary}>
      {% #fill "tags" %}
        {% #each post.tags as tag %}
          <TagBadge label={tag.name} color={tag.color} />
        {% /each %}
      {% /fill %}
    </Card>
  {% /each %}
</Component>
```

## Dynamic Components

Render a component whose name is determined at runtime:

```html
{% import * from "icons" %}
<Component is={icon_name} size="lg" />
```

The `is` attribute accepts an expression that resolves to a component name from the current import scope.

## Component Architecture

### Primitives

Leaf components with no child components. They accept props and render self-contained HTML.

Examples: buttons, badges, icons, inputs.

### Composites

Components that compose other components and/or use slots for flexible content injection.

Examples: cards, navigation bars, post lists.

### Folder Structure

```
templates/
  primitives/
    button/button.grov
    tag-badge/tag-badge.grov
  composites/
    card/card.grov
    nav/nav.grov
  layouts/
    base.grov
    docs.grov
```

### Path Resolution

`FileSystemStore` resolves component paths in this order:

1. **Exact match** — `composites/card` (file exists as-is)
2. **Append .grov** — `composites/card.grov` (flat file)
3. **Directory fallback** — `composites/card/card.grov` (folder-per-component)
