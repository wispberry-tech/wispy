// internal/parser/parser.go
package parser

import (
	"fmt"
	"strconv"

	"grove/internal/ast"
	"grove/internal/groverrors"
	"grove/internal/lexer"
)

// Parse converts a token stream into an AST.
// inline=true forbids {% extends %} and {% import %} (used by RenderTemplate).
// allowedTags is an optional whitelist of permitted tag names (nil = all allowed).
func Parse(tokens []lexer.Token, inline bool, allowedTags ...map[string]bool) (*ast.Program, error) {
	p := &parser{tokens: tokens, inline: inline}
	if len(allowedTags) > 0 && allowedTags[0] != nil {
		p.allowedTags = allowedTags[0]
	}
	return p.parseProgram()
}

type parser struct {
	tokens      []lexer.Token
	pos         int
	inline      bool
	allowedTags map[string]bool // nil = all allowed; non-nil = whitelist
}

// ─── Program ──────────────────────────────────────────────────────────────────

func (p *parser) parseProgram() (*ast.Program, error) {
	prog := &ast.Program{}
	for !p.atEOF() {
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			prog.Body = append(prog.Body, node)
		}
	}
	return prog, nil
}

func (p *parser) parseNode() (ast.Node, error) {
	tk := p.peek()
	switch tk.Kind {
	case lexer.TK_TEXT:
		p.advance()
		return &ast.TextNode{Value: tk.Value, Line: tk.Line}, nil
	case lexer.TK_OUTPUT_START:
		return p.parseOutput()
	case lexer.TK_TAG_START:
		return p.parseTag()
	case lexer.TK_EOF:
		return nil, nil
	default:
		return nil, p.errorf(tk.Line, tk.Col, "unexpected token %q", tk.Value)
	}
}

// parseBody reads nodes until one of the stopTags is the current tag name.
// It does NOT consume the stop tag itself.
func (p *parser) parseBody(stopTags ...string) ([]ast.Node, error) {
	var nodes []ast.Node
	for !p.atEOF() {
		// Peek at next tag name to detect stop conditions
		if p.peek().Kind == lexer.TK_TAG_START {
			name, ok := p.peekTagName()
			if ok {
				for _, stop := range stopTags {
					if name == stop {
						return nodes, nil
					}
				}
			}
		}
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// peekTagName returns the tag name of the upcoming {% name ... %} without consuming it.
func (p *parser) peekTagName() (string, bool) {
	if p.pos+1 < len(p.tokens) {
		return tokenTagName(p.tokens[p.pos+1])
	}
	return "", false
}

// tokenTagName extracts the string tag name from a token (handles keywords used as tag names).
func tokenTagName(tk lexer.Token) (string, bool) {
	switch tk.Kind {
	case lexer.TK_IDENT:
		return tk.Value, true
	case lexer.TK_IF:
		return "if", true
	case lexer.TK_ELSE:
		return "else", true
	case lexer.TK_NOT:
		return "not", true
	case lexer.TK_IN:
		return "in", true
	}
	return "", false
}

// ─── Output {{ expr }} ────────────────────────────────────────────────────────

func (p *parser) parseOutput() (*ast.OutputNode, error) {
	start := p.advance() // consume OUTPUT_START
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Kind != lexer.TK_OUTPUT_END {
		return nil, p.errorf(end.Line, end.Col, "expected }}, got %q", end.Value)
	}
	p.advance() // consume OUTPUT_END
	return &ast.OutputNode{
		Expr:       expr,
		StripLeft:  start.StripLeft,
		StripRight: end.StripRight,
		Line:       start.Line,
	}, nil
}

// ─── Tags {% name ... %} ──────────────────────────────────────────────────────

func (p *parser) parseTag() (ast.Node, error) {
	tagStart := p.advance() // consume TAG_START

	nameTok := p.peek()
	name, ok := tokenTagName(nameTok)
	if !ok {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected tag name after {%%")
	}

	// Sandbox: check allowed tags whitelist (skip internal close-tags like endif, endfor, etc.)
	if p.allowedTags != nil && !isCloseTag(name) {
		if !p.allowedTags[name] {
			return nil, &groverrors.ParseError{
				Line:    nameTok.Line,
				Column:  nameTok.Col,
				Message: fmt.Sprintf("sandbox: tag %q is not allowed", name),
			}
		}
	}

	switch name {
	case "raw":
		p.advance() // consume "raw"
		if p.peek().Kind != lexer.TK_TAG_END {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected %%} after raw")
		}
		p.advance()
		return p.consumeUntilEndraw(tagStart)

	case "extends":
		return p.parseExtends(tagStart)

	case "block":
		return p.parseBlock(tagStart)

	case "if":
		return p.parseIf(tagStart)

	case "unless":
		return nil, &groverrors.ParseError{
			Line:    tagStart.Line,
			Column:  tagStart.Col,
			Message: "unknown tag \"unless\": use {% if not ... %} instead",
		}

	case "for":
		return p.parseFor(tagStart)

	case "set":
		return p.parseSet(tagStart)

	case "with":
		return nil, &groverrors.ParseError{
			Line:    tagStart.Line,
			Column:  tagStart.Col,
			Message: `unknown tag "with": use {% let %} or {% set %} instead`,
		}

	case "capture":
		return p.parseCapture(tagStart)

	case "macro":
		return p.parseMacro(tagStart)

	case "call":
		return p.parseCall(tagStart)

	case "include":
		return p.parseInclude(tagStart)

	case "render":
		return p.parseRender(tagStart)

	case "import":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    tagStart.Line,
				Column:  tagStart.Col,
				Message: "import not allowed in inline templates",
			}
		}
		return p.parseImport(tagStart)

	case "component":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    tagStart.Line,
				Column:  tagStart.Col,
				Message: "component not allowed in inline templates",
			}
		}
		return p.parseComponent(tagStart)

	case "slot":
		return p.parseSlot(tagStart)

	case "props":
		return p.parseProps(tagStart)

	case "asset":
		if p.inline {
			return nil, &groverrors.ParseError{
				Line:    tagStart.Line,
				Column:  tagStart.Col,
				Message: "{% asset %} not allowed in inline templates",
			}
		}
		return p.parseAsset(tagStart)

	case "meta":
		return p.parseMeta(tagStart)

	case "hoist":
		return p.parseHoist(tagStart)

	default:
		return p.consumeTagRemainder(name, tagStart)
	}
}

