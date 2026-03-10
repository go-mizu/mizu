# MotherDuck Tool Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `tools/motherduck` — a UV+Python CLI that auto-registers MotherDuck accounts via Patchright+mail.tm, stores state in local DuckDB, and runs queries against MotherDuck cloud.

**Architecture:** Typer CLI with Docker-style grouped subcommands (`db create`, `db ls`, `db use`, `db rm`, `account ls`, `account rm`, top-level `register` and `query`). State persisted in `$HOME/data/motherduck/mother.duckdb` (three tables: accounts, databases, query_log). Patchright drives browser signup; mail.tm API delivers the magic link; `duckdb` Python SDK connects to MotherDuck cloud via `md:?motherduck_token=...` DSN.

**Tech Stack:** Python 3.11+, UV, Typer 0.12+, Rich 13+, Patchright 1.50+, duckdb 1.2+, httpx, Faker

**Spec:** `docs/superpowers/specs/2026-03-10-motherduck-design.md`

---

## File Map

| File | Responsibility |
|---|---|
| `tools/motherduck/pyproject.toml` | UV package config, deps, `motherduck` entry point |
| `tools/motherduck/src/motherduck/__init__.py` | Empty package marker |
| `tools/motherduck/src/motherduck/store.py` | Local DuckDB schema init + CRUD (accounts, databases, query_log) |
| `tools/motherduck/src/motherduck/identity.py` | Faker identity generation (display_name, password, email local) |
| `tools/motherduck/src/motherduck/email.py` | mail.tm API: create mailbox, poll for magic link URL |
| `tools/motherduck/src/motherduck/browser.py` | Patchright: drive app.motherduck.com signup, extract token |
| `tools/motherduck/src/motherduck/client.py` | MotherDuck DuckDB connection: connect, create_db, run_query |
| `tools/motherduck/src/motherduck/cli.py` | Typer app: register, account ls/rm, db create/ls/use/rm, query |
| `tools/motherduck/tests/test_store.py` | Unit tests for store.py with in-memory DuckDB |
| `tools/motherduck/tests/test_email.py` | Unit tests for email.py with mocked httpx |
| `tools/motherduck/tests/test_client.py` | Unit tests for client.py with mocked duckdb connection |

---

## Chunk 1: Project Scaffold + Store

### Task 1: Scaffold project structure

**Files:**
- Create: `tools/motherduck/pyproject.toml`
- Create: `tools/motherduck/src/motherduck/__init__.py`
- Create: `tools/motherduck/tests/__init__.py`

- [ ] **Step 1: Create directory tree**

```bash
mkdir -p tools/motherduck/src/motherduck tools/motherduck/tests
```

- [ ] **Step 2: Write `pyproject.toml`**

```toml
[project]
name = "motherduck"
version = "0.1.0"
description = "Auto-register MotherDuck accounts and manage cloud DuckDB databases"
requires-python = ">=3.11"
dependencies = [
    "typer>=0.12",
    "rich>=13.0",
    "patchright>=1.50",
    "duckdb>=1.2",
    "httpx>=0.27",
    "faker>=33.0",
]

[project.scripts]
motherduck = "motherduck.cli:app_entry"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/motherduck"]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "pytest-mock>=3.14",
]
```

- [ ] **Step 3: Create empty `__init__.py` files**

```bash
touch tools/motherduck/src/motherduck/__init__.py tools/motherduck/tests/__init__.py
```

- [ ] **Step 4: Bootstrap UV lockfile**

```bash
cd tools/motherduck && uv lock
```

Expected: `uv.lock` created.

- [ ] **Step 5: Commit scaffold**

```bash
git add tools/motherduck/pyproject.toml tools/motherduck/src/motherduck/__init__.py tools/motherduck/tests/__init__.py tools/motherduck/uv.lock
git commit -m "feat(motherduck): scaffold UV project"
```

---

### Task 2: Implement `store.py` (TDD)

**Files:**
- Create: `tools/motherduck/src/motherduck/store.py`
- Create: `tools/motherduck/tests/test_store.py`

- [ ] **Step 1: Write failing tests**

`tools/motherduck/tests/test_store.py`:

