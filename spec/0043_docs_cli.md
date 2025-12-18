# 0043: CLI Documentation Rewrite

## Summary

Complete rewrite of all CLI documentation pages, reorganized with each template as a group, written for absolute beginners with detailed explanations and practical examples.

## Background

The current CLI documentation is sparse and not beginner-friendly. This spec details a comprehensive rewrite that:

1. Explains every concept in detail for beginners
2. Provides practical, step-by-step tutorials
3. Organizes templates into their own groups
4. Documents the new `contract` command and all subcommands
5. Updates for 6 templates: minimal, api, contract, web, live, sync

## CLI Commands

### Available Commands

| Command | Description |
|---------|-------------|
| `mizu new` | Create a new project from a template |
| `mizu dev` | Run the current project in development mode |
| `mizu contract` | Work with service contracts (5 subcommands) |
| `mizu version` | Print version information |

### Contract Subcommands

| Subcommand | Description |
|------------|-------------|
| `mizu contract ls` | List services and methods |
| `mizu contract show` | Show method details |
| `mizu contract call` | Call a method |
| `mizu contract spec` | Export API specification |
| `mizu contract types` | List types and schemas |

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Emit machine-readable output |
| `--no-color` | Disable color output |
| `-q, --quiet` | Reduce output (errors only) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-h, --help` | Show help |

### Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | OK | Command completed successfully |
| 1 | Error | General error occurred |
| 2 | Usage | Invalid usage or arguments |
| 3 | NoProject | No runnable project found |

## Templates

### Available Templates (6)

| Template | Description | Use Case |
|----------|-------------|----------|
| `minimal` | Smallest runnable Mizu project | Learning, quick experiments |
| `api` | JSON API service with recommended layout | REST APIs, backend services |
| `contract` | Transport-neutral service contracts | Multi-protocol APIs (REST + JSON-RPC) |
| `web` | Full-stack web app with views | Server-rendered websites |
| `live` | Real-time interactive app | Interactive dashboards, chat |
| `sync` | Offline-first app with sync | Collaborative apps, offline support |

## New Documentation Structure

### docs/docs.json Navigation

```json
{
  "tab": "CLI",
  "groups": [
    {
      "group": "Getting Started",
      "pages": [
        "cli/overview",
        "cli/installation"
      ]
    },
    {
      "group": "Commands",
      "pages": [
        "cli/new",
        "cli/dev",
        "cli/contract",
        "cli/version"
      ]
    },
    {
      "group": "Templates Overview",
      "pages": [
        "cli/templates"
      ]
    },
    {
      "group": "Minimal Template",
      "pages": [
        "cli/minimal/overview",
        "cli/minimal/structure",
        "cli/minimal/tutorial"
      ]
    },
    {
      "group": "API Template",
      "pages": [
        "cli/api/overview",
        "cli/api/structure",
        "cli/api/tutorial"
      ]
    },
    {
      "group": "Contract Template",
      "pages": [
        "cli/contract-template/overview",
        "cli/contract-template/structure",
        "cli/contract-template/tutorial"
      ]
    },
    {
      "group": "Web Template",
      "pages": [
        "cli/web/overview",
        "cli/web/structure",
        "cli/web/tutorial"
      ]
    },
    {
      "group": "Live Template",
      "pages": [
        "cli/live/overview",
        "cli/live/structure",
        "cli/live/tutorial"
      ]
    },
    {
      "group": "Sync Template",
      "pages": [
        "cli/sync/overview",
        "cli/sync/structure",
        "cli/sync/tutorial"
      ]
    },
    {
      "group": "Reference",
      "pages": [
        "cli/global-flags",
        "cli/exit-codes",
        "cli/json-output"
      ]
    }
  ]
}
```

## Page Specifications

### Getting Started

#### cli/overview.mdx

**Purpose:** Introduce the CLI to absolute beginners

**Content:**
- What is the Mizu CLI?
- What problems does it solve?
- Quick start example (full walkthrough)
- Command overview table
- When to use which template (decision tree)
- Design philosophy explained simply

**Beginner Focus:**
- Explain what a CLI is
- Explain what scaffolding means
- Show exact terminal commands with expected output

#### cli/installation.mdx

**Purpose:** Get the CLI installed on any system

**Content:**
- Prerequisites (Go version, what is Go?)
- Installation via `go install`
- Installation from source
- Verify installation works
- Troubleshooting common issues
- Shell completion setup
- Updating to new versions

**Beginner Focus:**
- Explain PATH and GOPATH
- Show how to check Go version
- Platform-specific instructions (macOS, Linux, Windows)

### Commands

#### cli/new.mdx

**Purpose:** Complete reference for project creation

**Content:**
- What does `mizu new` do?
- Basic usage with examples
- All flags explained:
  - `-t, --template` (required)
  - `--list`
  - `--force`
  - `--dry-run`
  - `--name`
  - `--module`
  - `--license`
  - `--var`
- Template variables explained
- Practical examples for each scenario
- JSON output format
- Common mistakes and solutions

**Beginner Focus:**
- What is a Go module?
- What is a template variable?
- Step-by-step walkthrough

#### cli/dev.mdx

**Purpose:** Complete reference for development server

**Content:**
- What does `mizu dev` do?
- How it auto-detects your main package
- Signal handling (Ctrl+C)
- All flags explained:
  - `--cmd`
  - `--json`
- JSON lifecycle events
- When to use `--cmd`
- Graceful shutdown explained

**Beginner Focus:**
- What is a main package?
- What is graceful shutdown?
- How to pass arguments to your app

#### cli/contract.mdx

**Purpose:** Complete reference for contract commands

**Content:**
- What are service contracts?
- Why use the contract command?
- Subcommand overview
- Each subcommand detailed:
  - `ls` - List methods
  - `show` - Method details
  - `call` - Invoke methods
  - `spec` - Export OpenAPI/OpenRPC
  - `types` - View schemas
- Server URL configuration
- Authentication headers
- Practical workflows

**Beginner Focus:**
- What is JSON-RPC?
- What is OpenAPI?
- Interactive examples

#### cli/version.mdx

**Purpose:** Reference for version command

**Content:**
- What information is shown
- JSON output format
- Using version in scripts
- Build-time variables

### Templates Overview

#### cli/templates.mdx

**Purpose:** Help users choose the right template

**Content:**
- Template comparison table
- Decision flowchart
- When to use each template
- Common patterns
- Upgrading between templates

### Template Groups

Each template group has 3 pages:

#### {template}/overview.mdx

- What is this template for?
- Features included
- When to choose this template
- Prerequisites
- Quick create command

#### {template}/structure.mdx

- Directory layout diagram
- Every file explained
- Key concepts for this template
- Configuration options
- Customization points

#### {template}/tutorial.mdx

- Step-by-step guide from scratch
- Create the project
- Explore the code
- Make modifications
- Run and test
- Next steps

### Reference

#### cli/global-flags.mdx

- Each flag in detail
- Flag placement rules
- Environment variables
- Flag combinations

#### cli/exit-codes.mdx

- Each exit code meaning
- Using in shell scripts
- Error handling patterns
- CI/CD integration

#### cli/json-output.mdx

- JSON schema for each command
- Parsing examples (jq)
- Integration examples
- Streaming JSON (dev command)

## Files to Create

### New Files

```
docs/cli/
├── overview.mdx (rewrite)
├── installation.mdx (rewrite)
├── new.mdx (rewrite)
├── dev.mdx (rewrite)
├── contract.mdx (NEW)
├── version.mdx (rewrite)
├── templates.mdx (rewrite)
├── minimal/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── api/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── contract-template/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── web/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── live/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── sync/
│   ├── overview.mdx (NEW)
│   ├── structure.mdx (NEW)
│   └── tutorial.mdx (NEW)
├── global-flags.mdx (rewrite)
├── exit-codes.mdx (rewrite)
└── json-output.mdx (rewrite)
```

### Files to Remove

- `docs/cli/template-minimal.mdx` (replaced by minimal/*)
- `docs/cli/template-api.mdx` (replaced by api/*)
- `docs/cli/template-contract.mdx` (replaced by contract-template/*)

## Writing Guidelines

### For Absolute Beginners

1. **Assume nothing** - Explain every term on first use
2. **Show, don't tell** - Include actual terminal output
3. **One step at a time** - Break complex tasks into small steps
4. **Provide context** - Explain WHY, not just HOW
5. **Include mistakes** - Show common errors and how to fix them

### Code Examples

- Always show the command AND expected output
- Use realistic project names
- Include comments in code
- Test all examples before documenting

### Formatting

- Use callouts for important notes
- Use tables for reference information
- Use code blocks with language hints
- Keep paragraphs short (3-4 sentences max)

## Implementation Order

1. Write spec (this file)
2. Update docs/docs.json
3. Write overview.mdx (entry point)
4. Write installation.mdx
5. Write new.mdx
6. Write dev.mdx
7. Write contract.mdx
8. Write version.mdx
9. Write templates.mdx
10. Write minimal/* pages
11. Write api/* pages
12. Write contract-template/* pages
13. Write web/* pages
14. Write live/* pages
15. Write sync/* pages
16. Write reference pages (global-flags, exit-codes, json-output)
17. Remove old template pages

## Testing

After writing all pages:

1. Build docs locally
2. Verify all links work
3. Check navigation order
4. Test all code examples
5. Review for beginner clarity
