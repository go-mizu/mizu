// Package sentry provides error tracking and performance monitoring middleware for Mizu applications.
//
// The sentry middleware integrates with Sentry (or compatible error tracking services) to capture
// errors, panics, and custom messages from your application. It provides a lightweight implementation
// with support for custom transports, event hooks, and contextual data.
//
// # Basic Usage
//
// Initialize the middleware with default options:
//
//	app := mizu.New()
//	app.Use(sentry.New())
//
// Or with custom configuration:
//
//	app.Use(sentry.WithOptions(sentry.Options{
//		DSN:         "https://key@sentry.io/project",
//		Environment: "production",
//		Release:     "v1.0.0",
//		SampleRate:  1.0,
//	}))
//
// # Error Capture
//
// Errors returned from handlers are automatically captured:
//
//	app.Get("/", func(c *mizu.Ctx) error {
//		return errors.New("something went wrong")
//	})
//
// Manually capture errors with additional context:
//
//	app.Get("/api", func(c *mizu.Ctx) error {
//		if err := doSomething(); err != nil {
//			sentry.CaptureError(c, err)
//			return c.JSON(500, map[string]string{"error": "Internal error"})
//		}
//		return c.JSON(200, data)
//	})
//
// # Panic Recovery
//
// Panics are automatically captured and reported as fatal errors before being re-thrown:
//
//	app.Get("/panic", func(c *mizu.Ctx) error {
//		panic("critical failure") // Will be captured with stack trace
//	})
//
// # Context Enrichment
//
// Add user information, tags, and extra data to events:
//
//	app.Use(func(next mizu.Handler) mizu.Handler {
//		return func(c *mizu.Ctx) error {
//			sentry.SetUser(c, &sentry.User{
//				ID:    getUserID(c),
//				Email: getEmail(c),
//			})
//			sentry.SetTag(c, "version", "2.0")
//			sentry.SetExtra(c, "metadata", customData)
//			return next(c)
//		}
//	})
//
// # Event Hooks
//
// Use BeforeSend to modify or drop events before they're sent:
//
//	app.Use(sentry.WithOptions(sentry.Options{
//		BeforeSend: func(event *sentry.Event) *sentry.Event {
//			// Modify event
//			event.Tags["custom"] = "value"
//			// Or drop event
//			if shouldIgnore(event) {
//				return nil
//			}
//			return event
//		},
//	}))
//
// Use OnError to react to captured errors:
//
//	app.Use(sentry.WithOptions(sentry.Options{
//		OnError: func(event *sentry.Event) {
//			log.Printf("Error captured: %s", event.Message)
//		},
//	}))
//
// # Custom Transport
//
// Implement custom event delivery by providing a Transport:
//
//	type CustomTransport struct{}
//
//	func (t *CustomTransport) Send(event *sentry.Event) error {
//		// Send event to your service
//		return sendToService(event)
//	}
//
//	app.Use(sentry.WithOptions(sentry.Options{
//		Transport: &CustomTransport{},
//	}))
//
// # Testing
//
// Use MockTransport for testing:
//
//	transport := &sentry.MockTransport{}
//	app.Use(sentry.WithOptions(sentry.Options{
//		Transport: transport,
//	}))
//
//	// After running tests
//	events := transport.Events()
//	if len(events) != expectedCount {
//		t.Errorf("expected %d events, got %d", expectedCount, len(events))
//	}
//
// # Architecture
//
// The middleware is built around the following components:
//
// Hub: Central management structure that handles event capture, storage, and transport.
// Each middleware instance creates a Hub stored in the request context.
//
// Event: Represents an error event with metadata including event ID, timestamp, severity level,
// exception details with stack traces, HTTP request information, user context, tags, and extra data.
//
// Transport: Interface for sending events to Sentry. Custom transports can be implemented
// by satisfying the Transport interface.
//
// # Security
//
// Sensitive HTTP headers are automatically filtered from events:
//   - Authorization
//   - Cookie
//   - X-Api-Key
//
// Request bodies are not captured by default. Enable with CaptureRequestBody option and
// configure MaxRequestBodySize to limit size.
//
// # Performance
//
// Events are sent asynchronously to avoid blocking request processing. Use SampleRate
// to reduce the volume of events sent to Sentry:
//
//	app.Use(sentry.WithOptions(sentry.Options{
//		SampleRate: 0.1, // Capture 10% of events
//	}))
package sentry
