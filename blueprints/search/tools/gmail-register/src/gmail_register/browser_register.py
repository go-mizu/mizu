"""Browser-based Gmail signup using patchright + real Chrome.

Google signup flow at accounts.google.com/signup:
  1. First name + Last name
  2. Username (desired Gmail address) — may suggest alternatives
  3. Password + Confirm
  4. Phone verification (required) → SMS OTP
  5. Recovery email (skipped)
  6. Date of birth + Gender
  7. Privacy / Terms → Accept

Returns registered email address and the verified phone number.
"""

from __future__ import annotations

import os
import platform
import sys
import tempfile
import time
from dataclasses import dataclass

from .identity import Identity
from .sms import PhoneNumber, SmsError

SIGNUP_URL = "https://accounts.google.com/signup/v2/webcreateaccount?flowEntry=SignUp"

MONTHS = {
    1: "January", 2: "February", 3: "March", 4: "April",
    5: "May", 6: "June", 7: "July", 8: "August",
    9: "September", 10: "October", 11: "November", 12: "December",
}


@dataclass
class RegisteredAccount:
    email: str           # confirmed @gmail.com address
    phone: str           # phone used for verification


def _browser_args(headless: bool) -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage"]
    return args


def _maybe_reexec_xvfb(headless: bool) -> None:
    if platform.system() != "Linux" or headless or os.environ.get("DISPLAY"):
        return
    import shutil, subprocess
    xvfb = shutil.which("xvfb-run")
    if xvfb:
        sys.exit(subprocess.call([xvfb, "-a", sys.executable] + sys.argv))


