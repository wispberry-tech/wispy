# Plan 8: Editor Plugin Support for Wispy Grove

## Motivation

Without editor support, developers writing Wispy Grove templates get no syntax highlighting — the `{% %}`, `{# #}`, `<PascalCase>`, and `{expr}` constructs render as plain text inside HTML files. This makes templates harder to read and write. A dedicated language grammar gives developers keyword highlighting, delimiter coloring, component recognition, and proper scope recognition across VS Code (immediate), Neovim, and Zed (future).

---

## Strategy

**Two grammars, three editors:**

| Editor | Engine | Grammar Format | Priority |
|--------|--------|---------------|----------|
| VS Code | TextMate | `.tmLanguage.json` | Phase 1 (immediate) |
| Neovim | Tree-sitter | `grammar.js` + `highlights.scm` | Phase 2 |
| Zed | Tree-sitter | Same `grammar.js` + `highlights.scm` | Phase 2 |

Phase 1 ships a VS Code extension using TextMate grammars (regex-based, quick to write). Phase 2 adds a Tree-sitter parser that powers both Neovim and Zed with a single grammar.

---

## Repository Layout

```
wispy-grove-editor/
├── vscode/
│   ├── package.json
│   ├── language-configuration.json
│   └── syntaxes/
│       └── grove.tmLanguage.json
├── tree-sitter/                  # Phase 2
│   ├── grammar.js
│   ├── src/
│   └── queries/
│       └── highlights.scm
├── neovim/                       # Phase 2 (nvim-treesitter wrapper)
├── zed/                          # Phase 2 (extension wrapper)
├── LICENSE
└── README.md
```

---

## Phase 1: VS Code Extension

### File Association

- Language ID: `grov`
- File extensions: `.grov`
- Base language: HTML (`text.html.basic`)

### Scope Name

`text.html.grov` — extends the built-in HTML grammar via injection.

### Grammar Structure

The TextMate grammar uses **injections** to layer Grove syntax on top of `text.html.basic`. This means all standard HTML highlighting works unchanged, and Grove constructs are highlighted on top.

Grove uses a **unified delimiter** `{% %}` for all server-side operations (output, control flow, assignment, imports, etc.) and `{# #}` for comments. Components use `<PascalCase>` HTML-like elements, and dynamic attribute expressions use `{expression}` syntax.

#### Delimiter Scopes

| Syntax | TextMate Scope | Description |
|--------|---------------|-------------|
| `{%` / `%}` | `punctuation.section.embedded.begin/end.grov` | Tag delimiters (all server operations) |
| `{#` / `#}` | `punctuation.definition.comment.begin/end.grov` | Comment delimiters |
| `-` (in `{%-`, `-%}`) | `keyword.operator.whitespace.grov` | Whitespace strip marker |

#### Sigil Scopes

Sigils appear as the first character after `{%` and determine the tag type:

| Syntax | TextMate Scope | Description |
|--------|---------------|-------------|
| `#` (in `{% #if %}`, `{% #each %}`) | `keyword.operator.sigil.open.grov` | Block-open sigil |
| `:` (in `{% :else %}`, `{% :empty %}`) | `keyword.operator.sigil.branch.grov` | Branch-separator sigil |
| `/` (in `{% /if %}`, `{% /each %}`) | `keyword.operator.sigil.close.grov` | Block-close sigil |

#### Keyword Scopes

| Keywords | TextMate Scope |
|----------|---------------|
| `if`, `else`, `else if` | `keyword.control.conditional.grov` |
| `each`, `as`, `in`, `empty` | `keyword.control.loop.grov` |
| `set`, `let` | `keyword.control.assignment.grov` |
| `import`, `from` | `keyword.control.import.grov` |
| `slot`, `fill` | `keyword.control.slot.grov` |
| `capture` | `keyword.control.capture.grov` |
| `asset`, `meta`, `hoist` | `keyword.control.web.grov` |
| `verbatim` | `keyword.control.verbatim.grov` |
| `and`, `or`, `not` | `keyword.operator.logical.grov` |
| `true`, `false` | `constant.language.boolean.grov` |
| `nil`, `null` | `constant.language.null.grov` |

