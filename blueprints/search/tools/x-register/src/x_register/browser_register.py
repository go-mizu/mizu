"""Browser-based X signup using patchright + real Chrome.

Flow:
  1. Visit x.com (gets Cloudflare cookies)
  2. Navigate to /i/flow/signup
  3. Click "Create account"
  4. Fill name, email, birthday → click Next
  5. Skip customization → click Sign up
  6. Poll mail.tm for OTP → enter it
  7. Set password → skip remaining prompts
  8. Extract auth_token from cookies
"""

from __future__ import annotations

import os
import platform
import sys
import tempfile
import time
from dataclasses import dataclass

from .email import MailTmClient, Mailbox
from .identity import Identity

# Month name → value mapping for SELECTOR_1 (month select)
MONTHS = {
    1: "January", 2: "February", 3: "March", 4: "April",
    5: "May", 6: "June", 7: "July", 8: "August",
    9: "September", 10: "October", 11: "November", 12: "December",
}


@dataclass
class BrowserAccount:
    auth_token: str
    ct0: str
    user_id: str
    screen_name: str


def _browser_args() -> list[str]:
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


def sign_up_via_browser(
    identity: Identity,
    mailbox: Mailbox,
    mail_client: MailTmClient,
    headless: bool = True,
    verbose: bool = False,
) -> BrowserAccount:
    """Drive x.com signup form with patchright and return auth cookies."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser-reg] {msg}", flush=True)

    log(f"signing up {mailbox.address} as {identity.display_name!r}")

    user_data = tempfile.mkdtemp(prefix="x_reg_")

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel="chrome",
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Visit x.com to warm up CF cookies ----
            log("visiting x.com to warm up...")
            try:
                page.goto("https://x.com", timeout=25000)
            except Exception as e:
                log(f"x.com warn: {e}")
            _wait(8, log, "warming up")

            # ---- Step 2: Navigate to signup ----
            log("navigating to signup...")
            try:
                page.goto("https://x.com/i/flow/signup", timeout=25000)
            except Exception as e:
                log(f"signup nav warn: {e}")
            _wait(3, log)

            # ---- Step 3: Click "Create account" ----
            log("clicking 'Create account'...")
            create_btn = page.locator(
                'a[href="/i/flow/signup"] span:has-text("Create account"), '
                'span:has-text("Create account"), '
                'a:has-text("Create account")'
            )
            if create_btn.count() > 0:
                create_btn.first.click()
                _wait(2, log)
            log(f"url: {page.url}")

            # ---- Step 4: Fill name ----
            log(f"filling name: {identity.display_name!r}")
            page.wait_for_selector('input[name="name"]', timeout=15000)
            _fill(page, 'input[name="name"]', identity.display_name)

            # ---- Step 5: Switch to email if phone shown by default ----
            _wait(0.5, log)
            toggle = page.locator('span:has-text("Use email instead")')
            if toggle.count() > 0:
                log("switching to email input")
                toggle.first.click()
                _wait(0.8, log)

            # ---- Step 6: Fill email ----
            log(f"filling email: {mailbox.address}")
            page.wait_for_selector('input[name="email"]', timeout=8000)
            _fill(page, 'input[name="email"]', mailbox.address)

            # ---- Step 7: Fill birthday via selects ----
            log(f"setting birthday: {identity.birth_month}/{identity.birth_day}/{identity.birth_year}")
            selects = page.locator("select")
            count = selects.count()
            log(f"found {count} selects")
            if count >= 3:
                # Month (SELECTOR_1): select by label text
                selects.nth(0).select_option(label=MONTHS[identity.birth_month])
                _wait(0.3, log)
                # Day (SELECTOR_2): select by value (1-31 as strings)
                selects.nth(1).select_option(value=str(identity.birth_day))
                _wait(0.3, log)
                # Year (SELECTOR_2): select by value (year as string)
                selects.nth(2).select_option(value=str(identity.birth_year))
                _wait(0.3, log)

            # ---- Step 8: Click Next ----
            log("clicking Next...")
            _click_next(page)
            _wait(2, log)
            log(f"url: {page.url}")

            # ---- Step 9: Customization page → click Next (force past disabled state) ----
            _wait(3, log, "waiting for customization page")
            customize_next = page.locator('[data-testid="ocfSignupNextLink"]')
            if customize_next.count() > 0:
                log("skipping customization (force click)...")
                customize_next.first.click(force=True)
                _wait(2, log)

            # ---- Step 10: Click "Sign up" ----
            signup_btn = page.locator('[data-testid="ocfSignupButton"], button:has-text("Sign up")')
            if signup_btn.count() > 0:
                log("clicking Sign up...")
                signup_btn.first.click()
                _wait(2, log)
            log(f"url: {page.url}")

            # ---- Step 11: Wait for OTP screen ----
            log("waiting for OTP screen...")
            try:
                page.wait_for_selector(
                    'input[data-testid="ocfEnterTextTextInput"], '
                    'input[autocomplete="one-time-code"], '
                    'input[name="verfication_code"]',
                    timeout=25000,
                )
                log("OTP screen detected")
            except Exception as e:
                log(f"OTP screen wait: {e}")
                log(f"page text: {page.inner_text('body')[:200]!r}")

            # ---- Step 12: Poll mail.tm ----
            log("polling mail.tm for OTP...")
            otp = mail_client.poll_for_otp(mailbox, timeout=120)
            log(f"OTP: {otp}")

            # ---- Step 13: Enter OTP ----
            log("entering OTP...")
            otp_input = page.locator(
                'input[data-testid="ocfEnterTextTextInput"], '
                'input[autocomplete="one-time-code"]'
            )
            otp_input.first.fill(otp)
            _wait(0.5, log)
            _click_next(page)
            _wait(2, log)
            log(f"url after OTP: {page.url}")

            # ---- Step 14: Password ----
            log("setting password...")
            try:
                page.wait_for_selector('input[type="password"]', timeout=12000)
                _fill(page, 'input[type="password"]', identity.password)
                _wait(0.5, log)
                _click_next(page)
                _wait(2, log)
            except Exception as e:
                log(f"password step skipped: {e}")
            log(f"url after password: {page.url}")

            # ---- Step 15: Wait for home or skip remaining prompts ----
            log("completing onboarding...")
            _skip_prompts(page, log, max_attempts=8)
            log(f"final url: {page.url}")

            # ---- Step 16: Extract cookies ----
            cookies = {c["name"]: c["value"] for c in ctx.cookies()}
            auth_token = cookies.get("auth_token", "")
            ct0 = cookies.get("ct0", "")
            twid = cookies.get("twid", "").replace("u%3D", "").replace("u=", "")
            log(f"auth_token={'SET' if auth_token else 'MISSING'} ct0={'SET' if ct0 else 'MISSING'}")

            if not auth_token:
                _wait(5, log, "waiting for cookie")
                cookies = {c["name"]: c["value"] for c in ctx.cookies()}
                auth_token = cookies.get("auth_token", "")
                ct0 = cookies.get("ct0", "")
                twid = cookies.get("twid", "").replace("u%3D", "").replace("u=", "")

            if not auth_token:
                log(f"body: {page.inner_text('body')[:300]!r}")
                raise RuntimeError("auth_token not found in cookies after signup")

            # ---- Step 17: Get screen_name ----
            screen_name = _get_screen_name(page, ctx, log) or identity.username

            return BrowserAccount(
                auth_token=auth_token,
                ct0=ct0,
                user_id=twid,
                screen_name=screen_name,
            )

        finally:
            ctx.close()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _wait(seconds: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {seconds}s ({msg})...")
    time.sleep(seconds)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    # For comma-separated selectors, use first visible match
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=8000)
    el.click()
    time.sleep(0.3)
    el.type(text, delay=delay)
    time.sleep(0.4)


def _click_next(page) -> None:
    """Click the primary Next/Continue button."""
    for sel in [
        '[data-testid="ocfSignupNextLink"]',
        '[data-testid="ocfEnterTextNextButton"]',
        'button:has-text("Next")',
        '[role="button"]:has-text("Next")',
    ]:
        btn = page.locator(sel)
        if btn.count() > 0:
            btn.first.click()
            return


def _skip_prompts(page, log, max_attempts: int = 8) -> None:
    """Click through remaining onboarding prompts (skip / next)."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url
        if "/home" in url or "compose" in url:
            log(f"reached home at attempt {attempt}")
            return

        skipped = False
        for sel in [
            'button:has-text("Skip for now")',
            '[role="button"]:has-text("Skip for now")',
            'button:has-text("Skip")',
            'button:has-text("Next")',
            '[data-testid="ocfSignupNextLink"]',
        ]:
            btn = page.locator(sel)
            if btn.count() > 0:
                log(f"  skipping: {sel}")
                btn.first.click()
                skipped = True
                break

        if not skipped:
            log(f"  no skip button at attempt {attempt}, url={url}")
            break


