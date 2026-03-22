package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
	"github.com/matsumo_and/cogi/internal/graph"
	"github.com/matsumo_and/cogi/internal/indexer"
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

// handleAddRepository handles adding a repository to the index
func (s *Server) handleAddRepository(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params AddRepositoryParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	if params.Path == "" {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(params.Path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to resolve path: %v", err)), nil
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("path does not exist: %s", absPath)), nil
	}

	// Use directory name as repository name if not specified
	name := params.Name
	if name == "" {
		name = filepath.Base(absPath)
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	// Check if repository already exists
	if exists, _ := database.RepositoryExists(name, absPath); exists {
		return mcp.NewToolResultError(fmt.Sprintf("repository already indexed: %s", name)), nil
	}

	// Create repository entry
	repoID, err := database.CreateRepository(name, absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add repository: %v", err)), nil
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":       true,
		"repository_id": repoID,
		"name":          name,
		"path":          absPath,
		"message":       fmt.Sprintf("Repository '%s' added. Run indexing to build the code index.", name),
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleRemoveRepository handles removing a repository from the index
func (s *Server) handleRemoveRepository(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params RemoveRepositoryParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	if params.Name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	repo, err := database.GetRepositoryByName(params.Name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("repository not found: %s", params.Name)), nil
	}

	if err := database.DeleteRepository(repo.ID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove repository: %v", err)), nil
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Repository '%s' removed from index", params.Name),
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleListRepositories handles listing all repositories
func (s *Server) handleListRepositories(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	repos, err := database.ListRepositories()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list repositories: %v", err)), nil
	}

	repoList := make([]map[string]interface{}, len(repos))
	for i, repo := range repos {
		repoList[i] = map[string]interface{}{
			"id":            repo.ID,
			"name":          repo.Name,
			"path":          repo.Path,
			"last_indexed":  repo.LastIndexedAt,
			
		}
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"repositories": repoList,
		"count":        len(repos),
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleStatus handles status query
func (s *Server) handleStatus(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	repos, err := database.ListRepositories()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list repositories: %v", err)), nil
	}

	idx := indexer.New(database, s.config)
	repoStats := make([]map[string]interface{}, len(repos))

	for i, repo := range repos {
		stats, err := idx.GetStats(repo.ID)
		if err != nil {
			repoStats[i] = map[string]interface{}{
				"name":  repo.Name,
				"path":  repo.Path,
				"error": err.Error(),
			}
			continue
		}

		repoStats[i] = map[string]interface{}{
			"name":          repo.Name,
			"path":          repo.Path,
			"files":         stats.TotalFiles,
			"symbols":       stats.TotalSymbols,
			"last_indexed":  repo.LastIndexedAt,
		}
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"repositories": repoStats,
		"total_repos":  len(repos),
		"database":     s.config.Database.Path,
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleIndex handles repository indexing
func (s *Server) handleIndex(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params IndexRepositoryParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	var repos []*db.Repository
	if params.Repository != "" {
		repo, err := database.GetRepositoryByName(params.Repository)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("repository not found: %s", params.Repository)), nil
		}
		repos = []*db.Repository{repo}
	} else {
		repos, err = database.ListRepositories()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list repositories: %v", err)), nil
		}
	}

	if len(repos) == 0 {
		return mcp.NewToolResultError("no repositories to index"), nil
	}

	idx := indexer.New(database, s.config)
	results := make([]map[string]interface{}, len(repos))

	for i, repo := range repos {
		err := idx.IndexRepository(context.Background(), repo.ID, repo.Path, params.Full)
		if err != nil {
			results[i] = map[string]interface{}{
				"name":    repo.Name,
				"success": false,
				"error":   err.Error(),
			}
			continue
		}

		stats, _ := idx.GetStats(repo.ID)
		results[i] = map[string]interface{}{
			"name":    repo.Name,
			"success": true,
			"files":   stats.TotalFiles,
			"symbols": stats.TotalSymbols,
		}
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGraphCalls handles call graph queries
func (s *Server) handleGraphCalls(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params GraphCallsParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	if params.SymbolName == "" {
		return mcp.NewToolResultError("symbol_name parameter is required"), nil
	}
	if params.Direction == "" {
		params.Direction = "caller"
	}
	if params.Depth <= 0 {
		params.Depth = 3
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	gm := graph.New(database)

	var nodes []*graph.CallNode
	if params.Direction == "caller" {
		nodes, err = gm.GetCallersTree(params.SymbolName, params.Depth)
	} else {
		nodes, err = gm.GetCalleesTree(params.SymbolName, params.Depth)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get call graph: %v", err)), nil
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"symbol":    params.SymbolName,
		"direction": params.Direction,
		"depth":     params.Depth,
		"nodes":     nodes,
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGraphImports handles import graph queries
func (s *Server) handleGraphImports(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params GraphImportsParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	if params.FilePath == "" {
		return mcp.NewToolResultError("file_path parameter is required"), nil
	}
	if params.Direction == "" {
		params.Direction = "dependency"
	}
	if params.Depth <= 0 {
		params.Depth = 3
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	gm := graph.New(database)

	var nodes []*graph.ImportNode
	if params.Direction == "importer" {
		nodes, err = gm.GetImporters(params.FilePath, params.Depth)
	} else {
		nodes, err = gm.GetImportDependencies(params.FilePath, params.Depth)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get import graph: %v", err)), nil
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"file":      params.FilePath,
		"direction": params.Direction,
		"depth":     params.Depth,
		"nodes":     nodes,
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleOwnership handles code ownership queries
func (s *Server) handleOwnership(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var params OwnershipParams
	paramsJSON, err := json.Marshal(arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal parameters: %v", err)), nil
	}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse parameters: %v", err)), nil
	}

	if params.Mode == "" {
		return mcp.NewToolResultError("mode parameter is required (file, author, or top)"), nil
	}

	database, err := s.initDatabase()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open database: %v", err)), nil
	}
	defer func() { _ = database.Close() }()

	var resultData interface{}

	switch params.Mode {
	case "file":
		if params.File == "" {
			return mcp.NewToolResultError("file parameter is required for file mode"), nil
		}

		// Find file by path
		repos, _ := database.ListRepositories()
		var fileID int64
		found := false
		for _, repo := range repos {
			file, err := database.GetFileByPath(repo.ID, params.File)
			if err == nil {
				fileID = file.ID
				found = true
				break
			}
		}
		if !found {
			return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", params.File)), nil
		}

		if params.Line > 0 {
			ownership, err := database.GetOwnershipByLine(fileID, params.Line)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get ownership: %v", err)), nil
			}
			resultData = ownership
		} else {
			ownerships, err := database.GetOwnershipByFile(fileID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get ownership: %v", err)), nil
			}
			resultData = ownerships
		}

	case "author":
		if params.Author == "" {
			return mcp.NewToolResultError("author parameter is required for author mode"), nil
		}
		ownerships, err := database.GetOwnershipByAuthor(params.Author)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get ownership: %v", err)), nil
		}
		resultData = ownerships

	case "top":
		limit := params.Limit
		if limit <= 0 {
			limit = 10
		}
		authors, err := database.GetTopAuthors(limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get top authors: %v", err)), nil
		}
		resultData = authors

	default:
		return mcp.NewToolResultError(fmt.Sprintf("invalid mode: %s (must be file, author, or top)", params.Mode)), nil
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"mode": params.Mode,
		"data": resultData,
	}, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}
