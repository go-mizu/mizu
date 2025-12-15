package mizu

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	defaultShutdownTimeout = 15 * time.Second

	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second

	// Extra time to wait for the serve loop to return after Shutdown completes.
	serverExitGrace = 1 * time.Second

	healthOKBody = "ok\n"
)

// App owns the HTTP server lifecycle and embeds Router.
type App struct {
	*Router

	// ShutdownTimeout is the maximum graceful drain window.
	// Default: 15s.
	ShutdownTimeout time.Duration

	shuttingDown atomic.Bool
	server       *http.Server
}

// New creates an App with sane defaults.
func New() *App {
	return &App{
		Router:          NewRouter(),
		ShutdownTimeout: defaultShutdownTimeout,
	}
}

// Logger returns the router logger.
func (a *App) Logger() *slog.Logger { return a.Router.Logger() }

// Server returns the current server instance created by Listen/ListenTLS/Serve.
// Configure it before starting the server (TLSConfig, ConnState, ErrorLog, etc).
func (a *App) Server() *http.Server { return a.server }

// LivezHandler reports process liveness.
// It stays 200 during shutdown, unless you want restarts.
func (a *App) LivezHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, healthOKBody)
	})
}

// ReadyzHandler reports readiness.
// It returns 503 once shutdown has started.
func (a *App) ReadyzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if a.shuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, healthOKBody)
	})
}

// Listen starts an HTTP server on addr and manages signals.
func (a *App) Listen(addr string) error {
	srv := a.newServer(addr)
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServe() })
}

// ListenTLS starts an HTTPS server and manages signals.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	srv := a.newServer(addr)
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServeTLS(certFile, keyFile) })
}

// Serve serves on an existing listener and manages signals.
func (a *App) Serve(l net.Listener) error {
	srv := a.newServer(l.Addr().String())
	return a.serveWithSignals(srv, func() error { return srv.Serve(l) })
}

// serveContext runs the server with a parent context and graceful shutdown.
func (a *App) serveContext(ctx context.Context, srv *http.Server, serveFn func() error) error {
	// Reset readiness for reuse in tests.
	a.shuttingDown.Store(false)

	log := a.Logger().With(
		slog.String("addr", srv.Addr),
		slog.Int("pid", os.Getpid()),
		slog.String("go_version", runtime.Version()),
	)
	log.Info("server starting")

	errCh := make(chan error, 1)
	go a.runServer(errCh, serveFn)

	select {
	case err := <-errCh:
		if err != nil {
			log.Error("server start failed", slog.Any("error", err))
		}
		return err
	case <-ctx.Done():
		return a.shutdownServer(ctx, srv, errCh, log)
	}
}

func (a *App) runServer(errCh chan<- error, serveFn func() error) {
	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- err
		return
	}
	errCh <- nil
}

func (a *App) shutdownServer(
	ctx context.Context,
	srv *http.Server,
	errCh <-chan error,
	log *slog.Logger,
) error {
	start := time.Now()
	a.shuttingDown.Store(true)
	log.Info("shutdown initiated")

	timeout := a.ShutdownTimeout
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}

	// Use Background so we still attempt a graceful drain even though ctx is
	// already cancelled by the signal. The timeout bounds the drain window.
	drainCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(drainCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("graceful shutdown incomplete", slog.Any("error", err))
		_ = srv.Close()
	}

	// Never block forever waiting for serveFn to return.
	select {
	case err := <-errCh:
		if err != nil {
			log.Error("server exit error after shutdown", slog.Any("error", err))
			return err
		}
	case <-time.After(timeout + serverExitGrace):
		log.Warn("server did not exit after shutdown timeout")
	}

	log.Info("server stopped gracefully", slog.Duration("duration", time.Since(start)))
	return nil
}

func (a *App) newServer(addr string) *http.Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}

	// Preserve user overrides: only set BaseContext if still nil.
	if srv.BaseContext == nil {
		baseCtx := context.Background()
		srv.BaseContext = func(net.Listener) context.Context { return baseCtx }
	}

	a.server = srv
	return srv
}
