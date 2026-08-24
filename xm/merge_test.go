package xm_test

import (
	"maps"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

func TestMerge(t *testing.T) {
	defaults := map[string]int{"port": 8080, "timeout": 30}
	override := map[string]int{"port": 9090}

	got := xm.Merge(defaults, override)
	want := map[string]int{"port": 9090, "timeout": 30}

	if !maps.Equal(got, want) {
		t.Errorf("Merge gave %v, want %v", got, want)
	}
}

// TestMergeGivesTheLastOneTheLastWord is the layering the function is for, so
// it gets a case of its own with three maps rather than two.
func TestMergeGivesTheLastOneTheLastWord(t *testing.T) {
	got := xm.Merge(
		map[string]string{"host": "a"},
		map[string]string{"host": "b"},
		map[string]string{"host": "c"},
	)

	if got["host"] != "c" {
		t.Errorf("Merge left %q under host, want the last one, c", got["host"])
	}
}

func TestMergeLeavesTheInputsAlone(t *testing.T) {
	first := map[string]int{"a": 1}
	second := map[string]int{"a": 2}

	xm.Merge(first, second)

	if first["a"] != 1 || second["a"] != 2 {
		t.Errorf("Merge changed its inputs to %v and %v", first, second)
	}
}

func TestMergeOfNothing(t *testing.T) {
	got := xm.Merge[map[string]int]()
	if got == nil {
		t.Error("Merge of no maps gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("Merge of no maps gave %v, want nothing", got)
	}
}

// TestMergeWhenTheFirstMapIsNil covers the copy of the first map giving back a
// nil rather than an empty one, which would otherwise panic on the write that
// comes after it.
func TestMergeWhenTheFirstMapIsNil(t *testing.T) {
	got := xm.Merge(map[string]int(nil), map[string]int{"a": 1})

	if want := map[string]int{"a": 1}; !maps.Equal(got, want) {
		t.Errorf("Merge gave %v, want %v", got, want)
	}
}

func TestMergeOfOnlyNilMaps(t *testing.T) {
	got := xm.Merge(map[string]int(nil), nil)
	if got == nil {
		t.Error("Merge of nil maps gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("Merge of nil maps gave %v, want nothing", got)
	}
}

func TestMergeOfOneMapIsACopy(t *testing.T) {
	in := map[string]int{"a": 1}

	got := xm.Merge(in)
	got["a"] = 2

	if in["a"] != 1 {
		t.Error("writing to the result of Merge changed the input")
	}
}

func TestMergeWith(t *testing.T) {
	january := map[string]int{"ana": 3, "ben": 1}
	february := map[string]int{"ana": 4, "cara": 2}

	got := xm.MergeWith(func(k string, a, b int) int { return a + b }, january, february)
	want := map[string]int{"ana": 7, "ben": 1, "cara": 2}

	if !maps.Equal(got, want) {
		t.Errorf("MergeWith gave %v, want %v", got, want)
	}
}

// TestMergeWithRunsResolveOnlyOnAClash pins the promise in the doc comment,
// since a resolve that runs on every key would be a different function.
func TestMergeWithRunsResolveOnlyOnAClash(t *testing.T) {
	calls := 0
	resolve := func(k string, a, b int) int {
		calls++
		return b
	}

	xm.MergeWith(resolve,
		map[string]int{"a": 1, "b": 2},
		map[string]int{"b": 3, "c": 4},
	)

	if calls != 1 {
		t.Errorf("resolve ran %d times, want once, for b", calls)
	}
}

// TestMergeWithPassesTheArgumentsInOrder checks that the value already there
// comes before the one arriving, which is the part a caller can get backwards.
func TestMergeWithPassesTheArgumentsInOrder(t *testing.T) {
	got := xm.MergeWith(
		func(k string, kept, arriving string) string { return kept + arriving },
		map[string]string{"x": "first"},
		map[string]string{"x": "second"},
	)

	if got["x"] != "firstsecond" {
		t.Errorf("MergeWith gave %q, want firstsecond", got["x"])
	}
}

func TestMergeWithSeesTheKey(t *testing.T) {
	var seen []string
	xm.MergeWith(func(k string, a, b int) int {
		seen = append(seen, k)
		return b
	}, map[string]int{"a": 1}, map[string]int{"a": 2})

	if len(seen) != 1 || seen[0] != "a" {
		t.Errorf("resolve saw %q, want a", seen)
	}
}

func TestMergeWithOfNothing(t *testing.T) {
	got := xm.MergeWith[map[string]int](func(k string, a, b int) int { return b })
	if got == nil {
		t.Error("MergeWith of no maps gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("MergeWith of no maps gave %v, want nothing", got)
	}
}

func TestPick(t *testing.T) {
	row := map[string]string{"id": "7", "name": "ana", "password": "hunter2"}

	got := xm.Pick(row, "id", "name")
	want := map[string]string{"id": "7", "name": "ana"}

	if !maps.Equal(got, want) {
		t.Errorf("Pick gave %v, want %v", got, want)
	}
}

func TestPickSkipsAKeyThatIsNotThere(t *testing.T) {
	got := xm.Pick(map[string]int{"a": 1}, "a", "missing")

	if len(got) != 1 || got["a"] != 1 {
		t.Errorf("Pick gave %v, want only a", got)
	}
	if _, there := got["missing"]; there {
		t.Error("Pick put the missing key in the result")
	}
}

func TestPickOfNoKeys(t *testing.T) {
	got := xm.Pick(map[string]int{"a": 1})
	if len(got) != 0 {
		t.Errorf("Pick with no keys gave %v, want nothing", got)
	}
}

func TestOmit(t *testing.T) {
	row := map[string]string{"id": "7", "password": "hunter2", "apiKey": "sk-1"}

	got := xm.Omit(row, "password", "apiKey")
	want := map[string]string{"id": "7"}

	if !maps.Equal(got, want) {
		t.Errorf("Omit gave %v, want %v", got, want)
	}
}

func TestOmitIgnoresAKeyThatIsNotThere(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2}

	got := xm.Omit(in, "c")
	if !maps.Equal(got, in) {
		t.Errorf("Omit gave %v, want %v", got, in)
	}
}

func TestOmitLeavesTheInputAlone(t *testing.T) {
	in := map[string]string{"id": "7", "password": "hunter2"}

	xm.Omit(in, "password")
	if in["password"] != "hunter2" {
		t.Error("Omit took the key out of the input, want a new map")
	}
}

// TestPickAndOmitAreOpposites covers both from the other side, which is the
// property a caller reasons with when choosing between them.
func TestPickAndOmitAreOpposites(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2, "c": 3}

	kept := xm.Pick(in, "a", "b")
	dropped := xm.Omit(in, "a", "b")

	if !maps.Equal(xm.Merge(kept, dropped), in) {
		t.Errorf("Pick gave %v and Omit gave %v, want %v between them", kept, dropped, in)
	}
}
