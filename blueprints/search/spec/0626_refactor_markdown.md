# Markdown Worker — Architecture Refactor Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace embedded HTML/CSS strings in TypeScript with Workers Static Assets (real files), extract shared CSS into one `styles.css`, move docs to `docs.md` with server-side rendering and in-memory cache, and add in-memory URL conversion cache to avoid re-running Workers AI for the same URL.

**Architecture:**
```
src/
  index.ts          # Worker: dynamic routes only
  convert.ts        # + in-memory URL result cache (Map, 5-min TTL)
  docs.ts           # Thin shell: wraps pre-rendered HTML in sidebar layout
  content/
    docs.md         # Documentation source — imported as text module
public/
  styles.css        # Shared CSS — all pages link here (no duplication)
  index.html        # Landing page (was page.ts — real HTML file)
  preview.html      # Preview page (was preview.ts — real HTML file)
```

Worker handles: `POST /convert`, `GET /docs`, `GET /llms.txt`, `GET /https://...`
Workers Assets handles: `GET /` (index.html), `GET /preview` (preview.html), `GET /styles.css`

**Tech Stack:** Hono, Cloudflare Workers Static Assets, TypeScript, bundled `marked` (npm, not CDN), Geist (Google Fonts CDN), marked.js + DOMPurify (CDN, client-side only)

---

## Task 1: Add Workers Static Assets + shared `styles.css`

Extract all shared CSS from `page.ts`, `preview.ts`, `docs.ts` into a single `public/styles.css`, convert `page.ts` → `public/index.html` and `preview.ts` → `public/preview.html`, update wrangler.toml, and add assets fallback to `index.ts`.

**Files:**
- Create: `blueprints/search/tools/markdown/public/styles.css`
- Create: `blueprints/search/tools/markdown/public/index.html`
- Create: `blueprints/search/tools/markdown/public/preview.html`
- Modify: `blueprints/search/tools/markdown/wrangler.toml`
- Modify: `blueprints/search/tools/markdown/src/index.ts`
- Delete: `blueprints/search/tools/markdown/src/page.ts`
- Delete: `blueprints/search/tools/markdown/src/preview.ts`

**Step 1: Update `wrangler.toml` to add assets binding**

Add to the end of `wrangler.toml`:
```toml
[assets]
directory = "./public"
binding = "ASSETS"
```

Full updated file:
```toml
name = "markdown"
main = "src/index.ts"
compatibility_date = "2024-12-01"
compatibility_flags = ["nodejs_compat"]

[observability]
enabled = true

[ai]
binding = "AI"

[browser]
binding = "BROWSER"

[assets]
directory = "./public"
binding = "ASSETS"
```

**Step 2: Create `public/styles.css` — shared CSS for all pages**

This is a single CSS file containing all design tokens and component styles shared across index.html, preview.html, and the docs page shell. Extract the common CSS from the current page.ts and preview.ts into this file.

