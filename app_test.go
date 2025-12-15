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

func TestApp_HealthzHandler_OK(t *testing.T) {
	a := New()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rr := httptest.NewRecorder()
	a.HealthzHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Body.String(); got != "ok\n" {
		t.Fatalf("body = %q, want %q", got, "ok\n")
	}
}

func TestApp_HealthzHandler_ShuttingDown(t *testing.T) {
	a := New()
	a.shuttingDown.Store(true)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rr := httptest.NewRecorder()
	a.HealthzHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), "shutting down") {
		t.Fatalf("body = %q, want contains %q", rr.Body.String(), "shutting down")
	}
}

func TestApp_newServer_DefaultTimeouts(t *testing.T) {
	a := New()
	srv := a.newServer("127.0.0.1:0")

	if srv.Handler == nil {
		t.Fatalf("srv.Handler is nil")
	}
	if srv.Addr == "" {
		t.Fatalf("srv.Addr is empty")
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", srv.ReadHeaderTimeout, 5*time.Second)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want %s", srv.IdleTimeout, 60*time.Second)
	}
}

func TestApp_serveContext_ReturnsServeError(t *testing.T) {
	a := New()
	srv := a.newServer("127.0.0.1:0")
	want := errors.New("boom")

	err := a.serveContext(context.Background(), srv, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestApp_serveContext_IgnoresErrServerClosed(t *testing.T) {
	a := New()
	srv := a.newServer("127.0.0.1:0")

	err := a.serveContext(context.Background(), srv, func() error { return http.ErrServerClosed })
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestApp_serveContext_ShutdownGraceful(t *testing.T) {
	a := New()
	a.PreShutdownDelay = 0
	a.ShutdownTimeout = 2 * time.Second

	l := mustListen(t)
	defer func() { _ = l.Close() }()

	srv := a.newServer(l.Addr().String())
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "hi\n")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- a.serveContext(ctx, srv, func() error { return srv.Serve(l) })
	}()

	waitHTTP200(t, "http://"+l.Addr().String()+"/")

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveContext err = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for shutdown")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rr := httptest.NewRecorder()
	a.HealthzHandler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func mustListen(t *testing.T) net.Listener {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return l
}

func waitHTTP200(t *testing.T, url string) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		//nolint:gosec // G107: test hits a local ephemeral listener URL
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("server not ready: %s", url)
}

func waitBool(t *testing.T, d time.Duration, f func() bool) {
	t.Helper()

	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for condition")
}
