import { esc } from "./layout";

export function developersPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  const heroCta = isSignedIn
    ? `<a href="/api" class="btn btn--primary btn--lg">API Reference</a>
       <a href="/browse" class="btn btn--ghost btn--lg">Dashboard</a>`
    : `<a href="/api" class="btn btn--primary btn--lg">API Reference</a>
       <a href="#cta" class="btn btn--ghost btn--lg">Get started</a>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Developers — storage.now</title>
<meta name="description" content="Build on storage.now — a file API shaped like Unix. 18 operations, 8 resources, zero complexity.">
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
  </div>
</nav>

<main>

<!-- ═══ Hero ═══ -->
<div class="hero">
  <div class="glow-spot glow-spot--hero"></div>
  <div class="hero-content">
    <div class="hero-badge">DEVELOPER PLATFORM</div>
    <h1 class="hero-title">Build on storage</h1>
    <p class="hero-sub">A file API shaped like Unix. Store, share, and serve files<br>with 18 operations across 8 resources.</p>
    <div class="hero-ctas">${heroCta}</div>
  </div>
  <div class="hero-terminal">
    <div class="terminal">
      <div class="terminal-bar">
        <div class="terminal-dots"><span></span><span></span><span></span></div>
        <div class="terminal-title">terminal</div>
      </div>
      <div class="terminal-body"><span class="t-comment"># upload a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.liteio.dev/files/report.pdf \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-T</span> report.pdf

<span class="t-res">201 {"path":"report.pdf","size":524288}</span>

<span class="t-comment"># share it with anyone</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.liteio.dev/shares \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/json"</span> \\
    <span class="t-flag">-d</span> <span class="t-str">'{"path":"report.pdf","grantee":"public","permission":"viewer"}'</span>

<span class="t-res">201 {"token":"tok_a8f","url":"<a href='/p/tok_a8f' class='t-link'>/p/tok_a8f</a>"}</span></div>
    </div>
  </div>
</div>

<!-- ═══ Features ═══ -->
<section class="section" id="features">
  <div class="section-pad">
    <div class="section-label">CAPABILITIES</div>
    <div class="section-heading">Everything you need.<br>Nothing you don't.</div>
  </div>
  <div class="features">
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="feature-methods">PUT &middot; GET &middot; DELETE &middot; HEAD</div>
      <div class="feature-name">File I/O</div>
      <p>Store any file with a single HTTP request. PUT to write, GET to read, DELETE to remove, HEAD to stat. Content-Type is auto-detected from the file extension.</p>
      <div class="feature-path">/files/{path}</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
      </div>
      <div class="feature-methods">GET</div>
      <div class="feature-name">Directory Tree</div>
      <p>List files and folders at any depth. Virtual filesystem with directories created implicitly on write. Like <code>ls</code> for the cloud.</p>
      <div class="feature-path">/tree/{path}</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
      </div>
      <div class="feature-methods">POST &middot; GET &middot; DELETE</div>
      <div class="feature-name">Sharing</div>
      <p>Grant access with viewer, editor, or uploader permissions. Create public links for anyone or private shares to specific actors. Like <code>chmod</code> meets symlinks.</p>
      <div class="feature-path">/shares</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0018 9h-1.26A8 8 0 103 16.3"/></svg>
      </div>
      <div class="feature-methods">POST</div>
      <div class="feature-name">Presigned URLs</div>
      <p>Direct client-to-storage transfers. Upload gigabytes without touching your server. Data flows straight to the object store with zero proxy overhead.</p>
      <div class="feature-path">/presign</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z"/></svg>
      </div>
      <div class="feature-methods">MCP + OAuth 2.0</div>
      <div class="feature-name">AI Native</div>
      <p>Built-in Model Context Protocol server with OAuth 2.0 PKCE. Connect ChatGPT, Claude, or any MCP-compatible AI agent to manage your files.</p>
      <div class="feature-path">/mcp</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      </div>
      <div class="feature-methods">POST &middot; GET &middot; DELETE</div>
      <div class="feature-name">Keys & Audit</div>
      <p>Generate scoped API keys for your services. Full audit trail of every operation &mdash; file writes, shares, deletions. Like <code>/var/log</code> for your storage.</p>
      <div class="feature-path">/keys &middot; /log</div>
    </div>
  </div>
</section>

<!-- ═══ How it works ═══ -->
<section class="section" id="how">
  <div class="section-pad">
    <div class="section-label">GET STARTED</div>
    <div class="section-heading">Three steps. Five minutes.</div>
  </div>
  <div class="steps">
    <div class="step">
      <div class="step-num">01</div>
      <div class="step-name">Authenticate</div>
      <p>Register with Ed25519 for machines or magic link for humans. Get a Bearer token that works with every endpoint.</p>
      <pre class="step-code"><span class="t-cmd">POST</span> /auth/token
<span class="t-str">{"method":"ed25519",</span>
<span class="t-str"> "step":"challenge",</span>
<span class="t-str"> "actor":"a/my-bot"}</span>

<span class="t-res">&rarr; {"challenge_id":"ch_..."}</span></pre>
    </div>
    <div class="step">
      <div class="step-num">02</div>
      <div class="step-name">Store</div>
      <p>Upload files to any path. Directories are created automatically. Up to 100 MB per request, unlimited via presigned URLs.</p>
      <pre class="step-code"><span class="t-cmd">PUT</span> /files/docs/readme.md
<span class="t-str">Authorization: Bearer sk_...</span>
<span class="t-str">Content-Type: text/markdown</span>

<span class="t-res">&rarr; 201 {"path":"docs/readme.md"}</span></pre>
    </div>
    <div class="step">
      <div class="step-num">03</div>
      <div class="step-name">Build</div>
      <p>Share files, generate public links, connect AI agents, query the audit log. Build your product on a real filesystem API.</p>
      <pre class="step-code"><span class="t-cmd">POST</span> /shares
<span class="t-str">{"path":"docs/readme.md",</span>
<span class="t-str"> "grantee":"public",</span>
<span class="t-str"> "permission":"viewer"}</span>

<span class="t-res">&rarr; 201 {"url":"/p/tok_a8f"}</span></pre>
    </div>
  </div>
</section>

<!-- ═══ Stats ═══ -->
<section class="section" id="numbers">
  <div class="stats">
    <div class="stat">
      <div class="stat-num">18</div>
      <div class="stat-label">Operations</div>
    </div>
    <div class="stat">
      <div class="stat-num">8</div>
      <div class="stat-label">Resources</div>
    </div>
    <div class="stat">
      <div class="stat-num">300+</div>
      <div class="stat-label">Edge locations</div>
    </div>
    <div class="stat">
      <div class="stat-num">&lt;50ms</div>
      <div class="stat-label">Global latency</div>
    </div>
  </div>
</section>

<!-- ═══ Protocol ═══ -->
<section class="section" id="protocol">
  <div class="section-pad">
    <div class="section-label">THE PROTOCOL</div>
    <div class="section-heading">Unix philosophy. HTTP interface.</div>
    <p class="section-desc">Every resource maps to a Unix concept you already know. If you understand the filesystem, you understand the API.</p>
    <table class="resource-table">
      <thead><tr><th>Resource</th><th>Operations</th><th>Unix analog</th></tr></thead>
      <tbody>
        <tr><td>/auth</td><td>4</td><td>login / logout / useradd</td></tr>
        <tr><td>/files</td><td>4</td><td>read / write / rm / stat</td></tr>
        <tr><td>/tree</td><td>1</td><td>ls</td></tr>
        <tr><td>/shares</td><td>3</td><td>chmod / ln -s</td></tr>
        <tr><td>/p</td><td>1</td><td>readlink</td></tr>
        <tr><td>/presign</td><td>1</td><td>pipe</td></tr>
        <tr><td>/keys</td><td>3</td><td>ssh-keygen</td></tr>
        <tr><td>/log</td><td>1</td><td>tail /var/log</td></tr>
      </tbody>
    </table>
    <a href="/api" class="api-link">Full API reference &rarr;</a>
  </div>
</section>

<!-- ═══ CTA ═══ -->
<section class="section section--cta" id="cta">
  <div class="glow-spot glow-spot--cta"></div>
  <div class="section-pad">
    <div class="cta-label"><span class="cta-caret">&gt;</span> ready?</div>
    <div class="cta-title">Start building today</div>
    <p class="cta-desc">Read the API reference, grab an API key, and ship your first integration in minutes.</p>
    <div class="cta-actions">
      <a href="/api" class="btn btn--primary btn--lg">API Reference</a>
      ${isSignedIn ? '' : `<a href="/" class="btn btn--ghost btn--lg">Create free account</a>`}
    </div>
  </div>
</section>

</main>

<footer>
  <div class="section-pad">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/browse">browse</a>
      <a href="/developers">developers</a>
      <a href="/api">api</a>
      <a href="/pricing">pricing</a>
      <a href="/ai">ai</a>
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
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();
(function(){
  const els=document.querySelectorAll('.section');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.05,rootMargin:'0px 0px -60px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
