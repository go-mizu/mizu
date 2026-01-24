.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/search
CMD     ?= ./cmd/search
DATA_DIR ?= $(HOME)/data/blueprint/search

CGO_ENABLED ?= 0

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

# Frontend configuration
FRONTEND_DIR := app/frontend

# Database configuration
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= search
DB_PASS ?= search
DB_NAME ?= search
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@cd $(FRONTEND_DIR) && pnpm install

.PHONY: frontend-build
frontend-build: ## Build frontend for production
	@cd $(FRONTEND_DIR) && pnpm run build

.PHONY: frontend-dev
frontend-dev: ## Start frontend dev server
	@cd $(FRONTEND_DIR) && pnpm run dev

.PHONY: frontend-test
frontend-test: ## Run frontend tests
	@cd $(FRONTEND_DIR) && pnpm run test

.PHONY: build
build: frontend-build ## Build search binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: build-quick
build-quick: ## Build binary without frontend (faster for development)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install search binary to $$HOME/bin/search (alias for build)

.PHONY: run
run: ## Run the server in development mode
	@GOWORK=off $(GO) run $(CMD) serve --dev

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run $(CMD) init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run $(CMD) seed

.PHONY: crawl
crawl: ## Run the web crawler
	@GOWORK=off $(GO) run $(CMD) crawl

.PHONY: test
test: ## Run all Go tests
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

.PHONY: tidy
tidy: ## Run go mod tidy
	@$(GO) mod tidy

.PHONY: check-deps
check-deps: ## Check if required dependencies are installed
	@echo "Checking dependencies..."
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
	fi
	@echo -n "  PostgreSQL: "
	@if command -v psql >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(psql --version | head -1))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    Install: brew install postgresql"; \
	fi
	@echo ""

.PHONY: dev
dev: ## Show development mode instructions
	@echo ""
	@echo "Search - Full-Featured Search Engine"
	@echo ""
	@echo "Development mode:"
	@echo ""
	@echo "  Terminal 1 (PostgreSQL):"
	@echo "    make docker-up"
	@echo ""
	@echo "  Terminal 2 (Backend):"
	@echo "    make run"
	@echo ""
	@echo "  Terminal 3 (Frontend):"
	@echo "    make frontend-dev"
	@echo ""
	@echo "  Then open http://localhost:5173"
	@echo ""
	@echo "  Or run everything at once:"
	@echo "    make dev-all"
	@echo ""

.PHONY: dev-all
dev-all: ## Run backend and frontend in development mode
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173"
	@echo ""
	@trap 'kill %1 %2 2>/dev/null' EXIT; \
	(cd $(FRONTEND_DIR) && pnpm run dev) & \
	GOWORK=off $(GO) run $(CMD) serve --dev

# Docker targets
DOCKER_COMPOSE_POSTGRES := docker/postgres/docker-compose.yml

.PHONY: docker-up
docker-up: ## Start PostgreSQL with Docker
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) up -d
	@echo "PostgreSQL started at localhost:$(DB_PORT)"
	@echo "Connection: $(DATABASE_URL)"

.PHONY: docker-down
docker-down: ## Stop PostgreSQL Docker
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) down

.PHONY: docker-logs
docker-logs: ## View PostgreSQL logs
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) logs -f

.PHONY: docker-psql
docker-psql: ## Connect to PostgreSQL via psql
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -d $(DB_NAME)

.PHONY: docker-build
docker-build: ## Build Docker images
	@docker build -t search:latest .
	@echo "Built: search:latest"

# Database management
.PHONY: db-reset
db-reset: ## Reset the database (drop and recreate)
	@echo "Resetting database..."
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);"
	@echo "Database reset complete"

.PHONY: db-create-extensions
db-create-extensions: ## Create PostgreSQL extensions
	@echo "Creating extensions..."
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -d $(DB_NAME) -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -d $(DB_NAME) -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
	@docker compose -f $(DOCKER_COMPOSE_POSTGRES) exec postgres psql -U $(DB_USER) -d $(DB_NAME) -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
	@echo "Extensions created"

# Quick start
.PHONY: quickstart
quickstart: docker-up ## Quick start: start database, initialize, seed, and run
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@make init
	@make seed
	@make run

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Search - Full-Featured Search Engine"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
