# Contract Template & Documentation Implementation Plan

## Overview

This plan outlines the implementation of the `contract` CLI template and comprehensive documentation for Mizu's contract system. The contract package provides a transport-neutral service contract that allows writing plain Go services and exposing them via multiple transports (REST, JSON-RPC).

## Understanding the Contract System

### Core Concepts

1. **Service Registration**: Plain Go structs with methods are registered using `contract.Register(name, svc)` to create a `*contract.Service`
2. **Method Signatures**: Supported canonical forms:
   - `func (s *S) Method(ctx context.Context, in *In) (*Out, error)`
   - `func (s *S) Method(ctx context.Context) (*Out, error)`
   - `func (s *S) Method(ctx context.Context, in *In) error`
   - `func (s *S) Method(ctx context.Context) error`
3. **Type Registry**: Automatic JSON schema generation for input/output types
4. **Compiled Invokers**: Reflection performed once at registration, runtime calls use compiled invokers
5. **Transport Mounting**:
   - `contract.MountREST(mux, svc)` - REST endpoints with verb conventions
   - `contract.MountJSONRPC(mux, path, svc)` - JSON-RPC 2.0 endpoint
   - `contract.ServeOpenAPI(mux, path, svc)` - OpenAPI 3.1 schema

### REST Conventions

| Method Prefix | HTTP Verb |
|---------------|-----------|
| `Create*`     | POST      |
| `Get*`        | GET       |
| `List*`       | GET       |
| `Update*`     | PUT       |
| `Delete*`     | DELETE    |
| (default)     | POST      |

## Implementation Tasks

### 1. CLI Template: `cli/templates/contract/`

Create a new template demonstrating the contract pattern with REST and JSON-RPC transports.

#### Template Structure

```
cli/templates/contract/
├── template.json
├── cmd/
│   └── api/
│       └── main.go.tmpl
├── service/
│   └── todo/
│       └── todo.go.tmpl
└── app/
    └── server/
        ├── server.go.tmpl
        └── config.go.tmpl
```

#### Files to Create

1. **template.json** - Template metadata
2. **cmd/api/main.go.tmpl** - Entry point with graceful shutdown
3. **service/todo/todo.go.tmpl** - Example todo service (plain Go, no framework deps)
4. **app/server/server.go.tmpl** - Server setup with REST, JSON-RPC, and OpenAPI
5. **app/server/config.go.tmpl** - Configuration management

### 2. Documentation: Guides Tab - Contract Section

Add a new "Contract" group to the Guides tab in `docs/docs.json`.

#### Pages to Create in `docs/contract/`

| File | Title | Description |
|------|-------|-------------|
| `overview.mdx` | Overview | Introduction to contracts and transport-neutral services |
| `service.mdx` | Defining Services | How to define plain Go services with proper signatures |
| `register.mdx` | Registration | Using `contract.Register()` and inspecting contracts |
| `types.mdx` | Type System | Type registry, schemas, and JSON schema generation |
| `invoker.mdx` | Invokers | How compiled invokers work and calling methods |
| `rest.mdx` | REST Transport | Using `MountREST()` and REST conventions |
| `jsonrpc.mdx` | JSON-RPC Transport | Using `MountJSONRPC()` and JSON-RPC 2.0 |
| `openapi.mdx` | OpenAPI | Serving OpenAPI specs with `ServeOpenAPI()` |

### 3. CLI Tab Updates

Add new template documentation page:

1. Update `docs/docs.json` CLI tab Templates group to include `template-contract`
2. Create `docs/cli/template-contract.mdx` - Contract template documentation

### 4. Update Templates Overview

Update `docs/cli/templates.mdx` to include the new contract template in:
- Available Templates table
- Template list output
- Choosing a Template section
- Decision Guide table

## File Changes Summary

### New Files

| Path | Description |
|------|-------------|
| `cli/templates/contract/template.json` | Template metadata |
| `cli/templates/contract/cmd/api/main.go.tmpl` | Entry point |
| `cli/templates/contract/service/todo/todo.go.tmpl` | Todo service |
| `cli/templates/contract/app/server/server.go.tmpl` | Server setup |
| `cli/templates/contract/app/server/config.go.tmpl` | Configuration |
| `docs/contract/overview.mdx` | Contract overview |
| `docs/contract/service.mdx` | Service definition guide |
| `docs/contract/register.mdx` | Registration guide |
| `docs/contract/types.mdx` | Type system guide |
| `docs/contract/invoker.mdx` | Invoker guide |
| `docs/contract/rest.mdx` | REST transport guide |
| `docs/contract/jsonrpc.mdx` | JSON-RPC transport guide |
| `docs/contract/openapi.mdx` | OpenAPI guide |
| `docs/cli/template-contract.mdx` | Contract template docs |

### Modified Files

| Path | Changes |
|------|---------|
| `docs/docs.json` | Add Contract group to Guides tab, add template-contract to CLI tab |
| `docs/cli/templates.mdx` | Add contract template to listings |

## Implementation Order

1. Write PLAN.md (this file)
2. Create `cli/templates/contract/` template files
3. Create `docs/contract/*.mdx` documentation pages
4. Create `docs/cli/template-contract.mdx`
5. Update `docs/docs.json` with new sections
6. Update `docs/cli/templates.mdx` with new template

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Plain Go Service                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  service/todo/todo.go                                    │   │
│  │  - No framework dependencies                             │   │
│  │  - Pure business logic                                   │   │
│  │  - Easy to test                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ contract.Register()
┌─────────────────────────────────────────────────────────────────┐
│                    contract.Service                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  - Methods with compiled invokers                        │   │
│  │  - TypeRegistry with JSON schemas                        │   │
│  │  - Transport-neutral contract                            │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
    ┌──────────┐        ┌──────────┐        ┌──────────┐
    │   REST   │        │ JSON-RPC │        │  OpenAPI │
    │ MountREST│        │MountJSONRPC      │ServeOpenAPI
    └──────────┘        └──────────┘        └──────────┘
```

## Key Design Decisions

1. **Template focuses on contract pattern** - Unlike `api` template which uses Mizu handlers directly, this template demonstrates the contract-first approach
2. **Multiple transports in one server** - Show REST, JSON-RPC, and OpenAPI all served from the same service
3. **Clean service layer** - Service code has zero dependencies on HTTP or transport concerns
4. **Comprehensive documentation** - Each component of the contract system gets its own page
