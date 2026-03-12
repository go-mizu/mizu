"""Unit tests for client.py — MotherDuck connection wrapper."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch, call


def _make_mock_con(fetchall_return=None, fetchone_return=None, description=None):
    con = MagicMock()
    cursor = MagicMock()
    cursor.fetchall.return_value = fetchall_return or []
    cursor.fetchone.return_value = fetchone_return
    cursor.description = description or [("col1",), ("col2",)]
    con.execute.return_value = cursor
    return con


def test_create_db_executes_create(monkeypatch):
    from motherduck.client import MotherDuckClient

    mock_con = _make_mock_con()
    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        client.create_db("mydb")

    calls = [str(c) for c in mock_con.execute.call_args_list]
    assert any("CREATE DATABASE" in c and "mydb" in c for c in calls)


def test_run_query_returns_rows(monkeypatch):
    from motherduck.client import MotherDuckClient

    desc = [("a",), ("b",)]
    mock_con = _make_mock_con(
        fetchall_return=[(1, "x"), (2, "y")],
        description=desc,
    )
    mock_con.execute.return_value.description = desc
    mock_con.execute.return_value.fetchall.return_value = [(1, "x"), (2, "y")]

    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        rows, cols = client.run_query("mydb", "SELECT a, b FROM t")

    assert cols == ["a", "b"]
    assert rows == [(1, "x"), (2, "y")]


def test_run_query_uses_correct_db(monkeypatch):
    from motherduck.client import MotherDuckClient

    mock_con = _make_mock_con()
    with patch("duckdb.connect", return_value=mock_con):
        client = MotherDuckClient(token="tok123")
        client.run_query("targetdb", "SELECT 1")

    calls = [str(c) for c in mock_con.execute.call_args_list]
    assert any("USE" in c and "targetdb" in c for c in calls)
