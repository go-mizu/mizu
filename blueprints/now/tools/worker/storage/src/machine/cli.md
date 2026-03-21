# Storage CLI

> Single binary. Zero dependencies. macOS, Linux, Windows.

Upload, download, share, search, and manage files from your terminal.

## Installation

### macOS / Linux

```bash
curl -fsSL https://storage.liteio.dev/cli/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://storage.liteio.dev/cli/install.ps1 | iex
```

### Package Managers

```bash
npm i -g @liteio/storage-cli     # npm
bun i -g @liteio/storage-cli     # Bun
deno install -g npm:@liteio/storage-cli  # Deno
```

### Direct Download

| Platform | Architecture | URL |
|----------|-------------|-----|
| macOS | Apple Silicon (arm64) | `https://storage.liteio.dev/cli/releases/latest/storage-darwin-arm64` |
| macOS | Intel (amd64) | `https://storage.liteio.dev/cli/releases/latest/storage-darwin-amd64` |
| Linux | x64 (amd64) | `https://storage.liteio.dev/cli/releases/latest/storage-linux-amd64` |
| Linux | ARM (arm64) | `https://storage.liteio.dev/cli/releases/latest/storage-linux-arm64` |
| Windows | x64 (amd64) | `https://storage.liteio.dev/cli/releases/latest/storage-windows-amd64.exe` |
| Windows | ARM (arm64) | `https://storage.liteio.dev/cli/releases/latest/storage-windows-arm64.exe` |

## Quick Start

```bash
# 1. Install
curl -fsSL https://storage.liteio.dev/cli/install.sh | sh

# 2. Sign in (opens browser)
storage login

# 3. Upload a file
storage put report.pdf docs/

# 4. Share it
storage share docs/report.pdf
# → https://storage.liteio.dev/s/k7x9m2 (expires in 1h)
```

## Commands

### File Operations

#### `storage put <file> [destination]`

Upload a file or pipe from stdin. Streams directly to the edge — no temp files.

- **Aliases:** `upload`, `push`
- Use `-` as file to read from stdin: `pg_dump mydb | storage put - backup.sql`

```bash
storage put photo.jpg images/
```

#### `storage get <path> [destination]`

Download a file from storage to the current directory or a specified path.

- **Aliases:** `download`, `pull`

```bash
storage get images/photo.jpg
storage get images/photo.jpg ~/Downloads/
```

#### `storage cat <path>`

Print file contents to stdout. Useful for piping into other tools.

- **Aliases:** `read`

```bash
storage cat config.json | jq '.'
```

#### `storage ls [path]`

List files and folders. Shows name, size, content type, and last modified time.

- **Aliases:** `list`

```bash
storage ls docs/
```

#### `storage mv <source> <destination>`

Move or rename a file. Works across folders.

- **Aliases:** `move`, `rename`

```bash
storage mv draft.md final.md
storage mv drafts/post.md published/post.md
```

#### `storage rm <path...>`

Delete one or more files. Folders are deleted recursively.

- **Aliases:** `delete`, `del`

```bash
storage rm old-draft.md
storage rm archive/  # deletes folder and all contents
```

### Discovery

#### `storage find <query>`

Search files by name across your entire storage. Multi-word queries with relevance scoring.

- **Aliases:** `search`

```bash
storage find "quarterly report"
```

#### `storage stat`

Show storage usage: total file count and bytes used.

- **Aliases:** `stats`

```bash
storage stat
storage stat --json | jq '{files: .count, mb: (.bytes / 1048576 | floor)}'
```

### Sharing

#### `storage share <path> [--ttl <seconds>]`

Create a temporary public link. Anyone with the link can download.

- **Aliases:** `sign`
- Default TTL: 1 hour (3600 seconds)
- Maximum TTL: 7 days (604800 seconds)
- **Flags:** `--ttl`, `--expires`, `-x`

```bash
storage share report.pdf              # 1 hour link
storage share report.pdf --ttl 86400  # 24 hour link
```

### Authentication

#### `storage login [name]`

Authenticate via browser. Opens your default browser, verifies identity, saves the token.

- Pass a name to register a new account.

```bash
storage login
storage login my-username  # register new account
```

#### `storage logout`

Remove saved credentials and invalidate session.

#### `storage token [<token>]`

Show current auth token and its source, or set a new one.

```bash
storage token              # show current token
storage token sk_abc123    # set token
```

#### `storage key create <name> [--prefix <path>]`

Create a named API key. Optionally scope to a path prefix.

- **Aliases for `key`:** `keys`

```bash
storage key create github-deploy --prefix cdn/
```

#### `storage key list`

List all API keys with metadata (name, prefix, expiry).

#### `storage key revoke <id>`

Revoke an API key by ID. Immediately invalidates it.

- **Aliases:** `delete`, `rm`

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | Output as JSON (for scripting and piping into `jq`) |
| `--quiet` | `-q` | Suppress non-essential output |
| `--token` | `-t` | Use a specific auth token (overrides env and config) |
| `--endpoint` | | Override the API base URL |
| `--no-color` | | Disable colored output (also: `NO_COLOR=1` env var) |
| `--help` | `-h` | Show help for any command |
| `--version` | `-V` | Print CLI version |

## Authentication

Two methods are supported:

### Browser Login (interactive)

```bash
storage login
```

Opens your browser. No password needed. Token saved to `~/.config/storage/token`.

### API Keys (automation)

```bash
storage key create deploy --prefix cdn/
# Returns: sk_a8f3c7e2d1b9...4k2m
```

Set `STORAGE_TOKEN` as an environment variable in CI/CD or scripts.

**Token resolution order** (highest priority first):

1. `--token` flag
2. `STORAGE_TOKEN` environment variable
3. `~/.config/storage/token` file

## Environment Variables

| Variable | Description |
|----------|-------------|
| `STORAGE_TOKEN` | API key or session token |
| `STORAGE_ENDPOINT` | API base URL (default: `https://storage.liteio.dev`) |
| `NO_COLOR` | Set to `1` to disable colored output |
| `XDG_CONFIG_HOME` | Config directory base (default: `~/.config`) |

## Recipes

### Upload build artifacts from CI

```bash
export STORAGE_TOKEN=$SECRET_TOKEN
storage put dist/app.js cdn/v1.2.0/
storage put dist/app.css cdn/v1.2.0/
```

### Stream a database backup

```bash
pg_dump mydb | storage put - backups/$(date +%Y-%m-%d).sql
```

### Share a file for 24 hours

```bash
storage share docs/report.pdf --ttl 86400
```

### List files as JSON and filter with jq

```bash
storage ls docs/ --json | jq '.[].name'
```

### Create a scoped deploy key

```bash
storage key create github-deploy --prefix cdn/
# Add the returned token as STORAGE_TOKEN in your CI secrets
```

### Download and pipe to another tool

```bash
storage cat config.json | jq '.database'
```

### Move files between folders

```bash
storage mv drafts/post.md published/post.md
```

### Bulk delete old files

```bash
storage ls archive/ --json | jq -r '.[].path' | xargs -I{} storage rm {}
```

### Check storage usage

```bash
storage stat --json | jq '{files: .count, mb: (.bytes / 1048576 | floor)}'
```

## Links

- [API Reference](https://storage.liteio.dev/api)
- [Developer Guide](https://storage.liteio.dev/developers)
- [Pricing](https://storage.liteio.dev/pricing)
- [CLI Documentation (human view)](https://storage.liteio.dev/cli)
