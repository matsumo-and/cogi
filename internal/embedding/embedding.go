package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedding represents a text embedding vector
type Embedding struct {
	Vector []float32
	Text   string
}

// Client is an interface for embedding providers
type Client interface {
	Embed(ctx context.Context, texts []string) ([]Embedding, error)
	EmbedSingle(ctx context.Context, text string) (Embedding, error)
	Dimension() int
}

// OllamaClient implements the Client interface for Ollama
type OllamaClient struct {
	endpoint  string
	model     string
	dimension int
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama embedding client
func NewOllamaClient(endpoint, model string, dimension int) *OllamaClient {
	return &OllamaClient{
		endpoint:  endpoint,
		model:     model,
		dimension: dimension,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ollamaEmbedRequest is the request format for Ollama embedding API
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the response format for Ollama embedding API
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// EmbedSingle generates an embedding for a single text
func (c *OllamaClient) EmbedSingle(ctx context.Context, text string) (Embedding, error) {
	embeddings, err := c.Embed(ctx, []string{text})
	if err != nil {
		return Embedding{}, err
	}
	if len(embeddings) == 0 {
		return Embedding{}, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// Embed generates embeddings for multiple texts
func (c *OllamaClient) Embed(ctx context.Context, texts []string) ([]Embedding, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	embeddings := make([]Embedding, 0, len(texts))

	for _, text := range texts {
		req := ollamaEmbedRequest{
			Model:  c.model,
			Prompt: text,
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/embeddings", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call Ollama API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
		}

		var embedResp ollamaEmbedResponse
		if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(embedResp.Embedding) != c.dimension {
			return nil, fmt.Errorf("expected embedding dimension %d, got %d", c.dimension, len(embedResp.Embedding))
		}

		embeddings = append(embeddings, Embedding{
			Vector: embedResp.Embedding,
			Text:   text,
		})
	}

	return embeddings, nil
}

// Dimension returns the dimension of the embeddings
func (c *OllamaClient) Dimension() int {
	return c.dimension
}

// EmbedBatch generates embeddings in batches to optimize performance
func EmbedBatch(ctx context.Context, client Client, texts []string, batchSize int) ([]Embedding, error) {
	if batchSize <= 0 {
		batchSize = 32 // default batch size
	}

	allEmbeddings := make([]Embedding, 0, len(texts))

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := client.Embed(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}
