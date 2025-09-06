package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	contextmgr "github.com/mattsolo1/grove-context/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
	"google.golang.org/api/googleapi"
	"google.golang.org/genai"
)

// CacheInfo stores information about cached files.
// It includes the cache ID, name, file hashes for validation,
// the model used, creation/expiration timestamps, token count, repo name,
// clear tracking information, and usage statistics.
type CacheInfo struct {
	CacheID          string            `json:"cache_id"`
	CacheName        string            `json:"cache_name"`
	CachedFileHashes map[string]string `json:"cached_file_hashes"`
	Model            string            `json:"model"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	TokenCount       int               `json:"token_count,omitempty"`
	RepoName         string            `json:"repo_name,omitempty"`
	ClearReason      string            `json:"clear_reason,omitempty"`
	ClearedAt        *time.Time        `json:"cleared_at,omitempty"`
	RegenerationCount int              `json:"regeneration_count,omitempty"`
	
	// Usage tracking fields
	UsageStats       *CacheUsageStats  `json:"usage_stats,omitempty"`
}

// CacheUsageStats tracks usage statistics for a cache
type CacheUsageStats struct {
	TotalQueries     int               `json:"total_queries"`
	LastUsed         time.Time         `json:"last_used"`
	TotalCacheHits   int64             `json:"total_cache_hits"`   // Total cached tokens served
	TotalTokensSaved int64             `json:"total_tokens_saved"` // Tokens saved by using cache
	AverageHitRate   float64           `json:"average_hit_rate"`   // Average cache hit rate across all queries
	QueryHistory     []CacheQueryStats `json:"query_history,omitempty"` // Optional detailed history
}

// CacheQueryStats tracks statistics for a single query using the cache
type CacheQueryStats struct {
	Timestamp        time.Time `json:"timestamp"`
	CachedTokens     int32     `json:"cached_tokens"`
	DynamicTokens    int32     `json:"dynamic_tokens"`
	CompletionTokens int32     `json:"completion_tokens"`
	CacheHitRate     float64   `json:"cache_hit_rate"`
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

// SaveCacheInfo saves cache information to a JSON file
func SaveCacheInfo(filePath string, info *CacheInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache info: %w", err)
	}
	
	// Write to temporary file first for atomic operation
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}
	
	// Rename temporary file to final location (atomic operation)
	if err := os.Rename(tempFile, filePath); err != nil {
		// Clean up temp file if rename fails
		os.Remove(tempFile)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	
	return nil
}

// FindAndValidateCache finds and validates a specific cache by name
// This method does NOT check for file content changes - it's meant to force use of a specific cache
func (m *CacheManager) FindAndValidateCache(ctx context.Context, client *Client, cacheName string, disableExpiration bool) (*CacheInfo, error) {
	// Create pretty logger
	logger := pretty.New()
	
	// Construct path to cache info file
	cacheInfoFile := filepath.Join(m.cacheDir, "hybrid_"+cacheName+".json")
	
	// Load cache info
	info, err := LoadCacheInfo(cacheInfoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache '%s' not found", cacheName)
		}
		return nil, fmt.Errorf("loading cache info: %w", err)
	}
	
	logger.Info(fmt.Sprintf("Found cache '%s' for model %s", cacheName, info.Model))
	
	// Verify cache exists on the server
	exists, err := client.VerifyCacheExists(ctx, info.CacheID)
	if err != nil {
		return nil, fmt.Errorf("verifying cache on server: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("cache '%s' no longer exists on server", cacheName)
	}
	
	// Check if cache has expired (unless expiration is disabled)
	if !disableExpiration && time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("cache '%s' has expired (expired at %s)", cacheName, info.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
	}
	
	// Cache is valid
	if disableExpiration {
		logger.Success(fmt.Sprintf("Using specified cache '%s' (expiration check disabled)", cacheName))
	} else {
		logger.Success(fmt.Sprintf("Using specified cache '%s' (expires %s)", cacheName, info.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST")))
	}
	
	return info, nil
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
	var existingRegenerationCount int
	needNewCache := forceRecache

	if forceRecache {
		logger.Info("Forcing cache regeneration due to --recache flag")
	}

	// Check for existing cache info to preserve regeneration count
	if data, err := os.ReadFile(cacheInfoFile); err == nil {
		var existingInfo CacheInfo
		if err := json.Unmarshal(data, &existingInfo); err == nil {
			existingRegenerationCount = existingInfo.RegenerationCount
		}
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
			TokenCount:       estimatedTokens,
			RepoName:         getRepoName(m.workingDir),
			RegenerationCount: existingRegenerationCount + 1,
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
	// Check for googleapi.Error
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == 404
	}
	// Check for genai.APIError
	if apiErr, ok := err.(genai.APIError); ok {
		return apiErr.Code == 404
	}
	return false
}

// IsPermissionError checks if an error is a Google API permission/forbidden error
func IsPermissionError(err error) bool {
	// Check for googleapi.Error
	if apiErr, ok := err.(*googleapi.Error); ok {
		return apiErr.Code == 403
	}
	// Check for genai.APIError
	if apiErr, ok := err.(genai.APIError); ok {
		return apiErr.Code == 403
	}
	return false
}

// getRepoName returns the name of the git repository for the given working directory
func getRepoName(workingDir string) string {
	// Try to get git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		// Not a git repo or git command failed
		return ""
	}
	
	// Get the repository root path
	gitRoot := strings.TrimSpace(string(output))
	if gitRoot == "" {
		return ""
	}
	
	// Extract the directory name as the repo name
	return filepath.Base(gitRoot)
}

// UpdateCacheUsageStats updates usage statistics for a cache after it's been used
func (m *CacheManager) UpdateCacheUsageStats(cacheID string, cachedTokens, dynamicTokens, completionTokens int, cacheHitRate float64) error {
	// Find the cache file by searching for the cache ID
	files, err := os.ReadDir(m.cacheDir)
	if err != nil {
		return fmt.Errorf("reading cache directory: %w", err)
	}
	
	var cacheFile string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
			filePath := filepath.Join(m.cacheDir, file.Name())
			info, err := LoadCacheInfo(filePath)
			if err != nil {
				continue
			}
			if info.CacheID == cacheID {
				cacheFile = filePath
				break
			}
		}
	}
	
	if cacheFile == "" {
		// Cache file not found, which is OK - it might be in a different project
		return nil
	}
	
	// Load current cache info
	info, err := LoadCacheInfo(cacheFile)
	if err != nil {
		return fmt.Errorf("loading cache info: %w", err)
	}
	
	// Initialize usage stats if needed
	if info.UsageStats == nil {
		info.UsageStats = &CacheUsageStats{
			QueryHistory: []CacheQueryStats{},
		}
	}
	
	// Update statistics
	info.UsageStats.TotalQueries++
	info.UsageStats.LastUsed = time.Now()
	info.UsageStats.TotalCacheHits += int64(cachedTokens)
	info.UsageStats.TotalTokensSaved += int64(cachedTokens) // Tokens saved by not re-processing
	
	// Update average hit rate
	if info.UsageStats.TotalQueries == 1 {
		info.UsageStats.AverageHitRate = cacheHitRate
	} else {
		// Running average
		info.UsageStats.AverageHitRate = ((info.UsageStats.AverageHitRate * float64(info.UsageStats.TotalQueries-1)) + cacheHitRate) / float64(info.UsageStats.TotalQueries)
	}
	
	// Add to query history (limit to last 100 queries to avoid unbounded growth)
	queryStats := CacheQueryStats{
		Timestamp:        time.Now(),
		CachedTokens:     int32(cachedTokens),
		DynamicTokens:    int32(dynamicTokens),
		CompletionTokens: int32(completionTokens),
		CacheHitRate:     cacheHitRate,
	}
	
	info.UsageStats.QueryHistory = append(info.UsageStats.QueryHistory, queryStats)
	if len(info.UsageStats.QueryHistory) > 100 {
		// Keep only the last 100 queries
		info.UsageStats.QueryHistory = info.UsageStats.QueryHistory[len(info.UsageStats.QueryHistory)-100:]
	}
	
	// Save updated cache info
	return SaveCacheInfo(cacheFile, info)
}

// CacheAnalytics represents aggregated analytics for a cache
type CacheAnalytics struct {
	EfficiencyScore   float64   // 0-100 score based on hit rate and cost savings
	TotalSavings      float64   // Total cost savings in USD
	AverageSavingsPerQuery float64 // Average savings per query
	PeakUsageHour     int       // Hour of day with most usage (0-23)
	PeakUsageDay      string    // Day of week with most usage
	UsageByHour       [24]int   // Usage count by hour
	UsageByDay        map[string]int // Usage count by day of week
	HitRateTrend     []float64 // Recent hit rates for trending
}

// CalculateCacheAnalytics computes analytics for a given cache
func CalculateCacheAnalytics(info *CacheInfo) *CacheAnalytics {
	if info.UsageStats == nil || info.UsageStats.TotalQueries == 0 {
		return &CacheAnalytics{
			UsageByDay: make(map[string]int),
		}
	}
	
	analytics := &CacheAnalytics{
		UsageByDay: make(map[string]int),
	}
	
	// Calculate cost savings based on model and token counts
	costPerMillion := getCostPerMillionTokens(info.Model)
	totalCachedTokens := float64(info.UsageStats.TotalCacheHits)
	
	// Savings = cached tokens cost - (cached tokens cost * 0.25 for cache discount)
	// Gemini gives 75% discount on cached tokens
	analytics.TotalSavings = (totalCachedTokens / 1_000_000) * costPerMillion * 0.75
	
	if info.UsageStats.TotalQueries > 0 {
		analytics.AverageSavingsPerQuery = analytics.TotalSavings / float64(info.UsageStats.TotalQueries)
	}
	
	// Calculate efficiency score (0-100)
	// Based on: hit rate (50%), usage frequency (25%), cost savings (25%)
	hitRateScore := info.UsageStats.AverageHitRate * 50
	
	// Usage frequency score (normalize to 0-25 based on queries per day)
	daysSinceCreation := time.Since(info.CreatedAt).Hours() / 24
	if daysSinceCreation < 1 {
		daysSinceCreation = 1
	}
	queriesPerDay := float64(info.UsageStats.TotalQueries) / daysSinceCreation
	usageScore := math.Min(queriesPerDay * 2.5, 25) // Cap at 25 points
	
	// Cost savings score (normalize to 0-25 based on savings)
	savingsScore := math.Min(analytics.TotalSavings * 5, 25) // Cap at 25 points
	
	analytics.EfficiencyScore = hitRateScore + usageScore + savingsScore
	
	// Analyze usage patterns
	if len(info.UsageStats.QueryHistory) > 0 {
		// Count usage by hour and day
		for _, query := range info.UsageStats.QueryHistory {
			hour := query.Timestamp.Hour()
			dayName := query.Timestamp.Weekday().String()
			
			analytics.UsageByHour[hour]++
			analytics.UsageByDay[dayName]++
		}
		
		// Find peak usage hour
		maxHourUsage := 0
		for hour, count := range analytics.UsageByHour {
			if count > maxHourUsage {
				maxHourUsage = count
				analytics.PeakUsageHour = hour
			}
		}
		
		// Find peak usage day
		maxDayUsage := 0
		for day, count := range analytics.UsageByDay {
			if count > maxDayUsage {
				maxDayUsage = count
				analytics.PeakUsageDay = day
			}
		}
		
		// Calculate hit rate trend (last 10 queries)
		startIdx := len(info.UsageStats.QueryHistory) - 10
		if startIdx < 0 {
			startIdx = 0
		}
		
		for i := startIdx; i < len(info.UsageStats.QueryHistory); i++ {
			analytics.HitRateTrend = append(analytics.HitRateTrend, 
				info.UsageStats.QueryHistory[i].CacheHitRate)
		}
	}
	
	return analytics
}

// getCostPerMillionTokens returns the cost per million tokens for a given model
func getCostPerMillionTokens(model string) float64 {
	// Gemini pricing as of 2024
	switch {
	case strings.Contains(model, "gemini-exp"):
		return 2.50 // $2.50 per million input tokens
	case strings.Contains(model, "pro"):
		return 0.50 // $0.50 per million input tokens  
	case strings.Contains(model, "flash"):
		return 0.15 // $0.15 per million input tokens
	default:
		return 0.50 // Default to pro pricing
	}
}

