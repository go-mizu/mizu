package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/middleware"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/gorilla/websocket"
)

// Phoenix protocol message events
const (
	PhxJoin      = "phx_join"
	PhxLeave     = "phx_leave"
	PhxReply     = "phx_reply"
	PhxError     = "phx_error"
	PhxClose     = "phx_close"
	Heartbeat    = "heartbeat"
	Broadcast    = "broadcast"
	Presence     = "presence"
	PresenceState = "presence_state"
	PresenceDiff  = "presence_diff"
	PostgresChanges = "postgres_changes"
	AccessToken  = "access_token"
)

// PhoenixMessage represents a Phoenix protocol message
type PhoenixMessage struct {
	Topic   string      `json:"topic"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
	Ref     string      `json:"ref,omitempty"`
	JoinRef string      `json:"join_ref,omitempty"`
}

// JoinPayload represents the payload for a phx_join event
type JoinPayload struct {
	Config      *JoinConfig `json:"config,omitempty"`
	AccessToken string      `json:"access_token,omitempty"`
}

// JoinConfig represents channel join configuration
type JoinConfig struct {
	Broadcast       *BroadcastConfig       `json:"broadcast,omitempty"`
	Presence        *PresenceConfig        `json:"presence,omitempty"`
	PostgresChanges []PostgresChangeConfig `json:"postgres_changes,omitempty"`
	Private         bool                   `json:"private,omitempty"`
}

// BroadcastConfig represents broadcast configuration
type BroadcastConfig struct {
	Self bool `json:"self"`
	Ack  bool `json:"ack,omitempty"`
}

// PresenceConfig represents presence configuration
type PresenceConfig struct {
	Key string `json:"key"`
}

// PostgresChangeConfig represents postgres_changes configuration
type PostgresChangeConfig struct {
	ID     int64  `json:"id,omitempty"`
	Event  string `json:"event"`
	Schema string `json:"schema"`
	Table  string `json:"table,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// BroadcastPayload represents a broadcast message payload
type BroadcastPayload struct {
	Type    string      `json:"type"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// PresencePayload represents a presence message payload
type PresencePayload struct {
	Type    string      `json:"type"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// PresenceMeta represents presence metadata
type PresenceMeta struct {
	PhxRef string `json:"phx_ref"`
	// Additional custom fields stored as map
}

// ClientState holds per-connection state
type ClientState struct {
	conn           *websocket.Conn
	role           string
	userID         string
	subscribedTopics map[string]*TopicSubscription
	presenceKey    string
	mu             sync.RWMutex
}

// TopicSubscription holds subscription state for a topic
type TopicSubscription struct {
	JoinRef         string
	BroadcastConfig *BroadcastConfig
	PresenceConfig  *PresenceConfig
	PostgresChanges []PostgresChangeConfig
	PresenceState   map[string]interface{}
}

// RealtimeHandler handles realtime WebSocket endpoints.
type RealtimeHandler struct {
	store          *postgres.Store
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]*ClientState
	topicClients   map[string]map[*websocket.Conn]bool // topic -> clients subscribed
	presenceState  map[string]map[string]interface{}   // topic -> key -> presence data
	mu             sync.RWMutex
	pgChangeIDCounter int64
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
		clients:       make(map[*websocket.Conn]*ClientState),
		topicClients:  make(map[string]map[*websocket.Conn]bool),
		presenceState: make(map[string]map[string]interface{}),
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
	topicCount := len(h.topicClients)
	h.mu.RUnlock()

	channels, _ := h.store.Realtime().ListChannels(c.Context())

	return c.JSON(200, map[string]any{
		"connections": connCount,
		"channels":    len(channels),
		"topics":      topicCount,
		"server_time": time.Now().Format(time.RFC3339),
	})
}

// WebSocket handles WebSocket connections.
// SEC-017: WebSocket authentication via token query parameter or API key
func (h *RealtimeHandler) WebSocket(c *mizu.Ctx) error {
	// Check authentication before upgrading
	token := c.Query("apikey")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		token = c.Request().Header.Get("apikey")
	}

	// Validate token
	role := middleware.GetRole(c)
	if role == "" && token != "" {
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

	// Create client state
	clientState := &ClientState{
		conn:             conn,
		role:             role,
		subscribedTopics: make(map[string]*TopicSubscription),
	}

	// Register client
	h.mu.Lock()
	h.clients[conn] = clientState
	h.mu.Unlock()

	defer h.cleanupClient(conn, clientState)

	// Handle messages with Phoenix protocol
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg PhoenixMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			// Invalid JSON - send error response
			h.sendError(conn, "", "", "invalid_message", "Invalid JSON message format")
			continue
		}

		h.handleMessage(conn, clientState, &msg)
	}

	return nil
}

