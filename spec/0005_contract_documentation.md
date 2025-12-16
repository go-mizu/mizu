# Contract Documentation Restructuring Plan

## Overview

This document specifies a complete restructuring of the Mizu Contract documentation to provide the best possible Developer Experience (DX). The goal is to move Contract documentation from being a subsection of Guides into its own dedicated tab with comprehensive, well-organized content.

## Current State

### Current docs.json Structure

```json
{
  "navigation": {
    "tabs": [
      {
        "tab": "Guides",
        "groups": [
          // ... other groups
          {
            "group": "Contract",
            "pages": [
              "contract/overview",
              "contract/service",
              "contract/register",
              "contract/types",
              "contract/invoker",
              "contract/rest",
              "contract/jsonrpc",
              "contract/openapi"
            ]
          }
        ]
      },
      {"tab": "Middlewares", ...},
      {"tab": "CLI", ...},
      {"tab": "Examples", ...}
    ]
  }
}
```

### Problems with Current Structure

1. **Limited Visibility**: Contract is buried inside Guides tab
2. **Missing Transports**: No documentation for MCP and tRPC
3. **No Advanced Topics**: Missing middleware, testing, client generation
4. **Insufficient Examples**: Need more real-world patterns
5. **No Migration Guide**: Users need help transitioning
6. **No API Reference**: Detailed type/function documentation missing

## Design Principles for Best DX

### 1. Progressive Disclosure
- Start simple, reveal complexity as needed
- Quick start in under 5 minutes
- Advanced topics accessible but not overwhelming

### 2. Task-Oriented Structure
- Organize by what developers want to DO
- "I want to expose my service via MCP" -> MCP Transport page
- "I want to validate inputs" -> Types & Validation page

### 3. Copy-Paste Ready Examples
- Every code block should be runnable
- Include both minimal and complete examples
- Show input/output for curl commands

### 4. Clear Mental Models
- Architecture diagrams for complex concepts
- Comparison tables for transport options
- Decision trees for choosing approaches

### 5. Searchability
- Good page titles and descriptions
- Proper heading hierarchy
- Keywords in content

## New Documentation Structure

### Tab: Contract (Dedicated)

```json
{
  "tab": "Contract",
  "groups": [
    {
      "group": "Getting Started",
      "pages": [
        "contract/introduction",
        "contract/quick-start",
        "contract/architecture"
      ]
    },
    {
      "group": "Core Concepts",
      "pages": [
        "contract/service-definition",
        "contract/registration",
        "contract/types-and-schemas",
        "contract/method-signatures",
        "contract/errors"
      ]
    },
    {
      "group": "Transports",
      "pages": [
        "contract/transports-overview",
        "contract/transport-rest",
        "contract/transport-jsonrpc",
        "contract/transport-mcp",
        "contract/transport-trpc",
        "contract/transport-openapi"
      ]
    },
    {
      "group": "Advanced",
      "pages": [
        "contract/middleware",
        "contract/testing",
        "contract/client-generation",
        "contract/custom-transports"
      ]
    },
    {
      "group": "Reference",
      "pages": [
        "contract/api-reference",
        "contract/error-codes",
        "contract/comparison"
      ]
    }
  ]
}
```

## Detailed Page Specifications

### Getting Started Group

#### 1. Introduction (`contract/introduction.mdx`)

**Purpose**: Explain what Contract is, why it exists, and who should use it.

**Outline**:
```markdown
# Introduction to Contract

## What is Contract?
- Transport-neutral service definitions
- Write once, expose via REST, JSON-RPC, MCP, tRPC
- Type-safe with auto-generated schemas

## Why Use Contract?
- **Multi-Protocol APIs**: One codebase, many protocols
- **AI-Ready**: Built-in MCP support for AI tool integration
- **Type Safety**: JSON schemas from Go types
- **Clean Architecture**: Business logic separated from transport

## When to Use Contract
[Decision guide with checklist]

## When NOT to Use Contract
[List of scenarios where plain Mizu handlers are better]

## Prerequisites
- Go 1.21+
- Basic understanding of Mizu routing
- Familiarity with JSON-RPC or REST concepts (helpful)
```

#### 2. Quick Start (`contract/quick-start.mdx`)

**Purpose**: Get developers running in under 5 minutes.

