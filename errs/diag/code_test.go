package diag_test

import (
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

func TestCodeValid(t *testing.T) {
	for _, tt := range []struct {
		c    diag.Code
		want bool
	}{
		{"MZ1042", true},
		{"MZ0000", true},
		{"MZ9999", true},
		{"", false},
		{"MZ104", false},
		{"MZ10425", false},
		{"mz1042", false},
		{"XZ1042", false},
		{"MZ104a", false},
		{"MZ-042", false},
		{"MZ 042", false},
		{"1042MZ", false},
	} {
		if got := tt.c.Valid(); got != tt.want {
			t.Errorf("Code(%q).Valid() = %v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestCodeNumber(t *testing.T) {
	for _, tt := range []struct {
		c    diag.Code
		want int
	}{
		{"MZ1042", 1042},
		{"MZ0007", 7},
		{"MZ0000", 0},
		{"MZ9999", 9999},
		{"nonsense", 0},
		{"", 0},
	} {
		if got := tt.c.Number(); got != tt.want {
			t.Errorf("Code(%q).Number() = %d, want %d", tt.c, got, tt.want)
		}
	}
}

// The explain command and the docs URL are built out of the code rather than
// stored beside it, so a diagnostic cannot point at the page for a different
// one.
func TestCodeExplainAndDocs(t *testing.T) {
	c := diag.Code("MZ1042")
	if got, want := c.Explain(), "mizu explain MZ1042"; got != want {
		t.Errorf("Explain() = %q, want %q", got, want)
	}
	if got, want := c.Docs(), "https://mizu.dev/e/MZ1042"; got != want {
		t.Errorf("Docs() = %q, want %q", got, want)
	}
}

// A diagnostic with no code has nothing to point at, and pointing at
// mizu.dev/e/ is worse than pointing at nothing.
func TestCodeWithNothingToExplain(t *testing.T) {
	for _, c := range []diag.Code{"", "nonsense", "MZ104"} {
		if got := c.Explain(); got != "" {
			t.Errorf("Code(%q).Explain() = %q, want nothing", c, got)
		}
		if got := c.Docs(); got != "" {
			t.Errorf("Code(%q).Docs() = %q, want nothing", c, got)
		}
	}
}
