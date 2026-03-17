# Goodreads Auth Tool Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `tools/goodread` — a Python CLI that auto-registers Goodreads accounts via mail.tm + Patchright — and wire its exported cookies into the Go scraper to unlock login-gated pages (search with pagination, user shelves).

**Architecture:** Python tool (A1) stores accounts + cookies in its own DuckDB (`~/data/goodread/accounts.duckdb`); `cookies export` writes `~/data/goodread/cookies.json`. Go reads that JSON file to build an authenticated `*http.Client` with a pre-populated cookie jar. The two DBs never share a lock.

**Tech Stack:** Python 3.11+, uv, Patchright (browser automation), Typer + Rich (CLI), httpx (mail.tm), Faker (identity), DuckDB (store) — Go side: `net/http` CookieJar, `encoding/json`.

---

## File Map

### New files — Python tool

| Path | Responsibility |
|------|---------------|
| `tools/goodread/pyproject.toml` | uv project, deps, entry point script |
| `tools/goodread/Makefile` | sync / test / build targets |
| `tools/goodread/goodread_entry.py` | PyInstaller entry point |
| `tools/goodread/goodread-tool.spec` | PyInstaller spec |
| `tools/goodread/src/goodread_tool/__init__.py` | empty |
| `tools/goodread/src/goodread_tool/identity.py` | Faker name / email-local / password |
| `tools/goodread/src/goodread_tool/email.py` | mail.tm API client (adapted from motherduck) |
| `tools/goodread/src/goodread_tool/store.py` | DuckDB accounts table (email, password, user_id, cookies JSON) |
| `tools/goodread/src/goodread_tool/browser.py` | Patchright signup + cookie extraction |
| `tools/goodread/src/goodread_tool/cli.py` | Typer: register, account ls/rm, test, cookies export |
| `tools/goodread/tests/__init__.py` | empty |
| `tools/goodread/tests/test_store.py` | unit tests for Store (in-memory DuckDB) |
| `tools/goodread/tests/test_identity.py` | unit tests for identity generation |

### New / modified files — Go scraper

| Path | Responsibility |
|------|---------------|
| `pkg/scrape/goodread/cookies.go` | NEW — `LoadCookiesFromFile`, `DefaultCookiesPath` |
| `pkg/scrape/goodread/client.go` | MODIFY — add `NewClientWithCookies` |
| `pkg/scrape/goodread/parse_search.go` | MODIFY — add `ParseSearchHTML` for logged-in HTML |
| `cli/goodread.go` | MODIFY — `--auth` on `search` + `shelf` |

---

## Chunk 1: Python tool scaffold + identity + store

### Task 1: Project scaffold

**Files:**
- Create: `tools/goodread/pyproject.toml`
- Create: `tools/goodread/Makefile`
- Create: `tools/goodread/goodread_entry.py`
- Create: `tools/goodread/src/goodread_tool/__init__.py`
- Create: `tools/goodread/tests/__init__.py`

- [ ] **Create `tools/goodread/pyproject.toml`**

```toml
[project]
name = "goodread"
version = "0.1.0"
description = "Auto-register Goodreads accounts and export session cookies"
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
goodread = "goodread_tool.cli:app_entry"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/goodread_tool"]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "pytest-mock>=3.14",
]
```

- [ ] **Create `tools/goodread/Makefile`**

```makefile
.PHONY: sync test build install clean

BIN_NAME = goodread-tool
DIST_DIR = dist

sync:
	uv sync --group dev

test: sync
	uv run pytest tests/ -v

build: sync
	uv run pyinstaller --clean --noconfirm goodread-tool.spec

install: build
	mkdir -p $(HOME)/bin
	cp $(DIST_DIR)/$(BIN_NAME) $(HOME)/bin/$(BIN_NAME)
	chmod +x $(HOME)/bin/$(BIN_NAME)

clean:
	rm -rf dist/ build/ __pycache__/ *.pyc
```

- [ ] **Create `tools/goodread/goodread_entry.py`**

```python
"""PyInstaller entry point."""
import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))
from goodread_tool.cli import app_entry
if __name__ == '__main__':
    app_entry()
```

- [ ] **Create empty `src/goodread_tool/__init__.py` and `tests/__init__.py`**

- [ ] **Run `uv sync --group dev` from `tools/goodread/` — verify no errors**

```bash
cd tools/goodread && uv sync --group dev
```

- [ ] **Commit**

```bash
git add tools/goodread/
git commit -m "feat(goodread-tool): project scaffold"
```

---

### Task 2: identity.py

**Files:**
- Create: `tools/goodread/src/goodread_tool/identity.py`
- Create: `tools/goodread/tests/test_identity.py`

- [ ] **Write failing test**

```python
# tests/test_identity.py
from goodread_tool.identity import generate

def test_generate_returns_required_fields():
    ident = generate()
    assert ident.first_name
    assert ident.last_name
    assert ident.email_local  # no @ or domain
    assert "@" not in ident.email_local
    assert len(ident.password) >= 12

def test_generate_password_complexity():
    ident = generate()
    p = ident.password
    assert any(c.isupper() for c in p)
    assert any(c.islower() for c in p)
    assert any(c.isdigit() for c in p)
    assert any(c in "!@#$%^&*" for c in p)

def test_generate_unique():
    a, b = generate(), generate()
    assert a.email_local != b.email_local
```

- [ ] **Run to verify FAIL**

```bash
cd tools/goodread && uv run pytest tests/test_identity.py -v
# Expected: ImportError or ModuleNotFoundError
```

- [ ] **Implement `identity.py`**

