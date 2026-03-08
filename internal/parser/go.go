package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseGo parses Go source code
func (p *Parser) parseGo(root *sitter.Node, sourceCode []byte) []*Symbol {
	var symbols []*Symbol

	// Walk the tree
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkGoTree(cursor, sourceCode, "", &symbols)

	return symbols
}

// walkGoTree walks the Go AST tree
func (p *Parser) walkGoTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, symbols *[]*Symbol) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "function_declaration":
		p.parseGoFunction(node, sourceCode, scope, symbols)
	case "method_declaration":
		p.parseGoMethod(node, sourceCode, scope, symbols)
	case "type_declaration":
		p.parseGoType(node, sourceCode, scope, symbols)
	case "const_declaration", "var_declaration":
		p.parseGoVar(node, sourceCode, scope, symbols)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkGoTree(cursor, sourceCode, scope, symbols)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseGoFunction parses a Go function declaration
func (p *Parser) parseGoFunction(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
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

// parseGoMethod parses a Go method declaration
func (p *Parser) parseGoMethod(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
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

	*symbols = append(*symbols, symbol)
}

// parseGoType parses a Go type declaration
func (p *Parser) parseGoType(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	// type_declaration contains type_spec nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_spec" {
			p.parseGoTypeSpec(child, sourceCode, scope, symbols)
		}
	}
}

// parseGoTypeSpec parses a Go type spec
func (p *Parser) parseGoTypeSpec(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
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

	*symbols = append(*symbols, symbol)
}

// parseGoVar parses a Go variable/constant declaration
func (p *Parser) parseGoVar(node *sitter.Node, sourceCode []byte, scope string, symbols *[]*Symbol) {
	kind := "variable"
	if node.Type() == "const_declaration" {
		kind = "constant"
	}

	// var/const declarations can have multiple specs
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "var_spec" || child.Type() == "const_spec" {
			p.parseGoVarSpec(child, sourceCode, scope, kind, symbols)
		}
	}
}

// parseGoVarSpec parses a Go variable spec
func (p *Parser) parseGoVarSpec(node *sitter.Node, sourceCode []byte, scope, kind string, symbols *[]*Symbol) {
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

	*symbols = append(*symbols, symbol)
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
