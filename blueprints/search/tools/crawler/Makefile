DEPLOY_KEY    ?= $(HOME)/.ssh/id_ed25519_deploy
SERVER1_USER  ?= tam
SERVER1_HOST  ?= server
SERVER2_USER  ?= root
SERVER2_HOST  ?= server2
SERVER        ?= 1
REMOTE_USER   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_USER),$(SERVER1_USER))
REMOTE_HOST   ?= $(if $(filter 2,$(SERVER)),$(SERVER2_HOST),$(SERVER1_HOST))
REMOTE_SSH     = $(REMOTE_USER)@$(REMOTE_HOST)
SSH           ?= ssh -i $(DEPLOY_KEY) -o BatchMode=yes
SCP           ?= scp -i $(DEPLOY_KEY) -o BatchMode=yes

BINARY        ?= crawler-linux-$(SERVER)
IMAGE_NAME    := crawler-rust-builder

GUI_PORT      ?= 9111

# External Cargo target cache — shared with the root Makefile, survives git clean.
CARGO             ?= $(or $(shell command -v cargo 2>/dev/null),$(HOME)/.cargo/bin/cargo)
TARGET_DIR        ?= $(HOME)/.cache/mizu/crawler-target

.PHONY: build build-linux build-on-server deploy deploy-server1 deploy-server2 test remote-test remote-hn-recrawl remote-benchmark remote-benchmark-all remote-cc-recrawl remote-cc-recrawl-tui remote-cc-benchmark tunnel clean

## Local dev build (macOS — no wreq-engine; wreq uses BoringSSL which doesn't link on macOS ARM64)
## Uses external cache ($(TARGET_DIR)); installs to ~/bin/crawler automatically.
build:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --no-default-features
	@install -m 0755 $(TARGET_DIR)/release/crawler $(HOME)/bin/crawler
	@echo "Installed: $$($(HOME)/bin/crawler --version)"

## Cross-compile for Linux via Docker (Ubuntu 24.04, x86-64-v3/AVX2)
build-linux:
	@echo "Building Linux binary via Docker (may take 15-20min first time)..."
	docker build -t $(IMAGE_NAME) -f Dockerfile .
	@docker rm -f crawler-linux-tmp 2>/dev/null || true
	docker create --name crawler-linux-tmp $(IMAGE_NAME)
	docker cp crawler-linux-tmp:/build/target/release/crawler ./$(BINARY)
	docker rm crawler-linux-tmp
	@echo "Linux binary: ./$(BINARY)"

## Build and deploy on a remote server (no Docker, no download back)
## Rsyncs source, builds on server, copies binary to ~/bin/crawler.
build-on-server:
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@echo "Syncing source to $(REMOTE_SSH):~/build/crawler/ ..."
	$(SSH) $(REMOTE_SSH) "mkdir -p ~/build/crawler"
	rsync -av --delete --exclude target/ --exclude '.git/' \
	  -e "ssh -i $(DEPLOY_KEY) -o BatchMode=yes" \
	  ./ $(REMOTE_SSH):~/build/crawler/
	@echo "Building and deploying on $(REMOTE_SSH) ..."
	$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
	  SUDO=\"\"; [ \"\$$(id -u)\" != \"0\" ] && SUDO=\"sudo\"; \
	  NEED_PKGS=\"\"; \
	  command -v gcc &>/dev/null || NEED_PKGS=\"\$$NEED_PKGS build-essential\"; \
	  command -v pkg-config &>/dev/null || NEED_PKGS=\"\$$NEED_PKGS pkg-config\"; \
	  dpkg -s libssl-dev &>/dev/null || NEED_PKGS=\"\$$NEED_PKGS libssl-dev\"; \
	  command -v curl &>/dev/null || NEED_PKGS=\"\$$NEED_PKGS curl\"; \
	  if [ -n \"\$$NEED_PKGS\" ]; then \
	    echo \"Installing:\$$NEED_PKGS\"; \
	    \$$SUDO apt-get update -qq && \$$SUDO apt-get install -y -qq \$$NEED_PKGS; \
	  fi; \
	  GCC_VER=\$$(gcc --version 2>/dev/null | head -1 | awk \"{print \\\$$NF}\" | cut -d. -f1); \
	  if [ \"\$${GCC_VER:-0}\" -lt 10 ] 2>/dev/null; then \
	    echo \"GCC \$$GCC_VER too old (need >=10), trying to install gcc-10...\"; \
	    \$$SUDO apt-get update -qq && \$$SUDO apt-get install -y -qq gcc-10 g++-10 2>/dev/null || true; \
	    if command -v gcc-10 &>/dev/null; then \
	      \$$SUDO update-alternatives --install /usr/bin/gcc gcc \$$(which gcc-10) 10 2>/dev/null || true; \
	      export CC=\$$(which gcc-10); \
	      echo \"Using CC=\$$CC\"; \
	    else \
	      echo \"WARNING: gcc-10 not available, continuing with GCC \$$GCC_VER (may fail for aws-lc-rs)\"; \
	    fi; \
	  fi; \
	  if ! command -v cargo &>/dev/null; then \
	    curl --proto =https --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable; \
	    source ~/.cargo/env; \
	  fi; \
	  cd ~/build/crawler; \
	  RUSTFLAGS=\"-C target-cpu=x86-64-v3\" cargo build --release --features wreq-engine 2>&1; \
	  mkdir -p ~/bin && cp target/release/crawler ~/bin/crawler.new && mv ~/bin/crawler.new ~/bin/crawler && chmod +x ~/bin/crawler; \
	  echo BUILD_OK"'
	@echo "Built and deployed to $(REMOTE_SSH):~/bin/crawler"
	$(SSH) $(REMOTE_SSH) "~/bin/crawler --help | head -3"

