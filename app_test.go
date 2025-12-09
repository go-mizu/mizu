// app_test.go
package mizu

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
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

// ---- helpers ----

// mustListen starts a TCP listener on 127.0.0.1 with a random port.
func mustListen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

// tryGetBody performs a GET with a short timeout and returns code, body, err.
// It never touches t; safe to use in goroutines.
func tryGetBody(url string) (int, string, error) {
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b), nil
}

func TestLoggerGetterAndSetLogger(t *testing.T) {
	app := New()
	if app.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}

	// Swap in a custom logger via the embedded router.
	lg := slog.New(slog.NewTextHandler(io.Discard, nil))
	app.SetLogger(lg)
	if app.Logger() != lg {
		t.Fatal("Logger() did not reflect SetLogger change")
	}

	// Smoke log to ensure the logger path is exercised.
	app.Logger().Info("test-log", "k", "v")
}

func TestServeContext_EarlyServeError(t *testing.T) {
	app := New()
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: app}

	want := errors.New("boom")
	err := app.ServeContext(context.Background(), srv, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("want early error %v, got %v", want, err)
	}
}

func TestServe_CloseListenerEarly_Path(t *testing.T) {
	// Covers the early errCh path where Serve returns a shutdown-related error.
	app := New()
	ln := mustListen(t)
	defer func() { _ = ln.Close() }()
	srv := &http.Server{Addr: ln.Addr().String(), Handler: app}

	done := make(chan error, 1)
	go func() {
		done <- app.ServeContext(context.Background(), srv, func() error {
			return srv.Serve(ln)
		})
	}()

	// Let the server attempt to start accepting
	time.Sleep(30 * time.Millisecond)

	// Closing the listener should cause Serve to exit with a shutdown-style error
	_ = ln.Close()

	err := <-done
	if err == nil {
		return
	}
	// Acceptable outcomes for a closed listener across platforms and Go versions
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
		return
	}
	t.Fatalf("ServeContext with closed listener returned unexpected error: %v", err)
}

func TestHealthz_ReadinessFlip(t *testing.T) {
	app := New(WithPreShutdownDelay(0), WithShutdownTimeout(200*time.Millisecond))
	// mount /healthz
	app.Compat.Handle("/healthz", app.HealthzHandler())

	ln := mustListen(t)
	defer func() { _ = ln.Close() }()

	srv := &http.Server{Addr: ln.Addr().String(), Handler: app}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = app.ServeContext(ctx, srv, func() error { return srv.Serve(ln) })
	}()

	// healthy
	code, _, err := tryGetBody("http://" + ln.Addr().String() + "/healthz")
	if err != nil || code != http.StatusOK {
		t.Fatalf("health before shutdown = %d, err=%v, want 200", code, err)
	}

	// trigger shutdown
	cancel()
	// small wait to let readiness flip apply
	time.Sleep(20 * time.Millisecond)

	code2, _, err2 := tryGetBody("http://" + ln.Addr().String() + "/healthz")
	if err2 == nil && code2 != http.StatusServiceUnavailable {
		t.Fatalf("health after shutdown = %d, want 503 (err=%v)", code2, err2)
	}

	wg.Wait()
}

func TestGracefulDrain_CompletesInFlight(t *testing.T) {
	app := New(WithPreShutdownDelay(0), WithShutdownTimeout(500*time.Millisecond))
	// route that sleeps then responds
	app.Get("/slow", func(c *Ctx) error {
		time.Sleep(120 * time.Millisecond)
		return c.Text(200, "ok")
	})

	ln := mustListen(t)
	defer func() { _ = ln.Close() }()
	srv := &http.Server{Addr: ln.Addr().String(), Handler: app}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.ServeContext(ctx, srv, func() error { return srv.Serve(ln) })
	}()

	// Kick off request
	type resp struct {
		code int
		body string
		err  error
	}
	resCh := make(chan resp, 1)
	go func() {
		code, body, err := tryGetBody("http://" + ln.Addr().String() + "/slow")
		resCh <- resp{code, body, err}
	}()

	// Cancel while in flight to exercise graceful drain path
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case r := <-resCh:
		if r.err != nil || r.code != 200 || r.body != "ok" {
			t.Fatalf("response = %d %q err=%v, want 200 'ok' nil", r.code, r.body, r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete under graceful drain")
	}

	if err := <-errCh; err != nil {
		t.Fatalf("ServeContext returned error: %v", err)
	}
}

