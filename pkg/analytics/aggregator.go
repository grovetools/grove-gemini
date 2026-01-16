package analytics

import (
	"time"

	"github.com/grovetools/grove-gemini/pkg/logging"
)

// Bucket holds aggregated data for a time interval.
type Bucket struct {
	StartTime             time.Time
	TotalCost             float64
	TotalTokens           int64
	TotalPromptTokens     int64
	TotalCompletionTokens int64
	RequestCount          int
	ErrorCount            int
}

// Totals holds the summary statistics for a given time range.
type Totals struct {
	TotalCost     float64
	TotalTokens   int64
	TotalRequests int
	ErrorRate     float64
}

// AggregateLogs groups logs into time-based buckets.
func AggregateLogs(logs []logging.QueryLog, interval time.Duration, startTime time.Time, endTime time.Time) []Bucket {
	numBuckets := int(endTime.Sub(startTime)/interval) + 1
	buckets := make([]Bucket, numBuckets)

	for i := 0; i < numBuckets; i++ {
		buckets[i].StartTime = startTime.Add(time.Duration(i) * interval)
	}

	for _, log := range logs {
		if log.Timestamp.Before(startTime) || log.Timestamp.After(endTime) {
			continue
		}

		index := int(log.Timestamp.Sub(startTime) / interval)
		if index >= 0 && index < numBuckets {
			buckets[index].TotalCost += log.EstimatedCost
			buckets[index].TotalTokens += int64(log.TotalTokens)
			buckets[index].TotalPromptTokens += int64(log.PromptTokens)
			buckets[index].TotalCompletionTokens += int64(log.CompletionTokens)
			buckets[index].RequestCount++
			if !log.Success {
				buckets[index].ErrorCount++
			}
		}
	}

	return buckets
}

// CalculateTotals computes summary statistics from a slice of buckets.
func CalculateTotals(buckets []Bucket) Totals {
	var totals Totals
	for _, bucket := range buckets {
		totals.TotalCost += bucket.TotalCost
		totals.TotalTokens += bucket.TotalTokens
		totals.TotalRequests += bucket.RequestCount
	}
	if totals.TotalRequests > 0 {
		var totalErrors int
		for _, bucket := range buckets {
			totalErrors += bucket.ErrorCount
		}
		totals.ErrorRate = float64(totalErrors) / float64(totals.TotalRequests) * 100
	}
	return totals
}
