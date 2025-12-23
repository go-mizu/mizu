# Makefile for github.com/go-mizu/blueprints/finewiki
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BIN_DIR ?= $(HOME)/bin
BINARY  ?= $(BIN_DIR)/finewiki
CMD     ?= ./cmd/finewiki

DATA_DIR ?= $(HOME)/data/blueprint/finewiki

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build finewiki into $$HOME/bin
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: run
run: ## Run finewiki CLI (pass ARGS for subcommands)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) $(ARGS)

.PHONY: serve
serve: ## Run 'finewiki serve vi' (default language)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) serve vi

.PHONY: import
import: ## Run 'finewiki import vi' (default language)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) import vi

.PHONY: test
test: ## Run tests
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PKG)

.PHONY: test-race
test-race: ## Run tests with race detector
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -race $(PKG)

.PHONY: update
update: ## Refresh deps (rm go.sum; go get ./...; go mod tidy)
	@rm -f go.sum
	@$(GO) get -u ./...
	@$(GO) mod tidy

.PHONY: clean
clean: ## Remove binary
	@rm -f $(BINARY)

.PHONY: clean-data
clean-data: ## Remove all downloaded data
	@rm -rf "$(DATA_DIR)"
	@echo "Removed: $(DATA_DIR)"

.PHONY: help
help: ## Show available commands
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
