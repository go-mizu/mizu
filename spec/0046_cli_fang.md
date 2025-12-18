# Spec 0046: CLI Enhancement with Fang and Embedded Documentation

## Status
**Draft**

## Overview
Upgrade the Mizu CLI from manual flag parsing to a modern developer experience using [charmbracelet/fang](https://github.com/charmbracelet/fang) with [Cobra](https://github.com/spf13/cobra). Add embedded markdown documentation accessible via `--md` flag and rendered with [charmbracelet/glow](https://github.com/charmbracelet/glow), plus automatic man page generation.

## Motivation

### Current State
- Manual flag parsing in `cmd_root.go` using stdlib `flag` package
- Basic ANSI color support via custom `output` type
- Usage text hardcoded in `usage*()` functions
- No shell completions
- No man pages
- No markdown documentation

### Problems
1. **Inconsistent UX**: Manual flag parsing leads to inconsistent error handling and help formatting
2. **Limited Discoverability**: No markdown docs or man pages for offline reference
3. **Missing Features**: No shell completions, no automatic version flag handling
4. **Maintenance Burden**: Each command duplicates usage string formatting logic
5. **Not Modern DX**: CLI lacks styling and polish expected from modern Go tools

### Goals
1. Migrate to Fang/Cobra for consistent command structure and styling
2. Add `--md` flag to display markdown help (rendered via glow)
3. Embed markdown documentation for offline access
4. Generate man pages automatically
5. Add shell completion support
6. Improve error messages with styled output
7. Maintain backward compatibility with existing flags

## Architecture

### New Directory Structure
```
cmd/
├── go.mod                    # CLI module (add fang, cobra, glow dependencies)
├── cli/
│   ├── root.go               # Fang/Cobra root command setup
│   ├── new.go                # `mizu new` command
│   ├── dev.go                # `mizu dev` command
│   ├── contract.go           # `mizu contract` command group
│   ├── middleware.go         # `mizu middleware` command group
│   ├── version.go            # `mizu version` command
│   ├── docs.go               # Documentation helpers and --md handler
│   ├── output.go             # Styled output (migrated from io.go)
│   ├── format.go             # Table formatting (keep existing)
│   ├── ...                   # Other helper files
│   └── docs/                 # Embedded documentation
│       └── embed.go          # go:embed directives
├── docs/                     # Documentation source files
│   ├── mizu.md               # Main CLI manual (human-friendly)
│   ├── mizu.1                # Man page (roff format)
│   └── commands/
│       ├── new.md            # `mizu new` documentation
│       ├── new.1             # `mizu new` man page
│       ├── dev.md            # `mizu dev` documentation
│       ├── dev.1             # `mizu dev` man page
│       ├── contract.md       # `mizu contract` documentation
│       ├── contract.1        # `mizu contract` man page
│       ├── middleware.md     # `mizu middleware` documentation
│       ├── middleware.1      # `mizu middleware` man page
│       └── version.md        # `mizu version` documentation
└── mizu/
    └── main.go               # Entry point
```

### Dependencies to Add (cmd/go.mod)
```go
require (
    github.com/charmbracelet/fang v0.x.x
    github.com/charmbracelet/glow v2.x.x
    github.com/charmbracelet/glamour v0.x.x  // Markdown rendering
    github.com/charmbracelet/lipgloss v1.x.x // Styling
    github.com/spf13/cobra v1.8.x            // CLI framework (fang dependency)
    github.com/muesli/mango v0.x.x           // Man page generation
    github.com/muesli/mango-cobra v1.x.x     // Cobra integration for mango
)
```

## Implementation Plan

### Phase 1: Core Infrastructure

#### 1.1 Create Root Command with Fang

**File: `cmd/cli/root.go`**
```go
package cli

import (
    "context"
    "os"

    "github.com/charmbracelet/fang"
    "github.com/spf13/cobra"
)

// Version information set at build time
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

// Global flags available to all commands
type GlobalFlags struct {
    JSON    bool
    NoColor bool
    Quiet   bool
    Verbose int
    MD      bool // New: print markdown help
}

var globalFlags = &GlobalFlags{}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
    Use:   "mizu",
    Short: "Project toolkit for the go-mizu framework",
    Long: `mizu is the project toolkit for the go-mizu framework.

It provides commands for creating new projects, running development servers,
working with service contracts, and exploring available middlewares.`,
    SilenceUsage:  true,
    SilenceErrors: true,
}

func init() {
    // Global flags
    rootCmd.PersistentFlags().BoolVar(&globalFlags.JSON, "json", false, "Emit machine readable output")
    rootCmd.PersistentFlags().BoolVar(&globalFlags.NoColor, "no-color", false, "Disable color output")
    rootCmd.PersistentFlags().BoolVarP(&globalFlags.Quiet, "quiet", "q", false, "Reduce output (errors only)")
    rootCmd.PersistentFlags().CountVarP(&globalFlags.Verbose, "verbose", "v", "Increase verbosity (repeatable)")
    rootCmd.PersistentFlags().BoolVar(&globalFlags.MD, "md", false, "Print help as rendered markdown")

    // Register subcommands
    rootCmd.AddCommand(newCmd)
    rootCmd.AddCommand(devCmd)
    rootCmd.AddCommand(contractCmd)
    rootCmd.AddCommand(middlewareCmd)
    rootCmd.AddCommand(versionCmd)
    rootCmd.AddCommand(docsCmd) // Hidden: `mizu docs` for markdown browsing
}

// Run executes the CLI
func Run() int {
    // Configure fang options
    opts := []fang.Option{
        fang.WithVersion(Version),
    }

    if err := fang.Execute(context.Background(), rootCmd, opts...); err != nil {
        return 1
    }
    return 0
}
```

#### 1.2 Markdown Help System

**File: `cmd/cli/docs.go`**
```go
package cli

import (
    "embed"
    "fmt"
    "io"
    "os"
    "strings"

    "github.com/charmbracelet/glamour"
    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
)

//go:embed docs/*.md docs/commands/*.md
var docsFS embed.FS

// printMarkdownHelp renders and prints markdown help for a command
func printMarkdownHelp(cmd *cobra.Command) error {
    name := cmd.Name()
    if cmd.Parent() != nil && cmd.Parent().Name() != "mizu" {
        name = cmd.Parent().Name() + "_" + name
    }

    // Try to find matching markdown file
    path := fmt.Sprintf("docs/commands/%s.md", name)
    content, err := docsFS.ReadFile(path)
    if err != nil {
        // Fall back to root docs
        path = "docs/mizu.md"
        content, err = docsFS.ReadFile(path)
        if err != nil {
            return fmt.Errorf("documentation not found for %s", name)
        }
    }

    return renderMarkdown(os.Stdout, string(content))
}

// renderMarkdown renders markdown content with glamour
func renderMarkdown(w io.Writer, content string) error {
    // Auto-detect terminal width and style
    renderer, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(80),
    )
    if err != nil {
        // Fallback: print raw markdown
        _, err = fmt.Fprint(w, content)
        return err
    }

    rendered, err := renderer.Render(content)
    if err != nil {
        return err
    }

    _, err = fmt.Fprint(w, rendered)
    return err
}

// wrapRunE wraps a command's RunE to handle --md flag
func wrapRunE(original func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
    return func(cmd *cobra.Command, args []string) error {
        if globalFlags.MD {
            return printMarkdownHelp(cmd)
        }
        return original(cmd, args)
    }
}

// docsCmd provides a hidden command for browsing all documentation
var docsCmd = &cobra.Command{
    Use:    "docs [topic]",
    Short:  "Browse embedded documentation",
    Hidden: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) == 0 {
            return printMarkdownHelp(rootCmd)
        }
        // Find doc by topic
        topic := strings.ToLower(args[0])
        path := fmt.Sprintf("docs/commands/%s.md", topic)
        content, err := docsFS.ReadFile(path)
        if err != nil {
            return fmt.Errorf("documentation not found: %s", topic)
        }
        return renderMarkdown(os.Stdout, string(content))
    },
}
```

#### 1.3 Embed Documentation Structure

**File: `cmd/docs/embed.go`**
```go
package docs

import "embed"

// FS provides access to embedded documentation files
//
//go:embed *.md *.1 commands/*.md commands/*.1
var FS embed.FS
```

### Phase 2: Migrate Commands to Cobra

#### 2.1 New Command

**File: `cmd/cli/new.go`**
```go
package cli

import (
    "fmt"

    "github.com/spf13/cobra"
)

var newFlags struct {
    template string
    list     bool
    force    bool
    dryRun   bool
    name     string
    module   string
    license  string
    vars     []string
}

var newCmd = &cobra.Command{
    Use:   "new [path]",
    Short: "Create a new project from a template",
    Long: `Create a new project from a template.

Scaffolds a new Mizu project with the specified template into the target directory.
If no path is specified, the current directory is used.`,
    Example: `  # Create minimal project in current directory
  mizu new . --template minimal

  # Create API project in new directory
  mizu new ./myapp --template api

  # Preview what would be created
  mizu new ./myapp --template api --dry-run

  # List available templates
  mizu new --list`,
    Args: cobra.MaximumNArgs(1),
    RunE: wrapRunE(runNewCmd),
}

func init() {
    newCmd.Flags().StringVarP(&newFlags.template, "template", "t", "", "Template to render")
    newCmd.Flags().BoolVar(&newFlags.list, "list", false, "List available templates")
    newCmd.Flags().BoolVar(&newFlags.force, "force", false, "Overwrite existing files")
    newCmd.Flags().BoolVar(&newFlags.dryRun, "dry-run", false, "Print plan without writing")
    newCmd.Flags().StringVar(&newFlags.name, "name", "", "Project name")
    newCmd.Flags().StringVar(&newFlags.module, "module", "", "Go module path")
    newCmd.Flags().StringVar(&newFlags.license, "license", "MIT", "License identifier")
    newCmd.Flags().StringArrayVar(&newFlags.vars, "var", nil, "Template variable (k=v, repeatable)")
}

func runNewCmd(cmd *cobra.Command, args []string) error {
    out := newStyledOutput()

    if newFlags.list {
        return listTemplates(out)
    }

    if newFlags.template == "" {
        return fmt.Errorf("template is required (use --template or --list)")
    }

    targetPath := "."
    if len(args) > 0 {
        targetPath = args[0]
    }

    // ... rest of implementation (migrated from cmd_new.go)
    return nil
}
```

#### 2.2 Dev Command

**File: `cmd/cli/dev.go`**
```go
package cli

import (
    "github.com/spf13/cobra"
)

var devFlags struct {
    cmd string
}

var devCmd = &cobra.Command{
    Use:   "dev [flags] [-- args...]",
    Short: "Run the current project in development mode",
    Long: `Run the current project in development mode.

Automatically discovers the main package in cmd/* or the current directory,
builds it, and runs until interrupted.`,
    Example: `  # Auto-discover and run main package
  mizu dev

  # Specify explicit main package
  mizu dev --cmd ./cmd/server

  # Pass arguments to the application
  mizu dev -- --port 3000`,
    RunE: wrapRunE(runDevCmd),
}

func init() {
    devCmd.Flags().StringVar(&devFlags.cmd, "cmd", "", "Explicit main package path")
}

func runDevCmd(cmd *cobra.Command, args []string) error {
    // ... implementation migrated from cmd_dev.go
    return nil
}
```

#### 2.3 Contract Command Group

**File: `cmd/cli/contract.go`**
```go
package cli

import (
    "github.com/spf13/cobra"
)

var contractCmd = &cobra.Command{
    Use:   "contract",
    Short: "Work with service contracts",
    Long: `Work with service contracts.

Discover, inspect, and call methods on Mizu contract-based services.
Supports JSON-RPC, OpenAPI, and OpenRPC discovery.`,
    Example: `  # List all methods
  mizu contract ls

  # Call a method
  mizu contract call todo.Create '{"title":"Buy milk"}'

  # Export OpenAPI spec
  mizu contract spec > openapi.json

  # Use different server
  mizu contract ls http://api.example.com`,
}

var contractLsCmd = &cobra.Command{
    Use:     "ls [url]",
    Aliases: []string{"list"},
    Short:   "List services and methods",
    RunE:    wrapRunE(runContractLsCmd),
}

var contractShowCmd = &cobra.Command{
    Use:   "show <method> [url]",
    Short: "Show method details",
    Args:  cobra.MinimumNArgs(1),
    RunE:  wrapRunE(runContractShowCmd),
}

var contractCallCmd = &cobra.Command{
    Use:   "call <method> [input] [url]",
    Short: "Call a method",
    Args:  cobra.MinimumNArgs(1),
    RunE:  wrapRunE(runContractCallCmd),
}

var contractSpecCmd = &cobra.Command{
    Use:   "spec [url]",
    Short: "Export API specification",
    RunE:  wrapRunE(runContractSpecCmd),
}

var contractTypesCmd = &cobra.Command{
    Use:   "types [type] [url]",
    Short: "List types and schemas",
    RunE:  wrapRunE(runContractTypesCmd),
}

func init() {
    // Add subcommands
    contractCmd.AddCommand(contractLsCmd)
    contractCmd.AddCommand(contractShowCmd)
    contractCmd.AddCommand(contractCallCmd)
    contractCmd.AddCommand(contractSpecCmd)
    contractCmd.AddCommand(contractTypesCmd)

    // Flags for ls
    contractLsCmd.Flags().String("url", "", "Server URL")
    contractLsCmd.Flags().Bool("all", false, "Include deprecated methods")

    // Flags for show
    contractShowCmd.Flags().String("url", "", "Server URL")
    contractShowCmd.Flags().Bool("schema", false, "Show full JSON schema")

    // Flags for call
    contractCallCmd.Flags().String("url", "", "Server URL")
    contractCallCmd.Flags().Duration("timeout", 30*time.Second, "Request timeout")
    contractCallCmd.Flags().String("id", "", "Path parameter ID")
    contractCallCmd.Flags().Bool("raw", false, "Output raw response")
    contractCallCmd.Flags().StringArrayP("header", "H", nil, "Add header (key:value)")

    // Flags for spec
    contractSpecCmd.Flags().String("url", "", "Server URL")
    contractSpecCmd.Flags().String("format", "", "Output format (openapi, openrpc)")
    contractSpecCmd.Flags().Bool("pretty", false, "Pretty print JSON")
    contractSpecCmd.Flags().StringP("output", "o", "", "Output file")
    contractSpecCmd.Flags().String("service", "", "Export specific service")

    // Flags for types
    contractTypesCmd.Flags().String("url", "", "Server URL")
    contractTypesCmd.Flags().Bool("schema", false, "Show full JSON schema")
}

// RunE functions (implementations migrated from cmd_contract.go)
func runContractLsCmd(cmd *cobra.Command, args []string) error { return nil }
func runContractShowCmd(cmd *cobra.Command, args []string) error { return nil }
func runContractCallCmd(cmd *cobra.Command, args []string) error { return nil }
func runContractSpecCmd(cmd *cobra.Command, args []string) error { return nil }
func runContractTypesCmd(cmd *cobra.Command, args []string) error { return nil }
```

#### 2.4 Middleware Command Group

**File: `cmd/cli/middleware.go`**
```go
package cli

import (
    "github.com/spf13/cobra"
)

var middlewareCmd = &cobra.Command{
    Use:   "middleware",
    Short: "Explore available middlewares",
    Long: `Explore available middlewares for Mizu applications.

List all middlewares by category or show detailed information about specific ones.`,
    Example: `  # List all middlewares
  mizu middleware ls

  # Filter by category
  mizu middleware ls -c security

  # Show middleware details
  mizu middleware show helmet`,
}

var middlewareLsCmd = &cobra.Command{
    Use:     "ls",
    Aliases: []string{"list"},
    Short:   "List all middlewares",
    RunE:    wrapRunE(runMiddlewareLsCmd),
}

var middlewareShowCmd = &cobra.Command{
    Use:   "show <name>",
    Short: "Show details about a middleware",
    Args:  cobra.ExactArgs(1),
    RunE:  wrapRunE(runMiddlewareShowCmd),
}

func init() {
    middlewareCmd.AddCommand(middlewareLsCmd)
    middlewareCmd.AddCommand(middlewareShowCmd)

    middlewareLsCmd.Flags().StringP("category", "c", "", "Filter by category")
}

func runMiddlewareLsCmd(cmd *cobra.Command, args []string) error { return nil }
func runMiddlewareShowCmd(cmd *cobra.Command, args []string) error { return nil }
```

#### 2.5 Version Command

**File: `cmd/cli/version.go`**
```go
package cli

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    RunE:  wrapRunE(runVersionCmd),
}

func runVersionCmd(cmd *cobra.Command, args []string) error {
    out := newStyledOutput()

    if globalFlags.JSON {
        return out.writeJSON(map[string]string{
            "version":    Version,
            "go_version": runtime.Version(),
            "commit":     Commit,
            "built_at":   BuildTime,
        })
    }

    out.print("mizu version %s\n", Version)
    out.print("go version: %s\n", runtime.Version())
    out.print("commit: %s\n", Commit)
    out.print("built: %s\n", BuildTime)

    return nil
}
```

### Phase 3: Create Documentation Content

#### 3.1 Main Manual

**File: `cmd/docs/mizu.md`**
```markdown
# mizu(1) - Project Toolkit for go-mizu Framework

## SYNOPSIS

**mizu** [global options] *command* [command options] [arguments...]

## DESCRIPTION

**mizu** is the official CLI toolkit for the go-mizu web framework. It provides
commands for scaffolding new projects, running development servers, working with
service contracts, and exploring available middlewares.

## COMMANDS

### new
Create a new project from a template. Supports multiple project templates
including minimal, api, web, live, sync, and contract.

### dev
Run the current project in development mode. Auto-discovers the main package
and runs it with live output.

### contract
Work with service contracts. Discover methods, view schemas, and make calls
to contract-based services.

### middleware
Explore available middlewares. Browse the middleware catalog by category
and view usage examples.

### version
Print version information including build commit and Go version.

## GLOBAL OPTIONS

**--json**
: Emit machine-readable JSON output

**--no-color**
: Disable colored output

**-q, --quiet**
: Reduce output to errors only

**-v, --verbose**
: Increase verbosity (can be repeated)

**--md**
: Print help as rendered markdown

**-h, --help**
: Show help for any command

## EXAMPLES

Create a new API project:

    mizu new ./myapp --template api

Run development server:

    mizu dev

List contract methods:

    mizu contract ls

Show middleware details:

    mizu middleware show cors

## ENVIRONMENT

**MIZU_URL**
: Default server URL for contract commands (default: http://localhost:8080)

**NO_COLOR**
: Disable colors when set to any value

## FILES

**~/.config/mizu/config.yaml**
: User configuration file (reserved for future use)

## SEE ALSO

- mizu-new(1) - Create new projects
- mizu-dev(1) - Development server
- mizu-contract(1) - Contract operations
- mizu-middleware(1) - Middleware catalog

## AUTHORS

The go-mizu contributors <https://github.com/go-mizu/mizu>

## REPORTING BUGS

Report bugs at <https://github.com/go-mizu/mizu/issues>
```

#### 3.2 Command Documentation

**File: `cmd/docs/commands/new.md`**
```markdown
# mizu new - Create a new project

## SYNOPSIS

**mizu new** [path] [options]

## DESCRIPTION

Create a new Mizu project from a template. If no path is specified, the
current directory is used.

## OPTIONS

**-t, --template** *name*
: Template to use (required unless --list)

**--list**
: List available templates

**--force**
: Overwrite existing files

**--dry-run**
: Preview changes without writing files

**--name** *value*
: Project name (default: derived from path)

**--module** *value*
: Go module path (default: example.com/projectname)

**--license** *value*
: License identifier (default: MIT)

**--var** *key=value*
: Template variable (repeatable)

## TEMPLATES

| Template | Description |
|----------|-------------|
| minimal  | Bare-bones single-file app |
| api      | REST API with feature-based layout |
| web      | Server-rendered HTML application |
| live     | Real-time app with SSE |
| sync     | CRDT-based collaborative app |
| contract | Contract-first JSON-RPC service |

## EXAMPLES

Create minimal project:

    mizu new . --template minimal

Create API project in new directory:

    mizu new ./myapp --template api

Preview template output:

    mizu new ./myapp --template api --dry-run

List templates:

    mizu new --list

Custom module path:

    mizu new ./myapp --template api --module github.com/myorg/myapp

## SEE ALSO

mizu(1), mizu-dev(1)
```

**File: `cmd/docs/commands/dev.md`**
```markdown
# mizu dev - Run development server

## SYNOPSIS

**mizu dev** [options] [-- *args*...]

## DESCRIPTION

Run the current project in development mode. Automatically discovers the
main package in `cmd/*` or the current directory, builds it, and executes
until interrupted with Ctrl+C.

## OPTIONS

**--cmd** *path*
: Explicit main package path (overrides auto-discovery)

**--json**
: Emit lifecycle events as JSON

## DISCOVERY

The command searches for a runnable main package in this order:

1. Explicit `--cmd` path if provided
2. First directory in `cmd/` containing a main package
3. Current directory if it contains a main package

## SIGNALS

**SIGINT** (Ctrl+C), **SIGTERM**
: Graceful shutdown with 15-second timeout

## EXAMPLES

Auto-discover and run:

    mizu dev

Explicit main package:

    mizu dev --cmd ./cmd/server

Pass arguments to application:

    mizu dev -- --port 3000 --debug

JSON lifecycle events:

    mizu dev --json

## OUTPUT

Normal mode prints startup message and forwards stdout/stderr.

JSON mode emits events:
- `starting` - About to run the package
- `started` - Process started with PID
- `signal` - Signal received
- `stopped` - Process exited
- `error` - Error occurred

## SEE ALSO

mizu(1), mizu-new(1)
```

**File: `cmd/docs/commands/contract.md`**
```markdown
# mizu contract - Work with service contracts

## SYNOPSIS

**mizu contract** *subcommand* [options]

## DESCRIPTION

Discover, inspect, and call methods on Mizu contract-based services.
Supports multiple discovery protocols including JSON-RPC, OpenAPI, and OpenRPC.

## SUBCOMMANDS

### ls, list
List all services and methods from a running server.

    mizu contract ls [url] [--all]

### show
Show detailed information about a specific method.

    mizu contract show <method> [url] [--schema]

### call
Call a method with optional input data.

    mizu contract call <method> [input] [url] [options]

### spec
Export the API specification (OpenAPI or OpenRPC).

    mizu contract spec [url] [--format] [--pretty] [-o file]

### types
List types and their JSON schemas.

    mizu contract types [type] [url] [--schema]

## URL RESOLUTION

Server URL is resolved in this order:
1. Positional argument (if starts with http:// or https://)
2. `--url` flag
3. `MIZU_URL` environment variable
4. Default: http://localhost:8080

## EXAMPLES

List methods from local server:

    mizu contract ls

Call a method:

    mizu contract call todo.Create '{"title":"Buy milk"}'

Call with file input:

    mizu contract call todo.Create @input.json

Call with stdin:

    echo '{"title":"test"}' | mizu contract call todo.Create -

Export OpenAPI spec:

    mizu contract spec --pretty > openapi.json

Use different server:

    mizu contract ls http://api.example.com
    MIZU_URL=http://api.example.com mizu contract ls

## SEE ALSO

mizu(1), mizu-middleware(1)
```

**File: `cmd/docs/commands/middleware.md`**
```markdown
# mizu middleware - Explore available middlewares

## SYNOPSIS

**mizu middleware** *subcommand* [options]

## DESCRIPTION

Browse and explore the catalog of available middlewares for Mizu applications.
Middlewares are organized by category: security, logging, performance,
validation, and utilities.

## SUBCOMMANDS

### ls, list
List all middlewares, optionally filtered by category.

    mizu middleware ls [-c category]

### show
Show detailed information about a specific middleware.

    mizu middleware show <name>

## CATEGORIES

| Category    | Description |
|-------------|-------------|
| security    | Authentication, authorization, CORS, CSP |
| logging     | Request logging, metrics, tracing |
| performance | Compression, caching, rate limiting |
| validation  | Request validation, sanitization |
| utilities   | Recovery, timeout, request ID |

## EXAMPLES

List all middlewares:

    mizu middleware ls

Filter by category:

    mizu middleware ls -c security

Show middleware details:

    mizu middleware show helmet
    mizu middleware show cors

JSON output:

    mizu middleware ls --json
    mizu middleware show ratelimit --json

## SEE ALSO

mizu(1), mizu-new(1)
```

**File: `cmd/docs/commands/version.md`**
```markdown
# mizu version - Print version information

## SYNOPSIS

**mizu version** [options]

## DESCRIPTION

Print version information including the Mizu CLI version, Go version used
to build it, git commit hash, and build timestamp.

## OPTIONS

**--json**
: Output version information as JSON

## EXAMPLES

Print version:

    mizu version

JSON output:

    mizu version --json

## OUTPUT

Normal mode:
```
mizu version v0.3.0
go version: go1.24.11
commit: abc1234
built: 2025-01-15T10:30:00Z
```

JSON mode:
```json
{
  "version": "v0.3.0",
  "go_version": "go1.24.11",
  "commit": "abc1234",
  "built_at": "2025-01-15T10:30:00Z"
}
```

## SEE ALSO

mizu(1)
```

### Phase 4: Man Page Generation

#### 4.1 Mango Integration

**File: `cmd/cli/man.go`**
```go
package cli

import (
    "github.com/muesli/mango"
    mcobra "github.com/muesli/mango-cobra"
    "github.com/spf13/cobra"
)

// manCmd generates man pages (hidden, for build tooling)
var manCmd = &cobra.Command{
    Use:    "man",
    Short:  "Generate man pages",
    Hidden: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        manPage, err := mcobra.NewManPage(1, rootCmd)
        if err != nil {
            return err
        }

        manPage = manPage.WithSection("Authors", "The go-mizu contributors")
        manPage = manPage.WithSection("Bugs", "Report bugs at https://github.com/go-mizu/mizu/issues")

        fmt.Println(manPage.Build(mango.NewRenderer()))
        return nil
    },
}

func init() {
    rootCmd.AddCommand(manCmd)
}
```

### Phase 5: Styled Output

#### 5.1 Styled Output Helper

**File: `cmd/cli/output.go`**
```go
package cli

import (
    "encoding/json"
    "fmt"
    "io"
    "os"

    "github.com/charmbracelet/lipgloss"
)

// Styles for CLI output
var (
    titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
    errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
    successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
    dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
    cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

// styledOutput handles CLI output with modern styling
type styledOutput struct {
    stdout  io.Writer
    stderr  io.Writer
    noColor bool
}

func newStyledOutput() *styledOutput {
    noColor := globalFlags.NoColor || os.Getenv("NO_COLOR") != ""
    return &styledOutput{
        stdout:  os.Stdout,
        stderr:  os.Stderr,
        noColor: noColor,
    }
}

func (o *styledOutput) print(format string, args ...any) {
    if globalFlags.Quiet && !globalFlags.JSON {
        return
    }
    fmt.Fprintf(o.stdout, format, args...)
}

func (o *styledOutput) errorf(format string, args ...any) {
    if o.noColor {
        fmt.Fprintf(o.stderr, format, args...)
        return
    }
    fmt.Fprint(o.stderr, errorStyle.Render(fmt.Sprintf(format, args...)))
}

func (o *styledOutput) title(text string) string {
    if o.noColor {
        return text
    }
    return titleStyle.Render(text)
}

func (o *styledOutput) cyan(text string) string {
    if o.noColor {
        return text
    }
    return cyanStyle.Render(text)
}

func (o *styledOutput) dim(text string) string {
    if o.noColor {
        return text
    }
    return dimStyle.Render(text)
}

func (o *styledOutput) writeJSON(v any) error {
    enc := json.NewEncoder(o.stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
```

## Migration Checklist

### Pre-Migration
- [ ] Review all current flag definitions
- [ ] Identify any undocumented behaviors to preserve
- [ ] Create comprehensive test suite for existing commands

### Migration Steps
- [ ] Add dependencies to cmd/go.mod
- [ ] Create root.go with Fang/Cobra setup
- [ ] Create docs.go with markdown help system
- [ ] Create output.go with lipgloss styling
- [ ] Create cmd/docs/ directory structure
- [ ] Write mizu.md main manual
- [ ] Write commands/*.md documentation
- [ ] Migrate cmd_new.go to new.go
- [ ] Migrate cmd_dev.go to dev.go
- [ ] Migrate cmd_contract.go to contract.go
- [ ] Migrate cmd_middleware.go to middleware.go
- [ ] Migrate cmd_version.go to version.go
- [ ] Add man page generation
- [ ] Update main.go entry point
- [ ] Remove old cmd_*.go files
- [ ] Test all commands with --md flag
- [ ] Test shell completions
- [ ] Generate and verify man pages
- [ ] Update Makefile for doc generation

### Post-Migration
- [ ] Update README.md with new features
- [ ] Add docs on --md flag usage
- [ ] Verify backward compatibility
- [ ] Run full test suite
- [ ] Performance comparison (startup time)

## Testing Strategy

### Command Behavior Tests
```go
func TestNewCommand(t *testing.T) {
    // Test --list flag
    // Test --template flag
    // Test --dry-run flag
    // Test --md flag
}
```

### Markdown Help Tests
```go
func TestMarkdownHelp(t *testing.T) {
    // Verify docs exist for all commands
    // Test rendering without errors
    // Test fallback behavior
}
```

### Integration Tests
```bash
# Test shell completions
mizu completion bash > /tmp/mizu.bash
source /tmp/mizu.bash

# Test man page generation
mizu man > /tmp/mizu.1
man /tmp/mizu.1

# Test --md flag
mizu new --md
mizu dev --md
mizu contract --md
mizu middleware --md
```

## Benefits

1. **Modern DX**: Styled help, consistent formatting, shell completions
2. **Offline Docs**: Embedded markdown accessible anywhere
3. **Unix Philosophy**: Man pages for system integration
4. **Discoverability**: `--md` flag for quick, readable help
5. **Maintainability**: Markdown docs easier to update than code strings
6. **Professional Polish**: Charmbracelet styling out of the box

## Trade-offs

### Advantages
- Significantly improved user experience
- Consistent command structure via Cobra
- Automatic features (completions, versions, man pages)
- Styled error messages and help

### Disadvantages
- Additional dependencies (~5 new packages)
- Slightly larger binary size
- Migration effort for existing commands

### Mitigation
- Dependencies are well-maintained (Charmbracelet ecosystem)
- Binary size increase is minimal (~1-2MB)
- Migration can be done incrementally by command

## Dependencies Summary

| Package | Purpose | License |
|---------|---------|---------|
| charmbracelet/fang | CLI starter kit | MIT |
| spf13/cobra | Command framework | Apache 2.0 |
| charmbracelet/glamour | Markdown rendering | MIT |
| charmbracelet/lipgloss | Terminal styling | MIT |
| muesli/mango | Man page generation | MIT |
| muesli/mango-cobra | Cobra integration | MIT |

## Decision

**Status**: Draft - Awaiting approval

**Next Steps**:
1. Review and approve this spec
2. Create feature branch: `feat/cli-fang-docs`
3. Implement phases 1-5 in order
4. Test thoroughly
5. Document in release notes

---

**Author**: Claude Code
**Date**: 2025-12-18
**Version**: 1.0
