package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Language represents a programming language
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangJavaScript Language = "javascript"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangJava       Language = "java"
	LangCSharp     Language = "csharp"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
	LangJSON       Language = "json"
	LangText       Language = "text"
	LangUnknown    Language = "unknown"
)

// Symbol represents a parsed code symbol
type Symbol struct {
	Name        string
	Kind        string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Scope       string
	Visibility  string
	Docstring   string
	Signature   string
	CodeBody    string
}

// CallSite represents a function/method call
type CallSite struct {
	CallerName string // Name of the calling function/method
	CalleeName string // Name of the called function/method
	Line       int
	Column     int
	CallType   string // direct, method, indirect
}

// Import represents an import statement
type Import struct {
	ImportPath      string
	ImportType      string   // named, default, wildcard
	ImportedSymbols []string // List of imported symbols
	LineNumber      int
}

// ParseResult contains all parsed information from a file
type ParseResult struct {
	Symbols   []*Symbol
	CallSites []*CallSite
	Imports   []*Import
}

// Parser wraps a tree-sitter parser
type Parser struct {
	parser *sitter.Parser
	lang   Language
}

// New creates a new parser for the given language
func New(lang Language) (*Parser, error) {
	// JSON and Text don't need tree-sitter parsers
	if lang == LangJSON || lang == LangText {
		return &Parser{
			parser: nil,
			lang:   lang,
		}, nil
	}

	parser := sitter.NewParser()

	var tsLang *sitter.Language
	switch lang {
	case LangGo:
		tsLang = golang.GetLanguage()
	case LangTypeScript:
		tsLang = typescript.GetLanguage()
	case LangJavaScript:
		tsLang = javascript.GetLanguage()
	case LangPython:
		tsLang = python.GetLanguage()
	case LangRust:
		tsLang = rust.GetLanguage()
	case LangJava:
		tsLang = java.GetLanguage()
	case LangCSharp:
		tsLang = csharp.GetLanguage()
	case LangHTML:
		tsLang = html.GetLanguage()
	case LangCSS:
		tsLang = css.GetLanguage()
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	parser.SetLanguage(tsLang)

	return &Parser{
		parser: parser,
		lang:   lang,
	}, nil
}

// ParseFile parses a file and extracts symbols, calls, and imports
func (p *Parser) ParseFile(ctx context.Context, filePath string) (*ParseResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(ctx, content)
}

// Parse parses source code and extracts symbols, calls, and imports
func (p *Parser) Parse(ctx context.Context, sourceCode []byte) (*ParseResult, error) {
	result := &ParseResult{
		Symbols:   []*Symbol{},
		CallSites: []*CallSite{},
		Imports:   []*Import{},
	}

	// Special handling for JSON and Text (no tree-sitter needed)
	if p.lang == LangJSON {
		p.parseJSON(nil, sourceCode, result)
		return result, nil
	}

	if p.lang == LangText {
		p.parseText(nil, sourceCode, result)
		return result, nil
	}

	// For all other languages, use tree-sitter
	tree, err := p.parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	switch p.lang {
	case LangGo:
		p.parseGo(root, sourceCode, result)
	case LangTypeScript, LangJavaScript:
		p.parseTypeScript(root, sourceCode, result)
	case LangPython:
		p.parsePython(root, sourceCode, result)
	case LangRust:
		p.parseRust(root, sourceCode, result)
	case LangJava:
		p.parseJava(root, sourceCode, result)
	case LangCSharp:
		p.parseCSharp(root, sourceCode, result)
	case LangHTML:
		p.parseHTML(root, sourceCode, result)
	case LangCSS:
		p.parseCSS(root, sourceCode, result)
	default:
		return nil, fmt.Errorf("unsupported language: %s", p.lang)
	}

	return result, nil
}

// DetectLanguage detects the programming language from file extension
func DetectLanguage(filePath string) Language {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go":
		return LangGo
	case ".ts", ".tsx":
		return LangTypeScript
	case ".js", ".jsx":
		return LangJavaScript
	case ".py":
		return LangPython
	case ".rs":
		return LangRust
	case ".java":
		return LangJava
	case ".cs":
		return LangCSharp
	case ".html", ".htm":
		return LangHTML
	case ".css":
		return LangCSS
	case ".json":
		return LangJSON
	case ".txt", ".md", ".markdown", ".xml", ".yaml", ".yml", ".toml", ".ini", ".conf", ".config":
		// Common text-based formats fallback to text parser
		return LangText
	default:
		// Unknown extensions fallback to text parser
		return LangText
	}
}

// getNodeText extracts text from a node
func getNodeText(node *sitter.Node, sourceCode []byte) string {
	return string(sourceCode[node.StartByte():node.EndByte()])
}

// findChildByType finds a child node by type
func findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

// findPrecedingComment finds a comment node preceding the given node
func findPrecedingComment(node *sitter.Node, sourceCode []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if strings.Contains(prev.Type(), "comment") {
		return strings.TrimSpace(getNodeText(prev, sourceCode))
	}

	return ""
}
