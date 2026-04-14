# Layouts

Grove uses component composition for layouts — there is no separate template inheritance system. A layout is simply a component with named slots.

See [Components — Layouts via Components](components.md#layouts-via-components) for the full documentation.

## Quick Example

Define a layout as a component:

```html
{# base.grov #}
<!DOCTYPE html>
<html>
<head>
  <title>{% #slot "title" %}My Site{% /slot %}</title>
</head>
<body>
  <main>{% slot "content" %}</main>
  <footer>{% #slot "footer" %}&copy; 2026{% /slot %}</footer>
</body>
</html>
```

Pages import and fill slots:

```html
{# home.grov #}
{% import Base from "base" %}
<Base>
  {% #fill "title" %}Home — My Site{% /fill %}
  {% #fill "content" %}
    <h1>Welcome</h1>
  {% /fill %}
</Base>
```

## Multi-Level Layouts

Layouts can compose other layouts:

```html
{# section.grov #}
{% import Base from "base" %}
<Base>
  {% #fill "content" %}
    <div class="section">
      {% #slot "inner" %}section default{% /slot %}
    </div>
  {% /fill %}
</Base>
```

```html
{# page.grov #}
{% import Section from "section" %}
<Section>
  {% #fill "inner" %}page content{% /fill %}
</Section>
```

Rendering `page.html` produces:

```html
<!DOCTYPE html>
<html>
<head>
  <title>My Site</title>
</head>
<body>
  <main><div class="section">
    page content
  </div></main>
  <footer>&copy; 2026</footer>
</body>
</html>
```
