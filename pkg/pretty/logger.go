package pretty

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	corelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-core/tui/theme"
)

// Logger is a wrapper around the grove-core UnifiedLogger with Gemini-specific helpers.
type Logger struct {
	*corelogging.PrettyLogger
	ulog   *corelogging.UnifiedLogger
	writer io.Writer
	theme  *theme.Theme
}

// TokenFields represents token usage metrics with verbosity levels
type TokenFields struct {
	CachedTokens      int     `json:"cached_tokens" verbosity:"0"`       // metrics
	DynamicTokens     int     `json:"dynamic_tokens" verbosity:"0"`      // metrics
	CompletionTokens  int     `json:"completion_tokens" verbosity:"0"`   // metrics
	UserPromptTokens  int     `json:"user_prompt_tokens" verbosity:"0"`  // metrics
	TotalPromptTokens int     `json:"total_prompt_tokens" verbosity:"0"` // metrics
	ResponseTimeMs    int64   `json:"response_time_ms" verbosity:"0"`    // metrics
	CacheHitRate      float64 `json:"cache_hit_rate" verbosity:"0"`      // metrics - percentage (0-100)
	IsNewCache        bool    `json:"is_new_cache" verbosity:"0"`        // metrics
}

// ModelFields represents model information with verbosity level
type ModelFields struct {
	Model string `json:"model" verbosity:"3"` // metrics
}

// New creates a new Gemini-specific pretty logger.
func New() *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger(),
		ulog:         corelogging.NewUnifiedLogger("grove-gemini"),
		writer:       corelogging.GetGlobalOutput(),
		theme:        theme.DefaultTheme,
	}
}

// NewWithWriter creates a new Logger with a custom writer
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger().WithWriter(w),
		ulog:         corelogging.NewUnifiedLogger("grove-gemini"),
		writer:       w,
		theme:        theme.DefaultTheme,
	}
}

// WorkingDirectoryCtx logs the working directory to the writer from the context
func (l *Logger) WorkingDirectoryCtx(ctx context.Context, dir string) {
	pathStyle := lipgloss.NewStyle().Italic(true)
	l.ulog.Info("Working directory").
		Field("directory", dir).
		Pretty(fmt.Sprintf("%s Working directory: %s", theme.IconHome, pathStyle.Render(dir))).
		Log(ctx)
}

// WorkingDirectory logs the working directory
func (l *Logger) WorkingDirectory(dir string) {
	l.WorkingDirectoryCtx(context.Background(), dir)
}

// FoundRulesFileCtx logs that a rules file was found to the writer from the context
func (l *Logger) FoundRulesFileCtx(ctx context.Context, path string) {
	l.PathCtx(ctx, theme.IconChecklist+" Found rules file", path)
}

// FoundRulesFile logs that a rules file was found
func (l *Logger) FoundRulesFile(path string) {
	l.FoundRulesFileCtx(context.Background(), path)
}

// WarningCtx logs a warning message to the writer from the context
func (l *Logger) WarningCtx(ctx context.Context, message string) {
	l.WarnPrettyCtx(ctx, message)
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	l.WarningCtx(context.Background(), message)
}

// InfoCtx logs an info message to the writer from the context
func (l *Logger) InfoCtx(ctx context.Context, message string) {
	l.InfoPrettyCtx(ctx, message)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.InfoCtx(context.Background(), message)
}

// SuccessCtx logs a success message to the writer from the context
func (l *Logger) SuccessCtx(ctx context.Context, message string) {
	l.PrettyLogger.SuccessCtx(ctx, message)
}

// Success logs a success message
func (l *Logger) Success(message string) {
	l.SuccessCtx(context.Background(), message)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.ulog.Error(message).Emit()
}

// ModelCtx logs the model being used to the writer from the context
func (l *Logger) ModelCtx(ctx context.Context, model string) {
	l.ulog.Info("Calling Gemini API").
		Field("model", model).
		Pretty(fmt.Sprintf("%s Calling Gemini API with model: %s", theme.IconRobot, model)).
		Log(ctx)
}

// Model logs the model being used
func (l *Logger) Model(model string) {
	l.ModelCtx(context.Background(), model)
}

// UploadProgressCtx logs file upload progress to the writer from the context
func (l *Logger) UploadProgressCtx(ctx context.Context, message string) {
	l.ulog.Progress(message).
		Pretty(fmt.Sprintf("%s %s", theme.IconRunning, message)).
		Log(ctx)
}

