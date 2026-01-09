package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattsolo1/grove-gemini/pkg/gemini"
	"github.com/spf13/cobra"
	"google.golang.org/genai"
)

var (
	countTokensModel string
)

func newCountTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "count-tokens [text...]",
		Short: "Count tokens for a given text using Gemini API",
		Long: `Count the number of tokens in a piece of text using the Gemini API.

You can provide text in three ways:
1. As command line arguments: gemapi count-tokens "Your text here"
2. Via standard input: echo "Your text" | gemapi count-tokens
3. From a file: cat file.txt | gemapi count-tokens

This is useful for:
- Checking if your prompt fits within model limits
- Estimating costs before making API calls
- Understanding token usage for different types of content`,
		RunE: runCountTokens,
	}

	cmd.Flags().StringVarP(&countTokensModel, "model", "m", "gemini-1.5-flash-latest", "Model to use for token counting")

	return cmd
}

func runCountTokens(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get text to count
	var text string
	if len(args) > 0 {
		// Text provided as command line arguments
		text = strings.Join(args, " ")
	} else {
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder
		
		// Check if stdin is available
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// No pipe input
			ctx := context.Background()
			ulog.Info("Usage information").
				Pretty("No text provided. Use command line arguments or pipe text via stdin.\nExamples:\n  gemapi count-tokens \"Your text here\"\n  echo \"Your text\" | gemapi count-tokens\n  cat file.txt | gemapi count-tokens").
				PrettyOnly().
				Log(ctx)
			return fmt.Errorf("no input text provided")
		}

		// Read all input
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					builder.WriteString(line)
					break
				}
				return fmt.Errorf("error reading input: %w", err)
			}
			builder.WriteString(line)
		}
		text = builder.String()
	}

	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("no text provided to count")
	}

	// Create client
	client, err := gemini.NewClient(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Get the underlying genai client
	genaiClient := client.GetClient()

	// Count tokens
	ulog.Info("Counting tokens").
		Field("model", countTokensModel).
		Pretty(fmt.Sprintf("Counting tokens using model: %s", countTokensModel)).
		PrettyOnly().
		Log(ctx)

	tokenResp, err := genaiClient.Models.CountTokens(ctx,
		countTokensModel,
		[]*genai.Content{{Parts: []*genai.Part{{Text: text}}}},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to count tokens: %w", err)
	}

	// Display results
	var output strings.Builder
	output.WriteString("=== Token Count ===\n")
	output.WriteString(fmt.Sprintf("Model: %s\n", countTokensModel))
	output.WriteString(fmt.Sprintf("Total Tokens: %d\n", tokenResp.TotalTokens))

	// Calculate estimated costs based on current Gemini pricing
	// These are prompt token prices
	var pricePerMillion float64
	modelLower := strings.ToLower(countTokensModel)
	switch {
	case strings.Contains(modelLower, "gemini-2.5-pro"):
		pricePerMillion = 1.25 // $1.25 per million input tokens (<=200k)
	case strings.Contains(modelLower, "gemini-2.5-flash") && strings.Contains(modelLower, "lite"):
		pricePerMillion = 0.10 // $0.10 per million input tokens
	case strings.Contains(modelLower, "gemini-2.5-flash"):
		pricePerMillion = 0.30 // $0.30 per million input tokens
	case strings.Contains(modelLower, "gemini-2.0-flash") && strings.Contains(modelLower, "lite"):
		pricePerMillion = 0.075 // $0.075 per million input tokens
	case strings.Contains(modelLower, "gemini-2.0-flash"):
		pricePerMillion = 0.10 // $0.10 per million input tokens
	default:
		pricePerMillion = 0.10 // Default to 2.0 flash pricing
	}

	estimatedCost := float64(tokenResp.TotalTokens) / 1_000_000 * pricePerMillion
	output.WriteString(fmt.Sprintf("\nEstimated Input Cost: $%.6f\n", estimatedCost))

	// Show text preview if not too long
	if len(text) <= 200 {
		output.WriteString(fmt.Sprintf("\nText: %q\n", text))
	} else {
		output.WriteString(fmt.Sprintf("\nText Preview: %q...\n", text[:200]))
		output.WriteString(fmt.Sprintf("(Total length: %d characters)\n", len(text)))
	}

	// Model limits information
	output.WriteString("\n=== Model Context Information ===\n")
	switch {
	case strings.Contains(countTokensModel, "flash"):
		output.WriteString("Context Window: 1,048,576 tokens\n")
		output.WriteString(fmt.Sprintf("Usage: %.2f%% of context window\n", float64(tokenResp.TotalTokens)/1_048_576*100))
	case strings.Contains(countTokensModel, "pro"):
		output.WriteString("Context Window: 2,097,152 tokens\n")
		output.WriteString(fmt.Sprintf("Usage: %.2f%% of context window\n", float64(tokenResp.TotalTokens)/2_097_152*100))
	default:
		output.WriteString("Context Window: Model-specific (check documentation)\n")
	}

	ulog.Info("Token count results").
		Field("model", countTokensModel).
		Field("total_tokens", tokenResp.TotalTokens).
		Field("estimated_cost", estimatedCost).
		Field("text_length", len(text)).
		Pretty(output.String()).
		PrettyOnly().
		Log(ctx)

	return nil
}