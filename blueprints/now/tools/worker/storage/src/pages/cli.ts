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
<meta name="description" content="The Storage CLI. Upload, download, and share files from your terminal. Zero dependencies. Works on macOS, Linux, and Windows.">
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

<!-- HERO: two-column layout — left: headline + install, right: terminal demo -->
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

<!-- HOW IT WORKS -->
<section class="section" id="quickstart">
  <div class="wrap">
    <div class="label">How it works</div>
    <div class="h2">Three commands. That's it.</div>
    <p class="sub" style="margin-bottom:36px">Log in, upload a file, share it.</p>

    <div class="steps">
      <div class="step">
        <div class="step-num">01</div>
        <div class="step-title">Log in</div>
        <div class="step-desc">Opens your browser. Token saved locally.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> storage login</span>
          <button class="step-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="step">
        <div class="step-num">02</div>
        <div class="step-title">Upload</div>
        <div class="step-desc">Put any file into your storage.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> storage put report.pdf docs/</span>
          <button class="step-copy" onclick="copyText('storage put report.pdf docs/',this)">copy</button>
        </div>
      </div>
      <div class="step">
        <div class="step-num">03</div>
        <div class="step-title">Share</div>
        <div class="step-desc">Get a link. It expires automatically.</div>
        <div class="step-cmd">
          <span class="step-cmd-text"><span class="t-prompt">$</span> storage share docs/report.pdf</span>
          <button class="step-copy" onclick="copyText('storage share docs/report.pdf',this)">copy</button>
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
    <div class="h2">Everything the CLI can do.</div>
    <p class="sub" style="margin-bottom:36px">Every command supports <code>--json</code> for scripting and <code>--quiet</code> for silence.</p>

    <div class="cmd-grid">
      <div class="cmd-card">
        <div class="cmd-name">put</div>
        <div class="cmd-desc">Upload a file or stdin.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage put photo.jpg images/</span>
          <button class="cmd-copy" onclick="copyText('storage put photo.jpg images/',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">get</div>
        <div class="cmd-desc">Download a file to disk.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage get images/photo.jpg</span>
          <button class="cmd-copy" onclick="copyText('storage get images/photo.jpg',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">cat</div>
        <div class="cmd-desc">Print file contents to stdout.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage cat config.json</span>
          <button class="cmd-copy" onclick="copyText('storage cat config.json',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">ls</div>
        <div class="cmd-desc">List files at a path.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage ls docs/</span>
          <button class="cmd-copy" onclick="copyText('storage ls docs/',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">share</div>
        <div class="cmd-desc">Create a temporary public link.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage share report.pdf</span>
          <button class="cmd-copy" onclick="copyText('storage share report.pdf',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">rm</div>
        <div class="cmd-desc">Delete one or more files.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage rm old-draft.md</span>
          <button class="cmd-copy" onclick="copyText('storage rm old-draft.md',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">mv</div>
        <div class="cmd-desc">Move or rename a file.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage mv draft.md final.md</span>
          <button class="cmd-copy" onclick="copyText('storage mv draft.md final.md',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">find</div>
        <div class="cmd-desc">Search files by name.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage find "report"</span>
          <button class="cmd-copy" onclick="copyText('storage find &quot;report&quot;',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">stat</div>
        <div class="cmd-desc">Show storage usage.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage stat</span>
          <button class="cmd-copy" onclick="copyText('storage stat',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">login</div>
        <div class="cmd-desc">Authenticate via browser.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage login</span>
          <button class="cmd-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">token</div>
        <div class="cmd-desc">Show or set your token.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage token</span>
          <button class="cmd-copy" onclick="copyText('storage token',this)">copy</button>
        </div>
      </div>
      <div class="cmd-card">
        <div class="cmd-name">key</div>
        <div class="cmd-desc">Manage API keys.</div>
        <div class="cmd-example">
          <span class="cmd-example-text">$ storage key create deploy</span>
          <button class="cmd-copy" onclick="copyText('storage key create deploy',this)">copy</button>
        </div>
      </div>
    </div>
  </div>
</section>

<hr class="section-divider">

