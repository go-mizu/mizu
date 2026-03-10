"""Patchright-based ClickHouse Cloud account registration.

Flow:
  1. Open console.clickhouse.cloud/signUp?with=email
  2. Fill email + password → submit
  3. If email verification required: poll mail.tm for verification link
  4. Click through onboarding prompts
  5. Navigate to Settings → API Keys → create API key
  6. Extract key_id + key_secret
  7. Use REST API to get org_id
"""
from __future__ import annotations

import os
import platform
import re
import sys
import tempfile
import time

from .email import MailTmClient, Mailbox


def _browser_args() -> list[str]:
    args = ["--window-size=1280,900", "--lang=en-US"]
    if platform.system() == "Linux":
        args += ["--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage"]
    return args


def _maybe_reexec_xvfb(headless: bool) -> None:
    if platform.system() != "Linux" or headless or os.environ.get("DISPLAY"):
        return
    import shutil
    import subprocess
    xvfb = shutil.which("xvfb-run")
    if xvfb:
        sys.exit(subprocess.call([xvfb, "-a", sys.executable] + sys.argv))


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


def _clear_and_fill(page, selectors: list[str], text: str, log=None) -> str | None:
    for sel in selectors:
        try:
            inp = page.locator(sel)
            if inp.count() > 0:
                el = inp.first
                el.click()
                time.sleep(0.2)
                el.fill("")
                time.sleep(0.1)
                el.type(text, delay=55)
                time.sleep(0.3)
                if log:
                    log(f"filled (clear+type) via: {sel}")
                return sel
        except Exception:
            continue
    return None


def _log_page_state(page, log, label: str = "", max_chars: int = 500) -> str:
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:400]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""


