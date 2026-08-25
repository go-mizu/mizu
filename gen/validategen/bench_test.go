package validategen

import (
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// loadOnce keeps the go command and the type checker out of the numbers below.
// Loading a package is measured on its own in the gen package, and what
// matters here is what this generator adds to it.
func loadOnce(b *testing.B) []*gen.Package {
	b.Helper()
	pkgs, err := gen.Load(gen.Config{Dir: "testdata"}, "./app")
	if err != nil {
		b.Fatal(err)
	}
	return pkgs
}

// BenchmarkGenerate is the whole thing over the four marked structs in
// testdata, which between them reach nine struct types and every rule.
func BenchmarkGenerate(b *testing.B) {
	pkgs := loadOnce(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Generate(pkgs...); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAnalyze is the walk on its own, without the writing. It is where the
// tags are read and where the comparisons are worked out.
func BenchmarkAnalyze(b *testing.B) {
	pkgs := loadOnce(b)
	targets := markedIn(pkgs...)

	b.ReportAllocs()
	for b.Loop() {
		if p := Analyze(pkgs[0], targets); len(p.Structs) == 0 {
			b.Fatal("nothing to do")
		}
	}
}

// BenchmarkRender is the writing on its own, which is where the output size
// shows up.
func BenchmarkRender(b *testing.B) {
	pkgs := loadOnce(b)
	p := Analyze(pkgs[0], markedIn(pkgs...))

	b.ReportAllocs()
	for b.Loop() {
		if len(render(p)) == 0 {
			b.Fatal("nothing written")
		}
	}
}

func markedIn(pkgs ...*gen.Package) []gen.Target {
	all, _ := gen.Scan(pkgs...)
	var out []gen.Target
	for _, t := range all {
		if marked(t) {
			out = append(out, t)
		}
	}
	return out
}
