package conc_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/go-mizu/mizu/conc"
	"github.com/go-mizu/mizu/errs"
)

var (
	first  = errors.New("the first one")
	second = errors.New("the second one")
)

// TestWaitWithNothingStarted is the case a middleware hits on most requests,
// where the handler never started a goroutine at all.
func TestWaitWithNothingStarted(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	if err := g.Wait(); err != nil {
		t.Errorf("a group that ran nothing returned %v", err)
	}
}

func TestEveryGoroutineSucceeds(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	var done atomic.Int64
	for range 10 {
		g.Go(func(context.Context) error {
			done.Add(1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("nothing failed but Wait returned %v", err)
	}
	if n := done.Load(); n != 10 {
		t.Errorf("Wait returned after %d of 10 goroutines", n)
	}
}

func TestWaitReturnsTheError(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return first })

	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
}

// TestTheFirstErrorWins pins down which error comes out when two goroutines
// fail, since "the first one" is only a promise if the order is not the
// scheduler's to decide.
func TestTheFirstErrorWins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g, _ := conc.NewGroup(t.Context())
		a, b := make(chan struct{}), make(chan struct{})
		g.Go(func(context.Context) error { <-a; return first })
		g.Go(func(context.Context) error { <-b; return second })

		// Both are running and blocked, so neither Go call was turned away by
		// the cancellation the other one is about to cause.
		synctest.Wait()
		close(a)
		synctest.Wait()
		close(b)

		if err := g.Wait(); !errors.Is(err, first) {
			t.Errorf("Wait returned %v, want the error that happened first", err)
		}
	})
}

func TestAFailureCancelsTheGroup(t *testing.T) {
	g, ctx := conc.NewGroup(t.Context())
	stopped := make(chan struct{})
	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	})
	g.Go(func(context.Context) error { return first })

	<-stopped
	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
	if err := context.Cause(ctx); !errors.Is(err, first) {
		t.Errorf("the cause was %v, want the error that caused it", err)
	}
}

// TestTheCauseOutlivesTheCancellation is the reason for using cancel with a
// cause. A goroutine that gave up because a sibling failed should be able to
// report the sibling's failure rather than its own cancellation.
func TestTheCauseOutlivesTheCancellation(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	cause := make(chan error, 1)
	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		cause <- context.Cause(ctx)
		return nil
	})
	g.Go(func(context.Context) error { return first })
	g.Wait()

	if err := <-cause; !errors.Is(err, first) {
		t.Errorf("the goroutine saw %v, want %v", err, first)
	}
}

func TestTheParentCancelsTheGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	g, gctx := conc.NewGroup(ctx)
	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	cancel()

	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait returned %v, want a cancellation", err)
	}
	if err := gctx.Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("the group context ended with %v", err)
	}
}

// TestWaitCancelsTheContext keeps anybody from carrying on with the context a
// group handed out. It is finished when the group is.
func TestWaitCancelsTheContext(t *testing.T) {
	g, ctx := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return nil })
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Error("the context is still live after Wait returned")
	}
}

func TestWaitTwice(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return first })
	if err := g.Wait(); !errors.Is(err, first) {
		t.Fatalf("the first Wait returned %v", err)
	}
	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("the second Wait returned %v, want the same error again", err)
	}
}

func TestLimitCapsWhatRunsAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g, _ := conc.NewGroup(t.Context(), conc.Limit(2))
		var running, peak atomic.Int64
		hold := make(chan struct{})

		done := make(chan error, 1)
		go func() {
			for range 10 {
				g.Go(func(context.Context) error {
					n := running.Add(1)
					for {
						old := peak.Load()
						if n <= old || peak.CompareAndSwap(old, n) {
							break
						}
					}
					<-hold
					running.Add(-1)
					return nil
				})
			}
			done <- g.Wait()
		}()

		// Everything that could start has started, and two of ten is the limit
		// doing its job.
		synctest.Wait()
		if n := running.Load(); n != 2 {
			t.Errorf("%d goroutines are running under a limit of 2", n)
		}

		close(hold)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if n := peak.Load(); n > 2 {
			t.Errorf("%d goroutines ran at once under a limit of 2", n)
		}
	})
}

