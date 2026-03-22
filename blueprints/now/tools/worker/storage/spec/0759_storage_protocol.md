# 0759: Storage API Protocol Redesign

## Motivation

The current REST API grew organically and has several naming inconsistencies:

| Current Route | What it does | Problem |
|---|---|---|
| `PUT /f/*` | Upload file | Now returns 410 — dead endpoint |
| `GET /f/*` | Download file | Now returns 410 — dead endpoint |
| `DELETE /f/*` | Delete file | `/f/` is cryptic, not self-describing |
| `HEAD /f/*` | File metadata | Same |
| `GET /ls/*` | List directory | Unix command, not a REST resource |
| `GET /find?q=` | Search files | Verb-as-noun |
| `POST /mv` | Move/rename | Unix command |
| `GET /stat` | Storage stats | Unix command |
| `POST /share` | Create share link | OK, but inconsistent namespace |
| `GET /s/:token` | Access shared file | Extremely terse |
| `POST /presign/upload` | Get upload URL | Presign is an implementation detail, not a user concept |
| `POST /presign/complete` | Confirm upload | Same |
| `GET /presign/read/*` | Get download URL | Same |
| `POST /presign/multipart/create` | Initiate multipart | Same |
| `POST /presign/multipart/complete` | Complete multipart | Same |
| `POST /presign/multipart/abort` | Abort multipart | Same |

Meanwhile, the MCP tools use clean, predictable names:

| MCP Tool | REST Equivalent |
|---|---|
| `storage_list` | `GET /ls/*` |
| `storage_read` | `GET /presign/read/*` |
| `storage_write` | `POST /presign/upload` + `POST /presign/complete` |
| `storage_delete` | `DELETE /f/*` |
| `storage_search` | `GET /find?q=` |
| `storage_move` | `POST /mv` |
| `storage_share` | `POST /share` |
| `storage_stats` | `GET /stat` |

The MCP verbs are better. They use a consistent `storage_<verb>` pattern with clear, self-describing names. The REST API should follow their lead.

---

## Industry Comparison

### How others structure paths

| Service | Pattern | Example |
|---|---|---|
| **AWS S3** | `/{key}` (flat, query params for operations) | `GET /bucket/docs/report.pdf` |
| **Google Cloud Storage** | `/storage/v1/b/{bucket}/o/{object}` | `GET /storage/v1/b/my-bucket/o/report.pdf` |
| **Supabase Storage** | `/object/{bucket}/{path}` | `GET /object/files/docs/report.pdf` |
| **Vercel Blob** | `/{pathname}` (flat) | `PUT /docs/report.pdf` |
| **Firebase** | `/v0/b/{bucket}/o/{object}` | `GET /v0/b/default/o/report.pdf` |

### Key patterns from industry

1. **Resource-oriented paths** — S3 uses `/{key}`, Supabase uses `/object/`, GCS uses `/o/`. Nobody uses `/f/` or `/ls/`.
2. **Presigning is invisible** — S3 and GCS don't have `/presign/` routes. The presigned URL *is* the object URL with signing query params appended. The client either has a valid signature or doesn't. Supabase is the exception with `/object/sign/`.
3. **Operations on resources, not verbs** — Move is `POST /object/move` (Supabase) or `PUT` with `x-amz-copy-source` (S3). Not `POST /mv`.
4. **Listing is a query on the collection** — `GET /objects?prefix=docs/` (S3), `POST /object/list/{bucket}` (Supabase). Not a separate `/ls/` route.
5. **Metadata vs content** — S3 uses `HEAD` vs `GET` on the same URL. GCS uses `?alt=media`. Supabase uses `/object/info/`.

### What we should learn

- Use `/objects/` (or `/files/`) as a clear resource namespace — not `/f/`
- Group related operations under the same resource path
- Don't expose implementation details (`presign`) in the URL
- Use standard REST conventions: `GET` for read, `PUT` for write, `DELETE` for delete

---

## Design Decisions

### Decision 1: Resource namespace — `/objects/*` vs `/files/*`

**Option A: `/objects/*`** — Matches S3/GCS terminology. Technically correct (we store arbitrary objects, not just files).

**Option B: `/files/*`** — More intuitive for end users. Matches how everyone thinks about it. Supabase uses `/object/` (singular) but the mental model is "files."

**Recommendation: `/files/*`** — Our product is called "Storage" and targets developers who think in terms of files and folders, not objects and prefixes. The MCP tools already say "files and folders" in their descriptions.

### Decision 2: Upload model — presigned flow vs direct PUT

Now that we've removed the data proxy, uploads are a 3-step flow:

