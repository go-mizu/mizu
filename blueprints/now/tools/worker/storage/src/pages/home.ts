import { esc } from "./layout";

/* ── Reusable SVG icons ───────────────────────────────────────────── */
const CLAUDE_ICON = (s: number) =>
  `<svg viewBox="0 0 24 24" width="${s}" height="${s}"><path fill="currentColor" d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>`;

const GPT_ICON = (s: number) =>
  `<svg viewBox="0 0 24 24" width="${s}" height="${s}"><path fill="currentColor" d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/></svg>`;

const MOCKUP_CHROME = (url: string) =>
  `<div class="mockup-chrome"><div class="mockup-dots"><span></span><span></span><span></span></div><div class="mockup-url">${url}</div></div>`;

/* ── Folder icon (16px) ───────────────────────────────────────────── */
const ICON_FOLDER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>`;
const ICON_FILE = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;

export function homePage(actor: string | null = null): string {
  const isSignedIn = actor !== null;
  const displayName = actor ? esc(actor.slice(2)) : "";

  const navSession = isSignedIn
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  /* ── Hero variants ──────────────────────────────────────────────── */
  const heroSignedIn = `
<div class="hero-greeting">Welcome back, <strong>${displayName}</strong></div>
<div class="hero-actions">
  <a href="/browse" class="btn btn--primary"><span class="btn-icon">&gt;_</span> Go to my files</a>
</div>`;

  const heroSignedOut = `
<h1 class="hero-title">A home for<br><span class="shimmer">your files.</span></h1>
<p class="hero-sub">Store files. Use them in Claude and ChatGPT. Share with anyone.</p>
<div class="hero-ctas">
  <a href="#" class="btn btn--primary btn--lg" onclick="openRegister();return false">Get started</a>
</div>`;

  /* ── Register modal ─────────────────────────────────────────────── */
  const registerModal = isSignedIn
    ? ""
    : `
<div class="register-modal" id="register-modal">
  <div class="register-modal-bg" onclick="closeRegister()"></div>
  <div class="register-modal-box">
    <button class="register-modal-close" onclick="closeRegister()">&times;</button>
    <div class="register-modal-title">Get started</div>
    <p class="register-modal-sub">Enter your email. We'll send you a sign-in link.</p>
    <div class="prompt-form" id="signin-form">
      <input type="email" id="email-input" placeholder="you@email.com" autocomplete="email" spellcheck="false">
      <button id="signin-btn" onclick="signIn()">
        <span id="signin-text">Go</span>
        <span id="signin-loading" style="display:none"><span class="spinner"></span></span>
      </button>
    </div>
    <div class="prompt-error" id="signin-error"></div>
    <div class="register-modal-note">No password &middot; No credit card</div>
  </div>
</div>`;

  /* ── Bottom CTA (signed-out only) ───────────────────────────────── */
  const bottomCta = isSignedIn
    ? ""
    : `
<div class="bottom-cta" id="bottom-cta">
  <div class="bottom-cta-title">Ready to try?</div>
  <p class="bottom-cta-sub">No credit card needed.</p>
  <div class="prompt-form" id="bottom-form">
    <input type="email" placeholder="you@email.com" autocomplete="email" spellcheck="false" id="bottom-email">
    <button onclick="signInBottom()">Go</button>
  </div>
  <div class="prompt-error" id="bottom-error"></div>
  <div class="bottom-cta-note">No password &middot; We email you a link</div>
</div>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Storage — A home for your files</title>
<meta name="description" content="Store files, use them in Claude and ChatGPT, share with anyone.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/home.css">
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
      <a href="/cli">cli</a>
      <a href="/pricing">pricing</a>
    </div>
    <div class="nav-right">
      ${navSession}
      ${!isSignedIn ? '<button class="nav-register" onclick="openRegister()">Sign in</button>' : ''}
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<!-- ===== HUMAN VIEW ===== -->
<div class="human-view" id="human-view">

<!-- HERO -->
<section class="hero">
  <div class="hero-glow"></div>
  <div class="section-inner">
    ${isSignedIn ? heroSignedIn : heroSignedOut}
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     JOURNEY — each section is one action with one screenshot
     ═══════════════════════════════════════════════════════════════════ -->
<div class="journey">

