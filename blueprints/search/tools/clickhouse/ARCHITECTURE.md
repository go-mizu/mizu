# Architecture

## Overview

clickhouse-tool automates ClickHouse Cloud trial account creation using browser automation (Patchright) and disposable email (mail.tm), then provides a CLI for managing services and running SQL queries.

```
┌──────────┐     ┌──────────┐     ┌────────────────────┐
│  cli.py  │────▸│ store.py │────▸│ clickhouse.duckdb  │
│  (Typer) │     │ (DuckDB) │     │ accounts/services/ │
└────┬─────┘     └──────────┘     │ query_log          │
     │                            └────────────────────┘
     │  register
     ▼
┌────────────┐   ┌────────────┐   ┌──────────────────┐
│ identity.py│──▸│  email.py  │──▸│ mail.tm API       │
│ (Faker)    │   │  (httpx)   │   │ (disposable mail) │
└────────────┘   └────────────┘   └──────────────────┘
     │
     ▼
┌────────────┐   ┌──────────────────────────────────┐
│ browser.py │──▸│ ClickHouse Cloud Console         │
│(Patchright)│   │ (signup, onboard, settings)      │
└────────────┘   └──────────────────────────────────┘
     │
     │  query
     ▼
┌────────────┐   ┌──────────────────────────────────┐
│ client.py  │──▸│ ClickHouse Cloud Service         │
│(ch-connect)│   │ (HTTPS port 8443)                │
└────────────┘   └──────────────────────────────────┘
```

## Modules

### cli.py

Typer app with four command groups: `register`, `account {ls,rm}`, `service {create,ls,use,rm}`, `query`. All commands are thin wrappers that delegate to the other modules. Rich tables for output.

### identity.py

Generates a random identity using Faker: display name, email local part, and a strong password (14 chars, mixed case + digits + special). The email local part is `{first}{last}{NN}` — simple enough for mail.tm, random enough to avoid collisions.

### email.py

