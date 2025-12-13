// Package sse provides Server-Sent Events (SSE) middleware for Mizu.
//
// Server-Sent Events enable real-time server-to-client communication over HTTP.
// Unlike WebSocket, SSE is unidirectional (server to client) and uses standard HTTP,
// making it simpler to implement and more firewall-friendly.
//
// # Features
//
//   - Standards-compliant SSE implementation
//   - Buffered event channels for efficient delivery
//   - Client disconnection handling
//   - Event ID support for client resumption
//   - Named events for event type differentiation
//   - Configurable retry intervals
//   - Broker pattern for multi-client broadcasting
//   - Thread-safe operations
//
// # Basic Usage
//
// Simple event streaming:
//
//	app := mizu.New()
//
//	app.Get("/events", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    client.SendData("Hello, World!")
//	    <-client.Done
//	}))
//
// # Periodic Updates
//
// Sending events on a timer:
//
//	app.Get("/time", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    ticker := time.NewTicker(1 * time.Second)
//	    defer ticker.Stop()
//
//	    for {
//	        select {
//	        case t := <-ticker.C:
//	            client.SendData(t.Format(time.RFC3339))
//	        case <-client.Done:
//	            return
//	        }
//	    }
//	}))
//
// # Named Events
//
// Sending events with type names:
//
//	app.Get("/notifications", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    client.SendEvent("message", "New message received")
//	    client.SendEvent("alert", "System alert!")
//	    <-client.Done
//	}))
//
// # Full Event Control
//
// Using the Event structure for complete control:
//
//	app.Get("/events", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    client.Send(&sse.Event{
//	        ID:    "1",
//	        Event: "notification",
//	        Data:  "Important update",
//	        Retry: 5000,
//	    })
//	    <-client.Done
//	}))
//
// # Broadcasting with Broker
//
// Managing multiple clients with a broker:
//
//	broker := sse.NewBroker()
//
//	app.Get("/stream", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    broker.Register(client)
//	    <-client.Done
//	}))
//
//	app.Post("/broadcast", func(c *mizu.Ctx) error {
//	    message := c.FormValue("message")
//	    broker.BroadcastData(message)
//	    return c.Text(200, "Broadcasted")
//	})
//
// # Custom Options
//
// Configuring buffer size and retry interval:
//
//	app.Get("/events", sse.WithOptions(
//	    func(c *mizu.Ctx, client *sse.Client) {
//	        // Handler
//	    },
//	    sse.Options{
//	        BufferSize: 50,  // Larger buffer
//	        Retry:      5000, // 5 second retry
//	    },
//	))
//
// # Event Resumption
//
// Supporting client reconnection with Last-Event-ID:
//
//	app.Get("/events", sse.New(func(c *mizu.Ctx, client *sse.Client) {
//	    lastID := client.ID
//	    if lastID != "" {
//	        // Send missed events since lastID
//	        missedEvents := getEventsSince(lastID)
//	        for _, event := range missedEvents {
//	            client.Send(event)
//	        }
//	    }
//	    // Continue with new events
//	}))
//
// # Client-Side JavaScript
//
// Consuming SSE on the client:
//
//	const events = new EventSource('/events');
//
//	events.onmessage = (e) => {
//	    console.log('Data:', e.data);
//	};
//
//	events.addEventListener('notification', (e) => {
//	    console.log('Notification:', e.data);
//	});
//
// # Architecture
//
// The middleware implements a three-component architecture:
//
// 1. Client: Represents an individual SSE connection with buffered event
// channels and disconnection handling.
//
// 2. Event Loop: Goroutine-based event processing that listens for both
// new events and client disconnection signals.
//
// 3. Broker: Optional multi-client manager for broadcasting events to all
// connected clients using a fan-out pattern.
//
// # Connection Lifecycle
//
// 1. Accept Header Check: Validates text/event-stream or */*
// 2. Flusher Verification: Ensures HTTP flushing support
// 3. Header Setup: Sets required SSE headers
// 4. Client Creation: Initializes event and done channels
// 5. Event Loop Start: Launches event processing goroutine
// 6. Handler Execution: Runs user handler
// 7. Cleanup: Waits for disconnection
//
// # Event Format
//
// Events follow the W3C SSE specification:
//
//	id: 123
//	event: notification
//	retry: 5000
//	data: Hello, World!
//
// Multiline data is automatically split:
//
//	data: line 1
//	data: line 2
//	data: line 3
//
// # Concurrency Safety
//
// The implementation is thread-safe:
//
//   - Client send operations use mutex locks
//   - Broker uses RWMutex for client map access
//   - Done channel prevents panics on closed connections
//   - Multiple Close() calls are safe
//
// # Best Practices
//
//   - Use event IDs for client resumption
//   - Set appropriate retry intervals
//   - Always wait for client.Done before returning
//   - Use Broker for multi-client scenarios
//   - Keep payload sizes reasonable
//   - Handle client disconnections gracefully
package sse
