"""mail.tm API client: create mailbox and poll for magic link URL."""
from __future__ import annotations

import re
import time
from dataclasses import dataclass

import httpx

BASE = "https://api.mail.tm"
# MotherDuck sends a URL containing their domain
MAGIC_LINK_RE = re.compile(r"https://[^\s\"'<>]*motherduck\.com[^\s\"'<>]*")
POLL_INTERVAL = 3
POLL_TIMEOUT = 120


@dataclass
class Mailbox:
    address: str
    password: str
    token: str


class MailTmError(Exception):
    pass


class MailTmClient:
    def __init__(self, verbose: bool = False) -> None:
        self._verbose = verbose
        self._client = httpx.Client(timeout=15)

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [mail.tm] {msg}", flush=True)

    def _get_domain(self) -> str:
        resp = self._client.get(f"{BASE}/domains")
        resp.raise_for_status()
        domains = resp.json().get("hydra:member", [])
        active = [d["domain"] for d in domains if d.get("isActive")]
        if not active:
            raise MailTmError("No active mail.tm domains available")
        return active[0]

    def create_mailbox(self, local: str) -> Mailbox:
        domain = self._get_domain()
        address = f"{local}@{domain}"
        password = f"Mz{local[:6]}!9xQ"
        self._log(f"creating mailbox {address}")
        resp = self._client.post(
            f"{BASE}/accounts", json={"address": address, "password": password}
        )
        if resp.status_code not in (200, 201):
            raise MailTmError(
                f"create account failed: {resp.status_code} {resp.text[:200]}"
            )
        token = self._get_token(address, password)
        return Mailbox(address=address, password=password, token=token)

    def _get_token(self, address: str, password: str) -> str:
        resp = self._client.post(
            f"{BASE}/token", json={"address": address, "password": password}
        )
        resp.raise_for_status()
        return resp.json()["token"]

    def poll_for_magic_link(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox until a MotherDuck magic link arrives. Returns the URL."""
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()

        self._log(f"polling {mailbox.address} for magic link (timeout={timeout}s)")
        while time.time() < deadline:
            try:
                resp = self._client.get(f"{BASE}/messages", headers=headers)
                resp.raise_for_status()
                messages = resp.json().get("hydra:member", [])
                for msg in messages:
                    msg_id = msg.get("id", "")
                    if msg_id in seen:
                        continue
                    seen.add(msg_id)
                    subject = msg.get("subject", "")
                    intro = msg.get("intro", "")
                    self._log(f"  msg: subject={subject!r}")

                    # Fetch full message for link extraction
                    text = intro
                    try:
                        full = self._client.get(
                            f"{BASE}/messages/{msg_id}", headers=headers
                        )
                        body = full.json()
                        text = body.get("text", "") + " " + body.get("html", "") + " " + intro
                    except Exception:
                        pass

                    m = MAGIC_LINK_RE.search(text)
                    if m:
                        link = m.group(0).rstrip(".")
                        self._log(f"  magic link found: {link[:60]}...")
                        return link
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"Magic link not received within {timeout}s at {mailbox.address}"
        )

    def close(self) -> None:
        self._client.close()
