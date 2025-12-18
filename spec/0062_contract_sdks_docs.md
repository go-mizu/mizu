# Spec 0062: Contract SDKs Documentation

## Overview

Create comprehensive documentation for the SDK generators in `contract/v2/sdk/*`. The documentation should be detailed, practical, and accessible to absolute beginners while also being useful as a reference for experienced developers.

## Documentation Structure

### Pages to Create

| Page | Path | Description |
|------|------|-------------|
| SDK Overview | `docs/contract/sdk-overview.mdx` | Introduction to SDK generation, why use it, comparison to OpenAPI-based generation |
| Go SDK | `docs/contract/sdk-go.mdx` | Complete guide for generating and using Go SDKs |
| Python SDK | `docs/contract/sdk-python.mdx` | Complete guide for generating and using Python SDKs |
| TypeScript SDK | `docs/contract/sdk-typescript.mdx` | Complete guide for generating and using TypeScript SDKs |

### Updates Required

| File | Changes |
|------|---------|
| `docs/docs.json` | Add new SDK pages under Contracts tab |
| `docs/contract/client-generation.mdx` | Add links to native SDK generation |

## Content Guidelines

### Target Audience

- **Primary**: Developers new to Mizu and SDK generation
- **Secondary**: Experienced developers looking for API reference

### Writing Style

1. **Start with "Why"**: Explain the benefits before the how-to
2. **Complete Examples**: Every feature should have a runnable example
3. **Progressive Complexity**: Start simple, add complexity gradually
4. **Real-World Context**: Use realistic examples (todo app, user service, chat API)
5. **Error Scenarios**: Show what errors look like and how to handle them
6. **Best Practices**: Include tips and recommendations throughout

### Structure for Each SDK Page

1. **Introduction**
   - What is this SDK generator?
   - Key features and benefits
   - Prerequisites

2. **Quick Start**
   - Minimal working example
   - Step-by-step walkthrough

3. **Generated Code Structure**
   - File layout explanation
   - What each file contains

4. **Client Usage**
   - Creating a client
   - Configuration options
   - Making requests
   - Handling responses

5. **Type System**
   - How Go/Python/TS types are generated
   - Type mapping reference table
   - Struct/Interface generation
   - Optional and nullable fields
   - Enum and const fields

6. **Resources and Methods**
   - Resource-based API access
   - Method patterns (CRUD)
   - Input/Output types

7. **Streaming**
   - SSE streaming support
   - How to consume streams
   - Cancellation

8. **Error Handling**
   - Error types
   - Status codes
   - Retries

9. **Advanced Topics**
   - Custom headers
   - Authentication modes
   - Timeout configuration
   - HTTP client customization

10. **API Reference**
    - Generator function signature
    - Config options
    - Generated types summary

## SDK Overview Page Content

### Sections

1. **What is SDK Generation?**
   - Definition and purpose
   - Benefits over manual client code
   - Benefits over OpenAPI-generated clients

2. **Available SDK Generators**
   - Quick comparison table
   - When to use each

3. **How It Works**
   - Diagram: Contract Service -> SDK Generator -> Client Code
   - Explanation of the generation process

4. **Prerequisites**
   - Go service with contract
   - Understanding of types and resources

5. **Choosing the Right Approach**
   - Native SDK vs OpenAPI generation
   - Decision matrix

## Go SDK Page Content

### Key Topics

1. **Zero Dependencies**
   - Generated code only uses stdlib
   - Self-contained, copy-paste ready

2. **OpenAI-Style API**
   - Resource-based access: `client.Todos.Create()`
   - Fluent, discoverable API

3. **Type Safety**
   - Full Go type system
   - IDE autocompletion
   - Compile-time errors

4. **Streaming**
   - `EventStream[T]` generic type
   - `Next()`, `Event()`, `Err()`, `Close()`
   - Iterator pattern

5. **Code Example**
   - Complete working example
   - Server code
   - Generated client code
   - Usage code

## Python SDK Page Content

### Key Topics

1. **Modern Python**
   - Python 3.8+ with type hints
   - Dataclass-based models
   - httpx for HTTP client

2. **Sync and Async**
   - `OpenAI` for synchronous code
   - `AsyncOpenAI` for async/await

3. **uv-friendly**
   - pyproject.toml included
   - Ready for uv or pip install

4. **Pydantic-style Types**
   - Dataclass models with type annotations
   - Optional/nullable handling

5. **Streaming**
   - Iterator-based SSE streaming
   - Async iterator support

## TypeScript SDK Page Content

### Key Topics

1. **Runtime Agnostic**
   - Works on Node.js, Bun, Deno
   - Native fetch API
   - No external dependencies

2. **TypeScript-First**
   - Full type inference
   - Strict mode compatible
   - IDE integration

3. **Modern JavaScript**
   - ES modules
   - async/await
   - AsyncIterable for streams

4. **Streaming**
   - `for await...of` syntax
   - Abort controller for cancellation

5. **Package Ready**
   - package.json included
   - tsconfig.json included
   - Ready to publish to npm

## Navigation Updates

Add new group to docs.json under Contracts tab:

```json
{
  "group": "SDK Generation",
  "pages": [
    "contract/sdk-overview",
    "contract/sdk-go",
    "contract/sdk-python",
    "contract/sdk-typescript"
  ]
}
```

Position: After "Transports" group, before "Advanced" group.

## Success Criteria

- [ ] All 4 documentation pages created
- [ ] Each page has complete, runnable examples
- [ ] Type mapping tables are accurate
- [ ] Code examples are tested and work
- [ ] Navigation is updated in docs.json
- [ ] client-generation.mdx links to new pages
- [ ] Documentation is clear for beginners
- [ ] API reference is complete for experienced users

## Implementation Notes

### Example Service for Documentation

Use a consistent example throughout all SDK docs:

```go
// TodoService is used across all SDK documentation examples
type TodoAPI interface {
    Create(input CreateInput) (Todo, error)
    List() (ListOutput, error)
    Get(id string) (Todo, error)
    Delete(id string) error
    Stream(input StreamInput) iter.Seq[TodoEvent]
}

type Todo struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Done      bool      `json:"done"`
    CreatedAt time.Time `json:"created_at"`
}

type CreateInput struct {
    Title string `json:"title"`
}

type ListOutput struct {
    Items []Todo `json:"items"`
    Total int    `json:"total"`
}

type StreamInput struct {
    Watch bool `json:"watch"`
}

type TodoEvent struct {
    Type string `json:"type"`
    Todo *Todo  `json:"todo,omitempty"`
}
```

### Code Formatting

- Use syntax highlighting for all code blocks
- Include language identifiers (go, python, typescript, bash)
- Keep code examples concise but complete
- Add comments to explain non-obvious parts

### Cross-References

- Link between SDK pages where appropriate
- Link to core contract documentation
- Link to transport documentation (REST, JSON-RPC)
- Link to OpenAPI generation for comparison
