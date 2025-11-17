package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-gemini/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	requestsHours  int
	requestsLimit  int
	requestsModel  string
	requestsErrors bool
)

func newQueryRequestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "Query individual Gemini API requests from local logs",
		Long:  `Displays a table of individual Gemini API requests with details like timestamp, method, tokens, latency, and status.

This command reads from local logs since Google doesn't publish individual Gemini API requests to Cloud Logging.`,
		RunE:  runQueryRequests,
	}

	cmd.Flags().IntVarP(&requestsHours, "hours", "H", 1, "Number of hours to look back")
	cmd.Flags().IntVarP(&requestsLimit, "limit", "l", 100, "Maximum number of requests to display")
	cmd.Flags().StringVarP(&requestsModel, "model", "m", "", "Filter by model name")
	cmd.Flags().BoolVar(&requestsErrors, "errors", false, "Show only failed requests")

	return cmd
}

func runQueryRequests(cmd *cobra.Command, args []string) error {
	logger := logging.GetLogger()
	
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(requestsHours) * time.Hour)
	
	fmt.Printf("Fetching Gemini API requests for the last %d hour(s)...\n\n", requestsHours)
	
	logs, err := logger.ReadLogs(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}
	
	if len(logs) == 0 {
		fmt.Println("No requests found for the specified time range.")
		fmt.Println("\nNote: This command reads from local logs. Make sure you have made some Gemini API calls.")
		return nil
	}
	
	// Filter logs
	var filteredLogs []logging.QueryLog
	for _, log := range logs {
		// Filter by model if specified
		if requestsModel != "" && !strings.Contains(strings.ToLower(log.Model), strings.ToLower(requestsModel)) {
			continue
		}
		
		// Filter by errors if specified
		if requestsErrors && log.Success {
			continue
		}
		
		filteredLogs = append(filteredLogs, log)
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(filteredLogs, func(i, j int) bool {
		return filteredLogs[i].Timestamp.After(filteredLogs[j].Timestamp)
	})
	
	// Limit results
	if len(filteredLogs) > requestsLimit {
		filteredLogs = filteredLogs[:requestsLimit]
	}
	
	// Display table
	displayRequestsTable(filteredLogs)
	
	return nil
}

func displayRequestsTable(logs []logging.QueryLog) {
	// Header
	fmt.Printf("%-20s %-15s %-8s %-10s %-10s %-10s %-8s %-10s %-30s %-15s\n",
		"Timestamp", "Model", "Method", "Prompt", "Completion", "Total", "Latency", "Cost", "Repository/Branch", "Caller")
	fmt.Println(strings.Repeat("-", 170))
	
	// Rows
	for _, log := range logs {
		timestamp := log.Timestamp.Format("01-02 15:04:05.000")
		
		// Shorten model name
		model := log.Model
		if len(model) > 15 {
			parts := strings.Split(model, "-")
			if len(parts) > 2 {
				model = parts[1] + "-" + parts[2] // e.g., "2.0-flash"
			}
		}
		
		method := log.Method
		if method == "" {
			method = "Generate"
		}
		if len(method) > 8 {
			method = method[:8]
		}
		
		status := theme.IconSuccess
		if !log.Success {
			status = theme.IconError
			if log.Error != "" && len(log.Error) > 20 {
				status = theme.IconError + " " + log.Error[:17] + "..."
			}
		}
		
		// Format repo/branch info
		repoInfo := "-"
		if log.GitRepo != "" {
			// Extract just the repo name from github.com/user/repo
			parts := strings.Split(log.GitRepo, "/")
			repoName := ""
			if len(parts) >= 2 {
				repoName = parts[len(parts)-1]
			} else {
				repoName = log.GitRepo
			}
			if len(repoName) > 20 {
				repoName = repoName[:18] + ".."
			}
			
			branch := log.GitBranch
			if len(branch) > 8 {
				branch = branch[:6] + ".."
			}
			
			repoInfo = fmt.Sprintf("%s/%s", repoName, branch)
		}
		
		caller := log.Caller
		if caller == "" {
			caller = "-"
		} else if len(caller) > 15 {
			caller = caller[:13] + ".."
		}
		
		fmt.Printf("%-20s %-15s %-8s %-10d %-10d %-10d %-8.2fs %-10s %-30s %-15s %s\n",
			timestamp, 
			model, 
			method,
			log.PromptTokens,
			log.CompletionTokens,
			log.TotalTokens,
			log.ResponseTime,
			fmt.Sprintf("$%.6f", log.EstimatedCost),
			repoInfo,
			caller,
			status)
	}
	
	fmt.Printf("\nShowing %d request(s)\n", len(logs))
}