// ─── {% if %} ─────────────────────────────────────────────────────────────────

func (p *parser) parseIf(tagStart lexer.Token) (*ast.IfNode, error) {
	p.advance() // consume "if" token
	cond, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	node := &ast.IfNode{Condition: cond, Line: tagStart.Line}

	// Parse body until elif/else/endif
	node.Body, err = p.parseBody("elif", "else", "endif")
	if err != nil {
		return nil, err
	}

	// Parse elif/else chains
	for {
		tagName, _ := p.peekTagName()
		if tagName == "elif" {
			p.advance() // TAG_START
			p.advance() // "elif"
			elifCond, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if err := p.expectTagEnd(); err != nil {
				return nil, err
			}
			body, err := p.parseBody("elif", "else", "endif")
			if err != nil {
				return nil, err
			}
			node.Elifs = append(node.Elifs, ast.ElifClause{Condition: elifCond, Body: body})
		} else if tagName == "else" {
			p.advance() // TAG_START
			p.advance() // "else"
			if err := p.expectTagEnd(); err != nil {
				return nil, err
			}
			node.Else, err = p.parseBody("endif")
			if err != nil {
				return nil, err
			}
			break
		} else {
			break
		}
	}

	// Consume {% endif %}
	if err := p.expectTag("endif"); err != nil {
		return nil, err
	}
	return node, nil
}

// ─── {% for %} ────────────────────────────────────────────────────────────────

