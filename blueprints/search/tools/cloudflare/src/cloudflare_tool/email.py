"""mail.tm API client: create mailbox and poll for magic link URL."""
from __future__ import annotations

import re
import time
from dataclasses import dataclass

import httpx

BASE = "https://api.mail.tm"
# Cloudflare sends a URL containing their domain
MAGIC_LINK_RE = re.compile(r"https://[^\s\"'<>]*cloudflare\.com[^\s\"'<>]*")
ALL_URLS_RE = re.compile(r"https?://[^\s\"'<>]+")
POLL_INTERVAL = 3
POLL_TIMEOUT = 120


def _pick_verification_link(urls: list[str]) -> str:
    """Pick the actual verification link from all URLs in the email.

    Cloudflare verification links typically contain parameters like token=, verify=,
    or are on dash.cloudflare.com or accounts.cloudflare.com domains.
    """
    cleaned = [u.rstrip(".") for u in urls]

    # Priority 1: Cloudflare domain links (dash.cloudflare.com or accounts.cloudflare.com)
    for u in cleaned:
        if "dash.cloudflare.com" in u or "accounts.cloudflare.com" in u:
            return u

    # Priority 2: Links with verification-related query params
    for u in cleaned:
        lower = u.lower()
        if any(kw in lower for kw in ["token=", "verify", "confirm", "email-verification"]):
            return u

    # Priority 3: Long links with query parameters (likely not just branding)
    for u in cleaned:
        if "?" in u and len(u) > 60 and "cloudflare" in u:
            return u

    # Priority 4: Any long link with query params
    for u in cleaned:
        if "?" in u and len(u) > 60:
            return u

    # Fallback: first cloudflare link that isn't just the homepage
    for u in cleaned:
        if "cloudflare" in u and u.rstrip("/") != "https://cloudflare.com":
            return u

    # Last resort: first link
    return cleaned[0] if cleaned else ""


@dataclass
class Mailbox:
    address: str
    password: str
    id: str


class MailTmError(Exception):
    pass


class MailTmClient:
    def __init__(self, verbose: bool = False) -> None:
        self._verbose = verbose
        self._client = httpx.Client(timeout=15)
        self._token = None

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
        password = f"Cf{local[:6]}!9xQ"
        self._log(f"creating mailbox {address}")
        resp = self._client.post(
            f"{BASE}/accounts", json={"address": address, "password": password}
        )
        if resp.status_code not in (200, 201):
            raise MailTmError(
                f"create account failed: {resp.status_code} {resp.text[:200]}"
            )
        data = resp.json()
        mailbox_id = data.get("id")
        self._token = self._get_token(address, password)
        return Mailbox(address=address, password=password, id=mailbox_id)

    def _get_token(self, address: str, password: str) -> str:
        resp = self._client.post(
            f"{BASE}/token", json={"address": address, "password": password}
        )
        resp.raise_for_status()
        return resp.json()["token"]

    def poll_for_magic_link(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox until a Cloudflare magic link arrives. Returns the URL."""
        headers = {"Authorization": f"Bearer {self._token}"}
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
                        text_part = body.get("text", "")
                        html_parts = body.get("html", [])
                        # html can be a list of strings in mail.tm API
                        if isinstance(html_parts, list):
                            html_str = " ".join(html_parts)
                        else:
                            html_str = str(html_parts)
                        text = text_part + " " + html_str + " " + intro
                    except Exception as e:
                        self._log(f"  body fetch error: {e}")

                    # Find ALL URLs in email body and pick the verification link
                    all_urls = ALL_URLS_RE.findall(text)
                    link = _pick_verification_link(all_urls)
                    if link:
                        self._log(f"  magic link found: {link[:200]}")
                        return link
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"Magic link not received within {timeout}s at {mailbox.address}"
        )

    def reconnect(self, mailbox: Mailbox) -> None:
        """Reconnect to an existing mailbox (re-authenticate)."""
        self._token = self._get_token(mailbox.address, mailbox.password)

    def poll_for_verification_code(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox for a CF verification code (numeric). Returns the code string."""
        headers = {"Authorization": f"Bearer {self._token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()
        code_re = re.compile(r"\b(\d{6,8})\b")

        self._log(f"polling {mailbox.address} for verification code (timeout={timeout}s)")
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
                    self._log(f"  msg: subject={subject!r}")

                    # Check subject for code
                    m = code_re.search(subject)
                    if m:
                        self._log(f"  code from subject: {m.group(1)}")
                        return m.group(1)

                    # Fetch full message
                    try:
                        full = self._client.get(
                            f"{BASE}/messages/{msg_id}", headers=headers
                        )
                        body = full.json()
                        text = body.get("text", "") + " " + body.get("intro", "")
                        m = code_re.search(text)
                        if m:
                            self._log(f"  code from body: {m.group(1)}")
                            return m.group(1)
                    except Exception as e:
                        self._log(f"  body fetch error: {e}")
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"Verification code not received within {timeout}s at {mailbox.address}"
        )

    def close(self) -> None:
        self._client.close()
