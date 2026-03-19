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
       <a href="/" class="btn btn--ghost btn--lg">Get started</a>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Developers — storage.now</title>
<meta name="description" content="Build on storage.now — Buckets, Objects, Signed URLs. Three concepts, zero complexity.">
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
      <a href="/developers" class="active">developers</a>
      <a href="/api">api</a>
      <a href="/pricing">pricing</a>
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

<!-- Hero -->
<section class="hero">
  <div class="glow-spot glow-spot--hero"></div>
  <div class="hero-content">
    <div class="hero-badge">DEVELOPER PLATFORM</div>
    <h1 class="hero-title">Buckets. Objects.<br><span class="grad">Signed URLs.</span></h1>
    <p class="hero-sub">Three concepts. One base URL. That's the entire API.<br>Inspired by Supabase Storage, built for the edge.</p>
    <div class="hero-ctas">${heroCta}</div>
  </div>
  <div class="hero-terminal">
    <div class="terminal">
      <div class="terminal-bar">
        <div class="terminal-dots"><span></span><span></span><span></span></div>
        <div class="terminal-title">terminal</div>
      </div>
      <div class="terminal-body"><span class="t-comment"># create a bucket</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/bucket \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-d</span> <span class="t-str">'{"name":"docs","public":false}'</span>

<span class="t-res">201 {"id":"bk_a1b2","name":"docs"}</span>

<span class="t-comment"># upload an object</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.now/object/docs/report.pdf \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-T</span> report.pdf

<span class="t-res">201 {"path":"report.pdf","size":524288}</span>

<span class="t-comment"># share it with a signed URL</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/object/sign/docs \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-d</span> <span class="t-str">'{"path":"report.pdf","expires_in":3600}'</span>

<span class="t-res">200 {"signed_url":"<a href='/sign/tok_abc' class='t-link'>/sign/tok_abc</a>","expires_at":...}</span></div>
    </div>
  </div>
</section>

