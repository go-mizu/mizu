# 0547: AI Worker Auto-Registration & Account Management

## Overview

Port the Go perplexity package's auto-registration system (`pkg/dcrawler/perplexity/register.go`, `email.go`, `accounts.go`) to the CF Worker AI app (`tools/ai/`). The worker **automatically** creates Perplexity accounts using disposable emails in the **background**, then uses those accounts for pro/reasoning/deep research modes. **No public API exposure** — everything happens internally via `ctx.waitUntil()`.

## Background

The Go perplexity package already implements:
- **7 temp email providers** across 3 tiers (private/session/public)
- **Auto-registration** via magic link email verification
- **Account management** with round-robin rotation, exhaustion tracking
- **Session persistence** with cookies + CSRF tokens

The CF Worker currently supports:
- **Anonymous scraping** via SSE endpoint (auto mode only)
- **Official API** via `PERPLEXITY_API_KEY` (all modes)
- **Session pooling** in memory + KV with 429 rotation

Gap: No way to use pro/reasoning/deep modes without an API key.

## Architecture

### Background Registration Flow

Registration runs via `ctx.waitUntil()` — **no public endpoints**. Triggered automatically when:
1. A pro/reasoning/deep search is requested and no active accounts exist
2. An account is exhausted and active count drops below threshold
3. The `/api/warm` pre-warm endpoint detects no accounts

```
  User request (mode=pro)
         │
         ▼
  getSessionForMode(kv, mode, ctx)
         │
    ┌────▼─────────────────────────┐
    │ AccountManager.nextAccount() │
    │ → active account exists?     │
    └──┬───────────┬───────────────┘
       │yes        │no
       ▼           ▼
  Use account   ctx.waitUntil(backgroundRegister(kv))
  session       Fall back to anonymous session
                Registration happens in background:
                  1. Acquire lock (prevent concurrent)
                  2. Init anonymous session (CSRF)
                  3. Create temp email (tiered fallback)
                  4. POST /api/auth/signin/email
                  5. Poll inbox (3s interval, 25s timeout)
                  6. GET magic link callback + follow redirects
                  7. Save account to KV
                  8. Release lock
                  9. Log every step to accounts:log
```

### Search Flow with Accounts

```
GET /api/stream?q=...&mode=pro
         │
         ▼
    ┌─────────────────────────────────┐
    │ mode == 'auto'?                 │
    │   → Use anonymous session pool  │
    │ mode == 'pro'|'reasoning'|'deep'│
    │   → AccountManager.nextAccount()│
    │     → Load session cookies      │
    │     → Execute search            │
    │     → RecordUsage (pro_queries) │
    │     → If exhausted, rotate      │
    │     → If low accounts, bg reg   │
    └─────────────────────────────────┘
```

### KV Storage Layout

```
account:{id}        → Account JSON (email, session{csrfToken,cookies}, proQueries, status)
accounts:index      → AccountSummary[] (id, email, proQueries, status, createdAt)
accounts:robin      → number (round-robin pointer)
accounts:log        → RegistrationLog[] (last 50 entries, full debug info)
accounts:lock       → ISO timestamp (60s TTL, prevents concurrent registration)
```

## Email Providers (6 total)

Ported from Go, tried in tier order with shuffle within each tier:

| Provider | Tier | Auth | API |
|----------|------|------|-----|
| mail.tm | 1 (Private) | JWT | REST /domains, /accounts, /token, /messages |
| mail.gw | 1 (Private) | JWT | Same API as mail.tm, different domains |
| tempmail.lol | 1 (Private) | token | REST /generate, /auth/{token} |
| guerrillamail | 2 (Session) | sid_token | REST ajax.php?f=... |
| dropmail | 2 (Session) | GraphQL session | GraphQL /api/graphql/web-test-2 |
| tempmail.plus | 3 (Public) | None | REST /api/mails?email=... |

Each provider implements:
```typescript
interface TempEmailClient {
  email(): string
  waitForMessage(subject: string, timeoutMs: number): Promise<string>
}
```

## Debug Logging

Every registration step is logged to `accounts:log` in KV (last 50 entries):

