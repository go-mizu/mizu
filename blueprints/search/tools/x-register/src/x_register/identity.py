"""Realistic identity generation using Faker."""

from __future__ import annotations

import random
import secrets
import string
from dataclasses import dataclass

from faker import Faker

_fake = Faker()
_SPECIAL = "!@#$%^&*"


@dataclass
class Identity:
    display_name: str
    username: str
    email_local: str   # local part only; domain assigned by mail.tm
    password: str
    birth_year: int
    birth_month: int
    birth_day: int


def generate() -> Identity:
    """Return a randomly generated realistic identity."""
    display_name = _fake.name()

    # Username: internet-style, max 15 chars, alphanumeric + underscore
    raw = _fake.user_name().replace("-", "_").replace(".", "_")
    suffix = str(random.randint(10, 999))
    username = (raw + suffix)[:15]

    # Email local part: different from username for variety
    local_base = _fake.user_name().replace("-", "").replace(".", "")
    email_local = (local_base + str(random.randint(10, 99)))[:20]

    # Password: 14 chars, guaranteed upper + lower + digit + special
    pool = string.ascii_lowercase + string.ascii_uppercase + string.digits + _SPECIAL
    password = (
        secrets.choice(string.ascii_uppercase)
        + secrets.choice(string.ascii_lowercase)
        + secrets.choice(string.digits)
        + secrets.choice(_SPECIAL)
        + "".join(secrets.choice(pool) for _ in range(10))
    )
    # Shuffle to avoid predictable prefix
    chars = list(password)
    random.shuffle(chars)
    password = "".join(chars)

    return Identity(
        display_name=display_name,
        username=username,
        email_local=email_local,
        password=password,
        birth_year=random.randint(1975, 2002),
        birth_month=random.randint(1, 12),
        birth_day=random.randint(1, 28),
    )