// handleMessage processes incoming Phoenix messages
func (h *RealtimeHandler) handleMessage(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	switch msg.Event {
	case Heartbeat:
		h.handleHeartbeat(conn, msg)
	case PhxJoin:
		h.handleJoin(conn, state, msg)
	case PhxLeave:
		h.handleLeave(conn, state, msg)
	case Broadcast:
		h.handleBroadcast(conn, state, msg)
	case Presence:
		h.handlePresence(conn, state, msg)
	case AccessToken:
		h.handleAccessToken(conn, state, msg)
	default:
		// Echo back for unknown events (compatibility)
		h.sendReply(conn, msg.Topic, msg.Ref, "ok", map[string]interface{}{
			"message": "received",
		})
	}
}

// handleHeartbeat processes heartbeat messages
func (h *RealtimeHandler) handleHeartbeat(conn *websocket.Conn, msg *PhoenixMessage) {
	h.sendReply(conn, "phoenix", msg.Ref, "ok", map[string]interface{}{})
}

// handleJoin processes channel join requests
func (h *RealtimeHandler) handleJoin(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	// Parse join payload
	var joinPayload JoinPayload
	if payloadBytes, err := json.Marshal(msg.Payload); err == nil {
		json.Unmarshal(payloadBytes, &joinPayload)
	}

	// Create subscription
	subscription := &TopicSubscription{
		JoinRef: msg.JoinRef,
	}

	if joinPayload.Config != nil {
		subscription.BroadcastConfig = joinPayload.Config.Broadcast
		subscription.PresenceConfig = joinPayload.Config.Presence
		subscription.PostgresChanges = joinPayload.Config.PostgresChanges
	}

	// Assign IDs to postgres_changes subscriptions
	response := map[string]interface{}{}
	if len(subscription.PostgresChanges) > 0 {
		pgChanges := make([]map[string]interface{}, len(subscription.PostgresChanges))
		for i, pc := range subscription.PostgresChanges {
			id := atomic.AddInt64(&h.pgChangeIDCounter, 1)
			subscription.PostgresChanges[i].ID = id
			pgChanges[i] = map[string]interface{}{
				"id":     id,
				"event":  pc.Event,
				"schema": pc.Schema,
				"table":  pc.Table,
				"filter": pc.Filter,
			}
		}
		response["postgres_changes"] = pgChanges
	}

	// Store subscription
	state.mu.Lock()
	state.subscribedTopics[msg.Topic] = subscription
	if subscription.PresenceConfig != nil && subscription.PresenceConfig.Key != "" {
		state.presenceKey = subscription.PresenceConfig.Key
	}
	state.mu.Unlock()

	// Add to topic clients
	h.mu.Lock()
	if h.topicClients[msg.Topic] == nil {
		h.topicClients[msg.Topic] = make(map[*websocket.Conn]bool)
	}
	h.topicClients[msg.Topic][conn] = true
	h.mu.Unlock()

	// Send join reply
	h.sendReply(conn, msg.Topic, msg.Ref, "ok", response)

	// Send presence_state if presence is configured
	if subscription.PresenceConfig != nil {
		h.sendPresenceState(conn, msg.Topic)
		h.broadcastPresenceDiff(msg.Topic, conn, state, "join")
	}
}

// handleLeave processes channel leave requests
func (h *RealtimeHandler) handleLeave(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	state.mu.Lock()
	sub := state.subscribedTopics[msg.Topic]
	delete(state.subscribedTopics, msg.Topic)
	state.mu.Unlock()

	// Remove from topic clients
	h.mu.Lock()
	if h.topicClients[msg.Topic] != nil {
		delete(h.topicClients[msg.Topic], conn)
		if len(h.topicClients[msg.Topic]) == 0 {
			delete(h.topicClients, msg.Topic)
		}
	}
	h.mu.Unlock()

	// Broadcast presence diff for leave
	if sub != nil && sub.PresenceConfig != nil {
		h.broadcastPresenceDiff(msg.Topic, conn, state, "leave")
	}

	h.sendReply(conn, msg.Topic, msg.Ref, "ok", map[string]interface{}{})
}

