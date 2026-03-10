# motherduck

Auto-register MotherDuck (cloud DuckDB) accounts and run SQL queries — all from the command line.

## Install

Requires Python 3.11+ and [uv](https://docs.astral.sh/uv/).

```bash
cd blueprints/search/tools/motherduck
uv sync
```

Patchright (Playwright fork) needs a browser installed once:

```bash
uv run patchright install chromium
```

## Quick Start

Register a new account, create a database, and query it:

```bash
# Register (opens browser, creates mail.tm email, verifies, extracts API token)
uv run motherduck register

# Create a database
uv run motherduck db create mydb --default

# Run a query
uv run motherduck query "SELECT 42 AS answer"

# JSON output
uv run motherduck query "SELECT 1 AS n, 'hello' AS msg" --json
```

## Commands

### register

```bash
uv run motherduck register [--no-headless] [--verbose]
```

Creates a new MotherDuck account end-to-end:

1. Generates a random identity (name, email, password) via Faker
2. Creates a disposable mailbox on mail.tm
3. Opens app.motherduck.com in a browser, completes Auth0 signup
4. Verifies email via magic link from mail.tm
5. Completes onboarding (name, region, TOS)
6. Extracts API token from Settings > Tokens page
7. Stores all credentials locally in DuckDB

Options:
- `--no-headless` — show the browser window (useful for debugging)
- `--verbose` / `-v` — print detailed step-by-step logs

### account ls

```bash
uv run motherduck account ls
```

Lists all registered accounts with database counts and status.

### account rm

```bash
uv run motherduck account rm <email>
```

Deactivates an account locally (does not delete it from MotherDuck).

### db create

```bash
uv run motherduck db create <name> [--alias myalias] [--account user@example.com] [--default]
```

Creates a new database on MotherDuck using the stored API token.

### db ls

```bash
uv run motherduck db ls
```

Lists all databases with alias, account, query count, and default indicator.

### db use

```bash
uv run motherduck db use <alias>
```

Sets the default database for queries.

### db rm

```bash
uv run motherduck db rm <alias>
```

Removes a database from local state (does not delete it from MotherDuck).

### query

```bash
uv run motherduck query "<sql>" [--db <alias>] [--json]
```

Runs SQL against a MotherDuck database. Uses the default database unless `--db` is specified. Results are displayed as a Rich table or raw JSON with `--json`.

## Data Storage

All state is stored in a single DuckDB file:

```
~/data/motherduck/mother.duckdb
```

Three tables:
- `accounts` — email, password, API token
- `databases` — name, alias, account link, default flag
- `query_log` — SQL, rows returned, duration, timestamp

## Running Tests

```bash
uv run pytest tests/ -v
```

All tests are unit tests with mocked external dependencies (no network, no browser).

## Troubleshooting

**Registration fails at email verification**: mail.tm may be temporarily down. Retry after a minute.

**Token not extracted**: The tool tries multiple strategies — DOM elements, page text regex, localStorage, and cookies. Run with `--no-headless --verbose` to see what's happening.

**Browser errors on Linux**: Ensure Chromium dependencies are installed. On headless Linux, the tool auto-wraps with Xvfb if `DISPLAY` is not set.
