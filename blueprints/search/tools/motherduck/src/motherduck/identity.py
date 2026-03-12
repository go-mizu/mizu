"""Realistic identity generation for MotherDuck signup."""
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
    email_local: str   # local part only; domain assigned by mail.tm
    password: str      # stored in accounts table


def generate() -> Identity:
    """Return a randomly generated realistic identity."""
    display_name = _fake.name()
    local_base = _fake.user_name().replace("-", "").replace(".", "")
    email_local = (local_base + str(random.randint(10, 99)))[:20]

    pool = string.ascii_lowercase + string.ascii_uppercase + string.digits + _SPECIAL
    password = (
        secrets.choice(string.ascii_uppercase)
        + secrets.choice(string.ascii_lowercase)
        + secrets.choice(string.digits)
        + secrets.choice(_SPECIAL)
        + "".join(secrets.choice(pool) for _ in range(10))
    )
    chars = list(password)
    random.shuffle(chars)
    password = "".join(chars)

    return Identity(
        display_name=display_name,
        email_local=email_local,
        password=password,
    )
