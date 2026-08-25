package main

import (
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/bench/budget"
)

func TestTablePrintsEveryRowOnce(t *testing.T) {
	got := table()

	for _, r := range budget.Rows() {
		if n := strings.Count(got, "| `"+r.ID+"` |"); n != 1 {
			t.Errorf("%s appears %d times in the table, want once", r.ID, n)
		}
	}
}

func TestTablePrintsEveryGroupAsAHeading(t *testing.T) {
	got := table()

	for _, g := range budget.Groups() {
		if n := strings.Count(got, "### "+g+"\n"); n != 1 {
			t.Errorf("%q appears as a heading %d times, want once", g, n)
		}
	}
}

// TestTableIsMarkdownAllTheWayDown is the cheap way to catch a row whose
// operation text has a pipe in it, which would silently split the row into two
// columns wherever it landed.
func TestTableIsMarkdownAllTheWayDown(t *testing.T) {
	for line := range strings.Lines(table()) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if n := strings.Count(line, "|"); n != 7 {
			t.Errorf("a row has %d bars, want 7:\n%s", n, line)
		}
	}
}

func TestTableSaysWhichMilestoneBringsARow(t *testing.T) {
	got := table()

	if !strings.Contains(got, "| `router/match` |") || !strings.Contains(got, "| 07 | M1 |") {
		t.Errorf("the table does not say router/match arrives with M1:\n%s", got)
	}
	if !strings.Contains(got, "| 06 | measured |") {
		t.Errorf("the table does not say which rows are measured now:\n%s", got)
	}
}

func TestDuration(t *testing.T) {
	tests := map[time.Duration]string{
		0:                            "0 ns",
		5 * time.Nanosecond:          "5 ns",
		300 * time.Nanosecond:        "300 ns",
		2500 * time.Nanosecond:       "2.5 us",
		1 * time.Microsecond:         "1 us",
		36 * time.Microsecond:        "36 us",
		1100 * time.Microsecond:      "1.1 ms",
		40 * time.Millisecond:        "40 ms",
		250 * time.Millisecond:       "250 ms",
		2 * time.Second:              "2 s",
		1500 * time.Millisecond:      "1.5 s",
		budget.NoBudget:              "n/a",
		time.Duration(1234567890123): "1234.6 s",
	}
	for in, want := range tests {
		t.Run(want, func(t *testing.T) {
			if got := duration(in); got != want {
				t.Errorf("duration(%d) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestAllocs(t *testing.T) {
	tests := map[int]string{0: "0", 3: "3", 2000: "2000", budget.NoBudget: "n/a"}
	for in, want := range tests {
		if got := allocs(in); got != want {
			t.Errorf("allocs(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestArrives(t *testing.T) {
	if got := arrives(""); got != "measured" {
		t.Errorf("arrives(\"\") = %q, want measured", got)
	}
	if got := arrives("M4"); got != "M4" {
		t.Errorf("arrives(\"M4\") = %q, want M4", got)
	}
}
