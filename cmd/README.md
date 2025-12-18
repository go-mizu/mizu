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

## Module Separation

The CLI is a separate module to:
- Keep the core framework clean (zero dependencies)
- Allow CLI-specific dependencies without polluting the framework
- Enable independent versioning of tools vs library
- Reduce framework installation size for end-users
