import { esc } from "./layout";

export function homePage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  /* ── Hero ─────────────────────────────────────────────────────────── */
  const heroSignedIn = `
<div class="hero-greeting">Welcome back, <strong>${displayName}</strong></div>
<div class="hero-actions">
  <a href="/browse" class="btn btn--primary"><span class="btn-icon">&gt;_</span> Open dashboard</a>
  <a href="/docs" class="btn btn--ghost">Documentation</a>
</div>`;

  const heroSignedOut = `
<div class="hero-badge">Now available <span>&rarr;</span></div>
<h1 class="hero-title">One place for<br>your files.</h1>
<p class="hero-sub">Where your team collaborates, AI assists,<br>and developers build &mdash; all in one workspace.</p>
<div class="hero-ctas">
  <a href="#get-started" class="btn btn--primary btn--lg" onclick="document.getElementById('email-input')?.focus();return false">Get started free</a>
  <a href="/browse" class="btn btn--ghost btn--lg">See it in action</a>
</div>`;

  /* ── Store (split: text left, mockup right) ────────────────────────── */
  const storeSection = `
<section class="section" id="store">
  <div class="section-pad">
    <div class="split">
      <div>
        <div class="section-label">YOUR FILES</div>
        <div class="section-heading">Drop it in.<br>It's there.</div>
        <p class="section-desc">Documents, images, spreadsheets &mdash; anything your business needs. Organized in folders, found in seconds, shared with a link.</p>
      </div>
      <div class="window">
        <div class="window-bar">
          <div class="window-dots"><span></span><span></span><span></span></div>
          <div class="window-title">storage.liteio.dev/browse</div>
        </div>
        <div class="window-body">
          <div class="window-sidebar">
            <div class="sb-item active">My Files</div>
            <div class="sb-item">Shared</div>
            <div class="sb-item">Starred</div>
          </div>
          <div class="window-files">
            <div class="file-row"><span class="file-dot file-dot--folder"></span><span class="file-name">Client Proposals</span><span class="file-meta">12 files</span></div>
            <div class="file-row"><span class="file-dot file-dot--folder"></span><span class="file-name">Q1 Reports</span><span class="file-meta">8 files</span></div>
            <div class="file-row"><span class="file-dot file-dot--doc"></span><span class="file-name">brand-guidelines.pdf</span><span class="file-meta">4.2 MB</span></div>
            <div class="file-row"><span class="file-dot file-dot--sheet"></span><span class="file-name">budget-2026.xlsx</span><span class="file-meta">1.8 MB</span></div>
            <div class="file-row"><span class="file-dot file-dot--img"></span><span class="file-name">team-photo.jpg</span><span class="file-meta">5.1 MB</span></div>
            <div class="file-row"><span class="file-dot file-dot--folder"></span><span class="file-name">Contracts</span><span class="file-meta">15 files</span></div>
            <div class="file-row"><span class="file-dot file-dot--doc"></span><span class="file-name">nda-template.pdf</span><span class="file-meta">1.1 MB</span></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>`;

  /* ── Share (split: visual left, text right) ─────────────────────────── */
  const shareSection = `
<section class="section" id="share">
  <div class="section-pad">
    <div class="split split--reverse">
      <div>
        <div class="section-label">SHARING</div>
        <div class="section-heading">One link.<br>Anyone.</div>
        <p class="section-desc">Share with your team, clients, or partners. They click, they see. No account needed on their end.</p>
      </div>
      <div class="share-visual">
        <div class="link-demo">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
          <span class="link-url">storage.liteio.dev/p/a8f2c9e</span>
          <span class="link-badge">PUBLIC</span>
        </div>
        <div class="share-preview">
          <div class="share-preview-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          </div>
          <div class="share-preview-name">Q4-2025-revenue.xlsx</div>
          <div class="share-preview-meta">Shared by alex@ &middot; 1.8 MB &middot; Updated 2h ago</div>
        </div>
        <div class="share-access">
          <div class="share-access-item">
            <div class="share-access-label">Access</div>
            <div class="share-access-val">Anyone with link</div>
          </div>
          <div class="share-access-item">
            <div class="share-access-label">Expires</div>
            <div class="share-access-val">Never</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>`;

  /* ── AI (split: chat left, text+cards right) ────────────────────────── */
  const aiSection = `
<section class="section" id="ai">
  <div class="glow-spot glow-spot--ai"></div>
  <div class="section-pad">
    <div class="split">
      <div class="chat-demo">
        <div class="chat-msg chat-human">
          <span class="chat-who">You</span>
          <span class="chat-text">Find the Q4 revenue report</span>
        </div>
        <div class="chat-msg chat-ai">
          <span class="chat-who">AI</span>
          <span class="chat-text">Found it &mdash; Q4-2025-revenue.xlsx in Reports. Revenue was $2.4M, up 18% from Q3.</span>
        </div>
        <div class="chat-msg chat-human">
          <span class="chat-who">You</span>
          <span class="chat-text">Organize project files by client</span>
        </div>
        <div class="chat-msg chat-ai">
          <span class="chat-who">AI</span>
          <span class="chat-text">Done. Created 3 folders: Acme Corp, Global Inc, Nova Labs. Moved 28 files.</span>
        </div>
        <div class="chat-msg chat-human">
          <span class="chat-who">You</span>
          <span class="chat-text">Summarize all contracts expiring this quarter</span>
        </div>
        <div class="chat-msg chat-ai">
          <span class="chat-who">AI</span>
          <span class="chat-text">4 contracts expire by June. Total value: $340K. Created expiring-contracts-summary.md in your Documents folder.</span>
        </div>
      </div>
      <div class="ai-right">
        <div class="section-label">AI ASSISTANTS</div>
        <div class="section-heading">Your AI reads<br>your files.</div>
        <p class="section-desc">Connect ChatGPT or Claude. Ask anything about your documents. Get answers, summaries, and organized files &mdash; automatically.</p>
        <div class="ai-connect-cards">
          <div class="ai-connect-card">
            <div class="ai-connect-name">ChatGPT</div>
            <div class="ai-connect-how">Add as a custom GPT</div>
          </div>
          <div class="ai-connect-card">
            <div class="ai-connect-name">Claude</div>
            <div class="ai-connect-how">Connect via MCP</div>
          </div>
        </div>
        <p class="ai-connect-note">Paste your storage URL. Connected in under 2 minutes. <a href="/ai" class="subtle-link">Setup guide &rarr;</a></p>
      </div>
    </div>
  </div>
</section>`;

  /* ── Stats strip ────────────────────────────────────────────────────── */
  const statsSection = `
<section class="section" id="speed">
  <div class="section-pad">
    <div class="stats">
      <div class="stat">
        <div class="stat-num">300+</div>
        <div class="stat-label">Edge locations</div>
      </div>
      <div class="stat">
        <div class="stat-num">0ms</div>
        <div class="stat-label">Cold starts</div>
      </div>
      <div class="stat">
        <div class="stat-num">$0</div>
        <div class="stat-label">Transfer fees</div>
      </div>
      <div class="stat">
        <div class="stat-num">&lt;50ms</div>
        <div class="stat-label">Global latency</div>
      </div>
    </div>
  </div>
</section>`;

  /* ── Together (full-width trio) ─────────────────────────────────────── */
  const togetherSection = `
<section class="section" id="platform">
  <div class="section-pad">
    <div class="section-label">THE PLATFORM</div>
    <div class="section-heading">Three worlds. One home.</div>
  </div>
  <div class="trio">
    <div class="trio-card">
      <div class="trio-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4-4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
      </div>
      <div class="trio-name">Business</div>
      <p>Your team uploads, organizes, and shares files through a clean web interface. No training needed.</p>
    </div>
    <div class="trio-card">
      <div class="trio-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z"/></svg>
      </div>
      <div class="trio-name">AI</div>
      <p>Your AI assistants search, summarize, and organize those same files. Like a team member that never sleeps.</p>
    </div>
    <div class="trio-card">
      <div class="trio-icon">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
      </div>
      <div class="trio-name">Developers</div>
      <p>Your engineers build on a real API. Automate workflows, integrate systems, ship faster. <a href="/developers" class="subtle-link">Explore &rarr;</a></p>
    </div>
  </div>
</section>`;

  /* ── CTA ───────────────────────────────────────────────────────────── */
  const ctaSection = isSignedIn ? "" : `
<section class="section section--cta" id="get-started">
  <div class="glow-spot glow-spot--cta"></div>
  <div class="section-pad">
    <div class="cta-label"><span class="prompt-caret">&gt;</span> ready?</div>
    <div class="cta-title">Try it. It's free.</div>
    <div class="signin-card">
      <div class="signin-label"><span class="prompt-caret">&gt;</span> enter your email to start</div>
      <div class="prompt-form" id="signin-form">
        <span class="prompt-prefix">$</span>
        <input type="email" id="email-input" placeholder="you@example.com" autocomplete="email" spellcheck="false">
        <button id="signin-btn" onclick="signIn()">
          <span id="signin-text">Enter</span>
          <span id="signin-loading" style="display:none"><span class="spinner"></span></span>
        </button>
      </div>
      <div class="prompt-error" id="signin-error"></div>
      <div class="prompt-note">Magic link &middot; No password &middot; Free to start</div>
    </div>
  </div>
</section>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>storage.now &mdash; One place for your files</title>
<meta name="description" content="Where your team collaborates, AI assists, and developers build. File storage for business.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/home.css">
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
      <a href="/developers">developers</a>
      <a href="/docs">docs</a>
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

<div class="hero">
  <div class="glow-spot glow-spot--hero"></div>
  <div class="beam beam--center"></div>
  <div class="beam beam--left"></div>
  <div class="beam beam--right"></div>
  <div class="section-pad">
    ${isSignedIn ? heroSignedIn : heroSignedOut}
  </div>
</div>

${storeSection}
${shareSection}
${aiSection}
${statsSection}
${togetherSection}
${ctaSection}

</main>

<footer>
  <div class="section-pad">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/browse">browse</a>
      <a href="/developers">developers</a>
      <a href="/docs">docs</a>
      <a href="/pricing">pricing</a>
    </div>
  </div>
</footer>

<script>
/* Theme */
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();

/* Sign in */
async function signIn(){
  const input=document.getElementById('email-input');
  const btn=document.getElementById('signin-btn');
  const errEl=document.getElementById('signin-error');
  if(!input||!btn||!errEl) return;
  const email=input.value.trim();
  if(!email){errEl.textContent='email required';return}
  if(!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){errEl.textContent='invalid email';return}
  errEl.textContent='';
  btn.disabled=true;input.disabled=true;
  document.getElementById('signin-text').style.display='none';
  document.getElementById('signin-loading').style.display='inline-flex';
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();
    if(!res.ok) throw new Error(data.error?.message||'failed');
    if(data.magic_link){window.location.href=data.magic_link}
    else{document.getElementById('signin-form').innerHTML='<div class="prompt-success">check your inbox &mdash; magic link sent</div>'}
  }catch(err){
    errEl.textContent=err.message;btn.disabled=false;input.disabled=false;
    document.getElementById('signin-text').style.display='inline';
    document.getElementById('signin-loading').style.display='none';
  }
}
document.getElementById('email-input')?.addEventListener('keydown',e=>{if(e.key==='Enter')signIn()});

/* Scroll reveal */
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
