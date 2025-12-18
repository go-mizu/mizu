# Spec 0061: TypeScript SDK Generator

## Overview

Generate typed TypeScript SDK clients from `contract.Service` with best-in-class developer experience inspired by the OpenAI Node.js SDK.

**Package**: `contract/v2/sdk/ts` (package name: `sdkts`)

## Design Goals

1. **OpenAI-style DX** - Resource-oriented client with fluent access (`client.responses.create()`)
2. **Runtime agnostic** - Works on Node.js, Bun, and Deno without modifications
3. **TypeScript-first** - Full type inference, autocompletion, and compile-time safety
4. **Minimal dependencies** - Only fetch API (native in all modern runtimes)
5. **Streaming support** - First-class SSE streaming with async iterators
6. **Modern patterns** - ES modules, async/await, generics

## Generated Code Example

From the OpenAI contract, generate:

```typescript
// client.ts
export interface ClientOptions {
  apiKey?: string;
  baseURL?: string;
  timeout?: number;
  maxRetries?: number;
  defaultHeaders?: Record<string, string>;
}

export class OpenAI {
  readonly responses: ResponsesResource;
  readonly models: ModelsResource;

  private readonly _options: Required<ClientOptions>;

  constructor(options: ClientOptions = {}) {
    this._options = {
      apiKey: options.apiKey ?? "",
      baseURL: (options.baseURL ?? "https://api.openai.com").replace(/\/$/, ""),
      timeout: options.timeout ?? 60000,
      maxRetries: options.maxRetries ?? 2,
      defaultHeaders: options.defaultHeaders ?? {},
    };
    this.responses = new ResponsesResource(this);
    this.models = new ModelsResource(this);
  }

  // ... internal request method
}

// Usage
const client = new OpenAI({ apiKey: "sk-..." });
const response = await client.responses.create({ model: "gpt-4o", input: "Hello" });

// Streaming
for await (const event of client.responses.stream({ model: "gpt-4o", input: "Hello" })) {
  console.log(event);
}
```

## Generator API

```go
package sdkts

import (
    contract "github.com/go-mizu/mizu/contract/v2"
    "github.com/go-mizu/mizu/contract/v2/sdk"
)

// Config controls TypeScript SDK generation.
type Config struct {
    // Package is the npm package name.
    // Default: lowercase sanitized service name, or "sdk".
    Package string

    // Version is the package version for package.json.
    // Default: "0.0.0".
    Version string
}

// Generate produces a set of files for a TypeScript SDK.
// Output is an npm-compatible project with ES modules.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

## Type Mapping

### Primitives

| Contract Type | TypeScript Type |
|---------------|-----------------|
| `string` | `string` |
| `bool` | `boolean` |
| `int`, `int8`-`int64` | `number` |
| `uint`, `uint8`-`uint64` | `number` |
| `float32`, `float64` | `number` |
| `time.Time` | `Date` (ISO string on wire) |
| `json.RawMessage` | `unknown` |
| `any` | `unknown` |

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

```typescript
export interface Todo {
  id: string;
  title: string;
  done?: boolean;
}
```

### Slice Types

```yaml
- name: TodoList
  kind: slice
  elem: Todo
```

Generates:

```typescript
export type TodoList = Todo[];
```

### Map Types

```yaml
- name: Metadata
  kind: map
  elem: string
```

Generates:

```typescript
export type Metadata = Record<string, string>;
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

```typescript
export type ContentPart = ContentPartInputText | ContentPartInputImage;

// Type guard helpers
export function isContentPartInputText(v: ContentPart): v is ContentPartInputText {
  return v.type === "input_text";
}

export function isContentPartInputImage(v: ContentPart): v is ContentPartInputImage {
  return v.type === "input_image";
}
```

### Optional and Nullable Fields

| Contract | TypeScript Type |
|----------|-----------------|
| `optional: false, nullable: false` | `T` |
| `optional: true, nullable: false` | `T?` (optional property) |
| `optional: false, nullable: true` | `T \| null` |
| `optional: true, nullable: true` | `T \| null` (optional property) |

### Enum Fields

```yaml
- name: role
  type: string
  enum: [system, user, assistant]
```

Generates:

```typescript
/** One of: "system", "user", "assistant" */
role: "system" | "user" | "assistant";
```

### Const Fields

```yaml
- name: type
  type: string
  const: input_text
```

Generates:

```typescript
type: "input_text";
```

## Client Architecture

### Core Client Structure

```typescript
// _client.ts
export interface ClientOptions {
  apiKey?: string;
  baseURL?: string;
  timeout?: number;
  maxRetries?: number;
  defaultHeaders?: Record<string, string>;
}

interface RequestOptions {
  method: string;
  path: string;
  body?: unknown;
  headers?: Record<string, string>;
  stream?: boolean;
}

export class SDKError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "SDKError";
  }
}

export class APIConnectionError extends SDKError {
  constructor(message: string) {
    super(message);
    this.name = "APIConnectionError";
  }
}

export class APIStatusError extends SDKError {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "APIStatusError";
    this.status = status;
    this.body = body;
  }
}
```

### Authentication

