package ctxdata

import (
	"context"
	"log/slog"
)

// Mask is what a [Redacted] datum is logged as.
//
// It is a fixed string rather than a run of asterisks the length of the value,
// because the length of a secret is information about the secret.
const Mask = "[redacted]"

// Attrs is the logged data in ctx, as attributes, oldest first.
//
// A handler that adds these to every record is what [Logged] means, and this is
// also how to add them by hand to one record:
//
//	slog.LogAttrs(ctx, slog.LevelInfo, "rebuilding the index", ctxdata.Attrs(ctx)...)
//
// It is nil when nothing in the context is logged, so a context that carries
// only propagated data costs nothing.
func Attrs(ctx context.Context) []slog.Attr {
	entries := bag(ctx)
	n := 0
	for _, e := range entries {
		if e.datum.logged {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, n)
	for _, e := range entries {
		if e.datum.logged {
			attrs = append(attrs, attr(e))
		}
	}
	return attrs
}

// attr is one entry as an attribute, masked when the key said to.
func attr(e entry) slog.Attr {
	if e.datum.redacted {
		return slog.String(e.datum.name, Mask)
	}
	return slog.Any(e.datum.name, e.value)
}
