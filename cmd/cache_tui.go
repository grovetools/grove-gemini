package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/grovetools/core/tui/components/help"
	"github.com/grovetools/core/tui/keymap"
	"github.com/grovetools/core/tui/theme"
	"github.com/grovetools/grove-gemini/pkg/gemini"
)

type viewState int

const (
	listView viewState = iota
	inspectView
	helpView
	analyticsView
)

// combinedCacheInfo merges local and API cache data for display and sorting.
type combinedCacheInfo struct {
	LocalInfo *gemini.CacheInfo
	APIInfo   *gemini.CachedContentInfo

	// Pre-computed fields for display and sorting
	Name       string
	Status     string
	IsActive   bool
	CreateTime time.Time
}

// cacheTUIModel represents the state of the TUI
type cacheTUIModel struct {
	client           *gemini.Client
	allCaches        []combinedCacheInfo
	filteredCaches   []combinedCacheInfo
	table            table.Model
	inspectViewport  viewport.Model
	filterInput      textinput.Model
	help             help.Model
	keys             cacheKeyMap
	currentView      viewState
	isLoading        bool
	err              error
	width, height    int
	confirmingDelete bool
	confirmingWipe   bool
	workDir          string
}

// Messages
type cachesLoadedMsg struct{ caches []combinedCacheInfo }
type cacheDeletedMsg struct{}
type cacheWipedMsg struct{}
type errMsg struct{ err error }
type tickMsg time.Time

// cacheKeyMap extends keymap.Base with cache-specific bindings
type cacheKeyMap struct {
	keymap.Base
	Inspect   key.Binding
	Analytics key.Binding
	Delete    key.Binding
	Wipe      key.Binding
	Refresh   key.Binding
}

func newCacheKeyMap() cacheKeyMap {
	return cacheKeyMap{
		Base: keymap.NewBase(),
		Inspect: key.NewBinding(
			key.WithKeys("i", "enter"),
			key.WithHelp("i", "inspect"),
		),
		Analytics: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "analytics"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete from API"),
		),
		Wipe: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "wipe local"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
	}
}

func (k cacheKeyMap) Sections() []keymap.Section {
	return append(k.Base.Sections(),
		keymap.Section{
			Name:     "Cache Actions",
			Bindings: []key.Binding{k.Inspect, k.Analytics, k.Delete, k.Wipe, k.Refresh},
		},
	)
}

