// Package jsonrpc provides JSON-RPC 2.0 protocol support for the Mizu web framework.
//
// # Overview
//
// This package implements the JSON-RPC 2.0 specification, enabling RPC-style
// communication over HTTP. It supports single requests, batch requests, notifications,
// and standard error handling.
//
// # Basic Usage
//
// Create a new JSON-RPC server and register methods:
//
//	server := jsonrpc.NewServer()
//	server.Register("add", func(params map[string]any) (any, error) {
//	    a := params["a"].(float64)
//	    b := params["b"].(float64)
//	    return a + b, nil
//	})
//
//	app := mizu.New()
//	app.Post("/rpc", server.Handler())
//
// # Request Format
//
// JSON-RPC 2.0 requests must include:
//   - jsonrpc: Must be "2.0"
//   - method: The name of the method to invoke
//   - params: (optional) Parameters as a map
//   - id: (optional) Request identifier; omit for notifications
//
// Example request:
//
//	{
//	    "jsonrpc": "2.0",
//	    "method": "add",
//	    "params": {"a": 5, "b": 3},
//	    "id": 1
//	}
//
// # Response Format
//
// Successful responses include:
//   - jsonrpc: "2.0"
//   - result: The method's return value
//   - id: The request identifier
//
// Error responses include:
//   - jsonrpc: "2.0"
//   - error: Error object with code, message, and optional data
//   - id: The request identifier (or null)
//
// # Batch Requests
//
// The server automatically handles batch requests (arrays of request objects):
//
//	[
//	    {"jsonrpc": "2.0", "method": "add", "params": {"a": 1, "b": 2}, "id": 1},
//	    {"jsonrpc": "2.0", "method": "multiply", "params": {"a": 3, "b": 4}, "id": 2}
//	]
//
// Batch responses are returned as an array of response objects.
//
// # Notifications
//
// Requests without an ID field are treated as notifications. The server
// executes the method but does not send a response. For notifications,
// the server returns HTTP 204 No Content.
//
// # Error Handling
//
// The package defines standard JSON-RPC error codes:
//   - ParseError (-32700): Invalid JSON received
//   - InvalidRequest (-32600): Request doesn't conform to JSON-RPC spec
//   - MethodNotFound (-32601): Requested method doesn't exist
//   - InvalidParams (-32602): Invalid method parameters
//   - InternalError (-32603): Handler execution error
//
// Create custom errors using NewError:
//
//	return nil, jsonrpc.NewError(-32000, "Custom application error")
//
// # Method Handlers
//
// Method handlers must have the signature:
//
//	func(params map[string]any) (any, error)
//
// The handler receives parameters as a map and returns a result or error.
// Errors can be standard Go errors or JSON-RPC errors created with NewError.
//
// # Middleware
//
// The package provides a validation middleware that ensures:
//   - HTTP method is POST
//   - Content-Type is application/json
//
// Usage:
//
//	app := mizu.New()
//	app.Use(jsonrpc.Middleware())
//	app.Post("/rpc", server.Handler())
//
// # Architecture
//
// The implementation consists of:
//
//   - Server: Manages method registration and request routing
//   - Request/Response types: Structured types for protocol compliance
//   - Handler functions: Execute registered methods
//   - Error types: Standard and custom error handling
//
// Request processing flow:
//  1. Receive and parse HTTP POST body
//  2. Detect batch vs. single request
//  3. Validate JSON-RPC 2.0 compliance
//  4. Look up and execute method handler
//  5. Generate and return response
//
// # Thread Safety
//
// The Server type is safe for concurrent use after initial setup (method
// registration). However, method registration itself is not thread-safe
// and should be completed before the server handles requests.
package jsonrpc
