package xm_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

// sorted is here because most of this package gives back something in map
// order, which is no order at all, so the tests sort before they compare.
func sorted[T cmp.Ordered](s []T) []T {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

func TestKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	got := sorted(xm.Keys(m))
	if want := []string{"a", "b", "c"}; !slices.Equal(got, want) {
		t.Errorf("Keys gave %q, want %q", got, want)
	}
}

func TestValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	got := sorted(xm.Values(m))
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Values gave %v, want %v", got, want)
	}
}

// TestValuesKeepsRepeats is the difference between a slice of values and a set
// of them, and the reason there is no deduplication here.
func TestValuesKeepsRepeats(t *testing.T) {
	m := map[string]int{"a": 1, "b": 1}

	if got := xm.Values(m); len(got) != 2 {
		t.Errorf("Values gave %v, want both of them", got)
	}
}

func TestKeysAndValuesOfAnEmptyMap(t *testing.T) {
	var m map[string]int

	if got := xm.Keys(m); len(got) != 0 {
		t.Errorf("Keys of a nil map gave %q, want nothing", got)
	}
	if got := xm.Values(m); len(got) != 0 {
		t.Errorf("Values of a nil map gave %v, want nothing", got)
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{"pear": 1, "apple": 2, "quince": 3}

	got := xm.SortedKeys(m)
	if want := []string{"apple", "pear", "quince"}; !slices.Equal(got, want) {
		t.Errorf("SortedKeys gave %q, want %q", got, want)
	}
}

// TestSortedKeysIsTheSameEveryTime is the promise the whole function exists
// for, and a map big enough that ranging it twice would give two orders.
func TestSortedKeysIsTheSameEveryTime(t *testing.T) {
	m := make(map[int]bool, 64)
	for i := range 64 {
		m[i] = true
	}

	first := xm.SortedKeys(m)
	for range 20 {
		if got := xm.SortedKeys(m); !slices.Equal(got, first) {
			t.Fatalf("SortedKeys gave %v and then %v", first, got)
		}
	}
}

func TestSortedKeysOfAnEmptyMap(t *testing.T) {
	if got := xm.SortedKeys(map[string]int(nil)); len(got) != 0 {
		t.Errorf("SortedKeys of nothing gave %q, want nothing", got)
	}
}

func TestEntries(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}

	got := xm.Entries(m)
	slices.SortFunc(got, func(x, y xm.Entry[string, int]) int { return cmp.Compare(x.Key, y.Key) })

	want := []xm.Entry[string, int]{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	if !slices.Equal(got, want) {
		t.Errorf("Entries gave %v, want %v", got, want)
	}
}

func TestEntriesOfAnEmptyMap(t *testing.T) {
	if got := xm.Entries(map[string]int(nil)); len(got) != 0 {
		t.Errorf("Entries of nothing gave %v, want nothing", got)
	}
}

func TestFromEntries(t *testing.T) {
	es := []xm.Entry[string, int]{{Key: "a", Value: 1}, {Key: "b", Value: 2}}

	got := xm.FromEntries(es)
	if len(got) != 2 || got["a"] != 1 || got["b"] != 2 {
		t.Errorf("FromEntries gave %v, want a to 1 and b to 2", got)
	}
}

func TestFromEntriesKeepsTheLastOfADuplicate(t *testing.T) {
	es := []xm.Entry[string, int]{{Key: "a", Value: 1}, {Key: "a", Value: 2}}

	if got := xm.FromEntries(es); got["a"] != 2 {
		t.Errorf("FromEntries left %d under a, want the last one, 2", got["a"])
	}
}

func TestFromEntriesOfNothingIsAnEmptyMap(t *testing.T) {
	got := xm.FromEntries[string, int](nil)
	if got == nil {
		t.Error("FromEntries of nothing gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("FromEntries of nothing gave %v, want nothing", got)
	}
}

// TestEntriesAndFromEntriesComeBackTheSame is the round trip, which is the
// reason both are here.
func TestEntriesAndFromEntriesComeBackTheSame(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	got := xm.FromEntries(xm.Entries(m))
	if len(got) != len(m) {
		t.Fatalf("the round trip gave %d pairs, want %d", len(got), len(m))
	}
	for k, v := range m {
		if got[k] != v {
			t.Errorf("the round trip left %d under %q, want %d", got[k], k, v)
		}
	}
}
