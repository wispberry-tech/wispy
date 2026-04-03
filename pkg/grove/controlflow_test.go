// pkg/wispy/controlflow_test.go
package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"grove/pkg/grove"
)

// ─── IF / ELIF / ELSE ────────────────────────────────────────────────────────

func TestIf_Basic(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if active %}yes{% else %}no{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "yes", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "no", result.Body)
}

func TestIf_NoElse(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if active %}yes{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"active": false})
	require.NoError(t, err)
	require.Equal(t, "", result.Body)
}

func TestIf_Elif(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if role == "admin" %}admin{% elif role == "mod" %}mod{% else %}user{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "admin"})
	require.NoError(t, err)
	require.Equal(t, "admin", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "mod"})
	require.NoError(t, err)
	require.Equal(t, "mod", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"role": "viewer"})
	require.NoError(t, err)
	require.Equal(t, "user", result.Body)
}

func TestIf_Nested(t *testing.T) {
	eng := grove.New()
	tmpl := `{% if a %}{% if b %}both{% else %}only-a{% endif %}{% else %}neither{% endif %}`
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "both", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"a": true, "b": false})
	require.NoError(t, err)
	require.Equal(t, "only-a", result.Body)
}

// ─── UNLESS ──────────────────────────────────────────────────────────────────

func TestUnless_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% unless banned %}Welcome!{% endunless %}`,
		grove.Data{"banned": false})
	require.Error(t, err)
}

// ─── FOR ─────────────────────────────────────────────────────────────────────

func TestFor_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }},{% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "a,b,c,", result.Body)
}

func TestFor_Empty(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ x }}{% empty %}none{% endfor %}`,
		grove.Data{"items": []string{}})
	require.NoError(t, err)
	require.Equal(t, "none", result.Body)
}

func TestFor_LoopVariables(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index }}:{{ loop.first }}:{{ loop.last }} {% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "1:true:false 2:false:false 3:false:true ", result.Body)
}

func TestFor_LoopLength(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.length }}{% endfor %}`,
		grove.Data{"items": []int{1, 2, 3}})
	require.NoError(t, err)
	require.Equal(t, "333", result.Body)
}

func TestFor_LoopIndex0(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for x in items %}{{ loop.index0 }}{% endfor %}`,
		grove.Data{"items": []string{"a", "b"}})
	require.NoError(t, err)
	require.Equal(t, "01", result.Body)
}

func TestFor_Range(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(1, 4) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "123", result.Body)
}

func TestFor_RangeOneArg(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(3) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "012", result.Body)
}

func TestFor_RangeStep(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i in range(5, 0, -1) %}{{ i }}{% endfor %}`,
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "54321", result.Body)
}

func TestFor_NestedLoopDepth(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.depth }}{% endfor %}{% endfor %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1, 2},
		})
	require.NoError(t, err)
	require.Equal(t, "2222", result.Body)
}

func TestFor_TwoVarList(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for i, item in items %}{{ i }}:{{ item }} {% endfor %}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "0:a 1:b 2:c ", result.Body)
}

func TestFor_TwoVarMap(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for k, v in cfg %}{{ k }}={{ v }} {% endfor %}`,
		grove.Data{"cfg": map[string]any{"b": "2", "a": "1"}})
	require.NoError(t, err)
	// Keys sorted lexicographically
	require.Equal(t, "a=1 b=2 ", result.Body)
}

func TestFor_NestedParentLoop(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% for a in outer %}{% for b in inner %}{{ loop.parent.index }}{% endfor %}{% endfor %}`,
		grove.Data{
			"outer": []int{1, 2},
			"inner": []int{1},
		})
	require.NoError(t, err)
	require.Equal(t, "12", result.Body)
}

// ─── SET ─────────────────────────────────────────────────────────────────────

func TestSet_Basic(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set x = 42 %}{{ x }}`, grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestSet_Expression(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set total = price * qty %}{{ total }}`,
		grove.Data{"price": 5, "qty": 3})
	require.NoError(t, err)
	require.Equal(t, "15", result.Body)
}

func TestSet_StringConcat(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set greeting = "Hello, " ~ name %}{{ greeting }}`,
		grove.Data{"name": "World"})
	require.NoError(t, err)
	require.Equal(t, "Hello, World", result.Body)
}

func TestWith_Removed(t *testing.T) {
	eng := grove.New()
	_, err := eng.RenderTemplate(context.Background(),
		`{% with %}{% set x = 99 %}{% endwith %}`,
		grove.Data{})
	require.Error(t, err)
}

// ─── CAPTURE ─────────────────────────────────────────────────────────────────

