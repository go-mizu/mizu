# API Reference {#overview}

Complete reference for the storage.now API. Organize files into buckets, upload and download objects, and generate time-limited signed URLs for secure sharing — all through a consistent REST interface inspired by Supabase Storage.

Base URL: `https://storage.now`

All requests and responses use JSON, except file uploads and downloads which transfer raw bytes with the appropriate `Content-Type` header. Errors follow a consistent shape: `{"error": {"code": "...", "message": "..."}}`.

## Quick Start {#quickstart}

Get up and running in five steps. Register an identity, authenticate, create a bucket, upload a file, and share it with a signed URL.

```bash
# 1. Register an agent identity
curl -X POST storage.now/auth/register \
  -H "Content-Type: application/json" \
  -d '{"actor":"a/my-app","public_key":"<base64url-ed25519>"}'

# 2. Request a challenge and sign it
curl -X POST storage.now/auth/challenge \
  -d '{"actor":"a/my-app"}'
# → sign the nonce with your Ed25519 private key, then:
curl -X POST storage.now/auth/verify \
  -d '{"challenge_id":"ch_...","actor":"a/my-app","signature":"<base64url>"}'
# → {"access_token":"..."}

# 3. Create a bucket
curl -X POST storage.now/bucket \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"my-files","public":false}'

# 4. Upload a file
curl -X PUT storage.now/object/my-files/hello.txt \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Hello, world!"

# 5. Generate a signed URL to share it
curl -X POST storage.now/object/sign/my-files \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"path":"hello.txt","expires_in":3600}'
# → {"signed_url":"/sign/tok_abc123"}
```

## Authentication {#authentication}

All routes except `/auth/*`, `/sign/*`, `/upload/sign/*`, and `/object/public/*` require a Bearer token in the `Authorization` header. Tokens are obtained through one of three authentication methods, depending on your use case.

```
Authorization: Bearer <token>
```

| Method | Best for | How it works |
|---|---|---|
| Ed25519 | Machines, scripts, CI/CD | Register an Ed25519 public key, then sign server-issued challenges to prove identity. Tokens expire after 2 hours. |
| Magic link | Human users | Provide an email address to receive a one-time login link. Clicking the link sets a session cookie. |
| API key | Long-lived integrations | Generate a scoped API key via `POST /keys`. Use the returned `sk_...` token as a Bearer token. Keys can be restricted to specific operations and path prefixes. |

## Rate Limits {#rate-limits}

The API enforces per-IP sliding window rate limits. When a limit is exceeded, the server responds with `429 Too Many Requests` and a `Retry-After` header indicating how many seconds to wait before retrying.

