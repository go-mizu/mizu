package xs_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestFirst(t *testing.T) {
	got, ok := xs.First(slices.Values([]string{"a", "b", "c"}))
	if !ok || got != "a" {
		t.Errorf("First gave %q and %v, want %q and true", got, ok, "a")
	}
}

func TestFirstOfNothing(t *testing.T) {
	got, ok := xs.First(slices.Values([]string(nil)))
	if ok {
		t.Error("First of nothing said there was an element")
	}
	if got != "" {
		t.Errorf("First of nothing gave %q, want the zero value", got)
	}
}

// TestFirstReadsOneElement is most of why First is worth having over a Collect
// and an index.
func TestFirstReadsOneElement(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.First(in)
	if *read != 1 {
		t.Errorf("First read %d elements, want 1", *read)
	}
}

func TestLast(t *testing.T) {
	got, ok := xs.Last(slices.Values([]string{"a", "b", "c"}))
	if !ok || got != "c" {
		t.Errorf("Last gave %q and %v, want %q and true", got, ok, "c")
	}
}

func TestLastOfNothing(t *testing.T) {
	if _, ok := xs.Last(slices.Values([]int(nil))); ok {
		t.Error("Last of nothing said there was an element")
	}
}

func TestLastReadsEverything(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.Last(in)
	if *read != 4 {
		t.Errorf("Last read %d elements, want all 4", *read)
	}
}

func TestFind(t *testing.T) {
	in := slices.Values([]string{"go.mod", "main.go", "go.sum"})

	got, found := xs.Find(in, func(s string) bool { return strings.HasSuffix(s, ".go") })
	if !found || got != "main.go" {
		t.Errorf("Find gave %q and %v, want %q and true", got, found, "main.go")
	}
}

func TestFindWithNoMatch(t *testing.T) {
	in := slices.Values([]int{1, 3, 5})

	got, found := xs.Find(in, func(n int) bool { return n%2 == 0 })
	if found {
		t.Errorf("Find found %d, want nothing", got)
	}
	if got != 0 {
		t.Errorf("Find gave %d with nothing found, want the zero value", got)
	}
}

func TestFindStopsAtTheMatch(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.Find(in, func(n int) bool { return n == 2 })
	if *read != 2 {
		t.Errorf("Find read %d elements to reach the second, want 2", *read)
	}
}

func TestIndex(t *testing.T) {
	in := slices.Values([]string{"one", "two", "three"})

	got := xs.Index(in, func(s string) bool { return len(s) == 5 })
	if got != 2 {
		t.Errorf("Index gave %d, want 2", got)
	}
}

func TestIndexWithNoMatchIsMinusOne(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})

	if got := xs.Index(in, func(n int) bool { return n > 10 }); got != -1 {
		t.Errorf("Index gave %d with nothing to find, want -1", got)
	}
}

func TestIndexOfNothingIsMinusOne(t *testing.T) {
	if got := xs.Index(slices.Values([]int(nil)), func(int) bool { return true }); got != -1 {
		t.Errorf("Index over nothing gave %d, want -1", got)
	}
}

func TestIndexStopsAtTheMatch(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.Index(in, func(n int) bool { return n == 1 })
	if *read != 1 {
		t.Errorf("Index read %d elements to reach the first, want 1", *read)
	}
}

func TestAny(t *testing.T) {
	in := []int{1, 3, 4, 5}

	if !xs.Any(slices.Values(in), func(n int) bool { return n%2 == 0 }) {
		t.Error("Any said there was no even number in 1, 3, 4, 5")
	}
	if xs.Any(slices.Values(in), func(n int) bool { return n > 10 }) {
		t.Error("Any said there was a number over ten in 1, 3, 4, 5")
	}
}

func TestAnyOfNothingIsFalse(t *testing.T) {
	if xs.Any(slices.Values([]int(nil)), func(int) bool { return true }) {
		t.Error("Any over nothing was true")
	}
}

func TestAnyStopsAtTheFirstYes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.Any(in, func(n int) bool { return n == 2 })
	if *read != 2 {
		t.Errorf("Any read %d elements past the answer, want 2", *read)
	}
}

func TestAll(t *testing.T) {
	in := []int{2, 4, 6}

	if !xs.All(slices.Values(in), func(n int) bool { return n%2 == 0 }) {
		t.Error("All said 2, 4, 6 are not all even")
	}
	if xs.All(slices.Values(in), func(n int) bool { return n > 2 }) {
		t.Error("All said 2, 4, 6 are all over two")
	}
}

// TestAllOfNothingIsTrue is the answer that keeps All over two halves the same
// as All over the whole, which is why it is worth a test of its own.
func TestAllOfNothingIsTrue(t *testing.T) {
	if !xs.All(slices.Values([]int(nil)), func(int) bool { return false }) {
		t.Error("All over nothing was false")
	}
}

func TestAllStopsAtTheFirstNo(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.All(in, func(n int) bool { return n < 2 })
	if *read != 2 {
		t.Errorf("All read %d elements past the answer, want 2", *read)
	}
}

func TestNone(t *testing.T) {
	in := []int{1, 3, 5}

	if !xs.None(slices.Values(in), func(n int) bool { return n%2 == 0 }) {
		t.Error("None said there is an even number in 1, 3, 5")
	}
	if xs.None(slices.Values(in), func(n int) bool { return n == 3 }) {
		t.Error("None missed the 3")
	}
}

func TestNoneOfNothingIsTrue(t *testing.T) {
	if !xs.None(slices.Values([]int(nil)), func(int) bool { return true }) {
		t.Error("None over nothing was false")
	}
}

func TestNoneStopsAtTheFirstYes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	xs.None(in, func(n int) bool { return n == 2 })
	if *read != 2 {
		t.Errorf("None read %d elements past the answer, want 2", *read)
	}
}
