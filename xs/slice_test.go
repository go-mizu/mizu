package xs_test

import (
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

// The three functions that pick at random are tested for the properties that
// have to hold every time, and then once for the shuffling actually happening,
// which is a property that only holds over enough tries.

func TestShuffleKeepsEverything(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6, 7, 8}
	got := slices.Clone(in)

	xs.Shuffle(got)
	slices.Sort(got)
	if !slices.Equal(got, in) {
		t.Errorf("Shuffle gave back %v when sorted, want %v", got, in)
	}
}

func TestShuffleMovesThings(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6, 7, 8}

	for range 50 {
		got := slices.Clone(in)
		xs.Shuffle(got)
		if !slices.Equal(got, in) {
			return
		}
	}
	t.Error("fifty shuffles of eight elements all came back in order")
}

func TestShuffleOfNothing(t *testing.T) {
	var empty []int
	xs.Shuffle(empty)
	xs.Shuffle([]int{1})
}

func TestSample(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}

	got := xs.Sample(in, 3)
	if len(got) != 3 {
		t.Fatalf("Sample gave %d elements, want 3", len(got))
	}
	for _, v := range got {
		if !slices.Contains(in, v) {
			t.Errorf("Sample gave %d, which is not in the input", v)
		}
	}
}

// TestSampleDoesNotRepeatAPosition is the difference between sampling and
// picking at random n times over.
func TestSampleDoesNotRepeatAPosition(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6}

	for range 50 {
		got := xs.Sample(in, 4)
		sorted := slices.Clone(got)
		slices.Sort(sorted)
		if len(slices.Compact(sorted)) != 4 {
			t.Fatalf("Sample gave %v, which has a repeat in it", got)
		}
	}
}

func TestSampleOfMoreThanThereIs(t *testing.T) {
	in := []int{1, 2, 3}

	got := xs.Sample(in, 10)
	slices.Sort(got)
	if !slices.Equal(got, in) {
		t.Errorf("Sample of ten out of three gave %v, want all three", got)
	}
}

func TestSampleOfNoneOrOfNothing(t *testing.T) {
	if got := xs.Sample([]int{1, 2, 3}, 0); got != nil {
		t.Errorf("Sample of none gave %v, want nil", got)
	}
	if got := xs.Sample([]int{1, 2, 3}, -1); got != nil {
		t.Errorf("Sample of a negative number gave %v, want nil", got)
	}
	if got := xs.Sample([]int(nil), 3); got != nil {
		t.Errorf("Sample of nothing gave %v, want nil", got)
	}
}

func TestSampleLeavesTheInputAlone(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	before := slices.Clone(in)

	xs.Sample(in, 3)
	if !slices.Equal(in, before) {
		t.Errorf("Sample reordered the input to %v, want %v untouched", in, before)
	}
}

func TestRandom(t *testing.T) {
	in := []int{1, 2, 3}

	got, ok := xs.Random(in)
	if !ok {
		t.Fatal("Random said there was nothing in a slice of three")
	}
	if !slices.Contains(in, got) {
		t.Errorf("Random gave %d, which is not in the input", got)
	}
}

func TestRandomOfNothing(t *testing.T) {
	got, ok := xs.Random([]string(nil))
	if ok {
		t.Error("Random of nothing said there was an element")
	}
	if got != "" {
		t.Errorf("Random of nothing gave %q, want the zero value", got)
	}
}

func TestRandomDoesNotAlwaysPickTheSameOne(t *testing.T) {
	in := []int{1, 2, 3, 4, 5, 6, 7, 8}
	first, _ := xs.Random(in)

	for range 50 {
		if got, _ := xs.Random(in); got != first {
			return
		}
	}
	t.Error("fifty picks out of eight elements all gave the same one")
}

func TestPad(t *testing.T) {
	got := xs.Pad([]string{"a", "b"}, 4, "")
	if want := []string{"a", "b", "", ""}; !slices.Equal(got, want) {
		t.Errorf("Pad gave %q, want %q", got, want)
	}
}

