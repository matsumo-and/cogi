package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseRust parses Rust source code
func (p *Parser) parseRust(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractRustImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkRustTree(cursor, sourceCode, "", result)
}

// extractRustImports extracts use statements from Rust code
func (p *Parser) extractRustImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findRustImports(cursor, sourceCode, result)
}

// findRustImports recursively finds use declarations
func (p *Parser) findRustImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "use_declaration" {
		p.parseRustUseDeclaration(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findRustImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseRustUseDeclaration parses a Rust use declaration
func (p *Parser) parseRustUseDeclaration(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType = "named"
	var importedSymbols []string

	// Get the use clause
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "scoped_identifier", "identifier":
			importPath = getNodeText(child, sourceCode)
		case "use_list":
			// use foo::{A, B, C}
			importType = "named"
			for j := 0; j < int(child.ChildCount()); j++ {
				item := child.Child(j)
				if item.Type() == "identifier" || item.Type() == "scoped_identifier" {
					importedSymbols = append(importedSymbols, getNodeText(item, sourceCode))
				}
			}
		case "use_wildcard":
			// use foo::*
			importType = "wildcard"
			importedSymbols = append(importedSymbols, "*")
		case "use_as_clause":
			// use foo as bar
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

// walkRustTree walks the Rust AST tree
func (p *Parser) walkRustTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_item":
		p.parseRustFunction(node, sourceCode, scope, result)
	case "struct_item":
		p.parseRustStruct(node, sourceCode, scope, result)
	case "enum_item":
		p.parseRustEnum(node, sourceCode, scope, result)
	case "trait_item":
		p.parseRustTrait(node, sourceCode, scope, result)
	case "impl_item":
		p.parseRustImpl(node, sourceCode, scope, result)
	case "const_item", "static_item":
		p.parseRustConst(node, sourceCode, scope, result)
	case "type_item":
		p.parseRustTypeAlias(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkRustTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseRustFunction parses a Rust function declaration
func (p *Parser) parseRustFunction(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get function signature
	signature := extractRustSignature(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get function body
	bodyNode := findChildByType(node, "block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
		// Extract call sites from the body
		callerName := name
		if scope != "" {
			callerName = scope + "::" + name
		}
		p.extractRustCallSites(bodyNode, sourceCode, callerName, result)
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

// parseRustStruct parses a Rust struct declaration
func (p *Parser) parseRustStruct(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get struct definition
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

// parseRustEnum parses a Rust enum declaration
func (p *Parser) parseRustEnum(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get enum definition
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

// parseRustTrait parses a Rust trait declaration
func (p *Parser) parseRustTrait(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get trait definition
	signature := getNodeText(node, sourceCode)

	symbol := &Symbol{
		Name:        name,
		Kind:        "trait",
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

// parseRustImpl parses a Rust impl block
func (p *Parser) parseRustImpl(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Get the type being implemented
	typeNode := findChildByType(node, "type_identifier")
	var implType string
	if typeNode != nil {
		implType = getNodeText(typeNode, sourceCode)
	}

	// Walk through impl block to find methods
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	if cursor.GoToFirstChild() {
		for {
			child := cursor.CurrentNode()
			if child.Type() == "function_item" {
				p.parseRustFunction(child, sourceCode, implType, result)
			}
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}
}

// parseRustConst parses a Rust const/static declaration
func (p *Parser) parseRustConst(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)
	kind := "constant"
	if node.Type() == "static_item" {
		kind = "static"
	}

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	signature := getNodeText(node, sourceCode)

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
		CodeBody:    signature,
	}

	result.Symbols = append(result.Symbols, symbol)
}

// parseRustTypeAlias parses a Rust type alias
func (p *Parser) parseRustTypeAlias(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Check visibility
	visibility := "private"
	if hasRustVisibilityModifier(node, sourceCode, "pub") {
		visibility = "public"
	}

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	signature := getNodeText(node, sourceCode)

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

// extractRustCallSites extracts function/method calls from a code block
func (p *Parser) extractRustCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkRustCallSites(cursor, sourceCode, callerName, result)
}

// walkRustCallSites walks the AST to find call expressions
func (p *Parser) walkRustCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "call_expression" {
		p.parseRustCallExpression(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkRustCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseRustCallExpression parses a call expression
func (p *Parser) parseRustCallExpression(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
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
	case "scoped_identifier", "field_expression":
		// Method/scoped call: obj.method() or module::function()
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

// extractRustSignature extracts the Rust function signature
func extractRustSignature(node *sitter.Node, sourceCode []byte) string {
	signature := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "block" {
			break
		}
		signature += getNodeText(child, sourceCode) + " "
	}
	return strings.TrimSpace(signature)
}

// hasRustVisibilityModifier checks if a node has a specific visibility modifier
func hasRustVisibilityModifier(node *sitter.Node, sourceCode []byte, modifier string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "visibility_modifier" {
			text := getNodeText(child, sourceCode)
			return strings.Contains(text, modifier)
		}
	}
	return false
}
