// Package ws provides WebSocket functionality for realtime chat.
package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// OpCode represents WebSocket operation codes.
type OpCode int

const (
	OpDispatch        OpCode = 0  // Server -> Client: Event dispatch
	OpHeartbeat       OpCode = 1  // Client -> Server: Heartbeat
	OpIdentify        OpCode = 2  // Client -> Server: Auth
	OpPresenceUpdate  OpCode = 3  // Client -> Server: Update presence
	OpResume          OpCode = 6  // Client -> Server: Resume connection
	OpReconnect       OpCode = 7  // Server -> Client: Reconnect request
	OpInvalidSession  OpCode = 9  // Server -> Client: Invalid session
	OpHello           OpCode = 10 // Server -> Client: Hello
	OpHeartbeatAck    OpCode = 11 // Server -> Client: Heartbeat acknowledged
)

// Message represents a WebSocket message.
type Message struct {
	Op   OpCode          `json:"op"`
	T    string          `json:"t,omitempty"`
	D    json.RawMessage `json:"d,omitempty"`
	S    int64           `json:"s,omitempty"`
}

// Event names
const (
	EventReady              = "READY"
	EventMessageCreate      = "MESSAGE_CREATE"
	EventMessageUpdate      = "MESSAGE_UPDATE"
	EventMessageDelete      = "MESSAGE_DELETE"
	EventMessageReactionAdd = "MESSAGE_REACTION_ADD"
	EventMessageReactionRemove = "MESSAGE_REACTION_REMOVE"
	EventChannelCreate      = "CHANNEL_CREATE"
	EventChannelUpdate      = "CHANNEL_UPDATE"
	EventChannelDelete      = "CHANNEL_DELETE"
	EventMemberAdd          = "MEMBER_ADD"
	EventMemberRemove       = "MEMBER_REMOVE"
	EventMemberUpdate       = "MEMBER_UPDATE"
	EventPresenceUpdate     = "PRESENCE_UPDATE"
	EventTypingStart        = "TYPING_START"
	EventServerUpdate       = "SERVER_UPDATE"
	EventRoleCreate         = "ROLE_CREATE"
	EventRoleUpdate         = "ROLE_UPDATE"
	EventRoleDelete         = "ROLE_DELETE"
)

// Hub manages all WebSocket connections.
type Hub struct {
	// Connections by user ID
	connections map[string]map[*Connection]bool
	connLock    sync.RWMutex

	// Server subscriptions: serverID -> connections
	servers     map[string]map[*Connection]bool
	serverLock  sync.RWMutex

	// Channel subscriptions: channelID -> connections (for DMs)
	channels    map[string]map[*Connection]bool
	channelLock sync.RWMutex

	// Message sequence counter
	sequence     int64
	sequenceLock sync.Mutex

	// Channels for operations
	register   chan *Connection
	unregister chan *Connection
	broadcast  chan *Broadcast

	// Shutdown
	done chan struct{}
}

// Broadcast represents a message to broadcast.
type Broadcast struct {
	ServerID  string // Server to broadcast to
	ChannelID string // Channel to broadcast to (for DMs)
	UserID    string // Single user to send to
	Event     string
	Data      any
	ExcludeID string // User ID to exclude
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]map[*Connection]bool),
		servers:     make(map[string]map[*Connection]bool),
		channels:    make(map[string]map[*Connection]bool),
		register:    make(chan *Connection, 256),
		unregister:  make(chan *Connection, 256),
		broadcast:   make(chan *Broadcast, 256),
		done:        make(chan struct{}),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case b := <-h.broadcast:
			h.handleBroadcast(b)

		case <-ticker.C:
			h.cleanup()

		case <-h.done:
			return
		}
	}
}

// Stop stops the hub.
func (h *Hub) Stop() {
	close(h.done)
}

// Register registers a connection.
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister unregisters a connection.
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// Broadcast sends a broadcast message.
func (h *Hub) Broadcast(b *Broadcast) {
	h.broadcast <- b
}

