// internal/lexer/token.go
package lexer

// TokenKind identifies the category of a lexed token.
type TokenKind uint8

const (
	TK_EOF          TokenKind = iota
	TK_TEXT                   // raw text between delimiters
	TK_TAG_START              // {% or {%-
	TK_TAG_END                // %} or -%}
	// Literals
	TK_STRING // "..." or '...'
	TK_INT    // 123
	TK_FLOAT  // 1.23
	TK_TRUE   // true
	TK_FALSE  // false
	TK_NIL    // nil / null
	// Identifier
	TK_IDENT // foo, bar_baz, _priv
	// Punctuation
	TK_DOT      // .
	TK_LBRACKET // [
	TK_RBRACKET // ]
	TK_LPAREN   // (
	TK_RPAREN   // )
	TK_COMMA    // ,
	TK_PIPE     // |
	TK_ASSIGN   // = (named args)
	// Arithmetic
	TK_PLUS    // +
	TK_MINUS   // -
	TK_STAR    // *
	TK_SLASH   // /
	TK_PERCENT // %
	TK_TILDE   // ~ (string concat)
	// Comparison
	TK_EQ  // ==
	TK_NEQ // !=
	TK_LT  // <
	TK_LTE // <=
	TK_GT  // >
	TK_GTE // >=
	// Boolean keywords
	TK_AND  // and
	TK_OR   // or
	TK_NOT  // not
	TK_QUESTION // ?  (ternary)
	TK_COLON    // :  (ternary)
	TK_LBRACE   // { (map literal)
	TK_RBRACE   // } (map literal)
	TK_IN   // in   (for...in)
	// Svelte-style sigil tokens inside {% %}
	TK_BLOCK_OPEN   // #keyword  (Value = keyword, e.g. "if", "each", "fill")
	TK_BLOCK_BRANCH // :keyword  (Value = keyword, e.g. "else", "empty")
	TK_BLOCK_CLOSE  // /keyword  (Value = keyword, e.g. "if", "each", "fill")
	// PascalCase element tokens (components only)
	TK_ELEMENT_OPEN  // <Name (Value = element name)
	TK_ELEMENT_CLOSE // </Name> (Value = element name)
	TK_ELEMENT_END   // >
	TK_SELF_CLOSE    // />
)

// Token is a single lexed unit.
type Token struct {
	Kind       TokenKind
	Value      string // raw text value (identifier name, string content, number digits)
	Line       int    // 1-based line number
	Col        int    // 1-based column number
	StripLeft  bool   // {{- or {%-: strip whitespace to the left
	StripRight bool   // -}} or -%}: strip whitespace to the right
}