def register(
    identity: Identity,
    sms_client,
    proxy_config: dict | None = None,
    headless: bool = True,
    verbose: bool = False,
    country: str = "any",
    sms_service: str = "google",
) -> RegisteredAccount:
    """Drive accounts.google.com signup and return registered email."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    user_data = tempfile.mkdtemp(prefix="gmail_reg_")
    phone_number: PhoneNumber | None = None

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel="chrome",
            headless=headless,
            args=_browser_args(headless),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
            proxy=proxy_config,
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            log(f"navigating to {SIGNUP_URL}")
            try:
                page.goto(SIGNUP_URL, timeout=30000)
            except Exception as e:
                log(f"nav warn: {e}")
            _wait(2, log)

            # ---- Step 1: Name ----
            log(f"filling name: {identity.first_name!r} {identity.last_name!r}")
            page.wait_for_selector('input[name="firstName"]', timeout=15000)
            _fill(page, 'input[name="firstName"]', identity.first_name)
            _fill(page, 'input[name="lastName"]', identity.last_name)
            _click_next(page, log)
            _wait(1.5, log)

            # ---- Step 2: Username ----
            log(f"filling username: {identity.username}")
            # Google may show "Create your own Gmail address" option
            try:
                page.wait_for_selector(
                    'input[name="Username"], input[aria-label*="username" i], '
                    'input[aria-label*="Gmail" i]',
                    timeout=10000,
                )
            except Exception:
                # May be on a different step — look for any radio to choose own address
                own_radio = page.locator('text="Create your own Gmail address"')
                if own_radio.count() > 0:
                    own_radio.first.click()
                    _wait(0.5, log)

            username_input = page.locator(
                'input[name="Username"], '
                'input[aria-label*="username" i], '
                'input[aria-label*="Gmail address" i]'
            )
            if username_input.count() > 0:
                username_input.first.clear()
                username_input.first.fill(identity.username)
                _wait(0.5, log)
            _click_next(page, log)
            _wait(2, log)
            log(f"url after username: {page.url}")

            # Handle "username taken" — append more digits
            taken = page.locator(
                'text="That username is taken", '
                'text="Username already", '
                '[aria-live="assertive"]:has-text("taken")'
            )
            if taken.count() > 0:
                log("username taken, trying with extra suffix...")
                import random, string
                suffix = "".join(random.choices(string.digits, k=4))
                new_username = (identity.username.rstrip("0123456789") + suffix)[:28]
                identity = Identity(
                    first_name=identity.first_name,
                    last_name=identity.last_name,
                    username=new_username,
                    password=identity.password,
                    birth_year=identity.birth_year,
                    birth_month=identity.birth_month,
                    birth_day=identity.birth_day,
                    gender=identity.gender,
                )
                username_input = page.locator('input[name="Username"]')
                username_input.first.clear()
                username_input.first.fill(new_username)
                _wait(0.5, log)
                _click_next(page, log)
                _wait(2, log)

            # ---- Step 3: Password ----
            log("setting password...")
            try:
                page.wait_for_selector(
                    'input[name="Passwd"], input[type="password"]',
                    timeout=10000,
                )
                pwd = page.locator('input[name="Passwd"], input[type="password"]').first
                pwd.fill(identity.password)
                _wait(0.3, log)
                confirm = page.locator('input[name="PasswdAgain"], input[aria-label*="Confirm" i]')
                if confirm.count() > 0:
                    confirm.first.fill(identity.password)
                _wait(0.3, log)
                _click_next(page, log)
                _wait(2, log)
            except Exception as e:
                log(f"password step: {e}")
            log(f"url: {page.url}")

            # ---- Step 4: Phone verification ----
            phone_page = page.locator(
                'text="Verify your phone number", '
                'text="phone number", '
                'input[name="phoneNumberId"], '
                'input[aria-label*="phone" i]'
            )
            if phone_page.count() > 0 or "phone" in page.url.lower() or "phone" in page.inner_text("body").lower():
                log("phone verification required, getting number...")
                phone_number = sms_client.get_number(country=country, service=sms_service)
                log(f"phone: +{phone_number.number}")

                phone_input = page.locator(
                    'input[name="phoneNumberId"], '
                    'input[type="tel"], '
                    'input[aria-label*="phone" i]'
                )
                phone_input.first.fill(phone_number.number)
                _wait(0.5, log)

                # Click "Get code" / "Send"
                _click_button(page, ["Get code", "Next", "Send"], log)
                _wait(2, log)
                log(f"url after phone submit: {page.url}")

                # Poll for OTP
                log("waiting for SMS OTP...")
                try:
                    otp = sms_client.wait_for_otp(phone_number.activation_id)
                    log(f"OTP: {otp}")

                    otp_input = page.locator(
                        'input[name="code"], '
                        'input[aria-label*="code" i], '
                        'input[aria-label*="verification" i]'
                    )
                    otp_input.first.fill(otp)
                    _wait(0.5, log)
                    _click_button(page, ["Verify", "Next", "Confirm"], log)
                    _wait(2, log)
                    sms_client.finish(phone_number.activation_id)
                    phone_number = PhoneNumber(
                        number=phone_number.number,
                        activation_id=phone_number.activation_id,
                        service=phone_number.service,
                    )
                except SmsError as e:
                    if phone_number:
                        try:
                            sms_client.cancel(phone_number.activation_id)
                        except Exception:
                            pass
                    raise RuntimeError(f"SMS OTP failed: {e}") from e
            log(f"url after phone: {page.url}")

            # ---- Step 5: Recovery email (skip) ----
            skip = page.locator('text="Skip", button:has-text("Skip")')
            if skip.count() > 0:
                log("skipping recovery email...")
                skip.first.click()
                _wait(1.5, log)

            # ---- Step 6: Birthday + Gender ----
            log("filling birthday...")
            try:
                page.wait_for_selector(
                    'select#month, input[name="month"], [aria-label*="Month" i]',
                    timeout=10000,
                )
                # Month select
                month_sel = page.locator('select#month, select[name="month"]')
                if month_sel.count() > 0:
                    month_sel.first.select_option(value=str(identity.birth_month))
                # Day input
                day_input = page.locator('input#day, input[name="day"]')
                if day_input.count() > 0:
                    day_input.first.fill(str(identity.birth_day))
                # Year input
                year_input = page.locator('input#year, input[name="year"]')
                if year_input.count() > 0:
                    year_input.first.fill(str(identity.birth_year))
                # Gender select
                gender_sel = page.locator('select#gender, select[name="gender"]')
                if gender_sel.count() > 0:
                    gender_map = {"Male": "1", "Female": "2", "Rather not say": "3"}
                    gender_sel.first.select_option(value=gender_map.get(identity.gender, "3"))
                _wait(0.5, log)
                _click_next(page, log)
                _wait(2, log)
            except Exception as e:
                log(f"birthday step: {e}")
            log(f"url: {page.url}")

            # ---- Step 7: Privacy / Terms ----
            for btn_text in ["I agree", "Agree", "Accept", "Confirm"]:
                btn = page.locator(f'button:has-text("{btn_text}"), [role="button"]:has-text("{btn_text}")')
                if btn.count() > 0:
                    log(f"clicking {btn_text!r}...")
                    btn.first.click()
                    _wait(1.5, log)
                    break

            # More I agree / confirm buttons
            for _ in range(3):
                agree = page.locator('button:has-text("I agree"), button:has-text("Agree"), button:has-text("Confirm")')
                if agree.count() > 0:
                    agree.first.click()
                    _wait(1.5, log)
                else:
                    break

            log(f"final url: {page.url}")
            _wait(3, log, "waiting for account creation")

            # ---- Extract confirmed email ----
            email_addr = f"{identity.username}@gmail.com"

            # Try to read actual email from page (Google might have changed username)
            try:
                body = page.inner_text("body")
                import re
                m = re.search(r"([\w.]+@gmail\.com)", body)
                if m:
                    email_addr = m.group(1)
                    log(f"confirmed email from page: {email_addr}")
            except Exception:
                pass

            log(f"registered: {email_addr}")
            return RegisteredAccount(
                email=email_addr,
                phone=phone_number.number if phone_number else "",
            )

        except Exception:
            if phone_number and phone_number.activation_id != "manual":
                try:
                    sms_client.cancel(phone_number.activation_id)
                except Exception:
                    pass
            raise
        finally:
            ctx.close()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _wait(seconds: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {seconds}s ({msg})...")
    time.sleep(seconds)


def _fill(page, selector: str, text: str, delay: int = 50) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=8000)
    el.triple_click()
    el.type(text, delay=delay)
    time.sleep(0.3)


def _click_next(page, log=None) -> None:
    for sel in [
        'button:has-text("Next")',
        '[role="button"]:has-text("Next")',
        'button[type="submit"]',
    ]:
        btn = page.locator(sel)
        if btn.count() > 0:
            if log:
                log(f"clicking Next via {sel!r}")
            btn.first.click()
            return


def _click_button(page, labels: list[str], log=None) -> None:
    for label in labels:
        btn = page.locator(f'button:has-text("{label}"), [role="button"]:has-text("{label}")')
        if btn.count() > 0:
            if log:
                log(f"clicking {label!r}")
            btn.first.click()
            return