// BroadcastToServer broadcasts to all connections in a server.
func (h *Hub) BroadcastToServer(serverID, event string, data any, excludeUserID string) {
	h.broadcast <- &Broadcast{
		ServerID:  serverID,
		Event:     event,
		Data:      data,
		ExcludeID: excludeUserID,
	}
}

// BroadcastToChannel broadcasts to all connections subscribed to a channel.
func (h *Hub) BroadcastToChannel(channelID, event string, data any, excludeUserID string) {
	h.broadcast <- &Broadcast{
		ChannelID: channelID,
		Event:     event,
		Data:      data,
		ExcludeID: excludeUserID,
	}
}

// SendToUser sends a message to a specific user.
func (h *Hub) SendToUser(userID, event string, data any) {
	h.broadcast <- &Broadcast{
		UserID: userID,
		Event:  event,
		Data:   data,
	}
}

func (h *Hub) registerConnection(conn *Connection) {
	h.connLock.Lock()
	wasOffline := len(h.connections[conn.UserID]) == 0
	if h.connections[conn.UserID] == nil {
		h.connections[conn.UserID] = make(map[*Connection]bool)
	}
	h.connections[conn.UserID][conn] = true
	h.connLock.Unlock()

	log.Printf("WebSocket: User %s connected (total: %d)", conn.UserID, len(h.connections))

	// Broadcast presence update if user was offline
	if wasOffline {
		go func() {
			// Wait a bit for server subscriptions to be set up
			<-time.After(100 * time.Millisecond)
			h.broadcastPresenceUpdate(conn.UserID, "online")
		}()
	}
}

func (h *Hub) unregisterConnection(conn *Connection) {
	// Get server subscriptions before unlocking
	serverIDs := make([]string, len(conn.Servers))
	copy(serverIDs, conn.Servers)
	userID := conn.UserID

	h.connLock.Lock()
	if conns, ok := h.connections[conn.UserID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, conn.UserID)
		}
	}
	isNowOffline := len(h.connections[conn.UserID]) == 0
	h.connLock.Unlock()

	// Remove from server subscriptions
	h.serverLock.Lock()
	for _, serverID := range conn.Servers {
		if conns, ok := h.servers[serverID]; ok {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(h.servers, serverID)
			}
		}
	}
	h.serverLock.Unlock()

	// Remove from channel subscriptions
	h.channelLock.Lock()
	for _, channelID := range conn.Channels {
		if conns, ok := h.channels[channelID]; ok {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(h.channels, channelID)
			}
		}
	}
	h.channelLock.Unlock()

	conn.Close()

	log.Printf("WebSocket: User %s disconnected", userID)

	// Broadcast presence update if user is now offline
	if isNowOffline && len(serverIDs) > 0 {
		h.broadcastPresenceUpdateToServers(userID, "offline", serverIDs)
	}
}

func (h *Hub) handleBroadcast(b *Broadcast) {
	// Prepare message
	data, _ := json.Marshal(b.Data)
	msg := &Message{
		Op: OpDispatch,
		T:  b.Event,
		D:  data,
		S:  h.nextSequence(),
	}
	msgBytes, _ := json.Marshal(msg)

	// Broadcast to server
	if b.ServerID != "" {
		h.serverLock.RLock()
		conns := h.servers[b.ServerID]
		h.serverLock.RUnlock()

		for conn := range conns {
			if b.ExcludeID != "" && conn.UserID == b.ExcludeID {
				continue
			}
			conn.Send(msgBytes)
		}
		return
	}

	// Broadcast to channel
	if b.ChannelID != "" {
		h.channelLock.RLock()
		conns := h.channels[b.ChannelID]
		h.channelLock.RUnlock()

		for conn := range conns {
			if b.ExcludeID != "" && conn.UserID == b.ExcludeID {
				continue
			}
			conn.Send(msgBytes)
		}
		return
	}

	// Send to specific user
	if b.UserID != "" {
		h.connLock.RLock()
		conns := h.connections[b.UserID]
		h.connLock.RUnlock()

		for conn := range conns {
			conn.Send(msgBytes)
		}
		return
	}
}

