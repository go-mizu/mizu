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
            service_id = _handle_onboarding(page, log)
            log(f"url after onboarding: {page.url}")
            log(f"service_id from onboarding: {service_id}")

            # ---- Step 5: Wait for service provisioning + extract credentials ----
            host = ""
            port = 8443
            db_password = ""

            # Extract service_id from URL if not found during onboarding
            if not service_id:
                svc_match = re.search(r'/services/([a-f0-9-]{36})', page.url)
                if svc_match:
                    service_id = svc_match.group(1)
                    log(f"service_id from URL: {service_id}")

            if service_id:
                # Wait for service to finish provisioning
                host = _wait_for_service_ready(page, service_id, log, timeout=180)

                # Reset password to get known credentials
                if host:
                    db_password = _reset_service_password(page, service_id, log)

            return {
                "service_id": service_id,
                "host": host,
                "port": port,
                "db_password": db_password,
            }

        finally:
            ctx.close()


def _handle_onboarding(page, log, max_attempts: int = 25) -> str:
    """Handle ClickHouse Cloud onboarding. Returns service_id if a service is created."""
    service_id = ""
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
                return service_id

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

        # On the service creation page: just let it create (default settings are fine)
        if "onboard" in url and ("Configure your cloud service" in body
                                  or "Create service" in body):
            if _click_first(page, ['button:has-text("Create service")'], log):
                clicked = True
            elif _click_first(page, ['button:has-text("Create"):not([disabled])'], log):
                clicked = True

            if clicked:
                _wait(3, log, "service creation")
                _check_credentials_popup(page, log)
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
            if no_progress_count >= 3:
                log("  giving up on onboarding after 3 stalls")
                break
        else:
            no_progress_count = 0

    return service_id


def _check_credentials_popup(page, log) -> dict:
    """Check for a credentials popup after service creation."""
    try:
        body = page.inner_text("body")[:1000]
        if "password" in body.lower() and ("copy" in body.lower() or "credential" in body.lower()):
            log(f"credentials popup detected: {body[:300]!r}")
            # Try to find password in code/input elements
            for sel in ['code', 'input[readonly]', 'pre', '[data-testid*="password"]']:
                els = page.locator(sel)
                for i in range(min(els.count(), 10)):
                    text = els.nth(i).inner_text() or els.nth(i).get_attribute("value") or ""
                    text = text.strip()
                    if len(text) > 8 and " " not in text:
                        log(f"  credential value: {text[:20]}...")
    except Exception:
        pass
    return {}


def _wait_for_service_ready(page, service_id: str, log, timeout: int = 180) -> str:
    """Poll the service health page until the service is running. Returns host."""
    health_url = f"https://console.clickhouse.cloud/services/{service_id}/health"
    log(f"waiting for service {service_id[:12]}... to be ready (timeout={timeout}s)")

    start = time.time()
    while time.time() - start < timeout:
        try:
            page.goto(health_url, timeout=15000)
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
        log(f"  health check ({elapsed}s): {body[:150]!r}")

        # Check if service is running
        if "running" in body.lower() or "idle" in body.lower():
            log("service is ready!")

            # Try internal API via browser cookies (most reliable)
            host = _get_host_via_api(page, service_id, log)
            if host:
                return host

            # Try extracting host from HTML source
            host = _extract_host_from_html(page, log)
            if host:
                return host

            # Try the connect page with survey dismissal
            host = _get_host_from_connect_page(page, service_id, log)
            if host:
                return host

            log("host not found, but service is running")
            return ""

        if "provisioning" in body.lower():
            log("  still provisioning...")
            _wait(10, log, "polling interval")
            continue

        _wait(10, log, "polling interval")

    log(f"service did not become ready within {timeout}s")
    return ""


