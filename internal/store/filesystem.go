// internal/store/filesystem.go
package store

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileSystemStore loads templates from a root directory on disk.
// Template names are treated as forward-slash paths relative to the root.
// Names that resolve outside the root (via ".." traversal or absolute paths) are rejected.
type FileSystemStore struct {
	root string // absolute, cleaned OS path
}

// NewFileSystemStore creates a FileSystemStore rooted at root.
func NewFileSystemStore(root string) *FileSystemStore {
	return &FileSystemStore{root: filepath.Clean(root)}
}

// Load reads the template at name (forward-slash path) from the store root.
// Returns an error if name escapes the root via ".." components or is absolute.
func (s *FileSystemStore) Load(name string) ([]byte, error) {
	// 1. Clean using forward-slash path package (template names always use /)
	clean := path.Clean(name)

	// 2. Reject absolute paths
	if path.IsAbs(clean) {
		return nil, fmt.Errorf("template name %q is absolute — names must be relative", name)
	}

	// 3. Reject paths that start with ".." after cleaning
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return nil, fmt.Errorf("template name %q escapes the store root", name)
	}

	// 4. Build the full OS path
	full := filepath.Join(s.root, filepath.FromSlash(clean))

	// 5. Double-check containment: ensure full is under s.root
	// (defends against edge cases on non-Unix systems)
	rootPrefix := s.root + string(filepath.Separator)
	if !strings.HasPrefix(full+string(filepath.Separator), rootPrefix) {
		return nil, fmt.Errorf("template name %q escapes the store root", name)
	}

	// Step 1: try exact match
	if data, err := os.ReadFile(full); err == nil {
		return data, nil
	}

	// Steps 2 & 3 only apply when the name doesn't already have a .grov extension
	if !strings.HasSuffix(clean, ".grov") {
		// Step 2: try appending .grov
		if data, err := os.ReadFile(full + ".grov"); err == nil {
			return data, nil
		}

		// Step 3: try directory fallback — name/basename.grov
		base := path.Base(clean)
		dirFull := filepath.Join(full, base+".grov")
		if data, err := os.ReadFile(dirFull); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("template %q not found", name)
}
