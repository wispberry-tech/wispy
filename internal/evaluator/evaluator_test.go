package evaluator

import (
	"fmt"
	"testing"

	"template-wisp/internal/lexer"
	"template-wisp/internal/parser"
	"template-wisp/internal/scope"
)

func TestEvaluateExpression(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("name", "John")
	s.Set("age", 30)

	e := NewEvaluator(s)

	// Test evaluating a simple expression
	l := lexer.NewLexer("{% .name %}")
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "John" {
		t.Errorf("Expected 'John', got %q", output)
	}
}

func TestEvaluateExpressionStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("x", 42)

	e := NewEvaluator(s)

	// Test evaluating an expression statement
	l := lexer.NewLexer("{% .x %}")
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "42" {
		t.Errorf("Expected '42', got %q", output)
	}
}

func TestEvaluateAssignStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)

	// Test evaluating an assign statement
	l := lexer.NewLexer(`{% assign .name = "Alice" %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	_, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}

	// Check that the variable was set
	val, ok := s.Get("name")
	if !ok {
		t.Error("Variable 'name' not found in scope")
	}
	if val != "Alice" {
		t.Errorf("Expected 'Alice', got %v", val)
	}
}

func TestEvaluateIfStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("condition", true)
	s.Set("value", "yes")

	e := NewEvaluator(s)

	// Test evaluating an if statement
	l := lexer.NewLexer(`{% if .condition %}{% .value %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "yes" {
		t.Errorf("Expected 'yes', got %q", output)
	}
}

func TestEvaluateIfElseStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("condition", false)
	s.Set("value", "no")

	e := NewEvaluator(s)

	// Test evaluating an if-else statement
	l := lexer.NewLexer(`{% if .condition %}yes{% else %}{% .value %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "no" {
		t.Errorf("Expected 'no', got %q", output)
	}
}

func TestEvaluateUnlessStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("condition", false)
	s.Set("value", "yes")

	e := NewEvaluator(s)

	// Test evaluating an unless statement
	l := lexer.NewLexer(`{% unless .condition %}{% .value %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "yes" {
		t.Errorf("Expected 'yes', got %q", output)
	}
}

func TestEvaluateForStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	// Create a simple slice
	items := []interface{}{"a", "b", "c"}
	s.Set("items", items)

	e := NewEvaluator(s)

	// Test evaluating a for statement
	l := lexer.NewLexer(`{% for .item in .items %}{% .item %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "abc" {
		t.Errorf("Expected 'abc', got %q", output)
	}
}

func TestEvaluateWithStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	type User struct {
		Name string
	}
	user := User{Name: "Bob"}
	s.Set("user", user)

	e := NewEvaluator(s)

	// Test evaluating a with statement
	l := lexer.NewLexer(`{% with .user as .currentUser %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	_, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}

	// The with statement creates a new scope, so the variable is set in the child scope
	// We can't check it from the parent scope
	// Instead, we just verify that the evaluation succeeded
}

func TestEvaluateCommentStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)

	// Test evaluating a comment statement
	l := lexer.NewLexer(`{% comment %}This is a comment{% endcomment %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty string, got %q", output)
	}
}

func TestEvaluateRawStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% raw %}{{ .name }}{% endraw %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "{{ .name }}" {
		t.Errorf("Expected '{{ .name }}', got %q", output)
	}
}

func TestEvaluateCaseStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("status", "active")

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% case .status %}{% when "active" %}ON{% when "inactive" %}OFF{% else %}DEFAULT{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "ON" {
		t.Errorf("Expected 'ON', got %q", output)
	}
}

func TestEvaluateCaseStatementDefault(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("status", "unknown")

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% case .status %}{% when "active" %}ON{% when "inactive" %}OFF{% else %}DEFAULT{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "DEFAULT" {
		t.Errorf("Expected 'DEFAULT', got %q", output)
	}
}

func TestEvaluateCycleStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% cycle "odd" "even" %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "odd" {
		t.Errorf("Expected 'odd', got %q", output)
	}
}

func TestEvaluateBreakStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	items := []interface{}{"a", "b", "c", "d"}
	s.Set("items", items)
	s.Set("stop", "b")

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% for .item in .items %}{% .item %}{% if .item == .stop %}{% break %}{% end %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "ab" {
		t.Errorf("Expected 'ab', got %q", output)
	}
}

func TestEvaluateContinueStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	items := []interface{}{"a", "skip", "c"}
	s.Set("items", items)

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% for .item in .items %}{% if .item == "skip" %}{% continue %}{% end %}{% .item %}{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "ac" {
		t.Errorf("Expected 'ac', got %q", output)
	}
}

func TestEvaluateComponentStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)
	e.SetTemplateFn(func(name string) (string, error) {
		if name == "button" {
			return `<button>{% .title %}</button>`, nil
		}
		return "", fmt.Errorf("template not found: %s", name)
	})

	l := lexer.NewLexer(`{% component "button" .title %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	s.Set("title", "Click Me")

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "<button>Click Me</button>" {
		t.Errorf("Expected '<button>Click Me</button>', got %q", output)
	}
}

func TestEvaluateExtendsStatement(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)
	e.SetTemplateFn(func(name string) (string, error) {
		if name == "layout" {
			return `<html>{% block title %}Default Title{% end %}<body>{% content %}</body></html>`, nil
		}
		return "", fmt.Errorf("template not found: %s", name)
	})

	l := lexer.NewLexer(`{% extends "layout" %}{% block title %}My Page{% end %}{% content %}Hello World{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	expected := "<html>My Page<body>Hello World</body></html>"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestEvaluateExtendsWithDefaultBlock(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)
	e.SetTemplateFn(func(name string) (string, error) {
		if name == "layout" {
			return `<html>{% block title %}Default{% end %}</html>`, nil
		}
		return "", fmt.Errorf("template not found: %s", name)
	})

	// Child template that doesn't override the title block
	l := lexer.NewLexer(`{% extends "layout" %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	// With no override, the block's default content should be output
	expected := "<html>Default</html>"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestAutoEscaping(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("html", "<script>alert('xss')</script>")

	// With auto-escaping (default)
	e := NewEvaluator(s)
	l := lexer.NewLexer(`{% .html %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestAutoEscapingDisabled(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("html", "<b>bold</b>")

	e := NewEvaluator(s)
	e.SetAutoEscape(false)
	l := lexer.NewLexer(`{% .html %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "<b>bold</b>" {
		t.Errorf("Expected '<b>bold</b>', got %q", output)
	}
}

func TestSafeStringBypassesEscaping(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("html", SafeString{Value: "<b>safe</b>"})

	e := NewEvaluator(s)
	l := lexer.NewLexer(`{% .html %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	if output != "<b>safe</b>" {
		t.Errorf("Expected '<b>safe</b>', got %q", output)
	}
}

func TestMaxIterations(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	s.Set("x", true)

	e := NewEvaluator(s)
	e.SetMaxIterations(5)
	l := lexer.NewLexer(`{% while .x %}loop{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	_, err := e.Evaluate(program)
	if err == nil {
		t.Error("Expected iteration limit error")
	}
}

func TestEvaluateRawWithContentAfter(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)

	l := lexer.NewLexer(`{% raw %}{% if .x %}test{% end %}{% endraw %}AFTER`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	expected := "{% if .x %}test{% end %}AFTER"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestEvaluateExtendsWithRawBlock(t *testing.T) {
	s := scope.NewScope()
	defer s.Release()

	e := NewEvaluator(s)
	e.SetTemplateFn(func(name string) (string, error) {
		if name == "layout" {
			return `<html>{% block body %}{% end %}</html>`, nil
		}
		return "", fmt.Errorf("template not found: %s", name)
	})

	l := lexer.NewLexer(`{% extends "layout" %}{% block body %}<pre><code>{% raw %}{% if .x %}test{% end %}{% endraw %}</code></pre>{% end %}`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	output, err := e.Evaluate(program)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	expected := `<html><pre><code>{% if .x %}test{% end %}</code></pre></html>`
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}
