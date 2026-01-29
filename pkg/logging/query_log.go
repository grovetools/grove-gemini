package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/grovetools/core/pkg/paths"
)

// QueryLog represents a single API query log entry
type QueryLog struct {
	Timestamp        time.Time `json:"timestamp"`
	RequestID        string    `json:"request_id,omitempty"`
	Model           string    `json:"model"`
	Method          string    `json:"method,omitempty"`
	CachedTokens    int32     `json:"cached_tokens"`
	PromptTokens    int32     `json:"prompt_tokens"`
	UserPromptTokens int32     `json:"user_prompt_tokens,omitempty"` // Tokens from user's text prompt only
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
	stateDir := paths.StateDir()
	if stateDir == "" {
		return "", fmt.Errorf("could not determine grove state directory")
	}

	groveDir := filepath.Join(stateDir, "logs", "gemini")
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
	// Pricing as of Jan 2025 (per million tokens)
	// GCP has different pricing for "long context" (>128K tokens) vs "short context"
	var inputPrice, outputPrice float64

	modelLower := strings.ToLower(model)

	// Determine if this is a long context request (>128K tokens)
	// We use total prompt tokens (including cached) for this determination
	totalPromptTokens := promptTokens
	isLongContext := totalPromptTokens > 128000

	switch {
	// Gemini 2.5 Pro models
	case contains(modelLower, "gemini-2.5-pro"):
		if isLongContext {
			// Long context pricing (observed from GCP billing)
			inputPrice = 2.50
			outputPrice = 15.00
		} else {
			// Short context pricing
			inputPrice = 1.25
			outputPrice = 10.00
		}

	// Gemini 2.5 Flash models
	case contains(modelLower, "gemini-2.5-flash") && contains(modelLower, "lite"):
		inputPrice = 0.10
		outputPrice = 0.40
	case contains(modelLower, "gemini-2.5-flash"):
		// Flash doesn't seem to have long/short pricing tiers in billing data
		inputPrice = 0.30
		outputPrice = 2.50

	// Gemini 2.0 Flash models
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
		// Default to short context pricing for Pro
		inputPrice = 1.25
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