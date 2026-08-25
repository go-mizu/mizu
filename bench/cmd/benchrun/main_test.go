package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSaysWhatItDoesNotKnow(t *testing.T) {
	err := run("benchmark", ".", &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), `unknown command "benchmark"`) {
		t.Errorf("run of an unknown command gave %v", err)
	}
}

func TestRunTablePrintsTheTable(t *testing.T) {
	var out bytes.Buffer

	if err := run("table", ".", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "| `log/info/json` |") {
		t.Errorf("table printed %q", out.String())
	}
}

// TestModuleRootWalksUp is what makes the command work from anywhere inside
// bench, which is where somebody running it is standing.
func TestModuleRootWalksUp(t *testing.T) {
	root, err := moduleRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	fromDeeper, err := moduleRoot(filepath.Join(root, "micro"))
	if err != nil {
		t.Fatal(err)
	}
	if fromDeeper != root {
		t.Errorf("from micro the root is %s, want %s", fromDeeper, root)
	}
}

// TestModuleRootWantsTheBenchmarkModule is why it looks for the module line
// rather than for any go.mod. The toolkit next door has one too, and linting it
// would report nothing and mean nothing.
func TestModuleRootWantsTheBenchmarkModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/other\n\ngo 1.27\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	if _, err := moduleRoot(dir); err == nil {
		t.Error("a module that is not the benchmark module was accepted")
	}
}

func TestPlural(t *testing.T) {
	tests := map[int]string{0: "0 problems", 1: "1 problem", 2: "2 problems", 40: "40 problems"}
	for n, want := range tests {
		if got := plural(n, "problem", "problems"); got != want {
			t.Errorf("plural(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestHasModuleLine(t *testing.T) {
	tests := map[string]struct {
		in   string
		want bool
	}{
		"the benchmark module": {"module github.com/go-mizu/mizu/bench\n\ngo 1.27\n", true},
		"indented":             {"  module github.com/go-mizu/mizu/bench  \n", true},
		"the toolkit":          {"module github.com/go-mizu/mizu\n", false},
		"a longer path":        {"module github.com/go-mizu/mizu/bench/micro\n", false},
		"nothing":              {"", false},
		"only a comment": {
			"// module github.com/go-mizu/mizu/bench\nmodule example.com/x\n", false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := hasModuleLine(tt.in); got != tt.want {
				t.Errorf("hasModuleLine = %v, want %v", got, tt.want)
			}
		})
	}
}
