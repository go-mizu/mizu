# Spec 0045: Go Workspace for CLI Module Separation

## Status
**Draft**

## Overview
Separate the CLI (`cmd/mizu`) into its own Go module with independent dependencies while keeping the root `go.mod` clean and dependency-free. This enables the CLI to use external dependencies without polluting the core framework module.

## Motivation

### Current State
- Single `go.mod` at repository root
- CLI code in `cmd/cli/` package
- CLI binary entry point at `cmd/mizu/main.go`
- No external dependencies currently (CLI uses only stdlib)
- Users install via: `go install github.com/go-mizu/mizu/cmd/mizu@latest`

### Problems
1. **Dependency Pollution**: Adding CLI dependencies (e.g., CLI frameworks, formatters, build tools) would pollute the core framework's dependency tree
2. **User Impact**: Framework users who `go get github.com/go-mizu/mizu` would download unnecessary CLI dependencies
3. **Maintenance**: CLI and framework evolve at different rates with different requirements
4. **Module Clarity**: CLI is a tool, framework is a library - they should be separate modules

### Goals
1. Keep root `go.mod` clean with zero dependencies (pure stdlib framework)
2. Allow `cmd/go.mod` to have its own dependencies
3. Maintain `go install github.com/go-mizu/mizu/cmd/mizu@latest` compatibility
4. Enable local development with `go.work`
5. Preserve existing Makefile workflows

## Architecture

### Repository Structure
```
mizu/
├── go.mod                    # Root module (clean, no dependencies)
├── go.work                   # Workspace file (git-ignored, local dev only)
├── app.go, router.go, ...    # Framework code
├── cmd/
│   ├── go.mod                # CLI module (can have dependencies)
│   ├── go.sum                # CLI dependencies checksums
│   ├── cli/                  # CLI implementation
│   │   ├── cmd_*.go
│   │   └── ...
│   └── mizu/
│       └── main.go           # CLI entry point
└── spec/
    └── 0045_go_work_cli.md   # This document
```

### Module Relationships

**Root Module**: `github.com/go-mizu/mizu`
- Pure framework code
- No dependencies (only stdlib)
- Used by end-users building web apps

**CLI Module**: `github.com/go-mizu/mizu/cmd`
- CLI implementation in `cli/` package
- Binary entry point in `mizu/main.go`
- Depends on root module: `github.com/go-mizu/mizu`
- Can add external dependencies (e.g., for enhanced CLI features)

## Implementation Plan

### Phase 1: Create CLI Module

#### 1.1 Create `cmd/go.mod`
```bash
cd cmd
go mod init github.com/go-mizu/mizu/cmd
go mod edit -go=1.24.11
```

**Content**:
```go
module github.com/go-mizu/mizu/cmd

go 1.24.11

// Local development: replaced by go.work
// Remote install: uses latest published version
require github.com/go-mizu/mizu v0.0.0

// Workspace replace directive (only active when go.work is present)
// When installing via `go install`, this is ignored and Go fetches
// the actual published version from the remote repository
replace github.com/go-mizu/mizu => ../
```

**Key Points**:
- `require github.com/go-mizu/mizu v0.0.0` is a placeholder version
- `replace` directive points to parent directory for local development
- When `go install pkg@version` runs, it ignores `replace` directives in downloaded code
- The `replace` only works for local development (with or without go.work)

#### 1.2 Update Import Paths
No changes needed! Current imports in `cmd/cli/*.go` already use full paths:
- `github.com/go-mizu/mizu/cli` → stays the same (will be `../cli` relative to cmd module)

Actually, wait - the CLI imports look relative. Let me verify the import structure.

After checking `cmd/mizu/main.go`:
```go
import "github.com/go-mizu/mizu/cli"
```

This needs to become:
```go
import "github.com/go-mizu/mizu/cmd/cli"
```

Because the `cli` package will now be part of the `cmd` module.

**Action**: Update import in `cmd/mizu/main.go`:
```go
package main

import (
	"os"

	"github.com/go-mizu/mizu/cmd/cli"
)

func main() {
	os.Exit(cli.Run())
}
```

**Also Update**: The linker flags in `Makefile` that reference `github.com/go-mizu/mizu/cli`:
```makefile
# OLD
LDFLAGS := -s -w \
	-X github.com/go-mizu/mizu/cli.Version=$(VERSION) \
	-X github.com/go-mizu/mizu/cli.Commit=$(COMMIT) \
	-X github.com/go-mizu/mizu/cli.BuildTime=$(BUILD_TIME)

# NEW
LDFLAGS := -s -w \
	-X github.com/go-mizu/mizu/cmd/cli.Version=$(VERSION) \
	-X github.com/go-mizu/mizu/cmd/cli.Commit=$(COMMIT) \
	-X github.com/go-mizu/mizu/cmd/cli.BuildTime=$(BUILD_TIME)
```

### Phase 2: Create Workspace Configuration

#### 2.1 Create `go.work`
```bash
# In repository root
go work init
go work use .
go work use ./cmd
```

