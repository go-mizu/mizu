"""Unit tests for store.py — in-memory DuckDB."""
from __future__ import annotations

import pytest
from cloudflare_tool.store import Store


@pytest.fixture
def store():
    return Store(":memory:")


def test_init_creates_tables(store):
    tables = store.con.execute(
        "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
    ).fetchall()
    names = {r[0] for r in tables}
    assert "accounts" in names
    assert "tokens" in names
    assert "workers" in names
    assert "op_log" in names


def test_add_account(store):
    acc_id = store.add_account(
        email="test@x.com", password="pass", account_id="acc123"
    )
    assert acc_id is not None
    rows = store.list_accounts()
    assert len(rows) == 1
    assert rows[0]["email"] == "test@x.com"
    assert rows[0]["account_id"] == "acc123"
    assert rows[0]["is_active"] is True


def test_add_account_duplicate_raises(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    with pytest.raises(Exception):
        store.add_account(email="a@x.com", password="p2", account_id="a2")


def test_deactivate_account(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    store.deactivate_account("a@x.com")
    rows = store.list_accounts()
    assert rows[0]["is_active"] is False


def test_get_first_active_account(store):
    store.add_account(email="a@x.com", password="p", account_id="a1")
    acc = store.get_first_active_account()
    assert acc is not None
    assert acc["email"] == "a@x.com"
    assert acc["account_id"] == "a1"


def test_add_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(
        account_id=acc_id, name="my-token",
        token_value="tok_abc123", preset="browser-rendering"
    )
    assert tok_id is not None
    rows = store.list_tokens()
    assert len(rows) == 1
    assert rows[0]["name"] == "my-token"
    assert rows[0]["preset"] == "browser-rendering"


def test_set_default_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_token(account_id=acc_id, name="t2", token_value="v2", preset="workers")
    store.set_default_token("t1")
    store.set_default_token("t2")
    default = store.get_default_token()
    assert default["name"] == "t2"
    # Only one default
    rows = store.list_tokens()
    defaults = [r for r in rows if r["is_default"]]
    assert len(defaults) == 1


def test_remove_token(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.remove_token("t1")
    assert store.list_tokens() == []


def test_add_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="workers")
    w_id = store.add_worker(
        account_id=acc_id, token_id=tok_id,
        name="my-worker", alias="mw",
        url="https://my-worker.example.workers.dev"
    )
    assert w_id is not None
    rows = store.list_workers()
    assert len(rows) == 1
    assert rows[0]["alias"] == "mw"
    assert rows[0]["name"] == "my-worker"


def test_set_default_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w1", alias="w1", url="u1")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w2", alias="w2", url="u2")
    store.set_default_worker("w2")
    default = store.get_default_worker()
    assert default["alias"] == "w2"


def test_remove_worker(store):
    acc_id = store.add_account(email="a@x.com", password="p", account_id="a1")
    tok_id = store.add_token(account_id=acc_id, name="t1", token_value="v1", preset="all")
    store.add_worker(account_id=acc_id, token_id=tok_id, name="w1", alias="w1", url="u")
    store.remove_worker("w1")
    assert store.list_workers() == []


def test_log_operation(store):
    store.log_op(worker_id=None, operation="deploy", detail="my-worker", duration_ms=500)
    rows = store.con.execute("SELECT * FROM op_log").fetchall()
    assert len(rows) == 1
    assert rows[0][2] == "deploy"
