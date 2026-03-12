# Cloudflare Tool Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `tools/cloudflare` — a Python CLI that auto-registers Cloudflare accounts via browser automation, creates API tokens with permission presets, manages Workers (deploy/tail/invoke), and stores all credentials in DuckDB.

**Architecture:** Follows the exact pattern of `tools/motherduck` and `tools/clickhouse`. Patchright drives browser signup and token creation. CF REST API (httpx) handles workers ls/rm and account info. Wrangler subprocess handles deploy and tail. DuckDB stores local state. Typer + Rich for CLI.

**Tech Stack:** Python 3.11+, uv, Patchright (Playwright fork), Typer, Rich, httpx, DuckDB, Faker, pytest, wrangler (npx, external)

**Spec:** `blueprints/search/spec/0711_cloudflare.md`

**Reference implementations:**
- `blueprints/search/tools/motherduck/` — identity, email, browser, store, client patterns
- `blueprints/search/tools/clickhouse/` — cloud_api, service management patterns

---

## Chunk 1: Project scaffold + store

### Task 1: Scaffold the project

**Files:**
- Create: `tools/cloudflare/pyproject.toml`
- Create: `tools/cloudflare/Makefile`
- Create: `tools/cloudflare/src/cloudflare_tool/__init__.py`
- Create: `tools/cloudflare/tests/__init__.py`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p tools/cloudflare/src/cloudflare_tool
mkdir -p tools/cloudflare/tests
touch tools/cloudflare/src/cloudflare_tool/__init__.py
touch tools/cloudflare/tests/__init__.py
```

- [ ] **Step 2: Write `pyproject.toml`**

Create `tools/cloudflare/pyproject.toml`:

```toml
[project]
name = "cloudflare-tool"
version = "0.1.0"
description = "Auto-register Cloudflare accounts, create API tokens, and manage Workers"
requires-python = ">=3.11"
dependencies = [
    "typer>=0.12",
    "rich>=13.0",
    "patchright>=1.50",
    "duckdb>=1.2,<1.5",
    "httpx>=0.27",
    "faker>=33.0",
]

[project.scripts]
cloudflare-tool = "cloudflare_tool.cli:app_entry"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/cloudflare_tool"]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "pytest-mock>=3.14",
]
```

- [ ] **Step 3: Write `Makefile`**

Create `tools/cloudflare/Makefile`:

```makefile
.PHONY: install test lint

install:
	uv sync

test:
	uv run pytest tests/ -v

browser:
	uv run patchright install chromium
```

- [ ] **Step 4: Run `uv sync` to lock dependencies**

```bash
cd tools/cloudflare && uv sync
```

Expected: `uv.lock` created, no errors.

- [ ] **Step 5: Commit**

```bash
git add tools/cloudflare/pyproject.toml tools/cloudflare/Makefile \
        tools/cloudflare/src/cloudflare_tool/__init__.py \
        tools/cloudflare/tests/__init__.py tools/cloudflare/uv.lock
git commit -m "feat(cloudflare): scaffold project structure"
```

---

### Task 2: identity.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/identity.py`

Copy the identical pattern from `tools/motherduck/src/motherduck/identity.py`. No differences needed.

- [ ] **Step 1: Write `identity.py`**

Create `tools/cloudflare/src/cloudflare_tool/identity.py`:

```python
"""Random identity generation for Cloudflare account registration."""
from __future__ import annotations

import secrets
import string
from dataclasses import dataclass

from faker import Faker

_fake = Faker()

_PWD_CHARS = string.ascii_letters + string.digits + "!@#$%^&*"


@dataclass
class Identity:
    display_name: str
    email_local: str   # part before @, max 20 chars
    password: str


def generate() -> Identity:
    first = _fake.first_name()
    last = _fake.last_name()
    display_name = f"{first} {last}"
    raw_local = f"{first.lower()}{last.lower()}{secrets.randbelow(9999)}"
    email_local = raw_local[:20]
    password = "".join(secrets.choice(_PWD_CHARS) for _ in range(14))
    return Identity(display_name=display_name, email_local=email_local, password=password)
```

- [ ] **Step 2: Verify no import errors**

```bash
cd tools/cloudflare && uv run python -c "from cloudflare_tool.identity import generate; print(generate())"
```

Expected: prints an Identity dataclass with display_name, email_local, password.

- [ ] **Step 3: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/identity.py
git commit -m "feat(cloudflare): add identity generator"
```

---

### Task 3: email.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/email.py`
- Create: `tools/cloudflare/tests/test_email.py`

Copy from `tools/motherduck/src/motherduck/email.py` — mail.tm client is identical across all tools.

- [ ] **Step 1: Write failing tests**

Create `tools/cloudflare/tests/test_email.py`:

```python
"""Unit tests for email.py — mocked httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from cloudflare_tool.email import MailTmClient, Mailbox


@pytest.fixture
def mock_httpx(monkeypatch):
    client = MagicMock()
    monkeypatch.setattr("cloudflare_tool.email.httpx.Client", lambda **kw: client)
    return client


def test_create_mailbox(mock_httpx):
    # GET /domains → one domain
    domains_resp = MagicMock()
    domains_resp.raise_for_status = MagicMock()
    domains_resp.json.return_value = {"hydra:member": [{"domain": "example.tm", "isActive": True}]}

    # POST /accounts → mailbox created
    account_resp = MagicMock()
    account_resp.raise_for_status = MagicMock()
    account_resp.json.return_value = {"id": "mb1", "address": "alice@example.tm"}

    # POST /token → auth token
    token_resp = MagicMock()
    token_resp.raise_for_status = MagicMock()
    token_resp.json.return_value = {"token": "jwt-abc"}

    mock_httpx.get.return_value = domains_resp
    mock_httpx.post.side_effect = [account_resp, token_resp]

    c = MailTmClient()
    mb = c.create_mailbox("alice")

    assert mb.address == "alice@example.tm"
    assert mb.id == "mb1"


def test_poll_for_magic_link_finds_link(mock_httpx):
    messages_resp = MagicMock()
    messages_resp.raise_for_status = MagicMock()
    messages_resp.json.return_value = {
        "hydra:member": [{"id": "msg1"}]
    }

    msg_detail = MagicMock()
    msg_detail.raise_for_status = MagicMock()
    msg_detail.json.return_value = {
        "text": "Click here: https://dash.cloudflare.com/verify?token=abc123"
    }

    mock_httpx.get.side_effect = [messages_resp, msg_detail]

    # Bootstrap the client with a token by going through create_mailbox first,
    # then wire the subsequent GET/GET calls for poll_for_magic_link.
    # Re-use the same mock_httpx fixture; reset side_effects for the poll calls.
    mock_httpx.get.side_effect = [messages_resp, msg_detail]
    # Use create_mailbox to set internal token state, then call poll directly
    c = MailTmClient()
    mb = Mailbox(address="alice@example.tm", password="pw", id="mb1")
    # Set auth header directly via the public interface the motherduck email.py exposes:
    # MailTmClient stores the token as self._token after create_mailbox().
    # We replicate that here by assigning it — this is consistent with the
    # motherduck/clickhouse email.py implementation (see tools/motherduck/src/motherduck/email.py).
    c._token = "jwt-abc"

    link = c.poll_for_magic_link(mb, timeout=5)
    assert "cloudflare.com" in link or "verify" in link or link.startswith("https://")
```

- [ ] **Step 2: Run failing tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_email.py -v
```

Expected: FAIL with `ModuleNotFoundError` (email.py doesn't exist yet).

- [ ] **Step 3: Copy email.py from motherduck**

```bash
cp ../motherduck/src/motherduck/email.py src/cloudflare_tool/email.py
```

Then update the module docstring: replace "MotherDuck" with "Cloudflare" in the top docstring only. The file provides:
- `Mailbox` dataclass: `address`, `password`, `id`
- `MailTmClient`: `create_mailbox(local)`, `poll_for_magic_link(mailbox, timeout)`, `close()`
- Internal `_token` field set after `create_mailbox()` succeeds (used in test_email.py)

- [ ] **Step 4: Run tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_email.py -v
```

Expected: PASS (2 tests)

- [ ] **Step 5: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/email.py \
        tools/cloudflare/tests/test_email.py
git commit -m "feat(cloudflare): add mail.tm email client + tests"
```

---

### Task 4: store.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/store.py`
- Create: `tools/cloudflare/tests/test_store.py`

- [ ] **Step 1: Write failing tests first**

Create `tools/cloudflare/tests/test_store.py`:

