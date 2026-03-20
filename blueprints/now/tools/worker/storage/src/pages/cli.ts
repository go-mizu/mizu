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
<title>CLI - Liteio Storage</title>
<meta name="description" content="The Liteio Storage CLI. Upload, download, and share files from your terminal. Zero dependencies. Works with Node.js, Bun, and Deno.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=DM+Sans:ital,wght@0,400;0,500;0,600;0,700;0,800;0,900;1,400&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#FAFAF9;--surface:#FFF;--surface-alt:#F4F4F5;--surface-2:#E4E4E7;
  --text:#09090B;--text-2:#52525B;--text-3:#A1A1AA;
  --border:#E4E4E7;--ink:#09090B;
  --green:#16A34A;--blue:#2563EB;--amber:#D97706;--red:#DC2626;--purple:#7C3AED;
  --green-dim:#DCFCE7;--blue-dim:#DBEAFE;
}
html.dark{
  --bg:#09090B;--surface:#111113;--surface-alt:#18181B;--surface-2:#27272A;
  --text:#FAFAF9;--text-2:#A1A1AA;--text-3:#52525B;
  --border:#1E1E21;--ink:#FAFAF9;
  --green:#4ADE80;--blue:#60A5FA;--amber:#FBBF24;--red:#F87171;--purple:#C084FC;
  --green-dim:rgba(74,222,128,.08);--blue-dim:rgba(96,165,250,.08);
}
html{scroll-behavior:smooth}
body{font-family:'DM Sans',system-ui,sans-serif;color:var(--text);background:var(--bg);
  -webkit-font-smoothing:antialiased;overflow-x:hidden}
a{color:inherit;text-decoration:none}
code{font-family:'JetBrains Mono',monospace;font-size:.85em;
  background:var(--surface-alt);padding:2px 6px;border:1px solid var(--border)}

/* Grid bg */
.grid-bg{position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:
    linear-gradient(var(--border) 1px,transparent 1px),
    linear-gradient(90deg,var(--border) 1px,transparent 1px);
  background-size:80px 80px;opacity:.25}
html.dark .grid-bg{opacity:.06}

/* Nav */
nav{position:sticky;top:0;z-index:100;width:100%;
  background:color-mix(in srgb,var(--bg) 85%,transparent);
  backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);
  border-bottom:1px solid var(--border)}
.nav-inner{width:100%;padding:0 48px;height:56px;
  display:flex;align-items:center;justify-content:space-between}
.logo{font-weight:700;font-size:15px;letter-spacing:-.3px;
  display:flex;align-items:center;gap:8px}
.logo-dot{width:6px;height:6px;background:var(--text);display:inline-block}
.nav-links{display:flex;gap:32px}
.nav-links a{font-size:13px;color:var(--text-3);transition:color .15s;font-weight:500;letter-spacing:.01em}
.nav-links a:hover,.nav-links a.active{color:var(--text)}
.nav-right{display:flex;align-items:center;gap:16px}
.nav-user{font-size:13px;color:var(--text-2);font-weight:500}
.nav-signout{font-size:12px;color:var(--text-3);border:1px solid var(--border);
  padding:5px 14px;transition:all .15s;font-weight:500}
.nav-signout:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle{background:none;border:none;padding:8px;cursor:pointer;
  color:var(--text-3);display:flex;align-items:center;transition:all .15s}
.theme-toggle:hover{color:var(--text)}
html.dark .icon-sun{display:block}html.dark .icon-moon{display:none}
html:not(.dark) .icon-sun{display:none}html:not(.dark) .icon-moon{display:block}
.mobile-toggle{display:none;background:none;border:none;color:var(--text-3);cursor:pointer;padding:4px}

/* Layout */
.wrap{position:relative;z-index:1;max-width:1120px;margin:0 auto;padding:0 48px}
.label{font-family:'JetBrains Mono',monospace;font-size:10px;letter-spacing:2.5px;
  color:var(--text-3);font-weight:500;text-transform:uppercase;margin-bottom:16px}
