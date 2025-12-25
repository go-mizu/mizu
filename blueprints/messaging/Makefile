# Makefile for github.com/go-mizu/mizu/blueprints/messaging
.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/messaging
CMD     ?= ./cmd/messaging
DATA_DIR ?= $(HOME)/data/blueprint/messaging

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build messaging binary to $$HOME/bin/messaging
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: run
run: ## Run the server
	@$(GO) run $(CMD) serve --dev

.PHONY: init
init: ## Initialize the database
	@$(GO) run $(CMD) init

.PHONY: seed
seed: ## Seed the database with sample data
	@$(GO) run $(CMD) seed

.PHONY: test
test: ## Run all tests
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PKG)

.PHONY: update
update: ## Update dependencies
	@rm -f go.sum
	@$(GO) get -u ./...
	@$(GO) mod tidy

.PHONY: clean
clean: ## Remove binary and data directory
	@rm -f $(BINARY)
	@rm -rf $(DATA_DIR)
	@echo "Cleaned: $(BINARY), $(DATA_DIR)"

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run $(PKG)

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Messaging Blueprint - WhatsApp/Telegram-style messaging"
	@echo ""
	@echo "  Binary:   $(BINARY)"
	@echo "  Data:     $(DATA_DIR)"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