Key contents:
```css
/* Design tokens */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
:root {
  --sans: 'Geist', -apple-system, sans-serif;
  --mono: 'Geist Mono', ui-monospace, monospace;
  --bg: #fff; --fg: #0a0a0a; --fg2: #555; --fg3: #999;
  --border: #e8e8e8; --border2: #f0f0f0;
  --code-bg: #111; --code-fg: #e4e4e7;
  --w: 1120px;
}
html { font-size: 16px; }
body { font-family: var(--sans); background: var(--bg); color: var(--fg); line-height: 1.6; -webkit-font-smoothing: antialiased; }
a { color: inherit; text-decoration: none; }
code { font-family: var(--mono); font-size: 12px; background: #f5f5f5; padding: 2px 6px; }

/* Header — full width, no border by default */
header { padding: 14px 32px; display: flex; align-items: center; justify-content: space-between; }
.logo { font-size: 14px; font-weight: 500; letter-spacing: -.02em; display: flex; align-items: center; gap: 8px; }
.logo-sq { width: 22px; height: 22px; background: var(--fg); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
nav a { font-size: 13px; color: var(--fg3); margin-left: 20px; transition: color .15s; }
nav a:hover { color: var(--fg); }
hr.sep { border: none; border-top: 1px solid var(--border2); }

/* Layout container */
.w { max-width: var(--w); margin: 0 auto; padding: 0 32px; }

/* Buttons */
.cvt-btn { font-family: var(--sans); font-size: 14px; font-weight: 500; padding: 11px 22px; background: var(--fg); color: #fff; border: none; cursor: pointer; white-space: nowrap; display: flex; align-items: center; gap: 6px; transition: background .15s; }
.cvt-btn:hover { background: #333; }
.cvt-btn:disabled { opacity: .5; cursor: default; }
.url-in { flex: 1; font-family: var(--mono); font-size: 14px; padding: 11px 16px; border: 1px solid var(--border); border-right: none; background: #fff; color: var(--fg); outline: none; transition: border-color .15s; min-width: 0; }
.url-in::placeholder { color: var(--fg3); }
.url-in:focus { border-color: var(--fg); }

/* Badges */
.badge { font-family: var(--mono); font-size: 11px; padding: 2px 8px; flex-shrink: 0; }
.b-native { background: #f3f0ff; color: #5b21b6; }
.b-ai { background: #eff6ff; color: #1d4ed8; }
.b-browser { background: #fffbeb; color: #92400e; }
.b-dim { background: #f5f5f5; color: var(--fg2); }

/* Tabs */
.tab { font-size: 14px; padding: 12px 0; margin-right: 24px; margin-bottom: -1px; background: none; border: none; border-bottom: 2px solid transparent; cursor: pointer; color: var(--fg3); transition: color .15s, border-color .15s; }
.tab.on { color: var(--fg); border-bottom-color: var(--fg); }
.panel { display: none; }
.panel.on { display: block; }
.tact { font-size: 13px; padding: 6px 12px; background: none; border: 1px solid transparent; cursor: pointer; color: var(--fg3); display: flex; align-items: center; gap: 5px; transition: all .15s; font-family: var(--sans); }
.tact:hover { color: var(--fg); border-color: var(--border); background: #fafafa; }

/* Code tabs (in sections) */
.ctabs { border: 1px solid var(--border); }
.cbar { display: flex; background: #fafafa; border-bottom: 1px solid var(--border); padding: 0 4px; }
.ctab { font-family: var(--mono); font-size: 12px; padding: 9px 14px; background: none; border: none; border-bottom: 2px solid transparent; margin-bottom: -1px; cursor: pointer; color: var(--fg3); transition: color .15s, border-color .15s; }
.ctab.on { color: var(--fg); border-bottom-color: var(--fg); }
.cpanel { display: none; background: var(--code-bg); padding: 22px; overflow-x: auto; position: relative; }
.cpanel.on { display: block; }
.cpanel pre { font-family: var(--mono); font-size: 13.5px; line-height: 1.7; color: var(--code-fg); white-space: pre; margin: 0; }
.cbcopy { position: absolute; top: 12px; right: 12px; background: #222; border: 1px solid #333; color: #aaa; font-family: var(--mono); font-size: 11px; padding: 4px 10px; cursor: pointer; transition: all .15s; }
.cbcopy:hover { background: #333; color: #fff; }

/* Syntax highlight spans */
.c1 { color: #6b7280; } /* comment */
.c2 { color: #93c5fd; } /* keyword */
.c3 { color: #86efac; } /* string */
.c4 { color: #fcd34d; } /* function */

/* Grid trick (1px gap = border effect) */
.grid3 { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--border2); border: 1px solid var(--border2); }
.grid3-card { background: #fff; padding: 28px; }
@media (max-width: 640px) { .grid3 { grid-template-columns: 1fr; } }

/* Section labels */
.sec-lbl { font-family: var(--mono); font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--fg3); margin-bottom: 20px; }
.sec h2 { font-size: 32px; font-weight: 600; letter-spacing: -.03em; margin-bottom: 14px; }
.sec-sub { font-size: 17px; color: var(--fg2); margin-bottom: 40px; max-width: 540px; line-height: 1.6; }

/* Spinner */
@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin .7s linear infinite; }

/* Error box */
.err-box { background: #fff5f5; border: 1px solid #fecaca; padding: 11px 15px; font-size: 14px; color: #dc2626; }

/* Markdown preview prose */
.prose h1 { font-size: 1.6em; font-weight: 600; letter-spacing: -.02em; margin: 0 0 14px; }
.prose h2 { font-size: 1.3em; font-weight: 600; letter-spacing: -.01em; margin: 22px 0 8px; }
.prose h3 { font-size: 1.1em; font-weight: 600; margin: 16px 0 6px; }
.prose p { margin: 0 0 14px; color: #333; line-height: 1.75; }
.prose ul, .prose ol { padding-left: 1.5em; margin: 0 0 14px; }
.prose li { margin: 3px 0; color: #333; }
.prose a { color: var(--fg); text-decoration: underline; text-underline-offset: 2px; }
.prose blockquote { border-left: 2px solid #ddd; padding-left: 16px; color: #777; margin: 12px 0; }
.prose code { background: #f5f5f5; padding: 2px 6px; font-size: .875em; }
.prose pre { background: var(--code-bg); padding: 16px; overflow-x: auto; margin: 12px 0; }
.prose pre code { background: none; color: var(--code-fg); padding: 0; font-size: .875em; }
.prose table { border-collapse: collapse; width: 100%; margin: 12px 0; }
.prose th, .prose td { border: 1px solid var(--border); padding: 8px 14px; }
.prose th { background: #fafafa; font-weight: 600; }
.prose hr { border: none; border-top: 1px solid var(--border2); margin: 18px 0; }
```