func (p *parser) parseFor(tagStart lexer.Token) (*ast.ForNode, error) {
	p.advance() // consume "for"

	// Parse variable name(s)
	var1Tok := p.advance()
	if var1Tok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(var1Tok.Line, var1Tok.Col, "expected loop variable name after for")
	}
	var1 := var1Tok.Value

	var var2 string
	if p.peek().Kind == lexer.TK_COMMA {
		p.advance() // consume comma
		var2Tok := p.advance()
		if var2Tok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(var2Tok.Line, var2Tok.Col, "expected second loop variable name after ,")
		}
		var2 = var2Tok.Value
	}

	// Expect "in"
	inTok := p.advance()
	if inTok.Kind != lexer.TK_IN {
		return nil, p.errorf(inTok.Line, inTok.Col, "expected 'in' after loop variable(s)")
	}

	iterable, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	body, err := p.parseBody("empty", "endfor")
	if err != nil {
		return nil, err
	}

	var emptyBody []ast.Node
	tagName, _ := p.peekTagName()
	if tagName == "empty" {
		p.advance() // TAG_START
		p.advance() // "empty"
		if err := p.expectTagEnd(); err != nil {
			return nil, err
		}
		emptyBody, err = p.parseBody("endfor")
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectTag("endfor"); err != nil {
		return nil, err
	}

	return &ast.ForNode{
		Var1:     var1,
		Var2:     var2,
		Iterable: iterable,
		Body:     body,
		Empty:    emptyBody,
		Line:     tagStart.Line,
	}, nil
}

// ─── {% set %} ────────────────────────────────────────────────────────────────

func (p *parser) parseSet(tagStart lexer.Token) (*ast.SetNode, error) {
	p.advance() // consume "set"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected variable name after set")
	}
	eqTok := p.advance()
	if eqTok.Kind != lexer.TK_ASSIGN {
		return nil, p.errorf(eqTok.Line, eqTok.Col, "expected = after variable name in set")
	}
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.SetNode{Name: nameTok.Value, Expr: expr, Line: tagStart.Line}, nil
}

// ─── {% capture %} ────────────────────────────────────────────────────────────

func (p *parser) parseCapture(tagStart lexer.Token) (*ast.CaptureNode, error) {
	p.advance() // consume "capture"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected variable name after capture")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endcapture")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endcapture"); err != nil {
		return nil, err
	}
	return &ast.CaptureNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}

// ─── Expression parsing (Pratt) ───────────────────────────────────────────────

