package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseTypeScript parses TypeScript/JavaScript source code
func (p *Parser) parseTypeScript(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractTSImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkTSTree(cursor, sourceCode, "", result)
}

// extractTSImports extracts import statements from TypeScript/JavaScript code
func (p *Parser) extractTSImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findTSImports(cursor, sourceCode, result)
}

// findTSImports recursively finds import statements
func (p *Parser) findTSImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "import_statement" {
		p.parseTSImportStatement(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findTSImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseTSImportStatement parses a TypeScript/JavaScript import statement
func (p *Parser) parseTSImportStatement(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType string = "named"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string":
			// Import path: import "..." or import '...'
			path := getNodeText(child, sourceCode)
			importPath = strings.Trim(strings.Trim(path, "\""), "'")
		case "import_clause":
			// Parse import clause: import { A, B } from "..."
			importType, importedSymbols = p.parseTSImportClause(child, sourceCode)
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

// parseTSImportClause parses the import clause
func (p *Parser) parseTSImportClause(node *sitter.Node, sourceCode []byte) (string, []string) {
	importType := "named"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			// Default import: import X from "..."
			importType = "default"
			importedSymbols = append(importedSymbols, getNodeText(child, sourceCode))
		case "namespace_import":
			// Namespace import: import * as X from "..."
			importType = "wildcard"
			nameNode := findChildByType(child, "identifier")
			if nameNode != nil {
				importedSymbols = append(importedSymbols, getNodeText(nameNode, sourceCode))
			}
		case "named_imports":
			// Named imports: import { A, B } from "..."
			importType = "named"
			for j := 0; j < int(child.ChildCount()); j++ {
				specNode := child.Child(j)
				if specNode.Type() == "import_specifier" {
					nameNode := findChildByType(specNode, "identifier")
					if nameNode != nil {
						importedSymbols = append(importedSymbols, getNodeText(nameNode, sourceCode))
					}
				}
			}
		}
	}

	return importType, importedSymbols
}

// walkTSTree walks the TypeScript AST tree
func (p *Parser) walkTSTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_declaration":
		p.parseTSFunction(node, sourceCode, scope, result)
	case "class_declaration":
		p.parseTSClass(node, sourceCode, scope, result)
	case "method_definition":
		p.parseTSMethod(node, sourceCode, scope, result)
	case "interface_declaration":
		p.parseTSInterface(node, sourceCode, scope, result)
	case "type_alias_declaration":
		p.parseTSType(node, sourceCode, scope, result)
	case "lexical_declaration", "variable_declaration":
		p.parseTSVariable(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkTSTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseTSFunction parses a TypeScript function declaration
func (p *Parser) parseTSFunction(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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
		// Extract call sites from the body
		p.extractTSCallSites(bodyNode, sourceCode, name, result)
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

	result.Symbols = append(result.Symbols, symbol)
}

// parseTSClass parses a TypeScript class declaration
func (p *Parser) parseTSClass(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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

	result.Symbols = append(result.Symbols, symbol)
}

// parseTSMethod parses a TypeScript method definition
func (p *Parser) parseTSMethod(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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
		// Extract call sites from the body
		callerName := scope + "." + name
		p.extractTSCallSites(bodyNode, sourceCode, callerName, result)
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

	result.Symbols = append(result.Symbols, symbol)
}

// parseTSInterface parses a TypeScript interface declaration
func (p *Parser) parseTSInterface(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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

	result.Symbols = append(result.Symbols, symbol)
}

// parseTSType parses a TypeScript type alias declaration
func (p *Parser) parseTSType(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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

	result.Symbols = append(result.Symbols, symbol)
}

// parseTSVariable parses a TypeScript variable declaration
func (p *Parser) parseTSVariable(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
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

			result.Symbols = append(result.Symbols, symbol)
		}
	}
}

// extractTSCallSites extracts function/method calls from a code block
func (p *Parser) extractTSCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkTSCallSites(cursor, sourceCode, callerName, result)
}

// walkTSCallSites walks the AST to find call expressions
func (p *Parser) walkTSCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "call_expression" {
		p.parseTSCallExpression(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkTSCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseTSCallExpression parses a call expression
func (p *Parser) parseTSCallExpression(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	if node.ChildCount() == 0 {
		return
	}

	funcNode := node.Child(0)
	var calleeName string
	var callType string = "direct"

	switch funcNode.Type() {
	case "identifier":
		// Direct function call: foo()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "direct"
	case "member_expression":
		// Method call: obj.method() or package.function()
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
