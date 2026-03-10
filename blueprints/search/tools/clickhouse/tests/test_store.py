"""Tests for clickhouse_tool.store — uses in-memory DuckDB."""
import pytest
from clickhouse_tool.store import Store
from pathlib import Path


@pytest.fixture
def store(tmp_path):
    return Store(tmp_path / "test.duckdb")


def test_init_creates_tables(store):
    tables = store.con.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
    ).fetchall()
    names = {t[0] for t in tables}
    assert "accounts" in names
    assert "services" in names
    assert "query_log" in names


def test_add_account(store):
    store.add_account(
        email="test@example.com", password="pw",
        org_id="org1", api_key_id="kid", api_key_secret="ksec",
    )
    rows = store.list_accounts()
    assert len(rows) == 1
    assert rows[0]["email"] == "test@example.com"
    assert rows[0]["org_id"] == "org1"


def test_add_account_duplicate_raises(store):
    store.add_account(email="dup@test.com", password="pw")
    with pytest.raises(Exception):
        store.add_account(email="dup@test.com", password="pw2")


def test_deactivate_account(store):
    store.add_account(email="a@test.com", password="pw")
    store.deactivate_account("a@test.com")
    rows = store.list_accounts()
    assert rows[0]["is_active"] is False


def test_get_first_active_account(store):
    store.add_account(email="a@test.com", password="pw", api_key_id="k1", api_key_secret="s1")
    acc = store.get_first_active_account()
    assert acc["email"] == "a@test.com"
    assert acc["api_key_id"] == "k1"


def test_add_service(store):
    store.add_account(email="a@test.com", password="pw")
    acc = store.get_first_active_account()
    store.add_service(
        account_id=acc["id"], name="svc1", alias="s1",
        host="host.clickhouse.cloud", port=8443,
    )
    rows = store.list_services()
    assert len(rows) == 1
    assert rows[0]["name"] == "svc1"
    assert rows[0]["host"] == "host.clickhouse.cloud"


def test_set_default_clears_previous(store):
    store.add_account(email="a@test.com", password="pw")
    acc = store.get_first_active_account()
    store.add_service(account_id=acc["id"], name="s1", alias="a1")
    store.add_service(account_id=acc["id"], name="s2", alias="a2")
    store.set_default("a1")
    store.set_default("a2")
    svc1 = store.get_service("a1")
    svc2 = store.get_service("a2")
    assert svc1["is_default"] is False
    assert svc2["is_default"] is True


def test_get_default_service(store):
    store.add_account(email="a@test.com", password="pw")
    acc = store.get_first_active_account()
    store.add_service(account_id=acc["id"], name="s1", alias="a1", host="h1")
    store.set_default("a1")
    svc = store.get_default_service()
    assert svc["alias"] == "a1"
    assert svc["host"] == "h1"


def test_get_default_service_none_when_empty(store):
    assert store.get_default_service() is None


def test_remove_service(store):
    store.add_account(email="a@test.com", password="pw")
    acc = store.get_first_active_account()
    store.add_service(account_id=acc["id"], name="s1", alias="a1")
    store.remove_service("a1")
    assert store.get_service("a1") is None


def test_get_service_missing(store):
    assert store.get_service("nope") is None


def test_log_query(store):
    store.add_account(email="a@test.com", password="pw")
    acc = store.get_first_active_account()
    store.add_service(account_id=acc["id"], name="s1", alias="a1")
    svc = store.get_service("a1")
    store.log_query(service_id=svc["id"], sql="SELECT 1", rows_returned=1, duration_ms=50)
    count = store.con.execute("SELECT COUNT(*) FROM query_log").fetchone()[0]
    assert count == 1
