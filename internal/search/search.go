package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/matsumo_and/cogi/internal/db"
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
	defer func() { _ = rows.Close() }()

	return scanKeywordResults(rows)
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

