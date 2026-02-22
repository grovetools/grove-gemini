package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/config"
	"github.com/grovetools/core/tui/components/help"
	"github.com/grovetools/core/tui/keymap"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/grove-gemini/pkg/analytics"
	"github.com/grovetools/grove-gemini/pkg/logging"
)

// queryTuiKeyMap extends the base keymap with custom keybindings
type queryTuiKeyMap struct {
	keymap.Base
	DailyView    key.Binding
	WeeklyView   key.Binding
	MonthlyView  key.Binding
	ToggleMetric key.Binding
	PrevPeriod   key.Binding
	NextPeriod   key.Binding
}

// ShortHelp returns the short help keybindings
func (k queryTuiKeyMap) ShortHelp() []key.Binding {
	baseHelp := k.Base.ShortHelp()
	return append(baseHelp, k.DailyView, k.WeeklyView, k.MonthlyView, k.ToggleMetric, k.PrevPeriod, k.NextPeriod)
}

// FullHelp returns the full help keybindings
func (k queryTuiKeyMap) FullHelp() [][]key.Binding {
	baseHelp := k.Base.FullHelp()
	customKeys := []key.Binding{k.DailyView, k.WeeklyView, k.MonthlyView, k.ToggleMetric, k.PrevPeriod, k.NextPeriod}
	return append(baseHelp, customKeys)
}

// Sections returns grouped sections of key bindings for the full help view.
// Only includes sections that the query TUI actually implements.
func (k queryTuiKeyMap) Sections() []keymap.Section {
	// Customize navigation for table-based TUI
	nav := k.Base.NavigationSection()
	nav.Bindings = []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom}

	return []keymap.Section{
		nav,
		{
			Name:     "Time Frame",
			Bindings: []key.Binding{k.DailyView, k.WeeklyView, k.MonthlyView},
		},
		{
			Name:     "Period Navigation",
			Bindings: []key.Binding{k.PrevPeriod, k.NextPeriod},
		},
		{
			Name:     "Display",
			Bindings: []key.Binding{k.ToggleMetric},
		},
		k.Base.SystemSection(),
	}
}

// Main model for the TUI
type queryTuiModel struct {
	isLoading   bool
	logs        []logging.QueryLog
	buckets     []analytics.Bucket
	totals      analytics.Totals
	timeFrame   time.Duration
	timeOffset  int // Number of periods back from now (0 = current period)
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
func loadLogsCmd(timeFrame time.Duration, offset int) tea.Cmd {
	return func() tea.Msg {
		logger := logging.GetLogger()
		// Calculate the time range based on offset
		endTime := time.Now().Add(-time.Duration(offset) * timeFrame)
		startTime := endTime.Add(-timeFrame)
		logs, err := logger.ReadLogs(startTime, endTime)
		if err != nil {
			return logsLoadedMsg{err: err}
		}
		return logsLoadedMsg{logs: logs}
	}
}

// newQueryTuiKeyMap creates a new keymap with custom bindings
func newQueryTuiKeyMap(cfg *config.Config) queryTuiKeyMap {
	km := queryTuiKeyMap{
		Base: keymap.Load(cfg, "grove-gemini.gemini-query"),
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
		PrevPeriod: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "previous period"),
		),
		NextPeriod: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next period"),
		),
	}

	// Apply TUI-specific overrides from config
	keymap.ApplyTUIOverrides(cfg, "grove-gemini", "gemini-query", &km)

	return km
}

func initialModel() queryTuiModel {
	// Load config for keybinding overrides
	cfg, _ := config.LoadDefault()

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
	keys := newQueryTuiKeyMap(cfg)
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
	return loadLogsCmd(m.timeFrame, m.timeOffset)
}

func (m queryTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, loadLogsCmd(m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.WeeklyView):
			m.timeFrame = 7 * 24 * time.Hour
			m.timeOffset = 0 // Reset to current period
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.MonthlyView):
			m.timeFrame = 30 * 24 * time.Hour
			m.timeOffset = 0 // Reset to current period
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.PrevPeriod):
			m.timeOffset++
			m.isLoading = true
			return m, loadLogsCmd(m.timeFrame, m.timeOffset)
		case key.Matches(msg, m.keys.NextPeriod):
			if m.timeOffset > 0 {
				m.timeOffset--
				m.isLoading = true
				return m, loadLogsCmd(m.timeFrame, m.timeOffset)
			}
			return m, nil
		case key.Matches(msg, m.keys.ToggleMetric):
			if m.plotMetric == "cost" {
				m.plotMetric = "tokens"
			} else {
				m.plotMetric = "cost"
			}
			plotHeight := m.plot.Height
			if plotHeight == 0 {
				plotHeight = 10
			}
			m.plot = NewPlot(m.buckets, m.plotMetric, m.timeFrame, m.width, plotHeight)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetSize(m.width, m.height)
		m.plot.Width = m.width

		// Calculate heights - 50% for table, rest for plot
		titleHeight := 1    // "Gemini API Usage - X View"
		summaryHeight := 1  // Single line summary
		footerHeight := 1   // Help footer

		availableHeight := m.height - titleHeight - summaryHeight - footerHeight - 2
		plotHeight := availableHeight / 2
		tableHeight := availableHeight - plotHeight

		m.plot.Height = plotHeight
		m.table.SetHeight(tableHeight)
		return m, nil
	case logsLoadedMsg:
		m.isLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.logs = msg.logs

		// Aggregate logs - use same time range as loadLogsCmd
		endTime := time.Now().Add(-time.Duration(m.timeOffset) * m.timeFrame)
		startTime := endTime.Add(-m.timeFrame)

		// Calculate bucket size based on time frame
		// Daily view: 20-minute buckets (3x more granular)
		// Weekly/Monthly: Keep original granularity
		var bucketSize time.Duration
		if m.timeFrame == 24*time.Hour {
			bucketSize = m.timeFrame / 72 // 20-minute buckets for daily view
		} else {
			bucketSize = m.timeFrame / 24 // Original granularity for weekly/monthly
		}

		m.buckets = analytics.AggregateLogs(m.logs, bucketSize, startTime, endTime)
		m.totals = analytics.CalculateTotals(m.buckets)

		// Create plot with current dimensions
		plotHeight := m.plot.Height
		if plotHeight == 0 {
			plotHeight = 10 // Default height
		}
		m.plot = NewPlot(m.buckets, m.plotMetric, m.timeFrame, m.width, plotHeight)

		// Populate table
		var rows []table.Row
		for _, log := range m.logs {
			status := "*"
			if !log.Success {
				status = "x"
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
	// Ultra-compact single-line summary with no boxes
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.DefaultTheme.Colors.Cyan).
		Bold(true)

	cost := fmt.Sprintf("%s $%.2f", titleStyle.Render("Cost:"), m.totals.TotalCost)
	tokens := fmt.Sprintf("%s %dK", titleStyle.Render("Tokens:"), m.totals.TotalTokens/1000)
	requests := fmt.Sprintf("%s %d", titleStyle.Render("Requests:"), m.totals.TotalRequests)
	errors := fmt.Sprintf("%s %.1f%%", titleStyle.Render("Errors:"), m.totals.ErrorRate)

	return fmt.Sprintf("%s  │  %s  │  %s  │  %s", cost, tokens, requests, errors)
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

	header := titleStyle.Render(fmt.Sprintf("Gemini API Usage - %s View%s", timeFrameLabel, dateRange))

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
