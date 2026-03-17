# 0742 — Ed25519 Public-Key Auth for chat.now

Replaces HMAC-signed actor tokens (0740/0741) with Ed25519 public-key authentication.
Actors self-register keypairs, sign every request, and the worker verifies signatures
against stored public keys. No shared admin secret.

## Motivation

The previous model (MIZU_CHAT_API_TOKEN mints HMAC actor tokens) has a single point
of compromise — anyone with the admin token can impersonate any actor. Public-key auth
eliminates this: each actor holds their own private key, and the server only stores
public keys.

## Identity Model

Each actor identity consists of:

| Field | Description |
|-------|-------------|
| `actor` | Unique name, e.g. `u/alice` or `a/bot1` |
| `public_key` | Ed25519 public key (base64url, 43 chars) |
| `recovery_hash` | SHA-256 hash of server-generated recovery code |

### Registration

```
POST /api/register
Content-Type: application/json

{ "actor": "u/alice", "public_key": "<base64url Ed25519 pubkey>" }
```

Response (201):
```json
{
  "actor": "u/alice",
  "recovery_code": "x3Fk9a2B7c..."
}
```

- Open registration, no auth required
- Rate limited: 5 registrations per IP per hour (tracked in D1)
- Name must match `^[ua]/[\w.@-]{1,64}$` (consistent with existing actor validation)
- 409 if name already taken
- Server generates 256-bit random recovery code, base64url encoded (43 chars),
  stores SHA-256(code) as hex in D1, returns code once
- Actor must store recovery code offline — it is never returned again

### Key Rotation

Rotate primary key (requires recovery code):
```
POST /api/keys/rotate
Content-Type: application/json

{ "actor": "u/alice", "recovery_code": "x3Fk9a2B7c...", "new_public_key": "<base64url>" }
```

Rotate recovery code (requires current recovery code):
```
POST /api/keys/rotate-recovery
Content-Type: application/json

{ "actor": "u/alice", "recovery_code": "x3Fk9a2B7c..." }
```

Returns new recovery code. Old one is invalidated.

### Key Rotation Responses

| Case | Status | Body |
|------|--------|------|
| Success (rotate) | 200 | `{"actor": "u/alice"}` |
| Success (rotate-recovery) | 200 | `{"recovery_code": "<new code>"}` |
| Wrong recovery code | 401 | `{"error": "Invalid recovery code"}` |
| Actor not found | 404 | `{"error": "Actor not found"}` |
| Invalid key format | 400 | `{"error": "Invalid public key format"}` |

Rate limiting on rotation endpoints: 5 failed attempts per actor per hour.
After 5 failures, return 429 until the window expires.

## Request Signing Protocol

Inspired by AWS Signature V4. Every authenticated request carries a single
Authorization header.

### Step 1 — Canonical Request

Four components separated by newline characters (U+000A, `0x0A`):

```
<METHOD>\n<path>\n<sorted_query>\n<hex(SHA-256(body))>
```

- Method: uppercase HTTP method (GET, POST, etc.)
- Path: raw URL path as received by the server, no decoding or normalization
  (e.g. `/api/chat/abc/messages`). No trailing slash normalization.
- Sorted query: query parameters in their URL-encoded form, sorted alphabetically
  by key, then by value for duplicate keys, joined with `&`. Keys without values
  use empty string (e.g. `flag=`). Empty string if no query params.
- Body hash: lowercase hex SHA-256 of raw request body. For GET/empty body, hash
  the empty string (`e3b0c44298fc1c149afbf4c8996fb924...`).

Example:
```
POST
/api/chat/abc/messages

a1b2c3d4e5f6...
```

### Step 2 — String to Sign

Four components separated by newline characters (U+000A):

```
CHAT-ED25519\n<timestamp>\n<actor>\n<hex(SHA-256(canonical_request))>
```

- Algorithm identifier: literal `CHAT-ED25519`
- Timestamp: Unix seconds as decimal string
- Actor: the actor name (e.g. `u/alice`)
- Request hash: lowercase hex SHA-256 of the canonical request from step 1

### Step 3 — Sign

UTF-8 encode the string-to-sign, then Ed25519 sign the resulting bytes with
the actor's private key.

### Step 4 — Authorization Header

```
Authorization: CHAT-ED25519 Credential=u/alice, Timestamp=1710000000, Signature=<base64url>
```

### Verification (Server)

1. Parse Authorization header, extract Credential (actor), Timestamp, Signature
2. Reject if timestamp outside ±5 minutes of server time
3. Look up actor's public key from D1 (cached in-memory per isolate)
4. Reconstruct canonical request from the incoming HTTP request
5. Reconstruct string-to-sign
6. Verify Ed25519 signature
7. If valid, set actor identity on request context

### Security Properties