```python
"""Realistic identity for Goodreads signup."""
from __future__ import annotations
import random, secrets, string
from dataclasses import dataclass
from faker import Faker

_fake = Faker()
_SPECIAL = "!@#$%^&*"

@dataclass
class Identity:
    first_name: str
    last_name:  str
    email_local: str   # local part only; domain assigned by mail.tm
    password:    str

def generate() -> Identity:
    first = _fake.first_name()
    last  = _fake.last_name()
    base  = _fake.user_name().replace("-", "").replace(".", "")
    local = (base + str(random.randint(10, 99)))[:20]

    pool = string.ascii_lowercase + string.ascii_uppercase + string.digits + _SPECIAL
    pwd  = (
        secrets.choice(string.ascii_uppercase)
        + secrets.choice(string.ascii_lowercase)
        + secrets.choice(string.digits)
        + secrets.choice(_SPECIAL)
        + "".join(secrets.choice(pool) for _ in range(10))
    )
    chars = list(pwd); random.shuffle(chars); pwd = "".join(chars)
    return Identity(first_name=first, last_name=last, email_local=local, password=pwd)
```

- [ ] **Run tests — verify PASS**

```bash
cd tools/goodread && uv run pytest tests/test_identity.py -v
```

- [ ] **Commit**

```bash
git add tools/goodread/src/goodread_tool/identity.py tools/goodread/tests/test_identity.py
git commit -m "feat(goodread-tool): identity generation"
```

---

### Task 3: store.py

**Files:**
- Create: `tools/goodread/src/goodread_tool/store.py`
- Create: `tools/goodread/tests/test_store.py`

- [ ] **Write failing tests**

```python
# tests/test_store.py
import json
import pytest
from goodread_tool.store import Store

@pytest.fixture
def store():
    s = Store(":memory:")
    yield s
    s.close()

def test_add_and_list(store):
    store.add(email="a@x.com", password="pw", cookies=json.dumps([{"name":"s","value":"v"}]))
    rows = store.list_all()
    assert len(rows) == 1
    assert rows[0]["email"] == "a@x.com"

def test_get_by_email(store):
    store.add(email="b@x.com", password="pw2", cookies="[]")
    row = store.get("b@x.com")
    assert row is not None
    assert row["password"] == "pw2"

def test_get_missing_returns_none(store):
    assert store.get("nobody@x.com") is None

def test_update_cookies(store):
    store.add(email="c@x.com", password="pw", cookies="[]")
    store.update_cookies("c@x.com", json.dumps([{"name":"n","value":"v2"}]))
    row = store.get("c@x.com")
    cookies = json.loads(row["cookies"])
    assert cookies[0]["value"] == "v2"

def test_update_user_id(store):
    store.add(email="d@x.com", password="pw", cookies="[]")
    store.update_user_id("d@x.com", "12345")
    row = store.get("d@x.com")
    assert row["user_id"] == "12345"

def test_get_first_active(store):
    store.add(email="e@x.com", password="pw", cookies="[]")
    row = store.get_first_active()
    assert row["email"] == "e@x.com"

def test_remove(store):
    store.add(email="f@x.com", password="pw", cookies="[]")
    store.remove("f@x.com")
    assert store.get("f@x.com") is None
```

- [ ] **Run to verify FAIL**

```bash
cd tools/goodread && uv run pytest tests/test_store.py -v
```

- [ ] **Implement `store.py`**

```python
"""DuckDB-backed store for Goodreads accounts + session cookies."""
from __future__ import annotations
from pathlib import Path
from typing import Any
import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "goodread" / "accounts.duckdb"
DEFAULT_COOKIES_PATH = Path.home() / "data" / "goodread" / "cookies.json"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id         VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email      VARCHAR NOT NULL UNIQUE,
    password   VARCHAR NOT NULL,
    user_id    VARCHAR DEFAULT '',
    cookies    VARCHAR NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT now(),
    is_active  BOOLEAN DEFAULT true
);
"""

class Store:
    def __init__(self, path: str | Path = DEFAULT_DB_PATH) -> None:
        if str(path) != ":memory:":
            Path(path).parent.mkdir(parents=True, exist_ok=True)
        self.con = duckdb.connect(str(path))
        self.con.execute(_SCHEMA)

    def add(self, *, email: str, password: str, cookies: str, user_id: str = "") -> str:
        row = self.con.execute(
            "INSERT INTO accounts (email, password, cookies, user_id) "
            "VALUES (?, ?, ?, ?) RETURNING id",
            [email, password, cookies, user_id],
        ).fetchone()
        return row[0]

    def list_all(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            "SELECT id, email, password, user_id, created_at, is_active "
            "FROM accounts ORDER BY created_at DESC"
        ).fetchall()
        cols = ["id", "email", "password", "user_id", "created_at", "is_active"]
        return [dict(zip(cols, r)) for r in rows]

    def get(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, user_id, cookies FROM accounts WHERE email = ?",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "user_id", "cookies"], row))

    def get_first_active(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, user_id, cookies FROM accounts "
            "WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "user_id", "cookies"], row))

    def update_cookies(self, email: str, cookies: str) -> None:
        self.con.execute(
            "UPDATE accounts SET cookies = ? WHERE email = ?", [cookies, email]
        )

    def update_user_id(self, email: str, user_id: str) -> None:
        self.con.execute(
            "UPDATE accounts SET user_id = ? WHERE email = ?", [user_id, email]
        )

    def remove(self, email: str) -> None:
        self.con.execute("DELETE FROM accounts WHERE email = ?", [email])

    def deactivate(self, email: str) -> None:
        self.con.execute(
            "UPDATE accounts SET is_active = false WHERE email = ?", [email]
        )

    def close(self) -> None:
        self.con.close()
```

- [ ] **Run tests — verify PASS**

```bash
cd tools/goodread && uv run pytest tests/test_store.py -v
```

- [ ] **Commit**

```bash
git add tools/goodread/src/goodread_tool/store.py tools/goodread/tests/test_store.py
git commit -m "feat(goodread-tool): DuckDB store"
```

---

### Task 4: email.py — mail.tm client

**Files:**
- Create: `tools/goodread/src/goodread_tool/email.py`

This is identical to motherduck's `email.py` except the magic-link regex targets `goodreads.com`.

