# storage-worker

Self-hosted file storage API on Cloudflare Workers. S3-compatible semantics with buckets, objects, signed URLs, resumable uploads, OAuth 2.0, and MCP — all on the edge.

**Stack:** Hono + Cloudflare D1 (SQLite) + R2 (object storage)

**Live:** [storage.liteio.dev](https://storage.liteio.dev)

## Quick start

```bash
# Install dependencies
npm install

# Create the D1 database
npm run db:create

# Run schema migrations (local)
npm run db:migrate

# Start dev server
npm run dev

# Deploy to Cloudflare
npm run deploy
```

## Architecture

```
src/
├── index.ts              # Route definitions, CORS, middleware wiring
├── types.ts              # TypeScript interfaces (Env, Variables, row types)
├── docs.md               # API documentation (built into /api page)
├── middleware/
│   ├── auth.ts           # Bearer token resolution (session + API key)
│   ├── authorize.ts      # Scope enforcement, path prefix checks, RBAC
│   └── rate-limit.ts     # Sliding window rate limiter (D1-backed)
├── routes/
│   ├── auth.ts           # Registration, challenge/verify, magic links
│   ├── oauth.ts          # OAuth 2.0 + PKCE (RFC 7636, RFC 7591)
│   ├── buckets.ts        # Bucket CRUD
│   ├── objects.ts        # Object upload/download/delete/copy/move
│   ├── signed.ts         # Signed URL creation and access
│   ├── tus.ts            # TUS resumable uploads (v1.0.0)
│   ├── files.ts          # Legacy drive file operations
│   ├── folders.ts        # Legacy drive folder operations
│   ├── drive.ts          # Drive features (search, star, trash, stats)
│   ├── shares.ts         # Share management (RBAC: owner/editor/viewer)
│   ├── links.ts          # Public link access
│   ├── api-keys.ts       # Scoped API key management
│   └── mcp.ts            # MCP server (JSON-RPC 2.0)
├── lib/
│   ├── audit.ts          # Audit logging
│   ├── crypto.ts         # Crypto utilities
│   ├── error.ts          # Structured error responses
│   ├── id.ts             # ID generation (prefixed nanoid)
│   ├── mime.ts           # MIME type detection
│   ├── path.ts           # Path validation and sanitization
│   └── tus.ts            # TUS protocol constants and helpers
└── pages/
    ├── home.ts           # Landing page
    ├── developers.ts     # Developer docs page
    ├── docs.ts           # API reference page
    ├── pricing.ts        # Pricing page
    ├── browse.ts         # File browser SPA
    ├── cli.ts            # CLI install page
    ├── ai.ts             # AI/MCP page
    └── session.ts        # Session resolution for pages
```

## Infrastructure

| Resource | Binding | Purpose |
|----------|---------|---------|
| D1 | `DB` | Metadata, auth, sessions, rate limits |
| R2 | `BUCKET` | Object storage (files) |

### Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `RESEND_API_KEY` | No | Resend API key for magic link emails |
| `R2_ACCESS_KEY_ID` | No | R2 S3-compatible access key (for presigned URLs) |
| `R2_SECRET_ACCESS_KEY` | No | R2 S3-compatible secret key |
| `CF_ACCOUNT_ID` | No | Cloudflare account ID (for presigned URLs) |
| `R2_BUCKET_NAME` | No | R2 bucket name (for presigned URLs) |

## API

All protected endpoints require a `Bearer` token via the `Authorization` header or a `session` cookie.

Errors follow a consistent format:

```json
{ "error": { "code": "not_found", "message": "Bucket not found" } }
```

Error codes: `invalid_request` (400), `unauthorized` (401), `forbidden` (403), `not_found` (404), `conflict` (409), `too_large` (413), `rate_limited` (429), `internal` (500).

---

### Authentication

#### Register

```
POST /auth/register
{ "actor": "u/alice", "email": "alice@example.com" }
```

#### Magic link

```
POST /auth/magic-link
{ "email": "alice@example.com" }
→ Sends email with login link
```

```
GET /auth/magic/:token
→ Sets session cookie, redirects to /browse
```

#### Challenge/verify (public key auth)

```
POST /auth/challenge
{ "actor": "u/alice" }
→ { "challenge_id": "...", "nonce": "..." }
```

```
POST /auth/verify
{ "challenge_id": "...", "actor": "u/alice", "signature": "..." }
→ { "token": "...", "expires_at": ... }
```

#### Logout

```
POST /auth/logout
→ Clears session
```

---

### OAuth 2.0 (PKCE)

Full OAuth 2.0 authorization code flow with PKCE (S256). Supports dynamic client registration (RFC 7591).

#### Discovery

```
GET /.well-known/oauth-authorization-server
GET /.well-known/oauth-protected-resource
```

#### Client registration

```
POST /oauth/register
{
  "client_id": "my-app",
  "redirect_uris": ["http://localhost:3000/callback"],
  "client_name": "My App"
}
```

#### Authorization

```
GET /oauth/authorize?response_type=code&client_id=my-app&redirect_uri=...&code_challenge=...&code_challenge_method=S256&scope=storage:read+storage:write&state=...
→ Login page → Consent page → Redirect with ?code=...&state=...
```

#### Token exchange

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=...&redirect_uri=...&client_id=my-app&code_verifier=...
→ { "access_token": "sk_...", "token_type": "bearer", "expires_in": 7776000, "scope": "..." }
```

#### Scopes

| OAuth scope | Maps to |
|-------------|---------|
| `storage:read` | `files:read`, `folders:read`, `drive:read`, `bucket:read`, `object:read` |
| `storage:write` | `files:write`, `folders:write`, `drive:write`, `bucket:write`, `object:write` |
| `storage:admin` | `shares:read`, `shares:write`, `links:manage` |

---

### Buckets

```
POST   /bucket              Create bucket
GET    /bucket               List buckets
GET    /bucket/:id           Get bucket details
PATCH  /bucket/:id           Update bucket settings
DELETE /bucket/:id           Delete bucket (must be empty)
POST   /bucket/:id/empty     Empty all objects in a bucket
```

#### Create bucket

```json
POST /bucket
{
  "name": "documents",
  "public": false,
  "file_size_limit": 10485760,
  "allowed_mime_types": ["image/png", "image/jpeg"]
}
```

Bucket names: 2-63 characters, lowercase alphanumeric with `.`, `-`, `_`.

---

### Objects

```
POST   /object/:bucket/*path         Upload (error if exists)
PUT    /object/:bucket/*path         Upload (upsert)
GET    /object/:bucket/*path         Download
HEAD   /object/:bucket/*path         Head (metadata only)
GET    /object/info/*bucket/path     Object metadata as JSON
POST   /object/list/:bucket          List objects in bucket
POST   /object/move                  Move/rename object
POST   /object/copy                  Copy object (cross-bucket)
DELETE /object/:bucket               Batch delete objects
```

#### Upload

```bash
PUT /object/docs/report.pdf
Content-Type: application/pdf
Authorization: Bearer sk_...

<binary body>
```

Max file size: 100 MB (use TUS for larger files).

#### List objects

```json
POST /object/list/documents
{
  "prefix": "reports/",
  "limit": 100,
  "offset": 0,
  "search": "quarterly",
  "sort_by": { "column": "name", "order": "asc" }
}
```

#### Move

```json
POST /object/move
{ "bucket": "docs", "from": "old/path.pdf", "to": "new/path.pdf" }
```

#### Copy (cross-bucket)

```json
POST /object/copy
{
  "from_bucket": "docs",
  "from_path": "report.pdf",
  "to_bucket": "archive",
  "to_path": "2024/report.pdf"
}
```

#### Batch delete

```json
DELETE /object/docs
{ "paths": ["file1.txt", "file2.txt", "old/report.pdf"] }
```

#### Public access

```
GET /object/public/:bucket/*path
```

No authentication required. Only works for objects in public buckets.

---

### Signed URLs

#### Create signed download URL

```json
POST /object/sign/documents
{ "path": "report.pdf", "expires_in": 3600 }
→ { "signed_url": "/sign/abc123..." }
```

Batch:

```json
POST /object/sign/documents
{ "paths": ["a.pdf", "b.pdf"], "expires_in": 3600 }
→ [{ "path": "a.pdf", "signed_url": "/sign/..." }, ...]
```

Max expiry: 7 days (604800 seconds). Default: 1 hour.

#### Create signed upload URL

```json
POST /object/upload/sign/:bucket/:path
{ "content_type": "image/png", "expires_in": 900 }
→ { "signed_url": "/upload/sign/abc123...", "token": "abc123..." }
```

#### Access signed URL

```
GET  /sign/:token              Download via signed URL
PUT  /upload/sign/:token       Upload via signed URL
```

No authentication required.

---

### TUS Resumable Uploads

[TUS v1.0.0](https://tus.io/) for reliable uploads of large files. Available at both paths:

- `/upload/resumable` (native)
- `/storage/v1/upload/resumable` (Supabase-compatible)

```
POST   /upload/resumable       Create upload
HEAD   /upload/resumable/:id   Get upload progress
PATCH  /upload/resumable/:id   Append chunk
DELETE /upload/resumable/:id   Cancel upload
```

Extensions: `creation`, `termination`, `expiration`.

#### Create

```
POST /upload/resumable
Authorization: Bearer sk_...
Tus-Resumable: 1.0.0
Upload-Length: 52428800
Upload-Metadata: bucketName <base64>,objectName <base64>,contentType <base64>

→ 201 Created
   Location: /upload/resumable/<upload-id>
   Upload-Expires: ...
```

#### Resume

```
PATCH /upload/resumable/<upload-id>
Tus-Resumable: 1.0.0
Upload-Offset: 0
Content-Type: application/offset+octet-stream

<binary chunk>
```

Max upload size: 5 GB.

---

### Drive

Higher-level file management operations used by the browse UI.

```
GET    /drive/search?query=...&bucket=...&type=...   Search files
GET    /drive/recent                                   Recent files
GET    /drive/starred                                  Starred files
GET    /drive/trash                                    Trashed files
GET    /drive/stats                                    Usage statistics
POST   /drive/rename                                   Rename file/folder
POST   /drive/move                                     Move items
POST   /drive/copy                                     Copy file
POST   /drive/trash                                    Trash items
POST   /drive/restore                                  Restore from trash
DELETE /drive/trash                                     Empty trash
PATCH  /drive/star                                     Toggle star
PATCH  /drive/description                              Update description
```

#### Stats response

```json
{
  "file_count": 142,
  "folder_count": 23,
  "total_size": 524288000,
  "trash_count": 5,
  "quota": 5368709120
}
```

---

### Files & Folders (Legacy Drive API)

```
PUT    /files/*path        Upload file
GET    /files/*path        Download file
DELETE /files/*path        Delete file
HEAD   /files/*path        File metadata

POST   /folders            Create folder
GET    /folders/*path      List folder contents
DELETE /folders/*path      Delete folder
```

---

### Shares

RBAC-based sharing with folder inheritance (most-specific match wins).

Roles: `owner` > `editor` > `viewer` (with `uploader` as a separate branch).

```
POST /shares            Create share
GET  /shares            List shares you've created
GET  /shared            List shares with you
```

#### Create share

```json
POST /shares
{
  "path": "docs/reports/",
  "grantee": "u/bob",
  "permission": "editor"
}
```

### Public Links

```
GET /p/:token           Access public link
GET /p/:token/*path     Access file within public link
```

---

### API Keys

Scoped, hashed API keys with optional path restrictions.

```
POST   /keys             Create API key
GET    /keys             List API keys
DELETE /keys/:id         Revoke API key
```

#### Create

```json
POST /keys
{
  "name": "deploy-bot",
  "scopes": ["bucket:read", "object:read", "object:write"],
  "path_prefix": "cdn/"
}
→ { "id": "...", "key": "sk_...", "name": "deploy-bot", ... }
```

Available scopes: `files:read`, `files:write`, `folders:read`, `folders:write`, `drive:read`, `drive:write`, `bucket:read`, `bucket:write`, `object:read`, `object:write`, `shares:read`, `shares:write`, `links:manage`.

Default: `*` (all scopes).

---

### Audit Log

```
GET /audit?limit=50&offset=0
→ [{ "actor": "u/alice", "action": "object.upload", "resource": "docs/report.pdf", "ts": ... }, ...]
```

---

### MCP (Model Context Protocol)

JSON-RPC 2.0 endpoint for AI tool use. Implements the MCP spec with OAuth 2.0 bearer auth.

```
GET  /mcp               Server info + capabilities
POST /mcp               JSON-RPC 2.0 requests
```

Tools: `storage_list`, `storage_read`, `storage_write`, `storage_search`, `storage_move`, `storage_delete`, `storage_share`, `storage_stats`.

Returns `WWW-Authenticate` header on 401 with resource metadata URL for automatic OAuth discovery.

---

## Authentication flow

The API supports three authentication methods:

| Method | Token format | TTL | Use case |
|--------|-------------|-----|----------|
| Session | Random hex | 2 hours | Browser (cookie-based) |
| API key | `sk_*` prefix, SHA-256 hashed | 90 days (or custom) | CLI, CI/CD, integrations |
| OAuth | API key via PKCE exchange | 90 days | Third-party apps, MCP |

Resolution order:
1. `Authorization: Bearer <token>` header
2. `session=<token>` cookie

Both resolve to an actor identity. Sessions get `*` (all) scopes. API keys get their configured scopes.

## Rate limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| Auth (challenge/verify) | 10 req | 60s |
| Magic links | 5 req | 300s |
| Registration | 5 req | 300s |
| Uploads | 100 req | 60s |
| Public access | 60 req | 60s |
| Shares | 30 req | 60s |
| Public links | 20 req | 60s |

Rate limiting uses D1 sliding window counters with probabilistic cleanup.

## Database

Schema is defined in `schema.sql`. Tables:

| Table | Purpose |
|-------|---------|
| `actors` | User identities (human/agent, email, public key) |
| `sessions` | Login sessions (2h TTL) |
| `challenges` | Public key auth challenges |
| `magic_tokens` | Magic link tokens (15min TTL) |
| `buckets` | Storage containers (name, public, limits) |
| `objects` | File metadata (path, size, type, R2 key) |
| `signed_urls` | Time-limited access tokens |
| `api_keys` | Scoped API keys (SHA-256 hashed) |
| `oauth_clients` | Dynamically registered OAuth clients |
| `oauth_codes` | Authorization codes (5min TTL, single-use) |
| `tus_uploads` | In-progress resumable uploads |
| `rate_limits` | Sliding window counters |
| `audit_log` | Action log for compliance |

### Migrations

```bash
# Local
npm run db:migrate

# Production
npm run db:migrate:remote
```

Incremental migrations live in `migrations/`.

## Development

```bash
# Start dev server (port 8787)
npm run dev

# Run tests
npm test

# Deploy
npm run deploy
```

### Testing

Tests use `@cloudflare/vitest-pool-workers` for running within the Workers runtime.

```bash
npm test
```

## Pages

The worker serves several HTML pages (server-rendered, no framework):

| Route | Page |
|-------|------|
| `/` | Landing page |
| `/developers` | Developer documentation |
| `/api` | API reference (rendered from docs.md) |
| `/pricing` | Pricing |
| `/browse` | File browser SPA |
| `/cli` | CLI install instructions |
| `/ai` | AI/MCP integration page |

Static assets are served from `public/` via Cloudflare Workers assets.

## License

MIT