```python
"""Unit tests for store.py — in-memory DuckDB."""
from __future__ import annotations

import pytest
from cloudflare_tool.store import Store


@pytest.fixture
def store():
    return Store(":memory:")


def test_init_creates_tables(store):
    tables = store.con.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
    ).fetchall()
    names = {r[0] for r in tables}
    assert "accounts" in names
    assert "tokens" in names
    assert "workers" in names
    assert "op_log" in names


def test_add_account(store):
    acc_id = store.add_account(
        email="test@x.com", password="pass", account_id="acc123"
    )
    assert acc_id is not None
    rows = store.list_accounts()
    assert len(rows) == 1
    assert rows[0]["email"] == "test@x.com"
    assert rows[0]["account_id"] == "acc123"
    assert rows[0]["is_active"] is True


def test_add_account_duplicate_raises(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    with pytest.raises(Exception):
        store.add_account(email="a@x.com", password="p2", account_id="a2")


def test_deactivate_account(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    store.deactivate_account("a@x.com")
    rows = store.list_accounts()
    assert rows[0]["is_active"] is False


def test_get_first_active_account(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    acc = store.get_first_active_account()
    assert acc is not None
    assert acc["email"] == "a@x.com"
    assert acc["account_id"] == "a1"


def test_add_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(
        account_id=acc_id, name="my-token",
        token_value="tok_abc123", preset="browser-rendering"
    )
    assert tok_id is not None
    rows = store.list_tokens()
    assert len(rows) == 1
    assert rows[0]["name"] == "my-token"
    assert rows[0]["preset"] == "browser-rendering"


def test_set_default_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_token(account_id=acc_id, name="t2", token_value="v2", preset="workers")
    store.set_default_token("t1")
    store.set_default_token("t2")
    default = store.get_default_token()
    assert default["name"] == "t2"
    # Only one default
    rows = store.list_tokens()
    defaults = [r for r in rows if r["is_default"]]
    assert len(defaults) == 1


def test_remove_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.remove_token("t1")
    assert store.list_tokens() == []


def test_add_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="workers")
    w_id = store.add_worker(
        account_id=acc_id, token_id=tok_id,
        name="my-worker", alias="mw",
        url="https://my-worker.example.workers.dev"
    )
    assert w_id is not None
    rows = store.list_workers()
    assert len(rows) == 1
    assert rows[0]["alias"] == "mw"
    assert rows[0]["name"] == "my-worker"


def test_set_default_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w1", alias="w1", url="u1")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w2", alias="w2", url="u2")
    store.set_default_worker("w2")
    default = store.get_default_worker()
    assert default["alias"] == "w2"


def test_remove_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w1", alias="w1", url="u")
    store.remove_worker("w1")
    assert store.list_workers() == []


def test_log_operation(store):
    store.log_op(worker_id=None, operation="deploy", detail="my-worker", duration_ms=500)
    rows = store.con.execute("SELECT * FROM op_log").fetchall()
    assert len(rows) == 1
    assert rows[0][2] == "deploy"
```

- [ ] **Step 2: Run failing tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_store.py -v
```

Expected: FAIL (store.py doesn't exist).

- [ ] **Step 3: Write `store.py`**

Create `tools/cloudflare/src/cloudflare_tool/store.py`:

```python
"""Local DuckDB state: schema init, CRUD for accounts/tokens/workers/op_log."""
from __future__ import annotations

from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "cloudflare" / "cloudflare.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id          VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email       VARCHAR NOT NULL UNIQUE,
    password    VARCHAR NOT NULL,
    account_id  VARCHAR NOT NULL,
    subdomain   VARCHAR DEFAULT '',
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tokens (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    name         VARCHAR NOT NULL UNIQUE,
    token_id     VARCHAR DEFAULT '',
    token_value  VARCHAR NOT NULL,
    preset       VARCHAR NOT NULL DEFAULT 'all',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workers (
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

CREATE TABLE IF NOT EXISTS op_log (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    worker_id    VARCHAR,
    operation    VARCHAR NOT NULL,
    detail       VARCHAR DEFAULT '',
    duration_ms  INTEGER,
    ran_at       TIMESTAMP DEFAULT now()
);
"""


class Store:
    def __init__(self, path: str | Path = DEFAULT_DB_PATH) -> None:
        if str(path) != ":memory:":
            Path(path).parent.mkdir(parents=True, exist_ok=True)
        self.con = duckdb.connect(str(path))
        self.con.execute(_SCHEMA)

    # ------------------------------------------------------------------
    # Accounts
    # ------------------------------------------------------------------

    def add_account(
        self, *, email: str, password: str, account_id: str, subdomain: str = ""
    ) -> str:
        row = self.con.execute(
            "INSERT INTO accounts (email, password, account_id, subdomain) "
            "VALUES (?, ?, ?, ?) RETURNING id",
            [email, password, account_id, subdomain],
        ).fetchone()
        return row[0]

    def list_accounts(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT a.id, a.email, a.account_id, a.subdomain, a.is_active, a.created_at,
                   COUNT(DISTINCT t.id) AS token_count,
                   COUNT(DISTINCT w.id) AS worker_count
            FROM accounts a
            LEFT JOIN tokens t ON t.account_id = a.id
            LEFT JOIN workers w ON w.account_id = a.id
            GROUP BY a.id, a.email, a.account_id, a.subdomain, a.is_active, a.created_at
            ORDER BY a.created_at DESC
            """
        ).fetchall()
        cols = ["id", "email", "account_id", "subdomain", "is_active", "created_at",
                "token_count", "worker_count"]
        return [dict(zip(cols, r)) for r in rows]

    def deactivate_account(self, email: str) -> None:
        self.con.execute(
            "UPDATE accounts SET is_active = false WHERE email = ?", [email]
        )

    def get_first_active_account(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, account_id, subdomain "
            "FROM accounts WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "account_id", "subdomain"], row))

    def get_account_by_email(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, account_id, subdomain "
            "FROM accounts WHERE email = ? AND is_active = true",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "account_id", "subdomain"], row))

    def update_subdomain(self, email: str, subdomain: str) -> None:
        self.con.execute(
            "UPDATE accounts SET subdomain = ? WHERE email = ?", [subdomain, email]
        )

    # ------------------------------------------------------------------
    # Tokens
    # ------------------------------------------------------------------

    def add_token(
        self, *, account_id: str, name: str, token_value: str,
        preset: str = "all", token_id: str = ""
    ) -> str:
        row = self.con.execute(
            "INSERT INTO tokens (account_id, name, token_value, preset, token_id) "
            "VALUES (?, ?, ?, ?, ?) RETURNING id",
            [account_id, name, token_value, preset, token_id],
        ).fetchone()
        return row[0]

    def list_tokens(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT t.id, t.name, t.preset, t.is_default, t.created_at,
                   a.email, a.account_id
            FROM tokens t
            JOIN accounts a ON a.id = t.account_id
            ORDER BY t.created_at DESC
            """
        ).fetchall()
        cols = ["id", "name", "preset", "is_default", "created_at", "email", "account_id"]
        return [dict(zip(cols, r)) for r in rows]

    def get_token_by_name(self, name: str) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT t.id, t.name, t.token_value, t.preset, t.is_default,
                   a.id AS account_db_id, a.account_id, a.subdomain, a.email,
                   a.password
            FROM tokens t
            JOIN accounts a ON a.id = t.account_id
            WHERE t.name = ?
            """,
            [name],
        ).fetchone()
        if not row:
            return None
        return dict(zip(
            ["id", "name", "token_value", "preset", "is_default",
             "account_db_id", "account_id", "subdomain", "email", "password"],
            row,
        ))

    def get_default_token(self) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT t.id, t.name, t.token_value, t.preset, t.is_default,
                   a.id AS account_db_id, a.account_id, a.subdomain, a.email,
                   a.password
            FROM tokens t
            JOIN accounts a ON a.id = t.account_id
            WHERE t.is_default = true
            LIMIT 1
            """
        ).fetchone()
        if not row:
            return None
        return dict(zip(
            ["id", "name", "token_value", "preset", "is_default",
             "account_db_id", "account_id", "subdomain", "email", "password"],
            row,
        ))

    def set_default_token(self, name: str) -> None:
        self.con.execute("BEGIN")
        try:
            self.con.execute("UPDATE tokens SET is_default = false")
            self.con.execute(
                "UPDATE tokens SET is_default = true WHERE name = ?", [name]
            )
            self.con.execute("COMMIT")
        except Exception:
            self.con.execute("ROLLBACK")
            raise

    def remove_token(self, name: str) -> None:
        self.con.execute("DELETE FROM tokens WHERE name = ?", [name])

    # ------------------------------------------------------------------
    # Workers
    # ------------------------------------------------------------------

    def add_worker(
        self, *, account_id: str, token_id: str | None, name: str,
        alias: str, url: str = ""
    ) -> str:
        from datetime import datetime, timezone
        row = self.con.execute(
            "INSERT INTO workers (account_id, token_id, name, alias, url, deployed_at) "
            "VALUES (?, ?, ?, ?, ?, ?) RETURNING id",
            [account_id, token_id, name, alias, url,
             datetime.now(timezone.utc)],
        ).fetchone()
        return row[0]

    def list_workers(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT w.id, w.alias, w.name, w.url, w.is_default,
                   w.deployed_at, w.created_at,
                   a.email,
                   COUNT(o.id) AS op_count,
                   MAX(o.ran_at) AS last_op_at
            FROM workers w
            JOIN accounts a ON a.id = w.account_id
            LEFT JOIN op_log o ON o.worker_id = w.id
            GROUP BY w.id, w.alias, w.name, w.url, w.is_default,
                     w.deployed_at, w.created_at, a.email
            ORDER BY w.created_at DESC
            """
        ).fetchall()
        cols = ["id", "alias", "name", "url", "is_default", "deployed_at",
                "created_at", "email", "op_count", "last_op_at"]
        return [dict(zip(cols, r)) for r in rows]

    def get_worker(self, alias: str) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT w.id, w.alias, w.name, w.url, w.is_default,
                   a.account_id, a.subdomain, a.email,
                   t.token_value
            FROM workers w
            JOIN accounts a ON a.id = w.account_id
            LEFT JOIN tokens t ON t.id = w.token_id
            WHERE w.alias = ?
            """,
            [alias],
        ).fetchone()
        if not row:
            return None
        return dict(zip(
            ["id", "alias", "name", "url", "is_default",
             "account_id", "subdomain", "email", "token_value"],
            row,
        ))

    def get_default_worker(self) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT w.id, w.alias, w.name, w.url, w.is_default,
                   a.account_id, a.subdomain, a.email,
                   t.token_value
            FROM workers w
            JOIN accounts a ON a.id = w.account_id
            LEFT JOIN tokens t ON t.id = w.token_id
            WHERE w.is_default = true
            LIMIT 1
            """
        ).fetchone()
        if not row:
            return None
        return dict(zip(
            ["id", "alias", "name", "url", "is_default",
             "account_id", "subdomain", "email", "token_value"],
            row,
        ))

    def set_default_worker(self, alias: str) -> None:
        self.con.execute("BEGIN")
        try:
            self.con.execute("UPDATE workers SET is_default = false")
            self.con.execute(
                "UPDATE workers SET is_default = true WHERE alias = ?", [alias]
            )
            self.con.execute("COMMIT")
        except Exception:
            self.con.execute("ROLLBACK")
            raise

    def remove_worker(self, alias: str) -> None:
        self.con.execute("DELETE FROM workers WHERE alias = ?", [alias])

    def update_worker_url(self, alias: str, url: str) -> None:
        self.con.execute(
            "UPDATE workers SET url = ? WHERE alias = ?", [url, alias]
        )

    # ------------------------------------------------------------------
    # Op log
    # ------------------------------------------------------------------

    def log_op(
        self, *, worker_id: str | None, operation: str,
        detail: str = "", duration_ms: int = 0
    ) -> None:
        self.con.execute(
            "INSERT INTO op_log (worker_id, operation, detail, duration_ms) "
            "VALUES (?, ?, ?, ?)",
            [worker_id, operation, detail, duration_ms],
        )
