package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattsolo1/grove-gemini/pkg/config"
	"github.com/mattsolo1/grove-gemini/pkg/gcp"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var (
	billingProjectID string
	billingDatasetID string
	billingTableID   string
	billingDays      int
)

type BillingSummary struct {
	SKU        string  `bigquery:"sku_description"`
	TotalCost  float64 `bigquery:"total_cost"`
	TotalUsage float64 `bigquery:"total_usage_amount"`
	UsageUnit  string  `bigquery:"usage_unit"`
	Currency   string  `bigquery:"currency"`
}

func newQueryBillingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Query Gemini API billing data from BigQuery",
		Long: `Fetches and displays Gemini API billing information from a BigQuery billing export table.

This command requires a BigQuery billing export table containing detailed usage cost data to be enabled for your billing account. 

To set up billing export:
1. Go to the Google Cloud Console Billing section
2. Select your billing account
3. Click "Billing export"
4. Enable "Detailed usage cost" export to BigQuery
5. Note the dataset and table IDs created`,
		RunE: runQueryBilling,
	}

	// Get defaults from config
	defaultProject := config.GetDefaultProject("")
	defaultDataset := config.GetBillingDatasetID("")
	defaultTable := config.GetBillingTableID("")

	cmd.Flags().StringVarP(&billingProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().StringVarP(&billingDatasetID, "dataset-id", "d", defaultDataset, "BigQuery dataset ID containing billing export")
	cmd.Flags().StringVarP(&billingTableID, "table-id", "t", defaultTable, "BigQuery table ID for billing export")
	cmd.Flags().IntVar(&billingDays, "days", 7, "Number of days to look back")

	// Only mark as required if no defaults are available
	if defaultDataset == "" {
		cmd.MarkFlagRequired("dataset-id")
	}
	if defaultTable == "" {
		cmd.MarkFlagRequired("table-id")
	}

	return cmd
}

func runQueryBilling(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Apply config defaults if flags weren't set
	billingProjectID = config.GetDefaultProject(billingProjectID)
	billingDatasetID = config.GetBillingDatasetID(billingDatasetID)
	billingTableID = config.GetBillingTableID(billingTableID)

	// Validate required fields
	if billingProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	if billingDatasetID == "" {
		return fmt.Errorf("no billing dataset specified. Use --dataset-id flag or set a default with 'gemapi config set billing DATASET_ID TABLE_ID'")
	}

	if billingTableID == "" {
		return fmt.Errorf("no billing table specified. Use --table-id flag or set a default with 'gemapi config set billing DATASET_ID TABLE_ID'")
	}

	// Create BigQuery client
	client, err := gcp.NewBigQueryClient(ctx, billingProjectID)
	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	ulog.Info("Fetching billing data").
		Field("project_id", billingProjectID).
		Field("dataset_id", billingDatasetID).
		Field("table_id", billingTableID).
		Field("days", billingDays).
		Pretty(fmt.Sprintf("Fetching billing data for the last %d days...\n", billingDays)).
		PrettyOnly().
		Log(ctx)

	// Construct query to aggregate results
	query := fmt.Sprintf(`
		SELECT
			sku.description AS sku_description,
			SUM(cost) AS total_cost,
			SUM(usage.amount) AS total_usage_amount,
			usage.unit AS usage_unit,
			currency
		FROM
			`+"`%s.%s.%s`"+`
		WHERE
			service.description = 'Gemini API'
			AND DATE(usage_start_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
		GROUP BY
			sku_description, usage_unit, currency
		ORDER BY
			total_cost DESC
	`, billingProjectID, billingDatasetID, billingTableID, billingDays)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}

	var totalCost float64
	var currency string
	var summaries []BillingSummary
	recordCount := 0

	for {
		var record BillingSummary
		err := it.Next(&record)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading billing summary: %w", err)
		}
		summaries = append(summaries, record)
		totalCost += record.TotalCost
		currency = record.Currency
		recordCount++
	}

	if recordCount == 0 {
		ulog.Info("No billing data found").
			Field("days", billingDays).
			Field("project_id", billingProjectID).
			Field("dataset_id", billingDatasetID).
			Field("table_id", billingTableID).
			Pretty("No billing data found for Gemini API in the specified time range.\n\nPossible reasons:\n- Billing export may not be enabled\n- There may be a delay in billing data availability (up to 24 hours)\n- No Gemini API usage during the specified period").
			PrettyOnly().
			Log(ctx)
		return nil
	}

	var output strings.Builder
	output.WriteString("=== Gemini API Billing Summary ===\n")

	// Show summary by SKU
	output.WriteString("\n=== Cost Summary by SKU ===\n")
	for _, summary := range summaries {
		output.WriteString(fmt.Sprintf("%s\n", summary.SKU))
		output.WriteString(fmt.Sprintf("  Total Usage: %.2f %s\n", summary.TotalUsage, summary.UsageUnit))
		output.WriteString(fmt.Sprintf("  Total Cost: %s %.4f\n", summary.Currency, summary.TotalCost))

		// Calculate unit cost if applicable
		if summary.TotalUsage > 0 {
			unitCost := summary.TotalCost / summary.TotalUsage
			output.WriteString(fmt.Sprintf("  Unit Cost: %s %.6f per %s\n", summary.Currency, unitCost, summary.UsageUnit))
		}
		output.WriteString("\n")
	}

	output.WriteString("=== Total Cost ===\n")
	output.WriteString(fmt.Sprintf("Period: Last %d days\n", billingDays))
	output.WriteString(fmt.Sprintf("Total: %s %.4f\n", currency, totalCost))

	// Daily average
	if billingDays > 0 {
		dailyAvg := totalCost / float64(billingDays)
		output.WriteString(fmt.Sprintf("Daily Average: %s %.4f\n", currency, dailyAvg))

		// Projected monthly cost (30 days)
		monthlyProjection := dailyAvg * 30
		output.WriteString(fmt.Sprintf("Projected Monthly: %s %.2f\n", currency, monthlyProjection))
	}

	ulog.Info("Billing summary").
		Field("days", billingDays).
		Field("total_cost", totalCost).
		Field("currency", currency).
		Field("record_count", recordCount).
		Pretty(output.String()).
		PrettyOnly().
		Log(ctx)

	return nil
}