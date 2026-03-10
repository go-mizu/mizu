"""Patchright-based ClickHouse Cloud account registration.

Flow:
  1. Open console.clickhouse.cloud/signUp?with=email
  2. Fill email → submit → fill password → submit
  3. If email verification required: poll mail.tm for verification link
  4. Log in after verification
  5. Onboarding creates a service → capture host from service page
  6. Navigate to service connection tab → extract host + reset password
  7. Return {service_id, host, port, db_password}
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


def _log_page_state(page, log, label: str = "", max_chars: int = 500) -> str:
    try:
        body = page.inner_text("body")[:max_chars]
        log(f"{label}url={page.url}")
        log(f"{label}text={body[:400]!r}")
        return body
    except Exception as e:
        log(f"{label}page read error: {e}")
        return ""


def _do_auth0_login(page, email: str, password: str, log) -> None:
    """Handle Auth0 identifier-first login flow (email page → password page)."""
    try:
        page.wait_for_load_state("networkidle", timeout=10000)
    except Exception:
        pass
    _wait(2, log, "login page load")

    # Fill email - Auth0 login uses input[name="username"] type="text"
    filled = False
    for sel in ['input[name="username"]', 'input[name="email"]',
                'input[type="email"]', 'input[type="text"]']:
        try:
            inp = page.locator(sel)
            if inp.count() > 0:
                inp.first.fill(email)
                log(f"login: filled email via {sel}")
                filled = True
                break
        except Exception:
            continue

    if not filled:
        log("login: FAILED to fill email")
        return

    _wait(0.5, log)
    _click_first(page, ['button[type="submit"]', 'button:has-text("Continue")'], log)
    _wait(3, log, "waiting for password page")

    # Fill password
    try:
        page.wait_for_selector('input[type="password"]', state="visible", timeout=10000)
    except Exception as e:
        log(f"login: password input wait failed: {e}")
        return

    page.locator('input[type="password"]').first.fill(password)
    log("login: filled password")
    _wait(0.5, log)

    _click_first(page, [
        'button[type="submit"]', 'button:has-text("Continue")',
        'button:has-text("Sign in")', 'button:has-text("Log in")',
    ], log)
    _wait(5, log, "waiting for login completion")


def _search_json_for_credentials(obj, captured: dict, log) -> None:
    """Recursively search JSON for password and host fields."""
    if isinstance(obj, dict):
        for key, val in obj.items():
            if key == "password" and isinstance(val, str) and len(val) > 8:
                captured["password"] = val
                log(f"  [net] CAPTURED password: {val[:10]}...")
            elif key == "host" and isinstance(val, str) and "clickhouse.cloud" in val:
                captured["host"] = val
                log(f"  [net] CAPTURED host: {val}")
            elif key == "id" and isinstance(val, str) and len(val) == 36 and "-" in val:
                captured["service_id"] = val
            elif isinstance(val, (dict, list)):
                _search_json_for_credentials(val, captured, log)
    elif isinstance(obj, list):
        for item in obj:
            if isinstance(item, (dict, list)):
                _search_json_for_credentials(item, captured, log)


def register_via_browser(
    mailbox: Mailbox,
    mail_client: MailTmClient,
    password: str,
    headless: bool = True,
    verbose: bool = False,
) -> dict:
    """Drive ClickHouse Cloud signup, return dict with service connection info."""
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

            # ---- Step 2: Fill email (identifier page) ----
            log(f"filling signup form with {mailbox.address}")
            _fill_first(page, [
                'input[name="email"]', 'input[type="email"]',
                'input[placeholder*="email" i]',
            ], mailbox.address, log)
            _wait(0.5, log)

            _click_first(page, [
                'button[type="submit"]', 'button:has-text("Continue")',
                'button:has-text("Sign up")',
            ], log)
            _wait(3, log, "waiting for password page")
            log(f"url after email submit: {page.url}")

            # ---- Step 2b: Fill password (separate page) ----
            if "password" in page.url or page.locator('input[type="password"]').count() > 0:
                log("on password page, filling password...")
                _fill_first(page, [
                    'input[name="password"]', 'input[type="password"]',
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

                _click_first(page, [
                    'button[type="submit"]', 'button:has-text("Sign up")',
                    'button:has-text("Create account")', 'button:has-text("Continue")',
                ], log)
                _wait(5, log, "waiting for signup completion")
                log(f"url after password submit: {page.url}")

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

            # If redirected to login after verification, log in
            if "login" in page.url or "signIn" in page.url:
                log("logging in after verification...")
                _do_auth0_login(page, mailbox.address, password, log)
                log(f"url after login: {page.url}")

            # ---- Step 4: Onboarding — let it create a service ----
            log("handling onboarding...")
            _wait(5, log, "console load after login")
            _log_page_state(page, log, "pre-onboard: ")

            # Set up network interceptor to capture password from API response
            captured = {"password": "", "service_id": "", "host": ""}

            def _on_response(response):
                try:
                    url = response.url
                    if response.status < 200 or response.status >= 300:
                        return
                    ct = response.headers.get("content-type", "")
                    if "json" not in ct:
                        return
                    # Log ALL control-plane-internal responses
                    is_internal = "control-plane-internal" in url
                    text = response.text()
                    if is_internal:
                        log(f"  [net-api] {url.split('?')[-1][:40]} → {text[:300]}")
                    elif "clickhouse.cloud" in text or "password" in text.lower():
                        log(f"  [net] {url[:80]} → {text[:200]}")
                    try:
                        body = response.json()
                        _search_json_for_credentials(body, captured, log)
                    except Exception:
                        pass
                except Exception:
                    pass

            page.on("response", _on_response)

            service_id, db_password = _handle_onboarding(page, log)

            # Use network-captured values if onboarding didn't get them
            if not db_password and captured["password"]:
                db_password = captured["password"]
                log(f"using network-captured password: {db_password[:10]}...")
            if not service_id and captured["service_id"]:
                service_id = captured["service_id"]
                log(f"using network-captured service_id: {service_id}")
            log(f"url after onboarding: {page.url}")
            log(f"service_id from onboarding: {service_id}")
            if db_password:
                log(f"password captured during onboarding: {db_password[:10]}...")

            # ---- Step 5: Wait for service provisioning + extract credentials ----
            host = captured.get("host", "")
            if host:
                log(f"using network-captured host: {host}")
            port = 8443

            # Extract service_id from URL if not found during onboarding
            if not service_id:
                svc_match = re.search(r'/services/([a-f0-9-]{36})', page.url)
                if svc_match:
                    service_id = svc_match.group(1)
                    log(f"service_id from URL: {service_id}")

            if service_id:
                # Wait for service to finish provisioning (skip if already have host)
                if not host:
                    host = _wait_for_service_ready(page, service_id, log, timeout=180)

                # Check network-captured host after page navigation
                if not host and captured.get("host"):
                    host = captured["host"]
                    log(f"using network-captured host (post-ready): {host}")

                # Reset password only if we didn't capture it during onboarding
                if host and not db_password:
                    db_password = _reset_service_password(page, service_id, log)

                # Final check for network-captured password
                if not db_password and captured.get("password"):
                    db_password = captured["password"]
                    log(f"using network-captured password (final): {db_password[:10]}...")

            return {
                "service_id": service_id,
                "host": host,
                "port": port,
                "db_password": db_password,
            }

        finally:
            ctx.close()


def _handle_onboarding(page, log, max_attempts: int = 25) -> tuple[str, str]:
    """Handle ClickHouse Cloud onboarding. Returns (service_id, db_password)."""
    service_id = ""
    db_password = ""
    no_progress_count = 0

    for attempt in range(max_attempts):
        time.sleep(2)
        url = page.url

        # Extract service_id from URL at any point
        svc_match = re.search(r'/services/([a-f0-9-]{36})', url)
        if svc_match:
            service_id = svc_match.group(1)

        # Done if on main console pages (not onboard)
        if any(x in url for x in ["/services", "/settings", "/sql-console"]):
            if "/onboard" not in url:
                log(f"onboarding done at attempt {attempt}")
                return service_id, db_password

        body = ""
        try:
            body = page.inner_text("body")[:600]
        except Exception:
            pass
        log(f"  onboard[{attempt}] url={url}")
        log(f"  onboard[{attempt}] text={body[:200]!r}")

        clicked = False

        # On the "Personalize your experience" page: click a use case card
        if "onboard" in url and "Personalize" in body:
            for card_text in ["Data warehousing", "Real-time analytics",
                              "Observability", "Machine learning"]:
                card = page.locator(f'text="{card_text}"')
                if card.count() > 0:
                    try:
                        card.first.click()
                        log(f"  selected use case: {card_text}")
                        clicked = True
                        _wait(2, log, "use case selected")
                        break
                    except Exception:
                        pass
            if clicked:
                continue

        # On the service creation page: select AWS + create
        if "onboard" in url and ("Configure your cloud service" in body
                                  or "Create service" in body):
            # Prefer AWS as provider (most reliable for free tier)
            if "AWS" not in body[:200] or "Azure" in body[:200] or "GCP" in body[:200]:
                aws_btn = page.locator('text="AWS"')
                if aws_btn.count() > 0:
                    try:
                        aws_btn.first.click()
                        log("  switched to AWS provider")
                        _wait(1, log)
                    except Exception:
                        pass

            create_btn = page.locator('button:has-text("Create service"):not([disabled])')
            if create_btn.count() > 0:
                create_btn.first.click()
                log("  clicked Create service")
                clicked = True
            elif _click_first(page, ['button:has-text("Create"):not([disabled])'], log):
                clicked = True

            if clicked:
                _wait(5, log, "service creation + credentials popup")
                pw = _check_credentials_popup(page, log)
                if pw:
                    db_password = pw
                else:
                    # Poll a few more times — popup may take a moment
                    for _ in range(3):
                        _wait(3, log, "waiting for credentials popup")
                        pw = _check_credentials_popup(page, log)
                        if pw:
                            db_password = pw
                            break
                # Check if still on onboard page (creation may have failed)
                if "onboard" in page.url:
                    check_body = page.inner_text("body")[:500]
                    if "Configure" in check_body or "Create service" in check_body:
                        log("  service creation may have failed, retrying...")
                        no_progress_count = 0  # Reset stall counter
                continue

        # Try clicking survey/onboarding options
        for option_text in ["Other", "Personal project", "Evaluation",
                            "Software engineer", "Data engineer"]:
            opt = page.locator(f'text="{option_text}"')
            if opt.count() > 0:
                try:
                    opt.first.click()
                    log(f"  selected: {option_text}")
                    clicked = True
                    time.sleep(0.5)
                    break
                except Exception:
                    pass

        # Click advancement buttons
        for sel in [
            'button:has-text("Skip")',
            'a:has-text("Skip")',
            'button:has-text("Next")',
            'button:has-text("Get started")',
            'button:has-text("Done")',
            'button:has-text("Launch")',
            'button:has-text("Continue"):not([disabled])',
            'button:has-text("Start"):not([disabled])',
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
            no_progress_count += 1
            log(f"  no action at attempt {attempt} (stall={no_progress_count})")
            if no_progress_count >= 5:
                log("  giving up on onboarding after 5 stalls")
                break
        else:
            no_progress_count = 0

    return service_id, db_password


def _check_credentials_popup(page, log) -> str:
    """Check for a credentials popup after service creation. Returns password if found."""
    try:
        body = page.inner_text("body")[:2000]
        log(f"credentials check: {body[:300]!r}")

        # Always search for password, even without explicit "password" keyword
        # The popup may show credentials in various formats

        # Search code/readonly/pre elements for password-like values
        for sel in ['code', 'input[readonly]', 'pre', '[data-testid*="password"]',
                    'input[type="text"][readonly]', '[data-testid*="credential"]']:
            els = page.locator(sel)
            for i in range(min(els.count(), 10)):
                text = (els.nth(i).inner_text() or
                        els.nth(i).get_attribute("value") or "").strip()
                if len(text) > 8 and " " not in text and "\n" not in text:
                    log(f"  credential found: {text[:15]}...")
                    return text

        # Search HTML for password near relevant context
        html = page.evaluate("document.documentElement.innerHTML")
        # Look for values in elements after "password" labels
        pw_patterns = re.findall(
            r'(?:password|Password|credential)[^>]*?>([A-Za-z0-9!@#$%^&*_+=-]{12,64})<',
            html
        )
        if pw_patterns:
            log(f"  password from HTML: {pw_patterns[0][:15]}...")
            return pw_patterns[0]

        # Look for copy-button adjacent values
        copy_els = page.locator('[data-testid*="copy"], [data-clipboard], button[aria-label*="Copy"]')
        for i in range(min(copy_els.count(), 5)):
            try:
                # Check sibling/parent text
                parent = copy_els.nth(i).locator("..")
                text = parent.inner_text().strip()
                # Extract the longest non-space token
                tokens = [t for t in text.split() if len(t) > 8]
                for t in tokens:
                    if re.match(r'^[A-Za-z0-9!@#$%^&*_+=-]+$', t) and len(t) >= 12:
                        log(f"  password from copy-btn: {t[:15]}...")
                        return t
            except Exception:
                pass

    except Exception as e:
        log(f"credentials popup check error: {e}")
    return ""


def _wait_for_service_ready(page, service_id: str, log, timeout: int = 180) -> str:
    """Poll the service page until the service is running. Returns host."""
    # Use the service overview page — it often contains the host in the HTML
    overview_url = f"https://console.clickhouse.cloud/services/{service_id}"
    log(f"waiting for service {service_id[:12]}... to be ready (timeout={timeout}s)")

    start = time.time()
    while time.time() - start < timeout:
        try:
            page.goto(overview_url, timeout=15000)
            page.wait_for_load_state("networkidle", timeout=10000)
        except Exception:
            pass
        _wait(3, log)

        body = ""
        try:
            body = page.inner_text("body")[:1000]
        except Exception:
            pass

        elapsed = int(time.time() - start)
        body_lower = body.lower()
        log(f"  service check ({elapsed}s): {body[:150]!r}")

        # Still provisioning — wait
        if "provisioning" in body_lower:
            log("  still provisioning...")
            _wait(10, log, "polling interval")
            continue

        # Service is ready: either explicit status or SQL console loaded
        # (console transitions from "Provisioning..." to SQL editor once ready)
        is_ready = (
            "running" in body_lower
            or "idle" in body_lower
            or "sql" in body_lower  # SQL console appeared
            or "tables" in body_lower  # Tables view appeared
            or "queries" in body_lower  # Queries view appeared
        )

        if is_ready:
            log("service is ready!")

            # Method 1: Extract from overview page HTML (may have host)
            host = _extract_host_from_html(page, log)
            if host:
                log(f"host from overview page: {host}")
                return host

            # Method 2: Navigate to connect page via direct URL
            host = _get_host_from_connect_page(page, service_id, log)
            if host:
                return host

            log("host not found, but service is running")
            return ""

        _wait(10, log, "polling interval")

    log(f"service did not become ready within {timeout}s")
    return ""


def _get_host_via_internal_api(page, log) -> str:
    """Extract host from network responses captured during page navigation."""
    # The internal API (control-plane-internal.clickhouse.cloud) can't be called
    # directly — it requires console-session cookies. Instead, we rely on the
    # network interceptor that captures responses as the console makes its own calls.
    # This function is a no-op; host extraction happens via HTML parsing.
    return ""


def _reset_service_password(page, service_id: str, log) -> str:
    """Attempt to capture the service password.

    Note: ClickHouse Cloud's internal API does not expose the password
    through any accessible endpoint. The password is shown once during
    service creation in the console but cannot be reliably captured
    via browser automation.
    """
    # Try network-intercepted password (if captured during service creation)
    # The password would have been captured by the response listener

    # Try clicking "Reset password" and watching for network response
    settings_url = f"https://console.clickhouse.cloud/services/{service_id}/settings"
    try:
        page.goto(settings_url, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
    except Exception:
        pass
    _wait(3, log, "settings page")

    captured_pw = {"value": ""}

    def _on_reset_response(response):
        try:
            ct = response.headers.get("content-type", "")
            if "json" not in ct:
                return
            text = response.text()
            if "password" in text.lower():
                log(f"  [reset-net] {response.url[:60]} → {text[:200]}")
                pw_match = re.search(r'"password"\s*:\s*"([^"]+)"', text)
                if pw_match and len(pw_match.group(1)) > 8:
                    captured_pw["value"] = pw_match.group(1)
        except Exception:
            pass

    page.on("response", _on_reset_response)

    clicked = _click_first(page, ['button:has-text("Reset password")'], log)
    if clicked:
        _wait(5, log, "password reset response")
        if captured_pw["value"]:
            log(f"password captured from reset: {captured_pw['value'][:10]}...")
            return captured_pw["value"]

    log("password not captured — user must get it from console")
    return ""


def _extract_host_from_html(page, log) -> str:
    """Search the full HTML source for a ClickHouse host."""
    try:
        html = page.evaluate("document.documentElement.innerHTML")
        host = _extract_host_from_text(html, log)
        if host:
            return host
    except Exception as e:
        log(f"HTML extraction error: {e}")
    return ""


def _get_host_from_connect_page(page, service_id: str, log) -> str:
    """Navigate to the Connect page and extract the host from connection details."""
    connect_url = f"https://console.clickhouse.cloud/services/{service_id}/connect"
    log(f"navigating to connect page: {connect_url}")

    try:
        page.goto(connect_url, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
    except Exception as e:
        log(f"connect page navigation warn: {e}")
    _wait(3, log, "connect page load")

    # Dismiss any survey/modal popup
    _click_first(page, [
        'button[aria-label="Close"]',
        'button[aria-label="close"]',
        '[data-testid="close-button"]',
        'button:has-text("×")',
        'button:has-text("Close")',
    ], log)
    _wait(1, log)

    # Log page state for debugging
    _log_page_state(page, log, "connect-page: ", max_chars=800)

    # Try to expand connection details by clicking tabs/buttons
    _click_first(page, [
        '[data-testid*="connect"]',
        'button:has-text("Native")',
        'button:has-text("HTTPS")',
    ], log)
    _wait(2, log, "connection tab expand")

    # Try extracting from HTML
    host = _extract_host_from_html(page, log)
    if host:
        log(f"host from connect page HTML: {host}")
        return host

    # Try visible text
    try:
        body = page.inner_text("body")[:3000]
        host = _extract_host_from_text(body, log)
        if host:
            log(f"host from connect page text: {host}")
            return host
    except Exception as e:
        log(f"connect page text extraction error: {e}")

    # Retry once after waiting (page JS may still be rendering)
    log("host not found on connect page, retrying after 5s...")
    _wait(5, log, "connect page retry")
    host = _extract_host_from_html(page, log)
    if host:
        log(f"host from connect page HTML (retry): {host}")
        return host

    log("host NOT found on connect page after retry")
    return ""


def _extract_host_from_text(text: str, log) -> str:
    """Extract ClickHouse Cloud service hostname from text."""
    # Exclude known non-service hosts
    exclude = {"console.clickhouse.cloud", "auth.clickhouse.cloud",
               "api.clickhouse.cloud", "statuspage.clickhouse.cloud",
               "console-api-internal.clickhouse.cloud",
               "console-api.clickhouse.cloud",
               "control-plane-internal.clickhouse.cloud"}

    # ClickHouse Cloud service hosts: xxx.region.provider.clickhouse.cloud
    for match in re.finditer(r'([a-z0-9-]+\.[a-z0-9-]+\.(?:aws|gcp|azure)\.clickhouse\.cloud)', text):
        host = match.group(1)
        if host not in exclude:
            log(f"found host: {host}")
            return host

    # Broader: any subdomain of clickhouse.cloud (but not console/auth/api)
    for match in re.finditer(r'([a-z0-9][a-z0-9-]+\.clickhouse\.cloud)', text):
        host = match.group(1)
        if host not in exclude:
            log(f"found host (broad): {host}")
            return host

    return ""


