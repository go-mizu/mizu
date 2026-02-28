# Markdown Converter Enhancement Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign the markdown CF Worker with a here.now-inspired UI: full-width header, larger body, no rounded corners, hero with functional "copy for agent" button, `/preview?url=` redirect on convert, `/docs` page with sidebar, and remove the footer.

**Architecture:** Three TypeScript source files in `blueprints/search/tools/markdown/src/`. `index.ts` adds two new routes (`/preview`, `/docs`). Two new template files (`preview.ts`, `docs.ts`) are created. `page.ts` is rewritten. `convert.ts` is untouched.

**Tech Stack:** Hono, Cloudflare Workers, TypeScript, inline HTML/CSS/JS template literals, Geist font (Google Fonts CDN), marked.js + DOMPurify CDNs.

---

## Shared design tokens (apply everywhere)

All pages share these CSS variables and conventions:
- `--w: 1120px` (body max-width — up from 800px)
- `--sans: 'Geist', -apple-system, sans-serif`
- `--mono: 'Geist Mono', ui-monospace, monospace`
- `--fg: #0a0a0a` / `--fg2: #555` / `--fg3: #999`
- `--border: #e8e8e8` / `--border2: #f0f0f0`
- `--code-bg: #111` / `--code-fg: #e4e4e7`
- **Zero border-radius everywhere** — no `border-radius` on inputs, buttons, cards, badges
- Header: full-width with `padding: 0 32px`, **no border-bottom**, no max-width wrapper
- No `<footer>` element on any page

---

## Task 1: Rewrite `page.ts` — landing page

**Files:**
- Modify: `blueprints/search/tools/markdown/src/page.ts` (complete rewrite)

**What changes:**
1. Header full-width, no border-bottom, larger padding
2. Max-width 1120px, all border-radius removed
3. Hero: here.now-style with black-square numbered steps + functional "Copy setup instructions for my agent" button
4. On Convert: `window.location.href = '/preview?url=' + encodeURIComponent(url)` (no inline result)
5. API reference: dark code blocks with per-block copy buttons
6. Remove `<footer>`
7. Bigger section h2 (32px), friendlier body text (16px)
8. No Tailwind CDN — pure custom CSS

**Step 1: Write the complete new `page.ts`**

