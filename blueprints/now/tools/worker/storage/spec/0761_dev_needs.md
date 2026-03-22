# 0761 — Developers Page: From First Principles

> Delete the current `/developers` page and rebuild from scratch.
> Stop listing endpoints. Start showing value.

---

## 1. Why Rewrite

The current page is an endpoint catalog: "17 endpoints, three primitives, here's curl."
That tells developers *what* we have. It doesn't tell them *why they should care*.

Vercel Blob leads with "the simplest way to store files on a global network" and shows one code example. HuggingFace leads with "AI-native object storage" and shows a pricing table that destroys S3. Both answer the developer's real question: **"Why should I use this instead of what I already have?"**

Our page should answer that question in 5 seconds.

---

## 2. Developer Needs Analysis

When a developer evaluates a storage service, they ask these questions in order:

| Priority | Question | What they're really asking |
|----------|----------|---------------------------|
| 1 | **Can I start fast?** | How many minutes to first upload? Do I need to provision anything? |
| 2 | **How does it fit my stack?** | REST? SDK? Framework bindings? What does the code look like? |
| 3 | **What will it cost me?** | Especially egress. S3 egress is $90/TB. CloudFront is extra. |
| 4 | **Is it fast?** | Latency, CDN, global distribution. Can I serve files to users? |
| 5 | **Does it work with AI?** | Can Claude/ChatGPT read and write to it? MCP? |
| 6 | **Is it secure?** | Auth, encryption, access control, compliance. |
| 7 | **What makes this different?** | Why not S3? Why not Vercel Blob? Why not R2 directly? |
| 8 | **Can I automate it?** | CLI, CI/CD, API keys, scripting. |

The page must answer ALL of these. In this order.

---

## 3. Our Unique Value Propositions

What we have that others don't — or do worse:

### 3.1 Zero Egress, Zero Complexity

| Provider | Storage/TB/mo | Egress/TB | Hidden costs |
|----------|--------------|-----------|--------------|
| AWS S3 | $23 | $90 | PUT/GET request fees, CloudFront, transfer acceleration |
| Supabase Storage | ~$0.021/GB | Metered bandwidth | Requires Supabase project, bucket config, RLS policies |
| Vercel Blob | ~$0.15/GB | Included (w/ limits) | Tied to Vercel platform |
| **Storage** | **Included in plan** | **$0** | **None. The API is the product.** |

We don't charge per-request. We don't charge egress. The developer pays a flat plan and gets storage, bandwidth, and API access.

### 3.2 AI-Native from Day One

No other storage service has MCP built in. Period.

- Connect to Claude Desktop, Claude.ai, or ChatGPT in under a minute
- 8 MCP tools with full tool annotations (read-only hints, destructive hints)
- ChatGPT rich widgets (file browser, viewer, share links, stats)
- OAuth 2.0 + PKCE for secure third-party auth
- AI can save files, search, organize, and share — via natural language

This isn't a bolt-on integration. The MCP endpoint ships with the same Worker.

### 3.3 No SDK Required

```bash
# Initiate upload → get presigned URL → PUT directly to R2
curl -X POST https://storage.liteio.dev/files/uploads \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"path":"logo.svg"}'

# Download (302 redirect to R2)
curl -L https://storage.liteio.dev/files/logo.svg \
  -H "Authorization: Bearer $TOKEN" -o logo.svg
```

That's it. No `npm install`. No client initialization. No connection pooling.
Works with `curl`, `fetch`, `requests`, `http.NewRequest`, `HttpClient` — anything that speaks HTTP.

### 3.4 Path-Based Simplicity

No buckets. No containers. No object IDs. No ARNs. Everything under `/files`.

```
POST /files/uploads          → initiate upload (get presigned URL)
GET  /files/assets/logo.svg  → download (302 redirect to R2)
DELETE /files/assets/logo.svg → delete
GET  /files?prefix=assets/   → list folder
```

Directories are implicit from `/` separators. The API is the filesystem.

### 3.5 Presigned Uploads (Zero Proxy)

File bytes never touch our Worker. The upload flow:

1. `POST /presign/upload` → get a signed R2 URL (1 hour TTL)
2. `PUT <signed-url>` → upload directly to object store
3. `POST /presign/complete` → confirm and index

This means: no 100MB Worker limits, no bandwidth costs on our side, no proxy latency. Your client talks directly to the global object store.

