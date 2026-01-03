package cmd

import "github.com/spf13/cobra"

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Gemini API usage, metrics, and billing data from Google Cloud",
		Long:  `Provides commands to query various Google Cloud services for Gemini API metrics, token usage logs, and billing information.`,
	}

	// Add an explicit 'tui' command
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch an interactive TUI to visualize local query logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryTUI()
		},
	}
	cmd.AddCommand(tuiCmd)

	// Subcommands will be added here
	cmd.AddCommand(newQueryMetricsCmd())
	cmd.AddCommand(newQueryTokensCmd())
	cmd.AddCommand(newQueryBillingCmd())
	cmd.AddCommand(newQueryRequestsCmd())
	cmd.AddCommand(newQueryExploreCmd())
	cmd.AddCommand(newQueryLocalCmd())

	return cmd
}