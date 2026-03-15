"""mail.tm API client: create mailbox and poll for Goodreads confirmation link."""
from __future__ import annotations

import re
import time
from dataclasses import dataclass

import httpx

BASE = "https://api.mail.tm"
# Goodreads sends confirmation emails with a link containing /user/confirmation
VERIFY_LINK_RE = re.compile(r"https://[^\s\"'<>]*goodreads\.com[^\s\"'<>]*confirmation[^\s\"'<>]*")
ALL_URLS_RE = re.compile(r"https?://[^\s\"'<>]+")
POLL_INTERVAL = 3
POLL_TIMEOUT = 120


def _pick_verification_link(urls: list[str]) -> str:
    """Pick the Goodreads confirmation link from all URLs in the email."""
    cleaned = [u.rstrip(".,;)") for u in urls]

    # Priority 1: Goodreads confirmation links
    for u in cleaned:
        if "goodreads.com" in u and ("confirmation" in u or "confirm" in u):
            return u

    # Priority 2: Any goodreads.com link with a token param
    for u in cleaned:
        if "goodreads.com" in u and "?" in u and len(u) > 60:
            return u

    # Priority 3: Any goodreads.com link that's not the homepage
    for u in cleaned:
        if "goodreads.com" in u and u.rstrip("/") not in (
            "https://www.goodreads.com", "http://www.goodreads.com"
        ):
            return u

    return cleaned[0] if cleaned else ""


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

    def poll_for_verification_link(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox until a Goodreads confirmation link arrives. Returns the URL."""
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()

        self._log(f"polling {mailbox.address} for verification link (timeout={timeout}s)")
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
                        if isinstance(html_parts, list):
                            html_str = " ".join(html_parts)
                        else:
                            html_str = str(html_parts)
                        text = text_part + " " + html_str + " " + intro
                    except Exception as e:
                        self._log(f"  body fetch error: {e}")

                    all_urls = ALL_URLS_RE.findall(text)
                    link = _pick_verification_link(all_urls)
                    if link:
                        self._log(f"  verification link found: {link[:200]}")
                        return link
            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"Goodreads verification link not received within {timeout}s at {mailbox.address}"
        )

    def poll_for_otp(self, mailbox: Mailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll inbox for an OTP/verification code from Amazon/Goodreads.

        Amazon sends emails with 6-digit OTP codes for account verification.
        Returns the code string (e.g. "123456").
        """
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        seen: set[str] = set()

        self._log(f"polling {mailbox.address} for OTP (timeout={timeout}s)")
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

                    # Fetch full message
                    text = intro
                    try:
                        full = self._client.get(f"{BASE}/messages/{msg_id}", headers=headers)
                        body = full.json()
                        text_part = body.get("text", "")
                        html_parts = body.get("html", [])
                        if isinstance(html_parts, list):
                            html_str = " ".join(html_parts)
                        else:
                            html_str = str(html_parts)
                        text = text_part + " " + html_str + " " + intro
                    except Exception as e:
                        self._log(f"  body fetch error: {e}")

                    # Look for a 6-digit OTP code (common Amazon pattern)
                    otp_match = re.search(r'\b(\d{6})\b', text)
                    if otp_match:
                        code = otp_match.group(1)
                        self._log(f"  OTP found: {code}")
                        return code

                    # Also check for 4-digit or 8-digit codes
                    for pattern in [r'\b(\d{8})\b', r'\b(\d{4})\b']:
                        m = re.search(pattern, text)
                        if m:
                            code = m.group(1)
                            self._log(f"  code found ({len(code)}-digit): {code}")
                            return code

            except Exception as e:
                self._log(f"  poll error: {e}")
            time.sleep(POLL_INTERVAL)

        raise MailTmError(
            f"OTP not received within {timeout}s at {mailbox.address}"
        )

    def close(self) -> None:
        self._client.close()