**Outline**:
```markdown
# Quick Start

## 1. Install Mizu CLI
```bash
go install github.com/go-mizu/mizu/cmd/mizu@latest
```

## 2. Create a New Project
```bash
mizu new myapi --template contract
cd myapi
go mod tidy
```

## 3. Run the Server
```bash
go run ./cmd/api
```

## 4. Test Your API

### REST
```bash
curl -X POST http://localhost:8080/todos -d '{"title":"Hello"}'
```

### JSON-RPC
```bash
curl -X POST http://localhost:8080/rpc -d '{"jsonrpc":"2.0","id":1,"method":"Create","params":{"title":"Hello"}}'
```

### MCP
```bash
curl -X POST http://localhost:8080/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## 5. View OpenAPI Spec
Open http://localhost:8080/openapi.json

## Next Steps
- [Define Your Service](/contract/service-definition)
- [Choose a Transport](/contract/transports-overview)
```

#### 3. Architecture (`contract/architecture.mdx`)

**Purpose**: Explain the high-level design and flow.

**Content**:
- ASCII architecture diagram
- Registration flow explanation
- Transport layer abstraction
- Type system overview
- Request lifecycle

### Core Concepts Group

#### 4. Service Definition (`contract/service-definition.mdx`)

**Purpose**: Detailed guide on writing services.

**Outline**:
```markdown
# Service Definition

## Method Signatures
[All 4 supported signatures with examples]

## Service Metadata
[ServiceMeta interface]

## Method Metadata
[MethodMeta interface with HTTP overrides]

## Best Practices
- Naming conventions
- Input/Output struct design
- Error handling patterns
```

#### 5. Registration (`contract/registration.mdx`)

**Purpose**: Deep dive into the Register function.

**Content**:
- `contract.Register()` API
- What happens during registration
- Error handling
- Multiple service registration

#### 6. Types and Schemas (`contract/types-and-schemas.mdx`)

**Purpose**: Explain the type system and JSON schema generation.

**Outline**:
```markdown
# Types and Schemas

## Automatic Schema Generation
[How Go types become JSON schemas]

## Supported Types
- Primitives
- Structs
- Slices/Arrays
- Maps
- Pointers (nullable)
- time.Time

## Contract Tags
```go
type Input struct {
    Name  string `json:"name" contract:"required,minLength=1,maxLength=100"`
    Email string `json:"email" contract:"format=email"`
}
```

## Enums
[ContractEnum interface]

## TypeRegistry API
[Querying registered types]
```

#### 7. Method Signatures (`contract/method-signatures.mdx`)

**Purpose**: Detailed explanation of all method signatures.

**Content**:
- All 4 signature variants with use cases
- Input validation
- Output handling
- Context usage

#### 8. Errors (`contract/errors.mdx`)

**Purpose**: Error handling across transports.

**Content**:
- `contract.Error` type
- Error codes (gRPC-aligned)
- HTTP status mapping
- JSON-RPC error mapping
- Error constructors

### Transports Group

#### 9. Transports Overview (`contract/transports-overview.mdx`)

**Purpose**: Help developers choose the right transport.

**Outline**:
```markdown
# Transports Overview

## Available Transports

| Transport | Protocol | Best For |
|-----------|----------|----------|
| REST | HTTP | Web APIs, browser clients |
| JSON-RPC | JSON-RPC 2.0 | RPC-style APIs, batch calls |
| MCP | Model Context Protocol | AI tool integration |
| tRPC | tRPC-like | TypeScript clients |
| OpenAPI | OpenAPI 3.1 | Documentation, codegen |

## Comparison Table
[Feature matrix]

## Using Multiple Transports
```go
mux := http.NewServeMux()
contract.MountREST(mux, svc)
contract.MountJSONRPC(mux, "/rpc", svc)
mcp.Mount(mux, "/mcp", svc)
trpc.Mount(mux, "/trpc", svc)
openapi.Mount(mux, "/openapi.json", svc)
```
```

#### 10. REST Transport (`contract/transport-rest.mdx`)

**Enhance existing with**:
- More examples
- Custom path configuration
- Error handling details
- Query parameter handling (future)

#### 11. JSON-RPC Transport (`contract/transport-jsonrpc.mdx`)

**Enhance existing with**:
- New package API (`jsonrpc.Mount`)
- Options pattern
- Custom error mapping
- Middleware integration

#### 12. MCP Transport (`contract/transport-mcp.mdx`) [NEW]

**Outline**:
```markdown
# MCP Transport

