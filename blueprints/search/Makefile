.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/search
LINUX_BINARY ?= $(HOME)/bin/search-linux
LINUX_RUNTIME_LIB_DIR ?= $(HOME)/bin/search-linux-libs-focal-gcc13
LINUX_RUNTIME_LIBSTDCPP ?= $(LINUX_RUNTIME_LIB_DIR)/libstdc++.so.6
# Noble (Ubuntu 24.04) build — no bundled libstdc++ needed (system ships it)
LINUX_BINARY_NOBLE ?= $(HOME)/bin/search-linux-noble
CMD     ?= ./cmd/search
DATA_DIR ?= $(HOME)/data/blueprint/search
DEPLOY_KEY    ?= $(HOME)/.ssh/id_ed25519_deploy
# Both servers are Ubuntu 24.04 (Noble): server1 upgraded 2026-03, server2 always Noble
SERVER1_USER  ?= tam
SERVER1_HOST  ?= server
SERVER2_USER  ?= root
SERVER2_HOST  ?= server2
# Pass SERVER=1 to any remote/deploy target to operate on server1
SERVER        ?= 2
REMOTE_USER   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_USER),$(SERVER1_USER))
REMOTE_HOST   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_HOST),$(SERVER1_HOST))
REMOTE_SSH     = $(REMOTE_USER)@$(REMOTE_HOST)
SSH           ?= ssh -i $(DEPLOY_KEY) -o BatchMode=yes
SCP           ?= scp -i $(DEPLOY_KEY)

CGO_ENABLED ?= 1
# macOS only: Apple ld64 (Xcode 15+) auto-links -lc++ for C++ symbols AND sees the
# explicit -lc++ from duckdb-go-bindings/lib/darwin-arm64/prebuilt.go, producing:
#   ld: warning: ignoring duplicate libraries: '-lc++'
# Pass -no_warn_duplicate_libraries via CGO_LDFLAGS so it reaches the CGo cc step.
# On Linux, GNU ld doesn't auto-add -lc++ so this flag is not needed.
ifeq ($(shell uname -s),Darwin)
CGO_LDFLAGS ?= -Wl,-no_warn_duplicate_libraries
endif
# chdb support: set CHDB_LIB_DIR to the directory containing libchdb.so to
# enable the chdb build tag. e.g. make build-chdb CHDB_LIB_DIR=$$HOME/lib
CHDB_LIB_DIR ?= $(HOME)/lib
CHDB_HEADER_DIR ?= /tmp/libchdb
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
build: ## Build search binary (Go only; run frontend-build separately if needed)
	@mkdir -p $(dir $(BINARY))
	@echo "==> Compiling $(CMD) → $(BINARY)"
	@echo "    CGO_ENABLED=$(CGO_ENABLED)  version=$(VERSION)"
	@echo "    (CGO+DuckDB static link — first build may take 1-2 min)"
	CGO_ENABLED=$(CGO_ENABLED) CGO_LDFLAGS="$(CGO_LDFLAGS)" GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "==> Built: $(BINARY) ($(VERSION))"

.PHONY: build-chdb
build-chdb: ## Build search binary with chdb support (requires CHDB_LIB_DIR/libchdb.so)
	@mkdir -p $(dir $(BINARY))
	@echo "==> Compiling $(CMD) → $(BINARY) [+chdb]"
	@echo "    CHDB_LIB_DIR=$(CHDB_LIB_DIR)  version=$(VERSION)"
	@echo "    (CGO+DuckDB+chdb static link — may take 1-2 min)"
	CGO_ENABLED=1 \
		CGO_LDFLAGS="$(CGO_LDFLAGS) -L$(CHDB_LIB_DIR) -lchdb -Wl,-rpath,$(CHDB_LIB_DIR)" \
		CGO_CFLAGS="-I$(CHDB_HEADER_DIR)" \
		DYLD_LIBRARY_PATH="$(CHDB_LIB_DIR):$$DYLD_LIBRARY_PATH" \
		GOWORK=off $(GO) build -tags chdb -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "==> Built: $(BINARY) ($(VERSION)) [+chdb]"

.PHONY: build-quick
build-quick: build ## Alias for build (kept for compatibility)

.PHONY: build-linux
build-linux: ## Build Ubuntu/Linux binary to $$HOME/bin/search-linux (Docker, CGO enabled)
	@mkdir -p $(dir $(LINUX_BINARY))
	@echo "==> Building Linux (focal) binary via $(LINUX_CONTAINER)"
	@echo "    image=$(LINUX_IMAGE_TAG)  arch=$(LINUX_GOARCH)  version=$(VERSION)"
	@echo "    (QEMU cross-compile — may take 15-20 min on cold cache)"
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
		echo "==> [1/3] Building container image $$img ..."; \
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
		echo "==> [2/3] Extracting binary from container..."; \
		cid="$$($(LINUX_CONTAINER) create --platform "linux/$(LINUX_GOARCH)" "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux" "$$tmp_out/search-linux"; \
		$(LINUX_CONTAINER) cp "$$cid:/libstdc++.so.6" "$$tmp_out/libstdc++.so.6" 2>/dev/null || true; \
		echo "==> [3/3] Installing binary → $(LINUX_BINARY)"; \
		install -m 0755 "$$tmp_out/search-linux" "$(LINUX_BINARY)"; \
		if [ -f "$$tmp_out/libstdc++.so.6" ]; then \
			mkdir -p "$(LINUX_RUNTIME_LIB_DIR)"; \
			install -m 0644 "$$tmp_out/libstdc++.so.6" "$(LINUX_RUNTIME_LIBSTDCPP)"; \
			echo "    Extracted libstdc++.so.6 → $(LINUX_RUNTIME_LIBSTDCPP)"; \
		fi
	@echo "==> Built Linux binary: $(LINUX_BINARY) ($(VERSION))"

