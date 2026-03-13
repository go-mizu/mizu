"""Fake identity for Proton Mail registration."""
from __future__ import annotations

import random
import string
from dataclasses import dataclass


@dataclass
class Identity:
    username: str   # Proton username (becomes username@proton.me)
    password: str
    display_name: str


def generate() -> Identity:
    from faker import Faker
    fake = Faker()

    first = fake.first_name().lower()
    last  = fake.last_name().lower()
    suffix = "".join(random.choices(string.digits, k=4))
    username = f"{first}{last}{suffix}"[:40]
    display_name = f"{first.capitalize()} {last.capitalize()}"

    password = fake.password(length=14, special_chars=True, digits=True,
                             upper_case=True, lower_case=True)

    return Identity(username=username, password=password, display_name=display_name)
