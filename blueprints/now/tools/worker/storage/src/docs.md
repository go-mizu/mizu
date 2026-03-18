# storage.now API {#overview}

File storage for humans and AI agents. Upload, organize, and share files via a simple REST API.

Base URL: `https://storage.liteio.workers.dev`

| Feature | Detail |
|---|---|
| Max file size | 100 MB (single PUT) |
| Auth | Ed25519 challenge-response or magic link |
| Storage | S3-compatible object storage |
| Metadata | SQLite at edge |

# Authentication {#auth}

All API routes (except registration and auth endpoints) require a Bearer token or session cookie.

## Register an Actor {#register}

```
POST /actors
Content-Type: application/json

{
  "actor": "a/my-agent",
  "public_key": "<base64url-ed25519-public-key>",
  "type": "agent"
}

// Response
{
  "actor": "a/my-agent",
  "created": true
}
```

Humans use `u/name`, agents use `a/name`. Agents must provide an Ed25519 public key.

## Challenge-Response Auth {#challenge}

```
// Step 1: Get a challenge nonce
POST /auth/challenge
{"actor": "a/my-agent"}

// Response
{
  "challenge_id": "ch_...",
  "nonce": "abc123...",
  "expires_at": "2026-03-18T..."
}

// Step 2: Sign the nonce with your Ed25519 private key, then verify
POST /auth/verify
{
  "challenge_id": "ch_...",
  "actor": "a/my-agent",
  "signature": "<base64url-signature-of-nonce>"
}

// Response
{
  "access_token": "...",
  "expires_at": "2026-03-18T..."
}

// Step 3: Use the token
Authorization: Bearer <access_token>
```

## Magic Link Auth (Humans) {#magic-link}

```
POST /auth/magic-link
{"email": "alice@example.com"}

// Response
{"ok": true, "magic_link": "https://storage.liteio.workers.dev/auth/magic/..."}
```

Opens a session in the browser. No password needed.

# Files {#files}

Upload, download, and delete files using their path.

## Upload a File {#upload}

```
PUT /files/docs/readme.md
Authorization: Bearer <token>
Content-Type: text/markdown

<file bytes>

// Response (201 Created or 200 Updated)
{
  "id": "o_...",
  "path": "docs/readme.md",
  "name": "readme.md",
  "content_type": "text/markdown",
  "size": 1234,
  "created_at": 1710700000000
}
```

Parent folders are auto-created. Content-Type is auto-detected from extension if not provided.

## Download a File {#download}

```
GET /files/docs/readme.md
Authorization: Bearer <token>

// Response: file bytes
// Headers: Content-Type, Content-Length, ETag, Last-Modified, Content-Disposition
```

## Delete a File {#delete-file}

```
DELETE /files/docs/readme.md
Authorization: Bearer <token>

// Response
{"deleted": true}
```

## File Metadata (HEAD) {#head-file}

```
HEAD /files/docs/readme.md
Authorization: Bearer <token>

// Response headers: Content-Type, Content-Length, Last-Modified
```

# Folders {#folders}

Folders are virtual — they exist as metadata entries. Created automatically when uploading files, or explicitly.

## Create a Folder {#create-folder}

```
POST /folders
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/reports"}

// Response (201)
{
  "id": "o_...",
  "path": "docs/reports/",
  "name": "reports",
  "created": true
}
```

## List Folder Contents {#list-folder}

```
// List root
GET /folders
Authorization: Bearer <token>

// List subfolder
GET /folders/docs
Authorization: Bearer <token>

// Response
{
  "path": "docs/",
  "items": [
    {"name": "reports", "path": "docs/reports/", "is_folder": true, ...},
    {"name": "readme.md", "path": "docs/readme.md", "is_folder": false, "size": 1234, ...}
  ]
}
```

Items are sorted: folders first, then files alphabetically.

## Delete a Folder {#delete-folder}

```
DELETE /folders/docs/reports
Authorization: Bearer <token>

// Response
{"deleted": true}
```

Folder must be empty. Delete files inside first.

# Shares {#shares}

Share files with other actors. Grant read or write access.

## Create a Share {#create-share}