.PHONY: build-linux-noble
build-linux-noble: ## Build Ubuntu 24.04 (Noble) binary to $$HOME/bin/search-linux-noble (Docker, CGO enabled)
	@mkdir -p $(dir $(LINUX_BINARY_NOBLE))
	@echo "==> Building Linux Noble (Ubuntu 24.04) binary via $(LINUX_CONTAINER)"
	@echo "    image=$(LINUX_NOBLE_IMAGE_TAG)  arch=$(LINUX_GOARCH)  version=$(VERSION)"
	@echo "    (QEMU cross-compile — may take 15-20 min on cold cache)"
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
		echo "==> [1/3] Building container image $$img ..."; \
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
		echo "==> [2/3] Extracting binary from container..."; \
		cid="$$($(LINUX_CONTAINER) create --platform "linux/$(LINUX_GOARCH)" "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux-noble" "$$tmp_out/search-linux-noble"; \
		echo "==> [3/3] Installing binary → $(LINUX_BINARY_NOBLE)"; \
		install -m 0755 "$$tmp_out/search-linux-noble" "$(LINUX_BINARY_NOBLE)"
	@echo "==> Built Noble binary: $(LINUX_BINARY_NOBLE) ($(VERSION))"

# Noble (Ubuntu 24.04) + chdb build vars
LINUX_NOBLE_CHDB_DOCKERFILE ?= Dockerfile.linux-noble-chdb
LINUX_NOBLE_CHDB_IMAGE_TAG  ?= mizu-search-linux-noble-chdb-build:local
LINUX_BINARY_NOBLE_CHDB     ?= $(HOME)/bin/search-linux-noble-chdb
LINUX_BINARY_NOBLE_LIBCHDB  ?= $(HOME)/bin/libchdb.so

.PHONY: build-linux-noble-chdb
build-linux-noble-chdb: ## Build Ubuntu 24.04 binary with chdb support (downloads libchdb.so ~300MB, ~20min under QEMU)
	@mkdir -p $(dir $(LINUX_BINARY_NOBLE_CHDB))
	@tmp_out="$$(mktemp -d)"; \
		img="$(LINUX_NOBLE_CHDB_IMAGE_TAG)"; \
		cid=""; \
		set -e; \
		trap 'test -n "$$cid" && $(LINUX_CONTAINER) rm -f "$$cid" >/dev/null 2>&1 || true; rm -rf "$$tmp_out"' EXIT; \
		command -v "$(LINUX_CONTAINER)" >/dev/null 2>&1 || { echo "$(LINUX_CONTAINER) not found"; exit 127; }; \
		if [ "$(LINUX_CONTAINER)" = "podman" ]; then \
			if ! podman info >/dev/null 2>&1; then podman machine start >/dev/null; fi; \
			for i in $$(seq 1 30); do podman info >/dev/null 2>&1 && break; sleep 1; done; \
		fi; \
		$(LINUX_CONTAINER) build \
			--layers \
			--platform "linux/$(LINUX_GOARCH)" \
			--file "$(LINUX_NOBLE_CHDB_DOCKERFILE)" \
			--ignorefile "$(LINUX_DOCKERFILE).dockerignore" \
			--build-arg GO_VERSION="$(LINUX_GO_VERSION)" \
			--build-arg TARGETARCH="$(LINUX_GOARCH)" \
			--build-arg GO_LDFLAGS="$(GO_LDFLAGS)" \
			--target artifact \
			-t "$$img" \
			"$(LINUX_BUILD_CONTEXT)"; \
		cid="$$($(LINUX_CONTAINER) create --platform "linux/$(LINUX_GOARCH)" "$$img")"; \
		$(LINUX_CONTAINER) cp "$$cid:/search-linux-noble-chdb" "$$tmp_out/search-linux-noble-chdb"; \
		$(LINUX_CONTAINER) cp "$$cid:/libchdb.so" "$$tmp_out/libchdb.so"; \
		install -m 0755 "$$tmp_out/search-linux-noble-chdb" "$(LINUX_BINARY_NOBLE_CHDB)"; \
		install -m 0644 "$$tmp_out/libchdb.so" "$(LINUX_BINARY_NOBLE_LIBCHDB)"
	@echo "Built Noble+chdb binary: $(LINUX_BINARY_NOBLE_CHDB) ($(VERSION))"
	@echo "libchdb.so: $(LINUX_BINARY_NOBLE_LIBCHDB)"

