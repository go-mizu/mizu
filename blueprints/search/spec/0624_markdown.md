# spec/0624 — URL to Markdown Converter (markdown.go-mizu.workers.dev)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a URL-to-Markdown conversion Cloudflare Worker at `https://markdown.go-mizu.workers.dev`, cloning the functionality of markdown.new with a light-theme shadcn-inspired UI and Hono backend.

**Architecture:** Hono-based CF Worker at `tools/markdown/`. Three-tier conversion pipeline: (1) native Markdown negotiation via `Accept: text/markdown`, (2) Cloudflare Workers AI `env.AI.toMarkdown()`, (3) Browser Rendering via `@cloudflare/puppeteer`. Serves both an HTML landing page UI and a JSON/text API. Result shows which tier was used and server duration.

**Tech Stack:** TypeScript, Hono 4.x, `@cloudflare/puppeteer`, CF Workers AI binding, Wrangler 4.x. UI: Tailwind CDN + marked.js CDN (inline HTML, no build step).

---

## Three-Tier Pipeline

| Tier | Name | Trigger | Implementation |
|------|------|---------|----------------|
| 1 – Primary | Native Markdown | Site returns `Content-Type: text/markdown` | `fetch(url, { headers: { Accept: 'text/markdown' } })` |
| 2 – Workers AI | AI conversion | Site returns HTML | `env.AI.toMarkdown([{ name: 'page.html', blob }])` |
| 3 – Browser | Browser render | JS-heavy / AI fails | `puppeteer.launch(env.BROWSER)` → get HTML → AI |

Response always includes `method` (`primary` / `ai` / `browser`) and `durationMs`.

---

## Task 1: Scaffold Project

**Files:**
- Create: `tools/markdown/package.json`
- Create: `tools/markdown/wrangler.toml`
- Create: `tools/markdown/tsconfig.json`
- Create: `tools/markdown/src/index.ts` (stub)

**Step 1: Create directories**
```bash
mkdir -p tools/markdown/src
```

**Step 2: Create `tools/markdown/package.json`**
```json
{
  "name": "markdown",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "wrangler dev",
    "deploy": "wrangler deploy",
    "type-check": "tsc --noEmit"
  },
  "dependencies": {
    "hono": "^4.11.8",
    "@cloudflare/puppeteer": "^0.0.35"
  },
  "devDependencies": {
    "@cloudflare/workers-types": "^4.20260207.0",
    "wrangler": "^4.63.0",
    "typescript": "^5.9.3"
  }
}
```

**Step 3: Create `tools/markdown/wrangler.toml`**
```toml
name = "markdown"
main = "src/index.ts"
compatibility_date = "2024-12-01"
compatibility_flags = ["nodejs_compat"]

[observability]
enabled = true

[ai]
binding = "AI"

[[browser]]
binding = "BROWSER"
```

**Step 4: Create `tools/markdown/tsconfig.json`**
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ES2022",
    "moduleResolution": "bundler",
    "lib": ["ES2022"],
    "types": ["@cloudflare/workers-types"],
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true
  },
  "include": ["src/**/*.ts"]
}
```

**Step 5: Create stub `tools/markdown/src/index.ts`**
```typescript
import { Hono } from 'hono';

const app = new Hono();
app.get('/', (c) => c.text('ok'));
export default app;
```

**Step 6: Install dependencies**
```bash
cd tools/markdown && npm install
```

**Step 7: Commit**
```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add tools/markdown/
git commit -m "feat(markdown): scaffold CF Worker project"
```

---

## Task 2: Three-Tier Conversion Logic

**Files:**
- Create: `tools/markdown/src/convert.ts`

**Step 1: Create `tools/markdown/src/convert.ts`**

```typescript
import puppeteer from '@cloudflare/puppeteer';

export type ConversionMethod = 'primary' | 'ai' | 'browser';

export interface ConversionResult {
  markdown: string;
  method: ConversionMethod;
  durationMs: number;
  title: string;
  tokens?: number;
  sourceUrl: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AI: any; BROWSER: Fetcher };

const UA = 'Mozilla/5.0 (compatible; go-mizu-markdown/1.0; +https://markdown.go-mizu.workers.dev)';
const FETCH_TIMEOUT_MS = 10_000;
const BROWSER_TIMEOUT_MS = 20_000;