```typescript
export function renderPage(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>URL \u2192 Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --sans:'Geist',-apple-system,sans-serif;
  --mono:'Geist Mono',ui-monospace,monospace;
  --bg:#fff;--fg:#0a0a0a;--fg2:#555;--fg3:#999;
  --border:#e8e8e8;--border2:#f0f0f0;
  --code-bg:#111;--code-fg:#e4e4e7;
  --w:1120px
}
html{font-size:16px}
body{font-family:var(--sans);background:var(--bg);color:var(--fg);line-height:1.6;-webkit-font-smoothing:antialiased}
a{color:inherit;text-decoration:none}
code{font-family:var(--mono);font-size:12px;background:#f5f5f5;padding:2px 6px}

/* full-width header — no border, no max-width wrapper */
header{padding:14px 32px;display:flex;align-items:center;justify-content:space-between}
.logo{font-size:14px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px}
.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0}
nav a{font-size:13px;color:var(--fg3);margin-left:20px;transition:color .15s}
nav a:hover{color:var(--fg)}

/* body container */
.w{max-width:var(--w);margin:0 auto;padding:0 32px}

/* hero */
.hero{padding:80px 0 72px}
h1{font-size:clamp(40px,6vw,68px);font-weight:700;letter-spacing:-.04em;line-height:1.05;margin-bottom:32px}
.steps{margin-bottom:48px}
.step{display:flex;align-items:flex-start;gap:16px;margin-bottom:16px;font-size:18px;color:var(--fg2)}
.step-num{width:22px;height:22px;background:var(--fg);color:#fff;font-size:12px;font-weight:700;display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px}
.step code{font-size:15px;background:#f0f0f0;padding:2px 7px}
.hero-cta{display:grid;grid-template-columns:1fr 1fr;gap:48px;align-items:start}
@media(max-width:680px){.hero-cta{grid-template-columns:1fr}}
.cta-lbl{font-family:var(--mono);font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:12px}
.input-row{display:flex}
.url-in{flex:1;font-family:var(--mono);font-size:14px;padding:13px 16px;border:1px solid var(--border);border-right:none;background:#fff;color:var(--fg);outline:none;transition:border-color .15s;min-width:0}
.url-in::placeholder{color:var(--fg3)}
.url-in:focus{border-color:var(--fg)}
.cvt-btn{font-family:var(--sans);font-size:14px;font-weight:500;padding:13px 22px;background:var(--fg);color:#fff;border:none;cursor:pointer;white-space:nowrap;display:flex;align-items:center;gap:6px;transition:background .15s}
.cvt-btn:hover{background:#333}
.cvt-btn:disabled{opacity:.5;cursor:default}
.examples{margin-top:10px;font-size:13px;color:var(--fg3);display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.eg{color:var(--fg2);cursor:pointer;text-decoration:underline;text-decoration-color:var(--border);transition:color .15s}
.eg:hover{color:var(--fg)}
@keyframes spin{to{transform:rotate(360deg)}}
.spin{animation:spin .7s linear infinite}
.agent-btn{width:100%;font-family:var(--sans);font-size:16px;font-weight:500;padding:16px 22px;background:var(--fg);color:#fff;border:none;cursor:pointer;display:flex;align-items:center;justify-content:space-between;transition:background .15s}
.agent-btn:hover{background:#333}
.agent-btn svg{flex-shrink:0}

/* section dividers */
hr.sep{border:none;border-top:1px solid var(--border2)}

/* sections */
.sec{padding:72px 0}
.sec-lbl{font-family:var(--mono);font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:20px}
.sec h2{font-size:32px;font-weight:600;letter-spacing:-.03em;margin-bottom:14px}
.sec-sub{font-size:17px;color:var(--fg2);margin-bottom:40px;max-width:540px;line-height:1.6}

/* 3-step agent flow */
.agent-steps{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;background:var(--border2);border:1px solid var(--border2);margin-bottom:48px}
@media(max-width:640px){.agent-steps{grid-template-columns:1fr}}
.astep{background:#fff;padding:28px}
.astep-n{font-family:var(--mono);font-size:11px;color:var(--fg3);margin-bottom:10px}
.astep-t{font-size:16px;font-weight:600;margin-bottom:6px}
.astep-d{font-size:15px;color:var(--fg2);line-height:1.6}

/* code tabs */
.ctabs{border:1px solid var(--border)}
.cbar{display:flex;background:#fafafa;border-bottom:1px solid var(--border);padding:0 4px}
.ctab{font-family:var(--mono);font-size:12px;padding:9px 14px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-1px;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s}
.ctab.on{color:var(--fg);border-bottom-color:var(--fg)}
.cpanel{display:none;background:var(--code-bg);padding:22px;overflow-x:auto;position:relative}
.cpanel.on{display:block}
.cpanel pre{font-family:var(--mono);font-size:13.5px;line-height:1.7;color:var(--code-fg);white-space:pre;margin:0}
.c1{color:#6b7280}.c2{color:#93c5fd}.c3{color:#86efac}.c4{color:#fcd34d}
/* code block copy btn */
.cbcopy{position:absolute;top:12px;right:12px;background:#222;border:1px solid #333;color:#aaa;font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:all .15s}
.cbcopy:hover{background:#333;color:#fff}

/* pipeline */
.pgrid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;background:var(--border2);border:1px solid var(--border2)}
@media(max-width:580px){.pgrid{grid-template-columns:1fr}}
.pcard{background:#fff;padding:28px}
.pn{font-family:var(--mono);font-size:11px;color:var(--fg3);margin-bottom:10px}
.pt{font-size:16px;font-weight:600;margin-bottom:6px}
.pd{font-size:15px;color:var(--fg2);line-height:1.6}
.ptag{display:inline-block;margin-top:10px;font-family:var(--mono);font-size:11px;padding:2px 7px}
.b-native{background:#f3f0ff;color:#5b21b6}
.b-ai{background:#eff6ff;color:#1d4ed8}
.b-browser{background:#fffbeb;color:#92400e}

/* api ref */
.api-sec h2{font-size:32px;font-weight:600;letter-spacing:-.03em;margin-bottom:20px}
.ep{border:1px solid var(--border2);margin-bottom:10px}
.eph{padding:12px 16px;display:flex;align-items:center;gap:10px;background:#fafafa;border-bottom:1px solid var(--border2)}
.mtag{font-family:var(--mono);font-size:11px;font-weight:500;padding:2px 8px;flex-shrink:0}
.get{background:#f0fdf4;color:#15803d}
.post{background:#eff6ff;color:#1d4ed8}
.epath{font-family:var(--mono);font-size:14px}
.edesc{font-size:12px;color:var(--fg3);margin-left:auto}
.epb{padding:0}
/* dark code block inside endpoint */
.ep-code{background:var(--code-bg);padding:18px 20px;position:relative;overflow-x:auto}
.ep-code pre{font-family:var(--mono);font-size:13px;line-height:1.8;color:var(--code-fg);white-space:pre;margin:0}
.rk{color:#93c5fd}.rv{color:#e4e4e7}.rc{color:#6b7280}
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
    <a href="/docs">Docs</a>
    <a href="/llms.txt">llms.txt</a>
    <a href="https://github.com/go-mizu/mizu">GitHub</a>
  </nav>
</header>

<div class="w">
  <section class="hero">
    <h1>Any URL,<br>clean Markdown</h1>
    <div class="steps">
      <div class="step">
        <span class="step-num">1</span>
        <span>Prepend your URL: <code>markdown.go-mizu.workers.dev/{url}</code></span>
      </div>
      <div class="step">
        <span class="step-num">2</span>
        <span>Get <code>text/markdown</code> back &mdash; no account, no API key</span>
      </div>
    </div>
    <div class="hero-cta">
      <div>
        <div class="cta-lbl">Convert a URL</div>
        <form id="form" onsubmit="handleSubmit(event)">
          <div class="input-row">
            <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">
            <button type="submit" class="cvt-btn" id="sub-btn">
              <span id="btn-t">Convert</span>
              <svg id="btn-sp" class="spin" style="display:none" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
            </button>
          </div>
          <div class="examples">
            <span>Try:</span>
            <span class="eg" onclick="setEg('https://example.com')">example.com</span>
            <span style="color:var(--border2)">&middot;</span>
            <span class="eg" onclick="setEg('https://news.ycombinator.com')">news.ycombinator.com</span>
            <span style="color:var(--border2)">&middot;</span>
            <span class="eg" onclick="setEg('https://blog.cloudflare.com')">blog.cloudflare.com</span>
          </div>
        </form>
      </div>
      <div>
        <div class="cta-lbl">For your agent</div>
        <button class="agent-btn" id="agent-copy-btn" onclick="copyAgentInstructions()">
          Copy setup instructions for my agent
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="0"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </button>
        <div id="agent-copied" style="display:none;font-size:13px;color:var(--fg3);margin-top:8px">Copied to clipboard &mdash; paste into your agent chat.</div>
      </div>
    </div>
  </section>
</div>

<hr class="sep">

<div class="w">
  <section class="sec" id="agents">
    <p class="sec-lbl">How it works for agents</p>
    <h2>Get Markdown in one request</h2>
    <p class="sec-sub">No SDK, no setup. Any agent that can make HTTP requests works immediately.</p>
    <div class="agent-steps">
      <div class="astep">
        <div class="astep-n">01 / Request</div>
        <div class="astep-t">Prepend the URL</div>
        <div class="astep-d">Append any URL to <code>markdown.go-mizu.workers.dev/</code>. Query strings are preserved automatically.</div>
      </div>
      <div class="astep">
        <div class="astep-n">02 / Receive</div>
        <div class="astep-t">Plain text/markdown</div>
        <div class="astep-d">Response is clean <code>text/markdown</code>. Metadata in headers: method, duration, title, token count.</div>
      </div>
      <div class="astep">
        <div class="astep-n">03 / Scale</div>
        <div class="astep-t">Edge-cached at 1 hour</div>
        <div class="astep-d">CDN caches for 1 hour with <code>stale-while-revalidate</code> so latency stays low as you scale.</div>
      </div>
    </div>
    <div class="ctabs">
      <div class="cbar">
        <button class="ctab on" id="ctab-sh" onclick="switchCode('sh')">Shell</button>
        <button class="ctab" id="ctab-js" onclick="switchCode('js')">JavaScript</button>
        <button class="ctab" id="ctab-py" onclick="switchCode('py')">Python</button>
      </div>
      <div id="cpanel-sh" class="cpanel on">
        <button class="cbcopy" onclick="copyBlock('cpanel-sh',this)">copy</button>
        <pre><span class="c1"># GET \u2014 returns text/markdown</span>
curl https://markdown.go-mizu.workers.dev/https://example.com

<span class="c1"># POST \u2014 structured JSON with metadata</span>
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \\
  -H 'Content-Type: application/json' \\
  -d '{"url":"https://example.com"}' | jq .method</pre>
      </div>
      <div id="cpanel-js" class="cpanel">
        <button class="cbcopy" onclick="copyBlock('cpanel-js',this)">copy</button>
        <pre><span class="c1">// One line \u2014 returns markdown text</span>
<span class="c2">const</span> md = <span class="c2">await</span> <span class="c4">fetch</span>(
  <span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url
).<span class="c4">then</span>(r =&gt; r.<span class="c4">text</span>());

<span class="c1">// JSON API with method + timing</span>
<span class="c2">const</span> res = <span class="c2">await</span> <span class="c4">fetch</span>(<span class="c3">'https://markdown.go-mizu.workers.dev/convert'</span>, {
  method: <span class="c3">'POST'</span>,
  headers: { <span class="c3">'Content-Type'</span>: <span class="c3">'application/json'</span> },
  body: <span class="c3">JSON.stringify</span>({ url })
}).<span class="c4">then</span>(r =&gt; r.<span class="c4">json</span>());</pre>
      </div>
      <div id="cpanel-py" class="cpanel">
        <button class="cbcopy" onclick="copyBlock('cpanel-py',this)">copy</button>
        <pre><span class="c2">import</span> httpx

md = httpx.<span class="c4">get</span>(
    <span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url
).text

res = httpx.<span class="c4">post</span>(
    <span class="c3">'https://markdown.go-mizu.workers.dev/convert'</span>,
    json={<span class="c3">'url'</span>: url}
).<span class="c4">json</span>()</pre>
      </div>
    </div>
  </section>
</div>

<hr class="sep">

<div class="w">
  <section class="sec" id="pipeline">
    <p class="sec-lbl">Conversion pipeline</p>
    <h2>Three tiers, one result</h2>
    <p class="sec-sub">Every URL goes through the best available tier, falling back automatically until Markdown is produced.</p>
    <div class="pgrid">
      <div class="pcard">
        <div class="pn">Tier 1</div>
        <div class="pt">Native Markdown</div>
        <div class="pd">Sites that honour <code>Accept: text/markdown</code> return structured Markdown directly &mdash; zero parsing overhead.</div>
        <span class="ptag b-native">primary</span>
      </div>
      <div class="pcard">
        <div class="pn">Tier 2</div>
        <div class="pt">Workers AI</div>
        <div class="pd">HTML is converted via Cloudflare Workers AI <code>toMarkdown()</code> &mdash; fast, structure-aware extraction.</div>
        <span class="ptag b-ai">ai</span>
      </div>
      <div class="pcard">
        <div class="pn">Tier 3</div>
        <div class="pt">Browser Render</div>
        <div class="pd">JS-heavy SPAs are rendered in a headless browser first, capturing dynamic content before AI conversion.</div>
        <span class="ptag b-browser">browser</span>
      </div>
    </div>
  </section>
</div>

<hr class="sep">

<div class="w">
  <section class="sec api-sec" id="api">
    <p class="sec-lbl">API reference</p>
    <h2>Endpoints</h2>

    <div class="ep">
      <div class="eph">
        <span class="mtag get">GET</span>
        <span class="epath">/{url}</span>
        <span class="edesc">Returns text/markdown</span>
      </div>
      <div class="epb">
        <div class="ep-code">
          <button class="cbcopy" onclick="copyBlock('ep-code-get',this)">copy</button>
          <pre id="ep-code-get"><span class="rc"># Append any absolute URL (http:// or https://)</span>
curl https://markdown.go-mizu.workers.dev/https://example.com

<span class="rc"># Response headers</span>
<span class="rk">Content-Type</span><span class="rv">: text/markdown; charset=utf-8</span>
<span class="rk">X-Conversion-Method</span><span class="rv">: primary | ai | browser</span>
<span class="rk">X-Duration-Ms</span><span class="rv">: 342</span>
<span class="rk">X-Title</span><span class="rv">: Example Domain</span>
<span class="rk">X-Markdown-Tokens</span><span class="rv">: 1248</span>
<span class="rk">Cache-Control</span><span class="rv">: public, max-age=300, s-maxage=3600, stale-while-revalidate=86400</span></pre>
        </div>
      </div>
    </div>

    <div class="ep">
      <div class="eph">
        <span class="mtag post">POST</span>
        <span class="epath">/convert</span>
        <span class="edesc">Returns JSON</span>
      </div>
      <div class="epb">
        <div class="ep-code">
          <button class="cbcopy" onclick="copyBlock('ep-code-post',this)">copy</button>
          <pre id="ep-code-post"><span class="rc"># Request</span>
curl -X POST https://markdown.go-mizu.workers.dev/convert \\
  -H 'Content-Type: application/json' \\
  -d '{"url":"https://example.com"}'

<span class="rc"># Response JSON</span>
{
  <span class="rk">"markdown"</span><span class="rv">: "# Example Domain\n\n..."</span>,
  <span class="rk">"method"</span><span class="rv">: "ai"</span>,
  <span class="rk">"durationMs"</span><span class="rv">: 342</span>,
  <span class="rk">"title"</span><span class="rv">: "Example Domain"</span>,
  <span class="rk">"tokens"</span><span class="rv">: 1248</span>
}</pre>
        </div>
      </div>
    </div>
  </section>
</div>

<script>
var AGENT_INSTRUCTIONS = 'I\'d like you to use https://markdown.go-mizu.workers.dev, the URL-to-Markdown API.\\n\\nTo fetch any URL as Markdown:\\n  GET https://markdown.go-mizu.workers.dev/{url}\\n\\nFor structured JSON with metadata:\\n  POST https://markdown.go-mizu.workers.dev/convert\\n  Body: {"url": "https://example.com"}\\n\\nReturns text/markdown. No API key needed. Free.';

function handleSubmit(e) {
  e.preventDefault();
  var url = document.getElementById('url-in').value.trim();
  if (!url) return;
  document.getElementById('sub-btn').disabled = true;
  document.getElementById('btn-t').textContent = 'Loading\u2026';
  document.getElementById('btn-sp').style.display = '';
  window.location.href = '/preview?url=' + encodeURIComponent(url);
}

function setEg(url) {
  document.getElementById('url-in').value = url;
  handleSubmit({ preventDefault: function(){} });
}

async function copyAgentInstructions() {
  try {
    await navigator.clipboard.writeText(AGENT_INSTRUCTIONS);
    document.getElementById('agent-copied').style.display = 'block';
    setTimeout(function() { document.getElementById('agent-copied').style.display = 'none'; }, 3000);
  } catch(e) {
    var ta = document.createElement('textarea');
    ta.value = AGENT_INSTRUCTIONS;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
    document.getElementById('agent-copied').style.display = 'block';
    setTimeout(function() { document.getElementById('agent-copied').style.display = 'none'; }, 3000);
  }
}

function switchCode(lang) {
  ['sh','js','py'].forEach(function(k) {
    document.getElementById('cpanel-' + k).className = 'cpanel' + (k === lang ? ' on' : '');
    document.getElementById('ctab-' + k).className = 'ctab' + (k === lang ? ' on' : '');
  });
}

async function copyBlock(id, btn) {
  var el = document.getElementById(id);
  var text = el ? el.innerText || el.textContent : '';
  try {
    await navigator.clipboard.writeText(text.trim());
  } catch(e) {
    var ta = document.createElement('textarea');
    ta.value = text.trim();
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }
  var orig = btn.textContent;
  btn.textContent = 'copied!';
  setTimeout(function() { btn.textContent = orig; }, 2000);
}
</script>
</body>
</html>`;
}
```

**Step 2: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```
Expected: no errors.

