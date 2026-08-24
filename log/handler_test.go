package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
)

// recorder is the handler the composing handlers are tested against: it keeps
// what it was given and it cannot fail unless it is told to.
//
// WithAttrs and WithGroup return the recorder itself rather than a copy, so
// that everything lands in one place for a test to look at. A real handler
// does not do that, and none of the handlers under test can tell.
type recorder struct {
	mu      sync.Mutex
	level   slog.Level
	err     error
	records []slog.Record
	attrs   []slog.Attr
	groups  []string
}

func (rec *recorder) Enabled(_ context.Context, l slog.Level) bool {
	return l >= rec.level
}

func (rec *recorder) Handle(_ context.Context, r slog.Record) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.records = append(rec.records, r)
	return rec.err
}

func (rec *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.attrs = append(rec.attrs, attrs...)
	return rec
}

func (rec *recorder) WithGroup(name string) slog.Handler {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.groups = append(rec.groups, name)
	return rec
}

func (rec *recorder) count() int {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return len(rec.records)
}

// messages is what it was asked to write, in order.
func (rec *recorder) messages() []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	out := make([]string, len(rec.records))
	for i, r := range rec.records {
		out[i] = r.Message
	}
	return out
}

// TestWithNothingReturnsTheSameHandler is the case log/slog itself never sends,
// since Logger.With and Logger.WithGroup both filter it out. Anything that
// wraps a handler directly can still send it, and the answer is that a handler
// with nothing added to it is the handler it came from.
func TestWithNothingReturnsTheSameHandler(t *testing.T) {
	handlers := map[string]slog.Handler{
		"console": NewConsoleHandler(io.Discard, ConsoleOptions{}),
		"json":    NewJSONHandler(io.Discard, JSONOptions{}),
	}
	for name, h := range handlers {
		if got := h.WithAttrs(nil); got != h {
			t.Errorf("%s: WithAttrs(nil) made a new handler", name)
		}
		if got := h.WithGroup(""); got != h {
			t.Errorf("%s: WithGroup(\"\") made a new handler", name)
		}
	}
}

// TestWithAnEmptyGroupOnTheHandler is the other thing log/slog does not filter.
// A group with nothing in it writes nothing, whether it arrives with the record
// or was added to the handler earlier.
func TestWithAnEmptyGroupOnTheHandler(t *testing.T) {
	empty := []slog.Attr{{}, slog.Group("nothing")}

	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.With(anys(empty)...).Info("m", "n", 1)
	})
	if want := "10:44:02.113 INF m                            n=1\n"; got != want {
		t.Errorf("console:\ngot  %q\nwant %q", got, want)
	}

	got = jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.With(anys(empty)...).Info("m", "n", 1)
	})
	want := `{"time":"2026-08-24T10:44:02.113Z","level":"INFO","msg":"m","n":1}`
	if got != want {
		t.Errorf("json:\ngot  %s\nwant %s", got, want)
	}
}

func anys(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}
