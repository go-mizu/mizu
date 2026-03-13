"""Patchright-based Discord account registration and token extraction.

Flow (register):
  1. Open discord.com/register
  2. Fill username, email, password, date of birth
  3. Submit — Discord sends verification email
  4. Poll Proton Mail inbox for verification link, click it
  5. After verification, intercept Authorization header from API calls
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

import json
import os
import platform
import re
import time
import tempfile

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


def _fill_dob(page, birth_month: int, birth_day: int, birth_year: int, log) -> None:
    """Fill Discord date-of-birth selects robustly."""
    _MONTHS = ["January","February","March","April","May","June",
               "July","August","September","October","November","December"]
    month_name = _MONTHS[birth_month - 1]
    log(f"filling DOB: {month_name} {birth_day}, {birth_year}")

    # Dump select count + first few options for debugging
    try:
        # Use JS to bypass patchright selector issues (Shadow DOM etc.)
        js_info = page.evaluate("""() => {
            const sels = [...document.querySelectorAll('select')];
            const combos = [...document.querySelectorAll('[role="combobox"]')];
            return {
                selectCount: sels.length,
                selects: sels.map(s => ({id:s.id, name:s.name, aria:s.getAttribute('aria-label'), opts:Array.from(s.options).slice(0,4).map(o=>o.text)})),
                comboCount: combos.length,
                combos: combos.map(c => ({aria:c.getAttribute('aria-label'), text:c.textContent?.slice(0,30)}))
            };
        }""")
        log(f"  JS select count: {js_info['selectCount']}, combobox count: {js_info['comboCount']}")
        for s in js_info['selects']:
            log(f"    select id={s['id']!r} name={s['name']!r} aria={s['aria']!r} opts[:4]={s['opts']}")
        for c in js_info['combos']:
            log(f"    combobox aria={c['aria']!r} text={c['text']!r}")
        all_selects = page.locator('select')
        n = all_selects.count()
        log(f"  patchright select count: {n}")
    except Exception as e:
        log(f"  select dump warn: {e}")

    def _do_select(locator, value_str: str, label_str: str = "", name: str = "") -> bool:
        try:
            if locator.count() == 0:
                return False
            sel = locator.first
            if label_str:
                try:
                    sel.select_option(label=label_str)
                    log(f"  {name} set via label={label_str!r}")
                    return True
                except Exception:
                    pass
            sel.select_option(value_str)
            log(f"  {name} set via value={value_str!r}")
            return True
        except Exception as e:
            log(f"  {name} select failed: {e}")
            return False

    def _open_and_pick(aria_label: str, option_text: str, name: str) -> bool:
        """Open a Discord React dropdown and pick an option by typing + Enter."""
        try:
            container = page.locator(f'[aria-label="{aria_label}"]')
            if container.count() == 0:
                # Try by placeholder text
                container = page.locator(f'input[placeholder="{aria_label}"]')
            if container.count() == 0:
                log(f"  {name}: container not found")
                return False
            container.first.click()
            time.sleep(0.3)
            # Type the option text to filter the listbox
            container.first.type(option_text, delay=30)
            time.sleep(0.3)
            # Try clicking the first visible option in the listbox
            for opt_sel in [
                f'[role="option"]:has-text("{option_text}")',
                f'[role="listbox"] [role="option"]',
            ]:
                opts = page.locator(opt_sel).all()
                for opt in opts:
                    try:
                        if opt.is_visible():
                            txt = opt.inner_text(timeout=1000)
                            if option_text.lower() in txt.lower():
                                opt.click()
                                log(f"  {name} set via listbox: {txt[:30]!r}")
                                return True
                    except Exception:
                        pass
            # If no listbox, try pressing Enter/ArrowDown+Enter
            container.first.press("ArrowDown")
            time.sleep(0.2)
            container.first.press("Enter")
            log(f"  {name} set via keyboard ArrowDown+Enter")
            return True
        except Exception as e:
            log(f"  {name} open_and_pick failed: {e}")
        return False

    month_done = (
        _do_select(page.locator('select[aria-label="Month"]'), str(birth_month), month_name, "Month") or
        _open_and_pick("Month", month_name, "Month")
    )
    time.sleep(0.4)

    day_done = (
        _do_select(page.locator('select[aria-label="Day"]'), str(birth_day), str(birth_day), "Day") or
        _open_and_pick("Day", str(birth_day), "Day")
    )
    time.sleep(0.4)

    year_done = (
        _do_select(page.locator('select[aria-label="Year"]'), str(birth_year), str(birth_year), "Year") or
        _open_and_pick("Year", str(birth_year), "Year")
    )
    time.sleep(0.4)

    if not (month_done and day_done and year_done):
        log(f"  WARNING: DOB incomplete (month={month_done} day={day_done} year={year_done}) — fill manually")


def _poll_yopmail(yopmail_user: str, keyword: str, timeout: int, log) -> str:
    """Poll a yopmail.com inbox for a link containing keyword. Returns the URL or ''."""
    from patchright.sync_api import sync_playwright
    import tempfile

    user_data = tempfile.mkdtemp(prefix="yop_inbox_")
    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            headless=True,
            args=["--window-size=1280,900"],
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()
        try:
            inbox_url = f"https://yopmail.com/en/?login={yopmail_user}&p=inbox"
            log(f"opening yopmail inbox: {inbox_url}")
            page.goto(inbox_url, timeout=20000)
            time.sleep(5)

            seen_emails: set[str] = set()
            deadline = time.time() + timeout

            while time.time() < deadline:
                try:
                    # Access the inbox iframe (#ibx)
                    ibx = page.frame_locator("#ibx")
                    try:
                        email_rows = ibx.locator(".m, [id^='m'], .lm").all()
                        log(f"yopmail inbox rows: {len(email_rows)}")
                    except Exception:
                        email_rows = []

                    for row in email_rows[:10]:
                        try:
                            row_id = row.get_attribute("id", timeout=2000) or row.inner_text(timeout=2000)[:30]
                            if row_id in seen_emails:
                                continue
                            row.click(timeout=5000)
                            time.sleep(3)

                            # Read email from #mail iframe
                            mail_frame = page.frame_locator("#mail")
                            try:
                                mail_html = mail_frame.locator("html").inner_html(timeout=8000)
                                for url in re.findall(r"https?://[^\s\"'<>]+", mail_html):
                                    url = url.rstrip(".,;)\"'")
                                    if keyword.lower() in url.lower() and len(url) > 40:
                                        log(f"found link in yopmail: {url[:100]}")
                                        return url
                            except Exception:
                                pass

                            # Try anchor hrefs
                            try:
                                hrefs = mail_frame.locator("a[href]").evaluate_all("els => els.map(e => e.href)")
                                for url in hrefs:
                                    if keyword.lower() in url.lower() and len(url) > 40:
                                        log(f"found href in yopmail: {url[:100]}")
                                        return url
                            except Exception:
                                pass

                            seen_emails.add(row_id)
                        except Exception as e:
                            log(f"  yopmail row warn: {e}")

                    log(f"no {keyword} link in yopmail yet, waiting 10s...")
                    time.sleep(10)
                    try:
                        page.reload(timeout=10000)
                        time.sleep(3)
                    except Exception:
                        pass

                except Exception as ex:
                    log(f"yopmail poll warn: {ex}")
                    time.sleep(5)
        finally:
            ctx.close()
    return ""


def register_via_browser(
    email: str,
    username: str,
    password: str,
    birth_year: int,
    birth_month: int,
    birth_day: int,
    headless: bool = False,
    verbose: bool = True,
    proton_username: str = "",
    proton_password: str = "",
    yopmail_user: str = "",
) -> str:
    """Drive Discord registration, return the user token.

    email           — the email to use (e.g. user@proton.me or user@yopmail.com)
    yopmail_user    — if set, poll yopmail inbox and use route interception for verification
    proton_username / proton_password — if set, poll Proton Mail inbox for verification link
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering email={email} username={username}")
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

        # Intercept Authorization headers from the very start
        captured_tokens: list[str] = []

        def on_request(request):
            auth = request.headers.get("authorization", "")
            if auth and len(auth) > 20 and not auth.startswith("Bearer "):
                if auth not in captured_tokens:
                    captured_tokens.append(auth)
                    log(f"token intercepted: {auth[:20]}...")

        page.on("request", on_request)

        # If yopmail: intercept the registration API call and swap the email in the payload
        if yopmail_user and email.endswith("@yopmail.com"):
            real_email = email

            def _swap_email_in_register(route):
                req = route.request
                try:
                    body = json.loads(req.post_data or "{}")
                    if "email" in body and body.get("email", "").endswith("@gmail.com"):
                        body["email"] = real_email
                        log(f"intercepted register call — swapped email to {real_email}")
                        route.continue_(post_data=json.dumps(body).encode())
                        return
                except Exception:
                    pass
                route.continue_()

            page.route("**/api/v*/auth/register", _swap_email_in_register)
            log(f"route interceptor set for email swap ({email})")

        try:
            # Step 1: Open registration page
            log("opening discord.com/register...")
            page.goto("https://discord.com/register", timeout=30000)
            page.wait_for_load_state("networkidle", timeout=15000)
            _wait(2, log)

            # Step 2: Fill registration form
            log("filling registration form...")

            # Email — for yopmail, fill a gmail-style address to pass client-side validation;
            # the route interceptor will swap it to the real yopmail address in the HTTP request.
            fill_email = email
            if yopmail_user and email.endswith("@yopmail.com"):
                fill_email = email.replace("@yopmail.com", "@gmail.com")
                log(f"yopmail trick: filling {fill_email!r} (route interceptor will swap to {email!r})")
            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
            ], fill_email, log)
            _wait(1, log)

            # Warn if Discord shows email invalid
            try:
                err = page.inner_text("body")[:300].lower()
                if "invalid email" in err or "valid email" in err:
                    log(f"WARNING: Discord may have rejected the email domain — check browser")
            except Exception:
                pass

            # Display name (optional field, appears on some Discord versions)
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
            _wait(0.5, log)

            # Date of birth — scroll down to reveal DOB selects, then wait
            log("waiting for DOB selects to appear...")
            try:
                page.evaluate("window.scrollTo(0, document.body.scrollHeight)")
            except Exception:
                pass
            dob_deadline = time.time() + 20
            while time.time() < dob_deadline:
                try:
                    cnt = page.locator('select').count()
                    if cnt >= 1:
                        break
                    # Try clicking near the bottom to trigger lazy render
                    page.keyboard.press("Tab")
                except Exception:
                    pass
                time.sleep(0.5)
            _fill_dob(page, birth_month, birth_day, birth_year, log)

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

            # Check for username-taken error and auto-retry with different username
            try:
                _err_body = page.inner_text("body")[:400].lower()
                if "username is unavailable" in _err_body or "username is taken" in _err_body:
                    import random, string as _str
                    new_suffix = "".join(random.choices(_str.digits, k=6))
                    new_username = f"{username[:20]}{new_suffix}"
                    log(f"username taken, retrying with: {new_username}")
                    inp = page.locator('input[name="username"]')
                    if inp.count() > 0:
                        inp.first.triple_click()
                        inp.first.type(new_username, delay=40)
                        _wait(0.5, log)
                    _click_first(page, ['button[type="submit"]', 'button:has-text("Create account")'], log)
                    _wait(3, log, "after username retry")
                    username = new_username
            except Exception as e:
                log(f"username retry warn: {e}")

            # Step 4: Wait for captcha + page progression (up to 5 min)
            try:
                page.bring_to_front()
            except Exception:
                pass
            log("waiting for captcha solve (up to 5 min) — solve manually if shown...")
            deadline_captcha = time.time() + 300
            _last_body_log = 0.0
            while time.time() < deadline_captcha:
                try:
                    cur_url = page.url
                    cur_body = page.inner_text("body")[:600].lower()
                    done = (
                        "discord.com/register" not in cur_url
                        or any(k in cur_body for k in [
                            "check your email", "we sent you an email",
                            "check email", "email sent", "sent to your email",
                        ])
                    )
                    if done:
                        log(f"captcha done — url={cur_url}")
                        break
                    # Log page body every 30s to help debug
                    if time.time() - _last_body_log > 30:
                        log(f"  page body: {cur_body[:400]!r}")
                        _last_body_log = time.time()
                except Exception:
                    pass
                time.sleep(3)

            # Step 5: Verification email
            if proton_username and proton_password:
                log("polling Proton Mail inbox for Discord verification link...")
                try:
                    import sys, os as _os, threading
                    _pm_src = _os.path.join(_os.path.dirname(__file__),
                                             "../../../protonmail/src")
                    sys.path.insert(0, _os.path.abspath(_pm_src))
                    from protonmail_tool.browser import wait_for_link as pm_wait
                    # Run in thread to avoid sync-inside-async-loop conflict
                    _result: list = [None, None]
                    def _pm_thread():
                        try:
                            _result[0] = pm_wait(
                                username=proton_username,
                                password=proton_password,
                                keyword="discord",
                                timeout=120,
                                headless=False,
                                verbose=verbose,
                            )
                        except Exception as e:
                            _result[1] = e
                    t = threading.Thread(target=_pm_thread, daemon=True)
                    t.start()
                    t.join(timeout=150)
                    if _result[1]:
                        raise _result[1]
                    verify_link = _result[0]
                    if not verify_link:
                        raise RuntimeError("No verification link found")
                    log(f"verification link: {verify_link[:80]}")
                    page.goto(verify_link, timeout=30000)
                    try:
                        page.wait_for_load_state("networkidle", timeout=15000)
                    except Exception:
                        pass
                    _wait(4, log, "post-verification load")
                    log(f"url after verification: {page.url}")
                except Exception as e:
                    log(f"Proton Mail verification warn: {e}")
            elif yopmail_user:
                log(f"polling yopmail inbox for Discord verification link ({yopmail_user}@yopmail.com)...")
                try:
                    import threading
                    _yop_result: list = [None, None]

                    def _yop_thread():
                        try:
                            _yop_result[0] = _poll_yopmail(yopmail_user, "discord", timeout=180, log=log)
                        except Exception as e:
                            _yop_result[1] = e

                    t = threading.Thread(target=_yop_thread, daemon=True)
                    t.start()
                    t.join(timeout=200)
                    if _yop_result[1]:
                        raise _yop_result[1]
                    verify_link = _yop_result[0] or ""
                    if verify_link:
                        log(f"verification link: {verify_link[:80]}")
                        page.goto(verify_link, timeout=30000)
                        try:
                            page.wait_for_load_state("networkidle", timeout=15000)
                        except Exception:
                            pass
                        _wait(4, log, "post-verification load")
                        log(f"url after verification: {page.url}")
                    else:
                        log("no verification link found in yopmail within timeout")
                except Exception as e:
                    log(f"yopmail verification warn: {e}")
            else:
                log("no email credentials provided — skipping auto-verification")

            # Step 6: Extract token — try intercepted tokens first (captured since page load)
            _wait(3, log, "app load")
            token = captured_tokens[0] if captured_tokens else ""
            if token:
                log(f"token from interception: {token[:20]}...")
            if not token:
                token = _extract_token_from_page(page, log)
            if not token:
                token = _intercept_token(page, log, timeout=20)

            if not token:
                raise RuntimeError("Failed to extract Discord token after registration")

            return token

        finally:
            page.remove_listener("request", on_request)
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
