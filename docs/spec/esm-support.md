# ESM Support via the Asset Pipeline

**Date:** 2026-04-16
**Status:** Proposed
**Scope:** Add ES module (ESM) loading to Grove's asset pipeline — `<script type="module">` emission and importmap generation for fingerprinted URLs. No bundling, no JS parsing.

## Motivation

Grove's asset pipeline currently emits classic `<script src="…"></script>` tags and produces a flat `logical-name → fingerprinted-URL` manifest. Two gaps prevent authors from using modern ESM workflows:

1. **No module script tags.** `{% asset "app.js" type="script" %}` always emits a classic `<script>`. There is no way to produce `<script type="module" src="…">`.
2. **Fingerprinting breaks imports.** When `app.js` is built as `app.abc12345.js`, any `import … from "app.js"` in user code fails — the browser looks up the literal string, not the hashed URL.

A browser-native importmap (spec: [HTML Living Standard § importmap](https://html.spec.whatwg.org/multipage/webappapis.html#import-maps)) solves (2) without a bundler: the browser consults the importmap when resolving module specifiers and substitutes the hashed URL.

## Goals / Non-goals

**Goals**

- Emit `<script type="module" src="…">` from `{% asset %}` when authors opt in.
- Provide a pure-Go helper that turns a `*assets.Manifest` into an `<script type="importmap">` block.
- Keep the core `pkg/grove/assets/` package zero-dep; ship the importmap helper as an opt-in subpackage (`pkg/grove/assets/esm/`) mirroring `assets/minify/`.
- Dev and prod behave identically — one code path.

**Non-goals**

- No JavaScript parsing. No import-statement discovery. No dependency graph. No chunking/bundling.
- No automatic rewriting of relative imports in source files (see [Limitations](#limitations)).
- No sub-resource integrity (`integrity=…`) attribute. Could be layered later on top of the manifest.
- No preload/modulepreload auto-emission.

## Does this let me import JS modules "without issue"?

Short answer: **yes for bare specifiers, no for relative imports in fingerprinted files.** Read this section before planning your author experience.

Given a manifest:

```json
{ "assets": {
    "app/main.js":  "/static/app/main.abc12345.js",
    "app/util.js":  "/static/app/util.def67890.js"
}}
```

and an importmap generated from it:

```html
<script type="importmap">
{ "imports": {
    "app/main.js": "/static/app/main.abc12345.js",
    "app/main":    "/static/app/main.abc12345.js",
    "app/util.js": "/static/app/util.def67890.js",
    "app/util":    "/static/app/util.def67890.js"
}}
</script>
```

**Works:**

```js
// inside app/main.js (author source — pre-fingerprint)
import { clamp } from "app/util";        // ✅ bare specifier → importmap → hashed URL
import { clamp } from "app/util.js";     // ✅ same, with .js
```

**Does NOT work:**

```js
// inside app/main.js
import { clamp } from "./util.js";       // ❌ browser resolves relative to
                                         //    /static/app/main.abc12345.js's parent
                                         //    → /static/app/util.js → 404 (wrong hash)
```

The browser resolves a relative specifier against the **current module's URL** before consulting the importmap, so the importmap never sees `./util.js`. This is a spec-level behavior, not a Grove limitation.

**Author guidance:** use bare specifiers keyed on logical names (`"app/util"`) across your module graph. Treat the logical name as the canonical ID. This is the pattern jsconfig, Deno, and browser importmaps all converge on.

If authors insist on relative imports, two workarounds exist (both outside this spec):

1. **Disable hashing** for JS (`Config.HashFiles` per-type split) — keeps relative paths stable, loses cache-busting on JS.
2. **Add a rewriting transformer** later that parses imports and rewrites them to hashed URLs. This is the full-bundler path we're explicitly not taking now.

## Design

Two layers:

### Layer 1 — Core seam: `type="module"` scripts

**File:** `pkg/grove/result.go`

`FootHTML()` currently iterates `sortedAssets(r.Assets, "script")` and emits `<script src="…">`. Extend it to also iterate assets with `Type == "module"` and emit:

```html
<script type="module" src="…"></script>
```

No struct change to `Asset`; the `Type` field already carries the discriminator (`"stylesheet"`, `"script"`, `"preload"`).

**File:** `internal/compiler/…` / `internal/vm/…` (wherever `{% asset %}` writes `Asset.Type`) — allow `"module"` alongside the existing values. This is likely already a pass-through of the `type=` attr; verify.

**Template syntax** (no new tag):

```grov
{% asset "app/main.js" type="module" %}
```

Rendered (with manifest resolver wired up):

```html
<script type="module" src="/static/app/main.abc12345.js"></script>
```

### Layer 2 — Opt-in subpackage: `pkg/grove/assets/esm/`

Pure Go, depends only on `pkg/grove/assets` and `encoding/json`.

**Public API**

```go
package esm

type Options struct {
    // StripJSExt emits both "foo/bar.js" and "foo/bar" keys pointing at the
    // hashed URL, so authors can write `import x from "foo/bar"`.
    StripJSExt bool

    // Include optionally filters which logical names enter the importmap.
    // Default: all entries ending in ".js" or ".mjs".
    Include func(logicalName string) bool

    // Extra adds or overrides entries in the importmap's "imports" block
    // (e.g. CDN-hosted libraries).
    Extra map[string]string

    // Scopes is passed through as the importmap "scopes" field.
    Scopes map[string]map[string]string

    // Indent, if > 0, pretty-prints the importmap JSON.
    Indent int
}

// Importmap returns a full `<script type="importmap">…</script>` block built
// from the manifest's JS entries. Returns "" when there is nothing to emit.
func Importmap(m *assets.Manifest, opts Options) string
```

**Implementation outline**

1. Call `m.Entries()` (already public; `manifest.go:52`).
2. Filter to JS (`.js`, `.mjs`) applying `opts.Include` if set.
3. Build `imports`: `{logicalName: hashedURL}`, plus `{stemWithoutExt: hashedURL}` when `StripJSExt` is true.
4. Overlay `opts.Extra` on top of `imports` (Extra wins).
5. Marshal `{"imports": imports, "scopes": opts.Scopes}` deterministically (sorted keys).
6. Wrap in `<script type="importmap">…</script>`.

**Tests (`importmap_test.go`)**

| Case | Assertion |
|---|---|
| Empty manifest | returns `""` |
| JS + CSS mix | only `.js`/`.mjs` entries appear |
| `StripJSExt: true` | both `foo/bar` and `foo/bar.js` keys present, same URL |
| `Extra` override | Extra keys override manifest keys |
| `Scopes` | scopes block round-trips into JSON unchanged |
| Key ordering | repeated calls produce byte-identical output |

## Wiring — author usage

This change requires no new template tag. The importmap is a string produced in Go and passed into the render as global data, then emitted through the `safe` filter.

```go
package main

import (
    "github.com/wispberry-tech/grove/pkg/grove"
    "github.com/wispberry-tech/grove/pkg/grove/assets"
    "github.com/wispberry-tech/grove/pkg/grove/assets/esm"
)

func main() {
    manifest, _ := assets.LoadManifest("dist/manifest.json")

    importmapHTML := esm.Importmap(manifest, esm.Options{
        StripJSExt: true,
        Extra: map[string]string{
            "htmx.org": "https://esm.sh/htmx.org@2.0.0",
        },
    })

    eng := grove.New(
        grove.WithAssetResolver(manifest.Resolve),
        grove.WithGlobalData(map[string]any{
            "importmap": importmapHTML,
        }),
    )
    _ = eng
}
```

**Template (`layouts/base.grov`):**

```grov
<!doctype html>
<html>
<head>
  <title>{{ title }}</title>
  {{ importmap | safe }}
  {% hoist target="head" %}
</head>
<body>
  {% slot %}

  {% asset "app/main.js" type="module" %}
</body>
</html>
```

**Author module (`views/app/main.js` — source):**

```js
import { clamp }  from "app/util";
import { render } from "app/ui/render";

render(document.body, { value: clamp(5, 0, 10) });
```

**Rendered `<head>` (prod):**

```html
<script type="importmap">
{"imports":{
  "app/main":"/static/app/main.abc12345.js",
  "app/main.js":"/static/app/main.abc12345.js",
  "app/ui/render":"/static/app/ui/render.111aaaaa.js",
  "app/ui/render.js":"/static/app/ui/render.111aaaaa.js",
  "app/util":"/static/app/util.def67890.js",
  "app/util.js":"/static/app/util.def67890.js",
  "htmx.org":"https://esm.sh/htmx.org@2.0.0"
}}
</script>
```

**Rendered end-of-body:**

```html
<script type="module" src="/static/app/main.abc12345.js"></script>
```

The browser parses the importmap first, then loads `app/main.abc12345.js` as a module, and resolves its `import "app/util"` against the importmap — landing on the hashed URL.

## Files

| File | Change |
|---|---|
| `pkg/grove/result.go` | `FootHTML()` also emits `type="module"` for `Asset.Type == "module"` |
| `internal/compiler/…` *(TBD during impl)* | Accept `"module"` as a valid `{% asset %}` type value |
| `pkg/grove/assets/esm/importmap.go` | **new** — `Importmap(manifest, opts) string` |
| `pkg/grove/assets/esm/importmap_test.go` | **new** — table tests |
| `pkg/grove/assets/esm/doc.go` | **new** — package doc |
| `docs/asset-pipeline.md` | short "ESM modules" section linking to this spec |
| `examples/juicebar/` | *(optional)* convert one script to a module + importmap as a smoke test |

## Limitations

1. **Relative imports in fingerprinted files.** See [the dedicated section](#does-this-let-me-import-js-modules-without-issue). Authors must prefer bare specifiers.
2. **No import preloading.** Large module graphs may incur request waterfalls. A follow-up could generate `<link rel="modulepreload">` from the importmap.
3. **Importmap browser support.** All evergreen browsers as of 2024. Safari < 16.4 / older Firefox need a polyfill (e.g. `es-module-shims`) — outside Grove's scope to bundle.
4. **Single importmap per document.** The HTML spec allows only one importmap per document (as of this writing). `Importmap()` returns the full block; injecting it twice will be rejected by the browser.
5. **No integrity hashes.** Follow-up if desired.

## Verification

1. `go test ./pkg/grove/assets/esm/...` — unit tests for importmap generation.
2. Add an integration case to `pkg/grove/engine_test.go`: `{% asset "app.js" type="module" %}` + manifest resolver → `FootHTML()` contains `<script type="module" src="/static/app.abc12345.js">`.
3. `go clean -testcache && go test ./... -v` — full suite, no regressions.
4. `go build ./...`.
5. Manual in `examples/juicebar`: convert one page to use a module + importmap, verify in browser devtools that modules load and bare-specifier imports resolve with no 404s.

## Future work (not in scope)

- Import rewriting transformer for authors who want to keep relative imports.
- Automatic `modulepreload` emission for eager dependencies.
- `integrity` attribute generation from content hashes.
- Scopes auto-generation for nested manifests / per-component importmaps.
