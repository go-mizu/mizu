package conc_test

import (
	"context"
	"errors"
	"iter"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/go-mizu/mizu/conc"
	"github.com/go-mizu/mizu/errs"
)

func double(_ context.Context, n int) (int, error) { return n * 2, nil }

func TestMap(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	out, err := conc.Map(t.Context(), in, 2, double)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{2, 4, 6, 8, 10}; !slices.Equal(out, want) {
		t.Errorf("got %v, want %v", out, want)
	}
}

// TestMapKeepsTheOrder is the promise worth testing on its own, because the
// obvious implementation with a channel does not keep it and nothing about the
// results would look wrong.
func TestMapKeepsTheOrder(t *testing.T) {
	in := make([]int, 500)
	for i := range in {
		in[i] = i
	}

	out, err := conc.Map(t.Context(), in, 16, func(_ context.Context, n int) (string, error) {
		return strconv.Itoa(n), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range out {
		if s != strconv.Itoa(i) {
			t.Fatalf("position %d holds %q", i, s)
		}
	}
}

func TestMapReturnsNothingWithAnError(t *testing.T) {
	out, err := conc.Map(t.Context(), []int{1, 2, 3}, 1, func(_ context.Context, n int) (int, error) {
		if n == 2 {
			return 0, first
		}
		return n, nil
	})
	if !errors.Is(err, first) {
		t.Errorf("Map returned %v, want %v", err, first)
	}
	if out != nil {
		t.Errorf("Map returned %v alongside the error, want nothing", out)
	}
}

// TestMapStopsAfterAnError checks that the failure reaches the work that has
// not started, which is the reason to fan out through a group rather than a
// WaitGroup.
func TestMapStopsAfterAnError(t *testing.T) {
	in := make([]int, 100)
	var ran atomic.Int64

	_, err := conc.Map(t.Context(), in, 1, func(_ context.Context, _ int) (int, error) {
		if ran.Add(1) == 1 {
			return 0, first
		}
		return 0, nil
	})
	if !errors.Is(err, first) {
		t.Fatalf("Map returned %v, want %v", err, first)
	}
	if n := ran.Load(); n > 2 {
		t.Errorf("%d of 100 ran after the first one failed", n-1)
	}
}

func TestMapOfNothing(t *testing.T) {
	var in []int
	// len(in) is zero here, which is how somebody asks for no limit on an empty
	// slice. It has to work rather than panic.
	out, err := conc.Map(t.Context(), in, len(in), double)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("got %v, want nothing", out)
	}
}

func TestMapWithALimitBelowOne(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a limit of zero over a non-empty input was accepted")
		}
	}()
	conc.Map(t.Context(), []int{1}, 0, double)
}

func TestMapRecoversAPanic(t *testing.T) {
	_, err := conc.Map(t.Context(), []int{1}, 1, func(context.Context, int) (int, error) {
		return 0, boom()
	})
	if errs.CodeOf(err) != "panic" {
		t.Errorf("Map returned %v, want a recovered panic", err)
	}
}

func TestMapRespectsTheLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make([]int, 20)
		hold := make(chan struct{})
		var running atomic.Int64

		done := make(chan error, 1)
		go func() {
			_, err := conc.Map(t.Context(), in, 3, func(_ context.Context, n int) (int, error) {
				running.Add(1)
				<-hold
				running.Add(-1)
				return n, nil
			})
			done <- err
		}()

		synctest.Wait()
		if n := running.Load(); n != 3 {
			t.Errorf("%d elements are in flight under a limit of 3", n)
		}

		close(hold)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestEach(t *testing.T) {
	var total atomic.Int64
	err := conc.Each(t.Context(), []int{1, 2, 3, 4}, 2, func(_ context.Context, n int) error {
		total.Add(int64(n))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total.Load() != 10 {
		t.Errorf("the total was %d, want 10", total.Load())
	}
}

func TestEachReturnsTheError(t *testing.T) {
	err := conc.Each(t.Context(), []int{1, 2, 3}, 1, func(_ context.Context, n int) error {
		if n == 2 {
			return first
		}
		return nil
	})
	if !errors.Is(err, first) {
		t.Errorf("Each returned %v, want %v", err, first)
	}
}

func TestEachOfNothing(t *testing.T) {
	var in []int
	if err := conc.Each(t.Context(), in, len(in), func(context.Context, int) error {
		t.Error("it called the function for an empty input")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEachWithALimitBelowOne(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a limit of zero over a non-empty input was accepted")
		}
	}()
	conc.Each(t.Context(), []int{1}, 0, func(context.Context, int) error { return nil })
}

func TestAll(t *testing.T) {
	out, err := conc.All(t.Context(),
		func(context.Context) (string, error) { return "header", nil },
		func(context.Context) (string, error) { return "body", nil },
		func(context.Context) (string, error) { return "footer", nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"header", "body", "footer"}; !slices.Equal(out, want) {
		t.Errorf("got %v, want %v", out, want)
	}
}

func TestAllReturnsNothingWithAnError(t *testing.T) {
	out, err := conc.All(t.Context(),
		func(context.Context) (string, error) { return "header", nil },
		func(context.Context) (string, error) { return "", first },
	)
	if !errors.Is(err, first) {
		t.Errorf("All returned %v, want %v", err, first)
	}
	if out != nil {
		t.Errorf("All returned %v alongside the error, want nothing", out)
	}
}

// TestAllOfNothing is deliberately not an error. Everything in an empty list
// succeeded.
func TestAllOfNothing(t *testing.T) {
	out, err := conc.All[int](t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("got %v, want nothing", out)
	}
}

func TestAllRecoversAPanic(t *testing.T) {
	_, err := conc.All(t.Context(), func(context.Context) (int, error) { return 0, boom() })
	if errs.CodeOf(err) != "panic" {
		t.Errorf("All returned %v, want a recovered panic", err)
	}
}

func TestRace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		got, err := conc.Race(t.Context(),
			func(ctx context.Context) (string, error) {
				// This one never finishes on its own, so the only way out is
				// the cancellation that winning causes.
				<-ctx.Done()
				return "slow", ctx.Err()
			},
			func(context.Context) (string, error) { return "fast", nil },
		)

		if err != nil {
			t.Fatal(err)
		}
		if got != "fast" {
			t.Errorf("Race returned %q, want the one that finished first", got)
		}
	})
}

// TestRaceWaitsForTheLosers is the difference between this and starting two
// goroutines and taking the first answer. A loser still writing to a buffer
// after the caller has moved on is the failure mode.
func TestRaceWaitsForTheLosers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var stopped atomic.Bool
		_, err := conc.Race(t.Context(),
			func(ctx context.Context) (int, error) {
				<-ctx.Done()
				stopped.Store(true)
				return 0, ctx.Err()
			},
			func(context.Context) (int, error) { return 1, nil },
		)
		if err != nil {
			t.Fatal(err)
		}
		if !stopped.Load() {
			t.Error("Race returned with a loser still running")
		}
	})
}

// TestRaceTellsTheLosersWhy keeps a cancelled loser from logging that the
// request was abandoned when it was not.
func TestRaceTellsTheLosersWhy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cause := make(chan error, 1)
		_, err := conc.Race(t.Context(),
			func(ctx context.Context) (int, error) {
				<-ctx.Done()
				cause <- context.Cause(ctx)
				return 0, ctx.Err()
			},
			func(context.Context) (int, error) { return 1, nil },
		)
		if err != nil {
			t.Fatal(err)
		}
		if c := <-cause; c == nil || errors.Is(c, context.Canceled) {
			t.Errorf("the loser was told %v, want something that says it lost", c)
		}
	})
}

func TestRaceWhenEverythingFails(t *testing.T) {
	_, err := conc.Race(t.Context(),
		func(context.Context) (int, error) { return 0, first },
		func(context.Context) (int, error) { return 0, second },
	)
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Errorf("Race returned %v, want both failures", err)
	}
}

// TestRaceIgnoresAFailureThatCameFirst is the whole point. One of two ways of
// getting an answer failing is not the answer failing.
func TestRaceIgnoresAFailureThatCameFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Held until everything else in the bubble is blocked, which is after
		// Race has taken the failure and gone back to waiting.
		slow := make(chan struct{})
		go func() {
			synctest.Wait()
			close(slow)
		}()

		got, err := conc.Race(t.Context(),
			func(context.Context) (string, error) { return "", first },
			func(ctx context.Context) (string, error) {
				<-slow
				return "the answer", nil
			},
		)
		if err != nil {
			t.Fatalf("Race gave up after one failure: %v", err)
		}
		if got != "the answer" {
			t.Errorf("Race returned %q", got)
		}
	})
}

func TestRaceRecoversAPanic(t *testing.T) {
	got, err := conc.Race(t.Context(),
		func(context.Context) (int, error) { return 0, boom() },
		func(context.Context) (int, error) { return 7, nil },
	)
	if err != nil {
		t.Fatalf("a panic in one of them ended the race: %v", err)
	}
	if got != 7 {
		t.Errorf("Race returned %d, want 7", got)
	}
}

func TestRaceWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := conc.Race(ctx, func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Race returned %v, want a cancellation", err)
	}
}

func TestRaceWithNothing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Race with no functions was accepted, and there is no answer to give")
		}
	}()
	conc.Race[int](t.Context())
}

