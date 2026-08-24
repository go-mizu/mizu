package try

import (
	"math/rand/v2"
	"time"
)

// A Backoff is how long to wait after an attempt failed. The attempt is
// numbered from one, so the first wait is Backoff(1).
//
// It is an [Option] as well as a type, so a strategy is passed straight to
// [Do] and a custom one needs no wrapper:
//
//	try.Do(ctx, fn, try.Backoff(func(attempt int) time.Duration {
//		return time.Duration(attempt) * time.Second
//	}))
//
// A negative or zero duration is no wait at all, which is what a test that does
// not care about the timing wants.
type Backoff func(attempt int) time.Duration

func (b Backoff) apply(p *policy) {
	if b == nil {
		panic("try: a nil Backoff")
	}
	p.backoff = b
}

// Exponential doubles the wait each time, starting at base and stopping at max.
//
//	try.Exponential(100*time.Millisecond, 30*time.Second)
//	// 100ms, 200ms, 400ms, 800ms, ... 30s, 30s, 30s
//
// Use it inside [Jitter] rather than on its own. On its own it is the reason a
// hundred clients that failed together come back together, and then fail
// together again.
//
// Both durations must be positive and max must not be below base, or this
// panics, since a backoff that silently does nothing is worse than one that
// does not compile past the first test run.
func Exponential(base, max time.Duration) Backoff {
	switch {
	case base <= 0:
		panic("try: Exponential with a base that is not positive")
	case max < base:
		panic("try: Exponential with a max below the base")
	}
	return func(attempt int) time.Duration {
		if attempt < 1 {
			return base
		}
		// Shifting by 63 or more is undefined, and the cap makes anything past
		// the point where base doubles past max come out the same anyway.
		if shift := attempt - 1; shift < 62 {
			if d := base << shift; d >= base && d < max {
				return d
			}
		}
		return max
	}
}

// Jitter spreads a backoff over the whole interval up to what b asked for.
//
//	try.Jitter(try.Exponential(100*time.Millisecond, 30*time.Second))
//	// somewhere in [0, 100ms), then [0, 200ms), then [0, 400ms) ...
//
// This is full jitter, and it is the default for a reason worth stating. The
// point of backing off is not to be polite, it is to break up the crowd: a
// service that fails a thousand requests at once has a thousand clients whose
// timers now agree, and a backoff without jitter keeps them agreeing all the
// way down. Waiting a random amount up to the interval is what takes the spike
// apart, and the measurements say it beats the half-and-half version.
//
// The average wait is half of what b asked for, so a jittered backoff reaches a
// given total wait in about twice as many attempts. That is the trade, and it
// is worth it.
func Jitter(b Backoff) Backoff {
	if b == nil {
		panic("try: Jitter of a nil Backoff")
	}
	return func(attempt int) time.Duration {
		d := b(attempt)
		if d <= 0 {
			return 0
		}
		return rand.N(d)
	}
}
