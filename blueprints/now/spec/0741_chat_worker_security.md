# 0741 chat worker security

## Objective

Deep review of the chat worker identity model, permission enforcement, and attack surfaces. Documents the trust boundary, catalogs existing controls, identifies gaps, and specifies concrete fixes.

## 1. Threat Model

### Actors

| Actor type | Identifier | Trust level |
|---|---|---|
| Human user | `u/<name>` | Authenticated via shared API token. Identity asserted by `X-Actor` header. |
| Agent | `a/<name>` | Same as human. Typically a service or bot acting on behalf of users. |
| API token holder | Bearer token | Full access to all API operations. The token is the sole authentication credential. |
| Unauthenticated caller | None | Can only access `GET /` and `GET /docs`. All `/api/*` routes return 401. |

### Attack surfaces

| Surface | Risk | Mitigation |
|---|---|---|
| Token brute-force | Attacker guesses API token | Timing-safe comparison prevents timing side-channels. Token entropy is the primary defense. |
| Actor spoofing | Token holder sets arbitrary `X-Actor` | By design (see section 2). Token holder is trusted to assert actor identity. |
| Oversized request body | Denial-of-service via memory exhaustion | **Gap** — no body size limit currently enforced. Fixed in section 7. |
| SQL injection | Malicious input in query/body | All queries use parameterized bindings via D1 prepared statements. |
| Cross-origin abuse | Browser-based CSRF or data exfiltration | CORS is wide open (`*`). Acceptable because all mutations require a Bearer token in the `Authorization` header, which browsers do not attach automatically. |
| Private chat enumeration | Attacker probes chat IDs to discover private chats | Private chats return 404 (not 403) to prevent existence leakage. Chat IDs use high-entropy random generation. |
| Cursor manipulation | Attacker supplies a `before` cursor from a different chat | Cursor lookup is scoped to `chat_id`, preventing cross-chat data access. |
| Direct chat member overflow | "direct" chat accepts unlimited members | **Gap** — direct chats should enforce a 2-member limit. Fixed in section 7. |

## 2. Identity Model Review

### Current design

Authentication is two-layered:

1. **Bearer token** (`Authorization: Bearer <token>`) — proves the caller is authorized to use the API. Validated with timing-safe comparison in `auth.ts`.
2. **Actor header** (`X-Actor: u/alice` or `a/my-bot`) — declares which actor is performing the operation. Validated for format only (regex `^[ua]/[\w.@-]{1,64}$`).

The API token is **not** scoped to a specific actor. Any valid token holder can set `X-Actor` to any valid actor string.

### Trust boundary

The current design assumes a **single operator** (or a set of cooperating operators) sharing the same API token. The token holder is trusted to:

- Correctly assert actor identity via `X-Actor`
- Not impersonate other actors maliciously
- Manage their own actors responsibly

This is the service-account model: the API token authenticates the **system**, and the system asserts actor identity on behalf of its users. This is analogous to a backend server that authenticates to a database with a single credential and manages user sessions itself.

### When this model is appropriate

- A single application (e.g., a CLI tool, a web app backend) holds the token and sets `X-Actor` based on its own authenticated user session.
- Multiple cooperating services share the token and coordinate actor identity.
- Internal tooling where all token holders are trusted operators.

### When this model is NOT appropriate

- Multi-tenant scenarios where untrusted users each receive API credentials directly. In this case, a user holding the token could impersonate any other user by setting `X-Actor` to their identifier.

## 3. Actor Impersonation

### Current behavior (by design)

Any token holder can set `X-Actor` to any valid actor string. This is intentional:

- The token authenticates the **caller** (a system/service), not the **actor** (a user/agent within that system).
- The caller is trusted to assert the correct actor identity.
- This enables service-account patterns where a backend acts on behalf of multiple users.

### When impersonation is acceptable

- The token is held by a single trusted backend that sets `X-Actor` from its own session data.
- All token holders cooperate and respect actor boundaries.