- [ ] **Create `email.py`** by adapting `tools/motherduck/src/motherduck/email.py`:

Key changes from motherduck:
1. `MAGIC_LINK_RE = re.compile(r"https://[^\s\"'<>]*goodreads\.com[^\s\"'<>]*")`
2. Update `_pick_verification_link` to prefer `goodreads.com` links containing `confirm`, `verify`, or `token=`

```python
"""mail.tm API client for Goodreads email verification."""
from __future__ import annotations
import re, time
from dataclasses import dataclass
import httpx

BASE = "https://api.mail.tm"
ALL_URLS_RE = re.compile(r"https?://[^\s\"'<>]+")
POLL_INTERVAL = 3
POLL_TIMEOUT  = 120


def _pick_verification_link(urls: list[str]) -> str:
    cleaned = [u.rstrip(".") for u in urls]
    # Priority 1: Goodreads confirm/verify links
    for u in cleaned:
        if "goodreads.com" in u and any(
            kw in u.lower() for kw in ["confirm", "verify", "token=", "email"]
        ):
            return u
    # Priority 2: any long goodreads.com link with query params
    for u in cleaned:
        if "goodreads.com" in u and "?" in u and len(u) > 60:
            return u
    # Fallback: first link
    return cleaned[0] if cleaned else ""


@dataclass
class Mailbox:
    address: str
    password: str
    token:    str


class MailTmError(Exception):
    pass


class MailTmClient:
    def __init__(self, verbose: bool = False) -> None:
        self._verbose = verbose
        self._client  = httpx.Client(timeout=15)

    def _log(self, msg: str) -> None:
        if self._verbose:
            print(f"[{time.strftime('%H:%M:%S')}] [mail.tm] {msg}", flush=True)

    def _get_domain(self) -> str:
        resp = self._client.get(f"{BASE}/domains")
        resp.raise_for_status()
        domains = resp.json().get("hydra:member", [])
        active  = [d["domain"] for d in domains if d.get("isActive")]
        if not active:
            raise MailTmError("No active mail.tm domains available")
        return active[0]

    def create_mailbox(self, local: str) -> Mailbox:
        domain   = self._get_domain()
        address  = f"{local}@{domain}"
        password = f"Mz{local[:6]}!9xQ"
        self._log(f"creating mailbox {address}")
        resp = self._client.post(
            f"{BASE}/accounts", json={"address": address, "password": password}
        )
        if resp.status_code not in (200, 201):
            raise MailTmError(f"create account failed: {resp.status_code} {resp.text[:200]}")
        token = self._get_token(address, password)
        return Mailbox(address=address, password=password, token=token)

    def _get_token(self, address: str, password: str) -> str:
        resp = self._client.post(f"{BASE}/token", json={"address": address, "password": password})
        resp.raise_for_status()
        return resp.json()["token"]

    def poll_for_verification_link(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        headers  = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()
        self._log(f"polling {mailbox.address} for verification link (timeout={timeout}s)")
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
                    self._log(f"  msg: subject={msg.get('subject','')!r}")
                    text = msg.get("intro", "")
                    try:
                        full = self._client.get(f"{BASE}/messages/{msg_id}", headers=headers)
                        body = full.json()
                        html_parts = body.get("html", [])
                        html_str   = " ".join(html_parts) if isinstance(html_parts, list) else str(html_parts)
                        text = body.get("text", "") + " " + html_str + " " + text
                    except Exception as e:
                        self._log(f"  body fetch error: {e}")
                    all_urls = ALL_URLS_RE.findall(text)
                    link = _pick_verification_link(all_urls)
                    if link:
                        self._log(f"  verification link: {link[:120]}")
                        return link
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)
        raise MailTmError(f"Verification link not received within {timeout}s at {mailbox.address}")

    def close(self) -> None:
        self._client.close()
```

- [ ] **Commit**

```bash
git add tools/goodread/src/goodread_tool/email.py
git commit -m "feat(goodread-tool): mail.tm client"
```

---

## Chunk 2: browser.py — Patchright Goodreads signup

### Task 5: browser.py

**Files:**
- Create: `tools/goodread/src/goodread_tool/browser.py`

The Goodreads signup URL is `https://www.goodreads.com/user/sign_up`.