HTTP client for the [mail.tm API](https://docs.mail.tm/):

1. `_get_domain()` — fetches available domains, picks the first active one
2. `create_mailbox(local)` — creates `{local}@{domain}` with a derived password
3. `poll_for_verification_link(mailbox)` — polls inbox every 3s (up to 120s), extracts the verification URL from the first email

Link extraction uses `_pick_verification_link()` which prioritizes URLs containing `verify`, `confirm`, or `token=` keywords, then falls back to long ClickHouse URLs with query params.

### browser.py

The largest module (~1100 lines). Drives a Chromium browser through the full ClickHouse Cloud signup and service provisioning flow.

#### Registration Flow

`register_via_browser()` is the entry point:

```
signUp page ──▸ fill email ──▸ fill password ──▸ submit
    │
    ▼
email verification? ──▸ poll mail.tm ──▸ click link ──▸ login
    │
    ▼
onboarding ──▸ select use case ──▸ select AWS ──▸ Create service
    │
    ▼
wait for provisioning (up to 5 min) ──▸ detect "Provisioning" banner gone
    │
    ▼
connect page ──▸ extract host from HTML
    │
    ▼
settings page ──▸ reset password ──▸ click eye icon ──▸ read password
    │
    ▼
return {service_id, host, port, db_password}
```

#### Key Implementation Details

**Auth0 Login (`_do_auth0_login`)**

ClickHouse uses Auth0's identifier-first flow. The email and password are on separate pages. After submitting email, the code waits for the URL to change to `/password`, then fills and submits the password.

**Onboarding (`_handle_onboarding`)**

The onboarding wizard has multiple pages: use case selection, cloud provider/region, service creation. The code loops up to 25 attempts, detecting the current page by its text content and taking the appropriate action. It always selects AWS as the provider and dismisses overlays (cookie consent, etc.) with force-click fallback.

A `MutationObserver` is injected before clicking "Create service" to catch any modal/dialog that appears (the popup watcher). Credentials polling runs every 500ms for 30s checking both the observer captures and direct DOM queries.

**Service Readiness (`_wait_for_service_ready`)**

After service creation, the console shows a "Provisioning service, this may take a few moments..." banner above the SQL console. The code polls every 10s, checking for the absence of "provisioning" in the page body text. The SQL console content (tables, queries) appears alongside the provisioning banner, so we cannot use those as readiness indicators.

**Host Extraction (`_get_host_from_connect_page`)**

Navigates to `/services/{id}/connect`, dismisses any survey popup, clicks the Connect sidebar link, and extracts the host from the page HTML using a regex pattern: `{slug}.{region}.{provider}.clickhouse.cloud`.

**Password Reset (`_reset_service_password`)**

ClickHouse Cloud does not expose the service password via API. The `resetPassword` API returns status 200 with an empty body — the password is only visible client-side in the reset dialog.

The flow:
1. Navigate to `/services/{id}/settings`
2. Find "Reset password" under the "Service actions" section
3. Click it — a confirmation dialog appears
4. Click "Reset password" inside the dialog
5. Wait 5s for password generation
6. Click the eye icon (`[data-testid="password-display-eye-icon"]`) to reveal the masked password
7. Read the revealed password from the dialog text

Fallbacks: network response interception (checks `control-plane-internal` API responses for a `"password"` field), then clipboard interception via copy button.

**Helper Utilities**

- `_click_first(page, selectors)` — tries selectors in order, clicks the first match, returns which one worked
- `_fill_first(page, selectors, text)` — same pattern for input fields
- `_inject_popup_watcher(page)` — injects a MutationObserver that records all new dialog/modal DOM nodes
- `_looks_like_password(s)` — heuristic: 8-64 chars, no spaces, mixed case or special chars, not a button label
- `_dismiss_survey(page)` — clicks through "Tell us about your use case" popup

### store.py

DuckDB-backed local state. Three tables:

**accounts**: `id`, `email`, `password` (ClickHouse Cloud login), `org_id`, `api_key_id`, `api_key_secret`, `is_active`, `created_at`

**services**: `id`, `account_id` (FK), `cloud_id`, `name`, `alias` (unique, used for CLI), `host`, `port`, `db_user`, `db_password`, `provider`, `region`, `is_default`, `last_used_at`

**query_log**: `id`, `service_id` (FK), `sql`, `rows_returned`, `duration_ms`, `ran_at`

The `set_default(alias)` method uses a transaction to clear all `is_default` flags, then set the target. `get_default_service()` joins services with accounts to return full connection details.

### client.py

Thin wrapper around `clickhouse-connect`. Connects over HTTPS (port 8443) with username/password auth. Exposes `run_query(sql)` returning `(rows, column_names)` and `create_db(name)`.

### cloud_api.py

REST API client for `api.clickhouse.cloud/v1`. Used by `service create` to provision services programmatically (requires API keys). Supports `get_organizations`, `create_service`, `list_services`, `get_service`, `delete_service`.

## Dependencies

| Package | Purpose |
|---------|---------|
| typer | CLI framework |
| rich | Terminal output (tables, status spinners) |
| patchright | Browser automation (Playwright fork, less detectable) |
| duckdb | Local state storage |
| httpx | HTTP client (mail.tm API, cloud API) |
| faker | Random identity generation |
| clickhouse-connect | ClickHouse native HTTPS client |

## Testing

All tests are in `tests/` and use `unittest.mock` to avoid real network/browser calls:

- `test_store.py` — DuckDB CRUD operations (in-memory DB)
- `test_client.py` — ClickHouseClient with mocked clickhouse-connect
- `test_email.py` — mail.tm client with mocked httpx, verification link picker
- `test_cloud_api.py` — Cloud API with mocked httpx

No browser tests — `browser.py` is tested via integration runs (`register --no-headless --verbose`).
