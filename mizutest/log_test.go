package mizutest

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
)

// TestTheHandlerFollowsSlogsRules is the one test here that is not about mizu.
// slogtest is the standard suite every slog.Handler is meant to pass, and a
// handler that quietly drops an attribute is a fixture that says a warning was
// never logged when it was.
func TestTheHandlerFollowsSlogsRules(t *testing.T) {
	var log Log

	results := func() []map[string]any {
		var out []map[string]any
		for _, e := range log.Entries() {
			m := map[string]any{
				slog.MessageKey: e.Message,
				slog.LevelKey:   e.Level,
			}
			for k, v := range e.Attrs {
				nest(m, k, v)
			}
			out = append(out, m)
		}
		return out
	}

	// The time and the source are not recorded, because a fixture that keeps
	// them has a failure message that changes between runs for no reason.
	err := slogtest.TestHandler(log.Handler(), results)
	for _, err := range flatten(err) {
		if strings.Contains(err.Error(), slog.TimeKey) || strings.Contains(err.Error(), slog.SourceKey) {
			continue
		}
		t.Error(err)
	}
}

// nest puts a dotted name back into the nested shape slogtest reads, which is
// the shape slog itself hands a handler. The fixture flattens on the way in
// because a test asserting about a log wants to write down one name, and this
// undoes it for the suite. Splitting on a dot is safe here because slogtest
// does not use one in a key.
func nest(into map[string]any, name string, v any) {
	for {
		i := strings.IndexByte(name, '.')
		if i < 0 {
			into[name] = v
			return
		}
		group, ok := into[name[:i]].(map[string]any)
		if !ok {
			group = map[string]any{}
			into[name[:i]] = group
		}
		into, name = group, name[i+1:]
	}
}

func flatten(err error) []error {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return joined.Unwrap()
	}
	if err != nil {
		return []error{err}
	}
	return nil
}

func TestGroupsBecomeDottedNames(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	lg.With("service", "api").
		WithGroup("request").
		With("id", "req-1").
		WithGroup("route").
		Info("handled", "name", "posts.show", slog.Group("timing", "ms", 12))

	e := only(t, &log)
	want := map[string]any{
		"service":                 "api",
		"request.id":              "req-1",
		"request.route.name":      "posts.show",
		"request.route.timing.ms": int64(12),
	}
	for k, v := range want {
		if got := e.Attrs[k]; got != v {
			t.Errorf("%s = %v (%T), want %v (%T)", k, got, got, v, v)
		}
	}
	if len(e.Attrs) != len(want) {
		t.Errorf("recorded %v, want exactly %v", e.Attrs, want)
	}
}

// TestAttributesKeepTheGroupsTheyWereAddedUnder is the rule that is easy to get
// backwards. An attribute belongs to the groups open when it was added, not to
// the ones open when something is finally logged.
func TestAttributesKeepTheGroupsTheyWereAddedUnder(t *testing.T) {
	var log Log

	slog.New(log.Handler()).With("outside", 1).WithGroup("g").Info("hello", "inside", 2)

	e := only(t, &log)
	if _, ok := e.Attrs["outside"]; !ok {
		t.Errorf("an attribute added before the group was recorded as %v", e.Attrs)
	}
	if _, ok := e.Attrs["g.inside"]; !ok {
		t.Errorf("an attribute added inside the group was recorded as %v", e.Attrs)
	}
}

// TestEveryHandlerRecordsInTheSamePlace is why the store and the handler are
// separate types. With and WithGroup return new handlers, and a test asking the
// fixture what was logged has to see what they wrote.
func TestEveryHandlerRecordsInTheSamePlace(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	lg.Info("plain")
	lg.With("a", 1).Info("with attrs")
	lg.WithGroup("g").Info("with a group")
	lg.With("a", 1).WithGroup("g").With("b", 2).Info("with both")

	if got := len(log.Entries()); got != 4 {
		t.Errorf("the store holds %d entries, want 4:\n%s", got, &log)
	}
}

