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
    time.sleep(0.2)
    # Clear any existing content first
    el.fill("")
    time.sleep(0.1)
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


def _verify_form_values(page, expected_email: str, expected_password: str, log) -> bool:
    """Check that form fields contain the expected values before submit."""
    try:
        actual_email = page.locator('input[name="email"], input[type="email"]').first.input_value()
        actual_pass = page.locator('input[name="password"], input[type="password"]').first.input_value()
        email_ok = actual_email == expected_email
        pass_ok = actual_pass == expected_password
        if not email_ok:
            log(f"  WARN: email mismatch: got {actual_email!r}, expected {expected_email!r}")
        if not pass_ok:
            log(f"  WARN: password mismatch: got len={len(actual_pass)}, expected len={len(expected_password)}")
        if not email_ok or not pass_ok:
            # Re-fill with correct values
            log("  re-filling form with correct values...")
            try:
                e = page.locator('input[name="email"], input[type="email"]').first
                e.fill("")
                e.type(expected_email, delay=30)
            except Exception:
                pass
            try:
                p = page.locator('input[name="password"], input[type="password"]').first
                p.fill("")
                p.type(expected_password, delay=30)
            except Exception:
                pass
            return False
        log(f"  form values OK: email={actual_email}, pass=len({len(actual_pass)})")
        return True
    except Exception as e:
        log(f"  form verify error: {e}")
        return False


def _log_page(page, log, label: str = "", max_chars: int = 500) -> str:
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:300]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""



