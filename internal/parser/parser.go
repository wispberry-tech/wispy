// internal/parser/parser.go
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wispberry-tech/grove/internal/ast"
	"github.com/wispberry-tech/grove/internal/groverrors"
	"github.com/wispberry-tech/grove/internal/lexer"
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
	allowedTags map[string]bool            // nil = all allowed; non-nil = whitelist
	imports     map[string]importEntry      // local name → {src, compName}
}

type importEntry struct {
	src       string // template source path (e.g., "btn")
	compName  string // component name within that file (e.g., "Btn")
	namespace string // for wildcard imports with as="UI"
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
	// Store import map for dynamic component resolution
	if len(p.imports) > 0 {
		prog.ImportMap = make(map[string]string, len(p.imports))
		for localName, entry := range p.imports {
			if entry.compName != "*" {
				prog.ImportMap[localName] = entry.src + "#" + entry.compName
			}
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
	case lexer.TK_TAG_START:
		return p.parseTag()
	case lexer.TK_ELEMENT_OPEN:
		return p.parseElement()
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
// Sigil tokens are prefixed: "#if", ":else", "/each".
func tokenTagName(tk lexer.Token) (string, bool) {
	switch tk.Kind {
	case lexer.TK_IDENT:
		return tk.Value, true
	case lexer.TK_NOT:
		return "not", true
	case lexer.TK_IN:
		return "in", true
	case lexer.TK_BLOCK_OPEN:
		return "#" + tk.Value, true
	case lexer.TK_BLOCK_BRANCH:
		return ":" + tk.Value, true
	case lexer.TK_BLOCK_CLOSE:
		return "/" + tk.Value, true
	}
	return "", false
}

// ─── Tags {% name ... %} ──────────────────────────────────────────────────────

func (p *parser) parseTag() (ast.Node, error) {
	tagStart := p.advance() // consume TAG_START
	tok := p.peek()

	// Dispatch on sigil tokens: #keyword, :keyword, /keyword
	switch tok.Kind {
	case lexer.TK_BLOCK_OPEN:
		if p.allowedTags != nil && !p.allowedTags["#"+tok.Value] {
			return nil, p.errorf(tok.Line, tok.Col, "sandbox: tag #%s is not allowed", tok.Value)
		}
		switch tok.Value {
		case "if":
			return p.parseIf(tagStart)
		case "each":
			return p.parseEach(tagStart)
		case "fill":
			return nil, p.errorf(tok.Line, tok.Col, "#fill must appear inside a component body")
		case "slot":
			return p.parseSlotBlock(tagStart)
		case "capture":
			return p.parseCapture(tagStart)
		case "hoist":
			return p.parseHoist(tagStart)
		case "let":
			return p.parseLet(tagStart)
		default:
			return nil, p.errorf(tok.Line, tok.Col, "unknown block tag #%s", tok.Value)
		}

	case lexer.TK_BLOCK_BRANCH:
		return nil, p.errorf(tok.Line, tok.Col, "unexpected :%s outside of a block", tok.Value)

	case lexer.TK_BLOCK_CLOSE:
		return nil, p.errorf(tok.Line, tok.Col, "unexpected /%s outside of a block", tok.Value)
	}

	// Plain keyword tags (no sigil)
	name, ok := tokenTagName(tok)
	if ok {
		if p.allowedTags != nil && (name == "set" || name == "import" || name == "asset" || name == "meta" || name == "slot") {
			if !p.allowedTags[name] {
				return nil, p.errorf(tok.Line, tok.Col, "sandbox: tag %q is not allowed", name)
			}
		}
		switch name {
		case "set":
			return p.parseSet(tagStart)
		case "import":
			if p.inline {
				return nil, p.errorf(tagStart.Line, tagStart.Col, "import not allowed in inline templates")
			}
			return p.parseImport(tagStart)
		case "slot":
			return p.parseSlotInline(tagStart)
		case "asset":
			if p.inline {
				return nil, p.errorf(tagStart.Line, tagStart.Col, "asset not allowed in inline templates")
			}
			return p.parseAsset(tagStart)
		case "meta":
			return p.parseMeta(tagStart)

		// Removed syntax — produce clear errors
		case "extends":
			return nil, p.errorf(tagStart.Line, tagStart.Col,
				"extends syntax has been removed; use component composition with {%% #slot %%}/{%% #fill %%}")
		case "unless":
			return nil, p.errorf(tagStart.Line, tagStart.Col,
				`unknown tag "unless": use {%% #if not ... %%} instead`)
		case "with":
			return nil, p.errorf(tagStart.Line, tagStart.Col,
				`unknown tag "with": use {%% #let %%} or {%% set %%} instead`)
		}
	}

	// Default: parse as output expression {% expr %}
	expr, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	end := p.peek()
	if end.Kind != lexer.TK_TAG_END {
		return nil, p.errorf(end.Line, end.Col, "expected %%}, got %q", end.Value)
	}
	p.advance() // consume TAG_END
	return &ast.OutputNode{
		Expr:       expr,
		StripLeft:  tagStart.StripLeft,
		StripRight: end.StripRight,
		Line:       tagStart.Line,
	}, nil
}

// ─── PascalCase Element dispatch ─────────────────────────────────────────────

func (p *parser) parseElement() (ast.Node, error) {
	tk := p.peek()

	// <Component> for definition and dynamic invocation
	if tk.Value == "Component" {
		return p.parseComponentDefElement()
	}

	// Everything else must be an imported component invocation
	if p.resolveImport(tk.Value) != nil {
		return p.parseComponentInvocation()
	}
	return nil, p.errorf(tk.Line, tk.Col, "unknown element <%s>; did you forget {%% import ... from ... %%}?", tk.Value)
}

// resolveImport returns the importEntry for a component name, or nil if not found.
// Handles: explicit imports, wildcard imports, and namespaced wildcard imports (UI.Card).
func (p *parser) resolveImport(name string) *importEntry {
	// Direct match
	if entry, ok := p.imports[name]; ok {
		return &entry
	}

	// Check for namespaced wildcard: UI.Card → wildcard import with namespace "UI"
	if idx := strings.Index(name, "."); idx > 0 {
		prefix := name[:idx]
		for _, entry := range p.imports {
			if entry.compName == "*" && entry.namespace == prefix {
				e := importEntry{src: entry.src, compName: name[idx+1:], namespace: entry.namespace}
				return &e
			}
		}
	}

	// Check for non-namespaced wildcard: any PascalCase name → wildcard import
	for _, entry := range p.imports {
		if entry.compName == "*" && entry.namespace == "" {
			e := importEntry{src: entry.src, compName: name}
			return &e
		}
	}

	return nil
}

// readAttr reads the next attribute from an element's attribute list.
// Returns the attribute name, value (nil for bare attributes), and whether
// the element was closed with /> (selfClose) or > (when name is "").
func (p *parser) readAttr() (name string, value ast.Node, selfClose bool, err error) {
	tk := p.peek()

	if tk.Kind == lexer.TK_SELF_CLOSE {
		p.advance()
		return "", nil, true, nil
	}
	if tk.Kind == lexer.TK_ELEMENT_END {
		p.advance()
		return "", nil, false, nil
	}

	// Attribute name
	if tk.Kind != lexer.TK_IDENT && tk.Kind != lexer.TK_NOT && tk.Kind != lexer.TK_IN &&
		tk.Kind != lexer.TK_AND && tk.Kind != lexer.TK_OR {
		return "", nil, false, p.errorf(tk.Line, tk.Col, "expected attribute name, got %q", tk.Value)
	}
	name = tk.Value
	p.advance()

	// Check for colon suffix (let:data pattern)
	if p.peek().Kind == lexer.TK_COLON {
		p.advance()
		suffix := p.peek()
		if suffix.Kind == lexer.TK_IDENT {
			name = name + ":" + suffix.Value
			p.advance()
		}
	}

	// Check for = (attribute value)
	if p.peek().Kind != lexer.TK_ASSIGN {
		return name, nil, false, nil // bare attribute
	}
	p.advance() // consume =

	valTk := p.peek()
	if valTk.Kind == lexer.TK_STRING {
		p.advance()
		return name, &ast.StringLiteral{Value: valTk.Value, Line: valTk.Line}, false, nil
	}
	if valTk.Kind == lexer.TK_LBRACE {
		p.advance() // consume {
		expr, exprErr := p.parseExpr(0)
		if exprErr != nil {
			return "", nil, false, exprErr
		}
		if p.peek().Kind != lexer.TK_RBRACE {
			return "", nil, false, p.errorf(p.peek().Line, p.peek().Col, "expected } after expression")
		}
		p.advance() // consume }
		return name, expr, false, nil
	}

	return "", nil, false, p.errorf(valTk.Line, valTk.Col, "expected string or {expression} for attribute value")
}

// parseElementBody parses nodes until a closing element </closeElem> or a stop element <stopElem>.
// Does NOT consume the stop/close element.
func (p *parser) parseElementBody(closeElem string, stopElems ...string) ([]ast.Node, error) {
	var nodes []ast.Node
	for !p.atEOF() {
		tk := p.peek()

		// Stop on </CloseElem>
		if tk.Kind == lexer.TK_ELEMENT_CLOSE && tk.Value == closeElem {
			return nodes, nil
		}

		// Stop on <StopElem> (e.g. <ElseIf>, <Else>, <Empty>)
		if tk.Kind == lexer.TK_ELEMENT_OPEN {
			for _, stop := range stopElems {
				if tk.Value == stop {
					return nodes, nil
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

func (p *parser) expectElementClose(name string) error {
	tk := p.peek()
	if tk.Kind != lexer.TK_ELEMENT_CLOSE || tk.Value != name {
		return p.errorf(tk.Line, tk.Col, "expected </%s>, got %q", name, tk.Value)
	}
	p.advance()
	return nil
}

// ─── <Component name="X" prop1 prop2="default"> ─────────────────────────────

func (p *parser) parseComponentDefElement() (ast.Node, error) {
	openTk := p.advance() // consume TK_ELEMENT_OPEN("Component")

	// First pass: check if this is a dynamic component (<Component is={expr}>)
	// or a definition (<Component name="X">)
	var compName string
	var isExpr ast.Node
	var params []ast.MacroParam
	var props []ast.NamedArgNode
	var selfClose bool

	for {
		name, value, sc, err := p.readAttr()
		if err != nil {
			return nil, err
		}
		if name == "" {
			selfClose = sc
			break
		}
		if name == "is" {
			isExpr = value
			continue
		}
		if name == "name" {
			if s, ok := value.(*ast.StringLiteral); ok {
				compName = s.Value
				continue
			}
			// bare `name` or `name={expr}` after `name="X"` is a prop declaration
		}
		// For definitions: other attributes are props
		// For dynamic: other attributes are passed props
		param := ast.MacroParam{Name: name}
		if value != nil {
			param.Default = value
		}
		params = append(params, param)
		if value == nil {
			value = &ast.BoolLiteral{Value: true, Line: openTk.Line}
		}
		props = append(props, ast.NamedArgNode{Key: name, Value: value, Line: openTk.Line})
	}

	// Dynamic component: <Component is={expr} title="Hello" />
	if isExpr != nil {
		node := &ast.ComponentNode{
			Name:  "__dynamic__",
			Props: props,
			Line:  openTk.Line,
		}
		// Store the is-expression as a special prop
		node.Props = append([]ast.NamedArgNode{{Key: "__is__", Value: isExpr, Line: openTk.Line}}, node.Props...)
		if !selfClose {
			// Parse body
			var defaultFill []ast.Node
			var fills []ast.FillNode
			for !p.atEOF() {
				tk := p.peek()
				if tk.Kind == lexer.TK_ELEMENT_CLOSE && tk.Value == "Component" {
					p.advance()
					break
				}
				if tk.Kind == lexer.TK_ELEMENT_OPEN && tk.Value == "Fill" {
					fill, fillErr := p.parseFillElement()
					if fillErr != nil {
						return nil, fillErr
					}
					fills = append(fills, *fill)
					continue
				}
				n, parseErr := p.parseNode()
				if parseErr != nil {
					return nil, parseErr
				}
				if n != nil {
					defaultFill = append(defaultFill, n)
				}
			}
			node.DefaultFill = defaultFill
			node.Fills = fills
		}
		return node, nil
	}

	// Component definition: <Component name="X" ...>body</Component>
	if selfClose {
		return nil, p.errorf(openTk.Line, openTk.Col, "<Component> definition cannot be self-closing")
	}

	body, err := p.parseElementBody("Component")
	if err != nil {
		return nil, err
	}
	if closeErr := p.expectElementClose("Component"); closeErr != nil {
		return nil, closeErr
	}

	return &ast.ComponentDefNode{
		Name:  compName,
		Props: params,
		Body:  body,
		Line:  openTk.Line,
	}, nil
}

// ─── Component invocation: <Btn label="Save" /> ─────────────────────────────

func (p *parser) parseComponentInvocation() (ast.Node, error) {
	openTk := p.advance() // consume TK_ELEMENT_OPEN("Btn" etc.)
	elemName := openTk.Value

	entry := p.resolveImport(elemName)
	if entry == nil {
		return nil, p.errorf(openTk.Line, openTk.Col, "unknown component <%s>", elemName)
	}

	// Read props from attributes
	var props []ast.NamedArgNode
	var selfClose bool

	for {
		name, value, sc, err := p.readAttr()
		if err != nil {
			return nil, err
		}
		if name == "" {
			selfClose = sc
			break
		}
		if value == nil {
			// Bare attribute → boolean true
			value = &ast.BoolLiteral{Value: true, Line: openTk.Line}
		}
		props = append(props, ast.NamedArgNode{Key: name, Value: value, Line: openTk.Line})
	}

	// Use "src#CompName" so the engine can resolve named components
	templateName := entry.src + "#" + entry.compName
	node := &ast.ComponentNode{
		Name: templateName,
		Props: props,
		Line:  openTk.Line,
	}

	if selfClose {
		return node, nil
	}

	// Parse body: separate {% #fill %} tags from default content
	var defaultFill []ast.Node
	var fills []ast.FillNode

	for !p.atEOF() {
		tk := p.peek()
		if tk.Kind == lexer.TK_ELEMENT_CLOSE && tk.Value == elemName {
			p.advance()
			break
		}
		// Detect {% #fill "name" %}...{% /fill %}
		if tk.Kind == lexer.TK_TAG_START && p.pos+1 < len(p.tokens) &&
			p.tokens[p.pos+1].Kind == lexer.TK_BLOCK_OPEN && p.tokens[p.pos+1].Value == "fill" {
			tagStart := p.advance() // consume TAG_START
			fill, err := p.parseFillTag(tagStart)
			if err != nil {
				return nil, err
			}
			fills = append(fills, *fill)
			continue
		}
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if n != nil {
			defaultFill = append(defaultFill, n)
		}
	}

	node.DefaultFill = defaultFill
	node.Fills = fills
	return node, nil
}

// ─── <Fill slot="name"> ─────────────────────────────────────────────────────

func (p *parser) parseFillElement() (*ast.FillNode, error) {
	openTk := p.advance() // consume TK_ELEMENT_OPEN("Fill")

	var slotName string
	var letBindings map[string]string
	for {
		name, value, selfClose, err := p.readAttr()
		if err != nil {
			return nil, err
		}
		if name == "" {
			if selfClose {
				return &ast.FillNode{Name: slotName, LetBindings: letBindings, Line: openTk.Line}, nil
			}
			break
		}
		if name == "slot" {
			if s, ok := value.(*ast.StringLiteral); ok {
				slotName = s.Value
			}
		} else if strings.HasPrefix(name, "let:") {
			// let:data or let:data="alias"
			scopeKey := name[4:] // after "let:"
			localVar := scopeKey // default: same name
			if value != nil {
				if s, ok := value.(*ast.StringLiteral); ok {
					localVar = s.Value
				}
			}
			if letBindings == nil {
				letBindings = make(map[string]string)
			}
			letBindings[scopeKey] = localVar
		}
	}

	body, err := p.parseElementBody("Fill")
	if err != nil {
		return nil, err
	}
	if closeErr := p.expectElementClose("Fill"); closeErr != nil {
		return nil, closeErr
	}

	return &ast.FillNode{
		Name:        slotName,
		Body:        body,
		LetBindings: letBindings,
		Line:        openTk.Line,
	}, nil
}

// ─── {% if %} ─────────────────────────────────────────────────────────────────

func (p *parser) parseIf(tagStart lexer.Token) (*ast.IfNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("if")
	cond, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	node := &ast.IfNode{Condition: cond, Line: tagStart.Line}

	// Parse body until :else or /if
	node.Body, err = p.parseBody(":else", "/if")
	if err != nil {
		return nil, err
	}

	// Parse :else if / :else chains
	for {
		tagName, _ := p.peekTagName()
		if tagName == ":else" {
			p.advance() // TAG_START
			p.advance() // TK_BLOCK_BRANCH("else")
			// Check if this is {% :else if expr %} or just {% :else %}
			if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "if" {
				p.advance() // consume "if"
				elifCond, err := p.parseExpr(0)
				if err != nil {
					return nil, err
				}
				if err := p.expectTagEnd(); err != nil {
					return nil, err
				}
				body, err := p.parseBody(":else", "/if")
				if err != nil {
					return nil, err
				}
				node.Elifs = append(node.Elifs, ast.ElifClause{Condition: elifCond, Body: body})
			} else {
				// Plain {% :else %}
				if err := p.expectTagEnd(); err != nil {
					return nil, err
				}
				node.Else, err = p.parseBody("/if")
				if err != nil {
					return nil, err
				}
				break
			}
		} else {
			break
		}
	}

	// Consume {% /if %}
	if err := p.expectBlockClose("if"); err != nil {
		return nil, err
	}
	return node, nil
}

// ─── {% #each %} ─────────────────────────────────────────────────────────────

func (p *parser) parseEach(tagStart lexer.Token) (*ast.ForNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("each")

	// Parse iterable expression first: {% #each items as item %}
	iterable, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}

	// Expect "as" keyword
	asTok := p.peek()
	if asTok.Kind != lexer.TK_IDENT || asTok.Value != "as" {
		return nil, p.errorf(asTok.Line, asTok.Col, "expected 'as' after iterable in #each")
	}
	p.advance() // consume "as"

	// Read item variable name
	itemTok := p.advance()
	if itemTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(itemTok.Line, itemTok.Col, "expected variable name after 'as' in #each")
	}

	// Optional: comma + index/key variable
	// ForNode convention: Var1=key/index, Var2=value/item (matches OP_FOR_BIND_KV).
	// Single var: Var1=item, Var2=""
	// Two var: Var1=indexVar, Var2=itemVar
	var1 := itemTok.Value
	var var2 string
	if p.peek().Kind == lexer.TK_COMMA {
		p.advance() // consume comma
		idxTok := p.advance()
		if idxTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(idxTok.Line, idxTok.Col, "expected index variable name after ',' in #each")
		}
		// Swap: first name after "as" is value, second is key/index
		var1 = idxTok.Value
		var2 = itemTok.Value
	}

	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	body, err := p.parseBody(":empty", "/each")
	if err != nil {
		return nil, err
	}

	var emptyBody []ast.Node
	tagName, _ := p.peekTagName()
	if tagName == ":empty" {
		p.advance() // TAG_START
		p.advance() // TK_BLOCK_BRANCH("empty")
		if err := p.expectTagEnd(); err != nil {
			return nil, err
		}
		emptyBody, err = p.parseBody("/each")
		if err != nil {
			return nil, err
		}
	}

	if err := p.expectBlockClose("each"); err != nil {
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
	p.advance() // consume TK_BLOCK_OPEN("capture")
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected variable name after #capture")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("/capture")
	if err != nil {
		return nil, err
	}
	if err := p.expectBlockClose("capture"); err != nil {
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
		case lexer.TK_QUESTION:
			p.advance() // consume ?
			consequence, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != lexer.TK_COLON {
				return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ':' in ternary expression")
			}
			p.advance() // consume :
			alt, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			left = &ast.TernaryExpr{
				Condition:   left,
				Consequence: consequence,
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
	case lexer.TK_LBRACKET:
		return p.parseListLiteral(tk)
	case lexer.TK_LBRACE:
		return p.parseMapLiteral(tk)
	default:
		return nil, p.errorf(tk.Line, tk.Col, "unexpected token in expression: %q", tk.Value)
	}
}

func (p *parser) parseListLiteral(openTok lexer.Token) (ast.Node, error) {
	var elements []ast.Node
	for p.peek().Kind != lexer.TK_RBRACKET && !p.atEOF() {
		elem, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		} else {
			break
		}
	}
	if p.peek().Kind != lexer.TK_RBRACKET {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ] to close list literal")
	}
	p.advance()
	return &ast.ListLiteral{Elements: elements, Line: openTok.Line}, nil
}

func (p *parser) parseMapLiteral(openTok lexer.Token) (ast.Node, error) {
	var entries []ast.MapEntry
	for p.peek().Kind != lexer.TK_RBRACE && !p.atEOF() {
		keyTok := p.advance()
		if keyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "expected identifier key in map literal")
		}
		if p.peek().Kind != lexer.TK_COLON {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ':' after map key")
		}
		p.advance()
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ast.MapEntry{Key: keyTok.Value, Value: val})
		if p.peek().Kind == lexer.TK_COMMA {
			p.advance()
		} else {
			break
		}
	}
	if p.peek().Kind != lexer.TK_RBRACE {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected } to close map literal")
	}
	p.advance()
	return &ast.MapLiteral{Entries: entries, Line: openTok.Line}, nil
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
	case lexer.TK_QUESTION:
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


// ─── {% let %} ──────────────────────────────────────────────────────────────

// captureUntilEndTag extracts raw text between the current position and {% tagName %}.
// It consumes all tokens up to (but not including) the end tag.
func (p *parser) captureUntilEndTag(tagName string) (string, error) {
	var buf strings.Builder
	for !p.atEOF() {
		if p.peek().Kind == lexer.TK_TAG_START {
			if p.pos+1 < len(p.tokens) {
				name, ok := tokenTagName(p.tokens[p.pos+1])
				if ok && name == tagName {
					return buf.String(), nil
				}
			}
		}
		if p.peek().Kind == lexer.TK_TEXT {
			buf.WriteString(p.peek().Value)
		}
		p.advance()
	}
	return buf.String(), nil
}

func (p *parser) parseLet(tagStart lexer.Token) (*ast.LetNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("let") or TK_IDENT("let")
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	raw, err := p.captureUntilEndTag("/let")
	if err != nil {
		return nil, err
	}
	if err := p.expectBlockClose("let"); err != nil {
		return nil, err
	}
	tokens, err := lexer.TokenizeLetBody(raw)
	if err != nil {
		return nil, p.errorf(tagStart.Line, tagStart.Col, "let block: %v", err)
	}
	body, err := parseLetBody(tokens, tagStart.Line)
	if err != nil {
		return nil, &groverrors.ParseError{
			Line:    tagStart.Line,
			Column:  tagStart.Col,
			Message: err.Error(),
		}
	}
	return &ast.LetNode{Body: body, Line: tagStart.Line}, nil
}

// ─── let block mini-parser ──────────────────────────────────────────────────

type letParser struct {
	tokens []lexer.Token
	pos    int
}

func (lp *letParser) peek() lexer.Token {
	if lp.pos >= len(lp.tokens) {
		return lexer.Token{Kind: lexer.TK_EOF}
	}
	return lp.tokens[lp.pos]
}

func (lp *letParser) advance() lexer.Token {
	tok := lp.peek()
	if lp.pos < len(lp.tokens) {
		lp.pos++
	}
	return tok
}

func parseLetBody(tokens []lexer.Token, baseLine int) ([]ast.LetStmt, error) {
	lp := &letParser{tokens: tokens}
	return lp.parseStatements()
}

func (lp *letParser) parseStatements() ([]ast.LetStmt, error) {
	var stmts []ast.LetStmt
	for lp.peek().Kind != lexer.TK_EOF {
		tk := lp.peek()
		if tk.Kind == lexer.TK_IDENT {
			switch tk.Value {
			case "if":
				ifNode, err := lp.parseIf()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, ifNode)
			case "end", "elif", "else":
				return stmts, nil
			default:
				assign, err := lp.parseAssignment()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, assign)
			}
		} else {
			return nil, fmt.Errorf("let block line %d: unexpected token %q", tk.Line, tk.Value)
		}
	}
	return stmts, nil
}

