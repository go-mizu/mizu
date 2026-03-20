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
<title>Developers — Storage</title>
<meta name="description" content="Developer-first file storage. Upload files, organize directories, share with signed URLs. One base URL, zero complexity.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/developers.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Storage</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/developers" class="active">developers</a>
      <a href="/api">api</a>
      <a href="/cli">cli</a>
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
    <h1 class="hero-title">Ship files,<br><span class="grad">not infrastructure.</span></h1>
    <p class="hero-sub">A REST API for storing, organizing, and sharing files. Write to a path, read from a path, share with a link. Works with any language, any framework, any platform.</p>
    <div class="hero-ctas">${heroCta}</div>
  </div>
  <div class="hero-terminal">
    <div class="terminal">
      <div class="terminal-bar">
        <div class="terminal-dots"><span></span><span></span><span></span></div>
        <div class="terminal-title">terminal</div>
      </div>
      <div class="terminal-body"><span class="t-comment"># upload a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.now/f/docs/report.pdf \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-T</span> report.pdf

<span class="t-res">201 {"path":"docs/report.pdf","size":524288}</span>

<span class="t-comment"># download a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> storage.now/f/docs/report.pdf \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-o</span> report.pdf

<span class="t-res">200  &lt;binary&gt;  Content-Type: application/pdf</span>

<span class="t-comment"># share with an expiring link</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/share \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-d</span> <span class="t-str">'{"path":"docs/report.pdf","ttl":3600}'</span>

<span class="t-res">200 {"url":"<a href='/s/tok_abc' class='t-link'>/s/tok_abc</a>","expires_at":...}</span></div>
    </div>
  </div>
</section>

<!-- Why -->
<section class="section" id="why">
  <div class="section-pad">
    <div class="section-label">WHY STORAGE</div>
    <div class="section-heading">Built for how you actually work.</div>
  </div>
  <div class="features">
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      </div>
      <div class="feature-name">REST-native</div>
      <p>JSON in, JSON out. Predictable URLs, standard HTTP methods, consistent error shapes. Works with <code>curl</code>, <code>fetch</code>, or any HTTP client in any language. No proprietary SDK required.</p>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
      </div>
      <div class="feature-name">Edge-first</div>
      <p>Requests resolve at the nearest edge from 300+ locations worldwide. Metadata queries complete in under 50ms. File bytes stream directly between your users and the object store.</p>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      </div>
      <div class="feature-name">Secure by default</div>
      <p>Ed25519 challenge-response for machines. Magic links for humans. Scoped API keys for long-running jobs. Signed share links for collaboration. Credentials never touch the client.</p>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/></svg>
      </div>
      <div class="feature-name">Zero config</div>
      <p>No infrastructure to provision, no storage classes to choose, no regions to configure. Write a file and it exists. Replication, caching, and global delivery are handled automatically.</p>
    </div>
  </div>
</section>

