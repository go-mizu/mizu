package clock

import (
	"context"
	"time"
)

// A Clock is what time it is and how to wait.
//
// The real one is [Real] and a test one is [Fake]. Everything in mizu takes one
// from the context rather than calling [time.Now], so that a test can decide
// what day it is.
type Clock interface {
	// Now is the current time, the way [time.Now] gives it.
	Now() time.Time

	// Since is how long ago a time was. On the real clock this is
	// [time.Since], which uses the monotonic reading and is not affected by
	// the system clock being set.
	Since(t time.Time) time.Duration

	// After is a channel that receives once, after d. It is [time.After], and
	// the same caution applies: the timer behind it is not collected until it
	// fires, so a loop that abandons one leaks until then. Use NewTimer and
	// Stop it where that matters.
	After(d time.Duration) <-chan time.Time

	// NewTimer is one delivery after d.
	NewTimer(d time.Duration) Timer

	// NewTicker is a delivery every d. The period must be positive.
	NewTicker(d time.Duration) Ticker

	// Sleep waits for d or until the context is done, whichever comes first.
	// It returns the context's error in the second case and nil in the first.
	Sleep(ctx context.Context, d time.Duration) error
}

// A Timer fires once.
//
// It is [time.Timer] with the channel behind a method, since a struct with an
// exported field cannot be implemented a second way.
type Timer interface {
	// C is the channel the time is delivered on. It is buffered, so a timer
	// that fires with nobody receiving does not block and does not leak.
	C() <-chan time.Time

	// Stop prevents a timer that has not fired from firing, and reports
	// whether it stopped one. False means it had already fired or had already
	// been stopped.
	Stop() bool

	// Reset changes when the timer fires and reports what Stop would have.
	// A timer that has already fired may have a value waiting on C, and
	// resetting does not remove it.
	Reset(d time.Duration) bool
}

// A Ticker fires every period until stopped.
//
// A receiver that is not ready misses ticks rather than accumulating them, the
// way [time.Ticker] behaves, so a slow consumer falls behind by dropping work
// and not by growing a queue.
type Ticker interface {
	// C is the channel ticks are delivered on.
	C() <-chan time.Time

	// Stop ends the ticker. It does not close C, so a receiver blocked on C
	// stays blocked, which is the same trap [time.Ticker] has.
	Stop()

	// Reset changes the period. The next tick is one period from now.
	Reset(d time.Duration)
}

// Real is the clock that reads the machine.
//
// It is what [From] returns for a context nobody put one in, so this is only
// worth naming when a [Clock] has to be passed somewhere explicitly.
func Real() Clock { return realClock{} }

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (realClock) NewTimer(d time.Duration) Timer         { return realTimer{time.NewTimer(d)} }
func (realClock) NewTicker(d time.Duration) Ticker       { return realTicker{time.NewTicker(d)} }

func (c realClock) Sleep(ctx context.Context, d time.Duration) error { return sleep(ctx, c, d) }

type realTimer struct{ t *time.Timer }

func (t realTimer) C() <-chan time.Time        { return t.t.C }
func (t realTimer) Stop() bool                 { return t.t.Stop() }
func (t realTimer) Reset(d time.Duration) bool { return t.t.Reset(d) }

type realTicker struct{ t *time.Ticker }

func (t realTicker) C() <-chan time.Time   { return t.t.C }
func (t realTicker) Stop()                 { t.t.Stop() }
func (t realTicker) Reset(d time.Duration) { t.t.Reset(d) }

// sleep is Sleep for any clock, since the difference between a real wait and a
// fake one is entirely in the timer.
func sleep(ctx context.Context, c Clock, d time.Duration) error {
	// A caller who has already given up does not get a free pass because the
	// duration happened to be zero.
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}

	t := c.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// key is the context slot the clock lives in. It is a type of its own, so no
// other package can reach the value or overwrite it by accident.
type key struct{}

// With returns a context that reads the time from c.
//
// A test puts a [FakeClock] in at the top and everything under it agrees about
// what time it is. Nothing else in mizu calls this: middleware and test helpers
// put a clock in, and the rest of the code reads it out.
func With(ctx context.Context, c Clock) context.Context {
	return context.WithValue(ctx, key{}, c)
}

// From is the clock a context carries, or the real one if it carries none.
//
// It never returns nil, so a caller does not have to check, and code written
// against it works in production without anybody having set anything up.
func From(ctx context.Context) Clock {
	if c, ok := ctx.Value(key{}).(Clock); ok {
		return c
	}
	return realClock{}
}

// Now is the current time according to the context.
//
// This is the call that replaces [time.Now] everywhere in mizu.
func Now(ctx context.Context) time.Time { return From(ctx).Now() }

// Since is how long ago t was, according to the context.
func Since(ctx context.Context, t time.Time) time.Duration { return From(ctx).Since(t) }

// After is a channel that receives once after d, on the context's clock.
func After(ctx context.Context, d time.Duration) <-chan time.Time { return From(ctx).After(d) }

// NewTimer is a timer on the context's clock.
func NewTimer(ctx context.Context, d time.Duration) Timer { return From(ctx).NewTimer(d) }

// NewTicker is a ticker on the context's clock.
func NewTicker(ctx context.Context, d time.Duration) Ticker { return From(ctx).NewTicker(d) }

// Sleep waits for d on the context's clock, or until the context is done.
//
// It returns nil when the wait finished and the context's error when the caller
// gave up first, which is the difference between this and [time.Sleep] and the
// reason a shutdown does not have to outlast the longest sleep in the program.
func Sleep(ctx context.Context, d time.Duration) error { return From(ctx).Sleep(ctx, d) }
