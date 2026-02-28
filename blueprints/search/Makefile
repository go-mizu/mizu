.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/search
LINUX_BINARY ?= $(HOME)/bin/search-linux
LINUX_RUNTIME_LIB_DIR ?= $(HOME)/bin/search-linux-libs-focal-gcc11
LINUX_RUNTIME_LIBSTDCPP ?= $(LINUX_RUNTIME_LIB_DIR)/libstdc++.so.6
# Noble (Ubuntu 24.04) build — no bundled libstdc++ needed (system ships it)
LINUX_BINARY_NOBLE ?= $(HOME)/bin/search-linux-noble
CMD     ?= ./cmd/search
DATA_DIR ?= $(HOME)/data/blueprint/search
DEPLOY_KEY    ?= $(HOME)/.ssh/id_ed25519_deploy
SERVER1_USER  ?= tam
SERVER1_HOST  ?= server
SERVER2_USER  ?= root
SERVER2_HOST  ?= server2
# Pass SERVER=2 to any remote/deploy target to operate on server2
SERVER        ?= 1
REMOTE_USER   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_USER),$(SERVER1_USER))
REMOTE_HOST   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_HOST),$(SERVER1_HOST))
REMOTE_SSH     = $(REMOTE_USER)@$(REMOTE_HOST)
SSH           ?= ssh -i $(DEPLOY_KEY) -o BatchMode=yes
SCP           ?= scp -i $(DEPLOY_KEY)

CGO_ENABLED ?= 1
# Focal (Ubuntu 20.04) build vars — max compatibility (glibc 2.31)
LINUX_BASE_IMAGE ?= ubuntu:20.04
LINUX_GO_VERSION ?= 1.26.0
LINUX_GOARCH ?= amd64
LINUX_DOCKERFILE ?= Dockerfile.linux-focal
# Noble (Ubuntu 24.04) build vars — for Ubuntu 24.04 servers (glibc 2.39)
LINUX_NOBLE_DOCKERFILE ?= Dockerfile.linux-noble
LINUX_NOBLE_IMAGE_TAG ?= mizu-search-linux-noble-build:local
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
		cid="$$($(LINUX_CONTAINER) create --platform "linux/$(LINUX_GOARCH)" "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux" "$$tmp_out/search-linux"; \
		install -m 0755 "$$tmp_out/search-linux" "$(LINUX_BINARY)"
	@echo "Built Linux binary: $(LINUX_BINARY) ($(VERSION))"

.PHONY: build-linux-noble
build-linux-noble: ## Build Ubuntu 24.04 (Noble) binary to $$HOME/bin/search-linux-noble (Docker, CGO enabled)
	@mkdir -p $(dir $(LINUX_BINARY_NOBLE))
	@tmp_out="$$(mktemp -d)"; \
		img="$(LINUX_NOBLE_IMAGE_TAG)"; \
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
			--file "$(LINUX_NOBLE_DOCKERFILE)" \
			--ignorefile "$(LINUX_DOCKERFILE).dockerignore" \
			--build-arg GO_VERSION="$(LINUX_GO_VERSION)" \
			--build-arg TARGETARCH="$(LINUX_GOARCH)" \
			--build-arg GO_LDFLAGS="$(GO_LDFLAGS)" \
			--target artifact \
			-t "$$img" \
			"$(LINUX_BUILD_CONTEXT)"; \
		cid="$$($(LINUX_CONTAINER) create --platform "linux/$(LINUX_GOARCH)" "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux-noble" "$$tmp_out/search-linux-noble"; \
		install -m 0755 "$$tmp_out/search-linux-noble" "$(LINUX_BINARY_NOBLE)"
	@echo "Built Noble binary: $(LINUX_BINARY_NOBLE) ($(VERSION))"

