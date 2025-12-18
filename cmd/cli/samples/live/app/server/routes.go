package server

import (
	"example.com/live/handler"
)

func (a *App) routes() {
	// Static files
	a.app.Mount("/static/", staticHandler(a.cfg.Dev))

	// Page routes
	a.app.Get("/", handler.Home())
	a.app.Get("/counter", handler.Counter())

	// WebSocket for live updates
	a.app.Mount("/ws", a.liveServer.Handler())
}