<!-- STEP 1: Create an account -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">01</div>
    <div class="journey-title">Create an account</div>
    <div class="journey-desc">Enter your email. We send you a link. Click it and you're in. No password needed.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('storage.now')}
    <div class="mockup-body">
      <div class="mock-signup">
        <div class="mock-signup-title">Get started</div>
        <div class="mock-input">
          <div class="mock-input-field">you@email.com</div>
          <div class="mock-input-btn">Go</div>
        </div>
        <div class="mock-note">No password &middot; No credit card</div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 2: Upload a file -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">02</div>
    <div class="journey-title">Upload a file</div>
    <div class="journey-desc">Drag a file into your browser and drop it. Or click upload. Done.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('storage.now/browse')}
    <div class="mockup-body">
      <div class="mock-browser">
        <div class="mock-toolbar">
          <div class="mock-breadcrumb">files /</div>
          <div class="mock-toolbar-btn">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0018 9h-1.26A8 8 0 103 16.3"/></svg>
            Upload
          </div>
        </div>
        <div class="mock-file-list">
          <div class="mock-file mock-file--highlight">
            ${ICON_FILE}
            <div class="mock-file-name">quarterly-report.pdf</div>
            <div class="mock-file-meta">2.4 MB &middot; just now</div>
          </div>
          <div class="mock-file">
            ${ICON_FILE}
            <div class="mock-file-name">vacation-photos.zip</div>
            <div class="mock-file-meta">48 MB &middot; 2d ago</div>
          </div>
        </div>
        <div class="mock-file--drop">drop files here to upload</div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 3: Organize into folders -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">03</div>
    <div class="journey-title">Organize into folders</div>
    <div class="journey-desc">Make folders, move things around, rename stuff. It works just like your computer.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('storage.now/browse/work')}
    <div class="mockup-body">
      <div class="mock-browser">
        <div class="mock-toolbar">
          <div class="mock-breadcrumb">files / <strong>work</strong> /</div>
          <div class="mock-toolbar-btn">
            ${ICON_FOLDER}
            New folder
          </div>
        </div>
        <div class="mock-file-list">
          <div class="mock-file">
            ${ICON_FOLDER}
            <div class="mock-file-name mock-file-name--folder">invoices</div>
            <div class="mock-file-meta">3 files</div>
          </div>
          <div class="mock-file">
            ${ICON_FILE}
            <div class="mock-file-name">quarterly-report.pdf</div>
            <div class="mock-file-meta">2.4 MB</div>
          </div>
          <div class="mock-file">
            ${ICON_FILE}
            <div class="mock-file-name">meeting-notes.docx</div>
            <div class="mock-file-meta">340 KB</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 4: Connect to Claude -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">04</div>
    <div class="journey-title">Connect to Claude</div>
    <div class="journey-desc">Open Claude, go to Settings &rarr; Integrations &rarr; Add. Paste the URL and click Add.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('claude.ai/settings')}
    <div class="mockup-body">
      <div class="mock-connector">
        <div class="mock-connector-header">
          <span class="mock-connector-title">Add custom connector</span>
          <span class="mock-connector-badge">BETA</span>
        </div>
        <div class="mock-connector-sub">Connect Claude to your data and tools.</div>
        <div class="mock-connector-field">
          <div class="mock-connector-value">Storage</div>
        </div>
        <div class="mock-connector-field">
          <div class="mock-connector-value mock-connector-value--url">https://storage.liteio.dev/mcp</div>
        </div>
        <div class="mock-connector-actions">
          <div class="mock-connector-cancel">Cancel</div>
          <div class="mock-connector-add">Add</div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 5: Chat with Claude -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">05</div>
    <div class="journey-title">Chat with Claude</div>
    <div class="journey-desc">Just ask Claude about your files. It can save, find, list, and share them for you.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('claude.ai')}
    <div class="mockup-body">
      <div class="mock-chat">
        <div class="mock-chat-msg mock-chat-msg--user">
          <div class="mock-chat-pill">What files do I have?</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--bot">
          <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
          <div class="mock-chat-text">Here are your files:<br>&bull; <strong>quarterly-report.pdf</strong> <em>2.4 MB</em><br>&bull; <strong>vacation-photos.zip</strong> <em>48 MB</em><br>&bull; <strong>meeting-notes.docx</strong> <em>340 KB</em></div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--user">
          <div class="mock-chat-pill">Move the report into my work folder</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--bot">
          <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
          <div class="mock-chat-text">Done! Moved <strong>quarterly-report.pdf</strong> to <code>/work</code>.</div>
        </div>
        <div class="mock-chat-bar">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          <span>Message Claude...</span>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 6: Connect to ChatGPT -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">06</div>
    <div class="journey-title">Connect to ChatGPT</div>
    <div class="journey-desc">Same thing. Open ChatGPT, go to Settings, click Connected apps, search for Storage.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('chatgpt.com/settings')}
    <div class="mockup-body">
      <div class="mock-settings">
        <div class="mock-settings-header">
          <div class="mock-settings-icon">${GPT_ICON(22)}</div>
          <div class="mock-settings-name">Settings</div>
        </div>
        <div class="mock-settings-section">Connected apps</div>
        <div class="mock-settings-row mock-settings-row--active">
          <div class="mock-settings-row-icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="2" width="20" height="20"/><line x1="7" y1="2" x2="7" y2="22"/></svg></div>
          <div class="mock-settings-row-text"><strong>Storage</strong> &middot; File storage</div>
        </div>
        <div class="mock-url-input">
          <div class="mock-url-field">https://storage.liteio.dev/mcp</div>
          <div class="mock-url-btn">Connect</div>
        </div>
        <div class="mock-status mock-status--ok"><span class="mock-status-dot"></span> Connected</div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 7: Chat with ChatGPT -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">07</div>
    <div class="journey-title">Chat with ChatGPT</div>
    <div class="journey-desc">Same thing works in ChatGPT. Save files, ask questions, it all just works.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('chatgpt.com')}
    <div class="mockup-body">
      <div class="mock-chat">
        <div class="mock-chat-msg mock-chat-msg--user">
          <div class="mock-chat-pill">Save this as budget-2025.xlsx</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--bot">
          <div class="mock-chat-avatar">${GPT_ICON(14)}</div>
          <div class="mock-chat-text">Done! Saved <strong>budget-2025.xlsx</strong> to your storage.</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--user">
          <div class="mock-chat-pill">What's in my work folder?</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--bot">
          <div class="mock-chat-avatar">${GPT_ICON(14)}</div>
          <div class="mock-chat-text">Your <code>/work</code> folder has:<br>&bull; <strong>quarterly-report.pdf</strong><br>&bull; <strong>meeting-notes.docx</strong><br>&bull; <strong>budget-2025.xlsx</strong></div>
        </div>
        <div class="mock-chat-bar">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
          <span>Message ChatGPT...</span>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 8: Share a file -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">08</div>
    <div class="journey-title">Share a file</div>
    <div class="journey-desc">Ask your AI to share a file. It gives you a link you can send to anyone.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('claude.ai')}
    <div class="mockup-body">
      <div class="mock-chat">
        <div class="mock-chat-msg mock-chat-msg--user">
          <div class="mock-chat-pill">Share quarterly-report.pdf with my team</div>
        </div>
        <div class="mock-chat-msg mock-chat-msg--bot">
          <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
          <div class="mock-chat-text">Here's your sharing link:<br><br><code>storage.now/p/k7f2m</code><br><br>Anyone with this link can view and download the file.</div>
        </div>
        <div class="mock-chat-bar">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          <span>Message Claude...</span>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- STEP 9: Your friend opens it -->
