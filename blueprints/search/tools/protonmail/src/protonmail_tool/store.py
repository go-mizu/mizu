"""DuckDB-backed store for Proton Mail accounts."""
from __future__ import annotations

from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "protonmail" / "accounts.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email        VARCHAR NOT NULL UNIQUE,   -- full address e.g. user@proton.me
    username     VARCHAR NOT NULL,
    password     VARCHAR NOT NULL,
    display_name VARCHAR DEFAULT '',
    created_at   TIMESTAMP DEFAULT now(),
    is_active    BOOLEAN DEFAULT true
);
"""


class Store:
    def __init__(self, path: str | Path = DEFAULT_DB_PATH) -> None:
        if str(path) != ":memory:":
            Path(path).parent.mkdir(parents=True, exist_ok=True)
        self.con = duckdb.connect(str(path))
        self.con.execute(_SCHEMA)

    def add(self, *, username: str, password: str, display_name: str = "") -> str:
        email = f"{username}@proton.me"
        row = self.con.execute(
            "INSERT INTO accounts (email, username, password, display_name) "
            "VALUES (?, ?, ?, ?) RETURNING id",
            [email, username, password, display_name],
        ).fetchone()
        return row[0]

    def list_all(self) -> list[dict[str, Any]]:
        rows = self.con.execute(
            "SELECT id, email, username, password, display_name, created_at, is_active "
            "FROM accounts ORDER BY created_at DESC"
        ).fetchall()
        cols = ["id", "email", "username", "password", "display_name", "created_at", "is_active"]
        return [dict(zip(cols, r)) for r in rows]

    def get(self, email_or_username: str) -> dict[str, Any] | None:
        val = email_or_username
        if "@" not in val:
            val = f"{val}@proton.me"
        row = self.con.execute(
            "SELECT id, email, username, password, display_name FROM accounts WHERE email = ?",
            [val],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "username", "password", "display_name"], row))

    def get_first_active(self) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, username, password, display_name FROM accounts "
            "WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "username", "password", "display_name"], row))

    def remove(self, email_or_username: str) -> None:
        val = email_or_username
        if "@" not in val:
            val = f"{val}@proton.me"
        self.con.execute("DELETE FROM accounts WHERE email = ?", [val])

    def close(self) -> None:
        self.con.close()
