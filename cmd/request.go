package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	grovecontext "github.com/mattsolo1/grove-context/pkg/context"
	"github.com/mattsolo1/grove-gemini/pkg/gemini"
	"github.com/mattsolo1/grove-gemini/pkg/pretty"
	"github.com/spf13/cobra"
)

var (
	requestModel          string
	requestPrompt         string
	requestPromptFile     string
	requestWorkDir        string
	requestCacheTTL       string
	requestNoCache        bool
	requestRegenerateCtx  bool
	requestOutputFile     string
	requestContextFiles   []string
	requestYes            bool
)

func newRequestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request",
		Short: "Make a request to Gemini API with grove-context support",
		Long: `Make a request to the Gemini API using grove-context for automatic hot/cold context management.

This command works similarly to grove-flow's oneshot executor:
- Uses .grove/rules to generate context if available
- Manages cold context caching automatically
- Includes hot context as dynamic files
- Supports custom cache TTL and directives from rules file

Examples:
  # Simple prompt
  gemapi request -p "Explain the main function"
  
  # From file
  gemapi request -f prompt.md
  
  # With specific model and output file
  gemapi request -m gemini-2.0-flash -f prompt.md -o response.md
  
  # Regenerate context before request
  gemapi request --regenerate -p "Review the codebase architecture"
  
  # With custom working directory
  gemapi request -w /path/to/project -p "Analyze this project"`,
		RunE: runRequest,
	}

	cmd.Flags().StringVarP(&requestModel, "model", "m", "gemini-2.0-flash", "Gemini model to use")
	cmd.Flags().StringVarP(&requestPrompt, "prompt", "p", "", "Prompt text")
	cmd.Flags().StringVarP(&requestPromptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&requestWorkDir, "workdir", "w", "", "Working directory (defaults to current)")
	cmd.Flags().StringVar(&requestCacheTTL, "cache-ttl", "1h", "Cache TTL (e.g., 1h, 30m, 24h)")
	cmd.Flags().BoolVar(&requestNoCache, "no-cache", false, "Disable context caching")
	cmd.Flags().BoolVar(&requestRegenerateCtx, "regenerate", false, "Regenerate context before request")
	cmd.Flags().StringVarP(&requestOutputFile, "output", "o", "", "Write response to file instead of stdout")
	cmd.Flags().StringSliceVar(&requestContextFiles, "context", nil, "Additional context files to include")
	cmd.Flags().BoolVarP(&requestYes, "yes", "y", false, "Skip cache creation confirmation prompt")

	return cmd
}

