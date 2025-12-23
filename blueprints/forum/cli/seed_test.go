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

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestRunSeed(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "forum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	dataDir = tmpDir
	defer func() { dataDir = oldDataDir }()

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run seed (will create database and add sample data)
	err = runSeed(cmd, nil)
	if err != nil {
		t.Fatalf("runSeed failed: %v", err)
	}
}

func TestRunSeed_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/does/not/exist"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSeed(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
