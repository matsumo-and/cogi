package parser

import (
	"encoding/json"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseJSON parses JSON source code
func (p *Parser) parseJSON(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// Try to parse as JSON to extract structure
	var data interface{}
	if err := json.Unmarshal(sourceCode, &data); err != nil {
		// If JSON parsing fails, treat as plain text
		p.parseJSONAsText(sourceCode, result)
		return
	}

	// Walk JSON structure to extract keys
	p.walkJSONStructure(data, "", result)
}

// walkJSONStructure recursively walks JSON structure to extract symbols
func (p *Parser) walkJSONStructure(data interface{}, path string, result *ParseResult) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			currentPath := key
			if path != "" {
				currentPath = path + "." + key
			}

			// Create symbol for each key
			symbol := &Symbol{
				Name:        currentPath,
				Kind:        "property",
				StartLine:   0, // JSON parsing doesn't preserve line numbers easily
				StartColumn: 0,
				EndLine:     0,
				EndColumn:   0,
				Visibility:  "public",
				Signature:   fmt.Sprintf("%s: %T", currentPath, value),
			}

			result.Symbols = append(result.Symbols, symbol)

			// Recurse into nested structures
			p.walkJSONStructure(value, currentPath, result)
		}
	case []interface{}:
		// For arrays, we could optionally index array items
		// For now, we just note the array type
		if path != "" {
			symbol := &Symbol{
				Name:        path,
				Kind:        "array",
				StartLine:   0,
				StartColumn: 0,
				EndLine:     0,
				EndColumn:   0,
				Visibility:  "public",
				Signature:   fmt.Sprintf("%s: array[%d]", path, len(v)),
			}
			result.Symbols = append(result.Symbols, symbol)
		}
	}
}

// parseJSONAsText parses malformed JSON as plain text
func (p *Parser) parseJSONAsText(sourceCode []byte, result *ParseResult) {
	symbol := &Symbol{
		Name:        "json-document",
		Kind:        "document",
		StartLine:   1,
		StartColumn: 1,
		EndLine:     countLines(sourceCode),
		EndColumn:   1,
		Visibility:  "public",
		CodeBody:    string(sourceCode),
	}

	result.Symbols = append(result.Symbols, symbol)
}