**Step 3: Commit**

```bash
git add blueprints/search/tools/markdown/src/page.ts
git commit -m "feat(markdown): redesign landing page — full-width, here.now hero, no footer"
```

---

## Task 2: Create `src/preview.ts` — `/preview?url=` page

**Files:**
- Create: `blueprints/search/tools/markdown/src/preview.ts`

**What it does:**
- Renders a clean two-panel page: URL input at top, result below
- Client JS reads `?url=` param on load and auto-calls `POST /convert`
- Shows: method badge, duration badge, tokens badge, "View raw →" link, Copy button, Markdown/Preview tabs
- No footer, same header as landing page

**Step 1: Create `preview.ts`**

```typescript
export function renderPreview(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Preview \u2014 URL \u2192 Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script src="https://cdn.jsdelivr.net/npm/marked@15/marked.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/dompurify@3/dist/purify.min.js"></script>
  <style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --sans:'Geist',-apple-system,sans-serif;
  --mono:'Geist Mono',ui-monospace,monospace;
  --bg:#fff;--fg:#0a0a0a;--fg2:#555;--fg3:#999;
  --border:#e8e8e8;--border2:#f0f0f0;
  --code-bg:#111;--code-fg:#e4e4e7;
  --w:1120px
}
html{font-size:16px}
body{font-family:var(--sans);background:var(--bg);color:var(--fg);line-height:1.6;-webkit-font-smoothing:antialiased}
a{color:inherit;text-decoration:none}
code{font-family:var(--mono);font-size:12px;background:#f5f5f5;padding:2px 6px}

/* header */
header{padding:14px 32px;display:flex;align-items:center;justify-content:space-between}
.logo{font-size:14px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px}
.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0}
nav a{font-size:13px;color:var(--fg3);margin-left:20px;transition:color .15s}
nav a:hover{color:var(--fg)}
hr.sep{border:none;border-top:1px solid var(--border2)}

/* url bar */
.url-bar{padding:20px 32px;border-bottom:1px solid var(--border2)}
.url-bar-inner{max-width:var(--w);margin:0 auto;display:flex;gap:0}
.url-in{flex:1;font-family:var(--mono);font-size:14px;padding:11px 16px;border:1px solid var(--border);border-right:none;background:#fff;color:var(--fg);outline:none;transition:border-color .15s;min-width:0}
.url-in::placeholder{color:var(--fg3)}
.url-in:focus{border-color:var(--fg)}
.cvt-btn{font-family:var(--sans);font-size:14px;font-weight:500;padding:11px 22px;background:var(--fg);color:#fff;border:none;cursor:pointer;white-space:nowrap;display:flex;align-items:center;gap:6px;transition:background .15s}
.cvt-btn:hover{background:#333}
.cvt-btn:disabled{opacity:.5;cursor:default}
@keyframes spin{to{transform:rotate(360deg)}}
.spin{animation:spin .7s linear infinite}

/* state panels */
.w{max-width:var(--w);margin:0 auto;padding:0 32px}
.loading-state{padding:80px 0;text-align:center}
.loading-text{font-size:15px;color:var(--fg3);margin-top:16px}
.error-state{display:none;padding:48px 0}
.err-box{background:#fff5f5;border:1px solid #fecaca;padding:16px 20px;font-size:15px;color:#dc2626}

/* result */
.result-state{display:none}
.meta-bar{padding:14px 32px;border-bottom:1px solid var(--border2);display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.meta-bar-inner{max-width:var(--w);margin:0 auto;width:100%;display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.r-title{font-size:14px;font-weight:500;flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.badge{font-family:var(--mono);font-size:11px;padding:2px 8px;flex-shrink:0}
.b-native{background:#f3f0ff;color:#5b21b6}
.b-ai{background:#eff6ff;color:#1d4ed8}
.b-browser{background:#fffbeb;color:#92400e}
.b-dim{background:#f5f5f5;color:var(--fg2)}
.raw-link{font-size:13px;color:var(--fg2);text-decoration:underline;text-underline-offset:2px;white-space:nowrap}
.raw-link:hover{color:var(--fg)}
.tabbar{padding:0 32px;border-bottom:1px solid var(--border2);display:flex;align-items:center}
.tabbar-inner{max-width:var(--w);margin:0 auto;width:100%;display:flex;align-items:center}
.tab{font-size:14px;padding:12px 0;margin-right:24px;margin-bottom:-1px;background:none;border:none;border-bottom:2px solid transparent;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s}
.tab.on{color:var(--fg);border-bottom-color:var(--fg)}
.tacts{margin-left:auto;display:flex;gap:4px}
.tact{font-size:13px;padding:6px 12px;background:none;border:1px solid transparent;cursor:pointer;color:var(--fg3);display:flex;align-items:center;gap:5px;transition:all .15s;font-family:var(--sans)}
.tact:hover{color:var(--fg);border-color:var(--border);background:#fafafa}
.panel{display:none}
.panel.on{display:block}
.md-panel{padding:32px}
.md-inner{max-width:var(--w);margin:0 auto}
#md-out{font-family:var(--mono);font-size:13.5px;line-height:1.75;color:var(--fg);white-space:pre-wrap;word-break:break-word}
.pv-panel{padding:32px}
.pv-inner{max-width:860px;margin:0 auto}
/* prose */
.pv-inner h1{font-size:1.6em;font-weight:600;letter-spacing:-.02em;margin:0 0 14px}
.pv-inner h2{font-size:1.3em;font-weight:600;letter-spacing:-.01em;margin:22px 0 8px}
.pv-inner h3{font-size:1.1em;font-weight:600;margin:16px 0 6px}
.pv-inner p{margin:0 0 14px;color:#333;line-height:1.75}
.pv-inner ul,.pv-inner ol{padding-left:1.5em;margin:0 0 14px}
.pv-inner li{margin:3px 0;color:#333}
.pv-inner a{color:var(--fg);text-decoration:underline;text-underline-offset:2px}
.pv-inner blockquote{border-left:2px solid #ddd;padding-left:16px;color:#777;margin:12px 0}
.pv-inner code{background:#f5f5f5;padding:2px 6px;font-size:.875em}
.pv-inner pre{background:var(--code-bg);padding:16px;overflow-x:auto;margin:12px 0}
.pv-inner pre code{background:none;color:var(--code-fg);padding:0;font-size:.875em}
.pv-inner table{border-collapse:collapse;width:100%;margin:12px 0}
.pv-inner th,.pv-inner td{border:1px solid var(--border);padding:8px 14px}
.pv-inner th{background:#fafafa;font-weight:600}
.pv-inner hr{border:none;border-top:1px solid var(--border2);margin:18px 0}
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
    <a href="/docs">Docs</a>
    <a href="https://github.com/go-mizu/mizu">GitHub</a>
  </nav>
</header>

<div class="url-bar">
  <form class="url-bar-inner" id="form" onsubmit="handleSubmit(event)">
    <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">
    <button type="submit" class="cvt-btn" id="sub-btn">
      <span id="btn-t">Convert</span>
      <svg id="btn-sp" class="spin" style="display:none" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
    </button>
  </form>
</div>

<!-- Loading -->
<div id="loading-state" class="w loading-state" style="display:none">
  <svg class="spin" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#999" stroke-width="2" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
  <div class="loading-text">Converting&hellip;</div>
</div>

<!-- Error -->
<div id="error-state" class="w error-state" style="display:none">
  <div class="err-box" id="err-msg"></div>
</div>

<!-- Result -->
<div id="result-state" class="result-state">
  <div class="meta-bar">
    <div class="meta-bar-inner">
      <span id="r-title" class="r-title"></span>
      <span id="r-method" class="badge"></span>
      <span id="r-dur" class="badge b-dim"></span>
      <span id="r-tok" class="badge b-dim" style="display:none"></span>
      <a id="r-raw" class="raw-link" href="#" target="_blank">View raw \u2192</a>
    </div>
  </div>
  <div class="tabbar">
    <div class="tabbar-inner">
      <button class="tab on" id="tab-md" onclick="switchTab('md')">Markdown</button>
      <button class="tab" id="tab-pv" onclick="switchTab('pv')">Preview</button>
      <div class="tacts">
        <button class="tact" onclick="copyMd()">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="0"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
          <span id="copy-lbl">Copy</span>
        </button>
        <button class="tact" onclick="saveMd()">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          Save .md
        </button>
      </div>
    </div>
  </div>
  <div id="panel-md" class="panel md-panel on">
    <div class="md-inner"><pre id="md-out"></pre></div>
  </div>
  <div id="panel-pv" class="panel pv-panel">
    <div class="pv-inner" id="prev-out"></div>
  </div>
</div>

<script>
var md = '';
var currentUrl = '';

var METHOD_MAP = {
  primary: ['\u2726 Native', 'badge b-native'],
  ai:      ['\u26a1 Workers AI', 'badge b-ai'],
  browser: ['\uD83D\uDDA5 Browser', 'badge b-browser'],
};

function handleSubmit(e) {
  e.preventDefault();
  var url = document.getElementById('url-in').value.trim();
  if (!url) return;
  history.replaceState(null, '', '/preview?url=' + encodeURIComponent(url));
  convertUrl(url);
}

function setLoading(v) {
  document.getElementById('sub-btn').disabled = v;
  document.getElementById('btn-t').textContent = v ? 'Converting\u2026' : 'Convert';
  document.getElementById('btn-sp').style.display = v ? '' : 'none';
  document.getElementById('loading-state').style.display = v ? 'block' : 'none';
}

async function convertUrl(url) {
  currentUrl = url;
  document.getElementById('url-in').value = url;
  document.getElementById('error-state').style.display = 'none';
  document.getElementById('result-state').style.display = 'none';
  setLoading(true);
  try {
    var r = await fetch('/convert', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({url: url})
    });
    if (!r.ok) {
      var j = await r.json().catch(function() { return {error: 'Conversion failed'}; });
      throw new Error(j.error || 'HTTP ' + r.status);
    }
    showResult(await r.json(), url);
  } catch(e) {
    document.getElementById('err-msg').textContent = e.message || 'Conversion failed';
    document.getElementById('error-state').style.display = 'block';
  } finally {
    setLoading(false);
  }
}

function showResult(data, url) {
  md = data.markdown || '';
  document.getElementById('r-title').textContent = data.title || url;
  var cfg = METHOD_MAP[data.method] || METHOD_MAP.ai;
  var mb = document.getElementById('r-method');
  mb.textContent = cfg[0]; mb.className = cfg[1];
  document.getElementById('r-dur').textContent = data.durationMs + 'ms';
  var tb = document.getElementById('r-tok');
  if (data.tokens) { tb.textContent = '\u007e' + data.tokens.toLocaleString() + ' tokens'; tb.style.display = ''; }
  else { tb.style.display = 'none'; }
  document.getElementById('r-raw').href = '/' + url;
  document.getElementById('md-out').textContent = md;
  document.getElementById('prev-out').innerHTML = DOMPurify.sanitize(marked.parse(md));
  document.getElementById('result-state').style.display = 'block';
  switchTab('md');
}

function switchTab(t) {
  ['md','pv'].forEach(function(k) {
    document.getElementById('panel-' + k).className = 'panel' + (k === t ? ' on' : '') + (k === 'md' ? ' md-panel' : ' pv-panel');
    document.getElementById('tab-' + k).className = 'tab' + (k === t ? ' on' : '');
  });
}

async function copyMd() {
  try { await navigator.clipboard.writeText(md); }
  catch(e) { var ta = document.createElement('textarea'); ta.value = md; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta); }
  var el = document.getElementById('copy-lbl');
  el.textContent = 'Copied!';
  setTimeout(function() { el.textContent = 'Copy'; }, 2000);
}

function saveMd() {
  var blob = new Blob([md], {type: 'text/markdown'});
  var a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  var title = document.getElementById('r-title').textContent || 'document';
  a.download = title.replace(/[^\w\s-]/g,'').trim().replace(/\s+/g,'-').toLowerCase() + '.md';
  a.click();
  URL.revokeObjectURL(a.href);
}

window.addEventListener('load', function() {
  var params = new URLSearchParams(window.location.search);
  var url = params.get('url');
  if (url) {
    convertUrl(url);
  } else {
    // No URL: show empty input, focus it
    document.getElementById('url-in').focus();
  }
});
</script>
</body>
</html>`;
}
```

**Step 2: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```

