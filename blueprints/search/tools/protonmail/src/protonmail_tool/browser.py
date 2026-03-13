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


def _auto_solve_puzzle(page, log) -> bool:
    """Attempt to auto-solve Proton's drag-puzzle captcha.

    The captcha is rendered inside a cross-origin iframe so JS DOM queries can't
    reach it.  We use mouse coordinates derived from visual analysis of screenshots:
    - Viewport: 1280x900
    - The 'Human Verification' dialog takes up the center of the page
    - The puzzle piece thumbnail is at roughly (456, 286)
    - The hole (target) is in the puzzle image below, y≈566
    - Puzzle image x spans roughly 460–815
    Strategy: drag piece from thumbnail position diagonally to each x position
    along y=566 (the hole row) until the captcha accepts.
    """
    # Piece thumbnail start position
    piece_x, piece_y = 456, 286
    # Hole target row (below the piece thumbnail, in the puzzle image)
    target_y = 566
    # Puzzle image x range to scan
    puzzle_left_x = 460
    puzzle_right_x = 815

    log(f"  auto-solve: dragging piece from ({piece_x},{piece_y}) to y={target_y}, scanning x={puzzle_left_x}..{puzzle_right_x}")

    for target_x in range(puzzle_left_x, puzzle_right_x, 25):
        log(f"  drag to ({target_x},{target_y})")
        try:
            page.mouse.move(piece_x, piece_y)
            page.mouse.down()
            time.sleep(0.2)
            steps = 20
            for i in range(1, steps + 1):
                ix = piece_x + (target_x - piece_x) * i / steps
                iy = piece_y + (target_y - piece_y) * i / steps
                page.mouse.move(ix, iy)
                time.sleep(0.03)
            page.mouse.up()
            time.sleep(1.5)
        except Exception as e:
            log(f"  drag warn: {e}")
            return False

        # Check if captcha was accepted (URL changed or dialog gone)
        try:
            url = page.url
            if "signup" not in url:
                log("  URL changed after drag — captcha likely accepted!")
                return True
        except Exception:
            pass

        try:
            ss_path = f"/tmp/proton_drag_{int(time.time())}.png"
            page.screenshot(path=ss_path)
            log(f"  drag screenshot: {ss_path}")
        except Exception:
            pass

        time.sleep(0.5)

    return False


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

            # ── Step 3: Wait for username field ──────────────────────────
            # The main page has username+password+submit (React form, actual submission).
            # The iframes are anti-bot challenge overlays — filling them directly with click+type
            # triggers Proton's postMessage sync to the main page React state.
            # Strategy: find the FIRST VISIBLE username field (main page or iframe), click+type.
            log(f"waiting for username field (up to 60s)...")
            username_frame = None
            username_sel = None
            _USERNAME_SELS = [
                'input[id="username"]', 'input[name="username"]',
                'input[placeholder*="username" i]', 'input[autocomplete="username"]',
            ]
            deadline_u = time.time() + 60
            while time.time() < deadline_u:
                frame_urls = [f.url for f in page.frames]
                log(f"  frames ({len(frame_urls)}): {frame_urls}")

                # Check main page first (it has the React-controlled form that actually submits)
                for sel in _USERNAME_SELS:
                    try:
                        loc = page.locator(sel)
                        if loc.count() > 0 and loc.first.is_visible():
                            username_frame = page
                            username_sel = sel
                            break
                    except Exception:
                        pass

                # Fall back to iframes if main page has no visible field
                if not username_sel:
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

                if username_sel:
                    break
                time.sleep(3)

            if username_frame and username_sel:
                log(f"  found username via {username_sel!r} in frame={getattr(username_frame, 'url', 'main')[:60]}")
                el = username_frame.locator(username_sel).first
                el.click(); time.sleep(0.3)
                el.fill("")
                el.type(username, delay=55); time.sleep(0.5)
                # Press Tab to trigger blur → challenge iframe sends token to main page via postMessage
                el.press("Tab"); time.sleep(3)
                # Verify value was set
                try:
                    actual_val = el.input_value()
                    log(f"  username field value after type: {actual_val!r}")
                except Exception:
                    pass
            else:
                log("  WARNING: username field not found — fill manually")

            # Dump all visible inputs for debugging
            try:
                js_inputs = page.evaluate("""() => {
                    const all = [];
                    const inputs = document.querySelectorAll('input, button[type="submit"], button:not([type])');
                    for (const el of inputs) {
                        all.push({tag: el.tagName, type: el.type, id: el.id, name: el.name, placeholder: el.placeholder, value: el.value?.slice(0,20), visible: el.offsetParent !== null, text: el.textContent?.slice(0, 30)});
                    }
                    return all;
                }""")
                log(f"  main page inputs: {[x for x in js_inputs if x.get('visible')]}")
            except Exception as e:
                log(f"  input dump warn: {e}")

            # ── Step 5: Fill password + confirm password ─────────────────────
            log("filling password...")
            pwd_frame = username_frame if username_frame else page
            pwd_filled = False
            _PWD_SELS = ['input[id="password"]', 'input[name="password"]', 'input[type="password"]']
            # Wait up to 10s for password to appear
            pwd_deadline = time.time() + 10
            while time.time() < pwd_deadline and not pwd_filled:
                frames_to_try = ([pwd_frame] +
                                 [f for f in page.frames if f != pwd_frame and f != page.main_frame] +
                                 [page])
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
            # Always try to fill confirm password field (may have different selector)
            _CONFIRM_SELS = [
                'input[placeholder*="Confirm" i]',
                'input[placeholder*="Repeat" i]',
                'input[id*="confirm" i]',
                'input[name*="confirm" i]',
                'input[autocomplete="new-password"]:nth-of-type(2)',
            ]
            for sel in _CONFIRM_SELS:
                try:
                    els = [loc for loc in page.locator(sel).all() if loc.is_visible()]
                    if els:
                        els[0].click(); time.sleep(0.2)
                        els[0].fill(""); els[0].type(password, delay=55); time.sleep(0.3)
                        log(f"  confirm password via {sel!r}")
                        break
                except Exception:
                    pass
                if not pwd_filled:
                    time.sleep(1)
            if not pwd_filled:
                log("  WARNING: password field not found — fill manually")
            _wait(0.5, log)

            # ── Step 5: Dismiss upsell modal if present ───────────────────
            # Proton shows a "Mail Plus Special Offer" modal after filling the form.
            # This modal's "Get limited-time offer" is a button[type="submit"] that
            # would capture our submit click. Dismiss it first.
            for _ in range(3):
                dismissed = _click_first(page, [
                    'button:has-text("No, thanks")',
                    'button:has-text("Maybe later")',
                    'button:has-text("Skip")',
                    '[aria-label="Close"]',
                    'button[data-testid="modal-close-button"]',
                ], log)
                if dismissed:
                    _wait(1, log, "modal dismissed")
                    break
                time.sleep(0.5)

            # ── Step 5: Submit ────────────────────────────────────────────
            _click_first(page, [
                'button:has-text("Start using")',
                'button:has-text("Create account")',
                'button[type="submit"]',
                'button:has-text("Continue")',
                'button:has-text("Next")',
            ], log)
            _wait(3, log, "after submit")
            log(f"url after submit: {page.url}")

            # ── Step 5b: Dismiss "Mail Plus" upsell modal if it appears ──
            # Proton shows a promotional modal BEFORE the captcha, blocking it.
            # Dismiss it immediately so the captcha becomes visible.
            for _attempt in range(5):
                dismissed = _click_first(page, [
                    'button:has-text("No, thanks")',
                    'button:has-text("Maybe later")',
                    'button:has-text("No thanks")',
                    '[data-testid="modal-close-button"]',
                    'button[aria-label="Close"]',
                ], log)
                if dismissed:
                    log(f"  dismissed modal attempt {_attempt+1}")
                    _wait(1.5, log)
                else:
                    break

            # ── Step 6: Wait for captcha solve (up to 5 min) ─────────────
            try:
                page.bring_to_front()
            except Exception:
                pass
            # Screenshot to show current state
            try:
                _ss_path = f"/tmp/proton_signup_{int(time.time())}.png"
                page.screenshot(path=_ss_path)
                log(f"  screenshot saved: {_ss_path}")
            except Exception as e:
                log(f"  screenshot warn: {e}")
            # Capture baseline body text immediately after submit (before captcha overlay)
            _baseline_body = _body(page, 2000)
            log("=" * 60)
            log("SOLVE CAPTCHA: switch to browser, drag puzzle piece to hole")
            log("=" * 60)
            # macOS notification to alert user
            try:
                import subprocess
                subprocess.Popen(["osascript", "-e",
                    'display notification "Drag puzzle piece to hole in browser" with title "Proton: SOLVE CAPTCHA NOW"'])
            except Exception:
                pass

            deadline = time.time() + 600  # 10 minutes
            _last_screenshot = 0
            _last_puzzle_attempt = 0
            while time.time() < deadline:
                url = page.url
                body = _body(page, 2000)
                past_captcha = (
                    "account.proton.me/signup" not in url
                    or "congratulations" in body
                    or "recovery" in body
                    or "skip" in body
                    or "set up your" in body
                    or "mail.proton.me" in url
                    or "proton.me/u/" in url
                    or "account.proton.me/mail" in url
                    or "account.proton.me/setup" in url
                    or ("password" not in body and len(body) > 100 and body != _baseline_body)
                )
                if past_captcha:
                    log(f"captcha passed — url: {url}")
                    break

                # Try auto-solve puzzle every 30s if still on captcha
                if time.time() - _last_puzzle_attempt > 30:
                    _last_puzzle_attempt = time.time()
                    try:
                        _auto_solve_puzzle(page, log)
                    except Exception as e:
                        log(f"  auto-solve warn: {e}")

                if time.time() - _last_screenshot > 15:
                    try:
                        _ss = f"/tmp/proton_captcha_{int(time.time())}.png"
                        page.screenshot(path=_ss)
                        log(f"  captcha screenshot: {_ss}")
                        _last_screenshot = time.time()
                    except Exception:
                        pass
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
            # Check both URL and page content (account.proton.me/mail can be the login page)
            _body_txt = ""
            try:
                _body_txt = page.inner_text("body")[:300].lower()
            except Exception:
                pass
            _needs_login = (
                "login" in page.url
                or "sign in" in _body_txt
                or "email or username" in _body_txt
                or "mail.proton.me" not in page.url and "inbox" not in page.url
            )
            if _needs_login:
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

            # Wait for Proton to settle after login
            _wait(5, log)
            log(f"url after inbox nav: {page.url}")
            # Detect which Proton UI version we're in
            if "account.proton.me" in page.url:
                _base = "https://account.proton.me/mail"
            else:
                _base = "https://mail.proton.me/u/0"
            # Wait for Proton Mail app to finish loading (sign-in loading screen disappears)
            log("waiting for Proton Mail app to load...")
            _load_deadline = time.time() + 30
            while time.time() < _load_deadline:
                try:
                    body_text = page.inner_text("body")[:200].lower()
                    if "sign in" not in body_text and "loading" not in body_text:
                        break
                except Exception:
                    pass
                time.sleep(2)
            try:
                body_preview = page.inner_text("body")[:600]
                log(f"  inbox page body: {body_preview[:300]!r}")
            except Exception:
                pass

            # Poll for new email containing keyword
            log(f"polling inbox for link (keyword={keyword!r}, timeout={timeout}s)...")
            seen: set[str] = set()
            deadline = time.time() + timeout

            def _dismiss_modals(pg) -> None:
                """Dismiss any overlay modals blocking the inbox — loop until all gone."""
                for _attempt in range(8):
                    # Check if any modal overlay is present
                    try:
                        modal = pg.locator('div.modal-two')
                        if modal.count() == 0 or not modal.first.is_visible():
                            break
                    except Exception:
                        break

                    dismissed = False
                    # Try close/skip buttons first (avoid "Next" which advances wizards)
                    for sel in [
                        'button[data-testid="modal-close-button"]',
                        'button[aria-label="Close"]',
                        'button[aria-label="Close modal"]',
                        'button:has-text("Skip")',
                        'button:has-text("Maybe later")',
                        'button:has-text("No, thanks")',
                        'button:has-text("No thanks")',
                        'button:has-text("Got it")',
                        'button:has-text("Dismiss")',
                        'button:has-text("Close")',
                        'button:has-text("Let\'s get started")',
                    ]:
                        try:
                            btn = pg.locator(sel)
                            if btn.count() > 0 and btn.first.is_visible():
                                btn.first.click()
                                log(f"  dismissed modal: {sel}")
                                time.sleep(0.8)
                                dismissed = True
                                break
                        except Exception:
                            pass
                    if not dismissed:
                        # Escape as fallback for wizard/tour modals
                        try:
                            pg.keyboard.press("Escape")
                            log("  dismissed modal via Escape")
                            time.sleep(0.5)
                        except Exception:
                            break

            # Dismiss any onboarding modals immediately after login
            _dismiss_modals(page)

            # Try multiple selectors for both old (mail.proton.me) and new (account.proton.me/mail) UI
            ROW_SEL = (
                '[data-shortcut-target="item-container"], '
                '[data-element-id], '
                '[data-testid="message-item"], '
                '.message-list-item, '
                '[role="row"]'
            )

            def _scan_folder(folder_url: str) -> str:
                """Scan a Proton Mail folder. Returns found URL or empty string."""
                try:
                    page.goto(folder_url, timeout=20000)
                    try:
                        page.wait_for_load_state("networkidle", timeout=10000)
                    except Exception:
                        pass
                    _wait(2, log)
                except Exception as e:
                    log(f"navigate warn: {e}")
                    return ""

                _dismiss_modals(page)

                # Snapshot row IDs before clicking anything
                try:
                    row_els = page.locator(ROW_SEL).all()
                    # Also check JS count for debugging
                    try:
                        js_count = page.evaluate(f"() => document.querySelectorAll({ROW_SEL.split(',')[0]!r}).length + document.querySelectorAll('[data-element-id]').length")
                    except Exception:
                        js_count = "?"
                    log(f"  found {len(row_els)} rows (js_count={js_count}) in {page.url}")
                    row_ids = []
                    for el in row_els:
                        try:
                            rid = el.get_attribute("data-element-id", timeout=3000) or ""
                            row_ids.append(rid)
                        except Exception:
                            row_ids.append("")
                except Exception as e:
                    log(f"  row snapshot warn: {e}")
                    return ""

                for i, rid in enumerate(row_ids):
                    if rid and rid in seen:
                        continue
                    try:
                        # Re-find the row after any navigation
                        rows_now = page.locator(ROW_SEL).all()
                        if i >= len(rows_now):
                            break
                        row = rows_now[i]
                        try:
                            text = row.inner_text(timeout=5000)
                        except Exception:
                            text = f"row[{i}]"
                        log(f"  opening[{i}]: {text[:80]!r}")
                        _dismiss_modals(page)
                        try:
                            row.click(timeout=8000)
                        except Exception:
                            row.click(force=True)
                        _wait(5, log)

                        # Dismiss any modals triggered by opening the email
                        _dismiss_modals(page)

                        # Extract URLs from all frames (Proton renders email
                        # in an about:blank iframe with allow-same-origin sandbox)
                        all_urls: list[str] = []
                        for frame in page.frames:
                            try:
                                hrefs = frame.evaluate(
                                    "() => Array.from(document.querySelectorAll('a[href]')).map(a => a.href)"
                                )
                                all_urls += [h for h in hrefs if h.startswith("http")]
                            except Exception:
                                pass
                        # Also check visible text for URLs
                        try:
                            body_text = page.inner_text("body")
                            all_urls += re.findall(r"https?://\S+", body_text)
                        except Exception:
                            pass
                        for url in all_urls:
                            url = url.rstrip(".,;)")
                            if keyword and keyword.lower() in url.lower():
                                log(f"found link: {url[:80]}")
                                return url
                        if rid:
                            seen.add(rid)
                        page.go_back()
                        _wait(2, log)
                        _dismiss_modals(page)
                    except Exception as e:
                        log(f"  row[{i}] warn: {e}")
                        try:
                            page.goto(folder_url, timeout=10000)
                            _wait(2, log)
                        except Exception:
                            pass
                return ""

            if "account.proton.me" in _base:
                # New Proton web app — check inbox, spam, all-mail, newsletters
                FOLDERS = [
                    _base,                                          # inbox
                    "https://mail.proton.me/u/0/spam",             # spam
                    "https://mail.proton.me/u/0/newsletters",      # newsletters (transactional)
                    "https://mail.proton.me/u/0/all-mail",         # all mail
                    _base,                                          # inbox reload
                ]
            else:
                FOLDERS = [
                    f"{_base}/inbox",
                    f"{_base}/spam",
                    f"{_base}/all-mail",
                ]

            while time.time() < deadline:
                for folder in FOLDERS:
                    result = _scan_folder(folder)
                    if result:
                        return result
                    if time.time() >= deadline:
                        break
                _wait(10, log, "waiting for email")

            raise TimeoutError(f"No email with link received within {timeout}s")

        finally:
            ctx.close()


