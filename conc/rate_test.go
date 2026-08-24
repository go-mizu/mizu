package conc_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-mizu/mizu/conc"
)

// The timing here is all inside a testing/synctest bubble, where time moves
// when nothing is left to run and not otherwise. So "after 200 milliseconds"
// means exactly that rather than "after the machine got round to it", and none
// of these tests take any real time.

const tick = 200 * time.Millisecond

func TestDebounceRunsOnceAfterABurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		burst := conc.Debounce(tick, func() { runs.Add(1) })

		for range 100 {
			burst()
			time.Sleep(tick / 10)
		}
		synctest.Wait()
		if n := runs.Load(); n != 0 {
			t.Fatalf("it ran %d times during the burst, want none until it stopped", n)
		}

		time.Sleep(tick)
		synctest.Wait()
		if n := runs.Load(); n != 1 {
			t.Errorf("a burst of 100 produced %d runs, want 1", n)
		}
	})
}

func TestDebounceRunsAgainAfterTheNextQuietSpell(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		save := conc.Debounce(tick, func() { runs.Add(1) })

		for range 3 {
			save()
			time.Sleep(2 * tick)
			synctest.Wait()
		}

		if n := runs.Load(); n != 3 {
			t.Errorf("three calls spread out produced %d runs, want 3", n)
		}
	})
}

// TestDebounceRunsAfterTheLastCall pins the delay down. The run is d after the
// call that ended the burst, not d after the one that started it.
func TestDebounceRunsAfterTheLastCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var at time.Time
		var mu sync.Mutex

		nudge := conc.Debounce(tick, func() {
			mu.Lock()
			defer mu.Unlock()
			at = time.Now()
		})

		nudge()
		time.Sleep(tick / 2)
		nudge()

		time.Sleep(2 * tick)
		synctest.Wait()

		mu.Lock()
		defer mu.Unlock()
		if want := tick + tick/2; at.Sub(start) != want {
			t.Errorf("it ran %v after the first call, want %v", at.Sub(start), want)
		}
	})
}

// TestDebounceDoesNotRunAlongsideItself covers the timer that has already fired
// and cannot be stopped. Without the generation check it would run fn on its
// way out, and without the lock it would do so next to the run it lost to.
func TestDebounceDoesNotRunAlongsideItself(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running, peak, runs atomic.Int64

		slow := conc.Debounce(tick, func() {
			runs.Add(1)
			if n := running.Add(1); n > peak.Load() {
				peak.Store(n)
			}
			time.Sleep(4 * tick)
			running.Add(-1)
		})

		slow()
		time.Sleep(tick)
		synctest.Wait()
		if runs.Load() != 1 {
			t.Fatalf("the first run had not started")
		}

		// Every one of these lands while the first run is still going.
		for range 3 {
			slow()
			time.Sleep(tick / 2)
		}

		time.Sleep(10 * tick)
		synctest.Wait()

		if n := peak.Load(); n != 1 {
			t.Errorf("%d runs overlapped, want them one at a time", n)
		}
		if n := runs.Load(); n != 2 {
			t.Errorf("it ran %d times, want 2: the first one and the calls that arrived during it", n)
		}
	})
}

// TestDebounceDropsATimerThatLostTheRace covers the timer that has already
// fired, which Stop cannot call back. Without the generation count it would run
// fn after a later call had already replaced it.
func TestDebounceDropsATimerThatLostTheRace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		slow := conc.Debounce(tick, func() {
			runs.Add(1)
			time.Sleep(4 * tick)
		})

		// The first run starts one tick in and holds the line until five.
		slow()
		time.Sleep(tick)
		synctest.Wait()

		// This one's timer fires at three ticks and then waits behind the run
		// that is already going.
		time.Sleep(tick)
		slow()
		time.Sleep(2 * tick)
		synctest.Wait()

		// And this one replaces it while it is still waiting, so by the time
		// the run in progress finishes there is nothing left for the older
		// timer to do.
		slow()

		time.Sleep(10 * tick)
		synctest.Wait()

		if n := runs.Load(); n != 2 {
			t.Errorf("it ran %d times, want 2: the first one and the last call", n)
		}
	})
}

func TestDebounceFromManyGoroutines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		nudge := conc.Debounce(tick, func() { runs.Add(1) })

		g, _ := conc.NewGroup(t.Context())
		for range 50 {
			g.Go(func(ctx context.Context) error {
				nudge()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}

		time.Sleep(2 * tick)
		synctest.Wait()
		if n := runs.Load(); n != 1 {
			t.Errorf("50 goroutines produced %d runs, want 1", n)
		}
	})
}

func TestThrottleRunsTheFirstCall(t *testing.T) {
	var runs atomic.Int64
	report := conc.Throttle(time.Hour, func() { runs.Add(1) })

	report()
	if n := runs.Load(); n != 1 {
		t.Errorf("the first call ran %d times, want 1", n)
	}
}

func TestThrottleDropsWhatComesNext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		report := conc.Throttle(tick, func() { runs.Add(1) })

		for range 100 {
			report()
			time.Sleep(tick / 10)
		}
		synctest.Wait()

		// 100 calls a tenth of a tick apart is ten ticks of wall time, so ten
		// of them get through and the other ninety are dropped.
		if n := runs.Load(); n != 10 {
			t.Errorf("100 calls over ten windows ran %d times, want 10", n)
		}
	})
}

func TestThrottleOpensAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		beat := conc.Throttle(tick, func() { runs.Add(1) })

		beat()
		beat()
		time.Sleep(2 * tick)
		synctest.Wait()
		beat()

		if n := runs.Load(); n != 2 {
			t.Errorf("it ran %d times, want 2: one in each window", n)
		}
	})
}

// TestThrottleRunsOnTheCallersGoroutine is what makes a panic in fn land on the
// caller rather than on a timer nobody is watching.
func TestThrottleRunsOnTheCallersGoroutine(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a panic in fn did not reach the caller")
		}
	}()

	conc.Throttle(time.Hour, func() { panic("from inside") })()
}

// TestThrottleFromInsideItself is why fn runs outside the lock. A function that
// calls its own throttled form is turned away rather than deadlocked.
func TestThrottleFromInsideItself(t *testing.T) {
	var runs atomic.Int64
	var again func()
	again = conc.Throttle(time.Hour, func() {
		if runs.Add(1) == 1 {
			again()
		}
	})

	again()
	if n := runs.Load(); n != 1 {
		t.Errorf("it ran %d times, want 1", n)
	}
}

func TestThrottleFromManyGoroutines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int64
		report := conc.Throttle(time.Hour, func() { runs.Add(1) })

		g, _ := conc.NewGroup(t.Context())
		for range 50 {
			g.Go(func(ctx context.Context) error {
				report()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}

		if n := runs.Load(); n != 1 {
			t.Errorf("50 goroutines in one window ran it %d times, want 1", n)
		}
	})
}
