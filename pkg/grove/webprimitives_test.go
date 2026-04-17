// pkg/grove/webprimitives_test.go
// Rewritten for Svelte-hybrid syntax (TDD — these tests will FAIL).
package grove_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wispberry-tech/grove/pkg/grove"
)

// ─── {% #verbatim %} tests (was {% raw %}) ──────────────────────────────────

func TestVerbatim_OutputExpr(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #verbatim %}{% variable %}{% /verbatim %}`,
		grove.Data{"variable": "should not render"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "{% variable %}" {
		t.Errorf("want {%% variable %%}, got %q", result.Body)
	}
}

func TestVerbatim_OutputTag(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #verbatim %}{% #if foo %}bar{% /if %}{% /verbatim %}`,
		grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != `{% #if foo %}bar{% /if %}` {
		t.Errorf("got %q", result.Body)
	}
}

func TestVerbatim_MultiLine(t *testing.T) {
	eng := grove.New()
	src := "{% #verbatim %}\nline one\n{% expr %}\n{% /verbatim %}"
	result, err := eng.RenderTemplate(context.Background(), src, grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Body, "{% expr %}") {
		t.Errorf("expected {%% expr %%} in output, got %q", result.Body)
	}
}

func TestVerbatim_PreservesGroveDelimiters(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% #verbatim %}{% not processed %}{% /verbatim %}`,
		grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "{% not processed %}" {
		t.Errorf("expected literal {%% not processed %%}, got %q", result.Body)
	}
}

// ─── {% asset %} tests ──────────────────────────────────────────────────────

func newStoreEng(templates map[string]string, opts ...grove.Option) *grove.Engine {
	s := grove.NewMemoryStore()
	for k, v := range templates {
		s.Set(k, v)
	}
	opts = append(opts, grove.WithStore(s))
	return grove.New(opts...)
}

func TestAsset_StylesheetCollected(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app.css" type="stylesheet" %}hello`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "hello" {
		t.Errorf("body = %q", result.Body)
	}
	if len(result.Assets) != 1 {
		t.Fatalf("want 1 asset, got %d", len(result.Assets))
	}
	a := result.Assets[0]
	if a.Src != "app.css" || a.Type != "stylesheet" {
		t.Errorf("asset = %+v", a)
	}
}

func TestAsset_ScriptCollected(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app.js" type="script" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Assets) != 1 || result.Assets[0].Type != "script" {
		t.Fatalf("assets = %+v", result.Assets)
	}
}

func TestAsset_Deduplication(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app.js" type="script" %}{% asset "app.js" type="script" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Assets) != 1 {
		t.Errorf("want 1 asset after dedup, got %d", len(result.Assets))
	}
}

func TestAsset_BooleanAttrInFootHTML(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app.js" type="script" defer %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	foot := result.FootHTML()
	if !strings.Contains(foot, " defer") {
		t.Errorf("FootHTML() should contain bare 'defer', got %q", foot)
	}
	if strings.Contains(foot, `defer="`) {
		t.Errorf("FootHTML() should NOT contain defer=\"...\", got %q", foot)
	}
}

func TestAsset_Priority(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "b.css" type="stylesheet" %}{% asset "a.css" type="stylesheet" priority=10 %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	head := result.HeadHTML()
	aIdx := strings.Index(head, "a.css")
	bIdx := strings.Index(head, "b.css")
	if aIdx == -1 || bIdx == -1 {
		t.Fatalf("HeadHTML() = %q", head)
	}
	if aIdx > bIdx {
		t.Errorf("a.css (priority=10) should appear before b.css (priority=0): %q", head)
	}
}

func TestAsset_HeadHTML(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "style.css" type="stylesheet" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	head := result.HeadHTML()
	if !strings.Contains(head, `<link rel="stylesheet" href="style.css">`) {
		t.Errorf("HeadHTML() = %q", head)
	}
}

func TestAsset_FootHTML(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "main.js" type="script" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	foot := result.FootHTML()
	if !strings.Contains(foot, `<script src="main.js"`) {
		t.Errorf("FootHTML() = %q", foot)
	}
}

func TestAsset_ModuleFootHTML(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app/main.js" type="module" %}{% asset "vendor.js" type="script" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	foot := result.FootHTML()
	if !strings.Contains(foot, `<script type="module" src="app/main.js">`) {
		t.Errorf("FootHTML() missing module script, got %q", foot)
	}
	if !strings.Contains(foot, `<script src="vendor.js">`) {
		t.Errorf("FootHTML() missing classic script, got %q", foot)
	}
	// Classic scripts must come before module scripts.
	if strings.Index(foot, `type="module"`) < strings.Index(foot, `src="vendor.js"`) {
		t.Errorf("classic scripts should precede module scripts, got %q", foot)
	}
}

