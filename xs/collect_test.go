package xs_test

import (
	"errors"
	"iter"
	"slices"
	"strconv"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

// withErrors turns a slice of strings into the sequence of value and error
// pairs that CollectErr reads, by parsing each one as a number.
func withErrors(in []string) iter.Seq2[int, error] {
	return xs.MapErr(errorFree(in), strconv.Atoi)
}

// errorFree pairs every element with a nil error, which is what a source that
// cannot fail looks like.
func errorFree[T any](in []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, v := range in {
			if !yield(v, nil) {
				return
			}
		}
	}
}

func TestCollectErr(t *testing.T) {
	got, err := xs.CollectErr(withErrors([]string{"1", "2", "3"}))
	if err != nil {
		t.Fatalf("CollectErr gave an error over three good numbers: %v", err)
	}
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("CollectErr gave %v, want %v", got, want)
	}
}

func TestCollectErrReturnsNothingOnAnError(t *testing.T) {
	got, err := xs.CollectErr(withErrors([]string{"1", "two", "3"}))

	if err == nil {
		t.Fatal("CollectErr found no error in a sequence with one in it")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("CollectErr gave %v, want the parse error underneath", err)
	}
	if got != nil {
		t.Errorf("CollectErr gave %v alongside the error, want nothing", got)
	}
}

func TestCollectErrStopsAtTheFirstError(t *testing.T) {
	read := 0
	in := func(yield func(int, error) bool) {
		for i, err := range []error{nil, errors.New("no"), nil} {
			read++
			if !yield(i, err) {
				return
			}
		}
	}

	if _, err := xs.CollectErr(in); err == nil {
		t.Fatal("CollectErr found no error")
	}
	if read != 2 {
		t.Errorf("it read %d elements before stopping, want 2", read)
	}
}

func TestCollectErrOfNothing(t *testing.T) {
	got, err := xs.CollectErr(withErrors(nil))
	if err != nil {
		t.Fatalf("CollectErr of nothing gave an error: %v", err)
	}
	if got != nil {
		t.Errorf("CollectErr of nothing gave %v, want a nil slice", got)
	}
}

func TestGroupBy(t *testing.T) {
	in := slices.Values([]string{"go", "rust", "gc", "ruby", "git"})

	got := xs.GroupBy(in, func(s string) string { return s[:1] })
	if len(got) != 2 {
		t.Fatalf("GroupBy made %d groups, want 2", len(got))
	}
	if want := []string{"go", "gc", "git"}; !slices.Equal(got["g"], want) {
		t.Errorf("the g group is %q, want %q in the order they arrived", got["g"], want)
	}
	if want := []string{"rust", "ruby"}; !slices.Equal(got["r"], want) {
		t.Errorf("the r group is %q, want %q", got["r"], want)
	}
}

// TestGroupByHasNoEmptyGroups is the promise that a key nobody used is missing
// rather than present and empty, so reading it gives a nil slice.
func TestGroupByHasNoEmptyGroups(t *testing.T) {
	got := xs.GroupBy(slices.Values([]int{1, 2}), func(n int) string { return "n" })

	if _, found := got["nothing"]; found {
		t.Error("GroupBy made a group for a key nothing landed in")
	}
	if len(got["nothing"]) != 0 {
		t.Error("reading a missing key gave something")
	}
}

func TestGroupByOfNothingIsAnEmptyMap(t *testing.T) {
	got := xs.GroupBy(slices.Values([]int(nil)), func(n int) int { return n })
	if got == nil {
		t.Error("GroupBy of nothing gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("GroupBy of nothing gave %v, want nothing", got)
	}
}

func TestKeyBy(t *testing.T) {
	in := slices.Values([]user{{"ana", true}, {"ben", false}})

	got := xs.KeyBy(in, func(u user) string { return u.Name })
	if len(got) != 2 {
		t.Fatalf("KeyBy made %d entries, want 2", len(got))
	}
	if got["ben"].Active {
		t.Error("the entry under ben is not ben")
	}
}

// TestKeyByKeepsTheLastOfADuplicate is the difference from UniqueBy, which
// keeps the first, and it is the behaviour of the loop KeyBy replaces.
func TestKeyByKeepsTheLastOfADuplicate(t *testing.T) {
	in := []user{{"ana", false}, {"ana", true}}
	name := func(u user) string { return u.Name }

	if got := xs.KeyBy(slices.Values(in), name); !got["ana"].Active {
		t.Error("KeyBy kept the first ana, want the last")
	}

	kept := slices.Collect(xs.UniqueBy(slices.Values(in), name))
	if len(kept) != 1 || kept[0].Active {
		t.Error("UniqueBy kept the last ana, so this test is not testing the difference")
	}
}

func TestPartitionBy(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5})

	even, odd := xs.PartitionBy(in, func(n int) bool { return n%2 == 0 })
	if want := []int{2, 4}; !slices.Equal(even, want) {
		t.Errorf("the accepted side is %v, want %v", even, want)
	}
	if want := []int{1, 3, 5}; !slices.Equal(odd, want) {
		t.Errorf("the rejected side is %v, want %v", odd, want)
	}
}