<!-- USE CASES -->
<section class="section" id="use-cases">
  <div class="wrap">
    <div class="label">Use cases</div>
    <div class="h2">Built for real workflows.</div>
    <p class="sub" style="margin-bottom:36px">The CLI fits anywhere you have a terminal.</p>

    <div class="uc-grid">
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">CI / CD</div>
          <div class="uc-title">Deploy from any pipeline</div>
          <div class="uc-desc">Set <code>STORAGE_TOKEN</code> as a secret and upload build artifacts.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>storage put dist/app.js cdn/</span>
          <button class="uc-copy" onclick="copyText('storage put dist/app.js cdn/',this)">copy</button>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Backups</div>
          <div class="uc-title">Pipe straight to storage</div>
          <div class="uc-desc">No temp files needed. Stream from any command.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>pg_dump mydb | storage put - backup.sql</span>
          <button class="uc-copy" onclick="copyText('pg_dump mydb | storage put - backup.sql',this)">copy</button>
        </div>
      </div>
      <div class="uc">
        <div class="uc-head">
          <div class="uc-tag">Scripting</div>
          <div class="uc-title">JSON output on every command</div>
          <div class="uc-desc">Pipe <code>--json</code> output into jq or any tool.</div>
        </div>
        <div class="uc-cmd">
          <span class="uc-cmd-text"><span class="t-prompt">$ </span>storage ls --json | jq '.'</span>
          <button class="uc-copy" onclick="copyText('storage ls --json | jq \\'.\\'',this)">copy</button>
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
    <div class="h2">Two ways to authenticate.</div>
    <p class="sub" style="margin-bottom:36px">Browser login for your laptop. API keys for your scripts.</p>

    <div class="auth-grid">
      <div class="auth-card">
        <div class="auth-badge">Interactive</div>
        <div class="auth-title">Browser login</div>
        <div class="auth-desc">Opens your browser. No password needed.</div>
        <div class="auth-cmd">
          <span class="auth-cmd-text">$ storage login</span>
          <button class="auth-copy" onclick="copyText('storage login',this)">copy</button>
        </div>
      </div>
      <div class="auth-card">
        <div class="auth-badge">Automation</div>
        <div class="auth-title">API keys</div>
        <div class="auth-desc">For CI and scripts. Set <code>STORAGE_TOKEN</code> env var.</div>
        <div class="auth-cmd">
          <span class="auth-cmd-text">$ storage key create deploy</span>
          <button class="auth-copy" onclick="copyText('storage key create deploy',this)">copy</button>
        </div>
      </div>
    </div>

    <div class="token-note">
      Token resolution: <code>--token</code> flag &rarr; <code>STORAGE_TOKEN</code> env &rarr; <code>~/.config/storage/token</code>
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

<div class="machine-view" id="machine-view">
  <div class="md" id="md-content">
<span class="h1"># Storage CLI</span>

Upload, download, and share files from your terminal.
Single binary. Zero dependencies. macOS, Linux, Windows.

<span class="h2">## Installation</span>

macOS / Linux
curl -fsSL https://storage.liteio.dev/cli/install.sh | sh

Windows
irm https://storage.liteio.dev/cli/install.ps1 | iex

npm
npm i -g @liteio/storage-cli

Bun
bun i -g @liteio/storage-cli

Deno
deno install -g npm:@liteio/storage-cli

<span class="h2">## Commands</span>

<span class="h3">### upload</span>
storage put &lt;file&gt; &lt;destination&gt;
Upload a local file or stdin to storage.
<span class="dim">Supports --json and --quiet flags.</span>

<span class="h3">### download</span>
storage get &lt;path&gt;
Download a file from storage to disk.

storage cat &lt;path&gt;
Print file contents to stdout.

<span class="h3">### list</span>
storage ls [path]
List files and folders at a given path.
<span class="dim">Use --json for machine-readable output.</span>

<span class="h3">### share</span>
storage share &lt;path&gt;
Create a temporary public link to a file.
<span class="dim">Link expires automatically after 24h.</span>

<span class="h3">### search</span>
storage find &lt;query&gt;
Search files by name.

<span class="h3">### move</span>
storage mv &lt;source&gt; &lt;destination&gt;
Move or rename a file.

<span class="h3">### delete</span>
storage rm &lt;path&gt;
Delete one or more files.

<span class="h3">### stats</span>
storage stat
Show storage usage and quota.

<span class="h3">### auth</span>
storage login
Authenticate via browser. Opens your default browser and saves the token locally.

storage token
Show or set your current auth token.

storage key create &lt;name&gt;
Create a named API key for automation.

<span class="h2">## Authentication</span>

Two methods are supported:

1. Browser login (interactive)
   Run storage login. Opens your browser. No password needed.

2. API keys (automation)
   Run storage key create &lt;name&gt; to generate a key.
   Set STORAGE_TOKEN env var in CI or scripts.

Token resolution order:
--token flag, then STORAGE_TOKEN env, then ~/.config/storage/token

<span class="h2">## Usage Examples</span>

Upload a file
storage put photo.jpg images/

Download a file
storage get images/photo.jpg

List directory contents
storage ls docs/

Share a file
storage share docs/report.pdf

Pipe from another command
pg_dump mydb | storage put - backup.sql

JSON output for scripting
storage ls --json | jq '.'

Create an API key for CI
storage key create deploy

<span class="dim">All commands support --json for structured output and --quiet to suppress non-essential output.</span>
  </div>
</div>

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
