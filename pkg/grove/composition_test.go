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
	store.Set("greet.html", `<Component name="Greet" who>Hello, {% who %}!</Component>`)
	store.Set("page.html", `<Import src="greet" name="Greet" /><Greet who="World" />`)
	require.Equal(t, "Hello, World!", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_DefaultProp(t *testing.T) {
	// Old: {% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet() }}
	store := grove.NewMemoryStore()
	store.Set("greet.html", `<Component name="Greet" who="stranger">Hi {% who %}</Component>`)
	store.Set("page.html", `<Import src="greet" name="Greet" /><Greet />`)
	require.Equal(t, "Hi stranger", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_PropOverridesDefault(t *testing.T) {
	// Old: {% macro greet(name="stranger") %}Hi {{ name }}{% endmacro %}{{ greet(name="Wispy") }}
	store := grove.NewMemoryStore()
	store.Set("greet.html", `<Component name="Greet" who="stranger">Hi {% who %}</Component>`)
	store.Set("page.html", `<Import src="greet" name="Greet" /><Greet who="Wispy" />`)
	require.Equal(t, "Hi Wispy", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_MultipleProps(t *testing.T) {
	// Old: {% macro link(href, text, target="_self") %}...{% endmacro %}
	store := grove.NewMemoryStore()
	store.Set("link.html", `<Component name="Link" href text target="_self"><a href="{% href %}" target="{% target %}">{% text %}</a></Component>`)
	store.Set("page.html", `<Import src="link" name="Link" /><Link href="https://example.com" text="Click" target="_blank" />`)
	require.Equal(t, `<a href="https://example.com" target="_blank">Click</a>`, renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_IsolatedScope(t *testing.T) {
	// Components cannot read caller variables — scope is isolated
	store := grove.NewMemoryStore()
	store.Set("peek.html", `<Component name="Peek">[{% secret %}]</Component>`)
	store.Set("page.html", `<Set secret="outer" /><Import src="peek" name="Peek" /><Peek />`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_OutputIsSafe(t *testing.T) {
	// Component output is SafeHTML — not double-escaped
	store := grove.NewMemoryStore()
	store.Set("bold.html", `<Component name="Bold" text><b>{% text %}</b></Component>`)
	store.Set("page.html", `<Import src="bold" name="Bold" /><Bold text="hi" />`)
	require.Equal(t, "<b>hi</b>", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── SLOTS (replaces caller()) ───────────────────────────────────────────────

func TestComponent_Slot_Basic(t *testing.T) {
	// Old: {% macro card(title) %}<div><h2>{{ title }}</h2>{{ caller() }}</div>{% endmacro %}
	//      {% call card("Orders") %}<p>3 orders</p>{% endcall %}
	store := grove.NewMemoryStore()
	store.Set("card.html", `<Component name="Card" title><div><h2>{% title %}</h2><Slot /></div></Component>`)
	store.Set("page.html", `<Import src="card" name="Card" /><Card title="Orders"><p>3 orders</p></Card>`)
	require.Equal(t, "<div><h2>Orders</h2><p>3 orders</p></div>", renderStore(t, store, "page.html", grove.Data{}))
}

func TestComponent_Slot_RenderedTwice(t *testing.T) {
	// Old: caller() called twice renders body each time
	store := grove.NewMemoryStore()
	store.Set("wrap.html", `<Component name="Wrap"><Slot />|<Slot /></Component>`)
	store.Set("page.html", `<Import src="wrap" name="Wrap" /><Wrap>body</Wrap>`)
	require.Equal(t, "body|body", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── IMPORT + COMPONENT (replaces include) ───────────────────────────────────

func TestImport_Basic(t *testing.T) {
	// Old: {% include "nav.html" %} with shared scope
	// New: import + component with explicit props
	store := grove.NewMemoryStore()
	store.Set("nav.html", `<Component name="Nav" user><nav>{% user %}</nav></Component>`)
	store.Set("page.html", `before <Import src="nav" name="Nav" /><Nav user="Alice" /> after`)
	require.Equal(t, "before <nav>Alice</nav> after", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_WithProps(t *testing.T) {
	// Old: {% include "part.html" color="blue" size="lg" %}
	store := grove.NewMemoryStore()
	store.Set("part.html", `<Component name="Part" color size>{% color %}-{% size %}</Component>`)
	store.Set("page.html", `<Import src="part" name="Part" /><Part color="blue" size="lg" />`)
	require.Equal(t, "blue-lg", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_IsolatedScope(t *testing.T) {
	// Old: {% render "part.html" %} — isolated, secret not visible
	// New: components are always isolated
	store := grove.NewMemoryStore()
	store.Set("part.html", `<Component name="Part">[{% secret %}]</Component>`)
	store.Set("page.html", `<Set secret="hidden" /><Import src="part" name="Part" /><Part />`)
	require.Equal(t, "[]", renderStore(t, store, "page.html", grove.Data{}))
}

func TestImport_ExplicitProp(t *testing.T) {
	// Old: {% render "card.html" item="Widget" %} — isolated with explicit var
	store := grove.NewMemoryStore()
	store.Set("card.html", `<Component name="Card" item>[{% item %}][{% secret %}]</Component>`)
	store.Set("page.html", `<Set secret="hidden" /><Import src="card" name="Card" /><Card item="Widget" />`)
	require.Equal(t, "[Widget][]", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── IMPORT from multi-component file (replaces {% import "macros.html" as m %}) ─

func TestImport_FromComponentFile(t *testing.T) {
	// Old: {% import "macros.html" as m %}{{ m.greet("Wispy") }}
	store := grove.NewMemoryStore()
	store.Set("macros.html", `<Component name="Greet" who>Hello, {% who %}!</Component>`)
	store.Set("page.html", `<Import src="macros" name="Greet" /><Greet who="Wispy" />`)
	require.Equal(t, "Hello, Wispy!", renderStore(t, store, "page.html", grove.Data{}))
}

// ─── COMPONENT in FOR loop ───────────────────────────────────────────────────

func TestComponent_InsideForLoop(t *testing.T) {
	// Old: {% for item in items %}{% include "row.html" %}{% endfor %}
	// New: must pass loop var as prop
	store := grove.NewMemoryStore()
	store.Set("row.html", `<Component name="Row" item>{% item %},</Component>`)
	store.Set("page.html", `<Import src="row" name="Row" /><For each={items} as="item"><Row item={item} /></For>`)
	require.Equal(t, "a,b,c,", renderStore(t, store, "page.html", grove.Data{"items": []string{"a", "b", "c"}}))
}

// ─── NEW: Multi-import ───────────────────────────────────────────────────────

func TestImport_MultiImport(t *testing.T) {
	// Import multiple components from a single file
	store := grove.NewMemoryStore()
	store.Set("ui.html", `<Component name="Card" title><div class="card">{% title %}</div></Component>
<Component name="Badge" label><span class="badge">{% label %}</span></Component>
<Component name="Button" text><button>{% text %}</button></Component>`)
	store.Set("page.html", `<Import src="ui" name="Card, Badge, Button" /><Card title="Info" /><Badge label="new" /><Button text="OK" />`)
	require.Equal(t, `<div class="card">Info</div><span class="badge">new</span><button>OK</button>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Wildcard import ────────────────────────────────────────────────────

func TestImport_WildcardImport(t *testing.T) {
	// Import all components from a file with wildcard
	store := grove.NewMemoryStore()
	store.Set("ui.html", `<Component name="Card" title><div>{% title %}</div></Component>
<Component name="Badge" label><span>{% label %}</span></Component>`)
	store.Set("page.html", `<Import src="ui" name="*" /><Card title="X" /><Badge label="Y" />`)
	require.Equal(t, `<div>X</div><span>Y</span>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Wildcard with namespace ────────────────────────────────────────────

func TestImport_WildcardWithNamespace(t *testing.T) {
	// Import all with namespace prefix
	store := grove.NewMemoryStore()
	store.Set("ui.html", `<Component name="Card" title><div>{% title %}</div></Component>
<Component name="Badge" label><span>{% label %}</span></Component>`)
	store.Set("page.html", `<Import src="ui" name="*" as="UI" /><UI.Card title="X" /><UI.Badge label="Y" />`)
	require.Equal(t, `<div>X</div><span>Y</span>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Alias ──────────────────────────────────────────────────────────────

func TestImport_Alias(t *testing.T) {
	// Import with alias renames the component locally
	store := grove.NewMemoryStore()
	store.Set("cards.html", `<Component name="Card" title><div class="card">{% title %}</div></Component>`)
	store.Set("page.html", `<Import src="cards" name="Card" as="InfoCard" /><InfoCard title="Details" />`)
	require.Equal(t, `<div class="card">Details</div>`, renderStore(t, store, "page.html", grove.Data{}))
}

// ─── NEW: Duplicate local name error ─────────────────────────────────────────

func TestImport_DuplicateLocalName_Error(t *testing.T) {
	// Importing two components with the same local name is a parse error
	store := grove.NewMemoryStore()
	store.Set("a.html", `<Component name="Card" title>A: {% title %}</Component>`)
	store.Set("b.html", `<Component name="Card" title>B: {% title %}</Component>`)
	store.Set("page.html", `<Import src="a" name="Card" /><Import src="b" name="Card" /><Card title="X" />`)
	err := renderStoreErr(t, store, "page.html", grove.Data{})
	require.Error(t, err)
}

// ─── NEW: Missing component error ────────────────────────────────────────────

func TestImport_MissingComponent_Error(t *testing.T) {
	// Importing a name that doesn't exist in the target file is an error
	store := grove.NewMemoryStore()
	store.Set("ui.html", `<Component name="Card" title><div>{% title %}</div></Component>`)
	store.Set("page.html", `<Import src="ui" name="Badge" /><Badge label="X" />`)
	err := renderStoreErr(t, store, "page.html", grove.Data{})
	require.Error(t, err)
}

// ─── NEW: Circular dependency error ──────────────────────────────────────────

func TestImport_CircularDependency_Error(t *testing.T) {
	// a.html imports from b.html and b.html imports from a.html
	store := grove.NewMemoryStore()
	store.Set("a.html", `<Component name="A"><Import src="b" name="B" /><B /></Component>`)
	store.Set("b.html", `<Component name="B"><Import src="a" name="A" /><A /></Component>`)
	store.Set("page.html", `<Import src="a" name="A" /><A />`)
	err := renderStoreErr(t, store, "page.html", grove.Data{})
	require.Error(t, err)
}
