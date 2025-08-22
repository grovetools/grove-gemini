package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/mattsolo1/grove-gemini/pkg/config"
	"github.com/mattsolo1/grove-gemini/pkg/gcp"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	metricsProjectID string
	metricsHours     int
	metricsDebug     bool
)

func newQueryMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Query Gemini API metrics from Cloud Monitoring",
		Long:  `Fetches and displays Gemini API request counts, error rates, and latency metrics from Google Cloud Monitoring.`,
		RunE:  runQueryMetrics,
	}

	// Get default project from config
	defaultProject := config.GetDefaultProject("")
	
	cmd.Flags().StringVarP(&metricsProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().IntVarP(&metricsHours, "hours", "H", 24, "Number of hours to look back")
	cmd.Flags().BoolVar(&metricsDebug, "debug", false, "Enable debug output")

	return cmd
}

func runQueryMetrics(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Ensure we have a project ID
	if metricsProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	// Create monitoring client
	client, err := gcp.NewMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create monitoring client: %w", err)
	}
	defer client.Close()

	// Set time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(metricsHours) * time.Hour)

	interval := &monitoringpb.TimeInterval{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	methodMetrics := make(map[string]map[string]int64)

	// Query for request counts
	fmt.Printf("Fetching Gemini API metrics for the last %d hours...\n\n", metricsHours)

	// Try multiple filter approaches
	filters := []string{
		// Standard service runtime metrics
		`metric.type="serviceruntime.googleapis.com/api/request_count" AND resource.type="api" AND resource.labels.service="generativelanguage.googleapis.com"`,
		// Alternative: consumed_api resource type
		`metric.type="serviceruntime.googleapis.com/api/request_count" AND resource.type="consumed_api" AND resource.labels.service="generativelanguage.googleapis.com"`,
		// Alternative: Direct metric without resource filter
		`metric.type="generativelanguage.googleapis.com/request_count"`,
	}

	var successfulFilter string
	for _, filter := range filters {
		if metricsDebug {
			fmt.Printf("[DEBUG] Trying filter: %s\n", filter)
		}

		reqCounts := &monitoringpb.ListTimeSeriesRequest{
			Name:     fmt.Sprintf("projects/%s", metricsProjectID),
			Filter:   filter,
			Interval: interval,
		}

		it := client.ListTimeSeries(ctx, reqCounts)
		hasData := false
		seriesCount := 0
		for {
			series, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if metricsDebug {
					fmt.Printf("[DEBUG] Error with filter: %v\n", err)
				}
				break
			}

			hasData = true
			
			// Try different label keys for method
			method := ""
			if m, ok := series.Metric.Labels["method"]; ok {
				method = m
			} else if m, ok := series.Metric.Labels["api_method"]; ok {
				method = m
			} else if m, ok := series.Metric.Labels["api"]; ok {
				method = m
			}
			
			// If still no method, check resource labels
			if method == "" {
				if m, ok := series.Resource.Labels["method"]; ok {
					method = m
				} else if m, ok := series.Resource.Labels["api_method"]; ok {
					method = m
				}
			}
			
			// If still no method, check all labels in debug mode
			if method == "" && metricsDebug && seriesCount == 0 {
				fmt.Printf("[DEBUG] Available metric labels:\n")
				for k, v := range series.Metric.Labels {
					fmt.Printf("  %s: %s\n", k, v)
				}
				fmt.Printf("[DEBUG] Available resource labels:\n")
				for k, v := range series.Resource.Labels {
					fmt.Printf("  %s: %s\n", k, v)
				}
				method = "(unknown)"
			} else if method == "" {
				method = "(unknown)"
			}
			
			if methodMetrics[method] == nil {
				methodMetrics[method] = make(map[string]int64)
			}

			for _, point := range series.Points {
				methodMetrics[method]["requests"] += point.Value.GetInt64Value()
			}
			seriesCount++
		}

		if hasData {
			successfulFilter = filter
			if metricsDebug {
				fmt.Printf("[DEBUG] Found data with filter: %s\n", filter)
			}
			break
		}
	}

	// If no data found with any filter, try to list available metrics
	if len(methodMetrics) == 0 && metricsDebug {
		fmt.Println("[DEBUG] No metrics found. Attempting to list available metric descriptors...")
		listMetricDescriptors(ctx, client, metricsProjectID)
	}

	// Query for error counts (only if we found the right filter)
	if successfulFilter != "" {
		errorFilter := successfulFilter + ` AND metric.labels.response_code_class!="2xx"`
		if metricsDebug {
			fmt.Printf("[DEBUG] Error filter: %s\n", errorFilter)
		}

		reqErrors := &monitoringpb.ListTimeSeriesRequest{
			Name:     fmt.Sprintf("projects/%s", metricsProjectID),
			Filter:   errorFilter,
			Interval: interval,
		}

		errorIt := client.ListTimeSeries(ctx, reqErrors)
		for {
			series, err := errorIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				// Don't fail entirely if error metrics aren't available
				fmt.Printf("Warning: Could not fetch error metrics: %v\n", err)
				break
			}

			// Get method from resource labels
			method := ""
			if m, ok := series.Resource.Labels["method"]; ok {
				method = m
			} else {
				method = "(unknown)"
			}
			
			if methodMetrics[method] == nil {
				methodMetrics[method] = make(map[string]int64)
			}

			for _, point := range series.Points {
				methodMetrics[method]["errors"] += point.Value.GetInt64Value()
			}
		}
	}

	// Query for latency metrics
	reqLatency := &monitoringpb.ListTimeSeriesRequest{
		Name:     fmt.Sprintf("projects/%s", metricsProjectID),
		Filter:   `metric.type="serviceruntime.googleapis.com/api/request_latencies" AND resource.type="api" AND resource.labels.service="generativelanguage.googleapis.com"`,
		Interval: interval,
	}

	latencyIt := client.ListTimeSeries(ctx, reqLatency)
	for {
		series, err := latencyIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Don't fail entirely if latency metrics aren't available
			fmt.Printf("Warning: Could not fetch latency metrics: %v\n", err)
			break
		}

		method := series.Metric.Labels["method"]
		if methodMetrics[method] == nil {
			methodMetrics[method] = make(map[string]int64)
		}

		if len(series.Points) > 0 {
			dist := series.Points[0].Value.GetDistributionValue()
			if dist != nil {
				// Store average latency in milliseconds
				methodMetrics[method]["latency"] = int64(dist.Mean * 1000)
			}
		}
	}

	// Display results
	if len(methodMetrics) == 0 {
		fmt.Println("No metrics found for the specified time range.")
		if !metricsDebug {
			fmt.Println("\nTry running with --debug flag for more information.")
		}
		return nil
	}

	fmt.Println("=== Gemini API Metrics ===")
	fmt.Printf("Time Range: %s to %s\n\n", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	var totalRequests, totalErrors int64

	for method, metrics := range methodMetrics {
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("  Requests: %d\n", metrics["requests"])
		
		if errors, ok := metrics["errors"]; ok && metrics["requests"] > 0 {
			errorRate := float64(errors) / float64(metrics["requests"]) * 100
			fmt.Printf("  Errors: %d (%.2f%%)\n", errors, errorRate)
			totalErrors += errors
		} else {
			fmt.Printf("  Errors: 0 (0.00%%)\n")
		}

		if latency, ok := metrics["latency"]; ok {
			fmt.Printf("  Avg Latency: %dms\n", latency)
		}

		fmt.Println()
		totalRequests += metrics["requests"]
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	if totalRequests > 0 {
		totalErrorRate := float64(totalErrors) / float64(totalRequests) * 100
		fmt.Printf("Total Errors: %d (%.2f%%)\n", totalErrors, totalErrorRate)
	}

	return nil
}

// Helper function to list available metric descriptors
func listMetricDescriptors(ctx context.Context, client *monitoring.MetricClient, projectID string) {
	filter := `metric.type = starts_with("generativelanguage.googleapis.com/") OR metric.type = starts_with("serviceruntime.googleapis.com/")`
	
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", projectID),
		Filter: filter,
	}

	it := client.ListMetricDescriptors(ctx, req)
	fmt.Println("[DEBUG] Available metric types:")
	count := 0
	for {
		desc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("[DEBUG] Error listing metric descriptors: %v\n", err)
			return
		}
		if strings.Contains(desc.Type, "generativelanguage") || strings.Contains(desc.Type, "api") {
			fmt.Printf("  - %s\n", desc.Type)
			count++
		}
	}
	if count == 0 {
		fmt.Println("  (No relevant metrics found)")
	}
}