// TestLimitDoesNotDeadlockOnCancellation covers the corner where Go is parked
// on a full semaphore and the group ends underneath it. Waiting for a slot that
// is never coming would hang the caller rather than the goroutine.
func TestLimitDoesNotDeadlockOnCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		g, _ := conc.NewGroup(ctx, conc.Limit(1))

		hold := make(chan struct{})
		g.Go(func(context.Context) error { <-hold; return nil })

		var started atomic.Bool
		returned := make(chan struct{})
		go func() {
			g.Go(func(context.Context) error {
				started.Store(true)
				return nil
			})
			close(returned)
		}()

		synctest.Wait()
		select {
		case <-returned:
			t.Fatal("the second Go did not wait for a slot")
		default:
		}

		cancel()
		<-returned
		close(hold)

		if err := g.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("Wait returned %v, want a cancellation", err)
		}
		if started.Load() {
			t.Error("it started a goroutine after the group had been cancelled")
		}
	})
}

// TestGoAfterAFailureDoesNotRun is the same rule without a limit involved.
func TestGoAfterAFailureDoesNotRun(t *testing.T) {
	g, ctx := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return first })
	<-ctx.Done()

	var started atomic.Bool
	g.Go(func(context.Context) error {
		started.Store(true)
		return nil
	})

	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
	if started.Load() {
		t.Error("it ran a goroutine for a group that had already failed")
	}
}

func TestLimitBelowOne(t *testing.T) {
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Limit(%d) was accepted", n)
				}
			}()
			conc.Limit(n)
		}()
	}
}

func TestGoAfterWait(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Error("Go after Wait was accepted, so that goroutine had nothing waiting for it")
		}
	}()
	g.Go(func(context.Context) error { return nil })
}

// boom is a named function so that the stack trace has something to point at.
func boom() error { panic("kaboom") }

func TestAPanicBecomesAnError(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return boom() })

	err := g.Wait()
	if err == nil {
		t.Fatal("a goroutine panicked and Wait returned nil")
	}
	if k := errs.KindOf(err); k != errs.Internal {
		t.Errorf("the panic came back as %v, want internal", k)
	}
	if c := errs.CodeOf(err); c != "panic" {
		t.Errorf("the code was %q, want %q", c, "panic")
	}
	if !strings.Contains(err.Error(), "kaboom") {
		t.Errorf("the error does not say what the panic was: %v", err)
	}
}

// TestAPanicKeepsItsStack is the point of recovering in the goroutine rather
// than reporting that one panicked. A trace that stops at this package is a
// trace nobody can act on.
func TestAPanicKeepsItsStack(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { return boom() })

	err := g.Wait()
	for _, f := range errs.Stack(err) {
		if strings.HasSuffix(f.Func, "conc_test.boom") {
			return
		}
	}
	t.Errorf("the stack does not reach the line that panicked:\n%v", errs.Stack(err))
}

// TestAPanicWithAnError keeps errors.Is working through a panic, since a panic
// carrying an error is usually a library that gave up on a return value.
func TestAPanicWithAnError(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	g.Go(func(context.Context) error { panic(first) })

	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want something that is %v", err, first)
	}
}

// TestAPanicCancelsTheGroup is what stops the other goroutines from finishing
// work whose result is about to be thrown away.
func TestAPanicCancelsTheGroup(t *testing.T) {
	g, ctx := conc.NewGroup(t.Context())
	stopped := make(chan struct{})
	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		close(stopped)
		return nil
	})
	g.Go(func(context.Context) error { return boom() })

	<-stopped
	if err := g.Wait(); err == nil {
		t.Error("a panic did not become an error")
	}
	if ctx.Err() == nil {
		t.Error("a panic did not cancel the group")
	}
}

func TestPackageGoAndWait(t *testing.T) {
	_, ctx := conc.NewGroup(t.Context())
	var done atomic.Int64
	for range 5 {
		conc.Go(ctx, func(context.Context) error {
			done.Add(1)
			return nil
		})
	}

	if err := conc.Wait(ctx); err != nil {
		t.Fatalf("nothing failed but Wait returned %v", err)
	}
	if n := done.Load(); n != 5 {
		t.Errorf("Wait returned after %d of 5 goroutines", n)
	}
}