// seq is the input to the MapSeq tests, and counts what has been read out of it
// so that the tests can check nothing is pulled ahead of the limit.
func seq(n int, read *atomic.Int64) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if read != nil {
				read.Add(1)
			}
			if !yield(i) {
				return
			}
		}
	}
}

func TestMapSeq(t *testing.T) {
	var got []int
	for v, err := range conc.MapSeq(t.Context(), seq(10, nil), 3, double) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}

	want := []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestMapSeqKeepsTheOrder makes the elements finish in the opposite order to
// the one they arrived in, which is what an unordered pipeline would show.
func TestMapSeqKeepsTheOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const n = 20
		gates := make([]chan struct{}, n)
		for i := range gates {
			gates[i] = make(chan struct{})
		}

		// Release the last one first and let it cascade backwards, so every
		// element finishes before the one in front of it.
		go func() {
			synctest.Wait()
			for i := n - 1; i >= 0; i-- {
				close(gates[i])
				synctest.Wait()
			}
		}()

		var got []int
		for v, err := range conc.MapSeq(t.Context(), seq(n, nil), n, func(_ context.Context, i int) (int, error) {
			<-gates[i]
			return i, nil
		}) {
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, v)
		}

		for i, v := range got {
			if v != i {
				t.Fatalf("position %d holds %d, so the order came from the finishing and not the input", i, v)
			}
		}
	})
}

// TestMapSeqDoesNotReadAhead is what makes this usable on an input that does
// not fit in memory, or does not end.
func TestMapSeqDoesNotReadAhead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var read atomic.Int64
		hold := make(chan struct{})

		next, stop := iter.Pull2(conc.MapSeq(t.Context(), seq(1000, &read), 4, func(_ context.Context, i int) (int, error) {
			<-hold
			return i, nil
		}))
		defer stop()

		// Nothing has been asked for yet, so nothing should have been read.
		synctest.Wait()
		if n := read.Load(); n > 4 {
			t.Errorf("it read %d elements ahead under a limit of 4", n)
		}

		close(hold)
		if _, _, ok := next(); !ok {
			t.Fatal("the sequence was empty")
		}
	})
}

func TestMapSeqYieldsTheErrorAndCarriesOn(t *testing.T) {
	var got []int
	var failed []int

	for v, err := range conc.MapSeq(t.Context(), seq(6, nil), 2, func(_ context.Context, i int) (int, error) {
		if i%2 == 1 {
			return 0, first
		}
		return i, nil
	}) {
		if err != nil {
			failed = append(failed, len(got)+len(failed))
			continue
		}
		got = append(got, v)
	}

	if want := []int{0, 2, 4}; !slices.Equal(got, want) {
		t.Errorf("the successes were %v, want %v", got, want)
	}
	if want := []int{1, 3, 5}; !slices.Equal(failed, want) {
		t.Errorf("the failures were at %v, want %v", failed, want)
	}
}

// TestMapSeqStopsWhenTheCallerDoes is the leak test. Breaking out of the loop
// has to take everything with it before the loop is over.
func TestMapSeqStopsWhenTheCallerDoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running atomic.Int64

		for v, err := range conc.MapSeq(t.Context(), seq(1000, nil), 8, func(ctx context.Context, i int) (int, error) {
			running.Add(1)
			defer running.Add(-1)
			return i, nil
		}) {
			_ = v
			if err != nil {
				t.Fatal(err)
			}
			break
		}

		// The loop is over, so nothing it started is still going.
		if n := running.Load(); n != 0 {
			t.Errorf("%d elements are still running after the loop ended", n)
		}
	})
}

func TestMapSeqRecoversAPanic(t *testing.T) {
	seen := 0
	for _, err := range conc.MapSeq(t.Context(), seq(3, nil), 1, func(_ context.Context, i int) (int, error) {
		if i == 1 {
			return 0, boom()
		}
		return i, nil
	}) {
		seen++
		if i := seen - 1; i == 1 && errs.CodeOf(err) != "panic" {
			t.Errorf("element 1 came back as %v, want a recovered panic", err)
		}
	}
	if seen != 3 {
		t.Errorf("the sequence stopped after %d of 3, so a panic ended it", seen)
	}
}

func TestMapSeqWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	var errSeen error
	for _, err := range conc.MapSeq(ctx, seq(100, nil), 2, func(ctx context.Context, i int) (int, error) {
		cancel()
		<-ctx.Done()
		return 0, ctx.Err()
	}) {
		if err != nil {
			errSeen = err
			break
		}
	}
	if !errors.Is(errSeen, context.Canceled) {
		t.Errorf("the sequence ended with %v, want a cancellation", errSeen)
	}
}

