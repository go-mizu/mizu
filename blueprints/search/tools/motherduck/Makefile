.PHONY: build install clean sync test

BIN_NAME = motherduck-tool
DIST_DIR = dist

sync:
	uv sync --group dev

build: sync
	uv run pyinstaller --clean --noconfirm $(BIN_NAME).spec

install: build
	mkdir -p $(HOME)/bin
	cp $(DIST_DIR)/$(BIN_NAME) $(HOME)/bin/$(BIN_NAME)
	chmod +x $(HOME)/bin/$(BIN_NAME)

clean:
	rm -rf dist/ build/ __pycache__/ *.pyc

test: sync
	uv run pytest tests/ -v
