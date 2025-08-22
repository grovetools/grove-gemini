package gcp

import (
	"context"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/logging/logadmin"
)

// NewMonitoringClient creates a new Cloud Monitoring client
func NewMonitoringClient(ctx context.Context) (*monitoring.MetricClient, error) {
	return monitoring.NewMetricClient(ctx)
}

// NewLoggingAdminClient creates a new Cloud Logging admin client
func NewLoggingAdminClient(ctx context.Context, projectID string) (*logadmin.Client, error) {
	return logadmin.NewClient(ctx, projectID)
}

// NewBigQueryClient creates a new BigQuery client
func NewBigQueryClient(ctx context.Context, projectID string) (*bigquery.Client, error) {
	return bigquery.NewClient(ctx, projectID)
}