package mizutest

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
)

// Log is what the application logged, kept rather than printed.
//
// A test that passes should say nothing and a test that fails should say
// everything, which is hard to arrange when the logger writes to stderr as it
// goes. So the fixture keeps the records and the failure output reads them
// back. It is also the assertion surface for logging itself: a handler that is
// meant to warn about something can be asked whether it did.
//
// A Log is safe for concurrent use, since a handler under test may log from
// more than one goroutine.
type Log struct {
	mu      sync.Mutex
	entries []Entry
}

// Entry is one record, flattened to what an assertion or a failure message
// wants to read.
type Entry struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

// Handler is this log as an slog.Handler, for pointing a logger at it.
//
//	lg := slog.New(app.Log().Handler())
//
// [NewApp] has already done this for the application under test. It is here for
// anything the test builds itself and wants recorded in the same place.
func (l *Log) Handler() slog.Handler { return &handler{log: l} }

// Entries is a copy of everything logged so far, oldest first.
func (l *Log) Entries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.entries)
}

// Errors is the entries at error level and above, which is what a failing test
// usually wants and usually all it wants.
func (l *Log) Errors() []Entry { return l.AtLeast(slog.LevelError) }

// AtLeast is the entries at a level or above.
func (l *Log) AtLeast(level slog.Level) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var out []Entry
	for _, e := range l.entries {
		if e.Level >= level {
			out = append(out, e)
		}
	}
	return out
}

// Reset drops everything logged so far, for a test that makes one request to
// set something up and then wants to assert about the next one alone.
func (l *Log) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

// String is the whole log, one entry per line, in the form the failure output
// uses.
func (l *Log) String() string {
	var b strings.Builder
	for _, e := range l.Entries() {
		b.WriteString(e.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func (l *Log) append(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, e)
}

// String is one line: the level, the message, and the attributes by name.
// Sorted, because a map has no order and a failure message that changes between
// runs is one nobody trusts.
func (e Entry) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", e.Level, e.Message)
	for _, k := range slices.Sorted(maps.Keys(e.Attrs)) {
		fmt.Fprintf(&b, " %s=%v", k, e.Attrs[k])
	}
	return b.String()
}

// handler is the slog.Handler side. It is separate from Log because With and
// WithGroup return a new handler and all of them have to record into the same
// place, which they cannot do if the handler is also the store.
//
// The attributes it carries are flattened when they arrive rather than when a
// record does, because the groups an attribute belongs to are the ones open at
// the time it was added. Holding them as slog.Attr and naming them later puts
// logger.With(a).WithGroup("g") under g, where slog leaves it alone.
type handler struct {
	log    *Log
	attrs  map[string]any
	prefix string
}

func (h *handler) Enabled(context.Context, slog.Level) bool { return true }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	e := Entry{
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make(map[string]any, len(h.attrs)+r.NumAttrs()),
	}
	maps.Copy(e.Attrs, h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		put(e.Attrs, h.prefix, a)
		return true
	})

	h.log.append(e)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := make(map[string]any, len(h.attrs)+len(attrs))
	maps.Copy(next, h.attrs)
	for _, a := range attrs {
		put(next, h.prefix, a)
	}
	return &handler{log: h.log, attrs: next, prefix: h.prefix}
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &handler{log: h.log, attrs: h.attrs, prefix: h.prefix + name + "."}
}

// put flattens an attribute into the map under the open groups, joining them
// with a dot the way slog's text handler does, so a grouped attribute has a
// name a test can write down.
func put(into map[string]any, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		// A group with no attributes contributes nothing, and one with no name
		// adds no level, which is what slog does with both.
		inner := prefix
		if a.Key != "" {
			inner = prefix + a.Key + "."
		}
		for _, g := range a.Value.Group() {
			put(into, inner, g)
		}
		return
	}
	into[prefix+a.Key] = a.Value.Any()
}