def _handle_turnstile(page, log) -> None:
    """Wait for Turnstile to solve. Patchright patches make CF Turnstile auto-solve.

    Strategy:
    1. First check if Turnstile widget is present (iframe or response input)
    2. If present, wait for the token to be populated (auto-solve or manual)
    3. If not present, check submit button state as fallback
    """
    # Step 1: Wait for Turnstile widget to appear (up to 10s)
    log("checking for Turnstile widget...")
    has_turnstile = False
    try:
        page.wait_for_function(
            """() => {
                const iframes = document.querySelectorAll('iframe[src*="challenges.cloudflare.com"]');
                const resp = document.querySelector('[name="cf-turnstile-response"]');
                return iframes.length > 0 || (resp !== null);
            }""",
            timeout=10000,
        )
        has_turnstile = True
        log("Turnstile widget detected")
    except Exception:
        log("no Turnstile widget found after 10s")

    if has_turnstile:
        # Step 2: Wait for Turnstile to solve (token populated)
        # 4×15s loops so user gets feedback during manual solve
        for loop in range(4):
            log(f"waiting for Turnstile token (loop {loop + 1}/4, 15s)...")
            try:
                page.wait_for_function(
                    """() => {
                        const resp = document.querySelector('[name="cf-turnstile-response"]');
                        return resp && resp.value && resp.value.length > 10;
                    }""",
                    timeout=15000,
                )
                log("Turnstile token confirmed — solved")
                return
            except Exception:
                log(f"  not yet solved after {(loop + 1) * 15}s")
        log("Turnstile token not found after 60s — proceeding anyway")
        return

    # Step 3: No Turnstile found — check submit button as fallback
    log("no Turnstile — checking submit button state...")
    try:
        page.wait_for_function(
            """() => {
                const btn = document.querySelector('button[type="submit"]');
                return btn !== null && !btn.disabled;
            }""",
            timeout=10000,
        )
        log("submit button enabled — ready to submit")
    except Exception:
        log("submit button not ready — proceeding anyway")



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

            # Capture ALL network responses after submit for debugging
            _signup_responses = []
            def _on_response(resp):
                try:
                    method = resp.request.method
                    if method == "POST":
                        status = resp.status
                        try:
                            body = resp.text()[:300]
                        except Exception:
                            body = ""
                        _signup_responses.append((method, resp.url[:120], status, body))
                except Exception:
                    pass
            page.on("response", _on_response)

            # Validate form values before submit
            _verify_form_values(page, mailbox.address, password, log)

            # Handle Turnstile challenge + Submit (with retry)
            for submit_attempt in range(3):
                _signup_responses.clear()
                _handle_turnstile(page, log)

                _wait(0.5, log)
                # Click submit button
                _click_first(page, [
                    'button[type="submit"]',
                    'button:has-text("Sign up")',
                    'button:has-text("Create account")',
                    'button:has-text("Continue")',
                    'input[type="submit"]',
                ], log)
                # Also try Enter key as backup
                try:
                    page.keyboard.press("Enter")
                    log("pressed Enter key")
                except Exception:
                    pass

                # Wait for URL to change (AJAX signup navigates on success)
                log("waiting for signup to complete...")
                signup_done = False
                for tick in range(10):
                    time.sleep(2)
                    cur_url = page.url
                    if "sign-up" not in cur_url:
                        log(f"signup navigated: {cur_url}")
                        signup_done = True
                        break
                    # Check for error text
                    try:
                        err_text = page.inner_text("body", timeout=2000) or ""
                        err_lower = err_text.lower()
                        if "already registered" in err_lower or "account already exists" in err_lower:
                            log(f"signup error: email already registered")
                            raise RuntimeError("Email already registered")
                        if "this email" in err_lower and "cannot" in err_lower:
                            log(f"signup error: {err_text[:200]}")
                            raise RuntimeError(f"Signup blocked: {err_text[:200]}")
                    except RuntimeError:
                        raise
                    except Exception:
                        pass

                if signup_done:
                    break

                log(f"still on sign-up page after submit attempt {submit_attempt + 1}")
                # Log captured network responses for debugging
                if _signup_responses:
                    for method, url, status, body in _signup_responses:
                        log(f"  {method} {status} {url}")
                        if body:
                            log(f"    body: {body[:200]}")
                else:
                    log("  no POST requests captured after submit")
                if submit_attempt < 2:
                    log("retrying Turnstile + submit...")
                    _wait(3, log)

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

            # ---- Step 2: Email verification (fresh link) ----
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

            # ---- Step 5: Verify email (navigate fresh link while logged in) ----
            log("checking email_verified status...")
            verified = False
            try:
                import httpx as _httpx
                cookies = {c["name"]: c["value"] for c in ctx.cookies()
                           if "cloudflare.com" in c.get("domain", "")}
                hc = _httpx.Client(cookies=cookies, timeout=20.0)
                r = hc.get("https://dash.cloudflare.com/api/v4/user")
                verified = r.json().get("result", {}).get("email_verified", False)
                hc.close()
                log(f"email_verified={verified}")
            except Exception as e:
                log(f"email_verified check: {e}")

            if not verified:
                log("trying verification via mail.tm (step 5)...")
                try:
                    verify_link = mail_client.poll_for_magic_link(mailbox, timeout=60)
                    log(f"verification link: {verify_link[:80]}...")
                    _complete_email_verification(page, verify_link, log, email=mailbox.address, password=password)
                    log(f"url after verification: {page.url}")
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
        # Try any 32-char hex that looks like account ID in script tags
        m = re.search(r'"id"\s*:\s*"([0-9a-f]{32})"', html)
        if m:
            log(f"account_id from id field in HTML: {m.group(1)}")
            return m.group(1)
    except Exception as e:
        log(f"HTML extraction error: {e}")

    # Strategy 3: Call CF API via browser fetch (uses session cookies)
    try:
        result = page.evaluate("""async () => {
            try {
                const r = await fetch('https://api.cloudflare.com/client/v4/accounts?per_page=1', {
                    credentials: 'include',
                });
                const data = await r.json();
                if (data.success && data.result && data.result.length > 0) {
                    return data.result[0].id;
                }
            } catch(e) {}
            return '';
        }""")
        if result and re.match(r"^[0-9a-f]{32}$", result):
            log(f"account_id from CF API fetch: {result}")
            return result
    except Exception as e:
        log(f"CF API fetch error: {e}")

    # Strategy 4: Navigate to /api/v4/user and get account from memberships
    try:
        result = page.evaluate("""async () => {
            try {
                const r = await fetch('https://dash.cloudflare.com/api/v4/accounts?per_page=1', {
                    credentials: 'include',
                });
                const data = await r.json();
                if (data.success && data.result && data.result.length > 0) {
                    return data.result[0].id;
                }
            } catch(e) {}
            return '';
        }""")
        if result and re.match(r"^[0-9a-f]{32}$", result):
            log(f"account_id from dash API fetch: {result}")
            return result
    except Exception as e:
        log(f"dash API fetch error: {e}")

    return ""


