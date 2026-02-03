package web

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/app/web/handler/api"
	db "github.com/go-mizu/mizu/blueprints/bot/app/web/handler/dashboard"
	"github.com/go-mizu/mizu/blueprints/bot/app/web/rpc"
	"github.com/go-mizu/mizu/blueprints/bot/assets"
	"github.com/go-mizu/mizu/blueprints/bot/feature/agent"
	"github.com/go-mizu/mizu/blueprints/bot/feature/gateway"
	"github.com/go-mizu/mizu/blueprints/bot/feature/session"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	_ "github.com/go-mizu/mizu/blueprints/bot/pkg/channel/telegram" // register telegram driver
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/logring"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// Server wraps the HTTP router and services that need cleanup on shutdown.
type Server struct {
	Router       *mizu.Router
	gateway      *gateway.Service
	Logs         *logring.Ring
	Hub          *db.Hub
	drivers      []channel.Driver
	skillWatcher *skill.Watcher
}

// ServeHTTP delegates to the underlying router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

// Close releases resources held by the server's services.
func (s *Server) Close() {
	ctx := context.Background()
	for _, d := range s.drivers {
		_ = d.Disconnect(ctx)
	}
	if s.skillWatcher != nil {
		s.skillWatcher.Stop()
	}
	if s.gateway != nil {
		s.gateway.Close()
	}
}

