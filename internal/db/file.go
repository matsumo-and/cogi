package db

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// File represents a file in the codebase
type File struct {
	ID           int64
	RepositoryID int64
	Path         string
	Language     string
	LastModified time.Time
	FileHash     string
	IndexedAt    time.Time
}

// CreateFile creates a new file record
func (db *DB) CreateFile(repositoryID int64, path, language, fileHash string, lastModified time.Time) (*File, error) {
	now := time.Now().Unix()

	result, err := db.Exec(`
		INSERT INTO files (repository_id, path, language, last_modified, file_hash, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, repositoryID, path, language, lastModified.Unix(), fileHash, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get file ID: %w", err)
	}

	return db.GetFile(id)
}

// GetFile retrieves a file by ID
func (db *DB) GetFile(id int64) (*File, error) {
	var file File
	var lastModified, indexedAt int64

	err := db.QueryRow(`
		SELECT id, repository_id, path, language, last_modified, file_hash, indexed_at
		FROM files
		WHERE id = ?
	`, id).Scan(&file.ID, &file.RepositoryID, &file.Path, &file.Language, &lastModified, &file.FileHash, &indexedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	file.LastModified = time.Unix(lastModified, 0)
	file.IndexedAt = time.Unix(indexedAt, 0)

	return &file, nil
}

// GetFileByPath retrieves a file by repository ID and path
func (db *DB) GetFileByPath(repositoryID int64, path string) (*File, error) {
	var file File
	var lastModified, indexedAt int64

	err := db.QueryRow(`
		SELECT id, repository_id, path, language, last_modified, file_hash, indexed_at
		FROM files
		WHERE repository_id = ? AND path = ?
	`, repositoryID, path).Scan(&file.ID, &file.RepositoryID, &file.Path, &file.Language, &lastModified, &file.FileHash, &indexedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	file.LastModified = time.Unix(lastModified, 0)
	file.IndexedAt = time.Unix(indexedAt, 0)

	return &file, nil
}

// UpdateFile updates a file record
func (db *DB) UpdateFile(id int64, fileHash string, lastModified time.Time) error {
	now := time.Now().Unix()

	_, err := db.Exec(`
		UPDATE files
		SET file_hash = ?, last_modified = ?, indexed_at = ?
		WHERE id = ?
	`, fileHash, lastModified.Unix(), now, id)

	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	return nil
}

// DeleteFile deletes a file and all associated data
func (db *DB) DeleteFile(id int64) error {
	_, err := db.Exec("DELETE FROM files WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// ListFilesByRepository retrieves all files for a repository
func (db *DB) ListFilesByRepository(repositoryID int64) ([]*File, error) {
	rows, err := db.Query(`
		SELECT id, repository_id, path, language, last_modified, file_hash, indexed_at
		FROM files
		WHERE repository_id = ?
		ORDER BY path
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanFiles(rows)
}

// ListFiles retrieves all files across all repositories
func (db *DB) ListFiles() ([]*File, error) {
	rows, err := db.Query(`
		SELECT id, repository_id, path, language, last_modified, file_hash, indexed_at
		FROM files
		ORDER BY path
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanFiles(rows)
}

// scanFiles is a helper function to scan file rows
func scanFiles(rows *sql.Rows) ([]*File, error) {
	var files []*File
	for rows.Next() {
		var file File
		var lastModified, indexedAt int64

		err := rows.Scan(&file.ID, &file.RepositoryID, &file.Path, &file.Language, &lastModified, &file.FileHash, &indexedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		file.LastModified = time.Unix(lastModified, 0)
		file.IndexedAt = time.Unix(indexedAt, 0)

		files = append(files, &file)
	}

	return files, nil
}

// ComputeFileHash computes the SHA256 hash of a file
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file: %w", cerr))
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
