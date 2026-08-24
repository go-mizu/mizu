package log

import (
	"context"
	"log/slog"
	"slices"
	"testing"
)

func TestFilter(t *testing.T) {
	rec := new(recorder)
	log := slog.New(NewFilterHandler(rec, func(_ context.Context, r slog.Record) bool {
		return r.Message != "healthz"
	}))

	log.Info("healthz")
	log.Info("request")
	log.Error("healthz")

	if got := rec.messages(); !slices.Equal(got, []string{"request"}) {
		t.Errorf("the handler got %v, want only the request", got)
	}
}

// TestFilterContext is what a filter is for that a level cannot do: the answer
// is in the context rather than in the record.
func TestFilterContext(t *testing.T) {
	rec := new(recorder)
	h := NewFilterHandler(rec, func(ctx context.Context, _ slog.Record) bool {
		n, ok := attemptOf(ctx)
		return !ok || n > 1
	})
	log := slog.New(h)

	log.InfoContext(withAttempt(context.Background(), 1), "retrying")
	log.InfoContext(withAttempt(context.Background(), 2), "retrying")
	log.Info("no context at all")

	if got := rec.count(); got != 2 {
		t.Errorf("the handler got %d records, want 2", got)
	}
}

// TestFilterEnabled is the level question, which belongs to the handler being
// wrapped. A filter that answered it itself would stop the wrapped handler ever
// seeing a record it wanted.
func TestFilterEnabled(t *testing.T) {
	h := NewFilterHandler(&recorder{level: slog.LevelWarn}, keepAll)
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelInfo) {
		t.Error("a level the wrapped handler does not want is enabled")
	}
	if !h.Enabled(ctx, slog.LevelError) {
		t.Error("a level the wrapped handler wants is not enabled")
	}
}

func TestFilterWithAttrs(t *testing.T) {
	rec := new(recorder)
	h := NewFilterHandler(rec, func(_ context.Context, r slog.Record) bool {
		return r.Message == "kept"
	})

	log := slog.New(h).With("service", "api").WithGroup("db")
	log.Info("kept")
	log.Info("dropped")

	if rec.count() != 1 {
		t.Errorf("the handler got %d records, want 1", rec.count())
	}
	if len(rec.attrs) != 1 || rec.attrs[0].Key != "service" {
		t.Errorf("the wrapped handler got attrs %v", rec.attrs)
	}
	if !slices.Equal(rec.groups, []string{"db"}) {
		t.Errorf("the wrapped handler got groups %v", rec.groups)
	}

	if got := h.WithAttrs(nil); got != h {
		t.Error("WithAttrs(nil) made a new handler")
	}
	if got := h.WithGroup(""); got != h {
		t.Error("WithGroup(\"\") made a new handler")
	}
}

// TestFilterNeedsAFunction is the mistake worth catching at startup rather than
// at the first record.
func TestFilterNeedsAFunction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a filter with no function did not panic")
		}
	}()
	NewFilterHandler(new(recorder), nil)
}

func keepAll(context.Context, slog.Record) bool { return true }

type attemptKey struct{}

func withAttempt(ctx context.Context, n int) context.Context {
	return context.WithValue(ctx, attemptKey{}, n)
}

func attemptOf(ctx context.Context) (int, bool) {
	n, ok := ctx.Value(attemptKey{}).(int)
	return n, ok
}
