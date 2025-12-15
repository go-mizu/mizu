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

// App owns the HTTP server lifecycle and embeds Router.
type App struct {
	*Router

	// PreShutdownDelay is a small delay between flipping unready and starting shutdown.
	// Default: 1s.
	PreShutdownDelay time.Duration

	// ShutdownTimeout is the maximum graceful drain window.
	// Default: 15s.
	ShutdownTimeout time.Duration

	shuttingDown atomic.Bool
}

// New creates an App with sane defaults.
func New() *App {
	return &App{
		Router:           NewRouter(),
		PreShutdownDelay: 1 * time.Second,
		ShutdownTimeout:  15 * time.Second,
	}
}

// Logger returns the router logger.
func (a *App) Logger() *slog.Logger { return a.Router.Logger() }

// HealthzHandler reports health and readiness.
// It returns 503 once shutdown has started.
func (a *App) HealthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if a.shuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
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

	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	srv.BaseContext = func(net.Listener) context.Context { return baseCtx }

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
		return a.shutdownServer(ctx, srv, errCh, cancelBase, log)
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
	cancelBase context.CancelFunc,
	log *slog.Logger,
) error {
	start := time.Now()
	a.shuttingDown.Store(true)
	log.Info("shutdown initiated")

	a.waitPreShutdownDelay(ctx)

	timeout := a.ShutdownTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	drainCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(drainCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("graceful shutdown incomplete", slog.Any("error", err))
		_ = srv.Close()
	}

	cancelBase()

	if err := <-errCh; err != nil {
		log.Error("server exit error after shutdown", slog.Any("error", err))
		return err
	}

	log.Info("server stopped gracefully", slog.Duration("duration", time.Since(start)))
	return nil
}

func (a *App) waitPreShutdownDelay(ctx context.Context) {
	if d := a.PreShutdownDelay; d > 0 {
		t := time.NewTimer(d)
		select {
		case <-t.C:
		case <-ctx.Done():
		}
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
	}
}

func (a *App) newServer(addr string) *http.Server {
	// Set timeouts in the literal to satisfy gosec (G112).
	// Keep Read/WriteTimeout unset to avoid breaking streaming.
	return &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
