package api

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/gorilla/websocket"
)

// RealtimeHandler handles realtime WebSocket endpoints.
type RealtimeHandler struct {
	store    *postgres.Store
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mu       sync.RWMutex
}

// NewRealtimeHandler creates a new realtime handler.
func NewRealtimeHandler(store *postgres.Store) *RealtimeHandler {
	return &RealtimeHandler{
		store: store,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// SEC-016: Configure CheckOrigin to validate WebSocket origins
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				// Allow connections without origin header (e.g., CLI tools)
				if origin == "" {
					return true
				}
				// Allow localhost for development
				if strings.HasPrefix(origin, "http://localhost") ||
					strings.HasPrefix(origin, "https://localhost") ||
					strings.HasPrefix(origin, "http://127.0.0.1") ||
					strings.HasPrefix(origin, "https://127.0.0.1") {
					return true
				}
				// Check allowed origins from environment
				allowedOrigins := os.Getenv("LOCALBASE_ALLOWED_ORIGINS")
				if allowedOrigins == "" {
					// Default: allow same origin only in production
					return false
				}
				for _, allowed := range strings.Split(allowedOrigins, ",") {
					if strings.TrimSpace(allowed) == origin {
						return true
					}
				}
				return false
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

// ListChannels lists active channels.
func (h *RealtimeHandler) ListChannels(c *mizu.Ctx) error {
	channels, err := h.store.Realtime().ListChannels(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list channels"})
	}

	return c.JSON(200, channels)
}

// GetStats returns realtime connection stats.
func (h *RealtimeHandler) GetStats(c *mizu.Ctx) error {
	h.mu.RLock()
	connCount := len(h.clients)
	h.mu.RUnlock()

	channels, _ := h.store.Realtime().ListChannels(c.Context())

	return c.JSON(200, map[string]any{
		"connections": connCount,
		"channels":    len(channels),
		"server_time": time.Now().Format(time.RFC3339),
	})
}

// WebSocket handles WebSocket connections.
// SEC-017: WebSocket authentication via token query parameter or API key
func (h *RealtimeHandler) WebSocket(c *mizu.Ctx) error {
	// Check authentication before upgrading
	// WebSocket can't use standard headers easily, so we check query params
	token := c.Query("apikey")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		// Also check standard header (set before upgrade)
		token = c.Request().Header.Get("apikey")
	}

	// Validate token
	role := middleware.GetRole(c)
	if role == "" && token != "" {
		// Try to validate the token manually
		config := middleware.DefaultAPIKeyConfig()
		if token == config.AnonKey {
			role = "anon"
		} else if token == config.ServiceKey {
			role = "service_role"
		}
	}

	// Require at least anon role for WebSocket connections
	if role == "" {
		return c.JSON(401, map[string]any{
			"error":   "Unauthorized",
			"message": "WebSocket connection requires authentication. Pass apikey or token query parameter.",
		})
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Register client
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
	}()

	// Handle messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Process message (simplified)
		// In production, handle subscribe, unsubscribe, broadcast, etc.
		response := map[string]any{
			"type":    "ack",
			"message": string(message),
			"time":    time.Now().Format(time.RFC3339),
		}

		if err := conn.WriteJSON(response); err != nil {
			break
		}
	}

	return nil
}

// Broadcast sends a message to all connected clients.
func (h *RealtimeHandler) Broadcast(message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		_ = client.WriteJSON(message)
	}
}
