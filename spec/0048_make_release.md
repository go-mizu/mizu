# Spec 0048: Fix Goreleaser and Make Release

## Status
**Complete**

## Overview
Fix the goreleaser configuration, Dockerfile, and ensure the `make release` command works correctly with the existing GitHub Actions release workflow.

## Current Issues

### 1. Goreleaser Config (`before.hooks` format)
The current configuration uses map format for hooks which is not supported:
```yaml
before:
  hooks:
    - cmd: go mod tidy
      dir: cmd
    - cmd: go mod verify
      dir: cmd
```

This produces the error:
```
yaml: unmarshal errors:
  line 11: cannot unmarshal !!map into string
  line 13: cannot unmarshal !!map into string
```

**Fix**: Use string format with shell commands:
```yaml
before:
  hooks:
    - 'cd cmd && go mod tidy'
    - 'cd cmd && go mod verify'
```

### 1b. Goreleaser Go Workspace Conflict
When `go.work` exists, goreleaser's module info loading step fails with conflicting replacements.

**Fix**: Add global `GOWORK=off` environment variable:
```yaml
env:
  - GOWORK=off
```

### 2. Dockerfile Build Path Issues
The Dockerfile has incorrect paths for the CLI module:

1. **Ldflags path**: Uses `github.com/go-mizu/mizu/cli` but should be `github.com/go-mizu/mizu/cmd/cli`
2. **Build command**: Doesn't account for the separate `cmd/` module with its own `go.mod`
3. **Module download**: Copies root `go.mod` but CLI has separate `cmd/go.mod`

**Current (broken)**:
```dockerfile
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
  -ldflags="-s -w \
    -X github.com/go-mizu/mizu/cli.Version=${VERSION} \
    ..." \
  -o /out/mizu ./cmd/mizu
```

**Fix**: Handle the cmd/ module properly:
```dockerfile
WORKDIR /src
COPY . .
WORKDIR /src/cmd
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOWORK=off go build -trimpath \
  -ldflags="-s -w \
    -X github.com/go-mizu/mizu/cmd/cli.Version=${VERSION} \
    ..." \
  -o /out/mizu ./mizu
```

### 3. Makefile Release Targets
The Makefile already has `release`, `release-check`, and `release-snapshot` targets. These are correct but depend on the goreleaser config being fixed.

## Implementation Plan

### Step 1: Fix Goreleaser Hooks
Change the before.hooks section from map format to string format.

### Step 2: Fix Dockerfile
Update the Dockerfile to:
- Use correct ldflags paths (`cmd/cli` instead of `cli`)
- Handle the cmd/ module structure with GOWORK=off
- Copy files and build from the correct directory

### Step 3: Validate Configuration
Run `goreleaser check` to ensure config is valid.

### Step 4: Test Snapshot Release
Run `make release-snapshot` to test the full build process locally (without publishing).

## File Changes

### .goreleaser.yaml
```yaml
# Global environment - disable go workspace
env:
  - GOWORK=off

# Before hooks - use string format with cd
before:
  hooks:
    - 'cd cmd && go mod tidy'
    - 'cd cmd && go mod verify'
```

### Dockerfile
- Fix ldflags path: `github.com/go-mizu/mizu/cli` -> `github.com/go-mizu/mizu/cmd/cli`
- Change build strategy to work with cmd/ module
- Add `GOWORK=off` to build command

## Verification

1. `make release-check` - validates goreleaser config
2. `make release-snapshot` - builds snapshot locally
3. `docker build .` - verifies Dockerfile works

## GitHub Actions Integration

The existing `.github/workflows/release.yml` is already set up correctly:
- Triggers on `v*` tags
- Uses goreleaser-action v6 with `version: "~> v2"`
- Logs into GHCR for Docker image publishing
- Uses `GITHUB_TOKEN` and `HOMEBREW_TAP_TOKEN` secrets

No changes needed to the workflow file.

---

**Author**: Claude Code
**Date**: 2025-12-18
