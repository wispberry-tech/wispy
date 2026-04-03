// internal/lexer/lexer_test.go
package lexer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"grove/internal/lexer"
)

func kinds(tokens []lexer.Token) []lexer.TokenKind {
	out := make([]lexer.TokenKind, len(tokens))
	for i, t := range tokens {
		out[i] = t.Kind
	}
	return out
}

func TestLexer_PlainText(t *testing.T) {
	toks, err := lexer.Tokenize("Hello, World!")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{lexer.TK_TEXT, lexer.TK_EOF}, kinds(toks))
	require.Equal(t, "Hello, World!", toks[0].Value)
}

func TestLexer_OutputBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name }}")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{
		lexer.TK_OUTPUT_START, lexer.TK_IDENT, lexer.TK_OUTPUT_END, lexer.TK_EOF,
	}, kinds(toks))
	require.Equal(t, "name", toks[1].Value)
}

func TestLexer_Comment_IsStripped(t *testing.T) {
	toks, err := lexer.Tokenize("{# this is a comment #}after")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{lexer.TK_TEXT, lexer.TK_EOF}, kinds(toks))
	require.Equal(t, "after", toks[0].Value)
}

func TestLexer_WhitespaceStripLeft(t *testing.T) {
	toks, err := lexer.Tokenize("  {{- name }}")
	require.NoError(t, err)
	var start *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_OUTPUT_START {
			start = &toks[i]
		}
	}
	require.NotNil(t, start)
	require.True(t, start.StripLeft)
	// Preceding text whitespace should be removed
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			require.NotEqual(t, "  ", tk.Value, "whitespace before {{- should be stripped")
		}
	}
}

func TestLexer_WhitespaceStripRight(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name -}}  after")
	require.NoError(t, err)
	var end *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_OUTPUT_END {
			end = &toks[i]
		}
	}
	require.NotNil(t, end)
	require.True(t, end.StripRight)
	// Text after -}} should have leading whitespace stripped
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			require.NotEqual(t, "  after", tk.Value)
		}
	}
}

func TestLexer_TagBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{% block %}")
	require.NoError(t, err)
	require.Equal(t, []lexer.TokenKind{
		lexer.TK_TAG_START, lexer.TK_IDENT, lexer.TK_TAG_END, lexer.TK_EOF,
	}, kinds(toks))
	require.Equal(t, "block", toks[1].Value)
}

func TestLexer_RawBlock(t *testing.T) {
	toks, err := lexer.Tokenize("{% raw %}{{ not_parsed }}{% endraw %}")
	require.NoError(t, err)
	// raw block content should come out as a single TEXT token
	var textVal string
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TEXT {
			textVal = tk.Value
		}
	}
	require.Equal(t, "{{ not_parsed }}", textVal)
}

func TestLexer_Operators(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a + b - c * d / e % f ~ g }}")
	require.NoError(t, err)
	expected := []lexer.TokenKind{
		lexer.TK_PLUS, lexer.TK_MINUS, lexer.TK_STAR,
		lexer.TK_SLASH, lexer.TK_PERCENT, lexer.TK_TILDE,
	}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, op := range expected {
			if tk.Kind == op {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, expected, got)
}

func TestLexer_Comparison(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a == b != c < d <= e > f >= g }}")
	require.NoError(t, err)
	want := []lexer.TokenKind{lexer.TK_EQ, lexer.TK_NEQ, lexer.TK_LT, lexer.TK_LTE, lexer.TK_GT, lexer.TK_GTE}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, k := range want {
			if tk.Kind == k {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, want, got)
}

func TestLexer_Keywords(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a and b or not c }}")
	require.NoError(t, err)
	want := []lexer.TokenKind{lexer.TK_AND, lexer.TK_OR, lexer.TK_NOT}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, k := range want {
			if tk.Kind == k {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, want, got)
}

func TestLexer_TernaryTokens(t *testing.T) {
	toks, err := lexer.Tokenize("{{ a ? b : c }}")
	require.NoError(t, err)
	want := []lexer.TokenKind{lexer.TK_QUESTION, lexer.TK_COLON}
	var got []lexer.TokenKind
	for _, tk := range toks {
		for _, k := range want {
			if tk.Kind == k {
				got = append(got, tk.Kind)
			}
		}
	}
	require.Equal(t, want, got)
}

func TestLexer_IfElseAreIdents(t *testing.T) {
	toks, err := lexer.Tokenize("{{ if }}")
	require.NoError(t, err)
	for _, tk := range toks {
		if tk.Value == "if" {
			require.Equal(t, lexer.TK_IDENT, tk.Kind)
			return
		}
	}
	t.Fatal("no 'if' token found")
}

func TestLexer_BoolLiterals(t *testing.T) {
	toks, err := lexer.Tokenize("{{ true }} {{ false }}")
	require.NoError(t, err)
	var got []lexer.TokenKind
	for _, tk := range toks {
		if tk.Kind == lexer.TK_TRUE || tk.Kind == lexer.TK_FALSE {
			got = append(got, tk.Kind)
		}
	}
	require.Equal(t, []lexer.TokenKind{lexer.TK_TRUE, lexer.TK_FALSE}, got)
}

func TestLexer_StringLiteral(t *testing.T) {
	toks, err := lexer.Tokenize(`{{ "hello world" }}`)
	require.NoError(t, err)
	var str *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_STRING {
			str = &toks[i]
		}
	}
	require.NotNil(t, str)
	require.Equal(t, "hello world", str.Value)
}

func TestLexer_IntLiteral(t *testing.T) {
	toks, err := lexer.Tokenize("{{ 42 }}")
	require.NoError(t, err)
	var num *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_INT {
			num = &toks[i]
		}
	}
	require.NotNil(t, num)
	require.Equal(t, "42", num.Value)
}

func TestLexer_FloatLiteral(t *testing.T) {
	toks, err := lexer.Tokenize("{{ 3.14 }}")
	require.NoError(t, err)
	var num *lexer.Token
	for i := range toks {
		if toks[i].Kind == lexer.TK_FLOAT {
			num = &toks[i]
		}
	}
	require.NotNil(t, num)
	require.Equal(t, "3.14", num.Value)
}

func TestLexer_LineNumbers(t *testing.T) {
	toks, err := lexer.Tokenize("line1\n{{ name }}")
	require.NoError(t, err)
	for _, tk := range toks {
		if tk.Kind == lexer.TK_IDENT {
			require.Equal(t, 2, tk.Line)
			return
		}
	}
	t.Fatal("no IDENT token found")
}

func TestLexer_Filter(t *testing.T) {
	toks, err := lexer.Tokenize("{{ name | upcase }}")
	require.NoError(t, err)
	hasPipe := false
	for _, tk := range toks {
		if tk.Kind == lexer.TK_PIPE {
			hasPipe = true
		}
	}
	require.True(t, hasPipe)
}

func TestLexer_DotAccess(t *testing.T) {
	toks, err := lexer.Tokenize("{{ user.name }}")
	require.NoError(t, err)
	hasDot := false
	for _, tk := range toks {
		if tk.Kind == lexer.TK_DOT {
			hasDot = true
		}
	}
	require.True(t, hasDot)
}

func TestLexer_UnclosedOutput_Error(t *testing.T) {
	_, err := lexer.Tokenize("{{ unclosed")
	require.Error(t, err)
}

func TestLexer_UnclosedComment_Error(t *testing.T) {
	_, err := lexer.Tokenize("{# unclosed")
	require.Error(t, err)
}
