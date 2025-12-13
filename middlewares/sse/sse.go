// Package sse provides Server-Sent Events middleware for Mizu.
package sse

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

// Event represents an SSE event.
type Event struct {
	// ID is the event ID.
	ID string

	// Event is the event type.
	Event string

	// Data is the event data.
	Data string

	// Retry is the reconnection time in milliseconds.
	Retry int
}

// Client represents an SSE client.
type Client struct {
	// ID is the client identifier.
	ID string

	// Events is the channel for sending events.
	Events chan *Event

	// Done is closed when the client disconnects.
	Done chan struct{}

	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

// Handler is an SSE handler function.
type Handler func(c *mizu.Ctx, client *Client)

// Options configures the SSE middleware.
type Options struct {
	// BufferSize is the event channel buffer size.
	// Default: 10.
	BufferSize int

	// Retry is the default retry time in milliseconds.
	// Default: 3000.
	Retry int
}

// New creates SSE middleware with handler.
func New(handler Handler) mizu.Middleware {
	return WithOptions(handler, Options{})
}

// WithOptions creates SSE middleware with custom options.
//
//nolint:cyclop // SSE handling requires multiple connection and event checks
func WithOptions(handler Handler, opts Options) mizu.Middleware {
	if opts.BufferSize == 0 {
		opts.BufferSize = 10
	}
	if opts.Retry == 0 {
		opts.Retry = 3000
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check if client accepts SSE
			accept := c.Request().Header.Get("Accept")
			if !strings.Contains(accept, "text/event-stream") && accept != "*/*" && accept != "" {
				return next(c)
			}

			// Get flusher
			flusher, ok := c.Writer().(http.Flusher)
			if !ok {
				return c.Text(http.StatusInternalServerError, "streaming not supported")
			}

			// Set SSE headers
			c.Writer().Header().Set("Content-Type", "text/event-stream")
			c.Writer().Header().Set("Cache-Control", "no-cache")
			c.Writer().Header().Set("Connection", "keep-alive")
			c.Writer().Header().Set("X-Accel-Buffering", "no")

			// Create client
			client := &Client{
				ID:      c.Request().Header.Get("Last-Event-ID"),
				Events:  make(chan *Event, opts.BufferSize),
				Done:    make(chan struct{}),
				w:       c.Writer(),
				flusher: flusher,
			}

			// Send initial retry
			if opts.Retry > 0 {
				_, _ = fmt.Fprintf(c.Writer(), "retry: %d\n\n", opts.Retry)
				flusher.Flush()
			}

			// Handle client disconnection
			ctx := c.Request().Context()
			go func() {
				<-ctx.Done()
				close(client.Done)
			}()

			// Start event loop
			go func() {
				for {
					select {
					case event := <-client.Events:
						client.send(event)
					case <-client.Done:
						return
					}
				}
			}()

			// Call handler
			handler(c, client)

			// Wait for client to disconnect
			<-client.Done

			return nil
		}
	}
}

// Send sends an event to the client.
func (c *Client) Send(event *Event) {
	select {
	case c.Events <- event:
	case <-c.Done:
	}
}

// SendData sends a data-only event.
func (c *Client) SendData(data string) {
	c.Send(&Event{Data: data})
}

// SendEvent sends a named event with data.
func (c *Client) SendEvent(eventType, data string) {
	c.Send(&Event{Event: eventType, Data: data})
}

// Close closes the client connection.
func (c *Client) Close() {
	select {
	case <-c.Done:
	default:
		close(c.Done)
	}
}

func (c *Client) send(event *Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if event.ID != "" {
		_, _ = fmt.Fprintf(c.w, "id: %s\n", event.ID)
	}
	if event.Event != "" {
		_, _ = fmt.Fprintf(c.w, "event: %s\n", event.Event)
	}
	if event.Retry > 0 {
		_, _ = fmt.Fprintf(c.w, "retry: %d\n", event.Retry)
	}
	if event.Data != "" {
		// Split data by newlines
		lines := strings.Split(event.Data, "\n")
		for _, line := range lines {
			_, _ = fmt.Fprintf(c.w, "data: %s\n", line)
		}
	}
	_, _ = fmt.Fprintf(c.w, "\n")
	c.flusher.Flush()
}

// Broker manages multiple SSE clients.
type Broker struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Event
	mu         sync.RWMutex
}

// NewBroker creates a new SSE broker.
func NewBroker() *Broker {
	b := &Broker{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Event, 100),
	}
	go b.run()
	return b
}

func (b *Broker) run() {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
		case client := <-b.unregister:
			b.mu.Lock()
			delete(b.clients, client)
			b.mu.Unlock()
		case event := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client.Events <- event:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()
		}
	}
}

// Register registers a client with the broker.
func (b *Broker) Register(client *Client) {
	b.register <- client

	// Unregister on disconnect
	go func() {
		<-client.Done
		b.unregister <- client
	}()
}

// Broadcast sends an event to all connected clients.
func (b *Broker) Broadcast(event *Event) {
	b.broadcast <- event
}

// BroadcastData broadcasts data to all clients.
func (b *Broker) BroadcastData(data string) {
	b.Broadcast(&Event{Data: data})
}

// BroadcastEvent broadcasts a named event to all clients.
func (b *Broker) BroadcastEvent(eventType, data string) {
	b.Broadcast(&Event{Event: eventType, Data: data})
}

// ClientCount returns the number of connected clients.
func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