func (lp *letParser) parseAssignment() (*ast.LetAssignment, error) {
	nameTok := lp.advance()
	if lp.peek().Kind != lexer.TK_ASSIGN {
		return nil, fmt.Errorf("let block line %d: expected '=' after %q", nameTok.Line, nameTok.Value)
	}
	lp.advance() // consume =
	subP := &parser{tokens: lp.tokens, pos: lp.pos}
	expr, err := subP.parseExpr(0)
	if err != nil {
		return nil, fmt.Errorf("let block line %d: %v", nameTok.Line, err)
	}
	lp.pos = subP.pos
	return &ast.LetAssignment{Name: nameTok.Value, Expr: expr}, nil
}

func (lp *letParser) parseIf() (*ast.LetIf, error) {
	lp.advance() // consume "if"
	subP := &parser{tokens: lp.tokens, pos: lp.pos}
	cond, err := subP.parseExpr(0)
	if err != nil {
		return nil, err
	}
	lp.pos = subP.pos

	body, err := lp.parseStatements()
	if err != nil {
		return nil, err
	}

	node := &ast.LetIf{Condition: cond, Body: body}

	for {
		tk := lp.peek()
		if tk.Kind == lexer.TK_IDENT && tk.Value == "elif" {
			lp.advance()
			subP := &parser{tokens: lp.tokens, pos: lp.pos}
			elifCond, err := subP.parseExpr(0)
			if err != nil {
				return nil, err
			}
			lp.pos = subP.pos
			elifBody, err := lp.parseStatements()
			if err != nil {
				return nil, err
			}
			node.Elifs = append(node.Elifs, ast.LetElif{Condition: elifCond, Body: elifBody})
		} else if tk.Kind == lexer.TK_IDENT && tk.Value == "else" {
			lp.advance()
			elseBody, err := lp.parseStatements()
			if err != nil {
				return nil, err
			}
			node.Else = elseBody
			break
		} else {
			break
		}
	}

	if lp.peek().Kind != lexer.TK_IDENT || lp.peek().Value != "end" {
		return nil, fmt.Errorf("let block: expected 'end' to close if, got %q", lp.peek().Value)
	}
	lp.advance()
	return node, nil
}

