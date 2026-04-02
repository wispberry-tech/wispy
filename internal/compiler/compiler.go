// internal/compiler/compiler.go
package compiler

import (
	"fmt"
	"strings"

	"grove/internal/ast"
)

// Compile walks prog and emits Bytecode.
func Compile(prog *ast.Program) (*Bytecode, error) {
	c := &cmp{nameIdx: make(map[string]int)}
	if err := c.compileProgram(prog); err != nil {
		return nil, err
	}
	c.emit(OP_HALT, 0, 0, 0)
	bc := &Bytecode{
		Instrs:     c.instrs,
		Consts:     c.consts,
		Names:      c.names,
		Macros:     c.macros,
		Blocks:     c.blocks,
		Extends:    c.extends,
		Props:      c.props,
		Components: c.components,
	}
	est := 0
	for _, cn := range bc.Consts {
		if s, ok := cn.(string); ok {
			est += len(s)
		}
	}
	bc.EstimatedOutputSize = est
	return bc, nil
}

type cmp struct {
	instrs     []Instruction
	consts     []any
	names      []string
	nameIdx    map[string]int
	macros     []MacroDef
	blocks     []BlockDef
	extends    string
	props      []MacroParam
	components []ComponentDef
}

func (c *cmp) compileProgram(prog *ast.Program) error {
	// Check for extends — must be first node (ignoring leading whitespace/raw text and props)
	extendsIdx := -1
scanLoop:
	for i, node := range prog.Body {
		if _, ok := node.(*ast.ExtendsNode); ok {
			extendsIdx = i
			break
		}
		// TextNode and PropsNode are allowed before extends; anything else stops the scan
		switch node.(type) {
		case *ast.TextNode, *ast.PropsNode:
			// continue scanning
		default:
			break scanLoop
		}
	}
	if extendsIdx >= 0 {
		return c.compileExtendsTemplate(prog, extendsIdx)
	}
	return c.compileBody(prog.Body)
}

func (c *cmp) compileExtendsTemplate(prog *ast.Program, extendsIdx int) error {
	extendsNode := prog.Body[extendsIdx].(*ast.ExtendsNode)
	c.extends = extendsNode.Name

	// Validate: nothing output-producing before extends; allow PropsNode
	for _, node := range prog.Body[:extendsIdx] {
		switch n := node.(type) {
		case *ast.TextNode:
			if strings.TrimSpace(n.Value) != "" {
				return fmt.Errorf("compiler: content before extends at line %d", extendsNode.Line)
			}
		case *ast.PropsNode:
			// Props declaration is allowed before extends — compile it now
			if err := c.compileNode(n); err != nil {
				return err
			}
		default:
			return fmt.Errorf("compiler: content before extends at line %d", extendsNode.Line)
		}
	}

	// Compile only block definitions from the remaining nodes
	for _, node := range prog.Body[extendsIdx+1:] {
		switch n := node.(type) {
		case *ast.BlockNode:
			if err := c.compileBlockDef(n); err != nil {
				return err
			}
		case *ast.TextNode:
			// whitespace between blocks — ignore
		default:
			return fmt.Errorf("compiler: only block definitions allowed in extending template (line %d)", extendsNode.Line)
		}
	}

	c.emit(OP_EXTENDS, uint16(c.addName(extendsNode.Name)), 0, 0)
	return nil
}

func (c *cmp) compileBlockDef(n *ast.BlockNode) error {
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	c.blocks = append(c.blocks, BlockDef{Name: n.Name, Body: subBytecode(sub)})
	return nil
}

