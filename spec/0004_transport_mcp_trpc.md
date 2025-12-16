# MCP and tRPC Transport Layer Refactoring Specification

## Overview

This document specifies the refactoring of the MCP (Model Context Protocol) and tRPC transports from `contract/transport_mcp.go` and `contract/transport_trpc.go` into dedicated packages under `contract/transport/`, following the patterns established in spec/0003_transport.md.

## Goals

1. **Modular Architecture**: Move MCP and tRPC into dedicated packages
2. **Consistency**: Follow the same patterns as `jsonrpc` and `openapi` packages
3. **Clean Separation**: Remove duplicated code, use shared transport abstractions
4. **Enhanced Features**: Add options pattern, middleware support, and better error handling
5. **Deprecation Path**: Keep backward-compatible Mount functions in main package

## Current State

### Files to Refactor

```
contract/
├── transport_mcp.go      # ~500 lines - MCP implementation
├── transport_trpc.go     # ~220 lines - tRPC implementation
└── helpers.go            # Deprecated helper functions
```

### Problems with Current Implementation

1. **Duplicated Code**: Both MCP and tRPC have their own JSON helpers (`mcpIsJSONNull`, `trpcIsJSONNull`)
2. **No Options Pattern**: No way to customize behavior without modifying source
3. **Tight Coupling**: Implementations are in main package instead of isolated packages
4. **Limited Extensibility**: No middleware hooks for authentication, logging, etc.
5. **Inconsistent Error Handling**: Different error formats across transports

## Target Architecture

### Package Structure

```
contract/
├── contract.go              # Core: Service, Method, Register
├── schema.go                # TypeRegistry, schema generation
├── errors.go                # Error types and codes
├── transport.go             # Transport interfaces and shared types
├── transport/
│   ├── jsonrpc/             # (already exists)
│   │   ├── jsonrpc.go
│   │   ├── handler.go
│   │   └── jsonrpc_test.go
│   ├── openapi/             # (already exists)
│   │   ├── spec.go
│   │   ├── handler.go
│   │   └── openapi_test.go
│   ├── mcp/                 # NEW
│   │   ├── mcp.go           # MCP types and constants
│   │   ├── handler.go       # HTTP handler implementation
│   │   ├── tools.go         # Tool definition helpers
│   │   └── mcp_test.go
│   └── trpc/                # NEW
│       ├── trpc.go          # tRPC types and envelope
│       ├── handler.go       # HTTP handler implementation
│       └── trpc_test.go
├── transport_mcp.go         # Deprecated: forwarding wrapper
├── transport_trpc.go        # Deprecated: forwarding wrapper
└── helpers.go               # Deprecated: to be removed
```

## MCP Package (`contract/transport/mcp`)

### Overview

MCP (Model Context Protocol) is a standardized protocol for AI model interactions. This package implements a tools-only MCP server that exposes contract services as MCP tools.

### Types (`mcp.go`)

```go
package mcp

import (
    "encoding/json"
)

// Protocol versions supported
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
    Tools       *ToolCapabilities       `json:"tools,omitempty"`
    Resources   *ResourceCapabilities   `json:"resources,omitempty"`
    Prompts     *PromptCapabilities     `json:"prompts,omitempty"`
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
```

### Handler (`handler.go`)

