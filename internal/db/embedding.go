package db

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// Embedding represents an embedding entry in the database
type Embedding struct {
	ID          int64
	SymbolID    *int64 // NULL for file-level embeddings
	FileID      int64
	Granularity string // 'file', 'class', 'function'
	Vector      []float32
	Dimension   int
	ContentHash string

	// Metadata for search results and filtering
	RepositoryID int64
	FilePath     string
	Language     string
	SymbolKind   string // NULL for file-level
	SymbolName   string // NULL for file-level
	Scope        string
	Snippet      string // Text excerpt for display
	StartLine    int
	EndLine      int

	CreatedAt time.Time
}

// vectorToBytes converts a float32 slice to bytes
func vectorToBytes(vector []float32) []byte {
	bytes := make([]byte, len(vector)*4) // 4 bytes per float32
	for i, v := range vector {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}
	return bytes
}

// bytesToVector converts bytes to a float32 slice
func bytesToVector(bytes []byte) []float32 {
	vector := make([]float32, len(bytes)/4)
	for i := range vector {
		bits := binary.LittleEndian.Uint32(bytes[i*4:])
		vector[i] = math.Float32frombits(bits)
	}
	return vector
}

// CreateEmbedding inserts a new embedding record with full metadata
func (db *DB) CreateEmbedding(emb *Embedding) (int64, error) {
	now := time.Now().Unix()
	vectorBytes := vectorToBytes(emb.Vector)

	result, err := db.Exec(`
		INSERT INTO embeddings (
			symbol_id, file_id, granularity, vector, dimension, content_hash,
			repository_id, file_path, language, symbol_kind, symbol_name, scope,
			snippet, start_line, end_line, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		emb.SymbolID, emb.FileID, emb.Granularity, vectorBytes, len(emb.Vector), emb.ContentHash,
		emb.RepositoryID, emb.FilePath, emb.Language, emb.SymbolKind, emb.SymbolName, emb.Scope,
		emb.Snippet, emb.StartLine, emb.EndLine, now)

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
	var vectorBytes []byte

	query := `
		SELECT id, symbol_id, file_id, granularity, vector, dimension, content_hash,
		       repository_id, file_path, language, symbol_kind, symbol_name, scope,
		       snippet, start_line, end_line, created_at
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
		&e.ID, &e.SymbolID, &e.FileID, &e.Granularity, &vectorBytes, &e.Dimension, &e.ContentHash,
		&e.RepositoryID, &e.FilePath, &e.Language, &e.SymbolKind, &e.SymbolName, &e.Scope,
		&e.Snippet, &e.StartLine, &e.EndLine, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	e.Vector = bytesToVector(vectorBytes)
	e.CreatedAt = time.Unix(createdAt, 0)
	return &e, nil
}

// GetEmbeddingsBySymbolIDs retrieves embeddings for multiple symbols
func (db *DB) GetEmbeddingsBySymbolIDs(symbolIDs []int64) ([]Embedding, error) {
	if len(symbolIDs) == 0 {
		return []Embedding{}, nil
	}

	query := `
		SELECT id, symbol_id, file_id, granularity, vector, dimension, content_hash,
		       repository_id, file_path, language, symbol_kind, symbol_name, scope,
		       snippet, start_line, end_line, created_at
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
		var vectorBytes []byte

		err := rows.Scan(
			&e.ID, &e.SymbolID, &e.FileID, &e.Granularity, &vectorBytes, &e.Dimension, &e.ContentHash,
			&e.RepositoryID, &e.FilePath, &e.Language, &e.SymbolKind, &e.SymbolName, &e.Scope,
			&e.Snippet, &e.StartLine, &e.EndLine, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}

		e.Vector = bytesToVector(vectorBytes)
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

// UpdateEmbedding updates an existing embedding
func (db *DB) UpdateEmbedding(id int64, vector []float32, contentHash string) error {
	vectorBytes := vectorToBytes(vector)
	_, err := db.Exec(`
		UPDATE embeddings
		SET vector = ?, dimension = ?, content_hash = ?
		WHERE id = ?
	`, vectorBytes, len(vector), contentHash, id)

	if err != nil {
		return fmt.Errorf("failed to update embedding: %w", err)
	}

	return nil
}

// GetAllEmbeddings retrieves all embeddings with optional granularity filter
func (db *DB) GetAllEmbeddings(granularity string) ([]Embedding, error) {
	query := `
		SELECT id, symbol_id, file_id, granularity, vector, dimension, content_hash,
		       repository_id, file_path, language, symbol_kind, symbol_name, scope,
		       snippet, start_line, end_line, created_at
		FROM embeddings
	`
	args := []interface{}{}

	if granularity != "" {
		query += " WHERE granularity = ?"
		args = append(args, granularity)
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
		var vectorBytes []byte

		err := rows.Scan(
			&e.ID, &e.SymbolID, &e.FileID, &e.Granularity, &vectorBytes, &e.Dimension, &e.ContentHash,
			&e.RepositoryID, &e.FilePath, &e.Language, &e.SymbolKind, &e.SymbolName, &e.Scope,
			&e.Snippet, &e.StartLine, &e.EndLine, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}

		e.Vector = bytesToVector(vectorBytes)
		e.CreatedAt = time.Unix(createdAt, 0)
		embeddings = append(embeddings, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embeddings: %w", err)
	}

	return embeddings, nil
}

// EmbeddingExists checks if an embedding already exists for a symbol with given granularity and content hash
func (db *DB) EmbeddingExists(symbolID int64, granularity string, contentHash string) (bool, error) {
	var count int64
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM embeddings
		WHERE symbol_id = ? AND granularity = ? AND content_hash = ?
	`, symbolID, granularity, contentHash).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check embedding existence: %w", err)
	}

	return count > 0, nil
}

// FileEmbeddingExists checks if a file-level embedding already exists
func (db *DB) FileEmbeddingExists(fileID int64, contentHash string) (bool, error) {
	var count int64
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM embeddings
		WHERE file_id = ? AND granularity = 'file' AND content_hash = ?
	`, fileID, contentHash).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check file embedding existence: %w", err)
	}

	return count > 0, nil
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
