package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temp directory with:
//
//	testdata/
//	  flat.grov                          ← exact match with extension
//	  no-ext                             ← exact match without extension
//	  primitives/button/button.grov      ← directory fallback target
//	  composites/card.grov               ← .grov-appended target (flat file)
//	  composites/nav/nav.grov            ← directory fallback target
func setupTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"flat.grov":                     "flat",
		"no-ext":                        "noext",
		"primitives/button/button.grov": "button-dir",
		"composites/card.grov":          "card-flat",
		"composites/nav/nav.grov":       "nav-dir",
	}

	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		err := os.MkdirAll(filepath.Dir(full), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(full, []byte(content), 0o644)
		require.NoError(t, err)
	}

	return root
}

func TestFileSystemStore_Load_ExactMatch(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	data, err := s.Load("flat.grov")
	require.NoError(t, err)
	require.Equal(t, "flat", string(data))
}

func TestFileSystemStore_Load_ExactMatchNoExtension(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	data, err := s.Load("no-ext")
	require.NoError(t, err)
	require.Equal(t, "noext", string(data))
}

func TestFileSystemStore_Load_GrovAppended(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	data, err := s.Load("composites/card")
	require.NoError(t, err)
	require.Equal(t, "card-flat", string(data))
}

func TestFileSystemStore_Load_DirectoryFallback(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	data, err := s.Load("primitives/button")
	require.NoError(t, err)
	require.Equal(t, "button-dir", string(data))
}

func TestFileSystemStore_Load_DirectoryFallbackNested(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	data, err := s.Load("composites/nav")
	require.NoError(t, err)
	require.Equal(t, "nav-dir", string(data))
}

func TestFileSystemStore_Load_GrovAppendedPrefersOverDirectory(t *testing.T) {
	root := setupTestDir(t)

	// Add composites/card/card.grov alongside the existing composites/card.grov
	dir := filepath.Join(root, "composites", "card")
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "card.grov"), []byte("card-dir"), 0o644)
	require.NoError(t, err)

	s := NewFileSystemStore(root)

	data, err := s.Load("composites/card")
	require.NoError(t, err)
	// Step 2 (.grov appended) should win over step 3 (directory fallback)
	require.Equal(t, "card-flat", string(data))
}

func TestFileSystemStore_Load_NotFound(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	_, err := s.Load("does-not-exist")
	require.Error(t, err)
}

func TestFileSystemStore_Load_PathTraversal(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	_, err := s.Load("../outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes the store root")
}

func TestFileSystemStore_Load_AbsolutePath(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	_, err := s.Load("/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "absolute")
}

func TestFileSystemStore_Load_SkipsFallbackWhenExtensionPresent(t *testing.T) {
	root := setupTestDir(t)
	s := NewFileSystemStore(root)

	// "flat.grov" should resolve via exact match; it must NOT try "flat.grov.grov"
	data, err := s.Load("flat.grov")
	require.NoError(t, err)
	require.Equal(t, "flat", string(data))
}
