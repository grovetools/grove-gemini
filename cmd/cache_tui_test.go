package cmd

import (
	"testing"

	"github.com/grovetools/core/tui/keymap"
)

// TestCacheKeyMapAuditCoverage asserts the cache TUI keymap has no coverage
// gaps: every enabled binding appears in exactly one Sections() entry, no
// help-label lies, and no empty-help bindings. If this fails, the disable list
// or the Sections() membership in newCacheKeyMap is wrong — fix the code, not
// the test.
func TestCacheKeyMapAuditCoverage(t *testing.T) {
	if gaps := keymap.AuditCoverage(newCacheKeyMap(nil)); len(gaps) != 0 {
		t.Fatalf("audit gaps: %+v", gaps)
	}
}