```

- [ ] **Step 4: Run tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_store.py -v
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/store.py \
        tools/cloudflare/tests/test_store.py
git commit -m "feat(cloudflare): add DuckDB store with full schema + tests"
```

---

## Chunk 2: CF REST API client + workers ops

### Task 5: client.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/client.py`
- Create: `tools/cloudflare/tests/test_client.py`

- [ ] **Step 1: Write failing tests**

Create `tools/cloudflare/tests/test_client.py`:

```python
"""Unit tests for client.py — mocked httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from cloudflare_tool.client import CloudflareClient


@pytest.fixture
def client():
    return CloudflareClient(account_id="acc123", api_token="tok_abc")


def _mock_response(data: dict, status: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status
    resp.raise_for_status = MagicMock()
    resp.json.return_value = data
    return resp


def test_get_account_id(monkeypatch):
    """get_account_id() uses API to fetch first account."""
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": [{"id": "acc-xyz", "name": "My Account"}],
        "success": True,
    })
    monkeypatch.setattr("cloudflare_tool.client.httpx.Client", lambda **kw: mock_http)
    c = CloudflareClient(account_id="", api_token="tok_abc")
    account_id = c.get_account_id()
    assert account_id == "acc-xyz"


def test_get_subdomain(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": {"subdomain": "myaccount"},
        "success": True,
    })
    client._http = mock_http
    sub = client.get_subdomain()
    assert sub == "myaccount"


def test_list_workers(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": [
            {"id": "w1", "script": "my-worker", "created_on": "2024-01-01"},
        ],
        "success": True,
    })
    client._http = mock_http
    workers = client.list_workers()
    assert len(workers) == 1
    assert workers[0]["script"] == "my-worker"


def test_delete_worker(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.delete.return_value = _mock_response({"result": None, "success": True})
    client._http = mock_http
    # Should not raise
    client.delete_worker("my-worker")
    mock_http.delete.assert_called_once()


def test_close(client):
    """close() does not raise."""
    client.close()
```

- [ ] **Step 2: Run failing tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_client.py -v
```

Expected: FAIL (client.py doesn't exist).

- [ ] **Step 3: Write `client.py`**

Create `tools/cloudflare/src/cloudflare_tool/client.py`:

```python
"""Cloudflare REST API client for account info, workers management."""
from __future__ import annotations

import httpx

_BASE = "https://api.cloudflare.com/client/v4"


class CloudflareClient:
    def __init__(self, account_id: str, api_token: str) -> None:
        self.account_id = account_id
        self._http = httpx.Client(
            headers={
                "Authorization": f"Bearer {api_token}",
                "Content-Type": "application/json",
            },
            timeout=30.0,
        )

    def get_account_id(self) -> str:
        """Fetch the first account ID from the API (use when account_id not yet known)."""
        r = self._http.get(f"{_BASE}/accounts", params={"per_page": 1})
        r.raise_for_status()
        result = r.json().get("result", [])
        if not result:
            raise RuntimeError("No Cloudflare accounts found via API")
        self.account_id = result[0]["id"]
        return self.account_id

    def get_subdomain(self) -> str:
        """Return the workers.dev subdomain for this account."""
        r = self._http.get(
            f"{_BASE}/accounts/{self.account_id}/workers/subdomain"
        )
        r.raise_for_status()
        return r.json().get("result", {}).get("subdomain", "")

    def list_workers(self) -> list[dict]:
        """List all Worker scripts in this account."""
        r = self._http.get(
            f"{_BASE}/accounts/{self.account_id}/workers/scripts"
        )
        r.raise_for_status()
        return r.json().get("result", [])

    def delete_worker(self, name: str) -> None:
        """Delete a Worker script by name."""
        r = self._http.delete(
            f"{_BASE}/accounts/{self.account_id}/workers/scripts/{name}"
        )
        r.raise_for_status()

    def close(self) -> None:
        self._http.close()
```

- [ ] **Step 4: Run tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_client.py -v
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/client.py \
        tools/cloudflare/tests/test_client.py
git commit -m "feat(cloudflare): add CF REST API client + tests"
```

---

### Task 6: workers.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/workers.py`
- Create: `tools/cloudflare/tests/test_workers.py`

- [ ] **Step 1: Write failing tests**

Create `tools/cloudflare/tests/test_workers.py`:

```python
"""Unit tests for workers.py — mocked subprocess + httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch, call
import subprocess

from cloudflare_tool.workers import deploy, invoke, _wrangler_env


def test_wrangler_env():
    env = _wrangler_env(account_id="acc123", token="tok_abc")
    assert env["CLOUDFLARE_ACCOUNT_ID"] == "acc123"
    assert env["CLOUDFLARE_API_TOKEN"] == "tok_abc"


def test_deploy_returns_url(monkeypatch):
    mock_run = MagicMock()
    mock_run.return_value = MagicMock(
        returncode=0,
        stdout="Deployed my-worker (https://my-worker.example.workers.dev)\n",
        stderr="",
    )
    monkeypatch.setattr("cloudflare_tool.workers.subprocess.run", mock_run)

    url = deploy(
        account_id="acc123", token="tok_abc",
        name="my-worker", path="app/worker",
        subdomain="example",
    )
    assert url == "https://my-worker.example.workers.dev"
    mock_run.assert_called_once()
    cmd = mock_run.call_args[0][0]
    assert "wrangler" in " ".join(cmd)
    assert "deploy" in cmd


def test_deploy_raises_on_nonzero(monkeypatch):
    mock_run = MagicMock()
    mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="Error: bad config")
    monkeypatch.setattr("cloudflare_tool.workers.subprocess.run", mock_run)

    with pytest.raises(RuntimeError, match="wrangler deploy failed"):
        deploy(account_id="acc123", token="tok_abc", name="w", path=".", subdomain="s")


def test_invoke_get(monkeypatch):
    mock_client = MagicMock()
    resp = MagicMock()
    resp.status_code = 200
    resp.text = '{"ok": true}'
    mock_client.__enter__ = MagicMock(return_value=mock_client)
    mock_client.__exit__ = MagicMock(return_value=False)
    mock_client.request.return_value = resp
    monkeypatch.setattr("cloudflare_tool.workers.httpx.Client", lambda **kw: mock_client)

    status, body = invoke(
        url="https://my-worker.example.workers.dev",
        method="GET", path="/test",
    )
    assert status == 200
    assert body == '{"ok": true}'


def test_invoke_post_with_body(monkeypatch):
    mock_client = MagicMock()
    resp = MagicMock()
    resp.status_code = 201
    resp.text = "created"
    mock_client.__enter__ = MagicMock(return_value=mock_client)
    mock_client.__exit__ = MagicMock(return_value=False)
    mock_client.request.return_value = resp
    monkeypatch.setattr("cloudflare_tool.workers.httpx.Client", lambda **kw: mock_client)

    status, body = invoke(
        url="https://w.example.workers.dev",
        method="POST", path="/data",
        body='{"key": "value"}',
    )
    assert status == 201
    assert body == "created"
    # Verify body was passed
    kwargs = mock_client.request.call_args[1]
    assert kwargs.get("content") == '{"key": "value"}'
```

- [ ] **Step 2: Run failing tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_workers.py -v
```

Expected: FAIL (workers.py doesn't exist).

- [ ] **Step 3: Write `workers.py`**

Create `tools/cloudflare/src/cloudflare_tool/workers.py`:

```python
"""Worker operations: deploy/tail via wrangler subprocess, invoke via httpx."""
from __future__ import annotations

import os
import re
import subprocess
import sys
from typing import Any

import httpx


def _wrangler_env(account_id: str, token: str) -> dict[str, str]:
    """Build environment variables for wrangler subprocess."""
    env = os.environ.copy()
    env["CLOUDFLARE_ACCOUNT_ID"] = account_id
    env["CLOUDFLARE_API_TOKEN"] = token
    return env


def deploy(
    account_id: str,
    token: str,
    name: str,
    path: str,
    subdomain: str = "",
) -> str:
    """Deploy a Worker via wrangler. Returns the public URL."""
    cmd = ["npx", "wrangler", "deploy", "--name", name, path]
    result = subprocess.run(
        cmd,
        env=_wrangler_env(account_id, token),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"wrangler deploy failed (exit {result.returncode}):\n{result.stderr}"
        )

    # Extract URL from wrangler output
    output = result.stdout + result.stderr
    m = re.search(r"https://[\w\-]+\.[\w\-]+\.workers\.dev", output)
    if m:
        return m.group(0)

    # Fallback: construct URL from name + subdomain
    if subdomain:
        return f"https://{name}.{subdomain}.workers.dev"
    return f"https://{name}.workers.dev"


def tail(account_id: str, token: str, name: str) -> None:
    """Stream Worker logs via wrangler tail (runs until interrupted)."""
    cmd = ["npx", "wrangler", "tail", name]
    try:
        subprocess.run(
            cmd,
            env=_wrangler_env(account_id, token),
        )
    except KeyboardInterrupt:
        pass


def invoke(
    url: str,
    method: str = "GET",
    path: str = "/",
    body: str = "",
    headers: dict[str, str] | None = None,
    timeout: float = 30.0,
) -> tuple[int, str]:
    """Send a request to a Worker URL. Returns (status_code, response_body)."""
    full_url = url.rstrip("/") + (path if path.startswith("/") else f"/{path}")
    req_headers = headers or {}

    with httpx.Client(timeout=timeout) as client:
        resp = client.request(
            method=method.upper(),
            url=full_url,
            headers=req_headers,
            content=body if body else None,
        )

    return resp.status_code, resp.text
```

- [ ] **Step 4: Run tests**

```bash
cd tools/cloudflare && uv run pytest tests/test_workers.py -v
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/workers.py \
        tools/cloudflare/tests/test_workers.py
git commit -m "feat(cloudflare): add workers deploy/tail/invoke + tests"
```

---

## Chunk 3: Browser automation

### Task 7: browser.py — register_via_browser

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/browser.py`

No unit tests for browser.py (same policy as motherduck/clickhouse — tested via integration).

- [ ] **Step 1: Write `browser.py`**

Create `tools/cloudflare/src/cloudflare_tool/browser.py`:

```python
"""Patchright browser automation for Cloudflare signup and API token creation.

Registration flow:
  1. dash.cloudflare.com/sign-up → fill email + password → submit
  2. Poll mail.tm for verification email → click link
  3. Skip domain setup → accept Free plan
  4. Extract account_id from URL or API

Token creation flow:
  1. Login to dash.cloudflare.com
  2. Navigate to /profile/api-tokens → Create Token → Custom Token
  3. Set name + permissions from preset → submit
  4. Extract token value from confirmation page
"""
from __future__ import annotations

import os
import platform
import re
import tempfile
import time

from .email import MailTmClient, Mailbox


# ---------------------------------------------------------------------------
# Permission presets
# ---------------------------------------------------------------------------

# Maps preset name → list of (resource_type, resource, permission) tuples
# These map to CF's token permission UI labels
PRESETS: dict[str, list[tuple[str, str, str]]] = {
    "browser-rendering": [
        ("Account", "Browser Rendering", "Edit"),
    ],
    "workers": [
        ("Account", "Workers Scripts", "Edit"),
        ("Account", "Workers Routes", "Edit"),
    ],
    "r2": [
        ("Account", "R2 Storage", "Edit"),
    ],
    "kv": [
        ("Account", "Workers KV Storage", "Edit"),
    ],
    "dns": [
        ("Zone", "DNS", "Edit"),
    ],
    "all": [
        ("Account", "Browser Rendering", "Edit"),
        ("Account", "Workers Scripts", "Edit"),
        ("Account", "Workers Routes", "Edit"),
        ("Account", "R2 Storage", "Edit"),
        ("Account", "Workers KV Storage", "Edit"),
        ("Zone", "DNS", "Edit"),
    ],
}


# ---------------------------------------------------------------------------
# Browser helpers
# ---------------------------------------------------------------------------

def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += [
            "--no-sandbox", "--disable-setuid-sandbox",
            "--disable-dev-shm-usage", "--disable-gpu",
        ]
    return args


def _ensure_display() -> None:
    if platform.system() != "Linux" or os.environ.get("DISPLAY"):
        return
    import shutil, subprocess
    xvfb = shutil.which("Xvfb")
    if xvfb:
        display = ":99"
        proc = subprocess.Popen(
            [xvfb, display, "-screen", "0", "1280x900x24"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        import atexit
        atexit.register(proc.kill)
        time.sleep(0.5)
        os.environ["DISPLAY"] = display


def _wait(seconds: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {seconds}s ({msg})...")
    time.sleep(seconds)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=10000)
    el.click()
    time.sleep(0.3)
    el.type(text, delay=delay)
    time.sleep(0.4)


def _click_first(page, selectors: list[str], log=None) -> str | None:
    for sel in selectors:
        try:
            btn = page.locator(sel)
            if btn.count() > 0:
                btn.first.click()
                if log:
                    log(f"clicked: {sel}")
                return sel
        except Exception:
            continue
    return None


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            inp = page.locator(sel)
            if inp.count() > 0:
                _fill(page, sel, text)
                if log:
                    log(f"filled via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _log_page(page, log, label: str = "", max_chars: int = 500) -> str:
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:300]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""


def _on_dash(page) -> bool:
    return "dash.cloudflare.com" in page.url


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive Cloudflare signup. Returns account_id string."""
    from patchright.sync_api import sync_playwright
    import shutil

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="cf_reg_")
    channel = "chrome" if shutil.which("google-chrome") or shutil.which("google-chrome-stable") else None

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel=channel,
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Sign-up form ----
            log("opening dash.cloudflare.com/sign-up...")
            try:
                page.goto("https://dash.cloudflare.com/sign-up", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception as e:
                log(f"nav warn: {e}")
            _wait(2, log)
            log(f"url: {page.url}")

            # Fill email
            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
            ], mailbox.address, log)

            # Fill password
            _wait(0.5, log)
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)

            # Confirm password (CF sign-up has confirm field)
            _wait(0.3, log)
            confirm_inputs = page.locator('input[type="password"]')
            if confirm_inputs.count() >= 2:
                log("filling confirm password...")
                confirm_inputs.nth(1).click()
                time.sleep(0.2)
                confirm_inputs.nth(1).type(password, delay=55)
                time.sleep(0.3)

            # Submit
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Create account")',
                'button:has-text("Continue")',
                'input[type="submit"]',
            ], log)
            _wait(4, log, "waiting for signup response")
            log(f"url after submit: {page.url}")

            # ---- Step 2: Email verification ----
            body_text = _log_page(page, log, "post-signup: ")
            verify_keywords = [
                "verify", "check your email", "confirmation",
                "we sent", "email sent", "verification",
            ]
            if any(kw in body_text.lower() for kw in verify_keywords):
                log("email verification required — polling mail.tm...")
                verify_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
                log(f"got verification link: {verify_link[:60]}...")
                try:
                    page.goto(verify_link, timeout=30000)
                    page.wait_for_load_state("networkidle", timeout=15000)
                except Exception as e:
                    log(f"verification nav warn: {e}")
                _wait(4, log, "post-verification")
                log(f"url after verification: {page.url}")

            # ---- Step 3: Onboarding — skip domain setup ----
            log("completing onboarding...")
            _skip_onboarding(page, log, max_attempts=15)

            # ---- Step 4: Extract account_id ----
            log(f"url before account_id extraction: {page.url}")
            account_id = _extract_account_id(page, log)

            if not account_id:
                # Try navigating to dashboard home and extracting from URL
                try:
                    page.goto("https://dash.cloudflare.com/", timeout=20000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception as e:
                    log(f"dashboard nav warn: {e}")
                _wait(3, log)
                log(f"url: {page.url}")
                account_id = _extract_account_id(page, log)

            if not account_id:
                _log_page(page, log, "account_id-fail: ")
                raise RuntimeError(
                    "Failed to extract Cloudflare account_id. "
                    f"Current URL: {page.url}"
                )

            log(f"account_id: {account_id}")
            return account_id

        finally:
            ctx.close()


def _skip_onboarding(page, log, max_attempts: int = 15) -> None:
    """Click through Cloudflare onboarding: skip domain setup, accept Free plan."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url
        log(f"  onboarding attempt {attempt}: {url}")

        # Done if on dashboard home or account page
        if re.search(r"/[0-9a-f]{32}(/|$)", url) or "/home" in url:
            log(f"  onboarding done at attempt {attempt}")
            return

        # Skip domain / add domain later
        skip_clicked = _click_first(page, [
            'button:has-text("Skip")',
            'a:has-text("Skip")',
            'button:has-text("Add later")',
            'a:has-text("Add later")',
            'button:has-text("Skip for now")',
            'a:has-text("Skip for now")',
            '[data-testid*="skip"]',
        ], log)

        if not skip_clicked:
            # Try "Continue" / "Next" / "Get started"
            skip_clicked = _click_first(page, [
                'button:has-text("Continue")',
                'button:has-text("Next")',
                'button:has-text("Get started")',
                'button:has-text("Done")',
                'button:has-text("Finish")',
                'a:has-text("Continue")',
            ], log)

        if not skip_clicked:
            log(f"  no onboarding button found at attempt {attempt}")
            # Check if a plan selection is needed
            free_plan = _click_first(page, [
                'button:has-text("Free")',
                'a:has-text("Free")',
                '[data-testid*="free"]',
                'button:has-text("Select Free")',
            ], log)
            if not free_plan:
                log("  no clickable element, stopping onboarding")
                break


def _extract_account_id(page, log) -> str:
    """Extract account_id from current page URL or page content."""
    url = page.url

    # Strategy 1: URL path contains 32-char hex account_id
    # e.g. dash.cloudflare.com/abc123.../home
    m = re.search(r"/([0-9a-f]{32})(/|$)", url)
    if m:
        log(f"account_id from URL: {m.group(1)}")
        return m.group(1)

    # Strategy 2: Page HTML/text contains account_id pattern
    try:
        html = page.evaluate("document.documentElement.innerHTML")
        m = re.search(r'"account_id"\s*:\s*"([0-9a-f]{32})"', html)
        if m:
            log(f"account_id from HTML: {m.group(1)}")
            return m.group(1)
        # Try data-account-id attribute
        m = re.search(r'data-account-id="([0-9a-f]{32})"', html)
        if m:
            log(f"account_id from data attr: {m.group(1)}")
            return m.group(1)
    except Exception as e:
        log(f"HTML extraction error: {e}")

    return ""


# ---------------------------------------------------------------------------
# Token creation
# ---------------------------------------------------------------------------

def create_token_via_browser(
    email: str,
    password: str,
    token_name: str,
    preset: str = "all",
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Login to CF dashboard and create a named API token. Returns token value."""
    from patchright.sync_api import sync_playwright
    import shutil

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    permissions = PRESETS.get(preset, PRESETS["all"])
    log(f"creating token '{token_name}' with preset '{preset}' ({len(permissions)} permissions)")

    user_data = tempfile.mkdtemp(prefix="cf_tok_")
    channel = "chrome" if shutil.which("google-chrome") or shutil.which("google-chrome-stable") else None

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel=channel,
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Login ----
            log("logging in...")
            try:
                page.goto("https://dash.cloudflare.com/login", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception as e:
                log(f"login nav warn: {e}")
            _wait(2, log)

            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
            ], email, log)
            _wait(0.5, log)
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Log in")',
                'button:has-text("Sign in")',
                'button:has-text("Continue")',
            ], log)
            _wait(5, log, "waiting for login")
            log(f"url after login: {page.url}")

            # Check for 2FA / unusual login prompts
            body = _log_page(page, log, "post-login: ")

            # ---- Step 2: Navigate to API Tokens ----
            log("navigating to API tokens...")
            try:
                page.goto(
                    "https://dash.cloudflare.com/profile/api-tokens",
                    timeout=20000,
                )
                page.wait_for_load_state("networkidle", timeout=10000)
            except Exception as e:
                log(f"api-tokens nav warn: {e}")
            _wait(3, log)
            log(f"url: {page.url}")

            if "login" in page.url.lower() or "sign-in" in page.url.lower():
                raise RuntimeError(
                    f"Not logged in — redirected to login page. URL: {page.url}"
                )

            # ---- Step 3: Create Custom Token ----
            log("clicking 'Create Token'...")
            _click_first(page, [
                'button:has-text("Create Token")',
                'a:has-text("Create Token")',
                '[data-testid*="create-token"]',
            ], log)
            _wait(2, log)

            # Select "Custom Token" (vs templates)
            log("selecting 'Custom Token'...")
            _click_first(page, [
                'button:has-text("Get started"):near(:text("Custom token"))',
                'a:has-text("Get started"):near(:text("Custom token"))',
                '[data-testid*="custom"]',
                'button:has-text("Get started")',
                'a:has-text("Get started")',
            ], log)
            _wait(2, log)
            log(f"url: {page.url}")

            # ---- Step 4: Fill token name ----
            log(f"filling token name: {token_name}")
            _fill_first(page, [
                'input[name*="token" i]',
                'input[placeholder*="token" i]',
                'input[placeholder*="name" i]',
                'input[type="text"]:first-of-type',
                'input[aria-label*="name" i]',
            ], token_name, log)
            _wait(0.5, log)

            # ---- Step 5: Add permissions ----
            log(f"adding {len(permissions)} permissions...")
            _add_token_permissions(page, permissions, log)

            # ---- Step 6: Submit ----
            _wait(1, log)
            log("submitting token creation form...")
            _click_first(page, [
                'button:has-text("Continue to summary")',
                'button:has-text("Continue")',
                'button[type="submit"]:has-text("Continue")',
            ], log)
            _wait(2, log)
            _log_page(page, log, "summary: ")

            _click_first(page, [
                'button:has-text("Create Token")',
                'button[type="submit"]:has-text("Create")',
                'button:has-text("Confirm")',
            ], log)
            _wait(3, log, "token creation")
            log(f"url after create: {page.url}")

            # ---- Step 7: Extract token value ----
            log("extracting token value...")
            token_value = _extract_token_value(page, log)

            if not token_value:
                for retry in range(5):
                    _wait(3, log, f"token retry {retry + 1}/5")
                    token_value = _extract_token_value(page, log)
                    if token_value:
                        break

            if not token_value:
                _log_page(page, log, "token-fail: ")
                raise RuntimeError(
                    "Failed to extract token value from confirmation page"
                )

            log(f"token extracted (len={len(token_value)})")
            return token_value

        finally:
            ctx.close()


def _add_token_permissions(page, permissions: list[tuple[str, str, str]], log) -> None:
    """Add permission rows to the CF Custom Token creation form."""
    for i, (resource_type, resource, permission) in enumerate(permissions):
        log(f"  adding permission: {resource_type} > {resource} > {permission}")

        # Click "Add more" or "+" button for rows after the first
        if i > 0:
            _click_first(page, [
                'button:has-text("Add more")',
                'button:has-text("Add permission")',
                'button[aria-label*="add" i]',
                'button:has-text("+")',
            ], log)
            _wait(0.5, log)

        # Each permission row has two dropdowns: category and level
        # The new row is typically the last row in the permissions table
        rows = page.locator('[data-testid*="permission-row"], .permission-row, tr:has(select)')
        row = rows.last if rows.count() > 0 else page

        # Select resource type (Account/Zone)
        type_selects = row.locator('select, [role="combobox"]')
        if type_selects.count() >= 1:
            try:
                type_selects.first.select_option(label=resource_type)
                _wait(0.3, log)
            except Exception:
                # Try clicking and selecting from dropdown
                type_selects.first.click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{resource_type}")'], log)

        # Select specific resource (e.g., "Workers Scripts")
        if type_selects.count() >= 2:
            try:
                type_selects.nth(1).select_option(label=resource)
                _wait(0.3, log)
            except Exception:
                type_selects.nth(1).click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{resource}")'], log)

        # Select permission level (Edit/Read)
        if type_selects.count() >= 3:
            try:
                type_selects.nth(2).select_option(label=permission)
                _wait(0.3, log)
            except Exception:
                type_selects.nth(2).click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{permission}")'], log)


def _extract_token_value(page, log) -> str:
    """Extract the API token value from CF confirmation page."""
    # Strategy 1: specific CF token display elements
    for sel in [
        '[data-testid*="token-value"]',
        '[data-testid*="copy-token"]',
        'input[readonly][type="text"]',
        'code',
        'pre',
        '.copy-input input',
        '[aria-label*="token" i] input',
    ]:
        try:
            el = page.locator(sel).first
            if el.count() > 0:
                text = el.get_attribute("value") or el.inner_text() or ""
                text = text.strip()
                if len(text) > 20 and " " not in text:
                    log(f"token via {sel!r}: {text[:20]}...")
                    return text
        except Exception:
            pass

    # Strategy 2: page text regex for CF token patterns
    # CF tokens look like: abc123_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
    try:
        body = page.inner_text("body")
        for pat in [
            r"[A-Za-z0-9_\-]{40,}",          # generic long token
            r"[a-zA-Z0-9]{8}_[a-zA-Z0-9_\-]{30,}",  # CF token format
        ]:
            m = re.search(pat, body)
            if m:
                candidate = m.group(0)
                if len(candidate) > 30:
                    log(f"token via body regex: {candidate[:20]}...")
                    return candidate
    except Exception:
        pass

    # Strategy 3: localStorage
    try:
        keys = page.evaluate("() => Object.keys(localStorage)")
        for key in keys:
            val = page.evaluate(f"() => localStorage.getItem('{key}')")
            if val and len(val) > 30 and " " not in val:
                log(f"token via localStorage[{key!r}]")
                return val
    except Exception:
        pass

    return ""
```