// expectTagEnd consumes the closing %} of the current tag.
func (p *parser) expectTagEnd() error {
	if p.peek().Kind != lexer.TK_TAG_END {
		return p.errorf(p.peek().Line, p.peek().Col, "expected %%} got %q", p.peek().Value)
	}
	p.advance()
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

// expectBlockClose consumes {% /keyword %} and errors if it doesn't match.
func (p *parser) expectBlockClose(keyword string) error {
	if p.peek().Kind != lexer.TK_TAG_START {
		return p.errorf(p.peek().Line, p.peek().Col, "expected {%% /%s %%}", keyword)
	}
	p.advance() // TAG_START
	tok := p.peek()
	if tok.Kind != lexer.TK_BLOCK_CLOSE || tok.Value != keyword {
		return p.errorf(tok.Line, tok.Col, "expected /%s, got %q", keyword, tok.Value)
	}
	p.advance() // BLOCK_CLOSE
	return p.expectTagEnd()
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





// parseImport parses {% import Name from "path" %} and its variants:
//   {% import Card from "components/cards" %}
//   {% import Card as InfoCard from "components/cards" %}
//   {% import Card, Badge from "components/ui" %}
//   {% import * from "components/ui" %}
//   {% import * as UI from "components/ui" %}
func (p *parser) parseImport(tagStart lexer.Token) (ast.Node, error) {
	p.advance() // consume "import"

	if p.imports == nil {
		p.imports = make(map[string]importEntry)
	}

	// Check for wildcard: {% import * ... %}
	if p.peek().Kind == lexer.TK_STAR {
		p.advance() // consume *
		var namespace string
		if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "as" {
			p.advance() // consume "as"
			nsTok := p.advance()
			if nsTok.Kind != lexer.TK_IDENT {
				return nil, p.errorf(nsTok.Line, nsTok.Col, "expected namespace name after 'as' in wildcard import")
			}
			namespace = nsTok.Value
		}
		if p.peek().Kind != lexer.TK_IDENT || p.peek().Value != "from" {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected 'from' in import")
		}
		p.advance() // consume "from"
		pathTok := p.advance()
		if pathTok.Kind != lexer.TK_STRING {
			return nil, p.errorf(pathTok.Line, pathTok.Col, "expected quoted path after 'from' in import")
		}
		if err := p.expectTagEnd(); err != nil {
			return nil, err
		}
		key := "*"
		if namespace != "" {
			key = "*:" + namespace
		}
		p.imports[key] = importEntry{src: pathTok.Value, compName: "*", namespace: namespace}
		return &ast.TextNode{Value: "", Line: tagStart.Line}, nil
	}

	// Read name list: Name or Name, Name2, Name3
	firstTok := p.advance()
	if firstTok.Kind != lexer.TK_IDENT {
		return nil, p.errorf(firstTok.Line, firstTok.Col, "expected component name after 'import'")
	}
	names := []string{firstTok.Value}
	for p.peek().Kind == lexer.TK_COMMA {
		p.advance() // consume comma
		nextTok := p.advance()
		if nextTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(nextTok.Line, nextTok.Col, "expected component name after ',' in import")
		}
		names = append(names, nextTok.Value)
	}

	// Optional: "as" alias (only for single imports)
	var alias string
	if p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "as" {
		if len(names) > 1 {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "'as' cannot be used with comma-separated imports; use separate import statements")
		}
		p.advance() // consume "as"
		aliasTok := p.advance()
		if aliasTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(aliasTok.Line, aliasTok.Col, "expected alias name after 'as' in import")
		}
		alias = aliasTok.Value
	}

	// Expect "from"
	if p.peek().Kind != lexer.TK_IDENT || p.peek().Value != "from" {
		return nil, p.errorf(p.peek().Line, p.peek().Col, "expected 'from' in import")
	}
	p.advance() // consume "from"
	pathTok := p.advance()
	if pathTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(pathTok.Line, pathTok.Col, "expected quoted path after 'from' in import")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	// Register imports (check for duplicates)
	for _, name := range names {
		localName := name
		if len(names) == 1 && alias != "" {
			localName = alias
		}
		if _, exists := p.imports[localName]; exists {
			return nil, p.errorf(tagStart.Line, tagStart.Col, "duplicate import: %q is already imported", localName)
		}
		p.imports[localName] = importEntry{src: pathTok.Value, compName: name}
	}

	return &ast.TextNode{Value: "", Line: tagStart.Line}, nil
}