#### Component Scopes

| Syntax | TextMate Scope | Description |
|--------|---------------|-------------|
| `<Component` / `</Component>` | `entity.name.tag.definition.grov` | Component definition element |
| `<PascalCase` (e.g. `<Card`, `<Base>`) | `entity.name.tag.component.grov` | Component invocation element |
| `</PascalCase>` (e.g. `</Card>`) | `entity.name.tag.component.grov` | Component closing tag |
| `name=` (on `<Component>`) | `entity.other.attribute-name.component.grov` | Component name attribute |

**Detection rule:** Any HTML-like tag whose name starts with an uppercase letter (`[A-Z]`) is treated as a Grove component and highlighted with component scopes rather than standard HTML tag scopes.

#### Attribute Expression Scopes

| Syntax | TextMate Scope | Description |
|--------|---------------|-------------|
| `{` / `}` (in attribute values) | `punctuation.section.embedded.begin/end.attribute.grov` | Expression delimiters in attributes |
| Expression inside `{...}` | Same as expression scopes below | Dynamic attribute value |

Attribute expressions use single braces: `title={post.title}`, `elevated={isActive}`. These appear only inside HTML/component attribute positions — not in body text.

#### Expression Scopes

| Syntax | TextMate Scope |
|--------|---------------|
| `"string"`, `'string'` | `string.quoted.double/single.grov` |
| `123`, `1.23` | `constant.numeric.grov` |
| `\|` (pipe) | `keyword.operator.filter.grov` |
| Filter name (after `\|`) | `support.function.filter.grov` |
| `~` | `keyword.operator.concatenation.grov` |
| `+`, `-`, `*`, `/`, `%` | `keyword.operator.arithmetic.grov` |
| `==`, `!=`, `<`, `<=`, `>`, `>=` | `keyword.operator.comparison.grov` |
| `=` (in assignment / named args) | `keyword.operator.assignment.grov` |
| `?`, `:` (ternary) | `keyword.operator.ternary.grov` |
| `.` (attribute access) | `punctuation.accessor.grov` |
| `[`, `]` (index access) | `punctuation.bracket.square.grov` |
| Identifiers | `variable.other.grov` |
| Built-in functions (`range()`) | `support.function.builtin.grov` |

#### Comment Scopes

| Syntax | TextMate Scope |
|--------|---------------|
| `{# ... #}` (entire block) | `comment.block.grov` |

#### Verbatim Block

Inside `{% #verbatim %} ... {% /verbatim %}`, everything is treated as plain text — no Grove patterns match. The verbatim tags themselves are highlighted as keywords.

### Language Configuration (`language-configuration.json`)

```json
{
  "comments": {
    "blockComment": ["{#", "#}"]
  },
  "brackets": [
    ["{%", "%}"],
    ["{#", "#}"],
    ["(", ")"],
    ["[", "]"]
  ],
  "autoClosingPairs": [
    { "open": "{%", "close": " %}" },
    { "open": "{#", "close": " #}" },
    { "open": "\"", "close": "\"" },
    { "open": "'", "close": "'" },
    { "open": "(", "close": ")" },
    { "open": "[", "close": "]" },
    { "open": "{", "close": "}" }
  ],
  "surroundingPairs": [
    ["{%", "%}"],
    ["{#", "#}"],
    ["\"", "\""],
    ["'", "'"],
    ["(", ")"],
    ["[", "]"],
    ["{", "}"]
  ]
}
```

### `package.json` (Extension Manifest)

Key fields:

```json
{
  "name": "wispy-grove",
  "displayName": "Wispy Grove",
  "description": "Syntax highlighting for Wispy Grove templates",
  "categories": ["Programming Languages"],
  "contributes": {
    "languages": [{
      "id": "grov",
      "aliases": ["Wispy Grove", "Grove Template"],
      "extensions": [".grov"],
      "configuration": "./language-configuration.json"
    }],
    "grammars": [{
      "language": "grov",
      "scopeName": "text.html.grov",
      "path": "./syntaxes/grove.tmLanguage.json",
      "embeddedLanguages": {
        "text.html.basic": "html"
      }
    }]
  }
}
```

