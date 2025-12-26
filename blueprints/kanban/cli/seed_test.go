package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSeed(t *testing.T) {
	cmd := NewSeed()

	if cmd == nil {
		t.Fatal("NewSeed() returned nil")
	}

	if cmd.Use != "seed" {
		t.Errorf("Use = %s; want seed", cmd.Use)
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

func TestSeed_CommandStructure(t *testing.T) {
	cmd := NewSeed()

	// Verify it's a valid cobra command
	if _, ok := interface{}(cmd).(*cobra.Command); !ok {
		t.Error("NewSeed() did not return a *cobra.Command")
	}
}

func TestSeed_LongDescription(t *testing.T) {
	cmd := NewSeed()

	// Verify the long description mentions what gets created
	long := cmd.Long
	keywords := []string{"demo", "workspace", "project", "issues"}

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
