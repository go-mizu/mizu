package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/live"
	"github.com/go-mizu/mizu/view"

	"example.com/live/assets"
	"example.com/live/handler"
)

// App holds the application components.
type App struct {
	cfg        Config
	app        *mizu.App
	engine     *view.Engine
	liveServer *live.Server
	counter    *handler.CounterView
}

// New creates and configures a new application instance.
func New(cfg Config) *App {
	a := &App{
		cfg: cfg,
		app: mizu.New(),
	}

	// Setup view engine
	a.setupViews()

	// Setup live server
	a.setupLive()

	// Setup routes
	a.routes()

	return a
}

// Listen starts the HTTP server.
func (a *App) Listen(addr string) error {
	return a.app.Listen(addr)
}

func (a *App) setupViews() {
	cfg := view.Config{
		DefaultLayout: "default",
		Development:   a.cfg.Dev,
	}

	if !a.cfg.Dev {
		// Production: use embedded filesystem
		viewsFS, _ := fs.Sub(assets.ViewsFS, "views")
		cfg.FS = viewsFS
	} else {
		// Development: use disk filesystem
		cfg.Dir = "assets/views"
	}

	a.engine = view.New(cfg)

	// Load templates in production
	if !a.cfg.Dev {
		if err := a.engine.Load(); err != nil {
			panic("failed to load templates: " + err.Error())
		}
	}

	// Add view middleware
	a.app.Use(a.engine.Middleware())
}

func (a *App) setupLive() {
	// Create counter view handler
	a.counter = handler.NewCounterView()

	// Create live server
	a.liveServer = live.New(live.Options{
		OnMessage: a.handleLiveMessage,
		OnClose: func(s *live.Session, err error) {
			// Cleanup session from all views
			a.counter.RemoveSession(s.ID())
		},
	})
}

// LiveMessage is the application-level protocol envelope.
// The live package only handles transport (topic + data).
// The application defines its own message format in the data field.
type LiveMessage struct {
	Type    string          `json:"type"`
	Ref     string          `json:"ref,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (a *App) handleLiveMessage(ctx context.Context, s *live.Session, topic string, data []byte) {
	var msg LiveMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "subscribe":
		a.liveServer.Subscribe(s, topic)
		a.sendResponse(s, topic, "ack", msg.Ref, nil)
	case "unsubscribe":
		a.liveServer.Unsubscribe(s, topic)
	case "mount":
		a.handleMount(ctx, s, topic, msg)
	case "event":
		a.handleEvent(ctx, s, topic, msg)
	}
}

func (a *App) handleMount(ctx context.Context, s *live.Session, topic string, msg LiveMessage) {
	// Route to appropriate view handler based on topic
	switch topic {
	case "view:counter":
		a.counter.Mount(s, topic, msg.Ref)
	}
}

func (a *App) handleEvent(ctx context.Context, s *live.Session, topic string, msg LiveMessage) {
	// Route to appropriate view handler based on topic
	switch topic {
	case "view:counter":
		a.counter.HandleEvent(s, topic, msg.Payload)
	}
}

// sendResponse sends a response message to the client.
func (a *App) sendResponse(s *live.Session, topic, msgType, ref string, payload any) {
	resp := LiveMessage{Type: msgType, Ref: ref}
	if payload != nil {
		resp.Payload, _ = json.Marshal(payload)
	}
	data, _ := json.Marshal(resp)
	_ = s.Send(live.Message{Topic: topic, Data: data})
}

// staticHandler serves embedded static files
func staticHandler(dev bool) http.Handler {
	var staticFS fs.FS
	if dev {
		staticFS = os.DirFS("assets/static")
	} else {
		staticFS, _ = fs.Sub(assets.StaticFS, "static")
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
}
