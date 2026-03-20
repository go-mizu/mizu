# @liteio/storage-cli

The official CLI for [Liteio Storage API](https://storage.liteio.dev) — upload, download, organize, and share files from your terminal.

Zero dependencies. Single file. Works with Node 18+, Bun, and Deno.

```bash
npx @liteio/storage-cli --help
```

## Why?

- **No SDK needed** — one command to upload, download, or share any file
- **Pipe-friendly** — reads stdin, writes stdout, uses proper exit codes
- **Works everywhere** — Node.js, Bun, Deno, CI/CD, Docker, cron
- **Zero dependencies** — single `.mjs` file, nothing to install or audit
- **Secure by default** — OAuth PKCE login, scoped API keys, `0600` token permissions

## Install

```bash
# Run without installing (recommended for trying it out)
npx @liteio/storage-cli
bunx @liteio/storage-cli

# Install globally for everyday use
npm install -g @liteio/storage-cli
```

After installing globally, the `storage` command is available everywhere:

```bash
storage login
storage put photo.jpg images/
storage share images/photo.jpg --expires 7d
```

### Running with Deno

```bash
deno run --allow-all npm:@liteio/storage-cli --help
deno run --allow-all npm:@liteio/storage-cli ls
```

You can also create a shell alias for convenience:

```bash
alias storage="deno run --allow-all npm:@liteio/storage-cli"
```

## Quick start

```bash
# 1. Authenticate (opens browser, OAuth 2.0 PKCE — takes 5 seconds)
storage login

# 2. Upload a file
storage put report.pdf docs/report.pdf

# 3. List your files
storage ls docs/

# 4. Share it with anyone (link expires in 1 hour by default)
storage share docs/report.pdf
# → https://storage.liteio.dev/s/Xk9mP2nQ4rWz8AbCdEf3g

# 5. Download it back
storage get docs/report.pdf

# 6. Print a file to stdout (great for piping)
storage cat docs/data.json | jq '.items'
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `storage login` | Authenticate via browser (OAuth 2.0 with PKCE) |
| `storage logout` | Remove saved credentials |
| `storage token` | Show current token source and prefix |
| `storage token <token>` | Save a token directly (useful for CI or headless servers) |

### File operations

| Command | Description |
|---------|-------------|
| `storage ls [path]` | List files and directories. No args = root. |
| `storage put <file> [path]` | Upload a file. Use `-` to read from stdin. |
| `storage get <path> [dest]` | Download a file. Use `-` to write to stdout. |
| `storage cat <path>` | Print file contents to stdout (alias for `get <path> -`) |
| `storage rm <path...>` | Delete one or more files |
| `storage mv <from> <to>` | Move or rename a file (server-side, instant) |

### Sharing & discovery

| Command | Description |
|---------|-------------|
| `storage share <path>` | Create a signed share URL (default: 1 hour) |
| `storage find <query>` | Search files by name |
| `storage stat` | Show total files and storage usage |

### API keys

| Command | Description |
|---------|-------------|
| `storage key create <name>` | Create an API key (shown once — save it!) |
| `storage key list` | List all API keys |
| `storage key rm <id>` | Revoke an API key |

## Usage examples

### Upload

```bash
# Upload a file (destination = same filename at root)
storage put photo.jpg

# Upload to a specific path
storage put photo.jpg images/vacation/photo.jpg

# Upload to a directory (trailing slash appends the filename)
storage put photo.jpg images/

# Upload from stdin — great for piping
echo '{"key": "value"}' | storage put - config/settings.json

# Pipe command output directly to storage
curl -s https://example.com/data.csv | storage put - reports/latest.csv
```

### Download

```bash
# Download to current directory (keeps original filename)
storage get docs/report.pdf

# Download to a specific path
storage get docs/report.pdf ~/Downloads/report.pdf

# Download to stdout (use - as destination)
storage cat docs/data.json | jq '.items[] | .name'
```

### Share

Share links are short, opaque, and expire automatically. They reveal nothing about your account or file paths.

```bash
# Create a share link (default: expires in 1 hour)
storage share docs/report.pdf
# → https://storage.liteio.dev/s/Xk9mP2nQ4rWz8AbCdEf3g

# Custom expiration
storage share docs/report.pdf --expires 30m
storage share docs/report.pdf --expires 7d

# Get the URL as JSON (useful for scripting)
storage share docs/report.pdf --json | jq -r '.url'
```

Duration format: `30s`, `15m`, `2h`, `7d`, or plain seconds (`3600`).

### Delete

```bash
# Delete a single file
storage rm docs/old-report.pdf

# Delete a directory recursively (prompts for confirmation)
storage rm logs/ --recursive

# Skip confirmation prompt
storage rm logs/ --recursive --force
```

### Search

```bash
# Search by filename
storage find quarterly

# Get results as JSON for scripting
storage find "*.pdf" --json | jq '.results[].path'
```

### Move / rename

```bash
# Rename a file
storage mv drafts/post.md published/post.md

# Move to another directory
storage mv old/report.pdf archive/report.pdf
```

### API keys

Use API keys for CI/CD, scripts, and automation — they don't expire (unless you set one) and can be restricted to specific path prefixes.

```bash
# Create a key (full access)
storage key create deploy-bot

# Create a key restricted to a path prefix
storage key create uploads-only --prefix uploads/

# List all keys
storage key list

# Revoke a key by ID
storage key rm ak_abc123def456
```

## Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | Output as JSON (machine-readable) |
| `--quiet` | `-q` | Suppress non-essential output |
| `--token <token>` | `-t` | Override authentication token |
| `--endpoint <url>` | `-e` | Override API base URL |
| `--no-color` | | Disable colored output |
| `--version` | `-v` | Show version |
| `--help` | `-h` | Show help |

## Configuration

The CLI resolves configuration in this order (first match wins):

1. **CLI flags** (`--token`, `--endpoint`)
2. **Environment variables** (`STORAGE_TOKEN`, `STORAGE_ENDPOINT`)
3. **Token file** (`~/.config/storage/token`) — written by `storage login`
4. **Config file** (`~/.config/storage/config`)

### Environment variables

```bash
# Use an API key for CI/CD pipelines
export STORAGE_TOKEN=sk_your_api_key_here

# Point to a different endpoint (e.g. staging)
export STORAGE_ENDPOINT=https://storage.liteio.dev
```

### Config file

Located at `~/.config/storage/config` (or `$XDG_CONFIG_HOME/storage/config`):

```ini
endpoint=https://storage.liteio.dev
```

## JSON output

Every command supports `--json` for machine-readable output — perfect for scripting:

```bash
# List files as JSON
storage ls docs/ --json

# Extract file names with jq
storage ls docs/ --json | jq '.entries[].name'

# Get storage usage as a number
storage stat --json | jq '.bytes'
```

## Pipes & scripting

The CLI is designed as a well-behaved Unix tool: it reads stdin, writes stdout, sends status messages to stderr, and uses meaningful exit codes.

```bash
# Pipe a database dump directly to storage (no temp files)
pg_dump mydb | gzip | storage put - backups/db/$(date +%F).sql.gz

# Process downloaded JSON
storage cat docs/users.json | jq '.[] | select(.active)' > active.json

# Batch upload all PNGs in a directory
find ./images -name '*.png' -exec sh -c \
  'storage put "$1" images/"$(basename "$1")"' _ {} \;
```

## CI/CD

```yaml
# GitHub Actions example
- name: Upload build artifacts
  env:
    STORAGE_TOKEN: ${{ secrets.STORAGE_TOKEN }}
  run: |
    npx @liteio/storage-cli put ./dist/bundle.js cdn/bundle.js
    npx @liteio/storage-cli put ./dist/style.css cdn/style.css
```

## Exit codes

Scripts can react to specific error types:

| Code | Name | When |
|------|------|------|
| 0 | Success | Operation completed normally |
| 1 | Error | Unspecified runtime error |
| 2 | Usage | Bad arguments or flags |
| 3 | Auth | Missing or invalid token |
| 4 | Not found | File does not exist |
| 5 | Conflict | Name collision |
| 6 | Permission | Token lacks access to this path |
| 7 | Network | Connection failed or timed out |

```bash
storage get backups/latest.sql.gz .
case $? in
  0) echo "restored ok" ;;
  4) echo "no backup found" ;;
  3) echo "auth failed — run: storage login" ;;
  7) echo "network error — retry later" ;;
esac
```

## Runtime compatibility

| Runtime | Version | Install |
|---------|---------|---------|
| Node.js | 18+ | `npx @liteio/storage-cli` |
| Bun | 1.0+ | `bunx @liteio/storage-cli` |
| Deno | 1.35+ | `deno run --allow-all npm:@liteio/storage-cli` |

The CLI uses only Node.js built-in modules (`node:crypto`, `node:fs`, `node:http`, `node:os`, `node:path`, `node:child_process`, `node:buffer`, `node:process`) — zero external dependencies.

## License

MIT
