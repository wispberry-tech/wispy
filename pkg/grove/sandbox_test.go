package grove_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wispberry-tech/grove/pkg/grove"
)

// TestSandbox_AllowedTags_BlocksIf verifies that if tag is rejected when not in AllowedTags.
func TestSandbox_AllowedTags_BlocksIf(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedTags: []string{"set", "each"},
	}))
	err := renderErr(t, eng, `{% #if true %}yes{% /if %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "if")
}

// TestSandbox_AllowedTags_BlocksEach verifies that each tag is rejected when not in AllowedTags.
func TestSandbox_AllowedTags_BlocksEach(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedTags: []string{"set", "if"},
	}))
	err := renderErr(t, eng, `{% #each [] as x %}{% /each %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "each")
}

// TestSandbox_AllowedTags_BlocksImport verifies that import tag is rejected when not in AllowedTags.
func TestSandbox_AllowedTags_BlocksImport(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedTags: []string{"set", "if"},
	}))
	err := renderErr(t, eng, `{% import X from "foo" %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "import")
}

// TestSandbox_AllowedTags_AllowsSet verifies that allowed tags still work.
func TestSandbox_AllowedTags_AllowsSet(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedTags: []string{"set"},
	}))
	result := render(t, eng, `{% set x = 5 %}{% x %}`, grove.Data{})
	require.Equal(t, "5", result)
}

// TestSandbox_AllowedTags_NilAllowsAll verifies that nil AllowedTags allows all tags.
func TestSandbox_AllowedTags_NilAllowsAll(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedTags: nil,
	}))
	result := render(t, eng, `{% #if true %}{% /if %}{% set x = 1 %}{% #each [] as i %}{% /each %}ok`, grove.Data{})
	require.Equal(t, "ok", result)
}

// TestSandbox_AllowedFilters_BlocksUpper verifies that upper filter is rejected when not allowed.
func TestSandbox_AllowedFilters_BlocksUpper(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedFilters: []string{"lower"},
	}))
	err := renderErr(t, eng, `{% "hello" | upper %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "upper")
}

// TestSandbox_AllowedFilters_AllowsWhitelisted verifies that whitelisted filters work.
func TestSandbox_AllowedFilters_AllowsWhitelisted(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedFilters: []string{"upper", "length"},
	}))
	result := render(t, eng, `{% "hello" | upper %} {% [1, 2, 3] | length %}`, grove.Data{})
	require.Equal(t, "HELLO 3", result)
}

// TestSandbox_AllowedFilters_NilAllowsAll verifies that nil AllowedFilters allows all filters.
func TestSandbox_AllowedFilters_NilAllowsAll(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		AllowedFilters: nil,
	}))
	result := render(t, eng, `{% "x" | upper | lower | title | capitalize | trim | length %}`, grove.Data{})
	require.Equal(t, "1", result)
}

// TestSandbox_MaxLoopIter_Exceeded verifies that exceeding MaxLoopIter causes a RuntimeError.
func TestSandbox_MaxLoopIter_Exceeded(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		MaxLoopIter: 5,
	}))
	err := renderErr(t, eng, `{% #each range(1, 10) as i %}x{% /each %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "loop")
}

// TestSandbox_MaxLoopIter_NotExceeded verifies that staying within MaxLoopIter works.
func TestSandbox_MaxLoopIter_NotExceeded(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		MaxLoopIter: 5,
	}))
	result := render(t, eng, `{% #each range(1, 4) as i %}x{% /each %}`, grove.Data{})
	require.Equal(t, "xxx", result)
}

// TestSandbox_MaxLoopIter_ZeroIsUnlimited verifies that 0 means unlimited loop iterations.
func TestSandbox_MaxLoopIter_ZeroIsUnlimited(t *testing.T) {
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		MaxLoopIter: 0,
	}))
	result := render(t, eng, `{% #each range(1, 101) as i %}a{% /each %}`, grove.Data{})
	require.Equal(t, 100, len(result))
}

// Tier 4 #9: MaxLoopIter boundary. Counter increments on OP_FOR_INIT (first
// iteration) and OP_FOR_STEP (each subsequent continuation), so the counter
// equals the number of body executions. MaxLoopIter=N allows exactly N body
// executions across all loops in a render.
func TestSandbox_MaxLoopIter_Boundary(t *testing.T) {
	// Limit 5, 5 body executions: allowed.
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 5}))
	result := render(t, eng, `{% #each range(1, 6) as i %}{% i %},{% /each %}`, grove.Data{})
	require.Equal(t, "1,2,3,4,5,", result)

	// Limit 5, 6 body executions: one over → RuntimeError.
	eng2 := newEngine(t, grove.WithSandbox(grove.SandboxConfig{MaxLoopIter: 5}))
	err := renderErr(t, eng2, `{% #each range(1, 7) as i %}{% i %},{% /each %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "5")
}

// TestSandbox_MaxLoopIter_NestedLoops verifies that MaxLoopIter counts body
// executions across every loop level. Outer body running an inner loop
// contributes both its own execution and each of its inner iterations.
func TestSandbox_MaxLoopIter_NestedLoops(t *testing.T) {
	// Outer 3 × Inner 3 = 3 outer bodies + 9 inner bodies = 12 total.
	eng := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		MaxLoopIter: 12,
	}))
	result := render(t, eng, `{% #each range(1, 4) as i %}{% #each range(1, 4) as j %}x{% /each %}{% /each %}`, grove.Data{})
	require.Equal(t, 9, len(result))

	// One less than the needed budget → error.
	eng2 := newEngine(t, grove.WithSandbox(grove.SandboxConfig{
		MaxLoopIter: 11,
	}))
	err := renderErr(t, eng2, `{% #each range(1, 4) as i %}{% #each range(1, 4) as j %}x{% /each %}{% /each %}`, grove.Data{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "loop")
}
