package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configFileName = "gcp-config.json"
)

type GCPConfig struct {
	DefaultProject    string `json:"default_project,omitempty"`
	BillingDatasetID  string `json:"billing_dataset_id,omitempty"`
	BillingTableID    string `json:"billing_table_id,omitempty"`
}

// GetConfigPath returns the path to the GCP config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	groveDir := filepath.Join(homeDir, ".grove", "gemini-cache")
	return filepath.Join(groveDir, configFileName), nil
}

// LoadGCPConfig loads the GCP configuration from disk
func LoadGCPConfig() (*GCPConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &GCPConfig{}, nil
		}
		return nil, err
	}

	var config GCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveGCPConfig saves the GCP configuration to disk
func SaveGCPConfig(config *GCPConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetDefaultProject returns the default GCP project, checking in order:
// 1. Explicitly provided value
// 2. Environment variable GCP_PROJECT_ID
// 3. Saved configuration
func GetDefaultProject(explicit string) string {
	// If explicitly provided, use that
	if explicit != "" {
		return explicit
	}

	// Check environment variable
	if envProject := os.Getenv("GCP_PROJECT_ID"); envProject != "" {
		return envProject
	}

	// Check saved config
	config, err := LoadGCPConfig()
	if err == nil && config.DefaultProject != "" {
		return config.DefaultProject
	}

	return ""
}

// GetBillingDatasetID returns the billing dataset ID, checking in order:
// 1. Explicitly provided value
// 2. Environment variable GCP_BILLING_DATASET_ID
// 3. Saved configuration
func GetBillingDatasetID(explicit string) string {
	// If explicitly provided, use that
	if explicit != "" {
		return explicit
	}

	// Check environment variable
	if envDataset := os.Getenv("GCP_BILLING_DATASET_ID"); envDataset != "" {
		return envDataset
	}

	// Check saved config
	config, err := LoadGCPConfig()
	if err == nil && config.BillingDatasetID != "" {
		return config.BillingDatasetID
	}

	return ""
}

// GetBillingTableID returns the billing table ID, checking in order:
// 1. Explicitly provided value
// 2. Environment variable GCP_BILLING_TABLE_ID
// 3. Saved configuration
func GetBillingTableID(explicit string) string {
	// If explicitly provided, use that
	if explicit != "" {
		return explicit
	}

	// Check environment variable
	if envTable := os.Getenv("GCP_BILLING_TABLE_ID"); envTable != "" {
		return envTable
	}

	// Check saved config
	config, err := LoadGCPConfig()
	if err == nil && config.BillingTableID != "" {
		return config.BillingTableID
	}

	return ""
}