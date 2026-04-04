# Atomic Design & JS Colocation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish two-tier atomic design conventions for Grove components, add directory-fallback path resolution to `FileSystemStore`, update documentation, and refactor blog + store examples to the new structure with colocated JS.

**Architecture:** `FileSystemStore.Load()` gains a three-step fallback (exact → `.grov` appended → `dir/basename.grov`) so `{% component "composites/card" %}` resolves to `composites/card/card.grov`. Documentation is updated first (docs-first approach), then both examples are restructured into `primitives/` and `composites/` directories with folder-per-component layout and colocated JS files served statically.

**Tech Stack:** Go 1.24, `testify` (test-only)

---

### Task 1: FileSystemStore Fallback — Failing Tests

**Files:**
- Create: `internal/store/filesystem_test.go`

This task creates the test file for `FileSystemStore` with tests covering the new fallback resolution behavior: exact match, `.grov` appended, and directory fallback `name/basename.grov`.

- [ ] **Step 1: Create test directory structure on disk**

Create `internal/store/filesystem_test.go` with a `TestMain` or test helper that sets up a temporary directory with the following structure:

```
testdata/
  flat.grov                          ← exact match with extension
  no-ext                             ← exact match without extension (non-template file)
  primitives/
    button/
      button.grov                    ← directory fallback target
  composites/
    card.grov                        ← .grov-appended target (flat file)
    nav/
      nav.grov                       ← directory fallback target
  escapes/
    ../outside.grov                  ← should never be created; path traversal test
```

```go
package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// flat.grov — exact match with extension
	require.NoError(t, os.WriteFile(filepath.Join(dir, "flat.grov"), []byte("flat"), 0644))

	// no-ext — exact match without extension
	require.NoError(t, os.WriteFile(filepath.Join(dir, "no-ext"), []byte("noext"), 0644))

	// primitives/button/button.grov — directory fallback
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "primitives", "button"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "primitives", "button", "button.grov"), []byte("button-dir"), 0644))

	// composites/card.grov — flat file for .grov append
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "composites"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composites", "card.grov"), []byte("card-flat"), 0644))

	// composites/nav/nav.grov — directory fallback
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "composites", "nav"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composites", "nav", "nav.grov"), []byte("nav-dir"), 0644))

	return dir
}
```

- [ ] **Step 2: Write failing tests for all resolution paths**

Add the following tests to `internal/store/filesystem_test.go`:

```go
func TestFileSystemStore_Load_ExactMatch(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	data, err := s.Load("flat.grov")
	require.NoError(t, err)
	require.Equal(t, "flat", string(data))
}

func TestFileSystemStore_Load_ExactMatchNoExtension(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	data, err := s.Load("no-ext")
	require.NoError(t, err)
	require.Equal(t, "noext", string(data))
}

func TestFileSystemStore_Load_GrovAppended(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	// "composites/card" should resolve to "composites/card.grov"
	data, err := s.Load("composites/card")
	require.NoError(t, err)
	require.Equal(t, "card-flat", string(data))
}

func TestFileSystemStore_Load_DirectoryFallback(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	// "primitives/button" should resolve to "primitives/button/button.grov"
	data, err := s.Load("primitives/button")
	require.NoError(t, err)
	require.Equal(t, "button-dir", string(data))
}

func TestFileSystemStore_Load_DirectoryFallbackNested(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	// "composites/nav" should resolve to "composites/nav/nav.grov"
	data, err := s.Load("composites/nav")
	require.NoError(t, err)
	require.Equal(t, "nav-dir", string(data))
}

func TestFileSystemStore_Load_GrovAppendedPrefersOverDirectory(t *testing.T) {
	dir := setupTestDir(t)

	// Create both composites/card.grov (flat) AND composites/card/card.grov (dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "composites", "card"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composites", "card", "card.grov"), []byte("card-dir"), 0644))

	s := NewFileSystemStore(dir)

	// Should prefer flat file (step 2) over directory fallback (step 3)
	data, err := s.Load("composites/card")
	require.NoError(t, err)
	require.Equal(t, "card-flat", string(data))
}

func TestFileSystemStore_Load_NotFound(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	_, err := s.Load("does-not-exist")
	require.Error(t, err)
}

func TestFileSystemStore_Load_PathTraversal(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	_, err := s.Load("../outside")
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes the store root")
}

func TestFileSystemStore_Load_AbsolutePath(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	_, err := s.Load("/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "absolute")
}

func TestFileSystemStore_Load_SkipsFallbackWhenExtensionPresent(t *testing.T) {
	dir := setupTestDir(t)
	s := NewFileSystemStore(dir)

	// "flat.grov" has .grov extension — should NOT try flat.grov.grov or flat.grov/flat.grov.grov
	data, err := s.Load("flat.grov")
	require.NoError(t, err)
	require.Equal(t, "flat", string(data))
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/store/ -v -run TestFileSystemStore`

