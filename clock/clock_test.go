package clock_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-mizu/mizu/clock"
)

// midnight is the time every fake in here starts at. It is the last day of a
// month in a zone that is not UTC, because both of those are the sort of fact a
// fake clock exists to let a test state.
var midnight = time.Date(2026, 3, 31, 23, 55, 0, 0, time.FixedZone("JST", 9*3600))

// TestFromDefaultsToReal is the property that makes this usable: code written
// against the context works without anybody having set anything up.
func TestFromDefaultsToReal(t *testing.T) {
	ctx := t.Context()

	if c := clock.From(ctx); c == nil {
		t.Fatal("a context with no clock gave back nil")
	}

	before := time.Now()
	got := clock.Now(ctx)
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("the default clock says %v, which is outside %v to %v", got, before, after)
	}
}

func TestWithAndFrom(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	if got := clock.Now(ctx); !got.Equal(midnight) {
		t.Errorf("the context says it is %v, want %v", got, midnight)
	}
	if got := clock.From(ctx); got != clock.Clock(c) {
		t.Error("the clock that came back is not the one that went in")
	}

	// The zone travels with the time, because a test about what day it is in
	// Tokyo and one about what day it is in UTC are different tests.
	if got, want := clock.Now(ctx).Format(time.RFC3339), "2026-03-31T23:55:00+09:00"; got != want {
		t.Errorf("the time reads as %q, want %q", got, want)
	}

	// A context under one that has a clock still has it, and one over it does
	// not, which is the whole of how a test scopes this.
	sub := context.WithValue(ctx, struct{}{}, "anything")
	if got := clock.Now(sub); !got.Equal(midnight) {
		t.Errorf("a derived context lost the clock and says %v", got)
	}
}

// TestAdvanceIsWhatDayItIs is the case testing/synctest cannot cover, and the
// reason this package exists next to it.
func TestAdvanceIsWhatDayItIs(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	if got := clock.Now(ctx).Month(); got != time.March {
		t.Errorf("it is %v", got)
	}

	c.Advance(10 * time.Minute)

	now := clock.Now(ctx)
	if now.Month() != time.April || now.Day() != 1 {
		t.Errorf("ten minutes later it is %v, want the first of April", now)
	}
}

func TestSince(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)
	start := clock.Now(ctx)

	c.Advance(90 * time.Second)

	if got := clock.Since(ctx, start); got != 90*time.Second {
		t.Errorf("%v elapsed, want 90s", got)
	}

	// A time this clock has not reached is a negative duration rather than a
	// panic or a zero, the same as time.Since would give.
	if got := clock.Since(ctx, clock.Now(ctx).Add(time.Hour)); got != -time.Hour {
		t.Errorf("an hour from now is %v ago, want -1h", got)
	}
}

func TestTimer(t *testing.T) {
	c := clock.Fake(midnight)
	timer := c.NewTimer(time.Minute)
	defer timer.Stop()

	select {
	case at := <-timer.C():
		t.Fatalf("it fired at %v without the clock moving", at)
	default:
	}

	c.Advance(59 * time.Second)
	select {
	case at := <-timer.C():
		t.Fatalf("it fired at %v, a second early", at)
	default:
	}

	c.Advance(time.Second)
	select {
	case at := <-timer.C():
		if want := midnight.Add(time.Minute); !at.Equal(want) {
			t.Errorf("it delivered %v, want the deadline %v", at, want)
		}
	default:
		t.Fatal("it did not fire at its deadline")
	}

	// And once only.
	c.Advance(time.Hour)
	select {
	case at := <-timer.C():
		t.Errorf("it fired a second time, at %v", at)
	default:
	}
}

// TestTimerAtOrBeforeNow covers a deadline that has already gone by, which
// time.NewTimer delivers immediately rather than never.
func TestTimerAtOrBeforeNow(t *testing.T) {
	c := clock.Fake(midnight)

	for _, d := range []time.Duration{0, -time.Hour} {
		timer := c.NewTimer(d)
		select {
		case <-timer.C():
		default:
			t.Errorf("a timer for %v did not fire", d)
		}
		timer.Stop()
	}
}

func TestTimerStop(t *testing.T) {
	c := clock.Fake(midnight)

	timer := c.NewTimer(time.Minute)
	if !timer.Stop() {
		t.Error("stopping a timer that had not fired said it had")
	}
	if timer.Stop() {
		t.Error("stopping it twice said it stopped something twice")
	}

	c.Advance(time.Hour)
	select {
	case at := <-timer.C():
		t.Errorf("a stopped timer fired at %v", at)
	default:
	}
	if got := c.Waiting(); got != 0 {
		t.Errorf("%d waiters left after stopping the only one", got)
	}
}