**Step 3: Commit**

```bash
git add blueprints/search/tools/markdown/src/preview.ts
git commit -m "feat(markdown): add /preview?url= page"
```

---

## Task 3: Create `src/docs.ts` — `/docs` page

**Files:**
- Create: `blueprints/search/tools/markdown/src/docs.ts`

**What it does:**
- Left sidebar (240px sticky) with anchor links to sections
- Main content area (right): Overview, Quick start (code blocks with copy), API reference, Pipeline, Response headers, Limits
- Code blocks: dark (#111), copy button top-right corner
- No footer, same header as other pages

**Step 1: Create `docs.ts`**

```typescript
export function renderDocs(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs \u2014 URL \u2192 Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --sans:'Geist',-apple-system,sans-serif;
  --mono:'Geist Mono',ui-monospace,monospace;
  --bg:#fff;--fg:#0a0a0a;--fg2:#555;--fg3:#999;
  --border:#e8e8e8;--border2:#f0f0f0;
  --code-bg:#111;--code-fg:#e4e4e7;
  --sidebar:240px
}
html{font-size:16px}
body{font-family:var(--sans);background:var(--bg);color:var(--fg);line-height:1.6;-webkit-font-smoothing:antialiased}
a{color:inherit;text-decoration:none}
code{font-family:var(--mono);font-size:12.5px;background:#f5f5f5;padding:2px 6px}

/* header */
header{padding:14px 32px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid var(--border2)}
.logo{font-size:14px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px}
.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0}
nav a{font-size:13px;color:var(--fg3);margin-left:20px;transition:color .15s}
nav a:hover{color:var(--fg)}

/* layout */
.layout{display:flex;min-height:calc(100vh - 52px)}
.sidebar{width:var(--sidebar);flex-shrink:0;padding:32px 24px;position:sticky;top:0;height:calc(100vh - 52px);overflow-y:auto;border-right:1px solid var(--border2)}
.sidebar-title{font-size:11px;font-family:var(--mono);letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:16px}
.sidebar a{display:block;font-size:14px;color:var(--fg3);padding:4px 0;transition:color .15s}
.sidebar a:hover{color:var(--fg)}
.sidebar a.on{color:var(--fg);font-weight:500}
.content{flex:1;padding:48px 64px;max-width:900px}
@media(max-width:720px){.sidebar{display:none}.content{padding:32px 24px}}

/* content typography */
.content h1{font-size:36px;font-weight:700;letter-spacing:-.03em;margin-bottom:16px}
.content h2{font-size:24px;font-weight:600;letter-spacing:-.02em;margin:48px 0 14px;scroll-margin-top:24px}
.content h2:first-child{margin-top:0}
.content h3{font-size:17px;font-weight:600;margin:28px 0 10px}
.content p{font-size:16px;color:var(--fg2);line-height:1.75;margin-bottom:16px}
.content ul{padding-left:1.5em;margin-bottom:16px}
.content li{font-size:16px;color:var(--fg2);margin:4px 0;line-height:1.7}
.content strong{color:var(--fg);font-weight:600}

/* code blocks */
.cb{position:relative;margin:20px 0}
.cb pre{background:var(--code-bg);padding:20px 22px;overflow-x:auto;font-family:var(--mono);font-size:13.5px;line-height:1.7;color:var(--code-fg);white-space:pre}
.cb-copy{position:absolute;top:10px;right:10px;background:#222;border:1px solid #333;color:#aaa;font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:all .15s}
.cb-copy:hover{background:#333;color:#fff}
.c1{color:#6b7280}.c2{color:#93c5fd}.c3{color:#86efac}.c4{color:#fcd34d}
.rk{color:#93c5fd}.rv{color:#e4e4e7}.rc{color:#6b7280}

/* inline table */
.tbl{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
.tbl th{text-align:left;padding:8px 14px;border-bottom:2px solid var(--border);font-weight:600;color:var(--fg)}
.tbl td{padding:8px 14px;border-bottom:1px solid var(--border2);color:var(--fg2)}
.tbl td:first-child{font-family:var(--mono);font-size:12.5px;color:var(--fg)}

/* method tags */
.mtag{font-family:var(--mono);font-size:11px;font-weight:500;padding:2px 8px;display:inline-block;margin-right:6px}
.get{background:#f0fdf4;color:#15803d}
.post{background:#eff6ff;color:#1d4ed8}
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
  <nav class="sidebar">
    <div class="sidebar-title">Docs</div>
    <a href="#overview" class="on">Overview</a>
    <a href="#quickstart">Quick start</a>
    <a href="#api">API reference</a>
    <a href="#pipeline">Pipeline</a>
    <a href="#headers">Response headers</a>
    <a href="#limits">Limits</a>
  </nav>

  <div class="content">
    <h1>URL \u2192 Markdown</h1>
    <p>Free, instant URL-to-Markdown conversion for AI agents and LLM pipelines. No API key, no account.</p>

    <h2 id="overview">Overview</h2>
    <p>Publish any URL to get clean, structured Markdown back.</p>
    <ul>
      <li>Works with any HTTP/HTTPS URL</li>
      <li>Three-tier pipeline: native negotiation \u2192 Workers AI \u2192 Browser rendering</li>
      <li>Edge-cached for 1 hour with <code>stale-while-revalidate</code></li>
      <li>CORS-enabled for browser use</li>
    </ul>

    <h2 id="quickstart">Quick start</h2>
    <h3>1. Fetch as Markdown</h3>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre>curl https://markdown.go-mizu.workers.dev/https://example.com</pre>
    </div>
    <h3>2. Use the JSON API</h3>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre>curl -X POST https://markdown.go-mizu.workers.dev/convert \\
  -H 'Content-Type: application/json' \\
  -d '{"url":"https://example.com"}'</pre>
    </div>
    <h3>JavaScript</h3>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre><span class="c2">const</span> md = <span class="c2">await</span> <span class="c4">fetch</span>(
  <span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url
).<span class="c4">then</span>(r =&gt; r.<span class="c4">text</span>());</pre>
    </div>
    <h3>Python</h3>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre><span class="c2">import</span> httpx
md = httpx.<span class="c4">get</span>(<span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url).text</pre>
    </div>

    <h2 id="api">API reference</h2>

    <h3><span class="mtag get">GET</span> /{url}</h3>
    <p>Convert a URL to Markdown. Append any <code>http://</code> or <code>https://</code> URL directly to the worker base URL. Query strings are preserved.</p>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre>curl https://markdown.go-mizu.workers.dev/https://example.com?q=hello</pre>
    </div>

    <h3><span class="mtag post">POST</span> /convert</h3>
    <p>Convert a URL and receive a structured JSON response with metadata.</p>
    <div class="cb">
      <button class="cb-copy" onclick="copyBlock(this)">copy</button>
      <pre><span class="rc"># Request body</span>
{ <span class="rk">"url"</span>: <span class="rv">"https://example.com"</span> }

<span class="rc"># Response</span>
{
  <span class="rk">"markdown"</span>: <span class="rv">"# Example Domain\\n\\n..."</span>,
  <span class="rk">"method"</span>:   <span class="rv">"primary" | "ai" | "browser"</span>,
  <span class="rk">"durationMs"</span>: 342,
  <span class="rk">"title"</span>:    <span class="rv">"Example Domain"</span>,
  <span class="rk">"tokens"</span>:   1248
}</pre>
    </div>

    <h3><span class="mtag get">GET</span> /llms.txt</h3>
    <p>Machine-readable API summary for LLM agents. Lists all endpoints, parameters, and response shapes.</p>

    <h2 id="pipeline">Pipeline</h2>
    <p>Every URL goes through up to three tiers, falling back automatically:</p>
    <ul>
      <li><strong>Tier 1 \u2014 Native Markdown:</strong> Requests with <code>Accept: text/markdown</code>. Sites that support this return structured Markdown directly.</li>
      <li><strong>Tier 2 \u2014 Workers AI:</strong> Fetches HTML and converts via Cloudflare Workers AI <code>toMarkdown()</code>.</li>
      <li><strong>Tier 3 \u2014 Browser Render:</strong> For JS-heavy SPAs. Renders in a headless browser via <code>@cloudflare/puppeteer</code>, then passes to Workers AI.</li>
    </ul>

    <h2 id="headers">Response headers</h2>
    <p>The <code>GET /{url}</code> endpoint returns these headers:</p>
    <table class="tbl">
      <thead><tr><th>Header</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>X-Conversion-Method</td><td><code>primary</code>, <code>ai</code>, or <code>browser</code></td></tr>
        <tr><td>X-Duration-Ms</td><td>Server-side processing time in milliseconds</td></tr>
        <tr><td>X-Title</td><td>Percent-encoded page title (max 200 chars)</td></tr>
        <tr><td>X-Markdown-Tokens</td><td>Approximate token count (when available)</td></tr>
        <tr><td>Cache-Control</td><td><code>public, max-age=300, s-maxage=3600, stale-while-revalidate=86400</code></td></tr>
      </tbody>
    </table>

    <h2 id="limits">Limits</h2>
    <ul>
      <li>Max response body: <strong>5 MB</strong> per URL</li>
      <li>Fetch timeout: <strong>10 seconds</strong> (30 seconds for browser rendering)</li>
      <li>Protocols: <strong>http://</strong> and <strong>https://</strong> only</li>
      <li>Rate limits: Cloudflare Workers free tier (100,000 requests/day)</li>
    </ul>
  </div>
</div>

<script>
async function copyBlock(btn) {
  var cb = btn.parentElement;
  var pre = cb.querySelector('pre');
  var text = pre ? (pre.innerText || pre.textContent) : '';
  try { await navigator.clipboard.writeText(text.trim()); }
  catch(e) {
    var ta = document.createElement('textarea');
    ta.value = text.trim();
    document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta);
  }
  btn.textContent = 'copied!';
  setTimeout(function() { btn.textContent = 'copy'; }, 2000);
}

// Highlight active sidebar link on scroll
var sections = document.querySelectorAll('h2[id]');
var links = document.querySelectorAll('.sidebar a');
window.addEventListener('scroll', function() {
  var pos = window.scrollY + 80;
  var active = sections[0];
  sections.forEach(function(s) { if (s.offsetTop <= pos) active = s; });
  links.forEach(function(l) {
    l.className = l.getAttribute('href') === '#' + (active ? active.id : '') ? 'on' : '';
  });
});
</script>
</body>
</html>`;
}
```

**Step 2: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```

