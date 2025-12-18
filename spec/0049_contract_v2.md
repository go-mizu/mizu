# Contract v2 Architecture Specification

**Status:** Design Complete
**Package:** `contract/v2`

## Overview

Contract v2 is a transport-neutral API contract system designed for SDK-first code generation. Unlike v1's reflection-based approach (Go code → Contract), v2 uses a definition-first approach (YAML → Contract → Code).

## Design Principles

1. **Transport-neutral**: Single contract definition works for REST, JSON-RPC, SSE, WebSocket, async messaging, and gRPC
2. **SDK-first**: Generates OpenAI/Stripe style clients (`client.responses.create()`)
3. **Resource-oriented**: Methods grouped into resources/namespaces
4. **Streaming is first-class**: Explicit stream semantics (SSE, WS, bidirectional)
5. **Minimal surface area**: Small, focused API
6. **No pointer syntax in schema**: Clean YAML definitions
7. **Map keys are always string**: Simplified type system

## Core Data Model

### Service

The root descriptor for an API:

```go
type Service struct {
    Name        string      `yaml:"name"`
    Description string      `yaml:"description,omitempty"`
    Defaults    *Defaults   `yaml:"defaults,omitempty"`
    Resources   []*Resource `yaml:"resources"`
    Types       []*Type     `yaml:"types,omitempty"`
}
```

### Defaults

Global configuration hints for transports and SDK generators:

```go
type Defaults struct {
    BaseURL string            `yaml:"base_url,omitempty"`
    Auth    string            `yaml:"auth,omitempty"`      // "bearer", etc.
    Headers map[string]string `yaml:"headers,omitempty"`
}
```

### Resource

Groups related methods into a namespace:

```go
type Resource struct {
    Name        string    `yaml:"name"`
    Description string    `yaml:"description,omitempty"`
    Methods     []*Method `yaml:"methods"`
}
```

### Method

A single operation with optional streaming and HTTP binding:

```go
type Method struct {
    Name        string  `yaml:"name"`
    Description string  `yaml:"description,omitempty"`

    // Unary I/O
    Input  TypeRef `yaml:"input,omitempty"`
    Output TypeRef `yaml:"output,omitempty"`

    // Streaming (optional)
    Stream *struct {
        Mode      string  `yaml:"mode,omitempty"`       // "sse", "ws", "grpc", "async"
        Item      TypeRef `yaml:"item"`                 // Server → Client message
        Done      TypeRef `yaml:"done,omitempty"`       // Terminal message
        Error     TypeRef `yaml:"error,omitempty"`      // Typed stream error
        InputItem TypeRef `yaml:"input_item,omitempty"` // Client → Server (bidirectional)
    } `yaml:"stream,omitempty"`

    // HTTP binding (optional)
    HTTP *MethodHTTP `yaml:"http,omitempty"`
}

type MethodHTTP struct {
    Method string `yaml:"method"` // GET, POST, etc.
    Path   string `yaml:"path"`   // /v1/responses, /users/{id}
}
```

## Type System

### TypeRef

References a type by name. If it matches a declared Type.Name, it refers to that schema. Otherwise it's a primitive or external type:

```go
type TypeRef string
```

Primitives: `string`, `int`, `int64`, `bool`, `float64`, `time.Time`, `json.RawMessage`

### TypeKind

```go
const (
    KindStruct TypeKind = "struct"
    KindSlice  TypeKind = "slice"
    KindMap    TypeKind = "map"
    KindUnion  TypeKind = "union"  // Discriminated union
)
```

### Type

Schema definition with support for structs, slices, maps, and discriminated unions:

```go
type Type struct {
    Name        string   `yaml:"name"`
    Description string   `yaml:"description,omitempty"`
    Kind        TypeKind `yaml:"kind"`

    // Struct
    Fields []Field `yaml:"fields,omitempty"`

    // Slice/Map
    Elem TypeRef `yaml:"elem,omitempty"` // Slice element or map value

    // Union (discriminated)
    Tag      string    `yaml:"tag,omitempty"`      // Discriminator field ("type")
    Variants []Variant `yaml:"variants,omitempty"`
}

type Variant struct {
    Value       string  `yaml:"value"`       // Discriminator value
    Type        TypeRef `yaml:"type"`        // Referenced struct
    Description string  `yaml:"description,omitempty"`
}

type Field struct {
    Name        string  `yaml:"name"`
    Description string  `yaml:"description,omitempty"`
    Type        TypeRef `yaml:"type"`
    Optional    bool    `yaml:"optional,omitempty"`
    Nullable    bool    `yaml:"nullable,omitempty"`
    Enum        []string `yaml:"enum,omitempty"`  // Allowed values
    Const       string  `yaml:"const,omitempty"` // Fixed value (discriminators)
}
```

## Runtime Interfaces

### Descriptor

```go
type Descriptor interface {
    Descriptor() *Service
}
```

### Invoker

Unified runtime surface for all transports:

```go
type Invoker interface {
    Descriptor

    // Unary
    Call(ctx context.Context, resource, method string, in any) (any, error)
    NewInput(resource, method string) (any, error)

    // Streaming
    Stream(ctx context.Context, resource, method string, in any) (Stream, error)
}
```

### Stream

Live stream interface (SSE for Recv-only, WebSocket for bidirectional):

```go
type Stream interface {
    Recv() (any, error)
    Send(any) error
    Close() error
}
```

## Transport Implementations

### REST (`transport/rest`)

