package db

import (
	"database/sql"
	"errors"
	"fmt"
)

// ImportGraph represents a file/module import relationship
type ImportGraph struct {
	ID              int64
	FileID          int64
	ImportPath      string
	ImportType      string // named, default, wildcard
	ImportedSymbols string // JSON array
	LineNumber      int
}

// CreateImportGraph creates a new import graph record
func (db *DB) CreateImportGraph(ig *ImportGraph) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO import_graph (
			file_id, import_path, import_type, imported_symbols, line_number
		) VALUES (?, ?, ?, ?, ?)
	`, ig.FileID, ig.ImportPath, ig.ImportType, ig.ImportedSymbols, ig.LineNumber)

	if err != nil {
		return 0, fmt.Errorf("failed to create import graph: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get import graph ID: %w", err)
	}

	return id, nil
}

// BatchCreateImportGraph creates multiple import graph records in a single transaction
func (db *DB) BatchCreateImportGraph(importGraphs []*ImportGraph) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to rollback: %w", rerr))
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO import_graph (
			file_id, import_path, import_type, imported_symbols, line_number
		) VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if cerr := stmt.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close statement: %w", cerr))
		}
	}()

	for _, ig := range importGraphs {
		_, err = stmt.Exec(ig.FileID, ig.ImportPath, ig.ImportType,
			ig.ImportedSymbols, ig.LineNumber)
		if err != nil {
			return fmt.Errorf("failed to insert import graph: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetImportGraphByFile retrieves all import graph entries for a file
func (db *DB) GetImportGraphByFile(fileID int64) ([]*ImportGraph, error) {
	rows, err := db.Query(`
		SELECT id, file_id, import_path, import_type, imported_symbols, line_number
		FROM import_graph
		WHERE file_id = ?
		ORDER BY line_number
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get import graph by file: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanImportGraphs(rows)
}

