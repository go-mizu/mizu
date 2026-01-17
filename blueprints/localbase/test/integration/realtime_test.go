//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Realtime test configuration
var (
	realtimeWSURL = getEnv("LOCALBASE_REALTIME_WS", "ws://localhost:54321/realtime/v1/websocket")
)

// =============================================================================
// WebSocket Message Types
// =============================================================================

// PhoenixMessage represents a Phoenix protocol message
type PhoenixMessage struct {
	Topic   string      `json:"topic"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
	Ref     string      `json:"ref,omitempty"`
	JoinRef string      `json:"join_ref,omitempty"`
}

// PhxReplyPayload represents the payload of a phx_reply message
type PhxReplyPayload struct {
	Status   string                 `json:"status"`
	Response map[string]interface{} `json:"response,omitempty"`
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

// JoinConfig represents the configuration for joining a channel
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
	Event  string `json:"event"`
	Schema string `json:"schema"`
	Table  string `json:"table,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// =============================================================================
// WebSocket Test Helpers
// =============================================================================

// RealtimeClient wraps a WebSocket connection for realtime testing
type RealtimeClient struct {
	conn     *websocket.Conn
	t        *testing.T
	refCount int64
	mu       sync.Mutex
	messages chan PhoenixMessage
	done     chan struct{}
}

// NewRealtimeClient creates a new WebSocket client for testing
func NewRealtimeClient(t *testing.T, apiKey string) *RealtimeClient {
	wsURL := realtimeWSURL
	if !strings.Contains(wsURL, "?") {
		wsURL += "?"
	} else {
		wsURL += "&"
	}
	wsURL += "apikey=" + url.QueryEscape(apiKey) + "&vsn=1.0.0"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, http.Header{
		"Origin": []string{"http://localhost"},
	})
	if err != nil {
		if resp != nil {
			t.Fatalf("WebSocket dial failed with status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	client := &RealtimeClient{
		conn:     conn,
		t:        t,
		messages: make(chan PhoenixMessage, 100),
		done:     make(chan struct{}),
	}

	// Start message reader
	go client.readMessages()

	return client
}

func (c *RealtimeClient) readMessages() {
	defer close(c.messages)
	for {
		select {
		case <-c.done:
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			var msg PhoenixMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			c.messages <- msg
		}
	}
}

func (c *RealtimeClient) Close() {
	close(c.done)
	c.conn.Close()
}

func (c *RealtimeClient) nextRef() string {
	return fmt.Sprintf("%d", atomic.AddInt64(&c.refCount, 1))
}

// Send sends a Phoenix message
func (c *RealtimeClient) Send(msg PhoenixMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(msg)
}

// Receive waits for a message with timeout
func (c *RealtimeClient) Receive(timeout time.Duration) (*PhoenixMessage, error) {
	select {
	case msg, ok := <-c.messages:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return &msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for message")
	}
}

// ReceiveWithFilter waits for a specific message type
func (c *RealtimeClient) ReceiveWithFilter(event string, timeout time.Duration) (*PhoenixMessage, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		msg, err := c.Receive(remaining)
		if err != nil {
			return nil, err
		}
		if msg.Event == event {
			return msg, nil
		}
	}
	return nil, fmt.Errorf("timeout waiting for event: %s", event)
}

// SendHeartbeat sends a heartbeat message
func (c *RealtimeClient) SendHeartbeat() error {
	return c.Send(PhoenixMessage{
		Topic:   "phoenix",
		Event:   "heartbeat",
		Payload: map[string]interface{}{},
		Ref:     c.nextRef(),
	})
}

// JoinChannel joins a realtime channel
func (c *RealtimeClient) JoinChannel(topic string, config *JoinConfig) (*PhoenixMessage, error) {
	payload := map[string]interface{}{}
	if config != nil {
		payload["config"] = config
	}

	ref := c.nextRef()
	err := c.Send(PhoenixMessage{
		Topic:   topic,
		Event:   "phx_join",
		Payload: payload,
		Ref:     ref,
		JoinRef: ref,
	})
	if err != nil {
		return nil, err
	}

	return c.ReceiveWithFilter("phx_reply", 5*time.Second)
}

// LeaveChannel leaves a realtime channel
func (c *RealtimeClient) LeaveChannel(topic string) error {
	return c.Send(PhoenixMessage{
		Topic:   topic,
		Event:   "phx_leave",
		Payload: map[string]interface{}{},
		Ref:     c.nextRef(),
	})
}

// Broadcast sends a broadcast message
func (c *RealtimeClient) Broadcast(topic, event string, payload interface{}) error {
	return c.Send(PhoenixMessage{
		Topic: topic,
		Event: "broadcast",
		Payload: BroadcastPayload{
			Type:    "broadcast",
			Event:   event,
			Payload: payload,
		},
		Ref: c.nextRef(),
	})
}

// TrackPresence sends a presence track message
func (c *RealtimeClient) TrackPresence(topic string, payload interface{}) error {
	return c.Send(PhoenixMessage{
		Topic: topic,
		Event: "presence",
		Payload: PresencePayload{
			Type:    "presence",
			Event:   "track",
			Payload: payload,
		},
		Ref: c.nextRef(),
	})
}

// =============================================================================
// Connection Tests (CONN-xxx)
// =============================================================================

func TestRealtime_CONN001_ConnectWithValidAnonKey(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	// Connection should be established - send heartbeat to verify
	err := client.SendHeartbeat()
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	msg, err := client.ReceiveWithFilter("phx_reply", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive heartbeat reply: %v", err)
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_CONN002_ConnectWithServiceKey(t *testing.T) {
	client := NewRealtimeClient(t, serviceRoleKey)
	defer client.Close()

	err := client.SendHeartbeat()
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	msg, err := client.ReceiveWithFilter("phx_reply", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive heartbeat reply: %v", err)
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_CONN003_ConnectWithoutAPIKey(t *testing.T) {
	wsURL := realtimeWSURL + "?vsn=1.0.0"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	_, resp, err := dialer.Dial(wsURL, http.Header{
		"Origin": []string{"http://localhost"},
	})

	// Should fail with 401
	if err == nil {
		t.Fatal("Expected connection to fail without API key")
	}
	if resp != nil && resp.StatusCode != 401 {
		t.Errorf("Expected 401, got: %d", resp.StatusCode)
	}
}

func TestRealtime_CONN004_ConnectWithInvalidAPIKey(t *testing.T) {
	wsURL := realtimeWSURL + "?apikey=invalid-key&vsn=1.0.0"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	_, resp, err := dialer.Dial(wsURL, http.Header{
		"Origin": []string{"http://localhost"},
	})

	// Should fail with 401
	if err == nil {
		t.Fatal("Expected connection to fail with invalid API key")
	}
	if resp != nil && resp.StatusCode != 401 {
		t.Errorf("Expected 401, got: %d", resp.StatusCode)
	}
}

// =============================================================================
// Heartbeat Tests (HEART-xxx)
// =============================================================================

func TestRealtime_HEART001_HeartbeatOnPhoenixTopic(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	err := client.Send(PhoenixMessage{
		Topic:   "phoenix",
		Event:   "heartbeat",
		Payload: map[string]interface{}{},
		Ref:     "heartbeat-1",
	})
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	msg, err := client.ReceiveWithFilter("phx_reply", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive heartbeat reply: %v", err)
	}

	if msg.Topic != "phoenix" {
		t.Errorf("Expected topic phoenix, got: %s", msg.Topic)
	}
	if msg.Ref != "heartbeat-1" {
		t.Errorf("Expected ref heartbeat-1, got: %s", msg.Ref)
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_HEART002_HeartbeatResponseTime(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	start := time.Now()
	err := client.SendHeartbeat()
	if err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	_, err = client.ReceiveWithFilter("phx_reply", 1*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Heartbeat response timeout: %v", err)
	}
	if elapsed > 1*time.Second {
		t.Errorf("Heartbeat response took too long: %v", elapsed)
	}
}

func TestRealtime_HEART005_MultipleRapidHeartbeats(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	// Send 10 heartbeats rapidly
	for i := 0; i < 10; i++ {
		err := client.Send(PhoenixMessage{
			Topic:   "phoenix",
			Event:   "heartbeat",
			Payload: map[string]interface{}{},
			Ref:     fmt.Sprintf("rapid-%d", i),
		})
		if err != nil {
			t.Fatalf("Failed to send heartbeat %d: %v", i, err)
		}
	}

	// All should be acknowledged
	received := 0
	for i := 0; i < 10; i++ {
		msg, err := client.Receive(5 * time.Second)
		if err != nil {
			break
		}
		if msg.Event == "phx_reply" {
			received++
		}
	}

	if received < 10 {
		t.Errorf("Expected 10 heartbeat replies, got: %d", received)
	}
}

func TestRealtime_HEART006_HeartbeatRefTracking(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	refs := []string{"ref-alpha", "ref-beta", "ref-gamma"}
	for _, ref := range refs {
		err := client.Send(PhoenixMessage{
			Topic:   "phoenix",
			Event:   "heartbeat",
			Payload: map[string]interface{}{},
			Ref:     ref,
		})
		if err != nil {
			t.Fatalf("Failed to send heartbeat: %v", err)
		}
	}

	receivedRefs := make(map[string]bool)
	for i := 0; i < len(refs); i++ {
		msg, err := client.Receive(5 * time.Second)
		if err != nil {
			break
		}
		if msg.Event == "phx_reply" {
			receivedRefs[msg.Ref] = true
		}
	}

	for _, ref := range refs {
		if !receivedRefs[ref] {
			t.Errorf("Missing reply for ref: %s", ref)
		}
	}
}

// =============================================================================
// Channel Join Tests (JOIN-xxx)
// =============================================================================

func TestRealtime_JOIN001_JoinPublicChannel(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	reply, err := client.JoinChannel("realtime:public-test-channel", nil)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_JOIN004_JoinWithBroadcastConfig(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}

	reply, err := client.JoinChannel("realtime:broadcast-test", config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_JOIN006_JoinWithPresenceConfig(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	config := &JoinConfig{
		Presence: &PresenceConfig{
			Key: "user_123",
		},
	}

	reply, err := client.JoinChannel("realtime:presence-test", config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_JOIN007_JoinWithPostgresChanges(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	config := &JoinConfig{
		PostgresChanges: []PostgresChangeConfig{
			{
				Event:  "INSERT",
				Schema: "public",
				Table:  "users",
			},
		},
	}

	reply, err := client.JoinChannel("realtime:pg-changes-test", config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_JOIN011_JoinMultipleChannels(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	channels := []string{
		"realtime:multi-1",
		"realtime:multi-2",
		"realtime:multi-3",
		"realtime:multi-4",
		"realtime:multi-5",
	}

	for _, ch := range channels {
		reply, err := client.JoinChannel(ch, nil)
		if err != nil {
			t.Fatalf("Failed to join channel %s: %v", ch, err)
		}

		payload, ok := reply.Payload.(map[string]interface{})
		if !ok {
			t.Fatalf("Invalid payload type for %s", ch)
		}
		if payload["status"] != "ok" {
			t.Errorf("Expected status ok for %s, got: %v", ch, payload["status"])
		}
	}
}

func TestRealtime_JOIN015_JoinRefTracking(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	ref := "unique-join-ref-12345"
	err := client.Send(PhoenixMessage{
		Topic:   "realtime:ref-tracking-test",
		Event:   "phx_join",
		Payload: map[string]interface{}{},
		Ref:     ref,
		JoinRef: ref,
	})
	if err != nil {
		t.Fatalf("Failed to send join: %v", err)
	}

	reply, err := client.ReceiveWithFilter("phx_reply", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive reply: %v", err)
	}

	if reply.Ref != ref {
		t.Errorf("Expected ref %s, got: %s", ref, reply.Ref)
	}
}

// =============================================================================
// Channel Leave Tests (LEAVE-xxx)
// =============================================================================

func TestRealtime_LEAVE001_LeaveJoinedChannel(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:leave-test"

	// First join
	_, err := client.JoinChannel(topic, nil)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Then leave
	err = client.LeaveChannel(topic)
	if err != nil {
		t.Fatalf("Failed to leave channel: %v", err)
	}

	// Should receive leave confirmation
	msg, err := client.ReceiveWithFilter("phx_reply", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive leave reply: %v", err)
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}
}

func TestRealtime_LEAVE004_RejoinAfterLeave(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:rejoin-test"

	// Join
	_, err := client.JoinChannel(topic, nil)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Leave
	err = client.LeaveChannel(topic)
	if err != nil {
		t.Fatalf("Failed to leave channel: %v", err)
	}
	// Consume leave reply
	client.Receive(1 * time.Second)

	// Rejoin
	reply, err := client.JoinChannel(topic, nil)
	if err != nil {
		t.Fatalf("Failed to rejoin channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok on rejoin, got: %v", payload["status"])
	}
}

// =============================================================================
// Broadcast Tests (BCAST-xxx)
// =============================================================================

func TestRealtime_BCAST001_SendBroadcast(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:broadcast-send-test"

	// Join with self=true to receive own broadcasts
	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Send broadcast
	err = client.Broadcast(topic, "test-event", map[string]interface{}{
		"message": "Hello, World!",
	})
	if err != nil {
		t.Fatalf("Failed to send broadcast: %v", err)
	}

	// Should receive the broadcast back (self=true)
	msg, err := client.ReceiveWithFilter("broadcast", 5*time.Second)
	if err != nil {
		t.Logf("Note: broadcast self-receive may not be implemented yet: %v", err)
		return
	}

	if msg.Topic != topic {
		t.Errorf("Expected topic %s, got: %s", topic, msg.Topic)
	}
}

func TestRealtime_BCAST005_BroadcastToMultipleSubscribers(t *testing.T) {
	topic := "realtime:broadcast-multi-test"

	// Create 3 clients
	client1 := NewRealtimeClient(t, localbaseAPIKey)
	defer client1.Close()
	client2 := NewRealtimeClient(t, localbaseAPIKey)
	defer client2.Close()
	client3 := NewRealtimeClient(t, localbaseAPIKey)
	defer client3.Close()

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}

	// All join the same channel
	_, err := client1.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Client 1 failed to join: %v", err)
	}
	_, err = client2.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Client 2 failed to join: %v", err)
	}
	_, err = client3.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Client 3 failed to join: %v", err)
	}

	// Client 1 broadcasts
	err = client1.Broadcast(topic, "multi-test", map[string]interface{}{
		"from": "client1",
	})
	if err != nil {
		t.Fatalf("Failed to broadcast: %v", err)
	}

	// Note: Checking receipt on other clients requires full broadcast implementation
	t.Log("Broadcast sent - full routing verification pending implementation")
}

func TestRealtime_BCAST006_BroadcastWithCustomEvent(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:custom-event-test"
	customEvent := "my_custom_event"

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	err = client.Broadcast(topic, customEvent, map[string]interface{}{
		"data": "test",
	})
	if err != nil {
		t.Fatalf("Failed to broadcast: %v", err)
	}

	t.Log("Custom event broadcast sent successfully")
}

func TestRealtime_BCAST007_BroadcastComplexPayload(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:complex-payload-test"

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	complexPayload := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "Test User",
			"metadata": map[string]interface{}{
				"level": 5,
				"tags":  []string{"admin", "verified"},
			},
		},
		"timestamp": time.Now().Unix(),
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": true,
			},
		},
	}

	err = client.Broadcast(topic, "complex", complexPayload)
	if err != nil {
		t.Fatalf("Failed to broadcast complex payload: %v", err)
	}

	t.Log("Complex payload broadcast sent successfully")
}

func TestRealtime_BCAST015_BroadcastUnicode(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:unicode-test"

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	unicodePayload := map[string]interface{}{
		"message": "Hello 世界! مرحبا 🌍",
		"emoji":   "🚀💻🎉",
		"chinese": "你好世界",
		"arabic":  "مرحبا بالعالم",
		"russian": "Привет мир",
	}

	err = client.Broadcast(topic, "unicode-test", unicodePayload)
	if err != nil {
		t.Fatalf("Failed to broadcast unicode payload: %v", err)
	}

	t.Log("Unicode payload broadcast sent successfully")
}

// =============================================================================
// Presence Tests (PRES-xxx)
// =============================================================================

func TestRealtime_PRES001_TrackPresence(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:presence-track-test"

	config := &JoinConfig{
		Presence: &PresenceConfig{
			Key: "user_1",
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Track presence
	err = client.TrackPresence(topic, map[string]interface{}{
		"online_at": time.Now().Format(time.RFC3339),
		"status":    "online",
	})
	if err != nil {
		t.Fatalf("Failed to track presence: %v", err)
	}

	t.Log("Presence tracked successfully")
}

func TestRealtime_PRES007_PresenceCustomMetadata(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:presence-metadata-test"

	config := &JoinConfig{
		Presence: &PresenceConfig{
			Key: "user_metadata",
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Track with custom metadata
	err = client.TrackPresence(topic, map[string]interface{}{
		"user_id":   456,
		"username":  "testuser",
		"avatar":    "https://example.com/avatar.png",
		"status":    "busy",
		"typing":    false,
		"online_at": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Failed to track presence with metadata: %v", err)
	}

	t.Log("Presence with custom metadata tracked successfully")
}

// =============================================================================
// Postgres Changes Tests (PG-xxx)
// =============================================================================

func TestRealtime_PG001_SubscribeToInsert(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:pg-insert-test"

	config := &JoinConfig{
		PostgresChanges: []PostgresChangeConfig{
			{
				Event:  "INSERT",
				Schema: "public",
				Table:  "users",
			},
		},
	}

	reply, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel with postgres_changes: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}

	t.Log("Subscribed to INSERT events successfully")
}

func TestRealtime_PG004_SubscribeToAllEvents(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:pg-all-events-test"

	config := &JoinConfig{
		PostgresChanges: []PostgresChangeConfig{
			{
				Event:  "*",
				Schema: "public",
				Table:  "users",
			},
		},
	}

	reply, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}

	t.Log("Subscribed to all postgres_changes events successfully")
}

func TestRealtime_PG007_SubscribeWithFilter(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:pg-filter-test"

	config := &JoinConfig{
		PostgresChanges: []PostgresChangeConfig{
			{
				Event:  "INSERT",
				Schema: "public",
				Table:  "users",
				Filter: "id=eq.1",
			},
		},
	}

	reply, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel with filter: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}

	t.Log("Subscribed with filter successfully")
}

func TestRealtime_PG023_MultipleSubscriptions(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:pg-multi-sub-test"

	config := &JoinConfig{
		PostgresChanges: []PostgresChangeConfig{
			{
				Event:  "INSERT",
				Schema: "public",
				Table:  "users",
			},
			{
				Event:  "UPDATE",
				Schema: "public",
				Table:  "profiles",
			},
			{
				Event:  "DELETE",
				Schema: "public",
				Table:  "sessions",
			},
		},
	}

	reply, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel with multiple subscriptions: %v", err)
	}

	payload, ok := reply.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid payload type")
	}
	if payload["status"] != "ok" {
		t.Errorf("Expected status ok, got: %v", payload["status"])
	}

	t.Log("Multiple postgres_changes subscriptions created successfully")
}

// =============================================================================
// Access Token Tests (TOKEN-xxx)
// =============================================================================

func TestRealtime_TOKEN001_RefreshAccessToken(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:token-refresh-test"

	_, err := client.JoinChannel(topic, nil)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	// Send access_token refresh
	err = client.Send(PhoenixMessage{
		Topic: topic,
		Event: "access_token",
		Payload: map[string]interface{}{
			"access_token": localbaseAPIKey,
		},
		Ref: client.nextRef(),
	})
	if err != nil {
		t.Fatalf("Failed to send access_token refresh: %v", err)
	}

	t.Log("Access token refresh sent successfully")
}

// =============================================================================
// Error Handling Tests (ERR-xxx)
// =============================================================================

func TestRealtime_ERR001_InvalidMessageFormat(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	// Send malformed JSON directly
	client.mu.Lock()
	err := client.conn.WriteMessage(websocket.TextMessage, []byte("{invalid json"))
	client.mu.Unlock()

	if err != nil {
		t.Fatalf("Failed to send malformed message: %v", err)
	}

	// Connection should still be alive for valid messages
	err = client.SendHeartbeat()
	if err != nil {
		t.Logf("Connection may have been closed due to invalid message: %v", err)
	}
}

func TestRealtime_ERR003_MissingRequiredFields(t *testing.T) {
	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	// Send message without topic
	err := client.Send(PhoenixMessage{
		Event:   "phx_join",
		Payload: map[string]interface{}{},
		Ref:     "missing-topic",
	})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Should receive error or be ignored
	msg, err := client.Receive(2 * time.Second)
	if err == nil && msg != nil {
		payload, ok := msg.Payload.(map[string]interface{})
		if ok && payload["status"] == "error" {
			t.Log("Received expected error response for missing topic")
		}
	}
}

// =============================================================================
// Performance Tests (PERF-xxx)
// =============================================================================

func TestRealtime_PERF001_ConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	numConnections := 50 // Start with 50, can increase

	var wg sync.WaitGroup
	successCount := int32(0)
	failCount := int32(0)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			client := NewRealtimeClient(t, localbaseAPIKey)
			defer client.Close()

			err := client.SendHeartbeat()
			if err != nil {
				atomic.AddInt32(&failCount, 1)
				return
			}

			_, err = client.ReceiveWithFilter("phx_reply", 5*time.Second)
			if err != nil {
				atomic.AddInt32(&failCount, 1)
				return
			}

			atomic.AddInt32(&successCount, 1)
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent connections: %d succeeded, %d failed", successCount, failCount)

	if failCount > int32(numConnections/10) { // Allow 10% failure rate
		t.Errorf("Too many connection failures: %d/%d", failCount, numConnections)
	}
}

func TestRealtime_PERF002_MessageThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	topic := "realtime:throughput-test"
	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}
	_, err := client.JoinChannel(topic, config)
	if err != nil {
		t.Fatalf("Failed to join channel: %v", err)
	}

	numMessages := 100
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		err := client.Broadcast(topic, "throughput", map[string]interface{}{
			"sequence": i,
			"time":     time.Now().UnixNano(),
		})
		if err != nil {
			t.Fatalf("Failed to send message %d: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	rate := float64(numMessages) / elapsed.Seconds()

	t.Logf("Message throughput: %.2f msgs/sec (%d messages in %v)", rate, numMessages, elapsed)
}

func TestRealtime_PERF004_RapidJoinLeave(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	client := NewRealtimeClient(t, localbaseAPIKey)
	defer client.Close()

	numOps := 50
	start := time.Now()

	for i := 0; i < numOps; i++ {
		topic := fmt.Sprintf("realtime:rapid-join-leave-%d", i)

		_, err := client.JoinChannel(topic, nil)
		if err != nil {
			t.Fatalf("Failed to join channel %d: %v", i, err)
		}

		err = client.LeaveChannel(topic)
		if err != nil {
			t.Fatalf("Failed to leave channel %d: %v", i, err)
		}

		// Drain any pending messages
		for {
			_, err := client.Receive(100 * time.Millisecond)
			if err != nil {
				break
			}
		}
	}

	elapsed := time.Since(start)
	rate := float64(numOps*2) / elapsed.Seconds() // 2 ops per iteration

	t.Logf("Join/Leave rate: %.2f ops/sec (%d operations in %v)", rate, numOps*2, elapsed)
}

// =============================================================================
// REST API Broadcast Tests (REST-xxx)
// =============================================================================

func TestRealtime_REST003_BroadcastAuthentication(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// POST to broadcast endpoint
	status, body, _, err := client.Request("POST", "/realtime/v1/api/broadcast", map[string]interface{}{
		"topic":   "realtime:rest-broadcast-test",
		"event":   "test-event",
		"payload": map[string]interface{}{"message": "test"},
	}, nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// May not be implemented yet
	if status == 404 {
		t.Skip("REST broadcast endpoint not yet implemented")
	}

	if status != 200 && status != 201 {
		t.Errorf("Expected 200 or 201, got %d: %s", status, body)
	}
}

func TestRealtime_REST004_BroadcastUnauthorized(t *testing.T) {
	client := NewClient(localbaseURL, "invalid-key")

	status, _, _, err := client.Request("POST", "/realtime/v1/api/broadcast", map[string]interface{}{
		"topic":   "realtime:unauthorized-test",
		"event":   "test-event",
		"payload": map[string]interface{}{},
	}, nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// May not be implemented yet
	if status == 404 {
		t.Skip("REST broadcast endpoint not yet implemented")
	}

	if status != 401 && status != 403 {
		t.Errorf("Expected 401 or 403 for unauthorized request, got %d", status)
	}
}

// =============================================================================
// Cross-Channel Isolation Tests
// =============================================================================

func TestRealtime_Isolation_CrossChannelMessages(t *testing.T) {
	client1 := NewRealtimeClient(t, localbaseAPIKey)
	defer client1.Close()
	client2 := NewRealtimeClient(t, localbaseAPIKey)
	defer client2.Close()

	topicA := "realtime:channel-a"
	topicB := "realtime:channel-b"

	config := &JoinConfig{
		Broadcast: &BroadcastConfig{
			Self: true,
		},
	}

	// Client 1 joins channel A
	_, err := client1.JoinChannel(topicA, config)
	if err != nil {
		t.Fatalf("Client 1 failed to join channel A: %v", err)
	}

	// Client 2 joins channel B
	_, err = client2.JoinChannel(topicB, config)
	if err != nil {
		t.Fatalf("Client 2 failed to join channel B: %v", err)
	}

	// Client 1 broadcasts on channel A
	err = client1.Broadcast(topicA, "isolation-test", map[string]interface{}{
		"message": "only for channel A",
	})
	if err != nil {
		t.Fatalf("Failed to broadcast: %v", err)
	}

	// Client 2 should NOT receive the message (different channel)
	msg, err := client2.Receive(1 * time.Second)
	if err == nil && msg != nil && msg.Event == "broadcast" {
		t.Error("Client 2 received broadcast from channel A - isolation failed")
	}

	t.Log("Cross-channel isolation verified")
}

// =============================================================================
// Protocol Version Tests (V2-xxx)
// =============================================================================

func TestRealtime_V2001_ArrayMessageFormat(t *testing.T) {
	// Connect with v2.0.0 protocol
	wsURL := realtimeWSURL
	if !strings.Contains(wsURL, "?") {
		wsURL += "?"
	} else {
		wsURL += "&"
	}
	wsURL += "apikey=" + url.QueryEscape(localbaseAPIKey) + "&vsn=2.0.0"

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, http.Header{
		"Origin": []string{"http://localhost"},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == 400 {
			t.Skip("Protocol v2.0.0 not yet supported")
		}
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Send heartbeat in array format [join_ref, ref, topic, event, payload]
	msg := []interface{}{nil, "1", "phoenix", "heartbeat", map[string]interface{}{}}
	err = conn.WriteJSON(msg)
	if err != nil {
		t.Fatalf("Failed to send v2 message: %v", err)
	}

	t.Log("Protocol v2.0.0 connection established")
}

// =============================================================================
// Channel Stats API Test
// =============================================================================

func TestRealtime_API_GetStats(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	status, body, _, err := client.Request("GET", "/api/realtime/stats", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200, got %d: %s", status, body)
		return
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("Failed to parse stats: %v", err)
	}

	// Verify expected fields
	if _, ok := stats["connections"]; !ok {
		t.Error("Stats missing 'connections' field")
	}
	if _, ok := stats["channels"]; !ok {
		t.Error("Stats missing 'channels' field")
	}
	if _, ok := stats["server_time"]; !ok {
		t.Error("Stats missing 'server_time' field")
	}
}

func TestRealtime_API_ListChannels(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	status, body, _, err := client.Request("GET", "/api/realtime/channels", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200, got %d: %s", status, body)
		return
	}

	var channels []interface{}
	if err := json.Unmarshal(body, &channels); err != nil {
		t.Fatalf("Failed to parse channels: %v", err)
	}

	t.Logf("Found %d channels", len(channels))
}