### 3.6 Edge-First, Global by Default

- 300+ Cloudflare edge locations
- Auth resolves at the nearest edge
- Metadata queries: <50ms globally
- File downloads: nearest R2 replica
- No region to choose. No replication to configure.

### 3.7 Three Auth Methods, One Token

| Method | For | How |
|--------|-----|-----|
| Magic link | Humans | Email link → session cookie |
| Ed25519 challenge | Machines | Public key → signed challenge → token |
| API key | Automation | `sk_*` prefix, scoped to path prefix, 90-day TTL |

All three resolve to the same bearer token. All three work with the same API.
Plus: OAuth 2.0 with PKCE for third-party apps.

---

## 4. Page Design

### 4.0 Design Principles

- **Show, don't list.** Every feature gets a code example or visual, not a bullet point.
- **Answer "why" before "what."** Lead with the problem, then the solution.
- **Real code only.** No pseudo-code. Every example should be copy-pasteable.
- **Dark-first.** Developer pages look better dark. Light mode toggle available.
- **No marketing fluff.** Developers smell bullshit. Be direct.

### 4.1 Page Structure

```
┌─────────────────────────────────────────────────────────────────┐
│ NAV: Storage · developers(active) · api · cli · ai · pricing   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  HERO                                                           │
│  "Storage for developers"                                       │
│  "Upload a file in one request. Serve it globally. $0 egress."  │
│  [API Reference]  [Get started]                                 │
│                                                                 │
│  ┌─────────────────────────────────────────────┐                │
│  │ $ curl -X PUT .../f/demo.json \             │                │
│  │     -H "Authorization: Bearer $TOKEN" \     │                │
│  │     -d '{"hello":"world"}'                  │                │
│  │                                             │                │
│  │ → 201 · demo.json · 17 B · application/json│                │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "30 seconds to first upload"                          │
│                                                                 │
│  Three columns:                                                 │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │
│  │ 1. Get a key │ │ 2. Upload    │ │ 3. Share     │            │
│  │              │ │              │ │              │            │
│  │ POST /auth/  │ │ PUT /f/...   │ │ POST /share  │            │
│  │ challenge    │ │ -T file.pdf  │ │ → /s/k7f2m   │            │
│  └──────────────┘ └──────────────┘ └──────────────┘            │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "Why not just use S3?"                                │
│                                                                 │
│  Comparison table (our killer section):                          │
│  ┌──────────────────────────────────────────────────────┐       │
│  │           │ S3     │ R2 raw │ Vercel │ Storage       │       │
│  │ Egress    │ $90/TB │  $0    │ limit  │ $0            │       │
│  │ Setup     │ IAM+   │ CORS+  │ npm i  │ curl          │       │
│  │ Auth      │ IAM    │ HMAC   │ OAuth  │ 3 methods     │       │
│  │ AI/MCP    │ No     │ No     │ No     │ Built-in      │       │
│  │ SDK req'd │ Yes    │ Yes    │ Yes    │ No            │       │
│  │ Buckets   │ Yes    │ Yes    │ No     │ No            │       │
│  │ Regions   │ Pick 1 │ Auto   │ Auto   │ Auto          │       │
│  └──────────────────────────────────────────────────────┘       │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "The API"                                             │
│  "Files by path. Folders from structure. Links for sharing."    │
│                                                                 │
│  Three-panel layout (not just endpoint listing):                │
│                                                                 │
│  Panel 1: FILES                                                 │
│  Left: explanation + key behaviors                              │
│  Right: terminal with PUT/GET/DELETE example                    │
│                                                                 │
│  Panel 2: ORGANIZE                                              │
│  Left: explanation                                              │
│  Right: terminal with ls/find/mv/stat example                   │
│                                                                 │
│  Panel 3: SHARE                                                 │
│  Left: explanation                                              │
│  Right: terminal with share + recipient accessing               │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "Works with any language"                             │
│  Tab switcher: curl | JavaScript | Python | Go                  │
│  Same upload shown in all four                                  │
│  (Keep this from current page — it's good)                      │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "AI-native"                                           │
│  "Your files, inside Claude and ChatGPT."                       │
│                                                                 │
│  Two-column:                                                    │
│  Left: What you can do                                          │
│    - "Save this as notes.md"                                    │
│    - "What files do I have?"                                    │
│    - "Share report.pdf with my team"                            │
│    - "Find all .csv files"                                      │
│  Right: Mock of Claude/ChatGPT conversation                     │
│                                                                 │
│  Below: "Connect in under a minute"                             │
│    Claude: Settings → Integrations → Add → paste URL            │
│    ChatGPT: Settings → Connected apps → paste URL               │
│                                                                 │
│  Tech details (collapsed by default):                           │
│    - 8 MCP tools with annotations                               │
│    - OAuth 2.0 + PKCE                                           │
│    - JSON-RPC 2.0, session management                           │
│    - ChatGPT rich widgets                                       │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "How it works"                                        │
│  Architecture diagram (refined from current):                   │
│                                                                 │
│  Your App ──HTTPS──▶ Edge (300+) ──signed URL──▶ R2 Store       │
│                       │                                         │
│                       ├── Auth (< 1ms at edge)                  │
│                       ├── Metadata (D1 SQLite)                  │
│                       └── Presign (Web Crypto)                  │
│                                                                 │
│  Key insight: "File bytes never touch our servers."             │
│  Presigned URLs mean your client uploads/downloads              │
│  directly to/from the global object store.                      │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "Use cases"                                           │
│  Three cards:                                                   │
│                                                                 │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ App file uploads │ │ CI/CD artifacts │ │ AI workflows    │   │
│  │                 │ │                 │ │                 │   │
│  │ User avatars,   │ │ Build outputs,  │ │ Let Claude save │   │
│  │ documents, media│ │ deploy bundles, │ │ research, code  │   │
│  │ via presigned   │ │ test reports    │ │ snippets, data  │   │
│  │ URLs from your  │ │ via API keys    │ │ via MCP         │   │
│  │ frontend        │ │ in CI           │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│                                                                 │
│  Each card has a mini code example.                             │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SECTION: "Security"                                            │
│  Clean grid:                                                    │
│                                                                 │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │ Ed25519     │ │ Scoped keys │ │ Share links  │               │
│  │ Public key  │ │ Path-prefix │ │ Time-limited │               │
│  │ auth for    │ │ restricted  │ │ signed URLs  │               │
│  │ machines    │ │ API keys    │ │ auto-expire  │               │
│  ├─────────────┤ ├─────────────┤ ├─────────────┤               │
│  │ OAuth 2.0   │ │ Rate limits │ │ Audit log   │               │
│  │ PKCE flow   │ │ Per-endpoint│ │ Every action│               │
│  │ for apps    │ │ sliding     │ │ logged      │               │
│  │             │ │ window      │ │             │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  BOTTOM CTA                                                     │
│  "Start building."                                              │
│  "One request to upload. Zero egress to serve."                 │
│  [API Reference]  [Get started]                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Section Details

#### HERO

Title: **"Storage for developers"**
Subtitle: **"Upload a file in one request. Serve it globally. Zero egress fees."**

Terminal block shows a complete upload + response. Not a multi-step tutorial — a single curl command that works right now.

CTAs: `[API Reference]` (primary) + `[Get started]` (secondary/ghost)

#### "30 SECONDS TO FIRST UPLOAD"

Three numbered steps in a horizontal row. Each step has:
- A step number (01, 02, 03)
- A title
- A single terminal command
- An arrow connecting to the next step

```
01 Get a token              02 Upload a file           03 Share it
                     →                          →
