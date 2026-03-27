package parser

import (
	"fmt"
	"strconv"

	"template-wisp/internal/ast"
	"template-wisp/internal/lexer"
	"template-wisp/internal/scope"
)

// Parser holds the state of the parser.
type Parser struct {
	l              *lexer.Lexer
	curTok         lexer.Token
	peekTok        lexer.Token
	errors         []string
	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
	currentScope   *scope.Scope
	skipAdvance    bool // if true, skip nextToken() in ParseProgram
}

// NewParser creates a new parser with the given lexer.
func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		errors:         []string{},
		prefixParseFns: make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenType]infixParseFn),
		currentScope:   scope.NewScope(),
	}
	p.registerPrefix()
	p.registerInfix()

	// Read two tokens, so curTok and peekTok are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances the token stream.
func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.l.NextToken()
}

// ParseProgram parses the entire program.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curTok.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		// Only advance if we're not already at EOF and skipAdvance is not set
		if p.curTok.Type != lexer.EOF && !p.skipAdvance {
			p.nextToken()
		}
		p.skipAdvance = false
	}

	return program
}

// Errors returns any parser errors.
func (p *Parser) Errors() []string {
	return p.errors
}

// CurrentScope returns the current scope.
func (p *Parser) CurrentScope() *scope.Scope {
	return p.currentScope
}

// PushScope creates a new child scope.
func (p *Parser) PushScope() {
	p.currentScope = scope.NewChildScope(p.currentScope)
}

// PopScope destroys the current scope and returns to the parent.
func (p *Parser) PopScope() {
	if p.currentScope.Parent() != nil {
		parent := p.currentScope.Parent()
		p.currentScope.Release()
		p.currentScope = parent
	}
}

// PushIsolatedScope creates a new isolated scope.
func (p *Parser) PushIsolatedScope() {
	p.currentScope = scope.NewIsolatedScope()
}

// peekError adds an error for the expected token.
func (p *Parser) peekError(t lexer.TokenType) {
	msg := "expected next token to be %s, got %s instead"
	p.errors = append(p.errors, fmt.Sprintf(msg, t, p.peekTok.Type))
}

// parseStatement parses a statement.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curTok.Type {
	case lexer.LET:
		return p.parseLetStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.LBRACE_PCT:
		return p.parseWispStatement()
	case lexer.TEXT:
		return p.parseTextContent()
	case lexer.WHEN:
		return p.parseWispWhenStatement()
	case lexer.ELSIF:
		return p.parseWispElsifStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement parses a let or assign statement.
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curTok}

	// Check if this is Wisp style: let .name = value
	if p.peekTokIs(lexer.DOT) {
		p.nextToken() // consume LET
		p.nextToken() // consume DOT

		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		stmt.Name = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}

		// Expect = or ASSIGN_OP
		if !p.peekTokIs(lexer.ASSIGN_OP) && !p.peekTokIs(lexer.ASSIGN) {
			p.peekError(lexer.ASSIGN_OP)
			return nil
		}
		p.nextToken()

		// Parse the value expression
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)

		// Check for optional semicolon
		if p.peekTokIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return stmt
	}

	// Traditional style: let x = value
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}

	// Expect = or ASSIGN_OP
	if !p.peekTokIs(lexer.ASSIGN_OP) && !p.peekTokIs(lexer.ASSIGN) {
		p.peekError(lexer.ASSIGN_OP)
		return nil
	}
	p.nextToken()

	// Parse the value expression
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Check for optional semicolon
	if p.peekTokIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement parses a return statement.
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curTok}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseIfStatement parses an if statement.
func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curTok}

	// Check if this is Wisp style: {% if .condition %}
	if p.peekTokIs(lexer.LBRACE_PCT) {
		p.nextToken() // consume IF
		p.nextToken() // consume {%

		// Parse the condition
		stmt.Condition = p.parseExpression(LOWEST)

		// Expect %}
		if !p.expectPeek(lexer.RBRACE_PCT) {
			return nil
		}

		return stmt
	}

	// Traditional style: if (condition) { ... }
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if p.peekTokIs(lexer.ELSE) {
		p.nextToken()

		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}

		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

