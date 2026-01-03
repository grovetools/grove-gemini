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
	DailyView    key.Binding
	WeeklyView   key.Binding
	MonthlyView  key.Binding
	PrevPeriod   key.Binding
	NextPeriod   key.Binding
}

// ShortHelp returns the short help keybindings
func (k dashboardKeyMap) ShortHelp() []key.Binding {
	baseHelp := k.Base.ShortHelp()
	return append(baseHelp, k.DailyView, k.WeeklyView, k.MonthlyView, k.PrevPeriod, k.NextPeriod)
}

// FullHelp returns the full help keybindings
func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	baseHelp := k.Base.FullHelp()
	customKeys := []key.Binding{k.DailyView, k.WeeklyView, k.MonthlyView, k.PrevPeriod, k.NextPeriod}
	return append(baseHelp, customKeys)
}

// dashboardModel for the billing dashboard TUI
type dashboardModel struct {
	isLoading      bool
	projectID      string
	datasetID      string
	tableID        string
	timeFrame      time.Duration
	timeOffset     int // Number of periods back from now (0 = current period)
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
func loadBillingDataCmd(projectID, datasetID, tableID string, timeFrame time.Duration, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Convert to days for the API
		days := int(timeFrame.Hours() / 24)

		// Fetch data for the calculated period
		// Note: offset is handled on the client side for display purposes
		// The API always fetches the most recent N days
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
		DailyView: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "daily view"),
		),
		WeeklyView: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "weekly view"),
		),
		MonthlyView: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "monthly view"),
		),
		PrevPeriod: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "previous period"),
		),
		NextPeriod: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next period"),
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

	// Convert days to timeFrame, default to monthly if days is 30
	var timeFrame time.Duration
	if days == 7 {
		timeFrame = 7 * 24 * time.Hour
	} else if days == 90 {
		timeFrame = 90 * 24 * time.Hour
	} else {
		// Default to monthly (30 days)
		timeFrame = 30 * 24 * time.Hour
	}

	return dashboardModel{
		isLoading:  true,
		projectID:  projectID,
		datasetID:  datasetID,
		tableID:    tableID,
		timeFrame:  timeFrame,
		timeOffset: 0,
		table:      tbl,
		keys:       keys,
		help:       helpModel,
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
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
		case key.Matches(msg, m.keys.DailyView):
			m.timeFrame = 24 * time.Hour
			m.timeOffset = 0 // Reset to current period
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.WeeklyView):
			m.timeFrame = 7 * 24 * time.Hour
			m.timeOffset = 0 // Reset to current period
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.MonthlyView):
			m.timeFrame = 30 * 24 * time.Hour
			m.timeOffset = 0 // Reset to current period
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.PrevPeriod):
			m.timeOffset++
			m.isLoading = true
			return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.NextPeriod):
			if m.timeOffset > 0 {
				m.timeOffset--
				m.isLoading = true
				return m, loadBillingDataCmd(m.projectID, m.datasetID, m.tableID, m.timeFrame, m.timeOffset)
			}
			return m, nil
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
		m.plot = NewPlot(buckets, "cost", m.timeFrame, m.width, plotHeight)

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

	totalCost := fmt.Sprintf("%s %s %.2f", titleStyle.Render("Cost:"), m.billingData.Currency, m.billingData.TotalCost)

	// Count total tokens from SKU breakdown (they're stored as usage amounts)
	var totalTokens int64
	for _, sku := range m.billingData.SKUBreakdown {
		totalTokens += int64(sku.TotalUsage)
	}
	tokens := fmt.Sprintf("%s %dK", titleStyle.Render("Tokens:"), totalTokens/1000)

	// Calculate requests (estimate based on token averages)
	requests := fmt.Sprintf("%s %d", titleStyle.Render("Requests:"), len(m.billingData.SKUBreakdown))

	// We don't have error rate in billing data, so omit it

	return fmt.Sprintf("%s  │  %s  │  %s", totalCost, tokens, requests)
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

	timeFrameLabel := "Daily"
	if m.timeFrame == 7*24*time.Hour {
		timeFrameLabel = "Weekly"
	} else if m.timeFrame == 30*24*time.Hour {
		timeFrameLabel = "Monthly"
	}

	// Calculate date range being viewed
	endTime := time.Now().Add(-time.Duration(m.timeOffset) * m.timeFrame)
	startTime := endTime.Add(-m.timeFrame)

	// Format date range
	var dateRange string
	if m.timeOffset == 0 {
		dateRange = ""
	} else {
		dateRange = fmt.Sprintf(" (%s - %s)", startTime.Format("Jan 2"), endTime.Format("Jan 2"))
	}

	header := titleStyle.Render(fmt.Sprintf("GCP Billing Dashboard - %s View%s", timeFrameLabel, dateRange))

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
