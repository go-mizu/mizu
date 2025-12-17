package live

import (
	"sync"
	"testing"
	"time"
)

func TestPoke_Fields(t *testing.T) {
	p := Poke{
		Scope:  "user:123",
		Cursor: 42,
	}

	if p.Scope != "user:123" {
		t.Errorf("expected scope user:123, got %s", p.Scope)
	}
	if p.Cursor != 42 {
		t.Errorf("expected cursor 42, got %d", p.Cursor)
	}
}

func TestSyncPokeBroker_Poke(t *testing.T) {
	pubsub := NewInmemPubSub()
	broker := NewSyncPokeBroker(pubsub)

	// Create a session and subscribe
	ch := make(chan any, 10)
	pubsub.register("session1", ch)
	pubsub.Subscribe("session1", "scope1")

	// Poke the scope
	broker.Poke("scope1", 42)

	// Wait for message
	select {
	case msg := <-ch:
		poke, ok := msg.(Poke)
		if !ok {
			t.Fatalf("expected Poke, got %T", msg)
		}
		if poke.Scope != "scope1" {
			t.Errorf("expected scope scope1, got %s", poke.Scope)
		}
		if poke.Cursor != 42 {
			t.Errorf("expected cursor 42, got %d", poke.Cursor)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for poke")
	}
}

func TestSyncPokeBroker_MultipleSubscribers(t *testing.T) {
	pubsub := NewInmemPubSub()
	broker := NewSyncPokeBroker(pubsub)

	// Create two sessions subscribed to same scope
	ch1 := make(chan any, 10)
	ch2 := make(chan any, 10)
	pubsub.register("session1", ch1)
	pubsub.register("session2", ch2)
	pubsub.Subscribe("session1", "scope1")
	pubsub.Subscribe("session2", "scope1")

	broker.Poke("scope1", 1)

	// Both should receive
	var received int
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case <-ch1:
			received++
		case <-time.After(100 * time.Millisecond):
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case <-ch2:
			received++
		case <-time.After(100 * time.Millisecond):
		}
	}()

	wg.Wait()

	if received != 2 {
		t.Errorf("expected 2 receivers to get poke, got %d", received)
	}
}

func TestSyncPokeBroker_NoSubscribers(t *testing.T) {
	pubsub := NewInmemPubSub()
	broker := NewSyncPokeBroker(pubsub)

	// Should not panic with no subscribers
	broker.Poke("nonexistent", 1)
}

func TestSyncPokeBroker_MultiplePokes(t *testing.T) {
	pubsub := NewInmemPubSub()
	broker := NewSyncPokeBroker(pubsub)

	ch := make(chan any, 10)
	pubsub.register("session1", ch)
	pubsub.Subscribe("session1", "scope1")

	broker.Poke("scope1", 1)
	broker.Poke("scope1", 2)
	broker.Poke("scope1", 3)

	var cursors []uint64
	for i := 0; i < 3; i++ {
		select {
		case msg := <-ch:
			poke := msg.(Poke)
			cursors = append(cursors, poke.Cursor)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for poke")
		}
	}

	if len(cursors) != 3 {
		t.Errorf("expected 3 pokes, got %d", len(cursors))
	}
}

func TestSyncPokeBroker_ScopeIsolation(t *testing.T) {
	pubsub := NewInmemPubSub()
	broker := NewSyncPokeBroker(pubsub)

	ch1 := make(chan any, 10)
	ch2 := make(chan any, 10)
	pubsub.register("session1", ch1)
	pubsub.register("session2", ch2)
	pubsub.Subscribe("session1", "scope1")
	pubsub.Subscribe("session2", "scope2")

	broker.Poke("scope1", 1)

	// Only session1 should receive
	select {
	case msg := <-ch1:
		poke := msg.(Poke)
		if poke.Scope != "scope1" {
			t.Errorf("unexpected scope: %s", poke.Scope)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("session1 should receive poke")
	}

	// session2 should NOT receive
	select {
	case <-ch2:
		t.Fatal("session2 should not receive poke for scope1")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestSyncPokeBroker_Interface(t *testing.T) {
	// Verify it can be used as a sync.PokeBroker interface
	// This is a compile-time check
	pubsub := NewInmemPubSub()
	var _ interface {
		Poke(scope string, cursor uint64)
	} = NewSyncPokeBroker(pubsub)
}
