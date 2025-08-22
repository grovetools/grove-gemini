package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattsolo1/grove-gemini/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	localHours  int
	localLimit  int
	localModel  string
	localErrors bool
)

func newQueryLocalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Query local Gemini API logs",
		Long:  `Displays locally logged Gemini API requests with token usage, costs, and performance metrics.`,
		RunE:  runQueryLocal,
	}

	cmd.Flags().IntVarP(&localHours, "hours", "H", 24, "Number of hours to look back")
	cmd.Flags().IntVarP(&localLimit, "limit", "l", 100, "Maximum number of requests to display")
	cmd.Flags().StringVarP(&localModel, "model", "m", "", "Filter by model name")
	cmd.Flags().BoolVar(&localErrors, "errors", false, "Show only failed requests")

	return cmd
}

func runQueryLocal(cmd *cobra.Command, args []string) error {
	logger := logging.GetLogger()
	
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(localHours) * time.Hour)
	
	fmt.Printf("Fetching local Gemini API logs for the last %d hour(s)...\n\n", localHours)
	
	logs, err := logger.ReadLogs(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}
	
	if len(logs) == 0 {
		fmt.Println("No logs found for the specified time range.")
		return nil
	}
	
	// Filter logs
	var filteredLogs []logging.QueryLog
	for _, log := range logs {
		// Filter by model if specified
		if localModel != "" && !strings.Contains(strings.ToLower(log.Model), strings.ToLower(localModel)) {
			continue
		}
		
		// Filter by errors if specified
		if localErrors && log.Success {
			continue
		}
		
		filteredLogs = append(filteredLogs, log)
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(filteredLogs, func(i, j int) bool {
		return filteredLogs[i].Timestamp.After(filteredLogs[j].Timestamp)
	})
	
	// Limit results
	if len(filteredLogs) > localLimit {
		filteredLogs = filteredLogs[:localLimit]
	}
	
	// Display table
	displayLocalLogsTable(filteredLogs)
	
	// Summary
	if len(filteredLogs) > 10 {
		displaySummary(filteredLogs)
	}
	
	return nil
}

func displayLocalLogsTable(logs []logging.QueryLog) {
	// Header
	fmt.Printf("%-19s %-15s %-25s %-15s %7s %7s %7s %7s %6s %10s %6s %s\n",
		"Timestamp", "Model", "Repo/Branch", "Caller", "Cached", "Prompt", "Compl", "Total", "Cache%", "Cost", "Time", "Status")
	fmt.Println(strings.Repeat("-", 160))
	
	// Rows
	for _, log := range logs {
		timestamp := log.Timestamp.Format("01-02 15:04:05")
		
		// Shorten model name
		model := log.Model
		if len(model) > 15 {
			parts := strings.Split(model, "-")
			if len(parts) >= 3 {
				model = parts[1] + "-" + parts[2] // e.g., "2.0-flash"
			}
		}
		
		
		cachedStr := "-"
		if log.CachedTokens > 0 {
			cachedStr = fmt.Sprintf("%d", log.CachedTokens)
		}
		
		promptStr := fmt.Sprintf("%d", log.PromptTokens)
		completionStr := fmt.Sprintf("%d", log.CompletionTokens)
		totalStr := fmt.Sprintf("%d", log.TotalTokens)
		
		cacheRateStr := "-"
		if log.CacheHitRate > 0 {
			cacheRateStr = fmt.Sprintf("%.1f%%", log.CacheHitRate*100)
		}
		
		costStr := fmt.Sprintf("$%.6f", log.EstimatedCost)
		timeStr := fmt.Sprintf("%.2fs", log.ResponseTime)
		
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
			if len(repoName) > 12 {
				repoName = repoName[:10] + ".."
			}
			
			branch := log.GitBranch
			if len(branch) > 10 {
				branch = branch[:8] + ".."
			}
			
			repoInfo = fmt.Sprintf("%s/%s", repoName, branch)
		}
		
		caller := log.Caller
		if caller == "" {
			caller = "-"
		} else if len(caller) > 15 {
			caller = caller[:13] + ".."
		}
		
		statusStr := "✓"
		if !log.Success {
			statusStr = "✗"
			if log.Error != "" && len(log.Error) > 20 {
				statusStr = "✗ " + log.Error[:17] + "..."
			}
		}
		
		fmt.Printf("%-19s %-15s %-25s %-15s %7s %7s %7s %7s %6s %10s %6s %s\n",
			timestamp, model, repoInfo, caller, cachedStr, promptStr, completionStr, totalStr, cacheRateStr, costStr, timeStr, statusStr)
	}
}