POST /auth/challenge        PUT /f/report.pdf          POST /share
POST /auth/verify           -T ./report.pdf            {"path":"report.pdf"}
→ {"token":"sk_..."}        → 201 Created              → /s/k7f2m
```

This proves the "30 seconds" claim with real API calls.

#### "WHY NOT JUST USE S3?"

This is the conversion section. A comparison table that makes our advantages obvious.

Columns: Feature | AWS S3 | Supabase Storage | Vercel Blob | **Storage**

Rows:
- **Egress cost** — $90/TB | Metered | Limited | **$0**
- **Time to first upload** — Hours (IAM, bucket, CORS) | ~10 min | ~5 min (npm) | **30 seconds**
- **SDK required** — Yes (aws-sdk) | Yes (@supabase/storage-js) | Yes (@vercel/blob) | **No**
- **Auth methods** — IAM only | Supabase Auth | Vercel token | **4 methods**
- **AI integration** — None | None | None | **MCP built-in**
- **Bucket management** — Required | Required | Hidden | **None**
- **Region selection** — Required | Pick one | Auto | **Global**
- **File addressing** — s3://bucket/key | /object/{bucket}/{path} | blob URL | **/files/{path}**

Use checkmarks (✓) and crosses (✗) where binary. Use values where quantitative.

Last row should be a "Winner" indicator with Storage highlighted.

#### "THE API"

Three side-by-side panels, each showing one primitive with both explanation and code:

**Panel 1 — Store**
```
POST /files/uploads          Initiate upload (presigned URL)
GET  /files/docs/readme.md   Download (302 → R2)
HEAD /files/docs/readme.md   Metadata only
DEL  /files/docs/readme.md   Delete
```

**Panel 2 — Organize**
```
GET  /files?prefix=docs/     List folder contents
GET  /files/search?q=readme  Search by name
POST /files/move             Move or rename
GET  /files/stats            Storage usage
```

**Panel 3 — Share**
```
POST /files/share            Create signed link
GET  /s/:token               Access shared file
                             (no auth required)
