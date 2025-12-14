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
	defaultPreShutdownDelay  = 1 * time.Second
	defaultShutdownTimeout   = 15 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	minShutdownDrainBudget   = 1 * time.Millisecond
)

// App owns the HTTP server lifecycle and embeds Router.
type App struct {
	*Router

	server *http.Server

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

// WithShutdownTimeout sets the maximum graceful drain window (including pre-shutdown delay).
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
		preShutdownDelay: defaultPreShutdownDelay,
		shutdownTimeout:  defaultShutdownTimeout,
		server: &http.Server{
			// Safe defaults that do not break streaming/SSE by default.
			ReadHeaderTimeout: defaultReadHeaderTimeout,
			IdleTimeout:       defaultIdleTimeout,
		},
	}
	for _, o := range opts {
		o(a)
	}

	// Always route through the app (router).
	a.server.Handler = a
	return a
}

// Logger returns the router logger.
func (a *App) Logger() *slog.Logger { return a.Router.Logger() }

// Server returns the underlying http.Server so callers can customize timeouts,
// TLS config, error log, ConnContext, etc.
func (a *App) Server() *http.Server { return a.server }

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
	a.server.Addr = addr
	return a.serveWithSignals(a.server, func() error { return a.server.ListenAndServe() })
}

// ListenTLS starts an HTTPS server and manages signals.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	a.server.Addr = addr
	return a.serveWithSignals(a.server, func() error { return a.server.ListenAndServeTLS(certFile, keyFile) })
}

// Serve serves on an existing listener and manages signals.
func (a *App) Serve(l net.Listener) error {
	a.server.Addr = l.Addr().String()
	return a.serveWithSignals(a.server, func() error { return a.server.Serve(l) })
}

// ServeContext runs the server with a parent context and graceful shutdown.
func (a *App) ServeContext(ctx context.Context, server *http.Server, serveFn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if server == nil {
		server = a.server
	}
	if serveFn == nil {
		serveFn = func() error { return server.ListenAndServe() }
	}

	baseCtx, cancelBase := context.WithCancel(ctx)
	defer cancelBase()
	server.BaseContext = func(net.Listener) context.Context { return baseCtx }

	log := a.Logger().With(
		slog.String("addr", server.Addr),
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
			log.Error("server exited", slog.Any("error", err))
		}
		return err

	case <-ctx.Done():
		start := time.Now()
		a.shuttingDown.Store(true)
		log.Info("shutdown initiated")

		// Keep total shutdown budget bounded by shutdownTimeout.
		delay := a.preShutdownDelay
		if delay < 0 {
			delay = 0
		}
		if delay > a.shutdownTimeout {
			delay = a.shutdownTimeout
		}
		if delay > 0 {
			time.Sleep(delay)
		}

		drainBudget := a.shutdownTimeout - delay
		if drainBudget <= 0 {
			drainBudget = minShutdownDrainBudget
		}

		drainCtx, cancel := context.WithTimeout(context.Background(), drainBudget)
		defer cancel()

		if err := server.Shutdown(drainCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn("graceful shutdown incomplete", slog.Any("error", err))
			_ = server.Close()
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
