//go:build !windows

package mizu

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func (a *App) serveWithSignals(srv *http.Server, serveFn func() error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.serveContext(ctx, srv, serveFn)
}
