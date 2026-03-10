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
