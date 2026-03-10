"""Local DuckDB state: schema init, CRUD for accounts/databases/query_log."""
from __future__ import annotations

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
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
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
        row = self.con.execute(
            "INSERT INTO accounts (email, password, token) VALUES (?, ?, ?) RETURNING id",
            [email, password, token],
        ).fetchone()
        return row[0]

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

    def get_account_by_email(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, token FROM accounts WHERE email = ? AND is_active = true",
            [email],
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
        row = self.con.execute(
            "INSERT INTO databases (account_id, name, alias, notes) VALUES (?, ?, ?, ?) RETURNING id",
            [account_id, name, alias, notes],
        ).fetchone()
        return row[0]

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
        # Atomic: clear all then set one — wrapped in transaction
        self.con.execute("BEGIN")
        try:
            self.con.execute("UPDATE databases SET is_default = false")
            self.con.execute(
                "UPDATE databases SET is_default = true WHERE alias = ?", [alias]
            )
            self.con.execute("COMMIT")
        except Exception:
            self.con.execute("ROLLBACK")
            raise

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
