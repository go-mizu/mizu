.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/localflare
CMD     ?= ./cmd/localflare
DATA_DIR ?= $(HOME)/data/blueprint/localflare

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

# Frontend configuration
FRONTEND_DIR := app/frontend

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
build: frontend-build ## Build localflare binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: build-quick
build-quick: ## Build binary without frontend (faster for development)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install localflare binary to $$HOME/bin/localflare (alias for build)

.PHONY: run
run: ## Run the server in development mode
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
	@echo ""

.PHONY: dev
dev: ## Show development mode instructions
	@echo ""
	@echo "Development mode:"
	@echo ""
	@echo "  Terminal 1 (Backend):"
	@echo "    make run"
	@echo ""
	@echo "  Terminal 2 (Frontend):"
	@echo "    make frontend-dev"
	@echo ""
	@echo "  Then open http://localhost:5173"
	@echo ""

.PHONY: dev-all
dev-all: ## Run backend and frontend in development mode
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8787"
	@echo "Frontend: http://localhost:5173"
	@echo ""
	@trap 'kill %1 %2 2>/dev/null' EXIT; \
	(cd $(FRONTEND_DIR) && pnpm run dev) & \
	GOWORK=off $(GO) run $(CMD) serve --dev

# LiteIO configuration
LITEIO_BINARY ?= $(HOME)/bin/liteio
LITEIO_IMAGE  ?= liteio:latest
LITEIO_PORT   ?= 9200
LITEIO_DATA   ?= $(HOME)/data/liteio

.PHONY: liteio
liteio: liteio-binary liteio-docker ## Build LiteIO binary and Docker image

.PHONY: liteio-binary
liteio-binary: ## Build LiteIO binary
	@mkdir -p $(dir $(LITEIO_BINARY))
	@CGO_ENABLED=0 GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(LITEIO_BINARY) ./cmd/liteio
	@echo "Built: $(LITEIO_BINARY) ($(VERSION))"

.PHONY: liteio-docker
liteio-docker: ## Build LiteIO Docker image
	@docker build -f liteio.Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(TIME) \
		-t $(LITEIO_IMAGE) .
	@echo "Built: $(LITEIO_IMAGE)"

.PHONY: liteio-run
liteio-run: ## Run LiteIO server locally
	@mkdir -p $(LITEIO_DATA)
	@$(LITEIO_BINARY) --port $(LITEIO_PORT) --data-dir $(LITEIO_DATA)

.PHONY: liteio-docker-run
liteio-docker-run: ## Run LiteIO in Docker container
	@docker run -d --name liteio \
		-p $(LITEIO_PORT):9000 \
		-v $(LITEIO_DATA):/data \
		$(LITEIO_IMAGE)
	@echo "LiteIO running at http://localhost:$(LITEIO_PORT)"

.PHONY: liteio-docker-stop
liteio-docker-stop: ## Stop LiteIO Docker container
	@docker stop liteio 2>/dev/null || true
	@docker rm liteio 2>/dev/null || true
	@echo "LiteIO container stopped"

.PHONY: liteio-bench
liteio-bench: ## Run benchmarks against LiteIO
	@echo "Running benchmarks against LiteIO at localhost:$(LITEIO_PORT)..."
	@cd pkg/storage/bench && GOWORK=off $(GO) test -bench=. -benchmem -benchtime=3s ./... 2>&1 | tee ../report/liteio_raw.txt
	@echo "Benchmark complete. Raw output in pkg/storage/report/liteio_raw.txt"

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Localflare - Offline Cloudflare Clone"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
