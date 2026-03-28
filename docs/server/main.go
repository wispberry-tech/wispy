// Wisp Template Engine - Example HTTP Server
//
// An interactive web server that demonstrates the Wisp template engine
// by showing the output of each pipeline stage: lexer, parser, and renderer.
//
// Usage:
//
//	go run ./examples/server
//
// Then open http://localhost:8080 in your browser.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"template-wisp/internal/ast"
	"template-wisp/internal/lexer"
	"template-wisp/internal/parser"
	"template-wisp/pkg/engine"
)

// tmplStore implements store.TemplateStore for .tmpl files.
type tmplStore struct {
	baseDir string
}

func (s *tmplStore) ReadTemplate(name string) ([]byte, error) {
	path := filepath.Join(s.baseDir, name+".tmpl")
	return os.ReadFile(path)
}

func (s *tmplStore) ListTemplates() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".tmpl" {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.SetLevel(log.DebugLevel)
	log.SetReportTimestamp(true)

	// Create documentation engine with .tmpl file store
	tmplDir := filepath.Join(".", "docs", "server", "templates")
	if _, err := os.Stat(tmplDir); os.IsNotExist(err) {
		tmplDir = filepath.Join("templates")
	}
	docEngine := engine.NewWithStore(&tmplStore{baseDir: tmplDir})

	mux := http.NewServeMux()

	mux.HandleFunc("/api/tokens", handleAPITokens)
	mux.HandleFunc("/api/ast", handleAPIAST)
	mux.HandleFunc("/api/render", handleAPIRender)

	// Documentation routes
	mux.HandleFunc("/docs/", handleDocs(docEngine))
	mux.HandleFunc("/docs/tags", handleTagsIndex(docEngine))
	mux.HandleFunc("/docs/tags/", handleTestTag(docEngine))
	mux.HandleFunc("/docs/filters", handleFiltersIndex(docEngine))
	mux.HandleFunc("/docs/filters/", handleTestFilter(docEngine))

	addr := ":" + port
	server := &http.Server{
		Addr:         addr,
		Handler:      logMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info("Wisp Example Server starting", "addr", "http://localhost:"+port)
	log.Info("Routes available",
		"/api/tokens", "JSON token stream",
		"/api/ast", "JSON AST",
		"/api/render", "JSON render result",
		"/docs/", "Documentation pages",
		"/docs/tags/", "Tag examples",
		"/docs/filters/", "Filter examples",
	)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed", "error", err)
	}
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

// API handlers
type TokenInfo struct {
	Type    string `json:"type"`
	Literal string `json:"literal"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
}

func handleAPITokens(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	if template == "" {
		template = `{% .name %}`
	}

	log.Debug("Tokenizing template", "template", template)

	l := lexer.NewLexer(template)
	var tokens []TokenInfo
	for {
		tok := l.NextToken()
		tokens = append(tokens, TokenInfo{
			Type:    string(tok.Type),
			Literal: tok.Literal,
			Line:    tok.Line,
			Column:  tok.Column,
		})
		if tok.Type == lexer.EOF {
			break
		}
	}

	log.Debug("Tokenization complete", "count", len(tokens))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"template": template,
		"tokens":   tokens,
	})
}

type ASTNode struct {
	Type     string    `json:"type"`
	String   string    `json:"string"`
	Children []ASTNode `json:"children,omitempty"`
}

func handleAPIAST(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	if template == "" {
		template = `{% .name %}`
	}

	log.Debug("Parsing template", "template", template)

	l := lexer.NewLexer(template)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	var astNodes []ASTNode
	for _, stmt := range program.Statements {
		astNodes = append(astNodes, astToNode(stmt))
	}

	log.Debug("Parsing complete",
		"statements", len(program.Statements),
		"errors", len(p.Errors()),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"template": template,
		"nodes":    astNodes,
		"errors":   p.Errors(),
	})
}

type RenderResult struct {
	Template string `json:"template"`
	Data     string `json:"data"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

func handleAPIRender(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	if template == "" {
		template = `<h1>Hello {% .name %}!</h1>`
	}
	dataJSON := r.URL.Query().Get("data")
	if dataJSON == "" {
		dataJSON = `{"name": "World"}`
	}

	log.Debug("Rendering template", "template", template, "data", dataJSON)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		log.Warn("Failed to parse data JSON", "error", err)
		data = make(map[string]interface{})
	}

	// Use the engine for rendering
	e := engine.New()
	output, err := e.RenderString(template, data)

	result := RenderResult{
		Template: template,
		Data:     dataJSON,
	}

	if err != nil {
		result.Error = err.Error()
		log.Warn("Render error", "error", err)
	} else {
		result.Output = output
		log.Debug("Render complete", "output_len", len(output))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func astToNode(stmt ast.Statement) ASTNode {
	node := ASTNode{
		Type:   fmt.Sprintf("%T", stmt),
		String: stmt.String(),
	}

	switch s := stmt.(type) {
	case *ast.IfStatement:
		if s.Consequence != nil {
			for _, cs := range s.Consequence.Statements {
				node.Children = append(node.Children, astToNode(cs))
			}
		}
	case *ast.ForStatement:
		if s.Body != nil {
			for _, bs := range s.Body.Statements {
				node.Children = append(node.Children, astToNode(bs))
			}
		}
	case *ast.CaseStatement:
		if s.Body != nil {
			for _, bs := range s.Body.Statements {
				node.Children = append(node.Children, astToNode(bs))
			}
		}
	}

	return node
}

// Documentation handler factories

func handleDocs(docEngine *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/docs/")
		if name == "" {
			name = "getting-started"
		}
		name = "/" + name
		renderDoc(w, r, docEngine, name)
	}
}

func handleTagsIndex(docEngine *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderDoc(w, r, docEngine, "/tags/index")
	}
}

func handleFiltersIndex(docEngine *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderDoc(w, r, docEngine, "/filters/index")
	}
}

func handleTestTag(docEngine *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/docs/tags/")
		name = "/tags/" + name
		renderDoc(w, r, docEngine, name)
	}
}

func handleTestFilter(docEngine *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/docs/filters/")
		name = "/filters/" + name
		renderDoc(w, r, docEngine, name)
	}
}

func renderDoc(w http.ResponseWriter, r *http.Request, docEngine *engine.Engine, name string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Templates already contain {% extends "layouts/doc-layout" %}, render them directly
	output, err := docEngine.RenderFile(name, map[string]interface{}{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "<h1>Render Error</h1><pre>%s</pre>", err.Error())
		log.Warn("Render error", "name", name, "error", err)
		return
	}

	fmt.Fprint(w, output)
}