## Overview
MCP (Model Context Protocol) enables AI models to interact with your services as tools.

## Quick Start
```go
import "github.com/go-mizu/mizu/contract/transport/mcp"

mcp.Mount(mux, "/mcp", svc)
```

## MCP Concepts
- Tools (your methods)
- Resources (not yet supported)
- Prompts (not yet supported)

## Protocol Flow
[Sequence diagram]

## Tool Definition
[How methods become tools]

## Configuration Options
- WithServerInfo
- WithInstructions
- WithAllowedOrigins

## Testing with Claude
[How to connect to Claude Desktop]

## Security Considerations
- Origin validation
- Authentication patterns
```

#### 13. tRPC Transport (`contract/transport-trpc.mdx`) [NEW]

**Outline**:
```markdown
# tRPC Transport

## Overview
tRPC-like transport for typed RPC with client generation potential.

## Quick Start
```go
import "github.com/go-mizu/mizu/contract/transport/trpc"

trpc.Mount(mux, "/trpc", svc)
```

## Endpoint Layout
- POST /trpc/<procedure> - Call
- GET /trpc.meta - Introspection

## Response Envelope
```json
// Success
{"result": {"data": ...}}

// Error
{"error": {"code": "...", "message": "..."}}
```

## Introspection
[.meta endpoint details]
```

#### 14. OpenAPI Transport (`contract/transport-openapi.mdx`)

**Enhance existing with**:
- New package API
- Swagger UI integration
- Schema customization

### Advanced Group

#### 15. Middleware (`contract/middleware.mdx`) [NEW]

**Content**:
- Transport middleware vs contract middleware
- Authentication patterns
- Logging and tracing
- Rate limiting

#### 16. Testing (`contract/testing.mdx`) [NEW]

**Content**:
- Unit testing services
- Integration testing with transports
- Mock invokers
- Test helpers

#### 17. Client Generation (`contract/client-generation.mdx`) [NEW]

**Content**:
- OpenAPI clients
- TypeScript from tRPC
- Go client generation

#### 18. Custom Transports (`contract/custom-transports.mdx`) [NEW]

**Content**:
- Transport interface
- Implementing a custom transport
- gRPC example outline

### Reference Group

#### 19. API Reference (`contract/api-reference.mdx`) [NEW]

**Content**:
- All exported types
- All exported functions
- Package-by-package reference

#### 20. Error Codes (`contract/error-codes.mdx`) [NEW]

**Content**:
- Complete error code list
- HTTP status mappings
- JSON-RPC code mappings
- gRPC code mappings

#### 21. Comparison (`contract/comparison.mdx`) [NEW]

**Content**:
- Contract vs plain Mizu handlers
- Contract vs gRPC
- Contract vs tRPC (original)
- Transport comparison matrix

## Updated docs.json

```json
{
  "$schema": "https://mintlify.com/docs.json",
  "theme": "mint",
  "name": "Mizu",
  "colors": {
    "primary": "#60A5FA",
    "light": "#60A5FA",
    "dark": "#60A5FA"
  },
  "navigation": {
    "global": {
      "anchors": [
        {"anchor": "Discord", "href": "https://discord.gg/8QpMsNBB8n", "icon": "discord"},
        {"anchor": "GitHub", "href": "https://github.com/go-mizu/mizu", "icon": "github"}
      ]
    },
    "tabs": [
      {
        "tab": "Guides",
        "groups": [
          {
            "group": "Overview",
            "pages": ["overview/intro", "overview/why", "overview/features", "overview/use-cases", "overview/roadmap"]
          },
          {
            "group": "Getting Started",
            "pages": ["get-started/quick-start", "get-started/deployment"]
          },
          {
            "group": "Concepts",
            "pages": [
              "concepts/overview", "concepts/app", "concepts/routing", "concepts/handler",
              "concepts/context", "concepts/request", "concepts/response", "concepts/static",
              "concepts/logging", "concepts/error", "concepts/middleware"
            ]
          }
        ]
      },
      {
        "tab": "Contract",
        "groups": [
          {
            "group": "Getting Started",
            "pages": ["contract/introduction", "contract/quick-start", "contract/architecture"]
          },
          {
            "group": "Core Concepts",
            "pages": [
              "contract/service-definition",
              "contract/registration",
              "contract/types-and-schemas",
              "contract/method-signatures",
              "contract/errors"
            ]
          },
          {
            "group": "Transports",
            "pages": [
              "contract/transports-overview",
              "contract/transport-rest",
              "contract/transport-jsonrpc",
              "contract/transport-mcp",
              "contract/transport-trpc",
              "contract/transport-openapi"
            ]
          },
          {
            "group": "Advanced",
            "pages": [
              "contract/middleware",
              "contract/testing",
              "contract/client-generation",
              "contract/custom-transports"
            ]
          },
          {
            "group": "Reference",
            "pages": ["contract/api-reference", "contract/error-codes", "contract/comparison"]
          }
        ]
      },
      {
        "tab": "Middlewares",
        "groups": [...]
      },
      {
        "tab": "CLI",
        "groups": [...]
      },
      {
        "tab": "Examples",
        "groups": [...]
      }
    ]
  }
}
```

## Implementation Plan

### Phase 1: Structure Setup

1. Update `docs/docs.json` with new Contract tab structure
2. Create placeholder files for all new pages
3. Rename existing files to match new structure

### Phase 2: Core Content

1. **contract/introduction.mdx** - New overview page
2. **contract/quick-start.mdx** - Enhance from overview
3. **contract/architecture.mdx** - New architecture guide
4. **contract/service-definition.mdx** - Merge service.mdx + register.mdx
5. **contract/types-and-schemas.mdx** - Enhance types.mdx
6. **contract/errors.mdx** - New error handling guide

### Phase 3: Transports

1. **contract/transports-overview.mdx** - New transport comparison
2. **contract/transport-rest.mdx** - Enhance existing
3. **contract/transport-jsonrpc.mdx** - Enhance with new API
4. **contract/transport-mcp.mdx** - New page
5. **contract/transport-trpc.mdx** - New page
6. **contract/transport-openapi.mdx** - Enhance existing

### Phase 4: Advanced Topics

1. **contract/middleware.mdx** - New middleware guide
2. **contract/testing.mdx** - New testing guide
3. **contract/client-generation.mdx** - New client guide
4. **contract/custom-transports.mdx** - New custom transport guide

### Phase 5: Reference

1. **contract/api-reference.mdx** - New API reference
2. **contract/error-codes.mdx** - New error code reference
3. **contract/comparison.mdx** - New comparison guide

## File Mapping (Old to New)

| Old File | New File | Action |
|----------|----------|--------|
| contract/overview.mdx | contract/introduction.mdx | Rename + Enhance |
| contract/service.mdx | contract/service-definition.mdx | Rename + Enhance |
| contract/register.mdx | contract/registration.mdx | Rename |
| contract/types.mdx | contract/types-and-schemas.mdx | Rename + Enhance |
| contract/invoker.mdx | contract/method-signatures.mdx | Rename + Merge |
| contract/rest.mdx | contract/transport-rest.mdx | Rename + Enhance |
| contract/jsonrpc.mdx | contract/transport-jsonrpc.mdx | Rename + Enhance |
| contract/openapi.mdx | contract/transport-openapi.mdx | Rename + Enhance |
| (new) | contract/quick-start.mdx | Create |
| (new) | contract/architecture.mdx | Create |
| (new) | contract/errors.mdx | Create |
| (new) | contract/transports-overview.mdx | Create |
| (new) | contract/transport-mcp.mdx | Create |
| (new) | contract/transport-trpc.mdx | Create |
| (new) | contract/middleware.mdx | Create |
| (new) | contract/testing.mdx | Create |
| (new) | contract/client-generation.mdx | Create |
| (new) | contract/custom-transports.mdx | Create |
| (new) | contract/api-reference.mdx | Create |
| (new) | contract/error-codes.mdx | Create |
| (new) | contract/comparison.mdx | Create |

## Success Criteria

- [ ] Contract is its own tab in documentation
- [ ] All 5 transports (REST, JSON-RPC, MCP, tRPC, OpenAPI) are documented
- [ ] Quick start works in under 5 minutes
- [ ] All code examples are copy-paste ready
- [ ] Architecture diagram clearly explains the system
- [ ] Transport comparison helps users choose
- [ ] Advanced topics cover middleware, testing, client generation
- [ ] API reference is complete and accurate
- [ ] Error codes are fully documented with mappings
