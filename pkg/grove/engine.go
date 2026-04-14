// pkg/wispy/engine.go
package grove

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wispberry-tech/grove/internal/ast"
	"github.com/wispberry-tech/grove/internal/compiler"
	"github.com/wispberry-tech/grove/internal/filters"
	"github.com/wispberry-tech/grove/internal/groverrors"
	"github.com/wispberry-tech/grove/internal/lexer"
	"github.com/wispberry-tech/grove/internal/parser"
	"github.com/wispberry-tech/grove/internal/store"
	"github.com/wispberry-tech/grove/internal/vm"
)

// AssetResolver maps a logical asset name to a served URL. Returns (url, true)
// if resolved, ("", false) to fall through to the original name. Configure
// one via WithAssetResolver; typically callers pass assets.Manifest.Resolve.
type AssetResolver = vm.AssetResolver

// Option configures an Engine at creation time.
type Option func(*engineCfg)

type engineCfg struct {
	strictVariables bool
	store           store.Store
	cacheSize       int // 0 = use default (512)
	sandbox         *SandboxConfig
	assetResolver   AssetResolver
}

// SandboxConfig restricts what templates can do.
type SandboxConfig struct {
	// AllowedTags: nil = all allowed; non-nil = only listed tags permitted (ParseError otherwise).
	AllowedTags []string
	// AllowedFilters: nil = all allowed; non-nil = only listed filters permitted (ParseError otherwise).
	AllowedFilters []string
	// MaxLoopIter: maximum total loop iterations per render pass. 0 = unlimited.
	MaxLoopIter int
}

// WithStrictVariables makes undefined variable references return a RuntimeError.
func WithStrictVariables(strict bool) Option {
	return func(c *engineCfg) { c.strictVariables = strict }
}

// WithStore sets the template store used by Render(), include, render, and import.
func WithStore(s store.Store) Option {
	return func(c *engineCfg) { c.store = s }
}

// WithCacheSize sets the maximum number of compiled bytecode entries in the LRU cache.
// Default: 512. Pass 0 to use the default.
func WithCacheSize(n int) Option {
	return func(c *engineCfg) { c.cacheSize = n }
}

// WithSandbox applies sandbox restrictions to all templates rendered by this engine.
func WithSandbox(cfg SandboxConfig) Option {
	return func(c *engineCfg) { c.sandbox = &cfg }
}

// WithAssetResolver configures the engine to resolve {% asset %} logical
// names through the given function at render time. Pass a nil resolver, or
// omit this option, to disable resolution (default).
//
// Typical usage: WithAssetResolver(manifest.Resolve).
func WithAssetResolver(r AssetResolver) Option {
	return func(c *engineCfg) { c.assetResolver = r }
}

// Engine is the Wispy template engine. Create with New(). Safe for concurrent use.
type Engine struct {
	cfg       engineCfg
	globals   map[string]any
	filters   map[string]any         // vm.FilterFn | *vm.FilterDef (source of truth)
	filterFns map[string]vm.FilterFn // resolved filter functions (hot-path lookup)
	cache     *lruCache

	// assetResolver holds an atomic.Pointer so SetAssetResolver can swap it
	// concurrently with active renders without locking. A nil pointer means
	// "no resolver" — the hot path is a single atomic load + nil check.
	assetResolver atomic.Pointer[AssetResolver]

	// refAssets records logical asset names seen during rendering. Lazy-
	// allocated — nil when no resolver is configured, so apps not using the
	// pipeline pay nothing. Guarded by refMu.
	refMu     sync.Mutex
	refAssets map[string]struct{}
}

