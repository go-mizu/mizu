# spec/0711 — tools/cloudflare: Auto-Register Cloudflare Accounts + Workers Management

## Overview

`tools/cloudflare` is a Python CLI tool (following `tools/motherduck` and
`tools/clickhouse` patterns) that:

1. **Auto-registers** new Cloudflare accounts via browser automation (Patchright + mail.tm)
2. **Creates API tokens** with named permission presets via the CF dashboard
3. **Manages Workers** — deploy (wrangler subprocess), list, remove, tail, invoke
4. **Stores all credentials** locally in DuckDB
5. **Writes `cloudflare.json`** for compatibility with `pkg/scrape/cloudflare.go`

---

## Install

```bash
cd blueprints/search/tools/cloudflare
uv sync
uv run patchright install chromium
```

---

## Quick Start

```bash
# Register a new account (browser opens, email verified automatically)
uv run cloudflare register

# Create a Browser Rendering token (browser opens CF dashboard)
uv run cloudflare token create my-br-token --preset browser-rendering --default

# Write ~/data/cloudflare/cloudflare.json for pkg/scrape
uv run cloudflare token use my-br-token

# Deploy a Worker
uv run cloudflare worker deploy app/worker --name my-search-worker --default

# Invoke a Worker
uv run cloudflare worker invoke my-search-worker --path /search?q=test

# Tail Worker logs
uv run cloudflare worker tail my-search-worker
```

---

## Commands

### `register`

```bash
uv run cloudflare register [--no-headless] [--verbose] [--json]
```

Auto-registers a new Cloudflare account end-to-end:

1. Generates a random identity (name, email, password) via Faker
2. Creates a disposable mailbox on mail.tm
3. Opens `dash.cloudflare.com/sign-up` in a browser
4. Fills email + password, submits
5. Polls mail.tm for verification email, clicks verification link
6. Completes onboarding: skips domain setup, accepts Free plan
7. Extracts `account_id` from dashboard URL or `GET /accounts`
8. Stores credentials in DuckDB

Options:
- `--no-headless` — show browser window (for debugging)
- `--verbose` / `-v` — detailed step-by-step logs
- `--json` — print JSON to stdout instead of storing in DuckDB (for Go CLI integration)

JSON output shape:
```json
{
  "email": "...",
  "password": "...",
  "account_id": "..."
}
```

---

### `account ls`

```bash
uv run cloudflare account ls
```

Lists all registered accounts with token counts, worker counts, active status.

### `account rm <email>`

```bash
uv run cloudflare account rm user@mail.tm
```

Deactivates an account locally (does not delete from Cloudflare).

---

### `token create <name>`

```bash
uv run cloudflare token create <name> \
  [--preset browser-rendering|workers|r2|kv|dns|all] \
  [--account <email>] \
  [--default]
```

Creates a named API token via browser automation (CF dashboard > Profile > API Tokens):

1. Navigates to `dash.cloudflare.com/profile/api-tokens`
2. Logs in if needed (email + password from stored account)
3. Clicks "Create Token" > "Custom Token"
4. Sets token name and permissions from preset
5. Submits and extracts token value from confirmation page (shown once)
6. Stores in DuckDB

**Permission presets:**

| Preset | CF Permission Groups |
|---|---|
| `browser-rendering` | Account > Browser Rendering > Edit |
| `workers` | Account > Workers Scripts > Edit, Account > Workers Routes > Edit |
| `r2` | Account > R2 Storage > Edit |
| `kv` | Account > Workers KV Storage > Edit |
| `dns` | Zone > DNS > Edit |
| `all` | All of the above |

Default preset: `all`

### `token ls`

Lists all tokens with name, preset, account, default indicator, creation date.

### `token rm <name>`

Removes a token from local state (does not revoke on Cloudflare).

### `token use <name>`

Sets the token as default AND writes `~/data/cloudflare/cloudflare.json`:

```json
{
  "account_id": "<account_id from linked account>",
  "api_token":  "<token_value>"
}
```

This makes the credential immediately usable by `search scrape --cloudflare`
(`pkg/scrape/cloudflare.go` reads this file).

---

### `worker deploy <path>`

```bash
uv run cloudflare worker deploy <path> \
  [--name <worker-name>] \
  [--alias <local-alias>] \
  [--token <token-name>] \
  [--default]
```

Deploys a Worker via wrangler subprocess:

```bash
CLOUDFLARE_ACCOUNT_ID=<account_id> \
CLOUDFLARE_API_TOKEN=<token_value> \
npx wrangler deploy --name <name> <path>
```

- `<path>` — path to Worker source or `wrangler.toml` directory
- `--name` — Worker name on Cloudflare (default: dirname of path)
- `--alias` — local alias for CLI reference (default: same as name)
- `--token` — which stored token to use (default: default token)
- `--default` — set as default worker for invoke/tail

After deploy, records worker URL (`<name>.<subdomain>.workers.dev`) in DuckDB.

### `worker ls`

Lists all deployed workers with alias, name, URL, account, default indicator,
last invoked time.

### `worker rm <alias>`

Deletes a Worker from Cloudflare via REST API:

```
DELETE /accounts/{account_id}/workers/scripts/{name}
```

Also removes from local state.

### `worker tail <alias>`

Streams real-time Worker logs via wrangler subprocess:

```bash
CLOUDFLARE_ACCOUNT_ID=<account_id> \
CLOUDFLARE_API_TOKEN=<token_value> \
npx wrangler tail <name>
```

### `worker invoke <alias>`