Form fields (based on Goodreads' standard signup form):
- `user[first_name]` or `input[name*="first"]`
- `user[last_name]`  or `input[name*="last"]`
- `user[email]`      or `input[type="email"]`
- `user[password]`   or `input[type="password"]`
- Submit: `input[type="submit"][value*="Sign up"]` or `button[type="submit"]`

After signup → Goodreads sends a verification email → poll mail.tm → click link in browser → extract cookies.

- [ ] **Create `browser.py`**

```python
"""Patchright browser automation for Goodreads account registration.

Flow:
  1. Open www.goodreads.com/user/sign_up
  2. Fill first name, last name, email, password
  3. Submit form
  4. If email verification required: poll mail.tm, click link in browser
  5. Confirm we are logged in (URL contains /home or /dashboard or shows username)
  6. Extract full cookie jar from browser context
  7. Optionally extract numeric user_id from profile URL
"""
from __future__ import annotations
import json, os, platform, re, tempfile, time
from .email import MailTmClient, Mailbox


# ── helpers (same pattern as motherduck) ──────────────────────────────────────

def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox",
                 "--disable-dev-shm-usage", "--disable-gpu"]
    return args


def _ensure_display() -> None:
    if platform.system() != "Linux" or os.environ.get("DISPLAY"):
        return
    import shutil, subprocess
    xvfb = shutil.which("Xvfb")
    if xvfb:
        proc = subprocess.Popen(
            [xvfb, ":99", "-screen", "0", "1280x900x24"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        import atexit; atexit.register(proc.kill)
        time.sleep(0.5)
        os.environ["DISPLAY"] = ":99"


def _wait(s: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {s}s ({msg})...")
    time.sleep(s)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=10000)
    el.click(); time.sleep(0.3)
    el.fill(""); time.sleep(0.1)
    el.type(text, delay=delay); time.sleep(0.3)


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            if page.locator(sel).count() > 0:
                _fill(page, sel, text)
                if log: log(f"filled via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _click_first(page, selectors: list[str], log=None) -> str | None:
    for sel in selectors:
        try:
            btn = page.locator(sel)
            if btn.count() > 0:
                btn.first.click()
                if log: log(f"clicked: {sel}")
                return sel
        except Exception:
            continue
    return None


def _logged_in(page) -> bool:
    url = page.url
    return any(x in url for x in [
        "/home", "/dashboard", "/user/show/", "/shelves", "/review/list",
    ]) and "sign_in" not in url and "sign_up" not in url


def _extract_user_id(page, log=None) -> str:
    """Try to extract the numeric Goodreads user ID from the profile link."""
    try:
        # Profile link in nav: /user/show/<id>-<name>
        nav_html = page.evaluate("document.documentElement.innerHTML")
        m = re.search(r'/user/show/(\d+)', nav_html)
        if m:
            uid = m.group(1)
            if log: log(f"user_id={uid}")
            return uid
    except Exception as e:
        if log: log(f"user_id extract error: {e}")
    return ""


# ── main registration function ─────────────────────────────────────────────────

def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    first_name: str,
    last_name: str,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> tuple[str, str]:
    """Drive Goodreads signup. Returns (cookies_json, user_id)."""
    from patchright.sync_api import sync_playwright
    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            print(f"[{time.strftime('%H:%M:%S')}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="gr_reg_")

    with sync_playwright() as p:
        import shutil
        channel = "chrome" if (shutil.which("google-chrome") or
                               shutil.which("google-chrome-stable")) else None
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
            # ── Step 1: Open signup page ──────────────────────────────────────
            log("opening signup page...")
            page.goto("https://www.goodreads.com/user/sign_up", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)
            log(f"url: {page.url}")

            # ── Step 2: Fill form ─────────────────────────────────────────────
            log(f"filling form: {first_name} {last_name} / {mailbox.address}")

            _fill_first(page, [
                'input[name="user[first_name]"]',
                'input[name*="first_name"]',
                'input[id*="first_name"]',
                'input[placeholder*="First" i]',
            ], first_name, log)
            _wait(0.3, log)

            _fill_first(page, [
                'input[name="user[last_name]"]',
                'input[name*="last_name"]',
                'input[id*="last_name"]',
                'input[placeholder*="Last" i]',
            ], last_name, log)
            _wait(0.3, log)

            _fill_first(page, [
                'input[name="user[email]"]',
                'input[type="email"]',
                'input[name*="email"]',
                'input[placeholder*="email" i]',
            ], mailbox.address, log)
            _wait(0.3, log)

            _fill_first(page, [
                'input[name="user[password]"]',
                'input[type="password"]',
                'input[name*="password"]',
            ], password, log)
            _wait(0.5, log)

            # ── Step 3: Submit ────────────────────────────────────────────────
            log("submitting signup form...")
            _click_first(page, [
                'input[type="submit"][value*="Sign up" i]',
                'input[type="submit"][value*="Register" i]',
                'button[type="submit"]:has-text("Sign up")',
                'button[type="submit"]:has-text("Register")',
                'button[type="submit"]',
                'input[type="submit"]',
            ], log)
            _wait(4, log, "waiting for post-submit")
            log(f"url after submit: {page.url}")

            # ── Step 4: Handle email verification ─────────────────────────────
            body_text = page.inner_text("body")[:600]
            verify_kws = ["verify", "confirm", "check your email", "we sent", "activation"]
            if any(kw in body_text.lower() for kw in verify_kws):
                log("email verification required — polling mail.tm...")
                link = mail_client.poll_for_verification_link(mailbox, timeout=120)
                log(f"got link: {link[:80]}...")
                try:
                    page.goto(link, timeout=30000)
                    page.wait_for_load_state("networkidle", timeout=15000)
                except Exception as e:
                    log(f"verification link nav warn: {e}")
                _wait(3, log, "post-verification")
                log(f"url after verification: {page.url}")

            # ── Step 5: If not logged in yet, log in ──────────────────────────
            if not _logged_in(page):
                log("not logged in after signup — attempting login...")
                try:
                    page.goto("https://www.goodreads.com/user/sign_in", timeout=20000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception as e:
                    log(f"login nav warn: {e}")
                _wait(1, log)

                _fill_first(page, [
                    'input[name="user[email]"]',
                    'input[type="email"]',
                    'input[name*="email"]',
                ], mailbox.address, log)
                _wait(0.3, log)

                _fill_first(page, [
                    'input[name="user[password]"]',
                    'input[type="password"]',
                ], password, log)
                _wait(0.3, log)

                _click_first(page, [
                    'input[type="submit"][value*="Sign in" i]',
                    'button[type="submit"]:has-text("Sign in")',
                    'button[type="submit"]',
                    'input[type="submit"]',
                ], log)
                _wait(4, log, "waiting for login")
                log(f"url after login: {page.url}")

            # ── Step 6: Verify we're logged in ────────────────────────────────
            if not _logged_in(page) and "goodreads.com" in page.url:
                # Navigate home — logged-in users land on /home
                try:
                    page.goto("https://www.goodreads.com/home", timeout=15000)
                    _wait(2, log)
                except Exception:
                    pass
                log(f"url after home nav: {page.url}")

            if "sign_in" in page.url or "sign_up" in page.url:
                raise RuntimeError(
                    f"Registration failed — still on auth page: {page.url}\n"
                    f"Page text: {page.inner_text('body')[:400]}"
                )

            # ── Step 7: Extract cookies + user_id ─────────────────────────────
            log("extracting cookies...")
            cookies  = ctx.cookies()
            user_id  = _extract_user_id(page, log)
            cookies_json = json.dumps(cookies)
            log(f"extracted {len(cookies)} cookies, user_id={user_id!r}")
            return cookies_json, user_id

        finally:
            ctx.close()
```

- [ ] **Commit**

```bash
git add tools/goodread/src/goodread_tool/browser.py
git commit -m "feat(goodread-tool): Patchright browser registration"
```

---

## Chunk 3: CLI + cookies export

### Task 6: cli.py

**Files:**
- Create: `tools/goodread/src/goodread_tool/cli.py`

Commands:
- `register [--no-headless] [--verbose]`
- `account ls`
- `account rm <email>`
- `test [email]` — makes an httpx GET to `/review/list/<user_id>?shelf=read` with cookies, verifies 200 + book table present
- `cookies export [email] [--out <path>]` — writes simplified cookies JSON

- [ ] **Create `cli.py`**

```python
"""Typer CLI: goodread register / account / test / cookies."""
from __future__ import annotations
import json, sys
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH, DEFAULT_COOKIES_PATH

app         = typer.Typer(name="goodread", help="Manage Goodreads accounts and session cookies.", no_args_is_help=True)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
cookies_app = typer.Typer(help="Manage session cookies.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(cookies_app, name="cookies")

console     = Console()
err_console = Console(stderr=True)


def _store() -> Store:
    return Store(DEFAULT_DB_PATH)


# ── register ──────────────────────────────────────────────────────────────────

@app.command()
def register(
    no_headless: Annotated[bool, typer.Option("--no-headless")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Auto-register a new Goodreads account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    identity    = generate()
    mail_client = MailTmClient(verbose=verbose)

    with console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    console.print("[bold green]Opening browser for Goodreads signup...[/bold green]")

    try:
        cookies_json, user_id = register_via_browser(
            mailbox=mailbox,
            mail_client=mail_client,
            first_name=identity.first_name,
            last_name=identity.last_name,
            password=identity.password,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Registration failed:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        mail_client.close()

    store = _store()
    store.add(
        email=mailbox.address,
        password=identity.password,
        cookies=cookies_json,
        user_id=user_id,
    )

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    if user_id:
        console.print(f"[dim]User ID:[/dim] {user_id}")
    console.print(f"[dim]Cookies:[/dim] {len(json.loads(cookies_json))} stored")
    console.print(f"[dim]DB:[/dim] {DEFAULT_DB_PATH}")


# ── account ───────────────────────────────────────────────────────────────────

@account_app.command("ls")
def account_ls() -> None:
    """List all registered accounts."""
    store = _store()
    rows  = store.list_all()
    if not rows:
        console.print("[yellow]No accounts registered.[/yellow]")
        return
    t = Table(title="Goodreads Accounts", show_lines=True)
    t.add_column("Email", style="cyan")
    t.add_column("User ID")
    t.add_column("Cookies", justify="right")
    t.add_column("Active", justify="center")
    t.add_column("Created")
    for r in rows:
        try:
            n_cookies = len(json.loads(r.get("cookies") or "[]"))
        except Exception:
            n_cookies = 0
        active  = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        t.add_row(r["email"], r["user_id"] or "-", str(n_cookies), active, created)
    console.print(t)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Account email to remove")],
) -> None:
    """Remove an account from local state."""
    store = _store()
    if not store.get(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.remove(email)
    console.print(f"[yellow]Removed:[/yellow] {email}")


# ── test ──────────────────────────────────────────────────────────────────────

@app.command()
def test(
    email: Annotated[Optional[str], typer.Argument(help="Account email (default: first active)")] = None,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Verify a registered account works by fetching a login-gated page."""
    import httpx

    store = _store()
    row   = store.get(email) if email else store.get_first_active()
    if not row:
        err_console.print("[bold red]No account found.[/bold red] Run: goodread register")
        raise typer.Exit(1)

    cookies_list = json.loads(row["cookies"] or "[]")
    user_id      = row["user_id"]

    if not cookies_list:
        err_console.print("[bold red]No cookies stored for this account.[/bold red]")
        raise typer.Exit(1)

    # Build httpx cookies dict (name → value, goodreads.com domain only)
    jar = {c["name"]: c["value"] for c in cookies_list if "goodreads" in c.get("domain", "")}
    if verbose:
        console.print(f"[dim]Using {len(jar)} cookies for {row['email']}[/dim]")

    # Test URL: either user's own shelf or a public shelf endpoint
    if user_id:
        test_url = f"https://www.goodreads.com/review/list/{user_id}?shelf=read&per_page=10"
    else:
        # Fallback: try the user's home page
        test_url = "https://www.goodreads.com/home"

    console.print(f"Testing: [cyan]{test_url}[/cyan]")

    headers = {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                      "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml",
        "Accept-Language": "en-US,en;q=0.9",
    }

    try:
        resp = httpx.get(test_url, cookies=jar, headers=headers,
                         follow_redirects=True, timeout=20)
    except Exception as e:
        err_console.print(f"[bold red]Request failed:[/bold red] {e}")
        raise typer.Exit(1)

    if verbose:
        console.print(f"[dim]HTTP {resp.status_code} — final URL: {resp.url}[/dim]")

    # Detect login redirect
    if "sign_in" in str(resp.url):
        err_console.print(
            "[bold red]FAIL[/bold red] — redirected to sign-in. "
            "Cookies may be expired. Re-register: goodread register"
        )
        raise typer.Exit(1)

    if resp.status_code != 200:
        err_console.print(f"[bold red]FAIL[/bold red] — HTTP {resp.status_code}")
        raise typer.Exit(1)

    # Check for sign-in indicators in page body
    body = resp.text
    if "Your books" in body or "bookalike" in body or "My Books" in body or user_id in body:
        console.print(f"[bold green]✓ Account works:[/bold green] {row['email']}")
    elif "sign_in" in body.lower() or "LoginInterstitial" in body:
        err_console.print("[bold red]FAIL[/bold red] — login interstitial detected in page body")
        raise typer.Exit(1)
    else:
        console.print(f"[yellow]WARN[/yellow] — HTTP 200 but could not confirm logged-in state")
        if verbose:
            console.print(f"[dim]{body[:300]}[/dim]")


# ── cookies ───────────────────────────────────────────────────────────────────

@cookies_app.command("export")
def cookies_export(
    email: Annotated[Optional[str], typer.Argument(help="Account email (default: first active)")] = None,
    out: Annotated[Optional[str], typer.Option("--out", "-o", help="Output path")] = None,
) -> None:
    """Export session cookies to JSON for the Go scraper.

    Writes to ~/data/goodread/cookies.json by default.
    The file contains a JSON array of {name, value, domain, path} objects.
    """
    store = _store()
    row   = store.get(email) if email else store.get_first_active()
    if not row:
        err_console.print("[bold red]No account found.[/bold red] Run: goodread register")
        raise typer.Exit(1)

    raw_cookies  = json.loads(row["cookies"] or "[]")
    if not raw_cookies:
        err_console.print("[bold red]No cookies stored for this account.[/bold red]")
        raise typer.Exit(1)

    # Simplify to what Go needs: name, value, domain, path
    export = [
        {
            "name":   c["name"],
            "value":  c["value"],
            "domain": c.get("domain", ".goodreads.com"),
            "path":   c.get("path", "/"),
        }
        for c in raw_cookies
    ]

    dest = Path(out) if out else DEFAULT_COOKIES_PATH
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(json.dumps(export, indent=2))

    console.print(f"[bold green]✓ Exported {len(export)} cookies[/bold green] → {dest}")
    console.print(f"[dim]Account:[/dim] {row['email']}")


# ── entry point ───────────────────────────────────────────────────────────────

def app_entry() -> None:
    app()

if __name__ == "__main__":
    app_entry()
```

- [ ] **Run `uv run goodread --help` — verify all commands appear**

```bash
cd tools/goodread && uv run goodread --help
# Expected: register, account, test, cookies commands listed
uv run goodread account --help
uv run goodread cookies --help
```

- [ ] **Commit**

```bash
git add tools/goodread/src/goodread_tool/cli.py
git commit -m "feat(goodread-tool): Typer CLI (register, account, test, cookies export)"
```

---

### Task 7: PyInstaller spec

**Files:**
- Create: `tools/goodread/goodread-tool.spec`

- [ ] **Create spec** (mirror motherduck's spec, adapted for goodread_tool module)

```python
# -*- mode: python ; coding: utf-8 -*-
from PyInstaller.utils.hooks import collect_all, collect_submodules

datas, binaries, hiddenimports = [], [], []
for pkg in ('duckdb', 'patchright', 'faker', 'httpx', 'typer', 'rich'):
    tmp = collect_all(pkg)
    datas += tmp[0]; binaries += tmp[1]; hiddenimports += tmp[2]

hiddenimports += collect_submodules('goodread_tool')

a = Analysis(
    ['goodread_entry.py'],
    pathex=['src'],
    binaries=binaries,
    datas=datas,
    hiddenimports=hiddenimports,
    hookspath=[],
    runtime_hooks=[],
    excludes=['tkinter', 'matplotlib', 'numpy', 'pandas', 'scipy'],
    noarchive=False,
)
pyz = PYZ(a.pure)
exe = EXE(pyz, a.scripts, a.binaries, a.datas, [],
    name='goodread-tool', debug=False, strip=False, upx=False, console=True)
```

- [ ] **Run full test suite**

```bash
cd tools/goodread && uv run pytest tests/ -v
# Expected: all tests PASS
```

- [ ] **Commit**

```bash
git add tools/goodread/goodread-tool.spec
git commit -m "feat(goodread-tool): PyInstaller spec + all tests passing"
```

---

## Chunk 4: Go — cookies.go + authenticated client + search/shelf

### Task 8: pkg/scrape/goodread/cookies.go

**Files:**
- Create: `pkg/scrape/goodread/cookies.go`

- [ ] **Create `cookies.go`**

The Playwright cookie format is:
```json
[{"name":"...", "value":"...", "domain":".goodreads.com", "path":"/", ...}]
```

```go
package goodread

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultCookiesPath is where `goodread cookies export` writes cookies.json.
var DefaultCookiesPath = filepath.Join(os.Getenv("HOME"), "data", "goodread", "cookies.json")

type exportedCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// LoadCookiesFromFile reads the cookies JSON file written by `goodread cookies export`
// and returns them as []*http.Cookie ready for use with an http.CookieJar.
func LoadCookiesFromFile(path string) ([]*http.Cookie, error) {
	if path == "" {
		path = DefaultCookiesPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cookies file %s: %w", path, err)
	}
	var raw []exportedCookie
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse cookies JSON: %w", err)
	}
	var cookies []*http.Cookie
	for _, c := range raw {
		if c.Name == "" || c.Value == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
			// Set a generous expiry so cookies are not considered expired
			Expires: time.Now().Add(30 * 24 * time.Hour),
		})
	}
	return cookies, nil
}
```

- [ ] **Build to verify no compile errors**

```bash
cd /path/to/blueprints/search && go build ./pkg/scrape/goodread/...
# Expected: no errors (ld warning about duplicate libraries is OK on macOS)
```

- [ ] **Commit**

```bash
git add pkg/scrape/goodread/cookies.go
git commit -m "feat(goodread): LoadCookiesFromFile for authenticated client"
```

---

### Task 9: client.go — NewClientWithCookies

**Files:**
- Modify: `pkg/scrape/goodread/client.go`

A `http.CookieJar` needs to be populated with the cookies against the goodreads.com URL. Go's `cookiejar.New` from `net/http/cookiejar` handles domain matching automatically.

- [ ] **Add `NewClientWithCookies` to `client.go`**

Add these imports and function — place after `NewClient`:

```go
import (
    // add to existing imports:
    "net/http/cookiejar"
    "net/url"
)
```

```go
// NewClientWithCookies creates an authenticated Goodreads HTTP client using
// pre-loaded session cookies (from LoadCookiesFromFile).
func NewClientWithCookies(cfg Config, cookies []*http.Cookie) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	// Set cookies against the goodreads.com URL so they're sent on every request.
	u, _ := url.Parse("https://www.goodreads.com")
	jar.SetCookies(u, cookies)

	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     cfg.Workers + 2,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	return &Client{
		http: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			Jar:       jar,
		},
		userAgents: userAgents,
		delay:      cfg.Delay,
	}, nil
}
```

- [ ] **Build to verify**

```bash
go build ./pkg/scrape/goodread/...
```

- [ ] **Commit**

```bash
git add pkg/scrape/goodread/client.go
git commit -m "feat(goodread): NewClientWithCookies with pre-populated jar"
```

---

### Task 10: parse_search.go — ParseSearchHTML for logged-in search

**Files:**
- Modify: `pkg/scrape/goodread/parse_search.go`

When logged in, Goodreads serves the real search results page with server-side rendered HTML. The legacy `tr.bookContainer` selector works. Each row has:
- `a[href*='/book/show/']` — book URL + title
- `a[href*='/author/show/']` — author name

Pagination: `a.next_page` or `a[href*='page='][rel='next']`.

- [ ] **Add `ParseSearchHTML` and `ParseSearchNextPage` (restore for authenticated use)**

Append to `parse_search.go`:

```go
// ParseSearchHTML parses a Goodreads HTML search results page (requires login).
// Endpoint: GET /search?q=<query>&search[field]=books&page=N
//
// The HTML structure (tr.bookContainer) is server-side rendered only for
// authenticated sessions. Anonymous requests return a React login interstitial.
func ParseSearchHTML(doc *goquery.Document) []SearchResult {
	var results []SearchResult
	seen := map[string]bool{}

	doc.Find("tr.bookContainer").Each(func(_ int, row *goquery.Selection) {
		var r SearchResult
		// Book link + title
		row.Find("a[href*='/book/show/']").First().Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			if id := extractIDFromPath(href, "/book/show/"); id != "" {
				r.URL        = BaseURL + "/book/show/" + id
				r.EntityType = "book"
				r.Title      = strings.TrimSpace(a.Text())
			}
		})
		if r.URL != "" && !seen[r.URL] {
			seen[r.URL] = true
			results = append(results, r)
		}

		// Author inside the same row
		row.Find("a[href*='/author/show/']").First().Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			if id := extractIDFromPath(href, "/author/show/"); id != "" {
				authorURL := BaseURL + "/author/show/" + id
				if !seen[authorURL] {
					seen[authorURL] = true
					results = append(results, SearchResult{
						URL:        authorURL,
						EntityType: "author",
						Title:      strings.TrimSpace(a.Text()),
					})
				}
			}
		})
	})

	return results
}

// ParseSearchHTMLNextPage returns the next-page URL from an authenticated search page.
func ParseSearchHTMLNextPage(doc *goquery.Document) string {
	var next string
	doc.Find("a.next_page, a[rel='next'][href*='page=']").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if href != "" {
			if strings.HasPrefix(href, "/") {
				next = BaseURL + href
			} else {
				next = href
			}
		}
	})
	return next
}
```

Add `"github.com/PuerkitoBio/goquery"` to the imports (it's already there via the autocomplete function's `strings` import — but goquery must be added since ParseSearchHTML uses it). Also add `"strings"` if not present.

- [ ] **Verify `extractIDFromPath` exists** — it's in one of the other parse_*.go files:

```bash
grep -n "extractIDFromPath" pkg/scrape/goodread/*.go
```

- [ ] **Build**

```bash
go build ./pkg/scrape/goodread/...
```

- [ ] **Commit**

```bash
git add pkg/scrape/goodread/parse_search.go
git commit -m "feat(goodread): ParseSearchHTML + ParseSearchHTMLNextPage for authenticated search"
```

---

### Task 11: cli/goodread.go — --auth flags on search and shelf

**Files:**
- Modify: `cli/goodread.go`

#### search command — add `--auth` flag

When `--auth` is set:
1. Load cookies from `DefaultCookiesPath` (or `--cookies-file`)
2. Create `NewClientWithCookies`
3. Paginate through `/search?q=...&search[field]=books&page=N` using `ParseSearchHTML`
4. Up to `--max-pages` pages (default 5, max 10, ~20 results/page → 100 results/call)

When `--auth` is not set: existing autocomplete behavior (no change).

- [ ] **Update `newGoodreadSearch()` in `cli/goodread.go`**

Replace the existing `newGoodreadSearch()` function:

```go
func newGoodreadSearch() *cobra.Command {
	var dbPath, statePath string
	var delay int
	var useAuth bool
	var cookiesFile string
	var maxPages int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Goodreads and enqueue results",
		Long: `Search Goodreads and enqueue results.

Without --auth (default): uses the autocomplete API, returns up to ~20 results, no pagination.

With --auth: uses authenticated HTML search (/search?q=...) with full pagination.
Requires cookies exported by the goodread Python tool:
  uv run goodread cookies export   # writes ~/data/goodread/cookies.json`,
		Args: cobra.ExactArgs(1),
		Example: `  search goodread search "Dune"
  search goodread search "Frank Herbert" --auth
  search goodread search "fantasy" --auth --max-pages 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			query := args[0]

			if !useAuth {
				// Autocomplete path (no login needed)
				cfg := goodread.DefaultConfig()
				cfg.Delay = time.Duration(delay) * time.Millisecond
				client := goodread.NewClient(cfg)

				apiURL := goodread.BaseURL + "/book/auto_complete?format=json&q=" + url.QueryEscape(query)
				fmt.Printf("Searching (autocomplete): %s\n", apiURL)

				body, code, err := client.Fetch(cmd.Context(), apiURL)
				if err != nil {
					return fmt.Errorf("fetch: %w", err)
				}
				if code != 200 {
					return fmt.Errorf("unexpected HTTP %d", code)
				}
				results := goodread.ParseSearchAutocomplete(body)
				if len(results) == 0 {
					fmt.Println("No results found.")
					return nil
				}
				total := 0
				for _, r := range results {
					if stateDB.Enqueue(r.URL, r.EntityType, 5) == nil {
						fmt.Printf("  Enqueued [%s] %s\n", r.EntityType, r.Title)
						total++
					}
				}
				fmt.Printf("Enqueued %d URLs\n", total)
				return nil
			}

			// Authenticated HTML search
			cookiePath := cookiesFile
			if cookiePath == "" {
				cookiePath = goodread.DefaultCookiesPath
			}
			cookies, err := goodread.LoadCookiesFromFile(cookiePath)
			if err != nil {
				return fmt.Errorf("load cookies: %w\nRun: uv run goodread cookies export", err)
			}

			cfg := goodread.DefaultConfig()
			cfg.Delay = time.Duration(delay) * time.Millisecond
			client, err := goodread.NewClientWithCookies(cfg, cookies)
			if err != nil {
				return fmt.Errorf("create authenticated client: %w", err)
			}

			searchURL := goodread.BaseURL + "/search?q=" + url.QueryEscape(query) + "&search%5Bfield%5D=books"
			fmt.Printf("Searching (authenticated): %s\n", searchURL)

			total := 0
			currentURL := searchURL
			for page := 1; page <= maxPages; page++ {
				pageURL := currentURL
				if page > 1 {
					pageURL = searchURL + fmt.Sprintf("&page=%d", page)
				}
				doc, code, err := client.FetchHTML(cmd.Context(), pageURL)
				if err != nil {
					return fmt.Errorf("page %d: %w", page, err)
				}
				if code != 200 || doc == nil {
					break
				}

				results := goodread.ParseSearchHTML(doc)
				if len(results) == 0 {
					fmt.Printf("  Page %d: no results (end of results or login expired)\n", page)
					break
				}

				for _, r := range results {
					if stateDB.Enqueue(r.URL, r.EntityType, 5) == nil {
						fmt.Printf("  [%s] %s\n", r.EntityType, r.Title)
						total++
					}
				}
				fmt.Printf("  Page %d: +%d results (total=%d)\n", page, len(results), total)

				next := goodread.ParseSearchHTMLNextPage(doc)
				if next == "" {
					break
				}
			}

			fmt.Printf("Enqueued %d URLs\n", total)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().BoolVar(&useAuth, "auth", false, "Use authenticated HTML search (requires exported cookies)")
	cmd.Flags().StringVar(&cookiesFile, "cookies-file", "", "Path to cookies.json (default: ~/data/goodread/cookies.json)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 5, "Max pages to fetch in authenticated mode (20 results/page)")
	return cmd
}
```

#### shelf command — add `--auth` flag

- [ ] **Update `newGoodreadShelf()` to add `--auth` and `--cookies-file` flags**

Add to the `RunE` block — before `openDBs`, check if `--auth` is set and build the client accordingly. Replace the `client := goodread.NewClient(cfg)` line in the shelf task:

```go
// In newGoodreadShelf(), add these vars:
var useAuth bool
var cookiesFile string

// In RunE, replace:
//   client := goodread.NewClient(cfg)
// with:
var client *goodread.Client
if useAuth {
    cookiePath := cookiesFile
    if cookiePath == "" {
        cookiePath = goodread.DefaultCookiesPath
    }
    cookies, err := goodread.LoadCookiesFromFile(cookiePath)
    if err != nil {
        return fmt.Errorf("load cookies: %w\nRun: uv run goodread cookies export", err)
    }
    client, err = goodread.NewClientWithCookies(cfg, cookies)
    if err != nil {
        return fmt.Errorf("create authenticated client: %w", err)
    }
} else {
    client = goodread.NewClient(cfg)
}

// Add at end (after addDBFlags):
cmd.Flags().BoolVar(&useAuth, "auth", false, "Use authenticated session (requires exported cookies)")
cmd.Flags().StringVar(&cookiesFile, "cookies-file", "", "Path to cookies.json (default: ~/data/goodread/cookies.json)")
```

- [ ] **Build to verify no errors**

```bash
go build -o /tmp/search-test ./cmd/search/ 2>&1
```

- [ ] **Smoke test search (no auth — existing behavior must still work)**

```bash
/tmp/search-test goodread search "Dune"
# Expected: Enqueued 7 URLs (autocomplete, unchanged)
```

- [ ] **Smoke test shelf (no auth — should still get 401 login error)**

```bash
/tmp/search-test goodread shelf 1
# Expected: [failed] page=0 books=0 (login required)
```

- [ ] **Commit**

```bash
git add cli/goodread.go pkg/scrape/goodread/parse_search.go pkg/scrape/goodread/cookies.go pkg/scrape/goodread/client.go
git commit -m "feat(goodread): --auth flag on search+shelf, LoadCookiesFromFile, ParseSearchHTML"
```

---

## End-to-end test (manual, requires browser)

After all tasks complete, run the full registration + verification flow:

```bash
# 1. Register a Goodreads account
cd tools/goodread
uv run goodread register --verbose

# 2. Verify the account works
uv run goodread test --verbose

# 3. Export cookies for Go
uv run goodread cookies export
# → writes ~/data/goodread/cookies.json

# 4. Test authenticated search from Go CLI
search goodread search "Brandon Sanderson" --auth --max-pages 3
# Expected: ~3×20 = 60 results, book + author URLs enqueued

# 5. Test authenticated shelf from Go CLI (use the registered user_id)
search goodread shelf <user_id> --auth
# Expected: books listed (likely 0 on a fresh account, but NO 401 error)
```

If registration fails at the signup form (layout changed), run with `--no-headless --verbose` to debug.
