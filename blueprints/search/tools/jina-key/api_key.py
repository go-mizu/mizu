#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "patchright",
# ]
# ///
"""Get a Jina AI API key via patchright (undetected Playwright).

Strategy:
1. Navigate to jina.ai/?newKey — browser solves Cloudflare Turnstile
2. Route intercept: capture cf-turnstile-response token from keygen.jina.ai POST
3. If direct request succeeds (not rate-limited), return key immediately
4. If rate-limited (429), replay keygen POST through proxies with captured token

Usage (uv — zero manual install):
    uv run api_key.py
    uv run api_key.py --verbose
    uv run api_key.py --no-headless   # show browser window
"""

import argparse
import re
import socket
import ssl
import sys
import time
import urllib.request

KEY_RE = re.compile(r"jina_[a-f0-9]{32}[a-zA-Z0-9_-]+")

BLOCKLIST = {
    "jina_387ced4ff3f04305ac001d5d6577e184hKPgRPGo4yMp_3NIxVsW6XTZZWNL",
}

# Proxy lists — prioritize sources with clean (non-MITM) proxies
PROXY_SOURCES = [
    # HTTPS proxies (HTTP CONNECT — properly tunnel TLS)
    ("https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt", "https"),
    ("https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", "http"),
    # SOCKS5
    ("https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"),
    ("https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"),
]


def _log(msg, verbose=False, file=sys.stderr):
    if verbose:
        ts = time.strftime("%H:%M:%S")
        print(f"[{ts}] {msg}", file=file, flush=True)


def _is_good(key):
    return key and KEY_RE.match(key) and key not in BLOCKLIST


