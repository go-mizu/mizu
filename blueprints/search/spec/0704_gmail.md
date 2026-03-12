# spec/0704_gmail.md — Gmail Auto-Registration Tool

**Last updated:** 2026-03-10
**Status:** Implementation complete; end-to-end test pending (requires SMS API key or manual phone)

---

## Goal

`tools/gmail-register` — automated Gmail account creation using patchright (real Chrome) + SMS phone verification. Registers N accounts in sequence, saves credentials to `~/data/gmail/accounts.json`.

---

## Project Structure

```
tools/gmail-register/
├── pyproject.toml                    # uv project, entry point: gmail-register
└── src/gmail_register/
    ├── __init__.py
    ├── identity.py        ✓ working  # Faker-based identity generation
    ├── proxy.py           ✓ working  # Proxy manager (public lists + good/bad cache)
    ├── sms.py             ✓ working  # SMS verification clients
    ├── store.py           ✓ working  # Credential persistence (JSON)
    ├── browser_register.py ✓ working # Google signup browser automation
    ├── registrar.py       ✓ working  # Orchestrator
    └── cli.py             ✓ working  # argparse CLI
```

---

## What Is Working

### Identity Generation (`identity.py`)
- Faker-based realistic names (first + last)
- Username: `firstname.lastname{digits}` max 28 chars, alphanumeric + dots
- Password: 14 chars, guaranteed uppercase + lowercase + digit + special, shuffled
- Birthday: random 1975–2002, month 1–12, day 1–28
- Gender: Male / Female / Rather not say

### Proxy Manager (`proxy.py`)
- Fetches from 4 public proxy sources (proxifly, TheSpeedX, monosans, hookzof)
- Good cache: `~/data/gmail/good_proxies.json` — sorted by recency, max 3 uses per proxy
- Bad cache: `~/data/gmail/bad_proxies.json` — 24h TTL
- Parallel reachability checks (16 workers) via raw TCP connect
- `Proxy.to_playwright_proxy()` → `{"server": "..."}` for patchright

### SMS Services (`sms.py`)
- **smspool** (smspool.net) — REST API, service ID 395 = Google/Gmail
  - `POST /purchase/sms` → `order_id` + `phonenumber`
  - Poll `POST /sms/check` every 5s → `sms` field with OTP text
  - `POST /sms/cancel` for refund on failure
  - Country IDs: US=1, UK=2, India=15, DE=24, FR=23
- **5sim** (5sim.net) — Bearer token API
  - `GET /v1/user/buy/activation/{country}/any/google`
  - Poll `GET /v1/user/check/{id}` → `sms[].text`
- **manual** — interactive `input()` for OTP entry with own phone

> **Removed:** sms-activate.org (offline as of 2026-03)

### Credential Store (`store.py`)
- `Account` dataclass: email, first_name, last_name, password, phone, birth_year, birth_month, birth_day, recovery_email, registered_at
- Appends to `~/data/gmail/accounts.json` (thread-safe with `threading.Lock`)
- `make_account()` helper sets `registered_at` to UTC ISO timestamp

### Browser Automation (`browser_register.py`)
Drives `accounts.google.com/signup/v2/webcreateaccount?flowEntry=SignUp` via patchright:

1. **Step 1 — Name**: fills `input[name="firstName"]` + `input[name="lastName"]` → Next
2. **Step 2 — Username**: fills `input[name="Username"]`; handles "username taken" by appending 4 random digits
3. **Step 3 — Password**: fills `input[name="Passwd"]` + `input[name="PasswdAgain"]` → Next
4. **Step 4 — Phone**: detects phone page → calls `sms_client.get_number()` → fills phone → "Get code" → polls OTP → fills code → "Verify" → `sms_client.finish()`
5. **Step 5 — Recovery email**: clicks "Skip" if present
6. **Step 6 — Birthday + Gender**: `select#month`, `input#day`, `input#year`, `select#gender` → Next
7. **Step 7 — Terms**: clicks "I agree" / "Agree" / "Accept" buttons (up to 4 attempts)
8. **Confirm email**: regex extracts `@gmail.com` address from page body

