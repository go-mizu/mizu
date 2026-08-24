package hash

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/go-mizu/mizu/errs"
)

// TestConcurrency is the arithmetic behind the limit. It is worth stating
// rather than deriving, because the number decides whether a burst of logins is
// a queue or an out of memory kill.
func TestConcurrency(t *testing.T) {
	const gib = 1 << 30

	cases := []struct {
		about  string
		memory int
		limit  int64
		cpus   int
		want   int
	}{
		{"no limit set, so the processors decide", 19456, math.MaxInt64, 8, 8},
		{"a quarter of 1 GiB at 19 MiB a hash", 19456, gib, 32, 13},
		{"the same, with fewer processors", 19456, gib, 4, 4},
		{"a small container holds one", 19456, 128 << 20, 8, 1},
		{"a container too small for even one", 19456, 8 << 20, 8, 1},
		{"cheap hashes are bounded by processors", 64, 4 * gib, 8, 8},
		{"a machine with plenty of both", 19456, 64 * gib, 16, 16},
	}

	for _, c := range cases {
		if got := concurrency(c.memory, c.limit, c.cpus); got != c.want {
			t.Errorf("%s: %d, want %d", c.about, got, c.want)
		}
	}
}

// TestConcurrencyIsAlwaysAtLeastOne is the property the arithmetic must not
// lose. A limit of zero is a process that answers nothing at all, which is
// worse than the failure it is guarding against.
func TestConcurrencyIsAlwaysAtLeastOne(t *testing.T) {
	for _, limit := range []int64{0, 1, 1 << 20, 1 << 30, math.MaxInt64} {
		for _, cpus := range []int{0, 1, 2, 128} {
			if got := concurrency(1<<20, limit, cpus); got < 1 {
				t.Errorf("limit %d and %d processors gives %d", limit, cpus, got)
			}
		}
	}
}

// TestReadsTheEnvironment covers the two calls that ask the runtime rather than
// the caller. What they answer depends on the machine, so what is checked is
// that they answer something usable.
func TestReadsTheEnvironment(t *testing.T) {
	if memoryLimit() < 1 {
		t.Errorf("GOMEMLIMIT reads as %d", memoryLimit())
	}
	if cpus() < 1 {
		t.Errorf("GOMAXPROCS reads as %d", cpus())
	}
	if n := concurrency(19456, memoryLimit(), cpus()); n < 1 {
		t.Errorf("this machine allows %d hashes at once", n)
	}
}

func TestGateLetsWorkThrough(t *testing.T) {
	g := newGate(2, time.Second)

	first, err := g.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	second, err := g.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	first()
	second()

	// And the slots come back, so the gate is not a counter that runs out.
	for range 10 {
		release, err := g.enter(t.Context())
		if err != nil {
			t.Fatalf("enter: %v", err)
		}
		release()
	}
}

// TestGateWaits is the queue doing its job: the third caller does not run, it
// waits, and it gets in as soon as somebody leaves.
func TestGateWaits(t *testing.T) {
	g := newGate(1, 5*time.Second)

	held, err := g.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}

	waiting := make(chan error, 1)
	go func() {
		release, err := g.enter(t.Context())
		if err == nil {
			release()
		}
		waiting <- err
	}()

	select {
	case err := <-waiting:
		t.Fatalf("the second caller did not wait: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	held()
	if err := <-waiting; err != nil {
		t.Errorf("the second caller never got in: %v", err)
	}
}

// TestGateGivesUp is what a login flood looks like from the outside: an error
// that is a 503 and says so, rather than a request that hangs or a process that
// is killed.
func TestGateGivesUp(t *testing.T) {
	g := newGate(1, time.Millisecond)

	held, err := g.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	defer held()

	release, err := g.enter(t.Context())
	if release != nil {
		t.Error("enter returned a release along with the error")
	}
	if errs.CodeOf(err) != "hash.busy" {
		t.Fatalf("%v, want hash.busy", err)
	}
	if k := errs.KindOf(err); k != errs.Unavailable {
		t.Errorf("the kind is %s, want unavailable", k)
	}
	if got := errs.KindOf(err).Status(); got != 503 {
		t.Errorf("the status is %d, want 503", got)
	}
	if !errs.Retryable(err) {
		t.Error("being busy is not retryable, and it is the one thing that is")
	}
}

// TestGateStopsWaitingWhenTheCallerDoes is the other way out of the queue. A
// client that hung up is not worth 19 MiB.
func TestGateStopsWaitingWhenTheCallerDoes(t *testing.T) {
	g := newGate(1, time.Minute)

	held, err := g.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	defer held()

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()

	release, err := g.enter(ctx)
	if release != nil {
		t.Error("enter returned a release along with the error")
	}
	if errs.CodeOf(err) != "hash.canceled" {
		t.Fatalf("%v, want hash.canceled", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Error("the context error did not survive")
	}

	// A deadline that has already passed is the same path with a different
	// answer, and it is the one an HTTP server with a write timeout produces.
	ctx, cancel = context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	_, err = g.enter(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("%v, want a deadline", err)
	}
	if k := errs.KindOf(err); k != errs.Timeout {
		t.Errorf("the kind is %s, want timeout", k)
	}
}

// TestHashGivesUpWhenBusy is the same thing through the API a handler calls,
// since that is where it has to be right.
func TestHashGivesUpWhenBusy(t *testing.T) {
	h, err := New(Params{Memory: 64, Passes: 1, MaxConcurrent: 1, VerifyTimeout: time.Millisecond})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	encoded, err := h.Hash(t.Context(), "hunter2")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	// Hold the only slot from inside the gate, the way a hash in flight does.
	held, err := h.gate.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	defer held()

	if _, err := h.Hash(t.Context(), "hunter2"); errs.CodeOf(err) != "hash.busy" {
		t.Errorf("Hash: %v, want hash.busy", err)
	}

	ok, err := h.Verify(t.Context(), "hunter2", encoded)
	if errs.CodeOf(err) != "hash.busy" {
		t.Errorf("Verify: %v, want hash.busy", err)
	}
	if ok {
		t.Error("Verify said yes without checking anything")
	}
}

// TestVerifyChecksTheHashBeforeQueueing is the order the two failures happen
// in. A stored value that is not a hash is an answer this has already, and
// waiting five seconds to say so would be five seconds of a slot nobody needed.
func TestVerifyChecksTheHashBeforeQueueing(t *testing.T) {
	h, err := New(Params{Memory: 64, Passes: 1, MaxConcurrent: 1, VerifyTimeout: time.Minute})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	held, err := h.gate.enter(t.Context())
	if err != nil {
		t.Fatalf("enter: %v", err)
	}
	defer held()

	done := make(chan error, 1)
	go func() {
		_, err := h.Verify(t.Context(), "hunter2", "not a hash")
		done <- err
	}()

	select {
	case err := <-done:
		if errs.CodeOf(err) != "hash.malformed" {
			t.Errorf("%v, want hash.malformed", err)
		}
	case <-time.After(time.Second):
		t.Error("Verify queued behind a running hash to reject a value it could see was wrong")
	}
}
