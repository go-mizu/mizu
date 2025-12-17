package live

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/live/internal/ws"
)

// -----------------------------------------------------------------------------
// Message Tests
// -----------------------------------------------------------------------------

func TestEncodeMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		want    string
		wantErr bool
	}{
		{
			name: "basic message",
			msg: Message{
				Topic: "room:1",
				Data:  json.RawMessage(`{"text":"hello"}`),
			},
			want: `{"topic":"room:1","data":{"text":"hello"}}`,
		},
		{
			name: "topic only",
			msg: Message{
				Topic: "ping",
			},
			want: `{"topic":"ping"}`,
		},
		{
			name: "data only",
			msg: Message{
				Data: json.RawMessage(`{"foo":"bar"}`),
			},
			want: `{"data":{"foo":"bar"}}`,
		},
		{
			name: "empty message",
			msg:  Message{},
			want: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("encodeMessage() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecodeMessage(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantTopic string
		wantData  string
		wantErr   bool
	}{
		{
			name:      "basic message",
			data:      `{"topic":"room:1","data":{"text":"hello"}}`,
			wantTopic: "room:1",
			wantData:  `{"text":"hello"}`,
		},
		{
			name:      "topic only",
			data:      `{"topic":"ping"}`,
			wantTopic: "ping",
			wantData:  "",
		},
		{
			name:    "invalid json",
			data:    `{invalid`,
			wantErr: true,
		},
		{
			name:      "empty json",
			data:      `{}`,
			wantTopic: "",
			wantData:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, data, err := decodeMessage([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", topic, tt.wantTopic)
			}
			gotData := string(data)
			if gotData != tt.wantData {
				t.Errorf("data = %s, want %s", gotData, tt.wantData)
			}
		})
	}
}

func TestMessageRoundTrip(t *testing.T) {
	original := Message{
		Topic: "chat:room:123",
		Data:  json.RawMessage(`{"text":"Hello, World!"}`),
	}

	encoded, err := encodeMessage(original)
	if err != nil {
		t.Fatalf("encodeMessage() error = %v", err)
	}

	topic, data, err := decodeMessage(encoded)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}

	if topic != original.Topic {
		t.Errorf("Topic mismatch: got %s, want %s", topic, original.Topic)
	}
	if !bytes.Equal(data, original.Data) {
		t.Errorf("Data mismatch: got %s, want %s", data, original.Data)
	}
}

// -----------------------------------------------------------------------------
// Session Tests
// -----------------------------------------------------------------------------

func TestSession_ID(t *testing.T) {
	s := newSession("test-id", nil, 10, nil)
	if s.ID() != "test-id" {
		t.Errorf("ID() = %s, want test-id", s.ID())
	}
}

func TestSession_Value(t *testing.T) {
	type UserInfo struct {
		ID   string
		Role string
	}
	value := UserInfo{ID: "123", Role: "admin"}
	s := newSession("id", value, 10, nil)

	got, ok := s.Value().(UserInfo)
	if !ok {
		t.Fatal("Value() should be UserInfo type")
	}
	if got.ID != "123" {
		t.Errorf("Value().ID = %v, want 123", got.ID)
	}
	if got.Role != "admin" {
		t.Errorf("Value().Role = %v, want admin", got.Role)
	}
}

func TestSession_Value_Nil(t *testing.T) {
	s := newSession("id", nil, 10, nil)
	if s.Value() != nil {
		t.Errorf("Value() should be nil")
	}
}

func TestSession_Send_Success(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	msg := Message{Topic: "test", Data: json.RawMessage(`"hello"`)}
	err := s.Send(msg)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Verify message is in queue
	select {
	case received := <-s.sendCh:
		if received.Topic != msg.Topic {
			t.Errorf("received.Topic = %s, want %s", received.Topic, msg.Topic)
		}
	default:
		t.Error("expected message in queue")
	}
}

func TestSession_Send_Closed(t *testing.T) {
	s := newSession("id", nil, 10, nil)
	s.Close()

	err := s.Send(Message{Topic: "test"})
	if err != ErrSessionClosed {
		t.Errorf("Send() on closed session error = %v, want %v", err, ErrSessionClosed)
	}
}

