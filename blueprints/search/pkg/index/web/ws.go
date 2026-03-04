package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader allows any origin for development convenience.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ── WSClient ────────────────────────────────────────────────────────────

// WSClient represents a single WebSocket connection with its subscription set.
type WSClient struct {
	hub  *WSHub
	conn *websocket.Conn
	send chan []byte

	mu   sync.RWMutex
	subs map[string]struct{} // job IDs this client is subscribed to
}

// isSubscribed reports whether the client should receive a broadcast for jobID.
func (c *WSClient) isSubscribed(jobID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.subs["*"]; ok {
		return true
	}
	_, ok := c.subs[jobID]
	return ok
}

// subscribe adds the given job IDs to the client's subscription set.
func (c *WSClient) subscribe(jobIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range jobIDs {
		c.subs[id] = struct{}{}
	}
}

// unsubscribe removes the given job IDs from the client's subscription set.
func (c *WSClient) unsubscribe(jobIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range jobIDs {
		delete(c.subs, id)
	}
}

// readPump reads messages from the WebSocket and processes subscribe/unsubscribe commands.
// It runs in its own goroutine per client.
func (c *WSClient) readPump() {
	defer func() {
		c.hub.removeClient(c)
		c.conn.Close()
	}()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type   string   `json:"type"`
			JobIDs []string `json:"job_ids"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "subscribe":
			c.subscribe(msg.JobIDs)
		case "unsubscribe":
			c.unsubscribe(msg.JobIDs)
		}
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
// It runs in its own goroutine per client.
func (c *WSClient) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// ── WSHub ───────────────────────────────────────────────────────────────

// WSHub manages WebSocket client connections and broadcasts messages to
// clients subscribed to specific job IDs.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*WSClient]struct{}
	closed  bool
}

// NewWSHub creates a new WebSocket hub ready to accept connections.
func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*WSClient]struct{}),
	}
}

// HandleWS is an http.HandlerFunc that upgrades the connection to WebSocket
// and registers a new client with the hub.
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Non-websocket probes (e.g. curl/health checks) are expected.
		if strings.Contains(err.Error(), "client is not using the websocket protocol") {
			return
		}
		logErrorf("ws upgrade failed remote=%s ua=%q err=%v", r.RemoteAddr, r.UserAgent(), err)
		return
	}

	client := &WSClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 64),
		subs: make(map[string]struct{}),
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		conn.Close()
		return
	}
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go client.writePump()
	go client.readPump()
}

// Broadcast sends a JSON-encoded message to all clients subscribed to jobID.
// Clients subscribed to "*" receive all broadcasts.
func (h *WSHub) Broadcast(jobID string, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		logErrorf("ws broadcast marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.isSubscribed(jobID) {
			select {
			case c.send <- data:
			default:
				// Drop message if the client's send buffer is full.
			}
		}
	}
}

// Close shuts down the hub, closing all client connections.
func (h *WSHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for c := range h.clients {
		close(c.send)
		delete(h.clients, c)
	}
}

// removeClient unregisters a client from the hub and closes its send channel.
func (h *WSHub) removeClient(c *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		close(c.send)
		delete(h.clients, c)
	}
}
