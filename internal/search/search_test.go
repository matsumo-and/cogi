package search

import (
	"testing"
)

func TestPrepareFTS5Query(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single word",
			input:    "function",
			expected: "function",
		},
		{
			name:     "Multiple words",
			input:    "hello world",
			expected: "\"hello world\"",
		},
		{
			name:     "With extra spaces",
			input:    "  test query  ",
			expected: "\"test query\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareFTS5Query(tt.input)
			if result != tt.expected {
				t.Errorf("prepareFTS5Query(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected string
	}{
		{
			name:     "Zero",
			n:        0,
			expected: "",
		},
		{
			name:     "One",
			n:        1,
			expected: "?",
		},
		{
			name:     "Three",
			n:        3,
			expected: "?, ?, ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := placeholders(tt.n)
			if result != tt.expected {
				t.Errorf("placeholders(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestSortByScore(t *testing.T) {
	results := []Result{
		{SymbolID: 1, Score: 0.5},
		{SymbolID: 2, Score: 0.9},
		{SymbolID: 3, Score: 0.3},
		{SymbolID: 4, Score: 0.7},
	}

	sortByScore(results)

	expectedOrder := []int64{2, 4, 1, 3}
	for i, expected := range expectedOrder {
		if results[i].SymbolID != expected {
			t.Errorf("After sorting, position %d: expected symbol ID %d, got %d",
				i, expected, results[i].SymbolID)
		}
	}

	// Verify scores are in descending order
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Errorf("Scores not in descending order at position %d: %f < %f",
				i, results[i].Score, results[i+1].Score)
		}
	}
}

func TestMergeResults(t *testing.T) {
	keyword := []Result{
		{SymbolID: 1, SymbolName: "func1", Score: 0.8},
		{SymbolID: 2, SymbolName: "func2", Score: 0.6},
	}

	semantic := []Result{
		{SymbolID: 2, SymbolName: "func2", Score: 0.9}, // Duplicate
		{SymbolID: 3, SymbolName: "func3", Score: 0.7},
	}

	merged := mergeResults(keyword, semantic, 0.5, 0.5)

	// Should have 3 unique results
	if len(merged) != 3 {
		t.Errorf("Expected 3 unique results, got %d", len(merged))
	}

	// Check that symbol 2 has combined score
	foundSymbol2 := false
	for _, r := range merged {
		if r.SymbolID == 2 {
			foundSymbol2 = true
			expectedScore := float32(0.6*0.5 + 0.9*0.5) // (keyword * kwWeight) + (semantic * semWeight)
			if r.Score != expectedScore {
				t.Errorf("Expected combined score %f for symbol 2, got %f",
					expectedScore, r.Score)
			}
		}
	}

	if !foundSymbol2 {
		t.Error("Symbol 2 not found in merged results")
	}

	// Verify results are sorted by score
	for i := 0; i < len(merged)-1; i++ {
		if merged[i].Score < merged[i+1].Score {
			t.Errorf("Merged results not sorted by score at position %d", i)
		}
	}
}