.PHONY: deploy-linux-noble-chdb
deploy-linux-noble-chdb: build-linux-noble-chdb ## Deploy chdb binary + libchdb.so to remote ~/bin/ and install as ~/bin/search-chdb
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin"
	@$(SCP) "$(LINUX_BINARY_NOBLE_CHDB)" $(REMOTE_SSH):.search-chdb-upload.tmp
	@$(SCP) "$(LINUX_BINARY_NOBLE_LIBCHDB)" $(REMOTE_SSH):.libchdb-upload.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-chdb-upload.tmp $$HOME/bin/search-linux-noble-chdb; \
		install -m 0644 $$HOME/.libchdb-upload.tmp $$HOME/bin/libchdb.so; \
		rm -f $$HOME/.search-chdb-upload.tmp $$HOME/.libchdb-upload.tmp; \
		printf '"'"'#!/usr/bin/env bash\nexec \"$$HOME/bin/search-linux-noble-chdb\" \"\$$@\"\n'"'"' > $$HOME/bin/search-chdb; \
		chmod +x $$HOME/bin/search-chdb"'
	@echo "Deployed chdb: $(REMOTE_SSH):~/bin/search-chdb"

# ── Rust crawler (tools/crawler) ─────────────────────────────────────────────
# Locate cargo: prefer PATH, fall back to rustup default install location.
CARGO             ?= $(or $(shell command -v cargo 2>/dev/null),$(HOME)/.cargo/bin/cargo)
CRAWLER_DIR       := tools/crawler
CRAWLER_BINARY    ?= $(HOME)/bin/crawler
# Persistent Cargo target cache outside the repo — survives `git clean` and
# `make clean`, so subsequent builds only recompile changed crates (~seconds).
CRAWLER_CACHE_DIR ?= $(HOME)/.cache/mizu/crawler-target
# Set CRAWLER_CLEAN=1 to delete the cache dir after install (reclaims ~3 GB;
# the next build will be a full recompile). Default=0 keeps cache for fast rebuilds.
CRAWLER_CLEAN     ?= 0
# On macOS: wreq-engine uses boring2 (BoringSSL) which fails to link on ARM64.
# Disable default features so only the reqwest engine is compiled.
CRAWLER_FEATURES  ?= $(if $(filter Darwin,$(shell uname -s)),--no-default-features,)

.PHONY: build-crawler
build-crawler: ## Build Rust crawler to $$HOME/bin/crawler; set CRAWLER_CLEAN=1 to wipe cache after
	@mkdir -p $(dir $(CRAWLER_BINARY)) $(CRAWLER_CACHE_DIR)
	@echo "Building Rust crawler (cache: $(CRAWLER_CACHE_DIR))..."
	@cd $(CRAWLER_DIR) && CARGO_TARGET_DIR=$(CRAWLER_CACHE_DIR) $(CARGO) build --release $(CRAWLER_FEATURES)
	@install -m 0755 $(CRAWLER_CACHE_DIR)/release/crawler $(CRAWLER_BINARY)
	@echo "Built: $(CRAWLER_BINARY) ($$($(CRAWLER_BINARY) --version 2>/dev/null | head -1 || echo ok))"
	@if [ "$(CRAWLER_CLEAN)" = "1" ]; then \
		echo "Cleaning cache: $(CRAWLER_CACHE_DIR)"; \
		rm -rf $(CRAWLER_CACHE_DIR); \
	fi

.PHONY: build-crawler-server
build-crawler-server: ## Rsync + build Rust crawler on remote server → ~/bin/crawler (SERVER=1 or 2)
	$(MAKE) -C $(CRAWLER_DIR) build-on-server SERVER=$(SERVER) DEPLOY_KEY=$(DEPLOY_KEY)

# build-on-server: build the Go search binary natively on the remote server (no QEMU, fast).
# Strategy: rsync source (+ parent mizu module for the replace ../.. directive) to server,
# then build with -mod=mod so Go downloads transitive deps from GOPROXY on the server.
# No local vendoring needed — vendor/ stays gitignored.
# Installs Go $(LINUX_GO_VERSION) automatically if not present on the server.
.PHONY: build-on-server
build-on-server: ## Rsync source + build natively on remote server (fast, no QEMU). SERVER=1|2
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@echo "==> Ensuring Go $(LINUX_GO_VERSION) on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'set -e; \
		for GOBIN in /usr/local/go/bin/go $$HOME/local/go/bin/go; do \
			if [ -x "$$GOBIN" ] && [ "$$($$GOBIN version 2>/dev/null | awk "{print \$$3}")" = "go$(LINUX_GO_VERSION)" ]; then \
				echo "Already installed: $$($$GOBIN version)"; exit 0; \
			fi; \
		done; \
		echo "Installing Go $(LINUX_GO_VERSION)..."; \
		curl -fsSL "https://dl.google.com/go/go$(LINUX_GO_VERSION).linux-amd64.tar.gz" -o /tmp/go.tgz; \
		if tar -C /usr/local -xzf /tmp/go.tgz 2>/dev/null; then \
			echo "Installed to /usr/local/go"; \
		else \
			mkdir -p $$HOME/local && tar -C $$HOME/local -xzf /tmp/go.tgz; \
			echo "Installed to $$HOME/local/go (no root)"; \
		fi; \
		rm -f /tmp/go.tgz'
	@echo "==> Syncing mizu root (for replace ../.. directive) to $(REMOTE_SSH):~/.search-build/..."
	@$(SSH) $(REMOTE_SSH) "mkdir -p ~/.search-build/blueprints/search"
	@rsync -az --delete -e "ssh -i $(DEPLOY_KEY) -o BatchMode=yes" \
		--include='/go.mod' --include='/go.sum' --include='/*.go' \
		--include='/pkg/' --include='/pkg/**' \
		--exclude='*' \
		../../ $(REMOTE_SSH):~/.search-build/
	@echo "==> Syncing search source to $(REMOTE_SSH):~/.search-build/blueprints/search/..."
	@rsync -az --delete -e "ssh -i $(DEPLOY_KEY) -o BatchMode=yes" \
		--exclude='/.git' --exclude='*.duckdb' --exclude='/result/' \
		--exclude='/vendor/' --exclude='/tools/' --exclude='/app/' \
		--exclude='/spec/' --exclude='/docs/' --exclude='/reports/' \
		--exclude='/feature/' --exclude='/assets/' --exclude='/markdown.test/' \
		./ $(REMOTE_SSH):~/.search-build/blueprints/search/
	@echo "==> Building on $(REMOTE_SSH) (native amd64, -mod=mod)..."
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		export PATH=/usr/local/go/bin:$$HOME/local/go/bin:$$PATH; \
		cd ~/.search-build/blueprints/search; \
		GOWORK=off GOPROXY=https://proxy.golang.org,direct CGO_ENABLED=1 \
			go build -mod=mod -trimpath \
			-ldflags \"$(GO_LDFLAGS)\" \
			-o ~/bin/search-linux-noble ./cmd/search; \
		chmod +x ~/bin/search-linux-noble; \
		rm -f ~/bin/search; \
		printf '"'"'#!/usr/bin/env bash\nulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=$${GOMEMLIMIT:-2000000000}\nexport GODEBUG=netdns=go\nexport TORRENT_STORAGE_DEFAULT_FILE_IO=classic\nexec \"$$HOME/bin/search-linux-noble\" \"\$$@\"\n'"'"' > ~/bin/search; \
		chmod +x ~/bin/search; \
		echo \"Done: $$HOME/bin/search-linux-noble\""'
	@echo "Built and installed on $(REMOTE_SSH):~/bin/search"

