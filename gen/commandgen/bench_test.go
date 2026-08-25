package commandgen

import (
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// loadOnce keeps the go command and the type checker out of the numbers below.
// Loading a package is measured on its own in the gen package, and what matters
// here is what this generator adds to it.
func loadOnce(b *testing.B) []*gen.Package {
	b.Helper()
	pkgs, err := gen.Load(gen.Config{Dir: "testdata"}, "./app")
	if err != nil {
		b.Fatal(err)
	}
	return pkgs
}

// BenchmarkGenerate is the whole thing over the four commands in testdata,
// which between them have more flags than most commands do.
func BenchmarkGenerate(b *testing.B) {
	pkgs := loadOnce(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Generate(pkgs...); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAnalyze is the walk on its own, without the writing.
func BenchmarkAnalyze(b *testing.B) {
	pkgs := loadOnce(b)
	targets, _ := gen.Scan(pkgs...)

	b.ReportAllocs()
	for b.Loop() {
		if p := Analyze(pkgs[0], targets); len(p.Commands) == 0 {
			b.Fatal("nothing to do")
		}
	}
}

// BenchmarkRender is the writing on its own, which is where the output size
// shows up.
func BenchmarkRender(b *testing.B) {
	pkgs := loadOnce(b)
	targets, _ := gen.Scan(pkgs...)
	p := Analyze(pkgs[0], targets)

	b.ReportAllocs()
	for b.Loop() {
		if len(render(p)) == 0 {
			b.Fatal("nothing came out")
		}
	}
}

// BenchmarkFieldDocs is the walk over the syntax that pairs a comment with the
// declaration it is about, which runs once per package.
func BenchmarkFieldDocs(b *testing.B) {
	pkgs := loadOnce(b)
	b.ReportAllocs()
	for b.Loop() {
		if len(fieldDocs(pkgs[0])) == 0 {
			b.Fatal("no docs")
		}
	}
}
