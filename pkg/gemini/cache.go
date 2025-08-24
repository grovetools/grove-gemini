package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	contextmgr "github.com/mattsolo1/grove-context/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
	"google.golang.org/api/googleapi"
	"google.golang.org/genai"
)

// CacheInfo stores information about cached files.
// It includes the cache ID, name, file hashes for validation,
// the model used, and creation/expiration timestamps.
type CacheInfo struct {
	CacheID          string            `json:"cache_id"`
	CacheName        string            `json:"cache_name"`
	CachedFileHashes map[string]string `json:"cached_file_hashes"`
	Model            string            `json:"model"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
}

// CacheManager manages the cache lifecycle for Gemini API.
// It handles cache creation, validation, and expiration tracking
// for cached content used with the Gemini API.
type CacheManager struct {
	workingDir string
	cacheDir   string
}

// NewCacheManager creates a new cache manager
func NewCacheManager(workingDir string) *CacheManager {
	cacheDir := filepath.Join(workingDir, ".grove", "gemini-cache")
	return &CacheManager{
		workingDir: workingDir,
		cacheDir:   cacheDir,
	}
}

// LoadCacheInfo loads cache information from a JSON file
func LoadCacheInfo(filePath string) (*CacheInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading cache info file: %w", err)
	}
	
	var info CacheInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing cache info: %w", err)
	}
	
	return &info, nil
}

// GetOrCreateCache returns an existing valid cache or creates a new one
// The second return value indicates whether a new cache was created
func (m *CacheManager) GetOrCreateCache(ctx context.Context, client *Client, model string, coldContextFilePath string, ttl time.Duration, ignoreChanges bool, disableExpiration bool, forceRecache bool, skipConfirmation bool) (*CacheInfo, bool, error) {
	// Create pretty logger
	logger := pretty.New()
	
	// Check if caching is disabled via grove-context directive
	contextManager := contextmgr.NewManager(m.workingDir)
	shouldDisableCache, err := contextManager.ShouldDisableCache()
	if err != nil {
		// Log warning but continue - don't fail if we can't read the directive
		logger.Warning(fmt.Sprintf("Could not check cache directive: %v", err))
	}
	
	if shouldDisableCache {
		logger.CacheDisabled()
		return nil, false, nil
	}

	// Check if the cold context file exists
	if _, err := os.Stat(coldContextFilePath); err != nil {
		if os.IsNotExist(err) {
			// No cold context file, return nil (no cache to use)
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("checking cold context file: %w", err)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return nil, false, fmt.Errorf("creating cache directory: %w", err)
	}

	// Generate cache key based on the cold context file content
	cacheKey, err := generateCacheKey([]string{coldContextFilePath})
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate cache key: %w", err)
	}
	cacheInfoFile := filepath.Join(m.cacheDir, "hybrid_"+cacheKey+".json")

	// Try to load existing cache info
	var cacheInfo CacheInfo
	needNewCache := forceRecache

	if forceRecache {
		logger.Info("Forcing cache regeneration due to --recache flag")
	}

	if !needNewCache {
		if data, err := os.ReadFile(cacheInfoFile); err == nil {
			if err := json.Unmarshal(data, &cacheInfo); err == nil {
				logger.CacheInfo("Found existing cache info")

				// Verify cache exists on the server
				exists, err := client.VerifyCacheExists(ctx, cacheInfo.CacheID)
				if err != nil {
					logger.Warning(fmt.Sprintf("Could not verify cache on server: %v", err))
				} else if !exists {
					logger.Warning("Cache not found on server - will create new cache")
					needNewCache = true
				}

				// Check if cache expired
				if !needNewCache && !disableExpiration && time.Now().After(cacheInfo.ExpiresAt) {
					logger.CacheExpired(cacheInfo.ExpiresAt)
					needNewCache = true
				} else if !needNewCache {
					if changed, changedFiles := hasFilesChanged(cacheInfo.CachedFileHashes, []string{coldContextFilePath}); changed {
						if ignoreChanges {
							logger.Warning("Cache is frozen - detected file changes but using existing cache")
							logger.ChangedFiles(changedFiles)
							return &cacheInfo, false, nil
						}
						logger.ChangedFiles(changedFiles)
						fmt.Fprintln(os.Stderr)
						logger.Warning("Cache invalidated due to file changes - new cache required")
						needNewCache = true
					} else {
						if disableExpiration {
							logger.Success("Cache is valid (expiration disabled by @no-expire)")
						} else {
							logger.CacheValid(cacheInfo.ExpiresAt)
						}
						return &cacheInfo, false, nil
					}
				}
			}
		} else {
			logger.NoCache()
			needNewCache = true
		}
	}

	// Create new cache if needed
	if needNewCache {
		// First, check if the file is large enough for caching
		content, err := os.ReadFile(coldContextFilePath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read %s: %w", coldContextFilePath, err)
		}

		estimatedTokens := estimateTokens(content)
		minTokensForCache := 4096

		if estimatedTokens < minTokensForCache {
			fmt.Fprintln(os.Stderr)
			logger.Warning("Cached context is too small for Gemini caching")
			fmt.Fprintf(os.Stderr, "   Estimated tokens: %d (minimum required: %d)\n", estimatedTokens, minTokensForCache)
			logger.Info("   Suggestion: Move all content to hot context (.grove/context) for better performance")
			logger.Info("   Proceeding without cache...")
			return nil, false, nil // Return nil to indicate no cache should be used
		}

		// Show confirmation prompt unless skipped
		if !skipConfirmation {
			sizeBytes := int64(len(content))
			logger.Info(fmt.Sprintf("Cache confirmation required (skipConfirmation=%v)", skipConfirmation))
			if !logger.CacheCreationPrompt(estimatedTokens, sizeBytes, ttl) {
				logger.Warning("Cache creation cancelled by user")
				return nil, false, nil
			}
		}

		fmt.Fprintln(os.Stderr)
		logger.UploadProgress("Uploading files for cache...")
		logger.EstimatedTokens(estimatedTokens)

		fileHashes := make(map[string]string)
		var parts []*genai.Part

		// Calculate hash
		hashArray := sha256.Sum256(content)
		hash := hex.EncodeToString(hashArray[:])
		fileHashes[coldContextFilePath] = hash

		// Upload file
		f, err := uploadFile(ctx, client.GetClient(), coldContextFilePath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to upload %s: %w", coldContextFilePath, err)
		}
		parts = append(parts, genai.NewPartFromURI(f.URI, f.MIMEType))

		// Create cache
		fmt.Fprintln(os.Stderr)
		logger.CreatingCache()
		contents := []*genai.Content{
			genai.NewContentFromParts(parts, genai.RoleUser),
		}

		cacheConfig := &genai.CreateCachedContentConfig{
			Contents: contents,
			TTL:      ttl,
		}

		cache, err := client.GetClient().Caches.Create(ctx, model, cacheConfig)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create cache: %w", err)
		}

		// Save cache info
		cacheInfo = CacheInfo{
			CacheID:          cache.Name,
			CacheName:        cacheKey,
			CachedFileHashes: fileHashes,
			Model:            model,
			CreatedAt:        time.Now(),
			ExpiresAt:        cache.ExpireTime,
		}

		data, _ := json.MarshalIndent(cacheInfo, "", "  ")
		
		// Write to temporary file first for atomic operation
		tempFile := cacheInfoFile + ".tmp"
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			return nil, false, fmt.Errorf("failed to save cache info to temp file: %w", err)
		}
		
		// Rename temporary file to final location (atomic operation)
		if err := os.Rename(tempFile, cacheInfoFile); err != nil {
			// Clean up temp file if rename fails
			os.Remove(tempFile)
			return nil, false, fmt.Errorf("failed to rename cache info file: %w", err)
		}

		logger.CacheCreated(cache.Name, cache.ExpireTime)
	}

	return &cacheInfo, needNewCache, nil
}

// hashFile calculates SHA256 hash of a file
func hashFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// generateCacheKey creates a unique key for a set of files based on their content
func generateCacheKey(files []string) (string, error) {
	h := sha256.New()
	h.Write([]byte("hybrid_v2")) // v2 indicates content-based hashing
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", f, err)
		}
		h.Write(content)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// estimateTokens provides a rough estimate of token count for a file
// Using a simple heuristic: ~1 token per 4 characters (common for code/text)
func estimateTokens(content []byte) int {
	return len(content) / 4
}

// hasFilesChanged checks if any files have changed and returns the changed files
func hasFilesChanged(oldHashes map[string]string, files []string) (bool, []string) {
	var changedFiles []string

	for _, file := range files {
		newHash, err := hashFile(file)
		if err != nil {
			changedFiles = append(changedFiles, fmt.Sprintf("%s (error reading file: %v)", file, err))
			continue
		}

		if oldHash, exists := oldHashes[file]; !exists {
			changedFiles = append(changedFiles, fmt.Sprintf("%s (new file)", file))
		} else if oldHash != newHash {
			changedFiles = append(changedFiles, file)
		}
	}

	return len(changedFiles) > 0, changedFiles
}

// IsNotFoundError checks if an error is a Google API "Not Found" error
func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == 404
	}
	return false
}

