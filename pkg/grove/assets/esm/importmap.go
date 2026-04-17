package esm

import (
	"encoding/json"
	"maps"
	"strings"

	"github.com/wispberry-tech/grove/pkg/grove/assets"
)

// Options controls how an importmap is built from a Manifest.
type Options struct {
	// StripJSExt also emits "foo/bar" (ext-stripped) alongside "foo/bar.js",
	// both pointing at the same hashed URL. Lets authors write
	// `import x from "foo/bar"`.
	StripJSExt bool

	// Include optionally filters which logical names enter the importmap.
	// When nil, every .js and .mjs entry is included.
	Include func(logicalName string) bool

	// Extra adds or overrides entries in the "imports" block (e.g. CDN
	// libraries like {"htmx.org": "https://esm.sh/htmx.org@2.0.0"}).
	// Extra keys win over manifest-derived keys on conflict.
	Extra map[string]string

	// Scopes is passed through verbatim as the importmap "scopes" field.
	Scopes map[string]map[string]string

	// Indent, if > 0, pretty-prints the importmap JSON with the given
	// number of spaces per level. Defaults to compact output.
	Indent int
}

// Importmap returns a `<script type="importmap">...</script>` block built
// from the manifest's JS entries. Returns "" when there is nothing to emit
// (no matching manifest entries, no Extra, no Scopes).
func Importmap(m *assets.Manifest, opts Options) string {
	imports := map[string]string{}

	if m != nil {
		for logical, url := range m.Entries() {
			if !isJS(logical) {
				continue
			}
			if opts.Include != nil && !opts.Include(logical) {
				continue
			}
			imports[logical] = url
			if opts.StripJSExt {
				if stem := stripExt(logical); stem != logical {
					imports[stem] = url
				}
			}
		}
	}

	maps.Copy(imports, opts.Extra)

	if len(imports) == 0 && len(opts.Scopes) == 0 {
		return ""
	}

	payload := struct {
		Imports map[string]string            `json:"imports,omitempty"`
		Scopes  map[string]map[string]string `json:"scopes,omitempty"`
	}{
		Imports: imports,
		Scopes:  opts.Scopes,
	}

	var (
		body []byte
		err  error
	)
	if opts.Indent > 0 {
		body, err = json.MarshalIndent(payload, "", strings.Repeat(" ", opts.Indent))
	} else {
		body, err = marshalSorted(payload)
	}
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<script type="importmap">`)
	sb.Write(body)
	sb.WriteString("</script>")
	return sb.String()
}

// marshalSorted produces byte-deterministic JSON: compact, with every map
// key sorted. Go's encoding/json already sorts map keys, but we rebuild the
// payload through json.Marshal so the contract is explicit and survives
// future changes.
func marshalSorted(v any) ([]byte, error) {
	// encoding/json sorts map keys deterministically for string-keyed maps,
	// so a plain Marshal is sufficient. Kept as a seam in case we ever swap
	// encoders.
	return json.Marshal(v)
}

func isJS(name string) bool {
	return strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".mjs")
}

func stripExt(name string) string {
	switch {
	case strings.HasSuffix(name, ".mjs"):
		return name[:len(name)-4]
	case strings.HasSuffix(name, ".js"):
		return name[:len(name)-3]
	default:
		return name
	}
}