func TestPadLeavesALongEnoughSliceAlone(t *testing.T) {
	in := []int{1, 2, 3}

	got := xs.Pad(in, 3, 0)
	if !slices.Equal(got, in) {
		t.Errorf("Pad gave %v, want %v", got, in)
	}
	if got := xs.Pad(in, 1, 0); !slices.Equal(got, in) {
		t.Errorf("Pad to a shorter length gave %v, want the input back", got)
	}
}

// TestPadDoesNotWriteIntoTheInput is why Pad copies rather than appending, and
// the spare capacity is what would make an append visible to whoever else is
// holding a piece of that array.
func TestPadDoesNotWriteIntoTheInput(t *testing.T) {
	backing := make([]int, 2, 8)
	backing[0], backing[1] = 1, 2
	neighbour := backing[:8][2:]

	xs.Pad(backing, 5, 99)
	if neighbour[0] != 0 {
		t.Errorf("Pad wrote %d past the end of the input, want it untouched", neighbour[0])
	}
}

func TestPadOfNothing(t *testing.T) {
	got := xs.Pad([]int(nil), 2, 7)
	if want := []int{7, 7}; !slices.Equal(got, want) {
		t.Errorf("Pad of nothing gave %v, want %v", got, want)
	}
}

func TestDiff(t *testing.T) {
	got := xs.Diff([]string{"a", "b", "c"}, []string{"b"})
	if want := []string{"a", "c"}; !slices.Equal(got, want) {
		t.Errorf("Diff gave %q, want %q", got, want)
	}
}

func TestDiffKeepsTheOrderOfTheFirstSlice(t *testing.T) {
	got := xs.Diff([]int{5, 1, 4, 2}, []int{4})
	if want := []int{5, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("Diff gave %v, want %v in the order they were in", got, want)
	}
}

func TestDiffOfEmptySlices(t *testing.T) {
	if got := xs.Diff([]int(nil), []int{1}); got != nil {
		t.Errorf("Diff of nothing gave %v, want nil", got)
	}
	if got := xs.Diff([]int{1, 2}, nil); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Diff against nothing gave %v, want both elements", got)
	}
	if got := xs.Diff([]int{1}, []int{1}); got != nil {
		t.Errorf("Diff of the same thing gave %v, want nil", got)
	}
}

func TestIntersect(t *testing.T) {
	got := xs.Intersect([]string{"a", "b", "c"}, []string{"c", "a"})
	if want := []string{"a", "c"}; !slices.Equal(got, want) {
		t.Errorf("Intersect gave %q, want %q in the order of the first slice", got, want)
	}
}

func TestIntersectWithNothingInCommon(t *testing.T) {
	if got := xs.Intersect([]int{1, 2}, []int{3, 4}); got != nil {
		t.Errorf("Intersect gave %v, want nil", got)
	}
}

func TestUnion(t *testing.T) {
	got := xs.Union([]string{"a", "b"}, []string{"b", "c"})
	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("Union gave %q, want %q", got, want)
	}
}

func TestUnionOfEmptySlices(t *testing.T) {
	if got := xs.Union([]int(nil), nil); got != nil {
		t.Errorf("Union of nothing gave %v, want nil", got)
	}
	if got := xs.Union(nil, []int{1, 2}); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Union with an empty first slice gave %v, want both elements", got)
	}
}

// TestSetOperationsDropRepeats is the one promise the three of them share, and
// it is the thing to check before reaching for any of them with input that has
// repeats in it.
func TestSetOperationsDropRepeats(t *testing.T) {
	a := []int{1, 1, 2, 2, 3}
	b := []int{2, 2}

	if got := xs.Diff(a, b); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("Diff gave %v, want [1 3] with the repeats dropped", got)
	}
	if got := xs.Intersect(a, b); !slices.Equal(got, []int{2}) {
		t.Errorf("Intersect gave %v, want [2] with the repeats dropped", got)
	}
	if got := xs.Union(a, b); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Union gave %v, want [1 2 3] with the repeats dropped", got)
	}
}
