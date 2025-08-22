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

	fmt.Fprintf(os.Stderr, "🏠 Working directory: %s\n", workDir)

	// Check for .grove/rules file
	rulesPath := filepath.Join(workDir, ".grove", "rules")
	hasRules := false
	if _, err := os.Stat(rulesPath); err == nil {
		hasRules = true
		fmt.Fprintf(os.Stderr, "📋 Found rules file: %s\n", rulesPath)
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
				fmt.Fprintf(os.Stderr, "⚠️  Cold context not found, will regenerate\n")
			} else if _, err := os.Stat(hotContextFile); os.IsNotExist(err) {
				needsRegeneration = true
				fmt.Fprintf(os.Stderr, "⚠️  Hot context not found, will regenerate\n")
			}
		}

		if needsRegeneration {
			fmt.Fprintf(os.Stderr, "\n🔄 Regenerating context from rules...\n")
			
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
				fmt.Fprintf(os.Stderr, "\n📊 Context Summary:\n")
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
		fmt.Fprintf(os.Stderr, "⚠️  No .grove/rules file found - context management disabled\n")
		fmt.Fprintf(os.Stderr, "💡 Create .grove/rules to enable automatic context inclusion\n\n")
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
	var ignoreChanges, disableExpiration bool
	if ctxMgr != nil {
		// Check for custom expiration time
		if customTTL, err := ctxMgr.GetExpireTime(); err == nil && customTTL > 0 {
			ttl = customTTL
			fmt.Fprintf(os.Stderr, "⏱️  Using cache TTL from @expire-time directive: %s\n", ttl)
		}

		// Check for @freeze-cache directive
		if freeze, err := ctxMgr.ShouldFreezeCache(); err == nil && freeze {
			ignoreChanges = true
			fmt.Fprintf(os.Stderr, "❄️  Cache is frozen by @freeze-cache directive\n")
		}

		// Check for @no-expire directive
		if noExpire, err := ctxMgr.ShouldDisableExpiration(); err == nil && noExpire {
			disableExpiration = true
			fmt.Fprintf(os.Stderr, "🚫 Cache expiration disabled by @no-expire directive\n")
		}
	}

	// Get or create cache for cold context (if it exists and caching is enabled)
	var cacheInfo *gemini.CacheInfo
	if !requestNoCache {
		if info, err := os.Stat(coldContextFile); err == nil && info.Size() > 0 {
			cacheInfo, err = cacheManager.GetOrCreateCache(ctx, geminiClient, requestModel, coldContextFile, ttl, ignoreChanges, disableExpiration)
			if err != nil {
				return fmt.Errorf("managing cache: %w", err)
			}
		} else if err == nil && info.Size() == 0 {
			fmt.Fprintf(os.Stderr, "⚠️  Cold context file is empty, skipping cache\n")
		} else if os.IsNotExist(err) && hasRules {
			fmt.Fprintf(os.Stderr, "⚠️  No cold context file found\n")
		}
	}

	// Prepare dynamic files
	var dynamicFiles []string
	
	// Add hot context if it exists
	if _, err := os.Stat(hotContextFile); err == nil {
		dynamicFiles = append(dynamicFiles, hotContextFile)
		fmt.Fprintf(os.Stderr, "📁 Including hot context: %s\n", hotContextFile)
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
		fmt.Fprintf(os.Stderr, "📁 Including additional context: %s\n", absPath)
	}

	// Also check for CLAUDE.md in the working directory
	claudePath := filepath.Join(workDir, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		dynamicFiles = append(dynamicFiles, claudePath)
		fmt.Fprintf(os.Stderr, "📁 Including CLAUDE.md: %s\n", claudePath)
	}

	// Determine cache ID
	var cacheID string
	if cacheInfo != nil {
		cacheID = cacheInfo.CacheID
	}

	// Make the API request
	fmt.Fprintf(os.Stderr, "\n🤖 Calling Gemini API with model: %s\n", requestModel)
	
	opts := &gemini.GenerateContentOptions{
		WorkingDir: workDir,
		Caller:     "gemapi-request",
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
		fmt.Fprintf(os.Stderr, "\n✅ Response written to: %s\n", requestOutputFile)
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