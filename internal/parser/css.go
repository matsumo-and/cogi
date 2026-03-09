package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseCSS parses CSS source code
func (p *Parser) parseCSS(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Walk the tree for rules and selectors
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkCSSTree(cursor, sourceCode, "", result)
}

// walkCSSTree walks the CSS AST tree
func (p *Parser) walkCSSTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "rule_set":
		p.parseCSSRuleSet(node, sourceCode, scope, result)
	case "media_statement":
		p.parseCSSMediaStatement(node, sourceCode, scope, result)
	case "keyframes_statement":
		p.parseCSSKeyframesStatement(node, sourceCode, scope, result)
	case "import_statement":
		p.parseCSSImportStatement(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkCSSTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseCSSRuleSet parses a CSS rule set (selector + declarations)
func (p *Parser) parseCSSRuleSet(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Get selectors
	var selectors []string
	var declarations []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "selectors":
			selectors = p.parseCSSSelectors(child, sourceCode)
		case "block":
			declarations = p.parseCSSDeclarations(child, sourceCode)
		}
	}

	// Create a symbol for each significant selector
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}

		// Determine selector kind
		kind := "selector"
		if strings.HasPrefix(selector, ".") {
			kind = "class"
		} else if strings.HasPrefix(selector, "#") {
			kind = "id"
		} else if strings.HasPrefix(selector, "@") {
			kind = "at-rule"
		}

		// Build signature
		signature := selector + " { " + strings.Join(declarations, "; ") + " }"
		if len(signature) > 200 {
			signature = selector + " { ... }"
		}

		symbol := &Symbol{
			Name:        selector,
			Kind:        kind,
			StartLine:   int(node.StartPoint().Row) + 1,
			StartColumn: int(node.StartPoint().Column) + 1,
			EndLine:     int(node.EndPoint().Row) + 1,
			EndColumn:   int(node.EndPoint().Column) + 1,
			Scope:       scope,
			Visibility:  "public",
			Signature:   signature,
			CodeBody:    getNodeText(node, sourceCode),
		}

		result.Symbols = append(result.Symbols, symbol)
	}
}

// parseCSSSelectors extracts selectors from a selectors node
func (p *Parser) parseCSSSelectors(node *sitter.Node, sourceCode []byte) []string {
	var selectors []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		// Each child can be a selector or comma
		if child.Type() != "," {
			selector := strings.TrimSpace(getNodeText(child, sourceCode))
			if selector != "" {
				selectors = append(selectors, selector)
			}
		}
	}

	return selectors
}

// parseCSSDeclarations extracts property declarations from a block
func (p *Parser) parseCSSDeclarations(node *sitter.Node, sourceCode []byte) []string {
	var declarations []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "declaration" {
			decl := strings.TrimSpace(getNodeText(child, sourceCode))
			declarations = append(declarations, decl)
		}
	}

	return declarations
}

// parseCSSMediaStatement parses a @media rule
func (p *Parser) parseCSSMediaStatement(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Get media query
	var query string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "query" || child.Type() == "feature_query" {
			query = getNodeText(child, sourceCode)
			break
		}
	}

	name := "@media " + strings.TrimSpace(query)

	symbol := &Symbol{
		Name:        name,
		Kind:        "media-query",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  "public",
		Signature:   name,
		CodeBody:    getNodeText(node, sourceCode),
	}

	result.Symbols = append(result.Symbols, symbol)
}

// parseCSSKeyframesStatement parses a @keyframes rule
func (p *Parser) parseCSSKeyframesStatement(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Get keyframe name
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "keyframes_name" {
			name = getNodeText(child, sourceCode)
			break
		}
	}

	if name == "" {
		// Try to find identifier
		nameNode := findChildByType(node, "identifier")
		if nameNode != nil {
			name = getNodeText(nameNode, sourceCode)
		}
	}

	fullName := "@keyframes " + strings.TrimSpace(name)

	symbol := &Symbol{
		Name:        fullName,
		Kind:        "keyframes",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  "public",
		Signature:   fullName,
		CodeBody:    getNodeText(node, sourceCode),
	}

	result.Symbols = append(result.Symbols, symbol)
}

// parseCSSImportStatement parses a CSS @import statement
func (p *Parser) parseCSSImportStatement(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "string_value" || child.Type() == "call_expression" {
			path := getNodeText(child, sourceCode)
			// Remove quotes and url()
			path = strings.Trim(path, `"'`)
			path = strings.TrimPrefix(path, "url(")
			path = strings.TrimSuffix(path, ")")
			path = strings.Trim(path, `"' `)
			importPath = path
			break
		}
	}

	if importPath != "" {
		result.Imports = append(result.Imports, &Import{
			ImportPath:      importPath,
			ImportType:      "default",
			ImportedSymbols: []string{},
			LineNumber:      int(node.StartPoint().Row) + 1,
		})
	}
}