func TestSession_Send_QueueFull(t *testing.T) {
	s := newSession("id", nil, 2, nil)

	// Fill the queue
	s.Send(Message{Topic: "1"})
	s.Send(Message{Topic: "2"})

	// This should fail and close the session
	err := s.Send(Message{Topic: "3"})
	if err != ErrQueueFull {
		t.Errorf("Send() with full queue error = %v, want %v", err, ErrQueueFull)
	}

	if !s.IsClosed() {
		t.Error("session should be closed after queue full")
	}

	// CloseError should return ErrQueueFull
	if s.CloseError() != ErrQueueFull {
		t.Errorf("CloseError() = %v, want %v", s.CloseError(), ErrQueueFull)
	}
}

func TestSession_Close(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	if s.IsClosed() {
		t.Error("new session should not be closed")
	}

	err := s.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !s.IsClosed() {
		t.Error("session should be closed after Close()")
	}
}

func TestSession_Close_Idempotent(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	s.Close()
	err := s.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestSession_CloseError(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	// Before close
	if s.CloseError() != nil {
		t.Error("CloseError() should be nil before close")
	}

	// Clean close
	s.Close()
	if s.CloseError() != nil {
		t.Error("CloseError() should be nil for clean close")
	}
}

func TestSession_DefaultQueueSize(t *testing.T) {
	s := newSession("id", nil, 0, nil) // 0 should use default

	// Fill with more than default
	for i := 0; i < defaultQueueSize; i++ {
		err := s.Send(Message{Topic: "test"})
		if err != nil {
			t.Errorf("Send() %d error = %v", i, err)
			break
		}
	}

	// Next one should fail
	err := s.Send(Message{Topic: "overflow"})
	if err != ErrQueueFull {
		t.Errorf("Send() after filling queue error = %v, want %v", err, ErrQueueFull)
	}
}

// -----------------------------------------------------------------------------
// PubSub Tests
// -----------------------------------------------------------------------------

func TestPubSub_Subscribe(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")

	ps.mu.RLock()
	count := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if count != 1 {
		t.Errorf("topic1 subscriber count = %d, want 1", count)
	}
}

func TestPubSub_Subscribe_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.subscribe(nil, "topic1") // Should not panic

	ps.mu.RLock()
	count := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if count != 0 {
		t.Error("Count should be 0 for nil session subscribe")
	}
}

func TestPubSub_Subscribe_EmptyTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "") // Should not panic

	// Session should have no topics
	s.mu.RLock()
	topicCount := len(s.topics)
	s.mu.RUnlock()
	if topicCount != 0 {
		t.Error("session should have no topics after empty topic subscribe")
	}
}

func TestPubSub_Subscribe_Multiple(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)

	ps.subscribe(s1, "topic1")
	ps.subscribe(s2, "topic1")

	ps.mu.RLock()
	count := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if count != 2 {
		t.Errorf("topic1 subscriber count = %d, want 2", count)
	}
}

func TestPubSub_Unsubscribe(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")
	ps.unsubscribe(s, "topic1")

	ps.mu.RLock()
	count := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if count != 0 {
		t.Errorf("topic1 subscriber count = %d, want 0", count)
	}
}

func TestPubSub_Unsubscribe_NonSubscribed(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	// Should not panic
	ps.unsubscribe(s, "topic1")
}

func TestPubSub_Unsubscribe_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.unsubscribe(nil, "topic1") // Should not panic
}

func TestPubSub_UnsubscribeAll(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")
	ps.subscribe(s, "topic2")
	ps.subscribe(s, "topic3")

	ps.unsubscribeAll(s)

	ps.mu.RLock()
	c1 := len(ps.topics["topic1"])
	c2 := len(ps.topics["topic2"])
	c3 := len(ps.topics["topic3"])
	ps.mu.RUnlock()

	if c1 != 0 || c2 != 0 || c3 != 0 {
		t.Error("all topics should have 0 subscribers")
	}
}

func TestPubSub_UnsubscribeAll_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.unsubscribeAll(nil) // Should not panic
}

