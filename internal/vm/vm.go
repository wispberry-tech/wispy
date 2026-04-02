// internal/vm/vm.go
package vm

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"sync"

	"grove/internal/compiler"
	"grove/internal/scope"
)

// constCache maps compiled bytecode pointers to their pre-compiled Value slices.
// Since bytecode is immutable after compilation, we convert the untyped constant
// pool to []Value once and reuse it on every execution.
var constCache sync.Map // map[*compiler.Bytecode][]Value

// precompileConsts returns a []Value slice for the bytecode's constant pool,
// caching it so subsequent executions skip the type-switch in fromConst.
func precompileConsts(bc *compiler.Bytecode) []Value {
	if cached, ok := constCache.Load(bc); ok {
		return cached.([]Value)
	}
	vals := make([]Value, len(bc.Consts))
	for i, c := range bc.Consts {
		vals[i] = fromConst(c)
	}
	constCache.Store(bc, vals)
	return vals
}

// renderCtx accumulates page-level data (assets, meta, hoisted HTML, warnings)
// across an entire render pass including all sub-renders (components, includes, extends).
type renderCtx struct {
	assets      []assetEntry
	seenSrc     map[string]bool
	meta        map[string]string
	hoisted     map[string][]string
	warnings    []string
	maxLoopIter int // 0 = unlimited
	loopIter    int // running counter across all loops in this render pass
}

type assetEntry struct {
	src       string
	assetType string
	attrs     map[string]string
	priority  int
}

// ExecuteResult is the enriched output of a VM execution pass.
type ExecuteResult struct {
	Body string
	RC   *renderCtx
}

// ExportedAsset is the public view of an assetEntry for use by pkg/wispy.
type ExportedAsset struct {
	Src      string
	Type     string
	Attrs    map[string]string
	Priority int
}

func (rc *renderCtx) ExportAssets() []ExportedAsset {
	result := make([]ExportedAsset, len(rc.assets))
	for i, a := range rc.assets {
		result[i] = ExportedAsset{Src: a.src, Type: a.assetType, Attrs: a.attrs, Priority: a.priority}
	}
	return result
}

func (rc *renderCtx) ExportMeta() map[string]string {
	return rc.meta
}

func (rc *renderCtx) ExportHoisted() map[string][]string {
	return rc.hoisted
}

func (rc *renderCtx) ExportWarnings() []string {
	return rc.warnings
}

// loopState holds per-loop iterator state.
type loopState struct {
	items []Value  // iteration items (list elements or map values in key order)
	keys  []string // sorted map keys (nil for list loops)
	idx   int      // current index (0-based)
	isMap bool     // true when iterating a map
}

// captureFrame holds output redirection state for {% capture %}.
type captureFrame struct {
	buf    strings.Builder
	varIdx int // name index for the capture variable
}

// blockChainFrame tracks the current block execution context for super().
type blockChainFrame struct {
	name   string
	depth  int                  // current execution depth within the chain (0 = deepest child)
	bodies []*compiler.Bytecode // full super-chain for this block
}

// componentFrame holds state for an active component call.
type componentFrame struct {
	fills       []compiler.FillDef // fill bodies indexed by search
	callerScope *scope.Scope       // caller's scope — used for fill rendering
	passedProps map[string]any     // props passed at call site
}

// VM is a stack-based bytecode executor. Instances are pooled; do not hold references.
type VM struct {
	stack      [256]Value
	sp         int
	eng        EngineIface
	sc         *scope.Scope
	out        strings.Builder
	loops      [32]loopState
	loopVars   [32]loopVarData
	ldepth     int // current loop depth (0 = not in loop)
	captures   [8]captureFrame
	cdepth     int                             // current capture depth
	blockSlots map[string][]*compiler.Bytecode // per-render block override table
	blockChain []blockChainFrame               // current block execution context for super()
	compStack  [16]componentFrame
	csdepth    int        // current component stack depth
	rc         *renderCtx // page-level render context (assets, meta, hoisted)
}

var vmPool = sync.Pool{
	New: func() any {
		return &VM{}
	},
}

// currentWriter returns a pointer to the active output builder.
func (v *VM) currentWriter() *strings.Builder {
	if v.cdepth > 0 {
		return &v.captures[v.cdepth-1].buf
	}
	return &v.out
}

