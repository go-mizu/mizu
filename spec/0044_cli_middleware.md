# 0044: CLI Middleware Command

## Summary

Add a `mizu middleware` command to the CLI that helps developers discover, explore, and use the 100 available middlewares. The command provides listing, search, and detailed documentation for each middleware with copy-ready code snippets.

## Background

Mizu includes 100 middleware packages covering authentication, security, caching, rate limiting, logging, and more. Currently, developers must:

1. Browse the `middlewares/` directory manually
2. Read source code or doc.go files
3. Look up import paths and function signatures

This spec adds a CLI command that makes middleware discovery intuitive and provides immediate copy-paste code snippets.

## Design Goals

1. **Discoverability** - Find middlewares by name, category, or keyword
2. **Immediate Usability** - Copy-ready code snippets
3. **Beginner-Friendly** - Clear descriptions and examples
4. **Scriptable** - JSON output for tooling integration

## Commands

### Command Structure

```
mizu middleware <subcommand> [args] [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `ls` | List all middlewares (with optional category filter) |
| `show` | Show detailed information about a middleware |

### mizu middleware ls

List all available middlewares, optionally filtered by category.

**Usage:**
```bash
mizu middleware ls [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-c, --category <name>` | Filter by category |
| `--json` | Output as JSON |

**Example Output (Human):**

```
SECURITY (6)
  basicauth       HTTP Basic authentication
  bearerauth      Bearer token authentication
  helmet          Security headers
  cors            Cross-Origin Resource Sharing
  csrf            CSRF protection
  jwt             JSON Web Token validation

LOGGING (4)
  logger          HTTP request logging
  requestlog      Request body logging
  responselog     Response body logging
  trace           Distributed tracing

RATE LIMITING (5)
  ratelimit       Token bucket rate limiting
  throttle        Simple request throttling
  adaptive        Adaptive rate limiting
  bulkhead        Bulkhead pattern
  concurrency     Concurrency limiting

... (more categories)

100 middlewares available. Use 'mizu middleware show <name>' for details.
```

**Example Output (JSON):**

```json
{
  "categories": [
    {
      "name": "security",
      "middlewares": [
        {"name": "basicauth", "description": "HTTP Basic authentication"},
        {"name": "helmet", "description": "Security headers"}
      ]
    }
  ],
  "total": 100
}
```

### mizu middleware show

Show detailed information about a specific middleware.

**Usage:**
```bash
mizu middleware show <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |

**Example Output (Human):**

```
HELMET

Security headers middleware for Mizu web applications.

CATEGORY: security

IMPORT:
  github.com/go-mizu/mizu/middlewares/helmet

QUICK START:
  // Use with sensible defaults
  app.Use(helmet.Default())

CUSTOM OPTIONS:
  app.Use(helmet.New(helmet.Options{
      XFrameOptions:       "DENY",
      XContentTypeOptions: true,
      ReferrerPolicy:      "no-referrer",
  }))

CONVENIENCE FUNCTIONS:
  helmet.Default()                     Recommended security headers
  helmet.ContentSecurityPolicy(policy) Set CSP header
  helmet.XFrameOptions(value)          Set X-Frame-Options
  helmet.StrictTransportSecurity(...)  Set HSTS header

OPTIONS:
  ContentSecurityPolicy       string   Content-Security-Policy header value
  XFrameOptions               string   X-Frame-Options (DENY, SAMEORIGIN)
  XContentTypeOptions         bool     Enable X-Content-Type-Options: nosniff
  ReferrerPolicy              string   Referrer-Policy header value
  StrictTransportSecurity     *HSTSOptions  HSTS configuration
  PermissionsPolicy           string   Permissions-Policy header value
  ... (more options)

RELATED:
  cors, csrf, nonce, secure
```

## Categories

Middlewares are organized into these categories:

| Category | Description | Count |
|----------|-------------|-------|
| `security` | Authentication and security headers | 12 |
| `logging` | Request/response logging and tracing | 8 |
| `ratelimit` | Rate limiting and flow control | 7 |
| `cache` | Caching headers and strategies | 6 |
| `encoding` | Compression and content encoding | 4 |
| `resilience` | Circuit breaker, retry, timeout | 6 |
| `routing` | URL rewriting, redirects, method override | 5 |
| `content` | Static files, favicon, SPA | 5 |
| `api` | JSON-RPC, GraphQL, validation | 6 |
| `session` | Sessions, cookies, state | 4 |
| `observability` | Metrics, profiling, tracing | 6 |
| `misc` | Other utilities | 31 |

## Middleware Metadata

Each middleware has metadata extracted from its package:

```go
type middlewareInfo struct {
    Name        string   // Package name (e.g., "helmet")
    Description string   // Short description
    Category    string   // Category name
    Import      string   // Full import path
    Functions   []string // Exported functions (New, Default, etc.)
    Options     []option // Options struct fields
    Related     []string // Related middleware names
}
```

Metadata is embedded at build time from doc.go files and source analysis.

## Implementation

### Files to Create

```
cli/
├── cmd_middleware.go      # Command implementation
├── middleware_data.go     # Embedded middleware metadata
└── middleware_data_test.go
```

### cmd_middleware.go Structure

```go
package cli

import (
    "flag"
    "fmt"
    "sort"
    "strings"
)

type middlewareFlags struct {
    category string
}

func runMiddleware(args []string, gf *globalFlags) int {
    out := newOutput(gf.json, gf.quiet, gf.noColor, gf.verbose)

    if len(args) == 0 {
        usageMiddleware()
        return exitUsage
    }

    subcmd := args[0]
    subargs := args[1:]

    switch subcmd {
    case "ls", "list":
        return runMiddlewareLs(subargs, gf, out)
    case "show":
        return runMiddlewareShow(subargs, gf, out)
    default:
        out.errorf("error: unknown subcommand %q\n", subcmd)
        return exitUsage
    }
}

func runMiddlewareLs(args []string, gf *globalFlags, out *output) int {
    mf := &middlewareFlags{}
    fs := flag.NewFlagSet("middleware ls", flag.ContinueOnError)
    fs.StringVar(&mf.category, "category", "", "Filter by category")
    fs.StringVar(&mf.category, "c", "", "Filter by category (shorthand)")

    if err := fs.Parse(args); err != nil {
        if err == flag.ErrHelp {
            usageMiddlewareLs()
            return exitOK
        }
        return exitUsage
    }

    middlewares := getMiddlewares()
    if mf.category != "" {
        middlewares = filterByCategory(middlewares, mf.category)
    }

    if out.json {
        return printMiddlewaresJSON(out, middlewares)
    }
    return printMiddlewaresHuman(out, middlewares)
}

func runMiddlewareShow(args []string, gf *globalFlags, out *output) int {
    if len(args) == 0 {
        out.errorf("error: middleware name required\n")
        return exitUsage
    }

    name := strings.ToLower(args[0])
    mw := findMiddleware(name)
    if mw == nil {
        out.errorf("error: unknown middleware %q\n", name)
        out.errorf("Run 'mizu middleware ls' to see available middlewares.\n")
        return exitError
    }

    if gf.json {
        return printMiddlewareJSON(out, mw)
    }
    return printMiddlewareHuman(out, mw)
}
```

### middleware_data.go

Contains embedded metadata for all 100 middlewares:

```go
package cli

//go:generate go run ../scripts/gen_middleware_data.go

var middlewares = []middlewareInfo{
    {
        Name:        "helmet",
        Description: "Security headers middleware",
        Category:    "security",
        Import:      "github.com/go-mizu/mizu/middlewares/helmet",
        Functions:   []string{"New", "Default", "ContentSecurityPolicy", "XFrameOptions"},
        QuickStart:  "app.Use(helmet.Default())",
        Related:     []string{"cors", "csrf", "nonce"},
    },
    // ... 99 more
}
```

### Registration in cmd_root.go

Add to commands slice:

```go
var commands = []*command{
    {name: "new", short: "Create a new project from a template", run: runNew, usage: usageNew},
    {name: "dev", short: "Run the current project in development mode", run: runDev, usage: usageDev},
    {name: "contract", short: "Work with service contracts", run: runContract, usage: usageContract},
    {name: "middleware", short: "Explore available middlewares", run: runMiddleware, usage: usageMiddleware},
    {name: "version", short: "Print version information", run: runVersion, usage: usageVersion},
}
```

## Documentation Updates

### docs/cli/overview.mdx

Add middleware command to the commands table:

```markdown
| Command | What it does |
|---------|--------------|
| `mizu new` | Creates a new project from a template |
| `mizu dev` | Runs your project in development mode |
| `mizu contract` | Works with service contracts (for API projects) |
| `mizu middleware` | Explore available middlewares |
| `mizu version` | Shows the CLI version |
```

### docs/cli/middleware.mdx (New)

Create comprehensive documentation for the middleware command:

**Sections:**
- Overview - What the command does
- Quick Examples - Common usage patterns
- Subcommands - ls and show detailed
- Categories - Full category list
- JSON Output - Schema for scripting
- Common Workflows - Finding and using middlewares

### All CLI Docs Updates

Update all pages to:
1. Remove H1 headers (frontmatter title is sufficient)
2. Use shorter, simpler titles (Overview instead of "API Template Overview")
3. Add more beginner explanations
4. Ensure consistent formatting

## Files Changed

### New Files

| File | Description |
|------|-------------|
| `cli/cmd_middleware.go` | Command implementation |
| `cli/middleware_data.go` | Embedded middleware metadata |
| `docs/cli/middleware.mdx` | Command documentation |

### Modified Files

| File | Changes |
|------|---------|
| `cli/cmd_root.go` | Register middleware command |
| `docs/cli/overview.mdx` | Remove H1, add middleware to commands table |
| `docs/cli/installation.mdx` | Remove H1, simplify |
| `docs/cli/new.mdx` | Remove H1, simplify |
| `docs/cli/dev.mdx` | Remove H1, simplify |
| `docs/cli/contract.mdx` | Remove H1, simplify |
| `docs/cli/version.mdx` | Remove H1, simplify |
| `docs/cli/templates.mdx` | Remove H1, simplify |
| `docs/cli/global-flags.mdx` | Remove H1, simplify |
| `docs/cli/exit-codes.mdx` | Remove H1, simplify |
| `docs/cli/json-output.mdx` | Remove H1, simplify |
| `docs/cli/minimal/overview.mdx` | Remove H1, simplify |
| `docs/cli/minimal/structure.mdx` | Remove H1, simplify |
| `docs/cli/minimal/tutorial.mdx` | Remove H1, simplify |
| `docs/cli/api/overview.mdx` | Remove H1, simplify |
| `docs/cli/api/structure.mdx` | Remove H1, simplify |
| `docs/cli/api/tutorial.mdx` | Remove H1, simplify |
| `docs/cli/contract-template/overview.mdx` | Remove H1, simplify |
| `docs/cli/contract-template/structure.mdx` | Remove H1, simplify |
| `docs/cli/contract-template/tutorial.mdx` | Remove H1, simplify |
| `docs/cli/web/overview.mdx` | Remove H1, simplify |
| `docs/cli/web/structure.mdx` | Remove H1, simplify |
| `docs/cli/web/tutorial.mdx` | Remove H1, simplify |
| `docs/cli/live/overview.mdx` | Remove H1, simplify |
| `docs/cli/live/structure.mdx` | Remove H1, simplify |
| `docs/cli/live/tutorial.mdx` | Remove H1, simplify |
| `docs/cli/sync/overview.mdx` | Remove H1, simplify |
| `docs/cli/sync/structure.mdx` | Remove H1, simplify |
| `docs/cli/sync/tutorial.mdx` | Remove H1, simplify |

## Implementation Order

1. Write this spec
2. Create `cli/middleware_data.go` with all middleware metadata
3. Create `cli/cmd_middleware.go` with command implementation
4. Register command in `cli/cmd_root.go`
5. Create `docs/cli/middleware.mdx`
6. Update `docs/cli/overview.mdx`
7. Update all other CLI docs (remove H1, simplify titles)
8. Run tests

## Testing

```bash
# Test listing all middlewares
mizu middleware ls

# Test category filter
mizu middleware ls -c security

# Test JSON output
mizu middleware ls --json

# Test show command
mizu middleware show helmet
mizu middleware show cors
mizu middleware show ratelimit

# Test JSON output for show
mizu middleware show helmet --json

# Test error handling
mizu middleware show nonexistent
mizu middleware unknowncmd

# Run unit tests
go test ./cli/... -run TestMiddleware
```