### Built-in Filter Recognition

The following filter names should be recognized after `|` and highlighted as `support.function.filter.grov`:

**String:** `upper`, `lower`, `title`, `capitalize`, `trim`, `lstrip`, `rstrip`, `replace`, `truncate`, `center`, `ljust`, `rjust`, `split`, `wordcount`

**Collection:** `length`, `first`, `last`, `join`, `sort`, `reverse`, `unique`, `min`, `max`, `sum`, `map`, `batch`, `flatten`, `keys`, `values`

**Numeric:** `abs`, `round`, `ceil`, `floor`, `int`, `float`

**Type/Logic:** `default`, `string`, `bool`

**HTML:** `escape`, `striptags`, `nl2br`

**Special:** `safe`

---

## Phase 2: Tree-sitter Grammar (Neovim + Zed)

### Grammar Design

The Tree-sitter grammar (`grammar.js`) should parse Grove as an **embedded language** on top of HTML. The recommended approach:

1. Use `tree-sitter-html` as the base parser via `externals` or injection.
2. Define Grove-specific nodes for:
   - `output_expression` — `{% expr %}`
   - `tag_statement` — `{% #tag ... %}`, `{% :branch %}`, `{% /tag %}`
   - `comment` — `{# ... #}`
   - `component_definition` — `<Component name="X">...</Component>`
   - `component_invocation` — `<PascalCase>...</PascalCase>`
   - `attribute_expression` — `{expr}` inside attribute values
   - `expression` — operators, literals, filters, access chains
3. Emit named nodes for each tag type (`if_block`, `each_block`, `let_block`, `fill_block`, etc.) to enable semantic queries.

### `highlights.scm` Query Map

```scheme
; Delimiters
["{%" "%}" "{#" "#}"] @punctuation.bracket

; Sigils
["#" ":" "/"] @punctuation.special

; Comments
(comment) @comment

; Keywords — conditionals
["if" "else" "else if"] @keyword.conditional

; Keywords — loops
["each" "as" "empty"] @keyword.repeat

; Keywords — assignment
["set" "let"] @keyword

; Keywords — imports
["import" "from"] @keyword.import

; Keywords — slots
["slot" "fill"] @keyword

; Keywords — web primitives
["asset" "meta" "hoist" "capture"] @keyword

; Keywords — verbatim
["verbatim"] @keyword

; Keywords — logical
["and" "or" "not"] @keyword.operator

; Components
(component_definition
  (tag_name) @type.definition)

(component_invocation
  (tag_name) @type)

; Attribute expressions
(attribute_expression
  ["{" "}"] @punctuation.bracket)

; Literals
(string) @string
(number) @number
(boolean) @boolean
(nil) @constant.builtin

; Operators
["+" "-" "*" "/" "%" "~"] @operator
["==" "!=" "<" "<=" ">" ">="] @operator
["|"] @operator
["="] @operator
["?" ":"] @operator

; Filters
(filter_name) @function.builtin

; Built-in functions
((identifier) @function.builtin
  (#any-of? @function.builtin "range"))

; Variables
(identifier) @variable
```

### Neovim Integration

- Register the parser with `nvim-treesitter` via a custom parser config.
- Drop `highlights.scm` into `queries/grov/highlights.scm`.
- Add `injections.scm` to delegate HTML regions to `tree-sitter-html`.

### Zed Integration

- Create a Zed extension with `extension.toml` pointing to the Tree-sitter grammar.
- Include the same `highlights.scm` queries.
- Minimal Rust glue (Zed extensions compile Tree-sitter grammars to WASM).

---

## Syntax Edge Cases

These cases need explicit handling in both grammars:

