// app.go
package mizu

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// App owns the HTTP server lifecycle and embeds Router.
// It favors the standard library for graceful shutdown.
// Extras kept small: readiness flip, optional pre-shutdown delay, structured logs.
type App struct {
	*Router

	preShutdownDelay time.Duration // wait after marking unready
	shutdownTimeout  time.Duration // max drain window

	shuttingDown atomic.Bool // exposed by HealthzHandler
	log          *slog.Logger
}

// AppOption configures App.
type AppOption func(*App)

// WithLogger sets the logger. If nil, slog.Default is used.
func WithLogger(l *slog.Logger) AppOption {
	return func(a *App) {
		if l != nil {
			a.log = l
		}
	}
}

// WithPreShutdownDelay sets the delay after flipping readiness and before Shutdown.
func WithPreShutdownDelay(d time.Duration) AppOption {
	return func(a *App) {
		if d >= 0 {
			a.preShutdownDelay = d
		}
	}
}

// WithShutdownTimeout sets the maximum duration for http.Server.Shutdown.
func WithShutdownTimeout(d time.Duration) AppOption {
	return func(a *App) {
		if d > 0 {
			a.shutdownTimeout = d
		}
	}
}

// New creates an App with conservative defaults.
func New(opts ...AppOption) *App {
	r := NewRouter()
	a := &App{
		Router:           r,
		preShutdownDelay: 1 * time.Second,
		shutdownTimeout:  15 * time.Second,
		log:              r.Logger(),
	}
	for _, o := range opts {
		o(a)
	}
	if a.log == nil {
		a.log = slog.Default()
	}
	return a
}

// Logger returns the app logger.
func (a *App) Logger() *slog.Logger { return a.log }

// HealthzHandler reports 200 while serving and 503 after shutdown begins.
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

// Listen starts an HTTP server at addr and handles SIGINT and SIGTERM.
func (a *App) Listen(addr string) error {
	srv := &http.Server{Addr: addr, Handler: a}
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServe() })
}

// ListenTLS starts an HTTPS server and handles SIGINT and SIGTERM.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	srv := &http.Server{Addr: addr, Handler: a}
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServeTLS(certFile, keyFile) })
}

// Serve serves on a custom listener and handles SIGINT and SIGTERM.
func (a *App) Serve(l net.Listener) error {
	srv := &http.Server{Addr: l.Addr().String(), Handler: a}
	return a.serveWithSignals(srv, func() error { return srv.Serve(l) })
}

// ServeContext runs the server until ctx is canceled, then performs a graceful drain.
func (a *App) ServeContext(ctx context.Context, srv *http.Server, serveFn func() error) error {
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
	go func() {
		if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Error("server start failed", slog.Any("error", err))
		}
		return err

	case <-ctx.Done():
		start := time.Now()
		a.shuttingDown.Store(true)
		log.Info("shutdown initiated")

		if a.preShutdownDelay > 0 {
			time.Sleep(a.preShutdownDelay)
		}

		drainCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(drainCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Grace period expired or other failure. Close and cancel base to nudge handlers.
			log.Warn("graceful shutdown incomplete", slog.Any("error", err))
			_ = srv.Close()
			cancelBase()
		} else {
			// Drain completed. Cancel base to release any background waiters tied to BaseContext.
			cancelBase()
		}

		if err := <-errCh; err != nil {
			log.Error("server exit error after shutdown", slog.Any("error", err))
			return err
		}

		log.Info("server stopped gracefully", slog.Duration("duration", time.Since(start)))
		return nil
	}
}

// serveWithSignals wraps ServeContext with a signal-aware parent context.
func (a *App) serveWithSignals(srv *http.Server, serveFn func() error) error {
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.ServeContext(parent, srv, serveFn)
}