```bash
uv run cloudflare worker invoke <alias> \
  [--path /] \
  [--method GET|POST|PUT|DELETE] \
  [--body <json-string>] \
  [--header "Key: Value"] \
  [--json]
```

Sends an HTTP request directly to the Worker's public URL:

```
https://<name>.<subdomain>.workers.dev<path>
```

Prints response body (as Rich panel or raw JSON with `--json`).
Logs operation to `op_log` table.

### `worker use <alias>`

Sets default worker for `invoke` and `tail`.

---

## Data Storage

All state stored in a single DuckDB file:

```
~/data/cloudflare/cloudflare.duckdb
```

### Schema

```sql
CREATE TABLE accounts (
    id          VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email       VARCHAR NOT NULL UNIQUE,
    password    VARCHAR NOT NULL,
    account_id  VARCHAR NOT NULL,
    subdomain   VARCHAR DEFAULT '',   -- <subdomain>.workers.dev
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMP DEFAULT now()
);

CREATE TABLE tokens (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    name         VARCHAR NOT NULL,
    token_id     VARCHAR DEFAULT '',  -- CF token ID (for future revocation)
    token_value  VARCHAR NOT NULL,
    preset       VARCHAR NOT NULL DEFAULT 'all',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now()
);

CREATE TABLE workers (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    token_id     VARCHAR REFERENCES tokens(id),
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    url          VARCHAR DEFAULT '',
    is_default   BOOLEAN DEFAULT false,
    deployed_at  TIMESTAMP,
    created_at   TIMESTAMP DEFAULT now()
);

CREATE TABLE op_log (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    worker_id    VARCHAR,
    operation    VARCHAR NOT NULL,   -- deploy, invoke, tail, rm
    detail       VARCHAR DEFAULT '',
    duration_ms  INTEGER,
    ran_at       TIMESTAMP DEFAULT now()
);
```

---

## Module Architecture

```
src/cloudflare_tool/
├── cli.py        Typer app: register / account / token / worker subcommands
├── identity.py   Faker: random name, email local, strong password
├── email.py      mail.tm API client: create mailbox, poll for verification link
├── browser.py    Patchright: CF signup flow + token creation flow
├── store.py      DuckDB CRUD: accounts, tokens, workers, op_log
├── client.py     CF REST API: GET /accounts, workers ls/rm, account info
└── workers.py    Worker ops: wrangler subprocess (deploy/tail) + httpx invoke
```

### browser.py flows

**`register_via_browser(mailbox, mail_client, password, headless, verbose) → account_id`**

```
dash.cloudflare.com/sign-up
  → fill email + password → submit
  → poll mail.tm for verification email → click link
  → skip domain setup (click "Skip" / "Add later")
  → accept Free plan
  → extract account_id from URL (/home/<account_id>) or API
  → return account_id
```

**`create_token_via_browser(account, password, token_name, preset, headless, verbose) → token_value`**

```
dash.cloudflare.com/login
  → fill email + password → login
  → navigate to /profile/api-tokens
  → "Create Token" → "Custom Token"
  → fill token name
  → add permissions from preset map
  → "Continue to summary" → "Create Token"
  → extract token value from confirmation page
  → return token_value
```

### client.py (CF REST API)

Base URL: `https://api.cloudflare.com/client/v4`

```python
class CloudflareClient:
    def get_account_id(self) -> str          # GET /accounts → first result
    def get_subdomain(self, account_id) -> str  # GET /accounts/{id}/workers/subdomain
    def list_workers(self, account_id) -> list  # GET /accounts/{id}/workers/scripts
    def delete_worker(self, account_id, name)   # DELETE /accounts/{id}/workers/scripts/{name}
```

### workers.py

```python
def deploy(account_id, token, name, path) -> str    # wrangler subprocess → worker URL
def tail(account_id, token, name) -> None           # wrangler subprocess, streaming
def invoke(url, method, path, body, headers) -> tuple[int, str]  # httpx request
```

---

## cloudflare.json Compatibility

`token use <name>` writes:

```json
{
  "account_id": "<linked account's account_id>",
  "api_token":  "<token_value>"
}
```

to `~/data/cloudflare/cloudflare.json` — the exact format expected by
`pkg/scrape/cloudflare.go`.

---

## Dependencies

```toml
dependencies = [
    "typer>=0.12",
    "rich>=13.0",
    "patchright>=1.50",
    "duckdb>=1.2,<1.5",
    "httpx>=0.27",
    "faker>=33.0",
]
```

`wrangler` is invoked via `npx wrangler` (Node.js must be installed; no Python dep).

---

## Testing

All tests in `tests/` with mocked external dependencies (no network, no browser, no wrangler):

- `test_store.py` — DuckDB CRUD with in-memory DB
- `test_client.py` — `CloudflareClient` with mocked httpx responses
- `test_email.py` — mail.tm client with mocked httpx
- `test_workers.py` — `deploy()`, `invoke()` with mocked subprocess / httpx

```bash
uv run pytest tests/ -v
```

---

## Troubleshooting

**Registration fails at email verification**: mail.tm may be down. Retry after a minute.

**Token value not extracted**: Cloudflare shows the token value only once on the
confirmation page. Run `--no-headless --verbose` to see what's happening.

**wrangler not found**: Ensure Node.js and npm are installed. The tool uses
`npx wrangler` which downloads wrangler on first use.

**`subdomain` empty after registration**: Run `token use <name>` to trigger a
`GET /accounts/{id}/workers/subdomain` lookup which backfills the subdomain.