```python
"""Unit tests for store.py — uses in-memory DuckDB."""
from __future__ import annotations

import pytest
import duckdb

from motherduck.store import Store


@pytest.fixture
def store():
    """In-memory store for tests."""
    return Store(":memory:")


def test_init_creates_tables(store):
    tables = store.con.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
    ).fetchall()
    names = {r[0] for r in tables}
    assert "accounts" in names
    assert "databases" in names
    assert "query_log" in names


def test_add_account(store):
    acc_id = store.add_account(email="test@x.com", password="pass", token="tok123")
    assert acc_id is not None
    rows = store.list_accounts()
    assert len(rows) == 1
    assert rows[0]["email"] == "test@x.com"
    # list_accounts() returns summary columns, not token (use get_token_for_alias for token)
    assert rows[0]["is_active"] is True
    assert rows[0]["db_count"] == 0


def test_add_account_duplicate_email_raises(store):
    store.add_account(email="dupe@x.com", password="p", token="t")
    with pytest.raises(Exception):
        store.add_account(email="dupe@x.com", password="p2", token="t2")


def test_deactivate_account(store):
    store.add_account(email="a@x.com", password="p", token="t")
    store.deactivate_account("a@x.com")
    rows = store.list_accounts()
    assert rows[0]["is_active"] is False


def test_add_database(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    db_id = store.add_database(account_id=acc_id, name="mydb", alias="myalias")
    assert db_id is not None
    rows = store.list_databases()
    assert len(rows) == 1
    assert rows[0]["alias"] == "myalias"
    assert rows[0]["name"] == "mydb"
    assert rows[0]["is_default"] is False


def test_set_default_clears_previous(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    id1 = store.add_database(account_id=acc_id, name="db1", alias="a1")
    id2 = store.add_database(account_id=acc_id, name="db2", alias="a2")
    store.set_default("a1")
    store.set_default("a2")
    rows = store.list_databases()
    defaults = [r for r in rows if r["is_default"]]
    assert len(defaults) == 1
    assert defaults[0]["alias"] == "a2"


def test_get_default_db(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.set_default("a1")
    row = store.get_default_db()
    assert row is not None
    assert row["alias"] == "a1"


def test_get_default_db_none_when_empty(store):
    assert store.get_default_db() is None


def test_remove_database(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.remove_database("a1")
    assert store.list_databases() == []


def test_get_db_by_alias(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="mydb", alias="mine")
    row = store.get_db("mine")
    assert row is not None
    assert row["name"] == "mydb"


def test_get_db_missing_returns_none(store):
    assert store.get_db("nope") is None


def test_log_query(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    db_id = store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.log_query(db_id=db_id, sql="SELECT 1", rows_returned=1, duration_ms=5)
    rows = store.list_databases()
    assert rows[0]["query_count"] == 1


def test_get_token_for_alias(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="mytoken")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    token = store.get_token_for_alias("a1")
    assert token == "mytoken"
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd tools/motherduck && uv run pytest tests/test_store.py -v 2>&1 | head -20
```

Expected: `ModuleNotFoundError: No module named 'motherduck.store'`

- [ ] **Step 3: Implement `store.py`**

`tools/motherduck/src/motherduck/store.py`:

```python
"""Local DuckDB state: schema init, CRUD for accounts/databases/query_log."""
from __future__ import annotations

import os
from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "motherduck" / "mother.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id         VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email      VARCHAR NOT NULL UNIQUE,
    password   VARCHAR NOT NULL,
    token      VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    is_active  BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS databases (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    db_id         VARCHAR,
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
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

    def add_account(self, *, email: str, password: str, token: str) -> str:
        self.con.execute(
            "INSERT INTO accounts (email, password, token) VALUES (?, ?, ?) RETURNING id",
            [email, password, token],
        )
        return self.con.fetchone()[0]

    def list_accounts(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT a.id, a.email, a.created_at, a.is_active,
                   COUNT(d.id) AS db_count
            FROM accounts a
            LEFT JOIN databases d ON d.account_id = a.id
            GROUP BY a.id, a.email, a.created_at, a.is_active
            ORDER BY a.created_at DESC
            """
        ).fetchall()
        cols = ["id", "email", "created_at", "is_active", "db_count"]
        return [dict(zip(cols, r)) for r in rows]

    def deactivate_account(self, email: str) -> None:
        self.con.execute(
            "UPDATE accounts SET is_active = false WHERE email = ?", [email]
        )

    def get_first_active_account(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, token FROM accounts WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "token"], row))

    # ------------------------------------------------------------------
    # Databases
    # ------------------------------------------------------------------

    def add_database(
        self,
        *,
        account_id: str,
        name: str,
        alias: str,
        notes: str = "",
    ) -> str:
        self.con.execute(
            "INSERT INTO databases (account_id, name, alias, notes) VALUES (?, ?, ?, ?) RETURNING id",
            [account_id, name, alias, notes],
        )
        return self.con.fetchone()[0]

    def list_databases(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT d.alias, d.name, a.email, d.is_default,
                   d.created_at, d.last_used_at,
                   COUNT(q.id) AS query_count,
                   d.id
            FROM databases d
            JOIN accounts a ON a.id = d.account_id
            LEFT JOIN query_log q ON q.db_id = d.id
            GROUP BY d.alias, d.name, a.email, d.is_default, d.created_at, d.last_used_at, d.id
            ORDER BY d.created_at DESC
            """
        ).fetchall()
        cols = ["alias", "name", "email", "is_default", "created_at", "last_used_at", "query_count", "id"]
        return [dict(zip(cols, r)) for r in rows]

    def get_db(self, alias: str) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT d.id, d.name, d.alias, d.account_id, d.is_default,
                   a.token, a.email
            FROM databases d
            JOIN accounts a ON a.id = d.account_id
            WHERE d.alias = ?
            """,
            [alias],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "name", "alias", "account_id", "is_default", "token", "email"], row))

    def get_default_db(self) -> dict[str, Any] | None:
        row = self.con.execute(
            """
            SELECT d.id, d.name, d.alias, d.account_id, d.is_default,
                   a.token, a.email
            FROM databases d
            JOIN accounts a ON a.id = d.account_id
            WHERE d.is_default = true
            LIMIT 1
            """
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "name", "alias", "account_id", "is_default", "token", "email"], row))

    def set_default(self, alias: str) -> None:
        # Atomic: clear all then set one — wrapped in transaction for consistency
        self.con.begin()
        try:
            self.con.execute("UPDATE databases SET is_default = false")
            self.con.execute(
                "UPDATE databases SET is_default = true WHERE alias = ?", [alias]
            )
            self.con.commit()
        except Exception:
            self.con.rollback()
            raise

    def get_account_by_email(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, token FROM accounts WHERE email = ? AND is_active = true",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "token"], row))

    def remove_database(self, alias: str) -> None:
        self.con.execute("DELETE FROM databases WHERE alias = ?", [alias])

    def touch_last_used(self, alias: str) -> None:
        self.con.execute(
            "UPDATE databases SET last_used_at = now() WHERE alias = ?", [alias]
        )

    def get_token_for_alias(self, alias: str) -> str | None:
        row = self.con.execute(
            """
            SELECT a.token FROM databases d
            JOIN accounts a ON a.id = d.account_id
            WHERE d.alias = ?
            """,
            [alias],
        ).fetchone()
        return row[0] if row else None

    # ------------------------------------------------------------------
    # Query log
    # ------------------------------------------------------------------

    def log_query(
        self,
        *,
        db_id: str,
        sql: str,
        rows_returned: int,
        duration_ms: int,
    ) -> None:
        self.con.execute(
            "INSERT INTO query_log (db_id, sql, rows_returned, duration_ms) VALUES (?, ?, ?, ?)",
            [db_id, sql, rows_returned, duration_ms],
        )
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd tools/motherduck && uv run pytest tests/test_store.py -v
```

