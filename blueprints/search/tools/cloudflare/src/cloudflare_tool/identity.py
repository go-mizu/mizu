"""Random identity generation for Cloudflare account registration."""
from __future__ import annotations

import secrets
import string
from dataclasses import dataclass

from faker import Faker

_fake = Faker()

_PWD_CHARS = string.ascii_letters + string.digits + "!@#$%^&*"


@dataclass
class Identity:
    display_name: str
    email_local: str   # part before @, max 20 chars
    password: str


def generate() -> Identity:
    first = _fake.first_name()
    last = _fake.last_name()
    display_name = f"{first} {last}"
    raw_local = f"{first.lower()}{last.lower()}{secrets.randbelow(9999)}"
    email_local = raw_local[:20]
    password = "".join(secrets.choice(_PWD_CHARS) for _ in range(14))
    return Identity(display_name=display_name, email_local=email_local, password=password)
