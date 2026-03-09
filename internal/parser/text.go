package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// parseText parses plain text files (fallback for unsupported languages)
func (p *Parser) parseText(root *sitter.Node, sourceCode []byte, result *ParseResult) {
	// For plain text, we create a single document symbol
	// containing the entire file content

	lines := countLines(sourceCode)

	symbol := &Symbol{
		Name:        "text-document",
		Kind:        "document",
		StartLine:   1,
		StartColumn: 1,
		EndLine:     lines,
		EndColumn:   1,
		Visibility:  "public",
		CodeBody:    string(sourceCode),
	}

	result.Symbols = append(result.Symbols, symbol)

	// Optionally extract sections based on common patterns
	p.extractTextSections(sourceCode, result)
}

// extractTextSections tries to extract meaningful sections from text
func (p *Parser) extractTextSections(sourceCode []byte, result *ParseResult) {
	lines := strings.Split(string(sourceCode), "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Detect section headers (lines that end with colon or are all caps)
		if isSectionHeader(trimmed) {
			symbol := &Symbol{
				Name:        trimmed,
				Kind:        "section",
				StartLine:   i + 1,
				StartColumn: 1,
				EndLine:     i + 1,
				EndColumn:   len(line),
				Visibility:  "public",
				Signature:   trimmed,
				CodeBody:    trimmed,
			}

			result.Symbols = append(result.Symbols, symbol)
		}
	}
}

// isSectionHeader detects if a line looks like a section header
func isSectionHeader(line string) bool {
	// Check if line ends with colon
	if strings.HasSuffix(line, ":") && len(line) > 1 && len(line) < 100 {
		return true
	}

	// Check if line is all uppercase (common for headers)
	if len(line) > 3 && len(line) < 100 && strings.ToUpper(line) == line {
		// Make sure it contains at least some letters
		hasLetter := false
		for _, r := range line {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				hasLetter = true
				break
			}
		}
		return hasLetter
	}

	return false
}

// countLines counts the number of lines in the source code
func countLines(sourceCode []byte) int {
	if len(sourceCode) == 0 {
		return 0
	}

	count := 1
	for _, b := range sourceCode {
		if b == '\n' {
			count++
		}
	}

	return count
}
