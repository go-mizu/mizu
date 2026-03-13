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
            _wait(3, log)

            # ── Step 3: Wait for username field (may be in iframe) ────────
            log(f"waiting for username field (up to 60s)...")
            username_frame = None
            username_sel = None
            _USERNAME_SELS = [
                'input[id="username"]', 'input[name="username"]',
                'input[placeholder*="username" i]', 'input[autocomplete="username"]',
            ]
            deadline_u = time.time() + 60
            while time.time() < deadline_u:
                # Log all frames for debugging
                frame_urls = [f.url for f in page.frames]
                log(f"  frames ({len(frame_urls)}): {frame_urls}")

                # Check iframes first (Proton puts username in iframe, main page has hidden duplicate)
                for frame in page.frames:
                    if frame == page.main_frame:
                        continue
                    for sel in _USERNAME_SELS:
                        try:
                            loc = frame.locator(sel)
                            if loc.count() > 0 and loc.first.is_visible():
                                username_frame = frame
                                username_sel = sel
                                break
                        except Exception:
                            pass
                    if username_sel:
                        break

                # Fall back to main page only if visible
                if not username_sel:
                    for sel in _USERNAME_SELS:
                        try:
                            loc = page.locator(sel)
                            if loc.count() > 0 and loc.first.is_visible():
                                username_frame = page
                                username_sel = sel
                                break
                        except Exception:
                            pass

                if username_sel:
                    break
                time.sleep(3)

            if username_frame and username_sel:
                log(f"  found username via {username_sel!r} in frame={username_frame.url[:60]}")
                el = username_frame.locator(username_sel).first
                el.click(); time.sleep(0.3)
                el.fill("")
                el.type(username, delay=55); time.sleep(0.5)
            else:
                log("  WARNING: username field not found — fill manually")

            # ── Step 4: Fill password ─────────────────────────────────────
            log("filling password...")
            # Use the same frame as username (Proton embeds all fields in the same iframe)
            pwd_frame = username_frame if username_frame else page
            pwd_filled = False
            _PWD_SELS = ['input[id="password"]', 'input[name="password"]', 'input[type="password"]']
            # Also try the other frames if not found in username_frame
            frames_to_try = [pwd_frame] + [f for f in page.frames if f != pwd_frame and f != page.main_frame] + [page]
            for frame in frames_to_try:
                for sel in _PWD_SELS:
                    try:
                        inputs = [loc for loc in frame.locator(sel).all() if loc.is_visible()]
                        if inputs:
                            inputs[0].click(); time.sleep(0.2)
                            inputs[0].fill(""); inputs[0].type(password, delay=55); time.sleep(0.3)
                            log(f"  password[0] via {sel!r} in frame={frame.url[:60]}")
                            if len(inputs) > 1:
                                inputs[1].click(); time.sleep(0.2)
                                inputs[1].fill(""); inputs[1].type(password, delay=55); time.sleep(0.3)
                                log(f"  password[1] (confirm) via {sel!r}")
                            pwd_filled = True
                            break
                    except Exception:
                        pass
                if pwd_filled:
                    break
            if not pwd_filled:
                log("  WARNING: password field not found — fill manually")
            _wait(0.5, log)

            # ── Step 5: Submit ────────────────────────────────────────────
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
            # Navigate directly to mail - if not logged in, fill credentials
            log("opening mail.proton.me/inbox ...")
            page.goto("https://mail.proton.me/u/0/inbox", timeout=60000)
            try:
                page.wait_for_load_state("networkidle", timeout=20000)
            except Exception:
                pass
            _wait(3, log)
            log(f"url: {page.url}")

            # If redirected to login, fill credentials
            if "login" in page.url or "account.proton.me" in page.url:
                log("not logged in — filling credentials ...")
                _fill_first(page, ['input[id="username"]', 'input[name="username"]',
                                    'input[type="text"]'], username, log)
                _wait(0.5, log)
                _fill_first(page, ['input[id="password"]', 'input[name="password"]',
                                    'input[type="password"]'], password, log)
                _wait(0.5, log)
                _click_first(page, ['button[type="submit"]', 'button:has-text("Sign in")'], log)
                _wait(6, log, "login")
                log(f"url after login: {page.url}")
                # Navigate to inbox after login
                if "mail.proton.me" not in page.url:
                    page.goto("https://mail.proton.me/u/0/inbox", timeout=30000)
                    try:
                        page.wait_for_load_state("networkidle", timeout=20000)
                    except Exception:
                        pass
                    _wait(3, log)
                    log(f"url after inbox nav: {page.url}")

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

            def _dismiss_modals(pg) -> None:
                """Dismiss any overlay modals blocking the inbox."""
                for sel in [
                    'button[data-testid="modal-close-button"]',
                    'button:has-text("Got it")',
                    'button:has-text("Dismiss")',
                    'button:has-text("Close")',
                    'button:has-text("Skip")',
                    'button[aria-label="Close"]',
                ]:
                    try:
                        btn = pg.locator(sel)
                        if btn.count() > 0 and btn.first.is_visible():
                            btn.first.click()
                            time.sleep(0.5)
                    except Exception:
                        pass

            def _scan_folder(folder_url: str) -> str:
                """Scan a Proton Mail folder for a link. Returns URL or empty string."""
                try:
                    if page.url != folder_url:
                        page.goto(folder_url, timeout=20000)
                        try:
                            page.wait_for_load_state("networkidle", timeout=10000)
                        except Exception:
                            pass
                        _wait(2, log)
                except Exception:
                    pass

                _dismiss_modals(page)

                try:
                    # Broad selector covering multiple Proton Mail UI versions
                    row_sel = (
                        '[data-shortcut-target="item-container"], '
                        '.message-list-item, '
                        '[data-element-id], '
                        'li[role="option"], '
                        'div[role="row"]'
                    )
                    rows = page.locator(row_sel).all()
                    log(f"  found {len(rows)} email rows in {page.url}")
                    for row in rows:
                        rid = row.get_attribute("data-element-id") or row.inner_text()[:30]
                        if rid in seen:
                            continue
                        text = row.inner_text()
                        # Open ALL emails if no keyword match in row text (keyword may be in body)
                        log(f"  email: {text[:80]!r}")
                        _dismiss_modals(page)
                        try:
                            row.click(timeout=5000)
                        except Exception:
                            row.click(force=True)
                        _wait(2, log)
                        body_html = page.inner_text("body")
                        urls = re.findall(r"https?://\S+", body_html)
                        for url in urls:
                            url = url.rstrip(".,;)")
                            if not keyword or keyword.lower() in url.lower():
                                log(f"found link: {url[:80]}")
                                return url
                        seen.add(rid)
                        page.go_back()
                        _wait(1, log)
                except Exception as e:
                    log(f"folder scan warn: {e}")
                return ""

            while time.time() < deadline:
                # Check inbox
                result = _scan_folder("https://mail.proton.me/u/0/inbox")
                if result:
                    return result
                # Check spam
                result = _scan_folder("https://mail.proton.me/u/0/spam")
                if result:
                    return result
                # Reload inbox and wait
                _wait(10, log, "waiting for email")
                try:
                    page.goto("https://mail.proton.me/u/0/inbox", timeout=15000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception:
                    pass

            raise TimeoutError(f"No email with link received within {timeout}s")

        finally:
            ctx.close()
