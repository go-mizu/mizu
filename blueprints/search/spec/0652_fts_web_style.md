# spec/0652: FTS Web GUI — Clean Brutalist Redesign

## Goal

Redesign the `search cc fts web` interface with a modern, borderless, sharp-corner
aesthetic inspired by 2026 AI tool design trends (Linear, Perplexity, Vercel Geist).
Single-file rewrite of `pkg/index/web/static/index.html`.

## Design Principles

1. **No rounded corners** — `border-radius: 0` everywhere
2. **No card borders** — content separated by spacing and thin dividers only
3. **No shadows** — completely flat, single-plane UI
4. **No decorative elements** — no badges, pills, score bars, icon-in-circle logos
5. **Typography is the design** — hierarchy through size, weight, and spacing
6. **Generous whitespace** — 24-32px vertical gaps between content blocks
7. **Content-forward** — show more actual text, less chrome

## Typography

| Element | Font | Size | Weight | Color |
|---------|------|------|--------|-------|
| Page title | Geist Sans | 32px | 600 | primary |
| Subtitle | Geist Sans | 15px | 400 | muted (zinc-400) |
| Nav links | Geist Sans | 14px | 400 | zinc-400, hover: primary |
| Result title | Geist Sans | 16px | 500 | primary |
| Snippet text | Geist Sans | 14px | 400 | zinc-400 |
| Metadata | Geist Mono | 13px | 400 | zinc-500 |
| Stats inline | Geist Mono | 13px | 400 | zinc-500 |
| Search input | Geist Sans | 16px | 400 | primary |
| Prose body | Geist Sans | 15px | 400 | primary |
| Prose code | Geist Mono | 14px | 400 | zinc-300 |

Google Fonts: `Geist:wght@400;500;600` + `Geist+Mono:wght@400;500`

## Color Palette

### Dark (default)
- bg: `#09090b` (zinc-950)
- text: `#fafafa` (zinc-50)
- secondary: `#a1a1aa` (zinc-400)
- muted: `#71717a` (zinc-500)
- divider: `#27272a` (zinc-800)
- input-bg: `#18181b` (zinc-900)
- highlight: `rgba(250, 204, 21, 0.15)` (search match)

### Light
- bg: `#ffffff`
- text: `#09090b` (zinc-950)
- secondary: `#71717a` (zinc-500)
- muted: `#a1a1aa` (zinc-400)
- divider: `#e4e4e7` (zinc-200)
- input-bg: `#f4f4f5` (zinc-100)
- highlight: `rgba(250, 204, 21, 0.25)`

## Layout

- **Max width**: 768px centered (wider than 640, good for content reading)
- **Header height**: 48px (compact)
- **Padding**: 0 24px (side gutters)
- **Result spacing**: 24px vertical gap, separated by 1px divider
- **Browse sidebar**: 180px fixed width

## Pages

### 1. Home (`/`)

```
                    [nav: FTS · Browse · ☀]

          What are you looking for?

  [_________________________________________] ⌘K

          tantivy · 12,450 docs · 3 shards · 48 MB
```

- Title centered, 32px semibold
- Input: full-width, no border, zinc-900 bg, 48px height
- Stats: single centered line, mono, muted, separated by `·`
- No stat cards — just text

### 2. Search Results (`/search?q=...`)

```
  [header with inline search input]

  42 results · 23ms · 3 shards

  Understanding Neural Networks                    4.21
  00003 · 9c4852b9-f2bb-46c8-92a2-ab8619823d9e.md
  A comprehensive guide to how neural networks
  process information and learn from data through
  backpropagation and gradient descent methods...
  ────────────────────────────────────────────────
  Deep Learning Fundamentals                       3.87
  00001 · a1b2c3d4-e5f6-7890-abcd-ef1234567890.md
  ...
```

- Meta line: count + timing + shards, muted mono
- Each result: title (16px medium) + score right-aligned (mono)
- Second line: shard + doc_id (13px mono muted)
- Third block: snippet (14px, zinc-400, 3 lines max)
- Thin divider between results (zinc-800)
- No cards, no borders, no score bars

### 3. Document Viewer (`/doc/{shard}/{docid}`)

```
  ← Back

  00003 / 9c4852b9-f2bb-46c8-92a2-ab8619823d9e.md

  ─── Rendered  Markdown ───   1,234 words · 4.2 KB

  [prose content rendered with good typography]
```

- Back link, plain text with arrow
- Breadcrumb: shard / filename in mono
- Tab toggle: text-based, active = primary color, inactive = muted
- Metadata inline: word count + file size
- Prose: Tailwind Typography plugin, good heading sizes

### 4. Browse (`/browse/{shard}`)

```
  Shards          │  00003 · 342 files
                  │
  00000   1,204   │  9c4852b9-...md                2.1 KB
  00001     987   │  a1b2c3d4-...md                1.8 KB
  00002   1,102   │  b3c4d5e6-...md                  956 B
  ▸ 00003   342   │  ...
  00004     891   │
```

- Left sidebar: plain text list, active = primary, inactive = muted
- File count right-aligned
- Main area: filename + size as simple rows
- Thin vertical divider between sidebar and content

## Copy Changes

| Old | New |
|-----|-----|
| "Search documents..." | "What are you looking for?" |
| "Full-Text Search" | "FTS" (nav) / "What are you looking for?" (home) |
| "No results found for X" | "Nothing matched «X»" |
| "Try different keywords or check your index" | "Try different terms or broaden your query" |
| "No FTS index found. Run..." | "No index yet. Run `search cc fts index` to get started." |
| "Loading shards..." | "Loading..." |
| "Rendered" / "Markdown" | "Rendered" / "Source" |

## Animations

- Subtle fade-in on page load (200ms)
- Results stagger in from top (fadeUp, 30ms delay each)
- No bouncy animations — strictly ease-out, short duration

## Keyboard Shortcuts

- `⌘K` or `/` — focus search
- `Escape` — blur search

## Implementation

Single file change: rewrite `pkg/index/web/static/index.html`.
No Go backend changes needed — same API, same endpoints.
