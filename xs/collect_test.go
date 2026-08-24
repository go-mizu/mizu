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
