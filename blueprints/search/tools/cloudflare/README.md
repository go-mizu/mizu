# cloudflare-tool

Auto-register Cloudflare accounts, create API tokens, and manage Workers — all from the command line.

## Install

Requires Python 3.11+ and [uv](https://docs.astral.sh/uv/).

```bash
cd blueprints/search/tools/cloudflare
uv sync
uv run patchright install chromium
```

## Quick Start

```bash
# Register a new account
uv run cloudflare-tool register

# Create a Browser Rendering token (browser opens CF dashboard)
uv run cloudflare-tool token create my-br-token --preset browser-rendering --default

# Write ~/data/cloudflare/cloudflare.json for pkg/scrape
uv run cloudflare-tool token use my-br-token

# Deploy a Worker
uv run cloudflare-tool worker deploy app/worker --name my-worker --default

# Invoke a Worker
uv run cloudflare-tool worker invoke my-worker --path /search?q=test

# Tail Worker logs
uv run cloudflare-tool worker tail my-worker
```

## Commands

### register

```bash
uv run cloudflare-tool register [--no-headless] [--verbose] [--json]
```

Auto-registers a new Cloudflare account end-to-end:
1. Generates random identity via Faker + creates disposable mail.tm mailbox
2. Opens `dash.cloudflare.com/sign-up`, fills form, submits
3. Polls mail.tm for verification email, clicks link
4. Completes onboarding (skips domain, accepts Free plan)
5. Extracts `account_id` from dashboard URL
6. Stores in DuckDB

### account ls / rm

```bash
uv run cloudflare-tool account ls
uv run cloudflare-tool account rm <email>
```

### token create / ls / rm / use

```bash
uv run cloudflare-tool token create <name> [--preset browser-rendering|workers|r2|kv|dns|all] [--default]
uv run cloudflare-tool token ls
uv run cloudflare-tool token rm <name>
uv run cloudflare-tool token use <name>   # also writes cloudflare.json
```

**Presets:**
| Preset | Permissions |
|---|---|
| `browser-rendering` | Account > Browser Rendering > Edit |
| `workers` | Workers Scripts + Routes > Edit |
| `r2` | R2 Storage > Edit |
| `kv` | Workers KV Storage > Edit |
| `dns` | Zone > DNS > Edit |
| `all` | All of the above |

### worker deploy / ls / rm / tail / invoke / use

```bash
uv run cloudflare-tool worker deploy <path> [--name <name>] [--alias <alias>] [--default]
uv run cloudflare-tool worker ls
uv run cloudflare-tool worker rm <alias>
uv run cloudflare-tool worker tail <alias>
uv run cloudflare-tool worker invoke <alias> [--path /] [--method GET] [--body <json>] [--json]
uv run cloudflare-tool worker use <alias>
```

## Data Storage

All state in `~/data/cloudflare/cloudflare.duckdb`. Four tables:
- `accounts` — email, password, account_id, subdomain
- `tokens` — name, token_value, preset, is_default
- `workers` — name, alias, url, is_default
- `op_log` — operation audit trail

`token use <name>` also writes `~/data/cloudflare/cloudflare.json` for compatibility
with `search scrape --cloudflare` (`pkg/scrape/cloudflare.go`).

## Running Tests

```bash
uv run pytest tests/ -v
```

## Troubleshooting

**Registration fails at email verification**: mail.tm may be down. Retry after a minute.

**Token value not extracted**: CF shows the token only once. Run `--no-headless --verbose`.

**wrangler not found**: Ensure Node.js/npm installed. Tool uses `npx wrangler` (auto-downloads).

**subdomain empty**: Run `token use <name>` to trigger subdomain backfill via CF API.
