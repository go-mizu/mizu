# Spec 0009: Contract Documentation Rewrite for Beginners

## Summary

Rewrite all contract documentation in `docs/contract/*.mdx` to be accessible to absolute beginners who are new to:
- Go programming
- API development concepts
- RPC protocols (JSON-RPC, REST, etc.)
- AI tool integration (MCP)

## Goals

1. **Beginner-First**: Assume zero prior knowledge of the contract pattern
2. **Explain "Why"**: Every concept should explain why it exists before how it works
3. **Progressive Complexity**: Start simple, build up gradually
4. **Real Examples**: Use concrete, relatable examples (todo list, user management)
5. **Visual Learning**: Add diagrams and step-by-step walkthroughs

## Writing Guidelines

### Tone
- Friendly and encouraging, not intimidating
- Use "you" and "your" to speak directly to the reader
- Acknowledge that concepts might be new and that's okay

### Structure for Each Page
1. **What Is This?** - Plain English explanation in 1-2 sentences
2. **Why Do I Need This?** - Real-world problem this solves
3. **How Does It Work?** - High-level concept explanation
4. **Step-by-Step Tutorial** - Hands-on walkthrough
5. **Complete Example** - Full working code
6. **Common Questions** - FAQ section
7. **Next Steps** - What to read next

### Code Examples
- Always show complete, runnable code
- Include import statements
- Add comments explaining non-obvious lines
- Show both the code AND the output/result

### Vocabulary
- Define technical terms on first use
- Provide analogies to familiar concepts
- Avoid jargon without explanation

## Pages to Rewrite

### Core Concepts (Read First)

