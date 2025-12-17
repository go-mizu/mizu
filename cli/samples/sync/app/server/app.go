package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/live"
	"github.com/go-mizu/mizu/sync"
	synchttp "github.com/go-mizu/mizu/sync/http"
	"github.com/go-mizu/mizu/sync/memory"
	"github.com/go-mizu/mizu/view"

	"example.com/sync/assets"
	"example.com/sync/service/todo"
)

// App holds the application components.
type App struct {
	cfg           Config
	app           *mizu.App
	engine        *view.Engine
	syncEngine    *sync.Engine
	syncTransport *synchttp.Transport
	liveServer    *live.Server
	store         *todo.Store
}

// New creates and configures a new application instance.
func New(cfg Config) *App {
	a := &App{
		cfg:   cfg,
		app:   mizu.New(),
		store: todo.NewStore(),
	}

	// Setup view engine
	a.setupViews()

	// Setup live server first (needed for sync notifier)
	a.setupLive()

	// Setup sync engine with live notifier
	a.setupSync()

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
		viewsFS, _ := fs.Sub(assets.ViewsFS, "views")
		cfg.FS = viewsFS
	} else {
		cfg.Dir = "assets/views"
	}

	a.engine = view.New(cfg)

	if !a.cfg.Dev {
		if err := a.engine.Load(); err != nil {
			panic("failed to load templates: " + err.Error())
		}
	}

	a.app.Use(a.engine.Middleware())
}

// LiveMessage is the application-level protocol envelope.
type LiveMessage struct {
	Type    string          `json:"type"`
	Ref     string          `json:"ref,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (a *App) setupLive() {
	a.liveServer = live.New(live.Options{
		OnMessage: func(ctx context.Context, s *live.Session, topic string, data []byte) {
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
			}
		},
	})
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

func (a *App) setupSync() {
	// Create storage backends
	log := memory.NewLog()
	dedupe := memory.NewDedupe()

	// Create sync engine
	a.syncEngine = sync.New(sync.Options{
		Log:      log,
		Apply:    a.store.Apply,
		Snapshot: a.store.Snapshot,
		Dedupe:   dedupe,
	})

	// Create HTTP transport
	a.syncTransport = synchttp.New(synchttp.Options{
		Engine: a.syncEngine,
	})
}

// notifySync publishes a sync notification via live server.
func (a *App) notifySync(scope string, cursor uint64) {
	topic := "sync:" + scope
	data, _ := json.Marshal(LiveMessage{
		Type:    "sync",
		Payload: json.RawMessage(`{"cursor":` + uintToString(cursor) + `}`),
	})
	a.liveServer.Publish(topic, data)
}

func uintToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
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
