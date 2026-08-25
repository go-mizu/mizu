package golden_test

import (
	"regexp"
	"testing"

	"github.com/go-mizu/mizu/golden"
)

// The examples here are compiled and shown in the documentation rather than
// run. An assertion needs a test to fail, and an example has none, so the
// *testing.T in each of them is the one the surrounding test was handed.

func ExampleAssert() {
	var t *testing.T

	// testdata/TestRender.golden, written by go test ./... -update.
	golden.Assert(t, render())
}

func ExampleAssertString() {
	var t *testing.T

	golden.AssertString(t, string(render()))
}

// A test that asserts more than one thing needs a name per file, and a slash in
// the name is a directory.
func ExampleName() {
	var t *testing.T

	files := generate()
	for name, content := range files {
		golden.Assert(t, content, golden.Name("TestGenerate/"+name))
	}
}

// AssertJSON re-marshals both sides, so a member moving in a struct definition
// does not rewrite every golden file in the package.
func ExampleAssertJSON() {
	var t *testing.T

	golden.AssertJSON(t, map[string]any{"name": "ana", "id": 1})
}

// AssertSQL collapses the whitespace between tokens, so reindenting a query
// builder does not show up as a difference.
func ExampleAssertSQL() {
	var t *testing.T

	golden.AssertSQL(t, "select id, name\n  from users\n where id = ?")
}

// A value that changes on every run has to be replaced with something fixed
// before comparing, or the file never matches twice.
func ExampleScrub() {
	var t *testing.T

	port := regexp.MustCompile(`:\d{4,5}\b`)
	golden.AssertString(t, serverLog(), golden.Scrub(port, ":<port>"))
}

// The built-in scrubbers cover what usually leaks into a response body.
func ExampleScrubUUIDs() {
	var t *testing.T

	golden.AssertJSON(t, response(), golden.ScrubUUIDs(), golden.ScrubTimes())
}

func render() []byte              { return nil }
func generate() map[string][]byte { return nil }
func serverLog() string           { return "" }
func response() any               { return nil }
