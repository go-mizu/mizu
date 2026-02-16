# 0548: D1 Storage Migration & Enhanced Account Management

## Overview

Migrate all CF Worker storage from KV to D1 via `StorageBackend` abstraction layer. Enhance account system with encrypted credential storage, re-login capability, and smart auto-rotation. D1 is 100x more generous on writes than KV free tier.

## D1 Free Tier Limits

| Resource | Free Tier | vs KV Free |
|----------|-----------|------------|
| Databases | 10 per account | N/A |
| Storage per DB | 500 MB | KV: 1 GB |
| Rows read/day | 5,000,000 | KV: 100K reads |
| Rows written/day | **100,000** | **KV: 1,000** (100x less) |
| Max query/invocation | 50 | N/A |
| Connections | 6 simultaneous | N/A |
| Reset | 00:00 UTC daily | Same |

**Key insight**: KV's 1,000 writes/day limit is the bottleneck. D1's 100,000 writes/day eliminates this entirely for our use case (~50 writes per registration, ~5 writes per search).

## Storage Architecture

### StorageBackend Interface

```typescript
interface StorageBackend {
  readonly name: string
  get<T>(key: string): Promise<T | null>
  set<T>(key: string, value: T, ttlSeconds?: number): Promise<void>
  delete(key: string): Promise<void>
  list(prefix: string): Promise<string[]>
}
```

Three implementations:
- **MemoryStorage**: Global Map, instant, lost on isolate eviction
- **KVStorage**: CF KV, persistent, 1K writes/day limit
- **D1Storage**: CF D1, persistent, 100K writes/day, **default**

### Data Stored in D1 (via kv table)

| Key Pattern | Data | TTL | Source |
|-------------|------|-----|--------|
| `account:{id}` | Account JSON (session, credentials, quota) | 30 days | AccountManager |
| `accounts:index` | AccountSummary[] | permanent | AccountManager |
| `accounts:robin` | number (round-robin pointer) | permanent | AccountManager |
| `accounts:log` | RegistrationLog[] (last 50) | permanent | AccountManager |
| `accounts:lock` | ISO timestamp | 60 seconds | AccountManager |
| `thread:{id}` | Thread (messages, citations) | 30 days | ThreadManager |
| `threads:recent` | ThreadIndex (summaries) | permanent | ThreadManager |
| `sessions:pool` | SessionState[] | 1 hour | Session pool |
| `session:anon` | SessionState | 1 hour | Legacy compat |
| `og:{url}` | OG metadata | 24 hours | API routes |

## Enhanced Account System

### Account Lifecycle

```
register → active → exhausted → disabled
                 ↘ failed ↗
                     ↓
              re-login attempt
                     ↓
           success → active (restored)
           failure → disabled
```

States:
- **active**: Has remaining pro queries, session valid
- **exhausted**: proQueries == 0, kept for records
- **failed**: Session error (403, auth failure), eligible for re-login
- **disabled**: Permanently unusable (banned, re-login failed)

### Encrypted Credential Storage

Temp email passwords stored encrypted (AES-256-GCM) for re-login:

```typescript
interface Account {
  id: string
  email: string
  emailProvider: string           // 'mail.tm', 'mail.gw', etc.
  emailPasswordEnc: string        // AES-256-GCM encrypted (base64)
  session: SessionState
  proQueries: number
  status: 'active' | 'exhausted' | 'failed' | 'disabled'
  createdAt: string
  lastUsedAt: string
  disabledAt?: string
  disableReason?: string
  totalQueriesUsed: number        // lifetime counter
}
```

Encryption uses Web Crypto API (available in CF Workers):
- Algorithm: AES-256-GCM with random 12-byte IV
- Key derived from `ACCOUNT_SECRET` env var via PBKDF2
- Format: `{iv_hex}:{ciphertext_base64}`
- Decryptable for re-login, not one-way hash

### Re-login Flow

When a pro search fails with 401/403:
1. Mark current session as stale
2. Look up stored encrypted password
3. Decrypt password using ACCOUNT_SECRET
4. Re-authenticate with temp email provider (if provider still accepts the credentials)
5. Update session cookies + CSRF token
6. Retry the search
7. If re-login fails → mark account as `disabled`

### Smart Auto-Rotation

```
nextAccount() → round-robin active accounts
  ↓ no active accounts
needsRegistration() → true
  ↓
backgroundRegister() via waitUntil()
  ↓ meanwhile
fallback to anonymous session (auto mode)
```

After each pro query:
- Decrement `proQueries`, increment `totalQueriesUsed`
- If `proQueries == 0` → status = `exhausted`
- If active count < MIN_ACTIVE → trigger background registration

## Migration Plan

### Files Modified

| File | Change |
|------|--------|
| `types.ts` | Add `emailProvider`, `emailPasswordEnc`, `totalQueriesUsed`, `disabledAt`, `disableReason` to Account |
| `storage.ts` | Already done — 3 backends |
| `accounts.ts` | Already uses StorageBackend; add re-login, encrypted password support |
| `email.ts` | Return password from `TempEmailClient` + `createTempEmail()` |
| `register.ts` | Store encrypted password during registration |
| `threads.ts` | Accept StorageBackend instead of KVNamespace |
| `perplexity.ts` | Accept StorageBackend for session pool |
| `routes/api.ts` | Pass StorageBackend to ThreadManager, session pool |
| `routes/search.ts` | Pass StorageBackend to ThreadManager |
| `routes/thread.ts` | Pass StorageBackend to ThreadManager |
| `routes/history.ts` | Pass StorageBackend to ThreadManager |
| `index.ts` | Pass StorageBackend to ThreadManager |
| `crypto.ts` | NEW: AES-256-GCM encrypt/decrypt using Web Crypto API |
| `config.ts` | Add ACCOUNT_SECRET to env |

### New Files

- `src/crypto.ts` — `encrypt(plaintext, secret)` / `decrypt(ciphertext, secret)` using AES-256-GCM

### D1 Schema

Single `kv` table (already created):
```sql
CREATE TABLE kv (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  expires_at INTEGER  -- unix timestamp ms, NULL = no expiry
);
```

All data stored as JSON values with key-based access — same pattern as KV but with SQL persistence and 100x write quota.

## Security

- Passwords encrypted with AES-256-GCM (not plaintext, not hashed)
- `ACCOUNT_SECRET` stored as CF Worker secret (not in code)
- No session tokens in API responses (masked in debug endpoints)
- Debug endpoints are temporary (removed after testing)
- D1 data encrypted at rest by Cloudflare
