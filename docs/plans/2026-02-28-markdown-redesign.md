# Markdown Tool Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Full visual redesign of the markdown.go-mizu CF Worker tool — dark/light mode toggle, new logo, cleaned-up landing page content flow, preview empty state, and docs centered single-column layout.

**Architecture:** Four files change: `public/styles.css` (shared design tokens + dark mode vars), `public/index.html` (landing page rework), `public/preview.html` (empty state + dark mode), `src/docs.ts` + `src/content/docs.md` (remove sidebar, center content, add sections). All pages share one `toggleTheme()` pattern stored in `localStorage`. No new dependencies.

**Tech Stack:** HTML/CSS/JS (vanilla), Cloudflare Workers Static Assets, Hono (server side), Geist + Geist Mono fonts, Lucide icon SVG (inline).

**Spec:** `blueprints/search/spec/0627_markdown.md`

**Working directory for all commands:** `blueprints/search/tools/markdown/`

---

### Task 1: styles.css — dark mode vars + new shared components

**Files:**
- Modify: `public/styles.css`

This is the foundation. Every other task depends on these CSS variables being in place.

**Step 1: Read the current file**

```bash
cat public/styles.css
```

**Step 2: Replace the entire file with the new version**

The new `styles.css` must:
1. Keep all existing rules (reset, body, header, badges, spinner, url-in, cvt-btn)
2. Add `[data-theme="dark"]` variable overrides
3. Update badge styles to use CSS vars (neutral, works in both modes)
4. Add `.theme-toggle` button
5. Add `.site-footer`
6. Change `--bg`, `--fg`, etc. root vars to the new values

Full replacement:

```css
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --sans:'Geist',-apple-system,sans-serif;
  --mono:'Geist Mono',ui-monospace,monospace;
  --bg:#ffffff;--bg2:#f7f7f7;
  --fg:#0a0a0a;--fg2:#525252;--fg3:#a3a3a3;
  --border:#e5e5e5;--border2:#d4d4d4;
  --code-bg:#0c0c0c;--code-fg:#e4e4e7;
  --w:1120px
}
[data-theme="dark"]{
  --bg:#0a0a0a;--bg2:#111111;
  --fg:#f0f0ef;--fg2:#a3a3a3;--fg3:#525252;
  --border:#222222;--border2:#2e2e2e;
  --code-bg:#141414;--code-fg:#e4e4e7
}
html{font-size:15px}
body{font-family:var(--sans);background:var(--bg);color:var(--fg);line-height:1.6;-webkit-font-smoothing:antialiased;transition:background .2s,color .2s}
a{color:inherit;text-decoration:none}
code{font-family:var(--mono);font-size:11.5px;background:var(--bg2);padding:1px 5px}
.w{max-width:var(--w);margin:0 auto;padding:0 32px}

/* header */
header{padding:14px 32px;border-bottom:1px solid var(--border)}
.hdr{display:flex;align-items:center;justify-content:space-between}
.logo{font-size:13.5px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px;color:var(--fg)}
.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0;transition:background .2s}
nav{display:flex;align-items:center;gap:4px}
nav a{font-size:13px;color:var(--fg3);padding:6px 10px;transition:color .15s}
nav a:hover{color:var(--fg)}

/* theme toggle */
.theme-toggle{background:none;border:1px solid var(--border);color:var(--fg3);cursor:pointer;padding:6px 8px;display:flex;align-items:center;transition:color .15s,border-color .15s;margin-left:4px}
.theme-toggle:hover{color:var(--fg);border-color:var(--border2)}

/* badges */
.badge{font-family:var(--mono);font-size:11px;padding:2px 7px;white-space:nowrap;flex-shrink:0;background:var(--bg2);color:var(--fg2);border:1px solid var(--border)}
.b-native,.b-ai,.b-browser{background:var(--bg2);color:var(--fg2);border:1px solid var(--border)}
.b-dim{background:var(--bg2);color:var(--fg3);border:1px solid var(--border)}

/* spinner */
@keyframes spin{to{transform:rotate(360deg)}}
.spinner{display:inline-block;width:22px;height:22px;border:2px solid var(--border2);border-top-color:var(--fg);animation:spin .7s linear infinite}

/* url input + convert button */
.url-in{flex:1;font-family:var(--mono);font-size:14px;padding:10px 14px;border:1px solid var(--border);border-right:none;background:var(--bg);color:var(--fg);outline:none;transition:border-color .15s,background .2s,color .2s;min-width:0}
.url-in::placeholder{color:var(--fg3)}
.url-in:focus{border-color:var(--fg)}
.cvt-btn{font-family:var(--sans);font-size:14px;font-weight:500;padding:10px 22px;background:var(--fg);color:var(--bg);border:1px solid var(--fg);cursor:pointer;white-space:nowrap;transition:opacity .15s}
.cvt-btn:hover:not(:disabled){opacity:.85}
.cvt-btn:disabled{opacity:.5;cursor:default}

/* footer */
.site-footer{border-top:1px solid var(--border);padding:24px 32px;text-align:center;font-size:13px;color:var(--fg3)}
.site-footer a{color:var(--fg3);transition:color .15s}
.site-footer a:hover{color:var(--fg)}
.site-footer .sep{margin:0 8px;opacity:.4}
```

**Step 3: Verify the file looks correct**

```bash
wc -l public/styles.css
# should be ~65-70 lines
```

**Step 4: Commit**

```bash
git add public/styles.css
git commit -m "style(markdown): dark mode vars, theme toggle, neutral badges, footer"
```

---

### Task 2: Landing page full rework (`public/index.html`)

**Files:**
- Modify: `public/index.html`

This is the largest task. Rewrite the file completely. Read the current file first to understand the existing JS logic (handleSubmit, copyAgentInstructions, switchCode, copyBlock, fallbackCopy) — all of it is reused, just reorganized.