def _on_console(page) -> bool:
    return "console.clickhouse.cloud" in page.url or "clickhouse.cloud" in page.url


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> dict:
    """Drive ClickHouse Cloud signup, return dict with api_key_id, api_key_secret, org_id."""
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"registering {mailbox.address}")
    user_data = tempfile.mkdtemp(prefix="ch_reg_")

    with sync_playwright() as p:
        ctx = p.chromium.launch_persistent_context(
            user_data_dir=user_data,
            channel="chrome",
            headless=headless,
            args=_browser_args(),
            viewport={"width": 1280, "height": 900},
            locale="en-US",
        )
        page = ctx.pages[0] if ctx.pages else ctx.new_page()

        if verbose:
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            # ---- Step 1: Open signup page ----
            log("opening console.clickhouse.cloud/signUp...")
            try:
                page.goto("https://console.clickhouse.cloud/signUp?with=email", timeout=30000)
                page.wait_for_load_state("networkidle", timeout=15000)
            except Exception as e:
                log(f"signup page warn: {e}")
            _wait(3, log)
            log(f"landed on: {page.url}")
            _log_page_state(page, log, "signup: ")

            # ---- Step 2: Fill signup form ----
            log(f"filling signup form with {mailbox.address}")

            _fill_first(page, [
                'input[name="email"]',
                'input[type="email"]',
                'input[placeholder*="email" i]',
                'input[placeholder*="Email" i]',
            ], mailbox.address, log)

            _wait(0.5, log)

            _fill_first(page, [
                'input[name="password"]',
                'input[type="password"]',
            ], password, log)

            _wait(0.5, log)

            # Check for terms checkbox
            tos = page.locator('input[type="checkbox"]')
            if tos.count() > 0:
                for i in range(tos.count()):
                    try:
                        if not tos.nth(i).is_checked():
                            tos.nth(i).check()
                            log(f"checked checkbox {i}")
                    except Exception:
                        pass
                _wait(0.3, log)

            # Submit
            _click_first(page, [
                'button[type="submit"]',
                'button:has-text("Sign up")',
                'button:has-text("Create account")',
                'button:has-text("Get started")',
                'button:has-text("Continue")',
            ], log)
            _wait(5, log, "waiting for signup response")
            log(f"url after submit: {page.url}")

            # ---- Step 3: Handle email verification ----
            body_text = _log_page_state(page, log, "post-signup: ")

            verify_keywords = ["verify", "check your email", "confirmation",
                               "sent you", "activate your"]
            if any(kw in body_text.lower() for kw in verify_keywords):
                log("email verification required — polling mail.tm...")
                verify_link = mail_client.poll_for_verification_link(mailbox, timeout=120)
                log(f"got verification link: {verify_link[:80]}...")
                try:
                    page.goto(verify_link, timeout=30000)
                except Exception as e:
                    log(f"verification link nav warn: {e}")
                _wait(4, log, "post-verification load")
                log(f"url after verification: {page.url}")

                # May need to log in after verification
                if "signIn" in page.url or "login" in page.url:
                    log("logging in after verification...")
                    _fill_first(page, [
                        'input[name="email"]',
                        'input[type="email"]',
                    ], mailbox.address, log)
                    _fill_first(page, [
                        'input[name="password"]',
                        'input[type="password"]',
                    ], password, log)
                    _click_first(page, [
                        'button[type="submit"]',
                        'button:has-text("Sign in")',
                        'button:has-text("Log in")',
                        'button:has-text("Continue")',
                    ], log)
                    _wait(5, log, "waiting for login")
                    log(f"url after login: {page.url}")

            # ---- Step 4: Click through onboarding ----
            log("handling onboarding...")
            _skip_onboarding(page, log, max_attempts=15)
            log(f"url after onboarding: {page.url}")

            # ---- Step 5: Navigate to API Keys ----
            log("navigating to API keys settings...")
            try:
                page.goto("https://console.clickhouse.cloud/settings/api-keys", timeout=20000)
                page.wait_for_load_state("networkidle", timeout=10000)
            except Exception as e:
                log(f"settings nav warn: {e}")
            _wait(3, log, "settings load")
            log(f"url: {page.url}")
            _log_page_state(page, log, "api-keys-page: ")

            # ---- Step 6: Create API key ----
            log("creating API key...")
            _click_first(page, [
                'button:has-text("New API key")',
                'button:has-text("Create API key")',
                'button:has-text("Generate")',
                'button:has-text("+ New")',
                'button:has-text("Add")',
            ], log)
            _wait(2, log, "api key dialog")

            # Fill key name if dialog appears
            name_input = page.locator('input[placeholder*="name" i], input[name*="name" i]')
            if name_input.count() > 0:
                log("filling API key name...")
                name_input.first.fill("automation-key")
                _wait(0.5, log)

            # Submit key creation
            _click_first(page, [
                'button:has-text("Generate")',
                'button:has-text("Create")',
                'button:has-text("Save")',
                'button[type="submit"]',
            ], log)
            _wait(3, log, "key generation")
            _log_page_state(page, log, "after-key-create: ", max_chars=800)

            # ---- Step 7: Extract API key credentials ----
            log("extracting API key credentials...")
            key_id, key_secret = _extract_api_keys(page, log)

            if not key_id or not key_secret:
                _log_page_state(page, log, "key-fail: ", max_chars=800)
                raise RuntimeError("Failed to extract API key credentials from settings page")

            log(f"API key extracted: id={key_id[:10]}...")

            # ---- Step 8: Get org_id via REST API ----
            log("fetching organization ID via API...")
            org_id = ""
            try:
                import httpx
                resp = httpx.get(
                    "https://api.clickhouse.cloud/v1/organizations",
                    auth=(key_id, key_secret),
                    timeout=15,
                )
                resp.raise_for_status()
                orgs = resp.json().get("result", [])
                if orgs:
                    org_id = orgs[0].get("id", "")
                    log(f"org_id: {org_id}")
            except Exception as e:
                log(f"org_id fetch warn: {e}")

            return {
                "api_key_id": key_id,
                "api_key_secret": key_secret,
                "org_id": org_id,
            }

        finally:
            ctx.close()


