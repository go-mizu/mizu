package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a WebSocket message for testing.
type WSMessage struct {
	Op int             `json:"op"`
	T  string          `json:"t,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
	S  int64           `json:"s,omitempty"`
}

// WSClient wraps a WebSocket connection with a message channel for testing.
type WSClient struct {
	conn     *websocket.Conn
	messages chan *WSMessage
	done     chan struct{}
	closed   bool
	mu       sync.Mutex
}

// newWSClient creates a new WebSocket client that reads messages into a channel.
func newWSClient(conn *websocket.Conn) *WSClient {
	c := &WSClient{
		conn:     conn,
		messages: make(chan *WSMessage, 100),
		done:     make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// readLoop continuously reads messages from the WebSocket connection.
func (c *WSClient) readLoop() {
	defer close(c.done)
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		// Handle multiple messages in one frame (newline separated)
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			var msg WSMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			select {
			case c.messages <- &msg:
			default:
				// Drop message if channel is full
			}
		}
	}
}

// WaitForMessage waits for a specific message type with timeout.
func (c *WSClient) WaitForMessage(msgType string, timeout time.Duration) *WSMessage {
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.messages:
			if msg.T == msgType {
				return msg
			}
		case <-deadline:
			return nil
		case <-c.done:
			return nil
		}
	}
}

// ReadMessage reads the next message with timeout.
func (c *WSClient) ReadMessage(timeout time.Duration) *WSMessage {
	select {
	case msg := <-c.messages:
		return msg
	case <-time.After(timeout):
		return nil
	case <-c.done:
		return nil
	}
}

// DrainMessages removes all pending messages from the queue.
func (c *WSClient) DrainMessages(timeout time.Duration) int {
	count := 0
	deadline := time.After(timeout)
	for {
		select {
		case <-c.messages:
			count++
		case <-deadline:
			return count
		case <-c.done:
			return count
		default:
			// No more messages in queue, wait a bit then return
			select {
			case <-time.After(100 * time.Millisecond):
				return count
			case <-c.messages:
				count++
			case <-c.done:
				return count
			}
		}
	}
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		c.conn.Close()
	}
}

// connectWebSocket establishes a WebSocket connection for testing.
func connectWebSocket(t *testing.T, server *httptest.Server, token string) *WSClient {
	t.Helper()

	// Convert http to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + token

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v (response: %v)", err, resp)
	}

	return newWSClient(conn)
}

// TestWS_Connection tests basic WebSocket connection.
func TestWS_Connection(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Start HTTP test server
	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register and get token
	token := registerAndGetToken(t, srv.app, "wsuser")

	// Connect WebSocket
	client := connectWebSocket(t, ts, token)
	defer client.Close()

	// Should receive HELLO first (op 10)
	msg := client.ReadMessage(5 * time.Second)
	if msg == nil {
		t.Fatal("timeout waiting for HELLO")
	}
	if msg.Op != 10 {
		t.Errorf("expected HELLO (op 10), got op %d", msg.Op)
	}

	// Should receive READY event
	msg = client.ReadMessage(5 * time.Second)
	if msg == nil {
		t.Fatal("timeout waiting for READY")
	}
	if msg.Op != 0 || msg.T != "READY" {
		t.Errorf("expected READY event, got op=%d, t=%s", msg.Op, msg.T)
	}
}

// TestWS_InvalidToken tests WebSocket with invalid token.
func TestWS_InvalidToken(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=invalid-token"
	dialer := websocket.Dialer{}
	_, resp, err := dialer.Dial(wsURL, nil)

	if err == nil {
		t.Error("expected WebSocket connection to fail with invalid token")
	}

	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestWS_MessageCreate tests receiving MESSAGE_CREATE via WebSocket.
func TestWS_MessageCreate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register user
	token := registerAndGetToken(t, srv.app, "msgcreateuser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "WS Test Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "ws-test", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Connect WebSocket
	client := connectWebSocket(t, ts, token)
	defer client.Close()

	// Drain initial messages (HELLO, READY)
	client.DrainMessages(2 * time.Second)

	// Send a message via HTTP API
	msgBody := map[string]interface{}{"content": "Hello via WebSocket!"}
	msgRec := doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", msgBody, token)
	if msgRec.Code != http.StatusOK && msgRec.Code != http.StatusCreated {
		t.Fatalf("send message failed: %s", msgRec.Body.String())
	}

	// Should receive MESSAGE_CREATE via WebSocket
	msg := client.WaitForMessage("MESSAGE_CREATE", 5*time.Second)
	if msg == nil {
		t.Fatal("did not receive MESSAGE_CREATE")
	}

	// Verify message content
	var msgData map[string]interface{}
	if err := json.Unmarshal(msg.D, &msgData); err != nil {
		t.Fatalf("failed to parse message data: %v", err)
	}

	if msgData["content"] != "Hello via WebSocket!" {
		t.Errorf("message content = %v, want 'Hello via WebSocket!'", msgData["content"])
	}
	if msgData["channel_id"] != channelID {
		t.Errorf("channel_id = %v, want %s", msgData["channel_id"], channelID)
	}
}

// TestWS_MultiUserMessaging tests real-time messaging between two users.
func TestWS_MultiUserMessaging(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register two users
	aliceToken := registerAndGetToken(t, srv.app, "wsalice")
	bobToken := registerAndGetToken(t, srv.app, "wsbob")

	// Alice creates a server
	serverBody := map[string]interface{}{"name": "Multi User Server", "is_public": true}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Create channel
	channelBody := map[string]interface{}{"name": "multi-chat", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, aliceToken)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Bob joins the server
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)

	// Both connect via WebSocket
	aliceClient := connectWebSocket(t, ts, aliceToken)
	defer aliceClient.Close()

	bobClient := connectWebSocket(t, ts, bobToken)
	defer bobClient.Close()

	// Drain initial messages
	aliceClient.DrainMessages(2 * time.Second)
	bobClient.DrainMessages(2 * time.Second)

	// Alice sends a message
	msgBody := map[string]interface{}{"content": "Hello Bob!"}
	doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", msgBody, aliceToken)

	// Both should receive the message
	var wg sync.WaitGroup
	var aliceReceived, bobReceived bool
	var mu sync.Mutex

	wg.Add(2)

	go func() {
		defer wg.Done()
		msg := aliceClient.WaitForMessage("MESSAGE_CREATE", 5*time.Second)
		if msg != nil {
			var data map[string]interface{}
			json.Unmarshal(msg.D, &data)
			if data["content"] == "Hello Bob!" {
				mu.Lock()
				aliceReceived = true
				mu.Unlock()
			}
		}
	}()

	go func() {
		defer wg.Done()
		msg := bobClient.WaitForMessage("MESSAGE_CREATE", 5*time.Second)
		if msg != nil {
			var data map[string]interface{}
			json.Unmarshal(msg.D, &data)
			if data["content"] == "Hello Bob!" {
				mu.Lock()
				bobReceived = true
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	if !aliceReceived {
		t.Error("Alice did not receive her own message")
	}
	if !bobReceived {
		t.Error("Bob did not receive Alice's message")
	}
}

// TestWS_MessageUpdate tests receiving MESSAGE_UPDATE via WebSocket.
func TestWS_MessageUpdate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	token := registerAndGetToken(t, srv.app, "updateuser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "Update Test Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "update-test", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Create a message first
	msgBody := map[string]interface{}{"content": "Original message"}
	msgRec := doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", msgBody, token)
	var msgResp map[string]interface{}
	parseResponse(t, msgRec, &msgResp)
	messageID := msgResp["data"].(map[string]interface{})["id"].(string)

	// Connect WebSocket
	client := connectWebSocket(t, ts, token)
	defer client.Close()

	// Drain initial messages
	client.DrainMessages(2 * time.Second)

	// Update the message
	updateBody := map[string]interface{}{"content": "Updated message"}
	doRequest(t, srv.app, "PATCH", "/api/v1/channels/"+channelID+"/messages/"+messageID, updateBody, token)

	// Should receive MESSAGE_UPDATE
	msg := client.WaitForMessage("MESSAGE_UPDATE", 5*time.Second)
	if msg == nil {
		t.Fatal("did not receive MESSAGE_UPDATE")
	}

	var msgData map[string]interface{}
	json.Unmarshal(msg.D, &msgData)

	if msgData["content"] != "Updated message" {
		t.Errorf("updated content = %v, want 'Updated message'", msgData["content"])
	}
}

// TestWS_MessageDelete tests receiving MESSAGE_DELETE via WebSocket.
func TestWS_MessageDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	token := registerAndGetToken(t, srv.app, "deleteuser")

	// Create server and channel
	serverBody := map[string]interface{}{"name": "Delete Test Server"}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, token)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "delete-test", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, token)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Create a message
	msgBody := map[string]interface{}{"content": "Message to delete"}
	msgRec := doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", msgBody, token)
	var msgResp map[string]interface{}
	parseResponse(t, msgRec, &msgResp)
	messageID := msgResp["data"].(map[string]interface{})["id"].(string)

	// Connect WebSocket
	client := connectWebSocket(t, ts, token)
	defer client.Close()

	// Drain initial messages
	client.DrainMessages(2 * time.Second)

	// Delete the message
	doRequest(t, srv.app, "DELETE", "/api/v1/channels/"+channelID+"/messages/"+messageID, nil, token)

	// Should receive MESSAGE_DELETE
	msg := client.WaitForMessage("MESSAGE_DELETE", 5*time.Second)
	if msg == nil {
		t.Fatal("did not receive MESSAGE_DELETE")
	}

	var msgData map[string]interface{}
	json.Unmarshal(msg.D, &msgData)

	if msgData["id"] != messageID {
		t.Errorf("deleted message id = %v, want %s", msgData["id"], messageID)
	}
}

// TestWS_Reconnect tests WebSocket reconnection.
func TestWS_Reconnect(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	token := registerAndGetToken(t, srv.app, "reconnectuser")

	// First connection
	client1 := connectWebSocket(t, ts, token)

	// Drain HELLO and READY
	client1.DrainMessages(2 * time.Second)

	// Close first connection
	client1.Close()

	// Small delay
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	client2 := connectWebSocket(t, ts, token)
	defer client2.Close()

	// Should receive HELLO again
	msg := client2.ReadMessage(5 * time.Second)
	if msg == nil {
		t.Fatal("timeout waiting for HELLO on reconnect")
	}
	if msg.Op != 10 {
		t.Errorf("expected HELLO on reconnect, got op %d", msg.Op)
	}
}

// TestWS_RealTimeFlow tests the complete real-time messaging flow.
func TestWS_RealTimeFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Create users
	aliceToken := registerAndGetToken(t, srv.app, "flowalice2")
	bobToken := registerAndGetToken(t, srv.app, "flowbob2")

	// Alice creates server and channel
	serverBody := map[string]interface{}{"name": "Flow Server", "is_public": true}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	channelBody := map[string]interface{}{"name": "flow-chat", "type": "text"}
	channelRec := doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/channels", channelBody, aliceToken)
	var channelResp map[string]interface{}
	parseResponse(t, channelRec, &channelResp)
	channelID := channelResp["data"].(map[string]interface{})["id"].(string)

	// Bob joins
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)

	// Connect both via WebSocket
	aliceClient := connectWebSocket(t, ts, aliceToken)
	defer aliceClient.Close()
	bobClient := connectWebSocket(t, ts, bobToken)
	defer bobClient.Close()

	// Drain initial messages
	aliceClient.DrainMessages(2 * time.Second)
	bobClient.DrainMessages(2 * time.Second)

	// Test conversation flow
	messages := []struct {
		sender  string
		token   string
		content string
	}{
		{"Alice", aliceToken, "Hi Bob!"},
		{"Bob", bobToken, "Hey Alice!"},
		{"Alice", aliceToken, "How are you?"},
		{"Bob", bobToken, "Great, thanks!"},
	}

	for _, m := range messages {
		// Send message
		body := map[string]interface{}{"content": m.content}
		rec := doRequest(t, srv.app, "POST", "/api/v1/channels/"+channelID+"/messages", body, m.token)
		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("failed to send message from %s: %s", m.sender, rec.Body.String())
		}

		// Both should receive it
		aliceMsg := aliceClient.WaitForMessage("MESSAGE_CREATE", 5*time.Second)
		bobMsg := bobClient.WaitForMessage("MESSAGE_CREATE", 5*time.Second)

		if aliceMsg == nil || bobMsg == nil {
			t.Fatalf("message from %s not received by both users", m.sender)
		}

		// Verify content
		var aliceData, bobData map[string]interface{}
		json.Unmarshal(aliceMsg.D, &aliceData)
		json.Unmarshal(bobMsg.D, &bobData)

		if aliceData["content"] != m.content {
			t.Errorf("Alice received wrong content: %v", aliceData["content"])
		}
		if bobData["content"] != m.content {
			t.Errorf("Bob received wrong content: %v", bobData["content"])
		}
	}

	// Verify all messages are persisted
	listRec := doRequest(t, srv.app, "GET", "/api/v1/channels/"+channelID+"/messages?limit=10", nil, aliceToken)
	var listResp map[string]interface{}
	parseResponse(t, listRec, &listResp)

	msgs := listResp["data"].([]interface{})
	if len(msgs) != 4 {
		t.Errorf("expected 4 messages, got %d", len(msgs))
	}
}

// TestWS_OnlineUsers tests the online users API endpoint.
func TestWS_OnlineUsers(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register users
	aliceToken := registerAndGetToken(t, srv.app, "onlinealice")
	bobToken := registerAndGetToken(t, srv.app, "onlinebob")
	charlieToken := registerAndGetToken(t, srv.app, "onlinecharlie")

	// Alice creates a server
	serverBody := map[string]interface{}{"name": "Online Test Server", "is_public": true}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Bob and Charlie join the server
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, charlieToken)

	// Initially, check online users - all should be offline
	rec := doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/online", nil, aliceToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("get online users failed: %s", rec.Body.String())
	}

	var resp map[string]interface{}
	parseResponse(t, rec, &resp)
	data := resp["data"].(map[string]interface{})

	// All users should be offline initially
	offline := data["offline"].([]interface{})
	if len(offline) != 3 {
		t.Errorf("expected 3 offline users, got %d", len(offline))
	}

	// Connect Alice via WebSocket
	aliceClient := connectWebSocket(t, ts, aliceToken)
	defer aliceClient.Close()
	aliceClient.DrainMessages(2 * time.Second)

	// Wait for connection to be registered
	time.Sleep(200 * time.Millisecond)

	// Check online users - Alice should be online
	rec = doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/online", nil, aliceToken)
	parseResponse(t, rec, &resp)
	data = resp["data"].(map[string]interface{})

	online := data["online"].([]interface{})
	offline = data["offline"].([]interface{})

	if len(online) != 1 {
		t.Errorf("expected 1 online user, got %d", len(online))
	}
	if len(offline) != 2 {
		t.Errorf("expected 2 offline users, got %d", len(offline))
	}

	// Connect Bob
	bobClient := connectWebSocket(t, ts, bobToken)
	defer bobClient.Close()
	bobClient.DrainMessages(2 * time.Second)

	time.Sleep(200 * time.Millisecond)

	// Check again - Alice and Bob should be online
	rec = doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/online", nil, aliceToken)
	parseResponse(t, rec, &resp)
	data = resp["data"].(map[string]interface{})

	online = data["online"].([]interface{})
	offline = data["offline"].([]interface{})

	if len(online) != 2 {
		t.Errorf("expected 2 online users, got %d", len(online))
	}
	if len(offline) != 1 {
		t.Errorf("expected 1 offline user (Charlie), got %d", len(offline))
	}
}

// TestWS_PresenceUpdate tests receiving PRESENCE_UPDATE via WebSocket.
func TestWS_PresenceUpdate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register users
	aliceToken := registerAndGetToken(t, srv.app, "presencealice")
	bobToken := registerAndGetToken(t, srv.app, "presencebob")

	// Alice creates a server
	serverBody := map[string]interface{}{"name": "Presence Test Server", "is_public": true}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Bob joins the server
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)

	// Alice connects first
	aliceClient := connectWebSocket(t, ts, aliceToken)
	defer aliceClient.Close()
	aliceClient.DrainMessages(2 * time.Second)

	// Bob connects - Alice should receive a PRESENCE_UPDATE
	bobClient := connectWebSocket(t, ts, bobToken)
	defer bobClient.Close()
	bobClient.DrainMessages(2 * time.Second)

	// Alice should receive PRESENCE_UPDATE for Bob coming online
	msg := aliceClient.WaitForMessage("PRESENCE_UPDATE", 3*time.Second)
	if msg == nil {
		t.Fatal("Alice did not receive PRESENCE_UPDATE when Bob connected")
	}

	var presenceData map[string]interface{}
	if err := json.Unmarshal(msg.D, &presenceData); err != nil {
		t.Fatalf("failed to parse presence data: %v", err)
	}

	if presenceData["status"] != "online" {
		t.Errorf("expected status 'online', got %v", presenceData["status"])
	}

	// Bob disconnects - Alice should receive PRESENCE_UPDATE for offline
	bobClient.Close()

	// Wait a bit for the disconnect to be processed
	time.Sleep(200 * time.Millisecond)

	msg = aliceClient.WaitForMessage("PRESENCE_UPDATE", 3*time.Second)
	if msg == nil {
		t.Fatal("Alice did not receive PRESENCE_UPDATE when Bob disconnected")
	}

	if err := json.Unmarshal(msg.D, &presenceData); err != nil {
		t.Fatalf("failed to parse presence data: %v", err)
	}

	if presenceData["status"] != "offline" {
		t.Errorf("expected status 'offline', got %v", presenceData["status"])
	}
}

// TestWS_MultiUserPresence tests presence updates with multiple users.
func TestWS_MultiUserPresence(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.app)
	defer ts.Close()

	// Register users
	aliceToken := registerAndGetToken(t, srv.app, "multialice")
	bobToken := registerAndGetToken(t, srv.app, "multibob")
	charlieToken := registerAndGetToken(t, srv.app, "multicharlie")

	// Alice creates a server
	serverBody := map[string]interface{}{"name": "Multi Presence Server", "is_public": true}
	serverRec := doRequest(t, srv.app, "POST", "/api/v1/servers", serverBody, aliceToken)
	var serverResp map[string]interface{}
	parseResponse(t, serverRec, &serverResp)
	serverID := serverResp["data"].(map[string]interface{})["id"].(string)

	// Bob and Charlie join the server
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, bobToken)
	doRequest(t, srv.app, "POST", "/api/v1/servers/"+serverID+"/join", nil, charlieToken)

	// All users connect
	aliceClient := connectWebSocket(t, ts, aliceToken)
	defer aliceClient.Close()
	aliceClient.DrainMessages(2 * time.Second)

	bobClient := connectWebSocket(t, ts, bobToken)
	defer bobClient.Close()
	bobClient.DrainMessages(2 * time.Second)

	charlieClient := connectWebSocket(t, ts, charlieToken)
	defer charlieClient.Close()
	charlieClient.DrainMessages(2 * time.Second)

	// Wait for all connections to be registered
	time.Sleep(300 * time.Millisecond)

	// Verify all users are online
	rec := doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/online", nil, aliceToken)
	var resp map[string]interface{}
	parseResponse(t, rec, &resp)
	data := resp["data"].(map[string]interface{})

	online := data["online"].([]interface{})
	if len(online) != 3 {
		t.Errorf("expected 3 online users, got %d", len(online))
	}

	// Verify the offline list is empty or nil
	offlineRaw := data["offline"]
	if offlineRaw != nil {
		offline := offlineRaw.([]interface{})
		if len(offline) != 0 {
			t.Errorf("expected 0 offline users, got %d", len(offline))
		}
	}

	// Charlie disconnects
	charlieClient.Close()
	time.Sleep(200 * time.Millisecond)

	// Drain any presence updates
	aliceClient.DrainMessages(500 * time.Millisecond)
	bobClient.DrainMessages(500 * time.Millisecond)

	// Verify Charlie is now offline
	rec = doRequest(t, srv.app, "GET", "/api/v1/servers/"+serverID+"/online", nil, aliceToken)
	parseResponse(t, rec, &resp)
	data = resp["data"].(map[string]interface{})

	online = data["online"].([]interface{})
	offline := data["offline"].([]interface{})

	if len(online) != 2 {
		t.Errorf("expected 2 online users, got %d", len(online))
	}
	if len(offline) != 1 {
		t.Errorf("expected 1 offline user (Charlie), got %d", len(offline))
	}
}
