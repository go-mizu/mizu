import { esc } from "./layout";
import homeMd from "../machine/home.md";
import { markdownToHtml } from "../machine/render";

/* ── Reusable SVG icons ───────────────────────────────────────────── */
const CLAUDE_ICON = (s: number) =>
  `<svg viewBox="0 0 24 24" width="${s}" height="${s}"><path fill="currentColor" d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>`;

const GPT_ICON = (s: number) =>
  `<svg viewBox="0 0 24 24" width="${s}" height="${s}"><path fill="currentColor" d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/></svg>`;

const ICON_FOLDER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>`;
const ICON_FILE = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;

const MOCKUP_CHROME = (url: string) =>
  `<div class="mockup-chrome"><div class="mockup-dots"><span></span><span></span><span></span></div><div class="mockup-url">${url}</div></div>`;

export function homePage(actor: string | null = null, siteKey?: string): string {
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
  <a href="/dashboard" class="btn btn--primary"><span class="btn-icon">&gt;_</span> Go to Dashboard</a>
</div>`;

  const heroSignedOut = `
<h1 class="hero-title">Your team's files.<br><span class="shimmer">Always within reach.</span></h1>
<p class="hero-sub">Store, share, and find files across your team. Connected to Claude and ChatGPT so your AI can work with your files too.</p>
<div class="hero-ctas">
  <a href="#" class="btn btn--primary btn--lg" onclick="openRegister();return false">Get started</a>
  <a href="/developers" class="btn btn--ghost btn--lg">See how it works</a>
</div>
`;

  /* ── Register modal ─────────────────────────────────────────────── */
  const registerModal = isSignedIn
    ? ""
    : `
<div class="register-modal" id="register-modal">
  <div class="register-modal-bg" onclick="closeRegister()"></div>
  <div class="register-modal-box">
    <button class="register-modal-close" onclick="closeRegister()">&times;</button>
    <div class="register-modal-title">Sign in</div>
    <p class="register-modal-sub">Enter your email to receive a sign-in link.</p>
    <div class="prompt-form" id="signin-form">
      <input type="email" id="email-input" placeholder="you@example.com" autocomplete="email" spellcheck="false">
      ${siteKey ? `<div id="turnstile-container" class="cf-turnstile" data-sitekey="${siteKey}" data-theme="dark" data-action="magic-link" style="margin:8px 0"></div>` : ''}
      <button id="signin-btn" onclick="signIn()">
        <span id="signin-text">Continue</span>
        <span id="signin-loading" style="display:none"><span class="spinner"></span></span>
      </button>
    </div>
    <div class="prompt-error" id="signin-error"></div>
    <div class="register-modal-note">Link expires in 15 minutes. New here? We'll create your account automatically.</div>
  </div>
</div>`;

  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Storage — Your team's files, always within reach</title>
<meta name="description" content="Store, share, and find files across your team. Connected to Claude and ChatGPT.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/base.css">
<link rel="stylesheet" href="/home.css">
${siteKey ? '<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>' : ''}
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
      <a href="/architecture">architecture</a>
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

<!-- ═══════════════════════════════════════════════════════════════════
     HERO
     ═══════════════════════════════════════════════════════════════════ -->
