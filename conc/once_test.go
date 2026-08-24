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

func TestOnceRunsOnce(t *testing.T) {
	var calls atomic.Int64
	region := conc.Once(func() (string, error) {
		calls.Add(1)
		return "eu-west-1", nil
	})

	for range 5 {
		v, err := region()
		if v != "eu-west-1" || err != nil {
			t.Fatalf("got %q, %v", v, err)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("fn ran %d times, want 1", n)
	}
}

// TestOnceRemembersTheError is the decision worth pinning down. A failure is a
// result like any other, so the second caller is told the same thing as the
// first rather than being made to try again.
func TestOnceRemembersTheError(t *testing.T) {
	var calls atomic.Int64
	load := conc.Once(func() (int, error) {
		calls.Add(1)
		return 0, first
	})

	for range 3 {
		if _, err := load(); !errors.Is(err, first) {
			t.Fatalf("got %v, want %v", err, first)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("a failure ran fn %d times, want 1", n)
	}
}

func TestOncePanicBecomesAnError(t *testing.T) {
	var calls atomic.Int64
	load := conc.Once(func() (int, error) {
		calls.Add(1)
		return 0, boom()
	})

	_, err := load()
	if errs.CodeOf(err) != "panic" {
		t.Fatalf("got %v, want a recovered panic", err)
	}
	if !strings.Contains(err.Error(), "kaboom") {
		t.Errorf("the panic value is not in %q", err)
	}

	// The panic is the result, so the second caller gets it too rather than
	// running a function that has already been shown to fail.
	if _, again := load(); again != err {
		t.Errorf("the second call got %v, want the same error", again)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("fn ran %d times, want 1", n)
	}
}

// TestOnceKeepsTheStack checks the trace points at the line that panicked and
// not at the recover that caught it.
func TestOnceKeepsTheStack(t *testing.T) {
	load := conc.Once(func() (int, error) { return 0, boom() })
	_, err := load()

	stack := errs.Stack(err)
	if len(stack) == 0 {
		t.Fatal("the error carries no stack")
	}
	var found bool
	for _, f := range stack {
		if strings.HasSuffix(f.Func, "conc_test.boom") {
			found = true
		}
	}
	if !found {
		t.Errorf("boom is not in the stack: %v", stack)
	}
}

// TestOnceHoldsUpEveryoneElse is the property that separates this from a flag
// and a check. The second caller waits for the first rather than seeing a
// half-built value or running fn alongside it.
func TestOnceHoldsUpEveryoneElse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls, done atomic.Int64
		gate := make(chan struct{})

		load := conc.Once(func() (int, error) {
			calls.Add(1)
			<-gate
			return 7, nil
		})

		g, _ := conc.NewGroup(t.Context())
		for range 20 {
			g.Go(func(context.Context) error {
				v, err := load()
				if v != 7 || err != nil {
					t.Errorf("got %d, %v", v, err)
				}
				done.Add(1)
				return nil
			})
		}

		synctest.Wait()
		if n := done.Load(); n != 0 {
			t.Errorf("%d callers got an answer before fn returned", n)
		}

		close(gate)
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}
		if n := calls.Load(); n != 1 {
			t.Errorf("20 goroutines ran fn %d times, want 1", n)
		}
		if n := done.Load(); n != 20 {
			t.Errorf("%d of 20 callers got an answer", n)
		}
	})
}
