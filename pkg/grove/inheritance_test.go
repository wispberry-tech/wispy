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
	store.Set("base.html", `<html><body>{% #slot "content" %}base{% /slot %}</body></html>`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "content" %}child{% /fill %}</Base>`)
	require.Equal(t, "<html><body>child</body></html>", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Multiple slots (was: multiple blocks, child overrides all) ---

func TestLayout_MultipleSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `[{% #slot "a" %}A{% /slot %}|{% #slot "b" %}B{% /slot %}]`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "a" %}X{% /fill %}{% #fill "b" %}Y{% /fill %}</Base>`)
	require.Equal(t, "[X|Y]", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Partial override: only some slots filled ---

func TestLayout_PartialSlotOverride(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `[{% #slot "a" %}A{% /slot %}|{% #slot "b" %}B{% /slot %}]`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "a" %}X{% /fill %}</Base>`)
	require.Equal(t, "[X|B]", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Slot fallback: no Fill provided, fallback content renders ---

func TestLayout_SlotFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `{% #slot "footer" %}Default Footer{% /slot %}`)
	store.Set("child.html", `{% import Base from "base" %}<Base></Base>`)
	require.Equal(t, "Default Footer", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Slot fallback content (replaces super()) ---
// There is no super() equivalent. Fill completely replaces the fallback.
// If the user wants parent content, they must repeat it or extract it.

func TestLayout_SlotFallbackContent(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `{% #slot "title" %}Base Title{% /slot %}`)

	// Without a Fill, fallback renders.
	store.Set("nofill.html", `{% import Base from "base" %}<Base></Base>`)
	require.Equal(t, "Base Title", renderLayout(t, store, "nofill.html", grove.Data{}))

	// With a Fill, fallback is completely replaced (no way to "append" like super).
	store.Set("withfill.html", `{% import Base from "base" %}<Base>{% #fill "title" %}Child Title{% /fill %}</Base>`)
	require.Equal(t, "Child Title", renderLayout(t, store, "withfill.html", grove.Data{}))
}

// --- Data passed through to layout component ---

func TestLayout_DataPassedThrough(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<title>{% slot "title" %}</title>`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "title" %}{% page %}{% /fill %}</Base>`)
	require.Equal(t, "<title>Home</title>", renderLayout(t, store, "child.html", grove.Data{"page": "Home"}))
}

// --- Content outside slots in layout renders ---

func TestLayout_ContentOutsideSlotsRendered(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `BEFORE{% #slot "x" %}default{% /slot %}AFTER`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "x" %}override{% /fill %}</Base>`)
	require.Equal(t, "BEFOREoverrideAFTER", renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Nested composition (was: multi-level inheritance) ---

func TestLayout_NestedComposition(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("root.html", `[{% #slot "a" %}root{% /slot %}]`)
	store.Set("mid.html", `{% import Root from "root" %}<Root>{% #fill "a" %}{% #slot "content" %}mid{% /slot %}{% /fill %}</Root>`)
	store.Set("leaf.html", `{% import Mid from "mid" %}<Mid>{% #fill "content" %}leaf{% /fill %}</Mid>`)
	require.Equal(t, "[leaf]", renderLayout(t, store, "leaf.html", grove.Data{}))
}

// --- Three-level with wrapping markup at mid level ---

func TestLayout_NestedCompositionWithWrapper(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("grandparent.html", `<html>{% #slot "body" %}Root default{% /slot %}</html>`)
	store.Set("parent.html", `{% import Root from "grandparent" %}<Root>{% #fill "body" %}<div class="mid">{% #slot "content" %}Mid default{% /slot %}</div>{% /fill %}</Root>`)
	store.Set("child.html", `{% import Mid from "parent" %}<Mid>{% #fill "content" %}Leaf content{% /fill %}</Mid>`)
	require.Equal(t, `<html><div class="mid">Leaf content</div></html>`, renderLayout(t, store, "child.html", grove.Data{}))
}

// --- Leaf does not override mid's slot, fallback shows ---

func TestLayout_NestedCompositionFallback(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("root.html", `[{% #slot "a" %}root{% /slot %}]`)
	store.Set("mid.html", `{% import Root from "root" %}<Root>{% #fill "a" %}{% #slot "content" %}mid-default{% /slot %}{% /fill %}</Root>`)
	store.Set("leaf.html", `{% import Mid from "mid" %}<Mid></Mid>`)
	require.Equal(t, "[mid-default]", renderLayout(t, store, "leaf.html", grove.Data{}))
}

// --- Nested slots (was: block nested in block) ---

func TestLayout_NestedSlots(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `{% #slot "outer" %}[{% #slot "inner" %}inner-default{% /slot %}]{% /slot %}`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "inner" %}inner-override{% /fill %}</Base>`)
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
	store.Set("child.html", `{% import Layout from "missing" %}<Layout></Layout>`)
	err := renderLayoutErr(t, store, "child.html", grove.Data{})
	require.Error(t, err)
}

// --- Layout component renders correctly standalone (fallbacks show) ---

func TestLayout_StandaloneRender(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<nav>nav</nav>{% #slot "content" %}default{% /slot %}<footer>foot</footer>`)
	store.Set("page.html", `{% import Base from "base" %}<Base></Base>`)
	require.Equal(t, "<nav>nav</nav>default<footer>foot</footer>", renderLayout(t, store, "page.html", grove.Data{}))
}

// --- Four-level composition chain ---

func TestLayout_FourLevelChain(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("gp.html", `[{% #slot "x" %}gp{% /slot %}]`)
	store.Set("p.html", `{% import GP from "gp" %}<GP>{% #fill "x" %}{% #slot "x" %}p{% /slot %}{% /fill %}</GP>`)
	store.Set("c.html", `{% import P from "p" %}<P>{% #fill "x" %}{% #slot "x" %}c{% /slot %}{% /fill %}</P>`)
	store.Set("gc.html", `{% import C from "c" %}<C>{% #fill "x" %}gc{% /fill %}</C>`)
	require.Equal(t, "[gc]", renderLayout(t, store, "gc.html", grove.Data{}))
}
