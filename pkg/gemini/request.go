package gemini

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	grovecontext "github.com/mattsolo1/grove-context/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
)

// RequestOptions contains all the parameters for a request
type RequestOptions struct {
	Model           string
	Prompt          string
	PromptFiles     []string // Paths to files containing prompts (for display purposes)
	WorkDir         string
	CacheTTL        time.Duration
	NoCache         bool
	RegenerateCtx   bool
	Recache         bool
	UseCache        string
	ContextFiles    []string
	SkipConfirmation bool
	APIKey          string // Explicitly pass API key to avoid context issues
	// New fields for better logging context
	Caller   string
	JobID    string
	PlanName string
}

// RequestRunner handles the orchestration of Gemini API requests with context management
type RequestRunner struct {
	logger *pretty.Logger
}

// NewRequestRunner creates a new RequestRunner instance
func NewRequestRunner() *RequestRunner {
	return &RequestRunner{
		logger: pretty.New(),
	}
}

// Run executes a request with the given options
func (r *RequestRunner) Run(ctx context.Context, options RequestOptions) (string, error) {
	// Validate options
	if options.Prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}
	
	// Validate cache flags
	if options.UseCache != "" && options.Recache {
		return "", fmt.Errorf("UseCache and Recache are mutually exclusive")
	}

	// Determine working directory
	workDir := options.WorkDir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Make workDir absolute
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolving work directory: %w", err)
	}
	workDir = absWorkDir

	r.logger.WorkingDirectory(workDir)

	// Check for .grove/rules file
	rulesPath := filepath.Join(workDir, ".grove", "rules")
	hasRules := false
	if _, err := os.Stat(rulesPath); err == nil {
		hasRules = true
		r.logger.FoundRulesFile(rulesPath)
		
		// Log the rules file content
		rulesContent, err := os.ReadFile(rulesPath)
		if err == nil {
			r.logger.RulesFileContent(strings.TrimSpace(string(rulesContent)))
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("checking rules file: %w", err)
	}

	// Initialize context manager
	var ctxMgr *grovecontext.Manager
	if hasRules {
		ctxMgr = grovecontext.NewManager(workDir)
		
		// Regenerate context if requested or if context files don't exist
		coldContextFile := filepath.Join(workDir, ".grove", "cached-context")
		hotContextFile := filepath.Join(workDir, ".grove", "context")
		
		needsRegeneration := options.RegenerateCtx
		if !needsRegeneration {
			// Check if context files exist
			if _, err := os.Stat(coldContextFile); os.IsNotExist(err) {
				needsRegeneration = true
				r.logger.Warning("Cold context not found, will regenerate")
			} else if _, err := os.Stat(hotContextFile); os.IsNotExist(err) {
				needsRegeneration = true
				r.logger.Warning("Hot context not found, will regenerate")
			}
		}

		if needsRegeneration {
			fmt.Fprintln(os.Stderr)
			r.logger.Info("🔄 Regenerating context from rules...")
			
			// Update context from rules
			if err := ctxMgr.UpdateFromRules(); err != nil {
				return "", fmt.Errorf("updating context from rules: %w", err)
			}

			// Generate context file
			if err := ctxMgr.GenerateContext(true); err != nil {
				return "", fmt.Errorf("generating context: %w", err)
			}

			// Display stats
			files, _ := ctxMgr.ReadFilesList(grovecontext.FilesListFile)
			stats, err := ctxMgr.GetStats("request", files, 10)
			if err == nil {
				fmt.Fprintln(os.Stderr)
				r.logger.Info("📊 Context Summary:")
				fmt.Fprintf(os.Stderr, "  Total files: %d\n", stats.TotalFiles)
				fmt.Fprintf(os.Stderr, "  Total tokens: %s\n", grovecontext.FormatTokenCount(stats.TotalTokens))
				fmt.Fprintf(os.Stderr, "  Total size: %s\n", grovecontext.FormatBytes(int(stats.TotalSize)))

				if stats.TotalTokens > 500000 {
					return "", fmt.Errorf("context size exceeds limit: %d tokens (max 500,000)", stats.TotalTokens)
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	} else {
		r.logger.Warning("No .grove/rules file found - context management disabled")
		r.logger.Tip("Create .grove/rules to enable automatic context inclusion")
		fmt.Fprintln(os.Stderr)
	}

	// Initialize Gemini client
	geminiClient, err := NewClient(ctx, options.APIKey)
	if err != nil {
		return "", fmt.Errorf("creating Gemini client: %w", err)
	}

	// Prepare context files
	coldContextFile := filepath.Join(workDir, ".grove", "cached-context")
	hotContextFile := filepath.Join(workDir, ".grove", "context")

	// Initialize cache manager
	cacheManager := NewCacheManager(workDir)

	// Use provided TTL or default
	ttl := options.CacheTTL
	if ttl == 0 {
		ttl = 1 * time.Hour
	}

	// Check for @enable-cache directive in rules file (opt-in model)
	cachingEnabled := false
	if hasRules && !options.NoCache {
		rulesContent, err := os.ReadFile(rulesPath)
		if err == nil {
			// Caching is enabled only if @enable-cache is present
			if strings.Contains(string(rulesContent), "@enable-cache") {
				cachingEnabled = true
				// Display prominent warning about experimental caching
				r.logger.CacheWarning()
			}
		}
	}
	
	// Get cache directives from context manager if available
	var ignoreChanges, disableExpiration bool
	if ctxMgr != nil && cachingEnabled {
		// Check for custom expiration time
		if customTTL, err := ctxMgr.GetExpireTime(); err == nil && customTTL > 0 {
			ttl = customTTL
			r.logger.TTL(ttl.String())
		}

		// Check for @freeze-cache directive
		if freeze, err := ctxMgr.ShouldFreezeCache(); err == nil && freeze {
			ignoreChanges = true
			r.logger.CacheFrozen()
		}

		// Check for @no-expire directive
		if noExpire, err := ctxMgr.ShouldDisableExpiration(); err == nil && noExpire {
			disableExpiration = true
			r.logger.Info("🚫 Cache expiration disabled by @no-expire directive")
		}
	}

	// Get or create cache for cold context (if it exists and caching is enabled)
	var cacheInfo *CacheInfo
	var isNewCache bool
	if !options.NoCache && cachingEnabled {
		// Check if user specified a cache to use
		if options.UseCache != "" {
			r.logger.Info(fmt.Sprintf("Using specified cache: %s", options.UseCache))
			var err error
			cacheInfo, err = cacheManager.FindAndValidateCache(ctx, geminiClient, options.UseCache, disableExpiration)
			if err != nil {
				return "", fmt.Errorf("using specified cache: %w", err)
			}
			isNewCache = false
		} else {
			// Normal cache handling - create or find cache based on content
			if info, err := os.Stat(coldContextFile); err == nil && info.Size() > 0 {
				r.logger.Info(fmt.Sprintf("Cache settings: requestYes=%v, ignoreChanges=%v, disableExpiration=%v", options.SkipConfirmation, ignoreChanges, disableExpiration))
				cacheInfo, isNewCache, err = cacheManager.GetOrCreateCache(ctx, geminiClient, options.Model, coldContextFile, ttl, ignoreChanges, disableExpiration, options.Recache, options.SkipConfirmation)
				if err != nil {
					return "", fmt.Errorf("managing cache: %w", err)
				}
			} else if err == nil && info.Size() == 0 {
				r.logger.Warning("Cold context file is empty, skipping cache")
			} else if os.IsNotExist(err) && hasRules {
				r.logger.Warning("No cold context file found")
			}
		}
	} else if !options.NoCache && !cachingEnabled && hasRules {
		// Cache is disabled by default (no @enable-cache directive)
		if info, err := os.Stat(coldContextFile); err == nil && info.Size() > 0 {
			r.logger.CacheDisabledByDefault()
		}
	}

	// Prepare dynamic files
	var dynamicFiles []string
	
	// Add hot context if it exists
	if _, err := os.Stat(hotContextFile); err == nil {
		dynamicFiles = append(dynamicFiles, hotContextFile)
		r.logger.Info(fmt.Sprintf("Including hot context: %s", hotContextFile))
	}
	
	// If caching is not enabled, also include cold context as dynamic file
	if !cachingEnabled && cacheInfo == nil {
		if _, err := os.Stat(coldContextFile); err == nil {
			dynamicFiles = append(dynamicFiles, coldContextFile)
			r.logger.Info(fmt.Sprintf("Including cold context (cache disabled): %s", coldContextFile))
		}
	}

	// Add any additional context files
	for _, ctxFile := range options.ContextFiles {
		absPath, err := filepath.Abs(ctxFile)
		if err != nil {
			return "", fmt.Errorf("resolving context file %s: %w", ctxFile, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return "", fmt.Errorf("context file not found: %s", ctxFile)
		}
		dynamicFiles = append(dynamicFiles, absPath)
		r.logger.Info(fmt.Sprintf("Including additional context: %s", absPath))
	}

	// Also check for CLAUDE.md in the working directory
	claudePath := filepath.Join(workDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		dynamicFiles = append(dynamicFiles, claudePath)
		r.logger.Info(fmt.Sprintf("Including CLAUDE.md: %s", claudePath))
	}

	// Determine cache ID
	var cacheID string
	if cacheInfo != nil {
		cacheID = cacheInfo.CacheID
	}

	// Make the API request
	fmt.Fprintln(os.Stderr)
	r.logger.Model(options.Model)
	
	caller := "gemapi-request" // Default caller
	if options.Caller != "" {
		caller = options.Caller
	}
	
	opts := &GenerateContentOptions{
		WorkingDir: workDir,
		Caller:     caller,
		IsNewCache: isNewCache,
		PromptFiles: options.PromptFiles,
		JobID:       options.JobID,
		PlanName:    options.PlanName,
	}
	
	response, err := geminiClient.GenerateContentWithCacheAndOptions(ctx, options.Model, options.Prompt, cacheID, dynamicFiles, opts)
	if err != nil {
		return "", fmt.Errorf("Gemini API request failed: %w", err)
	}

	return response, nil
}