<div class="journey-step">
  <div class="journey-text">
    <div class="journey-num">09</div>
    <div class="journey-title">Your friend opens it</div>
    <div class="journey-desc">They click the link. They see the file. They can download it. No sign-up needed.</div>
  </div>
  <div class="journey-mockup">
    ${MOCKUP_CHROME('storage.now/p/k7f2m')}
    <div class="mockup-body">
      <div class="mock-preview">
        <div class="mock-preview-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div class="mock-preview-name">quarterly-report.pdf</div>
        <div class="mock-preview-size">2.4 MB</div>
        <div class="mock-preview-btn">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          Download
        </div>
        <div class="mock-preview-note">Shared via Storage &middot; No account needed</div>
      </div>
    </div>
  </div>
</div>

</div><!-- /journey -->

<!-- BOTTOM CTA -->
${bottomCta}

</div><!-- /human-view -->

${registerModal}

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button><span class="h1"># Storage</span>

A home for your files. Store, organize, share. Works with Claude and ChatGPT.

<span class="h2">## How it works</span>

<span class="h3">### 1. Create an account</span>
Enter your email, click the link we send. No password.

<span class="h3">### 2. Upload files</span>
Drag and drop into your browser, or ask your AI to save them.

<span class="h3">### 3. Organize</span>
Create folders, move files, rename things. Just like your computer.

