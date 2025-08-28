package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// GeminiRequestLogForTest matches the GeminiRequestLog structure for parsing
type GeminiRequestLogForTest struct {
	Timestamp      time.Time `json:"timestamp"`
	Model          string    `json:"model"`
	PromptText     string    `json:"prompt_text"`
	AttachedFiles  []string  `json:"attached_files"`
	TotalFiles     int       `json:"total_files"`
	CacheID        string    `json:"cache_id,omitempty"`
	WorkingDir     string    `json:"working_dir,omitempty"`
}

// GeminiDebugLogScenario tests that 'gemapi request' with GROVE_DEBUG=1 creates a request log file
func GeminiDebugLogScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "gemini-debug-log",
		Description: "Tests that 'gemapi request' with GROVE_DEBUG=1 creates a request log file",
		Tags:        []string{"gemini", "debug", "log"},
		Steps: []harness.Step{
			harness.NewStep("Setup project with context files", func(ctx *harness.Context) error {
				// Create context files
				fs.WriteString(filepath.Join(ctx.RootDir, "main.go"), `package main

import "fmt"

// TestMarker12345 - unique marker for testing
func main() {
    fmt.Println("Hello from main.go")
}`)
				
				fs.WriteString(filepath.Join(ctx.RootDir, "utils.go"), `package main

// UtilsMarker67890 - another unique marker
func HelperFunction() string {
    return "Helper from utils.go"
}`)

				// Create .grove/rules file to enable context
				groveDir := filepath.Join(ctx.RootDir, ".grove")
				fs.CreateDir(groveDir)
				
				// Simple rules file to include Go files
				rulesContent := `# Include all Go files
*.go
`
				fs.WriteString(filepath.Join(groveDir, "rules"), rulesContent)

				return nil
			}),
			
			harness.NewStep("Run gemapi request with debug enabled", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				// Create a prompt file
				promptPath := filepath.Join(ctx.RootDir, "test-prompt.txt")
				fs.WriteString(promptPath, "This is a test prompt for debugging")

				cmd := command.New(
					binary, "request",
					"-p", "This is a test prompt for debugging",
					"--context", "main.go",
					"--context", "utils.go",
					"--no-cache", // Disable caching to simplify test
				).Dir(ctx.RootDir)
				
				// Enable debug mode
				cmd.Env("GROVE_DEBUG=1")
				// Provide a fake API key so the client initializes
				cmd.Env("GEMINI_API_KEY=fake-key-for-testing")

				// We expect this command to fail because the API key is fake,
				// but the log should be written before the API call is made.
				result := cmd.Run()
				ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
				
				// The command should fail due to invalid API key, which is expected
				// We're testing that the log is created before the failure
				
				return nil // Don't fail the step due to the expected API error.
			}),
			
			harness.NewStep("Verify Gemini request log file", func(ctx *harness.Context) error {
				// The log should be created in .grove/logs/gemini_prompts/
				logDir := filepath.Join(ctx.RootDir, ".grove", "logs", "gemini_prompts")
				logPattern := filepath.Join(logDir, "unknown_job-*-gemini-request.json")
				
				matches, err := filepath.Glob(logPattern)
				if err != nil {
					return fmt.Errorf("error searching for gemini log file: %w", err)
				}
				if len(matches) == 0 {
					// Check if logs might be in a different location
					altLogDir := filepath.Join(ctx.RootDir, ".grove", "logs")
					if entries, err := os.ReadDir(altLogDir); err == nil {
						var foundDirs []string
						for _, entry := range entries {
							if entry.IsDir() {
								foundDirs = append(foundDirs, entry.Name())
							}
						}
						return fmt.Errorf("expected gemini request log file was not created. Pattern: %s\nFound directories in .grove/logs: %v", logPattern, foundDirs)
					}
					return fmt.Errorf("expected gemini request log file was not created. Pattern: %s", logPattern)
				}

				// Read and parse the JSON log
				logFile := matches[0]
				content, err := fs.ReadString(logFile)
				if err != nil {
					return fmt.Errorf("failed to read gemini log file %s: %w", logFile, err)
				}

				var logData GeminiRequestLogForTest
				if err := json.Unmarshal([]byte(content), &logData); err != nil {
					return fmt.Errorf("failed to parse log file JSON: %w\nContent:\n%s", err, content)
				}

				// Verify the content
				if logData.PromptText != "This is a test prompt for debugging" {
					return fmt.Errorf("log contains incorrect prompt text. Expected: 'This is a test prompt for debugging', Got: '%s'", logData.PromptText)
				}
				
				// Should have at least the two context files we specified
				if len(logData.AttachedFiles) < 2 {
					return fmt.Errorf("log should list at least 2 attached files, found %d: %v", len(logData.AttachedFiles), logData.AttachedFiles)
				}
				
				// Check that our files are in the attached files list
				hasMainGo := false
				hasUtilsGo := false
				for _, file := range logData.AttachedFiles {
					if strings.HasSuffix(file, "main.go") {
						hasMainGo = true
					}
					if strings.HasSuffix(file, "utils.go") {
						hasUtilsGo = true
					}
				}
				
				if !hasMainGo {
					return fmt.Errorf("log is missing main.go in attached files: %v", logData.AttachedFiles)
				}
				if !hasUtilsGo {
					return fmt.Errorf("log is missing utils.go in attached files: %v", logData.AttachedFiles)
				}
				
				// Verify other fields
				if logData.TotalFiles != len(logData.AttachedFiles) {
					return fmt.Errorf("total_files (%d) doesn't match attached_files length (%d)", logData.TotalFiles, len(logData.AttachedFiles))
				}
				
				if logData.WorkingDir == "" {
					return fmt.Errorf("working_dir should not be empty")
				}
				
				ctx.ShowCommandOutput("Gemini Log Verification", "✓ Gemini request log created and verified successfully.", "")
				ctx.ShowCommandOutput("Log Details", fmt.Sprintf("Log file: %s\nAttached files: %d\nPrompt: %s", filepath.Base(logFile), len(logData.AttachedFiles), logData.PromptText), "")
				return nil
			}),
			
			harness.NewStep("Cleanup", func(ctx *harness.Context) error {
				// Clean up the test directory
				// This is optional but helps keep test environments clean
				return nil
			}),
		},
	}
}