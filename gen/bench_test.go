package gen

import "testing"

// There is no BenchmarkLoad. A full Load runs `go list -export`, which builds
// the module, so the number it produced would be a measurement of the Go
// toolchain and the state of the build cache. What is worth measuring is the
// part written here: parsing the files and type-checking them against export
// data. So these benchmarks share one listing and time the rest.
//
// The fixture is four small packages. The numbers are not a prediction for a
// real module, they are a baseline to notice a change against.

func benchListing(b *testing.B) []*listed {
	b.Helper()
	list, err := listing()
	if err != nil {
		b.Fatalf("go list: %v", err)
	}
	return list
}

func benchLoader(b *testing.B, list []*listed, overlay map[string][]byte) *loader {
	b.Helper()
	l, err := newLoader(Config{Dir: fixture, Overlay: overlay}, list)
	if err != nil {
		b.Fatal(err)
	}
	return l
}

// BenchmarkCheck is everything Load does after the go command returns: parse
// every file, type-check every root, decode the export data each one reaches.
// A fresh loader per iteration, so each one pays for the export data the same
// way the first load in a process does.
func BenchmarkCheck(b *testing.B) {
	list := benchListing(b)
	b.ReportAllocs()
	for b.Loop() {
		benchLoader(b, list, nil).load()
	}
}

// BenchmarkCheckWithOverlay is the same work with a generated file replaced by
// a stub. It comes out faster than BenchmarkCheck rather than slower, because
// the stub is a package clause and the file it replaced was not, so the two
// are not measuring the same amount of Go. What the pair does say is that the
// overlay itself costs nothing worth finding: a map lookup per file, and no
// reading from disk for the files it covers.
func BenchmarkCheckWithOverlay(b *testing.B) {
	list := benchListing(b)
	overlay := map[string][]byte{"bootstrap/bootstrap_gen.go": []byte("package bootstrap\n")}
	b.ReportAllocs()
	for b.Loop() {
		benchLoader(b, list, overlay).load()
	}
}

// BenchmarkOrder is the topological sort on its own. It runs once per load and
// should stay far too cheap to think about.
func BenchmarkOrder(b *testing.B) {
	l := benchLoader(b, benchListing(b), nil)
	b.ReportAllocs()
	for b.Loop() {
		l.order()
	}
}

// BenchmarkNormalizeOverlay covers the path a caller retrying the bootstrap
// case takes, where the overlay is rebuilt for every attempt.
func BenchmarkNormalizeOverlay(b *testing.B) {
	in := map[string][]byte{
		"model/model_gen.go":     []byte("package model\n"),
		"store/store_gen.go":     []byte("package store\n"),
		"bootstrap/bootstrap.go": []byte("package bootstrap\n"),
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := normalizeOverlay(fixture, in); err != nil {
			b.Fatal(err)
		}
	}
}
