package pretty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	corelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-core/tui/theme"
	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around the grove-core PrettyLogger with Gemini-specific helpers.
type Logger struct {
	*corelogging.PrettyLogger
	writer io.Writer
	theme  *theme.Theme
	log    *logrus.Entry // For structured logging when needed
}

// TokenFields represents token usage metrics with verbosity levels
type TokenFields struct {
	CachedTokens      int   `json:"cached_tokens" verbosity:"0"`       // metrics
	DynamicTokens     int   `json:"dynamic_tokens" verbosity:"0"`      // metrics
	CompletionTokens  int   `json:"completion_tokens" verbosity:"0"`   // metrics
	UserPromptTokens  int   `json:"user_prompt_tokens" verbosity:"0"`  // metrics
	TotalPromptTokens int   `json:"total_prompt_tokens" verbosity:"0"` // metrics
	ResponseTimeMs    int64 `json:"response_time_ms" verbosity:"0"`    // metrics
	IsNewCache        bool  `json:"is_new_cache" verbosity:"0"`        // metrics
}

// ModelFields represents model information with verbosity level
type ModelFields struct {
	Model string `json:"model" verbosity:"3"` // metrics
}

// New creates a new Gemini-specific pretty logger.
func New() *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger(),
		writer:       os.Stderr,
		theme:        theme.DefaultTheme,
		log:          corelogging.NewLogger("grove-gemini"),
	}
}

// NewWithLogger creates a new logger with a specific structured logging backend.
func NewWithLogger(log *logrus.Entry) *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger(),
		writer:       os.Stderr,
		theme:        theme.DefaultTheme,
		log:          log,
	}
}

// NewWithWriter creates a new Logger with a custom writer
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger().WithWriter(w),
		writer:       w,
		theme:        theme.DefaultTheme,
		log:          corelogging.NewLogger("grove-gemini"),
	}
}

// WorkingDirectory logs the working directory
func (l *Logger) WorkingDirectory(dir string) {
	l.Path("🏠 Working directory", dir)
}

// FoundRulesFile logs that a rules file was found
func (l *Logger) FoundRulesFile(path string) {
	l.Path("📋 Found rules file", path)
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	l.WarnPretty(message)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.InfoPretty(message)
}

// Success logs a success message
func (l *Logger) Success(message string) {
	l.PrettyLogger.Success(message)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.theme.Error.Render(theme.IconError),
		l.theme.Error.Render(message))
}

// Model logs the model being used
func (l *Logger) Model(model string) {
	// Log structured data if backend available
	if l.log != nil {
		modelFields := ModelFields{
			Model: model,
		}
		fields := corelogging.StructToLogrusFields(modelFields)

		// Get caller information manually to point to the actual caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			fields["file"] = fmt.Sprintf("%s:%d", file, line)
			if fn := runtime.FuncForPC(pc); fn != nil {
				fields["func"] = fn.Name()
			}
		}

		// Create entry without logrus's automatic caller reporting to avoid duplication
		entry := l.log.WithFields(fields)
		entry.Info("Calling Gemini API")
	}
	// Display pretty UI
	fmt.Fprintf(l.writer, "\n🤖 %s %s\n\n",
		l.theme.Info.Render("Calling Gemini API with model:"),
		l.theme.Accent.Render(model))
}

// UploadProgress logs file upload progress
func (l *Logger) UploadProgress(message string) {
	l.Progress(message)
}

// UploadComplete logs successful file upload
func (l *Logger) UploadComplete(filename string, duration time.Duration) {
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.theme.Success.Render(theme.IconSuccess),
		l.theme.Success.Render(filename),
		l.theme.Muted.Render(fmt.Sprintf("(%.2fs)", duration.Seconds())))
}

// GeneratingResponse logs that response generation has started
func (l *Logger) GeneratingResponse() {
	fmt.Fprintf(l.writer, "\n🤖 %s\n",
		l.theme.Info.Render("Generating response..."))
}

