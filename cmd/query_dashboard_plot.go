package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-gemini/pkg/analytics"
)

// StackedPlotModel represents a plot with stacked bars by SKU
type StackedPlotModel struct {
	DailySummaries []analytics.DailyBillingSummary
	TimeFrame      time.Duration
	Width          int
	Height         int
	TopSKUs        []string // Top SKUs to show in stacks
}

// NewStackedPlot creates a new stacked plot from billing data
func NewStackedPlot(summaries []analytics.DailyBillingSummary, timeFrame time.Duration, width, height int) StackedPlotModel {
	// Identify top SKUs by total cost
	skuTotals := make(map[string]float64)
	for _, day := range summaries {
		for _, sku := range day.SKUs {
			skuTotals[sku.SKU] += sku.TotalCost
		}
	}

	// Sort and take top 5 SKUs
	type skuCost struct {
		sku  string
		cost float64
	}
	var sorted []skuCost
	for sku, cost := range skuTotals {
		sorted = append(sorted, skuCost{sku, cost})
	}
	// Simple bubble sort
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].cost > sorted[i].cost {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Take top 5
	topSKUs := make([]string, 0, 5)
	for i := 0; i < len(sorted) && i < 5; i++ {
		topSKUs = append(topSKUs, sorted[i].sku)
	}

	return StackedPlotModel{
		DailySummaries: summaries,
		TimeFrame:      timeFrame,
		Width:          width,
		Height:         height,
		TopSKUs:        topSKUs,
	}
}

func (p StackedPlotModel) View() string {
	if len(p.DailySummaries) == 0 || p.Width < 20 || p.Height < 5 {
		return ""
	}

	return p.renderStackedChart()
}

// getSKUColor returns a color/style for a given SKU index
func getSKUColor(index int) lipgloss.TerminalColor {
	colors := []lipgloss.TerminalColor{
		theme.DefaultTheme.Colors.Cyan,
		theme.DefaultTheme.Colors.Green,
		theme.DefaultTheme.Colors.Yellow,
		theme.DefaultTheme.Colors.Pink,
		theme.DefaultTheme.Colors.Blue,
	}
	return colors[index%len(colors)]
}

func (p StackedPlotModel) renderStackedChart() string {
	chartHeight := p.Height - 2 // Reserve space for Y-axis labels
	chartWidth := p.Width - 7   // Reserve space for Y-axis

	// Find max total cost for any day
	var maxValue float64
	for _, day := range p.DailySummaries {
		if day.TotalCost > maxValue {
			maxValue = day.TotalCost
		}
	}

	if maxValue == 0 {
		maxValue = 1
	}

	// Build the chart grid
	var lines []string
	for row := chartHeight - 1; row >= 0; row-- {
		var line strings.Builder

		// Y-axis label
		yValue := maxValue * float64(row+1) / float64(chartHeight)
		label := formatYAxisLabel(yValue, "cost")
		line.WriteString(fmt.Sprintf("%6s│", label))

		// Render bars
		for col := 0; col < chartWidth; col++ {
			dayIndex := col * len(p.DailySummaries) / chartWidth
			if dayIndex >= len(p.DailySummaries) {
				dayIndex = len(p.DailySummaries) - 1
			}
			day := p.DailySummaries[dayIndex]

			// Calculate which SKU (if any) should be shown at this height
			threshold := maxValue * float64(row+1) / float64(chartHeight)
			accumulatedCost := 0.0

			skuIndex := -1
			for i, topSKU := range p.TopSKUs {
				// Find this SKU in the day's breakdown
				for _, sku := range day.SKUs {
					if sku.SKU == topSKU {
						if accumulatedCost+sku.TotalCost >= threshold {
							skuIndex = i
							break
						}
						accumulatedCost += sku.TotalCost
						break
					}
				}
				if skuIndex >= 0 {
					break
				}
			}

			if skuIndex >= 0 {
				// Render with SKU color
				color := getSKUColor(skuIndex)
				style := lipgloss.NewStyle().Foreground(color)
				line.WriteString(style.Render("█"))
			} else if accumulatedCost >= threshold {
				// Other SKUs (not in top 5)
				style := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
				line.WriteString(style.Render("█"))
			} else {
				line.WriteString(" ")
			}
		}

		lines = append(lines, line.String())
	}

	// Add X-axis
	ticks, labelString := p.generateXAxisLabels(chartWidth)
	xAxis := "       └"
	for i := 0; i < chartWidth; i++ {
		if _, hasTick := ticks[i]; hasTick {
			xAxis += "┴"
		} else {
			xAxis += "─"
		}
	}
	lines = append(lines, xAxis)
	lines = append(lines, "        "+labelString)

	// Add legend
	legend := p.renderLegend()
	if legend != "" {
		lines = append(lines, "")
		lines = append(lines, legend)
	}

	return strings.Join(lines, "\n")
}

