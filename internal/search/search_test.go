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

