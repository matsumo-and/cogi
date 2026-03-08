package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parsePython parses Python source code
func (p *Parser) parsePython(root *sitter.Node, sourceCode []byte) []*Symbol {
	var symbols []*Symbol

	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkPythonTree(cursor, sourceCode, "", &symbols)

	return symbols
}

// walkPythonTree walks the Python AST tree
func (p *Parser) walkPythonTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, symbols *[]*Symbol) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_definition":
		p.parsePythonFunction(node, sourceCode, scope, symbols)
	case "class_definition":
		p.parsePythonClass(node, sourceCode, scope, symbols)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkPythonTree(cursor, sourceCode, scope, symbols)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parsePythonFunction parses a Python function definition
func (p *Parser) parsePythonFunction(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Determine visibility based on naming convention
	visibility := "public"
	if strings.HasPrefix(name, "_") {
		if strings.HasPrefix(name, "__") && !strings.HasSuffix(name, "__") {
			visibility = "private"
		} else {
			visibility = "protected"
		}
	}

	// Determine kind (function or method)
	kind := "function"
	if scope != "" {
		kind = "method"
	}

	// Get function signature
	signature := extractPythonSignature(node, sourceCode)

	// Get docstring
	docstring := extractPythonDocstring(node, sourceCode)

	// Get function body
	bodyNode := findChildByType(node, "block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
	}

	symbol := &Symbol{
		Name:        name,
		Kind:        kind,
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

// parsePythonClass parses a Python class definition
func (p *Parser) parsePythonClass(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Determine visibility based on naming convention
	visibility := "public"
	if strings.HasPrefix(name, "_") {
		visibility = "private"
	}

	// Get class signature (including base classes)
	signature := extractPythonClassSignature(node, sourceCode)

	// Get docstring
	docstring := extractPythonDocstring(node, sourceCode)

	// Get class body
	bodyNode := findChildByType(node, "block")
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

	// Parse methods within the class
	if bodyNode != nil {
		cursor := sitter.NewTreeCursor(bodyNode)
		defer cursor.Close()

		if cursor.GoToFirstChild() {
			for {
				if cursor.CurrentNode().Type() == "function_definition" {
					p.parsePythonFunction(cursor.CurrentNode(), sourceCode, name, symbols)
				}
				if !cursor.GoToNextSibling() {
					break
				}
			}
		}
	}
}

// extractPythonSignature extracts the Python function signature
func extractPythonSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		// Include def keyword, name, parameters, and return type
		if nodeType == "def" || nodeType == "identifier" ||
		   nodeType == "parameters" || nodeType == "->" ||
		   nodeType == "type" {
			signature += getNodeText(child, sourceCode) + " "
		}

		// Stop at the body
		if nodeType == "block" || nodeType == ":" {
			break
		}
	}
	return strings.TrimSpace(signature)
}

// extractPythonClassSignature extracts the Python class signature
func extractPythonClassSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		nodeType := child.Type()

		// Include class keyword, name, and argument list (base classes)
		if nodeType == "class" || nodeType == "identifier" ||
		   nodeType == "argument_list" {
			signature += getNodeText(child, sourceCode) + " "
		}

		// Stop at the body
		if nodeType == "block" || nodeType == ":" {
			break
		}
	}
	return strings.TrimSpace(signature)
}

// extractPythonDocstring extracts the Python docstring from a function or class
func extractPythonDocstring(node *sitter.Node, sourceCode []byte) string {
	// Find the block (body)
	bodyNode := findChildByType(node, "block")
	if bodyNode == nil {
		return ""
	}

	// Check if the first statement in the block is an expression_statement
	// containing a string (docstring)
	if bodyNode.ChildCount() == 0 {
		return ""
	}

	firstChild := bodyNode.Child(0)
	if firstChild.Type() == "expression_statement" {
		// Check if it contains a string
		for i := 0; i < int(firstChild.ChildCount()); i++ {
			child := firstChild.Child(i)
			if child.Type() == "string" {
				docstring := getNodeText(child, sourceCode)
				// Remove quotes
				docstring = strings.Trim(docstring, `"'`)
				// Handle triple quotes
				docstring = strings.TrimPrefix(docstring, `""`)
				docstring = strings.TrimSuffix(docstring, `""`)
				docstring = strings.TrimPrefix(docstring, `''`)
				docstring = strings.TrimSuffix(docstring, `''`)
				return strings.TrimSpace(docstring)
			}
		}
	}

	return ""
}