func TestTimerReset(t *testing.T) {
	c := clock.Fake(midnight)

	timer := c.NewTimer(time.Minute)
	defer timer.Stop()

	c.Advance(30 * time.Second)
	if !timer.Reset(time.Minute) {
		t.Error("resetting a live timer said it was not")
	}

	// The original deadline has gone by and the new one has not.
	c.Advance(31 * time.Second)
	select {
	case at := <-timer.C():
		t.Fatalf("it fired at %v, on the deadline it was reset away from", at)
	default:
	}

	c.Advance(29 * time.Second)
	select {
	case <-timer.C():
	default:
		t.Fatal("it did not fire on the new deadline")
	}

	// Resetting one that has fired says so, and it runs again.
	if timer.Reset(time.Minute) {
		t.Error("resetting a fired timer said it was live")
	}
	c.Advance(time.Minute)
	select {
	case <-timer.C():
	default:
		t.Error("a timer reset after firing did not fire again")
	}
}

// TestTimerResetToNow covers the deadline that is already here, which fires on
// the way out of Reset rather than waiting for the clock to be moved.
func TestTimerResetToNow(t *testing.T) {
	c := clock.Fake(midnight)

	timer := c.NewTimer(time.Hour)
	defer timer.Stop()

	if !timer.Reset(0) {
		t.Error("resetting a live timer said it was not")
	}
	select {
	case at := <-timer.C():
		if !at.Equal(midnight) {
			t.Errorf("it delivered %v, want %v", at, midnight)
		}
	default:
		t.Fatal("a timer reset to now did not fire")
	}
}

// TestTimersFireInDeadlineOrder matters because one Advance can pass several
// deadlines, and code that reads two channels in order sees whichever the clock
// delivered first.
func TestTimersFireInDeadlineOrder(t *testing.T) {
	c := clock.Fake(midnight)

	// Created out of order, so the answer comes from the deadlines and not
	// from the order they were registered in.
	ch := make(chan time.Duration, 3)
	for _, d := range []time.Duration{3 * time.Minute, time.Minute, 2 * time.Minute} {
		timer := c.NewTimer(d)
		defer timer.Stop()
		go func() {
			at := <-timer.C()
			ch <- at.Sub(midnight)
		}()
	}
	c.BlockUntil(3)
	c.Advance(time.Hour)

	// The goroutines race with each other after delivery, so what is checked
	// here is that all three arrived and each carries its own deadline.
	got := []time.Duration{<-ch, <-ch, <-ch}
	slices.Sort(got)
	want := []time.Duration{time.Minute, 2 * time.Minute, 3 * time.Minute}
	if !slices.Equal(got, want) {
		t.Errorf("the deadlines delivered were %v, want %v", got, want)
	}
}

func TestTicker(t *testing.T) {
	c := clock.Fake(midnight)

	ticker := c.NewTicker(time.Minute)
	defer ticker.Stop()

	for i := range 3 {
		c.Advance(time.Minute)
		select {
		case at := <-ticker.C():
			if want := midnight.Add(time.Duration(i+1) * time.Minute); !at.Equal(want) {
				t.Errorf("tick %d delivered %v, want %v", i, at, want)
			}
		default:
			t.Fatalf("tick %d did not arrive", i)
		}
	}
}

// TestTickerDropsMissedTicks is the behaviour time.Ticker has and that code
// written against it depends on. Delivering a burst instead would turn a slow
// consumer into a growing queue.
func TestTickerDropsMissedTicks(t *testing.T) {
	c := clock.Fake(midnight)

	ticker := c.NewTicker(time.Minute)
	defer ticker.Stop()

	c.Advance(time.Hour)

	got := 0
	for {
		select {
		case <-ticker.C():
			got++
			continue
		default:
		}
		break
	}
	if got != 1 {
		t.Errorf("an hour of a one minute ticker delivered %d ticks, want 1", got)
	}

	// And the next tick is one period from the hour mark, not from where the
	// sixty missed ones would have left it.
	c.Advance(time.Minute)
	select {
	case at := <-ticker.C():
		if want := midnight.Add(61 * time.Minute); !at.Equal(want) {
			t.Errorf("the next tick is %v, want %v", at, want)
		}
	default:
		t.Error("the ticker stopped after catching up")
	}
}

// TestTickerKeepsTheFirstUnreadTick is the same dropping behaviour arrived at one
// period at a time, which is what a consumer that stops consuming looks like.
// The tick already waiting is the one that stays, since the channel is a buffer
// of one and the send that finds it full gives up.
func TestTickerKeepsTheFirstUnreadTick(t *testing.T) {
	c := clock.Fake(midnight)

	ticker := c.NewTicker(time.Minute)
	defer ticker.Stop()

	c.Advance(time.Minute)
	c.Advance(time.Minute)
	c.Advance(time.Minute)

	select {
	case at := <-ticker.C():
		if want := midnight.Add(time.Minute); !at.Equal(want) {
			t.Errorf("the tick waiting is %v, want the first one at %v", at, want)
		}
	default:
		t.Fatal("no tick was waiting")
	}
	select {
	case at := <-ticker.C():
		t.Errorf("a second tick was queued behind it, at %v", at)
	default:
	}
}

