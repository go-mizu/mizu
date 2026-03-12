"""Patchright-based MotherDuck account registration.

Flow:
  1. Open app.motherduck.com → auto-redirects to Auth0 (auth.motherduck.com)
  2. On Auth0 login page: find "Sign up" link → click to go to signup page
  3. On signup page: enter email + password → submit
  4. If email verification required: poll mail.tm for verification link
  5. Click through any onboarding prompts (only on app.motherduck.com)
  6. Navigate to Settings > Tokens → generate + extract token

Note: MotherDuck uses Auth0 password-based auth. New accounts are created
via the /u/signup page, NOT the /u/login page.
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
        args += [
            "--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage",
            "--disable-gpu",
        ]
    return args


def _ensure_display() -> None:
    """On Linux without DISPLAY, start Xvfb. Needed even for headless Chromium on some VPS."""
    if platform.system() != "Linux" or os.environ.get("DISPLAY"):
        return
    import shutil
    import subprocess
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


def _on_auth0(page) -> bool:
    return "auth.motherduck.com" in page.url


def _on_app(page) -> bool:
    url = page.url
    return "app.motherduck.com" in url and "auth." not in url


def _click_first(page, selectors: list[str], log=None) -> str | None:
    """Try each selector, click the first match. Returns the matched selector or None."""
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


def _clear_and_fill(page, selectors: list[str], text: str, log=None) -> str | None:
    """Try each selector, clear then fill the first match."""
    for sel in selectors:
        try:
            inp = page.locator(sel)
            if inp.count() > 0:
                el = inp.first
                el.click()
                time.sleep(0.2)
                el.fill("")  # clear existing text
                time.sleep(0.1)
                el.type(text, delay=55)
                time.sleep(0.3)
                if log:
                    log(f"filled (clear+type) via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    """Try each selector, fill the first match. Returns the matched selector or None."""
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


def _log_page_state(page, log, label: str = "", max_chars: int = 500) -> str:
    """Log current URL and page text for diagnostics."""
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:400]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive MotherDuck signup, return the API token string."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="md_reg_")

    with sync_playwright() as p:
        # Use system Chrome if available, otherwise fall back to bundled Chromium
        import shutil
        channel = "chrome" if shutil.which("google-chrome") or shutil.which("google-chrome-stable") else None

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
            # ---- Step 1: Open app — auto-redirects to Auth0 ----
            log("opening app.motherduck.com...")
            try:
                page.goto("https://app.motherduck.com", timeout=30000)
                # Wait for the Auth0 redirect to complete (JS-based redirect)
                page.wait_for_url("**/auth.motherduck.com/**", timeout=15000)
            except Exception as e:
                log(f"landing/redirect warn: {e}")
                # If SPA didn't redirect (e.g., --single-process mode), try
                # extracting the Auth0 redirect URL from page source or use authorize endpoint
                if "auth.motherduck.com" not in page.url:
                    log("SPA redirect failed, trying to extract Auth0 redirect...")
                    # Check if the page source has a redirect URL
                    auth0_url = ""
                    try:
                        html = page.evaluate("document.documentElement.innerHTML")
                        log(f"  page HTML length: {len(html)}")
                        log(f"  page HTML preview: {html[:500]!r}")
                        import re as _re
                        import html as _html_mod
                        m = _re.search(r'(https://auth\.motherduck\.com/[^"\'>\s]+)', html)
                        if m:
                            auth0_url = _html_mod.unescape(m.group(1))
                            # Change screen_hint from login to signup
                            auth0_url = auth0_url.replace("screen_hint=login", "screen_hint=signup")
                            # Remove prompt=none (causes silent auth, breaks signup)
                            auth0_url = _re.sub(r'[&?]prompt=none', '', auth0_url)
                            log(f"  found Auth0 URL in HTML: {auth0_url[:120]}")
                    except Exception as ex:
                        log(f"  HTML extraction error: {ex}")

                    if not auth0_url:
                        # Use the Auth0 authorize endpoint directly
                        auth0_url = (
                            "https://auth.motherduck.com/authorize"
                            "?response_type=code"
                            "&client_id=bza3KWQpxRAFlTlRFXUo29AOg9xD7zcp"
                            "&redirect_uri=https%3A%2F%2Fapp.motherduck.com"
                            "&scope=openid+profile+email"
                            "&screen_hint=signup"
                        )
                        log(f"  using authorize endpoint: {auth0_url[:80]}")

                    try:
                        page.goto(auth0_url, timeout=30000)
                        page.wait_for_load_state("networkidle", timeout=15000)
                    except Exception as e2:
                        log(f"Auth0 redirect warn: {e2}")
            _wait(2, log)
            log(f"landed on: {page.url}")

            # ---- Step 2: Navigate to signup page ----
            # Auth0 login page has a "Sign up" link. Click it to reach /u/signup.
            if _on_auth0(page) and "signup" not in page.url:
                log("on Auth0 login — looking for Sign up link...")
                signup_clicked = _click_first(page, [
                    'a:has-text("Sign up")',
                    'a[href*="signup"]',
                    'button:has-text("Sign up")',
                    'a:has-text("Create account")',
                    'a:has-text("Don\'t have an account")',
                ], log)
                if signup_clicked:
                    _wait(3, log, "navigating to signup")
                else:
                    log("no signup link found — trying direct URL")
                    try:
                        # Construct signup URL from current login URL
                        current = page.url
                        signup_url = current.replace("/u/login/identifier", "/u/signup")
                        if signup_url == current:
                            signup_url = current.replace("/u/login/password", "/u/signup")
                        page.goto(signup_url, timeout=15000)
                        _wait(2, log)
                    except Exception as e:
                        log(f"direct signup URL warn: {e}")
                log(f"url: {page.url}")

            # ---- Step 3: Fill signup form ----
            log(f"filling signup form with {mailbox.address}")

            # Fill email
            email_sel = _fill_first(page, [
                'input[name="email"]',
                'input[name="username"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
                'input[placeholder*="address" i]',
            ], mailbox.address, log)
            if not email_sel:
                log("WARNING: no email input found!")
                _log_page_state(page, log, "  ")

            # Password might be on the same page or a separate step
            _wait(0.5, log)
            pwd_sel = _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)

            # Submit (email only or email+password)
            _wait(0.5, log)
            submit_sel = _click_first(page, [
                'button[name="action"]',
                'button[value="default"]',
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Continue")',
                'button:has-text("Create account")',
            ], log)
            if submit_sel:
                _wait(4, log, "waiting for signup response")
            log(f"url after first submit: {page.url}")

            # ---- Step 3b: Auth0 multi-step signup — password page ----
            # Auth0 may advance to /u/signup/password as a second step
            if "signup/password" in page.url or (
                _on_auth0(page) and not pwd_sel
            ):
                log("on Auth0 password step — filling password...")
                pwd_sel2 = _fill_first(page, [
                    'input[name="password"]',
                    'input[type="password"]',
                ], password, log)
                if pwd_sel2:
                    _wait(0.5, log)
                    _click_first(page, [
                        'button[name="action"]',
                        'button[value="default"]',
                        'button[type="submit"]',
                        'button:has-text("Sign up")',
                        'button:has-text("Continue")',
                    ], log)
                    _wait(5, log, "waiting for account creation")
                else:
                    log("WARNING: no password input on password page!")
                    _log_page_state(page, log, "  ")
                log(f"url after password submit: {page.url}")

            # ---- Step 4: Handle post-signup ----
            body_text = _log_page_state(page, log, "post-signup: ")

            # Check for errors
            error_keywords = ["error", "wrong", "invalid", "already exists", "try again"]
            if any(kw in body_text.lower() for kw in error_keywords):
                log(f"WARNING: possible error on page")

            # Check for email verification requirement
            verify_keywords = ["verify", "check your email", "confirmation",
                             "magic link", "link has been sent", "we sent"]
            if any(kw in body_text.lower() for kw in verify_keywords):
                log("email verification required — polling mail.tm...")
                magic_link = mail_client.poll_for_magic_link(mailbox, timeout=120)
                log(f"got verification link: {magic_link[:60]}...")
                try:
                    page.goto(magic_link, timeout=30000)
                except Exception as e:
                    log(f"verification link nav warn: {e}")
                _wait(4, log, "post-verification load")
                log(f"url after verification: {page.url}")

                # After email verification, clear stale Auth0 session then login
                log("email verified — clearing session and logging in...")

                # Click "Log out" if visible (clears Auth0 session)
                _click_first(page, [
                    'a:has-text("Log out")',
                    'button:has-text("Log out")',
                    'a:has-text("Logout")',
                ], log)
                _wait(2, log, "post-logout")

                # Clear Auth0 cookies to ensure fresh login
                ctx.clear_cookies()
                _wait(1, log)

                # Navigate to app → Auth0 redirect for login
                try:
                    page.goto("https://app.motherduck.com", timeout=60000)
                    page.wait_for_url("**/auth.motherduck.com/**", timeout=30000)
                except Exception as e:
                    log(f"login redirect warn: {e}")
                    # If SPA didn't redirect, extract Auth0 URL from HTML (same as step 1)
                    if "auth.motherduck.com" not in page.url:
                        log("SPA redirect failed for login, extracting Auth0 URL...")
                        import re as _re
                        import html as _html_mod
                        try:
                            html = page.evaluate("document.documentElement.innerHTML")
                            m = _re.search(r'(https://auth\.motherduck\.com/[^"\'>\s]+)', html)
                            if m:
                                auth0_url = _html_mod.unescape(m.group(1))
                                log(f"  using Auth0 URL: {auth0_url[:100]}")
                                page.goto(auth0_url, timeout=30000)
                                page.wait_for_load_state("networkidle", timeout=15000)
                        except Exception as e2:
                            log(f"  Auth0 URL extraction failed: {e2}")
                _wait(2, log)
                log(f"login page: {page.url}")

                # Only fill login if we're on Auth0
                if _on_auth0(page):
                    # Fill email
                    _fill_first(page, [
                        'input[name="username"]',
                        'input[name="email"]',
                        'input[type="email"]',
                    ], mailbox.address, log)

                    # Submit email
                    _click_first(page, [
                        'button[name="action"]',
                        'button[value="default"]',
                        'button[type="submit"]',
                        'button:has-text("Continue")',
                    ], log)
                    _wait(3, log, "waiting for password step")

                    # Fill password
                    _fill_first(page, [
                        'input[name="password"]',
                        'input[type="password"]',
                    ], password, log)

                    # Submit password
                    _click_first(page, [
                        'button[name="action"]',
                        'button[value="default"]',
                        'button[type="submit"]',
                        'button:has-text("Continue")',
                    ], log)
                    _wait(5, log, "waiting for login")
                    log(f"url after login: {page.url}")
                    _log_page_state(page, log, "login-result: ")
                else:
                    log(f"not on Auth0 after redirect, url={page.url}")
                    _log_page_state(page, log, "login-redirect-fail: ")

            # ---- Step 5: Click through onboarding (ONLY on app.motherduck.com) ----
            if _on_app(page):
                log("on MotherDuck app — clicking through onboarding...")
                _skip_onboarding(page, log, max_attempts=15)
            elif _on_auth0(page):
                log(f"still on Auth0 after signup: {page.url}")
                # Try navigating directly to the app
                try:
                    page.goto("https://app.motherduck.com", timeout=20000)
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception as e:
                    log(f"app redirect warn: {e}")
                _wait(3, log)
                log(f"url after app redirect: {page.url}")
                if _on_app(page):
                    _skip_onboarding(page, log, max_attempts=10)

            log(f"url before token extraction: {page.url}")

            # ---- Step 6: Go to Settings > Tokens ----
            log("navigating to token settings...")
            try:
                page.goto("https://app.motherduck.com/settings/tokens", timeout=20000)
                page.wait_for_load_state("networkidle", timeout=10000)
            except Exception as e:
                log(f"settings nav warn: {e}")
            _wait(3, log, "settings load")
            log(f"url: {page.url}")

            # If redirected back to Auth0, we're not logged in
            if _on_auth0(page):
                _log_page_state(page, log, "NOT LOGGED IN: ")
                raise RuntimeError(
                    f"Not logged in — Auth0 redirect after signup. "
                    f"URL: {page.url}"
                )

            # ---- Step 7: Generate token ----
            log("generating token...")

            # Wait for the page to fully render (SPA may need 30s+ to initialize)
            for wait_attempt in range(12):
                body_text = _log_page_state(page, log, "token-page: ", max_chars=800)
                if body_text.strip():
                    break
                _wait(5, log, f"waiting for page render ({wait_attempt+1}/12)")

            create_sel = _click_first(page, [
                'button:has-text("Generate token")',
                'button:has-text("Create token")',
                'button:has-text("New token")',
                'button:has-text("Generate")',
                'button:has-text("Create")',
            ], log)
            _wait(2, log, "token dialog")

            # Handle token creation dialog/modal (may ask for token name)
            _log_page_state(page, log, "after-create: ", max_chars=800)
            name_input = page.locator('input[placeholder*="token" i], input[placeholder*="name" i], input[name*="token" i], input[name*="name" i]')
            if name_input.count() > 0:
                log("  filling token name...")
                name_input.first.fill("api-token")
                time.sleep(0.5)
                _click_first(page, [
                    'button:has-text("Create")',
                    'button:has-text("Generate")',
                    'button:has-text("Save")',
                    'button:has-text("Confirm")',
                    'button[type="submit"]',
                ], log)
                _wait(3, log, "token creation")
            else:
                _wait(2, log, "token generation")

            # ---- Step 8: Extract token ----
            log("extracting token...")
            token = _extract_token(page, ctx, log)

            # If no token found, retry a few times — SPA may need time to store token
            if not token:
                for retry in range(5):
                    _wait(5, log, f"token retry {retry+1}/5")
                    # Reload to trigger SPA re-initialization
                    try:
                        page.reload(timeout=15000)
                        page.wait_for_load_state("networkidle", timeout=10000)
                    except Exception:
                        pass
                    _wait(3, log)
                    token = _extract_token(page, ctx, log)
                    if token:
                        break

            if not token:
                _log_page_state(page, log, "token-fail: ")
                raise RuntimeError("Failed to extract MotherDuck token from settings page")

            log(f"token extracted (len={len(token)})")
            return token

        finally:
            ctx.close()


def _skip_onboarding(page, log, display_name: str = "", max_attempts: int = 10) -> None:
    """Click through MotherDuck onboarding prompts.
    Only runs when on app.motherduck.com (NOT auth.motherduck.com).
    """
    from faker import Faker
    fake = Faker()

    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url

        # Only onboard on the app domain
        if "auth.motherduck.com" in url:
            log(f"  still on Auth0, stopping onboarding")
            return

        # Consider done if on main app page
        if any(x in url for x in ["/editor", "/home", "/settings", "/query"]):
            log(f"onboarding done at attempt {attempt}")
            return

        # Fill user-information form if present
        if "user-information" in url:
            first = display_name.split()[0] if display_name else fake.first_name()
            last = display_name.split()[-1] if display_name and len(display_name.split()) > 1 else fake.last_name()

            # Only fill the form once (check if already filled)
            first_input = page.locator('input[name="firstName"]')
            if first_input.count() > 0 and not first_input.input_value():
                log(f"  filling user info: {first} {last}")
                # Clear and fill first/last name fields
                _clear_and_fill(page, [
                    'input[name="firstName"]',
                    'input[name*="first" i]',
                ], first, log)
                _clear_and_fill(page, [
                    'input[name="lastName"]',
                    'input[name*="last" i]',
                ], last, log)
                time.sleep(0.5)

            # Select region if "Pick a region" is present
            region_trigger = page.locator('text="Pick a region"')
            if region_trigger.count() > 0:
                log(f"  selecting region...")
                region_trigger.first.click()
                time.sleep(1)
                # Try to select US East (most common)
                region_sel = _click_first(page, [
                    'text="US East (Ohio)"',
                    'text="US East"',
                    '[data-value*="us-east"]',
                    'li:has-text("US East")',
                    'div[role="option"]:has-text("US East")',
                    'div[role="option"]:first-child',
                    'li:first-child',
                ], log)
                if not region_sel:
                    # Fallback: just click the first option in any dropdown
                    options = page.locator('[role="option"], [role="listbox"] > *, select option')
                    if options.count() > 0:
                        log(f"  clicking first dropdown option")
                        options.first.click()
                time.sleep(0.5)

            # Check for and click any unchecked checkboxes (TOS, consent, etc.)
            checkboxes = page.locator('input[type="checkbox"]:visible')
            for i in range(checkboxes.count()):
                cb = checkboxes.nth(i)
                if not cb.is_checked():
                    log(f"  checking checkbox {i}")
                    cb.check()
                    time.sleep(0.3)

        # Handle survey page — select options for any visible question
        if "survey" in url:
            log(f"  on survey page, selecting options...")
            # Click first unselected radio button or option for each question
            for option_text in ["Software engineer", "Data engineer", "Other",
                                "Personal project", "Evaluation", "Company"]:
                opt = page.locator(f'text="{option_text}"')
                if opt.count() > 0:
                    try:
                        opt.first.click()
                        log(f"  selected: {option_text}")
                        time.sleep(0.3)
                        break
                    except Exception:
                        pass
            # Also try clicking radio buttons directly
            radios = page.locator('input[type="radio"]:visible')
            if radios.count() > 0 and not any(
                radios.nth(i).is_checked() for i in range(min(radios.count(), 10))
            ):
                radios.first.click()
                time.sleep(0.3)

        clicked = False
        for sel in [
            'button:has-text("Skip")',
            'button:has-text("Next")',
            'button:has-text("Get started")',
            'button:has-text("Done")',
            'button:has-text("Continue"):not([disabled])',
            '[role="button"]:has-text("Skip")',
        ]:
            btn = page.locator(sel)
            if btn.count() > 0:
                try:
                    log(f"  onboarding click: {sel}")
                    btn.first.click(timeout=5000)
                    clicked = True
                    break
                except Exception as e:
                    log(f"  click failed: {e}")
                    continue

        if not clicked:
            log(f"  no onboarding button at attempt {attempt}, url={url}")
            break


def _extract_token(page, ctx, log) -> str:
    """Try multiple strategies to extract the MotherDuck API token."""
    import re

    # Strategy 1: look for token displayed in a <code>, input, or textarea element
    for sel in [
        'code',
        'input[readonly]',
        'textarea[readonly]',
        '[data-testid*="token"]',
        'pre',
        '.token',
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

    # Strategy 3: localStorage — check all keys for token-like values
    try:
        all_keys = page.evaluate("() => Object.keys(localStorage)")
        log(f"localStorage keys: {all_keys}")
        # First check known keys
        for key in ["motherduck_token", "token", "md_token", "access_token"]:
            token = page.evaluate(
                f"() => localStorage.getItem('{key}')"
            )
            if token and len(token) > 20:
                log(f"token via localStorage[{key!r}]: {token[:20]}...")
                return token
        # Then scan all keys for JWT-like values
        for key in all_keys:
            val = page.evaluate(f"() => localStorage.getItem('{key}')")
            if val and len(val) > 30:
                # Check for JWT pattern
                m = re.search(r"eyJ[A-Za-z0-9\-_]{30,}\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+", val)
                if m:
                    log(f"JWT token in localStorage[{key!r}]: {m.group(0)[:30]}...")
                    return m.group(0)
                # Check for MotherDuck token pattern
                if val.startswith("motherduck_token_") or val.startswith("v0_"):
                    log(f"token in localStorage[{key!r}]: {val[:30]}...")
                    return val
    except Exception as ex:
        log(f"localStorage scan error: {ex}")

    # Strategy 4: cookies
    cookies = {c["name"]: c["value"] for c in ctx.cookies()}
    log(f"cookie names: {list(cookies.keys())}")
    for key in ["motherduck_token", "token", "auth_token", "md_token"]:
        if key in cookies and len(cookies[key]) > 20:
            log(f"token via cookie {key!r}")
            return cookies[key]
    # Scan all cookies for token-like values
    for key, val in cookies.items():
        if len(val) > 30:
            m = re.search(r"eyJ[A-Za-z0-9\-_]{30,}\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+", val)
            if m:
                log(f"JWT token in cookie {key!r}")
                return m.group(0)

    # Strategy 5: check page HTML for token patterns (SPA might not have rendered text but HTML might have data)
    try:
        html = page.evaluate("document.documentElement.innerHTML")
        if html:
            for pat in [
                r"eyJ[A-Za-z0-9\-_]{30,}\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+",
                r"motherduck_token_[A-Za-z0-9_\-]{20,}",
                r'"token"\s*:\s*"([^"]{20,})"',
            ]:
                m = re.search(pat, html)
                if m:
                    token = m.group(1) if m.lastindex else m.group(0)
                    log(f"token via HTML regex: {token[:30]}...")
                    return token
    except Exception:
        pass

    return ""