```
1. POST /presign/upload    → get signed URL
2. PUT  <signed-url>       → upload to R2
3. POST /presign/complete  → confirm + index
```

**Option A: Keep 3-step, rename routes** — `POST /files/uploads` to initiate, `POST /files/uploads/complete` to confirm. Clearer naming but same flow.

**Option B: Direct PUT with presigned response** — `PUT /files/docs/report.pdf` returns `{ upload_url, ... }` (HTTP 202 Accepted). Client uploads to the URL, then `POST /files/docs/report.pdf/confirm`. Feels more RESTful — PUT on the resource itself.

**Option C: Keep POST initiation, use resource path for confirm** — `POST /files/upload` to initiate (returns signed URL), `POST /files/{path}/complete` to confirm. The confirm knows which file by path.

**Recommendation: Option A** — The 3-step flow is inherent to presigned uploads (auth check → sign → confirm). Trying to hide it behind PUT semantics creates confusion ("why does PUT return a URL instead of uploading?"). Supabase has `/object/sign/` — we can do better with `/files/uploads`. The initiate/confirm pattern is explicit and honest.

### Decision 3: Download model — redirect vs URL response

**Option A: Return JSON with URL** — Current behavior. `GET /files/read/docs/report.pdf` → `{ url, size, type }`. Client fetches the URL separately.

**Option B: 302 redirect** — `GET /files/docs/report.pdf` → 302 to presigned R2 URL. Browser-native, works with `<img src>`, `<a href>`, curl follows redirects automatically.

**Option C: Both** — `GET /files/{path}` redirects by default. `GET /files/{path}?meta=true` returns JSON metadata. Or: `HEAD /files/{path}` for metadata, `GET /files/{path}` for redirect.

**Recommendation: Option C** — `GET /files/{path}` does a 302 redirect to R2 (zero proxy, works in browsers natively). `HEAD /files/{path}` returns metadata headers (size, type, etag). This matches S3 exactly: GET downloads, HEAD returns metadata. API clients that need the raw URL can read the `Location` header without following the redirect.

This is a significant improvement: it eliminates the awkward `/presign/read/*` pattern and makes downloads work with a single URL — `<img src="/files/photos/cat.jpg">` just works.

### Decision 4: Listing and search — combined or separate

**Current:** `GET /ls/*` for listing, `GET /find?q=` for search.

**Option A: Separate resources** — `GET /files?prefix=docs/` for listing, `GET /files/search?q=todo` for search.

**Option B: Combined** — `GET /files?prefix=docs/` lists, `GET /files?q=todo` searches. Prefix takes precedence. Single endpoint, different query params.

**Option C: Nested under /files** — `GET /files/` (root listing), `GET /files/docs/` (trailing slash = list), `GET /files/docs/report.pdf` (no trailing slash = download). Search via `GET /files?q=`.

**Recommendation: Option A** — Combining list and search overloads one endpoint with different semantics (prefix = exact hierarchy, q = fuzzy match). Keeping them separate is clearer. `GET /files` with query params for listing, `GET /files/search` for search.

Wait — if `GET /files/{path}` is the download redirect (Decision 3), then `GET /files?prefix=docs/` for listing works naturally: query-param-only requests return the collection, path requests return the specific resource.

### Decision 5: Move/rename

**Current:** `POST /mv` with `{ from, to }` body.

**Option A: `POST /files/move`** — Explicit action endpoint under the files namespace.

**Option B: `PATCH /files/{path}`** — `{ move_to: "new/path" }`. RESTful — PATCH the resource's location. But conflates metadata update with relocation.

**Option C: `POST /files/{from_path}/move`** — Target in body: `{ to: "new/path" }`. Source in URL, destination in body.

**Recommendation: Option A** — `POST /files/move` with `{ from, to }`. Simple, explicit, matches the MCP `storage_move` semantics. Supabase uses the same pattern (`POST /object/move`).

### Decision 6: Sharing

**Current:** `POST /share` creates link, `GET /s/:token` accesses it.

The sharing endpoints are fine conceptually but live outside the `/files` namespace.

**Recommendation:** Move to `POST /files/share` (create link) and keep `GET /s/:token` (short public URLs should stay short). The create endpoint belongs with file operations; the access endpoint is a separate concern (public, no auth, short URL).

### Decision 7: Stats

**Current:** `GET /stat`

**Recommendation:** `GET /files/stats` — under the files namespace, pluralized (industry standard: `/stats`, `/metrics`).

### Decision 8: Multipart uploads

**Current:** `/presign/multipart/create`, `/presign/multipart/complete`, `/presign/multipart/abort`

