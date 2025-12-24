package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCommand(t *testing.T) {
	if initCmd.Use != "init" {
		t.Errorf("Use: got %q, want %q", initCmd.Use, "init")
	}

	if initCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if initCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if initCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestRunInit(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
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
	dbPath := filepath.Join(tmpDir, "social.duckdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file should have been created")
	}
}

func TestRunInit_CreatesDataDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a subdirectory that doesn't exist yet
	newDataDir := filepath.Join(tmpDir, "newsubdir", "data")

	// Save and restore global state
	oldDataDir := dataDir
	dataDir = newDataDir
	defer func() { dataDir = oldDataDir }()

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Run init
	err = runInit(cmd, nil)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDataDir); os.IsNotExist(err) {
		t.Error("Data directory should have been created")
	}
}

func TestRunInit_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/cannot/be/created"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runInit(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestRunInit_Idempotent(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "social-test-*")
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

	// Run init twice - should be idempotent
	if err := runInit(cmd, nil); err != nil {
		t.Fatalf("First runInit failed: %v", err)
	}

	if err := runInit(cmd, nil); err != nil {
		t.Fatalf("Second runInit failed: %v", err)
	}
}
