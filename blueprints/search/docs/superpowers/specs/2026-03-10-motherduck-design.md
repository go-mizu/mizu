# MotherDuck Tool — Design Spec

**Date:** 2026-03-10
**Location:** `tools/motherduck/`

---

## Overview

A UV + Python CLI tool that auto-registers MotherDuck accounts (via Patchright browser automation + mail.tm disposable email), manages databases, and queries data. State persisted in a local DuckDB file.

---

## Stack

| Concern | Choice | Reason |
|---|---|---|
| Python | UV (script/project mode) | Matches project convention |
| Browser automation | Patchright (real Chrome) | Matches x-register; bypasses bot detection |
| Email | mail.tm API | Free disposable; already used in x-register |
| CLI framework | **Typer + Rich** | Modern Python; type-hint driven; beautiful tables/errors automatically |
| State store | DuckDB (`$HOME/data/motherduck/mother.duckdb`) | First-class in project; queryable; no ORM needed |
| Cloud DuckDB | `duckdb` Python SDK + motherduck extension | Official SDK; `md:?motherduck_token=...` DSN |
| Fake identity | Faker | Matches x-register |

### Why Typer + Rich over argparse
- Type hints = self-documenting; no manual `add_argument` boilerplate
- Rich auto-renders tables, progress bars, error panels
- `typer.Typer()` with `app.add_typer()` gives Docker-style command groups natively
- `--help` is beautiful out of the box (panels, colors)
- `--install-completion` for shell tab-completion with zero code

---

## CLI Command Structure

Docker-style grouped subcommands:

```
motherduck register              # Browser signup → store account + token

motherduck account ls            # Rich table: id, email, created_at, db_count
motherduck account rm <email>    # Soft-delete (set is_active=false)

motherduck db create <name>      # CREATE on MotherDuck, record locally
    --alias TEXT   (default: name)
    --account TEXT (email; default: first active account)
    --default      (set as default immediately)
motherduck db ls                 # Rich table: alias, name, account, default, queries, last_used
motherduck db use <alias>        # Set default database (clears previous)
motherduck db rm <alias>         # Remove from local state (not cloud)

motherduck query <sql>           # Run SQL, print Rich table of results
    --db TEXT      (alias or name; omit = use default)
    --json         (output raw JSON instead of Rich table)
```

### DX Principles Applied
- **Noun-verb grouping** (`db create`, `db ls`) mirrors Docker, `kubectl`, `gh`
- `register` is top-level (no group) — it's the entry action, not a sub-resource
- `query` is top-level — it's the primary operation, not a resource
- `db use` mirrors `kubectl config use-context` / `docker context use`
- All list commands use `ls` (not `list`) — short, Unix-standard
- Rich tables with color: default DB highlighted, active account bolded
- Errors exit with non-zero code + `[red]` panel via Rich, never tracebacks shown to user
- `--json` on `query` enables scripting/piping

---

## File Structure

```
tools/motherduck/
  pyproject.toml          # UV package, dependencies, scripts entry point
  src/motherduck/
    __init__.py
    cli.py                # Typer app + sub-apps wired together
    email.py              # mail.tm client (port from x-register with OTP→magic-link adaptation)
    browser.py            # Patchright signup: app.motherduck.com flow
    store.py              # Local DuckDB state: schema init, CRUD for accounts/databases/query_log
    client.py             # MotherDuck connection wrapper: connect, create_db, run_query
    identity.py           # Faker: display_name, password generation
```

---

## Local State Schema (`mother.duckdb`)

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id          VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email       VARCHAR NOT NULL UNIQUE,
    password    VARCHAR NOT NULL,
    token       VARCHAR NOT NULL,
    created_at  TIMESTAMP DEFAULT now(),
    is_active   BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS databases (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    name         VARCHAR NOT NULL,           -- actual MotherDuck DB name
    alias        VARCHAR NOT NULL UNIQUE,    -- local alias (default = name)
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    db_id         VARCHAR REFERENCES databases(id),
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
```

Invariant: `is_default = true` for at most one row in `databases` (enforced in store.py via transaction: clear all, then set one).

---

## Registration Flow (`browser.py`)

1. Create `mail.tm` mailbox via API
2. Open `https://app.motherduck.com` in patchright persistent Chrome context
3. Click "Sign up with email", fill mail.tm address
4. Poll mail.tm inbox for magic link (not OTP — MotherDuck sends a link)
5. Extract link URL from email body, navigate to it
6. Wait for account creation / onboarding screens, click through
7. Navigate to `app.motherduck.com/settings/tokens` → click "Generate token"
8. Extract token text from page
9. Close browser, store `(email, password, token)` in `accounts` table

---

## Token Extraction Strategy

MotherDuck tokens appear in the Settings → Tokens page as visible text after generation. Fallback: check `localStorage` via `page.evaluate("() => localStorage.getItem('motherduck_token')")` if page extraction fails.

---

## Dependencies (`pyproject.toml`)

```toml
dependencies = [
    "typer>=0.12",
    "rich>=13.0",
    "patchright>=1.50",
    "duckdb>=1.2",
    "httpx>=0.27",
    "faker>=33.0",
]
```

---

## Testing

Two test targets:
1. `test_store.py` — unit test schema init, account/db CRUD, default rotation, using in-memory DuckDB (`:memory:`)
2. Integration: `motherduck register --no-headless` run manually, then `motherduck db create test_db`, then `motherduck query "SELECT 1" --db test_db`
