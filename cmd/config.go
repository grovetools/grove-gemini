package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/grovetools/grove-gemini/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gemapi configuration",
		Long:  `Configure default settings for gemapi, such as the default GCP project.`,
	}

	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigGetCmd())

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
	}

	cmd.AddCommand(newConfigSetProjectCmd())
	cmd.AddCommand(newConfigSetBillingCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get configuration values",
	}

	cmd.AddCommand(newConfigGetProjectCmd())
	cmd.AddCommand(newConfigGetBillingCmd())

	return cmd
}

func newConfigSetProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project [PROJECT_ID]",
		Short: "Set the default GCP project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			projectID := args[0]

			cfg, err := config.LoadGCPConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg.DefaultProject = projectID

			if err := config.SaveGCPConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			configPath, _ := config.GetConfigPath()
			ulog.Success("Configuration updated").
				Field("project_id", projectID).
				Field("config_path", configPath).
				Pretty(fmt.Sprintf("Default GCP project set to: %s\nConfiguration saved to: %s", projectID, configPath)).
				PrettyOnly().
				Log(ctx)
			return nil
		},
	}
}

func newConfigGetProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project",
		Short: "Get the default GCP project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var output strings.Builder

			// Show all sources
			output.WriteString("GCP Project Resolution Order:\n")
			output.WriteString("1. Command flag: --project-id\n")

			envProject := config.GetDefaultProject("")
			if envProject != "" {
				output.WriteString(fmt.Sprintf("2. Environment variable GCP_PROJECT_ID: %s\n", envProject))
			} else {
				output.WriteString("2. Environment variable GCP_PROJECT_ID: (not set)\n")
			}

			cfg, err := config.LoadGCPConfig()
			if err == nil && cfg.DefaultProject != "" {
				output.WriteString(fmt.Sprintf("3. Saved configuration: %s\n", cfg.DefaultProject))
			} else {
				output.WriteString("3. Saved configuration: (not set)\n")
			}

			currentProject := config.GetDefaultProject("")
			output.WriteString(fmt.Sprintf("\nCurrent default project: %s\n", currentProject))

			ulog.Info("GCP project configuration").
				Field("current_project", currentProject).
				Field("env_project", envProject).
				Pretty(output.String()).
				PrettyOnly().
				Log(ctx)

			return nil
		},
	}
}

func newConfigSetBillingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "billing [DATASET_ID] [TABLE_ID]",
		Short: "Set the default BigQuery billing dataset and table",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			datasetID := args[0]
			tableID := args[1]

			cfg, err := config.LoadGCPConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg.BillingDatasetID = datasetID
			cfg.BillingTableID = tableID

			if err := config.SaveGCPConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			configPath, _ := config.GetConfigPath()
			ulog.Success("Billing configuration updated").
				Field("dataset_id", datasetID).
				Field("table_id", tableID).
				Field("config_path", configPath).
				Pretty(fmt.Sprintf("Billing dataset set to: %s\nBilling table set to: %s\nConfiguration saved to: %s", datasetID, tableID, configPath)).
				PrettyOnly().
				Log(ctx)
			return nil
		},
	}
}

func newConfigGetBillingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "billing",
		Short: "Get the default BigQuery billing configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var output strings.Builder

			// Show all sources
			output.WriteString("Billing Dataset Resolution Order:\n")
			output.WriteString("1. Command flag: --dataset-id\n")

			envDataset := config.GetBillingDatasetID("")
			if envDataset != "" {
				output.WriteString(fmt.Sprintf("2. Environment variable GCP_BILLING_DATASET_ID: %s\n", envDataset))
			} else {
				output.WriteString("2. Environment variable GCP_BILLING_DATASET_ID: (not set)\n")
			}

			cfg, err := config.LoadGCPConfig()
			if err == nil && cfg.BillingDatasetID != "" {
				output.WriteString(fmt.Sprintf("3. Saved configuration: %s\n", cfg.BillingDatasetID))
			} else {
				output.WriteString("3. Saved configuration: (not set)\n")
			}

			currentDataset := config.GetBillingDatasetID("")
			output.WriteString(fmt.Sprintf("\nCurrent billing dataset: %s\n", currentDataset))

			output.WriteString("\nBilling Table Resolution Order:\n")
			output.WriteString("1. Command flag: --table-id\n")

			envTable := config.GetBillingTableID("")
			if envTable != "" {
				output.WriteString(fmt.Sprintf("2. Environment variable GCP_BILLING_TABLE_ID: %s\n", envTable))
			} else {
				output.WriteString("2. Environment variable GCP_BILLING_TABLE_ID: (not set)\n")
			}

			if err == nil && cfg.BillingTableID != "" {
				output.WriteString(fmt.Sprintf("3. Saved configuration: %s\n", cfg.BillingTableID))
			} else {
				output.WriteString("3. Saved configuration: (not set)\n")
			}

			currentTable := config.GetBillingTableID("")
			output.WriteString(fmt.Sprintf("\nCurrent billing table: %s\n", currentTable))

			ulog.Info("Billing configuration").
				Field("current_dataset", currentDataset).
				Field("current_table", currentTable).
				Field("env_dataset", envDataset).
				Field("env_table", envTable).
				Pretty(output.String()).
				PrettyOnly().
				Log(ctx)

			return nil
		},
	}
}