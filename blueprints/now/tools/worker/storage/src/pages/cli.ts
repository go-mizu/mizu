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
<title>CLI — Storage</title>
<meta name="description" content="The Storage CLI. Upload, download, share, search, and manage files from your terminal. Single binary, zero dependencies.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/base.css">
<link rel="stylesheet" href="/cli.css">
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

<div class="human-view" id="human-view">
<main>

<!-- ═══════════════════════════════════════════════════════════════════
     HERO
     ═══════════════════════════════════════════════════════════════════ -->
<section class="hero">
  <div class="hero-inner">
    <div class="hero-left">
      <div class="hero-eyebrow">
        <span class="hero-eyebrow-dot"></span>
        @liteio/storage-cli
      </div>
      <h1 class="hero-title">Files from<br>your <em>terminal.</em></h1>
      <p class="hero-sub">Upload, download, and share files with one command. Single binary. No dependencies.</p>

      <div class="install-section">
        <div class="install-label">Install</div>
        <div class="install-tabs" id="install-tabs">
          <button class="install-tab active" data-method="curl">macOS / Linux</button>
          <button class="install-tab" data-method="windows">Windows</button>
          <button class="install-tab" data-method="npm">npm</button>
          <button class="install-tab" data-method="bun">Bun</button>
          <button class="install-tab" data-method="deno">Deno</button>
        </div>
        <div class="install-box">
          <span class="prompt" id="install-prompt">$</span>
          <span class="cmd" id="install-cmd">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
          <button class="copy-btn" onclick="copyInstall(this)">copy</button>
        </div>
        <div class="dl-grid" id="dl-grid"></div>
      </div>
    </div>

    <div class="hero-right">
      <div class="hero-term">
        <div class="term-bar">
          <div class="term-dots"><span></span><span></span><span></span></div>
          <span class="term-title">storage</span>
          <div style="width:27px"></div>
        </div>
        <div class="term-body">
