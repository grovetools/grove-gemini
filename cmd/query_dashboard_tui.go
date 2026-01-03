package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-core/tui/components/help"
	"github.com/mattsolo1/grove-core/tui/keymap"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/mattsolo1/grove-gemini/pkg/analytics"
)

// dashboardKeyMap extends the base keymap with custom keybindings
type dashboardKeyMap struct {
	keymap.Base
	SevenDays  key.Binding
	ThirtyDays key.Binding
	NinetyDays key.Binding
}

// ShortHelp returns the short help keybindings
func (k dashboardKeyMap) ShortHelp() []key.Binding {
	baseHelp := k.Base.ShortHelp()
	return append(baseHelp, k.SevenDays, k.ThirtyDays, k.NinetyDays)
}

// FullHelp returns the full help keybindings
func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	baseHelp := k.Base.FullHelp()
	customKeys := []key.Binding{k.SevenDays, k.ThirtyDays, k.NinetyDays}
	return append(baseHelp, customKeys)
}

// dashboardModel for the billing dashboard TUI
type dashboardModel struct {
	isLoading      bool
	projectID      string
	datasetID      string
	tableID        string
	days           int
	billingData    *analytics.BillingData
	table          table.Model
	plot           PlotModel
	keys           dashboardKeyMap
	help           help.Model
	err            error
	width          int
	height         int
}

// Message for when billing data is loaded
type billingDataLoadedMsg struct {
	data *analytics.BillingData
	err  error
}

// Command to load billing data
func loadBillingDataCmd(projectID, datasetID, tableID string, days int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := analytics.FetchBillingData(ctx, projectID, datasetID, tableID, days)
		if err != nil {
			return billingDataLoadedMsg{err: err}
		}
		return billingDataLoadedMsg{data: data}
	}
}

// newDashboardKeyMap creates a new keymap with custom bindings
func newDashboardKeyMap() dashboardKeyMap {
	return dashboardKeyMap{
		Base: keymap.NewBase(),
		SevenDays: key.NewBinding(
			key.WithKeys("7", "w"),
			key.WithHelp("7/w", "7 days"),
		),
		ThirtyDays: key.NewBinding(
			key.WithKeys("3", "m"),
			key.WithHelp("3/m", "30 days"),
		),
		NinetyDays: key.NewBinding(
			key.WithKeys("9", "q"),
			key.WithHelp("9/q", "90 days"),
		),
	}
}

func newDashboardModel(projectID, datasetID, tableID string, days int) dashboardModel {
	// Define table columns
	columns := []table.Column{
		{Title: "SKU", Width: 60},
		{Title: "Total Cost", Width: 15},
		{Title: "Total Usage", Width: 20},
		{Title: "% of Total", Width: 12},
	}
	tbl := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithHeight(10))

	// Setup keys and help
	keys := newDashboardKeyMap()
	helpModel := help.New(keys)

	return dashboardModel{
		isLoading: true,
		projectID: projectID,
		datasetID: datasetID,
		tableID:   tableID,
		days:      days,
		table:     tbl,
		keys:      keys,
		help:      helpModel,
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.days)
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Forward all messages to help component first for handling
	m.help, cmd = m.help.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If help is showing, let it handle all keys except quit
		if m.help.ShowAll {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.Toggle()
			return m, nil
		case key.Matches(msg, m.keys.SevenDays):
			m.days = 7
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.days)
		case key.Matches(msg, m.keys.ThirtyDays):
			m.days = 30
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.days)
		case key.Matches(msg, m.keys.NinetyDays):
			m.days = 90
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.days)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetSize(m.width, m.height)
		m.plot.Width = m.width

		// Calculate heights - 50% for table, rest for plot
		titleHeight := 1    // "GCP Billing Dashboard"
		summaryHeight := 1  // Single line summary
		footerHeight := 1   // Help footer

		availableHeight := m.height - titleHeight - summaryHeight - footerHeight - 2
		plotHeight := availableHeight / 2
		tableHeight := availableHeight - plotHeight

		m.plot.Height = plotHeight
		m.table.SetHeight(tableHeight)
		return m, nil
	case billingDataLoadedMsg:
		m.isLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.billingData = msg.data

		// Convert daily summaries to buckets for plotting
		var buckets []analytics.Bucket
		for _, daily := range msg.data.DailySummaries {
			buckets = append(buckets, analytics.Bucket{
				StartTime:   daily.Date,
				TotalCost:   daily.TotalCost,
				TotalTokens: int64(daily.TotalUsage),
			})
		}

		// Create plot with current dimensions
		plotHeight := m.plot.Height
		if plotHeight == 0 {
			plotHeight = 10 // Default height
		}
		timeFrame := time.Duration(m.days) * 24 * time.Hour
		m.plot = NewPlot(buckets, "cost", timeFrame, m.width, plotHeight)

		// Populate table with SKU breakdown
		var rows []table.Row
		for _, sku := range msg.data.SKUBreakdown {
			rows = append(rows, table.Row{
				sku.SKU,
				fmt.Sprintf("%s %.4f", msg.data.Currency, sku.TotalCost),
				fmt.Sprintf("%.0f %s", sku.TotalUsage, sku.UsageUnit),
				fmt.Sprintf("%.1f%%", sku.Percentage),
			})
		}
		m.table.SetRows(rows)
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m dashboardModel) renderSummaryView() string {
	if m.billingData == nil {
		return ""
	}

	// Ultra-compact single-line summary with no boxes
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Cyan).
		Bold(true)

	totalCost := fmt.Sprintf("%s %s %.2f", titleStyle.Render("Total Cost:"), m.billingData.Currency, m.billingData.TotalCost)
	dailyAvg := fmt.Sprintf("%s %s %.2f", titleStyle.Render("Daily Avg:"), m.billingData.Currency, m.billingData.TotalCost/float64(m.days))
	monthlyProj := fmt.Sprintf("%s %s %.2f", titleStyle.Render("Monthly Proj:"), m.billingData.Currency, (m.billingData.TotalCost/float64(m.days))*30)

	return fmt.Sprintf("%s  │  %s  │  %s", totalCost, dailyAvg, monthlyProj)
}

func (m dashboardModel) View() string {
	if m.isLoading {
		return "Loading billing data..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit", m.err)
	}

	// Show full help if requested
	if m.help.ShowAll {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.help.View())
	}

	// Render header
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Cyan).
		Bold(true)

	header := titleStyle.Render(fmt.Sprintf("GCP Billing Dashboard - Last %d Days", m.days))

	summaryView := m.renderSummaryView()
	plotView := m.plot.View()
	tableView := m.table.View()
	helpView := m.help.View()

	// Ultra-compact layout - no borders, no blank lines
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		summaryView,
		plotView,
		tableView,
		helpView,
	)
}
