"""Cloudflare REST API client for account info, workers management."""
from __future__ import annotations

import httpx

_BASE = "https://api.cloudflare.com/client/v4"


class CloudflareClient:
    def __init__(self, account_id: str, api_token: str) -> None:
        self.account_id = account_id
        self._http = httpx.Client(
            headers={
                "Authorization": f"Bearer {api_token}",
                "Content-Type": "application/json",
            },
            timeout=30.0,
        )

    def get_account_id(self) -> str:
        """Fetch the first account ID from the API (use when account_id not yet known)."""
        r = self._http.get(f"{_BASE}/accounts", params={"per_page": 1})
        r.raise_for_status()
        result = r.json().get("result", [])
        if not result:
            raise RuntimeError("No Cloudflare accounts found via API")
        self.account_id = result[0]["id"]
        return self.account_id

    def get_subdomain(self) -> str:
        """Return the workers.dev subdomain for this account."""
        r = self._http.get(
            f"{_BASE}/accounts/{self.account_id}/workers/subdomain"
        )
        r.raise_for_status()
        return r.json().get("result", {}).get("subdomain", "")

    def list_workers(self) -> list[dict]:
        """List all Worker scripts in this account."""
        r = self._http.get(
            f"{_BASE}/accounts/{self.account_id}/workers/scripts"
        )
        r.raise_for_status()
        return r.json().get("result", [])

    def delete_worker(self, name: str) -> None:
        """Delete a Worker script by name."""
        r = self._http.delete(
            f"{_BASE}/accounts/{self.account_id}/workers/scripts/{name}"
        )
        r.raise_for_status()

    def close(self) -> None:
        self._http.close()