**Recommendation:** Nest under uploads: `POST /files/uploads/multipart`, `POST /files/uploads/multipart/complete`, `POST /files/uploads/multipart/abort`. The multipart flow is a specialization of the upload flow. Same namespace, clear hierarchy.

### Decision 9: Auth routes — keep or move?

**Current:** `/auth/register`, `/auth/challenge`, `/auth/verify`, `/auth/keys`, `/auth/magic-link`, `/auth/magic/:token`, `/auth/logout`

These are not file operations. They should stay separate.

**Recommendation:** Keep `/auth/*` as-is. These are fine — auth is a separate domain. No change needed.

### Decision 10: API versioning

**Option A: URL prefix** — `/v1/files/*`. Allows breaking changes behind `/v2/`.

**Option B: No prefix** — Current behavior. Simpler URLs.

**Option C: Header-based** — `API-Version: 2024-03-20`. Stripe-style date versioning.

**Recommendation: Option B (no prefix) for now** — We're a young API with few consumers. Adding `/v1/` now creates overhead with no benefit. We can add versioning later if we need breaking changes, using the header approach (date-based, Stripe-style) since it doesn't pollute URLs.

---

## Proposed API v2

### Files

| Method | Path | Description | Body / Params |
|---|---|---|---|
| `GET` | `/files` | List files/folders | `?prefix=docs/&limit=200` |
| `GET` | `/files/{path}` | Download file (302 → R2) | Follows redirect; `Range` header supported |
| `HEAD` | `/files/{path}` | File metadata | Returns `Content-Length`, `Content-Type`, `ETag`, `Last-Modified` headers |
| `DELETE` | `/files/{path}` | Delete file or folder | Trailing `/` = recursive folder delete |
| `GET` | `/files/search` | Search by name | `?q=todo&limit=50` |
| `GET` | `/files/stats` | Storage usage | Returns `{ files, bytes }` |
| `POST` | `/files/move` | Move / rename | `{ "from": "a.txt", "to": "b.txt" }` |
| `POST` | `/files/share` | Create share link | `{ "path": "a.txt", "expires_in": 86400 }` |

### Uploads

| Method | Path | Description | Body |
|---|---|---|---|
| `POST` | `/files/uploads` | Initiate upload → presigned URL | `{ "path": "docs/a.pdf", "content_type": "..." }` |
| `POST` | `/files/uploads/complete` | Confirm upload → index in D1 | `{ "path": "docs/a.pdf" }` |
| `POST` | `/files/uploads/multipart` | Initiate multipart upload | `{ "path": "...", "part_count": 10 }` |
| `POST` | `/files/uploads/multipart/complete` | Complete multipart | `{ "path": "...", "upload_id": "...", "parts": [...] }` |
| `POST` | `/files/uploads/multipart/abort` | Abort multipart | `{ "path": "...", "upload_id": "..." }` |

### Sharing (public)

| Method | Path | Description |
|---|---|---|
| `GET` | `/s/{token}` | Access shared file (302 → R2 presigned URL) |

