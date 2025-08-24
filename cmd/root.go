package cmd

import (
	"github.com/mattsolo1/grove-core/cli"
	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

func init() {
	rootCmd = cli.NewStandardCommand("gemapi", "Tools for Google's Gemini API")

	// Add commands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newQueryCmd())
	rootCmd.AddCommand(newCountTokensCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newRequestCmd())
	rootCmd.AddCommand(newCacheCmd())
}

func Execute() error {
	return rootCmd.Execute()
}
