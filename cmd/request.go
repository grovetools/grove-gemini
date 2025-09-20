package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
	requestRecache        bool
	requestUseCache       string
	requestOutputFile     string
	requestContextFiles   []string
	requestYes            bool
	// Generation parameters
	requestTemperature     float32
	requestTopP            float32
	requestTopK            int32
	requestMaxOutputTokens int32
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
  
  # Force recreation of cache
  gemapi request --recache -p "Review the codebase architecture"
  
  # Use a specific cache
  gemapi request --use-cache 53f364cda78e82a8 -p "Review using old context"
  
  # With custom working directory
  gemapi request -w /path/to/project -p "Analyze this project"`,
		RunE: runRequest,
	}

	cmd.Flags().StringVarP(&requestModel, "model", "m", "gemini-2.0-flash", "Gemini model to use")
	cmd.Flags().StringVarP(&requestPrompt, "prompt", "p", "", "Prompt text")
	cmd.Flags().StringVarP(&requestPromptFile, "file", "f", "", "Read prompt from file")
	cmd.Flags().StringVarP(&requestWorkDir, "workdir", "w", "", "Working directory (defaults to current)")
	cmd.Flags().StringVar(&requestCacheTTL, "cache-ttl", "5m", "Cache TTL (e.g., 1h, 30m, 24h)")
	cmd.Flags().BoolVar(&requestNoCache, "no-cache", false, "Disable context caching")
	cmd.Flags().BoolVar(&requestRegenerateCtx, "regenerate", false, "Regenerate context before request")
	cmd.Flags().BoolVar(&requestRecache, "recache", false, "Force recreation of the Gemini cache")
	cmd.Flags().StringVar(&requestUseCache, "use-cache", "", "Specify a cache name (short hash) to use for this request, bypassing automatic selection")
	cmd.Flags().StringVarP(&requestOutputFile, "output", "o", "", "Write response to file instead of stdout")
	cmd.Flags().StringSliceVar(&requestContextFiles, "context", nil, "Additional context files to include")
	cmd.Flags().BoolVarP(&requestYes, "yes", "y", false, "Skip cache creation confirmation prompt")
	
	// Generation parameters
	cmd.Flags().Float32Var(&requestTemperature, "temperature", -1, "Temperature for randomness (0.0-2.0, -1 to use default)")
	cmd.Flags().Float32Var(&requestTopP, "top-p", -1, "Top-p nucleus sampling (0.0-1.0, -1 to use default)")
	cmd.Flags().Int32Var(&requestTopK, "top-k", -1, "Top-k sampling (-1 to use default)")
	cmd.Flags().Int32Var(&requestMaxOutputTokens, "max-output-tokens", -1, "Maximum tokens in response (-1 to use default)")

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

	// Parse cache TTL
	ttl := 1 * time.Hour
	if requestCacheTTL != "" {
		var err error
		ttl, err = time.ParseDuration(requestCacheTTL)
		if err != nil {
			return fmt.Errorf("parsing cache TTL: %w", err)
		}
	}

	// Create prompt files slice
	var promptFiles []string
	if requestPromptFile != "" {
		promptFiles = []string{requestPromptFile}
	}

	// Create options
	options := gemini.RequestOptions{
		Model:            requestModel,
		Prompt:           promptText,
		PromptFiles:      promptFiles,
		WorkDir:          requestWorkDir,
		CacheTTL:         ttl,
		NoCache:          requestNoCache,
		RegenerateCtx:    requestRegenerateCtx,
		Recache:          requestRecache,
		UseCache:         requestUseCache,
		ContextFiles:     requestContextFiles,
		SkipConfirmation: requestYes,
	}
	
	// Add generation parameters if specified
	if cmd.Flags().Changed("temperature") {
		options.Temperature = &requestTemperature
	}
	if cmd.Flags().Changed("top-p") {
		options.TopP = &requestTopP
	}
	if cmd.Flags().Changed("top-k") {
		options.TopK = &requestTopK
	}
	if cmd.Flags().Changed("max-output-tokens") {
		options.MaxOutputTokens = &requestMaxOutputTokens
	}

	// Create and run request runner
	runner := gemini.NewRequestRunner()
	response, err := runner.Run(ctx, options)
	if err != nil {
		return err
	}

	// Output the response
	if requestOutputFile != "" {
		// Write to file
		if err := os.WriteFile(requestOutputFile, []byte(response), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintln(os.Stderr)
		logger := pretty.New()
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