export function pricingPage(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pricing — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/pricing.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo">
      <span class="logo-dot"></span> storage.now
    </a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">browse</a>
      <a href="/developers">developers</a>
      <a href="/docs">docs</a>
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

<!-- ===== HERO ===== -->
<div class="hero">
  <div class="section-inner section-inner--center">
    <h1 class="hero-title">Simple, <span class="grad">predictable</span> pricing</h1>
    <p class="hero-sub">Free to start. Scale when you're ready.<br>Every plan includes unlimited downloads &mdash; zero bandwidth fees.</p>
  </div>
</div>

<!-- ===== TIERS ===== -->
<div class="section section--tiers">
  <div class="section-inner">
    <div class="tiers">

      <!-- FREE -->
      <div class="tier">
        <div class="tier-head">
          <div class="tier-name">Free</div>
          <div class="tier-price">$0</div>
          <div class="tier-period">forever</div>
        </div>
        <p class="tier-desc">For personal projects, experiments, and getting started with the API.</p>
        <a href="/" class="tier-cta">Get started</a>
        <div class="tier-list">
          <div class="tier-item">Upload and organize files</div>
          <div class="tier-item">Share files with other actors</div>
          <div class="tier-item">Web file browser</div>
          <div class="tier-item">Full REST API access</div>
          <div class="tier-item">Ed25519 agent authentication</div>
          <div class="tier-item">Magic link sign-in for humans</div>
        </div>
      </div>

      <!-- PRO -->
      <div class="tier tier--pro">
        <div class="tier-popular">Most popular</div>
        <div class="tier-head">
          <div class="tier-name">Pro</div>
          <div class="tier-price">$20<span class="tier-price-unit">/mo</span></div>
          <div class="tier-period">per account</div>
        </div>
        <p class="tier-desc">For production apps, multi-agent systems, and serious workloads.</p>
        <a href="/" class="tier-cta tier-cta--primary">Get started</a>
        <div class="tier-includes">Everything in Free, plus:</div>
        <div class="tier-list">
          <div class="tier-item tier-item--key">5x more storage</div>
          <div class="tier-item tier-item--key">Unlimited API requests</div>
          <div class="tier-item tier-item--key">Direct uploads &mdash; bypass the API</div>
          <div class="tier-item">Unlimited actors</div>
          <div class="tier-item">Larger file uploads</div>
          <div class="tier-item">Email support</div>
        </div>
      </div>

      <!-- MAX -->
      <div class="tier">
        <div class="tier-head">
          <div class="tier-name">Max</div>
          <div class="tier-price">$100<span class="tier-price-unit">/mo</span></div>
          <div class="tier-period">per account</div>
        </div>
        <p class="tier-desc">For teams and organizations that need more storage, more control, and dedicated support.</p>
        <a href="/" class="tier-cta">Get started</a>
        <div class="tier-includes">Everything in Pro, plus:</div>
        <div class="tier-list">
          <div class="tier-item tier-item--key">2x more storage than Pro</div>
          <div class="tier-item tier-item--key">Team sharing &amp; permissions</div>
          <div class="tier-item tier-item--key">Priority support</div>
          <div class="tier-item">Even larger file uploads</div>
          <div class="tier-item">Advanced usage analytics</div>
          <div class="tier-item">Early access to new features</div>
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
        <div class="inc-name">Zero bandwidth fees</div>
        <div class="inc-desc">Download your files as much as you want. No egress charges. No metering. No surprises.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
        </div>
        <div class="inc-name">Global edge delivery</div>
        <div class="inc-desc">Files served from 300+ locations worldwide. Sub-50ms metadata lookups.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
        </div>
        <div class="inc-name">Cryptographic auth</div>
        <div class="inc-desc">Ed25519 challenge-response for agents. Magic links for humans. No passwords.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
        </div>
        <div class="inc-name">Pure REST API</div>
        <div class="inc-desc">Works with curl, fetch, any HTTP client. No SDK required. Agent-first design.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
        </div>
        <div class="inc-name">Humans &amp; agents equal</div>
        <div class="inc-desc">Same API, same permissions model. Share files between humans and AI agents seamlessly.</div>
      </div>
      <div class="inc">
        <div class="inc-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div class="inc-name">Web file browser</div>
        <div class="inc-desc">Upload, browse, and manage files from any browser. Drag-and-drop support.</div>
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
        <div class="faq-q">How is bandwidth really free?</div>
        <div class="faq-a">It's the architecture, not a promotion. Our storage backend has zero egress fees built into the infrastructure. This isn't going away.</div>
      </div>
      <div class="faq">
        <div class="faq-q">What happens when I hit a limit?</div>
        <div class="faq-a">You'll get a clear error response. Your existing files stay accessible. Upgrade or free up space to continue uploading.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Can AI agents use the free plan?</div>
        <div class="faq-a">Absolutely. Register with an Ed25519 key and start storing files immediately. The free plan is designed for agents too.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Is there a per-seat charge?</div>
        <div class="faq-a">No. Pricing is per account, not per user or per agent. Add as many actors as your plan allows.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Can I switch plans anytime?</div>
        <div class="faq-a">Yes. Upgrade or downgrade at any time. Changes take effect immediately. No contracts, cancel anytime.</div>
      </div>
      <div class="faq">
        <div class="faq-q">Need more than Max?</div>
        <div class="faq-a">Get in touch. We offer custom plans for organizations with larger storage needs, SLAs, and dedicated support.</div>
      </div>
    </div>
  </div>
</div>

<!-- ===== CTA ===== -->
<div class="section section--cta">
  <div class="section-inner section-inner--center">
    <h2 class="cta-title">Start building today</h2>
    <p class="cta-sub">Free to start. No credit card required.</p>
    <a href="/" class="tier-cta tier-cta--primary cta-main">Get started free</a>
  </div>
</div>

<!-- Footer -->
<footer>
  <div class="section-inner">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/docs">docs</a>
      <a href="/pricing">pricing</a>
      <a href="/browse">browse</a>
    </div>
  </div>
</footer>

<script>
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
</script>
</body>
</html>`;
}
