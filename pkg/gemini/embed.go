package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// DefaultEmbeddingModel is the default model for embedding generation.
const DefaultEmbeddingModel = "gemini-embedding-001"

// EmbedText generates an embedding for a single text string.
// The taskType parameter should be "RETRIEVAL_DOCUMENT" for documents being indexed
// or "RETRIEVAL_QUERY" for search queries.
func (c *Client) EmbedText(ctx context.Context, model string, text string, taskType string) ([]float32, error) {
	results, err := c.EmbedBatch(ctx, model, []string{text}, taskType)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return results[0], nil
}

// EmbedBatch generates embeddings for multiple text strings in a single API call.
// The taskType parameter should be "RETRIEVAL_DOCUMENT" for documents being indexed
// or "RETRIEVAL_QUERY" for search queries.
func (c *Client) EmbedBatch(ctx context.Context, model string, texts []string, taskType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	// Build contents from texts
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = &genai.Content{
			Parts: []*genai.Part{{Text: text}},
		}
	}

	config := &genai.EmbedContentConfig{}
	if taskType != "" {
		config.TaskType = taskType
	}

	resp, err := c.client.Models.EmbedContent(ctx, model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("embed content: %w", err)
	}

	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Embeddings))
	}

	results := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		results[i] = emb.Values
	}

	return results, nil
}
