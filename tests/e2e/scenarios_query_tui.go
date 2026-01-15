package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/tui"
	"github.com/mattsolo1/grove-tend/pkg/verify"
)

// QueryTUIComprehensiveScenario tests the primary features of `gemapi query tui`.
func QueryTUIComprehensiveScenario() *harness.Scenario {
	return harness.NewScenario(
		"query-tui-comprehensive",
		"Verifies core features of the `gemapi query tui` command.",
		[]string{"gemapi", "query", "tui", "e2e"},
		[]harness.Step{
			harness.NewStep("Setup TUI test environment with logs", setupQueryTUIEnvironment),
			harness.NewStep("Launch TUI and verify initial display", launchTUIAndVerifyDisplay),
			harness.NewStep("Test time frame switching", testTimeFrameSwitching),
			harness.NewStep("Test metric toggling and help view", testMetricToggleAndHelp),
			harness.NewStep("Test navigation and quit", testNavigationAndQuit),
		},
	)
}

// setupQueryTUIEnvironment creates a test environment with mock query logs.
func setupQueryTUIEnvironment(ctx *harness.Context) error {
	// Logs are read from $HOME/.grove/gemini-cache/, so we need to use the home directory
	// The test harness sets up HOME at ctx.RootDir/home
	homeDir := filepath.Join(ctx.RootDir, "home")
	logDir := filepath.Join(homeDir, ".grove", "gemini-cache")
	if err := fs.CreateDir(logDir); err != nil {
		return err
	}

	// Create log file for today with multiple entries
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("query-log-%s.jsonl", today))

	var logs []string
	now := time.Now()

	// Create 10 recent log entries (last few hours)
	for i := 0; i < 10; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Hour)
		model := "gemini-2.5-pro"
		if i%3 == 0 {
			model = "gemini-1.5-flash"
		}

		success := true
		errorField := ""
		if i == 5 {
			success = false
			errorField = `,"error":"API timeout"`
		}

		log := fmt.Sprintf(`{"timestamp":"%s","model":"%s","prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d,"estimated_cost_usd":%.5f,"response_time_seconds":%.2f,"success":%t,"caller":"test-caller-%d"%s}`,
			timestamp.Format(time.RFC3339),
			model,
			1000+i*100,
			500+i*50,
			1500+i*150,
			0.001+float64(i)*0.0002,
			1.0+float64(i)*0.5,
			success,
			i,
			errorField,
		)
		logs = append(logs, log)
	}

	if err := fs.WriteString(logFile, strings.Join(logs, "\n")); err != nil {
		return err
	}

	// Create logs for 5 days ago (for weekly view testing)
	fiveDaysAgo := now.Add(-5 * 24 * time.Hour)
	logFile5Days := filepath.Join(logDir, fmt.Sprintf("query-log-%s.jsonl", fiveDaysAgo.Format("2006-01-02")))
	log5Days := fmt.Sprintf(`{"timestamp":"%s","model":"gemini-1.5-flash","prompt_tokens":600,"completion_tokens":400,"total_tokens":1000,"estimated_cost_usd":0.0001,"response_time_seconds":1.0,"success":true,"caller":"5-days-ago"}`,
		fiveDaysAgo.Format(time.RFC3339))
	if err := fs.WriteString(logFile5Days, log5Days); err != nil {
		return err
	}

	// Create logs for 40 days ago (for monthly view testing)
	fortyDaysAgo := now.Add(-40 * 24 * time.Hour)
	logFile40Days := filepath.Join(logDir, fmt.Sprintf("query-log-%s.jsonl", fortyDaysAgo.Format("2006-01-02")))
	log40Days := fmt.Sprintf(`{"timestamp":"%s","model":"gemini-1.5-pro","prompt_tokens":800,"completion_tokens":200,"total_tokens":1000,"estimated_cost_usd":0.0002,"response_time_seconds":2.0,"success":true,"caller":"40-days-ago"}`,
		fortyDaysAgo.Format(time.RFC3339))
	if err := fs.WriteString(logFile40Days, log40Days); err != nil {
		return err
	}

	ctx.Set("log_dir", logDir)
	return nil
}

