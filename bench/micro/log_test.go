package micro

import (
	"io"
	"log/slog"
	"testing"

	"github.com/go-mizu/mizu/log"
)

func init() {
	register("log/info/off", benchLogInfoOff)
	register("log/info/json", benchLogInfoJSON)
}

// benchLogInfoOff is the call that does not happen. Most of the log lines in a
// running service are these, because the level is warn in production and the
// debug lines somebody left in the code still get evaluated as far as the level
// check. What one costs decides whether leaving them in is reasonable.
//
// The attributes are on the call rather than left off, because that is the call
// a handler makes and the variadic slice is part of what it costs.
func benchLogInfoOff(b *testing.B) {
	l := slog.New(log.NewJSONHandler(io.Discard, log.JSONOptions{Level: slog.LevelWarn}))

	b.ReportAllocs()
	for b.Loop() {
		l.Info("loaded the user", "id", 4211, "cached", true)
	}
}

// benchLogInfoJSON is the line that does get written. Six attributes is about
// what a request line carries, and the writer is discarded so that what is
// measured is the encoding rather than the disk.
func benchLogInfoJSON(b *testing.B) {
	l := slog.New(log.NewJSONHandler(io.Discard, log.JSONOptions{}))

	b.ReportAllocs()
	for b.Loop() {
		l.Info("handled",
			"method", "GET",
			"route", "/posts/{id}",
			"status", 200,
			"bytes", 4096,
			"ms", 3,
			"request_id", "01JQ8Z9F4K6W2N7TB3XR5CVDMH",
		)
	}
}