def get_jina_key(headless=True, timeout=90, verbose=False):
    from patchright.sync_api import sync_playwright

    _log(f"Starting patchright (headless={headless})", verbose)

    captured_keys = []
    captured_tokens = []
    rate_limited = False

    with sync_playwright() as p:
        # Keep args minimal — patchright patches headless mode internally.
        # Extra headless flags can expose automation signals to Cloudflare Turnstile.
        browser_args = [
            "--disable-blink-features=AutomationControlled",
            "--window-size=1920,1080",
        ]

        _log(f"Launching chromium headless={headless} args={browser_args}", verbose)
        browser = p.chromium.launch(
            headless=headless,
            args=browser_args,
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            user_agent=(
                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
                "AppleWebKit/537.36 (KHTML, like Gecko) "
                "Chrome/131.0.0.0 Safari/537.36"
            ),
        )

        # --- Console messages from the page ---
        page_ref = [None]
        def _on_console(msg):
            _log(f"  [console:{msg.type}] {msg.text}", verbose)

        def _on_page_error(err):
            _log(f"  [page-error] {err}", verbose)

        def _on_request(req):
            _log(f"  [req] {req.method} {req.url[:120]}", verbose)

        def _on_response(resp):
            _log(f"  [resp] {resp.status} {resp.url[:120]}", verbose)

        def on_route(route):
            nonlocal rate_limited
            url = route.request.url
            if "keygen.jina.ai" not in url:
                route.continue_()
                return

            _log(f"  [keygen] intercepted {url}", verbose)

            # Extract turnstile token from multipart form data
            post_data = route.request.post_data or ""
            _log(f"  [keygen] post_data length={len(post_data)}", verbose)
            if "cf-turnstile-response" in post_data:
                lines = post_data.split("\n")
                capture_next = False
                for line in lines:
                    line = line.strip("\r")
                    if capture_next and line and not line.startswith("--"):
                        captured_tokens.append(line)
                        _log(f"  [keygen] captured turnstile token ({len(line)} chars)", verbose)
                        capture_next = False
                    if 'name="cf-turnstile-response"' in line:
                        capture_next = True

            # Redirect /empty -> /trial
            fetch_url = url.replace("/empty", "/trial") if "/empty" in url else url
            if "/empty" in url:
                _log(f"  [keygen] rewrote /empty -> /trial", verbose)

            try:
                _log(f"  [keygen] fetching {fetch_url}", verbose)
                resp = route.fetch(url=fetch_url)
                body = resp.body()
                text = body.decode("utf-8", errors="replace")
                _log(f"  [keygen] response status={resp.status} body={text[:300]}", verbose)

                if resp.status == 429:
                    _log(f"  [keygen] rate limited (429)", verbose)
                    rate_limited = True
                else:
                    m = KEY_RE.search(text)
                    if m and m.group(0) not in BLOCKLIST:
                        k = m.group(0)
                        captured_keys.append(k)
                        _log(f"  [keygen] captured key {k[:12]}...{k[-4:]}", verbose)
                route.fulfill(status=resp.status, headers=dict(resp.headers), body=body)
            except Exception as e:
                _log(f"  [keygen] fetch error: {e}", verbose)
                try:
                    route.continue_(url=fetch_url)
                except Exception as e2:
                    _log(f"  [keygen] continue error: {e2}", verbose)

        ctx.route("**/keygen.jina.ai/**", on_route)
        page = ctx.new_page()
        page_ref[0] = page

        # Wire up page event listeners
        page.on("console", _on_console)
        page.on("pageerror", _on_page_error)
        if verbose:
            page.on("request", _on_request)
            page.on("response", _on_response)

        _log("Navigating to https://jina.ai/?newKey ...", verbose)
        try:
            page.goto("https://jina.ai/?newKey", wait_until="domcontentloaded", timeout=30000)
            _log(f"Navigation done, URL={page.url}", verbose)
        except Exception as e:
            _log(f"Navigation failed: {e}", verbose)
            browser.close()
            raise RuntimeError(f"Failed to load jina.ai: {e}")

        _log("Sleeping 3s for Turnstile to fire...", verbose)
        time.sleep(3)

        # Dismiss cookie banner
        _log("Dismissing cookie banner...", verbose)
        try:
            result = page.evaluate("""() => {
                const aside = document.querySelector('#usercentrics-cmp-ui');
                if (aside && aside.shadowRoot) {
                    for (const b of aside.shadowRoot.querySelectorAll('button')) {
                        const t = (b.textContent || '').toLowerCase();
                        if (t.includes('deny') || t.includes('reject')) { b.click(); return 'clicked-reject'; }
                    }
                }
                if (aside) { aside.remove(); return 'removed'; }
                return 'no-banner';
            }""")
            _log(f"  cookie banner result: {result}", verbose)
        except Exception as e:
            _log(f"  cookie banner dismiss error (non-fatal): {e}", verbose)

        # Try to trigger the keygen flow by clicking any visible CTA button
        _log("Looking for Get API Key button...", verbose)
        try:
            btns = page.evaluate("""() => {
                const texts = [];
                for (const b of document.querySelectorAll('button, a[href], [role=button]')) {
                    const t = (b.textContent || b.innerText || '').trim().toLowerCase();
                    if (t.includes('api key') || t.includes('get key') || t.includes('new key') || t.includes('generate')) {
                        texts.push(t.slice(0,40));
                        b.click();
                    }
                }
                return texts;
            }""")
            if btns:
                _log(f"  clicked buttons: {btns}", verbose)
            else:
                _log(f"  no CTA button found, Turnstile should auto-fire", verbose)
        except Exception as e:
            _log(f"  button click error (non-fatal): {e}", verbose)

        # Give Turnstile time to fire after button click
        _log("Waiting 2s for Turnstile after button click...", verbose)
        time.sleep(2)

        # Try to interact with Turnstile iframe if present (triggers interactive challenge)
        _log("Checking for Turnstile iframe...", verbose)
        try:
            ts_info = page.evaluate("""() => {
                const frames = [];
                for (const f of document.querySelectorAll('iframe')) {
                    frames.push({src: f.src.slice(0,80), id: f.id, cls: f.className});
                }
                return frames;
            }""")
            _log(f"  iframes on page: {ts_info}", verbose)
        except Exception as e:
            _log(f"  iframe check error: {e}", verbose)

        # Main wait loop
        _log(f"Waiting up to {min(timeout, 30)}s for key/token...", verbose)
        deadline = time.time() + min(timeout, 30)
        last_log = 0

        while time.time() < deadline:
            if captured_keys:
                key = captured_keys[0]
                _log(f"Returning intercepted key {key[:12]}...{key[-4:]}", verbose)
                browser.close()
                return key

            if rate_limited and captured_tokens:
                _log("Rate limited but have token — switching to proxy replay", verbose)
                browser.close()
                return _replay_via_proxies(captured_tokens[-1], verbose=verbose)

            key = _extract_dom(page, verbose)
            if key:
                if _is_good(key):
                    _log(f"Returning DOM key {key[:12]}...{key[-4:]}", verbose)
                    browser.close()
                    return key
                else:
                    _log(f"  [dom] key rejected (blocklist/invalid): {key[:20]}...", verbose)

            elapsed = int(time.time() - (deadline - min(timeout, 30)))
            if verbose and elapsed - last_log >= 5:
                last_log = elapsed
                title = ""
                try:
                    title = page.title()
                except Exception:
                    pass
                _log(
                    f"  wait {elapsed}s | keys={len(captured_keys)} tokens={len(captured_tokens)}"
                    f" rl={rate_limited} url={page.url[:60]} title={title!r}",
                    verbose,
                )

            time.sleep(1)

        _log("Timed out waiting for key/token", verbose)
        browser.close()

        if captured_tokens:
            _log("Timeout — trying proxy replay with captured token...", verbose)
            return _replay_via_proxies(captured_tokens[-1], verbose=verbose)

        raise RuntimeError(
            f"Key not found after {timeout}s. No turnstile token captured."
        )


