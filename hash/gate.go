package hash

import (
	"context"
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// A password hash is the one request handler that is meant to be expensive, and
// the expense is memory. At the default parameters every hash in flight holds
// 19 MiB for as long as it runs, so a hundred people signing in at once is
// 1.9 GiB that appears in under a second and is not there in the heap profile
// anybody looked at.
//
// What happens next is an out of memory kill, which takes down every request on
// the process and not only the logins. The answer is to decide beforehand how
// many hashes may run at once, and to make the rest wait and then give up. A
// login that answers 503 after five seconds is a bad afternoon. A process that
// is killed is an outage.
//
// The limit is not a knob anybody has to find. It comes from GOMEMLIMIT and the
// configured memory cost, and it is on by default.

// gate is how many hashes may run at once.
type gate struct {
	slots   chan struct{}
	timeout time.Duration
}

// newGate returns a gate holding n slots, waiting at most timeout for one.
func newGate(n int, timeout time.Duration) *gate {
	return &gate{slots: make(chan struct{}, n), timeout: timeout}
}

// enter waits for a slot and returns the call that gives it back.
//
// Once a hash starts it runs to the end. Cancelling has to happen here, in the
// queue, because there is no way to stop the work in the middle and the memory
// is already held: a caller that walked away is one more reason to finish and
// release it, not to hold it longer.
func (g *gate) enter(ctx context.Context) (func(), error) {
	release := func() { <-g.slots }

	// The common case is a free slot, and it should not cost a timer.
	select {
	case g.slots <- struct{}{}:
		return release, nil
	default:
	}

	t := time.NewTimer(g.timeout)
	defer t.Stop()

	select {
	case g.slots <- struct{}{}:
		return release, nil
	case <-t.C:
		return nil, errs.Newf(errs.Unavailable, "hash.busy",
			"hash: waited %s for one of the %d hashes already running to finish", g.timeout, cap(g.slots))
	case <-ctx.Done():
		return nil, errs.Wrap(ctx.Err(), errs.KindOf(ctx.Err()), "hash.canceled",
			"hash: gave up waiting to start hashing")
	}
}

// concurrency is how many hashes may run at once when nobody said.
//
// Two things bound it. Memory is the one that matters: a quarter of GOMEMLIMIT
// is what hashing may hold, which leaves the rest of the process the three
// quarters it was already using. Processors bound it as well, because argon2id
// is busy the whole time it runs, and eight of them on four cores do not finish
// sooner than four do. They finish later, all together, which is the shape of a
// timeout for everybody instead of a queue for some.
//
// The processor bound counts a hash as one, and a hash of several lanes runs
// them at the same time. So a deployment raising Params.Lanes should think
// about Params.MaxConcurrent as well, since the two multiply.
//
// With no GOMEMLIMIT set there is nothing in the standard library that says how
// much memory the machine has, so the processor bound is the whole answer. A
// deployment that cares should set GOMEMLIMIT, which is worth doing for reasons
// that have nothing to do with this.
func concurrency(memory int, limit int64, cpus int) int {
	n := cpus

	if limit != math.MaxInt64 {
		if fits := limit / 4 / (int64(memory) * 1024); fits < int64(n) {
			n = int(fits)
		}
	}
	if n < 1 {
		// One at a time is slow and it still answers, which is more than the
		// alternative does.
		return 1
	}
	return n
}

// memoryLimit is GOMEMLIMIT, and reading it does not change it.
func memoryLimit() int64 {
	return debug.SetMemoryLimit(-1)
}

// cpus is how many processors the program may use, which is what GOMAXPROCS
// says and not what the machine has.
func cpus() int {
	return runtime.GOMAXPROCS(0)
}