Expected: all 13 tests pass.

- [ ] **Step 5: Commit**

```bash
git add tools/motherduck/src/motherduck/store.py tools/motherduck/tests/test_store.py
git commit -m "feat(motherduck): store.py — DuckDB state with accounts/databases/query_log"
```

---

## Chunk 2: Identity + Email

### Task 3: Implement `identity.py`

**Files:**
- Create: `tools/motherduck/src/motherduck/identity.py`

This module is trivial (no logic to test, just Faker calls). No test file needed.

- [ ] **Step 1: Write `identity.py`**

`tools/motherduck/src/motherduck/identity.py`:

```python
"""Realistic identity generation for MotherDuck signup."""
from __future__ import annotations

import random
import secrets
import string
from dataclasses import dataclass

from faker import Faker

_fake = Faker()
_SPECIAL = "!@#$%^&*"


@dataclass
class Identity:
    display_name: str
    email_local: str   # local part only; domain assigned by mail.tm
    password: str      # stored in accounts table


def generate() -> Identity:
    """Return a randomly generated realistic identity."""
    display_name = _fake.name()
    local_base = _fake.user_name().replace("-", "").replace(".", "")
    email_local = (local_base + str(random.randint(10, 99)))[:20]

    pool = string.ascii_lowercase + string.ascii_uppercase + string.digits + _SPECIAL
    password = (
        secrets.choice(string.ascii_uppercase)
        + secrets.choice(string.ascii_lowercase)
        + secrets.choice(string.digits)
        + secrets.choice(_SPECIAL)
        + "".join(secrets.choice(pool) for _ in range(10))
    )
    chars = list(password)
    random.shuffle(chars)
    password = "".join(chars)

    return Identity(
        display_name=display_name,
        email_local=email_local,
        password=password,
    )
```

- [ ] **Step 2: Commit**

```bash
git add tools/motherduck/src/motherduck/identity.py
git commit -m "feat(motherduck): identity.py — Faker identity generation"
```

---

### Task 4: Implement `email.py` (TDD)

**Files:**
- Create: `tools/motherduck/src/motherduck/email.py`
- Create: `tools/motherduck/tests/test_email.py`

Key difference from x-register: MotherDuck sends a **magic link** (URL), not a 6-digit OTP. Poll for a URL matching `app.motherduck.com` or a token link in the email body.

- [ ] **Step 1: Write failing tests**

`tools/motherduck/tests/test_email.py`:

```python
"""Unit tests for email.py — mail.tm client."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from motherduck.email import MailTmClient, MailTmError, Mailbox


@pytest.fixture
def client():
    return MailTmClient(verbose=False)


def _mock_response(status_code: int, json_data: dict):
    m = MagicMock()
    m.status_code = status_code
    m.json.return_value = json_data
    m.raise_for_status = MagicMock()
    return m


def test_get_domain(client):
    with patch.object(client._client, "get") as mock_get:
        mock_get.return_value = _mock_response(
            200,
            {"hydra:member": [{"domain": "mail.tm", "isActive": True}]},
        )
        domain = client._get_domain()
    assert domain == "mail.tm"


def test_get_domain_no_active_raises(client):
    with patch.object(client._client, "get") as mock_get:
        mock_get.return_value = _mock_response(
            200, {"hydra:member": [{"domain": "dead.tm", "isActive": False}]}
        )
        with pytest.raises(MailTmError, match="No active"):
            client._get_domain()


def test_create_mailbox(client):
    with (
        patch.object(client, "_get_domain", return_value="mail.tm"),
        patch.object(client._client, "post") as mock_post,
        patch.object(client, "_get_token", return_value="jwt123"),
    ):
        mock_post.return_value = _mock_response(201, {})
        mb = client.create_mailbox("testuser")
    assert mb.address == "testuser@mail.tm"
    assert mb.token == "jwt123"


def test_poll_for_magic_link_found(client):
    mb = Mailbox(address="a@mail.tm", password="p", token="jwt")
    msgs = [
        {
            "id": "msg1",
            "subject": "Sign in to MotherDuck",
            "intro": "Click here to sign in",
        }
    ]
    full_msg = {
        "text": "Click this link: https://app.motherduck.com/auth/magic?token=abc123 to sign in.",
        "html": "",
    }

    def fake_get(url, **kwargs):
        if "messages/msg1" in url:
            return _mock_response(200, full_msg)
        return _mock_response(200, {"hydra:member": msgs})

    with patch.object(client._client, "get", side_effect=fake_get):
        link = client.poll_for_magic_link(mb, timeout=10)
    assert "app.motherduck.com" in link or "motherduck" in link


def test_poll_for_magic_link_timeout(client):
    mb = Mailbox(address="a@mail.tm", password="p", token="jwt")
    # Mock sleep to avoid 3s real wait in test
    with (
        patch.object(client._client, "get") as mock_get,
        patch("motherduck.email.time.sleep"),
    ):
        mock_get.return_value = _mock_response(200, {"hydra:member": []})
        with pytest.raises(MailTmError, match="not received"):
            client.poll_for_magic_link(mb, timeout=1)
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd tools/motherduck && uv run pytest tests/test_email.py -v 2>&1 | head -10
```