<!-- Three Concepts -->
<section class="section" id="concepts">
  <div class="section-pad">
    <div class="section-label">THE MODEL</div>
    <div class="section-heading">Three concepts. That's it.</div>
    <p class="section-desc">Every storage API needs containers, files, and access control. We give each one name and a clean REST interface.</p>
  </div>
  <div class="concepts">
    <div class="concept">
      <div class="concept-num">01</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/></svg>
      </div>
      <div class="concept-name">Buckets</div>
      <p>Top-level containers with their own access rules. Public buckets serve files without auth. Private buckets require a token or signed URL.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-post">POST</span> /bucket</div>
        <div class="concept-method"><span class="m-get">GET</span> /bucket</div>
        <div class="concept-method"><span class="m-get">GET</span> /bucket/:id</div>
        <div class="concept-method"><span class="m-patch">PATCH</span> /bucket/:id</div>
        <div class="concept-method"><span class="m-delete">DEL</span> /bucket/:id</div>
      </div>
    </div>
    <div class="concept">
      <div class="concept-num">02</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="concept-name">Objects</div>
      <p>Files stored in buckets, addressed by path. Upload, download, list, move, copy, delete. Folders are implicit from path prefixes.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-put">PUT</span> /object/:bucket/*path</div>
        <div class="concept-method"><span class="m-get">GET</span> /object/:bucket/*path</div>
        <div class="concept-method"><span class="m-get">GET</span> /object/public/:bucket/*path</div>
        <div class="concept-method"><span class="m-post">POST</span> /object/list/:bucket</div>
        <div class="concept-method"><span class="m-delete">DEL</span> /object/:bucket</div>
      </div>
    </div>
    <div class="concept">
      <div class="concept-num">03</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
      </div>
      <div class="concept-name">Signed URLs</div>
      <p>Time-limited URLs for sharing and direct uploads. One primitive replaces public links, presigned URLs, and sharing permissions.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-post">POST</span> /object/sign/:bucket</div>
        <div class="concept-method"><span class="m-post">POST</span> /object/upload/sign/:bucket/*</div>
        <div class="concept-method"><span class="m-get">GET</span> /sign/:token</div>
        <div class="concept-method"><span class="m-put">PUT</span> /upload/sign/:token</div>
      </div>
    </div>
  </div>
</section>

<!-- Code Examples -->
<section class="section" id="examples">
  <div class="section-pad">
    <div class="section-label">EXAMPLES</div>
    <div class="section-heading">Everything in curl.</div>
  </div>
  <div class="examples">
    <div class="example">
      <div class="example-header">
        <div class="example-label">Create a public bucket</div>
        <div class="example-badge">Buckets</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/bucket \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/json"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{
    "name": "avatars",
    "public": true,
    "file_size_limit": 5242880,
    "allowed_mime_types": ["image/png", "image/jpeg"]
  }'</span>

<span class="t-res">&rarr; 201
{
  "id": "bk_a1b2c3",
  "name": "avatars",
  "public": true,
  "file_size_limit": 5242880
}</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Upload an object</div>
        <div class="example-badge">Objects</div>
      </div>
      <pre class="example-code"><span class="t-comment"># PUT creates or replaces</span>
<span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.now/object/avatars/alice.png \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: image/png"</span> \\
  <span class="t-flag">-T</span> alice.png

<span class="t-res">&rarr; 201
{
  "id": "o_x7y8z9",
  "bucket": "avatars",
  "path": "alice.png",
  "size": 48576
}</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Public download (no auth)</div>
        <div class="example-badge">Objects</div>
      </div>
      <pre class="example-code"><span class="t-comment"># public bucket &rarr; no token needed</span>
<span class="t-cmd">curl</span> storage.now/object/public/avatars/alice.png \\
  <span class="t-flag">-o</span> alice.png

<span class="t-res">&rarr; 200  &lt;binary&gt;
Content-Type: image/png
Cache-Control: public, max-age=3600</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">List with prefix &amp; search</div>
        <div class="example-badge">Objects</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/object/list/docs \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{
    "prefix": "reports/",
    "search": "q1",
    "sort_by": {"column": "name", "order": "asc"},
    "limit": 50
  }'</span>

<span class="t-res">&rarr; 200
[
  {"name":"q1.pdf","path":"reports/q1.pdf","size":524288},
  {"name":"q1-summary.md","path":"reports/q1-summary.md","size":2048}
]</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Create a signed download URL</div>
        <div class="example-badge">Signed URLs</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/object/sign/docs \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{"path":"report.pdf","expires_in":3600}'</span>

<span class="t-res">&rarr; 200
{
  "signed_url": "/sign/tok_abc123",
  "token": "tok_abc123",
  "expires_at": 1710896400000
}</span>

<span class="t-comment"># anyone can now download — no auth needed</span>
<span class="t-cmd">curl</span> storage.now/sign/tok_abc123 <span class="t-flag">-o</span> report.pdf</pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Signed upload (client-direct)</div>
        <div class="example-badge">Signed URLs</div>
      </div>
      <pre class="example-code"><span class="t-comment"># 1. Server creates upload URL</span>
<span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/object/upload/sign/docs/inbox/file.pdf \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span>

<span class="t-res">&rarr; {"signed_url":"/upload/sign/tok_xyz"}</span>

<span class="t-comment"># 2. Client uploads directly — no auth, no proxy</span>
<span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.now/upload/sign/tok_xyz \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/pdf"</span> \\
  <span class="t-flag">-T</span> file.pdf

<span class="t-res">&rarr; 201 {"path":"inbox/file.pdf","size":1048576}</span></pre>
    </div>
  </div>
</section>

<!-- Getting started -->
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
      <pre class="step-code"><span class="t-cmd">POST</span> /auth/register
<span class="t-str">{"actor":"a/my-bot",</span>
<span class="t-str"> "public_key":"&lt;ed25519&gt;"}</span>

<span class="t-res">&rarr; 201 {"actor":"a/my-bot"}</span>

<span class="t-cmd">POST</span> /auth/challenge
<span class="t-str">{"actor":"a/my-bot"}</span>

<span class="t-res">&rarr; {"challenge_id":"ch_..."}</span></pre>
    </div>
    <div class="step">
      <div class="step-num">02</div>
      <div class="step-name">Create a bucket</div>
      <p>Buckets hold your objects. Set visibility, file size limits, and allowed MIME types per bucket.</p>
      <pre class="step-code"><span class="t-cmd">POST</span> /bucket
<span class="t-str">Authorization: Bearer sk_...</span>

<span class="t-str">{"name":"my-files","public":false}</span>

<span class="t-res">&rarr; 201 {"id":"bk_...","name":"my-files"}</span></pre>
    </div>
    <div class="step">
      <div class="step-num">03</div>
      <div class="step-name">Upload &amp; share</div>
      <p>PUT files into your bucket. Generate signed URLs to share them. Build your product on three primitives.</p>
      <pre class="step-code"><span class="t-cmd">PUT</span> /object/my-files/hello.txt
<span class="t-str">Authorization: Bearer sk_...</span>
<span class="t-str">Content-Type: text/plain</span>

<span class="t-res">&rarr; 201 {"path":"hello.txt"}</span>

<span class="t-cmd">POST</span> /object/sign/my-files
<span class="t-str">{"path":"hello.txt","expires_in":86400}</span>

<span class="t-res">&rarr; {"signed_url":"/sign/tok_..."}</span></pre>
    </div>
  </div>
</section>

<!-- Architecture -->
<section class="section" id="arch">
  <div class="section-pad">
    <div class="section-label">ARCHITECTURE</div>
    <div class="section-heading">Edge-first. Zero hops.</div>
    <p class="section-desc">Every request hits the nearest edge location. Metadata resolves in under 50ms. File bytes flow directly between client and object storage.</p>
  </div>
  <div class="arch">
    <div class="arch-node">
      <div class="arch-icon">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
      </div>
      <div class="arch-name">Client</div>
      <div class="arch-desc">curl &middot; fetch &middot; SDK</div>
    </div>
    <div class="arch-arrow">
      <svg width="48" height="16" viewBox="0 0 48 16"><line x1="0" y1="8" x2="40" y2="8" stroke="var(--border)" stroke-width="1" stroke-dasharray="4 3"/><polyline points="38,4 46,8 38,12" fill="none" stroke="var(--text-3)" stroke-width="1"/></svg>
      <div class="arch-label">HTTPS</div>
    </div>
    <div class="arch-node arch-node--edge">
      <div class="arch-icon">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
      </div>
      <div class="arch-name">Edge (300+)</div>
      <div class="arch-desc">Auth &middot; buckets &middot; metadata</div>
    </div>
    <div class="arch-arrow">
      <svg width="48" height="16" viewBox="0 0 48 16"><line x1="0" y1="8" x2="40" y2="8" stroke="var(--border)" stroke-width="1" stroke-dasharray="4 3"/><polyline points="38,4 46,8 38,12" fill="none" stroke="var(--text-3)" stroke-width="1"/></svg>
      <div class="arch-label">R2</div>
    </div>
    <div class="arch-node">
      <div class="arch-icon">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/></svg>
      </div>
      <div class="arch-name">Object Store</div>
      <div class="arch-desc">Durable &middot; replicated</div>
    </div>
  </div>
</section>

<!-- API Surface -->
<section class="section" id="protocol">
  <div class="section-pad">
    <div class="section-label">THE API</div>
    <div class="section-heading">Six resource groups. Thirty endpoints.</div>
    <p class="section-desc">Inspired by Supabase Storage. Buckets for containers, objects for files, signed URLs for access control.</p>
    <table class="resource-table">
      <thead><tr><th>Resource</th><th>Endpoints</th><th>Purpose</th></tr></thead>
      <tbody>
        <tr><td>/bucket</td><td>6</td><td>Create, list, update, delete containers</td></tr>
        <tr><td>/object</td><td>10</td><td>Upload, download, list, move, copy, delete files</td></tr>
        <tr><td>/sign &middot; /upload/sign</td><td>4</td><td>Time-limited URLs for sharing &amp; direct upload</td></tr>
        <tr><td>/auth</td><td>6</td><td>Ed25519 challenge, magic link, sessions</td></tr>
        <tr><td>/keys</td><td>3</td><td>Scoped API keys for programmatic access</td></tr>
        <tr><td>/mcp &middot; /oauth</td><td>9</td><td>AI agent integration via MCP + OAuth 2.0</td></tr>
      </tbody>
    </table>
    <a href="/api" class="api-link">Full API reference &rarr;</a>
  </div>
</section>

<!-- Comparison -->
<section class="section" id="compare">
  <div class="section-pad">
    <div class="section-label">BEFORE &amp; AFTER</div>
    <div class="section-heading">Complexity, deleted.</div>
  </div>
  <div class="compare">
    <div class="compare-col compare-col--before">
      <div class="compare-label">Before</div>
      <div class="compare-list">
        <div class="compare-item compare-item--removed">/files/* <span>4 endpoints</span></div>
        <div class="compare-item compare-item--removed">/folders/* <span>3 endpoints</span></div>
        <div class="compare-item compare-item--removed">/shares/* <span>8 endpoints</span></div>
        <div class="compare-item compare-item--removed">/drive/* <span>13 endpoints</span></div>
        <div class="compare-item compare-item--removed">/links/* <span>3 endpoints</span></div>
        <div class="compare-item compare-item--removed">/presign/* <span>3 endpoints</span></div>
        <div class="compare-item compare-item--removed">/spaces/* <span>11 endpoints</span></div>
        <div class="compare-item">/auth/* <span>6 endpoints</span></div>
        <div class="compare-item">/api-keys <span>3 endpoints</span></div>
        <div class="compare-item">/mcp + /oauth <span>9 endpoints</span></div>
        <div class="compare-total">63 endpoints &middot; 10 resource groups</div>
      </div>
    </div>
    <div class="compare-arrow">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
    </div>
    <div class="compare-col compare-col--after">
      <div class="compare-label">After</div>
      <div class="compare-list">
        <div class="compare-item compare-item--new">/bucket <span>6 endpoints</span></div>
        <div class="compare-item compare-item--new">/object/* <span>10 endpoints</span></div>
        <div class="compare-item compare-item--new">/sign + /upload/sign <span>4 endpoints</span></div>
        <div class="compare-item">/auth/* <span>6 endpoints</span></div>
        <div class="compare-item">/keys <span>3 endpoints</span></div>
        <div class="compare-item">/mcp + /oauth <span>9 endpoints</span></div>
        <div class="compare-total compare-total--after">30 endpoints &middot; 6 resource groups</div>
      </div>
    </div>
  </div>
</section>

<!-- Stats -->
<section class="section" id="numbers">
  <div class="stats">
    <div class="stat">
      <div class="stat-num">30</div>
      <div class="stat-label">Endpoints</div>
    </div>
    <div class="stat">
      <div class="stat-num">6</div>
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

<!-- CTA -->
<section class="section section--cta">
  <div class="glow-spot glow-spot--cta"></div>
  <div class="section-pad">
    <div class="cta-label"><span class="cta-caret">&gt;</span> ready?</div>
    <div class="cta-title">Start building today</div>
    <p class="cta-desc">Create a bucket, upload your first object, and generate a signed URL in under 5 minutes.</p>
    <div class="cta-actions">
      <a href="/api" class="btn btn--primary btn--lg">API Reference</a>
      ${isSignedIn ? "" : `<a href="/" class="btn btn--ghost btn--lg">Create free account</a>`}
    </div>
  </div>
</section>

</main>

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