export async function convert(url: string, env: Env): Promise<ConversionResult> {
  const start = Date.now();

  // Validate URL
  const parsed = new URL(url); // throws TypeError if invalid
  if (!['http:', 'https:'].includes(parsed.protocol)) {
    throw new Error('Only http and https URLs are supported');
  }

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

  // Fetch HTML (shared between tiers 2 and 3)
  const html = await fetchHTML(url);

  if (html !== null) {
    // Tier 2: Workers AI toMarkdown
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

  // Tier 3: Browser Rendering via Puppeteer → AI
  const browserHtml = await tryBrowserRendering(url, env);
  const aiFromBrowser = await tryWorkersAI(browserHtml, env).catch(() => null);
  const markdown = aiFromBrowser?.markdown ?? stripHtml(browserHtml);

  return {
    markdown,
    method: 'browser',
    durationMs: Date.now() - start,
    title: extractTitleFromHTML(browserHtml),
    tokens: aiFromBrowser?.tokens,
    sourceUrl: url,
  };
}

// Tier 1: Accept: text/markdown negotiation
async function tryNativeMarkdown(url: string): Promise<string | null> {
  try {
    const resp = await fetch(url, {
      headers: {
        Accept: 'text/markdown, text/x-markdown, text/plain;q=0.9, */*;q=0.1',
        'User-Agent': UA,
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
    });
    if (!resp.ok) return null;
    const ct = resp.headers.get('content-type') ?? '';
    if (ct.includes('text/markdown') || ct.includes('text/x-markdown')) {
      return await resp.text();
    }
    return null;
  } catch {
    return null;
  }
}

// Fetch raw HTML for tier 2
async function fetchHTML(url: string): Promise<string | null> {
  try {
    const resp = await fetch(url, {
      headers: {
        Accept: 'text/html,application/xhtml+xml,*/*;q=0.8',
        'User-Agent': UA,
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
    });
    if (!resp.ok) return null;
    return await resp.text();
  } catch {
    return null;
  }
}

// Tier 2: Cloudflare Workers AI toMarkdown()
async function tryWorkersAI(
  html: string,
  env: Env
): Promise<{ markdown: string; tokens: number } | null> {
  const blob = new Blob([html], { type: 'text/html' });
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const results: any[] = await env.AI.toMarkdown([{ name: 'page.html', blob }]);
  if (!results || results.length === 0) return null;
  const r = results[0];
  const markdown = (r.data ?? r.result ?? '') as string;
  if (!markdown.trim()) return null;
  return { markdown, tokens: r.tokens ?? 0 };
}

// Tier 3: Browser Rendering via @cloudflare/puppeteer
async function tryBrowserRendering(url: string, env: Env): Promise<string> {
  const browser = await puppeteer.launch(env.BROWSER);
  try {
    const page = await browser.newPage();
    await page.setUserAgent(UA);
    await page.goto(url, {
      waitUntil: 'networkidle0',
      timeout: BROWSER_TIMEOUT_MS,
    });
    return await page.content();
  } finally {
    await browser.close();
  }
}

// Extract <title> from HTML
function extractTitleFromHTML(html: string): string {
  const m = html.match(/<title[^>]*>([^<]+)<\/title>/i);
  return m ? m[1].trim() : 'Untitled';
}

// Extract first H1 from Markdown
function extractTitleFromMarkdown(md: string): string {
  const m = md.match(/^#{1,2}\s+(.+)$/m);
  return m ? m[1].trim() : 'Untitled';
}

// Last-resort HTML strip (fallback if AI unavailable)
function stripHtml(html: string): string {
  return html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s{3,}/g, '\n\n')
    .trim();
}
```

**Step 2: Commit**
```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add tools/markdown/src/convert.ts
git commit -m "feat(markdown): three-tier conversion pipeline (native → AI → browser)"
```

---

## Task 3: Landing Page HTML Template

**Files:**
- Create: `tools/markdown/src/page.ts`

This is the complete shadcn-inspired light-theme single-page app, served as an inline HTML string. Uses Tailwind CDN + marked.js CDN. No separate build step.

**Step 1: Create `tools/markdown/src/page.ts`**

```typescript
export function renderPage(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>URL → Markdown · go-mizu</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://cdn.jsdelivr.net/npm/marked@15/marked.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .prose h1 { font-size: 1.5rem; font-weight: 700; margin: 1rem 0 0.5rem; color: #111827; }
    .prose h2 { font-size: 1.25rem; font-weight: 600; margin: 0.875rem 0 0.375rem; color: #111827; }
    .prose h3 { font-size: 1.1rem; font-weight: 600; margin: 0.75rem 0 0.25rem; color: #1f2937; }
    .prose h4 { font-size: 1rem; font-weight: 600; margin: 0.625rem 0 0.25rem; color: #1f2937; }
    .prose p { margin: 0.5rem 0; line-height: 1.65; color: #374151; }
    .prose ul { list-style: disc; padding-left: 1.5rem; margin: 0.5rem 0; }
    .prose ol { list-style: decimal; padding-left: 1.5rem; margin: 0.5rem 0; }
    .prose li { margin: 0.2rem 0; color: #374151; }
    .prose a { color: #4f46e5; text-decoration: underline; }
    .prose a:hover { color: #3730a3; }
    .prose blockquote { border-left: 3px solid #e5e7eb; padding-left: 1rem; color: #6b7280; margin: 0.75rem 0; font-style: italic; }
    .prose code { background: #f3f4f6; padding: 0.125rem 0.375rem; border-radius: 0.25rem; font-family: ui-monospace, monospace; font-size: 0.85em; color: #1f2937; }
    .prose pre { background: #f3f4f6; padding: 1rem; border-radius: 0.5rem; overflow-x: auto; margin: 0.75rem 0; }
    .prose pre code { background: none; padding: 0; font-size: 0.8rem; }
    .prose hr { border: none; border-top: 1px solid #e5e7eb; margin: 1rem 0; }
    .prose table { border-collapse: collapse; width: 100%; margin: 0.75rem 0; }
    .prose th, .prose td { border: 1px solid #e5e7eb; padding: 0.375rem 0.75rem; font-size: 0.875rem; }
    .prose th { background: #f9fafb; font-weight: 600; color: #374151; }
    .prose img { max-width: 100%; border-radius: 0.375rem; }
    .tab-active { border-bottom: 2px solid #18181b; color: #18181b; font-weight: 500; }
    .tab-inactive { border-bottom: 2px solid transparent; color: #6b7280; }
    .tab-inactive:hover { color: #374151; }
    .spinner { animation: spin 0.8s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>
</head>
<body class="bg-gray-50 min-h-screen text-gray-900 antialiased">

  <!-- Header -->
  <header class="bg-white border-b border-gray-200 sticky top-0 z-10">
    <div class="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <svg class="w-5 h-5 text-zinc-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" y1="13" x2="8" y2="13"/>
          <line x1="16" y1="17" x2="8" y2="17"/>
          <polyline points="10 9 9 9 8 9"/>
        </svg>
        <span class="font-semibold text-gray-900 tracking-tight">URL → Markdown</span>
        <span class="hidden sm:inline text-xs text-gray-400 font-normal">Convert any webpage to clean Markdown</span>
      </div>
      <a href="https://github.com/go-mizu/mizu" class="text-xs text-gray-400 hover:text-gray-600 transition-colors">go-mizu</a>
    </div>
  </header>

  <!-- Main -->
  <main class="max-w-4xl mx-auto px-4 py-8 space-y-4">

    <!-- URL Input Card -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
      <form id="form" onsubmit="handleSubmit(event)">
        <div class="flex gap-2">
          <input
            id="url-input"
            type="url"
            placeholder="https://example.com"
            autocomplete="off"
            spellcheck="false"
            class="flex-1 border border-gray-300 rounded-lg px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:border-transparent transition"
          />
          <button
            type="submit"
            id="submit-btn"
            class="bg-zinc-900 text-white px-5 py-2.5 rounded-lg text-sm font-medium hover:bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:ring-offset-2 transition-colors flex items-center gap-2 whitespace-nowrap"
          >
            <span id="btn-text">Convert</span>
            <svg id="btn-spinner" class="hidden spinner w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
            </svg>
          </button>
        </div>
        <div class="mt-2.5 flex items-center gap-1.5 flex-wrap">
          <span class="text-xs text-gray-400">Try:</span>
          <button type="button" onclick="setExample('https://example.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">example.com</button>
          <span class="text-gray-300 text-xs">·</span>
          <button type="button" onclick="setExample('https://news.ycombinator.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">news.ycombinator.com</button>
          <span class="text-gray-300 text-xs">·</span>
          <button type="button" onclick="setExample('https://blog.cloudflare.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">blog.cloudflare.com</button>
        </div>
      </form>
    </div>

    <!-- Error Card -->
    <div id="error-card" class="hidden bg-red-50 border border-red-200 rounded-xl p-4">
      <div class="flex items-start gap-3">
        <svg class="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <p id="error-msg" class="text-sm text-red-700"></p>
      </div>
    </div>

    <!-- Result Card -->
    <div id="result-card" class="hidden bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">

      <!-- Meta bar -->
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center gap-2 flex-wrap min-w-0">
        <svg class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
        <span id="result-title" class="text-sm font-medium text-gray-900 truncate flex-1 min-w-0"></span>
        <span id="method-badge" class="text-xs px-2.5 py-1 rounded-full font-medium flex-shrink-0 border"></span>
        <span id="duration-badge" class="text-xs px-2.5 py-1 rounded-full bg-gray-100 text-gray-600 font-mono flex-shrink-0"></span>
        <span id="tokens-badge" class="text-xs px-2.5 py-1 rounded-full bg-gray-100 text-gray-500 flex-shrink-0 hidden"></span>
      </div>

      <!-- Tab bar -->
      <div class="flex items-center border-b border-gray-200 px-4">
        <button id="tab-md" onclick="switchTab('md')"
          class="tab-active py-3 px-1 mr-4 text-sm transition-colors">
          Markdown
        </button>
        <button id="tab-preview" onclick="switchTab('preview')"
          class="tab-inactive py-3 px-1 mr-4 text-sm transition-colors">
          Preview
        </button>
        <div class="flex-1"></div>
        <button onclick="copyMarkdown()"
          class="text-xs text-gray-500 hover:text-gray-900 px-2 py-1.5 rounded hover:bg-gray-100 transition flex items-center gap-1">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <rect x="9" y="9" width="13" height="13" rx="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
          <span id="copy-label">Copy</span>
        </button>
        <button onclick="saveMarkdown()"
          class="text-xs text-gray-500 hover:text-gray-900 px-2 py-1.5 rounded hover:bg-gray-100 transition ml-1 flex items-center gap-1">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          Save .md
        </button>
      </div>

      <!-- Markdown panel -->
      <div id="panel-md" class="overflow-auto" style="max-height:65vh">
        <pre id="md-content" class="p-5 text-xs font-mono text-gray-800 whitespace-pre-wrap leading-relaxed"></pre>
      </div>

      <!-- Preview panel -->
      <div id="panel-preview" class="hidden overflow-auto p-6" style="max-height:65vh">
        <div id="preview-content" class="prose text-sm max-w-none"></div>
      </div>

    </div>

    <!-- How it works -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
      <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-4">How it works</h2>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-violet-100 text-violet-700 text-xs font-bold flex items-center justify-center flex-shrink-0">1</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Native Markdown</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">Sites supporting <code class="bg-gray-100 px-1 py-0.5 rounded text-gray-700">Accept: text/markdown</code> return clean Markdown directly from the edge — zero parsing.</div>
          </div>
        </div>
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center flex-shrink-0">2</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Workers AI</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">HTML pages are converted via Cloudflare Workers AI <code class="bg-gray-100 px-1 py-0.5 rounded text-gray-700">toMarkdown()</code> — intelligent, structured output.</div>
          </div>
        </div>
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-amber-100 text-amber-700 text-xs font-bold flex items-center justify-center flex-shrink-0">3</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Browser Render</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">JS-heavy pages are rendered in a headless browser for full content extraction before AI conversion.</div>
          </div>
        </div>
      </div>
      <div class="mt-5 pt-4 border-t border-gray-100 space-y-1.5">
        <div class="text-xs text-gray-400 font-mono">
          <span class="text-gray-500">GET</span>  /https://example.com → <span class="text-gray-600">text/markdown</span>  <span class="text-gray-300 ml-2"># X-Conversion-Method, X-Duration-Ms</span>
        </div>
        <div class="text-xs text-gray-400 font-mono">
          <span class="text-gray-500">POST</span> /convert <span class="text-gray-600">{"url":"..."}</span> → <span class="text-gray-600">{"markdown","method","durationMs","tokens"}</span>
        </div>
      </div>
    </div>

  </main>

<script>
let currentMarkdown = '';

function handleSubmit(e) {
  e.preventDefault();
  const url = document.getElementById('url-input').value.trim();
  if (!url) return;
  convertUrl(url);
}

function setExample(url) {
  document.getElementById('url-input').value = url;
  convertUrl(url);
}

function setLoading(loading) {
  const btn = document.getElementById('submit-btn');
  btn.disabled = loading;
  document.getElementById('btn-text').textContent = loading ? 'Converting…' : 'Convert';
  document.getElementById('btn-spinner').classList.toggle('hidden', !loading);
}

function showError(msg) {
  document.getElementById('error-card').classList.remove('hidden');
  document.getElementById('error-msg').textContent = msg;
  document.getElementById('result-card').classList.add('hidden');
}

function hideError() {
  document.getElementById('error-card').classList.add('hidden');
}

async function convertUrl(url) {
  setLoading(true);
  hideError();

  try {
    const resp = await fetch('/convert', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: 'Conversion failed' }));
      throw new Error(err.error || 'HTTP ' + resp.status);
    }
    showResult(await resp.json());
  } catch (err) {
    showError(err.message || 'Failed to convert. Please try again.');
  } finally {
    setLoading(false);
  }
}

const METHOD_CONFIG = {
  primary: { text: '✦ Native Markdown', cls: 'bg-violet-50 text-violet-700 border-violet-200' },
  ai:      { text: '⚡ Workers AI',      cls: 'bg-blue-50 text-blue-700 border-blue-200' },
  browser: { text: '🖥 Browser Render', cls: 'bg-amber-50 text-amber-700 border-amber-200' },
};

function showResult(data) {
  currentMarkdown = data.markdown || '';

  document.getElementById('result-title').textContent = data.title || data.sourceUrl || '';

  const cfg = METHOD_CONFIG[data.method] || METHOD_CONFIG.ai;
  const badge = document.getElementById('method-badge');
  badge.textContent = cfg.text;
  badge.className = 'text-xs px-2.5 py-1 rounded-full font-medium flex-shrink-0 border ' + cfg.cls;

  document.getElementById('duration-badge').textContent = data.durationMs + 'ms';

  const tokensBadge = document.getElementById('tokens-badge');
  if (data.tokens) {
    tokensBadge.textContent = '~' + data.tokens.toLocaleString() + ' tokens';
    tokensBadge.classList.remove('hidden');
  } else {
    tokensBadge.classList.add('hidden');
  }

  document.getElementById('md-content').textContent = currentMarkdown;
  document.getElementById('preview-content').innerHTML = marked.parse(currentMarkdown);

  document.getElementById('result-card').classList.remove('hidden');
  switchTab('md');
  document.getElementById('result-card').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function switchTab(tab) {
  const panels = { md: 'panel-md', preview: 'panel-preview' };
  const tabs   = { md: 'tab-md',   preview: 'tab-preview'   };
  for (const [key, panelId] of Object.entries(panels)) {
    document.getElementById(panelId).classList.toggle('hidden', key !== tab);
  }
  for (const [key, tabId] of Object.entries(tabs)) {
    document.getElementById(tabId).className =
      (key === tab ? 'tab-active' : 'tab-inactive') + ' py-3 px-1 mr-4 text-sm transition-colors';
  }
}

async function copyMarkdown() {
  try {
    await navigator.clipboard.writeText(currentMarkdown);
    const label = document.getElementById('copy-label');
    label.textContent = 'Copied!';
    setTimeout(() => { label.textContent = 'Copy'; }, 2000);
  } catch {
    // fallback: select text
  }
}

function saveMarkdown() {
  const blob = new Blob([currentMarkdown], { type: 'text/markdown' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  const title = document.getElementById('result-title').textContent || 'document';
  a.download = title.replace(/[^\w\s-]/g, '').trim().replace(/\s+/g, '-').toLowerCase() + '.md';
  a.click();
  URL.revokeObjectURL(a.href);
}

// Handle URL in path on load (e.g., /https://example.com)
window.addEventListener('load', () => {
  const path = window.location.pathname.slice(1);
  if (path.startsWith('http://') || path.startsWith('https://')) {
    document.getElementById('url-input').value = decodeURIComponent(path);
    convertUrl(decodeURIComponent(path));
  }
});
</script>
</body>
</html>`;
}
```

**Step 2: Commit**
```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add tools/markdown/src/page.ts
git commit -m "feat(markdown): shadcn-inspired light theme page with tabbed result view"
```

---

## Task 4: Hono Routes (index.ts)

**Files:**
- Modify: `tools/markdown/src/index.ts`

**Step 1: Replace `tools/markdown/src/index.ts`**

```typescript
import { Hono } from 'hono';
import { convert } from './convert';
import { renderPage } from './page';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Env = { AI: any; BROWSER: Fetcher };

const app = new Hono<{ Bindings: Env }>();

// Landing page
app.get('/', (c) => c.html(renderPage()));

// JSON API: POST /convert
app.post('/convert', async (c) => {
  let body: { url?: string };
  try {
    body = await c.req.json<{ url?: string }>();
  } catch {
    return c.json({ error: 'Invalid JSON body' }, 400);
  }
  if (!body.url || typeof body.url !== 'string') {
    return c.json({ error: 'url is required' }, 400);
  }
  try {
    const result = await convert(body.url, c.env);
    return c.json(result);
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Conversion failed';
    return c.json({ error: msg }, 422);
  }
});

// Text API: GET /:url+ (mirrors markdown.new/https://example.com pattern)
// Matches any path starting with http:// or https://
app.get('/*', async (c) => {
  const path = c.req.path.slice(1); // strip leading /
  if (!path.startsWith('http://') && !path.startsWith('https://')) {
    return c.notFound();
  }
  // Reconstruct full URL including query string
  const search = new URL(c.req.url).search;
  const url = path + search;
  try {
    const result = await convert(url, c.env);
    return new Response(result.markdown, {
      headers: {
        'Content-Type': 'text/markdown; charset=utf-8',
        'X-Conversion-Method': result.method,
        'X-Duration-Ms': String(result.durationMs),
        'X-Title': encodeURIComponent(result.title),
        ...(result.tokens ? { 'X-Markdown-Tokens': String(result.tokens) } : {}),
        'Access-Control-Allow-Origin': '*',
        'Cache-Control': 'public, max-age=300',
      },
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Conversion failed';
    return c.text(`Error: ${msg}`, 422);
  }
});

export default app;
```

**Step 2: Commit**
```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
git add tools/markdown/src/index.ts
git commit -m "feat(markdown): Hono routes — landing page, POST /convert, GET /:url+"
```

---

## Task 5: Deploy and Verify

**Step 1: Deploy**
```bash
cd tools/markdown && npm run deploy
```
Expected output: `Published markdown (https://markdown.<subdomain>.workers.dev)` or similar.

**Step 2: Check wrangler.toml name resolves to correct URL**

If the worker deploys under a different hostname than `markdown.go-mizu.workers.dev`, add a custom domain in the CF dashboard or check that the CF account subdomain matches `go-mizu`.

**Step 3: Verify landing page**
```bash
curl -sf https://markdown.go-mizu.workers.dev/ | head -3
```
Expected: `<!DOCTYPE html>` HTML response.

**Step 4: Verify GET API**
```bash
curl -sv https://markdown.go-mizu.workers.dev/https://example.com 2>&1 | grep -E "< X-Conversion|< X-Duration|^# Example"
```
Expected: headers `X-Conversion-Method: ai` (or `primary`) and `X-Duration-Ms: <number>`, body starts with `# Example Domain`.

**Step 5: Verify POST API**
```bash
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}' | jq '{method, durationMs, title}'
```
Expected:
```json
{
  "method": "ai",
  "durationMs": 300,
  "title": "Example Domain"
}
```

**Step 6: Browser smoke test**
- Open `https://markdown.go-mizu.workers.dev`
- Enter `https://news.ycombinator.com` → click Convert
- Verify: result card appears, method badge shows (blue = Workers AI), duration shown
- Click "Preview" tab → Markdown renders as HTML
- Click "Copy" → clipboard works (shows "Copied!")
- Click "Save .md" → downloads `.md` file
- Test example buttons (example.com, news.ycombinator.com)

**Step 7: Commit any fixes**
```bash
git add tools/markdown/ && git commit -m "fix(markdown): post-deploy corrections"
```

---

## API Reference (final)

### `GET /`
Serves the HTML landing page.

### `GET /:url+`
```
GET /https://example.com
→ 200 text/markdown
   X-Conversion-Method: ai
   X-Duration-Ms: 234
   X-Markdown-Tokens: 1024

# Example Domain
...
```

### `POST /convert`
```json
POST /convert
{ "url": "https://example.com" }

→ {
    "markdown": "# Example Domain\n...",
    "method": "ai",
    "durationMs": 234,
    "title": "Example Domain",
    "tokens": 1024,
    "sourceUrl": "https://example.com"
  }
```

## CF Bindings Required

```toml
[ai]
binding = "AI"       # CF Workers AI (free plan)

[[browser]]
binding = "BROWSER"  # CF Browser Rendering (requires paid plan)
```

Note: If `BROWSER` binding is unavailable (free plan), tier 3 will throw and the result will include an error. Tiers 1 and 2 always work on free plans.