func TestEntriesAreACopy(t *testing.T) {
	var log Log
	slog.New(log.Handler()).Info("hello")

	entries := log.Entries()
	entries[0].Message = "changed"

	if got := log.Entries()[0].Message; got != "hello" {
		t.Errorf("the store holds %q, so a caller can edit it from underneath", got)
	}
}

func TestErrorsAndAtLeastSelectByLevel(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	lg.Debug("debug")
	lg.Info("info")
	lg.Warn("warn")
	lg.Error("error")

	if got := len(log.Entries()); got != 4 {
		t.Fatalf("recorded %d entries, want 4", got)
	}
	if got := log.Errors(); len(got) != 1 || got[0].Message != "error" {
		t.Errorf("Errors gave %v, want the one error", got)
	}
	if got := log.AtLeast(slog.LevelWarn); len(got) != 2 {
		t.Errorf("AtLeast(warn) gave %v, want the warning and the error", got)
	}
	if got := log.AtLeast(slog.LevelDebug); len(got) != 4 {
		t.Errorf("AtLeast(debug) gave %d entries, want all 4", len(got))
	}
}

func TestResetDropsWhatCameBefore(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	lg.Info("setting up")
	log.Reset()
	lg.Info("the part under test")

	e := only(t, &log)
	if e.Message != "the part under test" {
		t.Errorf("after Reset the log holds %q", e.Message)
	}
}

func TestEntryStringSortsItsAttributes(t *testing.T) {
	e := Entry{
		Level:   slog.LevelWarn,
		Message: "slow query",
		Attrs:   map[string]any{"ms": 120, "table": "posts", "attempt": 2},
	}

	const want = "WARN slow query attempt=2 ms=120 table=posts"
	if got := e.String(); got != want {
		t.Errorf("Entry.String() = %q, want %q", got, want)
	}

	if got := (Entry{Level: slog.LevelInfo, Message: "hello"}).String(); got != "INFO hello" {
		t.Errorf("an entry with no attributes reads as %q", got)
	}
}

func TestLogStringIsOneLinePerEntry(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	lg.Info("first")
	lg.Error("second", "why", "boom")

	const want = "INFO first\nERROR second why=boom\n"
	if got := log.String(); got != want {
		t.Errorf("Log.String() = %q, want %q", got, want)
	}
	if got := (&Log{}).String(); got != "" {
		t.Errorf("an empty log reads as %q, want nothing", got)
	}
}

// TestTheLogTakesConcurrentWrites is not theoretical. A handler under test can
// log from a goroutine it started, and a fixture that races there fails a test
// that has nothing wrong with it.
func TestTheLogTakesConcurrentWrites(t *testing.T) {
	var log Log
	lg := slog.New(log.Handler())

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 32 {
				lg.Info("hello", "worker", i)
			}
			log.Entries()
		}()
	}
	wg.Wait()

	if got := len(log.Entries()); got != 8*32 {
		t.Errorf("recorded %d entries, want %d", got, 8*32)
	}
}

// only returns the single entry in a log, and says so when there is not one.
func only(t *testing.T, log *Log) Entry {
	t.Helper()

	entries := log.Entries()
	if len(entries) != 1 {
		t.Fatalf("the log holds %d entries, want 1:\n%s", len(entries), log)
	}
	return entries[0]
}

// TestWithNothingReturnsTheSameHandler is what keeps a logger built up in
// layers from allocating a handler per layer for no reason.
func TestWithNothingReturnsTheSameHandler(t *testing.T) {
	var log Log
	h := log.Handler()

	if got := h.WithAttrs(nil); got != h {
		t.Error("WithAttrs of nothing built a new handler")
	}
	if got := h.WithGroup(""); got != h {
		t.Error("WithGroup of no name built a new handler")
	}
}
