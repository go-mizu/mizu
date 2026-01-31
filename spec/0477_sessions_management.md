# 0477 - Session Management (OpenClaw-Compatible)

## Research Summary

Deep analysis of OpenClaw (ClawdBot) session management system from source code at
`$HOME/github/openclaw/openclaw` and on-disk session files at `$HOME/.openclaw`.

## 1. OpenClaw Session Storage Architecture

### 1.1 Directory Layout

```
~/.openclaw/agents/{agentId}/sessions/
  sessions.json           # Session metadata index
  {sessionId}.jsonl       # Per-session JSONL transcript
```

Default agent is `main`, so default path is `~/.openclaw/agents/main/sessions/`.

### 1.2 sessions.json Format

A `Record<string, SessionEntry>` keyed by **session key** strings:

```json
{
  "agent:main:main": {
    "sessionId": "08ef14de-...",
    "updatedAt": 1738312080000,
    "chatType": "direct",
    "deliveryContext": {
      "channel": "telegram",
      "target": "+14155551234",
      "account": "default"
    },
    "origin": {
      "label": "User",
      "provider": "telegram",
      "surface": "dm"
    },
    "compactionCount": 0,
    "skillsSnapshot": { ... },
    "inputTokens": 150000,
    "outputTokens": 8000,
    "totalTokens": 158000,
    "modelProvider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "contextTokens": 200000,
    "systemPromptReport": { "workspace": { ... } }
  }
}
```

Key SessionEntry fields:
- `sessionId`: UUID v4 identifier
- `updatedAt`: Timestamp (ms since epoch)
- `chatType`: "direct" | "group" | "channel"
- `deliveryContext`: Channel + target + account for message delivery
- `origin`: Original message source metadata
- `model`, `modelProvider`, `contextTokens`: Model configuration
- `inputTokens`, `outputTokens`, `totalTokens`: Token usage tracking
- `compactionCount`: How many times transcript was compacted
- `label`: User-set custom label
- `displayName`, `subject`: Display metadata
- `thinkingLevel`, `verboseLevel`, `sendPolicy`: Session flags

### 1.3 JSONL Transcript Format

Each line is a JSON object. Entry types:

**Session header (first line):**
```json
{"type":"session","version":2,"id":"08ef14de-...","timestamp":"2024-01-31T14:48:00.000Z","cwd":"/path/to/workspace"}
```

**Message entries:**
```json
{"type":"message","id":"msg_01...","timestamp":"2024-01-31T14:48:05.000Z","message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}
{"type":"message","id":"msg_02...","timestamp":"2024-01-31T14:48:10.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi there!"}]},"usage":{"input":1000,"output":50}}
```

**Model change entries:**
```json
{"type":"model_change","model":"claude-sonnet-4-20250514","timestamp":"2024-01-31T14:48:00.000Z"}
```

**Custom entries (cache TTL, model snapshots):**
```json
{"type":"custom","key":"openclaw.cache-ttl","value":45000,"timestamp":"..."}
{"type":"custom","key":"model-snapshot","value":{"model":"claude-sonnet-4-20250514","contextTokens":200000},"timestamp":"..."}
```

### 1.4 Session Key Format

- **Direct messages (collapsed):** `agent:{agentId}:main`
- **Direct messages (per-peer):** `agent:{agentId}:{peerId}`
- **Group sessions:** `agent:{agentId}:{channel}:group:{groupId}`
- **Thread sessions:** `{baseKey}:thread:{threadId}`

### 1.5 Session Lifecycle

1. **Creation:** On first message, `resolveSession()` builds session key, generates UUID, creates entry
2. **Freshness:** Evaluated against reset policy (daily at configurable hour, or idle timeout)
3. **Reset:** `/new` or `/reset` commands create new session ID, keep metadata
4. **Compaction:** Keeps last N lines (default 400), archives old file
5. **Deletion:** Remove from sessions.json, archive transcript file

### 1.6 Concurrency

