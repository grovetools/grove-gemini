package cmd

import (
	"fmt"

	"github.com/mattsolo1/grove-gemini/pkg/config"
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
			fmt.Printf("Default GCP project set to: %s\n", projectID)
			fmt.Printf("Configuration saved to: %s\n", configPath)
			return nil
		},
	}
}

func newConfigGetProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "project",
		Short: "Get the default GCP project",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show all sources
			fmt.Println("GCP Project Resolution Order:")
			fmt.Println("1. Command flag: --project-id")

			if envProject := config.GetDefaultProject(""); envProject != "" {
				fmt.Printf("2. Environment variable GCP_PROJECT_ID: %s\n", envProject)
			} else {
				fmt.Println("2. Environment variable GCP_PROJECT_ID: (not set)")
			}

			cfg, err := config.LoadGCPConfig()
			if err == nil && cfg.DefaultProject != "" {
				fmt.Printf("3. Saved configuration: %s\n", cfg.DefaultProject)
			} else {
				fmt.Println("3. Saved configuration: (not set)")
			}

			fmt.Printf("\nCurrent default project: %s\n", config.GetDefaultProject(""))

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
			fmt.Printf("Billing dataset set to: %s\n", datasetID)
			fmt.Printf("Billing table set to: %s\n", tableID)
			fmt.Printf("Configuration saved to: %s\n", configPath)
			return nil
		},
	}
}

func newConfigGetBillingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "billing",
		Short: "Get the default BigQuery billing configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show all sources
			fmt.Println("Billing Dataset Resolution Order:")
			fmt.Println("1. Command flag: --dataset-id")

			if envDataset := config.GetBillingDatasetID(""); envDataset != "" {
				fmt.Printf("2. Environment variable GCP_BILLING_DATASET_ID: %s\n", envDataset)
			} else {
				fmt.Println("2. Environment variable GCP_BILLING_DATASET_ID: (not set)")
			}

			cfg, err := config.LoadGCPConfig()
			if err == nil && cfg.BillingDatasetID != "" {
				fmt.Printf("3. Saved configuration: %s\n", cfg.BillingDatasetID)
			} else {
				fmt.Println("3. Saved configuration: (not set)")
			}

			fmt.Printf("\nCurrent billing dataset: %s\n", config.GetBillingDatasetID(""))

			fmt.Println("\nBilling Table Resolution Order:")
			fmt.Println("1. Command flag: --table-id")

			if envTable := config.GetBillingTableID(""); envTable != "" {
				fmt.Printf("2. Environment variable GCP_BILLING_TABLE_ID: %s\n", envTable)
			} else {
				fmt.Println("2. Environment variable GCP_BILLING_TABLE_ID: (not set)")
			}

			if err == nil && cfg.BillingTableID != "" {
				fmt.Printf("3. Saved configuration: %s\n", cfg.BillingTableID)
			} else {
				fmt.Println("3. Saved configuration: (not set)")
			}

			fmt.Printf("\nCurrent billing table: %s\n", config.GetBillingTableID(""))

			return nil
		},
	}
}