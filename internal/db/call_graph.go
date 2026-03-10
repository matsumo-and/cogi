package db

import (
	"database/sql"
	"errors"
	"fmt"
)

// CallGraph represents a function/method call relationship
type CallGraph struct {
	ID             int64
	CallerSymbolID int64
	CalleeSymbolID sql.NullInt64 // Nullable for external functions
	CalleeName     string
	CallLine       int
	CallColumn     int
	CallType       string // direct, method, indirect
}

// CreateCallGraph creates a new call graph record
func (db *DB) CreateCallGraph(cg *CallGraph) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO call_graph (
			caller_symbol_id, callee_symbol_id, callee_name,
			call_line, call_column, call_type
		) VALUES (?, ?, ?, ?, ?, ?)
	`, cg.CallerSymbolID, cg.CalleeSymbolID, cg.CalleeName,
		cg.CallLine, cg.CallColumn, cg.CallType)

	if err != nil {
		return 0, fmt.Errorf("failed to create call graph: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get call graph ID: %w", err)
	}

	return id, nil
}

// BatchCreateCallGraph creates multiple call graph records in a single transaction
func (db *DB) BatchCreateCallGraph(callGraphs []*CallGraph) (err error) {
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
		INSERT INTO call_graph (
			caller_symbol_id, callee_symbol_id, callee_name,
			call_line, call_column, call_type
		) VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if cerr := stmt.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close statement: %w", cerr))
		}
	}()

	for _, cg := range callGraphs {
		_, err = stmt.Exec(cg.CallerSymbolID, cg.CalleeSymbolID, cg.CalleeName,
			cg.CallLine, cg.CallColumn, cg.CallType)
		if err != nil {
			return fmt.Errorf("failed to insert call graph: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCallGraphByCaller retrieves all call graph entries for a caller symbol
func (db *DB) GetCallGraphByCaller(callerSymbolID int64) (result []*CallGraph, err error) {
	rows, err := db.Query(`
		SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
		       call_line, call_column, call_type
		FROM call_graph
		WHERE caller_symbol_id = ?
		ORDER BY call_line, call_column
	`, callerSymbolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get call graph by caller: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows: %w", cerr))
		}
	}()

	return scanCallGraphs(rows)
}

// GetCallGraphByCallee retrieves all call graph entries for a callee symbol
func (db *DB) GetCallGraphByCallee(calleeSymbolID int64) (result []*CallGraph, err error) {
	rows, err := db.Query(`
		SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
		       call_line, call_column, call_type
		FROM call_graph
		WHERE callee_symbol_id = ?
		ORDER BY call_line, call_column
	`, calleeSymbolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get call graph by callee: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows: %w", cerr))
		}
	}()

	return scanCallGraphs(rows)
}

// GetCallGraphByCalleeName retrieves all call graph entries by callee name (for external functions)
func (db *DB) GetCallGraphByCalleeName(calleeName string) (result []*CallGraph, err error) {
	rows, err := db.Query(`
		SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
		       call_line, call_column, call_type
		FROM call_graph
		WHERE callee_name = ?
		ORDER BY call_line, call_column
	`, calleeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get call graph by callee name: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows: %w", cerr))
		}
	}()

	return scanCallGraphs(rows)
}

// GetCallersRecursive retrieves all callers of a symbol recursively up to specified depth
func (db *DB) GetCallersRecursive(symbolID int64, depth int) (result []*CallGraph, err error) {
	if depth <= 0 {
		return nil, nil
	}

	// Use recursive CTE to find all callers up to specified depth
	rows, err := db.Query(`
		WITH RECURSIVE caller_tree(id, caller_symbol_id, callee_symbol_id, callee_name,
		                            call_line, call_column, call_type, depth) AS (
			-- Base case: direct callers
			SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
			       call_line, call_column, call_type, 1 as depth
			FROM call_graph
			WHERE callee_symbol_id = ?

			UNION ALL

			-- Recursive case: callers of callers
			SELECT cg.id, cg.caller_symbol_id, cg.callee_symbol_id, cg.callee_name,
			       cg.call_line, cg.call_column, cg.call_type, ct.depth + 1
			FROM call_graph cg
			JOIN caller_tree ct ON cg.callee_symbol_id = ct.caller_symbol_id
			WHERE ct.depth < ?
		)
		SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
		       call_line, call_column, call_type
		FROM caller_tree
		ORDER BY depth, call_line, call_column
	`, symbolID, depth)

	if err != nil {
		return nil, fmt.Errorf("failed to get callers recursively: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows: %w", cerr))
		}
	}()

	return scanCallGraphs(rows)
}

// GetCalleesRecursive retrieves all callees of a symbol recursively up to specified depth
func (db *DB) GetCalleesRecursive(symbolID int64, depth int) (result []*CallGraph, err error) {
	if depth <= 0 {
		return nil, nil
	}

	// Use recursive CTE to find all callees up to specified depth
	rows, err := db.Query(`
		WITH RECURSIVE callee_tree(id, caller_symbol_id, callee_symbol_id, callee_name,
		                            call_line, call_column, call_type, depth) AS (
			-- Base case: direct callees
			SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
			       call_line, call_column, call_type, 1 as depth
			FROM call_graph
			WHERE caller_symbol_id = ?

			UNION ALL

			-- Recursive case: callees of callees
			SELECT cg.id, cg.caller_symbol_id, cg.callee_symbol_id, cg.callee_name,
			       cg.call_line, cg.call_column, cg.call_type, ct.depth + 1
			FROM call_graph cg
			JOIN callee_tree ct ON cg.caller_symbol_id = ct.callee_symbol_id
			WHERE ct.depth < ? AND cg.callee_symbol_id IS NOT NULL
		)
		SELECT id, caller_symbol_id, callee_symbol_id, callee_name,
		       call_line, call_column, call_type
		FROM callee_tree
		ORDER BY depth, call_line, call_column
	`, symbolID, depth)

	if err != nil {
		return nil, fmt.Errorf("failed to get callees recursively: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows: %w", cerr))
		}
	}()

	return scanCallGraphs(rows)
}

// DeleteCallGraphByCaller deletes all call graph entries for a caller symbol
func (db *DB) DeleteCallGraphByCaller(callerSymbolID int64) error {
	_, err := db.Exec("DELETE FROM call_graph WHERE caller_symbol_id = ?", callerSymbolID)
	if err != nil {
		return fmt.Errorf("failed to delete call graph: %w", err)
	}
	return nil
}

// scanCallGraphs is a helper function to scan multiple call graph rows
func scanCallGraphs(rows *sql.Rows) ([]*CallGraph, error) {
	var callGraphs []*CallGraph

	for rows.Next() {
		var cg CallGraph
		var callType sql.NullString

		err := rows.Scan(&cg.ID, &cg.CallerSymbolID, &cg.CalleeSymbolID, &cg.CalleeName,
			&cg.CallLine, &cg.CallColumn, &callType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan call graph: %w", err)
		}

		if callType.Valid {
			cg.CallType = callType.String
		}

		callGraphs = append(callGraphs, &cg)
	}

	return callGraphs, nil
}

// GetCallGraphCount returns the total number of call graph entries
func (db *DB) GetCallGraphCount() (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM call_graph").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get call graph count: %w", err)
	}
	return count, nil
}
