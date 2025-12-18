# Contract MCP v2: Model Context Protocol Transport

**Status:** Implementation Ready
**Depends on:** 0054_contract_jsonrpc_v2.md

## Overview

This specification defines `contract/v2/transport/mcp` - an MCP (Model Context Protocol) transport that exposes contract methods as MCP tools. MCP is built on JSON-RPC 2.0 and enables LLM applications to invoke contract methods as AI tools.

## Goals

1. **Native mizu integration** - Use `mizu.Handler` directly
2. **Minimal API surface** - Follow Go standard library design philosophy
3. **Consistent with other transports** - Same API pattern: `Mount`, `Handler`, options
4. **MCP compliant** - Follow MCP specification (2024-11-05 protocol version)
5. **Automatic tool generation** - Contract methods become MCP tools with JSON Schema

## MCP Protocol Summary

MCP uses JSON-RPC 2.0 with specific methods:

| Method | Purpose |
|--------|---------|
| `initialize` | Client/server capability negotiation |
| `notifications/initialized` | Client confirms initialization |
| `tools/list` | List available tools |
| `tools/call` | Invoke a tool |

### Tool Definition

Each contract method becomes an MCP tool:

```json
{
  "name": "todos_create",
  "description": "Create a new todo item",
  "inputSchema": {
    "type": "object",
    "properties": {
      "title": {"type": "string"},
      "completed": {"type": "boolean"}
    },
    "required": ["title"]
  }
}
```

### Tool Naming Convention

Tool names follow `{resource}_{method}` pattern:
- `todos.list` → `todos_list`
- `todos.create` → `todos_create`
- `users.get` → `users_get`

## Design

### Core API

```go
package mcp

// Mount registers an MCP endpoint on a mizu router.
// The endpoint accepts POST requests with MCP JSON-RPC payloads.
func Mount(r *mizu.Router, path string, inv contract.Invoker, opts ...Option) error

// Handler returns a mizu.Handler for MCP requests.
// This is the primary API when you need direct control.
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error)
```

### Options

```go
// Option configures MCP transport behavior.
type Option func(*options)

// WithServerInfo sets server name and version for capability negotiation.
func WithServerInfo(name, version string) Option

// WithErrorMapper sets custom error-to-MCP-error mapping.
func WithErrorMapper(m ErrorMapper) Option

// WithMaxBodySize limits request body size (default: 1MB).
func WithMaxBodySize(n int64) Option
```

### Server Info

```go
// ServerInfo identifies the MCP server in capability negotiation.
type ServerInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}
```

### Error Mapping

```go
// ErrorMapper converts Go errors to MCP tool errors.
// Returns: isError flag, error message
type ErrorMapper func(error) (isError bool, message string)

// Default: all errors set isError=true with err.Error() message
```

### Usage Examples

#### Simple Mount

```go
// Register service
svc := contract.Register[TodoAPI](impl)

// Mount MCP endpoint
r := mizu.NewRouter()
mcp.Mount(r, "/mcp", svc)
```

#### With Server Info

```go
mcp.Mount(r, "/mcp", svc,
    mcp.WithServerInfo("todo-server", "1.0.0"),
)
```

#### With Middleware

```go
r := mizu.NewRouter()

// Apply auth middleware to MCP endpoint
api := r.Prefix("/mcp").With(authMiddleware)
mcp.Mount(api, "", svc)
```

## Wire Protocol

### Initialize Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "example-client",
      "version": "1.0.0"
    }
  }
}
```

### Initialize Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "todo-server",
      "version": "1.0.0"
    }
  }
}
```

### Tools List Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

### Tools List Response

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "todos_list",
        "description": "List all todos",
        "inputSchema": {
          "type": "object",
          "properties": {}
        }
      },
      {
        "name": "todos_create",
        "description": "Create a todo",
        "inputSchema": {
          "type": "object",
          "properties": {
            "title": {"type": "string"},
            "completed": {"type": "boolean"}
          },
          "required": ["title"]
        }
      }
    ]
  }
}
```

### Tools Call Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "todos_create",
    "arguments": {
      "title": "Buy groceries",
      "completed": false
    }
  }
}
```

### Tools Call Response (Success)

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"id\":\"1\",\"title\":\"Buy groceries\",\"completed\":false}"
      }
    ],
    "isError": false
  }
}
```

### Tools Call Response (Error)

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "todo not found"
      }
    ],
    "isError": true
  }
}
```

