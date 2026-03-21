# 0773 — Bot Protection

## Objective

Protect Storage from automated abuse — mass registration, magic-link email
bombing, upload spam, and API-key farming — using a defense-in-depth approach
combining Cloudflare Turnstile, IP-based rate limiting, and bot-signal
heuristics.

## Current State

| Endpoint | Existing protection | Gap |
|---|---|---|
| `POST /auth/register` | None | Open to mass bot registration |
| `POST /auth/magic-link` | 1 email/min per address (D1) | No IP limit; bots can spray many addresses |
| `POST /oauth/authorize` (login) | None | Sends email, no rate limit |
| `POST /oauth/register` | None | Unlimited dynamic client registration |
| `POST /auth/challenge` | 5-min TTL | No IP rate limit |
| `POST /auth/verify` | Single-use challenge | No IP rate limit |
| `POST /files/uploads` | Auth required | No per-user/IP rate limit |
| `POST /auth/keys` | Auth required | No per-user rate limit |
| `POST /files/share` | Auth required | No per-user rate limit |

## Target State

Three layers of defense, applied per-endpoint:

### Layer 1 — Cloudflare Turnstile (browser endpoints)

Turnstile is Cloudflare's free, privacy-preserving CAPTCHA replacement. It
issues a cryptographic token client-side, validated server-side via
`https://challenges.cloudflare.com/turnstile/v0/siteverify`.

**Where applied:**
- Home page sign-in modal → `POST /auth/magic-link`
- OAuth login page → `POST /oauth/authorize` (action=login)

**Widget mode:** Managed (Cloudflare decides interactive vs. invisible).

**Client-side:** Load `challenges.cloudflare.com/turnstile/v0/api.js`, render
widget with `data-sitekey`, attach `cf-turnstile-response` token to form
submission.

**Server-side:** Validate token via siteverify before processing. Reject with
403 if validation fails. Check `hostname` and `action` fields in response.

**Env vars:** `TURNSTILE_SECRET_KEY` (secret, in worker secrets),
`TURNSTILE_SITE_KEY` (public, passed to pages).

**API/CLI callers** skip Turnstile — they authenticate via Ed25519 keys or API
tokens, which are already bot-resistant by design.

### Layer 2 — IP-based rate limiting (all sensitive endpoints)

A lightweight D1-backed rate limiter. Counts requests per
`(endpoint, key)` in a sliding window.

**Table:**
```sql
CREATE TABLE IF NOT EXISTS rate_limits (
  endpoint TEXT NOT NULL,
  key      TEXT NOT NULL,
  ts       INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rl_lookup ON rate_limits(endpoint, key, ts);
```

**Limits:**

| Endpoint | Key | Limit | Window |
|---|---|---|---|
| `POST /auth/register` | IP | 5 | 1 hour |
| `POST /auth/magic-link` | IP | 10 | 15 min |
| `POST /auth/challenge` | IP | 30 | 1 hour |
| `POST /auth/verify` | IP | 20 | 1 hour |
| `POST /oauth/register` | IP | 5 | 1 hour |
| `POST /oauth/authorize` | IP | 10 | 15 min |
| `POST /files/uploads` | actor | 200 | 1 hour |
| `POST /files/uploads/multipart` | actor | 50 | 1 hour |
| `POST /auth/keys` | actor | 10 | 1 hour |
| `POST /files/share` | actor | 50 | 1 hour |

**Cleanup:** 1% probabilistic cleanup of expired rows per request
(`DELETE FROM rate_limits WHERE ts < ?`).

**Response:** 429 Too Many Requests with `Retry-After` header.

### Layer 3 — Bot signal heuristics (unauthenticated endpoints)

Lightweight middleware checking freely available Cloudflare request properties
and HTTP headers for bot indicators:

1. **Datacenter ASN detection:** Flag requests from known cloud provider ASNs
   (AWS, GCP, Azure, DigitalOcean, OVH, Hetzner, Linode) — legitimate browser
   traffic rarely originates from these.

