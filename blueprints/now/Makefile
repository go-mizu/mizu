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

.PHONY: help
help: ## Show commands
	@echo ""
	@echo "now"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
	@echo ""