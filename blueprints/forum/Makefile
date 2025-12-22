.PHONY: help build run test clean init install

help:
	@echo "Forum - Makefile commands"
	@echo ""
	@echo "  make build    - Build the binary"
	@echo "  make run      - Run the server"
	@echo "  make test     - Run tests"
	@echo "  make init     - Initialize database"
	@echo "  make install  - Install to ~/bin/forum"
	@echo "  make clean    - Clean build artifacts"

build:
	CGO_ENABLED=1 go build -o forum ./cmd/forum

run:
	go run ./cmd/forum serve

test:
	go test -v ./...

init:
	go run ./cmd/forum init

install:
	CGO_ENABLED=1 go build -o ~/bin/forum ./cmd/forum
	@echo "Installed to ~/bin/forum"

clean:
	rm -f forum
	rm -f forum.db
