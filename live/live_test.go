package live

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
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
				Type:  "message",
				Topic: "room:1",
				Ref:   "123",
				Body:  []byte("hello"),
			},
			want: `{"type":"message","topic":"room:1","ref":"123","body":"aGVsbG8="}`,
		},
		{
			name: "type only",
			msg: Message{
				Type: "ping",
			},
			want: `{"type":"ping"}`,
		},
		{
			name: "with empty topic",
			msg: Message{
				Type: "ack",
				Ref:  "abc",
			},
			want: `{"type":"ack","ref":"abc"}`,
		},
		{
			name: "nil body",
			msg: Message{
				Type:  "subscribe",
				Topic: "room:2",
			},
			want: `{"type":"subscribe","topic":"room:2"}`,
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
		name    string
		data    string
		want    Message
		wantErr bool
	}{
		{
			name: "basic message",
			data: `{"type":"message","topic":"room:1","ref":"123","body":"aGVsbG8="}`,
			want: Message{
				Type:  "message",
				Topic: "room:1",
				Ref:   "123",
				Body:  []byte("hello"),
			},
		},
		{
			name: "type only",
			data: `{"type":"ping"}`,
			want: Message{
				Type: "ping",
			},
		},
		{
			name:    "invalid json",
			data:    `{invalid`,
			wantErr: true,
		},
		{
			name: "empty json",
			data: `{}`,
			want: Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeMessage([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %s, want %s", got.Type, tt.want.Type)
			}
			if got.Topic != tt.want.Topic {
				t.Errorf("Topic = %s, want %s", got.Topic, tt.want.Topic)
			}
			if got.Ref != tt.want.Ref {
				t.Errorf("Ref = %s, want %s", got.Ref, tt.want.Ref)
			}
			if !bytes.Equal(got.Body, tt.want.Body) {
				t.Errorf("Body = %v, want %v", got.Body, tt.want.Body)
			}
		})
	}
}

func TestMessageRoundTrip(t *testing.T) {
	original := Message{
		Type:  "message",
		Topic: "chat:room:123",
		Ref:   "ref-456",
		Body:  []byte(`{"text":"Hello, World!"}`),
	}

	encoded, err := encodeMessage(original)
	if err != nil {
		t.Fatalf("encodeMessage() error = %v", err)
	}

	decoded, err := decodeMessage(encoded)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.Topic != original.Topic {
		t.Errorf("Topic mismatch: got %s, want %s", decoded.Topic, original.Topic)
	}
	if decoded.Ref != original.Ref {
		t.Errorf("Ref mismatch: got %s, want %s", decoded.Ref, original.Ref)
	}
	if !bytes.Equal(decoded.Body, original.Body) {
		t.Errorf("Body mismatch: got %s, want %s", decoded.Body, original.Body)
	}
}

// -----------------------------------------------------------------------------
// Meta Tests
// -----------------------------------------------------------------------------

func TestMeta_Get(t *testing.T) {
	meta := Meta{"key": "value", "num": 42}

	if meta.Get("key") != "value" {
		t.Errorf("Get(key) = %v, want value", meta.Get("key"))
	}
	if meta.Get("num") != 42 {
		t.Errorf("Get(num) = %v, want 42", meta.Get("num"))
	}
	if meta.Get("missing") != nil {
		t.Errorf("Get(missing) = %v, want nil", meta.Get("missing"))
	}
}

func TestMeta_Get_Nil(t *testing.T) {
	var meta Meta

	if meta.Get("key") != nil {
		t.Errorf("nil Meta.Get(key) = %v, want nil", meta.Get("key"))
	}
}

