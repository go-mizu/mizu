.PHONY: build run init seed test clean help

BINARY := drive
INSTALL_PATH := $(HOME)/bin/$(BINARY)
DATA_DIR := ./data

build:
	@echo "Building $(BINARY)..."
	@GOWORK=off go build -o $(INSTALL_PATH) ./cmd/drive

run:
	@GOWORK=off go run ./cmd/drive $(ARGS)

serve: build
	@$(INSTALL_PATH) serve --data $(DATA_DIR)

init: build
	@$(INSTALL_PATH) init --data $(DATA_DIR)

seed: build
	@$(INSTALL_PATH) seed --data $(DATA_DIR)

test:
	@GOWORK=off go test -v ./...

clean:
	@rm -f $(INSTALL_PATH)
	@rm -rf $(DATA_DIR)

tidy:
	@GOWORK=off go mod tidy

help:
	@echo "Available targets:"
	@echo "  build    - Build binary to ~/bin/$(BINARY)"
	@echo "  run      - Run with go run (use ARGS='...' to pass args)"
	@echo "  serve    - Build and start server"
	@echo "  init     - Initialize database"
	@echo "  seed     - Seed demo data"
	@echo "  test     - Run tests"
	@echo "  clean    - Remove binary and data"
	@echo "  tidy     - Run go mod tidy"
	@echo "  help     - Show this help"
