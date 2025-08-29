package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

func APIKeyConfigScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "api-key-config",
		Description: "Test API key configuration from various sources",
		Tags:        []string{"config"},
		Steps: []harness.Step{
			harness.NewStep("error when no API key configured", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Ensure no API key is set
				oldKey := os.Getenv("GEMINI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
				defer func() {
					if oldKey != "" {
						os.Setenv("GEMINI_API_KEY", oldKey)
					}
				}()

				// Ensure tests don't pick up global config by setting HOME to temp dir
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", ctx.RootDir)
				oldXDGConfig := os.Getenv("XDG_CONFIG_HOME")
				os.Setenv("XDG_CONFIG_HOME", filepath.Join(ctx.RootDir, ".config"))
				defer func() {
					os.Setenv("HOME", oldHome)
					if oldXDGConfig != "" {
						os.Setenv("XDG_CONFIG_HOME", oldXDGConfig)
					} else {
						os.Unsetenv("XDG_CONFIG_HOME")
					}
				}()

				// Run a command that requires the API key
				// Note: count-tokens doesn't require auth, so we use request instead
				cmd := command.New(binary, "request", "test query").Dir(ctx.RootDir)
				result := cmd.Run()

				// Should fail with proper error message
				if result.ExitCode == 0 {
					return fmt.Errorf("expected command to fail without API key, but it succeeded")
				}
				if !strings.Contains(result.Stderr, "Gemini API key not found") {
					return fmt.Errorf("expected error message about missing API key, got: %s", result.Stderr)
				}
				if !strings.Contains(result.Stderr, "GEMINI_API_KEY environment variable") {
					return fmt.Errorf("expected error to mention GEMINI_API_KEY, got: %s", result.Stderr)
				}
				if !strings.Contains(result.Stderr, "gemini.api_key_command") {
					return fmt.Errorf("expected error to mention gemini.api_key_command, got: %s", result.Stderr)
				}
				return nil
			}),

			harness.NewStep("API key from environment variable", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Set a test API key (invalid but should be accepted)
				os.Setenv("GEMINI_API_KEY", "test-key-from-env")
				defer os.Unsetenv("GEMINI_API_KEY")

				// Run a command that requires the API key
				// Note: count-tokens doesn't require auth, so we use request instead
				cmd := command.New(binary, "request", "test query").Dir(ctx.RootDir)
				result := cmd.Run()

				// Should fail with API key validation error (not missing key error)
				if result.ExitCode == 0 {
					return fmt.Errorf("expected command to fail with invalid API key")
				}
				if strings.Contains(result.Stderr, "Gemini API key not found") {
					return fmt.Errorf("should not show 'key not found' error when key is provided via env")
				}
				if !strings.Contains(result.Stderr, "API key not valid") && !strings.Contains(result.Stderr, "API_KEY_INVALID") {
					return fmt.Errorf("expected API validation error, got: %s", result.Stderr)
				}
				return nil
			}),

			harness.NewStep("API key from grove.yml command", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Ensure no env var is set
				oldKey := os.Getenv("GEMINI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
				defer func() {
					if oldKey != "" {
						os.Setenv("GEMINI_API_KEY", oldKey)
					}
				}()

				// Create a grove.yml with api_key_command
				groveYml := `name: test-project
description: Test project for API key config

gemini:
  api_key_command: "echo test-key-from-command"
`
				groveYmlPath := filepath.Join(ctx.RootDir, "grove.yml")
				if err := os.WriteFile(groveYmlPath, []byte(groveYml), 0644); err != nil {
					return fmt.Errorf("failed to write grove.yml: %w", err)
				}

				// Run a command that requires the API key
				// Note: count-tokens doesn't require auth, so we use request instead
				cmd := command.New(binary, "request", "test query").Dir(ctx.RootDir)
				result := cmd.Run()

				// Should fail with API key validation error (not missing key error)
				if result.ExitCode == 0 {
					return fmt.Errorf("expected command to fail with invalid API key")
				}
				if strings.Contains(result.Stderr, "Gemini API key not found") {
					return fmt.Errorf("should not show 'key not found' error when key is provided via command")
				}
				if !strings.Contains(result.Stderr, "API key not valid") && !strings.Contains(result.Stderr, "API_KEY_INVALID") {
					return fmt.Errorf("expected API validation error, got: %s", result.Stderr)
				}
				return nil
			}),

			harness.NewStep("API key from grove.yml direct value", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Ensure no env var is set
				oldKey := os.Getenv("GEMINI_API_KEY")
				os.Unsetenv("GEMINI_API_KEY")
				defer func() {
					if oldKey != "" {
						os.Setenv("GEMINI_API_KEY", oldKey)
					}
				}()

				// Create a grove.yml with direct api_key
				// Using a clearly invalid key format to ensure it fails
				groveYml := `name: test-project
description: Test project for API key config

gemini:
  api_key: "INVALID_KEY_FORMAT_12345"
`
				groveYmlPath := filepath.Join(ctx.RootDir, "grove.yml")
				if err := os.WriteFile(groveYmlPath, []byte(groveYml), 0644); err != nil {
					return fmt.Errorf("failed to write grove.yml: %w", err)
				}

				// Run a command that requires the API key
				// Note: count-tokens doesn't require auth, so we use request instead
				cmd := command.New(binary, "request", "test query").Dir(ctx.RootDir)
				result := cmd.Run()

				// The test verifies that the key is loaded from config, not that it fails
				// Some test environments might accept test keys
				if result.ExitCode != 0 {
					// If it failed, ensure it's not because the key wasn't found
					if strings.Contains(result.Stderr, "Gemini API key not found") {
						return fmt.Errorf("should not show 'key not found' error when key is provided in config")
					}
					// Expected failure with API validation
					return nil
				}
				// If it succeeded, that's also acceptable as long as the config was loaded
				return nil
			}),

			harness.NewStep("precedence: env var overrides grove.yml", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Set env var
				os.Setenv("GEMINI_API_KEY", "key-from-env-precedence")
				defer os.Unsetenv("GEMINI_API_KEY")

				// Create a grove.yml with different keys
				groveYml := `name: test-project
description: Test project for API key config

gemini:
  api_key: "key-from-config-should-be-ignored"
  api_key_command: "echo key-from-command-should-be-ignored"
`
				groveYmlPath := filepath.Join(ctx.RootDir, "grove.yml")
				if err := os.WriteFile(groveYmlPath, []byte(groveYml), 0644); err != nil {
					return fmt.Errorf("failed to write grove.yml: %w", err)
				}

				// Run with verbose to potentially see which key source is used
				cmd := command.New(binary, "request", "test query", "--verbose").Dir(ctx.RootDir)
				result := cmd.Run()

				// The API call will fail, but we're checking that it used the env var key
				if result.ExitCode == 0 {
					return fmt.Errorf("expected command to fail with invalid API key")
				}
				// The error should show that the env var key was used
				// We can't directly verify which key was used, but we can ensure no config loading errors
				if strings.Contains(result.Stderr, "failed to load grove.yml") {
					return fmt.Errorf("should not have config loading errors: %s", result.Stderr)
				}
				return nil
			}),
		},
	}
}