<span class="t-prompt">$</span> <span class="t-cmd">storage put</span> <span class="t-flag">demo.pdf docs/</span>
<br><span class="t-ok">PUT</span> <span class="t-dim">docs/demo.pdf</span> <span class="t-num">2.4 MB</span>
<span class="t-blank"></span>
<br><span class="t-prompt">$</span> <span class="t-cmd">storage ls</span> <span class="t-flag">docs/</span>
<br><span class="t-dim">demo.pdf</span>       <span class="t-num">2.4 MB</span>  <span class="t-dim">just now</span>
<br><span class="t-dim">readme.md</span>      <span class="t-num">1.2 KB</span>  <span class="t-dim">2h ago</span>
<span class="t-blank"></span>
<br><span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">docs/demo.pdf</span>
<br><span class="t-url">https://storage.liteio.dev/s/k7x9m2</span>
<br><span class="t-dim">expires in 24h</span>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     TUTORIAL: Get started in 60 seconds
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="quickstart">
  <div class="wrap">
    <div class="label">Tutorial</div>
    <div class="h2">Up and running in 60 seconds.</div>
    <p class="sub" style="margin-bottom:36px">Install, authenticate, upload your first file.</p>

    <div class="steps">
      <div class="step">
        <div class="step-num">01</div>
        <div class="step-title">Install the CLI</div>
        <div class="step-desc">A single binary, no runtime needed. Works immediately after install.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
          <button class="step-copy" onclick="copyText('curl -fsSL https://storage.liteio.dev/cli/install.sh | sh',this)">copy</button>
        </div>
      </div>
      <div class="step">
        <div class="step-num">02</div>
        <div class="step-title">Sign in</div>
        <div class="step-desc">Opens your browser, confirms your identity, saves the token locally.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> storage login</span>
          <button class="step-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="step">
        <div class="step-num">03</div>
        <div class="step-title">Upload something</div>
        <div class="step-desc">Any file, any size. It's in your storage in seconds.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> storage put report.pdf docs/</span>
          <button class="step-copy" onclick="copyText('storage put report.pdf docs/',this)">copy</button>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ═══════════════════════════════════════════════════════════════════
     COMMAND REFERENCE
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="commands">
  <div class="wrap">
    <div class="label">Command reference</div>
    <div class="h2">Everything the CLI can do.</div>
    <p class="sub" style="margin-bottom:36px">Every command supports <code>--json</code> for scripting and <code>--quiet</code> for silence.</p>

    <!-- File operations -->
    <div class="cmd-section-label">File operations</div>
    <div class="cmd-grid">
      <div class="cmd-card">
        <div class="cmd-name">put</div>
        <div class="cmd-aliases">aliases: upload, push</div>
        <div class="cmd-desc">Upload a file or pipe from stdin. Streams directly to the edge — no temp files.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage put photo.jpg images/</span>
          <button class="cmd-copy" onclick="copyText('storage put photo.jpg images/',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">get</div>
        <div class="cmd-aliases">aliases: download, pull</div>
        <div class="cmd-desc">Download a file to the current directory, or specify a destination path.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage get images/photo.jpg</span>
          <button class="cmd-copy" onclick="copyText('storage get images/photo.jpg',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">cat</div>
        <div class="cmd-aliases">aliases: read</div>
        <div class="cmd-desc">Print file contents to stdout. Perfect for piping into other tools.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage cat config.json | jq '.'</span>
          <button class="cmd-copy" onclick="copyText('storage cat config.json | jq \\'.\\'',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">ls</div>
        <div class="cmd-aliases">aliases: list</div>
        <div class="cmd-desc">List files and folders at a path. Shows name, size, and modified time.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage ls docs/</span>
          <button class="cmd-copy" onclick="copyText('storage ls docs/',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">mv</div>
        <div class="cmd-aliases">aliases: move, rename</div>
        <div class="cmd-desc">Move or rename a file. Works across folders.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage mv draft.md final.md</span>
          <button class="cmd-copy" onclick="copyText('storage mv draft.md final.md',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">rm</div>
        <div class="cmd-aliases">aliases: delete, del</div>
        <div class="cmd-desc">Delete one or more files. Folders too, with everything inside.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage rm old-draft.md</span>
          <button class="cmd-copy" onclick="copyText('storage rm old-draft.md',this)">copy</button>
        </div>
      </div>
    </div>

    <!-- Discovery -->
    <div class="cmd-section-label" style="margin-top:48px">Discovery</div>
    <div class="cmd-grid">
      <div class="cmd-card">
        <div class="cmd-name">find</div>
        <div class="cmd-aliases">aliases: search</div>
        <div class="cmd-desc">Search files by name across your entire storage. Multi-word queries with relevance scoring.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage find "quarterly report"</span>
          <button class="cmd-copy" onclick="copyText('storage find &quot;quarterly report&quot;',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">stat</div>
        <div class="cmd-aliases">aliases: stats</div>
        <div class="cmd-desc">See how much storage you're using. Shows file count and total bytes.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage stat</span>
          <button class="cmd-copy" onclick="copyText('storage stat',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">share</div>
        <div class="cmd-aliases">aliases: sign</div>
        <div class="cmd-desc">Create a temporary public link. Set a custom TTL or use the default (1 hour, max 7 days).</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage share report.pdf --ttl 86400</span>
          <button class="cmd-copy" onclick="copyText('storage share report.pdf --ttl 86400',this)">copy</button>
        </div>
      </div>
    </div>

    <!-- Auth & keys -->
    <div class="cmd-section-label" style="margin-top:48px">Authentication</div>
    <div class="cmd-grid">
      <div class="cmd-card">
        <div class="cmd-name">login</div>
        <div class="cmd-desc">Authenticate via your browser. Opens a browser tab, verifies your identity, saves the token to <code>~/.config/storage/token</code>.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage login</span>
          <button class="cmd-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">token</div>
        <div class="cmd-desc">Show your current token and where it came from (flag, env, or file). Or set a new one.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage token</span>
          <button class="cmd-copy" onclick="copyText('storage token',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">key</div>
        <div class="cmd-aliases">aliases: keys</div>
        <div class="cmd-desc">Create, list, or revoke API keys. Keys can be scoped to a path prefix for security.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage key create deploy --prefix cdn/</span>
          <button class="cmd-copy" onclick="copyText('storage key create deploy --prefix cdn/',this)">copy</button>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ═══════════════════════════════════════════════════════════════════
     GLOBAL FLAGS
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="flags">
  <div class="wrap">
    <div class="label">Global flags</div>
    <div class="h2">Works on every command.</div>
    <p class="sub" style="margin-bottom:36px">Pass these to any command to control output and auth.</p>

    <div class="flag-table">
      <div class="flag-row flag-header">
        <div class="flag-col-name">Flag</div>
        <div class="flag-col-short">Short</div>
        <div class="flag-col-desc">What it does</div>
      </div>
      <div class="flag-row">
        <div class="flag-col-name"><code>--json</code></div>
        <div class="flag-col-short"><code>-j</code></div>
        <div class="flag-col-desc">Output as JSON — pipe into <code>jq</code> or any tool</div>
      </div>
      <div class="flag-row">
        <div class="flag-col-name"><code>--quiet</code></div>
        <div class="flag-col-short"><code>-q</code></div>
        <div class="flag-col-desc">Suppress non-essential output</div>
      </div>
      <div class="flag-row">
        <div class="flag-col-name"><code>--token</code></div>
        <div class="flag-col-short"><code>-t</code></div>
        <div class="flag-col-desc">Use a specific token (overrides env and config file)</div>
      </div>
      <div class="flag-row">
        <div class="flag-col-name"><code>--endpoint</code></div>
        <div class="flag-col-short"></div>
        <div class="flag-col-desc">Override the API base URL</div>
      </div>
      <div class="flag-row">
        <div class="flag-col-name"><code>--no-color</code></div>
        <div class="flag-col-short"></div>
        <div class="flag-col-desc">Disable colored output (also: <code>NO_COLOR=1</code>)</div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ═══════════════════════════════════════════════════════════════════
     RECIPES: Real-world workflows
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="recipes">
  <div class="wrap">
    <div class="label">Recipes</div>
    <div class="h2">Real-world workflows.</div>
    <p class="sub" style="margin-bottom:36px">Copy-paste these into your terminal, CI config, or scripts.</p>

    <div class="uc-grid">
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">CI / CD</div>
          <div class="uc-title">Upload build artifacts</div>
          <div class="uc-desc">Set <code>STORAGE_TOKEN</code> as a secret in your CI. Upload happens in one step.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>storage put dist/app.js cdn/v1.2.0/</span>
          <button class="uc-copy" onclick="copyText('storage put dist/app.js cdn/v1.2.0/',this)">copy</button>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Backups</div>
          <div class="uc-title">Stream from any command</div>
          <div class="uc-desc">Pipe directly into storage. No temp files, no disk space needed.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>pg_dump mydb | storage put - backups/db.sql</span>
          <button class="uc-copy" onclick="copyText('pg_dump mydb | storage put - backups/db.sql',this)">copy</button>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Scripting</div>
          <div class="uc-title">JSON output for automation</div>
          <div class="uc-desc">Every command supports <code>--json</code>. Pipe into <code>jq</code>, <code>fx</code>, or your own scripts.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>storage ls --json | jq '.[].name'</span>
          <button class="uc-copy" onclick="copyText('storage ls --json | jq \\'.[].name\\'',this)">copy</button>
        </div>
      </div>
    </div>

    <!-- More recipes as terminal demos -->
    <div class="recipe-grid" style="margin-top:48px">
      <div class="recipe">
        <div class="recipe-label">Scoped API key for a deploy pipeline</div>
        <div class="hero-term">
          <div class="term-bar"><div class="term-dots"><span></span><span></span><span></span></div><span class="term-title">storage</span><div style="width:27px"></div></div>
          <div class="term-body">
