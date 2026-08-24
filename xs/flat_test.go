package xs_test

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestFlatten(t *testing.T) {
	pages := slices.Values([]iter.Seq[int]{
		slices.Values([]int{1, 2}),
		slices.Values([]int{3}),
		slices.Values([]int{4, 5}),
	})

	got := slices.Collect(xs.Flatten(pages))
	if want := []int{1, 2, 3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("Flatten gave %v, want %v", got, want)
	}
}

func TestFlattenSkipsEmptyOnes(t *testing.T) {
	empty := slices.Values([]int(nil))
	pages := slices.Values([]iter.Seq[int]{empty, slices.Values([]int{1}), empty})

	got := slices.Collect(xs.Flatten(pages))
	if want := []int{1}; !slices.Equal(got, want) {
		t.Errorf("Flatten gave %v, want %v", got, want)
	}
}

func TestFlattenOfNothing(t *testing.T) {
	if got := slices.Collect(xs.Flatten(slices.Values([]iter.Seq[int](nil)))); len(got) != 0 {
		t.Errorf("Flatten of nothing gave %v, want nothing", got)
	}
}

func TestFlattenStopsWhenTheCallerDoes(t *testing.T) {
	first, readFirst := counted([]int{1, 2})
	second, readSecond := counted([]int{3, 4})
	pages := slices.Values([]iter.Seq[int]{first, second})

	for range xs.Flatten(pages) {
		break
	}
	if *readFirst != 1 {
		t.Errorf("it read %d elements of the first sequence, want 1", *readFirst)
	}
	if *readSecond != 0 {
		t.Errorf("it read %d elements of the second sequence, want none", *readSecond)
	}
}

func TestFlatMap(t *testing.T) {
	posts := slices.Values([][]string{{"go", "http"}, {"go"}, nil})
	got := slices.Collect(xs.FlatMap(posts, slices.Values))

	if want := []string{"go", "http", "go"}; !slices.Equal(got, want) {
		t.Errorf("FlatMap gave %q, want %q", got, want)
	}
}

// TestFlatMapCanDropAnElement is the half of this that is a filter. An element
// that turns into nothing is not in the result.
func TestFlatMapCanDropAnElement(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4})
	got := slices.Collect(xs.FlatMap(in, func(n int) iter.Seq[int] {
		if n%2 == 0 {
			return slices.Values([]int(nil))
		}
		return slices.Values([]int{n, n})
	}))

	if want := []int{1, 1, 3, 3}; !slices.Equal(got, want) {
		t.Errorf("FlatMap gave %v, want %v", got, want)
	}
}

func TestFlatMapStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	calls := 0
	for range xs.FlatMap(in, func(n int) iter.Seq[int] {
		calls++
		return slices.Values([]int{n, n})
	}) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
	if calls != 1 {
		t.Errorf("fn ran %d times, want 1", calls)
	}
}