// FilesIncluded displays the list of files that will be included in the request
func (l *Logger) FilesIncluded(files []string) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintf(l.writer, "\n📁 %s\n",
		l.theme.Header.Render("Files attached to request:"))

	// Build display list with styled paths
	displayFiles := make([]string, len(files))
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)

	for i, file := range files {
		// Extract just the filename or last part of the path for display
		displayName := file
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			displayName = file[idx+1:]
		}

		// Check if this is likely a prompt file (has .md extension and not CLAUDE.md)
		isPromptFile := strings.HasSuffix(file, ".md") && displayName != "CLAUDE.md" &&
			displayName != "context" && displayName != "cached-context"

		// Show full path if it's a special file or prompt file
		if displayName == "CLAUDE.md" || displayName == "context" || displayName == "cached-context" {
			displayFiles[i] = pathStyle.Render(file)
		} else if isPromptFile {
			displayFiles[i] = pathStyle.Render(file) + " (prompt)"
		} else {
			displayFiles[i] = pathStyle.Render(displayName)
		}
	}

	l.List(displayFiles)
}

// TokenUsage displays token usage statistics in a styled box
func (l *Logger) TokenUsage(cached, dynamic, completion, promptTokens int, responseTime time.Duration, isNewCache bool) {
	// First, log structured data to backend if available
	if l.log != nil {
		tokenFields := TokenFields{
			CachedTokens:      cached,
			DynamicTokens:     dynamic,
			CompletionTokens:  completion,
			UserPromptTokens:  promptTokens,
			TotalPromptTokens: cached + dynamic,
			ResponseTimeMs:    responseTime.Milliseconds(),
			IsNewCache:        isNewCache,
		}
		fields := corelogging.StructToLogrusFields(tokenFields)

		// Get caller information manually to point to the actual caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			fields["file"] = fmt.Sprintf("%s:%d", file, line)
			if fn := runtime.FuncForPC(pc); fn != nil {
				fields["func"] = fn.Name()
			}
		}

		// Create entry without logrus's automatic caller reporting to avoid duplication
		entry := l.log.WithFields(fields)
		entry.Info("Token usage summary")
	}

	// Calculate derived metrics for UI display
	totalPrompt := cached + dynamic
	totalAPIUsage := dynamic + completion
	cacheHitRate := 0.0
	if totalPrompt > 0 {
		cacheHitRate = float64(cached) / float64(totalPrompt) * 100
	}

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
			l.theme.Bold.Render(fmt.Sprintf("%d tokens", dynamic))),
	}

	// Add user prompt tokens if available
	if promptTokens > 0 {
		content = append(content, fmt.Sprintf("%s %s",
			l.theme.Muted.Render("User Prompt:"),
			l.theme.Bold.Render(fmt.Sprintf("%d tokens", promptTokens))))
	}

	divider := l.theme.Muted.Render(strings.Repeat("─", 32))
	content = append(content, []string{
		divider,
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Total Prompt:"),
			l.theme.Bold.Render(fmt.Sprintf("%d tokens", totalPrompt))),
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Completion:"),
			l.theme.Bold.Render(fmt.Sprintf("%d tokens", completion))),
		divider,
		fmt.Sprintf("%s %s",
			l.theme.Muted.Render("Total API Usage:"),
			l.theme.Bold.Render(fmt.Sprintf("%d tokens", totalAPIUsage))),
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
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	box := tokenBox.Render(strings.Join(content, "\n"))

	fmt.Fprintf(l.writer, "\n📊 %s\n%s\n",
		l.theme.Header.Render("Token usage:"),
		box)
}

// CacheInfo logs cache-related information
func (l *Logger) CacheInfo(message string) {
	l.InfoPretty(message)
}

// CacheCreated logs successful cache creation
func (l *Logger) CacheCreated(cacheID string, expires time.Time) {
	relativeTime := formatRelativeTime(expires)
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.theme.Success.Render(theme.IconSuccess),
		l.theme.Success.Render("Cache created:"),
		pathStyle.Render(cacheID))
	fmt.Fprintf(l.writer, "  📅 %s %s\n",
		l.theme.Muted.Render("Expires"),
		l.theme.Muted.Render(relativeTime))
}

// ChangedFiles logs files that have changed
func (l *Logger) ChangedFiles(files []string) {
	fmt.Fprintf(l.writer, "🔄 %s\n",
		l.theme.Warning.Render("Cached files have changed:"))
	pathStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Italic(true)
	for _, file := range files {
		fmt.Fprintf(l.writer, "   %s %s\n",
			l.theme.Highlight.Render(theme.IconBullet),
			pathStyle.Render(file))
	}
}