func (p *parser) parseExpr(minPrec int) (ast.Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tk := p.peek()
		prec, isInfix := infixPrec(tk.Kind)
		if !isInfix || prec <= minPrec {
			break
		}

		switch tk.Kind {
		case lexer.TK_IF:
			p.advance() // consume if
			cond, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_ELSE {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected 'else' in ternary expression")
			}
			p.advance() // consume else
			alt, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			left = &ast.TernaryExpr{
				Condition:   cond,
				Consequence: left,
				Alternative: alt,
				Line:        tk.Line,
			}

		case lexer.TK_PIPE:
			p.advance() // consume |
			left, err = p.parseFilter(left)
			if err != nil {
				return nil, err
			}

		case lexer.TK_DOT:
			p.advance() // consume .
			attr := p.peek()
			if attr.Kind != lexer.TK_IDENT {
				return nil, p.errorf(attr.Line, attr.Col, "expected attribute name after .")
			}
			p.advance()
			left = &ast.AttributeAccess{Object: left, Key: attr.Value, Line: attr.Line}

		case lexer.TK_LBRACKET:
			p.advance() // consume [
			idx, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_RBRACKET {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ]")
			}
			p.advance()
			left = &ast.IndexAccess{Object: left, Key: idx, Line: tk.Line}

		case lexer.TK_LPAREN:
			// Function/macro call: identifier(args...) or obj.method(args...)
			p.advance() // consume (
			posArgs, namedArgs, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			// Distinguish built-in functions from macro calls
			if ident, ok := left.(*ast.Identifier); ok {
				switch ident.Name {
				case "range":
					if len(namedArgs) > 0 {
						return nil, p.errorf(tk.Line, tk.Col, "range() does not accept named arguments")
					}
					left = &ast.FuncCallNode{Name: "range", Args: posArgs, Line: ident.Line}
				case "caller":
					if len(posArgs)+len(namedArgs) > 0 {
						return nil, p.errorf(tk.Line, tk.Col, "caller() takes no arguments")
					}
					left = &ast.FuncCallNode{Name: "caller", Args: nil, Line: ident.Line}
				case "super":
					if len(posArgs)+len(namedArgs) > 0 {
						return nil, p.errorf(tk.Line, tk.Col, "super() takes no arguments")
					}
					left = &ast.FuncCallNode{Name: "super", Args: nil, Line: ident.Line}
				default:
					left = &ast.MacroCallExpr{Callee: left, PosArgs: posArgs, NamedArgs: namedArgs, Line: ident.Line}
				}
			} else {
				// AttributeAccess callee: forms.input(...)
				left = &ast.MacroCallExpr{Callee: left, PosArgs: posArgs, NamedArgs: namedArgs, Line: tk.Line}
			}

		default:
			p.advance()
			right, err := p.parseExpr(prec)
			if err != nil {
				return nil, err
			}
			left = &ast.BinaryExpr{Op: tk.Value, Left: left, Right: right, Line: tk.Line}
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (ast.Node, error) {
	tk := p.peek()
	switch tk.Kind {
	case lexer.TK_NOT:
		p.advance()
		// not has precedence 30 (below comparisons at 40, above and/or) so
		// parse the operand at prec=30 to allow postfix operators like .attr and [idx]
		operand, err := p.parseExpr(30)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "not", Operand: operand, Line: tk.Line}, nil
	case lexer.TK_MINUS:
		p.advance()
		// unary minus binds tighter than binary ops; use prec=70 (same as * / %)
		operand, err := p.parseExpr(70)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: "-", Operand: operand, Line: tk.Line}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (ast.Node, error) {
	tk := p.advance()
	switch tk.Kind {
	case lexer.TK_NIL:
		return &ast.NilLiteral{Line: tk.Line}, nil
	case lexer.TK_TRUE:
		return &ast.BoolLiteral{Value: true, Line: tk.Line}, nil
	case lexer.TK_FALSE:
		return &ast.BoolLiteral{Value: false, Line: tk.Line}, nil
	case lexer.TK_STRING:
		return &ast.StringLiteral{Value: tk.Value, Line: tk.Line}, nil
	case lexer.TK_INT:
		n, err := strconv.ParseInt(tk.Value, 10, 64)
		if err != nil {
			return nil, p.errorf(tk.Line, tk.Col, "invalid integer: %s", tk.Value)
		}
		return &ast.IntLiteral{Value: n, Line: tk.Line}, nil
	case lexer.TK_FLOAT:
		f, err := strconv.ParseFloat(tk.Value, 64)
		if err != nil {
			return nil, p.errorf(tk.Line, tk.Col, "invalid float: %s", tk.Value)
		}
		return &ast.FloatLiteral{Value: f, Line: tk.Line}, nil
	case lexer.TK_IDENT:
		return &ast.Identifier{Name: tk.Value, Line: tk.Line}, nil
	case lexer.TK_LPAREN:
		expr, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().Kind != lexer.TK_RPAREN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected )")
		}
		p.advance()
		return expr, nil
	default:
		return nil, p.errorf(tk.Line, tk.Col, "unexpected token in expression: %q", tk.Value)
	}
}

func (p *parser) parseFilter(value ast.Node) (ast.Node, error) {
	name := p.peek()
	if name.Kind != lexer.TK_IDENT {
		return nil, p.errorf(name.Line, name.Col, "expected filter name after |")
	}
	p.advance()

	var args []ast.Node
	if p.peek().Kind == lexer.TK_LPAREN {
		p.advance() // consume (
		for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
			arg, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.peek().Kind == lexer.TK_COMMA {
				p.advance()
			}
		}
		if p.peek().Kind != lexer.TK_RPAREN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after filter arguments")
		}
		p.advance()
	}

	return &ast.FilterExpr{
		Value:  value,
		Filter: name.Value,
		Args:   args,
		Line:   name.Line,
	}, nil
}

func infixPrec(k lexer.TokenKind) (int, bool) {
	switch k {
	case lexer.TK_IF:
		return 5, true
	case lexer.TK_OR:
		return 10, true
	case lexer.TK_AND:
		return 20, true
	case lexer.TK_EQ, lexer.TK_NEQ, lexer.TK_LT, lexer.TK_LTE, lexer.TK_GT, lexer.TK_GTE:
		return 40, true
	case lexer.TK_TILDE:
		return 50, true
	case lexer.TK_PLUS, lexer.TK_MINUS:
		return 60, true
	case lexer.TK_STAR, lexer.TK_SLASH, lexer.TK_PERCENT:
		return 70, true
	case lexer.TK_PIPE:
		return 90, true
	case lexer.TK_DOT, lexer.TK_LBRACKET, lexer.TK_LPAREN:
		return 100, true
	}
	return 0, false
}

// ─── Tag helpers ──────────────────────────────────────────────────────────────

