"""Unit tests for client.py — mocked httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from cloudflare_tool.client import CloudflareClient


@pytest.fixture
def client():
    return CloudflareClient(account_id="acc123", api_token="tok_abc")


def _mock_response(data: dict, status: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status
    resp.raise_for_status = MagicMock()
    resp.json.return_value = data
    return resp


def test_get_account_id(monkeypatch):
    """get_account_id() uses API to fetch first account."""
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": [{"id": "acc-xyz", "name": "My Account"}],
        "success": True,
    })
    monkeypatch.setattr("cloudflare_tool.client.httpx.Client", lambda **kw: mock_http)
    c = CloudflareClient(account_id="", api_token="tok_abc")
    account_id = c.get_account_id()
    assert account_id == "acc-xyz"


def test_get_subdomain(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": {"subdomain": "myaccount"},
        "success": True,
    })
    client._http = mock_http
    sub = client.get_subdomain()
    assert sub == "myaccount"


def test_list_workers(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.get.return_value = _mock_response({
        "result": [
            {"id": "w1", "script": "my-worker", "created_on": "2024-01-01"},
        ],
        "success": True,
    })
    client._http = mock_http
    workers = client.list_workers()
    assert len(workers) == 1
    assert workers[0]["script"] == "my-worker"


def test_delete_worker(client, monkeypatch):
    mock_http = MagicMock()
    mock_http.delete.return_value = _mock_response({"result": None, "success": True})
    client._http = mock_http
    # Should not raise
    client.delete_worker("my-worker")
    mock_http.delete.assert_called_once()


def test_close(client):
    """close() does not raise."""
    client.close()