// Execute runs bc with data as the render context and returns an ExecuteResult.
func Execute(ctx context.Context, bc *compiler.Bytecode, data map[string]any, eng EngineIface) (ExecuteResult, error) {
	v := vmPool.Get().(*VM)
	rc := &renderCtx{
		seenSrc:     make(map[string]bool),
		meta:        make(map[string]string),
		hoisted:     make(map[string][]string),
		maxLoopIter: eng.MaxLoopIter(),
	}
	defer func() {
		v.out.Reset()
		v.sp = 0
		v.sc = nil
		v.eng = nil
		v.ldepth = 0
		v.cdepth = 0
		for i := range v.captures {
			v.captures[i].buf.Reset()
		}
		v.blockSlots = nil
		if v.blockChain != nil {
			v.blockChain = v.blockChain[:0]
		}
		v.csdepth = 0
		v.rc = nil
		vmPool.Put(v)
	}()
	v.eng = eng
	v.rc = rc

	globalSc := scope.New(nil)
	for k, val := range eng.GlobalData() {
		globalSc.Set(k, FromAny(val))
	}
	renderSc := scope.New(globalSc)
	for k, val := range data {
		renderSc.Set(k, FromAny(val))
	}
	v.sc = scope.New(renderSc)

	// If this template extends another, build initial block slot table from child's Blocks
	if bc.Extends != "" {
		v.blockSlots = make(map[string][]*compiler.Bytecode)
		for i := range bc.Blocks {
			b := &bc.Blocks[i]
			v.blockSlots[b.Name] = []*compiler.Bytecode{b.Body}
		}
	}

	body, err := v.run(ctx, bc)
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{Body: body, RC: rc}, nil
}

