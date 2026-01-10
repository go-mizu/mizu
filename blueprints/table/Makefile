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

# Frontend/Backend configuration
FRONTEND_DIR := app/frontend
BACKEND_DIR := app/backend/hono

# Ports
GO_PORT ?= 8080
BACKEND_PORT ?= 3000
FRONTEND_PORT ?= 5173

# ============================================================================
# Dependencies
# ============================================================================

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@cd $(FRONTEND_DIR) && pnpm install

.PHONY: backend-install
backend-install: ## Install backend dependencies
	@cd $(BACKEND_DIR) && pnpm install

.PHONY: install-deps
install-deps: frontend-install backend-install ## Install all dependencies

# ============================================================================
# Build
# ============================================================================

.PHONY: frontend-build
frontend-build: ## Build frontend for production
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm run build
	@echo "Frontend built to assets/static/dist"

.PHONY: backend-build
backend-build: ## Build backend TypeScript
	@cd $(BACKEND_DIR) && pnpm run build 2>/dev/null || true

.PHONY: build
build: frontend-build ## Build table binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) .
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install table binary to $$HOME/bin/table (alias for build)

# ============================================================================
# Development
# ============================================================================

.PHONY: frontend-dev
frontend-dev: ## Start frontend dev server (port 5173)
	@echo "Starting frontend dev server on http://localhost:$(FRONTEND_PORT)"
	@echo "API requests will be proxied to http://localhost:$(BACKEND_PORT)"
	@cd $(FRONTEND_DIR) && pnpm run dev

.PHONY: backend-dev
backend-dev: ## Start backend dev server (port 3000)
	@echo "Starting backend dev server on http://localhost:$(BACKEND_PORT)"
	@cd $(BACKEND_DIR) && PORT=$(BACKEND_PORT) DATA_DIR=$(DATA_DIR) pnpm run dev

.PHONY: run
run: ## Run backend server directly (development mode)
	@cd $(BACKEND_DIR) && PORT=$(BACKEND_PORT) DATA_DIR=$(DATA_DIR) pnpm run dev

.PHONY: dev
dev: ## Instructions to run in dev mode
	@echo ""
	@echo "\033[36mDevelopment Mode\033[0m"
	@echo ""
	@echo "Run these commands in separate terminals:"
	@echo ""
	@echo "  \033[32m# Terminal 1: Backend API server (port $(BACKEND_PORT))\033[0m"
	@echo "  make backend-dev"
	@echo ""
	@echo "  \033[32m# Terminal 2: Frontend dev server (port $(FRONTEND_PORT))\033[0m"
	@echo "  make frontend-dev"
	@echo ""
	@echo "Then open http://localhost:$(FRONTEND_PORT) in your browser."
	@echo ""

.PHONY: dev-all
dev-all: ## Run both backend and frontend in parallel (requires tmux or background jobs)
	@echo "Starting backend and frontend..."
	@trap 'kill 0' INT; \
		(cd $(BACKEND_DIR) && PORT=$(BACKEND_PORT) DATA_DIR=$(DATA_DIR) pnpm run dev) & \
		sleep 2 && \
		(cd $(FRONTEND_DIR) && pnpm run dev) & \
		wait

# ============================================================================
# Database & CLI
# ============================================================================

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run . init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run . seed

.PHONY: serve
serve: ## Start the web server (Go wrapper that embeds frontend and proxies API)
	@echo "Starting Table server..."
	@echo "Go server on http://localhost:$(GO_PORT) (serves static files)"
	@echo "Node.js backend on http://localhost:$(BACKEND_PORT) (API)"
	@GOWORK=off $(GO) run . serve --dev --addr :$(GO_PORT)

# ============================================================================
# Testing
# ============================================================================

.PHONY: test
test: ## Run Go tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -v $(PKG)

.PHONY: frontend-test
frontend-test: ## Run frontend tests
	@cd $(FRONTEND_DIR) && pnpm run test

# ============================================================================
# Cleanup & Utilities
# ============================================================================

.PHONY: clean
clean: ## Remove binary and data directory
	@rm -f $(BINARY)
	@rm -rf $(DATA_DIR)
	@echo "Cleaned: $(BINARY), $(DATA_DIR)"

.PHONY: clean-build
clean-build: ## Remove built assets only
	@rm -rf assets/static/dist
	@rm -f $(BINARY)
	@echo "Cleaned build artifacts"

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
	@echo "\033[32mSetup complete!\033[0m"
	@echo ""
	@echo "To start in development mode:"
	@echo "  make dev        # Shows instructions for running backend + frontend"
	@echo ""
	@echo "To start with Go server (production-like):"
	@echo "  make serve      # Go serves static files and proxies API"
	@echo ""
	@echo "Login with: alice@example.com / password123"

.PHONY: reset
reset: clean init seed ## Reset database: clean, init, seed
	@echo "Database reset complete"

.PHONY: rebuild
rebuild: clean-build build ## Clean and rebuild everything

# ============================================================================
# Help
# ============================================================================

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "\033[36mTable Blueprint - Airtable-style collaborative database\033[0m"
	@echo ""
	@echo "\033[33mDevelopment:\033[0m"
	@grep -E '^(frontend-dev|backend-dev|dev|dev-all|run):.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[33mBuild:\033[0m"
	@grep -E '^(frontend-build|backend-build|build|install):.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[33mDatabase & Server:\033[0m"
	@grep -E '^(init|seed|serve):.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[33mTesting:\033[0m"
	@grep -E '^(test|test-v|frontend-test):.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[33mSetup & Utilities:\033[0m"
	@grep -E '^(setup|reset|clean|rebuild|install-deps|check-deps|update):.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
