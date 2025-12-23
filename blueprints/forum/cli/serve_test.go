package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestNewServe(t *testing.T) {
	cmd := NewServe()

	if cmd.Use != "serve" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "serve")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestRunServe_InvalidPath(t *testing.T) {
	// Save and restore global state
	oldDataDir := dataDir
	dataDir = "/nonexistent/path/that/does/not/exist"
	defer func() { dataDir = oldDataDir }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runServe(cmd, nil)
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestRunServe_StartsServer(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "forum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore global state
	oldDataDir := dataDir
	oldAddr := addr
	dataDir = tmpDir
	addr = "127.0.0.1:0" // Use random available port
	defer func() {
		dataDir = oldDataDir
		addr = oldAddr
	}()

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	// Server should start and stop when context is cancelled
	err = runServe(cmd, nil)
	// nil error is expected when context is cancelled
	if err != nil && ctx.Err() == nil {
		t.Fatalf("runServe failed unexpectedly: %v", err)
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		dev  bool
		want string
	}{
		{true, "development"},
		{false, "production"},
	}

	for _, tt := range tests {
		got := modeString(tt.dev)
		if got != tt.want {
			t.Errorf("modeString(%v): got %q, want %q", tt.dev, got, tt.want)
		}
	}
}
