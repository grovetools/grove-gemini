package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-gemini/pkg/analytics"
)

type PlotModel struct {
	Buckets   []analytics.Bucket
	Metric    string // "cost" or "tokens"
	TimeFrame time.Duration
	Width     int
	Height    int
}

func NewPlot(buckets []analytics.Bucket, metric string, timeFrame time.Duration, width, height int) PlotModel {
	return PlotModel{Buckets: buckets, Metric: metric, TimeFrame: timeFrame, Width: width, Height: height}
}

func (p PlotModel) View() string {
	if len(p.Buckets) == 0 || p.Width < 20 || p.Height < 5 {
		// Not enough space to render a meaningful chart with axes.
		return ""
	}

	return p.renderChartWithAxes()
}

func (p PlotModel) renderSparkline() string {
	// Find max value for scaling
	var maxValue float64
	for _, bucket := range p.Buckets {
		var val float64
		if p.Metric == "cost" {
			val = bucket.TotalCost
		} else {
			val = float64(bucket.TotalTokens)
		}
		if val > maxValue {
			maxValue = val
		}
	}

	if maxValue == 0 {
		maxValue = 1
	}

	// Sparkline characters from low to high
	sparks := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var plot []string
	for i := 0; i < p.Width; i++ {
		bucketIndex := i * len(p.Buckets) / p.Width
		if bucketIndex >= len(p.Buckets) {
			bucketIndex = len(p.Buckets) - 1
		}
		bucket := p.Buckets[bucketIndex]

		var val float64
		if p.Metric == "cost" {
			val = bucket.TotalCost
		} else {
			val = float64(bucket.TotalTokens)
		}

		// Normalize value and select spark character
		normalized := val / maxValue
		sparkIndex := int(normalized * float64(len(sparks)-1))
		plot = append(plot, sparks[sparkIndex])
	}

	// Render plot with theme
	plotStyle := lipgloss.NewStyle().Foreground(theme.DefaultTheme.Colors.Cyan)
	return plotStyle.Render(strings.Join(plot, ""))
}

func (p PlotModel) renderBarChart() string {
	// Find max value for scaling
	var maxValue float64
	for _, bucket := range p.Buckets {
		var val float64
		if p.Metric == "cost" {
			val = bucket.TotalCost
		} else {
			val = float64(bucket.TotalTokens)
		}
		if val > maxValue {
			maxValue = val
		}
	}

	if maxValue == 0 {
		maxValue = 1
	}

	// Create multi-line bar chart
	var lines []string
	for row := p.Height - 1; row >= 0; row-- {
		var line strings.Builder
		threshold := float64(row+1) / float64(p.Height)

		for i := 0; i < p.Width; i++ {
			bucketIndex := i * len(p.Buckets) / p.Width
			if bucketIndex >= len(p.Buckets) {
				bucketIndex = len(p.Buckets) - 1
			}
			bucket := p.Buckets[bucketIndex]

			var val float64
			if p.Metric == "cost" {
				val = bucket.TotalCost
			} else {
				val = float64(bucket.TotalTokens)
			}

			normalized := val / maxValue
			if normalized >= threshold {
				line.WriteString("█")
			} else {
				line.WriteString(" ")
			}
		}
		lines = append(lines, line.String())
	}

	plotStyle := lipgloss.NewStyle().Foreground(theme.DefaultTheme.Colors.Cyan)
	return plotStyle.Render(strings.Join(lines, "\n"))
}

// formatYAxisLabel creates a formatted label for the Y-axis.
func formatYAxisLabel(value float64, metric string) string {
	if metric == "cost" {
		if value < 10 {
			return fmt.Sprintf("$%.2f", value)
		}
		return fmt.Sprintf("$%.0f", value)
	}
	// "tokens"
	return formatTokenCountSimple(value)
}

// formatTokenCountSimple formats large token counts with K/M suffixes.
func formatTokenCountSimple(tokens float64) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", tokens/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.0fK", tokens/1_000)
	}
	return fmt.Sprintf("%.0f", tokens)
}