// consumeTagRemainder skips to TAG_END and emits a TagNode.
func (p *parser) consumeTagRemainder(name string, tagStart lexer.Token) (ast.Node, error) {
	p.advance() // consume tag name
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		p.advance()
	}
	if p.peek().Kind == lexer.TK_TAG_END {
		p.advance()
	}
	return &ast.TagNode{Name: name, Line: tagStart.Line}, nil
}

func (p *parser) consumeUntilEndraw(tagStart lexer.Token) (ast.Node, error) {
	var content string
	for !p.atEOF() {
		tk := p.peek()
		if tk.Kind == lexer.TK_TAG_START {
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == lexer.TK_IDENT &&
				p.tokens[p.pos+1].Value == "endraw" {
				p.advance()
				p.advance()
				if p.peek().Kind == lexer.TK_TAG_END {
					p.advance()
				}
				return &ast.RawNode{Value: content, Line: tagStart.Line}, nil
			}
		}
		if tk.Kind == lexer.TK_TEXT {
			content += tk.Value
		}
		p.advance()
	}
	return nil, p.errorf(tagStart.Line, tagStart.Col, "unclosed raw block")
}

// expectTagEnd consumes the closing %} of the current tag.
func (p *parser) expectTagEnd() error {
	if p.peek().Kind != lexer.TK_TAG_END {
		return p.errorf(p.peek().Line, p.peek().Col, "expected %%} got %q", p.peek().Value)
	}
	p.advance()
	return nil
}

// expectTag consumes a full {% name %} tag and errors if name doesn't match.
func (p *parser) expectTag(name string) error {
	if p.peek().Kind != lexer.TK_TAG_START {
		return p.errorf(p.peek().Line, p.peek().Col, "expected {%% %s %%}", name)
	}
	p.advance() // TAG_START
	tok := p.peek()
	tokName, ok := tokenTagName(tok)
	if !ok || tokName != name {
		return p.errorf(tok.Line, tok.Col, "expected tag %q, got %q", name, tok.Value)
	}
	p.advance() // tag name
	// skip any remaining tokens until TAG_END (handles end tags with no content)
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		p.advance()
	}
	if p.peek().Kind == lexer.TK_TAG_END {
		p.advance()
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (p *parser) peek() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Kind: lexer.TK_EOF}
}

func (p *parser) advance() lexer.Token {
	tk := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tk
}

func (p *parser) atEOF() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Kind == lexer.TK_EOF
}

func (p *parser) errorf(line, col int, format string, args ...any) *groverrors.ParseError {
	return &groverrors.ParseError{
		Line:    line,
		Column:  col,
		Message: fmt.Sprintf(format, args...),
	}
}

// isCloseTag returns true for closing/structural tags that should bypass the allowed-tags check.
// These are tags that are always needed as syntactic closers (e.g. endif, endfor, else, etc.).
func isCloseTag(name string) bool {
	switch name {
	case "endif", "endfor", "endcapture", "endmacro", "endcall",
		"endblock", "endslot", "endcomponent", "endfill", "endhoist",
		"else", "elif", "empty", "endraw":
		return true
	}
	return false
}

// ─── Plan 4: Macro + composition parser methods ───────────────────────────────

