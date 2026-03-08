package db

import (
	"os"
	"testing"
	"time"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "cogi_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()
	db, err := Open(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

// TestRepositoryCRUD tests repository CRUD operations
func TestRepositoryCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository
	repo, err := db.CreateRepository("test-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if repo.Name != "test-repo" {
		t.Errorf("Expected name 'test-repo', got '%s'", repo.Name)
	}

	// Get repository
	fetched, err := db.GetRepository(repo.ID)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if fetched.ID != repo.ID {
		t.Errorf("Expected ID %d, got %d", repo.ID, fetched.ID)
	}

	// List repositories
	repos, err := db.ListRepositories()
	if err != nil {
		t.Fatalf("Failed to list repositories: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}
}

// TestSymbolCRUD tests symbol CRUD operations
func TestSymbolCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup: Create repository and file
	repo, _ := db.CreateRepository("test-repo", "/path/to/repo")
	file, _ := db.CreateFile(repo.ID, "test.go", "go", "hash123", time.Now())

	// Create symbol
	symbol := &Symbol{
		FileID:      file.ID,
		Name:        "TestFunc",
		Kind:        "function",
		StartLine:   10,
		StartColumn: 1,
		EndLine:     20,
		EndColumn:   2,
		Visibility:  "public",
		Signature:   "func TestFunc()",
		CodeBody:    "func TestFunc() {}",
	}

	symbolID, err := db.CreateSymbol(symbol)
	if err != nil {
		t.Fatalf("Failed to create symbol: %v", err)
	}

	// Get symbol
	fetched, err := db.GetSymbol(symbolID)
	if err != nil {
		t.Fatalf("Failed to get symbol: %v", err)
	}

	if fetched.Name != "TestFunc" {
		t.Errorf("Expected name 'TestFunc', got '%s'", fetched.Name)
	}

	// Search symbols by name
	symbols, err := db.SearchSymbolsByName("TestFunc")
	if err != nil {
		t.Fatalf("Failed to search symbols: %v", err)
	}

	if len(symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(symbols))
	}

	// List symbols by file
	fileSymbols, err := db.ListSymbolsByFile(file.ID)
	if err != nil {
		t.Fatalf("Failed to list symbols: %v", err)
	}

	if len(fileSymbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(fileSymbols))
	}
}

// TestCallGraphCRUD tests call graph CRUD operations
func TestCallGraphCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup: Create repository, file, and symbols
	repo, _ := db.CreateRepository("test-repo", "/path/to/repo")
	file, _ := db.CreateFile(repo.ID, "test.go", "go", "hash123", time.Now())

	caller := &Symbol{
		FileID:    file.ID,
		Name:      "Caller",
		Kind:      "function",
		StartLine: 10,
	}
	callerID, _ := db.CreateSymbol(caller)

	callee := &Symbol{
		FileID:    file.ID,
		Name:      "Callee",
		Kind:      "function",
		StartLine: 20,
	}
	calleeID, _ := db.CreateSymbol(callee)

	// Create call graph entry
	cg := &CallGraph{
		CallerSymbolID: callerID,
		CalleeName:     "Callee",
		CallLine:       12,
		CallColumn:     5,
		CallType:       "direct",
	}
	cg.CalleeSymbolID.Valid = true
	cg.CalleeSymbolID.Int64 = calleeID

	cgID, err := db.CreateCallGraph(cg)
	if err != nil {
		t.Fatalf("Failed to create call graph: %v", err)
	}

	if cgID == 0 {
		t.Error("Expected non-zero call graph ID")
	}

	// Get call graph by caller
	callGraphs, err := db.GetCallGraphByCaller(callerID)
	if err != nil {
		t.Fatalf("Failed to get call graph by caller: %v", err)
	}

	if len(callGraphs) != 1 {
		t.Errorf("Expected 1 call graph entry, got %d", len(callGraphs))
	}

	if callGraphs[0].CalleeName != "Callee" {
		t.Errorf("Expected callee name 'Callee', got '%s'", callGraphs[0].CalleeName)
	}

	// Get call graph by callee
	callerGraphs, err := db.GetCallGraphByCallee(calleeID)
	if err != nil {
		t.Fatalf("Failed to get call graph by callee: %v", err)
	}

	if len(callerGraphs) != 1 {
		t.Errorf("Expected 1 caller, got %d", len(callerGraphs))
	}
}

