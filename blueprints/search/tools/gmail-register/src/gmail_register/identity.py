"""Realistic identity generation for Gmail signup."""

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
    first_name: str
    last_name: str
    username: str       # desired Gmail username (without @gmail.com)
    password: str
    birth_year: int
    birth_month: int
    birth_day: int
    gender: str         # "Male" | "Female" | "Rather not say"


def generate() -> Identity:
    """Generate a realistic Gmail identity."""
    first = _fake.first_name()
    last = _fake.last_name()

    # Gmail username: firstname.lastname + digits, max 30 chars
    base = f"{first.lower()}.{last.lower()}".replace(" ", "").replace("'", "")
    suffix = str(random.randint(100, 9999))
    username = (base + suffix)[:28]
    # Ensure only allowed chars (letters, digits, dots)
    username = "".join(c for c in username if c.isalnum() or c == ".")

    # Password
    pool = string.ascii_lowercase + string.ascii_uppercase + string.digits + _SPECIAL
    pwd_chars = (
        secrets.choice(string.ascii_uppercase)
        + secrets.choice(string.ascii_lowercase)
        + secrets.choice(string.digits)
        + secrets.choice(_SPECIAL)
        + "".join(secrets.choice(pool) for _ in range(10))
    )
    chars = list(pwd_chars)
    random.shuffle(chars)
    password = "".join(chars)

    gender = random.choice(["Male", "Female", "Rather not say"])

    return Identity(
        first_name=first,
        last_name=last,
        username=username,
        password=password,
        birth_year=random.randint(1975, 2002),
        birth_month=random.randint(1, 12),
        birth_day=random.randint(1, 28),
        gender=gender,
    )
