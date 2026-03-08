package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/parser"
)

// Indexer handles code indexing operations
type Indexer struct {
	db     *db.DB
	config *config.Config
}

// New creates a new Indexer
func New(database *db.DB, cfg *config.Config) *Indexer {
	return &Indexer{
		db:     database,
		config: cfg,
	}
}

// IndexRepository indexes a repository
func (idx *Indexer) IndexRepository(ctx context.Context, repoID int64, repoPath string) error {
	fmt.Printf("Indexing repository: %s\n", repoPath)

	// Walk the repository and collect files to index
	files, err := idx.collectFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	fmt.Printf("Found %d files to index\n", len(files))

	// Index files in parallel
	return idx.indexFiles(ctx, repoID, repoPath, files)
}

// collectFiles collects all indexable files in the repository
func (idx *Indexer) collectFiles(repoPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be excluded
			if idx.shouldExclude(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files that are too large
		if info.Size() > int64(idx.config.Indexing.MaxFileSizeMB*1024*1024) {
			return nil
		}

		// Skip excluded files
		if idx.shouldExclude(path) {
			return nil
		}

		// Check if file is a supported language
		lang := parser.DetectLanguage(path)
		if lang == parser.LangUnknown {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// shouldExclude checks if a path should be excluded
func (idx *Indexer) shouldExclude(path string) bool {
	for _, pattern := range idx.config.Indexing.ExcludePatterns {
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return true
		}

		// Also check if the pattern matches any part of the path
		if strings.Contains(path, strings.Trim(pattern, "*")) {
			return true
		}
	}
	return false
}

// indexFiles indexes multiple files in parallel
func (idx *Indexer) indexFiles(ctx context.Context, repoID int64, repoPath string, files []string) error {
	maxWorkers := idx.config.Performance.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	// Create worker pool
	jobs := make(chan string, len(files))
	results := make(chan error, len(files))

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				select {
				case <-ctx.Done():
					results <- ctx.Err()
					return
				default:
					err := idx.indexFile(ctx, repoID, repoPath, filePath)
					results <- err
				}
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect errors
	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("indexing completed with %d errors", len(errors))
	}

	// Update repository last_indexed_at
	if err := idx.db.UpdateRepositoryIndexedAt(repoID); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Optimize FTS5 index
	if err := idx.db.OptimizeFTS5(); err != nil {
		fmt.Printf("Warning: failed to optimize FTS5: %v\n", err)
	}

	return nil
}

// indexFile indexes a single file
func (idx *Indexer) indexFile(ctx context.Context, repoID int64, repoPath, filePath string) error {
	// Get relative path
	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Detect language
	lang := parser.DetectLanguage(filePath)
	if lang == parser.LangUnknown {
		return nil
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Compute file hash
	fileHash, err := db.ComputeFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to compute file hash: %w", err)
	}

	// Check if file already exists in DB
	existingFile, err := idx.db.GetFileByPath(repoID, relPath)
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	var fileID int64

	if existingFile != nil {
		// Check if file has changed
		if existingFile.FileHash == fileHash {
			// File hasn't changed, skip
			return nil
		}

		// File has changed, delete old symbols
		if err := idx.db.DeleteSymbolsByFile(existingFile.ID); err != nil {
			return fmt.Errorf("failed to delete old symbols: %w", err)
		}

		// Update file record
		if err := idx.db.UpdateFile(existingFile.ID, fileHash, fileInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}

		fileID = existingFile.ID
	} else {
		// Create new file record
		file, err := idx.db.CreateFile(repoID, relPath, string(lang), fileHash, fileInfo.ModTime())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		fileID = file.ID
	}

	// Parse the file
	p, err := parser.New(lang)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	symbols, err := p.ParseFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Insert symbols into database
	for _, sym := range symbols {
		dbSymbol := &db.Symbol{
			FileID:      fileID,
			Name:        sym.Name,
			Kind:        sym.Kind,
			StartLine:   sym.StartLine,
			StartColumn: sym.StartColumn,
			EndLine:     sym.EndLine,
			EndColumn:   sym.EndColumn,
			Scope:       sym.Scope,
			Visibility:  sym.Visibility,
			Docstring:   sym.Docstring,
			Signature:   sym.Signature,
			CodeBody:    sym.CodeBody,
		}

		_, err := idx.db.CreateSymbol(dbSymbol)
		if err != nil {
			return fmt.Errorf("failed to create symbol: %w", err)
		}
	}

	return nil
}

// GetIndexingStats returns statistics about the indexing progress
type IndexingStats struct {
	TotalFiles   int
	TotalSymbols int64
	LastIndexed  *time.Time
}

// GetStats returns indexing statistics for a repository
func (idx *Indexer) GetStats(repoID int64) (*IndexingStats, error) {
	repo, err := idx.db.GetRepository(repoID)
	if err != nil {
		return nil, err
	}

	files, err := idx.db.ListFilesByRepository(repoID)
	if err != nil {
		return nil, err
	}

	symbolCount, err := idx.db.GetSymbolCountByRepository(repoID)
	if err != nil {
		return nil, err
	}

	return &IndexingStats{
		TotalFiles:   len(files),
		TotalSymbols: symbolCount,
		LastIndexed:  repo.LastIndexedAt,
	}, nil
}
