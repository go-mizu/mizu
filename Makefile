# Makefile for github.com/go-mizu/mizu.v0
# Essentials only: test, cover, lint, tidy.

.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
GOFLAGS ?= -trimpath -mod=readonly

COVERMODE ?= atomic
COVERFILE ?= coverage.out
COVERHTML ?= coverage.html

RUN         ?=
COUNT       ?= 1
GOTESTFLAGS ?=

.PHONY: test
test: ## Run tests with race + coverage
	@$(GO) test $(GOFLAGS) -v -race -shuffle=on \
		-count=$(COUNT) $(if $(RUN),-run $(RUN),) $(GOTESTFLAGS) \
		-covermode=$(COVERMODE) -coverprofile="$(COVERFILE)" \
		$(PKG)

.PHONY: cover
cover: test ## Generate HTML coverage report
	@$(GO) tool cover -html="$(COVERFILE)" -o "$(COVERHTML)"
	@echo "Coverage HTML: $(COVERHTML)"

.PHONY: tidy
tidy: ## go mod tidy + verify
	@$(GO) mod tidy
	@$(GO) mod verify

.PHONY: lint
lint: ## golangci-lint (fallback to go vet)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run $(PKG) --timeout=5m; \
	else \
		$(GO) vet $(PKG); \
	fi

.PHONY: clean
clean: ## Remove coverage artifacts
	@rm -f "$(COVERFILE)" "$(COVERHTML)"

.PHONY: help
help: ## Show targets
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