Expected: `ModuleNotFoundError: No module named 'motherduck.email'`

- [ ] **Step 3: Implement `email.py`**

`tools/motherduck/src/motherduck/email.py`:

```python
"""mail.tm API client: create mailbox and poll for magic link URL."""
from __future__ import annotations

import re
import time
from dataclasses import dataclass

import httpx

BASE = "https://api.mail.tm"
# MotherDuck sends a URL containing their domain
MAGIC_LINK_RE = re.compile(r"https://[^\s\"'<>]*motherduck\.com[^\s\"'<>]*")
POLL_INTERVAL = 3
POLL_TIMEOUT = 120


@dataclass
class Mailbox:
    address: str
    password: str
    token: str


class MailTmError(Exception):
    pass


class MailTmClient:
    def __init__(self, verbose: bool = False) -> None:
        self._verbose = verbose
        self._client = httpx.Client(timeout=15)

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [mail.tm] {msg}", flush=True)

    def _get_domain(self) -> str:
        resp = self._client.get(f"{BASE}/domains")
        resp.raise_for_status()
        domains = resp.json().get("hydra:member", [])
        active = [d["domain"] for d in domains if d.get("isActive")]
        if not active:
            raise MailTmError("No active mail.tm domains available")
        return active[0]

    def create_mailbox(self, local: str) -> Mailbox:
        domain = self._get_domain()
        address = f"{local}@{domain}"
        password = f"Mz{local[:6]}!9xQ"
        self._log(f"creating mailbox {address}")
        resp = self._client.post(
            f"{BASE}/accounts", json={"address": address, "password": password}
        )
        if resp.status_code not in (200, 201):
            raise MailTmError(
                f"create account failed: {resp.status_code} {resp.text[:200]}"
            )
        token = self._get_token(address, password)
        return Mailbox(address=address, password=password, token=token)

    def _get_token(self, address: str, password: str) -> str:
        resp = self._client.post(
            f"{BASE}/token", json={"address": address, "password": password}
        )
        resp.raise_for_status()
        return resp.json()["token"]

    def poll_for_magic_link(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox until a MotherDuck magic link arrives. Returns the URL."""
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()

        self._log(f"polling {mailbox.address} for magic link (timeout={timeout}s)")
        while time.time() < deadline:
            try:
                resp = self._client.get(f"{BASE}/messages", headers=headers)
                resp.raise_for_status()
                messages = resp.json().get("hydra:member", [])
                for msg in messages:
                    msg_id = msg.get("id", "")
                    if msg_id in seen:
                        continue
                    seen.add(msg_id)
                    subject = msg.get("subject", "")
                    intro = msg.get("intro", "")
                    self._log(f"  msg: subject={subject!r}")

                    # Fetch full message for link extraction
                    text = intro
                    try:
                        full = self._client.get(
                            f"{BASE}/messages/{msg_id}", headers=headers
                        )
                        body = full.json()
                        text = body.get("text", "") + " " + body.get("html", "") + " " + intro
                    except Exception:
                        pass

                    m = MAGIC_LINK_RE.search(text)
                    if m:
                        link = m.group(0).rstrip(".")
                        self._log(f"  magic link found: {link[:60]}...")
                        return link
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"Magic link not received within {timeout}s at {mailbox.address}"
        )

    def close(self) -> None:
        self._client.close()
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd tools/motherduck && uv run pytest tests/test_email.py -v
```

Expected: 5 tests pass.

- [ ] **Step 5: Commit**

```bash
git add tools/motherduck/src/motherduck/email.py tools/motherduck/tests/test_email.py
git commit -m "feat(motherduck): email.py — mail.tm client with magic link polling"
```

---

## Chunk 3: Browser Registration

### Task 5: Implement `browser.py`

**Files:**
- Create: `tools/motherduck/src/motherduck/browser.py`

This module drives a real Chrome browser; unit testing with mocks provides little value here. Manual integration testing is the correct approach (documented at end of plan). Write the module directly with clear step comments.

**Note on Linux xvfb re-exec:** When `_maybe_reexec_xvfb` triggers, the parent exits via `sys.exit()` after the child finishes. The child browser subprocess returns the token via stdout — this is acceptable for headful Linux runs. On headless Linux (the default), xvfb re-exec is never triggered.

**MotherDuck signup flow at `app.motherduck.com`:**
1. Land on home/login page → click "Sign up" or "Continue with email"
2. Enter email address → click Continue/Submit
3. MotherDuck sends magic link email → poll mail.tm → navigate to link
4. Onboarding wizard → skip/next through prompts
5. Navigate to `app.motherduck.com/settings/tokens` → click "Generate token"
6. Read token text from page

- [ ] **Step 1: Write `browser.py`**

