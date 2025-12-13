// Package jsonrpc provides JSON-RPC 2.0 middleware for Mizu.
package jsonrpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
	ID      any            `json:"id,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id"`
}

// Error represents a JSON-RPC error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Handler handles a JSON-RPC method.
type Handler func(params map[string]any) (any, error)

// Server is a JSON-RPC server.
type Server struct {
	methods map[string]Handler
}

// NewServer creates a new JSON-RPC server.
func NewServer() *Server {
	return &Server{
		methods: make(map[string]Handler),
	}
}

// Register registers a method handler.
func (s *Server) Register(method string, handler Handler) {
	s.methods[method] = handler
}

// Handler returns a Mizu handler for the JSON-RPC server.
func (s *Server) Handler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		// Read request body
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return s.sendError(c, nil, ParseError, "Parse error")
		}

		// Check for batch request
		if len(body) > 0 && body[0] == '[' {
			return s.handleBatch(c, body)
		}

		// Single request
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			return s.sendError(c, nil, ParseError, "Parse error")
		}

		return s.handleRequest(c, req)
	}
}

func (s *Server) handleRequest(c *mizu.Ctx, req Request) error {
	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return s.sendError(c, req.ID, InvalidRequest, "Invalid Request")
	}

	// Find handler
	handler, ok := s.methods[req.Method]
	if !ok {
		return s.sendError(c, req.ID, MethodNotFound, "Method not found")
	}

	// Execute handler
	result, err := handler(req.Params)
	if err != nil {
		var rpcErr *Error
		if errors.As(err, &rpcErr) {
			return s.sendError(c, req.ID, rpcErr.Code, rpcErr.Message)
		}
		return s.sendError(c, req.ID, InternalError, err.Error())
	}

	// Notification (no ID)
	if req.ID == nil {
		c.Writer().WriteHeader(http.StatusNoContent)
		return nil
	}

	// Send result
	return c.JSON(http.StatusOK, Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

func (s *Server) handleBatch(c *mizu.Ctx, body []byte) error {
	var requests []Request
	if err := json.Unmarshal(body, &requests); err != nil {
		return s.sendError(c, nil, ParseError, "Parse error")
	}

	if len(requests) == 0 {
		return s.sendError(c, nil, InvalidRequest, "Invalid Request")
	}

	var responses []Response
	for _, req := range requests {
		if req.JSONRPC != "2.0" {
			responses = append(responses, Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: InvalidRequest, Message: "Invalid Request"},
				ID:      req.ID,
			})
			continue
		}

		handler, ok := s.methods[req.Method]
		if !ok {
			responses = append(responses, Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: MethodNotFound, Message: "Method not found"},
				ID:      req.ID,
			})
			continue
		}

		result, err := handler(req.Params)
		if err != nil {
			rpcErr := &Error{Code: InternalError, Message: err.Error()}
			var e *Error
			if errors.As(err, &e) {
				rpcErr = e
			}
			responses = append(responses, Response{
				JSONRPC: "2.0",
				Error:   rpcErr,
				ID:      req.ID,
			})
			continue
		}

		// Skip notifications
		if req.ID != nil {
			responses = append(responses, Response{
				JSONRPC: "2.0",
				Result:  result,
				ID:      req.ID,
			})
		}
	}

	if len(responses) == 0 {
		c.Writer().WriteHeader(http.StatusNoContent)
		return nil
	}

	return c.JSON(http.StatusOK, responses)
}

func (s *Server) sendError(c *mizu.Ctx, id any, code int, message string) error {
	return c.JSON(http.StatusOK, Response{
		JSONRPC: "2.0",
		Error:   &Error{Code: code, Message: message},
		ID:      id,
	})
}

// NewError creates a new JSON-RPC error.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func (e *Error) Error() string {
	return e.Message
}

// Middleware creates JSON-RPC middleware that validates requests.
func Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().Method != http.MethodPost {
				return c.JSON(http.StatusOK, Response{
					JSONRPC: "2.0",
					Error:   &Error{Code: InvalidRequest, Message: "Method must be POST"},
				})
			}

			contentType := c.Request().Header.Get("Content-Type")
			if contentType != "application/json" {
				return c.JSON(http.StatusOK, Response{
					JSONRPC: "2.0",
					Error:   &Error{Code: InvalidRequest, Message: "Content-Type must be application/json"},
				})
			}

			return next(c)
		}
	}
}