// generateXAxisLabels creates the tick marks and labels for the X-axis.
func generateXAxisLabels(buckets []analytics.Bucket, timeFrame time.Duration, width int) (map[int]struct{}, string) {
	if len(buckets) == 0 {
		return nil, ""
	}

	ticks := make(map[int]struct{})
	labels := make([]rune, width)
	for i := range labels {
		labels[i] = ' '
	}

	// Determine number of labels based on typical label length
	var typicalLabelLen int
	switch timeFrame {
	case 24 * time.Hour: // Daily - "15:04" = 5 chars
		typicalLabelLen = 5
	case 7 * 24 * time.Hour: // Weekly - "Mon Jan 2" = 10 chars
		typicalLabelLen = 11
	default: // Monthly - "Jan 2" = 5 chars
		typicalLabelLen = 6
	}

	// Space labels to avoid overlap (add padding between labels)
	minSpacing := typicalLabelLen + 3
	numLabels := width / minSpacing
	if numLabels == 0 {
		numLabels = 1
	}
	if numLabels > 10 {
		numLabels = 10 // Cap at 10 labels for readability
	}

	lastLabelEnd := -1                     // Track the end position of the last placed label
	usedLabels := make(map[string]bool)    // Track which label texts have been placed

	for i := 0; i <= numLabels; i++ {
		pos := i * (width - 1) / numLabels
		bucketIndex := pos * len(buckets) / width
		if bucketIndex >= len(buckets) {
			bucketIndex = len(buckets) - 1
		}

		bucket := buckets[bucketIndex]

		var label string
		switch timeFrame {
		case 24 * time.Hour: // Daily
			label = bucket.StartTime.Format("15:04")
		case 7 * 24 * time.Hour: // Weekly
			label = bucket.StartTime.Format("Mon Jan 2")
		default: // Monthly
			label = bucket.StartTime.Format("Jan 2")
		}

		// Skip if we've already placed this label text
		if usedLabels[label] {
			continue
		}

		// Center label under tick mark
		labelLen := len(label)
		startPos := pos - labelLen/2

		// Ensure label fits within bounds
		if startPos < 0 {
			startPos = 0
		}
		if startPos+labelLen > width {
			continue // Skip if can't fit
		}

		// Check for overlap with previous label (need at least 1 space gap)
		if startPos <= lastLabelEnd {
			continue // Skip if would overlap
		}

		// Place label
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

func (p PlotModel) renderChartWithAxes() string {
	const yAxisWidth = 8  // for labels like "$100.00"
	const xAxisHeight = 2 // for ticks and labels

	chartWidth := p.Width - yAxisWidth
	chartHeight := p.Height - xAxisHeight

	if chartWidth < 10 || chartHeight < 3 {
		return "Chart too small to render."
	}

	// Find max value for Y-axis scaling
	var maxValue float64
	for _, bucket := range p.Buckets {
		var val float64
		if p.Metric == "cost" {
			val = bucket.TotalCost
		} else {
			val = float64(bucket.TotalTokens)
		}
		if val > maxValue {
			maxValue = val
		}
	}
	if maxValue == 0 {
		maxValue = 1
	}

	// --- Prepare Data for Rendering ---

	// Bar heights for each column in the chart
	barHeights := make([]int, chartWidth)
	for i := 0; i < chartWidth; i++ {
		bucketIndex := i * len(p.Buckets) / chartWidth
		if bucketIndex >= len(p.Buckets) {
			bucketIndex = len(p.Buckets) - 1
		}
		bucket := p.Buckets[bucketIndex]

		var val float64
		if p.Metric == "cost" {
			val = bucket.TotalCost
		} else {
			val = float64(bucket.TotalTokens)
		}
		normalized := val / maxValue
		barHeights[i] = int(normalized * float64(chartHeight))
	}

	// Y-axis labels
	yLabels := make(map[int]string)
	yLabels[chartHeight-1] = formatYAxisLabel(maxValue, p.Metric)
	yLabels[0] = formatYAxisLabel(0, p.Metric)
	if chartHeight > 4 {
		yLabels[chartHeight/2] = formatYAxisLabel(maxValue/2, p.Metric)
	}

	// X-axis labels and ticks
	xTicks, xLabels := generateXAxisLabels(p.Buckets, p.TimeFrame, chartWidth)

	// --- Assemble the Chart ---
	var b strings.Builder
	plotStyle := lipgloss.NewStyle().Foreground(theme.DefaultTheme.Colors.Cyan)
	axisStyle := lipgloss.NewStyle().Foreground(theme.DefaultTheme.Colors.MutedText)

	// Render chart body and Y-axis
	for row := chartHeight - 1; row >= 0; row-- {
		// Y-axis label (right-aligned)
		if label, ok := yLabels[row]; ok {
			b.WriteString(axisStyle.Render(fmt.Sprintf("%*s", yAxisWidth-1, label)))
		} else {
			b.WriteString(strings.Repeat(" ", yAxisWidth-1))
		}

		// Vertical line separator
		b.WriteString(axisStyle.Render("│"))

		// Chart bars for this row
		for col := 0; col < chartWidth; col++ {
			if barHeights[col] > row {
				b.WriteString(plotStyle.Render("█"))
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}

	// Render X-axis corner and line
	b.WriteString(strings.Repeat(" ", yAxisWidth-1))
	b.WriteString(axisStyle.Render("└"))
	for i := 0; i < chartWidth; i++ {
		if _, ok := xTicks[i]; ok {
			b.WriteString(axisStyle.Render("┴"))
		} else {
			b.WriteString(axisStyle.Render("─"))
		}
	}
	b.WriteString("\n")

	// Render X-axis labels
	b.WriteString(strings.Repeat(" ", yAxisWidth))
	b.WriteString(xLabels)

	return b.String()
}
