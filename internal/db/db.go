package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// Open opens a database connection and initializes the schema
func Open(dbPath string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{DB: sqlDB}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	// Optimize SQLite settings
	if err := db.optimize(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// initSchema initializes the database schema
func (db *DB) initSchema() error {
	_, err := db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}

// optimize sets optimal SQLite performance settings
func (db *DB) optimize() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-262144", // 256MB cache
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=30000000000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// OptimizeFTS5 optimizes the FTS5 index
func (db *DB) OptimizeFTS5() error {
	_, err := db.Exec("INSERT INTO symbols_fts(symbols_fts) VALUES('optimize')")
	if err != nil {
		return fmt.Errorf("failed to optimize FTS5: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
