# jina-key

Get a free Jina AI API key automatically via undetected headless Chrome (patchright).

## How it works

1. Opens `jina.ai/?newKey` in a patched Chromium that bypasses Cloudflare Turnstile bot detection
2. Intercepts the `keygen.jina.ai/trial` POST to capture the key and the Turnstile token
3. If rate-limited (429), replays the POST through free proxies using the captured Turnstile token
4. Falls back to checking the DOM (input values, page text, localStorage) if the network intercept misses

## Quick start (uv — recommended)

No manual install needed. [uv](https://github.com/astral-sh/uv) reads the inline `# /// script` header and auto-installs `patchright` in an isolated environment.

```bash
# Install uv once (if not already)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Run — downloads patchright + Chromium on first run, then gets key
uv run api_key.py

# Verbose mode — logs every step to stderr
uv run api_key.py --verbose

# Show the browser window (useful for debugging)
uv run api_key.py --no-headless --verbose

# Custom timeout (seconds)
uv run api_key.py --timeout 120 --verbose
```

The script prints `KEY:<value>` on success or `ERROR:<reason>` to stderr + exits 1 on failure.

## Output format

```
KEY:jina_abcdef1234567890abcdef1234567890...
```

Pipe it to grab just the key:

```bash
uv run api_key.py | sed 's/^KEY://'
```

Or save to env:

```bash
export JINA_API_KEY=$(uv run api_key.py | sed 's/^KEY://')
```

## Verbose log example

```
[12:34:01] Starting patchright (headless=True)
[12:34:01] Launching chromium with args: [...]
[12:34:02] Navigating to https://jina.ai/?newKey ...
[12:34:04] Navigation done, URL=https://jina.ai/?newKey
[12:34:04] Sleeping 3s for Turnstile to fire...
[12:34:07] Dismissing cookie banner...
[12:34:07] Waiting up to 30s for key/token...
[12:34:07]   [keygen] intercepted https://keygen.jina.ai/trial
[12:34:07]   [keygen] captured turnstile token (1847 chars)
[12:34:08]   [keygen] response status=200 body={"key":"jina_..."}
[12:34:08]   [keygen] captured key jina_abcdef12...ef90
[12:34:08] Returning intercepted key jina_abcdef12...ef90
KEY:jina_abcdef1234567890abcdef1234567890XYZabc
```

## Modes

| Flag | Effect |
|------|--------|
| _(none)_ | Headless Chrome, no verbose output |
| `--verbose` | Headless + full step logs to stderr |
| `--no-headless` | Visible Chrome window |
| `--no-headless --verbose` | Visible window + step logs (best for debugging) |
| `--timeout N` | Change max wait from 90s |

## First run

On first run, patchright will download a patched Chromium binary (~150 MB). Subsequent runs skip this.

```
Downloading Playwright build of chromium...
```

## Troubleshooting

**`Key not found after 90s`** — Cloudflare may have changed their Turnstile flow. Try:
- `--no-headless --verbose` to watch the browser in real time
- Increase timeout: `--timeout 180`

**`Tried 50 proxies, all failed`** — You were rate-limited and the proxy list was exhausted. Wait a few minutes and retry. The Turnstile token has ~5 min lifetime.

**Chromium missing on server** — patchright downloads its own Chromium. On headless Linux servers ensure you have X libraries or use the `--no-sandbox` args (already included by default in headless mode).

## Manual install (without uv)

```bash
pip install patchright
python api_key.py --verbose
```
