# API Reference

Import:

```go
import "github.com/wispberry-tech/grove/pkg/grove"
```

## Engine

```go
func New(opts ...Option) *Engine
```

Creates a new template engine. Safe for concurrent use — multiple goroutines can call render methods simultaneously.

### Rendering Methods

```go
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error)
```

Compiles and renders an inline template string. Does not support `{% import %}`, `<Component>`, or other store-dependent features (these require a store).

```go
func (e *Engine) Render(ctx context.Context, name string, data Data) (RenderResult, error)
```

Loads a named template from the store, compiles it (with caching), and renders it. Requires `WithStore`.

```go
func (e *Engine) RenderTo(ctx context.Context, name string, data Data, w io.Writer) error
```

Like `Render`, but streams output to an `io.Writer`. Does not return a `RenderResult` — use `Render` if you need access to collected assets, meta, or hoisted content.

```go
func (e *Engine) LoadTemplate(name string) (*compiler.Bytecode, error)
```

Compiles and caches a template without rendering it. Useful for pre-warming the cache.

### Engine Configuration

```go
func (e *Engine) SetGlobal(key string, value any)
```

Registers a global variable available in all renders. Globals have the lowest priority — render data overrides them, and template-local variables override render data.

```go
func (e *Engine) RegisterFilter(name string, fn any)
```

Registers a custom filter. `fn` can be a `FilterFn` or a `*FilterDef` (created with `FilterFunc`).

## Options

```go
func WithStore(s store.Store) Option
```

Sets the template store used by `Render` and `{% import %}`.

```go
func WithStrictVariables(strict bool) Option
```

When true, accessing an undefined variable returns a `RuntimeError` instead of an empty string.

```go
func WithCacheSize(n int) Option
```

Sets the LRU cache capacity for compiled bytecode. Default: 512. Pass 0 to use the default.

```go
func WithSandbox(cfg SandboxConfig) Option
```

Applies security restrictions to all templates rendered by this engine.

```go
func WithAssetResolver(r AssetResolver) Option
```

