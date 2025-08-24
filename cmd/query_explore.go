package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"github.com/mattsolo1/grove-gemini/pkg/config"
	"github.com/mattsolo1/grove-gemini/pkg/gcp"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var (
	exploreProjectID string
	exploreHours     int
	exploreLimit     int
)

func newQueryExploreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explore",
		Short: "Explore available logs for Gemini API",
		Long: `Explores Cloud Logging to find what logs are available for the Gemini API service.
This command helps discover the correct resource types, log names, and payload structures.`,
		RunE: runQueryExplore,
	}

	// Get default project from config
	defaultProject := config.GetDefaultProject("")

	cmd.Flags().StringVarP(&exploreProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().IntVarP(&exploreHours, "hours", "H", 1, "Number of hours to look back")
	cmd.Flags().IntVarP(&exploreLimit, "limit", "l", 10, "Maximum number of entries to examine")

	return cmd
}

func runQueryExplore(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Ensure we have a project ID
	if exploreProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	// Create logging client
	client, err := gcp.NewLoggingAdminClient(ctx, exploreProjectID)
	if err != nil {
		return fmt.Errorf("failed to create logging client: %w", err)
	}
	defer client.Close()

	startTime := time.Now().Add(-time.Duration(exploreHours) * time.Hour)

	fmt.Printf("Exploring logs for Gemini API in the last %d hour(s)...\n\n", exploreHours)

	// First, try to find any logs related to generativelanguage
	filters := []struct {
		name   string
		filter string
	}{
		{
			name: "Any logs with generativelanguage in text",
			filter: fmt.Sprintf(`
				"generativelanguage"
				timestamp>="%s"
			`, startTime.Format(time.RFC3339)),
		},
		{
			name: "Logs with generativelanguage service label",
			filter: fmt.Sprintf(`
				resource.labels.service="generativelanguage.googleapis.com"
				timestamp>="%s"
			`, startTime.Format(time.RFC3339)),
		},
		{
			name: "API logs for any Google AI service",
			filter: fmt.Sprintf(`
				resource.type="api"
				protoPayload.serviceName:"google.ai"
				timestamp>="%s"
			`, startTime.Format(time.RFC3339)),
		},
		{
			name: "Audit logs for generativelanguage",
			filter: fmt.Sprintf(`
				logName:"cloudaudit.googleapis.com"
				protoPayload.serviceName="generativelanguage.googleapis.com"
				timestamp>="%s"
			`, startTime.Format(time.RFC3339)),
		},
		{
			name: "Any consumed_api logs",
			filter: fmt.Sprintf(`
				resource.type="consumed_api"
				timestamp>="%s"
				severity>="INFO"
			`, startTime.Format(time.RFC3339)),
		},
	}

	foundLogs := false
	for _, f := range filters {
		fmt.Printf("=== Trying: %s ===\n", f.name)
		fmt.Printf("Filter: %s\n", strings.TrimSpace(f.filter))

		entries := client.Entries(ctx, logadmin.Filter(f.filter), logadmin.NewestFirst())
		
		count := 0
		for {
			entry, err := entries.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}

			if count == 0 {
				foundLogs = true
				fmt.Println("\nFound logs! Sample entry structure:")
				
				// Log name
				fmt.Printf("\nLog Name: %s\n", entry.LogName)
				
				// Resource
				if entry.Resource != nil {
					fmt.Printf("\nResource:\n")
					fmt.Printf("  Type: %s\n", entry.Resource.Type)
					fmt.Printf("  Labels:\n")
					for k, v := range entry.Resource.Labels {
						fmt.Printf("    %s: %s\n", k, v)
					}
				}
				
				// Severity
				fmt.Printf("\nSeverity: %s\n", entry.Severity)
				
				// Payload type
				fmt.Printf("\nPayload Type: %T\n", entry.Payload)
				
				// If it's a proto payload, show structure
				if payload, ok := entry.Payload.(map[string]interface{}); ok {
					fmt.Printf("\nPayload Structure:\n")
					printPayloadStructure(payload, "  ")
				}
			}

			count++
			if count >= exploreLimit {
				fmt.Printf("\n(Found %d+ entries, showing first %d)\n", count, exploreLimit)
				break
			}
		}

		if count == 0 {
			fmt.Println("No logs found with this filter.")
		} else {
			fmt.Printf("\nTotal found: %d entries\n", count)
		}
		fmt.Println()
	}

	if !foundLogs {
		fmt.Println("No Gemini API logs found. Possible reasons:")
		fmt.Println("1. Cloud Logging might not be enabled for the Gemini API")
		fmt.Println("2. Audit logs might need to be enabled in your project")
		fmt.Println("3. The API might log to a different service name")
		fmt.Println("\nTo enable audit logs:")
		fmt.Println("1. Go to IAM & Admin > Audit Logs in Cloud Console")
		fmt.Println("2. Find 'Generative Language API' or similar")
		fmt.Println("3. Enable the desired log types")
	}

	return nil
}

func printPayloadStructure(data map[string]interface{}, indent string) {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s: (object)\n", indent, key)
			if len(v) > 0 && len(indent) < 8 { // Limit depth
				// Show a few keys
				subIndent := indent + "  "
				count := 0
				for subKey := range v {
					fmt.Printf("%s%s\n", subIndent, subKey)
					count++
					if count >= 5 {
						fmt.Printf("%s... (%d more fields)\n", subIndent, len(v)-5)
						break
					}
				}
			}
		case []interface{}:
			fmt.Printf("%s%s: (array, length=%d)\n", indent, key, len(v))
		case string:
			if len(v) > 50 {
				fmt.Printf("%s%s: \"%s...\"\n", indent, key, v[:50])
			} else {
				fmt.Printf("%s%s: \"%s\"\n", indent, key, v)
			}
		default:
			fmt.Printf("%s%s: %v\n", indent, key, value)
		}
	}
}