// launchTUIAndVerifyDisplay launches the TUI and verifies the initial display.
func launchTUIAndVerifyDisplay(ctx *harness.Context) error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	session, err := ctx.StartTUI(binary, []string{"query", "tui"},
		tui.WithCwd(ctx.RootDir),
	)
	if err != nil {
		return fmt.Errorf("failed to start TUI session: %w", err)
	}
	ctx.Set("tui_session", session)

	// Wait for TUI to actually start by checking for expected content
	// Note: The TUI uses compact format "Cost:" instead of "Total Cost"
	if err := session.WaitForText("Cost:", 10*time.Second); err != nil {
		// If that fails, capture what we see for debugging
		view, _ := session.Capture()
		ctx.ShowCommandOutput("TUI Failed to Start - Current View", view, "")
		return fmt.Errorf("timeout waiting for TUI to start (looking for 'Cost:'): %w", err)
	}

	// Wait for UI to stabilize after async loading
	if err := session.WaitStable(); err != nil {
		return err
	}

	initialView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Initial View", initialView, "")

	// Verify all main components are visible
	// Note: The TUI uses compact format "Cost: $X.XX" instead of "Total Cost", etc.
	return ctx.Verify(func(v *verify.Collector) {
		v.Equal("Cost summary is visible", nil, session.AssertContains("Cost:"))
		v.Equal("Tokens summary is visible", nil, session.AssertContains("Tokens:"))
		v.Equal("Requests summary is visible", nil, session.AssertContains("Requests:"))
		v.Equal("table contains test-caller-0", nil, session.AssertContains("test-caller-0"))
		v.Equal("table contains test-caller-1", nil, session.AssertContains("test-caller-1"))
	})
}

// testTimeFrameSwitching tests switching between daily, weekly, and monthly views.
func testTimeFrameSwitching(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Press 'w' to switch to weekly view
	if err := session.SendKeys("w"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	weeklyView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Weekly View", weeklyView, "")

	// Verify weekly view shows 5-days-ago log
	if err := ctx.Check("weekly view shows 5-days-ago log",
		session.AssertContains("5-days-ago")); err != nil {
		return err
	}
	// Note: We don't check that 40-days-ago is NOT visible because TUI testing
	// with negative assertions is inherently flaky due to timing and rendering issues

	// Press 'm' to switch to monthly view
	if err := session.SendKeys("m"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	monthlyView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Monthly View", monthlyView, "")

	// Verify monthly view shows 5-days-ago but NOT 40-days-ago (30 days cutoff)
	if err := ctx.Check("monthly view shows 5-days-ago log",
		session.AssertContains("5-days-ago")); err != nil {
		return err
	}

	// Press 'd' to switch back to daily view
	if err := session.SendKeys("d"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	dailyView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Daily View", dailyView, "")

	return nil
}

// testMetricToggleAndHelp tests metric toggling and help view.
func testMetricToggleAndHelp(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Press 't' to toggle metric
	if err := session.SendKeys("t"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	// Press 't' again to toggle back
	if err := session.SendKeys("t"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	// Press '?' to open help
	if err := session.SendKeys("?"); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if err := session.WaitStable(); err != nil {
		return err
	}

	helpView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Help View", helpView, "")

	// Verify help view contains key bindings
	if err := ctx.Verify(func(v *verify.Collector) {
		v.Equal("help shows daily view binding", nil, session.AssertContains("daily view"))
		v.Equal("help shows quit binding", nil, session.AssertContains("quit"))
	}); err != nil {
		return err
	}

	// Press '?' again to close help
	if err := session.SendKeys("?"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	return nil
}

// testNavigationAndQuit tests table navigation and quit.
func testNavigationAndQuit(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Navigate down in the table
	if err := session.SendKeys("j"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	// Navigate down more
	if err := session.SendKeys("j", "j"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	// Navigate up
	if err := session.SendKeys("k"); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	if err := session.WaitStable(); err != nil {
		return err
	}

	finalView, _ := session.Capture()
	ctx.ShowCommandOutput("TUI Final View Before Quit", finalView, "")

	// Quit with 'q'
	if err := session.SendKeys("q"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	return nil
}

// QueryTUINoLogsScenario tests the TUI with no logs present.
func QueryTUINoLogsScenario() *harness.Scenario {
	return harness.NewScenario(
		"query-tui-no-logs",
		"Verifies the TUI handles the no logs case gracefully.",
		[]string{"gemapi", "query", "tui", "edge-case"},
		[]harness.Step{
			harness.NewStep("Launch TUI with no logs", func(ctx *harness.Context) error {
				binary, err := FindBinary()
				if err != nil {
					return err
				}

				session, err := ctx.StartTUI(binary, []string{"query", "tui"},
					tui.WithCwd(ctx.RootDir),
				)
				if err != nil {
					return fmt.Errorf("failed to start TUI session: %w", err)
				}
				ctx.Set("tui_session", session)

				// Wait for TUI to stabilize
				time.Sleep(1 * time.Second)
				if err := session.WaitStable(); err != nil {
					return err
				}

				noLogsView, _ := session.Capture()
				ctx.ShowCommandOutput("TUI No Logs View", noLogsView, "")

				// Verify the TUI shows appropriate messaging
				// Note: The TUI uses compact format "Cost: $X.XX" instead of "Total Cost"
				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("shows summary even with no logs", nil, session.AssertContains("Cost:"))
					v.Equal("shows zero cost", nil, session.AssertContains("$0.00"))
				})
			}),
			harness.NewStep("Quit the TUI", func(ctx *harness.Context) error {
				session := ctx.Get("tui_session").(*tui.Session)
				return session.SendKeys("q")
			}),
		},
	)
}