func (v *VM) run(ctx context.Context, bc *compiler.Bytecode) (string, error) {
	ip := 0
	instrs := bc.Instrs
	consts := precompileConsts(bc)
	ps := profileInit()
	for ip < len(instrs) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		instr := instrs[ip]
		ip++
		profileRecord(&ps, instr.Op)

		switch instr.Op {
		case compiler.OP_HALT:
			profileFlush(&ps)
			return v.out.String(), nil

		case compiler.OP_PUSH_NIL:
			v.push(Nil)

		case compiler.OP_PUSH_CONST:
			v.push(consts[instr.A])

		case compiler.OP_LOAD:
			name := bc.Names[instr.A]
			val, found := v.sc.Get(name)
			if !found {
				if v.eng.StrictVariables() {
					return "", &runtimeErr{msg: fmt.Sprintf("undefined variable %q", name)}
				}
				v.push(Nil)
			} else {
				v.push(val.(Value))
			}

		case compiler.OP_GET_ATTR:
			obj := v.pop()
			name := bc.Names[instr.A]
			result, err := GetAttr(obj, name, v.eng.StrictVariables())
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		case compiler.OP_GET_INDEX:
			key := v.pop()
			obj := v.pop()
			result, err := GetIndex(obj, key)
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		case compiler.OP_OUTPUT:
			val := v.pop()
			w := v.currentWriter()
			if val.typ == TypeSafeHTML {
				w.WriteString(val.sval)
			} else if val.typ != TypeNil {
				w.WriteString(html.EscapeString(val.String()))
			}

		case compiler.OP_OUTPUT_RAW:
			val := v.pop()
			v.currentWriter().WriteString(val.String())

		case compiler.OP_ADD:
			b, a := v.pop(), v.pop()
			v.push(arithAdd(a, b))

		case compiler.OP_SUB:
			b, a := v.pop(), v.pop()
			v.push(arithSub(a, b))

		case compiler.OP_MUL:
			b, a := v.pop(), v.pop()
			v.push(arithMul(a, b))

		case compiler.OP_DIV:
			b, a := v.pop(), v.pop()
			result, err := arithDiv(a, b)
			if err != nil {
				return "", err
			}
			v.push(result)

		case compiler.OP_MOD:
			b, a := v.pop(), v.pop()
			result, err := arithMod(a, b)
			if err != nil {
				return "", err
			}
			v.push(result)

		case compiler.OP_CONCAT:
			b, a := v.pop(), v.pop()
			v.push(StringVal(a.String() + b.String()))

		case compiler.OP_EQ:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(valEqual(a, b)))

		case compiler.OP_NEQ:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(!valEqual(a, b)))

		case compiler.OP_LT:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r < 0))

		case compiler.OP_LTE:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r <= 0))

		case compiler.OP_GT:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r > 0))

		case compiler.OP_GTE:
			b, a := v.pop(), v.pop()
			r, err := valCompare(a, b)
			if err != nil {
				return "", err
			}
			v.push(BoolVal(r >= 0))

		case compiler.OP_AND:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(Truthy(a) && Truthy(b)))

		case compiler.OP_OR:
			b, a := v.pop(), v.pop()
			v.push(BoolVal(Truthy(a) || Truthy(b)))

		case compiler.OP_NOT:
			a := v.pop()
			v.push(BoolVal(!Truthy(a)))

		case compiler.OP_NEGATE:
			a := v.pop()
			switch a.typ {
			case TypeInt:
				v.push(IntVal(-a.ival))
			case TypeFloat:
				v.push(FloatVal(-a.fval))
			default:
				v.push(IntVal(0))
			}

		case compiler.OP_JUMP:
			ip = int(instr.A)

		case compiler.OP_JUMP_FALSE:
			cond := v.pop()
			if !Truthy(cond) {
				ip = int(instr.A)
			}

		case compiler.OP_FILTER:
			name := bc.Names[instr.A]
			argc := int(instr.B)
			args := make([]Value, argc)
			for i := argc - 1; i >= 0; i-- {
				args[i] = v.pop()
			}
			val := v.pop()
			fn, ok := v.eng.LookupFilter(name)
			if !ok {
				return "", &runtimeErr{msg: fmt.Sprintf("unknown filter %q", name)}
			}
			result, err := fn(val, args)
			if err != nil {
				return "", &runtimeErr{msg: err.Error()}
			}
			v.push(result)

		// ─── Plan 2 opcodes ───────────────────────────────────────────────────

		case compiler.OP_STORE_VAR:
			val := v.pop()
			v.sc.Set(bc.Names[instr.A], val)

		case compiler.OP_PUSH_SCOPE:
			v.sc = scope.New(v.sc)

		case compiler.OP_POP_SCOPE:
			if parent := v.sc.Parent(); parent != nil {
				v.sc = parent
			}

		case compiler.OP_FOR_INIT:
			coll := v.pop()
			ls, ok := v.makeLoopState(coll)
			if !ok || len(ls.items) == 0 {
				ip = int(instr.A) // jump to fallthrough (empty block or end)
				break
			}
			if v.ldepth >= len(v.loops) {
				return "", &runtimeErr{msg: "for loop nesting too deep (max 32)"}
			}
			v.loops[v.ldepth] = ls
			v.ldepth++

		case compiler.OP_FOR_BIND_1:
			ls := &v.loops[v.ldepth-1]
			varName := bc.Names[instr.A]
			v.sc.Set(varName, ls.items[ls.idx])
			v.sc.Set("loop", v.makeLoopVal())

		case compiler.OP_FOR_BIND_KV:
			ls := &v.loops[v.ldepth-1]
			name1 := bc.Names[instr.A]
			name2 := bc.Names[instr.B]
			if ls.isMap {
				v.sc.Set(name1, StringVal(ls.keys[ls.idx]))
				v.sc.Set(name2, ls.items[ls.idx])
			} else {
				v.sc.Set(name1, IntVal(int64(ls.idx)))
				v.sc.Set(name2, ls.items[ls.idx])
			}
			v.sc.Set("loop", v.makeLoopVal())

		case compiler.OP_FOR_STEP:
			ls := &v.loops[v.ldepth-1]
			ls.idx++
			if ls.idx < len(ls.items) {
				// Sandbox: check max loop iteration limit
				if v.rc != nil && v.rc.maxLoopIter > 0 {
					v.rc.loopIter++
					if v.rc.loopIter > v.rc.maxLoopIter {
						return "", &runtimeErr{msg: fmt.Sprintf("sandbox: loop iteration limit %d exceeded", v.rc.maxLoopIter)}
					}
				}
				ip = int(instr.A) // jump back to loop top
			} else {
				v.ldepth-- // pop loop state
			}

		case compiler.OP_CAPTURE_START:
			if v.cdepth >= len(v.captures) {
				return "", &runtimeErr{msg: "capture nesting too deep (max 8)"}
			}
			v.captures[v.cdepth].buf.Reset()
			v.captures[v.cdepth].varIdx = int(instr.A)
			v.cdepth++

		case compiler.OP_CAPTURE_END:
			v.cdepth--
			content := v.captures[v.cdepth].buf.String()
			varName := bc.Names[v.captures[v.cdepth].varIdx]
			v.sc.Set(varName, StringVal(content))

		case compiler.OP_CALL_RANGE:
			argc := int(instr.A)
			args := make([]int64, argc)
			for i := argc - 1; i >= 0; i-- {
				n, _ := v.pop().ToInt64()
				args[i] = n
			}
			v.push(buildRange(args))

		// ─── Plan 4 opcodes ───────────────────────────────────────────────────

		case compiler.OP_MACRO_DEF:
			def := &bc.Macros[instr.B]
			v.sc.Set(bc.Names[instr.A], MacroVal(def))

		case compiler.OP_MACRO_DEF_PUSH:
			def := &bc.Macros[instr.A]
			v.push(MacroVal(def))

		case compiler.OP_CALL_MACRO_VAL, compiler.OP_CALL_MACRO_CALL:
			posArgCount := int(instr.A)
			namedArgCount := int(instr.Flags)

			// Pop named args (key, value pairs) in reverse order
			namedArgs := make(map[string]Value, namedArgCount)
			for i := namedArgCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				namedArgs[key.String()] = val
			}

			// Pop positional args in reverse order
			posArgs := make([]Value, posArgCount)
			for i := posArgCount - 1; i >= 0; i-- {
				posArgs[i] = v.pop()
			}

			// Pop the macro value
			macroVal := v.pop()
			def, ok := macroVal.AsMacroDef()
			if !ok {
				return "", &runtimeErr{msg: "cannot call non-macro value"}
			}

			// Pop caller body (for OP_CALL_MACRO_CALL)
			var callerDef *compiler.MacroDef
			if instr.Op == compiler.OP_CALL_MACRO_CALL {
				callerVal := v.pop()
				callerDef, _ = callerVal.AsMacroDef()
			}

			// Build macro scope: globals only (macros are isolated)
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, FromAny(val))
			}
			macroSc := scope.New(globalSc)

			// Bind params: positional first, named override, defaults for rest
			for i, param := range def.Params {
				if i < len(posArgs) {
					macroSc.Set(param.Name, posArgs[i])
				} else if val, ok := namedArgs[param.Name]; ok {
					macroSc.Set(param.Name, val)
				} else if param.Default != nil {
					macroSc.Set(param.Name, fromConst(param.Default))
				} else {
					macroSc.Set(param.Name, Nil)
				}
			}

			// Bind __caller__ if present
			if callerDef != nil {
				macroSc.Set("__caller__", MacroVal(callerDef))
			}

			result, err := v.execMacro(ctx, def.Body, macroSc)
			if err != nil {
				return "", err
			}
			v.push(SafeHTMLVal(result))

		case compiler.OP_CALL_CALLER:
			callerRaw, found := v.sc.Get("__caller__")
			if !found {
				return "", &runtimeErr{msg: "caller() called outside of a {% call %} block"}
			}
			callerVal := callerRaw.(Value)
			callerDef, ok := callerVal.AsMacroDef()
			if !ok {
				return "", &runtimeErr{msg: "caller() called outside of a {% call %} block"}
			}
			// Caller body runs in the current scope (not isolated) — so it sees outer vars
			result, err := v.execMacro(ctx, callerDef.Body, v.sc)
			if err != nil {
				return "", err
			}
			v.push(SafeHTMLVal(result))

		case compiler.OP_INCLUDE:
			tmplName := bc.Names[instr.A]
			pairCount := int(instr.B)
			isolated := instr.Flags&1 != 0

			// Pop with-var pairs
			withVars := make(map[string]any, pairCount)
			for i := pairCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				withVars[key.String()] = val
			}

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("include %q: %v", tmplName, err)}
			}

			savedSC := v.sc
			if isolated {
				globalSc := scope.New(nil)
				for k, val := range v.eng.GlobalData() {
					globalSc.Set(k, FromAny(val))
				}
				v.sc = scope.New(globalSc)
			}
			if len(withVars) > 0 || isolated {
				v.sc = scope.New(v.sc)
				for k, val := range withVars {
					v.sc.Set(k, val.(Value))
				}
			}

			if _, err := v.run(ctx, subBC); err != nil {
				v.sc = savedSC
				return "", err
			}
			v.sc = savedSC

		case compiler.OP_RENDER:
			tmplName := bc.Names[instr.A]
			pairCount := int(instr.B)

			withVars := make(map[string]any, pairCount)
			for i := pairCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				withVars[key.String()] = val
			}

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("render %q: %v", tmplName, err)}
			}

			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, FromAny(val))
			}
			renderSc := scope.New(globalSc)
			for k, val := range withVars {
				renderSc.Set(k, val.(Value))
			}

			savedSC := v.sc
			v.sc = renderSc
			if _, err := v.run(ctx, subBC); err != nil {
				v.sc = savedSC
				return "", err
			}
			v.sc = savedSC

		case compiler.OP_IMPORT:
			tmplName := bc.Names[instr.A]
			alias := bc.Names[instr.B]

			subBC, err := v.eng.LoadTemplate(tmplName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("import %q: %v", tmplName, err)}
			}

			// Execute imported template in isolated scope to collect macro definitions
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, FromAny(val))
			}
			importSc := scope.New(globalSc)
			savedSC := v.sc
			v.sc = importSc

			// Redirect output of imported template to a throwaway capture
			if v.cdepth >= len(v.captures) {
				v.sc = savedSC
				return "", &runtimeErr{msg: "import: capture nesting too deep"}
			}
			v.captures[v.cdepth].buf.Reset()
			v.captures[v.cdepth].varIdx = -1
			v.cdepth++
			_, importErr := v.run(ctx, subBC)
			v.cdepth--
			v.sc = savedSC
			if importErr != nil {
				return "", importErr
			}

			// Collect all MacroVal entries from importSc into a map
			macroMap := make(map[string]any)
			importSc.ForEach(func(k string, val any) {
				if mv, ok := val.(Value); ok && mv.typ == TypeMacro {
					macroMap[k] = mv
				}
			})
			v.sc.Set(alias, FromAny(macroMap))

		// ─── Plan 5 opcodes ───────────────────────────────────────────────────

		case compiler.OP_EXTENDS:
			parentName := bc.Names[instr.A]
			parentBC, err := v.eng.LoadTemplate(parentName)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("extends %q: %v", parentName, err)}
			}

			// Merge parent's block defaults into blockSlots (child entries take priority — don't overwrite)
			if v.blockSlots == nil {
				v.blockSlots = make(map[string][]*compiler.Bytecode)
			}
			for i := range parentBC.Blocks {
				b := &parentBC.Blocks[i]
				// Append parent's default as the last (lowest priority) entry in the chain
				v.blockSlots[b.Name] = append(v.blockSlots[b.Name], b.Body)
			}

			// Execute the parent's main instruction stream (it will hit OP_BLOCK_RENDER for each slot)
			if _, err := v.run(ctx, parentBC); err != nil {
				return "", err
			}
			// After parent executes, we're done — return to skip remaining instructions in child
			return v.out.String(), nil

		case compiler.OP_BLOCK_RENDER:
			blockName := bc.Names[instr.A]
			defaultBlockIdx := int(instr.B)

			// Determine what bodies to execute: override chain, or just parent default
			var bodies []*compiler.Bytecode
			if v.blockSlots != nil {
				if chain, ok := v.blockSlots[blockName]; ok && len(chain) > 0 {
					bodies = chain
				}
			}
			if len(bodies) == 0 {
				// No override — use this template's default block body
				bodies = []*compiler.Bytecode{bc.Blocks[defaultBlockIdx].Body}
			}

			// Push block chain frame for super() support
			frame := blockChainFrame{name: blockName, depth: 0, bodies: bodies}
			v.blockChain = append(v.blockChain, frame)

			_, err := v.run(ctx, bodies[0])

			v.blockChain = v.blockChain[:len(v.blockChain)-1]
			if err != nil {
				return "", err
			}

		case compiler.OP_SUPER:
			if len(v.blockChain) == 0 {
				return "", &runtimeErr{msg: "super() called outside a block"}
			}
			frame := &v.blockChain[len(v.blockChain)-1]
			nextDepth := frame.depth + 1
			if nextDepth >= len(frame.bodies) {
				// No more parents — super() at the root, push empty string
				v.push(SafeHTMLVal(""))
				break
			}
			prevDepth := frame.depth
			frame.depth = nextDepth
			// Capture output of the parent block body into a SafeHTML value
			superResult, err := v.execBlockCapture(ctx, frame.bodies[nextDepth])
			frame.depth = prevDepth
			if err != nil {
				return "", err
			}
			v.push(SafeHTMLVal(superResult))

		// ─── Plan 6 opcodes ───────────────────────────────────────────────────

		case compiler.OP_COMPONENT:
			compDef := bc.Components[instr.A]
			propCount := int(instr.B)

			// Pop prop key-value pairs (pushed key-first, so pop in reverse)
			props := make(map[string]any, propCount)
			for i := propCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				props[key.String()] = val
			}

			// Load component template
			compBC, err := v.eng.LoadTemplate(compDef.Name)
			if err != nil {
				return "", &runtimeErr{msg: fmt.Sprintf("component %q: %v", compDef.Name, err)}
			}

			// Save caller scope; push component frame
			callerScope := v.sc
			if v.csdepth >= len(v.compStack) {
				return "", &runtimeErr{msg: "component nesting too deep (max 16)"}
			}
			v.compStack[v.csdepth] = componentFrame{
				fills:       compDef.Fills,
				callerScope: callerScope,
				passedProps: props,
			}
			v.csdepth++

			// Set up isolated component scope: globals → component scope
			globalSc := scope.New(nil)
			for k, val := range v.eng.GlobalData() {
				globalSc.Set(k, FromAny(val))
			}
			v.sc = scope.New(globalSc)

			// If no {% props %} declaration (permissive mode), bind all props now
			if compBC.Props == nil {
				for k, val := range props {
					v.sc.Set(k, val.(Value))
				}
			}

			_, err = v.run(ctx, compBC)
			v.csdepth--
			v.sc = callerScope
			if err != nil {
				return "", err
			}

		case compiler.OP_PROPS_INIT:
			if v.csdepth == 0 {
				return "", &runtimeErr{msg: "props declaration outside component context"}
			}
			frame := &v.compStack[v.csdepth-1]
			passed := frame.passedProps
			declared := bc.Props

			// Build declared set for unknown-prop check
			declaredSet := make(map[string]bool, len(declared))
			for _, p := range declared {
				declaredSet[p.Name] = true
			}
			// Check for unknown props
			for k := range passed {
				if !declaredSet[k] {
					return "", &runtimeErr{msg: fmt.Sprintf("component: unknown prop %q", k)}
				}
			}
			// Bind props: passed value or default; error if required and missing
			for _, p := range declared {
				if val, ok := passed[p.Name]; ok {
					v.sc.Set(p.Name, val.(Value))
				} else if p.Default != nil {
					v.sc.Set(p.Name, fromConst(p.Default))
				} else {
					return "", &runtimeErr{msg: fmt.Sprintf("component: missing required prop %q", p.Name)}
				}
			}

		case compiler.OP_SLOT:
			slotName := bc.Names[instr.A]
			defaultBlockIdx := int(instr.B)

			if v.csdepth == 0 {
				// {% slot %} used outside a component — render default content only
				if defaultBlockIdx != 0xFFFF {
					if _, err := v.run(ctx, bc.Blocks[defaultBlockIdx].Body); err != nil {
						return "", err
					}
				}
				break
			}

			frame := &v.compStack[v.csdepth-1]

			// Find matching fill by name
			var fillBody *compiler.Bytecode
			for i := range frame.fills {
				if frame.fills[i].Name == slotName {
					fillBody = frame.fills[i].Body
					break
				}
			}

			if fillBody != nil {
				// Render fill in caller scope (lazy render)
				savedSC := v.sc
				v.sc = scope.New(frame.callerScope)
				_, err := v.run(ctx, fillBody)
				v.sc = savedSC
				if err != nil {
					return "", err
				}
			} else if defaultBlockIdx != 0xFFFF {
				// No fill provided — render slot default content
				if _, err := v.run(ctx, bc.Blocks[defaultBlockIdx].Body); err != nil {
					return "", err
				}
			}
			// else: empty slot with no fill and no default — render nothing

		// ─── Plan 7 opcodes ───────────────────────────────────────────────────

		case compiler.OP_ASSET:
			attrPairCount := int(instr.A)
			// Pop in reverse order: priority last-pushed, then attr pairs, then type, then src
			priorityVal := v.pop()
			priority := 0
			if n, ok := priorityVal.ToInt64(); ok {
				priority = int(n)
			}
			attrs := make(map[string]string, attrPairCount)
			for i := attrPairCount - 1; i >= 0; i-- {
				val := v.pop()
				key := v.pop()
				attrs[key.String()] = val.String()
			}
			assetType := v.pop().String()
			src := v.pop().String()

			if !v.rc.seenSrc[src] {
				v.rc.seenSrc[src] = true
				v.rc.assets = append(v.rc.assets, assetEntry{
					src:       src,
					assetType: assetType,
					attrs:     attrs,
					priority:  priority,
				})
			}

		case compiler.OP_META:
			content := v.pop().String()
			key := bc.Consts[instr.A].(string)
			if _, exists := v.rc.meta[key]; exists {
				v.rc.warnings = append(v.rc.warnings,
					fmt.Sprintf("meta key %q overwritten", key))
			}
			v.rc.meta[key] = content

		case compiler.OP_HOIST:
			target := bc.Consts[instr.A].(string)
			hoistBC := bc.Blocks[instr.B].Body
			captured, err := v.execBlockCapture(ctx, hoistBC)
			if err != nil {
				return "", err
			}
			v.rc.hoisted[target] = append(v.rc.hoisted[target], captured)

		default:
			return "", fmt.Errorf("vm: unknown opcode %d at ip=%d", instr.Op, ip-1)
		}
	}
	return v.out.String(), nil
}

