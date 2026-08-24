package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRealLabelFile validates the file the repository actually uses. It needs
// no network and no token, so a bad label file fails in the unit test job
// rather than at the moment somebody merges to main and the sync runs.
func TestRealLabelFile(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "labels.yml")
	labels, err := LoadLabels(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) < 40 {
		t.Errorf("got %d labels, which is fewer than the set this repository is meant to have", len(labels))
	}

	// The two labels the merge automation looks issues up by. Without both of
	// them on a tracking issue, on-merge cannot find anything.
	need := []string{"tracking", "milestone/M0", "milestone/M9"}
	have := map[string]bool{}
	for _, l := range labels {
		have[l.Name] = true
	}
	for _, n := range need {
		if !have[n] {
			t.Errorf("%s is missing, and the merge automation depends on it", n)
		}
	}

	// Every milestone label the roadmap defines needs its own colour group to
	// be readable, but more importantly the description has to say which
	// milestone it is, since that is what shows in the label picker.
	for _, l := range labels {
		if strings.HasPrefix(l.Name, "milestone/") && l.Description == "" {
			t.Errorf("%s has no description", l.Name)
		}
	}
}

func TestLoadLabelsRejectsBadFiles(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, want string
	}{
		{
			"duplicate name",
			"groups:\n  - prefix: type\n    color: \"1d76db\"\n    labels:\n      - name: type/bug\n        description: One\n      - name: type/bug\n        description: Two\n",
			"defined twice",
		},
		{
			"prefix not honoured",
			"groups:\n  - prefix: type\n    color: \"1d76db\"\n    labels:\n      - name: bug\n        description: One\n",
			"must be named type/something",
		},
		{
			"no description",
			"groups:\n  - prefix: \"\"\n    color: \"1d76db\"\n    labels:\n      - name: bug\n",
			"no description",
		},
		{
			"short colour",
			"groups:\n  - prefix: \"\"\n    color: \"fff\"\n    labels:\n      - name: bug\n        description: One\n",
			"not six hex digits",
		},
		{
			"unknown field",
			"groups:\n  - prefix: \"\"\n    color: \"1d76db\"\n    colour: \"1d76db\"\n    labels:\n      - name: bug\n        description: One\n",
			"field colour not found",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "labels.yml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadLabels(path)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error was %q, want it to mention %q", err, tc.want)
			}
		})
	}
}
