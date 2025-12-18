# Spec 0048: Fix Goreleaser and Make Release

## Status
**Complete**

## Overview
Fix the goreleaser configuration, Dockerfile, and ensure the `make release` command works correctly with the existing GitHub Actions release workflow.

## Issues Found and Fixed

### 1. Dockerfile - GOWORK Conflict (Fixed)
**Problem**: `go mod download` failed with "conflicting replacements" error because `go.work` was copied into the Docker context and `GOWORK=off` was only set for the build command.

**Fix**: Added `ENV GOWORK=off` before the `go mod download` step.

### 2. .dockerignore - Truncated File (Fixed)
**Problem**: File was truncated (ended with "# Docker-relat") and missing `go.work` exclusion.

**Fix**: Completed the file and added `go.work` and `go.work.sum` exclusions.

### 3. Goreleaser Config - Already Working
The `.goreleaser.yaml` was correctly configured:
- Uses string format for before hooks: `'cd cmd && go mod tidy'`
- Has `GOWORK=off` in global env and build env
- `make release-check` passes
- `make release-snapshot` succeeds

## File Changes

### .dockerignore
Added:
```
# Docker-related
Dockerfile*

# Go workspace (causes conflicts during build)
go.work
go.work.sum

# Spec files
spec/
```

### Dockerfile
Added `ENV GOWORK=off` after build dependencies:
```dockerfile
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
ENV GOWORK=off  # <-- Added this line
```

## Makefile Release Targets
Already complete:
- `make release-check` - Validate goreleaser configuration
- `make release-snapshot` - Build a snapshot release locally
- `make release` - Build and publish a release (requires GITHUB_TOKEN)

## Verification Results

```bash
# Goreleaser config validation
$ make release-check
✓ 1 configuration file(s) validated

# Docker build
$ docker build -t mizu-test:latest .
✓ Successfully built

# Docker image test
$ docker run --rm mizu-test:latest version
mizu version dev
go version: go1.25.5
commit: unknown
built: unknown
```

## GitHub Actions Integration
The existing `.github/workflows/release.yml` is correctly configured:
- Triggers on `v*` tags
- Uses goreleaser-action v6 with `version: "~> v2"`
- Logs into GHCR for Docker image publishing
- Uses `GITHUB_TOKEN` and `HOMEBREW_TAP_TOKEN` secrets

No changes needed to the workflow file.

---

**Author**: Claude Code
**Date**: 2025-12-18
