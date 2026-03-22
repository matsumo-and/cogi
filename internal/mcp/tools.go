package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
	"github.com/matsumo_and/cogi/internal/search"
)

// handleSymbolSearch handles symbol search requests
func (s *Server) handleSymbolSearch(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse parameters
	var params SymbolSearchParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	// Validate required parameters
	if params.Name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	// Initialize database
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	// Search symbols
	var symbols []*db.Symbol
	if params.Kind != "" {
		symbols, err = database.SearchSymbolsByKind(params.Kind)
	} else {
		symbols, err = database.SearchSymbolsByName(params.Name)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search symbols: %v", err)), nil
	}

	// Filter by repository if specified
	var filteredSymbols []*db.Symbol
	if params.Repository != "" {
		for _, sym := range symbols {
			file, err := database.GetFile(sym.FileID)
			if err != nil {
				continue
			}
			repo, err := database.GetRepository(file.RepositoryID)
			if err != nil {
				continue
			}
			if repo.Name == params.Repository {
				filteredSymbols = append(filteredSymbols, sym)
			}
		}
		symbols = filteredSymbols
	}

	// Convert to response format
	results := make([]SearchResult, 0, len(symbols))
	for _, sym := range symbols {
		file, err := database.GetFile(sym.FileID)
		if err != nil {
			continue
		}

		results = append(results, SearchResult{
			SymbolName:  sym.Name,
			SymbolKind:  sym.Kind,
			FilePath:    file.Path,
			Language:    file.Language,
			StartLine:   sym.StartLine,
			StartColumn: sym.StartColumn,
			EndLine:     sym.EndLine,
			EndColumn:   sym.EndColumn,
			Signature:   sym.Signature,
			Docstring:   sym.Docstring,
			CodeBody:    sym.CodeBody,
		})
	}

	// Marshal results to JSON
	resultJSON, err := json.MarshalIndent(map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleKeywordSearch handles keyword search requests
func (s *Server) handleKeywordSearch(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse parameters
	var params KeywordSearchParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	// Validate required parameters
	if params.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// Set default limit
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Initialize database
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	// Perform keyword search
	searcher := search.NewKeywordSearcher(database)
	results, err := searcher.Search(context.Background(), search.KeywordSearchOptions{
		Query:      params.Query,
		Language:   params.Language,
		Repository: params.Repository,
		Limit:      params.Limit,
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("keyword search failed: %v", err)), nil
	}

	// Convert to response format
	jsonResults := convertResultsToJSON(results)

	// Marshal results to JSON
	resultJSON, err := json.MarshalIndent(map[string]interface{}{
		"results": jsonResults,
		"count":   len(jsonResults),
	}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleSemanticSearch handles semantic search requests
func (s *Server) handleSemanticSearch(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse parameters
	var params SemanticSearchParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	// Validate required parameters
	if params.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// Set default limit
	if params.Limit <= 0 {
		params.Limit = 10
	}

	// Initialize database
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	// Initialize Ollama embedding client
	embedClient := embedding.NewOllamaClient(
		s.config.Embedding.Endpoint,
		s.config.Embedding.Model,
		s.config.Embedding.Dimension,
	)

	// Perform semantic search
	searcher := search.NewSemanticSearcher(database, embedClient)
	results, err := searcher.Search(context.Background(), search.SemanticSearchOptions{
		Query:       params.Query,
		Granularity: params.Granularity,
		Language:    params.Language,
		Repository:  params.Repository,
		Limit:       params.Limit,
	})

	if err != nil {
		errorMsg := fmt.Sprintf("Semantic search failed: %v\n\nMake sure Ollama is running:\n  ollama serve\n  ollama pull %s", err, s.config.Embedding.Model)
		return mcp.NewToolResultError(errorMsg), nil
	}

	// Convert to response format
	jsonResults := convertResultsToJSON(results)

	// Marshal results to JSON
	resultJSON, err := json.MarshalIndent(map[string]interface{}{
		"results": jsonResults,
		"count":   len(jsonResults),
	}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleHybridSearch handles hybrid search requests
func (s *Server) handleHybridSearch(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Parse parameters
	var params HybridSearchParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	// Validate required parameters
	if params.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// Set defaults
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.KeywordWeight == 0 && params.SemanticWeight == 0 {
		params.KeywordWeight = 0.3
		params.SemanticWeight = 0.7
	}

	// Initialize database
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	// Initialize Ollama embedding client
	embedClient := embedding.NewOllamaClient(
		s.config.Embedding.Endpoint,
		s.config.Embedding.Model,
		s.config.Embedding.Dimension,
	)

	// Create searchers
	keywordSearcher := search.NewKeywordSearcher(database)
	semanticSearcher := search.NewSemanticSearcher(database, embedClient)
	hybridSearcher := search.NewHybridSearcher(keywordSearcher, semanticSearcher)

	// Perform hybrid search
	results, err := hybridSearcher.Search(context.Background(), search.HybridSearchOptions{
		Query:          params.Query,
		SymbolKind:     params.Kind,
		Language:       params.Language,
		Repository:     params.Repository,
		Limit:          params.Limit,
		KeywordWeight:  float32(params.KeywordWeight),
		SemanticWeight: float32(params.SemanticWeight),
	})

	if err != nil {
		errorMsg := fmt.Sprintf("Hybrid search failed: %v\n\nMake sure Ollama is running:\n  ollama serve\n  ollama pull %s", err, s.config.Embedding.Model)
		return mcp.NewToolResultError(errorMsg), nil
	}

	// Convert to response format
	jsonResults := convertResultsToJSON(results)

	// Marshal results to JSON
	resultJSON, err := json.MarshalIndent(map[string]interface{}{
		"results": jsonResults,
		"count":   len(jsonResults),
		"weights": map[string]interface{}{
			"keyword":  params.KeywordWeight,
			"semantic": params.SemanticWeight,
		},
	}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}
