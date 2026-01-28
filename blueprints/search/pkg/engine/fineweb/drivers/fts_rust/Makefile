.PHONY: all rust build test bench clean install-deps

RUST_DIR := fts_rust_core
LIB_DIR := lib
INCLUDE_DIR := include

# Detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        RUST_TARGET := aarch64-apple-darwin
    else
        RUST_TARGET := x86_64-apple-darwin
    endif
    LIB_NAME := libfts_rust_core.a
else ifeq ($(UNAME_S),Linux)
    ifeq ($(UNAME_M),aarch64)
        RUST_TARGET := aarch64-unknown-linux-gnu
    else
        RUST_TARGET := x86_64-unknown-linux-gnu
    endif
    LIB_NAME := libfts_rust_core.a
endif

# Build everything
all: rust

# Install Rust dependencies
install-deps:
	@command -v cargo >/dev/null 2>&1 || { \
		echo "Installing Rust..."; \
		curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y; \
		. $$HOME/.cargo/env; \
	}
	@cargo --version
	@rustup target add $(RUST_TARGET) 2>/dev/null || true

# Build Rust library
rust: install-deps
	@echo "Building Rust library for $(RUST_TARGET)..."
	@mkdir -p $(LIB_DIR)
	cd $(RUST_DIR) && cargo build --release --target $(RUST_TARGET)
	cp $(RUST_DIR)/target/$(RUST_TARGET)/release/$(LIB_NAME) $(LIB_DIR)/
	@echo "Library built: $(LIB_DIR)/$(LIB_NAME)"

# Build Rust library with debug symbols
rust-debug: install-deps
	@echo "Building Rust library (debug) for $(RUST_TARGET)..."
	@mkdir -p $(LIB_DIR)
	cd $(RUST_DIR) && cargo build --target $(RUST_TARGET)
	cp $(RUST_DIR)/target/$(RUST_TARGET)/debug/$(LIB_NAME) $(LIB_DIR)/
	@echo "Library built: $(LIB_DIR)/$(LIB_NAME)"

# Build with profiling enabled
rust-profile: install-deps
	@echo "Building Rust library with profiling..."
	@mkdir -p $(LIB_DIR)
	cd $(RUST_DIR) && cargo build --release --target $(RUST_TARGET) --features profiling
	cp $(RUST_DIR)/target/$(RUST_TARGET)/release/$(LIB_NAME) $(LIB_DIR)/
	@echo "Library built with profiling: $(LIB_DIR)/$(LIB_NAME)"

# Build Go driver
build: rust
	@echo "Building Go driver..."
	go build ./...

# Run tests
test: rust
	@echo "Running tests..."
	cd $(RUST_DIR) && cargo test
	go test -v ./...

# Run benchmarks
bench: rust
	@echo "Running benchmarks..."
	cd $(RUST_DIR) && cargo bench
	go test -bench=. -benchmem ./...

# Run Rust benchmarks only
bench-rust:
	cd $(RUST_DIR) && cargo bench

# Run Go benchmarks only
bench-go: rust
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	cd $(RUST_DIR) && cargo clean
	rm -f $(LIB_DIR)/*.a $(LIB_DIR)/*.so $(LIB_DIR)/*.dylib
	go clean ./...

# Generate flamegraph (requires cargo-flamegraph)
flamegraph:
	@command -v flamegraph >/dev/null 2>&1 || cargo install flamegraph
	cd $(RUST_DIR) && cargo flamegraph --bench indexing

# Run comprehensive linting (strict mode)
lint:
	@echo "Running comprehensive Rust lints..."
	cd $(RUST_DIR) && cargo clippy --all-targets --all-features -- -D warnings
	@echo "Checking Rust formatting..."
	cd $(RUST_DIR) && cargo fmt -- --check
	@echo "Running Go lints..."
	go vet ./...
	@echo "All lints passed!"

# Check code (alias for lint)
check: lint

# Format code
fmt:
	cd $(RUST_DIR) && cargo fmt
	go fmt ./...

# Fix lint issues automatically where possible
lint-fix:
	cd $(RUST_DIR) && cargo clippy --fix --allow-dirty --all-targets --all-features
	cd $(RUST_DIR) && cargo fmt
	go fmt ./...

# Show library size
size: rust
	@echo "Library size:"
	@ls -lh $(LIB_DIR)/$(LIB_NAME)
	@echo "\nSymbol count:"
	@nm $(LIB_DIR)/$(LIB_NAME) | wc -l

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build everything (default)"
	@echo "  rust         - Build Rust library (release)"
	@echo "  rust-debug   - Build Rust library (debug)"
	@echo "  rust-profile - Build Rust library with profiling"
	@echo "  build        - Build Go driver"
	@echo "  test         - Run all tests"
	@echo "  bench        - Run all benchmarks"
	@echo "  bench-rust   - Run Rust benchmarks only"
	@echo "  bench-go     - Run Go benchmarks only"
	@echo "  clean        - Clean all build artifacts (including Rust target/)"
	@echo "  flamegraph   - Generate flamegraph"
	@echo "  lint         - Run comprehensive linters (clippy with all features)"
	@echo "  check        - Alias for lint"
	@echo "  lint-fix     - Auto-fix lint issues where possible"
	@echo "  fmt          - Format code"
	@echo "  size         - Show library size"
	@echo "  help         - Show this help"
