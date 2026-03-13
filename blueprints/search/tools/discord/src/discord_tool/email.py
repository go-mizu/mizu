"""mail.tm temporary email client."""
from __future__ import annotations

import time
from dataclasses import dataclass

import httpx

_BASE = "https://api.mail.tm"


@dataclass
class Mailbox:
    address: str
    password: str
    token: str


class MailTmClient:
    def __init__(self) -> None:
        self._client = httpx.Client(timeout=30)

    def _domains(self) -> list[str]:
        r = self._client.get(f"{_BASE}/domains")
        r.raise_for_status()
        data = r.json()
        return [d["domain"] for d in data.get("hydra:member", [])]

    def create_mailbox(self, local: str, password: str) -> Mailbox:
        domains = self._domains()
        if not domains:
            raise RuntimeError("mail.tm returned no domains")
        address = f"{local}@{domains[0]}"
        r = self._client.post(f"{_BASE}/accounts", json={"address": address, "password": password})
        r.raise_for_status()
        # Get JWT token
        r2 = self._client.post(f"{_BASE}/token", json={"address": address, "password": password})
        r2.raise_for_status()
        token = r2.json()["token"]
        return Mailbox(address=address, password=password, token=token)

    def poll_for_link(self, mailbox: Mailbox, timeout: int = 120, keyword: str = "") -> str:
        """Poll inbox and return the first URL containing keyword (or any URL if empty)."""
        import re
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        while time.time() < deadline:
            r = self._client.get(f"{_BASE}/messages", headers=headers)
            if r.status_code == 200:
                msgs = r.json().get("hydra:member", [])
                for msg in msgs:
                    mid = msg["id"]
                    r2 = self._client.get(f"{_BASE}/messages/{mid}", headers=headers)
                    if r2.status_code == 200:
                        body = r2.json().get("text", "") + r2.json().get("html", "")
                        urls = re.findall(r"https?://\S+", body)
                        for url in urls:
                            url = url.rstrip(".,;)")
                            if not keyword or keyword in url:
                                return url
            time.sleep(5)
        raise TimeoutError(f"No email with link received within {timeout}s")

    def poll_for_magic_link(self, mailbox: Mailbox, timeout: int = 120) -> str:
        return self.poll_for_link(mailbox, timeout=timeout)
