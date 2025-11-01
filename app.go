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
	"sync"
	"sync/atomic"
	"time"
)

// App is your HTTP server container that embeds Router and handles start, logging, readiness, and graceful shutdown.
type App struct {
	*Router

	preShutdownDelay time.Duration // wait after flipping readiness before starting shutdown
	shutdownTimeout  time.Duration // graceful drain window
	forceCloseDelay  time.Duration // extra wait before forced close after grace expires
	signals          []os.Signal   // which signals trigger graceful shutdown
	shuttingDown     atomic.Bool   // readiness flag exposed via HealthzHandler
}

// AppOption lets you tune App in New; zero values are ignored.
type AppOption func(*App)

// WithLogger sets the slog logger for this app; if nil then slog.Default is used.
func WithLogger(l *slog.Logger) AppOption {
	return func(a *App) {
		if l != nil {
			a.log = l
		}
	}
}

// WithPreShutdownDelay waits this long after marking not ready before starting shutdown.
func WithPreShutdownDelay(d time.Duration) AppOption {
	return func(a *App) {
		if d >= 0 {
			a.preShutdownDelay = d
		}
	}
}

// WithShutdownTimeout gives in flight requests up to this long to finish during shutdown.
func WithShutdownTimeout(d time.Duration) AppOption {
	return func(a *App) {
		if d > 0 {
			a.shutdownTimeout = d
		}
	}
}

// WithForceCloseDelay waits this long after the grace period before forcing close.
func WithForceCloseDelay(d time.Duration) AppOption {
	return func(a *App) {
		if d > 0 {
			a.forceCloseDelay = d
		}
	}
}

// WithSignals chooses which OS signals start graceful shutdown.
func WithSignals(sigs ...os.Signal) AppOption {
	return func(a *App) {
		if len(sigs) > 0 {
			a.signals = sigs
		}
	}
}

// New creates an App with sensible defaults.
func New(opts ...AppOption) *App {
	r := NewRouter()
	a := &App{
		Router:           r,
		preShutdownDelay: 1 * time.Second,
		shutdownTimeout:  15 * time.Second,
		forceCloseDelay:  3 * time.Second,
		signals:          defaultSignals(),
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Logger returns the app logger.
func (a *App) Logger() *slog.Logger {
	return a.log
}

// HealthzHandler returns a readiness handler that reports 200 OK until shutdown then 503.
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

// Listen starts an HTTP server on addr and shuts down gracefully on signal.
func (a *App) Listen(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: a,
	}
	return a.listenContext(context.Background(), srv, addr, func() error { return srv.ListenAndServe() })
}

// ListenTLS starts an HTTPS server on addr with cert and key and shuts down gracefully.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: a,
	}
	return a.listenContext(context.Background(), srv, addr, func() error { return srv.ListenAndServeTLS(certFile, keyFile) })
}

// Serve runs the server on a provided listener, useful for tests or custom listeners.
func (a *App) Serve(l net.Listener) error {
	srv := &http.Server{
		Handler: a,
	}
	return a.listenContext(context.Background(), srv, l.Addr().String(), func() error { return srv.Serve(l) })
}

// listenContext manages server lifetime, signal handling, and graceful shutdown.
func (a *App) listenContext(parent context.Context, srv *http.Server, addr string, serveFn func() error) error {
	ongoingCtx, stopOngoing := context.WithCancel(context.Background())
	srv.BaseContext = func(_ net.Listener) context.Context { return ongoingCtx }

	log := a.Logger().With(
		slog.String("addr", addr),
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

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, a.signals...)
	defer signal.Stop(sigCh)

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-sigCh:
			cancel()
		}
	}()

	var once sync.Once
	force := func() {
		once.Do(func() {
			log.Warn("second signal received, forcing close")
			_ = srv.Close()
			stopOngoing()
		})
	}
	go func() {
		<-ctx.Done()
		<-sigCh
		force()
	}()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		start := time.Now()
		a.shuttingDown.Store(true)
		log.Info("shutdown initiated")

		if a.preShutdownDelay > 0 {
			time.Sleep(a.preShutdownDelay)
		}

		srv.SetKeepAlivesEnabled(false)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		done := make(chan struct{})
		var shutdownErr error
		go func() {
			shutdownErr = srv.Shutdown(shutdownCtx)
			close(done)
		}()

		select {
		case <-done:
		case <-shutdownCtx.Done():
			log.Warn("graceful shutdown timed out, waiting before hard close", slog.Duration("wait", a.forceCloseDelay))
			time.Sleep(a.forceCloseDelay)
			force()
			<-errCh
		}

		stopOngoing()

		if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			log.Error("server shutdown error", slog.Any("error", shutdownErr), slog.Duration("duration", time.Since(start)))
			return shutdownErr
		}

		select {
		case err := <-errCh:
			if err != nil {
				log.Error("server exit error after shutdown", slog.Any("error", err), slog.Duration("duration", time.Since(start)))
				return err
			}
		default:
			select {
			case <-ctx.Done():
			case err := <-errCh:
				if err != nil {
					log.Error("server exit error after shutdown", slog.Any("error", err), slog.Duration("duration", time.Since(start)))
					return err
				}
			}
		}

		log.Info("server stopped gracefully", slog.Duration("duration", time.Since(start)))
		return nil
	}
}
