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

const THEME_BTN = `<button class="hdr-theme" id="theme-btn" aria-label="Toggle theme">
  <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
  <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
</button>`;

/* ═══════════════════════════════════════════════════════════════════════
   DEMO (unauthenticated)
   ═══════════════════════════════════════════════════════════════════════ */
function browseLanding(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
${HEAD}
<title>Demo — storage.now</title>
</head>
<body>
<div class="dot-bg"></div>

<header class="hdr">
  <a href="/" class="hdr-logo"><span class="dot"></span>storage.now</a>
  <div class="hdr-end">
    <kbd class="hdr-kbd" id="search-trigger">/</kbd>
    <span class="hdr-tag">demo</span>
    <a href="/" class="hdr-link">sign up</a>
    ${THEME_BTN}
  </div>
</header>

<main class="main" id="main"></main>

<div id="cmd-palette"></div>
<div id="ctx-menu"></div>
<div id="modal-root"></div>
<div class="toast-wrap" id="toasts"></div>

<script>window.__BROWSE_CONFIG={mode:'demo'}</script>
<script src="/browse.js"></script>
</body></html>`;
}

/* ═══════════════════════════════════════════════════════════════════════
   AUTH (logged-in)
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
<div class="dot-bg"></div>

<header class="hdr">
  <a href="/" class="hdr-logo"><span class="dot"></span>storage.now</a>
  <div class="hdr-end">
    <kbd class="hdr-kbd" id="search-trigger">/</kbd>
    <span class="hdr-user">${displayName}</span>
    <a href="/auth/logout" class="hdr-link">sign out</a>
    ${THEME_BTN}
  </div>
</header>

<main class="main" id="main"></main>

<div id="cmd-palette"></div>
<div id="ctx-menu"></div>
<div id="modal-root"></div>
<div class="upload-panel" id="upload-panel">
  <div class="upload-head"><span id="upload-title">Uploading...</span><button id="upload-close">&times;</button></div>
  <div class="upload-list" id="upload-list"></div>
</div>
<div class="toast-wrap" id="toasts"></div>
<input type="file" id="file-input" multiple style="display:none">

<script>window.__BROWSE_CONFIG={mode:'auth',actor:${JSON.stringify(actor)}}</script>
<script src="/browse.js"></script>
</body></html>`;
}
