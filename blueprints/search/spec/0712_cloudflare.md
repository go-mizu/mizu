# 0712: Cloudflare Account Automation — Lessons Learned

## Summary

Browser automation tool (Patchright + mail.tm) for Cloudflare account registration,
email verification, and Global API Key extraction. Located in
`tools/cloudflare/src/cloudflare_tool/`.

## Architecture

```
CLI (cli.py)
  ├── register       → register_via_browser()     → signup + account_id extraction
  ├── token get-key   → extract_global_api_key_via_browser() → login + profile check + key extraction
  ├── token verify-all → same flow for all accounts
  └── token create    → manual token add to store

Browser (browser.py)
  ├── _login_to_dashboard()           → fill form, user solves CAPTCHA, loop-check redirect
  ├── _extract_global_api_key_from_session() → /profile/api-tokens → View → Dialog 1 (code) → key
  └── register_via_browser()          → signup form + mail.tm verification

Store (store.py)  → DuckDB: accounts, tokens, workers, op_log
Email (email.py)  → mail.tm API: disposable mailbox + verification code polling
Client (client.py) → CF REST API wrapper
```

## Key Lessons

### 1. CF API `email_verified` is unreliable
- `/api/v4/user` returns `email_verified: False` even when the Profile page shows "Verified"
- **Fix**: Check verification by visiting `/profile` and looking for "Verified" text in DOM
- Never block on API `email_verified` — it causes false negatives

### 2. Login form: fill + wait for user CAPTCHA
- Patchright patches auto-solve Turnstile on signup page but NOT on login page
- **Fix**: Fill email + password quickly, print `>>> Solve CAPTCHA and click 'Log in' <<<`,
  then loop-check every 3s if URL redirected away from `/login`
- Must wait for `input[name="email"]` to be visible before filling (React SPA render delay)
- Use `_verify_form_values()` after fill to confirm values stuck

### 3. Form fill timing
- CF login page is a React SPA — inputs may not exist in DOM immediately after navigation
- `page.wait_for_load_state("domcontentloaded")` is not enough
- **Fix**: `page.locator('input[name="email"]').first.wait_for(state="visible", timeout=15000)`

### 4. Global API Key extraction flow (working)
1. Login → check Profile page for "Verified"
2. Navigate to `/profile/api-tokens`
3. Click first "View" button (Global API Key section)
4. Dialog 1 "Verify Your Identity": click "Send Verification Code", poll mail.tm for 6-8 digit code, fill + submit
5. Key revealed → read from clipboard via `navigator.clipboard.readText()` or DOM fallback
- Use `force=True` on dialog button clicks to bypass CF's `data-base-ui-inert` overlay (Base UI portal intercepts pointer events)

### 5. Creating API tokens via CF API fails for new accounts
- `POST /user/tokens` returns code 1211 "Please verify your email" even when email is verified on Profile
- **Fix**: Extract the Global API Key via browser UI instead, then use `X-Auth-Email` + `X-Auth-Key` headers for API calls

### 6. CF email verification via verify link
- Direct HTTP GET to verify link → 403 (CF managed challenge blocks non-browser)
- Verify link in logged-in context → instant redirect to dashboard (React SPA ignores token)
- Fresh context login from verify page → `email_verified` stays False in API
- **Conclusion**: Verification link flow is unreliable. Manual CAPTCHA solve on login page is the most reliable path.

### 7. mail.tm account lifecycle
- mail.tm password pattern: `Cf{first6charsOfLocalPart}!9xQ` (set during registration)
- mail.tm accounts are temporary — may expire after some time
- Auth: `POST /token` with address + password → JWT bearer token
- Polling: `GET /messages` → filter by `hydra:member`, match 6-8 digit code in subject/body

### 8. Turnstile detection
- Check for Turnstile widget FIRST: `iframe[src*="challenges.cloudflare.com"]` or `[name="cf-turnstile-response"]`
- If found, wait for token: `cf-turnstile-response` input value length > 10
- If no Turnstile found, check submit button state as fallback
- **Bug**: `!btn?.disabled` returns true when button doesn't exist — must check `btn !== null && !btn.disabled`

### 9. `button[type="submit"]` false matches
- On CF dashboard, `button[type="submit"]` can match the AI chatbot "Send message" button
- Prefer specific selectors: `button:has-text("Log in")`, `button:has-text("Verify code")`

## CLI Commands

```bash
# Register new account (opens browser, user solves signup CAPTCHA)
uv run cloudflare-tool register --no-headless --verbose

# Extract Global API Key (opens browser, user solves login CAPTCHA)
uv run cloudflare-tool token get-key --account <email> --no-headless --verbose

# List accounts
uv run cloudflare-tool account ls

# List tokens
uv run cloudflare-tool token ls
```

## Files

| File | Purpose |
|------|---------|
| `browser.py` | All browser automation (Patchright) |
| `cli.py` | Typer CLI commands |
| `store.py` | DuckDB local state (accounts/tokens/workers/op_log) |
| `email.py` | mail.tm API client |
| `client.py` | CF REST API wrapper |