<span class="t-prompt">$</span> <span class="t-cmd">storage key create</span> <span class="t-flag">github-deploy --prefix cdn/</span>
<br><span class="t-ok">CREATED</span> <span class="t-dim">github-deploy</span>
<br><span class="t-dim">prefix:</span> cdn/
<br><span class="t-dim">token:</span>  <span class="t-url">sk_a8f3c7e2d1b9...4k2m</span>
<span class="t-blank"></span>
<br><span class="t-dim">Add to your CI secrets:</span>
<br><span class="t-dim">STORAGE_TOKEN=sk_a8f3c7e2d1b9...4k2m</span>
          </div>
        </div>
      </div>
      <div class="recipe">
        <div class="recipe-label">Share a file with a 24-hour link</div>
        <div class="hero-term">
          <div class="term-bar"><div class="term-dots"><span></span><span></span><span></span></div><span class="term-title">storage</span><div style="width:27px"></div></div>
          <div class="term-body">
<span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">docs/report.pdf --ttl 86400</span>
<br><span class="t-url">https://storage.liteio.dev/s/k7x9m2</span>
<br><span class="t-dim">expires in 24h</span>
<span class="t-blank"></span>
<br><span class="t-prompt">$</span> <span class="t-cmd">storage share</span> <span class="t-flag">photos/team.jpg</span>
<br><span class="t-url">https://storage.liteio.dev/s/m2n8p4</span>
<br><span class="t-dim">expires in 1h</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- ═══════════════════════════════════════════════════════════════════
     AUTHENTICATION
     ═══════════════════════════════════════════════════════════════════ -->
