package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattsolo1/grove-gemini/pkg/gemini"
)

type viewState int

const (
	listView viewState = iota
	inspectView
	helpView
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

// Key bindings
type keyMap struct {
	Up       string
	Down     string
	Filter   string
	Inspect  string
	Delete   string
	Wipe     string
	Refresh  string
	Help     string
	Quit     string
	Back     string
	Confirm  string
	Cancel   string
}

var keys = keyMap{
	Up:       "k",
	Down:     "j",
	Filter:   "/",
	Inspect:  "i",
	Delete:   "d",
	Wipe:     "w",
	Refresh:  "r",
	Help:     "?",
	Quit:     "q",
	Back:     "esc",
	Confirm:  "y",
	Cancel:   "n",
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	
	inspectTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12")).
				MarginBottom(1)

	inspectHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	statusStyles = map[string]lipgloss.Style{
		"✅ Active":  lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Green
		"⏰ Expired": lipgloss.NewStyle().Foreground(lipgloss.Color("11")), // Yellow
		"🚫 Cleared": lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Red
		"❓ Missing": lipgloss.NewStyle().Foreground(lipgloss.Color("8")),  // Grey
		"🔵 Local":   lipgloss.NewStyle().Foreground(lipgloss.Color("12")), // Blue
	}
	
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)
)

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
		{Title: "TOKENS", Width: 8},
		{Title: "TTL", Width: 10},
		{Title: "EXPIRES", Width: 8},
		{Title: "COST", Width: 10},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tbl.SetStyles(s)

	// Filter input
	ti := textinput.New()
	ti.Placeholder = "Filter by name, repo, or model..."
	ti.CharLimit = 156
	ti.Width = 50
	
	// Inspect viewport
	vp := viewport.New(80, 20)

	return &cacheTUIModel{
		client:          client,
		table:           tbl,
		filterInput:     ti,
		inspectViewport: vp,
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
				status = "🚫 Cleared"
			} else if existsInAPI {
				if time.Now().After(apiInfo.ExpireTime) {
					status = "⏰ Expired"
				} else {
					status = "✅ Active"
					isActive = true
				}
			} else {
				status = "❓ Missing"
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
				status := "✅ Active"
				isActive := true
				if time.Now().After(apiInfo.ExpireTime) {
					status = "⏰ Expired"
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
		if cache.Status == "🚫 Cleared" {
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
		if cache.Status == "✅ Active" || cache.Status == "⏰ Expired" {
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
			if msg.String() == "enter" || msg.String() == "esc" {
				m.filterInput.Blur()
			} else {
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.updateFilteredCaches()
				return m, cmd
			}
		}

		if m.confirmingDelete {
			switch msg.String() {
			case keys.Confirm:
				if len(m.filteredCaches) > 0 {
					selectedCache := m.filteredCaches[m.table.Cursor()]
					return m, deleteCacheCmd(m.client, selectedCache)
				}
				return m, nil
			case keys.Cancel, keys.Back:
				m.confirmingDelete = false
				return m, nil
			}
		}
		
		if m.confirmingWipe {
			switch msg.String() {
			case keys.Confirm:
				if len(m.filteredCaches) > 0 {
					selectedCache := m.filteredCaches[m.table.Cursor()]
					return m, wipeCacheCmd(selectedCache, m.workDir)
				}
				return m, nil
			case keys.Cancel, keys.Back:
				m.confirmingWipe = false
				return m, nil
			}
		}
		
		switch m.currentView {
		case listView:
			switch msg.String() {
			case keys.Quit:
				return m, tea.Quit
			case keys.Help:
				m.currentView = helpView
				return m, nil
			case keys.Filter:
				m.filterInput.Focus()
				return m, textinput.Blink
			case keys.Inspect, "enter":
				if len(m.filteredCaches) > 0 {
					m.currentView = inspectView
					m.prepareInspectView()
				}
				return m, nil
			case keys.Delete:
				if len(m.filteredCaches) > 0 {
					m.confirmingDelete = true
				}
				return m, nil
			case keys.Wipe:
				if len(m.filteredCaches) > 0 {
					m.confirmingWipe = true
				}
				return m, nil
			case keys.Refresh:
				m.isLoading = true
				return m, fetchCachesCmd(m.client, m.workDir)
			case "ctrl+c":
				return m, tea.Quit
			}

		case inspectView:
			switch msg.String() {
			case keys.Back, keys.Quit:
				m.currentView = listView
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			
		case helpView:
			switch msg.String() {
			case keys.Help, keys.Back, keys.Quit:
				m.currentView = listView
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
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
		return m.helpViewRender()
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render("Gemini API Cache Manager"))
	s.WriteString("\n\n")

	switch m.currentView {
	case listView:
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
		s.WriteString(m.table.View())
	case inspectView:
		s.WriteString(m.inspectViewport.View())
	}
	
	s.WriteString("\n")
	s.WriteString(m.footerView())
	
	return s.String()
}

func (m *cacheTUIModel) footerView() string {
	if m.confirmingDelete {
		if len(m.filteredCaches) > 0 {
			selectedCache := m.filteredCaches[m.table.Cursor()]
			return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(fmt.Sprintf("Delete cache '%s' from GCP? (y/n)", selectedCache.Name))
		}
	}
	
	if m.confirmingWipe {
		if len(m.filteredCaches) > 0 {
			selectedCache := m.filteredCaches[m.table.Cursor()]
			return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("⚠️  Wipe local file for '%s'? This cannot be undone! (y/n)", selectedCache.Name))
		}
	}

	switch m.currentView {
	case inspectView:
		return helpStyle.Render("Press ? for help")
	default: // listView
		return helpStyle.Render("Press ? for help")
	}
}

func (m *cacheTUIModel) helpViewRender() string {
	help := `╭─ Cache Manager Help ─────────────────────────────────────────────────────────────────────╮
│                                                                                         │
│  Navigation:                        Status Icons:                                       │
│    j/k or ↑/↓  Move up/down        ✅ Active   Cache is valid and accessible         │
│    enter or i  Inspect cache        ⏰ Expired  Cache has exceeded TTL                 │
│    /           Filter caches        🚫 Cleared  Cache was manually deleted             │
│    esc         Exit view/cancel     ❓ Missing  Local record but not in API           │
│                                     🔵 Local    Local-only view status                 │
│  Actions:                                                                               │
│    d           Delete from GCP      Other:                                              │
│    w           Wipe local file      r           Refresh cache list                      │
│    y/n         Confirm/cancel       ?           Show/hide this help                     │
│                                     q           Quit the application                    │
╰─────────────────────────────────────────────────────────────────────────────────────────╯`
	
	// Center the help box
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		helpBoxStyle.Render(help),
	)
}

func (m *cacheTUIModel) prepareInspectView() {
	if len(m.filteredCaches) == 0 {
		m.inspectViewport.SetContent("No cache selected.")
		return
	}
	cache := m.filteredCaches[m.table.Cursor()]
	
	var b strings.Builder
	b.WriteString(inspectTitleStyle.Render(fmt.Sprintf("Details for Cache: %s", cache.Name)))
	b.WriteString("\n\n")
	
	if cache.LocalInfo != nil {
		b.WriteString(inspectHeaderStyle.Render("--- Local Info ---"))
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
		b.WriteString(inspectHeaderStyle.Render("\n--- API Info ---"))
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
		tokens := "-"
		ttl := "-"
		expires := "-"
		cost := "-"

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
		} else if cache.APIInfo != nil {
			model = cache.APIInfo.Model
		}

		if cache.IsActive {
			var tokenCount int32
			var expireTime time.Time
			var createTime time.Time

			if cache.APIInfo != nil {
				tokenCount = cache.APIInfo.TokenCount
				expireTime = cache.APIInfo.ExpireTime
				createTime = cache.APIInfo.CreateTime
			} else if cache.LocalInfo != nil {
				tokenCount = int32(cache.LocalInfo.TokenCount)
				expireTime = cache.LocalInfo.ExpiresAt
				createTime = cache.LocalInfo.CreatedAt
			}

			if tokenCount > 0 {
				tokens = fmt.Sprintf("%dk", tokenCount/1000)
			}
			ttl = formatDuration(time.Until(expireTime))
			expires = expireTime.Local().Format("15:04")
			cost = calculateCacheCost(tokenCount, expireTime.Sub(createTime), model)
		}
		
		statusStyle, ok := statusStyles[cache.Status]
		if !ok {
			statusStyle = lipgloss.NewStyle()
		}

		rows[i] = table.Row{
			statusStyle.Render(cache.Status),
			cache.Name,
			repo,
			model,
			uses,
			regen,
			tokens,
			ttl,
			expires,
			cost,
		}
	}
	m.table.SetRows(rows)
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