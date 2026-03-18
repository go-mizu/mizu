# storage.now API {#overview}

File storage for humans and AI agents. Upload, organize, and share files via a simple REST API.

Base URL: `https://storage.liteio.workers.dev`

| Feature | Detail |
|---|---|
| Max file size | 100 MB (single PUT), unlimited via presigned URLs |
| Auth | Ed25519 challenge-response or magic link |
| Storage | S3-compatible object storage (R2) |
| Metadata | SQLite at edge (D1) |
| Soft delete | Files go to trash first, permanent delete on empty |

# Authentication {#auth}

All API routes (except registration, auth, and public pages) require a Bearer token or session cookie.

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

## Logout {#logout}

```
POST /auth/logout
GET /auth/logout
```

Ends the current session. Accepts both POST and GET (for link-based logout).

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

Parent folders are auto-created. Content-Type is auto-detected from extension if not provided. Max 100 MB.

## Download a File {#download}

```
GET /files/docs/readme.md
Authorization: Bearer <token>

// Response: file bytes
// Headers: Content-Type, Content-Length, ETag, Last-Modified, Content-Disposition
```

Trashed files are not downloadable. Sets `accessed_at` on the file.

## Delete a File {#delete-file}

```
DELETE /files/docs/readme.md
Authorization: Bearer <token>

// Response
{"deleted": true}
```

Permanently deletes the file and its R2 object. To soft-delete, use `POST /drive/trash` instead.

## File Metadata (HEAD) {#head-file}

```
HEAD /files/docs/readme.md
Authorization: Bearer <token>

// Response headers: Content-Type, Content-Length, Last-Modified
```

# Folders {#folders}

Folders are virtual metadata entries. Created automatically when uploading files, or explicitly.

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

Parent folders are auto-created. Returns `created: false` if already exists.

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
    {"name": "reports", "path": "docs/reports/", "is_folder": true, "starred": false, ...},
    {"name": "readme.md", "path": "docs/readme.md", "is_folder": false, "size": 1234, "starred": false, ...}
  ]
}
```

Items are sorted: folders first, then files alphabetically. Trashed items are excluded.

## Delete a Folder {#delete-folder}

```
DELETE /folders/docs/reports
Authorization: Bearer <token>

// Response
{"deleted": true}
```

Folder must be empty. Delete or move files inside first.

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

# Drive {#drive}

Drive endpoints provide Google Drive-class file management features: star, rename, move, copy, trash, restore, search, and more. All require authentication.

## Star / Unstar {#star}

```
PATCH /drive/star
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/readme.md", "starred": 1}

// Response
{"path": "docs/readme.md", "starred": 1}
```

Set `starred` to `1` to star, `0` to unstar.

## Rename {#rename}

```
POST /drive/rename
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/old-name.md", "new_name": "new-name.md"}

// Response
{
  "old_path": "docs/old-name.md",
  "new_path": "docs/new-name.md",
  "name": "new-name.md"
}
```

Renaming a folder cascades the path change to all children. R2 objects are copied to the new key and the old key is deleted.

## Move {#move}

```
POST /drive/move
Authorization: Bearer <token>
Content-Type: application/json

{"paths": ["docs/readme.md", "images/logo.svg"], "destination": "archive/"}

// Response
{"moved": 2}
```

Moves files and folders to a new parent directory. Destination must end with `/` or be empty string for root. Folder moves cascade to all children.

## Copy {#copy}

```
POST /drive/copy
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/readme.md"}

// Response
{
  "id": "o_...",
  "path": "docs/readme (copy).md",
  "name": "readme (copy).md"
}
```

Creates a duplicate file. Appends `(copy)` to the name, incrementing if needed. Folder copy is not supported.

## Trash {#trash}

```
POST /drive/trash
Authorization: Bearer <token>
Content-Type: application/json

{"paths": ["docs/readme.md", "images/"]}

// Response
{"trashed": 2}
```

Soft-deletes items by setting `trashed_at`. Folder trash cascades to all children. Trashed items are excluded from folder listings and search.

## Restore {#restore}

```
POST /drive/restore
Authorization: Bearer <token>
Content-Type: application/json

{"paths": ["docs/readme.md", "images/"]}

// Response
{"restored": 2}
```

Restores items from trash by clearing `trashed_at`. Folder restore cascades to all children.

## Empty Trash {#empty-trash}

```
DELETE /drive/trash
Authorization: Bearer <token>

// Response
{"deleted": 5}
```

Permanently deletes all trashed items. Removes R2 objects and D1 metadata. Shares on deleted objects are also removed. This action is irreversible.

## List Trash {#list-trash}

```
GET /drive/trash
Authorization: Bearer <token>

// Response
{
  "items": [
    {"id": "o_...", "name": "readme.md", "path": "docs/readme.md", "trashed_at": 1710700000000, ...}
  ]
}
```

## Recent Files {#recent}

```
GET /drive/recent
Authorization: Bearer <token>

// Response
{
  "items": [...]
}
```

Returns the 50 most recently accessed files (by `accessed_at`), excluding folders and trashed items.

## Starred Items {#starred}

```
GET /drive/starred
Authorization: Bearer <token>

// Response
{
  "items": [...]
}
```

Returns all starred, non-trashed items (files and folders).

## Search {#search}

```
GET /drive/search?q=readme&type=text&starred=1
Authorization: Bearer <token>

// Response
{
  "query": "readme",
  "items": [...]
}
```

Search files by name. Optional filters:

| Param | Description |
|---|---|
| q | Search term (matched against name with LIKE) |
| type | Filter by content_type prefix (e.g. `image`, `text`, `application/pdf`) |
| starred | Set to `1` to only return starred items |

## Drive Stats {#stats}

```
GET /drive/stats
Authorization: Bearer <token>

// Response
{
  "file_count": 24,
  "folder_count": 14,
  "total_size": 163405923,
  "trash_count": 2,
  "quota": 5368709120
}
```

Returns storage usage summary. Quota is 5 GB.

## Update Description {#description}

```
PATCH /drive/description
Authorization: Bearer <token>
Content-Type: application/json

{"path": "docs/readme.md", "description": "Project overview"}

// Response
{"path": "docs/readme.md", "description": "Project overview"}
```

# All Endpoints {#endpoints}

| Method | Path | Description | Auth |
|---|---|---|---|
| POST | /actors | Register actor | No |
| POST | /auth/challenge | Get challenge nonce | No |
| POST | /auth/verify | Verify signature, get token | No |
| POST | /auth/magic-link | Request magic link | No |
| GET | /auth/magic/:token | Verify magic link | No |
| POST | /auth/logout | End session | No |
| GET | /auth/logout | End session (link) | No |
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
| PATCH | /drive/star | Star or unstar item | Yes |
| POST | /drive/rename | Rename file or folder | Yes |
| POST | /drive/move | Move items to new folder | Yes |
| POST | /drive/copy | Duplicate a file | Yes |
| POST | /drive/trash | Trash items (soft delete) | Yes |
| POST | /drive/restore | Restore items from trash | Yes |
| DELETE | /drive/trash | Empty trash (permanent) | Yes |
| GET | /drive/trash | List trashed items | Yes |
| GET | /drive/recent | Recently accessed files | Yes |
| GET | /drive/starred | Starred items | Yes |
| GET | /drive/search | Search files by name | Yes |
| GET | /drive/stats | Storage usage stats | Yes |
| PATCH | /drive/description | Update file description | Yes |
