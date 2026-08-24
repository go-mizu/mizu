package xs_test

import (
	"errors"
	"iter"
	"slices"
	"strconv"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

// Two things are worth checking about every function here. What comes out, and
// how much of the input was read to produce it, since stopping early is most of
// why this package exists. counted returns a sequence and a pointer to the
// number of elements that have been pulled out of it.
func counted[T any](s []T) (iter.Seq[T], *int) {
	var n int
	return func(yield func(T) bool) {
		for _, v := range s {
			n++
			if !yield(v) {
				return
			}
		}
	}, &n
}

// drain reads a sequence to the end and throws the elements away, for the tests
// that are about how much was read rather than about what came out.
func drain[T any](seq iter.Seq[T]) {
	for range seq {
	}
}

func TestMap(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})
	got := slices.Collect(xs.Map(in, strconv.Itoa))

	if want := []string{"1", "2", "3"}; !slices.Equal(got, want) {
		t.Errorf("Map gave %q, want %q", got, want)
	}
}

func TestMapRunsNothingUntilItIsRead(t *testing.T) {
	calls := 0
	seq := xs.Map(slices.Values([]int{1, 2, 3}), func(n int) int {
		calls++
		return n
	})

	if calls != 0 {
		t.Fatalf("fn ran %d times before anything read the sequence", calls)
	}
	drain(seq)
	if calls != 3 {
		t.Errorf("fn ran %d times over three elements, want 3", calls)
	}
}

func TestMapStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5})

	for range xs.Map(in, strconv.Itoa) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements to yield one, want 1", *read)
	}
}

func TestMapOverAnEmptySequence(t *testing.T) {
	got := slices.Collect(xs.Map(slices.Values([]int(nil)), strconv.Itoa))
	if len(got) != 0 {
		t.Errorf("Map over nothing gave %q, want nothing", got)
	}
}

// TestMapErrPassesErrorsPast is the reason this variant exists. An element that
// arrived broken keeps its error and fn is not asked to deal with it.
func TestMapErrPassesErrorsPast(t *testing.T) {
	bad := errors.New("row 2 is unreadable")
	in := func(yield func(int, error) bool) {
		yield(1, nil)
		yield(0, bad)
		yield(3, nil)
	}

	var seen []int
	got := xs.MapErr(in, func(n int) (string, error) {
		seen = append(seen, n)
		return strconv.Itoa(n), nil
	})

	var values []string
	var errs []error
	for v, err := range got {
		values = append(values, v)
		errs = append(errs, err)
	}

	if want := []int{1, 3}; !slices.Equal(seen, want) {
		t.Errorf("fn saw %v, want %v: the broken element is not fn's problem", seen, want)
	}
	if want := []string{"1", "", "3"}; !slices.Equal(values, want) {
		t.Errorf("MapErr gave %q, want %q", values, want)
	}
	if !errors.Is(errs[1], bad) {
		t.Errorf("the error arrived as %v, want %v", errs[1], bad)
	}
	if errs[0] != nil || errs[2] != nil {
		t.Errorf("the good elements came with %v and %v, want no error", errs[0], errs[2])
	}
}

func TestMapErrCarriesAnErrorFromFn(t *testing.T) {
	bad := errors.New("cannot convert")
	in := func(yield func(int, error) bool) {
		yield(1, nil)
		yield(2, nil)
	}

	var errs []error
	for _, err := range xs.MapErr(in, func(n int) (string, error) {
		if n == 2 {
			return "", bad
		}
		return strconv.Itoa(n), nil
	}) {
		errs = append(errs, err)
	}

	if len(errs) != 2 {
		t.Fatalf("it yielded %d elements, want 2: an error does not end the sequence", len(errs))
	}
	if !errors.Is(errs[1], bad) {
		t.Errorf("the second element came with %v, want %v", errs[1], bad)
	}
}

func TestMapErrStopsWhenTheCallerDoes(t *testing.T) {
	bad := errors.New("no")
	read := 0
	in := func(yield func(int, error) bool) {
		for _, v := range []int{1, 2, 3} {
			read++
			if !yield(v, nil) {
				return
			}
		}
	}

	// Breaking on the first good element and on the first bad one are separate
	// paths through MapErr, so both get a turn.
	for range xs.MapErr(in, func(n int) (int, error) { return n, nil }) {
		break
	}
	if read != 1 {
		t.Errorf("it read %d elements before the break, want 1", read)
	}

	broken := func(yield func(int, error) bool) {
		read++
		if !yield(0, bad) {
			return
		}
		read++
		yield(0, bad)
	}
	read = 0
	for range xs.MapErr(broken, func(n int) (int, error) { return n, nil }) {
		break
	}
	if read != 1 {
		t.Errorf("it read %d broken elements before the break, want 1", read)
	}
}

func TestFilter(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5, 6})
	got := slices.Collect(xs.Filter(in, func(n int) bool { return n%2 == 0 }))

	if want := []int{2, 4, 6}; !slices.Equal(got, want) {
		t.Errorf("Filter gave %v, want %v", got, want)
	}
}

func TestFilterStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5, 6})

	for range xs.Filter(in, func(n int) bool { return n%2 == 0 }) {
		break
	}
	// Two elements to find one even number, and nothing after it.
	if *read != 2 {
		t.Errorf("it read %d elements to yield one, want 2", *read)
	}
}

func TestFilterKeepingNothing(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})
	got := slices.Collect(xs.Filter(in, func(int) bool { return false }))

	if len(got) != 0 {
		t.Errorf("a filter that keeps nothing gave %v", got)
	}
}

func TestReject(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5, 6})
	got := slices.Collect(xs.Reject(in, func(n int) bool { return n%2 == 0 }))

	if want := []int{1, 3, 5}; !slices.Equal(got, want) {
		t.Errorf("Reject gave %v, want %v", got, want)
	}
}

func TestRejectStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{2, 4, 5, 6})

	for range xs.Reject(in, func(n int) bool { return n%2 == 0 }) {
		break
	}
	if *read != 3 {
		t.Errorf("it read %d elements to yield one, want 3", *read)
	}
}

func TestTap(t *testing.T) {
	var seen []int
	in := slices.Values([]int{1, 2, 3})
	got := slices.Collect(xs.Tap(in, func(n int) { seen = append(seen, n) }))

	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Tap changed the sequence to %v, want %v", got, want)
	}
	if !slices.Equal(seen, got) {
		t.Errorf("fn saw %v, want %v", seen, got)
	}
}

// TestTapOnlySeesWhatIsRead pins down that a Tap in front of a Take of two sees
// two, which is the answer people expect to be wrong.
func TestTapOnlySeesWhatIsRead(t *testing.T) {
	var seen []int
	in := slices.Values([]int{1, 2, 3, 4, 5})
	drain(xs.Take(xs.Tap(in, func(n int) { seen = append(seen, n) }), 2))

	if want := []int{1, 2}; !slices.Equal(seen, want) {
		t.Errorf("fn saw %v, want %v", seen, want)
	}
}

func TestTapStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3})

	for range xs.Tap(in, func(int) {}) {
		break
	}
	if *read != 1 {
		t.Errorf("it read %d elements to yield one, want 1", *read)
	}
}