.h2{font-size:32px;font-weight:800;letter-spacing:-1px;line-height:1.2;margin-bottom:14px}
.sub{font-size:15px;color:var(--text-2);line-height:1.7;max-width:560px}
.grad{background:linear-gradient(135deg,var(--text) 0%,var(--text-3) 100%);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;font-weight:800}

/* Section */
.section{padding:96px 0;opacity:0;transform:translateY(28px);transition:opacity .7s ease,transform .7s ease}
.section.visible{opacity:1;transform:none}

/* Hero */
.hero{position:relative;z-index:1;padding:80px 0 72px;
  border-bottom:1px solid var(--border)}
.hero-inner{max-width:1120px;margin:0 auto;padding:0 48px}
.hero-eyebrow{display:inline-flex;align-items:center;gap:10px;
  font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:2px;
  color:var(--text-3);border:1px solid var(--border);padding:7px 16px;
  margin-bottom:36px;font-weight:500;text-transform:uppercase}
.hero-eyebrow-dot{width:5px;height:5px;background:var(--green);display:inline-block}
.hero-title{font-size:56px;font-weight:900;letter-spacing:-2.5px;line-height:1.08;
  margin-bottom:20px;max-width:700px}
.hero-title em{font-style:normal;color:var(--text-3)}
.hero-sub{font-size:17px;color:var(--text-2);line-height:1.7;max-width:520px;margin-bottom:48px}

.hero-install{display:flex;gap:12px;align-items:center;flex-wrap:wrap;margin-bottom:56px}
.install-box{display:flex;align-items:center;gap:0;border:1px solid var(--border);
  background:var(--surface);overflow:hidden;max-width:560px}
.install-box .prompt{font-family:'JetBrains Mono',monospace;font-size:13px;
  color:var(--text-3);padding:12px 16px;border-right:1px solid var(--border);
  background:var(--surface-alt);user-select:none;flex-shrink:0}
.install-box .cmd{font-family:'JetBrains Mono',monospace;font-size:13px;
  color:var(--text);padding:12px 18px;flex:1;user-select:all;cursor:pointer;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.copy-btn{font-family:'JetBrains Mono',monospace;font-size:11px;
  background:none;border:none;border-left:1px solid var(--border);
  color:var(--text-3);padding:12px 16px;cursor:pointer;transition:all .15s;
  white-space:nowrap;flex-shrink:0}
.copy-btn:hover{color:var(--text);background:var(--surface-alt)}
.install-tabs{display:flex;gap:0;border:1px solid var(--border);overflow:hidden}
.install-tab{font-family:'JetBrains Mono',monospace;font-size:11px;
  background:none;border:none;border-right:1px solid var(--border);
  color:var(--text-3);padding:8px 16px;cursor:pointer;transition:all .15s;font-weight:500}
.install-tab:last-child{border-right:none}
.install-tab.active{background:var(--ink);color:var(--bg)}
.install-tab:not(.active):hover{color:var(--text);background:var(--surface-alt)}
.hero-stats{display:flex;gap:40px;flex-wrap:wrap}
.hero-stat{display:flex;flex-direction:column;gap:2px}
.hero-stat-val{font-family:'JetBrains Mono',monospace;font-size:22px;font-weight:700;
  letter-spacing:-1px}
.hero-stat-label{font-size:12px;color:var(--text-3)}

/* Terminal */
.term{border:1px solid var(--border);background:var(--surface);overflow:hidden}
.term-bar{display:flex;align-items:center;padding:12px 18px;
  border-bottom:1px solid var(--border);gap:10px;background:var(--surface-alt)}
.term-dots{display:flex;gap:5px}
.term-dots span{width:9px;height:9px;background:var(--border)}
html.dark .term-dots span{background:var(--text-3)}
.term-title{font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--text-3);
  flex:1;text-align:center;letter-spacing:1px}
.term-body{padding:20px 24px;font-family:'JetBrains Mono',monospace;font-size:12.5px;
  line-height:1.9;overflow-x:auto}
