package micro

import (
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func init() {
	register("xs/map/1000", benchXsMap1000)
}

// benchXsMap1000 is the sequence helper that gets used the most, over a slice
// big enough that the per-element cost is what shows.
//
// xs.Map is lazy, so mapping on its own costs one closure and measures nothing.
// The collect is what makes the elements happen, and it is also what the caller
// writes, since a mapped sequence is nearly always read straight into a slice
// to hand to a template or a response.
func benchXsMap1000(b *testing.B) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i
	}

	b.ReportAllocs()
	for b.Loop() {
		out := slices.Collect(xs.Map(slices.Values(in), double))
		_ = out
	}
}

func double(n int) int { return n * 2 }
