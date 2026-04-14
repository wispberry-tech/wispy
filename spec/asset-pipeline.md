# Grove Asset Pipeline

**Date:** 2026-04-14
**Status:** Landed 2026-04-14
**Scope:** New opt-in asset build system -- co-located CSS/JS processing, content hashing, manifest-based URL resolution

## Motivation

Grove templates declare CSS/JS assets via `{% asset %}` tags. Assets accumulate into `RenderResult` during rendering, and the consuming app is responsible for assembling them into the final HTML document. Currently:

1. **No processing.** Co-located CSS/JS files are served raw -- no minification, no optimization.
2. **No cache-busting.** File paths are static. Browser caches must be manually invalidated on deploy.
3. **Awkward serving.** Apps need multiple static file server routes (`/css/*`, `/js/*`) with extension filters to prevent `.grov` source leaks.
4. **Manual HTML assembly.** `strings.Replace` on placeholder comments (`<!-- HEAD_ASSETS -->`) to inject collected assets.

The asset pipeline solves these problems while remaining fully opt-in. Apps that use Grove for string-only template rendering (no filesystem, no assets) pay zero cost.

## Design Principles

1. **Opt-in, zero cost when unused.** No new dependencies, no new code paths execute unless the app imports `pkg/grove/assets` and configures a builder.
2. **Logical asset names.** Templates reference assets by logical path relative to the views directory, not by URL. The pipeline resolves logical names to served URLs.
3. **Manifest-driven.** A JSON manifest maps logical names to content-hashed URLs. The engine resolves asset sources through the manifest at render time.
4. **Pluggable transforms.** A `Transformer` interface allows swapping minifiers, preprocessors, or external tools. Built-in: noop and tdewolff/minify (pure Go).
5. **Dev and prod modes.** Watch mode for development (rebuild on change), one-shot build for production.

## Package Structure

```
pkg/grove/assets/
    builder.go          Builder, Config, New(), Build(), Watch()
    manifest.go         Manifest type, Resolve(), JSON marshal/unmarshal
    transformer.go      Transformer interface, NoopTransformer
    serve.go            HTTP handler for dist directory
    minify/
        minify.go       MinifyTransformer wrapping tdewolff/minify
```

The `assets` package is a sibling of the main `grove` package. Importing it does not affect apps that don't use it. The `minify` sub-package is a separate import -- apps that don't want the tdewolff/minify dependency can use `NoopTransformer` or implement their own.

## Logical Asset Names

Templates use logical names -- file paths relative to the views directory:

```
{% asset "primitives/button/button.css" type="stylesheet" %}
{% asset "primitives/button/button.js" type="script" %}
```

These names correspond to actual files on disk:

```
views/
    primitives/
        button/
            button.grov
            button.css      <-- "primitives/button/button.css"
            button.js       <-- "primitives/button/button.js"
```

Without a resolver configured on the engine, the logical name passes through unchanged as the `src` in `RenderResult.Assets`. With a resolver (typically `manifest.Resolve`), the engine substitutes the hashed URL at render time.

### Migration from URL-style paths

Existing templates using URL-style paths:
```
{% asset "/css/primitives/button/button.css" type="stylesheet" %}
```

Must be updated to logical names:
```
{% asset "primitives/button/button.css" type="stylesheet" %}
```

## Asset Manifest

The manifest is a structured JSON document. The canonical field is `assets`, a map from logical name to served URL. Optional sibling fields (`sources`, `stats`) appear when the corresponding Config flags are set.

```json
{
  "assets": {
    "primitives/button/button.css": "/dist/primitives/button/button.a1b2c3d4.css",
    "primitives/button/button.js":  "/dist/primitives/button/button.e5f6a7b8.js",
    "base.css":                     "/dist/base.9f8e7d6c.css"
  }
}
```

