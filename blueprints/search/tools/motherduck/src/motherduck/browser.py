"""Patchright-based MotherDuck account registration.

Flow:
  1. Open app.motherduck.com → auto-redirects to Auth0 (auth.motherduck.com)
  2. On Auth0 login page: enter email in input[name="username"], click Continue
  3. Auth0 sends a passwordless magic link to the email
  4. Poll mail.tm for magic link → navigate to it
  5. Click through onboarding prompts
  6. Navigate to Settings > Tokens → generate + extract token

Note: MotherDuck uses Auth0 Universal Login (passwordless email).
There is no separate sign-up page — new accounts are created automatically
when an unrecognised email is submitted on the Auth0 login page.

Note on Linux xvfb re-exec: When _maybe_reexec_xvfb triggers, the parent exits via
sys.exit() after the child finishes. On headless Linux (the default), never triggered.
"""
from __future__ import annotations

import os
import platform
import sys
import tempfile
import time

from .email import MailTmClient, Mailbox


def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage"]
    return args


def _maybe_reexec_xvfb(headless: bool) -> None:
    if platform.system() != "Linux" or headless or os.environ.get("DISPLAY"):
        return
    import shutil
    import subprocess
    xvfb = shutil.which("xvfb-run")
    if xvfb:
        sys.exit(subprocess.call([xvfb, "-a", sys.executable] + sys.argv))


def _wait(seconds: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {seconds}s ({msg})...")
    time.sleep(seconds)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=10000)
    el.click()
    time.sleep(0.3)
    el.type(text, delay=delay)
    time.sleep(0.4)


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive MotherDuck signup, return the API token string."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="md_reg_")

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
            # ---- Step 1: Open app — auto-redirects to Auth0 ----
            log("opening app.motherduck.com (will redirect to Auth0)...")
            try:
                page.goto("https://app.motherduck.com", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=10000)
            except Exception as e:
                log(f"landing warn: {e}")
            _wait(2, log)
            log(f"landed on: {page.url}")

            # ---- Step 2: Enter email on Auth0 login page ----
            # Auth0 Universal Login uses input[name="username"] for the email field.
            # MotherDuck uses passwordless flow — entering any email triggers a magic link.
            log(f"entering email: {mailbox.address}")
            email_filled = False
            for sel in [
                'input[name="username"]',    # Auth0 identifier-first
                'input[type="email"]',
                'input[name="email"]',
                'input[placeholder*="email" i]',
                'input[placeholder*="address" i]',
            ]:
                try:
                    inp = page.locator(sel)
                    if inp.count() > 0:
                        _fill(page, sel, mailbox.address)
                        log(f"filled email via: {sel}")
                        email_filled = True
                        break
                except Exception:
                    continue

            if not email_filled:
                log(f"WARNING: no email input found on {page.url}")

            # Submit — Auth0 uses button[name="action"] or button[value="default"]
            for sel in [
                'button[name="action"]',          # Auth0 primary
                'button[value="default"]',         # Auth0 fallback
                'button[type="submit"]',
                'button:has-text("Continue")',
                'button:has-text("Send magic link")',
                'button:has-text("Sign in")',
                '[data-action-button-primary="true"]',
            ]:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"submitted via: {sel}")
                    _wait(3, log)
                    break

            log(f"url after email submit: {page.url}")

            # ---- Step 4: Poll mail.tm for magic link ----
            log("polling mail.tm for magic link...")
            magic_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
            log(f"got magic link: {magic_link[:60]}...")

            # ---- Step 5: Navigate to magic link ----
            log("navigating to magic link...")
            try:
                page.goto(magic_link, timeout=30000)
            except Exception as e:
                log(f"magic link nav warn: {e}")
            _wait(4, log, "post-magic-link load")
            log(f"url after magic link: {page.url}")

            # ---- Step 6: Click through onboarding ----
            log("clicking through onboarding...")
            _skip_onboarding(page, log, max_attempts=10)
            log(f"url after onboarding: {page.url}")

            # ---- Step 7: Go to Settings > Tokens ----
            log("navigating to token settings...")
            try:
                page.goto("https://app.motherduck.com/settings/tokens", timeout=20000)
            except Exception as e:
                log(f"settings nav warn: {e}")
            _wait(3, log, "settings load")
            log(f"url: {page.url}")

            # ---- Step 8: Generate token ----
            log("generating token...")
            for sel in [
                'button:has-text("Generate token")',
                'button:has-text("Create token")',
                'button:has-text("New token")',
                'button:has-text("Generate")',
            ]:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"clicked: {sel}")
                    _wait(2, log)
                    break

            # ---- Step 9: Extract token ----
            log("extracting token...")
            token = _extract_token(page, ctx, log)
            if not token:
                raise RuntimeError("Failed to extract MotherDuck token from settings page")

            log(f"token extracted (len={len(token)})")
            return token

        finally:
            ctx.close()


def _skip_onboarding(page, log, max_attempts: int = 10) -> None:
    """Click through MotherDuck onboarding prompts until done."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url
        # Consider done if on main app page
        if any(x in url for x in ["/editor", "/home", "/settings", "?onboarding=done"]):
            log(f"onboarding done at attempt {attempt}")
            return

        clicked = False
        for sel in [
            'button:has-text("Skip")',
            'button:has-text("Continue")',
            'button:has-text("Next")',
            'button:has-text("Get started")',
            'button:has-text("Done")',
            '[role="button"]:has-text("Skip")',
        ]:
            btn = page.locator(sel)
            if btn.count() > 0:
                log(f"  onboarding click: {sel}")
                btn.first.click()
                clicked = True
                break

        if not clicked:
            log(f"  no onboarding button at attempt {attempt}, url={url}")
            break


def _extract_token(page, ctx, log) -> str:
    """Try multiple strategies to extract the MotherDuck API token."""
    import re

    # Strategy 1: look for token displayed in a <code> or input element
    for sel in [
        'code',
        'input[readonly]',
        '[data-testid*="token"]',
        'pre',
    ]:
        try:
            el = page.locator(sel).first
            if el.count() > 0:
                text = el.inner_text() or el.get_attribute("value") or ""
                if len(text) > 20 and "\n" not in text.strip():
                    log(f"token via selector {sel!r}: {text[:20]}...")
                    return text.strip()
        except Exception:
            pass

    # Strategy 2: scan entire page text for MotherDuck token pattern
    # MotherDuck tokens are JWT-like: "eyJ..." or long alphanumeric strings
    try:
        body = page.inner_text("body")
        for pat in [
            r"eyJ[A-Za-z0-9\-_]{30,}\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+",
            r"motherduck_token_[A-Za-z0-9_\-]{20,}",
        ]:
            m = re.search(pat, body)
            if m:
                token = m.group(0)
                log(f"token via body regex: {token[:20]}...")
                return token
    except Exception:
        pass

    # Strategy 3: localStorage
    try:
        token = page.evaluate(
            "() => localStorage.getItem('motherduck_token') || localStorage.getItem('token')"
        )
        if token and len(token) > 20:
            log(f"token via localStorage: {token[:20]}...")
            return token
    except Exception:
        pass

    # Strategy 4: cookies
    cookies = {c["name"]: c["value"] for c in ctx.cookies()}
    for key in ["motherduck_token", "token", "auth_token"]:
        if key in cookies and len(cookies[key]) > 20:
            log(f"token via cookie {key!r}")
            return cookies[key]

    return ""
