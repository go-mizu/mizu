package pipeline

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub manages WebSocket client connections and broadcasts messages to
// clients subscribed to specific job IDs. Hub implements Broadcaster.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	closed  bool
}

// Compile-time check: Hub satisfies Broadcaster.
var _ Broadcaster = (*Hub)(nil)

// NewHub creates a new WebSocket hub ready to accept connections.
func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// HandleWS upgrades the connection to WebSocket and registers a new client.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if strings.Contains(err.Error(), "client is not using the websocket protocol") {
			return
		}
		logErrorf("ws upgrade failed remote=%s ua=%q err=%v", r.RemoteAddr, r.UserAgent(), err)
		return
	}

	c := &client{
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
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	go writePump(c)
	go readPump(c)
}

// Broadcast sends a JSON-encoded message to all clients subscribed to jobID.
// Clients subscribed to "*" receive all broadcasts.
func (h *Hub) Broadcast(jobID string, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		logErrorf("ws broadcast marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if isSubscribed(c, jobID) {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// BroadcastAll sends a JSON-encoded message to ALL connected clients.
func (h *Hub) BroadcastAll(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		logErrorf("ws broadcast-all marshal failed: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}

// Close shuts down the hub, closing all client connections.
func (h *Hub) Close() {
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

func removeClient(h *Hub, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		close(c.send)
		delete(h.clients, c)
	}
}

// client represents a single WebSocket connection with its subscription set.
type client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	mu   sync.RWMutex
	subs map[string]struct{}
}

func isSubscribed(c *client, jobID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.subs["*"]; ok {
		return true
	}
	_, ok := c.subs[jobID]
	return ok
}

func subscribe(c *client, jobIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range jobIDs {
		c.subs[id] = struct{}{}
	}
}

func unsubscribe(c *client, jobIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range jobIDs {
		delete(c.subs, id)
	}
}

func readPump(c *client) {
	defer func() {
		removeClient(c.hub, c)
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
			subscribe(c, msg.JobIDs)
		case "unsubscribe":
			unsubscribe(c, msg.JobIDs)
		}
	}
}

func writePump(c *client) {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
