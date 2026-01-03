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