// getStatusStyle returns the appropriate theme style for a given status
func getStatusStyle(status string) lipgloss.Style {
	switch status {
	case theme.IconSuccess + " Active":
		return theme.DefaultTheme.Success
	case theme.IconWarning + " Expired":
		return theme.DefaultTheme.Warning
	case theme.IconError + " Cleared":
		return theme.DefaultTheme.Error
	case theme.IconInfo + " Missing":
		return theme.DefaultTheme.Muted
	case theme.IconInfo + " Local":
		return theme.DefaultTheme.Info
	default:
		return lipgloss.NewStyle()
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// newCacheTUIModel initializes the TUI model
func newCacheTUIModel() (*cacheTUIModel, error) {
	ctx := context.Background()
	client, err := gemini.NewClient(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("creating Gemini client: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}

	// Table columns
	columns := []table.Column{
		{Title: "STATUS", Width: 18},
		{Title: "CACHE NAME", Width: 20},
		{Title: "REPO", Width: 15},
		{Title: "MODEL", Width: 25},
		{Title: "USES", Width: 6},
		{Title: "REGEN", Width: 6},
		{Title: "EFF", Width: 5},
		{Title: "TOKENS", Width: 8},
		{Title: "TTL", Width: 10},
		{Title: "EXPIRES", Width: 8},
		{Title: "SAVED", Width: 8},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.DefaultColors.Border).
		BorderBottom(true).
		Bold(false)
	// Remove background highlighting - use same style as unselected rows
	s.Selected = s.Cell.Copy()
	tbl.SetStyles(s)

	// Filter input
	ti := textinput.New()
	ti.Placeholder = "Filter by name, repo, or model..."
	ti.CharLimit = 156
	ti.Width = 50
	
	// Inspect viewport
	vp := viewport.New(80, 20)

	// Initialize keymap and help
	keys := newCacheKeyMap()
	helpModel := help.New(keys)
	helpModel.Title = "Cache Manager Help"

	return &cacheTUIModel{
		client:          client,
		table:           tbl,
		filterInput:     ti,
		inspectViewport: vp,
		help:            helpModel,
		keys:            keys,
		isLoading:       true,
		workDir:         workDir,
		currentView:     listView,
	}, nil
}

// fetchCachesCmd fetches and combines local and remote cache information.
func fetchCachesCmd(client *gemini.Client, workDir string) tea.Cmd {
	return func() tea.Msg {
		// This logic is adapted from the original `listCachesCombined` function.
		ctx := context.Background()
		cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
		localCaches := make(map[string]*gemini.CacheInfo)

		files, err := os.ReadDir(cacheDir)
		if err == nil {
			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "hybrid_") {
					info, err := gemini.LoadCacheInfo(filepath.Join(cacheDir, file.Name()))
					if err == nil {
						localCaches[info.CacheID] = info
					}
				}
			}
		}

		apiCaches, err := client.ListCachesFromAPI(ctx)
		if err != nil {
			// Handle permission errors gracefully
			if gemini.IsPermissionError(err) {
				// Continue with just local caches, no API data
				apiCaches = []gemini.CachedContentInfo{}
			} else {
				// For other errors, return an error message
				return errMsg{fmt.Errorf("could not query API: %w", err)}
			}
		}

		apiCacheMap := make(map[string]*gemini.CachedContentInfo)
		for i := range apiCaches {
			apiCacheMap[apiCaches[i].Name] = &apiCaches[i]
		}

		var combined []combinedCacheInfo
		processed := make(map[string]bool)

		// Process local caches
		for cacheID, localInfo := range localCaches {
			processed[cacheID] = true
			var status string
			isActive := false
			apiInfo, existsInAPI := apiCacheMap[cacheID]

			if localInfo.ClearedAt != nil {
				status = theme.IconError + " Cleared"
			} else if existsInAPI {
				if time.Now().After(apiInfo.ExpireTime) {
					status = theme.IconWarning + " Expired"
				} else {
					status = theme.IconSuccess + " Active"
					isActive = true
				}
			} else {
				status = theme.IconInfo + " Missing"
			}

			combined = append(combined, combinedCacheInfo{
				LocalInfo:  localInfo,
				APIInfo:    apiInfo,
				Name:       localInfo.CacheName,
				Status:     status,
				IsActive:   isActive,
				CreateTime: localInfo.CreatedAt,
			})
		}

		// Process API-only caches
		for _, apiInfo := range apiCaches {
			if !processed[apiInfo.Name] {
				status := theme.IconSuccess + " Active"
				isActive := true
				if time.Now().After(apiInfo.ExpireTime) {
					status = theme.IconWarning + " Expired"
					isActive = false
				}
				
				cacheName := apiInfo.Name
				if parts := strings.Split(apiInfo.Name, "/"); len(parts) > 1 {
					cacheName = parts[len(parts)-1]
				}
				if len(cacheName) > 16 {
					cacheName = cacheName[:16]
				}


				combined = append(combined, combinedCacheInfo{
					APIInfo:    &apiInfo,
					Name:       cacheName,
					Status:     status,
					IsActive:   isActive,
					CreateTime: apiInfo.CreateTime,
				})
			}
		}

		// Sort: active first, then by creation time
		sort.Slice(combined, func(i, j int) bool {
			if combined[i].IsActive != combined[j].IsActive {
				return combined[i].IsActive
			}
			return combined[i].CreateTime.After(combined[j].CreateTime)
		})

		return cachesLoadedMsg{caches: combined}
	}
}

func deleteCacheCmd(client *gemini.Client, cache combinedCacheInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Check if cache is already cleared or missing
		if cache.Status == theme.IconError+" Cleared" {
			return cacheDeletedMsg{} // Already cleared, nothing to do
		}
		
		cacheIDToDelete := ""
		if cache.APIInfo != nil {
			cacheIDToDelete = cache.APIInfo.Name
		} else if cache.LocalInfo != nil {
			cacheIDToDelete = cache.LocalInfo.CacheID
		}

		if cacheIDToDelete == "" {
			return errMsg{fmt.Errorf("cannot delete cache, missing ID")}
		}

		// Only try to delete from API if the cache is active or expired (not missing/cleared)
		if cache.Status == theme.IconSuccess+" Active" || cache.Status == theme.IconWarning+" Expired" {
			// Delete from API
			if err := client.DeleteCache(ctx, cacheIDToDelete); err != nil {
				return errMsg{fmt.Errorf("failed to delete from API: %w", err)}
			}
		}

		// Update local file if it exists
		if cache.LocalInfo != nil {
			workDir, _ := os.Getwd()
			cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
			path := filepath.Join(cacheDir, "hybrid_"+cache.LocalInfo.CacheName+".json")

			now := time.Now()
			cache.LocalInfo.ClearReason = "user-deleted"
			cache.LocalInfo.ClearedAt = &now
			if err := gemini.SaveCacheInfo(path, cache.LocalInfo); err != nil {
				return errMsg{fmt.Errorf("failed to update local cache file: %w", err)}
			}
		}
		return cacheDeletedMsg{}
	}
}

func wipeCacheCmd(cache combinedCacheInfo, workDir string) tea.Cmd {
	return func() tea.Msg {
		if cache.LocalInfo == nil {
			return cacheWipedMsg{} // No local file to wipe
		}
		
		cacheDir := filepath.Join(workDir, ".grove", "gemini-cache")
		path := filepath.Join(cacheDir, "hybrid_"+cache.LocalInfo.CacheName+".json")
		
		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				return errMsg{fmt.Errorf("failed to wipe local cache file: %w", err)}
			}
		}
		
		return cacheWipedMsg{}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- Bubbletea Interface ---

func (m *cacheTUIModel) Init() tea.Cmd {
	return tea.Batch(
		fetchCachesCmd(m.client, m.workDir),
		tickCmd(),
	)
}

func (m *cacheTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(m.width - 2)
		m.table.SetHeight(m.height - 8)
		m.inspectViewport.Width = m.width - 4
		m.inspectViewport.Height = m.height - 8
		m.filterInput.Width = m.width / 2
		m.help.SetSize(m.width, m.height)
		return m, nil

	case cachesLoadedMsg:
		m.isLoading = false
		m.allCaches = msg.caches
		m.updateFilteredCaches()
		return m, nil

	case cacheDeletedMsg:
		m.confirmingDelete = false
		// Refresh the list
		return m, fetchCachesCmd(m.client, m.workDir)

	case cacheWipedMsg:
		m.confirmingWipe = false
		// Refresh the list
		return m, fetchCachesCmd(m.client, m.workDir)

	case errMsg:
		m.err = msg.err
		m.isLoading = false
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchCachesCmd(m.client, m.workDir), tickCmd())

	case tea.KeyMsg:
		if m.filterInput.Focused() {
			if key.Matches(msg, m.keys.Confirm) || key.Matches(msg, m.keys.Back) {
				m.filterInput.Blur()
			} else {
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.updateFilteredCaches()
				return m, cmd
			}
		}

		if m.confirmingDelete {
			switch {
			case key.Matches(msg, m.keys.Confirm):
				if len(m.filteredCaches) > 0 {
					selectedCache := m.filteredCaches[m.table.Cursor()]
					return m, deleteCacheCmd(m.client, selectedCache)
				}
				return m, nil
			case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Back):
				m.confirmingDelete = false
				return m, nil
			}
		}

		if m.confirmingWipe {
			switch {
			case key.Matches(msg, m.keys.Confirm):
				if len(m.filteredCaches) > 0 {
					selectedCache := m.filteredCaches[m.table.Cursor()]
					return m, wipeCacheCmd(selectedCache, m.workDir)
				}
				return m, nil
			case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Back):
				m.confirmingWipe = false
				return m, nil
			}
		}

		switch m.currentView {
		case listView:
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Help):
				m.help.Toggle()
				m.currentView = helpView
				return m, nil
			case key.Matches(msg, m.keys.Search):
				m.filterInput.Focus()
				return m, textinput.Blink
			case key.Matches(msg, m.keys.Inspect):
				if len(m.filteredCaches) > 0 {
					m.currentView = inspectView
					m.prepareInspectView()
				}
				return m, nil
			case key.Matches(msg, m.keys.Delete):
				if len(m.filteredCaches) > 0 {
					m.confirmingDelete = true
				}
				return m, nil
			case key.Matches(msg, m.keys.Wipe):
				if len(m.filteredCaches) > 0 {
					m.confirmingWipe = true
				}
				return m, nil
			case key.Matches(msg, m.keys.Refresh):
				m.isLoading = true
				return m, fetchCachesCmd(m.client, m.workDir)
			case key.Matches(msg, m.keys.Analytics):
				m.currentView = analyticsView
				m.prepareAnalyticsView()
				return m, nil
			}

		case inspectView:
			switch {
			case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
				m.currentView = listView
				return m, nil
			}

		case helpView:
			switch {
			case key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
				m.help.Toggle()
				m.currentView = listView
				return m, nil
			}

		case analyticsView:
			switch {
			case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
				m.currentView = listView
				return m, nil
			}
		}
	}
	
	switch m.currentView {
	case listView:
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	case inspectView:
		m.inspectViewport, cmd = m.inspectViewport.Update(msg)
		cmds = append(cmds, cmd)
	case analyticsView:
		m.inspectViewport, cmd = m.inspectViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *cacheTUIModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}
	if m.isLoading {
		return "Loading caches..."
	}

	if m.currentView == helpView {
		return m.help.View()
	}

	var s strings.Builder
	s.WriteString(theme.DefaultTheme.Header.Render("Gemini API Cache Manager"))
	s.WriteString("\n\n")

	switch m.currentView {
	case listView:
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
		s.WriteString(m.renderTableWithArrow())
	case inspectView:
		s.WriteString(m.inspectViewport.View())
	case analyticsView:
		s.WriteString(m.inspectViewport.View())
	}
	
	s.WriteString("\n")
	s.WriteString(m.footerView())
	
	return s.String()
}

