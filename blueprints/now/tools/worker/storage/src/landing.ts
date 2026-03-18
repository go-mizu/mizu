export function landingPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? actor.slice(2) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${esc(displayName)}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  /* ── Signed-in hero ─────────────────────────────────────────────── */
  const heroSignedIn = `
<div class="hero-greeting">Welcome back, <span class="grad">${esc(displayName)}</span></div>
<div class="hero-actions">
  <a href="/browse" class="hero-btn hero-btn--primary">
    <span class="btn-icon">&gt;_</span> Open dashboard
  </a>
  <a href="/docs" class="hero-btn">
    <span class="btn-icon">{ }</span> API docs
  </a>
</div>`;

  /* ── Signed-out hero ────────────────────────────────────────────── */
  const heroSignedOut = `
<div class="hero-badge">STORAGE FOR HUMANS & AGENTS</div>
<h1 class="hero-title">Ship files at the<br><span class="grad">speed of thought</span></h1>
<p class="hero-sub">The file platform built for the AI era. Upload, organize, and share &mdash; from your code or your browser. Zero egress fees.</p>
<div class="hero-prompt" id="hero-prompt">
  <div class="prompt-label"><span class="prompt-caret">&gt;</span> enter your email to start</div>
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
</div>`;

  /* ── Stats strip ────────────────────────────────────────────────── */
  const statsStrip = `
<div class="stats">
  <div class="stat"><div class="stat-val"><span class="grad">300+</span></div><div class="stat-label">Edge locations</div></div>
  <div class="stat"><div class="stat-val"><span class="grad">&lt;50ms</span></div><div class="stat-label">Metadata latency</div></div>
  <div class="stat"><div class="stat-val"><span class="grad">$0</span></div><div class="stat-label">Egress fees</div></div>
  <div class="stat"><div class="stat-val"><span class="grad">REST</span></div><div class="stat-label">First-class API</div></div>
</div>`;

  /* ── Terminal session demo ──────────────────────────────────────── */
  const terminalDemo = `
<div class="term">
  <div class="term-bar">
    <div class="term-dots"><span></span><span></span><span></span></div>
    <div class="term-title">storage.now &mdash; session</div>
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

  /* ── CTA (signed-out only) ──────────────────────────────────────── */
  const ctaSection = isSignedIn ? "" : `
<div class="section section--cta">
  <div class="section-inner section-inner--center">
    <div class="cta-prompt-label"><span class="prompt-caret">&gt;</span> ready?</div>
    <div class="cta-title">Start shipping in <span class="grad">seconds</span></div>
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
<title>storage.now — File storage for humans and agents</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
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
    <a href="/docs">docs</a>
    <a href="/pricing">pricing</a>
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
  <div class="section-inner">
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
    <div class="section-heading">Upload, share, done.<br><span class="grad">Two requests.</span></div>
    ${terminalDemo}
  </div>
</div>

<!-- CAPABILITIES -->
<div class="section">
  <div class="section-inner">
    <div class="section-label">FEATURES</div>
    <div class="section-heading">Everything you need to <span class="grad">ship</span></div>
    <div class="caps-grid">
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg></div>
          <div class="cap-name">Upload anything</div>
        </div>
        <div class="cap-desc">PUT a file, get a URL. Auto-detects content type. Folders created on the fly. Up to 100MB per file.</div>
      </div>
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></div>
          <div class="cap-name">Organize with folders</div>
        </div>
        <div class="cap-desc">Virtual folder tree. Nested paths. List contents at any depth. Folders first, then files, sorted.</div>
      </div>
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg></div>
          <div class="cap-name">Share with anyone</div>
        </div>
        <div class="cap-desc">Grant read or write access to any actor. Humans and agents share the same permission model. Revoke anytime.</div>
      </div>
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg></div>
          <div class="cap-name">Cryptographic identity</div>
        </div>
        <div class="cap-desc">Ed25519 challenge-response for agents. Magic links for humans. No passwords, no rotating API keys.</div>
      </div>
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg></div>
          <div class="cap-name">Global edge network</div>
        </div>
        <div class="cap-desc">300+ locations worldwide. Sub-50ms metadata. Files served from the nearest edge. Zero egress fees.</div>
      </div>
      <div class="cap">
        <div class="cap-header">
          <div class="cap-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/><polyline points="13 2 13 9 20 9"/></svg></div>
          <div class="cap-name">Direct uploads</div>
        </div>
        <div class="cap-desc">Presigned URLs for direct-to-storage uploads. Skip the middleman. Zero-hop for large files.</div>
      </div>
    </div>
  </div>
</div>

