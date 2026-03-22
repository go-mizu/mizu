# Storage Developer Guide

> **Base URL:** `https://storage.liteio.dev`

Storage is a file storage API. Upload a file in one HTTP request, serve it globally with zero egress fees. No SDK required — every endpoint is plain HTTP with JSON.

## Quickstart

Three steps from zero to your first uploaded file.

### 1. Get a token

Register with your Ed25519 public key or request a magic link via email. Both return a bearer token.

```bash
# Ed25519 key auth
curl -X POST https://storage.liteio.dev/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"actor":"your-username","public_key":"base64-ed25519-pubkey"}'

# Then verify the signature
curl -X POST https://storage.liteio.dev/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"actor":"your-username","signature":"base64-signature"}'
```

### 2. Upload a file

Initiate the upload to get a presigned URL, PUT the file bytes there, then confirm.

```bash
# Step 1: Initiate
curl -X POST https://storage.liteio.dev/files/uploads \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"assets/logo.svg","content_type":"image/svg+xml"}'

# Step 2: Upload to the presigned URL returned in step 1
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: image/svg+xml" \
  --data-binary @logo.svg

# Step 3: Confirm
curl -X POST https://storage.liteio.dev/files/uploads/complete \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"assets/logo.svg"}'
```

### 3. Share it

Generate a time-limited public link. Anyone with the link can download — no auth required.

```bash
curl -X POST https://storage.liteio.dev/files/share \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"assets/logo.svg","ttl":86400}'

# Response: { "url": "https://storage.liteio.dev/s/k7f2m", "expires_at": 1711065600000 }
```

## API Surface

Everything lives under `/files`. Standard HTTP methods. Paths are your filesystem.

### Store

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/files/uploads` | Initiate a file upload (returns presigned PUT URL) |
| `POST` | `/files/uploads/complete` | Confirm upload after PUT completes |
| `GET` | `/files/{path}` | Download a file (302 redirect to presigned URL) |
| `HEAD` | `/files/{path}` | Get file metadata without downloading |
| `DELETE` | `/files/{path}` | Delete a file or folder |

### Organize

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/files?prefix={folder}` | List files in a folder |
| `GET` | `/files/search?q={query}` | Search files by name |
| `POST` | `/files/move` | Rename or move a file |
| `GET` | `/files/stats` | Get total file count and bytes used |

### Share

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/files/share` | Create a temporary public link (default: 1h, max: 7d) |
| `GET` | `/s/{token}` | Access a shared file (no auth required) |

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/challenge` | Request an Ed25519 challenge nonce |
| `POST` | `/auth/verify` | Verify signature and get session token |
| `POST` | `/auth/magic` | Send a magic link to an email address |
| `POST` | `/auth/register` | Register a new account with Ed25519 public key |
| `POST` | `/auth/logout` | Invalidate current session |

### API Keys

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/keys` | Create a scoped API key |
| `GET` | `/auth/keys` | List all API keys |
| `DELETE` | `/auth/keys/{id}` | Revoke an API key |

## Code Examples

### curl

```bash
# Upload a file
curl -X POST https://storage.liteio.dev/files/uploads \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"data/config.json","content_type":"application/json"}'

# Download (follows 302 redirect)
curl -L https://storage.liteio.dev/files/data/config.json \
  -H "Authorization: Bearer $TOKEN"
```

### JavaScript

```javascript
// Initiate upload, then PUT to the presigned URL
const { upload_url } = await fetch("https://storage.liteio.dev/files/uploads", {
  method: "POST",
  headers: { Authorization: `Bearer ${TOKEN}` },
  body: JSON.stringify({ path: "data/config.json" }),
}).then(r => r.json());

await fetch(upload_url, { method: "PUT", body: file });
```

### Python

```python
import requests

res = requests.post(
    "https://storage.liteio.dev/files/uploads",
    headers={"Authorization": f"Bearer {TOKEN}"},
    json={"path": "data/config.json"},
)
upload_url = res.json()["upload_url"]

requests.put(upload_url, data=file_bytes)
```

### Go

```go
body := strings.NewReader(`{"path":"data/config.json"}`)
req, _ := http.NewRequest("POST",
    "https://storage.liteio.dev/files/uploads", body)
req.Header.Set("Authorization", "Bearer "+token)

resp, _ := http.DefaultClient.Do(req)
// Parse upload_url from JSON response, then PUT file there
```

## MCP Protocol

Storage has 8 MCP tools built in. Connect once, then read, write, search, and share files from any MCP client.

### Available Tools

| Tool | Description |
|------|-------------|
| `storage_read` | Read a file's contents |
| `storage_write` | Write or overwrite a file |
| `storage_list` | List files in a folder |
| `storage_search` | Search files by name |
| `storage_share` | Create a temporary public link |
| `storage_move` | Move or rename a file |
| `storage_delete` | Delete a file |
| `storage_stats` | Get storage usage statistics |

### Connect Claude

1. Open **Settings > Integrations**
2. Click **Add custom connector**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Authorize with your email

### Connect ChatGPT

1. Open **Settings > Connected apps**
2. Click **Add app > Add by URL**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Sign in with your email

### Connect Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "storage": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://storage.liteio.dev/mcp"]
    }
  }
}
```

## Architecture

```
Your App ──HTTPS──> API Server ──Presigned URL──> Object Store (R2)
                    (auth + sign)                 (durable, zero egress)
```

File bytes never touch the API server. Your app sends an authenticated request, the API returns a presigned URL in under 50ms, and your app uploads or downloads directly to the object store.

- **Inline auth** — Token verification happens before presigning. Sub-millisecond overhead.
- **Direct transfers** — Clients upload and download directly via presigned URLs. No proxy bandwidth.
- **Metadata layer** — File index, sessions, and shares stored in SQLite. Fast reads, strong consistency.

## Security

- **Ed25519 auth** — Public key challenge-response. No shared secrets.
- **Scoped API keys** — Path-prefix restrictions. `sk_*` format, 90-day TTL.
- **Signed share links** — Time-limited URLs, 60 seconds to 7 days, auto-expire.
- **OAuth 2.0 + PKCE** — Standard flow for third-party apps. Dynamic client registration.
- **Rate limiting** — Per-endpoint sliding window. Auth: 10/min. Uploads: 100/min.
- **Audit logging** — Every action logged with actor, resource, and timestamp.

## Links

- [API Reference](https://storage.liteio.dev/api)
- [CLI Documentation](https://storage.liteio.dev/cli)
- [Pricing](https://storage.liteio.dev/pricing)
- [Developer Guide (human view)](https://storage.liteio.dev/developers)
