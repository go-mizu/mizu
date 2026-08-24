package conc

import "sync"

// Once returns a function that runs fn the first time it is called and gives
// back the same result every time after that.
//
//	region := conc.Once(func() (string, error) {
//		return metadata.Region(context.Background())
//	})
//
// Callers that arrive while the first one is still running wait for it, so fn
// runs exactly once no matter how many goroutines want the answer.
//
// The error is part of the result. A failure is remembered and handed to every
// later caller rather than letting the next one try again, because a function
// that sometimes runs twice is not a Once. Something that should be attempted
// again is a retry, and [github.com/go-mizu/mizu/try.Value] is where that
// lives.
//
// A panic in fn comes back as an error, the same as it does from a [Group], and
// is remembered like any other failure. [sync.OnceValues] re-panics on every
// call instead, which is the right answer when the caller has nowhere to put an
// error and the wrong one here.
func Once[T any](fn func() (T, error)) func() (T, error) {
	var (
		mu      sync.Mutex
		started bool

		// done is closed once the result is settled, and is what the callers
		// who arrived second wait on.
		//
		// sync.Once would be the obvious thing to build this out of. The reason
		// it is not here is that the callers it holds up are blocked on a mutex,
		// and a goroutine blocked on a mutex is not durably blocked, so a
		// testing/synctest bubble containing one stops its clock instead of
		// advancing it. Waiting on a channel is, and this package is written to
		// be tested there.
		done = make(chan struct{})

		v   T
		err error
	)

	return func() (T, error) {
		mu.Lock()
		mine := !started
		started = true
		mu.Unlock()

		if !mine {
			<-done
			return v, err
		}

		// The recover goes around fn rather than around the whole call, so that
		// a panic still leaves through this function with a result rather than
		// with the zero value a deferred recover would hand back.
		func() {
			defer func() {
				if p := recover(); p != nil {
					err = recovered(p)
				}
			}()
			v, err = fn()
		}()

		// Written first, closed second, so everybody waiting wakes to a result
		// that is already there.
		close(done)
		return v, err
	}
}
