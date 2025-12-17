package live

import (
	"sync"
	"testing"
	"time"
)

func TestInmemPubSub(t *testing.T) {
	t.Run("subscribe and publish", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch := make(chan any, 10)

		ps.register("session1", ch)
		ps.Subscribe("session1", "topic1")

		ps.Publish("topic1", "hello")

		select {
		case msg := <-ch:
			if msg != "hello" {
				t.Errorf("expected hello, got %v", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout waiting for message")
		}
	})

	t.Run("unsubscribe stops messages", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch := make(chan any, 10)

		ps.register("session1", ch)
		ps.Subscribe("session1", "topic1")
		ps.Unsubscribe("session1", "topic1")

		ps.Publish("topic1", "hello")

		select {
		case <-ch:
			t.Error("should not receive message after unsubscribe")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})

	t.Run("unsubscribe all", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch := make(chan any, 10)

		ps.register("session1", ch)
		ps.Subscribe("session1", "topic1", "topic2", "topic3")
		ps.UnsubscribeAll("session1")

		ps.Publish("topic1", "hello")
		ps.Publish("topic2", "hello")
		ps.Publish("topic3", "hello")

		select {
		case <-ch:
			t.Error("should not receive any messages after unsubscribe all")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})

	t.Run("broadcast to all sessions", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch1 := make(chan any, 10)
		ch2 := make(chan any, 10)

		ps.register("session1", ch1)
		ps.register("session2", ch2)

		ps.Broadcast("system message")

		for _, ch := range []chan any{ch1, ch2} {
			select {
			case msg := <-ch:
				if msg != "system message" {
					t.Errorf("expected system message, got %v", msg)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("timeout waiting for broadcast")
			}
		}
	})

	t.Run("multiple subscribers", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch1 := make(chan any, 10)
		ch2 := make(chan any, 10)

		ps.register("session1", ch1)
		ps.register("session2", ch2)
		ps.Subscribe("session1", "shared")
		ps.Subscribe("session2", "shared")

		ps.Publish("shared", "hello all")

		for _, ch := range []chan any{ch1, ch2} {
			select {
			case msg := <-ch:
				if msg != "hello all" {
					t.Errorf("expected hello all, got %v", msg)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("timeout waiting for message")
			}
		}
	})

	t.Run("unregister removes session", func(t *testing.T) {
		ps := NewInmemPubSub()
		ch := make(chan any, 10)

		ps.register("session1", ch)
		ps.Subscribe("session1", "topic1")

		// Should be subscribed
		if ps.TopicSubscribers("topic1") != 1 {
			t.Error("expected 1 subscriber")
		}

		ps.unregister("session1")

		// Publish should not crash
		ps.Publish("topic1", "hello")

		// Channel should be empty since session is unregistered
		select {
		case <-ch:
			t.Error("should not receive after unregister")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})

	t.Run("session count", func(t *testing.T) {
		ps := NewInmemPubSub()

		ps.register("s1", make(chan any, 1))
		ps.register("s2", make(chan any, 1))
		ps.register("s3", make(chan any, 1))

		if ps.SessionCount() != 3 {
			t.Errorf("expected 3 sessions, got %d", ps.SessionCount())
		}

		ps.unregister("s2")

		if ps.SessionCount() != 2 {
			t.Errorf("expected 2 sessions, got %d", ps.SessionCount())
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		ps := NewInmemPubSub()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				ch := make(chan any, 10)
				sid := string(rune('a' + id%26))

				ps.register(sid, ch)
				ps.Subscribe(sid, "topic")
				ps.Publish("topic", "msg")
				ps.Unsubscribe(sid, "topic")
				ps.unregister(sid)
			}(i)
		}

		wg.Wait()
	})
}
