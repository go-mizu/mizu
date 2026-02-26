.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/search
LINUX_BINARY ?= $(HOME)/bin/search-linux
LINUX_RUNTIME_LIB_DIR ?= $(HOME)/bin/search-linux-libs-focal-gcc11
LINUX_RUNTIME_LIBSTDCPP ?= $(LINUX_RUNTIME_LIB_DIR)/libstdc++.so.6
CMD     ?= ./cmd/search
DATA_DIR ?= $(HOME)/data/blueprint/search
REMOTE_USER ?= tam
REMOTE_HOST ?= server
REMOTE_BIN ?= $(HOME)/bin/search
REMOTE_SSH ?= $(REMOTE_USER)@$(REMOTE_HOST)
DEPLOY_KEY ?= $(HOME)/.ssh/id_ed25519_deploy
SSH ?= ssh -i $(DEPLOY_KEY) -o BatchMode=yes
SCP ?= scp -i $(DEPLOY_KEY)

CGO_ENABLED ?= 1
LINUX_BASE_IMAGE ?= ubuntu:20.04
LINUX_GO_VERSION ?= 1.26.0
LINUX_GOARCH ?= amd64
LINUX_DOCKERFILE ?= Dockerfile.linux
LINUX_BUILD_CONTEXT ?= $(abspath ../..)
LINUX_IGNOREFILE ?= $(LINUX_DOCKERFILE).dockerignore
LINUX_CONTAINER ?= podman
LINUX_IMAGE_TAG ?= mizu-search-linux-build:local

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

# Frontend configuration
FRONTEND_DIR := app/frontend

# Database configuration (SQLite)
DATABASE_PATH ?= $(HOME)/data/blueprints/search/search.db

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

.PHONY: build-linux
build-linux: ## Build Ubuntu/Linux binary to $$HOME/bin/search-linux (Docker, CGO enabled)
	@mkdir -p $(dir $(LINUX_BINARY))
	@tmp_out="$$(mktemp -d)"; \
		img="$(LINUX_IMAGE_TAG)"; \
		cid=""; \
		set -e; \
		trap 'test -n "$$cid" && $(LINUX_CONTAINER) rm -f "$$cid" >/dev/null 2>&1 || true; rm -rf "$$tmp_out"' EXIT; \
		command -v "$(LINUX_CONTAINER)" >/dev/null 2>&1 || { echo "$(LINUX_CONTAINER) not found"; exit 127; }; \
		if [ "$(LINUX_CONTAINER)" = "podman" ]; then \
			if ! podman machine inspect >/dev/null 2>&1; then \
				echo "Initializing podman machine..."; \
				podman machine init; \
			fi; \
			if ! podman info >/dev/null 2>&1; then \
				echo "Starting podman machine..."; \
				podman machine start >/dev/null; \
			fi; \
			for i in $$(seq 1 30); do \
				podman info >/dev/null 2>&1 && break; \
				sleep 1; \
			done; \
			podman info >/dev/null 2>&1 || { echo "podman not ready"; exit 1; }; \
		fi; \
		$(LINUX_CONTAINER) build \
			--layers \
			--platform "linux/$(LINUX_GOARCH)" \
			--file "$(LINUX_DOCKERFILE)" \
			--ignorefile "$(LINUX_IGNOREFILE)" \
			--build-arg BASE_IMAGE="$(LINUX_BASE_IMAGE)" \
			--build-arg GO_VERSION="$(LINUX_GO_VERSION)" \
			--build-arg TARGETARCH="$(LINUX_GOARCH)" \
			--build-arg GO_LDFLAGS="$(GO_LDFLAGS)" \
			--target artifact \
			-t "$$img" \
			"$(LINUX_BUILD_CONTEXT)"; \
		cid="$$($(LINUX_CONTAINER) create "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux" "$$tmp_out/search-linux"; \
		install -m 0755 "$$tmp_out/search-linux" "$(LINUX_BINARY)"
	@echo "Built Linux binary: $(LINUX_BINARY) ($(VERSION))"

.PHONY: deploy-linux
deploy-linux: build-linux ## Deploy linux binary to remote $$HOME/bin/search
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@test -f "$(LINUX_RUNTIME_LIBSTDCPP)" || { echo "Missing runtime lib: $(LINUX_RUNTIME_LIBSTDCPP)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin \$$HOME/bin/search-libs"
	@$(SCP) "$(LINUX_BINARY)" $(REMOTE_SSH):.search-upload.tmp
	@$(SCP) "$(LINUX_RUNTIME_LIBSTDCPP)" $(REMOTE_SSH):.libstdc++.so.6.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux; \
		install -m 0644 $$HOME/.libstdc++.so.6.tmp $$HOME/bin/search-libs/libstdc++.so.6; \
		rm -f $$HOME/.search-upload.tmp $$HOME/.libstdc++.so.6.tmp; \
		printf '"'"'#!/usr/bin/env bash\nulimit -n 65536 2>/dev/null || true\nexport LD_LIBRARY_PATH=\"$$HOME/bin/search-libs\$${LD_LIBRARY_PATH:+:$$LD_LIBRARY_PATH}\"\nexec \"$$HOME/bin/search-linux\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed: $(REMOTE_SSH):~/bin/search"

.PHONY: remote-search
remote-search: ## Run remote `search --help`
	@$(SSH) $(REMOTE_SSH) 'bash -lc "timeout 20s ~/bin/search --help | head -40; echo EXIT:$$?"'

.PHONY: remote-recrawl
remote-recrawl: ## Run cc recrawl --file p:0 --status-only on remote (foreground, 1500 workers)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search cc recrawl --file p:0 --status-only --workers 1500"'

.PHONY: remote-recrawl-bg
remote-recrawl-bg: ## Run cc recrawl --file p:0 --status-only on remote (background, ~/recrawl.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 >~/recrawl.log 2>&1 & echo PID:$$!"'

.PHONY: remote-tail
remote-tail: ## Tail recrawl log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/recrawl.log'

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
	@echo ""

.PHONY: dev
dev: ## Show development mode instructions
	@echo ""
	@echo "Search - Full-Featured Search Engine (SQLite)"
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

# Database management
.PHONY: db-reset
db-reset: ## Reset the SQLite database
	@echo "Resetting database..."
	@rm -f $(DATABASE_PATH)
	@echo "Database reset complete"

# Quick start
.PHONY: quickstart
quickstart: ## Quick start: initialize, seed, and run
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