`tools/motherduck/src/motherduck/browser.py`:

```python
"""Patchright-based MotherDuck account registration.

Flow:
  1. Open app.motherduck.com
  2. Click "Sign up" / "Continue with email"
  3. Enter mail.tm address → submit
  4. Poll mail.tm for magic link → navigate to it
  5. Click through onboarding prompts
  6. Navigate to Settings > Tokens → generate + extract token
"""
from __future__ import annotations

import os
import platform
import sys
import tempfile
import time

from .email import MailTmClient, Mailbox


def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage"]
    return args


def _maybe_reexec_xvfb(headless: bool) -> None:
    if platform.system() != "Linux" or headless or os.environ.get("DISPLAY"):
        return
    import shutil
    import subprocess
    xvfb = shutil.which("xvfb-run")
    if xvfb:
        sys.exit(subprocess.call([xvfb, "-a", sys.executable] + sys.argv))


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


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive MotherDuck signup, return the API token string."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="md_reg_")

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel="chrome",
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Load landing page ----
            log("opening app.motherduck.com...")
            try:
                page.goto("https://app.motherduck.com", timeout=30000)
            except Exception as e:
                log(f"landing warn: {e}")
            _wait(3, log, "page load")

            # ---- Step 2: Click "Sign up" or "Continue with email" ----
            log("clicking sign-up entry...")
            for sel in [
                'button:has-text("Sign up")',
                'a:has-text("Sign up")',
                'button:has-text("Continue with email")',
                '[href*="signup"]',
                'button:has-text("Get started")',
            ]:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"clicked: {sel}")
                    _wait(2, log)
                    break

            log(f"url after signup click: {page.url}")

            # ---- Step 3: Enter email ----
            log(f"entering email: {mailbox.address}")
            for sel in [
                'input[type="email"]',
                'input[name="email"]',
                'input[placeholder*="email" i]',
            ]:
                inp = page.locator(sel)
                if inp.count() > 0:
                    _fill(page, sel, mailbox.address)
                    log(f"filled email via: {sel}")
                    break

            # Submit email form
            for sel in [
                'button[type="submit"]',
                'button:has-text("Continue")',
                'button:has-text("Send magic link")',
                'button:has-text("Sign in")',
                'button:has-text("Submit")',
            ]:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"submitted via: {sel}")
                    _wait(2, log)
                    break

            log(f"url after email submit: {page.url}")

            # ---- Step 4: Poll mail.tm for magic link ----
            log("polling mail.tm for magic link...")
            magic_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
            log(f"got magic link: {magic_link[:60]}...")

            # ---- Step 5: Navigate to magic link ----
            log("navigating to magic link...")
            try:
                page.goto(magic_link, timeout=30000)
            except Exception as e:
                log(f"magic link nav warn: {e}")
            _wait(4, log, "post-magic-link load")
            log(f"url after magic link: {page.url}")

            # ---- Step 6: Click through onboarding ----
            log("clicking through onboarding...")
            _skip_onboarding(page, log, max_attempts=10)
            log(f"url after onboarding: {page.url}")

            # ---- Step 7: Go to Settings > Tokens ----
            log("navigating to token settings...")
            try:
                page.goto("https://app.motherduck.com/settings/tokens", timeout=20000)
            except Exception as e:
                log(f"settings nav warn: {e}")
            _wait(3, log, "settings load")
            log(f"url: {page.url}")

            # ---- Step 8: Generate token ----
            log("generating token...")
            for sel in [
                'button:has-text("Generate token")',
                'button:has-text("Create token")',
                'button:has-text("New token")',
                'button:has-text("Generate")',
            ]:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"clicked: {sel}")
                    _wait(2, log)
                    break

            # ---- Step 9: Extract token ----
            log("extracting token...")
            token = _extract_token(page, ctx, log)
            if not token:
                raise RuntimeError("Failed to extract MotherDuck token from settings page")

            log(f"token extracted (len={len(token)})")
            return token

        finally:
            ctx.close()


def _skip_onboarding(page, log, max_attempts: int = 10) -> None:
    """Click through MotherDuck onboarding prompts until done."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url
        # Consider done if on main app page
        if any(x in url for x in ["/editor", "/home", "/settings", "?onboarding=done"]):
            log(f"onboarding done at attempt {attempt}")
            return

        clicked = False
        for sel in [
            'button:has-text("Skip")',
            'button:has-text("Continue")',
            'button:has-text("Next")',
            'button:has-text("Get started")',
            'button:has-text("Done")',
            '[role="button"]:has-text("Skip")',
        ]:
            btn = page.locator(sel)
            if btn.count() > 0:
                log(f"  onboarding click: {sel}")
                btn.first.click()
                clicked = True
                break

        if not clicked:
            log(f"  no onboarding button at attempt {attempt}, url={url}")
            break


def _extract_token(page, ctx, log) -> str:
    """Try multiple strategies to extract the MotherDuck API token."""
    import re

    # Strategy 1: look for token displayed in a <code> or input element
    for sel in [
        'code',
        'input[readonly]',
        '[data-testid*="token"]',
        'pre',
    ]:
        try:
            el = page.locator(sel).first
            if el.count() > 0:
                text = el.inner_text() or el.get_attribute("value") or ""
                if len(text) > 20 and "\n" not in text.strip():
                    log(f"token via selector {sel!r}: {text[:20]}...")
                    return text.strip()
        except Exception:
            pass

    # Strategy 2: scan entire page text for MotherDuck token pattern
    # MotherDuck tokens are JWT-like: "eyJ..." or "motherduck_token_..."
    try:
        body = page.inner_text("body")
        for pat in [
            r"eyJ[A-Za-z0-9\-_]{30,}\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+",
            r"motherduck_token_[A-Za-z0-9_\-]{20,}",
        ]:
            m = re.search(pat, body)
            if m:
                token = m.group(0)
                log(f"token via body regex: {token[:20]}...")
                return token
    except Exception:
        pass

    # Strategy 3: localStorage
    try:
        token = page.evaluate(
            "() => localStorage.getItem('motherduck_token') || localStorage.getItem('token')"
        )
        if token and len(token) > 20:
            log(f"token via localStorage: {token[:20]}...")
            return token
    except Exception:
        pass

    # Strategy 4: cookies
    cookies = {c["name"]: c["value"] for c in ctx.cookies()}
    for key in ["motherduck_token", "token", "auth_token"]:
        if key in cookies and len(cookies[key]) > 20:
            log(f"token via cookie {key!r}")
            return cookies[key]

    return ""
```