### When impersonation is a risk

- If the token is distributed to end users directly, any user can impersonate any other user.
- If the token leaks, the attacker can act as any actor.

### Future enhancement: actor-scoped tokens

For multi-tenant deployments where the API token is distributed to end users, an optional actor-scoped token scheme could be introduced:

```
Authorization: Bearer <token>:<actor-scope>
```

Or via a signed JWT with an `actor` claim:

```
Authorization: Bearer eyJ...  (JWT with {"sub": "u/alice", ...})
```

The worker would verify the JWT signature and enforce that `X-Actor` matches the token's `sub` claim. This is **not implemented** and is documented here as a future enhancement path.

## 4. Permission Matrix

| Operation | Endpoint | Required auth | Actor required | Membership required | Visibility rule |
|---|---|---|---|---|---|
| Create chat | `POST /api/chat` | Bearer token | Yes | N/A (creator auto-joins) | N/A |
| List chats | `GET /api/chat` | Bearer token | Optional | No | Public chats always shown. Private chats shown only if actor is a member. Without `X-Actor`, only public chats returned. |
| Get chat | `GET /api/chat/:id` | Bearer token | Conditional | For private only | Public: any authenticated request. Private: members only (404 if not member). |
| Join chat | `POST /api/chat/:id/join` | Bearer token | Yes | No (joining creates membership) | Public: allowed. Private: blocked (403). |
| Send message | `POST /api/chat/:id/messages` | Bearer token | Yes | Yes | Members only (403 if not member). |
| List messages | `GET /api/chat/:id/messages` | Bearer token | Conditional | For private only | Public: any authenticated request can read. Private: members only (404 if not member). |

### Notes on public chat message readability

Any authenticated request (valid Bearer token) can read messages from a public chat without joining it. This is by design: public chats are intended to be openly readable. The `X-Actor` header is not required to read public chat messages. Joining is only required to **send** messages.

## 5. Current Security Controls

### 5.1 Timing-safe token comparison

`auth.ts` hashes both the provided token and the stored `AUTH_TOKEN` with SHA-256 before comparison. This prevents timing side-channel attacks and also eliminates length leakage since both digests are fixed-length (32 bytes).

### 5.2 Actor format validation

`actor.ts` enforces the regex `^[ua]/[\w.@-]{1,64}$` with a hard length cap of 67 characters. This prevents:

- Injection via actor strings (no special SQL/HTML characters beyond `@`, `.`, `-`)
- Unbounded storage from excessively long actor identifiers

### 5.3 Input length limits

| Field | Limit | Enforcement |
|---|---|---|
| Chat title | 200 characters | Truncated via `.slice(0, 200)` in `createChat` |
| Message text | 4,000 characters | Truncated via `.slice(0, 4000)` in `sendMessage` |
| Actor identifier | 64 characters (+ 2-char prefix + `/`) | Rejected (400) if exceeds limit |

### 5.4 Membership enforcement

- `sendMessage` checks `isMember()` before allowing message creation (403 if not a member).
- Private chat reads (`getChat`, `listMessages`) check `isMember()` and return 404 if the actor is not a member (preventing existence leakage).

### 5.5 Visibility enforcement

- `joinChat` blocks joining private chats with 403.
- `listChats` filters private chats from results unless the actor is a member.
- `getChat` returns 404 for private chats when the requester is not a member.

### 5.6 JSON parse error handling

Both `createChat` and `sendMessage` wrap `c.req.json()` in try/catch and return 400 with a descriptive error message, preventing unhandled exceptions from leaking stack traces.

### 5.7 Cursor scoping

The `before` cursor in `listMessages` is resolved via a query scoped to `chat_id`:

```sql
SELECT created_at FROM messages WHERE id = ? AND chat_id = ?
```

This prevents an attacker from using a message ID from chat A as a cursor in chat B to infer timing information.

### 5.8 Parameterized queries (SQL injection prevention)

