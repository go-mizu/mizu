# Makefile for github.com/go-mizu/blueprints/finewiki
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BIN_DIR ?= $(HOME)/bin
BINARY  ?= $(BIN_DIR)/finewiki
CMD     ?= ./cmd/finewiki

DATA_DIR ?= $(HOME)/data/blueprints
DB_FILE  ?= $(DATA_DIR)/finewiki.duckdb

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS ?= -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)

.PHONY: build
build: ## Build the binary
	@mkdir -p $(dir $(BINARY))
	@$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: ## Install to $$HOME/bin
	@mkdir -p $(BIN_DIR)
	@$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Installed: $(BINARY)"

.PHONY: run
run: ## Run the server (use ARGS="...")
	@mkdir -p $(DATA_DIR)
	@FINEWIKI_DUCKDB="$(DB_FILE)" $(GO) run -trimpath -ldflags "$(LDFLAGS)" $(CMD) $(ARGS)

.PHONY: test
test: ## Run tests
	@$(GO) test $(PKG)

.PHONY: update
update: ## Reset deps: rm go.sum; go get ./...; go mod tidy
	@rm -f go.sum
	@$(GO) get ./...
	@$(GO) mod tidy

.PHONY: clean
clean: ## Remove local build artifacts
	@rm -f $(BINARY)
	@rm -f "$(DB_FILE)"

.PHONY: help
help: ## Show help
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