- [ ] **Step 2: Commit**

```bash
git add tools/motherduck/src/motherduck/browser.py
git commit -m "feat(motherduck): browser.py — Patchright MotherDuck signup flow"
```

---

## Chunk 4: Client + CLI + Integration

### Task 6: Implement `client.py` (TDD)

**Files:**
- Create: `tools/motherduck/src/motherduck/client.py`
- Create: `tools/motherduck/tests/test_client.py`

- [ ] **Step 1: Write failing tests**

`tools/motherduck/tests/test_client.py`:

```python
"""Unit tests for client.py — MotherDuck connection wrapper."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch, call


def _make_mock_con(fetchall_return=None, fetchone_return=None, description=None):
    con = MagicMock()
    cursor = MagicMock()
    cursor.fetchall.return_value = fetchall_return or []
    cursor.fetchone.return_value = fetchone_return
    cursor.description = description or [("col1",), ("col2",)]
    con.execute.return_value = cursor
    return con


def test_create_db_executes_create(monkeypatch):
    from motherduck.client import MotherDuckClient

    mock_con = _make_mock_con()
    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        client.create_db("mydb")

    calls = [str(c) for c in mock_con.execute.call_args_list]
    assert any("CREATE DATABASE" in c and "mydb" in c for c in calls)


def test_run_query_returns_rows(monkeypatch):
    from motherduck.client import MotherDuckClient

    desc = [("a",), ("b",)]
    mock_con = _make_mock_con(
        fetchall_return=[(1, "x"), (2, "y")],
        description=desc,
    )
    mock_con.execute.return_value.description = desc
    mock_con.execute.return_value.fetchall.return_value = [(1, "x"), (2, "y")]

    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        rows, cols = client.run_query("mydb", "SELECT a, b FROM t")

    assert cols == ["a", "b"]
    assert rows == [(1, "x"), (2, "y")]


def test_run_query_uses_correct_db(monkeypatch):
    from motherduck.client import MotherDuckClient

    mock_con = _make_mock_con()
    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        client.run_query("targetdb", "SELECT 1")

    calls = [str(c) for c in mock_con.execute.call_args_list]
    assert any("USE" in c and "targetdb" in c for c in calls)
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd tools/motherduck && uv run pytest tests/test_client.py -v 2>&1 | head -10
```

Expected: `ModuleNotFoundError: No module named 'motherduck.client'`

- [ ] **Step 3: Implement `client.py`**

`tools/motherduck/src/motherduck/client.py`:

```python
"""MotherDuck DuckDB connection wrapper."""
from __future__ import annotations

import duckdb


class MotherDuckClient:
    def __init__(self, token: str) -> None:
        self._token = token
        self._con = duckdb.connect(f"md:?motherduck_token={token}")
        # Load motherduck extension
        try:
            self._con.execute("LOAD motherduck")
        except Exception:
            pass  # may already be loaded

    def create_db(self, name: str) -> None:
        """Create a new database in MotherDuck."""
        self._con.execute(f"CREATE DATABASE IF NOT EXISTS {name}")

    def run_query(
        self, db_name: str, sql: str
    ) -> tuple[list[tuple], list[str]]:
        """Run SQL against db_name. Returns (rows, column_names)."""
        self._con.execute(f"USE {db_name}")
        cursor = self._con.execute(sql)
        cols = [d[0] for d in (cursor.description or [])]
        rows = cursor.fetchall()
        return rows, cols

    def list_dbs(self) -> list[str]:
        """List all databases visible to this token."""
        rows = self._con.execute("SHOW DATABASES").fetchall()
        return [r[0] for r in rows]

    def close(self) -> None:
        self._con.close()
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd tools/motherduck && uv run pytest tests/test_client.py -v
```

Expected: 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add tools/motherduck/src/motherduck/client.py tools/motherduck/tests/test_client.py
git commit -m "feat(motherduck): client.py — MotherDuck DuckDB connection wrapper"
```

---

### Task 7: Implement `cli.py`

**Files:**
- Create: `tools/motherduck/src/motherduck/cli.py`

Typer app with two sub-apps (`account_app`, `db_app`) plus top-level `register` and `query` commands. Rich for all output.

- [ ] **Step 1: Write `cli.py`**

`tools/motherduck/src/motherduck/cli.py`:

```python
"""Typer CLI: motherduck register / account / db / query."""
from __future__ import annotations

import sys
import time
from pathlib import Path
from typing import Annotated, Optional

