.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO     ?= go
PKG    ?= ./...
CMD    ?= ./cmd/now
BINARY ?= $(HOME)/bin/now

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

.PHONY: build
build: ## Build now to $(HOME)/bin/now
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install now

.PHONY: run
run: ## Run now
	@GOWORK=off $(GO) run $(CMD)

.PHONY: test
test: ## Run tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: tidy
tidy: ## Run go mod tidy
	@GOWORK=off $(GO) mod tidy

.PHONY: update
update: ## Update dependencies
	@rm -f go.sum
	@GOWORK=off $(GO) get -u ./...
	@GOWORK=off $(GO) mod tidy
	@echo "Updated dependencies"

.PHONY: clean
clean: ## Remove built binary
	@rm -f $(BINARY)
	@echo "Cleaned: $(BINARY)"

# ── Storage CLI ──────────────────────────────────────────────

STORAGE_CMD    := ./cmd/storage
STORAGE_OUT    := dist
STORAGE_LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

.PHONY: storage
storage: ## Build storage CLI for current platform
	@mkdir -p $(STORAGE_OUT)
	@CGO_ENABLED=0 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage $(STORAGE_CMD)
	@echo "Built: $(STORAGE_OUT)/storage ($(VERSION))"

.PHONY: storage-install
storage-install: storage ## Install storage CLI to ~/bin
	@cp $(STORAGE_OUT)/storage $(HOME)/bin/storage
	@echo "Installed: $(HOME)/bin/storage"

.PHONY: storage-all
storage-all: ## Cross-compile storage CLI for all platforms
	@mkdir -p $(STORAGE_OUT)
	@echo "Building storage CLI $(VERSION) for all platforms..."
	@CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage-darwin-amd64 $(STORAGE_CMD)
	@echo "  storage-darwin-amd64"
	@CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage-darwin-arm64 $(STORAGE_CMD)
	@echo "  storage-darwin-arm64"
	@CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage-linux-amd64 $(STORAGE_CMD)
	@echo "  storage-linux-amd64"
	@CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage-linux-arm64 $(STORAGE_CMD)
	@echo "  storage-linux-arm64"
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GOWORK=off $(GO) build -trimpath -ldflags "$(STORAGE_LDFLAGS)" -o $(STORAGE_OUT)/storage-windows-amd64.exe $(STORAGE_CMD)
	@echo "  storage-windows-amd64.exe"
	@echo "Done. Binaries in $(STORAGE_OUT)/"

# ── Help ────────────────────────────────────────────────────

.PHONY: help
help: ## Show commands
	@echo ""
	@echo "now"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
	@echo ""