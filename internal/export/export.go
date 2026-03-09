package export

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/matsumo_and/cogi/internal/db"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
)

// SymbolExport represents a symbol for export
type SymbolExport struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	StartColumn int    `json:"start_column"`
	EndLine     int    `json:"end_line"`
	EndColumn   int    `json:"end_column"`
	Scope       string `json:"scope,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	Docstring   string `json:"docstring,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Repository  string `json:"repository"`
	Language    string `json:"language"`
}

// CallGraphExport represents a call graph edge for export
type CallGraphExport struct {
	ID              int64  `json:"id"`
	CallerSymbolID  int64  `json:"caller_symbol_id"`
	CalleeSymbolID  *int64 `json:"callee_symbol_id,omitempty"`
	CallerName      string `json:"caller_name"`
	CalleeName      string `json:"callee_name"`
	CallLine        int    `json:"call_line"`
	CallColumn      int    `json:"call_column"`
	CallType        string `json:"call_type"`
	CallerFilePath  string `json:"caller_file_path"`
	CallerSignature string `json:"caller_signature"`
}

// ImportGraphExport represents an import for export
type ImportGraphExport struct {
	ID              int64    `json:"id"`
	FilePath        string   `json:"file_path"`
	ImportPath      string   `json:"import_path"`
	ImportType      string   `json:"import_type"`
	ImportedSymbols []string `json:"imported_symbols"`
	LineNumber      int      `json:"line_number"`
	Repository      string   `json:"repository"`
	Language        string   `json:"language"`
}

// OwnershipExport represents ownership information for export
type OwnershipExport struct {
	ID             int64     `json:"id"`
	FilePath       string    `json:"file_path"`
	StartLine      int       `json:"start_line"`
	EndLine        int       `json:"end_line"`
	AuthorName     string    `json:"author_name"`
	AuthorEmail    string    `json:"author_email"`
	LastCommitHash string    `json:"last_commit_hash"`
	LastCommitDate time.Time `json:"last_commit_date"`
	CommitCount    int       `json:"commit_count"`
	Repository     string    `json:"repository"`
}

// RepositoryExport represents repository metadata for export
type RepositoryExport struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ExportData represents all exportable data
type ExportData struct {
	ExportedAt   time.Time              `json:"exported_at"`
	Version      string                 `json:"version"`
	Repositories []RepositoryExport     `json:"repositories,omitempty"`
	Symbols      []SymbolExport         `json:"symbols,omitempty"`
	CallGraph    []CallGraphExport      `json:"call_graph,omitempty"`
	ImportGraph  []ImportGraphExport    `json:"import_graph,omitempty"`
	Ownership    []OwnershipExport      `json:"ownership,omitempty"`
}

// Exporter handles data export
type Exporter struct {
	db *db.DB
}

// New creates a new Exporter
func New(database *db.DB) *Exporter {
	return &Exporter{
		db: database,
	}
}

