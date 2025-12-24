package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB

	// Heartbeat interval to send to client.
	heartbeatInterval = 45000 // 45 seconds
)

// Connection represents a WebSocket connection.
type Connection struct {
	ID        string
	UserID    string
	SessionID string
	Conn      *websocket.Conn
	Hub       *Hub
	sendCh    chan []byte
	Servers   []string // Subscribed server IDs
	Channels  []string // Subscribed channel IDs (for DMs)
	Sequence  int64

	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

// NewConnection creates a new connection.
func NewConnection(hub *Hub, conn *websocket.Conn, userID, sessionID string) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		ID:        generateID(),
		UserID:    userID,
		SessionID: sessionID,
		Conn:      conn,
		Hub:       hub,
		sendCh:    make(chan []byte, 256),
		Servers:   make([]string, 0),
		Channels:  make([]string, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the connection's read and write pumps.
func (c *Connection) Start() {
	go c.writePump()
	go c.readPump()

	// Send HELLO
	c.sendHello()
}

// Send sends a message to the connection.
func (c *Connection) Send(msg []byte) {
	select {
	case c.sendCh <- msg:
	default:
		// Channel full, connection is slow
		log.Printf("WebSocket: Send channel full for user %s", c.UserID)
	}
}

// SendMessage sends a typed message.
func (c *Connection) SendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.Send(data)
	return nil
}

// SendEvent sends an event to the connection.
func (c *Connection) SendEvent(event string, data any) error {
	d, _ := json.Marshal(data)
	return c.SendMessage(&Message{
		Op: OpDispatch,
		T:  event,
		D:  d,
		S:  c.nextSequence(),
	})
}

// Close closes the connection.
func (c *Connection) Close() {
	c.once.Do(func() {
		c.cancel()
		close(c.sendCh)
		c.Conn.Close()
	})
}

func (c *Connection) sendHello() {
	hello := map[string]any{
		"heartbeat_interval": heartbeatInterval,
	}
	data, _ := json.Marshal(hello)
	c.SendMessage(&Message{
		Op: OpHello,
		D:  data,
	})
}

func (c *Connection) readPump() {
	defer func() {
		c.Hub.Unregister(c)
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		c.handleMessage(message)
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return

		case message, ok := <-c.sendCh:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.sendCh)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.sendCh)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Connection) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("WebSocket: Invalid message from user %s: %v", c.UserID, err)
		return
	}

	switch msg.Op {
	case OpHeartbeat:
		c.handleHeartbeat()

	case OpIdentify:
		c.handleIdentify(msg.D)

	case OpPresenceUpdate:
		c.handlePresenceUpdate(msg.D)

	case OpResume:
		c.handleResume(msg.D)

	default:
		log.Printf("WebSocket: Unknown op %d from user %s", msg.Op, c.UserID)
	}
}

func (c *Connection) handleHeartbeat() {
	c.SendMessage(&Message{Op: OpHeartbeatAck})
}

func (c *Connection) handleIdentify(data json.RawMessage) {
	// Already authenticated via token in URL, just acknowledge
	// The ready event is sent after subscriptions are set up
}

func (c *Connection) handlePresenceUpdate(data json.RawMessage) {
	// Handle presence update
	var update struct {
		Status       string `json:"status"`
		CustomStatus string `json:"custom_status"`
	}
	if err := json.Unmarshal(data, &update); err != nil {
		return
	}

	// Broadcast presence update to subscribed servers
	for _, serverID := range c.Servers {
		c.Hub.BroadcastToServer(serverID, EventPresenceUpdate, map[string]any{
			"user_id": c.UserID,
			"status":  update.Status,
		}, "")
	}
}

func (c *Connection) handleResume(data json.RawMessage) {
	// Handle session resume
	var resume struct {
		SessionID string `json:"session_id"`
		Sequence  int64  `json:"seq"`
	}
	if err := json.Unmarshal(data, &resume); err != nil {
		c.SendMessage(&Message{Op: OpInvalidSession})
		return
	}

	// For now, just send reconnect to force fresh connection
	c.SendMessage(&Message{Op: OpReconnect})
}

func (c *Connection) nextSequence() int64 {
	c.Sequence++
	return c.Sequence
}

// Simple ID generation
var idCounter int64
var idLock sync.Mutex

func generateID() string {
	idLock.Lock()
	defer idLock.Unlock()
	idCounter++
	return string(rune(idCounter))
}