func (c *cmp) compileBody(nodes []ast.Node) error {
	for _, node := range nodes {
		if err := c.compileNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (c *cmp) compileNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.TextNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.RawNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.OutputNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_OUTPUT, 0, 0, 0)

	case *ast.TagNode:
		// Unimplemented tags are no-ops
		return nil

	case *ast.IfNode:
		return c.compileIf(n)

	case *ast.UnlessNode:
		return c.compileUnless(n)

	case *ast.ForNode:
		return c.compileFor(n)

	case *ast.SetNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_STORE_VAR, uint16(c.addName(n.Name)), 0, 0)

	case *ast.WithNode:
		c.emit(OP_PUSH_SCOPE, 0, 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_POP_SCOPE, 0, 0, 0)

	case *ast.CaptureNode:
		c.emit(OP_CAPTURE_START, uint16(c.addName(n.Name)), 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_CAPTURE_END, uint16(c.addName(n.Name)), 0, 0)

	case *ast.MacroNode:
		return c.compileMacro(n)

	case *ast.CallNode:
		return c.compileCallNode(n)

	case *ast.IncludeNode:
		return c.compileInclude(n)

	case *ast.RenderNode:
		return c.compileRender(n)

	case *ast.ImportNode:
		return c.compileImport(n)

	case *ast.BlockNode:
		// Base template: compile default body into Blocks, emit OP_BLOCK_RENDER
		if err := c.compileBlockDef(n); err != nil {
			return err
		}
		blockIdx := len(c.blocks) - 1
		c.emit(OP_BLOCK_RENDER, uint16(c.addName(n.Name)), uint16(blockIdx), 0)

	case *ast.ExtendsNode:
		// Should not reach here — handled by compileProgram
		return fmt.Errorf("compiler: unexpected extends node in compileNode (should be handled by compileProgram)")

	case *ast.PropsNode:
		for _, p := range n.Params {
			mp := MacroParam{Name: p.Name}
			if p.Default != nil {
				mp.Default = constValueOf(p.Default)
			}
			c.props = append(c.props, mp)
		}
		c.emit(OP_PROPS_INIT, 0, 0, 0)

	case *ast.SlotNode:
		if len(n.Default) == 0 {
			c.emit(OP_SLOT, uint16(c.addName(n.Name)), 0xFFFF, 0)
		} else {
			sub := &cmp{nameIdx: make(map[string]int)}
			if err := sub.compileBody(n.Default); err != nil {
				return err
			}
			sub.emit(OP_HALT, 0, 0, 0)
			defaultBC := subBytecode(sub)
			c.blocks = append(c.blocks, BlockDef{Name: "__slot__:" + n.Name, Body: defaultBC})
			defaultIdx := len(c.blocks) - 1
			c.emit(OP_SLOT, uint16(c.addName(n.Name)), uint16(defaultIdx), 0)
		}

	case *ast.ComponentNode:
		return c.compileComponent(n)

	case *ast.AssetNode:
		return c.compileAsset(n)

	case *ast.MetaNode:
		return c.compileMeta(n)

	case *ast.HoistNode:
		return c.compileHoist(n)

	default:
		return fmt.Errorf("compiler: unknown node type %T", node)
	}
	return nil
}

// ─── {% if %} compiler ────────────────────────────────────────────────────────

func (c *cmp) compileIf(n *ast.IfNode) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)

	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	var endJumps []int
	endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
	c.instrs[jfIdx].A = uint16(len(c.instrs))

	for _, elif := range n.Elifs {
		if err := c.compileExpr(elif.Condition); err != nil {
			return err
		}
		elifJfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileBody(elif.Body); err != nil {
			return err
		}
		endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
		c.instrs[elifJfIdx].A = uint16(len(c.instrs))
	}

	if len(n.Else) > 0 {
		if err := c.compileBody(n.Else); err != nil {
			return err
		}
	}

	end := uint16(len(c.instrs))
	for _, jIdx := range endJumps {
		c.instrs[jIdx].A = end
	}

	return nil
}

// ─── {% unless %} compiler ────────────────────────────────────────────────────

func (c *cmp) compileUnless(n *ast.UnlessNode) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	c.emit(OP_NOT, 0, 0, 0)
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
	if err := c.compileBody(n.Body); err != nil {
		return err
	}
	c.instrs[jfIdx].A = uint16(len(c.instrs))
	return nil
}

// ─── {% for %} compiler ───────────────────────────────────────────────────────

func (c *cmp) compileFor(n *ast.ForNode) error {
	if err := c.compileExpr(n.Iterable); err != nil {
		return err
	}

	forInitIdx := c.emitPlaceholder(OP_FOR_INIT)

	loopTop := uint16(len(c.instrs))
	if n.Var2 == "" {
		c.emit(OP_FOR_BIND_1, uint16(c.addName(n.Var1)), 0, 0)
	} else {
		c.emit(OP_FOR_BIND_KV, uint16(c.addName(n.Var1)), uint16(c.addName(n.Var2)), 0)
	}

	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	c.emit(OP_FOR_STEP, loopTop, 0, 0)

	if len(n.Empty) > 0 {
		jumpPastEmptyIdx := c.emitPlaceholder(OP_JUMP)
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
		if err := c.compileBody(n.Empty); err != nil {
			return err
		}
		c.instrs[jumpPastEmptyIdx].A = uint16(len(c.instrs))
	} else {
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
	}

	return nil
}

