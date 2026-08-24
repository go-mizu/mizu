package main

import (
	"strings"
	"testing"
)

const trackingBody = `## Goal

Foundations.

## Checklist

- [x] ` + "`M0-01`" + ` Repository, module split, CI matrix
- [ ] ` + "`M0-02`" + ` Codegen harness
- [ ] ` + "`M0-03`" + ` Config: TOML loading, layering, env expansion

## Acceptance criteria

- [ ] mizu new hello works on Linux and macOS
`

func TestParseChecklistIgnoresPlainCheckboxes(t *testing.T) {
	items := ParseChecklist(trackingBody)
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3: %+v", len(items), items)
	}
	// The acceptance criterion is a checkbox with no identifier, and a human
	// ticks it. If it ever shows up here, the merge automation would start
	// claiming criteria are met because a pull request merged.
	for _, it := range items {
		if !strings.HasPrefix(it.ID, "M0-") {
			t.Errorf("unexpected item %q", it.ID)
		}
	}
	if got, want := items[2].Text, "Config: TOML loading, layering, env expansion"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if !items[0].Done || items[1].Done {
		t.Errorf("done flags wrong: %+v", items)
	}
	if got, want := items[1].Milestone(), "M0"; got != want {
		t.Errorf("milestone = %q, want %q", got, want)
	}
}

func TestTick(t *testing.T) {
	body, item, changed, err := Tick(trackingBody, "M0-02")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if item.Text != "Codegen harness" {
		t.Errorf("item text = %q", item.Text)
	}
	if !strings.Contains(body, "- [x] `M0-02` Codegen harness") {
		t.Errorf("box not ticked:\n%s", body)
	}
	// Everything else survives untouched, including the criterion below.
	if strings.Count(body, "- [ ] ") != 2 {
		t.Errorf("wrong number of unticked boxes left:\n%s", body)
	}
	if done, total := Progress(ParseChecklist(body)); done != 2 || total != 3 {
		t.Errorf("progress = %d/%d, want 2/3", done, total)
	}
}

func TestTickIsIdempotent(t *testing.T) {
	// Re-running a workflow must not comment twice, so ticking a box that is
	// already ticked reports no change rather than an error.
	body, item, changed, err := Tick(trackingBody, "M0-01")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("changed = true for an already ticked item")
	}
	if body != trackingBody {
		t.Error("body was rewritten for an already ticked item")
	}
	if item.ID != "M0-01" {
		t.Errorf("item = %+v", item)
	}
}

func TestTickUnknownIDIsAnError(t *testing.T) {
	if _, _, _, err := Tick(trackingBody, "M0-99"); err == nil {
		t.Fatal("want an error for an identifier that does not exist")
	}
}

func TestChecklistIDs(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want []string
	}{
		{"one", "Adds the harness.\n\nChecklist: M0-02\n", []string{"M0-02"}},
		{"several", "Checklist: M0-02, M0-03\n", []string{"M0-02", "M0-03"}},
		{"backticked", "Checklist: `M1-07`\n", []string{"M1-07"}},
		{"none keyword", "Checklist: none\n", nil},
		{"absent", "Just a fix.\n", nil},
		{"deduplicated", "Checklist: M0-02\nChecklist: m0-02\n", []string{"M0-02"}},
		{
			"template comment is not a claim",
			"<!-- Checklist: M0-03 (delete this if the change finishes no item) -->\nA fix.\n",
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ChecklistIDs(tc.body)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	for _, tc := range []struct {
		done, total int
		want        string
	}{
		{0, 3, "`....................`"},
		{3, 3, "`####################`"},
		{1, 4, "`#####...............`"},
		{0, 0, ""},
	} {
		if got := progressBar(tc.done, tc.total); got != tc.want {
			t.Errorf("progressBar(%d, %d) = %q, want %q", tc.done, tc.total, got, tc.want)
		}
	}
}