// UploadProgress logs file upload progress
func (l *Logger) UploadProgress(message string) {
	l.UploadProgressCtx(context.Background(), message)
}

// UploadComplete logs successful file upload
func (l *Logger) UploadComplete(filename string, duration time.Duration) {
	l.ulog.Info(filename).
		Field("filename", filename).
		Field("duration_seconds", duration.Seconds()).
		Pretty(fmt.Sprintf("%s %s %s",
			theme.IconSuccess,
			filename,
			l.theme.Muted.Render(fmt.Sprintf("(%.2fs)", duration.Seconds())))).
		Log(context.Background())
}

// GeneratingResponse logs that response generation has started
func (l *Logger) GeneratingResponse() {
	l.ulog.Progress("Generating response...").
		Icon(theme.IconRobot).
		Log(context.Background())
}

// FilesIncludedCtx displays the list of files that will be included in the request to the writer from the context
func (l *Logger) FilesIncludedCtx(ctx context.Context, files []string) {
	if len(files) == 0 {
		return
	}

	pathStyle := lipgloss.NewStyle().Italic(true)
	promptStyle := l.theme.Muted
	var prettyLines []string
	prettyLines = append(prettyLines, fmt.Sprintf("%s Files attached to request:", theme.IconFile))

	for _, file := range files {
		// Extract just the filename or last part of the path for display
		displayName := file
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			displayName = file[idx+1:]
		}

		// Check if this is likely a prompt file (has .md extension and not CLAUDE.md)
		isPromptFile := strings.HasSuffix(file, ".md") && displayName != "CLAUDE.md" &&
			displayName != "context" && displayName != "cached-context"

		// Show full path if it's a special file or prompt file
		var displayItem string
		if displayName == "CLAUDE.md" || displayName == "context" || displayName == "cached-context" {
			displayItem = pathStyle.Render(file)
		} else if isPromptFile {
			displayItem = pathStyle.Render(file) + " " + promptStyle.Render("(prompt)")
		} else {
			displayItem = pathStyle.Render(displayName)
		}
		prettyLines = append(prettyLines, fmt.Sprintf("%s %s", l.theme.Highlight.Render(theme.IconBullet), displayItem))
	}

	l.ulog.Info("Files attached to request").
		Field("files", files).
		Field("count", len(files)).
		Pretty(strings.Join(prettyLines, "\n")).
		Log(ctx)
}

// FilesIncluded displays the list of files that will be included in the request
func (l *Logger) FilesIncluded(files []string) {
	l.FilesIncludedCtx(context.Background(), files)
}

// TokenUsageCtx displays token usage statistics in a styled box to the writer from the context
func (l *Logger) TokenUsageCtx(ctx context.Context, cached, dynamic, completion, promptTokens int, responseTime time.Duration, isNewCache bool) {
	// Calculate cache hit rate
	totalPrompt := cached + dynamic
	cacheHitRate := 0.0
	if totalPrompt > 0 {
		cacheHitRate = float64(cached) / float64(totalPrompt) * 100
	}

	// Calculate derived metrics for UI display
	totalAPIUsage := dynamic + completion

	// Conditional labels and styles
	cachedLabel := "Cold (Cached):"
	cacheHitRateLabel := "Cache Hit Rate:"
	cachedStyle := l.theme.Bold

	if isNewCache {
		cachedLabel = "New Cache Tokens:"
		cacheHitRateLabel = "Cache Hit Rate (fresh):"
		cachedStyle = l.theme.Warning // Use warning style for new cache tokens
	}

	// Create the content with proper formatting using theme
	content := []string{
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render(cachedLabel),
			cachedStyle.Render(fmt.Sprintf("%d tokens", cached))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Hot (Dynamic):"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens", dynamic))),
	}

	// Add user prompt tokens if available
	if promptTokens > 0 {
		content = append(content, fmt.Sprintf("%s %s",
			l.theme.Muted.Render("User Prompt:"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens", promptTokens))))
	}

	divider := l.theme.Muted.Render(strings.Repeat("─", 32))
	content = append(content, []string{
		divider,
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Total Prompt:"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens", totalPrompt))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Completion:"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens", completion))),
		divider,
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Total API Usage:"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens", totalAPIUsage))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render(cacheHitRateLabel),
			l.theme.Success.Render(fmt.Sprintf("%.1f%%", cacheHitRate))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Response Time:"),
			l.theme.Muted.Render(fmt.Sprintf("%.2fs", responseTime.Seconds()))),
	}...)

	// Join with newlines and apply box styling using theme
	tokenBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(l.theme.Colors.Violet).
		Padding(0, 1)

	box := tokenBox.Render(strings.Join(content, "\n"))

	l.ulog.Info("Gemini Response & Token Summary").
		Field("cached_tokens", cached).
		Field("dynamic_tokens", dynamic).
		Field("completion_tokens", completion).
		Field("user_prompt_tokens", promptTokens).
		Field("total_prompt_tokens", totalPrompt).
		Field("response_time_ms", responseTime.Milliseconds()).
		Field("cache_hit_rate", cacheHitRate).
		Field("is_new_cache", isNewCache).
		Pretty(fmt.Sprintf("%s Token usage:\n%s", theme.IconChart, box)).
		Log(ctx)
}