// CreatingCache logs cache creation start
func (l *Logger) CreatingCache() {
	fmt.Fprintf(l.writer, "\n💰 %s\n",
		l.theme.Warning.Render("Creating new cache (one-time operation)..."))
}

// NoCache logs when no cache is found
func (l *Logger) NoCache() {
	fmt.Fprintf(l.writer, "🆕 %s\n",
		l.theme.Info.Render("No existing cache found"))
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
	fmt.Fprintf(l.writer, "⏰ %s (%s)\n",
		l.theme.Warning.Render("Cache expired"),
		l.theme.Muted.Render(relativeTime))
}

// CacheFrozen logs when cache is frozen
func (l *Logger) CacheFrozen() {
	fmt.Fprintf(l.writer, "❄️ %s\n",
		l.theme.Info.Render("Cache is frozen by @freeze-cache directive"))
}

// CacheDisabled logs when cache is disabled
func (l *Logger) CacheDisabled() {
	fmt.Fprintf(l.writer, "🚫 %s\n",
		l.theme.Warning.Render("Cache disabled by @disable-cache directive"))
}

// TTL logs the cache TTL
func (l *Logger) TTL(ttl string) {
	fmt.Fprintf(l.writer, "⏱️ %s %s\n",
		l.theme.Info.Render("Using cache TTL from @expire-time directive:"),
		l.theme.Muted.Render(ttl))
}

// CacheDisabledByDefault logs when cache is disabled by default (opt-in model)
func (l *Logger) CacheDisabledByDefault() {
	fmt.Fprintf(l.writer, "ℹ️ %s\n",
		l.theme.Info.Render("Caching is disabled by default"))
	fmt.Fprintf(l.writer, "   %s\n",
		l.theme.Muted.Render("To enable caching, add @enable-cache to your .grove/rules file"))
}

// CacheWarning displays a prominent warning about experimental caching and costs
func (l *Logger) CacheWarning() {
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
		"⚠️  ALPHA FEATURE WARNING\n\n" +
			"Gemini Caching is experimental and can incur significant costs.\n" +
			"Please monitor your Google Cloud billing closely to avoid unexpected charges.\n\n" +
			"You can disable caching with the --no-cache flag or by removing @enable-cache from your rules.")

	fmt.Fprintln(l.writer, warningBox.Render(warningContent))
}

// EstimatedTokens logs estimated token count
func (l *Logger) EstimatedTokens(count int) {
	fmt.Fprintf(l.writer, "   %s %s\n",
		l.theme.Muted.Render("Estimated tokens:"),
		l.theme.Bold.Render(fmt.Sprintf("%d", count)))
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
	fmt.Fprintf(l.writer, "💡 %s\n",
		l.theme.Info.Render(message))
}

// RulesFileContent displays the rules file content in a styled box
func (l *Logger) RulesFileContent(content string) {
	// Create a box style specifically for rules content using theme
	rulesBox := l.theme.Box.
		BorderForeground(l.theme.Colors.Blue).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	fmt.Fprintf(l.writer, "📋 %s\n",
		l.theme.Header.Render("Rules file content:"))

	// Apply box styling to the content
	box := rulesBox.Render(content)
	fmt.Fprintln(l.writer, box)
}

// ContextSummary logs a context summary with styled formatting
func (l *Logger) ContextSummary(cold, hot int) {
	fmt.Fprintf(l.writer, "📊 %s\n",
		l.theme.Header.Render("Context Summary:"))
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.theme.Highlight.Render(theme.IconBullet),
		l.theme.Muted.Render("Cold files:"),
		l.theme.Bold.Render(fmt.Sprintf("%d", cold)))
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.theme.Highlight.Render(theme.IconBullet),
		l.theme.Muted.Render("Hot files:"),
		l.theme.Bold.Render(fmt.Sprintf("%d", hot)))
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
			l.theme.Bold.Render(fmt.Sprintf("%d tokens (%s)", tokens, sizeStr))),
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
	fmt.Fprintf(l.writer, "\n❓ %s",
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
