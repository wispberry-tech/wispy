// Package esm generates browser importmap <script> blocks from a Grove
// asset Manifest, letting authors load fingerprinted JavaScript via bare
// module specifiers without a bundler.
//
// This package is opt-in; importing it is the only way to pull in its code.
// The core pkg/grove/assets package has no knowledge of modules or import
// maps, mirroring how pkg/grove/assets/minify isolates the minifier
// dependency. esm itself uses only the standard library.
//
// Typical wiring:
//
//	manifest, _ := assets.LoadManifest("dist/manifest.json")
//	importmap := esm.Importmap(manifest, esm.Options{StripJSExt: true})
//	eng := grove.New(
//	    grove.WithAssetResolver(manifest.Resolve),
//	    grove.WithGlobalData(map[string]any{"importmap": importmap}),
//	)
//
// In templates, emit the importmap via {{ importmap | safe }} in <head>,
// then declare module scripts with {% asset "app.js" type="module" %}.
// See docs/spec/esm-support.md for the full design and its limits (in
// particular: relative imports in fingerprinted files do NOT resolve via
// importmap; authors should use bare specifiers).
package esm