- [ ] **Step 2: Verify syntax**

```bash
cd tools/cloudflare && uv run python -c "from cloudflare_tool.browser import register_via_browser, create_token_via_browser, PRESETS; print('OK', list(PRESETS.keys()))"
```

Expected: `OK ['browser-rendering', 'workers', 'r2', 'kv', 'dns', 'all']`

- [ ] **Step 3: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/browser.py
git commit -m "feat(cloudflare): add browser automation for signup and token creation"
```

---

## Chunk 4: CLI + README + integration

### Task 8: cli.py

**Files:**
- Create: `tools/cloudflare/src/cloudflare_tool/cli.py`

- [ ] **Step 1: Write `cli.py`**

Create `tools/cloudflare/src/cloudflare_tool/cli.py`:

```python
"""Typer CLI: cloudflare register / account / token / worker."""
from __future__ import annotations

import json
import os
import time as _time
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="cloudflare-tool",
    help="Manage Cloudflare accounts, API tokens, and Workers.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
token_app = typer.Typer(help="Manage API tokens.", no_args_is_help=True)
worker_app = typer.Typer(help="Manage Workers.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(token_app, name="token")
app.add_typer(worker_app, name="worker")

console = Console()
err_console = Console(stderr=True)

_CF_JSON = Path.home() / "data" / "cloudflare" / "cloudflare.json"


def _store() -> Store:
    return Store(DEFAULT_DB_PATH)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    no_headless: Annotated[bool, typer.Option("--no-headless")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
    json_out: Annotated[bool, typer.Option("--json")] = False,
) -> None:
    """Auto-register a new Cloudflare account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    status_console = Console(stderr=True) if json_out else console

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with status_console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    status_console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    status_console.print("[bold green]Opening browser for Cloudflare signup...[/bold green]")

    try:
        account_id = register_via_browser(
            mailbox=mailbox,
            mail_client=mail_client,
            password=identity.password,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Registration failed:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        mail_client.close()

    if json_out:
        print(json.dumps({
            "email": mailbox.address,
            "password": identity.password,
            "account_id": account_id,
        }))
        return

    store = _store()
    store.add_account(
        email=mailbox.address,
        password=identity.password,
        account_id=account_id,
    )

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    console.print(f"[dim]Account ID:[/dim] {account_id}")
    console.print(f"[dim]Stored in:[/dim] {DEFAULT_DB_PATH}")


# ---------------------------------------------------------------------------
# account
# ---------------------------------------------------------------------------

@account_app.command("ls")
def account_ls() -> None:
    """List all accounts."""
    store = _store()
    rows = store.list_accounts()
    if not rows:
        console.print("[yellow]No accounts registered.[/yellow]")
        return

    table = Table(title="Accounts", show_lines=True)
    table.add_column("Email", style="cyan")
    table.add_column("Account ID")
    table.add_column("Tokens", justify="right")
    table.add_column("Workers", justify="right")
    table.add_column("Active", justify="center")
    table.add_column("Created")

    for r in rows:
        active = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        acc_short = r["account_id"][:16] + "..." if len(r["account_id"]) > 16 else r["account_id"]
        table.add_row(
            r["email"], acc_short,
            str(r["token_count"]), str(r["worker_count"]),
            active, created,
        )
    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument()],
) -> None:
    """Deactivate an account (local only)."""
    store = _store()
    if not store.get_account_by_email(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.deactivate_account(email)
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# token
# ---------------------------------------------------------------------------

@token_app.command("create")
def token_create(
    name: Annotated[str, typer.Argument(help="Token name")],
    preset: Annotated[str, typer.Option("--preset", help="Permission preset")] = "all",
    account: Annotated[Optional[str], typer.Option("--account")] = None,
    no_headless: Annotated[bool, typer.Option("--no-headless")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
    set_default: Annotated[bool, typer.Option("--default")] = False,
) -> None:
    """Create a named API token via CF dashboard browser automation."""
    from .browser import create_token_via_browser, PRESETS

    if preset not in PRESETS:
        err_console.print(
            f"[bold red]Unknown preset:[/bold red] {preset}. "
            f"Choose: {', '.join(PRESETS)}"
        )
        raise typer.Exit(1)

    store = _store()
    if account:
        acc = store.get_account_by_email(account)
        if not acc:
            err_console.print(f"[bold red]Account not found:[/bold red] {account}")
            raise typer.Exit(1)
    else:
        acc = store.get_first_active_account()
        if not acc:
            err_console.print("[bold red]No active accounts. Run:[/bold red] cloudflare-tool register")
            raise typer.Exit(1)

    console.print(f"[bold green]Creating token '{name}' (preset: {preset})...[/bold green]")
    console.print("[dim]Opening browser to Cloudflare API Tokens page...[/dim]")

    try:
        token_value = create_token_via_browser(
            email=acc["email"],
            password=acc["password"],
            token_name=name,
            preset=preset,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Token creation failed:[/bold red] {e}")
        raise typer.Exit(1)

    store.add_token(
        account_id=acc["id"],
        name=name,
        token_value=token_value,
        preset=preset,
    )
    if set_default:
        store.set_default_token(name)
        _write_cf_json(store, name)

    console.print(f"[bold green]✓ Token created:[/bold green] {name}")
    console.print(f"[dim]Preset:[/dim] {preset}")
    if set_default:
        console.print(f"[green]Set as default. cloudflare.json updated.[/green]")


@token_app.command("ls")
def token_ls() -> None:
    """List all tokens."""
    store = _store()
    rows = store.list_tokens()
    if not rows:
        console.print("[yellow]No tokens. Run:[/yellow] cloudflare-tool token create <name>")
        return

    table = Table(title="API Tokens", show_lines=True)
    table.add_column("Name", style="cyan")
    table.add_column("Preset")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Created")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        name_str = f"[bold]{r['name']}[/bold]" if r["is_default"] else r["name"]
        table.add_row(name_str, r["preset"], r["email"], default, created)
    console.print(table)


@token_app.command("rm")
def token_rm(
    name: Annotated[str, typer.Argument()],
) -> None:
    """Remove a token from local state."""
    store = _store()
    if not store.get_token_by_name(name):
        err_console.print(f"[bold red]Token not found:[/bold red] {name}")
        raise typer.Exit(1)
    store.remove_token(name)
    console.print(f"[yellow]Removed:[/yellow] {name} [dim](local only)[/dim]")


@token_app.command("use")
def token_use(
    name: Annotated[str, typer.Argument()],
) -> None:
    """Set default token and write ~/data/cloudflare/cloudflare.json."""
    store = _store()
    tok = store.get_token_by_name(name)
    if not tok:
        err_console.print(f"[bold red]Token not found:[/bold red] {name}")
        raise typer.Exit(1)
    store.set_default_token(name)
    _write_cf_json(store, name)
    console.print(f"[green]Default set to:[/green] {name}")
    console.print(f"[dim]cloudflare.json written to:[/dim] {_CF_JSON}")


def _write_cf_json(store: Store, token_name: str) -> None:
    """Write ~/data/cloudflare/cloudflare.json for pkg/scrape compatibility."""
    tok = store.get_token_by_name(token_name)
    if not tok:
        return
    _CF_JSON.parent.mkdir(parents=True, exist_ok=True)
    _CF_JSON.write_text(json.dumps({
        "account_id": tok["account_id"],
        "api_token": tok["token_value"],
    }, indent=2))


# ---------------------------------------------------------------------------
# worker
# ---------------------------------------------------------------------------

@worker_app.command("deploy")
def worker_deploy(
    path: Annotated[str, typer.Argument(help="Path to Worker source or wrangler.toml dir")],
    name: Annotated[Optional[str], typer.Option("--name")] = None,
    alias: Annotated[Optional[str], typer.Option("--alias")] = None,
    token_name: Annotated[Optional[str], typer.Option("--token")] = None,
    set_default: Annotated[bool, typer.Option("--default")] = False,
) -> None:
    """Deploy a Worker via wrangler."""
    from .workers import deploy as _deploy

    # Derive name from path if not provided
    worker_name = name or Path(path).resolve().name
    worker_alias = alias or worker_name

    store = _store()

    # Resolve token
    if token_name:
        tok = store.get_token_by_name(token_name)
        if not tok:
            err_console.print(f"[bold red]Token not found:[/bold red] {token_name}")
            raise typer.Exit(1)
    else:
        tok = store.get_default_token()
        if not tok:
            err_console.print(
                "[bold red]No default token. Run:[/bold red] cloudflare-tool token use <name>"
            )
            raise typer.Exit(1)

    console.print(f"[bold green]Deploying '{worker_name}' from {path}...[/bold green]")

    t0 = _time.monotonic()
    try:
        url = _deploy(
            account_id=tok["account_id"],
            token=tok["token_value"],
            name=worker_name,
            path=path,
            subdomain=tok.get("subdomain", ""),
        )
    except Exception as e:
        err_console.print(f"[bold red]Deploy failed:[/bold red] {e}")
        raise typer.Exit(1)

    duration_ms = int((_time.monotonic() - t0) * 1000)

    # Store or update worker
    existing = store.get_worker(worker_alias)
    if existing:
        store.update_worker_url(worker_alias, url)
        w_id = existing["id"]
    else:
        # Get token DB id
        tok_row = store.get_token_by_name(tok["name"])
        w_id = store.add_worker(
            account_id=tok["account_db_id"],
            token_id=tok_row["id"] if tok_row else None,
            name=worker_name,
            alias=worker_alias,
            url=url,
        )

    store.log_op(worker_id=w_id, operation="deploy", detail=path, duration_ms=duration_ms)

    if set_default:
        store.set_default_worker(worker_alias)

    console.print(f"[bold green]✓ Deployed:[/bold green] {worker_name}")
    console.print(f"[dim]URL:[/dim] {url}")
    console.print(f"[dim]Alias:[/dim] {worker_alias}")
    if set_default:
        console.print("[green]Set as default.[/green]")


@worker_app.command("ls")
def worker_ls() -> None:
    """List all Workers."""
    store = _store()
    rows = store.list_workers()
    if not rows:
        console.print("[yellow]No workers. Run:[/yellow] cloudflare-tool worker deploy <path>")
        return

    table = Table(title="Workers", show_lines=True)
    table.add_column("Alias", style="cyan")
    table.add_column("Name")
    table.add_column("URL")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Ops", justify="right")
    table.add_column("Deployed")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        deployed = str(r["deployed_at"])[:16] if r["deployed_at"] else "-"
        url_short = r["url"][:40] + "..." if len(r.get("url", "")) > 40 else r.get("url", "")
        alias_str = f"[bold]{r['alias']}[/bold]" if r["is_default"] else r["alias"]
        table.add_row(
            alias_str, r["name"], url_short, r["email"],
            default, str(r["op_count"]), deployed,
        )
    console.print(table)


@worker_app.command("rm")
def worker_rm(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Delete a Worker from Cloudflare and remove from local state."""
    from .client import CloudflareClient

    store = _store()
    w = store.get_worker(alias)
    if not w:
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)

    tok = store.get_default_token()
    if tok:
        try:
            client = CloudflareClient(
                account_id=tok["account_id"],
                api_token=tok["token_value"],
            )
            client.delete_worker(w["name"])
            client.close()
            console.print(f"[dim]Deleted from Cloudflare:[/dim] {w['name']}")
        except Exception as e:
            err_console.print(f"[yellow]CF delete warning:[/yellow] {e}")

    store.remove_worker(alias)
    console.print(f"[yellow]Removed:[/yellow] {alias}")


