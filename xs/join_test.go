package xs_test

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestConcat(t *testing.T) {
	a := slices.Values([]int{1, 2})
	b := slices.Values([]int{3})
	c := slices.Values([]int{4, 5})

	got := slices.Collect(xs.Concat(a, b, c))
	if want := []int{1, 2, 3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("Concat gave %v, want %v", got, want)
	}
}

func TestConcatWithNothing(t *testing.T) {
	if got := slices.Collect(xs.Concat[int]()); len(got) != 0 {
		t.Errorf("Concat of nothing gave %v, want nothing", got)
	}
}

func TestConcatSkipsEmptyOnes(t *testing.T) {
	empty := slices.Values([]int(nil))
	got := slices.Collect(xs.Concat(empty, slices.Values([]int{1}), empty))

	if want := []int{1}; !slices.Equal(got, want) {
		t.Errorf("Concat gave %v, want %v", got, want)
	}
}

// TestConcatDoesNotReadWhatItDoesNotReach is what makes Concat safe in front of
// something expensive. The second sequence is never touched.
func TestConcatDoesNotReadWhatItDoesNotReach(t *testing.T) {
	first, readFirst := counted([]int{1, 2, 3})
	second, readSecond := counted([]int{4, 5})

	for range xs.Concat(first, second) {
		break
	}
	if *readFirst != 1 {
		t.Errorf("it read %d elements of the first sequence, want 1", *readFirst)
	}
	if *readSecond != 0 {
		t.Errorf("it read %d elements of the second sequence, want none", *readSecond)
	}
}

func TestRepeat(t *testing.T) {
	in := slices.Values([]int{1, 2})
	got := slices.Collect(xs.Repeat(in, 3))

	if want := []int{1, 2, 1, 2, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("Repeat gave %v, want %v", got, want)
	}
}

func TestRepeatNoTimes(t *testing.T) {
	for _, n := range []int{0, -1} {
		in, read := counted([]int{1, 2})
		if got := slices.Collect(xs.Repeat(in, n)); len(got) != 0 {
			t.Errorf("Repeat of %d gave %v, want nothing", n, got)
		}
		if *read != 0 {
			t.Errorf("Repeat of %d read %d elements, want none", n, *read)
		}
	}
}

// TestRepeatOfNothingGivesUp covers the pass that yields nothing. Without the
// check it would make the same empty pass n times, which for a large n is a
// hang with extra steps.
func TestRepeatOfNothingGivesUp(t *testing.T) {
	passes := 0
	empty := func(yield func(int) bool) { passes++ }

	if got := slices.Collect(xs.Repeat(empty, 1000)); len(got) != 0 {
		t.Errorf("Repeat of an empty sequence gave %v, want nothing", got)
	}
	if passes != 1 {
		t.Errorf("it made %d passes over an empty sequence, want 1", passes)
	}
}

func TestRepeatStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2})

	var got []int
	for v := range xs.Repeat(in, 100) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}
	if want := []int{1, 2, 1}; !slices.Equal(got, want) {
		t.Errorf("it gave %v, want %v", got, want)
	}
	if *read != 3 {
		t.Errorf("it read %d elements, want 3", *read)
	}
}

func TestCycle(t *testing.T) {
	in := slices.Values([]string{"red", "green"})
	got := slices.Collect(xs.Take(xs.Cycle(in), 5))

	if want := []string{"red", "green", "red", "green", "red"}; !slices.Equal(got, want) {
		t.Errorf("Cycle gave %q, want %q", got, want)
	}
}

// TestCycleOfNothingGivesUp is the one that would hang. A Cycle with nothing to
// yield has no downstream Take to end it, because a Take never sees an element.
func TestCycleOfNothingGivesUp(t *testing.T) {
	passes := 0
	empty := func(yield func(int) bool) { passes++ }

	if got := slices.Collect(xs.Take(xs.Cycle(empty), 10)); len(got) != 0 {
		t.Errorf("Cycle of an empty sequence gave %v, want nothing", got)
	}
	if passes != 1 {
		t.Errorf("it made %d passes over an empty sequence, want 1", passes)
	}
}

func TestCycleStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2})

	for range xs.Cycle(in) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

// TestCycleReadsTheInputEveryTimeRound says out loud what the doc comment warns
// about. A sequence that cannot be read twice cannot be cycled, and this is
// what "read twice" means.
func TestCycleReadsTheInputEveryTimeRound(t *testing.T) {
	starts := 0
	in := func(yield func(int) bool) {
		starts++
		for _, v := range []int{1, 2} {
			if !yield(v) {
				return
			}
		}
	}

	var seq iter.Seq[int] = xs.Cycle(in)
	drain(xs.Take(seq, 5))

	if starts != 3 {
		t.Errorf("it read the input %d times to produce five elements, want 3", starts)
	}
}

func TestInterleave(t *testing.T) {
	a := slices.Values([]int{1, 4, 7})
	b := slices.Values([]int{2, 5, 8})
	c := slices.Values([]int{3, 6, 9})

	got := slices.Collect(xs.Interleave(a, b, c))
	if want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}; !slices.Equal(got, want) {
		t.Errorf("Interleave gave %v, want %v", got, want)
	}
}

// TestInterleaveCarriesOnWithoutTheShortOnes is the difference from Zip. A
// sequence that runs out drops out and the rest keep going.
func TestInterleaveCarriesOnWithoutTheShortOnes(t *testing.T) {
	a := slices.Values([]int{1})
	b := slices.Values([]int{2, 20, 200})
	c := slices.Values([]int{3, 30})

	got := slices.Collect(xs.Interleave(a, b, c))
	if want := []int{1, 2, 3, 20, 30, 200}; !slices.Equal(got, want) {
		t.Errorf("Interleave gave %v, want %v", got, want)
	}
}

func TestInterleaveWithNothing(t *testing.T) {
	if got := slices.Collect(xs.Interleave[int]()); len(got) != 0 {
		t.Errorf("Interleave of nothing gave %v, want nothing", got)
	}
}

func TestInterleaveWithOne(t *testing.T) {
	got := slices.Collect(xs.Interleave(slices.Values([]int{1, 2, 3})))
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Interleave of one sequence gave %v, want %v", got, want)
	}
}

func TestInterleaveWhenEveryOneIsEmpty(t *testing.T) {
	empty := slices.Values([]int(nil))
	if got := slices.Collect(xs.Interleave(empty, empty)); len(got) != 0 {
		t.Errorf("Interleave gave %v, want nothing", got)
	}
}

func TestInterleaveStopsWhenTheCallerDoes(t *testing.T) {
	a, readA := counted([]int{1, 2, 3})
	b, readB := counted([]int{4, 5, 6})

	for range xs.Interleave(a, b) {
		break
	}
	if *readA != 1 {
		t.Errorf("it read %d elements of the first sequence, want 1", *readA)
	}
	if *readB != 0 {
		t.Errorf("it read %d elements of the second sequence, want none", *readB)
	}
}
