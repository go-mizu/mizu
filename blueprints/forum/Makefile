.PHONY: help build run test clean init install

help:
	@echo "Forum - Makefile commands"
	@echo ""
	@echo "  make build    - Build and install to $$HOME/bin/forum"
	@echo "  make run      - Run the server"
	@echo "  make test     - Run tests"
	@echo "  make init     - Initialize database"
	@echo "  make install  - Build and install to $$HOME/bin/forum (alias for build)"
	@echo "  make clean    - Clean build artifacts"

build:
	@mkdir -p $(HOME)/bin
	CGO_ENABLED=1 go build -o $(HOME)/bin/forum ./cmd/forum
	@echo "Built and installed to $(HOME)/bin/forum"

run:
	go run ./cmd/forum serve

test:
	go test -v ./...

init:
	go run ./cmd/forum init

install:
	@mkdir -p $(HOME)/bin
	CGO_ENABLED=1 go build -o $(HOME)/bin/forum ./cmd/forum
	@echo "Built and installed to $(HOME)/bin/forum"

clean:
	rm -f forum
	rm -f forum.db
