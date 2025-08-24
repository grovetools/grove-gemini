package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// QueryLog represents a single API query log entry
type QueryLog struct {
	Timestamp        time.Time `json:"timestamp"`
	Model           string    `json:"model"`
	Method          string    `json:"method,omitempty"`
	CachedTokens    int32     `json:"cached_tokens"`
	PromptTokens    int32     `json:"prompt_tokens"`
	CompletionTokens int32     `json:"completion_tokens"`
	TotalTokens     int32     `json:"total_tokens"`
	CacheHitRate    float64   `json:"cache_hit_rate"`
	ResponseTime    float64   `json:"response_time_seconds"`
	EstimatedCost   float64   `json:"estimated_cost_usd"`
	Error           string    `json:"error,omitempty"`
	CacheID         string    `json:"cache_id,omitempty"`
	Success         bool      `json:"success"`
	
	// Context information
	WorkingDir      string `json:"working_dir,omitempty"`
	GitRepo         string `json:"git_repo,omitempty"`
	GitBranch       string `json:"git_branch,omitempty"`
	GitCommit       string `json:"git_commit,omitempty"`
	Caller          string `json:"caller,omitempty"` // e.g., "grove-flow", "gemapi-request", "gemapi-count-tokens"
}

// QueryLogger handles logging of API queries
type QueryLogger struct {
	mu       sync.Mutex
	logFile  string
	disabled bool
}

var (
	defaultLogger *QueryLogger
	once         sync.Once
)

// GetLogger returns the singleton query logger instance
func GetLogger() *QueryLogger {
	once.Do(func() {
		logPath, err := getLogPath()
		if err != nil {
			// If we can't create the log path, create a disabled logger
			defaultLogger = &QueryLogger{disabled: true}
			return
		}
		defaultLogger = &QueryLogger{
			logFile: logPath,
		}
	})
	return defaultLogger
}

// getLogPath returns the path to the query log file
func getLogPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	groveDir := filepath.Join(homeDir, ".grove", "gemini-cache")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		return "", err
	}
	
	// Use date-based log files for easy rotation
	today := time.Now().Format("2006-01-02")
	return filepath.Join(groveDir, fmt.Sprintf("query-log-%s.jsonl", today)), nil
}

// Log adds a new query log entry
func (ql *QueryLogger) Log(entry QueryLog) error {
	if ql.disabled {
		return nil
	}
	
	ql.mu.Lock()
	defer ql.mu.Unlock()
	
	// Open file in append mode
	file, err := os.OpenFile(ql.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()
	
	// Write as JSON Lines format (one JSON object per line)
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}
	
	return nil
}

// ReadLogs reads log entries from the log file
func (ql *QueryLogger) ReadLogs(startTime, endTime time.Time) ([]QueryLog, error) {
	if ql.disabled {
		return nil, fmt.Errorf("logging is disabled")
	}
	
	ql.mu.Lock()
	defer ql.mu.Unlock()
	
	var allLogs []QueryLog
	
	// Check multiple days if time range spans multiple days
	// Use date.Before(endTime.AddDate(0, 0, 1)) to include the end date
	for date := startTime; date.Before(endTime.AddDate(0, 0, 1)); date = date.AddDate(0, 0, 1) {
		dayStr := date.Format("2006-01-02")
		logFile := filepath.Join(filepath.Dir(ql.logFile), fmt.Sprintf("query-log-%s.jsonl", dayStr))
		
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			continue
		}
		
		file, err := os.Open(logFile)
		if err != nil {
			continue
		}
		
		decoder := json.NewDecoder(file)
		for decoder.More() {
			var entry QueryLog
			if err := decoder.Decode(&entry); err != nil {
				continue
			}
			
			// Filter by time range (inclusive)
			if !entry.Timestamp.Before(startTime) && !entry.Timestamp.After(endTime) {
				allLogs = append(allLogs, entry)
			}
		}
		
		file.Close()
	}
	
	return allLogs, nil
}

// EstimateCost calculates the estimated cost based on token usage
// Cached tokens get a 75% discount on input pricing
func EstimateCost(model string, promptTokens, completionTokens int32) float64 {
	return EstimateCostWithCache(model, promptTokens, completionTokens, 0)
}

// EstimateCostWithCache calculates the estimated cost accounting for cached token discounts
// Cached tokens get a 75% discount on input pricing
func EstimateCostWithCache(model string, promptTokens, completionTokens, cachedTokens int32) float64 {
	// Pricing as of Dec 2024 (per million tokens)
	var inputPrice, outputPrice float64
	
	modelLower := strings.ToLower(model)
	
	switch {
	// Gemini 2.5 models
	case contains(modelLower, "gemini-2.5-pro"):
		// Note: We're using the base pricing, not accounting for >200k prompts
		inputPrice = 1.25
		outputPrice = 10.00
	case contains(modelLower, "gemini-2.5-flash") && contains(modelLower, "lite"):
		inputPrice = 0.10
		outputPrice = 0.40
	case contains(modelLower, "gemini-2.5-flash"):
		inputPrice = 0.30
		outputPrice = 2.50
		
	// Gemini 2.0 models
	case contains(modelLower, "gemini-2.0-flash") && contains(modelLower, "lite"):
		inputPrice = 0.075
		outputPrice = 0.30
	case contains(modelLower, "gemini-2.0-flash"):
		inputPrice = 0.10
		outputPrice = 0.40
		
	// Legacy patterns for backward compatibility
	case contains(modelLower, "flash"):
		inputPrice = 0.10   // Default to 2.0 flash pricing
		outputPrice = 0.40
	case contains(modelLower, "pro"):
		inputPrice = 1.25   // Default to 2.5 pro pricing
		outputPrice = 10.00
		
	default:
		// Default to 2.0 flash pricing
		inputPrice = 0.10
		outputPrice = 0.40
	}
	
	// Calculate costs with cache discount
	// Cached tokens get 75% discount
	const cacheDiscount = 0.25 // Pay only 25% of the price for cached tokens
	
	// Separate dynamic tokens from cached tokens
	dynamicTokens := promptTokens - cachedTokens
	
	cachedCost := float64(cachedTokens) / 1_000_000 * inputPrice * cacheDiscount
	dynamicCost := float64(dynamicTokens) / 1_000_000 * inputPrice
	outputCost := float64(completionTokens) / 1_000_000 * outputPrice
	
	return cachedCost + dynamicCost + outputCost
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}