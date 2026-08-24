package xm_test

import (
	"maps"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

func TestMap(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2}

	got := xm.Map(in, func(k string, v int) (int, string) { return v, k })
	want := map[int]string{1: "a", 2: "b"}

	if !maps.Equal(got, want) {
		t.Errorf("Map gave %v, want %v", got, want)
	}
}

func TestMapKeys(t *testing.T) {
	in := map[string]int{"Content-Type": 1, "ACCEPT": 2}

	got := xm.MapKeys(in, func(k string, v int) string { return strings.ToLower(k) })
	want := map[string]int{"content-type": 1, "accept": 2}

	if !maps.Equal(got, want) {
		t.Errorf("MapKeys gave %v, want %v", got, want)
	}
}

func TestMapValues(t *testing.T) {
	in := map[string][]int{"a": {1, 2, 3}, "b": nil}

	got := xm.MapValues(in, func(k string, v []int) int { return len(v) })
	want := map[string]int{"a": 3, "b": 0}

	if !maps.Equal(got, want) {
		t.Errorf("MapValues gave %v, want %v", got, want)
	}
}

// TestMapValuesSeesTheKey matters because the key is often what decides how the
// value should be rewritten.
func TestMapValuesSeesTheKey(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2}

	got := xm.MapValues(in, func(k string, v int) string { return k + strconv.Itoa(v) })
	want := map[string]string{"a": "a1", "b": "b2"}

	if !maps.Equal(got, want) {
		t.Errorf("MapValues gave %v, want %v", got, want)
	}
}

func TestMapOfAnEmptyMap(t *testing.T) {
	got := xm.Map(map[string]int(nil), func(k string, v int) (string, int) { return k, v })
	if got == nil {
		t.Error("Map of nothing gave a nil map, want an empty one")
	}
	if len(got) != 0 {
		t.Errorf("Map of nothing gave %v, want nothing", got)
	}
}

func TestFilter(t *testing.T) {
	in := map[string]bool{"logging": true, "tracing": false, "metrics": true}

	got := xm.Filter(in, func(k string, on bool) bool { return on })
	want := map[string]bool{"logging": true, "metrics": true}

	if !maps.Equal(got, want) {
		t.Errorf("Filter gave %v, want %v", got, want)
	}
}

// TestFilterLeavesTheInputAlone is the difference from maps.DeleteFunc, which
// is the reason to have Filter at all.
func TestFilterLeavesTheInputAlone(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2}
	before := maps.Clone(in)

	xm.Filter(in, func(k string, v int) bool { return v > 1 })
	if !maps.Equal(in, before) {
		t.Errorf("Filter changed the input to %v, want %v", in, before)
	}
}

func TestReject(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2, "c": 3}

	got := xm.Reject(in, func(k string, v int) bool { return v%2 == 1 })
	want := map[string]int{"b": 2}

	if !maps.Equal(got, want) {
		t.Errorf("Reject gave %v, want %v", got, want)
	}
}

func TestFilterAndRejectAreOpposites(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	even := func(k string, v int) bool { return v%2 == 0 }

	kept := xm.Filter(in, even)
	dropped := xm.Reject(in, even)

	if len(kept)+len(dropped) != len(in) {
		t.Errorf("Filter kept %d and Reject kept %d, want %d between them", len(kept), len(dropped), len(in))
	}
	for k := range kept {
		if _, both := dropped[k]; both {
			t.Errorf("%q came out of both", k)
		}
	}
}

// TestFilterKeepsTheMapType is what the ~map[K]V constraint is for, and a named
// map type is common enough in configuration code to be worth pinning.
func TestFilterKeepsTheMapType(t *testing.T) {
	type headers map[string]string

	var got headers = xm.Filter(headers{"a": "1", "b": ""}, func(k, v string) bool { return v != "" })
	if len(got) != 1 || got["a"] != "1" {
		t.Errorf("Filter gave %v, want only a", got)
	}
}

func TestInvert(t *testing.T) {
	in := map[string]int{"ana": 1, "ben": 2}

	got := xm.Invert(in)
	want := map[int]string{1: "ana", 2: "ben"}

	if !maps.Equal(got, want) {
		t.Errorf("Invert gave %v, want %v", got, want)
	}
}

// TestInvertOfAMapWithRepeatedValues pins the size rather than which key
// survives, since which one survives depends on the map order and is not
// something to promise.
func TestInvertOfAMapWithRepeatedValues(t *testing.T) {
	in := map[string]int{"a": 1, "b": 1}

	got := xm.Invert(in)
	if len(got) != 1 {
		t.Errorf("Invert gave %v, want one entry", got)
	}
	if got[1] != "a" && got[1] != "b" {
		t.Errorf("Invert left %q under 1, want one of the two keys", got[1])
	}
}

func TestInvertTwiceComesBack(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2}

	if got := xm.Invert(xm.Invert(in)); !maps.Equal(got, in) {
		t.Errorf("inverting twice gave %v, want %v", got, in)
	}
}