// ─── Expression compiler ──────────────────────────────────────────────────────

func (c *cmp) compileExpr(node ast.Node) error {
	switch n := node.(type) {
	case *ast.NilLiteral:
		c.emit(OP_PUSH_NIL, 0, 0, 0)

	case *ast.BoolLiteral:
		c.emitPushConst(n.Value)

	case *ast.IntLiteral:
		c.emitPushConst(n.Value)

	case *ast.FloatLiteral:
		c.emitPushConst(n.Value)

	case *ast.StringLiteral:
		c.emitPushConst(n.Value)

	case *ast.Identifier:
		c.emit(OP_LOAD, uint16(c.addName(n.Name)), 0, 0)

	case *ast.AttributeAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		c.emit(OP_GET_ATTR, uint16(c.addName(n.Key)), 0, 0)

	case *ast.IndexAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		if err := c.compileExpr(n.Key); err != nil {
			return err
		}
		c.emit(OP_GET_INDEX, 0, 0, 0)

	case *ast.BinaryExpr:
		if err := c.compileExpr(n.Left); err != nil {
			return err
		}
		if err := c.compileExpr(n.Right); err != nil {
			return err
		}
		switch n.Op {
		case "+":
			c.emit(OP_ADD, 0, 0, 0)
		case "-":
			c.emit(OP_SUB, 0, 0, 0)
		case "*":
			c.emit(OP_MUL, 0, 0, 0)
		case "/":
			c.emit(OP_DIV, 0, 0, 0)
		case "%":
			c.emit(OP_MOD, 0, 0, 0)
		case "~":
			c.emit(OP_CONCAT, 0, 0, 0)
		case "==":
			c.emit(OP_EQ, 0, 0, 0)
		case "!=":
			c.emit(OP_NEQ, 0, 0, 0)
		case "<":
			c.emit(OP_LT, 0, 0, 0)
		case "<=":
			c.emit(OP_LTE, 0, 0, 0)
		case ">":
			c.emit(OP_GT, 0, 0, 0)
		case ">=":
			c.emit(OP_GTE, 0, 0, 0)
		case "and":
			c.emit(OP_AND, 0, 0, 0)
		case "or":
			c.emit(OP_OR, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown binary op %q", n.Op)
		}

	case *ast.UnaryExpr:
		if err := c.compileExpr(n.Operand); err != nil {
			return err
		}
		switch n.Op {
		case "not":
			c.emit(OP_NOT, 0, 0, 0)
		case "-":
			c.emit(OP_NEGATE, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown unary op %q", n.Op)
		}

	case *ast.TernaryExpr:
		if err := c.compileExpr(n.Condition); err != nil {
			return err
		}
		jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileExpr(n.Consequence); err != nil {
			return err
		}
		jIdx := c.emitPlaceholder(OP_JUMP)
		c.instrs[jfIdx].A = uint16(len(c.instrs))
		if err := c.compileExpr(n.Alternative); err != nil {
			return err
		}
		c.instrs[jIdx].A = uint16(len(c.instrs))

	case *ast.FilterExpr:
		if err := c.compileExpr(n.Value); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := c.compileExpr(arg); err != nil {
				return err
			}
		}
		c.emit(OP_FILTER, uint16(c.addName(n.Filter)), uint16(len(n.Args)), 0)

	case *ast.MacroCallExpr:
		return c.compileMacroCall(n.Callee, n.PosArgs, n.NamedArgs, false)

	case *ast.FuncCallNode:
		switch n.Name {
		case "range":
			for _, arg := range n.Args {
				if err := c.compileExpr(arg); err != nil {
					return err
				}
			}
			c.emit(OP_CALL_RANGE, uint16(len(n.Args)), 0, 0)
		case "caller":
			c.emit(OP_CALL_CALLER, 0, 0, 0)
		case "super":
			c.emit(OP_SUPER, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown function %q", n.Name)
		}

	default:
		return fmt.Errorf("compiler: unknown expr type %T", node)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (c *cmp) emit(op Opcode, a, b uint16, flags uint8) {
	c.instrs = append(c.instrs, Instruction{Op: op, A: a, B: b, Flags: flags})
}

// emitPlaceholder emits an instruction with A=0 and returns its index for back-patching.
func (c *cmp) emitPlaceholder(op Opcode) int {
	idx := len(c.instrs)
	c.emit(op, 0, 0, 0)
	return idx
}

func (c *cmp) emitPushConst(v any) {
	idx := len(c.consts)
	c.consts = append(c.consts, v)
	c.emit(OP_PUSH_CONST, uint16(idx), 0, 0)
}

func (c *cmp) addName(name string) int {
	if idx, ok := c.nameIdx[name]; ok {
		return idx
	}
	idx := len(c.names)
	c.names = append(c.names, name)
	c.nameIdx[name] = idx
	return idx
}

// ─── Plan 4 compile methods ───────────────────────────────────────────────────

// compileMacro compiles {% macro name(params) %}body{% endmacro %}.
func (c *cmp) compileMacro(n *ast.MacroNode) error {
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	bodyBC := subBytecode(sub)

	params := make([]MacroParam, len(n.Params))
	for i, p := range n.Params {
		params[i].Name = p.Name
		if p.Default != nil {
			params[i].Default = constValueOf(p.Default)
		}
	}

	def := MacroDef{Name: n.Name, Params: params, Body: bodyBC}
	macroIdx := len(c.macros)
	c.macros = append(c.macros, def)
	c.emit(OP_MACRO_DEF, uint16(c.addName(n.Name)), uint16(macroIdx), 0)
	return nil
}

// constValueOf extracts a compile-time constant from a literal AST node.
func constValueOf(node ast.Node) any {
	switch n := node.(type) {
	case *ast.StringLiteral:
		return n.Value
	case *ast.IntLiteral:
		return n.Value
	case *ast.FloatLiteral:
		return n.Value
	case *ast.BoolLiteral:
		return n.Value
	case *ast.NilLiteral:
		return nil
	}
	return nil
}

// compileMacroCall compiles a macro call expression.
// withCaller=true means an extra caller body MacroVal sits below the macro on the stack.
func (c *cmp) compileMacroCall(callee ast.Node, posArgs []ast.Node, namedArgs []ast.NamedArgNode, withCaller bool) error {
	if err := c.compileExpr(callee); err != nil {
		return err
	}
	for _, arg := range posArgs {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	for _, na := range namedArgs {
		c.emitPushConst(na.Key)
		if err := c.compileExpr(na.Value); err != nil {
			return err
		}
	}
	op := OP_CALL_MACRO_VAL
	if withCaller {
		op = OP_CALL_MACRO_CALL
	}
	c.emit(op, uint16(len(posArgs)), 0, uint8(len(namedArgs)))
	return nil
}

// compileCallNode compiles {% call macro(args) %}body{% endcall %}.
func (c *cmp) compileCallNode(n *ast.CallNode) error {
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	bodyBC := subBytecode(sub)
	callerDef := MacroDef{Name: "__caller__", Params: nil, Body: bodyBC}
	callerIdx := len(c.macros)
	c.macros = append(c.macros, callerDef)

	c.emit(OP_MACRO_DEF_PUSH, uint16(callerIdx), 0, 0)
	if err := c.compileMacroCall(n.Callee, n.PosArgs, n.NamedArgs, true); err != nil {
		return err
	}
	// {% call %} is a statement — emit OP_OUTPUT to write result to output buffer
	c.emit(OP_OUTPUT, 0, 0, 0)
	return nil
}

// compileInclude compiles {% include "name" [with k=v] [isolated] %}.
func (c *cmp) compileInclude(n *ast.IncludeNode) error {
	for _, kv := range n.WithVars {
		c.emitPushConst(kv.Key)
		if err := c.compileExpr(kv.Value); err != nil {
			return err
		}
	}
	flags := uint8(0)
	if n.Isolated {
		flags = 1
	}
	c.emit(OP_INCLUDE, uint16(c.addName(n.Name)), uint16(len(n.WithVars)), flags)
	return nil
}

// compileRender compiles {% render "name" [with k=v] %}.
func (c *cmp) compileRender(n *ast.RenderNode) error {
	for _, kv := range n.WithVars {
		c.emitPushConst(kv.Key)
		if err := c.compileExpr(kv.Value); err != nil {
			return err
		}
	}
	c.emit(OP_RENDER, uint16(c.addName(n.Name)), uint16(len(n.WithVars)), 0)
	return nil
}

// compileImport compiles {% import "name" as alias %}.
func (c *cmp) compileImport(n *ast.ImportNode) error {
	c.emit(OP_IMPORT, uint16(c.addName(n.Name)), uint16(c.addName(n.Alias)), 0)
	return nil
}

// ─── Plan 6: Component compiler ───────────────────────────────────────────────

// subBytecode creates a Bytecode from a sub-compiler's output.
// This must include ALL data arrays the sub-compiler may have populated
// (Components, Blocks, etc.) so that opcodes referencing them by index work
// correctly when the sub-bytecode runs in isolation (e.g. block bodies in
// template inheritance, fill bodies in components, hoist bodies, etc.).
func subBytecode(sub *cmp) *Bytecode {
	bc := &Bytecode{
		Instrs:     sub.instrs,
		Consts:     sub.consts,
		Names:      sub.names,
		Macros:     sub.macros,
		Blocks:     sub.blocks,
		Components: sub.components,
	}
	est := 0
	for _, cn := range bc.Consts {
		if s, ok := cn.(string); ok {
			est += len(s)
		}
	}
	bc.EstimatedOutputSize = est
	return bc
}

// ─── Plan 7 compile methods ───────────────────────────────────────────────────

// compileAsset compiles {% asset "src" type="..." [attrs] [priority=N] %}.
// Stack layout pushed before OP_ASSET: src, type, k1, v1, ..., kN, vN, priority
func (c *cmp) compileAsset(n *ast.AssetNode) error {
	c.emitPushConst(n.Src)
	c.emitPushConst(n.AssetType)
	for _, attr := range n.Attrs {
		c.emitPushConst(attr.Key)
		if err := c.compileExpr(attr.Value); err != nil {
			return err
		}
	}
	c.emitPushConst(int64(n.Priority))
	c.emit(OP_ASSET, uint16(len(n.Attrs)), 0, 0)
	return nil
}

// compileMeta compiles {% meta name="key" content="val" %}.
func (c *cmp) compileMeta(n *ast.MetaNode) error {
	keyIdx := len(c.consts)
	c.consts = append(c.consts, n.Key)
	c.emitPushConst(n.Value)
	c.emit(OP_META, uint16(keyIdx), 0, 0)
	return nil
}

// compileHoist compiles {% hoist target="name" %}body{% endhoist %}.
func (c *cmp) compileHoist(n *ast.HoistNode) error {
	sub := &cmp{nameIdx: make(map[string]int)}
	if err := sub.compileBody(n.Body); err != nil {
		return err
	}
	sub.emit(OP_HALT, 0, 0, 0)
	blockBC := subBytecode(sub)
	c.blocks = append(c.blocks, BlockDef{Name: "__hoist__:" + n.Target, Body: blockBC})
	blockIdx := len(c.blocks) - 1
	targetIdx := len(c.consts)
	c.consts = append(c.consts, n.Target)
	c.emit(OP_HOIST, uint16(targetIdx), uint16(blockIdx), 0)
	return nil
}

// compileComponent compiles {% component "name" k=v %}...{% endcomponent %}.
func (c *cmp) compileComponent(n *ast.ComponentNode) error {
	// Compile named fill bodies; index 0 is always the default fill (Name="") if non-empty.
	// If DefaultFill is empty, we don't add a default fill entry — OP_SLOT will fall through
	// to the slot's own default content.
	var fills []FillDef
	if len(n.DefaultFill) > 0 {
		defaultSub := &cmp{nameIdx: make(map[string]int)}
		if err := defaultSub.compileBody(n.DefaultFill); err != nil {
			return err
		}
		defaultSub.emit(OP_HALT, 0, 0, 0)
		fills = append(fills, FillDef{Name: "", Body: subBytecode(defaultSub)})
	}

	for _, fill := range n.Fills {
		sub := &cmp{nameIdx: make(map[string]int)}
		if err := sub.compileBody(fill.Body); err != nil {
			return err
		}
		sub.emit(OP_HALT, 0, 0, 0)
		fills = append(fills, FillDef{Name: fill.Name, Body: subBytecode(sub)})
	}

	def := ComponentDef{Name: n.Name, Fills: fills}
	compIdx := len(c.components)
	c.components = append(c.components, def)

	// Push prop key-value pairs onto stack (key first, then value)
	for _, prop := range n.Props {
		c.emitPushConst(prop.Key)
		if err := c.compileExpr(prop.Value); err != nil {
			return err
		}
	}

	c.emit(OP_COMPONENT, uint16(compIdx), uint16(len(n.Props)), 0)
	return nil
}
