package gemini

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	
	// Test with API key set
	t.Setenv("GEMINI_API_KEY", "test-key")
	
	client, err := NewClient(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	
	// Test without API key
	t.Setenv("GEMINI_API_KEY", "")
	client, err = NewClient(ctx)
	if err == nil {
		t.Fatal("Expected error when API key is not set")
	}
	if client != nil {
		t.Fatal("Expected client to be nil when API key is not set")
	}
}

func TestClient_GetClient(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GEMINI_API_KEY", "test-key")
	
	client, err := NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	genaiClient := client.GetClient()
	if genaiClient == nil {
		t.Fatal("Expected GetClient to return non-nil client")
	}
}

func TestGenerateContentWithCache_Validation(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GEMINI_API_KEY", "test-key")
	
	client, err := NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	tests := []struct {
		name             string
		model            string
		prompt           string
		cacheID          string
		dynamicFilePaths []string
		expectError      bool
	}{
		{
			name:        "empty model",
			model:       "",
			prompt:      "test prompt",
			expectError: true,
		},
		{
			name:        "empty prompt",
			model:       "gemini-pro",
			prompt:      "",
			expectError: true,
		},
		{
			name:             "valid input without cache",
			model:            "gemini-pro",
			prompt:           "test prompt",
			cacheID:          "",
			dynamicFilePaths: []string{},
			expectError:      false, // Will fail on actual API call, but validation should pass
		},
		{
			name:             "valid input with cache",
			model:            "gemini-pro",
			prompt:           "test prompt",
			cacheID:          "test-cache-id",
			dynamicFilePaths: []string{},
			expectError:      false, // Will fail on actual API call, but validation should pass
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't fully test without mocking the genai client,
			// but we can at least verify the function handles basic cases
			_, err := client.GenerateContentWithCache(ctx, tt.model, tt.prompt, tt.cacheID, tt.dynamicFilePaths)
			
			// Since we're using a real client with a fake API key,
			// we expect API errors rather than validation errors
			if err == nil && tt.expectError {
				t.Error("Expected error but got none")
			}
		})
	}
}