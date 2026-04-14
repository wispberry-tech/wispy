# Components Replace Macros & Includes

In Grove's HTML-centric syntax, macros, includes, and imports are all replaced by the unified `<Component>` and `<Import>` system. See [Components](components.md) for the full documentation.

## Migration from Legacy Syntax

### Macros → Components

**Before (legacy):**
```
{% macro user_card(name, role="member") %}
  <div class="card">
    <strong>{{ name }}</strong>
    <span class="role">{{ role }}</span>
  </div>
{% endmacro %}

{{ user_card("Alice", "admin") }}
```

**After:**
```html
{# user-card.grov #}
<div class="card">
  <strong>{% name %}</strong>
  <span class="role">{% role %}</span>
</div>

{# page.grov #}
{% import UserCard from "user-card" %}
<UserCard name="Alice" role="admin" />
```

### Includes → Import + Component

**Before (legacy):**
```
{% include "partials/nav.grov" %}
{% render "partials/card.grov" title="Widget" %}
```

**After:**
```html
{% import Nav from "partials/nav" %}
{% import Card from "partials/card" %}

<Nav />
<Card title="Widget" />
```

All components have isolated scope — there is no shared-scope include. Pass data explicitly via props.

### call/caller → Slots

**Before (legacy):**
```
{% macro card(title) %}
  <div class="card">
    <h2>{{ title }}</h2>
    {{ caller() }}
  </div>
{% endmacro %}

{% call card("Orders") %}
  <p>3 pending orders</p>
{% endcall %}
```

**After:**
```html
{# card.grov #}
<div class="card">
  <h2>{% title %}</h2>
  {% slot %}
</div>

{# page.grov #}
{% import Card from "card" %}
<Card title="Orders">
  <p>3 pending orders</p>
</Card>
```
