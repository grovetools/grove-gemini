package gemini

import (
	"context"
	"testing"
)

func TestEmbedText_Validation(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GEMINI_API_KEY", "test-key")

	client, err := NewClient(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Empty text via EmbedBatch should fail
	_, err = client.EmbedBatch(ctx, "gemini-embedding-001", []string{}, "RETRIEVAL_DOCUMENT")
	if err == nil {
		t.Error("Expected error for empty texts, got nil")
	}
}

func TestEmbedBatch_Validation(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GEMINI_API_KEY", "test-key")

	client, err := NewClient(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name      string
		model     string
		texts     []string
		taskType  string
		expectErr bool
	}{
		{
			name:      "empty texts",
			model:     "gemini-embedding-001",
			texts:     []string{},
			taskType:  "RETRIEVAL_DOCUMENT",
			expectErr: true,
		},
		{
			name:      "nil texts",
			model:     "gemini-embedding-001",
			texts:     nil,
			taskType:  "RETRIEVAL_DOCUMENT",
			expectErr: true,
		},
		{
			name:      "valid input will fail on API with fake key",
			model:     "gemini-embedding-001",
			texts:     []string{"hello world"},
			taskType:  "RETRIEVAL_QUERY",
			expectErr: true, // API call fails with fake key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.EmbedBatch(ctx, tt.model, tt.texts, tt.taskType)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultEmbeddingModel(t *testing.T) {
	if DefaultEmbeddingModel != "gemini-embedding-001" {
		t.Errorf("Expected default embedding model to be gemini-embedding-001, got %s", DefaultEmbeddingModel)
	}
}
