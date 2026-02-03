// Package dashboard provides the Control Dashboard WebSocket hub and RPC dispatch.
package dashboard

import (
	"crypto/sha1" //nolint:gosec // Required by WebSocket protocol RFC 6455
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// RPCRequest is a JSON-RPC request from a dashboard client.
type RPCRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	Type   string          `json:"type,omitempty"` // "hello" for handshake
	Token  string          `json:"token,omitempty"`
}

// RPCResponse is a JSON-RPC response sent to a dashboard client.
type RPCResponse struct {
	ID     string `json:"id,omitempty"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// EventFrame is a broadcast event sent to all connected clients.
type EventFrame struct {
	Type    string `json:"type"`
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
}

// HelloOK is the handshake response.
type HelloOK struct {
	Type     string      `json:"type"`
	Protocol int         `json:"protocol"`
	Features RPCFeatures `json:"features"`
}

// RPCFeatures lists available RPC methods and event types.
type RPCFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

// MethodHandler is a function that handles an RPC method call.
type MethodHandler func(params json.RawMessage) (any, error)

// flusher is an interface for buffered writers that support Flush.
type flusher interface {
	Flush() error
}

// wsConn is a minimal WebSocket connection using raw frames.
type wsConn struct {
	rw io.ReadWriter
	mu sync.Mutex
}

func (c *wsConn) readFrame() (int, []byte, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(c.rw, buf); err != nil {
		return 0, nil, err
	}
	opcode := int(buf[0] & 0x0F)
	masked := buf[1]&0x80 != 0
	length := int(buf[1] & 0x7F)

	switch length {
	case 126:
		lb := make([]byte, 2)
		if _, err := io.ReadFull(c.rw, lb); err != nil {
			return 0, nil, err
		}
		length = int(lb[0])<<8 | int(lb[1])
	case 127:
		lb := make([]byte, 8)
		if _, err := io.ReadFull(c.rw, lb); err != nil {
			return 0, nil, err
		}
		length = int(lb[4])<<24 | int(lb[5])<<16 | int(lb[6])<<8 | int(lb[7])
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.rw, mask); err != nil {
			return 0, nil, err
		}
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(c.rw, data); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range data {
			data[i] ^= mask[i%4]
		}
	}

	return opcode, data, nil
}

func (c *wsConn) writeFrame(opcode int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	frame := []byte{0x80 | byte(opcode)}
	length := len(data)
	if length <= 125 {
		frame = append(frame, byte(length))
	} else if length <= 65535 {
		frame = append(frame, 126, byte(length>>8), byte(length))
	} else {
		frame = append(frame, 127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(length>>(8*i)))
		}
	}
	frame = append(frame, data...)
	if _, err := c.rw.Write(frame); err != nil {
		return err
	}
	// Flush buffered writer if available
	if f, ok := c.rw.(flusher); ok {
		return f.Flush()
	}
	return nil
}

func (c *wsConn) writeText(data []byte) error {
	return c.writeFrame(1, data) // text opcode
}

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Client represents a connected dashboard WebSocket client.
type Client struct {
	id          string
	ws          *wsConn
	authed      bool
	connectedAt time.Time
	remoteAddr  string
	userAgent   string
}

func (c *Client) send(data []byte) error {
	return c.ws.writeText(data)
}

// Hub manages WebSocket connections and dispatches RPC calls.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client
	methods  map[string]MethodHandler
	token    string // gateway auth token (empty = no auth)
	seqID    int
}

// NewHub creates a WebSocket hub.
func NewHub(token string) *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		methods: make(map[string]MethodHandler),
		token:   token,
	}
}

// Register adds an RPC method handler.
func (h *Hub) Register(method string, handler MethodHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.methods[method] = handler
}

// Broadcast sends an event to all authenticated clients.
func (h *Hub) Broadcast(event string, payload any) {
	frame := EventFrame{Type: "event", Event: event, Payload: payload}
	data, err := json.Marshal(frame)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		if c.authed {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.send(data)
	}
}

// Instances returns the list of connected dashboard clients.
func (h *Hub) Instances() []types.Instance {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var instances []types.Instance
	for _, c := range h.clients {
		if !c.authed {
			continue
		}
		instances = append(instances, types.Instance{
			ID:          c.id,
			RemoteAddr:  c.remoteAddr,
			ConnectedAt: c.connectedAt.UnixMilli(),
			UserAgent:   c.userAgent,
			Role:        "operator",
		})
	}
	return instances
}

// ClientCount returns the number of authenticated clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, c := range h.clients {
		if c.authed {
			count++
		}
	}
	return count
}

// WSHandler returns a Mizu handler that upgrades to WebSocket using Ctx.Hijack()
// and runs the JSON-RPC dispatch loop.
func (h *Hub) WSHandler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		r := c.Request()

		// Check WebSocket upgrade headers
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
			!strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
			return c.Text(400, "not a websocket request")
		}

		key := r.Header.Get("Sec-WebSocket-Key")
		if key == "" {
			return c.Text(400, "missing Sec-WebSocket-Key")
		}

		// Compute accept key
		hash := sha1.New() //nolint:gosec
		hash.Write([]byte(key + websocketGUID))
		acceptKey := base64.StdEncoding.EncodeToString(hash.Sum(nil))

		// Hijack the connection using Mizu's Ctx.Hijack which properly unwraps
		conn, bufrw, err := c.Hijack()
		if err != nil {
			return fmt.Errorf("websocket hijack: %w", err)
		}

		// Send upgrade response
		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
		_, _ = bufrw.WriteString(response)
		_ = bufrw.Flush()

		ws := &wsConn{rw: bufrw}

		// Register client
		h.mu.Lock()
		h.seqID++
		clientID := fmt.Sprintf("client-%d", h.seqID)
		client := &Client{
			id:          clientID,
			ws:          ws,
			connectedAt: time.Now(),
			remoteAddr:  r.RemoteAddr,
			userAgent:   r.UserAgent(),
		}
		h.clients[clientID] = client
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.clients, clientID)
			h.mu.Unlock()
			_ = conn.Close()
		}()

		// Read loop
		for {
			opcode, data, err := ws.readFrame()
			if err != nil {
				return nil // connection closed
			}
			if opcode == 8 { // close frame — echo it back per RFC 6455
				_ = ws.writeFrame(8, data)
				return nil
			}
			if opcode == 9 { // ping
				_ = ws.writeFrame(10, data) // pong
				continue
			}
			if opcode != 1 { // not text
				continue
			}

			var req RPCRequest
			if err := json.Unmarshal(data, &req); err != nil {
				h.sendError(client, "", "invalid JSON")
				continue
			}

			// Handle handshake
			if req.Type == "hello" {
				h.handleHello(client, &req)
				continue
			}

			// Require auth for all other requests
			if !client.authed {
				h.sendError(client, req.ID, "not authenticated")
				continue
			}

			// Dispatch RPC method
			h.handleRPC(client, &req)
		}
	}
}

func (h *Hub) handleHello(client *Client, req *RPCRequest) {
	// If no token configured, auto-authenticate
	if h.token == "" || req.Token == h.token {
		client.authed = true
	} else {
		data, _ := json.Marshal(map[string]string{
			"type":  "hello-error",
			"error": "invalid token",
		})
		_ = client.send(data)
		return
	}

	h.mu.RLock()
	methods := make([]string, 0, len(h.methods))
	for m := range h.methods {
		methods = append(methods, m)
	}
	h.mu.RUnlock()

	// Build snapshot for hello-ok
	snapshot := map[string]any{
		"presence": h.Instances(),
	}

	resp := map[string]any{
		"type":     "hello-ok",
		"protocol": 1,
		"features": map[string]any{
			"methods": methods,
			"events":  []string{"session.updated", "cron.updated", "channel.updated", "log.entry", "chat", "chat.message", "chat.typing", "chat.done", "presence", "health"},
		},
		"snapshot": snapshot,
	}

	data, _ := json.Marshal(resp)
	_ = client.send(data)
}

func (h *Hub) handleRPC(client *Client, req *RPCRequest) {
	h.mu.RLock()
	handler, ok := h.methods[req.Method]
	h.mu.RUnlock()

	if !ok {
		h.sendError(client, req.ID, fmt.Sprintf("unknown method: %s", req.Method))
		return
	}

	result, err := handler(req.Params)
	if err != nil {
		h.sendError(client, req.ID, err.Error())
		return
	}

	resp := RPCResponse{ID: req.ID, Result: result}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("dashboard: marshal response error: %v", err)
		return
	}
	_ = client.send(data)
}

func (h *Hub) sendError(client *Client, id, errMsg string) {
	resp := RPCResponse{ID: id, Error: errMsg}
	data, _ := json.Marshal(resp)
	_ = client.send(data)
}
