package cmd

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grovetools/core/tui/keymap"
)

// NOTE — there is deliberately no keymap.AuditCoverage test for
// queryTuiKeyMap / dashboardKeyMap here, unlike cache_tui_test.go. Both report
// 41 pre-existing "missing-from-sections" gaps because, unlike newCacheKeyMap,
// their constructors never disable the promoted keymap.Base fields they do not
// handle (Left/Right/Home/End/Confirm/…/Fold*). That is untouched by the
// canon-60 rebinds — the counts are 41/41 both before and after — and it is
// invisible to `grove keys audit`, since MakeTUIInfo exports Sections() only.
// Cleaning it up is a separate change: add a disable list mirroring
// newCacheKeyMap, then add the audit test here.

// TestQueryToggleMetricChord pins the canon-60 rebind: `t` is a bare namespace
// prefix (it arms, it never fires), and only the full `tm` chord resolves to
// toggle-metric. A regression to a flat `t` would make the first assertion fail.
func TestQueryToggleMetricChord(t *testing.T) {
	km := newQueryTuiKeyMap(nil)
	host := keymap.NewWhichKeyHost(nil, km.Namespaces()...)

	res, _, cmd := host.ProcessChord(keyMsg("t"))
	if res != keymap.ChordPending {
		t.Fatalf("`t` alone: got %v, want ChordPending (a prefix must arm, not fire)", res)
	}
	if cmd == nil {
		t.Error("an armed namespace prefix must return the show-delay tick cmd")
	}
	if !host.Armed() {
		t.Error("host should report Armed() after `t`")
	}

	res, matched, _ := host.ProcessChord(keyMsg("m"))
	if res != keymap.ChordMatched {
		t.Fatalf("`t`+`m`: got %v, want ChordMatched", res)
	}
	if got := matched.Keys(); len(got) != 1 || got[0] != "tm" {
		t.Errorf("matched binding keys = %v, want [tm]", got)
	}
}

// TestCacheAnalyticsChord is the gemini-cache half: `v` arms, `va` resolves to
// analytics, and the flat `y` confirm alias is gone (canon 60 §5.6) so `yy` can
// arm in this TUI in future.
func TestCacheAnalyticsChord(t *testing.T) {
	km := newCacheKeyMap(nil)
	host := keymap.NewWhichKeyHost(nil, km.Namespaces()...)

	if res, _, _ := host.ProcessChord(keyMsg("v")); res != keymap.ChordPending {
		t.Fatalf("`v` alone: got %v, want ChordPending", res)
	}
	res, matched, _ := host.ProcessChord(keyMsg("a"))
	if res != keymap.ChordMatched {
		t.Fatalf("`v`+`a`: got %v, want ChordMatched", res)
	}
	if got := matched.Keys(); len(got) != 1 || got[0] != "va" {
		t.Errorf("matched binding keys = %v, want [va]", got)
	}

	for _, k := range km.Base.Confirm.Keys() {
		if k == "y" {
			t.Errorf("confirm still binds flat `y`: keys = %v (canon 60 §5.6 drops it)", km.Base.Confirm.Keys())
		}
	}
}

func keyMsg(s string) tea.KeyMsg {
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