Full shape (including optional fields) is documented in [Manifest Format With Optional Fields](#manifest-format-with-optional-fields). `LoadManifest` also accepts the legacy bare-map form (`{logical: url}`) for backward compatibility with external tooling.

### Manifest API

```go
package assets

// Manifest maps logical asset names to served URLs.
type Manifest struct {
    entries map[string]string
}

// Resolve returns the served URL for a logical name.
// Returns (url, true) if found, ("", false) if not.
// When not found, callers should fall back to the original name.
func (m *Manifest) Resolve(logicalName string) (string, bool)

// Entries returns a copy of the manifest map.
func (m *Manifest) Entries() map[string]string

// LoadManifest reads a manifest from a JSON file.
func LoadManifest(path string) (*Manifest, error)

// Save writes the manifest to a JSON file atomically.
// Implementation: write to path+".tmp" then os.Rename. A crash mid-write
// leaves the previous manifest intact; consumers never observe a partial file.
func (m *Manifest) Save(path string) error
```

### Content Hashing

Hashes are computed from the **transformed** (post-minification) file content:

1. Read source file
2. Apply transformer (minify or noop)
3. SHA256 of transformed bytes
4. First 8 hex characters as hash suffix
5. Insert before file extension: `button.css` -> `button.a1b2c3d4.css`

Same content always produces the same hash. Files that haven't changed keep the same hash across builds -- browsers serve from cache.

## Builder API

```go
package assets

// Config controls the asset build pipeline.
type Config struct {
    // SourceDir is the root directory to scan for CSS/JS files.
    // Typically the same directory used as the grove FileSystemStore root.
    SourceDir string

    // OutputDir is where processed files are written.
    // Created automatically if it doesn't exist.
    OutputDir string

    // URLPrefix is prepended to output paths in the manifest.
    // Example: "/dist" produces manifest entries like "/dist/button.a1b2c3d4.css".
    // Default: "/dist"
    URLPrefix string

    // Extensions lists file extensions to process.
    // Default: [".css", ".js"]
    Extensions []string

    // HashFiles controls whether content hashes are inserted into filenames.
    // Default: true
    HashFiles bool

    // CSSTransformer processes CSS files. Default: NoopTransformer.
    CSSTransformer Transformer

    // JSTransformer processes JS files. Default: NoopTransformer.
    JSTransformer Transformer

    // ManifestPath is where to write the manifest JSON file.
    // Empty string means don't write to disk (manifest returned in memory only).
    ManifestPath string

    // EmitSourceMaps controls whether transformers that support source maps
    // (e.g. MinifyTransformer) emit .map files next to transformed output.
    // Manifest gains a sibling "sources" field mapping logical name to map URL.
    // Default: false. Recommended: true in dev, false in prod.
    EmitSourceMaps bool

    // IncludeBuildStats controls whether the manifest includes per-file
    // build statistics (duration, input/output bytes, compression ratio).
    // Default: false. Useful for dev HUDs and CI size-regression checks.
    IncludeBuildStats bool

    // PruneUnreferenced enables the prune pass. After Build(), the builder
    // consults the engine's referenced-name set (populated at render time)
    // and drops manifest entries for files that no template imported during
    // the sampling window. Files on disk in OutputDir are not deleted --
    // only the manifest is pruned.
    //
    // Requires Engine with WithAssetResolver installed. First build cannot
    // prune (no render data yet); subsequent builds in watch mode do.
    // Default: false.
    PruneUnreferenced bool
}
```

```go
// Builder scans, processes, and outputs asset files.
type Builder struct { ... }

// New creates a Builder with the given config.
// Does not perform any I/O until Build() or Watch() is called.
func New(cfg Config) *Builder

// Build scans SourceDir for files matching Extensions, applies transformers,
// writes processed files to OutputDir, and returns the resulting manifest.
//
// Build is safe to call multiple times sequentially. Each call produces a fresh
// manifest and overwrites previously built files in OutputDir.
//
// Build is NOT safe for concurrent invocation on the same Builder. An internal
// mutex serializes concurrent calls: the second caller blocks until the first
// returns. Use a single builder and call Build() from one goroutine.
func (b *Builder) Build() (*Manifest, error)

// WatchHandlers bundles callbacks for Watch mode.
type WatchHandlers struct {
    // OnChange is called after a successful (full or partial) rebuild
    // with the current manifest. Required.
    OnChange func(*Manifest)

    // OnError is called when a file fails to transform. The manifest passed
    // to OnChange reflects a partial swap: failed files keep their prior
    // entry, successful files update. Optional -- nil means errors are only
    // surfaced via OnEvent / Logger.
    OnError func(error)

    // OnEvent receives structured build events (discovered, built, skipped,
    // pruned, error). Optional -- useful for dev tooling, HUD, or piping
    // to the app's structured logger.
    OnEvent func(Event)
}

// Watch performs an initial Build(), then watches SourceDir for file changes.
// Rebuilds are partial: only changed files re-transform. On per-file failure,
// the failed file keeps its previous manifest entry; successful files swap.
// OnChange is called with the resulting manifest after each rebuild cycle.
//
// Watch runs in a background goroutine. Cancel ctx to stop watching.
// Returns an error if the initial build fails.
func (b *Builder) Watch(ctx context.Context, h WatchHandlers) error

// Handler returns an http.Handler that serves files from OutputDir.
// For hashed files, it sets Cache-Control: public, max-age=31536000, immutable.
// For non-hashed files, it sets standard caching headers. See HTTP Handler
// section for path-safety rules.
func (b *Builder) Handler() http.Handler

// Route returns the (pattern, handler) pair for mounting under URLPrefix.
// Equivalent to ("{URLPrefix}/", http.StripPrefix("{URLPrefix}/", b.Handler())).
// Use as: mux.Handle(builder.Route())
func (b *Builder) Route() (pattern string, handler http.Handler)

// SetReferencedNameProvider wires in a getter for the set of logical asset
// names that have been referenced during template rendering. Required when
// Config.PruneUnreferenced is true; ignored otherwise.
//
// The provider is typically Engine.ReferencedAssets (see Engine Integration).
// Safe to call at any time; read on each Build() invocation.
func (b *Builder) SetReferencedNameProvider(fn func() map[string]struct{})
```

## Transformer Interface

```go
// Transformer processes raw asset bytes.
// mediaType is "text/css" or "application/javascript".
type Transformer interface {
    Transform(src []byte, mediaType string) ([]byte, error)
}

// NoopTransformer returns input unchanged. This is the default.
type NoopTransformer struct{}

func (NoopTransformer) Transform(src []byte, _ string) ([]byte, error) {
    return src, nil
}
```

### Built-in: MinifyTransformer

Located in `pkg/grove/assets/minify/` (separate import to isolate the tdewolff/minify dependency):

```go
package minify

import "github.com/wispberry-tech/grove/pkg/grove/assets"

// MinifyTransformer uses tdewolff/minify for CSS and JS minification.
type MinifyTransformer struct { ... }

// New creates a MinifyTransformer.
func New() *MinifyTransformer

// Satisfies assets.Transformer
func (m *MinifyTransformer) Transform(src []byte, mediaType string) ([]byte, error)
```

### Custom Transformers

Users can implement `Transformer` to plug in any tool:

```go
// Example: esbuild transformer (user-provided)
type EsbuildTransformer struct{}

func (t EsbuildTransformer) Transform(src []byte, mediaType string) ([]byte, error) {
    // Shell out to esbuild binary
    cmd := exec.Command("esbuild", "--minify", "--loader=css")
    cmd.Stdin = bytes.NewReader(src)
    return cmd.Output()
}
```

## Engine Integration

Two additions to the Engine API. Both are no-ops when no resolver is configured.

The engine takes a **resolver function**, not a `*Manifest`. This keeps the core engine decoupled from the `assets` package (engine never imports it), matches the existing `Store` interface pattern, and lets users plug in CDN lookups, A/B variants, test fixtures, or any other logical-to-URL mapping without subclassing a Manifest type.

### Resolver Type

```go
// AssetResolver maps a logical asset name to a served URL.
// Returns (url, true) if resolved, ("", false) to fall through to the
// original name. A nil resolver is treated as pass-through.
type AssetResolver func(logicalName string) (string, bool)
```

`*assets.Manifest` satisfies this shape via its `Resolve` method -- callers pass `manifest.Resolve` as a method value.

### Option: WithAssetResolver

```go
// WithAssetResolver configures the engine to resolve asset URLs through
// the given function at render time.
func WithAssetResolver(r AssetResolver) Option
```

### Method: SetAssetResolver

```go
// SetAssetResolver atomically swaps the asset resolver.
// Safe for concurrent use. Designed for watch mode where the resolver
// updates on file changes while the engine continues serving requests.
func (e *Engine) SetAssetResolver(r AssetResolver)
```

### VM Change: OP_ASSET Resolution

In `internal/vm/vm.go`, the `OP_ASSET` handler gains a resolver call before storing the asset:

```go
// Resolve logical name (no-op if resolver is nil)
resolvedSrc := src
if resolve := v.eng.AssetResolver(); resolve != nil {
    if hashed, ok := resolve(src); ok {
        resolvedSrc = hashed
    }
}
// Deduplication uses resolvedSrc
if !v.rc.seenSrc[resolvedSrc] {
    v.rc.seenSrc[resolvedSrc] = true
    v.rc.assets = append(v.rc.assets, assetEntry{
        src: resolvedSrc,
        // ...
    })
}
```

When `AssetResolver()` returns nil (the default), this is a single nil check per asset -- effectively zero cost.

### EngineIface Addition

```go
type EngineIface interface {
    // ... existing methods ...

    // AssetResolver returns the configured resolver, or nil when unused.
    AssetResolver() AssetResolver

    // RecordAssetRef records a logical asset name seen during rendering.
    // No-op when the engine has no resolver configured (avoids allocating
    // the internal set for apps that don't use the pipeline).
    RecordAssetRef(logicalName string)
}
```

### Engine Methods for Referenced-Name Tracking

```go
// ReferencedAssets returns a snapshot of logical names seen via OP_ASSET
// since the engine started (or since ResetReferencedAssets was called).
// The returned map is a copy -- safe to mutate.
func (e *Engine) ReferencedAssets() map[string]struct{}

// ResetReferencedAssets clears the referenced-name set. Useful to scope
// prune passes to a specific time window (e.g. "names seen since last build").
func (e *Engine) ResetReferencedAssets()
```

The internal set uses `sync.Map` or a mutex-guarded `map[string]struct{}`. It is only allocated when a resolver is configured -- apps that never call `WithAssetResolver` pay nothing. See [Pruning Unreferenced Assets](#pruning-unreferenced-assets) for the full mechanism.

## Usage Examples

### Production Setup

```go
package main

import (
    "github.com/wispberry-tech/grove/pkg/grove"
    "github.com/wispberry-tech/grove/pkg/grove/assets"
    "github.com/wispberry-tech/grove/pkg/grove/assets/minify"
)

func main() {
    minifier := minify.New()

    builder := assets.New(assets.Config{
        SourceDir:      "views/",
        OutputDir:      "dist/",
        URLPrefix:      "/dist",
        CSSTransformer: minifier,
        JSTransformer:  minifier,
        ManifestPath:   "dist/manifest.json",
    })

    manifest, err := builder.Build()
    if err != nil {
        log.Fatal(err)
    }

    fs := grove.NewFileSystemStore("views/")
    engine := grove.New(
        grove.WithStore(fs),
        grove.WithAssetResolver(manifest.Resolve),
    )

    r := http.NewServeMux()
    r.Handle("/dist/", http.StripPrefix("/dist/", builder.Handler()))

    // Render a page -- asset URLs are automatically resolved to hashed paths
    result, _ := engine.Render(ctx, "index", grove.Data{"title": "Home"})
    // result.HeadHTML() outputs:
    //   <link rel="stylesheet" href="/dist/base.9f8e7d6c.css">
    //   <link rel="stylesheet" href="/dist/primitives/button/button.a1b2c3d4.css">
}
```

### Development Setup (Watch Mode)

```go
func main() {
    builder := assets.New(assets.Config{
        SourceDir: "views/",
        OutputDir: "dist/",
        URLPrefix: "/dist",
        HashFiles: false, // no hashing in dev -- easier debugging
    })

    manifest, err := builder.Build()
    if err != nil {
        log.Fatal(err)
    }

    engine := grove.New(
        grove.WithStore(grove.NewFileSystemStore("views/")),
        grove.WithAssetResolver(manifest.Resolve),
    )

    // Watch for changes -- swap resolver atomically on rebuild
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    builder.Watch(ctx, assets.WatchHandlers{
        OnChange: func(m *assets.Manifest) {
            engine.SetAssetResolver(m.Resolve)
            log.Println("assets rebuilt")
        },
        OnError: func(err error) {
            log.Printf("asset rebuild failed: %v", err)
        },
    })

    // ... start HTTP server ...
}
```

### No Pipeline (Current Behavior, Unchanged)

```go
func main() {
    engine := grove.New(grove.WithStore(grove.NewFileSystemStore("views/")))

    // No manifest -- asset src values pass through as-is
    result, _ := engine.Render(ctx, "index", grove.Data{})
    // result.HeadHTML() outputs whatever src the template declared
}
```

## File Processing Pipeline

For each file in SourceDir matching configured extensions:

```
1. Discover     Walk SourceDir, collect files by extension
2. Read         Read file bytes from disk
3. Transform    Apply CSSTransformer or JSTransformer based on extension
4. Hash         SHA256(transformed bytes) -> first 8 hex chars
5. Write        Write to OutputDir, preserving relative path structure
                button.css -> button.a1b2c3d4.css (if HashFiles=true)
                button.css -> button.css           (if HashFiles=false)
6. Manifest     Record mapping: logical name -> URL prefix + output filename
```

### Watch Mode Behavior

- Uses polling (500ms `time.Ticker`, per-file `os.Stat` mtime compare) — no fsnotify dependency. See `watch.go`.
- On file change: rebuild only changed files
- Debounce rapid changes (100ms window)
- **Partial swap on failure:** if a file fails to transform (syntax error, I/O error), it keeps its previous manifest entry. Successful files update. `OnError` fires for each failure. `OnChange` still fires with the partially-updated manifest so the working files reflect latest changes
- If the initial build fails, `Watch` returns an error before entering the watch loop

### Build Events

```go
// EventType classifies a build lifecycle event.
type EventType int

const (
    EventDiscovered EventType = iota // File found during scan
    EventBuilt                       // File transformed and written
    EventSkipped                     // Unchanged (cache hit)
    EventPruned                      // Entry removed by prune pass
    EventError                       // Transform or I/O failure
)

// Event is a single build lifecycle record.
type Event struct {
    Type       EventType
    LogicalName string
    OutputPath string        // Absolute path written (Built only)
    Duration   time.Duration // Build duration (Built only)
    InputSize  int           // Bytes before transform (Built only)
    OutputSize int           // Bytes after transform (Built only)
    Err        error         // Non-nil for EventError
}
```

## Pruning Unreferenced Assets

Filesystem discovery can pick up orphan files that no template references. `PruneUnreferenced: true` drops them from the manifest.

**Mechanism:**

1. VM calls `Engine.RecordAssetRef(logicalName)` inside OP_ASSET (before resolution). The engine stores it in a mutex-guarded set, allocated lazily only if a resolver is configured.
2. App wires the engine's getter into the builder once at startup: `builder.SetReferencedNameProvider(engine.ReferencedAssets)`.
3. After each `Build()`, if `PruneUnreferenced` is set and the provider returns a non-empty set, entries not in the set are removed from the manifest before it is returned.

**Caveats:**

- First build has no render data -- nothing is pruned until at least one render has happened. Use watch mode for pruning benefit in dev; run a warmup render pass in prod if needed.
- Files on disk in `OutputDir` are **not** deleted by prune -- only manifest entries. Users can clean `OutputDir` between builds if they want disk hygiene.
- Conditionally-imported assets (behind `{% #if %}`) may appear orphaned if that branch never rendered during sampling. Document this: prune is best-effort, template-driven discovery (future work) is authoritative.

## Manifest Format With Optional Fields

When `EmitSourceMaps` or `IncludeBuildStats` is enabled, the manifest gains companion top-level fields alongside the canonical `assets` map.

```json
{
  "assets": {
    "primitives/button/button.css": "/dist/primitives/button/button.a1b2c3d4.css"
  },
  "sources": {
    "primitives/button/button.css": "/dist/primitives/button/button.a1b2c3d4.css.map"
  },
  "stats": {
    "primitives/button/button.css": {
      "duration_ms": 3,
      "input_bytes": 1842,
      "output_bytes": 1109,
      "ratio": 0.60
    }
  }
}
```

`LoadManifest` reads whichever fields are present; `Save` writes only fields that have data. The legacy bare-map form (`{logical: url}` with no `assets` wrapper) is also accepted -- `LoadManifest` auto-detects which format is on disk by probing for the `assets` key.

## Logger Integration

For users who want `log/slog` or their existing logger, wire it through `OnEvent`:

```go
builder.Watch(ctx, assets.WatchHandlers{
    OnChange: func(m *assets.Manifest) { engine.SetAssetResolver(m.Resolve) },
    OnEvent: func(e assets.Event) {
        switch e.Type {
        case assets.EventBuilt:
            slog.Info("asset built",
                "name", e.LogicalName,
                "bytes", e.OutputSize,
                "ratio", float64(e.OutputSize)/float64(e.InputSize))
        case assets.EventError:
            slog.Error("asset failed", "name", e.LogicalName, "err", e.Err)
        }
    },
})
```

## HTTP Handler

`builder.Handler()` returns an `http.Handler` for the dist directory with smart caching:

- **Hashed files** (contain 8-char hex before extension): `Cache-Control: public, max-age=31536000, immutable`
- **Non-hashed files**: `Cache-Control: public, max-age=0, must-revalidate` with `ETag` support
- Serves `Content-Type` based on file extension
- Returns 404 for files not in dist directory

### Path Safety

The handler MUST defend against path traversal. Implementation rules:

1. Reject any request path containing `..` or null bytes before further processing.
2. Resolve the final absolute path via `filepath.Join(outputDirAbs, filepath.Clean("/" + reqPath))`, then verify it has `outputDirAbs` as prefix (`strings.HasPrefix(clean, outputDirAbs+string(os.PathSeparator))`). Mismatch -> 404.
3. Follow no symlinks out of `OutputDir` (`filepath.EvalSymlinks` + prefix check), or reject symlinks outright.
4. Set `X-Content-Type-Options: nosniff` on all responses.

`http.ServeFile` is NOT used directly -- its traversal protection relies on the caller having already cleaned the path, and it can leak metadata for directories. The handler wraps a safer custom lookup.

### URLPrefix / OutputDir Contract

`URLPrefix` and `OutputDir` are independent values with one binding rule: **the HTTP route where `builder.Handler()` is mounted must strip `URLPrefix` before the handler sees the path.**

```go
// URLPrefix: "/dist" -- manifest URLs look like /dist/button.a1b2c3d4.css
// OutputDir: "dist/" -- files written here
// Route must strip /dist/ so handler sees "button.a1b2c3d4.css"
r.Handle("/dist/", http.StripPrefix("/dist/", builder.Handler()))
```

Mismatch (e.g. mounting handler at `/assets/` while `URLPrefix: "/dist"`) produces 404s for every asset since manifest URLs point at `/dist/*` but no route serves that prefix. `Builder.Handler()` does not validate this -- it is the app's responsibility to wire them consistently.

For convenience, `Builder.Route() (pattern, handler)` returns both so `http.ServeMux` wiring is one line:

```go
r.Handle(builder.Route()) // equivalent to Handle("/dist/", StripPrefix("/dist/", handler))
```

## Non-Goals

These are explicitly out of scope for this spec. Some may be addressed in future phases.

- **Bundling** -- combining multiple CSS/JS files into single bundles. Treated as a future optional optimization, not a requirement. Individually hashed files with HTTP/2 and browser caching handle most workloads fine.
- **TypeScript compilation** -- users can implement a Transformer that shells out to `tsc` or `esbuild`.
- **CSS preprocessors** (Sass, Less, PostCSS) -- pluggable via Transformer interface.
- **Module graph analysis / tree-shaking** -- requires understanding JS imports, far beyond current scope.
- **Full source map tooling** -- basic `.map` emission is in-scope via `EmitSourceMaps` (transformers that support it write sibling files, referenced from the manifest). Deep tooling (cross-file source map merging, chained preprocessor maps, remote map serving strategy) is future work.
- **CDN integration** -- manifest URLs could point to a CDN, but the pipeline doesn't upload files.
- **Image optimization** -- different domain (lossy compression, format conversion). Separate tool.
- **Hot module replacement (HMR)** -- dev convenience, but requires WebSocket server and client-side JS. Out of scope.

## Future Phases

### Phase 2: CSS Bundling (Optional Optimization)

**Bundling is an optimization, not a requirement.** Loading multiple small, content-hashed files works fine for most apps -- browser caches them individually, HTTP/2 multiplexes requests, and unchanged files skip network entirely on repeat visits. Only bundle when profiling shows request overhead is a real bottleneck.

When it becomes worthwhile, analyze the template component graph to bundle CSS:

- `{% import Button from "primitives/button" %}` means the page needs `button.css`
- Collect all CSS dependencies for a page render
- Concatenate in priority order into a single `page.{hash}.css`
- Trade-off: fewer requests vs. larger cache invalidation blast radius (one component change busts whole bundle)

### Phase 3: JS Bundling (Optional Optimization)

Same optional status as CSS bundling. More complex:

- Module format (ESM vs IIFE)
- Dependency deduplication
- Load order matters more than CSS

### Phase 4: Advanced Transforms

- PostCSS plugin support via Transformer
- Tailwind CSS integration
- CSS custom property inlining for legacy browser support

## Dependencies

### Core (`pkg/grove/assets/`)

No new external dependencies. Uses only the Go standard library.

### Minify sub-package (`pkg/grove/assets/minify/`)

- `github.com/tdewolff/minify/v2` -- CSS and JS minification
- `github.com/tdewolff/parse/v2` -- transitive dependency of minify

These dependencies are only pulled in when the consuming app imports the `minify` sub-package.

## Files to Create

| File | Contents |
|------|----------|
| `pkg/grove/assets/builder.go` | `Builder`, `Config`, `New()`, `Build()`, `Watch()`, `WatchHandlers`, `Event`, `EventType`, `SetReferencedNameProvider()`, `Route()` |
| `pkg/grove/assets/manifest.go` | `Manifest`, `Resolve()`, `Entries()`, `LoadManifest()` (auto-detect legacy form), `Save()` (atomic write) |
| `pkg/grove/assets/transformer.go` | `Transformer` interface, `NoopTransformer` |
| `pkg/grove/assets/serve.go` | `Handler()` method on `Builder` (path-safe file serving) |
| `pkg/grove/assets/minify/minify.go` | `MinifyTransformer` wrapping tdewolff/minify (CSS + JS, optional source maps) |

## Files to Modify

| File | Change |
|------|--------|
| `pkg/grove/engine.go` | Add `AssetResolver` type, `WithAssetResolver()` option, `SetAssetResolver()`, `AssetResolver()`, `RecordAssetRef()`, `ReferencedAssets()`, `ResetReferencedAssets()`; internal resolver field + lazy referenced-name set |
| `internal/vm/vm.go` | `OP_ASSET`: call `v.eng.RecordAssetRef(src)`, then resolve via `v.eng.AssetResolver()` before storing |
| `internal/vm/value.go` | `EngineIface`: add `AssetResolver()` and `RecordAssetRef(string)` methods |
| `go.mod` | No change for core; `minify` sub-package adds tdewolff deps |
