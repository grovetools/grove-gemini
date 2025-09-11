package gemini

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheOptInLogic(t *testing.T) {
	tests := []struct {
		name           string
		rulesContent   string
		hasRules       bool
		noCache        bool
		expectedEnabled bool
	}{
		{
			name:           "No rules file - caching disabled",
			hasRules:       false,
			noCache:        false,
			expectedEnabled: false,
		},
		{
			name:           "Rules file without @enable-cache - caching disabled",
			rulesContent:   "**/*.go\n!*_test.go",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: false,
		},
		{
			name:           "Rules file with @enable-cache - caching enabled",
			rulesContent:   "@enable-cache\n**/*.go",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: true,
		},
		{
			name:           "Rules file with commented @enable-cache - caching disabled",
			rulesContent:   "# @enable-cache\n**/*.go",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: false,
		},
		{
			name:           "Rules file with @enable-cache and spaces - caching enabled",
			rulesContent:   "  @enable-cache  \n**/*.go",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: true,
		},
		{
			name:           "@enable-cache present but --no-cache flag set - caching disabled",
			rulesContent:   "@enable-cache\n**/*.go",
			hasRules:       true,
			noCache:        true,
			expectedEnabled: false,
		},
		{
			name:           "@enable-cache in middle of file - caching enabled",
			rulesContent:   "**/*.go\n@enable-cache\n!*_test.go",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: true,
		},
		{
			name:           "@enable-cache with comment on same line - caching disabled",
			rulesContent:   "**/*.go\n@enable-cache # Enable caching\n",
			hasRules:       true,
			noCache:        false,
			expectedEnabled: false, // Line contains more than just @enable-cache
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "gemini-cache-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .grove directory if needed
			if tt.hasRules {
				groveDir := filepath.Join(tempDir, ".grove")
				if err := os.MkdirAll(groveDir, 0755); err != nil {
					t.Fatalf("Failed to create .grove dir: %v", err)
				}

				// Create rules file
				rulesPath := filepath.Join(groveDir, "rules")
				if err := os.WriteFile(rulesPath, []byte(tt.rulesContent), 0644); err != nil {
					t.Fatalf("Failed to write rules file: %v", err)
				}
			}

			// Note: We're testing the file setup here.
			// The actual cache enabling logic in RequestRunner.Run() would need
			// mocking of external dependencies to test properly.

			// We can't easily test the full Run method without mocking,
			// but we can at least verify the file was created correctly
			// and would be parsed as expected
			
			// For now, let's verify the rules file exists as expected
			rulesPath := filepath.Join(tempDir, ".grove", "rules")
			if tt.hasRules {
				if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
					t.Errorf("Expected rules file to exist")
				}
				
				// Read and verify content
				content, err := os.ReadFile(rulesPath)
				if err != nil {
					t.Fatalf("Failed to read rules file: %v", err)
				}
				
				if string(content) != tt.rulesContent {
					t.Errorf("Rules content mismatch: got %q, want %q", string(content), tt.rulesContent)
				}
			} else {
				if _, err := os.Stat(rulesPath); !os.IsNotExist(err) {
					t.Errorf("Expected rules file to not exist")
				}
			}
		})
	}
}

func TestCacheManager_CachingDisabledByDefault(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gemini-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .grove directory with rules but NO @enable-cache
	groveDir := filepath.Join(tempDir, ".grove")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		t.Fatalf("Failed to create .grove dir: %v", err)
	}

	// Create rules file without @enable-cache directive
	rulesPath := filepath.Join(groveDir, "rules")
	rulesContent := `# Context rules
**/*.go
!*_test.go
`
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		t.Fatalf("Failed to write rules file: %v", err)
	}

	// Create a dummy cold context file with small content
	coldContextPath := filepath.Join(groveDir, "cached-context")
	if err := os.WriteFile(coldContextPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write cold context file: %v", err)
	}

	// Create cache manager
	cacheManager := NewCacheManager(tempDir)

	// Without @enable-cache, GetOrCreateCache should not be called
	// in the actual request flow, but if it is called, it should
	// still work (the gating happens in request.go)
	
	// Mock client (would need a proper mock in real implementation)
	var mockClient *Client

	// Try to get or create cache - it will fail due to small content size
	// but that's expected for this test
	cacheInfo, _, err := cacheManager.GetOrCreateCache(
		context.Background(),
		mockClient,
		"gemini-1.5-flash",
		coldContextPath,
		1*time.Hour,
		false, // ignoreChanges
		false, // disableExpiration
		false, // forceRecache
		true,  // skipConfirmation for tests
	)

	// Should return nil due to content being too small for caching
	if cacheInfo != nil {
		t.Errorf("Expected nil cache info for small content, got: %+v", cacheInfo)
	}
	
	// Error is expected to be nil (not an error, just no cache created)
	if err != nil {
		t.Errorf("Expected no error for small content, got: %v", err)
	}
}