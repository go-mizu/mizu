# 0551: LiteIO — Supabase Storage-Compatible REST API

## Overview

LiteIO now supports two API layers simultaneously:

1. **S3 API** (default, always on) — AWS S3-compatible API at `/` with SigV4 auth
2. **REST API** (opt-in via `--rest`) — Supabase Storage-compatible REST API at `/storage/v1` with JWT auth

The REST API provides a modern, JSON-based alternative to the XML-based S3 API, compatible with Supabase Storage client libraries (`@supabase/storage-js`, etc.).

**Module:** `github.com/liteio-dev/liteio`
**Blueprint:** `blueprints/liteio/`
**REST transport:** `pkg/storage/transport/rest/`
**OpenAPI package:** `pkg/openapi/`

## Supabase Storage API Research

### Protocol Overview

Supabase Storage is a RESTful file storage service built on top of PostgreSQL and S3. It uses:

- **JWT authentication** — Bearer tokens from Supabase Auth (HS256 by default)
- **JSON request/response** — All responses are JSON (not XML like S3)
- **Role-based access** — `anon`, `authenticated`, `service_role` via JWT claims
- **Public/private buckets** — Public buckets accessible without auth
- **TUS protocol** — Resumable uploads following TUS 1.0.0 specification
- **HMAC-signed URLs** — Temporary access tokens for objects

### Key Differences from S3

| Aspect | S3 | Supabase Storage |
|--------|-----|-----------------|
| Auth | SigV4 (HMAC-SHA256) | JWT Bearer token |
| Response format | XML | JSON |
| Error format | `<Error><Code>...</Code></Error>` | `{"statusCode":N,"error":"...","message":"..."}` |
| Bucket creation | PUT /{bucket} | POST /bucket with JSON body |
| Object upload | PUT /{bucket}/{key} | POST /object/{bucket}/{path} |
| Object download | GET /{bucket}/{key} | GET /object/{bucket}/{path} |
| Object listing | GET /{bucket}?prefix=... | POST /object/list/{bucket} with JSON body |
| Signed URLs | SigV4 query params | HMAC token in ?token= param |
| Resumable upload | Multipart upload | TUS protocol 1.0.0 |
| Metadata | x-amz-meta-* headers | JSON metadata in request body |

### Supabase Storage API Endpoints

#### Bucket Operations

| Method | Path | Description |
|--------|------|-------------|
| `POST /bucket` | Create a new bucket | JSON body: `{name, public, file_size_limit, allowed_mime_types}` |
| `GET /bucket` | List all buckets | Query: `limit`, `offset`, `search` |
| `GET /bucket/{id}` | Get bucket details | Returns full bucket metadata |
| `PUT /bucket/{id}` | Update bucket | JSON body: `{public, file_size_limit, allowed_mime_types}` |
| `DELETE /bucket/{id}` | Delete bucket | Bucket must be empty |
| `POST /bucket/{id}/empty` | Empty bucket | Deletes all objects |

#### Object Operations

| Method | Path | Description |
|--------|------|-------------|
| `POST /object/{bucket}/{path}` | Upload object | Raw body, `Content-Type` header, `x-upsert` header |
| `GET /object/{bucket}/{path}` | Download object | Supports `Range` header, `?download` query |
| `PUT /object/{bucket}/{path}` | Update existing object | Must already exist |
| `DELETE /object/{bucket}/{path}` | Delete single object | |
| `DELETE /object/{bucket}` | Bulk delete objects | JSON body: `{prefixes: [...]}` |
| `POST /object/list/{bucket}` | List objects | JSON body: `{prefix, limit, offset, sortBy}` |
| `POST /object/move` | Move object | JSON body: `{bucketId, sourceKey, destinationKey}` |
| `POST /object/copy` | Copy object | JSON body: `{bucketId, sourceKey, destinationKey}` |
| `GET /object/public/{bucket}/{path}` | Public download | No auth required |
| `GET /object/authenticated/{bucket}/{path}` | Auth download | JWT required |
| `GET /object/info/{bucket}/{path}` | Object metadata | Returns file info as JSON |
| `POST /object/sign/{bucket}/{path}` | Create signed URL | JSON body: `{expiresIn}` |
| `POST /object/sign/{bucket}` | Batch signed URLs | JSON body: `{expiresIn, paths: [...]}` |
| `POST /object/upload/sign/{bucket}/{path}` | Upload signed URL | Returns URL + token for upload |
| `GET /object/render/{bucket}/{path}` | Render signed URL | Requires `?token=` query param |

#### TUS Resumable Upload Operations (TUS 1.0.0)

