package xm_test

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

func TestOfIsTheSameMap(t *testing.T) {
	in := map[string]int{"a": 1}

	chain := xm.Of(in)
	chain["b"] = 2

	if in["b"] != 2 {
		t.Errorf("writing to the chain left the map as %v, want the write to land in it", in)
	}
}

func TestChain(t *testing.T) {
	in := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}

	got := xm.Of(in).
		Filter(func(k string, v int) bool { return v%2 == 0 }).
		Omit("d")
	want := map[string]int{"b": 2}

	if !maps.Equal(got, want) {
		t.Errorf("the chain gave %v, want %v", got, want)
	}
}

// TestChainIsAPlainMap is the reason there is no terminal method. Everything a
// map does, the chain does, and a caller expecting a map[K]V takes one without
// a conversion.
func TestChainIsAPlainMap(t *testing.T) {
	got := xm.Of(map[string]int{"a": 1, "b": 2}).Filter(func(k string, v int) bool { return v > 1 })

	if len(got) != 1 {
		t.Errorf("len gave %d, want 1", len(got))
	}
	if got["b"] != 2 {
		t.Errorf("indexing gave %d, want 2", got["b"])
	}
	if _, found := got["a"]; found {
		t.Error("a is still there, want it filtered out")
	}

	count := 0
	for range got {
		count++
	}
	if count != 1 {
		t.Errorf("ranging saw %d pairs, want 1", count)
	}

	var plain map[string]int = got
	if !maps.Equal(plain, map[string]int{"b": 2}) {
		t.Errorf("as a plain map it is %v, want map[b:2]", plain)
	}

	if !takesAPlainMap(got) {
		t.Error("passing the chain to a function taking map[string]int did not work")
	}
}

func takesAPlainMap(m map[string]int) bool { return m != nil }

// TestFreeFunctionsTakeTheChain is the other half of the same property: the
// functions that cannot be methods still take the chain as it stands.
func TestFreeFunctionsTakeTheChain(t *testing.T) {
	chain := xm.Of(map[string]int{"b": 2, "a": 1, "c": 3}).Reject(func(k string, v int) bool { return v == 3 })

	if got := xm.SortedKeys(chain); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("SortedKeys gave %v, want [a b]", got)
	}
	if got := xm.Invert(chain); !maps.Equal(got, map[int]string{1: "a", 2: "b"}) {
		t.Errorf("Invert gave %v, want map[1:a 2:b]", got)
	}
}

// TestFreeFunctionsHandBackTheChain matters because it is what lets a chain
// carry on through a free function rather than ending at one.
func TestFreeFunctionsHandBackTheChain(t *testing.T) {
	got := xm.Filter(xm.Of(map[string]int{"a": 1, "b": 2}), func(k string, v int) bool { return v > 1 }).
		MapValues(func(k string, v int) int { return v * 10 })

	if !maps.Equal(got, map[string]int{"b": 20}) {
		t.Errorf("the chain gave %v, want map[b:20]", got)
	}
}

func TestChainMapValues(t *testing.T) {
	got := xm.Of(map[string][]int{"a": {1, 2, 3}, "b": nil}).
		MapValues(func(k string, v []int) int { return len(v) })

	if !maps.Equal(got, map[string]int{"a": 3, "b": 0}) {
		t.Errorf("MapValues gave %v, want map[a:3 b:0]", got)
	}
}

func TestChainMapKeys(t *testing.T) {
	got := xm.Of(map[string]int{"Content-Type": 1, "ACCEPT": 2}).
		MapKeys(func(k string, v int) string { return strings.ToLower(k) })

	if !maps.Equal(got, map[string]int{"content-type": 1, "accept": 2}) {
		t.Errorf("MapKeys gave %v, want the keys lowered", got)
	}
}

func TestChainMap(t *testing.T) {
	got := xm.Of(map[string]int{"a": 1, "b": 2}).
		Map(func(k string, v int) (int, string) { return v, k })

	if !maps.Equal(got, map[int]string{1: "a", 2: "b"}) {
		t.Errorf("Map gave %v, want map[1:a 2:b]", got)
	}
}

// TestTheMappingMethodsNeedNoTypeArgument is the whole point of a generic
// method. If inference ever stopped working here, every call site above would
// have to name a type it does not name today.
func TestTheMappingMethodsNeedNoTypeArgument(t *testing.T) {
	m := xm.Of(map[string]int{"a": 1})

	if got := m.MapValues(func(k string, v int) string { return strconv.Itoa(v) }); got["a"] != "1" {
		t.Errorf("MapValues gave %v, want map[a:1] with a string in it", got)
	}
	if got := m.MapKeys(func(k string, v int) int { return len(k) }); got[1] != 1 {
		t.Errorf("MapKeys gave %v, want map[1:1]", got)
	}
	if got := m.Map(func(k string, v int) (bool, float64) { return true, 1.5 }); got[true] != 1.5 {
		t.Errorf("Map gave %v, want map[true:1.5]", got)
	}
}