Expected: `TestFileSystemStore_Load_ExactMatch`, `TestFileSystemStore_Load_ExactMatchNoExtension`, `TestFileSystemStore_Load_SkipsFallbackWhenExtensionPresent` PASS. `TestFileSystemStore_Load_GrovAppended`, `TestFileSystemStore_Load_DirectoryFallback`, `TestFileSystemStore_Load_DirectoryFallbackNested`, `TestFileSystemStore_Load_GrovAppendedPrefersOverDirectory` FAIL (fallback not implemented yet). Security tests PASS (existing behavior).

---

### Task 2: FileSystemStore Fallback — Implementation

**Files:**
- Modify: `internal/store/filesystem.go:26-51`

Implement the three-step fallback in `FileSystemStore.Load()`.

- [ ] **Step 1: Implement the fallback resolution**

Replace the `Load` method in `internal/store/filesystem.go` with:

```go
// Load reads the template at name (forward-slash path) from the store root.
// Resolution order:
//  1. Exact match: root/name
//  2. Append .grov: root/name.grov (only if name doesn't already end in .grov)
//  3. Directory fallback: root/name/basename.grov (only if name doesn't already end in .grov)
//
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

	// Nothing found — return error for the original name
	return nil, fmt.Errorf("template %q not found", name)
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/store/ -v -run TestFileSystemStore`

Expected: All 10 tests PASS.

- [ ] **Step 3: Run the full test suite to check nothing breaks**

Run: `go clean -testcache && go test ./... -v`

Expected: All existing tests PASS (existing templates use `.grov` extension so step 1 always matches).

- [ ] **Step 4: Commit**

```bash
git add internal/store/filesystem.go internal/store/filesystem_test.go
git commit -m "store: Add fallback resolution to FileSystemStore.Load()

Load now tries: exact match → name.grov → name/name.grov.
This supports folder-per-component layouts where components are
referenced by short paths like 'composites/card' instead of
'composites/card/card.grov'."
```

---

### Task 3: Update docs/components.md

**Files:**
- Modify: `docs/components.md`

Add two new sections before the existing "Using a Component" section: "Component Architecture" and "JS Colocation".

- [ ] **Step 1: Add Component Architecture section**

Insert the following at the top of `docs/components.md`, before the existing "Using a Component" heading (after the opening description paragraph):

```markdown
# Components

Components are reusable templates with a declared interface. They accept data through **props** and allow callers to inject content through **slots**.

## Component Architecture

Grove uses a two-tier classification for components:

### Primitives

Leaf components with no child components. They accept props and render self-contained HTML. Primitives do not use `{% component %}` internally and do not define `{% slot %}` tags.

Examples: buttons, badges, icons, inputs.

### Composites

Components that compose other components and/or use slots for flexible content injection. A composite uses `{% component %}` inside its template, defines `{% slot %}` tags, or both.

Examples: cards, navigation bars, author profiles.

**Decision rule:** If a component uses `{% component %}` or has `{% slot %}` tags, it's a composite. Otherwise it's a primitive.

### Folder Structure

Organize components into `primitives/` and `composites/` directories, with each component in its own folder:

` `` 
templates/
  primitives/
    button/
      button.grov
      button.js          ← optional JS for progressive enhancement
    tag-badge/
      tag-badge.grov
  composites/
    card/
      card.grov
    nav/
      nav.grov
      nav.js
` ``

### Path Resolution

When referencing components, use the short path without repeating the filename:

` ``jinja2
{% component "composites/card" %}
  ...
{% endcomponent %}
` ``

`FileSystemStore` resolves component paths in this order:

1. **Exact match** — `composites/card` (file exists as-is)
2. **Append .grov** — `composites/card.grov` (flat file without directory)
3. **Directory fallback** — `composites/card/card.grov` (folder-per-component)

This means flat-file components (without a directory) still work. The fallback only applies to names that don't already end in `.grov`.
```

- [ ] **Step 2: Add JS Colocation section**

Insert after the "Path Resolution" section, still before "Using a Component":