```go
package mcp

import (
    "context"
    "encoding/json"
    "net/http"
    "net/url"
    "strings"

    "github.com/go-mizu/mizu/contract"
)

// Handler handles MCP requests over HTTP (Streamable HTTP transport).
type Handler struct {
    service    *contract.Service
    resolver   contract.Resolver
    invoker    contract.TransportInvoker
    serverInfo ServerInfo

    // Options
    allowedOrigins []string
    instructions   string
}

// Option configures the handler.
type Option func(*Handler)

// WithServerInfo sets custom server info.
func WithServerInfo(info ServerInfo) Option {
    return func(h *Handler) { h.serverInfo = info }
}

// WithInstructions sets initialization instructions.
func WithInstructions(instructions string) Option {
    return func(h *Handler) { h.instructions = instructions }
}

// WithAllowedOrigins sets allowed CORS origins.
func WithAllowedOrigins(origins ...string) Option {
    return func(h *Handler) { h.allowedOrigins = origins }
}

// WithResolver sets a custom method resolver.
func WithResolver(r contract.Resolver) Option {
    return func(h *Handler) { h.resolver = r }
}

// WithInvoker sets a custom method invoker.
func WithInvoker(i contract.TransportInvoker) Option {
    return func(h *Handler) { h.invoker = i }
}

// NewHandler creates a new MCP handler.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
    h := &Handler{
        service:    svc,
        serverInfo: DefaultServerInfo(),
        resolver:   &contract.ServiceResolver{Service: svc},
        invoker:    &contract.DefaultInvoker{},
    }
    for _, opt := range opts {
        opt(h)
    }
    return h
}

// Name returns the transport name.
func (h *Handler) Name() string {
    return "mcp"
}

// ServeHTTP handles HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if !h.allowOrigin(r) {
        http.Error(w, "invalid origin", http.StatusForbidden)
        return
    }

    switch r.Method {
    case http.MethodGet:
        h.handleSSE(w, r)
    case http.MethodPost:
        h.handlePost(w, r)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
    // Minimal SSE response for endpoint validation
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    _, _ = w.Write([]byte(": mcp\n\n"))
}

func (h *Handler) handlePost(w http.ResponseWriter, r *http.Request) {
    // Validate protocol version if present
    if pv := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version")); pv != "" {
        if !isProtocolSupported(pv) {
            http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
            return
        }
    }

    var raw json.RawMessage
    if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
        writeRPCError(w, nil, parseError(err))
        return
    }

    raw = contract.TrimJSONSpace(raw)
    if len(raw) == 0 {
        writeRPCError(w, nil, invalidRequest("empty body"))
        return
    }

    // Check for response or notification (return 202)
    if isRPCResponse(raw) || isRPCNotification(raw) {
        w.WriteHeader(http.StatusAccepted)
        return
    }

    var req rpcRequest
    if err := json.Unmarshal(raw, &req); err != nil {
        writeRPCError(w, nil, parseError(err))
        return
    }

    if req.JSONRPC != "2.0" {
        writeRPCError(w, req.ID, invalidRequest("jsonrpc must be 2.0"))
        return
    }

    // Handle notification (no ID)
    if len(req.ID) == 0 || contract.IsJSONNull(req.ID) {
        w.WriteHeader(http.StatusAccepted)
        return
    }

    h.handleMethod(w, r.Context(), req)
}

func (h *Handler) handleMethod(w http.ResponseWriter, ctx context.Context, req rpcRequest) {
    switch req.Method {
    case "initialize":
        h.handleInitialize(w, req.ID, req.Params)
    case "notifications/initialized":
        w.WriteHeader(http.StatusAccepted)
    case "tools/list":
        h.handleToolsList(w, req.ID, req.Params)
    case "tools/call":
        h.handleToolsCall(w, ctx, req.ID, req.Params)
    default:
        writeRPCError(w, req.ID, methodNotFound(req.Method))
    }
}

func (h *Handler) handleInitialize(w http.ResponseWriter, id json.RawMessage, params json.RawMessage) {
    var p InitializeParams
    _ = json.Unmarshal(params, &p)

    negotiated := negotiateProtocol(p.ProtocolVersion)
    if negotiated == "" {
        writeRPCError(w, id, &rpcError{
            Code:    -32602,
            Message: "Unsupported protocol version",
            Data: map[string]any{
                "supported": []string{ProtocolLatest, ProtocolFallback, ProtocolLegacy},
                "requested": p.ProtocolVersion,
            },
        })
        return
    }

    result := InitializeResult{
        ProtocolVersion: negotiated,
        Capabilities: Capabilities{
            Tools: &ToolCapabilities{ListChanged: false},
        },
        ServerInfo:   h.serverInfo,
        Instructions: h.instructions,
    }

    writeRPCResult(w, id, result)
}

func (h *Handler) handleToolsList(w http.ResponseWriter, id json.RawMessage, params json.RawMessage) {
    tools := make([]Tool, 0, len(h.service.Methods))
    for _, m := range h.service.Methods {
        tools = append(tools, h.buildTool(m))
    }

    writeRPCResult(w, id, map[string]any{"tools": tools})
}

func (h *Handler) handleToolsCall(w http.ResponseWriter, ctx context.Context, id json.RawMessage, params json.RawMessage) {
    var p ToolCallParams
    if err := json.Unmarshal(params, &p); err != nil {
        writeRPCError(w, id, invalidParams(err))
        return
    }

    name := strings.TrimSpace(p.Name)
    if name == "" {
        writeRPCError(w, id, invalidParams(nil))
        return
    }

    method := h.resolveTool(name)
    if method == nil {
        writeRPCError(w, id, methodNotFound("tools/call: "+name))
        return
    }

    result, err := h.invoker.Invoke(ctx, method, p.Arguments)
    if err != nil {
        writeRPCResult(w, id, ErrorResult(err.Error()))
        return
    }

    text := "ok"
    if result != nil {
        b, _ := json.Marshal(result)
        text = string(b)
    }

    writeRPCResult(w, id, SuccessResult(text))
}

func (h *Handler) buildTool(m *contract.Method) Tool {
    tool := Tool{
        Name:        h.service.Name + "." + m.Name,
        Title:       m.FullName,
        Description: m.Description,
        InputSchema: map[string]any{
            "type":       "object",
            "properties": map[string]any{},
        },
    }

    if m.Input != nil {
        if schema, ok := h.service.Types.Schema(m.Input.ID); ok {
            tool.InputSchema = schema.JSON
        }
    }

    if m.Output != nil {
        if schema, ok := h.service.Types.Schema(m.Output.ID); ok {
            tool.OutputSchema = schema.JSON
        }
    }

    return tool
}

func (h *Handler) resolveTool(name string) *contract.Method {
    return h.resolver.Resolve(name)
}

func (h *Handler) allowOrigin(r *http.Request) bool {
    origin := strings.TrimSpace(r.Header.Get("Origin"))
    if origin == "" {
        return true
    }

    // Check allowed origins
    if len(h.allowedOrigins) > 0 {
        for _, allowed := range h.allowedOrigins {
            if allowed == "*" || allowed == origin {
                return true
            }
        }
        return false
    }

    // Default: same-host check
    u, err := url.Parse(origin)
    if err != nil || u.Host == "" {
        return false
    }
    return sameHost(u.Host, r.Host)
}

// Mount registers the MCP handler at the given path.
func Mount(mux *http.ServeMux, path string, svc *contract.Service, opts ...Option) {
    if mux == nil || svc == nil {
        return
    }
    if path == "" {
        path = "/mcp"
    }
    if !strings.HasPrefix(path, "/") {
        path = "/" + path
    }
    mux.Handle(path, NewHandler(svc, opts...))
}

// Helper functions and types...
// (rpcRequest, rpcError, writeRPCResult, writeRPCError, etc.)
```

