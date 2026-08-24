// Package clock is what time it is, taken from the context.
//
// The rule in mizu is that code does not call [time.Now]. A cache entry that
// expires, a rate limit that refills, a token that runs out, a schedule that
// fires and a retry that backs off all ask the context instead:
//
//	now := clock.Now(ctx)
//
// By default that is the real clock, a few nanoseconds more than calling
// time.Now yourself, and the section on cost below has the numbers. A test puts
// a different one in:
//
//	c := clock.Fake(time.Date(2026, 3, 31, 23, 55, 0, 0, time.UTC))
//	ctx = clock.With(ctx, c)
//
//	c.Advance(10 * time.Minute) // and it is April
//
// The payoff is tests that assert what happens at the end of a billing period,
// or after a token expires, without sleeping and without waiting for the
// calendar.
//
// # This is not a replacement for testing/synctest
//
// [testing/synctest] runs a bubble of goroutines on a clock that only moves
// when every goroutine in it is blocked. It is the right tool for anything
// about ordering: a timeout that must fire before another, a worker that must
// stop when its context is canceled, a rate limiter under load. It handles real
// [time.Sleep] and real timers, so code under test does not have to be written
// for it.
//
// What it does not do is let you choose what day it is. A bubble starts at
// midnight on 2000-01-01 and moves forward, so a test about the last day of a
// month, a leap day, or a token issued three weeks ago has nowhere to put that
// fact.
//
// So the two answer different questions. Use synctest when the test is about
// concurrency and duration. Use a fake clock when the test is about a date.
// They compose: a [FakeClock] inside a synctest bubble is fine, since advancing
// it is a plain function call and never blocks.
//
// # Timers
//
// [Clock.NewTimer] and [Clock.NewTicker] return interfaces rather than
// [time.Timer] and [time.Ticker], because a struct with an exported channel
// field cannot be implemented twice. The channel is a method:
//
//	t := clock.NewTimer(ctx, time.Second)
//	defer t.Stop()
//
//	select {
//	case <-t.C():
//	case <-ctx.Done():
//	}
//
// A fake timer behaves the way the real one does, including dropping ticks that
// a slow receiver missed rather than delivering them in a burst. Advancing a
// fake clock past ten ticker periods delivers one tick, which is what
// [time.Ticker] does and what code written against it already expects.
//
// # Sleeping
//
// [Sleep] takes a context and returns an error, unlike [time.Sleep], which
// takes neither and cannot be interrupted. A sleep that ignores cancellation is
// a shutdown that takes as long as the longest sleep in the program.
//
//	if err := clock.Sleep(ctx, backoff); err != nil {
//		return err // the caller gave up, and the error says so
//	}
//
// The error is [context.Canceled] or [context.DeadlineExceeded] unwrapped, so
// errors.Is reaches it and errs.KindOf classifies it without help.
//
// # Cost
//
// Reading the clock out of a context is one context.Value lookup and an
// interface call, and it allocates nothing.
//
// On an M4, [time.Now] takes about 32 nanoseconds and [Now] takes about 36. The
// clock reads the same hardware either way, so what the four nanoseconds buy is
// the lookup and the call, and that is the whole of the overhead. A context
// eight middleware deep with no clock in it comes to about 38, since the lookup
// walks to the bottom before defaulting to the real clock. A context with a
// clock near the top is about 34, because the walk stops there.
//
// So there is no reason to thread a [Clock] through a signature to avoid this.
// A loop reading the time a million times pays four milliseconds for it, and a
// loop reading the time a million times has a bigger problem than that.
//
// A [FakeClock] is faster than the real one, at about 8 nanoseconds for a
// [FakeClock.Now], since it reads a field under a mutex rather than the
// hardware. [FakeClock.Advance] scans the waiters, which is about 190
// nanoseconds with a hundred timers set. A test that sets a hundred thousand of
// them would notice; nothing else will.
package clock
