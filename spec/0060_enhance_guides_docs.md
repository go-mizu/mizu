# Spec 0060: Enhance Core Framework Documentation

## Overview

Enhance all documentation files in `docs/overview/*.mdx`, `docs/get-started/*.mdx`, and `docs/concepts/*.mdx` to:

1. **Improve DX** - Write for absolute beginners with clear explanations
2. **Fix accuracy** - Ensure documentation matches the latest core Mizu code
3. **Add practical usage** - Include real-world examples and common patterns

## Source Code Audit

Audited files: `app.go`, `router.go`, `context.go`, `logger.go`

### Documentation-Code Mismatches Found

| File | Issue | Fix |
|------|-------|-----|
| `concepts/app.mdx` | Shows `HealthzHandler()` | Code has `LivezHandler()` and `ReadyzHandler()` |
| `concepts/app.mdx` | Shows `WithPreShutdownDelay()` option | Not in current code - only `ShutdownTimeout` field |
| `concepts/app.mdx` | Shows `WithShutdownTimeout()` option | Not an option - set `app.ShutdownTimeout` directly |
| `overview/features.mdx` | Lists `app.UseFirst()` | Not in current code |
| `concepts/response.mdx` | Shows `c.File("path")` | Actual signature: `c.File(code int, path string)` |
| `concepts/response.mdx` | Shows `c.Download("path", "name")` | Actual signature: `c.Download(code int, path, name string)` |
| `concepts/middleware.mdx` | Shows `recover.New()` as built-in | These are separate middleware packages, not core |

### Actual API from Source Code

**App Configuration (app.go:29-46)**
```go
type App struct {
    *Router
    ShutdownTimeout time.Duration  // default: 15s
}

func New() *App  // No options pattern currently
```

**Health Handlers (app.go:55-79)**
```go
app.LivezHandler()   // Returns 200 always (for liveness probes)
app.ReadyzHandler()  // Returns 200 or 503 during shutdown (for readiness probes)
```

**Context Methods (context.go)**
```go
// File serving - both take code as first parameter
c.File(code int, path string) error
c.Download(code int, path, name string) error
```

**Logger Options (logger.go:31-44)**
```go
type LoggerOptions struct {
    Mode            Mode           // Auto, Dev, Prod
    Color           bool
    Logger          *slog.Logger
    RequestIDHeader string
    RequestIDGen    func() string
    UserAgent       bool
    Output          io.Writer
    TraceExtractor  func(ctx context.Context) (traceID, spanID string, sampled bool)
}
```

## Writing Style Guidelines

### For Absolute Beginners

1. **Explain concepts before code** - What is it? Why do we need it?
2. **Define terminology** - When introducing terms, explain what they mean
3. **Use analogies** - Connect technical concepts to familiar things
4. **Show complete examples** - Include imports and full runnable code
5. **Explain what happens** - Describe the flow and outcome
6. **Address common questions** - Anticipate "why?" and "what if?"

### Before (Current Style)
```markdown
### Create an app

Every Mizu project begins by creating an `App` instance.

\```go
app := mizu.New()
app.Get("/", home)
app.Listen(":3000")
\```
```

### After (Beginner-Friendly Style)
```markdown
### Create an app

An `App` is your web server. It handles incoming HTTP requests and sends responses back to clients (browsers, mobile apps, other servers).

When you create an app with `mizu.New()`, you get:
- A router that maps URLs to your handler functions
- Built-in request logging for debugging
- Graceful shutdown that lets active requests finish
- Panic recovery to prevent crashes

\```go
package main

import "github.com/go-mizu/mizu"

func main() {
    // Create a new web server
    app := mizu.New()

    // Define what happens when someone visits "/"
    app.Get("/", func(c *mizu.Ctx) error {
        return c.Text(200, "Hello!")
    })

    // Start listening for requests on port 3000
    // The server runs until you press Ctrl+C
    app.Listen(":3000")
}
\```

When you run this code:
1. The server starts on `localhost:3000`
2. Visiting that URL in a browser triggers your handler
3. The handler sends "Hello!" as plain text with status 200 (OK)
```

## Files to Enhance

### Overview Section (docs/overview/)

| File | Priority | Key Enhancements |
|------|----------|------------------|
| `intro.mdx` | High | Add what Mizu is for beginners, explain each bullet point |
| `quick-start.mdx` | High | Show complete runnable example with explanations |
| `why.mdx` | Medium | Add concrete comparisons, explain standard library concepts |
| `features.mdx` | High | Fix UseFirst mention, explain each feature category |
| `use-cases.mdx` | Medium | Add more detailed code examples for each use case |
| `roadmap.mdx` | Low | Keep as is |

### Get Started Section (docs/get-started/)

| File | Priority | Key Enhancements |
|------|----------|------------------|
| `quick-start.mdx` | High | Add what each line does, explain Go module setup |
| `deployment.mdx` | Medium | Add explanation of each deployment step |

### Concepts Section (docs/concepts/)

| File | Priority | Key Enhancements |
|------|----------|------------------|
| `overview.mdx` | Medium | Add brief explanations of each concept |
| `app.mdx` | High | Fix health handler names, fix configuration options |
| `routing.mdx` | High | Explain ServeMux patterns, path matching rules |
| `handler.mdx` | High | Explain error return pattern, signature anatomy |
| `context.mdx` | Medium | Explain request/response lifecycle |
| `request.mdx` | High | Add validation examples, explain each input type |
| `response.mdx` | High | Fix File/Download signatures, add streaming examples |
| `middleware.mdx` | High | Fix built-in middleware claims, explain middleware flow |
| `logging.mdx` | Medium | Fix LoggerOptions fields, add practical examples |
| `error.mdx` | Medium | Add custom error types, pattern examples |
| `static.mdx` | Low | Add cache header examples |

## Implementation Plan

### Phase 1: Fix Critical Issues
1. Fix all documentation-code mismatches
2. Update method signatures to match source code
3. Remove references to non-existent features

### Phase 2: Enhance Overview Section
1. Rewrite intro.mdx for beginners
2. Enhance quick-start.mdx with detailed explanations
3. Update features.mdx with accurate information

### Phase 3: Enhance Get Started Section
1. Enhance quick-start.mdx with step-by-step guidance
2. Improve deployment.mdx explanations

### Phase 4: Enhance Concepts Section
1. Fix and enhance app.mdx
2. Enhance routing.mdx with pattern explanations
3. Enhance handler.mdx with error handling patterns
4. Fix and enhance response.mdx
5. Fix and enhance middleware.mdx
6. Enhance remaining concept pages

## Success Criteria

- [ ] All code examples compile and run correctly
- [ ] All method signatures match source code
- [ ] Each concept is explained for absolute beginners
- [ ] Each page includes practical, runnable examples
- [ ] Terminology is defined when first introduced
- [ ] The "why" is explained, not just the "what"
