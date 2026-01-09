package cmd

import (
	grovelogging "github.com/mattsolo1/grove-core/logging"
)

// Package-level unified logger instance
var ulog = grovelogging.NewUnifiedLogger("grove-gemini.cmd")
