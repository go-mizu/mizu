"""Patchright browser automation for Goodreads account registration.

Flow:
  1. Open https://www.goodreads.com/user/sign_up
  2. Click "Sign up with email" → Amazon AP /ap/register
  3. Fill name, email, password, confirm password
  4. Submit → Amazon may show a CVF bot-challenge page (auto-resolves in ~10s)
     Then may show OTP verification page (email code)
  5. poll_otp_fn() is called in background to get OTP; enter it
  6. Extract session cookies

Alternative: goodread-tool login (manual login, no bot detection risk)

NOTE: Goodreads blocks headless Chrome entirely (returns empty body).
Always use headless=False.
"""
from __future__ import annotations

import base64
import json
import os
import platform
import re
import tempfile
import time
from typing import Callable


# ---------------------------------------------------------------------------
# Helpers
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


def _fill_input(page, selector: str, text: str, log=None) -> None:
    el = page.locator(selector).first
    el.wait_for(state="visible", timeout=12000)
    el.click()
    time.sleep(0.3)
    el.fill("")
    el.type(text, delay=60)
    time.sleep(0.5)
    if log:
        log(f"filled {selector!r}")


def _fill_first(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            loc = page.locator(sel)
            if loc.count() > 0 and loc.first.is_visible():
                _fill_input(page, sel, text, log)
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


def _body_text(page, max_chars: int = 800) -> str:
    try:
        return page.inner_text("body")[:max_chars]
    except Exception:
        return ""


def _open_context(p, headless: bool, user_data: str):
    """Open Chromium context using system Chrome on macOS.

    NOTE: Goodreads blocks headless browsers — use headless=False.
    """
    if platform.system() == "Darwin":
        chrome_path = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        channel = "chrome" if os.path.exists(chrome_path) else None
    else:
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
# Amazon image CAPTCHA auto-solver
# ---------------------------------------------------------------------------

def _solve_amazon_puzzle(page, log=None) -> bool:
    """Auto-solve Amazon image-selection CAPTCHA using Claude vision API.

    Amazon shows a grid of images with instructions like "Choose all the curtains".
    This function screenshots the puzzle, asks Claude which images match,
    clicks them, and submits.

    Returns True if a puzzle was detected and attempted, False if no puzzle found.
    """
    body = _body_text(page, 600)
    if "Solve this puzzle" not in body:
        return False

    # Extract instruction (e.g., "Choose all the curtains")
    m = re.search(r'Choose all (?:the )?(.+?)(?:\n|$)', body, re.IGNORECASE)
    category = m.group(0).strip() if m else "the matching items"
    if log:
        log(f"CAPTCHA puzzle detected: {category!r}")

    # Take a full-page screenshot for Claude
    screenshot_bytes = page.screenshot(full_page=False)
    b64_img = base64.standard_b64encode(screenshot_bytes).decode()

    # Find clickable image grid elements — try several selector patterns
    grid_locator = None
    for sel in [
        'div[class*="puzzle"] img',
        'div[class*="captcha"] img',
        'img[src*="captcha"]',
        'img[src*="puzzle"]',
        '[role="checkbox"] img',
        'div[tabindex="0"] img',
        'div[onclick] img',
        # Amazon's own widget often puts images inside clickable containers
        'div.a-box img',
        'table img',
        'td img',
    ]:
        loc = page.locator(sel)
        if loc.count() >= 2:
            grid_locator = loc
            if log:
                log(f"found {loc.count()} puzzle images via selector: {sel!r}")
            break

    # Fallback: any image smaller than 200x200 (typical puzzle tile size)
    if grid_locator is None:
        all_imgs = page.locator("img")
        candidates = []
        for i in range(min(all_imgs.count(), 30)):
            try:
                box = all_imgs.nth(i).bounding_box()
                if box and 20 < box["width"] < 300 and 20 < box["height"] < 300:
                    candidates.append(i)
            except Exception:
                pass
        if candidates:
            if log:
                log(f"fallback: found {len(candidates)} candidate images by size")
            # Rebuild locator from the candidate indices
            grid_locator = all_imgs  # we'll index manually

    # Ask Claude which images match the instruction
    try:
        import anthropic
        client = anthropic.Anthropic()
        response = client.messages.create(
            model="claude-opus-4-6",
            max_tokens=256,
            messages=[{
                "role": "user",
                "content": [
                    {
                        "type": "image",
                        "source": {
                            "type": "base64",
                            "media_type": "image/png",
                            "data": b64_img,
                        },
                    },
                    {
                        "type": "text",
                        "text": (
                            f"This is an Amazon image CAPTCHA. "
                            f"The instruction says: '{category}'. "
                            f"The page shows a grid of images. "
                            f"Count the images in the grid from left-to-right, "
                            f"top-to-bottom starting at 1. "
                            f"Return ONLY a JSON array of 1-based positions that match "
                            f"the instruction. Example: [2, 5]. "
                            f"If no images match, return []. "
                            f"Return ONLY the JSON array, nothing else."
                        ),
                    },
                ],
            }],
        )
        positions_text = response.content[0].text.strip()
        if log:
            log(f"Claude response: {positions_text}")

        pm = re.search(r'\[[\d,\s]*\]', positions_text)
        positions = json.loads(pm.group()) if pm else []
    except Exception as e:
        if log:
            log(f"Claude API call failed: {e} — leaving puzzle for manual solve")
        return True  # puzzle was detected, even if unsolved

    if not positions:
        if log:
            log("Claude found no matching images in grid")
    else:
        if log:
            log(f"clicking grid positions: {positions}")
        if grid_locator is not None:
            total = grid_locator.count()
            for pos in positions:
                idx = pos - 1
                if 0 <= idx < total:
                    try:
                        el = grid_locator.nth(idx)
                        # Click the parent container if the img itself isn't directly clickable
                        try:
                            el.click(timeout=3000)
                        except Exception:
                            el.locator("..").click(timeout=3000)
                        time.sleep(0.4)
                        if log:
                            log(f"  clicked position {pos}")
                    except Exception as e:
                        if log:
                            log(f"  failed to click position {pos}: {e}")

    # Click Confirm / Submit
    time.sleep(0.5)
    clicked = _click_first(page, [
        'button:has-text("Confirm")',
        'input[value="Confirm"]',
        'input[type="submit"]',
        'button[type="submit"]',
        'a:has-text("Confirm")',
    ], log)
    if log:
        log(f"puzzle submit: {clicked!r}")
    time.sleep(2)
    return True


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def register_via_browser(
    name: str,
    email: str,
    password: str,
    poll_otp_fn: Callable[[], str],
    headless: bool = False,
    verbose: bool = False,
) -> list[dict]:
    """Register a Goodreads account and return session cookies.

    poll_otp_fn: callable that blocks until an OTP code is available and
    returns it as a string (e.g. "123456"). Called in a background thread.

    headless=False is default — Goodreads blocks headless Chrome.
    Returns list of Playwright-format cookie dicts.
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [goodread-browser] {msg}", flush=True)

    log(f"registering {email} (headless={headless})")
    user_data = tempfile.mkdtemp(prefix="gr_reg_")

    with sync_playwright() as p:
        ctx = _open_context(p, headless, user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ── Step 1: Open sign_up page ─────────────────────────────────
            log("opening goodreads.com/user/sign_up ...")
            page.goto("https://www.goodreads.com/user/sign_up", timeout=60000)
            try:
                page.wait_for_load_state("networkidle", timeout=20000)
            except Exception:
                pass
            _wait(2, log)
            log(f"url: {page.url}")

            body = _body_text(page)
            if not body.strip():
                raise RuntimeError(
                    "Goodreads sign_up page returned empty body — "
                    "Goodreads blocks headless Chrome. Use headless=False."
                )

            # ── Step 2: Navigate to Amazon AP registration form ───────────
            # "Sign up with email" is an <a> link, not a form button
            signup_href = ""
            for i in range(page.locator("a").count()):
                try:
                    el = page.locator("a").nth(i)
                    if "sign up with email" in el.inner_text().lower():
                        signup_href = el.get_attribute("href") or ""
                        break
                except Exception:
                    pass

            if signup_href:
                log(f"navigating to AP register: {signup_href[:80]}")
                page.goto(signup_href, timeout=30000)
            else:
                _click_first(page, ['a:has-text("Sign up with email")'], log)
            try:
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception:
                pass
            _wait(2, log)
            log(f"url: {page.url}")

            # ── Step 3: Fill Amazon AP form ───────────────────────────────
            # Fields: #ap_customer_name, #ap_email, #ap_password,
            #         #ap_password_check, input#continue (submit)
            body = _body_text(page)
            log(f"AP page body: {body[:150]!r}")

            _fill_first(page, [
                'input#ap_customer_name', 'input[name="customerName"]',
                'input[placeholder*="name" i]',
            ], name, log)
            _wait(0.5, log)

            _fill_first(page, [
                'input#ap_email', 'input[name="email"]', 'input[type="email"]',
            ], email, log)
            _wait(0.5, log)

            _fill_first(page, [
                'input#ap_password', 'input[name="password"]',
            ], password, log)
            _wait(0.5, log)

            _fill_first(page, [
                'input#ap_password_check', 'input[name="passwordCheck"]',
            ], password, log)
            _wait(0.8, log)

            # ── Step 4: Submit ────────────────────────────────────────────
            log("submitting AP form...")
            _click_first(page, [
                'input#continue', 'input[type="submit"]',
                'button[type="submit"]',
            ], log)
            _wait(3, log, "post-submit")
            log(f"url after submit: {page.url}")

            # ── Step 5: Handle CVF (Amazon bot-challenge) ─────────────────
            # /ap/cvf/ is Amazon's Contact Verification Flow — it may be:
            #   a) Bot challenge that auto-resolves (aamation JS runs, redirects)
            #   b) OTP verification page
            # We wait up to 30s for the CVF page to auto-advance.
            cvf_wait_max = 270  # seconds — must exceed protonmail-tool startup + poll time
            cvf_start = time.time()
            log("starting Proton Mail OTP poll in background while waiting for CVF...")

            # Start OTP poll in background thread
            otp_result: list[str] = []
            import threading
            def _poll_otp():
                try:
                    otp = poll_otp_fn()
                    otp_result.append(otp)
                    log(f"OTP received: {otp}")
                except Exception as e:
                    log(f"OTP poll ended: {e}")
            otp_thread = threading.Thread(target=_poll_otp, daemon=True)
            otp_thread.start()

            while time.time() - cvf_start < cvf_wait_max:
                cur_url = page.url
                body_snippet = _body_text(page, 300)
                log(f"  CVF wait: url={cur_url[:60]} body={body_snippet[:120]!r}")

                # Success: no longer on CVF or AP, landed on Goodreads
                if "/ap/cvf/" not in cur_url and "/ap/" not in cur_url:
                    log("left CVF/AP pages — continuing")
                    break

                # Check for image CAPTCHA puzzle and auto-solve it
                if "Solve this puzzle" in body_snippet:
                    if log:
                        log("image CAPTCHA detected — attempting auto-solve...")
                    _solve_amazon_puzzle(page, log)
                    _wait(2, log, "post-puzzle")
                    continue

                # Check if OTP input appeared on page
                body = _body_text(page, 500)
                otp_inputs = page.locator(
                    'input[name="code"], input[id="auth-mfa-otpcode"], '
                    'input[autocomplete="one-time-code"], '
                    'input[type="text"][maxlength="6"]'
                )
                if otp_inputs.count() > 0 and otp_inputs.first.is_visible():
                    log("OTP input appeared on page!")
                    # Click "Resend code" to trigger another email send (helps with delays)
                    resend_clicked = _click_first(page, [
                        'a:has-text("Resend")', 'button:has-text("Resend")',
                        'a:has-text("resend")', 'span:has-text("Resend")',
                        'a[id*="resend"]', 'button[id*="resend"]',
                    ], log)
                    if resend_clicked:
                        log("clicked Resend — Amazon will resend the OTP email")
                    # Wait up to 240s — protonmail-tool needs time to start + poll inbox
                    for i in range(240):
                        if otp_result:
                            break
                        # Click resend every 60s if still waiting
                        if i > 0 and i % 60 == 0:
                            _click_first(page, [
                                'a:has-text("Resend")', 'button:has-text("Resend")',
                                'a:has-text("resend")',
                            ], log)
                            log(f"re-clicked Resend at t={i}s")
                        time.sleep(1)

                    if otp_result:
                        _fill_first(page, [
                            'input[name="code"]',
                            'input[id="auth-mfa-otpcode"]',
                            'input[autocomplete="one-time-code"]',
                            'input[type="text"][maxlength="6"]',
                        ], otp_result[0], log)
                        _wait(0.5, log)
                        _click_first(page, [
                            'input[type="submit"]',
                            'button[type="submit"]',
                            'button:has-text("Verify")',
                            'button:has-text("Continue")',
                        ], log)
                        _wait(2, log, "OTP submit")
                        log(f"url after OTP: {page.url}")
                        # Wait for natural redirect first (up to 10s)
                        for _ in range(10):
                            cur = page.url
                            if "goodreads.com" in cur and "/ap/" not in cur:
                                log(f"auto-redirected to goodreads: {cur[:60]}")
                                break
                            time.sleep(1)
                        # If still on /ap/cvf/verify, navigate to ap-handler to complete registration
                        if "/ap/" in page.url:
                            log("navigating to ap-handler/register to complete account creation...")
                            page.goto("https://www.goodreads.com/ap-handler/register", timeout=30000)
                            try:
                                page.wait_for_load_state("networkidle", timeout=15000)
                            except Exception:
                                pass
                        _wait(3, log, "post-OTP settle")
                        log(f"url after settle: {page.url}")
                    else:
                        log("WARNING: no OTP received — manual verification needed")
                    break

                time.sleep(3)

            # ── Step 6: Navigate to Goodreads after AP flow ───────────────
            final_url = page.url
            if "goodreads.com" not in final_url or "/ap/" in final_url:
                log(f"navigating to goodreads.com from {final_url[:60]}")
                page.goto("https://www.goodreads.com/", timeout=30000)
                try:
                    page.wait_for_load_state("networkidle", timeout=10000)
                except Exception:
                    pass
                _wait(2, log)

            log(f"final url: {page.url}")
            body = _body_text(page, 400)
            log(f"final body: {body[:200]!r}")

            # ── Step 7: Extract cookies ───────────────────────────────────
            log("extracting cookies...")
            all_cookies = ctx.cookies()
            gr_cookies = [
                c for c in all_cookies
                if any(d in c.get("domain", "").lower()
                       for d in ["goodreads", "amazon"])
            ]
            if not gr_cookies:
                gr_cookies = all_cookies

            log(f"extracted {len(gr_cookies)} cookies: {[c['name'] for c in gr_cookies]}")

            if not gr_cookies:
                raise RuntimeError("No cookies extracted after registration")

            return gr_cookies

        finally:
            ctx.close()


# ---------------------------------------------------------------------------
# Manual login — open browser, user logs in themselves
# ---------------------------------------------------------------------------

def login_via_browser(
    verbose: bool = False,
    timeout: int = 300,
) -> list[dict]:
    """Open a browser to goodreads.com/user/sign_in and wait for the user to log in.

    Returns session cookies once login is detected.
    This is the most reliable approach — no bot detection risk.
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [goodread-login] {msg}", flush=True)

    print(
        "\n[LOGIN] A browser window will open — log in to your Goodreads account.\n"
        "        The tool will automatically detect when you're logged in.\n",
        flush=True,
    )

    user_data = tempfile.mkdtemp(prefix="gr_login_")
    with sync_playwright() as p:
        ctx = _open_context(p, headless=False, user_data=user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        page.goto("https://www.goodreads.com/user/sign_in", timeout=30000)
        try:
            page.wait_for_load_state("networkidle", timeout=10000)
        except Exception:
            pass

        log("waiting for login (checking URL every 2s)...")
        deadline = time.time() + timeout
        while time.time() < deadline:
            cur_url = page.url
            log(f"  url={cur_url[:80]}")
            # Logged in if we're on goodreads.com without sign_in/ap/ in the URL
            if "goodreads.com" in cur_url and "sign_in" not in cur_url and "/ap/" not in cur_url:
                # Give the page a moment to render
                time.sleep(2)
                body = _body_text(page, 1000)
                log(f"  body snippet: {body[:200]!r}")
                # Accept if we see any logged-in indicators OR just if URL looks right
                logged_in_signals = ["Sign out", "My Books", "my-books", "profile", "shelf"]
                if any(s.lower() in body.lower() for s in logged_in_signals):
                    log(f"logged in (signal found)! url={cur_url}")
                    break
                # Fallback: if URL is on goodreads.com main pages, assume logged in
                if any(p in cur_url for p in ["/home", "/review/list", "/user/show", "goodreads.com/"]):
                    if len(body) > 500:  # page has real content
                        log(f"logged in (url heuristic)! url={cur_url}")
                        break
            time.sleep(2)
        else:
            raise TimeoutError(f"Login not detected within {timeout}s")

        _wait(2, log, "post-login stabilize")
        all_cookies = ctx.cookies()
        gr_cookies = [
            c for c in all_cookies
            if any(d in c.get("domain", "").lower() for d in ["goodreads", "amazon"])
        ]
        if not gr_cookies:
            gr_cookies = all_cookies
        log(f"extracted {len(gr_cookies)} cookies")
        ctx.close()
        return gr_cookies


# ---------------------------------------------------------------------------
# Cookie test — verify cookies authenticate with Goodreads
# ---------------------------------------------------------------------------

def test_cookies(cookies: list[dict], verbose: bool = False) -> str | None:
    """Test that stored cookies authenticate with Goodreads. Returns user_id or None."""
    import httpx

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [goodread-test] {msg}", flush=True)

    jar = {c["name"]: c["value"] for c in cookies if c.get("name")}
    headers = {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
    }

    try:
        client = httpx.Client(
            cookies=jar, headers=headers, follow_redirects=True, timeout=20,
        )
        # Test a page that requires login
        resp = client.get("https://www.goodreads.com/review/list/me")
        log(f"GET /review/list/me -> {resp.status_code}, url={resp.url}")

        if "sign_in" in str(resp.url):
            log("not authenticated — cookies rejected")
            client.close()
            return None

        # Also try the homepage to extract user_id
        resp2 = client.get("https://www.goodreads.com/")
        client.close()
        log(f"GET / -> {resp2.status_code}")

        body = resp2.text
        m = re.search(r'/user/show/(\d+)', body)
        if m:
            user_id = m.group(1)
            log(f"logged in as user_id={user_id}")
            return user_id

        if "sign_in" in str(resp2.url):
            return None

        log("appears logged in")
        return "unknown"

    except Exception as e:
        log(f"test error: {e}")
        return None