.PHONY: test-crawler
test-crawler: ## Run Rust crawler unit tests
	@cd $(CRAWLER_DIR) && CARGO_TARGET_DIR=$(CRAWLER_CACHE_DIR) $(CARGO) test $(CRAWLER_FEATURES)

.PHONY: deploy-linux-noble
deploy-linux-noble: build-linux-noble ## Deploy Ubuntu 24.04 binary to remote $$HOME/bin/search (no bundled libstdc++)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin"
	@$(SCP) "$(LINUX_BINARY_NOBLE)" $(REMOTE_SSH):.search-upload.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux-noble; \
		rm -f $$HOME/.search-upload.tmp; \
		rm -f $$HOME/bin/search; \
		printf '"'"'#!/usr/bin/env bash\nulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=$${GOMEMLIMIT:-2000000000}\nexport GODEBUG=netdns=go\nexport TORRENT_STORAGE_DEFAULT_FILE_IO=classic\nexec \"$$HOME/bin/search-linux-noble\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed Noble: $(REMOTE_SSH):~/bin/search"

.PHONY: install-chrome
install-chrome: ## One-time: install Chromium via apt on remote server (requires sudo/root, SERVER=1 or 2)
	@echo "Installing Chromium on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) "bash -lc 'sudo apt-get update -qq && sudo apt-get install -y --no-install-recommends chromium chromium-common fonts-liberation && which chromium && chromium --version'"
	@echo "Chromium installed on $(REMOTE_SSH)"

