package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewServe(t *testing.T) {
	cmd := NewServe()

	if cmd == nil {
		t.Fatal("NewServe() returned nil")
	}

	if cmd.Use != "serve" {
		t.Errorf("Use = %s; want serve", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.Long == "" {
		t.Error("Long description is empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestServe_Flags(t *testing.T) {
	cmd := NewServe()

	// Check addr flag exists
	addrFlag := cmd.Flags().Lookup("addr")
	if addrFlag == nil {
		t.Error("addr flag not found")
	} else {
		if addrFlag.DefValue != ":8080" {
			t.Errorf("addr default = %s; want :8080", addrFlag.DefValue)
		}
		if addrFlag.Shorthand != "a" {
			t.Errorf("addr shorthand = %s; want a", addrFlag.Shorthand)
		}
	}
}

func TestServe_CommandStructure(t *testing.T) {
	cmd := NewServe()

	// Verify it's a valid cobra command
	if _, ok := interface{}(cmd).(*cobra.Command); !ok {
		t.Error("NewServe() did not return a *cobra.Command")
	}

	// Verify flags are registered
	flags := cmd.Flags()
	if flags == nil {
		t.Error("Flags() returned nil")
	}
}
