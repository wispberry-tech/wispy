// pkg/wispy/engine_test.go
package grove_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newEngine(t *testing.T, opts ...grove.Option) *grove.Engine {
	t.Helper()
	return grove.New(opts...)
}

func render(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) string {
	t.Helper()
	result, err := eng.RenderTemplate(context.Background(), tmpl, data)
	require.NoError(t, err)
	return result.Body
}

func renderErr(t *testing.T, eng *grove.Engine, tmpl string, data grove.Data) error {
	t.Helper()
	_, err := eng.RenderTemplate(context.Background(), tmpl, data)
	return err
}

// Resolvable test type used by §25 tests
type testProduct struct {
	Name  string
	price float64
}

func (p testProduct) WispyResolve(key string) (any, bool) {
	switch key {
	case "name":
		return p.Name, true
	case "price":
		return p.price, true
	}
	return nil, false
}

// ─── 1. VARIABLES ─────────────────────────────────────────────────────────────

func TestVariables_SimpleString(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `Hello, {{ name }}!`, grove.Data{"name": "World"})
	require.Equal(t, "Hello, World!", got)
}

func TestVariables_NestedAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ user.address.city }}`, grove.Data{
		"user": grove.Data{"address": grove.Data{"city": "Berlin"}},
	})
	require.Equal(t, "Berlin", got)
}

func TestVariables_IndexAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ items[0] }}`, grove.Data{
		"items": []string{"alpha", "beta", "gamma"},
	})
	require.Equal(t, "alpha", got)
}

func TestVariables_MapAccess(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ config["debug"] }}`, grove.Data{
		"config": map[string]any{"debug": "true"},
	})
	require.Equal(t, "true", got)
}

func TestVariables_UndefinedReturnsEmpty(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ missing }}]`, grove.Data{})
	require.Equal(t, "[]", got)
}

func TestVariables_StrictModeErrors(t *testing.T) {
	eng := newEngine(t, grove.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ missing }}`, grove.Data{})
	require.Error(t, err)
	var re *grove.RuntimeError
	require.ErrorAs(t, err, &re)
	require.Contains(t, re.Message, "missing")
}

func TestVariables_Resolvable(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ product.name }}`, grove.Data{
		"product": testProduct{Name: "Widget", price: 9.99},
	})
	require.Equal(t, "Widget", got)
}

func TestVariables_ResolvableHidesUnexposed(t *testing.T) {
	eng := newEngine(t, grove.WithStrictVariables(true))
	err := renderErr(t, eng, `{{ product.secret }}`, grove.Data{
		"product": testProduct{Name: "Widget", price: 9.99},
	})
	require.Error(t, err)
}

// ─── 2. EXPRESSIONS ──────────────────────────────────────────────────────────

func TestExpressions_Arithmetic(t *testing.T) {
	eng := newEngine(t)
	cases := []struct{ tmpl, want string }{
		{`{{ 2 + 3 }}`, "5"},
		{`{{ 10 - 4 }}`, "6"},
		{`{{ 3 * 4 }}`, "12"},
		{`{{ 10 / 4 }}`, "2.5"},
		{`{{ 10 % 3 }}`, "1"},
	}
	for _, tc := range cases {
		got := render(t, eng, tc.tmpl, grove.Data{})
		require.Equal(t, tc.want, got, "template: %s", tc.tmpl)
	}
}

func TestExpressions_StringConcat(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ "Hello" ~ ", " ~ name ~ "!" }}`, grove.Data{"name": "Wispy"})
	require.Equal(t, "Hello, Wispy!", got)
}

func TestExpressions_Comparison(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ x > 5 }}`, grove.Data{"x": 10})
	require.Equal(t, "true", got)
}

func TestExpressions_LogicalOperators(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": true})
	require.Equal(t, "true", got)
	got = render(t, eng, `{{ a and b }}`, grove.Data{"a": true, "b": false})
	require.Equal(t, "false", got)
}

func TestExpressions_Ternary(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ active ? name : "Guest" }}`, grove.Data{
		"name": "Alice", "active": true,
	})
	require.Equal(t, "Alice", got)
	got = render(t, eng, `{{ active ? name : "Guest" }}`, grove.Data{
		"name": "Alice", "active": false,
	})
	require.Equal(t, "Guest", got)
}

func TestExpressions_Not(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ not banned }}`, grove.Data{"banned": false})
	require.Equal(t, "true", got)
}

// ─── 3. FILTERS (basic — full catalogue is Plan 3) ───────────────────────────

func TestFilters_SafeFilter_TrustedHTML(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, grove.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestFilters_CustomFilter(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("shout", func(v grove.Value, args []grove.Value) (grove.Value, error) {
		return grove.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
	})
	got := render(t, eng, `{{ msg | shout }}`, grove.Data{"msg": "hello"})
	require.Equal(t, "HELLO!!!", got)
}

func TestFilters_CustomFilterWithArgs(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("repeat", func(v grove.Value, args []grove.Value) (grove.Value, error) {
		n := grove.ArgInt(args, 0, 2)
		return grove.StringValue(strings.Repeat(v.String(), n)), nil
	})
	got := render(t, eng, `{{ "ha" | repeat(3) }}`, grove.Data{})
	require.Equal(t, "hahaha", got)
}

func TestFilters_CustomHTMLFilter_SkipsEscape(t *testing.T) {
	eng := newEngine(t)
	eng.RegisterFilter("bold", grove.FilterFunc(
		func(v grove.Value, _ []grove.Value) (grove.Value, error) {
			return grove.SafeHTMLValue("<b>" + v.String() + "</b>"), nil
		},
		grove.FilterOutputsHTML(),
	))
	got := render(t, eng, `{{ name | bold }}`, grove.Data{"name": "Wispy"})
	require.Equal(t, "<b>Wispy</b>", got)
}

// ─── 4. AUTO-ESCAPING ────────────────────────────────────────────────────────

func TestEscape_AutoEscapeByDefault(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ input }}`, grove.Data{
		"input": `<script>alert("xss")</script>`,
	})
	require.Equal(t, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`, got)
}

