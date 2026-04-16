# Bytecode & VM

Grove is a bytecode-compiled template engine. Templates are parsed once, compiled to immutable bytecode, and executed on a stack-based virtual machine. Compiled bytecode is cached and shared across goroutines; VM instances are pooled and reused. The result is a render path that allocates little per request and is safe for concurrent use out of the box.

## Pipeline

```
source (.grov) ─▶ lexer ─▶ parser ─▶ AST ─▶ compiler ─▶ bytecode ─▶ VM ─▶ RenderResult
```

| Stage | Package | Role |
|-------|---------|------|
| Lexer | `internal/lexer` | State-machine tokenizer — emits `{% %}` directives, component tags, and text chunks |
| Parser | `internal/parser` | Builds the AST from the token stream |
| AST | `internal/ast` | Tree of nodes (expressions, control flow, component calls) |
| Compiler | `internal/compiler` | Walks the AST and emits fixed-width bytecode instructions |
| Bytecode | `internal/compiler` | Immutable instruction slice plus constant, name, macro, block, and component pools |
| VM | `internal/vm` | Stack-based interpreter — executes instructions, runs filters, emits output |
| RenderResult | `pkg/grove` | Output body plus accumulated assets, meta, and hoisted content |

## Immutable bytecode

The compiler produces a `*compiler.Bytecode`:

```go
type Bytecode struct {
    Instrs     []Instruction
    Consts     []any             // constant pool
    Names      []string          // variable / attr / filter names
    Macros     []MacroDef
    Blocks     []BlockDef
    Components []ComponentDef
    // ...
}
```

Once compiled, `Bytecode` is never mutated. The same pointer is handed to every concurrent `Render` call — no defensive copying, no locking on read. All per-render state lives on the VM, not on the compiled artefact.

Instructions are fixed-width (8 bytes: `A` + `B` + `Op` + `Flags` + padding). The VM dispatches on `Op` in a tight switch; `A` and `B` index into the constant, name, block, or component pools.

### Opcode categories

A short tour of the instruction set (full list in `internal/compiler/bytecode.go`):

| Category | Opcodes | Purpose |
|----------|---------|---------|
| Stack & constants | `OP_PUSH_CONST`, `OP_PUSH_NIL` | Push values onto the operand stack |
| Variables | `OP_LOAD`, `OP_STORE_VAR`, `OP_GET_ATTR`, `OP_GET_INDEX` | Scope lookup, assignment, attribute / index access |
| Arithmetic & logic | `OP_ADD`, `OP_SUB`, `OP_MUL`, `OP_DIV`, `OP_MOD`, `OP_NEGATE`, `OP_EQ`, `OP_NEQ`, `OP_LT`, `OP_LTE`, `OP_GT`, `OP_GTE`, `OP_AND`, `OP_OR`, `OP_NOT`, `OP_CONCAT` | Expression evaluation |
| Output | `OP_OUTPUT`, `OP_OUTPUT_RAW` | Emit a value (escaped or verbatim) to the active buffer |
| Control flow | `OP_JUMP`, `OP_JUMP_FALSE`, `OP_FOR_INIT`, `OP_FOR_STEP`, `OP_FOR_BIND_1`, `OP_FOR_BIND_KV`, `OP_CALL_RANGE` | Conditionals and loops |
| Filters | `OP_FILTER` | Invoke a registered filter by name |
| Composition | `OP_IMPORT`, `OP_COMPONENT`, `OP_SLOT`, `OP_DYNAMIC_COMPONENT` | Component import and invocation |
| Inheritance | `OP_EXTENDS`, `OP_BLOCK_RENDER`, `OP_SUPER` | Template inheritance and block overrides |
| Macros | `OP_MACRO_DEF`, `OP_MACRO_DEF_PUSH`, `OP_CALL_MACRO_VAL`, `OP_CALL_MACRO_CALL`, `OP_CALL_CALLER` | Macro definition and call |
| Captures | `OP_CAPTURE_START`, `OP_CAPTURE_END` | Redirect output into a scope variable |
| Web primitives | `OP_ASSET`, `OP_META`, `OP_HOIST` | Collect render-wide side effects into `RenderResult` |
| Literals | `OP_BUILD_LIST`, `OP_BUILD_MAP` | List and map construction |