```markdown
## JS Colocation

Components can include a colocated JavaScript file for progressive enhancement. The JS file lives next to the `.grov` file with the same base name:

` ``
button/
  button.grov
  button.js
` ``

The JS file is always optional. Components without a `.js` file are perfectly valid.

### Including JS

Components declare their JS dependency using the `{% asset %}` tag:

` ``jinja2
{# primitives/button/button.grov #}
{% props label, href="#" %}
{% asset "/js/primitives/button/button.js" type="script" %}

<a href="{{ href }}" class="btn" data-button>{{ label }}</a>
` ``

The `RenderResult` asset system handles deduplication — if a page renders 10 buttons, the script is included once. The Go server serves JS files statically from the component directories.

### Progressive Enhancement

Server-rendered HTML must be functional without JavaScript. The JS enhances existing markup with smoother interactions, animations, or keyboard navigation. If JS fails to load, the component still works.

### Binding Convention

Components use `data-*` attributes to connect JS to markup. The template renders a `data-*` attribute, and the JS file queries for those attributes:

` ``js
// button.js
document.querySelectorAll('[data-button]').forEach(btn => {
  btn.addEventListener('click', () => {
    btn.classList.add('btn-loading');
    btn.setAttribute('aria-busy', 'true');
  });
});
` ``

No IDs, no class-name coupling — `data-*` attributes are the contract between template and script.
```

- [ ] **Step 3: Verify the existing content follows unchanged**

Ensure the rest of the file (from "## Using a Component" onward) is untouched.

- [ ] **Step 4: Commit**

```bash
git add docs/components.md
git commit -m "docs: Add atomic design conventions and JS colocation to components guide"
```

---

### Task 4: Restructure Blog Components — File Moves

**Files:**
- Move: `examples/blog/templates/components/*.grov` → `examples/blog/templates/primitives/` and `examples/blog/templates/composites/`
- Create: `examples/blog/templates/primitives/button/button.grov`
- Create: `examples/blog/templates/primitives/button/button.js`
- Create: `examples/blog/templates/composites/nav/nav.js`

Move existing components into the new directory structure and create new files.

- [ ] **Step 1: Create directory structure**

```bash
cd /home/theo/Work/grove/examples/blog/templates
mkdir -p primitives/tag-badge primitives/button primitives/footer
mkdir -p composites/card composites/nav composites/author-card composites/breadcrumbs
```

- [ ] **Step 2: Move existing component files**

```bash
cd /home/theo/Work/grove/examples/blog/templates
mv components/tag-badge.grov primitives/tag-badge/tag-badge.grov
mv components/footer.grov primitives/footer/footer.grov
mv components/card.grov composites/card/card.grov
mv components/nav.grov composites/nav/nav.grov
mv components/author-card.grov composites/author-card/author-card.grov
mv components/breadcrumbs.grov composites/breadcrumbs/breadcrumbs.grov
rmdir components
```

- [ ] **Step 3: Create button primitive with JS colocation**

Create `examples/blog/templates/primitives/button/button.grov`:

```jinja2
{% props label, href="#", variant="primary", type="link" %}
{% asset "/js/primitives/button/button.js" type="script" %}

{% if type == "link" %}
  <a href="{{ href }}" class="btn btn-{{ variant }}" data-button>{{ label }}</a>
{% else %}
  <button type="{{ type }}" class="btn btn-{{ variant }}" data-button>{{ label }}</button>
{% endif %}
```

Create `examples/blog/templates/primitives/button/button.js`:

```js
document.querySelectorAll('[data-button]').forEach(function (btn) {
  btn.addEventListener('click', function () {
    if (btn.classList.contains('btn-loading')) return;
    btn.classList.add('btn-loading');
    btn.setAttribute('aria-busy', 'true');
  });
});
```

- [ ] **Step 4: Create nav.js for mobile menu toggle**

Create `examples/blog/templates/composites/nav/nav.js`:

```js
document.querySelectorAll('[data-nav-toggle]').forEach(function (toggle) {
  var nav = toggle.closest('[data-nav]');
  if (!nav) return;
  var links = nav.querySelector('[data-nav-links]');
  if (!links) return;

  toggle.addEventListener('click', function () {
    var expanded = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!expanded));
    links.classList.toggle('nav-links-open');
  });
});
```

- [ ] **Step 5: Update nav.grov to support mobile toggle and declare its JS asset**

Replace the content of `examples/blog/templates/composites/nav/nav.grov` with:

```jinja2
{% props site_name %}
{% asset "/js/composites/nav/nav.js" type="script" %}

<nav class="nav" data-nav>
  <a href="/" class="nav-brand">{{ site_name }}</a>
  <button class="nav-toggle" data-nav-toggle aria-expanded="false" aria-label="Menu">&#9776;</button>
  <div class="nav-links" data-nav-links>
    <a href="/" class="nav-link">Home</a>
    <a href="/tags" class="nav-link">Tags</a>
    {% slot %}{% endslot %}
  </div>
</nav>
```

- [ ] **Step 6: Commit file moves and new files**

```bash
git add examples/blog/templates/primitives/ examples/blog/templates/composites/
git add -u examples/blog/templates/components/
git commit -m "blog: Restructure components into primitives/ and composites/ directories

Moves existing components into atomic design folder structure.
Adds button primitive with JS colocation and nav.js for mobile toggle."
```

---

### Task 5: Update Blog Template References

**Files:**
- Modify: `examples/blog/templates/base.grov`
- Modify: `examples/blog/templates/index.grov`
- Modify: `examples/blog/templates/post.grov`
- Modify: `examples/blog/templates/post-list.grov`
- Modify: `examples/blog/templates/author.grov`
- Modify: `examples/blog/templates/tag-list.grov`

Update all `{% component %}` and `{% include %}` references to use new paths.

- [ ] **Step 1: Update base.grov**

Change `components/nav.grov` → `composites/nav` and `components/footer.grov` → `primitives/footer`:

```jinja2
{% component "composites/nav" site_name=site_name %}{% endcomponent %}
```

```jinja2
{% component "primitives/footer" year=current_year %}{% endcomponent %}
```

- [ ] **Step 2: Update index.grov**

Change `components/card.grov` → `composites/card` and `components/tag-badge.grov` → `primitives/tag-badge`:

```jinja2
{% component "composites/card" title=post.title summary=post.summary href="/post/" ~ post.slug date=post.date author_name=post.author.name author_slug=post.author.slug %}
```

```jinja2
{% component "primitives/tag-badge" label=tag.name color=tag.color slug=tag.slug %}{% endcomponent %}
```

- [ ] **Step 3: Update post.grov**

Change all component references. Replace:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/tag-badge.grov" ...` → `{% component "primitives/tag-badge" ...`
- `{% component "components/author-card.grov" ...` → `{% component "composites/author-card" ...`
- `{% component "components/card.grov" ...` → `{% component "composites/card" ...`

- [ ] **Step 4: Update post-list.grov**

Replace:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/card.grov" ...` → `{% component "composites/card" ...`
- `{% component "components/tag-badge.grov" ...` → `{% component "primitives/tag-badge" ...`

- [ ] **Step 5: Update author.grov**

Replace:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/author-card.grov" ...` → `{% component "composites/author-card" ...`
- `{% component "components/card.grov" ...` → `{% component "composites/card" ...`
- `{% component "components/tag-badge.grov" ...` → `{% component "primitives/tag-badge" ...`

- [ ] **Step 6: Update tag-list.grov**

Replace:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`

- [ ] **Step 7: Update blog main.go to serve JS from templates directory**

Add a JS file server route in `examples/blog/main.go` after the existing static handler:

```go
// Serve colocated JS from component directories
r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir(templateDir))))
```

- [ ] **Step 8: Commit**

```bash
git add examples/blog/
git commit -m "blog: Update all template references to atomic design paths

