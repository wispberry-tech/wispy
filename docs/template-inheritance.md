# Template Inheritance

Template inheritance lets you define a base layout and override specific sections in child templates.

## Base Template

A base template defines the page structure with `{% block %}` override points:

```jinja2
{# base.grov #}
<!DOCTYPE html>
<html>
<head>
  <title>{% block title %}My Site{% endblock %}</title>
</head>
<body>
  <nav>...</nav>
  <main>
    {% block content %}{% endblock %}
  </main>
  <footer>
    {% block footer %}© 2026 My Site{% endblock %}
  </footer>
</body>
</html>
```

Blocks can have default content (like `title` and `footer` above) or be empty (like `content`). A base template renders on its own — blocks use their default content when not overridden.

## Child Template

A child template extends a parent with `{% extends %}` and overrides specific blocks:

```jinja2
{# home.grov #}
{% extends "base.grov" %}

{% block title %}Home — My Site{% endblock %}

{% block content %}
  <h1>Welcome</h1>
  <p>This replaces the content block.</p>
{% endblock %}
```

**Rules:**
- `{% extends %}` must be the first tag in the template
- Only `{% block %}` tags in the child are used — any content outside blocks is discarded
- Blocks not overridden keep the parent's default content
- `extends` requires a template store (`WithStore`) — it does not work with inline `RenderTemplate`

## super()

Include the parent block's content using `{{ super() }}`:

```jinja2
{# home.grov #}
{% extends "base.grov" %}

{% block title %}Home — {{ super() }}{% endblock %}
```

If the base template's `title` block contains `My Site`, this renders: `Home — My Site`.

## Multi-Level Inheritance

Inheritance chains to any depth. Each level can override blocks and call `super()`:

```jinja2
{# base.grov #}
<html>
<body>
  {% block content %}base{% endblock %}
</body>
</html>
```

```jinja2
{# section.grov #}
{% extends "base.grov" %}

{% block content %}
  <div class="section">
    {% block inner %}section default{% endblock %}
  </div>
{% endblock %}
```

```jinja2
{# page.grov #}
{% extends "section.grov" %}

{% block inner %}page content{% endblock %}
```

Rendering `page.grov` produces:

```html
<html>
<body>
  <div class="section">
    page content
  </div>
</body>
</html>
```

### super() chains

Each `super()` call reaches one level up. In a three-level chain:

```jinja2
{# base.grov #}
{% block title %}Base{% endblock %}

{# mid.grov #}
{% extends "base.grov" %}
{% block title %}Mid:{{ super() }}{% endblock %}

{# leaf.grov #}
{% extends "mid.grov" %}
{% block title %}Leaf:{{ super() }}{% endblock %}
```

Rendering `leaf.grov` produces: `Leaf:Mid:Base`.
