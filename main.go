package main

import (
	"os"

	grovelogging "github.com/mattsolo1/grove-core/logging"
	"github.com/mattsolo1/grove-gemini/cmd"
)

func main() {
	// CLI output goes to stdout (stderr is for errors only)
	grovelogging.SetGlobalOutput(os.Stdout)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
