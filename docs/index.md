# Grove Documentation

Grove is a bytecode-compiled template engine for Go. Templates are lexed, parsed into an AST, compiled to bytecode, and executed on a stack-based VM. The engine is safe for concurrent use — compiled bytecode is immutable and shared across goroutines, and VM instances are pooled.

## Contents

| Page | Description |
|------|-------------|
| [Getting Started](getting-started.md) | Install Grove, configure an engine, render your first template |
| [Template Syntax](template-syntax.md) | Variables, expressions, operators, control flow, loops, assignment, literals |
| [Template Inheritance](template-inheritance.md) | Base layouts with `extends`, `block`, and `super()` |
| [Components](components.md) | Reusable templates with `props`, `slot`, and `fill` |
| [Macros & Includes](macros-and-includes.md) | Template functions with `macro`, and composition with `include`, `render`, `import` |
| [Filters](filters.md) | All 42 built-in filters — string, collection, numeric, HTML, type conversion |
| [Web Primitives](web-primitives.md) | `asset`, `meta`, `hoist` tags and `RenderResult` integration |
| [API Reference](api-reference.md) | Go types, methods, options, stores, custom filters, error types |
| [Examples](examples.md) | Walkthrough of the blog example app |
