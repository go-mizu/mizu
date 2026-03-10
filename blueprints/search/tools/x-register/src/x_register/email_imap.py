"""IMAP email client for OTP polling.

Uses an existing email account (e.g. Outlook.com) with '+alias' addressing
so a single account generates unlimited unique addresses.

  base: automx001@outlook.com
  used: automx001+rand123@outlook.com → delivered to same inbox

X only sees unique addresses; IMAP reads from the shared inbox.

Supported providers:
  outlook  imap-mail.outlook.com:993
  gmail    imap.gmail.com:993  (needs App Password if 2FA enabled)
  yahoo    imap.mail.yahoo.com:993

Usage:
  client = ImapMailClient("outlook", "user@outlook.com", "password", verbose=True)
  mailbox = client.create_alias("rand123")   # → user+rand123@outlook.com
  otp = client.poll_for_otp(mailbox, timeout=120)
"""

from __future__ import annotations

import email as email_lib
import imaplib
import re
import time
from dataclasses import dataclass

OTP_RE = re.compile(r"\b(\d{6})\b")

PROVIDERS: dict[str, tuple[str, int]] = {
    "outlook": ("imap-mail.outlook.com", 993),
    "gmail": ("imap.gmail.com", 993),
    "yahoo": ("imap.mail.yahoo.com", 993),
}

POLL_INTERVAL = 5
POLL_TIMEOUT = 120


@dataclass
class ImapMailbox:
    address: str          # full alias address
    base_address: str     # base account address
    alias_tag: str        # the part after + (for filtering)


class ImapMailClient:
    """IMAP-based email client using +alias addressing."""

    def __init__(
        self,
        provider: str,
        address: str,
        password: str,
        verbose: bool = False,
    ):
        self._host, self._port = PROVIDERS[provider]
        self._address = address
        self._password = password
        self._verbose = verbose
        # Base address without +alias (for IMAP login)
        self._base = address.split("+")[0] + "@" + address.split("@")[1] if "+" in address else address

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [imap] {msg}", flush=True)

    def create_alias(self, tag: str) -> ImapMailbox:
        """Generate an alias address using +tag. Does not require server interaction."""
        local, domain = self._base.rsplit("@", 1)
        alias = f"{local}+{tag}@{domain}"
        self._log(f"alias: {alias}")
        return ImapMailbox(address=alias, base_address=self._base, alias_tag=tag)

    def poll_for_otp(self, mailbox: ImapMailbox, timeout: int = POLL_TIMEOUT) -> str:
        """Poll IMAP inbox for an X OTP sent to mailbox.address. Returns 6-digit code."""
        self._log(f"polling {mailbox.address} via {self._host}")
        deadline = time.time() + timeout
        seen_ids: set[bytes] = set()

        while time.time() < deadline:
            try:
                otp = self._check_inbox(mailbox, seen_ids)
                if otp:
                    return otp
            except Exception as e:
                self._log(f"  IMAP error: {e}")
            time.sleep(POLL_INTERVAL)

        raise RuntimeError(f"OTP not received within {timeout}s at {mailbox.address}")

    def _check_inbox(self, mailbox: ImapMailbox, seen_ids: set[bytes]) -> str | None:
        with imaplib.IMAP4_SSL(self._host, self._port) as imap:
            imap.login(self._base, self._password)
            imap.select("INBOX")

            # Search for messages from X/Twitter
            _, data = imap.search(None, '(OR FROM "twitter.com" FROM "x.com")')
            msg_ids = data[0].split()
            new_ids = [mid for mid in msg_ids if mid not in seen_ids]

            for mid in new_ids:
                seen_ids.add(mid)
                _, msg_data = imap.fetch(mid, "(RFC822)")
                if not msg_data or not msg_data[0]:
                    continue
                raw = msg_data[0][1]
                msg = email_lib.message_from_bytes(raw)

                # Filter by recipient matching alias
                to_header = msg.get("To", "") + msg.get("Delivered-To", "")
                if mailbox.alias_tag and mailbox.alias_tag not in to_header:
                    self._log(f"  skip: to={to_header[:60]!r} (not our alias)")
                    continue

                # Extract OTP from body
                body = _get_body(msg)
                self._log(f"  subject={msg.get('Subject','')!r} body_len={len(body)}")
                m = OTP_RE.search(body)
                if m:
                    code = m.group(1)
                    self._log(f"  OTP: {code}")
                    return code

        return None


def _get_body(msg) -> str:
    """Extract plain text body from email message."""
    if msg.is_multipart():
        for part in msg.walk():
            ct = part.get_content_type()
            if ct == "text/plain":
                try:
                    return part.get_payload(decode=True).decode("utf-8", errors="replace")
                except Exception:
                    pass
        # Fallback to HTML
        for part in msg.walk():
            if part.get_content_type() == "text/html":
                try:
                    return part.get_payload(decode=True).decode("utf-8", errors="replace")
                except Exception:
                    pass
    else:
        try:
            return msg.get_payload(decode=True).decode("utf-8", errors="replace")
        except Exception:
            pass
    return ""