func TestAsset_ModuleResolvedThroughManifest(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% asset "app/main.js" type="module" %}`,
	})
	eng.SetAssetResolver(func(logical string) (string, bool) {
		if logical == "app/main.js" {
			return "/static/app/main.abc12345.js", true
		}
		return "", false
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	foot := result.FootHTML()
	want := `<script type="module" src="/static/app/main.abc12345.js"></script>`
	if !strings.Contains(foot, want) {
		t.Errorf("FootHTML() = %q, want substring %q", foot, want)
	}
}

func TestAsset_FromComponent(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"button.html": `{% asset "button.css" type="stylesheet" %}<button>click</button>`,
		"page.html":   `{% import Button from "button" %}<Button />`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Assets) != 1 || result.Assets[0].Src != "button.css" {
		t.Errorf("assets from component not bubbled up: %+v", result.Assets)
	}
}

func TestAsset_InlineTemplateError(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% asset "app.css" type="stylesheet" %}`, grove.Data{})
	if err == nil {
		t.Fatal("expected ParseError for asset in inline template")
	}
}

// ─── {% #hoist %} tests ─────────────────────────────────────────────────────

func TestHoist_BasicTarget(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% #hoist "head" %}<style>.hero{}</style>{% /hoist %}body`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "body" {
		t.Errorf("body = %q", result.Body)
	}
	head := result.GetHoisted("head")
	if !strings.Contains(head, ".hero{}") {
		t.Errorf("GetHoisted(head) = %q", head)
	}
}

func TestHoist_MultipleBlocks(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% #hoist "head" %}first{% /hoist %}{% #hoist "head" %}second{% /hoist %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	head := result.GetHoisted("head")
	if !strings.Contains(head, "first") || !strings.Contains(head, "second") {
		t.Errorf("GetHoisted(head) = %q", head)
	}
	if strings.Index(head, "first") > strings.Index(head, "second") {
		t.Error("fragments should be in declaration order")
	}
}

func TestHoist_IndependentTargets(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% #hoist "head" %}HEAD{% /hoist %}{% #hoist "foot" %}FOOT{% /hoist %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.GetHoisted("head") != "HEAD" {
		t.Errorf("head = %q", result.GetHoisted("head"))
	}
	if result.GetHoisted("foot") != "FOOT" {
		t.Errorf("foot = %q", result.GetHoisted("foot"))
	}
}

func TestHoist_FromComponent(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"widget.html": `{% #hoist "head" %}<style>.widget{}</style>{% /hoist %}widget`,
		"page.html":   `{% import Widget from "widget" %}<Widget />`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.GetHoisted("head"), ".widget{}") {
		t.Errorf("hoist from component not bubbled up: %q", result.GetHoisted("head"))
	}
}

// ─── {% meta %} tests ───────────────────────────────────────────────────────

func TestMeta_NameAttr(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% meta name="description" content="A great page" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["description"] != "A great page" {
		t.Errorf("Meta[description] = %q", result.Meta["description"])
	}
}

func TestMeta_PropertyAttr(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% meta property="og:title" content="My Page" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["og:title"] != "My Page" {
		t.Errorf("Meta[og:title] = %q", result.Meta["og:title"])
	}
}

