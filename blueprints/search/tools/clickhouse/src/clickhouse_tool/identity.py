"""Faker-based identity generation for ClickHouse Cloud signup."""
from __future__ import annotations

import random
import string
from dataclasses import dataclass

from faker import Faker

fake = Faker()


@dataclass
class Identity:
    display_name: str
    email_local: str
    password: str


def generate() -> Identity:
    first = fake.first_name().lower()
    last = fake.last_name().lower()
    num = random.randint(10, 99)
    display_name = f"{first.title()} {last.title()}"
    email_local = f"{first}{last}{num}"
    password = _strong_password(14)
    return Identity(display_name=display_name, email_local=email_local, password=password)


def _strong_password(length: int = 14) -> str:
    chars = string.ascii_letters + string.digits + "!@#$%"
    pwd = [
        random.choice(string.ascii_uppercase),
        random.choice(string.ascii_lowercase),
        random.choice(string.digits),
        random.choice("!@#$%"),
    ]
    pwd += [random.choice(chars) for _ in range(length - 4)]
    random.shuffle(pwd)
    return "".join(pwd)