// parseBlockStatement parses a block of statements.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curTok}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokIs(lexer.RBRACE) && !p.curTokIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseExpressionStatement parses an expression statement.
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curTok}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseTextContent parses literal text content between Wisp tags.
func (p *Parser) parseTextContent() *ast.TextContent {
	stmt := &ast.TextContent{
		Token: p.curTok,
		Value: p.curTok.Literal,
	}
	return stmt
}

// parseExpression parses an expression.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curTok.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curTok.Type)
		return nil
	}
	leftExp := prefix(p)

	for !p.peekTokIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekTok.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(p, leftExp)
	}

	return leftExp
}

// Helper functions
func (p *Parser) curTokIs(t lexer.TokenType) bool {
	return p.curTok.Type == t
}

func (p *Parser) peekTokIs(t lexer.TokenType) bool {
	return p.peekTok.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekTok.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curTok.Type]; ok {
		return p
	}
	return LOWEST
}

var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.LTE:      LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.GTE:      LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.AS:       LOWEST,
	lexer.IN:       LOWEST,
	lexer.COMMA:    LOWEST,
}

// TODO: Define precedence constants
const (
	_ int = iota
	LOWEST
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	CALL
	PREFIX
)

// TODO: Initialize prefix and infix parse functions
var prefixParseFns map[lexer.TokenType]prefixParseFn
var infixParseFns map[lexer.TokenType]infixParseFn

type prefixParseFn func(p *Parser) ast.Expression
type infixParseFn func(p *Parser, left ast.Expression) ast.Expression

func (p *Parser) registerPrefix() {
	p.prefixParseFns[lexer.IDENT] = parseIdentifier
	p.prefixParseFns[lexer.NUMBER] = parseIntegerLiteral
	p.prefixParseFns[lexer.BANG] = parsePrefixExpression
	p.prefixParseFns[lexer.MINUS] = parsePrefixExpression
	p.prefixParseFns[lexer.TRUE] = parseBoolean
	p.prefixParseFns[lexer.FALSE] = parseBoolean
	p.prefixParseFns[lexer.LPAREN] = parseGroupedExpression
	p.prefixParseFns[lexer.IF] = parseIfExpression
	p.prefixParseFns[lexer.FUNCTION] = parseFunctionLiteral
	p.prefixParseFns[lexer.DOT] = parseDotExpression
	p.prefixParseFns[lexer.STRING] = parseStringLiteral
	p.prefixParseFns[lexer.PIPE] = parsePipeExpression
}

func (p *Parser) registerInfix() {
	p.infixParseFns[lexer.PLUS] = parseInfixExpression
	p.infixParseFns[lexer.MINUS] = parseInfixExpression
	p.infixParseFns[lexer.SLASH] = parseInfixExpression
	p.infixParseFns[lexer.ASTERISK] = parseInfixExpression
	p.infixParseFns[lexer.EQ] = parseInfixExpression
	p.infixParseFns[lexer.NOT_EQ] = parseInfixExpression
	p.infixParseFns[lexer.LT] = parseInfixExpression
	p.infixParseFns[lexer.LTE] = parseInfixExpression
	p.infixParseFns[lexer.GT] = parseInfixExpression
	p.infixParseFns[lexer.GTE] = parseInfixExpression
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	msg := "no prefix parse function for %s found"
	p.errors = append(p.errors, fmt.Sprintf(msg, t))
}

// parseIdentifier parses an identifier.
func parseIdentifier(p *Parser) ast.Expression {
	return &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
}

// parseIntegerLiteral parses an integer literal.
func parseIntegerLiteral(p *Parser) ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curTok}

	value, err := strconv.ParseInt(p.curTok.Literal, 0, 64)
	if err != nil {
		msg := "could not parse %q as integer"
		p.errors = append(p.errors, fmt.Sprintf(msg, p.curTok.Literal))
		return nil
	}

	lit.Value = value
	return lit
}

// parseBoolean parses a boolean literal.
func parseBoolean(p *Parser) ast.Expression {
	return &ast.Boolean{Token: p.curTok, Value: p.curTokIs(lexer.TRUE)}
}

// parsePrefixExpression parses a prefix expression.
func parsePrefixExpression(p *Parser) ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curTok,
		Operator: p.curTok.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseInfixExpression parses an infix expression.