func TestMeta_CollisionWarning(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% meta name="title" content="First" %}{% meta name="title" content="Second" %}`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["title"] != "Second" {
		t.Errorf("last write should win, got %q", result.Meta["title"])
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for meta key overwrite")
	}
}

func TestMeta_FromComponent(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"hero.html": `{% meta name="og:image" content="/hero.jpg" %}`,
		"page.html": `{% import Hero from "hero" %}<Hero />`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["og:image"] != "/hero.jpg" {
		t.Errorf("meta from component not bubbled up: %v", result.Meta)
	}
}

// ─── FileSystemStore tests ───────────────────────────────────────────────────

func TestFileSystemStore_Load(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "page.html"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	s := grove.NewFileSystemStore(dir)
	data, err := s.Load("page.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q", data)
	}
}

func TestFileSystemStore_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	s := grove.NewFileSystemStore(dir)
	_, err := s.Load("../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestFileSystemStore_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	s := grove.NewFileSystemStore(dir)
	_, err := s.Load("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestFileSystemStore_CleanedPath(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "tmpl.html"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	s := grove.NewFileSystemStore(dir)
	// a/../sub/tmpl.html cleans to sub/tmpl.html — should be allowed
	data, err := s.Load("a/../sub/tmpl.html")
	if err != nil {
		t.Fatalf("unexpected error for cleaned-to-safe path: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("got %q", data)
	}
}

func TestFileSystemStore_RenderFromFS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hi.html"), []byte("Hello {% name %}!"), 0644); err != nil {
		t.Fatal(err)
	}
	eng := grove.New(grove.WithStore(grove.NewFileSystemStore(dir)))
	result, err := eng.Render(context.Background(), "hi.html", grove.Data{"name": "Wispy"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "Hello Wispy!" {
		t.Errorf("body = %q", result.Body)
	}
}

// ─── LRU cache tests ────────────────────────────────────────────────────────

type countingStore struct {
	inner *grove.MemoryStore
	loads map[string]int
}

func (cs *countingStore) Load(name string) ([]byte, error) {
	cs.loads[name]++
	return cs.inner.Load(name)
}

func TestLRUCache_Hit(t *testing.T) {
	ms := grove.NewMemoryStore()
	ms.Set("t.html", "hello")
	cs := &countingStore{inner: ms, loads: make(map[string]int)}

	eng := grove.New(grove.WithStore(cs))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := eng.Render(ctx, "t.html", grove.Data{}); err != nil {
			t.Fatal(err)
		}
	}
	if cs.loads["t.html"] != 1 {
		t.Errorf("expected 1 store load (cache hit), got %d", cs.loads["t.html"])
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	ms := grove.NewMemoryStore()
	ms.Set("a.html", "A")
	ms.Set("b.html", "B")
	ms.Set("c.html", "C")
	cs := &countingStore{inner: ms, loads: make(map[string]int)}

	// Cache size 1: only 1 entry fits
	eng := grove.New(grove.WithStore(cs), grove.WithCacheSize(1))
	ctx := context.Background()

	if _, err := eng.Render(ctx, "a.html", grove.Data{}); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Render(ctx, "b.html", grove.Data{}); err != nil {
		t.Fatal(err)
	}
	// a.html was evicted — should be re-loaded
	if _, err := eng.Render(ctx, "a.html", grove.Data{}); err != nil {
		t.Fatal(err)
	}
	if cs.loads["a.html"] != 2 {
		t.Errorf("expected 2 loads for a.html (eviction + re-load), got %d", cs.loads["a.html"])
	}
}

// ─── RenderTo tests ──────────────────────────────────────────────────────────

func TestRenderTo_WritesBody(t *testing.T) {
	s := grove.NewMemoryStore()
	s.Set("t.html", "hello world")
	eng := grove.New(grove.WithStore(s))

	var buf bytes.Buffer
	if err := eng.RenderTo(context.Background(), "t.html", grove.Data{}, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello world" {
		t.Errorf("got %q", buf.String())
	}
}

func TestRenderTo_PropagatesError(t *testing.T) {
	eng := grove.New() // no store
	var buf bytes.Buffer
	err := eng.RenderTo(context.Background(), "missing.html", grove.Data{}, &buf)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ─── Sandbox tests ───────────────────────────────────────────────────────────

func TestSandbox_AllowedTagsBlocked(t *testing.T) {
	s := grove.NewMemoryStore()
	s.Set("t.html", `{% set x = 1 %}{% x %}`)
	eng := grove.New(
		grove.WithStore(s),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags: []string{"If", "For"},
		}),
	)
	_, err := eng.Render(context.Background(), "t.html", grove.Data{})
	if err == nil {
		t.Fatal("expected ParseError for disallowed tag")
	}
	if !strings.Contains(err.Error(), "set") {
		t.Errorf("error should mention 'set', got: %v", err)
	}
}

func TestSandbox_AllowedFiltersBlocked(t *testing.T) {
	s := grove.NewMemoryStore()
	s.Set("t.html", `{% "hello" | downcase %}`)
	eng := grove.New(
		grove.WithStore(s),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedFilters: []string{"upcase"},
		}),
	)
	_, err := eng.Render(context.Background(), "t.html", grove.Data{})
	if err == nil {
		t.Fatal("expected error for disallowed filter")
	}
	if !strings.Contains(err.Error(), "downcase") {
		t.Errorf("error should mention 'downcase', got: %v", err)
	}
}

func TestSandbox_MaxLoopIter(t *testing.T) {
	s := grove.NewMemoryStore()
	s.Set("t.html", `{% #each items as i %}{% i %}{% /each %}`)
	eng := grove.New(
		grove.WithStore(s),
		grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 3}),
	)
	// 10 items — should exceed limit of 3 iterations
	items := []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	_, err := eng.Render(context.Background(), "t.html", grove.Data{"items": items})
	if err == nil {
		t.Fatal("expected RuntimeError for MaxLoopIter exceeded")
	}
	if !strings.Contains(err.Error(), "limit") && !strings.Contains(err.Error(), "iteration") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSandbox_DisallowedTagInInclude(t *testing.T) {
	// sandbox restrictions apply to included sub-templates:
	// part.html uses {% set %} which is not in AllowedTags
	s := grove.NewMemoryStore()
	s.Set("page.html", `{% import Part from "part" %}<Part />`)
	s.Set("part.html", `{% set x = 1 %}{% x %}`)
	eng := grove.New(
		grove.WithStore(s),
		grove.WithSandbox(grove.SandboxConfig{
			AllowedTags: []string{"#if", "#each", "import", "Component"}, // import/Component allowed; set is not
		}),
	)
	_, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err == nil {
		t.Fatal("expected error for disallowed tag in included template")
	}
	if !strings.Contains(err.Error(), "set") {
		t.Errorf("error should mention 'set', got: %v", err)
	}
}

