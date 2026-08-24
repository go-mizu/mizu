package log

import (
	"context"
	"errors"
	"log/slog"
)

// NewMultiHandler sends every record to all of the handlers it is given.
//
// It is the answer to wanting two of something: a console for the terminal and
// a file for later, or the application's own handler alongside one a driver
// installed.
//
//	h := log.NewMultiHandler(
//		log.NewConsoleHandler(os.Stderr, log.ConsoleOptions{}),
//		log.NewJSONHandler(f, log.JSONOptions{Level: slog.LevelWarn}),
//	)
//
// Each handler decides for itself whether it wants a record, so the two in that
// example write different levels to different places.
//
// Handlers are called in the order they are given, and one that fails does not
// stop the next. The error is every failure joined together, since a record
// that reached one of two places is neither a success nor a total loss.
//
// Given nothing, the result is [slog.DiscardHandler], which is what a program
// with no logging configured wants rather than a nil to check for. Given one
// handler, that handler is returned as it is.
func NewMultiHandler(hs ...slog.Handler) slog.Handler {
	switch len(hs) {
	case 0:
		return slog.DiscardHandler
	case 1:
		return hs[0]
	}
	return &multi{hs: hs}
}

// multi is a struct rather than a slice so that two handlers can be compared,
// which is what a caller checking for [slog.DiscardHandler] does.
type multi struct{ hs []slog.Handler }

func (m *multi) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.hs {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m *multi) Handle(ctx context.Context, r slog.Record) error {
	var err error
	for _, h := range m.hs {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		// Each handler gets its own record, since a handler is allowed to add
		// attributes to the one it is given and the next one would then see
		// them.
		if e := h.Handle(ctx, r.Clone()); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func (m *multi) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return m
	}
	return m.each(func(h slog.Handler) slog.Handler { return h.WithAttrs(attrs) })
}

func (m *multi) WithGroup(name string) slog.Handler {
	if name == "" {
		return m
	}
	return m.each(func(h slog.Handler) slog.Handler { return h.WithGroup(name) })
}

// each is a multi holding what f made of every handler in this one.
func (m *multi) each(f func(slog.Handler) slog.Handler) slog.Handler {
	next := make([]slog.Handler, len(m.hs))
	for i, h := range m.hs {
		next[i] = f(h)
	}
	return &multi{hs: next}
}
