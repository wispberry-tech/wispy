// pkg/grove/inheritance_test.go
// Tests for layout-via-composition model: layouts are components with <Slot>/<Fill>.
// Replaces the old {% extends %}/{% block %}/super() inheritance system.
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// renderLayout creates an engine with the given MemoryStore and renders the named template.
func renderLayout(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) string {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), name, data)
	require.NoError(t, err)
	return result.Body
}

// renderLayoutErr creates an engine with the given MemoryStore and returns the render error.
func renderLayoutErr(t *testing.T, store *grove.MemoryStore, name string, data grove.Data) error {
	t.Helper()
	eng := grove.New(grove.WithStore(store))
	_, err := eng.Render(context.Background(), name, data)
	return err
}

// --- Child overrides slot (was: child extends base and overrides a block) ---

func TestLayout_ChildOverridesSlot(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><html><body><Slot name="content">base</Slot></body></html></Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="content">child</Fill></Base>`)
	require.Equal(t, "<html><body>child</body></html>", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Multiple slots (was: multiple blocks, child overrides all) ---

func TestLayout_MultipleSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base">[<Slot name="a">A</Slot>|<Slot name="b">B</Slot>]</Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="a">X</Fill><Fill slot="b">Y</Fill></Base>`)
	require.Equal(t, "[X|Y]", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Partial override: only some slots filled ---

func TestLayout_PartialSlotOverride(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base">[<Slot name="a">A</Slot>|<Slot name="b">B</Slot>]</Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="a">X</Fill></Base>`)
	require.Equal(t, "[X|B]", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Slot fallback: no Fill provided, fallback content renders ---

func TestLayout_SlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><Slot name="footer">Default Footer</Slot></Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base></Base>`)
	require.Equal(t, "Default Footer", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Slot fallback content (replaces super()) ---
// There is no super() equivalent. Fill completely replaces the fallback.
// If the user wants parent content, they must repeat it or extract it.

func TestLayout_SlotFallbackContent(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><Slot name="title">Base Title</Slot></Component>`)

	// Without a Fill, fallback renders.
	store.Set("nofill.html", `<Import src="base" name="Base" /><Base></Base>`)
	require.Equal(t, "Base Title", renderLayout(t, store, "nofill.html", grove.Data{}))

	// With a Fill, fallback is completely replaced (no way to "append" like super).
	store.Set("withfill.html", `<Import src="base" name="Base" /><Base><Fill slot="title">Child Title</Fill></Base>`)
	require.Equal(t, "Child Title", renderLayout(t, store, "withfill.html", grove.Data{}))
}

// --- Data passed through to layout component ---

func TestLayout_DataPassedThrough(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><title><Slot name="title"></Slot></title></Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="title">{% page %}</Fill></Base>`)
	require.Equal(t, "<title>Home</title>", renderLayout(t, store, "child.html", grove.Data{"page": "Home"}))
}

// --- Content outside slots in layout renders ---

func TestLayout_ContentOutsideSlotsRendered(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base">BEFORE<Slot name="x">default</Slot>AFTER</Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="x">override</Fill></Base>`)
	require.Equal(t, "BEFOREoverrideAFTER", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Nested composition (was: multi-level inheritance) ---

func TestLayout_NestedComposition(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("root.html", `<Component name="Root">[<Slot name="a">root</Slot>]</Component>`)
	store.Set("mid.html", `<Import src="root" name="Root" /><Component name="Mid"><Root><Fill slot="a"><Slot name="content">mid</Slot></Fill></Root></Component>`)
	store.Set("leaf.html", `<Import src="mid" name="Mid" /><Mid><Fill slot="content">leaf</Fill></Mid>`)
	require.Equal(t, "[leaf]", renderLayout(t, store, "leaf.html", grove.Data{}))
}

// --- Three-level with wrapping markup at mid level ---

func TestLayout_NestedCompositionWithWrapper(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("grandparent.html", `<Component name="Root"><html><Slot name="body">Root default</Slot></html></Component>`)
	store.Set("parent.html", `<Import src="grandparent" name="Root" /><Component name="Mid"><Root><Fill slot="body"><div class="mid"><Slot name="content">Mid default</Slot></div></Fill></Root></Component>`)
	store.Set("child.html", `<Import src="parent" name="Mid" /><Mid><Fill slot="content">Leaf content</Fill></Mid>`)
	require.Equal(t, `<html><div class="mid">Leaf content</div></html>`, renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Leaf does not override mid's slot, fallback shows ---

func TestLayout_NestedCompositionFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("root.html", `<Component name="Root">[<Slot name="a">root</Slot>]</Component>`)
	store.Set("mid.html", `<Import src="root" name="Root" /><Component name="Mid"><Root><Fill slot="a"><Slot name="content">mid-default</Slot></Fill></Root></Component>`)
	store.Set("leaf.html", `<Import src="mid" name="Mid" /><Mid></Mid>`)
	require.Equal(t, "[mid-default]", renderLayout(t, store, "leaf.html", grove.Data{}))
}

// --- Nested slots (was: block nested in block) ---

func TestLayout_NestedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><Slot name="outer">[<Slot name="inner">inner-default</Slot>]</Slot></Component>`)
	store.Set("child.html", `<Import src="base" name="Base" /><Base><Fill slot="inner">inner-override</Fill></Base>`)
	require.Equal(t, "[inner-override]", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- extends syntax is a parse error (removed from language) ---

func TestLayout_ExtendsIsParseError(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `base`)
	store.Set("bad.html", `{% extends "base.html" %}{% block content %}x{% endblock %}`)
	err := renderLayoutErr(t, store, "bad.html", grove.Data{})
	require.Error(t, err)
}

func TestLayout_ExtendsInInlineTemplate(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, grove.Data{})
	require.Error(t, err)
}

// --- Missing import target ---

func TestLayout_MissingImportTarget(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("child.html", `<Import src="missing" name="Layout" /><Layout></Layout>`)
	err := renderLayoutErr(t, store, "child.html", grove.Data{})
	require.Error(t, err)
}

// --- Layout component renders correctly standalone (fallbacks show) ---

func TestLayout_StandaloneRender(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<Component name="Base"><nav>nav</nav><Slot name="content">default</Slot><footer>foot</footer></Component>`)
	store.Set("page.html", `<Import src="base" name="Base" /><Base></Base>`)
	require.Equal(t, "<nav>nav</nav>default<footer>foot</footer>", renderLayout(t, store, "page.html", grove.Data{}))
}

// --- Four-level composition chain ---

func TestLayout_FourLevelChain(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("gp.html", `<Component name="GP">[<Slot name="x">gp</Slot>]</Component>`)
	store.Set("p.html", `<Import src="gp" name="GP" /><Component name="P"><GP><Fill slot="x"><Slot name="x">p</Slot></Fill></GP></Component>`)
	store.Set("c.html", `<Import src="p" name="P" /><Component name="C"><P><Fill slot="x"><Slot name="x">c</Slot></Fill></P></Component>`)
	store.Set("gc.html", `<Import src="c" name="C" /><C><Fill slot="x">gc</Fill></C>`)
	require.Equal(t, "[gc]", renderLayout(t, store, "gc.html", grove.Data{}))
}
