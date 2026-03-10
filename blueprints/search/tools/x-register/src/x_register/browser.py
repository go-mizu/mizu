"""Patchright browser helpers.

Used to fetch js_instrumentation from x.com — a bot-fingerprinting value
that must be submitted in the Signup onboarding subtask.
"""

from __future__ import annotations

import os
import platform
import re
import sys
import time

JS_INST_URL = "https://x.com/i/js_inst?c_name=ui_metrics"
JS_INST_RE = re.compile(r"return\s+(\{.*?\});", re.DOTALL)


def _browser_args(headless: bool) -> list[str]:
    args = [
        "--disable-blink-features=AutomationControlled",
        "--window-size=1920,1080",
    ]
    is_linux = platform.system() == "Linux"
    if is_linux:
        args += [
            "--no-sandbox",
            "--disable-setuid-sandbox",
            "--disable-dev-shm-usage",
            "--use-angle=swiftshader",
            "--enable-webgl",
            "--ignore-gpu-blocklist",
            "--enable-unsafe-swiftshader",
        ]
    elif headless:
        args += [
            "--use-angle=swiftshader",
            "--enable-webgl",
            "--ignore-gpu-blocklist",
            "--enable-unsafe-swiftshader",
        ]
    return args


def _maybe_reexec_xvfb(headless: bool) -> None:
    """On Linux non-headless with no DISPLAY, re-exec under xvfb-run."""
    import shutil, subprocess
    if platform.system() != "Linux" or headless:
        return
    if os.environ.get("DISPLAY"):
        return
    xvfb = shutil.which("xvfb-run")
    if not xvfb:
        return
    cmd = [xvfb, "-a", sys.executable] + sys.argv
    sys.exit(subprocess.call(cmd))


def get_js_instrumentation(
    proxy_url: str | None = None,
    headless: bool = True,
    verbose: bool = False,
) -> str:
    """Launch patchright, navigate to x.com/i/js_inst, return the JSON blob string.

    Args:
        proxy_url: Optional proxy URL, e.g. "http://host:port" or "socks5://host:port".
        headless: Whether to run headlessly.
        verbose: Log browser events to stdout.
    """
    from patchright.sync_api import sync_playwright

    _maybe_reexec_xvfb(headless)

    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [browser] {msg}", flush=True)

    log(f"launching chromium headless={headless} proxy={proxy_url}")

    proxy_config = None
    if proxy_url:
        proxy_config = {"server": proxy_url}

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=headless,
            args=_browser_args(headless),
            proxy=proxy_config,
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            proxy=proxy_config,
        )
        page = ctx.new_page()

        if verbose:
            page.on("console", lambda m: log(f"[console] {m.text[:100]}") if "GL Driver" not in m.text else None)
            page.on("pageerror", lambda e: log(f"[page-error] {e}"))

        try:
            log(f"navigating to {JS_INST_URL}")
            page.goto(JS_INST_URL, wait_until="domcontentloaded", timeout=20000)
            content = page.content()
            log(f"page loaded, content length={len(content)}")
        except Exception as e:
            log(f"navigation warning: {e}")
            try:
                content = page.content()
            except Exception:
                content = ""
        finally:
            browser.close()

    m = JS_INST_RE.search(content)
    if m:
        blob = m.group(1)
        log(f"js_instrumentation extracted ({len(blob)} chars)")
        return blob

    # Fallback: return the raw page text (X sometimes returns JSON directly)
    text = re.sub(r"<[^>]+>", "", content).strip()
    log(f"fallback js_instrumentation ({len(text)} chars)")
    return text or "{}"
