# Makefile for github.com/go-mizu/mizu
# Development, testing, and release targets.

.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

# Build configuration
GO        ?= go
PKG       ?= ./...
GOFLAGS   ?= -trimpath -mod=readonly
BINARY    ?= $(HOME)/bin/mizu
CMD_PATH  ?= ./cmd/mizu

# Git metadata
VERSION_DESCRIBE := $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
VERSION_TAG      := $(shell git describe --tags --exact-match --match "v*" 2>/dev/null)
VERSION          ?= $(if $(VERSION_TAG),$(VERSION_TAG),$(VERSION_DESCRIBE))

COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Linker flags
LDFLAGS := -s -w \
	-X github.com/go-mizu/mizu/cli.Version=$(VERSION) \
	-X github.com/go-mizu/mizu/cli.Commit=$(COMMIT) \
	-X github.com/go-mizu/mizu/cli.BuildTime=$(BUILD_TIME)

# Test configuration
COVERMODE    ?= atomic
COVERFILE    ?= coverage.out
COVERHTML    ?= coverage.html
RUN          ?=
COUNT        ?= 1
GOTESTFLAGS  ?=

# Development targets
.PHONY: build
build: ## Build the binary for current platform
	@mkdir -p $(dir $(BINARY))
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PATH)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: ## Install the binary to $$HOME/bin
	@mkdir -p $(dir $(BINARY))
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PATH)
	@echo "Installed: $(BINARY)"

.PHONY: run
run: ## Run the CLI (use ARGS="...")
	@$(GO) run $(GOFLAGS) -ldflags "$(LDFLAGS)" $(CMD_PATH) $(ARGS)

# Testing targets
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

# Release targets (local)
.PHONY: snapshot
snapshot: ## Build snapshot release (no publish)
	@set -e; \
	git fetch --tags --quiet; \
	goreleaser release --snapshot --clean

.PHONY: release-dry-run
release-dry-run: ## Dry run of release
	@set -e; \
	if ! git diff --quiet || ! git diff --cached --quiet; then \
		echo "Error: working tree is dirty"; \
		exit 1; \
	fi; \
	git fetch --tags --quiet; \
	goreleaser release --skip=publish --clean

.PHONY: release-check
release-check: ## Validate goreleaser config
	@goreleaser check

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image locally
	@set -e; \
    	if echo "$(VERSION)" | grep -q dirty; then \
    		echo "Error: refusing to build Docker image from dirty tree"; \
    		exit 1; \
    	fi; \
    	docker build -t ghcr.io/go-mizu/mizu:$(VERSION) \
    		--build-arg VERSION=$(VERSION) \
    		--build-arg COMMIT=$(COMMIT) \
    		--build-arg BUILD_TIME=dev \
    		.

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -it ghcr.io/go-mizu/mizu:$(VERSION) $(ARGS)

# Cleanup
.PHONY: clean
clean: ## Remove build artifacts
	@rm -f "$(COVERFILE)" "$(COVERHTML)" "$(BINARY)"
	@rm -rf dist
	@echo "Cleaned build artifacts"

.PHONY: print-version
print-version:
	@echo "$(VERSION)"

# Help
.PHONY: help
help: ## Show targets
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
