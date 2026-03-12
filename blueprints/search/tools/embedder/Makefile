CARGO         ?= $(or $(shell command -v cargo 2>/dev/null),$(HOME)/.cargo/bin/cargo)
TARGET_DIR    ?= $(HOME)/.cache/mizu/embedder-target

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

CHUNKS        ?= testdata/chunks.jsonl
BATCH_SIZE    ?= 64

MODEL_DIR     ?= $(HOME)/data/models/all-MiniLM-L6-v2
ONNX_MODEL    ?= $(MODEL_DIR)/onnx/model.onnx
TOKENIZER     ?= $(MODEL_DIR)/tokenizer.json

.PHONY: build build-candle build-candle-metal build-candle-accel build-fastembed build-ort \
        bench bench-candle-cpu bench-candle-metal bench-candle-accel bench-fastembed bench-ort-cpu bench-ort-coreml bench-all \
        gen-chunks build-on-server deploy clean

## ── Build ─────────────────────────────────────────────────────────────

## Build candle embedder (CPU only)
build-candle:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --manifest-path candle/Cargo.toml
	@install -m 0755 $(TARGET_DIR)/release/embedder-candle $(HOME)/bin/embedder-candle
	@echo "Installed: $(HOME)/bin/embedder-candle"

## Build candle embedder with Metal GPU (macOS)
build-candle-metal:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --manifest-path candle/Cargo.toml --features metal
	@install -m 0755 $(TARGET_DIR)/release/embedder-candle $(HOME)/bin/embedder-candle
	@echo "Installed: $(HOME)/bin/embedder-candle (metal)"

## Build candle embedder with Accelerate (macOS AMX)
build-candle-accel:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --manifest-path candle/Cargo.toml --features accelerate
	@install -m 0755 $(TARGET_DIR)/release/embedder-candle $(HOME)/bin/embedder-candle
	@echo "Installed: $(HOME)/bin/embedder-candle (accelerate)"

## Build fastembed embedder (ONNX Runtime, CPU)
build-fastembed:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --manifest-path fastembed/Cargo.toml
	@install -m 0755 $(TARGET_DIR)/release/embedder-fastembed $(HOME)/bin/embedder-fastembed
	@echo "Installed: $(HOME)/bin/embedder-fastembed"

## Build ORT embedder (CPU + CoreML)
build-ort:
	@mkdir -p $(HOME)/bin $(TARGET_DIR)
	CARGO_TARGET_DIR=$(TARGET_DIR) $(CARGO) build --release --manifest-path ort/Cargo.toml
	@install -m 0755 $(TARGET_DIR)/release/embedder-ort $(HOME)/bin/embedder-ort
	@echo "Installed: $(HOME)/bin/embedder-ort"

## Build all
build: build-ort build-candle-accel build-fastembed

## ── Test Corpus ───────────────────────────────────────────────────────

## Generate test chunks from real Go/markdown files
gen-chunks:
	bash gen_chunks.sh

## ── Benchmarks ────────────────────────────────────────────────────────

## Benchmark candle CPU
bench-candle-cpu: build-candle
	$(HOME)/bin/embedder-candle bench --input $(CHUNKS) --batch-size $(BATCH_SIZE)

## Benchmark candle Metal GPU
bench-candle-metal: build-candle-metal
	$(HOME)/bin/embedder-candle bench --input $(CHUNKS) --batch-size $(BATCH_SIZE) --metal

## Benchmark candle Accelerate
bench-candle-accel: build-candle-accel
	$(HOME)/bin/embedder-candle bench --input $(CHUNKS) --batch-size $(BATCH_SIZE)

## Benchmark fastembed (ONNX Runtime CPU)
bench-fastembed: build-fastembed
	$(HOME)/bin/embedder-fastembed bench --input $(CHUNKS) --batch-size $(BATCH_SIZE)

## Benchmark ORT CPU
bench-ort-cpu: build-ort
	$(HOME)/bin/embedder-ort bench --input $(CHUNKS) --batch-size $(BATCH_SIZE) \
		--model $(ONNX_MODEL) --tokenizer $(TOKENIZER)

## Benchmark ORT CoreML
bench-ort-coreml: build-ort
	$(HOME)/bin/embedder-ort bench --input $(CHUNKS) --batch-size $(BATCH_SIZE) \
		--model $(ONNX_MODEL) --tokenizer $(TOKENIZER) --coreml