# ---------------------------------------------------------------------------
# Email verification helper
# ---------------------------------------------------------------------------

def _complete_email_verification(page, verify_link: str, log, email: str = "", password: str = "") -> bool:
    """Complete email verification by navigating to the verification link.

    The ONLY method that works: log out (keep cookies for Turnstile), navigate to
    the verification link, fill credentials on the login form, and submit.
    CF verifies the email when you log in from the verification page.

    Returns True if email_verified becomes True.
    """
    import httpx as _httpx

    def _poll_verified(label: str) -> bool:
        """Check email_verified via API + Profile page visual check."""
        # Method 1: API check
        cookies = {c["name"]: c["value"] for c in page.context.cookies()
                   if "cloudflare.com" in c.get("domain", "")}
        client = _httpx.Client(cookies=cookies, timeout=20.0)
        try:
            for attempt in range(2):
                r = client.get("https://dash.cloudflare.com/api/v4/user")
                data = r.json()
                if data.get("success"):
                    verified = data["result"].get("email_verified", False)
                    log(f"  [{label}] API email_verified check {attempt + 1}: {verified}")
                    if verified:
                        return True
                if attempt < 1:
                    time.sleep(3)
        except Exception as e:
            log(f"  [{label}] status check failed: {e}")
        finally:
            client.close()
        return False

    def _try_verify_link(link: str, label: str) -> bool:
        """Open verify link in new window, fill form, user solves CAPTCHA.
        After login → auto-navigate to Profile → check verified."""
        browser = page.context.browser
        if not browser:
            log("  no browser ref")
            return False

        log(f"[{label}] opening verify link in new window...")
        fresh_ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="en-US",
        )
        verify_page = fresh_ctx.new_page()
        try:
            try:
                verify_page.goto(link, timeout=30000, wait_until="domcontentloaded")
            except Exception as e:
                log(f"  nav warn: {e}")

            # Wait for login form (up to 30s for interstitial)
            for tick in range(6):
                _wait(5, log)
                try:
                    body = verify_page.inner_text("body", timeout=5000) or ""
                except Exception:
                    body = ""
                if any(kw in body.lower() for kw in ["performing security", "ray id"]):
                    log(f"  interstitial ({tick})...")
                    continue
                break

            if "no longer valid" in body.lower():
                log("  link expired")
                fresh_ctx.close()
                return False

            # Fill email + password
            try:
                _fill_first(verify_page, ['input[name="email"]', 'input[type="email"]'], email, log)
                _wait(0.3, log)
                _fill_first(verify_page, ['input[name="password"]', 'input[type="password"]'], password, log)
                _wait(0.3, log)
                _verify_form_values(verify_page, email, password, log)
            except Exception as e:
                log(f"  form fill error: {e}")

            log("")
            log("  >>> Solve CAPTCHA and click 'Log in' <<<")
            log("")

            # Wait for page to navigate (up to 2 min)
            verify_url = verify_page.url
            for tick in range(24):
                time.sleep(5)
                try:
                    cur = verify_page.url
                    if cur != verify_url and "email-verification" not in cur:
                        log(f"  logged in → {cur}")
                        break
                except Exception:
                    pass
                if tick % 4 == 3:
                    log(f"  waiting... ({(tick+1)*5}s)")
            else:
                log("  timeout (2 min) — no navigation detected")
                fresh_ctx.close()
                return False

            _wait(3, log)

            # Auto-navigate to Profile page
            log("  navigating to Profile page...")
            try:
                verify_page.goto("https://dash.cloudflare.com/profile", timeout=15000)
                _wait(5, log, "profile load")
                body = verify_page.inner_text("body", timeout=5000) or ""
                log(f"  profile: {body[:400]}")
                if "verified" in body.lower():
                    log("  VERIFIED!")
                    return True
                log("  not verified on Profile page")
            except Exception as e:
                log(f"  profile error: {e}")

            # API check
            try:
                result = verify_page.evaluate("""async () => {
                    const r = await fetch('https://dash.cloudflare.com/api/v4/user',
                                          {credentials: 'include'});
                    return await r.json();
                }""")
                if result and result.get("success"):
                    v = result["result"].get("email_verified", False)
                    log(f"  API email_verified: {v}")
                    if v:
                        return True
            except Exception:
                pass

        except Exception as e:
            log(f"  [{label}] error: {e}")
        finally:
            try:
                fresh_ctx.close()
            except Exception:
                pass
        return False

    # ── Attempt 1: Use the provided verify link ──
    if _try_verify_link(verify_link, "attempt1"):
        return True

    # ── Attempt 2: Resend from original page, try fresh link ──
    log("attempt 2: resending verification email...")

    # Navigate original page to resend page (should still be logged in)
    try:
        page.goto(
            "https://dash.cloudflare.com/email-verification?token=invalid",
            timeout=20000, wait_until="domcontentloaded",
        )
    except Exception:
        pass
    _wait(5, log)

    # If we got redirected to login, re-login first
    if "login" in page.url.lower():
        log("  need to re-login first...")
        try:
            _login_to_dashboard(page, email, password, log)
            page.goto(
                "https://dash.cloudflare.com/email-verification?token=invalid",
                timeout=20000, wait_until="domcontentloaded",
            )
            _wait(5, log)
        except Exception as e:
            log(f"  re-login failed: {e}")

    try:
        el = page.locator('a:has-text("Resend link")').first
        if el.is_visible(timeout=3000):
            el.click(timeout=3000)
            log("  clicked 'Resend link'")
            _wait(5, log)
    except Exception as e:
        log(f"  resend click: {e}")

    # Poll for new verification email
    import re as _re
    _ALL_URLS_RE = _re.compile(r"https?://[^\s\"'<>]+")
    from .email import MailTmClient, Mailbox

    mail_password_local = f"Cf{email.split('@')[0][:6]}!9xQ"
    mail_client = MailTmClient(verbose=True)
    mailbox = Mailbox(address=email, password=mail_password_local, id="")
    try:
        mail_client.reconnect(mailbox)
    except Exception as e:
        log(f"  mail.tm reconnect failed: {e}")
        return False

    headers = {"Authorization": f"Bearer {mail_client._token}"}
    try:
        r = mail_client._client.get("https://api.mail.tm/messages", headers=headers)
        old_ids = {m.get("id") for m in r.json().get("hydra:member", [])}
    except Exception:
        old_ids = set()

    new_verify_link = None
    deadline = time.time() + 60
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
                body_data = full.json()
                text = body_data.get("text", "") + " " + str(body_data.get("html", ""))
                for u in _ALL_URLS_RE.findall(text):
                    if "email-verification" in u and "token=" in u:
                        new_verify_link = u.rstrip(".")
                        break
                if new_verify_link:
                    break
        except Exception as e:
            log(f"  poll error: {e}")
        if new_verify_link:
            break
        time.sleep(3)

    mail_client.close()

    if not new_verify_link:
        log("  no new verification email received")
        return False

    log(f"  got fresh verification link")
    return _try_verify_link(new_verify_link, "attempt2")