// ─── {% #hoist %} inside conditional ────────────────────────────────────────

func TestHoist_InsideConditional_True(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% #if show %}{% #hoist "x" %}hoisted{% /hoist %}{% /if %}body`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{"show": true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Body != "body" {
		t.Errorf("body = %q", result.Body)
	}
	if !strings.Contains(result.GetHoisted("x"), "hoisted") {
		t.Errorf("expected hoisted content, got %q", result.GetHoisted("x"))
	}
}

func TestHoist_InsideConditional_False(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"page.html": `{% #if show %}{% #hoist "x" %}hoisted{% /hoist %}{% /if %}body`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{"show": false})
	if err != nil {
		t.Fatal(err)
	}
	if result.GetHoisted("x") != "" {
		t.Errorf("expected no hoisted content when condition false, got %q", result.GetHoisted("x"))
	}
}

// ─── {% meta %} from include ────────────────────────────────────────────────

func TestMeta_FromInclude(t *testing.T) {
	eng := newStoreEng(map[string]string{
		"part.html": `{% meta name="description" content="from include" %}`,
		"page.html": `{% import Part from "part" %}<Part />body`,
	})
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["description"] != "from include" {
		t.Errorf("meta from include not bubbled up: %v", result.Meta)
	}
}

// ─── grove:data tests (Alpine integration) ──────────────────────────────────

func TestGroveData_BasicInjection(t *testing.T) {
	// grove:data="user" on an element with x-data serializes the server
	// variable as JSON into x-data, merging with existing Alpine state.
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`<div grove:data="user" x-data="{ tab: 'info' }">{% user.name %}</div>`,
		grove.Data{"user": map[string]any{"name": "Alice", "role": "admin"}})
	if err != nil {
		t.Fatal(err)
	}
	// grove:data attr is stripped; user is serialized into x-data
	if strings.Contains(result.Body, "grove:data") {
		t.Errorf("grove:data should be stripped from output, got %q", result.Body)
	}
	if !strings.Contains(result.Body, "Alice") {
		t.Errorf("expected user.name to be rendered, got %q", result.Body)
	}
	// The x-data should contain the merged object with serialized user data
	if !strings.Contains(result.Body, `x-data=`) {
		t.Errorf("expected x-data attribute in output, got %q", result.Body)
	}
}

func TestGroveData_MultipleVars(t *testing.T) {
	// grove:data="user, stats" injects multiple server variables into x-data
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`<div grove:data="user, stats" x-data="{ open: false }">content</div>`,
		grove.Data{
			"user":  map[string]any{"name": "Alice"},
			"stats": map[string]any{"views": 42},
		})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Body, "grove:data") {
		t.Errorf("grove:data should be stripped from output, got %q", result.Body)
	}
	if !strings.Contains(result.Body, `x-data=`) {
		t.Errorf("expected x-data attribute in output, got %q", result.Body)
	}
}

func TestGroveData_MissingVar_Error(t *testing.T) {
	// Variable in grove:data that doesn't exist in scope should produce an error
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`<div grove:data="nonexistent" x-data="{}">content</div>`,
		grove.Data{})
	if err == nil {
		t.Fatal("expected error for grove:data referencing missing variable")
	}
}

func TestGroveData_WithoutXData_Error(t *testing.T) {
	// grove:data without x-data on the same element should produce an error
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`<div grove:data="user">content</div>`,
		grove.Data{"user": map[string]any{"name": "Alice"}})
	if err == nil {
		t.Fatal("expected error for grove:data without x-data on same element")
	}
}

// ─── grove:nowarn test ───────────────────────────────────────────────────────

func TestGroveNowarn(t *testing.T) {
	// grove:nowarn="server-loop-in-client-scope" suppresses specific warnings
	eng := newStoreEng(map[string]string{
		"page.html": `<div x-data="{ items: [] }" grove:nowarn="server-loop-in-client-scope">{% #each items as item %}{% item %}{% /each %}</div>`,
	})
	result, err := eng.Render(context.Background(), "page.html",
		grove.Data{"items": []any{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	// The warning "server-loop-in-client-scope" should be suppressed
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "server-loop-in-client-scope") {
			t.Errorf("expected warning to be suppressed by grove:nowarn, got warning: %q", w)
		}
	}
	// grove:nowarn should be stripped from output
	if strings.Contains(result.Body, "grove:nowarn") {
		t.Errorf("grove:nowarn should be stripped from output, got %q", result.Body)
	}
}