2. **User-Agent validation:** Flag missing, very short (<10 chars), or known
   automation tool signatures (curl, wget, python-requests, httpx, node-fetch).

3. **Browser header validation:** Flag requests missing `Accept-Language`,
   `Sec-Fetch-Mode`, or `Accept-Encoding` — real browsers always send these.

4. **`cf.botManagement.score`** (if available on the plan): Block requests with
   score < 10, challenge score < 30.

**Scoring:** Each signal adds to a suspicion score (0-100). Requests scoring
>60 are blocked with 403. Requests scoring 30-60 are allowed but logged.
API/Bearer-authenticated requests skip this check entirely.

## Endpoint Protection Matrix

| Endpoint | Turnstile | Rate Limit | Bot Guard |
|---|---|---|---|
| `POST /auth/register` | — | IP 5/hr | Yes |
| `POST /auth/magic-link` | Yes (browser) | IP 10/15m | Yes |
| `POST /auth/challenge` | — | IP 30/hr | — |
| `POST /auth/verify` | — | IP 20/hr | — |
| `POST /oauth/authorize` | Yes (browser) | IP 10/15m | Yes |
| `POST /oauth/register` | — | IP 5/hr | Yes |
| `POST /files/uploads` | — | actor 200/hr | — |
| `POST /files/uploads/multipart` | — | actor 50/hr | — |
| `POST /auth/keys` | — | actor 10/hr | — |
| `POST /files/share` | — | actor 50/hr | — |

## Implementation

### Files changed

| File | Change |
|---|---|
| `src/types.ts` | Add `TURNSTILE_SECRET_KEY`, `TURNSTILE_SITE_KEY` to Env |
| `src/lib/turnstile.ts` | New — server-side Turnstile validation |
| `src/middleware/rate-limit.ts` | New — D1-based rate limiter |
| `src/middleware/bot-guard.ts` | New — bot heuristic scoring |
| `src/routes/auth.ts` | Add rate limiting to register/challenge/verify |
| `src/routes/magic.ts` | Add Turnstile + rate limiting |
| `src/routes/oauth.ts` | Add Turnstile to login + rate limiting |
| `src/routes/keys.ts` | Add rate limiting |
| `src/routes/files-v2.ts` | Add rate limiting to uploads/shares |
| `src/pages/home.ts` | Add Turnstile widget to sign-in modal |
| `schema.sql` | Add rate_limits table |
| `migrations/0773_rate_limits.sql` | New migration |
| `vitest.config.ts` | Add TURNSTILE_SECRET_KEY test binding |
| `src/__tests__/bot-protection.test.ts` | New — tests for all three layers |

### Cloudflare Dashboard setup (manual)

1. Go to Cloudflare Dashboard → Turnstile → Add Site
2. Domain: `storage.liteio.dev`
3. Widget mode: Managed
4. Copy Site Key → set as `TURNSTILE_SITE_KEY` env var
5. Copy Secret Key → `wrangler secret put TURNSTILE_SECRET_KEY`

### Testing strategy

- **Turnstile:** Use Cloudflare's test keys (`1x0000...AA` always passes,
  `2x0000...AA` always fails) in vitest config
- **Rate limiting:** Insert rows directly into D1 and verify 429 responses
- **Bot guard:** Mock `request.cf` with datacenter ASN + missing headers
- **Integration:** Full flow tests — sign in with Turnstile, hit rate limits,
  trigger bot guard

## Rollout

1. Deploy with `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET_KEY` set
2. Turnstile validation is **soft-enforced** — if `TURNSTILE_SECRET_KEY` is not
   configured, the check is skipped (graceful degradation)
3. Rate limits are always active
4. Bot guard is always active for unauthenticated endpoints
5. Monitor audit logs and 429/403 rates for the first week