# ---------------------------------------------------------------------------
# Shared login helper
# ---------------------------------------------------------------------------

def _login_to_dashboard(page, email: str, password: str, log) -> None:
    """Login to CF dashboard. Fills form quickly, user solves CAPTCHA + clicks submit."""
    log("logging in...")
    try:
        page.goto("https://dash.cloudflare.com/login", timeout=30000)
        page.wait_for_load_state("domcontentloaded", timeout=15000)
    except Exception as e:
        log(f"login nav warn: {e}")

    # Wait for email input to be visible (React SPA may take a moment)
    log("waiting for login form to render...")
    try:
        page.locator('input[name="email"], input[type="email"]').first.wait_for(
            state="visible", timeout=15000
        )
    except Exception as e:
        log(f"email input wait warn: {e}")
    _wait(1, log)

    # Fill form
    _fill_first(page, [
        'input[name="email"]',
        'input[type="email"]',
    ], email, log)
    _wait(0.3, log)
    _fill_first(page, [
        'input[name="password"]',
        'input[type="password"]',
    ], password, log)

    # Verify values stuck
    _verify_form_values(page, email, password, log)

    print("\n>>> Solve CAPTCHA and click 'Log in' <<<\n", flush=True)

    # Loop-check if form was submitted (redirected away from /login)
    for tick in range(60):  # up to 3 min
        time.sleep(3)
        current_url = page.url
        if "login" not in current_url.lower():
            log(f"login redirect detected → {current_url}")
            break
        # Only log every 5th tick to reduce noise
        if tick % 5 == 0:
            log(f"  waiting for login submit... ({tick * 3}s)")
        try:
            body_text = page.inner_text("body", timeout=2000) or ""
        except Exception:
            body_text = ""
        if "incorrect" in body_text.lower() or "invalid" in body_text.lower():
            raise RuntimeError(f"Login failed: {body_text[:200]}")
    else:
        log("login did not redirect after 3 min")

    if "login" in page.url.lower():
        raise RuntimeError(f"Login failed — still on login page: {page.url}")

    log(f"logged in: {page.url}")





