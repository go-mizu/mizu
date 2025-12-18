# Contract v2 Documentation Cleanup Specification

**Status:** In Progress
**Related:** 0057_contract_docs_v2.md

## Overview

This specification defines the cleanup tasks for contract documentation after the v2 rewrite. The goal is to remove obsolete v1 patterns, eliminate duplicate content, standardize frontmatter titles, and improve documentation organization.

## 1. docs.json Updates

### Tab Renaming

| Current | New |
|---------|-----|
| `"Contract"` | `"Contracts"` |
| `"View"` | `"Views"` |

### Ordering Review

Current order is acceptable:
1. Getting Started: overview, quick-start, architecture
2. Core Concepts: service, register, types, invoker, errors
3. Transports: transports-overview, rest, jsonrpc, mcp, trpc, openapi
4. Advanced: middleware, testing, client-generation
5. Reference: api-reference, error-codes

**Recommendation:** Remove `trpc.mdx` - tRPC transport is not implemented in v2. The file uses old v1 patterns and references non-existent package paths.

## 2. Frontmatter Title Shortening

All titles should be concise, preferring one word where possible.

| File | Current Title | New Title |
|------|---------------|-----------|
| `overview.mdx` | "Contract Overview" | "Overview" |
| `quick-start.mdx` | "Quick Start" | "Quick Start" (keep) |
| `architecture.mdx` | "Architecture" | "Architecture" (keep) |
| `service.mdx` | "Defining Services" | "Services" |
| `register.mdx` | "Registration" | "Registration" (keep) |
| `types.mdx` | "Type System" | "Types" |
| `invoker.mdx` | "Invokers" | "Invokers" (keep) |
| `errors.mdx` | "Error Handling" | "Errors" |
| `transports-overview.mdx` | "Transports Overview" | "Overview" |
| `rest.mdx` | "REST Transport" | "REST" |
| `jsonrpc.mdx` | "JSON-RPC Transport" | "JSON-RPC" |
| `mcp.mdx` | "MCP Transport" | "MCP" |
| `trpc.mdx` | "tRPC Transport" | **DELETE FILE** |
| `openapi.mdx` | "OpenAPI" | "OpenAPI" (keep) |
| `middleware.mdx` | "Middleware" | "Middleware" (keep) |
| `testing.mdx` | "Testing" | "Testing" (keep) |
| `client-generation.mdx` | "Client Generation" | "Clients" |
| `api-reference.mdx` | "API Reference" | "Reference" |
| `error-codes.mdx` | "Error Codes Reference" | "Error Codes" |

## 3. Obsolete v1 Patterns to Remove

### Files with v1 Patterns

The following files use the old v1 registration pattern `contract.Register("name", &struct{})` instead of the v2 pattern `contract.Register[API](impl, opts...)`:

| File | Status | Action |
|------|--------|--------|
| `trpc.mdx` | Fully obsolete | Delete file |
| `api-reference.mdx` | Fully obsolete | Complete rewrite required |
| `architecture.mdx` | Partially obsolete | Update code examples |
| `testing.mdx` | Fully obsolete | Complete rewrite required |
| `middleware.mdx` | Partially obsolete | Update code examples |
| `client-generation.mdx` | Partially obsolete | Update code examples |
| `invoker.mdx` | Already v2 | Good |

### v1 vs v2 Pattern Reference

**v1 Pattern (OBSOLETE):**
```go
svc, err := contract.Register("todo", &TodoService{})
if err != nil {
    log.Fatal(err)
}
contract.MountREST(mux, svc)
contract.MountJSONRPC(mux, "/rpc", svc)
```

**v2 Pattern (CURRENT):**
```go
svc := contract.Register[TodoAPI](impl,
    contract.WithName("Todo"),
    contract.WithDefaultResource("todos"),
)
rest.Mount(app.Router, svc)
jsonrpc.Mount(app.Router, "/rpc", svc)
```

### Package Import Changes

| v1 Import | v2 Import |
|-----------|-----------|
| `github.com/go-mizu/mizu/contract` | `github.com/go-mizu/mizu/contract/v2` |
| N/A (inline) | `github.com/go-mizu/mizu/contract/v2/transport/rest` |
| N/A (inline) | `github.com/go-mizu/mizu/contract/v2/transport/jsonrpc` |
| `github.com/go-mizu/mizu/contract/transport/trpc` | Not available in v2 |
| `github.com/go-mizu/mizu/contract/transport/mcp` | `github.com/go-mizu/mizu/contract/v2/transport/mcp` |

## 4. Duplicate Content Analysis

### Overlapping Content

