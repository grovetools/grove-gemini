package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grovetools/grove-gemini/pkg/gemini"
	"github.com/spf13/cobra"
)

var (
	embedModel    string
	embedTaskType string
)

func newEmbedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed [text...]",
		Short: "Generate embeddings for text using Gemini API",
		Long: `Generate vector embeddings for text using a Gemini embedding model.

You can provide text in three ways:
1. As command line arguments: grove-gemini embed "Your text here"
2. Via standard input: echo "Your text" | grove-gemini embed
3. From a file: cat file.txt | grove-gemini embed

Task types control how embeddings are optimized:
- RETRIEVAL_DOCUMENT: For documents being indexed (default)
- RETRIEVAL_QUERY: For search queries at retrieval time

Output is a JSON array of floats to stdout.`,
		RunE: runEmbed,
	}

	cmd.Flags().StringVarP(&embedModel, "model", "m", gemini.DefaultEmbeddingModel, "Embedding model to use")
	cmd.Flags().StringVarP(&embedTaskType, "task-type", "t", "RETRIEVAL_DOCUMENT", "Task type (RETRIEVAL_DOCUMENT, RETRIEVAL_QUERY)")

	return cmd
}

func runEmbed(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get text to embed
	var text string
	if len(args) > 0 {
		text = strings.Join(args, " ")
	} else {
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder

		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no input text provided. Use arguments or pipe text via stdin")
		}

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
		return fmt.Errorf("no text provided to embed")
	}

	// Create client
	client, err := gemini.NewClient(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Generate embedding
	ulog.Info("Generating embedding").
		Field("model", embedModel).
		Field("task_type", embedTaskType).
		Log(ctx)

	embedding, err := client.EmbedText(ctx, embedModel, text, embedTaskType)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Output as JSON array to stdout
	output, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	fmt.Println(string(output))

	ulog.Info("Embedding generated").
		Field("model", embedModel).
		Field("dimensions", len(embedding)).
		Field("text_length", len(text)).
		Log(ctx)

	return nil
}