# ---------------------------------------------------------------------------
# Global API Key extraction (verifies email first, then extracts key)
# ---------------------------------------------------------------------------

def extract_global_api_key_via_browser(
    email: str,
    password: str,
    mail_password: str,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Login to CF, verify email, confirm on profile, then extract Global API Key.

    Strict workflow — NO steps can be skipped:
    1. Login to dashboard
    2. Check email_verified via /api/v4/user
    3. If not verified, run email verification flow (resend link via mail.tm,
       navigate link while logged out, solve Turnstile, log in from verify page)
    4. Navigate to /profile and confirm "Verified" text is visible
    5. Navigate to /profile/api-tokens → View Global API Key
    6. Dialog 1: enter 7-digit code → Dialog 2: Turnstile → key revealed
    7. Read key from clipboard / DOM and return.

    Returns the 37-char hex Global API Key.
    """
    from patchright.sync_api import sync_playwright
    from .email import MailTmClient, Mailbox

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
            args=["--no-sandbox"],
        )
        ctx = browser.new_context(
            viewport={"width": 1440, "height": 900},
            locale="en-US",
        )
        try:
            ctx.grant_permissions(["clipboard-read", "clipboard-write"])
        except Exception:
            pass
        page = ctx.new_page()
        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            _login_to_dashboard(page, email, password, log)

            mail_client = MailTmClient(verbose=verbose)
            mailbox = Mailbox(address=email, password=mail_password, id="")
            try:
                mail_client.reconnect(mailbox)
            except Exception as e:
                log(f"mail.tm reconnect failed: {e}")
                raise

            try:
                # Check email verified by visiting Profile page (not unreliable API)
                log("navigating to Profile to check email verification...")
                try:
                    page.goto("https://dash.cloudflare.com/profile", timeout=20000,
                              wait_until="domcontentloaded")
                except Exception as e:
                    log(f"profile nav warn: {e}")
                _wait(3, log)
                try:
                    profile_text = page.inner_text("body", timeout=5000) or ""
                    if "verified" in profile_text.lower():
                        log("email verified (Profile page confirms 'Verified')")
                    else:
                        raise RuntimeError(
                            "Email NOT verified — 'Verified' not found on Profile page. "
                            "Verify email first, then retry."
                        )
                except RuntimeError:
                    raise
                except Exception as e:
                    log(f"profile check error: {e} — proceeding anyway")

                # Extract Global API Key via /profile/api-tokens → View → dialog flow
                return _extract_global_api_key_from_session(page, password, mail_client, mailbox, log)
            finally:
                mail_client.close()
        except Exception as e:
            log(f"extract_global_api_key_via_browser failed: {e}")
            raise
        finally:
            ctx.close()
            browser.close()


def create_token_with_global_api_key(
    email: str,
    global_api_key: str,
    account_id: str,
    token_name: str = "workers-all",
    preset: str = "all",
) -> str:
    """Create an API token using the Global API Key (no browser needed).

    Uses X-Auth-Email + X-Auth-Key headers. Email must be verified first.
    Returns the new API token value.
    """
    import httpx as _httpx

    headers = {
        "X-Auth-Email": email,
        "X-Auth-Key": global_api_key,
        "Content-Type": "application/json",
    }

    # Fetch permission groups
    r = _httpx.get(
        "https://api.cloudflare.com/client/v4/user/tokens/permission_groups",
        headers=headers, timeout=30,
    )
    r.raise_for_status()
    groups = r.json().get("result", [])
    by_name = {g["name"]: g["id"] for g in groups}

    target_perms = [
        "Workers Scripts Write",
        "Workers KV Storage Write",
        "Workers R2 Storage Write",
        "Workers Routes Write",
        "D1 Write",
        "Account Settings Read",
        "Browser Rendering Write",
    ]
    selected_ids = [by_name[n] for n in target_perms if n in by_name]
    if not selected_ids:
        # Fallback: any Workers Write
        selected_ids = [g["id"] for g in groups if "Workers" in g.get("name", "") and "Write" in g.get("name", "")][:5]

    body = {
        "name": token_name,
        "policies": [{
            "effect": "allow",
            "resources": {f"com.cloudflare.api.account.{account_id}": "*"},
            "permission_groups": [{"id": pid} for pid in selected_ids],
        }],
    }
    r2 = _httpx.post(
        "https://api.cloudflare.com/client/v4/user/tokens",
        headers=headers, json=body, timeout=30,
    )
    r2.raise_for_status()
    result = r2.json()
    if not result.get("success"):
        raise RuntimeError(f"Token creation failed: {result.get('errors', [])}")
    return result["result"]["value"]




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


def _force_click_dialog_btn(page, selectors: list[str], log=None) -> str | None:
    """Click a dialog button using force=True to bypass CF's focusFallback overlay."""
    for sel in selectors:
        try:
            btn = page.locator(sel)
            if btn.count() > 0:
                btn.first.click(force=True, timeout=5000)
                if log:
                    log(f"force-clicked: {sel}")
                return sel
        except Exception:
            continue
    return None


def _try_read_key(page, log) -> str:
    """Try to read the Global API Key from clipboard and DOM after CF reveals it."""
    try:
        body = page.inner_text("body", timeout=3000) or ""
    except Exception:
        body = ""
    if "protect this key" not in body.lower() and "copied" not in body.lower():
        return _extract_global_api_key(page, log)

    log("  key dialog detected — trying to extract key...")

    # Click any element with "copy" text to trigger clipboard write
    for sel in [
        'button:has-text("Click to copy")',
        'button:has-text("Copy")',
        ':has-text("Click to copy")',
        '[data-testid*="copy"]',
        'span:has-text("copy")',
    ]:
        try:
            els = page.locator(sel)
            if els.count() > 0:
                els.first.click(force=True, timeout=3000)
                log(f"  clicked '{sel}'")
                _wait(1, log)
                break
        except Exception:
            pass

    # Read from clipboard
    try:
        api_key = page.evaluate(
            "async () => { try { return await navigator.clipboard.readText(); } catch(e) { return ''; } }"
        )
        log(f"  clipboard value: {api_key[:20]!r}..." if api_key else "  clipboard empty")
        if api_key and re.match(r"^[a-f0-9]{32,40}$", api_key.strip()):
            log(f"  got key from clipboard: {api_key[:10]}...")
            return api_key.strip()
    except Exception as e:
        log(f"  clipboard read error: {e}")

    # JS-based DOM inspection: find hex keys in all element values/text/attributes
    try:
        key_from_js = page.evaluate("""() => {
            const hexRe = /^[a-f0-9]{32,40}$/;
            // Check all inputs
            for (const inp of document.querySelectorAll('input')) {
                const v = (inp.value || '').trim();
                if (hexRe.test(v)) return v;
                const dv = inp.getAttribute('data-value') || '';
                if (hexRe.test(dv.trim())) return dv.trim();
            }
            // Check all elements with data-* attributes
            for (const el of document.querySelectorAll('[data-value], [data-key], [data-api-key]')) {
                for (const attr of el.attributes) {
                    const v = (attr.value || '').trim();
                    if (hexRe.test(v)) return v;
                }
            }
            // Check code/pre/span elements
            for (const el of document.querySelectorAll('code, pre, span, div, p')) {
                const t = (el.textContent || '').trim();
                if (hexRe.test(t)) return t;
            }
            // Last resort: search entire body for hex pattern
            const m = document.body.innerHTML.match(/[a-f0-9]{37}/);
            if (m) return m[0];
            return '';
        }""")
        if key_from_js:
            log(f"  got key from JS DOM scan: {key_from_js[:10]}...")
            return key_from_js.strip()
        else:
            log("  JS DOM scan found nothing")
    except Exception as e:
        log(f"  JS DOM scan error: {e}")

    # Fallback: standard DOM extraction
    return _extract_global_api_key(page, log)


def _extract_global_api_key_from_session(page, password, mail_client, mailbox, log) -> str:
    """Extract Global API Key from an already-logged-in, email-verified session.

    PREREQUISITE: email_verified must be True before calling this function.
    The caller (extract_global_api_key_via_browser) handles email verification.

    CF flow (two-dialog):
    1. Click first 'View' button (Global API Key section)
    2. Dialog 1 'Verify Your Identity': click 'Send Verification Code',
       poll mail.tm for 7-digit code, fill + click 'Verify code'
    3. Dialog 2 (if Turnstile present): wait for Turnstile to auto-solve,
       re-fill code + submit. Key revealed once Turnstile passes.
    4. Read key from clipboard ('Protect this key like a password! / Copied')
       or from DOM as fallback.
    """
    import httpx as _httpx

    # Grant clipboard-read permission so navigator.clipboard.readText() works
    try:
        page.context.grant_permissions(["clipboard-read", "clipboard-write"])
    except Exception:
        pass

    # Navigate to API tokens page
    log("navigating to /profile/api-tokens...")
    try:
        page.goto("https://dash.cloudflare.com/profile/api-tokens", timeout=20000,
                  wait_until="domcontentloaded")
    except Exception as e:
        log(f"api-tokens nav warn: {e}")
    _wait(4, log)
    _log_page(page, log, "api-tokens: ", max_chars=400)

    # Mark existing mail.tm messages as seen (to detect only new codes)
    seen_ids: set[str] = set()
    try:
        token_hdr = {"Authorization": f"Bearer {mail_client._token}"}
        resp = _httpx.get("https://api.mail.tm/messages", headers=token_hdr, timeout=15)
        for m in resp.json().get("hydra:member", []):
            seen_ids.add(m["id"])
        log(f"  pre-marked {len(seen_ids)} existing mail messages")
    except Exception as e:
        log(f"  mail pre-mark error: {e}")

    def _poll_new_code(timeout=120) -> str:
        """Poll only new mail.tm messages for a 6-8 digit code."""
        import re as _re
        code_re = _re.compile(r"\b(\d{6,8})\b")
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                resp = _httpx.get("https://api.mail.tm/messages", headers=token_hdr, timeout=15)
                for msg in resp.json().get("hydra:member", []):
                    mid = msg["id"]
                    if mid in seen_ids:
                        continue
                    seen_ids.add(mid)
                    log(f"  new msg: {msg.get('subject')}")
                    m = code_re.search(msg.get("subject", ""))
                    if m:
                        return m.group(1)
                    full = _httpx.get(f"https://api.mail.tm/messages/{mid}", headers=token_hdr, timeout=15)
                    body = full.json()
                    text = body.get("text", "") + " " + body.get("intro", "")
                    m = code_re.search(text)
                    if m:
                        return m.group(1)
            except Exception as e:
                log(f"  poll error: {e}")
            time.sleep(3)
        raise RuntimeError(f"No code received within {timeout}s")

    # Click the FIRST "View" button (Global API Key, not Origin CA Key)
    log("clicking first 'View' button (Global API Key)...")
    view_btns = page.locator('button:has-text("View")').all()
    log(f"  found {len(view_btns)} View buttons")
    if not view_btns:
        raise RuntimeError("No 'View' buttons found on api-tokens page")
    view_btns[0].scroll_into_view_if_needed()
    view_btns[0].click()
    _wait(2, log)

    body_text = page.inner_text("body", timeout=3000) or ""
    log(f"  after click: {body_text[-300:]}")

    # === Dialog 1: Verify Your Identity ===
    code = ""
    if any(kw in body_text.lower() for kw in [
        "verify your identity", "send verification code",
        "verification code", "send code",
    ]):
        log("=== Dialog 1: Verify Your Identity ===")
        _force_click_dialog_btn(page, [
            'button:has-text("Send Verification Code")',
            'button:has-text("Send")',
            'button:has-text("Send code")',
        ], log)
        _wait(1, log)

        log("polling mail.tm for fresh verification code...")
        code = _poll_new_code(timeout=120)
        log(f"  code: {code}")

        _fill_first(page, [
            'input[name*="code" i]',
            'input[placeholder*="code" i]',
            'input[type="text"]',
            'input[type="number"]',
        ], code, log)
        _wait(0.5, log)

        # Use force=True to bypass CF's focusFallback overlay
        _force_click_dialog_btn(page, [
            'button:has-text("Verify code")',
            'button:has-text("Confirm")',
            'button:has-text("Verify")',
            'button[type="submit"]',
        ], log)
        _wait(3, log, "dialog 1 submit")

        body_text = page.inner_text("body", timeout=3000) or ""
        log(f"  after dialog 1: {body_text[-300:]}")

    # === Check if key was revealed immediately (no Dialog 2) ===
    api_key = _try_read_key(page, log)
    if api_key:
        return api_key

    # === Dialog 2: "Your API Key" (Turnstile + code re-entry) ===
    # CF shows a second dialog with Turnstile CAPTCHA that must be solved before
    # revealing the key. In Patchright (undetected browser) Turnstile auto-solves.
    # Poll until "protect this key" appears, re-submitting code as needed.
    if "your api key" in body_text.lower():
        log("=== Dialog 2: Your API Key (Turnstile) ===")
        deadline = time.time() + 120  # up to 2 minutes for Turnstile
        last_submit = 0.0
        while time.time() < deadline:
            current_text = page.inner_text("body", timeout=3000) or ""

            # Success: key has been revealed
            if "protect this key" in current_text.lower() or "copied" in current_text.lower():
                log("  key revealed!")
                api_key = _try_read_key(page, log)
                if api_key:
                    return api_key
                break

            # DOM extraction fallback (key visible without clipboard)
            api_key = _extract_global_api_key(page, log)
            if api_key:
                return api_key

            # Re-submit: fill code + click Verify when "invalid captcha" or periodically
            needs_submit = (
                "invalid captcha" in current_text.lower()
                or (time.time() - last_submit > 15)
            )
            if needs_submit and code:
                log(f"  re-filling code {code} and submitting...")
                try:
                    _fill_first(page, [
                        'input[name="code"]',
                        'input[name*="code" i]',
                        'input[placeholder*="code" i]',
                    ], code, log)
                    _wait(0.5, log)
                    _force_click_dialog_btn(page, [
                        'button:has-text("Verify code")',
                        'button:has-text("Verify")',
                        'button[type="submit"]',
                    ], log)
                    last_submit = time.time()
                    _wait(3, log)
                except Exception as e:
                    log(f"  re-submit error: {e}")
                    time.sleep(3)
            else:
                log(f"  waiting for Turnstile… ({int(deadline - time.time())}s left)")
                time.sleep(3)

        # One last try after loop
        api_key = _try_read_key(page, log)
        if api_key:
            return api_key
        api_key = _extract_global_api_key(page, log)
        if api_key:
            return api_key

    # === Password modal fallback ===
    if page.locator('input[type="password"]').count() > 0:
        log("  password modal — entering password")
        _fill_first(page, ['input[type="password"]'], password, log)
        _wait(0.5, log)
        _click_first(page, ['button:has-text("View")', 'button[type="submit"]'], log)
        _wait(3, log)
        api_key = _try_read_key(page, log)
        if api_key:
            return api_key
        api_key = _extract_global_api_key(page, log)
        if api_key:
            return api_key

    _log_page(page, log, "key-fail: ", max_chars=2000)
    raise RuntimeError("Failed to extract Global API Key from page")


def _extract_global_api_key(page, log) -> str:
    """Extract the Global API Key value from the page after clicking View."""
    # Strategy 1: input/textarea with the key value (shown after View)
    for sel in [
        'textarea',
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
