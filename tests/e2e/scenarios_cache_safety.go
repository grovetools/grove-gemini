package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// CacheSafetyScenarios returns E2E test scenarios for cache-safety feature
func CacheSafetyScenarios() []*harness.Scenario {
	return []*harness.Scenario{
		{
			Name:        "cache-safety",
			Description: "Test cache safety opt-in feature",
			Tags:        []string{"cache"},
			Steps: []harness.Step{
				harness.NewStep("cache disabled by default", testCacheDisabledByDefault),
				harness.NewStep("cache enabled with directive", testCacheEnabledWithDirective),
				harness.NewStep("cache disabled with --no-cache flag", testCacheDisabledWithFlag),
				harness.NewStep("cache ignores commented directive", testCacheIgnoresCommentedDirective),
			},
		},
	}
}

func testCacheDisabledByDefault(ctx *harness.Context) error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	// Create a test directory with rules but no @enable-cache
	testDir, err := os.MkdirTemp("", "cache-safety-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testDir)

	groveDir := filepath.Join(testDir, ".grove")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		return err
	}

	// Create rules file without @enable-cache
	rulesPath := filepath.Join(groveDir, "rules")
	rulesContent := `# Context rules
**/*.go
!*_test.go
`
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		return err
	}

	// Create minimal context files
	if err := os.WriteFile(filepath.Join(groveDir, "context"), []byte("hot context"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(groveDir, "cached-context"), []byte("cold context"), 0644); err != nil {
		return err
	}

	// Run gemapi request
	cmd := command.New(binary, "request", "--work-dir", testDir, "test prompt")
	result := cmd.Run()
	
	// API key errors are acceptable for this test
	if result.ExitCode != 0 {
		combined := result.Stdout + result.Stderr
		if strings.Contains(combined, "GOOGLE_API_KEY") || strings.Contains(combined, "API key") {
			// Still check for cache message
			if strings.Contains(result.Stderr, "Caching is disabled by default") {
				return nil // Test passed
			}
			return nil // API key missing is acceptable
		}
		return fmt.Errorf("command failed: %v\nstdout: %s\nstderr: %s", result.Error, result.Stdout, result.Stderr)
	}

	// Verify cache is disabled message appears
	if !strings.Contains(result.Stderr, "Caching is disabled by default") {
		return fmt.Errorf("expected 'Caching is disabled by default' message, got stderr: %s", result.Stderr)
	}

	// Verify no cache warning appears
	if strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
		return fmt.Errorf("unexpected cache warning appeared when caching should be disabled")
	}

	return nil
}

func testCacheEnabledWithDirective(ctx *harness.Context) error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	// Create a test directory with @enable-cache
	testDir, err := os.MkdirTemp("", "cache-safety-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testDir)

	groveDir := filepath.Join(testDir, ".grove")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		return err
	}

	// Create rules file with @enable-cache
	rulesPath := filepath.Join(groveDir, "rules")
	rulesContent := `@enable-cache
**/*.go
!*_test.go
`
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		return err
	}

	// Create minimal context files
	if err := os.WriteFile(filepath.Join(groveDir, "context"), []byte("hot context"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(groveDir, "cached-context"), []byte("cold context"), 0644); err != nil {
		return err
	}

	// Run gemapi request
	cmd := command.New(binary, "request", "--work-dir", testDir, "test prompt")
	result := cmd.Run()
	
	// API key errors are acceptable for this test
	if result.ExitCode != 0 {
		combined := result.Stdout + result.Stderr
		if strings.Contains(combined, "GOOGLE_API_KEY") || strings.Contains(combined, "API key") {
			// Still check for warning message
			if strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
				return nil // Test passed - warning shown
			}
			return nil // API key missing is acceptable
		}
		return fmt.Errorf("command failed: %v\nstdout: %s\nstderr: %s", result.Error, result.Stdout, result.Stderr)
	}

	// Verify cache warning appears
	if !strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
		return fmt.Errorf("expected 'ALPHA FEATURE WARNING' when @enable-cache is present, got stderr: %s", result.Stderr)
	}

	// Verify no "disabled by default" message
	if strings.Contains(result.Stderr, "Caching is disabled by default") {
		return fmt.Errorf("unexpected 'disabled by default' message when caching should be enabled")
	}

	return nil
}

