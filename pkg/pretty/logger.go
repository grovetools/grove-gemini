package pretty

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	corelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around the grove-core PrettyLogger with Gemini-specific helpers.
type Logger struct {
	*corelogging.PrettyLogger
	writer io.Writer
	styles Styles
	log    *logrus.Entry // For structured logging when needed
}

// Styles contains all the lipgloss styles for different log types
type Styles struct {
	// Headers and sections
	WorkDir      lipgloss.Style
	Model        lipgloss.Style
	Section      lipgloss.Style
	
	// Status messages
	Info         lipgloss.Style
	Success      lipgloss.Style
	Warning      lipgloss.Style
	Error        lipgloss.Style
	
	// Special elements
	Path         lipgloss.Style
	Number       lipgloss.Style
	Percentage   lipgloss.Style
	Duration     lipgloss.Style
	Command      lipgloss.Style
	
	// Icons
	Icon         lipgloss.Style
	SuccessIcon  lipgloss.Style
	WarningIcon  lipgloss.Style
	ErrorIcon    lipgloss.Style
	
	// Progress indicators
	ProgressBar  lipgloss.Style
	Spinner      lipgloss.Style
	
	// Boxes and containers
	Box          lipgloss.Style
	TokenBox     lipgloss.Style
}

// DefaultStyles returns the default styling configuration
func DefaultStyles() Styles {
	// Define color palette
	blue := lipgloss.Color("#3498db")
	green := lipgloss.Color("#2ecc71")
	yellow := lipgloss.Color("#f39c12")
	red := lipgloss.Color("#e74c3c")
	purple := lipgloss.Color("#9b59b6")
	cyan := lipgloss.Color("#1abc9c")
	gray := lipgloss.Color("#95a5a6")
	darkGray := lipgloss.Color("#7f8c8d")
	
	return Styles{
		// Headers and sections
		WorkDir: lipgloss.NewStyle().
			Foreground(blue).
			Bold(true),
		
		Model: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true),
		
		Section: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true).
			MarginTop(1),
		
		// Status messages
		Info: lipgloss.NewStyle().
			Foreground(blue),
		
		Success: lipgloss.NewStyle().
			Foreground(green),
		
		Warning: lipgloss.NewStyle().
			Foreground(yellow),
		
		Error: lipgloss.NewStyle().
			Foreground(red).
			Bold(true),
		
		// Special elements
		Path: lipgloss.NewStyle().
			Foreground(cyan).
			Italic(true),
		
		Number: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true),
		
		Percentage: lipgloss.NewStyle().
			Foreground(green).
			Bold(true),
		
		Duration: lipgloss.NewStyle().
			Foreground(gray),
		
		Command: lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true),
		
		// Icons
		Icon: lipgloss.NewStyle().
			MarginRight(1),
		
		SuccessIcon: lipgloss.NewStyle().
			Foreground(green).
			MarginRight(1),
		
		WarningIcon: lipgloss.NewStyle().
			Foreground(yellow).
			MarginRight(1),
		
		ErrorIcon: lipgloss.NewStyle().
			Foreground(red).
			MarginRight(1),
		
		// Progress indicators
		ProgressBar: lipgloss.NewStyle().
			Foreground(green),
		
		Spinner: lipgloss.NewStyle().
			Foreground(blue),
		
		// Boxes and containers
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(darkGray).
			Padding(1, 2),
		
		TokenBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1),
	}
}

// New creates a new Gemini-specific pretty logger.
func New() *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger(),
		writer:       os.Stderr,
		styles:       DefaultStyles(),
		log:          corelogging.NewLogger("grove-gemini"),
	}
}

// NewWithLogger creates a new logger with a specific structured logging backend.
func NewWithLogger(log *logrus.Entry) *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger(),
		writer:       os.Stderr,
		styles:       DefaultStyles(),
		log:          log,
	}
}

// NewWithWriter creates a new Logger with a custom writer
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		PrettyLogger: corelogging.NewPrettyLogger().WithWriter(w),
		writer:       w,
		styles:       DefaultStyles(),
		log:          corelogging.NewLogger("grove-gemini"),
	}
}

