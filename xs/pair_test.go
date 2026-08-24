package xs_test

import (
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestEnumerate(t *testing.T) {
	in := slices.Values([]string{"a", "b", "c"})

	var at []int
	var got []string
	for i, v := range xs.Enumerate(in) {
		at = append(at, i)
		got = append(got, v)
	}

	if want := []int{0, 1, 2}; !slices.Equal(at, want) {
		t.Errorf("Enumerate counted %v, want %v", at, want)
	}
	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("Enumerate gave %q, want %q", got, want)
	}
}

func TestEnumerateOverNothing(t *testing.T) {
	n := 0
	for range xs.Enumerate(slices.Values([]int(nil))) {
		n++
	}
	if n != 0 {
		t.Errorf("Enumerate over nothing yielded %d elements", n)
	}
}

func TestEnumerateStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.Enumerate(in) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements before the break, want 1", *read)
	}
}

func TestZip(t *testing.T) {
	names := slices.Values([]string{"ana", "ben", "cleo"})
	scores := slices.Values([]int{1, 2, 3})

	var got []string
	for name, score := range xs.Zip(names, scores) {
		got = append(got, name)
		if score != len(got) {
			t.Errorf("%s came with %d, want %d", name, score, len(got))
		}
	}
	if want := []string{"ana", "ben", "cleo"}; !slices.Equal(got, want) {
		t.Errorf("Zip gave %q, want %q", got, want)
	}
}

// TestZipEndsWithTheShorterOne covers both sides being short, since which of
// the two runs out first is a different path through the function.
func TestZipEndsWithTheShorterOne(t *testing.T) {
	long := []int{1, 2, 3, 4}
	short := []int{1, 2}

	n := 0
	for range xs.Zip(slices.Values(long), slices.Values(short)) {
		n++
	}
	if n != 2 {
		t.Errorf("a long first side gave %d pairs, want 2", n)
	}

	n = 0
	for range xs.Zip(slices.Values(short), slices.Values(long)) {
		n++
	}
	if n != 2 {
		t.Errorf("a short first side gave %d pairs, want 2", n)
	}
}

// TestZipDoesNotReadTheLeftovers is what makes zipping against a [xs.Cycle] or
// any other endless sequence work at all.
func TestZipDoesNotReadTheLeftovers(t *testing.T) {
	short := slices.Values([]int{1, 2})
	long, read := counted([]int{1, 2, 3, 4, 5})

	for range xs.Zip(short, long) {
	}
	// Two pairs, and one more pulled to find out the first side had ended.
	if *read > 3 {
		t.Errorf("it read %d elements of the longer side, want no more than 3", *read)
	}
}

func TestZipAgainstAnEndlessSequence(t *testing.T) {
	rows := slices.Values([]string{"a", "b", "c", "d"})
	colours := xs.Cycle(slices.Values([]string{"light", "dark"}))

	var got []string
	for row, colour := range xs.Zip(rows, colours) {
		got = append(got, row+" "+colour)
	}

	want := []string{"a light", "b dark", "c light", "d dark"}
	if !slices.Equal(got, want) {
		t.Errorf("Zip gave %q, want %q", got, want)
	}
}

func TestZipStopsWhenTheCallerDoes(t *testing.T) {
	a, readA := counted([]int{1, 2, 3})
	b, readB := counted([]int{1, 2, 3})

	for range xs.Zip(a, b) {
		break
	}
	if *readA != 1 {
		t.Errorf("it read %d elements of the first side, want 1", *readA)
	}
	if *readB != 1 {
		t.Errorf("it read %d elements of the second side, want 1", *readB)
	}
}

func TestUnzip(t *testing.T) {
	in := func(yield func(string, int) bool) {
		for i, s := range []string{"a", "b", "c"} {
			if !yield(s, i) {
				return
			}
		}
	}

	letters, numbers := xs.Unzip(in)
	if want := []string{"a", "b", "c"}; !slices.Equal(letters, want) {
		t.Errorf("Unzip gave %q on the left, want %q", letters, want)
	}
	if want := []int{0, 1, 2}; !slices.Equal(numbers, want) {
		t.Errorf("Unzip gave %v on the right, want %v", numbers, want)
	}
}

func TestUnzipOfNothing(t *testing.T) {
	empty := func(yield func(int, int) bool) {}

	as, bs := xs.Unzip(empty)
	if as != nil || bs != nil {
		t.Errorf("Unzip of nothing gave %v and %v, want two nil slices", as, bs)
	}
}

// TestZipAndUnzipComeBackTheSame is the pair of them as a round trip, which is
// the reason both are here.
func TestZipAndUnzipComeBackTheSame(t *testing.T) {
	names := []string{"ana", "ben"}
	scores := []int{10, 20}

	gotNames, gotScores := xs.Unzip(xs.Zip(slices.Values(names), slices.Values(scores)))
	if !slices.Equal(gotNames, names) || !slices.Equal(gotScores, scores) {
		t.Errorf("the round trip gave %q and %v, want %q and %v", gotNames, gotScores, names, scores)
	}
}
