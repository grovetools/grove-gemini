package gemini

import (
	"context"
	"fmt"
	"os"
	"time"

	ctxinfo "github.com/mattsolo1/grove-gemini/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/logging"
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
}

// GenerateContentWithCache generates content using a cached context and dynamic files
func (c *Client) GenerateContentWithCache(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string) (string, error) {
	return c.GenerateContentWithCacheAndOptions(ctx, model, prompt, cacheID, dynamicFilePaths, nil)
}

// GenerateContentWithCacheAndOptions generates content with additional context options
func (c *Client) GenerateContentWithCacheAndOptions(ctx context.Context, model string, prompt string, cacheID string, dynamicFilePaths []string, opts *GenerateContentOptions) (string, error) {
	// Upload dynamic files if any
	var requestParts []*genai.Part
	if len(dynamicFilePaths) > 0 {
		fmt.Fprintf(os.Stderr, "\n📤 Uploading dynamic files for request...\n")
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

	// Add the text prompt to requestParts
	if prompt != "" {
		requestParts = append(requestParts, &genai.Part{Text: prompt})
	}
	
	// Create content object with all parts
	userTurn := &genai.Content{
		Role:  genai.RoleUser,
		Parts: requestParts,
	}
	
	// Create contents slice for API
	contentsForAPI := []*genai.Content{userTurn}
	
	// Generate content with optional cache
	var result *genai.GenerateContentResponse
	var err error
	
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "\n🤖 Generating response...\n")
	
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
			Caller:      opts.Caller,
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
		fmt.Fprintf(os.Stderr, "\n📊 Token usage:\n")
		if cacheID != "" && result.UsageMetadata.CachedContentTokenCount > 0 {
			fmt.Fprintf(os.Stderr, "  Cached: %d tokens\n", result.UsageMetadata.CachedContentTokenCount)
		}
		fmt.Fprintf(os.Stderr, "  Dynamic + Prompt: %d tokens\n", result.UsageMetadata.PromptTokenCount)
		fmt.Fprintf(os.Stderr, "  Total: %d tokens\n", result.UsageMetadata.TotalTokenCount)
		
		cacheHitRate := float64(0)
		if result.UsageMetadata.CachedContentTokenCount > 0 && result.UsageMetadata.PromptTokenCount > 0 {
			cacheHitRate = float64(result.UsageMetadata.CachedContentTokenCount) / 
				float64(result.UsageMetadata.CachedContentTokenCount + result.UsageMetadata.PromptTokenCount) * 100
			fmt.Fprintf(os.Stderr, "  Cache usage: %.1f%%\n", cacheHitRate)
		}
		
		fmt.Fprintf(os.Stderr, "  Response time: %.2fs\n", duration.Seconds())
		
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
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:     result.UsageMetadata.TotalTokenCount,
			CacheHitRate:    cacheHitRate / 100, // Store as decimal
			ResponseTime:    duration.Seconds(),
			EstimatedCost:   logging.EstimateCost(model, result.UsageMetadata.PromptTokenCount, result.UsageMetadata.CandidatesTokenCount),
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