// renderTableWithArrow renders the table with an arrow indicator on the left side
func (m *cacheTUIModel) renderTableWithArrow() string {
	tableStr := m.table.View()
	lines := strings.Split(tableStr, "\n")

	// Calculate which line the selected row is on
	// Bubbles table structure:
	// Line 0: top border
	// Line 1: header row
	// Line 2: separator line after header
	// Line 3+: data rows (with potential separators between them)
	// If there are row separators, each row takes 2 lines (row + separator)
	selectedLineIndex := 3 + (m.table.Cursor() * 2)

	// Add the indicator to each line
	result := ""
	arrow := theme.DefaultTheme.Highlight.Render(theme.IconArrowRightBold)
	for i, line := range lines {
		// Add the indicator on the left for the selected row
		if i == selectedLineIndex {
			result += arrow + " " + line
		} else {
			result += "  " + line
		}
		result += "\n"
	}

	// Remove the trailing newline to match original behavior
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result
}

func (m *cacheTUIModel) footerView() string {
	if m.confirmingDelete {
		if len(m.filteredCaches) > 0 {
			selectedCache := m.filteredCaches[m.table.Cursor()]
			return theme.DefaultTheme.Warning.Render(fmt.Sprintf("Delete cache '%s' from GCP? (y/n)", selectedCache.Name))
		}
	}

	if m.confirmingWipe {
		if len(m.filteredCaches) > 0 {
			selectedCache := m.filteredCaches[m.table.Cursor()]
			return theme.DefaultTheme.Error.Render(fmt.Sprintf("%s  Wipe local file for '%s'? This cannot be undone! (y/n)", theme.IconWarning, selectedCache.Name))
		}
	}

	switch m.currentView {
	case inspectView:
		return theme.DefaultTheme.Muted.Render("Press ? for help")
	default: // listView
		return theme.DefaultTheme.Muted.Render("Press ? for help")
	}
}

