import pricingMd from "../machine/pricing.md";
import { markdownToHtml } from "../machine/render";

export function pricingPage(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pricing — Storage</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/base.css">
<link rel="stylesheet" href="/pricing.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo">
      <span class="logo-dot"></span> Storage
    </a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/developers">developers</a>
      <a href="/api">api</a>
      <a href="/cli">cli</a>
      <a href="/architecture">architecture</a>
      <a href="/pricing" class="active">pricing</a>
    </div>
    <div class="nav-right">
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<!-- ===== HUMAN VIEW ===== -->
<div class="human-view" id="human-view">

<!-- ===== HERO ===== -->
<div class="hero">
  <div class="section-inner section-inner--center">
    <h1 class="hero-title">Simple, <span class="grad">predictable</span> pricing</h1>
    <p class="hero-sub">Start free, scale when you're ready.<br>No egress fees. No bandwidth metering.</p>
  </div>
</div>

<!-- ===== TIERS ===== -->
<div class="section section--tiers">
  <div class="section-inner">
    <div class="tiers">

      <!-- FREE -->
      <div class="tier">
        <div class="tier-badge"></div>
        <div class="tier-head">
          <div class="tier-name">Free</div>
          <div class="tier-price">$0</div>
          <div class="tier-period">no credit card</div>
        </div>
        <p class="tier-desc">For personal projects and getting started.</p>
        <a href="/" class="tier-cta">Get started</a>
        <div class="tier-list">
          <div class="tier-item">Upload and organize files</div>
          <div class="tier-item">Share files with anyone</div>
          <div class="tier-item">Browse and manage in your browser</div>
          <div class="tier-item">API access for your apps</div>
          <div class="tier-item">Passwordless sign-in</div>
        </div>
      </div>

      <!-- PRO -->
      <div class="tier tier--pro">
        <div class="tier-badge tier-popular">Most popular</div>
        <div class="tier-head">
          <div class="tier-name">Pro</div>
          <div class="tier-price">$20<span class="tier-price-unit">/mo</span></div>
          <div class="tier-period">per account</div>
        </div>
        <p class="tier-desc">For production workloads and growing teams.</p>
        <a href="/" class="tier-cta tier-cta--primary">Get started</a>
        <div class="tier-includes">Everything in Free, plus:</div>
        <div class="tier-list">
          <div class="tier-item tier-item--key">More storage and larger files</div>
          <div class="tier-item tier-item--key">Faster uploads, direct to storage</div>
          <div class="tier-item">More users and agents</div>
          <div class="tier-item">Higher usage limits</div>
          <div class="tier-item">Email support</div>
        </div>
      </div>

      <!-- MAX -->
      <div class="tier">
        <div class="tier-badge"></div>
        <div class="tier-head">
          <div class="tier-name">Max</div>
          <div class="tier-price">$100<span class="tier-price-unit">/mo</span></div>
          <div class="tier-period">per account</div>
        </div>
        <p class="tier-desc">For teams that need collaboration and support.</p>
        <a href="/" class="tier-cta">Get started</a>
        <div class="tier-includes">Everything in Pro, plus:</div>
        <div class="tier-list">
          <div class="tier-item tier-item--key">Team sharing &amp; permissions</div>
          <div class="tier-item tier-item--key">Support for large files</div>
          <div class="tier-item">More storage and users</div>
          <div class="tier-item">Usage insights</div>
          <div class="tier-item">Priority support</div>
        </div>
      </div>

    </div>
  </div>
</div>

<!-- ===== EVERY PLAN INCLUDES ===== -->
<div class="section section--included">
  <div class="section-inner">
    <h2 class="inc-title">Every plan includes</h2>
    <div class="inc-grid">
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <div class="inc-name">No egress fees</div>
        <div class="inc-desc">Read and download your files without bandwidth charges. Built into the infrastructure, not a promotional offer.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
        </div>
        <div class="inc-name">Global edge network</div>
        <div class="inc-desc">Files served from 300+ edge locations. Sub-50ms metadata lookups worldwide.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
        </div>
        <div class="inc-name">Passwordless auth</div>
        <div class="inc-desc">Ed25519 challenge-response for agents. Magic links for humans. No passwords to manage or leak.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
        </div>
        <div class="inc-name">Plain REST API</div>
        <div class="inc-desc">Works with curl, fetch, or any HTTP client. No proprietary SDK. Standard HTTP verbs and JSON.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
        </div>
        <div class="inc-name">Humans &amp; agents</div>
        <div class="inc-desc">Same API for both. Share files between people and AI agents with one permissions model.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div class="inc-name">Web file browser</div>
        <div class="inc-desc">Upload, browse, preview, and manage files from any browser. Drag-and-drop included.</div>
      </div>
    </div>
  </div>
</div>

<!-- ===== FAQ ===== -->
<div class="section section--faq">
  <div class="section-inner">
    <h2 class="faq-title">Frequently asked questions</h2>
    <div class="faq-list">
      <div class="faq">
        <div class="faq-q">Why are there no egress fees?</div>
        <div class="faq-a">Our storage backend (Cloudflare R2) doesn't charge for egress. We pass that through. This is structural, not a promotion that will expire.</div>
      </div>
      <div class="faq">
        <div class="faq-q">What counts as an "actor"?</div>
        <div class="faq-a">Each human user or AI agent identity on your account is one actor. Higher plans support more actors for larger teams and agent fleets.</div>
      </div>
      <div class="faq">
        <div class="faq-q">What happens when I hit a storage or rate limit?</div>
        <div class="faq-a">You'll get a clear error with the specific limit. Existing files stay accessible. Upgrade or free up space to resume uploads.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Can AI agents use the free plan?</div>
        <div class="faq-a">Yes. Register with an Ed25519 key and start storing files immediately. Agents are first-class citizens on every plan.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Can I switch plans anytime?</div>
        <div class="faq-a">Yes. Upgrade or downgrade at any time. Changes take effect immediately. No long-term contracts.</div>
      </div>
      <div class="faq">
        <div class="faq-q">What if I need more than what Max offers?</div>
        <div class="faq-a">Contact us. We offer custom arrangements for organizations with larger capacity, SLA requirements, or dedicated support needs.</div>
      </div>
    </div>
  </div>
</div>

<!-- ===== CTA ===== -->
<div class="section section--cta">
  <div class="section-inner section-inner--center">
    <h2 class="cta-title">Start building today</h2>
    <p class="cta-sub">Free plan. No credit card required.</p>
    <a href="/" class="tier-cta tier-cta--primary cta-main">Get started</a>
  </div>
</div>

</div><!-- /human-view -->

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button>${markdownToHtml(pricingMd)}</div>
</div>

<!-- Floating mode switch -->
<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

<script>
/* Theme */
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light'){
    document.documentElement.classList.remove('dark');
  } else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches){
    document.documentElement.classList.remove('dark');
  }
})();

/* Mode switch */
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

/* Machine view copy */
function copyMd(){
  var el=document.getElementById('md-content');
  var text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(function(){
    var btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(function(){btn.textContent='copy'},2000);
  });
}
</script>
</body>
</html>`;
}
