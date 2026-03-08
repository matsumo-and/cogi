package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseTypeScript parses TypeScript/JavaScript source code
func (p *Parser) parseTypeScript(root *sitter.Node, sourceCode []byte) []*Symbol {
	var symbols []*Symbol

	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkTSTree(cursor, sourceCode, "", &symbols)

	return symbols
}

// walkTSTree walks the TypeScript AST tree
func (p *Parser) walkTSTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, symbols *[]*Symbol) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_declaration":
		p.parseTSFunction(node, sourceCode, scope, symbols)
	case "class_declaration":
		p.parseTSClass(node, sourceCode, scope, symbols)
	case "method_definition":
		p.parseTSMethod(node, sourceCode, scope, symbols)
	case "interface_declaration":
		p.parseTSInterface(node, sourceCode, scope, symbols)
	case "type_alias_declaration":
		p.parseTSType(node, sourceCode, scope, symbols)
	case "lexical_declaration", "variable_declaration":
		p.parseTSVariable(node, sourceCode, scope, symbols)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkTSTree(cursor, sourceCode, scope, symbols)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseTSFunction parses a TypeScript function declaration
func (p *Parser) parseTSFunction(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check for export keyword
	visibility := "private"
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "export" {
		visibility = "public"
	}

	// Get function signature
	signature := extractTSSignature(node, sourceCode)

	// Get JSDoc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get function body
	bodyNode := findChildByType(node, "statement_block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
	}

	symbol := &Symbol{
		Name:        name,
		Kind:        "function",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    codeBody,
	}

	*symbols = append(*symbols, symbol)
}

// parseTSClass parses a TypeScript class declaration
func (p *Parser) parseTSClass(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check for export keyword
	visibility := "private"
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "export" {
		visibility = "public"
	}

	// Get class signature (including extends/implements)
	signature := extractTSClassSignature(node, sourceCode)

	// Get JSDoc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get class body
	bodyNode := findChildByType(node, "class_body")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
	}

	symbol := &Symbol{
		Name:        name,
		Kind:        "class",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    codeBody,
	}

	*symbols = append(*symbols, symbol)
}

// parseTSMethod parses a TypeScript method definition
func (p *Parser) parseTSMethod(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "property_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Determine visibility from modifiers
	visibility := p.getTSVisibility(node, sourceCode)

	// Get method signature
	signature := extractTSSignature(node, sourceCode)

	// Get JSDoc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get method body
	bodyNode := findChildByType(node, "statement_block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
	}

	symbol := &Symbol{
		Name:        name,
		Kind:        "method",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    codeBody,
	}

	*symbols = append(*symbols, symbol)
}

// parseTSInterface parses a TypeScript interface declaration
func (p *Parser) parseTSInterface(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check for export keyword
	visibility := "private"
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "export" {
		visibility = "public"
	}

	// Get interface signature
	signature := getNodeText(node, sourceCode)

	// Get JSDoc comment
	docstring := findPrecedingComment(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "interface",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    signature,
	}

	*symbols = append(*symbols, symbol)
}

// parseTSType parses a TypeScript type alias declaration
func (p *Parser) parseTSType(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check for export keyword
	visibility := "private"
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "export" {
		visibility = "public"
	}

	// Get type signature
	signature := getNodeText(node, sourceCode)

	// Get JSDoc comment
	docstring := findPrecedingComment(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "type",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    signature,
	}

	*symbols = append(*symbols, symbol)
}

// parseTSVariable parses a TypeScript variable declaration
func (p *Parser) parseTSVariable(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	// Find variable declarators
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declarator" {
			nameNode := findChildByType(child, "identifier")
			if nameNode == nil {
				continue
			}

			name := getNodeText(nameNode, sourceCode)

			// Check for export keyword
			visibility := "private"
			if node.PrevSibling() != nil && node.PrevSibling().Type() == "export" {
				visibility = "public"
			}

			signature := getNodeText(child, sourceCode)

			symbol := &Symbol{
				Name:        name,
				Kind:        "variable",
				StartLine:   int(child.StartPoint().Row) + 1,
				StartColumn: int(child.StartPoint().Column) + 1,
				EndLine:     int(child.EndPoint().Row) + 1,
				EndColumn:   int(child.EndPoint().Column) + 1,
				Scope:       scope,
				Visibility:  visibility,
				Signature:   signature,
				CodeBody:    signature,
			}

			*symbols = append(*symbols, symbol)
		}
	}
}

// extractTSSignature extracts the TypeScript function/method signature
func extractTSSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "statement_block" {
			break
		}
		signature += getNodeText(child, sourceCode) + " "
	}
	return strings.TrimSpace(signature)
}

// extractTSClassSignature extracts the class signature including extends/implements
func extractTSClassSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "class_body" {
			break
		}
		signature += getNodeText(child, sourceCode) + " "
	}
	return strings.TrimSpace(signature)
}

// getTSVisibility determines the visibility from TypeScript modifiers
func (p *Parser) getTSVisibility(node *sitter.Node, sourceCode []byte) string {
	// Look for accessibility modifiers (public, private, protected)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "accessibility_modifier" {
			modifier := getNodeText(child, sourceCode)
			return modifier
		}
	}
	return "public" // Default in TypeScript
}