.PHONY: download-chrome
download-chrome: ## Download chrome-headless-shell to ~/bin/chrome-headless/ + wrapper at ~/bin/chromium (no sudo needed, SERVER=1 or 2)
	@echo "Downloading chrome-headless-shell on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		mkdir -p \$$HOME/bin \$$HOME/bin/chrome-headless \$$HOME/bin/chrome-libs; \
		VER=\$$(curl -fsSL https://googlechromelabs.github.io/chrome-for-testing/LATEST_RELEASE_STABLE); \
		echo \"Chrome version: \$$VER\"; \
		curl -fsSL \"https://storage.googleapis.com/chrome-for-testing-public/\$${VER}/linux64/chrome-headless-shell-linux64.zip\" -o /tmp/chrome-headless.zip; \
		unzip -o /tmp/chrome-headless.zip -d /tmp/chrome-headless-extract; \
		cp -r /tmp/chrome-headless-extract/chrome-headless-shell-linux64/. \$$HOME/bin/chrome-headless/; \
		chmod +x \$$HOME/bin/chrome-headless/chrome-headless-shell; \
		rm -rf /tmp/chrome-headless.zip /tmp/chrome-headless-extract; \
		printf '"'"'#!/usr/bin/env bash\nexec \"$$HOME/bin/chrome-headless/chrome-headless-shell\" \"\$$@\"\n'"'"' > \$$HOME/bin/chromium; \
		chmod +x \$$HOME/bin/chromium; \
		echo \"Downloading NSS/NSPR (pinned focal-security versions compatible with Chrome 146+)...\"; \
		rm -rf /tmp/chrome-libs-extract && mkdir -p /tmp/chrome-libs-extract; \
		curl -fsSL http://security.ubuntu.com/ubuntu/pool/main/n/nss/libnss3_3.98-0ubuntu0.20.04.2_amd64.deb -o /tmp/libnss3.deb; \
		curl -fsSL http://archive.ubuntu.com/ubuntu/pool/main/n/nspr/libnspr4_4.35-0ubuntu0.20.04.1_amd64.deb -o /tmp/libnspr4.deb; \
		dpkg-deb -x /tmp/libnss3.deb /tmp/chrome-libs-extract; \
		dpkg-deb -x /tmp/libnspr4.deb /tmp/chrome-libs-extract; \
		echo \"Downloading other required system libs via apt-get download...\"; \
		cd /tmp && apt-get download libatk1.0-0 libatk-bridge2.0-0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libxkbcommon0 libatspi2.0-0 libxrender1 libwayland-server0 2>/dev/null || true; \
		for f in /tmp/*.deb; do [ -f \"\$$f\" ] && dpkg-deb -x \"\$$f\" /tmp/chrome-libs-extract || true; done; \
		rm -rf \$$HOME/bin/chrome-libs && mkdir -p \$$HOME/bin/chrome-libs; \
		find /tmp/chrome-libs-extract -name '*.so*' ! -type l -exec cp {} \$$HOME/bin/chrome-libs/ \\; 2>/dev/null || true; \
		find /tmp/chrome-libs-extract -name '*.so.[0-9]*' -type l -exec cp -d {} \$$HOME/bin/chrome-libs/ \\; 2>/dev/null || true; \
		rm -f /tmp/libnss3.deb /tmp/libnspr4.deb /tmp/*.deb; rm -rf /tmp/chrome-libs-extract; \
		echo \"chrome-libs contents:\"; ls \$$HOME/bin/chrome-libs/ | grep -i nss; \
		LD_LIBRARY_PATH=\$$HOME/bin/chrome-libs \$$HOME/bin/chromium --version 2>&1 || echo \"(version check may fail - binary installed)\""'
	@echo "chrome-headless-shell installed at $(REMOTE_SSH):~/bin/chrome-headless/, wrapper at ~/bin/chromium"

.PHONY: deploy-linux-noble-browser
deploy-linux-noble-browser: build-linux-noble ## Deploy Ubuntu 24.04 binary with Chrome auto-detect wrapper (run download-chrome or install-chrome first)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin"
	@$(SCP) "$(LINUX_BINARY_NOBLE)" $(REMOTE_SSH):.search-upload.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux-noble; \
		rm -f $$HOME/.search-upload.tmp; \
		printf '"'"'#!/usr/bin/env bash\nulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=$${GOMEMLIMIT:-2000000000}\nexport GODEBUG=netdns=go\nexport TORRENT_STORAGE_DEFAULT_FILE_IO=classic\nif [ -d $$HOME/bin/chrome-libs ]; then export LD_LIBRARY_PATH=\"$$HOME/bin/chrome-libs\"; fi\nif [ -f /usr/bin/chromium ]; then export CHROME_BIN=/usr/bin/chromium; elif [ -f $$HOME/bin/chromium ]; then export CHROME_BIN=$$HOME/bin/chromium; fi\nexec \"$$HOME/bin/search-linux-noble\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed Noble+Chrome: $(REMOTE_SSH):~/bin/search"
	@echo "NOTE: Run 'make download-chrome SERVER=$(SERVER)' (no sudo) or 'make install-chrome SERVER=$(SERVER)' (needs sudo) to install Chrome first"

.PHONY: deploy-linux
deploy-linux: deploy-linux-noble ## Deploy binary to remote $$HOME/bin/search (both servers are Ubuntu 24.04 Noble)

.PHONY: deploy-linux-focal
deploy-linux-focal: build-linux ## [LEGACY] Deploy Ubuntu 20.04 focal binary with bundled libstdc++ (no longer used)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@test -f "$(LINUX_RUNTIME_LIBSTDCPP)" || { echo "Missing runtime lib: $(LINUX_RUNTIME_LIBSTDCPP)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin \$$HOME/bin/search-libs"
	@$(SCP) "$(LINUX_BINARY)" $(REMOTE_SSH):.search-upload.tmp
	@$(SCP) "$(LINUX_RUNTIME_LIBSTDCPP)" $(REMOTE_SSH):.libstdc++.so.6.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux; \
		install -m 0644 $$HOME/.libstdc++.so.6.tmp $$HOME/bin/search-libs/libstdc++.so.6; \
		rm -f $$HOME/.search-upload.tmp $$HOME/.libstdc++.so.6.tmp; \
		printf '"'"'#!/usr/bin/env bash\nulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=$${GOMEMLIMIT:-2000000000}\nexport GODEBUG=netdns=go\nexport LD_LIBRARY_PATH=\"$$HOME/bin/search-libs\$${LD_LIBRARY_PATH:+:$$LD_LIBRARY_PATH}\"\nexec \"$$HOME/bin/search-linux\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed (focal): $(REMOTE_SSH):~/bin/search"

.PHONY: deploy-linux-browser
deploy-linux-browser: deploy-linux-noble-browser ## Deploy binary with Chrome wrapper (both servers are Ubuntu 24.04 Noble)

DASHBOARD_PORT ?= 3457