```

Each panel: icon + title + 1-sentence description + endpoint list.
Below the three panels, a single line: *"That's the whole API. [See full reference →](/api)"*

#### "WORKS WITH ANY LANGUAGE"

Keep the existing tab switcher from the current page. It's well-executed.
Four tabs: curl, JavaScript, Python, Go.
Same upload operation in each language.

#### "AI-NATIVE"

Split layout:
- Left side: capabilities list with terminal-style examples
  ```
  "Save this analysis as report.md"  → storage_write
  "What files do I have?"            → storage_list
  "Find all CSV files"               → storage_search
  "Share the report with my team"    → storage_share
  ```

- Right side: mock conversation showing Claude or ChatGPT using Storage

Below: connection instructions with two cards (Claude + ChatGPT), each showing the URL to paste.

Expandable "Under the hood" section:
- 8 MCP tools with safety annotations
- OAuth 2.0 + PKCE for third-party auth
- JSON-RPC 2.0 protocol
- ChatGPT rich UI widgets (file browser, viewer, share links)

#### "HOW IT WORKS"

Architecture diagram. Three nodes connected by arrows:

```
Your App ──HTTPS──▶ Edge Worker (300+) ──Presigned URL──▶ R2 Object Store
                         │
                    ┌────┴────┐
                    │  D1 DB  │
                    │ (SQLite)│
                    └─────────┘
```

Three callout boxes below:
1. **Auth at the edge** — Token verification happens at the nearest of 300+ locations. Sub-millisecond.
2. **Zero-proxy transfers** — File bytes go directly between your client and R2 via presigned URLs. Our Worker never touches the data.
3. **Metadata in D1** — File index, sessions, and share links in SQLite at the edge. Fast reads, strong consistency.

#### "USE CASES"

Three cards with mini code examples:

**Card 1: App File Uploads**
```javascript
// Get a presigned URL for client-side upload
const { url } = await fetch('/presign/upload', {
  method: 'POST',
  headers: { Authorization: `Bearer ${token}` },
  body: JSON.stringify({ path: `avatars/${userId}.jpg` })
}).then(r => r.json());

// Client uploads directly to object store
await fetch(url, { method: 'PUT', body: file });
```

**Card 2: CI/CD Artifacts**
```yaml
# GitHub Actions
- name: Upload build
  run: |
    curl -X PUT https://storage.liteio.dev/f/builds/$SHA/app.tar.gz \
      -H "Authorization: Bearer ${{ secrets.STORAGE_TOKEN }}" \
      -T dist/app.tar.gz
```

**Card 3: AI Workflows**
```
You:    "Save the meeting transcript as notes/2025-03-20.md"
Claude: Done! Saved notes/2025-03-20.md (4.2 KB)