def _replay_via_proxies(turnstile_token, verbose=False):
    """Replay keygen.jina.ai/trial POST through proxies."""
    import random

    _log("Fetching proxy candidates...", verbose)
    proxies = _fetch_proxies(verbose)
    random.shuffle(proxies)
    _log(f"Testing up to 50 of {len(proxies)} proxies...", verbose)

    tested = 0
    for scheme, host, port in proxies:
        if tested >= 50:
            break

        # Quick connect test
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(3)
            s.connect((host, port))
            s.close()
        except Exception as e:
            _log(f"  proxy {scheme}://{host}:{port} unreachable: {e}", verbose)
            continue

        tested += 1
        _log(f"  [{tested}] trying {scheme}://{host}:{port}...", verbose)

        try:
            if scheme == "socks5":
                key = _keygen_via_socks5(host, port, turnstile_token, verbose)
            else:
                key = _keygen_via_http_connect(host, port, turnstile_token, verbose)
            if _is_good(key):
                _log(f"  SUCCESS via proxy {host}:{port}: {key[:12]}...{key[-4:]}", verbose)
                return key
            else:
                _log(f"  proxy {host}:{port} returned no valid key", verbose)
        except Exception as e:
            _log(f"  proxy {host}:{port} failed: {str(e)[:80]}", verbose)
            continue

    raise RuntimeError(
        f"Tried {tested} proxies, all failed. "
        "Turnstile token may have expired (~5 min lifetime)."
    )


def _fetch_proxies(verbose=False):
    """Fetch proxy candidates. Returns list of (scheme, host, port)."""
    results = []
    for url, scheme in PROXY_SOURCES:
        _log(f"  fetching proxy source [{scheme}] {url}", verbose)
        try:
            resp = urllib.request.urlopen(url, timeout=10)
            lines = resp.read().decode("utf-8", errors="replace").strip().split("\n")
            before = len(results)
            for line in lines:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if ":" in line:
                    parts = line.rsplit(":", 1)
                    try:
                        results.append((scheme, parts[0], int(parts[1])))
                    except ValueError:
                        pass
            _log(f"  -> got {len(results) - before} proxies from that source", verbose)
        except Exception as e:
            _log(f"  proxy source error: {e}", verbose)
    _log(f"Total proxy candidates: {len(results)}", verbose)
    return results


def _build_keygen_request(turnstile_token):
    """Build the HTTP request bytes for keygen.jina.ai/trial."""
    boundary = "----WebKitFormBoundaryPythonProxy"
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="cf-turnstile-response"\r\n'
        f"\r\n"
        f"{turnstile_token}\r\n"
        f"--{boundary}--\r\n"
    ).encode()

    headers = (
        f"POST /trial HTTP/1.1\r\n"
        f"Host: keygen.jina.ai\r\n"
        f"Content-Type: multipart/form-data; boundary={boundary}\r\n"
        f"Content-Length: {len(body)}\r\n"
        f"Origin: https://jina.ai\r\n"
        f"Referer: https://jina.ai/\r\n"
        f"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36\r\n"
        f"Accept: */*\r\n"
        f"Connection: close\r\n"
        f"\r\n"
    ).encode()

    return headers + body


def _read_response(ss):
    """Read full HTTP response from socket."""
    data = b""
    while True:
        try:
            chunk = ss.recv(4096)
            if not chunk:
                break
            data += chunk
        except socket.timeout:
            break
    return data.decode("utf-8", errors="replace")


