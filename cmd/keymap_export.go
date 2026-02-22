package cmd

import (
	"github.com/grovetools/core/tui/keymap"
)

// QueryTuiKeymapInfo returns the keymap metadata for the grove-gemini query TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func QueryTuiKeymapInfo() keymap.TUIInfo {
	return keymap.MakeTUIInfo(
		"gemini-query",
		"grove-gemini",
		"Query usage analytics and logs viewer",
		newQueryTuiKeyMap(nil),
	)
}

// DashboardKeymapInfo returns the keymap metadata for the grove-gemini dashboard TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func DashboardKeymapInfo() keymap.TUIInfo {
	return keymap.MakeTUIInfo(
		"gemini-dashboard",
		"grove-gemini",
		"BigQuery billing dashboard viewer",
		newDashboardKeyMap(nil),
	)
}

// CacheKeymapInfo returns the keymap metadata for the grove-gemini cache TUI.
// Used by the grove keys registry generator to aggregate all TUI keybindings.
func CacheKeymapInfo() keymap.TUIInfo {
	return keymap.MakeTUIInfo(
		"gemini-cache",
		"grove-gemini",
		"Response cache browser and manager",
		newCacheKeyMap(nil),
	)
}