<section class="section" id="auth">
  <div class="wrap">
    <div class="label">Authentication</div>
    <div class="h2">Two ways to authenticate.</div>
    <p class="sub" style="margin-bottom:36px">Browser login for your laptop. API keys for your scripts.</p>

    <div class="auth-grid">
      <div class="auth-card">
        <div class="auth-badge">Interactive</div>
        <div class="auth-title">Browser login</div>
        <div class="auth-desc">Opens your browser. No password needed. Token saved to <code>~/.config/storage/token</code>.</div>
        <div class="auth-cmd">
          <span class="auth-cmd-text">$ storage login</span>
          <button class="auth-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="auth-card">
        <div class="auth-badge">Automation</div>
        <div class="auth-title">API keys</div>
        <div class="auth-desc">For CI and scripts. Scope keys to a prefix for security. Set <code>STORAGE_TOKEN</code> env var.</div>
        <div class="auth-cmd">
          <span class="auth-cmd-text">$ storage key create deploy --prefix cdn/</span>
          <button class="auth-copy" onclick="copyText('storage key create deploy --prefix cdn/',this)">copy</button>
        </div>
      </div>
    </div>

    <div class="token-note">
      Token resolution order: <code>--token</code> flag &rarr; <code>STORAGE_TOKEN</code> env &rarr; <code>~/.config/storage/token</code>
    </div>
  </div>
</section>

<!-- CTA -->
<section class="section section--cta">
  <div class="wrap">
    <div class="cta-prompt">&gt; ready?</div>
    <div class="cta-title">Try it now.</div>
    <p class="cta-sub">One command to install. Works on macOS, Linux, and Windows.</p>

    <div class="install-box" style="max-width:520px;margin-bottom:32px">
      <span class="prompt">$</span>
      <span class="cmd" id="cta-cmd">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>
      <button class="copy-btn" onclick="copyInstallCta(this)">copy</button>
    </div>

    <div class="cta-actions">
      <a href="/api">API Reference</a>
      <a href="/developers">Developer Guide</a>
      <a href="/pricing">Pricing</a>
    </div>
  </div>
</section>

</main>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     MACHINE VIEW
     ═══════════════════════════════════════════════════════════════════ -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button><span class="h1"># Storage CLI</span>

Upload, download, share, and manage files from your terminal.
Single binary. Zero dependencies. macOS, Linux, Windows.

<span class="h2">## Installation</span>

<span class="h3">### macOS / Linux</span>
<span class="dim">curl -fsSL https://storage.liteio.dev/cli/install.sh | sh</span>

<span class="h3">### Windows (PowerShell)</span>
<span class="dim">irm https://storage.liteio.dev/cli/install.ps1 | iex</span>

<span class="h3">### npm / Bun / Deno</span>
<span class="dim">npm i -g @liteio/storage-cli</span>
<span class="dim">bun i -g @liteio/storage-cli</span>
<span class="dim">deno install -g npm:@liteio/storage-cli</span>

<span class="h3">### Direct download</span>
Apple Silicon  <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-darwin-arm64</span>
Intel Mac      <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-darwin-amd64</span>
Linux x64      <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-linux-amd64</span>
Linux ARM      <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-linux-arm64</span>
Windows x64    <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-windows-amd64.exe</span>
Windows ARM    <span class="link">https://storage.liteio.dev/cli/releases/latest/storage-windows-arm64.exe</span>

<span class="h2">## Quick Start</span>

