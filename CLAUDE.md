# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Grove is a bytecode-compiled template engine for Go. Templates (`.grov` files) are lexed, parsed into an AST, compiled to bytecode, and executed on a stack-based VM. The Go module is `github.com/wispberry-tech/grove` (Go 1.24). The only external dependency is `testify` (test-only).

## Commands

```bash
# Run all tests (verbose, no cache)
go clean -testcache && go test ./... -v

# Run tests for a single package
go test ./pkg/grove/ -v -run TestName

# Build check
go build ./...
```

No Makefile, linter config, or CI pipeline exists. Use `gofmt` for formatting.

## Architecture

**Render pipeline:** Source → Lexer → Parser → AST → Compiler → Bytecode → VM → RenderResult

| Package | Role |
|---------|------|
| `pkg/grove/` | Public API: `Engine`, `RenderResult`, options, store interfaces, filter registration |
| `internal/lexer/` | State-machine tokenizer |
| `internal/parser/` | Token stream → AST |
| `internal/ast/` | AST node types |
| `internal/compiler/` | AST → bytecode with opcode emission |
| `internal/vm/` | Stack-based bytecode interpreter, filter dispatch |
| `internal/scope/` | Variable lookup chain (scope stack with shadow handling) |
| `internal/filters/` | 40+ built-in filters (string, collection, numeric, HTML) |
| `internal/store/` | Template storage: `MemoryStore`, `FileSystemStore` |
| `internal/coerce/` | Type coercion between template value types |
| `internal/groverrors/` | Shared error types (`ParseError`, `RuntimeError`) |

**Key design points:**
- Compiled bytecode is immutable and shared across goroutines.
- VM instances are pooled via `sync.Pool`; rendering is atomic per call.
- Engine uses a mutex-protected LRU cache (default 512 entries) for compiled templates.
- `RenderResult` accumulates assets, meta tags, and hoisted content across nested renders.
- Auto-escaping is on by default; `safe` filter bypasses it.
- Sandboxing supports tag/filter whitelists and loop iteration limits.

## Template features

Variables, filters (pipe syntax), arithmetic/comparison/logical expressions (including ternary `? :`), `if`/`elif`/`else`, `for`/`empty`, `range`, `set`, `let` (multi-variable assignment block), `capture`, list/map literals, macros (`macro`/`call`), includes (`include`/`render`/`import`), layout inheritance (`extends`/`block`/`super()`), components with slots (`component`/`slot`/`fill`), and web primitives (`asset`/`meta`/`hoist`).

## Testing

Tests use table-driven patterns with `testify/require`. Test helpers `newEngine()`, `render()`, and `renderErr()` are defined in `pkg/grove/engine_test.go`. The test suite covers all template features across 9 test files in `pkg/grove/` and `internal/lexer/`.

## Other resources

- `spec/` — Design specifications and comparative analysis
- `plans/` — Phased implementation plans (1–8) documenting the build history
- `examples/blog/` — Full working blog app demonstrating the engine
