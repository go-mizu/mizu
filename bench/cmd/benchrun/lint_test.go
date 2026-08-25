package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkSource runs the syntax rules over a few lines of source. There is no
// type information, so the rule about ranging over a map is not exercised here;
// TestLintPassesOnTheRealModule is what runs the rules with types.
func checkSource(t *testing.T, src string) []problem {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "bench_test.go", "package p\n"+src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	c := &checker{fset: fset}
	c.file(f)
	return c.out
}

// rules returns the rule names the source broke, so a test can say which rule
// it expects without pinning the wording of the message.
func rules(t *testing.T, src string) []string {
	t.Helper()

	var out []string
	for _, p := range checkSource(t, src) {
		out = append(out, p.Rule)
	}
	return out
}

func TestLintAcceptsABenchmarkThatFollowsTheRules(t *testing.T) {
	if got := rules(t, `
func BenchmarkThing(b *testing.B) {
	in := build()
	b.ReportAllocs()
	for b.Loop() {
		_ = work(in)
	}
}
`); len(got) != 0 {
		t.Errorf("a benchmark that follows the rules was reported for %v", got)
	}
}

func TestLintRules(t *testing.T) {
	tests := map[string]struct {
		src  string
		want string
	}{
		"no ReportAllocs": {`
func BenchmarkThing(b *testing.B) {
	for b.Loop() {
		_ = work()
	}
}
`, "report-allocs"},

		"the old loop": {`
func BenchmarkThing(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = work()
	}
}
`, "b-loop"},

		"the clock inside the loop": {`
func BenchmarkThing(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = work(time.Now())
	}
}
`, "clock"},

		"time.Since inside the loop": {`
func BenchmarkThing(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = time.Since(start)
	}
}
`, "clock"},

		"the global random source": {`
func BenchmarkThing(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = work(rand.Intn(100))
	}
}
`, "rand-seed"},

		"the global source in the setup": {`
func BenchmarkThing(b *testing.B) {
	in := rand.Perm(1000)
	b.ReportAllocs()
	for b.Loop() {
		_ = work(in)
	}
}
`, "rand-seed"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := rules(t, tt.src)
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("the rules reported %v, want just %s", got, tt.want)
			}
		})
	}
}

// TestLintLeavesTheSetupAlone is the other side of the clock rule. Reading the
// clock before the loop is how a benchmark decides how much work to do, and
// there is nothing wrong with it.
func TestLintLeavesTheSetupAlone(t *testing.T) {
	if got := rules(t, `
func BenchmarkThing(b *testing.B) {
	start := time.Now()
	in := build(start)
	b.ReportAllocs()
	for b.Loop() {
		_ = work(in)
	}
}
`); len(got) != 0 {
		t.Errorf("reading the clock during setup was reported for %v", got)
	}
}

// TestLintAllowsASeededSource is what the rand rule is asking for rather than
// what it forbids.
func TestLintAllowsASeededSource(t *testing.T) {
	if got := rules(t, `
func BenchmarkThing(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	for b.Loop() {
		_ = work(r.IntN(100))
	}
}
`); len(got) != 0 {
		t.Errorf("a benchmark with its own seeded source was reported for %v", got)
	}
}

// TestLintLeavesAParentBenchmarkAlone covers BenchmarkBudget, which has no loop
// of its own and measures nothing, so it has no reason to report allocations.
func TestLintLeavesAParentBenchmarkAlone(t *testing.T) {
	if got := rules(t, `
func BenchmarkBudget(b *testing.B) {
	for _, id := range ids {
		b.Run(id, benchmarks[id])
	}
}
`); len(got) != 0 {
		t.Errorf("a benchmark that only calls b.Run was reported for %v", got)
	}
}

// TestLintReadsTheParameterName is why the rules take the name rather than
// assuming it is b. A benchmark whose parameter is called something else is
// unusual and it is not wrong.
func TestLintReadsTheParameterName(t *testing.T) {
	if got := rules(t, `
func BenchmarkThing(bench *testing.B) {
	for bench.Loop() {
		_ = work()
	}
}
`); len(got) != 1 || got[0] != "report-allocs" {
		t.Errorf("the rules reported %v for a benchmark whose parameter is not called b", got)
	}
}