.PHONY: deploy-linux-noble
deploy-linux-noble: build-linux-noble ## Deploy Ubuntu 24.04 binary to remote $$HOME/bin/search (no bundled libstdc++)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin"
	@$(SCP) "$(LINUX_BINARY_NOBLE)" $(REMOTE_SSH):.search-upload.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux-noble; \
		rm -f $$HOME/.search-upload.tmp; \
		printf '"'"'#!/usr/bin/env bash\nulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=2000000000\nexport GODEBUG=netdns=go\nexec \"$$HOME/bin/search-linux-noble\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed Noble: $(REMOTE_SSH):~/bin/search"

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
		printf '"'"'#!/usr/bin/env bash\nulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=2000000000\nexport GODEBUG=netdns=go\nexport LD_LIBRARY_PATH=\"$$HOME/bin/search-libs\$${LD_LIBRARY_PATH:+:$$LD_LIBRARY_PATH}\"\nexec \"$$HOME/bin/search-linux\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed: $(REMOTE_SSH):~/bin/search"

.PHONY: deploy-key-install
deploy-key-install: ## Authorize DEPLOY_KEY on remote server (SERVER=2 for server2)
	@test -f "$(DEPLOY_KEY).pub" || { echo "Deploy public key not found: $(DEPLOY_KEY).pub"; exit 1; }
	@echo "Installing deploy key on $(REMOTE_SSH)..."
	@ssh $(REMOTE_SSH) \
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys" \
		< "$(DEPLOY_KEY).pub"
	@echo "Done. Test with: make remote-search SERVER=$(SERVER)"

.PHONY: remote-search
remote-search: ## Run remote `search --help`
	@$(SSH) $(REMOTE_SSH) 'bash -lc "timeout 20s ~/bin/search --help | head -40; echo EXIT:$$?"'

.PHONY: remote-recrawl
remote-recrawl: ## Run cc recrawl --file p:0 (full body, 500 workers) on remote (foreground)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search cc recrawl --file p:0 --workers 500"'

.PHONY: remote-recrawl-bg
remote-recrawl-bg: ## Run cc recrawl --file p:0 (full body, 500 workers) on remote (background, ~/recrawl.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:0 --workers 500 >~/recrawl.log 2>&1 & echo PID:$$!"'

.PHONY: remote-tail
remote-tail: ## Tail recrawl log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/recrawl.log'

CC_RECRAWL_FLAGS ?= --domain-fail-threshold 0 --domain-timeout 0 --batch-size 5000 --workers 4096 --timeout 2000 --max-conns-per-domain 2

.PHONY: remote-cc-recrawl-p0
remote-cc-recrawl-p0: ## Launch cc recrawl --file p:0 in background (~/cc-p0.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:0 $(CC_RECRAWL_FLAGS) >~/cc-p0.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-p100
remote-cc-recrawl-p100: ## Launch cc recrawl --file p:100 in background (~/cc-p100.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:100 $(CC_RECRAWL_FLAGS) >~/cc-p100.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-p200
remote-cc-recrawl-p200: ## Launch cc recrawl --file p:200 in background (~/cc-p200.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:200 $(CC_RECRAWL_FLAGS) >~/cc-p200.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-p299
remote-cc-recrawl-p299: ## Launch cc recrawl --file p:299 in background (~/cc-p299.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:299 $(CC_RECRAWL_FLAGS) >~/cc-p299.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-p400
remote-cc-recrawl-p400: ## Launch cc recrawl --file p:400 in background (~/cc-p400.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:400 $(CC_RECRAWL_FLAGS) >~/cc-p400.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-p600
remote-cc-recrawl-p600: ## Launch cc recrawl --file p:600 in background (~/cc-p600.log)
	@$(SSH) $(REMOTE_SSH) 'nohup ~/bin/search cc recrawl --file p:600 $(CC_RECRAWL_FLAGS) >~/cc-p600.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-recrawl-all
remote-cc-recrawl-all: remote-cc-recrawl-p0 remote-cc-recrawl-p200 \
                        remote-cc-recrawl-p400 remote-cc-recrawl-p600 ## Launch all 4 cc recrawl partitions in background (parallel)