**Step 1: Read the current file to capture JS functions**

```bash
cat public/index.html
```

**Step 2: Write the new index.html**

Key structural changes:
- Theme-init script first in `<head>` (before stylesheets — prevents FOUC)
- New logo SVG: `square-m` from Lucide (rect + M path)
- Nav: `API · Docs · GitHub · [toggle]` — no "For agents", no "llms.txt"
- Hero: headline → tagline → url input → example links (no steps list, no 2-col CTA)
- Section 1 "For AI Agents": visible `<pre>` block showing the instructions text + copy button
- Section 2 "How it works": `<dl>` three-tier rows, one CORS/cache line — replaces both old sections
- Section 3 "Code examples": Shell/JS/Python/TypeScript tabs (add TS, same tab switching JS)
- Section 4 "API reference": GET + POST endpoints, remove response-headers block from GET code
- Footer: `© 2026 markdown.go-mizu · GitHub · Docs · llms.txt`
- `toggleTheme()` + `updateToggleIcon()` in script block

Full new file:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>URL → Markdown</title>
  <meta name="description" content="Convert any URL to clean Markdown. No API key. No account. Free. Built for AI agents and LLM pipelines.">
  <script>
  (function(){var t=localStorage.getItem('theme');if(t)document.documentElement.setAttribute('data-theme',t);})();
  </script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
/* landing page */
.hero{padding:80px 0 72px}
.hero h1{font-size:clamp(44px,6vw,72px);font-weight:700;letter-spacing:-.04em;line-height:1.05;margin-bottom:20px;white-space:pre-line}
.hero-tag{font-size:15px;color:var(--fg2);margin-bottom:36px}
.url-row{display:flex;max-width:640px}
.examples{margin-top:12px;font-size:13px;color:var(--fg3)}
.eg{color:var(--fg2);cursor:pointer;text-decoration:underline;text-decoration-color:var(--border2);transition:color .15s}
.eg:hover{color:var(--fg)}

/* sections */
.sec{padding:72px 0;border-top:1px solid var(--border)}
.sec h2{font-size:26px;font-weight:600;letter-spacing:-.025em;margin-bottom:8px}
.sec-sub{font-size:15px;color:var(--fg2);margin-bottom:40px;max-width:560px;line-height:1.75}

