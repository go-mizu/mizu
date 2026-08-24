package xs_test

import (
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestTake(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5})
	got := slices.Collect(xs.Take(in, 3))

	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Take gave %v, want %v", got, want)
	}
}

// TestTakeReadsNoFurtherThanItHasTo is the whole reason to reach for a sequence
// rather than a slice. Three out of five reads three.
func TestTakeReadsNoFurtherThanItHasTo(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5})
	drain(xs.Take(in, 3))

	if *read != 3 {
		t.Errorf("it read %d elements to take three, want 3", *read)
	}
}

func TestTakeMoreThanThereIs(t *testing.T) {
	in := slices.Values([]int{1, 2})
	got := slices.Collect(xs.Take(in, 10))

	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("Take of ten from two gave %v, want %v", got, want)
	}
}

func TestTakeNothing(t *testing.T) {
	for _, n := range []int{0, -1} {
		in, read := counted([]int{1, 2, 3})
		if got := slices.Collect(xs.Take(in, n)); len(got) != 0 {
			t.Errorf("Take of %d gave %v, want nothing", n, got)
		}
		if *read != 0 {
			t.Errorf("Take of %d read %d elements, want none", n, *read)
		}
	}
}

func TestTakeStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.Take(in, 3) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

func TestTakeWhile(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 1})
	got := slices.Collect(xs.TakeWhile(in, func(n int) bool { return n < 3 }))

	if want := []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("TakeWhile gave %v, want %v: it ends at the first failure and does not resume", got, want)
	}
}

func TestTakeWhileReadsOneMoreThanItYields(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5})
	drain(xs.TakeWhile(in, func(n int) bool { return n < 3 }))

	// Two yielded, and the third had to be read to find out it was the end.
	if *read != 3 {
		t.Errorf("it read %d elements, want 3", *read)
	}
}

func TestTakeWhileWhenTheFirstOneFails(t *testing.T) {
	in := slices.Values([]int{9, 1, 2})
	if got := slices.Collect(xs.TakeWhile(in, func(n int) bool { return n < 3 })); len(got) != 0 {
		t.Errorf("TakeWhile gave %v, want nothing", got)
	}
}

func TestTakeWhileStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.TakeWhile(in, func(int) bool { return true }) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

func TestDrop(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5})
	got := slices.Collect(xs.Drop(in, 2))

	if want := []int{3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("Drop gave %v, want %v", got, want)
	}
}

func TestDropMoreThanThereIs(t *testing.T) {
	in := slices.Values([]int{1, 2})
	if got := slices.Collect(xs.Drop(in, 10)); len(got) != 0 {
		t.Errorf("Drop of ten from two gave %v, want nothing", got)
	}
}

func TestDropNothing(t *testing.T) {
	for _, n := range []int{0, -1} {
		in := slices.Values([]int{1, 2, 3})
		got := slices.Collect(xs.Drop(in, n))
		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("Drop of %d gave %v, want %v", n, got, want)
		}
	}
}

func TestDropStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	for range xs.Drop(in, 2) {
		break
	}
	// Two skipped and one yielded, and nothing after that.
	if *read != 3 {
		t.Errorf("it read %d elements before the break, want 3", *read)
	}
}

func TestDropWhile(t *testing.T) {
	in := slices.Values([]int{1, 2, 5, 1, 6})
	got := slices.Collect(xs.DropWhile(in, func(n int) bool { return n < 3 }))

	if want := []int{5, 1, 6}; !slices.Equal(got, want) {
		t.Errorf("DropWhile gave %v, want %v: the 1 in the middle is not a leading one", got, want)
	}
}

// TestDropWhileStopsAsking pins down the difference between this and Reject.
// Once the condition has failed once it is not consulted again.
func TestDropWhileStopsAsking(t *testing.T) {
	asked := 0
	in := slices.Values([]int{1, 1, 5, 1, 1})
	drain(xs.DropWhile(in, func(n int) bool {
		asked++
		return n < 3
	}))

	if asked != 3 {
		t.Errorf("it asked %d times, want 3: two that skipped and the one that ended it", asked)
	}
}

func TestDropWhileWhenNothingIsSkipped(t *testing.T) {
	in := slices.Values([]int{5, 6})
	got := slices.Collect(xs.DropWhile(in, func(n int) bool { return n < 3 }))

	if want := []int{5, 6}; !slices.Equal(got, want) {
		t.Errorf("DropWhile gave %v, want %v", got, want)
	}
}

func TestDropWhileWhenEverythingIsSkipped(t *testing.T) {
	in := slices.Values([]int{1, 2})
	if got := slices.Collect(xs.DropWhile(in, func(n int) bool { return n < 3 })); len(got) != 0 {
		t.Errorf("DropWhile gave %v, want nothing", got)
	}
}

func TestDropWhileStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 5, 6})

	for range xs.DropWhile(in, func(n int) bool { return n < 3 }) {
		break
	}
	if *read != 2 {
		t.Errorf("it read %d elements before the break, want 2", *read)
	}
}
