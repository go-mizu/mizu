# Transport REST and OpenRPC Specification

## Overview

This document specifies the refactoring of transport packages to provide:

1. **transport/rest**: Full REST handler + OpenAPI spec generation (renamed from transport/openapi)
2. **transport/jsonrpc/openrpc**: OpenRPC 1.3 spec generation for JSON-RPC 2.0 services

## Goals

1. **Actual REST Handler**: `transport/rest` handles HTTP requests with full OpenAPI compatibility
2. **OpenAPI Spec Serving**: Same package generates and serves OpenAPI 3.1 documents
3. **OpenRPC for JSON-RPC**: Generate OpenRPC specs from contract services
4. **Clean Package Structure**: Remove deprecated files from contract package root

## Package Structure

```
contract/
├── contract.go           # Core: Service, Method, Register
├── schema.go             # TypeRegistry, schema generation
├── errors.go             # Error types and codes
├── transport.go          # Transport interfaces (unchanged)
├── transport/
│   ├── rest/             # (renamed from openapi)
│   │   ├── spec.go       # OpenAPI 3.1 document generation
│   │   ├── handler.go    # REST HTTP handler (new)
│   │   └── rest_test.go
│   └── jsonrpc/
│       ├── jsonrpc.go    # JSON-RPC 2.0 codec and types
│       ├── handler.go    # HTTP handler implementation
│       ├── openrpc.go    # OpenRPC 1.3 spec generation (new)
│       └── jsonrpc_test.go
```

## Files to Delete

Remove from `contract/`:
- `openapi.go` - Moved to transport/rest
- `transport_jsonrpc.go` - Moved to transport/jsonrpc
- `transport_rest.go` - Moved to transport/rest

## REST Package (`contract/transport/rest`)

### REST Handler

The REST handler serves HTTP requests following RESTful conventions, 100% compatible with the OpenAPI spec it generates.

#### Routing Conventions

| Method Prefix | HTTP Verb | Path Pattern | Example |
|---------------|-----------|--------------|---------|
| `Create*` | POST | `/{resource}` | `CreateTodo` -> `POST /todos` |
| `Get*` | GET | `/{resource}/{id}` | `GetTodo` -> `GET /todos/{id}` |
| `List*` | GET | `/{resource}` | `ListTodos` -> `GET /todos` |
| `Update*` | PUT | `/{resource}/{id}` | `UpdateTodo` -> `PUT /todos/{id}` |
| `Delete*` | DELETE | `/{resource}/{id}` | `DeleteTodo` -> `DELETE /todos/{id}` |
| `Patch*` | PATCH | `/{resource}/{id}` | `PatchTodo` -> `PATCH /todos/{id}` |
| Other | POST | `/{resource}/{method}` | `Archive` -> `POST /todos/archive` |

#### Custom Routes via MethodMeta

Methods can override default routing via `MethodOptions`:

```go
func (s *TodoService) ContractMeta() map[string]contract.MethodOptions {
    return map[string]contract.MethodOptions{
        "Search": {
            HTTPMethod: "GET",
            HTTPPath:   "/todos/search",
        },
        "BulkCreate": {
            HTTPMethod: "POST",
            HTTPPath:   "/todos/bulk",
        },
    }
}
```

#### Request Handling

**Path Parameters**:
- Extracted from URL path (e.g., `/todos/{id}` -> `id` field in input)
- Mapped to input struct field by name

**Query Parameters** (GET/DELETE):
- Parsed from query string for methods without request body
- Mapped to input struct fields by json tag name

**Request Body** (POST/PUT/PATCH):
- JSON decoded into input struct
- Content-Type: `application/json`

#### Response Handling

| Scenario | Status Code | Response Body |
|----------|-------------|---------------|
| Success with output | 200 OK | JSON encoded output |
| Success without output | 204 No Content | Empty |
| Validation error | 400 Bad Request | Error JSON |
| Not found | 404 Not Found | Error JSON |
| Internal error | 500 Internal Server Error | Error JSON |

**Error Response Format**:
```json
{
  "code": "INVALID_ARGUMENT",
  "message": "title is required",
  "details": { "field": "title" }
}
```

### Handler Implementation

