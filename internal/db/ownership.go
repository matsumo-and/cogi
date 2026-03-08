package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Ownership represents code ownership information
type Ownership struct {
	ID             int64
	FileID         int64
	StartLine      int
	EndLine        int
	AuthorName     string
	AuthorEmail    string
	LastCommitHash string
	LastCommitDate time.Time
	CommitCount    int
}

// CreateOwnership creates a new ownership record
func (db *DB) CreateOwnership(ownership *Ownership) error {
	_, err := db.Exec(`
		INSERT INTO ownership (
			file_id, start_line, end_line, author_name, author_email,
			last_commit_hash, last_commit_date, commit_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ownership.FileID, ownership.StartLine, ownership.EndLine,
		ownership.AuthorName, ownership.AuthorEmail,
		ownership.LastCommitHash, ownership.LastCommitDate.Unix(), ownership.CommitCount)

	if err != nil {
		return fmt.Errorf("failed to create ownership: %w", err)
	}

	return nil
}

// GetOwnershipByFile retrieves all ownership records for a file
func (db *DB) GetOwnershipByFile(fileID int64) ([]*Ownership, error) {
	rows, err := db.Query(`
		SELECT id, file_id, start_line, end_line, author_name, author_email,
		       last_commit_hash, last_commit_date, commit_count
		FROM ownership
		WHERE file_id = ?
		ORDER BY start_line
	`, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ownership: %w", err)
	}
	defer rows.Close()

	return scanOwnership(rows)
}

// GetOwnershipByAuthor retrieves all ownership records for an author
func (db *DB) GetOwnershipByAuthor(authorName string) ([]*Ownership, error) {
	rows, err := db.Query(`
		SELECT id, file_id, start_line, end_line, author_name, author_email,
		       last_commit_hash, last_commit_date, commit_count
		FROM ownership
		WHERE author_name = ?
		ORDER BY last_commit_date DESC
	`, authorName)
	if err != nil {
		return nil, fmt.Errorf("failed to query ownership: %w", err)
	}
	defer rows.Close()

	return scanOwnership(rows)
}

// GetOwnershipByLine retrieves ownership for a specific line in a file
func (db *DB) GetOwnershipByLine(fileID int64, lineNumber int) (*Ownership, error) {
	var o Ownership
	var commitDate int64

	err := db.QueryRow(`
		SELECT id, file_id, start_line, end_line, author_name, author_email,
		       last_commit_hash, last_commit_date, commit_count
		FROM ownership
		WHERE file_id = ? AND start_line <= ? AND end_line >= ?
		LIMIT 1
	`, fileID, lineNumber, lineNumber).Scan(
		&o.ID, &o.FileID, &o.StartLine, &o.EndLine,
		&o.AuthorName, &o.AuthorEmail,
		&o.LastCommitHash, &commitDate, &o.CommitCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ownership: %w", err)
	}

	o.LastCommitDate = time.Unix(commitDate, 0)
	return &o, nil
}

// DeleteOwnershipByFile deletes all ownership records for a file
func (db *DB) DeleteOwnershipByFile(fileID int64) error {
	_, err := db.Exec("DELETE FROM ownership WHERE file_id = ?", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete ownership: %w", err)
	}
	return nil
}

// GetTopAuthors retrieves the top contributors across all files
func (db *DB) GetTopAuthors(limit int) ([]AuthorStats, error) {
	rows, err := db.Query(`
		SELECT author_name, author_email,
		       COUNT(DISTINCT file_id) as file_count,
		       SUM(commit_count) as total_commits,
		       MAX(last_commit_date) as last_commit
		FROM ownership
		GROUP BY author_name, author_email
		ORDER BY total_commits DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query authors: %w", err)
	}
	defer rows.Close()

	var authors []AuthorStats
	for rows.Next() {
		var a AuthorStats
		var lastCommit int64

		err := rows.Scan(&a.AuthorName, &a.AuthorEmail, &a.FileCount, &a.TotalCommits, &lastCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan author: %w", err)
		}

		a.LastCommit = time.Unix(lastCommit, 0)
		authors = append(authors, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating authors: %w", err)
	}

	return authors, nil
}

// AuthorStats represents statistics for an author
type AuthorStats struct {
	AuthorName   string
	AuthorEmail  string
	FileCount    int
	TotalCommits int
	LastCommit   time.Time
}

// scanOwnership is a helper function to scan ownership rows
func scanOwnership(rows *sql.Rows) ([]*Ownership, error) {
	var ownerships []*Ownership

	for rows.Next() {
		var o Ownership
		var commitDate int64

		err := rows.Scan(
			&o.ID, &o.FileID, &o.StartLine, &o.EndLine,
			&o.AuthorName, &o.AuthorEmail,
			&o.LastCommitHash, &commitDate, &o.CommitCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ownership: %w", err)
		}

		o.LastCommitDate = time.Unix(commitDate, 0)
		ownerships = append(ownerships, &o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ownership: %w", err)
	}

	return ownerships, nil
}