## Implementation

### Wire Types

```go
// request is the JSON-RPC 2.0 request structure.
type request struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
    ID      any             `json:"id,omitempty"`
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
```

### MCP Types

```go
// initializeParams is the params for initialize request.
type initializeParams struct {
    ProtocolVersion string        `json:"protocolVersion"`
    Capabilities    any           `json:"capabilities"`
    ClientInfo      *clientInfo   `json:"clientInfo,omitempty"`
}

// clientInfo identifies the MCP client.
type clientInfo struct {
    Name    string `json:"name"`
    Version string `json:"version,omitempty"`
}

// initializeResult is the result for initialize response.
type initializeResult struct {
    ProtocolVersion string            `json:"protocolVersion"`
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
```

### Handler Factory

```go
func Handler(inv contract.Invoker, opts ...Option) (mizu.Handler, error) {
    if inv == nil {
        return nil, errors.New("mcp: nil invoker")
    }
    svc := inv.Descriptor()
    if svc == nil {
        return nil, errors.New("mcp: nil descriptor")
    }

    o := applyOptions(opts)
    tools := buildTools(svc)

    return func(c *mizu.Ctx) error {
        if c.Request().Method != http.MethodPost {
            c.Header().Set("Allow", "POST")
            return c.Status(http.StatusMethodNotAllowed).Text("method not allowed")
        }

        body, err := io.ReadAll(io.LimitReader(c.Request().Body, o.maxBodySize+1))
        if err != nil {
            return writeError(c, nil, errParse, "parse error", err.Error())
        }
        if int64(len(body)) > o.maxBodySize {
            return writeError(c, nil, errInvalidRequest, "request too large", nil)
        }

        var req request
        if err := json.Unmarshal(body, &req); err != nil {
            return writeError(c, nil, errParse, "parse error", err.Error())
        }

        switch req.Method {
        case "initialize":
            return handleInitialize(c, &req, o)
        case "notifications/initialized":
            return c.NoContent()
        case "tools/list":
            return handleToolsList(c, &req, tools)
        case "tools/call":
            return handleToolsCall(c, &req, inv, svc, o)
        default:
            return writeError(c, req.ID, errMethodNotFound, "method not found", nil)
        }
    }, nil
}
```

### Initialize Handler

```go
func handleInitialize(c *mizu.Ctx, req *request, o *options) error {
    result := initializeResult{
        ProtocolVersion: "2024-11-05",
        Capabilities: serverCapabilities{
            Tools: &toolsCapability{},
        },
    }
    if o.serverName != "" {
        result.ServerInfo = &serverInfo{
            Name:    o.serverName,
            Version: o.serverVersion,
        }
    }
    return writeResult(c, req.ID, result)
}
```

### Tools List Handler

```go
func handleToolsList(c *mizu.Ctx, req *request, tools []tool) error {
    return writeResult(c, req.ID, toolsListResult{Tools: tools})
}
```

### Tools Call Handler

```go
func handleToolsCall(c *mizu.Ctx, req *request, inv contract.Invoker, svc *contract.Service, o *options) error {
    var params toolsCallParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return writeError(c, req.ID, errInvalidParams, "invalid params", err.Error())
    }

    // Parse tool name: resource_method
    resource, method, ok := parseTool(params.Name)
    if !ok {
        return writeError(c, req.ID, errInvalidParams, "invalid tool name", nil)
    }

    // Find method in descriptor
    methodDesc := svc.Method(resource, method)
    if methodDesc == nil {
        return writeError(c, req.ID, errInvalidParams, "unknown tool", nil)
    }

    // Allocate input
    var in any
    if methodDesc.Input != "" {
        var err error
        in, err = inv.NewInput(resource, method)
        if err != nil {
            return writeError(c, req.ID, errInternal, "internal error", nil)
        }
    }

    // Decode arguments
    if in != nil && len(params.Arguments) > 0 && string(params.Arguments) != "null" {
        if err := json.Unmarshal(params.Arguments, in); err != nil {
            return writeError(c, req.ID, errInvalidParams, "invalid arguments", err.Error())
        }
    }

    // Invoke
    out, err := inv.Call(c.Context(), resource, method, in)
    if err != nil {
        isErr, msg := o.errorMapper(err)
        return writeResult(c, req.ID, toolsCallResult{
            Content: []content{{Type: "text", Text: msg}},
            IsError: isErr,
        })
    }

    // Marshal output
    var text string
    if out != nil {
        b, _ := json.Marshal(out)
        text = string(b)
    } else {
        text = "null"
    }

    return writeResult(c, req.ID, toolsCallResult{
        Content: []content{{Type: "text", Text: text}},
        IsError: false,
    })
}
```

