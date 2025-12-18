# Spec 0059: Go SDK Generator

## Overview

Generate typed Go SDK clients from `contract.Service` with OpenAI-style developer experience.

**Package**: `contract/v2/sdk/go` (package name: `sdkgo`)

## Design Goals

1. **OpenAI-style DX** - Resource-oriented client with fluent access
2. **Minimal public API** - Follow Go conventions (minimal exported surface)
3. **Rich features** - Types, clients, streaming, unions, options
4. **Self-contained output** - Generated code has zero dependencies except stdlib

## Generated Code Example

From the OpenAI contract, generate:

```go
package openai

// Client is the OpenAI API client.
type Client struct {
    Responses *ResponsesResource
    Models    *ModelsResource
    // internal
    baseURL string
    token   string
    http    *http.Client
}

// NewClient creates a new OpenAI client.
func NewClient(token string, opts ...Option) *Client {
    c := &Client{
        baseURL: "https://api.openai.com",
        token:   token,
        http:    http.DefaultClient,
    }
    for _, opt := range opts {
        opt(c)
    }
    c.Responses = &ResponsesResource{client: c}
    c.Models = &ModelsResource{client: c}
    return c
}

// Option configures the client.
type Option func(*Client)

func WithBaseURL(url string) Option { return func(c *Client) { c.baseURL = url } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// ResponsesResource handles response operations.
type ResponsesResource struct {
    client *Client
}

func (r *ResponsesResource) Create(ctx context.Context, in *ResponseCreateRequest) (*Response, error) {
    // POST /v1/responses with JSON body
}

func (r *ResponsesResource) Stream(ctx context.Context, in *ResponseCreateRequest) *EventStream[ResponseEvent] {
    // POST /v1/responses with SSE streaming
}

// ModelsResource handles model operations.
type ModelsResource struct {
    client *Client
}

func (r *ModelsResource) List(ctx context.Context) (*ModelList, error) {
    // GET /v1/models
}
```

## Generator API

```go
package sdkgo

// Config controls code generation.
type Config struct {
    Package string // Go package name (default: lowercase service name)
}

// Generate produces Go source code for a typed SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]byte, error)
```

## Type Mapping

### Primitives

| Contract Type | Go Type |
|---------------|---------|
| `string` | `string` |
| `bool` | `bool` |
| `int`, `int32` | `int32` |
| `int64` | `int64` |
| `int8`, `int16` | `int8`, `int16` |
| `uint`, `uint32` | `uint32` |
| `uint64` | `uint64` |
| `float32` | `float32` |
| `float64` | `float64` |
| `time.Time` | `time.Time` |
| `json.RawMessage` | `json.RawMessage` |
| `any` | `any` |

### Struct Types

```yaml
- name: Todo
  kind: struct
  fields:
    - name: id
      type: string
    - name: title
      type: string
    - name: done
      type: bool
      optional: true
```

Generates:

```go
type Todo struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Done  *bool  `json:"done,omitempty"` // optional → pointer + omitempty
}
```

### Slice Types

```yaml
- name: TodoList
  kind: slice
  elem: Todo
```

Generates:

```go
type TodoList []Todo
```

### Map Types

```yaml
- name: Metadata
  kind: map
  elem: string
```

Generates:

```go
type Metadata map[string]string
```

### Union Types (Discriminated)

```yaml
- name: ContentPart
  kind: union
  tag: type
  variants:
    - value: input_text
      type: ContentPartInputText
    - value: input_image
      type: ContentPartInputImage
```

Generates:

```go
// ContentPart is a discriminated union (tag: "type").
// Variants: ContentPartInputText, ContentPartInputImage
type ContentPart struct {
    InputText  *ContentPartInputText  `json:"-"`
    InputImage *ContentPartInputImage `json:"-"`
}

func (u *ContentPart) MarshalJSON() ([]byte, error) {
    if u.InputText != nil {
        return json.Marshal(u.InputText)
    }
    if u.InputImage != nil {
        return json.Marshal(u.InputImage)
    }
    return []byte("null"), nil
}

func (u *ContentPart) UnmarshalJSON(data []byte) error {
    var disc struct{ Type string `json:"type"` }
    if err := json.Unmarshal(data, &disc); err != nil {
        return err
    }
    switch disc.Type {
    case "input_text":
        u.InputText = new(ContentPartInputText)
        return json.Unmarshal(data, u.InputText)
    case "input_image":
        u.InputImage = new(ContentPartInputImage)
        return json.Unmarshal(data, u.InputImage)
    }
    return fmt.Errorf("unknown ContentPart type: %q", disc.Type)
}
```

