"""Local DuckDB state: Discord account credentials and tokens."""
from __future__ import annotations

from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "discord" / "accounts.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id         VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email      VARCHAR NOT NULL UNIQUE,
    username   VARCHAR NOT NULL,
    password   VARCHAR NOT NULL,
    token      VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    is_active  BOOLEAN DEFAULT true,
    notes      VARCHAR DEFAULT ''
);
"""


class Store:
    def __init__(self, path: str | Path = DEFAULT_DB_PATH) -> None:
        if str(path) != ":memory:":
            Path(path).parent.mkdir(parents=True, exist_ok=True)
        self.con = duckdb.connect(str(path))
        self.con.execute(_SCHEMA)

    def add_account(self, *, email: str, username: str, password: str, token: str, notes: str = "") -> str:
        row = self.con.execute(
            "INSERT INTO accounts (email, username, password, token, notes) VALUES (?, ?, ?, ?, ?) RETURNING id",
            [email, username, password, token, notes],
        ).fetchone()
        return row[0]

    def update_token(self, email: str, token: str) -> None:
        self.con.execute(
            "UPDATE accounts SET token = ? WHERE email = ?", [token, email]
        )

    def list_accounts(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            "SELECT id, email, username, token, created_at, is_active FROM accounts ORDER BY created_at DESC"
        ).fetchall()
        cols = ["id", "email", "username", "token", "created_at", "is_active"]
        return [dict(zip(cols, r)) for r in rows]

    def get_account(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, username, password, token FROM accounts WHERE email = ?",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "username", "password", "token"], row))

    def get_first_active(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, username, token FROM accounts WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "username", "token"], row))

    def deactivate(self, email: str) -> None:
        self.con.execute("UPDATE accounts SET is_active = false WHERE email = ?", [email])

    def remove(self, email: str) -> None:
        self.con.execute("DELETE FROM accounts WHERE email = ?", [email])

    def close(self) -> None:
        self.con.close()