// execBlockCapture runs a block body bytecode in the current scope, capturing output to a string.
// Used by OP_SUPER to capture the parent block's rendered content as a value.
func (v *VM) execBlockCapture(ctx context.Context, bc *compiler.Bytecode) (string, error) {
	if v.cdepth >= len(v.captures) {
		return "", &runtimeErr{msg: "block super() nesting too deep (max 8)"}
	}
	v.captures[v.cdepth].buf.Reset()
	v.captures[v.cdepth].varIdx = -1
	v.cdepth++

	_, err := v.run(ctx, bc)

	v.cdepth--
	if err != nil {
		return "", err
	}
	return v.captures[v.cdepth].buf.String(), nil
}

// execMacro runs bc in the given scope, capturing output to a string.
func (v *VM) execMacro(ctx context.Context, bc *compiler.Bytecode, sc *scope.Scope) (string, error) {
	if v.cdepth >= len(v.captures) {
		return "", &runtimeErr{msg: "macro call nesting too deep (max 8)"}
	}
	v.captures[v.cdepth].buf.Reset()
	v.captures[v.cdepth].varIdx = -1
	v.cdepth++

	savedSC := v.sc
	v.sc = sc

	_, err := v.run(ctx, bc)

	v.sc = savedSC
	v.cdepth--
	if err != nil {
		return "", err
	}
	return v.captures[v.cdepth].buf.String(), nil
}

