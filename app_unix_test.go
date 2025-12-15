//go:build !windows

package mizu

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestApp_serveContext_ShutdownTimeoutForcesClose(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 30 * time.Millisecond

	l := mustListen(t)
	defer func() { _ = l.Close() }()

	srv := a.newServer(l.Addr().String())

	var entered atomic.Bool
	block := make(chan struct{})

	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered.Store(true)
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		<-block
		_, _ = io.WriteString(w, "done\n")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- a.serveContext(ctx, srv, func() error { return srv.Serve(l) })
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	respCh := make(chan error, 1)
	go func() {
		//nolint:gosec // G107: test hits a local ephemeral listener URL
		resp, err := client.Get("http://" + l.Addr().String() + "/block")
		if err != nil {
			respCh <- err
			return
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		respCh <- nil
	}()

	waitBool(t, 2*time.Second, entered.Load)

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveContext err = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		close(block)
		t.Fatalf("timeout waiting for shutdown")
	}

	close(block)

	select {
	case <-respCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for client")
	}

	if !a.shuttingDown.Load() {
		t.Fatalf("shuttingDown = false, want true")
	}
}
