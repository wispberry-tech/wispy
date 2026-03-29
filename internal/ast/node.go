// internal/ast/node.go
package ast

// Node is the base interface for all AST nodes.
type Node interface{ groveNode() }

// Program is the root node.
type Program struct{ Body []Node }

func (*Program) groveNode() {}

// ─── Statement nodes ──────────────────────────────────────────────────────────

// TextNode holds raw text content (no interpolation).
type TextNode struct {
	Value string
	Line  int
}

func (*TextNode) groveNode() {}

// OutputNode holds an {{ expression }} to be evaluated and printed.
type OutputNode struct {
	Expr       Node
	StripLeft  bool
	StripRight bool
	Line       int
}

func (*OutputNode) groveNode() {}

// RawNode holds content from {% raw %}...{% endraw %} — printed verbatim.
type RawNode struct {
	Value string
	Line  int
}

func (*RawNode) groveNode() {}

// TagNode is an unrecognised or deferred tag (e.g. if/for/extends).
// The parser uses this as a placeholder for tags handled in later plans,
// and to reject banned tags (extends/import) in inline mode.
type TagNode struct {
	Name string
	Line int
}

func (*TagNode) groveNode() {}

// ─── Expression nodes ─────────────────────────────────────────────────────────

// NilLiteral is the nil/null literal.
type NilLiteral struct{ Line int }

func (*NilLiteral) groveNode() {}

// BoolLiteral is true or false.
type BoolLiteral struct {
	Value bool
	Line  int
}

func (*BoolLiteral) groveNode() {}

// IntLiteral is an integer literal.
type IntLiteral struct {
	Value int64
	Line  int
}

func (*IntLiteral) groveNode() {}

// FloatLiteral is a floating-point literal.
type FloatLiteral struct {
	Value float64
	Line  int
}

func (*FloatLiteral) groveNode() {}

// StringLiteral is a quoted string literal.
type StringLiteral struct {
	Value string
	Line  int
}

func (*StringLiteral) groveNode() {}

// Identifier is a variable reference.
type Identifier struct {
	Name string
	Line int
}

func (*Identifier) groveNode() {}

// AttributeAccess is obj.key — resolves key on obj.
type AttributeAccess struct {
	Object Node
	Key    string
	Line   int
}

func (*AttributeAccess) groveNode() {}

// IndexAccess is obj[key] — integer or string key.
type IndexAccess struct {
	Object Node
	Key    Node
	Line   int
}

func (*IndexAccess) groveNode() {}

// BinaryExpr is left op right.
// Op is one of: + - * / % ~ == != < <= > >= and or
type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
	Line  int
}

func (*BinaryExpr) groveNode() {}

// UnaryExpr is op operand.
// Op is one of: not -
type UnaryExpr struct {
	Op      string
	Operand Node
	Line    int
}

func (*UnaryExpr) groveNode() {}

// TernaryExpr is: Consequence if Condition else Alternative
// (Grove syntax: `value if cond else fallback`)
type TernaryExpr struct {
	Condition   Node
	Consequence Node
	Alternative Node
	Line        int
}

func (*TernaryExpr) groveNode() {}

// FilterExpr applies Filter(Args...) to Value.
// e.g. name | truncate(20, "…") → FilterExpr{Value: Identifier{name}, Filter: "truncate", Args: [20, "…"]}
type FilterExpr struct {
	Value  Node
	Filter string
	Args   []Node
	Line   int
}

func (*FilterExpr) groveNode() {}

// ─── Control flow nodes ───────────────────────────────────────────────────────

// ElifClause is a single elif branch in an IfNode.
type ElifClause struct {
	Condition Node
	Body      []Node
}

// IfNode is {% if cond %}...{% elif cond %}...{% else %}...{% endif %}.
type IfNode struct {
	Condition Node
	Body      []Node
	Elifs     []ElifClause
	Else      []Node // nil if no else branch
	Line      int
}

func (*IfNode) groveNode() {}

// UnlessNode is {% unless cond %}...{% endunless %} — equivalent to if not cond.
type UnlessNode struct {
	Condition Node
	Body      []Node
	Line      int
}

func (*UnlessNode) groveNode() {}

// ForNode is {% for var in iterable %}...{% empty %}...{% endfor %}.
// If Var2 is non-empty, it's a two-variable form (for k,v in map / for i,item in list).
type ForNode struct {
	Var1     string
	Var2     string // empty for single-var form
	Iterable Node
	Body     []Node
	Empty    []Node // nil if no {% empty %}
	Line     int
}

func (*ForNode) groveNode() {}

// SetNode is {% set name = expr %}.
type SetNode struct {
	Name string
	Expr Node
	Line int
}

func (*SetNode) groveNode() {}

// WithNode is {% with %}...{% endwith %} — creates an isolated scope.
type WithNode struct {
	Body []Node
	Line int
}

func (*WithNode) groveNode() {}

// CaptureNode is {% capture name %}...{% endcapture %} — renders body to a string variable.
type CaptureNode struct {
	Name string
	Body []Node
	Line int
}

func (*CaptureNode) groveNode() {}

// FuncCallNode is a function call expression: name(args...).
// Only built-in functions are supported in Plan 2: range().
type FuncCallNode struct {
	Name string
	Args []Node
	Line int
}

func (*FuncCallNode) groveNode() {}