// TestLintReadsAFunctionLiteral covers the shape the micro package registers,
// which is a function value rather than a declaration.
func TestLintReadsAFunctionLiteral(t *testing.T) {
	if got := rules(t, `
var thing = func(b *testing.B) {
	for b.Loop() {
		_ = work()
	}
}
`); len(got) != 1 || got[0] != "report-allocs" {
		t.Errorf("the rules reported %v for a benchmark written as a function value", got)
	}
}

// TestLintIgnoresATypeThatMentionsTestingB keeps the rules off a declaration
// with no body to check, such as the map the micro package registers into.
func TestLintIgnoresATypeThatMentionsTestingB(t *testing.T) {
	if got := rules(t, `
var benchmarks = map[string]func(*testing.B){}

func register(id string, fn func(*testing.B)) { benchmarks[id] = fn }
`); len(got) != 0 {
		t.Errorf("a declaration with no benchmark in it was reported for %v", got)
	}
}

func TestProblemReadsLikeACompilerError(t *testing.T) {
	p := problem{Pos: "micro/log_test.go:12:2", Rule: "clock", Msg: "reads the clock"}

	const want = "micro/log_test.go:12:2: reads the clock (clock)"
	if got := p.String(); got != want {
		t.Errorf("problem.String() = %q, want %q", got, want)
	}
}

func TestBenchParam(t *testing.T) {
	tests := map[string]struct {
		src  string
		want string
		ok   bool
	}{
		"the usual":           {"func f(b *testing.B) {}", "b", true},
		"another name":        {"func f(bench *testing.B) {}", "bench", true},
		"second parameter":    {"func f(name string, b *testing.B) {}", "b", true},
		"a test":              {"func f(t *testing.T) {}", "", false},
		"nothing at all":      {"func f() {}", "", false},
		"not a pointer":       {"func f(b testing.B) {}", "", false},
		"a type with no name": {"func f(func(*testing.B)) {}", "", false},
		"another package":     {"func f(b *other.B) {}", "", false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "x.go", "package p\n"+tt.src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			decl := f.Decls[0].(*ast.FuncDecl)

			got, ok := benchParam(decl.Type)
			if got != tt.want || ok != tt.ok {
				t.Errorf("benchParam = %q, %v, want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestRelativeShortensAPosition is about the failure being readable. A problem
// that names the whole path from the root of the disk is one nobody can scan a
// list of.
func TestRelativeShortensAPosition(t *testing.T) {
	root := filepath.FromSlash("/home/x/bench")

	if got := relative(root, filepath.Join(root, "micro", "log_test.go")+":9:3"); got != "micro/log_test.go:9:3" {
		t.Errorf("relative gave %q", got)
	}

	outside := filepath.FromSlash("/elsewhere/x.go") + ":1:1"
	if got := relative(root, outside); got != outside {
		t.Errorf("a position outside the module became %q, want it left alone", got)
	}
}

func TestLintCorpusFindsAFileNobodyWroteDown(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "testdata")
	if err := os.MkdirAll(filepath.Join(dir, "routes"), 0o777); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "| `listed.txt` | a benchmark | what it is |\n")
	write("listed.txt", "x\n")
	write("unlisted.json", "{}\n")
	write("routes/nested.txt", "x\n")

	got, err := lintCorpus(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("lintCorpus found %v, want the two files nobody listed", got)
	}
	for _, want := range []string{"unlisted.json", "routes/nested.txt"} {
		if !strings.Contains(got[0].Pos+got[1].Pos, want) {
			t.Errorf("lintCorpus did not report %s, it said %v", want, got)
		}
	}
}

func TestLintCorpusSaysWhenThereIsNoIndex(t *testing.T) {
	if _, err := lintCorpus(t.TempDir()); err == nil {
		t.Error("a testdata directory with no README passed")
	}
}

// TestLintPassesOnTheRealModule is the rules against the benchmarks that exist,
// with type information, which is the only place the rule about ranging over a
// map is exercised. It is also the test that fails when somebody writes a
// benchmark that breaks a rule, which is what the rules are for.
func TestLintPassesOnTheRealModule(t *testing.T) {
	if testing.Short() {
		t.Skip("loading the module takes a few seconds")
	}

	root, err := moduleRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := lint(root, &out); err != nil {
		t.Fatalf("lint: %v\n%s", err, out.String())
	}
}
