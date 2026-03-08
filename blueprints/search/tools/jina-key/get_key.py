#!/usr/bin/env python3
"""Get a Jina AI API key by solving Turnstile and calling keygen.jina.ai.

Approach:
1. Navigate to jina.ai to find the Turnstile sitekey
2. Solve Turnstile on a minimal page (patchright)
3. POST token to keygen.jina.ai/trial for 1M-token key

Usage:
    pip install patchright httpx
    patchright install chromium
    python get_key.py [--headless] [--verbose]

Output: prints "KEY:<jina_key>" on success.
"""

import argparse
import re
import sys
import time
import httpx


KEYGEN_URL_TRIAL = "https://keygen.jina.ai/trial"
KEYGEN_URL_EMPTY = "https://keygen.jina.ai/empty"
KEY_RE = re.compile(r"jina_[a-f0-9]{32}[a-zA-Z0-9_-]+")


def find_sitekey(verbose: bool = False) -> str:
    """Fetch jina.ai page source and extract Turnstile sitekey."""
    resp = httpx.get("https://jina.ai/", follow_redirects=True, timeout=15)
    # Look for Turnstile sitekey (0x...)
    m = re.search(r'["\']?(0x[A-Za-z0-9_-]{20,})["\']?', resp.text)
    if m:
        key = m.group(1)
        if verbose:
            print(f"  found sitekey: {key}", file=sys.stderr)
        return key
    # Also check for data-sitekey attribute
    m2 = re.search(r'data-sitekey=["\']([^"\']+)', resp.text)
    if m2:
        key = m2.group(1)
        if verbose:
            print(f"  found sitekey (data-attr): {key}", file=sys.stderr)
        return key
    raise RuntimeError("Could not find Turnstile sitekey on jina.ai")


def solve_turnstile(sitekey: str, headless: bool = False, verbose: bool = False) -> str:
    """Solve Turnstile on jina.ai, intercept the keygen call, redirect /empty -> /trial."""
    from patchright.sync_api import sync_playwright

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=headless,
            args=["--disable-blink-features=AutomationControlled", "--window-size=1920,1080"],
        )
        context = browser.new_context(viewport={"width": 1920, "height": 1080})

        # Intercept keygen requests: redirect /empty -> /trial
        def handle_route(route):
            url = route.request.url
            if "keygen.jina.ai/empty" in url:
                new_url = url.replace("/empty", "/trial")
                if verbose:
                    print(f"  INTERCEPTED: {url} -> {new_url}", file=sys.stderr)
                route.continue_(url=new_url)
            else:
                route.continue_()

        context.route("**/keygen.jina.ai/**", handle_route)

        page = context.new_page()

        if verbose:
            print("  navigating to jina.ai/?newKey with /empty->/trial intercept...", file=sys.stderr)

        page.goto("https://jina.ai/?newKey", wait_until="domcontentloaded")
        time.sleep(3)

        # Wait for key to appear (now via /trial = 1M tokens)
        for i in range(30):
            # Check DOM for key
            key_val = page.evaluate("""() => {
                const re = /jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/;
                for (const el of document.querySelectorAll('input, textarea')) {
                    const v = el.value || el.getAttribute('value') || '';
                    const m = re.exec(v);
                    if (m) return m[0];
                }
                for (const el of document.querySelectorAll('code, pre, span, div, p')) {
                    const v = el.textContent || '';
                    if (v.length > 5000) continue;
                    const m = re.exec(v);
                    if (m) return m[0];
                }
                return '';
            }""")
            if key_val and key_val.startswith("jina_"):
                if verbose:
                    print(f"  found key (via /trial): {key_val[:12]}...{key_val[-4:]}", file=sys.stderr)
                browser.close()
                return f"DIRECT_KEY:{key_val}"

            text = page.inner_text("body")
            m = KEY_RE.search(text)
            if m:
                browser.close()
                return f"DIRECT_KEY:{m.group(0)}"

            if verbose and i % 5 == 0:
                has_error = "cannot generate" in text.lower()
                has_ready = "key is ready" in text.lower()
                print(f"  scan {i*2}s: error={has_error} ready={has_ready}", file=sys.stderr)

            time.sleep(2)

        browser.close()
        raise RuntimeError("Turnstile did not solve / key not found")


def call_keygen(token: str, endpoint: str = KEYGEN_URL_TRIAL, verbose: bool = False) -> str:
    """POST Turnstile token to keygen.jina.ai to get API key."""
    resp = httpx.post(
        endpoint,
        json={"turnstile_token": token},
        headers={"Content-Type": "application/json"},
        timeout=15,
    )
    if verbose:
        print(f"  keygen response ({resp.status_code}): {resp.text[:200]}", file=sys.stderr)
    if resp.status_code >= 400:
        raise RuntimeError(f"keygen failed: HTTP {resp.status_code}: {resp.text[:200]}")
    # Extract key from response
    data = resp.json()
    if isinstance(data, dict):
        for k in ("key", "api_key", "apiKey", "data"):
            if k in data:
                val = data[k]
                if isinstance(val, str) and KEY_RE.match(val):
                    return val
                if isinstance(val, dict):
                    for k2 in ("key", "api_key", "apiKey"):
                        if k2 in val and KEY_RE.match(str(val[k2])):
                            return str(val[k2])
    # Try regex on full response
    m = KEY_RE.search(resp.text)
    if m:
        return m.group(0)
    raise RuntimeError(f"No key found in keygen response: {resp.text[:200]}")


