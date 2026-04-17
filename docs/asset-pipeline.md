# Asset Pipeline

Grove ships an optional asset pipeline under `pkg/grove/assets` that turns
colocated component CSS/JS into content-hashed, optionally-minified static
files and wires them into the engine through a resolver function.

Everything in this page is **opt-in**. Apps that don't import
`pkg/grove/assets` pay no extra cost — `{% asset "..." %}` tags still pass
through to `RenderResult` with the literal string you wrote.

## The big picture

```
templates/                        dist/
├── composites/nav/nav.css   ->   composites/nav/nav.a1b2c3d4.css
├── primitives/button/           
│   ├── button.css           ->   primitives/button/button.e5f6a7b8.css
│   └── button.js            ->   primitives/button/button.9a8b7c6d.js
                                   manifest.json
```

1. A `Builder` scans `SourceDir`, runs each file through a `Transformer`
   (`NoopTransformer` by default; `minify.New()` for production), hashes the
   output with SHA-256, writes `{stem}.{hash8}.{ext}` to `OutputDir`, and
   records the mapping in a `Manifest` (+ optional `manifest.json`).
2. You pass `manifest.Resolve` to `grove.WithAssetResolver`. At render time
   the VM looks up each `{% asset "..." %}` logical name through the
   resolver and substitutes the hashed URL before the asset is added to
   `RenderResult.Assets`.
3. `builder.Route()` returns a path-safe HTTP handler that serves files
   from `OutputDir` with `Cache-Control: immutable` on anything whose
   filename matches `\.[0-9a-f]{8}\.[^.]+$`.

## Logical names

Templates reference assets by **logical name** — a relative path from
`SourceDir`:

```grov
{# primitives/button/button.grov #}
{% asset "primitives/button/button.css" type="stylesheet" %}
{% asset "primitives/button/button.js"  type="script" %}
<a href="{% href %}" class="btn">{% label %}</a>
```

With a resolver configured, `primitives/button/button.css` becomes
`/dist/primitives/button/button.a1b2c3d4.css`. Without a resolver, it
passes through unchanged and you can point it at a plain static handler.

URL-style names (anything starting with `/`, `http://`, etc.) that are
**not in the manifest** pass through as-is. This is the designed escape
hatch for hand-managed globals:

```grov
{% asset "/static/base.css" type="stylesheet" priority=10 %}
```

## Minimal production setup

```go
import (
    "github.com/wispberry-tech/grove/pkg/grove"
    "github.com/wispberry-tech/grove/pkg/grove/assets"
    "github.com/wispberry-tech/grove/pkg/grove/assets/minify"
)

builder := assets.NewWithDefaults(assets.Config{
    SourceDir:      "templates",
    OutputDir:      "dist",
    URLPrefix:      "/dist",
    CSSTransformer: minify.New(),
    JSTransformer:  minify.New(),
    ManifestPath:   "dist/manifest.json",
})
manifest, err := builder.Build()
if err != nil {
    log.Fatal(err)
}

eng := grove.New(
    grove.WithStore(grove.NewFileSystemStore("templates")),
    grove.WithAssetResolver(manifest.Resolve),
)

mux := http.NewServeMux()
pattern, handler := builder.Route()
mux.Handle(pattern+"*", handler)
```

`NewWithDefaults` fills in `Extensions=[".css", ".js"]`, `HashFiles=true`,
`URLPrefix="/dist"`, and `NoopTransformer`s if left unset.

## Watch mode (development)