// handleBroadcast processes broadcast messages
func (h *RealtimeHandler) handleBroadcast(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	state.mu.RLock()
	sub := state.subscribedTopics[msg.Topic]
	state.mu.RUnlock()

	if sub == nil {
		h.sendReply(conn, msg.Topic, msg.Ref, "error", map[string]interface{}{
			"reason": "not subscribed to channel",
		})
		return
	}

	// Broadcast to all clients on the topic
	h.mu.RLock()
	clients := h.topicClients[msg.Topic]
	h.mu.RUnlock()

	broadcastMsg := PhoenixMessage{
		Topic:   msg.Topic,
		Event:   Broadcast,
		Payload: msg.Payload,
		// Server-sent broadcasts have no ref (empty string)
	}

	for client := range clients {
		// Check if this is the sender and self=false
		if client == conn {
			state.mu.RLock()
			senderSub := state.subscribedTopics[msg.Topic]
			state.mu.RUnlock()
			if senderSub != nil && senderSub.BroadcastConfig != nil && !senderSub.BroadcastConfig.Self {
				continue // Skip sender if self=false
			}
		}
		client.WriteJSON(broadcastMsg)
	}

	// Send acknowledgment if ack=true
	if sub.BroadcastConfig != nil && sub.BroadcastConfig.Ack {
		h.sendReply(conn, msg.Topic, msg.Ref, "ok", map[string]interface{}{})
	}
}

// handlePresence processes presence messages
func (h *RealtimeHandler) handlePresence(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	var presencePayload PresencePayload
	if payloadBytes, err := json.Marshal(msg.Payload); err == nil {
		json.Unmarshal(payloadBytes, &presencePayload)
	}

	state.mu.RLock()
	sub := state.subscribedTopics[msg.Topic]
	presenceKey := state.presenceKey
	state.mu.RUnlock()

	if sub == nil {
		h.sendReply(conn, msg.Topic, msg.Ref, "error", map[string]interface{}{
			"reason": "not subscribed to channel",
		})
		return
	}

	// Use presence key from subscription or generate one
	if presenceKey == "" {
		presenceKey = generateUUID()
		state.mu.Lock()
		state.presenceKey = presenceKey
		state.mu.Unlock()
	}

	switch presencePayload.Event {
	case "track":
		// Update presence state
		h.mu.Lock()
		if h.presenceState[msg.Topic] == nil {
			h.presenceState[msg.Topic] = make(map[string]interface{})
		}
		h.presenceState[msg.Topic][presenceKey] = map[string]interface{}{
			"metas": []map[string]interface{}{
				{
					"phx_ref": generateRef(),
					"payload": presencePayload.Payload,
				},
			},
		}
		h.mu.Unlock()

		// Broadcast presence diff
		h.broadcastPresenceDiff(msg.Topic, conn, state, "join")

	case "untrack":
		h.mu.Lock()
		if h.presenceState[msg.Topic] != nil {
			delete(h.presenceState[msg.Topic], presenceKey)
		}
		h.mu.Unlock()

		h.broadcastPresenceDiff(msg.Topic, conn, state, "leave")
	}

	h.sendReply(conn, msg.Topic, msg.Ref, "ok", map[string]interface{}{})
}

// handleAccessToken processes access token refresh
func (h *RealtimeHandler) handleAccessToken(conn *websocket.Conn, state *ClientState, msg *PhoenixMessage) {
	// Parse access token from payload
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		h.sendReply(conn, msg.Topic, msg.Ref, "error", map[string]interface{}{
			"reason": "invalid payload",
		})
		return
	}

	accessToken, _ := payload["access_token"].(string)
	if accessToken == "" {
		h.sendReply(conn, msg.Topic, msg.Ref, "error", map[string]interface{}{
			"reason": "missing access_token",
		})
		return
	}

	// Validate and update token (simplified - in production, validate JWT)
	config := middleware.DefaultAPIKeyConfig()
	if accessToken == config.AnonKey {
		state.role = "anon"
	} else if accessToken == config.ServiceKey {
		state.role = "service_role"
	}

	h.sendReply(conn, msg.Topic, msg.Ref, "ok", map[string]interface{}{})
}