func parseInfixExpression(p *Parser, left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curTok,
		Operator: p.curTok.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseGroupedExpression parses a grouped expression.
func parseGroupedExpression(p *Parser) ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return exp
}

// parseIfExpression parses an if expression.
func parseIfExpression(p *Parser) ast.Expression {
	expression := &ast.IfExpression{Token: p.curTok}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokIs(lexer.ELSE) {
		p.nextToken()

		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parseFunctionLiteral parses a function literal.
func parseFunctionLiteral(p *Parser) ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curTok}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters parses function parameters.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokIs(lexer.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return identifiers
}

// parseDotExpression parses a dot expression (variable access): .name, .user.name
func parseDotExpression(p *Parser) ast.Expression {
	// Create a dot expression node
	dotExpr := &ast.DotExpression{Token: p.curTok}

	// Expect an identifier after the dot
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	// The identifier is the field name
	dotExpr.Field = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}

	// Handle chained access: .user.name
	// Stop if we see RBRACE_PCT (end of Wisp statement) or other terminators
	for p.peekTokIs(lexer.DOT) && !p.peekTokIs(lexer.RBRACE_PCT) {
		p.nextToken() // consume the dot
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		// Add to chain
		nextField := &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
		dotExpr.Chain = append(dotExpr.Chain, nextField)
	}

	// Don't advance - let parseExpression() handle positioning
	// This ensures consistency with other prefix parse functions

	return dotExpr
}

// parseStringLiteral parses a string literal.
func parseStringLiteral(p *Parser) ast.Expression {
	lit := &ast.StringLiteral{Token: p.curTok, Value: p.curTok.Literal}
	// Don't advance - let parseExpression() handle positioning
	return lit
}

// parsePipeExpression parses a pipe expression (function call): . | date, . | format "%s"
func parsePipeExpression(p *Parser) ast.Expression {
	pipeExpr := &ast.PipeExpression{Token: p.curTok}

	// We're already at the identifier (function name), so just use it
	if p.curTok.Type != lexer.IDENT {
		p.errors = append(p.errors, fmt.Sprintf("expected identifier after pipe, got %s", p.curTok.Type))
		return nil
	}

	// The identifier is the function name
	pipeExpr.Function = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}

	// Parse optional arguments
	for !p.peekTokIs(lexer.RBRACE_PCT) && !p.peekTokIs(lexer.EOF) && !p.peekTokIs(lexer.PIPE) {
		p.nextToken()
		arg := p.parseExpression(LOWEST)
		if arg != nil {
			pipeExpr.Arguments = append(pipeExpr.Arguments, arg)
		}
	}

	return pipeExpr
}

// parseBodyStatements parses statements until a closing tag is found.
// Returns the body block and the closing tag type (e.g. lexer.END, lexer.ELSE, lexer.ENDRAW, etc.).
func (p *Parser) parseBodyStatements() (*ast.BlockStatement, lexer.TokenType) {
	body := &ast.BlockStatement{Statements: []ast.Statement{}}

	p.nextToken()

	for {
		// Check for stop tokens that should NOT be in the body
		switch p.curTok.Type {
		case lexer.END, lexer.ELSE,
			lexer.ENDRAW, lexer.ENDCOMMENT,
			lexer.RETURN, lexer.FUNCTION,
			lexer.EOF:
			return body, p.curTok.Type
		}

		// Check for closing tags inside {% %} BEFORE calling parseStatement
		if p.curTok.Type == lexer.LBRACE_PCT && isClosingTag(p.peekTok.Type) {
			return body, p.peekTok.Type
		}

		stmt := p.parseStatement()
		if stmt != nil {
			// Check if this statement is an end/else statement (signals end of block)
			// But when/elsif should be included in the body AND signal continuation
			switch s := stmt.(type) {
			case *ast.EndStatement, *ast.ElseStatement:
				// Don't add to body - advance past closing %} and return
				p.nextToken()
				return body, p.curTok.Type
			case *ast.ElsifStatement, *ast.WhenStatement:
				// Add to body but also return the type so the caller knows what came next
				body.Statements = append(body.Statements, s)
				// Advance past the statement and return
				p.nextToken()
				switch s.(type) {
				case *ast.ElsifStatement:
					return body, lexer.ELSIF
				case *ast.WhenStatement:
					return body, lexer.WHEN
				}
			default:
				// Skip empty text content
				if tc, ok := stmt.(*ast.TextContent); ok && tc.Value == "" {
					// skip
				} else {
					body.Statements = append(body.Statements, stmt)
				}
			}
		}
		p.nextToken()
	}
}

