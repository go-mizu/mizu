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
<meta name="description" content="The storage.now CLI. Upload, download, share, and automate file operations from your terminal. One binary. Zero dependencies.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
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
body{font-family:'Inter',system-ui,sans-serif;color:var(--text);background:var(--bg);
  -webkit-font-smoothing:antialiased;overflow-x:hidden}
a{color:inherit;text-decoration:none}
code{font-family:'JetBrains Mono',monospace;font-size:.85em;
  background:var(--surface-alt);padding:2px 6px;border:1px solid var(--border)}

/* ── Grid bg ── */
.grid-bg{position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:
    linear-gradient(var(--border) 1px,transparent 1px),
    linear-gradient(90deg,var(--border) 1px,transparent 1px);
  background-size:80px 80px;opacity:.25}
html.dark .grid-bg{opacity:.06}

/* ── Nav ── */
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

/* ── Layout primitives ── */
.wrap{position:relative;z-index:1;max-width:1120px;margin:0 auto;padding:0 48px}
.label{font-family:'JetBrains Mono',monospace;font-size:10px;letter-spacing:2.5px;
  color:var(--text-3);font-weight:500;text-transform:uppercase;margin-bottom:16px}
.h2{font-size:32px;font-weight:800;letter-spacing:-1px;line-height:1.2;margin-bottom:14px}
.sub{font-size:15px;color:var(--text-2);line-height:1.7;max-width:560px}
.grad{background:linear-gradient(135deg,var(--text) 0%,var(--text-3) 100%);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;font-weight:800}

/* ── Section ── */
.section{padding:96px 0;opacity:0;transform:translateY(28px);transition:opacity .7s ease,transform .7s ease}
.section.visible{opacity:1;transform:none}
.section--first{padding-top:0}

/* ── Hero ── */
.hero{position:relative;z-index:1;padding:80px 0 72px;
  border-bottom:1px solid var(--border)}
.hero-inner{max-width:1120px;margin:0 auto;padding:0 48px}
.hero-eyebrow{display:inline-flex;align-items:center;gap:10px;
  font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:2px;
  color:var(--text-3);border:1px solid var(--border);padding:7px 16px;
  margin-bottom:36px;font-weight:500;text-transform:uppercase}
.hero-eyebrow-dot{width:5px;height:5px;background:var(--green);display:inline-block}
.hero-title{font-size:60px;font-weight:900;letter-spacing:-2.5px;line-height:1.05;
  margin-bottom:20px;max-width:800px}
.hero-title em{font-style:normal;color:var(--text-3)}
.hero-sub{font-size:17px;color:var(--text-2);line-height:1.7;max-width:540px;margin-bottom:48px}
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

/* ── Terminal ── */
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
.t-err{color:var(--red)}
.t-url{color:var(--blue);text-decoration:underline;text-underline-offset:2px}
.t-dim{color:var(--text-3)}
.t-kv{color:var(--text-2)}
.t-num{color:var(--amber)}
.t-blank{display:block;height:6px}

/* ── Two-col layout ── */
.two-col{display:grid;grid-template-columns:1fr 1fr;gap:64px;align-items:start}
.two-col--wide{grid-template-columns:3fr 2fr}
.two-col--flipped{direction:rtl}.two-col--flipped>*{direction:ltr}