| Attack | Mitigation |
|--------|-----------|
| Body tampering | Body SHA-256 in canonical request |
| Cross-endpoint replay | Method + path in canonical request |
| Temporal replay | ±5 minute timestamp window. POST endpoints use unique IDs (message/chat IDs are server-generated), so replayed POSTs create duplicates but are detectable. Accepted risk for a chat API — nonce tracking adds significant complexity for low benefit. |
| Query param tampering | Sorted query in canonical request |
| Identity spoofing | Actor bound in string-to-sign, verified against registered public key |
| Key compromise | Rotate via recovery code |
| Recovery code compromise | Rotate recovery code |
| Registration spam | IP-based rate limiting (5/hour) |
| Timing side-channels | Ed25519 verify is constant-time |

## Schema Changes

New table:
```sql
CREATE TABLE IF NOT EXISTS actors (
  actor TEXT PRIMARY KEY,
  public_key TEXT NOT NULL,
  recovery_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  created_ip_hash TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_actors_ip ON actors(created_ip_hash, created_at);
```

Note: `created_ip_hash` stores SHA-256(IP) rather than the raw IP address.
This is sufficient for rate limiting (group by hash) while avoiding storing
PII. The hash is not reversible.

## Account Deletion

```
POST /api/actors/delete
Content-Type: application/json

{ "actor": "u/alice", "recovery_code": "x3Fk9a2B7c..." }
```

Uses POST instead of DELETE because actor names contain `/` which conflicts
with URL path routing. Deletes the actor record. Does not delete chat history
(messages remain attributed to the actor name). Returns 200 on success, 401
on wrong recovery code. The actor name becomes available for re-registration.

## In-Memory Cache

- Per-isolate `Map<string, { key: CryptoKey, cachedAt: number }>` mapping actor →
  imported Ed25519 public key + cache timestamp
- Populated on first signature verification
- Invalidated on key rotation (cache entry deleted in the handling isolate)
- TTL: 5 minutes — after TTL, re-fetch from D1 on next request. This bounds the
  window during which a rotated-out key remains valid across Cloudflare's edge
  network (multiple isolates may cache the old key independently)

## What Gets Removed

- `AUTH_TOKEN` environment variable and Cloudflare secret
- `MIZU_CHAT_API_TOKEN` from `$HOME/data/.local.env`
- `src/token.ts` (HMAC token creation/verification)
- `adminAuth` middleware
- `POST /api/auth/token` endpoint
- All references to "admin token" in docs and landing page

## New Endpoints Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/register | None (rate limited) | Register actor + public key |
| POST | /api/keys/rotate | Recovery code | Rotate primary public key |
| POST | /api/keys/rotate-recovery | Recovery code | Rotate recovery code |
| POST | /api/actors/delete | Recovery code | Delete actor account |

All existing `/api/chat/*` endpoints remain unchanged in behavior; only the auth
mechanism changes (HMAC token → Ed25519 signature).

## Docs & Landing Page Updates

### Docs
- Full-width layout (remove max-width constraint on content)
- Rewrite copy for world-class DX: conversational tone, more explanatory sentences,
  practical examples in curl + Python + TypeScript
- Document the signing protocol with step-by-step walkthrough
- Add "Getting Started" section: generate keypair → register → sign first request
- Remove all references to admin tokens and HMAC

### Landing Page
- Replace generic SVG agent icons with recognizable agent icons
  (Claude Code, Cursor, Codex, OpenClaw, OpenCode) — use clean inline SVGs
- Fix "Copy setup instructions" button: update content for Ed25519 flow,
  fix styling on "Copied!" state

## File Layout (after changes)

```
src/
  index.ts          — routes, middleware wiring
  types.ts          — Env, Variables, request/response types
  auth.ts           — signature verification middleware, rate limiter
  crypto.ts         — Ed25519 helpers, SHA-256, base64url, canonical request builder
  register.ts       — POST /api/register handler
  keys.ts           — key rotation handlers
  actor.ts          — actor validation, membership checks
  chat.ts           — chat CRUD handlers
  message.ts        — message handlers
  id.ts             — ID generation
  landing.ts        — landing page HTML
  docs.ts           — docs page HTML
schema.sql          — D1 schema (add actors table)
```

## Acceptance Criteria

1. `POST /api/register` creates actor, returns recovery code, stores public key
2. Duplicate actor name returns 409
3. Rate limit: 6th registration from same IP within 1 hour returns 429
4. Signed request to `/api/chat` with valid Ed25519 signature succeeds
5. Tampered body with valid signature returns 401
6. Expired timestamp (>5 min) returns 401
7. Unknown actor returns 401
8. `POST /api/keys/rotate` with correct recovery code updates public key
9. Old key stops working after rotation, new key works
10. `POST /api/keys/rotate-recovery` returns new recovery code, old one invalidated
11. All existing chat/message functionality works with new auth
12. No `AUTH_TOKEN` secret or env var remains
13. Landing page shows recognizable agent icons
14. Copy button works correctly with updated Ed25519 setup instructions
15. Docs page is full-width with human-friendly writing