### Tools Helper (`tools.go`)

```go
package mcp

import (
    "github.com/go-mizu/mizu/contract"
)

// ToolNamer customizes tool naming.
type ToolNamer func(svc *contract.Service, m *contract.Method) string

// DefaultToolNamer returns "service.method" format.
func DefaultToolNamer(svc *contract.Service, m *contract.Method) string {
    return svc.Name + "." + m.Name
}

// FlatToolNamer returns just the method name.
func FlatToolNamer(svc *contract.Service, m *contract.Method) string {
    return m.Name
}

// ToolFilter filters methods exposed as tools.
type ToolFilter func(m *contract.Method) bool

// AllTools exposes all methods.
func AllTools(m *contract.Method) bool {
    return true
}

// ExcludeDeprecated excludes deprecated methods.
func ExcludeDeprecated(m *contract.Method) bool {
    return !m.Deprecated
}
```

## tRPC Package (`contract/transport/trpc`)

### Overview

tRPC-like transport provides a simple HTTP-based RPC protocol with typed response envelopes.

### Types (`trpc.go`)

```go
package trpc

import "encoding/json"

// Envelope wraps all tRPC responses.
type Envelope struct {
    Result *Result `json:"result,omitempty"`
    Error  *Error  `json:"error,omitempty"`
}

// Result contains successful response data.
type Result struct {
    Data any `json:"data"`
}

// Error contains error information.
type Error struct {
    Message string `json:"message"`
    Code    string `json:"code"`
    Data    any    `json:"data,omitempty"`
}

// Error codes
const (
    CodeBadRequest    = "BAD_REQUEST"
    CodeNotFound      = "NOT_FOUND"
    CodeInternalError = "INTERNAL_ERROR"
    CodeUnauthorized  = "UNAUTHORIZED"
    CodeForbidden     = "FORBIDDEN"
)

// SuccessEnvelope creates a success response.
func SuccessEnvelope(data any) Envelope {
    return Envelope{
        Result: &Result{Data: data},
    }
}

// ErrorEnvelope creates an error response.
func ErrorEnvelope(code, message string) Envelope {
    return Envelope{
        Error: &Error{Code: code, Message: message},
    }
}

// ProcedureMeta contains metadata about a procedure.
type ProcedureMeta struct {
    Name     string         `json:"name"`
    FullName string         `json:"fullName"`
    Proc     string         `json:"proc"`
    Input    *TypeRef       `json:"input,omitempty"`
    Output   *TypeRef       `json:"output,omitempty"`
}

// TypeRef is a reference to a type.
type TypeRef struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// ServiceMeta contains service introspection data.
type ServiceMeta struct {
    Service string           `json:"service"`
    Methods []ProcedureMeta  `json:"methods"`
    Schemas []json.RawMessage `json:"schemas"`
}
```

