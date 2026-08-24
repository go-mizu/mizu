package clock

import (
	"context"
	"slices"
	"sync"
	"time"
)

// A FakeClock is a clock a test moves by hand.
//
// It starts at whatever [Fake] was given and stays there until [FakeClock.Set]
// or [FakeClock.Advance] moves it. Timers and tickers created on it fire when
// the time passes their deadline and not before, so a test for something that
// happens in an hour takes no longer than one that happens in a microsecond.
//
// Every method is safe to call from any goroutine.
type FakeClock struct {
	mu      sync.Mutex
	woken   sync.Cond // Broadcast when waiters changes, for BlockUntil.
	now     time.Time
	waiters []*waiter
}

// Fake is a clock stopped at t.
//
//	c := clock.Fake(time.Date(2026, 3, 31, 23, 55, 0, 0, time.UTC))
//	ctx = clock.With(ctx, c)
//
// Give it a time with a location, since a test about what day it is in Tokyo
// and a test about what day it is in UTC are different tests.
func Fake(t time.Time) *FakeClock {
	c := &FakeClock{now: t}
	c.woken.L = &c.mu
	return c
}

var _ Clock = (*FakeClock)(nil)

// waiter is one timer or ticker, waiting for the fake clock to reach deadline.
// A period of zero is a timer, which is removed after it fires, and anything
// else is a ticker, which is rescheduled.
type waiter struct {
	deadline time.Time
	period   time.Duration
	ch       chan time.Time
}

// Now is the time this clock is stopped at.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Since is how long ago t was on this clock, which may be negative when t is
// in this clock's future.
func (c *FakeClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

// Advance moves the clock forward by d and fires whatever that reaches.
//
// It panics on a negative duration. Moving a clock backwards is [FakeClock.Set],
// where what happens to the timers is written down.
//
// It returns once every timer it reached has been delivered to. Delivery is to
// a buffered channel, so this does not wait for anybody to receive, and a test
// that needs the code under test to have got there first should say so with
// [FakeClock.BlockUntil].
func (c *FakeClock) Advance(d time.Duration) {
	if d < 0 {
		panic("clock: Advance by a negative duration, use Set to move a fake clock backwards")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set(c.now.Add(d))
}

// Set moves the clock to t.
//
// Moving it forward fires whatever that reaches, the same as [FakeClock.Advance].
// Moving it backwards fires nothing and un-fires nothing: a timer that has
// already delivered has delivered, and one that has not now has further to
// wait. That is a system clock being stepped back, which is a thing that
// happens, and it is why [Clock.Since] on the real clock reads the monotonic
// value instead.
func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set(t)
}

// set moves the clock with the lock held.
func (c *FakeClock) set(t time.Time) {
	back := t.Before(c.now)
	c.now = t
	if !back {
		c.fire()
	}
}

// fire delivers to every waiter the clock has reached, oldest deadline first,
// with the lock held.
func (c *FakeClock) fire() {
	// Left nil until something is actually due, since the common move in a
	// test is advancing past none of the timers that are set.
	var due []*waiter
	for _, w := range c.waiters {
		if !w.deadline.After(c.now) {
			due = append(due, w)
		}
	}
	if len(due) == 0 {
		return
	}
	slices.SortStableFunc(due, func(a, b *waiter) int { return a.deadline.Compare(b.deadline) })

	for _, w := range due {
		// A non-blocking send is what the runtime does. A receiver that is not
		// ready misses this one rather than holding the clock up.
		select {
		case w.ch <- w.deadline:
		default:
		}
		if w.period <= 0 {
			c.remove(w)
			continue
		}

		// Skip whole periods that went by rather than delivering one tick per
		// period, which is what [time.Ticker] does with a slow receiver, and
		// which keeps Advance by an hour from counting out an hour of
		// nanoseconds.
		missed := c.now.Sub(w.deadline)/w.period + 1
		w.deadline = w.deadline.Add(time.Duration(missed) * w.period)
	}
}

// add registers a waiter and wakes anybody in BlockUntil.
func (c *FakeClock) add(w *waiter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waiters = append(c.waiters, w)
	c.woken.Broadcast()

	// A deadline that has already gone by fires now, which is what
	// time.NewTimer with a duration of zero does.
	if !w.deadline.After(c.now) {
		c.fire()
	}
}

// remove drops a waiter and reports whether it was there, with the lock held.
// Being in the list is the whole of a timer's state: one that has fired was
// taken out by fire, and one that was stopped was taken out here, so both
// answer the way [time.Timer.Stop] does.
func (c *FakeClock) remove(w *waiter) bool {
	i := slices.Index(c.waiters, w)
	if i < 0 {
		return false
	}
	c.waiters = slices.Delete(c.waiters, i, i+1)
	return true
}

// BlockUntil waits until n timers and tickers are waiting on this clock.
//
// It is the answer to the one race a fake clock has, which is a test advancing
// the clock before the code under test has created the timer it is waiting on:
//
//	go worker(ctx)      // sleeps for a minute somewhere inside
//	c.BlockUntil(1)     // it has got there
//	c.Advance(time.Minute)
//
// A count that never arrives blocks forever, and what catches that is the test
// binary's own timeout, whose goroutine dump names the goroutine that did not
// get where it was supposed to.
func (c *FakeClock) BlockUntil(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.waiters) < n {
		c.woken.Wait()
	}
}