func runRequest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate inputs
	if requestPrompt == "" && requestPromptFile == "" && len(args) == 0 {
		return fmt.Errorf("must provide prompt via -p, -f, or as argument")
	}

	// Get prompt text
	var promptText string
	if requestPrompt != "" {
		promptText = requestPrompt
	} else if requestPromptFile != "" {
		content, err := os.ReadFile(requestPromptFile)
		if err != nil {
			return fmt.Errorf("reading prompt file: %w", err)
		}
		promptText = string(content)
	} else if len(args) > 0 {
		promptText = strings.Join(args, " ")
	}

	// Determine working directory
	workDir := requestWorkDir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Make workDir absolute
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolving work directory: %w", err)
	}
	workDir = absWorkDir

	// Create pretty logger
	logger := pretty.New()
	
	logger.WorkingDirectory(workDir)

	// Check for .grove/rules file
	rulesPath := filepath.Join(workDir, ".grove", "rules")
	hasRules := false
	if _, err := os.Stat(rulesPath); err == nil {
		hasRules = true
		logger.FoundRulesFile(rulesPath)
		
		// Log the rules file content
		rulesContent, err := os.ReadFile(rulesPath)
		if err == nil {
			logger.RulesFileContent(strings.TrimSpace(string(rulesContent)))
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking rules file: %w", err)
	}

	// Initialize context manager
	var ctxMgr *grovecontext.Manager
	if hasRules {
		ctxMgr = grovecontext.NewManager(workDir)
		
		// Regenerate context if requested or if context files don't exist
		coldContextFile := filepath.Join(workDir, ".grove", "cached-context")
		hotContextFile := filepath.Join(workDir, ".grove", "context")
		
		needsRegeneration := requestRegenerateCtx
		if !needsRegeneration {
			// Check if context files exist
			if _, err := os.Stat(coldContextFile); os.IsNotExist(err) {
				needsRegeneration = true
				logger.Warning("Cold context not found, will regenerate")
			} else if _, err := os.Stat(hotContextFile); os.IsNotExist(err) {
				needsRegeneration = true
				logger.Warning("Hot context not found, will regenerate")
			}
		}

		if needsRegeneration {
			fmt.Fprintln(os.Stderr)
			logger.Info("🔄 Regenerating context from rules...")
			
			// Update context from rules
			if err := ctxMgr.UpdateFromRules(); err != nil {
				return fmt.Errorf("updating context from rules: %w", err)
			}

			// Generate context file
			if err := ctxMgr.GenerateContext(true); err != nil {
				return fmt.Errorf("generating context: %w", err)
			}

			// Display stats
			files, _ := ctxMgr.ReadFilesList(grovecontext.FilesListFile)
			stats, err := ctxMgr.GetStats("request", files, 10)
			if err == nil {
				fmt.Fprintln(os.Stderr)
				logger.Info("📊 Context Summary:")
				fmt.Fprintf(os.Stderr, "  Total files: %d\n", stats.TotalFiles)
				fmt.Fprintf(os.Stderr, "  Total tokens: %s\n", grovecontext.FormatTokenCount(stats.TotalTokens))
				fmt.Fprintf(os.Stderr, "  Total size: %s\n", grovecontext.FormatBytes(int(stats.TotalSize)))

				if stats.TotalTokens > 500000 {
					return fmt.Errorf("context size exceeds limit: %d tokens (max 500,000)", stats.TotalTokens)
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	} else {
		logger.Warning("No .grove/rules file found - context management disabled")
		logger.Tip("Create .grove/rules to enable automatic context inclusion")
		fmt.Fprintln(os.Stderr)
	}

	// Initialize Gemini client
	geminiClient, err := gemini.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("creating Gemini client: %w", err)
	}

	// Prepare context files
	coldContextFile := filepath.Join(workDir, ".grove", "cached-context")
	hotContextFile := filepath.Join(workDir, ".grove", "context")

	// Initialize cache manager
	cacheManager := gemini.NewCacheManager(workDir)

	// Parse cache TTL
	ttl := 1 * time.Hour
	if requestCacheTTL != "" {
		var err error
		ttl, err = time.ParseDuration(requestCacheTTL)
		if err != nil {
			return fmt.Errorf("parsing cache TTL: %w", err)
		}
	}

	// Get cache directives from context manager if available
	var ignoreChanges, disableExpiration, cacheDisabled bool
	if ctxMgr != nil {
		// Check for custom expiration time
		if customTTL, err := ctxMgr.GetExpireTime(); err == nil && customTTL > 0 {
			ttl = customTTL
			logger.TTL(ttl.String())
		}

		// Check for @freeze-cache directive
		if freeze, err := ctxMgr.ShouldFreezeCache(); err == nil && freeze {
			ignoreChanges = true
			logger.CacheFrozen()
		}

		// Check for @no-expire directive
		if noExpire, err := ctxMgr.ShouldDisableExpiration(); err == nil && noExpire {
			disableExpiration = true
			logger.Info("🚫 Cache expiration disabled by @no-expire directive")
		}
		
		// Check for @disable-cache directive
		if disabled, err := ctxMgr.ShouldDisableCache(); err == nil && disabled {
			cacheDisabled = true
		}
	}

	// Get or create cache for cold context (if it exists and caching is enabled)
	var cacheInfo *gemini.CacheInfo
	var isNewCache bool
	if !requestNoCache {
		if info, err := os.Stat(coldContextFile); err == nil && info.Size() > 0 {
			logger.Info(fmt.Sprintf("Cache settings: requestYes=%v, ignoreChanges=%v, disableExpiration=%v", requestYes, ignoreChanges, disableExpiration))
			cacheInfo, isNewCache, err = cacheManager.GetOrCreateCache(ctx, geminiClient, requestModel, coldContextFile, ttl, ignoreChanges, disableExpiration, requestYes)
			if err != nil {
				return fmt.Errorf("managing cache: %w", err)
			}
		} else if err == nil && info.Size() == 0 {
			logger.Warning("Cold context file is empty, skipping cache")
		} else if os.IsNotExist(err) && hasRules {
			logger.Warning("No cold context file found")
		}
	}

	// Prepare dynamic files
	var dynamicFiles []string
	
	// Add hot context if it exists
	if _, err := os.Stat(hotContextFile); err == nil {
		dynamicFiles = append(dynamicFiles, hotContextFile)
		logger.Info(fmt.Sprintf("Including hot context: %s", hotContextFile))
	}
	
	// If caching is disabled, also include cold context as dynamic file
	if cacheDisabled && cacheInfo == nil {
		if _, err := os.Stat(coldContextFile); err == nil {
			dynamicFiles = append(dynamicFiles, coldContextFile)
			logger.Info(fmt.Sprintf("Including cold context (cache disabled): %s", coldContextFile))
		}
	}

	// Add any additional context files
	for _, ctxFile := range requestContextFiles {
		absPath, err := filepath.Abs(ctxFile)
		if err != nil {
			return fmt.Errorf("resolving context file %s: %w", ctxFile, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("context file not found: %s", ctxFile)
		}
		dynamicFiles = append(dynamicFiles, absPath)
		logger.Info(fmt.Sprintf("Including additional context: %s", absPath))
	}

	// Also check for CLAUDE.md in the working directory
	claudePath := filepath.Join(workDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		dynamicFiles = append(dynamicFiles, claudePath)
		logger.Info(fmt.Sprintf("Including CLAUDE.md: %s", claudePath))
	}

	// Determine cache ID
	var cacheID string
	if cacheInfo != nil {
		cacheID = cacheInfo.CacheID
	}

	// Make the API request
	fmt.Fprintln(os.Stderr)
	logger.Model(requestModel)
	
	opts := &gemini.GenerateContentOptions{
		WorkingDir: workDir,
		Caller:     "gemapi-request",
		IsNewCache: isNewCache,
	}
	
	response, err := geminiClient.GenerateContentWithCacheAndOptions(ctx, requestModel, promptText, cacheID, dynamicFiles, opts)
	if err != nil {
		return fmt.Errorf("Gemini API request failed: %w", err)
	}

	// Output the response
	if requestOutputFile != "" {
		// Write to file
		if err := os.WriteFile(requestOutputFile, []byte(response), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintln(os.Stderr)
		logger.ResponseWritten(requestOutputFile)
	} else {
		// Write to stdout (not stderr) for piping
		fmt.Print(response)
		// Add newline if response doesn't end with one
		if !strings.HasSuffix(response, "\n") {
			fmt.Println()
		}
	}

	return nil
}

// Helper function to format file size
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}