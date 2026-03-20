import { esc } from "./layout";

const CLAUDE_ICON = `<svg viewBox="0 0 24 24" width="14" height="14"><path fill="currentColor" d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>`;

export function developersPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  const heroCta = isSignedIn
    ? `<a href="/api" class="btn btn--primary">API Reference</a>
       <a href="/browse" class="btn btn--ghost">Dashboard</a>`
    : `<a href="/api" class="btn btn--primary">API Reference</a>
       <a href="/" class="btn btn--ghost">Get started free</a>`;

  const bottomCta2 = isSignedIn ? "" : `<a href="/" class="btn btn--ghost">Create account</a>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Storage for Developers</title>
<meta name="description" content="Upload a file in one request. Serve it globally. Zero egress fees. AI-native storage with MCP built in.">
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
     HERO — split: left text, right terminal
     ═══════════════════════════════════════════════════════════════════ -->
<section class="hero">
  <div class="glow-spot glow-spot--hero"></div>
  <div class="hero-split">
    <div class="hero-text">
      <div class="hero-badge">FOR DEVELOPERS</div>
      <h1 class="hero-title">One request to upload.<br><span class="grad">Zero to serve.</span></h1>
      <p class="hero-sub">A flat REST API for files. No SDK, no buckets, no egress fees. Serve from 300+ edge locations. Connect to Claude and ChatGPT via MCP.</p>
      <div class="hero-ctas">${heroCta}</div>
    </div>
    <div class="hero-code">
      <div class="term">
        <div class="term-bar"><span class="term-dots"><i></i><i></i><i></i></span><span class="term-title">terminal</span></div>
        <pre class="term-body"><span class="c-dim"># upload</span>
<span class="c-mute">$</span> <span class="c-bold">curl</span> -X POST /files/uploads \\
    -H <span class="c-str">"Authorization: Bearer $TOKEN"</span> \\
    -d <span class="c-str">'{"path":"assets/logo.svg"}'</span>

<span class="c-dim"># share it</span>
<span class="c-mute">$</span> <span class="c-bold">curl</span> -X POST /files/share \\
    -d <span class="c-str">'{"path":"assets/logo.svg"}'</span>
<span class="c-dim c-it">&rarr; /s/k7f2m &middot; expires 24h</span></pre>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     QUICKSTART — compact numbered flow, not code walls
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="quickstart">
  <div class="inner">
    <div class="sec-label">QUICKSTART</div>
    <h2 class="sec-h">30 seconds to first upload</h2>
  </div>
  <div class="qs-flow">
    <div class="qs-step">
      <div class="qs-num">1</div>
      <div class="qs-body">
        <div class="qs-title">Get a token</div>
        <div class="qs-desc">Register with your Ed25519 key or email &mdash; get a bearer token.</div>
        <code class="qs-ep">POST /auth/challenge &rarr; POST /auth/verify</code>
      </div>
    </div>
    <div class="qs-line"></div>
    <div class="qs-step">
      <div class="qs-num">2</div>
      <div class="qs-body">
        <div class="qs-title">Upload a file</div>
        <div class="qs-desc">Initiate upload, PUT to presigned URL, confirm.</div>
        <code class="qs-ep">POST /files/uploads &rarr; PUT &lt;signed-url&gt; &rarr; POST /files/uploads/complete</code>
      </div>
    </div>
    <div class="qs-line"></div>
    <div class="qs-step">
      <div class="qs-num">3</div>
      <div class="qs-body">
        <div class="qs-title">Share it</div>
        <div class="qs-desc">Generate a time-limited link. Anyone can access it, no auth needed.</div>
        <code class="qs-ep">POST /files/share &rarr; GET /s/:token</code>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     COMPARISON TABLE — the conversion section
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="compare">
  <div class="inner">
    <div class="sec-label">WHY STORAGE</div>
    <h2 class="sec-h">Why not just use S3?</h2>
    <p class="sec-sub">Because you'll spend more time on IAM policies than on your product.</p>
  </div>
  <div class="tbl-wrap">
    <table class="cmp">
      <thead>
        <tr>
          <th scope="col"></th>
          <th scope="col">AWS S3</th>
          <th scope="col">Supabase Storage</th>
          <th scope="col">Vercel Blob</th>
          <th scope="col" class="cmp-hl">Storage</th>
        </tr>
      </thead>
      <tbody>
        <tr><th scope="row">Egress cost</th><td>$90 / TB</td><td>Metered</td><td>Limited</td><td class="cmp-hl"><strong>$0</strong></td></tr>
        <tr><th scope="row">Time to first upload</th><td>Hours</td><td>~10 min</td><td>~5 min</td><td class="cmp-hl"><strong>30 sec</strong></td></tr>
        <tr><th scope="row">SDK required</th><td>aws-sdk</td><td>@supabase/storage-js</td><td>@vercel/blob</td><td class="cmp-hl"><strong>No</strong></td></tr>
        <tr><th scope="row">Auth options</th><td>IAM</td><td>Supabase Auth</td><td>Vercel token</td><td class="cmp-hl"><strong>4 methods</strong></td></tr>
        <tr><th scope="row">AI / MCP</th><td class="cmp-no">&times;</td><td class="cmp-no">&times;</td><td class="cmp-no">&times;</td><td class="cmp-hl cmp-yes">&check; Built-in</td></tr>
        <tr><th scope="row">Buckets to manage</th><td>Required</td><td>Required</td><td>Hidden</td><td class="cmp-hl"><strong>None</strong></td></tr>
        <tr><th scope="row">Region config</th><td>Pick one</td><td>Pick one</td><td>Auto</td><td class="cmp-hl"><strong>Global</strong></td></tr>
        <tr><th scope="row">File addressing</th><td>s3://bucket/key</td><td>/object/{bucket}/{path}</td><td>Blob URL</td><td class="cmp-hl"><strong>/files/{path}</strong></td></tr>
      </tbody>
    </table>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     THE API — visual flow, not endpoint listing
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="api">
  <div class="inner">
    <div class="sec-label">THE API</div>
    <h2 class="sec-h">Everything under <code>/files</code></h2>
    <p class="sec-sub">One resource namespace. Standard HTTP methods. Paths are your filesystem.</p>
  </div>
  <div class="api-row">
    <div class="api-card">
      <div class="api-icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></div>
      <h3>Store</h3>
      <div class="api-endpoints">
        <div class="ep"><span class="m-post">POST</span> /files/uploads</div>
        <div class="ep"><span class="m-get">GET</span> /files/{path}</div>
        <div class="ep"><span class="m-get">HEAD</span> /files/{path}</div>
        <div class="ep"><span class="m-delete">DEL</span> /files/{path}</div>
      </div>
    </div>
    <div class="api-card">
      <div class="api-icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></div>
      <h3>Organize</h3>
      <div class="api-endpoints">
        <div class="ep"><span class="m-get">GET</span> /files?prefix=</div>
        <div class="ep"><span class="m-get">GET</span> /files/search?q=</div>
        <div class="ep"><span class="m-post">POST</span> /files/move</div>
        <div class="ep"><span class="m-get">GET</span> /files/stats</div>
      </div>
    </div>
    <div class="api-card">
      <div class="api-icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg></div>
      <h3>Share</h3>
      <div class="api-endpoints">
        <div class="ep"><span class="m-post">POST</span> /files/share</div>
        <div class="ep"><span class="m-get">GET</span> /s/:token</div>
      </div>
    </div>
  </div>
  <div class="api-note">That&rsquo;s the whole API. <a href="/api">Full reference &rarr;</a></div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     ANY LANGUAGE — compact code tabs
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="languages">
  <div class="inner">
    <div class="sec-label">ANY LANGUAGE</div>
    <h2 class="sec-h">No SDK. Just HTTP.</h2>
  </div>
  <div class="lang-box">
    <div class="lang-tabs" role="tablist">
      <button class="lang-tab active" data-lang="curl" onclick="switchLang('curl')" role="tab" aria-selected="true">curl</button>
      <button class="lang-tab" data-lang="js" onclick="switchLang('js')" role="tab" aria-selected="false">JavaScript</button>
      <button class="lang-tab" data-lang="python" onclick="switchLang('python')" role="tab" aria-selected="false">Python</button>
      <button class="lang-tab" data-lang="go" onclick="switchLang('go')" role="tab" aria-selected="false">Go</button>
    </div>
    <pre class="lang-panel active" id="lang-curl" role="tabpanel"><span class="c-dim"># initiate upload</span>
<span class="c-bold">curl</span> -X POST https://storage.liteio.dev/files/uploads \\
  -H <span class="c-str">"Authorization: Bearer $TOKEN"</span> \\
  -d <span class="c-str">'{"path":"data/config.json","content_type":"application/json"}'</span>

<span class="c-dim"># download (follows 302 redirect to R2)</span>
<span class="c-bold">curl</span> -L https://storage.liteio.dev/files/data/config.json \\
  -H <span class="c-str">"Authorization: Bearer $TOKEN"</span></pre>
    <pre class="lang-panel" id="lang-js" role="tabpanel"><span class="c-dim">// initiate upload, then PUT to the presigned URL</span>
<span class="c-bold">const</span> { upload_url } = <span class="c-bold">await</span> fetch(<span class="c-str">"https://storage.liteio.dev/files/uploads"</span>, {
  method: <span class="c-str">"POST"</span>,
  headers: { Authorization: <span class="c-str">\`Bearer \${TOKEN}\`</span> },
  body: JSON.stringify({ path: <span class="c-str">"data/config.json"</span> }),
}).then(r =&gt; r.json());

<span class="c-bold">await</span> fetch(upload_url, { method: <span class="c-str">"PUT"</span>, body: file });</pre>
    <pre class="lang-panel" id="lang-python" role="tabpanel"><span class="c-bold">import</span> requests

<span class="c-dim"># initiate upload</span>
res = requests.post(
    <span class="c-str">"https://storage.liteio.dev/files/uploads"</span>,
    headers={<span class="c-str">"Authorization"</span>: <span class="c-str">f"Bearer {TOKEN}"</span>},
    json={<span class="c-str">"path"</span>: <span class="c-str">"data/config.json"</span>},
)
upload_url = res.json()[<span class="c-str">"upload_url"</span>]

<span class="c-dim"># upload directly to object store</span>
requests.put(upload_url, data=file_bytes)</pre>
    <pre class="lang-panel" id="lang-go" role="tabpanel">
<span class="c-dim">// initiate upload</span>
body := strings.NewReader(<span class="c-str">\`{"path":"data/config.json"}\`</span>)
req, _ := http.NewRequest(<span class="c-str">"POST"</span>,
    <span class="c-str">"https://storage.liteio.dev/files/uploads"</span>, body)
req.Header.Set(<span class="c-str">"Authorization"</span>, <span class="c-str">"Bearer "</span>+token)

resp, _ := http.DefaultClient.Do(req)
<span class="c-dim">// parse upload_url from response, PUT file there</span></pre>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     AI-NATIVE — immersive chat mockup
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec sec--flush" id="ai">
  <div class="inner">
    <div class="sec-label">AI-NATIVE</div>
    <h2 class="sec-h">Your files, inside Claude and ChatGPT.</h2>
    <p class="sec-sub">MCP protocol built in. 8 tools. Connect once, then manage files with natural language.</p>
  </div>
  <div class="ai-wrap">
    <div class="ai-left">
      <div class="ai-cap"><span class="ai-say">"Save this as report.md"</span><span class="ai-tool">storage_write</span></div>
      <div class="ai-cap"><span class="ai-say">"What files do I have?"</span><span class="ai-tool">storage_list</span></div>
      <div class="ai-cap"><span class="ai-say">"Find all CSV files"</span><span class="ai-tool">storage_search</span></div>
      <div class="ai-cap"><span class="ai-say">"Share the report"</span><span class="ai-tool">storage_share</span></div>
      <div class="ai-cap"><span class="ai-say">"How much space am I using?"</span><span class="ai-tool">storage_stats</span></div>
      <div class="ai-cap"><span class="ai-say">"Move it to /work"</span><span class="ai-tool">storage_move</span></div>
      <div class="ai-connect">
        <div class="ai-cc"><strong>Claude</strong><br>Settings &rarr; Integrations &rarr; Add<br><code>https://storage.liteio.dev/mcp</code></div>
        <div class="ai-cc"><strong>ChatGPT</strong><br>Settings &rarr; Connected apps<br><code>https://storage.liteio.dev/mcp</code></div>
      </div>
    </div>
    <div class="ai-right">
      <div class="term term--chat">
        <div class="term-bar"><span class="term-dots"><i></i><i></i><i></i></span><span class="term-title">claude.ai</span></div>
        <div class="chat">
          <div class="chat-user">Save the meeting notes as notes/2025-03-20.md</div>
          <div class="chat-bot"><span class="chat-av">${CLAUDE_ICON}</span><span>Done! Saved <strong>notes/2025-03-20.md</strong> (4.2 KB)</span></div>
          <div class="chat-user">What files do I have in notes/?</div>
          <div class="chat-bot"><span class="chat-av">${CLAUDE_ICON}</span><span>Your <strong>notes/</strong> folder has 3 files:<br>&bull; 2025-03-20.md &mdash; 4.2 KB<br>&bull; 2025-03-18.md &mdash; 2.1 KB<br>&bull; ideas.txt &mdash; 890 B</span></div>
          <div class="chat-user">Share the latest one</div>
          <div class="chat-bot"><span class="chat-av">${CLAUDE_ICON}</span><span>Here&rsquo;s your link:<br><code>storage.liteio.dev/s/m9x2k</code><br>Expires in 24 hours.</span></div>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     ARCHITECTURE — horizontal flow
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="arch">
  <div class="inner">
    <div class="sec-label">ARCHITECTURE</div>
    <h2 class="sec-h">Edge-first. Zero proxy.</h2>
    <p class="sec-sub">File bytes never touch our servers. Auth at the edge, data direct to the object store.</p>
  </div>
  <div class="arch">
    <div class="arch-node"><div class="arch-ico"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg></div><div class="arch-nm">Your App</div><div class="arch-d">curl &middot; fetch &middot; CLI</div></div>
    <div class="arch-arr"><span class="arch-lbl">HTTPS</span></div>
    <div class="arch-node arch-node--hl"><div class="arch-ico"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg></div><div class="arch-nm">Edge Worker</div><div class="arch-d">300+ &middot; &lt;50ms &middot; Auth + Sign</div></div>
    <div class="arch-arr"><span class="arch-lbl">Presigned</span></div>
    <div class="arch-node"><div class="arch-ico"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/></svg></div><div class="arch-nm">R2 Store</div><div class="arch-d">Durable &middot; $0 egress</div></div>
  </div>
  <div class="arch-row">
    <div class="arch-fact"><strong>Auth at the edge</strong> Token verification at the nearest of 300+ locations. Sub-millisecond.</div>
    <div class="arch-fact"><strong>Zero-proxy transfers</strong> Clients upload and download directly via presigned URLs. No Worker bandwidth.</div>
    <div class="arch-fact"><strong>Metadata in D1</strong> File index, sessions, and shares in edge SQLite. Fast reads, strong consistency.</div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     USE CASES — text-driven, not code walls
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="cases">
  <div class="inner">
    <div class="sec-label">USE CASES</div>
    <h2 class="sec-h">Drop it into any stack.</h2>
  </div>
  <div class="use-row">
    <div class="use-card">
      <div class="use-badge">FRONTEND</div>
      <h3>App file uploads</h3>
      <p>Get a presigned URL from the API, upload directly from the browser to R2. No server proxy. Works with any framework &mdash; React, Vue, Svelte, vanilla JS.</p>
    </div>
    <div class="use-card">
      <div class="use-badge">DEVOPS</div>
      <h3>CI/CD artifacts</h3>
      <p>Push build outputs, deploy bundles, and test reports from GitHub Actions or any CI. One <code>curl</code> per artifact. Scoped API keys for each pipeline.</p>
    </div>
    <div class="use-card">
      <div class="use-badge">MCP</div>
      <h3>AI workflows</h3>
      <p>Let Claude or ChatGPT save research, code snippets, and generated files directly to your storage. Search and share them by asking.</p>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     SECURITY — compact inline grid
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="security">
  <div class="inner">
    <div class="sec-label">SECURITY</div>
    <h2 class="sec-h">Secure by default. No passwords stored.</h2>
  </div>
  <div class="sec-grid">
    <div class="sg"><strong>Ed25519 auth</strong> Public key challenge-response. No shared secrets.</div>
    <div class="sg"><strong>Scoped API keys</strong> Path-prefix restrictions. <code>sk_*</code> format, 90-day TTL.</div>
    <div class="sg"><strong>Signed share links</strong> Time-limited URLs. 60s to 7 days. Auto-expire.</div>
    <div class="sg"><strong>OAuth 2.0 + PKCE</strong> Standard flow for third-party apps. Dynamic client registration.</div>
    <div class="sg"><strong>Rate limiting</strong> Per-endpoint sliding window. Auth: 10/min. Uploads: 100/min.</div>
    <div class="sg"><strong>Audit logging</strong> Every action logged with actor, resource, and timestamp.</div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     CTA
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec sec--cta">
  <div class="glow-spot glow-spot--cta"></div>
  <div class="inner cta-inner">
    <h2 class="cta-h">Start building.</h2>
    <p class="cta-sub">One request to upload. Zero egress to serve. Connect AI in a minute.</p>
    <div class="cta-actions">${heroCta}${bottomCta2}</div>
  </div>
</section>

</main>

<script>
function toggleTheme(){var d=document.documentElement.classList.toggle('dark');localStorage.setItem('theme',d?'dark':'light')}
(function(){var s=localStorage.getItem('theme');if(s==='light')document.documentElement.classList.remove('dark');else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();
(function(){var els=document.querySelectorAll('.sec');if(!els.length)return;var obs=new IntersectionObserver(function(es){es.forEach(function(e){if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}})},{threshold:0.06,rootMargin:'0px 0px -40px 0px'});els.forEach(function(s){obs.observe(s)})})();
function switchLang(l){document.querySelectorAll('.lang-tab').forEach(function(t){t.classList.remove('active');t.setAttribute('aria-selected','false')});document.querySelectorAll('.lang-panel').forEach(function(p){p.classList.remove('active')});var tab=document.querySelector('.lang-tab[data-lang="'+l+'"]');if(tab){tab.classList.add('active');tab.setAttribute('aria-selected','true')}var p=document.getElementById('lang-'+l);if(p)p.classList.add('active')}
</script>
</body>
</html>`;
}
