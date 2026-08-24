package log

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
)

// handlers is every handler the benchmarks run, including the two in the
// standard library, since the question a reader has is what these cost compared
// to what they already have.
var handlers = []struct {
	name string
	new  func(io.Writer, slog.Leveler) slog.Handler
}{
	{"console", func(w io.Writer, l slog.Leveler) slog.Handler {
		return NewConsoleHandler(w, ConsoleOptions{Level: l, Color: ColorNever})
	}},
	{"json", func(w io.Writer, l slog.Leveler) slog.Handler {
		return NewJSONHandler(w, JSONOptions{Level: l})
	}},
	{"slog/json", func(w io.Writer, l slog.Leveler) slog.Handler {
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: l})
	}},
	{"slog/text", func(w io.Writer, l slog.Leveler) slog.Handler {
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: l})
	}},
}

func benchmark(b *testing.B, f func(*testing.B, *slog.Logger)) {
	b.Helper()
	for _, h := range handlers {
		b.Run(h.name, func(b *testing.B) {
			log := slog.New(h.new(io.Discard, slog.LevelDebug))
			b.ReportAllocs()
			f(b, log)
		})
	}
}

// BenchmarkRecord is the shape of most log calls: a message and a few
// attributes, none of which need anything clever.
func BenchmarkRecord(b *testing.B) {
	ctx := context.Background()
	benchmark(b, func(b *testing.B, log *slog.Logger) {
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelInfo, "request",
				slog.String("method", "GET"),
				slog.String("path", "/posts"),
				slog.Int("status", 200),
				slog.Duration("dur", 4200*time.Microsecond),
			)
		}
	})
}

// BenchmarkArgs is the same record written the other way. The any arguments are
// boxed by the call itself, before any handler sees them, which is the cost
// LogAttrs avoids.
func BenchmarkArgs(b *testing.B) {
	benchmark(b, func(b *testing.B, log *slog.Logger) {
		for b.Loop() {
			log.Info("request", "method", "GET", "path", "/posts", "status", 200, "dur", 4200*time.Microsecond)
		}
	})
}

// BenchmarkWith is a logger carrying attributes, which is what a request scoped
// logger looks like. The attributes are formatted once by With and copied after
// that.
func BenchmarkWith(b *testing.B) {
	ctx := context.Background()
	benchmark(b, func(b *testing.B, log *slog.Logger) {
		log = log.With("service", "api", "region", "eu-west-1", "version", "1.4.0")
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
		}
	})
}

// BenchmarkContextData is the cost of reading the data middleware left on the
// context and putting it in every record.
func BenchmarkContextData(b *testing.B) {
	ctx := ctxdata.With(context.Background(), tenantID, "acme")
	ctx = ctxdata.With(ctx, apiKey, "sk_live_7f3a")
	benchmark(b, func(b *testing.B, log *slog.Logger) {
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
		}
	})
}

// BenchmarkError is a record carrying an error, which is expanded into its
// message, kind and code.
func BenchmarkError(b *testing.B) {
	ctx := context.Background()
	err := errs.New(errs.Unavailable, "mail.down", "dial tcp: connection refused")
	benchmark(b, func(b *testing.B, log *slog.Logger) {
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelError, "job failed",
				slog.String("job", "SendWelcome"),
				slog.Any("err", err),
			)
		}
	})
}

// BenchmarkDisabled is the call that writes nothing, which is the one that runs
// most often in production. It is the cost of a program leaving its debug
// logging in.
func BenchmarkDisabled(b *testing.B) {
	ctx := context.Background()
	for _, h := range handlers {
		b.Run(h.name, func(b *testing.B) {
			log := slog.New(h.new(io.Discard, slog.LevelInfo))
			b.ReportAllocs()
			for b.Loop() {
				log.LogAttrs(ctx, slog.LevelDebug, "cache lookup", slog.String("key", "post:12"))
			}
		})
	}
}

// BenchmarkMulti is what fanning out to two handlers costs over writing to one.
func BenchmarkMulti(b *testing.B) {
	ctx := context.Background()
	one := NewJSONHandler(io.Discard, JSONOptions{})
	two := NewMultiHandler(one, NewJSONHandler(io.Discard, JSONOptions{}))

	for _, c := range []struct {
		name string
		h    slog.Handler
	}{{"one", one}, {"two", two}} {
		b.Run(c.name, func(b *testing.B) {
			log := slog.New(c.h)
			b.ReportAllocs()
			for b.Loop() {
				log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
			}
		})
	}
}

// BenchmarkFilter is the cost of asking a question about every record.
func BenchmarkFilter(b *testing.B) {
	ctx := context.Background()
	h := NewFilterHandler(NewJSONHandler(io.Discard, JSONOptions{}), func(_ context.Context, r slog.Record) bool {
		return r.Message != "healthz"
	})
	log := slog.New(h)

	b.ReportAllocs()
	for b.Loop() {
		log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
	}
}

// BenchmarkSampling has the two paths that matter: the record that is written
// and the one that is counted and dropped, which is the one that runs when a
// program is in the state sampling is for.
func BenchmarkSampling(b *testing.B) {
	ctx := context.Background()

	b.Run("written", func(b *testing.B) {
		log := slog.New(NewSamplingHandler(NewJSONHandler(io.Discard, JSONOptions{}), SampleOptions{Every: 1}))
		b.ReportAllocs()
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
		}
	})

	b.Run("dropped", func(b *testing.B) {
		log := slog.New(NewSamplingHandler(NewJSONHandler(io.Discard, JSONOptions{}), SampleOptions{Initial: 1, Every: 1 << 30}))
		b.ReportAllocs()
		for b.Loop() {
			log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
		}
	})

	// The counters are sharded so that this is an atomic add rather than a
	// queue behind one lock.
	b.Run("dropped-parallel", func(b *testing.B) {
		log := slog.New(NewSamplingHandler(NewJSONHandler(io.Discard, JSONOptions{}), SampleOptions{Initial: 1, Every: 1 << 30}))
		b.ReportAllocs()
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				log.LogAttrs(ctx, slog.LevelInfo, "request", slog.Int("status", 200))
			}
		})
	})
}