## Template cache (LRU, default 512)

`Engine` holds an LRU cache of compiled bytecode keyed by template name:

```go
eng := grove.New(
    grove.WithStore(grove.NewFileSystemStore("templates")),
    grove.WithCacheSize(1024), // default is 512
)
```

- First `Render("page")` loads the template from the store, compiles it, and caches the `*Bytecode`.
- Every subsequent `Render("page")` is a hash-map lookup plus a VM execution — no lexing, parsing, or compiling.
- The cache is mutex-protected. Compilation for a given name happens once; contending callers block on the lock and then hit the cache.

Compiled entries live for the lifetime of the engine. To pick up template changes, restart the process (production) or rebuild the engine. The `FileSystemStore` reads from disk on each compile, so a cache miss after eviction or restart automatically picks up the latest source.

## VM pool

The VM is allocated once per goroutine and returned to a shared `sync.Pool` at the end of each render:

```go
var vmPool = sync.Pool{
    New: func() any { return &VM{} },
}
```

Each render cycle:

1. Acquire a VM from the pool.
2. Attach the compiled bytecode, render data, and a fresh `renderCtx` (assets, meta, hoisted buffers).
3. Dispatch instructions in a tight switch until `OP_HALT`.
4. Copy accumulated output and render context into the returned `RenderResult`.
5. Reset per-render state and return the VM to the pool.

The VM struct (stack, scope chain, loop and capture frames, component stack) is reused across renders. Only the data that must change per request — template data, output buffer contents, render context — is re-initialised.

## Concurrency guarantees

- `Engine.Render`, `Engine.RenderTo`, and `Engine.RenderTemplate` are safe to call from any number of goroutines.
- Globals set via `SetGlobal` should be established during bootstrap. Reads are unlocked and assume globals are stable after startup.
- Filters registered via `RegisterFilter` are called from pooled VMs concurrently — implementations must be goroutine-safe. Pure functions over their arguments are the easiest path.
- `RenderResult` returned from `Render` is not shared — each call gets its own. Writing to it after the call returns is fine.

## RenderResult accumulation

`{% asset %}`, `{% meta %}`, and `{% #hoist %}` all write into the VM's per-render context rather than emitting inline HTML. Nested component renders merge their context into the parent's, so a component three levels deep can declare a stylesheet and it bubbles up to the top-level `RenderResult` without extra plumbing.

See [Web Primitives](web-primitives.md) for the full `RenderResult` API.

## Sandboxing

`WithSandbox` applies three restrictions, each enforced at a different stage of the pipeline:

```go
eng := grove.New(
    grove.WithStore(store),
    grove.WithSandbox(grove.SandboxConfig{
        AllowedTags:    []string{"if", "each", "set"},
        AllowedFilters: []string{"upper", "lower", "escape"},
        MaxLoopIter:    5000,
    }),
)
```

| Knob | Stage | Error | Behaviour |
|------|-------|-------|-----------|
| `AllowedTags` | Parser | `ParseError` | Parser rejects any `{% tag %}` not in the list before the AST is built. |
| `AllowedFilters` | Post-compile | `ParseError` | After compile, the engine walks the bytecode (including macros, blocks, and component fills) and rejects any `OP_FILTER` referencing a filter not in the list. Caching only happens on pass. |
| `MaxLoopIter` | VM runtime | `RuntimeError` | VM counts body executions across all loops in a render pass (each iteration of every nested loop counts once). `MaxLoopIter=N` permits exactly N body executions; `0` means unlimited. |

`nil` for either list means "allow all". The three knobs compose — you can use any subset.

## See also

- [API Reference](api-reference.md) — engine options, render methods, store interfaces
- [Components](components.md) — how component invocation uses `OP_COMPONENT` and `OP_SLOT`
- [Web Primitives](web-primitives.md) — what `OP_ASSET`, `OP_META`, and `OP_HOIST` collect
