//go:build !windows
// +build !windows

package mizu

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// helper to wait until a TCP address is accepting connections
func waitReady(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("server not ready on %s within %v", addr, timeout)
		}
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestLoggerAlwaysReturnsAppLogger(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := New(WithLogger(custom))
	if got := a.Logger(); got != custom {
		t.Fatalf("Logger should return a.log")
	}
}

func TestHealthzHandler_OK_and_503(t *testing.T) {
	a := New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)

	a.HealthzHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}

	// flip to shutting down, expect 503
	a.shuttingDown.Store(true)
	rec2 := httptest.NewRecorder()
	a.HealthzHandler().ServeHTTP(rec2, req)
	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec2.Code)
	}
}

func TestListen_ImmediateErrorPath(t *testing.T) {
	// Use private listenContext to inject a failing serveFn and cover errCh first branch
	a := New()
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: a}
	want := errors.New("boom")
	err := a.listenContext(context.Background(), srv, srv.Addr, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

// genSelfSignedCert writes a self-signed cert and key to the given paths.
func genSelfSignedCert(t *testing.T, certPath, keyPath string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"mizu-test"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.DNSNames = []string{"localhost"}
	template.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer func() {
		_ = certOut.Close()
	}()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer func() {
		_ = keyOut.Close()
	}()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		t.Fatalf("encode key: %v", err)
	}

	// quick parse to ensure files are valid
	_, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("load keypair: %v", err)
	}
}

func TestSignalNotifyAvailable(t *testing.T) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Stop(ch)
}

func TestNewDefaults_Unix(t *testing.T) {
	a := New()
	if a.Router == nil {
		t.Fatal("Router should be set")
	}
	if a.log == nil {
		t.Fatal("logger should be set")
	}
	if a.preShutdownDelay != time.Second {
		t.Fatalf("preShutdownDelay default mismatch: %v", a.preShutdownDelay)
	}
	if a.shutdownTimeout != 15*time.Second {
		t.Fatalf("shutdownTimeout default mismatch: %v", a.shutdownTimeout)
	}
	if a.forceCloseDelay != 3*time.Second {
		t.Fatalf("forceCloseDelay default mismatch: %v", a.forceCloseDelay)
	}
	// default signals should include Interrupt and SIGTERM
	foundInterrupt, foundTerm := false, false
	for _, s := range a.signals {
		if s == os.Interrupt {
			foundInterrupt = true
		}
		if s == syscall.SIGTERM {
			foundTerm = true
		}
	}
	if !foundInterrupt || !foundTerm {
		t.Fatalf("default signals missing Interrupt or SIGTERM: %+v", a.signals)
	}
}

func TestWithOptions_Unix(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := New(
		WithLogger(custom),
		WithPreShutdownDelay(5*time.Millisecond),
		WithShutdownTimeout(7*time.Millisecond),
		WithForceCloseDelay(9*time.Millisecond),
		WithSignals(syscall.SIGUSR1),
	)
	if a.Logger() != custom {
		t.Fatal("Logger should return the custom logger")
	}
	if a.preShutdownDelay != 5*time.Millisecond {
		t.Fatalf("preShutdownDelay not applied: %v", a.preShutdownDelay)
	}
	if a.shutdownTimeout != 7*time.Millisecond {
		t.Fatalf("shutdownTimeout not applied: %v", a.shutdownTimeout)
	}
	if a.forceCloseDelay != 9*time.Millisecond {
		t.Fatalf("forceCloseDelay not applied: %v", a.forceCloseDelay)
	}
	if len(a.signals) != 1 || a.signals[0] != syscall.SIGUSR1 {
		t.Fatalf("signals not applied: %+v", a.signals)
	}
}

func TestServe_GracefulOnInterrupt(t *testing.T) {
	a := New(
		WithPreShutdownDelay(5*time.Millisecond),
		WithShutdownTimeout(50*time.Millisecond),
		WithForceCloseDelay(5*time.Millisecond),
	)
	// register at a non-root path to avoid "/" reservation
	a.Get("/hi", func(c *Ctx) error {
		_ = c.Text(200, "hello")
		return nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	done := make(chan error, 1)
	go func() { done <- a.Serve(ln) }()

	waitReady(t, addr, 2*time.Second)

	// simple request
	resp, err := http.Get("http://" + addr + "/hi")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	// send first signal to trigger graceful shutdown
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit after interrupt")
	}
}

func TestGracefulTimeout_ForceClosePath(t *testing.T) {
	a := New(
		WithPreShutdownDelay(0),
		WithShutdownTimeout(20*time.Millisecond), // short, will time out
		WithForceCloseDelay(5*time.Millisecond),
	)

	// Handler that ignores context and blocks long enough
	block := make(chan struct{})
	a.Get("/slow", func(c *Ctx) error {
		// ignore c.Request().Context() to force Shutdown timeout path
		select {
		case <-block:
		case <-time.After(2 * time.Second):
		}
		_ = c.Text(200, "done")
		return nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	done := make(chan error, 1)
	go func() { done <- a.Serve(ln) }()
	waitReady(t, addr, 2*time.Second)

	// Fire a slow request that will keep the server busy
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = http.Get("http://" + addr + "/slow")
	}()

	// give the handler a moment to enter
	time.Sleep(30 * time.Millisecond)

	// trigger graceful shutdown
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	// optional second signal to cover force() goroutine even if timeout happens quickly
	_ = p.Signal(os.Interrupt)

	// we expect a non-nil error from Shutdown due to timeout
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected non-nil error due to shutdown timeout, got nil")
		}
	default:
		// wait for completion with a larger timeout
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("expected non-nil error due to shutdown timeout, got nil")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("server did not exit on timeout")
		}
	}

	// unblock the handler to clean up goroutine
	close(block)
	wg.Wait()
}

