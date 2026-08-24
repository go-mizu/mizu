package log

import (
	"context"
	"log/slog"
)

// NewFilterHandler passes a record to h when keep says so, and drops it
// otherwise.
//
// Level is not what this is for, since every handler here takes a Level of its
// own. This is for the records a level cannot describe: the health check that
// runs every second, one noisy package, or everything below warn unless the
// request is the one being traced.
//
//	h := log.NewFilterHandler(h, func(ctx context.Context, r slog.Record) bool {
//		path, _ := ctxdata.Get(ctx, RequestPath)
//		return path != "/healthz"
//	})
//
// keep is called for every record that passes the level, so it is on the hot
// path and should not do anything a request would wait for. It sees the record
// as it was made, without the attributes the logger carries from
// [slog.Logger.With], since those belong to the handler rather than to the
// record. What it does have is the context, which is where the interesting
// answers are.
//
// It panics if keep is nil, since a filter that cannot decide is a mistake in
// the program rather than a policy of writing everything.
func NewFilterHandler(h slog.Handler, keep func(context.Context, slog.Record) bool) slog.Handler {
	if keep == nil {
		panic("log: NewFilterHandler needs a keep function")
	}
	return &filter{h: h, keep: keep}
}

type filter struct {
	h    slog.Handler
	keep func(context.Context, slog.Record) bool
}

func (f *filter) Enabled(ctx context.Context, l slog.Level) bool {
	return f.h.Enabled(ctx, l)
}

func (f *filter) Handle(ctx context.Context, r slog.Record) error {
	if !f.keep(ctx, r) {
		return nil
	}
	return f.h.Handle(ctx, r)
}

func (f *filter) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return f
	}
	return &filter{h: f.h.WithAttrs(attrs), keep: f.keep}
}

func (f *filter) WithGroup(name string) slog.Handler {
	if name == "" {
		return f
	}
	return &filter{h: f.h.WithGroup(name), keep: f.keep}
}
