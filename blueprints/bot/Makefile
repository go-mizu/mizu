.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/bot
CMD     ?= ./cmd/bot
DATA_DIR ?= $(HOME)/data/blueprint/bot

OPENBOT_BINARY ?= $(HOME)/bin/openbot
OPENBOT_CMD    ?= ./cmd/openbot

CGO_ENABLED ?= 0

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build bot binary to $$HOME/bin/bot
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install bot binary (alias for build)

.PHONY: run
run: ## Run the gateway server in development mode
	@GOWORK=off $(GO) run $(CMD) serve --dev

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run $(CMD) init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run $(CMD) seed

.PHONY: test
test: ## Run all Go tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -v $(PKG)

.PHONY: openbot
openbot: ## Build openbot binary to $$HOME/bin/openbot
	@mkdir -p $(dir $(OPENBOT_BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(OPENBOT_BINARY) $(OPENBOT_CMD)
	@echo "Built: $(OPENBOT_BINARY)"

.PHONY: run-openbot
run-openbot: ## Run openbot (standalone Telegram bot)
	@GOWORK=off $(GO) run $(OPENBOT_CMD)

.PHONY: clean
clean: ## Remove binary and data directory
	@rm -f $(BINARY)
	@rm -rf $(DATA_DIR)
	@echo "Cleaned: $(BINARY), $(DATA_DIR)"

.PHONY: update
update: ## Update dependencies
	@rm -f go.sum
	@$(GO) get -u ./...
	@$(GO) mod tidy

.PHONY: tidy
tidy: ## Run go mod tidy
	@GOWORK=off $(GO) mod tidy

.PHONY: db-reset
db-reset: ## Reset the SQLite database
	@echo "Resetting database..."
	@rm -f $(HOME)/data/blueprints/bot/bot.db
	@echo "Database reset complete"

.PHONY: quickstart
quickstart: ## Quick start: initialize, seed, and run
	@make init
	@make seed
	@make run

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Bot - Multi-Channel Chat Gateway"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
