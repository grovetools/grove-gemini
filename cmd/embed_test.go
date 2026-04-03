package cmd

import (
	"testing"

	"github.com/grovetools/grove-gemini/pkg/gemini"
)

func TestNewEmbedCmd(t *testing.T) {
	cmd := newEmbedCmd()

	if cmd.Use != "embed [text...]" {
		t.Errorf("Expected Use to be 'embed [text...]', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	// Check flags
	modelFlag := cmd.Flags().Lookup("model")
	if modelFlag == nil {
		t.Fatal("Expected 'model' flag to be registered")
	}
	if modelFlag.DefValue != gemini.DefaultEmbeddingModel {
		t.Errorf("Expected model default to be %q, got %q", gemini.DefaultEmbeddingModel, modelFlag.DefValue)
	}
	if modelFlag.Shorthand != "m" {
		t.Errorf("Expected model shorthand to be 'm', got %q", modelFlag.Shorthand)
	}

	taskTypeFlag := cmd.Flags().Lookup("task-type")
	if taskTypeFlag == nil {
		t.Fatal("Expected 'task-type' flag to be registered")
	}
	if taskTypeFlag.DefValue != "RETRIEVAL_DOCUMENT" {
		t.Errorf("Expected task-type default to be 'RETRIEVAL_DOCUMENT', got %q", taskTypeFlag.DefValue)
	}
	if taskTypeFlag.Shorthand != "t" {
		t.Errorf("Expected task-type shorthand to be 't', got %q", taskTypeFlag.Shorthand)
	}
}

func TestEmbedCmdRegistered(t *testing.T) {
	// Verify the embed command is registered in root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "embed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'embed' command to be registered in root command")
	}
}