### Tool Building

```go
func buildTools(svc *contract.Service) []tool {
    var tools []tool
    for _, res := range svc.Resources {
        if res == nil {
            continue
        }
        for _, m := range res.Methods {
            if m == nil {
                continue
            }
            t := tool{
                Name:        res.Name + "_" + m.Name,
                Description: m.Description,
                InputSchema: buildInputSchema(m, svc),
            }
            tools = append(tools, t)
        }
    }
    return tools
}

func buildInputSchema(m *contract.Method, svc *contract.Service) jsonSchema {
    schema := jsonSchema{Type: "object"}
    if m.Input == "" {
        return schema
    }

    // Find input type
    inputType := findType(svc, string(m.Input))
    if inputType == nil || inputType.Kind != contract.KindStruct {
        return schema
    }

    schema.Properties = make(map[string]jsonSchema)
    for _, f := range inputType.Fields {
        schema.Properties[f.Name] = typeRefToSchema(f.Type, svc)
        if !f.Optional {
            schema.Required = append(schema.Required, f.Name)
        }
    }
    return schema
}

func typeRefToSchema(ref contract.TypeRef, svc *contract.Service) jsonSchema {
    s := string(ref)
    switch s {
    case "string":
        return jsonSchema{Type: "string"}
    case "int":
        return jsonSchema{Type: "integer"}
    case "float":
        return jsonSchema{Type: "number"}
    case "bool":
        return jsonSchema{Type: "boolean"}
    case "any":
        return jsonSchema{Type: "object"}
    }
    if strings.HasPrefix(s, "[]") {
        return jsonSchema{
            Type:  "array",
            Items: ptr(typeRefToSchema(contract.TypeRef(s[2:]), svc)),
        }
    }
    // Nested struct
    t := findType(svc, s)
    if t != nil && t.Kind == contract.KindStruct {
        schema := jsonSchema{Type: "object", Properties: make(map[string]jsonSchema)}
        for _, f := range t.Fields {
            schema.Properties[f.Name] = typeRefToSchema(f.Type, svc)
            if !f.Optional {
                schema.Required = append(schema.Required, f.Name)
            }
        }
        return schema
    }
    return jsonSchema{Type: "object"}
}

func findType(svc *contract.Service, name string) *contract.Type {
    for _, t := range svc.Types {
        if t != nil && t.Name == name {
            return t
        }
    }
    return nil
}
```

## File Structure

```
contract/v2/transport/mcp/
├── mcp.go          # Mount, Handler (main API)
├── options.go      # Option types and defaults
├── wire.go         # Wire types (request, response, MCP types)
├── schema.go       # JSON Schema generation from contract types
└── mcp_test.go     # Comprehensive tests
```

## Tests

### Unit Tests

```go
func TestHandler(t *testing.T)              // Handler creation
func TestMount(t *testing.T)                // Mount on router
func TestInitialize(t *testing.T)           // Initialize handshake
func TestToolsList(t *testing.T)            // List tools
func TestToolsCall(t *testing.T)            // Call tool
func TestToolsCallError(t *testing.T)       // Tool execution error
func TestParseError(t *testing.T)           // Invalid JSON
func TestInvalidRequest(t *testing.T)       // Missing jsonrpc/method
func TestMethodNotFound(t *testing.T)       // Unknown MCP method
func TestInvalidParams(t *testing.T)        // Invalid tool params
func TestServerInfo(t *testing.T)           // Custom server info
func TestErrorMapper(t *testing.T)          // Custom error mapping
func TestMaxBodySize(t *testing.T)          // Body size limit
func TestToolNaming(t *testing.T)           // resource_method naming
func TestInputSchema(t *testing.T)          // JSON Schema generation
```

### Integration Tests

```go
func TestTodoAPI(t *testing.T)              // Full CRUD via MCP
func TestMultiResource(t *testing.T)        // Multiple resources as tools
func TestNoInput(t *testing.T)              // Tool without input
func TestNoOutput(t *testing.T)             // Tool without output
```

## CLI Template Changes

### Updated server.go.tmpl

