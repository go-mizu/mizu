package mcp

import "encoding/json"

// Standard JSON-RPC 2.0 error codes.
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603
)

// request is the JSON-RPC 2.0 request structure.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// hasID returns true if the request has an id (not a notification).
func (r *request) hasID() bool {
	return r.ID != nil
}

// response is the JSON-RPC 2.0 response structure.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is the JSON-RPC 2.0 error structure.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// initializeResult is the result for initialize response.
type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      *serverInfo        `json:"serverInfo,omitempty"`
}

// serverCapabilities declares server features.
type serverCapabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

// toolsCapability indicates tools support.
type toolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// serverInfo identifies the MCP server.
type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// tool represents an MCP tool definition.
type tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema jsonSchema `json:"inputSchema"`
}

// jsonSchema is a simplified JSON Schema object.
type jsonSchema struct {
	Type       string                `json:"type"`
	Properties map[string]jsonSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Items      *jsonSchema           `json:"items,omitempty"`
}

// toolsListResult is the result for tools/list response.
type toolsListResult struct {
	Tools []tool `json:"tools"`
}

// toolsCallParams is the params for tools/call request.
type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// toolsCallResult is the result for tools/call response.
type toolsCallResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// content is a content item in tool result.
type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