`Builder.Watch(ctx, handlers)` runs the initial build, then polls
`SourceDir` every 500 ms and rebuilds changed files with a 100 ms debounce.
On each successful (partial or full) rebuild it calls `OnChange` with the
updated `*Manifest`:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go builder.Watch(ctx, assets.WatchHandlers{
    OnChange: func(m *assets.Manifest) {
        eng.SetAssetResolver(m.Resolve) // atomic swap; safe vs concurrent renders
    },
    OnError: func(err error) {
        log.Printf("asset rebuild failed: %v", err)
    },
    OnEvent: func(e assets.Event) {
        // EventBuilt / EventSkipped / EventError / EventDiscovered / EventPruned
    },
})
```

**Partial swap semantics:** if one file fails to transform, it keeps its
previous manifest entry while successful files update. `OnError` fires
per failure; `OnChange` still fires with the partially-updated manifest.

## Pruning unreferenced assets

A filesystem scan can pick up CSS/JS that no template actually uses. Set
`Config.PruneUnreferenced: true` and wire the engine's referenced-name
tracker into the builder:

```go
builder.SetReferencedNameProvider(eng.ReferencedAssets)
```

After every `Build()`, entries whose logical name was never seen by the VM
are dropped from the returned manifest (the files on disk are left alone —
clean `OutputDir` yourself if you want disk hygiene).

Caveats:
- The first build has no render data, so nothing is pruned until at least
  one render has happened. Run a warm-up render in prod or rely on watch
  mode in dev.
- Assets behind conditional branches (`{% #if %}`) may appear orphaned if
  that branch never rendered during sampling. Prune is best-effort.

## Manifest format

Canonical shape:

```json
{
  "assets": {
    "primitives/button/button.css": "/dist/primitives/button/button.a1b2c3d4.css"
  },
  "sources": { "...": "..." },
  "stats":   { "...": {"duration_ms": 3, "input_bytes": 1842, "output_bytes": 1109, "ratio": 0.60} }
}
```

`sources` appears when `EmitSourceMaps` is set; `stats` when
`IncludeBuildStats` is set. `LoadManifest` also reads the legacy bare-map
form `{"logical": "url"}` for compatibility with third-party tooling.

`Manifest.Save` writes atomically (write-temp + rename), so crashes
mid-write leave the previous manifest intact.

## Custom transformers

Implement the two-method interface to plug in any tool:

```go
type Transformer interface {
    Transform(src []byte, mediaType string) ([]byte, error)
}
```

`mediaType` is `"text/css"` for `.css` and `"application/javascript"` for
`.js`. Return `(src, nil)` unchanged for a passthrough. Example wrapping
`esbuild`:

```go
type Esbuild struct{}

func (Esbuild) Transform(src []byte, mediaType string) ([]byte, error) {
    cmd := exec.Command("esbuild", "--minify", "--loader=css")
    cmd.Stdin = bytes.NewReader(src)
    return cmd.Output()
}
```

## HTTP handler

`Builder.Handler()` returns an `http.Handler` that serves `OutputDir`:

- Rejects `..` and null bytes before any filesystem call.
- `filepath.Clean` + prefix check against `OutputDir` to block traversal.
- `filepath.EvalSymlinks` to reject escape via symlink.
- Hashed filenames (regex `\.[0-9a-f]{8}\.[^.]+$`) get
  `Cache-Control: public, max-age=31536000, immutable`.
- Non-hashed files get an `ETag` and `Cache-Control: must-revalidate`.
- `X-Content-Type-Options: nosniff` on every response.
- `Content-Type` set from file extension.

`Builder.Route() (pattern, handler)` pre-wraps the handler with
`http.StripPrefix(URLPrefix+"/", ...)`. Mount it on any router:

```go
pattern, handler := builder.Route()
mux.Handle(pattern+"*", handler)       // chi, gorilla
// or:
http.Handle(pattern, handler)          // net/http mux (trailing slash)
```

## Engine API summary

| Symbol | Purpose |
|--------|---------|
| `grove.AssetResolver` | `func(logicalName string) (string, bool)` |
| `grove.WithAssetResolver(r AssetResolver) Option` | Install resolver on `grove.New` |
| `(*Engine).SetAssetResolver(r)` | Atomic swap at runtime (watch mode) |
| `(*Engine).AssetResolver() AssetResolver` | Getter; `nil` when unused |
| `(*Engine).ReferencedAssets() map[string]struct{}` | Snapshot for prune / debugging |
| `(*Engine).ResetReferencedAssets()` | Clear tracker |
| `(*Engine).RecordAssetRef(name string)` | Internal; called by OP_ASSET |

## Minify sub-package

`pkg/grove/assets/minify` wraps `github.com/tdewolff/minify/v2`. Import it
only when you want minification — the main `pkg/grove/assets` has no
external dependencies.

```go
import "github.com/wispberry-tech/grove/pkg/grove/assets/minify"

t := minify.New()
out, err := t.Transform(src, "text/css")
```

Unknown media types return an explicit error rather than passing through
silently, so misconfigured pipelines fail loudly.

## ESM modules

The pipeline supports loading fingerprinted JavaScript as ES modules via
a browser-native importmap. No bundling; no JS parsing. Two pieces:

1. **`{% asset "app.js" type="module" %}`** — `FootHTML()` emits
   `<script type="module" src="...">` for assets whose `type` is
   `"module"`. Classic `"script"` assets are emitted first, modules
   after.
2. **`pkg/grove/assets/esm`** (opt-in subpackage) — `esm.Importmap(manifest, opts)`
   builds a `<script type="importmap">` block from the manifest's JS
   entries. Inject the returned string as global template data and emit
   via the `safe` filter.

```go
import "github.com/wispberry-tech/grove/pkg/grove/assets/esm"

importmap := esm.Importmap(manifest, esm.Options{
    StripJSExt: true, // also emit "foo/bar" alongside "foo/bar.js"
    Extra: map[string]string{
        "htmx.org": "https://esm.sh/htmx.org@2.0.0",
    },
})

eng := grove.New(
    grove.WithAssetResolver(manifest.Resolve),
    grove.WithGlobalData(map[string]any{"importmap": importmap}),
)
```

```grov
<head>
  {{ importmap | safe }}
</head>
<body>
  {% slot %}
  {% asset "app/main.js" type="module" %}
</body>
```

**Important limit:** the importmap resolves **bare specifiers**
(`import x from "app/util"`) to hashed URLs. It does NOT resolve
**relative imports** (`import x from "./util.js"`) inside fingerprinted
files — the browser resolves relative specifiers against the current
module's URL before consulting the importmap, so the hashed sibling is
never found. Author modules using bare specifiers keyed on logical
names.

Full rationale, examples, and future work: [`spec/esm-support.md`](spec/esm-support.md).

## Non-goals

The pipeline is intentionally small. **Not included** (may be future
work — see [`spec/asset-pipeline.md`](../spec/asset-pipeline.md) §Non-Goals):
bundling, TypeScript compilation, PostCSS/Sass, tree-shaking, CDN upload,
image optimization, HMR.

## See also

- [Web Primitives](web-primitives.md) — the underlying `{% asset %}` tag
  and how `RenderResult.HeadHTML()` / `FootHTML()` consume it
- [API Reference](api-reference.md#asset-pipeline) — full Go API
- [`spec/asset-pipeline.md`](../spec/asset-pipeline.md) — design rationale
  and historical context
- [`examples/juicebar`](../examples/juicebar) — working integration with
  minified component assets, a hashed manifest, and a custom `currency`
  filter