<!-- ENDPOINTS -->
<div class="section">
  <div class="section-inner">
    <div class="section-label">API REFERENCE</div>
    <div class="section-heading">The complete <span class="grad">surface</span></div>
    <div class="endpoints">
      <div class="ep-group">
        <div class="ep-group-title">Auth</div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/actors</span> <span class="ep-desc">Register</span></div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/auth/challenge</span> <span class="ep-desc">Get nonce</span></div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/auth/verify</span> <span class="ep-desc">Get token</span></div>
      </div>
      <div class="ep-group">
        <div class="ep-group-title">Files</div>
        <div class="ep"><span class="ep-method ep-method--put">PUT</span> <span class="ep-path">/files/*path</span> <span class="ep-desc">Upload</span></div>
        <div class="ep"><span class="ep-method ep-method--get">GET</span> <span class="ep-path">/files/*path</span> <span class="ep-desc">Download</span></div>
        <div class="ep"><span class="ep-method ep-method--del">DEL</span> <span class="ep-path">/files/*path</span> <span class="ep-desc">Delete</span></div>
        <div class="ep"><span class="ep-method ep-method--head">HEAD</span> <span class="ep-path">/files/*path</span> <span class="ep-desc">Metadata</span></div>
      </div>
      <div class="ep-group">
        <div class="ep-group-title">Folders</div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/folders</span> <span class="ep-desc">Create</span></div>
        <div class="ep"><span class="ep-method ep-method--get">GET</span> <span class="ep-path">/folders/*path</span> <span class="ep-desc">List</span></div>
        <div class="ep"><span class="ep-method ep-method--del">DEL</span> <span class="ep-path">/folders/*path</span> <span class="ep-desc">Delete</span></div>
      </div>
      <div class="ep-group">
        <div class="ep-group-title">Presign</div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/presign/upload</span> <span class="ep-desc">Get upload URL</span></div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/presign/download</span> <span class="ep-desc">Get download URL</span></div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/presign/complete</span> <span class="ep-desc">Confirm upload</span></div>
      </div>
      <div class="ep-group">
        <div class="ep-group-title">Shares</div>
        <div class="ep"><span class="ep-method ep-method--post">POST</span> <span class="ep-path">/shares</span> <span class="ep-desc">Grant</span></div>
        <div class="ep"><span class="ep-method ep-method--get">GET</span> <span class="ep-path">/shares</span> <span class="ep-desc">List</span></div>
        <div class="ep"><span class="ep-method ep-method--del">DEL</span> <span class="ep-path">/shares/:id</span> <span class="ep-desc">Revoke</span></div>
        <div class="ep"><span class="ep-method ep-method--get">GET</span> <span class="ep-path">/shared</span> <span class="ep-desc">Shared w/ me</span></div>
      </div>
    </div>
    <a href="/docs" class="docs-link">&gt; read the full docs &rarr;</a>
  </div>
</div>

<!-- CTA -->
${ctaSection}

</div>

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button><span class="h1"># storage.now</span>

File storage API for agents and humans.
Base URL: https://storage.liteio.workers.dev

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

<span class="h2">## All endpoints</span>

POST   /actors                     Register
POST   /auth/challenge             Get challenge
POST   /auth/verify                Verify, get token
POST   /auth/magic-link            Magic link (email)
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

<span class="h2">## Links</span>

<span class="link">https://storage.liteio.workers.dev/docs</span>  Full documentation
<span class="link">https://storage.liteio.workers.dev/browse</span>  File browser</div>
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
      <a href="/docs">docs</a>
      <a href="/pricing">pricing</a>
      <a href="/browse">browse</a>
    </div>
  </div>
</footer>

<script>
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light'){
    document.documentElement.classList.remove('dark');
  } else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches){
    document.documentElement.classList.remove('dark');
  }
})();

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

function copyMd(){
  const el=document.getElementById('md-content');
  const text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(()=>{
    const btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}

async function signIn(){
  const input=document.getElementById('email-input');
  const btn=document.getElementById('signin-btn');
  const errEl=document.getElementById('signin-error');
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
  let i=0;
  function show(){
    if(i>=lines.length)return;
    lines[i].classList.add('visible');
    i++;
    const delay=lines[i-1].classList.contains('term-line--res')?80:
                lines[i-1].classList.contains('term-blank')?300:160;
    setTimeout(show,delay);
  }
  const obs=new IntersectionObserver((entries)=>{
    if(entries[0].isIntersecting){obs.disconnect();show()}
  },{threshold:0.3});
  const tb=document.getElementById('term-body');
  if(tb)obs.observe(tb);
})();
</script>
</body>
</html>`;
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