// parseCallArgs parses the argument list inside ( ) of a macro/function call.
// Returns positional args (in order) and named args (key=value).
// Positional args must come before named args.
func (p *parser) parseCallArgs() (posArgs []ast.Node, namedArgs []ast.NamedArgNode, err error) {
	for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
		// Named arg: ident = expr (look-ahead two tokens)
		if p.peek().Kind == lexer.TK_IDENT &&
			p.pos+1 < len(p.tokens) &&
			p.tokens[p.pos+1].Kind == lexer.TK_ASSIGN {
			keyTok := p.advance() // consume ident
			p.advance()           // consume =
			val, e := p.parseExpr(0)
			if e != nil {
				return nil, nil, e
			}
			namedArgs = append(namedArgs, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		} else {
			if len(namedArgs) > 0 {
				return nil, nil, p.errorf(p.peek().Line, p.peek().Col, "positional argument after named argument")
			}
			arg, e := p.parseExpr(0)
			if e != nil {
				return nil, nil, e
			}
			posArgs = append(posArgs, arg)
		}
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if p.peek().Kind != lexer.TK_RPAREN {
		return nil, nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after arguments")
	}
	p.advance() // consume )
	return posArgs, namedArgs, nil
}

// parseMacroParams parses the parameter list of a macro definition: (p1, p2="default")
func (p *parser) parseMacroParams() ([]ast.MacroParam, error) {
	if p.peek().Kind != lexer.TK_LPAREN {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ( after macro name")
	}
	p.advance() // consume (
	var params []ast.MacroParam
	for p.peek().Kind != lexer.TK_RPAREN && !p.atEOF() {
		nameTok := p.advance()
		if nameTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(nameTok.Line, nameTok.Col, "expected parameter name in macro definition")
		}
		param := ast.MacroParam{Name: nameTok.Value}
		if p.peek().Kind == lexer.TK_ASSIGN {
			p.advance() // consume =
			def, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			param.Default = def
		}
		params = append(params, param)
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if p.peek().Kind != lexer.TK_RPAREN {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ) after macro parameters")
	}
	p.advance() // consume )
	return params, nil
}

// parseMacro parses {% macro name(params) %}...{% endmacro %}.
func (p *parser) parseMacro(tagStart lexer.Token) (*ast.MacroNode, error) {
	p.advance() // consume "macro"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected macro name after macro")
	}
	params, err := p.parseMacroParams()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endmacro")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endmacro"); err != nil {
		return nil, err
	}
	return &ast.MacroNode{Name: nameTok.Value, Params: params, Body: body, Line: tagStart.Line}, nil
}

// parseCall parses {% call macro(args) %}body{% endcall %}.
func (p *parser) parseCall(tagStart lexer.Token) (*ast.CallNode, error) {
	p.advance() // consume "call"
	callee, err := p.parseExpr(90)
	if err != nil {
		return nil, err
	}
	mc, ok := callee.(*ast.MacroCallExpr)
	if !ok {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "{%% call %%} requires a macro call expression, e.g. {%% call myMacro(args) %%}")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endcall")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endcall"); err != nil {
		return nil, err
	}
	return &ast.CallNode{
		Callee:    mc.Callee,
		PosArgs:   mc.PosArgs,
		NamedArgs: mc.NamedArgs,
		Body:      body,
		Line:      tagStart.Line,
	}, nil
}

// parseWithVars parses an optional "with key=val, key2=val2" clause.
// Stops at tag end or "isolated" keyword.
func (p *parser) parseWithVars() ([]ast.NamedArgNode, error) {
	if p.peek().Kind != lexer.TK_IDENT || p.peek().Value != "with" {
		return nil, nil
	}
	p.advance() // consume "with"
	var vars []ast.NamedArgNode
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "isolated" {
			break
		}
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected variable name in with clause")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after variable name in with clause")
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		vars = append(vars, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	return vars, nil
}

// parseInclude parses {% include "name" [with k=v, ...] [isolated] %}.
func (p *parser) parseInclude(tagStart lexer.Token) (*ast.IncludeNode, error) {
	p.advance() // consume "include"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after include")
	}
	withVars, err := p.parseWithVars()
	if err != nil {
		return nil, err
	}
	isolated := false
	if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "isolated" {
		p.advance()
		isolated = true
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.IncludeNode{Name: nameTok.Value, WithVars: withVars, Isolated: isolated, Line: tagStart.Line}, nil
}

// parseRender parses {% render "name" [with k=v, ...] %} — always isolated.
func (p *parser) parseRender(tagStart lexer.Token) (*ast.RenderNode, error) {
	p.advance() // consume "render"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after render")
	}
	withVars, err := p.parseWithVars()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.RenderNode{Name: nameTok.Value, WithVars: withVars, Line: tagStart.Line}, nil
}

// parseImport parses {% import "name" as alias %}.
func (p *parser) parseImport(tagStart lexer.Token) (*ast.ImportNode, error) {
	p.advance() // consume "import"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after import")
	}
	asTok := p.advance()
	if asTok.Kind != lexer.TK_IDENT || asTok.Value != "as" {
		return nil, p.errorf(asTok.Line, asTok.Col, "expected 'as' after template name in import")
	}
	aliasTok := p.advance()
	if aliasTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(aliasTok.Line, aliasTok.Col, "expected alias name after 'as' in import")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.ImportNode{Name: nameTok.Value, Alias: aliasTok.Value, Line: tagStart.Line}, nil
}

