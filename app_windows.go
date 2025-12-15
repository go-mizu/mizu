//go:build windows

package mizu

import (
	"context"
	"net/http"
)

func (a *App) serveWithSignals(srv *http.Server, serveFn func() error) error {
	// Signals not reliably injectable. Run under plain context.
	return a.serveContext(context.Background(), srv, serveFn)
}