// GetImportGraphByPath retrieves all import graph entries by import path
func (db *DB) GetImportGraphByPath(importPath string) ([]*ImportGraph, error) {
	rows, err := db.Query(`
		SELECT id, file_id, import_path, import_type, imported_symbols, line_number
		FROM import_graph
		WHERE import_path = ?
		ORDER BY line_number
	`, importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get import graph by path: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanImportGraphs(rows)
}

// GetImportGraphByPathPattern retrieves import graph entries matching a path pattern
func (db *DB) GetImportGraphByPathPattern(pattern string) ([]*ImportGraph, error) {
	rows, err := db.Query(`
		SELECT id, file_id, import_path, import_type, imported_symbols, line_number
		FROM import_graph
		WHERE import_path LIKE ?
		ORDER BY import_path, line_number
	`, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get import graph by pattern: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanImportGraphs(rows)
}

// GetImportersRecursive retrieves all files that import a given path recursively
func (db *DB) GetImportersRecursive(fileID int64, depth int) ([]*ImportGraph, error) {
	if depth <= 0 {
		return nil, nil
	}

	// First, get the file path for the given fileID
	var filePath string
	err := db.QueryRow("SELECT path FROM files WHERE id = ?", fileID).Scan(&filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file path: %w", err)
	}

	// Use recursive CTE to find all importers up to specified depth
	rows, err := db.Query(`
		WITH RECURSIVE importer_tree(id, file_id, import_path, import_type,
		                              imported_symbols, line_number, depth) AS (
			-- Base case: direct importers
			SELECT ig.id, ig.file_id, ig.import_path, ig.import_type,
			       ig.imported_symbols, ig.line_number, 1 as depth
			FROM import_graph ig
			JOIN files f ON ig.file_id = f.id
			WHERE ig.import_path LIKE '%' || ? || '%'

			UNION ALL

			-- Recursive case: importers of importers
			SELECT ig.id, ig.file_id, ig.import_path, ig.import_type,
			       ig.imported_symbols, ig.line_number, it.depth + 1
			FROM import_graph ig
			JOIN importer_tree it ON ig.import_path LIKE '%' || (
				SELECT path FROM files WHERE id = it.file_id
			) || '%'
			WHERE it.depth < ?
		)
		SELECT id, file_id, import_path, import_type, imported_symbols, line_number
		FROM importer_tree
		ORDER BY depth, line_number
	`, filePath, depth)

	if err != nil {
		return nil, fmt.Errorf("failed to get importers recursively: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanImportGraphs(rows)
}

// GetDependenciesRecursive retrieves all dependencies of a file recursively
func (db *DB) GetDependenciesRecursive(fileID int64, depth int) ([]*ImportGraph, error) {
	if depth <= 0 {
		return nil, nil
	}

	// Use recursive CTE to find all dependencies up to specified depth
	rows, err := db.Query(`
		WITH RECURSIVE dependency_tree(id, file_id, import_path, import_type,
		                                imported_symbols, line_number, depth) AS (
			-- Base case: direct dependencies
			SELECT id, file_id, import_path, import_type,
			       imported_symbols, line_number, 1 as depth
			FROM import_graph
			WHERE file_id = ?

			UNION ALL

			-- Recursive case: dependencies of dependencies
			SELECT ig.id, ig.file_id, ig.import_path, ig.import_type,
			       ig.imported_symbols, ig.line_number, dt.depth + 1
			FROM import_graph ig
			JOIN dependency_tree dt ON ig.file_id IN (
				SELECT id FROM files WHERE path LIKE '%' || dt.import_path || '%'
			)
			WHERE dt.depth < ?
		)
		SELECT id, file_id, import_path, import_type, imported_symbols, line_number
		FROM dependency_tree
		ORDER BY depth, line_number
	`, fileID, depth)

	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies recursively: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanImportGraphs(rows)
}

// DetectCircularDependencies detects circular import dependencies for a file
func (db *DB) DetectCircularDependencies(fileID int64) ([][]*ImportGraph, error) {
	// Get file path
	var filePath string
	err := db.QueryRow("SELECT path FROM files WHERE id = ?", fileID).Scan(&filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file path: %w", err)
	}

	// Find cycles using recursive CTE
	rows, err := db.Query(`
		WITH RECURSIVE dependency_path(file_id, import_path, path_trace, depth) AS (
			-- Base case
			SELECT file_id, import_path, import_path as path_trace, 1 as depth
			FROM import_graph
			WHERE file_id = ?

			UNION ALL

			-- Recursive case
			SELECT ig.file_id, ig.import_path,
			       dp.path_trace || ' -> ' || ig.import_path,
			       dp.depth + 1
			FROM import_graph ig
			JOIN dependency_path dp ON ig.file_id IN (
				SELECT id FROM files WHERE path LIKE '%' || dp.import_path || '%'
			)
			WHERE dp.depth < 100  -- Safety limit
			  AND dp.path_trace NOT LIKE '%' || ig.import_path || '%'  -- Avoid revisiting
		)
		SELECT file_id, import_path, path_trace
		FROM dependency_path
		WHERE import_path LIKE '%' || ? || '%'
		ORDER BY depth
	`, fileID, filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to detect circular dependencies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Parse results - this is a simplified version
	// In practice, we'd need to reconstruct the actual cycles
	var cycles [][]*ImportGraph
	// TODO: Implement cycle reconstruction from path traces

	return cycles, nil
}

// DeleteImportGraphByFile deletes all import graph entries for a file
func (db *DB) DeleteImportGraphByFile(fileID int64) error {
	_, err := db.Exec("DELETE FROM import_graph WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete import graph: %w", err)
	}
	return nil
}

// scanImportGraphs is a helper function to scan multiple import graph rows
func scanImportGraphs(rows *sql.Rows) ([]*ImportGraph, error) {
	var importGraphs []*ImportGraph

	for rows.Next() {
		var ig ImportGraph
		var importType, importedSymbols sql.NullString

		err := rows.Scan(&ig.ID, &ig.FileID, &ig.ImportPath, &importType,
			&importedSymbols, &ig.LineNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to scan import graph: %w", err)
		}

		if importType.Valid {
			ig.ImportType = importType.String
		}
		if importedSymbols.Valid {
			ig.ImportedSymbols = importedSymbols.String
		}

		importGraphs = append(importGraphs, &ig)
	}

	return importGraphs, nil
}

// GetImportGraphCount returns the total number of import graph entries
func (db *DB) GetImportGraphCount() (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM import_graph").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get import graph count: %w", err)
	}
	return count, nil
}

// GetImportGraphCountByRepository returns the number of import graph entries for a repository
func (db *DB) GetImportGraphCountByRepository(repositoryID int64) (int64, error) {
	var count int64
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM import_graph ig
		JOIN files f ON ig.file_id = f.id
		WHERE f.repository_id = ?
	`, repositoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get import graph count: %w", err)
	}
	return count, nil
}