All D1 queries use `.prepare(...).bind(...)` with positional parameters. No string interpolation or concatenation is used in SQL construction. This eliminates SQL injection as an attack vector.

## 6. Gaps and Enhancements

### 6a. Direct chat member limit (gap — fixed)

**Issue:** A chat with `kind: "direct"` is semantically a 1-to-1 conversation, but the `joinChat` handler does not enforce any member limit. Additional actors can join, turning a "direct" chat into a group conversation.

**Fix:** In `joinChat`, before inserting a new member into a `direct` chat, count existing members. If there are already 2, reject the join with 403.

### 6b. Public chat message read access (by design)

**Issue:** Any authenticated request can read messages from a public chat without being a member.

**Decision:** This is acceptable and intentional. Public chats are designed for open readability. The membership requirement is only for sending messages. No change needed.

### 6c. Actor consistency across operations (future consideration)

**Issue:** No protection against actor A creating a chat, then actor B (same token) performing destructive operations on it. Since the API currently has no delete or update endpoints, this is moot.

**Recommendation:** When delete/update endpoints are added, consider adding creator-only or role-based access controls. For now, the single-operator trust model (section 2) makes this a non-issue.

### 6d. Request body size limit (gap — fixed)

**Issue:** No limit on request body size. A malicious caller could send a multi-gigabyte POST body, causing memory exhaustion on the worker.

**Fix:** Add middleware that rejects request bodies larger than 64 KB for all `/api/*` routes. This is generous for JSON chat payloads (the largest field is message text at 4,000 characters) while preventing abuse.

### 6e. SQL injection (verified — no issue)

All queries use D1 parameterized bindings (`prepare` + `bind`). No dynamic SQL construction exists in the codebase. This is secure.

### 6f. CORS policy (acceptable)

The worker uses `cors()` with default settings (allows all origins). For an API-token-protected service, this is acceptable because:

- Browsers cannot attach the `Authorization: Bearer <token>` header automatically (it is not a "simple" header).
- CORS preflight (`OPTIONS`) will occur for all API requests from browsers, but the actual request still requires the token.
- The landing and docs pages (`GET /`, `GET /docs`) are public and benefit from open CORS.

If the API were cookie-authenticated, open CORS would be a critical vulnerability. With Bearer token auth, it is safe.

## 7. Implementation Changes

### 7a. Direct chat 2-member limit in `joinChat`

**File:** `src/chat.ts`

In `joinChat`, after verifying the chat exists and is public, add a check for `direct` kind chats:

```typescript
if (chat.kind === "direct") {
  const { count } = await c.env.DB.prepare(
    "SELECT COUNT(*) as count FROM members WHERE chat_id = ?"
  ).bind(id).first<{ count: number }>() ?? { count: 0 };
  if (count >= 2) {
    return c.json({ error: "Direct chat is full (max 2 members)" }, 403);
  }
}
```

### 7b. Request body size limit middleware

**File:** `src/index.ts`

Add a body size limit middleware before the auth middleware for `/api/*` routes:

```typescript
const MAX_BODY_SIZE = 65_536; // 64 KB

app.use("/api/*", async (c, next) => {
  const contentLength = c.req.header("Content-Length");
  if (contentLength && parseInt(contentLength, 10) > MAX_BODY_SIZE) {
    return c.json({ error: "Request body too large" }, 413);
  }
  await next();
});
```

This checks `Content-Length` before reading the body. For requests without `Content-Length` (chunked transfer), the body is still bounded by Cloudflare Workers' own limits (typically 100 MB for free plans), but the application-level field truncation (title: 200 chars, text: 4,000 chars) provides secondary protection.

## Acceptance Criteria

1. `POST /api/chat/:id/join` on a `direct` chat with 2 existing members returns 403.
2. `POST /api/*` with `Content-Length` exceeding 64 KB returns 413.
3. All existing security controls documented in this spec are verified present in the codebase.
4. The trust boundary (single-operator model) is explicitly documented.
5. The permission matrix matches the implemented behavior.