### Handler (`handler.go`)

```go
package trpc

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/go-mizu/mizu/contract"
)

// Handler handles tRPC requests over HTTP.
type Handler struct {
    service  *contract.Service
    resolver contract.Resolver
    invoker  contract.TransportInvoker
    basePath string
}

// Option configures the handler.
type Option func(*Handler)

// WithResolver sets a custom method resolver.
func WithResolver(r contract.Resolver) Option {
    return func(h *Handler) { h.resolver = r }
}

// WithInvoker sets a custom method invoker.
func WithInvoker(i contract.TransportInvoker) Option {
    return func(h *Handler) { h.invoker = i }
}

// NewHandler creates a new tRPC handler.
func NewHandler(basePath string, svc *contract.Service, opts ...Option) *Handler {
    h := &Handler{
        service:  svc,
        basePath: basePath,
        resolver: &contract.ServiceResolver{Service: svc},
        invoker:  &contract.DefaultInvoker{},
    }
    for _, opt := range opts {
        opt(h)
    }
    return h
}

// Name returns the transport name.
func (h *Handler) Name() string {
    return "trpc"
}

// MetaHandler returns a handler for the .meta endpoint.
func (h *Handler) MetaHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }

        meta := h.buildMeta()
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(meta)
    }
}

// CallHandler returns a handler for procedure calls.
func (h *Handler) CallHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }

        proc := strings.TrimPrefix(r.URL.Path, h.basePath+"/")
        proc = strings.TrimSpace(proc)

        if proc == "" {
            h.writeError(w, http.StatusBadRequest, "missing procedure")
            return
        }

        method := h.resolver.Resolve(proc)
        if method == nil {
            h.writeError(w, http.StatusBadRequest, "unknown procedure")
            return
        }

        h.handleCall(w, r.Context(), method, r)
    }
}

func (h *Handler) handleCall(w http.ResponseWriter, ctx context.Context, m *contract.Method, r *http.Request) {
    var raw json.RawMessage
    if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
        // EOF is acceptable (empty body)
        if !isEOFError(err) {
            h.writeError(w, http.StatusBadRequest, err.Error())
            return
        }
    }

    result, err := h.invoker.Invoke(ctx, m, raw)
    if err != nil {
        // Application errors return 200 with error envelope
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(ErrorEnvelope(CodeInternalError, err.Error()))
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(SuccessEnvelope(result))
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(ErrorEnvelope(CodeBadRequest, msg))
}

func (h *Handler) buildMeta() ServiceMeta {
    methods := make([]ProcedureMeta, 0, len(h.service.Methods))
    for _, m := range h.service.Methods {
        pm := ProcedureMeta{
            Name:     m.Name,
            FullName: m.FullName,
            Proc:     h.service.Name + "." + m.Name,
        }
        if m.Input != nil {
            pm.Input = &TypeRef{ID: m.Input.ID, Name: m.Input.Name}
        }
        if m.Output != nil {
            pm.Output = &TypeRef{ID: m.Output.ID, Name: m.Output.Name}
        }
        methods = append(methods, pm)
    }

    schemas := make([]json.RawMessage, 0)
    for _, s := range h.service.Types.Schemas() {
        b, _ := json.Marshal(s)
        schemas = append(schemas, b)
    }

    return ServiceMeta{
        Service: h.service.Name,
        Methods: methods,
        Schemas: schemas,
    }
}

// Mount registers tRPC handlers at the given base path.
func Mount(mux *http.ServeMux, base string, svc *contract.Service, opts ...Option) {
    if mux == nil || svc == nil {
        return
    }
    if base == "" {
        base = "/trpc"
    }
    if !strings.HasPrefix(base, "/") {
        base = "/" + base
    }
    base = strings.TrimRight(base, "/")

    h := NewHandler(base, svc, opts...)
    mux.HandleFunc(base+".meta", h.MetaHandler())
    mux.HandleFunc(base+"/", h.CallHandler())
}

func isEOFError(err error) bool {
    return err != nil && strings.Contains(err.Error(), "EOF")
}
```

