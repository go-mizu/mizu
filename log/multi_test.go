package log

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

func TestMulti(t *testing.T) {
	a, b := new(recorder), new(recorder)
	log := slog.New(NewMultiHandler(a, b))

	log.Info("one")
	log.Warn("two")

	for name, rec := range map[string]*recorder{"a": a, "b": b} {
		if got := rec.messages(); !slices.Equal(got, []string{"one", "two"}) {
			t.Errorf("%s got %v", name, got)
		}
	}
}

// TestMultiNone is the program that configured no logging at all. It gets a
// handler that writes nothing rather than a nil to check for.
func TestMultiNone(t *testing.T) {
	h := NewMultiHandler()
	if h != slog.DiscardHandler {
		t.Errorf("NewMultiHandler() = %#v, want slog.DiscardHandler", h)
	}
	if h.Enabled(context.Background(), slog.LevelError) {
		t.Error("the handler that writes nothing is enabled")
	}
}

// TestMultiOne is the case worth not paying for: one handler needs no fan out.
func TestMultiOne(t *testing.T) {
	rec := new(recorder)
	if got := NewMultiHandler(rec); got != slog.Handler(rec) {
		t.Errorf("NewMultiHandler(h) = %#v, want the handler it was given", got)
	}
}

// TestMultiLevels is the reason for having two: a console reading everything
// and a file keeping the warnings.
func TestMultiLevels(t *testing.T) {
	all := &recorder{level: slog.LevelDebug}
	warn := &recorder{level: slog.LevelWarn}
	log := slog.New(NewMultiHandler(all, warn))

	log.Debug("quiet")
	log.Warn("loud")

	if got := all.messages(); !slices.Equal(got, []string{"quiet", "loud"}) {
		t.Errorf("the handler taking everything got %v", got)
	}
	if got := warn.messages(); !slices.Equal(got, []string{"loud"}) {
		t.Errorf("the handler taking warnings got %v", got)
	}
}

// TestMultiEnabled is what the logger asks before it makes a record at all. One
// handler wanting it is enough.
func TestMultiEnabled(t *testing.T) {
	h := NewMultiHandler(&recorder{level: slog.LevelError}, &recorder{level: slog.LevelWarn})
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelWarn) != true {
		t.Error("a level one handler wants is not enabled")
	}
	if h.Enabled(ctx, slog.LevelInfo) != false {
		t.Error("a level no handler wants is enabled")
	}
}

// TestMultiErrors is the property that makes fan out useful: the second handler
// runs even though the first one failed, and both failures are reported.
func TestMultiErrors(t *testing.T) {
	first := errors.New("the disk is full")
	second := errors.New("the socket is closed")
	a := &recorder{err: first}
	b := &recorder{err: second}

	err := NewMultiHandler(a, b).Handle(context.Background(), slog.Record{})
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Errorf("Handle = %v, want both failures", err)
	}
	if b.count() != 1 {
		t.Error("the first handler failing stopped the second")
	}

	if err := NewMultiHandler(new(recorder), new(recorder)).Handle(context.Background(), slog.Record{}); err != nil {
		t.Errorf("Handle of two handlers that worked = %v", err)
	}
}

// TestMultiRecordIsolation is why each handler gets its own record. A handler
// is allowed to add attributes to the one it is given, and the next handler
// must not see them.
func TestMultiRecordIsolation(t *testing.T) {
	rec := new(recorder)
	h := NewMultiHandler(handlerFunc(func(_ context.Context, r slog.Record) error {
		r.AddAttrs(slog.String("mine", "yes"))
		return nil
	}), rec)

	r := slog.NewRecord(at, slog.LevelInfo, "m", 0)
	r.AddAttrs(slog.Int("n", 1))
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	var keys []string
	rec.records[0].Attrs(func(a slog.Attr) bool {
		keys = append(keys, a.Key)
		return true
	})
	if !slices.Equal(keys, []string{"n"}) {
		t.Errorf("the second handler saw %v, want only the record's own attribute", keys)
	}
}

func TestMultiWithAttrs(t *testing.T) {
	a, b := new(recorder), new(recorder)
	h := NewMultiHandler(a, b)

	log := slog.New(h).With("service", "api").WithGroup("db")
	log.Info("query")

	for name, rec := range map[string]*recorder{"a": a, "b": b} {
		if len(rec.attrs) != 1 || rec.attrs[0].Key != "service" {
			t.Errorf("%s got attrs %v", name, rec.attrs)
		}
		if !slices.Equal(rec.groups, []string{"db"}) {
			t.Errorf("%s got groups %v", name, rec.groups)
		}
		if rec.count() != 1 {
			t.Errorf("%s got %d records", name, rec.count())
		}
	}

	if got := h.WithAttrs(nil); got != h {
		t.Error("WithAttrs(nil) made a new handler")
	}
	if got := h.WithGroup(""); got != h {
		t.Error("WithGroup(\"\") made a new handler")
	}
}

// TestMultiOutput is the pair someone actually writes: a line for the terminal
// and an object for whatever collects the logs.
func TestMultiOutput(t *testing.T) {
	var human, machine strings.Builder
	h := NewMultiHandler(
		NewConsoleHandler(&human, ConsoleOptions{Color: ColorNever}),
		NewJSONHandler(&machine, JSONOptions{}),
	)
	slog.New(&fixedTime{h}).Info("request", "status", 200)

	if want := "10:44:02.113 INF request                      status=200\n"; human.String() != want {
		t.Errorf("console got %q, want %q", human.String(), want)
	}
	want := `{"time":"2026-08-24T10:44:02.113Z","level":"INFO","msg":"request","status":200}` + "\n"
	if machine.String() != want {
		t.Errorf("json got %s, want %s", machine.String(), want)
	}
}

// handlerFunc is a handler that is only a Handle, for the cases where the rest
// of the interface is not what is being tested.
type handlerFunc func(context.Context, slog.Record) error

func (f handlerFunc) Enabled(context.Context, slog.Level) bool { return true }
func (f handlerFunc) Handle(ctx context.Context, r slog.Record) error {
	return f(ctx, r)
}
func (f handlerFunc) WithAttrs([]slog.Attr) slog.Handler { return f }
func (f handlerFunc) WithGroup(string) slog.Handler      { return f }
