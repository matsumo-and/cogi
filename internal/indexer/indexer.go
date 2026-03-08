package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
	"github.com/matsumo_and/cogi/internal/parser"
)

// Indexer handles code indexing operations
type Indexer struct {
	db          *db.DB
	config      *config.Config
	embedClient embedding.Client
}

// New creates a new Indexer
func New(database *db.DB, cfg *config.Config) *Indexer {
	// Initialize embedding client
	embedClient := embedding.NewOllamaClient(
		cfg.Embedding.Endpoint,
		cfg.Embedding.Model,
		cfg.Embedding.Dimension,
	)

	return &Indexer{
		db:          database,
		config:      cfg,
		embedClient: embedClient,
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

	// Update repository last_indexed_at
	if err := idx.db.UpdateRepositoryIndexedAt(repoID); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Optimize FTS5 index
	if err := idx.db.OptimizeFTS5(); err != nil {
		fmt.Printf("Warning: failed to optimize FTS5: %v\n", err)
	}

	// Generate embeddings for indexed symbols
	fmt.Println("\nGenerating embeddings...")
	if err := idx.generateEmbeddings(ctx, repoID); err != nil {
		fmt.Printf("Warning: failed to generate embeddings: %v\n", err)
		fmt.Println("Note: Make sure Ollama is running and the model is available.")
	}

	// Return error if there were any indexing errors
	if len(errors) > 0 {
		return fmt.Errorf("indexing completed with %d errors", len(errors))
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

		// File has changed, delete old data
		if err := idx.db.DeleteSymbolsByFile(existingFile.ID); err != nil {
			return fmt.Errorf("failed to delete old symbols: %w", err)
		}
		if err := idx.db.DeleteImportGraphByFile(existingFile.ID); err != nil {
			return fmt.Errorf("failed to delete old import graph: %w", err)
		}
		// Note: Call graph is deleted via CASCADE when symbols are deleted

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

	parseResult, err := p.ParseFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Insert symbols into database and build symbol name to ID mapping
	symbolNameToID := make(map[string]int64)
	for _, sym := range parseResult.Symbols {
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

		symbolID, err := idx.db.CreateSymbol(dbSymbol)
		if err != nil {
			return fmt.Errorf("failed to create symbol: %w", err)
		}

		// Build mapping for call graph resolution
		fullName := sym.Name
		if sym.Scope != "" {
			fullName = sym.Scope + "." + sym.Name
		}
		symbolNameToID[fullName] = symbolID
		symbolNameToID[sym.Name] = symbolID // Also map just the name
	}

	// Insert import graph entries
	if len(parseResult.Imports) > 0 {
		var importGraphs []*db.ImportGraph
		for _, imp := range parseResult.Imports {
			importedSymbolsJSON := ""
			if len(imp.ImportedSymbols) > 0 {
				// Simple JSON array encoding
				importedSymbolsJSON = `["` + strings.Join(imp.ImportedSymbols, `","`) + `"]`
			}

			importGraphs = append(importGraphs, &db.ImportGraph{
				FileID:          fileID,
				ImportPath:      imp.ImportPath,
				ImportType:      imp.ImportType,
				ImportedSymbols: importedSymbolsJSON,
				LineNumber:      imp.LineNumber,
			})
		}

		if err := idx.db.BatchCreateImportGraph(importGraphs); err != nil {
			return fmt.Errorf("failed to create import graph: %w", err)
		}
	}

	// Insert call graph entries
	if len(parseResult.CallSites) > 0 {
		var callGraphs []*db.CallGraph
		for _, call := range parseResult.CallSites {
			// Try to resolve caller symbol ID
			callerID, ok := symbolNameToID[call.CallerName]
			if !ok {
				// Try without scope
				parts := strings.Split(call.CallerName, ".")
				if len(parts) > 0 {
					callerID, ok = symbolNameToID[parts[len(parts)-1]]
				}
			}

			if !ok {
				// Caller not found in current file, skip
				continue
			}

			// Try to resolve callee symbol ID (might be in another file)
			calleeID, calleeFound := symbolNameToID[call.CalleeName]

			// Create call graph entry
			cg := &db.CallGraph{
				CallerSymbolID: callerID,
				CalleeName:     call.CalleeName,
				CallLine:       call.Line,
				CallColumn:     call.Column,
				CallType:       call.CallType,
			}

			if calleeFound {
				cg.CalleeSymbolID.Valid = true
				cg.CalleeSymbolID.Int64 = calleeID
			}

			callGraphs = append(callGraphs, cg)
		}

		if len(callGraphs) > 0 {
			if err := idx.db.BatchCreateCallGraph(callGraphs); err != nil {
				return fmt.Errorf("failed to create call graph: %w", err)
			}
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

// generateEmbeddings generates embeddings for all symbols in a repository
func (idx *Indexer) generateEmbeddings(ctx context.Context, repoID int64) error {
	// Get all symbols for this repository
	symbols, err := idx.db.GetSymbolsByRepository(repoID)
	if err != nil {
		return fmt.Errorf("failed to get symbols: %w", err)
	}

	if len(symbols) == 0 {
		fmt.Println("No symbols to generate embeddings for")
		return nil
	}

	fmt.Printf("Generating embeddings for %d symbols...\n", len(symbols))

	// Prepare texts for embedding generation
	type embeddingTask struct {
		symbolID    int64
		granularity string
		text        string
	}

	var tasks []embeddingTask

	for _, sym := range symbols {
		// Generate class-level embedding for classes/structs/interfaces
		if sym.Kind == "class" || sym.Kind == "struct" || sym.Kind == "interface" {
			classText := buildClassLevelText(sym)
			contentHash := computeHash(classText)

			// Check if embedding already exists
			exists, err := idx.db.EmbeddingExists(sym.ID, "class", contentHash)
			if err != nil {
				return fmt.Errorf("failed to check embedding existence: %w", err)
			}

			if !exists {
				tasks = append(tasks, embeddingTask{
					symbolID:    sym.ID,
					granularity: "class",
					text:        classText,
				})
			}
		}

		// Generate function-level embedding for functions/methods
		if sym.Kind == "function" || sym.Kind == "method" {
			functionText := buildFunctionLevelText(sym)
			contentHash := computeHash(functionText)

			// Check if embedding already exists
			exists, err := idx.db.EmbeddingExists(sym.ID, "function", contentHash)
			if err != nil {
				return fmt.Errorf("failed to check embedding existence: %w", err)
			}

			if !exists {
				tasks = append(tasks, embeddingTask{
					symbolID:    sym.ID,
					granularity: "function",
					text:        functionText,
				})
			}
		}
	}

	if len(tasks) == 0 {
		fmt.Println("All embeddings are up to date")
		return nil
	}

	fmt.Printf("Generating %d new embeddings...\n", len(tasks))

	// Extract texts for batch embedding
	texts := make([]string, len(tasks))
	for i, task := range tasks {
		texts[i] = task.text
	}

	// Generate embeddings in batches
	batchSize := idx.config.Embedding.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	embeddings, err := embedding.EmbedBatch(ctx, idx.embedClient, texts, batchSize)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Store embeddings in database
	for i, task := range tasks {
		contentHash := computeHash(task.text)
		_, err := idx.db.CreateEmbedding(task.symbolID, task.granularity, embeddings[i].Vector, contentHash)
		if err != nil {
			return fmt.Errorf("failed to store embedding: %w", err)
		}
	}

	fmt.Printf("✓ Generated %d embeddings successfully\n", len(tasks))
	return nil
}

// buildClassLevelText creates text representation for class-level embedding
func buildClassLevelText(sym *db.Symbol) string {
	var parts []string

	// Add docstring if available
	if sym.Docstring != "" {
		parts = append(parts, sym.Docstring)
	}

	// Add signature
	if sym.Signature != "" {
		parts = append(parts, sym.Signature)
	} else {
		parts = append(parts, fmt.Sprintf("%s %s", sym.Kind, sym.Name))
	}

	// Add code body (truncated if too long)
	if sym.CodeBody != "" {
		body := sym.CodeBody
		if len(body) > 2000 {
			body = body[:2000] + "..."
		}
		parts = append(parts, body)
	}

	return strings.Join(parts, "\n\n")
}

// buildFunctionLevelText creates text representation for function-level embedding
func buildFunctionLevelText(sym *db.Symbol) string {
	var parts []string

	// Add docstring if available
	if sym.Docstring != "" {
		parts = append(parts, sym.Docstring)
	}

	// Add signature
	if sym.Signature != "" {
		parts = append(parts, sym.Signature)
	} else {
		parts = append(parts, fmt.Sprintf("%s %s", sym.Kind, sym.Name))
	}

	// Add code body
	if sym.CodeBody != "" {
		parts = append(parts, sym.CodeBody)
	}

	return strings.Join(parts, "\n\n")
}

// computeHash computes SHA256 hash of a string
func computeHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}
