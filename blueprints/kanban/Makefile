.PHONY: build run test clean dev install

BINARY_NAME=kanban
BUILD_DIR=./build

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kanban

run:
	@go run ./cmd/kanban $(ARGS)

dev:
	@go run ./cmd/kanban serve --dev

test:
	@go test -v ./...

clean:
	@rm -rf $(BUILD_DIR)

install:
	@echo "Installing $(BINARY_NAME) to $$HOME/bin..."
	@GOWORK=off go build -o $$HOME/bin/$(BINARY_NAME) ./cmd/kanban

init:
	@go run ./cmd/kanban init