def _get_screen_name(page, ctx, log) -> str:
    """Try to extract screen_name from the profile or settings page."""
    try:
        page.goto("https://x.com/settings/account", timeout=12000)
        time.sleep(3)
        # Look for username in URL or page content
        url = page.url
        if "/settings" in url:
            # Try to find @username in the page
            text = page.inner_text("body")
            import re
            m = re.search(r"@([A-Za-z0-9_]{1,15})", text)
            if m:
                log(f"screen_name from settings: {m.group(1)}")
                return m.group(1)
    except Exception as e:
        log(f"get_screen_name: {e}")
    return ""


def sign_up_via_browser_imap(
    identity: "Identity",
    email_address: str,
    imap_client,
    imap_mailbox,
    headless: bool = True,
    verbose: bool = False,
) -> BrowserAccount:
    """Same as sign_up_via_browser but uses IMAP for OTP instead of mail.tm."""
    from .email import Mailbox as FakeMailbox
    from .email import MailTmClient

    # Create a shim that wraps imap polling
    class _ImapShim:
        def poll_for_otp(self, mb, timeout=120):
            return imap_client.poll_for_otp(imap_mailbox, timeout=timeout)

    fake_mb = FakeMailbox(address=email_address, password="", token="")
    shim = _ImapShim()

    # Temporarily monkey-patch to reuse main function
    result = _sign_up_core(
        identity=identity,
        email_address=email_address,
        otp_poller=shim,
        fake_mailbox=fake_mb,
        headless=headless,
        verbose=verbose,
    )
    return result


