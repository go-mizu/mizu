package web

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/bot/feature/agent"
	"github.com/go-mizu/mizu/blueprints/bot/feature/gateway"
	"github.com/go-mizu/mizu/blueprints/bot/feature/session"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/store"
)

// NewServer creates the HTTP server with all routes.
func NewServer(s store.Store, devMode bool) *mizu.Router {
	// Create services
	agentSvc := agent.NewService(s)
	sessionSvc := session.NewService(s)
	provider := llm.NewClaude()
	gatewaySvc := gateway.NewService(s, provider)

	// Create handlers
	agentHandler := api.NewAgentHandler(agentSvc)
	channelHandler := api.NewChannelHandler(s)
	sessionHandler := api.NewSessionHandler(sessionSvc)
	messageHandler := api.NewMessageHandler(s)
	gatewayHandler := api.NewGatewayHandler(gatewaySvc, s)
	webhookHandler := api.NewWebhookHandler(gatewaySvc)

	// Create router
	r := mizu.NewRouter()

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

	return r
}