def _parse_response(text, verbose=False):
    """Parse HTTP response and extract key."""
    status_line = text.split("\r\n")[0] if text else "?"
    parts = text.split("\r\n\r\n", 1)
    resp_body = parts[1] if len(parts) > 1 else text

    _log(f"    [{status_line}] body={resp_body[:200]}", verbose)

    if "429" in status_line or "rate limit" in resp_body.lower():
        raise RuntimeError("rate limited on this proxy too")

    if "CF verification failed" in resp_body:
        raise RuntimeError("turnstile token rejected")

    m = KEY_RE.search(resp_body)
    if m and m.group(0) not in BLOCKLIST:
        return m.group(0)
    return None


def _keygen_via_http_connect(proxy_host, proxy_port, turnstile_token, verbose=False):
    """Make keygen request via HTTP CONNECT proxy (proper TLS tunnel)."""
    _log(f"    CONNECT {proxy_host}:{proxy_port} -> keygen.jina.ai:443", verbose)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))

    connect_req = (
        f"CONNECT keygen.jina.ai:443 HTTP/1.1\r\n"
        f"Host: keygen.jina.ai:443\r\n"
        f"\r\n"
    ).encode()
    s.send(connect_req)

    resp = b""
    while b"\r\n\r\n" not in resp:
        chunk = s.recv(4096)
        if not chunk:
            s.close()
            raise RuntimeError("CONNECT: no response")
        resp += chunk

    resp_str = resp.decode("utf-8", errors="replace")
    status = resp_str.split("\r\n")[0]
    _log(f"    CONNECT response: {status}", verbose)
    if "200" not in status:
        s.close()
        raise RuntimeError(f"CONNECT failed: {status}")

    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")
    _log(f"    TLS handshake ok, sending keygen POST...", verbose)

    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()

    return _parse_response(text, verbose)


def _keygen_via_socks5(proxy_host, proxy_port, turnstile_token, verbose=False):
    """Make keygen request via SOCKS5 proxy."""
    _log(f"    SOCKS5 {proxy_host}:{proxy_port} -> keygen.jina.ai:443", verbose)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))

    s.send(b"\x05\x01\x00")
    resp = s.recv(2)
    if resp != b"\x05\x00":
        s.close()
        raise RuntimeError("SOCKS5 auth rejected")
    _log(f"    SOCKS5 auth ok", verbose)

    target = b"keygen.jina.ai"
    s.send(b"\x05\x01\x00\x03" + bytes([len(target)]) + target + (443).to_bytes(2, "big"))
    resp = s.recv(10)
    if len(resp) < 2 or resp[1] != 0:
        s.close()
        raise RuntimeError(f"SOCKS5 connect failed (code={resp[1] if len(resp)>=2 else '?'})")
    _log(f"    SOCKS5 tunnel established", verbose)

    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")
    _log(f"    TLS handshake ok, sending keygen POST...", verbose)

    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()

    return _parse_response(text, verbose)


def _extract_dom(page, verbose=False) -> str:
    """Check input values, innerText, localStorage."""
    try:
        result = page.evaluate("""() => {
            const re = /jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/;
            for (const el of document.querySelectorAll('input, textarea')) {
                const m = re.exec(el.value || '');
                if (m) return 'input:' + m[0];
            }
            const m1 = re.exec(document.body.innerText);
            if (m1) return 'body:' + m1[0];
            for (let i = 0; i < localStorage.length; i++) {
                const m = re.exec(localStorage.getItem(localStorage.key(i)) || '');
                if (m) return 'ls:' + m[0];
            }
            return '';
        }""") or ""
        if result:
            src, _, key = result.partition(":")
            _log(f"  [dom] found key in {src}", verbose)
            return key
        return ""
    except Exception as e:
        _log(f"  [dom] eval error: {e}", verbose)
        return ""


def main():
    parser = argparse.ArgumentParser(description="Get Jina AI API key via undetected headless Chrome")
    parser.add_argument("--no-headless", dest="headless", action="store_false", default=True,
                        help="Show browser window (default: headless)")
    parser.add_argument("--timeout", type=int, default=90,
                        help="Max seconds to wait for key (default: 90)")
    parser.add_argument("--verbose", "-v", action="store_true",
                        help="Verbose step-by-step logging to stderr")
    args = parser.parse_args()

    _log(f"args: headless={args.headless} timeout={args.timeout}", args.verbose)

    try:
        key = get_jina_key(headless=args.headless, timeout=args.timeout, verbose=args.verbose)
        print(f"KEY:{key}", flush=True)
    except Exception as e:
        print(f"ERROR:{e}", file=sys.stderr, flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
