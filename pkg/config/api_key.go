package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	core_config "github.com/grovetools/core/config"
	core_errors "github.com/grovetools/core/errors"
)

// apiKeyCommandTimeout bounds how long api_key_command may run. Offline, the
// typical command (gcloud) retries for tens of seconds; without this bound a
// single agent launch stalls the whole critical path. Package-level so tests
// can override it for a fast deadline-expiry check.
var apiKeyCommandTimeout = 10 * time.Second

//go:generate sh -c "cd ../.. && go run ./tools/schema-generator/"

// GeminiConfig defines the structure for the 'gemini' extension in grove.yml
type GeminiConfig struct {
	APIKey        string `yaml:"api_key" jsonschema:"description=Direct API key for Google Gemini" jsonschema_extras:"x-layer=global,x-priority=200,x-sensitive=true,x-important=true,x-hint=Consider using api_key_command to fetch from a secrets manager"`
	APIKeyCommand string `yaml:"api_key_command" jsonschema:"description=Shell command to retrieve API key (e.g. gcloud secrets or 1password)" jsonschema_extras:"x-layer=global,x-priority=60,x-important=true"`
}

// ResolveAPIKey resolves the Gemini API key from multiple sources in order of precedence:
// 1. GEMINI_API_KEY environment variable
// 2. Command output from gemini.api_key_command in grove.yml
// 3. Direct value from gemini.api_key in grove.yml
func ResolveAPIKey() (string, error) {
	// First priority: Environment variable
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Second and third priority: grove.yml configuration
	cfg, err := core_config.LoadDefault()
	if err != nil {
		// Check if it's a "config not found" error
		if core_errors.Is(err, core_errors.ErrCodeConfigNotFound) {
			// No config file - this is okay, but we have no API key
			return "", fmt.Errorf("Gemini API key not found. Please configure it using one of:\n" +
				"  1. Set GEMINI_API_KEY environment variable\n" +
				"  2. Add 'gemini.api_key_command' to grove.yml\n" +
				"  3. Add 'gemini.api_key' to grove.yml")
		}
		// Some other error loading config
		return "", fmt.Errorf("failed to load grove.yml: %w", err)
	}

	// Parse the gemini extension
	var geminiCfg GeminiConfig
	if err := cfg.UnmarshalExtension("gemini", &geminiCfg); err != nil {
		// Extension exists but couldn't be parsed
		return "", fmt.Errorf("failed to parse 'gemini' configuration from grove.yml: %w", err)
	}

	// Second priority: Command execution
	if geminiCfg.APIKeyCommand != "" {
		return runAPIKeyCommand(geminiCfg.APIKeyCommand)
	}

	// Third priority: Direct API key
	if geminiCfg.APIKey != "" {
		return geminiCfg.APIKey, nil
	}

	// No API key found anywhere
	return "", fmt.Errorf("Gemini API key not found. Please configure it using one of:\n" +
		"  1. Set GEMINI_API_KEY environment variable\n" +
		"  2. Add 'gemini.api_key_command' to grove.yml\n" +
		"  3. Add 'gemini.api_key' to grove.yml")
}

// runAPIKeyCommand executes the configured api_key_command under a bounded
// deadline (apiKeyCommandTimeout) and returns its trimmed output. The deadline
// prevents an offline secrets command (e.g. gcloud retrying its network calls)
// from stalling the agent-launch critical path for tens of seconds.
func runAPIKeyCommand(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiKeyCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command) //nolint:gosec // command comes from trusted grove.yml config
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("api_key_command '%s' timed out after %s (network offline?): %w", command, apiKeyCommandTimeout, err)
		}
		return "", fmt.Errorf("failed to execute api_key_command '%s': %w", command, err)
	}

	apiKey := strings.TrimSpace(string(output))
	if apiKey == "" {
		return "", fmt.Errorf("api_key_command '%s' returned empty output", command)
	}
	return apiKey, nil
}
