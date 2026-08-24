package xs_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestUnique(t *testing.T) {
	in := slices.Values([]string{"go", "http", "go", "db", "http"})
	got := slices.Collect(xs.Unique(in))

	if want := []string{"go", "http", "db"}; !slices.Equal(got, want) {
		t.Errorf("Unique gave %q, want %q", got, want)
	}
}

// TestUniqueRemovesRepeatsThatAreNotNeighbours is the difference from
// slices.Compact, which only looks at the element before.
func TestUniqueRemovesRepeatsThatAreNotNeighbours(t *testing.T) {
	in := []int{1, 2, 1, 2, 1}

	got := slices.Collect(xs.Unique(slices.Values(in)))
	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("Unique gave %v, want %v", got, want)
	}

	compacted := slices.Compact(slices.Clone(in))
	if slices.Equal(compacted, got) {
		t.Error("slices.Compact gave the same answer, so this test is not testing the difference")
	}
}

func TestUniqueOfNothing(t *testing.T) {
	if got := slices.Collect(xs.Unique(slices.Values([]int(nil)))); len(got) != 0 {
		t.Errorf("Unique of nothing gave %v, want nothing", got)
	}
}

func TestUniqueStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.Unique(in) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

func TestUniqueBy(t *testing.T) {
	in := slices.Values([]string{"Ana", "BEN", "ana", "Cleo", "ben"})
	got := slices.Collect(xs.UniqueBy(in, strings.ToLower))

	if want := []string{"Ana", "BEN", "Cleo"}; !slices.Equal(got, want) {
		t.Errorf("UniqueBy gave %q, want %q: the first spelling of each is the one kept", got, want)
	}
}

func TestUniqueByOverSomethingNotComparable(t *testing.T) {
	type post struct {
		ID   int
		Tags []string // A slice, so a post cannot be a map key.
	}
	in := slices.Values([]post{
		{1, []string{"go"}},
		{2, nil},
		{1, []string{"http"}},
	})

	got := slices.Collect(xs.UniqueBy(in, func(p post) int { return p.ID }))
	if len(got) != 2 {
		t.Fatalf("UniqueBy gave %d posts, want 2", len(got))
	}
	if want := []string{"go"}; !slices.Equal(got[0].Tags, want) {
		t.Errorf("the post that was kept has tags %q, want %q from the first one", got[0].Tags, want)
	}
}

func TestUniqueByStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.UniqueBy(in, func(n int) int { return n }) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}
