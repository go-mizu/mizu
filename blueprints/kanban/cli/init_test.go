package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewInit(t *testing.T) {
	cmd := NewInit()

	if cmd == nil {
		t.Fatal("NewInit() returned nil")
	}

	if cmd.Use != "init" {
		t.Errorf("Use = %s; want init", cmd.Use)
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

func TestInit_CommandStructure(t *testing.T) {
	cmd := NewInit()

	// Verify it's a valid cobra command
	if _, ok := interface{}(cmd).(*cobra.Command); !ok {
		t.Error("NewInit() did not return a *cobra.Command")
	}
}

func TestInit_LongDescription(t *testing.T) {
	cmd := NewInit()

	// Verify the long description mentions key features
	long := cmd.Long
	keywords := []string{"database", "tables", "Users", "Workspaces", "Projects", "Issues"}

	for _, kw := range keywords {
		found := false
		for i := 0; i < len(long)-len(kw)+1; i++ {
			if long[i:i+len(kw)] == kw {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Long description missing keyword: %s", kw)
		}
	}
}
