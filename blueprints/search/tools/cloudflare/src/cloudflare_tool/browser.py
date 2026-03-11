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


def _browser_args(headless: bool = True) -> list[str]:
    args = [
        "--disable-blink-features=AutomationControlled",
        "--window-size=1920,1080",
        "--lang=en-US",
    ]
    if platform.system() == "Linux":
        args += [
            "--no-sandbox", "--disable-setuid-sandbox",
            "--disable-dev-shm-usage",
            "--use-angle=swiftshader",
            "--enable-webgl",
            "--ignore-gpu-blocklist",
            "--enable-unsafe-swiftshader",
        ]
    elif platform.system() == "Darwin" and headless:
        # macOS headless: SwiftShader for WebGL (GPU not available in headless)
        args += [
            "--use-angle=swiftshader",
            "--enable-webgl",
            "--ignore-gpu-blocklist",
            "--enable-unsafe-swiftshader",
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


def _handle_turnstile(page, log) -> None:
    """Wait for Turnstile to solve. Patchright patches make CF Turnstile auto-solve.

    Strategy: just wait for the submit button to become enabled, which happens
    when Turnstile has been satisfied. No manual clicking needed.
    """
    log("waiting for Turnstile / submit button to be enabled (up to 60s)...")
    try:
        page.wait_for_function(
            "() => !document.querySelector('button[type=\"submit\"]')?.disabled",
            timeout=60000,
        )
        log("submit button enabled — Turnstile solved")
        return
    except Exception:
        log("submit button still disabled after 60s — trying Turnstile click...")

    # Fallback: try clicking inside the Turnstile iframe
    try:
        ts_iframe = page.locator('iframe[src*="challenges.cloudflare.com"]').first
        box = ts_iframe.bounding_box()
        if box:
            cx = box["x"] + box["width"] / 2
            cy = box["y"] + box["height"] / 2
            page.mouse.move(cx, cy, steps=10)
            time.sleep(0.3)
            page.mouse.click(cx, cy)
            log(f"clicked Turnstile ({cx:.0f}, {cy:.0f})")
        # Wait another 15s after click
        page.wait_for_function(
            "() => !document.querySelector('button[type=\"submit\"]')?.disabled",
            timeout=15000,
        )
        log("submit button enabled after click")
    except Exception as e:
        log(f"Turnstile click fallback: {e} — continuing anyway")


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
    extract_api_key: bool = True,
) -> tuple[str, str]:
    """Drive Cloudflare signup. Returns (account_id, global_api_key).

    If extract_api_key=True, stays logged in after signup to extract the
    Global API Key from /profile/api-tokens (handles verification code via
    mail.tm while the mailbox is still active).
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    channel = _detect_chrome_channel()

    with sync_playwright() as p:
        browser = p.chromium.launch(
            channel=channel,
            headless=headless,
            args=_browser_args(headless),
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        page = ctx.new_page()
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

            # Handle Turnstile challenge
            _handle_turnstile(page, log)

            # Submit
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Create account")',
                'button:has-text("Continue")',
                'input[type="submit"]',
            ], log)
            _wait(5, log, "waiting for signup response")
            log(f"url after submit: {page.url}")

            # ---- Step 1b: Handle CF bot/Turnstile interstitial ----
            # CF may show a "Performing security verification" managed challenge.
            # With real Chrome it auto-resolves; wait up to 30s for it to clear.
            body_text = _log_page(page, log, "post-signup: ")
            bot_keywords = ["performing security verification", "ray id", "security service"]
            if any(kw in body_text.lower() for kw in bot_keywords):
                log("CF bot interstitial detected — waiting up to 30s for auto-resolve...")
                for _ in range(10):
                    time.sleep(3)
                    body_text = page.inner_text("body") or ""
                    log(f"interstitial check: url={page.url}")
                    if not any(kw in body_text.lower() for kw in bot_keywords):
                        log("interstitial resolved")
                        break
                else:
                    log("interstitial did not resolve after 30s — continuing anyway")
                body_text = page.inner_text("body") or ""

            # ---- Step 2: Email verification ----
            verify_keywords = [
                "check your email", "we sent", "email sent",
                "verify your email", "confirmation email",
            ]
            if any(kw in body_text.lower() for kw in verify_keywords):
                log("email verification required — polling mail.tm...")
                verify_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
                log(f"got verification link: {verify_link[:60]}...")
                _complete_email_verification(page, verify_link, log, email=mailbox.address, password=password)
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

            # ---- Step 5: Verify email (required before creating tokens) ----
            log("verifying email via mail.tm (step 5)...")
            try:
                verify_link = mail_client.poll_for_magic_link(mailbox, timeout=60)
                log(f"verification link: {verify_link[:80]}...")
                _complete_email_verification(page, verify_link, log, email=mailbox.address, password=password)
            except Exception as e:
                log(f"email verification failed: {e}")

            # ---- Step 6: Create API token via fetch (while logged in) ----
            api_key = ""
            if extract_api_key:
                log("creating API token via dashboard proxy...")
                try:
                    api_key = _create_token_via_fetch(
                        page, account_id, "global-api-key", log,
                    )
                    log(f"API token created: {api_key[:10]}...")
                except Exception as e:
                    log(f"API token creation failed: {e}")
                    # If email not verified, wait and retry
                    if "verify" in str(e).lower():
                        log("email verification may need more time, waiting 10s...")
                        _wait(10, log, "verification propagation")
                        try:
                            page.goto("https://dash.cloudflare.com/", timeout=20000)
                            page.wait_for_load_state("networkidle", timeout=15000)
                        except Exception:
                            pass
                        _wait(3, log)
                        try:
                            api_key = _create_token_via_fetch(
                                page, account_id, "global-api-key", log,
                            )
                            log(f"API token created on retry: {api_key[:10]}...")
                        except Exception as e2:
                            log(f"API token retry also failed: {e2}")

            return account_id, api_key

        finally:
            ctx.close()
            browser.close()


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
# Email verification helper
# ---------------------------------------------------------------------------

def _complete_email_verification(page, verify_link: str, log, email: str = "", password: str = "") -> bool:
    """Complete email verification by navigating to the link while logged in.

    CF's verification page triggers a managed challenge (~25-30s) that runs
    client-side JS.  After the challenge resolves, CF marks email_verified=True.

    The page MUST be in a logged-in session — the token is processed
    server-side once the challenge passes.

    Returns True if email_verified becomes True.
    """
    import httpx as _httpx

    log("completing email verification (navigate while logged in)...")

    # Navigate to the verification link in the current (logged-in) context
    try:
        page.goto(verify_link, timeout=30000, wait_until="domcontentloaded")
    except Exception as e:
        log(f"  nav warn: {e}")

    # CF shows a managed challenge page ("Performing security verification").
    # Wait up to 60s for the challenge to auto-resolve.
    log("waiting for CF challenge to resolve (up to 60s)...")
    for tick in range(20):
        time.sleep(3)
        try:
            body = page.inner_text("body", timeout=2000) or ""
        except Exception:
            body = ""
        if "no longer valid" in body.lower():
            log("  token expired — link no longer valid")
            return False
        if "security" not in body.lower() and body.strip():
            log(f"  challenge resolved at {tick * 3}s")
            break
        if tick % 5 == 4:
            log(f"  still waiting ({tick * 3}s)...")
    else:
        log("  challenge timeout (60s)")

    # Check email_verified — poll a few times for propagation
    cookies = {c["name"]: c["value"] for c in page.context.cookies()
               if "cloudflare.com" in c.get("domain", "")}
    client = _httpx.Client(cookies=cookies, timeout=20.0)
    try:
        for attempt in range(6):
            r = client.get("https://dash.cloudflare.com/api/v4/user")
            data = r.json()
            if data.get("success"):
                verified = data["result"].get("email_verified", False)
                log(f"  email_verified check {attempt + 1}: {verified}")
                if verified:
                    return True
            if attempt < 5:
                time.sleep(5)
    except Exception as e:
        log(f"  status check failed: {e}")
    finally:
        client.close()

    return False


# ---------------------------------------------------------------------------
# Shared login helper
# ---------------------------------------------------------------------------

def _login_to_dashboard(page, email: str, password: str, log) -> None:
    """Login to CF dashboard. Handles Turnstile + bot interstitial. Raises on failure."""
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

    _handle_turnstile(page, log)

    log("clicking submit...")
    try:
        submit = page.locator('button[type="submit"]').first
        submit.click(timeout=5000)
        log("submit clicked")
    except Exception as e:
        log(f"submit click note: {e}")
        _click_first(page, [
            'button:has-text("Log in")',
            'button:has-text("Sign in")',
            'button:has-text("Continue")',
        ], log)

    # Wait for login to redirect away from /login page (up to 90s)
    log("waiting for login redirect...")
    for tick in range(30):
        time.sleep(3)
        current_url = page.url
        log(f"  login check {tick}: url={current_url}")
        if "login" not in current_url.lower():
            log("login redirect detected")
            break
        try:
            body_text = page.inner_text("body", timeout=3000) or ""
        except Exception:
            body_text = ""
        if "incorrect" in body_text.lower() or "invalid" in body_text.lower():
            raise RuntimeError(f"Login failed: {body_text[:200]}")
        bot_keywords = ["performing security verification", "ray id", "security service"]
        if any(kw in body_text.lower() for kw in bot_keywords):
            log("CF bot interstitial — waiting for resolve...")
            continue
        # Try re-clicking submit if still on login
        if tick == 10:
            log("retrying Turnstile + submit...")
            _handle_turnstile(page, log)
            _wait(0.5, log)
            try:
                page.locator('button[type="submit"]').first.click(timeout=5000)
            except Exception:
                pass
    else:
        log("login did not redirect after 90s")

    if "login" in page.url.lower():
        raise RuntimeError(f"Login failed — still on login page: {page.url}")

    log(f"logged in: {page.url}")


# ---------------------------------------------------------------------------
# Email verification for existing accounts
# ---------------------------------------------------------------------------

def verify_email_via_browser(
    email: str,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> bool:
    """Login to CF dashboard, resend verification email, complete via mail.tm.

    Returns True if email_verified becomes True.
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    channel = _detect_chrome_channel()
    local = email.split("@")[0]
    mail_pass = f"Cf{local[:6]}!9xQ"

    with sync_playwright() as p:
        browser = p.chromium.launch(
            channel=channel,
            headless=headless,
            args=_browser_args(headless),
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        page = ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            _login_to_dashboard(page, email, password, log)
            return _verify_email_flow(page, email, mail_pass, log)
        finally:
            ctx.close()
            browser.close()


def _verify_email_flow(page, email: str, mail_password: str, log, cf_password: str = "") -> bool:
    """Complete email verification from a logged-in dashboard session.

    1. Check current email_verified status via /api/v4/user
    2. Navigate to /email-verification?token=invalid to trigger "Resend link"
    3. Click "Resend link" → CF sends new verification email
    4. Poll mail.tm for the NEW verification email
    5. Navigate to the fresh link while still logged in
    6. Wait ~30s for CF's managed challenge to auto-resolve
    7. Confirm email_verified is now True
    """
    import httpx as _httpx
    from .email import MailTmClient, Mailbox

    # Check current status
    cookie_dict = {
        c["name"]: c["value"]
        for c in page.context.cookies()
        if "cloudflare.com" in c.get("domain", "")
    }
    client = _httpx.Client(cookies=cookie_dict, timeout=30.0)
    try:
        r = client.get("https://dash.cloudflare.com/api/v4/user")
        user_data = r.json()
        if user_data.get("success"):
            user = user_data.get("result", {})
            verified = user.get("email_verified", False)
            log(f"email_verified={verified} for {user.get('email', '?')}")
            if verified:
                log("already verified!")
                return True
    except Exception as e:
        log(f"user check failed: {e}")
    finally:
        client.close()

    # Connect to mail.tm BEFORE triggering resend (so we don't miss it)
    mail_client = MailTmClient(verbose=True)
    mailbox = Mailbox(address=email, password=mail_password, id="")
    try:
        mail_client.reconnect(mailbox)
        log("reconnected to mail.tm")
    except Exception as e:
        log(f"mail.tm reconnect failed: {e}")
        mail_client.close()
        return False

    # Drain existing messages so we only get the NEW verification email
    import re as _re
    _ALL_URLS_RE = _re.compile(r"https?://[^\s\"'<>]+")
    headers = {"Authorization": f"Bearer {mail_client._token}"}
    try:
        r = mail_client._client.get("https://api.mail.tm/messages", headers=headers)
        old_ids = {m.get("id") for m in r.json().get("hydra:member", [])}
        log(f"drained {len(old_ids)} existing messages")
    except Exception:
        old_ids = set()

    # Navigate to /email-verification?token=invalid to get the "Resend link" page
    log("navigating to email-verification page to trigger resend...")
    try:
        page.goto(
            "https://dash.cloudflare.com/email-verification?token=invalid",
            timeout=20000, wait_until="domcontentloaded",
        )
    except Exception as e:
        log(f"nav warn: {e}")
    _wait(8, log, "waiting for React to render")

    # Click "Resend link"
    resent = False
    try:
        el = page.locator('a:has-text("Resend link")').first
        if el.is_visible(timeout=3000):
            el.click(timeout=3000)
            resent = True
            log("clicked 'Resend link'")
            _wait(2, log)
    except Exception as e:
        log(f"Resend link click: {e}")

    if not resent:
        log("Resend link not found — polling for existing verification email")

    # Poll mail.tm for the NEW verification link (skip old messages)
    log("polling mail.tm for new verification email...")
    deadline = time.time() + 90
    verify_link = None
    while time.time() < deadline:
        try:
            r = mail_client._client.get("https://api.mail.tm/messages", headers=headers)
            for msg in r.json().get("hydra:member", []):
                mid = msg.get("id", "")
                if mid in old_ids:
                    continue
                log(f"  new email: {msg.get('subject', '')!r}")
                full = mail_client._client.get(
                    f"https://api.mail.tm/messages/{mid}", headers=headers
                )
                body = full.json()
                text = body.get("text", "") + " " + str(body.get("html", ""))
                for u in _ALL_URLS_RE.findall(text):
                    if "email-verification" in u and "token=" in u:
                        verify_link = u.rstrip(".")
                        break
                if verify_link:
                    break
        except Exception as e:
            log(f"  poll error: {e}")
        if verify_link:
            break
        time.sleep(3)

    mail_client.close()

    if not verify_link:
        log("no verification email found")
        return False

    log(f"got verification link (len={len(verify_link)})")

    # Navigate to the link while STILL logged in and wait for CF challenge
    return _complete_email_verification(page, verify_link, log)


def verify_and_create_token(
    email: str,
    password: str,
    account_id: str,
    token_name: str = "global-api-key",
    headless: bool = True,
    verbose: bool = False,
) -> tuple[bool, str]:
    """Login, verify email if needed, then create API token. Returns (verified, token_value)."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    channel = _detect_chrome_channel()
    local = email.split("@")[0]
    mail_pass = f"Cf{local[:6]}!9xQ"

    with sync_playwright() as p:
        browser = p.chromium.launch(
            channel=channel,
            headless=headless,
            args=_browser_args(headless),
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        page = ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            _login_to_dashboard(page, email, password, log)

            # Step 1: Verify email
            verified = _verify_email_flow(page, email, mail_pass, log, cf_password=password)
            if not verified:
                log("email verification failed — cannot create token")
                return False, ""

            # Step 2: Create token
            log("email verified — creating token...")
            token = _create_token_via_fetch(page, account_id, token_name, log)
            return True, token
        except Exception as e:
            log(f"verify_and_create_token failed: {e}")
            return False, ""
        finally:
            ctx.close()
            browser.close()


# ---------------------------------------------------------------------------
# Token creation via in-browser fetch (no UI automation needed)
# ---------------------------------------------------------------------------

def create_token_via_browser_fetch(
    email: str,
    password: str,
    account_id: str,
    token_name: str = "global-api-key",
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Login to CF dashboard and create an API token via fetch(). Returns token value.

    Uses the browser's session cookies to call the CF API directly — avoids
    the complex permission UI entirely.
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    channel = _detect_chrome_channel()

    with sync_playwright() as p:
        browser = p.chromium.launch(
            channel=channel,
            headless=headless,
            args=_browser_args(headless),
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        page = ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            _login_to_dashboard(page, email, password, log)
            return _create_token_via_fetch(page, account_id, token_name, log)
        finally:
            ctx.close()
            browser.close()


def _create_token_via_fetch(page, account_id: str, token_name: str, log) -> str:
    """Create an API token using session cookies extracted from the logged-in browser.

    Extracts cookies from the browser context, then uses httpx to call the CF
    API directly (bypasses CORS restrictions of in-page fetch).
    """
    import httpx as _httpx
    import json as _json

    # Step 1: Extract session cookies from the browser
    log("extracting session cookies...")
    ctx = page.context
    all_cookies = ctx.cookies()
    # Build a cookie jar for api.cloudflare.com
    cookie_dict = {}
    for c in all_cookies:
        domain = c.get("domain", "")
        if "cloudflare.com" in domain:
            cookie_dict[c["name"]] = c["value"]

    log(f"extracted {len(cookie_dict)} cloudflare cookies")
    if not cookie_dict:
        raise RuntimeError("No cloudflare cookies found in browser context")

    headers = {"Content-Type": "application/json"}
    client = _httpx.Client(cookies=cookie_dict, headers=headers, timeout=30.0)

    try:
        # Step 2: Check email verification status and existing tokens
        log("checking account status...")
        try:
            r = client.get("https://dash.cloudflare.com/api/v4/user")
            user_data = r.json()
            if user_data.get("success"):
                user = user_data.get("result", {})
                log(f"  email: {user.get('email', '?')}")
                log(f"  user keys: {list(user.keys())}")
                # Print ALL user data to find verification status
                log(f"  email_verified: {user.get('email_verified', '?')}")
                log(f"  suspended: {user.get('suspended', '?')}")
        except Exception as e:
            log(f"  user check failed: {e}")

        # Check if we can list existing tokens (requires email verification)
        try:
            r = client.get("https://dash.cloudflare.com/api/v4/user/tokens")
            tokens_data = r.json()
            log(f"  GET /user/tokens success={tokens_data.get('success')}")
            if not tokens_data.get("success"):
                log(f"  GET /user/tokens errors: {tokens_data.get('errors', [])}")
        except Exception as e:
            log(f"  tokens check failed: {e}")

        # Step 3: Fetch permission groups
        # Try dashboard proxy first (uses session auth), then direct API
        log("fetching permission groups via CF API...")
        perm_data = None
        for base in [
            "https://dash.cloudflare.com/api/v4",
            "https://api.cloudflare.com/client/v4",
        ]:
            url = f"{base}/user/tokens/permission_groups"
            log(f"  trying {url}...")
            r = client.get(url)
            perm_data = r.json()
            if perm_data.get("success"):
                log(f"  success via {base}")
                break
            log(f"  failed: {str(perm_data.get('errors', []))[:200]}")

        if not perm_data or not perm_data.get("success"):
            raise RuntimeError(f"Failed to fetch permission groups: {perm_data.get('errors', []) if perm_data else 'no response'}")

        groups = perm_data.get("result", [])
        log(f"got {len(groups)} permission groups")

        # Step 3: Select key permissions (keep it small — large lists may trigger anti-abuse)
        target_perms = [
            "Workers Scripts Write",
            "Workers KV Storage Write",
            "Workers R2 Storage Write",
            "Workers Routes Write",
            "D1 Write",
            "Account Settings Read",
        ]
        group_by_name = {g["name"]: g["id"] for g in groups}
        selected_ids = []
        for name in target_perms:
            if name in group_by_name:
                selected_ids.append(group_by_name[name])
                log(f"  + {name}")

        # If none found, fall back to any Workers-related
        if not selected_ids:
            for g in groups:
                name = g.get("name", "")
                if "Workers" in name and "Write" in name:
                    selected_ids.append(g["id"])
                    log(f"  fallback + {name}")
                    if len(selected_ids) >= 5:
                        break

        log(f"selected {len(selected_ids)} permission groups")

        # Step 4: Create the token
        policy = {
            "effect": "allow",
            "resources": {f"com.cloudflare.api.account.{account_id}": "*"},
            "permission_groups": [{"id": pid} for pid in selected_ids],
        }
        body = {"name": token_name, "policies": [policy]}

        log(f"creating token '{token_name}'...")
        result = None
        for base in [
            "https://dash.cloudflare.com/api/v4",
            "https://api.cloudflare.com/client/v4",
        ]:
            url = f"{base}/user/tokens"
            log(f"  trying POST {url}...")
            r = client.post(url, json=body)
            result = r.json()
            if result.get("success"):
                log(f"  success via {base}")
                break
            log(f"  failed: {str(result.get('errors', []))[:200]}")

        if not result or not result.get("success"):
            raise RuntimeError(f"Failed to create token: {result.get('errors', []) if result else 'no response'}")

        token_value = result.get("result", {}).get("value", "")
        if not token_value:
            raise RuntimeError("Token created but no value returned")

        log(f"token created (len={len(token_value)})")
        return token_value

    finally:
        client.close()


def _extract_global_api_key_from_session(page, password, mail_client, mailbox, log) -> str:
    """Extract Global API Key from an already-logged-in browser session.

    Handles identity verification (email code) using the active mail_client/mailbox.
    """
    # Navigate to API tokens page
    log("navigating to /profile/api-tokens...")
    try:
        page.goto("https://dash.cloudflare.com/profile/api-tokens", timeout=20000)
        page.wait_for_load_state("networkidle", timeout=15000)
    except Exception as e:
        log(f"api-tokens nav warn: {e}")
    _wait(3, log)
    _log_page(page, log, "api-tokens: ", max_chars=800)

    # Scroll down and click "View" for Global API Key
    log("clicking 'View' for Global API Key...")
    view_clicked = False
    for sel in [
        'button:has-text("View"):right-of(:text("Global API Key"))',
        'button:has-text("View")',
    ]:
        try:
            btns = page.locator(sel)
            if btns.count() > 0:
                target = btns.last if sel == 'button:has-text("View")' else btns.first
                target.scroll_into_view_if_needed()
                _wait(0.5, log)
                target.click()
                view_clicked = True
                log("clicked 'View'")
                break
        except Exception as e:
            log(f"  '{sel}' failed: {e}")

    if not view_clicked:
        raise RuntimeError("Could not find Global API Key 'View' button")

    _wait(2, log)

    # Handle identity verification if CF asks for it
    body_text = page.inner_text("body") or ""
    if any(kw in body_text.lower() for kw in [
        "verify your identity", "verification code", "send a verification",
        "send code", "receive the code",
    ]):
        log("identity verification required — sending code...")
        _click_first(page, [
            'button:has-text("Send")',
            'button:has-text("Send code")',
            'button:has-text("Send verification")',
            'button:has-text("Receive")',
        ], log)
        _wait(2, log)

        # Poll mail.tm for the code
        log("polling mail.tm for verification code...")
        code = mail_client.poll_for_verification_code(mailbox, timeout=120)
        log(f"got verification code: {code}")

        # Enter the code
        _fill_first(page, [
            'input[type="text"]',
            'input[type="number"]',
            'input[name*="code" i]',
            'input[placeholder*="code" i]',
        ], code, log)
        _wait(0.5, log)

        _click_first(page, [
            'button:has-text("Verify")',
            'button:has-text("Confirm")',
            'button:has-text("Submit")',
            'button[type="submit"]',
        ], log)
        _wait(3, log, "post-verify")
        _log_page(page, log, "post-verify: ", max_chars=500)

        # After verification, need to click "View" again for Global API Key
        log("re-clicking 'View' after verification...")
        for sel in [
            'button:has-text("View"):right-of(:text("Global API Key"))',
            'button:has-text("View")',
        ]:
            try:
                btns = page.locator(sel)
                if btns.count() > 0:
                    target = btns.last if sel == 'button:has-text("View")' else btns.first
                    target.scroll_into_view_if_needed()
                    _wait(0.5, log)
                    target.click()
                    log("re-clicked 'View'")
                    break
            except Exception as e:
                log(f"  re-click '{sel}' failed: {e}")
        _wait(2, log)

    # Handle password confirmation modal
    body_text = page.inner_text("body") or ""
    if page.locator('input[type="password"]').count() > 0:
        log("entering password for key reveal...")
        _fill_first(page, [
            'input[type="password"]',
        ], password, log)
        _wait(0.5, log)
        _click_first(page, [
            'button:has-text("View")',
            'button[type="submit"]',
            'button:has-text("Confirm")',
        ], log)
        _wait(3, log, "waiting for key")

    # Extract the key
    api_key = _extract_global_api_key(page, log)
    if not api_key:
        for retry in range(5):
            _wait(2, log, f"key retry {retry + 1}/5")
            api_key = _extract_global_api_key(page, log)
            if api_key:
                break

    if not api_key:
        _log_page(page, log, "key-fail: ", max_chars=2000)
        raise RuntimeError("Failed to extract Global API Key from page")

    return api_key


def _extract_global_api_key(page, log) -> str:
    """Extract the Global API Key value from the page after clicking View."""
    # Strategy 1: input with the key value (usually readonly input shown after View)
    for sel in [
        'input[readonly][type="text"]',
        'input[type="text"][value]',
        'code',
        'pre',
        '[data-testid*="api-key"]',
        '[data-testid*="global"]',
    ]:
        try:
            els = page.locator(sel)
            if els.count() > 0:
                for i in range(els.count()):
                    el = els.nth(i)
                    text = el.get_attribute("value") or el.inner_text() or ""
                    text = text.strip()
                    # Global API Key is a 37-char hex string
                    if re.match(r"^[a-f0-9]{37}$", text):
                        log(f"key via {sel!r}: {text[:10]}...")
                        return text
        except Exception:
            pass

    # Strategy 2: broader text search — 37-char hex pattern
    try:
        body = page.inner_text("body")
        m = re.search(r"\b([a-f0-9]{37})\b", body)
        if m:
            log(f"key via body regex: {m.group(1)[:10]}...")
            return m.group(1)
    except Exception:
        pass

    # Strategy 3: any long hex string (32-40 chars)
    try:
        body = page.inner_text("body")
        m = re.search(r"\b([a-f0-9]{32,40})\b", body)
        if m:
            log(f"key via broad regex: {m.group(1)[:10]}...")
            return m.group(1)
    except Exception:
        pass

    return ""


# ---------------------------------------------------------------------------
# Token creation (via browser UI — Custom Token)
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

    channel = _detect_chrome_channel()

    with sync_playwright() as p:
        browser = p.chromium.launch(
            channel=channel,
            headless=headless,
            args=_browser_args(headless),
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        page = ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            _login_to_dashboard(page, email, password, log)

            # ---- Navigate to API Tokens ----
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

            if "login" in page.url.lower():
                raise RuntimeError(
                    f"Not logged in — redirected to login page. URL: {page.url}"
                )

            # ---- Create Custom Token ----
            log("clicking 'Create Token'...")
            _click_first(page, [
                'button:has-text("Create Token")',
                'a:has-text("Create Token")',
                '[data-testid*="create-token"]',
            ], log)
            _wait(2, log)

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

            # ---- Fill token name ----
            log(f"filling token name: {token_name}")
            _fill_first(page, [
                'input[name*="token" i]',
                'input[placeholder*="token" i]',
                'input[placeholder*="name" i]',
                'input[type="text"]:first-of-type',
                'input[aria-label*="name" i]',
            ], token_name, log)
            _wait(0.5, log)

            # ---- Add permissions ----
            log(f"adding {len(permissions)} permissions...")
            _add_token_permissions(page, permissions, log)

            # ---- Submit ----
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

            # ---- Extract token value ----
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
            browser.close()


def _add_token_permissions(page, permissions: list[tuple[str, str, str]], log) -> None:
    """Add permission rows to the CF Custom Token creation form."""
    for i, (resource_type, resource, permission) in enumerate(permissions):
        log(f"  adding permission: {resource_type} > {resource} > {permission}")

        if i > 0:
            _click_first(page, [
                'button:has-text("Add more")',
                'button:has-text("Add permission")',
                'button[aria-label*="add" i]',
                'button:has-text("+")',
            ], log)
            _wait(0.5, log)

        rows = page.locator('[data-testid*="permission-row"], .permission-row, tr:has(select)')
        row = rows.last if rows.count() > 0 else page

        # React Select comboboxes — scroll into view before interacting
        type_selects = row.locator('select, [role="combobox"]')

        for idx, (label, val) in enumerate([(None, resource_type), (None, resource), (None, permission)]):
            if type_selects.count() <= idx:
                break
            el = type_selects.nth(idx)
            try:
                el.scroll_into_view_if_needed()
                _wait(0.2, log)
                el.select_option(label=val)
                _wait(0.3, log)
            except Exception:
                try:
                    el.scroll_into_view_if_needed()
                    el.click(force=True)
                    _wait(0.3, log)
                    _click_first(page, [f'[role="option"]:has-text("{val}")'], log)
                    _wait(0.3, log)
                except Exception as e:
                    log(f"  permission select failed for {val}: {e}")


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
