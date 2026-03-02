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
SCP           ?= scp -i $(DEPLOY_KEY)

BINARY        ?= crawler-linux-$(SERVER)
IMAGE_NAME    := crawler-rust-builder

.PHONY: build build-linux build-on-server deploy deploy-server1 deploy-server2 test remote-test remote-hn-recrawl remote-benchmark clean

## Local dev build (macOS)
build:
	cargo build --release

## Cross-compile for Linux via Docker (Ubuntu 24.04, x86-64-v3/AVX2)
build-linux:
	@echo "Building Linux binary via Docker (may take 15-20min first time)..."
	docker build -t $(IMAGE_NAME) -f Dockerfile .
	@docker rm -f crawler-linux-tmp 2>/dev/null || true
	docker create --name crawler-linux-tmp $(IMAGE_NAME)
	docker cp crawler-linux-tmp:/build/target/release/crawler ./$(BINARY)
	docker rm crawler-linux-tmp
	@echo "Linux binary: ./$(BINARY)"

## Build Linux binary on a remote server (no Docker needed)
## Rsyncs source to ~/build/crawler on the server, installs Rust if needed, builds, downloads binary.
build-on-server:
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@echo "Syncing source to $(REMOTE_SSH):~/build/crawler/ ..."
	$(SSH) $(REMOTE_SSH) "mkdir -p ~/build/crawler"
	rsync -av --delete --exclude target/ --exclude '.git/' \
	  -e "ssh -i $(DEPLOY_KEY) -o BatchMode=yes" \
	  ./ $(REMOTE_SSH):~/build/crawler/
	@echo "Building on $(REMOTE_SSH) ..."
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
	  RUSTFLAGS=\"-C target-cpu=x86-64-v3\" cargo build --release 2>&1; \
	  echo BUILD_OK"'
	@echo "Downloading binary from $(REMOTE_SSH) ..."
	$(SCP) $(REMOTE_SSH):~/build/crawler/target/release/crawler ./$(BINARY)
	@echo "Linux binary: ./$(BINARY)"

## Deploy to a server (SERVER=1 or SERVER=2)
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
	cargo test

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

clean:
	cargo clean
	rm -f $(BINARY)
