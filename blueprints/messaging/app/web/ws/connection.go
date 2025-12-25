package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// Connection represents a WebSocket connection.
type Connection struct {
	hub       *Hub
	conn      *websocket.Conn
	userID    string
	sessionID string
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

// NewConnection creates a new Connection.
func NewConnection(hub *Hub, conn *websocket.Conn, userID, sessionID string) *Connection {
	return &Connection{
		hub:       hub,
		conn:      conn,
		userID:    userID,
		sessionID: sessionID,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
	}
}

// Start starts the connection's read and write pumps.
func (c *Connection) Start() {
	go c.writePump()
	go c.readPump()
}

// Close closes the connection.
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

// SendEvent sends an event to the connection.
func (c *Connection) SendEvent(eventType string, data any) {
	msg, err := MarshalMessage(OpDispatch, eventType, data)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}
	select {
	case c.send <- msg:
	default:
		// Buffer full, skip
	}
}

// SendRaw sends raw bytes to the connection.
func (c *Connection) SendRaw(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

func (c *Connection) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Op {
	case OpHeartbeat:
		ack, _ := MarshalMessage(OpHeartbeatAck, "", nil)
		c.SendRaw(ack)

	case OpTyping:
		// Handle typing indicator
		if payload, ok := msg.Data.(map[string]any); ok {
			if chatID, ok := payload["chat_id"].(string); ok {
				c.hub.BroadcastToChat(chatID, EventTypingStart, map[string]any{
					"user_id": c.userID,
					"chat_id": chatID,
				}, c.userID)
			}
		}
	}
}

// UserID returns the user ID.
func (c *Connection) UserID() string {
	return c.userID
}

// SessionID returns the session ID.
func (c *Connection) SessionID() string {
	return c.sessionID
}
