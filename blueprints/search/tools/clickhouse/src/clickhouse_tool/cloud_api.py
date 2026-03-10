"""ClickHouse Cloud REST API wrapper."""
from __future__ import annotations

import httpx


class CloudAPIError(Exception):
    pass


class ClickHouseCloudAPI:
    BASE = "https://api.clickhouse.cloud/v1"

    def __init__(self, key_id: str, key_secret: str):
        self._client = httpx.Client(
            base_url=self.BASE,
            auth=(key_id, key_secret),
            timeout=60,
        )

    def get_organizations(self) -> list[dict]:
        resp = self._client.get("/organizations")
        resp.raise_for_status()
        return resp.json().get("result", [])

    def create_service(
        self, org_id: str, name: str,
        provider: str = "aws", region: str = "us-east-1",
        tier: str = "development",
    ) -> dict:
        body = {
            "name": name,
            "provider": provider,
            "region": region,
            "tier": tier,
        }
        resp = self._client.post(
            f"/organizations/{org_id}/services", json=body
        )
        if resp.status_code not in (200, 201):
            raise CloudAPIError(
                f"create service failed: {resp.status_code} {resp.text[:300]}"
            )
        return resp.json().get("result", {})

    def list_services(self, org_id: str) -> list[dict]:
        resp = self._client.get(f"/organizations/{org_id}/services")
        resp.raise_for_status()
        return resp.json().get("result", [])

    def get_service(self, org_id: str, service_id: str) -> dict:
        resp = self._client.get(
            f"/organizations/{org_id}/services/{service_id}"
        )
        resp.raise_for_status()
        return resp.json().get("result", {})

    def delete_service(self, org_id: str, service_id: str) -> None:
        resp = self._client.delete(
            f"/organizations/{org_id}/services/{service_id}"
        )
        resp.raise_for_status()

    def close(self) -> None:
        self._client.close()