.PHONY: remote-cc-recrawl-sequential
remote-cc-recrawl-sequential: ## Run p:0 → p:100 → p:200 → p:299 sequentially in background (safer for low-RAM servers)
	@$(SSH) $(REMOTE_SSH) 'nohup bash -c " \
		~/bin/search cc recrawl --file p:0   $(CC_RECRAWL_FLAGS) >~/cc-p0.log   2>&1; \
		~/bin/search cc recrawl --file p:100 $(CC_RECRAWL_FLAGS) >~/cc-p100.log 2>&1; \
		~/bin/search cc recrawl --file p:200 $(CC_RECRAWL_FLAGS) >~/cc-p200.log 2>&1; \
		~/bin/search cc recrawl --file p:299 $(CC_RECRAWL_FLAGS) >~/cc-p299.log 2>&1; \
		echo ALL_DONE \
	" >~/cc-sequential.log 2>&1 & echo PID:$$!'

.PHONY: remote-cc-tail-p0
remote-cc-tail-p0: ## Tail cc recrawl p:0 log
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p0.log'

.PHONY: remote-cc-tail-p100
remote-cc-tail-p100: ## Tail cc recrawl p:100 log
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p100.log'

.PHONY: remote-cc-tail-p200
remote-cc-tail-p200: ## Tail cc recrawl p:200 log
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p200.log'

.PHONY: remote-cc-tail-p299
remote-cc-tail-p299: ## Tail cc recrawl p:299 log
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p299.log'

.PHONY: remote-cc-status
remote-cc-status: ## Show last 2 lines of all active cc recrawl logs
	@$(SSH) $(REMOTE_SSH) 'tail -n2 ~/cc-p0.log ~/cc-p100.log ~/cc-p200.log ~/cc-p299.log 2>/dev/null'

.PHONY: remote-hn-recrawl
remote-hn-recrawl: ## Run hn recrawl (keepalive, auto-config) on remote server
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search hn recrawl"'

.PHONY: remote-hn-recrawl-bg
remote-hn-recrawl-bg: ## Run hn recrawl (keepalive, auto-config) in background (~/hn-recrawl.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search hn recrawl >~/hn-recrawl.log 2>&1 & echo PID:$$!"'

.PHONY: remote-hn-recrawl-swarm
remote-hn-recrawl-swarm: ## Run hn recrawl --engine swarm (body mode, auto-config) on remote server
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search hn recrawl --engine swarm --domain-timeout 30000"'

.PHONY: remote-hn-recrawl-swarm-bg
remote-hn-recrawl-swarm-bg: ## Run hn recrawl swarm in background (~/hn-swarm.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search hn recrawl --engine swarm --domain-timeout 30000 >~/hn-swarm.log 2>&1 & echo PID:$$!"'

.PHONY: seed-copy
seed-copy: ## Copy HN hn_domains.duckdb from server1 → server2 via local pipe (1.2 GB)
	@echo "Preparing server2..."
	@$(SSH) $(SERVER2_USER)@$(SERVER2_HOST) 'mkdir -p ~/data/hn/recrawl'
	@echo "Streaming hn_domains.duckdb: $(SERVER1_USER)@$(SERVER1_HOST) → local → $(SERVER2_USER)@$(SERVER2_HOST)..."
	@$(SSH) $(SERVER1_USER)@$(SERVER1_HOST) "cat ~/data/hn/hn_domains.duckdb" | \
		$(SSH) $(SERVER2_USER)@$(SERVER2_HOST) "cat > ~/data/hn/hn_domains.duckdb"
	@echo "Done. Run: make remote-hn-recrawl SERVER=2"

.PHONY: remote-hn-tail
remote-hn-tail: ## Tail hn swarm recrawl log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/hn-swarm.log'

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

.PHONY: bench-chunk
bench-chunk: ## Run all 3 chunk modes with 200K seeds and compare results
	@for mode in stream batch pipeline; do \
		echo "=== Running chunk mode: $$mode ==="; \
		rm -f $(DATA_DIR)/hn/recrawl/results/*.duckdb; \
		$(BINARY) hn recrawl --limit 200000 --no-retry --chunk-mode $$mode; \
	done
	@echo ""
	@echo "=== Benchmark Results ==="
	@for mode in stream batch pipeline; do \
		echo "--- $$mode ---"; \
		cat $(DATA_DIR)/hn/recrawl/bench_chunk_$$mode.json 2>/dev/null || echo "(not found)"; \
	done

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Search - Full-Featured Search Engine"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
