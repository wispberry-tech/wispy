// internal/lexer/lexer.go
package lexer

import (
	"fmt"
	"strings"
)

// Tokenize breaks src into tokens. Returns a ParseError on invalid syntax.
func Tokenize(src string) ([]Token, error) {
	l := &lx{src: src, line: 1, col: 1}
	return l.run()
}

// TokenizeLetBody tokenizes raw text as bare expression content (no delimiters).
// It produces the same token kinds as the inner tag scanner but operates on
// plain text (assignments, if/elif/else/end, expressions).
func TokenizeLetBody(src string) ([]Token, error) {
	l := &lx{src: src, line: 1, col: 1}
	for l.pos < len(l.src) {
		l.skipSpaces()
		if l.pos >= len(l.src) {
			break
		}
		if l.src[l.pos] == '\n' || l.src[l.pos] == '\r' {
			l.advance()
			continue
		}
		if err := l.lexOneToken(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Kind: TK_EOF, Line: l.line, Col: l.col})
	return l.tokens, nil
}

type lx struct {
	src       string
	pos       int
	line      int
	col       int
	tokens    []Token
	stripNext bool // when true, strip leading whitespace of the next TEXT token
}

// lexErr carries a line number for ParseError wrapping in engine.go.
type lexErr struct {
	line int
	msg  string
}

func (e *lexErr) Error() string { return fmt.Sprintf("line %d: %s", e.line, e.msg) }
func (e *lexErr) LexLine() int  { return e.line }

func (l *lx) run() ([]Token, error) {
	for l.pos < len(l.src) {
		if err := l.step(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Kind: TK_EOF, Line: l.line, Col: l.col})
	return l.tokens, nil
}

func (l *lx) step() error {
	if l.pos+1 < len(l.src) {
		pair := l.src[l.pos : l.pos+2]
		switch pair {
		case "{{":
			return l.lexOutput()
		case "{%":
			return l.lexTag()
		case "{#":
			return l.lexComment()
		}
	}
	l.lexText()
	return nil
}

// ─── Text ─────────────────────────────────────────────────────────────────────

func (l *lx) lexText() {
	start := l.pos
	startLine := l.line
	startCol := l.col
	for l.pos < len(l.src) {
		if l.pos+1 < len(l.src) {
			p := l.src[l.pos : l.pos+2]
			if p == "{{" || p == "{%" || p == "{#" {
				break
			}
		}
		l.advance()
	}
	if l.pos > start {
		text := l.src[start:l.pos]
		if l.stripNext {
			text = strings.TrimLeft(text, " \t\r\n")
			l.stripNext = false
		}
		if text != "" {
			l.tokens = append(l.tokens, Token{Kind: TK_TEXT, Value: text, Line: startLine, Col: startCol})
		}
	}
}

// ─── Output {{ }} ─────────────────────────────────────────────────────────────

func (l *lx) lexOutput() error {
	line, col := l.line, l.col
	l.pos += 2
	l.col += 2
	stripLeft := l.consumeIf('-')
	if stripLeft {
		l.stripLastTextRight()
	}
	l.tokens = append(l.tokens, Token{Kind: TK_OUTPUT_START, Value: "{{", Line: line, Col: col, StripLeft: stripLeft})
	return l.lexInner("}}")
}

// ─── Tag {% %} — with special handling for {% raw %} ─────────────────────────

func (l *lx) lexTag() error {
	line, col := l.line, l.col
	l.pos += 2
	l.col += 2
	stripLeft := l.consumeIf('-')
	if stripLeft {
		l.stripLastTextRight()
	}

	// Peek at tag name to detect {% raw %}
	savedPos, savedLine, savedCol := l.pos, l.line, l.col
	l.skipSpaces()
	if strings.HasPrefix(l.src[l.pos:], "raw") && !l.isIdentContinue(l.pos+3) {
		rawNameEnd := l.pos + 3
		l.pos = rawNameEnd
		l.col += 3
		l.skipSpaces()
		stripTagRight := l.consumeIf('-')
		if !l.hasPrefix("%}") {
			return &lexErr{line: line, msg: "expected %} after raw"}
		}
		l.pos += 2
		l.col += 2
		return l.lexRawContent(line, stripLeft, stripTagRight)
	}
	// Restore: not a raw tag
	l.pos, l.line, l.col = savedPos, savedLine, savedCol

	l.tokens = append(l.tokens, Token{Kind: TK_TAG_START, Value: "{%", Line: line, Col: col, StripLeft: stripLeft})
	return l.lexInner("%}")
}

