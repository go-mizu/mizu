package mizu

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	a := New()

	if a.Router == nil {
		t.Fatalf("Router is nil")
	}
	if a.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("ShutdownTimeout = %v, want %v", a.ShutdownTimeout, defaultShutdownTimeout)
	}
	if a.Server() != nil {
		t.Fatalf("Server() = %v, want nil before starting", a.Server())
	}
	if a.Logger() == nil {
		t.Fatalf("Logger() is nil")
	}
}

func TestLivezHandler_AlwaysOK(t *testing.T) {
	a := New()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)

	a.LivezHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
	if rr.Body.String() != healthOKBody {
		t.Fatalf("body = %q, want %q", rr.Body.String(), healthOKBody)
	}
}

func TestReadyzHandler_OK_WhenNotShuttingDown(t *testing.T) {
	a := New()
	a.shuttingDown.Store(false)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	a.ReadyzHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != healthOKBody {
		t.Fatalf("body = %q, want %q", rr.Body.String(), healthOKBody)
	}
}

func TestReadyzHandler_503_WhenShuttingDown(t *testing.T) {
	a := New()
	a.shuttingDown.Store(true)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	a.ReadyzHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), "shutting down") {
		t.Fatalf("body = %q, want contains %q", rr.Body.String(), "shutting down")
	}
}

func TestNewServer_SetsServerAndTimeouts(t *testing.T) {
	a := New()
	srv := a.newServer(":0")

	if a.Server() != srv {
		t.Fatalf("Server() != newly created server")
	}
	if srv.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if srv.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", srv.IdleTimeout, defaultIdleTimeout)
	}
	if srv.Handler == nil {
		t.Fatalf("Handler is nil")
	}
	if srv.BaseContext == nil {
		t.Fatalf("BaseContext is nil")
	}
}

func TestRunServer_ReturnsError(t *testing.T) {
	a := New()
	errCh := make(chan error, 1)

	want := errors.New("boom")
	a.runServer(errCh, func() error { return want })

	got := <-errCh
	if !errors.Is(got, want) {
		t.Fatalf("err = %v, want %v", got, want)
	}
}

func TestRunServer_IgnoresErrServerClosed(t *testing.T) {
	a := New()
	errCh := make(chan error, 1)

	a.runServer(errCh, func() error { return http.ErrServerClosed })

	got := <-errCh
	if got != nil {
		t.Fatalf("err = %v, want nil", got)
	}
}

func TestRunServer_Nil(t *testing.T) {
	a := New()
	errCh := make(chan error, 1)

	a.runServer(errCh, func() error { return nil })

	got := <-errCh
	if got != nil {
		t.Fatalf("err = %v, want nil", got)
	}
}

func TestServeContext_ReturnsStartError(t *testing.T) {
	a := New()
	srv := a.newServer(":0")

	want := errors.New("start failed")
	err := a.serveContext(context.Background(), srv, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestServeContext_ShutdownOnContextCancel(t *testing.T) {
	a := New()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	srv := a.newServer(l.Addr().String())

	// Use a simple handler so the server can accept a request.
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.serveContext(ctx, srv, func() error { return srv.Serve(l) })
	}()

	waitForHTTP(t, "http://"+l.Addr().String()+"/")

	// Trigger shutdown path.
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("serveContext err = %v, want nil", err)
	}

	// After shutdown started, readiness should be 503 (serveContext flips it).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	a.ReadyzHandler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestShutdownServer_GracefulTimeout_Closes(t *testing.T) {
	a := New()
	a.ShutdownTimeout = 1 * time.Millisecond

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	srv := a.newServer(l.Addr().String())

	block := make(chan struct{})
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	})

	errCh := make(chan error, 1)
	go a.runServer(errCh, func() error { return srv.Serve(l) })

	// Ensure an in-flight request exists so Shutdown has something to wait on.
	clientErr := make(chan error, 1)
	go func() {
		_, err := http.Get("http://" + l.Addr().String() + "/")
		clientErr <- err
	}()

	waitForTCP(t, l.Addr().String())

	// ctx is already done, matching how serveContext calls shutdownServer.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should hit the "graceful shutdown incomplete" branch due to timeout,
	// then Close, and finally return when Serve exits.
	if err := a.shutdownServer(ctx, srv, errCh, a.Logger()); err != nil {
		t.Fatalf("shutdownServer err = %v, want nil", err)
	}

	close(block)

	// Request should fail due to Close/Shutdown. We only assert it returns.
	<-clientErr
}

func TestShutdownServer_DefaultTimeoutWhenNonPositive(t *testing.T) {
	a := New()
	a.ShutdownTimeout = 0

	srv := &http.Server{ReadHeaderTimeout: time.Second}
	errCh := make(chan error, 1)
	errCh <- nil

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.shutdownServer(ctx, srv, errCh, a.Logger()); err != nil {
		t.Fatalf("shutdownServer err = %v, want nil", err)
	}
}

func TestShutdownServer_ReturnsServeExitError(t *testing.T) {
	a := New()
	a.ShutdownTimeout = 1 * time.Millisecond

	srv := &http.Server{ReadHeaderTimeout: time.Second}
	errCh := make(chan error, 1)
	want := errors.New("serve exit error")
	errCh <- want

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := a.shutdownServer(ctx, srv, errCh, a.Logger())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestShutdownServer_DoesNotHangIfServeNeverReturns(t *testing.T) {
	a := New()
	a.ShutdownTimeout = 1 * time.Millisecond

	// Not started server: Shutdown returns quickly (ErrServerClosed),
	// then we hit the "did not exit after shutdown timeout" select branch.
	srv := &http.Server{ReadHeaderTimeout: time.Second}
	errCh := make(chan error) // never receives

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := a.shutdownServer(ctx, srv, errCh, a.Logger()); err != nil {
		t.Fatalf("shutdownServer err = %v, want nil", err)
	}
	if time.Since(start) < serverExitGrace {
		t.Fatalf("shutdownServer returned too fast, want it to wait ~serverExitGrace")
	}
}

func waitForTCP(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server not reachable on %s", addr)
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:gosec // URL is constructed in tests
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server not reachable at %s", url)
}
