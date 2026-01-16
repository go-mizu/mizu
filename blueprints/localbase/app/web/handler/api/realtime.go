package api

import (
	"sync"
	"time"

	"github.com/go-mizu/mizu"
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
func (h *RealtimeHandler) WebSocket(c *mizu.Ctx) error {
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
