package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Embedding represents an embedding entry in the database
type Embedding struct {
	ID            int64
	SymbolID      int64
	Granularity   string
	QdrantPointID string
	ContentHash   string
	CreatedAt     time.Time
}

// CreateEmbedding inserts a new embedding record
func (db *DB) CreateEmbedding(symbolID int64, granularity, qdrantPointID, contentHash string) (int64, error) {
	now := time.Now().Unix()

	result, err := db.Exec(`
		INSERT INTO embeddings (symbol_id, granularity, qdrant_point_id, content_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, symbolID, granularity, qdrantPointID, contentHash, now)

	if err != nil {
		return 0, fmt.Errorf("failed to create embedding: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// GetEmbeddingBySymbolID retrieves embeddings for a symbol
func (db *DB) GetEmbeddingBySymbolID(symbolID int64, granularity string) (*Embedding, error) {
	var e Embedding
	var createdAt int64

	query := `
		SELECT id, symbol_id, granularity, qdrant_point_id, content_hash, created_at
		FROM embeddings
		WHERE symbol_id = ?
	`
	args := []interface{}{symbolID}

	if granularity != "" {
		query += " AND granularity = ?"
		args = append(args, granularity)
	}

	query += " LIMIT 1"

	err := db.QueryRow(query, args...).Scan(
		&e.ID,
		&e.SymbolID,
		&e.Granularity,
		&e.QdrantPointID,
		&e.ContentHash,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	e.CreatedAt = time.Unix(createdAt, 0)
	return &e, nil
}

// GetEmbeddingsBySymbolIDs retrieves embeddings for multiple symbols
func (db *DB) GetEmbeddingsBySymbolIDs(symbolIDs []int64) ([]Embedding, error) {
	if len(symbolIDs) == 0 {
		return []Embedding{}, nil
	}

	query := `
		SELECT id, symbol_id, granularity, qdrant_point_id, content_hash, created_at
		FROM embeddings
		WHERE symbol_id IN (` + placeholders(len(symbolIDs)) + `)
	`

	args := make([]interface{}, len(symbolIDs))
	for i, id := range symbolIDs {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	embeddings := []Embedding{}
	for rows.Next() {
		var e Embedding
		var createdAt int64

		err := rows.Scan(
			&e.ID,
			&e.SymbolID,
			&e.Granularity,
			&e.QdrantPointID,
			&e.ContentHash,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}

		e.CreatedAt = time.Unix(createdAt, 0)
		embeddings = append(embeddings, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embeddings: %w", err)
	}

	return embeddings, nil
}

// DeleteEmbedding deletes an embedding by ID
func (db *DB) DeleteEmbedding(id int64) error {
	_, err := db.Exec("DELETE FROM embeddings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}
	return nil
}

// DeleteEmbeddingsBySymbolID deletes all embeddings for a symbol
func (db *DB) DeleteEmbeddingsBySymbolID(symbolID int64) error {
	_, err := db.Exec("DELETE FROM embeddings WHERE symbol_id = ?", symbolID)
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}
	return nil
}

// GetEmbeddingByQdrantID retrieves an embedding by Qdrant point ID
func (db *DB) GetEmbeddingByQdrantID(qdrantPointID string) (*Embedding, error) {
	var e Embedding
	var createdAt int64

	err := db.QueryRow(`
		SELECT id, symbol_id, granularity, qdrant_point_id, content_hash, created_at
		FROM embeddings
		WHERE qdrant_point_id = ?
	`, qdrantPointID).Scan(
		&e.ID,
		&e.SymbolID,
		&e.Granularity,
		&e.QdrantPointID,
		&e.ContentHash,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	e.CreatedAt = time.Unix(createdAt, 0)
	return &e, nil
}

// UpdateEmbedding updates an existing embedding
func (db *DB) UpdateEmbedding(id int64, qdrantPointID, contentHash string) error {
	_, err := db.Exec(`
		UPDATE embeddings
		SET qdrant_point_id = ?, content_hash = ?
		WHERE id = ?
	`, qdrantPointID, contentHash, id)

	if err != nil {
		return fmt.Errorf("failed to update embedding: %w", err)
	}

	return nil
}

// Helper function to generate SQL placeholders
func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ", ?"
	}
	return s
}
