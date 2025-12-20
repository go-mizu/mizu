# Spec 0106: Core Documentation Improvements

## Overview

This spec outlines the comprehensive plan for improving the Core documentation tab (formerly "Guides"). The goal is to make every page beginner-friendly with detailed explanations, practical examples, and clear progression from simple to complex concepts.

## Current State Analysis

### Pages in Core Tab

The Core tab contains 18 pages across 3 groups:

**Overview Group (5 pages)**
1. `overview/intro` - Introduction to Mizu
2. `overview/why` - Why Mizu exists (philosophy)
3. `overview/features` - Feature list
4. `overview/use-cases` - When to use Mizu
5. `overview/roadmap` - Development roadmap

**Getting Started Group (2 pages)**
6. `get-started/quick-start` - First Mizu application
7. `get-started/deployment` - Production deployment

**Concepts Group (11 pages)**
8. `concepts/overview` - Concepts navigation
9. `concepts/app` - App lifecycle
10. `concepts/routing` - URL routing
11. `concepts/handler` - Handler functions
12. `concepts/context` - Request context
13. `concepts/request` - Reading request data
14. `concepts/response` - Sending responses
15. `concepts/static` - Static file serving
16. `concepts/logging` - Structured logging
17. `concepts/error` - Error handling
18. `concepts/middleware` - Middleware pattern

### Quality Assessment

| Page | Current Quality | Needs Improvement |
|------|-----------------|-------------------|
| overview/intro | Excellent | Minor tweaks |
| overview/why | Excellent | Minor tweaks |
| overview/features | Very Good | Add more depth |
| overview/use-cases | Excellent | Minor tweaks |
| overview/roadmap | Good | Expand content |
| get-started/quick-start | Excellent | Minor tweaks |
| get-started/deployment | Excellent | Add cloud examples |
| concepts/overview | Good | Make more visual |
| concepts/app | Excellent | Minor tweaks |
| concepts/routing | Very Good | Add edge cases |
| concepts/handler | Excellent | Minor tweaks |
| concepts/context | Good | **Expand significantly** |
| concepts/request | Very Good | Add more examples |
| concepts/response | Excellent | Minor tweaks |
| concepts/static | Good | **Expand significantly** |
| concepts/logging | Excellent | Minor tweaks |
| concepts/error | Good | **Expand significantly** |
| concepts/middleware | Excellent | Minor tweaks |

## Improvement Plan by Page

---

### 1. overview/intro.mdx

**Current State:** Excellent introduction with water metaphor and comparison table.

**Improvements:**
- Add a visual "architecture at a glance" diagram showing request flow
- Include version badge and Go version requirement prominently
- Add "5-minute read" indicator
- Expand the quick taste example with inline comments explaining each line
- Add a "What makes Mizu different" callout box

**Target Length:** 400-500 lines

---

### 2. overview/why.mdx

**Current State:** Strong philosophy explanation with code comparisons.

**Improvements:**
- Add "Before/After" visual comparisons
- Include real developer testimonials or use cases
- Add decision tree: "Is Mizu right for you?"
- Expand the "What Mizu is NOT" section
- Add links to related concepts

**Target Length:** 350-400 lines

---

### 3. overview/features.mdx

**Current State:** Good feature list but could be more comprehensive.

**Improvements:**
- Add feature availability badges (stable, beta, planned)
- Include mini code snippets for each feature
- Add comparison with other frameworks per feature
- Create feature matrix table
- Add performance characteristics where relevant
- Include configuration options summary

**Target Length:** 500-600 lines

---

### 4. overview/use-cases.mdx

**Current State:** Excellent practical scenarios.

**Improvements:**
- Add more real-world company examples (anonymized)
- Include architecture diagrams for each use case
- Add complexity ratings for each scenario
- Include "time to production" estimates
- Add links to corresponding templates

**Target Length:** 450-500 lines

---

### 5. overview/roadmap.mdx

**Current State:** Brief roadmap.

**Improvements:**
- Add visual timeline
- Include GitHub milestone links
- Add "Want to contribute?" section
- Include RFC/proposal process explanation
- Add community voting on features
- Include version history summary

**Target Length:** 300-350 lines

---

### 6. get-started/quick-start.mdx

**Current State:** Excellent step-by-step tutorial.

**Improvements:**
- Add terminal recordings/animations for commands
- Include troubleshooting tips for common issues
- Add checkpoint summaries after each step
- Include IDE setup recommendations (VS Code, GoLand)
- Add "Common mistakes" warnings
- Include performance tips

**Target Length:** 600-700 lines

---

### 7. get-started/deployment.mdx

**Current State:** Excellent multi-platform guide.

**Improvements:**
- Add cloud platform sections:
  - AWS (ECS, Lambda, EC2)
  - Google Cloud (Cloud Run, GKE)
  - Azure (Container Apps)
  - DigitalOcean (App Platform)
  - Railway, Fly.io, Render
- Include cost comparison table
- Add monitoring and observability setup
- Include scaling strategies
- Add CI/CD pipeline examples

**Target Length:** 800-900 lines

---

### 8. concepts/overview.mdx

**Current State:** Basic navigation page.

**Improvements:**
- Add visual concept map showing relationships
- Include learning path recommendations
- Add "Start here if..." recommendations
- Include complexity indicators for each concept
- Add estimated reading time for each page
- Create beginner vs advanced tracks

**Target Length:** 250-300 lines

---

### 9. concepts/app.mdx

**Current State:** Excellent coverage of App lifecycle.

**Improvements:**
- Add lifecycle diagram (startup → running → shutdown)
- Include more configuration examples
- Add multi-app scenarios (main + admin server)
- Include testing with App
- Add common configuration patterns
- Include signal handling details

**Target Length:** 500-550 lines