Updates component/include paths to use primitives/ and composites/
short paths. Adds JS static file serving for colocated scripts."
```

---

### Task 6: Restructure Store Components

**Files:**
- Move: `examples/store/templates/components/product-card.grov` → `examples/store/templates/composites/product-card/product-card.grov`
- Move: `examples/store/templates/components/breadcrumbs.grov` → `examples/store/templates/composites/breadcrumbs/breadcrumbs.grov`
- Create: `examples/store/templates/primitives/button/button.grov`
- Create: `examples/store/templates/primitives/button/button.js`
- Modify: `examples/store/templates/index.grov`
- Modify: `examples/store/templates/product-list.grov`
- Modify: `examples/store/templates/category.grov`
- Modify: `examples/store/templates/search.grov`
- Modify: `examples/store/templates/product.grov`
- Modify: `examples/store/templates/cart.grov`
- Modify: `examples/store/main.go`

- [ ] **Step 1: Create directory structure and move files**

```bash
cd /home/theo/Work/grove/examples/store/templates
mkdir -p primitives/button composites/product-card composites/breadcrumbs
mv components/product-card.grov composites/product-card/product-card.grov
mv components/breadcrumbs.grov composites/breadcrumbs/breadcrumbs.grov
rmdir components
```

- [ ] **Step 2: Create button primitive (same pattern as blog)**

Create `examples/store/templates/primitives/button/button.grov`:

```jinja2
{% props label, href="#", variant="primary", type="link" %}
{% asset "/js/primitives/button/button.js" type="script" %}

