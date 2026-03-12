"""Gmail account credential persistence."""

from __future__ import annotations

import json
import threading
import time
from dataclasses import asdict, dataclass
from pathlib import Path

DATA_DIR = Path.home() / "data" / "gmail"
ACCOUNTS_FILE = DATA_DIR / "accounts.json"

_lock = threading.Lock()


@dataclass
class Account:
    email: str           # full Gmail address, e.g. john.doe1984@gmail.com
    first_name: str
    last_name: str
    password: str
    phone: str           # phone number used for verification
    birth_year: int
    birth_month: int
    birth_day: int
    recovery_email: str  # empty if skipped
    registered_at: str


def save(account: Account) -> None:
    """Append account to ~/data/gmail/accounts.json (thread-safe)."""
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
    try:
        raw = json.loads(ACCOUNTS_FILE.read_text())
        return [Account(**r) for r in raw]
    except Exception:
        return []


def make_account(
    *,
    email: str,
    first_name: str,
    last_name: str,
    password: str,
    phone: str,
    birth_year: int,
    birth_month: int,
    birth_day: int,
    recovery_email: str = "",
) -> Account:
    return Account(
        email=email,
        first_name=first_name,
        last_name=last_name,
        password=password,
        phone=phone,
        birth_year=birth_year,
        birth_month=birth_month,
        birth_day=birth_day,
        recovery_email=recovery_email,
        registered_at=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    )
