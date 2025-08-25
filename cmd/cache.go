package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-gemini/pkg/gemini"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local cache of Gemini contexts",
		Long:  `Provides commands to manage the local cache of Gemini API context data. You can list, inspect, clear, and prune cached items.`,
	}

	cmd.AddCommand(newCacheListCmd())
	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePruneCmd())
	cmd.AddCommand(newCacheInspectCmd())

	return cmd
}

func newCacheListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all locally known caches for the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			fmt.Printf("%-18s %-20s %-10s %s\n", "CACHE NAME", "MODEL", "STATUS", "EXPIRES IN")
			fmt.Println(strings.Repeat("-", 70))

			var count int
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
					info, err := gemini.LoadCacheInfo(filepath.Join(cacheDir, file.Name()))
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not read cache info for %s: %v\n", file.Name(), err)
						continue
					}
					
					status := "✅ Valid"
					expiresIn := time.Until(info.ExpiresAt).Round(time.Second)
					expiresInStr := formatDuration(expiresIn)
					if time.Now().After(info.ExpiresAt) {
						status = "⏰ Expired"
						expiresInStr = "expired"
					}

					fmt.Printf("%-18s %-20s %-10s %s\n", info.CacheName, info.Model, status, expiresInStr)
					count++
				}
			}
			
			if count == 0 {
				fmt.Println("No caches found in this project.")
			}

			return nil
		},
	}
}

func newCacheClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear [cache-name...] | --all",
		Short: "Remove a specific cache or all caches",
		Long:  `Removes local cache information files. Note: This does not delete the cache from Google's servers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			if !all && len(args) == 0 {
				return fmt.Errorf("must specify a cache name to clear, or use the --all flag")
			}
			
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
			
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
				for _, file := range files {
					if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
						path := filepath.Join(cacheDir, file.Name())
						if err := os.Remove(path); err != nil {
							fmt.Fprintf(os.Stderr, "Failed to remove %s: %v\n", path, err)
						} else {
							fmt.Printf("Removed cache info: %s\n", file.Name())
							clearedCount++
						}
					}
				}
				fmt.Printf("\nSuccessfully cleared %d cache(s).\n", clearedCount)
			} else {
				for _, cacheName := range args {
					fileName := "hybrid_" + cacheName + ".json"
					path := filepath.Join(cacheDir, fileName)
					if err := os.Remove(path); err != nil {
						if os.IsNotExist(err) {
							fmt.Fprintf(os.Stderr, "Cache '%s' not found.\n", cacheName)
						} else {
							fmt.Fprintf(os.Stderr, "Failed to remove cache '%s': %v\n", cacheName, err)
						}
					} else {
						fmt.Printf("Removed cache info: %s\n", cacheName)
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Clear all caches in the current project")
	return cmd
}

func newCachePruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove all expired cache records",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			
			prunedCount := 0
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
					info, err := gemini.LoadCacheInfo(filepath.Join(cacheDir, file.Name()))
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not read cache info for %s: %v\n", file.Name(), err)
						continue
					}
					
					if time.Now().After(info.ExpiresAt) {
						path := filepath.Join(cacheDir, file.Name())
						if err := os.Remove(path); err != nil {
							fmt.Fprintf(os.Stderr, "Failed to remove expired cache %s: %v\n", file.Name(), err)
						} else {
							fmt.Printf("Pruned expired cache: %s\n", info.CacheName)
							prunedCount++
						}
					}
				}
			}
			
			if prunedCount == 0 {
				fmt.Println("No expired caches to prune.")
			} else {
				fmt.Printf("\nSuccessfully pruned %d expired cache(s).\n", prunedCount)
			}
			
			return nil
		},
	}
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