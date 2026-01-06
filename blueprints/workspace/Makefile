.DEFAULT_GOAL := help
SHELL := /usr/bin/env bash

GO      ?= go
PKG     ?= ./...
BINARY  ?= $(HOME)/bin/workspace
CMD     ?= ./cmd/workspace
DATA_DIR ?= $(HOME)/data/blueprint/workspace

CGO_ENABLED ?= 1

VERSION ?= $(shell git describe --tags --dirty --match "v*" 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TIME    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDVARS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(TIME)
GO_LDFLAGS ?= -s -w $(LDVARS)

# Frontend configuration
FRONTEND_DIR := app/frontend

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

.PHONY: frontend-test-watch
frontend-test-watch: ## Run frontend tests in watch mode
	@cd $(FRONTEND_DIR) && pnpm run test:watch

.PHONY: build
build: frontend-build ## Build workspace binary (includes frontend build)
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Built: $(BINARY) ($(VERSION))"

.PHONY: install
install: build ## Install workspace binary to $$HOME/bin/workspace (alias for build)

.PHONY: run
run: ## Run the server
	@GOWORK=off $(GO) run $(CMD) serve --dev

.PHONY: init
init: ## Initialize the database
	@GOWORK=off $(GO) run $(CMD) init

.PHONY: seed
seed: ## Seed the database with sample data
	@GOWORK=off $(GO) run $(CMD) seed

.PHONY: test
test: frontend-test ## Run all tests (Go + frontend)
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-go
test-go: ## Run Go tests only
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test $(PKG)

.PHONY: test-v
test-v: ## Run all tests with verbose output
	@CGO_ENABLED=$(CGO_ENABLED) GOWORK=off $(GO) test -v $(PKG)

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

.PHONY: check-deps
check-deps: ## Check if required dependencies are installed
	@echo "Checking dependencies..."
	@echo -n "  Chrome/Chromium (required for PDF export): "
	@if [ -f "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then \
		echo "\033[32mfound\033[0m (macOS)"; \
	elif command -v google-chrome >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(command -v google-chrome))"; \
	elif command -v chromium >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(command -v chromium))"; \
	elif command -v chromium-browser >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(command -v chromium-browser))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    PDF export requires Chrome or Chromium."; \
		echo "    Install: https://www.google.com/chrome/ or 'brew install chromium'"; \
	fi
	@echo -n "  pnpm: "
	@if command -v pnpm >/dev/null 2>&1; then \
		echo "\033[32mfound\033[0m ($$(pnpm --version))"; \
	else \
		echo "\033[31mnot found\033[0m"; \
		echo "    Install: npm install -g pnpm"; \
	fi
	@echo ""

.PHONY: install-chrome
install-chrome: ## Install Chrome/Chromium (macOS only)
	@if [ "$$(uname)" = "Darwin" ]; then \
		if [ -f "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then \
			echo "Chrome already installed"; \
		else \
			echo "Installing Chromium via Homebrew..."; \
			brew install --cask chromium || echo "Failed. Please install Chrome from https://www.google.com/chrome/"; \
		fi \
	else \
		echo "Please install Chrome/Chromium for your platform:"; \
		echo "  Ubuntu/Debian: sudo apt install chromium-browser"; \
		echo "  Fedora: sudo dnf install chromium"; \
		echo "  Or download from: https://www.google.com/chrome/"; \
	fi

# Desktop application targets
DESKTOP_DIR := app/desktop
CGO_LDFLAGS_DARWIN := -mmacosx-version-min=11.0

.PHONY: desktop-dev
desktop-dev: ## Run desktop app in development mode
	@cd $(DESKTOP_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS_DARWIN)" GOWORK=off wails dev

.PHONY: desktop-build
desktop-build: ## Build desktop app for current platform
	@cd $(DESKTOP_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS_DARWIN)" GOWORK=off wails build -clean

.PHONY: desktop-build-darwin
desktop-build-darwin: ## Build desktop app for macOS (universal binary)
	@cd $(DESKTOP_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS_DARWIN)" GOWORK=off wails build -clean -platform darwin/universal

.PHONY: desktop-build-windows
desktop-build-windows: ## Build desktop app for Windows (amd64)
	@cd $(DESKTOP_DIR) && GOWORK=off wails build -clean -platform windows/amd64

.PHONY: desktop-build-linux
desktop-build-linux: ## Build desktop app for Linux (amd64)
	@cd $(DESKTOP_DIR) && GOWORK=off wails build -clean -platform linux/amd64

.PHONY: desktop-build-all
desktop-build-all: ## Build desktop app for all platforms
	@echo "Building for macOS..."
	@cd $(DESKTOP_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS_DARWIN)" GOWORK=off wails build -clean -platform darwin/universal
	@echo "Building for Windows..."
	@cd $(DESKTOP_DIR) && GOWORK=off wails build -clean -platform windows/amd64
	@echo "Building for Linux..."
	@cd $(DESKTOP_DIR) && GOWORK=off wails build -clean -platform linux/amd64
	@echo "All desktop builds complete!"

.PHONY: desktop-clean
desktop-clean: ## Clean desktop build artifacts
	@rm -rf $(DESKTOP_DIR)/build/bin
	@rm -rf $(DESKTOP_DIR)/build/frontend
	@echo "Desktop build artifacts cleaned"

.PHONY: help
help: ## Show available commands
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