.t-prompt{color:var(--text-3);user-select:none}
.t-cmd{color:var(--text);font-weight:600}
.t-flag{color:var(--text-2)}
.t-str{color:var(--text-2);opacity:.75}
.t-comment{color:var(--text-3);font-style:italic}
.t-ok{color:var(--green);font-weight:600}
.t-url{color:var(--blue);text-decoration:underline;text-underline-offset:2px}
.t-dim{color:var(--text-3)}
.t-num{color:var(--amber)}
.t-blank{display:block;height:6px}

/* Steps */
.steps{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.step{background:var(--bg);padding:32px 28px}
.step-num{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);
  font-weight:600;margin-bottom:14px;letter-spacing:1px}
.step-title{font-size:16px;font-weight:700;letter-spacing:-.3px;margin-bottom:8px}
.step-desc{font-size:13px;color:var(--text-2);line-height:1.6;margin-bottom:16px}
.step .term{border:none;border-top:1px solid var(--border)}
.step .term-body{padding:14px 16px;font-size:11.5px;line-height:1.8}

/* Command grid */
.cmd-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.cmd-card{background:var(--bg);padding:24px;transition:background .15s}
.cmd-card:hover{background:var(--surface-alt)}
.cmd-name{font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:700;
  margin-bottom:6px;color:var(--text)}
.cmd-desc{font-size:13px;color:var(--text-2);line-height:1.5}
.cmd-example{font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text-2);padding:8px 12px;background:var(--surface-alt);
  border:1px solid var(--border);white-space:pre;overflow-x:auto;
  line-height:1.7;margin-top:12px}

/* Use case cards */
.uc-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.uc{background:var(--bg);padding:0}
.uc-head{padding:28px 28px 20px;border-bottom:1px solid var(--border)}
.uc-tag{font-family:'JetBrains Mono',monospace;font-size:9px;letter-spacing:2px;
  text-transform:uppercase;color:var(--text-3);font-weight:500;margin-bottom:10px}
.uc-title{font-size:16px;font-weight:700;letter-spacing:-.3px;margin-bottom:6px}
.uc-desc{font-size:13px;color:var(--text-2);line-height:1.6}
.uc-term{padding:16px 20px}
.uc-term .term-body{padding:12px 14px;font-size:11px;line-height:1.8}

/* Auth */
.auth-box{border:1px solid var(--border);overflow:hidden}
.auth-method{padding:28px;border-bottom:1px solid var(--border)}
.auth-method:last-child{border-bottom:none}
.auth-method-title{font-size:15px;font-weight:700;margin-bottom:6px;display:flex;align-items:center;gap:8px}
.auth-method-badge{font-family:'JetBrains Mono',monospace;font-size:9px;padding:2px 8px;
  border:1px solid var(--border);color:var(--text-3);font-weight:500;letter-spacing:.5px}
.auth-method-desc{font-size:13px;color:var(--text-2);line-height:1.6;margin-bottom:12px}
.auth-method-code{font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text-2);padding:10px 14px;background:var(--surface-alt);
  border:1px solid var(--border);white-space:pre;overflow-x:auto;line-height:1.7}

/* Buttons */
.btn{font-size:14px;font-weight:500;padding:11px 24px;border:1px solid var(--border);
  display:inline-flex;align-items:center;gap:8px;transition:all .15s;color:var(--text-2);
  cursor:pointer;text-decoration:none}
.btn:hover{border-color:var(--text-3);color:var(--text)}
.btn--primary{background:var(--ink);border-color:var(--ink);color:var(--bg);font-weight:600}
.btn--primary:hover{opacity:.85;color:var(--bg)}
.btn--lg{padding:14px 32px;font-size:15px}