func TestPubSub_Publish(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")

	data := []byte(`{"text":"hello"}`)
	ps.publish("topic1", data)

	select {
	case received := <-s.sendCh:
		if received.Topic != "topic1" {
			t.Errorf("received.Topic = %s, want topic1", received.Topic)
		}
		if !bytes.Equal(received.Data, data) {
			t.Errorf("received.Data = %s, want %s", received.Data, data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message in queue")
	}
}

func TestPubSub_Publish_MultipleSubscribers(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)
	s3 := newSession("s3", nil, 10, nil)

	ps.subscribe(s1, "topic1")
	ps.subscribe(s2, "topic1")
	ps.subscribe(s3, "topic1")

	data := []byte(`{"event":"test"}`)
	ps.publish("topic1", data)

	// All should receive
	for _, s := range []*Session{s1, s2, s3} {
		select {
		case <-s.sendCh:
			// Good
		case <-time.After(100 * time.Millisecond):
			t.Errorf("session %s did not receive message", s.ID())
		}
	}
}

func TestPubSub_Publish_EmptyTopic(t *testing.T) {
	ps := newMemPubSub()
	ps.publish("", []byte(`{}`)) // Should not panic
}

func TestPubSub_Publish_NoSubscribers(t *testing.T) {
	ps := newMemPubSub()
	ps.publish("topic1", []byte(`{}`)) // Should not panic
}

func TestPubSub_Concurrent(t *testing.T) {
	ps := newMemPubSub()
	sessions := make([]*Session, 100)
	for i := range sessions {
		sessions[i] = newSession(string(rune('a'+i)), nil, 100, nil)
	}

	var wg sync.WaitGroup

	// Concurrent subscribes
	for _, s := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			ps.subscribe(s, "topic1")
			ps.subscribe(s, "topic2")
		}(s)
	}

	wg.Wait()

	ps.mu.RLock()
	count := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if count != 100 {
		t.Errorf("topic1 subscriber count = %d, want 100", count)
	}

	// Concurrent publishes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.publish("topic1", []byte(`{}`))
		}(i)
	}

	wg.Wait()

	// Concurrent unsubscribes
	for _, s := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			ps.unsubscribeAll(s)
		}(s)
	}

	wg.Wait()

	ps.mu.RLock()
	countAfter := len(ps.topics["topic1"])
	ps.mu.RUnlock()

	if countAfter != 0 {
		t.Errorf("topic1 subscriber count = %d after unsubscribe all, want 0", countAfter)
	}
}

// -----------------------------------------------------------------------------
// Server Tests
// -----------------------------------------------------------------------------

func TestNew_Defaults(t *testing.T) {
	srv := New(Options{})

	if srv.opts.QueueSize != defaultQueueSize {
		t.Errorf("QueueSize = %d, want %d", srv.opts.QueueSize, defaultQueueSize)
	}
	if srv.opts.IDGenerator == nil {
		t.Error("IDGenerator should have default")
	}
}

func TestNew_CustomOptions(t *testing.T) {
	customIDGen := func() string { return "custom-id" }

	srv := New(Options{
		QueueSize:   100,
		IDGenerator: customIDGen,
	})

	if srv.opts.QueueSize != 100 {
		t.Errorf("QueueSize = %d, want 100", srv.opts.QueueSize)
	}
	if srv.opts.IDGenerator() != "custom-id" {
		t.Errorf("IDGenerator() = %s, want custom-id", srv.opts.IDGenerator())
	}
}

func TestServer_Handler(t *testing.T) {
	srv := New(Options{})
	handler := srv.Handler()

	if handler == nil {
		t.Error("Handler() should not return nil")
	}
}

func TestServer_Subscribe_Unsubscribe(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)

	srv.Subscribe(s, "topic1")

	srv.pubsub.mu.RLock()
	count := len(srv.pubsub.topics["topic1"])
	srv.pubsub.mu.RUnlock()

	if count != 1 {
		t.Errorf("topic1 subscriber count = %d, want 1", count)
	}

	srv.Unsubscribe(s, "topic1")

	srv.pubsub.mu.RLock()
	countAfter := len(srv.pubsub.topics["topic1"])
	srv.pubsub.mu.RUnlock()

	if countAfter != 0 {
		t.Errorf("topic1 subscriber count = %d, want 0", countAfter)
	}
}