### Optional and Nullable Fields

| Contract | Go Type | JSON Tag |
|----------|---------|----------|
| `optional: false, nullable: false` | `T` | `json:"name"` |
| `optional: true, nullable: false` | `*T` | `json:"name,omitempty"` |
| `optional: false, nullable: true` | `*T` | `json:"name"` |
| `optional: true, nullable: true` | `*T` | `json:"name,omitempty"` |

### Enum Fields

```yaml
- name: role
  type: string
  enum: [system, user, assistant]
```

Generates field with comment:

```go
// Role is one of: system, user, assistant
Role string `json:"role"`
```

### Const Fields

```yaml
- name: type
  type: string
  const: input_text
```

Generates:

```go
Type string `json:"type"` // always "input_text"
```

With automatic population in constructors.

## Client Generation

### Resource Pattern

Each resource becomes a struct attached to the client:

```go
type Client struct {
    Todos  *TodosResource
    Users  *UsersResource
    // ...
}
```

### Method Signatures

| Method Pattern | Generated Signature |
|----------------|---------------------|
| No input, no output | `Method(ctx) error` |
| No input, with output | `Method(ctx) (*Output, error)` |
| With input, no output | `Method(ctx, *Input) error` |
| With input, with output | `Method(ctx, *Input) (*Output, error)` |

### Streaming Methods

For methods with `stream` config:

```go
// Stream returns an event stream for server-sent events.
func (r *ResponsesResource) Stream(ctx context.Context, in *ResponseCreateRequest) *EventStream[ResponseEvent] {
    return r.client.stream(ctx, "POST", "/v1/responses", in)
}

// EventStream reads server-sent events.
type EventStream[T any] struct {
    // internal
}

func (s *EventStream[T]) Next() bool   // advance to next event
func (s *EventStream[T]) Event() T     // current event (call after Next)
func (s *EventStream[T]) Err() error   // error if Next returned false
func (s *EventStream[T]) Close() error // close underlying connection
```

Usage:

```go
stream := client.Responses.Stream(ctx, &ResponseCreateRequest{...})
defer stream.Close()

for stream.Next() {
    event := stream.Event()
    switch {
    case event.OutputTextDelta != nil:
        fmt.Print(event.OutputTextDelta.Delta)
    case event.ResponseCompleted != nil:
        // done
    }
}
if err := stream.Err(); err != nil {
    return err
}
```

### HTTP Implementation

```go
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
    // Build URL
    u := c.baseURL + path

    // Encode body for non-GET
    var body io.Reader
    if method != "GET" && in != nil {
        b, _ := json.Marshal(in)
        body = bytes.NewReader(b)
    }

    req, _ := http.NewRequestWithContext(ctx, method, u, body)
    req.Header.Set("Content-Type", "application/json")
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }

    resp, err := c.http.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return decodeError(resp)
    }

    if out != nil {
        return json.NewDecoder(resp.Body).Decode(out)
    }
    return nil
}
```

## Error Handling

Generate standard error type:

```go
// Error represents an API error response.
type Error struct {
    StatusCode int    `json:"-"`
    Code       string `json:"code,omitempty"`
    Message    string `json:"message"`
}

func (e *Error) Error() string {
    if e.Code != "" {
        return fmt.Sprintf("%s: %s", e.Code, e.Message)
    }
    return e.Message
}
```

## Generated File Structure

Single file output (`openai.go`):

