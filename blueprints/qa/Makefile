# Makefile for github.com/go-mizu/mizu/blueprints/qa
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BIN_DIR ?= $(HOME)/bin
BINARY  ?= $(BIN_DIR)/qa
CMD     ?= ./cmd/qa

DATA_DIR ?= $(HOME)/data/blueprint/qa

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build qa into $$HOME/bin
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: run
run: ## Run qa CLI (pass ARGS for subcommands)
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) $(ARGS)

.PHONY: serve
serve: ## Run 'qa serve'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) serve

.PHONY: dev
dev: ## Run 'qa serve --dev'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) serve --dev

.PHONY: init
init: ## Run 'qa init'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) init

.PHONY: seed
seed: ## Run 'qa seed'
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) run -trimpath -ldflags "$(GO_LDFLAGS)" $(CMD) seed

.PHONY: test
test: ## Run all tests
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v $(PKG)

.PHONY: test-race
test-race: ## Run tests with race detector
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -race $(PKG)

.PHONY: test-store
test-store: ## Run store tests only
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v ./store/duckdb/...

.PHONY: test-cli
test-cli: ## Run CLI tests only
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v ./cli/...

.PHONY: test-e2e
test-e2e: ## Run e2e tests (set E2E_TEST=1)
	@CGO_ENABLED=$(CGO_ENABLED) E2E_TEST=1 $(GO) test -v -tags=e2e ./app/web/...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -coverprofile=coverage.out $(PKG)
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: update
update: ## Refresh deps (rm go.sum; go get ./...; go mod tidy)
	@rm -f go.sum
	@$(GO) get -u ./...
	@$(GO) mod tidy

.PHONY: clean
clean: ## Remove binary
	@rm -f $(BINARY)
	@rm -f coverage.out coverage.html

.PHONY: clean-data
clean-data: ## Remove all data
	@rm -rf "$(DATA_DIR)"
	@echo "Removed: $(DATA_DIR)"

.PHONY: fmt
fmt: ## Format code
	@$(GO) fmt $(PKG)

.PHONY: vet
vet: ## Run go vet
	@$(GO) vet $(PKG)

.PHONY: lint
lint: fmt vet ## Run fmt and vet

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "QA Blueprint"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