// TestMapSeqWhenTheContextEndsWhileWaitingForRoom is the cancellation nothing
// else notices. The pipeline is full, the goroutine reading the input is
// waiting for room rather than running anything, and no piece of work fails. So
// the only thing holding the reason is the context, and it still has to reach
// the caller.
func TestMapSeqWhenTheContextEndsWhileWaitingForRoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		release := make(chan struct{})

		go func() {
			// The first wait is for the pipeline to fill and the input to park
			// against it. The second is for the cancellation to be acted on
			// before anything makes room, which is what keeps this shape the
			// same on every run.
			synctest.Wait()
			cancel()
			synctest.Wait()
			close(release)
		}()

		var got []int
		var last error
		for v, err := range conc.MapSeq(ctx, seq(5, nil), 2, func(_ context.Context, i int) (int, error) {
			<-release
			return i, nil
		}) {
			last = err
			if err == nil {
				got = append(got, v)
			}
		}

		if len(got) == 0 {
			t.Error("nothing came out of the pipeline that was already full")
		}
		if !errors.Is(last, context.Canceled) {
			t.Errorf("the sequence ended with %v, want a cancellation", last)
		}
	})
}

// TestMapSeqWhenTheInputPanics covers the one thing that can fail that is not
// the caller's function: the sequence being read from. What it had already
// handed over still comes out, and the panic is the last thing the loop sees.
func TestMapSeqWhenTheInputPanics(t *testing.T) {
	bad := func(yield func(int) bool) {
		yield(1)
		yield(2)
		panic("the source gave up")
	}

	var got []int
	var last error
	for v, err := range conc.MapSeq(t.Context(), bad, 4, double) {
		last = err
		if err == nil {
			got = append(got, v)
		}
	}

	if want := []int{2, 4}; !slices.Equal(got, want) {
		t.Errorf("the elements before the panic were %v, want %v", got, want)
	}
	if errs.CodeOf(last) != "panic" {
		t.Errorf("the sequence ended with %v, want the panic from the input", last)
	}
}

// TestMapSeqOnADeadContext is the case that used to hang. Nothing starts, so
// nothing closes the pipeline, and a loop waiting on it would wait forever.
func TestMapSeqOnADeadContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	yielded := 0
	var last error
	for _, err := range conc.MapSeq(ctx, seq(10, nil), 2, double) {
		yielded++
		last = err
	}

	if yielded != 1 {
		t.Errorf("it yielded %d times on a context that was already over, want 1", yielded)
	}
	if !errors.Is(last, context.Canceled) {
		t.Errorf("it yielded %v, want a cancellation", last)
	}
}

// TestMapSeqWhenTheGroupEndsMidStream covers the window where an element has
// taken its place in the pipeline and the group then refuses to start the
// goroutine that was going to fill it. Nothing else would ever write there, so
// getting this wrong is a loop that never ends.
//
// Cancelling from inside the sequence is what makes that window land in the
// same place every run: the sequence is read by the goroutine that fills the
// pipeline, so the cancellation happens between one element taking its slot and
// the next one being started.
func TestMapSeqWhenTheGroupEndsMidStream(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	in := func(yield func(int) bool) {
		for i := range 10 {
			if i == 1 {
				cancel()
			}
			if !yield(i) {
				return
			}
		}
	}

	var last error
	for _, err := range conc.MapSeq(ctx, in, 4, double) {
		last = err
	}

	if !errors.Is(last, context.Canceled) {
		t.Fatalf("the sequence ended with %v, want a cancellation", last)
	}
}

func TestMapSeqOfNothing(t *testing.T) {
	for range conc.MapSeq(t.Context(), seq(0, nil), 1, double) {
		t.Error("an empty sequence yielded something")
	}
}

func TestMapSeqWithALimitBelowOne(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a limit of zero was accepted")
		}
	}()
	conc.MapSeq(t.Context(), seq(1, nil), 0, double)
}

// TestMapSeqWithALimitOfOne is the corner the pipeline sizing has to get right,
// since the queue behind the element in flight is empty at that limit.
func TestMapSeqWithALimitOfOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running, peak atomic.Int64
		var got []int

		for v, err := range conc.MapSeq(t.Context(), seq(5, nil), 1, func(_ context.Context, i int) (int, error) {
			n := running.Add(1)
			if n > peak.Load() {
				peak.Store(n)
			}
			synctest.Wait()
			running.Add(-1)
			return i, nil
		}) {
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, v)
		}

		if want := []int{0, 1, 2, 3, 4}; !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if peak.Load() != 1 {
			t.Errorf("%d ran at once under a limit of 1", peak.Load())
		}
	})
}
