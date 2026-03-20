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
<title>Storage for Developers</title>
<meta name="description" content="A flat REST API for storing, organizing, and sharing files. Works with any language. No SDK required.">
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
      <a href="/ai">ai</a>
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

<!-- ═══════════════════════════════════════════════════════════════════
     HERO
     ═══════════════════════════════════════════════════════════════════ -->
<section class="hero">
  <div class="glow-spot glow-spot--hero"></div>
  <div class="hero-inner">
    <div class="hero-badge">STORAGE FOR DEVELOPERS</div>
    <h1 class="hero-title">One API for<br><span class="grad">every file.</span></h1>
    <p class="hero-sub">Store and serve files on a global edge network.<br>One REST interface. Any language. No SDK.</p>
    <div class="hero-ctas">${heroCta}</div>
  </div>
  <div class="hero-terminal">
    <div class="terminal">
      <div class="terminal-bar">
        <div class="terminal-dots"><span></span><span></span><span></span></div>
        <div class="terminal-title">terminal</div>
      </div>
      <div class="terminal-body"><span class="t-comment"># upload a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> https://storage.liteio.dev/f/assets/logo.svg \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-T</span> logo.svg

<span class="t-res">&rarr; 201  &middot;  assets/logo.svg  &middot;  4.8 KB  &middot;  image/svg+xml</span>

<span class="t-comment"># share with anyone (expires in 24h)</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl</span> <span class="t-flag">-X POST</span> https://storage.liteio.dev/share \\
    <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
    <span class="t-flag">-d</span> <span class="t-str">'{"path":"assets/logo.svg","ttl":86400}'</span>

<span class="t-res">&rarr; 200  &middot;  <a href='/s/k7f2m' class='t-link'>/s/k7f2m</a>  &middot;  expires in 24h</span></div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     METRICS BAR
     ═══════════════════════════════════════════════════════════════════ -->
