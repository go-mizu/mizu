# 0746: chat-now TUI Redesign — Modern Conversation UI

## Status: Draft

## Summary

Complete TUI redesign inspired by Claude Code, Gemini CLI, and Codex CLI. Replace the three-panel layout with a single-column conversation stream. Add markdown rendering with syntax highlighting, gradient header, multi-line composer, and fuzzy room switcher overlay. Keep auth, API client, store, and CLI commands unchanged.

## What Changes

### Delete
- `src/tui/RoomList.tsx` — replaced by `Ctrl+K` overlay
- `src/tui/MemberList.tsx` — members visible from conversation
- `src/tui/InputBar.tsx` — replaced by multi-line Composer
- `src/tui/StatusBar.tsx` — replaced by Header
- `src/tui/Prompt.tsx` — replaced by Overlay component

### Rewrite
- `src/tui/App.tsx` — single-column layout with overlays
- `src/tui/MessageView.tsx` → `src/tui/MessageStream.tsx` — markdown rendering

### New
- `src/tui/Header.tsx` — gradient branding + room + identity
- `src/tui/Composer.tsx` — multi-line input, Enter sends, Shift+Enter newline
- `src/tui/RoomSwitcher.tsx` — Ctrl+K fuzzy filter overlay
- `src/tui/Overlay.tsx` — generic overlay container
- `src/tui/Markdown.tsx` — marked AST → Ink components with syntax highlighting

### Keep As-Is
- `src/auth/*` — config, signer
- `src/api/*` — client, transport, types
- `src/store/chat.ts` — minor tweak: remove focusedPanel (no panels)
- `src/cli.tsx` — no changes
- `src/utils/format.ts` — keep actor colors
- `src/utils/keys.ts` — update keybinding map

## Layout

```
╭─────────────────────────────────────────────╮
│  ✦ chat-now         #general    u/alice     │
╰─────────────────────────────────────────────╯

  u/bob                              10:32 AM
  hey everyone, check this out:

  ```ts
  const x = await fetch("/api/chat");
  ```

  u/alice                            10:33 AM
  nice! **looks good** to me

─────────────────────────────────────────────
  > type a message...
  >
─────────────────────────────────────────────
```

Single column. No panels. No borders except header box and composer separator.

## Components

### Header
One-line bar with gradient app name (via `ink-gradient`), current room, and identity.
- Left: `✦ chat-now` in gradient
- Center: `#room-name` or `@peer`
- Right: `u/actor`
- Boxed with rounded border

### MessageStream
Full-width scrollable message list. Each message:
- Line 1: actor name (colored) + timestamp (dim, right-aligned)
- Line 2+: message body rendered as markdown
- Blank line between messages

Markdown rendering via `marked` parse → custom Ink renderer:
- `**bold**` → `<Text bold>`
- `*italic*` → `<Text italic>`
- `` `code` `` → `<Text color="yellow">`
- Fenced code blocks → `lowlight.highlight()` → colored `<Text>` spans
- Links → `<Text color="cyan" underline>`

Auto-scrolls to bottom on new messages. Scroll up with arrow keys / PageUp.

### Composer
Multi-line text input at bottom, separated by horizontal rule.
- `>` prompt on each line
- Enter sends (when single line or cursor at end)
- Shift+Enter inserts newline
- Shows line count indicator when multi-line: `[3 lines]`
- Escape clears

### RoomSwitcher (Ctrl+K overlay)
Floating centered box:
```
┌─ Switch Room ──────────────────┐
│ > gen                          │
│                                │
│   #general                     │
│   #dev                         │
│   @bob                         │
└────────────────────────────────┘
```
Type to fuzzy-filter. Arrow keys to navigate. Enter to select. Escape to dismiss.

### Overlay
Generic overlay wrapper used by RoomSwitcher, create-room prompt, and join-room prompt. Renders a bordered box centered over the message stream.

## Dependencies

### Add
| Package | Version | Purpose |
|---------|---------|---------|
| `ink-gradient` | ^3.0.0 | Gradient text in header |
| `ink-spinner` | ^5.0.0 | Loading states |
| `lowlight` | ^3.3.0 | Syntax highlighting (uses highlight.js) |
| `marked` | ^15.0.0 | Markdown parsing |
| `ink-text-input` | ^6.0.0 | Text input with cursor |

### Remove
| Package | Reason |
|---------|--------|
| `@inkjs/ui` | Too generic, replaced by custom components |

### Keep
`ink`, `react`, `commander`, `@noble/ed25519`, `@noble/hashes`, `zustand`

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | New line in composer |
| `Ctrl+K` | Fuzzy room switcher |
| `Ctrl+←` / `Ctrl+→` | Cycle prev/next room |
| `Ctrl+N` | Create room (overlay prompt) |
| `Ctrl+J` | Join room (overlay prompt) |
| `Ctrl+R` | Force refresh |
| `Ctrl+C` / `Ctrl+Q` | Quit |
| `↑` / `↓` | Scroll messages |
| `PageUp` / `PageDown` | Scroll by page |
| `Escape` | Dismiss overlay / clear composer |

## Store Changes

Remove `focusedPanel` / `cycleFocus` / `setFocus` — no panels to cycle. Add:
- `overlay: null | 'switcher' | 'create' | 'join'`
- `setOverlay(mode): void`
- `composerLines: number` (for UI hint)

## File Structure

```
src/tui/
├── App.tsx              # Layout, keybindings, overlay dispatch
├── Header.tsx           # Gradient branding + room + identity
├── MessageStream.tsx    # Scrollable messages with markdown
├── Composer.tsx         # Multi-line input
├── RoomSwitcher.tsx     # Ctrl+K fuzzy overlay
├── Overlay.tsx          # Generic overlay container
└── Markdown.tsx         # marked → Ink renderer with syntax highlighting
```

## Testing

- `Markdown.tsx`: Unit test marked→Ink rendering for bold, italic, code, fenced blocks
- Everything else: manual TUI testing
- Existing tests (signer, config, client, store) remain unchanged