func (l *lx) lexRawContent(startLine int, stripTagLeft, stripTagRight bool) error {
	if stripTagRight {
		l.stripNext = true
	}
	contentStart := l.pos
	for l.pos < len(l.src) {
		if l.hasPrefix("{%") {
			// Check for {% endraw %}
			saved := l.pos
			savedLine := l.line
			savedCol := l.col
			l.pos += 2
			l.col += 2
			_ = l.consumeIf('-')
			l.skipSpaces()
			if strings.HasPrefix(l.src[l.pos:], "endraw") && !l.isIdentContinue(l.pos+6) {
				content := l.src[contentStart:saved]
				l.pos += 6
				l.col += 6
				l.skipSpaces()
				stripR := l.consumeIf('-')
				if !l.hasPrefix("%}") {
					return &lexErr{line: l.line, msg: "expected %} after endraw"}
				}
				l.pos += 2
				l.col += 2
				if stripTagRight {
					content = strings.TrimLeft(content, " \t\r\n")
				}
				if stripR {
					content = strings.TrimRight(content, " \t\r\n")
				}
				if content != "" {
					l.tokens = append(l.tokens, Token{Kind: TK_TEXT, Value: content, Line: startLine + 1})
				}
				if stripR {
					l.stripNext = true
				}
				return nil
			}
			// Not endraw — restore and continue
			l.pos = saved
			l.line = savedLine
			l.col = savedCol
		}
		l.advance()
	}
	return &lexErr{line: startLine, msg: "unclosed raw block"}
}

// ─── Comment {# #} ────────────────────────────────────────────────────────────

func (l *lx) lexComment() error {
	line := l.line
	l.pos += 2
	l.col += 2
	for l.pos+1 < len(l.src) {
		if l.src[l.pos] == '#' && l.src[l.pos+1] == '}' {
			l.pos += 2
			l.col += 2
			return nil
		}
		l.advance()
	}
	return &lexErr{line: line, msg: "unclosed comment"}
}

// ─── Inner token scanner (shared by {{ }} and {% %}) ─────────────────────────

func (l *lx) lexInner(close string) error {
	for l.pos < len(l.src) {
		l.skipSpaces()
		// Check for close with optional strip: -}} or -%}
		stripRight := false
		if l.pos < len(l.src) && l.src[l.pos] == '-' && l.hasPrefix("-"+close) {
			stripRight = true
			l.pos++
			l.col++
		}
		if l.hasPrefix(close) {
			kind := TK_OUTPUT_END
			if close == "%}" {
				kind = TK_TAG_END
			}
			l.tokens = append(l.tokens, Token{Kind: kind, Value: close, Line: l.line, Col: l.col, StripRight: stripRight})
			l.pos += 2
			l.col += 2
			if stripRight {
				l.stripNext = true
			}
			return nil
		}
		if err := l.lexOneToken(); err != nil {
			return err
		}
	}
	return &lexErr{line: l.line, msg: "unexpected end of template, expected closing delimiter"}
}

func (l *lx) lexOneToken() error {
	if l.pos >= len(l.src) {
		return &lexErr{line: l.line, msg: "unexpected EOF"}
	}
	line, col := l.line, l.col
	ch := l.src[l.pos]

	switch {
	case ch == '"' || ch == '\'':
		return l.lexString(ch)
	case ch >= '0' && ch <= '9':
		return l.lexNumber()
	case ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
		return l.lexIdent()
	}

	// Two-char operators first
	if l.pos+1 < len(l.src) {
		two := l.src[l.pos : l.pos+2]
		var kind TokenKind
		switch two {
		case "==":
			kind = TK_EQ
		case "!=":
			kind = TK_NEQ
		case "<=":
			kind = TK_LTE
		case ">=":
			kind = TK_GTE
		}
		if kind != 0 {
			l.tokens = append(l.tokens, Token{Kind: kind, Value: two, Line: line, Col: col})
			l.pos += 2
			l.col += 2
			return nil
		}
	}

	// Single-char operators
	l.pos++
	l.col++
	var kind TokenKind
	switch ch {
	case '+':
		kind = TK_PLUS
	case '-':
		kind = TK_MINUS
	case '*':
		kind = TK_STAR
	case '/':
		kind = TK_SLASH
	case '%':
		kind = TK_PERCENT
	case '~':
		kind = TK_TILDE
	case '<':
		kind = TK_LT
	case '>':
		kind = TK_GT
	case '|':
		kind = TK_PIPE
	case '.':
		kind = TK_DOT
	case '[':
		kind = TK_LBRACKET
	case ']':
		kind = TK_RBRACKET
	case '(':
		kind = TK_LPAREN
	case ')':
		kind = TK_RPAREN
	case ',':
		kind = TK_COMMA
	case '=':
		kind = TK_ASSIGN
	case '?':
		kind = TK_QUESTION
	case ':':
		kind = TK_COLON
	case '{':
		kind = TK_LBRACE
	case '}':
		kind = TK_RBRACE
	default:
		return &lexErr{line: line, msg: fmt.Sprintf("unexpected character: %q", ch)}
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: string(ch), Line: line, Col: col})
	return nil
}

