package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/search"
)

// Server represents the MCP server instance
type Server struct {
	mcpServer *server.MCPServer
	config    *config.Config
}

// New creates a new MCP server instance
func New() (*Server, error) {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	s := &Server{
		config: cfg,
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"cogi",
		"1.0.0",
	)

	// Register all search tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return s, nil
}

// Start starts the MCP server with stdio transport
func (s *Server) Start(ctx context.Context) error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers all MCP tools with their schemas
func (s *Server) registerTools() error {
	// Tool 1: Symbol Search
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_search_symbol",
		Description: "Search for code symbols (functions, classes, variables, etc.) by name or kind",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Symbol name to search for (required)",
				},
				"kind": map[string]interface{}{
					"type":        "string",
					"description": "Filter by symbol kind (optional: function, class, variable, method, struct, etc.)",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Filter by repository name (optional)",
				},
			},
			Required: []string{"name"},
		},
	}, s.handleSymbolSearch)

	// Tool 2: Keyword Search
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_search_keyword",
		Description: "Full-text keyword search across indexed codebases using SQLite FTS5 with BM25 ranking",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (supports FTS5 syntax)",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by programming language (optional: go, python, javascript, etc.)",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Filter by repository name (optional)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 20, max 100)",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleKeywordSearch)

	// Tool 3: Semantic Search
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_search_semantic",
		Description: "Semantic code search using vector embeddings and cosine similarity (requires Ollama)",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language query describing what you're looking for",
				},
				"granularity": map[string]interface{}{
					"type":        "string",
					"description": "Search granularity (optional: class for class-level, function for function-level)",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by programming language (optional)",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Filter by repository name (optional)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 10)",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleSemanticSearch)

	// Tool 4: Hybrid Search
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_search_hybrid",
		Description: "Hybrid search combining keyword (BM25) and semantic (vector) search with weighted scoring",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"kind": map[string]interface{}{
					"type":        "string",
					"description": "Filter by symbol kind (optional)",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by programming language (optional)",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Filter by repository name (optional)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results (default 10)",
				},
				"keyword_weight": map[string]interface{}{
					"type":        "number",
					"description": "Weight for keyword results 0.0-1.0 (default 0.3)",
				},
				"semantic_weight": map[string]interface{}{
					"type":        "number",
					"description": "Weight for semantic results 0.0-1.0 (default 0.7)",
				},
			},
			Required: []string{"query"},
		},
	}, s.handleHybridSearch)

	// Tool 5: Add Repository
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_add_repository",
		Description: "Add a repository to Cogi's index for code analysis",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the repository directory",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Custom name for the repository (optional, uses directory name if not specified)",
				},
			},
			Required: []string{"path"},
		},
	}, s.handleAddRepository)

	// Tool 6: Remove Repository
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_remove_repository",
		Description: "Remove a repository from Cogi's index",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the repository to remove",
				},
			},
			Required: []string{"name"},
		},
	}, s.handleRemoveRepository)

	// Tool 7: List Repositories
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_list_repositories",
		Description: "List all indexed repositories",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleListRepositories)

	// Tool 8: Status
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_status",
		Description: "Get status and statistics of all indexed repositories",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleStatus)

	// Tool 9: Index Repository
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_index",
		Description: "Build or update code index for repositories",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Name of specific repository to index (optional, indexes all if not specified)",
				},
				"full": map[string]interface{}{
					"type":        "boolean",
					"description": "Force full reindex instead of incremental (optional, default false)",
				},
			},
		},
	}, s.handleIndex)

	// Tool 10: Call Graph
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_graph_calls",
		Description: "Get call graph showing who calls a symbol or what a symbol calls",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"symbol_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the symbol to analyze",
				},
				"direction": map[string]interface{}{
					"type":        "string",
					"description": "Direction: 'caller' (who calls this) or 'callee' (what this calls). Default: caller",
				},
				"depth": map[string]interface{}{
					"type":        "number",
					"description": "Depth of graph traversal (default 3)",
				},
			},
			Required: []string{"symbol_name", "direction"},
		},
	}, s.handleGraphCalls)

	// Tool 11: Import Graph
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_graph_imports",
		Description: "Get import graph showing file dependencies",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to analyze",
				},
				"direction": map[string]interface{}{
					"type":        "string",
					"description": "Direction: 'importer' (who imports this file) or 'dependency' (what this file imports). Default: dependency",
				},
				"depth": map[string]interface{}{
					"type":        "number",
					"description": "Depth of graph traversal (default 3)",
				},
			},
			Required: []string{"file_path", "direction"},
		},
	}, s.handleGraphImports)

	// Tool 12: Code Ownership
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "cogi_ownership",
		Description: "Query code ownership information based on git blame",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Query mode: 'file' (ownership by file), 'author' (files by author), or 'top' (top contributors)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "File path (required for mode=file)",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (optional, for mode=file)",
				},
				"author": map[string]interface{}{
					"type":        "string",
					"description": "Author name (required for mode=author)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Number of results (for mode=top, default 10)",
				},
			},
			Required: []string{"mode"},
		},
	}, s.handleOwnership)

	return nil
}

// initDatabase initializes database connection with config
func (s *Server) initDatabase() (*db.DB, error) {
	database, err := db.Open(s.config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return database, nil
}

// convertResultsToJSON converts search.Result slice to JSON-friendly format
func convertResultsToJSON(results []search.Result) []SearchResult {
	jsonResults := make([]SearchResult, len(results))
	for i, r := range results {
		jsonResults[i] = SearchResult{
			SymbolName:  r.SymbolName,
			SymbolKind:  r.SymbolKind,
			FilePath:    r.FilePath,
			Language:    r.Language,
			StartLine:   r.StartLine,
			StartColumn: r.StartColumn,
			EndLine:     r.EndLine,
			EndColumn:   r.EndColumn,
			Signature:   r.Signature,
			Docstring:   r.Docstring,
			CodeBody:    r.CodeBody,
			Score:       r.Score,
			Snippet:     r.Snippet,
		}
	}
	return jsonResults
}
