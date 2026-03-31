# Grove Components + Slots — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the component system — `{% component %}`, `{% slot %}`, `{% fill %}`, and `{% props %}` — to the full pipeline: AST nodes, parser, bytecode, compiler, and VM.

**Architecture:** Components load their template from the store (like `{% render %}`), but add two things on top: (1) **named and default slot dispatch** — the component template declares `{% slot %}` holes that are filled by the caller, (2) **prop validation** — `{% props %}` in the component template declares required and optional parameters with runtime errors for missing or unknown props. Fill content is **lazily rendered in the caller's scope**: fill bodies are compiled into `ComponentDef.Fills` in the bytecode; when the component template hits `OP_SLOT`, the VM executes the matching fill bytecode with the caller's scope saved on the component call stack. This correctly isolates component variables from fill content while allowing fill content to reference caller variables. Components may themselves use `{% extends %}` — the block slot mechanism from Plan 5 and the component fill mechanism are orthogonal and work together.

**Tech Stack:** Go 1.24, standard library, `github.com/stretchr/testify v1.9.0`. Module: `grove`.

---

## Scope: Plan 6 of 7

| Plan | Delivers |
|------|---------|
| 1 — done | Core engine: variables, expressions, auto-escape, filters, global context |
| 2 — done | Control flow: if/elif/else/unless, for/empty/range, set, with, capture |
| 3 — done | Built-in filter catalogue (41 filters) |
| 4 — done | Macros + template composition: macro/call, include, render, import, MemoryStore |
| 5 — done | Layout inheritance: extends/block/super() |
| **6 — this plan** | Components + slots: component/slot/fill/props |
| 7 | Web app primitives: asset/hoist, sandbox, FileSystemStore, hot-reload, HTTP integration |

---

## TDD Approach

**Phase 1 (Task 1):** Write all tests — they fail. That's correct.
**Phase 2 (Tasks 2–5):** Implement feature by feature until `go test ./...` is green.

---

## Syntax Reference

```html
{# Component definition — components/card.html #}
{% props title, variant="default", elevated=false %}

<div class="card card--{{ variant }}">
  <h2>{{ title }}</h2>
  {% slot "actions" %}{% endslot %}          {# named slot, empty default #}
  <div class="body">{% slot %}{% endslot %}</div>   {# default (unnamed) slot #}
  <footer>
    {% slot "footer" %}Default footer{% endslot %}  {# named slot with fallback #}
  </footer>
</div>

{# Caller template #}
{% component "components/card.html" title="Orders" variant="primary" %}
  {# everything outside fill tags → default slot #}
  <p>You have {{ count }} orders.</p>

  {% fill "actions" %}
    <button>View All</button>
  {% endfill %}
{% endcomponent %}
```

---

## Key Design Decisions

### Slot scope
Fill content executes in the **caller's scope**, not the component's. The component's props are NOT visible inside fills. This matches Jinja2's `{% call %}` semantics and is the correct behaviour.

### Props validation (runtime errors)
- **Missing required prop** (declared in `{% props %}` with no default, not passed) → `RuntimeError`
- **Unknown prop** (passed a key not declared in `{% props %}`) → `RuntimeError`
- **No `{% props %}` declaration** → no validation; passed props are bound by name (permissive mode)

### Default slot
Content in the `{% component %}` body that is NOT inside a `{% fill %}` block becomes the fill for the unnamed default slot (`{% slot %}`). If all body content is inside named fills, the default slot receives an empty body.

### `{% asset %}` in component templates
**Deferred to Plan 7.** For now, `{% asset %}` in a component template is a `ParseError` (not silently ignored) so users know it isn't supported yet.

### Components + inheritance
If the component template uses `{% extends %}`, the Plan 5 block slot mechanism handles it transparently. The component fill table (`v.compStack`) persists through the inheritance walk, so `{% slot %}` opcodes in parent templates are resolved correctly.

---

## File Map

| File | Change |
|------|--------|
| `pkg/grove/component_test.go` | NEW — all Plan 6 tests |
| `internal/ast/node.go` | ADD `ComponentNode`, `SlotNode`, `PropsNode` |
| `internal/parser/parser.go` | Parse `component`/`endcomponent`, `slot`/`endslot`, `props` tags; `fill`/`endfill` consumed inside component body parser |
| `internal/compiler/bytecode.go` | ADD `FillDef`, `ComponentDef`; ADD `Props []MacroParam`, `Components []ComponentDef` to `Bytecode` |
| `internal/compiler/compiler.go` | Compile `ComponentNode` → `OP_COMPONENT`; `SlotNode` → `OP_SLOT`; `PropsNode` → `OP_PROPS_INIT` |
| `internal/vm/vm.go` | ADD `compStack []componentFrame`; handle `OP_COMPONENT`, `OP_SLOT`, `OP_PROPS_INIT` |

