package graph

import (
	"os"
	"testing"
	"time"

	"github.com/matsumo_and/cogi/internal/db"
)

// setupTestDB creates a temporary test database with sample data
func setupTestDB(t *testing.T) (*db.DB, func()) {
	tmpFile, err := os.CreateTemp("", "cogi_graph_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	dbPath := tmpFile.Name()
	database, err := db.Open(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.Remove(dbPath)
	}

	return database, cleanup
}

// TestGetCallersTree tests retrieving callers of a function
func TestGetCallersTree(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	repo, _ := database.CreateRepository("test-repo", "/test")
	file, _ := database.CreateFile(repo.ID, "test.go", "go", "hash", time.Now())

	// Create symbols: main -> Run -> Helper
	helperID, _ := database.CreateSymbol(&db.Symbol{
		FileID:    file.ID,
		Name:      "Helper",
		Kind:      "function",
		StartLine: 10,
	})

	runID, _ := database.CreateSymbol(&db.Symbol{
		FileID:    file.ID,
		Name:      "Run",
		Kind:      "function",
		StartLine: 20,
	})

	mainID, _ := database.CreateSymbol(&db.Symbol{
		FileID:    file.ID,
		Name:      "main",
		Kind:      "function",
		StartLine: 30,
	})

	// Create call graph: Run calls Helper, main calls Run
	cg1 := &db.CallGraph{
		CallerSymbolID: runID,
		CalleeName:     "Helper",
		CallLine:       22,
		CallColumn:     5,
		CallType:       "direct",
	}
	cg1.CalleeSymbolID.Valid = true
	cg1.CalleeSymbolID.Int64 = helperID
	database.CreateCallGraph(cg1)

	cg2 := &db.CallGraph{
		CallerSymbolID: mainID,
		CalleeName:     "Run",
		CallLine:       32,
		CallColumn:     5,
		CallType:       "direct",
	}
	cg2.CalleeSymbolID.Valid = true
	cg2.CalleeSymbolID.Int64 = runID
	database.CreateCallGraph(cg2)

	// Test: Get callers of Helper
	gm := New(database)
	nodes, err := gm.GetCallersTree("Helper", 2)
	if err != nil {
		t.Fatalf("Failed to get callers tree: %v", err)
	}

	if len(nodes) < 1 {
		t.Errorf("Expected at least 1 caller, got %d", len(nodes))
	}

	// Verify Run is in the callers
	foundRun := false
	for _, node := range nodes {
		if node.Symbol.Name == "Run" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Error("Expected 'Run' to be a caller of 'Helper'")
	}
}

// TestGetCalleesTree tests retrieving callees of a function
func TestGetCalleesTree(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	repo, _ := database.CreateRepository("test-repo", "/test")
	file, _ := database.CreateFile(repo.ID, "test.go", "go", "hash", time.Now())

	// Create symbols
	mainID, _ := database.CreateSymbol(&db.Symbol{
		FileID:    file.ID,
		Name:      "main",
		Kind:      "function",
		StartLine: 10,
	})

	helperID, _ := database.CreateSymbol(&db.Symbol{
		FileID:    file.ID,
		Name:      "Helper",
		Kind:      "function",
		StartLine: 20,
	})

	// Create call graph: main calls Helper
	cg := &db.CallGraph{
		CallerSymbolID: mainID,
		CalleeName:     "Helper",
		CallLine:       12,
		CallColumn:     5,
		CallType:       "direct",
	}
	cg.CalleeSymbolID.Valid = true
	cg.CalleeSymbolID.Int64 = helperID
	database.CreateCallGraph(cg)

	// Test: Get callees of main
	gm := New(database)
	nodes, err := gm.GetCalleesTree("main", 2)
	if err != nil {
		t.Fatalf("Failed to get callees tree: %v", err)
	}

	if len(nodes) < 1 {
		t.Errorf("Expected at least 1 callee, got %d", len(nodes))
	}

	// Verify Helper is in the callees
	foundHelper := false
	for _, node := range nodes {
		if node.Symbol.Name == "Helper" {
			foundHelper = true
			break
		}
	}
	if !foundHelper {
		t.Error("Expected 'Helper' to be a callee of 'main'")
	}
}

// TestGetImportDependencies tests retrieving import dependencies
func TestGetImportDependencies(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup test data
	repo, _ := database.CreateRepository("test-repo", "/test")
	file, _ := database.CreateFile(repo.ID, "main.go", "go", "hash", time.Now())

	// Create import graph entries
	imports := []*db.ImportGraph{
		{
			FileID:          file.ID,
			ImportPath:      "fmt",
			ImportType:      "default",
			ImportedSymbols: "",
			LineNumber:      3,
		},
		{
			FileID:          file.ID,
			ImportPath:      "strings",
			ImportType:      "default",
			ImportedSymbols: "",
			LineNumber:      4,
		},
	}
	database.BatchCreateImportGraph(imports)

	// Test: Get dependencies
	gm := New(database)
	nodes, err := gm.GetImportDependencies("main.go", 1)
	if err != nil {
		t.Fatalf("Failed to get import dependencies: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(nodes))
	}

	// Verify imports
	foundFmt := false
	foundStrings := false
	for _, node := range nodes {
		if node.ImportPath == "fmt" {
			foundFmt = true
		}
		if node.ImportPath == "strings" {
			foundStrings = true
		}
	}

	if !foundFmt {
		t.Error("Expected 'fmt' import")
	}
	if !foundStrings {
		t.Error("Expected 'strings' import")
	}
}

// TestFormatCallTree tests call tree formatting
func TestFormatCallTree(t *testing.T) {
	nodes := []*CallNode{
		{
			Symbol: &db.Symbol{
				Name:      "Caller",
				Scope:     "",
				StartLine: 10,
			},
			Depth:    0,
			CallType: "direct",
		},
	}

	output := FormatCallTree(nodes, "callers")
	if output == "" {
		t.Error("Expected non-empty output")
	}

	if output == "No results found." {
		t.Error("Expected formatted tree, got 'No results found.'")
	}
}

// TestFormatImportTree tests import tree formatting
func TestFormatImportTree(t *testing.T) {
	nodes := []*ImportNode{
		{
			FilePath:   "main.go",
			ImportPath: "fmt",
			ImportType: "default",
			Symbols:    []string{},
			Depth:      0,
			LineNumber: 3,
		},
	}

	output := FormatImportTree(nodes, "dependencies")
	if output == "" {
		t.Error("Expected non-empty output")
	}

	if output == "No results found." {
		t.Error("Expected formatted tree, got 'No results found.'")
	}
}
