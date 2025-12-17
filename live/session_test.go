package live

import (
	"testing"
	"time"
)

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

func TestSession_Done(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	select {
	case <-s.Done():
		t.Error("Done() should not be closed for open session")
	default:
		// Expected
	}

	s.Close()

	select {
	case <-s.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Done() should be closed after Close()")
	}
}

func TestSession_CloseError(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	if s.CloseError() != nil {
		t.Error("CloseError() should be nil for open session")
	}

	s.closeWithError(ErrQueueFull)

	if s.CloseError() != ErrQueueFull {
		t.Errorf("CloseError() = %v, want %v", s.CloseError(), ErrQueueFull)
	}
}

func TestSession_Topics(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	if len(s.Topics()) != 0 {
		t.Error("new session should have no topics")
	}

	s.addTopic("topic1")
	s.addTopic("topic2")

	topics := s.Topics()
	if len(topics) != 2 {
		t.Errorf("Topics() length = %d, want 2", len(topics))
	}

	// Check topics are present (order not guaranteed)
	topicSet := make(map[string]bool)
	for _, t := range topics {
		topicSet[t] = true
	}
	if !topicSet["topic1"] || !topicSet["topic2"] {
		t.Errorf("Topics() = %v, want [topic1, topic2]", topics)
	}
}

func TestSession_AddTopic(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	s.addTopic("topic1")
	if !s.hasTopic("topic1") {
		t.Error("hasTopic(topic1) = false after addTopic")
	}
}

func TestSession_RemoveTopic(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	s.addTopic("topic1")
	s.removeTopic("topic1")

	if s.hasTopic("topic1") {
		t.Error("hasTopic(topic1) = true after removeTopic")
	}
}

func TestSession_ClearTopics(t *testing.T) {
	s := newSession("id", nil, 10, nil)

	s.addTopic("topic1")
	s.addTopic("topic2")
	s.addTopic("topic3")

	cleared := s.clearTopics()
	if len(cleared) != 3 {
		t.Errorf("clearTopics() returned %d topics, want 3", len(cleared))
	}

	if len(s.Topics()) != 0 {
		t.Error("Topics() should be empty after clearTopics()")
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
