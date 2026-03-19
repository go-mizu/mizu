import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { getSessionActor } from "./session";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export async function browsePage(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.html(browseLanding());
  return c.html(browseApp(actor));
}

/* ═══════════════════════════════════════════════════════════════════════
   Shared HTML fragments
   ═══════════════════════════════════════════════════════════════════════ */

const HEAD = `<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">`;

const SEARCH_BAR = `<div class="nav-search" id="nav-search">
    <span class="search-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></span>
    <input type="text" id="search-input" placeholder="Search files..." autocomplete="off" spellcheck="false">
  </div>`;

const THEME_BTN = `<button class="theme-toggle" id="theme-btn" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>`;

const MOBILE_TOGGLE = `<button class="mobile-toggle" id="mobile-toggle" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>`;

const APP_SHELL = `<div id="app">
  <div class="sidebar-backdrop" id="sidebar-backdrop"></div>
  <aside id="sidebar">
    <div class="sidebar-nav" id="sidebar-nav"></div>
    <div class="sidebar-quota" id="quota"></div>
  </aside>

  <main id="main">
    <div id="toolbar"></div>
    <div id="bulk-bar"></div>
    <div class="file-content" id="file-content"></div>`;

const OVERLAYS = `<div id="ctx-menu"></div>
<div id="modal-overlay"></div>
<div id="preview-overlay"></div>`;

/* ═══════════════════════════════════════════════════════════════════════
   DEMO (unauthenticated) — interactive demo with sample data
   ═══════════════════════════════════════════════════════════════════════ */
function browseLanding(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
${HEAD}
<title>Demo — storage.now</title>
</head>
<body>
<div class="grid-bg"></div>

<div class="demo-banner" id="demo-banner">
  <span class="demo-banner-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
  <span>You're exploring a <strong>live demo</strong> — this is what your drive looks like. <a href="/" class="demo-banner-link">Sign up free</a> to create your own.</span>
  <button class="demo-banner-close" onclick="document.getElementById('demo-banner').remove()">&times;</button>
</div>

<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  ${MOBILE_TOGGLE}
  ${SEARCH_BAR}
  <div class="nav-right">
    <span class="nav-user" style="opacity:.6">demo</span>
    <a href="/" class="nav-signout">sign up</a>
    ${THEME_BTN}
  </div>
</nav>

${APP_SHELL}
  </main>

  <aside id="detail-panel" class="hidden"></aside>
</div>

${OVERLAYS}
<div class="toast-container" id="toasts"></div>

<script>window.__BROWSE_CONFIG={mode:'demo'}</script>
<script src="/browse.js"></script>
</body></html>`;
}

/* ═══════════════════════════════════════════════════════════════════════
   AUTH (logged-in) — real file browser with uploads, drag-drop, etc.
   ═══════════════════════════════════════════════════════════════════════ */
function browseApp(actor: string): string {
  const displayName = esc(actor.slice(2));
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
${HEAD}
<title>Browse — storage.now</title>
</head>
<body>
<div class="grid-bg"></div>

<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  ${MOBILE_TOGGLE}
  ${SEARCH_BAR}
  <div class="nav-right">
    <span class="nav-user">${displayName}</span>
    <a href="/auth/logout" class="nav-signout">sign out</a>
    ${THEME_BTN}
  </div>
</nav>

${APP_SHELL}
    <div class="drop-overlay" id="drop-overlay">
      <div class="drop-overlay-icon"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg></div>
      <div class="drop-overlay-text">Drop files to upload</div>
      <div class="drop-overlay-sub">Files will be uploaded to the current folder</div>
    </div>
  </main>

  <aside id="detail-panel" class="hidden"></aside>
</div>

${OVERLAYS}
<div class="upload-panel" id="upload-panel">
  <div class="upload-header"><span class="upload-title" id="upload-title">Uploading...</span><button class="upload-close" id="upload-close">&times;</button></div>
  <div class="upload-list" id="upload-list"></div>
</div>
<div class="toast-container" id="toasts"></div>
<input type="file" id="file-input" multiple style="display:none">

<script>window.__BROWSE_CONFIG={mode:'auth',actor:${JSON.stringify(actor)}}</script>
<script src="/browse.js"></script>
</body></html>`;
}
