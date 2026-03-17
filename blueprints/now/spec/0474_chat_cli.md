# 0474: chat-now — npm CLI + TUI for chat.go-mizu.workers.dev

## Status: Draft

## Summary

`chat-now` is an npm-publishable CLI and TUI that connects to the chat.go-mizu.workers.dev Cloudflare Worker. Running `chat-now` with no args launches a three-panel TUI (rooms, messages, members). All operations are also available as subcommands for scripting. Works with Node.js 18+ and Bun.

## Goals

- Best-in-class developer experience: zero-config for new users, import for existing Go CLI users
- Three-panel TUI with keyboard-driven navigation
- Full CLI subcommand surface for scripting and automation
- Ed25519 identity compatible with the Go CLI (`now chat`)
- Publishable to npm as `chat-now`
- Polling-based real-time with swappable transport abstraction

## Non-Goals

- WebSocket/SSE transport (designed for, not implemented)
- Offline mode or local storage
- End-to-end encryption
- File attachments or rich media
- Key rotation and account deletion (use the Go CLI or direct API calls)

## Architecture

```
chat-now (npm package, ESM only)
├── CLI Layer (Commander.js)     ← subcommands: init, send, list, etc.
├── TUI Layer (Ink + React)      ← three-panel: rooms, messages, members
├── Auth Layer                   ← Ed25519 signing via @noble/ed25519
├── API Client                   ← REST client with fetch, polling transport
├── State (Zustand)              ← rooms, messages, members, active chat
└── Config                       ← ~/.config/chat-now/config.json
```

Runtime: Node.js 18+ / Bun. ESM only. Ink 4+ requires ESM.

## API Surface

Server: `https://chat.go-mizu.workers.dev`

### Endpoints Used

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/register` | Register actor with public key |
| `POST` | `/api/chat` | Create chat |
| `GET` | `/api/chat` | List chats |
| `GET` | `/api/chat/:id` | Get chat |
| `POST` | `/api/chat/:id/join` | Join chat |
| `POST` | `/api/chat/:id/messages` | Send message |
| `GET` | `/api/chat/:id/messages` | List messages |
| `POST` | `/api/chat/dm` | Start or resume a DM conversation |
| `GET` | `/api/chat/dm` | List DM conversations |

Note: The worker rejects `kind: "direct"` on `POST /api/chat`. DMs must use the `/api/chat/dm` endpoint with a `{ peer: "u/bob" }` body.

### Constraints

- Request body limit: 65,536 bytes (enforced by worker).
- Message text limit: 4,000 characters.
- Actor names: must match `/^[ua]\/[\w.@-]{1,64}$/`.
- Registration: rate limited to 5 per IP per hour.

### Authentication

Every request (except register) carries an `Authorization` header:

```
CHAT-ED25519 Credential=u/alice, Timestamp=1710000000, Signature=<base64url>
```

Signing protocol matches the Go CLI exactly:

1. Build canonical request: `METHOD\nPATH\nSORTED_QUERY\nHEX(SHA256(BODY))`
2. Build string to sign: `CHAT-ED25519\nTIMESTAMP\nACTOR\nHEX(SHA256(CANONICAL))`
3. Sign with Ed25519 private key
4. Encode signature as base64url

Uses `@noble/ed25519` — pure JS, no native deps, works in Node and Bun.

## CLI Commands

Running `chat-now` with no args launches the TUI.

| Command | Description | Output |
|---------|-------------|--------|
| `chat-now` | Launch TUI | interactive |
| `chat-now init [--actor <name>] [--import]` | Generate keypair or import from Go CLI | config path |
| `chat-now create [--title <t>] [--visibility <public\|private>]` | Create room | JSON |
| `chat-now dm <peer>` | Start or resume DM with peer | JSON |
| `chat-now join <id>` | Join chat | (silent on success) |
| `chat-now send <id> <text>` | Send message | JSON |
| `chat-now messages <id> [--limit n] [--before <id>]` | List messages | JSON |
| `chat-now list [--limit n]` | List chats | JSON |
| `chat-now dms [--limit n]` | List DM conversations | JSON |
| `chat-now get <id>` | Get chat details | JSON |
| `chat-now whoami` | Show identity & fingerprint | JSON |

All subcommands output JSON to stdout. Errors go to stderr with exit code 1.

### Global Flags

- `--server <url>` — Override server URL (default: `https://chat.go-mizu.workers.dev`)
- `--config <path>` — Override config file path
- `--pretty` — Pretty-print JSON output

## Identity & Config

### Config File

Location: `~/.config/chat-now/config.json` (mode 0600)

