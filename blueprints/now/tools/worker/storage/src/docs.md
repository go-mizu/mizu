# API Reference {#overview}

Complete reference for the storage.now API. 18 operations. 8 resources. Unix philosophy.

Base URL: `https://storage.liteio.dev`

All requests use JSON for request and response bodies (except file uploads/downloads which use raw bytes). All responses include appropriate `Content-Type` headers.

## Quick Start {#quickstart}

Get up and running in three commands:

```
# 1. Register an agent identity
POST /auth/register
{"actor": "a/my-app", "public_key": "<base64url-ed25519-pubkey>"}

# 2. Authenticate (challenge-response)
POST /auth/token
{"method": "ed25519", "step": "challenge", "actor": "a/my-app"}
# → sign the nonce, then:
POST /auth/token
{"method": "ed25519", "step": "verify", "challenge_id": "ch_...", "actor": "a/my-app", "signature": "<base64url>"}
# → {"access_token": "..."}

# 3. Upload a file
PUT /files/hello.txt
Authorization: Bearer <access_token>
Content-Type: text/plain

Hello, world!
```

## Authentication {#authentication}

All routes except `/auth/*` and `/p/*` require a Bearer token:

```
Authorization: Bearer <token>
```

Three ways to get a token:

| Method | For | How |
|---|---|---|
| Ed25519 | Machines / scripts | Register public key, sign challenges |
| Magic link | Humans | Enter email, click link in inbox |
| API key | Services | Generate via `POST /keys`, use `sk_...` as Bearer token |

## Rate Limits {#rate-limits}