```
// Code generated by sdkgo. DO NOT EDIT.
package openai

import (...)

// --- Types ---
type ResponseCreateRequest struct {...}
type Response struct {...}
// ...

// --- Unions ---
type ContentPart struct {...}
func (u *ContentPart) MarshalJSON() ([]byte, error) {...}
func (u *ContentPart) UnmarshalJSON(data []byte) error {...}

// --- Client ---
type Client struct {...}
func NewClient(token string, opts ...Option) *Client {...}
type Option func(*Client)
func WithBaseURL(url string) Option {...}
func WithHTTPClient(h *http.Client) Option {...}

// --- Resources ---
type ResponsesResource struct {...}
func (r *ResponsesResource) Create(ctx context.Context, in *ResponseCreateRequest) (*Response, error) {...}
func (r *ResponsesResource) Stream(ctx context.Context, in *ResponseCreateRequest) *EventStream[ResponseEvent] {...}

// --- Streaming ---
type EventStream[T any] struct {...}
func (s *EventStream[T]) Next() bool {...}
func (s *EventStream[T]) Event() T {...}
func (s *EventStream[T]) Err() error {...}
func (s *EventStream[T]) Close() error {...}

// --- Errors ---
type Error struct {...}
```

## Implementation Details

### Internal Types

```go
package sdkgo

// Generator holds state during code generation.
type generator struct {
    svc    *contract.Service
    pkg    string
    buf    *bytes.Buffer
    types  map[string]*contract.Type // lookup
    indent int
}

func (g *generator) emit(format string, args ...any)  // write formatted line
func (g *generator) emitTypes()                       // all type definitions
func (g *generator) emitUnions()                      // union marshal/unmarshal
func (g *generator) emitClient()                      // client struct + new
func (g *generator) emitResources()                   // resource structs + methods
func (g *generator) emitStreaming()                   // EventStream generic
func (g *generator) emitHelpers()                     // do(), stream(), error
```

### Naming Conventions

- Type names: Use contract type name as-is (already PascalCase)
- Field names: Convert JSON snake_case to Go PascalCase
- Resource names: PascalCase + "Resource" suffix
- Method names: PascalCase from contract method name

### Import Management

Track imports and emit only used ones:

```go
func (g *generator) useImport(pkg string)
func (g *generator) emitImports()
```

Standard imports used:
- `bytes` - request body
- `context` - context.Context
- `encoding/json` - marshal/unmarshal
- `fmt` - error formatting
- `io` - io.Reader
- `net/http` - HTTP client
- `time` - time.Time (if used)

## Testing Strategy

### Type Generation Tests

```go
func TestGenerateStruct(t *testing.T) {
    svc := &contract.Service{
        Types: []*contract.Type{{
            Name: "Todo",
            Kind: contract.KindStruct,
            Fields: []contract.Field{
                {Name: "id", Type: "string"},
                {Name: "title", Type: "string"},
            },
        }},
    }
    code, _ := Generate(svc, nil)
    // verify struct definition
}
```

### Union Tests

```go
func TestGenerateUnion(t *testing.T) {
    // verify MarshalJSON/UnmarshalJSON generation
}
```

### Full SDK Tests

```go
func TestGenerateOpenAI(t *testing.T) {
    svc := loadYAML("samples/openai/api.yaml")
    code, err := Generate(svc, nil)
    // verify compiles: go/parser.ParseFile
    // verify key structures present
}
```

### Compile Verification

```go
func TestGeneratedCodeCompiles(t *testing.T) {
    code, _ := Generate(svc, nil)
    fset := token.NewFileSet()
    _, err := parser.ParseFile(fset, "generated.go", code, parser.AllErrors)
    if err != nil {
        t.Fatalf("generated code doesn't compile: %v", err)
    }
}
```

## Future Extensions

Not in scope for v1, but design allows:

- **Multiple files**: Split types.go, client.go, resources.go
- **Request builders**: Fluent builder pattern for complex inputs
- **Pagination**: Iterator pattern for list methods
- **Retry/backoff**: Configurable retry policies
- **Middleware**: Request/response interceptors
- **Mocking**: Interface extraction for testing

## Success Criteria

- [ ] `Generate(svc, cfg)` produces valid Go code
- [ ] Generated code compiles with `go build`
- [ ] All contract.Type kinds supported (struct, slice, map, union)
- [ ] Client with resource pattern works
- [ ] Streaming methods produce EventStream
- [ ] Union types marshal/unmarshal correctly
- [ ] OpenAI sample generates complete SDK