// New creates a configured Engine.
func New(opts ...Option) *Engine {
	e := &Engine{
		globals:   make(map[string]any),
		filters:   make(map[string]any),
		filterFns: make(map[string]vm.FilterFn),
	}
	for _, o := range opts {
		o(&e.cfg)
	}
	cacheSize := e.cfg.cacheSize
	if cacheSize <= 0 {
		cacheSize = 512
	}
	e.cache = newLRUCache(cacheSize)

	if e.cfg.assetResolver != nil {
		r := e.cfg.assetResolver
		e.assetResolver.Store(&r)
	}

	e.RegisterFilter("safe", vm.FilterFn(func(v vm.Value, _ []vm.Value) (vm.Value, error) {
		return vm.SafeHTMLVal(v.String()), nil
	}))
	for name, fn := range filters.Builtins() {
		e.RegisterFilter(name, fn)
	}
	return e
}

// SetGlobal registers a value available in all render calls.
func (e *Engine) SetGlobal(key string, value any) { e.globals[key] = value }

// RegisterFilter registers a custom filter function.
func (e *Engine) RegisterFilter(name string, fn any) {
	e.filters[name] = fn
	// Pre-resolve to FilterFn for hot-path lookup.
	switch f := fn.(type) {
	case vm.FilterFn:
		e.filterFns[name] = f
	case func(vm.Value, []vm.Value) (vm.Value, error):
		e.filterFns[name] = vm.FilterFn(f)
	case *vm.FilterDef:
		e.filterFns[name] = f.Fn
	}
}

// RenderTemplate compiles and renders an inline template string.
func (e *Engine) RenderTemplate(ctx context.Context, src string, data Data) (RenderResult, error) {
	tokens, err := lexer.Tokenize(src)
	if err != nil {
		line := 0
		type liner interface{ LexLine() int }
		if le, ok := err.(liner); ok {
			line = le.LexLine()
		}
		return RenderResult{}, &groverrors.ParseError{Message: err.Error(), Line: line}
	}

	prog, err := parser.Parse(tokens, true, e.allowedTagsMap())
	if err != nil {
		return RenderResult{}, err
	}

	bc, err := e.compileChecked(prog)
	if err != nil {
		return RenderResult{}, err
	}

	er, err := vm.Execute(ctx, bc, map[string]any(data), e, "")
	if err != nil {
		return RenderResult{}, wrapRuntimeErr(err, "")
	}
	result := resultFromExecute(er)
	if err := processGroveDirectives(&result, data); err != nil {
		return RenderResult{}, err
	}
	return result, nil
}

// Render compiles and renders a named template from the engine's store.
func (e *Engine) Render(ctx context.Context, name string, data Data) (RenderResult, error) {
	bc, err := e.LoadTemplate(name)
	if err != nil {
		return RenderResult{}, err
	}
	er, err := vm.Execute(ctx, bc, map[string]any(data), e, name)
	if err != nil {
		return RenderResult{}, wrapRuntimeErr(err, name)
	}
	result := resultFromExecute(er)
	if err := processGroveDirectives(&result, data); err != nil {
		return RenderResult{}, err
	}
	return result, nil
}

// RenderTo renders a named template and writes the body to w.
func (e *Engine) RenderTo(ctx context.Context, name string, data Data, w io.Writer) error {
	result, err := e.Render(ctx, name, data)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, result.Body)
	return err
}

// LoadTemplate loads, lexes, parses, and compiles a named template from the store.
// Results are cached by name in the LRU cache. Implements vm.EngineIface.
func (e *Engine) LoadTemplate(name string) (*compiler.Bytecode, error) {
	if bc, ok := e.cache.get(name); ok {
		return bc, nil
	}
	if e.cfg.store == nil {
		return nil, fmt.Errorf("no store configured — use grove.WithStore() to load named templates")
	}
	src, err := e.cfg.store.Load(name)
	if err != nil {
		// Try with .html extension for import resolution
		src2, err2 := e.cfg.store.Load(name + ".html")
		if err2 != nil {
			return nil, err // return original error
		}
		src = src2
		name = name + ".html"
	}
	tokens, err := lexer.Tokenize(string(src))
	if err != nil {
		return nil, &groverrors.ParseError{Message: err.Error()}
	}
	prog, err := parser.Parse(tokens, false, e.allowedTagsMap())
	if err != nil {
		return nil, err
	}
	bc, err := e.compileChecked(prog)
	if err != nil {
		return nil, err
	}
	e.cache.set(name, bc)
	return bc, nil
}