def _skip_onboarding(page, log, max_attempts: int = 15) -> None:
    """Click through ClickHouse Cloud onboarding prompts."""
    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url

        # Done if on main console pages
        if any(x in url for x in ["/services", "/settings", "/sql-console",
                                    "/query", "/dashboards"]):
            log(f"onboarding done at attempt {attempt}")
            return

        # Try selecting options on survey/setup pages
        for option_text in ["Personal project", "Evaluation", "Software engineer",
                            "Data engineer", "Other", "Skip"]:
            opt = page.locator(f'text="{option_text}"')
            if opt.count() > 0:
                try:
                    opt.first.click()
                    log(f"  selected: {option_text}")
                    time.sleep(0.5)
                    break
                except Exception:
                    pass

        # Fill any visible text inputs (org name, etc.)
        inputs = page.locator('input[type="text"]:visible')
        if inputs.count() > 0:
            for i in range(inputs.count()):
                inp = inputs.nth(i)
                if not inp.input_value():
                    try:
                        inp.fill("automation-org")
                        log(f"  filled input {i}")
                    except Exception:
                        pass

        # Select region if dropdown present
        region_trigger = page.locator('text="Select a region"')
        if region_trigger.count() > 0:
            region_trigger.first.click()
            time.sleep(1)
            _click_first(page, [
                'text="US East"',
                'text="us-east-1"',
                '[data-value*="us-east"]',
                'li:first-child',
            ], log)
            time.sleep(0.5)

        # Click advancement buttons
        clicked = False
        for sel in [
            'button:has-text("Skip")',
            'button:has-text("Next")',
            'button:has-text("Get started")',
            'button:has-text("Done")',
            'button:has-text("Launch")',
            'button:has-text("Continue"):not([disabled])',
            'button:has-text("Create"):not([disabled])',
        ]:
            btn = page.locator(sel)
            if btn.count() > 0:
                try:
                    log(f"  onboarding click: {sel}")
                    btn.first.click(timeout=5000)
                    clicked = True
                    break
                except Exception:
                    continue

        if not clicked:
            log(f"  no onboarding button at attempt {attempt}, url={url}")
            break


def _extract_api_keys(page, log) -> tuple[str, str]:
    """Try to extract API key ID and secret from the page."""

    # Strategy 1: Look for displayed key values in code/pre/input elements
    body = ""
    try:
        body = page.inner_text("body")
    except Exception:
        pass

    # ClickHouse API keys look like UUIDs or long alphanumeric strings
    key_pattern = re.compile(r"[a-zA-Z0-9]{20,}")

    # Look for labeled key fields
    for label in ["Key ID", "key_id", "API Key ID"]:
        el = page.locator(f'text="{label}"')
        if el.count() > 0:
            # Try to get the next sibling or nearby code/input
            parent = el.first
            try:
                nearby = parent.locator(".. >> code, .. >> input[readonly], .. >> span")
                if nearby.count() > 0:
                    text = nearby.first.inner_text() or nearby.first.get_attribute("value") or ""
                    if len(text) > 10:
                        log(f"found key_id near label: {text[:20]}...")
            except Exception:
                pass

    # Strategy 2: Look for copyable/readonly inputs or code blocks with key-like values
    key_id = ""
    key_secret = ""

    for sel in ['code', 'input[readonly]', 'pre', '[data-testid*="key"]',
                '.api-key', 'span.key']:
        try:
            els = page.locator(sel)
            for i in range(min(els.count(), 10)):
                el = els.nth(i)
                text = el.inner_text() or el.get_attribute("value") or ""
                text = text.strip()
                if len(text) > 15 and "\n" not in text and " " not in text:
                    if not key_id:
                        key_id = text
                        log(f"key candidate 1 via {sel}: {text[:20]}...")
                    elif text != key_id and not key_secret:
                        key_secret = text
                        log(f"key candidate 2 via {sel}: {text[:20]}...")
                        break
            if key_id and key_secret:
                break
        except Exception:
            pass

    # Strategy 3: Scan body text for patterns
    if not key_id or not key_secret:
        matches = key_pattern.findall(body)
        candidates = [m for m in matches if len(m) > 20]
        if len(candidates) >= 2 and not key_id:
            key_id = candidates[0]
            key_secret = candidates[1]
            log(f"keys via body scan: id={key_id[:20]}... secret={key_secret[:20]}...")
        elif len(candidates) == 1 and not key_id:
            key_id = candidates[0]
            log(f"partial key via body: id={key_id[:20]}...")

    return key_id, key_secret
