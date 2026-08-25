//go:build unix

package console

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// raise sends a signal to this process. It is safe in a test because interrupt
// has asked for the signal, so it is delivered to Go rather than killing the
// test binary.
func raise(t *testing.T, sig syscall.Signal) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), sig); err != nil {
		t.Fatalf("sending %v to this process: %v", sig, err)
	}
}

// buffer is a writer that several goroutines can use, which the one interrupt
// starts and the test both do.
type buffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *buffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *buffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func TestTheFirstSignalCancelsTheContext(t *testing.T) {
	var w buffer
	ctx, stop := interrupt(&w, func() { t.Error("the first signal stopped the process") })
	defer stop()

	raise(t, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the context was not cancelled")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("the context ended with %v", ctx.Err())
	}
}

func TestTheSecondSignalStopsTheProcess(t *testing.T) {
	var w buffer
	now := make(chan struct{})
	ctx, stop := interrupt(&w, func() { close(now) })
	defer stop()

	raise(t, syscall.SIGINT)
	<-ctx.Done()

	// The message comes from the goroutine that is also waiting for the second
	// signal, so once it is written that goroutine is ready for it.
	waitFor(t, func() bool { return strings.Contains(w.String(), "shutting down") })

	raise(t, syscall.SIGINT)
	select {
	case <-now:
	case <-time.After(5 * time.Second):
		t.Fatal("the second signal did not stop the process")
	}
}

func TestStopEndsTheSignalHandling(t *testing.T) {
	var w buffer
	_, stop := interrupt(&w, func() { t.Error("a signal arrived after stop") })
	stop()

	// SIGURG is the one signal a process ignores by default, so raising it here
	// cannot end the test binary now that nothing is diverting signals. The
	// point is what does not happen: no goroutine left waiting, no message.
	raise(t, syscall.SIGURG)
	time.Sleep(50 * time.Millisecond)

	if w.String() != "" {
		t.Errorf("it said %q after stop", w.String())
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if ok() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting")
}
