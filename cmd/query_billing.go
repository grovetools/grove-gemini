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

type BillingRecord struct {
	Service     string  `bigquery:"service"`
	SKU         string  `bigquery:"sku_description"`
	UsageStart  string  `bigquery:"usage_start_time"`
	UsageAmount float64 `bigquery:"usage_amount"`
	UsageUnit   string  `bigquery:"usage_unit"`
	Cost        float64 `bigquery:"cost"`
	Currency    string  `bigquery:"currency"`
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

	// Get default project from config
	defaultProject := config.GetDefaultProject("")

	cmd.Flags().StringVarP(&billingProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().StringVarP(&billingDatasetID, "dataset-id", "d", "", "BigQuery dataset ID containing billing export (required)")
	cmd.Flags().StringVarP(&billingTableID, "table-id", "t", "", "BigQuery table ID for billing export (required)")
	cmd.Flags().IntVar(&billingDays, "days", 7, "Number of days to look back")
	cmd.MarkFlagRequired("dataset-id")
	cmd.MarkFlagRequired("table-id")

	return cmd
}

func runQueryBilling(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Ensure we have a project ID
	if billingProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	// Create BigQuery client
	client, err := gcp.NewBigQueryClient(ctx, billingProjectID)
	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	fmt.Printf("Fetching billing data for the last %d days...\n\n", billingDays)

	// Construct query
	query := fmt.Sprintf(`
		SELECT
			service.description as service,
			sku.description as sku_description,
			FORMAT_TIMESTAMP("%%Y-%%m-%%d %%H:%%M:%%S", usage_start_time) as usage_start_time,
			usage.amount as usage_amount,
			usage.unit as usage_unit,
			cost,
			currency
		FROM %s.%s.%s
		WHERE service.description = "Generative Language API"
			AND DATE(usage_start_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
		ORDER BY usage_start_time DESC
		LIMIT 1000
	`, billingProjectID, billingDatasetID, billingTableID, billingDays)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}

	fmt.Println("=== Gemini API Billing Data ===")

	var totalCost float64
	var currency string
	skuCosts := make(map[string]float64)
	skuUsage := make(map[string]struct {
		Amount float64
		Unit   string
	})

	recordCount := 0
	for {
		var record BillingRecord
		err := it.Next(&record)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading billing record: %w", err)
		}

		recordCount++
		if recordCount <= 10 { // Show first 10 records as examples
			fmt.Printf("SKU: %s\n", record.SKU)
			fmt.Printf("  Usage: %.2f %s\n", record.UsageAmount, record.UsageUnit)
			fmt.Printf("  Cost: %s %.4f\n", record.Currency, record.Cost)
			fmt.Printf("  Time: %s\n\n", record.UsageStart)
		}

		totalCost += record.Cost
		currency = record.Currency
		skuCosts[record.SKU] += record.Cost

		// Track usage amounts
		if usage, exists := skuUsage[record.SKU]; exists {
			usage.Amount += record.UsageAmount
			skuUsage[record.SKU] = usage
		} else {
			skuUsage[record.SKU] = struct {
				Amount float64
				Unit   string
			}{Amount: record.UsageAmount, Unit: record.UsageUnit}
		}
	}

	if recordCount == 0 {
		fmt.Println("No billing data found for Generative Language API in the specified time range.")
		fmt.Println("\nPossible reasons:")
		fmt.Println("- Billing export may not be enabled")
		fmt.Println("- There may be a delay in billing data availability (up to 24 hours)")
		fmt.Println("- No Gemini API usage during the specified period")
		return nil
	}

	// Show summary
	if recordCount > 10 {
		fmt.Printf("... (%d more records)\n\n", recordCount-10)
	}

	fmt.Println("=== Cost Summary by SKU ===")
	for sku, cost := range skuCosts {
		usage := skuUsage[sku]
		fmt.Printf("%s\n", sku)
		fmt.Printf("  Total Usage: %.2f %s\n", usage.Amount, usage.Unit)
		fmt.Printf("  Total Cost: %s %.4f\n", currency, cost)
		
		// Calculate unit cost if applicable
		if usage.Amount > 0 {
			unitCost := cost / usage.Amount
			fmt.Printf("  Unit Cost: %s %.6f per %s\n", currency, unitCost, usage.Unit)
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