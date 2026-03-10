package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parsePython parses Python source code
func (p *Parser) parsePython(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractPythonImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkPythonTree(cursor, sourceCode, "", result)
}

// extractPythonImports extracts import statements from Python code
func (p *Parser) extractPythonImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findPythonImports(cursor, sourceCode, result)
}

// findPythonImports recursively finds import statements
func (p *Parser) findPythonImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "import_statement":
		p.parsePythonImportStatement(node, sourceCode, result)
	case "import_from_statement":
		p.parsePythonImportFromStatement(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findPythonImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parsePythonImportStatement parses: import module [as alias]
func (p *Parser) parsePythonImportStatement(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "dotted_name" || child.Type() == "aliased_import" {
			importPath := ""
			importType := "default"
			var importedSymbols []string

			if child.Type() == "dotted_name" {
				importPath = getNodeText(child, sourceCode)
			} else if child.Type() == "aliased_import" {
				// import X as Y
				nameNode := findChildByType(child, "dotted_name")
				if nameNode != nil {
					importPath = getNodeText(nameNode, sourceCode)
				}
				aliasNode := findChildByType(child, "identifier")
				if aliasNode != nil {
					importedSymbols = append(importedSymbols, getNodeText(aliasNode, sourceCode))
					importType = "named"
				}
			}

			if importPath != "" {
				result.Imports = append(result.Imports, &Import{
					ImportPath:      importPath,
					ImportType:      importType,
					ImportedSymbols: importedSymbols,
					LineNumber:      int(node.StartPoint().Row) + 1,
				})
			}
		}
	}
}

// parsePythonImportFromStatement parses: from module import X [, Y] [as Z]
func (p *Parser) parsePythonImportFromStatement(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType = "named"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		switch childType {
		case "dotted_name":
			// Module path (appears first before "import" keyword)
			if importPath == "" {
				importPath = getNodeText(child, sourceCode)
			} else {
				// Named import after "import" keyword
				importedSymbols = append(importedSymbols, getNodeText(child, sourceCode))
			}
		case "relative_import":
			// Relative import: from . import X or from .. import Y
			importPath = getNodeText(child, sourceCode)
		case "wildcard_import":
			// from module import *
			importType = "wildcard"
			importedSymbols = append(importedSymbols, "*")
		case "identifier":
			// Named import
			if importPath != "" { // Only if we already have the module path
				importedSymbols = append(importedSymbols, getNodeText(child, sourceCode))
			}
		case "aliased_import":
			// from module import X as Y
			nameNode := child.Child(0)
			if nameNode != nil {
				importedSymbols = append(importedSymbols, getNodeText(nameNode, sourceCode))
			}
		}
	}

	if importPath != "" {
		result.Imports = append(result.Imports, &Import{
			ImportPath:      importPath,
			ImportType:      importType,
			ImportedSymbols: importedSymbols,
			LineNumber:      int(node.StartPoint().Row) + 1,
		})
	}
}

// walkPythonTree walks the Python AST tree
func (p *Parser) walkPythonTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_definition":
		p.parsePythonFunction(node, sourceCode, scope, result)
	case "class_definition":
		p.parsePythonClass(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkPythonTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parsePythonFunction parses a Python function definition
func (p *Parser) parsePythonFunction(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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
		// Extract call sites from the body
		callerName := name
		if scope != "" {
			callerName = scope + "." + name
		}
		p.extractPythonCallSites(bodyNode, sourceCode, callerName, result)
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

	result.Symbols = append(result.Symbols, symbol)
}

// parsePythonClass parses a Python class definition
func (p *Parser) parsePythonClass(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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

	result.Symbols = append(result.Symbols, symbol)

	// Parse methods within the class
	if bodyNode != nil {
		cursor := sitter.NewTreeCursor(bodyNode)
		defer cursor.Close()

		if cursor.GoToFirstChild() {
			for {
				if cursor.CurrentNode().Type() == "function_definition" {
					p.parsePythonFunction(cursor.CurrentNode(), sourceCode, name, result)
				}
				if !cursor.GoToNextSibling() {
					break
				}
			}
		}
	}
}

// extractPythonCallSites extracts function/method calls from a code block
func (p *Parser) extractPythonCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkPythonCallSites(cursor, sourceCode, callerName, result)
}

// walkPythonCallSites walks the AST to find call expressions
func (p *Parser) walkPythonCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "call" {
		p.parsePythonCallExpression(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkPythonCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parsePythonCallExpression parses a call expression
func (p *Parser) parsePythonCallExpression(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	if node.ChildCount() == 0 {
		return
	}

	funcNode := node.Child(0)
	var calleeName string
	var callType string

	switch funcNode.Type() {
	case "identifier":
		// Direct function call: foo()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "direct"
	case "attribute":
		// Method call: obj.method()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "method"
	default:
		// Other complex expressions
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "indirect"
	}

	if calleeName != "" {
		result.CallSites = append(result.CallSites, &CallSite{
			CallerName: callerName,
			CalleeName: calleeName,
			Line:       int(node.StartPoint().Row) + 1,
			Column:     int(node.StartPoint().Column) + 1,
			CallType:   callType,
		})
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