**Content**:
```
go 1.24.11

use (
	.
	./cmd
)
```

**Purpose**:
- Links both modules for local development
- Allows cmd module to use local (unreleased) framework changes
- Only needed for development, not for end-users

#### 2.2 Add `go.work` to `.gitignore`
```bash
echo "go.work" >> .gitignore
echo "go.work.sum" >> .gitignore
```

**Rationale**:
- Workspace files are developer-specific
- Each contributor may configure differently
- End-users don't need workspaces
- Remote `go install` doesn't use workspaces

### Phase 3: Update Build Configuration

#### 3.1 Update `Makefile`
```makefile
# Existing variables stay the same
CMD_PATH  ?= ./cmd/mizu

# Update LDFLAGS (see Phase 1.2 above)
LDFLAGS := -s -w \
	-X github.com/go-mizu/mizu/cmd/cli.Version=$(VERSION) \
	-X github.com/go-mizu/mizu/cmd/cli.Commit=$(COMMIT) \
	-X github.com/go-mizu/mizu/cmd/cli.BuildTime=$(BUILD_TIME)

# Existing targets work as-is:
# - make install: builds from ./cmd/mizu
# - make run: runs from ./cmd/mizu
# - make test: tests all packages

# Add workspace target for developer convenience
.PHONY: workspace
workspace: ## Initialize go.work for local development
	@if [ ! -f go.work ]; then \
		go work init; \
		go work use .; \
		go work use ./cmd; \
		echo "Created go.work"; \
	else \
		echo "go.work already exists"; \
	fi
```

#### 3.2 Update Documentation

**Update `README.md`** section on CLI installation:

```markdown
### CLI Installation

The Mizu CLI provides project scaffolding and development tools.

**Using Go install:**

```bash
go install github.com/go-mizu/mizu/cmd/mizu@latest
```

This downloads and installs the CLI from the published module.

**From source (for development):**

```bash
git clone https://github.com/go-mizu/mizu.git
cd mizu
make workspace  # Creates go.work for local development
make install
```

This builds the CLI using your local changes and installs to `$HOME/bin`.
```

**Create `cmd/README.md`** to document the CLI module:

```markdown
# Mizu CLI

This directory contains the Mizu CLI tool as a separate Go module.

## Module Structure

- **Module**: `github.com/go-mizu/mizu/cmd`
- **Binary**: `cmd/mizu/main.go`
- **Package**: `cli/` (CLI implementation)
- **Dependencies**: Managed independently from core framework

## Installation

```bash
go install github.com/go-mizu/mizu/cmd/mizu@latest
```

## Development

When working on the CLI from the repository:

```bash
cd mizu/        # Repository root
make workspace  # Create go.work
make install    # Build and install to $HOME/bin
```

The workspace setup allows the CLI to use the local framework code.

## Adding Dependencies

The CLI module can have its own dependencies:

```bash
cd cmd/
go get github.com/some/package@latest
```

This won't affect the core framework's dependency tree.
```

### Phase 4: Version Management Strategy

#### 4.1 Release Process

**Tagging Strategy**:
- Framework tags: `v0.x.y` (e.g., `v0.3.0`)
- CLI tags: `cmd/v0.x.y` (e.g., `cmd/v0.3.0`)

**Release Workflow**:
1. Make framework changes in root
2. Make CLI changes in `cmd/`
3. Tag framework: `git tag v0.3.0`
4. Tag CLI: `git tag cmd/v0.3.0`
5. Push tags: `git push --tags`

**Synchronized Releases**:
For most releases, both modules are tagged together:
```bash
VERSION=0.3.0
git tag v${VERSION}
git tag cmd/v${VERSION}
git push --tags
```

**Independent Releases** (when only CLI changes):
```bash
git tag cmd/v0.3.1
git push --tags
```

#### 4.2 Dependency Version Updates

**cmd/go.mod Maintenance**:

After releasing framework `v0.3.0`, update CLI's requirement:
```bash
cd cmd/
go get github.com/go-mizu/mizu@v0.3.0
go mod tidy
git add go.mod go.sum
git commit -m "cmd: update framework dependency to v0.3.0"
```

**Automation Option**:
Add to release script or Makefile:
```makefile
.PHONY: release-prep
release-prep: ## Prepare for synchronized release
	@VERSION=$$(git describe --tags --abbrev=0 | sed 's/^v//'); \
	cd cmd && go get github.com/go-mizu/mizu@v$$VERSION && go mod tidy
```

### Phase 5: CI/CD Updates

#### 5.1 GitHub Actions Workflow

Update `.github/workflows/test.yml` (or similar):

```yaml
name: Test

on: [push, pull_request]

jobs:
  test-framework:
    name: Test Framework
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      # Test root module
      - name: Test framework
        run: go test -v ./...
        working-directory: .

  test-cli:
    name: Test CLI
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      # Test CLI module
      - name: Test CLI
        run: go test -v ./...
        working-directory: ./cmd

  build-cli:
    name: Build CLI
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      # Build CLI binary
      - name: Build
        run: go build -v ./cmd/mizu
```

