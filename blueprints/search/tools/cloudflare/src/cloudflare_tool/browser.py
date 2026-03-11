"""Patchright browser automation for Cloudflare signup and API token creation.

Registration flow:
  1. dash.cloudflare.com/sign-up → fill email + password → submit
  2. Poll mail.tm for verification email → click link
  3. Skip domain setup → accept Free plan
  4. Extract account_id from URL or API

Token creation flow:
  1. Login to dash.cloudflare.com
  2. Navigate to /profile/api-tokens → Create Token → Custom Token
  3. Set name + permissions from preset → submit
  4. Extract token value from confirmation page
"""
from __future__ import annotations

import os
import platform
import re
import tempfile
import time

from .email import MailTmClient, Mailbox


# ---------------------------------------------------------------------------
# Permission presets
# ---------------------------------------------------------------------------

# Maps preset name → list of (resource_type, resource, permission) tuples
# These map to CF's token permission UI labels
PRESETS: dict[str, list[tuple[str, str, str]]] = {
    "browser-rendering": [
        ("Account", "Browser Rendering", "Edit"),
    ],
    "workers": [
        ("Account", "Workers Scripts", "Edit"),
        ("Account", "Workers Routes", "Edit"),
    ],
    "r2": [
        ("Account", "R2 Storage", "Edit"),
    ],
    "kv": [
        ("Account", "Workers KV Storage", "Edit"),
    ],
    "dns": [
        ("Zone", "DNS", "Edit"),
    ],
    "all": [
        ("Account", "Browser Rendering", "Edit"),
        ("Account", "Workers Scripts", "Edit"),
        ("Account", "Workers Routes", "Edit"),
        ("Account", "R2 Storage", "Edit"),
        ("Account", "Workers KV Storage", "Edit"),
        ("Zone", "DNS", "Edit"),
    ],
}


# ---------------------------------------------------------------------------
# Browser helpers
# ---------------------------------------------------------------------------

def _detect_chrome_channel() -> str | None:
    """Return 'chrome' if system Chrome is available, else None (Chromium)."""
    import shutil
    # Linux: chrome is usually in PATH
    if shutil.which("google-chrome") or shutil.which("google-chrome-stable"):
        return "chrome"
    # macOS: Chrome at default install path
    if platform.system() == "Darwin":
        mac_chrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        if os.path.exists(mac_chrome):
            return "chrome"
    return None


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


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            inp = page.locator(sel)
            if inp.count() > 0:
                _fill(page, sel, text)
                if log:
                    log(f"filled via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _log_page(page, log, label: str = "", max_chars: int = 500) -> str:
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:300]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""


