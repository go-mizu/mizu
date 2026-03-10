# clickhouse-tool

Auto-register ClickHouse Cloud trial accounts and run SQL queries — all from the command line.

## Install

Requires Python 3.11+ and [uv](https://docs.astral.sh/uv/).

```bash
cd blueprints/search/tools/clickhouse
uv sync
```

Patchright (Playwright fork) needs a browser installed once:

```bash
uv run patchright install chromium
```

## Quick Start

Register a new account, provision a service, and query it:

```bash
# Register (opens browser, creates mail.tm email, verifies, provisions service)
uv run clickhouse-tool register

# Run a query against the default service
uv run clickhouse-tool query "SELECT version()"

# JSON output
uv run clickhouse-tool query "SELECT 1 AS n, 'hello' AS msg" --json
```

## Commands

### register

```bash
uv run clickhouse-tool register [--no-headless] [--verbose]
```

Creates a new ClickHouse Cloud trial account end-to-end:

1. Generates a random identity (name, email, password) via Faker
2. Creates a disposable mailbox on mail.tm
3. Opens the ClickHouse Cloud signup page in a browser
4. Fills the signup form, verifies email, completes onboarding
5. Waits for the service to finish provisioning
6. Resets the service password and captures it via the console UI
7. Stores all credentials locally in DuckDB

Options:
- `--no-headless` — show the browser window (useful for debugging)
- `--verbose` / `-v` — print detailed step-by-step logs

### account ls

```bash
uv run clickhouse-tool account ls
```

Lists all registered accounts with service counts and status.

### account rm

```bash
uv run clickhouse-tool account rm <email>
```

Deactivates an account locally (does not delete it from ClickHouse Cloud).

### service ls

```bash
uv run clickhouse-tool service ls
```

Lists all services with host, alias, query count, and default indicator.

### service create

```bash
uv run clickhouse-tool service create <name> [--provider aws] [--region us-east-1] [--alias myalias] [--default]
```

Creates a new service on ClickHouse Cloud via the REST API. Requires an account with API keys.

### service use

```bash
uv run clickhouse-tool service use <alias>
```

Sets the default service for queries.

### service rm

```bash
uv run clickhouse-tool service rm <alias>
```

Removes a service from local state (does not delete it from ClickHouse Cloud).

### query

```bash
uv run clickhouse-tool query "<sql>" [--service <alias>] [--json]
```

Runs SQL against a ClickHouse Cloud service. Uses the default service unless `--service` is specified. Results are displayed as a Rich table or raw JSON with `--json`.

## Data Storage

All state is stored in a single DuckDB file:

```
~/data/clickhouse/clickhouse.duckdb
```

Three tables:
- `accounts` — email, password, org_id, API keys
- `services` — host, port, db_user, db_password, cloud_id, alias
- `query_log` — SQL, rows returned, duration, timestamp

## Running Tests

```bash
uv run pytest tests/ -v
```

All tests are unit tests with mocked external dependencies (no network, no browser).

## Troubleshooting

**Registration fails at email verification**: mail.tm may be temporarily down. Retry after a minute.

**Service takes too long to provision**: The tool waits up to 5 minutes. GCP asia-southeast1 typically provisions in ~2 minutes.

**Password not captured**: The tool resets the service password via the Settings page and reads it by clicking the eye icon in the reset dialog. If this fails, you can manually reset the password in the ClickHouse Cloud console.

**Browser errors on Linux**: Ensure Chromium dependencies are installed. On headless Linux, the tool auto-wraps with Xvfb if `DISPLAY` is not set.
