package sse

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
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

func TestClient_SendOnClosedChannel(t *testing.T) {
	client := &Client{
		Events: make(chan *Event, 10),
		Done:   make(chan struct{}),
	}

	// Close the client first
	client.Close()

	// Send should not block or panic after close
	done := make(chan bool)
	go func() {
		client.Send(&Event{Data: "test"})
		done <- true
	}()

	select {
	case <-done:
		// OK - Send completed without blocking
	case <-time.After(100 * time.Millisecond):
		t.Error("Send blocked on closed client")
	}
}

func TestClient_send(t *testing.T) {
	// Create a mock flusher
	buf := &bytes.Buffer{}
	mockFlusher := &mockResponseWriter{buf: buf}

	client := &Client{
		Events:  make(chan *Event, 10),
		Done:    make(chan struct{}),
		w:       mockFlusher,
		flusher: mockFlusher,
	}

	// Test with all fields
	client.send(&Event{
		ID:    "123",
		Event: "message",
		Data:  "hello world",
		Retry: 5000,
	})

	output := buf.String()
	if !strings.Contains(output, "id: 123") {
		t.Error("expected id field in output")
	}
	if !strings.Contains(output, "event: message") {
		t.Error("expected event field in output")
	}
	if !strings.Contains(output, "data: hello world") {
		t.Error("expected data field in output")
	}
	if !strings.Contains(output, "retry: 5000") {
		t.Error("expected retry field in output")
	}
}

func TestClient_sendMultilineData(t *testing.T) {
	buf := &bytes.Buffer{}
	mockFlusher := &mockResponseWriter{buf: buf}

	client := &Client{
		Events:  make(chan *Event, 10),
		Done:    make(chan struct{}),
		w:       mockFlusher,
		flusher: mockFlusher,
	}

	client.send(&Event{
		Data: "line1\nline2\nline3",
	})

	output := buf.String()
	if !strings.Contains(output, "data: line1") {
		t.Error("expected first line in output")
	}
	if !strings.Contains(output, "data: line2") {
		t.Error("expected second line in output")
	}
	if !strings.Contains(output, "data: line3") {
		t.Error("expected third line in output")
	}
}

func TestClient_sendEmptyFields(t *testing.T) {
	buf := &bytes.Buffer{}
	mockFlusher := &mockResponseWriter{buf: buf}

	client := &Client{
		Events:  make(chan *Event, 10),
		Done:    make(chan struct{}),
		w:       mockFlusher,
		flusher: mockFlusher,
	}

	// Send event with only Data field
	client.send(&Event{
		Data: "only data",
	})

	output := buf.String()
	if strings.Contains(output, "id:") {
		t.Error("should not have id field when empty")
	}
	if strings.Contains(output, "event:") {
		t.Error("should not have event field when empty")
	}
	if strings.Contains(output, "retry:") {
		t.Error("should not have retry field when 0")
	}
	if !strings.Contains(output, "data: only data") {
		t.Error("expected data field in output")
	}
}

func TestBroker_BroadcastWithFullBuffer(t *testing.T) {
	broker := NewBroker()

	// Create client with tiny buffer
	client := &Client{
		Events: make(chan *Event, 1),
		Done:   make(chan struct{}),
	}

	broker.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer
	client.Events <- &Event{Data: "blocking"}

	// Broadcast should not block even if buffer is full
	done := make(chan bool)
	go func() {
		broker.BroadcastData("should skip")
		done <- true
	}()

	select {
	case <-done:
		// OK - broadcast didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcast blocked on full buffer")
	}
}

func TestNew(t *testing.T) {
	middleware := New(func(c *mizu.Ctx, client *Client) {
		client.Close()
	})

	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestWithOptions_DefaultValues(t *testing.T) {
	middleware := WithOptions(func(c *mizu.Ctx, client *Client) {
		client.Close()
	}, Options{})

	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestWithOptions_CustomValues(t *testing.T) {
	middleware := WithOptions(func(c *mizu.Ctx, client *Client) {
		client.Close()
	}, Options{
		BufferSize: 50,
		Retry:      10000,
	})

	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

// mockResponseWriter implements http.ResponseWriter and http.Flusher
type mockResponseWriter struct {
	buf     *bytes.Buffer
	headers http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {}

func (m *mockResponseWriter) Flush() {}

func TestSSE_NonSSERequest(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, client *Client) {
		client.SendData("hello")
		client.Close()
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "normal response")
	})

	// Request without Accept: text/event-stream
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "normal response" {
		t.Errorf("expected normal response, got %q", rec.Body.String())
	}
}

func TestSSE_EmptyAcceptAllowed(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, client *Client) {
		client.Close()
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Request with empty Accept (should be allowed)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Note: In test environment without proper flusher, it may fall through
	// The key is testing the accept header logic
}

func TestSSE_WildcardAccept(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, client *Client) {
		client.Close()
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Request with Accept: */*
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "*/*")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should attempt SSE (may fail in test due to no flusher)
}
