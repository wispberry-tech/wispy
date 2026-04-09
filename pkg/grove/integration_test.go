// pkg/grove/integration_test.go
package grove_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// --- Component definition + slot fill (replaces macro + component) ---

func TestIntegration_ComponentAndSlotFill(t *testing.T) {
	// A component defined in one template, imported and used with slot fill in another.
	store := grove.NewMemoryStore()
	store.Set("card.html", `<div class="card"><Slot /></div>`)
	// Badge is now a component defined in its own template.
	store.Set("badge.html", `<Component name="Badge"><span>{% label %}</span></Component>`)
	store.Set("page.html", `<Import src="card.html" name="Card" /><Import src="badge.html" name="Badge" /><Card><Badge label="New" /></Card>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, `<div class="card"><span>New</span></div>`, result.Body)
}

// --- Imported component used inside another component's fill ---

func TestIntegration_ImportedComponentInSlotFill(t *testing.T) {
	store := grove.NewMemoryStore()
	// "tag" component renders a dynamic HTML element (replaces the macro)
	store.Set("tags.html", `<Component name="Tag"><{% name %}></Component>`)
	store.Set("wrap.html", `<section><Slot /></section>`)
	store.Set("page.html", `<Import src="tags.html" name="Tag" /><Import src="wrap.html" name="Wrap" /><Wrap><Tag name="span" /></Wrap>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<section><span></section>", result.Body)
}

// --- Asset + hoist bubble from component to page ---

func TestIntegration_ComponentBubblesAssetAndHoist(t *testing.T) {
	// Asset declared and hoist emitted inside a component should appear in the
	// top-level RenderResult, not in the component body.
	store := grove.NewMemoryStore()
	store.Set("widget.html", `<ImportAsset src="widget.css" type="stylesheet" /><Hoist target="foot"><script>w()</script></Hoist><div>widget</div>`)
	store.Set("page.html", `<Import src="widget.html" name="Widget" /><Widget />`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<div>widget</div>", result.Body)
	require.Len(t, result.Assets, 1)
	require.Equal(t, "widget.css", result.Assets[0].Src)
	require.Contains(t, result.GetHoisted("foot"), "w()")
}

// --- Layout composition: child fills slots in parent (replaces extends/block) ---

func TestIntegration_LayoutCompositionWithDataVars(t *testing.T) {
	// Variables from render data are visible in both layout and fill content.
	store := grove.NewMemoryStore()
	store.Set("base.html", `<html><title><Slot name="title" /></title><body><Slot name="body" /></body></html>`)
	store.Set("page.html", `<Import src="base.html" name="Base" /><Base><Fill slot="title">{% site %} — {% page_title %}</Fill><Fill slot="body">{% content %}</Fill></Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{
		"site":       "Acme",
		"page_title": "Home",
		"content":    "Welcome!",
	})
	require.NoError(t, err)
	require.Equal(t, "<html><title>Acme — Home</title><body>Welcome!</body></html>", result.Body)
}

// --- Concurrent renders - race detector target ---

func TestIntegration_ConcurrentRenders(t *testing.T) {
	// Multiple goroutines render the same multi-template composition concurrently.
	// Run with -race to detect data races: go test -race ./pkg/grove/...
	store := grove.NewMemoryStore()
	store.Set("base.html", `[<Slot name="title">base</Slot>|<Slot name="body" />]`)
	store.Set("page.html", `<Import src="base.html" name="Base" /><Base><Fill slot="title">{% title %}</Fill><Fill slot="body">{% content %}</Fill></Base>`)

	eng := grove.New(grove.WithStore(store))
	ctx := context.Background()

	const goroutines = 20
	const rendersEach = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errors := make(chan error, goroutines*rendersEach)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < rendersEach; i++ {
				result, err := eng.Render(ctx, "page.html", grove.Data{
					"title":   "Page",
					"content": "hello",
				})
				if err != nil {
					errors <- err
					return
				}
				if !strings.Contains(result.Body, "Page") {
					errors <- nil
				}
			}
		}()
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent render error: %v", err)
		} else {
			t.Error("concurrent render produced unexpected output")
		}
	}
}

// --- Component inside layout fill (replaces component inside block of extends) ---

func TestIntegration_ComponentInsideLayoutFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<html><body><Slot name="content" /></body></html>`)
	store.Set("card.html", `<div><Slot /></div>`)
	store.Set("page.html", `<Import src="base.html" name="Base" /><Import src="card.html" name="Card" /><Base><Fill slot="content"><Card>hello</Card></Fill></Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<html><body><div>hello</div></body></html>", result.Body)
}

func TestIntegration_AssetInsideLayoutFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body><Slot name="content" /></body>`)
	store.Set("child.html", `<Import src="base.html" name="Base" /><Base><Fill slot="content"><ImportAsset src="app.css" type="stylesheet" />content</Fill></Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Len(t, result.Assets, 1)
}

func TestIntegration_HoistInsideLayoutFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body><Slot name="content" /></body>`)
	store.Set("child.html", `<Import src="base.html" name="Base" /><Base><Fill slot="content"><Hoist target="head"><style>.x{}</style></Hoist>content</Fill></Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Contains(t, result.GetHoisted("head"), ".x{}")
}
