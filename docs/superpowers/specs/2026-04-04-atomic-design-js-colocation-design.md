# Atomic Design & JS Colocation for Grove Components

**Date:** 2026-04-04
**Status:** Draft
**Scope:** Engine change, documentation update, example refactoring

## Overview

Establish a two-tier atomic design convention for Grove components (primitives and composites), introduce a folder-per-component structure with colocated JavaScript for progressive enhancement, and update the engine's component path resolution to support clean paths.

## Component Architecture

### Two-Tier Classification

- **Primitives** — Leaf components with no child components. Accept props, render self-contained HTML. They do not use `{% component %}` internally and do not define `{% slot %}` tags.
- **Composites** — Compose primitives and/or use slots for flexible content injection. They use `{% component %}` internally or define `{% slot %}` tags (or both).

**Decision rule:** If a component uses `{% component %}` inside its template or has `{% slot %}` tags, it's a composite. Otherwise it's a primitive.

### Folder-Per-Component Structure

```
templates/
  primitives/
    tag-badge/
      tag-badge.grov
    button/
      button.grov
      button.js
  composites/
    card/
      card.grov
      card.js
    nav/
      nav.grov
      nav.js
    author-card/
      author-card.grov
```

Components without JS simply omit the `.js` file — the JS file is always optional.

## Engine Change: Component Path Resolution

Update the `FileSystemStore.Load()` method in `internal/store/filesystem.go` to support a two-step fallback:

1. `{% component "composites/card" %}` → first try `composites/card.grov`
2. If not found → try `composites/card/card.grov`

This allows clean paths like `{% component "composites/card" %}` without repeating the component name, while keeping flat-file components working for simple cases that don't need a directory.

The resolution applies to the `component` tag only. Other includes (`include`, `render`, `import`, `extends`) are unaffected.

## JS Colocation & Progressive Enhancement

### File Placement

A component's JS file lives next to its `.grov` file with the same base name:

```
button/
  button.grov
  button.js
```

### How JS Gets Included

- Components declare their JS dependency using `{% asset "js" "primitives/button/button.js" %}`
- The `RenderResult` asset system handles deduplication — rendering 10 buttons includes the script once
- The Go server serves JS files statically from the component directories

### Progressive Enhancement Philosophy

- Server-rendered HTML must be functional without JS
- JS enhances existing markup (animations, keyboard navigation, smoother interactions)
- If JS fails to load, the component still works

### Binding Convention

Components use `data-*` attributes to connect JS to markup. The `.grov` template renders a `data-*` attribute (e.g., `data-dropdown`), and the JS file queries for those attributes. No IDs, no class-name coupling.

Example:

```html
<!-- dropdown.grov -->
<details data-dropdown>
  <summary>{{ label }}</summary>
  <div>{% slot %}{% endslot %}</div>
</details>
```

```js
// dropdown.js
document.querySelectorAll('[data-dropdown]').forEach(el => {
  // enhance with smooth animations, keyboard nav, etc.
});
```

## Documentation Updates

Expand `docs/components.md` with two new sections inserted before the existing syntax reference:

1. **Component Architecture** — Primitives vs composites definitions, folder structure convention, decision rule for classification, path resolution behavior
2. **JS Colocation** — File placement, `{% asset %}` usage, progressive enhancement philosophy, `data-*` binding convention

Existing syntax content (props, slots, fills, scope rules) stays unchanged.

## Example Refactoring

### Blog Example

Current flat structure under `templates/components/` gets restructured:

**Primitives:**
- `primitives/tag-badge/tag-badge.grov` — self-contained, no slots, no child components
- `primitives/button/button.grov` + `button.js` — new component, demonstrates JS colocation

**Composites:**
- `composites/card/card.grov` — has "tags" slot, composes tag-badge
- `composites/nav/nav.grov` + `nav.js` — JS for mobile menu toggle (enhances `<details>` fallback)
- `composites/footer/footer.grov` — stays as-is structurally
- `composites/author-card/author-card.grov` — has default slot
- `composites/breadcrumbs/breadcrumbs.grov` — stays as-is structurally

**Path updates:** All `{% component %}` references updated to use new paths:
```
{% component "composites/card" %}
{% component "primitives/tag-badge" %}
```

### Store Example

**Composites:**
- `composites/product-card/product-card.grov` — moved from flat structure

**Primitives:**
- `primitives/button/button.grov` + `button.js` — new, same progressive enhancement pattern as blog

**Path updates:** All `{% component %}` references updated.

### New JS Files

Two JS files demonstrate progressive enhancement:
- `button.js` — loading state on click (disables button, shows spinner)
- `nav.js` — mobile hamburger menu toggle, enhances a `<details>`-based fallback

## Out of Scope

- Build step or bundling for JS files
- Web components or custom element registration
- Changes to `include`, `render`, `import`, or `extends` path resolution
- Docs example and email example refactoring (neither uses components)
- New example applications
