package gemini

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattsolo1/grove-gemini/pkg/config"
	ctxinfo "github.com/mattsolo1/grove-gemini/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/logging"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
	"github.com/sirupsen/logrus"
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
	// Generation parameters
	Temperature     *float32
	TopP            *float32
	TopK            *int32
	MaxOutputTokens *int32
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
	// Get request ID from environment for tracing
	requestID := os.Getenv("GROVE_REQUEST_ID")

	// Create pretty logger for UI output
	logger := pretty.NewWithLogger(log)
	
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
	
	// Structured logging for Gemini requests using grove-core logging
	// This logs detailed request information when log level is set to debug
	if log.Logger.IsLevelEnabled(logrus.DebugLevel) {
		// Create structured log fields
		fields := logrus.Fields{
			"request_id":     requestID,
			"timestamp":      time.Now(),
			"model":          model,
			"cache_id":       cacheID,
			"prompt_text":    prompt,
			"attached_files": allFilesToUpload,
			"total_files":    len(allFilesToUpload),
		}

		// Add optional fields if available
		if opts != nil {
			if opts.WorkingDir != "" {
				fields["working_dir"] = opts.WorkingDir
			}
			if opts.JobID != "" {
				fields["job_id"] = opts.JobID
			}
			if opts.PlanName != "" {
				fields["plan_name"] = opts.PlanName
			}
		}

		// Log with structured fields
		log.WithFields(fields).Debug("Preparing Gemini API request")
	}
	
	// Upload all files
	var requestParts []*genai.Part
	var uploadResults []FileUploadResult
	if len(allFilesToUpload) > 0 {
		fmt.Fprintln(os.Stderr)
		logger.UploadProgressCtx(ctx, fmt.Sprintf("Uploading %d files for request...", len(allFilesToUpload)))
		for _, filePath := range allFilesToUpload {
			// Upload file
			f, duration, err := uploadFile(ctx, c.client, filePath)
			if err != nil {
				return "", fmt.Errorf("failed to upload file %s: %w", filePath, err)
			}

			// Track upload result
			uploadResults = append(uploadResults, FileUploadResult{
				FilePath:   filePath,
				FileURI:    f.URI,
				MIMEType:   f.MIMEType,
				DurationMs: duration.Milliseconds(),
			})

			// Create part from URI
			part := genai.NewPartFromURI(f.URI, f.MIMEType)
			requestParts = append(requestParts, part)
		}

		// Log all uploads as a single structured log entry
		if len(uploadResults) > 0 {
			log.WithFields(logrus.Fields{
				"request_id":   requestID,
				"file_count":   len(uploadResults),
				"uploads":      uploadResults,
				"total_time_ms": func() int64 {
					var total int64
					for _, r := range uploadResults {
						total += r.DurationMs
					}
					return total
				}(),
			}).Info("Files uploaded to Gemini API")
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
	
	// Create display files list (deduplicated - use the already deduplicated list)
	displayFiles := allFilesToUpload

	// Show files before making the request
	if len(displayFiles) > 0 {
		logger.FilesIncludedCtx(ctx, displayFiles)
	}

	// Generate content with optional cache
	var result *genai.GenerateContentResponse
	var err error

	startTime := time.Now()
	logger.GeneratingResponse()

	// Build generation config with parameters
	config := &genai.GenerateContentConfig{}

	// Add cache if provided
	if cacheID != "" {
		config.CachedContent = cacheID
	}

	// Add generation parameters from options
	if opts != nil {
		if opts.Temperature != nil {
			config.Temperature = opts.Temperature
		}
		if opts.TopP != nil {
			config.TopP = opts.TopP
		}
		if opts.TopK != nil {
			// Convert int32 to float32 for TopK
			topKFloat := float32(*opts.TopK)
			config.TopK = &topKFloat
		}
		if opts.MaxOutputTokens != nil {
			config.MaxOutputTokens = int32(*opts.MaxOutputTokens)
		}
	}

	log.WithFields(logrus.Fields{
		"request_id": requestID,
		"model":      model,
		"cache_id":   cacheID,
	}).Info("Calling Gemini API")

	result, err = c.client.Models.GenerateContent(
		ctx,
		model,
		contentsForAPI,
		config,
	)
	
	if err != nil {
		// Gather context information
		var contextInfo *ctxinfo.Info
		if opts != nil && opts.WorkingDir != "" {
			contextInfo = ctxinfo.GetContextInfo(opts.WorkingDir)
		} else {
			contextInfo = ctxinfo.GetContextInfo("")
		}
		
		// Log the failed query
		geminiLogger := logging.GetLogger()
		logEntry := logging.QueryLog{
			Timestamp:    startTime,
			RequestID:    requestID,
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
		if err := geminiLogger.Log(logEntry); err != nil {
			// Don't fail the request if logging fails
			fmt.Fprintf(os.Stderr, "Warning: Failed to log query: %v\n", err)
		}
		
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
		
		// Calculate cache hit rate for logging
		cacheHitRate := float64(0)
		if totalPromptTokens > 0 {
			cacheHitRate = float64(cachedTokens) / float64(totalPromptTokens)
		}
		
		logger.TokenUsageCtx(
			ctx,
			cachedTokens,
			dynamicTokens,
			completionTokens,
			promptTokens,
			duration,
			isNewCache,
		)
		
		// Gather context information
		var contextInfo *ctxinfo.Info
		if opts != nil && opts.WorkingDir != "" {
			contextInfo = ctxinfo.GetContextInfo(opts.WorkingDir)
		} else {
			contextInfo = ctxinfo.GetContextInfo("")
		}
		
		// Log the query
		geminiLogger := logging.GetLogger()
		logEntry := logging.QueryLog{
			Timestamp:        startTime,
			RequestID:        requestID,
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
		
		if err := geminiLogger.Log(logEntry); err != nil {
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
		// Debug: log the error type
		// fmt.Fprintf(os.Stderr, "DEBUG: DeleteCache error type: %T, error: %v\n", err, err)
		
		// If it's already deleted (404) or permission denied (403), don't return an error
		// 403 often means the cache doesn't exist on GCP
		if IsNotFoundError(err) || IsPermissionError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}