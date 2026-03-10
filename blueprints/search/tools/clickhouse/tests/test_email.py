"""Tests for clickhouse_tool.email with mocked httpx."""
from unittest.mock import MagicMock, patch
import pytest
from clickhouse_tool.email import MailTmClient, MailTmError, _pick_verification_link


def test_pick_verification_link_verify_keyword():
    urls = ["https://clickhouse.com/", "https://auth.example.com/verify?token=abc"]
    assert _pick_verification_link(urls) == "https://auth.example.com/verify?token=abc"


def test_pick_verification_link_clickhouse_with_params():
    urls = ["https://clickhouse.com/", "https://clickhouse.cloud/confirm?code=xyz" + "a" * 50]
    result = _pick_verification_link(urls)
    assert "confirm" in result


def test_pick_verification_link_fallback():
    urls = ["https://clickhouse.com/"]
    assert _pick_verification_link(urls) == "https://clickhouse.com/"


def test_pick_verification_link_empty():
    assert _pick_verification_link([]) == ""


@patch("clickhouse_tool.email.httpx.Client")
def test_get_domain(MockClient):
    client_instance = MagicMock()
    MockClient.return_value = client_instance
    resp = MagicMock()
    resp.json.return_value = {"hydra:member": [{"domain": "test.tm", "isActive": True}]}
    resp.raise_for_status = MagicMock()
    client_instance.get.return_value = resp

    mc = MailTmClient()
    domain = mc._get_domain()
    assert domain == "test.tm"


@patch("clickhouse_tool.email.httpx.Client")
def test_get_domain_no_active_raises(MockClient):
    client_instance = MagicMock()
    MockClient.return_value = client_instance
    resp = MagicMock()
    resp.json.return_value = {"hydra:member": [{"domain": "x.tm", "isActive": False}]}
    resp.raise_for_status = MagicMock()
    client_instance.get.return_value = resp

    mc = MailTmClient()
    with pytest.raises(MailTmError):
        mc._get_domain()
