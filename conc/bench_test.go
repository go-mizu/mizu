package conc_test

import (
	"bytes"
	"context"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

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

// The fan-out helpers are all a group underneath, so what these measure is what
// each one adds to BenchmarkFanOut: a slice of results for Map, a channel per
// element for MapSeq, nothing much for Each.

func BenchmarkMap(b *testing.B) {
	ctx := context.Background()
	fn := func(_ context.Context, n int) (int, error) { return n, nil }

	for _, n := range []int{10, 100, 1000} {
		in := make([]int, n)
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				ints, err = conc.Map(ctx, in, 8, fn)
			}
		})
	}
}

func BenchmarkEach(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context, int) error { return nil }

	for _, n := range []int{10, 100, 1000} {
		in := make([]int, n)
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				err = conc.Each(ctx, in, 8, fn)
			}
		})
	}
}

// BenchmarkMapSeq is the price of streaming rather than collecting. Each
// element gets a channel of its own, which is what keeps the results in order
// without holding all of them at once.
func BenchmarkMapSeq(b *testing.B) {
	ctx := context.Background()
	fn := func(_ context.Context, n int) (int, error) { return n, nil }

	for _, n := range []int{10, 100, 1000} {
		in := slices.Values(make([]int, n))
		b.Run(strconv.Itoa(n)+" elements", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				for v, e := range conc.MapSeq(ctx, in, 8, fn) {
					sink, err = v, e
				}
			}
		})
	}
}

func BenchmarkAll(b *testing.B) {
	ctx := context.Background()
	fn := func(context.Context) (int, error) { return 0, nil }

	b.ReportAllocs()
	for b.Loop() {
		ints, err = conc.All(ctx, fn, fn, fn)
	}
}

// BenchmarkRace has one function that answers at once and two that wait to be
// cancelled, so what it measures is the winner plus the cost of stopping the
// field and waiting for it.
func BenchmarkRace(b *testing.B) {
	ctx := context.Background()
	winner := func(context.Context) (int, error) { return 1, nil }
	loser := func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	}

	b.ReportAllocs()
	for b.Loop() {
		sink, err = conc.Race(ctx, winner, loser, loser)
	}
}

// BenchmarkDebounce is the call that does not run anything, which is the one
// that happens over and over. Every one of them stops a timer and starts
// another.
func BenchmarkDebounce(b *testing.B) {
	nudge := conc.Debounce(time.Hour, func() {})

	b.ReportAllocs()
	for b.Loop() {
		nudge()
	}
}

// BenchmarkThrottle is the call that is turned away, since the one that runs is
// whatever fn costs plus a timer.
func BenchmarkThrottle(b *testing.B) {
	report := conc.Throttle(time.Hour, func() {})
	report()

	b.ReportAllocs()
	for b.Loop() {
		report()
	}
}

// BenchmarkOnce is the call after the first, which is what a lazily built
// dependency pays on every use.
func BenchmarkOnce(b *testing.B) {
	region := conc.Once(func() (int, error) { return 1, nil })
	region()

	b.ReportAllocs()
	for b.Loop() {
		sink, err = region()
	}
}

// BenchmarkPool is a Get and a Put together, since neither happens without the
// other.
func BenchmarkPool(b *testing.B) {
	p := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })

	b.ReportAllocs()
	for b.Loop() {
		v := p.Get()
		p.Put(v)
	}
}

// BenchmarkSyncPool is the same thing untyped, so the difference between the
// two is what the type costs. It should be nothing.
func BenchmarkSyncPool(b *testing.B) {
	p := sync.Pool{New: func() any { return new(bytes.Buffer) }}

	b.ReportAllocs()
	for b.Loop() {
		v := p.Get().(*bytes.Buffer)
		p.Put(v)
	}
}

var (
	err   error
	group *conc.Group
	ints  []int
	sink  int
)
