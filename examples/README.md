# Grove Template Engine — Examples

This directory contains four complete examples demonstrating Grove's capabilities:

## Quick Start

Run any example locally with Go:

```bash
go run ./examples/blog/     # Start Meridian tech blog
go run ./examples/store/    # Start Coldfront Supply Co. shop
go run ./examples/email/    # Start email template preview server
go run ./examples/docs/     # Start Grove documentation site
```

Each example runs on its own port (look for "listening on" in the output).

---

## Example Breakdown

### 📝 Blog — *Meridian* (`blog/`)

A **professional tech publication** with article management, tagging, author bios, and deep reading experiences.

**Demonstrates:**
- Component composition (base layouts, cards, author profiles)
- Per-component CSS and JS co-located with `.grov` templates
- Conditional rendering (`{% if %}`) and loops (`{% each %}`)
- Template inheritance via slots (`{% slot %}` / `{% #fill %}`)
- Semantic HTML and accessibility patterns
- Advanced CSS patterns (drop caps, serif typography)
- Mobile-responsive design with hamburger navigation

**Design approach:** Editorial sophistication — Georgia serif body text, generous whitespace, dark header/footer with cream backgrounds.

---

### 🏪 Store — *Coldfront Supply Co.* (`store/`)

A **premium outdoor equipment shop** with product catalogs, filtering, sorting, cart management, and checkout.

**Demonstrates:**
- Reusable component system (extracted Nav, Footer, ProductCard)
- Complex data structures (products, categories, cart items)
- Form handling and JavaScript interop (sort dropdown with query param preservation)
- Captured variables via `{% #let %}` blocks
- Custom filter registration (`currency` filter in Go)
- Grid layouts at multiple breakpoints
- Loading states and accessibility in interactive components

**Design approach:** Functional clarity — white product cards on cream pages, consistent spacing scale, button hierarchy (primary/secondary/ghost).

---

### ✉️ Email — *Grove Cloud* (`email/`)

**Transactional email templates** for order confirmations, password resets, plan changes, usage alerts, and account welcome.

**Demonstrates:**
- Email-safe HTML patterns (table-based layouts, inline styles)
- Component helpers for common patterns (buttons, dividers, spacers, usage bars)
- Preheader optimization via `{% #hoist %}`
- Captured blocks for reusable email content
- Variable scope and conditional rendering for user-specific messaging
- Cross-example integration (order confirmations reference Coldfront Supply Co.)

**Design approach:** Professional SaaS emails — green header accent, clear information hierarchy, MSO-safe HTML for Outlook compatibility.

---

### 📚 Docs — *Grove Documentation* (`docs/`)

**Developer documentation** for Grove's template syntax, built-in filters, and architecture.

**Demonstrates:**
- Sidebar navigation with active-page highlighting
- Deep component nesting (Base → DocsLayout → page templates)
- Code examples and admonition blocks (Note/Warning/Tip)
- Search and category filtering
- Landing page with quick-start section
- Accessibility: breadcrumbs, skip-to-content link, proper nav semantics

**Design approach:** Technical simplicity — minimal decoration, readable monospace code, dark sidebar for navigation hierarchy.

---

## Design System

All examples share a **unified design foundation** (`_shared/tokens.css`):

- **Colors:** Primary green (#2E6740), dark navy, cream page background, semantic alerts
- **Spacing:** 4px base unit (--space-1 through --space-16)
- **Typography:** System sans-serif for UI, serif for editorial, monospace for code
- **Components:** Buttons, cards, forms, navigation, tables with consistent styling

Each example extends the shared tokens with its own personality:
- **Meridian** adds Georgia serif, drop-cap styling, featured content
- **Coldfront** adds product grids, sticky sidebars, cart tables  
- **Grove Cloud** adds email-safe tables, usage bars, transactional affordances
- **Docs** adds sidebar highlight, code block headers, admonitions

---

## What You'll Learn

Working through these examples, you'll master:

✅ **Component Architecture:** How to build reusable components with proper prop passing  
✅ **Layout Patterns:** Base layouts, slots, nested components, sidebar+main patterns  
✅ **Control Flow:** Loops, conditionals, empty states, ternary expressions  
✅ **Data Binding:** Passing data through component trees, scope handling  
✅ **Interop:** Capturing template output, registering custom filters, hoisting content  
✅ **Accessibility:** Semantic HTML, ARIA labels, keyboard navigation, skip links  
✅ **Responsive Design:** Mobile-first CSS, grid layouts, breakpoint management  
✅ **Production Patterns:** Error handling, loading states, form submission  

---

## File Structure

```
examples/
├── _shared/
│   └── tokens.css              # Shared design tokens (imported by all)
├── blog/
│   ├── main.go                 # Server & data fixtures
│   ├── static/
│   │   ├── base.css            # Global resets, layout, utilities
│   │   └── tokens.css          # Copy of shared tokens
│   └── templates/
│       ├── base.grov           # Main layout — declares base.css
│       ├── index.grov          # Homepage
│       ├── post.grov           # Single article
│       ├── composites/
│       │   ├── nav/
│       │   │   ├── nav.grov    # Navigation component
│       │   │   ├── nav.css     # Nav styles (co-located)
│       │   │   └── nav.js      # Mobile menu toggle
│       │   └── card/
│       │       ├── card.grov   # Post card component
│       │       └── card.css    # Card styles (co-located)
│       └── primitives/
│           └── button/
│               ├── button.grov # Button component
│               ├── button.css  # Button styles (co-located)
│               └── button.js   # Loading state behavior
├── store/
├── email/
├── docs/
└── README.md                   # This file
```

Each component owns its CSS and JS files. Components declare their dependencies via `{% asset %}`, which bubble up through composition and are deduplicated in `RenderResult`. Global styles live in `static/base.css` and load first (via `priority=10`).

---

## Running in Development

For **live reloading** during development, use [entr](https://github.com/eradman/entr) or similar:

```bash
ls examples/blog/templates/*.grov | entr go run ./examples/blog/
```

Each example includes sample data in `main.go` (fixture data for products, posts, etc.). Edit these files to customize the demo content.

---

## Learn More

- [Grove Repository](https://github.com/wispberry-tech/grove)
- Grove uses the sigil-based syntax: `{% ... %}` for directives, `{% variable %}` for output
- All templates are compiled to bytecode and executed on Grove's stack-based VM
- Performance: templates render in microseconds, support concurrent rendering