```typescript
interface RegistrationLog {
  timestamp: string
  event: 'start' | 'email_created' | 'signin_sent' | 'email_received' | 'auth_complete' | 'account_saved' | 'error'
  message: string         // Human-readable description
  provider?: string       // Email provider name
  email?: string          // Generated email address
  accountId?: string      // Assigned account ID
  durationMs?: number     // Time since registration start
  error?: string          // Error message + stack trace
}
```

Read logs via wrangler CLI:
```bash
npx wrangler kv:key get --binding=KV accounts:log
npx wrangler kv:key get --binding=KV accounts:index
npx wrangler kv:key get --binding=KV account:{id}
```

## CF Worker Constraints

- **30s execution limit**: Registration runs via `waitUntil()` (same 30s limit)
  - Email creation: ~1-3s
  - Signin request: ~1-2s
  - Email polling: ~5-15s (most providers deliver within 10s)
  - Magic link: ~1-2s
  - Total: ~10-22s (fits within limit)
- **No utls/TLS fingerprinting**: CF Worker `fetch` handles TLS natively
  - Chrome headers still set for Cloudflare WAF bypass
- **No DuckDB**: KV for account storage (simpler, sufficient for <100 accounts)
- **No filesystem**: KV replaces session file persistence
- **Concurrent protection**: Lock in KV prevents overlapping registrations

## File Structure

```
tools/ai/src/
├── email.ts          # NEW: 6 temp email providers (TempEmailClient interface)
├── register.ts       # NEW: backgroundRegister() — runs via waitUntil()
├── accounts.ts       # NEW: AccountManager — KV CRUD + rotation + logging
├── config.ts         # EXISTING: Already has MAGIC_LINK_REGEX, SIGNIN_SUBJECT, signin endpoint
├── types.ts          # EXISTING: Already has Account, AccountSummary types
├── perplexity.ts     # EXISTING: Already accepts sessionOverride parameter
└── routes/
    └── api.ts        # UPDATED: Uses getSessionForMode() + recordAccountUsage()
```

## Account Lifecycle

1. **Created**: Background registration or future CLI sync
2. **Active**: Used for pro/reasoning/deep queries, `proQueries > 0`
3. **Exhausted**: `proQueries == 0`, skipped by rotation
4. **Failed**: HTTP errors or auth failures, skipped
5. Accounts stored with 30-day TTL in KV

## Pro Query Tracking

Each new account gets 5 pro queries (Perplexity free tier).
- After each pro/reasoning/deep search, decrement `proQueries` via `recordUsage()`
- When `proQueries == 0`, mark as `exhausted`
- AccountManager skips exhausted/failed accounts in round-robin
- If no active accounts remain, fall back to anonymous (auto mode) + trigger bg registration

## Security

- **No public account endpoints** — all management is internal
- No API keys or session tokens exposed in any response
- Session cookies stored in KV (CF encrypts at rest)
- Registration lock prevents abuse via concurrent requests
- The only externally visible signal is `accounts: { active, total }` in `/api/warm` response

## Implementation Notes

### Porting from Go

| Go Pattern | CF Worker Equivalent |
|------------|---------------------|
| `cookiejar.Jar` | Cookie string in KV |
| `utls.HelloChrome_Auto` | CF Worker `fetch` (no fingerprint needed) |
| DuckDB tables | KV key-value store |
| `sync.Mutex` | KV lock + single-threaded V8 isolate |
| `time.Sleep` polling | `setTimeout` + `await` polling |
| `os.ReadFile` session | `kv.get()` session |
| `context.Context` | `AbortSignal` |
| `fmt.Println` logging | KV-based `RegistrationLog[]` |

### Key Differences from Go

1. **Background execution**: Registration runs via `waitUntil()`, not as a blocking request
2. **Simpler auth**: CF Worker fetch from edge IPs — no TLS fingerprinting needed
3. **Shorter timeout**: 25s for email waiting (vs 120s in Go) due to CF 30s limit
4. **No public API**: Everything internal — debug via `wrangler kv:key get`
5. **KV instead of DuckDB**: Simpler storage, no SQL queries needed
6. **Automatic trigger**: Registration fires on-demand when accounts are needed