// WorkingDirectory logs the working directory
func (l *Logger) WorkingDirectory(dir string) {
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.styles.Icon.Render("🏠"),
		l.styles.Info.Render("Working directory:"),
		l.styles.Path.Render(dir))
}

// FoundRulesFile logs that a rules file was found
func (l *Logger) FoundRulesFile(path string) {
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.styles.Icon.Render("📋"),
		l.styles.Info.Render("Found rules file:"),
		l.styles.Path.Render(path))
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.WarningIcon.Render("⚠️"),
		l.styles.Warning.Render(message))
}

// Info logs an info message
func (l *Logger) Info(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("📁"),
		l.styles.Info.Render(message))
}

// Success logs a success message (override to use Gemini style)
func (l *Logger) Success(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.SuccessIcon.Render("✅"),
		l.styles.Success.Render(message))
}

// Error logs an error message
func (l *Logger) Error(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.ErrorIcon.Render("❌"),
		l.styles.Error.Render(message))
}

// Model logs the model being used
func (l *Logger) Model(model string) {
	// Log structured data if backend available
	if l.log != nil {
		l.log.WithField("model", model).Info("Calling Gemini API")
	}
	// Display pretty UI
	fmt.Fprintf(l.writer, "\n%s %s %s\n\n",
		l.styles.Icon.Render("🤖"),
		l.styles.Info.Render("Calling Gemini API with model:"),
		l.styles.Model.Render(model))
}

// UploadProgress logs file upload progress
func (l *Logger) UploadProgress(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("📤"),
		l.styles.Info.Render(message))
}

// UploadComplete logs successful file upload
func (l *Logger) UploadComplete(filename string, duration time.Duration) {
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.styles.SuccessIcon.Render("✅"),
		l.styles.Success.Render(filename),
		l.styles.Duration.Render(fmt.Sprintf("(%.2fs)", duration.Seconds())))
}

// GeneratingResponse logs that response generation has started
func (l *Logger) GeneratingResponse() {
	fmt.Fprintf(l.writer, "\n%s %s\n",
		l.styles.Icon.Render("🤖"),
		l.styles.Info.Render("Generating response..."))
}

// FilesIncluded displays the list of files that will be included in the request
func (l *Logger) FilesIncluded(files []string) {
	if len(files) == 0 {
		return
	}
	
	fmt.Fprintf(l.writer, "\n%s %s\n",
		l.styles.Icon.Render("📁"),
		l.styles.Section.Render("Files attached to request:"))
	
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
		if displayName == "CLAUDE.md" || displayName == "context" || displayName == "cached-context" {
			fmt.Fprintf(l.writer, "  • %s\n", l.styles.Path.Render(file))
		} else if isPromptFile {
			fmt.Fprintf(l.writer, "  • %s (prompt)\n", l.styles.Path.Render(file))
		} else {
			fmt.Fprintf(l.writer, "  • %s\n", l.styles.Path.Render(displayName))
		}
	}
}

