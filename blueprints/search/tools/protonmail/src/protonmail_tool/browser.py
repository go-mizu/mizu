"""Patchright browser automation for Proton Mail.

Registration flow (account.proton.me/signup):
  1. Choose "Free" plan
  2. Fill username
  3. Fill + confirm password
  4. Solve captcha (manual in --no-headless mode)
  5. Skip recovery email / phone
  6. Complete onboarding → email is username@proton.me

Inbox reading flow (mail.proton.me):
  1. Log in with username + password
  2. Navigate to inbox
  3. Find and return verification link URLs
"""
from __future__ import annotations

import os
import platform
import re
import tempfile
import time


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox",
                 "--disable-dev-shm-usage", "--disable-gpu"]
    return args


def _ensure_display() -> None:
    if platform.system() != "Linux" or os.environ.get("DISPLAY"):
        return
    import shutil, subprocess
    xvfb = shutil.which("Xvfb")
    if xvfb:
        display = ":99"
        proc = subprocess.Popen([xvfb, display, "-screen", "0", "1280x900x24"],
                                 stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        import atexit; atexit.register(proc.kill)
        time.sleep(0.5)
        os.environ["DISPLAY"] = display


def _wait(s: float, log=None, msg: str = "") -> None:
    if log and msg:
        log(f"waiting {s}s ({msg})...")
    time.sleep(s)


def _fill(page, selector: str, text: str, delay: int = 55) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=12000)
    el.click(); time.sleep(0.3)
    el.type(text, delay=delay); time.sleep(0.4)


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            if page.locator(sel).count() > 0:
                _fill(page, sel, text)
                if log: log(f"filled via: {sel}")
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
                if log: log(f"clicked: {sel}")
                return sel
        except Exception:
            continue
    return None


def _body(page, max_chars: int = 500) -> str:
    try:
        return page.inner_text("body")[:max_chars].lower()
    except Exception:
        return ""