// TokenUsage displays token usage statistics in a styled box
func (l *Logger) TokenUsage(cached, dynamic, completion, promptTokens int, responseTime time.Duration, isNewCache bool) {
	l.TokenUsageCtx(context.Background(), cached, dynamic, completion, promptTokens, responseTime, isNewCache)
}

// CacheInfo logs cache-related information
func (l *Logger) CacheInfo(message string) {
	l.InfoPretty(message)
}

// CacheCreated logs successful cache creation
func (l *Logger) CacheCreated(cacheID string, expires time.Time) {
	relativeTime := formatRelativeTime(expires)
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.theme.Success.Render(theme.IconSuccess),
		l.theme.Success.Render("Cache created:"),
		pathStyle.Render(cacheID))
	fmt.Fprintf(l.writer, "%s %s %s\n",
		theme.IconCalendar,
		l.theme.Muted.Render("Expires"),
		l.theme.Muted.Render(relativeTime))
}

// ChangedFiles logs files that have changed
func (l *Logger) ChangedFiles(files []string) {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Warning.Render(theme.IconSync+" Cached files have changed:"))
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)
	for _, file := range files {
		fmt.Fprintf(l.writer, "%s %s\n",
			l.theme.Highlight.Render(theme.IconBullet),
			pathStyle.Render(file))
	}
}

// CreatingCache logs cache creation start
func (l *Logger) CreatingCache() {
	fmt.Fprintf(l.writer, "\n%s\n",
		l.theme.Warning.Render(theme.IconMoney+" Creating new cache (one-time operation)..."))
}

// NoCache logs when no cache is found
func (l *Logger) NoCache() {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Info.Render(theme.IconSparkle+" No existing cache found"))
}

// CacheValid logs when cache is valid
func (l *Logger) CacheValid(until time.Time) {
	relativeTime := formatRelativeTime(until)
	fmt.Fprintf(l.writer, "%s %s (%s %s)\n",
		l.theme.Success.Render(theme.IconSuccess),
		l.theme.Success.Render("Cache is valid"),
		l.theme.Muted.Render("expires"),
		l.theme.Muted.Render(relativeTime))
}

// CacheExpired logs when cache has expired
func (l *Logger) CacheExpired(at time.Time) {
	relativeTime := formatRelativeTime(at)
	fmt.Fprintf(l.writer, "%s (%s)\n",
		l.theme.Warning.Render(theme.IconClock+" Cache expired"),
		l.theme.Muted.Render(relativeTime))
}

// CacheFrozen logs when cache is frozen
func (l *Logger) CacheFrozen() {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Info.Render(theme.IconSnowflake+" Cache is frozen by @freeze-cache directive"))
}

// CacheDisabled logs when cache is disabled
func (l *Logger) CacheDisabled() {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Warning.Render(theme.IconStop+" Cache disabled by @disable-cache directive"))
}

// TTL logs the cache TTL
func (l *Logger) TTL(ttl string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.theme.Info.Render(theme.IconClock+" Using cache TTL from @expire-time directive:"),
		l.theme.Muted.Render(ttl))
}

// CacheDisabledByDefault logs when cache is disabled by default (opt-in model)
func (l *Logger) CacheDisabledByDefault() {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Info.Render(theme.IconInfo+" Caching is disabled by default"))
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Muted.Render(theme.IconBullet+" To enable caching, add @enable-cache to your .grove/rules file"))
}

// CacheWarningCtx displays a prominent warning about experimental caching and costs to the writer from the context
func (l *Logger) CacheWarningCtx(ctx context.Context) {
	writer := corelogging.GetWriter(ctx)

	// Create a warning box style using theme colors
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(l.theme.Colors.Orange).
		Foreground(l.theme.Colors.Orange).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1).
		Bold(true)

	warningContent := fmt.Sprintf(
		"WARNING:  ALPHA FEATURE WARNING\n\n" +
			"Gemini Caching is experimental and can incur significant costs.\n" +
			"Please monitor your Google Cloud billing closely to avoid unexpected charges.\n\n" +
			"You can disable caching with the --no-cache flag or by removing @enable-cache from your rules.")

	fmt.Fprintln(writer, warningBox.Render(warningContent))
}