**Step 3: Commit**

```bash
git add blueprints/search/tools/markdown/src/docs.ts
git commit -m "feat(markdown): add /docs page with sidebar"
```

---

## Task 4: Update `src/index.ts` — add `/preview` and `/docs` routes

**Files:**
- Modify: `blueprints/search/tools/markdown/src/index.ts`

**Step 1: Add the two new imports and routes**

At the top of the file, add:
```typescript
import { renderPreview } from './preview';
import { renderDocs } from './docs';
```

After the landing page route (`app.get('/', ...)`), add:
```typescript
// Docs page
app.get('/docs', (c) => c.html(renderDocs()));

// Preview page — client reads ?url= and calls POST /convert
app.get('/preview', (c) => c.html(renderPreview()));
```

The final route order in `index.ts` must be:
1. `GET /` → landing page
2. `GET /docs` → docs page
3. `GET /preview` → preview page
4. `GET /llms.txt` → llms.txt (already exists)
5. `POST /convert` → JSON API (already exists)
6. `OPTIONS /*` → CORS preflight (already exists)
7. `GET /*` → URL conversion (catch-all, must stay last)

**Step 2: Verify TypeScript compiles**

```bash
cd blueprints/search/tools/markdown && npx tsc --noEmit
```
Expected: no errors.