func TestCapture(t *testing.T) {
	eng := grove.New()
	eng.RegisterFilter("upcase", func(v grove.Value, _ []grove.Value) (grove.Value, error) {
		s := v.String()
		result := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result[i] = c
		}
		return grove.StringValue(string(result)), nil
	})
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture greeting %}Hello, {{ name }}!{% endcapture %}{{ greeting | upcase }}`,
		grove.Data{"name": "Wispy Grove"})
	require.NoError(t, err)
	require.Equal(t, "HELLO, WISPY GROVE!", result.Body)
}

func TestCapture_UsedInIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture msg %}{% if active %}on{% else %}off{% endif %}{% endcapture %}[{{ msg }}]`,
		grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "[on]", result.Body)
}

// ─── SET scope in loop ────────────────────────────────────────────────────────

func TestSet_InLoop_PersistsAfterLoop(t *testing.T) {
	// for loops do not push a new scope, so set inside loop mutates outer scope
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% set last = "" %}{% for x in items %}{% set last = x %}{% endfor %}{{ last }}`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "c", result.Body)
}

// ─── CAPTURE in loop ─────────────────────────────────────────────────────────

func TestCapture_InsideLoop(t *testing.T) {
	// capture can accumulate loop body output into a variable
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		`{% capture out %}{% for x in items %}{{ x }},{% endfor %}{% endcapture %}[{{ out }}]`,
		grove.Data{"items": []string{"a", "b", "c"}})
	require.NoError(t, err)
	require.Equal(t, "[a,b,c,]", result.Body)
}

// ─── LET ─────────────────────────────────────────────────────────────────────

func TestLet_BasicAssignment(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 42\n{% endlet %}{{ x }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "42", result.Body)
}

func TestLet_MultipleAssignments(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  a = 1\n  b = 2\n  c = 3\n{% endlet %}{{ a }},{{ b }},{{ c }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2,3", result.Body)
}

func TestLet_WithConditional(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% endlet %}{{ x }}",
		grove.Data{"flag": true})
	require.NoError(t, err)
	require.Equal(t, "flagged", result.Body)
}

func TestLet_ConditionalFalse(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = \"default\"\n  if flag\n    x = \"flagged\"\n  end\n{% endlet %}{{ x }}",
		grove.Data{"flag": false})
	require.NoError(t, err)
	require.Equal(t, "default", result.Body)
}

func TestLet_ElifElse(t *testing.T) {
	eng := grove.New()
	tmpl := "{% let %}\n  color = \"gray\"\n  if type == \"error\"\n    color = \"red\"\n  elif type == \"success\"\n    color = \"green\"\n  else\n    color = \"blue\"\n  end\n{% endlet %}{{ color }}"

	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "error"})
	require.NoError(t, err)
	require.Equal(t, "red", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "success"})
	require.NoError(t, err)
	require.Equal(t, "green", result.Body)

	result, err = eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "info"})
	require.NoError(t, err)
	require.Equal(t, "blue", result.Body)
}

func TestLet_ExpressionWithFilters(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  name = raw_name | upper\n{% endlet %}{{ name }}",
		grove.Data{"raw_name": "alice"})
	require.NoError(t, err)
	require.Equal(t, "ALICE", result.Body)
}

func TestLet_WritesToOuterScope(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 1\n{% endlet %}{% let %}\n  y = x + 1\n{% endlet %}{{ y }}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "2", result.Body)
}

func TestLet_NestedIf(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  x = 0\n  if a\n    if b\n      x = 1\n    end\n  end\n{% endlet %}{{ x }}",
		grove.Data{"a": true, "b": true})
	require.NoError(t, err)
	require.Equal(t, "1", result.Body)
}

func TestLet_BlankLinesIgnored(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n\n  x = 1\n\n  y = 2\n\n{% endlet %}{{ x }},{{ y }}", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "1,2", result.Body)
}

func TestLet_WithMapLiteral(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  theme = {bg: \"#fff\", fg: \"#000\"}\n{% endlet %}{{ theme.bg }}",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "#fff", result.Body)
}

func TestLet_NoOutput(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"before{% let %}\n  x = 1\n{% endlet %}after",
		grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "beforeafter", result.Body)
}

func TestLet_MultiLineMapLiteral(t *testing.T) {
	eng := grove.New()
	tmpl := "{% let %}\n  themes = {\n    warn: \"yellow\",\n    err: \"red\",\n    info: \"blue\"\n  }\n  color = themes[type]\n{% endlet %}{{ color }}"
	result, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{"type": "err"})
	require.NoError(t, err)
	require.Equal(t, "red", result.Body)
}

func TestLet_TernaryInExpression(t *testing.T) {
	eng := grove.New()
	result, err := eng.RenderTemplate(context.Background(),
		"{% let %}\n  label = active ? \"on\" : \"off\"\n{% endlet %}{{ label }}",
		grove.Data{"active": true})
	require.NoError(t, err)
	require.Equal(t, "on", result.Body)
}
