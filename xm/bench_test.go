package xm_test

import (
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/go-mizu/mizu/xm"
)

var sink int

// build makes a map of n pairs with keys that sort in a different order from
// the one they were inserted in.
func build(n int) map[string]int {
	m := make(map[string]int, n)
	for i := range n {
		m[strconv.Itoa(i)] = i
	}
	return m
}

func BenchmarkKeys(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Keys(m))
	}
}

// BenchmarkKeysStdlib is the same work through the standard library, so the
// cost of the slice is visible next to the cost of the sequence.
func BenchmarkKeysStdlib(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(slices.Collect(maps.Keys(m)))
	}
}

func BenchmarkSortedKeys(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.SortedKeys(m))
	}
}

func BenchmarkEntries(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Entries(m))
	}
}

func BenchmarkMapValues(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.MapValues(m, func(k string, v int) int { return v * 2 }))
	}
}

func BenchmarkFilter(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Filter(m, func(k string, v int) bool { return v%2 == 0 }))
	}
}

func BenchmarkMerge(b *testing.B) {
	first, second := build(1000), build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Merge(first, second))
	}
}

// BenchmarkMergeStdlib is the two-step version a caller writes without Merge,
// which allocates the same once but has to name the intermediate map.
func BenchmarkMergeStdlib(b *testing.B) {
	first, second := build(1000), build(1000)
	b.ReportAllocs()

	for b.Loop() {
		out := maps.Clone(first)
		maps.Copy(out, second)
		sink = len(out)
	}
}

// BenchmarkPick is against a map far bigger than the key list, which is the
// case Pick is for and the reason it walks the keys rather than the map.
func BenchmarkPick(b *testing.B) {
	m := build(10000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Pick(m, "1", "2", "3"))
	}
}

func BenchmarkOmit(b *testing.B) {
	m := build(1000)
	b.ReportAllocs()

	for b.Loop() {
		sink = len(xm.Omit(m, "1", "2", "3"))
	}
}

func BenchmarkUpdate(b *testing.B) {
	counts := map[string]int{"key": 0}
	b.ReportAllocs()

	for b.Loop() {
		xm.Update(counts, "key", func(n int) int { return n + 1 })
	}
	sink = counts["key"]
}

// BenchmarkChain is the chain against the free functions doing the same work,
// which is what says whether a method costs anything over the function it
// wraps.
func BenchmarkChain(b *testing.B) {
	m := build(1000)
	keep := func(k string, v int) bool { return v%2 == 0 }
	double := func(k string, v int) int { return v * 2 }

	b.Run("chained", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.Of(m).Filter(keep).MapValues(double))
		}
	})

	b.Run("free functions", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.MapValues(xm.Filter(m, keep), double))
		}
	})
}

// BenchmarkChainMapValues is the generic method on its own against the free
// function, with no other step in the way to hide the difference.
func BenchmarkChainMapValues(b *testing.B) {
	m := build(1000)
	double := func(k string, v int) int { return v * 2 }

	b.Run("method", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.Of(m).MapValues(double))
		}
	})

	b.Run("free function", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.MapValues(m, double))
		}
	})
}

// BenchmarkChainMerge is the one method that does not hand straight through:
// it builds a slice to put the receiver in front of the others.
func BenchmarkChainMerge(b *testing.B) {
	first, second := build(1000), build(1000)

	b.Run("method", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.Of(first).Merge(second))
		}
	})

	b.Run("free function", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sink = len(xm.Merge(first, second))
		}
	})
}