// compileChecked compiles a program and enforces AllowedFilters sandbox restriction.
func (e *Engine) compileChecked(prog *ast.Program) (*compiler.Bytecode, error) {
	bc, err := compiler.Compile(prog)
	if err != nil {
		return nil, &groverrors.ParseError{Message: err.Error()}
	}
	if e.cfg.sandbox != nil && e.cfg.sandbox.AllowedFilters != nil {
		allowed := make(map[string]bool, len(e.cfg.sandbox.AllowedFilters))
		for _, f := range e.cfg.sandbox.AllowedFilters {
			allowed[f] = true
		}
		if err := checkAllowedFilters(bc, allowed); err != nil {
			return nil, err
		}
	}
	// Pre-compile constants so the first render is as fast as subsequent ones.
	vm.PrecompileConsts(bc)
	return bc, nil
}

// checkAllowedFilters walks bc and all sub-bytecodes checking OP_FILTER instructions.
func checkAllowedFilters(bc *compiler.Bytecode, allowed map[string]bool) error {
	for _, instr := range bc.Instrs {
		if instr.Op == compiler.OP_FILTER {
			name := bc.Names[instr.A]
			if !allowed[name] {
				return &groverrors.ParseError{Message: fmt.Sprintf("sandbox: filter %q is not allowed", name)}
			}
		}
	}
	// Recurse into sub-bytecodes
	for i := range bc.Macros {
		if err := checkAllowedFilters(bc.Macros[i].Body, allowed); err != nil {
			return err
		}
	}
	for i := range bc.Blocks {
		if err := checkAllowedFilters(bc.Blocks[i].Body, allowed); err != nil {
			return err
		}
	}
	for i := range bc.Components {
		for j := range bc.Components[i].Fills {
			if err := checkAllowedFilters(bc.Components[i].Fills[j].Body, allowed); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── vm.EngineIface implementation ───────────────────────────────────────────

func (e *Engine) LookupFilter(name string) (vm.FilterFn, bool) {
	fn, ok := e.filterFns[name]
	return fn, ok
}

func (e *Engine) StrictVariables() bool      { return e.cfg.strictVariables }
func (e *Engine) GlobalData() map[string]any { return e.globals }

// MaxLoopIter returns the sandbox max loop iteration limit (0 = unlimited).
func (e *Engine) MaxLoopIter() int {
	if e.cfg.sandbox != nil {
		return e.cfg.sandbox.MaxLoopIter
	}
	return 0
}

// AssetResolver returns the currently configured asset resolver, or nil when
// unused. Safe to call concurrently with SetAssetResolver.
func (e *Engine) AssetResolver() AssetResolver {
	p := e.assetResolver.Load()
	if p == nil {
		return nil
	}
	return *p
}

// SetAssetResolver atomically swaps the asset resolver. Safe for concurrent
// use; designed for watch mode where the resolver updates on file changes
// while the engine continues serving requests. Pass nil to disable.
func (e *Engine) SetAssetResolver(r AssetResolver) {
	if r == nil {
		e.assetResolver.Store(nil)
		return
	}
	e.assetResolver.Store(&r)
}

// RecordAssetRef records a logical asset name seen during rendering. No-op
// when no resolver is configured — the map is not allocated, so apps that
// don't use the pipeline pay nothing.
func (e *Engine) RecordAssetRef(logicalName string) {
	if e.assetResolver.Load() == nil {
		return
	}
	e.refMu.Lock()
	if e.refAssets == nil {
		e.refAssets = make(map[string]struct{})
	}
	e.refAssets[logicalName] = struct{}{}
	e.refMu.Unlock()
}

// ReferencedAssets returns a snapshot of logical asset names seen via
// OP_ASSET since the engine started (or since ResetReferencedAssets was
// called). The returned map is a copy — safe to mutate.
func (e *Engine) ReferencedAssets() map[string]struct{} {
	e.refMu.Lock()
	defer e.refMu.Unlock()
	out := make(map[string]struct{}, len(e.refAssets))
	for k := range e.refAssets {
		out[k] = struct{}{}
	}
	return out
}

// ResetReferencedAssets clears the referenced-name set.
func (e *Engine) ResetReferencedAssets() {
	e.refMu.Lock()
	e.refAssets = nil
	e.refMu.Unlock()
}

// ─── grove:data / grove:nowarn post-processing ──────────────────────────────

// processGroveDirectives handles grove:data and grove:nowarn attributes in the rendered output.
func processGroveDirectives(result *RenderResult, data Data) error {
	body := result.Body

	// Process grove:nowarn — strip attribute from output
	for {
		idx := strings.Index(body, " grove:nowarn=")
		if idx < 0 {
			break
		}
		end := findAttrEnd(body, idx+len(" grove:nowarn="))
		body = body[:idx] + body[end:]
	}

	// Process grove:data — resolve variables and merge into x-data
	for {
		idx := strings.Index(body, " grove:data=")
		if idx < 0 {
			break
		}
		attrEnd := findAttrEnd(body, idx+len(" grove:data="))
		attrVal := extractQuotedValue(body[idx+len(" grove:data=") : attrEnd])

		// Find the element this attribute belongs to (scan backwards for <)
		elemStart := strings.LastIndex(body[:idx], "<")
		if elemStart < 0 {
			return fmt.Errorf("grove:data found outside an HTML element")
		}

		// Find x-data on the same element
		elemEnd := strings.Index(body[elemStart:], ">")
		if elemEnd < 0 {
			return fmt.Errorf("unclosed HTML element with grove:data")
		}
		elemEnd += elemStart + 1
		elemContent := body[elemStart:elemEnd]

		xdataIdx := strings.Index(elemContent, " x-data=")
		if xdataIdx < 0 {
			return fmt.Errorf("grove:data requires x-data on the same element")
		}

		// Parse variable names
		varNames := strings.Split(attrVal, ",")
		var jsonParts []string
		for _, name := range varNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			val, ok := data[name]
			if !ok {
				return fmt.Errorf("grove:data: variable %q not found", name)
			}
			jsonBytes, err := jsonMarshal(val)
			if err != nil {
				jsonBytes = []byte("null")
			}
			jsonParts = append(jsonParts, name+": "+string(jsonBytes))
		}

		// Merge into x-data: prepend variables to the existing object
		xdataAbsIdx := elemStart + xdataIdx + len(" x-data=")
		xdataEnd := findAttrEnd(body, xdataAbsIdx)
		xdataVal := extractQuotedValue(body[xdataAbsIdx:xdataEnd])

		// Merge: inject vars at the start of the x-data object
		merged := xdataVal
		if strings.HasPrefix(strings.TrimSpace(xdataVal), "{") {
			inner := strings.TrimSpace(xdataVal)
			inner = strings.TrimPrefix(inner, "{")
			inner = strings.TrimSuffix(inner, "}")
			inner = strings.TrimSpace(inner)
			if inner != "" {
				merged = "{ " + strings.Join(jsonParts, ", ") + ", " + inner + " }"
			} else {
				merged = "{ " + strings.Join(jsonParts, ", ") + " }"
			}
		}

		// Rebuild: remove grove:data attr, replace x-data value
		body = body[:idx] + body[attrEnd:] // remove grove:data
		// Recalculate x-data position (shifted after removing grove:data)
		shift := attrEnd - idx
		newXdataAbsIdx := xdataAbsIdx - shift
		newXdataEnd := xdataEnd - shift
		if newXdataAbsIdx > idx {
			// x-data was after grove:data in the element
			newXdataAbsIdx -= 0 // already shifted
		}
		// Replace x-data value
		xdataQuoteStart := newXdataAbsIdx
		xdataQuoteEnd := newXdataEnd
		body = body[:xdataQuoteStart] + `"` + merged + `"` + body[xdataQuoteEnd:]
	}

	result.Body = body
	return nil
}

// findAttrEnd finds the end of an HTML attribute value starting at pos (after the =).
// Handles both "quoted" and 'single-quoted' values.
func findAttrEnd(s string, pos int) int {
	if pos >= len(s) {
		return pos
	}
	quote := s[pos]
	if quote != '"' && quote != '\'' {
		return pos
	}
	end := strings.IndexByte(s[pos+1:], quote)
	if end < 0 {
		return len(s)
	}
	return pos + end + 2 // include closing quote
}

// extractQuotedValue extracts the value from "value" or 'value'.
func extractQuotedValue(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// jsonMarshal serializes a Go value to JSON.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (e *Engine) allowedTagsMap() map[string]bool {
	if e.cfg.sandbox == nil || e.cfg.sandbox.AllowedTags == nil {
		return nil
	}
	m := make(map[string]bool, len(e.cfg.sandbox.AllowedTags))
	for _, t := range e.cfg.sandbox.AllowedTags {
		m[t] = true
	}
	return m
}

func wrapRuntimeErr(err error, templateName string) error {
	if _, ok := err.(*groverrors.RuntimeError); ok {
		return err
	}
	return &groverrors.RuntimeError{Template: templateName, Message: err.Error()}
}

func resultFromExecute(er vm.ExecuteResult) RenderResult {
	if er.RC == nil {
		return RenderResult{Body: er.Body}
	}
	rc := er.RC
	r := RenderResult{
		Body:    er.Body,
		Meta:    rc.ExportMeta(),
		Hoisted: rc.ExportHoisted(),
	}
	for _, a := range rc.ExportAssets() {
		r.Assets = append(r.Assets, Asset{
			Src:      a.Src,
			Type:     a.Type,
			Attrs:    a.Attrs,
			Priority: a.Priority,
		})
	}
	for _, msg := range rc.ExportWarnings() {
		r.Warnings = append(r.Warnings, Warning{Message: msg})
	}
	return r
}

// ─── LRU cache ────────────────────────────────────────────────────────────────

type lruEntry struct {
	name       string
	bc         *compiler.Bytecode
	prev, next *lruEntry
}

type lruCache struct {
	mu      sync.Mutex
	cap     int
	entries map[string]*lruEntry
	head    *lruEntry // most recently used
	tail    *lruEntry // least recently used
}

func newLRUCache(cap int) *lruCache {
	return &lruCache{cap: cap, entries: make(map[string]*lruEntry)}
}

func (c *lruCache) get(name string) (*compiler.Bytecode, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[name]
	if !ok {
		return nil, false
	}
	c.moveToHead(e)
	return e.bc, true
}

func (c *lruCache) set(name string, bc *compiler.Bytecode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[name]; ok {
		e.bc = bc
		c.moveToHead(e)
		return
	}
	e := &lruEntry{name: name, bc: bc}
	c.entries[name] = e
	c.addToHead(e)
	if len(c.entries) > c.cap {
		c.evictTail()
	}
}

func (c *lruCache) addToHead(e *lruEntry) {
	e.next = c.head
	e.prev = nil
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *lruCache) moveToHead(e *lruEntry) {
	if e == c.head {
		return
	}
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == c.tail {
		c.tail = e.prev
	}
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
}

func (c *lruCache) evictTail() {
	if c.tail == nil {
		return
	}
	old := c.tail
	delete(c.entries, old.name)
	c.tail = old.prev
	if c.tail != nil {
		c.tail.next = nil
	} else {
		c.head = nil
	}
}