// TestImportGraphCRUD tests import graph CRUD operations
func TestImportGraphCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup: Create repository and file
	repo, _ := db.CreateRepository("test-repo", "/path/to/repo")
	file, _ := db.CreateFile(repo.ID, "test.go", "go", "hash123", time.Now())

	// Create import graph entry
	ig := &ImportGraph{
		FileID:          file.ID,
		ImportPath:      "fmt",
		ImportType:      "default",
		ImportedSymbols: `["Println","Printf"]`,
		LineNumber:      5,
	}

	igID, err := db.CreateImportGraph(ig)
	if err != nil {
		t.Fatalf("Failed to create import graph: %v", err)
	}

	if igID == 0 {
		t.Error("Expected non-zero import graph ID")
	}

	// Get import graph by file
	importGraphs, err := db.GetImportGraphByFile(file.ID)
	if err != nil {
		t.Fatalf("Failed to get import graph by file: %v", err)
	}

	if len(importGraphs) != 1 {
		t.Errorf("Expected 1 import graph entry, got %d", len(importGraphs))
	}

	if importGraphs[0].ImportPath != "fmt" {
		t.Errorf("Expected import path 'fmt', got '%s'", importGraphs[0].ImportPath)
	}

	// Get import graph by path
	fmtImports, err := db.GetImportGraphByPath("fmt")
	if err != nil {
		t.Fatalf("Failed to get import graph by path: %v", err)
	}

	if len(fmtImports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(fmtImports))
	}
}

// TestBatchOperations tests batch insert operations
func TestBatchOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup
	repo, _ := db.CreateRepository("test-repo", "/path/to/repo")
	file, _ := db.CreateFile(repo.ID, "test.go", "go", "hash123", time.Now())
	symbolID, _ := db.CreateSymbol(&Symbol{
		FileID:    file.ID,
		Name:      "TestFunc",
		Kind:      "function",
		StartLine: 10,
	})

	// Batch create call graphs
	callGraphs := []*CallGraph{
		{
			CallerSymbolID: symbolID,
			CalleeName:     "func1",
			CallLine:       11,
			CallColumn:     5,
			CallType:       "direct",
		},
		{
			CallerSymbolID: symbolID,
			CalleeName:     "func2",
			CallLine:       12,
			CallColumn:     5,
			CallType:       "direct",
		},
	}

	err := db.BatchCreateCallGraph(callGraphs)
	if err != nil {
		t.Fatalf("Failed to batch create call graphs: %v", err)
	}

	// Verify
	cgs, _ := db.GetCallGraphByCaller(symbolID)
	if len(cgs) != 2 {
		t.Errorf("Expected 2 call graph entries, got %d", len(cgs))
	}

	// Batch create import graphs
	importGraphs := []*ImportGraph{
		{
			FileID:     file.ID,
			ImportPath: "fmt",
			ImportType: "default",
			LineNumber: 3,
		},
		{
			FileID:     file.ID,
			ImportPath: "strings",
			ImportType: "default",
			LineNumber: 4,
		},
	}

	err = db.BatchCreateImportGraph(importGraphs)
	if err != nil {
		t.Fatalf("Failed to batch create import graphs: %v", err)
	}

	// Verify
	igs, _ := db.GetImportGraphByFile(file.ID)
	if len(igs) != 2 {
		t.Errorf("Expected 2 import graph entries, got %d", len(igs))
	}
}
