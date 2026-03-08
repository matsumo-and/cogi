package search

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
)

// Result represents a search result
type Result struct {
	SymbolID    int64
	SymbolName  string
	SymbolKind  string
	FilePath    string
	Language    string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Signature   string
	Docstring   string
	CodeBody    string
	Score       float32 // Relevance score (for semantic search or BM25)
	Snippet     string  // Highlighted snippet
}

// scoredResult represents a symbol ID with a similarity score
type scoredResult struct {
	symbolID int64
	score    float32
}

// KeywordSearcher performs keyword-based search using FTS5
type KeywordSearcher struct {
	db *db.DB
}

// NewKeywordSearcher creates a new keyword searcher
func NewKeywordSearcher(database *db.DB) *KeywordSearcher {
	return &KeywordSearcher{db: database}
}

// KeywordSearchOptions contains options for keyword search
type KeywordSearchOptions struct {
	Query      string
	Language   string // Filter by language (optional)
	SymbolKind string // Filter by symbol kind (optional)
	Repository string // Filter by repository name (optional)
	Limit      int    // Maximum number of results
}

// Search performs FTS5 keyword search
func (s *KeywordSearcher) Search(ctx context.Context, opts KeywordSearchOptions) ([]Result, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if opts.Limit <= 0 {
		opts.Limit = 20 // Default limit
	}

	// Build FTS5 query
	// Escape special characters and prepare for FTS5
	ftsQuery := prepareFTS5Query(opts.Query)

	// Build SQL query with filters
	query := `
		SELECT
			s.id,
			s.name,
			s.kind,
			f.path,
			f.language,
			s.start_line,
			s.start_column,
			s.end_line,
			s.end_column,
			s.signature,
			s.docstring,
			s.code_body,
			bm25(symbols_fts) as score,
			snippet(symbols_fts, 4, '<mark>', '</mark>', '...', 64) as snippet
		FROM symbols_fts
		JOIN symbols s ON symbols_fts.rowid = s.id
		JOIN files f ON s.file_id = f.id
		JOIN repositories r ON f.repository_id = r.id
		WHERE symbols_fts MATCH ?
	`

	args := []interface{}{ftsQuery}

	// Add filters
	if opts.Language != "" {
		query += " AND f.language = ?"
		args = append(args, opts.Language)
	}
	if opts.SymbolKind != "" {
		query += " AND s.kind = ?"
		args = append(args, opts.SymbolKind)
	}
	if opts.Repository != "" {
		query += " AND r.name = ?"
		args = append(args, opts.Repository)
	}

	query += " ORDER BY score DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	return scanKeywordResults(rows)
}

// SemanticSearcher performs semantic search using vector similarity
type SemanticSearcher struct {
	db          *db.DB
	embedClient embedding.Client
}

// NewSemanticSearcher creates a new semantic searcher
func NewSemanticSearcher(database *db.DB, embedClient embedding.Client) *SemanticSearcher {
	return &SemanticSearcher{
		db:          database,
		embedClient: embedClient,
	}
}

// SemanticSearchOptions contains options for semantic search
type SemanticSearchOptions struct {
	Query       string
	Granularity string // 'class' or 'function' or empty for both
	Language    string // Filter by language (optional)
	SymbolKind  string // Filter by symbol kind (optional)
	Repository  string // Filter by repository name (optional)
	Limit       int    // Maximum number of results
}

// Search performs semantic vector search
func (s *SemanticSearcher) Search(ctx context.Context, opts SemanticSearchOptions) ([]Result, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if opts.Limit <= 0 {
		opts.Limit = 20 // Default limit
	}

	// Generate embedding for query
	embed, err := s.embedClient.EmbedSingle(ctx, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get all embeddings from database with optional granularity filter
	embeddings, err := s.db.GetAllEmbeddings(opts.Granularity)
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings: %w", err)
	}

	if len(embeddings) == 0 {
		return []Result{}, nil
	}

	// Build symbol details query with filters
	query := `
		SELECT
			s.id,
			s.name,
			s.kind,
			f.path,
			f.language,
			s.start_line,
			s.start_column,
			s.end_line,
			s.end_column,
			s.signature,
			s.docstring,
			s.code_body
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		JOIN repositories r ON f.repository_id = r.id
		WHERE 1=1
	`
	args := []interface{}{}

	if opts.Language != "" {
		query += " AND f.language = ?"
		args = append(args, opts.Language)
	}
	if opts.SymbolKind != "" {
		query += " AND s.kind = ?"
		args = append(args, opts.SymbolKind)
	}
	if opts.Repository != "" {
		query += " AND r.name = ?"
		args = append(args, opts.Repository)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch symbols: %w", err)
	}
	defer rows.Close()

	// Scan all matching symbols
	symbols, err := scanResults(rows)
	if err != nil {
		return nil, err
	}

	// Create map of symbol ID to symbol
	symbolMap := make(map[int64]*Result)
	for i := range symbols {
		symbolMap[symbols[i].SymbolID] = &symbols[i]
	}

	// Calculate cosine similarity for each embedding and collect results
	scoredResults := make([]scoredResult, 0, len(embeddings))

	for _, emb := range embeddings {
		// Skip file-level embeddings (no symbol ID)
		if emb.SymbolID == nil {
			continue
		}

		// Skip if symbol was filtered out
		if _, ok := symbolMap[*emb.SymbolID]; !ok {
			continue
		}

		// Calculate cosine similarity
		score := cosineSimilarity(embed.Vector, emb.Vector)
		scoredResults = append(scoredResults, scoredResult{
			symbolID: *emb.SymbolID,
			score:    score,
		})
	}

	// Sort by score descending
	sortScoredResults(scoredResults)

	// Limit and build final results
	if len(scoredResults) > opts.Limit {
		scoredResults = scoredResults[:opts.Limit]
	}

	results := make([]Result, 0, len(scoredResults))
	for _, sr := range scoredResults {
		if symbol, ok := symbolMap[sr.symbolID]; ok {
			symbol.Score = sr.score
			results = append(results, *symbol)
		}
	}

	return results, nil
}

