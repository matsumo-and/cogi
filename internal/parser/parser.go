package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Language represents a programming language
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangJavaScript Language = "javascript"
	LangPython     Language = "python"
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

// Parser wraps a tree-sitter parser
type Parser struct {
	parser *sitter.Parser
	lang   Language
}

// New creates a new parser for the given language
func New(lang Language) (*Parser, error) {
	parser := sitter.NewParser()

	var tsLang *sitter.Language
	switch lang {
	case LangGo:
		tsLang = golang.GetLanguage()
	case LangTypeScript:
		tsLang = typescript.GetLanguage()
	case LangPython:
		tsLang = python.GetLanguage()
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	parser.SetLanguage(tsLang)

	return &Parser{
		parser: parser,
		lang:   lang,
	}, nil
}

// ParseFile parses a file and extracts symbols
func (p *Parser) ParseFile(ctx context.Context, filePath string) ([]*Symbol, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(ctx, content)
}

// Parse parses source code and extracts symbols
func (p *Parser) Parse(ctx context.Context, sourceCode []byte) ([]*Symbol, error) {
	tree, err := p.parser.ParseCtx(ctx, nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	var symbols []*Symbol
	switch p.lang {
	case LangGo:
		symbols = p.parseGo(root, sourceCode)
	case LangTypeScript, LangJavaScript:
		symbols = p.parseTypeScript(root, sourceCode)
	case LangPython:
		symbols = p.parsePython(root, sourceCode)
	default:
		return nil, fmt.Errorf("unsupported language: %s", p.lang)
	}

	return symbols, nil
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
	default:
		return LangUnknown
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
