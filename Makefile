# Makefile for github.com/go-mizu/mizu
# Development, testing, and release targets.

.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

# --------------------------
# Build configuration
# --------------------------
GO        ?= go
PKG       ?= ./...
GOFLAGS   ?= -trimpath -mod=readonly
BINARY    ?= $(HOME)/bin/mizu
CMD_PATH  ?= ./cmd/mizu

# --------------------------
# Git metadata
# --------------------------
VERSION_DESCRIBE := $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
VERSION_TAG      := $(shell git describe --tags --exact-match --match "v*" 2>/dev/null)
VERSION          ?= $(if $(VERSION_TAG),$(VERSION_TAG),$(VERSION_DESCRIBE))

COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# --------------------------
# Linker flags
# --------------------------
LDFLAGS := -s -w \
	-X github.com/go-mizu/mizu/cli.Version=$(VERSION) \
	-X github.com/go-mizu/mizu/cli.Commit=$(COMMIT) \
	-X github.com/go-mizu/mizu/cli.BuildTime=$(BUILD_TIME)

# --------------------------
# Test configuration
# --------------------------
COVERMODE    ?= atomic
COVERFILE    ?= coverage.out
COVERHTML    ?= coverage.html
RUN          ?=
COUNT        ?= 1
GOTESTFLAGS  ?=

# Test selection knobs
# CHANGED=1  -> only test packages affected by git diff vs BASE
# EXCLUDE    -> space-separated substrings to exclude from package import paths
# BASE       -> git ref used as diff base when CHANGED=1
CHANGED      ?=
BASE         ?= origin/main

# Default exclusion: all middleware packages (heavy, rarely changed)
EXCLUDE      ?= middlewares

# --------------------------
# Development targets
# --------------------------
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

# --------------------------
# Testing targets
# --------------------------
.PHONY: test
test: ## Run tests (supports CHANGED=1 BASE=... EXCLUDE="...")
	@set -euo pipefail; \
	PKGS="$(PKG)"; \
	if [ -n "$(CHANGED)" ]; then \
		if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
			echo "Error: CHANGED=1 requires a git repository"; \
			exit 1; \
		fi; \
		git fetch --quiet --all --tags >/dev/null 2>&1 || true; \
		FILES="$$(git diff --name-only "$(BASE)...HEAD" -- '*.go' ':!vendor/**' ':!**/*_test.go' 2>/dev/null || true)"; \
		if [ -z "$$FILES" ]; then \
			echo "No changed .go files vs $(BASE); nothing to test."; \
			exit 0; \
		fi; \
		DIRS="$$(printf "%s\n" "$$FILES" | xargs -n1 dirname | sort -u)"; \
		PKGS="$$(for d in $$DIRS; do \
			$(GO) list $(GOFLAGS) "./$$d" 2>/dev/null || true; \
		done | sort -u)"; \
	fi; \
	if [ -n "$(EXCLUDE)" ] && [ -n "$$PKGS" ]

help: ## Show help
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""