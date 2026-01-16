package analytics

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/civil"
	"github.com/grovetools/grove-gemini/pkg/gcp"
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
	Date       civil.Date `bigquery:"date"`
	SKU        string     `bigquery:"sku_description"`
	TotalCost  float64    `bigquery:"total_cost"`
	TotalUsage float64    `bigquery:"total_usage_amount"`
	UsageUnit  string     `bigquery:"usage_unit"`
	Currency   string     `bigquery:"currency"`
}

// FetchBillingData retrieves and aggregates billing data from BigQuery
func FetchBillingData(ctx context.Context, projectID, datasetID, tableID string, days, offsetDays int) (*BillingData, error) {
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
			AND DATE(usage_start_time) BETWEEN DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY) AND DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
		GROUP BY
			date, sku_description, usage_unit, currency
		ORDER BY
			date ASC, total_cost DESC
	`, projectID, datasetID, tableID, days+offsetDays, offsetDays)

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

		// Convert civil.Date to time.Time
		date := time.Date(row.Date.Year, time.Month(row.Date.Month), row.Date.Day, 0, 0, 0, 0, time.UTC)
		dateKey := row.Date.String() // Use string representation as map key

		// Aggregate by day
		if _, exists := dailyMap[dateKey]; !exists {
			dailyMap[dateKey] = &DailyBillingSummary{
				Date: date,
				SKUs: []SKUCostBreakdown{},
			}
		}
		dailyMap[dateKey].TotalCost += row.TotalCost
		dailyMap[dateKey].TotalUsage += row.TotalUsage
		dailyMap[dateKey].SKUs = append(dailyMap[dateKey].SKUs, SKUCostBreakdown{
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

	// Sort daily summaries by date (ascending)
	// Note: Maps are unordered in Go, so we must sort after conversion
	for i := 0; i < len(dailySummaries)-1; i++ {
		for j := i + 1; j < len(dailySummaries); j++ {
			if dailySummaries[j].Date.Before(dailySummaries[i].Date) {
				dailySummaries[i], dailySummaries[j] = dailySummaries[j], dailySummaries[i]
			}
		}
	}

	// Fill in missing days with zero-cost entries for accurate timeline visualization
	if len(dailySummaries) > 0 {
		filledSummaries := []DailyBillingSummary{}

		// Calculate the expected date range
		endDate := time.Now().Add(-time.Duration(offsetDays) * 24 * time.Hour)
		startDate := endDate.Add(-time.Duration(days) * 24 * time.Hour)

		// Create a map for quick lookup
		summaryMap := make(map[string]DailyBillingSummary)
		for _, summary := range dailySummaries {
			summaryMap[summary.Date.Format("2006-01-02")] = summary
		}

		// Fill in all days in the range
		for d := startDate; !d.After(endDate); d = d.Add(24 * time.Hour) {
			dateKey := d.Format("2006-01-02")
			if summary, exists := summaryMap[dateKey]; exists {
				filledSummaries = append(filledSummaries, summary)
			} else {
				// Add zero-cost entry for missing day
				filledSummaries = append(filledSummaries, DailyBillingSummary{
					Date:       d,
					TotalCost:  0,
					TotalUsage: 0,
					SKUs:       []SKUCostBreakdown{},
				})
			}
		}

		dailySummaries = filledSummaries
	}

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