func TestServer_Publish(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)
	srv.Subscribe(s, "topic1")

	data := []byte(`{"text":"hello"}`)
	srv.Publish("topic1", data)

	select {
	case received := <-s.sendCh:
		if received.Topic != "topic1" {
			t.Errorf("received.Topic = %s, want topic1", received.Topic)
		}
		if !bytes.Equal(received.Data, data) {
			t.Errorf("received.Data = %s, want %s", received.Data, data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message in session queue")
	}
}

func TestServer_addSession(t *testing.T) {
	srv := New(Options{})
	s := newSession("test-id", nil, 10, srv)

	srv.addSession(s)

	_, exists := srv.sessions.Load("test-id")
	if !exists {
		t.Error("addSession should register the session")
	}
}

func TestServer_removeSession(t *testing.T) {
	srv := New(Options{})
	s := newSession("test-id", nil, 10, srv)
	srv.addSession(s)

	// Subscribe to topics
	srv.Subscribe(s, "topic1")
	srv.Subscribe(s, "topic2")

	srv.removeSession(s)

	_, exists := srv.sessions.Load("test-id")
	if exists {
		t.Error("removeSession should unregister the session")
	}

	// Topics should be unsubscribed
	srv.pubsub.mu.RLock()
	count := len(srv.pubsub.topics["topic1"])
	srv.pubsub.mu.RUnlock()

	if count != 0 {
		t.Error("removeSession should unsubscribe from all topics")
	}
}

func TestServer_OnAuth(t *testing.T) {
	type UserInfo struct {
		ID string
	}

	authCalled := false
	srv := New(Options{
		OnAuth: func(ctx context.Context, r *http.Request) (any, error) {
			authCalled = true
			return UserInfo{ID: "test"}, nil
		},
	})

	if srv.opts.OnAuth == nil {
		t.Error("OnAuth should be set")
	}

	value, err := srv.opts.OnAuth(context.Background(), nil)
	if err != nil {
		t.Errorf("OnAuth error = %v", err)
	}
	if !authCalled {
		t.Error("OnAuth was not called")
	}
	user, ok := value.(UserInfo)
	if !ok {
		t.Error("OnAuth should return UserInfo")
	}
	if user.ID != "test" {
		t.Errorf("OnAuth user.ID = %v, want test", user.ID)
	}
}

func TestServer_OnMessage(t *testing.T) {
	var receivedTopic string
	var receivedData []byte
	var receivedSession *Session

	srv := New(Options{
		OnMessage: func(ctx context.Context, s *Session, topic string, data []byte) {
			receivedSession = s
			receivedTopic = topic
			receivedData = data
		},
	})

	s := newSession("test-id", nil, 10, srv)
	topic := "topic1"
	data := []byte(`{"foo":"bar"}`)

	srv.opts.OnMessage(context.Background(), s, topic, data)

	if receivedSession != s {
		t.Error("OnMessage session mismatch")
	}
	if receivedTopic != topic {
		t.Errorf("OnMessage topic = %s, want %s", receivedTopic, topic)
	}
	if !bytes.Equal(receivedData, data) {
		t.Errorf("OnMessage data = %s, want %s", receivedData, data)
	}
}

func TestServer_OnClose(t *testing.T) {
	var closedSession *Session
	var closeErr error

	srv := New(Options{
		OnClose: func(s *Session, err error) {
			closedSession = s
			closeErr = err
		},
	})

	s := newSession("test-id", nil, 10, srv)
	srv.opts.OnClose(s, ErrQueueFull)

	if closedSession != s {
		t.Error("OnClose session mismatch")
	}
	if closeErr != ErrQueueFull {
		t.Errorf("OnClose err = %v, want %v", closeErr, ErrQueueFull)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID() returned empty string")
	}
	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("generateID() length = %d, want 32", len(id1))
	}
	if id1 == id2 {
		t.Error("generateID() should generate unique IDs")
	}
}

// -----------------------------------------------------------------------------
// WebSocket Tests
// -----------------------------------------------------------------------------

func TestHandleConn_NotWebSocket(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleConn_ForbiddenOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"https://allowed.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Origin", "https://forbidden.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestHandleConn_AllowedOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"https://allowed.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Origin", "https://allowed.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	// Will fail to hijack in test, but shouldn't be forbidden
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden for allowed origin")
	}
}

func TestHandleConn_WildcardOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"*"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Origin", "https://any.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Error("wildcard should allow any origin")
	}
}

