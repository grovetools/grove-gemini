package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattsolo1/grove-gemini/pkg/config"
	ctxinfo "github.com/mattsolo1/grove-gemini/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/logging"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
	"google.golang.org/genai"
)

// Client wraps the Google Generative AI client
type Client struct {
	client *genai.Client
}

// NewClient creates a new Gemini client
func NewClient(ctx context.Context, apiKeyOverride string) (*Client, error) {
	var apiKey string
	var err error

	if apiKeyOverride != "" {
		apiKey = apiKeyOverride
	} else {
		apiKey, err = config.ResolveAPIKey()
		if err != nil {
			return nil, err
		}
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{client: client}, nil
}

// GenerateContentOptions contains options for content generation
type GenerateContentOptions struct {
	WorkingDir  string
	Caller      string
	IsNewCache  bool
	PromptFiles []string // Paths to prompt files to be included in the request
	JobID       string   // Job ID for logging purposes
	PlanName    string   // Plan name for logging purposes
}

// GeminiRequestLog holds the details of a request for debugging purposes
type GeminiRequestLog struct {
	Timestamp        time.Time `json:"timestamp"`
	Model            string    `json:"model"`
	CacheID          string    `json:"cache_id,omitempty"`
	PromptText       string    `json:"prompt_text"`
	AttachedFiles    []string  `json:"attached_files"`
	TotalFiles       int       `json:"total_files"`
	WorkingDir       string    `json:"working_dir,omitempty"`
	JobID            string    `json:"job_id,omitempty"`
	PlanName         string    `json:"plan_name,omitempty"`
}

// GenerateContentWithCache generates content using a cached context and dynamic files
func (c *Client) GenerateContentWithCache(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string) (string, error) {
	return c.GenerateContentWithCacheAndOptions(ctx, model, prompt, cacheID, dynamicFilePaths, nil)
}

// GenerateContentWithCacheAndOptions generates content with additional context options
func (c *Client) GenerateContentWithCacheAndOptions(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string, opts *GenerateContentOptions) (string, error) {
	// Create pretty logger
	logger := pretty.New()
	
	// Create a map to track uploaded files and prevent duplicates
	uploadedFiles := make(map[string]bool)
	allFilesToUpload := []string{}
	
	// Collect all files to upload (dynamic files + prompt files)
	for _, filePath := range dynamicFilePaths {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return "", fmt.Errorf("resolving dynamic file path %s: %w", filePath, err)
		}
		if !uploadedFiles[absPath] {
			allFilesToUpload = append(allFilesToUpload, absPath)
			uploadedFiles[absPath] = true
		}
	}
	
	// Add prompt files to the upload list (avoiding duplicates)
	if opts != nil && len(opts.PromptFiles) > 0 {
		for _, pFile := range opts.PromptFiles {
			absPath, err := filepath.Abs(pFile)
			if err != nil {
				return "", fmt.Errorf("resolving prompt file path %s: %w", pFile, err)
			}
			if !uploadedFiles[absPath] {
				allFilesToUpload = append(allFilesToUpload, absPath)
				uploadedFiles[absPath] = true
			}
		}
	}
	
	// Debug logging for Gemini requests (moved before upload to ensure it happens even if upload fails)
	if os.Getenv("GROVE_DEBUG") != "" && opts != nil && opts.WorkingDir != "" {
		// Determine log directory
		var promptLogDir string
		if opts.PlanName != "" {
			// Use plan-specific directory if available
			promptLogDir = filepath.Join(opts.WorkingDir, ".grove", "logs", opts.PlanName, "prompts")
		} else {
			// Fallback to generic directory
			promptLogDir = filepath.Join(opts.WorkingDir, ".grove", "logs", "gemini_prompts")
		}
		
		if err := os.MkdirAll(promptLogDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Warning: could not create gemini prompt log directory: %v\n", err)
		} else {
			// Create the log entry
			logEntry := GeminiRequestLog{
				Timestamp:     time.Now(),
				Model:         model,
				CacheID:       cacheID,
				PromptText:    prompt,
				AttachedFiles: allFilesToUpload,
				TotalFiles:    len(allFilesToUpload),
				WorkingDir:    opts.WorkingDir,
				JobID:         opts.JobID,
				PlanName:      opts.PlanName,
			}
			
			logData, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG] Warning: could not marshal gemini log entry: %v\n", err)
			} else {
				timestamp := time.Now().Format("20060102150405")
				jobID := opts.JobID
				if jobID == "" {
					jobID = "unknown_job"
				}
				logFileName := fmt.Sprintf("%s-%s-gemini-request.json", jobID, timestamp)
				logFilePath := filepath.Join(promptLogDir, logFileName)
				
				if err := os.WriteFile(logFilePath, logData, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "[DEBUG] Warning: could not write gemini request log file: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "[DEBUG] Gemini request details for job '%s' saved to: %s\n", jobID, logFilePath)
				}
			}
		}
	}
	
	// Upload all files
	var requestParts []*genai.Part
	if len(allFilesToUpload) > 0 {
		fmt.Fprintln(os.Stderr)
		logger.UploadProgress(fmt.Sprintf("Uploading %d files for request...", len(allFilesToUpload)))
		for _, filePath := range allFilesToUpload {
			// Upload file
			f, err := uploadFile(ctx, c.client, filePath)
			if err != nil {
				return "", fmt.Errorf("failed to upload file %s: %w", filePath, err)
			}
			
			// Create part from URI
			part := genai.NewPartFromURI(f.URI, f.MIMEType)
			requestParts = append(requestParts, part)
		}
	}

	// Count tokens for the user prompt separately
	var promptTokens int
	if prompt != "" {
		// Count tokens for just the prompt text
		tokenResp, err := c.client.Models.CountTokens(ctx,
			model,
			[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}},
			nil,
		)
		if err == nil {
			promptTokens = int(tokenResp.TotalTokens)
		}
		// Continue even if token counting fails - it's not critical
		
		requestParts = append(requestParts, &genai.Part{Text: prompt})
	}
	
	// Create content object with all parts
	userTurn := &genai.Content{
		Role:  genai.RoleUser,
		Parts: requestParts,
	}
	
	// Create contents slice for API
	contentsForAPI := []*genai.Content{userTurn}
	
	// Display files that will be included in the prompt
	var promptFiles []string
	if opts != nil && len(opts.PromptFiles) > 0 {
		promptFiles = opts.PromptFiles
	}
	
	// Create display files list
	displayFiles := make([]string, len(dynamicFilePaths))
	copy(displayFiles, dynamicFilePaths)
	
	// Add prompt files to display list
	if len(promptFiles) > 0 {
		displayFiles = append(displayFiles, promptFiles...)
	}
	
	// Show files before making the request
	if len(displayFiles) > 0 {
		logger.FilesIncluded(displayFiles)
	}
	
	// Generate content with optional cache
	var result *genai.GenerateContentResponse
	var err error
	
	startTime := time.Now()
	logger.GeneratingResponse()
	
	if cacheID != "" {
		result, err = c.client.Models.GenerateContent(
			ctx,
			model,
			contentsForAPI,
			&genai.GenerateContentConfig{
				CachedContent: cacheID,
			},
		)
	} else {
		// No cache, just dynamic files
		result, err = c.client.Models.GenerateContent(
			ctx,
			model,
			contentsForAPI,
			&genai.GenerateContentConfig{},
		)
	}
	
	if err != nil {
		// Gather context information
		var contextInfo *ctxinfo.Info
		if opts != nil && opts.WorkingDir != "" {
			contextInfo = ctxinfo.GetContextInfo(opts.WorkingDir)
		} else {
			contextInfo = ctxinfo.GetContextInfo("")
		}
		
		// Log the failed query
		logger := logging.GetLogger()
		logEntry := logging.QueryLog{
			Timestamp:    startTime,
			Model:       model,
			Method:      "GenerateContent",
			ResponseTime: time.Since(startTime).Seconds(),
			Error:       err.Error(),
			CacheID:     cacheID,
			Success:     false,
			WorkingDir:  contextInfo.WorkingDir,
			GitRepo:     contextInfo.GitRepo,
			GitBranch:   contextInfo.GitBranch,
			GitCommit:   contextInfo.GitCommit,
		}
		if opts != nil && opts.Caller != "" {
			logEntry.Caller = opts.Caller
		} else {
			logEntry.Caller = ctxinfo.GetCaller()
		}
		logger.Log(logEntry)
		
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Show token usage and log the query
	if result.UsageMetadata != nil {
		// Extract all token components
		cachedTokens := int(result.UsageMetadata.CachedContentTokenCount)
		totalPromptTokens := int(result.UsageMetadata.PromptTokenCount)
		completionTokens := int(result.UsageMetadata.CandidatesTokenCount)
		
		// Calculate actual dynamic tokens (prompt tokens minus cached tokens)
		dynamicTokens := totalPromptTokens - cachedTokens
		
		// Extract isNewCache flag from options
		isNewCache := false
		if opts != nil {
			isNewCache = opts.IsNewCache
		}
		
		logger.TokenUsage(
			cachedTokens,
			dynamicTokens,
			completionTokens,
			promptTokens,
			duration,
			isNewCache,
		)
		
		// Calculate cache hit rate for logging
		cacheHitRate := float64(0)
		if totalPromptTokens > 0 {
			cacheHitRate = float64(cachedTokens) / float64(totalPromptTokens)
		}
		
		// Gather context information
		var contextInfo *ctxinfo.Info
		if opts != nil && opts.WorkingDir != "" {
			contextInfo = ctxinfo.GetContextInfo(opts.WorkingDir)
		} else {
			contextInfo = ctxinfo.GetContextInfo("")
		}
		
		// Log the query
		logger := logging.GetLogger()
		logEntry := logging.QueryLog{
			Timestamp:        startTime,
			Model:           model,
			Method:          "GenerateContent",
			CachedTokens:    result.UsageMetadata.CachedContentTokenCount,
			PromptTokens:    result.UsageMetadata.PromptTokenCount,
			UserPromptTokens: int32(promptTokens),
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:     result.UsageMetadata.TotalTokenCount,
			CacheHitRate:    cacheHitRate, // Store as decimal
			ResponseTime:    duration.Seconds(),
			EstimatedCost:   logging.EstimateCostWithCache(model, result.UsageMetadata.PromptTokenCount, result.UsageMetadata.CandidatesTokenCount, result.UsageMetadata.CachedContentTokenCount),
			CacheID:         cacheID,
			Success:         true,
			WorkingDir:      contextInfo.WorkingDir,
			GitRepo:         contextInfo.GitRepo,
			GitBranch:       contextInfo.GitBranch,
			GitCommit:       contextInfo.GitCommit,
		}
		
		if opts != nil && opts.Caller != "" {
			logEntry.Caller = opts.Caller
		} else {
			logEntry.Caller = ctxinfo.GetCaller()
		}
		
		if err := logger.Log(logEntry); err != nil {
			// Don't fail the request if logging fails
			fmt.Fprintf(os.Stderr, "Warning: Failed to log query: %v\n", err)
		}
		
		// Update cache usage statistics
		if cacheID != "" && opts != nil && opts.WorkingDir != "" {
			// Try to update cache usage stats
			cacheManager := NewCacheManager(opts.WorkingDir)
			if err := cacheManager.UpdateCacheUsageStats(cacheID, cachedTokens, dynamicTokens, completionTokens, cacheHitRate); err != nil {
				// Don't fail the request if updating stats fails
				fmt.Fprintf(os.Stderr, "Warning: Failed to update cache usage stats: %v\n", err)
			}
		}
	}

	return result.Text(), nil
}