/* ── Use case cards ── */
.use-cases{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.uc{background:var(--bg);padding:0}
.uc-head{padding:28px 28px 20px;border-bottom:1px solid var(--border)}
.uc-tag{font-family:'JetBrains Mono',monospace;font-size:9px;letter-spacing:2px;
  text-transform:uppercase;color:var(--text-3);font-weight:500;margin-bottom:10px}
.uc-title{font-size:16px;font-weight:700;letter-spacing:-.3px;margin-bottom:6px}
.uc-desc{font-size:13px;color:var(--text-2);line-height:1.6}
.uc-term{padding:20px 28px}
.uc-term .term-body{padding:16px 18px;font-size:11.5px;line-height:1.8}

/* ── Command reference ── */
.cmd-section{display:grid;grid-template-columns:1fr 1fr 1fr;gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.cmd-card{background:var(--bg);padding:28px 24px;transition:background .15s}
.cmd-card:hover{background:var(--surface-alt)}
.cmd-name{font-family:'JetBrains Mono',monospace;font-size:15px;font-weight:700;
  margin-bottom:6px;color:var(--text);display:flex;align-items:center;gap:8px}
.cmd-badge{font-size:9px;padding:2px 7px;border:1px solid var(--border);
  color:var(--text-3);font-weight:500;letter-spacing:.5px;vertical-align:middle}
.cmd-desc{font-size:13px;color:var(--text-2);line-height:1.6;margin-bottom:14px}
.cmd-flags{display:flex;flex-direction:column;gap:4px}
.cmd-flag{display:flex;align-items:baseline;gap:8px;font-size:11.5px}
.cmd-flag-name{font-family:'JetBrains Mono',monospace;color:var(--text);font-weight:500;flex-shrink:0}
.cmd-flag-desc{color:var(--text-3)}
.cmd-example{font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text-2);padding:10px 14px;background:var(--surface-alt);
  border:1px solid var(--border);white-space:pre;overflow-x:auto;
  line-height:1.7;margin-top:14px}

/* ── Feature list ── */
.features{display:grid;grid-template-columns:repeat(4,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.feat{background:var(--bg);padding:36px 28px;transition:background .15s}
.feat:hover{background:var(--surface-alt)}
.feat-icon{color:var(--text-3);margin-bottom:16px}
.feat-icon svg{width:24px;height:24px}
.feat-name{font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:600;
  letter-spacing:-.2px;margin-bottom:8px}
.feat-desc{font-size:13px;color:var(--text-2);line-height:1.6}

/* ── Pipes ── */
.pipes{display:grid;grid-template-columns:repeat(2,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);overflow:hidden}
.pipe{background:var(--bg);padding:24px 28px}
.pipe-label{font-family:'JetBrains Mono',monospace;font-size:12px;font-weight:600;
  letter-spacing:-.2px;margin-bottom:10px;color:var(--text)}
.pipe-code{font-family:'JetBrains Mono',monospace;font-size:11.5px;
  color:var(--text-2);line-height:1.75;white-space:pre;overflow-x:auto}

/* ── Auth explainer ── */
.auth-steps{display:flex;flex-direction:column;gap:0;
  border:1px solid var(--border);overflow:hidden}
.auth-step{display:flex;gap:20px;padding:24px 28px;
  border-bottom:1px solid var(--border)}
.auth-step:last-child{border-bottom:none}
.auth-step-num{font-family:'JetBrains Mono',monospace;font-size:12px;
  color:var(--text-3);font-weight:600;flex-shrink:0;padding-top:2px;
  width:28px}
.auth-step-body{flex:1}
.auth-step-title{font-size:14px;font-weight:600;margin-bottom:4px}
.auth-step-desc{font-size:13px;color:var(--text-2);line-height:1.6}
.auth-step-code{font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text-3);margin-top:8px;padding:8px 12px;
  background:var(--surface-alt);border:1px solid var(--border)}

/* ── Exit codes ── */
.exit-table{width:100%;border-collapse:collapse;
  font-family:'JetBrains Mono',monospace;font-size:12px;
  border:1px solid var(--border);overflow:hidden}
.exit-table th{text-align:left;font-size:9px;letter-spacing:2px;text-transform:uppercase;
  color:var(--text-3);padding:10px 16px;border-bottom:2px solid var(--border);
  background:var(--surface-alt)}
.exit-table td{padding:10px 16px;border-bottom:1px solid var(--border);color:var(--text-2)}
.exit-table td:first-child{color:var(--amber);font-weight:700;width:56px}
.exit-table td:nth-child(2){color:var(--text);font-weight:500;width:180px}
.exit-table tr:last-child td{border-bottom:none}

/* ── Buttons ── */
.btn{font-size:14px;font-weight:500;padding:11px 24px;border:1px solid var(--border);
  display:inline-flex;align-items:center;gap:8px;transition:all .15s;color:var(--text-2);
  cursor:pointer;text-decoration:none}
.btn:hover{border-color:var(--text-3);color:var(--text)}
.btn--primary{background:var(--ink);border-color:var(--ink);color:var(--bg);font-weight:600}
.btn--primary:hover{opacity:.85;color:var(--bg)}
.btn--lg{padding:14px 32px;font-size:15px}

/* ── CTA section ── */
.section--cta{padding:120px 0 80px;border-top:1px solid var(--border)}
.section--cta .wrap{display:flex;flex-direction:column;align-items:center;text-align:center}
.cta-title{font-size:40px;font-weight:900;letter-spacing:-1.5px;margin-bottom:14px}
.cta-sub{font-size:16px;color:var(--text-2);line-height:1.7;margin-bottom:40px;max-width:440px}
.cta-actions{display:flex;gap:12px;flex-wrap:wrap;justify-content:center}

/* ── Divider ── */
.section-divider{border:none;border-top:1px solid var(--border);margin:0}

/* ── Responsive ── */
@media(max-width:1024px){
  .two-col{grid-template-columns:1fr;gap:40px}
  .two-col--wide{grid-template-columns:1fr}
  .two-col--flipped{direction:ltr}
  .use-cases{grid-template-columns:1fr}
  .cmd-section{grid-template-columns:1fr 1fr}
  .features{grid-template-columns:repeat(2,1fr)}
}
@media(max-width:768px){
  .hero-title{font-size:44px;letter-spacing:-1.5px}
  .h2{font-size:28px}
  .cmd-section{grid-template-columns:1fr}
  .pipes{grid-template-columns:1fr}
  .features{grid-template-columns:1fr}
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
  .hero-title{font-size:36px;letter-spacing:-1px}
  .hero-sub{font-size:15px}
  .hero-install{flex-direction:column;align-items:stretch}
  .install-box .cmd{font-size:11px}
  .term-body{padding:14px 16px;font-size:11px}
  .section--cta{padding:72px 0 56px}
  .cta-title{font-size:28px}
  .cta-actions{flex-direction:column;width:100%;max-width:300px}
  .btn--lg,.btn{justify-content:center}
  .hero-stats{gap:24px}
  .two-col{gap:32px}
  .uc-head,.uc-term{padding-left:20px;padding-right:20px}
  .cmd-card{padding:20px}
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

<!-- ══════════════════════════════════════════════════════════
     HERO
     ══════════════════════════════════════════════════════════ -->
<section class="hero">
  <div class="hero-inner">
    <div class="hero-eyebrow">
      <span class="hero-eyebrow-dot"></span>
      storage.now cli
    </div>

    <h1 class="hero-title">Your terminal.<br><em>Already knows</em><br>what to do.</h1>
    <p class="hero-sub">One binary. Upload, download, share, and automate file operations from any shell. Works anywhere curl works &mdash; locally, in CI, in cron, in Docker.</p>

    <div class="hero-install">
      <div style="display:flex;flex-direction:column;gap:8px;flex:1;max-width:580px">
        <div class="install-tabs" id="install-tabs">
          <button class="install-tab active" data-cmd="curl -fsSL https://storage.liteio.dev/cli/install.sh | sh" data-label="macOS / Linux">macOS / Linux</button>
          <button class="install-tab" data-cmd="iwr https://storage.liteio.dev/cli/install.ps1 | iex" data-label="Windows">Windows</button>
          <button class="install-tab" data-cmd="brew install storage-now" data-label="Homebrew">Homebrew</button>
          <button class="install-tab" data-cmd="npm install -g @storage-now/cli" data-label="npm">npm</button>
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
        <div class="hero-stat-val">~2 MB</div>
        <div class="hero-stat-label">single binary</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">0</div>
        <div class="hero-stat-label">dependencies</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">9</div>
        <div class="hero-stat-label">core commands</div>
      </div>
      <div class="hero-stat">
        <div class="hero-stat-val">macOS / Linux / Win</div>
        <div class="hero-stat-label">platforms</div>
      </div>
    </div>
  </div>
</section>

<!-- ══════════════════════════════════════════════════════════
     QUICKSTART
     ══════════════════════════════════════════════════════════ -->
<section class="section section--first" id="quickstart" style="padding-top:96px">
  <div class="wrap">
    <div class="label">Quickstart</div>
    <div class="h2">From zero to sharing<br>in <span class="grad">four commands.</span></div>
    <p class="sub" style="margin-bottom:40px">Install once. Log in once. Then <code>put</code>, <code>get</code>, <code>share</code> &mdash; that's the entire mental model.</p>

    <div class="term" style="max-width:760px">
      <div class="term-bar">
        <div class="term-dots"><span></span><span></span><span></span></div>
        <div class="term-title">terminal</div>
      </div>
      <div class="term-body"><span class="t-comment"># 1. Install</span>
<span class="t-prompt">$</span> <span class="t-cmd">curl -fsSL</span> <span class="t-str">https://storage.liteio.dev/cli/install.sh</span> <span class="t-flag">| sh</span>
  <span class="t-ok">Installed</span> <span class="t-dim">storage v1.0.0 → /usr/local/bin/storage</span>
<span class="t-blank"></span>
<span class="t-comment"># 2. Log in — opens browser for OAuth, saves token locally</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage login</span>
  <span class="t-dim">Opening https://storage.liteio.dev/auth/cli...</span>
  <span class="t-ok">Authenticated</span> <span class="t-dim">as alice@example.com → ~/.config/storage/token</span>
<span class="t-blank"></span>
<span class="t-comment"># 3. Upload a file</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">quarterly-report.pdf</span> <span class="t-flag">docs</span>
  <span class="t-ok">Uploaded</span> <span class="t-dim">quarterly-report.pdf (1.2 MB) → docs/quarterly-report.pdf</span>
<span class="t-blank"></span>
<span class="t-comment"># 4. Share it — creates a signed URL, copies to clipboard</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">docs/quarterly-report.pdf</span> <span class="t-flag">| pbcopy</span>
  <span class="t-ok">Shared</span> <span class="t-url">https://storage.liteio.dev/sign/tok_h8Kf2m</span>
  <span class="t-dim">Expires in 1 hour &middot; link copied to clipboard</span></div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ══════════════════════════════════════════════════════════
     USE CASES
     ══════════════════════════════════════════════════════════ -->
<section class="section" id="use-cases">
  <div class="wrap">
    <div class="label">Use Cases</div>
    <div class="h2">Where it actually<br><span class="grad">gets used.</span></div>
    <p class="sub" style="margin-bottom:48px">Real workflows, not toy examples. Storage CLI fits cleanly into the places you already work.</p>
  </div>

  <!-- CI/CD -->
  <div style="border-top:1px solid var(--border);border-bottom:1px solid var(--border)">
    <div class="wrap" style="padding-top:64px;padding-bottom:64px">
      <div class="two-col two-col--wide">
        <div>
          <div class="label">CI/CD &amp; Deployment</div>
          <h3 style="font-size:22px;font-weight:800;letter-spacing:-.5px;margin-bottom:12px">Publish build artifacts.<br>Distribute releases.</h3>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:20px">
            Drop <code>storage put</code> into any CI pipeline. Upload compiled binaries, static sites, Docker layer tarballs, or test reports. Set <code>STORAGE_TOKEN</code> as a secret, use <code>--quiet</code> and <code>--json</code> for clean machine output.
          </p>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7">
            Teammates and downstream jobs can pull the artifact with <code>storage get</code> without needing cloud credentials &mdash; just a shared token or a signed URL.
          </p>
        </div>
        <div class="term">
          <div class="term-bar">
            <div class="term-dots"><span></span><span></span><span></span></div>
            <div class="term-title">.github/workflows/release.yml</div>
          </div>
          <div class="term-body"><span class="t-comment"># In your CI environment:</span>
<span class="t-comment"># STORAGE_TOKEN is set as a repo secret</span>
<span class="t-blank"></span>
<span class="t-comment"># Build and upload the binary</span>
<span class="t-prompt">$</span> <span class="t-cmd">go build</span> <span class="t-flag">-o dist/myapp-linux-amd64 ./cmd</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">dist/myapp-linux-amd64</span> \
    <span class="t-flag">releases/v$(cat VERSION)</span> <span class="t-flag">--quiet</span>
  <span class="t-ok">Uploaded</span> <span class="t-dim">myapp-linux-amd64 (12.4 MB)</span>
<span class="t-blank"></span>
<span class="t-comment"># Create a signed URL for the release page</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage share</span> \
    <span class="t-flag">releases/v1.4.2/myapp-linux-amd64</span> \
    <span class="t-flag">--expires 30d</span> <span class="t-flag">--json</span> \
  <span class="t-flag">| jq -r</span> <span class="t-str">'.url'</span> <span class="t-flag">&gt; release-url.txt</span>
<span class="t-blank"></span>
<span class="t-comment"># Upload test report</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">coverage.html</span> \
    <span class="t-flag">ci-reports/$(git rev-parse --short HEAD)</span></div>
        </div>
      </div>
    </div>
  </div>

  <!-- Backup -->
  <div style="border-bottom:1px solid var(--border);background:var(--surface-alt)">
    <div class="wrap" style="padding-top:64px;padding-bottom:64px">
      <div class="two-col two-col--wide two-col--flipped">
        <div>
          <div class="label">Backup &amp; Archival</div>
          <h3 style="font-size:22px;font-weight:800;letter-spacing:-.5px;margin-bottom:12px">Automated backups<br>that actually run.</h3>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:20px">
            Cron + storage CLI = offsite backup without an AWS account. The <code>--quiet</code> flag suppresses output when there's nothing wrong, so your inbox stays clean. Exit codes let cron or systemd know when something failed.
          </p>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7">
            Pipe <code>pg_dump</code>, <code>tar</code>, or <code>sqlite3 .dump</code> directly into <code>storage put -</code> &mdash; no temp files needed.
          </p>
        </div>
        <div class="term">
          <div class="term-bar">
            <div class="term-dots"><span></span><span></span><span></span></div>
            <div class="term-title">backup.sh (runs nightly via cron)</div>
          </div>
          <div class="term-body"><span class="t-comment">#!/bin/sh</span>
<span class="t-comment"># Pipe postgres dump straight to storage</span>
<span class="t-comment"># No temp file, no disk space needed</span>
<span class="t-prompt">$</span> <span class="t-cmd">pg_dump</span> <span class="t-flag">mydb</span> \
  <span class="t-flag">| gzip</span> \
  <span class="t-flag">| storage put -</span> \
    <span class="t-str">backups/db/$(date +%F).sql.gz</span> \
    <span class="t-flag">--quiet</span>
<span class="t-blank"></span>
<span class="t-comment"># Upload config files with timestamp</span>
<span class="t-prompt">$</span> <span class="t-cmd">tar czf</span> <span class="t-flag">-</span> <span class="t-str">/etc/nginx /etc/ssl</span> \
  <span class="t-flag">| storage put -</span> \
    <span class="t-str">backups/config/$(date +%F).tar.gz</span>
<span class="t-blank"></span>
<span class="t-comment"># Prune backups older than 30 days</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage ls</span> <span class="t-flag">--json backups/db</span> \
  <span class="t-flag">| jq -r</span> <span class="t-str">'.[].path'</span> \
  <span class="t-flag">| head -n -30</span> \
  <span class="t-flag">| xargs -I{} storage rm {}</span></div>
        </div>
      </div>
    </div>
  </div>

  <!-- Scripting & Automation -->
  <div style="border-bottom:1px solid var(--border)">
    <div class="wrap" style="padding-top:64px;padding-bottom:64px">
      <div class="two-col two-col--wide">
        <div>
          <div class="label">Scripting &amp; Automation</div>
          <h3 style="font-size:22px;font-weight:800;letter-spacing:-.5px;margin-bottom:12px">Generate. Upload.<br>Share. Repeat.</h3>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:20px">
            Every command supports <code>--json</code> for machine-readable output and <code>--quiet</code> for silence. Combine with <code>jq</code>, <code>awk</code>, or any language that can call a subprocess.
          </p>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7">
            Use API keys (not your login token) for automation. Scope them to specific buckets and permissions. Rotate them without re-authenticating.
          </p>
        </div>
        <div class="term">
          <div class="term-bar">
            <div class="term-dots"><span></span><span></span><span></span></div>
            <div class="term-title">terminal</div>
          </div>
          <div class="term-body"><span class="t-comment"># Generate weekly report and share it</span>
<span class="t-prompt">$</span> <span class="t-cmd">generate-report</span> <span class="t-flag">--week 12</span> \
  <span class="t-flag">| storage put -</span> \
    <span class="t-str">reports/week-12.pdf</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">reports/week-12.pdf</span> \
    <span class="t-flag">--expires 7d</span> <span class="t-flag">--json</span> \
  <span class="t-flag">| jq -r</span> <span class="t-str">'.url'</span> \
  <span class="t-flag">| mail -s</span> <span class="t-str">"Week 12 report"</span> <span class="t-str">team@co.com</span>
<span class="t-blank"></span>
<span class="t-comment"># Bulk upload a directory</span>
<span class="t-prompt">$</span> <span class="t-cmd">find</span> <span class="t-flag">./assets -name '*.png'</span> \
  <span class="t-flag">| while read f; do</span>
    <span class="t-cmd">storage put</span> <span class="t-str">"$f"</span> <span class="t-str">cdn/images</span>
  <span class="t-flag">done</span>
<span class="t-blank"></span>
<span class="t-comment"># List files as JSON, process with jq</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage ls</span> <span class="t-flag">--json reports</span> \
  <span class="t-flag">| jq</span> <span class="t-str">'[.[] | {name,size,age:.modified}]'</span></div>
        </div>
      </div>
    </div>
  </div>

  <!-- Data science / notebooks -->
  <div style="border-bottom:1px solid var(--border);background:var(--surface-alt)">
    <div class="wrap" style="padding-top:64px;padding-bottom:64px">
      <div class="two-col two-col--wide two-col--flipped">
        <div>
          <div class="label">Data &amp; ML workflows</div>
          <h3 style="font-size:22px;font-weight:800;letter-spacing:-.5px;margin-bottom:12px">Datasets, checkpoints,<br>and model artifacts.</h3>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:20px">
            Store training datasets and model checkpoints with a consistent path structure. Share versioned artifacts with your team without AWS credentials. Large files upload efficiently via TUS resumable uploads under the hood.
          </p>
          <p style="font-size:14px;color:var(--text-2);line-height:1.7">
            Pull datasets into any environment &mdash; laptop, GPU server, Colab &mdash; with a single <code>storage get</code>. No boto3, no AWSCLI, no IAM role gymnastics.
          </p>
        </div>
        <div class="term">
          <div class="term-bar">
            <div class="term-dots"><span></span><span></span><span></span></div>
            <div class="term-title">terminal</div>
          </div>
          <div class="term-body"><span class="t-comment"># Upload a training dataset (large file)</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">imagenet-subset.tar</span> \
    <span class="t-str">datasets/imagenet-v2</span>
  <span class="t-dim">Uploading imagenet-subset.tar (4.7 GB)</span>
  <span class="t-dim">[████████████░░░░] 64% 3.0 GB/s</span>
<span class="t-blank"></span>
<span class="t-comment"># Save a checkpoint after each epoch</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">checkpoint_ep42.pt</span> \
    <span class="t-str">runs/exp-12/checkpoints</span> <span class="t-flag">--quiet</span>
<span class="t-blank"></span>
<span class="t-comment"># Pull dataset on a new machine</span>
<span class="t-prompt">$</span> <span class="t-cmd">storage get</span> \
    <span class="t-str">datasets/imagenet-v2/imagenet-subset.tar</span> \
    <span class="t-flag">.</span>
  <span class="t-ok">Downloaded</span> <span class="t-dim">imagenet-subset.tar (4.7 GB)</span></div>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ══════════════════════════════════════════════════════════
     COMMAND REFERENCE
     ══════════════════════════════════════════════════════════ -->
<section class="section" id="commands">
  <div class="wrap">
    <div class="label">Reference</div>
    <div class="h2">Every command.<br><span class="grad">Nothing hidden.</span></div>
    <p class="sub" style="margin-bottom:48px">Named after the UNIX tools you already know. Every command accepts <code>--json</code> and <code>--quiet</code>.</p>
  </div>

  <div class="cmd-section" style="border-left:none;border-right:none">
    <div class="cmd-card">
      <div class="cmd-name">login<span class="cmd-badge">AUTH</span></div>
      <div class="cmd-desc">Authenticate via browser OAuth. Opens your browser, exchanges the code, and saves the token to <code>~/.config/storage/token</code> with <code>0600</code> permissions.</div>
      <div class="cmd-example">$ storage login
$ storage login --token sk_live_...  # non-interactive</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">put<span class="cmd-badge">UPLOAD</span></div>
      <div class="cmd-desc">Upload a file or stdin to a bucket. Pass <code>-</code> as the source to read from stdin. Progress bar shown automatically when connected to a TTY.</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--quiet</span><span class="cmd-flag-desc">suppress output</span></div>
        <div class="cmd-flag"><span class="cmd-flag-name">--public</span><span class="cmd-flag-desc">make publicly readable</span></div>
      </div>
      <div class="cmd-example">$ storage put report.pdf docs
$ echo hello | storage put - docs/hi.txt
$ storage put *.log logs --quiet</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">get<span class="cmd-badge">DOWNLOAD</span></div>
      <div class="cmd-desc">Download a file to disk or stdout. Pass <code>-</code> as the destination to stream to stdout, useful for piping into other commands.</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--force</span><span class="cmd-flag-desc">overwrite existing file</span></div>
      </div>
      <div class="cmd-example">$ storage get docs/report.pdf
$ storage get docs/data.csv - | wc -l
$ storage get docs/report.pdf ~/Desktop/</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">ls<span class="cmd-badge">LIST</span></div>
      <div class="cmd-desc">List buckets (no args) or objects in a bucket. Tabular output by default, JSON with <code>--json</code> for scripting.</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--json</span><span class="cmd-flag-desc">machine-readable output</span></div>
        <div class="cmd-flag"><span class="cmd-flag-name">--recursive</span><span class="cmd-flag-desc">include subdirectories</span></div>
      </div>
      <div class="cmd-example">$ storage ls
$ storage ls docs
$ storage ls docs/reports --json | jq length</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">share<span class="cmd-badge">SHARE</span></div>
      <div class="cmd-desc">Create a signed URL for a file. The URL is valid for the specified duration and requires no authentication to access. Print or pipe the URL to clipboard.</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--expires</span><span class="cmd-flag-desc">1h, 7d, 30d (default 1h)</span></div>
        <div class="cmd-flag"><span class="cmd-flag-name">--json</span><span class="cmd-flag-desc">output <code>{url, expires}</code></span></div>
      </div>
      <div class="cmd-example">$ storage share docs/report.pdf
$ storage share pics/photo.jpg --expires 7d
$ storage share f.pdf --json | jq -r .url | pbcopy</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">rm<span class="cmd-badge">DELETE</span></div>
      <div class="cmd-desc">Delete one or more objects. Add <code>--recursive</code> to delete an entire prefix. Exits with code 4 if the object does not exist (useful for idempotent scripts).</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--recursive</span><span class="cmd-flag-desc">delete prefix and all contents</span></div>
        <div class="cmd-flag"><span class="cmd-flag-name">--force</span><span class="cmd-flag-desc">no error if not found</span></div>
      </div>
      <div class="cmd-example">$ storage rm docs/draft.md
$ storage rm logs/ --recursive
$ storage rm tmp/ --recursive --force</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">cat<span class="cmd-badge">STREAM</span></div>
      <div class="cmd-desc">Print a file's contents directly to stdout. Identical to <code>storage get &lt;path&gt; -</code> but with a familiar name. Ideal for config files and small text content.</div>
      <div class="cmd-example">$ storage cat docs/config.json | jq .
$ storage cat scripts/deploy.sh | bash
$ storage cat data.csv | awk -F, '{print $2}'</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">mv &nbsp;·&nbsp; cp<span class="cmd-badge">ORGANIZE</span></div>
      <div class="cmd-desc">Move or rename (<code>mv</code>) and duplicate (<code>cp</code>) objects server-side. No data transfer &mdash; instant even for large files. Works across buckets.</div>
      <div class="cmd-example">$ storage mv docs/old.md docs/new.md
$ storage mv drafts/post.md published/
$ storage cp template.md new-project.md</div>
    </div>
    <div class="cmd-card">
      <div class="cmd-name">search<span class="cmd-badge">FIND</span></div>
      <div class="cmd-desc">Search objects by name across all buckets. Supports glob patterns. Use <code>--json</code> to pipe results into other tools.</div>
      <div class="cmd-flags">
        <div class="cmd-flag"><span class="cmd-flag-name">--bucket</span><span class="cmd-flag-desc">limit to one bucket</span></div>
        <div class="cmd-flag"><span class="cmd-flag-name">--type</span><span class="cmd-flag-desc">filter by MIME type</span></div>
      </div>
      <div class="cmd-example">$ storage search quarterly-report
$ storage search "*.pdf" --bucket docs
$ storage search "" --type image/png --json</div>
    </div>
  </div>

  <div style="margin-top:1px;border-top:1px solid var(--border);border-bottom:1px solid var(--border)">
    <div class="wrap" style="padding-top:40px;padding-bottom:40px">
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:48px">
        <div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:1px;color:var(--text-3);margin-bottom:14px;text-transform:uppercase">Bucket management</div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-2);line-height:2">storage bucket create avatars
storage bucket create logs --public
storage bucket ls
storage bucket rm old-bucket</div>
        </div>
        <div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:1px;color:var(--text-3);margin-bottom:14px;text-transform:uppercase">API key management</div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-2);line-height:2">storage key create ci --scope object:write
storage key create readonly --scope object:read
storage key ls
storage key rm key_abc123</div>
        </div>
        <div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:1px;color:var(--text-3);margin-bottom:14px;text-transform:uppercase">Usage &amp; info</div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-2);line-height:2">storage stats
storage whoami
storage version
storage help &lt;command&gt;</div>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ══════════════════════════════════════════════════════════
     UNIX COMPOSABILITY
     ══════════════════════════════════════════════════════════ -->
<section class="section" id="composability">
  <div class="wrap">
    <div class="label">Unix Philosophy</div>
    <div class="h2">Reads stdin. Writes stdout.<br><span class="grad">Composes cleanly.</span></div>
    <p class="sub" style="margin-bottom:48px">Storage CLI is a well-behaved unix program. Errors go to stderr. Exit codes are meaningful. Colors are suppressed when stdout is not a TTY.</p>
  </div>

  <div class="pipes" style="border-left:none;border-right:none">
    <div class="pipe">
      <div class="pipe-label">Pipe postgres backup directly to storage</div>
      <div class="pipe-code"><span class="t-cmd">pg_dump</span> mydb | <span class="t-cmd">gzip</span> | <span class="t-cmd">storage put</span> - backups/$(date +%F).sql.gz</div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Share and copy link to clipboard</div>
      <div class="pipe-code"><span class="t-cmd">storage share</span> docs/report.pdf --json \
  | <span class="t-cmd">jq</span> -r '.url' | <span class="t-cmd">pbcopy</span></div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Sync a local directory to a bucket</div>
      <div class="pipe-code"><span class="t-cmd">find</span> ./dist -type f | <span class="t-cmd">while</span> read f; <span class="t-cmd">do</span>
  <span class="t-cmd">storage put</span> "$f" cdn/"\${f#./dist/}"
<span class="t-cmd">done</span></div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Download all PDFs and count pages</div>
      <div class="pipe-code"><span class="t-cmd">storage ls</span> --json docs \
  | <span class="t-cmd">jq</span> -r '.[] | select(.name | endswith(".pdf")) | .path' \
  | <span class="t-cmd">xargs</span> -I{} storage get {} .</div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Upload command output as a file</div>
      <div class="pipe-code"><span class="t-cmd">kubectl</span> logs deploy/api --since=1h \
  | <span class="t-cmd">storage put</span> - logs/api-$(date +%s).log</div>
    </div>
    <div class="pipe">
      <div class="pipe-label">Deploy static site on every push</div>
      <div class="pipe-code"><span class="t-cmd">npm run build</span> &amp;&amp; \
<span class="t-cmd">storage rm</span> cdn/ --recursive --force &amp;&amp; \
<span class="t-cmd">find</span> dist -type f | <span class="t-cmd">while</span> read f; <span class="t-cmd">do</span>
  <span class="t-cmd">storage put</span> "$f" cdn/"\${f#dist/}" --quiet
<span class="t-cmd">done</span></div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ══════════════════════════════════════════════════════════
     AUTHENTICATION
     ══════════════════════════════════════════════════════════ -->
<section class="section" id="auth">
  <div class="wrap">
    <div class="two-col">
      <div>
        <div class="label">Authentication</div>
        <div class="h2">Tokens that<br><span class="grad">know their place.</span></div>
        <p class="sub" style="margin-bottom:32px">Two credential types for two contexts. Your personal session token for interactive use; scoped API keys for automation.</p>

        <div class="auth-steps">
          <div class="auth-step">
            <div class="auth-step-num">01</div>
            <div class="auth-step-body">
              <div class="auth-step-title">Interactive login (OAuth PKCE)</div>
              <div class="auth-step-desc">Run <code>storage login</code>. A browser window opens, you approve, and the CLI receives a token via localhost callback. Token saved to <code>~/.config/storage/token</code> with mode <code>600</code>. No password involved.</div>
              <div class="auth-step-code">$ storage login
Opening https://storage.liteio.dev/auth/cli...
Authenticated as alice@example.com</div>
            </div>
          </div>
          <div class="auth-step">
            <div class="auth-step-num">02</div>
            <div class="auth-step-body">
              <div class="auth-step-title">API keys for CI &amp; scripts</div>
              <div class="auth-step-desc">Create a scoped key with <code>storage key create</code>. Use the <code>STORAGE_TOKEN</code> env var or <code>--token</code> flag. Keys can be scoped to specific operations and rotated without re-authenticating.</div>
              <div class="auth-step-code">$ storage key create ci-deploy --scope object:write
sk_live_abc123...  ← save this, shown once

# Use in CI:
STORAGE_TOKEN=sk_live_abc123 storage put ...</div>
            </div>
          </div>
          <div class="auth-step">
            <div class="auth-step-num">03</div>
            <div class="auth-step-body">
              <div class="auth-step-title">Token resolution order</div>
              <div class="auth-step-desc">The CLI checks credentials in this order: <code>--token</code> flag → <code>STORAGE_TOKEN</code> env var → <code>~/.config/storage/token</code> file. The first one found is used.</div>
            </div>
          </div>
        </div>
      </div>

      <div>
        <div class="label" style="margin-bottom:24px">Exit codes</div>
        <p style="font-size:13px;color:var(--text-2);line-height:1.7;margin-bottom:20px">Every failure has a unique code. Scripts can react to specific errors, not just success vs. failure. All codes are stable across versions.</p>
        <table class="exit-table">
          <thead>
            <tr><th>Code</th><th>Name</th><th>When</th></tr>
          </thead>
          <tbody>
            <tr><td>0</td><td>Success</td><td>Operation completed normally</td></tr>
            <tr><td>1</td><td>Error</td><td>Unspecified runtime error</td></tr>
            <tr><td>2</td><td>Usage</td><td>Bad arguments or flags</td></tr>
            <tr><td>3</td><td>Auth</td><td>Missing or invalid token</td></tr>
            <tr><td>4</td><td>Not found</td><td>Bucket or object does not exist</td></tr>
            <tr><td>5</td><td>Conflict</td><td>Object already exists (no overwrite)</td></tr>
            <tr><td>6</td><td>Permission</td><td>Token lacks required scope</td></tr>
            <tr><td>7</td><td>Network</td><td>Connection failed or timed out</td></tr>
          </tbody>
        </table>
        <div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--text-2);
          padding:16px 18px;background:var(--surface-alt);border:1px solid var(--border);
          margin-top:16px;line-height:1.9">
<span class="t-cmd">storage get</span> backups/db/latest.sql.gz .
<span class="t-cmd">case</span> $? <span class="t-cmd">in</span>
  0) echo "restored ok" ;;
  4) echo "no backup found" ;;
  3) echo "auth failed — check token" ;;
  7) echo "network error — retry?" ;;