| Method | Path | Description |
|--------|------|-------------|
| `OPTIONS /upload/resumable/` | Discovery | Returns TUS capabilities |
| `POST /upload/resumable/{bucket}/{path}` | Create upload | `Upload-Length`, `Upload-Metadata` headers |
| `PATCH /upload/resumable/{bucket}/{path}` | Upload chunk | `Upload-Offset`, `Content-Type: application/offset+octet-stream` |
| `HEAD /upload/resumable/{bucket}/{path}` | Get status | Returns current offset |
| `DELETE /upload/resumable/{bucket}/{path}` | Cancel upload | Cleans up temp files |

### Authentication Model

#### JWT Token Structure (Supabase-compatible)

```json
{
  "sub": "user-uuid",
  "aud": "authenticated",
  "iss": "supabase",
  "iat": 1708300800,
  "exp": 1708304400,
  "role": "authenticated"
}
```

#### Supported Roles

| Role | Access Level |
|------|-------------|
| `anon` | Public bucket access only |
| `authenticated` | Read/write to permitted buckets |
| `service_role` | Full admin access (bucket CRUD, all objects) |

#### Authentication Flow

1. Client sends `Authorization: Bearer <jwt>` header
2. Server verifies HS256 signature against `--jwt-secret`
3. Claims extracted: `sub` (user ID), `role`, `exp` (expiration)
4. Role stored in request context for handler-level authorization

### Signed URL Token Format

LiteIO uses a custom HMAC-signed token format (not JWT) for signed URLs:

```
<base64url-encoded-payload>.<base64url-encoded-hmac-sha256-signature>
```

Payload:
```json
{
  "bucket": "my-bucket",
  "path": "folder/file.txt",
  "method": "GET",
  "exp": 1708304400
}
```

The token is passed as a `?token=` query parameter on render/upload URLs.

## Implementation Architecture

```
liteio serve --rest --jwt-secret "my-secret"
                │
                ▼
┌──────────────────────────────────────────────────┐
│              HTTP Server (Mizu)                   │
├──────────────┬───────────────────────────────────┤
│  / (S3 API)  │  /storage/v1 (REST API)           │
│              │  /docs (OpenAPI UI)                │
│  SigV4 Auth  │  JWT Auth                          │
│  XML         │  JSON                              │
├──────────────┴───────────────────────────────────┤
│         pkg/storage (unified interface)           │
├──────────┬──────────┬─────────┬─────────┬────────┤
│  local   │  memory  │ rabbit  │ usagi   │ devnull│
└──────────┴──────────┴─────────┴─────────┴────────┘
```

### Key Components

#### REST Transport (`pkg/storage/transport/rest/`)

| File | Description |
|------|-------------|
| `rest.go` | Server struct, route registration, error handling (Supabase format) |
| `auth.go` | JWT authentication middleware, claims context, role checking |
| `jwt.go` | HS256 JWT verify/create (zero external dependencies) |
| `handle_bucket.go` | Bucket CRUD (create, list, get, update, delete, empty) |
| `handle_object.go` | Object operations (upload, download, list, move, copy, signed URLs) |
| `handle_upload.go` | TUS 1.0.0 resumable upload protocol |
| `signed_url.go` | HMAC-signed URL token generation and validation |
| `openapi.go` | OpenAPI 3.1 document generation for all endpoints |

#### OpenAPI Package (`pkg/openapi/`)

| File | Description |
|------|-------------|
| `openapi.go` | Full OpenAPI 3.1 model (Document, PathItem, Operation, Schema, etc.) |
| `handle_docs.go` | Embedded HTML templates, UI selector (scalar/redoc/swagger/rapidoc/stoplight) |
| `docs/*.html` | 5 HTML templates for interactive API documentation |

### Storage Interface Mapping

The REST API maps to the same `storage.Storage` interface used by S3:

| REST Operation | Storage Interface Method |
|----------------|------------------------|
| Create Bucket | `Storage.CreateBucket(ctx, name, opts)` |
| Delete Bucket | `Storage.DeleteBucket(ctx, name, opts)` |
| List Buckets | `Storage.Buckets(ctx, limit, offset, opts)` |
| Get Bucket | `Bucket.Info(ctx)` |
| Upload Object | `Bucket.Write(ctx, key, reader, size, contentType, opts)` |
| Download Object | `Bucket.Open(ctx, key, offset, length, opts)` |
| Delete Object | `Bucket.Delete(ctx, key, opts)` |
| Stat Object | `Bucket.Stat(ctx, key, opts)` |
| List Objects | `Bucket.List(ctx, prefix, limit, offset, opts)` |
| Move Object | `Bucket.Move(ctx, dstKey, srcBucket, srcKey, opts)` |
| Copy Object | `Bucket.Copy(ctx, dstKey, srcBucket, srcKey, opts)` |

## Configuration

### CLI Flags (new)

| Flag | Default | Description |
|------|---------|-------------|
| `--rest` | `false` | Enable Supabase Storage REST API at `/storage/v1` |
| `--jwt-secret` | — | JWT secret for REST API authentication |