Support three auth modes from contract defaults:

```typescript
function applyAuth(headers: Record<string, string>, auth: string, apiKey: string): void {
  if (!apiKey) return;
  const mode = (auth || "bearer").toLowerCase();
  if (mode === "none") return;
  if (mode === "basic") {
    headers["Authorization"] = `Basic ${apiKey}`;
    return;
  }
  headers["Authorization"] = `Bearer ${apiKey}`;
}
```

### Resource Pattern

Each resource becomes a class:

```typescript
export class ResponsesResource {
  constructor(private readonly _client: OpenAI) {}

  async create(request: CreateRequest): Promise<Response> {
    return this._client._request({
      method: "POST",
      path: "/v1/responses",
      body: request,
    });
  }

  stream(request: CreateRequest): AsyncIterable<ResponseEvent> {
    return this._client._stream({
      method: "POST",
      path: "/v1/responses",
      body: request,
    });
  }
}
```

### Method Signatures

| Method Pattern | Generated Signature |
|----------------|---------------------|
| No input, no output | `method(): Promise<void>` |
| No input, with output | `method(): Promise<Output>` |
| With input, no output | `method(input: Input): Promise<void>` |
| With input, with output | `method(input: Input): Promise<Output>` |

### Streaming Methods

For methods with `stream` config (SSE mode):

```typescript
// AsyncIterator-based streaming
stream(request: CreateRequest): AsyncIterable<ResponseEvent> & {
  controller: AbortController;
}
```

Usage:

```typescript
const stream = client.responses.stream({ model: "gpt-4o", input: "Hello" });

for await (const event of stream) {
  if (event.type === "response.output_text") {
    process.stdout.write(event.text);
  }
}

// Or abort early
stream.controller.abort();
```

## Streaming Implementation

### SSE Parser

```typescript
// _streaming.ts
export interface StreamOptions<T> {
  response: Response;
  parse: (data: string) => T;
  controller: AbortController;
}

export async function* streamSSE<T>(options: StreamOptions<T>): AsyncGenerator<T> {
  const reader = options.response.body?.getReader();
  if (!reader) return;

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() ?? "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          if (data === "[DONE]") return;
          yield options.parse(data);
        }
      }
    }

    // Process remaining buffer
    if (buffer.startsWith("data: ")) {
      const data = buffer.slice(6);
      if (data !== "[DONE]") {
        yield options.parse(data);
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export class Stream<T> implements AsyncIterable<T> {
  readonly controller: AbortController;
  private _iterator: AsyncGenerator<T>;

  constructor(
    private readonly _client: { _streamRequest(opts: any): Promise<Response> },
    private readonly _options: { method: string; path: string; body?: unknown },
    private readonly _parse: (data: string) => T,
  ) {
    this.controller = new AbortController();
    this._iterator = this._stream();
  }

  private async *_stream(): AsyncGenerator<T> {
    const response = await this._client._streamRequest({
      ...this._options,
      signal: this.controller.signal,
    });

    yield* streamSSE({
      response,
      parse: this._parse,
      controller: this.controller,
    });
  }

  [Symbol.asyncIterator](): AsyncIterator<T> {
    return this._iterator;
  }
}
```

## Generated File Structure

```
sdk/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts          # Main exports
│   ├── _client.ts        # Client class with HTTP logic
│   ├── _types.ts         # All type definitions
│   ├── _resources.ts     # Resource classes
│   └── _streaming.ts     # SSE streaming utilities
```

### package.json

```json
{
  "name": "{{.Package}}",
  "version": "{{.Version}}",
  "type": "module",
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "import": "./dist/index.js",
      "types": "./dist/index.d.ts"
    }
  },
  "files": ["dist"],
  "scripts": {
    "build": "tsc",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "lib": ["ES2020", "DOM"],
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true,
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*"]
}
```

## Implementation Details

### Model Structure

```go
type model struct {
    Package string
    Version string

    Service struct {
        Name        string
        Sanitized   string
        Description string
    }

    Defaults struct {
        BaseURL string
        Auth    string
        Headers []kv
    }

    Types     []typeModel
    Resources []resourceModel

    HasSSE bool
}

type typeModel struct {
    Name        string
    Description string
    Kind        contract.TypeKind

    Fields   []fieldModel  // for struct
    Elem     string        // for slice/map
    Tag      string        // for union (discriminator field)
    Variants []variantModel // for union
}

type fieldModel struct {
    Name        string
    TSName      string  // camelCase
    Description string
    TSType      string

    Optional bool
    Nullable bool
    Enum     []string
    Const    string
}

type resourceModel struct {
    Name        string
    TSName      string  // camelCase
    ClassName   string  // PascalCase + "Resource"
    Description string
    Methods     []methodModel
}

type methodModel struct {
    Name        string
    TSName      string  // camelCase
    Description string

    HasInput  bool
    HasOutput bool
    InputType string
    OutputType string

    HTTPMethod string
    HTTPPath   string

    IsStreaming    bool
    StreamMode     string
    StreamIsSSE    bool
    StreamItemType string
}
```

### Naming Conventions

