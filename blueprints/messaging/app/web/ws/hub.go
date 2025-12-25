// Package ws provides WebSocket functionality.
package ws

import (
	"encoding/json"
	"sync"
)

// Event types
const (
	EventReady         = "READY"
	EventMessageCreate = "MESSAGE_CREATE"
	EventMessageUpdate = "MESSAGE_UPDATE"
	EventMessageDelete = "MESSAGE_DELETE"
	EventTypingStart   = "TYPING_START"
	EventTypingStop    = "TYPING_STOP"
	EventPresenceUpdate = "PRESENCE_UPDATE"
	EventChatCreate    = "CHAT_CREATE"
	EventChatUpdate    = "CHAT_UPDATE"
	EventStoryCreate   = "STORY_CREATE"
	EventStoryDelete   = "STORY_DELETE"
	EventCallIncoming  = "CALL_INCOMING"
	EventCallAccepted  = "CALL_ACCEPTED"
	EventCallEnded     = "CALL_ENDED"
)

// Op codes
const (
	OpDispatch     = 0
	OpHeartbeat    = 1
	OpHeartbeatAck = 2
	OpIdentify     = 3
	OpReady        = 4
	OpTyping       = 5
	OpPresence     = 6
	OpAck          = 7
)

// Message represents a WebSocket message.
type Message struct {
	Op   int    `json:"op"`
	Type string `json:"t,omitempty"`
	Data any    `json:"d,omitempty"`
}

// Hub manages WebSocket connections.
type Hub struct {
	connections map[*Connection]bool
	chatSubs    map[string]map[*Connection]bool // chatID -> connections
	userConns   map[string]*Connection          // userID -> connection
	broadcast   chan *broadcastMsg
	register    chan *Connection
	unregister  chan *Connection
	subscribe   chan *subscription
	unsubscribe chan *subscription
	stop        chan struct{}
	mu          sync.RWMutex
}

type broadcastMsg struct {
	chatID  string
	event   string
	data    any
	exclude string // userID to exclude
}

type subscription struct {
	conn   *Connection
	chatID string
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		chatSubs:    make(map[string]map[*Connection]bool),
		userConns:   make(map[string]*Connection),
		broadcast:   make(chan *broadcastMsg, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		subscribe:   make(chan *subscription),
		unsubscribe: make(chan *subscription),
		stop:        make(chan struct{}),
	}
}

// Run starts the hub.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn] = true
			h.userConns[conn.userID] = conn
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				delete(h.userConns, conn.userID)
				// Remove from all chat subscriptions
				for chatID, conns := range h.chatSubs {
					delete(conns, conn)
					if len(conns) == 0 {
						delete(h.chatSubs, chatID)
					}
				}
				conn.Close()
			}
			h.mu.Unlock()

		case sub := <-h.subscribe:
			h.mu.Lock()
			if h.chatSubs[sub.chatID] == nil {
				h.chatSubs[sub.chatID] = make(map[*Connection]bool)
			}
			h.chatSubs[sub.chatID][sub.conn] = true
			h.mu.Unlock()

		case sub := <-h.unsubscribe:
			h.mu.Lock()
			if conns, ok := h.chatSubs[sub.chatID]; ok {
				delete(conns, sub.conn)
				if len(conns) == 0 {
					delete(h.chatSubs, sub.chatID)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if conns, ok := h.chatSubs[msg.chatID]; ok {
				for conn := range conns {
					if msg.exclude != "" && conn.userID == msg.exclude {
						continue
					}
					conn.SendEvent(msg.event, msg.data)
				}
			}
			h.mu.RUnlock()

		case <-h.stop:
			h.mu.Lock()
			for conn := range h.connections {
				conn.Close()
			}
			h.connections = make(map[*Connection]bool)
			h.chatSubs = make(map[string]map[*Connection]bool)
			h.userConns = make(map[string]*Connection)
			h.mu.Unlock()
			return
		}
	}
}

// Stop stops the hub.
func (h *Hub) Stop() {
	close(h.stop)
}

// Register registers a connection.
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister unregisters a connection.
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// SubscribeToChat subscribes a connection to a chat.
func (h *Hub) SubscribeToChat(conn *Connection, chatID string) {
	h.subscribe <- &subscription{conn: conn, chatID: chatID}
}

// UnsubscribeFromChat unsubscribes a connection from a chat.
func (h *Hub) UnsubscribeFromChat(conn *Connection, chatID string) {
	h.unsubscribe <- &subscription{conn: conn, chatID: chatID}
}

// BroadcastToChat broadcasts an event to a chat.
func (h *Hub) BroadcastToChat(chatID, event string, data any, excludeUserID string) {
	h.broadcast <- &broadcastMsg{
		chatID:  chatID,
		event:   event,
		data:    data,
		exclude: excludeUserID,
	}
}

// SendToUser sends an event to a specific user.
func (h *Hub) SendToUser(userID, event string, data any) {
	h.mu.RLock()
	conn, ok := h.userConns[userID]
	h.mu.RUnlock()
	if ok {
		conn.SendEvent(event, data)
	}
}

// GetChatOnlineUsers returns online user IDs for a chat.
func (h *Hub) GetChatOnlineUsers(chatID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userIDs []string
	if conns, ok := h.chatSubs[chatID]; ok {
		for conn := range conns {
			userIDs = append(userIDs, conn.userID)
		}
	}
	return userIDs
}

// IsUserOnline checks if a user is online.
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.userConns[userID]
	return ok
}

// MarshalMessage marshals a message to JSON.
func MarshalMessage(op int, eventType string, data any) ([]byte, error) {
	msg := Message{
		Op:   op,
		Type: eventType,
		Data: data,
	}
	return json.Marshal(msg)
}
