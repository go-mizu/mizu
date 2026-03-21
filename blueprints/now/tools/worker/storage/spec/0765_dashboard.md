# 0765 — Dashboard

Post-login dashboard for authenticated users. Replaces the bare `/browse`
redirect with a full control center.

---

## 1. Design Philosophy

Follow the existing design system exactly:
- **Monochrome** — black/white/gray, accent colors only for status
- **No rounded corners** — 0px border-radius everywhere
- **Inter + JetBrains Mono** — sans-serif for UI, monospace for data/code
- **Rectilinear** — thin 1px borders, clean grid alignment
- **Dark/light toggle** — respects existing theme persistence

Inspired by Vercel's dashboard (dense, developer-focused, monochrome) and
Linear's audit log (filterable table with timestamps).

## 2. Route

```
GET /dashboard    →  authenticated: render dashboard
                     unauthenticated: redirect to /
```

## 3. Layout

Single page with a left sidebar for navigation and a main content area.
The sidebar collapses on mobile (≤ 768px) to a top tab bar.

```
┌──────────────────────────────────────────────────────────┐
│  nav (same global nav as all pages)                      │
├────────────┬─────────────────────────────────────────────┤
│            │                                             │
│  sidebar   │  main content area                          │
│            │                                             │
│  Overview  │  (changes based on active section)          │
│  Files     │                                             │
│  Events    │                                             │
│  Audit     │                                             │
│  API Keys  │                                             │
│  Settings  │                                             │
│            │                                             │
├────────────┴─────────────────────────────────────────────┤
```

## 4. Sections

### 4.1 Overview (default)

Summary cards in a 3-column grid:

| Card           | Data Source         | Display                          |
|----------------|---------------------|----------------------------------|
| Total Files    | GET /files/stats    | file count + formatted bytes     |
| API Keys       | GET /auth/keys      | active key count                 |
| Recent Events  | GET /files/log      | last 5 events as mini timeline   |

Below cards: **Recent Activity** — last 10 events as a table (tx, action,
path, time). Clicking a row navigates to Events section filtered to that path.

### 4.2 Files

Complete file browser built from first principles (not embedding `/browse`).

- Breadcrumb navigation (click segments to navigate up)
- File listing: folders first, then files alphabetically
- Row hover shows action buttons: download, share, rename, delete
- Search: filters files by name in real-time
- Upload: drag-and-drop zone + upload button with progress bar
- Create folder: modal prompt for name
- Share: generates share link with 24h TTL, modal with copy button
- Rename: modal prompt, calls /files/move
- Delete: confirmation modal, calls DELETE /files/{path}
- Download: opens file in new tab via /files/{path} redirect

### 4.3 Events

Full event log table. Columns: tx, action, path, size, message, time.

- Pagination: load 50 at a time, "Load more" button
- Filter by action type (write/move/delete) — client-side toggle
- Filter by path prefix — text input
- Reverse chronological order (newest first)
- Action badges: write=green, move=blue, delete=red

### 4.4 Audit Log

Full audit trail. Columns: action, path, IP, timestamp.

- Pagination: 50 rows, "Load more"
- Filter by action type
- Shows IP addresses and relative timestamps
- 90-day retention notice

API route needed: `GET /dashboard/audit` (new, protected)

### 4.5 API Keys

Full CRUD UI for API tokens.

- **List**: table with id, name, prefix, expires_at, created_at
- **Create**: inline form (name, prefix, expiry) → shows token ONCE in a
  copyable code block with warning
- **Delete**: click delete → confirmation modal → revoke

Uses existing endpoints:
- GET /auth/keys
- POST /auth/keys
- DELETE /auth/keys/{id}

### 4.6 Settings

- **Account info**: actor name, email
- **Active sessions**: count of active sessions
- **Sign out**: button → POST /auth/logout

## 5. New API Endpoints

### GET /dashboard/audit

Returns audit log entries for the authenticated user.

```
Query params:
  limit   — max 200, default 50
  offset  — pagination offset
  action  — filter by action type (optional)

Response:
{
  entries: [{ action, path, ip, ts }],
  total: number
}
```

### GET /dashboard/shares

Returns active (non-expired) share links for the authenticated user.

```
Response:
{
  shares: [{ token, path, expires_at, created_at }]
}
```

## 6. Implementation Files

| File                        | Purpose                              |
|-----------------------------|--------------------------------------|
| src/pages/dashboard.ts      | SSR page renderer                    |
| public/dashboard.css        | Dashboard-specific styles            |
| public/dashboard.js         | Client-side SPA logic                |
| src/routes/dashboard.ts     | API endpoints (audit, shares)        |
| src/index.ts                | Wire up routes + page                |

## 7. Implementation Checklist

- [x] Create `src/routes/dashboard.ts` with audit + shares + account endpoints
- [x] Create `src/pages/dashboard.ts` with SSR page shell
- [x] Create `public/dashboard.css` following design system
- [x] Create `public/dashboard.js` with all client-side logic
- [x] Wire dashboard route in `src/index.ts`
- [x] Update home page hero CTA to link to /dashboard
- [x] Update magic link redirect to /dashboard
- [x] Overview section: stats cards + recent activity table
- [x] Files section: full file browser (breadcrumbs, list, upload, mkdir, rename, delete, share, download, search)
- [x] Events section: full event log with action filters (write/move/delete)
- [x] Audit section: full audit log with action filters + pagination
- [x] API Keys section: list + create form + delete with token reveal + copy
- [x] Settings section: account info + active sessions + sign out
- [x] Mobile responsive (sidebar → horizontal tab bar at ≤768px)
- [x] Dark/light theme support
- [x] All actions functional (create key, delete key, load more, filters, upload, share, rename, delete files)
- [x] Modal system (confirm delete, prompt rename/new folder, share link)
- [x] Toast notifications (success/error feedback)
- [x] Drag-and-drop file upload with progress bar