// ExportAll exports all data to the given writer
func (e *Exporter) ExportAll(writer io.Writer, format ExportFormat) error {
	if format != FormatJSON {
		return fmt.Errorf("unsupported format: %s", format)
	}

	data := &ExportData{
		ExportedAt: time.Now(),
		Version:    "1.0",
	}

	// Export repositories
	repos, err := e.exportRepositories()
	if err != nil {
		return fmt.Errorf("failed to export repositories: %w", err)
	}
	data.Repositories = repos

	// Export symbols
	symbols, err := e.exportSymbols()
	if err != nil {
		return fmt.Errorf("failed to export symbols: %w", err)
	}
	data.Symbols = symbols

	// Export call graph
	callGraph, err := e.exportCallGraph()
	if err != nil {
		return fmt.Errorf("failed to export call graph: %w", err)
	}
	data.CallGraph = callGraph

	// Export import graph
	importGraph, err := e.exportImportGraph()
	if err != nil {
		return fmt.Errorf("failed to export import graph: %w", err)
	}
	data.ImportGraph = importGraph

	// Export ownership
	ownership, err := e.exportOwnership()
	if err != nil {
		return fmt.Errorf("failed to export ownership: %w", err)
	}
	data.Ownership = ownership

	// Encode to JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// ExportSymbols exports only symbols to the given writer
func (e *Exporter) ExportSymbols(writer io.Writer, format ExportFormat, repositoryID *int64) error {
	if format != FormatJSON {
		return fmt.Errorf("unsupported format: %s", format)
	}

	symbols, err := e.exportSymbolsFiltered(repositoryID)
	if err != nil {
		return fmt.Errorf("failed to export symbols: %w", err)
	}

	data := map[string]interface{}{
		"exported_at": time.Now(),
		"version":     "1.0",
		"symbols":     symbols,
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportRepositories exports all repositories
func (e *Exporter) exportRepositories() ([]RepositoryExport, error) {
	repos, err := e.db.ListRepositories()
	if err != nil {
		return nil, err
	}

	result := make([]RepositoryExport, len(repos))
	for i, repo := range repos {
		lastIndexedAt := time.Time{}
		if repo.LastIndexedAt != nil {
			lastIndexedAt = *repo.LastIndexedAt
		}

		result[i] = RepositoryExport{
			ID:            repo.ID,
			Name:          repo.Name,
			Path:          repo.Path,
			LastIndexedAt: lastIndexedAt,
			CreatedAt:     repo.CreatedAt,
			UpdatedAt:     repo.UpdatedAt,
		}
	}

	return result, nil
}

// exportSymbols exports all symbols
func (e *Exporter) exportSymbols() ([]SymbolExport, error) {
	return e.exportSymbolsFiltered(nil)
}

// exportSymbolsFiltered exports symbols, optionally filtered by repository
func (e *Exporter) exportSymbolsFiltered(repositoryID *int64) ([]SymbolExport, error) {
	query := `
		SELECT
			s.id, s.name, s.kind, s.start_line, s.start_column, s.end_line, s.end_column,
			s.scope, s.visibility, s.docstring, s.signature,
			f.path as file_path, f.language,
			r.name as repository_name
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		JOIN repositories r ON f.repository_id = r.id
	`

	args := []interface{}{}
	if repositoryID != nil {
		query += " WHERE f.repository_id = ?"
		args = append(args, *repositoryID)
	}

	query += " ORDER BY s.id"

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []SymbolExport
	for rows.Next() {
		var s SymbolExport
		var scope, visibility, docstring, signature *string

		err := rows.Scan(
			&s.ID, &s.Name, &s.Kind, &s.StartLine, &s.StartColumn, &s.EndLine, &s.EndColumn,
			&scope, &visibility, &docstring, &signature,
			&s.FilePath, &s.Language, &s.Repository,
		)
		if err != nil {
			return nil, err
		}

		if scope != nil {
			s.Scope = *scope
		}
		if visibility != nil {
			s.Visibility = *visibility
		}
		if docstring != nil {
			s.Docstring = *docstring
		}
		if signature != nil {
			s.Signature = *signature
		}

		symbols = append(symbols, s)
	}

	return symbols, rows.Err()
}

// exportCallGraph exports the call graph
func (e *Exporter) exportCallGraph() ([]CallGraphExport, error) {
	query := `
		SELECT
			cg.id, cg.caller_symbol_id, cg.callee_symbol_id, cg.callee_name,
			cg.call_line, cg.call_column, cg.call_type,
			s.name as caller_name, s.signature as caller_signature,
			f.path as caller_file_path
		FROM call_graph cg
		JOIN symbols s ON cg.caller_symbol_id = s.id
		JOIN files f ON s.file_id = f.id
		ORDER BY cg.id
	`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callGraph []CallGraphExport
	for rows.Next() {
		var cg CallGraphExport

		err := rows.Scan(
			&cg.ID, &cg.CallerSymbolID, &cg.CalleeSymbolID, &cg.CalleeName,
			&cg.CallLine, &cg.CallColumn, &cg.CallType,
			&cg.CallerName, &cg.CallerSignature, &cg.CallerFilePath,
		)
		if err != nil {
			return nil, err
		}

		callGraph = append(callGraph, cg)
	}

	return callGraph, rows.Err()
}

// exportImportGraph exports the import graph
func (e *Exporter) exportImportGraph() ([]ImportGraphExport, error) {
	query := `
		SELECT
			ig.id, ig.import_path, ig.import_type, ig.imported_symbols, ig.line_number,
			f.path as file_path, f.language,
			r.name as repository_name
		FROM import_graph ig
		JOIN files f ON ig.file_id = f.id
		JOIN repositories r ON f.repository_id = r.id
		ORDER BY ig.id
	`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var importGraph []ImportGraphExport
	for rows.Next() {
		var ig ImportGraphExport
		var importedSymbolsJSON *string

		err := rows.Scan(
			&ig.ID, &ig.ImportPath, &ig.ImportType, &importedSymbolsJSON, &ig.LineNumber,
			&ig.FilePath, &ig.Language, &ig.Repository,
		)
		if err != nil {
			return nil, err
		}

		// Parse imported symbols JSON
		if importedSymbolsJSON != nil && *importedSymbolsJSON != "" {
			if err := json.Unmarshal([]byte(*importedSymbolsJSON), &ig.ImportedSymbols); err != nil {
				ig.ImportedSymbols = []string{}
			}
		} else {
			ig.ImportedSymbols = []string{}
		}

		importGraph = append(importGraph, ig)
	}

	return importGraph, rows.Err()
}

// exportOwnership exports ownership information
func (e *Exporter) exportOwnership() ([]OwnershipExport, error) {
	query := `
		SELECT
			o.id, o.start_line, o.end_line, o.author_name, o.author_email,
			o.last_commit_hash, o.last_commit_date, o.commit_count,
			f.path as file_path,
			r.name as repository_name
		FROM ownership o
		JOIN files f ON o.file_id = f.id
		JOIN repositories r ON f.repository_id = r.id
		ORDER BY o.id
	`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ownership []OwnershipExport
	for rows.Next() {
		var o OwnershipExport
		var lastCommitDate int64

		err := rows.Scan(
			&o.ID, &o.StartLine, &o.EndLine, &o.AuthorName, &o.AuthorEmail,
			&o.LastCommitHash, &lastCommitDate, &o.CommitCount,
			&o.FilePath, &o.Repository,
		)
		if err != nil {
			return nil, err
		}

		o.LastCommitDate = time.Unix(lastCommitDate, 0)

		ownership = append(ownership, o)
	}

	return ownership, rows.Err()
}
