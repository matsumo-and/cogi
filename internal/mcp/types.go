package mcp

// SymbolSearchParams represents input parameters for symbol search
type SymbolSearchParams struct {
	Name       string `json:"name"`
	Kind       string `json:"kind,omitempty"`
	Repository string `json:"repository,omitempty"`
}

// KeywordSearchParams represents input parameters for keyword search
type KeywordSearchParams struct {
	Query      string `json:"query"`
	Language   string `json:"language,omitempty"`
	Repository string `json:"repository,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// SemanticSearchParams represents input parameters for semantic search
type SemanticSearchParams struct {
	Query       string `json:"query"`
	Granularity string `json:"granularity,omitempty"`
	Language    string `json:"language,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// HybridSearchParams represents input parameters for hybrid search
type HybridSearchParams struct {
	Query          string  `json:"query"`
	Kind           string  `json:"kind,omitempty"`
	Language       string  `json:"language,omitempty"`
	Repository     string  `json:"repository,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	KeywordWeight  float64 `json:"keyword_weight,omitempty"`
	SemanticWeight float64 `json:"semantic_weight,omitempty"`
}

// AddRepositoryParams represents input parameters for adding a repository
type AddRepositoryParams struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path"`
}

// RemoveRepositoryParams represents input parameters for removing a repository
type RemoveRepositoryParams struct {
	Name string `json:"name"`
}

// IndexRepositoryParams represents input parameters for indexing
type IndexRepositoryParams struct {
	Repository string `json:"repository,omitempty"`
	Full       bool   `json:"full,omitempty"`
}

// GraphCallsParams represents input parameters for call graph
type GraphCallsParams struct {
	SymbolName string `json:"symbol_name"`
	Direction  string `json:"direction"`
	Depth      int    `json:"depth,omitempty"`
}

// GraphImportsParams represents input parameters for import graph
type GraphImportsParams struct {
	FilePath  string `json:"file_path"`
	Direction string `json:"direction"`
	Depth     int    `json:"depth,omitempty"`
}

// OwnershipParams represents input parameters for ownership queries
type OwnershipParams struct {
	Mode   string `json:"mode"`
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Author string `json:"author,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// SearchResult represents a search result in MCP response format
type SearchResult struct {
	SymbolName  string  `json:"symbol_name"`
	SymbolKind  string  `json:"symbol_kind"`
	FilePath    string  `json:"file_path"`
	Language    string  `json:"language"`
	StartLine   int     `json:"start_line"`
	StartColumn int     `json:"start_column"`
	EndLine     int     `json:"end_line"`
	EndColumn   int     `json:"end_column"`
	Signature   string  `json:"signature,omitempty"`
	Docstring   string  `json:"docstring,omitempty"`
	CodeBody    string  `json:"code_body,omitempty"`
	Score       float32 `json:"score,omitempty"`
	Snippet     string  `json:"snippet,omitempty"`
}
