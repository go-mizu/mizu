//go:build !windows

package mizu

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestApp_serveWithSignals_SIGTERM_TriggersGracefulShutdown(t *testing.T) {
	a := New()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	srv := a.newServer(l.Addr().String())
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	done := make(chan error, 1)
	go func() {
		done <- a.serveWithSignals(srv, func() error { return srv.Serve(l) })
	}()

	waitForTCP(t, l.Addr().String())

	// Send SIGTERM to our own process. serveWithSignals uses signal.NotifyContext,
	// so the signal should cancel the context and start graceful shutdown
	// instead of terminating the test process.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill(SIGTERM): %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveWithSignals returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for serveWithSignals to return")
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	a.ReadyzHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), "shutting down") {
		t.Fatalf("readyz body = %q, want contains %q", rr.Body.String(), "shutting down")
	}
}

func TestApp_serveWithSignals_PropagatesServeError(t *testing.T) {
	a := New()

	want := context.Canceled // any non-ErrServerClosed error is fine
	err := a.serveWithSignals(a.newServer(":0"), func() error { return want })
	if err != want {
		t.Fatalf("err = %v, want %v", err, want)
	}
}
