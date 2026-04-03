// internal/ast/node.go
package ast

// Node is the base interface for all AST nodes.
type Node interface{ wispyNode() }

// Program is the root node.
type Program struct{ Body []Node }

func (*Program) wispyNode() {}

// ─── Statement nodes ──────────────────────────────────────────────────────────

// TextNode holds raw text content (no interpolation).
type TextNode struct {
	Value string
	Line  int
}

func (*TextNode) wispyNode() {}

// OutputNode holds an {{ expression }} to be evaluated and printed.
type OutputNode struct {
	Expr       Node
	StripLeft  bool
	StripRight bool
	Line       int
}

func (*OutputNode) wispyNode() {}

// RawNode holds content from {% raw %}...{% endraw %} — printed verbatim.
type RawNode struct {
	Value string
	Line  int
}

func (*RawNode) wispyNode() {}

// TagNode is an unrecognised or deferred tag (e.g. if/for/extends).
// The parser uses this as a placeholder for tags handled in later plans,
// and to reject banned tags (extends/import) in inline mode.
type TagNode struct {
	Name string
	Line int
}

func (*TagNode) wispyNode() {}

// ─── Expression nodes ─────────────────────────────────────────────────────────

// NilLiteral is the nil/null literal.
type NilLiteral struct{ Line int }

func (*NilLiteral) wispyNode() {}

// BoolLiteral is true or false.
type BoolLiteral struct {
	Value bool
	Line  int
}

func (*BoolLiteral) wispyNode() {}

// IntLiteral is an integer literal.
type IntLiteral struct {
	Value int64
	Line  int
}

func (*IntLiteral) wispyNode() {}

// FloatLiteral is a floating-point literal.
type FloatLiteral struct {
	Value float64
	Line  int
}

func (*FloatLiteral) wispyNode() {}

// StringLiteral is a quoted string literal.
type StringLiteral struct {
	Value string
	Line  int
}

func (*StringLiteral) wispyNode() {}

// Identifier is a variable reference.
type Identifier struct {
	Name string
	Line int
}

func (*Identifier) wispyNode() {}

// AttributeAccess is obj.key — resolves key on obj.
type AttributeAccess struct {
	Object Node
	Key    string
	Line   int
}

func (*AttributeAccess) wispyNode() {}

// IndexAccess is obj[key] — integer or string key.
type IndexAccess struct {
	Object Node
	Key    Node
	Line   int
}

func (*IndexAccess) wispyNode() {}

// BinaryExpr is left op right.
// Op is one of: + - * / % ~ == != < <= > >= and or
type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
	Line  int
}

func (*BinaryExpr) wispyNode() {}

// UnaryExpr is op operand.
// Op is one of: not -
type UnaryExpr struct {
	Op      string
	Operand Node
	Line    int
}

func (*UnaryExpr) wispyNode() {}

// TernaryExpr is: Condition ? Consequence : Alternative
type TernaryExpr struct {
	Condition   Node
	Consequence Node
	Alternative Node
	Line        int
}

func (*TernaryExpr) wispyNode() {}

// FilterExpr applies Filter(Args...) to Value.
// e.g. name | truncate(20, "…") → FilterExpr{Value: Identifier{name}, Filter: "truncate", Args: [20, "…"]}
type FilterExpr struct {
	Value  Node
	Filter string
	Args   []Node
	Line   int
}

func (*FilterExpr) wispyNode() {}

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

func (*IfNode) wispyNode() {}

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

func (*ForNode) wispyNode() {}

// SetNode is {% set name = expr %}.
type SetNode struct {
	Name string
	Expr Node
	Line int
}

func (*SetNode) wispyNode() {}

// CaptureNode is {% capture name %}...{% endcapture %} — renders body to a string variable.
type CaptureNode struct {
	Name string
	Body []Node
	Line int
}

func (*CaptureNode) wispyNode() {}

// FuncCallNode is a function call expression: name(args...).
// Only built-in functions are supported in Plan 2: range().
type FuncCallNode struct {
	Name string
	Args []Node
	Line int
}

func (*FuncCallNode) wispyNode() {}

// NamedArgNode is a key=value argument in a macro call: name="Alice".
type NamedArgNode struct {
	Key   string
	Value Node
	Line  int
}

func (*NamedArgNode) wispyNode() {}

// MacroParam is a single parameter in a macro definition.
type MacroParam struct {
	Name    string
	Default Node // nil = required parameter; non-nil = default expression
}

