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
<p class="hero-hello">Welcome back, <strong>${displayName}</strong></p>
<div class="hero-actions">
  <a href="/browse" class="btn btn--primary btn--lg">Open your files</a>
  <a href="/api" class="btn btn--ghost btn--lg">API Reference</a>
</div>`;

  const heroSignedOut = `
<h1 class="hero-title">All your files.<br>One&nbsp;safe&nbsp;place.</h1>
<p class="hero-sub">Store anything. Find it instantly. Share it with one link.<br>Simple enough for everyone on your team.</p>
<div class="hero-ctas">
  <a href="#get-started" class="btn btn--primary btn--lg" onclick="document.getElementById('email-input')?.focus();return false">Start for free</a>
  <a href="/browse" class="btn btn--ghost btn--lg">See how it works</a>
</div>`;

  /* ── Store ────────────────────────────────────────────────────────── */
  const storeSection = `
<section class="section" id="store">
  <div class="section-pad section-center">
    <h2 class="section-headline">Drop it in.<br>It just works.</h2>
    <p class="section-body">PDFs, photos, spreadsheets, presentations &mdash; anything you need for work. Drag it in, and it&rsquo;s saved. Organized in folders you create, found the moment you need it.</p>
  </div>
  <div class="section-pad">
    <div class="window">
      <div class="window-bar">
        <div class="window-dots"><span></span><span></span><span></span></div>
        <div class="window-title">My Files</div>
      </div>
      <div class="window-body">
        <div class="file-row"><span class="file-icon file-icon--folder"></span><span class="file-name">Client Proposals</span><span class="file-meta">12 files</span></div>
        <div class="file-row"><span class="file-icon file-icon--folder"></span><span class="file-name">Q1 Reports</span><span class="file-meta">8 files</span></div>
        <div class="file-row"><span class="file-icon file-icon--pdf"></span><span class="file-name">brand-guidelines.pdf</span><span class="file-meta">4.2 MB</span></div>
        <div class="file-row"><span class="file-icon file-icon--sheet"></span><span class="file-name">budget-2026.xlsx</span><span class="file-meta">1.8 MB</span></div>
        <div class="file-row"><span class="file-icon file-icon--img"></span><span class="file-name">team-photo.jpg</span><span class="file-meta">5.1 MB</span></div>
        <div class="file-row"><span class="file-icon file-icon--folder"></span><span class="file-name">Contracts</span><span class="file-meta">15 files</span></div>
      </div>
    </div>
  </div>
</section>`;

  /* ── Share ─────────────────────────────────────────────────────────── */
  const shareSection = `
<section class="section" id="share">
  <div class="section-pad section-center">
    <h2 class="section-headline">Share with<br>one link.</h2>
    <p class="section-body">Click share, get a link, send it to anyone. They open it and see the file &mdash; no sign-up, no app to install, no confusion. It&rsquo;s that simple.</p>
  </div>
  <div class="section-pad">
    <div class="share-demo">
      <div class="share-step">
        <div class="step-num">1</div>
        <div class="step-text">Click &ldquo;Share&rdquo; on any file</div>
      </div>
      <div class="share-arrow">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
      </div>
      <div class="share-step">
        <div class="step-num">2</div>
        <div class="step-text">Copy the link</div>
      </div>
      <div class="share-arrow">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
      </div>
      <div class="share-step">
        <div class="step-num">3</div>
        <div class="step-text">Anyone can open it</div>
      </div>
    </div>
  </div>
</section>`;

  /* ── AI ────────────────────────────────────────────────────────────── */
  const aiSection = `
<section class="section" id="ai">
  <div class="section-pad section-center">
    <h2 class="section-headline">Ask your files<br>a question.</h2>
    <p class="section-body">Connect your favorite AI assistant &mdash; ChatGPT, Claude, or others. Ask it anything about your documents. It reads them for you and gives you answers in plain English.</p>
  </div>
  <div class="section-pad">
    <div class="chat-demo">
      <div class="chat-msg chat-human">
        <div class="chat-bubble">What was our revenue last quarter?</div>
      </div>
      <div class="chat-msg chat-ai">
        <div class="chat-bubble">I found Q4-2025-revenue.xlsx in your Reports folder. Revenue was $2.4M, up 18% from Q3. Want me to create a summary?</div>
      </div>
      <div class="chat-msg chat-human">
        <div class="chat-bubble">Yes, and sort my project files by client</div>
      </div>
      <div class="chat-msg chat-ai">
        <div class="chat-bubble">Done! Created 3 client folders: Acme Corp, Global Inc, and Nova Labs. Moved 28 files. I also saved the revenue summary as a new document in your Reports folder.</div>
      </div>
    </div>
  </div>
</section>`;

  /* ── Why section (value props) ────────────────────────────────────── */
  const whySection = `
