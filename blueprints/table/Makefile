.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/table
DATA_DIR ?= $(HOME)/data/blueprint/table

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X github.com/go-mizu/blueprints/table/cli.Version=$(VERSION) -X github.com/go-mizu/blueprints/table/cli.Commit=$(COMMIT) -X github.com/go-mizu/blueprints/table/cli.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

# Frontend configuration
FRONTEND_DIR := app/frontend
BACKEND_DIR := app/backend/hono

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@cd $(FRONTEND_DIR) && pnpm install

.PHONY: frontend-build
frontend-build: ## Build frontend for production
	@cd $(FRONTEND_DIR) && pnpm run build

.PHONY: frontend-dev
frontend-dev: ## Start frontend dev server
	@cd $(FRONTEND_DIR) && pnpm run dev

.PHONY: backend-install
backend-install: ## Install backend dependencies
	@cd $(BACKEND_DIR) && pnpm install

.PHONY: backend-build
backend-build: ## Build backend for production
	@cd $(BACKEND_DIR) && pnpm run build

.PHONY: backend-dev
backend-dev: ## Start backend dev server
	@cd $(BACKEND_DIR) && pnpm run dev

.PHONY: install-deps
install-deps: frontend-install backend-install ## Install all dependencies

.PHONY: build
build: frontend-build backend-build ## Build table binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) .
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install table binary to $$HOME/bin/table (alias for build)

.PHONY: run
run: ## Run the backend server
	@cd $(BACKEND_DIR) && pnpm run dev

.PHONY: dev
dev: ## Run both backend and frontend in dev mode (requires tmux or separate terminals)
	@echo "Starting backend and frontend in dev mode..."
	@echo "Run these in separate terminals:"
	@echo "  make backend-dev"
	@echo "  make frontend-dev"

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run . init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run . seed

.PHONY: serve
serve: ## Start the web server (Go wrapper)
	@GOWORK=off $(GO) run . serve --dev

.PHONY: test
test: ## Run Go tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run tests with verbose output
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

.PHONY: check-deps
check-deps: ## Check if required dependencies are installed
	@echo "Checking dependencies..."
	@echo -n "  Node.js: "
	@if command -v node >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(node --version))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    Install: https://nodejs.org/"; \
	fi
	@echo -n "  pnpm: "
	@if command -v pnpm >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(pnpm --version))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    Install: npm install -g pnpm"; \
	fi
	@echo -n "  Go: "
	@if command -v go >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(go version | cut -d' ' -f3))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    Install: https://go.dev/dl/"; \
	fi
	@echo ""

.PHONY: setup
setup: check-deps install-deps init seed ## Full setup: check deps, install, init db, seed data
	@echo ""
	@echo "Setup complete! Run 'make run' to start the server."
	@echo "Login with: alice@example.com / password123"

.PHONY: reset
reset: clean init seed ## Reset database: clean, init, seed
	@echo "Database reset complete"

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Table Blueprint - Airtable-style collaborative database"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