// TestMapValuesAsAMethodValue is the other thing a type parameter on a method
// buys, written out because it is the only way to name one.
func TestMapValuesAsAMethodValue(t *testing.T) {
	rewrite := xm.Of(map[string]int{"a": 1, "b": 2}).MapValues[string]

	got := rewrite(func(k string, v int) string { return k + strconv.Itoa(v) })
	if !maps.Equal(got, map[string]string{"a": "a1", "b": "b2"}) {
		t.Errorf("the method value gave %v, want map[a:a1 b:b2]", got)
	}
}

// TestChainAcrossThreeTypeChanges is the case free functions cannot express
// without naming an intermediate map at every step.
func TestChainAcrossThreeTypeChanges(t *testing.T) {
	got := xm.Of(map[string]int{"a": 1, "bb": 2, "ccc": 3}).
		Filter(func(k string, v int) bool { return v > 1 }).
		MapValues(func(k string, v int) string { return strconv.Itoa(v * 2) }).
		MapKeys(func(k string, v string) int { return len(k) }).
		Map(func(k int, v string) (string, int) { n, _ := strconv.Atoi(v); return strconv.Itoa(k), n })

	if !maps.Equal(got, map[string]int{"2": 4, "3": 6}) {
		t.Errorf("the chain gave %v, want map[2:4 3:6]", got)
	}
}

func TestChainKeysAndValues(t *testing.T) {
	m := xm.Of(map[string]int{"a": 1, "b": 2})

	keys := m.Keys()
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"a", "b"}) {
		t.Errorf("Keys gave %v, want [a b]", keys)
	}

	values := m.Values()
	slices.Sort(values)
	if !slices.Equal(values, []int{1, 2}) {
		t.Errorf("Values gave %v, want [1 2]", values)
	}
}

func TestChainEntries(t *testing.T) {
	got := xm.Of(map[string]int{"a": 1, "b": 2}).Entries()
	slices.SortFunc(got, func(x, y xm.Entry[string, int]) int { return strings.Compare(x.Key, y.Key) })

	want := []xm.Entry[string, int]{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	if !slices.Equal(got, want) {
		t.Errorf("Entries gave %v, want %v", got, want)
	}
}

func TestChainPickAndOmit(t *testing.T) {
	m := xm.Of(map[string]string{"id": "1", "name": "ana", "password": "x"})

	if got := m.Pick("id", "name", "missing"); !maps.Equal(got, map[string]string{"id": "1", "name": "ana"}) {
		t.Errorf("Pick gave %v, want id and name", got)
	}
	if got := m.Omit("password"); !maps.Equal(got, map[string]string{"id": "1", "name": "ana"}) {
		t.Errorf("Omit gave %v, want everything but the password", got)
	}
}

func TestChainMerge(t *testing.T) {
	defaults := xm.Of(map[string]int{"port": 80, "timeout": 30})

	got := defaults.Merge(map[string]int{"port": 8080}, map[string]int{"port": 9090, "workers": 4})
	want := map[string]int{"port": 9090, "timeout": 30, "workers": 4}

	if !maps.Equal(got, want) {
		t.Errorf("Merge gave %v, want %v", got, want)
	}
	if defaults["port"] != 80 {
		t.Errorf("Merge changed the receiver to %v, want it left alone", defaults)
	}
}

func TestChainMergeWithNoOthers(t *testing.T) {
	in := xm.Of(map[string]int{"a": 1})

	got := in.Merge()
	if !maps.Equal(got, map[string]int{"a": 1}) {
		t.Errorf("Merge of nothing gave %v, want a copy", got)
	}

	got["b"] = 2
	if _, leaked := in["b"]; leaked {
		t.Error("Merge handed back the receiver, want a copy")
	}
}

func TestChainMergeWith(t *testing.T) {
	add := func(k string, kept, arriving int) int { return kept + arriving }

	got := xm.Of(map[string]int{"ana": 3, "ben": 1}).
		MergeWith(add, map[string]int{"ana": 4}, map[string]int{"cal": 2})
	want := map[string]int{"ana": 7, "ben": 1, "cal": 2}

	if !maps.Equal(got, want) {
		t.Errorf("MergeWith gave %v, want %v", got, want)
	}
}

func TestChainGetOr(t *testing.T) {
	m := xm.Of(map[string]int{"set": 0})

	if got := m.GetOr("set", 8080); got != 0 {
		t.Errorf("GetOr gave %d for a key set to zero, want 0", got)
	}
	if got := m.GetOr("unset", 8080); got != 8080 {
		t.Errorf("GetOr gave %d for a missing key, want 8080", got)
	}
}

func TestChainUpdate(t *testing.T) {
	counts := xm.Of(map[string]int{})

	counts.Update("word", func(n int) int { return n + 1 })
	counts.Update("word", func(n int) int { return n + 1 })

	if counts["word"] != 2 {
		t.Errorf("Update left %d, want 2", counts["word"])
	}
}

// TestTheMethodsMatchTheFreeFunctions is what makes the whole file safe to
// skim. Every method here is the free function with the arguments the other way
// round, and this is that written down.
func TestTheMethodsMatchTheFreeFunctions(t *testing.T) {
	in := map[string]int{"a": 1, "bb": 2, "ccc": 3, "dddd": 4}
	m := xm.Of(in)

	keep := func(k string, v int) bool { return v%2 == 0 }
	if got, want := m.Filter(keep), xm.Filter(in, keep); !maps.Equal(got, want) {
		t.Errorf("Filter: method gave %v, function gave %v", got, want)
	}
	if got, want := m.Reject(keep), xm.Reject(in, keep); !maps.Equal(got, want) {
		t.Errorf("Reject: method gave %v, function gave %v", got, want)
	}
	if got, want := m.Pick("a", "bb"), xm.Pick(in, "a", "bb"); !maps.Equal(got, want) {
		t.Errorf("Pick: method gave %v, function gave %v", got, want)
	}
	if got, want := m.Omit("a"), xm.Omit(in, "a"); !maps.Equal(got, want) {
		t.Errorf("Omit: method gave %v, function gave %v", got, want)
	}

	double := func(k string, v int) int { return v * 2 }
	if got, want := m.MapValues(double), xm.MapValues(in, double); !maps.Equal(got, want) {
		t.Errorf("MapValues: method gave %v, function gave %v", got, want)
	}

	length := func(k string, v int) int { return len(k) }
	if got, want := m.MapKeys(length), xm.MapKeys(in, length); !maps.Equal(got, want) {
		t.Errorf("MapKeys: method gave %v, function gave %v", got, want)
	}

	swap := func(k string, v int) (int, string) { return v, k }
	if got, want := m.Map(swap), xm.Map(in, swap); !maps.Equal(got, want) {
		t.Errorf("Map: method gave %v, function gave %v", got, want)
	}

	other := map[string]int{"a": 100, "e": 5}
	if got, want := m.Merge(other), xm.Merge(in, other); !maps.Equal(got, want) {
		t.Errorf("Merge: method gave %v, function gave %v", got, want)
	}

	add := func(k string, kept, arriving int) int { return kept + arriving }
	if got, want := m.MergeWith(add, other), xm.MergeWith(add, in, other); !maps.Equal(got, want) {
		t.Errorf("MergeWith: method gave %v, function gave %v", got, want)
	}

	if got, want := m.GetOr("a", -1), xm.GetOr(in, "a", -1); got != want {
		t.Errorf("GetOr: method gave %d, function gave %d", got, want)
	}
	if got, want := m.GetOr("nope", -1), xm.GetOr(in, "nope", -1); got != want {
		t.Errorf("GetOr: method gave %d, function gave %d", got, want)
	}

	gotKeys, wantKeys := m.Keys(), xm.Keys(in)
	slices.Sort(gotKeys)
	slices.Sort(wantKeys)
	if !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("Keys: method gave %v, function gave %v", gotKeys, wantKeys)
	}

	gotValues, wantValues := m.Values(), xm.Values(in)
	slices.Sort(gotValues)
	slices.Sort(wantValues)
	if !slices.Equal(gotValues, wantValues) {
		t.Errorf("Values: method gave %v, function gave %v", gotValues, wantValues)
	}

	if got, want := len(m.Entries()), len(xm.Entries(in)); got != want {
		t.Errorf("Entries: method gave %d pairs, function gave %d", got, want)
	}
}

