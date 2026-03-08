package db

import (
	"database/sql"
	"fmt"
)

// Symbol represents a code symbol (function, class, variable, etc.)
type Symbol struct {
	ID          int64
	FileID      int64
	Name        string
	Kind        string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Scope       string
	Visibility  string
	Docstring   string
	Signature   string
	CodeBody    string
}

// CreateSymbol creates a new symbol record
func (db *DB) CreateSymbol(symbol *Symbol) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO symbols (
			file_id, name, kind, start_line, start_column, end_line, end_column,
			scope, visibility, docstring, signature, code_body
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, symbol.FileID, symbol.Name, symbol.Kind, symbol.StartLine, symbol.StartColumn,
		symbol.EndLine, symbol.EndColumn, symbol.Scope, symbol.Visibility,
		symbol.Docstring, symbol.Signature, symbol.CodeBody)

	if err != nil {
		return 0, fmt.Errorf("failed to create symbol: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get symbol ID: %w", err)
	}

	return id, nil
}

// GetSymbol retrieves a symbol by ID
func (db *DB) GetSymbol(id int64) (*Symbol, error) {
	var symbol Symbol
	var scope, visibility, docstring, signature, codeBody sql.NullString

	err := db.QueryRow(`
		SELECT id, file_id, name, kind, start_line, start_column, end_line, end_column,
		       scope, visibility, docstring, signature, code_body
		FROM symbols
		WHERE id = ?
	`, id).Scan(&symbol.ID, &symbol.FileID, &symbol.Name, &symbol.Kind,
		&symbol.StartLine, &symbol.StartColumn, &symbol.EndLine, &symbol.EndColumn,
		&scope, &visibility, &docstring, &signature, &codeBody)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("symbol not found")
		}
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	if scope.Valid {
		symbol.Scope = scope.String
	}
	if visibility.Valid {
		symbol.Visibility = visibility.String
	}
	if docstring.Valid {
		symbol.Docstring = docstring.String
	}
	if signature.Valid {
		symbol.Signature = signature.String
	}
	if codeBody.Valid {
		symbol.CodeBody = codeBody.String
	}

	return &symbol, nil
}

// ListSymbolsByFile retrieves all symbols for a file
func (db *DB) ListSymbolsByFile(fileID int64) ([]*Symbol, error) {
	rows, err := db.Query(`
		SELECT id, file_id, name, kind, start_line, start_column, end_line, end_column,
		       scope, visibility, docstring, signature, code_body
		FROM symbols
		WHERE file_id = ?
		ORDER BY start_line, start_column
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to list symbols: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

// SearchSymbolsByName searches symbols by name (exact match)
func (db *DB) SearchSymbolsByName(name string) ([]*Symbol, error) {
	rows, err := db.Query(`
		SELECT id, file_id, name, kind, start_line, start_column, end_line, end_column,
		       scope, visibility, docstring, signature, code_body
		FROM symbols
		WHERE name = ?
		ORDER BY name
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

// SearchSymbolsByKind searches symbols by kind
func (db *DB) SearchSymbolsByKind(kind string) ([]*Symbol, error) {
	rows, err := db.Query(`
		SELECT id, file_id, name, kind, start_line, start_column, end_line, end_column,
		       scope, visibility, docstring, signature, code_body
		FROM symbols
		WHERE kind = ?
		ORDER BY name
	`, kind)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

// FullTextSearch performs full-text search on symbols using FTS5
func (db *DB) FullTextSearch(query string, limit int) ([]*Symbol, error) {
	rows, err := db.Query(`
		SELECT s.id, s.file_id, s.name, s.kind, s.start_line, s.start_column,
		       s.end_line, s.end_column, s.scope, s.visibility, s.docstring,
		       s.signature, s.code_body
		FROM symbols s
		JOIN symbols_fts fts ON s.id = fts.rowid
		WHERE symbols_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform full-text search: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

// DeleteSymbolsByFile deletes all symbols for a file
func (db *DB) DeleteSymbolsByFile(fileID int64) error {
	_, err := db.Exec("DELETE FROM symbols WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete symbols: %w", err)
	}
	return nil
}

// scanSymbols is a helper function to scan multiple symbol rows
func scanSymbols(rows *sql.Rows) ([]*Symbol, error) {
	var symbols []*Symbol

	for rows.Next() {
		var symbol Symbol
		var scope, visibility, docstring, signature, codeBody sql.NullString

		err := rows.Scan(&symbol.ID, &symbol.FileID, &symbol.Name, &symbol.Kind,
			&symbol.StartLine, &symbol.StartColumn, &symbol.EndLine, &symbol.EndColumn,
			&scope, &visibility, &docstring, &signature, &codeBody)
		if err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}

		if scope.Valid {
			symbol.Scope = scope.String
		}
		if visibility.Valid {
			symbol.Visibility = visibility.String
		}
		if docstring.Valid {
			symbol.Docstring = docstring.String
		}
		if signature.Valid {
			symbol.Signature = signature.String
		}
		if codeBody.Valid {
			symbol.CodeBody = codeBody.String
		}

		symbols = append(symbols, &symbol)
	}

	return symbols, nil
}

// GetSymbolCount returns the total number of symbols
func (db *DB) GetSymbolCount() (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get symbol count: %w", err)
	}
	return count, nil
}

// GetSymbolCountByRepository returns the number of symbols for a repository
func (db *DB) GetSymbolCountByRepository(repositoryID int64) (int64, error) {
	var count int64
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repository_id = ?
	`, repositoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get symbol count: %w", err)
	}
	return count, nil
}

// GetSymbolsByRepository retrieves all symbols for a repository
func (db *DB) GetSymbolsByRepository(repositoryID int64) ([]*Symbol, error) {
	rows, err := db.Query(`
		SELECT s.id, s.file_id, s.name, s.kind, s.start_line, s.start_column,
		       s.end_line, s.end_column, s.scope, s.visibility, s.docstring,
		       s.signature, s.code_body
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repository_id = ?
		ORDER BY s.name
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols by repository: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}
