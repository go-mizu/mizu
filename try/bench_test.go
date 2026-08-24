package try_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/try"
)

// The budget is the path where nothing goes wrong. Wrapping a call in a retry
// should not be a decision anybody has to think about, and it is only that if
// the successful call costs nothing.

func BenchmarkDoSucceeds(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		err = try.Do(ctx, fn)
	}
}

// BenchmarkCallDirectly is what the line would have been without the retry, so
// the difference between the two is what the wrapper costs.
func BenchmarkCallDirectly(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		err = fn(ctx)
	}
}

// BenchmarkDoSucceedsWithOptions is the same call with the options a real one
// carries, since they are applied on every call rather than built once.
func BenchmarkDoSucceedsWithOptions(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }
	backoff := try.Jitter(try.Exponential(100*time.Millisecond, 30*time.Second))

	b.ReportAllocs()
	for b.Loop() {
		err = try.Do(ctx, fn, try.Attempts(5), backoff, try.Budget(30*time.Second))
	}
}

func BenchmarkValueSucceeds(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) (string, error) { return "ok", nil }

	b.ReportAllocs()
	for b.Loop() {
		out, err = try.Value(ctx, fn)
	}
}

// BenchmarkDoGivesUp is the failing path, on a fake clock so the waiting costs
// nothing and what is left is the retry machinery itself.
func BenchmarkDoGivesUp(b *testing.B) {
	ctx := clock.With(context.Background(), clock.Fake(time.Unix(0, 0)))
	down := errs.New(errs.Unavailable, "", "the upstream is down")
	fn := func(context.Context) error { return down }
	nowait := try.Backoff(func(int) time.Duration { return 0 })

	b.ReportAllocs()
	for b.Loop() {
		err = try.Do(ctx, fn, nowait, try.Attempts(3))
	}
}

// BenchmarkDoDoesNotRetry is the other failing path, where the error says there
// is no point. It is the one a request handler hits on a bad request, so it
// should cost about what one failed call costs.
func BenchmarkDoDoesNotRetry(b *testing.B) {
	ctx := context.Background()
	missing := errs.New(errs.NotFound, "", "no such user")
	fn := func(context.Context) error { return missing }

	b.ReportAllocs()
	for b.Loop() {
		err = try.Do(ctx, fn)
	}
}

func BenchmarkExponential(b *testing.B) {
	backoff := try.Exponential(100*time.Millisecond, 30*time.Second)

	b.ReportAllocs()
	for b.Loop() {
		wait = backoff(4)
	}
}

func BenchmarkJitter(b *testing.B) {
	backoff := try.Jitter(try.Exponential(100*time.Millisecond, 30*time.Second))

	b.ReportAllocs()
	for b.Loop() {
		wait = backoff(4)
	}
}

var (
	err  error
	out  string
	wait time.Duration
)
