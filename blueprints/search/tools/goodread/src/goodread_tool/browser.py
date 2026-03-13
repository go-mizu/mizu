"""Patchright browser automation for Goodreads account registration.

Flow:
  1. Open https://www.goodreads.com/user/sign_up
  2. Fill name, email, password
  3. Submit — Goodreads sends a confirmation email via mail.tm
  4. Poll mail.tm for the confirmation link
  5. Navigate to confirmation link in browser (logs the account in)
  6. Extract and return session cookies

Cookie format returned: list of dicts with name/value/domain/path/expires/...
"""
from __future__ import annotations

import os
import platform
import tempfile
import time

from .email import MailTmClient, Mailbox


# ---------------------------------------------------------------------------
# Helpers (shared with motherduck / protonmail pattern)
# ---------------------------------------------------------------------------

def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += [
            "--no-sandbox", "--disable-setuid-sandbox",
            "--disable-dev-shm-usage", "--disable-gpu",
        ]
    return args


def _ensure_display() -> None:
    if platform.system() != "Linux" or os.environ.get("DISPLAY"):
        return
    import shutil, subprocess
    xvfb = shutil.which("Xvfb")
    if xvfb:
        display = ":99"
        proc = subprocess.Popen(
            [xvfb, display, "-screen", "0", "1280x900x24"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        import atexit
        atexit.register(proc.kill)
        time.sleep(0.5)
        os.environ["DISPLAY"] = display


def _wait(s: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {s}s ({msg})...")
    time.sleep(s)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=12000)
    el.click()
    time.sleep(0.3)
    el.fill("")
    el.type(text, delay=delay)
    time.sleep(0.4)


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            if page.locator(sel).count() > 0 and page.locator(sel).first.is_visible():
                _fill(page, sel, text)
                if log:
                    log(f"filled via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _click_first(page, selectors: list[str], log=None) -> str | None:
    for sel in selectors:
        try:
            btn = page.locator(sel)
            if btn.count() > 0:
                btn.first.click()
                if log:
                    log(f"clicked: {sel}")
                return sel
        except Exception:
            continue
    return None


def _body_text(page, max_chars: int = 500) -> str:
    try:
        return page.inner_text("body")[:max_chars]
    except Exception:
        return ""


def _open_context(p, headless: bool, user_data: str):
    import shutil
    channel = (
        "chrome"
        if shutil.which("google-chrome") or shutil.which("google-chrome-stable")
        else None
    )
    return p.chromium.launch_persistent_context(
        user_data_dir=user_data,
        channel=channel,
        headless=headless,
        args=_browser_args(),
        viewport={"width": 1280, "height": 900},
        locale="en-US",
    )


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_via_browser(
    name: str,
    email: str,
    password: str,
    mail_client: MailTmClient,
    mailbox: Mailbox,
    headless: bool = True,
    verbose: bool = False,
) -> list[dict]:
    """Register a Goodreads account and return session cookies.

    Returns list of cookie dicts (Playwright format):
        [{"name": "...", "value": "...", "domain": "...", "path": "...", ...}]
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [goodread-browser] {msg}", flush=True)

    log(f"registering {email}")
    user_data = tempfile.mkdtemp(prefix="gr_reg_")

    with sync_playwright() as p:
        ctx = _open_context(p, headless, user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ── Step 1: Open signup page ──────────────────────────────────
            log("opening goodreads.com/user/sign_up ...")
            page.goto("https://www.goodreads.com/user/sign_up", timeout=60000)
            try:
                page.wait_for_load_state("networkidle", timeout=20000)
            except Exception:
                pass
            _wait(2, log)
            log(f"url: {page.url}")

            # If redirected to sign_in (already logged in?), bail early
            if "sign_in" in page.url:
                log("redirected to sign_in — maybe already logged in?")

            # ── Step 2: Fill signup form ──────────────────────────────────
            log(f"filling name: {name}")
            name_sel = _fill_first(page, [
                'input[name="user[name]"]',
                'input[id="user_name"]',
                'input[name="name"]',
                'input[placeholder*="name" i]',
                'input[autocomplete="name"]',
            ], name, log)
            if not name_sel:
                log("WARNING: name field not found")

            _wait(0.5, log)
            log(f"filling email: {email}")
            email_sel = _fill_first(page, [
                'input[name="user[email]"]',
                'input[id="user_email"]',
                'input[type="email"]',
                'input[name="email"]',
                'input[placeholder*="email" i]',
            ], email, log)
            if not email_sel:
                log("WARNING: email field not found")

            _wait(0.5, log)
            log("filling password")
            pwd_sel = _fill_first(page, [
                'input[name="user[password]"]',
                'input[id="user_password"]',
                'input[type="password"]',
                'input[name="password"]',
            ], password, log)
            if not pwd_sel:
                log("WARNING: password field not found")

            _wait(0.5, log)

            # ── Step 3: Submit ────────────────────────────────────────────
            log("submitting signup form...")
            _click_first(page, [
                'input[type="submit"]',
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Create account")',
                'button:has-text("Join")',
                'input[value*="Sign up" i]',
                'input[value*="Create" i]',
            ], log)
            _wait(5, log, "form submission")
            log(f"url after submit: {page.url}")

            # Log page state
            body = _body_text(page, 600)
            log(f"page body: {body[:300]!r}")

            # Check for error messages
            if any(kw in body.lower() for kw in ["already taken", "already registered", "invalid"]):
                raise RuntimeError(f"Signup error: {body[:200]}")

            # ── Step 4: Poll mail.tm for confirmation email ───────────────
            # Goodreads sends a "confirm your email" message
            check_email_keywords = [
                "confirm", "check your email", "verification", "sent you",
                "email has been sent", "activate",
            ]
            if any(kw in body.lower() for kw in check_email_keywords):
                log("email confirmation required — polling mail.tm...")
            else:
                log("no confirmation message detected — polling mail.tm anyway...")

            verify_link = mail_client.poll_for_verification_link(mailbox, timeout=120)
            log(f"verification link: {verify_link[:80]}...")

            # ── Step 5: Navigate to confirmation link ─────────────────────
            log("navigating to confirmation link...")
            try:
                page.goto(verify_link, timeout=30000)
                try:
                    page.wait_for_load_state("networkidle", timeout=15000)
                except Exception:
                    pass
            except Exception as e:
                log(f"confirmation nav warn: {e}")
            _wait(3, log, "post-confirmation load")
            log(f"url after confirmation: {page.url}")

            # After confirmation, Goodreads may redirect to the homepage or dashboard
            # Check if we're logged in
            body_after = _body_text(page, 400)
            log(f"post-confirm body: {body_after[:200]!r}")

            if "sign_in" in page.url:
                # Not logged in — try logging in manually
                log("not logged in after confirmation — attempting login...")
                page.goto("https://www.goodreads.com/user/sign_in", timeout=30000)
                try:
                    page.wait_for_load_state("networkidle", timeout=15000)
                except Exception:
                    pass
                _wait(2, log)

                _fill_first(page, [
                    'input[name="user[email]"]',
                    'input[id="user_email"]',
                    'input[type="email"]',
                    'input[name="email"]',
                ], email, log)
                _wait(0.5, log)
                _fill_first(page, [
                    'input[name="user[password]"]',
                    'input[id="user_password"]',
                    'input[type="password"]',
                    'input[name="password"]',
                ], password, log)
                _wait(0.5, log)
                _click_first(page, [
                    'input[type="submit"]',
                    'button[type="submit"]',
                    'button:has-text("Sign in")',
                    'input[value*="Sign in" i]',
                ], log)
                _wait(5, log, "login")
                log(f"url after manual login: {page.url}")

            # ── Step 6: Extract cookies ───────────────────────────────────
            log("extracting cookies...")
            all_cookies = ctx.cookies()
            # Keep only goodreads.com cookies
            gr_cookies = [
                c for c in all_cookies
                if "goodreads" in c.get("domain", "").lower()
            ]
            if not gr_cookies:
                # If no domain-filtered cookies, keep all (may be needed)
                gr_cookies = all_cookies
            log(f"extracted {len(gr_cookies)} cookies: {[c['name'] for c in gr_cookies]}")

            if not gr_cookies:
                raise RuntimeError("No cookies extracted — registration may have failed")

            return gr_cookies

        finally:
            ctx.close()


# ---------------------------------------------------------------------------
# Cookie test — verify cookies can authenticate a protected request
# ---------------------------------------------------------------------------

def test_cookies(cookies: list[dict], verbose: bool = False) -> str | None:
    """Open a browser with stored cookies, fetch a login-gated page.

    Returns user_id string if logged in, or None on failure.
    """
    import httpx

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [goodread-test] {msg}", flush=True)

    # Convert cookie dicts to httpx cookie format
    jar = {}
    for c in cookies:
        name = c.get("name", "")
        value = c.get("value", "")
        if name:
            jar[name] = value

    headers = {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
    }

    try:
        client = httpx.Client(
            cookies=jar,
            headers=headers,
            follow_redirects=True,
            timeout=20,
        )
        resp = client.get("https://www.goodreads.com/")
        client.close()
        log(f"GET / -> {resp.status_code}, final_url={resp.url}")

        body = resp.text
        # If we're logged in, Goodreads homepage shows the user's name or nav links
        if "sign_in" in str(resp.url) or "Sign in" in body[:2000]:
            log("not authenticated — cookies rejected")
            return None

        # Try to extract user_id from page
        import re
        m = re.search(r'/user/show/(\d+)', body)
        if m:
            user_id = m.group(1)
            log(f"logged in as user_id={user_id}")
            return user_id

        # Logged in but couldn't extract user_id
        log("appears logged in (no sign_in redirect)")
        return "unknown"

    except Exception as e:
        log(f"test error: {e}")
        return None