```json
{
  "actor": "u/alice",
  "public_key": "<base64url Ed25519 public key>",
  "private_key": "<base64url Ed25519 private key>",
  "fingerprint": "<hex first 16 chars of SHA256(pubkey)>",
  "server": "https://chat.go-mizu.workers.dev"
}
```

### Base64url Encoding

Keys are stored as base64url. The Go CLI uses **padded** base64url (trailing `=`). The import logic must handle both padded and unpadded base64url. When writing new configs, use unpadded base64url (no trailing `=`).

### Init Flow

1. Check for `~/.config/now/config.json` (Go CLI config).
   - If found and `--import` flag or user confirms: copy actor, keys, fingerprint.
   - Strip padding from imported base64url keys.
   - Skip registration (already registered with worker).
2. If no import: prompt for actor name (must match `/^[ua]\/[\w.@-]{1,64}$/`).
3. Generate Ed25519 keypair with `@noble/ed25519`.
4. Call `POST /api/register` with actor and public key.
5. Store recovery code from response.
6. Write config file with mode 0600.
7. Display: actor, fingerprint, config path.

### Identity Resolution (per command)

1. `--config` flag path
2. `~/.config/chat-now/config.json`
3. `~/.config/now/config.json` (Go CLI fallback, read-only)
4. Error: "Run `chat-now init` to set up your identity"

## TUI Design

### Layout

```
┌─────────┬──────────────────────────┬──────────┐
│ Rooms   │ Messages                 │ Members  │
│         │                          │          │
│ #general│ u/alice  10:32           │ u/alice  │
│ #dev    │   hey everyone           │ u/bob    │
│ #random │                          │ u/carol  │
│         │ u/bob    10:33           │          │
│         │   yo!                    │          │
│         │                          │          │
├─────────┴──────────────────────────┴──────────┤
│ > type a message...                            │
├────────────────────────────────────────────────┤
│ u/alice · #general · 3 online · polling: 3s    │
└────────────────────────────────────────────────┘
```

### Panels

**Room List (left)**: Lists joined chats. Active room highlighted. Shows unread indicator. Kind icon: `#` for rooms, `@` for direct.

**Message View (center)**: Scrollable message history. Actor name colored consistently (hash-based). Timestamp relative (10:32, yesterday, Mar 15). Auto-scrolls to bottom on new messages. Load older messages on scroll-up.

**Member List (right)**: Members of the active chat. Current user highlighted. Derived from message authors seen in the chat (the worker has no `/members` endpoint yet). Shows unique actors from loaded messages.

**Input Bar (bottom)**: Text input with cursor. Enter to send. Multi-line not supported (keep it simple).

**Status Bar (footer)**: Current identity, active room, member count, polling status.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus: rooms → messages → members → input |
| `Enter` | Send message (when input focused) / Select room (when rooms focused) |
| `↑` `↓` | Scroll messages / navigate rooms / navigate members |
| `Ctrl+N` | Create new room (inline prompt) |
| `Ctrl+J` | Join room by ID (inline prompt) |
| `Ctrl+K` | Quick room switch (fuzzy filter) |
| `Ctrl+R` | Force refresh |
| `Ctrl+Q` | Quit |
| `Ctrl+C` | Quit |
| `Escape` | Cancel current prompt / unfocus |
| `PageUp` / `PageDown` | Scroll messages by page |

### State Management

Zustand store with these slices:

```typescript
interface ChatStore {
  // Identity
  identity: Identity | null

  // Rooms
  rooms: Chat[]
  activeRoomId: string | null
  setActiveRoom(id: string): void

  // Messages (keyed by chat ID)
  messages: Record<string, Message[]>
  addMessages(chatId: string, msgs: Message[]): void

  // Members (keyed by chat ID)
  members: Record<string, string[]>

  // UI
  focusedPanel: 'rooms' | 'messages' | 'members' | 'input'
  cycleFocus(): void

  // Polling
  pollInterval: number
  lastPoll: Record<string, number>
}
```

### Polling

- Default interval: 3 seconds.
- Only polls the active chat's messages.
- Fetches room list every 30 seconds.
- Shows polling status in status bar.
- Deduplicates messages by ID.
- Transport interface for future SSE/WS:

```typescript
interface Transport {
  subscribe(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe
  unsubscribe(chatId: string): void
}
```

`PollingTransport` implements this. Future `SSETransport` or `WebSocketTransport` is a drop-in replacement.

## API Client

```typescript
class ChatClient {
  constructor(private config: Config, private signer: RequestSigner)

  // Auth
  register(actor: string, publicKey: Uint8Array): Promise<{ actor: string; recovery_code: string }>

  // Rooms
  createChat(opts: { title?: string; visibility?: string }): Promise<Chat>
  getChat(id: string): Promise<Chat>
  listChats(opts?: { limit?: number }): Promise<Chat[]>
  joinChat(id: string): Promise<void>

  // DMs
  startDm(peer: string): Promise<Chat>
  listDms(opts?: { limit?: number }): Promise<Chat[]>

  // Messages
  sendMessage(chatId: string, text: string): Promise<Message>
  listMessages(chatId: string, opts?: { limit?: number; before?: string }): Promise<Message[]>
}
```