## Backward Compatibility

### Deprecated Wrappers

Update existing files to forward to new packages:

```go
// contract/transport_mcp.go
package contract

import "github.com/go-mizu/mizu/contract/transport/mcp"

// MountMCP mounts an MCP endpoint.
// Deprecated: Use mcp.Mount instead.
func MountMCP(mux *http.ServeMux, path string, svc *Service) {
    mcp.Mount(mux, path, svc)
}
```

```go
// contract/transport_trpc.go
package contract

import "github.com/go-mizu/mizu/contract/transport/trpc"

// MountTRPC mounts a tRPC endpoint.
// Deprecated: Use trpc.Mount instead.
func MountTRPC(mux *http.ServeMux, base string, svc *Service) {
    trpc.Mount(mux, base, svc)
}
```

### Remove helpers.go

After migration, remove `contract/helpers.go` as its functions are replaced by `contract/transport.go`:

| Old Function | New Function |
|--------------|--------------|
| `jsonIsNull()` | `IsJSONNull()` |
| `jsonTrimSpace()` | `TrimJSONSpace()` |
| `jsonSafeErr()` | `SafeErrorString()` |

## Files to Remove After Migration

```
contract/
├── helpers.go              # Remove entirely
├── transport_jsonrpc.go    # Keep as deprecated wrapper
├── transport_rest.go       # Keep as deprecated wrapper (REST stays in main pkg)
├── transport_mcp.go        # Replace with deprecated wrapper
└── transport_trpc.go       # Replace with deprecated wrapper
```

## Implementation Plan

### Phase 1: Create New Packages

1. Create `contract/transport/mcp/` directory
2. Create `contract/transport/trpc/` directory
3. Implement MCP handler following the spec above
4. Implement tRPC handler following the spec above
5. Add tests for both packages

### Phase 2: Update Main Package

1. Update `transport_mcp.go` to be a thin wrapper
2. Update `transport_trpc.go` to be a thin wrapper
3. Remove `helpers.go` (functions already in `transport.go`)
4. Update any internal usages

### Phase 3: Documentation

1. Add documentation page for MCP transport
2. Add documentation page for tRPC transport
3. Update contract overview to mention all transports
4. Add migration guide for users using deprecated APIs

## Testing Strategy

### Unit Tests

Each package should have comprehensive tests:

```go
// transport/mcp/mcp_test.go
func TestHandler_Initialize(t *testing.T)
func TestHandler_ToolsList(t *testing.T)
func TestHandler_ToolsCall(t *testing.T)
func TestHandler_Notification(t *testing.T)
func TestHandler_OriginValidation(t *testing.T)
func TestProtocolNegotiation(t *testing.T)

// transport/trpc/trpc_test.go
func TestHandler_Call(t *testing.T)
func TestHandler_Meta(t *testing.T)
func TestHandler_MethodResolution(t *testing.T)
func TestEnvelope_Success(t *testing.T)
func TestEnvelope_Error(t *testing.T)
```

### Integration Tests

```go
// contract_test.go
func TestMCP_Integration(t *testing.T)
func TestTRPC_Integration(t *testing.T)
func TestAllTransports_SameService(t *testing.T)
```

## Success Criteria

- [ ] All existing tests pass
- [ ] New packages have >90% test coverage
- [ ] No breaking changes to public APIs
- [ ] Deprecated functions have documentation comments
- [ ] Documentation updated for new APIs
- [ ] Clean package boundaries (no import cycles)
- [ ] Follows Go stdlib naming conventions