// isClosingTag checks if a token type is a closing tag keyword.
func isClosingTag(t lexer.TokenType) bool {
	switch t {
	case lexer.END, lexer.ELSE,
		lexer.ENDRAW, lexer.ENDCOMMENT:
		return true
	}
	return false
}

// consumeEndTag consumes the {% end %} or {% endraw %} or {% endcomment %} tag.
func (p *Parser) consumeEndTag() bool {
	// Handle case where curTok is at the closing keyword directly
	if p.curTokIs(lexer.END) || p.curTokIs(lexer.ENDRAW) || p.curTokIs(lexer.ENDCOMMENT) {
		p.nextToken() // consume keyword, now at RBRACE_PCT
		return true
	}

	// Handle case where curTok is at LBRACE_PCT and peek is the closing keyword
	if p.curTokIs(lexer.LBRACE_PCT) && isClosingTag(p.peekTok.Type) {
		p.nextToken() // consume LBRACE_PCT, now at closing keyword
		p.nextToken() // consume keyword, now at RBRACE_PCT
		return true
	}

	return false
}

// parseWispStatement parses a Wisp statement inside {% %}.
func (p *Parser) parseWispStatement() ast.Statement {
	// We're at LBRACE_PCT, now look at what's inside
	p.nextToken() // move past {%

	// Check what type of statement this is
	switch p.curTok.Type {
	case lexer.DOT:
		// Variable access or assignment: {% .name %} or {% .name = value %}
		return p.parseWispVariableStatement()
	case lexer.IF:
		// Conditional: {% if .condition %}
		return p.parseWispIfStatement()
	case lexer.UNLESS:
		// Negated conditional: {% unless .condition %}
		return p.parseWispUnlessStatement()
	case lexer.ASSIGN:
		// Variable assignment: {% assign .name = value %}
		return p.parseWispAssignStatement()
	case lexer.FOR:
		// Loop: {% for .item in .items %}
		return p.parseWispForStatement()
	case lexer.WHILE:
		// Conditional loop: {% while .condition %}
		return p.parseWispWhileStatement()
	case lexer.RANGE:
		// Range loop: {% range .start .end %}
		return p.parseWispRangeStatement()
	case lexer.CASE:
		// Switch statement: {% case .value %}
		return p.parseWispCaseStatement()
	case lexer.WITH:
		// Context block: {% with .user as .currentUser %}
		return p.parseWispWithStatement()
	case lexer.CYCLE:
		// Cycle tag: {% cycle 'odd' 'even' %}
		return p.parseWispCycleStatement()
	case lexer.INCREMENT:
		// Increment tag: {% increment .counter %}
		return p.parseWispIncrementStatement()
	case lexer.DECREMENT:
		// Decrement tag: {% decrement .counter %}
		return p.parseWispDecrementStatement()
	case lexer.BREAK:
		// Break tag: {% break %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.BreakStatement{Token: p.curTok}
	case lexer.CONTINUE:
		// Continue tag: {% continue %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.ContinueStatement{Token: p.curTok}
	case lexer.INCLUDE:
		// Include tag: {% include "template" %}
		return p.parseWispIncludeStatement()
	case lexer.RENDER:
		// Render tag: {% render "template" .data %}
		return p.parseWispRenderStatement()
	case lexer.COMPONENT:
		// Component tag: {% component "Button" .props %}
		return p.parseWispComponentStatement()
	case lexer.EXTENDS:
		// Extends tag: {% extends "layout" %}
		return p.parseWispExtendsStatement()
	case lexer.BLOCK:
		// Block tag: {% block name %}
		return p.parseWispBlockStatement()
	case lexer.CONTENT:
		// Content tag: {% content %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.ContentStatement{Token: p.curTok}
	case lexer.RAW:
		// Raw block: {% raw %}
		return p.parseWispRawStatement()
	case lexer.COMMENT:
		// Comment block: {% comment %}
		return p.parseWispCommentStatement()
	case lexer.END:
		// End tag: {% end %}
		p.expectPeek(lexer.RBRACE_PCT)
		p.PopScope()
		return &ast.EndStatement{Token: p.curTok}
	case lexer.ENDRAW:
		// Endraw tag: {% endraw %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.EndStatement{Token: p.curTok}
	case lexer.ENDCOMMENT:
		// Endcomment tag: {% endcomment %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.EndStatement{Token: p.curTok}
	case lexer.ELSE:
		// Else tag: {% else %}
		p.expectPeek(lexer.RBRACE_PCT)
		return &ast.ElseStatement{Token: p.curTok}
	case lexer.ELSIF:
		// Elsif tag: {% elsif .condition %}
		return p.parseWispElsifStatement()
	case lexer.WHEN:
		// When tag: {% when "value" %}
		return p.parseWispWhenStatement()
	default:
		// Expression statement
		return p.parseWispExpressionStatement()
	}
}