```
POST /shares
Authorization: Bearer <token>
Content-Type: application/json

{
  "path": "docs/readme.md",
  "grantee": "u/bob",
  "permission": "read"
}

// Response (201)
{
  "id": "sh_...",
  "path": "docs/readme.md",
  "grantee": "u/bob",
  "permission": "read",
  "created_at": 1710700000000
}
```

## List Shares {#list-shares}

```
GET /shares
Authorization: Bearer <token>

// Response
{
  "given": [...],    // shares you created
  "received": [...]  // shares granted to you
}
```

## Revoke a Share {#revoke-share}

```
DELETE /shares/sh_abc123
Authorization: Bearer <token>

// Response
{"deleted": true}
```

## Shared Files {#shared-files}

```
// List files shared with you
GET /shared
Authorization: Bearer <token>

// Download a shared file
GET /shared/u%2Falice/docs/readme.md
Authorization: Bearer <token>
```

# Presigned URLs {#presign}

For large files or high-throughput scenarios, use presigned URLs to upload/download directly to object storage — bypassing the worker entirely. No file bytes pass through the API server.

**Flow:** Request a signed URL → PUT/GET directly to storage → Confirm upload (for writes).

## Get Upload URL {#presign-upload}

```
POST /presign/upload
Authorization: Bearer <token>
Content-Type: application/json

{
  "path": "models/v3.bin",
  "content_type": "application/octet-stream",
  "expires": 3600
}

// Response
{
  "upload_url": "https://...signed-url...",
  "path": "models/v3.bin",
  "content_type": "application/octet-stream",
  "expires_in": 3600,
  "method": "PUT",
  "headers": {"Content-Type": "application/octet-stream"}
}

// Then upload directly:
curl -X PUT "UPLOAD_URL" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @model-v3.bin
```

The signed URL is valid for `expires` seconds (default 1 hour, max 24 hours). After uploading, call `/presign/complete` to sync metadata.

## Get Download URL {#presign-download}

```
POST /presign/download
Authorization: Bearer <token>
Content-Type: application/json

{"path": "models/v3.bin", "expires": 3600}

// Response
{
  "download_url": "https://...signed-url...",
  "path": "models/v3.bin",
  "name": "v3.bin",
  "content_type": "application/octet-stream",
  "size": 47185920,
  "expires_in": 3600
}

// Then download directly:
curl -o v3.bin "DOWNLOAD_URL"
```

No confirmation needed for downloads. The file must exist (verified against metadata before signing).

## Confirm Upload {#presign-complete}

```
POST /presign/complete
Authorization: Bearer <token>
Content-Type: application/json

{"path": "models/v3.bin"}

// Response
{
  "id": "o_...",
  "path": "models/v3.bin",
  "name": "v3.bin",
  "content_type": "application/octet-stream",
  "size": 47185920,
  "created_at": 1710700000000
}
```

Call after a presigned upload succeeds. Verifies the file exists in storage, reads its actual size/type, and creates or updates the metadata record. Parent folders are auto-created.

# All Endpoints {#endpoints}

| Method | Path | Description | Auth |
|---|---|---|---|
| POST | /actors | Register actor | No |
| POST | /auth/challenge | Get challenge nonce | No |
| POST | /auth/verify | Verify signature, get token | No |
| POST | /auth/magic-link | Request magic link | No |
| GET | /auth/magic/:token | Verify magic link | No |
| POST | /auth/logout | End session | No |
| PUT | /files/*path | Upload file | Yes |
| GET | /files/*path | Download file | Yes |
| DELETE | /files/*path | Delete file | Yes |
| HEAD | /files/*path | File metadata | Yes |
| POST | /presign/upload | Get presigned upload URL | Yes |
| POST | /presign/download | Get presigned download URL | Yes |
| POST | /presign/complete | Confirm presigned upload | Yes |
| POST | /folders | Create folder | Yes |
| GET | /folders | List root | Yes |
| GET | /folders/*path | List folder contents | Yes |
| DELETE | /folders/*path | Delete empty folder | Yes |
| POST | /shares | Create share | Yes |
| GET | /shares | List shares | Yes |
| DELETE | /shares/:id | Revoke share | Yes |
| GET | /shared | Files shared with me | Yes |
| GET | /shared/:owner/*path | Download shared file | Yes |
