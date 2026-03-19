export function landingPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? actor.slice(2) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${esc(displayName)}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  /* ── Signed-in hero ─────────────────────────────────────────────── */
  const heroSignedIn = `
<div class="hero-greeting">Welcome back, <strong>${esc(displayName)}</strong></div>
<div class="hero-actions">
  <a href="/browse" class="btn btn--primary">
    <span class="btn-icon">&gt;_</span> Open dashboard
  </a>
  <a href="/api" class="btn btn--ghost">API Reference</a>
</div>`;

  /* ── Signed-out hero ────────────────────────────────────────────── */
  const heroSignedOut = `
<div class="hero-badge">FILE STORAGE FOR THE AI ERA</div>
<h1 class="hero-title">Store. Share.<br><span class="shimmer">Ship.</span></h1>
<p class="hero-sub">The file platform built for humans and AI agents.<br>Zero egress fees. Global edge. REST-native.</p>
<div class="hero-ctas">
  <a href="#get-started" class="btn btn--primary btn--lg" onclick="document.getElementById('email-input').focus();return false">Get started free</a>
  <a href="/api" class="btn btn--ghost btn--lg">API Reference &rarr;</a>
</div>`;

  /* ── Sign-in form (signed-out only) ─────────────────────────────── */
  const signInForm = isSignedIn
    ? ""
    : `
<div class="signin-section" id="get-started">
  <div class="section-inner section-inner--center">
    <div class="signin-card">
      <div class="signin-label"><span class="prompt-caret">&gt;</span> enter your email to start</div>
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
    </div>
  </div>
</div>`;

  /* ── Stats strip ────────────────────────────────────────────────── */
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

  /* ── Terminal demo ──────────────────────────────────────────────── */
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

  /* ── Feature tabs ───────────────────────────────────────────────── */
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

  /* ── API surface cards ──────────────────────────────────────────── */
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

  /* ── CTA (signed-out only) ──────────────────────────────────────── */
  const ctaSection = isSignedIn
    ? ""
    : `
<div class="section section--cta">
  <div class="section-inner section-inner--center">
    <div class="cta-label"><span class="prompt-caret">&gt;</span> ready?</div>
    <div class="cta-title">Start shipping in <span class="shimmer">seconds</span></div>
    <div class="prompt-form" id="signin-form-2">
      <span class="prompt-prefix">$</span>
      <input type="email" id="email-input-2" placeholder="you@example.com" autocomplete="email" spellcheck="false">
      <button onclick="signInBottom()">Enter</button>
    </div>
  </div>
</div>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>storage.now &mdash; File storage for humans and agents</title>
<meta name="description" content="File storage built for humans and AI agents. Upload, organize, and share via REST. Zero egress fees. Global edge network.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/landing.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <a href="/" class="logo">
    <span class="logo-dot"></span> storage.now
  </a>
  <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-links">
    <a href="/browse">browse</a>
    <a href="/api">api</a>
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
</nav>

<!-- ===== HUMAN VIEW ===== -->
<div class="human-view" id="human-view">

<!-- HERO -->
<div class="hero">
  <div class="hero-glow"></div>
  <div class="section-inner section-inner--center">
    ${isSignedIn ? heroSignedIn : heroSignedOut}
  </div>
</div>

<!-- SIGN IN -->
${signInForm}

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
    <a href="/api" class="docs-link">&gt; read the full API reference &rarr;</a>
  </div>
</div>

<!-- CTA -->
${ctaSection}

</div>

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button><span class="h1"># storage.now</span>

File storage API for agents and humans.
Base URL: https://storage.liteio.dev

<span class="h2">## Quick start</span>

<span class="h3">### 1. Register</span>

POST /actors
Content-Type: application/json

{
  "actor": "a/your-agent",
  "public_key": "&lt;base64url-public-key&gt;",
  "type": "agent"
}

<span class="dim">&rarr; {"actor":"a/your-agent","created":true}</span>

<span class="h3">### 2. Authenticate</span>

POST /auth/challenge
{"actor": "a/your-agent"}

<span class="dim">&rarr; {"challenge_id":"ch_...","nonce":"...","expires_at":...}</span>

POST /auth/verify
{
  "challenge_id": "ch_...",
  "actor": "a/your-agent",
  "signature": "&lt;base64url-signed-nonce&gt;"
}

<span class="dim">&rarr; {"access_token":"...","expires_at":...}</span>

<span class="h3">### 3. Upload a file</span>

PUT /files/docs/readme.md
Authorization: Bearer &lt;token&gt;
Content-Type: text/markdown

&lt;file bytes&gt;

<span class="dim">&rarr; {"id":"o_...","path":"docs/readme.md","name":"readme.md","size":1234}</span>

<span class="h3">### 4. Download</span>

GET /files/docs/readme.md
Authorization: Bearer &lt;token&gt;

<span class="dim">&rarr; file bytes (Content-Type: text/markdown)</span>

<span class="h3">### 5. List folder</span>

GET /folders/docs
Authorization: Bearer &lt;token&gt;

<span class="dim">&rarr; {"path":"docs/","items":[{"name":"readme.md","size":1234,...}]}</span>

<span class="h2">## All endpoints (36)</span>

POST   /actors                     Register
POST   /auth/challenge             Get challenge
POST   /auth/verify                Verify, get token
POST   /auth/magic-link            Magic link (email)
GET    /auth/magic/:token          Verify magic link
POST   /auth/logout                End session
GET    /auth/logout                End session (link)
PUT    /files/*path                Upload file
GET    /files/*path                Download file
DELETE /files/*path                Delete file
HEAD   /files/*path                File metadata
POST   /presign/upload             Get presigned upload URL
POST   /presign/download           Get presigned download URL
POST   /presign/complete           Confirm presigned upload
POST   /folders                    Create folder
GET    /folders                    List root
GET    /folders/*path              List folder
DELETE /folders/*path              Delete folder
POST   /shares                     Share a file
GET    /shares                     List shares
DELETE /shares/:id                 Revoke share
GET    /shared                     Files shared with me
GET    /shared/:owner/*path        Download shared file
PATCH  /drive/star                 Star / unstar
POST   /drive/rename               Rename file or folder
POST   /drive/move                 Move items
POST   /drive/copy                 Duplicate file
POST   /drive/trash                Trash (soft delete)
POST   /drive/restore              Restore from trash
DELETE /drive/trash                Empty trash
GET    /drive/trash                List trashed items
GET    /drive/recent               Recently accessed
GET    /drive/starred              Starred items
GET    /drive/search               Search by name
GET    /drive/stats                Storage usage
PATCH  /drive/description          Update description

<span class="h2">## Links</span>

<span class="link">https://storage.liteio.dev/api</span>  API reference
<span class="link">https://storage.liteio.dev/browse</span>  File browser</div>
</div>

<!-- Floating mode switch -->
<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

<!-- Footer -->
<footer>
  <div class="section-inner">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/api">api</a>
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

/* Mode switch */
function setMode(mode){
  const btns=document.querySelectorAll('.mode-switch button');
  btns.forEach(b=>b.classList.remove('active'));
  if(mode==='human'){
    btns[0].classList.add('active');
    document.getElementById('human-view').classList.remove('hidden');
    document.getElementById('machine-view').classList.remove('active');
  } else {
    btns[1].classList.add('active');
    document.getElementById('human-view').classList.add('hidden');
    document.getElementById('machine-view').classList.add('active');
  }
}

/* Machine view copy */
function copyMd(){
  const el=document.getElementById('md-content');
  const text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(()=>{
    const btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}

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
async function signInBottom(){
  const input=document.getElementById('email-input-2');
  if(!input) return;
  const email=input.value.trim();
  if(!email||!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)) return;
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();if(!res.ok) return;
    if(data.magic_link) window.location.href=data.magic_link;
    else document.getElementById('signin-form-2').innerHTML='<div class="prompt-success">check your inbox</div>';
  }catch{}
}
document.getElementById('email-input')?.addEventListener('keydown',e=>{if(e.key==='Enter')signIn()});
document.getElementById('email-input-2')?.addEventListener('keydown',e=>{if(e.key==='Enter')signInBottom()});

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

function esc(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
