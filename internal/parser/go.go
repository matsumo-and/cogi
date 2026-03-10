package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseGo parses Go source code
func (p *Parser) parseGo(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractGoImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkGoTree(cursor, sourceCode, "", result)
}

// extractGoImports extracts import statements from Go code
func (p *Parser) extractGoImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findGoImports(cursor, sourceCode, result)
}

// findGoImports recursively finds import declarations
func (p *Parser) findGoImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "import_declaration" {
		p.parseGoImportDeclaration(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findGoImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseGoImportDeclaration parses a Go import declaration
func (p *Parser) parseGoImportDeclaration(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "import_spec" {
			p.parseGoImportSpec(child, sourceCode, result)
		} else if child.Type() == "import_spec_list" {
			// Multiple imports in parentheses
			for j := 0; j < int(child.ChildCount()); j++ {
				spec := child.Child(j)
				if spec.Type() == "import_spec" {
					p.parseGoImportSpec(spec, sourceCode, result)
				}
			}
		}
	}
}

// parseGoImportSpec parses a single import spec
func (p *Parser) parseGoImportSpec(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType = "default"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "interpreted_string_literal":
			// Remove quotes
			path := getNodeText(child, sourceCode)
			importPath = strings.Trim(path, "\"")
		case "package_identifier":
			// Named import (alias)
			alias := getNodeText(child, sourceCode)
			if alias == "." {
				importType = "wildcard"
			} else {
				importType = "named"
				importedSymbols = append(importedSymbols, alias)
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

// walkGoTree walks the Go AST tree
func (p *Parser) walkGoTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_declaration":
		p.parseGoFunction(node, sourceCode, scope, result)
	case "method_declaration":
		p.parseGoMethod(node, sourceCode, scope, result)
	case "type_declaration":
		p.parseGoType(node, sourceCode, scope, result)
	case "const_declaration", "var_declaration":
		p.parseGoVar(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkGoTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseGoFunction parses a Go function declaration
func (p *Parser) parseGoFunction(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)
	visibility := "private"
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		visibility = "public"
	}

	// Get function signature
	signature := extractGoSignature(node, sourceCode)

	// Get docstring (preceding comment)
	docstring := findPrecedingComment(node, sourceCode)

	// Get function body
	bodyNode := findChildByType(node, "block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
		// Extract call sites from the body
		p.extractGoCallSites(bodyNode, sourceCode, name, result)
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

// parseGoMethod parses a Go method declaration
func (p *Parser) parseGoMethod(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "field_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)
	visibility := "private"
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		visibility = "public"
	}

	// Get receiver type
	receiverNode := findChildByType(node, "parameter_list")
	receiverType := ""
	if receiverNode != nil && receiverNode.ChildCount() > 0 {
		receiverType = extractGoReceiverType(receiverNode, sourceCode)
	}

	// Get method signature
	signature := extractGoSignature(node, sourceCode)

	// Get docstring
	docstring := findPrecedingComment(node, sourceCode)

	// Get method body
	bodyNode := findChildByType(node, "block")
	codeBody := ""
	if bodyNode != nil {
		codeBody = getNodeText(bodyNode, sourceCode)
		// Extract call sites from the body
		callerName := name
		if receiverType != "" {
			callerName = receiverType + "." + name
		}
		p.extractGoCallSites(bodyNode, sourceCode, callerName, result)
	}

	symbol := &Symbol{
		Name:        name,
		Kind:        "method",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       receiverType,
		Visibility:  visibility,
		Docstring:   docstring,
		Signature:   signature,
		CodeBody:    codeBody,
	}

	result.Symbols = append(result.Symbols, symbol)
}

// parseGoType parses a Go type declaration
func (p *Parser) parseGoType(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// type_declaration contains type_spec nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_spec" {
			p.parseGoTypeSpec(child, sourceCode, scope, result)
		}
	}
}

// parseGoTypeSpec parses a Go type spec
func (p *Parser) parseGoTypeSpec(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "type_identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)
	visibility := "private"
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		visibility = "public"
	}

	// Determine type kind (struct, interface, etc.)
	kind := "type"
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "struct_type":
			kind = "struct"
		case "interface_type":
			kind = "interface"
		}
	}

	// Get docstring
	docstring := findPrecedingComment(node.Parent(), sourceCode)

	// Get type definition
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

// parseGoVar parses a Go variable/constant declaration
func (p *Parser) parseGoVar(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	kind := "variable"
	if node.Type() == "const_declaration" {
		kind = "constant"
	}

	// var/const declarations can have multiple specs
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "var_spec" || child.Type() == "const_spec" {
			p.parseGoVarSpec(child, sourceCode, scope, kind, result)
		}
	}
}

// parseGoVarSpec parses a Go variable spec
func (p *Parser) parseGoVarSpec(node *sitter.Node, sourceCode []byte, scope, kind string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)
	visibility := "private"
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		visibility = "public"
	}

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
		Signature:   signature,
		CodeBody:    signature,
	}

	result.Symbols = append(result.Symbols, symbol)
}

// extractGoCallSites extracts function/method calls from a code block
func (p *Parser) extractGoCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkGoCallSites(cursor, sourceCode, callerName, result)
}

// walkGoCallSites walks the AST to find call expressions
func (p *Parser) walkGoCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "call_expression":
		p.parseGoCallExpression(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkGoCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseGoCallExpression parses a call expression
func (p *Parser) parseGoCallExpression(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
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
	case "selector_expression":
		// Method call: obj.Method() or package.Function()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "method"
	case "parenthesized_expression":
		// Indirect call: (funcVar)()
		calleeName = getNodeText(funcNode, sourceCode)
		callType = "indirect"
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

// extractGoSignature extracts the function/method signature
func extractGoSignature(node *sitter.Node, sourceCode []byte) string {
	// Get the signature part (everything except the body)
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

// extractGoReceiverType extracts the receiver type from a parameter list
func extractGoReceiverType(node *sitter.Node, sourceCode []byte) string {
	if node.ChildCount() == 0 {
		return ""
	}

	// The first parameter is the receiver
	param := node.Child(0)
	if param.Type() == "parameter_declaration" {
		typeNode := findChildByType(param, "pointer_type")
		if typeNode == nil {
			typeNode = findChildByType(param, "type_identifier")
		}
		if typeNode != nil {
			return strings.TrimPrefix(getNodeText(typeNode, sourceCode), "*")
		}
	}

	return ""
}