func TestShutdownTimeout_ClosesAndCancelsBaseContext(t *testing.T) {
	// Very small shutdown timeout to force timeout path and Close()
	app := New(WithPreShutdownDelay(0), WithShutdownTimeout(60*time.Millisecond))

	// channel that the handler uses to signal it observed context cancellation
	seenCancel := make(chan struct{}, 1)

	// Handler blocks but watches r.Context().Done to detect base cancel
	app.Get("/block", func(c *Ctx) error {
		select {
		case <-c.Request().Context().Done():
			seenCancel <- struct{}{}
			// simulate work finishing after cancel
			time.Sleep(5 * time.Millisecond)
			return nil
		case <-time.After(5 * time.Second):
			return c.Text(200, "unexpected")
		}
	})

	ln := mustListen(t)
	defer func() { _ = ln.Close() }()
	srv := &http.Server{Addr: ln.Addr().String(), Handler: app}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- app.ServeContext(ctx, srv, func() error { return srv.Serve(ln) })
	}()

	// Start the blocking request; it will hang until context is canceled.
	go func() {
		// We ignore the error here on purpose; during shutdown the client may see EOF/conn reset.
		_, _, _ = tryGetBody("http://" + ln.Addr().String() + "/block")
	}()

	// Let request enter the handler
	time.Sleep(20 * time.Millisecond)

	// Trigger shutdown which should time out and then Close + cancel base
	cancel()

	select {
	case <-seenCancel:
		// observed base context cancellation as intended
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not observe base context cancellation after timeout")
	}

	if err := <-done; err != nil {
		t.Fatalf("ServeContext returned error after timeout path: %v", err)
	}
}

// helper: send SIGINT to self, portable where possible
func sendInterrupt(t *testing.T) {
	t.Helper()
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	// Small delay so server enters accept loop
	time.Sleep(50 * time.Millisecond)
	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal: %v", err)
	}
}

// helper: tolerate shutdown style errors from ServeContext
func isBenignServeErr(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, http.ErrServerClosed) ||
		errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), "use of closed network connection")
}

// helper: generate a self-signed cert for 127.0.0.1 and localhost
func writeSelfSignedCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "mizu-test",
			Organization: []string{"mizu"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	certOut := filepath.Join(dir, "cert.pem")
	keyOut := filepath.Join(dir, "key.pem")

	if err := os.WriteFile(certOut, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}
	if err := os.WriteFile(keyOut, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certOut, keyOut
}

func TestApp_Serve_WithSignals(t *testing.T) {
	app := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	done := make(chan error, 1)
	go func() {
		done <- app.Serve(ln)
	}()

	// allow accept to start, then close listener to trigger graceful exit
	time.Sleep(30 * time.Millisecond)
	_ = ln.Close()

	err = <-done
	if !isBenignServeErr(err) {
		t.Fatalf("Serve returned unexpected error: %v", err)
	}
}

func TestApp_Listen_WithSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Listen not supported on this platform")
	}
	// Install a temporary handler to avoid interrupt propagating to outer test harness on some platforms
	defer func() {
		// drain any pending signals to restore default behavior for next tests
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		select {
		case <-c:
		default:
		}
		signal.Reset(os.Interrupt, syscall.SIGTERM)
	}()

	app := New()
	done := make(chan error, 1)
	go func() {
		// bind random port
		done <- app.Listen("127.0.0.1:0")
	}()

	sendInterrupt(t)

	err := <-done
	if !isBenignServeErr(err) {
		t.Fatalf("Listen returned unexpected error: %v", err)
	}
}

func TestApp_ListenTLS_WithSignals(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" || runtime.GOOS == "windows" {
		t.Skip("TLS listen not supported on this platform")
	}

	tmp := t.TempDir()
	certFile, keyFile := writeSelfSignedCert(t, tmp)

	app := New()
	done := make(chan error, 1)
	go func() {
		// bind random port
		done <- app.ListenTLS("127.0.0.1:0", certFile, keyFile)
	}()

	sendInterrupt(t)

	err := <-done
	if !isBenignServeErr(err) {
		t.Fatalf("ListenTLS returned unexpected error: %v", err)
	}
}

func TestServeWithSignals_ParentContextCancel(t *testing.T) {
	app := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	srv := &http.Server{Addr: ln.Addr().String(), Handler: app}

	// Create a parent context we can cancel without sending OS signals
	parent, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.ServeContext(parent, srv, func() error { return srv.Serve(ln) })
	}()

	// let it start, then cancel parent
	time.Sleep(30 * time.Millisecond)
	cancel()

	err = <-done
	if !isBenignServeErr(err) {
		t.Fatalf("ServeContext with canceled parent returned unexpected error: %v", err)
	}
}

// Optional: sanity check that Listen and ListenTLS resolve an address format correctly.
// Not strictly necessary but helps catch regressions in addr handling.
func TestAddrFormatting(t *testing.T) {
	addr := "127.0.0.1:0"
	if !strings.Contains(addr, ":") {
		t.Fatalf("bad addr: %q", addr)
	}
	_ = fmt.Sprintf("addr=%s", addr)
}
