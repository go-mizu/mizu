# Pack Python Tool as Self-Contained Binary

## Goal

Pack `tools/clickhouse` as a single self-contained executable (`clickhouse-tool`) that can be distributed and run without exposing Python source code or requiring a Python environment.

## Approach: PyInstaller `--onefile`

PyInstaller bundles Python interpreter, all dependencies (including patchright, duckdb, clickhouse-connect), and source bytecode into a single executable. The executable self-extracts to a temp dir on first run and re-uses the cache on subsequent runs.

**Why PyInstaller over alternatives:**
- `nuitka` compiles to C; adds complexity and build time. Not needed here.
- `zipapp` doesn't bundle native extensions (patchright, duckdb use `.so`).
- `shiv` requires Python already installed on target. Not self-contained.
- PyInstaller handles native extensions, is widely supported, and is the industry standard for this use case.

**Source protection:** PyInstaller includes `.pyc` bytecode (not plain `.py`). Source is not human-readable from the binary. Good enough for our purposes (we don't need military-grade obfuscation).

## Binary Output Path

```
tools/clickhouse/dist/clickhouse-tool          # macOS/Linux
```

The binary is built per-platform (no cross-compilation). For server2 deployment, build on server2 or use the noble Docker build.

## Implementation Steps

### 1. Add PyInstaller dependency

In `tools/clickhouse/pyproject.toml`, add to `[dependency-groups] dev`:
```toml
"pyinstaller>=6.0",
```

### 2. Add `--json` flag to Python `register` command

The Go CLI calls `clickhouse-tool register --json` to get structured output on stdout without writing to DuckDB (Go manages the DuckDB side). The flag causes register to:
- Print a JSON object to stdout: `{"email": ..., "password": ..., "org_id": ..., "api_key_id": ..., "api_key_secret": ..., "service_id": ..., "host": ..., "port": ..., "db_password": ...}`
- Skip all `store.*` calls
- Still print human-readable status to stderr (rich console on stderr)

### 3. PyInstaller spec file

Create `tools/clickhouse/clickhouse-tool.spec`:

```python
# -*- mode: python ; coding: utf-8 -*-
from PyInstaller.utils.hooks import collect_all, collect_submodules

datas = []
binaries = []
hiddenimports = []

# clickhouse-connect
tmp_ret = collect_all('clickhouse_connect')
datas += tmp_ret[0]; binaries += tmp_ret[1]; hiddenimports += tmp_ret[2]

# duckdb
tmp_ret = collect_all('duckdb')
datas += tmp_ret[0]; binaries += tmp_ret[1]; hiddenimports += tmp_ret[2]

# patchright/playwright
tmp_ret = collect_all('patchright')
datas += tmp_ret[0]; binaries += tmp_ret[1]; hiddenimports += tmp_ret[2]

# faker locales
tmp_ret = collect_all('faker')
datas += tmp_ret[0]; binaries += tmp_ret[1]; hiddenimports += tmp_ret[2]

hiddenimports += collect_submodules('clickhouse_tool')

a = Analysis(
    ['src/clickhouse_tool/cli.py'],
    pathex=['src'],
    binaries=binaries,
    datas=datas,
    hiddenimports=hiddenimports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=['tkinter', 'matplotlib', 'numpy', 'pandas'],
    noarchive=False,
)
pyz = PYZ(a.pure)
exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.datas,
    [],
    name='clickhouse-tool',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=False,
    console=True,
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
```

**Note:** Patchright uses a separately installed Chromium binary (not bundled in the PyInstaller output). The Chromium installation lives at `~/.cache/ms-playwright` and is found at runtime via patchright's registry. The PyInstaller binary just calls patchright APIs; the browser itself is not bundled.

### 4. Makefile target

Add to `tools/clickhouse/Makefile` (create if not exists):

```makefile
.PHONY: build install clean

DIST_DIR = dist
BIN_NAME = clickhouse-tool

build:
	uv run pyinstaller --clean --noconfirm $(BIN_NAME).spec

install: build
	cp $(DIST_DIR)/$(BIN_NAME) $(HOME)/bin/$(BIN_NAME)
	chmod +x $(HOME)/bin/$(BIN_NAME)

clean:
	rm -rf dist/ build/ __pycache__/
```

### 5. Entry point for PyInstaller

PyInstaller uses the `Analysis` source file as its entry point. `src/clickhouse_tool/cli.py` already ends with `app_entry()`. The spec uses `src/clickhouse_tool/cli.py` directly.

However, PyInstaller requires the entry point to call the main function at the module level (under `if __name__ == '__main__'`). We need to ensure `cli.py` has:

```python
if __name__ == '__main__':
    app_entry()
```

This is already implicit through `app_entry()` being the script target, but we add it explicitly for clarity.

## Build Command

```bash
cd tools/clickhouse
uv sync --group dev
uv run pyinstaller --clean --noconfirm clickhouse-tool.spec
# Binary at: dist/clickhouse-tool
```

## Go CLI Integration

The Go CLI (`cli/clickhouse.go`) calls the binary as a subprocess:

```
clickhouse-tool register --json 2>/dev/null
```

- stdout: JSON result (parsed by Go)
- stderr: human progress output (shown to user live via `os.Stderr`)
- exit code: 0 = success, non-zero = failure

The binary is expected at `~/bin/clickhouse-tool` (same location as `~/bin/search`).

## Binary Size Estimate

- Python interpreter: ~8MB
- duckdb: ~25MB
- patchright (without Chromium): ~15MB
- clickhouse-connect + httpx: ~5MB
- faker: ~8MB
- typer + rich: ~3MB
- **Total**: ~60-70MB

This is acceptable for a CLI tool. The binary is not included in git (add `dist/` to `.gitignore`).
