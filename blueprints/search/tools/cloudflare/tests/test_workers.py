"""Unit tests for workers.py — mocked subprocess + httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch, call
import subprocess

from cloudflare_tool.workers import deploy, invoke, _wrangler_env


def test_wrangler_env():
    env = _wrangler_env(account_id="acc123", token="tok_abc")
    assert env["CLOUDFLARE_ACCOUNT_ID"] == "acc123"
    assert env["CLOUDFLARE_API_TOKEN"] == "tok_abc"


def test_deploy_returns_url(monkeypatch):
    mock_run = MagicMock()
    mock_run.return_value = MagicMock(
        returncode=0,
        stdout="Deployed my-worker (https://my-worker.example.workers.dev)\n",
        stderr="",
    )
    monkeypatch.setattr("cloudflare_tool.workers.subprocess.run", mock_run)

    url = deploy(
        account_id="acc123", token="tok_abc",
        name="my-worker", path="app/worker",
        subdomain="example",
    )
    assert url == "https://my-worker.example.workers.dev"
    mock_run.assert_called_once()
    cmd = mock_run.call_args[0][0]
    assert "wrangler" in " ".join(cmd)
    assert "deploy" in cmd


def test_deploy_raises_on_nonzero(monkeypatch):
    mock_run = MagicMock()
    mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="Error: bad config")
    monkeypatch.setattr("cloudflare_tool.workers.subprocess.run", mock_run)

    with pytest.raises(RuntimeError, match="wrangler deploy failed"):
        deploy(account_id="acc123", token="tok_abc", name="w", path=".", subdomain="s")


def test_invoke_get(monkeypatch):
    mock_client = MagicMock()
    resp = MagicMock()
    resp.status_code = 200
    resp.text = '{"ok": true}'
    mock_client.__enter__ = MagicMock(return_value=mock_client)
    mock_client.__exit__ = MagicMock(return_value=False)
    mock_client.request.return_value = resp
    monkeypatch.setattr("cloudflare_tool.workers.httpx.Client", lambda **kw: mock_client)

    status, body = invoke(
        url="https://my-worker.example.workers.dev",
        method="GET", path="/test",
    )
    assert status == 200
    assert body == '{"ok": true}'


def test_invoke_post_with_body(monkeypatch):
    mock_client = MagicMock()
    resp = MagicMock()
    resp.status_code = 201
    resp.text = "created"
    mock_client.__enter__ = MagicMock(return_value=mock_client)
    mock_client.__exit__ = MagicMock(return_value=False)
    mock_client.request.return_value = resp
    monkeypatch.setattr("cloudflare_tool.workers.httpx.Client", lambda **kw: mock_client)

    status, body = invoke(
        url="https://w.example.workers.dev",
        method="POST", path="/data",
        body='{"key": "value"}',
    )
    assert status == 201
    assert body == "created"
    # Verify body was passed
    kwargs = mock_client.request.call_args[1]
    assert kwargs.get("content") == '{"key": "value"}'