// sendReply sends a phx_reply message
func (h *RealtimeHandler) sendReply(conn *websocket.Conn, topic, ref, status string, response map[string]interface{}) {
	reply := PhoenixMessage{
		Topic: topic,
		Event: PhxReply,
		Payload: map[string]interface{}{
			"status":   status,
			"response": response,
		},
		Ref: ref,
	}
	conn.WriteJSON(reply)
}

// sendError sends an error message
func (h *RealtimeHandler) sendError(conn *websocket.Conn, topic, ref, code, message string) {
	reply := PhoenixMessage{
		Topic: topic,
		Event: PhxReply,
		Payload: map[string]interface{}{
			"status": "error",
			"response": map[string]interface{}{
				"code":    code,
				"message": message,
			},
		},
		Ref: ref,
	}
	conn.WriteJSON(reply)
}

// sendPresenceState sends the current presence state to a client
func (h *RealtimeHandler) sendPresenceState(conn *websocket.Conn, topic string) {
	h.mu.RLock()
	state := h.presenceState[topic]
	h.mu.RUnlock()

	if state == nil {
		state = map[string]interface{}{}
	}

	msg := PhoenixMessage{
		Topic:   topic,
		Event:   PresenceState,
		Payload: state,
	}
	conn.WriteJSON(msg)
}

// broadcastPresenceDiff broadcasts presence changes to all clients on a topic
func (h *RealtimeHandler) broadcastPresenceDiff(topic string, sourceConn *websocket.Conn, state *ClientState, action string) {
	state.mu.RLock()
	presenceKey := state.presenceKey
	state.mu.RUnlock()

	if presenceKey == "" {
		return
	}

	h.mu.RLock()
	clients := h.topicClients[topic]
	presenceData := h.presenceState[topic]
	h.mu.RUnlock()

	var joins, leaves map[string]interface{}
	if action == "join" {
		joins = map[string]interface{}{}
		if presenceData != nil {
			if data, ok := presenceData[presenceKey]; ok {
				joins[presenceKey] = data
			}
		}
		leaves = map[string]interface{}{}
	} else {
		joins = map[string]interface{}{}
		leaves = map[string]interface{}{
			presenceKey: map[string]interface{}{
				"metas": []map[string]interface{}{},
			},
		}
	}

	msg := PhoenixMessage{
		Topic: topic,
		Event: PresenceDiff,
		Payload: map[string]interface{}{
			"joins":  joins,
			"leaves": leaves,
		},
	}

	for client := range clients {
		client.WriteJSON(msg)
	}
}

// cleanupClient removes a client and cleans up subscriptions
func (h *RealtimeHandler) cleanupClient(conn *websocket.Conn, state *ClientState) {
	state.mu.RLock()
	topics := make([]string, 0, len(state.subscribedTopics))
	for topic, sub := range state.subscribedTopics {
		topics = append(topics, topic)
		// Broadcast presence leave if applicable
		if sub.PresenceConfig != nil {
			h.broadcastPresenceDiff(topic, conn, state, "leave")
		}
	}
	state.mu.RUnlock()

	h.mu.Lock()
	delete(h.clients, conn)
	for _, topic := range topics {
		if h.topicClients[topic] != nil {
			delete(h.topicClients[topic], conn)
			if len(h.topicClients[topic]) == 0 {
				delete(h.topicClients, topic)
			}
		}
	}
	h.mu.Unlock()
}

// BroadcastToTopic sends a message to all clients subscribed to a topic
func (h *RealtimeHandler) BroadcastToTopic(topic string, event string, payload interface{}) {
	h.mu.RLock()
	clients := h.topicClients[topic]
	h.mu.RUnlock()

	msg := PhoenixMessage{
		Topic: topic,
		Event: event,
		Payload: map[string]interface{}{
			"type":    "broadcast",
			"event":   event,
			"payload": payload,
		},
	}

	for client := range clients {
		client.WriteJSON(msg)
	}
}

// BroadcastAll sends a message to all connected clients
func (h *RealtimeHandler) BroadcastAll(message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.WriteJSON(message)
	}
}

// generateUUID generates a simple UUID-like string
func generateUUID() string {
	return time.Now().Format("20060102150405.000000000")
}

// generateRef generates a unique reference string
func generateRef() string {
	return time.Now().Format("150405.000000000")
}