def _reset_service_password(page, service_id: str, log) -> str:
    """Navigate to service settings and reset the password."""
    settings_url = f"https://console.clickhouse.cloud/services/{service_id}/settings"
    try:
        page.goto(settings_url, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
    except Exception:
        pass
    _wait(3, log, "settings page for password reset")

    # Click "Reset password"
    clicked = _click_first(page, [
        'button:has-text("Reset password")',
        'a:has-text("Reset password")',
    ], log)

    if not clicked:
        log("Reset password button not found")
        return ""

    _wait(2, log, "password reset dialog")

    # Look for a modal/dialog that appeared
    body = ""
    try:
        body = page.inner_text("body")[:2000]
    except Exception:
        pass
    log(f"reset-dialog: {body[:400]!r}")

    # Look for confirmation dialog — check for modal/dialog elements
    dialog = page.locator('[role="dialog"], [role="alertdialog"], .modal, [data-testid*="modal"]')
    if dialog.count() > 0:
        dialog_text = dialog.first.inner_text()[:500]
        log(f"dialog found: {dialog_text[:200]!r}")

    # Click confirmation buttons — might need multiple clicks through dialog steps
    for attempt in range(3):
        for sel in [
            '[role="dialog"] button:has-text("Generate")',
            '[role="dialog"] button:has-text("Reset")',
            '[role="dialog"] button:has-text("Confirm")',
            'button:has-text("Generate new password")',
            'button:has-text("Reset password")',
            'button:has-text("Confirm")',
            'button:has-text("Yes, reset")',
            'button:has-text("Yes")',
        ]:
            try:
                btn = page.locator(sel)
                if btn.count() > 0:
                    btn.first.click()
                    log(f"reset confirm[{attempt}]: clicked {sel}")
                    _wait(3, log, "password dialog step")
                    break
            except Exception:
                continue

        # Check if password is now visible
        try:
            body = page.inner_text("body")[:2000]
        except Exception:
            body = ""
        log(f"after-reset[{attempt}]: {body[:400]!r}")

        # Check dialog content specifically
        dialog = page.locator('[role="dialog"], [role="alertdialog"], .modal')
        if dialog.count() > 0:
            try:
                dialog_text = dialog.first.inner_text()[:500]
                log(f"dialog[{attempt}]: {dialog_text[:300]!r}")
            except Exception:
                pass

        # Search for password in HTML (might be in data attributes or hidden elements)
        try:
            html = page.evaluate("document.documentElement.innerHTML")
            # Look for password-like strings near "password" context
            pw_patterns = re.findall(
                r'(?:password|Password|credential)[^>]*?>([A-Za-z0-9!@#$%^&*_-]{12,64})<',
                html
            )
            if pw_patterns:
                log(f"password from HTML: {pw_patterns[0][:10]}...")
                return pw_patterns[0]

            # Look for copy-able elements with long alphanumeric values
            copy_els = page.locator('[data-testid*="copy"], [data-clipboard], .copy-text, code')
            for i in range(min(copy_els.count(), 10)):
                text = copy_els.nth(i).inner_text().strip()
                if len(text) > 8 and " " not in text and "\n" not in text:
                    log(f"copy-able value: {text[:15]}...")
                    return text
        except Exception as e:
            log(f"HTML password search error: {e}")

    # Look for password in code/readonly elements
    for sel in ['code', 'input[readonly]', 'pre', '[data-testid*="password"]',
                'input[type="text"][readonly]']:
        try:
            els = page.locator(sel)
            for i in range(min(els.count(), 10)):
                text = els.nth(i).inner_text() or els.nth(i).get_attribute("value") or ""
                text = text.strip()
                if len(text) > 8 and " " not in text and "\n" not in text:
                    log(f"password candidate: {text[:10]}...")
                    return text
        except Exception:
            pass

    # Try to find password-like string in body
    # ClickHouse generates passwords like random alphanumeric strings
    pw_match = re.search(r'(?:password|Password)[\s:]*([A-Za-z0-9!@#$%^&*]{12,})', body)
    if pw_match:
        log(f"password from text: {pw_match.group(1)[:10]}...")
        return pw_match.group(1)

    log("could not extract reset password")
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
    """Navigate to the Connect tab and extract the host from connection details."""
    # First dismiss any survey popup
    _click_first(page, [
        'button[aria-label="Close"]',
        'button[aria-label="close"]',
        '[data-testid="close-button"]',
        'button:has-text("×")',
        'button:has-text("Close")',
    ], log)
    _wait(1, log)

    # Click the "Connect" link in the sidebar
    clicked = _click_first(page, [
        'a:has-text("Connect")',
        '[data-testid*="connect"]',
        'nav a:has-text("Connect")',
    ], log)

    if clicked:
        _wait(3, log, "connect page load")
        # Look for host in the connection details
        host = _extract_host_from_html(page, log)
        if host:
            return host
        body = _log_page_state(page, log, "connect-page: ", max_chars=1000)
        host = _extract_host_from_text(body, log)
        if host:
            return host

    return ""


def _get_host_via_api(page, service_id: str, log) -> str:
    """Use the browser's authenticated session to call internal API for service details."""
    try:
        # ClickHouse Cloud console makes API calls to its own backend
        # Try fetching service details via the internal API
        result = page.evaluate(f"""
            (async () => {{
                try {{
                    const resp = await fetch('/api/services/{service_id}');
                    if (resp.ok) {{
                        const data = await resp.json();
                        return JSON.stringify(data);
                    }}
                    return 'status:' + resp.status;
                }} catch (e) {{
                    return 'error:' + e.message;
                }}
            }})()
        """)
        log(f"internal API response: {str(result)[:200]}")

        if isinstance(result, str) and "clickhouse.cloud" in result:
            host = _extract_host_from_text(result, log)
            if host:
                return host

        # Try the cloud API endpoint
        result = page.evaluate(f"""
            (async () => {{
                try {{
                    const resp = await fetch('https://api.clickhouse.cloud/v1/organizations');
                    if (!resp.ok) return 'org-status:' + resp.status;
                    const orgs = await resp.json();
                    const orgId = orgs.result?.[0]?.id;
                    if (!orgId) return 'no-org';

                    const svcResp = await fetch('https://api.clickhouse.cloud/v1/organizations/' + orgId + '/services/{service_id}');
                    if (!svcResp.ok) return 'svc-status:' + svcResp.status;
                    const svc = await svcResp.json();
                    return JSON.stringify(svc);
                }} catch (e) {{
                    return 'error:' + e.message;
                }}
            }})()
        """)
        log(f"cloud API response: {str(result)[:200]}")

        if isinstance(result, str) and "clickhouse.cloud" in result:
            host = _extract_host_from_text(result, log)
            if host:
                return host

    except Exception as e:
        log(f"API extraction error: {e}")

    return ""


def _get_service_host(page, service_id: str, log) -> str:
    """Navigate to service page and extract connection host."""
    # Try the service settings page
    settings_url = f"https://console.clickhouse.cloud/services/{service_id}/settings"
    try:
        page.goto(settings_url, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
    except Exception:
        pass
    _wait(3, log, "service settings load")

    body = _log_page_state(page, log, "service-settings: ", max_chars=1000)

    # Look for host in the page text
    host = _extract_host_from_text(body, log)
    if host:
        return host

    # Try the service connection/connect page
    connect_url = f"https://console.clickhouse.cloud/services/{service_id}/connect"
    try:
        page.goto(connect_url, timeout=10000)
        page.wait_for_load_state("networkidle", timeout=8000)
    except Exception:
        pass
    _wait(2, log, "service connect load")

    body = _log_page_state(page, log, "service-connect: ", max_chars=1000)
    host = _extract_host_from_text(body, log)
    if host:
        return host

    # Try the main service page
    svc_url = f"https://console.clickhouse.cloud/services/{service_id}"
    try:
        page.goto(svc_url, timeout=10000)
        page.wait_for_load_state("networkidle", timeout=8000)
    except Exception:
        pass
    _wait(2, log, "service page load")

    body = _log_page_state(page, log, "service-main: ", max_chars=1000)
    host = _extract_host_from_text(body, log)
    return host or ""


def _extract_host_from_text(text: str, log) -> str:
    """Extract ClickHouse Cloud service hostname from text."""
    # Exclude known non-service hosts
    exclude = {"console.clickhouse.cloud", "auth.clickhouse.cloud",
               "api.clickhouse.cloud", "statuspage.clickhouse.cloud",
               "console-api-internal.clickhouse.cloud",
               "console-api.clickhouse.cloud"}

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


def _find_host_in_page(page, log) -> str:
    """Search current page for ClickHouse host."""
    try:
        # Look in code blocks and inputs
        for sel in ['code', 'input[readonly]', 'input[value*="clickhouse"]', 'pre']:
            els = page.locator(sel)
            for i in range(min(els.count(), 10)):
                text = els.nth(i).inner_text() or els.nth(i).get_attribute("value") or ""
                host = _extract_host_from_text(text, log)
                if host:
                    return host

        # Search entire page body
        body = page.inner_text("body")
        return _extract_host_from_text(body, log)
    except Exception:
        return ""