func (l *lx) lexString(quote byte) error {
	line, col := l.line, l.col
	l.pos++
	l.col++
	var buf strings.Builder
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == quote {
			l.pos++
			l.col++
			l.tokens = append(l.tokens, Token{Kind: TK_STRING, Value: buf.String(), Line: line, Col: col})
			return nil
		}
		if ch == '\\' && l.pos+1 < len(l.src) {
			l.pos++
			l.col++
			switch l.src[l.pos] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			case '\'':
				buf.WriteByte('\'')
			default:
				buf.WriteByte('\\')
				buf.WriteByte(l.src[l.pos])
			}
			l.pos++
			l.col++
			continue
		}
		if ch == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		buf.WriteByte(ch)
		l.pos++
	}
	return &lexErr{line: line, msg: "unclosed string literal"}
}

func (l *lx) lexNumber() error {
	line, col := l.line, l.col
	start := l.pos
	isFloat := false
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch >= '0' && ch <= '9' {
			l.pos++
			l.col++
		} else if ch == '.' && !isFloat &&
			l.pos+1 < len(l.src) && l.src[l.pos+1] >= '0' && l.src[l.pos+1] <= '9' {
			isFloat = true
			l.pos++
			l.col++
		} else {
			break
		}
	}
	kind := TK_INT
	if isFloat {
		kind = TK_FLOAT
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: l.src[start:l.pos], Line: line, Col: col})
	return nil
}

func (l *lx) lexIdent() error {
	line, col := l.line, l.col
	start := l.pos
	for l.pos < len(l.src) && l.isIdentChar(l.pos) {
		l.pos++
		l.col++
	}
	val := l.src[start:l.pos]
	kind := TK_IDENT
	switch val {
	case "and":
		kind = TK_AND
	case "or":
		kind = TK_OR
	case "not":
		kind = TK_NOT
	case "true":
		kind = TK_TRUE
	case "false":
		kind = TK_FALSE
	case "nil", "null":
		kind = TK_NIL
	case "in":
		kind = TK_IN
	}
	l.tokens = append(l.tokens, Token{Kind: kind, Value: val, Line: line, Col: col})
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (l *lx) advance() {
	if l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *lx) skipSpaces() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *lx) consumeIf(ch byte) bool {
	if l.pos < len(l.src) && l.src[l.pos] == ch {
		l.pos++
		l.col++
		return true
	}
	return false
}

func (l *lx) hasPrefix(s string) bool {
	return strings.HasPrefix(l.src[l.pos:], s)
}

func (l *lx) isIdentChar(pos int) bool {
	if pos >= len(l.src) {
		return false
	}
	ch := l.src[pos]
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func (l *lx) isIdentContinue(pos int) bool {
	return l.isIdentChar(pos)
}

func (l *lx) stripLastTextRight() {
	for i := len(l.tokens) - 1; i >= 0; i-- {
		if l.tokens[i].Kind == TK_TEXT {
			l.tokens[i].Value = strings.TrimRight(l.tokens[i].Value, " \t\r\n")
			if l.tokens[i].Value == "" {
				l.tokens = append(l.tokens[:i], l.tokens[i+1:]...)
			}
			return
		}
		break // stop at first non-text token
	}
}
