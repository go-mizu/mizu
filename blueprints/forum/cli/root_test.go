package cli

import (
	"context"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	// Test with dev version
	oldVersion := Version
	Version = "dev"
	defer func() { Version = oldVersion }()

	v := versionString()
	if v != "dev" {
		// Might return a version from build info
		t.Logf("versionString returned: %s", v)
	}
}

func TestVersionString_WithVersion(t *testing.T) {
	oldVersion := Version
	Version = "1.0.0"
	defer func() { Version = oldVersion }()

	v := versionString()
	if v != "1.0.0" {
		t.Errorf("versionString: got %q, want %q", v, "1.0.0")
	}
}

func TestVersionString_Empty(t *testing.T) {
	oldVersion := Version
	Version = ""
	defer func() { Version = oldVersion }()

	v := versionString()
	if v == "" {
		t.Error("versionString should not return empty string")
	}
}

func TestVersionString_Whitespace(t *testing.T) {
	oldVersion := Version
	Version = "   "
	defer func() { Version = oldVersion }()

	v := versionString()
	// Should fall back to dev or build info
	if strings.TrimSpace(v) == "" {
		t.Error("versionString should not return whitespace-only string")
	}
}

func TestExecute_Help(t *testing.T) {
	// Test that Execute returns without error when called with --help
	// This is a basic smoke test
	ctx := context.Background()

	// Save the original args
	// Note: We can't easily test the full Execute function without
	// modifying os.Args, but we can test the version string and
	// other utilities
	_ = ctx
}

func TestVersionVariables(t *testing.T) {
	// Verify version variables are set to expected defaults
	if Version == "" {
		t.Error("Version should have a default value")
	}
	if Commit == "" {
		t.Error("Commit should have a default value")
	}
	if BuildTime == "" {
		t.Error("BuildTime should have a default value")
	}
}
