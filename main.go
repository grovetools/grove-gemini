package main

import (
	"os"

	grovelogging "github.com/grovetools/core/logging"
	"github.com/grovetools/grove-gemini/cmd"
)

func main() {
	// CLI logging/progress goes to stderr so stdout can be used for piping LLM responses
	grovelogging.SetGlobalOutput(os.Stderr)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
