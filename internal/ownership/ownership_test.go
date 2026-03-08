package ownership

import (
	"testing"
	"time"
)

func TestParseGitBlame(t *testing.T) {
	t.Skip("Skipping git blame parsing test - requires actual git repository")
	// This test would need a real git repository to work properly
	// The parseGitBlame function is tested indirectly through integration tests
}

func TestGroupByAuthor(t *testing.T) {
	// Create mock blame results
	now := time.Now()
	results := []BlameResult{
		{Line: 1, CommitHash: "abc123", AuthorName: "John", AuthorEmail: "john@example.com", CommitDate: now},
		{Line: 2, CommitHash: "abc123", AuthorName: "John", AuthorEmail: "john@example.com", CommitDate: now},
		{Line: 3, CommitHash: "abc123", AuthorName: "John", AuthorEmail: "john@example.com", CommitDate: now},
		{Line: 4, CommitHash: "def456", AuthorName: "Jane", AuthorEmail: "jane@example.com", CommitDate: now},
		{Line: 5, CommitHash: "def456", AuthorName: "Jane", AuthorEmail: "jane@example.com", CommitDate: now},
		{Line: 6, CommitHash: "abc123", AuthorName: "John", AuthorEmail: "john@example.com", CommitDate: now},
	}

	analyzer := &Analyzer{}
	ranges := analyzer.groupByAuthor(results)

	if len(ranges) != 3 {
		t.Errorf("Expected 3 ranges, got %d", len(ranges))
	}

	// First range: John, lines 1-3
	if ranges[0].AuthorName != "John" {
		t.Errorf("Expected first range author 'John', got '%s'", ranges[0].AuthorName)
	}
	if ranges[0].StartLine != 1 || ranges[0].EndLine != 3 {
		t.Errorf("Expected first range lines 1-3, got %d-%d", ranges[0].StartLine, ranges[0].EndLine)
	}
	if ranges[0].CommitCount != 3 {
		t.Errorf("Expected first range commit count 3, got %d", ranges[0].CommitCount)
	}

	// Second range: Jane, lines 4-5
	if ranges[1].AuthorName != "Jane" {
		t.Errorf("Expected second range author 'Jane', got '%s'", ranges[1].AuthorName)
	}
	if ranges[1].StartLine != 4 || ranges[1].EndLine != 5 {
		t.Errorf("Expected second range lines 4-5, got %d-%d", ranges[1].StartLine, ranges[1].EndLine)
	}

	// Third range: John again, line 6
	if ranges[2].AuthorName != "John" {
		t.Errorf("Expected third range author 'John', got '%s'", ranges[2].AuthorName)
	}
	if ranges[2].StartLine != 6 || ranges[2].EndLine != 6 {
		t.Errorf("Expected third range line 6, got %d-%d", ranges[2].StartLine, ranges[2].EndLine)
	}
}

func TestGroupByAuthorEmptyResults(t *testing.T) {
	analyzer := &Analyzer{}
	ranges := analyzer.groupByAuthor([]BlameResult{})

	if len(ranges) != 0 {
		t.Errorf("Expected empty ranges for empty input, got %d", len(ranges))
	}
}

func TestGroupByAuthorSingleResult(t *testing.T) {
	now := time.Now()
	results := []BlameResult{
		{Line: 1, CommitHash: "abc123", AuthorName: "John", AuthorEmail: "john@example.com", CommitDate: now},
	}

	analyzer := &Analyzer{}
	ranges := analyzer.groupByAuthor(results)

	if len(ranges) != 1 {
		t.Errorf("Expected 1 range, got %d", len(ranges))
	}

	if ranges[0].StartLine != 1 || ranges[0].EndLine != 1 {
		t.Errorf("Expected range line 1, got %d-%d", ranges[0].StartLine, ranges[0].EndLine)
	}
}