// TokenUsage displays token usage statistics in a styled box
func (l *Logger) TokenUsage(cached, dynamic, completion, promptTokens int, responseTime time.Duration, isNewCache bool) {
	// First, log structured data to backend if available
	if l.log != nil {
		l.log.WithFields(logrus.Fields{
			"cached_tokens":       cached,
			"dynamic_tokens":      dynamic,
			"completion_tokens":   completion,
			"user_prompt_tokens":  promptTokens,
			"total_prompt_tokens": cached + dynamic,
			"response_time_ms":    responseTime.Milliseconds(),
			"is_new_cache":        isNewCache,
		}).Info("Token usage summary")
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
	cachedStyle := l.styles.Number
	
	if isNewCache {
		cachedLabel = "New Cache Tokens:"
		cacheHitRateLabel = "Cache Hit Rate (fresh):"
		cachedStyle = l.styles.Warning // Use warning style for new cache tokens
	}
	
	// Create the content with proper formatting
	content := []string{
		fmt.Sprintf("%s %s",
			l.styles.Info.Render(cachedLabel),
			cachedStyle.Render(fmt.Sprintf("%d tokens", cached))),
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Hot (Dynamic):"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens", dynamic))),
	}
	
	// Add user prompt tokens if available
	if promptTokens > 0 {
		content = append(content, fmt.Sprintf("%s %s",
			l.styles.Info.Render("User Prompt:"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens", promptTokens))))
	}
	
	content = append(content, []string{
		"--------------------------------",
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Total Prompt:"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens", totalPrompt))),
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Completion:"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens", completion))),
		"--------------------------------",
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Total API Usage:"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens", totalAPIUsage))),
		fmt.Sprintf("%s %s",
			l.styles.Info.Render(cacheHitRateLabel),
			l.styles.Percentage.Render(fmt.Sprintf("%.1f%%", cacheHitRate))),
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Response Time:"),
			l.styles.Duration.Render(fmt.Sprintf("%.2fs", responseTime.Seconds()))),
	}...)
	
	// Join with newlines and apply box styling
	box := l.styles.TokenBox.Render(strings.Join(content, "\n"))
	
	fmt.Fprintf(l.writer, "\n%s %s\n%s\n",
		l.styles.Icon.Render("📊"),
		l.styles.Section.Render("Token usage:"),
		box)
}

// CacheInfo logs cache-related information
func (l *Logger) CacheInfo(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("📁"),
		l.styles.Info.Render(message))
}

// CacheCreated logs successful cache creation
func (l *Logger) CacheCreated(cacheID string, expires time.Time) {
	relativeTime := formatRelativeTime(expires)
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.styles.SuccessIcon.Render("✅"),
		l.styles.Success.Render("Cache created:"),
		l.styles.Path.Render(cacheID))
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.styles.Icon.Render("📅"),
		l.styles.Info.Render("Expires"),
		l.styles.Duration.Render(relativeTime))
}

// ChangedFiles logs files that have changed
func (l *Logger) ChangedFiles(files []string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("🔄"),
		l.styles.Warning.Render("Cached files have changed:"))
	for _, file := range files {
		fmt.Fprintf(l.writer, "   %s %s\n",
			l.styles.Icon.Render("•"),
			l.styles.Path.Render(file))
	}
}

// CreatingCache logs cache creation start
func (l *Logger) CreatingCache() {
	fmt.Fprintf(l.writer, "\n%s %s\n",
		l.styles.WarningIcon.Render("💰"),
		l.styles.Warning.Render("Creating new cache (one-time operation)..."))
}

// NoCache logs when no cache is found
func (l *Logger) NoCache() {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("🆕"),
		l.styles.Info.Render("No existing cache found"))
}

// CacheValid logs when cache is valid
func (l *Logger) CacheValid(until time.Time) {
	relativeTime := formatRelativeTime(until)
	fmt.Fprintf(l.writer, "%s %s (%s %s)\n",
		l.styles.SuccessIcon.Render("✅"),
		l.styles.Success.Render("Cache is valid"),
		l.styles.Info.Render("expires"),
		l.styles.Duration.Render(relativeTime))
}

// CacheExpired logs when cache has expired
func (l *Logger) CacheExpired(at time.Time) {
	relativeTime := formatRelativeTime(at)
	fmt.Fprintf(l.writer, "%s %s (%s)\n",
		l.styles.Icon.Render("⏰"),
		l.styles.Warning.Render("Cache expired"),
		l.styles.Duration.Render(relativeTime))
}

// CacheFrozen logs when cache is frozen
func (l *Logger) CacheFrozen() {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("❄️"),
		l.styles.Info.Render("Cache is frozen by @freeze-cache directive"))
}

// CacheDisabled logs when cache is disabled
func (l *Logger) CacheDisabled() {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("🚫"),
		l.styles.Warning.Render("Cache disabled by @disable-cache directive"))
}

