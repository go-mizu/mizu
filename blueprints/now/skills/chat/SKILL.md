---
name: chat
description: Work with chat.now chats, messages, rooms, and direct conversations from code assistants and terminal agents.
---

# Chat

Use this skill when the user wants to work with `now chat`, `chat.now`, rooms, direct chats, messages, chat history, or chat automation.

## Setup

Install the CLI:

```sh
cd blueprints/now && make install
```

Configure identity (generates ed25519 keypair):

```sh
now chat init --actor u/alice
```

Or set actor per command:

```sh
now chat create --kind room --title general --actor u/alice
```

## Commands

| Command | Use | Output |
|---------|-----|--------|
| `now chat init [--actor <actor>]` | Generate keypair and configure identity | Config file path |
| `now chat create --kind <room\|direct> [--title <t>] [--json]` | Create a chat | Chat ID (or JSON with --json) |
| `now chat get <id>` | Get a chat (members only) | JSON |
| `now chat list [--kind <room\|direct>] [--limit <n>]` | List your chats | JSON with items array |
| `now chat join <id> [--token <t>]` | Join a chat | Nothing on success |
| `now chat send <id> <text> [--json]` | Send a signed message | Message ID (or JSON with --json) |
| `now chat messages <id> [--limit <n>] [--before <mid>]` | List messages (members only) | JSON with items array |

All commands accept `--actor <actor>` to override the configured identity.
All commands accept `--db <path>` to override the database path (default: `$HOME/data/now/chat.duckdb`).

## Security Model

### Identity

- `now chat init` generates an **ed25519 keypair** stored in `~/.config/now/config.json`
- Each actor has a **friendly name** (`u/alice`) and a **fingerprint** (first 16 hex chars of sha256 of public key)
- The name is **bound to the public key** at first registration — impersonation is impossible without the private key

### Signing

- Every operation is **signed** with the actor's private key
- The signature covers: operation, all parameters, actor name, fingerprint, nonce, and timestamp
- Messages store the signature for **non-repudiation** — authorship is provable by anyone

### Replay Protection

- Each request includes a **unique nonce** (16 random bytes) and **timestamp**
- Requests older than 30 seconds are rejected
- Duplicate nonces are rejected

### Permissions

- You can only read (get, list, messages) and write (send) to chats you are a member of
- Creating a chat auto-joins you as a member
- Rooms are open to join. Direct chats are limited to 2 members
- Non-members receive `permission denied` (does not leak whether the chat exists)

## Workflows

Create a room and send a message:

```sh
CHAT=$(now chat create --kind room --title general)
now chat send "$CHAT" "hello everyone"
```

Read message history:

```sh
now chat messages "$CHAT" --limit 20
```

Paginate older messages:

```sh
now chat messages "$CHAT" --before m_abc123 --limit 20
```

Switch actors in a script:

```sh
now chat send "$CHAT" --actor a/bot1 "automated report ready"
now chat send "$CHAT" --actor u/alice "acknowledged"
```

Get full JSON from create:

```sh
now chat create --kind room --title general --json
```

## Actor Format

- `u/:id` for users (e.g. `u/alice`)
- `a/:id` for agents (e.g. `a/bot1`)

## Data Storage

- Data persists in DuckDB at `$HOME/data/now/chat.duckdb`
- Override with `--db /path/to/other.duckdb` or `NOW_DB` env var
- Tables: `keys`, `chats`, `members`, `messages`

## Error Handling

Commands print errors to stderr and exit with code 1:

- `no actor configured` — run `now chat init` or use `--actor`
- `permission denied` — you are not a member of this chat
- `identity conflict` — actor name already registered with a different key
- `invalid signature` — ed25519 verification failed
- `request expired` — signed request older than 30 seconds
- `nonce reused` — replay attempt detected
- `chat not found` — the chat ID does not exist (only shown to members)
- `kind must be "room" or "direct"` — invalid --kind value

## Tips

- Use `$(now chat create ...)` to capture the chat ID in scripts
- Use `--json` on create/send when you need the full object
- The `--before` flag on messages enables cursor-based pagination
- All output is compact JSON, suitable for piping to `jq`
- Config file at `~/.config/now/config.json` has mode 0600 (private key inside)
- Your fingerprint is in the config — share it for others to verify your identity
