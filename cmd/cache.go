package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tablecomponent "github.com/grovetools/core/tui/components/table"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/grove-gemini/pkg/gemini"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local cache of Gemini contexts",
		Long:  `Provides commands to manage the local cache of Gemini API context data. You can list, inspect, clear, and prune cached items. Use 'gemapi cache tui' to launch the interactive interface.`,
	}

	// Add an explicit 'tui' command
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive cache management TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCacheTUI()
		},
	}
	cmd.AddCommand(tuiCmd)
	cmd.AddCommand(newCacheListCmd())
	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePruneCmd())
	cmd.AddCommand(newCacheInspectCmd())

	return cmd
}

func newCacheListCmd() *cobra.Command {
	var localOnly, apiOnly bool
	
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all caches with both local and API status",
		Long: `List cached contents showing both local storage and Google API status.
By default, shows a combined view of local cache files and their status on Google's servers.
Use --local-only or --api-only to filter the view.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if localOnly && apiOnly {
				return fmt.Errorf("cannot use both --local-only and --api-only flags")
			}
			
			if apiOnly {
				return listCachesFromAPI()
			}
			
			if localOnly {
				return listLocalCachesOnly()
			}
			
			// Default: show combined view
			return listCachesCombined()
		},
	}
	
	cmd.Flags().BoolVar(&localOnly, "local-only", false, "Show only local cache information")
	cmd.Flags().BoolVar(&apiOnly, "api-only", false, "Show only caches from Google's API servers")
	
	return cmd
}

func newCacheClearCmd() *cobra.Command {
	var withLocal, preserveLocal bool
	
	cmd := &cobra.Command{
		Use:   "clear [cache-name...] | --all",
		Short: "Clear caches from Google's servers (default: remote-only)",
		Long: `Clears caches from Google's servers and updates local tracking.
By default, only clears the remote cache and marks the local file as cleared.
Use --with-local to also remove the local cache file.
Use --preserve-local to skip updating the local cache file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			if !all && len(args) == 0 {
				return fmt.Errorf("must specify a cache name to clear, or use the --all flag")
			}
			
			if withLocal && preserveLocal {
				return fmt.Errorf("cannot use both --with-local and --preserve-local flags")
			}
			
			ctx := context.Background()
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
			
			// Always create client since we default to clearing remote
			client, err := gemini.NewClient(ctx, "")
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}
			
			if all {
				files, err := os.ReadDir(cacheDir)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Println("No cache directory found. Nothing to clear.")
						return nil
					}
					return fmt.Errorf("reading cache directory: %w", err)
				}
				
				clearedCount := 0
				apiDeletedCount := 0
				for _, file := range files {
					if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
						path := filepath.Join(cacheDir, file.Name())
						
						// Load cache info to get the cache ID
						info, err := gemini.LoadCacheInfo(path)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Warning: could not read cache info for %s: %v\n", file.Name(), err)
							continue
						}
						
						// Skip if already cleared
						if info.ClearedAt != nil {
							continue
						}
						
						// Delete from API
						if client != nil {
							if err := client.DeleteCache(ctx, info.CacheID); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: failed to delete cache from API: %v\n", err)
							} else {
								apiDeletedCount++
								fmt.Printf("Deleted from API: %s\n", info.CacheName)
							}
						}
						
						// Update or remove local file based on flags
						if withLocal {
							// Remove the local file entirely
							if err := os.Remove(path); err != nil {
								fmt.Fprintf(os.Stderr, "Failed to remove local file %s: %v\n", path, err)
							} else {
								fmt.Printf("Removed local cache: %s\n", info.CacheName)
								clearedCount++
							}
						} else if !preserveLocal {
							// Update the cache info with clear reason
							now := time.Now()
							info.ClearReason = "user-cleared"
							info.ClearedAt = &now
							
							data, _ := json.MarshalIndent(info, "", "  ")
							if err := os.WriteFile(path, data, 0644); err != nil {
								fmt.Fprintf(os.Stderr, "Failed to update cache info: %v\n", err)
							} else {
								fmt.Printf("Marked as cleared: %s\n", info.CacheName)
								clearedCount++
							}
						}
					}
				}
				
				if withLocal {
					fmt.Printf("\nDeleted %d cache(s) from API and removed %d local file(s).\n", apiDeletedCount, clearedCount)
				} else if preserveLocal {
					fmt.Printf("\nDeleted %d cache(s) from API (local files unchanged).\n", apiDeletedCount)
				} else {
					fmt.Printf("\nDeleted %d cache(s) from API and marked %d local cache(s) as cleared.\n", apiDeletedCount, clearedCount)
				}
			} else {
				for _, cacheName := range args {
					fileName := "hybrid_" + cacheName + ".json"
					path := filepath.Join(cacheDir, fileName)
					
					// Load cache info to get the cache ID
					info, err := gemini.LoadCacheInfo(path)
					if err != nil {
						if os.IsNotExist(err) {
							fmt.Fprintf(os.Stderr, "Cache '%s' not found locally.\n", cacheName)
						} else {
							fmt.Fprintf(os.Stderr, "Failed to read cache '%s': %v\n", cacheName, err)
						}
						continue
					}
					
					// Skip if already cleared
					if info.ClearedAt != nil {
						fmt.Printf("Cache '%s' already marked as cleared.\n", cacheName)
						continue
					}
					
					// Delete from API
					if client != nil {
						if err := client.DeleteCache(ctx, info.CacheID); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to delete cache from API: %v\n", err)
						} else {
							fmt.Printf("Deleted from API: %s\n", cacheName)
						}
					}
					
					// Update or remove local file based on flags
					if withLocal {
						// Remove the local file entirely
						if err := os.Remove(path); err != nil {
							fmt.Fprintf(os.Stderr, "Failed to remove local cache '%s': %v\n", cacheName, err)
						} else {
							fmt.Printf("Removed local cache: %s\n", cacheName)
						}
					} else if !preserveLocal {
						// Update the cache info with clear reason
						now := time.Now()
						info.ClearReason = "user-cleared"
						info.ClearedAt = &now
						
						data, _ := json.MarshalIndent(info, "", "  ")
						if err := os.WriteFile(path, data, 0644); err != nil {
							fmt.Fprintf(os.Stderr, "Failed to update cache info: %v\n", err)
						} else {
							fmt.Printf("Marked as cleared: %s\n", cacheName)
						}
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Clear all caches in the current project")
	cmd.Flags().BoolVar(&withLocal, "with-local", false, "Also remove local cache files (default: mark as cleared)")
	cmd.Flags().BoolVar(&preserveLocal, "preserve-local", false, "Don't update local cache files at all")
	
	return cmd
}

func newCachePruneCmd() *cobra.Command {
	var removeLocal bool
	
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Mark expired caches as cleared and optionally clean up",
		Long: `Marks expired cache records as cleared and removes them from Google's API.
By default, updates local files to mark them as expired.
Use --remove-local to also remove the local cache files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
			
			files, err := os.ReadDir(cacheDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No cache directory found. Nothing to prune.")
					return nil
				}
				return fmt.Errorf("reading cache directory: %w", err)
			}
			
			// Create client for API operations
			client, err := gemini.NewClient(ctx, "")
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}
			
			prunedCount := 0
			apiDeletedCount := 0
			
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
					path := filepath.Join(cacheDir, file.Name())
					info, err := gemini.LoadCacheInfo(path)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not read cache info for %s: %v\n", file.Name(), err)
						continue
					}
					
					// Skip if already cleared
					if info.ClearedAt != nil {
						continue
					}
					
					if time.Now().After(info.ExpiresAt) {
						// Try to delete from API (it might already be gone)
						if client != nil {
							if err := client.DeleteCache(ctx, info.CacheID); err == nil {
								apiDeletedCount++
							}
						}
						
						if removeLocal {
							// Remove the file
							if err := os.Remove(path); err != nil {
								fmt.Fprintf(os.Stderr, "Failed to remove expired cache %s: %v\n", file.Name(), err)
							} else {
								fmt.Printf("Removed expired cache: %s\n", info.CacheName)
								prunedCount++
							}
						} else {
							// Mark as expired
							now := time.Now()
							info.ClearReason = "expired"
							info.ClearedAt = &now
							
							data, _ := json.MarshalIndent(info, "", "  ")
							if err := os.WriteFile(path, data, 0644); err != nil {
								fmt.Fprintf(os.Stderr, "Failed to update cache info: %v\n", err)
							} else {
								fmt.Printf("Marked as expired: %s\n", info.CacheName)
								prunedCount++
							}
						}
					}
				}
			}
			
			if prunedCount == 0 {
				fmt.Println("No expired caches to prune.")
			} else {
				if removeLocal {
					fmt.Printf("\nRemoved %d expired cache file(s) and deleted %d from API.\n", prunedCount, apiDeletedCount)
				} else {
					fmt.Printf("\nMarked %d cache(s) as expired and deleted %d from API.\n", prunedCount, apiDeletedCount)
				}
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&removeLocal, "remove-local", false, "Remove local cache files instead of marking them")
	
	return cmd
}

func newCacheInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [cache-name]",
		Short: "Show detailed information about a specific cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheName := args[0]
			
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			
			cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
			cacheFile := filepath.Join(cacheDir, "hybrid_"+cacheName+".json")
			
			// Load cache info
			info, err := gemini.LoadCacheInfo(cacheFile)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("cache '%s' not found", cacheName)
				}
				return fmt.Errorf("loading cache info: %w", err)
			}
			
			// Determine status
			status := "Valid"
			if time.Now().After(info.ExpiresAt) {
				status = "Expired"
			}
			
			// Print header
			fmt.Println("╭─────────────────────────────────────────────────────────────────╮")
			fmt.Printf("│ Cache Details: %s%s │\n", info.CacheName, strings.Repeat(" ", 48-len(info.CacheName)))
			fmt.Println("├─────────────────────────────────────────────────────────────────┤")
			
			// Print basic info
			fmt.Printf("│ Server Cache ID: %-46s │\n", info.CacheID)
			fmt.Printf("│ Model:           %-46s │\n", info.Model)
			fmt.Printf("│ Status:          %-46s │\n", status)
			fmt.Printf("│ Created:         %-46s │\n", info.CreatedAt.Local().Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("│ Expires:         %-46s │\n", info.ExpiresAt.Local().Format("2006-01-02 15:04:05 MST"))
			
			// Print usage statistics
			if info.UsageStats != nil && info.UsageStats.TotalQueries > 0 {
				fmt.Println("├─────────────────────────────────────────────────────────────────┤")
				fmt.Println("│ Usage Statistics:                                               │")
				fmt.Printf("│ Total Queries:   %-46d │\n", info.UsageStats.TotalQueries)
				fmt.Printf("│ Last Used:       %-46s │\n", info.UsageStats.LastUsed.Local().Format("2006-01-02 15:04:05 MST"))
				fmt.Printf("│ Cache Hit Rate:  %-46s │\n", fmt.Sprintf("%.1f%%", info.UsageStats.AverageHitRate*100))
				fmt.Printf("│ Tokens Served:   %-46s │\n", fmt.Sprintf("%d", info.UsageStats.TotalCacheHits))
				fmt.Printf("│ Tokens Saved:    %-46s │\n", fmt.Sprintf("%d", info.UsageStats.TotalTokensSaved))
			}
			
			// Print cached files
			if len(info.CachedFileHashes) > 0 {
				fmt.Println("├─────────────────────────────────────────────────────────────────┤")
				fmt.Println("│ Cached Files:                                                   │")
				for file, hash := range info.CachedFileHashes {
					// Truncate long file paths
					displayFile := file
					if len(displayFile) > 60 {
						displayFile = "..." + displayFile[len(displayFile)-57:]
					}
					fmt.Printf("│   %-61s │\n", displayFile)
					fmt.Printf("│     SHA256: %-51s │\n", hash[:16]+"...")
				}
			}
			
			fmt.Println("╰─────────────────────────────────────────────────────────────────╯")
			
			return nil
		},
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// calculateCacheCost calculates the cost of storing cached content
// Based on Gemini pricing: https://ai.google.dev/pricing
// Cached content costs $1.00 per million tokens per hour for Gemini 1.5 Pro/Flash
// For Gemini 2.0, pricing may differ
func calculateCacheCost(tokenCount int32, duration time.Duration, model string) string {
	if tokenCount <= 0 || duration <= 0 {
		return "-"
	}
	
	// Cost per million tokens per hour in USD
	var costPerMillionTokensPerHour float64
	
	// Set pricing based on model
	switch {
	case strings.Contains(model, "gemini-2.0"):
		// Gemini 2.0 models - using same rate for now, update when pricing is announced
		costPerMillionTokensPerHour = 1.00
	case strings.Contains(model, "gemini-1.5-pro"), strings.Contains(model, "gemini-1.5-flash"):
		// Gemini 1.5 Pro and Flash
		costPerMillionTokensPerHour = 1.00
	default:
		// Default pricing
		costPerMillionTokensPerHour = 1.00
	}
	
	// Calculate cost
	tokens := float64(tokenCount)
	hours := duration.Hours()
	cost := (tokens / 1_000_000) * hours * costPerMillionTokensPerHour
	
	// Format cost
	if cost < 0.01 {
		return "<$0.01"
	} else if cost < 1.00 {
		return fmt.Sprintf("$%.2f", cost)
	} else {
		return fmt.Sprintf("$%.2f", cost)
	}
}

// cacheRow holds data for a cache row with sorting metadata
type cacheRow struct {
	data       []string
	status     string
	createTime time.Time
	isActive   bool
}

func listCachesCombined() error {
	ctx := context.Background()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	
	// Load local caches
	cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
	localCaches := make(map[string]*gemini.CacheInfo)
	
	files, err := os.ReadDir(cacheDir)
	if err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
				info, err := gemini.LoadCacheInfo(filepath.Join(cacheDir, file.Name()))
				if err == nil {
					localCaches[info.CacheID] = info
				}
			}
		}
	}
	
	// Query API caches
	client, err := gemini.NewClient(ctx, "")
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	
	fmt.Println("Querying caches from Google API...")
	apiCaches, err := client.ListCachesFromAPI(ctx)
	if err != nil {
		// If API fails, fall back to local only
		fmt.Fprintf(os.Stderr, "Warning: Could not query API (%v), showing local caches only\n", err)
		return listLocalCachesOnly()
	}
	
	// Create a map of API caches for easy lookup
	apiCacheMap := make(map[string]*gemini.CachedContentInfo)
	for i := range apiCaches {
		apiCacheMap[apiCaches[i].Name] = &apiCaches[i]
	}
	
	// Create table data
	var cacheRows []cacheRow
	shown := make(map[string]bool)
	
	// First add local caches
	for cacheID, localInfo := range localCaches {
		shown[cacheID] = true
		
		// Determine overall status
		var status string
		var isValid bool
		
		// Check if manually cleared
		if localInfo.ClearedAt != nil {
			status = theme.IconError + " Cleared"
			isValid = false
		} else {
			// Check if exists in API
			apiCache, existsInAPI := apiCacheMap[cacheID]

			if existsInAPI {
				// Cache exists in API
				if time.Now().After(apiCache.ExpireTime) {
					status = theme.IconWarning + " Expired"
					isValid = false
				} else {
					status = theme.IconSuccess + " Active"
					isValid = true
				}
			} else {
				// Not found in API
				status = theme.IconInfo + " Missing"
				isValid = false
			}
		}
		
		// Format values based on validity
		var usesStr, tokenStr, expiresInStr, expireTimeStr, costStr string
		
		// Get usage count
		if localInfo.UsageStats != nil && localInfo.UsageStats.TotalQueries > 0 {
			usesStr = fmt.Sprintf("%d", localInfo.UsageStats.TotalQueries)
		} else {
			usesStr = "0"
		}
		
		if isValid {
			// Check if we have API data
			apiCache, existsInAPI := apiCacheMap[cacheID]
			
			if existsInAPI {
				// Valid cache - show all data from API
				if apiCache.TokenCount > 0 {
					tokenStr = fmt.Sprintf("%dk", apiCache.TokenCount/1000)
				} else if localInfo.TokenCount > 0 {
					tokenStr = fmt.Sprintf("%dk", localInfo.TokenCount/1000)
				} else {
					tokenStr = "-"
				}
				
				expiresIn := time.Until(apiCache.ExpireTime).Round(time.Second)
				expiresInStr = formatDuration(expiresIn)
				expireTimeStr = apiCache.ExpireTime.Local().Format("15:04")
				
				// Calculate cost
				if apiCache.TokenCount > 0 {
					totalDuration := apiCache.ExpireTime.Sub(apiCache.CreateTime)
					costStr = calculateCacheCost(apiCache.TokenCount, totalDuration, apiCache.Model)
				} else if localInfo.TokenCount > 0 {
					totalDuration := localInfo.ExpiresAt.Sub(localInfo.CreatedAt)
					costStr = calculateCacheCost(int32(localInfo.TokenCount), totalDuration, localInfo.Model)
				} else {
					costStr = "-"
				}
			} else {
				// Valid locally but not in API
				tokenStr = "-"
				expiresInStr = "-"
				expireTimeStr = "-"
				costStr = "-"
			}
		} else {
			// Invalid/missing cache - show dashes
			tokenStr = "-"
			expiresInStr = "-"
			expireTimeStr = "-"
			costStr = "-"
		}
		
		// Get repo name, truncate if needed
		repoName := localInfo.RepoName
		if repoName == "" {
			repoName = "-"
		} else if len(repoName) > 15 {
			repoName = repoName[:12] + "..."
		}
		
		// Determine if cache is active
		isActive := status == theme.IconSuccess+" Active"
		
		// Get creation time from local info or API
		createTime := localInfo.CreatedAt
		if apiCache, ok := apiCacheMap[cacheID]; ok && !createTime.After(apiCache.CreateTime) {
			createTime = apiCache.CreateTime
		}
		
		cacheRows = append(cacheRows, cacheRow{
			data: []string{
				localInfo.CacheName,
				repoName,
				localInfo.Model,
				status,
				usesStr,
				tokenStr,
				expiresInStr,
				expireTimeStr,
				costStr,
			},
			status:     status,
			createTime: createTime,
			isActive:   isActive,
		})
	}
	
	// Then add API-only caches (not in local)
	for _, apiCache := range apiCaches {
		if !shown[apiCache.Name] {
			cacheName := apiCache.Name
			if parts := strings.Split(apiCache.Name, "/"); len(parts) > 1 {
				cacheName = parts[len(parts)-1]
			}
			if len(cacheName) > 16 {
				cacheName = cacheName[:16]
			}
			
			// These are API-only, so no local file
			var status string
			var isValid bool

			if time.Now().After(apiCache.ExpireTime) {
				status = theme.IconWarning + " Expired"
				isValid = false
			} else {
				status = theme.IconSuccess + " Active"
				isValid = true
			}
			
			// Format values based on validity
			var usesStr, tokenStr, expiresInStr, expireTimeStr, costStr string
			
			// API-only caches don't have local usage stats
			usesStr = "0"
			
			if isValid {
				// Valid cache - show all data
				if apiCache.TokenCount > 0 {
					tokenStr = fmt.Sprintf("%dk", apiCache.TokenCount/1000)
				} else {
					tokenStr = "-"
				}
				
				expiresIn := time.Until(apiCache.ExpireTime).Round(time.Second)
				expiresInStr = formatDuration(expiresIn)
				expireTimeStr = apiCache.ExpireTime.Local().Format("15:04")
				
				// Calculate cost
				totalDuration := apiCache.ExpireTime.Sub(apiCache.CreateTime)
				costStr = calculateCacheCost(apiCache.TokenCount, totalDuration, apiCache.Model)
			} else {
				// Expired cache - show dashes
				tokenStr = "-"
				expiresInStr = "-"
				expireTimeStr = "-"
				costStr = "-"
			}
			
			// Determine if cache is active
			isActive := status == theme.IconSuccess+" Active"
			
			cacheRows = append(cacheRows, cacheRow{
				data: []string{
					cacheName,
					"-", // No repo info for API-only caches
					apiCache.Model,
					status,
					usesStr,
					tokenStr,
					expiresInStr,
					expireTimeStr,
					costStr,
				},
				status:     status,
				createTime: apiCache.CreateTime,
				isActive:   isActive,
			})
		}
	}
	
	// Sort caches: active first, then by creation time (newest first)
	sort.Slice(cacheRows, func(i, j int) bool {
		// First priority: active caches come first
		if cacheRows[i].isActive != cacheRows[j].isActive {
			return cacheRows[i].isActive
		}
		// Second priority: newer caches come first
		return cacheRows[i].createTime.After(cacheRows[j].createTime)
	})
	
	// Convert sorted cacheRows back to rows for table
	var rows [][]string
	for _, cr := range cacheRows {
		rows = append(rows, cr.data)
	}
	
	// Create styled table
	t := tablecomponent.NewStyledTable().
		Headers("CACHE NAME", "REPO", "MODEL", "STATUS", "USES", "TOKENS", "TTL", "EXPIRES", "COST").
		Rows(rows...)
	
	fmt.Println()
	fmt.Println(t)
	fmt.Printf("\nLocal caches: %d, API caches: %d\n", len(localCaches), len(apiCaches))
	
	return nil
}

func listLocalCachesOnly() error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")

	files, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No cache directory found. Nothing to list.")
			return nil
		}
		return fmt.Errorf("reading cache directory: %w", err)
	}

	var cacheRows []cacheRow
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
			info, err := gemini.LoadCacheInfo(filepath.Join(cacheDir, file.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read cache info for %s: %v\n", file.Name(), err)
				continue
			}
			
			// Note: Local-only view doesn't check API, so we show local status
			status := theme.IconInfo + " Local"
			isValid := !time.Now().After(info.ExpiresAt)
			
			// Format values based on local validity
			var usesStr, tokenStr, expiresInStr, expireTimeStr, costStr string
			
			// Get usage count
			if info.UsageStats != nil && info.UsageStats.TotalQueries > 0 {
				usesStr = fmt.Sprintf("%d", info.UsageStats.TotalQueries)
			} else {
				usesStr = "0"
			}
			
			if isValid {
				// Valid locally - show all data
				if info.TokenCount > 0 {
					tokenStr = fmt.Sprintf("%dk", info.TokenCount/1000)
				} else {
					tokenStr = "-"
				}
				
				expiresIn := time.Until(info.ExpiresAt).Round(time.Second)
				expiresInStr = formatDuration(expiresIn)
				expireTimeStr = info.ExpiresAt.Local().Format("15:04")
				
				// Calculate cost
				totalDuration := info.ExpiresAt.Sub(info.CreatedAt)
				costStr = calculateCacheCost(int32(info.TokenCount), totalDuration, info.Model)
			} else {
				// Expired locally
				status = theme.IconWarning + " Expired"
				tokenStr = "-"
				expiresInStr = "-"
				expireTimeStr = "-"
				costStr = "-"
			}
			
			// Get repo name, truncate if needed
			repoName := info.RepoName
			if repoName == "" {
				repoName = "-"
			} else if len(repoName) > 15 {
				repoName = repoName[:12] + "..."
			}
			
			// Determine if cache is active (for local-only, active means not expired and not cleared)
			isActive := isValid && info.ClearedAt == nil && status != theme.IconWarning+" Expired"
			
			cacheRows = append(cacheRows, cacheRow{
				data: []string{
					info.CacheName,
					repoName,
					info.Model,
					status,
					usesStr,
					tokenStr,
					expiresInStr,
					expireTimeStr,
					costStr,
				},
				status:     status,
				createTime: info.CreatedAt,
				isActive:   isActive,
			})
		}
	}
	
	if len(cacheRows) == 0 {
		fmt.Println("No caches found in this project.")
		return nil
	}
	
	// Sort caches: active first, then by creation time (newest first)
	sort.Slice(cacheRows, func(i, j int) bool {
		// First priority: active caches come first
		if cacheRows[i].isActive != cacheRows[j].isActive {
			return cacheRows[i].isActive
		}
		// Second priority: newer caches come first
		return cacheRows[i].createTime.After(cacheRows[j].createTime)
	})
	
	// Convert sorted cacheRows back to rows for table
	var rows [][]string
	for _, cr := range cacheRows {
		rows = append(rows, cr.data)
	}
	
	// Create styled table
	t := tablecomponent.NewStyledTable().
		Headers("CACHE NAME", "REPO", "MODEL", "STATUS", "USES", "TOKENS", "TTL", "EXPIRES", "COST").
		Rows(rows...)
	
	fmt.Println(t)

	return nil
}

func listCachesFromAPI() error {
	ctx := context.Background()
	
	// Create client
	client, err := gemini.NewClient(ctx, "")
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	
	// List caches from API
	fmt.Println("Querying caches from Google API...")
	caches, err := client.ListCachesFromAPI(ctx)
	if err != nil {
		return fmt.Errorf("listing caches from API: %w", err)
	}
	
	if len(caches) == 0 {
		fmt.Println("No caches found on Google's servers.")
		return nil
	}
	
	// Build table rows
	var cacheRows []cacheRow
	for _, cache := range caches {
		// Extract cache ID from the full name (format: cachedContents/abc123...)
		cacheName := cache.Name
		if parts := strings.Split(cache.Name, "/"); len(parts) > 1 {
			cacheName = parts[len(parts)-1]
		}
		
		// Determine status
		status := theme.IconSuccess + " Valid"
		expiresIn := time.Until(cache.ExpireTime).Round(time.Second)
		expiresInStr := formatDuration(expiresIn)
		if time.Now().After(cache.ExpireTime) {
			status = theme.IconWarning + " Expired"
			expiresInStr = "expired"
		}
		
		// Truncate cache name if too long
		if len(cacheName) > 20 {
			cacheName = cacheName[:17] + "..."
		}
		
		// Format token count from API
		tokenStr := "-"
		if cache.TokenCount > 0 {
			tokenStr = fmt.Sprintf("%dk", cache.TokenCount/1000)
		}
		
		// API-only caches don't have local usage stats
		usesStr := "0"
		
		// Format expire time
		expireTimeStr := cache.ExpireTime.Local().Format("15:04")
		if time.Now().After(cache.ExpireTime) {
			expireTimeStr = "expired"
		}
		
		// Calculate cost
		totalDuration := cache.ExpireTime.Sub(cache.CreateTime)
		costStr := calculateCacheCost(cache.TokenCount, totalDuration, cache.Model)
		
		// Determine if cache is active
		isActive := status == theme.IconSuccess+" Valid"
		
		cacheRows = append(cacheRows, cacheRow{
			data: []string{
				cacheName,
				"-", // No repo info from API
				cache.Model,
				status,
				usesStr,
				tokenStr,
				expiresInStr,
				expireTimeStr,
				costStr,
			},
			status:     status,
			createTime: cache.CreateTime,
			isActive:   isActive,
		})
	}
	
	// Sort caches: active first, then by creation time (newest first)
	sort.Slice(cacheRows, func(i, j int) bool {
		// First priority: active caches come first
		if cacheRows[i].isActive != cacheRows[j].isActive {
			return cacheRows[i].isActive
		}
		// Second priority: newer caches come first
		return cacheRows[i].createTime.After(cacheRows[j].createTime)
	})
	
	// Convert sorted cacheRows back to rows for table
	var rows [][]string
	for _, cr := range cacheRows {
		rows = append(rows, cr.data)
	}
	
	// Create styled table
	t := tablecomponent.NewStyledTable().
		Headers("CACHE NAME", "REPO", "MODEL", "STATUS", "USES", "TOKENS", "TTL", "EXPIRES", "COST").
		Rows(rows...)
	
	fmt.Println()
	fmt.Println(t)
	fmt.Printf("\nTotal: %d cache(s) found on Google's servers\n", len(caches))
	
	return nil
}