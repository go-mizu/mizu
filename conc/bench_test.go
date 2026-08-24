package conc_test

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/go-mizu/mizu/conc"
)

// The number that matters is what a group costs over a bare go statement and a
// WaitGroup, because that difference is what somebody is being asked to pay for
// panics coming back as errors and for a failure cancelling the rest.

func BenchmarkNewGroup(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		group, _ = conc.NewGroup(ctx)
	}
}

func BenchmarkNewGroupWithLimit(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		group, _ = conc.NewGroup(ctx, conc.Limit(8))
	}
}

// BenchmarkOneGoroutine is the whole round trip: build a group, run one thing,
// wait for it. That is what a handler starting a background write pays.
func BenchmarkOneGoroutine(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		g, _ := conc.NewGroup(ctx)
		g.Go(fn)
		err = g.Wait()
	}
}

// BenchmarkOneRawGoroutine is the same thing written by hand, so the difference
// between the two is the price of the group.
func BenchmarkOneRawGoroutine(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
		wg.Wait()
	}
}

// BenchmarkPackageGo is the same again through the context, which adds a Value
// lookup on a context that has the group one layer down.
func BenchmarkPackageGo(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		_, gctx := conc.NewGroup(ctx)
		conc.Go(gctx, fn)
		err = conc.Wait(gctx)
	}
}

// BenchmarkWaitWithNothingStarted is what a middleware pays on every request
// where the handler never started anything, which is most of them.
func BenchmarkWaitWithNothingStarted(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		_, gctx := conc.NewGroup(ctx)
		err = conc.Wait(gctx)
	}
}

// BenchmarkFanOut is the batch shape, where the fixed cost of the group is
// spread over enough work to disappear and what is left is scheduling.
func BenchmarkFanOut(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	for _, n := range []int{10, 100, 1000} {
		b.Run(strconv.Itoa(n)+" goroutines", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				g, _ := conc.NewGroup(ctx)
				for range n {
					g.Go(fn)
				}
				err = g.Wait()
			}
		})
	}
}

// BenchmarkFanOutWithLimit is the same fan-out through the semaphore, which is
// a channel send and receive per goroutine on top of everything else.
func BenchmarkFanOutWithLimit(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return nil }

	for _, n := range []int{10, 100, 1000} {
		b.Run(strconv.Itoa(n)+" goroutines", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				g, _ := conc.NewGroup(ctx, conc.Limit(8))
				for range n {
					g.Go(fn)
				}
				err = g.Wait()
			}
		})
	}
}

// BenchmarkFailure is the path where the error has to be recorded and the
// context cancelled while everything else is still running.
func BenchmarkFailure(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return first }

	b.ReportAllocs()
	for b.Loop() {
		g, _ := conc.NewGroup(ctx)
		for range 10 {
			g.Go(fn)
		}
		err = g.Wait()
	}
}

// BenchmarkPanic is the expensive path, since capturing a stack is not cheap.
// It is here to say how expensive, not because anything should be hitting it.
func BenchmarkPanic(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) error { return boom() }

	b.ReportAllocs()
	for b.Loop() {
		g, _ := conc.NewGroup(ctx)
		g.Go(fn)
		err = g.Wait()
	}
}

var (
	err   error
	group *conc.Group
)
