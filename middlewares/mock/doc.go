// Package mock provides request mocking middleware for the Mizu web framework.
//
// The mock middleware enables developers to create predefined responses for specific
// routes, making it ideal for testing, development, and API stubbing scenarios.
//
// # Features
//
//   - Thread-safe mock response registration and retrieval
//   - Method-specific and path-only mocking patterns
//   - Multiple response types (JSON, Text, HTML, File, Redirect, Error)
//   - Configurable passthrough mode for unmatched requests
//   - Default response fallback for unmocked paths
//   - Prefix-based mocking for entire route segments
//
// # Basic Usage
//
// Creating a simple mock response:
//
//	app := mizu.New()
//	app.Use(mock.New(map[string]*mock.Response{
//		"/api/users": mock.JSON(200, []User{
//			{ID: 1, Name: "John"},
//			{ID: 2, Name: "Jane"},
//		}),
//	}))
//
// # Response Helpers
//
// The package provides several helper functions for common response types:
//
//	// JSON responses
//	mock.JSON(200, data)
//
//	// Text responses
//	mock.Text(200, "Hello World")
//
//	// HTML responses
//	mock.HTML(200, "<h1>Hello</h1>")
//
//	// File responses
//	mock.File("application/pdf", pdfData)
//
//	// Redirects
//	mock.Redirect("/new-location", 302)
//
//	// Error responses
//	mock.Error(400, "Invalid request")
//
// # Advanced Configuration
//
// Using Options for advanced configuration:
//
//	app.Use(mock.WithOptions(mock.Options{
//		Mocks: map[string]*mock.Response{
//			"/api/users": mock.JSON(200, users),
//		},
//		DefaultResponse: mock.Error(503, "Service unavailable"),
//		Passthrough: false,
//	}))
//
// # Dynamic Mock Management
//
// For dynamic mock registration during runtime:
//
//	m := mock.NewMock()
//	m.Register("/test", mock.Text(200, "mocked"))
//	m.RegisterMethod("POST", "/submit", mock.JSON(201, result))
//	app.Use(m.Middleware())
//
//	// Later in your code
//	m.Clear() // Remove all mocks
//
// # Prefix Mocking
//
// Mock all routes under a specific prefix:
//
//	app.Use(mock.Prefix("/api/v2", mock.Error(501, "V2 not implemented")))
//
// # Thread Safety
//
// All mock operations are protected by read/write locks, making this middleware
// safe for concurrent use in production environments. The implementation uses
// sync.RWMutex to allow multiple concurrent reads while serializing writes.
//
// # Implementation Details
//
// The middleware uses a two-tier matching system:
//  1. First checks method-specific mocks (e.g., "GET /api/users")
//  2. Falls back to path-only mocks (e.g., "/api/users" for any method)
//  3. Applies default response if configured
//  4. Either passes through to the next handler or returns 404
//
// This allows for flexible mocking patterns where you can have different responses
// for different HTTP methods on the same path, or a generic response for all methods.
package mock