/* CTA */
.section--cta{padding:120px 0 80px;border-top:1px solid var(--border)}
.section--cta .wrap{display:flex;flex-direction:column;align-items:center;text-align:center}
.cta-title{font-size:40px;font-weight:900;letter-spacing:-1.5px;margin-bottom:14px}
.cta-sub{font-size:16px;color:var(--text-2);line-height:1.7;margin-bottom:40px;max-width:440px}
.cta-actions{display:flex;gap:12px;flex-wrap:wrap;justify-content:center}

/* Divider */
.section-divider{border:none;border-top:1px solid var(--border);margin:0}

/* Responsive */
@media(max-width:1024px){
  .steps{grid-template-columns:1fr}
  .cmd-grid{grid-template-columns:1fr 1fr}
  .uc-grid{grid-template-columns:1fr}
}
@media(max-width:768px){
  .hero-title{font-size:40px;letter-spacing:-1.5px}
  .h2{font-size:28px}
  .cmd-grid{grid-template-columns:1fr}
  .section{padding:64px 0}
}
@media(max-width:640px){
  .nav-inner{padding:0 20px;height:48px}
  .nav-links{display:none;position:absolute;top:48px;left:0;right:0;
    flex-direction:column;padding:12px 20px;gap:16px;z-index:100;
    border-bottom:1px solid var(--border);
    background:color-mix(in srgb,var(--bg) 96%,transparent);
    backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px)}
  .nav-links.open{display:flex}
  .mobile-toggle{display:block}
  .hero-inner,.wrap{padding:0 20px}
  .hero{padding:56px 0 48px}
  .hero-title{font-size:32px;letter-spacing:-1px}
  .hero-sub{font-size:15px}
  .hero-install{flex-direction:column;align-items:stretch}
  .install-box .cmd{font-size:11px}
  .term-body{padding:14px 16px;font-size:11px}
  .section--cta{padding:72px 0 56px}
  .cta-title{font-size:28px}
  .cta-actions{flex-direction:column;width:100%;max-width:300px}
  .btn--lg,.btn{justify-content:center}
  .hero-stats{gap:24px}
  .uc-head,.uc-term{padding-left:16px;padding-right:16px}
  .cmd-card{padding:20px}
  .steps{gap:0}
  .step{padding:24px 20px}
}
</style>
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Liteio Storage</a>
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

<!-- HERO -->
<section class="hero">
  <div class="hero-inner">
    <div class="hero-eyebrow">
      <span class="hero-eyebrow-dot"></span>
      @liteio/storage-cli
    </div>

    <h1 class="hero-title">Files from<br>your <em>terminal.</em></h1>
    <p class="hero-sub">Upload, download, and share files with one command. Available as a single binary or npm package.</p>

    <div class="hero-install">
      <div style="display:flex;flex-direction:column;gap:8px;flex:1;max-width:620px">
        <div class="install-tabs" id="install-tabs">
          <button class="install-tab active" data-cmd="curl -fsSL https://storage.liteio.dev/cli/install.sh | sh" data-label="macOS / Linux">macOS / Linux</button>
          <button class="install-tab" data-cmd="irm https://storage.liteio.dev/cli/install.ps1 | iex" data-label="Windows">Windows</button>
          <button class="install-tab" data-cmd="npx @liteio/storage-cli" data-label="npm">npm</button>
          <button class="install-tab" data-cmd="bunx @liteio/storage-cli" data-label="Bun">Bun</button>
          <button class="install-tab" data-cmd="deno run --allow-all npm:@liteio/storage-cli" data-label="Deno">Deno</button>
        </div>
        <div class="install-box">
          <span class="prompt">$</span>
          <span class="cmd" id="install-cmd">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
          <button class="copy-btn" id="copy-install-btn" onclick="copyInstall()">copy</button>
        </div>
      </div>
    </div>

    <div class="hero-stats">
      <div class="hero-stat">
        <div class="hero-stat-val">0</div>
        <div class="hero-stat-label">dependencies</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">12</div>
        <div class="hero-stat-label">commands</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">3</div>
        <div class="hero-stat-label">runtimes</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">~8 MB</div>
        <div class="hero-stat-label">single binary</div>
      </div>
    </div>
  </div>
