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
