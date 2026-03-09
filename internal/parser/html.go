package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseHTML parses HTML source code
func (p *Parser) parseHTML(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Walk the tree for elements
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	p.walkHTMLTree(cursor, sourceCode, "", result)
}

// walkHTMLTree walks the HTML AST tree
func (p *Parser) walkHTMLTree(cursor *sitter.TreeCursor, sourceCode []byte, scope string, result *ParseResult) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "element":
		p.parseHTMLElement(node, sourceCode, scope, result)
	case "script_element":
		p.parseHTMLScriptElement(node, sourceCode, scope, result)
	case "style_element":
		p.parseHTMLStyleElement(node, sourceCode, scope, result)
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			p.walkHTMLTree(cursor, sourceCode, scope, result)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// parseHTMLElement parses an HTML element
func (p *Parser) parseHTMLElement(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Get start tag
	startTagNode := findChildByType(node, "start_tag")
	if startTagNode == nil {
		return
	}

	// Get tag name
	tagNameNode := findChildByType(startTagNode, "tag_name")
	if tagNameNode == nil {
		return
	}

	tagName := getNodeText(tagNameNode, sourceCode)

	// Get attributes (especially id and class)
	var elementID string
	var attributes []string

	for i := 0; i < int(startTagNode.ChildCount()); i++ {
		child := startTagNode.Child(i)
		if child.Type() == "attribute" {
			attrName, attrValue := p.parseHTMLAttribute(child, sourceCode)
			attributes = append(attributes, attrName)

			if attrName == "id" {
				elementID = attrValue
			}
		}
	}

	// Create symbol for element with id or significant tags
	significantTags := map[string]bool{
		"html": true, "head": true, "body": true, "header": true, "footer": true,
		"nav": true, "main": true, "section": true, "article": true, "aside": true,
		"div": true, "form": true, "table": true,
	}

	if elementID != "" || significantTags[tagName] {
		name := tagName
		if elementID != "" {
			name = tagName + "#" + elementID
		}

		// Build signature with attributes
		signature := "<" + tagName
		if len(attributes) > 0 {
			signature += " " + strings.Join(attributes, " ")
		}
		signature += ">"

		symbol := &Symbol{
			Name:        name,
			Kind:        "element",
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

// parseHTMLScriptElement parses an HTML script element
func (p *Parser) parseHTMLScriptElement(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	// Script element is similar to regular element
	name := "script"

	// Get start tag to extract attributes
	startTagNode := findChildByType(node, "start_tag")
	if startTagNode != nil {
		var attributes []string
		for i := 0; i < int(startTagNode.ChildCount()); i++ {
			child := startTagNode.Child(i)
			if child.Type() == "attribute" {
				attrName, _ := p.parseHTMLAttribute(child, sourceCode)
				attributes = append(attributes, attrName)
			}
		}

		signature := "<script"
		if len(attributes) > 0 {
			signature += " " + strings.Join(attributes, " ")
		}
		signature += ">"

		symbol := &Symbol{
			Name:        name,
			Kind:        "script",
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

// parseHTMLStyleElement parses an HTML style element
func (p *Parser) parseHTMLStyleElement(node *sitter.Node, sourceCode []byte, scope string, result *ParseResult) {
	name := "style"

	symbol := &Symbol{
		Name:        name,
		Kind:        "style",
		StartLine:   int(node.StartPoint().Row) + 1,
		StartColumn: int(node.StartPoint().Column) + 1,
		EndLine:     int(node.EndPoint().Row) + 1,
		EndColumn:   int(node.EndPoint().Column) + 1,
		Scope:       scope,
		Visibility:  "public",
		Signature:   "<style>",
		CodeBody:    getNodeText(node, sourceCode),
	}

	result.Symbols = append(result.Symbols, symbol)
}

// parseHTMLAttribute parses an HTML attribute
func (p *Parser) parseHTMLAttribute(node *sitter.Node, sourceCode []byte) (string, string) {
	var name, value string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "attribute_name":
			name = getNodeText(child, sourceCode)
		case "attribute_value", "quoted_attribute_value":
			value = getNodeText(child, sourceCode)
			// Remove quotes
			value = strings.Trim(value, `"'`)
		}
	}

	return name, value
}