<span class="h3">### 4. Connect your AI</span>
Claude: Settings > Integrations > Add custom connector > paste https://storage.liteio.dev/mcp
ChatGPT: Settings > Connected apps > Add by URL > paste https://storage.liteio.dev/mcp

<span class="h3">### 5. Chat with your AI</span>
Ask it to save, find, list, move, or share files. It just works.

<span class="h3">### 6. Share files</span>
Ask your AI or click share. Get a link. Send it to anyone.
They click it and download. No sign-up needed.

<span class="h2">## Quick facts</span>

Downloads:     Always free, no limits
Speed:         Servers in 300+ locations
Sign-in:       Email link (no password)
Works on:      Any device with a browser
AI:            Claude, ChatGPT

<span class="h2">## Links</span>

<span class="link">https://storage.liteio.dev/ai</span>        Set up AI
<span class="link">https://storage.liteio.dev/pricing</span>   Pricing
<span class="link">https://storage.liteio.dev/browse</span>    Your files</div>
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
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();

/* Mode switch */
function setMode(mode){
  const btns=document.querySelectorAll('.mode-switch button');
  btns.forEach(b=>b.classList.remove('active'));
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
  const el=document.getElementById('md-content');
  const text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(()=>{
    const btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}

/* Register modal */
function openRegister(){
  const m=document.getElementById('register-modal');
  if(m){m.classList.add('open');setTimeout(()=>document.getElementById('email-input')?.focus(),100)}
}
function closeRegister(){
  const m=document.getElementById('register-modal');
  if(m){m.classList.remove('open')}
}
document.addEventListener('keydown',e=>{if(e.key==='Escape')closeRegister()});

/* Sign in (modal) */
async function signIn(){
  const input=document.getElementById('email-input');
  const btn=document.getElementById('signin-btn');
  const errEl=document.getElementById('signin-error');
  if(!input||!btn||!errEl) return;
  const email=input.value.trim();
  if(!email){errEl.textContent='Please enter your email address';return}
  if(!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){errEl.textContent='That doesn\\'t look like an email address';return}
  errEl.textContent='';
  btn.disabled=true;input.disabled=true;
  document.getElementById('signin-text').style.display='none';
  document.getElementById('signin-loading').style.display='inline-flex';
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();
    if(!res.ok) throw new Error(data.message||'Something went wrong. Please try again.');
    document.getElementById('signin-form').innerHTML='<div class="prompt-success">Check your inbox! We sent you a sign-in link.</div>'
  }catch(err){
    errEl.textContent=err.message;btn.disabled=false;input.disabled=false;
    document.getElementById('signin-text').style.display='inline';
    document.getElementById('signin-loading').style.display='none';
  }
}
document.getElementById('email-input')?.addEventListener('keydown',e=>{if(e.key==='Enter')signIn()});

/* Sign in (bottom CTA) */
async function signInBottom(){
  const input=document.getElementById('bottom-email');
  const errEl=document.getElementById('bottom-error');
  if(!input||!errEl) return;
  const email=input.value.trim();
  if(!email){errEl.textContent='Please enter your email address';return}
  if(!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){errEl.textContent='That doesn\\'t look like an email address';return}
  errEl.textContent='';
  input.disabled=true;
  try{
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email})});
    const data=await res.json();
    if(!res.ok) throw new Error(data.message||'Something went wrong.');
    document.getElementById('bottom-form').innerHTML='<div class="prompt-success">Check your inbox!</div>'
  }catch(err){
    errEl.textContent=err.message;input.disabled=false;
  }
}
document.getElementById('bottom-email')?.addEventListener('keydown',e=>{if(e.key==='Enter')signInBottom()});

/* Scroll reveal */
(function(){
  const els=document.querySelectorAll('.journey-step,.bottom-cta');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.08,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