## Deploy a local binary to a server (SERVER=1 or SERVER=2)
deploy: $(BINARY)
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	$(SSH) $(REMOTE_SSH) "mkdir -p \$$HOME/bin"
	$(SCP) ./$(BINARY) $(REMOTE_SSH):.crawler-upload.tmp
	$(SSH) $(REMOTE_SSH) "mv .crawler-upload.tmp ~/bin/crawler && chmod +x ~/bin/crawler"
	@echo "Deployed to $(REMOTE_SSH):~/bin/crawler"
	$(SSH) $(REMOTE_SSH) "~/bin/crawler --help | head -5"

deploy-server1:
	$(MAKE) deploy SERVER=1

deploy-server2:
	$(MAKE) deploy SERVER=2

## Run tests
test:
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) test --no-default-features

## Check remote binary works
remote-test:
	$(SSH) $(REMOTE_SSH) "~/bin/crawler --help"

HN_SEED      ?= ~/data/hn/recrawl/hn_pages.duckdb

## HN recrawl on remote server (auto-config)
remote-hn-recrawl:
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED)"

## Quick benchmark: 10K seeds, devnull writer (measures raw fetch speed)
remote-benchmark:
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED) --limit 10000 --writer devnull"

## Benchmark all writer/engine combos on remote
remote-benchmark-all:
	@echo "=== reqwest + devnull (baseline) ==="
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED) --limit 10000 --engine reqwest --writer devnull --no-retry"
	@echo "=== hyper + devnull ==="
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED) --limit 10000 --engine hyper --writer devnull --no-retry"
	@echo "=== reqwest + duckdb ==="
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED) --limit 10000 --engine reqwest --writer duckdb --no-retry"
	@echo "=== reqwest + binary ==="
	$(SSH) $(REMOTE_SSH) "~/bin/crawler hn recrawl --seed $(HN_SEED) --limit 10000 --engine reqwest --writer binary --no-retry"

CC_CRAWL   ?= CC-MAIN-2026-04
CC_FILE    ?= p:0

## CC recrawl on remote server with GUI dashboard (auto-config, pass-2 retry)
## Runs in foreground so you see logs; open tunnel in a separate terminal.
## Access dashboard: make tunnel SERVER=<N> (in another terminal)
remote-cc-recrawl:
	$(SSH) -t $(REMOTE_SSH) "~/bin/crawler cc recrawl \
	  --file $(CC_FILE) \
	  --crawl $(CC_CRAWL) \
	  --gui --gui-port $(GUI_PORT)"

## CC recrawl foreground with TUI (no GUI, for direct SSH sessions without port forwarding)
remote-cc-recrawl-tui:
	$(SSH) -t $(REMOTE_SSH) "~/bin/crawler cc recrawl \
	  --file $(CC_FILE) \
	  --crawl $(CC_CRAWL)"

## CC quick benchmark: 50k seeds, devnull writer, no retry — measures raw fetch speed
remote-cc-benchmark:
	$(SSH) $(REMOTE_SSH) "~/bin/crawler cc recrawl \
	  --file $(CC_FILE) \
	  --crawl $(CC_CRAWL) \
	  --limit 50000 \
	  --writer devnull \
	  --no-retry \
	  --no-tui"

## SSH tunnel for GUI dashboard: forwards localhost:9111 -> server:9111
tunnel:
	@echo "Tunneling localhost:$(GUI_PORT) -> $(REMOTE_SSH):$(GUI_PORT)"
	@echo "Open http://localhost:$(GUI_PORT) in your browser"
	$(SSH) -o ServerAliveInterval=5 -o ServerAliveCountMax=120 -o ExitOnForwardFailure=yes \
	  -N -L $(GUI_PORT):localhost:$(GUI_PORT) $(REMOTE_SSH)

clean:
	cargo clean
	rm -f $(BINARY)
