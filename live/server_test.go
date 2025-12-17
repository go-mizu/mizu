package live

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	srv := New(Options{})

	opts := srv.Options()
	if opts.Codec == nil {
		t.Error("Codec should default to JSONCodec")
	}
	if opts.QueueSize != defaultQueueSize {
		t.Errorf("QueueSize = %d, want %d", opts.QueueSize, defaultQueueSize)
	}
	if opts.IDGenerator == nil {
		t.Error("IDGenerator should have default")
	}
}

func TestNew_CustomOptions(t *testing.T) {
	customCodec := JSONCodec{}
	customIDGen := func() string { return "custom-id" }

	srv := New(Options{
		Codec:       customCodec,
		QueueSize:   100,
		IDGenerator: customIDGen,
	})

	opts := srv.Options()
	if opts.QueueSize != 100 {
		t.Errorf("QueueSize = %d, want 100", opts.QueueSize)
	}
	if opts.IDGenerator() != "custom-id" {
		t.Errorf("IDGenerator() = %s, want custom-id", opts.IDGenerator())
	}
}

func TestServer_Handler(t *testing.T) {
	srv := New(Options{})
	handler := srv.Handler()

	if handler == nil {
		t.Error("Handler() should not return nil")
	}
}

func TestServer_PubSub(t *testing.T) {
	srv := New(Options{})
	ps := srv.PubSub()

	if ps == nil {
		t.Error("PubSub() should not return nil")
	}
}

func TestServer_Session(t *testing.T) {
	srv := New(Options{})

	// No session initially
	if srv.Session("nonexistent") != nil {
		t.Error("Session(nonexistent) should return nil")
	}

	// Add a session
	s := newSession("test-id", nil, 10, srv)
	srv.addSession(s)

	if srv.Session("test-id") != s {
		t.Error("Session(test-id) should return the session")
	}

	// Remove session
	srv.removeSession(s)
	if srv.Session("test-id") != nil {
		t.Error("Session(test-id) should return nil after removal")
	}
}

func TestServer_Sessions(t *testing.T) {
	srv := New(Options{})

	if len(srv.Sessions()) != 0 {
		t.Error("Sessions() should be empty initially")
	}

	s1 := newSession("s1", nil, 10, srv)
	s2 := newSession("s2", nil, 10, srv)
	srv.addSession(s1)
	srv.addSession(s2)

	sessions := srv.Sessions()
	if len(sessions) != 2 {
		t.Errorf("Sessions() has %d sessions, want 2", len(sessions))
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

func TestServer_Publish(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)
	srv.PubSub().Subscribe(s, "topic1")

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

	if srv.Session("test-id") == nil {
		t.Error("addSession should register the session")
	}
}

func TestServer_removeSession(t *testing.T) {
	srv := New(Options{})
	s := newSession("test-id", nil, 10, srv)
	srv.addSession(s)

	// Subscribe to topics
	srv.PubSub().Subscribe(s, "topic1")
	srv.PubSub().Subscribe(s, "topic2")

	srv.removeSession(s)

	if srv.Session("test-id") != nil {
		t.Error("removeSession should unregister the session")
	}

	// Topics should be unsubscribed
	if srv.PubSub().Count("topic1") != 0 {
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

func TestUintToString(t *testing.T) {
	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{18446744073709551615, "18446744073709551615"}, // max uint64
	}

	for _, tt := range tests {
		got := uintToString(tt.n)
		if got != tt.want {
			t.Errorf("uintToString(%d) = %s, want %s", tt.n, got, tt.want)
		}
	}
}
