"""Local DuckDB state: schema init, CRUD for Goodreads accounts and cookies."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "goodread" / "accounts.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id         VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email      VARCHAR NOT NULL UNIQUE,
    password   VARCHAR NOT NULL,
    user_id    VARCHAR DEFAULT '',
    cookies    VARCHAR DEFAULT '[]',
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

    # ------------------------------------------------------------------
    # Accounts
    # ------------------------------------------------------------------

    def add_account(self, *, email: str, password: str) -> str:
        row = self.con.execute(
            "INSERT INTO accounts (email, password) VALUES (?, ?) RETURNING id",
            [email, password],
        ).fetchone()
        return row[0]

    def update_cookies(self, email: str, cookies: list[dict]) -> None:
        self.con.execute(
            "UPDATE accounts SET cookies = ? WHERE email = ?",
            [json.dumps(cookies), email],
        )

    def update_user_id(self, email: str, user_id: str) -> None:
        self.con.execute(
            "UPDATE accounts SET user_id = ? WHERE email = ?",
            [user_id, email],
        )

    def list_accounts(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            """
            SELECT id, email, password, user_id, created_at, is_active
            FROM accounts
            ORDER BY created_at DESC
            """
        ).fetchall()
        cols = ["id", "email", "password", "user_id", "created_at", "is_active"]
        return [dict(zip(cols, r)) for r in rows]

    def get_first_active(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, user_id, cookies FROM accounts WHERE is_active = true ORDER BY created_at DESC LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "user_id", "cookies"], row))

    def get_by_email(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, user_id, cookies FROM accounts WHERE email = ? AND is_active = true",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "user_id", "cookies"], row))

    def deactivate(self, email: str) -> None:
        self.con.execute(
            "UPDATE accounts SET is_active = false WHERE email = ?", [email]
        )

    def get_cookies(self, email: str) -> list[dict]:
        """Return parsed cookies list for an account."""
        row = self.con.execute(
            "SELECT cookies FROM accounts WHERE email = ? AND is_active = true",
            [email],
        ).fetchone()
        if not row or not row[0]:
            return []
        try:
            return json.loads(row[0])
        except Exception:
            return []

    def export_cookies_file(self, email: str | None, path: str | Path) -> str:
        """Write cookies JSON to path. Returns the email used."""
        acct = self.get_by_email(email) if email else self.get_first_active()
        if not acct:
            raise ValueError("No active account found")
        cookies = json.loads(acct["cookies"]) if acct["cookies"] else []
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(cookies, indent=2))
        return acct["email"]

    def close(self) -> None:
        self.con.close()
