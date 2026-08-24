package log

import (
	"io"
	"log/slog"
	"testing"
)

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