| Files | Overlap | Resolution |
|-------|---------|------------|
| `errors.mdx` + `error-codes.mdx` | Both explain error codes and mappings | Keep `errors.mdx` for patterns/usage, keep `error-codes.mdx` for comprehensive reference |
| `overview.mdx` + `quick-start.mdx` | Both have complete TodoAPI examples | `overview.mdx` focuses on concepts, `quick-start.mdx` focuses on step-by-step tutorial |
| `openapi.mdx` + `client-generation.mdx` | Both explain OpenAPI spec generation | `openapi.mdx` focuses on spec details, `client-generation.mdx` focuses on client tools |

**No files to merge** - current separation is appropriate.

## 5. Files to Delete

| File | Reason |
|------|--------|
| `trpc.mdx` | tRPC transport not implemented in v2; uses all v1 patterns |

## 6. Files Requiring Complete Rewrite

### api-reference.mdx

Current state: Uses entirely v1 API patterns.

Required changes:
- Replace all `contract.Register("name", &struct{})` with `contract.Register[API](impl, opts...)`
- Update all transport mount functions to v2 package paths
- Remove `contract.MountREST`, `contract.MountJSONRPC`, `contract.ServeOpenAPI`
- Add `rest.Mount`, `jsonrpc.Mount`, `mcp.Mount`
- Update `Service`, `Method`, `TypeRef` structs to v2 definitions
- Remove tRPC-related sections

### testing.mdx

Current state: Uses entirely v1 patterns.

Required changes:
- Update all examples to use interface-first pattern
- Replace `contract.Register("todo", todo.NewService())` with `contract.Register[TodoAPI](impl)`
- Update transport mount examples
- Remove tRPC testing references

## 7. Files Requiring Partial Updates

### architecture.mdx

Areas needing update:
- Registration example in "How It Works" section
- Any code snippets showing v1 patterns

### middleware.mdx

Areas needing update:
- Section "Using Your Custom Invoker" references tRPC transport
- Update import paths to v2
- Update `contract.DefaultInvoker(svc)` pattern if changed in v2

### client-generation.mdx

Areas needing update:
- Section "Step 1: Serve OpenAPI Spec" uses v1 patterns
- Remove tRPC references in "TypeScript Client from tRPC Meta"
- Update all Go code examples to v2

## 8. Implementation Checklist

### Phase 1: docs.json Updates
- [ ] Rename tab "Contract" to "Contracts"
- [ ] Rename tab "View" to "Views"
- [ ] Remove `contract/trpc` from pages list

### Phase 2: Delete Obsolete File
- [ ] Delete `docs/contract/trpc.mdx`

### Phase 3: Frontmatter Updates
- [ ] `overview.mdx`: title "Contract Overview" -> "Overview"
- [ ] `service.mdx`: title "Defining Services" -> "Services"
- [ ] `types.mdx`: title "Type System" -> "Types"
- [ ] `errors.mdx`: title "Error Handling" -> "Errors"
- [ ] `transports-overview.mdx`: title "Transports Overview" -> "Overview"
- [ ] `rest.mdx`: title "REST Transport" -> "REST"
- [ ] `jsonrpc.mdx`: title "JSON-RPC Transport" -> "JSON-RPC"
- [ ] `mcp.mdx`: title "MCP Transport" -> "MCP"
- [ ] `client-generation.mdx`: title "Client Generation" -> "Clients"
- [ ] `api-reference.mdx`: title "API Reference" -> "Reference"
- [ ] `error-codes.mdx`: title "Error Codes Reference" -> "Error Codes"

### Phase 4: Complete Rewrites
- [ ] Rewrite `api-reference.mdx` for v2 API
- [ ] Rewrite `testing.mdx` for v2 patterns

### Phase 5: Partial Updates
- [ ] Update `architecture.mdx` code examples
- [ ] Update `middleware.mdx` code examples
- [ ] Update `client-generation.mdx` code examples

### Phase 6: Cross-Reference Cleanup
- [ ] Remove all links to `/contract/trpc`
- [ ] Update links in "See Also" sections

## 9. Files Already Correct (No Changes Needed)

These files already use v2 patterns:
- `overview.mdx`
- `quick-start.mdx`
- `service.mdx`
- `register.mdx`
- `types.mdx`
- `invoker.mdx`
- `errors.mdx`
- `transports-overview.mdx`
- `rest.mdx`
- `jsonrpc.mdx`
- `mcp.mdx`
- `openapi.mdx`
- `error-codes.mdx`

## 10. Validation

After cleanup, verify:
1. All code examples use `contract.Register[API](impl, ...)` pattern
2. All imports use `github.com/go-mizu/mizu/contract/v2` path
3. No references to tRPC transport
4. All frontmatter titles are concise
5. No broken cross-references

## Notes

- The `invoker.mdx` file already uses v2 patterns correctly
- `error-codes.mdx` uses generic patterns that work for both v1 and v2
- Transport-specific docs (rest, jsonrpc, mcp, openapi) already use v2 patterns