// parseWispVariableStatement parses a variable access or assignment statement.
func (p *Parser) parseWispVariableStatement() ast.Statement {
	// Check if this is a pipe expression: . | function
	if p.peekTokIs(lexer.PIPE) {
		// Create a dot expression for the '.'
		dotExpr := &ast.DotExpression{Token: p.curTok}
		p.nextToken() // consume DOT, now at PIPE

		// Now parse the pipe expression
		// The pipe expression will be parsed by parsePipeExpression
		p.nextToken() // move past PIPE to the identifier
		pipeExpr := parsePipeExpression(p)

		// Expect %}
		if !p.expectPeek(lexer.RBRACE_PCT) {
			return nil
		}

		return &ast.ExpressionStatement{
			Token:      dotExpr.Token,
			Expression: pipeExpr,
		}
	}

	// Otherwise, parse as dot expression
	dotExpr := parseDotExpression(p)

	// Type assert to DotExpression
	dotExprTyped, ok := dotExpr.(*ast.DotExpression)
	if !ok {
		p.errors = append(p.errors, "expected dot expression")
		return nil
	}

	// Check if this is an assignment
	if p.peekTokIs(lexer.ASSIGN_OP) {
		// Assignment: {% .name = value %}
		p.nextToken() // consume =
		p.nextToken() // move to value

		// Parse the value expression
		value := p.parseExpression(LOWEST)

		// Expect %}
		if !p.expectPeek(lexer.RBRACE_PCT) {
			return nil
		}

		// Create assignment statement
		return &ast.AssignStatement{
			Token: dotExprTyped.Token,
			Name:  dotExprTyped.Field,
			Value: value,
		}
	}

	// Just a variable access: {% .name %}
	// parseDotExpression no longer advances, so expectPeek should work
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return &ast.ExpressionStatement{
		Token:      dotExprTyped.Token,
		Expression: dotExprTyped,
	}
}

// parseWispIfStatement parses an if statement: {% if .condition %}
func (p *Parser) parseWispIfStatement() ast.Statement {
	stmt := &ast.IfStatement{Token: p.curTok}

	// Parse the condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Create a new scope for the if block
	p.PushScope()

	// Parse the consequence (statements between {% if %} and {% else %} or {% end %})
	consequence := &ast.BlockStatement{Token: p.curTok}
	consequence.Statements = []ast.Statement{}

	// Move to the next token to start parsing the body
	p.nextToken()

	// Parse statements until we encounter else or end
	for !p.curTokIs(lexer.ELSE) && !p.curTokIs(lexer.END) && !p.curTokIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			// Check if this is an else or end statement
			if _, isElse := stmt.(*ast.ElseStatement); isElse {
				// Stop parsing consequence
				break
			}
			if _, isEnd := stmt.(*ast.EndStatement); isEnd {
				// Stop parsing consequence
				break
			}
			consequence.Statements = append(consequence.Statements, stmt)
		}
		p.nextToken()
	}

	stmt.Consequence = consequence

	// Check if there's an else branch
	if p.curTokIs(lexer.ELSE) {
		// Parse the else branch
		p.nextToken() // consume else

		// Parse the alternative (statements between {% else %} and {% end %})
		alternative := &ast.BlockStatement{Token: p.curTok}
		alternative.Statements = []ast.Statement{}

		// Parse statements until we encounter end
		for !p.curTokIs(lexer.END) && !p.curTokIs(lexer.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				alternative.Statements = append(alternative.Statements, stmt)
			}
			p.nextToken()
		}

		stmt.Alternative = alternative
	}

	// Consume the end tag
	if p.curTokIs(lexer.END) {
		p.nextToken() // consume end
		if !p.expectPeek(lexer.RBRACE_PCT) {
			return nil
		}
	}

	// Restore the parent scope
	p.PopScope()

	return stmt
}

