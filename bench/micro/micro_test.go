package micro

import (
	"maps"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/bench/budget"
)

// benchmarks is what BenchmarkBudget runs, keyed by budget ID. Each subsystem
// file adds its own, so a new benchmark is one file and not also an edit to a
// list kept somewhere else.
var benchmarks = map[string]func(*testing.B){}

// register adds one. Both of the ways it can be wrong are wrong at startup
// rather than in a run somebody has to read the output of: two benchmarks for
// one operation means the run reports the same name twice, and a benchmark with
// no row means a number nobody agreed to.
func register(id string, fn func(*testing.B)) {
	if _, dup := benchmarks[id]; dup {
		panic("micro: two benchmarks named " + id)
	}
	if _, ok := budget.Lookup(id); !ok {
		panic("micro: no budget row for " + id + ", add one to bench/budget")
	}
	benchmarks[id] = fn
}

// BenchmarkBudget runs every budgeted operation.
//
// The order is sorted rather than whatever the map hands back, so that two runs
// of the same code produce output that can be diffed line for line.
func BenchmarkBudget(b *testing.B) {
	for _, id := range slices.Sorted(maps.Keys(benchmarks)) {
		b.Run(id, benchmarks[id])
	}
}

// TestEveryMeasuredRowIsMeasured is the other half of what register checks. A
// row with no milestone against it is a promise that the operation is watched
// now, and this is what makes deleting the benchmark without deleting the
// promise fail.
//
// benchrun check does the same comparison against the names the toolchain
// actually reports, which catches the case where a benchmark is registered
// under one name and runs under another. This one is here because it costs
// nothing and runs on every go test.
func TestEveryMeasuredRowIsMeasured(t *testing.T) {
	for _, r := range budget.Rows() {
		if !r.Measured() {
			continue
		}
		if _, ok := benchmarks[r.ID]; !ok {
			t.Errorf("%s has no benchmark and no milestone that brings one", r.ID)
		}
	}
}
