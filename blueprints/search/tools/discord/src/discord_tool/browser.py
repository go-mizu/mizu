"""Patchright-based Discord account registration and token extraction.

Flow (register):
  1. Create mail.tm mailbox for email
  2. Open discord.com/register
  3. Fill username, email, password, date of birth
  4. Submit — Discord sends verification email
  5. Poll mail.tm for verification link, click it
  6. After verification, intercept Authorization header from API calls
     OR extract token from localStorage via JS evaluation

Flow (login):
  1. Open discord.com/login
  2. Fill email + password
  3. After login, intercept Authorization header from API calls

Token extraction:
  Discord stores the user token in localStorage (encrypted by the app).
  The reliable method is to intercept outgoing XHR/fetch requests and
  read the Authorization header, which contains the raw user token.
"""
from __future__ import annotations

import os
import platform
import re
import time
import tempfile

from .email import MailTmClient, Mailbox

# JS snippet to extract Discord token from webpack module registry.
# Works on discord.com in a browser context.
_TOKEN_JS = """
(() => {
    try {
        // Method 1: webpack chunk
        const chunk = window.webpackChunkdiscord_app;
        if (chunk) {
            const modules = [];
            chunk.push([[''], {}, (e) => { for (let c in e.c) modules.push(e.c[c]); }]);
            const m = modules.find(m => m?.exports?.default?.getToken !== undefined);
            if (m) return m.exports.default.getToken();
        }
    } catch(e) {}
    try {
        // Method 2: localStorage scan
        for (let i = 0; i < localStorage.length; i++) {
            const k = localStorage.key(i);
            const v = localStorage.getItem(k);
            if (v && v.length > 50 && /^[A-Za-z0-9._-]{50,}$/.test(v)) {
                return v;
            }
        }
    } catch(e) {}
    return null;
})()
"""


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


def _extract_token_from_page(page, log) -> str:
    """Try JS evaluation first, then page text scan."""
    try:
        token = page.evaluate(_TOKEN_JS)
        if token and len(token) > 50:
            log(f"token via JS webpack: {token[:20]}...")
            return token
    except Exception as e:
        log(f"JS eval warn: {e}")

    # Fallback: scan page source for token pattern (MFA token format or standard)
    try:
        html = page.evaluate("document.documentElement.innerHTML")
        # Discord tokens: base64url.base64url.base64url (3 segments) or shorter MFA tokens
        m = re.search(r'"token"\s*:\s*"([A-Za-z0-9._-]{50,})"', html)
        if m:
            log(f"token via HTML regex: {m.group(1)[:20]}...")
            return m.group(1)
    except Exception:
        pass

    return ""