**Step 3: Create `public/index.html` — landing page**

This is the content of the current `page.ts` `renderPage()` function — but as a real HTML file. The page links to `/styles.css` instead of embedding CSS. Remove all CSS from the `<style>` block (it's now in styles.css). Keep only page-specific CSS in a small `<style>` block (e.g., hero layout, step badges, CTA grid, agent button, pipeline cards, API endpoint cards).

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>URL → Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  /* Page-specific: hero, steps, CTA grid, agent button, pipeline, API ref */
  .hero { padding: 80px 0 72px; }
  h1 { font-size: clamp(40px,6vw,68px); font-weight: 700; letter-spacing: -.04em; line-height: 1.05; margin-bottom: 32px; white-space: pre-line; }
  .steps { margin-bottom: 48px; }
  .step { display: flex; align-items: flex-start; gap: 16px; margin-bottom: 16px; font-size: 18px; color: var(--fg2); }
  .step-num { width: 22px; height: 22px; background: var(--fg); color: #fff; font-size: 12px; font-weight: 700; display: flex; align-items: center; justify-content: center; flex-shrink: 0; margin-top: 2px; }
  .cta-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 48px; align-items: start; }
  @media(max-width:680px) { .cta-grid { grid-template-columns: 1fr; } }
  .cta-lbl { font-family: var(--mono); font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--fg3); margin-bottom: 12px; }
  .input-row { display: flex; }
  .examples { margin-top: 10px; font-size: 13px; color: var(--fg3); display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .eg { color: var(--fg2); cursor: pointer; text-decoration: underline; text-decoration-color: var(--border); transition: color .15s; }
  .eg:hover { color: var(--fg); }
  .agent-btn { width: 100%; font-family: var(--sans); font-size: 16px; font-weight: 500; padding: 16px 22px; background: var(--fg); color: #fff; border: none; cursor: pointer; display: flex; align-items: center; justify-content: space-between; transition: background .15s; }
  .agent-btn:hover { background: #333; }
  .sec { padding: 72px 0; }
  .pn { font-family: var(--mono); font-size: 11px; color: var(--fg3); margin-bottom: 10px; }
  .pt { font-size: 16px; font-weight: 600; margin-bottom: 6px; }
  .pd { font-size: 15px; color: var(--fg2); line-height: 1.6; }
  .ptag { display: inline-block; margin-top: 10px; font-family: var(--mono); font-size: 11px; padding: 2px 7px; }
  .ep { border: 1px solid var(--border2); margin-bottom: 10px; }
  .eph { padding: 12px 16px; display: flex; align-items: center; gap: 10px; background: #fafafa; border-bottom: 1px solid var(--border2); }
  .mtag { font-family: var(--mono); font-size: 11px; font-weight: 500; padding: 2px 8px; flex-shrink: 0; }
  .get { background: #f0fdf4; color: #15803d; }
  .post { background: #eff6ff; color: #1d4ed8; }
  .epath { font-family: var(--mono); font-size: 14px; }
  .edesc { font-size: 12px; color: var(--fg3); margin-left: auto; }
  .ep-code { background: var(--code-bg); padding: 18px 20px; overflow-x: auto; position: relative; }
  .ep-code pre { font-family: var(--mono); font-size: 13px; line-height: 1.8; color: var(--code-fg); white-space: pre; margin: 0; }
  .rk { color: #93c5fd; } .rv { color: #e4e4e7; } .rc { color: #6b7280; }
  </style>
</head>
<body>
  <!-- Copy the full <header>, <main> content from current page.ts renderPage() output -->
  <!-- The only change: remove the embedded <style> block (it's now /styles.css) -->
  <!-- Keep the inline <script> block at the bottom unchanged -->
</body>
</html>
```

The full HTML body is identical to the current `renderPage()` output. The only differences are:
1. `<link rel="stylesheet" href="/styles.css">` replaces the bulk of the `<style>` block
2. A small page-specific `<style>` block remains for layout-only rules not in styles.css

**Step 4: Create `public/preview.html`**

Same process as index.html. Content is identical to current `renderPreview()` output but with:
1. `<link rel="stylesheet" href="/styles.css">` replaces the shared CSS
2. Small `<style>` block for preview-specific rules (url-bar, loading/error/result states, meta-bar, tab-bar, markdown/preview panels)

**Step 5: Update `src/index.ts` — add Env.ASSETS + assets fallback**

Add `ASSETS: Fetcher` to the `Env` type:
```typescript
type Env = { AI: any; BROWSER: Fetcher; ASSETS: Fetcher };
```

Add assets fallback at the END of the Hono app (after all other routes, before `export default app`):
```typescript
// Fall through to static assets (public/ directory)
app.get('*', (c) => c.env.ASSETS.fetch(c.req.raw));
```

**Step 6: Delete `src/page.ts` and `src/preview.ts`**

Remove the imports from `index.ts`:
```typescript
// DELETE these lines:
import { renderPage } from './page';
import { renderPreview } from './preview';
// DELETE these routes:
app.get('/', (c) => c.html(renderPage()));
app.get('/preview', (c) => c.html(renderPreview()));
```

The landing page and preview page are now served directly from Workers Assets.

**Step 7: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```
Expected: no errors.

**Step 8: Commit**

```bash
git add public/ src/index.ts wrangler.toml
git rm src/page.ts src/preview.ts
git commit -m "refactor(markdown): Workers Static Assets — real HTML files, shared styles.css"
```

---

## Task 2: `docs.md` source + server-side render with in-memory cache

Import `docs.md` as a text module, render it once with bundled `marked` (npm), cache the result in a module-level variable. The `/docs` route serves a thin HTML shell wrapping the server-rendered content.

**Files:**
- Install: `marked` npm package
- Create: `blueprints/search/tools/markdown/src/content/docs.md`
- Modify: `blueprints/search/tools/markdown/wrangler.toml` — add text module rule
- Modify: `blueprints/search/tools/markdown/tsconfig.json` — add `.md` type declaration
- Modify: `blueprints/search/tools/markdown/src/docs.ts` — accept pre-rendered HTML param
- Modify: `blueprints/search/tools/markdown/src/index.ts` — render docs.md, pass to renderDocs()
- Delete: `blueprints/search/tools/markdown/src/docs.ts` (rewrite from scratch, keep only shell)

**Step 1: Install `marked` as a bundled dependency**

```bash
cd blueprints/search/tools/markdown && npm install marked
```

**Step 2: Add text module rule to `wrangler.toml`**

```toml
[[rules]]
type = "Text"
globs = ["**/*.md"]
```

This tells wrangler to import `.md` files as raw text strings.

**Step 3: Add TypeScript declaration for `.md` imports**

Create `blueprints/search/tools/markdown/src/env.d.ts`:
```typescript
declare module '*.md' {
  const content: string;
  export default content;
}
```

**Step 4: Create `src/content/docs.md`**

Write the documentation as clean Markdown. Content mirrors the current `docs.ts` sections:

```markdown
# URL → Markdown

Free, instant URL-to-Markdown conversion for AI agents and LLM pipelines. No API key, no account.

## Overview

Convert any HTTP/HTTPS URL to clean, structured Markdown with a single request.

- Works with any HTTP/HTTPS URL
- Three-tier pipeline: native negotiation → Workers AI → Browser rendering
- Edge-cached for 1 hour with stale-while-revalidate
- CORS-enabled for browser and agent use

## Quick start

### Fetch as Markdown

    curl https://markdown.go-mizu.workers.dev/https://example.com

### Use the JSON API

    curl -X POST https://markdown.go-mizu.workers.dev/convert \
      -H 'Content-Type: application/json' \
      -d '{"url":"https://example.com"}'

### JavaScript

    const md = await fetch(
      'https://markdown.go-mizu.workers.dev/' + url
    ).then(r => r.text());

### Python

    import httpx
    md = httpx.get('https://markdown.go-mizu.workers.dev/' + url).text

## API reference

### GET /{url}

Convert a URL to Markdown. Append any `http://` or `https://` URL to the worker base URL. Query strings are preserved.

    curl https://markdown.go-mizu.workers.dev/https://example.com?q=hello

### POST /convert

Convert a URL and receive a structured JSON response.

Request body: `{"url": "https://example.com"}`

Response:
    {
      "markdown": "# Example Domain\n\n...",
      "method": "primary" | "ai" | "browser",
      "durationMs": 342,
      "title": "Example Domain",
      "tokens": 1248
    }

### GET /llms.txt

Machine-readable API summary for LLM agents.

## Pipeline

Every URL goes through up to three tiers, falling back automatically:

- **Tier 1 — Native Markdown:** Requests with `Accept: text/markdown`. Sites that support this return structured Markdown directly.
- **Tier 2 — Workers AI:** Fetches HTML and converts via Cloudflare Workers AI `toMarkdown()`.
- **Tier 3 — Browser Render:** For JS-heavy SPAs. Renders in headless browser via `@cloudflare/puppeteer`, then passes to Workers AI.

## Response headers

The `GET /{url}` endpoint returns these headers:

| Header | Description |
|---|---|
| X-Conversion-Method | `primary`, `ai`, or `browser` |
| X-Duration-Ms | Server-side processing time in milliseconds |
| X-Title | Percent-encoded page title (max 200 chars) |
| X-Markdown-Tokens | Approximate token count (when available) |
| Cache-Control | `public, max-age=300, s-maxage=3600, stale-while-revalidate=86400` |

## Limits

- Max response body: **5 MB** per URL
- Fetch timeout: **10 seconds** (30 seconds for browser rendering)
- Protocols: **http://** and **https://** only
- Rate limits: Cloudflare Workers free tier (100,000 requests/day)
```

**Step 5: Rewrite `src/docs.ts` — thin HTML shell only**

The docs page is now a thin shell that accepts pre-rendered HTML content:

```typescript
export function renderDocs(contentHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs — URL → Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  /* Docs-specific layout */
  header { border-bottom: 1px solid var(--border2); }
  .layout { display: flex; min-height: calc(100vh - 52px); }
  .sidebar { width: 240px; flex-shrink: 0; padding: 32px 24px; position: sticky; top: 0; height: calc(100vh - 52px); overflow-y: auto; border-right: 1px solid var(--border2); }
  .sidebar-title { font-family: var(--mono); font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--fg3); margin-bottom: 16px; }
  .sidebar a { display: block; font-size: 14px; color: var(--fg3); padding: 4px 0; transition: color .15s; }
  .sidebar a:hover { color: var(--fg); }
  .sidebar a.on { color: var(--fg); font-weight: 500; }
  .content { flex: 1; padding: 48px 64px; max-width: 900px; }
  @media(max-width:720px) { .sidebar { display: none; } .content { padding: 32px 24px; } }
  /* Rendered markdown content styles */
  .content h1 { font-size: 36px; font-weight: 700; letter-spacing: -.03em; margin-bottom: 16px; }
  .content h2 { font-size: 22px; font-weight: 600; letter-spacing: -.02em; margin: 48px 0 14px; scroll-margin-top: 24px; }
  .content h3 { font-size: 17px; font-weight: 600; margin: 28px 0 10px; }
  .content p { font-size: 16px; color: var(--fg2); line-height: 1.75; margin-bottom: 16px; }
  .content ul { padding-left: 1.5em; margin-bottom: 16px; }
  .content li { font-size: 16px; color: var(--fg2); margin: 4px 0; line-height: 1.7; }
  .content strong { color: var(--fg); font-weight: 600; }
  .content pre { background: var(--code-bg); padding: 20px 22px; overflow-x: auto; margin: 20px 0; position: relative; }
  .content pre code { font-family: var(--mono); font-size: 13.5px; line-height: 1.7; color: var(--code-fg); background: none; padding: 0; }
  .content table { width: 100%; border-collapse: collapse; margin: 16px 0; font-size: 14px; }
  .content th { text-align: left; padding: 8px 14px; border-bottom: 2px solid var(--border); font-weight: 600; }
  .content td { padding: 8px 14px; border-bottom: 1px solid var(--border2); color: var(--fg2); }
  .content td:first-child { font-family: var(--mono); font-size: 12.5px; color: var(--fg); }
  /* Copy buttons on pre blocks */
  .content pre .copy-btn { position: absolute; top: 10px; right: 10px; background: #222; border: 1px solid #333; color: #aaa; font-family: var(--mono); font-size: 11px; padding: 4px 10px; cursor: pointer; transition: all .15s; }
  .content pre .copy-btn:hover { background: #333; color: #fff; }
  </style>
</head>
<body>
<header>
  <a href="/" class="logo">
    <span class="logo-sq">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
      </svg>
    </span>
    markdown.go-mizu
  </a>
  <nav>
    <a href="/">Home</a>
    <a href="/llms.txt">llms.txt</a>
    <a href="https://github.com/go-mizu/mizu">GitHub</a>
  </nav>
</header>

<div class="layout">
  <nav class="sidebar" id="sidebar">
    <div class="sidebar-title">Docs</div>
    <!-- JS builds nav from h2 headings -->
  </nav>
  <div class="content" id="content">
    ${contentHtml}
  </div>
</div>

<script>
// Build sidebar from h2 headings in rendered content
(function() {
  var headings = document.querySelectorAll('.content h2');
  var sidebar = document.getElementById('sidebar');
  headings.forEach(function(h) {
    if (!h.id) {
      h.id = h.textContent.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    }
    var a = document.createElement('a');
    a.href = '#' + h.id;
    a.textContent = h.textContent;
    sidebar.appendChild(a);
  });

  // Add copy buttons to pre blocks
  document.querySelectorAll('.content pre').forEach(function(pre) {
    var btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'copy';
    btn.onclick = function() {
      var text = (pre.querySelector('code') || pre).innerText || '';
      navigator.clipboard.writeText(text.trim()).catch(function() {
        var ta = document.createElement('textarea');
        ta.style.position = 'fixed'; ta.style.top = '-9999px';
        ta.value = text.trim();
        document.body.appendChild(ta); ta.select();
        document.execCommand('copy'); document.body.removeChild(ta);
      });
      btn.textContent = 'copied!';
      setTimeout(function() { btn.textContent = 'copy'; }, 2000);
    };
    pre.style.position = 'relative';
    pre.appendChild(btn);
  });

  // Scroll-spy
  var links = [];
  document.querySelectorAll('.sidebar a[href^="#"]').forEach(function(l) { links.push(l); });
  var sections = document.querySelectorAll('.content h2[id]');
  window.addEventListener('scroll', function() {
    var pos = window.scrollY + 80;
    var active = null;
    sections.forEach(function(s) { if (s.offsetTop <= pos) active = s; });
    links.forEach(function(l) {
      l.className = (active && l.getAttribute('href') === '#' + active.id) ? 'on' : '';
    });
  }, { passive: true });
})();
</script>
</body>
</html>`;
}
```

Note: `${contentHtml}` is a real template literal interpolation here — this is the TypeScript function body, not embedded HTML content. This is valid.

**Step 6: Update `src/index.ts` — render docs.md once, cache in module-level variable**

Add these imports at the top of `index.ts`:
```typescript
import { marked } from 'marked';
import docsMarkdown from './content/docs.md';
import { renderDocs } from './docs';
```

Add the module-level cache variable:
```typescript
// Cached once per worker isolate lifetime
let cachedDocsHtml: string | null = null;
```

Update the `/docs` route:
```typescript
app.get('/docs', (c) => {
  if (!cachedDocsHtml) {
    const rendered = marked.parse(docsMarkdown);
    cachedDocsHtml = typeof rendered === 'string' ? rendered : '';
  }
  return c.html(renderDocs(cachedDocsHtml));
});
```

**Step 7: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```
Expected: no errors. The `.md` import should resolve cleanly via `env.d.ts`.

**Step 8: Commit**

```bash
git add src/content/docs.md src/docs.ts src/index.ts src/env.d.ts wrangler.toml package.json package-lock.json
git commit -m "feat(markdown): docs.md server-side render with in-memory cache"
```

---

## Task 3: In-memory URL conversion cache in `convert.ts`

Add a module-level `Map` to cache conversion results within a worker isolate (avoids re-running Workers AI for the same URL within 5 minutes).

**Files:**
- Modify: `blueprints/search/tools/markdown/src/convert.ts`

**Background:** Cloudflare Workers isolates are reused across many requests before being recycled. A module-level `Map` persists across all requests handled by the same isolate. For popular URLs this gives a significant latency improvement — the Workers AI tier (the slow one) is skipped entirely on cache hits.

**Step 1: Add cache to `convert.ts`**

Add at the top of the file (after imports):
```typescript
interface CachedResult {
  result: ConversionResult;
  cachedAt: number; // Date.now()
}

const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes — matches HTTP max-age=300

const resultCache = new Map<string, CachedResult>();
```

**Step 2: Add cache lookup + store to the `convert()` function**

At the start of `convert()`, before the tier logic:
```typescript
// Check in-memory cache
const cached = resultCache.get(url);
if (cached && Date.now() - cached.cachedAt < CACHE_TTL_MS) {
  return { ...cached.result, durationMs: 0 }; // durationMs 0 = cache hit
}
```

At the end of `convert()`, just before each `return` statement, store the result. The cleanest way is to add a single store point:

```typescript
export async function convert(url: string, env: Env): Promise<ConversionResult> {
  const start = Date.now();

  // Check cache
  const cached = resultCache.get(url);
  if (cached && Date.now() - cached.cachedAt < CACHE_TTL_MS) {
    return { ...cached.result, durationMs: 0 };
  }

  // ... existing tier logic unchanged ...

  // At the final return point, wrap to store result:
  const result: ConversionResult = { ... };
  resultCache.set(url, { result, cachedAt: Date.now() });
  return result;
}
```

Since the function has multiple return points (one per tier), the cleanest refactor is to collect the result at the end of the try/catch and store it once. Restructure the function to:

```typescript
export async function convert(url: string, env: Env): Promise<ConversionResult> {
  const start = Date.now();

  // Validate URL
  const parsed = new URL(url);
  if (!['http:', 'https:'].includes(parsed.protocol)) {
    throw new Error('Only http and https URLs are supported');
  }

  // Check in-memory cache
  const cached = resultCache.get(url);
  if (cached && Date.now() - cached.cachedAt < CACHE_TTL_MS) {
    return { ...cached.result, durationMs: 0 };
  }

  const result = await doConvert(url, env, start);

  // Store in cache (don't cache failed conversions — only store if we have real content)
  if (result.markdown && result.markdown !== 'Unable to retrieve page content.') {
    resultCache.set(url, { result, cachedAt: Date.now() });
  }

  return result;
}

// Internal: actual conversion logic (extracted from convert())
async function doConvert(url: string, env: Env, start: number): Promise<ConversionResult> {
  // Tier 1: Native Markdown negotiation
  const nativeResult = await tryNativeMarkdown(url);
  if (nativeResult !== null) {
    return {
      markdown: nativeResult,
      method: 'primary',
      durationMs: Date.now() - start,
      title: extractTitleFromMarkdown(nativeResult),
      sourceUrl: url,
    };
  }

  // Tier 2: Workers AI
  const html = await fetchHTML(url);
  if (html !== null) {
    const aiResult = await tryWorkersAI(html, env).catch(() => null);
    if (aiResult !== null) {
      return {
        markdown: aiResult.markdown,
        method: 'ai',
        durationMs: Date.now() - start,
        title: extractTitleFromHTML(html),
        tokens: aiResult.tokens,
        sourceUrl: url,
      };
    }
  }

  // Tier 3: Browser Rendering
  let browserHtml = '';
  try {
    browserHtml = await tryBrowserRendering(url, env);
  } catch {
    // Browser unavailable — fall through
  }
  const aiFromBrowser = browserHtml ? await tryWorkersAI(browserHtml, env).catch(() => null) : null;
  const markdown = aiFromBrowser?.markdown ?? (browserHtml ? stripHtml(browserHtml) : 'Unable to retrieve page content.');

  return {
    markdown,
    method: 'browser',
    durationMs: Date.now() - start,
    title: browserHtml ? extractTitleFromHTML(browserHtml) : 'Untitled',
    tokens: aiFromBrowser?.tokens,
    sourceUrl: url,
  };
}
```

**Step 3: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```
Expected: no errors.

**Step 4: Commit**

```bash
git add src/convert.ts
git commit -m "feat(markdown): in-memory URL conversion cache (5min TTL per isolate)"
```

---

## Task 4: Deploy and verify

**Step 1: Deploy**

```bash
cd blueprints/search/tools/markdown && npm run deploy
```

**Step 2: Verify all routes**

```bash
# Landing page (served from Workers Assets)
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/
# Expected: 200

# Preview page (served from Workers Assets)
curl -s -o /dev/null -w "%{http_code}" "https://markdown.go-mizu.workers.dev/preview?url=https://example.com"
# Expected: 200

# Shared CSS served
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/styles.css
# Expected: 200

# Docs page (server-side rendered from docs.md)
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/docs
# Expected: 200

# URL conversion (should work)
curl -s https://markdown.go-mizu.workers.dev/https://example.com | head -3
# Expected: markdown content

# POST /convert
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' | grep '"method"'
# Expected: "method":"primary" or "ai"

# Cache hit test (second request should have durationMs: 0)
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' | grep durationMs
# Expected: "durationMs":0 (if same worker isolate)

# llms.txt
curl -s https://markdown.go-mizu.workers.dev/llms.txt | head -3
# Expected: # URL to Markdown API
```

**Step 3: Commit**

No new code changes — if tests pass, just note the deployment is live.

---

## Verification Checklist

- [ ] `public/styles.css` exists and all three pages (`/`, `/preview`, `/docs`) link to it
- [ ] No duplicate CSS between `index.html` and `preview.html`
- [ ] `public/index.html` and `public/preview.html` are real HTML files (not TypeScript strings)
- [ ] `src/page.ts` and `src/preview.ts` are deleted
- [ ] `src/docs.ts` is a thin shell function accepting `contentHtml: string` parameter
- [ ] `src/content/docs.md` exists with all documentation sections
- [ ] `/docs` route renders from `docs.md` (no static HTML in TypeScript, only the shell)
- [ ] Second request to `/convert` for the same URL returns `"durationMs": 0` (cache hit)
- [ ] `npx tsc --noEmit` passes with zero errors