<section class="metrics section">
  <div class="metrics-inner">
    <div class="metric">
      <div class="metric-value">17</div>
      <div class="metric-label">Endpoints</div>
    </div>
    <div class="metric-sep"></div>
    <div class="metric">
      <div class="metric-value">300+</div>
      <div class="metric-label">Edge locations</div>
    </div>
    <div class="metric-sep"></div>
    <div class="metric">
      <div class="metric-value">&lt;50ms</div>
      <div class="metric-label">Global latency</div>
    </div>
    <div class="metric-sep"></div>
    <div class="metric">
      <div class="metric-value">$0</div>
      <div class="metric-label">Egress fees</div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     THE API — Three primitives
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="api">
  <div class="section-pad">
    <div class="section-label">THE API</div>
    <h2 class="section-heading">Three primitives. That's it.</h2>
    <p class="section-desc">Files addressed by path, directories from structure, sharing via signed links. No buckets, no containers, no IDs.</p>
  </div>
  <div class="pillars">
    <div class="pillar">
      <div class="pillar-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <h3 class="pillar-name">Files</h3>
      <p class="pillar-desc">Read and write any file by path. Upload any content type, stream downloads. Paths are your namespace.</p>
      <div class="pillar-endpoints">
        <div class="endpoint"><span class="m-put">PUT</span> /f/*path</div>
        <div class="endpoint"><span class="m-get">GET</span> /f/*path</div>
        <div class="endpoint"><span class="m-delete">DEL</span> /f/*path</div>
        <div class="endpoint"><span class="m-get">HEAD</span> /f/*path</div>
      </div>
    </div>
    <div class="pillar">
      <div class="pillar-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
      </div>
      <h3 class="pillar-name">Discovery</h3>
      <p class="pillar-desc">Directories are implicit from path separators. List contents, search by name, move files, check usage.</p>
      <div class="pillar-endpoints">
        <div class="endpoint"><span class="m-get">GET</span> /ls/*prefix</div>
        <div class="endpoint"><span class="m-get">GET</span> /find?q=</div>
        <div class="endpoint"><span class="m-post">POST</span> /mv</div>
        <div class="endpoint"><span class="m-get">GET</span> /stat</div>
      </div>
    </div>
    <div class="pillar">
      <div class="pillar-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
      </div>
      <h3 class="pillar-name">Sharing</h3>
      <p class="pillar-desc">Generate time-limited signed URLs for any file. Recipients access via a short token. No auth required for them.</p>
      <div class="pillar-endpoints">
        <div class="endpoint"><span class="m-post">POST</span> /share</div>
        <div class="endpoint"><span class="m-get">GET</span> /s/:token</div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     DEVELOPER EXPERIENCE — Feature grid
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="dx">
  <div class="section-pad">
    <div class="section-label">DEVELOPER EXPERIENCE</div>
    <h2 class="section-heading">Built for the way you work.</h2>
  </div>
  <div class="dx-grid">
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      </div>
      <h3 class="dx-name">REST-native</h3>
      <p>Standard HTTP methods, predictable URLs, consistent JSON responses. Works with <code>curl</code>, <code>fetch</code>, or any HTTP client.</p>
    </div>
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
      </div>
      <h3 class="dx-name">Edge-first</h3>
      <p>Every request resolves at the nearest edge from 300+ locations. Metadata queries complete in under 50ms globally.</p>
    </div>
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      </div>
      <h3 class="dx-name">Secure by default</h3>
      <p>Ed25519 challenge-response for machines. Magic links for humans. Signed share links for collaboration.</p>
    </div>
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/></svg>
      </div>
      <h3 class="dx-name">Zero config</h3>
      <p>No infrastructure to provision, no storage classes, no regions to choose. Write a file and it exists globally.</p>
    </div>
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="16" rx="2"/><line x1="4" y1="10" x2="20" y2="10"/><line x1="10" y1="4" x2="10" y2="20"/></svg>
      </div>
      <h3 class="dx-name">CLI included</h3>
      <p>Upload, download, list, and share from your terminal. Scriptable. Pipeable. macOS, Linux, and Windows.</p>
    </div>
    <div class="dx-card">
      <div class="dx-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
      </div>
      <h3 class="dx-name">AI-native</h3>
      <p>MCP protocol support built in. Connect to Claude or ChatGPT and let AI manage files with natural language.</p>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     ANY LANGUAGE — Multi-language code showcase
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="languages">
  <div class="section-pad">
    <div class="section-label">ANY LANGUAGE</div>
    <h2 class="section-heading">No SDK. Just HTTP.</h2>
    <p class="section-desc">If it can make an HTTP request, it works with Storage. The same upload in four languages.</p>
  </div>
  <div class="lang-showcase">
    <div class="lang-tabs">
      <button class="lang-tab active" data-lang="curl" onclick="switchLang('curl')">curl</button>
      <button class="lang-tab" data-lang="js" onclick="switchLang('js')">JavaScript</button>
      <button class="lang-tab" data-lang="python" onclick="switchLang('python')">Python</button>
      <button class="lang-tab" data-lang="go" onclick="switchLang('go')">Go</button>
    </div>
    <div class="lang-panels">
      <pre class="lang-panel active" id="lang-curl"><span class="t-cmd">curl</span> <span class="t-flag">-X PUT</span> https://storage.liteio.dev/f/data/config.json \\
  <span class="t-flag">-H</span> <span class="t-str">"Authorization: Bearer $TOKEN"</span> \\
  <span class="t-flag">-H</span> <span class="t-str">"Content-Type: application/json"</span> \\
  <span class="t-flag">-d</span> <span class="t-str">'{"theme":"dark","lang":"en"}'</span>

<span class="t-res">&rarr; 201 {"path":"data/config.json","size":32}</span></pre>
      <pre class="lang-panel" id="lang-js"><span class="t-kw">const</span> res = <span class="t-kw">await</span> <span class="t-cmd">fetch</span>(<span class="t-str">"https://storage.liteio.dev/f/data/config.json"</span>, {
  method: <span class="t-str">"PUT"</span>,
  headers: {
    <span class="t-str">"Authorization"</span>: <span class="t-str">\`Bearer \${TOKEN}\`</span>,
    <span class="t-str">"Content-Type"</span>: <span class="t-str">"application/json"</span>,
  },
  body: JSON.stringify({ theme: <span class="t-str">"dark"</span>, lang: <span class="t-str">"en"</span> }),
});

<span class="t-kw">const</span> data = <span class="t-kw">await</span> res.json();
<span class="t-comment">// → {path: "data/config.json", size: 32}</span></pre>
      <pre class="lang-panel" id="lang-python"><span class="t-kw">import</span> requests

res = requests.<span class="t-cmd">put</span>(
    <span class="t-str">"https://storage.liteio.dev/f/data/config.json"</span>,
    headers={<span class="t-str">"Authorization"</span>: <span class="t-str">f"Bearer {TOKEN}"</span>},
    json={<span class="t-str">"theme"</span>: <span class="t-str">"dark"</span>, <span class="t-str">"lang"</span>: <span class="t-str">"en"</span>},
)

<span class="t-comment"># → {"path": "data/config.json", "size": 32}</span></pre>
      <pre class="lang-panel" id="lang-go">body := strings.NewReader(<span class="t-str">\`{"theme":"dark","lang":"en"}\`</span>)
req, _ := http.<span class="t-cmd">NewRequest</span>(<span class="t-str">"PUT"</span>,
    <span class="t-str">"https://storage.liteio.dev/f/data/config.json"</span>, body)
req.Header.Set(<span class="t-str">"Authorization"</span>, <span class="t-str">"Bearer "</span>+token)
req.Header.Set(<span class="t-str">"Content-Type"</span>, <span class="t-str">"application/json"</span>)

resp, _ := http.DefaultClient.<span class="t-cmd">Do</span>(req)
<span class="t-comment">// → 201 {"path":"data/config.json","size":32}</span></pre>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     IN PRACTICE — Real-world integration
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="practice">
  <div class="section-pad">
    <div class="section-label">IN PRACTICE</div>
    <h2 class="section-heading">Drop it into any stack.</h2>
    <p class="section-desc">Storage fits into your existing application with a few lines of code. No SDK to install, no config to manage.</p>
  </div>
  <div class="practice-grid">
    <div class="practice-card">
      <div class="practice-header">
        <div class="practice-label">Next.js API Route</div>
        <div class="practice-badge">FRAMEWORK</div>
      </div>
      <pre class="practice-code"><span class="t-kw">export async function</span> <span class="t-cmd">POST</span>(req: Request) {
  <span class="t-kw">const</span> form = <span class="t-kw">await</span> req.formData();
  <span class="t-kw">const</span> file = form.get(<span class="t-str">"file"</span>) <span class="t-kw">as</span> File;

  <span class="t-comment">// Upload to Storage — just a fetch call</span>
  <span class="t-kw">const</span> res = <span class="t-kw">await</span> <span class="t-cmd">fetch</span>(
    <span class="t-str">\`https://storage.liteio.dev/f/uploads/\${file.name}\`</span>,
    {
      method: <span class="t-str">"PUT"</span>,
      headers: { Authorization: <span class="t-str">\`Bearer \${process.env.TOKEN}\`</span> },
      body: file,
    }
  );

  <span class="t-kw">return</span> Response.json(<span class="t-kw">await</span> res.json());
}</pre>
    </div>
    <div class="practice-card">
      <div class="practice-header">
        <div class="practice-label">CLI Workflow</div>
        <div class="practice-badge">TERMINAL</div>
      </div>
      <pre class="practice-code"><span class="t-comment"># upload an entire directory</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage</span> upload ./dist assets/v2.1/
<span class="t-res">  assets/v2.1/index.html     2.4 KB  &check;</span>
<span class="t-res">  assets/v2.1/app.js        48.1 KB  &check;</span>
<span class="t-res">  assets/v2.1/style.css      6.8 KB  &check;</span>
<span class="t-res">  3 files  &middot;  57.3 KB  &middot;  142ms</span>

<span class="t-comment"># share the whole folder</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage</span> share assets/v2.1/ <span class="t-flag">--ttl</span> 7d
<span class="t-res">  &rarr; /s/m9x2k  &middot;  expires in 7 days</span></pre>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     ARCHITECTURE
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="arch">
  <div class="section-pad">
    <div class="section-label">ARCHITECTURE</div>
    <h2 class="section-heading">Edge-first. Global by default.</h2>
    <p class="section-desc">Every request resolves at the nearest edge. Auth and routing happen there. File bytes stream directly to and from the object store.</p>
  </div>
  <div class="arch">
    <div class="arch-node">
      <div class="arch-icon">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
      </div>
      <div class="arch-name">Your App</div>
      <div class="arch-desc">curl &middot; fetch &middot; SDK &middot; CLI</div>
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
      <div class="arch-desc">Durable &middot; replicated &middot; $0 egress</div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     CTA
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section section--cta">
  <div class="glow-spot glow-spot--cta"></div>
  <div class="section-pad">
    <h2 class="cta-title">Start building today.</h2>
    <p class="cta-desc">Seventeen endpoints. Five minutes to first upload. Zero egress fees.</p>
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
function switchLang(lang){
  document.querySelectorAll('.lang-tab').forEach(t=>t.classList.remove('active'));
  document.querySelectorAll('.lang-panel').forEach(p=>p.classList.remove('active'));
  document.querySelector('.lang-tab[data-lang="'+lang+'"]').classList.add('active');
  document.getElementById('lang-'+lang).classList.add('active');
}
</script>
</body>
</html>`;
}
