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
    # Guarantee CF password requirements: ≥8 chars, 1 uppercase, 1 digit, 1 special char
    required = [
        secrets.choice(string.ascii_uppercase),
        secrets.choice(string.ascii_lowercase),
        secrets.choice(string.digits),
        secrets.choice("!@#$%^&*"),
    ]
    rest = [secrets.choice(_PWD_CHARS) for _ in range(10)]
    pool = required + rest
    # Fisher-Yates shuffle via SystemRandom (secrets-backed)
    rng = secrets.SystemRandom()
    rng.shuffle(pool)
    password = "".join(pool)
    return Identity(display_name=display_name, email_local=email_local, password=password)