| Component | Purpose |
|-----------|---------|
| `Server` | HTTP server using contract descriptor as routing table |
| `Client` | Generic HTTP client for contract-defined APIs |
| `OpenAPIDocument()` | Generates OpenAPI 3.0.3 specification |

**Server Features:**
- Routes from `Method.HTTP` bindings
- Path params from `{name}` segments
- Query params for GET
- JSON body for non-GET
- Standard error JSON responses

**Client Features:**
- Path parameter substitution
- Query param encoding
- Bearer token support
- Custom headers

### JSON-RPC (`transport/jsonrpc`)

| Component | Purpose |
|-----------|---------|
| `Server` | JSON-RPC 2.0 server over HTTP |
| `Client` | JSON-RPC 2.0 client with batch support |
| `OpenRPCDocument()` | Generates OpenRPC specification |

**Method Naming:** `<resource>.<method>` (e.g., `models.list`, `responses.create`)

**Features:**
- Object params only (no arrays)
- Batch requests
- Notifications (no response)

### Async (`transport/async`)

| Component | Purpose |
|-----------|---------|
| `Server` | Message broker server (NATS, Redis, Kafka, etc.) |
| `Client` | Request-reply and notification client |
| `AsyncAPIDocument()` | Generates AsyncAPI 2.6.0 specification |

**Broker Interface:**
```go
type Broker interface {
    Publish(ctx context.Context, topic string, payload []byte) error
    Subscribe(ctx context.Context, topic string, handler func([]byte)) (unsubscribe func(), err error)
}
```

**Topic Naming:** `<service>.<resource>.<method>.request` / `.response`

**Wire Format:**
```go
type Envelope struct {
    ID      string          `json:"id,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    ReplyTo string          `json:"reply_to,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
}
```

## API Definition Format (YAML)

Example API definition:

```yaml
name: GitHub
description: GitHub REST API
defaults:
  base_url: https://api.github.com
  auth: bearer
  headers:
    accept: application/vnd.github+json

resources:
  - name: repos
    description: Repositories
    methods:
      - name: get
        description: Get a repository
        input: ReposGetRequest
        output: Repo
        http:
          method: GET
          path: /repos/{owner}/{repo}

types:
  - name: ReposGetRequest
    kind: struct
    fields:
      - name: owner
        type: string
      - name: repo
        type: string

  - name: Repo
    kind: struct
    fields:
      - name: id
        type: int64
      - name: name
        type: string
      - name: description
        type: string
        nullable: true
```

## Discriminated Unions

v2 supports discriminated unions for polymorphic types:

```yaml
- name: ContentPart
  kind: union
  tag: type
  variants:
    - value: input_text
      type: ContentPartInputText
    - value: input_image
      type: ContentPartInputImage

- name: ContentPartInputText
  kind: struct
  fields:
    - name: type
      type: string
      const: input_text
    - name: text
      type: string
```

## Streaming Support

Methods can declare explicit streaming semantics:

```yaml
- name: stream
  description: Stream response events (SSE)
  input: ResponseCreateRequest
  output: Response
  stream:
    mode: sse
    item: ResponseEvent
  http:
    method: POST
    path: /v1/responses
```

Stream modes:
- `sse`: Server-Sent Events (server → client)
- `ws`: WebSocket (bidirectional)
- `grpc`: gRPC streaming
- `async`: Async message broker

## Comparison: v1 vs v2

| Aspect | v1 | v2 |
|--------|----|----|
| **Approach** | Code-first (reflection) | Definition-first (YAML) |
| **Type extraction** | Runtime reflection | Declarative schema |
| **Multi-language** | Go only | Code gen for any language |
| **Unions** | Not supported | First-class discriminated unions |
| **Streaming** | Not explicit | First-class stream semantics |
| **Async** | Not supported | Built-in async transport |
| **OpenAPI/OpenRPC** | Generated from runtime | Generated from definition |

## File Structure

```
contract/v2/
├── contract.go              # Core types and interfaces
├── spec/                    # Specification documents
├── samples/                 # Example API definitions
│   ├── openai/api.yaml
│   ├── github/api.yaml
│   ├── wordpress/api.yaml
│   └── petstore/api.yml
└── transport/
    ├── rest/
    │   ├── server.go        # HTTP server
    │   ├── client.go        # HTTP client
    │   └── openapi.go       # OpenAPI generator
    ├── jsonrpc/
    │   ├── server.go        # JSON-RPC server
    │   ├── client.go        # JSON-RPC client
    │   └── openrpc.go       # OpenRPC generator
    └── async/
        ├── server.go        # Async server
        ├── client.go        # Async client
        └── asyncapi.go      # AsyncAPI generator
```

## Usage Example

```go
// Load contract from YAML
data, _ := os.ReadFile("api.yaml")
var svc contract.Service
yaml.Unmarshal(data, &svc)

// Create REST client
client, _ := rest.NewClient(&svc)
client.BaseURL = "https://api.github.com"
client.Token = os.Getenv("GITHUB_TOKEN")

// Call method
var repo Repo
err := client.Call(ctx, "repos", "get", &ReposGetRequest{
    Owner: "go-mizu",
    Repo:  "mizu",
}, &repo)

// Generate OpenAPI spec
spec, _ := rest.OpenAPIDocument(&svc)
os.WriteFile("openapi.json", spec, 0644)
```

## Next Steps

1. CLI integration (spec 0050)
2. TypeScript code generator
3. Go code generator
4. gRPC transport
5. WebSocket streaming transport