---

### 10. concepts/routing.mdx

**Current State:** Very good routing explanation.

**Improvements:**
- Add routing precedence rules
- Include route conflict resolution
- Add advanced patterns (versioned APIs, localization)
- Include route debugging tips
- Add performance considerations for many routes
- Include route table visualization example
- Add regex and custom matchers

**Target Length:** 550-600 lines

---

### 11. concepts/handler.mdx

**Current State:** Excellent handler guide.

**Improvements:**
- Add handler testing strategies
- Include dependency injection patterns
- Add handler composition examples
- Include async handler patterns
- Add handler performance tips
- Include common handler architectures (clean, layered)

**Target Length:** 500-550 lines

---

### 12. concepts/context.mdx **[MAJOR EXPANSION NEEDED]**

**Current State:** Brief coverage of Ctx wrapper.

**Improvements:**
- Add comprehensive Ctx method reference
- Include context lifecycle diagram
- Add context value best practices
- Include request-scoped storage patterns
- Add timeout handling examples
- Include cancellation propagation
- Add context debugging tips
- Include performance considerations
- Add middleware context patterns
- Include testing with context

**Target Length:** 600-700 lines

**Key Sections to Add:**
1. What is Context? (conceptual explanation)
2. The Mizu Ctx wrapper
3. Accessing request and response
4. Storing values in context
5. Timeouts and cancellation
6. Context in middleware
7. Testing with context
8. Best practices
9. Common patterns
10. API reference

---

### 13. concepts/request.mdx

**Current State:** Very good coverage of request reading.

**Improvements:**
- Add request validation patterns
- Include file upload best practices
- Add streaming request body handling
- Include multipart form advanced usage
- Add request size limits and security
- Include content negotiation

**Target Length:** 550-600 lines

---

### 14. concepts/response.mdx

**Current State:** Excellent comprehensive guide.

**Improvements:**
- Add response compression patterns
- Include conditional response (ETag, Last-Modified)
- Add response streaming for large files
- Include response buffering control
- Add response middleware patterns
- Include performance optimization tips

**Target Length:** 650-700 lines

---

### 15. concepts/static.mdx **[MAJOR EXPANSION NEEDED]**

**Current State:** Brief static files coverage.

**Improvements:**
- Add SPA fallback patterns
- Include cache control strategies
- Add embedded files best practices
- Include CDN integration
- Add hot reload in development
- Include security (path traversal prevention)
- Add compression configuration
- Include virtual file systems
- Add custom file handlers
- Include performance tuning

**Target Length:** 500-550 lines

**Key Sections to Add:**
1. Serving static files basics
2. File system options (local, embed, memory)
3. Route configuration
4. Caching and cache busting
5. Compression (gzip, brotli)
6. SPA support
7. Security considerations
8. Performance optimization
9. CDN integration
10. Development vs production

---

### 16. concepts/logging.mdx

**Current State:** Excellent structured logging guide.

**Improvements:**
- Add log aggregation integration (ELK, Loki, CloudWatch)
- Include log sampling for high-traffic
- Add correlation ID patterns
- Include sensitive data redaction
- Add log level best practices by environment
- Include structured logging with context

**Target Length:** 550-600 lines

---

### 17. concepts/error.mdx **[MAJOR EXPANSION NEEDED]**

**Current State:** Basic error handling coverage.

**Improvements:**
- Add custom error types
- Include error wrapping and unwrapping
- Add error categorization (client vs server)
- Include HTTP error responses best practices
- Add error logging patterns
- Include error recovery strategies
- Add error testing
- Include API error format standards (RFC 7807)
- Add error monitoring integration
- Include graceful degradation patterns

**Target Length:** 600-650 lines

**Key Sections to Add:**
1. Error handling philosophy
2. Returning errors from handlers
3. The ErrorHandler
4. Custom error types
5. HTTP error responses
6. Error wrapping with context
7. Panic recovery
8. Error logging
9. Error monitoring
10. Testing error scenarios
11. Best practices

---

### 18. concepts/middleware.mdx

**Current State:** Excellent comprehensive guide.

**Improvements:**
- Add middleware testing strategies
- Include middleware ordering analyzer
- Add conditional middleware patterns
- Include middleware performance profiling
- Add common middleware recipes
- Include third-party middleware integration

**Target Length:** 650-700 lines

---

## Implementation Priority

### Phase 1: Critical Expansions
1. concepts/context.mdx - Most underdeveloped core concept
2. concepts/error.mdx - Essential for production use
3. concepts/static.mdx - Common use case needs better coverage

### Phase 2: Important Additions
4. get-started/deployment.mdx - Add cloud platforms
5. concepts/routing.mdx - Add advanced patterns
6. concepts/middleware.mdx - Add testing section

### Phase 3: Polish and Enhancement
7. overview/intro.mdx - Add diagrams
8. concepts/overview.mdx - Add visual concept map
9. concepts/request.mdx - Add validation
10. All other pages - Minor improvements

## Writing Guidelines for Beginners

### Tone
- Friendly but professional
- Assume reader is new to Go web development
- Explain "why" before "how"
- Use analogies to explain concepts

### Structure
- Start with a clear introduction
- Use progressive complexity
- Include runnable examples
- End with summary and next steps

### Code Examples
- Keep examples short (10-30 lines)
- Include inline comments
- Show both simple and complete versions
- Always show expected output

### Visual Elements
- Use callout boxes for tips and warnings
- Include diagrams for complex flows
- Use tables for comparisons
- Add code tabs for multiple approaches

## Success Metrics

- Each page should be readable in 5-15 minutes
- All code examples should be copy-paste runnable
- Each concept should link to related concepts
- Pages should answer common beginner questions
- Each page should include "What's Next" section