- File-based lock: `{storePath}.lock` with 25ms poll, 10s timeout, 30s stale eviction
- Session store cached in memory with 45s TTL, validated via file mtime
- Atomic writes on Unix: write temp file, atomic rename, 0o600 permissions

## 2. OpenClaw CLI Commands

### 2.1 Core Commands

| Command | Description |
|---------|-------------|
| `openclaw agent` | Run agent turn via Gateway |
| `openclaw sessions` | List stored sessions |
| `openclaw status` | Channel health + recent sessions |
| `openclaw health` | Gateway health check |
| `openclaw message send` | Send message to channel |
| `openclaw agents list` | List configured agents |

### 2.2 `openclaw agent` Flags

```
-m, --message <text>     Message body (required)
-t, --to <id>            Recipient for session key derivation
--session-id <uuid>      Explicit session ID
--agent <id>             Agent ID
--thinking <level>       Extended thinking mode
--deliver                Send reply to channel
--channel <channel>      Delivery channel
--json                   JSON output
--local                  Run locally (needs API key)
--timeout <seconds>      Timeout (default 600)
```

### 2.3 `openclaw sessions` Flags

```
--json                   JSON output
--store <path>           Session store path
--active <minutes>       Filter by activity window
```

### 2.4 `openclaw message send` Flags

```
--channel <channel>      Channel type
-t, --target <dest>      Delivery target (required)
-m, --message <text>     Message body (required)
--media <path-or-url>    Attach media
--reply-to <id>          Reply to message
--thread-id <id>         Thread ID
--json                   JSON output
```

### 2.5 `openclaw status` Flags

```
--json                   JSON output
--all                    Full diagnosis
--usage                  Model usage/quota
--deep                   Probe channels
```

## 3. Implementation Plan for OpenBot

### 3.1 File-Based Session Store

Add a file-based session store (`pkg/session/filestore.go`) that:
- Stores `sessions.json` index at `~/.openbot/agents/{agentId}/sessions/`
- Writes JSONL transcripts for each session
- Implements session key derivation matching OpenClaw
- Supports daily reset and idle timeout policies
- Uses file locking for concurrent access
- Integrates with existing SQLite store (dual-write: SQLite for queries, files for OpenClaw compatibility)

### 3.2 CLI Commands

Rewrite `cmd/openbot` to match OpenClaw CLI:

| OpenBot Command | Matches |
|-----------------|---------|
| `openbot agent -m <msg>` | `openclaw agent -m <msg>` |
| `openbot sessions` | `openclaw sessions` |
| `openbot status` | `openclaw status` |
| `openbot message send -t <target> -m <msg>` | `openclaw message send` |
| `openbot history [id]` | Session transcript viewer |
| `openbot help` | Usage help |

### 3.3 Session-Channel Integration

- Telegram messages derive session key: `agent:default:{telegramUserId}`
- Group messages: `agent:default:telegram:group:{chatId}`
- Session key stored in session metadata
- Messages written to both SQLite and JSONL transcript

### 3.4 E2E Testing

- Real Telegram bot connection test (requires bot token)
- Real Anthropic API tool call test (requires API key)
- File listing scenario: user asks to list files, bot uses `list_files` tool, returns results via Telegram
- Session persistence: verify JSONL transcript written correctly

## 4. On-Disk Session Files Observed

From `~/.openclaw/agents/main/sessions/`:

```
sessions.json                              (5.5KB, session index)
08ef14de-xxxx-xxxx-xxxx-xxxxxxxxxxxx.jsonl (145KB, 63 lines)
```

JSONL entry type distribution:
- `session`: 1 (header)
- `model_change`: 1
- `thinking_level_change`: 1
- `custom:model-snapshot`: 1
- `custom:openclaw.cache-ttl`: 6
- `message`: 53

## 5. OpenClaw Gateway RPC Methods for Sessions

```
sessions.list      - List sessions with metadata
sessions.preview   - Preview session contents
sessions.resolve   - Resolve session key from label/ID
sessions.patch     - Update session metadata
sessions.reset     - Reset session (new ID, keep metadata)
sessions.delete    - Delete session
sessions.compact   - Compact transcript
```
