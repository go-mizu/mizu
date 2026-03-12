"""Unit tests for email.py — mocked httpx."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from cloudflare_tool.email import MailTmClient, Mailbox, MailTmError


@pytest.fixture
def client():
    return MailTmClient(verbose=False)


def _mock_response(status_code: int, json_data: dict):
    m = MagicMock()
    m.status_code = status_code
    m.json.return_value = json_data
    m.raise_for_status = MagicMock()
    return m


def test_get_domain(client):
    with patch.object(client._client, "get") as mock_get:
        mock_get.return_value = _mock_response(
            200,
            {"hydra:member": [{"domain": "example.tm", "isActive": True}]},
        )
        domain = client._get_domain()
    assert domain == "example.tm"


def test_get_domain_no_active_raises(client):
    with patch.object(client._client, "get") as mock_get:
        mock_get.return_value = _mock_response(
            200, {"hydra:member": [{"domain": "dead.tm", "isActive": False}]}
        )
        with pytest.raises(MailTmError, match="No active"):
            client._get_domain()


def test_create_mailbox(client):
    with (
        patch.object(client, "_get_domain", return_value="example.tm"),
        patch.object(client._client, "post") as mock_post,
        patch.object(client, "_get_token", return_value="jwt123"),
    ):
        mock_post.return_value = _mock_response(201, {"id": "mb1"})
        mb = client.create_mailbox("alice")
    assert mb.address == "alice@example.tm"
    assert mb.id == "mb1"
    assert client._token == "jwt123"


def test_poll_for_magic_link_found(client):
    mb = Mailbox(address="a@example.tm", password="p", id="mb1")
    client._token = "jwt123"
    msgs = [
        {
            "id": "msg1",
            "subject": "Verify your Cloudflare account",
            "intro": "Click here to verify",
        }
    ]
    full_msg = {
        "text": "Click this link: https://dash.cloudflare.com/verify?token=abc123 to verify.",
        "html": "",
    }

    def fake_get(url, **kwargs):
        if "messages/msg1" in url:
            return _mock_response(200, full_msg)
        return _mock_response(200, {"hydra:member": msgs})

    with patch.object(client._client, "get", side_effect=fake_get):
        link = client.poll_for_magic_link(mb, timeout=10)
    assert "dash.cloudflare.com" in link or "cloudflare" in link


def test_poll_for_magic_link_timeout(client):
    mb = Mailbox(address="a@example.tm", password="p", id="mb1")
    client._token = "jwt123"
    # Mock sleep to avoid 3s real wait in test
    with (
        patch.object(client._client, "get") as mock_get,
        patch("cloudflare_tool.email.time.sleep"),
    ):
        mock_get.return_value = _mock_response(200, {"hydra:member": []})
        with pytest.raises(MailTmError, match="not received"):
            client.poll_for_magic_link(mb, timeout=1)
