# 0766 — Dashboard v2

Full dashboard rebuild. Every section deeply designed for the best DX.
Incorporates patterns from the browse.js file explorer: command palette,
keyboard shortcuts, context menus, inline file preview with syntax highlighting,
markdown rendering, media players, and real upload progress.

---

## 1. Design System

Exact match to existing system in `base.css`:

| Token              | Light          | Dark           |
|--------------------|----------------|----------------|
| `--bg`             | #FAFAF9        | #09090B        |
| `--surface`        | #FFF           | #18181B        |
| `--surface-alt`    | #F4F4F5        | #18181B        |
| `--text`           | #09090B        | #FAFAF9        |
| `--text-2`         | #52525B        | #A1A1AA        |
| `--text-3`         | #A1A1AA        | #52525B        |
| `--border`         | #E4E4E7        | #27272A        |
| `--green`          | #22C55E        | #4ADE80        |
| `--blue`           | #3B82F6        | #60A5FA        |
| `--amber`          | #F59E0B        | #FBBF24        |
| `--red`            | #EF4444        | #F87171        |

Rules:
- **0px border-radius** everywhere
- **1px solid var(--border)** for all borders
- **Inter** for UI, **JetBrains Mono** for data/code
- Dark/light toggle via `html.dark` class, localStorage persistence

## 2. Architecture

Single SSR page shell (`src/pages/dashboard.ts`) + vanilla JS SPA (`public/dashboard.js`).

```
GET /dashboard → authenticated → SSR HTML shell
                 unauthenticated → redirect /
```

URL hash routing: `#overview`, `#files`, `#files/docs/`, `#events`, `#audit`, `#keys`, `#settings`.
Browser back/forward works. Deep links to file paths work.

### State Management

Global `S` state object (same pattern as browse.js):

```js
var S = {
  section: 'overview',      // current section
  path: '',                 // file browser path
  searchQ: '',              // search query
  items: [],                // current file list
  previewItem: null,        // file being previewed
  previewContent: null,     // text content for preview
  uploading: [],            // upload queue
};
```

## 3. Layout

```
┌──────────────────────────────────────────────────────────┐
│  nav (sticky, blur, same as all pages)                    │
├────────────┬─────────────────────────────────────────────┤
│            │                                             │
│  sidebar   │  main content area                          │
│  200px     │  max-width: 960px                           │
│  sticky    │                                             │
│            │  (changes based on active section)          │
│  Overview  │                                             │
│  Files     │                                             │
│  Events    │                                             │
│  Audit     │                                             │
│  API Keys  │                                             │
│  Settings  │                                             │
│            │                                             │
├────────────┴─────────────────────────────────────────────┤
```

Mobile (≤ 768px): sidebar collapses to horizontal scrollable tab bar.

## 4. Global Features

### 4.1 Command Palette (Cmd+K / Ctrl+K)

Full-screen overlay with search input. Searches files across the entire storage.
Arrow keys navigate results, Enter opens, Esc closes.

Results show file icon + name + path. Selecting a file navigates to Files
section and opens preview. Selecting a folder navigates into it.

### 4.2 Keyboard Shortcuts

| Key          | Context     | Action                          |
|--------------|-------------|---------------------------------|
| `/`          | Global      | Focus search (section-aware)    |
| `Cmd+K`      | Global      | Open command palette            |
| `Esc`        | Global      | Close modal / preview / palette |
| `?`          | Global      | Show shortcuts modal            |
| `←` `→`      | Preview     | Previous / next file            |
| `Backspace`  | Files       | Navigate to parent folder       |
| `g o`        | Global      | Go to Overview                  |
| `g f`        | Global      | Go to Files                     |
| `g e`        | Global      | Go to Events                    |
| `g a`        | Global      | Go to Audit                     |
| `g k`        | Global      | Go to Keys                      |
| `g s`        | Global      | Go to Settings                  |

### 4.3 Toast Notifications

Bottom-right stack. Auto-dismiss after 3.5s. Types: ok (green), err (red), info (default).

### 4.4 Modal System

- **Confirm modal**: title, text, cancel/confirm buttons
- **Prompt modal**: title, text, input field, cancel/confirm
- **Custom modal**: arbitrary HTML content (for share links, shortcuts, etc.)

All modals: backdrop click closes, Esc closes, slide-up animation.

## 5. Sections

### 5.1 Overview

Summary cards (3-column grid, 1px gap borders):