func (p StackedPlotModel) renderLegend() string {
	if len(p.TopSKUs) == 0 {
		return ""
	}

	var legendItems []string
	for i, sku := range p.TopSKUs {
		color := getSKUColor(i)
		style := lipgloss.NewStyle().Foreground(color)
		// Shorten SKU names for legend
		shortName := sku
		if len(shortName) > 30 {
			shortName = shortName[:27] + "..."
		}
		legendItems = append(legendItems, style.Render("█")+" "+shortName)
	}

	// Format legend in columns if needed
	if len(legendItems) <= 3 {
		return "        " + strings.Join(legendItems, "  │  ")
	}

	// Split into two rows
	row1 := strings.Join(legendItems[:3], "  │  ")
	row2 := strings.Join(legendItems[3:], "  │  ")
	return "        " + row1 + "\n        " + row2
}

func (p StackedPlotModel) generateXAxisLabels(width int) (map[int]struct{}, string) {
	if len(p.DailySummaries) == 0 {
		return nil, ""
	}

	ticks := make(map[int]struct{})
	labels := make([]rune, width)
	for i := range labels {
		labels[i] = ' '
	}

	// Determine number of labels based on typical label length
	var typicalLabelLen int
	switch p.TimeFrame {
	case 24 * time.Hour: // Daily - "15:04" = 5 chars
		typicalLabelLen = 5
	case 7 * 24 * time.Hour: // Weekly - "Mon Jan 2" = 10 chars
		typicalLabelLen = 11
	default: // Monthly - "Jan 2" = 5 chars
		typicalLabelLen = 6
	}

	minSpacing := typicalLabelLen + 3
	numLabels := width / minSpacing
	if numLabels == 0 {
		numLabels = 1
	}
	if numLabels > 10 {
		numLabels = 10
	}

	lastLabelEnd := -1
	usedLabels := make(map[string]bool)

	for i := 0; i <= numLabels; i++ {
		pos := i * (width - 1) / numLabels
		dayIndex := pos * len(p.DailySummaries) / width
		if dayIndex >= len(p.DailySummaries) {
			dayIndex = len(p.DailySummaries) - 1
		}

		day := p.DailySummaries[dayIndex]

		var label string
		switch p.TimeFrame {
		case 24 * time.Hour:
			label = day.Date.Format("15:04")
		case 7 * 24 * time.Hour:
			label = day.Date.Format("Mon Jan 2")
		default:
			label = day.Date.Format("Jan 2")
		}

		if usedLabels[label] {
			continue
		}

		labelLen := len(label)
		startPos := pos - labelLen/2

		if startPos < 0 {
			startPos = 0
		}
		if startPos+labelLen > width {
			continue
		}

		if startPos <= lastLabelEnd {
			continue
		}

		for j, char := range label {
			if startPos+j < width {
				labels[startPos+j] = char
			}
		}
		ticks[pos] = struct{}{}
		lastLabelEnd = startPos + labelLen
		usedLabels[label] = true
	}

	return ticks, string(labels)
}
