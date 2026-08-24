package archtest

import "testing"

// Only the queries are benchmarked. Load spends its time in the go command,
// so timing it would report the toolchain rather than this code, and running
// it a few hundred times in a row is a good way to make a CI job slow for no
// answer. The queries run against a graph that is already in memory, and
// those are the numbers worth watching, because a lint rule that walks the
// graph once per package has to stay cheap as the toolkit grows.

func benchGraph(b *testing.B) *Graph {
	b.Helper()
	g, err := Load(fixture, "./...")
	if err != nil {
		b.Fatalf("Load: %v", err)
	}
	return g
}

func BenchmarkDepsOf(b *testing.B) {
	g := benchGraph(b)
	for b.Loop() {
		g.DepsOf("mizu.test/graph/app")
	}
}

func BenchmarkChainHit(b *testing.B) {
	g := benchGraph(b)
	for b.Loop() {
		g.Chain("mizu.test/graph/wire", "net/http")
	}
}

func BenchmarkChainMiss(b *testing.B) {
	// The miss walks the whole reachable set before giving up, so it is the
	// worst case and the one that bounds a run over every package.
	g := benchGraph(b)
	for b.Loop() {
		g.Chain("mizu.test/graph/app", "mizu.test/graph/wire")
	}
}

func BenchmarkAllowOnly(b *testing.B) {
	g := benchGraph(b)
	for b.Loop() {
		g.AllowOnly("std")
	}
}

func BenchmarkForbid(b *testing.B) {
	g := benchGraph(b)
	for b.Loop() {
		g.Forbid("mizu.test/graph/...", "mizu.test/graph/wire")
	}
}
