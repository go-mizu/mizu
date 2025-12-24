# Makefile for github.com/go-mizu/blueprints/microblog
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BIN_DIR ?= $(HOME)/bin
BINARY  ?= $(BIN_DIR)/microblog
CMD     ?= ./cmd/microblog

DATA_DIR ?= $(HOME)/data/blueprint/microblog

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build microblog into $$HOME/bin
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Build and install to $$HOME/bin

.PHONY: run
run: ## Run microblog CLI (pass ARGS for subcommands)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) $(ARGS)

.PHONY: serve
serve: ## Run 'microblog serve'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) serve

.PHONY: dev
dev: ## Run 'microblog serve --dev'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) serve --dev

.PHONY: init
init: ## Run 'microblog init'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) init

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
clean-data: ## Remove all data
	@rm -rf "$(DATA_DIR)"
	@echo "Removed: $(DATA_DIR)"

.PHONY: help
help: ## Show available commands
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
