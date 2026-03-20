import { esc } from "./layout";

export function cliPage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CLI &mdash; storage.now</title>
<meta name="description" content="The storage.now command-line interface. One file, zero dependencies. Upload, download, and share from your terminal.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#FAFAF9;--surface:#FFF;--surface-alt:#F4F4F5;--surface-2:#E4E4E7;
  --text:#09090B;--text-2:#52525B;--text-3:#A1A1AA;
  --border:#E4E4E7;--ink:#09090B;
  --green:#22C55E;--blue:#3B82F6;--amber:#F59E0B;--red:#EF4444;--purple:#A855F7;
}
html.dark{
  --bg:#09090B;--surface:#111113;--surface-alt:#18181B;--surface-2:#27272A;
  --text:#FAFAF9;--text-2:#A1A1AA;--text-3:#52525B;
  --border:#1E1E21;--ink:#FAFAF9;
  --green:#4ADE80;--blue:#60A5FA;--amber:#FBBF24;--red:#F87171;--purple:#C084FC;
}
html{scroll-behavior:smooth}
body{font-family:'Inter',system-ui,sans-serif;color:var(--text);background:var(--bg);
  -webkit-font-smoothing:antialiased;overflow-x:hidden}
a{color:inherit;text-decoration:none}
code{font-family:'JetBrains Mono',monospace;font-size:0.85em;
  background:var(--surface-alt);padding:2px 6px;border:1px solid var(--border)}

@keyframes fadeUp{from{opacity:0;transform:translateY(32px)}to{opacity:1;transform:none}}

.grad{
  background:linear-gradient(135deg,var(--text) 0%,var(--text-3) 100%);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;
  background-clip:text;font-weight:800}

/* ── Grid bg ── */
.grid-bg{position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:
    linear-gradient(var(--border) 1px,transparent 1px),
    linear-gradient(90deg,var(--border) 1px,transparent 1px);
  background-size:80px 80px;opacity:.25}
html.dark .grid-bg{opacity:.06}
.grid-bg::after{content:'';position:fixed;inset:0;pointer-events:none;
  background-image:radial-gradient(circle 1px,var(--text-3) 0.5px,transparent 0.5px);
  background-size:80px 80px;opacity:.15}
html.dark .grid-bg::after{opacity:.1}

/* ── Nav ── */
nav{position:sticky;top:0;z-index:100;width:100%;
  background:color-mix(in srgb,var(--bg) 85%,transparent);
  backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px)}
.nav-inner{width:100%;padding:0 48px;height:64px;
  display:flex;align-items:center;justify-content:space-between}
.logo{font-weight:700;font-size:15px;letter-spacing:-0.3px;
  display:flex;align-items:center;gap:8px}
.logo-dot{width:6px;height:6px;background:var(--text);display:inline-block}
.nav-links{display:flex;gap:32px}
.nav-links a{font-size:14px;color:var(--text-3);transition:color .15s;font-weight:500}
.nav-links a:hover,.nav-links a.active{color:var(--text)}
.nav-right{display:flex;align-items:center;gap:16px}
.nav-user{font-size:14px;color:var(--text-2);font-weight:500}
.nav-signout{font-size:13px;color:var(--text-3);border:1px solid var(--border);
  padding:6px 16px;transition:all .15s;font-weight:500}
.nav-signout:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle{background:none;border:none;padding:8px;cursor:pointer;
  color:var(--text-3);display:flex;align-items:center;transition:all .15s}
.theme-toggle:hover{color:var(--text)}
.theme-toggle .icon-sun{display:none}
.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}
html.dark .theme-toggle .icon-moon{display:none}
.mobile-toggle{display:none;background:none;border:none;color:var(--text-3);cursor:pointer;padding:4px}

/* ── Sections ── */
.section{position:relative;z-index:1;width:100%;padding:120px 0;
  opacity:0;transform:translateY(32px);transition:opacity .8s ease,transform .8s ease}
.section.visible{opacity:1;transform:none}
.section-pad{max-width:1200px;margin:0 auto;padding:0 48px}
.section-label{font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:2px;
  color:var(--text-3);margin-bottom:20px;font-weight:500;text-transform:uppercase}
.section-heading{font-size:36px;font-weight:800;letter-spacing:-1px;line-height:1.2;
  margin-bottom:16px}
.section-desc{font-size:16px;color:var(--text-2);line-height:1.7;margin-bottom:40px;
  max-width:560px}

/* ── Hero ── */
.hero{position:relative;z-index:1;width:100%;padding:100px 0 0;
  animation:fadeUp .8s ease both;overflow:hidden}
