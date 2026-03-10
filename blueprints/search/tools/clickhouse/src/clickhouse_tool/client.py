"""ClickHouse query client using clickhouse-connect."""
from __future__ import annotations


class ClickHouseClient:
    def __init__(self, host: str, port: int = 8443,
                 username: str = "default", password: str = ""):
        import clickhouse_connect
        self._client = clickhouse_connect.get_client(
            host=host, port=port,
            username=username, password=password,
            secure=True,
        )

    def run_query(self, sql: str) -> tuple[list, list]:
        result = self._client.query(sql)
        return list(result.result_rows), list(result.column_names)

    def create_db(self, name: str) -> None:
        self._client.command(f"CREATE DATABASE IF NOT EXISTS {name}")

    def close(self) -> None:
        self._client.close()