| Card         | Source             | Display                        |
|--------------|--------------------|--------------------------------|
| Total Files  | GET /files/stats   | count + formatted total size   |
| Storage Used | GET /files/stats   | formatted bytes + file count   |
| API Keys     | GET /auth/keys     | active key count               |

Below: **Recent Activity** table — last 10 events from GET /files/log.
Columns: Action (badge), Path, Size, Time.
Clicking a row navigates to Events section.

All data fetched in parallel with loading spinners per card.

### 5.2 Files

Complete file browser built from first principles. Inspired by the existing
browse.js but redesigned for the dashboard context.

#### Toolbar
```
[ Breadcrumbs: ~ / docs / reports / ]
[ Search input           ] [ + Folder ] [ ↑ Upload ]
```

#### File List
Table with columns: Icon, Name, Size, Modified, Actions (on hover).

- **Sort**: folders first, then alphabetical by name
- **Icons**: type-aware (folder, file, image, video, audio, code, markdown,
  doc, sheet, archive, text) — same detection as browse.js
- **Double-click**: open folder / preview file
- **Right-click**: context menu (Open, Download, Rename, Move, Share, Delete)
- **Single-click**: select (visual highlight)
- **Search**: debounced 250ms, calls /files/search, results replace list
- **Empty state**: "This folder is empty" with upload CTA

#### Context Menu
Right-click on file/folder shows context menu:
- Folder: Open, Rename, Delete
- File: Preview, Download, Share, Rename, Delete

#### File Preview
Inline preview panel (replaces file list when active):
- **Breadcrumb bar** with back button, path segments, file info
- **Navigation**: ← → arrows to browse siblings, count indicator
- **Preview types**:
  - Code: syntax highlighting (JS, TS, Go, Python, Rust, etc.)
  - Markdown: rendered with headings, lists, tables, code blocks, math
  - Text/CSV: plain display / table view
  - Images: direct display via presigned URL
  - Audio: custom player with waveform, progress, volume
  - Video: custom player with controls, fullscreen
  - Other: icon + name + size + download button
- **Markdown toggle**: switch between rendered and source view
- **Actions**: Download, Copy link, Share

#### Upload
- **Drag & drop**: full-window drop zone with visual feedback
- **File input**: click Upload button to browse
- **Folder upload**: recursive directory traversal (webkitGetAsEntry)
- **Upload panel**: persistent bottom panel showing all uploads
  - Progress bar per file with percentage
  - Real XHR progress (not fake steps)
  - Parallel uploads (3 concurrent max)
  - Retry on failure (3 attempts with backoff)
  - Status: pending, uploading, retrying, done, error
- **3-step presigned flow**: init → PUT to R2 → complete

#### Share
Modal with TTL selector (1h, 1d, 7d, 30d). Creates share link,
auto-copies to clipboard, shows copyable URL.

#### Rename
Prompt modal, pre-filled with current name. Calls POST /files/move.

#### Delete
Confirm modal with file name. Calls DELETE /files/{path}.

#### New Folder
Prompt modal for folder name. Calls POST /files/mkdir.

### 5.3 Events

Full event mutation log (writes, moves, deletes).

- **Source**: GET /files/log?limit=200
- **Columns**: TX, Action (badge), Path, Size, Message, Time
- **Filters**: All | Write | Move | Delete (client-side toggle buttons)
- **Badges**: write=green, move=blue, delete=red
- **Empty state**: "No events" with filter context
- **Path truncation**: 45 chars with leading ellipsis

### 5.4 Audit Log

Complete API action trail.

- **Source**: GET /dashboard/audit (paginated, server-side)
- **Columns**: Action (badge), Path, IP, Time
- **Filters**: All | Read | Write | Delete | Share | Login (server-side filter)
- **Pagination**: 50 per page, "Load more (N remaining)" button
- **Badges**: Different colors per action type
  - write/register = green
  - read = gray
  - rm = red
  - mv = blue
  - login = amber
  - share = amber
  - key* = blue
- **90-day retention notice** in subtitle

### 5.5 API Keys

Full CRUD for API tokens.

#### Create Form
Inline form with fields:
- **Name**: text input, placeholder "my-bot"
- **Path Prefix**: text input, placeholder "docs/ (optional)"
- **Expires**: select (Never, 1h, 1d, 7d, 30d, 90d)
- **Create** button (solid black)

#### Token Reveal
After creation, amber-bordered box shows:
- Full token in monospace code block
- Copy button
- Warning: "Store this token securely. It will not be shown again."