// ─── Plan 5: Layout inheritance parser methods ────────────────────────────────

// parseExtends parses {% extends "name" %}.
// Inline templates may not use extends.
func (p *parser) parseExtends(tagStart lexer.Token) (*ast.ExtendsNode, error) {
	if p.inline {
		return nil, &groverrors.ParseError{
			Line:    tagStart.Line,
			Column:  tagStart.Col,
			Message: "extends not allowed in inline templates",
		}
	}
	p.advance() // consume "extends"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after extends")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.ExtendsNode{Name: nameTok.Value, Line: tagStart.Line}, nil
}

// parseBlock parses {% block name %}...{% endblock %}.
func (p *parser) parseBlock(tagStart lexer.Token) (*ast.BlockNode, error) {
	p.advance() // consume "block"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected block name after block")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endblock")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endblock"); err != nil {
		return nil, err
	}
	return &ast.BlockNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}

// ─── Plan 6: Component + Slots parser methods ─────────────────────────────────

// parsePropsParams parses a props parameter list: name, name2="default", ...
// Like parseMacroParams but no surrounding parens; loops until TK_TAG_END.
func (p *parser) parsePropsParams() ([]ast.MacroParam, error) {
	var params []ast.MacroParam
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		nameTok := p.advance()
		if nameTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(nameTok.Line, nameTok.Col, "expected parameter name in props declaration")
		}
		param := ast.MacroParam{Name: nameTok.Value}
		if p.peek().Kind == lexer.TK_ASSIGN {
			p.advance() // consume =
			def, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			param.Default = def
		}
		params = append(params, param)
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	return params, nil
}

// parseProps parses {% props name, name2="default", ... %}.
func (p *parser) parseProps(tagStart lexer.Token) (*ast.PropsNode, error) {
	p.advance() // consume "props"
	params, err := p.parsePropsParams()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.PropsNode{Params: params, Line: tagStart.Line}, nil
}

// parseSlot parses {% slot ["name"] %}...{% endslot %}.
func (p *parser) parseSlot(tagStart lexer.Token) (*ast.SlotNode, error) {
	p.advance() // consume "slot"
	name := ""
	if p.peek().Kind == lexer.TK_STRING {
		name = p.advance().Value
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endslot")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endslot"); err != nil {
		return nil, err
	}
	return &ast.SlotNode{Name: name, Default: body, Line: tagStart.Line}, nil
}

// parseComponent parses {% component "name" k=v, ... %}...{% endcomponent %}.
// The body is scanned to separate {% fill %} blocks from default-slot content.
func (p *parser) parseComponent(tagStart lexer.Token) (*ast.ComponentNode, error) {
	p.advance() // consume "component"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted template name after component")
	}

	// Parse props: key=val key2=val2 (until TAG_END)
	var props []ast.NamedArgNode
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected prop name in component tag")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after prop name")
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		props = append(props, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		}
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	// Parse body: separate {% fill %} from default-slot content
	node := &ast.ComponentNode{Name: nameTok.Value, Props: props, Line: tagStart.Line}
	if err := p.parseComponentBody(node); err != nil {
		return nil, err
	}
	return node, nil
}

// parseComponentBody parses until {% endcomponent %}, routing {% fill %} blocks
// into node.Fills and everything else into node.DefaultFill.
func (p *parser) parseComponentBody(node *ast.ComponentNode) error {
	for !p.atEOF() {
		if p.peek().Kind == lexer.TK_TAG_START {
			tagName, ok := p.peekTagName()
			if ok {
				switch tagName {
				case "endcomponent":
					return p.expectTag("endcomponent")
				case "fill":
					fill, err := p.parseFill()
					if err != nil {
						return err
					}
					node.Fills = append(node.Fills, *fill)
					continue
				}
			}
		}
		n, err := p.parseNode()
		if err != nil {
			return err
		}
		if n != nil {
			node.DefaultFill = append(node.DefaultFill, n)
		}
	}
	return p.errorf(p.peek().Line, p.peek().Col, "unclosed component block — expected endcomponent")
}

