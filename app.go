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

	preShutdownDelay time.Duration
	shutdownTimeout  time.Duration

	shuttingDown atomic.Bool
}

// AppOption configures App.
type AppOption func(*App)

// WithPreShutdownDelay sets a delay between flipping unready and starting shutdown.
func WithPreShutdownDelay(d time.Duration) AppOption {
	return func(a *App) {
		if d >= 0 {
			a.preShutdownDelay = d
		}
	}
}

// WithShutdownTimeout sets the maximum graceful drain window.
func WithShutdownTimeout(d time.Duration) AppOption {
	return func(a *App) {
		if d > 0 {
			a.shutdownTimeout = d
		}
	}
}

// New creates an App with sane defaults.
func New(opts ...AppOption) *App {
	a := &App{
		Router:           NewRouter(),
		preShutdownDelay: 1 * time.Second,
		shutdownTimeout:  15 * time.Second,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Logger returns the router logger.
func (a *App) Logger() *slog.Logger { return a.Router.Logger() }

// HealthzHandler reports readiness and liveness.
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
	srv := &http.Server{ //nolint:gosec // G112: Users can configure timeouts via Server struct directly
		Addr:    addr,
		Handler: a,
	}
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServe() })
}

// ListenTLS starts an HTTPS server and manages signals.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	srv := &http.Server{ //nolint:gosec // G112: Users can configure timeouts via Server struct directly
		Addr:    addr,
		Handler: a,
	}
	return a.serveWithSignals(srv, func() error { return srv.ListenAndServeTLS(certFile, keyFile) })
}

// Serve serves on an existing listener and manages signals.
func (a *App) Serve(l net.Listener) error {
	srv := &http.Server{ //nolint:gosec // G112: Users can configure timeouts via Server struct directly
		Addr:    l.Addr().String(),
		Handler: a,
	}
	return a.serveWithSignals(srv, func() error { return srv.Serve(l) })
}

// ServeContext runs the server with a parent context and graceful shutdown.
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
			log.Warn("graceful shutdown incomplete", slog.Any("error", err))
			_ = srv.Close()
			cancelBase()
		} else {
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