def wait_for_otp(
    username: str,
    password: str,
    timeout: int = 120,
    headless: bool = False,
    verbose: bool = True,
) -> str:
    """Log in to Proton Mail web and poll inbox for a numeric OTP/verification code.

    Amazon/Goodreads sends 6-digit OTP codes for account verification.
    Returns the code string (e.g. "123456").
    """
    from patchright.sync_api import sync_playwright

    _ensure_display()

    def log(msg: str) -> None:
        if verbose:
            print(f"[{time.strftime('%H:%M:%S')}] [proton-otp] {msg}", flush=True)

    user_data = tempfile.mkdtemp(prefix="pm_otp_")

    with sync_playwright() as p:
        ctx = _open_context(p, headless, user_data)
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        try:
            log("opening mail.proton.me/inbox ...")
            page.goto("https://mail.proton.me/u/0/inbox", timeout=60000)
            try:
                page.wait_for_load_state("networkidle", timeout=20000)
            except Exception:
                pass
            _wait(3, log)
            log(f"url: {page.url}")

            _body_txt = ""
            try:
                _body_txt = page.inner_text("body")[:300].lower()
            except Exception:
                pass
            _needs_login = (
                "login" in page.url
                or "sign in" in _body_txt
                or "email or username" in _body_txt
                or ("mail.proton.me" not in page.url and "inbox" not in page.url)
            )
            if _needs_login:
                log("not logged in — filling credentials ...")
                _fill_first(page, ['input[id="username"]', 'input[name="username"]',
                                   'input[type="text"]'], username, log)
                _wait(0.5, log)
                _click_first(page, ['button:has-text("Continue")', 'button[type="submit"]'], log)
                _wait(1, log)
                _fill_first(page, ['input[id="password"]', 'input[name="password"]',
                                   'input[type="password"]'], password, log)
                _wait(0.5, log)
                _click_first(page, ['button:has-text("Sign in")', 'button[type="submit"]'], log)
                try:
                    page.wait_for_load_state("networkidle", timeout=20000)
                except Exception:
                    pass
                _wait(3, log)
                log(f"after login url: {page.url}")

            _base = page.url.rstrip("/")
            if "account.proton.me" in _base:
                _base = "https://mail.proton.me/u/0/inbox"
            elif not _base.endswith("/inbox"):
                _base = "https://mail.proton.me/u/0/inbox"

            ROW_SEL = (
                '[data-shortcut-target="item-container"], '
                '[data-element-id], '
                '[data-testid="message-item"], '
                '.message-list-item, '
                '[role="row"]'
            )

            seen: set[str] = set()
            deadline = time.time() + timeout
            log(f"polling inbox for OTP (timeout={timeout}s)...")

            def _dismiss_modals(pg) -> None:
                for _ in range(8):
                    try:
                        modal = pg.locator('div.modal-two')
                        if modal.count() == 0 or not modal.first.is_visible():
                            break
                    except Exception:
                        break
                    dismissed = False
                    for sel in [
                        'button[data-testid="modal-close-button"]',
                        'button[aria-label="Close"]',
                        'button:has-text("Skip")',
                        'button:has-text("Maybe later")',
                        'button:has-text("Got it")',
                    ]:
                        try:
                            btn = pg.locator(sel)
                            if btn.count() > 0 and btn.first.is_visible():
                                btn.first.click()
                                time.sleep(0.8)
                                dismissed = True
                                break
                        except Exception:
                            pass
                    if not dismissed:
                        try:
                            pg.keyboard.press("Escape")
                            time.sleep(0.5)
                        except Exception:
                            break

            def _scan_folder_for_otp(folder_url: str) -> str:
                """Scan folder and return first OTP code found, or empty string."""
                try:
                    page.goto(folder_url, timeout=20000)
                    try:
                        page.wait_for_load_state("networkidle", timeout=10000)
                    except Exception:
                        pass
                    _wait(2, log)
                except Exception as e:
                    log(f"navigate warn: {e}")
                    return ""

                _dismiss_modals(page)

                try:
                    row_els = page.locator(ROW_SEL).all()
                    log(f"  found {len(row_els)} rows in {page.url}")
                    row_ids = []
                    for el in row_els:
                        try:
                            rid = el.get_attribute("data-element-id", timeout=3000) or ""
                            row_ids.append(rid)
                        except Exception:
                            row_ids.append("")
                except Exception as e:
                    log(f"  row snapshot warn: {e}")
                    return ""

                for i, rid in enumerate(row_ids):
                    if rid and rid in seen:
                        continue
                    try:
                        rows_now = page.locator(ROW_SEL).all()
                        if i >= len(rows_now):
                            break
                        row = rows_now[i]
                        try:
                            row_text = row.inner_text(timeout=5000)
                        except Exception:
                            row_text = f"row[{i}]"
                        log(f"  row[{i}]: {row_text[:120]!r}")

                        # Check row preview text first (subject/preview may contain OTP)
                        for pattern in [r'\b(\d{6})\b', r'\b(\d{8})\b']:
                            m = re.search(pattern, row_text)
                            if m:
                                code = m.group(1)
                                log(f"  OTP in row preview: {code}")
                                return code

                        _dismiss_modals(page)
                        try:
                            row.click(timeout=8000)
                        except Exception:
                            row.click(force=True)
                        _wait(8, log)  # longer wait for email content to decrypt+render
                        _dismiss_modals(page)

                        # Extract text from all frames and email content areas
                        full_text = ""

                        # Method 1: Proton Mail email content selectors (main frame)
                        for sel in [
                            '[data-testid="message-body-content"]',
                            '[class*="messageBody"]',
                            '[class*="message-body"]',
                            '[class*="MessageBody"]',
                            '.message',
                            # Proton Mail uses an iframe for email body
                            'iframe[title]',
                            'iframe[class*="mail"]',
                        ]:
                            try:
                                els = page.locator(sel).all()
                                for el in els[:3]:
                                    t = el.inner_text(timeout=3000)
                                    if t and len(t) > 10:
                                        full_text += t + "\n"
                            except Exception:
                                pass

                        # Method 2: All child frames via JS evaluate
                        for frame in page.frames:
                            try:
                                t = frame.evaluate(
                                    "() => document.body ? document.body.innerText : ''"
                                )
                                # Skip the main Proton Mail app navigation (too long, starts with "Proton Mail")
                                if t and not t.startswith("Proton Mail\n"):
                                    full_text += t + "\n"
                            except Exception:
                                pass

                        # Method 3: Page inner text (full, including navigation)
                        # as last resort — search everything
                        if not full_text.strip():
                            try:
                                full_text = page.inner_text("body")
                            except Exception:
                                pass

                        log(f"  email full text [{len(full_text)} chars]: {full_text[:400]!r}")

                        # Look for OTP codes: 6-digit first, then 8-digit
                        # Exclude 4-digit patterns — too many false positives (years, etc.)
                        for pattern in [r'\b(\d{6})\b', r'\b(\d{8})\b']:
                            m = re.search(pattern, full_text)
                            if m:
                                code = m.group(1)
                                log(f"  OTP found: {code}")
                                return code

                        if rid:
                            seen.add(rid)
                        page.go_back()
                        _wait(2, log)
                        _dismiss_modals(page)
                    except Exception as e:
                        log(f"  row[{i}] warn: {e}")
                        try:
                            page.goto(folder_url, timeout=10000)
                            _wait(2, log)
                        except Exception:
                            pass
                return ""

            FOLDERS = [
                "https://mail.proton.me/u/0/inbox",
                "https://mail.proton.me/u/0/spam",
                "https://mail.proton.me/u/0/newsletters",
                "https://mail.proton.me/u/0/all-mail",
            ]

            while time.time() < deadline:
                for folder in FOLDERS:
                    result = _scan_folder_for_otp(folder)
                    if result:
                        return result
                    if time.time() >= deadline:
                        break
                _wait(10, log, "waiting for OTP email")

            raise TimeoutError(f"No OTP received within {timeout}s")

        finally:
            ctx.close()