// TestChainOfANilMap checks that starting from a map nobody made still gives
// something to write into, the same as the free functions do.
func TestChainOfANilMap(t *testing.T) {
	m := xm.Of(map[string]int(nil))

	if got := m.Filter(func(k string, v int) bool { return true }); got == nil {
		t.Error("Filter of nil gave a nil map, want an empty one")
	}
	if got := m.Merge(); got == nil {
		t.Error("Merge of nil gave a nil map, want an empty one")
	}
	if got := m.MapValues(func(k string, v int) int { return v }); got == nil {
		t.Error("MapValues of nil gave a nil map, want an empty one")
	}
	if got := m.GetOr("a", 7); got != 7 {
		t.Errorf("GetOr on nil gave %d, want the fallback", got)
	}
	if got := m.Keys(); len(got) != 0 {
		t.Errorf("Keys of nil gave %v, want nothing", got)
	}
}

// TestChainDropsANamedMapType is the one thing the chain gives up that the free
// functions do not. Filter of a headers gives back a headers, and the chain
// gives back an M, so getting the name back is a conversion the caller writes.
// It is pinned here because it is the only reason to reach for the free
// functions when the chain would otherwise do.
func TestChainDropsANamedMapType(t *testing.T) {
	type headers map[string]string

	in := headers{"a": "1", "b": ""}
	nonEmpty := func(k, v string) bool { return v != "" }

	var kept headers = xm.Filter(in, nonEmpty)
	if len(kept) != 1 || kept["a"] != "1" {
		t.Errorf("Filter gave %v, want only a", kept)
	}

	back := headers(xm.Of(in).Filter(nonEmpty))
	if !maps.Equal(back, kept) {
		t.Errorf("the chain gave %v, want %v", back, kept)
	}
}