// HybridSearcher combines keyword and semantic search
type HybridSearcher struct {
	keywordSearcher  *KeywordSearcher
	semanticSearcher *SemanticSearcher
}

// NewHybridSearcher creates a new hybrid searcher
func NewHybridSearcher(keywordSearcher *KeywordSearcher, semanticSearcher *SemanticSearcher) *HybridSearcher {
	return &HybridSearcher{
		keywordSearcher:  keywordSearcher,
		semanticSearcher: semanticSearcher,
	}
}

// HybridSearchOptions contains options for hybrid search
type HybridSearchOptions struct {
	Query          string
	Language       string
	SymbolKind     string
	Repository     string
	Limit          int
	KeywordWeight  float32 // Weight for keyword results (0-1)
	SemanticWeight float32 // Weight for semantic results (0-1)
}

// Search performs hybrid search combining keyword and semantic results
func (h *HybridSearcher) Search(ctx context.Context, opts HybridSearchOptions) ([]Result, error) {
	if opts.KeywordWeight == 0 && opts.SemanticWeight == 0 {
		opts.KeywordWeight = 0.5
		opts.SemanticWeight = 0.5
	}

	// Perform both searches in parallel
	type searchResult struct {
		results []Result
		err     error
	}

	keywordChan := make(chan searchResult, 1)
	semanticChan := make(chan searchResult, 1)

	// Keyword search
	go func() {
		results, err := h.keywordSearcher.Search(ctx, KeywordSearchOptions{
			Query:      opts.Query,
			Language:   opts.Language,
			SymbolKind: opts.SymbolKind,
			Repository: opts.Repository,
			Limit:      opts.Limit,
		})
		keywordChan <- searchResult{results, err}
	}()

	// Semantic search
	go func() {
		results, err := h.semanticSearcher.Search(ctx, SemanticSearchOptions{
			Query:      opts.Query,
			Language:   opts.Language,
			SymbolKind: opts.SymbolKind,
			Repository: opts.Repository,
			Limit:      opts.Limit,
		})
		semanticChan <- searchResult{results, err}
	}()

	// Wait for results
	keywordRes := <-keywordChan
	semanticRes := <-semanticChan

	if keywordRes.err != nil && semanticRes.err != nil {
		return nil, fmt.Errorf("both searches failed: keyword=%v, semantic=%v", keywordRes.err, semanticRes.err)
	}

	// Merge results with weighted scores
	merged := mergeResults(keywordRes.results, semanticRes.results, opts.KeywordWeight, opts.SemanticWeight)

	// Limit results
	if len(merged) > opts.Limit {
		merged = merged[:opts.Limit]
	}

	return merged, nil
}

// Helper functions

func prepareFTS5Query(query string) string {
	// Simple FTS5 query preparation
	// For more complex queries, this can be enhanced
	query = strings.TrimSpace(query)

	// If query contains spaces, treat as phrase for better results
	if strings.Contains(query, " ") {
		return fmt.Sprintf("\"%s\"", query)
	}

	return query
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func scanResults(rows *sql.Rows) ([]Result, error) {
	results := []Result{}

	for rows.Next() {
		var r Result

		err := rows.Scan(
			&r.SymbolID,
			&r.SymbolName,
			&r.SymbolKind,
			&r.FilePath,
			&r.Language,
			&r.StartLine,
			&r.StartColumn,
			&r.EndLine,
			&r.EndColumn,
			&r.Signature,
			&r.Docstring,
			&r.CodeBody,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

func scanKeywordResults(rows *sql.Rows) ([]Result, error) {
	results := []Result{}

	for rows.Next() {
		var r Result
		var snippet sql.NullString

		err := rows.Scan(
			&r.SymbolID,
			&r.SymbolName,
			&r.SymbolKind,
			&r.FilePath,
			&r.Language,
			&r.StartLine,
			&r.StartColumn,
			&r.EndLine,
			&r.EndColumn,
			&r.Signature,
			&r.Docstring,
			&r.CodeBody,
			&r.Score,
			&snippet,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if snippet.Valid {
			r.Snippet = snippet.String
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

func sortByScore(results []Result) {
	// Simple bubble sort by score (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func mergeResults(keyword, semantic []Result, kwWeight, semWeight float32) []Result {
	// Create a map to deduplicate by symbol ID
	resultMap := make(map[int64]*Result)

	// Add keyword results
	for _, r := range keyword {
		r.Score *= kwWeight
		resultMap[r.SymbolID] = &r
	}

	// Add/merge semantic results
	for _, r := range semantic {
		r.Score *= semWeight
		if existing, ok := resultMap[r.SymbolID]; ok {
			// Combine scores
			existing.Score += r.Score
		} else {
			resultMap[r.SymbolID] = &r
		}
	}

	// Convert map to slice
	merged := make([]Result, 0, len(resultMap))
	for _, r := range resultMap {
		merged = append(merged, *r)
	}

	// Sort by combined score
	sortByScore(merged)

	return merged
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// sortScoredResults sorts scored results by score descending
func sortScoredResults(results []scoredResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