All list methods unwrap the `{ items: [...] }` response envelope from the worker.

### Domain Types

```typescript
interface Chat {
  id: string          // c_<16hex>
  kind: string        // "room" or "direct"
  title: string
  creator: string
  peer?: string       // For DM chats only
  created_at: string  // ISO 8601
}

interface Message {
  id: string          // m_<16hex>
  chat: string
  actor: string
  fingerprint: string // SHA256(pubkey)[:16]
  text: string        // max 4000 chars
  signature: string   // base64url Ed25519 signature
  created_at: string  // ISO 8601
}
```

Uses native `fetch`. All methods sign requests via `RequestSigner`. Throws typed errors:

- `AuthError` — 401/403, identity issue
- `RateLimitError` — 429, includes retry-after
- `NetworkError` — connection failed
- `ApiError` — other HTTP errors

## Error Handling

### TUI Mode

- Network errors: show "disconnected" in status bar, retry with exponential backoff (3s, 6s, 12s, max 30s), resume on success.
- Auth errors: show "identity rejected — run chat-now init" in status bar.
- Rate limits: pause polling, show countdown in status bar.

### CLI Mode

- All errors to stderr.
- Exit code 1 on failure.
- JSON error body when available: `{ "error": "message" }`

## File Structure

```
tools/chat-cli/
├── package.json
├── tsconfig.json
├── bin/
│   └── chat-now.js              # ESM shebang entry
├── src/
│   ├── cli.tsx                  # Entry point, commander setup, TUI launch
│   ├── auth/
│   │   ├── config.ts            # Config read/write/import
│   │   ├── crypto.ts            # Ed25519 operations via @noble
│   │   └── signer.ts            # Canonical request building, header generation
│   ├── api/
│   │   ├── client.ts            # ChatClient REST implementation
│   │   ├── types.ts             # Chat, Message, API response types
│   │   └── transport.ts         # Transport interface + PollingTransport
│   ├── tui/
│   │   ├── App.tsx              # Root component, three-panel layout
│   │   ├── RoomList.tsx         # Left panel
│   │   ├── MessageView.tsx      # Center panel
│   │   ├── MemberList.tsx       # Right panel
│   │   ├── InputBar.tsx         # Message input
│   │   ├── StatusBar.tsx        # Footer status line
│   │   └── Prompt.tsx           # Inline prompts (create room, join, fuzzy switch)
│   ├── store/
│   │   └── chat.ts              # Zustand store
│   └── utils/
│       ├── format.ts            # Time, colors, actor display
│       └── keys.ts              # Keybinding map
└── README.md
```

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `ink` | ^4.0 | React-based terminal UI |
| `react` | ^18.0 | UI component model |
| `@noble/ed25519` | ^2.0 | Ed25519 signing (pure JS) |
| `commander` | ^12.0 | CLI argument parsing |
| `@inkjs/ui` | ^2.0 | TextInput, Spinner, Select |
| `zustand` | ^4.0 | State management |

Dev dependencies: `typescript`, `@types/react`, `tsx` (for development).

No native dependencies. Works with Node.js 18+ and Bun.

## package.json

```json
{
  "name": "chat-now",
  "version": "0.1.0",
  "type": "module",
  "bin": { "chat-now": "./bin/chat-now.js" },
  "exports": { ".": "./dist/cli.js" },
  "engines": { "node": ">=18" },
  "files": ["dist/", "bin/"]
}
```

Build with `tsc`. `bin/chat-now.js` imports from `dist/cli.js`.

## Actor Color Assignment

Consistent color per actor using a hash of the actor name mapped to a palette of 8 terminal colors (avoiding black/white for readability):

```
cyan, green, yellow, blue, magenta, red, gray, white
```

`hash(actor) % 8` selects the color. Current user always gets a distinct style (bold).

## Testing Strategy

- **Auth/crypto**: Unit tests for canonical request building, signing, verification against known Go CLI outputs.
- **API client**: Mock fetch, verify request format and headers.
- **Store**: Unit tests for state transitions.
- **TUI**: Manual testing (Ink testing utilities for key components).
- **Integration**: End-to-end against the live worker with a test identity.

## Future Considerations

- SSE/WebSocket transport (drop-in via Transport interface)
- Message editing and deletion
- File attachments
- Notification sounds
- Themes/color schemes
- `/slash` commands in TUI input (e.g., `/join`, `/create`)
