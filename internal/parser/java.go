package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseJava parses Java source code
func (p *Parser) parseJava(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Extract imports first
	p.extractJavaImports(root, sourceCode, result)

	// Walk the tree for symbols and calls
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkJavaTree(cursor, sourceCode, "", result)
}

// extractJavaImports extracts import statements from Java code
func (p *Parser) extractJavaImports(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.findJavaImports(cursor, sourceCode, result)
}

// findJavaImports recursively finds import declarations
func (p *Parser) findJavaImports(cursor *sitter.TreeCursor, sourceCode []byte, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "import_declaration" {
		p.parseJavaImportDeclaration(node, sourceCode, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.findJavaImports(cursor, sourceCode, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseJavaImportDeclaration parses a Java import declaration
func (p *Parser) parseJavaImportDeclaration(node *sitter.Node, sourceCode []byte, result *ParseResult) {
	var importPath string
	var importType = "named"
	var importedSymbols []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "scoped_identifier":
			importPath = getNodeText(child, sourceCode)
			// Extract the class name from the path
			parts := strings.Split(importPath, ".")
			if len(parts) > 0 {
				importedSymbols = append(importedSymbols, parts[len(parts)-1])
			}
		case "asterisk":
			// import package.*
			importType = "wildcard"
			importedSymbols = append(importedSymbols, "*")
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

// walkJavaTree walks the Java AST tree
func (p *Parser) walkJavaTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "class_declaration":
		p.parseJavaClass(node, sourceCode, scope, result)
	case "interface_declaration":
		p.parseJavaInterface(node, sourceCode, scope, result)
	case "enum_declaration":
		p.parseJavaEnum(node, sourceCode, scope, result)
	case "method_declaration":
		p.parseJavaMethod(node, sourceCode, scope, result)
	case "field_declaration":
		p.parseJavaField(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkJavaTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseJavaClass parses a Java class declaration
func (p *Parser) parseJavaClass(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getJavaVisibility(node, sourceCode)

	// Get doc comment
	docstring := findPrecedingComment(node, sourceCode)

	// Get class signature
	signature := extractJavaClassSignature(node, sourceCode)

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

// parseJavaInterface parses a Java interface declaration
func (p *Parser) parseJavaInterface(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getJavaVisibility(node, sourceCode)

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

// parseJavaEnum parses a Java enum declaration
func (p *Parser) parseJavaEnum(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getJavaVisibility(node, sourceCode)

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

// parseJavaMethod parses a Java method declaration
func (p *Parser) parseJavaMethod(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	nameNode := findChildByType(node, "identifier")
	if nameNode == nil {
		return
	}

	name := getNodeText(nameNode, sourceCode)

	// Get visibility
	visibility := getJavaVisibility(node, sourceCode)

	// Get method signature
	signature := extractJavaMethodSignature(node, sourceCode)

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
		p.extractJavaCallSites(bodyNode, sourceCode, callerName, result)
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

// parseJavaField parses a Java field declaration
func (p *Parser) parseJavaField(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Field declaration can have multiple variable declarators
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "variable_declarator" {
			nameNode := findChildByType(child, "identifier")
			if nameNode == nil {
				continue
			}

			name := getNodeText(nameNode, sourceCode)

			// Get visibility
			visibility := getJavaVisibility(node, sourceCode)

			// Get doc comment
			docstring := findPrecedingComment(node, sourceCode)

			signature := getNodeText(node, sourceCode)

			symbol := &Symbol{
				Name:        name,
				Kind:        "field",
				StartLine:   int(child.StartPoint().Row) + 1,
				StartColumn: int(child.StartPoint().Column) + 1,
				EndLine:     int(child.EndPoint().Row) + 1,
				EndColumn:   int(child.EndPoint().Column) + 1,
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

// extractJavaCallSites extracts method calls from a code block
func (p *Parser) extractJavaCallSites(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	p.walkJavaCallSites(cursor, sourceCode, callerName, result)
}

// walkJavaCallSites walks the AST to find method invocations
func (p *Parser) walkJavaCallSites(cursor *sitter.TreeCursor, sourceCode []byte, callerName string, result *ParseResult) {
	node := cursor.CurrentNode()

	if node.Type() == "method_invocation" {
		p.parseJavaMethodInvocation(node, sourceCode, callerName, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkJavaCallSites(cursor, sourceCode, callerName, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseJavaMethodInvocation parses a method invocation
func (p *Parser) parseJavaMethodInvocation(node *sitter.Node, sourceCode []byte, callerName string, result *ParseResult) {
	var calleeName string
	var callType = "direct"

	nameNode := findChildByType(node, "identifier")
	if nameNode != nil {
		calleeName = getNodeText(nameNode, sourceCode)
		callType = "direct"
	}

	// Check for object/class method call
	objectNode := findChildByType(node, "field_access")
	if objectNode != nil {
		calleeName = getNodeText(objectNode, sourceCode)
		callType = "method"
	}

	if calleeName == "" {
		// Try to get the full invocation text
		calleeName = getNodeText(node, sourceCode)
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

// getJavaVisibility extracts visibility modifier from a Java node
func getJavaVisibility(node *sitter.Node, sourceCode []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "modifiers" {
			text := getNodeText(child, sourceCode)
			if strings.Contains(text, "public") {
				return "public"
			} else if strings.Contains(text, "private") {
				return "private"
			} else if strings.Contains(text, "protected") {
				return "protected"
			}
		}
	}
	return "package" // Default package-private
}

// extractJavaClassSignature extracts the Java class signature
func extractJavaClassSignature(node *sitter.Node, sourceCode []byte) string {
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

// extractJavaMethodSignature extracts the Java method signature
func extractJavaMethodSignature(node *sitter.Node, sourceCode []byte) string {
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
