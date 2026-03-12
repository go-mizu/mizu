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
   - Good proxies (worked before) are tried first from ~/data/jina/good_proxies.json
   - Failed proxies are skipped and added to ~/data/jina/bad_proxies.json

Usage (uv — zero manual install):
    uv run api_key.py
    uv run api_key.py --verbose
    uv run api_key.py --no-headless   # show browser window
"""

import argparse
import json
import os
import re
import socket
import ssl
import sys
import time
import threading
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

KEY_RE = re.compile(r"jina_[a-f0-9]{32}[a-zA-Z0-9_-]+")

BLOCKLIST = {
    "jina_387ced4ff3f04305ac001d5d6577e184hKPgRPGo4yMp_3NIxVsW6XTZZWNL",
}

PROXY_SOURCES = [
    ("https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt", "https"),
    ("https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", "http"),
    ("https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"),
    ("https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"),
]

DATA_DIR = Path.home() / "data" / "jina"
BAD_PROXIES_FILE = DATA_DIR / "bad_proxies.json"
GOOD_PROXIES_FILE = DATA_DIR / "good_proxies.json"


# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

def _log(msg, verbose=False, file=sys.stderr):
    if verbose:
        ts = time.strftime("%H:%M:%S")
        print(f"[{ts}] {msg}", file=file, flush=True)


def _is_good(key):
    return key and KEY_RE.match(key) and key not in BLOCKLIST


def _check_key_balance(key, verbose=False):
    """Check key balance via dash.jina.ai. Returns (trial_balance, total_balance, data)."""
    url = f"https://dash.jina.ai/api/v1/api_key/fe_user?api_key={key}"
    req = urllib.request.Request(
        url,
        headers={"User-Agent": "Mozilla/5.0", "Accept": "application/json"}
    )
    try:
        resp = urllib.request.urlopen(req, timeout=10)
        data = json.loads(resp.read())
        wallet = data.get("wallet", {})
        trial_balance = wallet.get("trial_balance", 0)
        total_balance = wallet.get("total_balance", 0)
        trial_end = wallet.get("trial_end", "")[:10]
        _log(f"  [balance] trial={trial_balance} total={total_balance} trial_end={trial_end}", verbose)
        return trial_balance, total_balance, data
    except Exception as e:
        _log(f"  [balance] check failed: {e}", verbose)
        raise


def _validate_key(key, verbose=False):
    """Return key if it has positive balance; raise RuntimeError if zero balance."""
    _log(f"Checking balance for {key[:12]}...{key[-4:]}", verbose)
    try:
        trial_balance, total_balance, _data = _check_key_balance(key, verbose)
        if total_balance <= 0 and trial_balance <= 0:
            raise RuntimeError(f"Key has zero balance (trial={trial_balance} total={total_balance})")
        _log(f"Key valid: trial_balance={trial_balance} total_balance={total_balance}", verbose)
        return key
    except RuntimeError:
        raise
    except Exception as e:
        # If balance API is unreachable, return the key anyway (best-effort)
        _log(f"  [balance] API unreachable — returning key without balance check: {e}", verbose)
        return key


# ---------------------------------------------------------------------------
# Proxy persistence
# ---------------------------------------------------------------------------

def _load_json(path):
    try:
        if path.exists():
            return json.loads(path.read_text())
    except Exception:
        pass
    return {}


def _save_json(path, data):
    try:
        DATA_DIR.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data, indent=2))
    except Exception:
        pass


def _proxy_key(scheme, host, port):
    return f"{scheme}://{host}:{port}"


def _load_bad_proxies():
    """Return set of proxy keys that have failed."""
    data = _load_json(BAD_PROXIES_FILE)
    # Prune entries older than 24h
    cutoff = time.time() - 86400
    return {k for k, v in data.items() if v.get("ts", 0) > cutoff}


def _record_bad_proxy(scheme, host, port, reason, verbose=False):
    key = _proxy_key(scheme, host, port)
    data = _load_json(BAD_PROXIES_FILE)
    data[key] = {"ts": time.time(), "reason": reason[:80]}
    _save_json(BAD_PROXIES_FILE, data)
    _log(f"  [proxy-cache] bad: {key} ({reason[:40]})", verbose)


def _record_good_proxy(scheme, host, port, verbose=False):
    key = _proxy_key(scheme, host, port)
    data = _load_json(GOOD_PROXIES_FILE)
    data[key] = {"ts": time.time(), "uses": data.get(key, {}).get("uses", 0) + 1}
    _save_json(GOOD_PROXIES_FILE, data)
    _log(f"  [proxy-cache] good: {key}", verbose)


def _load_good_proxies():
    """Return list of (scheme, host, port) sorted by most recent success."""
    data = _load_json(GOOD_PROXIES_FILE)
    result = []
    for key, meta in sorted(data.items(), key=lambda x: -x[1].get("ts", 0)):
        try:
            scheme, rest = key.split("://", 1)
            host, port_s = rest.rsplit(":", 1)
            result.append((scheme, host, int(port_s)))
        except Exception:
            pass
    return result


# ---------------------------------------------------------------------------
# Browser / Playwright
# ---------------------------------------------------------------------------

def _maybe_reexec_xvfb():
    """On Linux non-headless with no DISPLAY, re-exec via xvfb-run for a fresh virtual display."""
    import platform, shutil, subprocess
    if platform.system() != "Linux":
        return
    if os.environ.get("DISPLAY"):
        return  # display already set — nothing to do
    # On Linux: default is non-headless. Only skip if user explicitly passed --headless.
    # (There's no --headless flag in the argparser, so we re-exec whenever no DISPLAY is set.)
    xvfb = shutil.which("xvfb-run")
    if not xvfb:
        return  # xvfb-run not installed — let it fail naturally
    # Re-exec: xvfb-run -a <python> <this script> <args>
    # xvfb-run -a auto-allocates a free display number and sets DISPLAY+XAUTHORITY
    cmd = [xvfb, "-a", sys.executable] + sys.argv
    sys.exit(subprocess.call(cmd))


def get_jina_key(headless=True, timeout=90, verbose=False):
    from patchright.sync_api import sync_playwright

    _log(f"Starting patchright (headless={headless})", verbose)

    if not headless:
        _log(f"  DISPLAY={os.environ.get('DISPLAY', '(not set)')}", verbose)

    captured_keys = []
    captured_tokens = []
    rate_limited = False

    with sync_playwright() as p:
        import platform
        browser_args = [
            "--disable-blink-features=AutomationControlled",
            "--window-size=1920,1080",
        ]
        if platform.system() == "Linux":
            # Required on Linux servers / Docker / CI to prevent sandbox/shm crashes.
            # Always use SwiftShader on Linux — Xvfb has no real GPU so GL init would
            # crash without software rendering fallback.
            browser_args += [
                "--no-sandbox",
                "--disable-setuid-sandbox",
                "--disable-dev-shm-usage",
                "--use-angle=swiftshader",
                "--enable-webgl",
                "--ignore-gpu-blocklist",
                "--enable-unsafe-swiftshader",
            ]
        elif headless:
            # macOS headless: SwiftShader for WebGL (GPU not available in headless).
            # Not used for macOS non-headless — "Google SwiftShader" renderer string
            # is detected as a VM fingerprint; real GPU is better there.
            browser_args += [
                "--use-angle=swiftshader",
                "--enable-webgl",
                "--ignore-gpu-blocklist",
                "--enable-unsafe-swiftshader",
            ]
        _log(f"Launching chromium headless={headless} args={browser_args}", verbose)
        browser = p.chromium.launch(headless=headless, args=browser_args)
        ctx = browser.new_context(viewport={"width": 1920, "height": 1080})

        # --- Event listeners ---
        def _on_console(msg):
            # Filter noisy WebGL driver messages
            t = msg.text
            if "GL Driver" in t or "GPU stall" in t or "WebGL" in t:
                return
            _log(f"  [console:{msg.type}] {t[:120]}", verbose)

        def _on_page_error(err):
            _log(f"  [page-error] {err}", verbose)

        _INTERESTING = ("keygen.jina.ai", "jina.ai/api", "api.jina.ai", "dash.jina.ai",
                        "challenges.cloudflare.com", "auth.")

        def _on_request(req):
            url = req.url
            if any(x in url for x in _INTERESTING):
                _log(f"  [req] {req.method} {url[:120]}", verbose)

        def _on_response(resp):
            url = resp.url
            if any(x in url for x in _INTERESTING):
                _log(f"  [resp] {resp.status} {url[:120]}", verbose)

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

            try:
                _log(f"  [keygen] fetching {url}", verbose)
                resp = route.fetch(url=url)
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
                    route.continue_(url=url)
                except Exception as e2:
                    _log(f"  [keygen] continue error: {e2}", verbose)

        ctx.route("**/keygen.jina.ai/**", on_route)
        page = ctx.new_page()

        page.on("console", _on_console)
        page.on("pageerror", _on_page_error)
        if verbose:
            page.on("request", _on_request)
            page.on("response", _on_response)

        _log("Navigating to https://jina.ai/?newKey ...", verbose)
        try:
            # networkidle ensures React/Quasar fully initializes before we proceed
            page.goto("https://jina.ai/?newKey", wait_until="networkidle", timeout=20000)
            _log(f"Navigation done, URL={page.url}", verbose)
        except Exception as e:
            _log(f"Navigation warning (continuing): {e}", verbose)

        _log("Sleeping 2s after networkidle...", verbose)
        time.sleep(2)

        # Dismiss cookie banner — required for Turnstile to fire
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
            _log(f"  cookie banner: {result}", verbose)
        except Exception as e:
            _log(f"  cookie banner error (non-fatal): {e}", verbose)

        # Main wait loop
        _log(f"Waiting up to {min(timeout, 30)}s for key/token...", verbose)
        deadline = time.time() + min(timeout, 30)
        last_log = 0

        while time.time() < deadline:
            if captured_keys:
                key = captured_keys[0]
                _log(f"Validating intercepted key {key[:12]}...{key[-4:]}", verbose)
                browser.close()
                return _validate_key(key, verbose)

            if rate_limited and captured_tokens:
                # Rate limited: close browser immediately (browser /empty fires after close,
                # but context will be disposed by then — can't capture it that way).
                # Try direct /empty (different rate limit bucket), then proxy replay.
                _log("Rate limited — closing browser and trying /empty directly...", verbose)
                browser.close()
                token = captured_tokens[-1]
                key = _try_direct_empty(token, verbose)
                if key and _is_good(key):
                    try:
                        return _validate_key(key, verbose)
                    except RuntimeError:
                        pass
                _log("Switching to proxy replay", verbose)
                return _replay_via_proxies(token, verbose=verbose)

            key = _extract_dom(page, verbose)
            if _is_good(key):
                _log(f"Returning DOM key {key[:12]}...{key[-4:]}", verbose)
                browser.close()
                return key

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


# ---------------------------------------------------------------------------
# DOM extraction
# ---------------------------------------------------------------------------

def _extract_dom(page, verbose=False) -> str:
    """Check input values, innerText, localStorage for jina_ key."""
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
            if _is_good(key):
                _log(f"  [dom] found valid key in {src}", verbose)
            else:
                _log(f"  [dom] found rejected key in {src}: {key[:20]}...", verbose)
            return key
        return ""
    except Exception as e:
        _log(f"  [dom] eval error: {e}", verbose)
        return ""


# ---------------------------------------------------------------------------
# Proxy replay
# ---------------------------------------------------------------------------

PROXY_WORKERS = 16  # parallel workers for fresh proxies
PROXY_MAX = 150     # max proxies to attempt total


def _try_one_proxy(scheme, host, port, turnstile_token, verbose, stop_event, label=""):
    """Try a single proxy; return key string or None. Records to bad/good cache."""
    if stop_event.is_set():
        return None
    pk = _proxy_key(scheme, host, port)
    # Quick connect test
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(3)
        s.connect((host, port))
        s.close()
    except Exception as e:
        reason = str(e)[:60]
        _log(f"  unreachable {pk}: {reason}", verbose)
        _record_bad_proxy(scheme, host, port, f"connect: {reason}", verbose=False)
        return None

    if stop_event.is_set():
        return None
    _log(f"  {label}trying {pk}...", verbose)
    try:
        if scheme == "socks5":
            result = _keygen_via_socks5(host, port, turnstile_token, verbose)
        else:
            result = _keygen_via_http_connect(host, port, turnstile_token, verbose)
        if _is_good(result):
            try:
                result = _validate_key(result, verbose)
                _log(f"  SUCCESS via {pk}: {result[:12]}...{result[-4:]}", verbose)
                _record_good_proxy(scheme, host, port, verbose)
                return result
            except RuntimeError as ve:
                reason = str(ve)
                _log(f"  {pk}: key rejected — {reason}", verbose)
                _record_bad_proxy(scheme, host, port, reason, verbose=False)
                return None
        reason = "no valid key in response"
        _log(f"  {pk}: {reason}", verbose)
        _record_bad_proxy(scheme, host, port, reason, verbose=False)
        return None
    except Exception as e:
        reason = str(e)[:80]
        _log(f"  {pk} failed: {reason}", verbose)
        _record_bad_proxy(scheme, host, port, reason, verbose=False)
        return None


def _replay_via_proxies(turnstile_token, verbose=False):
    """Replay keygen.jina.ai/trial POST through proxies (parallel workers)."""
    import random

    bad_set = _load_bad_proxies()
    good_proxies = _load_good_proxies()
    _log(f"Proxy cache: {len(good_proxies)} good, {len(bad_set)} bad", verbose)

    _log("Fetching fresh proxy list...", verbose)
    fresh = _fetch_proxies(verbose)

    good_keys = {_proxy_key(*g) for g in good_proxies}
    fresh_filtered = [p for p in fresh if _proxy_key(*p) not in good_keys and _proxy_key(*p) not in bad_set]
    random.shuffle(fresh_filtered)

    _log(f"Strategy: {len(good_proxies)} good (sequential) → {len(fresh_filtered)} fresh ({PROXY_WORKERS} workers, skip {len(bad_set)} bad)", verbose)

    stop = threading.Event()

    # 1. Good proxies first — tried sequentially (there are few of them)
    for scheme, host, port in good_proxies:
        if stop.is_set():
            break
        key = _try_one_proxy(scheme, host, port, turnstile_token, verbose, stop, label="[good] ")
        if key:
            stop.set()
            return key

    # 2. Fresh proxies in parallel
    candidates = fresh_filtered[:PROXY_MAX]
    _log(f"Starting {PROXY_WORKERS} parallel workers on {len(candidates)} fresh proxies...", verbose)

    found = threading.Event()
    result_holder = [None]

    def worker(proxy):
        scheme, host, port = proxy
        if stop.is_set():
            return
        key = _try_one_proxy(scheme, host, port, turnstile_token, verbose, stop)
        if key and not found.is_set():
            found.set()
            stop.set()
            result_holder[0] = key

    with ThreadPoolExecutor(max_workers=PROXY_WORKERS) as pool:
        futures = [pool.submit(worker, p) for p in candidates]
        for f in as_completed(futures):
            if found.is_set():
                break

    if result_holder[0]:
        return result_holder[0]

    raise RuntimeError(
        f"Tried up to {PROXY_MAX} proxies, all failed. "
        "Turnstile token may have expired (~5 min lifetime)."
    )


def _fetch_proxies(verbose=False):
    """Fetch proxy candidates from public lists."""
    results = []
    for url, scheme in PROXY_SOURCES:
        _log(f"  fetching [{scheme}] {url}", verbose)
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
            _log(f"  -> {len(results) - before} proxies", verbose)
        except Exception as e:
            _log(f"  source error: {e}", verbose)
    _log(f"Total proxy candidates: {len(results)}", verbose)
    return results


# ---------------------------------------------------------------------------
# Low-level keygen HTTP (proxy tunnels)
# ---------------------------------------------------------------------------

def _build_keygen_request(turnstile_token, endpoint="/trial"):
    boundary = "----WebKitFormBoundaryPythonProxy"
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="cf-turnstile-response"\r\n'
        f"\r\n"
        f"{turnstile_token}\r\n"
        f"--{boundary}--\r\n"
    ).encode()
    headers = (
        f"POST {endpoint} HTTP/1.1\r\n"
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


def _try_direct_empty(turnstile_token, verbose=False):
    """Try keygen.jina.ai/empty directly (no proxy). /empty may have looser rate limits."""
    _log("  [direct-empty] trying keygen.jina.ai/empty directly...", verbose)
    try:
        ctx = ssl.create_default_context()
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(8)
        s.connect(("keygen.jina.ai", 443))
        ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")
        ss.send(_build_keygen_request(turnstile_token, endpoint="/empty"))
        text = _read_response(ss)
        ss.close()
        return _parse_response(text, verbose)
    except Exception as e:
        _log(f"  [direct-empty] failed: {e}", verbose)
        return None


def _read_response(ss):
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
    _log(f"    CONNECT {proxy_host}:{proxy_port} -> keygen.jina.ai:443", verbose)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))
    s.send((
        f"CONNECT keygen.jina.ai:443 HTTP/1.1\r\nHost: keygen.jina.ai:443\r\n\r\n"
    ).encode())

    resp = b""
    while b"\r\n\r\n" not in resp:
        chunk = s.recv(4096)
        if not chunk:
            s.close()
            raise RuntimeError("CONNECT: no response")
        resp += chunk

    status = resp.decode("utf-8", errors="replace").split("\r\n")[0]
    _log(f"    CONNECT response: {status}", verbose)
    if "200" not in status:
        s.close()
        raise RuntimeError(f"CONNECT failed: {status}")

    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")
    _log(f"    TLS ok, sending POST...", verbose)
    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()
    return _parse_response(text, verbose)


def _keygen_via_socks5(proxy_host, proxy_port, turnstile_token, verbose=False):
    _log(f"    SOCKS5 {proxy_host}:{proxy_port} -> keygen.jina.ai:443", verbose)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((proxy_host, proxy_port))
    s.send(b"\x05\x01\x00")
    resp = s.recv(2)
    if resp != b"\x05\x00":
        s.close()
        raise RuntimeError("SOCKS5 auth rejected")
    target = b"keygen.jina.ai"
    s.send(b"\x05\x01\x00\x03" + bytes([len(target)]) + target + (443).to_bytes(2, "big"))
    resp = s.recv(10)
    if len(resp) < 2 or resp[1] != 0:
        s.close()
        raise RuntimeError(f"SOCKS5 connect failed (code={resp[1] if len(resp)>=2 else '?'})")
    _log(f"    SOCKS5 tunnel ok", verbose)
    ctx = ssl.create_default_context()
    ss = ctx.wrap_socket(s, server_hostname="keygen.jina.ai")
    _log(f"    TLS ok, sending POST...", verbose)
    ss.send(_build_keygen_request(turnstile_token))
    text = _read_response(ss)
    ss.close()
    return _parse_response(text, verbose)


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main():
    import platform as _plat
    # On Linux, default to non-headless (xvfb-run handles virtual display automatically).
    # Headless mode on Linux doesn't pass Turnstile reliably.
    _linux = _plat.system() == "Linux"
    _headless_default = not _linux

    # On Linux non-headless with no $DISPLAY: restart under xvfb-run to get a virtual display
    _maybe_reexec_xvfb()

    parser = argparse.ArgumentParser(description="Get Jina AI API key via undetected headless Chrome")
    parser.add_argument("--no-headless", dest="headless", action="store_false", default=_headless_default,
                        help="Show browser window (default: headless on macOS, non-headless on Linux)")
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