// SubscribeToServer subscribes a connection to a server.
func (h *Hub) SubscribeToServer(conn *Connection, serverID string) {
	h.serverLock.Lock()
	defer h.serverLock.Unlock()

	if h.servers[serverID] == nil {
		h.servers[serverID] = make(map[*Connection]bool)
	}
	h.servers[serverID][conn] = true
	conn.Servers = append(conn.Servers, serverID)
}

// UnsubscribeFromServer unsubscribes a connection from a server.
func (h *Hub) UnsubscribeFromServer(conn *Connection, serverID string) {
	h.serverLock.Lock()
	defer h.serverLock.Unlock()

	if conns, ok := h.servers[serverID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.servers, serverID)
		}
	}

	// Remove from connection's server list
	for i, id := range conn.Servers {
		if id == serverID {
			conn.Servers = append(conn.Servers[:i], conn.Servers[i+1:]...)
			break
		}
	}
}

// SubscribeToChannel subscribes a connection to a channel (for DMs).
func (h *Hub) SubscribeToChannel(conn *Connection, channelID string) {
	h.channelLock.Lock()
	defer h.channelLock.Unlock()

	if h.channels[channelID] == nil {
		h.channels[channelID] = make(map[*Connection]bool)
	}
	h.channels[channelID][conn] = true
	conn.Channels = append(conn.Channels, channelID)
}

// GetOnlineUserIDs returns IDs of users currently connected.
func (h *Hub) GetOnlineUserIDs() []string {
	h.connLock.RLock()
	defer h.connLock.RUnlock()

	ids := make([]string, 0, len(h.connections))
	for id := range h.connections {
		ids = append(ids, id)
	}
	return ids
}

// IsUserOnline checks if a user has any active connections.
func (h *Hub) IsUserOnline(userID string) bool {
	h.connLock.RLock()
	defer h.connLock.RUnlock()
	return len(h.connections[userID]) > 0
}

// GetServerOnlineUsers returns online users in a server.
func (h *Hub) GetServerOnlineUsers(serverID string) []string {
	h.serverLock.RLock()
	defer h.serverLock.RUnlock()

	seen := make(map[string]bool)
	var ids []string

	if conns, ok := h.servers[serverID]; ok {
		for conn := range conns {
			if !seen[conn.UserID] {
				seen[conn.UserID] = true
				ids = append(ids, conn.UserID)
			}
		}
	}

	return ids
}

func (h *Hub) nextSequence() int64 {
	h.sequenceLock.Lock()
	defer h.sequenceLock.Unlock()
	h.sequence++
	return h.sequence
}

func (h *Hub) cleanup() {
	// Cleanup is handled by connection timeout
}

// broadcastPresenceUpdate broadcasts a presence update to all servers the user is subscribed to.
func (h *Hub) broadcastPresenceUpdate(userID, status string) {
	h.connLock.RLock()
	conns := h.connections[userID]
	h.connLock.RUnlock()

	// Collect all server IDs the user is subscribed to
	serverIDSet := make(map[string]bool)
	for conn := range conns {
		for _, serverID := range conn.Servers {
			serverIDSet[serverID] = true
		}
	}

	// Broadcast to each server
	for serverID := range serverIDSet {
		h.BroadcastToServer(serverID, EventPresenceUpdate, map[string]any{
			"user_id": userID,
			"status":  status,
		}, "")
	}
}

// broadcastPresenceUpdateToServers broadcasts a presence update to specific servers.
func (h *Hub) broadcastPresenceUpdateToServers(userID, status string, serverIDs []string) {
	for _, serverID := range serverIDs {
		h.BroadcastToServer(serverID, EventPresenceUpdate, map[string]any{
			"user_id": userID,
			"status":  status,
		}, "")
	}
}