func TestEscape_SafeFilterBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{{ html | safe }}`, grove.Data{"html": "<b>bold</b>"})
	require.Equal(t, "<b>bold</b>", got)
}

func TestEscape_RawBlockBypassesEscape(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `{% raw %}{{ not_a_variable }}{% endraw %}`, grove.Data{})
	require.Equal(t, "{{ not_a_variable }}", got)
}

func TestEscape_NilValueNoOutput(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, `[{{ val }}]`, grove.Data{"val": nil})
	require.Equal(t, "[]", got)
}

// ─── 5. WHITESPACE CONTROL ───────────────────────────────────────────────────

func TestWhitespace_StripLeft(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name }}", grove.Data{"name": "Wispy"})
	require.Equal(t, "Wispy", got)
}

func TestWhitespace_StripRight(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "{{ name -}}  ", grove.Data{"name": "Wispy"})
	require.Equal(t, "Wispy", got)
}

func TestWhitespace_StripBoth(t *testing.T) {
	eng := newEngine(t)
	got := render(t, eng, "  {{- name -}}  extra", grove.Data{"name": "Wispy"})
	require.Equal(t, "Wispyextra", got)
}

func TestWhitespace_TagStrip(t *testing.T) {
	eng := newEngine(t)
	// Uses {% raw %} as the tag vehicle since control-flow tags are Plan 2
	got := render(t, eng, "before\n{%- raw -%}\nhello\n{%- endraw -%}\nafter", grove.Data{})
	require.Equal(t, "beforehelloafter", got)
}

// ─── 6. GLOBAL CONTEXT ───────────────────────────────────────────────────────

func TestGlobalContext_AvailableInAllRenders(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("siteName", "Acme Corp")
	got1 := render(t, eng, `{{ siteName }}`, grove.Data{})
	got2 := render(t, eng, `Welcome to {{ siteName }}`, grove.Data{})
	require.Equal(t, "Acme Corp", got1)
	require.Equal(t, "Welcome to Acme Corp", got2)
}

func TestGlobalContext_RenderContextOverridesGlobal(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("greeting", "Hello")
	got := render(t, eng, `{{ greeting }}`, grove.Data{"greeting": "Hi"})
	require.Equal(t, "Hi", got)
}

func TestGlobalContext_LocalScopeOverridesRenderContext(t *testing.T) {
	eng := newEngine(t)
	eng.SetGlobal("x", "global")
	got := render(t, eng, `{{ x }}`, grove.Data{"x": "render"})
	require.Equal(t, "render", got)
}

// ─── 7. ERROR HANDLING ───────────────────────────────────────────────────────

func TestError_ParseError_LineNumber(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), "line1\n{{ unclosed", grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, 2, pe.Line)
}

func TestError_UndefinedFilterInStrictMode(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ name | nonexistent }}`, grove.Data{"name": "x"})
	require.Error(t, err)
}

func TestError_DivisionByZero(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{{ 10 / x }}`, grove.Data{"x": 0})
	require.Error(t, err)
}

// ─── 8. RENDERTEMPLATE INLINE RESTRICTIONS ───────────────────────────────────

func TestRenderTemplate_ExtendsIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% extends "base.html" %}`, grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Contains(t, pe.Message, "extends not allowed in inline templates")
}

func TestRenderTemplate_ImportIsParseError(t *testing.T) {
	eng := newEngine(t)
	_, err := eng.RenderTemplate(context.Background(), `{% import "macros.html" as m %}`, grove.Data{})
	require.Error(t, err)
	var pe *grove.ParseError
	require.ErrorAs(t, err, &pe)
	require.Contains(t, pe.Message, "import not allowed in inline templates")
}

// ─── 9. CONCURRENT RENDERS ───────────────────────────────────────────────────

func TestEngine_ConcurrentRenders(t *testing.T) {
	eng := newEngine(t)
	const goroutines = 50
	const renders = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines*renders)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < renders; i++ {
				got, err := eng.RenderTemplate(context.Background(),
					`Hello, {{ name }}! ({{ id }})`,
					grove.Data{"name": "Wispy", "id": id},
				)
				if err != nil {
					errors <- err
					return
				}
				expected := fmt.Sprintf("Hello, Wispy! (%d)", id)
				if got.Body != expected {
					errors <- fmt.Errorf("goroutine %d: got %q, want %q", id, got.Body, expected)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

// ─── BENCHMARKS ──────────────────────────────────────────────────────────────

func BenchmarkRender_SimpleSubstitution(b *testing.B) {
	eng := grove.New()
	data := grove.Data{"name": "World", "count": 42}
	bgCtx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}! Count: {{ count }}.`, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRender_Parallel(b *testing.B) {
	eng := grove.New()
	data := grove.Data{"name": "World"}
	bgCtx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := eng.RenderTemplate(bgCtx, `Hello, {{ name }}!`, data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