func (m *cacheTUIModel) prepareInspectView() {
	if len(m.filteredCaches) == 0 {
		m.inspectViewport.SetContent("No cache selected.")
		return
	}
	cache := m.filteredCaches[m.table.Cursor()]
	
	var b strings.Builder
	b.WriteString(theme.DefaultTheme.Title.Render(fmt.Sprintf("Details for Cache: %s", cache.Name)))
	b.WriteString("\n\n")

	if cache.LocalInfo != nil {
		b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render("--- Local Info ---"))
		b.WriteString(fmt.Sprintf("\nCache ID: %s", cache.LocalInfo.CacheID))
		b.WriteString(fmt.Sprintf("\nRepo: %s", cache.LocalInfo.RepoName))
		b.WriteString(fmt.Sprintf("\nModel: %s", cache.LocalInfo.Model))
		b.WriteString(fmt.Sprintf("\nCreated: %s", cache.LocalInfo.CreatedAt.Local().Format(time.RFC1123)))
		b.WriteString(fmt.Sprintf("\nExpires: %s", cache.LocalInfo.ExpiresAt.Local().Format(time.RFC1123)))
		b.WriteString(fmt.Sprintf("\nToken Count: %d", cache.LocalInfo.TokenCount))
		if cache.LocalInfo.ClearedAt != nil {
			b.WriteString(fmt.Sprintf("\nCleared: %s (%s)", cache.LocalInfo.ClearedAt.Local().Format(time.RFC1123), cache.LocalInfo.ClearReason))
		}
		
		if cache.LocalInfo.UsageStats != nil {
			b.WriteString("\n\nUsage Statistics:")
			b.WriteString(fmt.Sprintf("\n  Total Queries: %d", cache.LocalInfo.UsageStats.TotalQueries))
			b.WriteString(fmt.Sprintf("\n  Last Used: %s", cache.LocalInfo.UsageStats.LastUsed.Local().Format(time.RFC1123)))
			b.WriteString(fmt.Sprintf("\n  Cache Hit Rate: %.1f%%", cache.LocalInfo.UsageStats.AverageHitRate*100))
			b.WriteString(fmt.Sprintf("\n  Tokens Served: %d", cache.LocalInfo.UsageStats.TotalCacheHits))
			b.WriteString(fmt.Sprintf("\n  Tokens Saved: %d", cache.LocalInfo.UsageStats.TotalTokensSaved))
		}
		
		if len(cache.LocalInfo.CachedFileHashes) > 0 {
			b.WriteString("\n\nCached Files:")
			for file, hash := range cache.LocalInfo.CachedFileHashes {
				b.WriteString(fmt.Sprintf("\n  %s", file))
				b.WriteString(fmt.Sprintf("\n    SHA256: %s...", hash[:16]))
			}
		}
		b.WriteString("\n")
	}

	if cache.APIInfo != nil {
		b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render("\n--- API Info ---"))
		b.WriteString(fmt.Sprintf("\nID: %s", cache.APIInfo.Name))
		b.WriteString(fmt.Sprintf("\nModel: %s", cache.APIInfo.Model))
		b.WriteString(fmt.Sprintf("\nToken Count: %d", cache.APIInfo.TokenCount))
		b.WriteString(fmt.Sprintf("\nCreated: %s", cache.APIInfo.CreateTime.Local().Format(time.RFC1123)))
		b.WriteString(fmt.Sprintf("\nExpires: %s", cache.APIInfo.ExpireTime.Local().Format(time.RFC1123)))
		b.WriteString(fmt.Sprintf("\nUpdate Time: %s", cache.APIInfo.UpdateTime.Local().Format(time.RFC1123)))
		b.WriteString("\n")
	}
	
	m.inspectViewport.SetContent(b.String())
	m.inspectViewport.GotoTop()
}