| Endpoint | Limit | Window |
|---|---|---|
| POST /auth/register | 10 requests | 1 minute |
| POST /auth/challenge | 20 requests | 1 minute |
| POST /auth/magic-link | 5 requests | 1 minute |
| PUT /object/* | 100 requests | 1 minute |
| GET /sign/* | 200 requests | 1 minute |

# Buckets {#buckets}

Buckets are top-level containers that organize your objects into logical groups — similar to S3 buckets or filesystem mount points. Each bucket belongs to a single owner and has its own visibility setting, file size limit, and MIME type allowlist.

Public buckets expose their objects for unauthenticated read access at `/object/public/:bucket/*path`, making them ideal for serving avatars, static assets, or any content that should be freely accessible. Private buckets require a valid Bearer token or signed URL for every operation.

## POST /bucket {#bucket-create}

Creates a new bucket. Bucket names must be globally unique within your account, between 2 and 63 characters long, and contain only lowercase letters, numbers, dots, hyphens, and underscores. The name must start with a letter or number.

You can optionally set a `file_size_limit` to reject uploads that exceed a certain size, and an `allowed_mime_types` array to restrict which file types can be stored. Both constraints are enforced at upload time and apply to all objects in the bucket.

Returns the newly created bucket object with a `201 Created` status.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| name | string | Yes | Bucket name. Must be 2–63 characters, lowercase alphanumeric with `.`, `-`, `_`. Must start with a letter or number. Example: `"avatars"`, `"project-assets"`. |
| public | boolean | No | Whether objects in this bucket are readable without authentication. When set to `true`, objects are accessible at `/object/public/:bucket/*path`. Defaults to `false`. |
| file_size_limit | integer | No | Maximum allowed file size in bytes for objects uploaded to this bucket. Set to `null` or omit to allow files of any size. For example, `5242880` limits uploads to 5 MB. |
| allowed_mime_types | string[] | No | Array of MIME type strings that are permitted for upload. For example, `["image/png", "image/jpeg", "image/webp"]` restricts the bucket to image files only. Set to `null` or omit to allow any content type. |

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | Unique bucket identifier, prefixed with `bk_`. Example: `"bk_a1b2c3"`. |
| name | string | The bucket name as provided in the request. |
| owner | string | Actor identifier of the bucket owner. |
| public | boolean | Whether the bucket allows unauthenticated reads. |
| file_size_limit | integer\|null | Maximum file size in bytes, or `null` if unrestricted. |
| allowed_mime_types | string[]\|null | Allowed MIME types, or `null` if unrestricted. |
| created_at | integer | Creation timestamp as Unix epoch in milliseconds. |

```bash
curl -X POST storage.now/bucket \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "avatars",
    "public": true,
    "file_size_limit": 5242880,
    "allowed_mime_types": ["image/png", "image/jpeg"]
  }'
```

```javascript
const res = await fetch("https://storage.now/bucket", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "avatars",
    public: true,
    file_size_limit: 5242880,
    allowed_mime_types: ["image/png", "image/jpeg"],
  }),
});
const bucket = await res.json();
```

```json
{
  "id": "bk_a1b2c3",
  "name": "avatars",
  "owner": "a/my-app",
  "public": true,
  "file_size_limit": 5242880,
  "allowed_mime_types": ["image/png", "image/jpeg"],
  "created_at": 1710892800000
}
```

## GET /bucket {#bucket-list}

Returns a list of all buckets owned by the authenticated actor. Each bucket in the response includes its name, visibility, and creation timestamp. Use this to discover available buckets before listing their contents.

**Response** — Array of bucket objects.

```bash
curl storage.now/bucket \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/bucket", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const buckets = await res.json();
```

```json
[
  {"id": "bk_a1b2c3", "name": "avatars", "public": true, "created_at": 1710892800000},
  {"id": "bk_d4e5f6", "name": "documents", "public": false, "created_at": 1710892900000}
]
```

## GET /bucket/:id {#bucket-get}

Retrieves detailed information about a single bucket, including storage statistics such as the total number of objects and their combined size. This is useful for monitoring bucket usage and displaying storage dashboards.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| id | string | The bucket ID. Example: `"bk_a1b2c3"`. |

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | Bucket ID. |
| name | string | Bucket name. |
| owner | string | Actor who created the bucket. |
| public | boolean | Whether the bucket allows unauthenticated reads. |
| file_size_limit | integer\|null | Maximum file size in bytes, or `null` if unrestricted. |
| allowed_mime_types | string[]\|null | Allowed MIME types, or `null` if unrestricted. |
| object_count | integer | Total number of objects currently stored in the bucket. |
| total_size | integer | Combined size of all objects in bytes. |
| created_at | integer | Creation timestamp in milliseconds. |
| updated_at | integer | Last modification timestamp in milliseconds. |

```bash
curl storage.now/bucket/bk_a1b2c3 \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/bucket/bk_a1b2c3", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const bucket = await res.json();
```

```json
{
  "id": "bk_a1b2c3",
  "name": "avatars",
  "owner": "a/my-app",
  "public": true,
  "file_size_limit": 5242880,
  "allowed_mime_types": ["image/png", "image/jpeg"],
  "object_count": 142,
  "total_size": 89456000,
  "created_at": 1710892800000,
  "updated_at": 1710903600000
}
```

## PATCH /bucket/:id {#bucket-update}

Updates the configuration of an existing bucket. Only the fields you include in the request body are modified — all other settings remain unchanged. This is useful for toggling visibility, adjusting upload constraints, or relaxing MIME type restrictions without recreating the bucket.

Changing a bucket from private to public immediately exposes all existing objects at `/object/public/:bucket/*path`. Changing from public to private immediately revokes unauthenticated access.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| id | string | The bucket ID to update. |

**Request body** — All fields are optional. Only provided fields are updated.

| Parameter | Type | Description |
|---|---|---|
| public | boolean | Change the bucket's visibility. Setting to `true` makes objects publicly readable; `false` requires authentication. |
| file_size_limit | integer\|null | Change the maximum allowed file size. Set to `null` to remove the limit entirely. Existing objects that exceed the new limit are not affected. |
| allowed_mime_types | string[]\|null | Change the set of permitted MIME types. Set to `null` to allow any content type. Existing objects of disallowed types are not removed. |

**Response** — Returns the updated bucket object.

```bash
curl -X PATCH storage.now/bucket/bk_a1b2c3 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"public": false}'
```

```javascript
const res = await fetch("https://storage.now/bucket/bk_a1b2c3", {
  method: "PATCH",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ public: false }),
});
```

```json
{
  "id": "bk_a1b2c3",
  "name": "avatars",
  "public": false,
  "updated_at": 1710903600000
}
```

## DELETE /bucket/:id {#bucket-delete}

Deletes a bucket permanently. The bucket must be empty before it can be deleted — if the bucket still contains objects, the request fails with `409 Conflict`. Use `POST /bucket/:id/empty` to remove all objects first, then delete the bucket.

Returns `{"deleted": true}` on success.

```bash
curl -X DELETE storage.now/bucket/bk_a1b2c3 \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/bucket/bk_a1b2c3", {
  method: "DELETE",
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
```

```json
{"deleted": true}
```

## POST /bucket/:id/empty {#bucket-empty}

Removes all objects from a bucket without deleting the bucket itself. This is a bulk operation — every object in the bucket is permanently deleted from both the database and the underlying object store. The bucket configuration (name, visibility, limits) remains intact.

This operation cannot be undone. Consider creating signed download URLs for any objects you need to preserve before emptying the bucket.

Returns `{"emptied": true}` on success.

```bash
curl -X POST storage.now/bucket/bk_a1b2c3/empty \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/bucket/bk_a1b2c3/empty", {
  method: "POST",
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
```

```json
{"emptied": true}
```

# Objects {#objects}

Objects are the files stored within your buckets. Each object has a path that acts as its unique identifier within the bucket — paths use `/` as a conventional separator, and folders are implicit rather than explicit entities. For example, uploading to `reports/q1.pdf` does not require creating a `reports/` folder first.

Content types are auto-detected from the file extension at upload time. You can override this by providing an explicit `Content-Type` header. Objects can be up to 100 MB per request; for larger files, use signed upload URLs which stream bytes directly to the object store.

## PUT /object/:bucket/*path {#object-upsert}

Uploads a file to the specified path in a bucket. If an object already exists at that path, it is replaced with the new content and the `updated_at` timestamp is refreshed. If no object exists, a new one is created.

The request body should contain the raw file bytes, not JSON. The server validates the upload against the bucket's `file_size_limit` and `allowed_mime_types` constraints — if either check fails, the request is rejected with `413 Too Large` or `415 Unsupported Media Type`.

Returns the object metadata with `201 Created` for new objects or `200 OK` for replacements.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | The name of the target bucket. Example: `"documents"`. |
| *path | string | The full path for the object within the bucket, including any folder-like prefixes. Example: `"reports/q1.pdf"`. |

**Headers**

| Header | Required | Description |
|---|---|---|
| Content-Type | No | MIME type of the uploaded file. If omitted, the server infers the type from the file extension — `.pdf` becomes `application/pdf`, `.png` becomes `image/png`, and so on. Provide this header when the extension is ambiguous or missing. |
| Content-Length | No | File size in bytes. Including this header allows the server to reject oversized uploads before reading the full body. |

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | Unique object identifier, prefixed with `o_`. Example: `"o_x7y8z9"`. |
| bucket | string | Name of the bucket containing the object. |
| path | string | Full object path within the bucket. |
| name | string | The filename — the last segment of the path. For `"reports/q1.pdf"` this is `"q1.pdf"`. |
| content_type | string | The resolved MIME type of the object. |
| size | integer | File size in bytes. |
| created_at | integer | Creation timestamp in milliseconds. |

```bash
curl -X PUT storage.now/object/documents/reports/q1.pdf \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/pdf" \
  -T report.pdf
```

```javascript
const res = await fetch("https://storage.now/object/documents/reports/q1.pdf", {
  method: "PUT",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/pdf",
  },
  body: fileBuffer,
});
const obj = await res.json();
```

```json
{
  "id": "o_x7y8z9",
  "bucket": "documents",
  "path": "reports/q1.pdf",
  "name": "q1.pdf",
  "content_type": "application/pdf",
  "size": 524288,
  "created_at": 1710892800000
}
```

## POST /object/:bucket/*path {#object-create}

Uploads a file, but only if no object already exists at the given path. If the path is already occupied, the request fails with `409 Conflict`. Use `PUT` instead if you want to overwrite existing objects.

Accepts the same path parameters, headers, and request body as `PUT /object/:bucket/*path`. Always returns `201 Created` on success.

## GET /object/:bucket/*path {#object-download}

Downloads an object as raw bytes. The response includes `Content-Type`, `Content-Length`, `ETag`, and `Last-Modified` headers for cache integration. Supports HTTP `Range` headers for partial downloads — useful for resumable transfers or streaming media.

Add `?download` to the URL to force the browser to download the file instead of displaying it inline. You can optionally specify a custom filename: `?download=report-final.pdf`.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | Bucket name. |
| *path | string | Object path within the bucket. |

**Query parameters**

| Parameter | Type | Description |
|---|---|---|
| download | string | When present, the response includes a `Content-Disposition: attachment` header. Optionally provide a value to set the download filename, like `?download=my-report.pdf`. |

**Response** — Raw file bytes with appropriate HTTP headers. No JSON body.

```bash
# Download a file
curl storage.now/object/documents/reports/q1.pdf \
  -H "Authorization: Bearer $TOKEN" \
  -o q1.pdf

# Force download with custom filename
curl "storage.now/object/documents/reports/q1.pdf?download=Q1-Report.pdf" \
  -H "Authorization: Bearer $TOKEN" \
  -o Q1-Report.pdf
```

```javascript
const res = await fetch("https://storage.now/object/documents/reports/q1.pdf", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const blob = await res.blob();
```

## GET /object/public/:bucket/*path {#object-download-public}

Downloads an object from a public bucket. No authentication is required — this endpoint is designed for serving static assets, avatars, and any publicly accessible content directly to browsers or CDNs.

If the bucket does not exist or is not marked as public, the request returns `404 Not Found`. Responses include a `Cache-Control: public, max-age=3600` header to encourage caching.

```bash
curl storage.now/object/public/avatars/alice.png -o alice.png
```

```javascript
// In a browser, simply use the URL directly:
const img = document.createElement("img");
img.src = "https://storage.now/object/public/avatars/alice.png";
```

## HEAD /object/:bucket/*path {#object-head}

Returns file metadata as HTTP response headers without transferring the file body. This is the most efficient way to check whether an object exists and inspect its size and type — no bytes are downloaded.

**Response headers**

| Header | Description |
|---|---|
| Content-Type | The MIME type of the object. |
| Content-Length | File size in bytes. |
| Last-Modified | Timestamp of the last modification in HTTP date format. |

```bash
curl -I storage.now/object/documents/reports/q1.pdf \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/object/documents/reports/q1.pdf", {
  method: "HEAD",
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
console.log(res.headers.get("Content-Length")); // "524288"
console.log(res.headers.get("Content-Type"));   // "application/pdf"
```

## GET /object/info/:bucket/*path {#object-info}

Returns detailed metadata about an object as a JSON response. Unlike `HEAD`, which returns metadata in HTTP headers, this endpoint provides a structured JSON body that includes the object ID, custom metadata, and precise timestamps. Use this when you need the full object record for display or processing.

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | Unique object identifier. |
| bucket | string | Bucket name. |
| path | string | Full object path. |
| name | string | Filename (last path segment). |
| content_type | string | MIME type. |
| size | integer | File size in bytes. |
| metadata | object | Custom key-value metadata attached to the object. Defaults to `{}`. |
| created_at | integer | Creation timestamp in milliseconds. |
| updated_at | integer | Last modification timestamp in milliseconds. |
| accessed_at | integer\|null | Last download timestamp in milliseconds, or `null` if never accessed. |

```bash
curl storage.now/object/info/documents/reports/q1.pdf \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/object/info/documents/reports/q1.pdf", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const info = await res.json();
```

```json
{
  "id": "o_x7y8z9",
  "bucket": "documents",
  "path": "reports/q1.pdf",
  "name": "q1.pdf",
  "content_type": "application/pdf",
  "size": 524288,
  "metadata": {},
  "created_at": 1710892800000,
  "updated_at": 1710892800000,
  "accessed_at": 1710903600000
}
```

## POST /object/list/:bucket {#object-list}

Lists objects within a bucket, with support for prefix filtering, text search, pagination, and custom sort order. This is the primary way to browse bucket contents, build file managers, or enumerate objects for batch processing.

The `prefix` parameter acts like a directory filter — setting it to `"reports/"` returns only objects whose paths start with that prefix. Combine it with `search` to find objects by filename within a specific directory.

Results are paginated with `limit` and `offset`. The default sort is ascending by name, but you can sort by `size`, `created_at`, or `updated_at` in either direction.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | The name of the bucket to list. |

**Request body** — All fields are optional.

| Parameter | Type | Default | Description |
|---|---|---|---|
| prefix | string | `""` | Filter objects whose path starts with this string. Use trailing slashes to scope to a "folder", like `"reports/"` or `"images/thumbnails/"`. |
| search | string | `""` | Free-text search term matched against object filenames. For example, `"q1"` matches both `"q1.pdf"` and `"q1-summary.md"`. |
| limit | integer | `100` | Maximum number of objects to return. Must be between 1 and 1000. |
| offset | integer | `0` | Number of objects to skip before starting to return results. Use with `limit` for pagination. |
| sort_by | object | `{"column":"name","order":"asc"}` | Controls the sort order of results. |
| sort_by.column | string | `"name"` | Which field to sort by. One of: `name`, `size`, `created_at`, `updated_at`. |
| sort_by.order | string | `"asc"` | Sort direction. Either `"asc"` for ascending or `"desc"` for descending. |

**Response** — Array of object metadata.

| Field | Type | Description |
|---|---|---|
| id | string | Object ID. |
| name | string | Filename. |
| path | string | Full path within the bucket. |
| content_type | string | MIME type. |
| size | integer | File size in bytes. |
| metadata | object | Custom metadata key-value pairs. |
| created_at | integer | Creation timestamp in milliseconds. |
| updated_at | integer | Last modification timestamp in milliseconds. |

```bash
curl -X POST storage.now/object/list/documents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prefix": "reports/",
    "search": "q1",
    "sort_by": {"column": "name", "order": "asc"},
    "limit": 50
  }'
```

```javascript
const res = await fetch("https://storage.now/object/list/documents", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    prefix: "reports/",
    limit: 50,
    sort_by: { column: "updated_at", order: "desc" },
  }),
});
const objects = await res.json();
```

```json
[
  {"id": "o_a1b2", "name": "q1.pdf", "path": "reports/q1.pdf", "size": 524288, "content_type": "application/pdf", "created_at": 1710892800000},
  {"id": "o_c3d4", "name": "q1-summary.md", "path": "reports/q1-summary.md", "size": 2048, "content_type": "text/markdown", "created_at": 1710893000000}
]
```

## DELETE /object/:bucket {#object-delete}

Deletes one or more objects from a bucket in a single request. Provide an array of paths to remove — up to 100 per request. Objects are permanently deleted from both the database and the underlying object store.

Paths that do not match any existing object are silently ignored and included in the `deleted` response array. This makes the operation idempotent — calling it twice with the same paths produces the same result.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | The name of the bucket containing the objects. |

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| paths | string[] | Yes | Array of object paths to delete. Maximum 100 paths per request. Example: `["reports/q1.pdf", "reports/q1-summary.md"]`. |

**Response**

| Field | Type | Description |
|---|---|---|
| deleted | string[] | Array of paths that were deleted. |

```bash
curl -X DELETE storage.now/object/documents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"paths": ["reports/q1.pdf", "reports/q1-summary.md"]}'
```

```javascript
const res = await fetch("https://storage.now/object/documents", {
  method: "DELETE",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    paths: ["reports/q1.pdf", "reports/q1-summary.md"],
  }),
});
const result = await res.json();
```

```json
{"deleted": ["reports/q1.pdf", "reports/q1-summary.md"]}
```

## POST /object/move {#object-move}

Moves or renames an object within a bucket. The object's content is preserved — only the path changes. If an object already exists at the destination path, the request fails with `409 Conflict`.

This is a metadata-only operation and does not copy the underlying file data, so it completes instantly regardless of file size.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| bucket | string | Yes | The name of the bucket containing the object. |
| from | string | Yes | The current path of the object. Example: `"reports/q1.pdf"`. |
| to | string | Yes | The desired new path. Example: `"archive/2025/q1.pdf"`. |

**Response**

| Field | Type | Description |
|---|---|---|
| path | string | The new path of the object after the move. |

```bash
curl -X POST storage.now/object/move \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bucket": "documents", "from": "reports/q1.pdf", "to": "archive/2025/q1.pdf"}'
```

```javascript
const res = await fetch("https://storage.now/object/move", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    bucket: "documents",
    from: "reports/q1.pdf",
    to: "archive/2025/q1.pdf",
  }),
});
```

```json
{"path": "archive/2025/q1.pdf"}
```

## POST /object/copy {#object-copy}

Creates a copy of an object, optionally across different buckets. Both the source and destination buckets must belong to the authenticated actor. A new object ID is generated for the copy, and the file content is duplicated in the object store.

If an object already exists at the destination path, the request fails with `409 Conflict`.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| from_bucket | string | Yes | Name of the source bucket. |
| from_path | string | Yes | Path of the object to copy. |
| to_bucket | string | Yes | Name of the destination bucket. Can be the same as `from_bucket` for an in-bucket copy. |
| to_path | string | Yes | Destination path for the new copy. |

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | The new object ID for the copy. |
| path | string | Destination path. |

```bash
curl -X POST storage.now/object/copy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "from_bucket": "documents",
    "from_path": "reports/q1.pdf",
    "to_bucket": "archive",
    "to_path": "2025/q1.pdf"
  }'
```

```javascript
const res = await fetch("https://storage.now/object/copy", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    from_bucket: "documents",
    from_path: "reports/q1.pdf",
    to_bucket: "archive",
    to_path: "2025/q1.pdf",
  }),
});
```

```json
{"id": "o_e5f6g7", "path": "2025/q1.pdf"}
```

# Signed URLs {#signed-urls}

Signed URLs provide time-limited access to private objects without requiring the recipient to authenticate. They are the universal sharing primitive — one mechanism that covers three common scenarios:

1. **Sharing a private file** — Generate a signed download URL and send it to anyone. They can download the file until the URL expires.
2. **Accepting uploads without exposing credentials** — Generate a signed upload URL and give it to a client. They can upload exactly one file to the specified path.
3. **Temporary public access** — Signed URLs work like pre-authenticated links with an automatic expiration, so you never need to toggle a bucket's visibility.

Signed download URLs are reusable — the same URL can be accessed multiple times until it expires. Signed upload URLs are single-use — once a file is uploaded through the URL, it is consumed and cannot be used again.

## POST /object/sign/:bucket {#sign-create}

Creates one or more signed download URLs for objects in a bucket. You can sign a single path or batch-sign up to 100 paths in one request. Each signed URL grants unauthenticated read access to the object for the specified duration.

The default expiration is 1 hour (3600 seconds). You can set a custom expiration up to 7 days (604800 seconds). After expiration, the signed URL returns `403 Forbidden`.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | Name of the bucket containing the objects. |

**Request body — single path**

| Parameter | Type | Required | Description |
|---|---|---|---|
| path | string | Yes* | Object path to sign. Provide either `path` for a single URL or `paths` for a batch — not both. |
| expires_in | integer | No | How long the URL remains valid, in seconds. Defaults to `3600` (1 hour). Maximum `604800` (7 days). |

**Request body — batch**

| Parameter | Type | Required | Description |
|---|---|---|---|
| paths | string[] | Yes* | Array of object paths to sign. Up to 100 paths per request. |
| expires_in | integer | No | Expiry duration in seconds, applied to all URLs in the batch. |

**Response — single**

| Field | Type | Description |
|---|---|---|
| signed_url | string | Relative URL for downloading the object. Example: `"/sign/tok_abc123"`. |
| token | string | The signed URL token, which can be used to construct the full URL. |
| path | string | The object path that was signed. |
| expires_at | integer | Expiration timestamp in milliseconds. |

**Response — batch** — Array of signed URL objects, one per path.

```bash
# Sign a single path
curl -X POST storage.now/object/sign/documents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path": "reports/q1.pdf", "expires_in": 3600}'
```

```javascript
const res = await fetch("https://storage.now/object/sign/documents", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    path: "reports/q1.pdf",
    expires_in: 3600,
  }),
});
const signed = await res.json();
```

```json
{
  "signed_url": "/sign/tok_abc123",
  "token": "tok_abc123",
  "path": "reports/q1.pdf",
  "expires_at": 1710896400000
}
```

```bash
# Batch sign multiple paths
curl -X POST storage.now/object/sign/documents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"paths": ["reports/q1.pdf", "reports/q2.pdf"], "expires_in": 3600}'
```

```json
[
  {"signed_url": "/sign/tok_abc", "path": "reports/q1.pdf", "token": "tok_abc", "expires_at": 1710896400000},
  {"signed_url": "/sign/tok_def", "path": "reports/q2.pdf", "token": "tok_def", "expires_at": 1710896400000}
]
```

## POST /object/upload/sign/:bucket/*path {#sign-upload-create}

Creates a signed upload URL that allows anyone to upload a single file to the specified path without authentication. This is ideal for accepting file uploads from untrusted clients — for example, letting users upload profile pictures directly from a browser without exposing your API key.

The signed upload URL expires after 1 hour by default. Once a file is uploaded through the URL, the URL is consumed and cannot be reused.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| bucket | string | Name of the target bucket. |
| *path | string | The exact path where the uploaded object will be stored. Example: `"inbox/photo.jpg"`. |

**Response**

| Field | Type | Description |
|---|---|---|
| signed_url | string | Relative URL for uploading. Example: `"/upload/sign/tok_xyz"`. |
| token | string | The signed URL token. |
| path | string | Target object path. |
| expires_at | integer | Expiration timestamp in milliseconds. |

```bash
curl -X POST storage.now/object/upload/sign/documents/inbox/upload.pdf \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch(
  "https://storage.now/object/upload/sign/documents/inbox/upload.pdf",
  {
    method: "POST",
    headers: { "Authorization": `Bearer ${TOKEN}` },
  }
);
const { signed_url, token } = await res.json();
```

```json
{
  "signed_url": "/upload/sign/tok_xyz",
  "token": "tok_xyz",
  "path": "inbox/upload.pdf",
  "expires_at": 1710896400000
}
```

## GET /sign/:token {#sign-download}

Downloads an object using a signed URL. No authentication required — the token embedded in the URL serves as the access credential. This is the endpoint your users visit when they click a shared link.

If the signed URL has expired, the response is `403 Forbidden` with an error message indicating the expiration. If the token does not exist, the response is `404 Not Found`.

**Query parameters**

| Parameter | Type | Description |
|---|---|---|
| download | string | When present, adds `Content-Disposition: attachment` to the response, prompting the browser to download the file. Provide a value to suggest a filename, like `?download=report.pdf`. |

**Response** — Raw file bytes with `Content-Type`, `Content-Length`, and `ETag` headers.

```bash
curl storage.now/sign/tok_abc123 -o report.pdf

# Force download with custom filename
curl "storage.now/sign/tok_abc123?download=Q1-Financial-Report.pdf" -o report.pdf
```

```javascript
const res = await fetch("https://storage.now/sign/tok_abc123");
const blob = await res.blob();
const url = URL.createObjectURL(blob);
```

## PUT /upload/sign/:token {#sign-upload}

Uploads a file using a signed upload URL. No authentication required — the token in the URL authorizes the upload. The file is stored at the path that was specified when the signed URL was created.

The signed URL is consumed after a successful upload and cannot be reused. If the URL has expired, the response is `403 Forbidden`. If the token does not exist or has already been used, the response is `404 Not Found`.

**Headers**

| Header | Required | Description |
|---|---|---|
| Content-Type | No | MIME type of the uploaded file. If omitted, the type is inferred from the target path extension. |

**Response** — `201 Created` with the newly created object metadata.

```bash
curl -X PUT storage.now/upload/sign/tok_xyz \
  -H "Content-Type: application/pdf" \
  -T file.pdf
```

```javascript
const res = await fetch("https://storage.now/upload/sign/tok_xyz", {
  method: "PUT",
  headers: { "Content-Type": "application/pdf" },
  body: fileBuffer,
});
const obj = await res.json();
```

```json
{
  "id": "o_h8i9j0",
  "bucket": "documents",
  "path": "inbox/upload.pdf",
  "size": 1048576,
  "content_type": "application/pdf",
  "created_at": 1710892800000
}
```

# Auth {#auth}

Identity and session management. The auth system supports two methods: Ed25519 challenge-response for machines and magic links for humans. Both methods produce Bearer tokens that grant access to all authenticated API endpoints.

Agent identities use the `a/name` format and authenticate with Ed25519 key pairs — ideal for bots, scripts, and CI/CD pipelines. Human identities use the `u/email` format and authenticate via email-based magic links — no passwords needed.

## POST /auth/register {#auth-register}

Creates a new actor identity. Every interaction with the API is associated with an actor — this is how you establish one. Agents must provide an Ed25519 public key at registration time; humans must provide an email address.

If an actor with the same identifier already exists, the request fails with `409 Conflict`. Actor identifiers are permanent and cannot be changed after creation.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| actor | string | Yes | Unique actor identifier. Use `a/name` for agents (e.g. `"a/deploy-bot"`) or `u/email` for humans (e.g. `"u/alice@example.com"`). The prefix determines the actor type. |
| type | string | No | Explicitly set the actor type: `"agent"` or `"human"`. Defaults to `"agent"` if the identifier starts with `a/`, or `"human"` if it starts with `u/`. |
| public_key | string | Yes* | Base64url-encoded Ed25519 public key. Required for agent actors. This key is used to verify challenge signatures during authentication. |
| email | string | Yes* | Email address. Required for human actors. Magic link emails are sent to this address. |

**Response**

| Field | Type | Description |
|---|---|---|
| actor | string | The registered actor identifier. |
| created_at | integer | Registration timestamp in milliseconds. |

```bash
curl -X POST storage.now/auth/register \
  -H "Content-Type: application/json" \
  -d '{"actor": "a/deploy-bot", "public_key": "<base64url-ed25519>"}'
```

```javascript
const res = await fetch("https://storage.now/auth/register", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    actor: "a/deploy-bot",
    public_key: publicKeyBase64url,
  }),
});
```

```json
{"actor": "a/deploy-bot", "created_at": 1710892800000}
```

## POST /auth/challenge {#auth-challenge}

Requests an authentication challenge for Ed25519 key-based login. The server returns a random nonce that must be signed with the actor's private key and submitted to `POST /auth/verify` within 5 minutes.

This is the first step in the two-step Ed25519 authentication flow. The challenge proves that the client possesses the private key corresponding to the public key registered for the actor.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| actor | string | Yes | The actor identifier to authenticate as. Must be a registered agent actor. |

**Response**

| Field | Type | Description |
|---|---|---|
| challenge_id | string | Unique challenge identifier, prefixed with `ch_`. Pass this to the verify step. |
| nonce | string | Random string to sign with your Ed25519 private key. |
| expires_at | integer | Challenge expiration timestamp in milliseconds. Challenges expire after 5 minutes. |

```bash
curl -X POST storage.now/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"actor": "a/deploy-bot"}'
```

```javascript
const res = await fetch("https://storage.now/auth/challenge", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ actor: "a/deploy-bot" }),
});
const { challenge_id, nonce } = await res.json();
// Sign the nonce with your Ed25519 private key
```

```json
{
  "challenge_id": "ch_k4m5n6",
  "nonce": "a1b2c3d4e5f6...",
  "expires_at": 1710893100000
}
```

## POST /auth/verify {#auth-verify}

Verifies an Ed25519 signature against a previously issued challenge and returns an access token on success. The signature must be the base64url-encoded result of signing the challenge nonce with the actor's Ed25519 private key.

Access tokens expire after 2 hours. After expiration, request a new challenge and verify again to obtain a fresh token.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| challenge_id | string | Yes | The challenge ID returned by `POST /auth/challenge`. |
| actor | string | Yes | The actor identifier. Must match the actor from the challenge request. |
| signature | string | Yes | Base64url-encoded Ed25519 signature of the challenge nonce. |

**Response**

| Field | Type | Description |
|---|---|---|
| access_token | string | Bearer token for authenticating subsequent API requests. |
| expires_at | integer | Token expiration timestamp in milliseconds. Tokens are valid for 2 hours. |

```bash
curl -X POST storage.now/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"challenge_id": "ch_k4m5n6", "actor": "a/deploy-bot", "signature": "<base64url>"}'
```

```javascript
const res = await fetch("https://storage.now/auth/verify", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    challenge_id: challenge_id,
    actor: "a/deploy-bot",
    signature: signatureBase64url,
  }),
});
const { access_token } = await res.json();
```

```json
{"access_token": "tok_session_...", "expires_at": 1710900000000}
```

## POST /auth/magic-link {#auth-magic-link}

Sends a magic link email for passwordless human authentication. The email contains a one-time link that, when clicked, creates a session and redirects to the storage browser. Magic links expire after 15 minutes.

Rate limited to 5 requests per minute per IP to prevent email abuse.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| email | string | Yes | Email address of the human actor. Must match a registered `u/email` actor. |

**Response** — `202 Accepted`.

```bash
curl -X POST storage.now/auth/magic-link \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com"}'
```

```json
{"sent": true}
```

## GET /auth/magic/:token {#auth-magic-verify}

Verifies a magic link token from an email click. On success, sets a `session` cookie with `HttpOnly` and `Secure` flags, then redirects to `/browse`. This endpoint is not typically called directly — it is the target of the link sent by `POST /auth/magic-link`.

If the token is invalid or expired, the response is `401 Unauthorized`.

```
GET /auth/magic/mtk_abc123

→ 302 Found
Set-Cookie: session=...; Path=/; HttpOnly; Secure
Location: /browse
```

## POST /auth/logout {#auth-logout}

Ends the current session by invalidating the Bearer token or session cookie. After logout, the token can no longer be used for authentication and any subsequent requests with it will receive `401 Unauthorized`.

```bash
curl -X POST storage.now/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
await fetch("https://storage.now/auth/logout", {
  method: "POST",
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
```

```json
{"ok": true}
```

# Keys {#keys}

Scoped API keys for long-lived programmatic access. Unlike session tokens that expire after 2 hours, API keys can last indefinitely or until a specified expiration date. They are ideal for CI/CD pipelines, cron jobs, and third-party integrations that need persistent access.

Each key can be scoped to specific operations (`bucket:read`, `object:write`, etc.) and restricted to a path prefix so it only has access to a subset of your objects. Key tokens are prefixed with `sk_` and can be used as Bearer tokens in the `Authorization` header, just like session tokens.

## POST /keys {#key-create}

Creates a new API key. The key token is returned exactly once in the response — store it securely, because it cannot be retrieved again. If you lose the token, you must revoke the key and create a new one.

**Request body**

| Parameter | Type | Required | Description |
|---|---|---|---|
| name | string | Yes | A human-readable label for the key. Use something descriptive like `"ci-deploy"`, `"backup-cronjob"`, or `"frontend-uploads"`. |
| scopes | string | No | Comma-separated list of permission scopes. Defaults to `"*"` (all permissions). Available scopes: `bucket:read`, `bucket:write`, `object:read`, `object:write`. |
| path_prefix | string | No | Restrict the key to objects whose path starts with this prefix. For example, `"uploads/"` limits the key to only interact with objects under the `uploads/` directory. Defaults to `""` (all paths). |
| expires_at | integer | No | Expiration timestamp in milliseconds. After this time, the key is automatically invalidated. Omit or set to `null` for a key that never expires. |

**Response**

| Field | Type | Description |
|---|---|---|
| id | string | Unique key identifier, prefixed with `ak_`. |
| name | string | The key name as provided. |
| token | string | The API key token, prefixed with `sk_`. **Shown once — store it securely.** |
| scopes | string | The granted scopes. |
| created_at | integer | Creation timestamp in milliseconds. |

```bash
curl -X POST storage.now/keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-deploy", "scopes": "object:read,object:write"}'
```

```javascript
const res = await fetch("https://storage.now/keys", {
  method: "POST",
  headers: {
    "Authorization": `Bearer ${TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    name: "ci-deploy",
    scopes: "object:read,object:write",
  }),
});
const { token } = await res.json();
// Store token securely — it cannot be retrieved again
```

```json
{
  "id": "ak_abc",
  "name": "ci-deploy",
  "token": "sk_live_a1b2c3d4e5f6...",
  "scopes": "object:read,object:write",
  "created_at": 1710892800000
}
```

Use the token as: `Authorization: Bearer sk_live_a1b2c3d4e5f6...`

## GET /keys {#key-list}

Returns a list of all API keys belonging to the authenticated actor. For security, key tokens are never included in the response — only metadata such as the key name, scopes, and creation date. Use this to audit your active keys or find the ID of a key you want to revoke.

**Response** — Array of key objects (without tokens).

```bash
curl storage.now/keys \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/keys", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const keys = await res.json();
```

```json
[
  {"id": "ak_abc", "name": "ci-deploy", "scopes": "object:read,object:write", "created_at": 1710892800000},
  {"id": "ak_def", "name": "backup", "scopes": "*", "created_at": 1710893000000}
]
```

## DELETE /keys/:id {#key-revoke}

Revokes an API key immediately. Any requests using the revoked key's token will receive `401 Unauthorized` from the moment of revocation. This operation is permanent — a revoked key cannot be restored.

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| id | string | The key ID to revoke. Example: `"ak_abc"`. |

```bash
curl -X DELETE storage.now/keys/ak_abc \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
await fetch("https://storage.now/keys/ak_abc", {
  method: "DELETE",
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
```

```json
{"deleted": true}
```

# Audit {#audit}

Complete audit trail of every operation performed through the API. Every bucket creation, object upload, key rotation, and authentication event is recorded with the actor, resource, client IP, and timestamp. Use the audit log for security monitoring, compliance reporting, or debugging unexpected behavior.

## GET /audit {#audit-read}

Queries the audit log with optional filtering by action type and cursor-based pagination. Events are returned in reverse chronological order — most recent events first.

Use the `before` parameter for pagination: take the `ts` value of the last event in the current page and pass it as `before` in the next request to fetch older events.

**Query parameters**

| Parameter | Type | Default | Description |
|---|---|---|---|
| action | string | — | Filter events by action type. For example, `"object.upload"` returns only upload events, `"bucket.create"` returns only bucket creation events. Omit to return all event types. |
| limit | integer | `50` | Maximum number of events to return per page. Must be between 1 and 200. |
| before | integer | — | Return only events with a timestamp strictly less than this value (Unix milliseconds). Use for pagination by passing the `ts` of the last event from the previous page. |

**Response**

| Field | Type | Description |
|---|---|---|
| items | array | Array of audit event objects, ordered by timestamp descending. |
| items[].action | string | The event action. See the list of event actions below. |
| items[].resource | string | The affected resource path or identifier. |
| items[].detail | object\|null | Additional event-specific data, such as the new bucket configuration or the number of deleted objects. |
| items[].ip | string | Client IP address that triggered the event. |
| items[].ts | integer | Event timestamp in milliseconds. |

**Event actions:** `bucket.create`, `bucket.update`, `bucket.delete`, `bucket.empty`, `object.upload`, `object.delete`, `object.move`, `object.copy`, `signed_url.create`, `signed_url.batch`, `signed_url.upload_create`, `auth.login`, `auth.logout`, `key.create`, `key.revoke`.

```bash
# Get the 10 most recent upload events
curl "storage.now/audit?action=object.upload&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Paginate to the next page
curl "storage.now/audit?action=object.upload&limit=10&before=1710892800000" \
  -H "Authorization: Bearer $TOKEN"
```

```javascript
const res = await fetch("https://storage.now/audit?action=object.upload&limit=10", {
  headers: { "Authorization": `Bearer ${TOKEN}` },
});
const { items } = await res.json();
```

```json
{
  "items": [
    {"action": "object.upload", "resource": "documents/reports/q1.pdf", "detail": {"size": 524288}, "ip": "203.0.113.42", "ts": 1710892800000},
    {"action": "object.upload", "resource": "avatars/alice.png", "detail": {"size": 45678}, "ip": "203.0.113.42", "ts": 1710892700000}
  ]
}
```

# All Endpoints {#endpoints}

| # | Method | Path | Group | Description |
|---|---|---|---|---|
| 1 | POST | /bucket | Buckets | Create a new bucket |
| 2 | GET | /bucket | Buckets | List all buckets |
| 3 | GET | /bucket/:id | Buckets | Get bucket details and stats |
| 4 | PATCH | /bucket/:id | Buckets | Update bucket configuration |
| 5 | DELETE | /bucket/:id | Buckets | Delete an empty bucket |
| 6 | POST | /bucket/:id/empty | Buckets | Remove all objects from a bucket |
| 7 | POST | /object/:bucket/*path | Objects | Upload (error if exists) |
| 8 | PUT | /object/:bucket/*path | Objects | Upload or replace |
| 9 | GET | /object/:bucket/*path | Objects | Download (authenticated) |
| 10 | GET | /object/public/:bucket/*path | Objects | Download from public bucket |
| 11 | HEAD | /object/:bucket/*path | Objects | Get metadata as headers |
| 12 | GET | /object/info/:bucket/*path | Objects | Get metadata as JSON |
| 13 | POST | /object/list/:bucket | Objects | List and search objects |
| 14 | DELETE | /object/:bucket | Objects | Batch delete objects |
| 15 | POST | /object/move | Objects | Move or rename an object |
| 16 | POST | /object/copy | Objects | Copy an object |
| 17 | POST | /object/sign/:bucket | Signed URLs | Create signed download URL(s) |
| 18 | POST | /object/upload/sign/:bucket/*path | Signed URLs | Create signed upload URL |
| 19 | GET | /sign/:token | Signed URLs | Download via signed URL |
| 20 | PUT | /upload/sign/:token | Signed URLs | Upload via signed URL |
| 21 | POST | /auth/register | Auth | Register a new actor |
| 22 | POST | /auth/challenge | Auth | Request Ed25519 challenge |
| 23 | POST | /auth/verify | Auth | Verify signature, get token |
| 24 | POST | /auth/magic-link | Auth | Send magic link email |
| 25 | GET | /auth/magic/:token | Auth | Verify magic link |
| 26 | POST | /auth/logout | Auth | End current session |
| 27 | POST | /keys | Keys | Create a scoped API key |
| 28 | GET | /keys | Keys | List API keys |
| 29 | DELETE | /keys/:id | Keys | Revoke an API key |
| 30 | GET | /audit | Audit | Query the audit log |

# Error Codes {#errors}

All errors return a consistent JSON structure with a machine-readable `code` and a human-readable `message`. The HTTP status code always matches the error category.

```json
{
  "error": {
    "code": "not_found",
    "message": "Object 'reports/q1.pdf' does not exist in bucket 'documents'."
  }
}
```

| Code | HTTP | When it occurs |
|---|---|---|
| invalid_request | 400 | The request body is malformed, a required field is missing, or a parameter value is out of range. |
| unauthorized | 401 | No Bearer token was provided, the token has expired, or the token is invalid. |
| forbidden | 403 | The token is valid but does not have sufficient scope or permission for the requested operation. Also returned for expired signed URLs. |
| not_found | 404 | The requested bucket, object, or resource does not exist. For public endpoints, also returned when a bucket is not public. |
| conflict | 409 | A resource with the same identifier already exists. For example, creating a bucket with a name that is already taken, or using `POST` to upload to a path that is already occupied. |
| too_large | 413 | The uploaded file exceeds the bucket's `file_size_limit` or the global 100 MB per-request limit. |
| unsupported_type | 415 | The uploaded file's MIME type is not in the bucket's `allowed_mime_types` list. |
| rate_limited | 429 | Too many requests in the current window. Check the `Retry-After` response header for the number of seconds to wait. |
| internal | 500 | An unexpected server error occurred. If this persists, please report it. |

# Extensions {#extensions}

Extensions provide optional capabilities beyond the core storage API. They are not required for basic file storage and may not be available on all deployments.

| Extension | Endpoints | Purpose |
|---|---|---|
| MCP | 2 | Model Context Protocol integration for AI agents like Claude and ChatGPT. Exposes storage operations as MCP tools. |
| OAuth 2.0 | 7 | Third-party application authorization using RFC 6749 with PKCE. Enables building apps that access storage on behalf of users. |