</section>

<!-- HOW IT WORKS -->
<section class="section" id="quickstart" style="padding-top:96px">
  <div class="wrap">
    <div class="label">How it works</div>
    <div class="h2">Three steps to <span class="grad">your first upload.</span></div>
    <p class="sub" style="margin-bottom:40px">Log in, upload a file, share it. That's the whole workflow.</p>
  </div>

  <div class="wrap">
    <div class="steps">
      <div class="step">
        <div class="step-num">01</div>
        <div class="step-title">Log in</div>
        <div class="step-desc">Opens your browser. Token saved locally.</div>
        <div class="term">
          <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">storage login</span></div>
        </div>
      </div>
      <div class="step">
        <div class="step-num">02</div>
        <div class="step-title">Upload</div>
        <div class="step-desc">Put any file into storage.</div>
        <div class="term">
          <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">report.pdf docs/</span></div>
        </div>
      </div>
      <div class="step">
        <div class="step-num">03</div>
        <div class="step-title">Share</div>
        <div class="step-desc">Get a link. It expires automatically.</div>
        <div class="term">
          <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">docs/report.pdf</span></div>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- COMMANDS -->
<section class="section" id="commands">
  <div class="wrap">
    <div class="label">Commands</div>
    <div class="h2">Everything the <span class="grad">CLI can do.</span></div>
    <p class="sub" style="margin-bottom:40px">Every command supports <code>--json</code> for scripting and <code>--quiet</code> for silence.</p>
  </div>

  <div class="cmd-grid" style="border-left:none;border-right:none">
    <div class="cmd-card">
      <div class="cmd-name">put</div>
      <div class="cmd-desc">Upload a file or stdin.</div>
      <div class="cmd-example">$ storage put photo.jpg images/</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">get</div>
      <div class="cmd-desc">Download a file.</div>
      <div class="cmd-example">$ storage get images/photo.jpg</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">cat</div>
      <div class="cmd-desc">Print file to stdout.</div>
      <div class="cmd-example">$ storage cat config.json</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">ls</div>
      <div class="cmd-desc">List files at a path.</div>
      <div class="cmd-example">$ storage ls docs/</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">share</div>
      <div class="cmd-desc">Create a temporary link.</div>
      <div class="cmd-example">$ storage share report.pdf</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">rm</div>
      <div class="cmd-desc">Delete one or more files.</div>
      <div class="cmd-example">$ storage rm old-draft.md</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">mv</div>
      <div class="cmd-desc">Move or rename a file.</div>
      <div class="cmd-example">$ storage mv draft.md final.md</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">find</div>
      <div class="cmd-desc">Search files by name.</div>
      <div class="cmd-example">$ storage find "report"</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">stat</div>
      <div class="cmd-desc">Show storage usage.</div>
      <div class="cmd-example">$ storage stat</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">login / logout</div>
      <div class="cmd-desc">Sign in or sign out.</div>
      <div class="cmd-example">$ storage login</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">token</div>
      <div class="cmd-desc">Show or set your token.</div>
      <div class="cmd-example">$ storage token</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">key</div>
      <div class="cmd-desc">Manage API keys.</div>
      <div class="cmd-example">$ storage key create deploy</div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- USE CASES -->
<section class="section" id="use-cases">
  <div class="wrap">
    <div class="label">Use cases</div>
    <div class="h2">Built for <span class="grad">real workflows.</span></div>
    <p class="sub" style="margin-bottom:40px">The CLI fits anywhere you have a terminal.</p>
  </div>

  <div class="wrap">
    <div class="uc-grid">
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">CI / CD</div>
          <div class="uc-title">Deploy from any pipeline</div>
          <div class="uc-desc">Set <code>STORAGE_TOKEN</code> as a secret and upload.</div>
        </div>
        <div class="uc-term">
          <div class="term" style="border:none">
            <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">dist/app.js cdn/</span></div>
          </div>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Backups</div>
          <div class="uc-title">Pipe straight to storage</div>
          <div class="uc-desc">No temp files. Stream from any command.</div>
        </div>
        <div class="uc-term">
          <div class="term" style="border:none">
            <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">pg_dump</span> <span class="t-flag">mydb | storage put - backup.sql</span></div>
          </div>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Scripting</div>
          <div class="uc-title">JSON output on every command</div>
          <div class="uc-desc">Pipe <code>--json</code> output into any tool.</div>
        </div>
        <div class="uc-term">
          <div class="term" style="border:none">
            <div class="term-body"><span class="t-prompt">$</span> <span class="t-cmd">storage ls</span> <span class="t-flag">--json | jq '.'</span></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- AUTH -->