```go
package server

import (
    "github.com/go-mizu/mizu"
    contract "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
    "github.com/go-mizu/mizu/contract/v2/transport/mcp"
    "github.com/go-mizu/mizu/contract/v2/transport/rest"
    "{{.Module}}/service/todo"
)

func New(cfg Config, todoSvc *todo.Service) (*mizu.App, error) {
    // Register service using code-first approach
    svc := contract.Register[todo.API](todoSvc,
        contract.WithDefaultResource("todos"),
        contract.WithName("Todo"),
        contract.WithDescription("Todo management service"),
    )

    // Create mizu app
    app := mizu.New()

    // Mount REST API
    if err := rest.Mount(app.Router, svc); err != nil {
        return nil, err
    }

    // Mount JSON-RPC 2.0 endpoint
    if err := jsonrpc.Mount(app.Router, "/rpc", svc); err != nil {
        return nil, err
    }

    // Mount MCP endpoint (for LLM tool integration)
    if err := mcp.Mount(app.Router, "/mcp", svc,
        mcp.WithServerInfo("{{.Name}}", "1.0.0"),
    ); err != nil {
        return nil, err
    }

    // Serve OpenAPI 3.0 spec
    app.Router.Get("/openapi.json", func(c *mizu.Ctx) error {
        spec, err := rest.OpenAPI(svc.Descriptor())
        if err != nil {
            return c.JSON(500, map[string]string{"error": err.Error()})
        }
        c.Header().Set("Content-Type", "application/json")
        _, err = c.Write(spec)
        return err
    })

    // Serve OpenRPC spec
    app.Router.Get("/openrpc.json", func(c *mizu.Ctx) error {
        spec, err := jsonrpc.OpenRPC(svc.Descriptor())
        if err != nil {
            return c.JSON(500, map[string]string{"error": err.Error()})
        }
        c.Header().Set("Content-Type", "application/json")
        _, err = c.Write(spec)
        return err
    })

    return app, nil
}
```

### Updated main.go.tmpl

```go
package main

import (
    "log"

    "{{.Module}}/app/server"
    "{{.Module}}/service/todo"
)

func main() {
    cfg := server.LoadConfig()

    // Create the plain Go service (no framework dependencies)
    todoSvc := &todo.Service{}

    // Create the mizu app with all transports
    app, err := server.New(cfg, todoSvc)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("listening on %s", cfg.Addr)
    log.Printf("REST:     http://localhost%s/todos", cfg.Addr)
    log.Printf("JSON-RPC: http://localhost%s/rpc", cfg.Addr)
    log.Printf("MCP:      http://localhost%s/mcp", cfg.Addr)
    log.Printf("OpenAPI:  http://localhost%s/openapi.json", cfg.Addr)

    if err := app.Listen(cfg.Addr); err != nil {
        log.Fatal(err)
    }
}
```

### Updated template.json

```json
{
  "name": "contract",
  "description": "Code-first API contract with mizu integration, REST, JSON-RPC, MCP, and OpenAPI (v2)",
  "tags": ["go", "mizu", "contract", "rest", "jsonrpc", "mcp", "openapi"],
  "variables": {
    "name": { "description": "Project name", "default": "" },
    "module": { "description": "Go module path", "default": "" }
  }
}
```

## Comparison with Other Transports

| Feature | REST | JSON-RPC | MCP |
|---------|------|----------|-----|
| Mount function | `Mount(r, inv)` | `Mount(r, path, inv)` | `Mount(r, path, inv)` |
| Handler function | `Handler(inv)` | `Handler(inv)` | `Handler(inv)` |
| Error mapper | `WithErrorMapper` | `WithErrorMapper` | `WithErrorMapper` |
| Body size limit | `WithMaxBodySize` | `WithMaxBodySize` | `WithMaxBodySize` |
| Server info | N/A | N/A | `WithServerInfo` |
| Spec generation | `OpenAPI(svc)` | `OpenRPC(svc)` | N/A |

## Summary

This specification defines:

1. **MCP transport** - Exposes contract methods as MCP tools
2. **Consistent API** - Same patterns as REST and JSON-RPC transports
3. **Automatic schemas** - Contract types become JSON Schema for tool inputs
4. **Minimal surface** - Mount, Handler, and three options
5. **LLM integration** - Any MCP-compatible client can invoke contract methods as AI tools

Sources:
- [MCP Specification](https://modelcontextprotocol.io/specification/2025-06-18)
- [MCP Tools](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP Lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)
