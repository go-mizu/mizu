# Makefile for github.com/go-mizu/mizu
# Development, testing, and release targets.

.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

# Build configuration
GO        ?= go
PKG       ?= ./...
GOFLAGS   ?= -trimpath -mod=readonly
BINARY    ?= mizu
CMD_PATH  ?= ./cmd/mizu

# Version information (can be overridden)
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Linker flags for version injection
LDFLAGS   := -s -w \
	-X 'github.com/go-mizu/mizu/cli.Version=$(VERSION)' \
	-X 'github.com/go-mizu/mizu/cli.Commit=$(COMMIT)' \
	-X 'github.com/go-mizu/mizu/cli.BuildTime=$(BUILD_TIME)'

# Test configuration
COVERMODE ?= atomic
COVERFILE ?= coverage.out
COVERHTML ?= coverage.html
RUN       ?=
COUNT     ?= 1
GOTESTFLAGS ?=

# Release configuration
DIST_DIR  ?= dist

# =============================================================================
# Development
# =============================================================================

.PHONY: build
build: ## Build the binary for current platform
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PATH)
	@echo "Built: $(BINARY)"

.PHONY: install
install: ## Install the binary to GOPATH/bin
	@$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(CMD_PATH)
	@echo "Installed: $(shell go env GOPATH)/bin/$(BINARY)"

.PHONY: run
run: ## Run the CLI (use ARGS="..." for arguments)
	@$(GO) run $(GOFLAGS) -ldflags "$(LDFLAGS)" $(CMD_PATH) $(ARGS)

# =============================================================================
# Testing
# =============================================================================

.PHONY: test
test: ## Run tests with race + coverage
	@$(GO) test $(GOFLAGS) -v -race -shuffle=on \
		-count=$(COUNT) $(if $(RUN),-run $(RUN),) $(GOTESTFLAGS) \
		-covermode=$(COVERMODE) -coverprofile="$(COVERFILE)" \
		$(PKG)

.PHONY: cover
cover: test ## Generate HTML coverage report
	@$(GO) tool cover -html="$(COVERFILE)" -o "$(COVERHTML)"
	@echo "Coverage HTML: $(COVERHTML)"

.PHONY: tidy
tidy: ## go mod tidy + verify
	@$(GO) mod tidy
	@$(GO) mod verify

.PHONY: lint
lint: ## golangci-lint (fallback to go vet)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run $(PKG) --timeout=5m; \
	else \
		$(GO) vet $(PKG); \
	fi

# =============================================================================
# Release (local)
# =============================================================================

.PHONY: snapshot
snapshot: ## Build snapshot release (no publish)
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Error: goreleaser is not installed."; \
		echo "Install: go install github.com/goreleaser/goreleaser/v2@latest"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

.PHONY: release-dry-run
release-dry-run: ## Dry run of release (validates config)
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Error: goreleaser is not installed."; \
		echo "Install: go install github.com/goreleaser/goreleaser/v2@latest"; \
		exit 1; \
	fi
	goreleaser release --skip=publish --clean

.PHONY: release-check
release-check: ## Validate goreleaser config
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Error: goreleaser is not installed."; \
		echo "Install: go install github.com/goreleaser/goreleaser/v2@latest"; \
		exit 1; \
	fi
	goreleaser check

# =============================================================================
# Docker (local)
# =============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image locally
	docker build -t ghcr.io/go-mizu/mizu:$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -it ghcr.io/go-mizu/mizu:$(VERSION) $(ARGS)

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Remove build artifacts
	@rm -f "$(COVERFILE)" "$(COVERHTML)" "$(BINARY)"
	@rm -rf "$(DIST_DIR)"
	@echo "Cleaned build artifacts"

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## Show targets
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
