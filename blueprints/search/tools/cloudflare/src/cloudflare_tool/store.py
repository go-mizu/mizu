"""Local DuckDB state: schema init, CRUD for accounts/tokens/workers/op_log."""
from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "cloudflare" / "cloudflare.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id              VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email           VARCHAR NOT NULL UNIQUE,
    password        VARCHAR NOT NULL,
    account_id      VARCHAR NOT NULL,
    global_api_key  VARCHAR DEFAULT '',
    subdomain       VARCHAR DEFAULT '',
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMP DEFAULT now()
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
        self._migrate()

    def _migrate(self) -> None:
        """Run schema migrations for existing databases."""
        try:
            self.con.execute(
                "ALTER TABLE accounts ADD COLUMN IF NOT EXISTS global_api_key VARCHAR DEFAULT ''"
            )
        except Exception:
            pass  # Column already exists or table doesn't exist yet

    # ------------------------------------------------------------------
    # Accounts
    # ------------------------------------------------------------------

    def add_account(
        self, *, email: str, password: str, account_id: str,
        global_api_key: str = "", subdomain: str = "",
    ) -> str:
        row = self.con.execute(
            "INSERT INTO accounts (email, password, account_id, global_api_key, subdomain) "
            "VALUES (?, ?, ?, ?, ?) RETURNING id",
            [email, password, account_id, global_api_key, subdomain],
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
            "SELECT id, email, password, account_id, global_api_key, subdomain "
            "FROM accounts WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "account_id", "global_api_key", "subdomain"], row))

    def get_account_by_email(self, email: str) -> dict[str, Any] | None:
        row = self.con.execute(
            "SELECT id, email, password, account_id, global_api_key, subdomain "
            "FROM accounts WHERE email = ? AND is_active = true",
            [email],
        ).fetchone()
        if not row:
            return None
        return dict(zip(["id", "email", "password", "account_id", "global_api_key", "subdomain"], row))

    def update_global_api_key(self, email: str, api_key: str) -> None:
        self.con.execute(
            "UPDATE accounts SET global_api_key = ? WHERE email = ?", [api_key, email]
        )

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
        if not self.get_token_by_name(name):
            raise ValueError(f"Token not found: {name!r}")
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
        if not self.get_worker(alias):
            raise ValueError(f"Worker not found: {alias!r}")
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