def _on_dash(page) -> bool:
    return "dash.cloudflare.com" in page.url


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive Cloudflare signup. Returns account_id string."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="cf_reg_")
    channel = _detect_chrome_channel()

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel=channel,
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Sign-up form ----
            log("opening dash.cloudflare.com/sign-up...")
            try:
                page.goto("https://dash.cloudflare.com/sign-up", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception as e:
                log(f"nav warn: {e}")
            _wait(2, log)
            log(f"url: {page.url}")

            # Fill email
            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
            ], mailbox.address, log)

            # Fill password
            _wait(0.5, log)
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)

            # Confirm password (CF sign-up has confirm field)
            _wait(0.3, log)
            confirm_inputs = page.locator('input[type="password"]')
            if confirm_inputs.count() >= 2:
                log("filling confirm password...")
                confirm_inputs.nth(1).click()
                time.sleep(0.2)
                confirm_inputs.nth(1).type(password, delay=55)
                time.sleep(0.3)

            # Submit
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Create account")',
                'button:has-text("Continue")',
                'input[type="submit"]',
            ], log)
            _wait(4, log, "waiting for signup response")
            log(f"url after submit: {page.url}")

            # ---- Step 2: Email verification ----
            body_text = _log_page(page, log, "post-signup: ")
            verify_keywords = [
                "verify", "check your email", "confirmation",
                "we sent", "email sent", "verification",
            ]
            if any(kw in body_text.lower() for kw in verify_keywords):
                log("email verification required — polling mail.tm...")
                verify_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
                log(f"got verification link: {verify_link[:60]}...")
                try:
                    page.goto(verify_link, timeout=30000)
                    page.wait_for_load_state("networkidle", timeout=15000)
                except Exception as e:
                    log(f"verification nav warn: {e}")
                _wait(4, log, "post-verification")
                log(f"url after verification: {page.url}")

            # ---- Step 3: Onboarding — skip domain setup ----
            log("completing onboarding...")
            _skip_onboarding(page, log, max_attempts=15)

            # ---- Step 4: Extract account_id ----
            log(f"url before account_id extraction: {page.url}")
            account_id = _extract_account_id(page, log)

            if not account_id:
                # Try navigating to dashboard home and extracting from URL
                try:
                    page.goto("https://dash.cloudflare.com/", timeout=20000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception as e:
                    log(f"dashboard nav warn: {e}")
                _wait(3, log)
                log(f"url: {page.url}")
                account_id = _extract_account_id(page, log)

            if not account_id:
                _log_page(page, log, "account_id-fail: ")
                raise RuntimeError(
                    "Failed to extract Cloudflare account_id. "
                    f"Current URL: {page.url}"
                )

            log(f"account_id: {account_id}")
            return account_id

        finally:
            ctx.close()


def _skip_onboarding(page, log, max_attempts: int = 15) -> None:
    """Click through Cloudflare onboarding: skip domain setup, accept Free plan."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url
        log(f"  onboarding attempt {attempt}: {url}")

        # Done if on dashboard home or account page
        if re.search(r"/[0-9a-f]{32}(/|$)", url) or "/home" in url:
            log(f"  onboarding done at attempt {attempt}")
            return

        # Skip domain / add domain later
        skip_clicked = _click_first(page, [
            'button:has-text("Skip")',
            'a:has-text("Skip")',
            'button:has-text("Add later")',
            'a:has-text("Add later")',
            'button:has-text("Skip for now")',
            'a:has-text("Skip for now")',
            '[data-testid*="skip"]',
        ], log)

        if not skip_clicked:
            # Try "Continue" / "Next" / "Get started"
            skip_clicked = _click_first(page, [
                'button:has-text("Continue")',
                'button:has-text("Next")',
                'button:has-text("Get started")',
                'button:has-text("Done")',
                'button:has-text("Finish")',
                'a:has-text("Continue")',
            ], log)

        if not skip_clicked:
            log(f"  no onboarding button found at attempt {attempt}")
            # Check if a plan selection is needed
            free_plan = _click_first(page, [
                'button:has-text("Free")',
                'a:has-text("Free")',
                '[data-testid*="free"]',
                'button:has-text("Select Free")',
            ], log)
            if not free_plan:
                log("  no clickable element, stopping onboarding")
                break


def _extract_account_id(page, log) -> str:
    """Extract account_id from current page URL or page content."""
    url = page.url

    # Strategy 1: URL path contains 32-char hex account_id
    # e.g. dash.cloudflare.com/abc123.../home
    m = re.search(r"/([0-9a-f]{32})(/|$)", url)
    if m:
        log(f"account_id from URL: {m.group(1)}")
        return m.group(1)

    # Strategy 2: Page HTML/text contains account_id pattern
    try:
        html = page.evaluate("document.documentElement.innerHTML")
        m = re.search(r'"account_id"\s*:\s*"([0-9a-f]{32})"', html)
        if m:
            log(f"account_id from HTML: {m.group(1)}")
            return m.group(1)
        # Try data-account-id attribute
        m = re.search(r'data-account-id="([0-9a-f]{32})"', html)
        if m:
            log(f"account_id from data attr: {m.group(1)}")
            return m.group(1)
    except Exception as e:
        log(f"HTML extraction error: {e}")

    return ""


# ---------------------------------------------------------------------------
# Token creation
# ---------------------------------------------------------------------------

def create_token_via_browser(
    email: str,
    password: str,
    token_name: str,
    preset: str = "all",
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Login to CF dashboard and create a named API token. Returns token value."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    permissions = PRESETS.get(preset, PRESETS["all"])
    log(f"creating token '{token_name}' with preset '{preset}' ({len(permissions)} permissions)")

    user_data = tempfile.mkdtemp(prefix="cf_tok_")
    channel = _detect_chrome_channel()

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel=channel,
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Login ----
            log("logging in...")
            try:
                page.goto("https://dash.cloudflare.com/login", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception as e:
                log(f"login nav warn: {e}")
            _wait(2, log)

            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
            ], email, log)
            _wait(0.5, log)
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Log in")',
                'button:has-text("Sign in")',
                'button:has-text("Continue")',
            ], log)
            _wait(5, log, "waiting for login")
            log(f"url after login: {page.url}")

            # Check for 2FA / unusual login prompts
            body = _log_page(page, log, "post-login: ")

            # ---- Step 2: Navigate to API Tokens ----
            log("navigating to API tokens...")
            try:
                page.goto(
                    "https://dash.cloudflare.com/profile/api-tokens",
                    timeout=20000,
                )
                page.wait_for_load_state("networkidle", timeout=10000)
            except Exception as e:
                log(f"api-tokens nav warn: {e}")
            _wait(3, log)
            log(f"url: {page.url}")

            if "login" in page.url.lower() or "sign-in" in page.url.lower():
                raise RuntimeError(
                    f"Not logged in — redirected to login page. URL: {page.url}"
                )

            # ---- Step 3: Create Custom Token ----
            log("clicking 'Create Token'...")
            _click_first(page, [
                'button:has-text("Create Token")',
                'a:has-text("Create Token")',
                '[data-testid*="create-token"]',
            ], log)
            _wait(2, log)

            # Select "Custom Token" (vs templates)
            log("selecting 'Custom Token'...")
            _click_first(page, [
                'button:has-text("Get started"):near(:text("Custom token"))',
                'a:has-text("Get started"):near(:text("Custom token"))',
                '[data-testid*="custom"]',
                'button:has-text("Get started")',
                'a:has-text("Get started")',
            ], log)
            _wait(2, log)
            log(f"url: {page.url}")

            # ---- Step 4: Fill token name ----
            log(f"filling token name: {token_name}")
            _fill_first(page, [
                'input[name*="token" i]',
                'input[placeholder*="token" i]',
                'input[placeholder*="name" i]',
                'input[type="text"]:first-of-type',
                'input[aria-label*="name" i]',
            ], token_name, log)
            _wait(0.5, log)

            # ---- Step 5: Add permissions ----
            log(f"adding {len(permissions)} permissions...")
            _add_token_permissions(page, permissions, log)

            # ---- Step 6: Submit ----
            _wait(1, log)
            log("submitting token creation form...")
            _click_first(page, [
                'button:has-text("Continue to summary")',
                'button:has-text("Continue")',
                'button[type="submit"]:has-text("Continue")',
            ], log)
            _wait(2, log)
            _log_page(page, log, "summary: ")

            _click_first(page, [
                'button:has-text("Create Token")',
                'button[type="submit"]:has-text("Create")',
                'button:has-text("Confirm")',
            ], log)
            _wait(3, log, "token creation")
            log(f"url after create: {page.url}")

            # ---- Step 7: Extract token value ----
            log("extracting token value...")
            token_value = _extract_token_value(page, log)

            if not token_value:
                for retry in range(5):
                    _wait(3, log, f"token retry {retry + 1}/5")
                    token_value = _extract_token_value(page, log)
                    if token_value:
                        break

            if not token_value:
                _log_page(page, log, "token-fail: ")
                raise RuntimeError(
                    "Failed to extract token value from confirmation page"
                )

            log(f"token extracted (len={len(token_value)})")
            return token_value

        finally:
            ctx.close()


def _add_token_permissions(page, permissions: list[tuple[str, str, str]], log) -> None:
    """Add permission rows to the CF Custom Token creation form."""
    for i, (resource_type, resource, permission) in enumerate(permissions):
        log(f"  adding permission: {resource_type} > {resource} > {permission}")

        # Click "Add more" or "+" button for rows after the first
        if i > 0:
            _click_first(page, [
                'button:has-text("Add more")',
                'button:has-text("Add permission")',
                'button[aria-label*="add" i]',
                'button:has-text("+")',
            ], log)
            _wait(0.5, log)

        # Each permission row has two dropdowns: category and level
        # The new row is typically the last row in the permissions table
        rows = page.locator('[data-testid*="permission-row"], .permission-row, tr:has(select)')
        row = rows.last if rows.count() > 0 else page

        # Select resource type (Account/Zone)
        type_selects = row.locator('select, [role="combobox"]')
        if type_selects.count() >= 1:
            try:
                type_selects.first.select_option(label=resource_type)
                _wait(0.3, log)
            except Exception:
                # Try clicking and selecting from dropdown
                type_selects.first.click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{resource_type}")'], log)

        # Select specific resource (e.g., "Workers Scripts")
        if type_selects.count() >= 2:
            try:
                type_selects.nth(1).select_option(label=resource)
                _wait(0.3, log)
            except Exception:
                type_selects.nth(1).click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{resource}")'], log)

        # Select permission level (Edit/Read)
        if type_selects.count() >= 3:
            try:
                type_selects.nth(2).select_option(label=permission)
                _wait(0.3, log)
            except Exception:
                type_selects.nth(2).click()
                _wait(0.3, log)
                _click_first(page, [f'[role="option"]:has-text("{permission}")'], log)


def _extract_token_value(page, log) -> str:
    """Extract the API token value from CF confirmation page."""
    # Strategy 1: specific CF token display elements
    for sel in [
        '[data-testid*="token-value"]',
        '[data-testid*="copy-token"]',
        'input[readonly][type="text"]',
        'code',
        'pre',
        '.copy-input input',
        '[aria-label*="token" i] input',
    ]:
        try:
            els = page.locator(sel)
            if els.count() > 0:
                el = els.first
                text = el.get_attribute("value") or el.inner_text() or ""
                text = text.strip()
                if len(text) > 20 and " " not in text:
                    log(f"token via {sel!r}: {text[:20]}...")
                    return text
        except Exception:
            pass

    # Strategy 2: page text regex for CF token patterns
    # CF tokens look like: abc123_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
    _CF_TOKEN_RE = re.compile(r"[a-zA-Z0-9]{8}_[a-zA-Z0-9_\-]{30,}")
    try:
        body = page.inner_text("body")
        m = _CF_TOKEN_RE.search(body)
        if m:
            candidate = m.group(0)
            log(f"token via body regex: {candidate[:20]}...")
            return candidate
    except Exception:
        pass

    # Strategy 3: localStorage (targeted keys only)
    _TOKEN_KEYS = {"token", "api_token", "cf_token", "apiToken", "cfToken"}
    try:
        keys = page.evaluate("() => Object.keys(localStorage)")
        for key in keys:
            if key not in _TOKEN_KEYS:
                continue
            val = page.evaluate("(k) => localStorage.getItem(k)", key)
            if val and _CF_TOKEN_RE.search(val):
                log(f"token via localStorage[{key!r}]")
                return val
    except Exception:
        pass

    return ""