func TestTickerStopAndReset(t *testing.T) {
	c := clock.Fake(midnight)

	ticker := c.NewTicker(time.Minute)
	ticker.Stop()
	c.Advance(time.Hour)
	select {
	case at := <-ticker.C():
		t.Errorf("a stopped ticker fired at %v", at)
	default:
	}

	ticker.Reset(time.Second)
	c.Advance(time.Second)
	select {
	case <-ticker.C():
	default:
		t.Error("a ticker that was reset after being stopped did not fire")
	}
	ticker.Stop()
}

func TestTickerRejectsANonPositivePeriod(t *testing.T) {
	c := clock.Fake(midnight)

	for _, d := range []time.Duration{0, -time.Second} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewTicker(%v) did not panic", d)
				}
			}()
			c.NewTicker(d)
		}()
	}

	ticker := c.NewTicker(time.Minute)
	defer ticker.Stop()
	defer func() {
		if recover() == nil {
			t.Error("Reset to zero did not panic")
		}
	}()
	ticker.Reset(0)
}

// TestContextTimerAndTicker covers the package level constructors, which are
// what framework code calls and which have to reach the context's clock rather
// than the real one.
func TestContextTimerAndTicker(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	timer := clock.NewTimer(ctx, time.Minute)
	defer timer.Stop()
	ticker := clock.NewTicker(ctx, time.Minute)
	defer ticker.Stop()

	if got := c.Waiting(); got != 2 {
		t.Fatalf("%d waiters on the fake clock, want 2, so one of them is on the real one", got)
	}

	c.Advance(time.Minute)
	for name, ch := range map[string]<-chan time.Time{"the timer": timer.C(), "the ticker": ticker.C()} {
		select {
		case <-ch:
		default:
			t.Errorf("%s did not fire", name)
		}
	}
}

func TestAfter(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	ch := clock.After(ctx, time.Minute)
	select {
	case <-ch:
		t.Fatal("it fired without the clock moving")
	default:
	}

	c.Advance(time.Minute)
	select {
	case at := <-ch:
		if want := midnight.Add(time.Minute); !at.Equal(want) {
			t.Errorf("it delivered %v, want %v", at, want)
		}
	default:
		t.Fatal("it did not fire")
	}
}

// TestSleepWakes uses BlockUntil rather than a sleep of its own, which is the
// difference between a test that always passes and one that passes on a busy
// machine as well.
func TestSleepWakes(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	done := make(chan error, 1)
	go func() { done <- clock.Sleep(ctx, time.Hour) }()

	c.BlockUntil(1)
	c.Advance(time.Hour)

	if err := <-done; err != nil {
		t.Errorf("a sleep that ran its course returned %v", err)
	}
	if got := c.Waiting(); got != 0 {
		t.Errorf("%d waiters left after the sleep finished", got)
	}
}

func TestSleepGivesUpWithTheCaller(t *testing.T) {
	c := clock.Fake(midnight)
	ctx, cancel := context.WithCancel(clock.With(t.Context(), c))

	done := make(chan error, 1)
	go func() { done <- clock.Sleep(ctx, time.Hour) }()

	c.BlockUntil(1)
	cancel()

	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Errorf("%v, want context.Canceled", err)
	}

	// The timer goes with it, or a shutdown leaks one per abandoned sleep.
	c.Advance(time.Hour)
	if got := c.Waiting(); got != 0 {
		t.Errorf("%d waiters left after the sleep was canceled", got)
	}
}

func TestSleepChecksTheContextFirst(t *testing.T) {
	c := clock.Fake(midnight)
	ctx, cancel := context.WithCancel(clock.With(t.Context(), c))
	cancel()

	// Zero would otherwise be a way to sleep past a cancellation, which is
	// exactly the loop a shutdown is trying to stop.
	for _, d := range []time.Duration{0, -time.Second, time.Hour} {
		if err := clock.Sleep(ctx, d); !errors.Is(err, context.Canceled) {
			t.Errorf("sleeping %v on a canceled context: %v", d, err)
		}
	}
	if got := c.Waiting(); got != 0 {
		t.Errorf("%d timers created for a sleep that never started", got)
	}
}

func TestSleepOfNothing(t *testing.T) {
	ctx := clock.With(t.Context(), clock.Fake(midnight))

	for _, d := range []time.Duration{0, -time.Second} {
		if err := clock.Sleep(ctx, d); err != nil {
			t.Errorf("sleeping %v returned %v", d, err)
		}
	}
}

