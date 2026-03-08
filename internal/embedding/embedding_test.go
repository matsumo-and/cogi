package embedding

import (
	"context"
	"testing"
)

func TestOllamaClient_Dimension(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "nomic-embed-text", 768)

	if client.Dimension() != 768 {
		t.Errorf("Expected dimension 768, got %d", client.Dimension())
	}
}

func TestEmbedBatch(t *testing.T) {
	// Mock client for testing
	mockClient := &mockEmbeddingClient{
		dimension: 4,
	}

	texts := []string{"hello", "world", "test", "batch"}
	embeddings, err := EmbedBatch(context.Background(), mockClient, texts, 2)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(embeddings) != 4 {
		t.Errorf("Expected 4 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if emb.Text != texts[i] {
			t.Errorf("Expected text %s, got %s", texts[i], emb.Text)
		}
		if len(emb.Vector) != 4 {
			t.Errorf("Expected vector length 4, got %d", len(emb.Vector))
		}
	}
}

// Mock embedding client for testing
type mockEmbeddingClient struct {
	dimension int
}

func (m *mockEmbeddingClient) Embed(ctx context.Context, texts []string) ([]Embedding, error) {
	embeddings := make([]Embedding, len(texts))
	for i, text := range texts {
		embeddings[i] = Embedding{
			Vector: make([]float32, m.dimension),
			Text:   text,
		}
		// Fill with dummy values
		for j := range embeddings[i].Vector {
			embeddings[i].Vector[j] = float32(i + j)
		}
	}
	return embeddings, nil
}

func (m *mockEmbeddingClient) EmbedSingle(ctx context.Context, text string) (Embedding, error) {
	embeddings, err := m.Embed(ctx, []string{text})
	if err != nil {
		return Embedding{}, err
	}
	return embeddings[0], nil
}

func (m *mockEmbeddingClient) Dimension() int {
	return m.dimension
}