// updateFilteredCaches applies the filter text to the cache list.
func (m *cacheTUIModel) updateFilteredCaches() {
	filter := strings.ToLower(m.filterInput.Value())
	
	var filtered []combinedCacheInfo
	if filter == "" {
		filtered = m.allCaches
	} else {
		for _, cache := range m.allCaches {
			repo := ""
			if cache.LocalInfo != nil {
				repo = cache.LocalInfo.RepoName
			}
			model := ""
			if cache.LocalInfo != nil {
				model = cache.LocalInfo.Model
			} else if cache.APIInfo != nil {
				model = cache.APIInfo.Model
			}

			if strings.Contains(strings.ToLower(cache.Name), filter) ||
				strings.Contains(strings.ToLower(repo), filter) ||
				strings.Contains(strings.ToLower(model), filter) {
				filtered = append(filtered, cache)
			}
		}
	}
	
	m.filteredCaches = filtered
	
	// If the cursor is now out of bounds, reset it.
	if m.table.Cursor() >= len(filtered) {
		m.table.SetCursor(max(0, len(filtered)-1))
	}
	m.updateTableRows()
}

// updateTableRows populates the table with the current filtered caches.
func (m *cacheTUIModel) updateTableRows() {
	rows := make([]table.Row, len(m.filteredCaches))
	for i, cache := range m.filteredCaches {
		repo := "-"
		model := "-"
		uses := "0"
		regen := "0"
		efficiency := "-"
		tokens := "-"
		ttl := "-"
		expires := "-"
		saved := "-"

		if cache.LocalInfo != nil {
			repo = cache.LocalInfo.RepoName
			if len(repo) > 15 {
				repo = repo[:12] + "..."
			}
			model = cache.LocalInfo.Model
			if cache.LocalInfo.UsageStats != nil {
				uses = fmt.Sprintf("%d", cache.LocalInfo.UsageStats.TotalQueries)
			}
			if cache.LocalInfo.RegenerationCount > 0 {
				regen = fmt.Sprintf("%d", cache.LocalInfo.RegenerationCount)
			}
			
			// Calculate efficiency and savings
			if cache.LocalInfo.UsageStats != nil && cache.LocalInfo.UsageStats.TotalQueries > 0 {
				analytics := gemini.CalculateCacheAnalytics(cache.LocalInfo)
				efficiency = fmt.Sprintf("%.0f", analytics.EfficiencyScore)
				if analytics.TotalSavings > 0.01 {
					saved = fmt.Sprintf("$%.2f", analytics.TotalSavings)
				} else {
					saved = "$0.00"
				}
			}
		} else if cache.APIInfo != nil {
			model = cache.APIInfo.Model
		}

		if cache.IsActive {
			var tokenCount int32
			var expireTime time.Time

			if cache.APIInfo != nil {
				tokenCount = cache.APIInfo.TokenCount
				expireTime = cache.APIInfo.ExpireTime
			} else if cache.LocalInfo != nil {
				tokenCount = int32(cache.LocalInfo.TokenCount)
				expireTime = cache.LocalInfo.ExpiresAt
			}

			if tokenCount > 0 {
				tokens = fmt.Sprintf("%dk", tokenCount/1000)
			}
			ttl = formatDuration(time.Until(expireTime))
			expires = expireTime.Local().Format("15:04")
		}
		
		statusStyle := getStatusStyle(cache.Status)

		rows[i] = table.Row{
			statusStyle.Render(cache.Status),
			cache.Name,
			repo,
			model,
			uses,
			regen,
			efficiency,
			tokens,
			ttl,
			expires,
			saved,
		}
	}
	m.table.SetRows(rows)
}


