package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	core_config "github.com/mattsolo1/grove-core/config"
	core_errors "github.com/mattsolo1/grove-core/errors"
)

// GeminiConfig defines the structure for the 'gemini' extension in grove.yml
type GeminiConfig struct {
	APIKey        string `yaml:"api_key"`
	APIKeyCommand string `yaml:"api_key_command"`
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
		cmd := exec.Command("sh", "-c", geminiCfg.APIKeyCommand)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to execute api_key_command '%s': %w", geminiCfg.APIKeyCommand, err)
		}
		apiKey := strings.TrimSpace(string(output))
		if apiKey == "" {
			return "", fmt.Errorf("api_key_command '%s' returned empty output", geminiCfg.APIKeyCommand)
		}
		return apiKey, nil
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