<span class="dim">Step 1: Install</span>
curl -fsSL https://storage.liteio.dev/cli/install.sh | sh

<span class="dim">Step 2: Sign in</span>
storage login

<span class="dim">Step 3: Upload a file</span>
storage put report.pdf docs/

<span class="dim">Step 4: Share it</span>
storage share docs/report.pdf

<span class="h2">## Commands</span>

<span class="h3">### File Operations</span>

<span class="dim">storage put &lt;file&gt; [destination]</span>
Upload a file or stdin to storage.
Aliases: upload, push
Use - as file to read from stdin: pg_dump mydb | storage put - backup.sql

<span class="dim">storage get &lt;path&gt; [destination]</span>
Download a file from storage to disk.
Aliases: download, pull

<span class="dim">storage cat &lt;path&gt;</span>
Print file contents to stdout.
Aliases: read

<span class="dim">storage ls [path]</span>
List files and folders at a given path.
Aliases: list
Shows name, size, content type, and last modified time.

<span class="dim">storage mv &lt;source&gt; &lt;destination&gt;</span>
Move or rename a file.
Aliases: move, rename

<span class="dim">storage rm &lt;path...&gt;</span>
Delete one or more files. Folders too (recursive).
Aliases: delete, del

<span class="h3">### Discovery</span>

<span class="dim">storage find &lt;query&gt;</span>
Search files by name across your entire storage.
Aliases: search
Multi-word queries with relevance scoring.

<span class="dim">storage stat</span>
Show storage usage: total files and bytes used.
Aliases: stats

<span class="h3">### Sharing</span>

<span class="dim">storage share &lt;path&gt; [--ttl &lt;seconds&gt;]</span>
Create a temporary public link.
Aliases: sign
Default TTL: 1 hour. Maximum: 7 days (604800 seconds).
Flags: --ttl, --expires, -x

<span class="h3">### Authentication</span>

<span class="dim">storage login [name]</span>
Authenticate via browser. Opens your default browser and saves the token.
Optional: pass a name to register a new account.

<span class="dim">storage logout</span>
Remove saved credentials and invalidate session.

<span class="dim">storage token [&lt;token&gt;]</span>
Show current auth token and its source, or set a new one.

<span class="dim">storage key create &lt;name&gt; [--prefix &lt;path&gt;]</span>
Create a named API key. Optionally scope to a path prefix.
Aliases for key: keys

<span class="dim">storage key list</span>
List all API keys with metadata.
Aliases: ls

<span class="dim">storage key revoke &lt;id&gt;</span>
Revoke an API key by ID.
Aliases: delete, rm

<span class="h2">## Global Flags</span>

--json, -j       Output as JSON (for scripting)
--quiet, -q      Suppress non-essential output
--token, -t      Use a specific token
--endpoint       Override API base URL
--no-color       Disable colored output (also: NO_COLOR=1)
--help, -h       Show help
--version, -V    Print version

<span class="h2">## Authentication</span>

Two methods are supported:

<span class="h3">### Browser login (interactive)</span>
storage login
Opens your browser. Saves token to ~/.config/storage/token.

<span class="h3">### API keys (automation)</span>
storage key create &lt;name&gt; --prefix &lt;path&gt;
Creates a scoped API key. Set STORAGE_TOKEN in your CI or scripts.

Token resolution order:
1. --token flag (highest priority)
2. STORAGE_TOKEN environment variable
3. ~/.config/storage/token file

<span class="h2">## Environment Variables</span>

