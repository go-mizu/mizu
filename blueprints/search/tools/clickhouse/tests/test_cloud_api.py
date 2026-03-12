"""Tests for clickhouse_tool.cloud_api with mocked httpx."""
from unittest.mock import MagicMock, patch
import pytest
from clickhouse_tool.cloud_api import ClickHouseCloudAPI, CloudAPIError


@pytest.fixture
def mock_client():
    with patch("clickhouse_tool.cloud_api.httpx.Client") as MockClient:
        client_instance = MagicMock()
        MockClient.return_value = client_instance
        api = ClickHouseCloudAPI(key_id="kid", key_secret="ksec")
        yield api, client_instance


def test_get_organizations(mock_client):
    api, client = mock_client
    resp = MagicMock()
    resp.json.return_value = {"result": [{"id": "org1", "name": "My Org"}]}
    resp.raise_for_status = MagicMock()
    client.get.return_value = resp

    orgs = api.get_organizations()
    assert len(orgs) == 1
    assert orgs[0]["id"] == "org1"
    client.get.assert_called_with("/organizations")


def test_create_service(mock_client):
    api, client = mock_client
    resp = MagicMock()
    resp.status_code = 200
    resp.json.return_value = {
        "result": {
            "service": {
                "id": "svc1",
                "endpoints": [{"host": "h.clickhouse.cloud", "port": 8443}],
            },
            "password": "gen-pwd",
        }
    }
    client.post.return_value = resp

    result = api.create_service("org1", "test-svc")
    assert result["service"]["id"] == "svc1"
    assert result["password"] == "gen-pwd"


def test_create_service_failure(mock_client):
    api, client = mock_client
    resp = MagicMock()
    resp.status_code = 400
    resp.text = "bad request"
    client.post.return_value = resp

    with pytest.raises(CloudAPIError):
        api.create_service("org1", "fail-svc")


def test_list_services(mock_client):
    api, client = mock_client
    resp = MagicMock()
    resp.json.return_value = {"result": [{"id": "s1"}, {"id": "s2"}]}
    resp.raise_for_status = MagicMock()
    client.get.return_value = resp

    svcs = api.list_services("org1")
    assert len(svcs) == 2


def test_delete_service(mock_client):
    api, client = mock_client
    resp = MagicMock()
    resp.raise_for_status = MagicMock()
    client.delete.return_value = resp

    api.delete_service("org1", "svc1")
    client.delete.assert_called_with("/organizations/org1/services/svc1")