func (m *cacheTUIModel) prepareAnalyticsView() {
	var b strings.Builder
	b.WriteString(theme.DefaultTheme.Title.Render("Cache Analytics Dashboard"))
	b.WriteString("\n\n")
	
	// Calculate total stats across all caches
	totalSavings := 0.0
	totalQueries := 0
	totalCachedTokens := int64(0)
	avgEfficiency := 0.0
	cacheCount := 0
	
	// Hour usage histogram
	hourlyUsage := [24]int{}
	dayUsage := make(map[string]int)
	
	for _, cache := range m.allCaches {
		if cache.LocalInfo != nil && cache.LocalInfo.UsageStats != nil {
			analytics := gemini.CalculateCacheAnalytics(cache.LocalInfo)
			totalSavings += analytics.TotalSavings
			totalQueries += cache.LocalInfo.UsageStats.TotalQueries
			totalCachedTokens += cache.LocalInfo.UsageStats.TotalCacheHits
			avgEfficiency += analytics.EfficiencyScore
			cacheCount++
			
			// Aggregate usage patterns
			for hour, count := range analytics.UsageByHour {
				hourlyUsage[hour] += count
			}
			for day, count := range analytics.UsageByDay {
				dayUsage[day] += count
			}
		}
	}
	
	if cacheCount > 0 {
		avgEfficiency /= float64(cacheCount)
	}
	
	// Overall Statistics
	b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render(theme.IconChart + " Overall Statistics"))
	b.WriteString(fmt.Sprintf("\n\nTotal Cost Savings: $%.2f", totalSavings))
	b.WriteString(fmt.Sprintf("\nTotal Queries: %d", totalQueries))
	b.WriteString(fmt.Sprintf("\nTotal Cached Tokens: %s", formatTokenCount(totalCachedTokens)))
	b.WriteString(fmt.Sprintf("\nAverage Efficiency Score: %.1f/100", avgEfficiency))
	b.WriteString(fmt.Sprintf("\nActive Caches: %d", cacheCount))
	
	// Top Performing Caches
	b.WriteString("\n\n")
	b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render(theme.IconTrophy + " Top Performing Caches"))
	b.WriteString("\n\n")
	
	// Sort caches by efficiency
	type cacheWithScore struct {
		cache combinedCacheInfo
		score float64
		savings float64
	}
	
	var scoredCaches []cacheWithScore
	for _, cache := range m.allCaches {
		if cache.LocalInfo != nil && cache.LocalInfo.UsageStats != nil && cache.LocalInfo.UsageStats.TotalQueries > 0 {
			analytics := gemini.CalculateCacheAnalytics(cache.LocalInfo)
			scoredCaches = append(scoredCaches, cacheWithScore{
				cache: cache,
				score: analytics.EfficiencyScore,
				savings: analytics.TotalSavings,
			})
		}
	}
	
	// Sort by efficiency score
	sort.Slice(scoredCaches, func(i, j int) bool {
		return scoredCaches[i].score > scoredCaches[j].score
	})
	
	// Show top 5
	for i := 0; i < len(scoredCaches) && i < 5; i++ {
		sc := scoredCaches[i]
		b.WriteString(fmt.Sprintf("%d. %s (Score: %.1f, Saved: $%.2f)\n",
			i+1, sc.cache.Name, sc.score, sc.savings))
	}
	
	// Usage Patterns
	b.WriteString("\n")
	b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render("📈 Usage Patterns"))
	b.WriteString("\n\n")
	
	// Find peak hour
	maxHour := 0
	maxHourCount := 0
	for hour, count := range hourlyUsage {
		if count > maxHourCount {
			maxHour = hour
			maxHourCount = count
		}
	}
	
	b.WriteString(fmt.Sprintf("Peak Usage Hour: %02d:00 (%d queries)\n", maxHour, maxHourCount))
	
	// Find peak day
	maxDay := ""
	maxDayCount := 0
	for day, count := range dayUsage {
		if count > maxDayCount {
			maxDay = day
			maxDayCount = count
		}
	}
	
	if maxDay != "" {
		b.WriteString(fmt.Sprintf("Peak Usage Day: %s (%d queries)\n", maxDay, maxDayCount))
	}
	
	// Hourly distribution chart
	b.WriteString("\nHourly Usage Distribution:\n")
	for hour := 0; hour < 24; hour++ {
		count := hourlyUsage[hour]
		bar := strings.Repeat("█", count/2) // Scale down for display
		if count > 0 && len(bar) == 0 {
			bar = "▏" // Show at least a thin bar
		}
		b.WriteString(fmt.Sprintf("%02d:00 %s %d\n", hour, bar, count))
	}
	
	// Cache Hit Rate Trends
	b.WriteString("\n")
	b.WriteString(theme.DefaultTheme.Header.Copy().Underline(false).MarginBottom(0).Render("📉 Cache Hit Rate Trends"))
	b.WriteString("\n\n")
	
	// Collect recent hit rates from all caches
	var recentHitRates []float64
	for _, cache := range m.allCaches {
		if cache.LocalInfo != nil && cache.LocalInfo.UsageStats != nil {
			analytics := gemini.CalculateCacheAnalytics(cache.LocalInfo)
			recentHitRates = append(recentHitRates, analytics.HitRateTrend...)
		}
	}
	
	if len(recentHitRates) > 0 {
		// Calculate average hit rate trend
		avgTrend := 0.0
		for _, rate := range recentHitRates {
			avgTrend += rate
		}
		avgTrend /= float64(len(recentHitRates))
		
		b.WriteString(fmt.Sprintf("Average Recent Hit Rate: %.1f%%\n", avgTrend*100))
		
		// Show visual trend with sparkline
		b.WriteString("\nRecent Hit Rate Trend:\n")
		sparkline := generateSparkline(recentHitRates)
		b.WriteString(sparkline + "\n")
	} else {
		b.WriteString("No hit rate data available yet.\n")
	}
	
	m.inspectViewport.SetContent(b.String())
	m.inspectViewport.GotoTop()
}

// generateSparkline creates a simple ASCII sparkline chart
func generateSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}
	
	// Sparkline characters from low to high
	sparks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	// Find min and max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	
	// Handle case where all values are the same
	if max == min {
		return strings.Repeat("▄", len(data))
	}
	
	// Build sparkline
	var result strings.Builder
	for _, v := range data {
		// Normalize to 0-7 range
		normalized := (v - min) / (max - min)
		index := int(normalized * 7)
		if index > 7 {
			index = 7
		}
		result.WriteString(sparks[index])
	}
	
	return result.String()
}

// formatTokenCount formats large token counts for display
func formatTokenCount(tokens int64) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	} else if tokens >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}

// runCacheTUI runs the interactive TUI for cache management
func runCacheTUI() error {
	model, err := newCacheTUIModel()
	if err != nil {
		return fmt.Errorf("could not initialize TUI model: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running cache TUI: %w", err)
	}
	return nil
}