func TestBlockUntil(t *testing.T) {
	c := clock.Fake(midnight)

	// Zero is already true, so it returns rather than waiting for something
	// that will not happen.
	c.BlockUntil(0)

	reached := make(chan struct{})
	go func() {
		c.BlockUntil(2)
		close(reached)
	}()

	a := c.NewTimer(time.Hour)
	defer a.Stop()
	select {
	case <-reached:
		t.Fatal("it stopped waiting at one of two")
	default:
	}

	b := c.NewTimer(time.Hour)
	defer b.Stop()
	<-reached
}

func TestSetMovesBothWays(t *testing.T) {
	c := clock.Fake(midnight)

	timer := c.NewTimer(time.Hour)
	defer timer.Stop()

	// Backwards fires nothing and leaves the timer with further to go.
	c.Set(midnight.Add(-24 * time.Hour))
	if got := c.Now(); !got.Equal(midnight.Add(-24 * time.Hour)) {
		t.Errorf("it is %v", got)
	}
	select {
	case at := <-timer.C():
		t.Fatalf("moving the clock back fired a timer, at %v", at)
	default:
	}

	// Forwards past the deadline fires it, and the deadline is still the one
	// it was given rather than one relative to where the clock went.
	c.Set(midnight.Add(2 * time.Hour))
	select {
	case at := <-timer.C():
		if want := midnight.Add(time.Hour); !at.Equal(want) {
			t.Errorf("it delivered %v, want %v", at, want)
		}
	default:
		t.Fatal("it did not fire")
	}
}

func TestAdvanceRejectsGoingBackwards(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Advance by a negative duration did not panic")
		}
	}()
	clock.Fake(midnight).Advance(-time.Second)
}

// TestFakeIsSafeUnderRace runs the operations that touch the waiter list from
// several goroutines at once, since a clock is read from every request.
func TestFakeIsSafeUnderRace(t *testing.T) {
	c := clock.Fake(midnight)
	ctx := clock.With(t.Context(), c)

	const workers = 8
	done := make(chan struct{})
	for range workers {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
				timer := c.NewTimer(time.Millisecond)
				<-timer.C()
				timer.Reset(time.Millisecond)
				timer.Stop()
				_ = clock.Now(ctx)
			}
		}()
	}

	for range 200 {
		c.Advance(time.Millisecond)
	}
	close(done)
}

// TestRealClock covers the path production takes. The durations are short
// because what is being checked is that the calls reach the standard library,
// not that the standard library keeps time.
func TestRealClock(t *testing.T) {
	c := clock.Real()

	start := c.Now()
	if c.Since(start) < 0 {
		t.Error("time ran backwards")
	}

	select {
	case <-c.After(time.Millisecond):
	case <-time.After(time.Second):
		t.Error("After did not fire")
	}

	timer := c.NewTimer(time.Millisecond)
	select {
	case <-timer.C():
	case <-time.After(time.Second):
		t.Error("the timer did not fire")
	}
	if timer.Stop() {
		t.Error("stopping a fired timer said it stopped one")
	}
	if timer.Reset(time.Millisecond) {
		t.Error("resetting a fired timer said it was live")
	}
	timer.Stop()

	ticker := c.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for range 2 {
		select {
		case <-ticker.C():
		case <-time.After(time.Second):
			t.Fatal("the ticker did not tick")
		}
	}
	ticker.Reset(time.Millisecond)

	if err := c.Sleep(t.Context(), time.Millisecond); err != nil {
		t.Errorf("a real sleep returned %v", err)
	}
}

// TestRealSleepUnderSynctest is the other half of the package doc: ordering and
// cancellation are synctest's job, and it works on the real clock, so the code
// under test does not have to be written for either one.
func TestRealSleepUnderSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		slept := make(chan error, 1)
		go func() { slept <- clock.Sleep(ctx, time.Hour) }()

		synctest.Wait()
		select {
		case err := <-slept:
			t.Fatalf("it returned %v before the hour was up", err)
		default:
		}

		cancel()
		if err := <-slept; !errors.Is(err, context.Canceled) {
			t.Errorf("%v, want context.Canceled", err)
		}
	})
}

// TestFakeInsideSynctest says the two compose, which the package doc claims.
// Advancing a fake clock is a function call and never blocks, so it does not
// deadlock a bubble that is waiting for every goroutine to be idle.
func TestFakeInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := clock.Fake(midnight)
		ctx := clock.With(context.Background(), c)

		done := make(chan time.Time, 1)
		go func() {
			_ = clock.Sleep(ctx, 24*time.Hour)
			done <- clock.Now(ctx)
		}()

		synctest.Wait()
		c.Advance(24 * time.Hour)

		if got := (<-done).Day(); got != 1 {
			t.Errorf("a day later it is the %d, want the first", got)
		}
	})
}