.hero-content{max-width:1200px;margin:0 auto;padding:0 48px;
  text-align:center;display:flex;flex-direction:column;align-items:center}
.hero-badge{font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:2px;
  color:var(--text-3);border:1px solid var(--border);padding:8px 20px;
  margin-bottom:32px;font-weight:500}
.hero-title{font-size:56px;font-weight:800;line-height:1.1;letter-spacing:-2px;
  margin-bottom:24px}
.hero-sub{font-size:16px;color:var(--text-2);line-height:1.7;max-width:520px;
  margin:0 auto 40px}

/* ── Install box ── */
.install-box{max-width:640px;width:100%;margin:0 auto 80px;
  border:1px solid var(--border);background:var(--surface);overflow:hidden}
.install-bar{display:flex;align-items:center;padding:14px 20px;
  border-bottom:1px solid var(--border);gap:12px;background:var(--surface-alt)}
.install-dots{display:flex;gap:6px}
.install-dots span{width:10px;height:10px;background:var(--border)}
html.dark .install-dots span{background:var(--text-3)}
.install-title{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);
  flex:1;text-align:center}
.install-body{padding:28px 32px;display:flex;align-items:center;gap:16px;
  font-family:'JetBrains Mono',monospace;font-size:14px;color:var(--text-2)}
.install-body .prompt{color:var(--text-3);user-select:none}
.install-body .cmd{flex:1;color:var(--text);user-select:all;cursor:pointer}
.install-copy{background:none;border:1px solid var(--border);color:var(--text-3);
  font-family:'JetBrains Mono',monospace;font-size:11px;padding:6px 14px;
  cursor:pointer;transition:all .15s;letter-spacing:0.5px}
.install-copy:hover{color:var(--text);border-color:var(--text-3)}

/* ── Terminal demo ── */
.term{max-width:760px;width:100%;margin:0 auto;
  border:1px solid var(--border);background:var(--surface);overflow:hidden}
.term-bar{display:flex;align-items:center;padding:14px 20px;
  border-bottom:1px solid var(--border);gap:12px;background:var(--surface-alt)}
.term-dots{display:flex;gap:6px}
.term-dots span{width:10px;height:10px;background:var(--border)}
html.dark .term-dots span{background:var(--text-3)}
.term-title{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);
  flex:1;text-align:center}
.term-body{padding:24px 28px;font-family:'JetBrains Mono',monospace;font-size:12.5px;
  line-height:1.9;white-space:pre;overflow-x:auto;color:var(--text-2)}
.t-prompt{color:var(--text-3);user-select:none}
.t-cmd{color:var(--text);font-weight:500}
.t-flag{color:var(--text-2)}
.t-str{color:var(--text-2);opacity:0.8}
.t-res{color:var(--text-3);font-style:italic}
.t-comment{color:var(--text-3);opacity:0.6}
.t-ok{color:var(--green)}
.t-url{color:var(--text-2);text-decoration:underline;text-underline-offset:2px}
.t-dim{color:var(--text-3)}

/* ── Command grid ── */
.cmd-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden;
  max-width:1200px;margin:0 auto}
.cmd-card{background:var(--bg);padding:36px 32px;transition:background .2s}
.cmd-card:hover{background:var(--surface-alt)}
.cmd-name{font-family:'JetBrains Mono',monospace;font-size:16px;font-weight:700;
  letter-spacing:-0.3px;margin-bottom:8px;color:var(--text)}
.cmd-desc{font-size:14px;color:var(--text-2);line-height:1.6;margin-bottom:16px}
.cmd-example{font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--text-3);
  padding:10px 14px;border:1px solid var(--border);background:var(--surface);
  white-space:pre;overflow-x:auto;line-height:1.6}

