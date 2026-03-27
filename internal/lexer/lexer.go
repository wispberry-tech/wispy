package lexer

import (
	"unicode"
)

// Lexer holds the state of the scanner.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
	inWispStmt   bool // whether we're inside a {% %} block
}

// NewLexer returns a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar reads the next character and advances the position.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// skipWhitespace skips over whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

// isLetter checks if the byte is a letter or underscore.
func isLetter(ch byte) bool {
	return ch == '_' || unicode.IsLetter(rune(ch))
}

// isDigit checks if the byte is a digit.
func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// If we're not in a Wisp statement, check for text content or start of Wisp statement
	if !l.inWispStmt {
		// Check if we're at the start of a Wisp statement
		if l.ch == '{' && l.peekChar() == '%' {
			l.readChar() // consume '{'
			l.readChar() // consume '%'
			l.inWispStmt = true
			tok = NewToken(LBRACE_PCT, "{%", l.line, l.column)
			return tok
		}

		// Otherwise, read text content until we find {% or EOF
		if l.ch != 0 {
			position := l.position
			for l.ch != 0 && !(l.ch == '{' && l.peekChar() == '%') {
				l.readChar()
			}
			tok.Type = TEXT
			tok.Literal = l.input[position:l.position]
			tok.Line = l.line
			tok.Column = l.column
			return tok
		}
	}

	// We're in a Wisp statement, tokenize normally
	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar() // consume first '='
			l.readChar() // consume second '='
			tok = NewToken(EQ, "==", l.line, l.column)
		} else {
			tok = NewToken(ASSIGN_OP, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case '+':
		tok = NewToken(PLUS, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '-':
		tok = NewToken(MINUS, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '!':
		if l.peekChar() == '=' {
			l.readChar() // consume '!'
			l.readChar() // consume '='
			tok = NewToken(NOT_EQ, "!=", l.line, l.column)
		} else {
			tok = NewToken(BANG, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case '*':
		tok = NewToken(ASTERISK, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '/':
		tok = NewToken(SLASH, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '<':
		if l.peekChar() == '=' {
			l.readChar() // consume '<'
			l.readChar() // consume '='
			tok = NewToken(LTE, "<=", l.line, l.column)
		} else {
			tok = NewToken(LT, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case '>':
		if l.peekChar() == '=' {
			l.readChar() // consume '>'
			l.readChar() // consume '='
			tok = NewToken(GTE, ">=", l.line, l.column)
		} else {
			tok = NewToken(GT, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case ',':
		tok = NewToken(COMMA, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case ';':
		tok = NewToken(SEMICOLON, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '(':
		tok = NewToken(LPAREN, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case ')':
		tok = NewToken(RPAREN, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '{':
		if l.peekChar() == '%' {
			l.readChar() // consume '{'
			l.readChar() // consume '%'
			l.inWispStmt = true
			tok = NewToken(LBRACE_PCT, "{%", l.line, l.column)
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case '}':
		if l.peekChar() == '%' {
			l.readChar() // consume '}'
			l.readChar() // consume '%'
			l.inWispStmt = false
			tok = NewToken(RBRACE_PCT, "%}", l.line, l.column)
		} else {
			tok = NewToken(RBRACE, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	case '[':
		tok = NewToken(LBRACKET, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case ']':
		tok = NewToken(RBRACKET, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '"':
		position := l.position + 1 // skip opening quote
		for {
			l.readChar()
			if l.ch == '"' || l.ch == 0 {
				break
			}
		}
		tok.Type = STRING
		tok.Literal = l.input[position:l.position]
		tok.Line = l.line
		tok.Column = l.column
		l.readChar() // Advance past closing quote
		return tok
	case '.':
		tok = NewToken(DOT, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '|':
		tok = NewToken(PIPE, string(l.ch), l.line, l.column)
		l.readChar()
		return tok
	case '%':
		if l.peekChar() == '}' {
			l.readChar() // consume '%'
			l.readChar() // consume '}'
			l.inWispStmt = false
			tok = NewToken(RBRACE_PCT, "%}", l.line, l.column)
			return tok
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), l.line, l.column)
			l.readChar()
			return tok
		}
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = l.lookupIdent(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column
			// Don't call readChar() here - readIdentifier() already positioned us at the next char
			return tok
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			tok.Line = l.line
			tok.Column = l.column
			// Don't call readChar() here - readNumber() already positioned us at the next char
			return tok
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), l.line, l.column)
		}
		l.readChar()
		return tok
	}
}

// readIdentifier reads an identifier (sequence of letters and digits).
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (sequence of digits).
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// lookupIdent returns the token type for an identifier, or IDENT if not a keyword.
func (l *Lexer) lookupIdent(ident string) TokenType {
	switch ident {
	case "let":
		return LET
	case "if":
		return IF
	case "else":
		return ELSE
	case "elsif":
		return ELSIF
	case "unless":
		return UNLESS
	case "end":
		return END
	case "true":
		return TRUE
	case "false":
		return FALSE
	case "assign":
		return ASSIGN
	case "for":
		return FOR
	case "while":
		return WHILE
	case "range":
		return RANGE
	case "case":
		return CASE
	case "when":
		return WHEN
	case "with":
		return WITH
	case "cycle":
		return CYCLE
	case "increment":
		return INCREMENT
	case "decrement":
		return DECREMENT
	case "break":
		return BREAK
	case "continue":
		return CONTINUE
	case "include":
		return INCLUDE
	case "render":
		return RENDER
	case "component":
		return COMPONENT
	case "extends":
		return EXTENDS
	case "block":
		return BLOCK
	case "content":
		return CONTENT
	case "raw":
		return RAW
	case "comment":
		return COMMENT
	case "as":
		return AS
	case "in":
		return IN
	case "endraw":
		return ENDRAW
	case "endcomment":
		return ENDCOMMENT
	}
	return IDENT
}

// GetPosition returns the current lexer position.
func (l *Lexer) GetPosition() int {
	return l.position
}

// SetPosition sets the lexer position and updates internal state.
func (l *Lexer) SetPosition(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(l.input) {
		pos = len(l.input)
	}
	l.position = pos
	l.readPosition = pos + 1
	if pos < len(l.input) {
		l.ch = l.input[pos]
	} else {
		l.ch = 0
	}
}

// GetInput returns the lexer's input string.
func (l *Lexer) GetInput() string {
	return l.input
}

// CaptureRawText captures raw text from the current position until {% endraw %} is found.
func (l *Lexer) CaptureRawText() string {
	startPos := l.position
	if startPos >= len(l.input) {
		return ""
	}

	// Scan forward to find {% endraw %}
	for i := startPos; i < len(l.input)-7; i++ {
		if l.input[i] != '{' || l.input[i+1] != '%' {
			continue
		}

		// Found a {% block - check if it's {% endraw %}
		j := i + 2
		// Skip whitespace
		for j < len(l.input) && (l.input[j] == ' ' || l.input[j] == '\t' || l.input[j] == '\n' || l.input[j] == '\r') {
			j++
		}

		// Check for "endraw" (6 characters)
		if j+6 <= len(l.input) && l.input[j:j+6] == "endraw" {
			// Found {% endraw %}, capture everything before it
			rawContent := l.input[startPos:i]

			// Advance lexer position past {% endraw %}
			j += 6
			// Skip whitespace
			for j < len(l.input) && (l.input[j] == ' ' || l.input[j] == '\t' || l.input[j] == '\n' || l.input[j] == '\r') {
				j++
			}
			// Look for %}
			if j+1 < len(l.input) && l.input[j] == '%' && l.input[j+1] == '}' {
				j += 2
			}

			l.position = j
			l.readPosition = j + 1
			if j < len(l.input) {
				l.ch = l.input[j]
			} else {
				l.ch = 0
			}
			return rawContent
		}
	}

	// No {% endraw %} found, return everything until EOF
	return l.input[startPos:]
}
