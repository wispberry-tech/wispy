package grove_test

import (
	"context"
	"fmt"

	"github.com/wispberry-tech/grove/pkg/grove"
)

// ExampleWithAssetResolver shows how logical {% asset %} names get rewritten
// through a resolver. The resolver signature matches assets.Manifest.Resolve
// so in real code you would typically pass manifest.Resolve here.
func ExampleWithAssetResolver() {
	store := grove.NewMemoryStore()
	store.Set("page.html", `{% asset "button.css" type="stylesheet" %}<p>hi</p>`)

	resolver := func(logical string) (string, bool) {
		return "/dist/" + logical + ".a1b2c3d4.css", true
	}

	eng := grove.New(
		grove.WithStore(store),
		grove.WithAssetResolver(resolver),
	)

	result, err := eng.Render(context.Background(), "page.html", grove.Data{})
	if err != nil {
		panic(err)
	}

	fmt.Println(result.Assets[0].Src)
	// Output: /dist/button.css.a1b2c3d4.css
}
