package async

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// client represents a connected SSE client.
type client struct {
	id     string
	events chan []byte
}

// hub manages SSE connections and message routing.
type hub struct {
	mu      sync.RWMutex
	clients map[string]*client
	bufSize int
}

func newHub(bufSize int) *hub {
	return &hub{
		clients: make(map[string]*client),
		bufSize: bufSize,
	}
}

func (h *hub) register(c *client) {
	h.mu.Lock()
	h.clients[c.id] = c
	h.mu.Unlock()
}

func (h *hub) unregister(id string) {
	h.mu.Lock()
	if c, ok := h.clients[id]; ok {
		close(c.events)
		delete(h.clients, id)
	}
	h.mu.Unlock()
}

func (h *hub) broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		select {
		case c.events <- data:
		default:
			// Buffer full, drop message
		}
	}
}

func (h *hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func generateID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