.PHONY: deploy-dashboard
deploy-dashboard: deploy-linux-noble-browser ## Build, deploy, and (re)start FTS dashboard in a screen session (SERVER=2 for server2)
	@echo "==> Stopping existing dashboard on $(REMOTE_SSH)..."
	-@$(SSH) $(REMOTE_SSH) 'screen -X -S dashboard quit 2>/dev/null; true'
	-@$(SSH) $(REMOTE_SSH) 'pkill -f "cc fts dashboard" 2>/dev/null; true'
	@sleep 1
	@echo "==> Starting dashboard on $(REMOTE_SSH):$(DASHBOARD_PORT) in screen session 'dashboard'..."
	@$(SSH) $(REMOTE_SSH) 'screen -dmS dashboard bash -lc "exec $$HOME/bin/search cc fts dashboard --port $(DASHBOARD_PORT) >> /tmp/dashboard.log 2>&1"'
	@sleep 3
	@echo "==> Verifying dashboard is responding..."
	@$(SSH) $(REMOTE_SSH) 'curl -sf http://localhost:$(DASHBOARD_PORT)/api/scrape/list > /dev/null && echo "Dashboard running on port $(DASHBOARD_PORT)" || { echo "ERROR: dashboard not responding"; tail -20 /tmp/dashboard.log; exit 1; }'

.PHONY: deploy-cc-watch
deploy-cc-watch: ## Deploy cc_watch.sh and (re)start g_watch screen session (SERVER=2 for server2; HF_TOKEN=hf_... to write ~/.hf_token)
	@echo "==> Deploying cc_watch.sh to $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'mkdir -p ~/scripts'
	@$(SCP) scripts/cc_watch.sh $(REMOTE_SSH):~/scripts/cc_watch.sh
	@$(SSH) $(REMOTE_SSH) 'chmod +x ~/scripts/cc_watch.sh'
	@if [ -n "$(HF_TOKEN)" ]; then \
		echo "==> Writing HF_TOKEN to $(REMOTE_SSH):~/.hf_token..."; \
		$(SSH) $(REMOTE_SSH) "echo '$(HF_TOKEN)' > ~/.hf_token && chmod 600 ~/.hf_token"; \
	fi
	@echo "==> Stopping existing g_watch session on $(REMOTE_SSH)..."
	-@$(SSH) $(REMOTE_SSH) 'screen -S g_watch -X quit 2>/dev/null; true'
	-@$(SSH) $(REMOTE_SSH) 'pkill -f "cc publish --watch" 2>/dev/null; true'
	@sleep 1
	@echo "==> Starting g_watch on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'screen -dmS g_watch bash -c "bash ~/scripts/cc_watch.sh; exec bash"'
	@sleep 2
	@echo "==> Verifying g_watch is running on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'screen -ls | grep g_watch && echo "g_watch started" || { echo "ERROR: g_watch not found"; exit 1; }'

.PHONY: remote-cc-watch-attach
remote-cc-watch-attach: ## Attach to g_watch screen session on remote server
	@$(SSH) -t $(REMOTE_SSH) 'screen -r g_watch'

.PHONY: remote-cc-watch-kill
remote-cc-watch-kill: ## Kill g_watch screen session on remote server
	@$(SSH) $(REMOTE_SSH) 'screen -S g_watch -X quit 2>/dev/null && echo "Killed" || echo "No session found"'
	-@$(SSH) $(REMOTE_SSH) 'pkill -f "cc publish --watch" 2>/dev/null; true'

.PHONY: remote-cc-watch-log
remote-cc-watch-log: ## Tail cc_watch log on remote server
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/log/cc_watch.log'

# CC_SCHED_START / CC_SCHED_END define the file index range for the scheduler.
# Defaults: server1=0–4999, server2=5000–9999.
CC_SCHED_START ?= $(if $(filter 2,$(SERVER)),5000,0)
CC_SCHED_END   ?= $(if $(filter 2,$(SERVER)),9999,4999)
# HF rate limit: 128 commits/hour per token across ALL repos (cc, arctic, goodreads, etc.).
# Default --commit-interval 180 → 20/hour per server, 40/hour total, 88 headroom for other repos.
# Adjust via COMMIT_INTERVAL env var in cc_watch.sh or --commit-interval flag.

.PHONY: deploy-cc-sched
deploy-cc-sched: ## (Re)start g_sched using Go --schedule on remote server (SERVER=2 for server2)
	@echo "==> Stopping existing g_sched on $(REMOTE_SSH)..."
	-@$(SSH) $(REMOTE_SSH) 'screen -S g_sched -X quit 2>/dev/null; pkill -f "cc publish --schedule" 2>/dev/null; true'
	@sleep 1
	@echo "==> Starting g_sched on $(REMOTE_SSH) (range $(CC_SCHED_START)–$(CC_SCHED_END))..."
	@$(SSH) $(REMOTE_SSH) 'screen -dmS g_sched bash -c "export PATH=$$HOME/bin:$$PATH; search cc publish --schedule --start $(CC_SCHED_START) --end $(CC_SCHED_END); exec bash"'
	@sleep 2
	@echo "==> Verifying g_sched is running on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'screen -ls | grep g_sched && echo "g_sched started" || { echo "ERROR: g_sched not found"; exit 1; }'

.PHONY: remote-cc-sched-log
remote-cc-sched-log: ## Tail cc_schedule log on remote server
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/log/cc_schedule_$(CC_SCHED_START)_$(CC_SCHED_END).log'