/* agent instructions block */
.agent-block{background:var(--code-bg);padding:24px 28px;max-width:680px;position:relative;margin-bottom:16px}
.agent-block pre{font-family:var(--mono);font-size:13px;line-height:1.85;color:var(--code-fg);white-space:pre;margin:0}
.agent-copy{position:absolute;top:12px;right:12px;background:#222;border:1px solid #333;color:#aaa;font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:color .15s}
.agent-copy:hover{color:#fff}

/* how it works tiers */
.tier-list{display:flex;flex-direction:column;gap:0;max-width:680px;border:1px solid var(--border)}
.tier-row{display:grid;grid-template-columns:180px 1fr;border-bottom:1px solid var(--border)}
.tier-row:last-child{border-bottom:none}
.tier-name{padding:16px 20px;font-family:var(--mono);font-size:12px;color:var(--fg3);border-right:1px solid var(--border);display:flex;align-items:center}
.tier-desc{padding:16px 20px;font-size:14px;color:var(--fg2);line-height:1.65}
.tier-desc strong{color:var(--fg);font-weight:500}
.tier-note{margin-top:16px;font-size:13px;color:var(--fg3);max-width:680px}

/* code tabs */
.cbar{display:flex;background:var(--bg2);border-bottom:1px solid var(--border);border-top:1px solid var(--border);border-left:1px solid var(--border);border-right:1px solid var(--border)}
.ctab{font-family:var(--mono);font-size:13px;padding:9px 16px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-1px;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s}
.ctab.on{color:var(--fg);border-bottom-color:var(--fg)}
.cpanel{display:none;background:var(--code-bg);padding:22px;overflow-x:auto;position:relative;border:1px solid var(--border);border-top:none}
.cpanel.on{display:block}
.cpanel pre{font-family:var(--mono);font-size:13px;line-height:1.7;color:var(--code-fg);white-space:pre;margin:0}
.c1{color:#6b7280}.c2{color:#93c5fd}.c3{color:#86efac}.c4{color:#fcd34d}
.copy-btn{position:absolute;top:12px;right:12px;background:#222;border:1px solid #333;color:#aaa;font-size:11px;font-family:var(--mono);padding:4px 10px;cursor:pointer;transition:color .15s}
.copy-btn:hover{color:#fff}

/* api endpoints */
.ep{border:1px solid var(--border);margin-bottom:16px}
.eph{padding:12px 16px;display:flex;align-items:center;gap:10px;background:var(--bg2);border-bottom:1px solid var(--border)}
.mtag{font-family:var(--mono);font-size:11px;font-weight:600;padding:3px 8px;flex-shrink:0}
.get{background:#f0fdf4;color:#15803d}
.post{background:#eff6ff;color:#1d4ed8}
[data-theme="dark"] .get{background:#052e16;color:#86efac}
[data-theme="dark"] .post{background:#0c1a3d;color:#93c5fd}
.epath{font-family:var(--mono);font-size:13px}
.edesc{font-size:12px;color:var(--fg3);margin-left:auto}
.epcode{background:var(--code-bg);padding:22px;overflow-x:auto;position:relative;border:none}
.epcode pre{font-family:var(--mono);font-size:12.5px;line-height:1.8;color:#e4e4e7;white-space:pre;margin:0}
.rk{color:#93c5fd}.rv{color:#e4e4e7}.rc{color:#6b7280}
  </style>
</head>
<body>

<header>
  <div class="hdr">
    <a href="/" class="logo">
      <span class="logo-sq">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--bg)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <path d="M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16"/>
        </svg>
      </span>
      markdown.go-mizu
    </a>
    <nav>
      <a href="#api">API</a>
      <a href="/docs">Docs</a>
      <a href="https://github.com/go-mizu/mizu">GitHub</a>
      <button class="theme-toggle" id="theme-toggle" onclick="toggleTheme()" title="Toggle dark mode">
        <svg id="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/></svg>
        <svg id="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
      </button>
    </nav>
  </div>
</header>

<main>

<div class="w">
  <section class="hero">
    <h1>Any URL →
Clean Markdown.</h1>
    <p class="hero-tag">No API key. No account. Instant.</p>
    <form id="form" onsubmit="handleSubmit(event)">
      <div class="url-row">
        <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">
        <button type="submit" class="cvt-btn">Convert</button>
      </div>
    </form>
    <div class="examples">Try: <span class="eg" onclick="setEg('https://example.com')">example.com</span> &middot; <span class="eg" onclick="setEg('https://news.ycombinator.com')">news.ycombinator.com</span> &middot; <span class="eg" onclick="setEg('https://docs.python.org/3/')">docs.python.org</span></div>
  </section>
</div>

<div class="w">
  <section class="sec" id="agents">
    <h2>For AI Agents</h2>
    <p class="sec-sub">Copy these instructions into your agent's system prompt.</p>
    <div class="agent-block">
      <button class="agent-copy" id="agent-copy-btn" onclick="copyAgentInstructions()">copy</button>
      <pre id="agent-instructions">Use https://markdown.go-mizu.workers.dev to read any URL
as clean Markdown. No auth required. Free.

Fetch any URL as Markdown text:
  GET https://markdown.go-mizu.workers.dev/{url}
  Returns: text/markdown

Fetch with JSON metadata (method, duration, title):
  POST https://markdown.go-mizu.workers.dev/convert
  Content-Type: application/json
  {"url": "https://example.com"}
  Returns: {"markdown": "...", "method": "...", "durationMs": N, "title": "..."}</pre>
    </div>
  </section>
</div>

<div class="w">
  <section class="sec" id="how">
    <h2>How it works</h2>
    <p class="sec-sub">Every URL goes through three tiers, falling back automatically until clean Markdown is produced.</p>
    <div class="tier-list">
      <div class="tier-row">
        <div class="tier-name">Tier 1 · Native</div>
        <div class="tier-desc"><strong>Accept: text/markdown</strong> negotiation — sites that natively serve Markdown return it directly.</div>
      </div>
      <div class="tier-row">
        <div class="tier-name">Tier 2 · Workers AI</div>
        <div class="tier-desc">HTML fetched and converted via <strong>Cloudflare Workers AI</strong> toMarkdown() — fast, structure-aware extraction.</div>
      </div>
      <div class="tier-row">
        <div class="tier-name">Tier 3 · Browser</div>
        <div class="tier-desc">JS-heavy SPAs rendered in a <strong>headless browser</strong> via Puppeteer before AI conversion.</div>
      </div>
    </div>
    <p class="tier-note">Responses are edge-cached for 1 hour with stale-while-revalidate. CORS enabled — fetch from any origin.</p>
  </section>
</div>

<div class="w">
  <section class="sec" id="code">
    <h2>Code examples</h2>
    <p class="sec-sub">Works with any HTTP client.</p>
    <div class="cbar">
      <button class="ctab on" id="ctab-sh" onclick="switchCode('sh')">Shell</button>
      <button class="ctab" id="ctab-js" onclick="switchCode('js')">JavaScript</button>
      <button class="ctab" id="ctab-py" onclick="switchCode('py')">Python</button>
      <button class="ctab" id="ctab-ts" onclick="switchCode('ts')">TypeScript</button>
    </div>
    <div id="cpanel-sh" class="cpanel on">
      <button class="copy-btn" onclick="copyBlock('cpanel-sh',this)">copy</button>
      <pre><span class="c1"># Returns text/markdown</span>
curl https://markdown.go-mizu.workers.dev/https://example.com

<span class="c1"># Returns structured JSON</span>
curl -s -X POST https://markdown.go-mizu.workers.dev/convert \
  -H <span class="c3">'Content-Type: application/json'</span> \
  -d <span class="c3">'{"url":"https://example.com"}'</span></pre>
    </div>
    <div id="cpanel-js" class="cpanel">
      <button class="copy-btn" onclick="copyBlock('cpanel-js',this)">copy</button>
      <pre><span class="c1">// Markdown text</span>
<span class="c2">const</span> md = <span class="c2">await</span> <span class="c4">fetch</span>(
  <span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url
).<span class="c4">then</span>(r => r.<span class="c4">text</span>());

<span class="c1">// JSON with metadata</span>
<span class="c2">const</span> res = <span class="c2">await</span> <span class="c4">fetch</span>(<span class="c3">'https://markdown.go-mizu.workers.dev/convert'</span>, {
  method: <span class="c3">'POST'</span>,
  headers: { <span class="c3">'Content-Type'</span>: <span class="c3">'application/json'</span> },
  body: JSON.<span class="c4">stringify</span>({ url })
}).<span class="c4">then</span>(r => r.<span class="c4">json</span>());</pre>
    </div>
    <div id="cpanel-py" class="cpanel">
      <button class="copy-btn" onclick="copyBlock('cpanel-py',this)">copy</button>
      <pre><span class="c2">import</span> httpx

<span class="c1"># Markdown text</span>
md = httpx.<span class="c4">get</span>(<span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url).text

<span class="c1"># JSON with metadata</span>
res = httpx.<span class="c4">post</span>(
    <span class="c3">'https://markdown.go-mizu.workers.dev/convert'</span>,
    json={<span class="c3">'url'</span>: url}
).<span class="c4">json</span>()</pre>
    </div>
    <div id="cpanel-ts" class="cpanel">
      <button class="copy-btn" onclick="copyBlock('cpanel-ts',this)">copy</button>
      <pre><span class="c2">const</span> md = <span class="c2">await</span> <span class="c4">fetch</span>(
  <span class="c3">'https://markdown.go-mizu.workers.dev/'</span> + url
).<span class="c4">then</span>(r => r.<span class="c4">text</span>());

<span class="c2">interface</span> ConvertResult {
  markdown: <span class="c3">string</span>;
  method: <span class="c3">'primary'</span> | <span class="c3">'ai'</span> | <span class="c3">'browser'</span>;
  durationMs: <span class="c3">number</span>;
  title: <span class="c3">string</span>;
  tokens?: <span class="c3">number</span>;
}
<span class="c2">const</span> res = <span class="c2">await</span> <span class="c4">fetch</span>(<span class="c3">'https://markdown.go-mizu.workers.dev/convert'</span>, {
  method: <span class="c3">'POST'</span>,
  headers: { <span class="c3">'Content-Type'</span>: <span class="c3">'application/json'</span> },
  body: JSON.<span class="c4">stringify</span>({ url })
}).<span class="c4">then</span>(r => r.<span class="c4">json</span>() <span class="c2">as</span> Promise&lt;ConvertResult&gt;);</pre>
    </div>
  </section>
</div>

<div class="w">
  <section class="sec" id="api">
    <h2>API reference</h2>

    <div class="ep">
      <div class="eph">
        <span class="mtag get">GET</span>
        <span class="epath">/{url}</span>
        <span class="edesc">Returns text/markdown</span>
      </div>
      <div class="epcode">
        <button class="copy-btn" onclick="copyBlock('ep-get',this)">copy</button>
        <pre id="ep-get"><span class="rc"># Append any absolute URL (http:// or https://)</span>
curl https://markdown.go-mizu.workers.dev/https://example.com</pre>
      </div>
    </div>

    <div class="ep">
      <div class="eph">
        <span class="mtag post">POST</span>
        <span class="epath">/convert</span>
        <span class="edesc">Returns JSON</span>
      </div>
      <div class="epcode">
        <button class="copy-btn" onclick="copyBlock('ep-post',this)">copy</button>
        <pre id="ep-post"><span class="rc"># Request</span>
curl -X POST https://markdown.go-mizu.workers.dev/convert \
  -H <span class="rv">'Content-Type: application/json'</span> \
  -d <span class="rv">'{"url":"https://example.com"}'</span>

<span class="rc"># Response</span>
{
  <span class="rk">"markdown"</span>: <span class="rv">"# Example Domain\n\n..."</span>,
  <span class="rk">"method"</span>: <span class="rv">"ai"</span>,
  <span class="rk">"durationMs"</span>: <span class="rv">342</span>,
  <span class="rk">"title"</span>: <span class="rv">"Example Domain"</span>,
  <span class="rk">"tokens"</span>: <span class="rv">1248</span>
}</pre>
      </div>
    </div>
  </section>
</div>

</main>

<footer class="site-footer">
  <span>© 2026 markdown.go-mizu</span>
  <span class="sep">·</span><a href="https://github.com/go-mizu/mizu">GitHub</a>
  <span class="sep">·</span><a href="/docs">Docs</a>
  <span class="sep">·</span><a href="/llms.txt">llms.txt</a>
</footer>

<script>
var AGENT_INSTRUCTIONS = document.getElementById('agent-instructions').textContent;

function handleSubmit(e) {
  e.preventDefault();
  var url = document.getElementById('url-in').value.trim();
  if (url) window.location.href = '/preview?url=' + encodeURIComponent(url);
}

function setEg(url) {
  document.getElementById('url-in').value = url;
  handleSubmit({ preventDefault: function() {} });
}

function copyAgentInstructions() {
  var btn = document.getElementById('agent-copy-btn');
  function done() {
    btn.textContent = 'copied!';
    setTimeout(function() { btn.textContent = 'copy'; }, 2000);
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(AGENT_INSTRUCTIONS.trim()).then(done).catch(function() { fallbackCopy(AGENT_INSTRUCTIONS.trim()); done(); });
  } else {
    fallbackCopy(AGENT_INSTRUCTIONS.trim());
    done();
  }
}

function fallbackCopy(text) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.focus();
  ta.select();
  document.execCommand('copy');
  document.body.removeChild(ta);
}

function switchCode(lang) {
  ['sh','js','py','ts'].forEach(function(k) {
    document.getElementById('cpanel-'+k).className = 'cpanel'+(k===lang?' on':'');
    document.getElementById('ctab-'+k).className = 'ctab'+(k===lang?' on':'');
  });
}

function copyBlock(panelId, btn) {
  var el = document.getElementById(panelId);
  if (!el) return;
  var target = el.querySelector('pre') || el;
  var text = target.innerText || target.textContent || '';
  var origText = btn.textContent;
  function done() {
    btn.textContent = 'copied!';
    setTimeout(function() { btn.textContent = origText; }, 2000);
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(done).catch(function() { fallbackCopy(text); done(); });
  } else {
    fallbackCopy(text);
    done();
  }
}

function updateToggleIcon() {
  var dark = document.documentElement.getAttribute('data-theme') === 'dark';
  document.getElementById('icon-moon').style.display = dark ? 'none' : '';
  document.getElementById('icon-sun').style.display = dark ? '' : 'none';
}

function toggleTheme() {
  var cur = document.documentElement.getAttribute('data-theme');
  var next = cur === 'dark' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', next);
  localStorage.setItem('theme', next);
  updateToggleIcon();
}

updateToggleIcon();
</script>
</body>
</html>
```

**Step 3: Verify by opening in browser (local wrangler dev)**

```bash
npx wrangler dev --port 8787
# Open http://localhost:8787
# Check: hero, agent block, how-it-works tier rows, code tabs (4), API endpoints, footer
# Check: dark mode toggle works, persists on page reload
```

**Step 4: Commit**

```bash
git add public/index.html
git commit -m "feat(markdown): rework landing — agent section, tier rows, TS tab, dark mode"
```

---

### Task 3: Preview page (`public/preview.html`)

**Files:**
- Modify: `public/preview.html`

Changes from current file:
1. Add theme-init script in `<head>` (first script)
2. Replace logo SVG with `square-m`
3. Add dark mode toggle button to the right end of the top bar
4. Add empty state `<div id="empty-state">` shown when no `?url=` param
5. Update badge class assignment — remove emoji from method labels, use neutral badge
6. Add `toggleTheme()` + `updateToggleIcon()` + theme init to script block
7. Apply `--bg`/`--fg` vars to all inline styles that were previously hardcoded

**Step 1: Write the new preview.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Preview — markdown.go-mizu</title>
  <script>
  (function(){var t=localStorage.getItem('theme');if(t)document.documentElement.setAttribute('data-theme',t);})();
  </script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script src="https://cdn.jsdelivr.net/npm/marked@15/marked.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/dompurify@3/dist/purify.min.js"></script>
  <link rel="stylesheet" href="/styles.css">
  <style>
.top-bar{width:100%;border-bottom:1px solid var(--border);padding:10px 20px;display:flex;align-items:center;gap:12px;background:var(--bg)}
.top-bar .logo-sq{flex-shrink:0}
.url-form{flex:1;display:flex;max-width:860px}

/* empty state */
.empty-state{display:none;padding:100px 32px;text-align:center}
.empty-icon{margin:0 auto 20px;width:48px;height:48px;background:var(--fg3);mask-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Crect x='3' y='3' width='18' height='18' rx='2'/%3E%3Cpath d='M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16'/%3E%3C/svg%3E");mask-size:contain;mask-repeat:no-repeat;mask-position:center;-webkit-mask-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Crect x='3' y='3' width='18' height='18' rx='2'/%3E%3Cpath d='M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16'/%3E%3C/svg%3E");-webkit-mask-size:contain;-webkit-mask-repeat:no-repeat;-webkit-mask-position:center}
.empty-txt{font-size:14px;color:var(--fg3);margin-bottom:14px}
.empty-links{font-size:13px;color:var(--fg3)}
.empty-eg{color:var(--fg2);cursor:pointer;text-decoration:underline;text-decoration-color:var(--border2);transition:color .15s}
.empty-eg:hover{color:var(--fg)}

/* loading / error states */
.loading-state{display:none;padding:80px 32px;text-align:center}
.loading-txt{font-size:14px;color:var(--fg3);vertical-align:middle;margin-left:10px}
.error-state{display:none;padding:32px}
.error-box{max-width:var(--w);margin:0 auto;background:var(--bg2);border:1px solid var(--border2);padding:16px 20px;font-size:14px;color:var(--fg2)}
.result-state{display:none}

/* meta bar */
.meta-bar{width:100%;padding:12px 32px;border-bottom:1px solid var(--border);background:var(--bg)}
.meta-inner{max-width:var(--w);margin:0 auto;display:flex;align-items:center;gap:10px}
.r-title{flex:1;font-size:14px;font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--fg)}
.r-raw{font-size:13px;color:var(--fg3);white-space:nowrap;flex-shrink:0;text-decoration:underline;text-decoration-color:var(--border2)}
.r-raw:hover{color:var(--fg)}

/* tab bar */
.tab-bar{width:100%;padding:0 32px;border-bottom:1px solid var(--border);background:var(--bg)}
.tab-bar-inner{max-width:var(--w);margin:0 auto;display:flex;align-items:center;justify-content:space-between}
.tabs{display:flex}
.tab{font-size:13px;padding:12px 16px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-1px;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s;font-family:var(--sans)}
.tab.on{color:var(--fg);border-bottom-color:var(--fg)}
.tab-actions{display:flex;gap:8px}
.act-btn{font-family:var(--sans);font-size:13px;padding:7px 14px;background:var(--bg);color:var(--fg2);border:1px solid var(--border);cursor:pointer;transition:border-color .15s,color .15s}
.act-btn:hover{border-color:var(--border2);color:var(--fg)}

/* panels */
.panel{display:none}
.panel.on{display:block}
.md-panel{padding:32px;background:var(--bg)}
.pv-panel{padding:32px;background:var(--bg)}
.md-wrap{max-width:var(--w);margin:0 auto}
.pv-wrap{max-width:860px;margin:0 auto}
#md-out{font-family:var(--mono);font-size:13.5px;line-height:1.75;white-space:pre-wrap;color:var(--fg)}

/* prose preview */
#prev-out h1{font-size:1.6em;font-weight:600;margin-bottom:16px;letter-spacing:-.02em;color:var(--fg)}
#prev-out h2{font-size:1.3em;font-weight:600;margin:24px 0 12px;color:var(--fg)}
#prev-out h3{font-size:1.1em;font-weight:600;margin:20px 0 10px;color:var(--fg)}
#prev-out h4,#prev-out h5,#prev-out h6{font-weight:600;margin:16px 0 8px;color:var(--fg)}
#prev-out p{color:var(--fg2);line-height:1.75;margin-bottom:14px}
#prev-out ul,#prev-out ol{padding-left:24px;margin-bottom:14px}
#prev-out li{line-height:1.7;color:var(--fg2)}
#prev-out code{font-family:var(--mono);font-size:12px;background:var(--bg2);padding:1px 5px}
#prev-out pre{background:var(--code-bg);padding:20px;overflow-x:auto;margin-bottom:16px}
#prev-out pre code{background:none;padding:0;color:var(--code-fg);font-size:13px}
#prev-out blockquote{border-left:3px solid var(--border2);padding-left:16px;color:var(--fg3);margin-bottom:14px}
#prev-out a{color:var(--fg);text-decoration:underline;text-decoration-color:var(--border2)}
#prev-out a:hover{text-decoration-color:var(--fg)}
#prev-out table{width:100%;border-collapse:collapse;margin-bottom:16px;font-size:14px}
#prev-out th,#prev-out td{border:1px solid var(--border2);padding:8px 12px;text-align:left;color:var(--fg2)}
#prev-out th{background:var(--bg2);font-weight:600;color:var(--fg)}
#prev-out hr{border:none;border-top:1px solid var(--border);margin:24px 0}
#prev-out img{max-width:100%;height:auto}
  </style>
</head>
<body>

<div class="top-bar">
  <a href="/" class="logo-sq" title="markdown.go-mizu">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--bg)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect x="3" y="3" width="18" height="18" rx="2"/>
      <path d="M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16"/>
    </svg>
  </a>
  <form class="url-form" id="form" onsubmit="handleSubmit(event)">
    <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">
    <button type="submit" id="sub-btn" class="cvt-btn"><span id="btn-sp" style="display:none" class="spinner"></span><span id="btn-t">Convert</span></button>
  </form>
  <button class="theme-toggle" id="theme-toggle" onclick="toggleTheme()" title="Toggle dark mode">
    <svg id="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/></svg>
    <svg id="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
  </button>
</div>

<div class="empty-state" id="empty-state">
  <div class="empty-icon"></div>
  <p class="empty-txt">Enter a URL above to preview its Markdown output.</p>
  <div class="empty-links">Try: <span class="empty-eg" onclick="setEg('https://example.com')">example.com</span> &middot; <span class="empty-eg" onclick="setEg('https://news.ycombinator.com')">news.ycombinator.com</span> &middot; <span class="empty-eg" onclick="setEg('https://docs.python.org/3/')">docs.python.org</span></div>
</div>

<div class="loading-state" id="loading-state">
  <span class="spinner" style="vertical-align:middle"></span>
  <span class="loading-txt">Converting…</span>
</div>

<div class="error-state" id="error-state">
  <div class="error-box" id="err-msg"></div>
</div>

<div class="result-state" id="result-state">
  <div class="meta-bar">
    <div class="meta-inner">
      <span class="r-title" id="r-title"></span>
      <span class="badge b-dim" id="r-method"></span>
      <span class="badge b-dim" id="r-dur"></span>
      <span class="badge b-dim" id="r-tok" style="display:none"></span>
      <a class="r-raw" id="r-raw" href="#" target="_blank" rel="noopener">raw ↗</a>
    </div>
  </div>
  <div class="tab-bar">
    <div class="tab-bar-inner">
      <div class="tabs">
        <button class="tab on" id="tab-md" onclick="switchTab('md')">Markdown</button>
        <button class="tab" id="tab-pv" onclick="switchTab('pv')">Preview</button>
      </div>
      <div class="tab-actions">
        <button class="act-btn" onclick="copyMd()"><span id="copy-lbl">Copy</span></button>
        <button class="act-btn" onclick="saveMd()">Save .md</button>
      </div>
    </div>
  </div>
  <div id="panel-md" class="panel md-panel on">
    <div class="md-wrap"><pre id="md-out"></pre></div>
  </div>
  <div id="panel-pv" class="panel pv-panel">
    <div class="pv-wrap"><div id="prev-out"></div></div>
  </div>
</div>

<script>
var md = '';
var currentUrl = '';

function handleSubmit(e) {
  e.preventDefault();
  var url = document.getElementById('url-in').value.trim();
  if (!url) return;
  history.replaceState(null, '', '/preview?url=' + encodeURIComponent(url));
  convertUrl(url);
}

function setEg(url) {
  document.getElementById('url-in').value = url;
  document.getElementById('url-in').focus();
  convertUrl(url);
}

function setLoading(v) {
  document.getElementById('sub-btn').disabled = v;
  document.getElementById('btn-t').textContent = v ? 'Converting…' : 'Convert';
  document.getElementById('btn-sp').style.display = v ? '' : 'none';
  document.getElementById('loading-state').style.display = v ? 'block' : 'none';
  document.getElementById('empty-state').style.display = 'none';
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
    showResult(await r.json());
  } catch(e) {
    document.getElementById('err-msg').textContent = e.message || 'Conversion failed';
    document.getElementById('error-state').style.display = 'block';
  } finally {
    setLoading(false);
  }
}

function showResult(data) {
  md = (data.markdown || '').replace(/^[-*_]{3,}\s*$/gm, '').replace(/\n{3,}/g, '\n\n').trim();
  document.getElementById('r-title').textContent = data.title || currentUrl;
  document.getElementById('r-method').textContent = data.method || '';
  document.getElementById('r-dur').textContent = data.durationMs + 'ms';
  var tb = document.getElementById('r-tok');
  if (data.tokens) { tb.textContent = '~' + data.tokens.toLocaleString() + ' tok'; tb.style.display = ''; }
  else { tb.style.display = 'none'; }
  document.getElementById('r-raw').href = '/' + currentUrl;
  document.getElementById('md-out').textContent = md;
  var parsed = marked.parse(md);
  if (typeof parsed !== 'string') return;
  document.getElementById('prev-out').innerHTML = DOMPurify.sanitize(parsed);
  document.getElementById('result-state').style.display = 'block';
  switchTab('md');
}

function switchTab(t) {
  ['md','pv'].forEach(function(k) {
    document.getElementById('panel-'+k).className = 'panel'+(k===t?' on':'')+(k==='md'?' md-panel':' pv-panel');
    document.getElementById('tab-'+k).className = 'tab'+(k===t?' on':'');
  });
}

async function copyMd() {
  try { await navigator.clipboard.writeText(md); }
  catch(e) {
    var ta = document.createElement('textarea');
    ta.value = md; document.body.appendChild(ta); ta.select();
    document.execCommand('copy'); document.body.removeChild(ta);
  }
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
  document.body.appendChild(a); a.click(); document.body.removeChild(a);
  setTimeout(function() { URL.revokeObjectURL(a.href); }, 100);
}

function updateToggleIcon() {
  var dark = document.documentElement.getAttribute('data-theme') === 'dark';
  document.getElementById('icon-moon').style.display = dark ? 'none' : '';
  document.getElementById('icon-sun').style.display = dark ? '' : 'none';
}

function toggleTheme() {
  var cur = document.documentElement.getAttribute('data-theme');
  var next = cur === 'dark' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', next);
  localStorage.setItem('theme', next);
  updateToggleIcon();
}

window.addEventListener('load', function() {
  updateToggleIcon();
  var params = new URLSearchParams(window.location.search);
  var url = params.get('url');
  if (url) {
    convertUrl(url);
  } else {
    document.getElementById('empty-state').style.display = 'block';
  }
});
</script>
</body>
</html>
```

**Step 2: Verify empty state**

```bash
# With wrangler dev running:
# Open http://localhost:8787/preview — should show empty state with icon + message + example links
# Open http://localhost:8787/preview?url=https://example.com — should auto-convert
# Toggle dark mode — all surfaces should flip correctly
```

**Step 3: Commit**

```bash
git add public/preview.html
git commit -m "feat(markdown): preview empty state, dark mode, neutral badges, square-m logo"
```

---

### Task 4: Docs page — centered layout + content improvements

**Files:**
- Modify: `src/docs.ts`
- Modify: `src/content/docs.md`

**Step 1: Rewrite `src/content/docs.md`**

Replace the entire file with this version (fenced code blocks, two new sections):

````markdown
# URL → Markdown

Free, instant URL-to-Markdown conversion for AI agents and LLM pipelines. No API key, no account.

## Overview

Convert any HTTP/HTTPS URL to clean, structured Markdown with a single request.

- Works with any HTTP/HTTPS URL
- Three-tier pipeline: native negotiation → Workers AI → Browser rendering
- Edge-cached for 1 hour with stale-while-revalidate
- CORS-enabled — fetch from any origin, no proxy needed

## Quick start

Fetch as Markdown:

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com
```

Use the JSON API:

```bash
curl -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

JavaScript:

```javascript
const md = await fetch(
  'https://markdown.go-mizu.workers.dev/' + url
).then(r => r.text());
```

Python:

```python
import httpx
md = httpx.get('https://markdown.go-mizu.workers.dev/' + url).text
```

## GET /{url}

Convert a URL to Markdown. Append any `http://` or `https://` URL to the worker base URL. Query strings are preserved.

```bash
curl https://markdown.go-mizu.workers.dev/https://example.com?q=hello
```

## POST /convert

Convert a URL and receive a structured JSON response.

Request body:

```json
{"url": "https://example.com"}
```

Response:

```json
{
  "markdown": "# Example Domain\n\n...",
  "method": "primary",
  "durationMs": 342,
  "title": "Example Domain",
  "tokens": 1248
}
```

`method` is one of `primary`, `ai`, or `browser`.

## Conversion pipeline

Every URL goes through up to three tiers, falling back automatically:

- **Tier 1 — Native:** Requests with `Accept: text/markdown`. Sites that support this return structured Markdown directly.
- **Tier 2 — Workers AI:** Fetches HTML and converts via Cloudflare Workers AI `toMarkdown()`.
- **Tier 3 — Browser:** For JS-heavy SPAs. Renders in a headless browser via Puppeteer, then passes to Workers AI.

## Response headers

The `GET /{url}` endpoint returns these headers:

| Header | Description |
|---|---|
| `X-Conversion-Method` | `primary`, `ai`, or `browser` |
| `X-Duration-Ms` | Server-side processing time in milliseconds |
| `X-Title` | Percent-encoded page title (max 200 chars) |
| `X-Markdown-Tokens` | Approximate token count (when available) |
| `Cache-Control` | `public, max-age=300, s-maxage=3600, stale-while-revalidate=86400` |

## Error responses

| Status | When |
|---|---|
| `400` | Missing or invalid `url` field in POST body |
| `422` | Conversion failed (fetch error, unsupported content) |

Error body for `POST /convert`:

```json
{"error": "description of what went wrong"}
```

The `GET /{url}` endpoint returns plain text: `Error: description`

## CORS

All endpoints return `Access-Control-Allow-Origin: *`. You can call the API directly from browser JavaScript with no proxy needed.

```javascript
// Works in browser — no CORS errors
const md = await fetch(
  'https://markdown.go-mizu.workers.dev/' + url
).then(r => r.text());
```

## Limits

- Max response body: **5 MB** per URL
- Fetch timeout: **10 seconds** (30 seconds for browser rendering)
- Protocols: **http://** and **https://** only
- Rate limits: Cloudflare Workers free tier (100,000 requests/day)
````

**Step 2: Rewrite `src/docs.ts`**

Remove the sidebar entirely. Center the content. Add theme toggle. Use `styles.css`.

```typescript
export function renderDocs(contentHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs — markdown.go-mizu</title>
  <script>
  (function(){var t=localStorage.getItem('theme');if(t)document.documentElement.setAttribute('data-theme',t);})();
  <\/script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  .doc-wrap{max-width:720px;margin:0 auto;padding:48px 32px 80px}
  @media(max-width:640px){.doc-wrap{padding:32px 20px 60px}}
  /* rendered markdown */
  .doc-wrap h1{font-size:32px;font-weight:700;letter-spacing:-.03em;margin-bottom:12px;color:var(--fg)}
  .doc-wrap>p:first-of-type{font-size:16px;color:var(--fg2);margin-bottom:40px;line-height:1.7}
  .doc-wrap h2{font-size:20px;font-weight:600;letter-spacing:-.02em;margin:48px 0 14px;scroll-margin-top:24px;color:var(--fg);padding-top:48px;border-top:1px solid var(--border)}
  .doc-wrap h2:first-of-type{margin-top:0;padding-top:0;border-top:none}
  .doc-wrap h3{font-size:16px;font-weight:600;margin:28px 0 10px;color:var(--fg)}
  .doc-wrap p{font-size:15px;color:var(--fg2);line-height:1.75;margin-bottom:14px}
  .doc-wrap ul,.doc-wrap ol{padding-left:1.5em;margin-bottom:16px}
  .doc-wrap li{font-size:15px;color:var(--fg2);margin:4px 0;line-height:1.7}
  .doc-wrap strong{color:var(--fg);font-weight:600}
  .doc-wrap a{color:var(--fg);text-decoration:underline;text-underline-offset:2px;text-decoration-color:var(--border2)}
  .doc-wrap a:hover{text-decoration-color:var(--fg)}
  .doc-wrap code{font-family:var(--mono);font-size:12px;background:var(--bg2);padding:2px 6px;color:var(--fg)}
  .doc-wrap pre{background:var(--code-bg);padding:20px 22px;overflow-x:auto;margin:16px 0;position:relative}
  .doc-wrap pre code{font-family:var(--mono);font-size:13px;line-height:1.7;color:var(--code-fg);background:none;padding:0}
  .doc-wrap table{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
  .doc-wrap th{text-align:left;padding:8px 14px;border-bottom:1px solid var(--border);font-weight:600;color:var(--fg)}
  .doc-wrap td{padding:8px 14px;border-bottom:1px solid var(--border2);color:var(--fg2)}
  .doc-wrap td:first-child code{font-size:12px}
  /* copy buttons */
  .doc-wrap pre .copy-btn{position:absolute;top:10px;right:10px;background:var(--bg2);border:1px solid var(--border);color:var(--fg3);font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:all .15s}
  .doc-wrap pre .copy-btn:hover{color:var(--fg)}
  </style>
</head>
<body>
<header>
  <div class="hdr">
    <a href="/" class="logo">
      <span class="logo-sq">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--bg)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <path d="M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16"/>
        </svg>
      </span>
      markdown.go-mizu
    </a>
    <nav>
      <a href="/">Home</a>
      <a href="https://github.com/go-mizu/mizu">GitHub</a>
      <button class="theme-toggle" id="theme-toggle" onclick="toggleTheme()" title="Toggle dark mode">
        <svg id="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/></svg>
        <svg id="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
      </button>
    </nav>
  </div>
</header>

<div class="doc-wrap">
  ${contentHtml}
</div>

<script>
(function() {
  // Copy buttons on pre blocks
  document.querySelectorAll('.doc-wrap pre').forEach(function(pre) {
    var btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'copy';
    btn.onclick = function() {
      var text = (pre.querySelector('code') || pre).innerText || '';
      var self = btn;
      function done() { self.textContent = 'copied!'; setTimeout(function() { self.textContent = 'copy'; }, 2000); }
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text.trim()).then(done).catch(done);
      } else {
        var ta = document.createElement('textarea');
        ta.style.position='fixed';ta.style.top='-9999px';ta.value=text.trim();
        document.body.appendChild(ta);ta.select();document.execCommand('copy');document.body.removeChild(ta);
        done();
      }
    };
    pre.style.position = 'relative';
    pre.appendChild(btn);
  });
})();

function updateToggleIcon() {
  var dark = document.documentElement.getAttribute('data-theme') === 'dark';
  document.getElementById('icon-moon').style.display = dark ? 'none' : '';
  document.getElementById('icon-sun').style.display = dark ? '' : 'none';
}

function toggleTheme() {
  var cur = document.documentElement.getAttribute('data-theme');
  var next = cur === 'dark' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', next);
  localStorage.setItem('theme', next);
  updateToggleIcon();
}

updateToggleIcon();
<\/script>
</body>
</html>`;
}
```

**Step 3: Verify TypeScript compiles**

```bash
npx tsc --noEmit
# Expected: no errors
```

**Step 4: Commit**

```bash
git add src/docs.ts src/content/docs.md
git commit -m "feat(markdown): docs centered layout, fenced code blocks, error + CORS sections, dark mode"
```

---

### Task 5: Deploy and verify

**Step 1: Build check**

```bash
npx tsc --noEmit
# Expected: no errors
```

**Step 2: Deploy**

```bash
npm run deploy
```

Expected output includes:
```
✨ Success! Uploaded 3 files
Deployed markdown triggers
  https://markdown.go-mizu.workers.dev
```

**Step 3: Verify each page**

```bash
# Landing — check status
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/
# Expected: 200

# Preview — check status
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/preview
# Expected: 200

# Docs — check status
curl -s -o /dev/null -w "%{http_code}" https://markdown.go-mizu.workers.dev/docs
# Expected: 200

# API — still works
curl -s -o /dev/null -w "%{http_code}" -X POST https://markdown.go-mizu.workers.dev/convert \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
# Expected: 200
```

**Step 4: Manual checks**

- `/` — dark mode toggle visible in nav, persists on reload, all 4 sections present
- `/preview` — empty state shows with no URL param; `?url=https://example.com` auto-converts; dark mode synced
- `/docs` — content centered, no sidebar, code blocks have copy buttons, dark mode works
- Navigate `/` → `/preview` → `/docs` → `/` — theme persists throughout

**Step 5: Commit**

```bash
git add -A
git commit -m "chore(markdown): deploy 0627 — dark mode, redesign, docs rewrite"
```
