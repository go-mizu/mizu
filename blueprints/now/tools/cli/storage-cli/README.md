# @anthropic/storage

CLI for [storage.now](https://storage.liteio.dev) — upload, download, and share files from your terminal.

Zero dependencies. Single file. Works with Node 18+, Bun, and Deno.

```
npx @anthropic/storage
```

## Install

```bash
# Run without installing
npx @anthropic/storage
bunx @anthropic/storage

# Install globally
npm install -g @anthropic/storage
```

After installing globally, the `storage` command is available everywhere:

```bash
storage login
storage put photo.jpg images
storage share images/photo.jpg --expires 7d
```

## Quick start

```bash
# Authenticate (opens browser, OAuth 2.0 PKCE)
storage login

# Upload a file
storage put report.pdf docs

# List your buckets
storage ls

# List objects in a bucket
storage ls docs

# Download a file
storage get docs/report.pdf

# Share with a signed URL (expires in 1 hour by default)
storage share docs/report.pdf

# Print file to stdout (pipe to other tools)
storage cat docs/data.json | jq '.items'
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `storage login` | Authenticate via browser (OAuth 2.0 with PKCE) |
| `storage logout` | Remove saved credentials |
| `storage token` | Show current token source |
| `storage token <token>` | Save a token directly (for CI/headless) |

### File operations

| Command | Description |
|---------|-------------|
| `storage ls` | List all buckets |
| `storage ls <bucket> [prefix]` | List objects in a bucket |
| `storage put <file> [bucket/path]` | Upload a file |
| `storage get <bucket/path> [dest]` | Download a file |
| `storage cat <bucket/path>` | Print file contents to stdout |
| `storage rm <bucket/path...>` | Delete one or more objects |
| `storage mv <from> <to>` | Move or rename an object |
| `storage cp <from> <to>` | Copy an object (supports cross-bucket) |

### Sharing & discovery

| Command | Description |
|---------|-------------|
| `storage share <bucket/path>` | Create a signed download URL |
| `storage info <bucket/path>` | Show object metadata |
| `storage search <query>` | Search for objects by name |
| `storage stats` | Show storage usage and quota |

### Buckets & API keys

| Command | Description |
|---------|-------------|
| `storage bucket create <name>` | Create a new bucket |
| `storage bucket rm <name>` | Delete a bucket |
| `storage key create <name>` | Create an API key |
| `storage key list` | List API keys |
| `storage key revoke <id>` | Revoke an API key |

## Usage

### Upload

```bash
# Upload to default bucket
storage put photo.jpg

# Upload to a specific bucket and path
storage put photo.jpg images/vacation/photo.jpg

# Upload with explicit content type
storage put data.bin assets/data.bin --type application/octet-stream

# Upload from stdin
echo '{"key": "value"}' | storage put - config/settings.json

# Pipe a command's output
curl -s https://example.com/data.csv | storage put - reports/latest.csv
```

### Download

```bash
# Download to current directory
storage get docs/report.pdf

# Download to a specific path
storage get docs/report.pdf ~/Downloads/report.pdf

# Download to stdout
storage get docs/report.pdf -

# Download and pipe to another command
storage cat docs/data.json | jq '.items[] | .name'

# Download a public file (no auth required)
storage get images/logo.png --public
```

### Share

```bash
# Signed URL (default: 1 hour)
storage share docs/report.pdf

# Custom expiration
storage share docs/report.pdf --expires 30m
storage share docs/report.pdf --expires 7d
storage share docs/report.pdf --expires 3600    # seconds
```

Duration format: `30s`, `15m`, `2h`, `7d`, or plain seconds.

### Delete

```bash
# Delete a single object
storage rm docs/old-report.pdf

# Delete multiple objects (same bucket)
storage rm docs/file1.txt docs/file2.txt docs/file3.txt

# Delete recursively by prefix
storage rm docs/archive/ --recursive
```

### Search

```bash
# Search across all buckets
storage search quarterly

# Search in a specific bucket
storage search report --bucket docs

# Filter by MIME type
storage search --type image/png --bucket assets
```

### Buckets

```bash
# Create a private bucket
storage bucket create documents

# Create a public bucket
storage bucket create cdn --public

# Create with file size limit and allowed types
storage bucket create uploads --size-limit 10MB --types "image/png,image/jpeg"

# Delete a bucket (must be empty)
storage bucket rm old-bucket

# Delete a bucket and all its contents
storage bucket rm old-bucket --force
```

### API keys

```bash
# Create a key with full access
storage key create deploy-bot

# Create a scoped key
storage key create readonly --scope "files:read,bucket:read"

# Create a key restricted to a path prefix
storage key create uploads-only --prefix "uploads/"

# List all keys
storage key list

# Revoke a key
storage key revoke <id>
```

## Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | Output as JSON |
| `--quiet` | `-q` | Suppress non-essential output |
| `--token <token>` | `-t` | Override authentication token |
| `--endpoint <url>` | `-e` | Override API base URL |
| `--no-color` | | Disable colored output |
| `--version` | `-v` | Show version |
| `--help` | `-h` | Show help |

## Configuration

Configuration is resolved in order of priority:

1. **CLI flags** (`--token`, `--endpoint`) — highest priority
2. **Environment variables** (`STORAGE_TOKEN`, `STORAGE_ENDPOINT`, `STORAGE_BUCKET`)
3. **Token file** (`~/.config/storage/token`) — written by `storage login`
4. **Config file** (`~/.config/storage/config`)

### Environment variables

```bash
# Set token for CI/CD pipelines
export STORAGE_TOKEN=sk_your_api_key_here

# Custom endpoint
export STORAGE_ENDPOINT=https://storage.example.com

# Default bucket (used when no bucket is specified)
export STORAGE_BUCKET=my-default-bucket
```

### Config file

`~/.config/storage/config` (or `$XDG_CONFIG_HOME/storage/config`):

```ini
endpoint=https://storage.liteio.dev
bucket=default
```

## JSON output

Every command supports `--json` for machine-readable output:

```bash
# List buckets as JSON
storage ls --json

# List objects as JSON
storage ls docs --json

# Pipe to jq
storage ls docs --json | jq '.[].path'

# Use in scripts
TOTAL=$(storage stats --json | jq '.total_size')
```

## Pipes & scripting

The CLI is designed to compose with Unix pipes:

```bash
# Upload from a pipe
tar czf - ./src | storage put - backups/src.tar.gz

# Process downloaded data
storage cat docs/users.json | jq '.[] | select(.active)' > active.json

# Batch upload
find ./images -name '*.png' -exec sh -c 'storage put "$1" assets/"$(basename "$1")"' _ {} \;

# Mirror a directory
for f in ./dist/*; do
  storage put "$f" cdn/"$(basename "$f")"
done
```

## CI/CD

```yaml
# GitHub Actions
- name: Deploy to storage
  env:
    STORAGE_TOKEN: ${{ secrets.STORAGE_TOKEN }}
  run: |
    npx @anthropic/storage put ./dist/bundle.js cdn/bundle.js
    npx @anthropic/storage put ./dist/style.css cdn/style.css
```

```bash
# Direct token usage (no login required)
storage token sk_your_api_key_here
storage put ./build/app.js cdn/app.js
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error (bad arguments) |
| 3 | Authentication error |
| 4 | Not found |
| 5 | Conflict |
| 6 | Permission denied |
| 7 | Network error |

## Runtime compatibility

| Runtime | Version | Status |
|---------|---------|--------|
| Node.js | 18+ | Supported |
| Bun | 1.0+ | Supported |
| Deno | 1.35+ | Supported |

The CLI uses only Node.js built-in modules (`node:crypto`, `node:fs`, `node:http`, `node:os`, `node:path`, `node:child_process`, `node:stream/promises`) — no dependencies to install or audit.

## License

MIT