func TestMeta_GetString(t *testing.T) {
	meta := Meta{"str": "value", "num": 42}

	if meta.GetString("str") != "value" {
		t.Errorf("GetString(str) = %s, want value", meta.GetString("str"))
	}
	if meta.GetString("num") != "" {
		t.Errorf("GetString(num) = %s, want empty string", meta.GetString("num"))
	}
	if meta.GetString("missing") != "" {
		t.Errorf("GetString(missing) = %s, want empty string", meta.GetString("missing"))
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

func TestSession_Meta(t *testing.T) {
	meta := Meta{"user_id": "123", "role": "admin"}
	s := newSession("id", meta, 10, nil)

	if s.Meta().GetString("user_id") != "123" {
		t.Errorf("Meta().user_id = %v, want 123", s.Meta()["user_id"])
	}
	if s.Meta().GetString("role") != "admin" {
		t.Errorf("Meta().role = %v, want admin", s.Meta()["role"])
	}
}

func TestSession_Meta_Nil(t *testing.T) {
	s := newSession("id", nil, 10, nil)
	if s.Meta() != nil {
		t.Errorf("Meta() should be nil")
	}
}

func TestSession_Send_Success(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	msg := Message{Type: "test", Body: []byte("hello")}
	err := s.Send(msg)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Verify message is in queue
	select {
	case received := <-s.sendCh:
		if received.Type != msg.Type {
			t.Errorf("received.Type = %s, want %s", received.Type, msg.Type)
		}
	default:
		t.Error("expected message in queue")
	}
}

func TestSession_Send_Closed(t *testing.T) {
	s := newSession("id", nil, 10, nil)
	s.Close()

	err := s.Send(Message{Type: "test"})
	if err != ErrSessionClosed {
		t.Errorf("Send() on closed session error = %v, want %v", err, ErrSessionClosed)
	}
}

func TestSession_Send_QueueFull(t *testing.T) {
	s := newSession("id", nil, 2, nil)

	// Fill the queue
	s.Send(Message{Type: "1"})
	s.Send(Message{Type: "2"})

	// This should fail and close the session
	err := s.Send(Message{Type: "3"})
	if err != ErrQueueFull {
		t.Errorf("Send() with full queue error = %v, want %v", err, ErrQueueFull)
	}

	if !s.IsClosed() {
		t.Error("session should be closed after queue full")
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

func TestSession_DefaultQueueSize(t *testing.T) {
	s := newSession("id", nil, 0, nil) // 0 should use default

	// Fill with more than default
	for i := 0; i < defaultQueueSize; i++ {
		err := s.Send(Message{Type: "test"})
		if err != nil {
			t.Errorf("Send() %d error = %v", i, err)
			break
		}
	}

	// Next one should fail
	err := s.Send(Message{Type: "overflow"})
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

	if ps.count("topic1") != 1 {
		t.Errorf("count(topic1) = %d, want 1", ps.count("topic1"))
	}
}

func TestPubSub_Subscribe_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.subscribe(nil, "topic1") // Should not panic

	if ps.count("topic1") != 0 {
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

	if ps.count("topic1") != 2 {
		t.Errorf("count(topic1) = %d, want 2", ps.count("topic1"))
	}
}

func TestPubSub_Unsubscribe(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")
	ps.unsubscribe(s, "topic1")

	if ps.count("topic1") != 0 {
		t.Errorf("count(topic1) = %d, want 0", ps.count("topic1"))
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

	if ps.count("topic1") != 0 || ps.count("topic2") != 0 || ps.count("topic3") != 0 {
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

	msg := Message{Type: "test", Body: []byte("hello")}
	ps.publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Type != msg.Type {
			t.Errorf("received.Type = %s, want %s", received.Type, msg.Type)
		}
		if received.Topic != "topic1" {
			t.Errorf("received.Topic = %s, want topic1", received.Topic)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message in queue")
	}
}

func TestPubSub_Publish_SetsTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")

	// Publish without topic in message
	msg := Message{Type: "test"}
	ps.publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Topic != "topic1" {
			t.Errorf("received.Topic = %s, want topic1 (auto-set)", received.Topic)
		}
	default:
		t.Error("expected message in queue")
	}
}

func TestPubSub_Publish_PreservesTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.subscribe(s, "topic1")

	// Publish with topic already set
	msg := Message{Type: "test", Topic: "original"}
	ps.publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Topic != "original" {
			t.Errorf("received.Topic = %s, want original (preserved)", received.Topic)
		}
	default:
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

	msg := Message{Type: "test"}
	ps.publish("topic1", msg)

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
	ps.publish("", Message{Type: "test"}) // Should not panic
}

func TestPubSub_Publish_NoSubscribers(t *testing.T) {
	ps := newMemPubSub()
	ps.publish("topic1", Message{Type: "test"}) // Should not panic
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

	if ps.count("topic1") != 100 {
		t.Errorf("count(topic1) = %d, want 100", ps.count("topic1"))
	}

	// Concurrent publishes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.publish("topic1", Message{Type: "test"})
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

	if ps.count("topic1") != 0 {
		t.Errorf("count(topic1) = %d after unsubscribe all, want 0", ps.count("topic1"))
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

func TestServer_SessionCount(t *testing.T) {
	srv := New(Options{})

	if srv.SessionCount() != 0 {
		t.Errorf("SessionCount() = %d, want 0", srv.SessionCount())
	}

	s1 := newSession("s1", nil, 10, srv)
	s2 := newSession("s2", nil, 10, srv)
	srv.addSession(s1)
	srv.addSession(s2)

	if srv.SessionCount() != 2 {
		t.Errorf("SessionCount() = %d, want 2", srv.SessionCount())
	}
}

func TestServer_Subscribe_Unsubscribe(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)

	srv.Subscribe(s, "topic1")
	if srv.pubsub.count("topic1") != 1 {
		t.Errorf("count(topic1) = %d, want 1", srv.pubsub.count("topic1"))
	}

	srv.Unsubscribe(s, "topic1")
	if srv.pubsub.count("topic1") != 0 {
		t.Errorf("count(topic1) = %d, want 0", srv.pubsub.count("topic1"))
	}
}

func TestServer_Publish(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)
	srv.Subscribe(s, "topic1")

	msg := Message{Type: "test", Body: []byte("hello")}
	srv.Publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Type != msg.Type {
			t.Errorf("received.Type = %s, want %s", received.Type, msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected message in session queue")
	}
}

func TestServer_Broadcast(t *testing.T) {
	srv := New(Options{})
	s1 := newSession("s1", nil, 10, srv)
	s2 := newSession("s2", nil, 10, srv)
	s3 := newSession("s3", nil, 10, srv)
	srv.addSession(s1)
	srv.addSession(s2)
	srv.addSession(s3)

	msg := Message{Type: "broadcast", Body: []byte("hello all")}
	srv.Broadcast(msg)

	for _, s := range []*Session{s1, s2, s3} {
		select {
		case received := <-s.sendCh:
			if received.Type != msg.Type {
				t.Errorf("session %s received.Type = %s, want %s", s.ID(), received.Type, msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("session %s did not receive broadcast", s.ID())
		}
	}
}

func TestServer_addSession(t *testing.T) {
	srv := New(Options{})
	s := newSession("test-id", nil, 10, srv)

	srv.addSession(s)

	if srv.SessionCount() != 1 {
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

	if srv.SessionCount() != 0 {
		t.Error("removeSession should unregister the session")
	}

	// Topics should be unsubscribed
	if srv.pubsub.count("topic1") != 0 {
		t.Error("removeSession should unsubscribe from all topics")
	}
}

func TestServer_OnAuth(t *testing.T) {
	authCalled := false
	srv := New(Options{
		OnAuth: func(ctx context.Context, r *http.Request) (Meta, error) {
			authCalled = true
			return Meta{"user": "test"}, nil
		},
	})

	if srv.opts.OnAuth == nil {
		t.Error("OnAuth should be set")
	}

	meta, err := srv.opts.OnAuth(context.Background(), nil)
	if err != nil {
		t.Errorf("OnAuth error = %v", err)
	}
	if !authCalled {
		t.Error("OnAuth was not called")
	}
	if meta.GetString("user") != "test" {
		t.Errorf("OnAuth meta = %v, want user=test", meta)
	}
}

func TestServer_OnMessage(t *testing.T) {
	var receivedMsg Message
	var receivedSession *Session

	srv := New(Options{
		OnMessage: func(ctx context.Context, s *Session, msg Message) {
			receivedSession = s
			receivedMsg = msg
		},
	})

	s := newSession("test-id", nil, 10, srv)
	msg := Message{Type: "test", Topic: "topic1"}

	srv.opts.OnMessage(context.Background(), s, msg)

	if receivedSession != s {
		t.Error("OnMessage session mismatch")
	}
	if receivedMsg.Type != msg.Type {
		t.Errorf("OnMessage msg.Type = %s, want %s", receivedMsg.Type, msg.Type)
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

func TestIsWebSocketUpgrade(t *testing.T) {
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

			if got := isWebSocketUpgrade(req); got != tt.expected {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestComputeAcceptKey(t *testing.T) {
	// Test case from RFC 6455
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	result := computeAcceptKey(key)
	if result != expected {
		t.Errorf("computeAcceptKey(%q) = %q, want %q", key, result, expected)
	}
}

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
	// Missing Sec-WebSocket-Key
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleConn_AuthFailed(t *testing.T) {
	srv := New(Options{
		OnAuth: func(ctx context.Context, r *http.Request) (Meta, error) {
			return nil, ErrAuthFailed
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
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
	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := ws.writeMessage(wsTextMessage, []byte("hello"))
	if err != nil {
		t.Errorf("writeMessage error: %v", err)
	}

	if mock.writeBuf.Len() == 0 {
		t.Error("expected data to be written")
	}
}

func TestWsConn_WriteMessageLengths(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 10},
		{"medium 126", 126},
		{"large 127", 200},
		{"very large", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn()
			ws := &wsConn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			data := make([]byte, tt.length)
			err := ws.writeMessage(wsBinaryMessage, data)
			if err != nil {
				t.Errorf("writeMessage error: %v", err)
			}

			if mock.writeBuf.Len() == 0 {
				t.Error("expected data to be written")
			}
		})
	}
}

func TestWsConn_ReadMessage(t *testing.T) {
	tests := []struct {
		name     string
		frame    []byte
		wantType int
		wantData []byte
		wantErr  bool
	}{
		{
			name: "short unmasked text",
			frame: []byte{
				0x81, // FIN + text opcode
				0x05, // length 5
				'h', 'e', 'l', 'l', 'o',
			},
			wantType: wsTextMessage,
			wantData: []byte("hello"),
		},
		{
			name: "masked text",
			frame: []byte{
				0x81,                   // FIN + text opcode
				0x85,                   // masked + length 5
				0x37, 0xfa, 0x21, 0x3d, // mask key
				0x7f, 0x9f, 0x4d, 0x51, 0x58, // masked "Hello"
			},
			wantType: wsTextMessage,
			wantData: []byte("Hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn()
			mock.readBuf.Write(tt.frame)

			ws := &wsConn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			msgType, data, err := ws.readMessage()
			if (err != nil) != tt.wantErr {
				t.Errorf("readMessage error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if msgType != tt.wantType {
					t.Errorf("got type %d, want %d", msgType, tt.wantType)
				}
				if !bytes.Equal(data, tt.wantData) {
					t.Errorf("got data %q, want %q", data, tt.wantData)
				}
			}
		})
	}
}

func TestWsConn_ReadMessageExtendedLength(t *testing.T) {
	mock := newMockConn()
	data := make([]byte, 200)
	for i := range data {
		data[i] = 'a'
	}
	frame := []byte{
		0x82,       // FIN + binary opcode
		0x7e,       // length 126 indicator
		0x00, 0xc8, // 200 in big endian
	}
	frame = append(frame, data...)
	mock.readBuf.Write(frame)

	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	msgType, received, err := ws.readMessage()
	if err != nil {
		t.Errorf("readMessage error: %v", err)
		return
	}
	if msgType != wsBinaryMessage {
		t.Errorf("got type %d, want %d", msgType, wsBinaryMessage)
	}
	if len(received) != 200 {
		t.Errorf("got length %d, want 200", len(received))
	}
}

func TestWsConn_ReadMessageErrors(t *testing.T) {
	// Test read error on first byte
	mock := newMockConn()
	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	_, _, err := ws.readMessage()
	if err == nil {
		t.Error("expected error on empty read")
	}

	// Test read error on second byte
	mock2 := newMockConn()
	mock2.readBuf.Write([]byte{0x81}) // Only first byte
	ws2 := &wsConn{
		conn:   mock2,
		reader: bufio.NewReader(mock2.readBuf),
		writer: bufio.NewWriter(mock2.writeBuf),
	}

	_, _, err = ws2.readMessage()
	if err == nil {
		t.Error("expected error on partial frame")
	}
}

func TestMessageConstants(t *testing.T) {
	if wsTextMessage != 1 {
		t.Errorf("wsTextMessage = %d, want 1", wsTextMessage)
	}
	if wsBinaryMessage != 2 {
		t.Errorf("wsBinaryMessage = %d, want 2", wsBinaryMessage)
	}
	if wsCloseMessage != 8 {
		t.Errorf("wsCloseMessage = %d, want 8", wsCloseMessage)
	}
	if wsPingMessage != 9 {
		t.Errorf("wsPingMessage = %d, want 9", wsPingMessage)
	}
	if wsPongMessage != 10 {
		t.Errorf("wsPongMessage = %d, want 10", wsPongMessage)
	}
}

// Integration test with a real HTTP test server
func TestServer_Integration(t *testing.T) {
	var mu sync.Mutex
	var receivedMessages []Message
	var closedSessions []*Session

	// Create server pointer first so we can reference it in callbacks
	var srv *Server

	srv = New(Options{
		QueueSize: 10,
		OnAuth: func(ctx context.Context, r *http.Request) (Meta, error) {
			token := r.Header.Get("Authorization")
			if token == "" {
				return nil, ErrAuthFailed
			}
			return Meta{"token": token}, nil
		},
		OnMessage: func(ctx context.Context, s *Session, msg Message) {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()

			// Handle subscribe
			if msg.Type == "subscribe" {
				srv.Subscribe(s, msg.Topic)
				_ = s.Send(Message{Type: "ack", Topic: msg.Topic, Ref: msg.Ref})
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
	_ = receivedMessages
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
		{ErrAuthFailed, "live: authentication failed"},
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
		srv.Publish("sync:"+scope, Message{
			Type:  "sync",
			Topic: "sync:" + scope,
			Body:  []byte(`{"cursor":42}`),
		})
	}

	notifyFunc("test-scope", 42)

	select {
	case msg := <-s.sendCh:
		if msg.Type != "sync" {
			t.Errorf("msg.Type = %s, want sync", msg.Type)
		}
		if msg.Topic != "sync:test-scope" {
			t.Errorf("msg.Topic = %s, want sync:test-scope", msg.Topic)
		}
		if !strings.Contains(string(msg.Body), "42") {
			t.Errorf("msg.Body = %s, want to contain 42", msg.Body)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected sync notification")
	}
}