// Waiting is how many timers and tickers are waiting on this clock right now.
//
// It is for an assertion that something did not set a timer. Waiting for one to
// appear is [FakeClock.BlockUntil], because a loop around this is a race with a
// sleep in it.
func (c *FakeClock) Waiting() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.waiters)
}

// After is a channel that receives once, when this clock reaches d from now.
func (c *FakeClock) After(d time.Duration) <-chan time.Time { return c.NewTimer(d).C() }

// NewTimer is a timer on this clock. It fires when the clock is moved past d
// from now, and a duration that is zero or negative fires immediately.
func (c *FakeClock) NewTimer(d time.Duration) Timer {
	t := &fakeTimer{clock: c, w: &waiter{deadline: c.Now().Add(d), ch: make(chan time.Time, 1)}}
	c.add(t.w)
	return t
}

// NewTicker is a ticker on this clock. It panics on a period that is not
// positive, the same as [time.NewTicker].
func (c *FakeClock) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("clock: non-positive interval for NewTicker")
	}
	t := &fakeTicker{clock: c, w: &waiter{deadline: c.Now().Add(d), period: d, ch: make(chan time.Time, 1)}}
	c.add(t.w)
	return t
}

// Sleep waits for the clock to be moved past d, or for the context to be done.
//
// Nothing moves a fake clock on its own, so this returns when another goroutine
// advances it. A test that calls this on the same goroutine that would have
// advanced it deadlocks, and that is the test saying something true.
func (c *FakeClock) Sleep(ctx context.Context, d time.Duration) error { return sleep(ctx, c, d) }

type fakeTimer struct {
	clock *FakeClock
	w     *waiter
}

func (t *fakeTimer) C() <-chan time.Time { return t.w.ch }

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	return t.clock.remove(t.w)
}

func (t *fakeTimer) Reset(d time.Duration) bool {
	c := t.clock
	c.mu.Lock()
	defer c.mu.Unlock()

	active := c.remove(t.w)
	t.w.deadline = c.now.Add(d)
	c.waiters = append(c.waiters, t.w)
	c.woken.Broadcast()

	if !t.w.deadline.After(c.now) {
		c.fire()
	}
	return active
}

type fakeTicker struct {
	clock *FakeClock
	w     *waiter
}

func (t *fakeTicker) C() <-chan time.Time { return t.w.ch }

func (t *fakeTicker) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.clock.remove(t.w)
}

func (t *fakeTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic("clock: non-positive interval for Reset")
	}
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()

	t.clock.remove(t.w)
	t.w.deadline, t.w.period = t.clock.now.Add(d), d
	t.clock.waiters = append(t.clock.waiters, t.w)
	t.clock.woken.Broadcast()
}