@worker_app.command("tail")
def worker_tail(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Stream real-time Worker logs via wrangler tail."""
    from .workers import tail as _tail

    store = _store()
    w = store.get_worker(alias)
    if not w:
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)

    tok = store.get_default_token()
    if not tok:
        err_console.print(
            "[bold red]No default token. Run:[/bold red] cloudflare-tool token use <name>"
        )
        raise typer.Exit(1)

    console.print(f"[bold green]Tailing '{w['name']}' (Ctrl+C to stop)...[/bold green]")
    _tail(
        account_id=tok["account_id"],
        token=tok["token_value"],
        name=w["name"],
    )


@worker_app.command("invoke")
def worker_invoke(
    alias: Annotated[str, typer.Argument()],
    path: Annotated[str, typer.Option("--path")] = "/",
    method: Annotated[str, typer.Option("--method")] = "GET",
    body: Annotated[Optional[str], typer.Option("--body")] = None,
    header: Annotated[Optional[list[str]], typer.Option("--header")] = None,
    json_out: Annotated[bool, typer.Option("--json")] = False,
) -> None:
    """Send an HTTP request to a Worker."""
    from .workers import invoke as _invoke

    store = _store()
    w = store.get_worker(alias)
    if not w:
        # Try default worker
        w = store.get_default_worker()
        if not w:
            err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
            raise typer.Exit(1)

    if not w.get("url"):
        err_console.print(f"[bold red]Worker has no URL:[/bold red] {alias}")
        raise typer.Exit(1)

    headers: dict[str, str] = {}
    for h in (header or []):
        if ":" in h:
            k, v = h.split(":", 1)
            headers[k.strip()] = v.strip()

    t0 = _time.monotonic()
    status, response_body = _invoke(
        url=w["url"], method=method,
        path=path, body=body or "",
        headers=headers,
    )
    duration_ms = int((_time.monotonic() - t0) * 1000)

    store.log_op(
        worker_id=w["id"], operation="invoke",
        detail=f"{method} {path} → {status}",
        duration_ms=duration_ms,
    )

    if json_out:
        print(json.dumps({
            "status": status, "body": response_body, "duration_ms": duration_ms
        }))
        return

    color = "green" if status < 400 else "red"
    from rich.panel import Panel
    console.print(Panel(
        response_body,
        title=f"[{color}]{status} {method} {path}[/{color}]  "
              f"[dim]{duration_ms}ms[/dim]",
        border_style=color,
    ))


@worker_app.command("use")
def worker_use(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Set the default Worker for invoke/tail."""
    store = _store()
    if not store.get_worker(alias):
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)
    store.set_default_worker(alias)
    console.print(f"[green]Default worker set to:[/green] {alias}")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()


