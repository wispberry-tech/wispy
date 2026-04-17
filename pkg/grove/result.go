// pkg/wispy/result.go
package grove

import (
	"fmt"
	"sort"
	"strings"
)

// Asset represents a single CSS, JS, or other resource declared via {% asset %}.
type Asset struct {
	Src      string
	Type     string            // "stylesheet", "script", "preload", etc.
	Attrs    map[string]string // boolean attrs stored as key→""; serialized as bare attr in HTML
	Priority int               // higher = earlier in its type-group; default 0
}

// Warning is a non-fatal message generated during rendering (e.g. meta key collision).
type Warning struct {
	Message string
}

// RenderResult holds the full output of a render operation.
type RenderResult struct {
	Body     string
	Assets   []Asset             // collected via {% asset %}, deduplicated by Src
	Meta     map[string]string   // collected via {% meta %}; last write wins
	Hoisted  map[string][]string // target → ordered fragments from {% hoist target="..." %}
	Warnings []Warning           // non-fatal runtime warnings (e.g. meta key overwrite)
}

// HeadHTML returns <link rel="stylesheet"> tags for all Type=="stylesheet" assets,
// sorted by descending Priority within the group.
func (r RenderResult) HeadHTML() string {
	var sb strings.Builder
	for _, a := range sortedAssets(r.Assets, "stylesheet") {
		sb.WriteString(`<link rel="stylesheet" href="`)
		sb.WriteString(a.Src)
		sb.WriteByte('"')
		sb.WriteString(formatAttrs(a.Attrs))
		sb.WriteString(">\n")
	}
	return sb.String()
}

// FootHTML returns <script> tags for all Type=="script" and Type=="module"
// assets, sorted by descending Priority within each group. Classic scripts
// are emitted first, then module scripts.
func (r RenderResult) FootHTML() string {
	var sb strings.Builder
	for _, a := range sortedAssets(r.Assets, "script") {
		sb.WriteString(`<script src="`)
		sb.WriteString(a.Src)
		sb.WriteByte('"')
		sb.WriteString(formatAttrs(a.Attrs))
		sb.WriteString("></script>\n")
	}
	for _, a := range sortedAssets(r.Assets, "module") {
		sb.WriteString(`<script type="module" src="`)
		sb.WriteString(a.Src)
		sb.WriteByte('"')
		sb.WriteString(formatAttrs(a.Attrs))
		sb.WriteString("></script>\n")
	}
	return sb.String()
}

// GetHoisted returns the concatenated content for the given hoist target.
func (r RenderResult) GetHoisted(target string) string {
	frags := r.Hoisted[target]
	if len(frags) == 0 {
		return ""
	}
	return strings.Join(frags, "")
}

// sortedAssets returns assets of the given type, sorted by descending Priority.
func sortedAssets(assets []Asset, assetType string) []Asset {
	var filtered []Asset
	for _, a := range assets {
		if a.Type == assetType {
			filtered = append(filtered, a)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Priority > filtered[j].Priority
	})
	return filtered
}

// formatAttrs serializes an attr map as HTML attributes.
// Keys with value "" are emitted as bare attributes (e.g. "defer").
// Keys with non-empty values are emitted as key="value".
func formatAttrs(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	// Sort for deterministic output
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		v := attrs[k]
		if v == "" {
			sb.WriteByte(' ')
			sb.WriteString(k)
		} else {
			sb.WriteString(fmt.Sprintf(` %s="%s"`, k, v))
		}
	}
	return sb.String()
}
