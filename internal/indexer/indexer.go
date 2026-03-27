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
	"github.com/matsumo_and/cogi/internal/ownership"
	"github.com/matsumo_and/cogi/internal/parser"
)

// Indexer handles code indexing operations
type Indexer struct {
	db                *db.DB
	config            *config.Config
	embedClient       embedding.Client
	ownershipAnalyzer *ownership.Analyzer
}

// New creates a new Indexer
func New(database *db.DB, cfg *config.Config) *Indexer {
	// Initialize embedding client
	embedClient := embedding.NewOllamaClient(
		cfg.Embedding.Endpoint,
		cfg.Embedding.Model,
		cfg.Embedding.Dimension,
	)

	// Initialize ownership analyzer
	ownershipAnalyzer := ownership.New(database)

	return &Indexer{
		db:                database,
		config:            cfg,
		embedClient:       embedClient,
		ownershipAnalyzer: ownershipAnalyzer,
	}
}

// IndexRepository performs a full index of a repository
func (idx *Indexer) IndexRepository(ctx context.Context, repoID int64, repoPath string, fullIndex bool) error {
	fmt.Printf("Indexing repository: %s\n", repoPath)

	// Walk the repository and collect files to index
	files, err := idx.collectFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	fmt.Printf("Found %d files to index\n", len(files))

	// Clean up deleted files if not doing full index
	if !fullIndex {
		if err := idx.cleanupDeletedFiles(ctx, repoID, repoPath, files); err != nil {
			fmt.Printf("Warning: failed to cleanup deleted files: %v\n", err)
		}
	}

	// Index files in parallel
	return idx.indexFiles(ctx, repoID, repoPath, files, fullIndex)
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

// cleanupDeletedFiles removes database entries for files that no longer exist
func (idx *Indexer) cleanupDeletedFiles(ctx context.Context, repoID int64, repoPath string, existingFiles []string) error {
	// Create a map of existing files for quick lookup
	existingFilesMap := make(map[string]bool)
	for _, file := range existingFiles {
		relPath, err := filepath.Rel(repoPath, file)
		if err != nil {
			continue
		}
		existingFilesMap[relPath] = true
	}

	// Get all files from database
	dbFiles, err := idx.db.ListFilesByRepository(repoID)
	if err != nil {
		return fmt.Errorf("failed to list files from database: %w", err)
	}

	// Find and delete files that no longer exist
	var deletedCount int
	for _, dbFile := range dbFiles {
		if !existingFilesMap[dbFile.Path] {
			// File no longer exists, delete it
			if err := idx.db.DeleteFile(dbFile.ID); err != nil {
				fmt.Printf("Warning: failed to delete file %s: %v\n", dbFile.Path, err)
				continue
			}
			deletedCount++
		}
	}

	if deletedCount > 0 {
		fmt.Printf("Cleaned up %d deleted files\n", deletedCount)
	}

	return nil
}

// indexFiles indexes multiple files in parallel
func (idx *Indexer) indexFiles(ctx context.Context, repoID int64, repoPath string, files []string, fullIndex bool) error {
	maxWorkers := idx.config.Performance.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	// Progress tracking
	progress := NewProgressBar(int64(len(files)), "Indexing files")

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
					err := idx.indexFile(ctx, repoID, repoPath, filePath, fullIndex)
					results <- err
					progress.Increment()
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

	progress.Finish()

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

	// Analyze ownership for indexed files
	fmt.Println("\nAnalyzing code ownership...")
	if err := idx.analyzeOwnership(ctx, repoID, repoPath); err != nil {
		fmt.Printf("Warning: failed to analyze ownership: %v\n", err)
		fmt.Println("Note: Make sure you're in a git repository.")
	}

	// Return error if there were any indexing errors
	if len(errors) > 0 {
		return fmt.Errorf("indexing completed with %d errors", len(errors))
	}

	return nil
}

// indexFile indexes a single file
func (idx *Indexer) indexFile(ctx context.Context, repoID int64, repoPath, filePath string, fullIndex bool) error {
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
		// Check if file has changed (hash-based check)
		if existingFile.FileHash == fileHash {
			// File hasn't changed, skip
			return nil
		}

		// If not full index, also check timestamp
		if !fullIndex {
			// Check if file was modified after last indexing
			if !fileInfo.ModTime().After(existingFile.IndexedAt) {
				// File hasn't been modified, skip
				return nil
			}
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
	// Check for dimension compatibility with existing embeddings
	existingEmbeddings, err := idx.db.GetAllEmbeddings("")
	if err != nil {
		return fmt.Errorf("failed to get existing embeddings: %w", err)
	}

	if len(existingEmbeddings) > 0 {
		// Check if any embedding has different dimension
		expectedDim := idx.config.Embedding.Dimension
		for _, emb := range existingEmbeddings {
			if emb.Dimension != expectedDim {
				fmt.Printf("\n⚠️  WARNING: Dimension mismatch detected!\n")
				fmt.Printf("   Existing embeddings: %d dimensions\n", emb.Dimension)
				fmt.Printf("   Current model: %d dimensions\n", expectedDim)
				fmt.Printf("   Model: %s\n\n", idx.config.Embedding.Model)
				fmt.Printf("   Existing embeddings will be automatically regenerated with new dimensions.\n")
				fmt.Printf("   This may take some time depending on the codebase size.\n\n")
				break
			}
		}
	}

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

	// Group symbols by file for file-level embeddings
	fileSymbols := make(map[int64][]*db.Symbol)
	for _, sym := range symbols {
		fileSymbols[sym.FileID] = append(fileSymbols[sym.FileID], sym)
	}

	// Pre-fetch file information and imports for all files
	fileInfoMap := make(map[int64]*db.File)
	fileImportsMap := make(map[int64][]*db.ImportGraph)
	for fileID := range fileSymbols {
		file, err := idx.db.GetFile(fileID)
		if err != nil {
			fmt.Printf("Warning: failed to get file %d: %v\n", fileID, err)
			continue
		}
		fileInfoMap[fileID] = file

		imports, err := idx.db.GetImportGraphByFile(fileID)
		if err != nil {
			fmt.Printf("Warning: failed to get imports for file %d: %v\n", fileID, err)
			imports = []*db.ImportGraph{} // Use empty slice on error
		}
		fileImportsMap[fileID] = imports
	}

	// Prepare texts for embedding generation
	type embeddingTask struct {
		symbol      *db.Symbol // nil for file-level embeddings
		fileID      int64
		granularity string
		text        string
	}

	var tasks []embeddingTask

	// Generate file-level embeddings
	for fileID, syms := range fileSymbols {
		fileText := buildFileLevelText(syms)
		contentHash := computeHash(fileText)

		// Check if embedding already exists for this file
		exists, err := idx.db.FileEmbeddingExists(fileID, contentHash)
		if err != nil {
			return fmt.Errorf("failed to check file embedding existence: %w", err)
		}

		if !exists {
			tasks = append(tasks, embeddingTask{
				symbol:      nil, // file-level has no specific symbol
				fileID:      fileID,
				granularity: "file",
				text:        fileText,
			})
		}
	}

	for _, sym := range symbols {
		// Get file information for this symbol
		fileInfo := fileInfoMap[sym.FileID]
		if fileInfo == nil {
			continue // Skip if file info not available
		}
		filePath := fileInfo.Path
		imports := fileImportsMap[sym.FileID]
		if imports == nil {
			imports = []*db.ImportGraph{}
		}

		// Generate class-level embedding for classes/structs/interfaces
		if sym.Kind == "class" || sym.Kind == "struct" || sym.Kind == "interface" {
			classText := buildClassLevelText(sym, filePath, imports)
			contentHash := computeHash(classText)

			// Check if embedding already exists
			exists, err := idx.db.EmbeddingExists(sym.ID, "class", contentHash)
			if err != nil {
				return fmt.Errorf("failed to check embedding existence: %w", err)
			}

			if !exists {
				tasks = append(tasks, embeddingTask{
					symbol:      sym,
					granularity: "class",
					text:        classText,
				})
			}
		}

		// Generate function-level embedding for functions/methods
		if sym.Kind == "function" || sym.Kind == "method" {
			// Get call graph for this function
			callees, err := idx.db.GetCallGraphByCaller(sym.ID)
			if err != nil {
				fmt.Printf("Warning: failed to get callees for symbol %d: %v\n", sym.ID, err)
				callees = []*db.CallGraph{} // Use empty slice on error
			}

			functionText := buildFunctionLevelText(sym, filePath, imports, callees)
			contentHash := computeHash(functionText)

			// Check if embedding already exists
			exists, err := idx.db.EmbeddingExists(sym.ID, "function", contentHash)
			if err != nil {
				return fmt.Errorf("failed to check embedding existence: %w", err)
			}

			if !exists {
				tasks = append(tasks, embeddingTask{
					symbol:      sym,
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

	// Determine batch size and number of workers
	batchSize := idx.config.Embedding.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	// Process embeddings in parallel batches
	numWorkers := idx.config.Performance.MaxWorkers
	if numWorkers <= 0 {
		numWorkers = 4
	}

	// Split tasks into batches
	var batches [][]embeddingTask
	for i := 0; i < len(tasks); i += batchSize {
		end := i + batchSize
		if end > len(tasks) {
			end = len(tasks)
		}
		batches = append(batches, tasks[i:end])
	}

	fmt.Printf("Processing %d batches with %d workers...\n", len(batches), numWorkers)

	// Progress tracking
	progress := NewProgressBar(int64(len(batches)), "")

	// Create worker pool for batch processing
	batchJobs := make(chan []embeddingTask, len(batches))
	batchResults := make(chan error, len(batches))

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for batch := range batchJobs {
				select {
				case <-ctx.Done():
					batchResults <- ctx.Err()
					return
				default:
					// Extract texts for this batch
					texts := make([]string, len(batch))
					for i, task := range batch {
						texts[i] = task.text
					}

					// Generate embeddings for this batch
					embeds, err := embedding.EmbedBatch(ctx, idx.embedClient, texts, len(texts))
					if err != nil {
						batchResults <- fmt.Errorf("worker %d: failed to generate embeddings: %w", workerID, err)
						continue
					}

					// Store embeddings in database
					for i, task := range batch {
						contentHash := computeHash(task.text)

						var emb *db.Embedding

						if task.granularity == "file" {
							// File-level embedding
							file, err := idx.db.GetFile(task.fileID)
							if err != nil {
								batchResults <- fmt.Errorf("worker %d: failed to get file: %w", workerID, err)
								break
							}

							// Create snippet from the beginning of the text
							snippet := task.text
							if len(snippet) > 500 {
								snippet = snippet[:500] + "..."
							}

							emb = &db.Embedding{
								SymbolID:     nil, // File-level has no specific symbol
								FileID:       task.fileID,
								Granularity:  "file",
								Vector:       embeds[i].Vector,
								ContentHash:  contentHash,
								RepositoryID: file.RepositoryID,
								FilePath:     file.Path,
								Language:     file.Language,
								SymbolKind:   "",
								SymbolName:   "",
								Scope:        "",
								Snippet:      snippet,
								StartLine:    0,
								EndLine:      0,
							}
						} else {
							// Symbol-level embedding (class or function)
							file, err := idx.db.GetFile(task.symbol.FileID)
							if err != nil {
								batchResults <- fmt.Errorf("worker %d: failed to get file: %w", workerID, err)
								break
							}

							// Create snippet from docstring or signature
							snippet := task.symbol.Docstring
							if snippet == "" {
								snippet = task.symbol.Signature
							}
							if len(snippet) > 500 {
								snippet = snippet[:500] + "..."
							}

							emb = &db.Embedding{
								SymbolID:     &task.symbol.ID,
								FileID:       task.symbol.FileID,
								Granularity:  task.granularity,
								Vector:       embeds[i].Vector,
								ContentHash:  contentHash,
								RepositoryID: file.RepositoryID,
								FilePath:     file.Path,
								Language:     file.Language,
								SymbolKind:   task.symbol.Kind,
								SymbolName:   task.symbol.Name,
								Scope:        task.symbol.Scope,
								Snippet:      snippet,
								StartLine:    task.symbol.StartLine,
								EndLine:      task.symbol.EndLine,
							}
						}

						_, err = idx.db.CreateEmbedding(emb)
						if err != nil {
							batchResults <- fmt.Errorf("worker %d: failed to store embedding: %w", workerID, err)
							break
						}
					}

					batchResults <- nil

					// Update progress
					progress.Increment()
				}
			}
		}(w)
	}

	// Send batch jobs
	for _, batch := range batches {
		batchJobs <- batch
	}
	close(batchJobs)

	// Wait for all workers to finish
	wg.Wait()
	close(batchResults)

	// Collect errors
	var errors []error
	for err := range batchResults {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("embedding generation completed with %d errors: %v", len(errors), errors[0])
	}

	progress.Finish()
	fmt.Printf("✓ Generated %d embeddings successfully\n", len(tasks))
	return nil
}

// buildClassLevelText creates text representation for class-level embedding
// Enhanced with file path and imports for better context
func buildClassLevelText(sym *db.Symbol, filePath string, imports []*db.ImportGraph) string {
	var parts []string

	// Add file path
	if filePath != "" {
		parts = append(parts, fmt.Sprintf("File: %s", filePath))
	}

	// Add imports
	if len(imports) > 0 {
		importPaths := make([]string, 0, len(imports))
		seen := make(map[string]bool)
		for _, imp := range imports {
			if !seen[imp.ImportPath] {
				importPaths = append(importPaths, imp.ImportPath)
				seen[imp.ImportPath] = true
			}
		}
		if len(importPaths) > 0 {
			parts = append(parts, fmt.Sprintf("Imports: %s", strings.Join(importPaths, ", ")))
		}
	}

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
// Enhanced with file path, imports, and call graph for better context
func buildFunctionLevelText(sym *db.Symbol, filePath string, imports []*db.ImportGraph, callees []*db.CallGraph) string {
	var parts []string

	// Add file path
	if filePath != "" {
		parts = append(parts, fmt.Sprintf("File: %s", filePath))
	}

	// Add imports
	if len(imports) > 0 {
		importPaths := make([]string, 0, len(imports))
		seen := make(map[string]bool)
		for _, imp := range imports {
			if !seen[imp.ImportPath] {
				importPaths = append(importPaths, imp.ImportPath)
				seen[imp.ImportPath] = true
			}
		}
		if len(importPaths) > 0 {
			parts = append(parts, fmt.Sprintf("Imports: %s", strings.Join(importPaths, ", ")))
		}
	}

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

	// Add call information (functions this symbol calls)
	if len(callees) > 0 {
		calleeNames := make([]string, 0, len(callees))
		seen := make(map[string]bool)
		for _, callee := range callees {
			if !seen[callee.CalleeName] {
				calleeNames = append(calleeNames, callee.CalleeName)
				seen[callee.CalleeName] = true
			}
		}
		if len(calleeNames) > 0 {
			parts = append(parts, fmt.Sprintf("Calls: %s", strings.Join(calleeNames, ", ")))
		}
	}

	// Add code body
	if sym.CodeBody != "" {
		parts = append(parts, sym.CodeBody)
	}

	return strings.Join(parts, "\n\n")
}

// buildFileLevelText creates text representation for file-level embedding
func buildFileLevelText(symbols []*db.Symbol) string {
	var parts []string

	// Add file summary from top-level docstrings
	for _, sym := range symbols {
		// Include top-level classes and important functions
		if sym.Kind == "class" || sym.Kind == "struct" || sym.Kind == "interface" {
			if sym.Docstring != "" {
				parts = append(parts, fmt.Sprintf("%s %s: %s", sym.Kind, sym.Name, sym.Docstring))
			} else if sym.Signature != "" {
				parts = append(parts, sym.Signature)
			}
		} else if sym.Kind == "function" && sym.Scope == "" {
			// Top-level functions only
			if sym.Docstring != "" {
				parts = append(parts, fmt.Sprintf("function %s: %s", sym.Name, sym.Docstring))
			} else if sym.Signature != "" {
				parts = append(parts, sym.Signature)
			}
		}
	}

	// If we have too many items, truncate
	text := strings.Join(parts, "\n")
	if len(text) > 3000 {
		text = text[:3000] + "\n... (truncated)"
	}

	return text
}

// analyzeOwnership analyzes code ownership for all files in a repository
func (idx *Indexer) analyzeOwnership(ctx context.Context, repoID int64, repoPath string) error {
	// Get all files for this repository
	files, err := idx.db.ListFilesByRepository(repoID)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files to analyze")
		return nil
	}

	fmt.Printf("Analyzing ownership for %d files...\n", len(files))

	progress := NewProgressBar(int64(len(files)), "Analyzing ownership")

	var successCount int
	var errorCount int

	for _, file := range files {
		// Build absolute file path
		absPath := filepath.Join(repoPath, file.Path)

		// Analyze ownership for this file
		if err := idx.ownershipAnalyzer.AnalyzeFile(ctx, repoPath, absPath, file.ID); err != nil {
			// Don't fail the entire operation for a single file
			errorCount++
		} else {
			successCount++
		}
		progress.Increment()
	}

	progress.Finish()

	if errorCount > 0 {
		fmt.Printf("✓ Analyzed %d files, %d errors\n", successCount, errorCount)
	} else {
		fmt.Printf("✓ Analyzed %d files successfully\n", successCount)
	}

	return nil
}

// computeHash computes SHA256 hash of a string
func computeHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}