```go
package rest

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/go-mizu/mizu/contract"
)

// Handler serves REST endpoints for a contract service.
type Handler struct {
    service  *contract.Service
    routes   map[string]map[string]*route // path -> method -> route
    basePath string
}

type route struct {
    method    *contract.Method
    pathVars  []string
}

// Option configures the handler.
type Option func(*Handler)

// WithBasePath sets a custom base path prefix.
func WithBasePath(path string) Option {
    return func(h *Handler) { h.basePath = path }
}

// NewHandler creates a REST handler for the service.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
    h := &Handler{
        service:  svc,
        routes:   make(map[string]map[string]*route),
        basePath: "/" + pluralize(svc.Name),
    }
    for _, opt := range opts {
        opt(h)
    }
    h.buildRoutes()
    return h
}

func (h *Handler) buildRoutes() {
    for _, m := range h.service.Methods {
        httpMethod := m.HTTPMethod
        if httpMethod == "" {
            httpMethod = inferHTTPMethod(m.Name)
        }

        path := m.HTTPPath
        if path == "" {
            path = h.inferPath(m)
        }

        if h.routes[path] == nil {
            h.routes[path] = make(map[string]*route)
        }
        h.routes[path][httpMethod] = &route{
            method:   m,
            pathVars: extractPathVars(path),
        }
    }
}

func (h *Handler) Name() string { return "rest" }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Find matching route
    route, pathValues := h.matchRoute(r.URL.Path, r.Method)
    if route == nil {
        h.writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
        return
    }

    // Build input
    in, err := h.buildInput(r, route, pathValues)
    if err != nil {
        h.writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
        return
    }

    // Invoke method
    out, err := route.method.Invoker.Call(r.Context(), in)
    if err != nil {
        h.handleError(w, err)
        return
    }

    // Write response
    if out == nil {
        w.WriteHeader(http.StatusNoContent)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(out)
}

// Mount registers the REST handler at the service's base path.
func Mount(mux *http.ServeMux, svc *contract.Service, opts ...Option) {
    h := NewHandler(svc, opts...)
    // Register each unique path
    for path := range h.routes {
        mux.Handle(path, h)
    }
}
```

### OpenAPI Spec Serving

The same package serves OpenAPI documentation:

```go
// SpecHandler serves the OpenAPI specification.
type SpecHandler struct {
    document *Document
    json     []byte
}

// NewSpecHandler creates an OpenAPI spec handler.
func NewSpecHandler(services ...*contract.Service) (*SpecHandler, error) {
    doc := Generate(services...)
    data, err := doc.MarshalIndent()
    if err != nil {
        return nil, err
    }
    return &SpecHandler{document: doc, json: data}, nil
}

func (h *SpecHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(h.json)
}

// MountWithSpec registers both REST handler and OpenAPI spec.
func MountWithSpec(mux *http.ServeMux, specPath string, svc *contract.Service, opts ...Option) error {
    Mount(mux, svc, opts...)

    h, err := NewSpecHandler(svc)
    if err != nil {
        return err
    }
    mux.Handle(specPath, h)
    return nil
}
```

## OpenRPC Package (`contract/transport/jsonrpc/openrpc`)

### OpenRPC 1.3 Specification

OpenRPC is the OpenAPI equivalent for JSON-RPC 2.0 services.

#### Document Structure

```go
package jsonrpc

// OpenRPCDocument represents an OpenRPC 1.3 document.
type OpenRPCDocument struct {
    OpenRPC      string           `json:"openrpc"`
    Info         *OpenRPCInfo     `json:"info"`
    Servers      []*OpenRPCServer `json:"servers,omitempty"`
    Methods      []*OpenRPCMethod `json:"methods"`
    Components   *OpenRPCComponents `json:"components,omitempty"`
}

// OpenRPCInfo provides metadata about the API.
type OpenRPCInfo struct {
    Title          string          `json:"title"`
    Description    string          `json:"description,omitempty"`
    TermsOfService string          `json:"termsOfService,omitempty"`
    Contact        *OpenRPCContact `json:"contact,omitempty"`
    License        *OpenRPCLicense `json:"license,omitempty"`
    Version        string          `json:"version"`
}

// OpenRPCServer describes a server.
type OpenRPCServer struct {
    Name        string `json:"name,omitempty"`
    URL         string `json:"url"`
    Summary     string `json:"summary,omitempty"`
    Description string `json:"description,omitempty"`
}

// OpenRPCMethod describes a JSON-RPC method.
type OpenRPCMethod struct {
    Name        string                `json:"name"`
    Summary     string                `json:"summary,omitempty"`
    Description string                `json:"description,omitempty"`
    Tags        []*OpenRPCTag         `json:"tags,omitempty"`
    Params      []*OpenRPCContentDesc `json:"params"`
    Result      *OpenRPCContentDesc   `json:"result,omitempty"`
    Deprecated  bool                  `json:"deprecated,omitempty"`
    Errors      []*OpenRPCError       `json:"errors,omitempty"`
    Examples    []*OpenRPCExample     `json:"examples,omitempty"`
}

// OpenRPCContentDesc describes a parameter or result.
type OpenRPCContentDesc struct {
    Name        string        `json:"name"`
    Summary     string        `json:"summary,omitempty"`
    Description string        `json:"description,omitempty"`
    Required    bool          `json:"required,omitempty"`
    Schema      *SchemaRef    `json:"schema"`
}

// OpenRPCError describes a possible error.
type OpenRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

// OpenRPCComponents holds reusable objects.
type OpenRPCComponents struct {
    ContentDescriptors map[string]*OpenRPCContentDesc `json:"contentDescriptors,omitempty"`
    Schemas            map[string]*Schema             `json:"schemas,omitempty"`
    Errors             map[string]*OpenRPCError       `json:"errors,omitempty"`
    Tags               map[string]*OpenRPCTag         `json:"tags,omitempty"`
}
```

