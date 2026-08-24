package xs_test

import (
	"maps"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestReduce(t *testing.T) {
	in := slices.Values([]string{"go", "gopher", "http"})

	got, ok := xs.Reduce(in, func(a, b string) string {
		if len(b) > len(a) {
			return b
		}
		return a
	})
	if !ok {
		t.Fatal("Reduce said there was nothing to reduce")
	}
	if got != "gopher" {
		t.Errorf("Reduce gave %q, want %q", got, "gopher")
	}
}

func TestReduceOfOneElement(t *testing.T) {
	calls := 0
	got, ok := xs.Reduce(slices.Values([]int{7}), func(a, b int) int {
		calls++
		return a + b
	})

	if !ok || got != 7 {
		t.Errorf("Reduce gave %d and %v, want 7 and true", got, ok)
	}
	if calls != 0 {
		t.Errorf("it called the function %d times over one element, want 0", calls)
	}
}

func TestReduceOfNothing(t *testing.T) {
	got, ok := xs.Reduce(slices.Values([]int(nil)), func(a, b int) int { return a + b })
	if ok {
		t.Error("Reduce of nothing said there was an answer")
	}
	if got != 0 {
		t.Errorf("Reduce of nothing gave %d, want the zero value", got)
	}
}

func TestFold(t *testing.T) {
	in := slices.Values([]string{"a", "b", "c"})

	got := xs.Fold(in, 0, func(n int, s string) int { return n + len(s) })
	if got != 3 {
		t.Errorf("Fold gave %d, want 3", got)
	}
}

// TestFoldIntoAnotherType is the reason Fold exists next to Reduce.
func TestFoldIntoAnotherType(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})

	got := xs.Fold(in, new(strings.Builder), func(b *strings.Builder, n int) *strings.Builder {
		b.WriteString(strings.Repeat("x", n))
		return b
	})
	if want := "xxxxxx"; got.String() != want {
		t.Errorf("Fold built %q, want %q", got.String(), want)
	}
}

func TestFoldOfNothingGivesBackInit(t *testing.T) {
	got := xs.Fold(slices.Values([]int(nil)), 42, func(a, b int) int { return a + b })
	if got != 42 {
		t.Errorf("Fold of nothing gave %d, want the 42 it started with", got)
	}
}

func TestSum(t *testing.T) {
	if got := xs.Sum(slices.Values([]int{1, 2, 3, 4})); got != 10 {
		t.Errorf("Sum gave %d, want 10", got)
	}
	if got := xs.Sum(slices.Values([]float64{0.5, 0.25})); got != 0.75 {
		t.Errorf("Sum gave %v, want 0.75", got)
	}
}

func TestSumOfNothingIsZero(t *testing.T) {
	if got := xs.Sum(slices.Values([]int(nil))); got != 0 {
		t.Errorf("Sum of nothing gave %d, want 0", got)
	}
}

// TestSumOverANamedType covers the tilde in the Number constraint.
func TestSumOverANamedType(t *testing.T) {
	type cents int64

	if got := xs.Sum(slices.Values([]cents{199, 250})); got != 449 {
		t.Errorf("Sum gave %d, want 449", got)
	}
}

func TestProduct(t *testing.T) {
	if got := xs.Product(slices.Values([]int{2, 3, 4})); got != 24 {
		t.Errorf("Product gave %d, want 24", got)
	}
}

func TestProductOfNothingIsOne(t *testing.T) {
	if got := xs.Product(slices.Values([]int(nil))); got != 1 {
		t.Errorf("Product of nothing gave %d, want 1", got)
	}
}

func TestMinAndMax(t *testing.T) {
	in := []int{3, 1, 4, 1, 5}

	if got, ok := xs.Min(slices.Values(in)); !ok || got != 1 {
		t.Errorf("Min gave %d and %v, want 1 and true", got, ok)
	}
	if got, ok := xs.Max(slices.Values(in)); !ok || got != 5 {
		t.Errorf("Max gave %d and %v, want 5 and true", got, ok)
	}
}

func TestMinAndMaxOfNothing(t *testing.T) {
	if _, ok := xs.Min(slices.Values([]int(nil))); ok {
		t.Error("Min of nothing said there was an answer")
	}
	if _, ok := xs.Max(slices.Values([]int(nil))); ok {
		t.Error("Max of nothing said there was an answer")
	}
}