/* ── Features list ── */
.features{display:grid;grid-template-columns:repeat(4,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden;
  max-width:1200px;margin:0 auto}
.feature{background:var(--bg);padding:40px 32px;text-align:center;transition:background .2s}
.feature:hover{background:var(--surface-alt)}
.feature-icon{color:var(--text-3);margin-bottom:16px;display:flex;justify-content:center}
.feature-name{font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:600;
  letter-spacing:-0.2px;margin-bottom:8px}
.feature-desc{font-size:13px;color:var(--text-2);line-height:1.6}

/* ── Pipe examples ── */
.pipes{display:grid;grid-template-columns:repeat(2,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden;
  max-width:1200px;margin:0 auto}
.pipe{background:var(--bg);padding:28px 32px}
.pipe-label{font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:600;
  letter-spacing:-0.2px;margin-bottom:12px}
.pipe-code{font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--text-2);
  line-height:1.7;white-space:pre;overflow-x:auto}

/* ── Buttons ── */
.btn{font-size:14px;font-weight:500;padding:12px 28px;border:1px solid var(--border);
  display:inline-flex;align-items:center;gap:8px;transition:all .15s;color:var(--text-2);
  cursor:pointer;text-decoration:none}
.btn:hover{border-color:var(--text-3);color:var(--text)}
.btn--primary{background:var(--ink);border-color:var(--ink);color:var(--bg);font-weight:600}
.btn--primary:hover{opacity:0.85;color:var(--bg)}
.btn--ghost{background:transparent}
.btn--lg{padding:15px 36px;font-size:15px}

/* ── CTA ── */
.section--cta{padding:140px 0 100px}
.section--cta .section-pad{display:flex;flex-direction:column;align-items:center;text-align:center}
.cta-label{font-family:'JetBrains Mono',monospace;font-size:14px;color:var(--text-3);
  margin-bottom:16px}
.cta-caret{color:var(--text);font-weight:700}
.cta-title{font-size:36px;font-weight:800;letter-spacing:-1px;margin-bottom:16px}
.cta-desc{font-size:16px;color:var(--text-2);line-height:1.7;margin-bottom:40px;max-width:480px}
.cta-actions{display:flex;gap:14px;justify-content:center}

/* ── Exit codes table ── */
.exit-table{width:100%;border-collapse:collapse;font-family:'JetBrains Mono',monospace;
  font-size:13px;margin-top:24px;max-width:600px}
.exit-table th{text-align:left;font-size:10px;letter-spacing:1px;text-transform:uppercase;
  color:var(--text-3);padding:10px 16px;border-bottom:2px solid var(--border)}
.exit-table td{padding:10px 16px;border-bottom:1px solid var(--border);color:var(--text-2)}
.exit-table td:first-child{color:var(--text);font-weight:600;width:60px}
.exit-table tr:last-child td{border-bottom:none}

/* ── Responsive ── */
@media(max-width:1100px){
  .cmd-grid{grid-template-columns:repeat(2,1fr)}
  .features{grid-template-columns:repeat(2,1fr)}
}
@media(max-width:768px){
  .hero{padding:64px 0 0}
  .hero-title{font-size:40px;letter-spacing:-1px}
  .section{padding:80px 0}
  .section-heading{font-size:28px;letter-spacing:-0.5px}
  .cmd-grid{grid-template-columns:1fr}
  .pipes{grid-template-columns:1fr}
  .features{grid-template-columns:1fr}
}
@media(max-width:640px){
  .nav-inner{padding:0 20px;height:52px}
  .nav-links{display:none;position:absolute;top:52px;left:0;right:0;
    flex-direction:column;padding:16px 20px;gap:16px;z-index:100;
    border-bottom:1px solid var(--border);
    background:color-mix(in srgb,var(--bg) 95%,transparent);
    backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px)}
  .nav-links.open{display:flex}
  .mobile-toggle{display:block}
  .hero{padding:48px 0 0}
  .hero-content{padding:0 20px}
  .hero-badge{font-size:10px;padding:6px 14px}
  .hero-title{font-size:32px;letter-spacing:-0.5px}
  .hero-sub{font-size:14px;margin-bottom:32px}
  .hero-sub br{display:none}
  .section{padding:56px 0}
  .section-pad{padding:0 20px}
  .install-body{padding:20px;font-size:12px;flex-direction:column;gap:12px}
  .term-body{padding:16px 20px;font-size:11px}
  .cmd-card{padding:24px 20px}
  .pipe{padding:20px}
  .feature{padding:28px 20px}
  .section--cta{padding:64px 0}
  .cta-title{font-size:24px}
  .cta-actions{flex-direction:column;width:100%;max-width:320px}
  .btn--lg,.btn{justify-content:center}
}
</style>
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
      <a href="/developers">developers</a>
      <a href="/api">api</a>
      <a href="/cli" class="active">cli</a>
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

