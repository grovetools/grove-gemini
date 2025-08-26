package gemini

import (
	"context"
	"fmt"
	"os"
	"time"

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
func NewClient(ctx context.Context) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
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
	WorkingDir string
	Caller     string
	IsNewCache bool
	PromptFiles []string // Paths to prompt files to be included in the request
}

// GenerateContentWithCache generates content using a cached context and dynamic files
func (c *Client) GenerateContentWithCache(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string) (string, error) {
	return c.GenerateContentWithCacheAndOptions(ctx, model, prompt, cacheID, dynamicFilePaths, nil)
}

// GenerateContentWithCacheAndOptions generates content with additional context options
func (c *Client) GenerateContentWithCacheAndOptions(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string, opts *GenerateContentOptions) (string, error) {
	// Create pretty logger
	logger := pretty.New()
	
	// Upload dynamic files if any
	var requestParts []*genai.Part
	if len(dynamicFilePaths) > 0 {
		fmt.Fprintln(os.Stderr)
		logger.UploadProgress("Uploading dynamic files for request...")
		for _, filePath := range dynamicFilePaths {
			// Upload dynamic file
			f, err := uploadFile(ctx, c.client, filePath)
			if err != nil {
				return "", fmt.Errorf("failed to upload dynamic file %s: %w", filePath, err)
			}
			
			// Create part from URI
			part := genai.NewPartFromURI(f.URI, f.MIMEType)
			requestParts = append(requestParts, part)
		}
	}

	// Read and add content from prompt files
	if opts != nil && len(opts.PromptFiles) > 0 {
		for _, pFile := range opts.PromptFiles {
			content, err := os.ReadFile(pFile)
			if err != nil {
				return "", fmt.Errorf("failed to read prompt file %s: %w", pFile, err)
			}
			requestParts = append(requestParts, &genai.Part{Text: string(content)})
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