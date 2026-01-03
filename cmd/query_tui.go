package cmd

import (
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
	"github.com/mattsolo1/grove-gemini/pkg/logging"
)

// queryTuiKeyMap extends the base keymap with custom keybindings
type queryTuiKeyMap struct {
	keymap.Base
	DailyView    key.Binding
	WeeklyView   key.Binding
	MonthlyView  key.Binding
	ToggleMetric key.Binding
}

// Main model for the TUI
type queryTuiModel struct {
	isLoading   bool
	logs        []logging.QueryLog
	buckets     []analytics.Bucket
	totals      analytics.Totals
	timeFrame   time.Duration
	table       table.Model
	plot        PlotModel
	plotMetric  string // "cost" or "tokens"
	keys        queryTuiKeyMap
	help        help.Model
	err         error
	width       int
	height      int
}

// Message for when logs are loaded
type logsLoadedMsg struct {
	logs []logging.QueryLog
	err  error
}

// Command to load logs
func loadLogsCmd(timeFrame time.Duration) tea.Cmd {
	return func() tea.Msg {
		logger := logging.GetLogger()
		logs, err := logger.ReadLogs(time.Now().Add(-timeFrame), time.Now())
		if err != nil {
			return logsLoadedMsg{err: err}
		}
		return logsLoadedMsg{logs: logs}
	}
}

// newQueryTuiKeyMap creates a new keymap with custom bindings
func newQueryTuiKeyMap() queryTuiKeyMap {
	return queryTuiKeyMap{
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
		ToggleMetric: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle metric"),
		),
	}
}

func initialModel() queryTuiModel {
	// Define table columns
	columns := []table.Column{
		{Title: "Timestamp", Width: 15},
		{Title: "Model", Width: 15},
		{Title: "Caller", Width: 15},
		{Title: "Total Tokens", Width: 12},
		{Title: "Cost", Width: 12},
		{Title: "Time", Width: 10},
		{Title: "Status", Width: 8},
	}
	tbl := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithHeight(10))

	// Setup keys and help
	keys := newQueryTuiKeyMap()
	helpModel := help.New(keys)

	return queryTuiModel{
		isLoading:  true,
		timeFrame:  24 * time.Hour,
		plotMetric: "cost",
		table:      tbl,
		keys:       keys,
		help:       helpModel,
	}
}

func (m queryTuiModel) Init() tea.Cmd {
	return loadLogsCmd(m.timeFrame)
}

func (m queryTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.keys.DailyView):
			m.timeFrame = 24 * time.Hour
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame)
		case key.Matches(msg, m.keys.WeeklyView):
			m.timeFrame = 7 * 24 * time.Hour
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame)
		case key.Matches(msg, m.keys.MonthlyView):
			m.timeFrame = 30 * 24 * time.Hour
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame)
		case key.Matches(msg, m.keys.ToggleMetric):
			if m.plotMetric == "cost" {
				m.plotMetric = "tokens"
			} else {
				m.plotMetric = "cost"
			}
			m.plot = NewPlot(m.buckets, m.plotMetric, m.width, 5)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetSize(m.width, m.height)
		m.plot.Width = m.width - 4 // Account for plot box padding

		// Recalculate table height based on remaining space
		summaryHeight := 3
		plotHeight := 7 // Plot + box
		headerHeight := 1
		footerHeight := 1
		m.table.SetHeight(m.height - summaryHeight - plotHeight - headerHeight - footerHeight)
		return m, nil
	case logsLoadedMsg:
		m.isLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.logs = msg.logs

		// Aggregate logs
		endTime := time.Now()
		startTime := endTime.Add(-m.timeFrame)
		m.buckets = analytics.AggregateLogs(m.logs, m.timeFrame/24, startTime, endTime) // e.g., hourly for 24h
		m.totals = analytics.CalculateTotals(m.buckets)

		// Create plot
		m.plot = NewPlot(m.buckets, m.plotMetric, m.width, 5) // 5 lines high

		// Populate table
		var rows []table.Row
		for _, log := range m.logs {
			status := "✓"
			if !log.Success {
				status = "✗"
			}
			rows = append(rows, table.Row{
				log.Timestamp.Format("15:04:05"),
				log.Model,
				log.Caller,
				fmt.Sprintf("%d", log.TotalTokens),
				fmt.Sprintf("$%.4f", log.EstimatedCost),
				fmt.Sprintf("%.2fs", log.ResponseTime),
				status,
			})
		}
		m.table.SetRows(rows)
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m queryTuiModel) renderSummaryView() string {
	if m.width == 0 {
		m.width = 120 // Fallback width
	}

	boxWidth := (m.width / 3) - 6
	if boxWidth < 20 {
		boxWidth = 20
	}

	// Create style with explicit height to ensure content is visible
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.DefaultTheme.Colors.Cyan).
		Padding(0, 1).
		Width(boxWidth).
		Height(4).  // Explicit height: 2 lines of content + padding
		Align(lipgloss.Left, lipgloss.Top)

	// Format content with proper spacing
	costContent := fmt.Sprintf("\nTotal Cost\n$%.4f\n", m.totals.TotalCost)
	tokenContent := fmt.Sprintf("\nTotal Tokens\n%d K\n", m.totals.TotalTokens/1000)
	requestContent := fmt.Sprintf("\nTotal Requests\n%d (%.1f%% err)\n", m.totals.TotalRequests, m.totals.ErrorRate)

	costBox := boxStyle.Render(costContent)
	tokenBox := boxStyle.Render(tokenContent)
	requestBox := boxStyle.Render(requestContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, costBox, "  ", tokenBox, "  ", requestBox)
}

func (m queryTuiModel) View() string {
	if m.isLoading {
		return "Loading logs..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	// Show full help if requested
	if m.help.ShowAll {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.help.View())
	}

	summaryView := m.renderSummaryView()
	plotView := m.plot.View()
	tableView := m.table.View()
	helpView := m.help.View()

	plotBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(plotView)

	// Add spacing between sections
	return lipgloss.JoinVertical(lipgloss.Left,
		summaryView,
		"",  // Blank line
		plotBox,
		"",  // Blank line
		tableView,
		"",  // Blank line
		helpView,
	)
}

func runQueryTUI() error {
	m := initialModel()
	// Set reasonable default dimensions before first render
	m.width = 120
	m.height = 40

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	return nil
}
