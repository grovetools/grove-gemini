package cmd

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"github.com/grovetools/grove-gemini/pkg/config"
	"github.com/grovetools/grove-gemini/pkg/gcp"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var (
	tokensProjectID string
	tokensHours     int
	tokensDebug     bool
)

type TokenUsage struct {
	Timestamp        time.Time
	Method           string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheHit         bool
	Latency          float64
}

func newQueryTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Query detailed token usage from Cloud Logging",
		Long:  `Fetches and displays detailed Gemini API token usage information including prompt tokens, completion tokens, cache hits, and estimated costs from Google Cloud Logging.`,
		RunE:  runQueryTokens,
	}

	// Get default project from config
	defaultProject := config.GetDefaultProject("")

	cmd.Flags().StringVarP(&tokensProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().IntVarP(&tokensHours, "hours", "H", 24, "Number of hours to look back")
	cmd.Flags().BoolVar(&tokensDebug, "debug", false, "Enable debug output")

	return cmd
}

func runQueryTokens(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Ensure we have a project ID
	if tokensProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	// Create logging client
	client, err := gcp.NewLoggingAdminClient(ctx, tokensProjectID)
	if err != nil {
		return fmt.Errorf("failed to create logging client: %w", err)
	}
	defer client.Close()

	// Build filter - include all the v1beta endpoints
	startTime := time.Now().Add(-time.Duration(tokensHours) * time.Hour)
	
	// Try different filter approaches
	filters := []string{
		// Primary filter with all methods
		fmt.Sprintf(`
			resource.type="api"
			resource.labels.service="generativelanguage.googleapis.com"
			timestamp>="%s"
			(protoPayload.methodName="google.ai.generativelanguage.v1beta.GenerativeService.GenerateContent" OR
			 protoPayload.methodName="google.ai.generativelanguage.v1beta.GenerativeService.StreamGenerateContent" OR
			 protoPayload.methodName="google.ai.generativelanguage.v1beta.CacheService.CreateCachedContent" OR
			 protoPayload.methodName="google.ai.generativelanguage.v1beta.FileService.CreateFile" OR
			 protoPayload.methodName="google.ai.generativelanguage.v1beta.FileService.GetFile" OR
			 protoPayload.methodName="google.ai.generativelanguage.v1beta.FileService.DeleteFile")
		`, startTime.Format(time.RFC3339)),
		// Alternative: Try without resource type
		fmt.Sprintf(`
			resource.labels.service="generativelanguage.googleapis.com"
			timestamp>="%s"
			protoPayload.methodName:"google.ai.generativelanguage.v1beta"
		`, startTime.Format(time.RFC3339)),
		// Alternative: Try consumed_api resource type
		fmt.Sprintf(`
			resource.type="consumed_api"
			resource.labels.service="generativelanguage.googleapis.com"
			timestamp>="%s"
		`, startTime.Format(time.RFC3339)),
		// Alternative: Try audited_resource
		fmt.Sprintf(`
			resource.type="audited_resource"
			resource.labels.service="generativelanguage.googleapis.com"
			timestamp>="%s"
		`, startTime.Format(time.RFC3339)),
	}

	fmt.Printf("Fetching token usage logs for the last %d hours...\n\n", tokensHours)

	var tokenUsages []TokenUsage
	var successfulFilter bool

	for i, filter := range filters {
		if tokensDebug {
			fmt.Printf("[DEBUG] Trying filter %d:\n%s\n", i+1, filter)
		}

		entries := client.Entries(ctx, logadmin.Filter(filter))
		
		entryCount := 0
		for {
			entry, err := entries.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if tokensDebug {
					fmt.Printf("[DEBUG] Error with filter %d: %v\n", i+1, err)
				}
				break
			}

			entryCount++
			if tokensDebug && entryCount == 1 {
				fmt.Printf("[DEBUG] Found entries with filter %d\n", i+1)
				fmt.Printf("[DEBUG] Sample entry payload type: %T\n", entry.Payload)
			}

			// Parse the payload
			if payload, ok := entry.Payload.(map[string]interface{}); ok {
				usage := TokenUsage{
					Timestamp: entry.Timestamp,
				}
				
				// Extract method name
				if protoPayload, ok := payload["protoPayload"].(map[string]interface{}); ok {
					if methodName, ok := protoPayload["methodName"].(string); ok {
						usage.Method = methodName
					}
					
					// Extract response data
					if response, ok := protoPayload["response"].(map[string]interface{}); ok {
						if promptTokens, ok := getFloat64(response, "promptTokenCount"); ok {
							usage.PromptTokens = int64(promptTokens)
						}
						if completionTokens, ok := getFloat64(response, "candidatesTokenCount"); ok {
							usage.CompletionTokens = int64(completionTokens)
						}
						if totalTokens, ok := getFloat64(response, "totalTokenCount"); ok {
							usage.TotalTokens = int64(totalTokens)
						}
						if cacheHit, ok := response["cacheHitMetadata"].(map[string]interface{}); ok && len(cacheHit) > 0 {
							usage.CacheHit = true
						}
					}
					
					// Extract latency
					if latency, ok := getFloat64(protoPayload, "latency"); ok {
						usage.Latency = latency
					}
				}
				
				// Only add if we have token data
				if usage.TotalTokens > 0 {
					tokenUsages = append(tokenUsages, usage)
					successfulFilter = true
				}
			}
		}

		if successfulFilter {
			if tokensDebug {
				fmt.Printf("[DEBUG] Successfully found %d entries with token data using filter %d\n", len(tokenUsages), i+1)
			}
			break
		}
	}

	if len(tokenUsages) == 0 {
		fmt.Println("No token usage data found for the specified time range.")
		if !tokensDebug {
			fmt.Println("\nTry running with --debug flag for more information.")
			fmt.Println("\nPossible reasons:")
			fmt.Println("- Cloud Logging might not be enabled for the Gemini API")
			fmt.Println("- The logs might have a different structure than expected")
			fmt.Println("- No API calls were made in the specified time range")
		}
		return nil
	}

	// Display summary
	printTokenSummary(tokenUsages)

	return nil
}