<!-- ═══ HERO ═══ -->
<section class="hero">
  <div class="hero-content">
    <div class="hero-badge">COMMAND LINE INTERFACE</div>
    <h1 class="hero-title">Your files.<br><span class="grad">Your terminal.</span></h1>
    <p class="hero-sub">One file. Zero dependencies. Upload, download, and share from any shell. Pipes, scripts, and cron &mdash; it just works.</p>
  </div>

  <div style="max-width:760px;width:100%;margin:0 auto;padding:0 48px">
    <div class="install-box">
      <div class="install-bar">
        <div class="install-dots"><span></span><span></span><span></span></div>
        <div class="install-title">install</div>
      </div>
      <div class="install-body">
        <span class="prompt">$</span>
        <span class="cmd" id="install-cmd">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
        <button class="install-copy" onclick="copyInstall()">copy</button>
      </div>
    </div>
  </div>
</section>

<!-- ═══ LIVE DEMO ═══ -->
<section class="section" id="demo">
  <div class="section-pad">
    <div class="section-label">30-SECOND DEMO</div>
    <div class="section-heading">Login. Upload. Share. <span class="grad">Done.</span></div>
  </div>
  <div style="max-width:760px;width:100%;margin:0 auto;padding:0 48px">
    <div class="term">
      <div class="term-bar">
        <div class="term-dots"><span></span><span></span><span></span></div>
        <div class="term-title">terminal</div>
      </div>
      <div class="term-body"><span class="t-comment"># authenticate</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage login</span>
<span class="t-dim">email:</span> alice@example.com
  <span class="t-ok">Authenticated</span> Token saved to ~/.config/storage/token

<span class="t-comment"># upload a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">report.pdf docs</span>
  <span class="t-ok">Uploaded</span> report.pdf (512 KB) to docs

<span class="t-comment"># list your files</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage ls</span> <span class="t-flag">docs</span>
<span class="t-dim">PATH                  SIZE       TYPE                MODIFIED</span>
report.pdf          512 KB     application/pdf      just now
readme.md             2 KB     text/markdown        3d ago

<span class="t-comment"># share it</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">docs/report.pdf</span>
<span class="t-url">https://storage.liteio.dev/sign/tok_abc123</span>
  <span class="t-ok">Expires</span> in 1 hour

<span class="t-comment"># download it</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage get</span> <span class="t-flag">docs/report.pdf</span>
  <span class="t-ok">Downloaded</span> report.pdf (512 KB)</div>
    </div>
  </div>
</section>

<!-- ═══ COMMANDS ═══ -->
<section class="section" id="commands">
  <div class="section-pad">
    <div class="section-label">COMMANDS</div>
    <div class="section-heading">Familiar. Obvious. <span class="grad">Complete.</span></div>
    <p class="section-desc">Mirrors the tools you already know &mdash; ls, cp, mv, rm, cat. Every command supports <code>--json</code> for scripting and <code>--quiet</code> for cron.</p>
  </div>
  <div class="cmd-grid">
    <div class="cmd-card">
      <div class="cmd-name">ls</div>
      <div class="cmd-desc">List buckets or objects</div>
      <div class="cmd-example">$ storage ls
$ storage ls docs
$ storage ls docs reports/</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">put</div>
      <div class="cmd-desc">Upload a file or stdin</div>
      <div class="cmd-example">$ storage put photo.jpg pics
$ echo hi | storage put - docs/hi.txt</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">get</div>
      <div class="cmd-desc">Download a file</div>
      <div class="cmd-example">$ storage get docs/report.pdf
$ storage get docs/data.csv -</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">share</div>
      <div class="cmd-desc">Create a signed URL</div>
      <div class="cmd-example">$ storage share docs/report.pdf
$ storage share pics/photo.jpg -x 7d</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">rm</div>
      <div class="cmd-desc">Delete objects</div>
      <div class="cmd-example">$ storage rm docs/draft.md
$ storage rm docs/old/ --recursive</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">cat</div>
      <div class="cmd-desc">Print file to stdout</div>
      <div class="cmd-example">$ storage cat docs/config.json
$ storage cat docs/data.csv | wc -l</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">mv &middot; cp</div>
      <div class="cmd-desc">Move, rename, or copy</div>
      <div class="cmd-example">$ storage mv docs/old.md docs/new.md
$ storage cp docs/tpl.md docs/copy.md</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">search</div>
      <div class="cmd-desc">Find objects by name</div>
      <div class="cmd-example">$ storage search report
$ storage search --type image/png</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">bucket &middot; key &middot; stats</div>
      <div class="cmd-desc">Manage buckets, API keys, usage</div>
      <div class="cmd-example">$ storage bucket create avatars --public
$ storage key create ci --scope object:*
$ storage stats</div>
    </div>
  </div>
</section>