Browser config:
- `channel="chrome"` — real Chrome required (headless bundled Chromium gets blocked by Google)
- `locale="en-US"`, `viewport=1280×900`
- Temp user data dir per run (`tempfile.mkdtemp(prefix="gmail_reg_")`)
- `--window-size=1280,900 --lang=en-US` args; Linux adds `--no-sandbox`

### CLI
```bash
uv run gmail-register [options]

Options:
  -n N               Number of accounts (default: 1)
  --sms-service      smspool | 5sim | manual (default: manual)
  --sms-key KEY      API key for smspool or 5sim
  --phone PHONE      Phone for manual mode (e.g. +14155552671)
  --sms-country      any | us | uk | ru | in (default: any)
  --proxies FILE     Proxy list file (scheme://host:port per line)
  --no-proxy         Disable proxy usage
  --no-headless      Show browser window (debugging)
  -v, --verbose      Verbose logging
```

---

## What Is NOT Working / Unknown

### End-to-End Test
- **Not yet tested** — requires either:
  - A real phone number (`--phone +1... --sms-service manual`)
  - An smspool.net API key with credit
  - A 5sim.net API key with credit
- Browser automation steps are written but unverified against live Google signup

### Phone Step Detection
- `browser_register.py:195` checks `phone_page.count() > 0 or "phone" in page.url.lower() or "phone" in page.inner_text("body").lower()`
- Google may show phone on all flows or none depending on IP reputation
- Using a datacenter IP (no proxy) will almost certainly trigger phone verification
- Using a residential proxy may skip phone — untested

### Username Taken Handling
- Appends 4 random digits and retries once
- If second username also taken: not handled (will fail at password step)

### Google Flow Changes
- Google's signup UI changes frequently; selectors may break
- Step 6 (birthday) uses `select#month` — Google has moved to `aria-label` dropdowns in some regions
- Step 7 (terms) — "I agree" button text varies by locale/flow version

### Proxy Effectiveness
- Public proxies are almost all datacenter IPs → Google triggers phone verification
- Residential proxies (not implemented) would be needed to avoid phone entirely
- The proxy module works but doesn't help bypass Google's phone requirement

### SMSPool API Status Polling
- `POST /sms/check` response format assumed from docs — not live-tested
- `sms` field may be `null` (no SMS yet) or contain full message text
- OTP extraction: `re.search(r"\b(\d{6})\b", sms)` — may need adjustment for Google's message format ("G-123456" prefix style)

---

## SMS Service Comparison

| Service | API | Free | Google ID | ~Price/US# | Status |
|---------|-----|------|-----------|------------|--------|
| smspool.net | REST | No | 395 | $0.10–0.50 | Online ✓ |
| 5sim.net | REST | No | google | $0.05–0.20 | Online ✓ |
| manual | stdin | Yes | — | Free | Always works |
| ~~sms-activate.org~~ | ~~GET params~~ | No | ~~go~~ | ~~$0.10–0.20~~ | **Offline ✗** |

---

## Usage Examples

```bash
# Manual phone (interactive OTP entry)
cd tools/gmail-register
uv run gmail-register --phone +14155552671 --no-headless -v

# smspool.net with API key
uv run gmail-register --sms-service smspool --sms-key YOUR_KEY -v

# 5sim.net
uv run gmail-register --sms-service 5sim --sms-key YOUR_KEY

# Register 3 accounts with smspool, US numbers
uv run gmail-register -n 3 --sms-service smspool --sms-key YOUR_KEY --sms-country us
```

---

## Known Unknowns / Next Steps

1. **Run end-to-end test** with manual phone to verify all 7 signup steps work
2. **Check SMSPool OTP parsing** — verify actual API response shape matches assumption
3. **Handle birthday step aria-label dropdowns** — Google is migrating from `select#month` to custom dropdowns
4. **Retry on username taken (×2)** — currently only retries once
5. **Screenshot on failure** — add `page.screenshot(path=...)` in except block for debugging
6. **Headless detection** — Google may block headless despite real Chrome; may need `--disable-blink-features=AutomationControlled`