func printTokenSummary(usages []TokenUsage) {
	var totalPrompt, totalCompletion, totalTokens int64
	var cacheHits int
	methodCounts := make(map[string]int)
	
	for _, u := range usages {
		totalPrompt += u.PromptTokens
		totalCompletion += u.CompletionTokens
		totalTokens += u.TotalTokens
		if u.CacheHit {
			cacheHits++
		}
		methodCounts[u.Method]++
	}
	
	fmt.Println("=== Token Usage Summary ===")
	fmt.Printf("Total Requests: %d\n", len(usages))
	fmt.Printf("Total Prompt Tokens: %d\n", totalPrompt)
	fmt.Printf("Total Completion Tokens: %d\n", totalCompletion)
	fmt.Printf("Total Tokens: %d\n", totalTokens)
	
	if len(usages) > 0 {
		cacheHitRate := float64(cacheHits) / float64(len(usages)) * 100
		fmt.Printf("Cache Hit Rate: %.2f%% (%d/%d)\n", cacheHitRate, cacheHits, len(usages))
		
		// Method breakdown
		fmt.Println("\nBreakdown by Method:")
		for method, count := range methodCounts {
			fmt.Printf("  %s: %d requests\n", method, count)
		}
		
		// Estimated costs (using Gemini 1.5 Flash pricing as default)
		const (
			pricePerKInput  = 0.075 / 1000   // $0.075 per million tokens
			pricePerKOutput = 0.30 / 1000    // $0.30 per million tokens
		)
		
		inputCost := float64(totalPrompt) / 1000 * pricePerKInput
		outputCost := float64(totalCompletion) / 1000 * pricePerKOutput
		
		fmt.Printf("\n=== Estimated Costs (Gemini 1.5 Flash) ===\n")
		fmt.Printf("Input: $%.6f\n", inputCost)
		fmt.Printf("Output: $%.6f\n", outputCost)
		fmt.Printf("Total: $%.6f\n", inputCost+outputCost)
		
		// Per-request averages
		avgPrompt := float64(totalPrompt) / float64(len(usages))
		avgCompletion := float64(totalCompletion) / float64(len(usages))
		avgTotal := float64(totalTokens) / float64(len(usages))
		
		fmt.Printf("\n=== Per-Request Averages ===\n")
		fmt.Printf("Avg Prompt Tokens: %.0f\n", avgPrompt)
		fmt.Printf("Avg Completion Tokens: %.0f\n", avgCompletion)
		fmt.Printf("Avg Total Tokens: %.0f\n", avgTotal)
		
		// Latency statistics
		var totalLatency float64
		var latencyCount int
		for _, u := range usages {
			if u.Latency > 0 {
				totalLatency += u.Latency
				latencyCount++
			}
		}
		if latencyCount > 0 {
			avgLatency := totalLatency / float64(latencyCount)
			fmt.Printf("Avg Latency: %.2fs\n", avgLatency)
		}
	}
}

// Helper function to safely extract float64 values from interface{}
func getFloat64(m map[string]interface{}, key string) (float64, bool) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, true
		case int64:
			return float64(v), true
		case int:
			return float64(v), true
		}
	}
	return 0, false
}