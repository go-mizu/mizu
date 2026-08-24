package clock_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/go-mizu/mizu/clock"
)

// The claim in the package doc is that reading the clock out of a context costs
// about the same as calling time.Now, so BenchmarkTimeNow is the number every
// other one here is read against.

func BenchmarkTimeNow(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		now = time.Now()
	}
}

func BenchmarkNow(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		now = clock.Now(ctx)
	}
}

// BenchmarkNowThroughLayers is the shape a request has by the time it reaches a
// handler, where the server, the router and the middleware have each wrapped
// the context. A Value lookup walks that chain, so this is the one that says
// whether the depth matters.
func BenchmarkNowThroughLayers(b *testing.B) {
	ctx := context.Background()
	for range 8 {
		ctx = context.WithValue(ctx, struct{ n int }{}, 1)
	}
	b.ReportAllocs()
	for b.Loop() {
		now = clock.Now(ctx)
	}
}

// BenchmarkNowWithClock is a context somebody put a clock in, which is the hit
// on the Value lookup rather than the walk to the bottom of the chain.
func BenchmarkNowWithClock(b *testing.B) {
	ctx := clock.With(context.Background(), clock.Real())
	b.ReportAllocs()
	for b.Loop() {
		now = clock.Now(ctx)
	}
}

// BenchmarkNowFromHeld is what a hot loop does instead: read the clock once and
// keep it. It is the floor the three above are paying the lookup on top of.
func BenchmarkNowFromHeld(b *testing.B) {
	c := clock.From(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		now = c.Now()
	}
}

func BenchmarkWith(b *testing.B) {
	ctx := context.Background()
	c := clock.Real()
	b.ReportAllocs()
	for b.Loop() {
		sink = clock.With(ctx, c)
	}
}

// BenchmarkFakeNow matters because a test suite reads a fake clock as often as
// production reads the real one, and this one takes a mutex to do it.
func BenchmarkFakeNow(b *testing.B) {
	ctx := clock.With(context.Background(), clock.Fake(time.Unix(0, 0)))
	b.ReportAllocs()
	for b.Loop() {
		now = clock.Now(ctx)
	}
}

func BenchmarkFakeTimer(b *testing.B) {
	c := clock.Fake(time.Unix(0, 0))
	b.ReportAllocs()
	for b.Loop() {
		t := c.NewTimer(time.Hour)
		t.Stop()
	}
}

// BenchmarkFakeAdvance is a clock with timers on it that the advance does not
// reach, which is the common case in a test: a scheduler sets a hundred timers
// and the test moves past one of them.
func BenchmarkFakeAdvance(b *testing.B) {
	for _, waiters := range []int{0, 10, 100} {
		b.Run(strconv.Itoa(waiters)+" timers", func(b *testing.B) {
			c := clock.Fake(time.Unix(0, 0))
			for range waiters {
				defer c.NewTimer(never).Stop()
			}
			b.ReportAllocs()
			for b.Loop() {
				c.Advance(time.Nanosecond)
			}
		})
	}
}

// BenchmarkFakeAdvancePastATicker is the path that skips whole periods rather
// than counting them out, so the hour and the nanosecond should cost the same.
func BenchmarkFakeAdvancePastATicker(b *testing.B) {
	for _, d := range []time.Duration{time.Nanosecond, time.Hour} {
		b.Run(d.String(), func(b *testing.B) {
			c := clock.Fake(time.Unix(0, 0))
			t := c.NewTicker(time.Nanosecond)
			defer t.Stop()
			b.ReportAllocs()
			for b.Loop() {
				c.Advance(d)
			}
		})
	}
}

// never is longer than any benchmark advances the clock, so a timer set for it
// stays on the list without ever being reached.
const never = time.Duration(1) << 62

var (
	now  time.Time
	sink context.Context
)