<!-- ═══ UNIX PHILOSOPHY ═══ -->
<section class="section" id="unix">
  <div class="section-pad">
    <div class="section-label">UNIX PHILOSOPHY</div>
    <div class="section-heading">Pipes. Scripts. <span class="grad">Composability.</span></div>
    <p class="section-desc">Reads stdin, writes stdout, errors go to stderr. Works with every tool in your shell.</p>
  </div>
  <div class="pipes">
    <div class="pipe">
      <div class="pipe-label">Upload all PDFs</div>
      <div class="pipe-code"><span class="t-cmd">for</span> f <span class="t-cmd">in</span> *.pdf; <span class="t-cmd">do</span>
  storage put "$f" docs
<span class="t-cmd">done</span></div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Pipe to jq</div>
      <div class="pipe-code">storage cat docs/data.json \\
  | <span class="t-cmd">jq</span> '.items[] | .name'</div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Generate &amp; upload</div>
      <div class="pipe-code">generate-report | storage put - \\
  docs/daily-$(date +%F).txt</div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Share &amp; copy</div>
      <div class="pipe-code">storage share docs/report.pdf \\
  | head -1 | <span class="t-cmd">pbcopy</span></div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Mirror a bucket locally</div>
      <div class="pipe-code">storage ls -j docs | jq -r '.[].path' \\
  | <span class="t-cmd">while</span> read p; <span class="t-cmd">do</span>
    storage get "docs/$p" "./backup/$p"
  <span class="t-cmd">done</span></div>
    </div>
    <div class="pipe">
      <div class="pipe-label">CI/CD deploy</div>
      <div class="pipe-code">tar czf - ./dist | storage put - \\
  releases/v$(cat VERSION).tar.gz</div>
    </div>
  </div>
</section>

<!-- ═══ FEATURES ═══ -->
<section class="section" id="features">
  <div class="section-pad">
    <div class="section-label">DESIGN</div>
    <div class="section-heading">Built for real work.</div>
  </div>
  <div class="features">
    <div class="feature">
      <div class="feature-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="feature-name">Single file</div>
      <div class="feature-desc">One bash script. No runtime, no package manager. Drop it in PATH and go.</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      </div>
      <div class="feature-name">Secure tokens</div>
      <div class="feature-desc">Tokens stored with 0600 permissions. Never logged, never printed. HTTPS only.</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      </div>
      <div class="feature-name">TTY aware</div>
      <div class="feature-desc">Colors and progress for humans. Clean output for pipes. Automatic detection.</div>
    </div>
    <div class="feature">
      <div class="feature-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
      </div>
      <div class="feature-name">--json everywhere</div>
      <div class="feature-desc">Every command supports JSON output for scripting with jq, Python, or any tool.</div>
    </div>
  </div>
</section>

<!-- ═══ EXIT CODES ═══ -->
<section class="section" id="exit-codes">
  <div class="section-pad">
    <div class="section-label">RELIABILITY</div>
    <div class="section-heading">Meaningful exit codes.</div>
    <p class="section-desc">Every failure mode has a documented exit code. Your scripts can react to specific errors, not just success vs. failure.</p>
    <table class="exit-table">
      <thead><tr><th>Code</th><th>Meaning</th></tr></thead>
      <tbody>
        <tr><td>0</td><td>Success</td></tr>
        <tr><td>1</td><td>General error</td></tr>
        <tr><td>2</td><td>Usage error (bad arguments)</td></tr>
        <tr><td>3</td><td>Authentication error</td></tr>
        <tr><td>4</td><td>Not found (bucket or object)</td></tr>
        <tr><td>5</td><td>Conflict (already exists)</td></tr>
        <tr><td>6</td><td>Permission denied</td></tr>
        <tr><td>7</td><td>Network error</td></tr>
      </tbody>
    </table>
  </div>
</section>

<!-- ═══ CTA ═══ -->
<section class="section section--cta">
  <div class="section-pad">
    <div class="cta-label"><span class="cta-caret">&gt;</span> ready?</div>
    <div class="cta-title">Install in one line</div>
    <p class="cta-desc">Works on macOS, Linux, and WSL. Requires only curl.</p>
    <div class="cta-actions">
      <a href="/api" class="btn btn--primary btn--lg">API Reference</a>
      <a href="/developers" class="btn btn--ghost btn--lg">Developer Guide</a>
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
function copyInstall(){
  const cmd=document.getElementById('install-cmd');
  navigator.clipboard.writeText(cmd.textContent).then(()=>{
    const btn=document.querySelector('.install-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}
</script>
</body>
</html>`;
}