{% if type == "link" %}
  <a href="{{ href }}" class="btn btn-{{ variant }}" data-button>{{ label }}</a>
{% else %}
  <button type="{{ type }}" class="btn btn-{{ variant }}" data-button>{{ label }}</button>
{% endif %}
```

Create `examples/store/templates/primitives/button/button.js`:

```js
document.querySelectorAll('[data-button]').forEach(function (btn) {
  btn.addEventListener('click', function () {
    if (btn.classList.contains('btn-loading')) return;
    btn.classList.add('btn-loading');
    btn.setAttribute('aria-busy', 'true');
  });
});
```

- [ ] **Step 3: Update all template references**

In **index.grov** change:
- `{% component "components/product-card.grov" ...` → `{% component "composites/product-card" ...`

In **product-list.grov** change:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/product-card.grov" ...` → `{% component "composites/product-card" ...`

In **category.grov** change:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/product-card.grov" ...` → `{% component "composites/product-card" ...`

In **search.grov** change:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/product-card.grov" ...` → `{% component "composites/product-card" ...`

In **product.grov** change:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`
- `{% component "components/product-card.grov" ...` → `{% component "composites/product-card" ...`

In **cart.grov** change:
- `{% include "components/breadcrumbs.grov" %}` → `{% include "composites/breadcrumbs/breadcrumbs.grov" %}`

- [ ] **Step 4: Update store main.go to serve JS**

Add a JS file server route in `examples/store/main.go` after the existing static handler:

```go
// Serve colocated JS from component directories
r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir(templateDir))))
```

- [ ] **Step 5: Commit**

```bash
git add examples/store/
git add -u examples/store/templates/components/
git commit -m "store: Restructure components into atomic design layout

Moves product-card and breadcrumbs into composites/.
Adds button primitive with JS colocation.
Updates all template references to new paths."
```

---

### Task 7: Verify Everything Works

**Files:** None (verification only)

- [ ] **Step 1: Run the full test suite**

Run: `go clean -testcache && go test ./... -v`

Expected: All tests PASS including new FileSystemStore tests.

- [ ] **Step 2: Verify build**

Run: `go build ./...`

Expected: Clean build with no errors.

- [ ] **Step 3: Verify blog example directory structure**

Run: `find examples/blog/templates -type f | sort`

Expected output:
```
examples/blog/templates/author.grov
examples/blog/templates/base.grov
examples/blog/templates/composites/author-card/author-card.grov
examples/blog/templates/composites/breadcrumbs/breadcrumbs.grov
examples/blog/templates/composites/card/card.grov
examples/blog/templates/composites/nav/nav.grov
examples/blog/templates/composites/nav/nav.js
examples/blog/templates/index.grov
examples/blog/templates/post-list.grov
examples/blog/templates/post.grov
examples/blog/templates/primitives/button/button.grov
examples/blog/templates/primitives/button/button.js
examples/blog/templates/primitives/footer/footer.grov
examples/blog/templates/primitives/tag-badge/tag-badge.grov
examples/blog/templates/tag-list.grov
```

- [ ] **Step 4: Verify store example directory structure**

Run: `find examples/store/templates -type f | sort`

Expected output:
```
examples/store/templates/base.grov
examples/store/templates/cart.grov
examples/store/templates/category.grov
examples/store/templates/composites/breadcrumbs/breadcrumbs.grov
examples/store/templates/composites/product-card/product-card.grov
examples/store/templates/index.grov
examples/store/templates/macros/filters.grov
examples/store/templates/macros/pricing.grov
examples/store/templates/primitives/button/button.grov
examples/store/templates/primitives/button/button.js
examples/store/templates/product-list.grov
examples/store/templates/product.grov
examples/store/templates/search.grov
```

- [ ] **Step 5: Verify no references to old component paths remain**

Run: `grep -r "components/" examples/blog/templates/ examples/store/templates/`

Expected: No matches (all old `components/` references have been updated).
