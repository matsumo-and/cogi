package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseCSharp parses C# source code
func (p *Parser) parseCSharp(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractCSharpImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkCSharpTree(cursor, sourceCode, "", result)
}

// extractCSharpImports extracts using statements from C# code
func (p *Parser) extractCSharpImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findCSharpImports(cursor, sourceCode, result)
}

// findCSharpImports recursively finds using directives
func (p *Parser) findCSharpImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "using_directive" {
		p.parseCSharpUsingDirective(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findCSharpImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseCSharpUsingDirective parses a C# using directive
func (p *Parser) parseCSharpUsingDirective(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType string = "named"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "qualified_name", "identifier":
			importPath = getNodeText(child, sourceCode)
		case "name_equals":
			// using alias = namespace
			importType = "named"
			nameNode := findChildByType(child, "identifier")
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

// walkCSharpTree walks the C# AST tree
func (p *Parser) walkCSharpTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "class_declaration":
		p.parseCSharpClass(node, sourceCode, scope, result)
	case "interface_declaration":
		p.parseCSharpInterface(node, sourceCode, scope, result)
	case "struct_declaration":
		p.parseCSharpStruct(node, sourceCode, scope, result)
	case "enum_declaration":
		p.parseCSharpEnum(node, sourceCode, scope, result)
	case "method_declaration":
		p.parseCSharpMethod(node, sourceCode, scope, result)
	case "field_declaration":
		p.parseCSharpField(node, sourceCode, scope, result)
	case "property_declaration":
		p.parseCSharpProperty(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkCSharpTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseCSharpClass parses a C# class declaration
func (p *Parser) parseCSharpClass(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get class signature
	signature := extractCSharpClassSignature(node, sourceCode)

	// Get class body
	bodyNode := findChildByType(node, "declaration_list")
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

// parseCSharpInterface parses a C# interface declaration
func (p *Parser) parseCSharpInterface(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get interface signature
	signature := getNodeText(node, sourceCode)

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

// parseCSharpStruct parses a C# struct declaration
func (p *Parser) parseCSharpStruct(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get struct signature
	signature := getNodeText(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "struct",
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

// parseCSharpEnum parses a C# enum declaration
func (p *Parser) parseCSharpEnum(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get enum signature
	signature := getNodeText(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "enum",
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

// parseCSharpMethod parses a C# method declaration
func (p *Parser) parseCSharpMethod(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get method signature
	signature := extractCSharpMethodSignature(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get method body
	bodyNode := findChildByType(node, "block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
		// Extract call sites from the body
		callerName := name
		if scope != "" {
			callerName = scope + "." + name
		}
		p.extractCSharpCallSites(bodyNode, sourceCode, callerName, result)
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

// parseCSharpField parses a C# field declaration
func (p *Parser) parseCSharpField(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Field declaration can have multiple variable declarators
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declaration" {
			for j := 0; j < int(child.ChildCount()); j++ {
				varDecl := child.Child(j)
				if varDecl.Type() == "variable_declarator" {
					nameNode := findChildByType(varDecl, "identifier")
					if nameNode == nil {
						continue
					}

					name := getNodeText(nameNode, sourceCode)

					// Get visibility
					visibility := getCSharpVisibility(node, sourceCode)

					// Get doc comment
					docstring := findPrecedingComment(node, sourceCode)

					signature := getNodeText(node, sourceCode)

					symbol := &Symbol{
						Name:        name,
						Kind:        "field",
						StartLine:   int(varDecl.StartPoint().Row) + 1,
						StartColumn: int(varDecl.StartPoint().Column) + 1,
						EndLine:     int(varDecl.EndPoint().Row) + 1,
						EndColumn:   int(varDecl.EndPoint().Column) + 1,
						Scope:       scope,
						Visibility:  visibility,
						Docstring:   docstring,
						Signature:   signature,
						CodeBody:    signature,
					}

					result.Symbols = append(result.Symbols, symbol)
				}
			}
		}
	}
}

// parseCSharpProperty parses a C# property declaration
func (p *Parser) parseCSharpProperty(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getCSharpVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	signature := getNodeText(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "property",
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

// extractCSharpCallSites extracts method calls from a code block
func (p *Parser) extractCSharpCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkCSharpCallSites(cursor, sourceCode, callerName, result)
}

// walkCSharpCallSites walks the AST to find invocation expressions
func (p *Parser) walkCSharpCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "invocation_expression" {
		p.parseCSharpInvocationExpression(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkCSharpCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseCSharpInvocationExpression parses an invocation expression
func (p *Parser) parseCSharpInvocationExpression(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	if node.ChildCount() == 0 {
		return
	}

	funcNode := node.Child(0)
	var calleeName string
	var callType string = "direct"

	switch funcNode.Type() {
	case "identifier":
		// Direct method call: Foo()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "direct"
	case "member_access_expression":
		// Member call: obj.Method()
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

// getCSharpVisibility extracts visibility modifier from a C# node
func getCSharpVisibility(node *sitter.Node, sourceCode []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifier" || strings.Contains(child.Type(), "modifier") {
			text := getNodeText(child, sourceCode)
			if strings.Contains(text, "public") {
				return "public"
			} else if strings.Contains(text, "private") {
				return "private"
			} else if strings.Contains(text, "protected") {
				if strings.Contains(text, "internal") {
					return "protected internal"
				}
				return "protected"
			} else if strings.Contains(text, "internal") {
				return "internal"
			}
		}
	}
	return "private" // Default private
}

// extractCSharpClassSignature extracts the C# class signature
func extractCSharpClassSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "declaration_list" {
			break
		}
		signature += getNodeText(child, sourceCode) + " "
	}
	return strings.TrimSpace(signature)
}

// extractCSharpMethodSignature extracts the C# method signature
func extractCSharpMethodSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "block" || child.Type() == "arrow_expression_clause" {
			break
		}
		signature += getNodeText(child, sourceCode) + " "
	}
	return strings.TrimSpace(signature)
}
