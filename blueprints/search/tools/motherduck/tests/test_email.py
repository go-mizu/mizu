"""Unit tests for email.py — mail.tm client."""
from __future__ import annotations

import pytest
from unittest.mock import MagicMock, patch

from motherduck.email import MailTmClient, MailTmError, Mailbox


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
            {"hydra:member": [{"domain": "mail.tm", "isActive": True}]},
        )
        domain = client._get_domain()
    assert domain == "mail.tm"


def test_get_domain_no_active_raises(client):
    with patch.object(client._client, "get") as mock_get:
        mock_get.return_value = _mock_response(
            200, {"hydra:member": [{"domain": "dead.tm", "isActive": False}]}
        )
        with pytest.raises(MailTmError, match="No active"):
            client._get_domain()


def test_create_mailbox(client):
    with (
        patch.object(client, "_get_domain", return_value="mail.tm"),
        patch.object(client._client, "post") as mock_post,
        patch.object(client, "_get_token", return_value="jwt123"),
    ):
        mock_post.return_value = _mock_response(201, {})
        mb = client.create_mailbox("testuser")
    assert mb.address == "testuser@mail.tm"
    assert mb.token == "jwt123"


def test_poll_for_magic_link_found(client):
    mb = Mailbox(address="a@mail.tm", password="p", token="jwt")
    msgs = [
        {
            "id": "msg1",
            "subject": "Sign in to MotherDuck",
            "intro": "Click here to sign in",
        }
    ]
    full_msg = {
        "text": "Click this link: https://app.motherduck.com/auth/magic?token=abc123 to sign in.",
        "html": "",
    }

    def fake_get(url, **kwargs):
        if "messages/msg1" in url:
            return _mock_response(200, full_msg)
        return _mock_response(200, {"hydra:member": msgs})

    with patch.object(client._client, "get", side_effect=fake_get):
        link = client.poll_for_magic_link(mb, timeout=10)
    assert "app.motherduck.com" in link or "motherduck" in link


def test_poll_for_magic_link_timeout(client):
    mb = Mailbox(address="a@mail.tm", password="p", token="jwt")
    # Mock sleep to avoid 3s real wait in test
    with (
        patch.object(client._client, "get") as mock_get,
        patch("motherduck.email.time.sleep"),
    ):
        mock_get.return_value = _mock_response(200, {"hydra:member": []})
        with pytest.raises(MailTmError, match="not received"):
            client.poll_for_magic_link(mb, timeout=1)