func TestHandleConn_MissingKey(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	// Missing Sec-WebSocket-Key
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleConn_AuthFailed(t *testing.T) {
	srv := New(Options{
		OnAuth: func(ctx context.Context, r *http.Request) (any, error) {
			return nil, ErrSessionClosed // Any error
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleConn_InvalidVersion(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "12") // Wrong version
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUpgradeRequired {
		t.Errorf("expected %d, got %d", http.StatusUpgradeRequired, rec.Code)
	}

	// Should include Sec-WebSocket-Version header
	if rec.Header().Get("Sec-WebSocket-Version") != "13" {
		t.Errorf("expected Sec-WebSocket-Version: 13 header")
	}
}

func TestHandleConn_MissingVersion(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	// Missing Sec-WebSocket-Version
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUpgradeRequired {
		t.Errorf("expected %d, got %d", http.StatusUpgradeRequired, rec.Code)
	}
}

func TestHandleConn_InvalidKey(t *testing.T) {
	srv := New(Options{})

	tests := []struct {
		name string
		key  string
	}{
		{"too short", "dG9vIHNob3J0"},                     // Decodes to less than 16 bytes
		{"invalid base64", "not-valid-base64!!!"},
		{"too long", "dGhlIHNhbXBsZSBub25jZSBleHRyYQ=="}, // 20 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Key", tt.key)
			req.Header.Set("Sec-WebSocket-Version", "13")
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected %d, got %d for key %q", http.StatusBadRequest, rec.Code, tt.key)
			}
		})
	}
}

func TestServer_CheckOrigin(t *testing.T) {
	checkOriginCalled := false
	srv := New(Options{
		CheckOrigin: func(r *http.Request) bool {
			checkOriginCalled = true
			return r.Header.Get("Origin") == "https://custom.com"
		},
	})

	// Should be allowed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Origin", "https://custom.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if !checkOriginCalled {
		t.Error("CheckOrigin callback was not called")
	}
	// Should not be forbidden
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden for allowed custom origin")
	}

	// Should be forbidden
	checkOriginCalled = false
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Upgrade", "websocket")
	req2.Header.Set("Connection", "Upgrade")
	req2.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req2.Header.Set("Sec-WebSocket-Version", "13")
	req2.Header.Set("Origin", "https://other.com")
	rec2 := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec2, req2)

	if !checkOriginCalled {
		t.Error("CheckOrigin callback was not called")
	}
	if rec2.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec2.Code)
	}
}

func TestNew_ReadLimitDefault(t *testing.T) {
	srv := New(Options{})

	if srv.opts.ReadLimit != defaultReadLimit {
		t.Errorf("ReadLimit = %d, want %d", srv.opts.ReadLimit, defaultReadLimit)
	}
}

func TestNew_CustomReadLimit(t *testing.T) {
	srv := New(Options{
		ReadLimit: 1024,
	})

	if srv.opts.ReadLimit != 1024 {
		t.Errorf("ReadLimit = %d, want 1024", srv.opts.ReadLimit)
	}
}

// mockConn is a mock net.Conn for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestWsConn_WriteMessage(t *testing.T) {
	mock := newMockConn()
	wsConn := &ws.Conn{}
	// We can't easily test the internal ws.Conn without exposing internals
	// Just test that it doesn't panic
	_ = wsConn
	_ = mock
}

// Integration test with a real HTTP test server
func TestServer_Integration(t *testing.T) {
	var mu sync.Mutex
	var receivedTopics []string
	var closedSessions []*Session

	// Create server pointer first so we can reference it in callbacks
	var srv *Server

	srv = New(Options{
		QueueSize: 10,
		OnAuth: func(ctx context.Context, r *http.Request) (any, error) {
			token := r.Header.Get("Authorization")
			if token == "" {
				return nil, ErrSessionClosed
			}
			return map[string]string{"token": token}, nil
		},
		OnMessage: func(ctx context.Context, s *Session, topic string, data []byte) {
			mu.Lock()
			receivedTopics = append(receivedTopics, topic)
			mu.Unlock()

			// Handle subscribe command in data
			var cmd struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &cmd) == nil && cmd.Type == "subscribe" {
				srv.Subscribe(s, topic)
			}
		},
		OnClose: func(s *Session, err error) {
			mu.Lock()
			closedSessions = append(closedSessions, s)
			mu.Unlock()
		},
	})

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Test 1: Auth required
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("HTTP GET error: %v", err)
	}
	resp.Body.Close()
	// Not a WebSocket request, should get bad request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("non-WS request status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Test 2: WebSocket upgrade without auth
	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	// No Authorization header

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated WS request status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Verify closedSessions is used
	_ = closedSessions
	_ = receivedTopics
}

