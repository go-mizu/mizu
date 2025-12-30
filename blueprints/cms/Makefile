.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/cms
CMD     ?= ./cmd/cms
DATA_DIR ?= $(HOME)/data/blueprint/cms

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build cms binary to $$HOME/bin/cms
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install cms binary to $$HOME/bin/cms (alias for build)

.PHONY: run
run: ## Run the server
	@GOWORK=off $(GO) run $(CMD) serve --dev

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run $(CMD) init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run $(CMD) seed

.PHONY: test
test: ## Run all tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -v $(PKG)

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

# E2E Testing targets
.PHONY: e2e-setup
e2e-setup: ## Install e2e test dependencies
	@cd e2e && npm install && npx playwright install

.PHONY: e2e
e2e: ## Run e2e tests
	@cd e2e && npm test

.PHONY: e2e-ui
e2e-ui: ## Run e2e tests with Playwright UI
	@cd e2e && npm run test:ui

.PHONY: e2e-debug
e2e-debug: ## Run e2e tests in debug mode
	@cd e2e && npm run test:debug

.PHONY: e2e-headed
e2e-headed: ## Run e2e tests in headed browser mode
	@cd e2e && npm run test:headed

.PHONY: e2e-chromium
e2e-chromium: ## Run e2e tests in Chromium only
	@cd e2e && npm run test:chromium

.PHONY: e2e-firefox
e2e-firefox: ## Run e2e tests in Firefox only
	@cd e2e && npm run test:firefox

.PHONY: e2e-webkit
e2e-webkit: ## Run e2e tests in WebKit only
	@cd e2e && npm run test:webkit

.PHONY: e2e-report
e2e-report: ## Show e2e test report
	@cd e2e && npm run report

.PHONY: help
help: ## Show available commands
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
