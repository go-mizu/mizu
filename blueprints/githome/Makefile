.PHONY: build run dev test clean install init seed

# Variables
BINARY_NAME=githome
BUILD_DIR=./build
DATA_DIR=./data

# Build the binary
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/githome

# Run in development mode
dev:
	go run ./cmd/githome serve --dev

# Run the server
run:
	go run ./cmd/githome serve

# Initialize database
init:
	go run ./cmd/githome init

# Seed demo data
seed:
	go run ./cmd/githome seed

# Run tests
test:
	go test -v ./...

# Install to $HOME/bin
install:
	go build -o $(HOME)/bin/$(BINARY_NAME) ./cmd/githome

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Reset data
reset:
	rm -rf $(DATA_DIR)
	go run ./cmd/githome init
	go run ./cmd/githome seed

# Download dependencies
deps:
	go mod tidy
	go mod download
