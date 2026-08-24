package ctxdata

import (
	"context"
	"log/slog"
	"testing"
)

// The budget in doc 06 is two allocations for With and one per record for the
// attributes of up to eight logged keys.

func BenchmarkWith(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		sink = With(ctx, tenantID, "acme")
	}
}

// BenchmarkWithDeep is the shape a request has by the time it reaches a
// handler, where the context already carries four data.
func BenchmarkWithDeep(b *testing.B) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")
	b.ReportAllocs()
	for b.Loop() {
		sink = With(ctx, attempt, 3)
	}
}

func BenchmarkWithReplace(b *testing.B) {
	ctx := With(context.Background(), tenantID, "acme")
	b.ReportAllocs()
	for b.Loop() {
		sink = With(ctx, tenantID, "globex")
	}
}

// BenchmarkContextWithValue is the baseline the two above are measured
// against, since a datum here is a context value with a type on it.
func BenchmarkContextWithValue(b *testing.B) {
	type key struct{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		sink = context.WithValue(ctx, key{}, "acme")
	}
}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")
	ctx = With(ctx, attempt, 3)
	b.ReportAllocs()
	for b.Loop() {
		tenant, _ = Get(ctx, tenantID)
	}
}

// BenchmarkGetThroughLayers is the one that matters, since a handler reads a
// datum through whatever the server, the router and the middleware wrapped the
// context in. All of them are one Value lookup away.
func BenchmarkGetThroughLayers(b *testing.B) {
	ctx := With(context.Background(), tenantID, "acme")
	for range 8 {
		ctx = context.WithValue(ctx, struct{ n int }{}, 1)
	}
	b.ReportAllocs()
	for b.Loop() {
		tenant, _ = Get(ctx, tenantID)
	}
}

func BenchmarkAttrs(b *testing.B) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")
	ctx = With(ctx, attempt, 3)
	b.ReportAllocs()
	for b.Loop() {
		attrs = Attrs(ctx)
	}
}

// BenchmarkAll is what a handler does instead when it has a buffer of its own.
func BenchmarkAll(b *testing.B) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")
	ctx = With(ctx, attempt, 3)
	b.ReportAllocs()
	for b.Loop() {
		n := 0
		for e := range All(ctx) {
			if e.Logged {
				n++
			}
		}
		count = n
	}
}

var (
	sink   context.Context
	tenant string
	attrs  []slog.Attr
	count  int
)
