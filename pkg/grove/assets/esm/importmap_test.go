package esm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove/assets"
)

func mustManifest(t *testing.T, entries map[string]string) *assets.Manifest {
	t.Helper()
	m := assets.NewManifest()
	for k, v := range entries {
		m.Set(k, v)
	}
	return m
}

func parseImports(t *testing.T, s string) map[string]string {
	t.Helper()
	require.True(t, strings.HasPrefix(s, `<script type="importmap">`), "missing importmap opener: %q", s)
	require.True(t, strings.HasSuffix(s, `</script>`), "missing closing </script>: %q", s)
	body := strings.TrimSuffix(strings.TrimPrefix(s, `<script type="importmap">`), `</script>`)
	var payload struct {
		Imports map[string]string            `json:"imports"`
		Scopes  map[string]map[string]string `json:"scopes"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &payload))
	return payload.Imports
}

func TestImportmap_EmptyManifest(t *testing.T) {
	require.Equal(t, "", Importmap(nil, Options{}))
	require.Equal(t, "", Importmap(assets.NewManifest(), Options{}))
}

func TestImportmap_FiltersToJS(t *testing.T) {
	m := mustManifest(t, map[string]string{
		"app.js":   "/s/app.aaa.js",
		"app.mjs":  "/s/app.bbb.mjs",
		"app.css":  "/s/app.ccc.css",
		"logo.svg": "/s/logo.ddd.svg",
	})
	got := parseImports(t, Importmap(m, Options{}))
	require.Equal(t, map[string]string{
		"app.js":  "/s/app.aaa.js",
		"app.mjs": "/s/app.bbb.mjs",
	}, got)
}

func TestImportmap_StripJSExt(t *testing.T) {
	m := mustManifest(t, map[string]string{
		"app/main.js": "/s/app/main.aaa.js",
		"app/util.js": "/s/app/util.bbb.js",
	})
	got := parseImports(t, Importmap(m, Options{StripJSExt: true}))
	require.Equal(t, map[string]string{
		"app/main":    "/s/app/main.aaa.js",
		"app/main.js": "/s/app/main.aaa.js",
		"app/util":    "/s/app/util.bbb.js",
		"app/util.js": "/s/app/util.bbb.js",
	}, got)
}

func TestImportmap_IncludeFilter(t *testing.T) {
	m := mustManifest(t, map[string]string{
		"public/a.js":  "/s/a.js",
		"private/b.js": "/s/b.js",
	})
	got := parseImports(t, Importmap(m, Options{
		Include: func(n string) bool { return strings.HasPrefix(n, "public/") },
	}))
	require.Equal(t, map[string]string{"public/a.js": "/s/a.js"}, got)
}

func TestImportmap_ExtraOverrides(t *testing.T) {
	m := mustManifest(t, map[string]string{
		"vendor.js": "/s/vendor.aaa.js",
	})
	got := parseImports(t, Importmap(m, Options{
		Extra: map[string]string{
			"vendor.js": "https://cdn.example.com/vendor.js",
			"htmx.org":  "https://esm.sh/htmx.org@2.0.0",
		},
	}))
	require.Equal(t, map[string]string{
		"vendor.js": "https://cdn.example.com/vendor.js",
		"htmx.org":  "https://esm.sh/htmx.org@2.0.0",
	}, got)
}

func TestImportmap_ScopesPassthrough(t *testing.T) {
	scopes := map[string]map[string]string{
		"/scoped/": {"lib": "/scoped/lib.js"},
	}
	out := Importmap(nil, Options{Scopes: scopes})
	require.Contains(t, out, `"scopes"`)
	require.Contains(t, out, `"/scoped/"`)
	require.Contains(t, out, `"lib":"/scoped/lib.js"`)
}

func TestImportmap_EmptyWithOnlyNonJSManifest(t *testing.T) {
	m := mustManifest(t, map[string]string{"app.css": "/s/app.css"})
	require.Equal(t, "", Importmap(m, Options{}))
}

func TestImportmap_Deterministic(t *testing.T) {
	m := mustManifest(t, map[string]string{
		"z.js": "/s/z.js",
		"a.js": "/s/a.js",
		"m.js": "/s/m.js",
	})
	first := Importmap(m, Options{})
	for range 20 {
		require.Equal(t, first, Importmap(m, Options{}))
	}
}

func TestImportmap_Indent(t *testing.T) {
	m := mustManifest(t, map[string]string{"app.js": "/s/app.js"})
	out := Importmap(m, Options{Indent: 2})
	require.Contains(t, out, "\n  \"imports\"")
}

func TestStripExt(t *testing.T) {
	require.Equal(t, "foo", stripExt("foo.js"))
	require.Equal(t, "foo", stripExt("foo.mjs"))
	require.Equal(t, "foo.txt", stripExt("foo.txt"))
	require.Equal(t, "foo", stripExt("foo"))
}