def _open_context(p, headless: bool, user_data: str):
    import shutil
    channel = "chrome" if shutil.which("google-chrome") or shutil.which("google-chrome-stable") else None
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
    username: str,
    password: str,
    display_name: str = "",
    headless: bool = False,
    verbose: bool = True,
) -> str:
    """Register a new Proton Mail account. Returns the full email address."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            print(f"[{time.strftime('%H:%M:%S')}] [proton] {msg}", flush=True)

    log(f"registering username: {username}")
    user_data = tempfile.mkdtemp(prefix="pm_reg_")

    with sync_playwright() as p:
        ctx = _open_context(p, headless, user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        try:
            # ── Step 1: Open signup page ──────────────────────────────────
            log("opening account.proton.me/signup ...")
            page.goto("https://account.proton.me/signup", timeout=60000)
            try:
                page.wait_for_load_state("networkidle", timeout=30000)
            except Exception:
                pass  # SPA may never reach networkidle; continue anyway
            _wait(2, log)
            log(f"url: {page.url}")

            # ── Step 2: Choose Free plan if shown ─────────────────────────
            _click_first(page, [
                'button:has-text("Get Proton for free")',
                'button:has-text("Get started for free")',
                'button:has-text("Free")',
                '[data-testid="plan-free"] button',
                'button[aria-label*="free" i]',
            ], log)
            _wait(2, log)

            # ── Step 3: Fill username ─────────────────────────────────────
            log(f"filling username: {username}")
            # Proton username might be inside an iframe
            username_filled = False

            # Try direct input first
            for sel in ['input[id="username"]', 'input[name="username"]',
                        'input[placeholder*="username" i]', 'input[autocomplete="username"]']:
                try:
                    if page.locator(sel).count() > 0:
                        _fill(page, sel, username)
                        log(f"  username via: {sel}")
                        username_filled = True
                        break
                except Exception:
                    pass

            # Proton embeds the username field in an iframe
            if not username_filled:
                log("  trying iframe for username...")
                for frame in page.frames:
                    for sel in ['input[id="username"]', 'input[name="username"]',
                                'input[placeholder*="username" i]']:
                        try:
                            if frame.locator(sel).count() > 0:
                                el = frame.locator(sel).first
                                el.click(); time.sleep(0.3)
                                el.type(username, delay=55); time.sleep(0.4)
                                log(f"  username in iframe via: {sel}")
                                username_filled = True
                                break
                        except Exception:
                            pass
                    if username_filled:
                        break

            _wait(0.5, log)

            # ── Step 4: Fill password ─────────────────────────────────────
            log("filling password...")
            for sel in ['input[id="password"]', 'input[name="password"]',
                        'input[type="password"]']:
                try:
                    inputs = page.locator(sel).all()
                    if inputs:
                        inputs[0].click(); time.sleep(0.2)
                        inputs[0].type(password, delay=55); time.sleep(0.3)
                        log(f"  password[0] via: {sel}")
                        if len(inputs) > 1:
                            inputs[1].click(); time.sleep(0.2)
                            inputs[1].type(password, delay=55); time.sleep(0.3)
                            log(f"  password[1] (confirm) via: {sel}")
                        break
                except Exception:
                    pass
            _wait(0.5, log)

            # ── Step 5: Submit / Continue ─────────────────────────────────
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Continue")',
                'button:has-text("Create account")',
                'button:has-text("Next")',
            ], log)
            _wait(3, log, "after submit")
            log(f"url after submit: {page.url}")

            # ── Step 6: Wait for captcha solve (up to 5 min) ─────────────
            log("waiting for captcha / human verification (solve manually)...")
            deadline = time.time() + 300
            while time.time() < deadline:
                url = page.url
                body = _body(page, 300)
                # Detect successful progression: URL changed away from signup
                # or onboarding/inbox appeared
                past_captcha = (
                    "account.proton.me/signup" not in url
                    or "congratulations" in body
                    or "recovery" in body
                    or "skip" in body
                    or "set up" in body
                    or "mail.proton.me" in url
                    or "proton.me/u/" in url
                )
                if past_captcha:
                    log(f"captcha passed — url: {url}")
                    break
                time.sleep(3)

            _wait(2, log)

            # ── Step 7: Skip recovery email / phone ───────────────────────
            log("skipping recovery options...")
            for _ in range(8):
                clicked = _click_first(page, [
                    'button:has-text("Skip")',
                    'button:has-text("Maybe later")',
                    'button:has-text("No, thanks")',
                    'button:has-text("Continue")',
                    'button:has-text("Next")',
                    'button:has-text("Done")',
                    'button:has-text("Get started")',
                    'button:has-text("Go to inbox")',
                    'button:has-text("Start using")',
                ], log)
                if not clicked:
                    break
                _wait(2, log)
                url = page.url
                if "mail.proton.me" in url or "inbox" in url:
                    log(f"reached inbox: {url}")
                    break

            log(f"final url: {page.url}")
            email = f"{username}@proton.me"
            log(f"registered: {email}")
            return email

        finally:
            ctx.close()


# ---------------------------------------------------------------------------
# Inbox polling — open Proton Mail web, read new emails for a link
# ---------------------------------------------------------------------------

def wait_for_link(
    username: str,
    password: str,
    keyword: str = "",
    timeout: int = 120,
    headless: bool = False,
    verbose: bool = True,
) -> str:
    """Log in to Proton Mail web and poll inbox for a URL containing keyword."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            print(f"[{time.strftime('%H:%M:%S')}] [proton-inbox] {msg}", flush=True)

    user_data = tempfile.mkdtemp(prefix="pm_inbox_")

    with sync_playwright() as p:
        ctx = _open_context(p, headless, user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        try:
            # Log in
            log("opening mail.proton.me ...")
            page.goto("https://account.proton.me/login", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)

            _fill_first(page, ['input[id="username"]', 'input[name="username"]',
                                'input[type="text"]'], username, log)
            _wait(0.5, log)
            _fill_first(page, ['input[id="password"]', 'input[name="password"]',
                                'input[type="password"]'], password, log)
            _wait(0.5, log)
            _click_first(page, ['button[type="submit"]', 'button:has-text("Sign in")'], log)
            _wait(5, log, "login")
            log(f"url after login: {page.url}")

            # Wait for inbox to load
            deadline = time.time() + 30
            while time.time() < deadline:
                if "mail.proton.me" in page.url or "inbox" in page.url.lower():
                    break
                time.sleep(2)

            # Poll for new email containing keyword
            log(f"polling inbox for link (keyword={keyword!r}, timeout={timeout}s)...")
            seen: set[str] = set()
            deadline = time.time() + timeout

            while time.time() < deadline:
                # Click first unread message that matches
                try:
                    # Look for email rows
                    rows = page.locator('[data-shortcut-target="item-container"], .message-list-item').all()
                    for row in rows:
                        rid = row.get_attribute("data-element-id") or row.inner_text()[:30]
                        if rid in seen:
                            continue
                        # Check subject/sender text
                        text = row.inner_text()
                        if keyword and keyword.lower() not in text.lower():
                            continue
                        log(f"opening email: {text[:60]!r}")
                        row.click()
                        _wait(2, log)
                        # Get email body
                        body_html = page.inner_text("body")
                        urls = re.findall(r"https?://\S+", body_html)
                        for url in urls:
                            url = url.rstrip(".,;)")
                            if not keyword or keyword.lower() in url.lower():
                                log(f"found link: {url[:80]}")
                                return url
                        seen.add(rid)
                        # Go back to inbox
                        page.go_back()
                        _wait(1, log)
                except Exception as e:
                    log(f"inbox scan warn: {e}")

                # Reload inbox every 10s
                _wait(10, log, "waiting for email")
                try:
                    page.reload(timeout=15000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception:
                    pass

            raise TimeoutError(f"No email with link received within {timeout}s")

        finally:
            ctx.close()
