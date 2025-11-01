# Makefile for github.com/go-mizu/mizu.v0
# Focus on library DX: test, coverage, lint, tidy, bench, release.

.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

# --------------------------
# Settings
# --------------------------
GO          ?= go
PKG         ?= ./...
GOFLAGS     ?= -trimpath -mod=readonly
COVERPKG    ?= ./...
COVERMODE   ?= atomic
COVERFILE   ?= coverage.out
COVERHTML   ?= coverage.html

# Prefer gotestsum if available for nicer output
GOTESTSUM   := $(shell command -v gotestsum 2>/dev/null)

# Optional test controls
RUN        ?=
COUNT      ?= 1
CPU        ?=
# Extra flags can be passed via environment:
#   GOTESTFLAGS="-race -tags=integration"
GOTESTFLAGS ?=

# --------------------------
# Test & Coverage
# --------------------------

.PHONY: test
test: ## Run full test suite with coverage (shuffle on)
ifdef GOTESTSUM
	@$(GOTESTSUM) --format=short-verbose -- \
	  -shuffle=on -covermode=$(COVERMODE) -coverpkg=$(COVERPKG) \
	  -coverprofile=$(COVERFILE) -count=$(COUNT) $(if $(CPU),-cpu $(CPU),) \
	  $(if $(RUN),-run '$(RUN)',) $(GOTESTFLAGS) $(PKG)
else
	@$(GO) test $(GOFLAGS) \
	  -v -shuffle=on -covermode=$(COVERMODE) -coverpkg=$(COVERPKG) \
	  -coverprofile=$(COVERFILE) -count=$(COUNT) $(if $(CPU),-cpu $(CPU),) \
	  $(if $(RUN),-run '$(RUN)',) $(GOTESTFLAGS) $(PKG)
endif
	@$(GO) tool cover -func=$(COVERFILE) | tail -n1

.PHONY: test-short
test-short: ## Fast tests without coverage
	@$(GO) test $(GOFLAGS) -short -shuffle=on -count=$(COUNT) $(if $(RUN),-run '$(RUN)',) $(GOTESTFLAGS) $(PKG)

.PHONY: test-race
test-race: ## Run tests with -race
	@$(GO) test $(GOFLAGS) -race -shuffle=on -count=$(COUNT) $(if $(RUN),-run '$(RUN)',) $(GOTESTFLAGS) $(PKG)

.PHONY: cover
cover: test ## Generate HTML coverage report
	@$(GO) tool cover -html=$(COVERFILE) -o $(COVERHTML)
	@echo "Coverage HTML: $(COVERHTML)"

.PHONY: update-golden
update-golden: ## Re-run tests with -update flag (for golden files)
ifdef GOTESTSUM
	@$(GOTESTSUM) --format=short-verbose -- -shuffle=on -update $(GOTESTFLAGS) $(PKG)
else
	@$(GO) test $(GOFLAGS) -v -shuffle=on -update $(GOTESTFLAGS) $(PKG)
endif

.PHONY: bench
bench: ## Run benchmarks with mem stats
	@$(GO) test $(GOFLAGS) -bench=. -benchmem $(PKG)

# --------------------------
# Maintenance
# --------------------------

.PHONY: fmt
fmt: ## go fmt
	@$(GO) fmt $(PKG)

.PHONY: vet
vet: ## go vet
	@$(GO) vet $(PKG)

.PHONY: lint
lint: ## golangci-lint if available; fallback to vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint running..."; \
		golangci-lint run ./... --timeout=5m; \
	else \
		echo "golangci-lint not found; running go vet"; \
		$(GO) vet $(PKG); \
	fi

.PHONY: vuln
vuln: ## govulncheck (no install; uses go run)
	@GO111MODULE=on $(GO) run golang.org/x/vuln/cmd/govulncheck@latest $(PKG)

.PHONY: tidy
tidy: ## go mod tidy and verify
	@$(GO) mod tidy
	@$(GO) mod verify

.PHONY: clean
clean: ## Remove coverage artifacts
	@rm -f $(COVERFILE) $(COVERHTML)

.PHONY: ci
ci: tidy fmt vet lint test ## Minimal CI pipeline locally

# --------------------------
# Release
# --------------------------

GIT_REMOTE      ?= origin
DEFAULT_BRANCH  ?= main

.PHONY: release
release: ## Tag and release with GoReleaser. Usage: make release VERSION=X.Y.Z
ifndef VERSION
	$(error VERSION not set. Usage: make release VERSION=X.Y.Z)
endif
	@if ! [[ "$(VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9\.-]+)?$$ ]]; then \
		echo "Invalid VERSION '$(VERSION)'. Expected SemVer like 1.2.3"; exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree not clean. Commit or stash changes first."; exit 1; \
	fi
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser not found. Install it first."; exit 1; \
	fi
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "GITHUB_TOKEN not set"; exit 1; \
	fi
	@set -euo pipefail; \
	echo "Preparing mizu v$(VERSION)"; \
	echo "$(VERSION)" > VERSION; \
	git add -A; \
	git commit -m "release: v$(VERSION)" || echo "Nothing to commit"; \
	git tag -f v$(VERSION); \
	git push $(GIT_REMOTE) HEAD:$(DEFAULT_BRANCH); \
	git push -f $(GIT_REMOTE) v$(VERSION); \
	goreleaser release --clean; \
	echo "Release v$(VERSION) complete"

.PHONY: snapshot
snapshot: ## Snapshot (dry-run) release via GoReleaser
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "goreleaser not found. Install it first."; exit 1; \
	fi
	@goreleaser release --snapshot --clean

# --------------------------
# Help
# --------------------------
.PHONY: help
help: ## Show this help
	@echo ""
	@echo "mizu Makefile - common developer tasks"
	@echo "--------------------------------------"
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
