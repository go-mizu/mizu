"""Local DuckDB state for ClickHouse Cloud accounts, services, and query log."""
from __future__ import annotations

from pathlib import Path

import duckdb

DEFAULT_DB_PATH = Path.home() / "data" / "clickhouse" / "clickhouse.duckdb"

_SCHEMA = """
CREATE TABLE IF NOT EXISTS accounts (
    id          VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email       VARCHAR NOT NULL UNIQUE,
    password    VARCHAR NOT NULL,
    org_id      VARCHAR DEFAULT '',
    api_key_id  VARCHAR DEFAULT '',
    api_key_secret VARCHAR DEFAULT '',
    created_at  TIMESTAMP DEFAULT now(),
    is_active   BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS services (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    cloud_id     VARCHAR DEFAULT '',
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    host         VARCHAR DEFAULT '',
    port         INTEGER DEFAULT 8443,
    db_user      VARCHAR DEFAULT 'default',
    db_password  VARCHAR DEFAULT '',
    provider     VARCHAR DEFAULT 'aws',
    region       VARCHAR DEFAULT 'us-east-1',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    service_id    VARCHAR REFERENCES services(id),
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
"""


class Store:
    def __init__(self, db_path: Path = DEFAULT_DB_PATH):
        db_path.parent.mkdir(parents=True, exist_ok=True)
        self.con = duckdb.connect(str(db_path))
        for stmt in _SCHEMA.strip().split(";"):
            stmt = stmt.strip()
            if stmt:
                self.con.execute(stmt)

    # ---- accounts ----

    def add_account(
        self, email: str, password: str,
        org_id: str = "", api_key_id: str = "", api_key_secret: str = "",
    ) -> None:
        self.con.execute(
            "INSERT INTO accounts (email, password, org_id, api_key_id, api_key_secret) "
            "VALUES (?, ?, ?, ?, ?)",
            [email, password, org_id, api_key_id, api_key_secret],
        )

    def list_accounts(self) -> list[dict]:
        rows = self.con.execute(
            "SELECT a.email, "
            "  (SELECT COUNT(*) FROM services s WHERE s.account_id = a.id) AS svc_count, "
            "  a.is_active, a.created_at, a.org_id "
            "FROM accounts a ORDER BY a.created_at DESC"
        ).fetchall()
        return [
            {"email": r[0], "svc_count": r[1], "is_active": r[2],
             "created_at": r[3], "org_id": r[4]}
            for r in rows
        ]

    def deactivate_account(self, email: str) -> None:
        self.con.execute("UPDATE accounts SET is_active = false WHERE email = ?", [email])

    def get_first_active_account(self) -> dict | None:
        row = self.con.execute(
            "SELECT id, email, api_key_id, api_key_secret, org_id FROM accounts "
            "WHERE is_active = true ORDER BY created_at LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return {"id": row[0], "email": row[1], "api_key_id": row[2],
                "api_key_secret": row[3], "org_id": row[4]}

    def get_account_by_email(self, email: str) -> dict | None:
        row = self.con.execute(
            "SELECT id, email, api_key_id, api_key_secret, org_id FROM accounts "
            "WHERE email = ?", [email]
        ).fetchone()
        if not row:
            return None
        return {"id": row[0], "email": row[1], "api_key_id": row[2],
                "api_key_secret": row[3], "org_id": row[4]}

    # ---- services ----

    def add_service(
        self, account_id: str, name: str, alias: str,
        cloud_id: str = "", host: str = "", port: int = 8443,
        db_user: str = "default", db_password: str = "",
        provider: str = "aws", region: str = "us-east-1",
    ) -> None:
        self.con.execute(
            "INSERT INTO services (account_id, cloud_id, name, alias, host, port, "
            "db_user, db_password, provider, region) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            [account_id, cloud_id, name, alias, host, port,
             db_user, db_password, provider, region],
        )

    def list_services(self) -> list[dict]:
        rows = self.con.execute(
            "SELECT s.alias, s.name, a.email, s.is_default, s.host, "
            "  (SELECT COUNT(*) FROM query_log q WHERE q.service_id = s.id) AS query_count, "
            "  s.last_used_at, s.created_at "
            "FROM services s JOIN accounts a ON s.account_id = a.id "
            "ORDER BY s.is_default DESC, s.created_at DESC"
        ).fetchall()
        return [
            {"alias": r[0], "name": r[1], "email": r[2], "is_default": r[3],
             "host": r[4], "query_count": r[5], "last_used_at": r[6], "created_at": r[7]}
            for r in rows
        ]

    def get_service(self, alias: str) -> dict | None:
        row = self.con.execute(
            "SELECT s.id, s.name, s.alias, s.host, s.port, s.db_user, s.db_password, "
            "  s.is_default, s.cloud_id, a.api_key_id, a.api_key_secret, a.org_id "
            "FROM services s JOIN accounts a ON s.account_id = a.id "
            "WHERE s.alias = ?", [alias]
        ).fetchone()
        if not row:
            return None
        return {
            "id": row[0], "name": row[1], "alias": row[2], "host": row[3],
            "port": row[4], "db_user": row[5], "db_password": row[6],
            "is_default": row[7], "cloud_id": row[8],
            "api_key_id": row[9], "api_key_secret": row[10], "org_id": row[11],
        }

    def get_default_service(self) -> dict | None:
        row = self.con.execute(
            "SELECT s.alias FROM services s WHERE s.is_default = true LIMIT 1"
        ).fetchone()
        if not row:
            return None
        return self.get_service(row[0])

    def set_default(self, alias: str) -> None:
        self.con.execute("BEGIN")
        try:
            self.con.execute("UPDATE services SET is_default = false WHERE is_default = true")
            self.con.execute("UPDATE services SET is_default = true WHERE alias = ?", [alias])
            self.con.execute("COMMIT")
        except Exception:
            self.con.execute("ROLLBACK")
            raise

    def remove_service(self, alias: str) -> None:
        self.con.execute("DELETE FROM services WHERE alias = ?", [alias])

    def touch_last_used(self, alias: str) -> None:
        self.con.execute(
            "UPDATE services SET last_used_at = now() WHERE alias = ?", [alias]
        )

    # ---- query log ----

    def log_query(self, service_id: str, sql: str,
                  rows_returned: int, duration_ms: int) -> None:
        self.con.execute(
            "INSERT INTO query_log (service_id, sql, rows_returned, duration_ms) "
            "VALUES (?, ?, ?, ?)",
            [service_id, sql, rows_returned, duration_ms],
        )