import json
import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="motherduck",
    help="Manage MotherDuck accounts and cloud DuckDB databases.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
db_app = typer.Typer(help="Manage databases.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(db_app, name="db")

console = Console()
err_console = Console(stderr=True)


def _store() -> Store:
    return Store(DEFAULT_DB_PATH)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    no_headless: Annotated[bool, typer.Option("--no-headless", help="Show browser window")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Auto-register a new MotherDuck account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    console.print("[bold green]Opening browser for MotherDuck signup...[/bold green]")

    try:
        token = register_via_browser(
            mailbox=mailbox,
            mail_client=mail_client,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Registration failed:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        mail_client.close()

    store = _store()
    acc_id = store.add_account(
        email=mailbox.address,
        password=mailbox.password,
        token=token,
    )

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    console.print(f"[dim]Token:[/dim] {token[:20]}...")
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
    table.add_column("Databases", justify="right")
    table.add_column("Active", justify="center")
    table.add_column("Created At")

    for r in rows:
        active = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        table.add_row(r["email"], str(r["db_count"]), active, created)

    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Account email to deactivate")],
) -> None:
    """Deactivate an account (local only — does not delete from MotherDuck)."""
    store = _store()
    if not store.get_account_by_email(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.deactivate_account(email)
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# db
# ---------------------------------------------------------------------------

@db_app.command("create")
def db_create(
    name: Annotated[str, typer.Argument(help="Database name on MotherDuck")],
    alias: Annotated[Optional[str], typer.Option("--alias", help="Local alias (default: same as name)")] = None,
    account: Annotated[Optional[str], typer.Option("--account", help="Account email to use")] = None,
    set_default: Annotated[bool, typer.Option("--default", help="Set as default database")] = False,
) -> None:
    """Create a new database on MotherDuck."""
    from .client import MotherDuckClient

    alias = alias or name
    store = _store()

    # Resolve account
    if account:
        row = store.get_account_by_email(account)
        if not row:
            err_console.print(f"[bold red]Account not found:[/bold red] {account}")
            raise typer.Exit(1)
        acc_id, token = row["id"], row["token"]
    else:
        acc = store.get_first_active_account()
        if not acc:
            err_console.print("[bold red]No active accounts. Run:[/bold red] motherduck register")
            raise typer.Exit(1)
        acc_id, token = acc["id"], acc["token"]

    with console.status(f"[green]Creating database '{name}' on MotherDuck..."):
        try:
            client = MotherDuckClient(token=token)
            client.create_db(name)
            client.close()
        except Exception as e:
            err_console.print(f"[bold red]Failed to create database:[/bold red] {e}")
            raise typer.Exit(1)

    store.add_database(account_id=acc_id, name=name, alias=alias)
    if set_default:
        store.set_default(alias)

    console.print(f"[bold green]✓ Created:[/bold green] {name} [dim](alias: {alias})[/dim]")
    if set_default:
        console.print(f"[green]Set as default.[/green]")


@db_app.command("ls")
def db_ls() -> None:
    """List all databases."""
    store = _store()
    rows = store.list_databases()
    if not rows:
        console.print("[yellow]No databases. Run:[/yellow] motherduck db create <name>")
        return

    table = Table(title="Databases", show_lines=True)
    table.add_column("Alias", style="cyan")
    table.add_column("Name")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Queries", justify="right")
    table.add_column("Last Used")
    table.add_column("Created")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        last = str(r["last_used_at"])[:16] if r["last_used_at"] else "-"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        alias_str = f"[bold]{r['alias']}[/bold]" if r["is_default"] else r["alias"]
        table.add_row(
            alias_str, r["name"], r["email"], default,
            str(r["query_count"]), last, created,
        )

    console.print(table)


@db_app.command("use")
def db_use(
    alias: Annotated[str, typer.Argument(help="Alias to set as default")],
) -> None:
    """Set the default database."""
    store = _store()
    if not store.get_db(alias):
        err_console.print(f"[bold red]No database with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.set_default(alias)
    console.print(f"[green]Default set to:[/green] {alias}")


@db_app.command("rm")
def db_rm(
    alias: Annotated[str, typer.Argument(help="Alias to remove")],
) -> None:
    """Remove a database from local state (does not delete from MotherDuck cloud)."""
    store = _store()
    if not store.get_db(alias):
        err_console.print(f"[bold red]No database with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.remove_database(alias)
    console.print(f"[yellow]Removed:[/yellow] {alias} [dim](local state only)[/dim]")


# ---------------------------------------------------------------------------
# query
# ---------------------------------------------------------------------------

@app.command()
def query(
    sql: Annotated[str, typer.Argument(help="SQL to run")],
    db: Annotated[Optional[str], typer.Option("--db", help="DB alias or name (default: use default)")] = None,
    json_out: Annotated[bool, typer.Option("--json", help="Output raw JSON")] = False,
) -> None:
    """Run SQL against a MotherDuck database."""
    from .client import MotherDuckClient
    import time as _time

    store = _store()

    # Resolve DB
    if db:
        db_row = store.get_db(db)
        if not db_row:
            err_console.print(f"[bold red]No database with alias:[/bold red] {db}")
            raise typer.Exit(1)
    else:
        db_row = store.get_default_db()
        if not db_row:
            err_console.print(
                "[bold red]No default database set. Use:[/bold red] motherduck db use <alias>"
            )
            raise typer.Exit(1)

    token = db_row["token"]
    db_name = db_row["name"]
    db_id = db_row["id"]

    t0 = _time.monotonic()
    try:
        client = MotherDuckClient(token=token)
        rows, cols = client.run_query(db_name, sql)
        client.close()
    except Exception as e:
        err_console.print(f"[bold red]Query failed:[/bold red] {e}")
        raise typer.Exit(1)

    duration_ms = int((_time.monotonic() - t0) * 1000)
    store.log_query(db_id=db_id, sql=sql, rows_returned=len(rows), duration_ms=duration_ms)
    store.touch_last_used(db_row["alias"])

    if json_out:
        data = [dict(zip(cols, r)) for r in rows]
        print(json.dumps(data, indent=2, default=str))
        return

    if not rows:
        console.print("[dim]No rows returned.[/dim]")
        return

    table = Table(show_lines=True)
    for col in cols:
        table.add_column(col)
    for row in rows:
        table.add_row(*[str(v) if v is not None else "[dim]NULL[/dim]" for v in row])

    console.print(table)
    console.print(
        f"[dim]{len(rows)} row(s) · {duration_ms}ms · db: {db_name}[/dim]"
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()
```

- [ ] **Step 2: Commit**

```bash
git add tools/motherduck/src/motherduck/cli.py
git commit -m "feat(motherduck): cli.py — Typer CLI with account/db/query commands"
```

---

### Task 8: Run all unit tests

- [ ] **Step 1: Run full test suite**

```bash
cd tools/motherduck && uv run pytest tests/ -v
```

Expected output (all pass):
```
tests/test_store.py::test_init_creates_tables PASSED
tests/test_store.py::test_add_account PASSED
tests/test_store.py::test_add_account_duplicate_email_raises PASSED
tests/test_store.py::test_deactivate_account PASSED
tests/test_store.py::test_add_database PASSED
tests/test_store.py::test_set_default_clears_previous PASSED
tests/test_store.py::test_get_default_db PASSED
tests/test_store.py::test_get_default_db_none_when_empty PASSED
tests/test_store.py::test_remove_database PASSED
tests/test_store.py::test_get_db_by_alias PASSED
tests/test_store.py::test_get_db_missing_returns_none PASSED
tests/test_store.py::test_log_query PASSED
tests/test_store.py::test_get_token_for_alias PASSED
tests/test_email.py::test_get_domain PASSED
tests/test_email.py::test_get_domain_no_active_raises PASSED
tests/test_email.py::test_create_mailbox PASSED
tests/test_email.py::test_poll_for_magic_link_found PASSED
tests/test_email.py::test_poll_for_magic_link_timeout PASSED
tests/test_client.py::test_create_db_executes_create PASSED
tests/test_client.py::test_run_query_returns_rows PASSED
tests/test_client.py::test_run_query_uses_correct_db PASSED
===================== 21 passed ================
```

- [ ] **Step 2: Install Patchright browser (one-time setup)**

```bash
cd tools/motherduck && uv run patchright install chrome
```

- [ ] **Step 3: Commit final state**

```bash
git add tools/motherduck/
git commit -m "feat(motherduck): complete implementation with 21 passing tests"
```

---

### Task 9: Integration Test

Manual end-to-end test to verify the full registration + query flow works.

- [ ] **Step 1: Install tool in development mode**

```bash
cd tools/motherduck && uv sync
```

- [ ] **Step 2: Register an account (visible browser for debugging)**

```bash
cd tools/motherduck && uv run motherduck register --no-headless --verbose
```

Expected: browser opens, navigates to app.motherduck.com, submits email, waits for magic link, clicks through onboarding, extracts token. Output ends with:
```
✓ Registered: <email>@<mail.tm domain>
Token: eyJ...
```

- [ ] **Step 3: Verify account stored**

```bash
uv run motherduck account ls
```

Expected: Rich table with 1 row, `Active = ✓`.

- [ ] **Step 4: Create a database**

```bash
uv run motherduck db create test_db --alias test --default
```

Expected:
```
✓ Created: test_db (alias: test)
Set as default.
```

- [ ] **Step 5: List databases**

```bash
uv run motherduck db ls
```

Expected: Rich table showing `test` alias, `test_db` name, `●` for default.

- [ ] **Step 6: Run a query**

```bash
uv run motherduck query "SELECT 1 AS result, 'hello' AS msg"
```

Expected: Rich table with columns `result`, `msg` and one row `1 | hello`.

- [ ] **Step 7: Run a query with explicit `--db`**

```bash
uv run motherduck query "SELECT 2 + 2 AS math" --db test
```

Expected: Rich table with `math = 4`.

- [ ] **Step 8: Run query with `--json` output**

```bash
uv run motherduck query "SELECT 42 AS n" --json
```

Expected:
```json
[{"n": 42}]
```

- [ ] **Step 9: Check query was logged in db ls**

```bash
uv run motherduck db ls
```

Expected: `Queries` column shows `3` (or however many were run).

- [ ] **Step 10: Test `db use` for default rotation**

```bash
uv run motherduck db create test_db2 --alias test2
uv run motherduck db use test2
uv run motherduck db ls
```

Expected: `test2` now has `●`, `test` does not.

- [ ] **Step 11: Note on cloud cleanup**

`db rm` removes from local state only. To delete databases from MotherDuck cloud, log into `app.motherduck.com` and delete via the UI. Test accounts created during integration can be reused across runs — `register` creates a new account each time.

> Integration test complete. Commit any outstanding changes:
> ```bash
> git add tools/motherduck/
> git commit -m "test(motherduck): integration test verified"
> ```
