"""Unit tests for store.py — uses in-memory DuckDB."""
from __future__ import annotations

import pytest
import duckdb

from motherduck.store import Store


@pytest.fixture
def store():
    """In-memory store for tests."""
    return Store(":memory:")


def test_init_creates_tables(store):
    tables = store.con.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
    ).fetchall()
    names = {r[0] for r in tables}
    assert "accounts" in names
    assert "databases" in names
    assert "query_log" in names


def test_add_account(store):
    acc_id = store.add_account(email="test@x.com", password="pass", token="tok123")
    assert acc_id is not None
    rows = store.list_accounts()
    assert len(rows) == 1
    assert rows[0]["email"] == "test@x.com"
    # list_accounts() returns summary columns, not token (use get_token_for_alias for token)
    assert rows[0]["is_active"] is True
    assert rows[0]["db_count"] == 0


def test_add_account_duplicate_email_raises(store):
    store.add_account(email="dupe@x.com", password="p", token="t")
    with pytest.raises(Exception):
        store.add_account(email="dupe@x.com", password="p2", token="t2")


def test_deactivate_account(store):
    store.add_account(email="a@x.com", password="p", token="t")
    store.deactivate_account("a@x.com")
    rows = store.list_accounts()
    assert rows[0]["is_active"] is False


def test_add_database(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    db_id = store.add_database(account_id=acc_id, name="mydb", alias="myalias")
    assert db_id is not None
    rows = store.list_databases()
    assert len(rows) == 1
    assert rows[0]["alias"] == "myalias"
    assert rows[0]["name"] == "mydb"
    assert rows[0]["is_default"] is False


def test_set_default_clears_previous(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    id1 = store.add_database(account_id=acc_id, name="db1", alias="a1")
    id2 = store.add_database(account_id=acc_id, name="db2", alias="a2")
    store.set_default("a1")
    store.set_default("a2")
    rows = store.list_databases()
    defaults = [r for r in rows if r["is_default"]]
    assert len(defaults) == 1
    assert defaults[0]["alias"] == "a2"


def test_get_default_db(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.set_default("a1")
    row = store.get_default_db()
    assert row is not None
    assert row["alias"] == "a1"


def test_get_default_db_none_when_empty(store):
    assert store.get_default_db() is None


def test_remove_database(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.remove_database("a1")
    assert store.list_databases() == []


def test_get_db_by_alias(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    store.add_database(account_id=acc_id, name="mydb", alias="mine")
    row = store.get_db("mine")
    assert row is not None
    assert row["name"] == "mydb"


def test_get_db_missing_returns_none(store):
    assert store.get_db("nope") is None


def test_log_query(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="t")
    db_id = store.add_database(account_id=acc_id, name="db1", alias="a1")
    store.log_query(db_id=db_id, sql="SELECT 1", rows_returned=1, duration_ms=5)
    rows = store.list_databases()
    assert rows[0]["query_count"] == 1


def test_get_token_for_alias(store):
    acc_id = store.add_account(email="a@x.com", password="p", token="mytoken")
    store.add_database(account_id=acc_id, name="db1", alias="a1")
    token = store.get_token_for_alias("a1")
    assert token == "mytoken"
