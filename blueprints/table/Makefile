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
ASSETS_DIR := assets/static

# Ports
PORT ?= 8080
FRONTEND_PORT ?= 5173

# ============================================================================
# Dependencies
# ============================================================================

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@cd $(FRONTEND_DIR) && pnpm install

.PHONY: install-deps
install-deps: frontend-install ## Install all dependencies

# ============================================================================
# Build
# ============================================================================

.PHONY: frontend-build
frontend-build: ## Build frontend for production
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm run build
	@echo "Frontend built to $(ASSETS_DIR)"

.PHONY: build
build: frontend-build ## Build table binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) .
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: build-quick
build-quick: ## Build table binary without frontend (for development)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) .
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install table binary to $$HOME/bin/table (alias for build)

# ============================================================================
# Development
# ============================================================================

.PHONY: frontend-dev
frontend-dev: ## Start frontend dev server (port 5173) with HMR
	@echo ""
	@echo "\033[36m┌─────────────────────────────────────────────────────────┐\033[0m"
	@echo "\033[36m│           Table Blueprint - Frontend Dev Mode           │\033[0m"
	@echo "\033[36m└─────────────────────────────────────────────────────────┘\033[0m"
	@echo ""
	@echo "  \033[32mFrontend:\033[0m  http://localhost:$(FRONTEND_PORT)"
	@echo "  \033[33mBackend:\033[0m   http://localhost:$(PORT) (proxied)"
	@echo ""
	@echo "  \033[90mFeatures: HMR enabled, API proxy active\033[0m"
	@echo "  \033[90mLogin: alice@example.com / password123\033[0m"
	@echo ""
	@cd $(FRONTEND_DIR) && pnpm run dev

.PHONY: backend-dev
backend-dev: ## Start Go backend server (port 8080)
	@echo ""
	@echo "\033[36m┌─────────────────────────────────────────────────────────┐\033[0m"
	@echo "\033[36m│           Table Blueprint - Backend Dev Mode            │\033[0m"
	@echo "\033[36m└─────────────────────────────────────────────────────────┘\033[0m"
	@echo ""
	@echo "  \033[32mAPI Server:\033[0m  http://localhost:$(PORT)"
	@echo "  \033[33mDatabase:\033[0m    $(DATA_DIR)/table.db"
	@echo ""
	@GOWORK=off $(GO) run . serve --addr :$(PORT) --dev

.PHONY: run
run: ## Run Go server directly (development mode)
	@GOWORK=off $(GO) run . serve --addr :$(PORT)

.PHONY: dev
dev: ## Instructions to run in dev mode
	@echo ""
	@echo "\033[36m┌─────────────────────────────────────────────────────────┐\033[0m"
	@echo "\033[36m│        Table Blueprint - Development Mode Setup         │\033[0m"
	@echo "\033[36m└─────────────────────────────────────────────────────────┘\033[0m"
	@echo ""
	@echo "\033[33m1. Start Go Backend Server\033[0m (Terminal 1)"
	@echo ""
	@echo "   \033[32mmake backend-dev\033[0m"
	@echo "   → Runs on http://localhost:$(PORT)"
	@echo ""
	@echo "\033[33m2. Start Frontend Dev Server\033[0m (Terminal 2)"
	@echo ""
	@echo "   \033[32mmake frontend-dev\033[0m"
	@echo "   → Runs on http://localhost:$(FRONTEND_PORT) with HMR"
	@echo ""
	@echo "\033[33m3. Open Browser\033[0m"
	@echo ""
	@echo "   → http://localhost:$(FRONTEND_PORT)"
	@echo "   → Login: alice@example.com / password123"
	@echo ""
	@echo "\033[90mFeatures to explore:\033[0m"
	@echo "   • 7 View Types: Grid, Kanban, Calendar, Gallery, Timeline, Form, List"
	@echo "   • 15 Field Types: Text, Select, Date, Rating, Checkbox, Currency, etc."
	@echo "   • Filtering, Sorting, Grouping"
	@echo "   • Comments & Activity tracking"
	@echo ""

.PHONY: dev-all
dev-all: ## Run both backend and frontend in parallel
	@echo ""
	@echo "\033[36mStarting Table Blueprint in Development Mode...\033[0m"
	@echo ""
	@trap 'kill 0' INT; \
		(GOWORK=off $(GO) run . serve --addr :$(PORT)) & \
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
serve: ## Start the web server (Go server serves API + embedded frontend)
	@echo "Starting Table server..."
	@echo "Server: http://localhost:$(PORT)"
	@GOWORK=off $(GO) run . serve --addr :$(PORT)

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
	@rm -rf $(ASSETS_DIR)
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
	@echo "To start production server:"
	@echo "  make serve      # Go serves static files and API"
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
	@grep -E '^(frontend-build|build|build-quick|install):.*?## ' $(MAKEFILE_LIST) | \
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
