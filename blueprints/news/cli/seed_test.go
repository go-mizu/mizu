package cli

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSeed(t *testing.T) {
	cmd := NewSeed()

	if cmd.Use != "seed" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "seed")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Should have subcommands
	if len(cmd.Commands()) == 0 {
		t.Error("seed command should have subcommands")
	}
}

func TestNewSeedHN(t *testing.T) {
	cmd := NewSeedHN()

	if cmd.Use != "hn" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "hn")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}

	// Check flags exist
	if cmd.Flags().Lookup("feed") == nil {
		t.Error("feed flag should exist")
	}
	if cmd.Flags().Lookup("limit") == nil {
		t.Error("limit flag should exist")
	}
	if cmd.Flags().Lookup("with-comments") == nil {
		t.Error("with-comments flag should exist")
	}
}

func TestNewSeedSample(t *testing.T) {
	cmd := NewSeedSample()

	if cmd.Use != "sample" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "sample")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestRunSeedSample(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "news-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	dataDir = tmpDir
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err = runSeedSample(cmd, nil)
	if err != nil {
		t.Fatalf("runSeedSample failed: %v", err)
	}
}

func TestRunSeedSample_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/does/not/exist"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSeedSample(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
