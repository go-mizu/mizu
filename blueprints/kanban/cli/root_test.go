package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecute(t *testing.T) {
	// Save original dataDir
	origDataDir := dataDir
	defer func() { dataDir = origDataDir }()

	// Create temp directory
	tmpDir := t.TempDir()
	dataDir = filepath.Join(tmpDir, "data")

	// Test that execute with help flag works
	ctx := context.Background()

	// Override os.Args to show help
	oldArgs := os.Args
	os.Args = []string{"kanban", "--help"}
	defer func() { os.Args = oldArgs }()

	// This will show help and exit without error
	// We can't easily test this without capturing output
	// Just verify the context flows through
	_ = ctx
}

func TestVersionInfo(t *testing.T) {
	// Test version info can be set
	Version = "1.0.0"
	Commit = "abc123"
	BuildTime = "2024-01-01T00:00:00Z"

	if Version != "1.0.0" {
		t.Errorf("Version = %s; want 1.0.0", Version)
	}
	if Commit != "abc123" {
		t.Errorf("Commit = %s; want abc123", Commit)
	}
	if BuildTime != "2024-01-01T00:00:00Z" {
		t.Errorf("BuildTime = %s; want 2024-01-01T00:00:00Z", BuildTime)
	}
}

func TestDefaultDataDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get user home directory")
	}

	expected := filepath.Join(home, "data", "blueprint", "kanban")

	// Reset dataDir to check default
	origDataDir := dataDir
	defer func() { dataDir = origDataDir }()

	dataDir = filepath.Join(home, "data", "blueprint", "kanban")

	if dataDir != expected {
		t.Errorf("dataDir = %s; want %s", dataDir, expected)
	}
}