<section class="section" id="why">
  <div class="section-pad section-center">
    <h2 class="section-headline">Built for the<br>way you work.</h2>
  </div>
  <div class="section-pad">
    <div class="values">
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <h3 class="value-title">Safe and sound</h3>
        <p class="value-desc">Your files are stored securely and backed up automatically. No more worrying about lost USB drives or crashed hard disks.</p>
      </div>
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        </div>
        <h3 class="value-title">Find anything fast</h3>
        <p class="value-desc">No more digging through folders. Just search by name, date, or type. Your file shows up before you finish typing.</p>
      </div>
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4-4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
        </div>
        <h3 class="value-title">Work together</h3>
        <p class="value-desc">Create shared spaces for your team. Everyone sees the same files, stays on the same page. No more emailing attachments back and forth.</p>
      </div>
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z"/></svg>
        </div>
        <h3 class="value-title">AI that helps</h3>
        <p class="value-desc">Your AI assistant can read, search, and organize your files. Like having a super-smart intern who never takes a break.</p>
      </div>
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
        </div>
        <h3 class="value-title">Works everywhere</h3>
        <p class="value-desc">Open it on your phone, tablet, or computer. No app to install &mdash; just open your browser and your files are right there.</p>
      </div>
      <div class="value-card">
        <div class="value-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
        </div>
        <h3 class="value-title">Developer friendly</h3>
        <p class="value-desc">If your team has developers, they&rsquo;ll love it too. A clean, simple interface for building custom tools on top of your files. <a href="/developers" class="subtle-link">Learn more &rarr;</a></p>
      </div>
    </div>
  </div>
</section>`;

  /* ── Social proof / numbers ───────────────────────────────────────── */
  const numbersSection = `
<section class="section" id="numbers">
  <div class="section-pad">
    <div class="numbers">
      <div class="number-item">
        <div class="number-big">Fast</div>
        <div class="number-label">Files load instantly, everywhere in the world</div>
      </div>
      <div class="number-item">
        <div class="number-big">Free</div>
        <div class="number-label">Get started at no cost. Upgrade when you need more</div>
      </div>
      <div class="number-item">
        <div class="number-big">Simple</div>
        <div class="number-label">If you can use email, you can use this</div>
      </div>
    </div>
  </div>
</section>`;

  /* ── CTA ───────────────────────────────────────────────────────────── */
  const ctaSection = isSignedIn ? "" : `
<section class="section section--cta" id="get-started">
  <div class="section-pad section-center">
    <h2 class="cta-headline">Ready to try it?</h2>
    <p class="cta-sub">Enter your email. We&rsquo;ll send you a link to get started.<br>No password to remember. No credit card.</p>
    <div class="signin-card">
      <div class="prompt-form" id="signin-form">
        <input type="email" id="email-input" placeholder="you@company.com" autocomplete="email" spellcheck="false">
        <button id="signin-btn" onclick="signIn()">
          <span id="signin-text">Get started</span>
          <span id="signin-loading" style="display:none"><span class="spinner"></span></span>
        </button>
      </div>
      <div class="prompt-error" id="signin-error"></div>
      <p class="cta-fine">Free to start &middot; No credit card &middot; Takes 30 seconds</p>
    </div>
  </div>
</section>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>storage.now &mdash; All your files. One safe place.</title>
<meta name="description" content="Store, share, and find your files in one simple place. Works with AI. Free to start.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/home.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo">storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/spaces">Spaces</a>
      <a href="/pricing">Pricing</a>
      <a href="/developers">Developers</a>
      <a href="/api">API</a>
    </div>
    <div class="nav-right">
      ${navSession}
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<main>

<section class="hero">
  ${isSignedIn ? heroSignedIn : heroSignedOut}
</section>

${storeSection}
${shareSection}
${aiSection}
${whySection}
${numbersSection}
${ctaSection}

</main>

<footer>
  <div class="footer-inner">
    <span class="footer-brand">storage.now</span>
    <div class="footer-links">
      <a href="/spaces">Spaces</a>
      <a href="/developers">Developers</a>
      <a href="/api">API</a>
      <a href="/pricing">Pricing</a>
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
  if(!email){errEl.textContent='Please enter your email';return}
  if(!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){errEl.textContent="That doesn't look like an email address";return}
  errEl.textContent='';
  btn.disabled=true;input.disabled=true;
  document.getElementById('signin-text').style.display='none';
  document.getElementById('signin-loading').style.display='inline-flex';
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();
    if(!res.ok) throw new Error(data.error?.message||'Something went wrong. Please try again.');
    if(data.magic_link){window.location.href=data.magic_link}
    else{document.getElementById('signin-form').innerHTML='<div class="prompt-success">Check your inbox &mdash; we sent you a link to sign in.</div>'}
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
  },{threshold:0.05,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