// CacheWarning displays a prominent warning about experimental caching and costs
func (l *Logger) CacheWarning() {
	l.CacheWarningCtx(context.Background())
}

// EstimatedTokens logs estimated token count
func (l *Logger) EstimatedTokens(count int) {
	fmt.Fprintf(l.writer, "   %s %s\n",
		l.theme.Muted.Render("Estimated tokens:"),
		l.theme.Normal.Render(fmt.Sprintf("%d", count)))
}

// ResponseWritten logs successful response write
func (l *Logger) ResponseWritten(path string) {
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.theme.Success.Render(theme.IconSuccess),
		l.theme.Success.Render("Response written to:"),
		pathStyle.Render(path))
}

// Tip logs a helpful tip
func (l *Logger) Tip(message string) {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Info.Render(theme.IconLightbulb+" "+message))
}

// RulesFileContent displays the rules file content in a styled box
func (l *Logger) RulesFileContent(content string) {
	// Create a box style specifically for rules content using theme
	rulesBox := l.theme.Box.
		BorderForeground(l.theme.Colors.Blue).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Header.Render(theme.IconChecklist+" Rules file content:"))

	// Apply box styling to the content
	box := rulesBox.Render(content)
	fmt.Fprintln(l.writer, box)
}

// ContextSummary logs a context summary with styled formatting
func (l *Logger) ContextSummary(cold, hot int) {
	fmt.Fprintf(l.writer, "%s\n",
		l.theme.Header.Render(theme.IconChart+" Context Summary:"))
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.theme.Highlight.Render(theme.IconBullet),
		l.theme.Muted.Render("Cold files:"),
		l.theme.Normal.Render(fmt.Sprintf("%d", cold)))
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.theme.Highlight.Render(theme.IconBullet),
		l.theme.Muted.Render("Hot files:"),
		l.theme.Normal.Render(fmt.Sprintf("%d", hot)))
}

// CacheCreationPrompt shows cache creation details and prompts for confirmation
func (l *Logger) CacheCreationPrompt(tokens int, sizeBytes int64, ttl time.Duration) bool {
	// Create a prominent box for the cache creation warning using theme
	warningBox := l.theme.Box.
		BorderForeground(l.theme.Colors.Yellow).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	// Format size
	sizeStr := formatFileSize(sizeBytes)
	relativeTime := formatRelativeTime(time.Now().Add(ttl))

	content := []string{
		l.theme.Warning.Bold(true).Render("NEW CACHE CREATION REQUIRED"),
		"",
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Cache size:"),
			l.theme.Normal.Render(fmt.Sprintf("%d tokens (%s)", tokens, sizeStr))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Expires:"),
			l.theme.Muted.Render(relativeTime)),
		"",
		"Creating a cache will upload context to Gemini's servers.",
		"This is a one-time operation that may incur costs.",
	}

	box := warningBox.Render(strings.Join(content, "\n"))
	fmt.Fprintln(l.writer)
	fmt.Fprintln(l.writer, box)

	// Prompt for confirmation
	fmt.Fprintf(l.writer, "\n%s %s",
		theme.IconHelp,
		l.theme.Warning.Render("Do you want to create this cache? [y/N]: "))

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// formatRelativeTime formats a time relative to now in a human-friendly way
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)

	// Future time (expires in...)
	if diff > 0 {
		hours := int(diff.Hours())
		if hours < 1 {
			minutes := int(diff.Minutes())
			if minutes <= 1 {
				return "in less than a minute"
			}
			return fmt.Sprintf("in %d minutes", minutes)
		} else if hours == 1 {
			return "in 1 hour"
		} else if hours < 24 {
			return fmt.Sprintf("in %d hours", hours)
		} else {
			days := hours / 24
			if days == 1 {
				return "in 1 day"
			}
			return fmt.Sprintf("in %d days", days)
		}
	}

	// Past time (expired...ago)
	diff = -diff
	hours := int(diff.Hours())
	if hours < 1 {
		minutes := int(diff.Minutes())
		if minutes <= 1 {
			return "less than a minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if hours == 1 {
		return "1 hour ago"
	} else if hours < 24 {
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := hours / 24
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatFileSize formats bytes into human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
