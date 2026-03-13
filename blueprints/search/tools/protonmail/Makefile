.PHONY: build install clean sync

BIN_NAME = protonmail-tool
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