Installs a resolver called at render time for every `{% asset %}` tag.
See [Asset Pipeline](#asset-pipeline) below.

## SandboxConfig

```go
type SandboxConfig struct {
    AllowedTags    []string  // nil = all allowed; non-nil = whitelist
    AllowedFilters []string  // nil = all allowed; non-nil = whitelist
    MaxLoopIter    int       // 0 = unlimited
}
```

- `AllowedTags`: when set, only listed tag names are permitted (e.g., "if", "each", "set", "import"). Others cause a `ParseError` at compile time.
- `AllowedFilters`: when set, only listed filters are permitted. Others cause a `ParseError` at compile time.
- `MaxLoopIter`: maximum total loop body executions across all loops in a single render. Each iteration of any loop — including nested loops — counts as one. Exceeding this causes a `RuntimeError`.

```go
eng := grove.New(grove.WithSandbox(grove.SandboxConfig{
    AllowedTags:    []string{"if", "each", "import", "set"},
    AllowedFilters: []string{"upper", "lower", "escape", "safe", "default"},
    MaxLoopIter:    10000,
}))
```

## Data

```go
type Data map[string]any
```

The map type passed to render methods. Values can be any Go type: strings, numbers, booleans, slices (`[]any`), maps (`map[string]any`), or types implementing `Resolvable`.

## Resolvable

```go
type Resolvable interface {
    GroveResolve(key string) (any, bool)
}
```

Implement this interface on Go types to control which fields are accessible in templates. Only keys that return `(value, true)` are visible. All other field access returns empty (or errors in strict mode).

```go
type User struct {
    Name     string
    Email    string
    password string
}

func (u User) GroveResolve(key string) (any, bool) {
    switch key {
    case "name":
        return u.Name, true
    case "email":
        return u.Email, true
    }
    return nil, false
}
```

```html
{% user.name %}      {# "Alice" #}
{% user.email %}     {# "alice@example.com" #}
{% user.password %}  {# empty — not exposed #}
```

## Stores

### MemoryStore

```go
func NewMemoryStore() *MemoryStore
```

Creates an empty in-memory template store. Thread-safe.

```go
func (s *MemoryStore) Set(name, content string)
```

Adds or updates a template.

```go
store := grove.NewMemoryStore()
store.Set("base.grov", `<html>{% slot %}</html>`)
store.Set("page.grov", `{% import Base from "base" %}<Base>Hello</Base>`)
```

### FileSystemStore

```go
func NewFileSystemStore(root string) *FileSystemStore
```

Creates a store that loads templates from disk. Template names are forward-slash paths relative to `root`.

```go
store := grove.NewFileSystemStore("./templates")
eng := grove.New(grove.WithStore(store))

// Loads ./templates/pages/home.html
result, err := eng.Render(ctx, "pages/home.html", data)
```

**Security:** Rejects absolute paths and `..` traversal. Paths are cleaned and verified to stay within the root directory.

## RenderResult

```go
type RenderResult struct {
    Body     string
    Assets   []Asset
    Meta     map[string]string
    Hoisted  map[string][]string
    Warnings []Warning
}
```

See [Web Primitives](web-primitives.md) for detailed documentation on `RenderResult`, `Asset`, `Warning`, and the helper methods `HeadHTML()`, `FootHTML()`, and `GetHoisted()`.

## Asset Pipeline

```go
type AssetResolver func(logicalName string) (string, bool)
```

Maps a logical asset name (the `src` argument of `{% asset %}`) to a served
URL. Returning `("", false)` causes the engine to fall through to the original
name. A nil resolver is treated as passthrough.

### Engine methods

```go
func (e *Engine) SetAssetResolver(r AssetResolver)
func (e *Engine) AssetResolver() AssetResolver
func (e *Engine) RecordAssetRef(logicalName string)
func (e *Engine) ReferencedAssets() map[string]struct{}
func (e *Engine) ResetReferencedAssets()
```

- `SetAssetResolver` is an atomic swap — safe to call from any goroutine
  while concurrent renders are in flight. Used by watch mode to update the
  active manifest without a restart.
- `AssetResolver` returns the current resolver, or `nil` when unused.
- `RecordAssetRef` is called by the VM during OP_ASSET; the engine stores
  the logical name in a lazy set (allocated only when a resolver is
  configured).
- `ReferencedAssets` returns a snapshot of that set for pruning or debugging.
- `ResetReferencedAssets` clears the set — typically between prune windows.

### Sibling package: `pkg/grove/assets`

```go
import "github.com/wispberry-tech/grove/pkg/grove/assets"
```

| Symbol | Purpose |
|--------|---------|
| `Config` | Builder configuration: `SourceDir`, `OutputDir`, `URLPrefix`, `Extensions`, `HashFiles`, `CSSTransformer`, `JSTransformer`, `ManifestPath`, `EmitSourceMaps`, `IncludeBuildStats`, `PruneUnreferenced` |
| `New(cfg Config) *Builder` | Construct without filling defaults |
| `NewWithDefaults(cfg Config) *Builder` | Construct with sane defaults for empty fields |
| `(*Builder).Build() (*Manifest, error)` | One-shot build; serialized via internal mutex |
| `(*Builder).Watch(ctx, WatchHandlers) error` | Initial build + polling watcher |
| `(*Builder).Handler() http.Handler` | Path-safe file server for `OutputDir` |
| `(*Builder).Route() (pattern string, h http.Handler)` | Pre-wrapped with `http.StripPrefix` |
| `(*Builder).SetReferencedNameProvider(func() map[string]struct{})` | Wire in `Engine.ReferencedAssets` for pruning |
| `(*Builder).Config() Config` | Read back the builder's config |
| `Transformer` | `Transform(src []byte, mediaType string) ([]byte, error)` |
| `NoopTransformer` | Passthrough `Transformer` |
| `Manifest` | Logical-to-URL map with concurrent-safe reads |
| `(*Manifest).Resolve(string) (string, bool)` | Satisfies `AssetResolver` via method value |
| `(*Manifest).Entries() map[string]string` | Copy of the canonical map |
| `(*Manifest).Sources()` / `.Stats()` | Optional sibling data |
| `(*Manifest).Set`, `.SetSource`, `.SetStats`, `.Delete` | Mutation API for hand-assembled manifests |
| `(*Manifest).Save(path string) error` | Atomic write (temp + rename) |
| `LoadManifest(path string) (*Manifest, error)` | Auto-detects structured + legacy bare-map formats |
| `BuildStats` | `{DurationMs, InputBytes, OutputBytes, Ratio}` |
| `WatchHandlers` | `{OnChange, OnError, OnEvent}` |
| `Event`, `EventType` | `EventDiscovered`, `EventBuilt`, `EventSkipped`, `EventPruned`, `EventError` |

### Optional sub-package: `pkg/grove/assets/minify`

```go
import "github.com/wispberry-tech/grove/pkg/grove/assets/minify"
```

| Symbol | Purpose |
|--------|---------|
| `New() *MinifyTransformer` | Construct (registers `text/css` + `application/javascript` minifiers) |
| `(*MinifyTransformer).Transform([]byte, string) ([]byte, error)` | Satisfies `assets.Transformer` |

This sub-package pulls in `github.com/tdewolff/minify/v2`. The core
`pkg/grove/assets` package has no external dependencies.

See [Asset Pipeline](asset-pipeline.md) for usage guides, watch mode, and
pruning semantics.

## Filter Types

```go
type FilterFn func(v Value, args []Value) (Value, error)
```

The function signature for filter implementations. `v` is the piped value, `args` are any arguments passed in parentheses.

```go
type FilterDef struct { /* ... */ }
```

A filter with metadata. Create with `FilterFunc`:

```go
func FilterFunc(fn FilterFn, opts ...FilterOption) *FilterDef
```

```go
func FilterOutputsHTML() FilterOption
```

Marks a filter as returning trusted HTML, which bypasses auto-escaping.

```go
type FilterSet map[string]any
```

A named collection of filters for bulk registration.

## Value Types

```go
type Value /* opaque runtime value */
```

The template runtime value type. Used in filter functions.

```go
var Nil Value // zero value (nil)
```

```go
func StringValue(s string) Value
```

Wraps a Go string as a template `Value`.

```go
func SafeHTMLValue(s string) Value
```

Wraps trusted HTML as a `Value` — auto-escaping is skipped when this value is output.

```go
func ArgInt(args []Value, i, def int) int
```

Helper for filter implementations. Returns `args[i]` as an integer, or `def` if `i` is out of range.

## Error Types

### ParseError

```go
type ParseError struct {
    Template string
    Line     int
    Column   int
    // ...
}
```

Returned for syntax errors detected during compilation. `Template` is the template name (or empty for inline templates). `Line` and `Column` identify the source location.

### RuntimeError

```go
type RuntimeError struct {
    // ...
}
```

Returned for errors during template execution: division by zero, missing required props, strict mode undefined variables, sandbox loop limit exceeded.

Both error types implement the `error` interface. Use `errors.As` for type checking:

```go
var pe grove.ParseError
if errors.As(err, &pe) {
    fmt.Printf("Syntax error at line %d\n", pe.Line)
}
```
