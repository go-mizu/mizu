package conc

import (
	"sync"
	"time"
)

// Debounce returns a function that runs fn once the calls have stopped coming.
//
//	reindex := conc.Debounce(200*time.Millisecond, rebuildSearchIndex)
//	for range fileChanges {
//		reindex()
//	}
//
// Every call pushes the run d further out, so a burst of a thousand calls
// arriving faster than d apart produces one run, d after the last of them. This
// is for work that only the latest version of matters: rebuilding an index,
// reloading a config file, saving a draft.
//
// The returned function is safe to call from several goroutines and returns
// straight away. fn runs later, on a timer's own goroutine, so a panic in it
// takes the process down the way any panic outside a [Group] does. Debounce
// something small that cannot fail, and when the work is neither, hand it to a
// group from inside:
//
//	save := conc.Debounce(time.Second, func() {
//		conc.Go(ctx, persist)
//	})
//
// fn never runs alongside itself. A call that lands while it is running
// schedules another one, and anything the newer call has already superseded is
// dropped rather than queued.
func Debounce(d time.Duration, fn func()) func() {
	var (
		mu sync.Mutex
		t  *time.Timer

		// gen counts the calls, so a timer that has already fired can tell
		// whether it is still the most recent one. Stop is not enough on its
		// own: a timer that fired a moment ago cannot be stopped, and without
		// this it would run fn on its way out.
		gen uint64
	)

	// run keeps fn to one at a time. A timer waiting here is usually one that
	// is about to find out it is stale, which is why the check comes after the
	// wait rather than before it.
	//
	// It is a channel and not a mutex because a goroutine waiting on a mutex is
	// not durably blocked, so a testing/synctest bubble holding one would stop
	// its clock rather than advance it. A channel wait is, and this package is
	// written to be tested there.
	run := make(chan struct{}, 1)

	return func() {
		mu.Lock()
		defer mu.Unlock()

		gen++
		mine := gen
		if t != nil {
			t.Stop()
		}
		t = time.AfterFunc(d, func() {
			run <- struct{}{}
			defer func() { <-run }()

			mu.Lock()
			stale := gen != mine
			mu.Unlock()
			if stale {
				return
			}
			fn()
		})
	}
}

// Throttle returns a function that runs fn at most once every d.
//
//	report := conc.Throttle(time.Second, publishProgress)
//	for _, row := range rows {
//		process(row)
//		report()
//	}
//
// The first call runs fn. Calls in the next d are dropped, and the one after
// that runs it again. This is for work where the newest call is the one worth
// making rather than the last one: a progress bar, a heartbeat, a metric that
// nobody needs at loop speed.
//
// The calls in between are dropped and not queued. Something that runs every
// call but spaces them out is a rate limiter, which holds the caller back
// instead of letting it through.
//
// fn runs on the goroutine that called the returned function, so it panics
// where the caller can see it and this is the one place in the package where a
// panic is not turned into an error. [Debounce] cannot do the same, since
// running later is the whole point of it.
func Throttle(d time.Duration, fn func()) func() {
	var (
		mu   sync.Mutex
		open = true
	)

	return func() {
		mu.Lock()
		if !open {
			mu.Unlock()
			return
		}
		open = false
		// A timer rather than a stored time, so there is no clock to read and
		// no wall clock jumping backwards to reason about. Inside a
		// testing/synctest bubble this is the fake clock, which is what makes
		// the behaviour testable rather than approximately testable.
		time.AfterFunc(d, func() {
			mu.Lock()
			open = true
			mu.Unlock()
		})
		mu.Unlock()

		// Outside the lock, so fn can call the throttled function itself and be
		// turned away rather than deadlock.
		fn()
	}
}
