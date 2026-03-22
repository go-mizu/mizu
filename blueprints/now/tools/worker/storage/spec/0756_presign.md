# 0756: Presigned R2 URLs — Zero-Proxy Data Transfer

## Problem

All file reads and writes proxy through the Cloudflare Worker:

```
Browser/CLI → Worker (auth + buffer/stream) → R2
```

This has three costs:
1. **Memory**: `PUT /f/*` calls `arrayBuffer()`, buffering the entire file (up to 100 MB) in Worker RAM.
2. **Bandwidth**: Every byte passes through the Worker — double egress on reads, double ingress on writes.
3. **Latency**: An extra hop adds ~5–20 ms per request.

The TUS resumable upload endpoint (`/upload/resumable`) referenced by `browse.js` never existed — uploads were silently broken. Additionally, `btoa()` crashes on non-ASCII filenames (e.g. Chinese characters).

## Goals

1. File data transfers directly between client and R2 — Worker only handles auth and URL signing.
2. Browser uploads show real-time progress via `XMLHttpRequest.upload.onprogress`.
3. CLI uploads and downloads use presigned URLs instead of proxying through `/f/*`.
4. Share links (`/s/:token`) redirect to presigned R2 URLs instead of proxying file content.
5. Existing REST API (`PUT/GET /f/*`) remains for backward compatibility but streams instead of buffering.

## Non-Goals

- R2 custom domain integration (presigned URLs require the S3-compatible endpoint).
- Multipart uploads for files > 5 GB.
- Client-side encryption.

---

## Architecture

### Data Flow — Before vs After

```
BEFORE (proxy):
  Client ──PUT body──→ Worker ──put()──→ R2
  Client ←─stream────← Worker ←─get()──← R2

AFTER (presign):
  Client ──POST /presign/upload──→ Worker (auth, sign, ~2ms)
  Client ──PUT body──────────────→ R2 directly
  Client ──POST /presign/complete→ Worker (HEAD + DB update)

  Client ──GET /presign/read/*───→ Worker (auth, sign, ~2ms)
  Client ──GET presigned-url─────→ R2 directly
```

### S3 V4 Presigned URL Signing

Presigned URLs use AWS Signature Version 4 (the standard R2 supports). The signing is implemented in `src/lib/presign.ts` using only the Web Crypto API — zero npm dependencies.

**Signing algorithm:**

```
1. Canonical Request = METHOD + URI + QueryString + Headers + "UNSIGNED-PAYLOAD"
2. String to Sign    = "AWS4-HMAC-SHA256" + timestamp + scope + SHA256(canonical)
3. Signing Key       = HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), "auto"), "s3"), "aws4_request")
4. Signature         = HMAC(signingKey, stringToSign)
5. URL               = endpoint/bucket/key?queryParams&X-Amz-Signature=signature
```

**R2-specific details:**
- Region is always `"auto"`
- Endpoint format: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`
- Bucket name appears in the URL path (path-style, not virtual-hosted)
- Object keys use format `<actor>/<user-path>` (e.g. `u/test.y9td/docs/report.pdf`)

### Environment Variables (Worker Secrets)

| Secret | Value | Purpose |
|--------|-------|---------|
| `R2_ENDPOINT` | `https://<ACCOUNT_ID>.r2.cloudflarestorage.com` | S3-compatible endpoint |
| `R2_ACCESS_KEY_ID` | From CF dashboard → R2 → API Tokens | Signing identity |
| `R2_SECRET_ACCESS_KEY` | From CF dashboard → R2 → API Tokens | Signing secret |
| `R2_BUCKET_NAME` | `storage-files` (optional, defaults) | Bucket name in URL path |

### R2 Bucket CORS Configuration

Required for browser uploads/downloads to the S3 endpoint (cross-origin from `storage.liteio.dev`):

```json
{
  "rules": [{
    "allowed": {
      "origins": ["https://storage.liteio.dev"],
      "methods": ["GET", "PUT", "HEAD"],
      "headers": ["Content-Type"]
    },
    "exposed_headers": ["ETag", "Content-Length"],
    "max_age_seconds": 3600
  }]
}
```

---

## API Routes

### `POST /presign/upload` (auth required)

Request a presigned PUT URL for direct-to-R2 upload.

**Request:**
```json
{ "path": "docs/report.pdf", "content_type": "application/pdf" }
```

**Response:**
```json
{
  "url": "https://<account>.r2.cloudflarestorage.com/storage-files/<actor>/docs/report.pdf?X-Amz-...",
  "content_type": "application/pdf",
  "expires_in": 3600
}
```

**Client then PUTs directly to the returned URL** — the Worker is not involved in the data transfer.

### `POST /presign/complete` (auth required)

Called after a presigned upload finishes. Verifies the object in R2 via `HEAD`, then updates the D1 search index.

**Request:**
```json
{ "path": "docs/report.pdf" }
```

**Response:**
```json
{ "path": "docs/report.pdf", "name": "report.pdf", "size": 1048576, "type": "application/pdf", "updated_at": 1774005339496 }
```

### `GET /presign/read/*` (auth required)

Returns a presigned GET URL for direct-from-R2 download.

