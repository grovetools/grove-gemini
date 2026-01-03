package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattsolo1/grove-gemini/pkg/config"
	"github.com/spf13/cobra"
)

var (
	dashboardDays int
)

func newQueryDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Interactive dashboard for GCP billing data visualization",
		Long: `Launches an interactive TUI dashboard that visualizes Gemini API billing data from BigQuery.

Features:
- Real-time cost visualization
- Daily/weekly/monthly views
- SKU breakdown table
- Interactive navigation`,
		RunE: runQueryDashboard,
	}

	// Get defaults from config
	defaultProject := config.GetDefaultProject("")
	defaultDataset := config.GetBillingDatasetID("")
	defaultTable := config.GetBillingTableID("")

	cmd.Flags().StringVarP(&billingProjectID, "project-id", "p", defaultProject, "GCP project ID")
	cmd.Flags().StringVarP(&billingDatasetID, "dataset-id", "d", defaultDataset, "BigQuery dataset ID containing billing export")
	cmd.Flags().StringVarP(&billingTableID, "table-id", "t", defaultTable, "BigQuery table ID for billing export")
	cmd.Flags().IntVar(&dashboardDays, "days", 30, "Number of days to display")

	// Only mark as required if no defaults are available
	if defaultDataset == "" {
		cmd.MarkFlagRequired("dataset-id")
	}
	if defaultTable == "" {
		cmd.MarkFlagRequired("table-id")
	}

	return cmd
}

func runQueryDashboard(cmd *cobra.Command, args []string) error {
	// Apply config defaults if flags weren't set
	billingProjectID = config.GetDefaultProject(billingProjectID)
	billingDatasetID = config.GetBillingDatasetID(billingDatasetID)
	billingTableID = config.GetBillingTableID(billingTableID)

	// Validate required fields
	if billingProjectID == "" {
		return fmt.Errorf("no GCP project specified. Use --project-id flag or set a default with 'gemapi config set project PROJECT_ID'")
	}

	if billingDatasetID == "" {
		return fmt.Errorf("no billing dataset specified. Use --dataset-id flag or set a default with 'gemapi config set billing DATASET_ID TABLE_ID'")
	}

	if billingTableID == "" {
		return fmt.Errorf("no billing table specified. Use --table-id flag or set a default with 'gemapi config set billing DATASET_ID TABLE_ID'")
	}

	// Initialize and run the TUI
	p := tea.NewProgram(
		newDashboardModel(billingProjectID, billingDatasetID, billingTableID, dashboardDays),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running dashboard: %w", err)
	}

	return nil
}
