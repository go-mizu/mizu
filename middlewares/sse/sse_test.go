package sse

import (
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	event := &Event{
		ID:    "123",
		Event: "message",
		Data:  "hello world",
		Retry: 5000,
	}

	if event.ID != "123" {
		t.Errorf("expected ID '123', got %q", event.ID)
	}
	if event.Event != "message" {
		t.Errorf("expected Event 'message', got %q", event.Event)
	}
	if event.Data != "hello world" {
		t.Errorf("expected Data 'hello world', got %q", event.Data)
	}
	if event.Retry != 5000 {
		t.Errorf("expected Retry 5000, got %d", event.Retry)
	}
}

func TestClient_Close(t *testing.T) {
	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	// Close should not panic
	client.Close()

	// Second close should not panic
	client.Close()

	// Done should be closed
	select {
	case <-client.Done:
		// OK
	default:
		t.Error("expected Done to be closed")
	}
}

func TestClient_Send(t *testing.T) {
	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	event := &Event{Data: "test"}
	client.Send(event)

	select {
	case received := <-client.Events:
		if received.Data != "test" {
			t.Errorf("expected 'test', got %q", received.Data)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestClient_SendData(t *testing.T) {
	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	client.SendData("hello")

	select {
	case received := <-client.Events:
		if received.Data != "hello" {
			t.Errorf("expected 'hello', got %q", received.Data)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestClient_SendEvent(t *testing.T) {
	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	client.SendEvent("custom", "data")

	select {
	case received := <-client.Events:
		if received.Event != "custom" {
			t.Errorf("expected Event 'custom', got %q", received.Event)
		}
		if received.Data != "data" {
			t.Errorf("expected Data 'data', got %q", received.Data)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestBroker(t *testing.T) {
	broker := NewBroker()

	// Create clients
	client1 := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}
	client2 := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	broker.Register(client1)
	broker.Register(client2)

	// Wait for registration
	time.Sleep(10 * time.Millisecond)

	if broker.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", broker.ClientCount())
	}

	// Broadcast
	broker.BroadcastData("broadcast test")

	// Check both clients received
	for i, client := range []*Client{client1, client2} {
		select {
		case event := <-client.Events:
			if event.Data != "broadcast test" {
				t.Errorf("client %d: expected 'broadcast test', got %q", i, event.Data)
			}
		case <-time.After(time.Second):
			t.Errorf("client %d: timeout waiting for event", i)
		}
	}

	// Disconnect client1
	client1.Close()

	// Wait for unregistration
	time.Sleep(10 * time.Millisecond)

	if broker.ClientCount() != 1 {
		t.Errorf("expected 1 client after disconnect, got %d", broker.ClientCount())
	}
}

func TestBroker_BroadcastEvent(t *testing.T) {
	broker := NewBroker()

	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	broker.Register(client)
	time.Sleep(10 * time.Millisecond)

	broker.BroadcastEvent("notification", "new message")

	select {
	case event := <-client.Events:
		if event.Event != "notification" {
			t.Errorf("expected Event 'notification', got %q", event.Event)
		}
		if event.Data != "new message" {
			t.Errorf("expected Data 'new message', got %q", event.Data)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestBroker_Broadcast(t *testing.T) {
	broker := NewBroker()

	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	broker.Register(client)
	time.Sleep(10 * time.Millisecond)

	broker.Broadcast(&Event{
		ID:    "1",
		Event: "update",
		Data:  "data",
		Retry: 1000,
	})

	select {
	case event := <-client.Events:
		if event.ID != "1" {
			t.Errorf("expected ID '1', got %q", event.ID)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestOptions(t *testing.T) {
	opts := Options{
		BufferSize: 20,
		Retry:      5000,
	}

	if opts.BufferSize != 20 {
		t.Errorf("expected BufferSize 20, got %d", opts.BufferSize)
	}
	if opts.Retry != 5000 {
		t.Errorf("expected Retry 5000, got %d", opts.Retry)
	}
}