// parseWispUnlessStatement parses an unless statement: {% unless .condition %}
func (p *Parser) parseWispUnlessStatement() ast.Statement {
	stmt := &ast.UnlessStatement{Token: p.curTok}

	// Parse the condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body statements
	p.PushScope()
	body, _ := p.parseBodyStatements()
	stmt.Consequence = body
	p.PopScope()

	// Consume the end tag
	p.consumeEndTag()

	return stmt
}

// parseWispAssignStatement parses an assign statement: {% assign .name = value %}
func (p *Parser) parseWispAssignStatement() ast.Statement {
	stmt := &ast.AssignStatement{Token: p.curTok}

	// Expect a dot
	if !p.expectPeek(lexer.DOT) {
		return nil
	}

	// Parse the variable name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}

	// Expect =
	if !p.expectPeek(lexer.ASSIGN_OP) {
		return nil
	}

	// Parse the value
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispForStatement parses a for loop: {% for .item in .items %}
func (p *Parser) parseWispForStatement() ast.Statement {
	stmt := &ast.ForStatement{Token: p.curTok}

	// Parse the loop variable (with or without index)
	if p.peekTokIs(lexer.DOT) {
		p.nextToken() // consume 'for'
		p.nextToken() // consume dot
		// We're now at the identifier, so just use it
		if p.curTok.Type != lexer.IDENT {
			p.errors = append(p.errors, fmt.Sprintf("expected identifier after dot, got %s", p.curTok.Type))
			return nil
		}
		stmt.LoopVar = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Check for index variable: {% for .i, .item in .items %}
	if p.peekTokIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next identifier
		if p.curTokIs(lexer.DOT) {
			p.nextToken() // consume dot
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			stmt.IndexVar = stmt.LoopVar
			stmt.LoopVar = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
		}
	}

	// Expect 'in'
	if !p.expectPeek(lexer.IN) {
		return nil
	}

	// Parse the collection
	p.nextToken()
	stmt.Collection = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body statements
	p.PushScope()
	body, _ := p.parseBodyStatements()
	stmt.Body = body
	p.PopScope()

	// Consume the end tag
	p.consumeEndTag()

	return stmt
}

// parseWispWhileStatement parses a while loop: {% while .condition %}
func (p *Parser) parseWispWhileStatement() ast.Statement {
	stmt := &ast.WhileStatement{Token: p.curTok}

	// Parse the condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body statements
	p.PushScope()
	body, _ := p.parseBodyStatements()
	stmt.Body = body
	p.PopScope()

	// Consume the end tag
	p.consumeEndTag()

	return stmt
}

// parseWispRangeStatement parses a range loop: {% range .start .end %}
func (p *Parser) parseWispRangeStatement() ast.Statement {
	stmt := &ast.RangeStatement{Token: p.curTok}

	// Parse start value
	p.nextToken()
	stmt.Start = p.parseExpression(LOWEST)

	// Parse end value
	p.nextToken()
	stmt.End = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body statements
	p.PushScope()
	body, _ := p.parseBodyStatements()
	stmt.Body = body
	p.PopScope()

	// Consume the end tag
	p.consumeEndTag()

	return stmt
}

// parseWispCaseStatement parses a case statement: {% case .value %}
func (p *Parser) parseWispCaseStatement() ast.Statement {
	stmt := &ast.CaseStatement{Token: p.curTok}

	// Parse the value
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body - collect all statements until {% end %}
	// Unlike other blocks, case bodies include when and else clauses
	p.PushScope()
	body := &ast.BlockStatement{Statements: []ast.Statement{}}

	p.nextToken()
	for {
		// Stop at end tag
		if p.curTok.Type == lexer.END {
			break
		}
		if p.curTok.Type == lexer.EOF {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			// Include everything except EndStatement
			if _, isEnd := stmt.(*ast.EndStatement); isEnd {
				break
			}
			body.Statements = append(body.Statements, stmt)
		}
		p.nextToken()
	}

	stmt.Body = body
	p.PopScope()

	// Consume the end tag (%})
	if p.curTokIs(lexer.END) {
		p.nextToken() // move to %}
	}

	return stmt
}

// parseWispWithStatement parses a with block: {% with .user as .currentUser %}
func (p *Parser) parseWispWithStatement() ast.Statement {
	stmt := &ast.WithStatement{Token: p.curTok}

	// Parse the source expression
	p.nextToken()
	stmt.Source = p.parseExpression(LOWEST)

	// Advance past the expression to reach AS
	p.nextToken()

	// After parsing expression and advancing, curTok should be AS
	if !p.curTokIs(lexer.AS) {
		p.errors = append(p.errors, fmt.Sprintf("expected AS, got %s", p.curTok.Type))
		return nil
	}

	// Parse the target variable
	p.nextToken()
	if p.curTokIs(lexer.DOT) {
		p.nextToken()
		// After consuming DOT, curTok should be IDENT
		if p.curTok.Type != lexer.IDENT {
			p.errors = append(p.errors, fmt.Sprintf("expected identifier after dot, got %s", p.curTok.Type))
			return nil
		}
		stmt.Target = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Consume the RBRACE_PCT token
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse the body statements
	p.PushScope()
	body, _ := p.parseBodyStatements()
	stmt.Body = body
	p.PopScope()

	// Consume the end tag
	p.consumeEndTag()

	return stmt
}

// parseWispCycleStatement parses a cycle tag: {% cycle 'odd' 'even' %}
func (p *Parser) parseWispCycleStatement() ast.Statement {
	stmt := &ast.CycleStatement{Token: p.curTok}

	// Parse the values
	for !p.peekTokIs(lexer.RBRACE_PCT) && !p.peekTokIs(lexer.EOF) {
		p.nextToken()
		value := p.parseExpression(LOWEST)
		if value != nil {
			stmt.Values = append(stmt.Values, value)
		}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispIncrementStatement parses an increment tag: {% increment .counter %}
func (p *Parser) parseWispIncrementStatement() ast.Statement {
	stmt := &ast.IncrementStatement{Token: p.curTok}

	// Parse the variable
	p.nextToken()
	if p.curTokIs(lexer.DOT) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispDecrementStatement parses a decrement tag: {% decrement .counter %}
func (p *Parser) parseWispDecrementStatement() ast.Statement {
	stmt := &ast.DecrementStatement{Token: p.curTok}

	// Parse the variable
	p.nextToken()
	if p.curTokIs(lexer.DOT) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Variable = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispIncludeStatement parses an include tag: {% include "template" %}
func (p *Parser) parseWispIncludeStatement() ast.Statement {
	stmt := &ast.IncludeStatement{Token: p.curTok}

	// Parse the template name
	p.nextToken()
	if p.curTokIs(lexer.STRING) {
		stmt.Template = &ast.StringLiteral{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Optional context
	if p.peekTokIs(lexer.DOT) {
		p.nextToken()
		stmt.Context = parseDotExpression(p)
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispRenderStatement parses a render tag: {% render "template" .data %}
func (p *Parser) parseWispRenderStatement() ast.Statement {
	stmt := &ast.RenderStatement{Token: p.curTok}

	// Parse the template name
	p.nextToken()
	if p.curTokIs(lexer.STRING) {
		stmt.Template = &ast.StringLiteral{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Parse parameters
	for !p.peekTokIs(lexer.RBRACE_PCT) && !p.peekTokIs(lexer.EOF) {
		p.nextToken()
		param := p.parseExpression(LOWEST)
		if param != nil {
			stmt.Params = append(stmt.Params, param)
		}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispComponentStatement parses a component tag: {% component "Button" .props %}
func (p *Parser) parseWispComponentStatement() ast.Statement {
	stmt := &ast.ComponentStatement{Token: p.curTok}

	// Parse the component name
	p.nextToken()
	if p.curTokIs(lexer.STRING) {
		stmt.Name = &ast.StringLiteral{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Parse props
	for !p.peekTokIs(lexer.RBRACE_PCT) && !p.peekTokIs(lexer.EOF) {
		p.nextToken()
		prop := p.parseExpression(LOWEST)
		if prop != nil {
			stmt.Props = append(stmt.Props, prop)
		}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispExtendsStatement parses an extends tag: {% extends "layout" %}
func (p *Parser) parseWispExtendsStatement() ast.Statement {
	stmt := &ast.ExtendsStatement{Token: p.curTok}

	// Parse the layout name
	p.nextToken()
	if p.curTokIs(lexer.STRING) {
		stmt.Layout = &ast.StringLiteral{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispBlockStatement parses a block tag: {% block name %}
func (p *Parser) parseWispBlockStatement() ast.Statement {
	stmt := &ast.BlockTagStatement{Token: p.curTok}

	// Parse the block name
	p.nextToken()
	if p.curTokIs(lexer.IDENT) {
		stmt.Name = &ast.Identifier{Token: p.curTok, Value: p.curTok.Literal}
	}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispRawStatement parses a raw block: {% raw %}
// At entry: curTok is RAW, lexer position is after "raw" keyword
func (p *Parser) parseWispRawStatement() ast.Statement {
	stmt := &ast.RawStatement{Token: p.curTok}
	input := p.l.GetInput()

	// The lexer position is after "raw" keyword
	// We need to find the %} that closes {% raw %}
	// Search backward from current position to find %}
	searchStart := p.l.GetPosition()
	rawTagEnd := searchStart

	// Search backward for %} to find end of {% raw %} tag
	for i := searchStart; i >= 1; i-- {
		if input[i-1] == '%' && input[i] == '}' {
			rawTagEnd = i + 1
			break
		}
	}

	// Now search for {% endraw %} from rawTagEnd
	for i := rawTagEnd; i < len(input)-7; i++ {
		if input[i] != '{' || input[i+1] != '%' {
			continue
		}
		j := i + 2
		for j < len(input) && (input[j] == ' ' || input[j] == '\t' || input[j] == '\n' || input[j] == '\r') {
			j++
		}
		if j+6 <= len(input) && input[j:j+6] == "endraw" {
			stmt.Content = input[rawTagEnd:i]

			// Calculate position after {% endraw %}
			j += 6
			for j < len(input) && (input[j] == ' ' || input[j] == '\t' || input[j] == '\n' || input[j] == '\r') {
				j++
			}
			if j+1 < len(input) && input[j] == '%' && input[j+1] == '}' {
				j += 2
			}

			// Sync lexer to position after {% endraw %}
			p.l.SetPosition(j)

			// Fetch tokens from the new position
			p.curTok = p.l.NextToken()
			p.peekTok = p.l.NextToken()

			// Tell ParseProgram to skip the next advance
			p.skipAdvance = true

			return stmt
		}
	}

	// No {% endraw %} found - set EOF to stop parsing
	p.curTok = lexer.NewToken(lexer.EOF, "", 0, 0)
	return stmt
}

// parseWispCommentStatement parses a comment block: {% comment %}
func (p *Parser) parseWispCommentStatement() ast.Statement {
	stmt := &ast.CommentStatement{Token: p.curTok}

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	// Parse body statements until {% endcomment %} or {% end %} (skip content)
	body, closingType := p.parseBodyStatements()
	_ = body // Comments discard their body content

	// Consume the closing tag
	if closingType == lexer.ENDCOMMENT || closingType == lexer.END {
		p.consumeEndTag()
	}

	return stmt
}

// parseWispElsifStatement parses an elsif statement: {% elsif .condition %}
func (p *Parser) parseWispElsifStatement() ast.Statement {
	stmt := &ast.ElsifStatement{Token: p.curTok}

	// Parse the condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispWhenStatement parses a when statement: {% when "value" %}
func (p *Parser) parseWispWhenStatement() ast.Statement {
	stmt := &ast.WhenStatement{Token: p.curTok}

	// Parse the value
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}

// parseWispExpressionStatement parses an expression statement inside {% %}.
func (p *Parser) parseWispExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curTok}

	stmt.Expression = p.parseExpression(LOWEST)

	// Expect %}
	if !p.expectPeek(lexer.RBRACE_PCT) {
		return nil
	}

	return stmt
}
