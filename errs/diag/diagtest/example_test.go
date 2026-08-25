package diagtest_test

import (
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/errs/diag/diagtest"
)

// The examples here are compiled and shown in the documentation rather than
// run. An assertion needs a test to fail, and an example has none, so the
// *testing.T in each of them is the one the surrounding test was handed.

// A package with a corpus runs it with one test, and every directory under
// testdata/diag becomes a subtest named after itself.
func ExampleRun() {
	var t *testing.T

	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return load(c.Path("app.toml"))
	})
}

// The function is handed the case rather than a path, so it asks for the input
// files it wants by name and the corpus can move without it changing.
func ExampleCase() {
	var t *testing.T

	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return parse(c.Read(tb, "input.toml"))
	})
}

// Lines is for the entries whose input is a command line, one token per line,
// which is what a corpus for a command looks like.
func ExampleCase_Lines() {
	var t *testing.T

	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return run(c.Lines(tb, "args"))
	})
}

// Check is what [Run] holds every report to, and it is exported so a package
// that produces diagnostics somewhere other than a corpus can be held to the
// same rules in an ordinary test.
func ExampleCheck() {
	var t *testing.T

	diagtest.Check(t, diag.List{{
		Code:    "MZ1042",
		Message: `unknown config key "database.pool_size"`,
		File:    "config/app.toml",
	}})
}

func load(path string) error  { return nil }
func parse(src []byte) error  { return nil }
func run(args []string) error { return nil }
