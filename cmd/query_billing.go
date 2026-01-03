package cmd

import (
	"context"
	"fmt"

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

	fmt.Printf("Fetching billing data for the last %d days...\n\n", billingDays)

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

	fmt.Println("=== Gemini API Billing Summary ===")

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
		fmt.Println("No billing data found for Gemini API in the specified time range.")
		fmt.Println("\nPossible reasons:")
		fmt.Println("- Billing export may not be enabled")
		fmt.Println("- There may be a delay in billing data availability (up to 24 hours)")
		fmt.Println("- No Gemini API usage during the specified period")
		return nil
	}

	// Show summary by SKU
	fmt.Println("\n=== Cost Summary by SKU ===")
	for _, summary := range summaries {
		fmt.Printf("%s\n", summary.SKU)
		fmt.Printf("  Total Usage: %.2f %s\n", summary.TotalUsage, summary.UsageUnit)
		fmt.Printf("  Total Cost: %s %.4f\n", summary.Currency, summary.TotalCost)

		// Calculate unit cost if applicable
		if summary.TotalUsage > 0 {
			unitCost := summary.TotalCost / summary.TotalUsage
			fmt.Printf("  Unit Cost: %s %.6f per %s\n", summary.Currency, unitCost, summary.UsageUnit)
		}
		fmt.Println()
	}

	fmt.Printf("=== Total Cost ===\n")
	fmt.Printf("Period: Last %d days\n", billingDays)
	fmt.Printf("Total: %s %.4f\n", currency, totalCost)
	
	// Daily average
	if billingDays > 0 {
		dailyAvg := totalCost / float64(billingDays)
		fmt.Printf("Daily Average: %s %.4f\n", currency, dailyAvg)
		
		// Projected monthly cost (30 days)
		monthlyProjection := dailyAvg * 30
		fmt.Printf("Projected Monthly: %s %.2f\n", currency, monthlyProjection)
	}

	return nil
}