// NewServer creates the HTTP server with all routes.
func NewServer(s store.Store, devMode bool) *Server {
	startTime := time.Now()

	// Create log ring buffer
	logs := logring.New(5000)
	logs.Info("gateway", "Server starting")

	// Create services
	agentSvc := agent.NewService(s)
	sessionSvc := session.NewService(s)
	provider := llm.NewClaude()
	gatewaySvc := gateway.NewService(s, provider)

	// Create API handlers
	agentHandler := api.NewAgentHandler(agentSvc)
	channelHandler := api.NewChannelHandler(s)
	sessionHandler := api.NewSessionHandler(sessionSvc)
	messageHandler := api.NewMessageHandler(s)
	gatewayHandler := api.NewGatewayHandler(gatewaySvc, s)
	webhookHandler := api.NewWebhookHandler(gatewaySvc)

	// Create WebSocket hub for dashboard
	// Use GATEWAY_TOKEN env var when set; empty = no auth required (dev-friendly)
	hub := db.NewHub(os.Getenv("GATEWAY_TOKEN"))

	// Wire broadcaster for real-time chat events
	gatewaySvc.SetBroadcaster(hub)

	// Register all RPC methods
	rpc.RegisterAll(hub, s, gatewaySvc, logs, startTime)

	// Create router
	r := mizu.NewRouter()

	// WebSocket endpoint for dashboard real-time communication
	r.Get("/ws", hub.WSHandler())

	// API routes
	r.Get("/api/status", gatewayHandler.Status)
	r.Get("/api/health", gatewayHandler.Health)
	r.Get("/api/commands", gatewayHandler.Commands)

	// Agent CRUD
	r.Get("/api/agents", agentHandler.List)
	r.Post("/api/agents", agentHandler.Create)
	r.Get("/api/agents/{id}", agentHandler.Get)
	r.Put("/api/agents/{id}", agentHandler.Update)
	r.Delete("/api/agents/{id}", agentHandler.Delete)

	// Channel CRUD
	r.Get("/api/channels", channelHandler.List)
	r.Post("/api/channels", channelHandler.Create)
	r.Get("/api/channels/{id}", channelHandler.Get)
	r.Put("/api/channels/{id}", channelHandler.Update)
	r.Delete("/api/channels/{id}", channelHandler.Delete)

	// Session management
	r.Get("/api/sessions", sessionHandler.List)
	r.Get("/api/sessions/{id}", sessionHandler.Get)
	r.Delete("/api/sessions/{id}", sessionHandler.Delete)
	r.Post("/api/sessions/{id}/reset", sessionHandler.Reset)

	// Messages
	r.Get("/api/messages", messageHandler.List)
	r.Post("/api/messages", messageHandler.Send)

	// Bindings
	r.Get("/api/bindings", gatewayHandler.ListBindings)
	r.Post("/api/bindings", gatewayHandler.CreateBinding)
	r.Delete("/api/bindings/{id}", gatewayHandler.DeleteBinding)

	// Webhook endpoints
	r.Post("/api/webhook/{channelId}", webhookHandler.Receive)

	// Gateway message send (direct send via API)
	r.Post("/api/gateway/send", gatewayHandler.Send)

	// Serve frontend
	if devMode {
		r.Get("/{path...}", func(c *mizu.Ctx) error {
			return c.Text(200, "Frontend dev server at http://localhost:5174")
		})
	} else {
		staticContent, _ := fs.Sub(assets.StaticFS, "static")
		indexHTML, _ := fs.ReadFile(staticContent, "index.html")
		fileServer := http.FileServer(http.FS(staticContent))
		r.Get("/{path...}", func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			if path == "/" {
				path = "/index.html"
			}
			if info, err := fs.Stat(staticContent, path[1:]); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(200, string(indexHTML))
		})
	}

	logs.Info("gateway", "Server initialized (dev=%v)", devMode)

	// Start channel drivers (Telegram, etc.)
	var drivers []channel.Driver
	channels, err := s.ListChannels(context.Background())
	if err != nil {
		logs.Warn("gateway", "Failed to list channels: %v", err)
	}
	for _, ch := range channels {
		if ch.Status == "disabled" || ch.Config == "" {
			continue
		}
		// Use a holder so the handler closure can reference the driver
		// after it's created (the closure captures the pointer, not the value).
		var driverRef channel.Driver
		driver, err := channel.New(ch.Type, ch.Config, func(ctx context.Context, msg *types.InboundMessage) error {
			result, err := gatewaySvc.ProcessMessage(ctx, msg)
			if err != nil {
				log.Printf("channel %s message error: %v", ch.Type, err)
				return err
			}
			outMsg := &types.OutboundMessage{
				ChannelType: msg.ChannelType,
				ChannelID:   msg.ChannelID,
				PeerID:      msg.PeerID,
				Content:     result.Content,
				ReplyTo:     msg.ReplyTo,
				ThreadID:    msg.ThreadID,
			}
			return driverRef.Send(ctx, outMsg)
		})
		if err != nil {
			logs.Warn("gateway", "Failed to create %s driver: %v", ch.Type, err)
			continue
		}
		driverRef = driver
		if err := driver.Connect(context.Background()); err != nil {
			logs.Warn("gateway", "Failed to connect %s driver: %v", ch.Type, err)
			continue
		}
		drivers = append(drivers, driver)
		logs.Info("gateway", "Started %s channel driver (%s)", ch.Type, ch.Name)
	}

	// Start skill file watcher — monitors skill directories for changes and
	// broadcasts "skills.updated" to connected dashboard clients.
	// Matches OpenClaw's refresh.ts ensureSkillsWatcher behaviour.
	var watchDirs []string
	watchDirs = append(watchDirs, skill.BundledSkillsDir())
	if home, err := os.UserHomeDir(); err == nil {
		watchDirs = append(watchDirs, home+"/.openbot/skills")
	}
	if rawCfg, err := config.LoadRawConfig(""); err == nil {
		watchDirs = append(watchDirs, skill.ParseExtraDirs(rawCfg)...)
	}
	skillWatcher := skill.StartWatcher(watchDirs, func() {
		logs.Info("skills", "Skill files changed, version=%d", skill.CurrentVersion())
		hub.Broadcast("skills.updated", map[string]any{
			"version": skill.CurrentVersion(),
		})
	})

	return &Server{
		Router:       r,
		gateway:      gatewaySvc,
		Logs:         logs,
		Hub:          hub,
		drivers:      drivers,
		skillWatcher: skillWatcher,
	}
}