### Auth (unchanged)

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/register` | Register actor |
| `POST` | `/auth/challenge` | Request challenge |
| `POST` | `/auth/verify` | Verify signature → session token |
| `POST` | `/auth/keys` | Create API key |
| `GET` | `/auth/keys` | List API keys |
| `DELETE` | `/auth/keys/{id}` | Delete API key |
| `POST` | `/auth/magic-link` | Send magic link email |
| `GET` | `/auth/magic/{token}` | Verify magic link |
| `POST` | `/auth/logout` | Logout |

### MCP (unchanged)

| Method | Path | Description |
|---|---|---|
| `GET` | `/mcp` | SSE stream |
| `POST` | `/mcp` | JSON-RPC 2.0 |
| `DELETE` | `/mcp` | Terminate session |

### OAuth (unchanged)

| Method | Path | Description |
|---|---|---|
| `GET` | `/.well-known/oauth-protected-resource` | Resource metadata |
| `GET` | `/.well-known/oauth-authorization-server` | AS metadata |
| `POST` | `/oauth/register` | Dynamic client registration |
| `GET` | `/oauth/authorize` | Authorization endpoint |
| `POST` | `/oauth/token` | Token exchange |

---

## Migration from Current to Proposed

### Route Mapping (old → new)

| Old Route | New Route | Migration |
|---|---|---|
| `DELETE /f/{path}` | `DELETE /files/{path}` | Redirect or alias |
| `HEAD /f/{path}` | `HEAD /files/{path}` | Redirect or alias |
| `GET /ls/*` | `GET /files?prefix=` | Redirect; change query param format |
| `GET /find?q=` | `GET /files/search?q=` | Redirect or alias |
| `POST /mv` | `POST /files/move` | Redirect or alias |
| `GET /stat` | `GET /files/stats` | Redirect or alias |
| `POST /share` | `POST /files/share` | Redirect or alias |
| `POST /presign/upload` | `POST /files/uploads` | Redirect or alias |
| `POST /presign/complete` | `POST /files/uploads/complete` | Redirect or alias |
| `GET /presign/read/*` | `GET /files/{path}` (302) | Breaking change — response format changes from JSON to redirect |
| `POST /presign/multipart/create` | `POST /files/uploads/multipart` | Redirect or alias |
| `POST /presign/multipart/complete` | `POST /files/uploads/multipart/complete` | Redirect or alias |
| `POST /presign/multipart/abort` | `POST /files/uploads/multipart/abort` | Redirect or alias |

### Migration strategy

**Phase 1: Add new routes alongside old ones.** Both `/f/*` and `/files/*` work. Old routes log deprecation warnings (`X-Deprecated: true` header).

**Phase 2: Update clients.** CLI, browser, MCP tools, docs, examples — all point to new routes.

**Phase 3: Remove old routes.** After a reasonable period (e.g. 30 days), old routes return `410 Gone` with migration instructions.

Since we already returned 410 for `PUT/GET /f/*`, users are accustomed to endpoint deprecation. The same pattern applies here.

---

## MCP Tool Changes

MCP tool names (`storage_list`, `storage_read`, etc.) are already well-named. Their internal implementation would switch to the new routes, but since they call internal functions (not HTTP), only the route registration changes. The tool interface stays identical.

One potential improvement: the `storage_write` tool currently proxies file data through the Worker (for text content and URL downloads). For text content, this is unavoidable (the MCP client sends text, the server writes it). For URL downloads, the server fetches the URL — this is also unavoidable since the MCP client can't do arbitrary fetches. So MCP write is the one place where data *does* flow through the Worker, and that's architecturally correct.

---

## Impact on Developers Page and Docs

The `/developers` page currently shows the old routes (`PUT /f/*`, `GET /f/*`, `GET /ls/*`, etc.) including dead endpoints. This needs a complete refresh to show:

1. The new `/files/*` routes
2. The presigned upload flow (3-step: initiate → upload → confirm)
3. The download redirect (just `GET /files/{path}`)
4. Updated curl/JS/Python/Go examples
5. The architecture diagram now showing client → R2 direct data flow

---

## Implementation Tasks

1. **Create new route file `src/routes/files-v2.ts`** — Implement all `/files/*` routes. Internally reuses existing handler logic from `files.ts`, `ls.ts`, `find.ts`, `mv.ts`, `stat.ts`, `share.ts`, `presign.ts`.

2. **Add download redirect** — `GET /files/{path}` does auth → HEAD check → 302 to presigned R2 URL. This is the biggest new behavior.

3. **Wire up listing** — `GET /files?prefix=` calls the same D1 query as current `ls.ts`.

4. **Add deprecation layer** — Old routes forward to new handlers with `X-Deprecated: true` response header.

5. **Update CLI** — Change all endpoint paths from `/presign/*`, `/ls/*`, `/find`, `/mv`, `/stat`, `/share` to `/files/*`.

6. **Update browse.js** — Change all fetch URLs to new paths. Download links can use `/files/{path}` directly (browser follows 302).

7. **Update MCP handlers** — Internal function calls, not route changes. Update if handler signatures change.

8. **Update developers page** — New examples, new route table, new architecture diagram showing direct-to-R2 data flow.

9. **Update OpenAPI spec** — Route annotations for new paths.

10. **Remove old routes** — After migration period, replace with 410 Gone.

---

## Open Questions

1. **Should `GET /files/docs/` (trailing slash) be a list operation?** This would let `GET /files/{path}` serve both download and list based on trailing slash. Elegant but potentially confusing — a missing slash gives you a 302 redirect, adding one gives you a JSON listing.

2. **Should `POST /files/uploads` accept the file body directly for small files?** This would make simple uploads a single request instead of three. The server writes to R2 via binding (no presign needed for small payloads). Threshold could be 10 MB. Trade-off: simpler DX for small files vs. one more code path.

3. **Should share links be under `/shares/` instead of `/files/share`?** If sharing becomes a first-class resource (list shares, revoke shares, update expiry), a dedicated `/shares` namespace might be cleaner: `POST /shares` (create), `GET /shares` (list my shares), `DELETE /shares/{token}` (revoke).

4. **Response format for `GET /files?prefix=`** — Currently `GET /ls/` returns `{ entries: [...] }`. Should we add pagination cursors? The current implementation uses `limit` but no cursor. For large directories, cursor-based pagination would be more robust.
