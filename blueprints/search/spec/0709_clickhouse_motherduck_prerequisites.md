# ClickHouse + MotherDuck CLI Prerequisites

## Overview

`search clickhouse` and `search motherduck` are two-layer systems:

1. **Go CLI layer** — ships in the `search` binary (built via `make deploy-linux-noble`)
2. **Python binary layer** — `~/bin/clickhouse-tool` and `~/bin/motherduck-tool` (built per-server from tools/)

The Go layer handles state management and SQL queries. The Python layer handles browser automation for account registration only.

---

## Layer 1: search binary

The `search` binary already contains the Go CLI commands. Deploy normally:

```bash
# From blueprints/search/ on macOS:
make deploy-linux-noble SERVER=2   # server2 (root@server2)
make deploy-linux-noble SERVER=1   # server1 (tam@server)
```

**Verify:**
```bash
~/bin/search clickhouse --help
~/bin/search motherduck --help
```

---

## Layer 2: Python tool binaries

These must be built **on each server** (PyInstaller produces platform-specific binaries).

### Prerequisites per server

#### Ubuntu 24.04 (Noble) — both server1 and server2

**1. uv** (Python package manager)
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
source ~/.local/bin/env    # or restart shell
```

**2. Chromium + Chrome** (for browser automation)

The tools use patchright (Playwright fork) with Chrome. Install Google Chrome:
```bash
# Download and install Chrome
cd /tmp
wget -q https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
apt-get install -y ./google-chrome-stable_current_amd64.deb   # as root
# Or for non-root:
dpkg -i google-chrome-stable_current_amd64.deb
apt-get install -f -y
```

Verify:
```bash
google-chrome --version
# Google Chrome 133.x.x.x
```

**3. Xvfb** (virtual display for headless Chrome on Linux)
```bash
apt-get install -y xvfb    # Ubuntu/Debian
# The tool auto-starts Xvfb if DISPLAY is not set
```

**4. Python build tools** (needed for patchright native extensions)
```bash
apt-get install -y build-essential libssl-dev
```

#### server1 ONLY — broken system Python workaround

server1's system Python 3.12 has a broken `_ctypes` module (undefined symbol `_PyErr_SetLocaleString`). uv's managed Python must be used explicitly:

```bash
# Install uv-managed Python 3.12
~/.local/bin/uv python install 3.12

# When running uv sync/build, specify uv-managed Python:
UVP=~/.local/share/uv/python/cpython-3.12.13-linux-x86_64-gnu/bin/python3.12
uv sync --group dev --python $UVP
uv run pyinstaller --clean --noconfirm <spec-file>
```

The `make install` target in each tool's Makefile uses the default `uv sync` which may pick up broken system Python on server1. **Workaround**: delete `.venv` first and sync with explicit Python:

```bash
rm -rf .venv
~/.local/bin/uv sync --group dev --python ~/.local/share/uv/python/cpython-3.12.13-linux-x86_64-gnu/bin/python3.12
~/.local/bin/uv run pyinstaller --clean --noconfirm <spec>.spec
```

---

### Build + install commands

```bash
# ClickHouse tool
cd ~/src/mizu/blueprints/search/tools/clickhouse
rm -rf .venv   # server1 only: force uv-managed Python
~/.local/bin/uv sync --group dev
~/.local/bin/uv run pyinstaller --clean --noconfirm clickhouse-tool.spec
cp dist/clickhouse-tool ~/bin/clickhouse-tool && chmod +x ~/bin/clickhouse-tool

# MotherDuck tool
cd ~/src/mizu/blueprints/search/tools/motherduck
rm -rf .venv   # server1 only: force uv-managed Python
~/.local/bin/uv sync --group dev
~/.local/bin/uv run pyinstaller --clean --noconfirm motherduck-tool.spec
cp dist/motherduck-tool ~/bin/motherduck-tool && chmod +x ~/bin/motherduck-tool
```

Or use `make install` on server2 (uv-managed Python works out of the box):
```bash
cd ~/src/mizu/blueprints/search/tools/clickhouse && make install
cd ~/src/mizu/blueprints/search/tools/motherduck && make install
```

**Verify:**
```bash
~/bin/clickhouse-tool --help
~/bin/motherduck-tool --help
```

---

## Binary discovery order

The Go CLI finds `clickhouse-tool` / `motherduck-tool` as follows:
1. `$CLICKHOUSE_TOOL` / `$MOTHERDUCK_TOOL` env var
2. `~/bin/clickhouse-tool` / `~/bin/motherduck-tool`
3. `PATH` lookup

---

## Data files

State is stored in per-tool DuckDB files (same path used by both Python and Go layers):

| Tool | State file |
|------|-----------|
| ClickHouse | `~/data/clickhouse/clickhouse.duckdb` |
| MotherDuck | `~/data/motherduck/mother.duckdb` |

The Go CLI creates these files on first run. The Python tools also create them (when not using `--json`). Both layers share the same schema — do not run them concurrently against the same file.

---

## Registration flow (requires browser + Chrome + network)

```bash
# ClickHouse: opens Chrome, creates mail.tm email, signs up, extracts service credentials
search clickhouse register [--no-headless] [--verbose]

# MotherDuck: opens Chrome, creates mail.tm email, signs up, extracts API token
search motherduck register [--no-headless] [--verbose]
```

After registration, query immediately:
```bash
search clickhouse query "SELECT version()"
search motherduck query "SELECT 42 AS answer"
```

---

## Upgrade procedure

When source code changes (browser.py, cli.py, etc.):

```bash
# Sync updated source to server
rsync -avz --exclude='.venv' --exclude='dist' --exclude='build' \
  tools/clickhouse/ server2:~/src/mizu/blueprints/search/tools/clickhouse/
rsync -avz --exclude='.venv' --exclude='dist' --exclude='build' \
  tools/motherduck/ server2:~/src/mizu/blueprints/search/tools/motherduck/

# Rebuild on server
ssh server2 "cd ~/src/mizu/blueprints/search/tools/clickhouse && make install"
ssh server2 "cd ~/src/mizu/blueprints/search/tools/motherduck && make install"
```

The `search` Go binary is upgraded via the normal `make deploy-linux-noble SERVER=2` flow.

---

## Binary size (expected)

| Binary | Platform | Size |
|--------|----------|------|
| `clickhouse-tool` | macOS arm64 | ~75 MB |
| `clickhouse-tool` | Linux amd64 | ~93 MB |
| `motherduck-tool` | macOS arm64 | ~73 MB |
| `motherduck-tool` | Linux amd64 | ~88 MB |

These are single-file executables that self-extract on first run. Subsequent runs use the extracted cache in `/tmp/_MEIXXXXXX/` (auto-cleaned).

---

## Tested configurations

| Server | OS | search binary | clickhouse-tool | motherduck-tool | Tested |
|--------|-----|--------------|-----------------|-----------------|--------|
| local (macOS) | macOS 15 arm64 | make install | make install (clickhouse) | make install (motherduck) | ✓ query, account ls, service ls |
| server2 | Ubuntu 24.04 amd64 | make deploy-linux-noble SERVER=2 | make install | make install | ✓ query, account ls, db ls |
| server1 | Ubuntu 24.04 amd64 | make deploy-linux-noble SERVER=1 | built manually (uv-managed Python) | built manually (uv-managed Python) | ✓ binary works, no pre-existing accounts |