def extract_token_manual(
    headless: bool = False,
    verbose: bool = False,
    wait: int = 300,
    email: str = "",
    password: str = "",
) -> str:
    """Open Discord login in a visible browser, optionally pre-fill credentials,
    wait for login, then capture the token from intercepted Authorization headers.
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log("opening discord.com/login")
    user_data = tempfile.mkdtemp(prefix="dc_extract_")
    captured_tokens: list[str] = []

    with sync_playwright() as p:
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

        def on_request(request):
            auth = request.headers.get("authorization", "")
            if auth and len(auth) > 20 and not auth.startswith("Bearer "):
                if auth not in captured_tokens:
                    captured_tokens.append(auth)
                    log(f"token intercepted: {auth[:20]}...")

        page.on("request", on_request)

        try:
            page.goto("https://discord.com/login", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)

            # Pre-fill credentials if provided
            if email:
                log(f"filling email: {email}")
                _fill_first(page, [
                    'input[name="email"]',
                    'input[type="email"]',
                    'input[placeholder*="email" i]',
                    'input[placeholder*="Phone" i]',
                ], email, log)
                _wait(0.5, log)

            if password:
                log("filling password")
                _fill_first(page, [
                    'input[name="password"]',
                    'input[type="password"]',
                ], password, log)
                _wait(0.5, log)

            if email and password:
                log("submitting login form")
                _click_first(page, [
                    'button[type="submit"]',
                    'button:has-text("Log In")',
                    'button:has-text("Login")',
                ], log)
                _wait(3, log, "waiting for login response")
                log(f"url after submit: {page.url}")

            if not email:
                log("waiting for manual login (fill email/password in the browser)...")

            # Wait for token capture
            deadline = time.time() + wait
            while time.time() < deadline:
                if captured_tokens:
                    break
                try:
                    cur_url = page.url
                    if "discord.com/channels" in cur_url or "@me" in cur_url:
                        token = page.evaluate(_TOKEN_JS)
                        if token and len(token) > 50:
                            log(f"token via JS: {token[:20]}...")
                            captured_tokens.append(token)
                            break
                except Exception:
                    pass
                time.sleep(2)

            if not captured_tokens:
                raise RuntimeError(f"No token captured within {wait}s")

            return captured_tokens[0]

        finally:
            page.remove_listener("request", on_request)
            ctx.close()


def _intercept_token(page, log, timeout: int = 30) -> str:
    """Route interception: capture Authorization header from Discord API calls."""
    captured: list[str] = []

    def on_request(request):
        auth = request.headers.get("authorization", "")
        if auth and len(auth) > 20 and not auth.startswith("Bearer "):
            captured.append(auth)

    page.on("request", on_request)

    # Trigger an API call by navigating to channels
    try:
        page.goto("https://discord.com/channels/@me", timeout=20000)
    except Exception:
        pass

    deadline = time.time() + timeout
    while time.time() < deadline and not captured:
        time.sleep(1)

    page.remove_listener("request", on_request)
    if captured:
        token = captured[0]
        log(f"token via request interception: {token[:20]}...")
        return token
    return ""


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    username: str,
    password: str,
    birth_year: int,
    birth_month: int,
    birth_day: int,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive Discord registration, return the user token."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address} / {username}")
    user_data = tempfile.mkdtemp(prefix="dc_reg_")

    with sync_playwright() as p:
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

        try:
            # Step 1: Open registration page
            log("opening discord.com/register...")
            page.goto("https://discord.com/register", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)

            # Step 2: Fill registration form
            log("filling registration form...")

            # Email — try current domain; if Discord marks it invalid, retry with others
            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
            ], mailbox.address, log)
            _wait(1, log)

            # Check for invalid email error; if found, cycle through 1secmail domains
            try:
                _wait(1, log)
                err_text = page.inner_text("body")[:400].lower()
                if "invalid email" in err_text or "valid email" in err_text or "not valid" in err_text:
                    log(f"domain rejected — trying maildrop.cc then 1secmail domains")
                    from .email import create_maildrop_mailbox, create_1sec_mailbox, _ONESEC_DOMAINS
                    local = mailbox.address.split("@")[0]
                    candidate_domains = [("maildrop.cc", "maildrop")] + \
                                        [(d, "1secmail") for d in _ONESEC_DOMAINS]
                    accepted = False
                    for domain, provider in candidate_domains:
                        new_addr = f"{local}@{domain}"
                        log(f"  trying: {new_addr}")
                        inp = page.locator('input[name="email"], input[type="email"]').first
                        inp.click()
                        inp.fill("")
                        time.sleep(0.2)
                        inp.type(new_addr, delay=55)
                        _wait(1.5, log)
                        err_text = page.inner_text("body")[:400].lower()
                        if "invalid email" not in err_text and "valid email" not in err_text and "not valid" not in err_text:
                            log(f"  accepted: {new_addr}")
                            if provider == "maildrop":
                                mailbox = create_maildrop_mailbox(local, mailbox.password)
                            else:
                                mailbox = create_1sec_mailbox(local, mailbox.password, domain)
                            accepted = True
                            break
                    if not accepted:
                        log("WARNING: all domains rejected — continuing anyway")
            except Exception as e:
                log(f"email validation check warn: {e}")

            # Display name (optional, may not appear)
            _fill_first(page, [
                'input[name="global_name"]',
                'input[placeholder*="display" i]',
            ], username, log)
            _wait(0.3, log)

            # Username
            _fill_first(page, [
                'input[name="username"]',
                'input[placeholder*="username" i]',
            ], username, log)
            _wait(0.3, log)

            # Password
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)
            _wait(0.3, log)

            # Date of birth — Discord uses 3 native <select> elements with labels Month/Day/Year.
            # Month options have text values like "January", "February" etc.
            _MONTHS = ["January","February","March","April","May","June",
                       "July","August","September","October","November","December"]
            log(f"filling date of birth: {_MONTHS[birth_month-1]} {birth_day}, {birth_year}")
            try:
                # Find selects by their placeholder/aria-label or by position
                selects = page.locator('select')
                count = selects.count()
                log(f"  found {count} select elements")
                if count >= 3:
                    # Discord order: Month (0), Day (1), Year (2)
                    selects.nth(0).select_option(label=_MONTHS[birth_month - 1])
                    _wait(0.3, log)
                    selects.nth(1).select_option(str(birth_day))
                    _wait(0.3, log)
                    selects.nth(2).select_option(str(birth_year))
                    _wait(0.3, log)
                    log("  DOB filled")
                elif count > 0:
                    # Fewer selects — try to identify by visible label text
                    for i in range(count):
                        sel = selects.nth(i)
                        # Check options to determine type
                        opts = sel.locator('option').all_inner_texts()
                        if any(m in opts for m in _MONTHS):
                            sel.select_option(label=_MONTHS[birth_month - 1])
                            log(f"  month select at nth({i})")
                        elif any(str(d) == o for d in range(1, 32) for o in opts[:5]):
                            sel.select_option(str(birth_day))
                            log(f"  day select at nth({i})")
                        elif any(str(y) in opts for y in range(birth_year - 2, birth_year + 3)):
                            sel.select_option(str(birth_year))
                            log(f"  year select at nth({i})")
                        _wait(0.2, log)
                else:
                    log("  WARNING: no <select> elements found for DOB — may need manual fill")
            except Exception as e:
                log(f"DOB fill warn: {e}")

            # Step 3: Submit
            _wait(0.5, log)
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Continue")',
                'button:has-text("Register")',
                'button:has-text("Create account")',
            ], log)
            _wait(5, log, "waiting for registration response")
            log(f"url after submit: {page.url}")

            # Step 4: Wait for captcha solve / page state change (up to 3 min)
            # Discord shows hCaptcha which must be solved manually in --no-headless mode.
            # We watch for the URL to leave /register OR body to show email sent message.
            log("waiting for captcha solve (up to 3 min) — solve manually if prompted...")
            deadline_captcha = time.time() + 180
            while time.time() < deadline_captcha:
                try:
                    cur_url = page.url
                    cur_body = page.inner_text("body")[:300].lower()
                    email_sent = any(k in cur_body for k in ["check your email", "verify", "sent", "we sent"])
                    left_register = "discord.com/register" not in cur_url
                    if email_sent or left_register:
                        log(f"captcha done (url={cur_url}, email_sent={email_sent})")
                        break
                except Exception:
                    pass
                time.sleep(3)

            # Step 5: Poll mail.tm for verification email
            log("polling mail.tm for verification email (up to 90s)...")
            try:
                verify_link = mail_client.poll_for_link(mailbox, timeout=90, keyword="discord")
                log(f"got verification link: {verify_link[:80]}...")
                page.goto(verify_link, timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
                _wait(4, log, "post-verification load")
                log(f"url after verification: {page.url}")
            except TimeoutError:
                log("no verification email received — proceeding without it")

            # Step 6: Extract token
            _wait(3, log, "app load")
            token = _extract_token_from_page(page, log)
            if not token:
                token = _intercept_token(page, log, timeout=20)

            if not token:
                raise RuntimeError("Failed to extract Discord token after registration")

            return token

        finally:
            ctx.close()


def login_via_browser(
    email: str,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Drive Discord login, return the user token."""
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"logging in {email}")
    user_data = tempfile.mkdtemp(prefix="dc_login_")

    with sync_playwright() as p:
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

        try:
            # Set up request interception before navigation
            captured_tokens: list[str] = []

            def on_request(request):
                auth = request.headers.get("authorization", "")
                if auth and len(auth) > 20 and not auth.startswith("Bearer "):
                    captured_tokens.append(auth)

            page.on("request", on_request)

            log("opening discord.com/login...")
            page.goto("https://discord.com/login", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)

            # Fill email
            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
                'input[placeholder*="Phone number" i]',
            ], email, log)
            _wait(0.5, log)

            # Fill password
            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)
            _wait(0.5, log)

            # Submit
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Log In")',
                'button:has-text("Login")',
            ], log)
            _wait(6, log, "waiting for login")
            log(f"url after login: {page.url}")

            # Wait for captured token from intercepted requests
            deadline = time.time() + 15
            while time.time() < deadline and not captured_tokens:
                time.sleep(1)

            if captured_tokens:
                token = captured_tokens[0]
                log(f"token via interception: {token[:20]}...")
                page.remove_listener("request", on_request)
                return token

            # Fallback: JS extraction
            _wait(3, log, "app load")
            token = _extract_token_from_page(page, log)
            if token:
                return token

            raise RuntimeError("Failed to extract Discord token after login")

        finally:
            ctx.close()
