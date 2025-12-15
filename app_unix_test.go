//go:build !windows

package mizu

import (
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestApp_ServeWithSignals_SIGTERM_ShutsDown(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 250 * time.Millisecond

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	srv := a.newServer(l.Addr().String())
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	done := make(chan error, 1)
	go func() {
		done <- a.serveWithSignals(srv, func() error { return srv.Serve(l) })
	}()

	// Wait until the server responds (means serve loop is running),
	// then send SIGTERM to trigger NotifyContext cancellation.
	client := &http.Client{Timeout: 1 * time.Second}

	deadline := time.Now().Add(1 * time.Second)
	for {
		resp, e := client.Get("http://" + l.Addr().String() + "/")
		if e == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become ready: %v", e)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Trigger shutdown via signal.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill(SIGTERM): %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveWithSignals returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveWithSignals did not return in time")
	}

	if !a.shuttingDown.Load() {
		t.Fatalf("shuttingDown = false, want true after signal shutdown")
	}
}