// MacroNode is {% macro name(p1, p2="default") %}...{% endmacro %}.
type MacroNode struct {
	Name   string
	Params []MacroParam
	Body   []Node
	Line   int
}

func (*MacroNode) wispyNode() {}

// MacroCallExpr is a macro call expression: name(args...) or ns.name(args...).
// Callee is an Identifier or AttributeAccess.
type MacroCallExpr struct {
	Callee    Node
	PosArgs   []Node
	NamedArgs []NamedArgNode
	Line      int
}

func (*MacroCallExpr) wispyNode() {}

// CallNode is {% call macro(args) %}body{% endcall %} — call with a caller body.
type CallNode struct {
	Callee    Node           // the macro being called (Identifier or AttributeAccess)
	PosArgs   []Node
	NamedArgs []NamedArgNode
	Body      []Node         // the caller() body
	Line      int
}

func (*CallNode) wispyNode() {}

// IncludeNode is {% include "name" [k=v ...] %}.
type IncludeNode struct {
	Name     string         // template name (string literal)
	WithVars []NamedArgNode // extra variables (empty = no with clause)
	Line     int
}

func (*IncludeNode) wispyNode() {}

// RenderNode is {% render "name" [with k=v, ...] %} — always isolated.
type RenderNode struct {
	Name     string
	WithVars []NamedArgNode
	Line     int
}

func (*RenderNode) wispyNode() {}

// ImportNode is {% import "name" as alias %}.
type ImportNode struct {
	Name  string // template name
	Alias string // namespace identifier
	Line  int
}

func (*ImportNode) wispyNode() {}

// ExtendsNode is {% extends "name" %} — must be the first non-whitespace node.
type ExtendsNode struct {
	Name string
	Line int
}

func (*ExtendsNode) wispyNode() {}

// BlockNode is {% block name %}...{% endblock %}.
// In an extending template, Block nodes define overrides.
// In a base template, Block nodes define named slots with default content.
type BlockNode struct {
	Name string
	Body []Node
	Line int
}

func (*BlockNode) wispyNode() {}

// PropsNode is {% props name, name2="default", ... %} — declares accepted props.
// Must appear at the top of a component template. Reuses MacroParam for params.
type PropsNode struct {
	Params []MacroParam
	Line   int
}

func (*PropsNode) wispyNode() {}

// FillNode is {% fill "name" %}...{% endfill %} inside a component call body.
// FillNode is NOT directly part of the template AST — it is consumed by the parser
// when parsing a ComponentNode and stored in ComponentNode.Fills.
type FillNode struct {
	Name string
	Body []Node
	Line int
}

// ComponentNode is {% component "name" k=v, ... %}...{% endcomponent %}.
type ComponentNode struct {
	Name        string         // template name (string literal)
	Props       []NamedArgNode // passed props (key=value pairs)
	DefaultFill []Node         // body content outside fill blocks → fed to {% slot %}
	Fills       []FillNode     // named {% fill %}...{% endfill %} blocks
	Line        int
}

func (*ComponentNode) wispyNode() {}

// SlotNode is {% slot ["name"] %}...{% endslot %} inside a component template.
type SlotNode struct {
	Name    string // "" = default slot
	Default []Node // fallback content rendered when no matching fill
	Line    int
}

func (*SlotNode) wispyNode() {}

// ─── Plan 7 nodes ─────────────────────────────────────────────────────────────

// AssetNode declares an asset (CSS/JS/other) to collect into RenderResult.Assets.
// Src and AssetType are required. Attrs holds remaining key=value attributes (defer, async, etc.).
// Priority controls ordering within the asset's type group (higher = earlier). Default 0.
type AssetNode struct {
	Src       string
	AssetType string         // from type= attr ("stylesheet", "script", etc.)
	Attrs     []NamedArgNode // remaining attrs; bare idents get Value=StringLiteral{""}
	Priority  int            // from priority= attr
	Line      int
}

func (*AssetNode) wispyNode() {}

// MetaNode declares a metadata entry for RenderResult.Meta.
// Key is the value of the name=, property=, or http-equiv= attribute.
// Value is the value of the content= attribute.
type MetaNode struct {
	Key   string
	Value string
	Line  int
}

func (*MetaNode) wispyNode() {}

// HoistNode collects its rendered body into RenderResult.Hoisted[Target].
// Target is a user-defined string (e.g. "head", "foot", "analytics").
type HoistNode struct {
	Target string
	Body   []Node
	Line   int
}

func (*HoistNode) wispyNode() {}