<!-- Three Concepts -->
<section class="section" id="concepts">
  <div class="section-pad">
    <div class="section-label">THE MODEL</div>
    <div class="section-heading">Three primitives. That's the whole API.</div>
    <p class="section-desc">Files addressed by path, directories from path structure, and sharing via signed links. No containers to create, no IDs to track.</p>
  </div>
  <div class="concepts">
    <div class="concept">
      <div class="concept-num">01</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="concept-name">Files</div>
      <p>Read and write files by path. Upload any content type, download with streaming. Use HEAD to stat without fetching bytes. Paths are your namespace.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-put">PUT</span> /f/*path</div>
        <div class="concept-method"><span class="m-get">GET</span> /f/*path</div>
        <div class="concept-method"><span class="m-delete">DEL</span> /f/*path</div>
        <div class="concept-method"><span class="m-get">HEAD</span> /f/*path</div>
        <div class="concept-method"><span class="m-post">POST</span> /mv</div>
      </div>
    </div>
    <div class="concept">
      <div class="concept-num">02</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
      </div>
      <div class="concept-name">Directories</div>
      <p>Directories are implicit from path separators. List contents by prefix, search by filename, delete entire trees with a trailing slash. No special creation step.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-get">GET</span> /ls/*prefix</div>
        <div class="concept-method"><span class="m-get">GET</span> /find?q=query</div>
        <div class="concept-method"><span class="m-delete">DEL</span> /f/dir/</div>
        <div class="concept-method"><span class="m-get">GET</span> /stat</div>
      </div>
    </div>
    <div class="concept">
      <div class="concept-num">03</div>
      <div class="concept-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
      </div>
      <div class="concept-name">Sharing</div>
      <p>Generate time-limited signed URLs for any file. Share private files with anyone. Recipients access via a short token URL. No auth required for recipients.</p>
      <div class="concept-code">
        <div class="concept-method"><span class="m-post">POST</span> /share</div>
        <div class="concept-method"><span class="m-get">GET</span> /s/:token</div>
      </div>
    </div>
  </div>
</section>

<!-- Code Examples -->
<section class="section" id="examples">
  <div class="section-pad">
    <div class="section-label">EXAMPLES</div>
    <div class="section-heading">Copy, paste, ship.</div>
  </div>
  <div class="examples">
    <div class="example">
      <div class="example-header">
        <div class="example-label">Upload a file</div>
        <div class="example-badge">Files</div>
      </div>
      <pre class="example-code"><span class="t-comment"># PUT creates or replaces</span>
<span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> storage.now/f/avatars/alice.png \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: image/png"</span> \\
  <span class="t-flag">-T</span> alice.png

<span class="t-res">&rarr; 201
{
  "path": "avatars/alice.png",
  "size": 48576
}</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Download a file</div>
        <div class="example-badge">Files</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> storage.now/f/avatars/alice.png \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-o</span> alice.png

<span class="t-res">&rarr; 200  &lt;binary&gt;
Content-Type: image/png
Content-Length: 48576</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">List &amp; search files</div>
        <div class="example-badge">Directories</div>
      </div>
      <pre class="example-code"><span class="t-comment"># list a directory</span>
<span class="t-cmd">curl</span> storage.now/ls/reports/ \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span>

<span class="t-res">&rarr; 200
[
  {"name":"q1.pdf","path":"reports/q1.pdf","size":524288},
  {"name":"q1-summary.md","path":"reports/q1-summary.md","size":2048}
]</span>

<span class="t-comment"># search by filename</span>
<span class="t-cmd">curl</span> <span class="t-str">"storage.now/find?q=q1"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span>

<span class="t-res">&rarr; 200
[
  {"name":"q1.pdf","path":"reports/q1.pdf","size":524288},
  {"name":"q1-summary.md","path":"reports/q1-summary.md","size":2048}
]</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Share with expiring links</div>
        <div class="example-badge">Sharing</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/share \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/json"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{"path":"docs/report.pdf","ttl":3600}'</span>

<span class="t-res">&rarr; 200
{
  "url": "/s/tok_abc123",
  "token": "tok_abc123",
  "expires_at": 1710896400000
}</span>

<span class="t-comment"># anyone can download, no auth required</span>
<span class="t-cmd">curl</span> storage.now/s/tok_abc123 <span class="t-flag">-o</span> report.pdf</pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Move &amp; rename files</div>
        <div class="example-badge">Files</div>
      </div>
      <pre class="example-code"><span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> storage.now/mv \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/json"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{"from":"drafts/report.pdf","to":"final/report.pdf"}'</span>

<span class="t-res">&rarr; 200
{
  "from": "drafts/report.pdf",
  "to": "final/report.pdf"
}</span></pre>
    </div>
    <div class="example">
      <div class="example-header">
        <div class="example-label">Check file metadata</div>
        <div class="example-badge">Files</div>
      </div>
      <pre class="example-code"><span class="t-comment"># HEAD returns metadata without body</span>
<span class="t-cmd">curl</span> <span class="t-flag">-I</span> storage.now/f/docs/report.pdf \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span>

<span class="t-res">&rarr; 200
Content-Type: application/pdf
Content-Length: 524288
Last-Modified: Thu, 20 Mar 2026 12:00:00 GMT</span>

<span class="t-comment"># storage usage stats</span>
<span class="t-cmd">curl</span> storage.now/stat \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span>

<span class="t-res">&rarr; 200
{"files":142,"bytes":67108864}</span></pre>
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
      <div class="step-name">Upload a file</div>
      <p>Write files by path. No setup, no containers. Just PUT to the path where you want the file to live.</p>
      <pre class="step-code"><span class="t-cmd">PUT</span> /f/my-files/hello.txt
<span class="t-str">Authorization: Bearer sk_...</span>
<span class="t-str">Content-Type: text/plain</span>

<span class="t-res">&rarr; 201 {"path":"my-files/hello.txt"}</span></pre>
    </div>
    <div class="step">
      <div class="step-num">03</div>
      <div class="step-name">Share it</div>
      <p>Generate a signed link to share any file. Recipients access it without authentication. Links expire automatically.</p>
      <pre class="step-code"><span class="t-cmd">POST</span> /share
<span class="t-str">Authorization: Bearer sk_...</span>

<span class="t-str">{"path":"my-files/hello.txt","ttl":86400}</span>

<span class="t-res">&rarr; {"url":"/s/tok_..."}</span>

<span class="t-comment"># anyone can access:</span>
<span class="t-cmd">GET</span> /s/tok_...
<span class="t-res">&rarr; 200  Hello, world!</span></pre>
    </div>
  </div>
</section>

<!-- Architecture -->
<section class="section" id="arch">
  <div class="section-pad">
    <div class="section-label">ARCHITECTURE</div>
    <div class="section-heading">Edge-first. Global by default.</div>
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
      <div class="arch-desc">Auth &middot; routing &middot; metadata</div>
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
    <div class="section-heading">Four resource groups. Seventeen endpoints.</div>
    <p class="section-desc">A flat REST interface covering files, discovery, sharing, and authentication.</p>
    <table class="resource-table">
      <thead><tr><th>Resource</th><th>Endpoints</th><th>Purpose</th></tr></thead>
      <tbody>
        <tr><td>/f/*path</td><td>4</td><td>Read, write, delete, stat files</td></tr>
        <tr><td>/ls &middot; /find &middot; /mv &middot; /stat</td><td>4</td><td>List directories, search, move, usage stats</td></tr>
        <tr><td>/share &middot; /s/:token</td><td>2</td><td>Create and access signed share links</td></tr>
        <tr><td>/auth &middot; /auth/keys</td><td>7</td><td>Register, challenge, verify, logout, API keys</td></tr>
      </tbody>
    </table>
    <a href="/api" class="api-link">Full API reference &rarr;</a>
  </div>
</section>

<!-- Stats -->
<section class="section" id="numbers">
  <div class="stats">
    <div class="stat">
      <div class="stat-num">17</div>
      <div class="stat-label">Endpoints</div>
    </div>
    <div class="stat">
      <div class="stat-num">4</div>
      <div class="stat-label">Resource groups</div>
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
    <p class="cta-desc">One base URL, seventeen endpoints, five minutes to your first upload.</p>
    <div class="cta-actions">
      <a href="/api" class="btn btn--primary btn--lg">API Reference</a>
      ${isSignedIn ? "" : `<a href="/" class="btn btn--ghost btn--lg">Create account</a>`}
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
