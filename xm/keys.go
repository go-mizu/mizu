package xm

import (
	"cmp"
	"maps"
	"slices"
)

// Keys returns the keys as a slice, in no particular order.
//
//	registered := xm.Keys(handlers)
//
// [maps.Keys] is the same thing as a sequence, for a pipeline. This is the one
// for when a slice is what comes next. [SortedKeys] is the one to use when the
// order has to be the same twice running.
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Values returns the values as a slice, in no particular order. Two keys with
// the same value give two elements, since this is not a set.
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// SortedKeys returns the keys as a slice, in order.
//
//	for _, name := range xm.SortedKeys(fields) {
//		fmt.Fprintf(w, "\t%s %s\n", name, fields[name])
//	}
//
// This is what generated code, a configuration dump and anything else that has
// to produce the same output twice running should use. Ranging over a map
// directly gives a different order every time, which turns a diff into noise
// and a test into a flake.
func SortedKeys[M ~map[K]V, K cmp.Ordered, V any](m M) []K {
	return slices.Sorted(maps.Keys(m))
}

// Entry is one key and value out of a map, for the places a pair has to be a
// value rather than two return values: a slice, a sort, a JSON document.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

// Entries returns the key and value pairs as a slice, in no particular order.
//
//	byCount := xm.Entries(counts)
//	slices.SortFunc(byCount, func(a, b xm.Entry[string, int]) int {
//		return cmp.Compare(b.Value, a.Value)
//	})
//
// Sorting by value is what this is usually for, since a map cannot be sorted
// and [SortedKeys] only orders by key. [maps.All] is the lazy form, for a
// pipeline that does not need the slice.
func Entries[M ~map[K]V, K comparable, V any](m M) []Entry[K, V] {
	out := make([]Entry[K, V], 0, len(m))
	for k, v := range m {
		out = append(out, Entry[K, V]{k, v})
	}
	return out
}

// FromEntries builds a map out of a slice of pairs, which is what comes back
// from a decoded document or from [Entries] after a sort.
//
// A key that turns up twice keeps the last entry with it, the same as the loop
// this replaces.
func FromEntries[K comparable, V any](es []Entry[K, V]) map[K]V {
	out := make(map[K]V, len(es))
	for _, e := range es {
		out[e.Key] = e.Value
	}
	return out
}
