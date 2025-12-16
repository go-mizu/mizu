// Package mcp implements MCP (Model Context Protocol) transport for contract services.
//
// MCP is a standardized protocol for AI model interactions. This package
// implements a tools-only MCP server that exposes contract services as MCP tools.
//
// Usage:
//
//	svc, _ := contract.Register("todo", &TodoService{})
//	mcp.Mount(mux, "/mcp", svc)
//
// The MCP transport supports:
//   - initialize: Protocol negotiation
//   - tools/list: List available tools
//   - tools/call: Call a tool (method)
//
// Tool names follow the "<service>.<method>" convention (e.g., "todo.Create").
package mcp

import "encoding/json"

// Protocol versions supported by this implementation.
const (
	ProtocolLatest   = "2025-06-18"
	ProtocolFallback = "2025-03-26"
	ProtocolLegacy   = "2024-11-05"
)

// ServerInfo contains MCP server metadata.
type ServerInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// DefaultServerInfo returns default server info.
func DefaultServerInfo() ServerInfo {
	return ServerInfo{
		Name:    "mizu-contract",
		Title:   "Mizu Contract MCP Server",
		Version: "0.1.0",
	}
}

// Capabilities describes server capabilities.
type Capabilities struct {
	Tools *ToolCapabilities `json:"tools,omitempty"`
}

// ToolCapabilities describes tool-related capabilities.
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged"`
}

// InitializeParams are parameters for the initialize request.
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      json.RawMessage `json:"clientInfo,omitempty"`
}

// InitializeResult is the result of initialization.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Instructions    string       `json:"instructions,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name         string         `json:"name"`
	Title        string         `json:"title,omitempty"`
	Description  string         `json:"description,omitempty"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

// ToolCallParams are parameters for tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult is the result of a tool call.
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

// ContentBlock represents content in a tool result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// TextContent creates a text content block.
func TextContent(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// ErrorResult creates an error tool result.
func ErrorResult(msg string) ToolCallResult {
	return ToolCallResult{
		Content: []ContentBlock{TextContent(msg)},
		IsError: true,
	}
}

// SuccessResult creates a success tool result.
func SuccessResult(text string) ToolCallResult {
	return ToolCallResult{
		Content: []ContentBlock{TextContent(text)},
		IsError: false,
	}
}

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is a JSON-RPC 2.0 error.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

func parseError(err error) *rpcError {
	msg := "Parse error"
	if err != nil {
		return &rpcError{Code: codeParseError, Message: msg, Data: err.Error()}
	}
	return &rpcError{Code: codeParseError, Message: msg}
}

func invalidRequest(msg string) *rpcError {
	if msg == "" {
		msg = "Invalid Request"
	}
	return &rpcError{Code: codeInvalidRequest, Message: msg}
}

func methodNotFound(method string) *rpcError {
	return &rpcError{
		Code:    codeMethodNotFound,
		Message: "Method not found",
		Data:    map[string]any{"method": method},
	}
}

func invalidParams(err error) *rpcError {
	if err != nil {
		return &rpcError{Code: codeInvalidParams, Message: "Invalid params", Data: err.Error()}
	}
	return &rpcError{Code: codeInvalidParams, Message: "Invalid params"}
}

// isProtocolSupported checks if a protocol version is supported.
func isProtocolSupported(v string) bool {
	switch v {
	case ProtocolLatest, ProtocolFallback, ProtocolLegacy:
		return true
	default:
		return false
	}
}

// negotiateProtocol returns the best protocol version to use.
func negotiateProtocol(requested string) string {
	if requested == "" {
		return ProtocolLatest
	}
	if isProtocolSupported(requested) {
		return requested
	}
	return ""
}