func TestPackageGoReportsTheError(t *testing.T) {
	_, ctx := conc.NewGroup(t.Context())
	conc.Go(ctx, func(context.Context) error { return first })

	if err := conc.Wait(ctx); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
}

func TestPackageGoWithoutAGroup(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Go without a group was accepted, so that goroutine was on its own")
		}
	}()
	conc.Go(t.Context(), func(context.Context) error { return nil })
}

// TestPackageWaitWithoutAGroup is deliberately not a panic. Nothing was
// started, so there is nothing to leak.
func TestPackageWaitWithoutAGroup(t *testing.T) {
	if err := conc.Wait(t.Context()); err != nil {
		t.Errorf("Wait on a context with no group returned %v", err)
	}
}

type ctxKey struct{}

// TestPackageGoKeepsTheCallersContext is why Go takes a context at all. A
// request picks up a user, a tenant and a trace after the group was created,
// and a goroutine that lost all three would log against the wrong account.
func TestPackageGoKeepsTheCallersContext(t *testing.T) {
	_, ctx := conc.NewGroup(t.Context())
	ctx = context.WithValue(ctx, ctxKey{}, "tenant-7")

	seen := make(chan any, 1)
	conc.Go(ctx, func(ctx context.Context) error {
		seen <- ctx.Value(ctxKey{})
		return nil
	})
	if err := conc.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	if v := <-seen; v != "tenant-7" {
		t.Errorf("the goroutine saw %v, want tenant-7", v)
	}
}

// TestTheMethodUsesTheGroupsContext is the other half of that. A Group has no
// context handed to it, so its goroutines get the one it was built with.
func TestTheMethodUsesTheGroupsContext(t *testing.T) {
	g, ctx := conc.NewGroup(context.WithValue(t.Context(), ctxKey{}, "tenant-7"))

	seen := make(chan any, 1)
	g.Go(func(ctx context.Context) error {
		seen <- ctx.Value(ctxKey{})
		return nil
	})
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Error("Wait left the context live")
	}
	if v := <-seen; v != "tenant-7" {
		t.Errorf("the goroutine saw %v, want tenant-7", v)
	}
}

// TestNestedGo is what makes the rule about bare go statements hold all the way
// down. A goroutine that starts another one is still inside the same group, so
// the outer Wait covers both.
func TestNestedGo(t *testing.T) {
	_, ctx := conc.NewGroup(t.Context())
	var inner atomic.Int64

	conc.Go(ctx, func(ctx context.Context) error {
		for range 3 {
			conc.Go(ctx, func(context.Context) error {
				inner.Add(1)
				return nil
			})
		}
		return nil
	})

	if err := conc.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if n := inner.Load(); n != 3 {
		t.Errorf("%d of 3 nested goroutines finished before Wait returned", n)
	}
}

// TestNestedFailure checks that a nested goroutine's error is the group's
// error, rather than being lost one level down.
func TestNestedFailure(t *testing.T) {
	_, ctx := conc.NewGroup(t.Context())
	conc.Go(ctx, func(ctx context.Context) error {
		conc.Go(ctx, func(context.Context) error { return first })
		return nil
	})

	if err := conc.Wait(ctx); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
}

// TestManyGoroutines is the shape the race detector has something to say about.
func TestManyGoroutines(t *testing.T) {
	g, _ := conc.NewGroup(t.Context(), conc.Limit(16))
	var total atomic.Int64
	for i := range 1000 {
		g.Go(func(context.Context) error {
			total.Add(int64(i))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	if want := int64(999 * 1000 / 2); total.Load() != want {
		t.Errorf("the total was %d, want %d", total.Load(), want)
	}
}

// TestManyFailures runs the failing path under contention, where fail is called
// from a thousand goroutines and exactly one of them gets to set the error.
func TestManyFailures(t *testing.T) {
	g, _ := conc.NewGroup(t.Context())
	for range 1000 {
		g.Go(func(context.Context) error { return first })
	}
	if err := g.Wait(); !errors.Is(err, first) {
		t.Errorf("Wait returned %v, want %v", err, first)
	}
}