// parseFill parses {% fill "name" %}...{% endfill %}.
// Called when positioned AT TK_TAG_START.
func (p *parser) parseFill() (*ast.FillNode, error) {
	tagStart := p.peek()
	p.advance() // consume {%
	p.advance() // consume "fill"
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted slot name after fill")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("endfill")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endfill"); err != nil {
		return nil, err
	}
	return &ast.FillNode{Name: nameTok.Value, Body: body, Line: tagStart.Line}, nil
}

// ─── Plan 7: Web primitives parser methods ────────────────────────────────────

// parseAsset parses {% asset "src" type="stylesheet" [k=v | bareIdent]* [priority=N] %}.
// Bare idents (no = after them) are treated as boolean attributes (value = "").
func (p *parser) parseAsset(tagStart lexer.Token) (*ast.AssetNode, error) {
	p.advance() // consume "asset"
	srcTok := p.advance()
	if srcTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(srcTok.Line, srcTok.Col, "expected quoted asset src after asset")
	}

	node := &ast.AssetNode{Src: srcTok.Value, Line: tagStart.Line}

	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected attribute name in asset tag")
		}
		key := keyTok.Value

		// Check for = (value attr) or no = (boolean attr)
		if p.peek().Kind == lexer.TK_ASSIGN {
			p.advance() // consume =
			val, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			switch key {
			case "type":
				// type must be a string literal
				if sl, ok := val.(*ast.StringLiteral); ok {
					node.AssetType = sl.Value
				} else {
					return nil, p.errorf(keyTok.Line, keyTok.Col, "asset type= must be a string literal")
				}
			case "priority":
				// priority must be an integer literal
				if il, ok := val.(*ast.IntLiteral); ok {
					node.Priority = int(il.Value)
				} else {
					return nil, p.errorf(keyTok.Line, keyTok.Col, "asset priority= must be an integer literal")
				}
			default:
				node.Attrs = append(node.Attrs, ast.NamedArgNode{Key: key, Value: val, Line: keyTok.Line})
			}
		} else {
			// Boolean attr: bare ident → value = ""
			node.Attrs = append(node.Attrs, ast.NamedArgNode{
				Key:   key,
				Value: &ast.StringLiteral{Value: "", Line: keyTok.Line},
				Line:  keyTok.Line,
			})
		}
	}

	return node, p.expectTagEnd()
}

// parseMeta parses {% meta name="key" content="val" %} (or property=, http-equiv=).
// The metadata key is derived from the value of the name=, property=, or http-equiv= attribute.
func (p *parser) parseMeta(tagStart lexer.Token) (*ast.MetaNode, error) {
	p.advance() // consume "meta"

	var metaKey, metaContent string
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected attribute name in meta tag")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after %q in meta tag", keyTok.Value)
		}
		p.advance() // consume =
		valTok := p.advance()
		if valTok.Kind != lexer.TK_STRING {
			return nil, p.errorf(valTok.Line, valTok.Col, "meta attribute values must be string literals")
		}
		switch keyTok.Value {
		case "name", "property", "http-equiv":
			metaKey = valTok.Value
		case "content":
			metaContent = valTok.Value
		}
		// ignore unknown attrs silently
	}

	if metaKey == "" {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "meta tag requires name=, property=, or http-equiv= attribute")
	}
	return &ast.MetaNode{Key: metaKey, Value: metaContent, Line: tagStart.Line}, p.expectTagEnd()
}

// parseHoist parses {% hoist target="name" %}...{% endhoist %}.
func (p *parser) parseHoist(tagStart lexer.Token) (*ast.HoistNode, error) {
	p.advance() // consume "hoist"

	var target string
	for p.peek().Kind != lexer.TK_TAG_END && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected attribute name in hoist tag")
		}
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected = after %q in hoist tag", keyTok.Value)
		}
		p.advance() // consume =
		valTok := p.advance()
		if valTok.Kind != lexer.TK_STRING {
			return nil, p.errorf(valTok.Line, valTok.Col, "hoist target must be a string literal")
		}
		if keyTok.Value == "target" {
			target = valTok.Value
		}
	}
	if target == "" {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "hoist tag requires target= attribute")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	body, err := p.parseBody("endhoist")
	if err != nil {
		return nil, err
	}
	if err := p.expectTag("endhoist"); err != nil {
		return nil, err
	}
	return &ast.HoistNode{Target: target, Body: body, Line: tagStart.Line}, nil
}
