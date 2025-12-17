package live

import (
	"sync"
	"testing"
	"time"
)

func TestMemPubSub_Subscribe(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")

	if !s.hasTopic("topic1") {
		t.Error("session should have topic after Subscribe")
	}

	who := ps.Who("topic1")
	if len(who) != 1 || who[0] != s {
		t.Errorf("Who(topic1) = %v, want [%v]", who, s)
	}
}

func TestMemPubSub_Subscribe_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.Subscribe(nil, "topic1") // Should not panic

	if ps.Count("topic1") != 0 {
		t.Error("Count should be 0 for nil session subscribe")
	}
}

func TestMemPubSub_Subscribe_EmptyTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "") // Should not panic

	if len(s.Topics()) != 0 {
		t.Error("session should have no topics after empty topic subscribe")
	}
}

func TestMemPubSub_Subscribe_Multiple(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)

	ps.Subscribe(s1, "topic1")
	ps.Subscribe(s2, "topic1")

	if ps.Count("topic1") != 2 {
		t.Errorf("Count(topic1) = %d, want 2", ps.Count("topic1"))
	}
}

func TestMemPubSub_Unsubscribe(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")
	ps.Unsubscribe(s, "topic1")

	if s.hasTopic("topic1") {
		t.Error("session should not have topic after Unsubscribe")
	}

	if ps.Count("topic1") != 0 {
		t.Errorf("Count(topic1) = %d, want 0", ps.Count("topic1"))
	}
}

func TestMemPubSub_Unsubscribe_NonSubscribed(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	// Should not panic
	ps.Unsubscribe(s, "topic1")
}

func TestMemPubSub_Unsubscribe_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.Unsubscribe(nil, "topic1") // Should not panic
}

func TestMemPubSub_Unsubscribe_CleanupEmptyTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")
	ps.Unsubscribe(s, "topic1")

	topics := ps.Topics()
	for _, topic := range topics {
		if topic == "topic1" {
			t.Error("empty topic should be removed")
		}
	}
}

func TestMemPubSub_UnsubscribeAll(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")
	ps.Subscribe(s, "topic2")
	ps.Subscribe(s, "topic3")

	ps.UnsubscribeAll(s)

	if len(s.Topics()) != 0 {
		t.Errorf("session should have no topics after UnsubscribeAll")
	}

	if ps.Count("topic1") != 0 || ps.Count("topic2") != 0 || ps.Count("topic3") != 0 {
		t.Error("all topics should have 0 subscribers")
	}
}

func TestMemPubSub_UnsubscribeAll_NilSession(t *testing.T) {
	ps := newMemPubSub()
	ps.UnsubscribeAll(nil) // Should not panic
}

func TestMemPubSub_Publish(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")

	msg := Message{Type: "test", Body: []byte("hello")}
	ps.Publish("topic1", msg)

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

func TestMemPubSub_Publish_SetsTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")

	// Publish without topic in message
	msg := Message{Type: "test"}
	ps.Publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Topic != "topic1" {
			t.Errorf("received.Topic = %s, want topic1 (auto-set)", received.Topic)
		}
	default:
		t.Error("expected message in queue")
	}
}

func TestMemPubSub_Publish_PreservesTopic(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")

	// Publish with topic already set
	msg := Message{Type: "test", Topic: "original"}
	ps.Publish("topic1", msg)

	select {
	case received := <-s.sendCh:
		if received.Topic != "original" {
			t.Errorf("received.Topic = %s, want original (preserved)", received.Topic)
		}
	default:
		t.Error("expected message in queue")
	}
}

func TestMemPubSub_Publish_MultipleSubscribers(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)
	s3 := newSession("s3", nil, 10, nil)

	ps.Subscribe(s1, "topic1")
	ps.Subscribe(s2, "topic1")
	ps.Subscribe(s3, "topic1")

	msg := Message{Type: "test"}
	ps.Publish("topic1", msg)

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

func TestMemPubSub_Publish_EmptyTopic(t *testing.T) {
	ps := newMemPubSub()
	ps.Publish("", Message{Type: "test"}) // Should not panic
}

func TestMemPubSub_Publish_NoSubscribers(t *testing.T) {
	ps := newMemPubSub()
	ps.Publish("topic1", Message{Type: "test"}) // Should not panic
}

func TestMemPubSub_Who(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)

	ps.Subscribe(s1, "topic1")
	ps.Subscribe(s2, "topic1")

	who := ps.Who("topic1")
	if len(who) != 2 {
		t.Errorf("Who(topic1) has %d sessions, want 2", len(who))
	}
}

func TestMemPubSub_Who_Empty(t *testing.T) {
	ps := newMemPubSub()

	who := ps.Who("nonexistent")
	if who != nil && len(who) != 0 {
		t.Errorf("Who(nonexistent) = %v, want nil or empty", who)
	}
}

func TestMemPubSub_Topics(t *testing.T) {
	ps := newMemPubSub()
	s := newSession("s1", nil, 10, nil)

	ps.Subscribe(s, "topic1")
	ps.Subscribe(s, "topic2")
	ps.Subscribe(s, "topic3")

	topics := ps.Topics()
	if len(topics) != 3 {
		t.Errorf("Topics() has %d topics, want 3", len(topics))
	}
}

func TestMemPubSub_Topics_Empty(t *testing.T) {
	ps := newMemPubSub()

	topics := ps.Topics()
	if len(topics) != 0 {
		t.Errorf("Topics() = %v, want empty", topics)
	}
}

func TestMemPubSub_Count(t *testing.T) {
	ps := newMemPubSub()
	s1 := newSession("s1", nil, 10, nil)
	s2 := newSession("s2", nil, 10, nil)

	if ps.Count("topic1") != 0 {
		t.Errorf("Count(topic1) = %d before subscribing, want 0", ps.Count("topic1"))
	}

	ps.Subscribe(s1, "topic1")
	if ps.Count("topic1") != 1 {
		t.Errorf("Count(topic1) = %d, want 1", ps.Count("topic1"))
	}

	ps.Subscribe(s2, "topic1")
	if ps.Count("topic1") != 2 {
		t.Errorf("Count(topic1) = %d, want 2", ps.Count("topic1"))
	}
}

func TestMemPubSub_Concurrent(t *testing.T) {
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
			ps.Subscribe(s, "topic1")
			ps.Subscribe(s, "topic2")
		}(s)
	}

	wg.Wait()

	if ps.Count("topic1") != 100 {
		t.Errorf("Count(topic1) = %d, want 100", ps.Count("topic1"))
	}

	// Concurrent publishes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.Publish("topic1", Message{Type: "test"})
		}(i)
	}

	wg.Wait()

	// Concurrent unsubscribes
	for _, s := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			ps.UnsubscribeAll(s)
		}(s)
	}

	wg.Wait()

	if ps.Count("topic1") != 0 {
		t.Errorf("Count(topic1) = %d after unsubscribe all, want 0", ps.Count("topic1"))
	}
}