## Embed real markdown with ORT
embed-ort: build-ort
	$(HOME)/bin/embedder-ort embed \
		--input $(MD_INPUT) --output $(MD_OUTPUT) \
		--model $(ONNX_MODEL) --tokenizer $(TOKENIZER) \
		--batch-size $(BATCH_SIZE)

## Run all benchmarks with multiple batch sizes
bench-all: build
	@echo "=== ORT CPU (winner) ==="
	@for bs in 1 16 32 64 128 256; do \
		echo "--- batch_size=$$bs ---"; \
		$(HOME)/bin/embedder-ort bench --input $(CHUNKS) --batch-size $$bs \
			--model $(ONNX_MODEL) --tokenizer $(TOKENIZER); \
	done
	@echo ""
	@echo "=== Candle Accelerate ==="
	@for bs in 1 16 32 64 128 256; do \
		echo "--- batch_size=$$bs ---"; \
		$(HOME)/bin/embedder-candle bench --input $(CHUNKS) --batch-size $$bs \
			--model-dir $(MODEL_DIR); \
	done
	@echo ""
	@echo "=== Fastembed (ONNX CPU) ==="
	@for bs in 1 16 32 64 128 256; do \
		echo "--- batch_size=$$bs ---"; \
		$(HOME)/bin/embedder-fastembed bench --input $(CHUNKS) --batch-size $$bs; \
	done

## ── Deploy ────────────────────────────────────────────────────────────

## Build and deploy on remote server
build-on-server:
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	@echo "Syncing source to $(REMOTE_SSH):~/build/embedder/ ..."
	$(SSH) $(REMOTE_SSH) "mkdir -p ~/build/embedder"
	rsync -av --delete --exclude target/ --exclude '.git/' --exclude testdata/ \
	  -e "ssh -i $(DEPLOY_KEY) -o BatchMode=yes" \
	  ./ $(REMOTE_SSH):~/build/embedder/
	@echo "Building on $(REMOTE_SSH) ..."
	$(SSH) $(REMOTE_SSH) 'bash -lc "set -e; \
	  if ! command -v cargo &>/dev/null; then \
	    curl --proto =https --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable; \
	    source ~/.cargo/env; \
	  fi; \
	  cd ~/build/embedder; \
	  RUSTFLAGS=\"-C target-cpu=x86-64-v3\" cargo build --release --manifest-path candle/Cargo.toml 2>&1; \
	  RUSTFLAGS=\"-C target-cpu=x86-64-v3\" cargo build --release --manifest-path fastembed/Cargo.toml 2>&1; \
	  mkdir -p ~/bin; \
	  cp target/release/embedder-candle ~/bin/embedder-candle; \
	  cp target/release/embedder-fastembed ~/bin/embedder-fastembed; \
	  chmod +x ~/bin/embedder-candle ~/bin/embedder-fastembed; \
	  echo BUILD_OK"'
	@echo "Built and deployed to $(REMOTE_SSH)"

## Deploy test data to server
deploy-chunks:
	@test -f "$(DEPLOY_KEY)" || { echo "Deploy key not found: $(DEPLOY_KEY)"; exit 1; }
	$(SSH) $(REMOTE_SSH) "mkdir -p ~/build/embedder/testdata"
	$(SCP) $(CHUNKS) $(REMOTE_SSH):~/build/embedder/testdata/chunks.jsonl
	@echo "Deployed chunks to $(REMOTE_SSH)"

## Run benchmarks on remote server
remote-bench:
	$(SSH) $(REMOTE_SSH) 'bash -lc "cd ~/build/embedder && \
	  echo \"=== Candle CPU ===\" && \
	  for bs in 1 16 32 64 128 256; do \
	    echo \"--- batch_size=$$bs ---\"; \
	    ~/bin/embedder-candle bench --input testdata/chunks.jsonl --batch-size $$bs; \
	  done && \
	  echo \"\" && \
	  echo \"=== Fastembed (ONNX CPU) ===\" && \
	  for bs in 1 16 32 64 128 256; do \
	    echo \"--- batch_size=$$bs ---\"; \
	    ~/bin/embedder-fastembed bench --input testdata/chunks.jsonl --batch-size $$bs; \
	  done"'

## ── Clean ─────────────────────────────────────────────────────────────

clean:
	rm -rf $(TARGET_DIR)
	rm -f $(HOME)/bin/embedder-candle $(HOME)/bin/embedder-fastembed
