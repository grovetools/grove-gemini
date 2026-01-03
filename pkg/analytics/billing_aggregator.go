package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/mattsolo1/grove-gemini/pkg/gcp"
	"google.golang.org/api/iterator"
)

// DailyBillingSummary represents aggregated billing data for a single day
type DailyBillingSummary struct {
	Date       time.Time
	TotalCost  float64
	TotalUsage float64
	SKUs       []SKUCostBreakdown
}

// SKUCostBreakdown represents cost and usage for a specific SKU
type SKUCostBreakdown struct {
	SKU        string
	TotalCost  float64
	TotalUsage float64
	UsageUnit  string
	Percentage float64 // Percentage of total cost
}

// BillingData represents the complete billing dataset
type BillingData struct {
	DailySummaries []DailyBillingSummary
	TotalCost      float64
	Currency       string
	SKUBreakdown   []SKUCostBreakdown
}

type billingQueryRow struct {
	Date       string  `bigquery:"date"`
	SKU        string  `bigquery:"sku_description"`
	TotalCost  float64 `bigquery:"total_cost"`
	TotalUsage float64 `bigquery:"total_usage_amount"`
	UsageUnit  string  `bigquery:"usage_unit"`
	Currency   string  `bigquery:"currency"`
}

// FetchBillingData retrieves and aggregates billing data from BigQuery
func FetchBillingData(ctx context.Context, projectID, datasetID, tableID string, days int) (*BillingData, error) {
	client, err := gcp.NewBigQueryClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}
	defer client.Close()

	// Query for daily aggregated data
	query := fmt.Sprintf(`
		SELECT
			DATE(usage_start_time) AS date,
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
			date, sku_description, usage_unit, currency
		ORDER BY
			date ASC, total_cost DESC
	`, projectID, datasetID, tableID, days)

	q := client.Query(query)
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}

	// Process query results
	dailyMap := make(map[string]*DailyBillingSummary)
	skuTotals := make(map[string]*SKUCostBreakdown)
	var totalCost float64
	var currency string

	for {
		var row billingQueryRow
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading billing data: %w", err)
		}

		// Parse date
		date, err := time.Parse("2006-01-02", row.Date)
		if err != nil {
			continue
		}

		// Aggregate by day
		if _, exists := dailyMap[row.Date]; !exists {
			dailyMap[row.Date] = &DailyBillingSummary{
				Date: date,
				SKUs: []SKUCostBreakdown{},
			}
		}
		dailyMap[row.Date].TotalCost += row.TotalCost
		dailyMap[row.Date].TotalUsage += row.TotalUsage
		dailyMap[row.Date].SKUs = append(dailyMap[row.Date].SKUs, SKUCostBreakdown{
			SKU:        row.SKU,
			TotalCost:  row.TotalCost,
			TotalUsage: row.TotalUsage,
			UsageUnit:  row.UsageUnit,
		})

		// Aggregate SKU totals
		if _, exists := skuTotals[row.SKU]; !exists {
			skuTotals[row.SKU] = &SKUCostBreakdown{
				SKU:       row.SKU,
				UsageUnit: row.UsageUnit,
			}
		}
		skuTotals[row.SKU].TotalCost += row.TotalCost
		skuTotals[row.SKU].TotalUsage += row.TotalUsage

		totalCost += row.TotalCost
		currency = row.Currency
	}

	// Convert maps to slices
	var dailySummaries []DailyBillingSummary
	for _, summary := range dailyMap {
		dailySummaries = append(dailySummaries, *summary)
	}

	// Sort daily summaries by date
	// (Note: Already sorted by query ORDER BY)

	// Calculate SKU percentages and convert to slice
	var skuBreakdown []SKUCostBreakdown
	for _, sku := range skuTotals {
		if totalCost > 0 {
			sku.Percentage = (sku.TotalCost / totalCost) * 100
		}
		skuBreakdown = append(skuBreakdown, *sku)
	}

	// Sort SKU breakdown by cost (descending)
	for i := 0; i < len(skuBreakdown)-1; i++ {
		for j := i + 1; j < len(skuBreakdown); j++ {
			if skuBreakdown[j].TotalCost > skuBreakdown[i].TotalCost {
				skuBreakdown[i], skuBreakdown[j] = skuBreakdown[j], skuBreakdown[i]
			}
		}
	}

	return &BillingData{
		DailySummaries: dailySummaries,
		TotalCost:      totalCost,
		Currency:       currency,
		SKUBreakdown:   skuBreakdown,
	}, nil
}