// Test that server properly handles origin checking
func TestServer_Origins(t *testing.T) {
	tests := []struct {
		name          string
		serverOrigins []string
		requestOrigin string
		expectForbid  bool
	}{
		{
			name:          "no origins allows all",
			serverOrigins: nil,
			requestOrigin: "https://any.com",
			expectForbid:  false,
		},
		{
			name:          "wildcard allows all",
			serverOrigins: []string{"*"},
			requestOrigin: "https://any.com",
			expectForbid:  false,
		},
		{
			name:          "exact match allowed",
			serverOrigins: []string{"https://allowed.com"},
			requestOrigin: "https://allowed.com",
			expectForbid:  false,
		},
		{
			name:          "no match forbidden",
			serverOrigins: []string{"https://allowed.com"},
			requestOrigin: "https://other.com",
			expectForbid:  true,
		},
		{
			name:          "multiple origins allowed",
			serverOrigins: []string{"https://a.com", "https://b.com"},
			requestOrigin: "https://b.com",
			expectForbid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(Options{Origins: tt.serverOrigins})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Origin", tt.requestOrigin)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if tt.expectForbid && rec.Code != http.StatusForbidden {
				t.Errorf("expected forbidden, got %d", rec.Code)
			}
			if !tt.expectForbid && rec.Code == http.StatusForbidden {
				t.Error("unexpected forbidden")
			}
		})
	}
}

// Test error variables
func TestErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrSessionClosed, "live: session closed"},
		{ErrQueueFull, "live: send queue full"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("%v.Error() = %s, want %s", tt.err, tt.err.Error(), tt.want)
		}
	}
}

// Test sync notifier pattern (moved to user code / sync package)
func TestSyncNotifierPattern(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)
	srv.Subscribe(s, "sync:test-scope")

	// This is how sync package would implement a notifier
	notifyFunc := func(scope string, cursor uint64) {
		srv.Publish("sync:"+scope, []byte(`{"cursor":42}`))
	}

	notifyFunc("test-scope", 42)

	select {
	case msg := <-s.sendCh:
		if msg.Topic != "sync:test-scope" {
			t.Errorf("msg.Topic = %s, want sync:test-scope", msg.Topic)
		}
		if !strings.Contains(string(msg.Data), "42") {
			t.Errorf("msg.Data = %s, want to contain 42", msg.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected sync notification")
	}
}

// Test internal ws package integration
func TestWsPackageIntegration(t *testing.T) {
	// Test IsUpgradeRequest via ws package
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "valid upgrade",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			expected: true,
		},
		{
			name: "case insensitive",
			headers: map[string]string{
				"Upgrade":    "WebSocket",
				"Connection": "upgrade",
			},
			expected: true,
		},
		{
			name: "connection with keep-alive",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "keep-alive, Upgrade",
			},
			expected: true,
		},
		{
			name: "missing upgrade",
			headers: map[string]string{
				"Connection": "Upgrade",
			},
			expected: false,
		},
		{
			name: "missing connection",
			headers: map[string]string{
				"Upgrade": "websocket",
			},
			expected: false,
		},
		{
			name:     "no headers",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := ws.IsUpgradeRequest(req); got != tt.expected {
				t.Errorf("IsUpgradeRequest() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWsValidateKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"valid key", "dGhlIHNhbXBsZSBub25jZQ==", true}, // RFC example
		{"invalid base64", "not-valid-base64!!!", false},
		{"too short", "dG9vIHNob3J0", false},                     // < 16 bytes
		{"too long", "dGhlIHNhbXBsZSBub25jZSBleHRyYQ==", false}, // > 16 bytes
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ws.ValidateKey(tt.key); got != tt.valid {
				t.Errorf("ValidateKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

// Test wsConn via internal/ws package
func TestWsConnReadMessage(t *testing.T) {
	mock := newMockConn()

	// Create a masked text frame
	frame := []byte{
		0x81,                   // FIN + text opcode
		0x85,                   // masked + length 5
		0x37, 0xfa, 0x21, 0x3d, // mask key
		0x7f, 0x9f, 0x4d, 0x51, 0x58, // masked "Hello"
	}
	mock.readBuf.Write(frame)

	wsConn := &ws.Conn{}
	// Note: We can't easily create a valid ws.Conn without using Upgrade
	// This test just verifies the mock setup works
	_ = wsConn
	_ = bufio.NewReader(mock.readBuf)
}
