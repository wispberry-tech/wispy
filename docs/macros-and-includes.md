# Macros & Includes

## include

Include a template inline. The included template shares the current scope:

```jinja2
{% include "partials/nav.grov" %}
```

Pass additional variables:

```jinja2
{% include "partials/nav.grov" section="about" active=true %}
```

The included template sees all variables from the current scope plus the explicitly passed ones.

## render

Like `include`, but with an isolated scope — only explicitly passed variables are visible:

```jinja2
{% render "partials/card.grov" title="Widget" price=9.99 %}
```

The rendered template cannot access the caller's variables. Use `render` when you want self-contained fragments that don't depend on page context.

## include vs render

| | `include` | `render` |
|--|-----------|----------|
| **Scope** | Shared — sees caller's variables | Isolated — only passed variables |
| **Use when** | Partial needs page context | Fragment should be self-contained |
| **Example** | Navigation bar that needs `current_page` | Email template snippet |

Both require a template store (`WithStore`).

## macro

Define reusable template functions:

```jinja2
{% macro user_card(name, role="member") %}
  <div class="card">
    <strong>{{ name }}</strong>
    <span class="role">{{ role }}</span>
  </div>
{% endmacro %}
```

Call a macro like a function:

```jinja2
{{ user_card("Alice", "admin") }}
{{ user_card("Bob") }}
```

Macros support positional and keyword arguments:

```jinja2
{% macro link(href, text, target="_self") %}
  <a href="{{ href }}" target="{{ target }}">{{ text }}</a>
{% endmacro %}

{{ link("https://example.com", "Example") }}
{{ link("https://example.com", "Example", target="_blank") }}
```

**Macros have isolated scope** — they cannot access variables from the surrounding template. Only the arguments passed to the macro are available inside it.

## call and caller()

Use `{% call %}` to pass a block of content to a macro:

```jinja2
{% macro card(title) %}
  <div class="card">
    <h2>{{ title }}</h2>
    <div class="body">
      {{ caller() }}
    </div>
  </div>
{% endmacro %}

{% call card("Orders") %}
  <p>You have 3 pending orders.</p>
{% endcall %}
```

Inside the macro, `{{ caller() }}` renders the content from the `{% call %}` block. `caller()` can be called multiple times.

## import

Import macros from another template file into a namespace:

```jinja2
{% import "macros/ui.grov" as ui %}

{{ ui.user_card("Alice") }}
{{ ui.link("https://example.com", "Click here") }}
```

`import` requires a template store. The imported template is executed, and any macros defined in it become available through the namespace.
