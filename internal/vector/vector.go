package vector

import (
	"math"
	"sort"
)

// SearchResult represents a vector search result with similarity score
type SearchResult struct {
	SymbolID    int64
	Granularity string
	Score       float32 // Cosine similarity (0-1, higher is better)
}

// CosineSimilarity calculates the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical direction
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// SearchBySimilarity searches for similar vectors using brute-force cosine similarity
// embeddings: slice of (symbolID, granularity, vector) tuples
// queryVector: the query vector
// limit: maximum number of results to return
// Returns results sorted by descending similarity score
func SearchBySimilarity(embeddings []EmbeddingData, queryVector []float32, limit int) []SearchResult {
	results := make([]SearchResult, 0, len(embeddings))

	for _, emb := range embeddings {
		score := CosineSimilarity(queryVector, emb.Vector)
		results = append(results, SearchResult{
			SymbolID:    emb.SymbolID,
			Granularity: emb.Granularity,
			Score:       score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// EmbeddingData represents vector data for search
type EmbeddingData struct {
	SymbolID    int64
	Granularity string
	Vector      []float32
}