---

## Task 1: Write All Tests

**Files:**
- Create: `pkg/grove/component_test.go`

- [ ] **Step 1: Create `pkg/grove/component_test.go`**

```go
// pkg/grove/component_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// renderComponent creates an engine with a store and renders the named template.
func renderComponent(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) string {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// ─── Basic component + default slot ──────────────────────────────────────────

func TestComponent_DefaultSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "box.html" %}<p>Hello</p>{% endcomponent %}`)
	require.Equal(t, "<div><p>Hello</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("box.html", `<div>{% slot %}fallback{% endslot %}</div>`)
	store.Set("page.html", `{% component "box.html" %}{% endcomponent %}`) // no fill
	require.Equal(t, "<div>fallback</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<header>{% slot "title" %}{% endslot %}</header><main>{% slot %}{% endslot %}</main>`)
	store.Set("page.html", `{% component "card.html" %}body{% fill "title" %}My Title{% endfill %}{% endcomponent %}`)
	require.Equal(t, "<header>My Title</header><main>body</main>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedSlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<footer>{% slot "footer" %}Default Footer{% endslot %}</footer>`)
	store.Set("page.html", `{% component "card.html" %}{% endcomponent %}`) // no footer fill
	require.Equal(t, "<footer>Default Footer</footer>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleNamedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("layout.html", `[{% slot "a" %}A{% endslot %}|{% slot "b" %}B{% endslot %}]`)
	store.Set("page.html", `{% component "layout.html" %}{% fill "a" %}X{% endfill %}{% fill "b" %}Y{% endfill %}{% endcomponent %}`)
	require.Equal(t, "[X|Y]", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Props ────────────────────────────────────────────────────────────────────

func TestComponent_Props_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label, type="button" %}<button type="{{ type }}">{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="Save" type="submit" %}{% endcomponent %}`)
	require.Equal(t, `<button type="submit">Save</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_Default(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label, type="button" %}<button type="{{ type }}">{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="OK" %}{% endcomponent %}`) // type uses default
	require.Equal(t, `<button type="button">OK</button>`, renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_Props_MissingRequired_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label %}<button>{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" %}{% endcomponent %}`) // label missing
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "label")
}

func TestComponent_Props_UnknownProp_Error(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("btn.html", `{% props label %}<button>{{ label }}</button>`)
	store.Set("page.html", `{% component "btn.html" label="OK" unknown="x" %}{% endcomponent %}`)
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
}

func TestComponent_NoProps_PermissiveMode(t *testing.T) {
	// No {% props %} declaration — any passed props are bound, no validation
	store := grove.NewMemoryStore()
	store.Set("tag.html", `<span class="{{ cls }}">{{ text }}</span>`)
	store.Set("page.html", `{% component "tag.html" cls="badge" text="New" %}{% endcomponent %}`)
	require.Equal(t, `<span class="badge">New</span>`, renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Fill scope (caller's variables visible inside fills) ─────────────────────

func TestComponent_FillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "wrap.html" %}<p>{{ message }}</p>{% endcomponent %}`)
	require.Equal(t, "<div><p>Hello!</p></div>", renderComponent(t, store, "page.html", grove.Data{"message": "Hello!"}))
}

func TestComponent_FillDoesNotSeeComponentProps(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `{% props secret="hidden" %}<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "wrap.html" secret="topsecret" %}<p>{{ secret }}</p>{% endcomponent %}`)
	// "secret" inside the fill renders from caller scope, not component scope
	// caller scope has no "secret" var → renders empty (non-strict mode)
	require.Equal(t, "<div><p></p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

func TestComponent_NamedFillSeesCallerVars(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("card.html", `<h2>{% slot "title" %}{% endslot %}</h2>`)
	store.Set("page.html", `{% component "card.html" %}{% fill "title" %}{{ heading }}{% endfill %}{% endcomponent %}`)
	require.Equal(t, "<h2>My Heading</h2>", renderComponent(t, store, "page.html", grove.Data{"heading": "My Heading"}))
}

// ─── Nested components ────────────────────────────────────────────────────────

func TestComponent_Nested(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("inner.html", `[{% slot %}{% endslot %}]`)
	store.Set("outer.html", `<div>{% slot %}{% endslot %}</div>`)
	store.Set("page.html", `{% component "outer.html" %}{% component "inner.html" %}content{% endcomponent %}{% endcomponent %}`)
	require.Equal(t, "<div>[content]</div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── Component + inheritance ──────────────────────────────────────────────────

func TestComponent_WithExtends(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base-card.html", `{% props title %}<div><h2>{{ title }}</h2>{% slot %}{% endslot %}</div>`)
	// card.html extends base-card.html — inheriting its layout
	store.Set("card.html", `{% props title %}{% extends "base-card.html" %}`)
	store.Set("page.html", `{% component "card.html" title="News" %}<p>Content</p>{% endcomponent %}`)
	require.Equal(t, "<div><h2>News</h2><p>Content</p></div>", renderComponent(t, store, "page.html", grove.Data{}))
}

// ─── component in inline template is an error ─────────────────────────────────

func TestComponent_InInlineTemplate_Error(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(), `{% component "x.html" %}{% endcomponent %}`, grove.Data{})
	require.Error(t, err)
}

// ─── component requires a store ───────────────────────────────────────────────

func TestComponent_NoStore_Error(t *testing.T) {
	eng := grove.New() // no store
	_, err := eng.RenderTemplate(context.Background(), `{% component "x.html" %}{% endcomponent %}`, grove.Data{})
	require.Error(t, err)
}
```

---

## Task 2: AST Nodes + Parser

**Files:**
- Modify: `internal/ast/node.go`
- Modify: `internal/parser/parser.go`

### Step 2a: Add AST nodes to `internal/ast/node.go`

Add after the existing `BlockNode`:

```go
// PropsNode is {% props name, name2="default", ... %} — declares accepted props.
// Must appear at the top of a component template. Reuses MacroParam for params.
type PropsNode struct {
	Params []MacroParam
	Line   int
}

func (*PropsNode) groveNode() {}

// FillNode is {% fill "name" %}...{% endfill %} inside a component call body.
// FillNode is NOT directly part of the template AST — it is consumed by the parser
// when parsing a ComponentNode and stored in ComponentNode.Fills.
type FillNode struct {
	Name string
	Body []Node
	Line int
}

// ComponentNode is {% component "name" k=v, ... %}...{% endcomponent %}.
type ComponentNode struct {
	Name        string         // template name (string literal)
	Props       []NamedArgNode // passed props (key=value pairs)
	DefaultFill []Node         // body content outside fill blocks → fed to {% slot %}
	Fills       []FillNode     // named {% fill %}...{% endfill %} blocks
	Line        int
}

func (*ComponentNode) groveNode() {}

// SlotNode is {% slot ["name"] %}...{% endslot %} inside a component template.
type SlotNode struct {
	Name    string // "" = default slot
	Default []Node // fallback content rendered when no matching fill
	Line    int
}

func (*SlotNode) groveNode() {}
```

### Step 2b: Parser changes in `internal/parser/parser.go`

**In `parseTag()`** add cases (reject `component` in inline mode):

```go
case "component":
    if p.inline {
        return nil, &groverrors.ParseError{Line: tagStart.Line, Col: tagStart.Col,
            Message: "component not allowed in inline templates"}
    }
    return p.parseComponent(tagStart)

case "slot":
    return p.parseSlot(tagStart)

case "props":
    return p.parseProps(tagStart)
```

**New parser methods:**

```go
// parseProps parses {% props name, name2="default", ... %}.
// Reuses parseMacroParams() — the syntax is identical.
func (p *parser) parseProps(tagStart lexer.Token) (*ast.PropsNode, error) {
	p.advance() // consume "props"
	params, err := p.parseMacroParams()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.PropsNode{Params: params, Line: tagStart.Line}, nil
}

// parseSlot parses {% slot ["name"] %}...{% endslot %}.
func (p *parser) parseSlot(tagStart lexer.Token) (*ast.SlotNode, error) {
	p.advance() // consume "slot"
	name := ""
	if p.peek().Kind == lexer.TK_STRING {
		name = p.advance().Value
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endslot")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endslot"); err != nil {
		return nil, err
	}
	return &ast.SlotNode{Name: name, Default: body, Line: tagStart.Line}, nil
}

// parseComponent parses {% component "name" k=v, ... %}...{% endcomponent %}.
// The body is scanned to separate {% fill %} blocks from default-slot content.
func (p *parser) parseComponent(tagStart lexer.Token) (*ast.ComponentNode, error) {
	p.advance() // consume "component"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after component")
	}

	// Parse props: key=val, key2=val2
	var props []ast.NamedArgNode
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected prop name in component tag")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after prop name")
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		props = append(props, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	// Parse body: separate {% fill %} from default-slot content
	node := &ast.ComponentNode{Name: nameTok.Value, Props: props, Line: tagStart.Line}
	if err := p.parseComponentBody(node); err != nil {
		return nil, err
	}
	return node, nil
}

// parseComponentBody parses until {% endcomponent %}, routing {% fill %} blocks
// into node.Fills and everything else into node.DefaultFill.
func (p *parser) parseComponentBody(node *ast.ComponentNode) error {
	for !p.atEOF() {
		// Check for {% endcomponent %}
		if p.peekTag("endcomponent") {
			p.consumeTag("endcomponent")
			return nil
		}
		// Check for {% fill "name" %}
		if p.peekTag("fill") {
			fill, err := p.parseFill()
			if err != nil {
				return err
			}
			node.Fills = append(node.Fills, *fill)
			continue
		}
		// Everything else goes to DefaultFill
		n, err := p.parseNode()
		if err != nil {
			return err
		}
		if n != nil {
			node.DefaultFill = append(node.DefaultFill, n)
		}
	}
	return p.errorf(p.peek().Line, p.peek().Col, "unclosed component block — expected endcomponent")
}

// parseFill parses {% fill "name" %}...{% endfill %}.
func (p *parser) parseFill() (*ast.FillNode, error) {
	tagStart := p.peek()
	p.advance() // consume {%
	p.advance() // consume "fill"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted slot name after fill")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endfill")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endfill"); err != nil {
		return nil, err
	}
	return &ast.FillNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}
```

> **Note on `peekTag` / `consumeTag` helpers:** These inspect the token stream for `{% tagname %}` without consuming. You may need to add these parser helpers or adapt to existing parser state helpers. Look at how `parseBody` detects its stop keyword for reference.

---

## Task 3: Bytecode + Compiler

**Files:**
- Modify: `internal/compiler/bytecode.go`
- Modify: `internal/compiler/compiler.go`

### Step 3a: Bytecode additions (`internal/compiler/bytecode.go`)

New opcodes:
```go
// OP_COMPONENT — A=comp_idx B=prop_pair_count
// Stack: B pairs of (key Value, value Value) pushed before this op (key first)
// Pops B*2 values, loads component template, validates props, renders with fill table.
OP_COMPONENT

// OP_SLOT — A=name_idx B=default_fill_idx (index into bc.Blocks for fallback)
// Checks v.compStack.top().fills for a matching name. If found, executes fill body
// in caller scope. If not found, executes bc.Blocks[B] (the default content).
// B=0xFFFF means no default (empty slot).
OP_SLOT

// OP_PROPS_INIT — no operands; reads bc.Props and v.compStack.top().props;
// validates required/unknown props and binds them into the current scope.
OP_PROPS_INIT
```

New types:
```go
// FillDef is a compiled fill body associated with a named slot.
// Name="" is the default (unnamed) slot fill.
type FillDef struct {
	Name string
	Body *Bytecode
}

// ComponentDef holds the compiled fill bodies for a single {% component %} call site.
type ComponentDef struct {
	Name  string    // template name
	Fills []FillDef // compiled fill bodies; index 0 is always the default fill (Name="")
}
```

Add to `Bytecode`:
```go
Props      []MacroParam    // from {% props %} declaration; nil = no declaration (permissive)
Components []ComponentDef  // one entry per {% component %} call in this template
```

### Step 3b: Compiler additions (`internal/compiler/compiler.go`)

Add to `cmp` struct:
```go
props      []MacroParam
components []ComponentDef
```

Update `Compile()` return to include `Props` and `Components`.

**Compile `PropsNode` (statement — emits `OP_PROPS_INIT`):**
```go
case *ast.PropsNode:
	// Store declaration on the bytecode
	for _, p := range n.Params {
		mp := MacroParam{Name: p.Name}
		if p.Default != nil {
			mp.Default = constValueOf(p.Default)
		}
		c.props = append(c.props, mp)
	}
	c.emit(OP_PROPS_INIT, 0, 0, 0)
```

**Compile `SlotNode` (statement — emits `OP_SLOT`):**
```go
case *ast.SlotNode:
	// Compile default content into Blocks (reuse existing BlockDef)
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Default); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	defaultBC := &Bytecode{Instrs: sub.instrs, Consts: sub.consts, Names: sub.names, Macros: sub.macros}
	c.blocks = append(c.blocks, BlockDef{Name: "__slot__:" + n.Name, Body: defaultBC})
	defaultIdx := len(c.blocks) - 1
	c.emit(OP_SLOT, uint16(c.addName(n.Name)), uint16(defaultIdx), 0)
```

**Compile `ComponentNode` (statement — emits `OP_COMPONENT`):**
```go
case *ast.ComponentNode:
	return c.compileComponent(n)
```

```go
func (c *cmp) compileComponent(n *ast.ComponentNode) error {
	// Compile default fill body
	defaultSub := &cmp{nameIdx: make(map[string]int)}
	if err := defaultSub.compileBody(n.DefaultFill); err != nil {
		return err
	}
	defaultSub.emit(OP_HALT, 0, 0, 0)
	defaultBC := &Bytecode{
		Instrs: defaultSub.instrs, Consts: defaultSub.consts,
		Names: defaultSub.names, Macros: defaultSub.macros,
	}

	// Compile named fill bodies
	fills := []FillDef{{Name: "", Body: defaultBC}} // index 0 = default fill
	for _, fill := range n.Fills {
		sub := &cmp{nameIdx: make(map[string]int)}
		if err := sub.compileBody(fill.Body); err != nil {
			return err
		}
		sub.emit(OP_HALT, 0, 0, 0)
		fillBC := &Bytecode{
			Instrs: sub.instrs, Consts: sub.consts,
			Names: sub.names, Macros: sub.macros,
		}
		fills = append(fills, FillDef{Name: fill.Name, Body: fillBC})
	}

	def := ComponentDef{Name: n.Name, Fills: fills}
	compIdx := len(c.components)
	c.components = append(c.components, def)

	// Push prop key-value pairs onto stack
	for _, prop := range n.Props {
		c.emitPushConst(prop.Key)
		if err := c.compileExpr(prop.Value); err != nil {
			return err
		}
	}

	c.emit(OP_COMPONENT, uint16(compIdx), uint16(len(n.Props)), 0)

	// OP_COMPONENT result is output directly to the writer — no OUTPUT needed
	return nil
}
```

---

## Task 4: VM Execution

**Files:**
- Modify: `internal/vm/vm.go`

### Step 4a: New VM types and fields

```go
// componentFrame holds state for an active component call.
type componentFrame struct {
	fills       []compiler.FillDef // fill bodies indexed by search
	callerScope *scope.Scope       // caller's scope — used for fill rendering
}

// Add to VM struct:
compStack [16]componentFrame
csdepth   int // current component stack depth
```

Add to pool cleanup defer:
```go
v.csdepth = 0
```

### Step 4b: `OP_COMPONENT` handler

```go
case compiler.OP_COMPONENT:
	compDef := bc.Components[instr.A]
	propCount := int(instr.B)

	// Pop prop key-value pairs (pushed key-first)
	props := make(map[string]any, propCount)
	for i := propCount - 1; i >= 0; i-- {
		val := v.pop()
		key := v.pop()
		props[key.String()] = val
	}

	// Load component template
	compBC, err := v.eng.LoadTemplate(compDef.Name)
	if err != nil {
		return "", &runtimeErr{msg: fmt.Sprintf("component %q: %v", compDef.Name, err)}
	}

	// Save caller scope; push component frame
	callerScope := v.sc
	if v.csdepth >= len(v.compStack) {
		return "", &runtimeErr{msg: "component nesting too deep (max 16)"}
	}
	v.compStack[v.csdepth] = componentFrame{
		fills:       compDef.Fills,
		callerScope: callerScope,
	}
	v.csdepth++

	// Set up component scope: globals → component scope
	globalSc := scope.New(nil)
	for k, val := range v.eng.GlobalData() {
		globalSc.Set(k, val)
	}
	v.sc = scope.New(globalSc)

	// Bind props immediately if no {% props %} declaration in component
	// (if bc.Props is non-nil, OP_PROPS_INIT will do validation + binding)
	if compBC.Props == nil {
		for k, val := range props {
			v.sc.Set(k, val)
		}
	} else {
		// Store passed props for OP_PROPS_INIT to consume
		v.compStack[v.csdepth-1].callerScope = callerScope // already set
		// Pass props via a temporary scope var — or store on frame
		// Simplest: store on the frame with a separate field
		// (add passedProps map[string]any to componentFrame)
	}

	_, err = v.run(ctx, compBC)
	v.csdepth--
	v.sc = callerScope
	if err != nil {
		return "", err
	}
```

> **Implementation note:** Add `passedProps map[string]any` to `componentFrame` to carry the passed props through to `OP_PROPS_INIT`. Set it when pushing the frame.

### Step 4c: `OP_PROPS_INIT` handler

```go
case compiler.OP_PROPS_INIT:
	if v.csdepth == 0 {
		return "", &runtimeErr{msg: "props declaration outside component context"}
	}
	frame := &v.compStack[v.csdepth-1]
	passed := frame.passedProps
	declared := bc.Props

	// Check for unknown props
	declared set := make(map[string]bool, len(declared))
	for _, p := range declared {
		declaredSet[p.Name] = true
	}
	for k := range passed {
		if !declaredSet[k] {
			return "", &runtimeErr{msg: fmt.Sprintf("component %q: unknown prop %q", "?", k)}
		}
	}

	// Bind props (passed value or default)
	for _, p := range declared {
		if val, ok := passed[p.Name]; ok {
			v.sc.Set(p.Name, val)
		} else if p.Default != nil {
			v.sc.Set(p.Name, FromAny(p.Default))
		} else {
			return "", &runtimeErr{msg: fmt.Sprintf("component: missing required prop %q", p.Name)}
		}
	}
```

### Step 4d: `OP_SLOT` handler

```go
case compiler.OP_SLOT:
	slotName := bc.Names[instr.A]
	defaultBlockIdx := int(instr.B)

	if v.csdepth == 0 {
		// {% slot %} used outside a component — render default only
		if _, err := v.run(ctx, bc.Blocks[defaultBlockIdx].Body); err != nil {
			return "", err
		}
		break
	}

	frame := &v.compStack[v.csdepth-1]

	// Find matching fill
	var fillBody *compiler.Bytecode
	for i := range frame.fills {
		if frame.fills[i].Name == slotName {
			fillBody = frame.fills[i].Body
			break
		}
	}

	if fillBody != nil {
		// Render fill in caller scope (lazy render)
		savedSC := v.sc
		v.sc = scope.New(frame.callerScope)
		_, err := v.run(ctx, fillBody)
		v.sc = savedSC
		if err != nil {
			return "", err
		}
	} else if defaultBlockIdx != 0xFFFF {
		// No fill provided — render slot default content
		if _, err := v.run(ctx, bc.Blocks[defaultBlockIdx].Body); err != nil {
			return "", err
		}
	}
	// else: empty slot with no fill and no default — render nothing
```

> **Note on default slot lookup:** The default (unnamed) slot is looked up with `slotName = ""`. The default fill for the component is stored in `frame.fills[0]` with `Name=""`. Named fills start at index 1. This means `OP_SLOT` with `slotName=""` naturally finds the default fill.

---

## Task 5: Wire Up + Verify

- [ ] Run `go build ./...` — fix any compile errors.
- [ ] Run `go test ./...` — all Plan 6 tests should pass; existing tests must remain green.
- [ ] Run `go vet ./...` — no issues.

---

## Edge Cases to Watch

| Case | Expected behaviour |
|------|--------------------|
| `{% slot %}` outside a component | Renders its default content (graceful fallback) |
| Component not found in store | `RuntimeError: component "x": template not found` |
| `{% component %}` with no store | `RuntimeError` (store nil check) |
| `{% component %}` in inline template | `ParseError` |
| Unknown prop when `{% props %}` declared | `RuntimeError: unknown prop "x"` |
| Missing required prop | `RuntimeError: missing required prop "label"` |
| No `{% props %}` — any props bound silently | no error |
| Nested components: inner fill sees inner caller scope | ✓ (compStack handles nesting) |
| Component template uses `{% extends %}` | ✓ (compStack persists through inheritance walk) |
| `{% fill %}` outside `{% component %}` body | `ParseError` (fill is only parsed inside component body) |
| Same slot filled twice | Last `{% fill %}` wins (parser keeps last occurrence) |
| `{% asset %}` in component template | `ParseError` (deferred to Plan 7) |

---

## What This Plan Does NOT Include

- `{% asset %}` / `{% hoist %}` — Plan 7
- `FileSystemStore`, hot-reload, HTTP integration — Plan 7
- Sandbox mode — Plan 7
- Bytecode LRU cache — Plan 7
- `{% raw %}` block — Plan 7