STORAGE_TOKEN       API key or session token
STORAGE_ENDPOINT    API base URL (default: https://storage.liteio.dev)
NO_COLOR            Disable colored output
XDG_CONFIG_HOME     Config directory base (default: ~/.config)

<span class="h2">## Recipes</span>

<span class="h3">### Upload build artifacts from CI</span>
export STORAGE_TOKEN=$SECRET_TOKEN
storage put dist/app.js cdn/v1.2.0/
storage put dist/app.css cdn/v1.2.0/

<span class="h3">### Stream a database backup</span>
pg_dump mydb | storage put - backups/$(date +%Y-%m-%d).sql

<span class="h3">### Share a file for 24 hours</span>
storage share docs/report.pdf --ttl 86400

<span class="h3">### List files as JSON and filter with jq</span>
storage ls docs/ --json | jq '.[].name'

<span class="h3">### Create a scoped deploy key</span>
storage key create github-deploy --prefix cdn/

<span class="h3">### Download and pipe to another tool</span>
storage cat config.json | jq '.database'

<span class="h3">### Move files between folders</span>
storage mv drafts/post.md published/post.md

<span class="h3">### Bulk delete old files</span>
storage ls archive/ --json | jq -r '.[].path' | xargs -I{} storage rm {}

<span class="h3">### Check storage usage</span>
storage stat --json | jq '{files: .count, mb: (.bytes / 1048576 | floor)}'

<span class="h2">## Links</span>

<span class="link">https://storage.liteio.dev/api</span>         API reference
<span class="link">https://storage.liteio.dev/developers</span>  Developer guide
<span class="link">https://storage.liteio.dev/pricing</span>     Pricing
<span class="link">https://storage.liteio.dev/cli</span>         This page (human view)
  </div>
</div>

<!-- Floating mode switch -->
<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

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
  const prompt=document.getElementById('install-prompt');
  const dlGrid=document.getElementById('dl-grid');
  if(!tabs.length||!cmd) return;

  const methods={
    curl:{cmd:'curl -fsSL https://storage.liteio.dev/cli/install.sh | sh',prompt:'$',
      dl:[
        {href:'/cli/releases/latest/storage-darwin-arm64',name:'Apple Silicon',meta:'arm64'},
        {href:'/cli/releases/latest/storage-darwin-amd64',name:'Intel Mac',meta:'amd64'},
        {href:'/cli/releases/latest/storage-linux-amd64',name:'Linux x64',meta:'amd64'},
        {href:'/cli/releases/latest/storage-linux-arm64',name:'Linux ARM',meta:'arm64'},
      ]},
    windows:{cmd:'irm https://storage.liteio.dev/cli/install.ps1 | iex',prompt:'>',
      dl:[
        {href:'/cli/releases/latest/storage-windows-amd64.exe',name:'Windows x64',meta:'amd64'},
        {href:'/cli/releases/latest/storage-windows-arm64.exe',name:'Windows ARM',meta:'arm64'},
      ]},
    npm:{cmd:'npm i -g @liteio/storage-cli',prompt:'$',dl:[]},
    bun:{cmd:'bun i -g @liteio/storage-cli',prompt:'$',dl:[]},
    deno:{cmd:'deno install -g npm:@liteio/storage-cli',prompt:'$',dl:[]},
  };

  const dlIcon='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>';

  function activate(method){
    const m=methods[method];
    cmd.textContent=m.cmd;
    prompt.textContent=m.prompt;
    if(m.dl.length){
      dlGrid.innerHTML=m.dl.map(d=>
        '<a href="'+d.href+'" class="dl-btn"><span class="dl-icon">'+dlIcon+'</span><span class="dl-info"><span class="dl-name">'+d.name+'</span><span class="dl-meta">'+d.meta+'</span></span></a>'
      ).join('');
      dlGrid.style.display='flex';
    } else {
      dlGrid.innerHTML='';
      dlGrid.style.display='none';
    }
  }

  tabs.forEach(tab=>{
    tab.addEventListener('click',()=>{
      tabs.forEach(t=>t.classList.remove('active'));
      tab.classList.add('active');
      activate(tab.dataset.method);
    });
  });
  activate('curl');
})();

/* Copy helpers */
function copyText(text,btn){
  navigator.clipboard.writeText(text).then(()=>{
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},1500);
  });
}

function copyInstall(btn){
  const el=document.getElementById('install-cmd');
  if(el) copyText(el.textContent,btn);
}

function copyInstallCta(btn){
  const el=document.getElementById('cta-cmd');
  if(el) copyText(el.textContent,btn);
}

function copyMd(){
  const el=document.getElementById('md-content');
  if(el){
    navigator.clipboard.writeText(el.innerText).then(()=>{
      const btn=el.querySelector('.md-copy');
      if(btn){btn.textContent='copied';setTimeout(()=>{btn.textContent='copy'},1500)}
    });
  }
}

function setMode(mode){
  var btns=document.querySelectorAll('.mode-switch button');
  btns.forEach(function(b){b.classList.remove('active')});
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
</script>
</body>
</html>`;
}