// makeLoopState converts a Value into a loopState.
func (v *VM) makeLoopState(coll Value) (loopState, bool) {
	switch coll.typ {
	case TypeList:
		lst, _ := coll.oval.([]Value)
		return loopState{items: lst}, true
	case TypeMap:
		m, _ := coll.oval.(map[string]any)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := make([]Value, len(keys))
		for i, k := range keys {
			vals[i] = FromAny(m[k])
		}
		return loopState{items: vals, keys: keys, isMap: true}, true
	case TypeNil:
		return loopState{}, false
	}
	return loopState{}, false
}

// makeLoopVal constructs the `loop` magic variable without allocation.
func (v *VM) makeLoopVal() Value {
	ls := &v.loops[v.ldepth-1]
	ld := &v.loopVars[v.ldepth-1]
	ld.index = ls.idx
	ld.length = len(ls.items)
	ld.depth = v.ldepth
	if v.ldepth > 1 {
		ld.parent = &v.loopVars[v.ldepth-2]
	} else {
		ld.parent = nil
	}
	return loopVarVal(ld)
}

// buildRange implements range(stop), range(start, stop), range(start, stop, step).
func buildRange(args []int64) Value {
	var start, stop, step int64
	switch len(args) {
	case 1:
		start, stop, step = 0, args[0], 1
	case 2:
		start, stop, step = args[0], args[1], 1
	case 3:
		start, stop, step = args[0], args[1], args[2]
	default:
		return ListVal(nil)
	}
	if step == 0 {
		return ListVal(nil)
	}
	var items []Value
	if step > 0 {
		for i := start; i < stop; i += step {
			items = append(items, IntVal(i))
		}
	} else {
		for i := start; i > stop; i += step {
			items = append(items, IntVal(i))
		}
	}
	return ListVal(items)
}