<section class="section" id="auth">
  <div class="wrap">
    <div class="label">Authentication</div>
    <div class="h2">Two ways to <span class="grad">authenticate.</span></div>
    <p class="sub" style="margin-bottom:40px">Browser login for your laptop. API keys for your scripts.</p>

    <div style="display:grid;grid-template-columns:1fr 1fr;gap:24px;max-width:860px">
      <div class="auth-box">
        <div class="auth-method">
          <div class="auth-method-title">Browser login <span class="auth-method-badge">INTERACTIVE</span></div>
          <div class="auth-method-desc">Opens your browser. No password needed.</div>
          <div class="auth-method-code">$ storage login</div>
        </div>
      </div>
      <div class="auth-box">
        <div class="auth-method">
          <div class="auth-method-title">API keys <span class="auth-method-badge">AUTOMATION</span></div>
          <div class="auth-method-desc">For CI and scripts. Set <code>STORAGE_TOKEN</code> env var.</div>
          <div class="auth-method-code">$ storage key create deploy</div>
        </div>
      </div>
    </div>

    <p style="font-size:13px;color:var(--text-3);margin-top:20px;max-width:860px">
      Token resolution: <code>--token</code> flag, then <code>STORAGE_TOKEN</code> env var, then <code>~/.config/storage/token</code> file.
    </p>
  </div>
</section>

<!-- CTA -->
<section class="section section--cta">
  <div class="wrap">
    <div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--text-3);margin-bottom:16px">&gt; ready?</div>
    <div class="cta-title">Try it now.</div>
    <p class="cta-sub">One command to start. Nothing to install permanently.</p>

    <div class="install-box" style="max-width:560px;margin-bottom:32px">
      <span class="prompt">$</span>
      <span class="cmd">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
      <button class="copy-btn" onclick="copyCta()">copy</button>
    </div>

    <div class="cta-actions">
      <a href="/api" class="btn btn--primary btn--lg">API Reference</a>
      <a href="/developers" class="btn btn--lg">Developer Guide</a>
      <a href="/pricing" class="btn btn--lg">Pricing</a>
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

/* Scroll reveal */
(function(){
  const els=document.querySelectorAll('.section');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.04,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();

/* Install tabs */
(function(){
  const tabs=document.querySelectorAll('#install-tabs .install-tab');
  const cmd=document.getElementById('install-cmd');
  if(!tabs.length||!cmd) return;
  tabs.forEach(tab=>{
    tab.addEventListener('click',()=>{
      tabs.forEach(t=>t.classList.remove('active'));
      tab.classList.add('active');
      cmd.textContent=tab.dataset.cmd;
    });
  });
})();

function copyInstall(){
  const cmd=document.getElementById('install-cmd');
  if(!cmd) return;
  navigator.clipboard.writeText(cmd.textContent).then(()=>{
    const btn=document.getElementById('copy-install-btn');
    if(!btn) return;
    btn.textContent='copied!';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}

function copyCta(){
  navigator.clipboard.writeText('curl -fsSL https://storage.liteio.dev/cli/install.sh | sh').then(()=>{
    const btns=document.querySelectorAll('.section--cta .copy-btn');
    btns.forEach(b=>{b.textContent='copied!';setTimeout(()=>{b.textContent='copy'},2000)});
  });
}
</script>
</body>
</html>`;
}
