package cli

import (
	"testing"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "social" {
		t.Errorf("Use: got %q, want %q", rootCmd.Use, "social")
	}

	if rootCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	expectedCommands := []string{"serve", "init", "seed"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd.Use == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q not found", expected)
		}
	}
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	// Check --data flag
	dataFlag := flags.Lookup("data")
	if dataFlag == nil {
		t.Error("--data flag should exist")
	} else {
		want := defaultDataDir()
		if dataFlag.DefValue != want {
			t.Errorf("--data default: got %q, want %q", dataFlag.DefValue, want)
		}
	}

	// Check --addr flag
	addrFlag := flags.Lookup("addr")
	if addrFlag == nil {
		t.Error("--addr flag should exist")
	} else {
		if addrFlag.DefValue != ":8080" {
			t.Errorf("--addr default: got %q, want %q", addrFlag.DefValue, ":8080")
		}
	}

	// Check --dev flag
	devFlag := flags.Lookup("dev")
	if devFlag == nil {
		t.Error("--dev flag should exist")
	} else {
		if devFlag.DefValue != "false" {
			t.Errorf("--dev default: got %q, want %q", devFlag.DefValue, "false")
		}
	}
}

func TestVersion(t *testing.T) {
	// Version should have a default value
	if Version == "" {
		t.Error("Version should have a default value")
	}
}

func TestExecute(t *testing.T) {
	// This is a basic smoke test - we can't fully test Execute
	// without modifying os.Args, but we can verify it exists
	// Execute() returns error, which is the expected interface
	_ = Execute
}
