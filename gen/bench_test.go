package gen

import (
	"bytes"
	"go/token"
	"testing"
)

// There is no BenchmarkLoad. A full Load runs `go list -export`, which builds
// the module, so the number it produced would be a measurement of the Go
// toolchain and the state of the build cache. What is worth measuring is the
// part written here: parsing the files and type-checking them against export
// data. So these benchmarks share one listing and time the rest.
//
// The fixture is six small packages. The numbers are not a prediction for a
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

// BenchmarkScan walks the loaded packages for markers. It runs once per
// generation, over every file in the module, so it wants to stay well under
// the cost of the type-checking it follows.
func BenchmarkScan(b *testing.B) {
	pkgs := benchLoader(b, benchListing(b), nil).load()
	b.ReportAllocs()
	for b.Loop() {
		Scan(pkgs...)
	}
}

// BenchmarkParseMarker is one comment, which is the inner loop of the walk and
// the only part of it that does real string work.
func BenchmarkParseMarker(b *testing.B) {
	const text = `//mizu:command name="users:prune" desc="Delete users who never verified" standalone`
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parseMarker(text, token.Position{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseOrdinaryComment is the case that dominates any real package,
// where almost every comment is prose and the answer is no.
func BenchmarkParseOrdinaryComment(b *testing.B) {
	const text = "// Prune deletes drafts that nobody has touched in a year."
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parseMarker(text, token.Position{}); err != nil {
			b.Fatal(err)
		}
	}
}

// generatedFile is about what a small generator produces: a header, a package
// clause, and a handful of declarations.
var generatedFile = []byte(Header("columns", "1", "model/model.go") + `
package model

// UserTable is where a User is stored.
const UserTable = "users"

// UserColumns are its columns, in the order the fields are declared.
var UserColumns = []string{
	"id",
	"email",
	"created",
}
`)

// BenchmarkFormat is gofmt over one generated file, which is the most
// expensive thing the writer does per file and the reason a generator does not
// have to get its own whitespace right.
func BenchmarkFormat(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Format("model/columns_gen.go", generatedFile); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerated is the header check, which runs twice per file: once on
// what the generator produced and once on what is already on disk.
func BenchmarkGenerated(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if !Generated(generatedFile) {
			b.Fatal("no header")
		}
	}
}

// BenchmarkWriteUnchanged is the case that dominates a real run, where the
// file on disk already says what the generator was going to say. It is a
// format, a read, and a compare, with nothing written.
func BenchmarkWriteUnchanged(b *testing.B) {
	w := &Writer{Dir: b.TempDir()}
	f := File{Path: "model/columns_gen.go", Data: generatedFile}
	if _, err := w.Write(f); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		r, err := w.Write(f)
		if err != nil {
			b.Fatal(err)
		}
		if r[0].Changed() {
			b.Fatal("the file changed under the benchmark")
		}
	}
}

// BenchmarkWriteUpdated is the same with the write on the end: a temporary
// file, a chmod, and a rename. Two versions alternate so every iteration has
// something to do.
func BenchmarkWriteUpdated(b *testing.B) {
	w := &Writer{Dir: b.TempDir()}
	files := [2]File{
		{Path: "model/columns_gen.go", Data: generatedFile},
		{Path: "model/columns_gen.go", Data: append(bytes.Clone(generatedFile), "\nconst Extra = 1\n"...)},
	}
	if _, err := w.Write(files[0]); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		i++
		r, err := w.Write(files[i%2])
		if err != nil {
			b.Fatal(err)
		}
		if r[0].Status != Updated {
			b.Fatalf("status is %v", r[0].Status)
		}
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