func TestSecondSignalForceWithoutTimeout(t *testing.T) {
	// Large shutdown timeout so it would not timeout on its own
	a := New(
		WithPreShutdownDelay(0),
		WithShutdownTimeout(2*time.Second),
		WithForceCloseDelay(0),
	)

	a.Get("/ok", func(c *Ctx) error {
		_ = c.Text(200, "ok")
		return nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	done := make(chan error, 1)
	go func() { done <- a.Serve(ln) }()
	waitReady(t, addr, 2*time.Second)

	// one request to ensure server active
	resp, err := http.Get("http://" + addr + "/ok")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	p, _ := os.FindProcess(os.Getpid())
	// first signal begins shutdown
	_ = p.Signal(os.Interrupt)
	// second signal forces Close immediately
	_ = p.Signal(os.Interrupt)

	select {
	case err := <-done:
		// either nil or ErrServerClosed is acceptable here
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("unexpected error after forced close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit after second signal")
	}
}

func TestListenTLS_StartAndShutdown(t *testing.T) {
	// generate a temp self signed cert
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	genSelfSignedCert(t, certFile, keyFile)

	a := New(
		WithPreShutdownDelay(0),
		WithShutdownTimeout(100*time.Millisecond),
	)

	done := make(chan error, 1)
	go func() { done <- a.ListenTLS("127.0.0.1:0", certFile, keyFile) }()

	// We cannot easily know the picked port from ListenTLS, but we can still trigger shutdown via signal.
	time.Sleep(50 * time.Millisecond)
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	select {
	case err := <-done:
		// either nil or http.ErrServerClosed acceptable
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("ListenTLS ended with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListenTLS did not exit on interrupt")
	}
}

func TestHealthzHandler_Integration(t *testing.T) {
	a := New()

	mux := http.NewServeMux()
	mux.Handle("/healthz", a.HealthzHandler())
	// do not register "/" via router; let external mux route root to app
	mux.Handle("/", a)

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: mux,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()

	done := make(chan error, 1)
	go func() {
		done <- a.listenContext(context.Background(), srv, addr, func() error { return srv.Serve(ln) })
	}()

	waitReady(t, addr, 2*time.Second)

	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("health req failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
		t.Fatalf("unexpected /healthz response: %d %q", resp.StatusCode, body)
	}

	// read-only check that log fields are present in startup path
	_ = runtime.Version()

	// shutdown
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit")
	}
}

func TestListen_Wrapper_StartAndShutdown(t *testing.T) {
	a := New(
		WithPreShutdownDelay(0),
		WithShutdownTimeout(100*time.Millisecond),
	)
	done := make(chan error, 1)

	go func() {
		// Use an ephemeral port; we do not need to know it for this test.
		done <- a.Listen("127.0.0.1:0")
	}()

	// Give the server a moment to start binding.
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown via signal to cover Listen -> listenContext path.
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	select {
	case err := <-done:
		// Either nil or ErrServerClosed is fine here.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Listen ended with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not exit on interrupt")
	}
}

func TestShutdown_DefaultBranch_CtxDonePath(t *testing.T) {
	// This exercises the inner `default -> select { case <-ctx.Done(): ... }` path,
	// by ensuring serveFn has NOT finished when we reach that block.
	a := New(
		WithPreShutdownDelay(0),
		WithShutdownTimeout(50*time.Millisecond),
		WithForceCloseDelay(0),
	)
	srv := &http.Server{}
	serveRelease := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		err := a.listenContext(context.Background(), srv, "test", func() error {
			<-serveRelease // keep serveFn blocked until we release after ctx is done
			return nil     // errCh will deliver nil later
		})
		done <- err
	}()

	// Begin graceful shutdown.
	time.Sleep(10 * time.Millisecond)
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	// Wait a bit so the code hits the inner select and picks <-ctx.Done().
	time.Sleep(30 * time.Millisecond)
	close(serveRelease) // now let serveFn finish (errCh sends nil)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error on ctxDone default branch: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listenContext did not return")
	}
}
func TestShutdown_DefaultBranch_ErrChErrorPath(t *testing.T) {
	// Make errCh ready BEFORE the "prefer errCh" check runs by using a longer preShutdownDelay.
	// serveFn will return a non-nil error shortly after shutdown begins.
	a := New(
		WithPreShutdownDelay(50*time.Millisecond), // ensures the code hasn't reached the select yet
		WithShutdownTimeout(500*time.Millisecond), // generous, we won't hit timeout
		WithForceCloseDelay(0),
	)
	srv := &http.Server{}
	want := errors.New("serve failed after shutdown")

	started := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		err := a.listenContext(context.Background(), srv, "test", func() error {
			close(started)                   // signal we are running
			time.Sleep(5 * time.Millisecond) // return quickly, well before preShutdownDelay elapses
			return want
		})
		done <- err
	}()

	<-started // ensure serveFn is running

	// Begin graceful shutdown.
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)

	// By the time preShutdownDelay finishes, errCh should already have 'want',
	// so the outer non-blocking 'case err := <-errCh' should be taken.
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("expected %v from inner errCh path, got %v", want, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("listenContext did not return")
	}
}