<section class="hero">
  <div class="hero-glow"></div>
  <div class="inner">
    ${isSignedIn ? heroSignedIn : heroSignedOut}
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     VALUE PILLARS — 3-column overview
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="pillars">
  <div class="inner">
    <div class="pillars">
      <div class="pillar">
        <div class="pillar-icon"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg></div>
        <h3 class="pillar-title">Store</h3>
        <p class="pillar-desc">Upload any file from your browser, CLI, or API. Organize with folders and search. Your files live in one place.</p>
      </div>
      <div class="pillar">
        <div class="pillar-icon"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg></div>
        <h3 class="pillar-title">Share</h3>
        <p class="pillar-desc">Generate a link. Send it to anyone. They open it and download. No account required. Links auto-expire when you want.</p>
      </div>
      <div class="pillar">
        <div class="pillar-icon"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg></div>
        <h3 class="pillar-title">AI-connected</h3>
        <p class="pillar-desc">Claude and ChatGPT can read, write, and share your files directly. Connect once, then just ask.</p>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     FEATURE 1 — File management
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="organize">
  <div class="inner">
    <div class="feature">
      <div class="feature-text">
        <div class="sec-label">ORGANIZE</div>
        <h2 class="sec-h">Everything in one place.</h2>
        <p class="feature-desc">Upload files from your browser or drag and drop. Create folders, move files, rename them. Search across everything. It works the way you expect a filesystem to work.</p>
        <div class="feature-facts">
          <div class="feature-fact"><strong>Drag and drop</strong> upload from any browser</div>
          <div class="feature-fact"><strong>Folders</strong> that nest as deep as you need</div>
          <div class="feature-fact"><strong>Search</strong> by file name across all your storage</div>
        </div>
      </div>
      <div class="feature-visual">
        <div class="term">
          ${MOCKUP_CHROME('storage.liteio.dev/browse/work')}
          <div class="mockup-body">
            <div class="mock-browser">
              <div class="mock-toolbar">
                <div class="mock-breadcrumb">files / <strong>work</strong> /</div>
                <div class="mock-toolbar-btn">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0018 9h-1.26A8 8 0 103 16.3"/></svg>
                  Upload
                </div>
              </div>
              <div class="mock-file-list">
                <div class="mock-file">
                  ${ICON_FOLDER}
                  <div class="mock-file-name mock-file-name--folder">invoices</div>
                  <div class="mock-file-meta">3 files</div>
                </div>
                <div class="mock-file">
                  ${ICON_FOLDER}
                  <div class="mock-file-name mock-file-name--folder">presentations</div>
                  <div class="mock-file-meta">7 files</div>
                </div>
                <div class="mock-file mock-file--highlight">
                  ${ICON_FILE}
                  <div class="mock-file-name">Q1-report.pdf</div>
                  <div class="mock-file-meta">2.4 MB &middot; just now</div>
                </div>
                <div class="mock-file">
                  ${ICON_FILE}
                  <div class="mock-file-name">team-budget.xlsx</div>
                  <div class="mock-file-meta">890 KB &middot; 2d ago</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     FEATURE 2 — Sharing
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="share">
  <div class="inner">
    <div class="feature feature--reverse">
      <div class="feature-text">
        <div class="sec-label">SHARE</div>
        <h2 class="sec-h">One link. No sign-up needed.</h2>
        <p class="feature-desc">Share any file with a single link. Your recipient clicks it and downloads. No account, no app install, no friction. Set expiry from one hour to seven days.</p>
        <div class="feature-facts">
          <div class="feature-fact"><strong>Time-limited links</strong> that auto-expire</div>
          <div class="feature-fact"><strong>No recipient sign-up</strong> required to download</div>
          <div class="feature-fact"><strong>Ask your AI</strong> to share files for you</div>
        </div>
      </div>
      <div class="feature-visual">
        <div class="share-flow">
          <div class="share-step">
            <div class="share-step-label">You share</div>
            <div class="term term--compact">
              <div class="mockup-body">
                <div class="mock-chat" style="gap:12px">
                  <div class="mock-chat-msg mock-chat-msg--user">
                    <div class="mock-chat-pill">Share Q1-report.pdf with my team</div>
                  </div>
                  <div class="mock-chat-msg mock-chat-msg--bot">
                    <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
                    <div class="mock-chat-text">Here's your link:<br><code>storage.liteio.dev/s/k7f2m</code><br>Expires in 24 hours.</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="share-arrow"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg></div>
          <div class="share-step">
            <div class="share-step-label">They download</div>
            <div class="term term--compact">
              <div class="mockup-body">
                <div class="mock-preview">
                  <div class="mock-preview-icon">
                    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                  </div>
                  <div class="mock-preview-name">Q1-report.pdf</div>
                  <div class="mock-preview-size">2.4 MB</div>
                  <div class="mock-preview-btn">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                    Download
                  </div>
                  <div class="mock-preview-note">No account needed</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     FEATURE 3 — AI integration
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="ai">
  <div class="inner">
    <div class="feature">
      <div class="feature-text">
        <div class="sec-label">AI INTEGRATION</div>
        <h2 class="sec-h">Your AI knows your files.</h2>
        <p class="feature-desc">Connect Storage to Claude or ChatGPT in under a minute. Then ask your AI to save files, find documents, organize folders, or share links with your team.</p>
        <div class="feature-facts">
          <div class="feature-fact"><strong>Claude</strong> via Settings &rarr; Integrations</div>
          <div class="feature-fact"><strong>ChatGPT</strong> via Settings &rarr; Connected apps</div>
          <div class="feature-fact"><strong>8 tools</strong> for read, write, search, share, and more</div>
        </div>
        <a href="/developers#ai" class="btn btn--ghost" style="margin-top:24px">Learn more &rarr;</a>
      </div>
      <div class="feature-visual">
        <div class="ai-demos">
          <div class="term">
            <div class="term-bar"><span class="term-dots"><i></i><i></i><i></i></span><span class="term-title">claude.ai</span></div>
            <div class="mockup-body">
              <div class="mock-chat">
                <div class="mock-chat-msg mock-chat-msg--user">
                  <div class="mock-chat-pill">What files do I have in /work?</div>
                </div>
                <div class="mock-chat-msg mock-chat-msg--bot">
                  <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
                  <div class="mock-chat-text">Your <strong>/work</strong> folder has 4 files:<br>&bull; invoices/ <em>3 files</em><br>&bull; Q1-report.pdf <em>2.4 MB</em><br>&bull; team-budget.xlsx <em>890 KB</em></div>
                </div>
                <div class="mock-chat-msg mock-chat-msg--user">
                  <div class="mock-chat-pill">Save this meeting summary as notes/standup-march-21.md</div>
                </div>
                <div class="mock-chat-msg mock-chat-msg--bot">
                  <div class="mock-chat-avatar">${CLAUDE_ICON(14)}</div>
                  <div class="mock-chat-text">Saved <strong>notes/standup-march-21.md</strong> (1.2 KB)</div>
                </div>
              </div>
            </div>
          </div>
          <div class="term">
            <div class="term-bar"><span class="term-dots"><i></i><i></i><i></i></span><span class="term-title">chatgpt.com</span></div>
            <div class="mockup-body">
              <div class="mock-chat">
                <div class="mock-chat-msg mock-chat-msg--user">
                  <div class="mock-chat-pill">Find all PDF files in my storage</div>
                </div>
                <div class="mock-chat-msg mock-chat-msg--bot">
                  <div class="mock-chat-avatar">${GPT_ICON(14)}</div>
                  <div class="mock-chat-text">Found 3 PDFs:<br>&bull; work/Q1-report.pdf <em>2.4 MB</em><br>&bull; work/invoices/inv-001.pdf <em>45 KB</em><br>&bull; work/invoices/inv-002.pdf <em>52 KB</em></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     SECURITY — trust strip
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec" id="security">
  <div class="inner">
    <div class="sec-label">SECURITY</div>
    <h2 class="sec-h">Secure by default.</h2>
  </div>
  <div class="inner">
    <div class="trust-grid">
      <div class="trust-item">
        <div class="trust-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg></div>
        <div>
          <strong>No passwords</strong>
          <p>Sign in with email magic links or Ed25519 keys. Nothing to leak.</p>
        </div>
      </div>
      <div class="trust-item">
        <div class="trust-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></div>
        <div>
          <strong>Encrypted at rest</strong>
          <p>Every file stored in encrypted object storage. TLS in transit.</p>
        </div>
      </div>
      <div class="trust-item">
        <div class="trust-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></div>
        <div>
          <strong>Auto-expiring links</strong>
          <p>Shared links expire on your schedule. 1 hour to 7 days.</p>
        </div>
      </div>
      <div class="trust-item">
        <div class="trust-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg></div>
        <div>
          <strong>Audit logging</strong>
          <p>Every action logged with actor, resource, and timestamp.</p>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- ═══════════════════════════════════════════════════════════════════
     CTA
     ═══════════════════════════════════════════════════════════════════ -->
<section class="sec sec--cta">
  <div class="inner cta-inner">
    <h2 class="cta-h">Start storing.</h2>
    <p class="cta-sub">Set up in 30 seconds.</p>
    <div class="cta-actions">
      ${isSignedIn
        ? '<a href="/browse" class="btn btn--primary">Go to my files</a>'
        : '<a href="#" class="btn btn--primary btn--lg" onclick="openRegister();return false">Get started</a>'}
      <a href="/pricing" class="btn btn--ghost">See pricing</a>
    </div>
  </div>
</section>

</div><!-- /human-view -->

${registerModal}

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button>${markdownToHtml(homeMd)}</div>
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

/* Sign in */
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
    const payload={email};
    const tsInput=document.querySelector('[name="cf-turnstile-response"]');
    if(tsInput&&tsInput.value)payload['cf-turnstile-response']=tsInput.value;
    const res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
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

/* Scroll reveal */
(function(){
  const els=document.querySelectorAll('.sec');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.06,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