if __name__ == "__main__":
    app_entry()
```

- [ ] **Step 2: Verify imports**

```bash
cd tools/cloudflare && uv run python -c "from cloudflare_tool.cli import app_entry; print('OK')"
```

Expected: `OK`

- [ ] **Step 3: Smoke-test help output**

```bash
cd tools/cloudflare && uv run cloudflare-tool --help
cd tools/cloudflare && uv run cloudflare-tool token --help
cd tools/cloudflare && uv run cloudflare-tool worker --help
```

Expected: help text printed for each, no errors.

- [ ] **Step 4: Commit**

```bash
git add tools/cloudflare/src/cloudflare_tool/cli.py
git commit -m "feat(cloudflare): add full Typer CLI (register/account/token/worker)"
```

---

### Task 9: README.md + ARCHITECTURE.md

**Files:**
- Create: `tools/cloudflare/README.md`
- Create: `tools/cloudflare/ARCHITECTURE.md`

- [ ] **Step 1: Write README.md**

Create `tools/cloudflare/README.md`:

````markdown
# cloudflare-tool

Auto-register Cloudflare accounts, create API tokens, and manage Workers — all from the command line.

## Install

Requires Python 3.11+ and [uv](https://docs.astral.sh/uv/).

```bash
cd blueprints/search/tools/cloudflare
uv sync
uv run patchright install chromium
```

## Quick Start

```bash
# Register a new account
uv run cloudflare-tool register

# Create a Browser Rendering token (browser opens CF dashboard)
uv run cloudflare-tool token create my-br-token --preset browser-rendering --default

# Write ~/data/cloudflare/cloudflare.json for pkg/scrape
uv run cloudflare-tool token use my-br-token

# Deploy a Worker
uv run cloudflare-tool worker deploy app/worker --name my-worker --default

# Invoke a Worker
uv run cloudflare-tool worker invoke my-worker --path /search?q=test

# Tail Worker logs
uv run cloudflare-tool worker tail my-worker
```

## Commands

### register

```bash
uv run cloudflare-tool register [--no-headless] [--verbose] [--json]
```

Auto-registers a new Cloudflare account end-to-end:
1. Generates random identity via Faker + creates disposable mail.tm mailbox
2. Opens `dash.cloudflare.com/sign-up`, fills form, submits
3. Polls mail.tm for verification email, clicks link
4. Completes onboarding (skips domain, accepts Free plan)
5. Extracts `account_id` from dashboard URL
6. Stores in DuckDB

### account ls / rm

```bash
uv run cloudflare-tool account ls
uv run cloudflare-tool account rm <email>
```

### token create / ls / rm / use

```bash
uv run cloudflare-tool token create <name> [--preset browser-rendering|workers|r2|kv|dns|all] [--default]
uv run cloudflare-tool token ls
uv run cloudflare-tool token rm <name>
uv run cloudflare-tool token use <name>   # also writes cloudflare.json
```

**Presets:**
| Preset | Permissions |
|---|---|
| `browser-rendering` | Account > Browser Rendering > Edit |
| `workers` | Workers Scripts + Routes > Edit |
| `r2` | R2 Storage > Edit |
| `kv` | Workers KV Storage > Edit |
| `dns` | Zone > DNS > Edit |
| `all` | All of the above |

### worker deploy / ls / rm / tail / invoke / use

```bash
uv run cloudflare-tool worker deploy <path> [--name <name>] [--alias <alias>] [--default]
uv run cloudflare-tool worker ls
uv run cloudflare-tool worker rm <alias>
uv run cloudflare-tool worker tail <alias>
uv run cloudflare-tool worker invoke <alias> [--path /] [--method GET] [--body <json>] [--json]
uv run cloudflare-tool worker use <alias>
```

## Data Storage

All state in `~/data/cloudflare/cloudflare.duckdb`. Four tables:
- `accounts` — email, password, account_id, subdomain
- `tokens` — name, token_value, preset, is_default
- `workers` — name, alias, url, is_default
- `op_log` — operation audit trail

`token use <name>` also writes `~/data/cloudflare/cloudflare.json` for compatibility
with `search scrape --cloudflare` (`pkg/scrape/cloudflare.go`).

## Running Tests

```bash
uv run pytest tests/ -v
```

## Troubleshooting

**Registration fails at email verification**: mail.tm may be down. Retry after a minute.

**Token value not extracted**: CF shows the token only once. Run `--no-headless --verbose`.

**wrangler not found**: Ensure Node.js/npm installed. Tool uses `npx wrangler` (auto-downloads).

**subdomain empty**: Run `token use <name>` to trigger subdomain backfill via CF API.
````

- [ ] **Step 2: Write ARCHITECTURE.md**

Create `tools/cloudflare/ARCHITECTURE.md`:

````markdown
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

### cli.py (~530 lines)
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
````

- [ ] **Step 3: Commit**

```bash
git add tools/cloudflare/README.md tools/cloudflare/ARCHITECTURE.md
git commit -m "docs(cloudflare): add README and ARCHITECTURE"
```

---

### Task 10: Run all tests + final verification

**Files:** No new files.

- [ ] **Step 1: Run full test suite**

```bash
cd tools/cloudflare && uv run pytest tests/ -v
```

Expected: All tests PASS. Collect at minimum:
- `test_store.py` — 12+ tests
- `test_client.py` — 5 tests
- `test_email.py` — 2 tests
- `test_workers.py` — 5 tests

- [ ] **Step 2: Verify CLI entry points**

```bash
cd tools/cloudflare && uv run cloudflare-tool --help
uv run cloudflare-tool account --help
uv run cloudflare-tool token --help
uv run cloudflare-tool worker --help
```

- [ ] **Step 3: Verify store creates file**

```bash
cd tools/cloudflare && uv run python -c "
from cloudflare_tool.store import Store
s = Store('/tmp/test_cf.duckdb')
acc_id = s.add_account(email='t@x.com', password='pw', account_id='acc123')
tok_id = s.add_token(account_id=acc_id, name='test-tok', token_value='v', preset='all')
s.set_default_token('test-tok')
print('accounts:', s.list_accounts())
print('tokens:', s.list_tokens())
print('default token:', s.get_default_token()['name'])
import os; os.unlink('/tmp/test_cf.duckdb')
print('OK')
"
```

Expected: prints accounts, tokens, default token name, `OK`.

- [ ] **Step 4: Final commit**

```bash
git add tools/cloudflare/
git commit -m "test(cloudflare): verify all tests pass"
```

---

## Summary

| Task | Files | Tests |
|------|-------|-------|
| 1. Scaffold | pyproject.toml, Makefile | — |
| 2. identity.py | identity.py | manual |
| 3. email.py | email.py | test_email.py |
| 4. store.py | store.py | test_store.py |
| 5. client.py | client.py | test_client.py |
| 6. workers.py | workers.py | test_workers.py |
| 7. browser.py | browser.py | integration only |
| 8. cli.py | cli.py | smoke tests |
| 9. Docs | README.md, ARCHITECTURE.md | — |
| 10. Final | — | full suite |

Total: ~24 source files across 4 modules. All unit-testable code covered.
Browser automation tested via `register --no-headless --verbose` integration run.
