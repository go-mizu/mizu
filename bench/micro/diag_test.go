package micro

import (
	"io"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

func init() {
	register("diag/text", benchDiagText)
	register("diag/json", benchDiagJSON)
}

// The list every renderer benchmark works from, built once because a benchmark
// that builds its own input is measuring the input.
var reported = diag.List{{
	Code:    "MZ1042",
	Message: `unknown config key "database.pool_size"`,
	File:    "config/app.toml",
	Range:   diag.Span(3, 1, 9),
	Detail:  "no such field in Config.Database",
	Suggestions: []diag.Suggestion{{
		Message:    `did you mean "max_open_conns"?`,
		Confidence: diag.High,
	}},
	Fix: "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
}}

// source stands in for the file on disk. Reading it is the caller's job and not
// part of what is budgeted here, so it comes back from memory.
func source(string) ([]byte, error) {
	return []byte("[database]\nurl = \"postgres://localhost/blog\"\npool_size = 25\n"), nil
}

// benchDiagText is what a command does on the way out when a person is reading.
// It is on the failure path, which stops being cold the moment a project has a
// hundred things wrong with it and somebody is fixing them one at a time.
func benchDiagText(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if err := diag.Text(io.Discard, reported, diag.WithSource(source)); err != nil {
			b.Fatal(err)
		}
	}
}

// benchDiagJSON is the same value on the way to an editor or an agent, which is
// the reader that gets the whole list rather than the first three of each kind.
func benchDiagJSON(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if err := diag.JSON(io.Discard, reported); err != nil {
			b.Fatal(err)
		}
	}
}
