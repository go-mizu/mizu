package budget

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// idShape is what a budget ID looks like: lowercase words joined by slashes.
// It is also a benchmark name and a command line argument, so anything that
// needs quoting or that a regexp would read as syntax is out.
var idShape = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)*$`)

func TestEveryRowIsFilledIn(t *testing.T) {
	for _, r := range Rows() {
		if !idShape.MatchString(r.ID) {
			t.Errorf("%q is not a usable ID, want lowercase words joined by slashes", r.ID)
		}
		if r.Op == "" {
			t.Errorf("%s says nothing about what it measures", r.ID)
		}
		if r.Group == "" {
			t.Errorf("%s is in no group, so the table has nowhere to print it", r.ID)
		}
		if r.Doc == "" {
			t.Errorf("%s names no design document, so there is nothing to argue with when it moves", r.ID)
		}
		if r.Time < 0 && r.Time != NoBudget {
			t.Errorf("%s has a negative time budget of %v", r.ID, r.Time)
		}
		if r.Allocs < 0 && r.Allocs != NoBudget {
			t.Errorf("%s has a negative allocation budget of %d", r.ID, r.Allocs)
		}
		if r.Time == NoBudget && r.Allocs == NoBudget {
			t.Errorf("%s has no budget at all, which makes it a row about nothing", r.ID)
		}
	}
}

// TestIDsAreUnique is the one that has to hold for the rest of the tooling to
// work. Two rows with one ID means two benchmarks reporting under one name, and
// a run where the second silently replaces the first.
func TestIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rows() {
		if seen[r.ID] {
			t.Errorf("%s appears twice", r.ID)
		}
		seen[r.ID] = true
	}
}

// TestNoIDIsInsideAnother is about how go test reads a -bench argument. The
// pattern is matched against each slash-separated part, so an ID that is a
// prefix of another one selects both and a run of one operation quietly
// measures two.
func TestNoIDIsInsideAnother(t *testing.T) {
	rows := Rows()
	for _, a := range rows {
		for _, b := range rows {
			if a.ID != b.ID && strings.HasPrefix(b.ID, a.ID+"/") {
				t.Errorf("%s is inside %s, so -bench=Budget/%s runs both", a.ID, b.ID, a.ID)
			}
		}
	}
}

// TestGroupsAreContiguous keeps the printed table from repeating a heading.
// Groups returns the first appearance of each name, so a row filed under a
// group that has already been printed is a section that goes missing.
func TestGroupsAreContiguous(t *testing.T) {
	var order []string
	for _, r := range Rows() {
		if len(order) == 0 || order[len(order)-1] != r.Group {
			if slices.Contains(order, r.Group) {
				t.Errorf("%s is in %q, which the table has already printed", r.ID, r.Group)
			}
			order = append(order, r.Group)
		}
	}
	if got := Groups(); !slices.Equal(got, order) {
		t.Errorf("Groups gave %v, want %v", got, order)
	}
}

// TestMilestonesLookLikeMilestones catches the typo that would make benchrun
// check pass a row nobody is going to write a benchmark for.
func TestMilestonesLookLikeMilestones(t *testing.T) {
	shape := regexp.MustCompile(`^M[0-9]$`)
	for _, r := range Rows() {
		if r.Since != "" && !shape.MatchString(r.Since) {
			t.Errorf("%s arrives with %q, which is not a milestone", r.ID, r.Since)
		}
	}
}

func TestMeasured(t *testing.T) {
	if !(Row{}).Measured() {
		t.Error("a row with no milestone against it reads as not measured")
	}
	if (Row{Since: "M1"}).Measured() {
		t.Error("a row waiting on M1 reads as measured")
	}
}

func TestLookup(t *testing.T) {
	got, ok := Lookup("log/info/json")
	if !ok {
		t.Fatal("log/info/json is not in the budget")
	}
	if got.Doc != "06" {
		t.Errorf("log/info/json comes from doc %q, want 06", got.Doc)
	}
	if _, ok := Lookup("nothing/like/this"); ok {
		t.Error("an ID nobody wrote down was found")
	}
}

// TestRowsIsACopy is what lets benchrun sort the result without reordering the
// table for the next caller.
func TestRowsIsACopy(t *testing.T) {
	rows := Rows()
	rows[0].ID = "changed"

	if got := Rows()[0].ID; got == "changed" {
		t.Error("Rows hands back the table itself, so a caller can edit it from underneath")
	}
}

// TestSomethingIsMeasured guards against the whole set quietly moving to a
// later milestone, which would leave benchrun check with nothing to compare and
// passing.
func TestSomethingIsMeasured(t *testing.T) {
	var n int
	for _, r := range Rows() {
		if r.Measured() {
			n++
		}
	}
	if n == 0 {
		t.Error("nothing in the budget is measured, so check has nothing to check")
	}
}
