"""Fake identity generation for Discord account registration."""
from __future__ import annotations

import random
import string
from dataclasses import dataclass


@dataclass
class Identity:
    username: str      # Discord username (3-32 chars, no spaces)
    display_name: str  # Display name shown in UI
    email_local: str   # Local part for mail.tm address
    password: str
    birth_year: int
    birth_month: int
    birth_day: int


def generate() -> Identity:
    from faker import Faker
    fake = Faker()

    first = fake.first_name()
    last = fake.last_name()
    display_name = f"{first} {last}"

    suffix = "".join(random.choices(string.digits, k=4))
    username = f"{first.lower()}{last.lower()}{suffix}"[:32]
    email_local = f"{first.lower()}{last.lower()}{suffix}"

    # Generate password: 12+ chars, mixed case + digit + special
    password = (
        fake.password(length=14, special_chars=True, digits=True,
                      upper_case=True, lower_case=True)
    )

    # Must be 13+ years old for Discord
    birth_year = random.randint(1985, 2005)
    birth_month = random.randint(1, 12)
    birth_day = random.randint(1, 28)

    return Identity(
        username=username,
        display_name=display_name,
        email_local=email_local,
        password=password,
        birth_year=birth_year,
        birth_month=birth_month,
        birth_day=birth_day,
    )