**Response:**
```json
{
  "url": "https://<account>.r2.cloudflarestorage.com/storage-files/<actor>/docs/report.pdf?X-Amz-...",
  "size": 1048576,
  "type": "application/pdf",
  "etag": "abc123",
  "expires_in": 3600
}
```

### `GET /s/:token` (no auth)

Share links now **302 redirect** to a presigned R2 URL instead of proxying the file body through the Worker.

---

## Browser Changes (`browse.js`)

### Upload Flow

```js
// 1. Get presigned URL
POST /presign/upload { path, content_type }
// 2. Upload directly to R2 with XHR (progress events)
xhr.open('PUT', presignedUrl)
xhr.upload.onprogress = trackProgress
xhr.send(file)
// 3. Confirm (updates DB)
POST /presign/complete { path }
```

### Download / Preview Flow

```js
// resolveFileUrl(path) → presigned URL
GET /presign/read/<path>  →  { url }
// Use URL for img src, iframe src, fetch(), download link
```

All binary previews (images, PDFs, audio, video) use `data-file-path` attributes resolved post-render. Text previews fetch content via the presigned URL.

### Removed

- TUS resumable upload code (`/upload/resumable` — endpoint never existed)
- `btoa()` calls on file paths (crashed on non-ASCII characters)
- All fallback-to-proxy paths — presigned URLs are the only data path

---

## CLI Changes (`@liteio/storage-cli`)

### Upload (`put` command)

Before: `PUT /f/<path>` with file body through Worker.

After:
```
1. POST /presign/upload  → { url, content_type }
2. PUT  <presigned-url>   → direct to R2 (with Content-Type header)
3. POST /presign/complete → { path, size, type, updated_at }
```

Progress tracking via `Content-Length` header matching.

### Download (`get` / `cat` commands)

Before: `GET /f/<path>` streamed through Worker.

After:
```
1. GET /presign/read/<path> → { url, size, type }
2. GET <presigned-url>       → direct from R2
```

Stream directly from R2 — zero Worker bandwidth.

---

## REST API (`PUT/GET /f/*`) — Backward Compatibility

The existing `/f/*` endpoints remain for MCP tools and API clients that don't support presigned URLs.

**Write change**: `writeFile` now streams the request body to R2 (`BUCKET.put(key, stream)`) instead of buffering with `arrayBuffer()`. File size is obtained via `HEAD` after upload. This eliminates the 100 MB Worker memory spike.

**Read**: Unchanged — already streamed via `ReadableStream`.

---

## File Changelist

### New Files

| File | Purpose |
|------|---------|
| `src/lib/presign.ts` | S3 V4 presigned URL generator (Web Crypto, zero deps) |
| `src/routes/presign.ts` | `POST /presign/upload`, `POST /presign/complete`, `GET /presign/read/*` |
| `spec/0756_presign.md` | This specification |

### Modified Files

| File | Changes |
|------|---------|
| `src/types.ts` | Added `R2_ENDPOINT`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET_NAME` to `Env` |
| `src/index.ts` | Register presign routes |
| `src/routes/files.ts` | `writeFile` streams body instead of `arrayBuffer()` |
| `src/routes/share.ts` | `GET /s/:token` redirects to presigned URL instead of proxying |
| `public/browse.js` | Upload via presigned PUT + XHR progress; download/preview via presigned GET; removed TUS code |
| `vitest.config.ts` | Added R2 test bindings |
| `src/__tests__/storage.test.ts` | Share test expects 302 redirect with presigned URL |
| `tools/cli/storage-cli/bin/storage.mjs` | Upload and download use presigned URLs |
| `tools/cli/storage-cli/package.json` | Version bump |

---

## Testing

### Unit Tests (vitest)

- Share link returns 302 with `X-Amz-Signature` in Location header
- All 58 existing tests pass with presign bindings in test config

### Integration Tests (curl)

```bash
TOKEN="<session-token>"

# Presigned read
curl -H "Authorization: Bearer $TOKEN" https://storage.liteio.dev/presign/read/README.md
# → { "url": "https://...r2.cloudflarestorage.com/...", "size": 502 }

# Download directly from R2
curl "$(curl -s -H "Authorization: Bearer $TOKEN" https://storage.liteio.dev/presign/read/README.md | jq -r .url)"
# → file content (no Worker proxy)

# Presigned upload
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"path":"test.txt","content_type":"text/plain"}' \
  https://storage.liteio.dev/presign/upload
# → { "url": "https://...?X-Amz-Signature=..." }

# Upload directly to R2
curl -X PUT "<presigned-url>" -H "Content-Type: text/plain" -d "Hello"
# → 200 OK

# Confirm upload
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"path":"test.txt"}' https://storage.liteio.dev/presign/complete
# → { "path": "test.txt", "size": 5, ... }
```

### CLI Tests

```bash
# Upload via presigned URL
storage put test.txt docs/test.txt
# → should show "Uploaded docs/test.txt (X B)"

# Download via presigned URL
storage get docs/test.txt /tmp/test.txt
# → should show "Downloaded test.txt (X B)"

# Cat via presigned URL
storage cat docs/test.txt
# → file content to stdout
```