You:    "Share it with the team"
Claude: Here's the link: storage.liteio.dev/s/m9x2k (expires in 24h)
```

#### "SECURITY"

Six cards in a 3×2 grid:
1. **Ed25519 Auth** — Public key challenge-response. No passwords for machines.
2. **Scoped API Keys** — Restrict keys to path prefixes. `sk_*` format, 90-day TTL.
3. **Signed Share Links** — Time-limited URLs. 60 seconds to 7 days. Auto-expire.
4. **OAuth 2.0 + PKCE** — Standard flow for third-party apps. Dynamic client registration.
5. **Rate Limiting** — Per-endpoint sliding window. Auth: 10/min. Uploads: 100/min.
6. **Audit Logging** — Every action logged with actor, resource, and timestamp.

#### BOTTOM CTA

Title: **"Start building."**
Subtitle: **"One request to upload. Zero egress to serve. Connect AI in a minute."**
CTAs: `[API Reference]` + `[Get started]`

---

## 5. What to Delete

Delete the current `src/pages/developers.ts` entirely. It has:
- An endpoint-listing approach that reads like API docs (we have `/api` for that)
- Metrics bar that's nice but doesn't tell a story
- "Three primitives" section that's technically correct but not compelling
- Architecture diagram that's too abstract
- Feature grid that's generic ("REST-native", "Edge-first" — could be any service)

The new page should be a fresh file. Keep the nav structure, fonts, and theme toggle.

---

## 6. What to Keep / Reuse

- **Language tab switcher** — The curl/JS/Python/Go section is well done. Reuse the interaction pattern.
- **Terminal styling** — The `<pre>` blocks with syntax coloring classes (`.t-cmd`, `.t-flag`, `.t-str`, etc.) are good. Reuse.
- **Theme toggle** — Keep the dark/light toggle with localStorage persistence.
- **Nav structure** — Same nav links: developers, api, cli, ai, pricing.
- **Scroll reveal** — The IntersectionObserver animation pattern works. Reuse.

---

## 7. Content Hierarchy (Mobile)

On mobile, sections stack vertically. The comparison table scrolls horizontally.
The three-panel API section stacks to single column.
Language tabs remain as tabs (not accordions).

Priority order for mobile (what they see first):
1. Hero + terminal example
2. 30-second quickstart
3. Comparison table (horizontal scroll)
4. AI-native section
5. Language examples
6. The API details
7. Use cases
8. Architecture
9. Security
10. CTA

---

## 8. Metrics to Display

Replace the old metrics bar with contextual numbers woven into sections:

| Metric | Where it appears |
|--------|-----------------|
| **300+ edge locations** | Architecture section |
| **<50ms global latency** | Architecture section |
| **$0 egress** | Hero subtitle + comparison table |
| **30 seconds to first upload** | Quickstart section title |
| **8 MCP tools** | AI-native section |
| **4 auth methods** | Comparison table + Security section |
| **0 SDKs required** | Hero + comparison table |

Don't put them in a bar. Let them live where they prove a point.

---

## 9. Implementation Notes

### File structure
- Delete: `src/pages/developers.ts` (current file)
- Create: `src/pages/developers.ts` (new file, same export signature)
- Create: `public/developers.css` (if not already serving from a separate file — check current CSS setup)
- No changes to routing in `src/index.ts`

### Export signature
```typescript
export function developersPage(actor: string | null = null): string
```
Same function signature. Same route. New content.

### CSS approach
- Self-contained CSS (no Tailwind, no external frameworks)
- CSS custom properties for theming (already in use across pages)
- Grid and flexbox layout
- Responsive breakpoints: 640px, 768px, 1024px, 1280px

### Accessibility
- Semantic HTML: `<section>`, `<nav>`, `<main>`, `<table>`
- `aria-label` on interactive elements
- Keyboard-navigable tab switcher
- Color contrast ratios ≥ 4.5:1 (WCAG AA)
- Comparison table uses `<th scope="col">` and `<th scope="row">`

### Performance
- No JavaScript frameworks
- Inline critical CSS or single stylesheet
- SVG icons inline (no icon font)
- No images to load (all rendered in HTML/CSS)
- Intersection Observer for scroll animations (lazy)

---

## 10. Success Criteria

The new page succeeds if:

1. A developer understands what Storage does in **5 seconds** (hero)
2. They can see a working code example in **10 seconds** (terminal block)
3. They understand why to choose Storage over S3 in **30 seconds** (comparison table)
4. They know how to get started in **60 seconds** (quickstart section)
5. They discover the AI integration naturally (it's woven in, not buried)
6. They click through to `/api` or sign up — the page has exactly two goals

The page does NOT need to:
- Document every endpoint (that's `/api`)
- Explain MCP protocol details (that's `/ai`)
- Show pricing tiers (that's `/pricing`)
- Teach authentication flows (that's the API docs)

It needs to make developers think: *"This is simpler than what I'm using. Let me try it."*