// TestMinAndMaxOverStrings is here because the ordering that matters most often
// is not a numeric one.
func TestMinAndMaxOverStrings(t *testing.T) {
	in := []string{"pear", "apple", "quince"}

	if got, _ := xs.Min(slices.Values(in)); got != "apple" {
		t.Errorf("Min gave %q, want %q", got, "apple")
	}
	if got, _ := xs.Max(slices.Values(in)); got != "quince" {
		t.Errorf("Max gave %q, want %q", got, "quince")
	}
}

// TestMinPropagatesNaN pins the behaviour the doc comment promises, which is the
// one the min builtin has.
func TestMinPropagatesNaN(t *testing.T) {
	in := slices.Values([]float64{1, math.NaN(), 2})

	got, ok := xs.Min(in)
	if !ok || !math.IsNaN(got) {
		t.Errorf("Min gave %v, want NaN", got)
	}
}

func TestMinByAndMaxBy(t *testing.T) {
	in := []user{{"ana", true}, {"bo", false}, {"cleopatra", true}}

	if got, ok := xs.MinBy(slices.Values(in), func(u user) int { return len(u.Name) }); !ok || got.Name != "bo" {
		t.Errorf("MinBy gave %q, want %q", got.Name, "bo")
	}
	if got, ok := xs.MaxBy(slices.Values(in), func(u user) int { return len(u.Name) }); !ok || got.Name != "cleopatra" {
		t.Errorf("MaxBy gave %q, want %q", got.Name, "cleopatra")
	}
}

// TestMinByKeepsTheFirstOfATie is the promise in the doc comment, and it is the
// same promise for MaxBy.
func TestMinByKeepsTheFirstOfATie(t *testing.T) {
	in := []user{{"ana", true}, {"ben", false}, {"cat", true}}
	length := func(u user) int { return len(u.Name) }

	if got, _ := xs.MinBy(slices.Values(in), length); got.Name != "ana" {
		t.Errorf("MinBy gave %q, want the first of the tie, %q", got.Name, "ana")
	}
	if got, _ := xs.MaxBy(slices.Values(in), length); got.Name != "ana" {
		t.Errorf("MaxBy gave %q, want the first of the tie, %q", got.Name, "ana")
	}
}

func TestMinByAndMaxByOfNothing(t *testing.T) {
	empty := slices.Values([]user(nil))
	name := func(u user) string { return u.Name }

	if _, ok := xs.MinBy(empty, name); ok {
		t.Error("MinBy of nothing said there was an answer")
	}
	if _, ok := xs.MaxBy(empty, name); ok {
		t.Error("MaxBy of nothing said there was an answer")
	}
}

// TestMinByRunsTheKeyOncePerElement matters because the key is the caller's
// code and can be the expensive part.
func TestMinByRunsTheKeyOncePerElement(t *testing.T) {
	calls := 0
	in := slices.Values([]int{5, 3, 9, 1})

	xs.MinBy(in, func(n int) int {
		calls++
		return n
	})
	if calls != 4 {
		t.Errorf("it called the key %d times over four elements, want 4", calls)
	}
}

func TestCount(t *testing.T) {
	if got := xs.Count(slices.Values([]int{1, 2, 3})); got != 3 {
		t.Errorf("Count gave %d, want 3", got)
	}
	if got := xs.Count(slices.Values([]int(nil))); got != 0 {
		t.Errorf("Count of nothing gave %d, want 0", got)
	}
}

func TestCountReadsEverything(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.Count(in)
	if *read != 4 {
		t.Errorf("Count read %d elements, want all 4", *read)
	}
}

func TestCountBy(t *testing.T) {
	in := slices.Values([]string{"go", "rust", "gc", "ruby", "git"})

	got := xs.CountBy(in, func(s string) byte { return s[0] })
	want := map[byte]int{'g': 3, 'r': 2}

	if !maps.Equal(got, want) {
		t.Errorf("CountBy gave %v, want %v", got, want)
	}
}

func TestCountByOfNothingIsAnEmptyMap(t *testing.T) {
	got := xs.CountBy(slices.Values([]int(nil)), func(n int) int { return n })
	if got == nil {
		t.Error("CountBy of nothing gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("CountBy of nothing gave %v, want nothing", got)
	}
}