| Context | Convention | Example |
|---------|------------|---------|
| Type names | PascalCase | `CreateRequest` |
| Field names | camelCase | `inputText` |
| Method names | camelCase | `create`, `stream` |
| Resource names | camelCase | `responses` |
| Class names | PascalCase | `ResponsesResource` |

### Template Functions

```go
template.FuncMap{
    "tsQuote":   func(s string) string { return fmt.Sprintf("%q", s) },
    "tsIdent":   tsIdent,  // validate TS identifier
    "camel":     toCamel,  // snake_case -> camelCase
    "pascal":    toPascal, // snake_case -> PascalCase
    "join":      strings.Join,
    "trim":      strings.TrimSpace,
    "lower":     strings.ToLower,
}
```

## Testing Strategy

### Multi-Runtime Tests

Tests verify the SDK works across Node.js, Bun, and Deno:

```go
func TestTSSDK_E2E_Node(t *testing.T) {
    if !e2eEnabled() {
        t.Skip("SDKTS_E2E not enabled")
    }
    requireNode(t)
    // ... test with Node.js
}

func TestTSSDK_E2E_Bun(t *testing.T) {
    if !e2eEnabled() {
        t.Skip("SDKTS_E2E not enabled")
    }
    requireBun(t)
    // ... test with Bun
}

func TestTSSDK_E2E_Deno(t *testing.T) {
    if !e2eEnabled() {
        t.Skip("SDKTS_E2E not enabled")
    }
    requireDeno(t)
    // ... test with Deno
}
```

### TypeScript Type Checking

```go
func TestGenerate_TypeChecks(t *testing.T) {
    svc := minimalServiceContract(t)
    root := writeGeneratedTSSDK(t, svc)

    // Run tsc --noEmit to verify types
    cmd := exec.Command("npx", "tsc", "--noEmit")
    cmd.Dir = root
    // ...
}
```

### E2E HTTP Tests

```go
func TestTSSDK_E2E_HTTP_RequestShape(t *testing.T) {
    // Start test server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request shape
        // Return JSON response
    }))

    // Generate SDK, run TypeScript test script
    script := `
import { OpenAI } from './src/index.js';
const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const resp = await client.responses.create({ model: 'gpt-4o' });
console.log(resp.id);
`
    // Execute with Node/Bun/Deno
}
```

### SSE Streaming Tests

```go
func TestTSSDK_E2E_SSE_Stream(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        flusher := w.(http.Flusher)

        io.WriteString(w, "data: {\"type\":\"text\",\"content\":\"a\"}\n\n")
        flusher.Flush()

        io.WriteString(w, "data: {\"type\":\"text\",\"content\":\"b\"}\n\n")
        flusher.Flush()

        io.WriteString(w, "data: [DONE]\n\n")
        flusher.Flush()
    }))

    script := `
import { OpenAI } from './src/index.js';
const client = new OpenAI({ apiKey: 'sk-test', baseURL: '` + srv.URL + `' });
const events = [];
for await (const event of client.responses.stream({ model: 'gpt-4o' })) {
    events.push(event);
}
console.log(events.length);
console.log(events.map(e => e.content).join(''));
`
    // Expect output: "2\nab"
}
```

## Runtime Detection Helpers

```go
func requireNode(t *testing.T) {
    t.Helper()
    if _, err := exec.LookPath("node"); err != nil {
        t.Skip("node not installed")
    }
}

func requireBun(t *testing.T) {
    t.Helper()
    if _, err := exec.LookPath("bun"); err != nil {
        t.Skip("bun not installed")
    }
}

func requireDeno(t *testing.T) {
    t.Helper()
    if _, err := exec.LookPath("deno"); err != nil {
        t.Skip("deno not installed")
    }
}
```

## Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `SDKTS_E2E` | `0`, `1`, `strict` | Enable E2E tests |
| `SDKTS_RUNTIME` | `node`, `bun`, `deno`, `all` | Which runtime(s) to test |

## Runtime-Specific Notes

### Node.js

- Requires Node 18+ for native fetch
- Uses `npx tsc` for type checking
- Execute: `node --experimental-vm-modules script.mjs`

### Bun

- Native TypeScript support
- Built-in fetch
- Execute: `bun run script.ts`

### Deno

- Native TypeScript support
- Built-in fetch
- Execute: `deno run --allow-net script.ts`
- May need import maps for npm packages

## Success Criteria

- [ ] `Generate(svc, cfg)` produces valid TypeScript
- [ ] Generated code type-checks with `tsc --noEmit`
- [ ] All contract.Type kinds supported (struct, slice, map, union)
- [ ] Client with resource pattern works
- [ ] Streaming methods produce AsyncIterable
- [ ] Tests pass on Node.js 18+
- [ ] Tests pass on Bun
- [ ] Tests pass on Deno
- [ ] OpenAI-like sample generates complete SDK

## Future Extensions

Not in scope for v1:

- **Request builders** - Fluent builder pattern
- **Pagination** - Async iterator for paginated lists
- **Retry/backoff** - Configurable retry policies
- **File uploads** - Multipart form data
- **WebSocket streaming** - Bidirectional streams
- **JSDoc generation** - Rich documentation in types
