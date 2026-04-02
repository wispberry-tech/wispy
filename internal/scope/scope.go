// internal/scope/scope.go
package scope

// Scope is a single frame in the variable lookup chain.
// Variables are looked up local-first, then parent, then parent's parent, etc.
type Scope struct {
	vars   map[string]any
	parent *Scope
}

// New creates a new Scope with an optional parent.
func New(parent *Scope) *Scope {
	return &Scope{vars: make(map[string]any), parent: parent}
}

// NewWithSize creates a scope with a pre-sized variable map.
func NewWithSize(parent *Scope, size int) *Scope {
	return &Scope{vars: make(map[string]any, size), parent: parent}
}

// Reset clears all variables and sets a new parent, reusing the map memory.
func (s *Scope) Reset(parent *Scope) {
	for k := range s.vars {
		delete(s.vars, k)
	}
	s.parent = parent
}

// Set stores key=value in this scope frame.
func (s *Scope) Set(key string, value any) {
	s.vars[key] = value
}

// Get looks up key in this scope and all parent scopes.
func (s *Scope) Get(key string) (any, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if v, ok := cur.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

// SetParent sets the parent scope (used during scope setup in Execute).
func (s *Scope) SetParent(p *Scope) {
	s.parent = p
}

// Parent returns the parent scope, or nil if this is the root scope.
func (s *Scope) Parent() *Scope {
	return s.parent
}

// ForEach calls fn for each key/value pair in this scope's own bindings (not parent).
func (s *Scope) ForEach(fn func(key string, val any)) {
	for k, v := range s.vars {
		fn(k, v)
	}
}
