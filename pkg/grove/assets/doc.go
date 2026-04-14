// Package assets is Grove's opt-in asset build pipeline. It scans a source
// directory for CSS and JS files, applies a pluggable [Transformer], writes
// content-hashed copies to an output directory, and produces a JSON [Manifest]
// mapping logical asset names to served URLs.
//
// The Grove engine resolves {% asset %} logical names through a resolver
// function (typically [*Manifest.Resolve]) configured via
// grove.WithAssetResolver. Apps that do not import this package pay zero cost.
//
// # Overview
//
// A typical production wiring:
//
//	import (
//	    "github.com/wispberry-tech/grove/pkg/grove"
//	    "github.com/wispberry-tech/grove/pkg/grove/assets"
//	    "github.com/wispberry-tech/grove/pkg/grove/assets/minify"
//	)
//
//	builder := assets.NewWithDefaults(assets.Config{
//	    SourceDir:      "templates",
//	    OutputDir:      "dist",
//	    URLPrefix:      "/dist",
//	    CSSTransformer: minify.New(),
//	    JSTransformer:  minify.New(),
//	    ManifestPath:   "dist/manifest.json",
//	})
//	manifest, err := builder.Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	eng := grove.New(
//	    grove.WithStore(grove.NewFileSystemStore("templates")),
//	    grove.WithAssetResolver(manifest.Resolve),
//	)
//
//	pattern, handler := builder.Route()
//	mux.Handle(pattern+"*", handler)
//
// # Watch mode
//
// For development, [*Builder.Watch] polls SourceDir on a 500ms tick with 100ms
// debounce. Failed files keep their previous manifest entry (partial swap).
// Call [*grove.Engine.SetAssetResolver] from the OnChange handler to pick up
// new hashes without restarting.
//
// # Pruning
//
// Set Config.PruneUnreferenced and wire the engine's tracker:
//
//	builder.SetReferencedNameProvider(eng.ReferencedAssets)
//
// After the first render, subsequent builds drop manifest entries for files
// no template actually referenced.
//
// # Minify sub-package
//
// [github.com/wispberry-tech/grove/pkg/grove/assets/minify] wraps
// github.com/tdewolff/minify/v2 as a [Transformer]. The main assets package
// has no external dependencies; importing the minify sub-package pulls in
// tdewolff/minify and tdewolff/parse.
package assets