// ─── Stack helpers ────────────────────────────────────────────────────────────

func (v *VM) push(val Value) {
	if v.sp >= len(v.stack) {
		panic("vm: stack overflow")
	}
	v.stack[v.sp] = val
	v.sp++
}

func (v *VM) pop() Value {
	v.sp--
	return v.stack[v.sp]
}

// ─── Arithmetic ───────────────────────────────────────────────────────────────

func fromConst(c any) Value {
	switch x := c.(type) {
	case bool:
		return BoolVal(x)
	case int64:
		return IntVal(x)
	case float64:
		return FloatVal(x)
	case string:
		return StringVal(x)
	}
	return Nil
}

func arithAdd(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af + bf)
	}
	ai, aok := a.ToInt64()
	bi, bok := b.ToInt64()
	if aok && bok {
		return IntVal(ai + bi)
	}
	return StringVal(a.String() + b.String())
}

func arithSub(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af - bf)
	}
	ai, _ := a.ToInt64()
	bi, _ := b.ToInt64()
	return IntVal(ai - bi)
}

func arithMul(a, b Value) Value {
	if a.typ == TypeFloat || b.typ == TypeFloat {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		return FloatVal(af * bf)
	}
	ai, _ := a.ToInt64()
	bi, _ := b.ToInt64()
	return IntVal(ai * bi)
}

