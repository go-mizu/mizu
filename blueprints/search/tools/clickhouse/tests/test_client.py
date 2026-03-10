"""Tests for clickhouse_tool.client with mocked clickhouse_connect."""
from unittest.mock import MagicMock, patch
import pytest


@patch.dict("sys.modules", {"clickhouse_connect": MagicMock()})
def test_run_query():
    import sys
    mock_cc = sys.modules["clickhouse_connect"]
    mock_client = MagicMock()
    mock_cc.get_client.return_value = mock_client

    result_mock = MagicMock()
    result_mock.result_rows = [(1, "hello")]
    result_mock.column_names = ["id", "msg"]
    mock_client.query.return_value = result_mock

    from clickhouse_tool.client import ClickHouseClient
    client = ClickHouseClient(host="h.cloud", port=8443, password="pw")
    rows, cols = client.run_query("SELECT 1")

    assert rows == [(1, "hello")]
    assert cols == ["id", "msg"]
    mock_client.query.assert_called_with("SELECT 1")


@patch.dict("sys.modules", {"clickhouse_connect": MagicMock()})
def test_create_db():
    import sys
    mock_cc = sys.modules["clickhouse_connect"]
    mock_client = MagicMock()
    mock_cc.get_client.return_value = mock_client

    from clickhouse_tool.client import ClickHouseClient
    client = ClickHouseClient(host="h.cloud", port=8443, password="pw")
    client.create_db("mydb")

    mock_client.command.assert_called_with("CREATE DATABASE IF NOT EXISTS mydb")


@patch.dict("sys.modules", {"clickhouse_connect": MagicMock()})
def test_close():
    import sys
    mock_cc = sys.modules["clickhouse_connect"]
    mock_client = MagicMock()
    mock_cc.get_client.return_value = mock_client

    from clickhouse_tool.client import ClickHouseClient
    client = ClickHouseClient(host="h.cloud", port=8443, password="pw")
    client.close()

    mock_client.close.assert_called_once()
