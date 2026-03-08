#!/usr/bin/env python3
"""Get a Jina AI API key via patchright (undetected Playwright).

Strategy:
1. Navigate to jina.ai/?newKey — browser solves Cloudflare Turnstile
2. Route intercept: capture cf-turnstile-response token from keygen.jina.ai POST
3. If direct request succeeds (not rate-limited), return key immediately
4. If rate-limited (429), replay keygen POST through proxies with captured token

Usage:
    pip install patchright
    python get_key.py [--verbose]
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


def _is_good(key):
    return key and KEY_RE.match(key) and key not in BLOCKLIST


def get_jina_key(headless=False, timeout=90, verbose=False):
    from patchright.sync_api import sync_playwright

    captured_keys = []
    captured_tokens = []
    rate_limited = False

    with sync_playwright() as p:
        launch_args = ["--disable-blink-features=AutomationControlled", "--window-size=1920,1080"]
        # Linux: --no-sandbox (root), --use-angle=swiftshader (software GL for xvfb)
        import platform
        if platform.system() == "Linux":
            launch_args.extend(["--no-sandbox", "--disable-dev-shm-usage", "--use-angle=swiftshader"])
        browser = p.chromium.launch(
            headless=headless,
            args=launch_args,
        )
        ctx = browser.new_context(viewport={"width": 1920, "height": 1080})

        def on_route(route):
            nonlocal rate_limited
            url = route.request.url
            if "keygen.jina.ai" not in url:
                route.continue_()
                return

            # Extract turnstile token from multipart form data
            post_data = route.request.post_data or ""
            if "cf-turnstile-response" in post_data:
                lines = post_data.split("\n")
                capture_next = False
                for line in lines:
                    line = line.strip("\r")
                    if capture_next and line and not line.startswith("--"):
                        captured_tokens.append(line)
                        if verbose:
                            print(f"  CAPTURED turnstile token ({len(line)} chars)", file=sys.stderr)
                        capture_next = False
                    if 'name="cf-turnstile-response"' in line:
                        capture_next = True

            # Redirect /empty -> /trial
            fetch_url = url.replace("/empty", "/trial") if "/empty" in url else url
            if "/empty" in url and verbose:
                print(f"  INTERCEPT: /empty -> /trial", file=sys.stderr)

            try:
                resp = route.fetch(url=fetch_url)
                body = resp.body()
                text = body.decode("utf-8", errors="replace")
                if verbose:
                    print(f"  keygen [{resp.status}]: {text[:300]}", file=sys.stderr)

                if resp.status == 429:
                    rate_limited = True
                else:
                    m = KEY_RE.search(text)
                    if m and m.group(0) not in BLOCKLIST:
                        captured_keys.append(m.group(0))
                        if verbose:
                            k = m.group(0)
                            print(f"  CAPTURED KEY: {k[:12]}...{k[-4:]}", file=sys.stderr)
                route.fulfill(status=resp.status, headers=dict(resp.headers), body=body)
            except Exception as e:
                if verbose:
                    print(f"  keygen error: {e}", file=sys.stderr)
                # "Request context disposed" = browser closing during fetch
                # Token was already captured — fall through to proxy replay
                if "disposed" in str(e).lower() and captured_tokens:
                    rate_limited = True  # force proxy replay path
                try:
                    route.continue_(url=fetch_url)
                except Exception:
                    pass

        ctx.route("**/keygen.jina.ai/**", on_route)
        page = ctx.new_page()

        if verbose:
            print("  loading jina.ai/?newKey...", file=sys.stderr)

        try:
            page.goto("https://jina.ai/?newKey", wait_until="load", timeout=60000)
        except Exception as e:
            browser.close()
            raise RuntimeError(f"Failed to load jina.ai: {e}")
        time.sleep(3)

        # Dismiss cookie banner (may fail in headless — non-critical)
        try:
            page.evaluate("""() => {
                const aside = document.querySelector('#usercentrics-cmp-ui');
                if (aside && aside.shadowRoot) {
                    for (const b of aside.shadowRoot.querySelectorAll('button')) {
                        const t = (b.textContent || '').toLowerCase();
                        if (t.includes('deny') || t.includes('reject')) { b.click(); return; }
                    }
                }
                if (aside) aside.remove();
            }""")
        except Exception:
            pass  # Page may have crashed — route intercept still works

        # Wait for keygen request to fire
        deadline = time.time() + min(timeout, 30)
        while time.time() < deadline:
            if captured_keys:
                key = captured_keys[0]
                if verbose:
                    print(f"  returning direct key: {key[:12]}...{key[-4:]}", file=sys.stderr)
                browser.close()
                return key

            if rate_limited and captured_tokens:
                if verbose:
                    print(f"  rate limited with token — switching to proxy replay", file=sys.stderr)
                browser.close()
                return _replay_via_proxies(captured_tokens[-1], verbose=verbose)

            key = _extract_dom(page)
            if _is_good(key):
                if verbose:
                    print(f"  returning DOM key: {key[:12]}...{key[-4:]}", file=sys.stderr)
                browser.close()
                return key

            elapsed = int(time.time() - (deadline - min(timeout, 30)))
            if verbose and elapsed % 10 == 0:
                print(f"  wait {elapsed}s: keys={len(captured_keys)} tokens={len(captured_tokens)} rl={rate_limited}", file=sys.stderr)

            time.sleep(2)

        browser.close()

        if captured_tokens:
            if verbose:
                print(f"  timeout, trying proxy replay...", file=sys.stderr)
            return _replay_via_proxies(captured_tokens[-1], verbose=verbose)

        raise RuntimeError(
            f"Key not found after {timeout}s. "
            "No turnstile token captured."
        )


def _replay_via_proxies(turnstile_token, verbose=False):
    """Replay keygen.jina.ai/trial POST through proxies."""
    import random

    proxies = _fetch_proxies(verbose)
    random.shuffle(proxies)

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
        except Exception:
            continue

        tested += 1
        if verbose:
            print(f"  proxy [{tested}] {scheme}://{host}:{port}...", file=sys.stderr)

        try:
            if scheme == "socks5":
                key = _keygen_via_socks5(host, port, turnstile_token, verbose)
            else:
                key = _keygen_via_http_connect(host, port, turnstile_token, verbose)
            if _is_good(key):
                if verbose:
                    print(f"  SUCCESS: {key[:12]}...{key[-4:]}", file=sys.stderr)
                return key
        except Exception as e:
            if verbose:
                err = str(e)[:80]
                print(f"    failed: {err}", file=sys.stderr)
            continue

    raise RuntimeError(
        f"Tried {tested} proxies, all failed. "
        "Turnstile token may have expired (~5 min lifetime)."
    )


def _fetch_proxies(verbose=False):
    """Fetch proxy candidates. Returns list of (scheme, host, port)."""
    results = []
    for url, scheme in PROXY_SOURCES:
        try:
            resp = urllib.request.urlopen(url, timeout=10)
            lines = resp.read().decode("utf-8", errors="replace").strip().split("\n")
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
        except Exception as e:
            if verbose:
                print(f"  proxy source error: {e}", file=sys.stderr)
    if verbose:
        print(f"  fetched {len(results)} proxy candidates", file=sys.stderr)
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
        f"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36\r\n"
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

    if verbose:
        print(f"    [{status_line}] {resp_body[:200]}", file=sys.stderr)

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
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))

    # Send CONNECT request
    connect_req = (
        f"CONNECT keygen.jina.ai:443 HTTP/1.1\r\n"
        f"Host: keygen.jina.ai:443\r\n"
        f"\r\n"
    ).encode()
    s.send(connect_req)

    # Read CONNECT response
    resp = b""
    while b"\r\n\r\n" not in resp:
        chunk = s.recv(4096)
        if not chunk:
            s.close()
            raise RuntimeError("CONNECT: no response")
        resp += chunk

    resp_str = resp.decode("utf-8", errors="replace")
    if "200" not in resp_str.split("\r\n")[0]:
        s.close()
        raise RuntimeError(f"CONNECT failed: {resp_str.split(chr(13))[0]}")

    # TLS wrap on the tunnel
    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")

    # Send keygen request
    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()

    return _parse_response(text, verbose)


def _keygen_via_socks5(proxy_host, proxy_port, turnstile_token, verbose=False):
    """Make keygen request via SOCKS5 proxy."""
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))

    # SOCKS5 handshake
    s.send(b"\x05\x01\x00")
    resp = s.recv(2)
    if resp != b"\x05\x00":
        s.close()
        raise RuntimeError("SOCKS5 auth rejected")

    # CONNECT to keygen.jina.ai:443
    target = b"keygen.jina.ai"
    s.send(b"\x05\x01\x00\x03" + bytes([len(target)]) + target + (443).to_bytes(2, "big"))
    resp = s.recv(10)
    if len(resp) < 2 or resp[1] != 0:
        s.close()
        raise RuntimeError("SOCKS5 connect failed")

    # TLS wrap
    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")

    # Send keygen request
    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()

    return _parse_response(text, verbose)


def _extract_dom(page) -> str:
    """Check input values, innerText, localStorage."""
    try:
        return page.evaluate("""() => {
            const re = /jina_[a-f0-9]{32}[a-zA-Z0-9_-]+/;
            for (const el of document.querySelectorAll('input, textarea')) {
                const m = re.exec(el.value || '');
                if (m) return m[0];
            }
            const m1 = re.exec(document.body.innerText);
            if (m1) return m1[0];
            for (let i = 0; i < localStorage.length; i++) {
                const m = re.exec(localStorage.getItem(localStorage.key(i)) || '');
                if (m) return m[0];
            }
            return '';
        }""") or ""
    except Exception:
        return ""


def main():
    parser = argparse.ArgumentParser(description="Get Jina AI API key")
    parser.add_argument("--headless", action="store_true")
    parser.add_argument("--timeout", type=int, default=90)
    parser.add_argument("--verbose", "-v", action="store_true")
    args = parser.parse_args()

    try:
        key = get_jina_key(headless=args.headless, timeout=args.timeout, verbose=args.verbose)
        print(f"KEY:{key}")
    except Exception as e:
        print(f"ERROR:{e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