// TTL logs the cache TTL
func (l *Logger) TTL(ttl string) {
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.styles.Icon.Render("⏱️"),
		l.styles.Info.Render("Using cache TTL from @expire-time directive:"),
		l.styles.Duration.Render(ttl))
}

// CacheDisabledByDefault logs when cache is disabled by default (opt-in model)
func (l *Logger) CacheDisabledByDefault() {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("ℹ️"),
		l.styles.Info.Render("Caching is disabled by default"))
	fmt.Fprintf(l.writer, "   %s\n",
		l.styles.Info.Render("To enable caching, add @enable-cache to your .grove/rules file"))
}

// CacheWarning displays a prominent warning about experimental caching and costs
func (l *Logger) CacheWarning() {
	// Create a warning box style
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#ff9800")).
		Foreground(lipgloss.Color("#ff9800")).
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
		l.styles.Info.Render("Estimated tokens:"),
		l.styles.Number.Render(fmt.Sprintf("%d", count)))
}

// ResponseWritten logs successful response write
func (l *Logger) ResponseWritten(path string) {
	fmt.Fprintf(l.writer, "%s %s %s\n",
		l.styles.SuccessIcon.Render("✅"),
		l.styles.Success.Render("Response written to:"),
		l.styles.Path.Render(path))
}

// Tip logs a helpful tip
func (l *Logger) Tip(message string) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("💡"),
		l.styles.Info.Render(message))
}

// RulesFileContent displays the rules file content in a styled box
func (l *Logger) RulesFileContent(content string) {
	// Create a box style specifically for rules content
	rulesBox := l.styles.Box.
		BorderForeground(lipgloss.Color("#3498db")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)
	
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("📋"),
		l.styles.Section.Render("Rules file content:"))
	
	// Apply box styling to the content
	box := rulesBox.Render(content)
	fmt.Fprintln(l.writer, box)
}

// ContextSummary logs a context summary with styled formatting
func (l *Logger) ContextSummary(cold, hot int) {
	fmt.Fprintf(l.writer, "%s %s\n",
		l.styles.Icon.Render("📊"),
		l.styles.Section.Render("Context Summary:"))
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.styles.Icon.Render("•"),
		l.styles.Info.Render("Cold files:"),
		l.styles.Number.Render(fmt.Sprintf("%d", cold)))
	fmt.Fprintf(l.writer, "  %s %s %s\n",
		l.styles.Icon.Render("•"),
		l.styles.Info.Render("Hot files:"),
		l.styles.Number.Render(fmt.Sprintf("%d", hot)))
}

// CacheCreationPrompt shows cache creation details and prompts for confirmation
func (l *Logger) CacheCreationPrompt(tokens int, sizeBytes int64, ttl time.Duration) bool {
	// Create a prominent box for the cache creation warning
	warningBox := l.styles.Box.
		BorderForeground(lipgloss.Color("#f39c12")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)
	
	// Format size
	sizeStr := formatFileSize(sizeBytes)
	relativeTime := formatRelativeTime(time.Now().Add(ttl))
	
	content := []string{
		l.styles.Warning.Bold(true).Render("NEW CACHE CREATION REQUIRED"),
		"",
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Cache size:"),
			l.styles.Number.Render(fmt.Sprintf("%d tokens (%s)", tokens, sizeStr))),
		fmt.Sprintf("%s %s",
			l.styles.Info.Render("Expires:"),
			l.styles.Duration.Render(relativeTime)),
		"",
		l.styles.Info.Render("Creating a cache will upload context to Gemini's servers."),
		l.styles.Info.Render("This is a one-time operation that may incur costs."),
	}
	
	box := warningBox.Render(strings.Join(content, "\n"))
	fmt.Fprintln(l.writer)
	fmt.Fprintln(l.writer, box)
	
	// Prompt for confirmation
	fmt.Fprintf(l.writer, "\n%s %s",
		l.styles.Icon.Render("❓"),
		l.styles.Warning.Render("Do you want to create this cache? [y/N]: "))
	
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