### Generation Function

```go
// OpenRPCVersion is the OpenRPC specification version.
const OpenRPCVersion = "1.3.0"

// GenerateOpenRPC creates an OpenRPC document from a service.
func GenerateOpenRPC(svc *contract.Service) *OpenRPCDocument {
    doc := &OpenRPCDocument{
        OpenRPC: OpenRPCVersion,
        Info: &OpenRPCInfo{
            Title:   svc.Name + " API",
            Version: "1.0.0",
        },
        Components: &OpenRPCComponents{
            Schemas: make(map[string]*Schema),
        },
    }

    if svc.Description != "" {
        doc.Info.Description = svc.Description
    }
    if svc.Version != "" {
        doc.Info.Version = svc.Version
    }

    // Add schemas
    for _, schema := range svc.Types.Schemas() {
        doc.Components.Schemas[schema.ID] = convertSchema(schema.JSON)
    }

    // Add methods
    for _, m := range svc.Methods {
        doc.Methods = append(doc.Methods, convertMethod(svc, m))
    }

    return doc
}

func convertMethod(svc *contract.Service, m *contract.Method) *OpenRPCMethod {
    method := &OpenRPCMethod{
        Name:        svc.Name + "." + m.Name,
        Summary:     m.Summary,
        Description: m.Description,
        Deprecated:  m.Deprecated,
        Params:      []*OpenRPCContentDesc{},
    }

    // Add tags
    for _, tag := range m.Tags {
        method.Tags = append(method.Tags, &OpenRPCTag{Name: tag})
    }

    // Add params (single object param for named params)
    if m.Input != nil {
        method.Params = append(method.Params, &OpenRPCContentDesc{
            Name:     "params",
            Required: true,
            Schema:   &SchemaRef{Ref: "#/components/schemas/" + m.Input.ID},
        })
    }

    // Add result
    if m.Output != nil {
        method.Result = &OpenRPCContentDesc{
            Name:   "result",
            Schema: &SchemaRef{Ref: "#/components/schemas/" + m.Output.ID},
        }
    }

    // Add standard errors
    method.Errors = standardOpenRPCErrors()

    return method
}

func standardOpenRPCErrors() []*OpenRPCError {
    return []*OpenRPCError{
        {Code: -32700, Message: "Parse error"},
        {Code: -32600, Message: "Invalid Request"},
        {Code: -32601, Message: "Method not found"},
        {Code: -32602, Message: "Invalid params"},
        {Code: -32603, Message: "Internal error"},
    }
}
```

### OpenRPC Handler

```go
// OpenRPCHandler serves OpenRPC documents over HTTP.
type OpenRPCHandler struct {
    document *OpenRPCDocument
    json     []byte
}

// NewOpenRPCHandler creates an OpenRPC handler.
func NewOpenRPCHandler(svc *contract.Service) (*OpenRPCHandler, error) {
    doc := GenerateOpenRPC(svc)
    data, err := json.MarshalIndent(doc, "", "  ")
    if err != nil {
        return nil, err
    }
    return &OpenRPCHandler{document: doc, json: data}, nil
}

func (h *OpenRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(h.json)
}

// MountOpenRPC registers the OpenRPC handler at the given path.
func MountOpenRPC(mux *http.ServeMux, path string, svc *contract.Service) error {
    if path == "" {
        path = "/openrpc.json"
    }
    h, err := NewOpenRPCHandler(svc)
    if err != nil {
        return err
    }
    mux.Handle(path, h)
    return nil
}

// MountWithOpenRPC registers JSON-RPC handler and OpenRPC spec.
func MountWithOpenRPC(mux *http.ServeMux, rpcPath, specPath string, svc *contract.Service, opts ...Option) error {
    Mount(mux, rpcPath, svc, opts...)
    return MountOpenRPC(mux, specPath, svc)
}
```

## Usage Examples

### REST with OpenAPI