func displaySummary(logs []logging.QueryLog) {
	fmt.Printf("\n=== Summary (showing %d requests) ===\n", len(logs))
	
	var totalCost float64
	var totalPromptTokens, totalCompletionTokens, totalCachedTokens int64
	var totalResponseTime float64
	var errorCount int
	var cacheHits int
	
	modelCosts := make(map[string]float64)
	modelCounts := make(map[string]int)
	
	for _, log := range logs {
		totalCost += log.EstimatedCost
		totalPromptTokens += int64(log.PromptTokens)
		totalCompletionTokens += int64(log.CompletionTokens)
		totalCachedTokens += int64(log.CachedTokens)
		totalResponseTime += log.ResponseTime
		
		if !log.Success {
			errorCount++
		}
		if log.CachedTokens > 0 {
			cacheHits++
		}
		
		// Group by model
		modelKey := log.Model
		if strings.Contains(modelKey, "flash") {
			modelKey = "flash"
		} else if strings.Contains(modelKey, "pro") {
			modelKey = "pro"
		}
		modelCosts[modelKey] += log.EstimatedCost
		modelCounts[modelKey]++
	}
	
	fmt.Printf("Total Cost: $%.6f\n", totalCost)
	fmt.Printf("Total Tokens: %d (Prompt: %d, Completion: %d, Cached: %d)\n", 
		totalPromptTokens+totalCompletionTokens, totalPromptTokens, totalCompletionTokens, totalCachedTokens)
	
	if errorCount > 0 {
		fmt.Printf("Error Rate: %.1f%% (%d errors)\n", float64(errorCount)/float64(len(logs))*100, errorCount)
	}
	
	if cacheHits > 0 {
		fmt.Printf("Cache Hit Rate: %.1f%% (%d requests with cache)\n", float64(cacheHits)/float64(len(logs))*100, cacheHits)
		
		// Calculate cache savings
		avgCacheRate := float64(totalCachedTokens) / float64(totalPromptTokens+totalCachedTokens)
		savedTokens := float64(totalCachedTokens) * 0.75 // 75% discount on cached tokens
		savedCost := savedTokens / 1_000_000 * 0.075 // Assuming flash input pricing
		fmt.Printf("Cache Savings: ~$%.6f (%.1f%% avg cache rate)\n", savedCost, avgCacheRate*100)
	}
	
	fmt.Printf("Average Response Time: %.2fs\n", totalResponseTime/float64(len(logs)))
	
	// Cost breakdown by model
	if len(modelCosts) > 1 {
		fmt.Println("\nCost by Model:")
		for model, cost := range modelCosts {
			fmt.Printf("  %s: $%.6f (%d requests)\n", model, cost, modelCounts[model])
		}
	}
	
	// Hourly rate
	hourlyRate := totalCost / float64(localHours) 
	dailyProjection := hourlyRate * 24
	monthlyProjection := dailyProjection * 30
	fmt.Printf("\nProjected Costs:")
	fmt.Printf("  Hourly: $%.6f\n", hourlyRate)
	fmt.Printf("  Daily: $%.2f\n", dailyProjection)
	fmt.Printf("  Monthly: $%.2f\n", monthlyProjection)
}