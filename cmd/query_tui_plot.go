package cmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-gemini/pkg/analytics"
)

type PlotModel struct {
	Buckets []analytics.Bucket
	Metric  string // "cost" or "tokens"
	Width   int
	Height  int
}

func NewPlot(buckets []analytics.Bucket, metric string, width, height int) PlotModel {
	return PlotModel{Buckets: buckets, Metric: metric, Width: width, Height: height}
}

func (p PlotModel) View() string {
	if len(p.Buckets) == 0 {
		return ""
	}

	if p.Height <= 1 {
		// Single line sparkline for compact mode
		return p.renderSparkline()
	}

	// Multi-line bar chart
	return p.renderBarChart()
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
