# jina-key

Get a free Jina AI API key automatically via undetected Chrome (patchright).

## How it works

1. Opens `jina.ai/?newKey` in a patched Chromium and waits for the page to fully load (`networkidle`)
2. Dismisses the cookie consent banner — required for Turnstile to fire
3. Route intercept captures the `keygen.jina.ai/trial` POST with the Cloudflare Turnstile token + key
4. If rate-limited (429 on `/trial`), tries `keygen.jina.ai/empty` directly (different rate-limit bucket)
5. If still rate-limited, replays the Turnstile token through free proxies:
   - Good proxies from `~/data/jina/good_proxies.json` tried **first** (sequential)
   - Then fresh proxies from public lists in **parallel** (16 workers)
   - Failed proxies saved to `~/data/jina/bad_proxies.json` and skipped next run

## Quick start (uv — recommended)

No manual install needed. [uv](https://github.com/astral-sh/uv) reads the inline `# /// script` header and auto-installs `patchright` in an isolated environment.

```bash
# Install uv once (if not already)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Run — downloads patchright + Chromium on first run (~150 MB), then gets key
uv run api_key.py

# Verbose mode — logs every step to stderr
uv run api_key.py --verbose

# Custom timeout
uv run api_key.py --timeout 120
```

Output: `KEY:<value>` on success, `ERROR:<reason>` to stderr + exit 1 on failure.

```bash
# Extract just the key
export JINA_API_KEY=$(uv run api_key.py | sed 's/^KEY://')
```

## Platform defaults

| Platform | Default mode | Notes |
|----------|-------------|-------|
| macOS | Headless | SwiftShader WebGL; use `--no-headless` to show window |
| Linux server | Non-headless (Xvfb) | Auto-starts virtual display via `xvfb-run -a`; requires `xvfb-run` installed |

**Linux requirement:** `xvfb-run` must be installed (`apt install xvfb`). Headless mode on Linux does not reliably pass Cloudflare Turnstile.

## Verbose log example (success on first try)

```
[17:51:09] args: headless=False timeout=90
[17:51:09] Starting patchright (headless=False)
[17:51:09]   DISPLAY=:100
[17:51:10] Launching chromium headless=False args=[...]
[17:51:11] Navigating to https://jina.ai/?newKey ...
[17:51:15] Navigation done, URL=https://jina.ai/?newKey
[17:51:15] Sleeping 2s after networkidle...
[17:51:17] Dismissing cookie banner...
  [req] POST https://keygen.jina.ai/trial
  [keygen] intercepted https://keygen.jina.ai/trial
  [keygen] captured turnstile token (1008 chars)
  [keygen] response status=201 body={"api_key":"jina_8ddd..."}
  [keygen] captured key jina_8ddd529...snHf
[17:51:17]   cookie banner: clicked-reject
[17:51:17] Waiting up to 30s for key/token...
[17:51:19] Validating intercepted key jina_8ddd529...snHf
  [balance] trial=10000000 total=10000000 trial_end=2036-03-08
[17:51:20] Key valid: trial_balance=10000000 total_balance=10000000
KEY:jina_8ddd5291e9294163bc7c23039c6df932_tSM_NbpJ_8OZ_5FgKGsIQY8snHf
```

## Verbose log example (proxy replay)

```
[17:54:22] Rate limited — closing browser and trying /empty directly...
  [direct-empty] trying keygen.jina.ai/empty directly...
  [direct-empty] failed: rate limited
[17:54:23] Switching to proxy replay
[17:54:23] Proxy cache: 1 good, 0 bad
[17:54:23] Fetching fresh proxy list...
[17:54:24] Strategy: 1 good (sequential) → 3634 fresh (16 workers, skip 0 bad)
  [good] trying http://138.124.53.25:7443...
    CONNECT 138.124.53.25:7443 -> keygen.jina.ai:443
    [HTTP/1.1 201 Created] body={"api_key":"jina_69ef..."}
  SUCCESS via http://138.124.53.25:7443: jina_69ef34e...ahuZ
  [balance] trial=10000000 total=10000000 trial_end=2036-03-08
KEY:jina_69ef34ed12214c1a8beeed3b35b74d05gbjGQJDrW1OlKUZfq8TtrCAkahuZ
```

## Proxy cache

On every run, the proxy cache at `~/data/jina/` is updated:

| File | Contents |
|------|----------|
| `good_proxies.json` | Proxies that returned a valid key — tried first next run |
| `bad_proxies.json` | Proxies that failed (auto-expires after 24h) |

Bad entries auto-expire after 24 hours so proxies that may recover are retried.

## Flags

| Flag | Default | Effect |
|------|---------|--------|
| _(none)_ | | macOS: headless; Linux: non-headless with auto Xvfb |
| `--verbose` / `-v` | | Log every step to stderr |
| `--no-headless` | | Show browser window (macOS); same as default on Linux |
| `--timeout N` | 90 | Max seconds to wait for browser to capture token |

## Rate limiting

Jina AI rate-limits trial key requests per IP. If you hit the limit, the script automatically:
1. Tries `keygen.jina.ai/empty` directly (sometimes has a different quota)
2. Falls back to proxy replay through free public proxies

The `~/data/jina/good_proxies.json` cache ensures working proxies are reused across runs. Rate limits typically reset within a few hours.

## First run

patchright downloads a patched Chromium binary on first run (~150 MB):

```
Downloading patchright (39.1MiB)
Downloaded patchright
```

## Manual install (without uv)

```bash
pip install patchright
python api_key.py --verbose
```
