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
	store.Set("card.html", `<div class="card">{% slot %}</div>`)
	// Badge is now a component defined in its own template.
	store.Set("badge.html", `<span>{% label %}</span>`)
	store.Set("page.html", `{% import Card from "card" %}{% import Badge from "badge" %}<Card><Badge label="New" /></Card>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, `<div class="card"><span>New</span></div>`, result.Body)
}

// --- Imported component used inside another component's fill ---

func TestIntegration_ImportedComponentInSlotFill(t *testing.T) {
	store := grove.NewMemoryStore()
	// "tag" component renders a dynamic HTML element (replaces the macro)
	store.Set("tags.html", `<{% name %}>`)
	store.Set("wrap.html", `<section>{% slot %}</section>`)
	store.Set("page.html", `{% import Tag from "tags" %}{% import Wrap from "wrap" %}<Wrap><Tag name="span" /></Wrap>`)

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
	store.Set("widget.html", `{% asset "widget.css" type="stylesheet" %}{% #hoist "foot" %}<script>w()</script>{% /hoist %}<div>widget</div>`)
	store.Set("page.html", `{% import Widget from "widget" %}<Widget />`)

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
	store.Set("base.html", `<html><title>{% slot "title" %}</title><body>{% slot "body" %}</body></html>`)
	store.Set("page.html", `{% import Base from "base" %}<Base>{% #fill "title" %}{% site %} — {% page_title %}{% /fill %}{% #fill "body" %}{% content %}{% /fill %}</Base>`)

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
	store.Set("base.html", `[{% #slot "title" %}base{% /slot %}|{% slot "body" %}]`)
	store.Set("page.html", `{% import Base from "base" %}<Base>{% #fill "title" %}{% title %}{% /fill %}{% #fill "body" %}{% content %}{% /fill %}</Base>`)

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
	store.Set("base.html", `<html><body>{% slot "content" %}</body></html>`)
	store.Set("card.html", `<div>{% slot %}</div>`)
	store.Set("page.html", `{% import Base from "base" %}{% import Card from "card" %}<Base>{% #fill "content" %}<Card>hello</Card>{% /fill %}</Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<html><body><div>hello</div></body></html>", result.Body)
}

func TestIntegration_AssetInsideLayoutFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body>{% slot "content" %}</body>`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "content" %}{% asset "app.css" type="stylesheet" %}content{% /fill %}</Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Len(t, result.Assets, 1)
}

func TestIntegration_HoistInsideLayoutFill(t *testing.T) {
	store := grove.NewMemoryStore()
	store.Set("base.html", `<body>{% slot "content" %}</body>`)
	store.Set("child.html", `{% import Base from "base" %}<Base>{% #fill "content" %}{% #hoist "head" %}<style>.x{}</style>{% /hoist %}content{% /fill %}</Base>`)

	eng := grove.New(grove.WithStore(store))
	result, err := eng.Render(context.Background(), "child.html", grove.Data{})
	require.NoError(t, err)
	require.Equal(t, "<body>content</body>", result.Body)
	require.Contains(t, result.GetHoisted("head"), ".x{}")
}