.PHONY: remote-cc-sched-attach
remote-cc-sched-attach: ## Attach to g_sched screen session on remote server
	@$(SSH) -t $(REMOTE_SSH) 'screen -r g_sched'

.PHONY: deploy-linux-focal-browser
deploy-linux-focal-browser: build-linux ## [LEGACY] Deploy Ubuntu 20.04 focal binary with Chrome wrapper (bundled libstdc++)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@test -f "$(LINUX_RUNTIME_LIBSTDCPP)" || { echo "Missing runtime lib: $(LINUX_RUNTIME_LIBSTDCPP)"; exit 1; }
	@$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin \$$HOME/bin/search-libs"
	@$(SCP) "$(LINUX_BINARY)" $(REMOTE_SSH):.search-upload.tmp
	@$(SCP) "$(LINUX_RUNTIME_LIBSTDCPP)" $(REMOTE_SSH):.libstdc++.so.6.tmp
	@$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
		install -m 0755 $$HOME/.search-upload.tmp $$HOME/bin/search-linux; \
		install -m 0644 $$HOME/.libstdc++.so.6.tmp $$HOME/bin/search-libs/libstdc++.so.6; \
		rm -f $$HOME/.search-upload.tmp $$HOME/.libstdc++.so.6.tmp; \
		printf '"'"'#!/usr/bin/env bash\nulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true\nexport GOMEMLIMIT=$${GOMEMLIMIT:-2000000000}\nexport GODEBUG=netdns=go\nexport LD_LIBRARY_PATH=\"$$HOME/bin/search-libs\$${LD_LIBRARY_PATH:+:$$LD_LIBRARY_PATH}\"\nif [ -d $$HOME/bin/chrome-libs ]; then export LD_LIBRARY_PATH=\"$$HOME/bin/chrome-libs:\$${LD_LIBRARY_PATH}\"; fi\nif [ -f /usr/bin/chromium ]; then export CHROME_BIN=/usr/bin/chromium; elif [ -f $$HOME/bin/chromium ]; then export CHROME_BIN=$$HOME/bin/chromium; fi\nexec \"$$HOME/bin/search-linux\" \"\$$@\"\n'"'"' > $$HOME/bin/search; \
		chmod +x $$HOME/bin/search"'
	@echo "Deployed Focal+Chrome: $(REMOTE_SSH):~/bin/search"
	@echo "NOTE: Run 'make download-chrome SERVER=$(SERVER)' to install Chrome first"

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

CC_RECRAWL_FLAGS ?= --domain-fail-threshold 0 --domain-dead-probe 10 --domain-timeout -1 --domain-stall-ratio 20 --batch-size 5000 --workers 4096 --timeout 2000 --max-conns-per-domain 2

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
remote-cc-recrawl-sequential: ## Run p:0→p:100→p:200→p:299 sequentially in a named screen session (survives SSH disconnect)
	@$(SSH) $(REMOTE_SSH) 'screen -dmS cc-sequential bash -c " \
		~/bin/search cc recrawl --file p:0   $(CC_RECRAWL_FLAGS) >~/cc-p0.log   2>&1; \
		~/bin/search cc recrawl --file p:100 $(CC_RECRAWL_FLAGS) >~/cc-p100.log 2>&1; \
		~/bin/search cc recrawl --file p:200 $(CC_RECRAWL_FLAGS) >~/cc-p200.log 2>&1; \
		~/bin/search cc recrawl --file p:299 $(CC_RECRAWL_FLAGS) >~/cc-p299.log 2>&1; \
		echo ALL_DONE; exec bash \
	"; echo "Screen session cc-sequential started (attach: screen -r cc-sequential)"'

.PHONY: remote-cc-screen-attach
remote-cc-screen-attach: ## Attach to cc-sequential screen session on remote server
	@$(SSH) -t $(REMOTE_SSH) 'screen -r cc-sequential'

.PHONY: remote-cc-screen-kill
remote-cc-screen-kill: ## Kill the cc-sequential screen session on remote server
	@$(SSH) $(REMOTE_SSH) 'screen -S cc-sequential -X quit 2>/dev/null && echo "Killed" || echo "No session found"'

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

# ---------------------------------------------------------------------------
# Arctic publish (with auto-restart on stall / crash)
# ---------------------------------------------------------------------------
ARCTIC_FROM ?= 2005-12
ARCTIC_STALL ?= 90m

.PHONY: deploy-arctic
deploy-arctic: deploy-linux-noble ## Build, deploy, and (re)start arctic_run with auto-restart loop (SERVER=2 for server2)
	@echo "==> Stopping existing arctic_run on $(REMOTE_SSH)..."
	-@$(SSH) $(REMOTE_SSH) 'screen -S arctic_run -X quit 2>/dev/null; pkill -f "arctic publish" 2>/dev/null; true'
	@sleep 2
	@echo "==> Starting arctic_run with restart loop on $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'screen -dmS arctic_run bash -c "\
		export PATH=$$HOME/bin:\$$PATH; \
		export TORRENT_STORAGE_DEFAULT_FILE_IO=classic; \
		export HF_XET_FIXED_UPLOAD_CONCURRENCY=4; \
		n=0; \
		while true; do \
			n=\$$((n+1)); \
			LOG=$$HOME/log/arctic_run_\$${n}.log; \
			echo \"[\$$(date)] arctic_run attempt \$$n starting\" >> $$HOME/log/arctic_restart.log; \
			GOMEMLIMIT=6GiB HF_TOKEN=\$$(cat $$HOME/.hf_token) \
				search arctic publish --from $(ARCTIC_FROM) --max-commit-stall $(ARCTIC_STALL) 2>&1 | tee \$$LOG; \
			rc=\$$?; \
			echo \"[\$$(date)] arctic_run exited \$$rc (attempt \$$n)\" >> $$HOME/log/arctic_restart.log; \
			if [ \$$rc -eq 0 ]; then echo \"[\$$(date)] arctic done — exiting loop\" >> $$HOME/log/arctic_restart.log; break; fi; \
			echo \"[\$$(date)] restarting in 30s...\" >> $$HOME/log/arctic_restart.log; \
			sleep 30; \
		done; \
		exec bash"'
	@sleep 2
	@$(SSH) $(REMOTE_SSH) 'screen -ls | grep arctic_run && echo "arctic_run started (restart loop)" || { echo "ERROR: arctic_run not found"; exit 1; }'