// ─── Plan 6: Component + Slots parser methods ─────────────────────────────────



// parseSlotInline parses self-closing slot tags: {% slot %} or {% slot "name" data={expr} %}
func (p *parser) parseSlotInline(tagStart lexer.Token) (*ast.SlotNode, error) {
	p.advance() // consume "slot" ident
	name := ""
	if p.peek().Kind == lexer.TK_STRING {
		name = p.advance().Value
	}
	// Optional scope data: key={expr} pairs
	var scopeData []ast.NamedArgNode
	for p.peek().Kind == lexer.TK_IDENT && p.peek().Kind != lexer.TK_TAG_END {
		keyTok := p.advance()
		if p.peek().Kind != lexer.TK_ASSIGN {
			// Not a key=value, put it back conceptually — but we already consumed it.
			// This shouldn't happen in well-formed templates.
			return nil, p.errorf(keyTok.Line, keyTok.Col, "unexpected token %q in slot tag", keyTok.Value)
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		scopeData = append(scopeData, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	return &ast.SlotNode{Name: name, ScopeData: scopeData, Line: tagStart.Line}, nil
}

// parseSlotBlock parses {% #slot "name" %}default content{% /slot %}
func (p *parser) parseSlotBlock(tagStart lexer.Token) (*ast.SlotNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("slot")
	name := ""
	if p.peek().Kind == lexer.TK_STRING {
		name = p.advance().Value
	}
	// Optional scope data
	var scopeData []ast.NamedArgNode
	for p.peek().Kind == lexer.TK_IDENT && p.peek().Kind != lexer.TK_TAG_END {
		keyTok := p.advance()
		if p.peek().Kind != lexer.TK_ASSIGN {
			return nil, p.errorf(keyTok.Line, keyTok.Col, "unexpected token %q in #slot tag", keyTok.Value)
		}
		p.advance() // consume =
		val, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		scopeData = append(scopeData, ast.NamedArgNode{Key: keyTok.Value, Value: val, Line: keyTok.Line})
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("/slot")
	if err != nil {
		return nil, err
	}
	if err := p.expectBlockClose("slot"); err != nil {
		return nil, err
	}
	return &ast.SlotNode{Name: name, Default: body, ScopeData: scopeData, Line: tagStart.Line}, nil
}

// parseFillTag parses {% #fill "name" [let:key ...] %}...{% /fill %}
func (p *parser) parseFillTag(tagStart lexer.Token) (*ast.FillNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("fill")
	nameTok := p.advance()
	if nameTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(nameTok.Line, nameTok.Col, "expected quoted slot name after #fill")
	}

	// Optional let: bindings
	var letBindings map[string]string
	for p.peek().Kind == lexer.TK_IDENT && p.peek().Value == "let" {
		p.advance() // consume "let"
		if p.peek().Kind != lexer.TK_COLON {
			return nil, p.errorf(p.peek().Line, p.peek().Col, "expected ':' after 'let' in #fill tag")
		}
		p.advance() // consume ":"
		scopeKeyTok := p.advance()
		if scopeKeyTok.Kind != lexer.TK_IDENT {
			return nil, p.errorf(scopeKeyTok.Line, scopeKeyTok.Col, "expected identifier after 'let:' in #fill tag")
		}
		scopeKey := scopeKeyTok.Value
		localVar := scopeKey // default: same name
		if p.peek().Kind == lexer.TK_ASSIGN {
			p.advance() // consume =
			aliasTok := p.advance()
			if aliasTok.Kind == lexer.TK_STRING {
				localVar = aliasTok.Value
			}
		}
		if letBindings == nil {
			letBindings = make(map[string]string)
		}
		letBindings[scopeKey] = localVar
	}

	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}
	body, err := p.parseBody("/fill")
	if err != nil {
		return nil, err
	}
	if err := p.expectBlockClose("fill"); err != nil {
		return nil, err
	}
	return &ast.FillNode{Name: nameTok.Value, Body: body, LetBindings: letBindings, Line: tagStart.Line}, nil
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

// parseHoist parses {% #hoist "target" %}...{% /hoist %}.
func (p *parser) parseHoist(tagStart lexer.Token) (*ast.HoistNode, error) {
	p.advance() // consume TK_BLOCK_OPEN("hoist")

	targetTok := p.advance()
	if targetTok.Kind != lexer.TK_STRING {
		return nil, p.errorf(targetTok.Line, targetTok.Col, "expected quoted target name after #hoist")
	}
	if err := p.expectTagEnd(); err != nil {
		return nil, err
	}

	body, err := p.parseBody("/hoist")
	if err != nil {
		return nil, err
	}
	if err := p.expectBlockClose("hoist"); err != nil {
		return nil, err
	}
	return &ast.HoistNode{Target: targetTok.Value, Body: body, Line: tagStart.Line}, nil
}
