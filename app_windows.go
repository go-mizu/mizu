//go:build windows || js || wasip1

package mizu

import (
	"context"
	"net/http"
)

func (a *App) serveWithSignals(srv *http.Server, serveFn func() error) error {
	// Signals not reliably injectable. Run under plain context.
	return a.ServeContext(context.Background(), srv, serveFn)
}
