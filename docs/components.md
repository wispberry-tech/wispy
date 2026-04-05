# Components

Components are reusable templates with a declared interface. They accept data through **props** and allow callers to inject content through **slots**.

## Component Architecture

Grove uses a two-tier classification for components:

### Primitives

Leaf components with no child components. They accept props and render self-contained HTML. Primitives do not use `{% component %}` internally and do not define `{% slot %}` tags.

Examples: buttons, badges, icons, inputs.

### Composites

Components that compose other components and/or use slots for flexible content injection. A composite uses `{% component %}` inside its template, defines `{% slot %}` tags, or both.

Examples: cards, navigation bars, author profiles.

**Decision rule:** If a component uses `{% component %}` or has `{% slot %}` tags, it's a composite. Otherwise it's a primitive.

### Folder Structure

Organize components into `primitives/` and `composites/` directories, with each component in its own folder:

```
templates/
  primitives/
    button/
      button.grov
      button.js          ← optional JS for progressive enhancement
    tag-badge/
      tag-badge.grov
  composites/
    card/
      card.grov
    nav/
      nav.grov
      nav.js
```

### Path Resolution

When referencing components, use the short path without repeating the filename:

```jinja2
{% component "composites/card" %}
  ...
{% endcomponent %}
```

`FileSystemStore` resolves component paths in this order:

1. **Exact match** — `composites/card` (file exists as-is)
2. **Append .grov** — `composites/card.grov` (flat file without directory)
3. **Directory fallback** — `composites/card/card.grov` (folder-per-component)

This means flat-file components (without a directory) still work. The fallback only applies to names that don't already end in `.grov`.

## JS Colocation

Components can include a colocated JavaScript file for progressive enhancement. The JS file lives next to the `.grov` file with the same base name:

```
button/
  button.grov
  button.js
```

The JS file is always optional. Components without a `.js` file are perfectly valid.

### Including JS

Components declare their JS dependency using the `{% asset %}` tag:

```jinja2
{# primitives/button/button.grov #}
{% props label, href="#" %}
{% asset "/js/primitives/button/button.js" type="script" %}

<a href="{{ href }}" class="btn" data-button>{{ label }}</a>
```

The `RenderResult` asset system handles deduplication — if a page renders 10 buttons, the script is included once. The Go server serves JS files statically from the component directories.

### Progressive Enhancement

Server-rendered HTML must be functional without JavaScript. The JS enhances existing markup with smoother interactions, animations, or keyboard navigation. If JS fails to load, the component still works.

### Binding Convention

Components use `data-*` attributes to connect JS to markup. The template renders a `data-*` attribute, and the JS file queries for those attributes:

```js
// button.js
document.querySelectorAll('[data-button]').forEach(btn => {
  btn.addEventListener('click', () => {
    btn.classList.add('btn-loading');
    btn.setAttribute('aria-busy', 'true');
  });
});
```

No IDs, no class-name coupling — `data-*` attributes are the contract between template and script.

## Using a Component

```jinja2
{% component "components/card.grov" title="Hello" summary="A card" %}
  <p>This goes into the default slot.</p>
{% endcomponent %}
```

The first argument is the template path (loaded from the store). Remaining arguments are space-separated `key=value` props passed to the component.

`component` requires a template store — it does not work with inline `RenderTemplate`.

## Defining Props

Declare accepted props at the top of a component template with `{% props %}`:

```jinja2
{# components/button.grov #}
{% props label, href="/", variant="primary" %}

<a href="{{ href }}" class="btn btn-{{ variant }}">{{ label }}</a>
```

- Props with a default value (like `href` and `variant`) are optional
- Props without a default (like `label`) are required — passing no value causes a `RuntimeError`
- Passing an unknown prop causes a `RuntimeError`
- If a component has no `{% props %}` declaration, it accepts any props without restriction

## Default Slot

Content between `{% component %}` and `{% endcomponent %}` fills the default slot:

```jinja2
{# components/box.grov #}
<div class="box">
  {% slot %}No content provided{% endslot %}
</div>
```

```jinja2
{# Using it: #}
{% component "components/box.grov" %}
  <p>This replaces "No content provided"</p>
{% endcomponent %}
```

The text inside `{% slot %}...{% endslot %}` is fallback content, rendered when the caller doesn't provide any.

## Named Slots

Components can define multiple injection points with named slots:

```jinja2
{# components/card.grov #}
{% props title, summary %}

<article>
  <h2>{{ title }}</h2>
  <p>{{ summary }}</p>
  <div class="tags">
    {% slot "tags" %}{% endslot %}
  </div>
  <div class="actions">
    {% slot "actions" %}<a href="#">Read more</a>{% endslot %}
  </div>
</article>
```

Callers fill named slots with `{% fill %}`:

```jinja2
{% component "components/card.grov" title="My Post" summary="A summary" %}
  {% fill "tags" %}
    <span class="tag">Go</span>
    <span class="tag">Templates</span>
  {% endfill %}
  {% fill "actions" %}
    <a href="/post/1">Read</a>
    <a href="/post/1/edit">Edit</a>
  {% endfill %}
{% endcomponent %}
```

Unfilled named slots render their fallback content.

## Scope Rules

This is the key design decision in Grove's component system:

- **Props** are available inside the component template. The component cannot see the caller's variables.
- **Fills see the caller's scope**, not the component's. This means you can use your page data inside a `{% fill %}` block without threading it through props.

```jinja2
{# page.grov — caller's scope has "posts" #}
{% component "components/card.grov" title="Recent" summary="Latest posts" %}
  {% fill "tags" %}
    {# This sees "posts" from the page, not from the card component #}
    {% for post in posts %}
      <span>{{ post.title }}</span>
    {% endfor %}
  {% endfill %}
{% endcomponent %}
```

## Nesting Components

Components can use other components:

```jinja2
{# components/post-list.grov #}
{% props posts %}
{% for post in posts %}
  {% component "components/card.grov" title=post.title summary=post.summary %}
    {% fill "tags" %}
      {% for tag in post.tags %}
        {% component "components/tag.grov" label=tag.name color=tag.color %}{% endcomponent %}
      {% endfor %}
    {% endfill %}
  {% endcomponent %}
{% endfor %}
```

Components can also use template inheritance (`{% extends %}`).
