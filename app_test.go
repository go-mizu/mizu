package mizu

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestApp_NewDefaults(t *testing.T) {
	a := New()
	if a == nil {
		t.Fatal("New() returned nil")
	}
	if a.Router == nil {
		t.Fatal("New() Router is nil")
	}
	if got, want := a.PreShutdownDelay, defaultPreShutdownDelay; got != want {
		t.Fatalf("PreShutdownDelay = %v, want %v", got, want)
	}
	if got, want := a.ShutdownTimeout, defaultShutdownTimeout; got != want {
		t.Fatalf("ShutdownTimeout = %v, want %v", got, want)
	}
	if a.Server() != nil {
		t.Fatalf("Server() = %v, want nil before starting", a.Server())
	}
}

func TestApp_NewServer_StoresAndSetsTimeouts(t *testing.T) {
	a := New()
	srv := a.newServer("127.0.0.1:12345")

	if srv == nil {
		t.Fatal("newServer returned nil")
	}
	if got := a.Server(); got != srv {
		t.Fatalf("Server() did not return stored server pointer")
	}
	if got, want := srv.Addr, "127.0.0.1:12345"; got != want {
		t.Fatalf("srv.Addr = %q, want %q", got, want)
	}
	if got, want := srv.ReadHeaderTimeout, defaultReadHeaderTimeout; got != want {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", got, want)
	}
	if got, want := srv.IdleTimeout, defaultIdleTimeout; got != want {
		t.Fatalf("IdleTimeout = %v, want %v", got, want)
	}
	if srv.Handler == nil {
		t.Fatalf("srv.Handler is nil")
	}
}

func TestApp_HealthzHandler_OKAndShuttingDown(t *testing.T) {
	a := New()
	h := a.HealthzHandler()

	// OK path.
	{
		rr := httptestRecorder()
		req, _ := http.NewRequest(http.MethodGet, "http://example/healthz", nil)
		h.ServeHTTP(rr, req)

		if rr.code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.code, http.StatusOK)
		}
		if ct := rr.header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
		}
		if rr.body != healthOKBody {
			t.Fatalf("body = %q, want %q", rr.body, healthOKBody)
		}
	}

	// Shutting down path.
	{
		a.shuttingDown.Store(true)

		rr := httptestRecorder()
		req, _ := http.NewRequest(http.MethodGet, "http://example/healthz", nil)
		h.ServeHTTP(rr, req)

		if rr.code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rr.code, http.StatusServiceUnavailable)
		}
		if ct := rr.header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
		}
		if rr.body == "" {
			t.Fatalf("expected error body, got empty")
		}
	}
}

func TestApp_WaitPreShutdownDelay_NoDelayAndCanceled(t *testing.T) {
	a := New()

	// No delay should return immediately.
	a.PreShutdownDelay = 0
	start := time.Now()
	a.waitPreShutdownDelay(context.Background())
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("waitPreShutdownDelay took too long with no delay")
	}

	// Canceled ctx should return immediately even with delay configured.
	a.PreShutdownDelay = 5 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start = time.Now()
	a.waitPreShutdownDelay(ctx)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("waitPreShutdownDelay took too long with canceled ctx")
	}
}

func TestApp_RunServer_ErrorsAndErrServerClosed(t *testing.T) {
	a := New()

	t.Run("returns_error", func(t *testing.T) {
		errCh := make(chan error, 1)
		a.runServer(errCh, func() error { return errors.New("boom") }, func() {})
		if err := <-errCh; err == nil || err.Error() != "boom" {
			t.Fatalf("err = %v, want boom", err)
		}
	})

	t.Run("treats_ErrServerClosed_as_nil", func(t *testing.T) {
		errCh := make(chan error, 1)
		a.runServer(errCh, func() error { return http.ErrServerClosed }, func() {})
		if err := <-errCh; err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})
}

func TestApp_ServeContext_ServerStartFailure(t *testing.T) {
	a := New()
	srv := a.newServer("127.0.0.1:0")

	want := errors.New("start failed")
	err := a.serveContext(context.Background(), srv, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestApp_ServeContext_ShutdownGraceful(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.serveContext(ctx, srv, func() error { return srv.Serve(l) }) }()

	// Wait until it responds.
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://" + l.Addr().String() + "/")
	if err != nil {
		cancel()
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveContext returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveContext did not return in time")
	}
}

func TestApp_ServeContext_ShutdownTimeoutTriggersClose(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 25 * time.Millisecond

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	block := make(chan struct{})
	srv := a.newServer(l.Addr().String())
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Hold the request open so Shutdown times out.
		<-block
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.serveContext(ctx, srv, func() error { return srv.Serve(l) }) }()

	// Fire a request that will hang in the handler.
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, "http://"+l.Addr().String()+"/", nil)
	go func() {
		_, _ = client.Do(req)
	}()

	// Give the request a moment to reach the handler, then cancel to initiate shutdown.
	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// shutdownServer should swallow the shutdown timeout and still return nil here.
		// It logs a warning and calls Close as a fallback.
		if err != nil {
			t.Fatalf("serveContext returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(block)
		t.Fatal("serveContext did not return in time")
	}

	close(block)
}

func TestApp_ShutdownTimeoutDefaultWhenNonPositive(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 0 // should fall back to defaultShutdownTimeout

	srv := a.newServer("127.0.0.1:0")

	// Provide an errCh that returns promptly so we do not wait for the default timeout.
	errCh := make(chan error, 1)
	errCh <- nil

	err := a.shutdownServer(context.Background(), srv, errCh, func() {}, a.Logger())
	if err != nil {
		t.Fatalf("shutdownServer returned error: %v", err)
	}
}

func TestApp_ShutdownServer_ReturnsExitError(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 50 * time.Millisecond

	srv := a.newServer("127.0.0.1:0")

	boom := errors.New("exit boom")
	errCh := make(chan error, 1)
	errCh <- boom

	err := a.shutdownServer(context.Background(), srv, errCh, func() {}, a.Logger())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

// --- Minimal recorder (portable, no httptest.ResponseRecorder dependency) ---

type miniRecorder struct {
	header http.Header
	code   int
	body   string
}

func httptestRecorder() *miniRecorder {
	return &miniRecorder{header: make(http.Header)}
}

func (r *miniRecorder) Header() http.Header { return r.header }

func (r *miniRecorder) WriteHeader(statusCode int) { r.code = statusCode }

func (r *miniRecorder) Write(p []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	r.body += string(p)
	return len(p), nil
}
