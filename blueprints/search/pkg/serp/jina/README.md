# Jina AI SERP Provider

Auto-registration and search client for Jina AI's search API (`s.jina.ai`).

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  CLI (cli/serp.go)                                           │
│                                                              │
│  search serp search "query"  ──→  Pick key from store        │
│  search serp signup jina     ──→  Register new key           │
│  search serp list            ──→  Show keys + balances       │
│  search serp rotate          ──→  Check & prune depleted     │
│  search serp install         ──→  Install python+patchright  │
│  search serp add-key jina K  ──→  Manual key add             │
└──────────┬───────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│  pkg/serp/jina/                                              │
│                                                              │
│  client.go    — Provider.Search() → POST s.jina.ai           │
│              — CheckBalance()    → GET dash.jina.ai          │
│  register.go  — registrar.Register() → runs embedded Python  │
│  get_key.py   — embedded via go:embed (patchright script)    │
│  store.go     — KeyStore (~/data/jina/keys.json)             │
└──────────────────────────────────────────────────────────────┘
```

## Key Storage

Keys are stored at `$HOME/data/jina/keys.json`:

```json
{
  "keys": [
    {
      "api_key": "jina_88b875f7a20b...",
      "created_at": "2026-03-08T23:17:00Z",
      "balance": 10000000,
      "checked_at": "2026-03-08T23:30:00Z"
    }
  ]
}
```

## Registration Flow

The registration uses a two-phase strategy to obtain keys with **10M tokens** (10-year trial):

```
Phase 1: Browser (patchright, direct connection)
┌─────────────┐       ┌──────────────┐       ┌─────────────────┐
│  patchright  │──────→│  jina.ai/    │──────→│  keygen.jina.ai │
│  (headless)  │       │  ?newKey     │       │  /trial         │
└─────────────┘       └──────────────┘       └────────┬────────┘
       │                                              │
       │  context.route() intercept captures:         │
       │  1. cf-turnstile-response token              │
       │  2. Redirects /empty → /trial                │
       │                                              ▼
       │                                    ┌──────────────────┐
       │                                    │  HTTP 201: key   │ ← direct success
       │                                    │  HTTP 429: rate  │ ← need proxy
       │                                    └──────────────────┘
       │
Phase 2: Proxy replay (if rate-limited)
       │
       ▼
┌─────────────────────────────────────────────────┐
│  Fetch ~3700 free SOCKS5/HTTP proxies           │
│  For each proxy:                                │
│    1. TCP connect test (3s timeout)             │
│    2. SOCKS5 handshake / HTTP CONNECT           │
│    3. TLS wrap (server_hostname verification)   │
│    4. Replay keygen POST with captured token    │
│    5. Parse response for jina_ key              │
└─────────────────────────────────────────────────┘
```

### Why patchright (not rod)?

Rod's CDP Fetch domain **cannot intercept** the Turnstile keygen request.
Tested: `page.HijackRequests()`, `browser.HijackRequests()`, CDP Network
events, JS fetch/XHR monkey-patching — none fire for `keygen.jina.ai`.

Patchright's Playwright-based `context.route()` intercepts at the browser
process level and captures all requests including cross-origin Turnstile
iframe traffic.

### Why proxy replay?

`keygen.jina.ai` rate-limits by IP (~5 requests). The Turnstile token
is NOT IP-bound — it can be generated on the local IP and replayed from
a proxy IP. This separates the browser (expensive) from the key generation
(cheap HTTP POST).

## Search API

```
POST https://s.jina.ai/
Authorization: Bearer jina_...
Content-Type: application/json

{"q": "search query", "num": 10}
```

Response: `{code: 200, data: [{title, url, description, content}, ...]}`

## Balance API

```
GET https://dash.jina.ai/api/v1/api_key/fe_user?api_key=jina_...
```

Response: `{wallet: {total_balance: 10000000, trial_balance: 10000000, ...}}`

## Usage

### Quick start

```bash
# Install dependencies (python3 + patchright + browser)
search serp install

# Search with auto-provisioning (gets a key if none stored)
search serp search "your query" --auto

# Search with JSON output
search serp search "your query" --json

# Manually add a key
search serp add-key jina jina_your_key_here
```

### Key management

```bash
# List all keys with cached balances
search serp list

# List with live balance refresh from API
search serp list --refresh

# Check balances and remove depleted keys
search serp rotate

# Register a new key explicitly
search serp signup jina --verbose
```

### Server deployment (headless Linux)

```bash
# Deploy binary
make build-on-server SERVER=2

# On server: install all dependencies
~/bin/search serp install

# On server: auto-provision and search
~/bin/search serp search "query" --auto
```

On Linux without a display (`$DISPLAY` unset), the Go wrapper automatically
uses `xvfb-run` to provide a virtual framebuffer. Turnstile requires a
visible browser context — headless Chrome cannot solve it.

Chrome flags added on Linux:
- `--no-sandbox` (required as root)
- `--use-angle=swiftshader` (software GL, prevents crash under xvfb)
- `--disable-dev-shm-usage` (avoids /dev/shm exhaustion)

## Dependencies

| Component | Purpose | Install |
|-----------|---------|---------|
| python3 | Runs patchright script | `apt install python3` |
| patchright | Undetected Playwright fork | `pip install patchright` |
| chromium | Browser for Turnstile solving | `python3 -m patchright install chromium` |
| system libs | libcups, libnss, etc. | `python3 -m patchright install-deps chromium` |
| xvfb | Virtual framebuffer (Linux) | `apt install xvfb` |

The `search serp install` command handles all of the above automatically.

## Files

| File | Description |
|------|-------------|
| `client.go` | Search API client + balance checker |
| `register.go` | Go wrapper: embeds Python script, runs via `exec.Command` |
| `get_key.py` | Embedded patchright script (two-phase key generation) |
| `store.go` | Key store (`~/data/jina/keys.json`) |
| `register_test.go` | Integration test (needs browser + network) |

## Key format

```
jina_[a-f0-9]{32}[a-zA-Z0-9_-]+
```

Example: `jina_88b875f7a20b403c82b601d429794037IWiTtQH5Ne7Q5qgpX1pclmpMDZrS`

Keys from `/trial` endpoint: 10M tokens, 10-year trial expiry.
