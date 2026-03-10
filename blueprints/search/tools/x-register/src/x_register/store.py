"""Account credential persistence."""

from __future__ import annotations

import json
import threading
import time
from dataclasses import asdict, dataclass
from pathlib import Path

DATA_DIR = Path.home() / "data" / "x"
ACCOUNTS_FILE = DATA_DIR / "accounts.json"

_lock = threading.Lock()


@dataclass
class Account:
    email: str
    email_password: str
    display_name: str
    username: str
    password: str
    auth_token: str
    ct0: str
    user_id: str
    tweet_id: str
    registered_at: str


def save(account: Account) -> None:
    """Append account to ~/data/x/accounts.json (thread-safe)."""
    with _lock:
        DATA_DIR.mkdir(parents=True, exist_ok=True)
        accounts: list[dict] = []
        if ACCOUNTS_FILE.exists():
            try:
                accounts = json.loads(ACCOUNTS_FILE.read_text())
            except Exception:
                accounts = []
        accounts.append(asdict(account))
        ACCOUNTS_FILE.write_text(json.dumps(accounts, indent=2))


def load_all() -> list[Account]:
    """Load all saved accounts."""
    try:
        raw = json.loads(ACCOUNTS_FILE.read_text())
        return [Account(**r) for r in raw]
    except Exception:
        return []


def make_account(
    *,
    email: str,
    email_password: str,
    display_name: str,
    username: str,
    password: str,
    auth_token: str,
    ct0: str,
    user_id: str,
    tweet_id: str,
) -> Account:
    return Account(
        email=email,
        email_password=email_password,
        display_name=display_name,
        username=username,
        password=password,
        auth_token=auth_token,
        ct0=ct0,
        user_id=user_id,
        tweet_id=tweet_id,
        registered_at=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    )