#### Keys Table
Columns: Name, ID, Prefix, Expires, Created, Actions (delete button).

#### Delete
Confirm modal: "Revoke key [name]? Any services using this key will lose
access immediately." Calls DELETE /auth/keys/{id}.

### 5.6 Settings

#### Account Section
Key-value rows with borders:
- Actor (e.g., "test")
- Email (or "not set" in muted text)
- Created date
- Active sessions count

#### Active Shares Section
Table of non-expired share links from GET /dashboard/shares:
- Path, Expires, Created
- Revoke button per share (future API)

#### Actions Section
- **Sign Out** button (red border, danger style)
  → Confirm modal → redirect to /auth/logout

## 6. API Endpoints Used

### Existing
| Endpoint                    | Method | Used In       |
|-----------------------------|--------|---------------|
| /files                      | GET    | Files         |
| /files/search               | GET    | Files, Palette|
| /files/stats                | GET    | Overview      |
| /files/log                  | GET    | Overview, Events |
| /files/move                 | POST   | Files rename  |
| /files/share                | POST   | Files share   |
| /files/mkdir                | POST   | Files         |
| /files/uploads              | POST   | Files upload  |
| /files/uploads/complete     | POST   | Files upload  |
| /files/{path}               | GET    | Files download|
| /files/{path}               | DELETE | Files delete  |
| /auth/keys                  | GET    | Overview, Keys|
| /auth/keys                  | POST   | Keys create   |
| /auth/keys/{id}             | DELETE | Keys delete   |
| /auth/logout                | GET    | Settings      |

### Dashboard-specific
| Endpoint            | Method | Used In       |
|---------------------|--------|---------------|
| /dashboard/audit    | GET    | Audit         |
| /dashboard/shares   | GET    | Settings      |
| /dashboard/account  | GET    | Settings      |

## 7. Implementation Files

| File                      | Purpose                        | Lines (est.) |
|---------------------------|--------------------------------|-------------|
| spec/0766_dashboard.md    | This spec                      | —           |
| src/pages/dashboard.ts    | SSR page shell (update)        | ~110        |
| src/routes/dashboard.ts   | API endpoints (keep as-is)     | 84          |
| public/dashboard.css      | Dashboard styles (full rewrite)| ~550        |
| public/dashboard.js       | SPA logic (full rewrite)       | ~1400       |

## 8. Implementation Checklist

### Infrastructure
- [ ] Hash-based routing with history support
- [ ] Global state object (`S`)
- [ ] Command palette (Cmd+K / Ctrl+K)
- [ ] Keyboard shortcuts (/, Esc, ?, arrows, g+key combos)
- [ ] Toast notification system
- [ ] Modal system (confirm, prompt, custom)
- [ ] Context menu system

### Overview
- [ ] Stat cards with parallel data loading
- [ ] Recent activity table with clickable rows

### Files
- [ ] Breadcrumb navigation
- [ ] File list with type-aware icons
- [ ] Search with debounce
- [ ] File preview with syntax highlighting
- [ ] Markdown rendering (headings, lists, tables, code blocks)
- [ ] Image preview via presigned URL
- [ ] Audio player with waveform + controls
- [ ] Video player with controls + fullscreen
- [ ] Context menu (right-click)
- [ ] Drag-and-drop upload with folder support
- [ ] Upload panel with real XHR progress
- [ ] Parallel uploads (3 concurrent)
- [ ] Retry on failure
- [ ] Share modal with TTL + copy
- [ ] Rename modal
- [ ] Delete confirmation
- [ ] New folder modal
- [ ] Sibling navigation (← →) in preview

### Events
- [ ] Event log table
- [ ] Client-side action filters (All/Write/Move/Delete)
- [ ] Action badges with colors

### Audit
- [ ] Paginated audit log
- [ ] Server-side action filters
- [ ] Load more pagination
- [ ] Action badges

### API Keys
- [ ] Create form (name, prefix, expiry)
- [ ] Token reveal box with copy
- [ ] Keys table
- [ ] Delete with confirmation

### Settings
- [ ] Account info (actor, email, created, sessions)
- [ ] Active shares list
- [ ] Sign out with confirmation

### Responsive
- [ ] Sidebar → horizontal tabs at ≤768px
- [ ] Hide table columns at ≤480px
- [ ] Touch support (long-press for context menu)
- [ ] Mobile-friendly modals
