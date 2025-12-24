package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewInit(t *testing.T) {
	cmd := NewInit()

	if cmd.Use != "init" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "init")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestRunInit(t *testing.T) {
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

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run init
	err = runInit(cmd, nil)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "news.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file should have been created")
	}
}

func TestRunInit_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/does/not/exist"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runInit(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}