<span class="t-cmd">esac</span></div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ══════════════════════════════════════════════════════════
     FEATURES
     ══════════════════════════════════════════════════════════ -->
<section class="section" id="features">
  <div class="wrap">
    <div class="label">Design</div>
    <div class="h2" style="margin-bottom:48px">Built to stay<br><span class="grad">out of your way.</span></div>
  </div>
  <div class="features" style="border-left:none;border-right:none">
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </div>
      <div class="feat-name">Single binary</div>
      <div class="feat-desc">One file, ~2 MB. No runtime, no package manager, no virtualenv. Drop it in <code>/usr/local/bin</code> and it works everywhere.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
      </div>
      <div class="feat-name">TTY-aware output</div>
      <div class="feat-desc">Progress bars and colors for humans. Plain text for pipes. Detection is automatic &mdash; no flags needed to switch modes.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
      </div>
      <div class="feat-name">--json everywhere</div>
      <div class="feat-desc">Every command that produces output supports <code>--json</code>. Parse with <code>jq</code>, Python, Node.js, or any JSON-capable tool.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      </div>
      <div class="feat-name">Secure by default</div>
      <div class="feat-desc">Token files are written with <code>0600</code> permissions. Tokens are never logged or printed. HTTPS-only with certificate pinning.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"/></svg>
      </div>
      <div class="feat-name">Resumable uploads</div>
      <div class="feat-desc">Large files use TUS protocol under the hood. If a connection drops, the upload resumes automatically from where it left off.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
      </div>
      <div class="feat-name">Scoped API keys</div>
      <div class="feat-desc">Create keys scoped to specific buckets and operations. Your CI job only gets write access to the one bucket it needs. Rotate without disruption.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
      </div>
      <div class="feat-name">Edge network</div>
      <div class="feat-desc">Uploads and downloads are served from 300+ edge locations. Fast anywhere in the world. Zero egress fees.</div>
    </div>
    <div class="feat">
      <div class="feat-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
      </div>
      <div class="feat-name">Language SDKs</div>
      <div class="feat-desc">Prefer code? Use the REST API or official SDKs for Node.js, Go, and Python. The CLI is just a thin wrapper over the same API you can call directly.</div>
    </div>
  </div>
</section>

<!-- ══════════════════════════════════════════════════════════
     CTA
     ══════════════════════════════════════════════════════════ -->
<section class="section section--cta">
  <div class="wrap">
    <div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--text-3);margin-bottom:16px">&gt; ready?</div>
    <div class="cta-title">Install it now.<br>You're already in the terminal.</div>
    <p class="cta-sub">One command to install. One command to log in. Then it's just files.</p>

    <div class="install-box" style="max-width:520px;margin-bottom:32px">
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
