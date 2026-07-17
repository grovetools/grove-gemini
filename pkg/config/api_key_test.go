package config

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestRunAPIKeyCommand_Timeout verifies that a command which hangs longer than
// the deadline is aborted promptly (rather than blocking the agent-launch
// critical path) and returns a clear timeout error. Uses a short injected
// timeout so the test finishes quickly instead of waiting the real 10s.
func TestRunAPIKeyCommand_Timeout(t *testing.T) {
	orig := apiKeyCommandTimeout
	apiKeyCommandTimeout = 200 * time.Millisecond
	defer func() { apiKeyCommandTimeout = orig }()

	start := time.Now()
	_, err := runAPIKeyCommand("sleep 60")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a command that outruns the deadline, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got: %v", err)
	}
	// Should abort near the deadline, well before the 60s sleep completes.
	if elapsed > 5*time.Second {
		t.Fatalf("command was not aborted promptly: took %s", elapsed)
	}
}

// TestRunAPIKeyCommand_Success verifies the happy path: output is captured and
// trimmed, and no timeout fires for a fast command.
func TestRunAPIKeyCommand_Success(t *testing.T) {
	key, err := runAPIKeyCommand("printf '  my-secret-key\\n'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "my-secret-key" {
		t.Fatalf("expected trimmed output 'my-secret-key', got %q", key)
	}
}

// TestRunAPIKeyCommand_EmptyOutput verifies an empty command result is rejected.
func TestRunAPIKeyCommand_EmptyOutput(t *testing.T) {
	_, err := runAPIKeyCommand("true")
	if err == nil {
		t.Fatal("expected an error for empty command output, got nil")
	}
	if !strings.Contains(err.Error(), "empty output") {
		t.Fatalf("expected an empty-output error, got: %v", err)
	}
}

// TestRunAPIKeyCommand_DeadlineExceededClassification is a lower-level guard that
// the deadline-expiry branch is what fires (not the generic exec failure).
func TestRunAPIKeyCommand_DeadlineExceededClassification(t *testing.T) {
	orig := apiKeyCommandTimeout
	apiKeyCommandTimeout = 100 * time.Millisecond
	defer func() { apiKeyCommandTimeout = orig }()

	_, err := runAPIKeyCommand("sleep 30")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The wrapped error should ultimately stem from the context deadline.
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected deadline-exceeded-derived error, got: %v", err)
	}
}