```go
package main

import (
    "net/http"

    "github.com/go-mizu/mizu/contract"
    "github.com/go-mizu/mizu/contract/transport/rest"
)

func main() {
    svc, _ := contract.Register("todo", &TodoService{})
    mux := http.NewServeMux()

    // Mount REST endpoints + OpenAPI spec
    rest.MountWithSpec(mux, "/openapi.json", svc)

    // Endpoints:
    // POST   /todos           -> Create
    // GET    /todos/{id}      -> Get
    // GET    /todos           -> List
    // PUT    /todos/{id}      -> Update
    // DELETE /todos/{id}      -> Delete
    // GET    /openapi.json    -> OpenAPI spec

    http.ListenAndServe(":8080", mux)
}
```

### JSON-RPC with OpenRPC

```go
package main

import (
    "net/http"

    "github.com/go-mizu/mizu/contract"
    "github.com/go-mizu/mizu/contract/transport/jsonrpc"
)

func main() {
    svc, _ := contract.Register("todo", &TodoService{})
    mux := http.NewServeMux()

    // Mount JSON-RPC endpoint + OpenRPC spec
    jsonrpc.MountWithOpenRPC(mux, "/rpc", "/openrpc.json", svc)

    // Endpoints:
    // POST /rpc              -> JSON-RPC 2.0 handler
    // GET  /openrpc.json     -> OpenRPC spec

    http.ListenAndServe(":8080", mux)
}
```

### Combined REST + JSON-RPC

```go
func main() {
    svc, _ := contract.Register("todo", &TodoService{})
    mux := http.NewServeMux()

    // REST API
    rest.MountWithSpec(mux, "/api/openapi.json", svc,
        rest.WithBasePath("/api/v1/todos"))

    // JSON-RPC API
    jsonrpc.MountWithOpenRPC(mux, "/rpc", "/rpc/openrpc.json", svc)

    http.ListenAndServe(":8080", mux)
}
```

## Error Mapping

### REST Error Codes

| Contract Error Code | HTTP Status | Response Code |
|---------------------|-------------|---------------|
| `ErrCodeInvalidArgument` | 400 Bad Request | `INVALID_ARGUMENT` |
| `ErrCodeNotFound` | 404 Not Found | `NOT_FOUND` |
| `ErrCodeAlreadyExists` | 409 Conflict | `ALREADY_EXISTS` |
| `ErrCodePermissionDenied` | 403 Forbidden | `PERMISSION_DENIED` |
| `ErrCodeUnauthenticated` | 401 Unauthorized | `UNAUTHENTICATED` |
| `ErrCodeResourceExhausted` | 429 Too Many Requests | `RESOURCE_EXHAUSTED` |
| `ErrCodeFailedPrecondition` | 400 Bad Request | `FAILED_PRECONDITION` |
| `ErrCodeAborted` | 409 Conflict | `ABORTED` |
| `ErrCodeOutOfRange` | 400 Bad Request | `OUT_OF_RANGE` |
| `ErrCodeUnimplemented` | 501 Not Implemented | `UNIMPLEMENTED` |
| `ErrCodeInternal` | 500 Internal Server Error | `INTERNAL` |
| `ErrCodeUnavailable` | 503 Service Unavailable | `UNAVAILABLE` |
| `ErrCodeDataLoss` | 500 Internal Server Error | `DATA_LOSS` |
| `ErrCodeCanceled` | 499 Client Closed Request | `CANCELED` |
| `ErrCodeDeadlineExceeded` | 504 Gateway Timeout | `DEADLINE_EXCEEDED` |

### OpenRPC Error Codes

Uses standard JSON-RPC 2.0 error codes plus application-defined codes:

| Contract Error Code | JSON-RPC Code |
|---------------------|---------------|
| `ErrCodeInvalidArgument` | -32602 (Invalid params) |
| `ErrCodeNotFound` | -32601 (Method not found) |
| `ErrCodeInternal` | -32603 (Internal error) |
| `ErrCodeCanceled` | -32001 |
| `ErrCodeDeadlineExceeded` | -32002 |
| `ErrCodeAlreadyExists` | -32003 |
| `ErrCodePermissionDenied` | -32004 |
| Others | -32000 to -32099 range |

## Migration

### From Old API

```go
// Old (deprecated)
contract.MountREST(mux, svc)
contract.ServeOpenAPI(mux, "/openapi.json", svc)
contract.MountJSONRPC(mux, "/rpc", svc)

// New (preferred)
rest.MountWithSpec(mux, "/openapi.json", svc)
jsonrpc.MountWithOpenRPC(mux, "/rpc", "/openrpc.json", svc)
```

## Success Criteria

- [ ] `transport/rest` handles HTTP requests correctly
- [ ] REST routing matches OpenAPI spec exactly
- [ ] Path parameters extracted and mapped correctly
- [ ] Query parameters parsed for GET requests
- [ ] Error responses follow consistent format
- [ ] OpenRPC spec generated correctly for JSON-RPC services
- [ ] All existing tests pass
- [ ] New tests for REST handler and OpenRPC