func TestPartitionByWithEverythingOnOneSide(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})

	yes, no := xs.PartitionBy(in, func(n int) bool { return true })
	if len(yes) != 3 {
		t.Errorf("the accepted side has %d, want 3", len(yes))
	}
	if no != nil {
		t.Errorf("the rejected side is %v, want nil", no)
	}
}

func TestPartitionByOfNothing(t *testing.T) {
	yes, no := xs.PartitionBy(slices.Values([]int(nil)), func(n int) bool { return true })
	if yes != nil || no != nil {
		t.Errorf("PartitionBy of nothing gave %v and %v, want two nil slices", yes, no)
	}
}

func TestJoin(t *testing.T) {
	in := slices.Values([]string{"a", "b", "c"})

	if got := xs.Join(in, ", "); got != "a, b, c" {
		t.Errorf("Join gave %q, want %q", got, "a, b, c")
	}
}

func TestJoinOfOneAndOfNothing(t *testing.T) {
	if got := xs.Join(slices.Values([]string{"only"}), ", "); got != "only" {
		t.Errorf("Join of one gave %q, want %q with no separator", got, "only")
	}
	if got := xs.Join(slices.Values([]string(nil)), ", "); got != "" {
		t.Errorf("Join of nothing gave %q, want the empty string", got)
	}
}

// TestJoinKeepsEmptyElements is worth pinning, since an empty string between
// two separators is a real value and not a missing one.
func TestJoinKeepsEmptyElements(t *testing.T) {
	in := slices.Values([]string{"a", "", "c"})

	if got := xs.Join(in, ","); got != "a,,c" {
		t.Errorf("Join gave %q, want %q", got, "a,,c")
	}
}

func TestEachErr(t *testing.T) {
	var got []int

	err := xs.EachErr(errorFree([]int{1, 2, 3}), func(n int) error {
		got = append(got, n)
		return nil
	})
	if err != nil {
		t.Fatalf("EachErr gave an error over three good elements: %v", err)
	}
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("it saw %v, want %v", got, want)
	}
}

func TestEachErrStopsAtAnErrorFromTheSequence(t *testing.T) {
	boom := errors.New("the sequence gave up")
	in := func(yield func(int, error) bool) {
		if !yield(1, nil) {
			return
		}
		if !yield(0, boom) {
			return
		}
		yield(3, nil)
	}

	seen := 0
	err := xs.EachErr(in, func(n int) error {
		seen++
		return nil
	})
	if !errors.Is(err, boom) {
		t.Errorf("EachErr gave %v, want the error from the sequence", err)
	}
	if seen != 1 {
		t.Errorf("fn ran %d times, want 1 before the error", seen)
	}
}

func TestEachErrStopsAtAnErrorFromFn(t *testing.T) {
	boom := errors.New("fn gave up")

	seen := 0
	err := xs.EachErr(errorFree([]int{1, 2, 3}), func(n int) error {
		seen++
		if n == 2 {
			return boom
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Errorf("EachErr gave %v, want the error from fn", err)
	}
	if seen != 2 {
		t.Errorf("fn ran %d times, want 2", seen)
	}
}

func TestEachErrOfNothing(t *testing.T) {
	err := xs.EachErr(errorFree([]int(nil)), func(n int) error { return errors.New("no") })
	if err != nil {
		t.Errorf("EachErr over nothing gave %v, want nil", err)
	}
}
