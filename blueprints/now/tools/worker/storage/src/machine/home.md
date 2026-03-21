# Storage

> Store, share, and find files. Connected to Claude and ChatGPT via MCP.

Storage is an MCP-native file storage service. Upload files from your browser, CLI, API, or AI assistant. Share with a link. Search across everything.

## Connect Your AI

### Claude.ai

1. Open **Settings > Integrations**
2. Click **Add custom connector**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Click Add, verify your email — done

### ChatGPT

1. Open **Settings > Connected apps**
2. Click **Add app > Add by URL**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Sign in with email — done

### Claude Desktop

Add to your `claude_desktop_config.json` (Settings > Developer > Edit Config):

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

Restart Claude Desktop after saving.

## MCP Tools

| Tool | Description |
|------|-------------|
| `storage_read` | Read a file's contents |
| `storage_write` | Create or overwrite a file |
| `storage_list` | List files in a folder |
| `storage_search` | Search files by name |
| `storage_share` | Create a temporary public link |
| `storage_move` | Move or rename a file |
| `storage_delete` | Delete a file |
| `storage_stats` | Show storage usage |

## API

Base URL: `https://storage.liteio.dev`

### File Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/files/uploads` | Initiate upload (returns presigned PUT URL) |
| `POST` | `/files/uploads/complete` | Confirm upload after PUT completes |
| `GET` | `/files/{path}` | Download a file (302 to presigned URL) |
| `HEAD` | `/files/{path}` | Get file metadata |
| `DELETE` | `/files/{path}` | Delete a file or folder |
| `GET` | `/files?prefix={folder}` | List files in a folder |
| `GET` | `/files/search?q={query}` | Search files by name |
| `POST` | `/files/move` | Move or rename a file |
| `POST` | `/files/share` | Create a temporary public link |
| `GET` | `/files/stats` | Storage usage (count, bytes) |

### Authentication

All endpoints require `Authorization: Bearer <token>` except shared links (`/s/{token}`).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/challenge` | Request Ed25519 challenge nonce |
| `POST` | `/auth/verify` | Verify signature, get session token |
| `POST` | `/auth/magic` | Send magic link to email |
| `POST` | `/auth/keys` | Create a scoped API key |
| `GET` | `/auth/keys` | List API keys |
| `DELETE` | `/auth/keys/{id}` | Revoke an API key |

## Quick Example

```bash
# Upload
curl -X POST https://storage.liteio.dev/files/uploads \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"docs/readme.md","content_type":"text/markdown"}'

# Share
curl -X POST https://storage.liteio.dev/files/share \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"docs/readme.md","ttl":86400}'
```

## CLI

```bash
curl -fsSL https://storage.liteio.dev/cli/install.sh | sh
storage login
storage put report.pdf docs/
storage share docs/report.pdf
```

## Security

- **No passwords** — Email magic links or Ed25519 key auth
- **Encrypted at rest** — R2 object storage with TLS in transit
- **Scoped API keys** — Path-prefix restrictions, 90-day TTL
- **Auto-expiring share links** — 1 hour to 7 days
- **Audit logging** — Every action logged with actor and timestamp

## Links

- [Developer Guide](https://storage.liteio.dev/developers) — API docs, code examples, MCP setup
- [API Reference](https://storage.liteio.dev/api) — Full endpoint documentation
- [CLI](https://storage.liteio.dev/cli) — Terminal interface
- [Pricing](https://storage.liteio.dev/pricing) — Free tier and plans