.PHONY: deploy-arctic-scripts
deploy-arctic-scripts: ## Deploy arctic scripts (backward, gaps, watchdog) to remote server
	@echo "==> Deploying arctic scripts to $(REMOTE_SSH)..."
	@$(SSH) $(REMOTE_SSH) 'mkdir -p $$HOME/scripts $$HOME/log'
	@$(SCP) scripts/arctic_backward.sh scripts/arctic_gaps.sh scripts/watchdog.sh scripts/hf_check.sh $(REMOTE_SSH):scripts/
	@$(SSH) $(REMOTE_SSH) 'chmod +x $$HOME/scripts/*.sh'
	@echo "==> Installing cron jobs..."
	@$(SSH) $(REMOTE_SSH) 'crontab -l 2>/dev/null | grep -v arctic_gaps | grep -v watchdog | grep -v hf_check > /tmp/cron_clean; \
		echo "*/5 * * * * bash $$HOME/scripts/watchdog.sh" >> /tmp/cron_clean; \
		echo "*/30 * * * * bash $$HOME/scripts/hf_check.sh" >> /tmp/cron_clean; \
		echo "*/30 * * * * bash $$HOME/scripts/arctic_gaps.sh" >> /tmp/cron_clean; \
		crontab /tmp/cron_clean; rm /tmp/cron_clean'
	@echo "==> Cron jobs installed:"
	@$(SSH) $(REMOTE_SSH) 'crontab -l | grep -E "(watchdog|hf_check|arctic_gaps)"'

.PHONY: remote-arctic-attach
remote-arctic-attach: ## Attach to arctic_run screen session on remote server
	@$(SSH) -t $(REMOTE_SSH) 'screen -r arctic_run'

.PHONY: remote-arctic-kill
remote-arctic-kill: ## Kill arctic_run screen session on remote server
	@$(SSH) $(REMOTE_SSH) 'screen -S arctic_run -X quit 2>/dev/null && echo "Killed" || echo "No session found"'
	-@$(SSH) $(REMOTE_SSH) 'pkill -f "arctic publish" 2>/dev/null; true'

.PHONY: remote-arctic-tail
remote-arctic-tail: ## Tail the most recent arctic log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f $$(ls -t $$HOME/log/arctic_run_*.log 2>/dev/null | head -1)'

.PHONY: remote-arctic-status
remote-arctic-status: ## Show arctic process status, restart history, and gaps
	@$(SSH) $(REMOTE_SSH) 'echo "=== Process ==="; ps aux | grep "arctic publish" | grep -v grep; \
		echo ""; echo "=== Restart log ==="; tail -20 $$HOME/log/arctic_restart.log 2>/dev/null || echo "(no restart log)"; \
		echo ""; echo "=== Gaps status ==="; cat $$HOME/log/arctic_gaps_status.json 2>/dev/null || echo "(no gaps status)"; \
		echo ""; echo "=== Recent failures ==="; tail -10 $$HOME/data/arctic/repo/failures.csv 2>/dev/null || echo "(no failures)"'

.PHONY: remote-arctic-gaps
remote-arctic-gaps: ## Run gap detection on remote server
	@$(SSH) $(REMOTE_SSH) 'bash $$HOME/scripts/arctic_gaps.sh && cat $$HOME/log/arctic_gaps_status.json'

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
	@CGO_ENABLED=$(CGO_ENABLED) CGO_LDFLAGS="$(CGO_LDFLAGS)" GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) CGO_LDFLAGS="$(CGO_LDFLAGS)" GOWORK=off $(GO) test -v $(PKG)

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

GUI_PORT ?= 9111
DASHBOARD_PORT ?= 3457

.PHONY: tunnel
tunnel: ## SSH tunnel for crawler GUI + FTS dashboard (localhost:9111 + localhost:3457)
	@echo "Tunneling localhost:$(GUI_PORT) + localhost:$(DASHBOARD_PORT) -> $(REMOTE_SSH)"
	@echo "Open http://localhost:$(GUI_PORT) (crawler) or http://localhost:$(DASHBOARD_PORT) (dashboard)"
	$(SSH) -N -L $(GUI_PORT):localhost:$(GUI_PORT) -L $(DASHBOARD_PORT):localhost:$(DASHBOARD_PORT) $(REMOTE_SSH)

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Search - Full-Featured Search Engine"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
