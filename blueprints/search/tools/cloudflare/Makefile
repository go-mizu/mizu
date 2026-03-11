.PHONY: install test lint

install:
	uv sync

test:
	uv run pytest tests/ -v

browser:
	uv run patchright install chromium
