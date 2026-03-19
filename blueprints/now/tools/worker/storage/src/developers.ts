import { esc } from "./layout";

export function developersPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  /* -- Signed-in hero -------------------------------------------------- */
  const heroSignedIn = `
<div class="hero-greeting">Welcome back, <strong>${displayName}</strong></div>
<div class="hero-actions">
  <a href="/browse" class="btn btn--primary">
    <span class="btn-icon">&gt;_</span> Open dashboard
  </a>
  <a href="/docs" class="btn btn--ghost">Documentation</a>
</div>`;

  /* -- Signed-out hero ------------------------------------------------- */
  const heroSignedOut = `
<div class="hero-badge">DEVELOPER PLATFORM</div>
<h1 class="hero-title">Build with<br><span class="grad">storage.now</span></h1>
<p class="hero-sub">36 REST endpoints. Zero egress. Global edge. MCP-native.<br>The file storage API that gets out of your way.</p>
<div class="hero-ctas">
  <a href="#get-started" class="btn btn--primary btn--lg" onclick="document.getElementById('email-input')?.focus();return false">Get API key</a>
  <a href="/docs" class="btn btn--ghost btn--lg">Read the docs &rarr;</a>
</div>`;

  /* -- Stats strip ----------------------------------------------------- */
  const statsStrip = `
<div class="stats">
  <div class="stat">
    <div class="stat-val">300+</div>
    <div class="stat-label">Edge locations</div>
  </div>
  <div class="stat">
    <div class="stat-val">&lt;50ms</div>
    <div class="stat-label">Metadata latency</div>
  </div>
  <div class="stat">
    <div class="stat-val">$0</div>
    <div class="stat-label">Egress fees</div>
  </div>
  <div class="stat">
    <div class="stat-val">5 GB</div>
    <div class="stat-label">Free storage</div>
  </div>
</div>`;

  /* -- Terminal demo --------------------------------------------------- */
  const terminalDemo = `
<div class="term">
  <div class="term-bar">
    <div class="term-dots"><span></span><span></span><span></span></div>
    <div class="term-title">storage.now</div>
  </div>
  <div class="term-body" id="term-body">
    <div class="term-line"><span class="t-prompt">$</span> <span class="t-cmd">curl -X PUT</span> <span class="t-url">/files/models/v3.bin</span> <span class="t-flag">\\</span></div>
    <div class="term-line"><span class="t-indent"></span><span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer sk_..."</span> <span class="t-flag">\\</span></div>
    <div class="term-line"><span class="t-indent"></span><span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/octet-stream"</span> <span class="t-flag">\\</span></div>
    <div class="term-line"><span class="t-indent"></span><span class="t-flag">--data-binary</span> <span class="t-str">@model-v3.bin</span></div>
    <div class="term-line term-line--res"><span class="t-json">{</span></div>
    <div class="term-line term-line--res"><span class="t-indent"></span><span class="t-key">"id"</span><span class="t-json">:</span> <span class="t-val">"o_7f2a9c..."</span><span class="t-json">,</span></div>
    <div class="term-line term-line--res"><span class="t-indent"></span><span class="t-key">"path"</span><span class="t-json">:</span> <span class="t-val">"models/v3.bin"</span><span class="t-json">,</span></div>
    <div class="term-line term-line--res"><span class="t-indent"></span><span class="t-key">"size"</span><span class="t-json">:</span> <span class="t-num">47185920</span><span class="t-json">,</span></div>
    <div class="term-line term-line--res"><span class="t-indent"></span><span class="t-key">"content_type"</span><span class="t-json">:</span> <span class="t-val">"application/octet-stream"</span></div>
    <div class="term-line term-line--res"><span class="t-json">}</span></div>
    <div class="term-line term-blank"></div>
    <div class="term-line"><span class="t-prompt">$</span> <span class="t-cmd">curl -X POST</span> <span class="t-url">/shares</span> <span class="t-flag">\\</span></div>
    <div class="term-line"><span class="t-indent"></span><span class="t-flag">-d</span> <span class="t-str">'{"path":"models/v3.bin","grantee":"a/inference","permission":"read"}'</span></div>
    <div class="term-line term-line--res"><span class="t-json">{</span> <span class="t-key">"id"</span><span class="t-json">:</span> <span class="t-val">"sh_9ea9..."</span><span class="t-json">,</span> <span class="t-key">"permission"</span><span class="t-json">:</span> <span class="t-val">"read"</span> <span class="t-json">}</span></div>
    <div class="term-line term-blank"></div>
    <div class="term-line"><span class="t-prompt">$</span> <span class="t-cursor"></span></div>
  </div>
</div>`;

  /* -- Code examples --------------------------------------------------- */
  const codeExamples = `
<div class="code-tabs">
  <button class="code-tab active" data-lang="curl">curl</button>
  <button class="code-tab" data-lang="js">JavaScript</button>
  <button class="code-tab" data-lang="python">Python</button>
</div>
<div class="code-panels">
  <pre class="code-panel active" id="lang-curl"><span class="t-prompt">$</span> <span class="t-cmd">curl -X PUT</span> <span class="t-url">https://storage.liteio.dev/files/hello.txt</span> <span class="t-flag">\\</span>
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer sk_..."</span> <span class="t-flag">\\</span>
    <span class="t-flag">-H</span> <span class="t-str">"Content-Type: text/plain"</span> <span class="t-flag">\\</span>
    <span class="t-flag">-d</span> <span class="t-str">"Hello, world!"</span>

<span class="t-dim">&rarr; 201 Created</span>
<span class="t-json">{</span>
  <span class="t-key">"id"</span><span class="t-json">:</span> <span class="t-val">"o_a1b2c3"</span><span class="t-json">,</span>
  <span class="t-key">"path"</span><span class="t-json">:</span> <span class="t-val">"hello.txt"</span><span class="t-json">,</span>
  <span class="t-key">"name"</span><span class="t-json">:</span> <span class="t-val">"hello.txt"</span><span class="t-json">,</span>
  <span class="t-key">"size"</span><span class="t-json">:</span> <span class="t-num">13</span>
<span class="t-json">}</span></pre>
  <pre class="code-panel" id="lang-js"><span class="t-key">const</span> res <span class="t-json">=</span> <span class="t-key">await</span> <span class="t-cmd">fetch</span><span class="t-json">(</span><span class="t-str">"https://storage.liteio.dev/files/hello.txt"</span><span class="t-json">,</span> <span class="t-json">{</span>
  <span class="t-key">method</span><span class="t-json">:</span> <span class="t-str">"PUT"</span><span class="t-json">,</span>
  <span class="t-key">headers</span><span class="t-json">:</span> <span class="t-json">{</span>
    <span class="t-str">"Authorization"</span><span class="t-json">:</span> <span class="t-str">"Bearer sk_..."</span><span class="t-json">,</span>
    <span class="t-str">"Content-Type"</span><span class="t-json">:</span> <span class="t-str">"text/plain"</span><span class="t-json">,</span>
  <span class="t-json">},</span>
  <span class="t-key">body</span><span class="t-json">:</span> <span class="t-str">"Hello, world!"</span><span class="t-json">,</span>
<span class="t-json">});</span>

<span class="t-key">const</span> file <span class="t-json">=</span> <span class="t-key">await</span> res<span class="t-json">.</span><span class="t-cmd">json</span><span class="t-json">();</span>
<span class="t-dim">// &rarr; { id: "o_a1b2c3", path: "hello.txt", size: 13 }</span></pre>
  <pre class="code-panel" id="lang-python"><span class="t-key">import</span> <span class="t-cmd">requests</span>

res <span class="t-json">=</span> requests<span class="t-json">.</span><span class="t-cmd">put</span><span class="t-json">(</span>
    <span class="t-str">"https://storage.liteio.dev/files/hello.txt"</span><span class="t-json">,</span>
    <span class="t-key">headers</span><span class="t-json">={</span>
        <span class="t-str">"Authorization"</span><span class="t-json">:</span> <span class="t-str">"Bearer sk_..."</span><span class="t-json">,</span>
        <span class="t-str">"Content-Type"</span><span class="t-json">:</span> <span class="t-str">"text/plain"</span><span class="t-json">,</span>
    <span class="t-json">},</span>
    <span class="t-key">data</span><span class="t-json">=</span><span class="t-str">"Hello, world!"</span><span class="t-json">,</span>
<span class="t-json">)</span>

file <span class="t-json">=</span> res<span class="t-json">.</span><span class="t-cmd">json</span><span class="t-json">()</span>
<span class="t-dim"># &rarr; {"id": "o_a1b2c3", "path": "hello.txt", "size": 13}</span></pre>
</div>`;

  /* -- Feature tabs ---------------------------------------------------- */
  const featureTabs = `
<div class="feat-tabs" id="feat-tabs">
  <button class="feat-tab active" data-tab="upload">Upload</button>
  <button class="feat-tab" data-tab="organize">Organize</button>
  <button class="feat-tab" data-tab="share">Share</button>
  <button class="feat-tab" data-tab="presign">Direct Upload</button>
</div>
<div class="feat-panels">
  <div class="feat-panel active" id="tab-upload">
    <div class="feat-code"><pre>PUT /files/docs/readme.md
Authorization: Bearer sk_...
Content-Type: text/markdown

&lt;file bytes&gt;

<span class="t-dim">&rarr; 201 Created</span>
{
  "id": "o_7f2a9c",
  "path": "docs/readme.md",
  "name": "readme.md",
  "size": 1234
}</pre></div>
    <div class="feat-info">
      <div class="feat-name">Upload anything</div>
      <p>PUT a file, get a URL. Content-Type auto-detected from extension. Parent folders created on the fly. Up to 100 MB per request, or unlimited via presigned URLs.</p>
    </div>
  </div>
  <div class="feat-panel" id="tab-organize">
    <div class="feat-code"><pre>GET /folders/docs
Authorization: Bearer sk_...

<span class="t-dim">&rarr; 200 OK</span>
{
  "path": "docs/",
  "items": [
    {"name": "reports", "is_folder": true},
    {"name": "readme.md", "size": 1234}
  ]
}

POST /drive/rename
{"path": "docs/old.md", "new_name": "new.md"}

<span class="t-dim">&rarr; {"old_path":"docs/old.md","new_path":"docs/new.md"}</span></pre></div>
    <div class="feat-info">
      <div class="feat-name">Organize with folders</div>
      <p>Virtual folder tree with nested paths. Star, rename, move, copy, trash, and restore. Full Google Drive-class file management through a simple REST API.</p>
    </div>
  </div>
  <div class="feat-panel" id="tab-share">
    <div class="feat-code"><pre>POST /shares
Authorization: Bearer sk_...
Content-Type: application/json

{
  "path": "models/v3.bin",
  "grantee": "a/inference",
  "permission": "read"
}

<span class="t-dim">&rarr; 201 Created</span>
{"id": "sh_9ea9", "permission": "read"}</pre></div>
    <div class="feat-info">
      <div class="feat-name">Share with anyone</div>
      <p>Grant read or write access to any actor. Humans and AI agents share the same permission model. List, filter, and revoke shares at any time.</p>
    </div>
  </div>
  <div class="feat-panel" id="tab-presign">
    <div class="feat-code"><pre>POST /presign/upload
Authorization: Bearer sk_...
Content-Type: application/json

{
  "path": "models/v3.bin",
  "content_type": "application/octet-stream"
}

<span class="t-dim">&rarr; 200 OK</span>
{
  "upload_url": "https://...signed...",
  "method": "PUT",
  "expires_in": 3600
}</pre></div>
    <div class="feat-info">
      <div class="feat-name">Direct to storage</div>
      <p>Presigned URLs for zero-hop uploads direct to object storage. No file bytes pass through the API server. Ideal for large files and high-throughput pipelines.</p>
    </div>
  </div>
</div>`;

  /* -- API surface cards ----------------------------------------------- */
  const apiSurface = `
<div class="api-grid">
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Auth</div>
      <div class="api-card-count">7 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">POST</span> /actors</div>
      <div class="api-route"><span class="ep-method">POST</span> /auth/challenge</div>
      <div class="api-route"><span class="ep-method">POST</span> /auth/verify</div>
      <div class="api-route"><span class="ep-method">POST</span> /auth/magic-link</div>
      <div class="api-route dim">+3 more</div>
    </div>
  </div>
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Files</div>
      <div class="api-card-count">4 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">PUT</span> /files/*path</div>
      <div class="api-route"><span class="ep-method">GET</span> /files/*path</div>
      <div class="api-route"><span class="ep-method">DEL</span> /files/*path</div>
      <div class="api-route"><span class="ep-method">HEAD</span> /files/*path</div>
    </div>
  </div>
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Folders</div>
      <div class="api-card-count">4 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">POST</span> /folders</div>
      <div class="api-route"><span class="ep-method">GET</span> /folders</div>
      <div class="api-route"><span class="ep-method">GET</span> /folders/*path</div>
      <div class="api-route"><span class="ep-method">DEL</span> /folders/*path</div>
    </div>
  </div>
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Presign</div>
      <div class="api-card-count">3 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">POST</span> /presign/upload</div>
      <div class="api-route"><span class="ep-method">POST</span> /presign/download</div>
      <div class="api-route"><span class="ep-method">POST</span> /presign/complete</div>
    </div>
  </div>
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Shares</div>
      <div class="api-card-count">5 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">POST</span> /shares</div>
      <div class="api-route"><span class="ep-method">GET</span> /shares</div>
      <div class="api-route"><span class="ep-method">DEL</span> /shares/:id</div>
      <div class="api-route"><span class="ep-method">GET</span> /shared</div>
      <div class="api-route"><span class="ep-method">GET</span> /shared/:owner/*</div>
    </div>
  </div>
  <div class="api-card">
    <div class="api-card-head">
      <div class="api-card-title">Drive</div>
      <div class="api-card-count">13 routes</div>
    </div>
    <div class="api-card-routes">
      <div class="api-route"><span class="ep-method">PATCH</span> /drive/star</div>
      <div class="api-route"><span class="ep-method">POST</span> /drive/rename</div>
      <div class="api-route"><span class="ep-method">POST</span> /drive/move</div>
      <div class="api-route"><span class="ep-method">GET</span> /drive/search</div>
      <div class="api-route dim">+9 more</div>
    </div>
  </div>
</div>`;

  /* -- Integrations ---------------------------------------------------- */
  const integrations = `
<div class="integration-grid">
  <div class="integration-card">
    <h3>MCP Server</h3>
    <p>Model Context Protocol. 8 tools for file management. Connect any MCP-compatible AI assistant.</p>
  </div>
  <div class="integration-card">
    <h3>ChatGPT</h3>
    <p>OAuth 2.0 PKCE flow. Dynamic client registration. Works with ChatGPT plugins and custom GPTs.</p>
  </div>
  <div class="integration-card">
    <h3>Claude</h3>
    <p>Native MCP support. Direct file access. Seamless integration with Claude Desktop and API.</p>
  </div>
</div>`;

  /* -- CTA section (signed-out only) ---------------------------------- */
  const ctaSection = isSignedIn
    ? ""
    : `
<div class="section section--cta" id="get-started">
  <div class="section-inner section-inner--center">
    <div class="signin-card">
      <div class="signin-label"><span class="prompt-caret">&gt;</span> ready?</div>
      <div class="cta-title">Start building in <span class="shimmer">seconds</span></div>
      <div class="prompt-form" id="signin-form">
        <span class="prompt-prefix">$</span>
        <input type="email" id="email-input" placeholder="you@example.com" autocomplete="email" spellcheck="false">
        <button id="signin-btn" onclick="signIn()">
          <span id="signin-text">Enter</span>
          <span id="signin-loading" style="display:none"><span class="spinner"></span></span>
        </button>
      </div>
      <div class="prompt-error" id="signin-error"></div>
      <div class="prompt-note">Magic link &middot; No password &middot; Free forever</div>
      <div class="prompt-success" id="signin-success" style="display:none"></div>
    </div>
  </div>
</div>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>storage.now &mdash; Developer Platform</title>
<meta name="description" content="36 REST endpoints. Zero egress. Global edge. MCP-native. The file storage API that gets out of your way.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/developers.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">browse</a>
      <a href="/developers" class="active">developers</a>
      <a href="/docs">docs</a>
      <a href="/pricing">pricing</a>
      <a href="/ai">ai</a>
    </div>
    <div class="nav-right">
      ${navSession}
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<!-- HERO -->
<div class="hero">
  <div class="section-inner section-inner--center">
    ${isSignedIn ? heroSignedIn : heroSignedOut}
  </div>
</div>

<!-- STATS -->
<div class="section section--stats">
  <div class="section-inner">
    ${statsStrip}
  </div>
</div>

<!-- LIVE TERMINAL -->
<div class="section section--terminal">
  <div class="section-inner">
    <div class="section-label">HOW IT WORKS</div>
    <div class="section-heading">Upload, share, done. <span class="grad">Two requests.</span></div>
    ${terminalDemo}
  </div>
</div>

<!-- CODE EXAMPLES -->
<div class="section section--code">
  <div class="section-inner">
    <div class="section-label">SDK EXAMPLES</div>
    <div class="section-heading">One API. <span class="grad">Any language.</span></div>
    ${codeExamples}
  </div>
</div>

<!-- FEATURES -->
<div class="section section--features">
  <div class="section-inner">
    <div class="section-label">CAPABILITIES</div>
    <div class="section-heading">Everything you need to <span class="grad">ship</span></div>
    ${featureTabs}
  </div>
</div>

<!-- API SURFACE -->
<div class="section section--api">
  <div class="section-inner">
    <div class="section-label">API REFERENCE</div>
    <div class="section-heading"><span class="grad">36</span> endpoints. One base URL.</div>
    <div class="section-sub">https://storage.liteio.dev</div>
    ${apiSurface}
    <a href="/docs" class="docs-link">&gt; read the full documentation &rarr;</a>
  </div>
</div>

<!-- INTEGRATIONS -->
<div class="section section--integrations">
  <div class="section-inner">
    <div class="section-label">INTEGRATIONS</div>
    <div class="section-heading">Built for the <span class="grad">AI era</span></div>
    ${integrations}
    <a href="/ai" class="integration-link">Set up AI integration &rarr;</a>
  </div>
</div>

<!-- CTA -->
${ctaSection}

<!-- Footer -->
<footer>
  <div class="section-inner">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/docs">docs</a>
      <a href="/pricing">pricing</a>
      <a href="/ai">ai</a>
      <a href="/browse">browse</a>
    </div>
  </div>
</footer>

<script>
/* Theme */
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();

/* Feature tabs */
(function(){
  const tabs=document.querySelectorAll('.feat-tab');
  const panels=document.querySelectorAll('.feat-panel');
  if(!tabs.length) return;
  tabs.forEach(tab=>{
    tab.addEventListener('click',()=>{
      tabs.forEach(t=>t.classList.remove('active'));
      panels.forEach(p=>p.classList.remove('active'));
      tab.classList.add('active');
      const panel=document.getElementById('tab-'+tab.dataset.tab);
      if(panel) panel.classList.add('active');
    });
  });
})();

/* Code language tabs */
(function(){
  const tabs=document.querySelectorAll('.code-tab');
  const panels=document.querySelectorAll('.code-panel');
  if(!tabs.length) return;
  tabs.forEach(tab=>{
    tab.addEventListener('click',()=>{
      tabs.forEach(t=>t.classList.remove('active'));
      panels.forEach(p=>p.classList.remove('active'));
      tab.classList.add('active');
      const panel=document.getElementById('lang-'+tab.dataset.lang);
      if(panel) panel.classList.add('active');
    });
  });
})();

/* Terminal typing animation */
(function(){
  const lines=document.querySelectorAll('#term-body .term-line');
  if(!lines.length) return;
  let i=0;
  function show(){
    if(i>=lines.length)return;
    lines[i].classList.add('visible');
    i++;
    const prev=lines[i-1];
    const delay=prev.classList.contains('term-line--res')?80:
                prev.classList.contains('term-blank')?300:160;
    setTimeout(show,delay);
  }
  const obs=new IntersectionObserver((entries)=>{
    if(entries[0].isIntersecting){obs.disconnect();show()}
  },{threshold:0.3});
  const tb=document.getElementById('term-body');
  if(tb)obs.observe(tb);
})();

/* Sign in */
async function signIn(){
  const input=document.getElementById('email-input');
  const btn=document.getElementById('signin-btn');
  const errEl=document.getElementById('signin-error');
  if(!input||!btn||!errEl) return;
  const email=input.value.trim();
  if(!email){errEl.textContent='email required';return}
  if(!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){errEl.textContent='invalid email';return}
  errEl.textContent='';
  btn.disabled=true;input.disabled=true;
  document.getElementById('signin-text').style.display='none';
  document.getElementById('signin-loading').style.display='inline-flex';
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();
    if(!res.ok) throw new Error(data.error?.message||'failed');
    if(data.magic_link){window.location.href=data.magic_link}
    else{document.getElementById('signin-form').innerHTML='<div class="prompt-success">check your inbox &mdash; magic link sent</div>'}
  }catch(err){
    errEl.textContent=err.message;btn.disabled=false;input.disabled=false;
    document.getElementById('signin-text').style.display='inline';
    document.getElementById('signin-loading').style.display='none';
  }
}
document.getElementById('email-input')?.addEventListener('keydown',e=>{if(e.key==='Enter')signIn()});

/* Scroll reveal */
(function(){
  const els=document.querySelectorAll('.section, .signin-section');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.08,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