| Endpoint | Limit | Window |
|---|---|---|
| POST /auth/register | 10 | 1 minute |
| POST /auth/token | 20 | 1 minute |
| POST /auth/magic-link | 5 | 1 minute |
| PUT /files/* | 100 | 1 minute |
| POST /shares | 50 | 1 minute |
| POST /links | 50 | 1 minute |
| GET /p/* | 200 | 1 minute |

Rate-limited responses return `429 Too Many Requests` with `Retry-After` header.

| Resource | Ops | Unix analog |
|---|---|---|
| /auth | 4 | login/logout |
| /files | 4 | read/write/rm/stat |
| /tree | 1 | ls |
| /shares | 3 | chmod/ln |
| /p | 1 | readlink |
| /presign | 1 | pipe |
| /keys | 3 | ssh-keygen |
| /log | 1 | /var/log |

# Auth — 4 ops {#auth}

Identity and session management. Two methods: Ed25519 challenge-response (machines) and magic link (humans).

## POST /auth/register {#register}

Create an identity. Like `useradd`.

```
POST /auth/register
Content-Type: application/json

{"actor": "a/my-agent", "public_key": "<base64url-ed25519>"}

→ 201 Created
{"actor": "a/my-agent", "created": true}
```

Actors: `a/name` (agent, Ed25519) or `u/email` (human, magic link).

## POST /auth/token {#auth-token}

Authenticate. Multiplexed by `method` field.

**Ed25519 — step 1: challenge**

```
POST /auth/token
Content-Type: application/json

{"method": "ed25519", "step": "challenge", "actor": "a/my-agent"}

→ 200 OK
{"challenge_id": "ch_...", "nonce": "abc123...", "expires_at": "..."}
```

**Ed25519 — step 2: verify**

```
POST /auth/token
Content-Type: application/json

{"method": "ed25519", "step": "verify", "challenge_id": "ch_...", "actor": "a/my-agent", "signature": "<base64url>"}

→ 200 OK
{"access_token": "...", "expires_at": "..."}
```

Sign the nonce with your Ed25519 private key. The signature must be base64url-encoded.

**Magic link**

```
POST /auth/token
Content-Type: application/json

{"method": "magic", "email": "user@example.com"}

→ 202 Accepted
{"sent": true}
```

Use the token: `Authorization: Bearer <access_token>`

## GET /auth/callback/:token {#auth-callback}

Verify a magic link. Sets session cookie and redirects.

```
GET /auth/callback/mtk_abc123

→ 302 Found
Set-Cookie: session=...; Path=/; HttpOnly; Secure
Location: /browse
```

## DELETE /auth/token {#auth-logout}

End the current session. Like `logout`.

```
DELETE /auth/token
Authorization: Bearer <token>

→ 200 OK
{"ok": true}
```

# Files — 4 ops {#files}

Core file I/O. Like `read(2)`, `write(2)`, `unlink(2)`, `stat(2)`.

Paths are virtual. Parent directories are created implicitly on write. Content-Type is auto-detected from extension.

## PUT /files/{path} {#file-write}

Write a file. Like `write(fd, buf)`.

```
PUT /files/docs/readme.md
Authorization: Bearer <token>
Content-Type: text/markdown

<file bytes>

→ 201 Created
{"id": "o_...", "path": "docs/readme.md", "size": 1234}
```

Max 100 MB per request. Use `/presign` for larger files.

## GET /files/{path} {#file-read}

Read a file. Like `read(fd, buf)`. Returns raw bytes with appropriate Content-Type.

```
GET /files/docs/readme.md
Authorization: Bearer <token>

→ 200 OK
Content-Type: text/markdown
Content-Length: 1234

<file bytes>
```

Supports `Range` headers for partial reads.

## DELETE /files/{path} {#file-delete}

Remove a file. Like `unlink(path)`.

```
DELETE /files/docs/readme.md
Authorization: Bearer <token>

→ 200 OK
{"deleted": true, "path": "docs/readme.md"}
```

## HEAD /files/{path} {#file-stat}

Stat a file. Returns metadata headers, no body. Like `stat(path)`.

```
HEAD /files/docs/readme.md
Authorization: Bearer <token>

→ 200 OK
Content-Type: text/markdown
Content-Length: 1234
ETag: "abc123"
Last-Modified: Wed, 18 Mar 2026 12:00:00 GMT
```

# Tree — 1 op {#tree}

Directory listing. Like `readdir(3)` or `ls`.

## GET /tree/{path?} {#tree-list}

List directory contents. Omit path for root.

```
GET /tree/docs
Authorization: Bearer <token>

→ 200 OK
{
  "path": "docs/",
  "items": [
    {"name": "readme.md", "size": 1234, "modified": "2026-03-18T12:00:00Z"},
    {"name": "images", "is_folder": true}
  ]
}
```

Root: `GET /tree` → `{"path": "/", "items": [...]}`

Each item includes: `name`, `size` (files only), `modified` (ISO 8601), `is_folder` (directories only), `content_type` (files only).

# Shares — 3 ops {#shares}

Access control. Like `chmod`, `chown`, `ln -s`.

A share grants another actor access to a path. Set `grantee` to `"public"` for anonymous access.

Permission levels: `viewer` (read), `editor` (read/write), `uploader` (write-only).

## POST /shares {#share-create}

Grant access.

```
POST /shares
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/readme.md", "grantee": "a/bot", "permission": "viewer"}

→ 201 Created
{"id": "sh_...", "token": "tok_abc", "permission": "viewer"}
```

Public share:

```
{"path": "docs/readme.md", "grantee": "public", "permission": "viewer"}

→ 201 Created
{"id": "sh_...", "token": "tok_xyz", "url": "/p/tok_xyz"}
```

## GET /shares {#share-list}

List shares you've granted.

```
GET /shares
Authorization: Bearer <token>

→ 200 OK
{"shares": [{"id": "sh_...", "path": "docs/readme.md", "grantee": "a/bot", "permission": "viewer"}]}
```

## DELETE /shares/{id} {#share-revoke}

Revoke a share.

```
DELETE /shares/sh_abc123
Authorization: Bearer <token>

→ 200 OK
{"deleted": true}
```

# Public Access — 1 op {#public}

Anonymous file access via share token. Like following a symlink.

## GET /p/{token} {#public-access}

Access a shared resource. No auth required for public shares.

```
GET /p/tok_xyz

→ 200 OK
Content-Type: text/markdown

<file bytes>
```

Private shares require the grantee's Bearer token.

# Presign — 1 op {#presign}

Direct-to-storage transfer. Like Unix pipes — data flows directly between client and object store, bypassing the API server.

## POST /presign {#presign-url}

Get a presigned URL for direct upload or download.

```
POST /presign
Authorization: Bearer <token>
Content-Type: application/json

{"path": "models/v3.bin", "method": "PUT", "content_type": "application/octet-stream"}

→ 200 OK
{"url": "https://...signed...", "method": "PUT", "expires_in": 3600}
```

For download: `{"path": "models/v3.bin", "method": "GET"}`

Upload the file directly to the presigned URL — the data never passes through the API server. URLs expire after 1 hour.

# Keys — 3 ops {#keys}

API key management. Like `ssh-keygen` and `~/.ssh/authorized_keys`.

API keys are long-lived Bearer tokens for programmatic access. Use them in CI/CD, scripts, and integrations.

## POST /keys {#key-create}

Create an API key. Token shown once.

```
POST /keys
Authorization: Bearer <token>
Content-Type: application/json

{"name": "ci-deploy"}

→ 201 Created
{"id": "key_...", "name": "ci-deploy", "token": "sk_..."}
```

Use as: `Authorization: Bearer sk_...`

## GET /keys {#key-list}

List API keys. Tokens not included.

```
GET /keys
Authorization: Bearer <token>

→ 200 OK
{"keys": [{"id": "key_...", "name": "ci-deploy", "created": "2026-03-18T12:00:00Z"}]}
```

## DELETE /keys/{id} {#key-revoke}

Revoke an API key. Takes effect immediately.

```
DELETE /keys/key_abc123
Authorization: Bearer <token>

→ 200 OK
{"deleted": true}
```

# Log — 1 op {#log}

Audit trail. Like `tail /var/log/auth.log`.

Every file write, share creation, key rotation, and authentication event is logged.

## GET /log {#log-read}

Read audit events. Supports `?limit=N` and `?cursor=` for pagination.

```
GET /log?limit=10
Authorization: Bearer <token>

→ 200 OK
{
  "events": [
    {"ts": "2026-03-18T12:00:00Z", "action": "file.write", "path": "docs/readme.md"},
    {"ts": "2026-03-18T11:59:00Z", "action": "share.create", "path": "docs/readme.md"}
  ],
  "cursor": "cur_..."
}
```

Event actions: `file.write`, `file.read`, `file.delete`, `share.create`, `share.revoke`, `key.create`, `key.revoke`, `auth.login`, `auth.logout`.

# All 18 Operations {#endpoints}

| # | Method | Path | Resource | Unix analog |
|---|---|---|---|---|
| 1 | POST | /auth/register | Auth | useradd |
| 2 | POST | /auth/token | Auth | login |
| 3 | GET | /auth/callback/:token | Auth | login (verify) |
| 4 | DELETE | /auth/token | Auth | logout |
| 5 | PUT | /files/{path} | Files | write(fd, buf) |
| 6 | GET | /files/{path} | Files | read(fd, buf) |
| 7 | DELETE | /files/{path} | Files | unlink(path) |
| 8 | HEAD | /files/{path} | Files | stat(path) |
| 9 | GET | /tree/{path?} | Tree | readdir(path) |
| 10 | POST | /shares | Shares | chmod |
| 11 | GET | /shares | Shares | getfacl |
| 12 | DELETE | /shares/{id} | Shares | setfacl -x |
| 13 | GET | /p/{token} | Public | readlink |
| 14 | POST | /presign | Presign | pipe |
| 15 | POST | /keys | Keys | ssh-keygen |
| 16 | GET | /keys | Keys | ls ~/.ssh |
| 17 | DELETE | /keys/{id} | Keys | rm key |
| 18 | GET | /log | Log | tail /var/log |

# Error Codes {#errors}

All errors return JSON: `{"error": {"code": "...", "message": "..."}}`

| Code | HTTP | Meaning |
|---|---|---|
| invalid_request | 400 | Malformed request body or missing required field |
| unauthorized | 401 | Missing, expired, or invalid Bearer token |
| forbidden | 403 | Valid token but insufficient permission for this resource |
| not_found | 404 | File, share, or resource does not exist |
| conflict | 409 | Resource already exists (e.g., duplicate actor registration) |
| too_large | 413 | Request body exceeds the 100 MB limit |
| rate_limited | 429 | Too many requests — check `Retry-After` header |
| internal | 500 | Unexpected server error |

# Extensions {#extensions}

Extensions are not part of the core protocol. Implementations MAY support them.

| Extension | Endpoints | Purpose |
|---|---|---|
| Drive | 13 | Convenience coreutils (star, trash, search, rename, move, copy) |
| OAuth 2.0 | 7 | Third-party authorization (RFC 6749 + PKCE) |
| MCP | 2 | Model Context Protocol for AI agents |
| Spaces | 11 | Collaborative workspaces |

The reference implementation at `storage.liteio.dev` supports all extensions.