1. **Whitespace stripping:** `{%-`, `-%}` — the `-` is part of the delimiter, not an operator.
2. **Verbatim blocks:** `{% #verbatim %}...{% /verbatim %}` — everything between is literal text, no pattern matching.
3. **Nested delimiters in strings:** `{% "use {% braces %}" %}` — strings inside expressions can contain delimiter-like characters.
4. **Filter chains:** `{% value | filter1 | filter2(arg) %}` — each `|` starts a new filter context.
5. **Inline ternary:** `{% x ? "yes" : "no" %}` — `?` and `:` as ternary operators inside expressions, not sigils.
6. **Sigil disambiguation:** `#`, `:`, `/` are sigils only as the first non-whitespace token after `{%`. Inside expressions they are operators or have no special meaning.
7. **PascalCase vs HTML tags:** Tags starting with `[A-Z]` are component invocations; tags starting with `[a-z]` are standard HTML. The grammar must distinguish these.
8. **Attribute expressions:** `{expr}` in attribute values uses single braces — must only match inside attribute positions, not arbitrary text.
9. **Component `name` attribute:** `<Component name="Card">` — the `name` attribute value is a component name, not a string expression.
10. **Multi-line tags:** Tags and expressions can span multiple lines.
11. **Import paths:** `{% import Card from "components/cards" %}` — the string after `from` is a file path, not a general expression.
12. **Named arguments with `=`:** `{% asset "/app.css" type="stylesheet" %}` — distinguish assignment `=` from comparison operators.

---

## Acceptance Criteria

### Phase 1 (VS Code)
- [ ] `.grov` files auto-detect as Grove language
- [ ] HTML syntax highlighting works unchanged inside Grove files
- [ ] `{% %}` delimiters are highlighted for all server operations (output, control flow, assignment, imports, etc.)
- [ ] `{# #}` comment delimiters are highlighted and content is dimmed/grayed
- [ ] Sigils (`#`, `:`, `/`) are visually distinct within `{% %}` tags
- [ ] Keywords inside `{% %}` tags are highlighted by category (conditional, loop, assignment, import, etc.)
- [ ] Expressions inside `{% %}` are highlighted (strings, numbers, operators, identifiers)
- [ ] Filter names after `|` are highlighted as built-in functions
- [ ] PascalCase elements (`<Card>`, `<Base>`) are highlighted as component invocations, distinct from HTML tags
- [ ] `<Component>` definition elements are highlighted as component definitions
- [ ] `{expression}` in attribute values is highlighted with expression scopes
- [ ] `{% #verbatim %}` blocks suppress all Grove highlighting
- [ ] Whitespace strip markers (`-`) are highlighted
- [ ] Auto-closing pairs work for `{% %}` and `{# #}` delimiter types
- [ ] Block comment toggle uses `{# #}`

### Phase 2 (Tree-sitter)
- [ ] Tree-sitter grammar parses all Grove constructs into named AST nodes
- [ ] Sigil-based block structure (`#open`, `:branch`, `/close`) is represented in the AST
- [ ] Component definitions and invocations have distinct AST node types
- [ ] Attribute expressions (`{expr}`) are parsed as expression nodes
- [ ] `highlights.scm` provides semantic highlighting for all token types
- [ ] Neovim highlights Grove files correctly via nvim-treesitter
- [ ] Zed highlights Grove files correctly via extension
- [ ] HTML injection works (HTML regions are parsed by tree-sitter-html)

---

## Reference: Similar Projects to Study

These VS Code extensions for similar templating engines are good references for implementation patterns:

- **vscode-svelte** — Svelte syntax with `{#if}`, `{:else}`, `{/if}` sigil-based control flow — closest match to Grove's sigil conventions
- **vscode-nunjucks** — Jinja2-like syntax, similar `{% %}` delimiter set
- **vscode-liquid** — Shopify Liquid, same `{% %}` pattern
- **vscode-jinja** — Direct Jinja2 support
- **tree-sitter-svelte** — Tree-sitter grammar for Svelte — useful reference for sigil-based block parsing and component element handling
- **tree-sitter-embedded-template** — Generic Tree-sitter grammar for `{% %}` embedded templates
