# Architecture

## Overview

motherduck automates MotherDuck (cloud DuckDB) account creation using browser automation (Patchright) and disposable email (mail.tm), then provides a CLI for managing databases and running SQL queries via the MotherDuck API token.

```
┌──────────┐     ┌──────────┐     ┌──────────────────┐
│  cli.py  │────▸│ store.py │────▸│ mother.duckdb    │
│  (Typer) │     │ (DuckDB) │     │ accounts/dbs/    │
└────┬─────┘     └──────────┘     │ query_log        │
     │                            └──────────────────┘
     │  register
     ▼
┌────────────┐   ┌────────────┐   ┌──────────────────┐
│ identity.py│──▸│  email.py  │──▸│ mail.tm API      │
│ (Faker)    │   │  (httpx)   │   │ (disposable mail)│
└────────────┘   └────────────┘   └──────────────────┘
     │
     ▼
┌────────────┐   ┌──────────────────────────────────┐
│ browser.py │──▸│ MotherDuck Console               │
│(Patchright)│   │ (Auth0 signup, onboarding,       │
└────────────┘   │  token extraction)               │
     │           └──────────────────────────────────┘
     │  query
     ▼
┌────────────┐   ┌──────────────────────────────────┐
│ client.py  │──▸│ MotherDuck Cloud                 │
│  (DuckDB)  │   │ (md:?motherduck_token=...)       │
└────────────┘   └──────────────────────────────────┘
```

## Modules

### cli.py

Typer app with four command groups: `register`, `account {ls,rm}`, `db {create,ls,use,rm}`, `query`. All commands are thin wrappers that delegate to the other modules. Rich tables for output.

### identity.py

Generates a random identity using Faker: display name, email local part (max 20 chars), and a strong password (14 chars, mixed case + digits + special). Uses the `secrets` module for cryptographic randomness in password generation.

### email.py

HTTP client for the [mail.tm API](https://docs.mail.tm/):

1. `_get_domain()` — fetches available domains, picks the first active one
2. `create_mailbox(local)` — creates `{local}@{domain}` with a derived password
3. `poll_for_magic_link(mailbox)` — polls inbox every 3s (up to 120s), extracts the verification URL

Link extraction uses `_pick_magic_link()` which prioritizes:
1. Auth0 domains (`auth.motherduck.com`, `*.auth0.com`)
2. Links with `verify`, `confirm`, or `email-verification` params
3. Long MotherDuck URLs with query params
4. Fallback: first URL found

### browser.py

The largest module (~600 lines). Drives a Chromium browser through the full MotherDuck signup flow.

#### Registration Flow

`register_via_browser()` is the entry point. The 8-step flow:

```
app.motherduck.com ──▸ redirect to Auth0
    │
    ▼
Auth0 signup page ──▸ fill email + password ──▸ submit
    │
    ▼
email verification? ──▸ poll mail.tm ──▸ click magic link
    │
    ▼
clear session ──▸ logout ──▸ re-login with credentials
    │
    ▼
onboarding ──▸ fill name ──▸ select region ──▸ accept TOS
    │
    ▼
Settings > Tokens ──▸ generate token ──▸ extract from page
    │
    ▼
return token string
```

#### Key Implementation Details

**Auth0 Login**

MotherDuck uses Auth0 hosted login. The signup flow navigates from `app.motherduck.com` to `auth.motherduck.com/u/signup`. After email verification, the tool clears the session and re-logs in to get a clean authenticated state.

Helper functions `_on_auth0(page)` and `_on_app(page)` detect which domain the browser is on to handle the Auth0 ↔ app transitions.

**Onboarding (`_skip_onboarding`)**

After first login, MotherDuck shows an onboarding wizard:
- User information form (first/last name)
- Region selection (US East default)
- TOS checkboxes
- Optional survey questions

The code loops up to 10 attempts, detecting the current step and filling/clicking appropriately. It only acts on `app.motherduck.com` pages — if still on Auth0, it waits.

**Token Extraction (`_extract_token`)**

Four strategies in priority order:
1. **DOM elements**: `<code>`, `input[readonly]`, `textarea[readonly]`, `[data-testid*="token"]`, `<pre>`, `.token`
2. **Page text regex**: scans full page text for JWT patterns or `motherduck_token_` strings
3. **localStorage**: checks keys `motherduck_token`, `token`, `md_token`, `access_token`
4. **Cookies**: checks same key names in browser cookies

**Browser Configuration**
- Persistent browser context with temp user data dir
- Chrome channel (uses system Chrome install)
- Viewport: 1280x900, locale: en-US
- Humanized typing delay (55ms between keystrokes)
- Linux: auto-wraps with Xvfb if `DISPLAY` not set

### store.py

DuckDB-backed local state. Three tables:

**accounts**: `id`, `email`, `password` (MotherDuck login), `token` (API token), `is_active`, `created_at`

**databases**: `id`, `account_id` (FK), `name`, `alias` (unique, used for CLI), `is_default`, `notes`, `last_used_at`, `created_at`

**query_log**: `id`, `db_id` (FK), `sql`, `rows_returned`, `duration_ms`, `ran_at`

The `set_default(alias)` method uses a transaction to clear all `is_default` flags, then set the target. `get_default_db()` joins databases with accounts to return connection details including the API token.

### client.py

Connects to MotherDuck using the DuckDB `md:` protocol with the API token. The connection string is `md:?motherduck_token={token}`. Exposes `create_db(name)`, `run_query(db_name, sql)` returning `(rows, column_names)`, and `list_dbs()`.

## Comparison with clickhouse-tool

| Aspect | motherduck | clickhouse-tool |
|--------|-----------|-----------------|
| Auth | API token (extracted from browser) | Username + password (reset via console) |
| Connection | DuckDB `md:` protocol | clickhouse-connect HTTPS |
| Password capture | N/A (token-based) | Eye icon click in reset dialog |
| Cloud API | None (DuckDB driver handles it) | REST API for service management |
| State tables | accounts, databases, query_log | accounts, services, query_log |

## Dependencies

| Package | Purpose |
|---------|---------|
| typer | CLI framework |
| rich | Terminal output (tables, status spinners) |
| patchright | Browser automation (Playwright fork, less detectable) |
| duckdb | Local state storage + MotherDuck connection |
| httpx | HTTP client (mail.tm API) |
| faker | Random identity generation |

## Testing

All tests are in `tests/` and use `unittest.mock` to avoid real network/browser calls:

- `test_store.py` — DuckDB CRUD operations (in-memory DB)
- `test_client.py` — MotherDuckClient with mocked DuckDB
- `test_email.py` — mail.tm client with mocked httpx, magic link picker

No browser tests — `browser.py` is tested via integration runs (`register --no-headless --verbose`).
