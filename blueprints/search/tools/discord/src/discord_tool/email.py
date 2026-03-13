"""Temporary email clients: mail.tm and 1secmail (fallback)."""
from __future__ import annotations

import re
import time
from dataclasses import dataclass
from typing import Protocol

import httpx

_BASE_MAILTM = "https://api.mail.tm"
_BASE_1SEC = "https://www.1secmail.com/api/v1/"

# 1secmail domains — most are lesser-known and less likely to be blocklisted
_ONESEC_DOMAINS = [
    "1secmail.com", "1secmail.net", "1secmail.org",
    "kzccv.com", "qiott.com", "wuuvo.com",
    "icznn.com", "ezztt.com",
]


@dataclass
class Mailbox:
    address: str
    password: str
    token: str      # JWT for mail.tm; empty string for 1secmail
    provider: str   # "mailtm" or "1secmail"


class MailTmClient:
    def __init__(self) -> None:
        self._client = httpx.Client(timeout=30)

    def _domains(self) -> list[str]:
        r = self._client.get(f"{_BASE_MAILTM}/domains")
        r.raise_for_status()
        data = r.json()
        return [d["domain"] for d in data.get("hydra:member", [])]

    def create_mailbox(self, local: str, password: str, domain: str = "") -> Mailbox:
        if not domain:
            domains = self._domains()
            if not domains:
                raise RuntimeError("mail.tm returned no domains")
            domain = domains[0]
        address = f"{local}@{domain}"
        r = self._client.post(f"{_BASE_MAILTM}/accounts", json={"address": address, "password": password})
        r.raise_for_status()
        r2 = self._client.post(f"{_BASE_MAILTM}/token", json={"address": address, "password": password})
        r2.raise_for_status()
        token = r2.json()["token"]
        return Mailbox(address=address, password=password, token=token, provider="mailtm")

    def poll_for_link(self, mailbox: Mailbox, timeout: int = 120, keyword: str = "") -> str:
        if mailbox.provider == "1secmail":
            return _poll_1sec(mailbox, timeout=timeout, keyword=keyword)
        if mailbox.provider == "maildrop":
            return _poll_maildrop(mailbox, timeout=timeout, keyword=keyword)
        headers = {"Authorization": f"Bearer {mailbox.token}"}
        deadline = time.time() + timeout
        while time.time() < deadline:
            r = self._client.get(f"{_BASE_MAILTM}/messages", headers=headers)
            if r.status_code == 200:
                for msg in r.json().get("hydra:member", []):
                    r2 = self._client.get(f"{_BASE_MAILTM}/messages/{msg['id']}", headers=headers)
                    if r2.status_code == 200:
                        body = r2.json().get("text", "") + r2.json().get("html", "")
                        for url in re.findall(r"https?://\S+", body):
                            url = url.rstrip(".,;)")
                            if not keyword or keyword in url:
                                return url
            time.sleep(5)
        raise TimeoutError(f"No email with link received within {timeout}s")

    def poll_for_magic_link(self, mailbox: Mailbox, timeout: int = 120) -> str:
        return self.poll_for_link(mailbox, timeout=timeout)


def create_1sec_mailbox(local: str, password: str = "", domain: str = "") -> Mailbox:
    """Create a 1secmail inbox (no registration needed — just pick an address)."""
    if not domain:
        domain = _ONESEC_DOMAINS[0]
    address = f"{local}@{domain}"
    return Mailbox(address=address, password=password, token="", provider="1secmail")


def create_maildrop_mailbox(local: str, password: str = "") -> Mailbox:
    """Create a maildrop.cc inbox (no registration — any username works)."""
    address = f"{local}@maildrop.cc"
    return Mailbox(address=address, password=password, token="", provider="maildrop")


def _poll_1sec(mailbox: Mailbox, timeout: int = 120, keyword: str = "") -> str:
    client = httpx.Client(timeout=30)
    local, domain = mailbox.address.split("@", 1)
    deadline = time.time() + timeout
    while time.time() < deadline:
        r = client.get(_BASE_1SEC, params={
            "action": "getMessages", "login": local, "domain": domain
        })
        if r.status_code == 200:
            for msg in r.json():
                r2 = client.get(_BASE_1SEC, params={
                    "action": "readMessage",
                    "login": local, "domain": domain, "id": msg["id"]
                })
                if r2.status_code == 200:
                    data = r2.json()
                    body = data.get("textBody", "") + data.get("htmlBody", "")
                    for url in re.findall(r"https?://\S+", body):
                        url = url.rstrip(".,;)")
                        if not keyword or keyword in url:
                            return url
        time.sleep(5)
    raise TimeoutError(f"No email with link received within {timeout}s")


def _poll_maildrop(mailbox: Mailbox, timeout: int = 120, keyword: str = "") -> str:
    """Poll maildrop.cc inbox via GraphQL API."""
    client = httpx.Client(timeout=30)
    local = mailbox.address.split("@")[0]
    gql_url = "https://api.maildrop.cc/graphql"
    deadline = time.time() + timeout
    seen_ids: set[str] = set()
    while time.time() < deadline:
        r = client.post(gql_url, json={
            "query": f'{{ inbox(mailbox: "{local}") {{ id subject }} }}'
        })
        if r.status_code == 200:
            msgs = (r.json().get("data") or {}).get("inbox") or []
            for msg in msgs:
                mid = msg["id"]
                if mid in seen_ids:
                    continue
                seen_ids.add(mid)
                r2 = client.post(gql_url, json={
                    "query": f'{{ message(mailbox: "{local}", id: "{mid}") {{ html text }} }}'
                })
                if r2.status_code == 200:
                    data = (r2.json().get("data") or {}).get("message") or {}
                    body = (data.get("html") or "") + (data.get("text") or "")
                    for url in re.findall(r"https?://\S+", body):
                        url = url.rstrip(".,;)")
                        if not keyword or keyword in url:
                            return url
        time.sleep(5)
    raise TimeoutError(f"No email with link received within {timeout}s")
