# Architecture

## Overview

cloudflare-tool automates Cloudflare account creation using Patchright (browser) and
mail.tm (disposable email), creates API tokens via the CF dashboard, manages Workers via
wrangler + REST API, and stores all credentials locally in DuckDB.

```
┌──────────┐     ┌──────────┐     ┌─────────────────────┐
│  cli.py  │────▸│ store.py │────▸│ cloudflare.duckdb   │
│  (Typer) │     │ (DuckDB) │     │ accounts/tokens/    │
└────┬─────┘     └──────────┘     │ workers/op_log      │
     │                            └─────────────────────┘
     │  register / token create
     ▼
┌────────────┐   ┌────────────┐   ┌──────────────────────┐
│ identity.py│──▸│  email.py  │──▸│ mail.tm API          │
│ (Faker)    │   │  (httpx)   │   │ (disposable email)   │
└────────────┘   └────────────┘   └──────────────────────┘
     │
     ▼
┌────────────┐   ┌──────────────────────────────────────┐
│ browser.py │──▸│ Cloudflare Dashboard                 │
│(Patchright)│   │ (signup, token creation)             │
└────────────┘   └──────────────────────────────────────┘
     │
     │  worker deploy/tail
     ▼
┌────────────┐   ┌──────────────────────────────────────┐
│ workers.py │──▸│ wrangler subprocess (deploy, tail)   │
│            │──▸│ httpx (invoke → workers.dev URL)     │
└────────────┘   └──────────────────────────────────────┘
     │
     │  worker ls/rm / account info
     ▼
┌────────────┐   ┌──────────────────────────────────────┐
│ client.py  │──▸│ Cloudflare REST API                  │
│  (httpx)   │   │ api.cloudflare.com/client/v4         │
└────────────┘   └──────────────────────────────────────┘
```

## Registration Flow

```
dash.cloudflare.com/sign-up
  → fill email + password → submit
  → poll mail.tm → click verification link
  → skip domain setup → accept Free plan
  → extract account_id from URL (/home/<32-hex-id>)
  → store in DuckDB
```

## Token Creation Flow

```
dash.cloudflare.com/login → authenticate
  → /profile/api-tokens → "Create Token" → "Custom Token"
  → fill name → add permissions from preset
  → "Continue to summary" → "Create Token"
  → extract token value from confirmation page (shown once)
  → store in DuckDB
```

## Workers Flow

```
deploy:  CLOUDFLARE_ACCOUNT_ID + CLOUDFLARE_API_TOKEN env vars
         → npx wrangler deploy --name <name> <path>
         → parse URL from stdout → store in DuckDB

tail:    npx wrangler tail <name>   (streaming subprocess)

invoke:  httpx POST/GET → https://<name>.<subdomain>.workers.dev<path>
         → log to op_log

ls/rm:   GET/DELETE api.cloudflare.com/client/v4/accounts/{id}/workers/scripts
```

## Modules

### cli.py (~520 lines)
Typer app with four command groups: `register`, `account {ls,rm}`, `token {create,ls,rm,use}`,
`worker {deploy,ls,rm,tail,invoke,use}`. All commands are thin wrappers delegating to other
modules. Rich tables for list output.

### identity.py
Generates random identity: display name, email_local (≤20 chars), 14-char password using
`secrets` module. Same implementation as motherduck/clickhouse.

### email.py
mail.tm HTTP client (identical to motherduck/clickhouse): `create_mailbox()`, `poll_for_magic_link()`.
Internal `_token` field set after successful `create_mailbox()`.

### browser.py (~650 lines)
Two entry points:
- `register_via_browser()` → drives CF signup, returns `account_id`
- `create_token_via_browser()` → logs in, creates token, returns `token_value`

`PRESETS` dict maps preset name to list of `(resource_type, resource, permission)` tuples.

### store.py
DuckDB-backed local state. Four tables: `accounts`, `tokens`, `workers`, `op_log`.
`set_default_token()` and `set_default_worker()` use transactions to ensure exactly one default.

### client.py
Thin httpx wrapper for CF REST API: `get_account_id()`, `get_subdomain()`,
`list_workers()`, `delete_worker()`.

### workers.py
- `deploy()` — subprocess `npx wrangler deploy`, returns URL
- `tail()` — subprocess `npx wrangler tail`, streaming
- `invoke()` — httpx request to workers.dev URL

## Comparison with motherduck / clickhouse-tool

| Aspect | cloudflare-tool | motherduck | clickhouse-tool |
|--------|----------------|-----------|-----------------|
| Auth | API token (browser extraction) | API token (browser) | Username + password |
| Managed resource | Workers | Databases | Services |
| Cloud API | CF REST + wrangler | DuckDB driver | REST API |
| Deploy | wrangler subprocess | N/A | REST API |
| Tail | wrangler subprocess | N/A | N/A |
| JSON compat | cloudflare.json | N/A | N/A |

## Dependencies

| Package | Purpose |
|---------|---------|
| typer | CLI framework |
| rich | Terminal output (tables, panels, spinners) |
| patchright | Browser automation (Playwright fork) |
| duckdb | Local state storage |
| httpx | HTTP client (CF REST API, worker invoke) |
| faker | Random identity generation |
| wrangler | Worker deploy/tail (via npx, not a Python dep) |
