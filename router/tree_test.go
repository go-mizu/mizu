package router

import (
	"fmt"
	"strings"
	"testing"
)

func TestFirstSegment(t *testing.T) {
	cases := []struct{ in, seg, rest string }{
		{"/", "/", ""},
		{"/a", "a", ""},
		{"/a/", "a", "/"},
		{"/a/b", "a", "/b"},
		{"/a/b/c", "a", "/b/c"},
		{"//a", "", "/a"},
		{"/caf%C3%A9/x", "café", "/x"},
		{"/a%2Fb", "a/b", ""},
		{"/a%zz", "a%zz", ""},
	}
	for _, c := range cases {
		seg, rest := firstSegment(c.in)
		if seg != c.seg || rest != c.rest {
			t.Errorf("firstSegment(%q) = %q, %q, want %q, %q", c.in, seg, rest, c.seg, c.rest)
		}
	}
}

// A mapping holds a node's literal children as a slice while there are few of
// them and turns into a map after that, and has to read the same either way.
func TestMappingGrows(t *testing.T) {
	var m mapping
	keys := make([]string, 0, scanUpTo*3)
	for i := range cap(keys) {
		keys = append(keys, fmt.Sprintf("k%d", i))
	}

	for i, k := range keys {
		m.add(k, &node{})
		if want := i < scanUpTo; (m.m == nil) != want {
			t.Errorf("after %d keys, slice = %v, want %v", i+1, m.m == nil, want)
		}
		for _, seen := range keys[:i+1] {
			if m.find(seen) == nil {
				t.Errorf("after %d keys, find(%q) came back empty", i+1, seen)
			}
		}
		if m.find("nothing") != nil {
			t.Errorf("find(%q) found something", "nothing")
		}
	}

	var count int
	m.each(func(string, *node) { count++ })
	if count != len(keys) {
		t.Errorf("each visited %d children, want %d", count, len(keys))
	}
}

// values keeps the first few wildcard values in an array and spills the rest,
// and backtracking cuts it back, so the two halves have to agree about where
// each value is.
func TestValues(t *testing.T) {
	var v values
	want := make([]string, 0, onStack*2+1)
	for i := range cap(want) {
		s := fmt.Sprintf("v%d", i)
		v.push(s)
		want = append(want, s)
	}
	if v.n != len(want) {
		t.Fatalf("n = %d, want %d", v.n, len(want))
	}
	for i, w := range want {
		if got := v.at(i); got != w {
			t.Errorf("at(%d) = %q, want %q", i, got, w)
		}
	}

	// Cutting back and pushing again is what a failed branch does.
	for _, mark := range []int{len(want) - 1, onStack + 1, onStack, onStack - 1, 1, 0} {
		v.take(mark)
		if v.n != mark {
			t.Fatalf("after take(%d), n = %d", mark, v.n)
		}
		v.push("again")
		if got := v.at(mark); got != "again" {
			t.Errorf("after take(%d), at(%d) = %q, want %q", mark, mark, got, "again")
		}
		v.take(mark)
	}

	// take never grows the record, so a mark from a branch that pushed nothing
	// leaves it alone.
	v.push("x")
	v.take(99)
	if v.n != 1 {
		t.Errorf("take(99) with 1 value left n = %d, want 1", v.n)
	}
	v.reset()
	if v.n != 0 {
		t.Errorf("reset left n = %d, want 0", v.n)
	}
}

// A route with more wildcards than the stack holds still matches, and is the one
// case that allocates.
func TestMoreWildcardsThanTheStackHolds(t *testing.T) {
	var pat, path strings.Builder
	var names []string
	for i := range onStack + 3 {
		name := fmt.Sprintf("p%d", i)
		names = append(names, name)
		fmt.Fprintf(&pat, "/{%s}", name)
		fmt.Fprintf(&path, "/v%d", i)
	}

	r := New()
	r.Handle(pat.String(), http200)
	_, ps, ok := r.Lookup("GET", "", path.String())
	if !ok {
		t.Fatalf("%s did not match %s", pat.String(), path.String())
	}
	if ps.Len() != len(names) {
		t.Fatalf("Len() = %d, want %d", ps.Len(), len(names))
	}
	for i, name := range names {
		if got, want := ps.Get(name), fmt.Sprintf("v%d", i); got != want {
			t.Errorf("Get(%q) = %q, want %q", name, got, want)
		}
	}
}
