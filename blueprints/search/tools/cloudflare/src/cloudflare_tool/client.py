"""Cloudflare REST API client for account info, workers management."""
from __future__ import annotations

import httpx

_BASE = "https://api.cloudflare.com/client/v4"


class CloudflareClient:
    def __init__(
        self,
        account_id: str,
        api_token: str,
        auth_email: str = "",
        auth_type: str = "bearer",
    ) -> None:
        self.account_id = account_id
        if auth_type == "global-api-key" and auth_email:
            headers = {
                "X-Auth-Email": auth_email,
                "X-Auth-Key": api_token,
                "Content-Type": "application/json",
            }
        else:
            headers = {
                "Authorization": f"Bearer {api_token}",
                "Content-Type": "application/json",
            }
        self._http = httpx.Client(headers=headers, timeout=30.0)

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

    # ------------------------------------------------------------------
    # Token management (requires Global API Key or token-creation token)
    # ------------------------------------------------------------------

    def verify_token(self) -> dict:
        """Verify the current token/key works. Returns result dict."""
        r = self._http.get(f"{_BASE}/user/tokens/verify")
        r.raise_for_status()
        return r.json().get("result", {})

    def list_permission_groups(self) -> list[dict]:
        """List available permission groups for token creation."""
        r = self._http.get(f"{_BASE}/user/tokens/permission_groups")
        r.raise_for_status()
        return r.json().get("result", [])

    def create_api_token(self, name: str, policies: list[dict]) -> dict:
        """Create a user API token. Returns result dict with 'value' key."""
        r = self._http.post(
            f"{_BASE}/user/tokens",
            json={"name": name, "policies": policies},
        )
        r.raise_for_status()
        return r.json().get("result", {})

    # ------------------------------------------------------------------
    # D1 Database
    # ------------------------------------------------------------------

    def create_d1(self, name: str) -> dict:
        """Create a D1 database. Returns the result dict with uuid, name, etc."""
        r = self._http.post(
            f"{_BASE}/accounts/{self.account_id}/d1/database",
            json={"name": name},
        )
        r.raise_for_status()
        return r.json().get("result", {})

    def list_d1(self) -> list[dict]:
        """List all D1 databases in this account."""
        r = self._http.get(
            f"{_BASE}/accounts/{self.account_id}/d1/database",
        )
        r.raise_for_status()
        return r.json().get("result", [])

    def query_d1(self, database_id: str, sql: str, params: list | None = None) -> dict:
        """Execute a SQL query on a D1 database. Returns the result dict."""
        body: dict = {"sql": sql}
        if params:
            body["params"] = params
        r = self._http.post(
            f"{_BASE}/accounts/{self.account_id}/d1/database/{database_id}/query",
            json=body,
        )
        r.raise_for_status()
        return r.json().get("result", [])

    def delete_d1(self, database_id: str) -> None:
        """Delete a D1 database."""
        r = self._http.delete(
            f"{_BASE}/accounts/{self.account_id}/d1/database/{database_id}",
        )
        r.raise_for_status()

    def close(self) -> None:
        self._http.close()