def get_jina_key_direct(headless: bool = False, timeout: int = 60, verbose: bool = False) -> str:
    """Simplest approach: just navigate to ?newKey with patchright and wait."""
    from patchright.sync_api import sync_playwright

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=headless,
            args=["--disable-blink-features=AutomationControlled", "--window-size=1920,1080"],
        )
        context = browser.new_context(viewport={"width": 1920, "height": 1080})
        page = context.new_page()

        if verbose:
            print("  navigating to jina.ai/?newKey...", file=sys.stderr)

        page.goto("https://jina.ai/?newKey", wait_until="domcontentloaded")
        time.sleep(3)

        # Try clicking Turnstile if visible
        turnstile_clicked = False
        deadline = time.time() + timeout
        while time.time() < deadline:
            text = page.inner_text("body")
            m = KEY_RE.search(text)
            if m:
                key = m.group(0)
                if verbose:
                    print(f"  found key in text: {key[:12]}...{key[-4:]}", file=sys.stderr)
                browser.close()
                return key

            # Check all inputs, textareas, code blocks for key
            key_val = page.evaluate("""() => {
                const re = /jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/;
                // inputs
                for (const el of document.querySelectorAll('input, textarea')) {
                    const v = el.value || el.getAttribute('value') || '';
                    const m = re.exec(v);
                    if (m) return m[0];
                }
                // code, pre, span, div with short text
                for (const el of document.querySelectorAll('code, pre, span, div, p')) {
                    const v = el.textContent || '';
                    if (v.length > 5000) continue;
                    const m = re.exec(v);
                    if (m) return m[0];
                }
                // clipboard / data attributes
                for (const el of document.querySelectorAll('[data-key], [data-value], [data-api-key]')) {
                    for (const attr of el.attributes) {
                        const m = re.exec(attr.value);
                        if (m) return m[0];
                    }
                }
                return '';
            }""")
            if key_val and key_val.startswith("jina_"):
                if verbose:
                    print(f"  found key in DOM: {key_val[:12]}...{key_val[-4:]}", file=sys.stderr)
                browser.close()
                return key_val

            # If page says "key is ready", dump more info to find the key
            if "key is ready" in text.lower() or "free api key" in text.lower():
                if verbose:
                    # Dump the section around "Free API Key"
                    snippet = page.evaluate("""() => {
                        const body = document.body.innerText;
                        const idx = body.toLowerCase().indexOf('free api key');
                        if (idx >= 0) return body.substring(Math.max(0, idx - 200), idx + 500);
                        const idx2 = body.toLowerCase().indexOf('key is ready');
                        if (idx2 >= 0) return body.substring(Math.max(0, idx2 - 200), idx2 + 500);
                        return body.substring(0, 1000);
                    }""")
                    print(f"  KEY AREA TEXT: {snippet}", file=sys.stderr)
                    # Also dump all input values
                    inputs_info = page.evaluate("""() => {
                        const result = [];
                        document.querySelectorAll('input').forEach(el => {
                            result.push({type: el.type, name: el.name, value: el.value, id: el.id});
                        });
                        return JSON.stringify(result);
                    }""")
                    print(f"  ALL INPUTS: {inputs_info}", file=sys.stderr)

            # Try clicking Turnstile checkbox
            if not turnstile_clicked:
                try:
                    frame = page.frame_locator("iframe[src*='turnstile']")
                    checkbox = frame.locator("input[type='checkbox'], .cb-lb, div[role='checkbox']")
                    if checkbox.count() > 0:
                        checkbox.first.click(timeout=3000)
                        turnstile_clicked = True
                        if verbose:
                            print("  clicked Turnstile checkbox inside iframe", file=sys.stderr)
                        time.sleep(3)
                        continue
                except Exception:
                    pass

                # Try clicking the Turnstile container div
                try:
                    container = page.query_selector(".cf-turnstile, [data-turnstile-callback]")
                    if container:
                        container.click()
                        turnstile_clicked = True
                        if verbose:
                            print("  clicked Turnstile container", file=sys.stderr)
                        time.sleep(3)
                        continue
                except Exception:
                    pass

            elapsed = int(timeout - (deadline - time.time()))
            if verbose and elapsed % 10 == 0:
                has_error = "cannot generate" in text.lower()
                has_ready = "key is ready" in text.lower()
                print(f"  scan {elapsed}s: error={has_error} ready={has_ready}", file=sys.stderr)

            # Reload after 25s if still error
            if elapsed == 25 and "cannot generate" in text.lower():
                if verbose:
                    print("  reloading page...", file=sys.stderr)
                page.reload(wait_until="domcontentloaded")
                turnstile_clicked = False
                time.sleep(3)
                continue

            time.sleep(2)

        browser.close()
        raise RuntimeError("Key not found within timeout")


def main():
    parser = argparse.ArgumentParser(description="Get Jina AI API key")
    parser.add_argument("--headless", action="store_true", help="Run headless")
    parser.add_argument("--timeout", type=int, default=60, help="Timeout in seconds")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--method", choices=["direct", "turnstile"], default="direct",
                        help="Method: direct (?newKey page) or turnstile (solve + keygen API)")
    args = parser.parse_args()

    try:
        if args.method == "turnstile":
            if args.verbose:
                print("  finding Turnstile sitekey...", file=sys.stderr)
            sitekey = find_sitekey(verbose=args.verbose)

            if args.verbose:
                print(f"  solving Turnstile (sitekey={sitekey[:20]}...)...", file=sys.stderr)
            token = solve_turnstile(sitekey, headless=args.headless, verbose=args.verbose)

            if token.startswith("DIRECT_KEY:"):
                key = token.split(":", 1)[1]
            else:
                if args.verbose:
                    print("  calling keygen.jina.ai/trial...", file=sys.stderr)
                key = call_keygen(token, verbose=args.verbose)
        else:
            key = get_jina_key_direct(
                headless=args.headless, timeout=args.timeout, verbose=args.verbose
            )

        print(f"KEY:{key}")

    except Exception as e:
        print(f"ERROR:{e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