func arithDiv(a, b Value) (Value, error) {
	af, _ := a.ToFloat64()
	bf, _ := b.ToFloat64()
	if bf == 0 {
		return Nil, &runtimeErr{msg: "division by zero"}
	}
	result := af / bf
	if a.typ == TypeInt && b.typ == TypeInt && result == float64(int64(result)) {
		return IntVal(int64(result)), nil
	}
	return FloatVal(result), nil
}

func arithMod(a, b Value) (Value, error) {
	bi, bok := b.ToInt64()
	if !bok || bi == 0 {
		bf, _ := b.ToFloat64()
		if bf == 0 {
			return Nil, &runtimeErr{msg: "modulo by zero"}
		}
	}
	ai, _ := a.ToInt64()
	return IntVal(ai % bi), nil
}

// ─── Comparison ───────────────────────────────────────────────────────────────

func valEqual(a, b Value) bool {
	if a.typ != b.typ {
		if (a.typ == TypeInt || a.typ == TypeFloat) && (b.typ == TypeInt || b.typ == TypeFloat) {
			af, _ := a.ToFloat64()
			bf, _ := b.ToFloat64()
			return af == bf
		}
		return false
	}
	switch a.typ {
	case TypeNil:
		return true
	case TypeBool:
		return a.ival == b.ival
	case TypeInt:
		return a.ival == b.ival
	case TypeFloat:
		return a.fval == b.fval
	case TypeString, TypeSafeHTML:
		return a.sval == b.sval
	}
	return false
}

func valCompare(a, b Value) (int, error) {
	if (a.typ == TypeInt || a.typ == TypeFloat) && (b.typ == TypeInt || b.typ == TypeFloat) {
		af, _ := a.ToFloat64()
		bf, _ := b.ToFloat64()
		if af < bf {
			return -1, nil
		} else if af > bf {
			return 1, nil
		}
		return 0, nil
	}
	if a.typ == TypeString && b.typ == TypeString {
		if a.sval < b.sval {
			return -1, nil
		} else if a.sval > b.sval {
			return 1, nil
		}
		return 0, nil
	}
	return 0, &runtimeErr{msg: fmt.Sprintf("cannot compare %v and %v", a.typ, b.typ)}
}

// ─── Runtime error ────────────────────────────────────────────────────────────

type runtimeErr struct {
	msg string
}

func (e *runtimeErr) Error() string { return e.msg }