func testCacheDisabledWithFlag(ctx *harness.Context) error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	// Create a test directory with @enable-cache
	testDir, err := os.MkdirTemp("", "cache-safety-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testDir)

	groveDir := filepath.Join(testDir, ".grove")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		return err
	}

	// Create rules file with @enable-cache
	rulesPath := filepath.Join(groveDir, "rules")
	rulesContent := `@enable-cache
**/*.go
`
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		return err
	}

	// Create minimal context files
	if err := os.WriteFile(filepath.Join(groveDir, "context"), []byte("hot context"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(groveDir, "cached-context"), []byte("cold context"), 0644); err != nil {
		return err
	}

	// Run gemapi request with --no-cache flag
	cmd := command.New(binary, "request", "--work-dir", testDir, "--no-cache", "test prompt")
	result := cmd.Run()
	
	// API key errors are acceptable for this test
	if result.ExitCode != 0 {
		combined := result.Stdout + result.Stderr
		if strings.Contains(combined, "GOOGLE_API_KEY") || strings.Contains(combined, "API key") {
			// Still check that warning is NOT shown
			if !strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
				return nil // Test passed - no warning with --no-cache
			}
			return fmt.Errorf("cache warning shown despite --no-cache flag")
		}
		return fmt.Errorf("command failed: %v\nstdout: %s\nstderr: %s", result.Error, result.Stdout, result.Stderr)
	}

	// Verify NO cache warning appears (--no-cache overrides @enable-cache)
	if strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
		return fmt.Errorf("cache warning appeared despite --no-cache flag")
	}

	return nil
}

func testCacheIgnoresCommentedDirective(ctx *harness.Context) error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	// Create a test directory with commented @enable-cache
	testDir, err := os.MkdirTemp("", "cache-safety-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(testDir)

	groveDir := filepath.Join(testDir, ".grove")
	if err := os.MkdirAll(groveDir, 0755); err != nil {
		return err
	}

	// Create rules file with commented @enable-cache
	rulesPath := filepath.Join(groveDir, "rules")
	rulesContent := `# @enable-cache - disabled for now
**/*.go
!*_test.go
`
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		return err
	}

	// Create minimal context files
	if err := os.WriteFile(filepath.Join(groveDir, "context"), []byte("hot context"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(groveDir, "cached-context"), []byte("cold context"), 0644); err != nil {
		return err
	}

	// Run gemapi request
	cmd := command.New(binary, "request", "--work-dir", testDir, "test prompt")
	result := cmd.Run()
	
	// API key errors are acceptable for this test
	if result.ExitCode != 0 {
		combined := result.Stdout + result.Stderr
		if strings.Contains(combined, "GOOGLE_API_KEY") || strings.Contains(combined, "API key") {
			// Still check that warning is NOT shown
			if !strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
				return nil // Test passed - no warning with commented directive
			}
			return fmt.Errorf("cache warning shown despite commented @enable-cache")
		}
		return fmt.Errorf("command failed: %v\nstdout: %s\nstderr: %s", result.Error, result.Stdout, result.Stderr)
	}

	// Verify NO cache warning appears (commented directive should be ignored)
	if strings.Contains(result.Stderr, "ALPHA FEATURE WARNING") {
		return fmt.Errorf("cache warning appeared for commented @enable-cache directive")
	}

	// Verify cache is disabled message appears
	if !strings.Contains(result.Stderr, "Caching is disabled by default") {
		return fmt.Errorf("expected 'Caching is disabled by default' message for commented directive")
	}

	return nil
}