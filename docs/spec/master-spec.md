# Grove Template Engine — Master Technical Specification

**Module:** `github.com/wispberry-tech/grove`
**Language:** Go 1.24
**Status:** Living document — updated as the engine evolves
**External dependencies:** `testify` (test-only)

---

## Table of Contents

1. [Design Philosophy](#1-design-philosophy)
2. [Architecture Overview](#2-architecture-overview)
3. [Render Pipeline](#3-render-pipeline)
4. [Lexer](#4-lexer)
5. [Parser](#5-parser)
6. [AST](#6-ast)
7. [Compiler & Bytecode](#7-compiler--bytecode)
8. [Virtual Machine](#8-virtual-machine)
9. [Value System](#9-value-system)
10. [Scope & Variable Resolution](#10-scope--variable-resolution)
11. [Type Coercion](#11-type-coercion)
12. [Template Syntax Reference](#12-template-syntax-reference)
13. [Filters](#13-filters)
14. [Template Composition](#14-template-composition)
15. [Template Inheritance](#15-template-inheritance)
16. [Components & Slots](#16-components--slots)
17. [Macros](#17-macros)
18. [Web Primitives](#18-web-primitives)
19. [Template Stores](#19-template-stores)
20. [Caching](#20-caching)
21. [Security & Sandboxing](#21-security--sandboxing)
22. [Public API](#22-public-api)
23. [Error Model](#23-error-model)
24. [Concurrency Model](#24-concurrency-model)
25. [Performance Characteristics](#25-performance-characteristics)

---

## 1. Design Philosophy

### Goals

1. **Balanced performance** — bytecode-compiled VM targeting ~1–3M renders/sec on realistic templates, without requiring a build step or sacrificing hot-reload capability.
2. **Rich expression language** — Jinja2-level expressiveness (ternary `? :`, chained filters with arguments, arithmetic, map/list literals) combined with Go idioms.
3. **Web-application primitives** — `RenderResult` with asset deduplication, metadata hoisting, and content hoisting are first-class, not afterthoughts.
4. **Explicit security** — auto-escaping on by default; `safe` is the only escape hatch. Sandbox mode enforced at the VM level.
5. **Opt-in type exposure** — Go types control what fields templates can access via the `Resolvable` interface. No reflection-based struct walking.

### Design Principles

- **Parse once, render many** — bytecode is immutable and shared across goroutines; only VM frames are per-render.
- **Zero surprise defaults** — auto-escape on, undefined variables return empty string (configurable to error via strict mode).
- **Explicit over implicit** — trust is explicit (`| safe`), scope is explicit (`render` vs `include`), type exposure is explicit (`GroveResolve`).
- **Composition over inheritance** — components with slots are preferred over deep inheritance chains.
- **Render side-effects are first-class** — assets, metadata, and hoisted content flow through `RenderResult`, not ad-hoc mechanisms.
- **Zero external dependencies** — one `go get`, no surprises.

---

## 2. Architecture Overview

### Package Layout

| Package | Role |
|---------|------|
| `pkg/grove/` | Public API: `Engine`, `RenderResult`, `Data`, `Resolvable`, options, store aliases, filter registration |
| `internal/lexer/` | State-machine tokenizer producing `[]Token` |
| `internal/parser/` | Recursive-descent parser: `[]Token` → `*ast.Program` |
| `internal/ast/` | AST node type definitions |
| `internal/compiler/` | AST → `*Bytecode` with opcode emission |
| `internal/vm/` | Stack-based bytecode interpreter, value types, filter dispatch |
| `internal/scope/` | Variable lookup chain with scope-stack shadowing |
| `internal/filters/` | Built-in filter implementations (string, collection, numeric, HTML) |
| `internal/store/` | Template storage backends: `MemoryStore`, `FileSystemStore` |
| `internal/groverrors/` | Shared error types: `ParseError`, `RuntimeError` |

### Data Flow

```
Template Source (string or []byte)
        │
        ▼
  ┌──────────────┐
  │    Lexer     │  State-machine tokenizer
  └──────────────┘
        │ []Token
        ▼
  ┌──────────────┐
  │    Parser    │  Recursive descent → AST
  └──────────────┘
        │ *ast.Program
        ▼
  ┌──────────────┐
  │   Compiler   │  Walks AST → []Instruction + constant/name pools
  └──────────────┘
        │ *Bytecode
        ▼
  ┌──────────────┐
  │  LRU Cache   │  keyed by template name (512 entries default)
  └──────────────┘
        │
        ▼
  ┌──────────────┐     ┌────────────────┐
  │  VM (pooled) │ ←── │  render data   │
  └──────────────┘     └────────────────┘
        │
        ▼
  RenderResult { Body, Assets, Meta, Hoisted, Warnings }
```

---

## 3. Render Pipeline

### Inline Rendering (`RenderTemplate`)

1. **Lex** — `lexer.Tokenize(src)` → `[]Token`
2. **Parse** — `parser.Parse(tokens, inline=true, allowedTags)` → `*ast.Program`
   - Inline mode rejects `extends`, `import`, `include`, `render`, `component` tags
3. **Compile** — `compiler.Compile(prog)` → `*Bytecode`
   - Sandbox `AllowedFilters` check if configured
4. **Execute** — `vm.Execute(ctx, bc, data, engine)` → `ExecuteResult`
5. **Wrap** — `resultFromExecute(er)` → `RenderResult`

### Named Rendering (`Render`)

1. **Cache check** — LRU lookup by template name
2. On miss: **Load** from store → **Lex** → **Parse** (inline=false) → **Compile** → **Cache store**
3. **Execute** → **Wrap** → `RenderResult`

---

## 4. Lexer

**File:** `internal/lexer/lexer.go`, `internal/lexer/token.go`

### Token Types

```
TK_EOF             End of input
TK_TEXT             Raw text between delimiters
TK_TAG_START        {% or {%-
TK_TAG_END          %} or -%}
TK_STRING           "..." or '...'
TK_INT              123
TK_FLOAT            1.23
TK_TRUE             true
TK_FALSE            false
TK_NIL              nil / null
TK_IDENT            foo, bar_baz, _priv
TK_DOT              .
TK_LBRACKET         [
TK_RBRACKET         ]
TK_LPAREN           (
TK_RPAREN           )
TK_COMMA            ,
TK_PIPE             |
TK_ASSIGN           =
TK_PLUS             +
TK_MINUS            -
TK_STAR             *
TK_SLASH            /
TK_PERCENT          %
TK_TILDE            ~ (string concatenation)
TK_EQ               ==
TK_NEQ              !=
TK_LT               <
TK_LTE              <=
TK_GT               >
TK_GTE              >=
TK_AND              and
TK_OR               or
TK_NOT              not
TK_QUESTION         ? (ternary)
TK_COLON            : (ternary / map literal / branch separator)
TK_LBRACE           { (map literal)
TK_RBRACE           } (map literal)
TK_BLOCK_OPEN       #keyword (opens block: {% #if %}, {% #each %}, etc.)
TK_BLOCK_BRANCH     :keyword (branch separator: {% :else %}, {% :empty %}, etc.)
TK_BLOCK_CLOSE      /keyword (closes block: {% /if %}, {% /each %}, etc.)
TK_ELEMENT_OPEN     <Name (PascalCase element start)
TK_ELEMENT_CLOSE    </Name> (PascalCase element close)
TK_ELEMENT_END      >
TK_SELF_CLOSE       /> (self-closing element)
```

### Token Structure

```go
type Token struct {
    Kind       TokenKind
    Value      string  // raw text (identifier name, string content, number digits)
    Line       int     // 1-based line number
    Col        int     // 1-based column number
    StripLeft  bool    // {%- or {%-: strip whitespace to the left
    StripRight bool    // -%} or -%}: strip whitespace to the right
}
```

### Whitespace Control

- `{%- expr -%}` strips whitespace before and after a tag
- The `-` must be adjacent to the delimiter (`{%-` not `{% -`)

---

## 5. Parser

**File:** `internal/parser/parser.go`

### Grammar

Recursive-descent parser. Key entry points:

- `Parse(tokens, inline, allowedTags)` → `*ast.Program`
- `parseTag()` — dispatches on tag name to specific parsers
- `parseExpression()` — entry point for expression parsing

### Operator Precedence (highest to lowest)

| Level | Operators | Description |
|-------|-----------|-------------|
| 1 | `.`, `[]`, `()` | Attribute access, index, function/macro call |
| 2 | `\|` | Filter pipe |
| 3 | `not`, `-` (unary) | Negation |
| 4 | `*`, `/`, `%` | Multiplicative |
| 5 | `+`, `-`, `~` | Additive, string concatenation |
| 6 | `<`, `<=`, `>`, `>=`, `==`, `!=` | Comparison |
| 7 | `and` | Logical AND |
| 8 | `or` | Logical OR |
| 9 | `? :` | Ternary conditional |

### Tag Dispatch

The parser recognizes the following tag names and sigil-based blocks:
- **Block openers** (sigil `#`): `#if`, `#each`, `#let`, `#capture`, `#fill`, `#hoist`, `#verbatim`
- **Branch separators** (sigil `:`): `:else`, `:else if`, `:empty`
- **Block closers** (sigil `/`): `/if`, `/each`, `/let`, `/capture`, `/fill`, `/hoist`, `/verbatim`
- **Standalone tags**: `set`, `import`, `asset`, `meta`, `slot`, `include`, `render`

Sandbox `AllowedTags` whitelist is checked at parse time; banned tags produce a `ParseError`.

Sandbox `AllowedTags` whitelist is checked at parse time; banned tags produce a `ParseError`.

---

## 6. AST

**File:** `internal/ast/node.go`

All nodes implement the `Node` interface (`wispyNode()` marker method).

### Expression Nodes

| Node | Fields | Description |
|------|--------|-------------|
| `NilLiteral` | `Line` | `nil`/`null` literal |
| `BoolLiteral` | `Value bool`, `Line` | `true`/`false` |
| `IntLiteral` | `Value int64`, `Line` | Integer literal |
| `FloatLiteral` | `Value float64`, `Line` | Float literal |
| `StringLiteral` | `Value string`, `Line` | Quoted string |
| `Identifier` | `Name`, `Line` | Variable reference |
| `AttributeAccess` | `Object Node`, `Key string`, `Line` | `obj.key` |
| `IndexAccess` | `Object Node`, `Key Node`, `Line` | `obj[key]` |
| `BinaryExpr` | `Op string`, `Left`, `Right Node`, `Line` | `left op right` |
| `UnaryExpr` | `Op string`, `Operand Node`, `Line` | `not x`, `-x` |
| `TernaryExpr` | `Condition`, `Consequence`, `Alternative Node`, `Line` | `cond ? a : b` |
| `FilterExpr` | `Value Node`, `Filter string`, `Args []Node`, `Line` | `val \| filter(args)` |
| `ListLiteral` | `Elements []Node`, `Line` | `[a, b, c]` |
| `MapLiteral` | `Entries []MapEntry`, `Line` | `{ key: val }` |
| `FuncCallNode` | `Name string`, `Args []Node`, `Line` | `range(1, 10)` |
| `MacroCallExpr` | `Callee Node`, `PosArgs`, `NamedArgs`, `Line` | `macro(args)` |
| `NamedArgNode` | `Key string`, `Value Node`, `Line` | `key=value` in calls |

### Statement Nodes

| Node | Fields | Description |
|------|--------|-------------|
| `Program` | `Body []Node` | Root node |
| `TextNode` | `Value string`, `Line` | Raw text content |
| `OutputNode` | `Expr Node`, `StripLeft`, `StripRight bool`, `Line` | `{% expr %}` |
| `RawNode` | `Value string`, `Line` | `{% #verbatim %}...{% /verbatim %}` |
| `IfNode` | `Condition`, `Body`, `Elifs []ElifClause`, `Else []Node`, `Line` | `{% #if %}...{% :else if %}...{% :else %}...{% /if %}` |
| `ForNode` | `Var1`, `Var2 string`, `Iterable Node`, `Body`, `Empty []Node`, `Line` | `{% #each items as x %}...{% :empty %}...{% /each %}` |
| `SetNode` | `Name string`, `Expr Node`, `Line` | `{% set x = expr %}` |
| `LetNode` | `Body []LetStmt`, `Line` | `{% #let %}...{% /let %}` |
| `CaptureNode` | `Name string`, `Body []Node`, `Line` | `{% #capture x %}...{% /capture %}` |
| `IncludeNode` | `Name string`, `WithVars []NamedArgNode`, `Line` | `{% include %}` |
| `RenderNode` | `Name string`, `WithVars []NamedArgNode`, `Line` | `{% render %}` (always isolated) |
| `ImportNode` | `Name string`, `Alias string`, `Line` | `{% import Name from "path" %}` |
| `AssetNode` | `Src`, `AssetType string`, `Attrs`, `Priority int`, `Line` | `{% asset %}` |
| `MetaNode` | `Key`, `Value string`, `Line` | `{% meta %}` |
| `HoistNode` | `Target string`, `Body []Node`, `Line` | `{% #hoist %}...{% /hoist %}` |
| `SlotNode` | `Name string`, `Default []Node`, `Line` | `{% slot %}` or `{% #slot "name" %}...{% /slot %}` |

**Legacy nodes** (kept for compatibility but not part of current spec):
- `MacroNode`, `CallNode` — Macros removed from element-based parser
- `ExtendsNode`, `BlockNode`, `PropsNode`, `ComponentNode` — Template inheritance/components migrated to `<PascalCase>` elements

### Let Block Sub-Nodes

| Node | Fields | Description |
|------|--------|-------------|
| `LetAssignment` | `Name string`, `Expr Node` | `name = expression` inside let |
| `LetIf` | `Condition Node`, `Body`, `Elifs`, `Else []LetStmt` | `if/elif/else/end` inside let |

---

## 7. Compiler & Bytecode

**Files:** `internal/compiler/compiler.go`, `internal/compiler/bytecode.go`

### Instruction Format

Fixed-width 8-byte instruction for cache-line-friendly execution:

```go
type Instruction struct {
    A     uint16  // primary operand (const index, name index, jump target, arg count)
    B     uint16  // secondary operand
    Op    Opcode  // uint8
    Flags uint8   // modifier bits
    _     [2]byte // reserved
}
```

### Opcode Reference

#### Core Opcodes

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_HALT` | — | Stop execution |
| `OP_PUSH_CONST` | A=const_idx | Push constant from pool onto stack |
| `OP_PUSH_NIL` | — | Push nil value |
| `OP_LOAD` | A=name_idx | Scope lookup by name, push result |
| `OP_GET_ATTR` | A=name_idx | Pop object, push `obj.name` |
| `OP_GET_INDEX` | — | Pop key, pop object, push `obj[key]` |
| `OP_OUTPUT` | — | Pop value, HTML-escape, write to output (unless SafeHTML) |
| `OP_OUTPUT_RAW` | — | Pop value, write verbatim (no escaping) |
| `OP_STORE_VAR` | A=name_idx | Pop value, store in local scope |

#### Arithmetic / Comparison / Logic

| Opcode | Description |
|--------|-------------|
| `OP_ADD` | Pop b, pop a, push a+b |
| `OP_SUB` | Pop b, pop a, push a-b |
| `OP_MUL` | Pop b, pop a, push a*b |
| `OP_DIV` | Pop b, pop a, push a/b |
| `OP_MOD` | Pop b, pop a, push a%b |
| `OP_CONCAT` | Pop b, pop a, push string concatenation |
| `OP_EQ`, `OP_NEQ` | Equality / inequality |
| `OP_LT`, `OP_LTE`, `OP_GT`, `OP_GTE` | Ordered comparison |
| `OP_AND`, `OP_OR` | Logical operators |
| `OP_NOT` | Logical negation |
| `OP_NEGATE` | Unary minus |

#### Control Flow

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_JUMP` | A=target_ip | Unconditional jump |
| `OP_JUMP_FALSE` | A=target_ip | Pop value, jump if falsy |
| `OP_FOR_INIT` | A=fallthrough_ip | Pop collection, push loop state; jump to A if empty |
| `OP_FOR_BIND_1` | A=var_name_idx | Bind current item to scope; bind `loop` metadata |
| `OP_FOR_BIND_KV` | A=key_idx, B=val_idx | Bind key+value (map) or index+value (list two-var) |
| `OP_FOR_STEP` | A=loop_top_ip | Advance index; jump to A if more items; else pop loop state |
| `OP_CALL_RANGE` | A=argc | Pop argc int args, push list per range semantics |

#### Capture

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_CAPTURE_START` | A=var_name_idx | Redirect output to capture buffer |
| `OP_CAPTURE_END` | — | Flush capture buffer to scope variable; restore output |

#### Filters

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_FILTER` | A=name_idx, B=argc | Pop argc args then value, apply filter, push result |

#### Macros

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_MACRO_DEF` | A=name_idx, B=macro_idx | Store MacroDef as MacroVal in scope |
| `OP_MACRO_DEF_PUSH` | A=macro_idx | Push MacroVal onto stack (for caller body) |
| `OP_CALL_MACRO_VAL` | A=posArgCount, Flags=namedArgCount | Call a macro value; push SafeHTML result |
| `OP_CALL_MACRO_CALL` | — | Like `OP_CALL_MACRO_VAL` but also pops caller body |
| `OP_CALL_CALLER` | — | Call `__caller__` macro in current scope; push SafeHTML result |

#### Composition

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_INCLUDE` | A=name_idx, Flags=bit0 unused, B=with_pair_count | Include template with optional variables |
| `OP_RENDER` | A=name_idx, B=with_pair_count | Render template (always isolated scope) |
| `OP_IMPORT` | A=name_idx, B=alias_idx | Import macros as namespace |

#### Inheritance

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_EXTENDS` | A=name_idx | Load parent template, merge block slots, execute parent |
| `OP_BLOCK_RENDER` | A=name_idx, B=block_idx | Render a block slot (override or default) |
| `OP_SUPER` | — | Render one level up the current block's super-chain |

#### Components

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_COMPONENT` | A=comp_idx, B=prop_pair_count | Load component, validate props, render with fills |
| `OP_SLOT` | A=name_idx, B=default_block_idx | Render matching fill or default content (0xFFFF = no default) |
| `OP_PROPS_INIT` | — | Validate and bind props into scope |

#### Web Primitives

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_ASSET` | A=attr_pair_count | Collect asset into render context |
| `OP_META` | A=key_const_idx | Collect metadata key/value into render context |
| `OP_HOIST` | A=target_const_idx, B=body_block_idx | Render body, append to hoisted[target] |

#### Literals

| Opcode | Operands | Description |
|--------|----------|-------------|
| `OP_BUILD_LIST` | A=element_count | Pop N values, build list, push |
| `OP_BUILD_MAP` | A=entry_count | Pop N*2 values (key/val pairs), build ordered map, push |

### Bytecode Structure

```go
type Bytecode struct {
    Instrs              []Instruction
    Consts              []any            // constant pool: string | int64 | float64 | bool
    Names               []string         // name pool: variable, attribute, filter names
    Macros              []MacroDef       // compiled inline macros
    Blocks              []BlockDef       // compiled block bodies (defaults + overrides)
    Extends             string           // parent template name (empty if none)
    Props               []MacroParam     // from {% props %} declaration
    Components          []ComponentDef   // one per {% component %} call
    EstimatedOutputSize int              // hint for output buffer sizing
}
```

Bytecode is **immutable after compilation** and safe for concurrent use across goroutines.

### Supporting Types

```go
type MacroDef struct {
    Name   string
    Params []MacroParam
    Body   *Bytecode
}

type MacroParam struct {
    Name    string
    Default any   // nil = required; string/int64/float64/bool = default constant
}

type BlockDef struct {
    Name string
    Body *Bytecode
}

type ComponentDef struct {
    Name  string
    Fills []FillDef
}

type FillDef struct {
    Name string   // "" = default slot fill
    Body *Bytecode
}
```

---

## 8. Virtual Machine

**File:** `internal/vm/vm.go`

### VM Structure

```go
type VM struct {
    stack      [256]Value            // fixed-size value stack (no allocation for typical templates)
    sp         int                   // stack pointer
    eng        EngineIface           // callback interface to Engine
    sc         *scope.Scope          // current scope
    out        strings.Builder       // primary output buffer
    loops      [32]loopState         // pre-allocated loop state slots
    loopVars   [32]loopVarData       // pre-allocated loop metadata
    loopScopes [32]*scope.Scope      // pre-allocated scopes for loop bodies
    ldepth     int                   // current loop depth (0 = not in loop)
    captures   [8]captureFrame       // capture buffer stack
    cdepth     int                   // current capture depth
    blockSlots map[string][]*Bytecode // per-render block override table
    blockChain []blockChainFrame     // block execution context for super()
    compStack  [16]componentFrame    // component call stack
    csdepth    int                   // component stack depth
    rc         *renderCtx            // page-level render context (assets, meta, hoisted)
}
```

### Instance Pooling

VM instances are pooled via `sync.Pool`. Each render call:
1. Acquires a VM from the pool
2. Initializes scope with render data + engine globals
3. Executes bytecode
4. Resets VM state
5. Returns VM to pool

### Execution Loop

The core execution loop is a `for` loop with a `switch` on `instr.Op`. Key behaviors:

- **Output** (`OP_OUTPUT`): pops value, HTML-escapes unless `IsSafeHTML()`, writes to current output buffer (or capture buffer if in capture mode)
- **Scope lookup** (`OP_LOAD`): looks up variable by name through scope chain
- **Constant cache**: bytecode constant pools are pre-compiled to `[]Value` eagerly in `engine.compileChecked` and stored on the `Bytecode` itself, recursing into every reachable sub-bytecode (macros, blocks, component fills). Because the write happens once, single-threaded, before the bytecode is cached, subsequent concurrent reads are race-free
- **Loop iteration**: `OP_FOR_INIT` initializes loop state (counting the first body execution); `OP_FOR_STEP` advances (counting each subsequent body execution); sandbox `MaxLoopIter` is incremented and checked at both sites, so the counter equals total body executions across all nested loops
- **Block rendering**: `OP_BLOCK_RENDER` checks `blockSlots` for override chain; executes override or default
- **Super chains**: `OP_SUPER` advances one level up the block chain; `RuntimeError` if called outside a block

### Render Context

The `renderCtx` is shared across an entire render pass (including sub-renders from includes, components, extends):

```go
type renderCtx struct {
    assets      []assetEntry          // collected assets
    seenSrc     map[string]bool       // deduplication index
    meta        map[string]string     // metadata (last-write-wins)
    hoisted     map[string][]string   // target → ordered fragments
    warnings    []string              // non-fatal warnings
    maxLoopIter int                   // sandbox limit (0 = unlimited)
    loopIter    int                   // running counter across all loops
}
```

### EngineIface — VM → Engine Callback

```go
type EngineIface interface {
    LookupFilter(name string) (FilterFn, bool)
    StrictVariables() bool
    GlobalData() map[string]any
    LoadTemplate(name string) (*compiler.Bytecode, error)
    MaxLoopIter() int
}
```

---

## 9. Value System

**File:** `internal/vm/value.go`

### Value Types

| Type | Constant | Storage | Description |
|------|----------|---------|-------------|
| Nil | `TypeNil` | — | Zero value |
| Bool | `TypeBool` | `ival` (0/1) | Boolean |
| Int | `TypeInt` | `ival` (int64) | Integer |
| Float | `TypeFloat` | `fval` (float64) | Floating point |
| String | `TypeString` | `sval` | String |
| SafeHTML | `TypeSafeHTML` | `sval` | Trusted HTML (bypasses auto-escape) |
| List | `TypeList` | `oval` ([]Value) | Ordered list |
| Map | `TypeMap` | `oval` (map[string]any or *OrderedMap) | Key-value map |
| Resolvable | `TypeResolvable` | `oval` (Resolvable) | Go type with GroveResolve |
| Macro | `TypeMacro` | `oval` (*MacroDef) | Compiled macro |
| LoopVar | `TypeLoopVar` | `oval` (*loopVarData) | Loop metadata |

### Value Structure

```go
type Value struct {
    typ  ValueType
    ival int64    // Bool, Int
    fval float64  // Float
    sval string   // String, SafeHTML
    oval any      // List, Map, Resolvable, Macro, LoopVar
}
```

### Constructors

| Constructor | Input | Result |
|-------------|-------|--------|
| `BoolVal(b)` | `bool` | TypeBool |
| `IntVal(n)` | `int64` | TypeInt |
| `FloatVal(f)` | `float64` | TypeFloat |
| `StringVal(s)` | `string` | TypeString |
| `SafeHTMLVal(s)` | `string` | TypeSafeHTML |
| `ListVal(items)` | `[]Value` | TypeList |
| `MapVal(m)` | `map[string]any` | TypeMap |
| `OrderedMapVal(m)` | `*OrderedMap` | TypeMap (ordered) |
| `ResolvableVal(r)` | `Resolvable` | TypeResolvable |
| `MacroVal(m)` | `*MacroDef` | TypeMacro |

### FromAny — Go → Value Conversion

`FromAny(v any) Value` converts arbitrary Go values:

- `nil` → `Nil`
- `bool` → `BoolVal`
- `int`, `int8–64`, `uint`, `uint64` → `IntVal`
- `float32`, `float64` → `FloatVal`
- `string` → `StringVal`
- `Value` → passthrough
- `Resolvable` → `ResolvableVal`
- `[]any`, `[]string`, `[]int` → `ListVal` (recursive)
- `map[string]any` → `MapVal`
- `*OrderedMap` → `OrderedMapVal`
- Named map types (via reflect) → `MapVal`
- Arbitrary slices (via reflect) → `ListVal` (recursive)
- Fallback: `fmt.Sprintf("%v", v)` → `StringVal`

### Truthiness (Jinja2/Python semantics)

| Type | Truthy when |
|------|-------------|
| Nil | Never |
| Bool | `true` |
| Int | Non-zero |
| Float | Non-zero |
| String/SafeHTML | Non-empty |
| List | Non-empty |
| Map | Non-empty |
| Resolvable | Not nil |
| LoopVar | Always |

### Attribute Resolution (`GetAttr`)

Resolution order for `obj.name`:

1. **Map**: key lookup
2. **Resolvable**: `GroveResolve(name)` call
3. **LoopVar**: special fields (`index`, `index0`, `first`, `last`, `length`, `depth`, `parent`)
4. **Nil**: returns nil (or error in strict mode)

### Index Resolution (`GetIndex`)

- **List**: integer index, returns nil for out-of-bounds
- **Map**: string key lookup

### OrderedMap

Map literals create `*OrderedMap` values that preserve insertion order:

```go
type OrderedMap struct {
    keys []string
    vals map[string]any
}
```

Methods: `Set(key, val)`, `Get(key)`, `Len()`, `Keys()`

---

## 10. Scope & Variable Resolution

**File:** `internal/scope/scope.go`

### Scope Chain

```go
type Scope struct {
    vars   map[string]any
    parent *Scope
}
```

- `Get(key)` walks the chain from current scope to root, returning the first match
- `Set(key, value)` always writes to the current (topmost) scope frame
- Each `for` loop creates a child scope
- `include` shares the caller's scope (plus optional extra variables)
- `render` creates an isolated scope (only passed variables + globals visible)

### Lookup Chain

```
1. Local scope stack ({% set %}, loop vars, macro args)
        │ not found
        ▼
2. Render context (data passed to Render/RenderTemplate)
        │ not found
        ▼
3. Engine global context (engine.SetGlobal())
        │ not found
        ▼
4. Return nil (default) or RuntimeError (StrictVariables mode)
```

---

## 11. Type Coercion

Type coercion is now inline within the VM value system (`internal/vm/`). The standalone `internal/coerce/` package was removed.

### ToBool

| Input | Result |
|-------|--------|
| `nil` | `false` |
| `bool` | value |
| `int` / `int64` | `!= 0` |
| `float64` | `!= 0` |
| `string` | `!= ""` |
| anything else | `true` |

### ToString

| Input | Result |
|-------|--------|
| `nil` | `""` |
| `string` | passthrough |
| `bool` | `"true"` / `"false"` |
| `int` | `strconv.Itoa` |
| `int64` | `strconv.FormatInt` |
| `float64` | `strconv.FormatFloat` (shortest representation) |
| anything else | `fmt.Sprintf("%v", v)` |

### Value-Level Conversions

- `Value.ToInt64()`: Int→direct, Float→truncate, Bool→0/1, String→parse
- `Value.ToFloat64()`: Float→direct, Int→promote, String→parse

---

## 12. Template Syntax Reference

### Delimiters

```
{% expression %}      Output — evaluate and print
{% tag %}             Structural tags — control flow, assignment, composition
{# comment #}         Comments — stripped at parse time, zero runtime cost
```

### Whitespace Control

```
{%- expr -%}          Strip whitespace before and after tag
```

### Variables & Access

```
{% user.name %}               Attribute access
{% items[0].title %}          Index + attribute
{% config["debug"] %}         String key index
{% user.address.city %}       Nested access
```

### Expressions

```
{% count + 1 %}               Arithmetic
{% price * 1.2 %}
{% "Hello, " ~ user.name %}   String concatenation
{% price * 1.2 | round(2) %}  Filter after expression
{% user.role == "admin" %}     Comparison
{% a > b and c != d %}         Logical operators
{% not user.banned %}          Negation
{% active ? name : "Guest" %} Ternary conditional
```

### Filters

```
{% name | upper %}
{% bio | truncate(120, "…") %}
{% items | sort | reverse | first %}
{% price | round(2) %}
{% user_input | safe %}        Bypass auto-escape (explicit trust)
```

### Assignment

```
{% set title = "Welcome" %}
{% set total = items | sum %}
```

### Let Block (multi-variable assignment)

```
{% #let %}
  bg = "#d1ecf1"
  border = "#bee5eb"
  fg = "#0c5460"

  {% :else if type == "warning" %}
    bg = "#fff3cd"
    fg = "#856404"
  {% :else if type == "error" %}
    bg = "#f8d7da"
    fg = "#721c24"
  {% /let %}
```

Rules:
- Bare `name = expression` per line (no delimiters)
- Full expression syntax on right-hand side
- All assigned variables written to outer scope
- No HTML output inside the block

### Control Flow

```
{% #if condition %}
  ...
{% :else if condition %}
  ...
{% :else %}
  ...
{% /if %}

{% #each items as item %}
  {% loop.index %}: {% item.name %}
{% :empty %}
  No items found.
{% /each %}

{% #each items as i, item %}     Two-variable form (index + value)
{% #each map as key, value %}    Two-variable form (key + value for maps)
{% #each range(1, 11) as i %}    Range iteration
{% /each %}
```

### Loop Variable (`loop`)

| Variable | Description |
|----------|-------------|
| `loop.index` | 1-based position |
| `loop.index0` | 0-based position |
| `loop.first` | `true` on first iteration |
| `loop.last` | `true` on last iteration |
| `loop.length` | Total items |
| `loop.depth` | 1 for outer, 2 for first nested, etc. |
| `loop.parent` | Parent loop's `loop` object |

### Range Function

- `range(stop)` → `[0, stop)`
- `range(start, stop)` → `[start, stop)` (end-exclusive)
- `range(start, stop, step)` → stepped sequence
- Negative step produces descending sequence
- All arguments coerced to integers

### Data Literals

```
{% set colors = ["red", "green", "blue"] %}
{% set matrix = [[1, 2], [3, 4]] %}
{% set theme = { bg: "#fff", fg: "#333", icon: "!" } %}
{% set nested = { card: { padding: "1rem" } } %}
```

- Lists: `[expr, ...]` — comma-separated, trailing comma allowed
- Maps: `{ key: expr, ... }` — keys are unquoted identifiers, ordered by insertion
- No computed keys, no spread/merge operators

### Capture

```
{% #capture nav %}
  {% #each menu as item %}{% item.label %}{% /each %}
{% /capture %}
{% nav %}
```

### Verbatim Block

```
{% #verbatim %}
  {% this is not evaluated %}
{% /verbatim %}
```

### Comments

```
{# This is a comment — stripped at parse time #}
```

---

## 13. Filters

### String Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `upper` | `upper` | Convert to uppercase |
| `lower` | `lower` | Convert to lowercase |
| `title` | `title` | Title-case each word |
| `capitalize` | `capitalize` | First character uppercase, rest lowercase |
| `trim` | `trim` | Strip leading and trailing whitespace |
| `lstrip` | `lstrip` | Strip leading whitespace |
| `rstrip` | `rstrip` | Strip trailing whitespace |
| `replace` | `replace(old, new)` | Replace first occurrence |
| `truncate` | `truncate(n, suffix="…")` | Truncate to n chars (suffix excluded from count) |
| `center` | `center(width)` | Center-pad to width |
| `ljust` | `ljust(width)` | Left-justify (right-pad) to width |
| `rjust` | `rjust(width)` | Right-justify (left-pad) to width |
| `split` | `split(sep)` | Split string into list |
| `wordcount` | `wordcount` | Count words in string |

### Collection Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `length` | `length` | Number of elements |
| `first` | `first` | First element or nil |
| `last` | `last` | Last element or nil |
| `join` | `join(sep="")` | Join list elements with separator |
| `sort` | `sort` | Sort ascending |
| `reverse` | `reverse` | Reverse the list |
| `unique` | `unique` | Remove duplicates |
| `min` | `min` | Minimum value |
| `max` | `max` | Maximum value |
| `sum` | `sum` | Sum of numeric list |
| `map` | `map(attr)` | Extract attribute from each element |
| `batch` | `batch(size)` | Split list into batches of given size |
| `flatten` | `flatten` | Flatten one level of nesting |
| `keys` | `keys` | Map keys as list |
| `values` | `values` | Map values as list |

### Numeric Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `abs` | `abs` | Absolute value |
| `round` | `round(n=0)` | Round to n decimal places |
| `ceil` | `ceil` | Round up to nearest integer |
| `floor` | `floor` | Round down to nearest integer |
| `int` | `int` | Convert to integer |
| `float` | `float` | Convert to float |

### Type / Logic Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `default` | `default(fallback)` | Return fallback if value is nil, false, or empty string |
| `string` | `string` | Convert to string |
| `bool` | `bool` | Convert to boolean |

### HTML Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `escape` | `escape` | HTML-escape (redundant unless piped from `safe`) |
| `striptags` | `striptags` | Remove all HTML tags |
| `nl2br` | `nl2br` | Replace newlines with `<br>` |

### Special Filters

| Filter | Signature | Description |
|--------|-----------|-------------|
| `safe` | `safe` | Mark string as trusted HTML — bypasses auto-escape. This is the **only** escape hatch for auto-escaping. |

### Custom Filter Registration

```go
eng.RegisterFilter("shout", func(v grove.Value, args []grove.Value) (grove.Value, error) {
    return grove.StringValue(strings.ToUpper(v.String()) + "!!!"), nil
})

// Filter that returns trusted HTML
eng.RegisterFilter("markdown", grove.FilterFunc(
    func(v grove.Value, args []grove.Value) (grove.Value, error) {
        html := renderMarkdown(v.String())
        return grove.SafeHTMLValue(html), nil
    },
    grove.FilterOutputsHTML(),
))
```

---

## 14. Template Composition

### Include (scope-sharing)

```
{% include "partials/nav.grov" %}
{% include "partials/nav.grov" section="about" active=true %}
```

- Shares the caller's scope — the included template can read all parent variables
- Extra key=value parameters are additional variables injected into scope
- Space-separated parameters (no commas, no `with` keyword)

### Render (isolated scope)

```
{% render "components/card.grov" title="Widget" %}
```

- **Always** creates an isolated scope — only passed parameters and engine globals are visible
- Equivalent to `include ... isolated` from the original design

### Import (component import)

```
{% import Card from "components/card" %}
{% import Badge from "components/badge" %}

<Card title="My Card">
  <Badge label="New" />
</Card>
```

- Brings named components into scope
- Components are now the primary unit of composition (no macros in current system)

---

## 15. Layouts & Composition

Grove does not use template inheritance (`extends`/`block`). Instead, layouts are components with named slots.

### Layout as Component

```
{# layouts/base.grov #}
<Component name="Base" title>
  <!DOCTYPE html>
  <html>
  <head><title>{% title %}</title></head>
  <body>
    {% slot "content" %}
  </body>
  </html>
</Component>
```

### Using a Layout

```
{% import Base from "layouts/base" %}

<Base title="About">
  {% #fill "content" %}
    <h1>About Us</h1>
  {% /fill %}
</Base>
```

### Multi-Level Layouts

Layouts compose naturally — a layout component can itself use another layout component as its base, creating hierarchies without special syntax.

---

## 16. Components & Slots

### Component Definition

```
{# components/card.grov #}
<Component name="Card" title variant="default">
  <div class="card card--{% variant %}">
    <h2>{% title %}</h2>
    {% slot "actions" %}
    <div class="body">{% slot %}</div>
    <footer>{% slot "footer" %}Default footer{% /slot %}</footer>
  </div>
</Component>
```

### Component Invocation

```
{% import Card from "components/card" %}

<Card title="Orders" variant="primary">
  <p>Main content goes in the default slot.</p>

  {% #fill "actions" %}
    <button>View All</button>
  {% /fill %}
</Card>
```

### Props

- Declared as attributes on `<Component>` definition tag
- Required props (no default value) cause a `RuntimeError` if missing
- Unknown props (not declared) cause a `RuntimeError`

### Slots

- `{% slot %}` — default (unnamed) slot (self-closing)
- `{% #slot "name" %}fallback{% /slot %}` — named slot with fallback content
- `{% #fill "name" %}content{% /fill %}` — provides content for a named slot
- Content outside `{% #fill %}` blocks feeds the default slot
- Fills are rendered in the **caller's scope** (not the component's scope)

---

## 17. Macros

**Status: Removed.** Macros are no longer part of the Grove template language. Use components (`<Component>`) instead — they provide the same composition capabilities with better clarity and type safety.

## 18. Web Primitives

### Asset Declaration

```
{% asset "composites/nav/nav.css" type="stylesheet" %}
{% asset "composites/nav/nav.js" type="script" defer %}
```

- The `src` argument is a logical name. At render time, the engine resolves
  it through a configured `AssetResolver` (see §22) and stores the resolved
  URL on the asset; if no resolver is set, the name passes through unchanged.
- Deduplication uses the **resolved** `src` — two logical names that resolve
  to the same URL collapse to one asset.
- `Priority` attribute controls ordering within type groups (higher = earlier).
- Supports boolean attributes (`defer`, `async`) stored as key→"" in Attrs.

### Asset Output Helpers

```go
result.HeadHTML()  // <link rel="stylesheet"> tags for stylesheet assets
result.FootHTML()  // <script> tags for script assets
```

### Asset Resolution

The engine accepts a pluggable resolver of type
`grove.AssetResolver = func(logicalName string) (string, bool)`. Configure
it via `grove.WithAssetResolver` at construction or swap it at runtime with
`(*Engine).SetAssetResolver` (atomic; safe against concurrent renders).
`(*Engine).ReferencedAssets` exposes the set of logical names seen during
rendering, for prune passes.

The optional sibling package `pkg/grove/assets` provides a `Builder` +
`Manifest` that implement this resolver over a content-hashed, minified
`dist/` directory. Importing it is opt-in; the core engine never imports
it. See [`spec/asset-pipeline.md`](../../spec/asset-pipeline.md).

### Metadata

```
{% meta name="description" content="A page about Grove." %}
{% meta property="og:title" content="Grove Engine" %}
```

- Collected into `RenderResult.Meta` (map[string]string)
- Last-write-wins semantics; warns on key collision

### Hoisting

```
{% #hoist %}
  <script>trackPage("{% page.title %}");</script>
{% /hoist %}
```

- Target is implicitly the named section (e.g., "head", "foot")
- Body is rendered and appended to `RenderResult.Hoisted[target]`
- Retrieved via `result.GetHoisted(target)`

---

## 19. Template Stores

### Store Interface

```go
type Store interface {
    Load(name string) ([]byte, error)
}
```

### MemoryStore

```go
store := grove.NewMemoryStore()
store.Set("page.grov", `Hello {{ name }}!`)
eng := grove.New(grove.WithStore(store))
```

- Thread-safe (`sync.RWMutex`)
- Ideal for tests and embedded templates

### FileSystemStore

```go
store := grove.NewFileSystemStore("/path/to/templates")
eng := grove.New(grove.WithStore(store))
```

- Template names are forward-slash relative paths
- **Path traversal prevention**: names cleaned via `path.Clean`; `..` and absolute paths rejected before any disk I/O
- **Fallback resolution** for name `foo`:
  1. Exact match: `<root>/foo`
  2. Extension: `<root>/foo.grov`
  3. Directory: `<root>/foo/foo.grov`
- Thread-safe (stateless reads)

---

## 20. Caching

### LRU Cache

- Default capacity: **512 entries**
- Configurable: `grove.WithCacheSize(n)`
- O(1) get/set via doubly-linked list + hash map
- Keyed by template name (for named templates via `Render`)
- Inline templates (`RenderTemplate`) are not cached

### Eviction

When cache is full, the least-recently-used entry is evicted. Cache is protected by `sync.Mutex` for concurrent access.

### Constant Cache

VM constant pools are pre-compiled from `[]any` to `[]Value` on first execution, cached in a `sync.Map` keyed by `*Bytecode` pointer.

---

## 21. Security & Sandboxing

### Auto-Escaping

- **On by default** — all `OP_OUTPUT` values are HTML-escaped unless the value is `TypeSafeHTML`
- `{{ user_input | safe }}` marks a string as trusted HTML — the **only** way to bypass auto-escape
- Filters that return `SafeHTMLValue` (via `FilterOutputsHTML()`) bypass escape automatically
- `OP_OUTPUT_RAW` writes verbatim (used for raw blocks)

### Sandbox Configuration

```go
eng := grove.New(grove.WithSandbox(grove.SandboxConfig{
    AllowedTags:    []string{"if", "for", "set"},
    AllowedFilters: []string{"upper", "lower", "truncate", "escape"},
    MaxLoopIter:    10_000,
}))
```

### Enforcement Tiers

1. **Compile-time**: `AllowedTags` checked by the parser; `AllowedFilters` checked after compilation by walking bytecode instructions. Banned tags/filters produce `ParseError` before any execution.
2. **Runtime**: `MaxLoopIter` incremented and checked at `OP_FOR_INIT` (first iteration) and `OP_FOR_STEP` (each subsequent iteration); the running counter equals total body executions across all loops (including nested) in a render pass. Exceeding the limit produces `RuntimeError`.

### Path Traversal Prevention

`FileSystemStore.Load()` rejects:
- Absolute paths (`/etc/passwd`)
- Paths that escape the root after cleaning (`../../secret`)
- Double-checked containment via prefix match

### Resolvable — Explicit Field Exposure

```go
type User struct {
    ID        int
    Name      string
    AuthToken string  // deliberately hidden
}

func (u User) GroveResolve(key string) (any, bool) {
    switch key {
    case "id":   return u.ID, true
    case "name": return u.Name, true
    }
    return nil, false
}
```

Grove does **not** walk struct fields via `reflect`. Only types implementing `Resolvable` (or passed as `grove.Data`/`map[string]any`) are accessible.

### Security Assumption

The sandbox controls what *templates* can do — it does **not** sandbox registered Go filters and tags. Custom filters have full Go access. When operating in sandbox mode, `AllowedFilters` must enumerate only filters that are safe for untrusted input.

---

## 22. Public API

### Engine

```go
// Create
eng := grove.New(opts ...grove.Option) *grove.Engine

// Render named template from store
result, err := eng.Render(ctx, "page.grov", grove.Data{"key": "value"})

// Render inline template string
result, err := eng.RenderTemplate(ctx, `Hello {{ name }}!`, grove.Data{"name": "World"})

// Render to writer
err := eng.RenderTo(ctx, "page.grov", data, w)

// Register globals
eng.SetGlobal("siteName", "Acme")

// Register custom filter
eng.RegisterFilter("name", filterFn)

// Load and compile (cache-through)
bc, err := eng.LoadTemplate("page.grov")
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithStore(s)` | `nil` | Template storage backend |
| `WithStrictVariables(bool)` | `false` | Error on undefined variable access |
| `WithCacheSize(n)` | `512` | LRU cache capacity |
| `WithSandbox(cfg)` | `nil` | Sandbox restrictions |
| `WithAssetResolver(r)` | `nil` | Logical-name → URL mapping for `{% asset %}`. See §18 and `spec/asset-pipeline.md` |

### Data

```go
type Data map[string]any
```

### RenderResult

```go
type RenderResult struct {
    Body     string
    Assets   []Asset
    Meta     map[string]string
    Hoisted  map[string][]string
    Warnings []Warning
}
```

Methods:
- `HeadHTML() string` — stylesheet `<link>` tags
- `FootHTML() string` — script `<script>` tags
- `GetHoisted(target string) string` — concatenated hoisted content

### Asset

```go
type Asset struct {
    Src      string
    Type     string
    Attrs    map[string]string
    Priority int
}
```

### Asset Resolver

```go
type AssetResolver = func(logicalName string) (string, bool)
```

Engine methods: `SetAssetResolver(r)`, `AssetResolver() AssetResolver`,
`RecordAssetRef(name)`, `ReferencedAssets() map[string]struct{}`,
`ResetReferencedAssets()`.

### Asset Pipeline (`pkg/grove/assets`)

Optional sibling package. See `docs/asset-pipeline.md` for usage and
`spec/asset-pipeline.md` for design. Surface:

- `Config`, `Builder`, `New`, `NewWithDefaults`
- `(*Builder).Build`, `.Watch`, `.Handler`, `.Route`, `.SetReferencedNameProvider`, `.Config`
- `Manifest`, `NewManifest`, `LoadManifest`, `(*Manifest).Resolve`, `.Entries`, `.Sources`, `.Stats`, `.Set`, `.SetSource`, `.SetStats`, `.Delete`, `.Save`
- `Transformer`, `NoopTransformer`
- `WatchHandlers{OnChange, OnError, OnEvent}`, `Event`, `EventType`, `BuildStats`

Optional `pkg/grove/assets/minify` sub-package provides a `MinifyTransformer`
backed by `tdewolff/minify/v2`.

### Filter Types

```go
type FilterFn = func(v Value, args []Value) (Value, error)
type FilterDef struct { Fn FilterFn; OutputsHTML bool }
type FilterSet = map[string]any
```

### Value (re-exported from internal/vm)

```go
type Value = vm.Value
var Nil = vm.Nil
func StringValue(s string) Value
func SafeHTMLValue(s string) Value
func ArgInt(args []Value, i, def int) int
```

### Resolvable

```go
type Resolvable interface {
    GroveResolve(key string) (any, bool)
}
```

### Store Types (re-exported)

```go
type MemoryStore = store.MemoryStore
func NewMemoryStore() *MemoryStore
type FileSystemStore = store.FileSystemStore
func NewFileSystemStore(root string) *FileSystemStore
```

---

## 23. Error Model

### ParseError

Returned for syntax errors and sandbox violations at compile time.

```go
type ParseError struct {
    Template string  // template name (empty for inline)
    Message  string
    Line     int     // 1-based
    Column   int     // 1-based
}
// Format: "template.grov:42:7: unexpected token"
// Format (inline): "line 42:7: unexpected token"
```

### RuntimeError

Returned for errors during template execution.

```go
type RuntimeError struct {
    Template string  // template name (empty for inline)
    Message  string
    Line     int     // 1-based
}
// Format: "template.grov:42: undefined variable 'x'"
// Format (inline): "line 42: undefined variable 'x'"
```

Both types implement the `error` interface and can be unwrapped with `errors.As`.

---

## 24. Concurrency Model

- **Bytecode**: immutable after compilation; safe to share across goroutines
- **Engine**: safe for concurrent use; internal state protected by LRU cache mutex
- **VM instances**: pooled via `sync.Pool`; acquired per-render, reset and returned after use
- **MemoryStore**: protected by `sync.RWMutex`
- **FileSystemStore**: stateless reads; inherently safe
- **Constant cache**: `sync.Map` for lock-free concurrent reads after initial population
- **Render context**: allocated per-render, not shared across concurrent renders

### VM Fixed-Size Arrays

The VM uses fixed-size arrays to avoid allocations:
- `stack [256]Value` — value stack
- `loops [32]loopState` — loop state (max nesting depth: 32)
- `loopVars [32]loopVarData` — loop metadata
- `loopScopes [32]*scope.Scope` — loop body scopes
- `captures [8]captureFrame` — capture buffer stack (max nesting: 8)
- `compStack [16]componentFrame` — component call stack (max nesting: 16)

---

## 25. Performance Characteristics

### Design Choices for Performance

1. **Bytecode VM over tree-walking** — tight switch loop, branch-predictor-friendly; flat `[]Instruction` is cache-line-friendly
2. **Fixed-width 8-byte instructions** — no variable-length decoding overhead
3. **VM pooling** — `sync.Pool` eliminates per-render allocation of VM state
4. **Constant pre-compilation** — `[]any` → `[]Value` conversion happens once per bytecode, cached globally
5. **Fixed-size arrays** — stack, loops, captures, component stack all use pre-allocated arrays
6. **OrderedMap for map literals** — preserves insertion order without sort overhead
7. **EstimatedOutputSize** — compiler hints for output buffer pre-sizing

### Expected Throughput

| Template complexity | Estimated throughput |
|---------------------|---------------------|
| Simple variable substitution (`Hello {{ name }}`) | ~8–10M/s |
| Typical web page (50 variables, 2 loops, filters) | ~1–3M/s |
| Heavy inheritance + components (5 levels, 10 includes) | ~300k–800k/s |
| Sandbox mode (counter checked per loop step) | ~600k–1.5M/s |

### Test Command

```bash
go clean -testcache && go test ./... -v
```