def _sign_up_core(
    identity: "Identity",
    email_address: str,
    otp_poller,
    fake_mailbox,
    headless: bool = True,
    verbose: bool = False,
) -> BrowserAccount:
    """Core signup driver — used by both mail.tm and IMAP paths."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser-reg] {msg}", flush=True)

    log(f"signing up {email_address} as {identity.display_name!r}")

    user_data = tempfile.mkdtemp(prefix="x_reg_")

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel="chrome",
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            log("visiting x.com to warm up...")
            try:
                page.goto("https://x.com", timeout=25000)
            except Exception as e:
                log(f"x.com warn: {e}")
            _wait(8, log, "warming up")

            log("navigating to signup...")
            try:
                page.goto("https://x.com/i/flow/signup", timeout=25000)
            except Exception as e:
                log(f"signup nav warn: {e}")
            _wait(3, log)

            log("clicking 'Create account'...")
            create_btn = page.locator(
                'a[href="/i/flow/signup"] span:has-text("Create account"), '
                'span:has-text("Create account"), '
                'a:has-text("Create account")'
            )
            if create_btn.count() > 0:
                create_btn.first.click()
                _wait(2, log)
            log(f"url: {page.url}")

            log(f"filling name: {identity.display_name!r}")
            page.wait_for_selector('input[name="name"]', timeout=15000)
            _fill(page, 'input[name="name"]', identity.display_name)

            _wait(0.5, log)
            toggle = page.locator('span:has-text("Use email instead")')
            if toggle.count() > 0:
                log("switching to email input")
                toggle.first.click()
                _wait(0.8, log)

            log(f"filling email: {email_address}")
            page.wait_for_selector('input[name="email"]', timeout=8000)
            _fill(page, 'input[name="email"]', email_address)

            log(f"setting birthday: {identity.birth_month}/{identity.birth_day}/{identity.birth_year}")
            selects = page.locator("select")
            count = selects.count()
            log(f"found {count} selects")
            if count >= 3:
                selects.nth(0).select_option(label=MONTHS[identity.birth_month])
                _wait(0.3, log)
                selects.nth(1).select_option(value=str(identity.birth_day))
                _wait(0.3, log)
                selects.nth(2).select_option(value=str(identity.birth_year))
                _wait(0.3, log)

            log("clicking Next...")
            _click_next(page)
            _wait(2, log)
            log(f"url: {page.url}")

            _wait(3, log, "waiting for customization page")
            customize_next = page.locator('[data-testid="ocfSignupNextLink"]')
            if customize_next.count() > 0:
                log("skipping customization (force click)...")
                customize_next.first.click(force=True)
                _wait(2, log)

            signup_btn = page.locator('[data-testid="ocfSignupButton"], button:has-text("Sign up")')
            if signup_btn.count() > 0:
                log("clicking Sign up...")
                signup_btn.first.click()
                _wait(2, log)
            log(f"url: {page.url}")

            log("waiting for OTP screen...")
            try:
                page.wait_for_selector(
                    'input[data-testid="ocfEnterTextTextInput"], '
                    'input[autocomplete="one-time-code"], '
                    'input[name="verfication_code"]',
                    timeout=25000,
                )
                log("OTP screen detected")
            except Exception as e:
                log(f"OTP screen wait: {e}")
                body_text = page.inner_text('body')[:200]
                log(f"page text: {body_text!r}")
                if "oops" in body_text.lower() or "wrong" in body_text.lower():
                    raise RuntimeError(f"X rejected signup: {body_text[:100]}")

            log("polling for OTP...")
            otp = otp_poller.poll_for_otp(fake_mailbox, timeout=120)
            log(f"OTP: {otp}")

            log("entering OTP...")
            otp_input = page.locator(
                'input[data-testid="ocfEnterTextTextInput"], '
                'input[autocomplete="one-time-code"]'
            )
            otp_input.first.fill(otp)
            _wait(0.5, log)
            _click_next(page)
            _wait(2, log)
            log(f"url after OTP: {page.url}")

            log("setting password...")
            try:
                page.wait_for_selector('input[type="password"]', timeout=12000)
                _fill(page, 'input[type="password"]', identity.password)
                _wait(0.5, log)
                _click_next(page)
                _wait(2, log)
            except Exception as e:
                log(f"password step skipped: {e}")
            log(f"url after password: {page.url}")

            log("completing onboarding...")
            _skip_prompts(page, log, max_attempts=8)
            log(f"final url: {page.url}")

            cookies = {c["name"]: c["value"] for c in ctx.cookies()}
            auth_token = cookies.get("auth_token", "")
            ct0 = cookies.get("ct0", "")
            twid = cookies.get("twid", "").replace("u%3D", "").replace("u=", "")
            log(f"auth_token={'SET' if auth_token else 'MISSING'} ct0={'SET' if ct0 else 'MISSING'}")

            if not auth_token:
                _wait(5, log, "waiting for cookie")
                cookies = {c["name"]: c["value"] for c in ctx.cookies()}
                auth_token = cookies.get("auth_token", "")
                ct0 = cookies.get("ct0", "")
                twid = cookies.get("twid", "").replace("u%3D", "").replace("u=", "")

            if not auth_token:
                raise RuntimeError("auth_token not found in cookies after signup")

            screen_name = _get_screen_name(page, ctx, log) or identity.username

            return BrowserAccount(
                auth_token=auth_token,
                ct0=ct0,
                user_id=twid,
                screen_name=screen_name,
            )

        finally:
            ctx.close()
