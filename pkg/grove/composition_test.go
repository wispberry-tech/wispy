// pkg/grove/composition_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// renderStore creates an engine with the given store and renders the named template.
func renderStore(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) string {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// renderStoreErr creates an engine with the given store and renders, returning the error.
func renderStoreErr(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) error {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), name, data)
	return err
}

// ─── MemoryStore + eng.Render() ──────────────────────────────────────────────

func TestRender_NamedTemplate_Basic(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("hello.html", `Hello, {% name %}!`)
	require.Equal(t, "Hello, Wispy!", renderStore(t, store, "hello.html", grove.Data{"name": "Wispy"}))
}

func TestRender_NamedTemplate_NotFound(t *testing.T) {
	store := grove.NewMemoryStore()
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), "missing.html", grove.Data{})
	require.Error(t, err)
}

// ─── COMPONENTS (replaces inline macros) ─────────────────────────────────────

func TestComponent_BasicProp(t *testing.T) {
	// Old: {% macro greet(name) %}Hello, {{ name }}!{% endmacro %}{{ greet("World") }}
	// New: component in store, caller imports and invokes
	store := grove.NewMemoryStore()
	store.Set("greet.html", `Hello, {% who %}!`)
	store.Set("page.html", `{% import Greet from "greet" %}<Greet who="World" />`)
	require.Equal(t, "Hello, World!", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultProp(t *testing.T) {
	// Old: {% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet() }}
	store := grove.NewMemoryStore()
	store.Set("greet.html", `Hi {% who %}`)
	store.Set("page.html", `{% import Greet from "greet" %}<Greet who="stranger" />`)
	require.Equal(t, "Hi stranger", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_PropOverridesDefault(t *testing.T) {
	// Old: {% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet(name="Wispy") }}
	store := grove.NewMemoryStore()
	store.Set("greet.html", `Hi {% who %}`)
	store.Set("page.html", `{% import Greet from "greet" %}<Greet who="Wispy" />`)
	require.Equal(t, "Hi Wispy", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleProps(t *testing.T) {
	// Old: {% macro link(href, text, target="_self") %}...{% endmacro %}
	store := grove.NewMemoryStore()
	store.Set("link.html", `<a href="{% href %}" target="{% target %}">{% text %}</a>`)
	store.Set("page.html", `{% import Link from "link" %}<Link href="https://example.com" text="Click" target="_blank" />`)
	require.Equal(t, `<a href="https://example.com" target="_blank">Click</a>`, renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_IsolatedScope(t *testing.T) {
	// Components cannot read caller variables — scope is isolated
	store := grove.NewMemoryStore()
	store.Set("peek.html", `[{% secret %}]`)
	store.Set("page.html", `{% set secret = "outer" %}{% import Peek from "peek" %}<Peek />`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_OutputIsSafe(t *testing.T) {
	// Component output is SafeHTML — not double-escaped
	store := grove.NewMemoryStore()
	store.Set("bold.html", `<b>{% text %}</b>`)
	store.Set("page.html", `{% import Bold from "bold" %}<Bold text="hi" />`)
	require.Equal(t, "<b>hi</b>", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── SLOTS (replaces caller()) ───────────────────────────────────────────────

func TestComponent_Slot_Basic(t *testing.T) {
	// Old: {% macro card(title) %}<div><h2>{{ title }}</h2>{{ caller() }}</div>{% endmacro %}
	//      {% call card("Orders") %}<p>3 orders</p>{% endcall %}
	store := grove.NewMemoryStore()
	store.Set("card.html", `<div><h2>{% title %}</h2>{% slot %}</div>`)
	store.Set("page.html", `{% import Card from "card" %}<Card title="Orders"><p>3 orders</p></Card>`)
	require.Equal(t, "<div><h2>Orders</h2><p>3 orders</p></div>", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_Slot_RenderedTwice(t *testing.T) {
	// Old: caller() called twice renders body each time
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `{% slot %}|{% slot %}`)
	store.Set("page.html", `{% import Wrap from "wrap" %}<Wrap>body</Wrap>`)
	require.Equal(t, "body|body", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── IMPORT + COMPONENT (replaces include) ───────────────────────────────────

func TestImport_Basic(t *testing.T) {
	// Old: {% include "nav.html" %} with shared scope
	// New: import + component with explicit props
	store := grove.NewMemoryStore()
	store.Set("nav.html", `<nav>{% user %}</nav>`)
	store.Set("page.html", `before {% import Nav from "nav" %}<Nav user="Alice" /> after`)
	require.Equal(t, "before <nav>Alice</nav> after", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_WithProps(t *testing.T) {
	// Old: {% include "part.html" color="blue" size="lg" %}
	store := grove.NewMemoryStore()
	store.Set("part.html", `{% color %}-{% size %}`)
	store.Set("page.html", `{% import Part from "part" %}<Part color="blue" size="lg" />`)
	require.Equal(t, "blue-lg", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_IsolatedScope(t *testing.T) {
	// Old: {% render "part.html" %} — isolated, secret not visible
	// New: components are always isolated
	store := grove.NewMemoryStore()
	store.Set("part.html", `[{% secret %}]`)
	store.Set("page.html", `{% set secret = "hidden" %}{% import Part from "part" %}<Part />`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_ExplicitProp(t *testing.T) {
	// Old: {% render "card.html" item="Widget" %} — isolated with explicit var
	store := grove.NewMemoryStore()
	store.Set("card.html", `[{% item %}][{% secret %}]`)
	store.Set("page.html", `{% set secret = "hidden" %}{% import Card from "card" %}<Card item="Widget" />`)
	require.Equal(t, "[Widget][]", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── IMPORT from multi-component file (replaces {% import "macros.html" as m %}) ─

func TestImport_FromComponentFile(t *testing.T) {
	// Old: {% import "macros.html" as m %}{{ m.greet("Wispy") }}
	store := grove.NewMemoryStore()
	store.Set("macros.html", `Hello, {% who %}!`)
	store.Set("page.html", `{% import Greet from "macros" %}<Greet who="Wispy" />`)
	require.Equal(t, "Hello, Wispy!", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── COMPONENT in FOR loop ───────────────────────────────────────────────────

func TestComponent_InsideForLoop(t *testing.T) {
	// Old: {% for item in items %}{% include "row.html" %}{% endfor %}
	// New: must pass loop var as prop
	store := grove.NewMemoryStore()
	store.Set("row.html", `{% item %},`)
	store.Set("page.html", `{% import Row from "row" %}{% #each items as item %}<Row item={item} />{% /each %}`)
	require.Equal(t, "a,b,c,", renderStore(t, store, "page.html", grove.Data{"items": []string{"a", "b", "c"}}))
}

// ─── NEW: Multi-import ───────────────────────────────────────────────────────

func TestImport_MultiImport(t *testing.T) {
	// Import multiple components from separate files in a single import statement
	store := grove.NewMemoryStore()
	store.Set("card.html", `<div class="card">{% title %}</div>`)
	store.Set("badge.html", `<span class="badge">{% label %}</span>`)
	store.Set("button.html", `<button>{% text %}</button>`)
	// Each component now lives in its own file — multi-import from a single file no longer applies.
	store.Set("page.html", `{% import Card from "card" %}{% import Badge from "badge" %}{% import Button from "button" %}<Card title="Info" /><Badge label="new" /><Button text="OK" />`)
	require.Equal(t, `<div class="card">Info</div><span class="badge">new</span><button>OK</button>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Alias ──────────────────────────────────────────────────────────────

func TestImport_Alias(t *testing.T) {
	// Import with alias renames the component locally
	store := grove.NewMemoryStore()
	store.Set("cards.html", `<div class="card">{% title %}</div>`)
	store.Set("page.html", `{% import Card as InfoCard from "cards" %}<InfoCard title="Details" />`)
	require.Equal(t, `<div class="card">Details</div>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Duplicate local name error ─────────────────────────────────────────

func TestImport_DuplicateLocalName_Error(t *testing.T) {
	// Importing two components with the same local name is a parse error
	store := grove.NewMemoryStore()
	store.Set("a.html", `A: {% title %}`)
	store.Set("b.html", `B: {% title %}`)
	store.Set("page.html", `{% import Card from "a" %}{% import Card from "b" %}<Card title="X" />`)
	err := renderStoreErr(t, store, "page.html", grove.Data{})
	require.Error(t, err)
}

// ─── NEW: Circular dependency error ──────────────────────────────────────────

func TestImport_CircularDependency_Error(t *testing.T) {
	// a.html imports from b.html and b.html imports from a.html
	store := grove.NewMemoryStore()
	store.Set("a.html", `{% import B from "b" %}<B />`)
	store.Set("b.html", `{% import A from "a" %}<A />`)
	store.Set("page.html", `{% import A from "a" %}<A />`)
	err := renderStoreErr(t, store, "page.html", grove.Data{})
	require.Error(t, err)
}