// GetClient returns the underlying genai client for cache operations
func (c *Client) GetClient() *genai.Client {
	return c.client
}

// VerifyCacheExists checks if a cache exists on the server
func (c *Client) VerifyCacheExists(ctx context.Context, cacheID string) (bool, error) {
	_, err := c.client.Caches.Get(ctx, cacheID, nil)
	if err != nil {
		// Check if it's a 404 Not Found error
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to verify cache: %w", err)
	}
	return true, nil
}

// CachedContentInfo represents information about a cached content from the API
type CachedContentInfo struct {
	Name        string
	Model       string
	DisplayName string
	CreateTime  time.Time
	UpdateTime  time.Time
	ExpireTime  time.Time
	TokenCount  int32
}

// ListCachesFromAPI lists all cached contents from the Google API
func (c *Client) ListCachesFromAPI(ctx context.Context) ([]CachedContentInfo, error) {
	var caches []CachedContentInfo
	
	// Iterate through all cached contents using the All method
	for cache, err := range c.client.Caches.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("failed to list caches from API: %w", err)
		}
		
		tokenCount := int32(0)
		if cache.UsageMetadata != nil {
			tokenCount = cache.UsageMetadata.TotalTokenCount
		}
		
		info := CachedContentInfo{
			Name:        cache.Name,
			Model:       cache.Model,
			DisplayName: cache.DisplayName,
			CreateTime:  cache.CreateTime,
			UpdateTime:  cache.UpdateTime,
			ExpireTime:  cache.ExpireTime,
			TokenCount:  tokenCount,
		}
		caches = append(caches, info)
	}
	
	return caches, nil
}

// DeleteCache deletes a cache from the Google API
func (c *Client) DeleteCache(ctx context.Context, cacheID string) error {
	_, err := c.client.Caches.Delete(ctx, cacheID, nil)
	if err != nil {
		// If it's already deleted (404), don't return an error
		if IsNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}