#### 1. overview.mdx
**Current State**: Brief overview, assumes familiarity with service-oriented architecture
**Changes Needed**:
- Add "What is a Contract?" section explaining the concept
- Explain the problem it solves (multiple API protocols, type safety, etc.)
- Add analogy: contracts like ordering from a menu (you know what you're getting)
- Include a simple "before and after" showing complexity without vs with Contract
- Add visual diagram of the architecture

#### 2. quick-start.mdx
**Current State**: Code-heavy, minimal explanation
**Changes Needed**:
- Break into smaller, digestible steps
- Explain each line of code
- Add "What just happened?" after each step
- Include expected output at each stage
- Add troubleshooting section for common errors

#### 3. service.mdx
**Current State**: Technical explanation of method signatures
**Changes Needed**:
- Start with "What is a Service?" using real-world analogies
- Explain context.Context for beginners (like a conversation thread)
- Show method signature patterns with plain English explanations
- Add "Naming Your Methods" section with conventions
- Include validation examples and error handling

#### 4. register.mdx
**Current State**: API documentation style
**Changes Needed**:
- Explain what registration does and why it's needed
- Walk through the reflection process in simple terms
- Show what the Service struct contains after registration
- Include visual diagram of registration process
- Add debugging tips

#### 5. types.mdx
**Current State**: Technical JSON schema focus
**Changes Needed**:
- Explain "What are types?" for someone new to typed languages
- Show how Go types become JSON schemas
- Include visual mapping: Go struct -> JSON Schema -> Protocol
- Explain common type patterns (required fields, optional, arrays)
- Add section on custom types

#### 6. invoker.mdx
**Current State**: Internal implementation details
**Changes Needed**:
- Explain "What is an Invoker?" - like a phone operator connecting calls
- Show the call flow from request to response
- Explain why compiled invokers are faster than reflection
- Include when/why you'd customize the invoker

### Error Handling

#### 7. errors.mdx
**Current State**: Good coverage, somewhat technical
**Changes Needed**:
- Start with "Why error handling matters in APIs"
- Explain the Error struct with real scenarios
- Show the "golden path" - how errors flow from service to client
- Add decision tree: "Which error code should I use?"
- Include anti-patterns and best practices

#### 8. error-codes.mdx
**Current State**: Reference table, minimal explanation
**Changes Needed**:
- Add real-world scenario for each error code
- Explain HTTP/JSON-RPC/gRPC mapping rationale
- Include "When to use this" for each code
- Add examples of good error messages
- Create quick reference card at the end

### Architecture

#### 9. architecture.mdx
**Current State**: Good diagrams, technical language
**Changes Needed**:
- Add "Big Picture" section with layered explanation
- Explain each component with real-world analogies
- Show request lifecycle step-by-step with visuals
- Add "Why this design?" explaining decisions
- Include comparison with alternatives

#### 10. middleware.mdx
**Current State**: Good examples, assumes middleware knowledge
**Changes Needed**:
- Explain "What is Middleware?" with layered analogy (security checkpoints)
- Show the wrapping concept visually
- Build up from simple to complex middleware
- Include common patterns (logging, auth, rate limiting)
- Add ordering explanation with diagrams

### Transports

#### 11. transports-overview.mdx
**Current State**: Good comparison table
**Changes Needed**:
- Add "What is a Transport?" explanation
- Explain when to use each transport with decision flowchart
- Include real-world use cases for each
- Show how the same service works across all transports
- Add comparison of client code for each

#### 12. rest.mdx
**Current State**: Technical REST focus
**Changes Needed**:
- Explain REST for beginners (HTTP verbs, resources, etc.)
- Walk through verb convention mapping with examples
- Show complete request/response cycle
- Include curl examples with explanations
- Add section on customizing REST behavior

#### 13. jsonrpc.mdx
**Current State**: Protocol-focused
**Changes Needed**:
- Explain "What is JSON-RPC?" vs REST
- Walk through request/response format
- Explain batching with real use case
- Show notifications and when to use them
- Include debugging tips

#### 14. openapi.mdx
**Current State**: Spec generation focused
**Changes Needed**:
- Explain "What is OpenAPI?" and why it matters
- Show how to view generated spec
- Walk through spec structure
- Include Swagger UI setup guide
- Add client generation examples

#### 15. mcp.mdx
**Current State**: Good MCP explanation
**Changes Needed**:
- Expand "What is MCP?" for AI newcomers
- Explain tools concept with examples
- Walk through Claude Desktop integration
- Show tool discovery and calling
- Include security considerations

#### 16. trpc.mdx
**Current State**: TypeScript-focused
**Changes Needed**:
- Explain "What is tRPC?" vs other transports
- Show response envelope format
- Walk through metadata endpoint
- Include TypeScript client example
- Add comparison with original tRPC

### Practical

#### 17. testing.mdx
**Current State**: Good coverage
**Changes Needed**:
- Add "Why test your services?" motivation
- Show testing pyramid (unit -> integration -> e2e)
- Walk through testing each layer
- Include test helpers and utilities
- Add debugging failed tests section

#### 18. client-generation.mdx
**Current State**: Tool references
**Changes Needed**:
- Explain "What is client generation?"
- Walk through OpenAPI generation step-by-step
- Show TypeScript client from scratch
- Include multiple languages (Go, Python, TypeScript)
- Add best practices for generated clients

### Reference

#### 19. api-reference.mdx
**Current State**: Good API docs
**Changes Needed**:
- Add "How to read this reference" guide
- Group by use case (core, errors, transports)
- Include more examples for each function
- Add cross-references between related items
- Create quick lookup index

## Implementation Order

### Phase 1: Foundation (Core Understanding)
1. overview.mdx - Sets the stage for everything
2. quick-start.mdx - Gets readers hands-on immediately
3. architecture.mdx - Explains the big picture

### Phase 2: Core Concepts
4. service.mdx - How to write services
5. register.mdx - How registration works
6. types.mdx - Understanding type system

### Phase 3: Error Handling
7. errors.mdx - Error handling patterns
8. error-codes.mdx - Error code reference

### Phase 4: Transports
9. transports-overview.mdx - Transport comparison
10. rest.mdx - REST transport
11. jsonrpc.mdx - JSON-RPC transport
12. openapi.mdx - OpenAPI generation
13. mcp.mdx - MCP for AI
14. trpc.mdx - tRPC transport

### Phase 5: Advanced & Reference
15. invoker.mdx - Invoker internals
16. middleware.mdx - Middleware patterns
17. testing.mdx - Testing guide
18. client-generation.mdx - Client generation
19. api-reference.mdx - Complete reference

## Success Criteria

A successful rewrite means:
1. Someone with Go basics can build an API using Contract in 30 minutes
2. Each page can stand alone while linking to related concepts
3. Code examples copy-paste and run without modification
4. Common questions are answered before they're asked
5. Readers understand "why" not just "how"

## Template Structure

Each page should follow this structure:

```mdx
---
title: "Page Title"
description: "One sentence description for SEO"
---

# Page Title

Brief 1-2 sentence overview of what this page covers.

## What Is [Concept]?

Plain English explanation. Use analogies. Define terms.

## Why Do You Need This?

Real-world problem this solves. Before/after comparison if applicable.

## How It Works

Conceptual explanation with diagrams if helpful.

## Step-by-Step Guide

### Step 1: [Action]

Explanation of what we're doing.

```go
// Code with comments
```

What this code does in plain English.

### Step 2: [Action]

Continue the pattern...

## Complete Example

Full working code that readers can copy and run.

## Common Questions

### Q: Frequently asked question?

A: Clear answer.

### Q: Another common question?

A: Clear answer.

## What's Next?

- [Related Page 1](/contract/page1) - Brief description
- [Related Page 2](/contract/page2) - Brief description
```
