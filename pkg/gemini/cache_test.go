package gemini

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCacheManager(t *testing.T) {
	tmpDir := t.TempDir()
	
	cm := NewCacheManager(tmpDir)
	if cm == nil {
		t.Fatal("Expected CacheManager to be created")
	}
	if cm.workingDir != tmpDir {
		t.Errorf("Expected workingDir to be %s, got %s", tmpDir, cm.workingDir)
	}
	expectedCacheDir := filepath.Join(tmpDir, ".grove", "gemini-cache")
	if cm.cacheDir != expectedCacheDir {
		t.Errorf("Expected cacheDir to be %s, got %s", expectedCacheDir, cm.cacheDir)
	}
}

func TestCacheInfo_Structure(t *testing.T) {
	// Test that CacheInfo can be properly created and serialized
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	
	info := CacheInfo{
		CacheID:   "test-cache-id",
		CacheName: "test-cache-name",
		CachedFileHashes: map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		},
		Model:     "gemini-pro",
		CreatedAt: now,
		ExpiresAt: expires,
	}
	
	// Verify fields are set correctly
	if info.CacheID != "test-cache-id" {
		t.Errorf("Expected CacheID to be test-cache-id, got %s", info.CacheID)
	}
	if info.Model != "gemini-pro" {
		t.Errorf("Expected Model to be gemini-pro, got %s", info.Model)
	}
	if len(info.CachedFileHashes) != 2 {
		t.Errorf("Expected 2 file hashes, got %d", len(info.CachedFileHashes))
	}
}

func TestGetOrCreateCache_WithoutColdContext(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCacheManager(tmpDir)
	
	// Test with non-existent cold context file
	ctx := context.Background()
	nonExistentFile := filepath.Join(tmpDir, "non-existent.txt")
	
	// This should return nil without error (no cache to use)
	cacheInfo, _, err := cm.GetOrCreateCache(ctx, nil, "gemini-pro", nonExistentFile, 24*time.Hour, false, false, false, true)
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}
	if cacheInfo != nil {
		t.Error("Expected nil cache info for non-existent file")
	}
}

func TestGetOrCreateCache_SmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCacheManager(tmpDir)
	
	// Create a small file (less than 4096 estimated tokens)
	smallFile := filepath.Join(tmpDir, "small.txt")
	smallContent := "This is a small file with very little content."
	if err := os.WriteFile(smallFile, []byte(smallContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	ctx := context.Background()
	
	// This should return nil (file too small for caching)
	cacheInfo, _, err := cm.GetOrCreateCache(ctx, nil, "gemini-pro", smallFile, 24*time.Hour, false, false, false, true)
	if err != nil {
		t.Errorf("Expected no error for small file, got %v", err)
	}
	if cacheInfo != nil {
		t.Error("Expected nil cache info for small file")
	}
}