**Step 3: Deploy and verify**

```bash
cd blueprints/search/tools/markdown && npm run deploy
```

Verify:
```bash
# Landing page loads
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/
# Expected: 200

# Preview page loads
curl -s -o /dev/null -w "%{http_code}" "https://markdown.go-mizu.workers.dev/preview?url=https://example.com"
# Expected: 200

# Docs page loads
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/docs
# Expected: 200

# Convert still works
curl -s https://markdown.go-mizu.workers.dev/https://example.com | head -5
# Expected: markdown content

# POST /convert still works
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' | grep '"method"'
# Expected: "method":"primary" or "ai"
```

**Step 4: Commit**

```bash
git add blueprints/search/tools/markdown/src/index.ts
git commit -m "feat(markdown): add /docs and /preview routes"
```

---

## Verification Checklist

After all tasks complete:

- [ ] `https://markdown.go-mizu.workers.dev/` — full-width header (no border), hero with numbered steps, "Copy for agent" button works, clicking Convert navigates to `/preview?url=...`, no footer
- [ ] `https://markdown.go-mizu.workers.dev/preview?url=https://example.com` — auto-converts on load, shows method badge + duration, "View raw →" links to `/https://example.com`, Copy works, Save works, tab switching works
- [ ] `https://markdown.go-mizu.workers.dev/docs` — sidebar with anchor navigation, dark code blocks with copy buttons, scroll highlight works
- [ ] No rounded corners anywhere (inputs, buttons, badges, cards)
- [ ] Max-width 1120px on all pages
- [ ] `npx tsc --noEmit` passes with zero errors
