# spec/0703 — X (Twitter) Account Auto-Registrar

## Goal

Auto-register N X accounts using mail.tm for email verification and patchright for
browser automation (JS instrumentation). Save credentials to `~/data/x/accounts.json`.
Verify each account by tweeting "hello, world!".

## Tool Location

`tools/x-register/` — standalone `uv` Python project with `src/` layout.

## Usage

```bash
# Register 1 account (default)
uv run x-register

# Register 5 accounts with proxy list
uv run x-register --count 5 --proxies proxies.txt

# With Arkose solver (CapSolver)
uv run x-register --count 3 --solver-key YOUR_KEY

# Verbose with visible browser
uv run x-register --count 1 --verbose --no-headless
```

## Architecture

```
tools/x-register/
├── pyproject.toml
└── src/x_register/
    ├── __init__.py
    ├── cli.py          # argparse entry: --count, --proxies, --solver-key, --verbose, --no-headless
    ├── registrar.py    # orchestrates one registration end-to-end
    ├── browser.py      # patchright: fetch js_instrumentation token
    ├── email.py        # mail.tm API: create mailbox, poll for OTP
    ├── identity.py     # faker: realistic name/username/birthday/password
    ├── proxy.py        # public proxy lists, good/bad JSON cache
    ├── twitter_api.py  # curl_cffi: guest token, onboarding tasks, tweet
    ├── captcha.py      # optional Arkose solver (CapSolver / 2captcha)
    └── store.py        # ~/data/x/accounts.json persistence
```

**Data dir:** `~/data/x/`
- `accounts.json` — registered account credentials
- `good_proxies.json` — proxies that worked (sorted by recency)
- `bad_proxies.json` — proxies that failed (pruned after 24h)

## Registration Flow

```
1. identity.py    Generate realistic name, username, birthday, password
2. email.py       Create mail.tm mailbox → address + bearer token
3. browser.py     Launch patchright → fetch js_instrumentation from x.com/i/js_inst
4. proxy.py       Pick proxy (good cache → fresh list → file proxies)
5. twitter_api.py POST /guest/activate.json → guest_token
6. twitter_api.py POST /onboarding/task.json?flow_name=signup → flow_token
              ↳ If ArkoseEmail subtask → captcha.py solve (or skip if no solver)
7. twitter_api.py POST /onboarding/begin_verification.json → triggers OTP email
8. email.py       Poll /messages until 6-digit OTP arrives (120s timeout)
9. twitter_api.py POST /onboarding/task.json [Signup + EmailVerification subtasks]
10. twitter_api.py POST /onboarding/task.json [EnterPassword subtask]
11. twitter_api.py POST remaining steps (skip avatar, skip bio, follow @elonmusk)
12.               Extract auth_token + screen_name from final response cookies
13. twitter_api.py POST CreateTweet "hello, world!" via GraphQL
14. store.py      Save {email, password, username, auth_token, tweet_id} to accounts.json
```

## Twitter Internal API

All requests use `curl_cffi` with `impersonate="chrome136"` for TLS fingerprinting.

**Stable Bearer token (public, same across all X clients):**
```
AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
```

**Key endpoints:**
| Method | URL | Purpose |
|--------|-----|---------|
| POST | `https://api.x.com/1.1/guest/activate.json` | Get guest token |
| GET | `https://x.com/i/js_inst?c_name=ui_metrics` | JS instrumentation blob |
| POST | `https://api.x.com/1.1/onboarding/task.json?flow_name=signup` | Start signup flow |
| POST | `https://api.x.com/1.1/onboarding/begin_verification.json` | Send OTP to email |
| POST | `https://api.x.com/1.1/onboarding/task.json` | Submit subtasks |
| POST | `https://twitter.com/i/api/graphql/.../CreateTweet` | Post tweet |

**Required headers:**
```python
{
    "authorization": f"Bearer {BEARER}",
    "x-twitter-active-user": "yes",
    "x-twitter-client-language": "en",
    "x-guest-token": guest_token,       # before auth
    "x-csrf-token": ct0_cookie,         # always (from ct0 cookie)
    "content-type": "application/json",
}
```

## mail.tm API

Base URL: `https://api.mail.tm`

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/domains` | List available domains |
| POST | `/accounts` | Create mailbox `{address, password}` |
| POST | `/token` | Get Bearer token `{address, password}` |
| GET | `/messages` | Poll inbox (Bearer required) |

OTP extraction: `re.search(r'\b(\d{6})\b', message["intro"] + message.get("text",""))`

## Proxy Module

Port of `tools/jina-key/api_key.py` proxy pattern, refactored as `ProxyManager` class:

- **Sources:** 4 public HTTPS/HTTP/SOCKS5 lists (same as jina-key)
- **Cache:** `good_proxies.json` (sorted by recency, skip if `uses >= 3`), `bad_proxies.json` (24h TTL)
- **Strategy:** good proxies first (sequential) → fresh list (parallel, 16 workers) → file proxies
- **Format for curl_cffi:** `{"https": "http://host:port"}` or `{"https": "socks5://host:port"}`

## Identity Generation

Uses `faker` library:
- `display_name`: `fake.name()` → "Sarah Johnson"
- `username`: `fake.user_name()[:15]` with random suffix if needed for uniqueness
- `email_local`: `fake.user_name() + str(random.randint(10,99))`
- `password`: 12+ chars, upper+lower+digit+special
- `birthday`: year 1975–2000, month 1–12, day 1–28

## Arkose / FunCaptcha

Public key: `2CB16598-CB82-4CF7-B332-5990DB66F3AB`

- If `ArkoseEmail` subtask appears and `--solver-key` is set → call CapSolver API
- If no solver key → log warning and attempt to continue (X sometimes skips Arkose on residential IPs)
- Solver services supported: `capsolver` (default), `2captcha`

## Output Format

`~/data/x/accounts.json` — append-only list:
```json
[
  {
    "email": "user42@indigobook.com",
    "email_password": "mailtm-password",
    "display_name": "Sarah Johnson",
    "username": "sarahjohnson87",
    "password": "MyPass123!",
    "auth_token": "abc123...",
    "ct0": "csrf-token",
    "user_id": "1234567890",
    "tweet_id": "9876543210",
    "registered_at": "2026-03-10T12:34:56Z"
  }
]
```

## Error Handling

- **Arkose without solver:** log warning, mark attempt as partial success (no tweet)
- **OTP timeout (120s):** retry mail.tm mailbox with different address
- **Proxy failure:** mark bad, pick next proxy
- **Rate limit (429):** rotate proxy, retry
- **Phone verification required:** log and skip (cannot automate without SMS service)