### Environment Variables (new)

| Variable | Description |
|----------|-------------|
| `LITEIO_REST` | Set `true` to enable REST API |
| `LITEIO_JWT_SECRET` | JWT secret key |

### Usage Examples

```bash
# S3-only mode (default)
liteio

# Dual mode: S3 + REST API (no auth on REST)
liteio --rest

# Dual mode with JWT auth on REST API
liteio --rest --jwt-secret "super-secret-key"

# Environment variables
LITEIO_REST=true LITEIO_JWT_SECRET=secret liteio
```

### Usage with Supabase JS Client

```javascript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  'http://localhost:9000',
  'your-jwt-token',  // anon key or service role key
  {
    storage: {
      url: 'http://localhost:9000/storage/v1'
    }
  }
)

// Upload
const { data, error } = await supabase.storage
  .from('my-bucket')
  .upload('folder/file.txt', fileBody, {
    contentType: 'text/plain',
    upsert: true,
  })

// Download
const { data: blob } = await supabase.storage
  .from('my-bucket')
  .download('folder/file.txt')

// Get public URL
const { data: { publicUrl } } = supabase.storage
  .from('my-bucket')
  .getPublicUrl('folder/file.txt')

// Create signed URL
const { data: { signedUrl } } = await supabase.storage
  .from('my-bucket')
  .createSignedUrl('folder/file.txt', 3600) // 1 hour

// List files
const { data: files } = await supabase.storage
  .from('my-bucket')
  .list('folder/', { limit: 100, offset: 0 })

// Move file
const { error } = await supabase.storage
  .from('my-bucket')
  .move('old/path.txt', 'new/path.txt')

// Delete file
const { error } = await supabase.storage
  .from('my-bucket')
  .remove(['folder/file.txt'])
```

### Usage with curl

```bash
# Set JWT token
TOKEN="eyJ..."

# Create bucket
curl -X POST http://localhost:9000/storage/v1/bucket \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-bucket","public":true}'

# Upload file
curl -X POST http://localhost:9000/storage/v1/object/my-bucket/docs/readme.txt \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Hello, LiteIO!"

# Download file
curl http://localhost:9000/storage/v1/object/my-bucket/docs/readme.txt \
  -H "Authorization: Bearer $TOKEN"

# List objects
curl -X POST http://localhost:9000/storage/v1/object/list/my-bucket \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"prefix":"docs/","limit":100}'

# Create signed URL
curl -X POST http://localhost:9000/storage/v1/object/sign/my-bucket/docs/readme.txt \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expiresIn":3600}'

# Public access (no auth)
curl http://localhost:9000/storage/v1/object/public/my-bucket/docs/readme.txt

# OpenAPI docs
open http://localhost:9000/docs/
```

### TUS Resumable Upload Example

```bash
# Create upload
curl -X POST http://localhost:9000/storage/v1/upload/resumable/my-bucket/large-file.zip \
  -H "Authorization: Bearer $TOKEN" \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Length: 10485760" \
  -H "Upload-Metadata: content-type YXBwbGljYXRpb24vemlw" \
  -D -

# Upload chunk (returns 204 No Content)
curl -X PATCH http://localhost:9000/storage/v1/upload/resumable/my-bucket/large-file.zip \
  -H "Authorization: Bearer $TOKEN" \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Offset: 0" \
  -H "Content-Type: application/offset+octet-stream" \
  --data-binary @chunk1.bin

# Check upload status
curl -I http://localhost:9000/storage/v1/upload/resumable/my-bucket/large-file.zip \
  -H "Authorization: Bearer $TOKEN" \
  -H "Tus-Resumable: 1.0.0"
```

## API Documentation

When REST API is enabled, interactive API documentation is available at:

- **GET /docs** — OpenAPI 3.1 JSON specification
- **GET /docs/** — Interactive documentation UI

The documentation UI supports multiple renderers via `?ui=` query parameter:

| UI | URL |
|----|-----|
| Scalar (default) | `/docs/?ui=scalar` |
| Redoc | `/docs/?ui=redoc` |
| Swagger UI | `/docs/?ui=swagger` |
| RapiDoc | `/docs/?ui=rapidoc` |
| Stoplight Elements | `/docs/?ui=stoplight` |

## Error Response Format

All REST API errors follow the Supabase Storage error format:

```json
{
  "statusCode": 404,
  "error": "Not Found",
  "message": "object not found"
}
```

## Not Implemented (vs full Supabase Storage)

These Supabase Storage features are not implemented in v1:

- Row-level security (RLS) policies
- PostgreSQL-backed metadata (uses in-memory/filesystem instead)
- Image transformations (`?width=`, `?height=`, etc.)
- Bucket metadata persistence across restarts (in-memory drivers)
- S3 backend proxy mode
- Webhook notifications
- Rate limiting per user/role

These may be added in future versions as needed.