#### 5.2 Makefile Test Updates

Ensure tests run for both modules:

```makefile
.PHONY: test
test: test-framework test-cli ## Run all tests

.PHONY: test-framework
test-framework: ## Test framework only
	@# Existing test logic for root module
	@$(GO) test $(GOTESTFLAGS) $(PKG)

.PHONY: test-cli
test-cli: ## Test CLI only
	@cd cmd && $(GO) test -v ./...
```

## Migration Checklist

### Pre-Migration
- [ ] Review current CLI dependencies (currently none)
- [ ] Identify future dependencies to be added
- [ ] Communicate changes to contributors
- [ ] Document workspace setup in contributing guide

### Migration Steps
- [ ] Create `cmd/go.mod` with framework dependency
- [ ] Update import path in `cmd/mizu/main.go`
- [ ] Update linker flags in `Makefile`
- [ ] Create `go.work` template (add to `.gitignore`)
- [ ] Update `Makefile` with workspace target
- [ ] Update `README.md` installation instructions
- [ ] Create `cmd/README.md` documentation
- [ ] Update CI/CD workflows
- [ ] Test local build: `make install`
- [ ] Test workspace setup: `make workspace`
- [ ] Test remote install: `go install github.com/go-mizu/mizu/cmd/mizu@latest`

### Post-Migration
- [ ] Tag first release with new structure
- [ ] Update framework tag: `v0.3.0` (or appropriate version)
- [ ] Update CLI tag: `cmd/v0.3.0`
- [ ] Verify users can install via `go install`
- [ ] Verify developers can use `make workspace`
- [ ] Document release process in `CONTRIBUTING.md`

## Testing Strategy

### Test Cases

1. **Local Development with Workspace**
   ```bash
   git clone https://github.com/go-mizu/mizu.git
   cd mizu
   make workspace
   make install
   ~/bin/mizu version  # Should work
   ```

2. **Local Development without Workspace**
   ```bash
   git clone https://github.com/go-mizu/mizu.git
   cd mizu
   make install  # Should still work (uses replace directive)
   ~/bin/mizu version
   ```

3. **Remote Install (Standard User Flow)**
   ```bash
   go install github.com/go-mizu/mizu/cmd/mizu@latest
   mizu version  # Should work
   ```

4. **Framework Usage (Unaffected)**
   ```bash
   mkdir /tmp/myapp
   cd /tmp/myapp
   go mod init myapp
   go get github.com/go-mizu/mizu@latest
   # Should have zero transitive dependencies
   ```

5. **CLI with Future Dependencies**
   ```bash
   cd mizu/cmd
   go get github.com/fatih/color@latest  # Example dependency
   go mod tidy
   cd ..
   make install  # CLI builds with new dependency

   # Verify framework stays clean
   go list -m all  # Should show only framework module
   ```

## Benefits

1. **Dependency Isolation**: CLI tools can use rich dependencies without affecting framework users
2. **Module Clarity**: Clear separation between tool and library
3. **Faster Framework Adoption**: Users don't download unnecessary CLI dependencies
4. **Independent Evolution**: CLI and framework can be versioned and released independently
5. **Smaller Framework**: Keeps core framework minimal and focused

## Trade-offs

### Advantages
- Clean framework module (zero dependencies)
- CLI can use modern tooling without guilt
- Better module semantics
- Independent versioning

### Disadvantages
- More complex release process (two tags)
- Need to maintain version sync between modules
- Developers need to understand workspace setup
- Slightly more complex repository structure

### Mitigation
- Automate release tagging (Makefile targets)
- Document workspace setup clearly
- Provide `make workspace` convenience target
- Keep synchronized releases as default

## Future Enhancements

1. **CLI Dependencies**: Once migrated, add useful CLI dependencies:
   - `github.com/fatih/color` - Better terminal colors
   - `github.com/mattn/go-isatty` - TTY detection
   - Code generation libraries for templates

2. **Additional Modules**: Could extend pattern to other tools:
   - `github.com/go-mizu/mizu/tools` - Development tools
   - `github.com/go-mizu/mizu/gen` - Code generation

3. **Release Automation**: GitHub Actions workflow to:
   - Auto-tag synchronized releases
   - Update cmd/go.mod framework version
   - Validate module graph

## References

- [Go Workspaces](https://go.dev/doc/tutorial/workspaces)
- [Multi-module repositories](https://github.com/golang/go/wiki/Modules#multi-module-repositories)
- [Module version numbering](https://go.dev/doc/modules/version-numbers)
- [Major subdirectories](https://go.dev/wiki/Modules#is-it-possible-to-add-a-module-to-a-multi-module-repository)

## Decision

**Status**: Awaiting approval

**Next Steps**:
1. Review and approve this spec
2. Create feature branch: `feat/go-workspace-cli`
3. Implement Phase 1-5 in order
4. Test thoroughly before merge
5. Document in release notes

---

**Author**: Claude Code
**Date**: 2025-12-18
**Version**: 1.0
