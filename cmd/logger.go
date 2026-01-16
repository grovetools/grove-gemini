package cmd

import (
	grovelogging "github.com/grovetools/core/logging"
)

// Package-level unified logger instance
var ulog = grovelogging.NewUnifiedLogger("grove-gemini.cmd")
