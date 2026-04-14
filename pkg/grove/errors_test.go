package grove_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// ─── VM Runtime Errors ────────────────────────────────────────────────────────

// TestError_ModuloByZero verifies that modulo by zero returns a RuntimeError.
func TestError_ModuloByZero(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% 5 % 0 %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "modulo")
}

// TestError_IndexOnNonCollection verifies that indexing non-indexable types errors.
func TestError_IndexOnNonCollection(t *testing.T) {
	eng := newEngine(t)
	cases := []struct {
		tmpl string
		desc string
	}{
		{`{% 5[0] %}`, "index on int"},
		{`{% true[0] %}`, "index on bool"},
		{`{% nil[0] %}`, "index on nil"},
	}
	for _, tc := range cases {
		err := renderErr(t, eng, tc.tmpl, grove.Data{})
		require.Error(t, err, tc.desc)
	}
}

// TestError_IndexOutOfBounds verifies that out-of-bounds access returns nil/empty.
func TestError_IndexOutOfBounds(t *testing.T) {
	eng := newEngine(t)
	// Accessing beyond list bounds returns nil (not an error)
	result := render(t, eng, `[{% [1, 2, 3][10] %}]`, grove.Data{})
	require.Equal(t, "[]", result)
}

// TestError_AttrOnNil verifies that accessing attributes on nil returns nil (not error).
func TestError_AttrOnNil(t *testing.T) {
	eng := newEngine(t)
	// In Grove, accessing .field on nil returns nil, not an error
	result := render(t, eng, `{% nil.field %}`, grove.Data{})
	require.Equal(t, "", result)
}

// TestError_StrictMode_NestedMissing verifies that StrictVariables catches nested undefined access.
func TestError_StrictMode_NestedMissing(t *testing.T) {
	eng := newEngine(t, grove.WithStrictVariables(true))
	err := renderErr(t, eng, `{% user.name %}`, grove.Data{})
	require.Error(t, err)
}

// ─── Parser Errors ────────────────────────────────────────────────────────────

// TestError_ParseError_UnclosedIf verifies that unclosed {% #if %} is a parse error.
func TestError_ParseError_UnclosedIf(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #if true %}content`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "if")
}

// TestError_ParseError_UnclosedEach verifies that unclosed {% #each %} is a parse error.
func TestError_ParseError_UnclosedEach(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #each items as x %}content`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "each")
}

// TestError_ParseError_UnclosedLet verifies that unclosed {% #let %} is a parse error.
func TestError_ParseError_UnclosedLet(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #let %}x = 1`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "let")
}

// TestError_ParseError_UnclosedCapture verifies that unclosed {% #capture %} is a parse error.
func TestError_ParseError_UnclosedCapture(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #capture x %}content`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "capture")
}

// TestError_ParseError_UnclosedFill verifies that unclosed {% #fill %} is a parse error.
func TestError_ParseError_UnclosedFill(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #fill "x" %}content`, grove.Data{})
	require.Error(t, err)
}

// TestError_ParseError_UnclosedVerbatim verifies that unclosed {% #verbatim %} is a parse error.
func TestError_ParseError_UnclosedVerbatim(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #verbatim %}content`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "verbatim")
}

// TestError_ParseError_UnclosedHoist verifies that unclosed {% #hoist %} is a parse error.
func TestError_ParseError_UnclosedHoist(t *testing.T) {
	eng := newEngine(t)
	err := renderErr(t, eng, `{% #hoist %}content`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "hoist")
}

// TestError_ParseError_InvalidSet verifies that invalid {% set %} syntax is a parse error.
func TestError_ParseError_InvalidSet(t *testing.T) {
	eng := newEngine(t)
	cases := []struct {
		tmpl string
		desc string
	}{
		{`{% set %}`, "set without var"},
		{`{% set x %}`, "set without ="},
		{`{% set x = %}`, "set without value"},
	}
	for _, tc := range cases {
		err := renderErr(t, eng, tc.tmpl, grove.Data{})
		require.Error(t, err, tc.desc)
	}
}

// TestError_ParseError_InvalidImport verifies that invalid {% import %} syntax is a parse error.
func TestError_ParseError_InvalidImport(t *testing.T) {
	eng := newEngine(t)
	cases := []struct {
		tmpl string
		desc string
	}{
		{`{% import %}`, "import without name"},
		{`{% import Foo %}`, "import without from"},
		{`{% import Foo from %}`, "import without path"},
	}
	for _, tc := range cases {
		err := renderErr(t, eng, tc.tmpl, grove.Data{})
		require.Error(t, err, tc.desc)
	}
}

// TestError_ParseError_LineNumbers verifies that parse errors report correct line numbers.
func TestError_ParseError_LineNumbers(t *testing.T) {
	eng := newEngine(t)
	tmpl := `line 1
line 2
{% #if true %}
line 4
line 5`
	_, err := eng.RenderTemplate(context.Background(), tmpl, grove.Data{})
	require.Error(t, err)
	// The error should reference line 3 or later (unclosed if tag on line 3)
	require.Contains(t, err.Error(), "if")
}
