package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository represents a code repository
type Repository struct {
	ID            int64
	Name          string
	Path          string
	LastIndexedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateRepository creates a new repository record
func (db *DB) CreateRepository(name, path string) (*Repository, error) {
	now := time.Now().Unix()

	result, err := db.Exec(`
		INSERT INTO repositories (name, path, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, name, path, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository ID: %w", err)
	}

	return db.GetRepository(id)
}

// GetRepository retrieves a repository by ID
func (db *DB) GetRepository(id int64) (*Repository, error) {
	var repo Repository
	var lastIndexedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, name, path, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE id = ?
	`, id).Scan(&repo.ID, &repo.Name, &repo.Path, &lastIndexedAt, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if lastIndexedAt.Valid {
		t := time.Unix(lastIndexedAt.Int64, 0)
		repo.LastIndexedAt = &t
	}

	repo.CreatedAt = time.Unix(createdAt, 0)
	repo.UpdatedAt = time.Unix(updatedAt, 0)

	return &repo, nil
}

// GetRepositoryByName retrieves a repository by name
func (db *DB) GetRepositoryByName(name string) (*Repository, error) {
	var repo Repository
	var lastIndexedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, name, path, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE name = ?
	`, name).Scan(&repo.ID, &repo.Name, &repo.Path, &lastIndexedAt, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if lastIndexedAt.Valid {
		t := time.Unix(lastIndexedAt.Int64, 0)
		repo.LastIndexedAt = &t
	}

	repo.CreatedAt = time.Unix(createdAt, 0)
	repo.UpdatedAt = time.Unix(updatedAt, 0)

	return &repo, nil
}

// GetRepositoryByPath retrieves a repository by path
func (db *DB) GetRepositoryByPath(path string) (*Repository, error) {
	var repo Repository
	var lastIndexedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, name, path, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE path = ?
	`, path).Scan(&repo.ID, &repo.Name, &repo.Path, &lastIndexedAt, &createdAt, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if lastIndexedAt.Valid {
		t := time.Unix(lastIndexedAt.Int64, 0)
		repo.LastIndexedAt = &t
	}

	repo.CreatedAt = time.Unix(createdAt, 0)
	repo.UpdatedAt = time.Unix(updatedAt, 0)

	return &repo, nil
}

// ListRepositories retrieves all repositories
func (db *DB) ListRepositories() ([]*Repository, error) {
	rows, err := db.Query(`
		SELECT id, name, path, last_indexed_at, created_at, updated_at
		FROM repositories
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var repos []*Repository
	for rows.Next() {
		var repo Repository
		var lastIndexedAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(&repo.ID, &repo.Name, &repo.Path, &lastIndexedAt, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}

		if lastIndexedAt.Valid {
			t := time.Unix(lastIndexedAt.Int64, 0)
			repo.LastIndexedAt = &t
		}

		repo.CreatedAt = time.Unix(createdAt, 0)
		repo.UpdatedAt = time.Unix(updatedAt, 0)

		repos = append(repos, &repo)
	}

	return repos, nil
}

// UpdateRepositoryIndexedAt updates the last_indexed_at timestamp
func (db *DB) UpdateRepositoryIndexedAt(id int64) error {
	now := time.Now().Unix()

	_, err := db.Exec(`
		UPDATE repositories
		SET last_indexed_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, id)

	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	return nil
}

// DeleteRepository deletes a repository and all associated data
func (db *DB) DeleteRepository(id int64) error {
	_, err := db.Exec("DELETE FROM repositories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	return nil
}

// RepositoryExists checks if a repository exists by name or path
func (db *DB) RepositoryExists(name, path string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM repositories
		WHERE name = ? OR path = ?
	`, name, path).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check repository existence: %w", err)
	}

	return count > 0, nil
}
