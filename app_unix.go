//go:build !windows && !js && !wasip1

package mizu

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
)

func (a *App) serveWithSignals(srv *http.Server, serveFn func() error) error {
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.ServeContext(parent, srv, serveFn)
}
