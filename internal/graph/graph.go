package graph

import (
	"fmt"
	"strings"

	"github.com/matsumo_and/cogi/internal/db"
)

// GraphManager handles graph operations
type GraphManager struct {
	db *db.DB
}

// New creates a new GraphManager
func New(database *db.DB) *GraphManager {
	return &GraphManager{
		db: database,
	}
}

// CallNode represents a node in the call graph
type CallNode struct {
	Symbol   *db.Symbol
	Depth    int
	CallType string
}

// ImportNode represents a node in the import graph
type ImportNode struct {
	FilePath   string
	ImportPath string
	ImportType string
	Symbols    []string
	Depth      int
	LineNumber int
}

// GetCallersTree retrieves all callers of a symbol as a tree
func (gm *GraphManager) GetCallersTree(symbolName string, depth int) ([]*CallNode, error) {
	// Find the symbol first
	symbols, err := gm.db.SearchSymbolsByName(symbolName)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbol: %w", err)
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbol not found: %s", symbolName)
	}

	// Use the first match
	symbol := symbols[0]

	// Get callers recursively
	callGraphs, err := gm.db.GetCallersRecursive(symbol.ID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get callers: %w", err)
	}

	// Convert to CallNodes with symbol information
	var nodes []*CallNode
	seenSymbols := make(map[int64]bool)

	for _, cg := range callGraphs {
		if seenSymbols[cg.CallerSymbolID] {
			continue
		}
		seenSymbols[cg.CallerSymbolID] = true

		callerSymbol, err := gm.db.GetSymbol(cg.CallerSymbolID)
		if err != nil {
			continue
		}

		nodes = append(nodes, &CallNode{
			Symbol:   callerSymbol,
			Depth:    0, // Depth calculation would need enhancement
			CallType: cg.CallType,
		})
	}

	return nodes, nil
}

// GetCalleesTree retrieves all callees of a symbol as a tree
func (gm *GraphManager) GetCalleesTree(symbolName string, depth int) ([]*CallNode, error) {
	// Find the symbol first
	symbols, err := gm.db.SearchSymbolsByName(symbolName)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbol: %w", err)
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbol not found: %s", symbolName)
	}

	// Use the first match
	symbol := symbols[0]

	// Get callees recursively
	callGraphs, err := gm.db.GetCalleesRecursive(symbol.ID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get callees: %w", err)
	}

	// Convert to CallNodes with symbol information
	var nodes []*CallNode
	seenNames := make(map[string]bool)

	for _, cg := range callGraphs {
		if seenNames[cg.CalleeName] {
			continue
		}
		seenNames[cg.CalleeName] = true

		var calleeSymbol *db.Symbol
		if cg.CalleeSymbolID.Valid {
			calleeSymbol, _ = gm.db.GetSymbol(cg.CalleeSymbolID.Int64)
		}

		// If we don't have the symbol, create a placeholder
		if calleeSymbol == nil {
			calleeSymbol = &db.Symbol{
				Name: cg.CalleeName,
				Kind: "unknown",
			}
		}

		nodes = append(nodes, &CallNode{
			Symbol:   calleeSymbol,
			Depth:    0,
			CallType: cg.CallType,
		})
	}

	return nodes, nil
}

// GetImportDependencies retrieves all dependencies (imports) of a file
func (gm *GraphManager) GetImportDependencies(filePath string, depth int) ([]*ImportNode, error) {
	// Find the file
	// Note: We need repository ID - for now, search all repos
	// TODO: Add proper repository context
	files, err := gm.db.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var fileID int64
	var found bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, filePath) || f.Path == filePath {
			fileID = f.ID
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Get dependencies recursively
	importGraphs, err := gm.db.GetDependenciesRecursive(fileID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Convert to ImportNodes
	var nodes []*ImportNode
	for _, ig := range importGraphs {
		// Parse imported symbols JSON
		symbols := parseImportedSymbols(ig.ImportedSymbols)

		nodes = append(nodes, &ImportNode{
			FilePath:   filePath,
			ImportPath: ig.ImportPath,
			ImportType: ig.ImportType,
			Symbols:    symbols,
			Depth:      0,
			LineNumber: ig.LineNumber,
		})
	}

	return nodes, nil
}

// GetImporters retrieves all files that import a given file
func (gm *GraphManager) GetImporters(filePath string, depth int) ([]*ImportNode, error) {
	// Find the file
	files, err := gm.db.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var fileID int64
	var found bool
	for _, f := range files {
		if strings.HasSuffix(f.Path, filePath) || f.Path == filePath {
			fileID = f.ID
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Get importers recursively
	importGraphs, err := gm.db.GetImportersRecursive(fileID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get importers: %w", err)
	}

	// Convert to ImportNodes
	var nodes []*ImportNode
	for _, ig := range importGraphs {
		// Get file path
		file, err := gm.db.GetFile(ig.FileID)
		if err != nil {
			continue
		}

		symbols := parseImportedSymbols(ig.ImportedSymbols)

		nodes = append(nodes, &ImportNode{
			FilePath:   file.Path,
			ImportPath: ig.ImportPath,
			ImportType: ig.ImportType,
			Symbols:    symbols,
			Depth:      0,
			LineNumber: ig.LineNumber,
		})
	}

	return nodes, nil
}

// FormatCallTree formats a call tree as a string
func FormatCallTree(nodes []*CallNode, direction string) string {
	if len(nodes) == 0 {
		return "No results found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Call Graph (%s):\n", direction))
	sb.WriteString(strings.Repeat("-", 80))
	sb.WriteString("\n\n")

	for _, node := range nodes {
		indent := strings.Repeat("  ", node.Depth)
		sb.WriteString(fmt.Sprintf("%s%s %s.%s (%s:%d)\n",
			indent,
			getCallTypeSymbol(node.CallType),
			node.Symbol.Scope,
			node.Symbol.Name,
			"file",
			node.Symbol.StartLine,
		))
	}

	return sb.String()
}

// FormatImportTree formats an import tree as a string
func FormatImportTree(nodes []*ImportNode, direction string) string {
	if len(nodes) == 0 {
		return "No results found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Import Graph (%s):\n", direction))
	sb.WriteString(strings.Repeat("-", 80))
	sb.WriteString("\n\n")

	for _, node := range nodes {
		indent := strings.Repeat("  ", node.Depth)
		symbolsStr := ""
		if len(node.Symbols) > 0 {
			symbolsStr = fmt.Sprintf(" { %s }", strings.Join(node.Symbols, ", "))
		}

		sb.WriteString(fmt.Sprintf("%s%s %s%s (%s:%d)\n",
			indent,
			getImportTypeSymbol(node.ImportType),
			node.ImportPath,
			symbolsStr,
			node.FilePath,
			node.LineNumber,
		))
	}

	return sb.String()
}

// Helper functions

func getCallTypeSymbol(callType string) string {
	switch callType {
	case "direct":
		return "→"
	case "method":
		return "⇒"
	case "indirect":
		return "⤷"
	default:
		return "•"
	}
}

func getImportTypeSymbol(importType string) string {
	switch importType {
	case "named":
		return "⊕"
	case "default":
		return "⊙"
	case "wildcard":
		return "⊛"
	default:
		return "•"
	}
}

func parseImportedSymbols(jsonStr string) []string {
	if jsonStr == "" {
		return nil
	}

	// Simple JSON array parsing (just remove brackets and quotes)
	jsonStr = strings.Trim(jsonStr, "[]")
	jsonStr = strings.ReplaceAll(jsonStr, `"`, "")

	if jsonStr == "" {
		return nil
	}

	return strings.Split(jsonStr, ",")
}
