package consoletest

import (
	"testing"

	"github.com/go-mizu/mizu/console"
)

// quiet is a testing.TB that costs nothing, so that the numbers below are the
// fixture and the command rather than the testing package.
type quiet struct{ testing.TB }

func (quiet) Helper() {}

// BenchmarkRun is a command with nothing to parse and nothing to answer, which
// is what the fixture costs before the command does anything.
func BenchmarkRun(b *testing.B) {
	tb := quiet{}
	b.ReportAllocs()
	for b.Loop() {
		Run(tb, &greet{Name: "Ada"})
	}
}

// BenchmarkRunWithArgs adds parsing a command line into the command.
func BenchmarkRunWithArgs(b *testing.B) {
	tb := quiet{}
	b.ReportAllocs()
	for b.Loop() {
		Run(tb, &greet{}, Args("-l", "Ada"))
	}
}

// BenchmarkPrompts is four questions answered, which is the part of this
// package that does the work: every answer reads the question back off the
// stream it was written to.
func BenchmarkPrompts(b *testing.B) {
	tb := quiet{}
	b.ReportAllocs()
	for b.Loop() {
		Run(tb, &setup{},
			Answer("Project name", "blog"),
			Choose("Database", "postgres"),
			ChooseAll("What else", "queue", "cache"),
			Confirm("Run go mod tidy", true))
	}
}

// BenchmarkAssertions is a result being asked about, which a test does more
// often than it runs a command.
func BenchmarkAssertions(b *testing.B) {
	tb := quiet{}
	r := Run(tb, &greet{Name: "Ada"})

	b.ReportAllocs()
	for b.Loop() {
		r.AssertSuccess()
		r.AssertExitCode(console.CodeOK)
		r.AssertOutputContains("Ada")
	}
}
