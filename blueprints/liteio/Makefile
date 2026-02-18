.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/liteio
CMD     ?= ./cmd/liteio

CGO_ENABLED ?= 0

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

.PHONY: build
build: ## Build liteio binary to $$HOME/bin/liteio
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install liteio binary (alias for build)

.PHONY: run
run: ## Run the server with default settings
	@GOWORK=off $(GO) run $(CMD)

.PHONY: run-memory
run-memory: ## Run the server with in-memory driver (ephemeral)
	@GOWORK=off $(GO) run $(CMD) --driver "memory://"

.PHONY: test
test: ## Run all Go tests
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -v $(PKG)

.PHONY: bench
bench: ## Run benchmarks
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -bench=. -benchmem $(PKG)

.PHONY: clean
clean: ## Remove binary
	@rm -f $(BINARY)
	@echo "Cleaned: $(BINARY)"

.PHONY: tidy
tidy: ## Run go mod tidy
	@GOWORK=off $(GO) mod tidy

.PHONY: update
update: ## Update dependencies
	@rm -f go.sum
	@GOWORK=off $(GO) get -u ./...
	@GOWORK=off $(GO) mod tidy

# Docker targets
IMAGE ?= liteio:latest

.PHONY: docker
docker: ## Build Docker image
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(TIME) \
		-t $(IMAGE) .
	@echo "Built: $(IMAGE)"

.PHONY: docker-run
docker-run: ## Run LiteIO in Docker container
	@docker run -d --name liteio \
		-p 9000:9000 \
		-v $(HOME)/data/liteio:/data \
		$(IMAGE)
	@echo "LiteIO running at http://localhost:9000"

.PHONY: docker-stop
docker-stop: ## Stop LiteIO Docker container
	@docker stop liteio 2>/dev/null || true
	@docker rm liteio 2>/dev/null || true
	@echo "LiteIO container stopped"

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "LiteIO — Lightweight S3-Compatible Object Storage"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
