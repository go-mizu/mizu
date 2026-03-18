import type { Context } from "hono";
import type { Env, Variables } from "./types";
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
   BROWSE LANDING (unauthenticated)
   ═══════════════════════════════════════════════════════════════════════ */
function browseLanding(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Browse — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">
</head>
<body>
<div class="grid-bg"></div>
<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  <div class="nav-search"><span class="search-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></span><input type="text" placeholder="Search files..." disabled></div>
  <div class="nav-right">
    <a href="/" class="nav-signout">sign in</a>
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div class="browse-landing">
  <div class="browse-hero">
    <div class="section-inner">
      <div class="browse-hero-badge">FILE BROWSER</div>
      <h1 class="browse-hero-title">Your files,<br><span class="grad">organized</span></h1>
      <p class="browse-hero-sub">A Google Drive-class file browser. Grid &amp; list views, drag-and-drop, search, sharing, trash &mdash; all from the edge.</p>
      <div class="browse-hero-actions">
        <a href="/" class="hero-btn hero-btn--primary">Get started free</a>
        <a href="/docs" class="hero-btn">Read the docs</a>
      </div>
      <div class="browse-stats-strip">
        <div class="browse-stat"><div class="browse-stat-val"><span class="grad">Grid + List</span></div><div class="browse-stat-label">View modes</div></div>
        <div class="browse-stat"><div class="browse-stat-val"><span class="grad">Drag &amp; Drop</span></div><div class="browse-stat-label">Upload &amp; move</div></div>
        <div class="browse-stat"><div class="browse-stat-val"><span class="grad">Search</span></div><div class="browse-stat-label">Find anything</div></div>
        <div class="browse-stat"><div class="browse-stat-val"><span class="grad">Share</span></div><div class="browse-stat-label">With anyone</div></div>
      </div>
    </div>
  </div>

  <div class="browse-features">
    <div class="section-inner">
      <div class="browse-features-heading">Everything you'd expect from a <span class="grad">modern drive</span></div>
      <div class="features-grid">
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg></div><div class="feature-card-name">Grid &amp; list views</div></div><div class="feature-card-desc">Switch between grid cards with thumbnails and a detailed list view with sortable columns.</div></div>
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></div><div class="feature-card-name">Search &amp; filter</div></div><div class="feature-card-desc">Find any file instantly. Filter by type, date, or starred status. Results as you type.</div></div>
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8"/><polyline points="16 6 12 2 8 6"/><line x1="12" y1="2" x2="12" y2="15"/></svg></div><div class="feature-card-name">Share with actors</div></div><div class="feature-card-desc">Grant read or write access to humans and AI agents. Same permission model for both.</div></div>
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg></div><div class="feature-card-name">Trash &amp; restore</div></div><div class="feature-card-desc">Accidentally deleted? No problem. Restore from trash anytime. Empty trash when you're sure.</div></div>
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg></div><div class="feature-card-name">Star favorites</div></div><div class="feature-card-desc">Star your important files for quick access. Your starred files are always one click away.</div></div>
        <div class="feature-card"><div class="feature-card-header"><div class="feature-card-icon"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg></div><div class="feature-card-name">Keyboard shortcuts</div></div><div class="feature-card-desc">Navigate, select, rename, delete — all from the keyboard. Press ? to see all shortcuts.</div></div>
      </div>
    </div>
  </div>

  <div class="browse-cta">
    <div class="section-inner">
      <div class="browse-cta-title">Ready to <span class="grad">organize</span>?</div>
      <p class="browse-cta-sub">Free to start. 5 GB storage. Zero bandwidth fees.</p>
      <a href="/" class="hero-btn hero-btn--primary">Get started free</a>
    </div>
  </div>
</div>

<div class="browse-footer">
  <div class="section-inner">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links"><a href="/docs">docs</a><a href="/pricing">pricing</a></div>
  </div>
</div>

<script>
function toggleTheme(){const d=document.documentElement.classList.toggle('dark');localStorage.setItem('theme',d?'dark':'light')}
(function(){const s=localStorage.getItem('theme');if(s==='light')document.documentElement.classList.remove('dark');
else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();
</script>
</body></html>`;
}

/* ═══════════════════════════════════════════════════════════════════════
   BROWSE APP (authenticated)
   ═══════════════════════════════════════════════════════════════════════ */
function browseApp(actor: string): string {
  const displayName = esc(actor.slice(2));
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Browse — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">
</head>
<body>

<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  <button class="mobile-toggle" id="mobile-toggle" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-search" id="nav-search">
    <span class="search-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></span>
    <input type="text" id="search-input" placeholder="Search files..." autocomplete="off" spellcheck="false">
  </div>
  <div class="nav-right">
    <span class="nav-user">${displayName}</span>
    <a href="/auth/logout" class="nav-signout">sign out</a>
    <button class="theme-toggle" id="theme-btn" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div id="app">
  <div class="sidebar-backdrop" id="sidebar-backdrop"></div>
  <aside id="sidebar">
    <div class="sidebar-nav" id="sidebar-nav"></div>
    <div class="sidebar-quota" id="quota"></div>
  </aside>

  <main id="main">
    <div id="toolbar"></div>
    <div id="bulk-bar"></div>
    <div class="file-content" id="file-content"></div>
    <div class="drop-overlay" id="drop-overlay">
      <div class="drop-overlay-icon"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg></div>
      <div class="drop-overlay-text">Drop files to upload</div>
      <div class="drop-overlay-sub">Files will be uploaded to the current folder</div>
    </div>
  </main>

  <aside id="detail-panel" class="hidden"></aside>
</div>

<div id="ctx-menu"></div>
<div id="modal-overlay"></div>
<div class="upload-panel" id="upload-panel">
  <div class="upload-header"><span class="upload-title" id="upload-title">Uploading...</span><button class="upload-close" id="upload-close">&times;</button></div>
  <div class="upload-list" id="upload-list"></div>
</div>
<div class="toast-container" id="toasts"></div>
<input type="file" id="file-input" multiple style="display:none">

<script>
(function(){
'use strict';
const ACTOR=${JSON.stringify(actor)};

/* ── Icons ────────────────────────────────────────────────────────── */
const I={
  folder:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>',
  file:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  image:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2"/></svg>',
  audio:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',
  doc:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>',
  code:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  archive:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="21 8 21 21 3 21 3 8"/><rect x="1" y="3" width="22" height="5"/><line x1="10" y1="12" x2="14" y2="12"/></svg>',
  sheet:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>',
  text:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
  star:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  starFill:'<svg viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  trash:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
  download:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  upload:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
  share:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>',
  rename:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
  move:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/><line x1="12" y1="11" x2="12" y2="17"/><polyline points="9 14 12 17 15 14"/></svg>',
  copy:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>',
  info:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
  x:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
  home:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>',
  shared:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
  clock:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  grid:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>',
  list:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  plus:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  restore:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 102.13-9.36L1 10"/></svg>',
  link:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>',
};

/* ── State ─────────────────────────────────────────────────────────── */
const S={
  path:'',view:localStorage.getItem('sv')||'list',
  sort:{col:localStorage.getItem('sc')||'name',asc:localStorage.getItem('sa')!=='0'},
  items:[],selected:new Set(),section:'files',
  detail:null,detailOpen:false,detailTab:'details',
  clipboard:null,stats:null,searchMode:false,searchQ:'',
  uploading:[]
};

const $=id=>document.getElementById(id);

/* ── Helpers ───────────────────────────────────────────────────────── */
function h(s){return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function fmtSize(b){if(!b)return '—';if(b<1024)return b+' B';if(b<1048576)return (b/1024).toFixed(1)+' KB';if(b<1073741824)return (b/1048576).toFixed(1)+' MB';return (b/1073741824).toFixed(1)+' GB'}
function fmtTime(ts){if(!ts)return '—';const d=Date.now()-ts,s=d/1000;if(s<60)return 'just now';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';if(s<2592000)return Math.floor(s/86400)+'d ago';return new Date(ts).toLocaleDateString()}

function fileType(item){
  if(item.is_folder)return 'folder';
  const ct=item.content_type||'';const n=item.name||'';
  if(ct.startsWith('image/'))return 'image';
  if(ct.startsWith('video/'))return 'video';
  if(ct.startsWith('audio/'))return 'audio';
  if(ct==='application/pdf'||ct.includes('document')||ct.includes('msword'))return 'doc';
  if(ct.includes('spreadsheet')||ct.includes('csv')||ct==='text/csv')return 'sheet';
  if(ct.includes('zip')||ct.includes('tar')||ct.includes('gzip')||ct.includes('compressed'))return 'archive';
  const ext=n.split('.').pop()?.toLowerCase()||'';
  if(['js','ts','py','go','rs','java','c','cpp','h','rb','php','sh','bash','yaml','yml','toml','json','xml','html','css','sql','md'].includes(ext))return 'code';
  if(['txt','log','text'].includes(ext)||ct.startsWith('text/'))return 'text';
  return 'generic';
}
function fileIconHtml(item){const t=fileType(item);return '<div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>'}

/* ── API ───────────────────────────────────────────────────────────── */
const api={
  get:u=>fetch(u).then(r=>r.json()),
  post:(u,b)=>fetch(u,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(r=>r.json()),
  patch:(u,b)=>fetch(u,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(r=>r.json()),
  del:u=>fetch(u,{method:'DELETE'}).then(r=>r.json()),
};

/* ── Toast ─────────────────────────────────────────────────────────── */
function toast(msg,type='info'){
  const el=document.createElement('div');
  el.className='toast toast--'+type;
  el.innerHTML='<span class="toast-msg">'+h(msg)+'</span>';
  $('toasts').appendChild(el);
  setTimeout(()=>{el.classList.add('fade-out');setTimeout(()=>el.remove(),300)},3000);
}

/* ── Render: Sidebar ──────────────────────────────────────────────── */
function renderSidebar(){
  const items=[
    {id:'files',icon:I.folder,label:'My Files'},
    {id:'shared',icon:I.shared,label:'Shared with me'},
    {id:'recent',icon:I.clock,label:'Recent'},
    {id:'starred',icon:I.star,label:'Starred'},
    {id:'trash',icon:I.trash,label:'Trash',badge:S.stats?.trash_count||0},
  ];
  let html='';
  for(const it of items){
    const active=S.section===it.id?' active':'';
    const badge=it.badge?'<span class="item-badge">'+it.badge+'</span>':'';
    html+='<div class="sidebar-item'+active+'" data-section="'+it.id+'"><span class="item-icon">'+it.icon+'</span><span class="item-label">'+it.label+'</span>'+badge+'</div>';
  }
  $('sidebar-nav').innerHTML=html;
}

function renderQuota(){
  if(!S.stats)return;
  const pct=Math.min(100,S.stats.total_size/S.stats.quota*100);
  const cls=pct>80?'danger':pct>50?'warn':'ok';
  $('quota').innerHTML='<div class="quota-bar"><div class="quota-fill quota-fill--'+cls+'" style="width:'+pct.toFixed(1)+'%"></div></div><div class="quota-text">'+fmtSize(S.stats.total_size)+' of '+fmtSize(S.stats.quota)+' used</div>';
}

/* ── Render: Toolbar ──────────────────────────────────────────────── */
function renderToolbar(){
  const parts=S.path.replace(/\\/$/,'').split('/').filter(Boolean);
  let bc='<span class="breadcrumb-home" onclick="B.nav(\\'\\')"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg></span>';
  let acc='';
  for(const p of parts){
    acc+=p+'/';
    const a=acc;
    bc+='<span class="breadcrumb-sep">/</span><span class="breadcrumb-segment" onclick="B.nav(\\''+h(a)+'\\')">'+h(p)+'</span>';
  }
  if(S.searchMode)bc='<span class="breadcrumb-segment">Search results for "'+h(S.searchQ)+'"</span>';

  const lActive=S.view==='list'?' active':'';
  const gActive=S.view==='grid'?' active':'';
  const sectionIsTrash=S.section==='trash';

  let right='';
  if(!S.searchMode&&S.section==='files'){
    right='<div class="toolbar-group"><button class="tool-btn'+lActive+'" onclick="B.setView(\\'list\\')" title="List view">'+I.list+'</button><button class="tool-btn'+gActive+'" onclick="B.setView(\\'grid\\')" title="Grid view">'+I.grid+'</button></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<div class="sort-dropdown" id="sort-dd"><button class="tool-btn" onclick="B.toggleSort()">Sort</button><div class="sort-menu" id="sort-menu"><div class="sort-option'+(S.sort.col==='name'?' active':'')+'" onclick="B.setSort(\\'name\\')">Name <span class="sort-dir">'+(S.sort.col==='name'?(S.sort.asc?'A→Z':'Z→A'):'')+'</span></div><div class="sort-option'+(S.sort.col==='updated_at'?' active':'')+'" onclick="B.setSort(\\'updated_at\\')">Modified <span class="sort-dir">'+(S.sort.col==='updated_at'?(S.sort.asc?'Old→New':'New→Old'):'')+'</span></div><div class="sort-option'+(S.sort.col==='size'?' active':'')+'" onclick="B.setSort(\\'size\\')">Size <span class="sort-dir">'+(S.sort.col==='size'?(S.sort.asc?'Small→Big':'Big→Small'):'')+'</span></div></div></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<button class="tool-btn" onclick="B.newFolder()" title="New folder">'+I.plus+' Folder</button>';
    right+='<button class="tool-btn tool-btn--primary" onclick="$(\\'file-input\\').click()" title="Upload">'+I.upload+' Upload</button>';
  }
  if(sectionIsTrash){
    right='<button class="tool-btn bulk-btn--danger" onclick="B.emptyTrash()">'+I.trash+' Empty trash</button>';
  }

  $('toolbar').innerHTML='<div class="breadcrumb">'+bc+'</div>'+right;
}

/* ── Render: Bulk bar ─────────────────────────────────────────────── */
function renderBulk(){
  const n=S.selected.size;
  const el=$('bulk-bar');
  if(!n){el.className='';el.innerHTML='';return}
  el.className='visible';
  const isTrash=S.section==='trash';
  let btns='';
  if(isTrash){
    btns='<button class="bulk-btn" onclick="B.bulkRestore()">'+I.restore+' Restore</button>';
  } else {
    btns='<button class="bulk-btn" onclick="B.bulkDownload()">'+I.download+' Download</button>';
    btns+='<button class="bulk-btn" onclick="B.bulkStar()">'+I.star+' Star</button>';
    btns+='<button class="bulk-btn" onclick="B.bulkMove()">'+I.move+' Move</button>';
    btns+='<button class="bulk-btn bulk-btn--danger" onclick="B.bulkTrash()">'+I.trash+' Trash</button>';
  }
  el.innerHTML='<span class="bulk-count">'+n+' selected</span><div class="bulk-actions">'+btns+'</div><button class="bulk-dismiss" onclick="B.clearSel()">'+I.x+'</button>';
}

/* ── Render: File list ────────────────────────────────────────────── */
function renderItems(){
  const fc=$('file-content');
  if(!S.items.length){
    const msgs={files:'This folder is empty',shared:'No files shared with you yet',recent:'No recently accessed files',starred:'No starred files',trash:'Trash is empty'};
    fc.innerHTML='<div class="empty-state"><div class="empty-icon"><svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></div><div class="empty-title">'+(msgs[S.section]||'No files')+'</div><div class="empty-sub">'+(S.section==='files'?'Upload files or create a folder to get started.':'')+'</div>'+(S.section==='files'?'<button class="empty-action" onclick="$(\\'file-input\\').click()">'+I.upload+' Upload files</button>':'')+'</div>';
    return;
  }
  if(S.view==='grid')renderGrid(fc);
  else renderList(fc);
}

function renderList(fc){
  const allSel=S.items.length>0&&S.items.every(i=>S.selected.has(i.path));
  let html='<div id="file-list"><div class="list-header"><input type="checkbox" class="row-check col--check"'+(allSel?' checked':'')+' onchange="B.selectAll(this.checked)"><span class="col col--icon"></span><span class="col col--name" onclick="B.setSort(\\'name\\')">Name'+(S.sort.col==='name'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span><span class="col col--modified" onclick="B.setSort(\\'updated_at\\')">Modified'+(S.sort.col==='updated_at'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span><span class="col col--size" onclick="B.setSort(\\'size\\')">Size'+(S.sort.col==='size'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span></div>';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const starCls=item.starred?' starred':'';
    html+='<div class="file-row'+(isF?' file-row--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><input type="checkbox" class="row-check col--check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)">'+fileIconHtml(item)+'<span class="file-name"><span class="file-name-text">'+h(item.name)+(isF?'/':'')+'</span><button class="file-star'+starCls+'" onclick="event.stopPropagation();B.star(\\''+h(item.path)+'\\','+(!item.starred?1:0)+')">'+(item.starred?I.starFill:I.star)+'</button></span><span class="file-modified">'+fmtTime(item.updated_at)+'</span><span class="file-size">'+fmtSize(item.is_folder?0:item.size)+'</span></div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  // Click handlers
  fc.querySelectorAll('.file-row').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.file-star'))return;
      const p=el.dataset.path;
      if(e.detail===2){// double click
        const item=S.items.find(i=>i.path===p);
        if(item?.is_folder)B.nav(p);
        else if(item)B.downloadFile(item);
        return;
      }
      B.clickSel(p,e);
    });
  });
}

function renderGrid(fc){
  let html='<div id="file-grid">';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const t=fileType(item);
    const starCls=item.starred?' starred':'';
    html+='<div class="grid-card'+(isF?' grid-card--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><div class="grid-check"><input type="checkbox" class="row-check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)"></div><button class="grid-star'+starCls+'" onclick="event.stopPropagation();B.star(\\''+h(item.path)+'\\','+(!item.starred?1:0)+')">'+(item.starred?I.starFill:I.star)+'</button><div class="grid-thumb"><div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div></div><div class="grid-name">'+h(item.name)+(isF?'/':'')+'</div></div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  fc.querySelectorAll('.grid-card').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.grid-star'))return;
      const p=el.dataset.path;
      if(e.detail===2){const item=S.items.find(i=>i.path===p);if(item?.is_folder)B.nav(p);else if(item)B.downloadFile(item);return}
      B.clickSel(p,e);
    });
  });
}

/* ── Render: Detail panel ─────────────────────────────────────────── */
function renderDetail(){
  const dp=$('detail-panel');
  if(!S.detailOpen||!S.detail){dp.className='hidden';return}
  dp.className='';
  const it=S.detail;
  const t=fileType(it);
  const detTab=S.detailTab==='details'?' active':'';
  const actTab=S.detailTab==='activity'?' active':'';
  let body='';
  if(S.detailTab==='details'){
    body='<div class="detail-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>';
    body+='<div class="detail-filename">'+h(it.name)+'</div>';
    body+='<div class="detail-type">'+(it.is_folder?'Folder':h(it.content_type||'Unknown'))+'</div>';
    if(!it.is_folder)body+='<div class="detail-field"><div class="detail-label">Size</div><div class="detail-value">'+fmtSize(it.size)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Location</div><div class="detail-value">'+h(it.path)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Created</div><div class="detail-value">'+fmtTime(it.created_at)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Modified</div><div class="detail-value">'+fmtTime(it.updated_at)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Description</div><textarea class="detail-description" placeholder="Add a description..." onblur="B.saveDesc(\\''+h(it.path)+'\\',this.value)">'+(it.description?h(it.description):'')+'</textarea></div>';
  } else {
    body='<div class="activity-list"><div class="activity-item"><div class="activity-avatar">U</div><div class="activity-content"><div class="activity-text"><strong>You</strong> uploaded this file</div><div class="activity-time">'+fmtTime(it.created_at)+'</div></div></div>';
    if(it.updated_at!==it.created_at)body+='<div class="activity-item"><div class="activity-avatar">U</div><div class="activity-content"><div class="activity-text"><strong>You</strong> modified this file</div><div class="activity-time">'+fmtTime(it.updated_at)+'</div></div></div>';
    body+='</div>';
  }
  dp.innerHTML='<div class="detail-header"><div class="detail-tabs"><span class="detail-tab'+detTab+'" onclick="B.detailTab(\\'details\\')">Details</span><span class="detail-tab'+actTab+'" onclick="B.detailTab(\\'activity\\')">Activity</span></div><button class="detail-close" onclick="B.closeDetail()">'+I.x+'</button></div><div class="detail-body">'+body+'</div>';
}

/* ── Render: Context menu ─────────────────────────────────────────── */
function renderCtx(x,y,item){
  const m=$('ctx-menu');
  let html='';
  if(!item){
    // Empty area context
    html+='<div class="ctx-item" onclick="B.newFolder()">'+I.plus+' New folder</div>';
    html+='<div class="ctx-item" onclick="$(\\'file-input\\').click()">'+I.upload+' Upload files</div>';
    if(S.clipboard){html+='<div class="ctx-divider"></div><div class="ctx-item" onclick="B.paste()">'+I.copy+' Paste</div>'}
    m.innerHTML=html;
  } else if(S.section==='trash'){
    html+='<div class="ctx-item" onclick="B.restoreItems([\\''+h(item.path)+'\\'])">'+I.restore+' Restore</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item ctx-item--danger" onclick="B.permDelete([\\''+h(item.path)+'\\'])">'+I.trash+' Delete forever</div>';
    m.innerHTML=html;
  } else {
    if(item.is_folder)html+='<div class="ctx-item" onclick="B.nav(\\''+h(item.path)+'\\')">'+I.folder+' Open</div>';
    else html+='<div class="ctx-item" onclick="B.downloadFile(S.items.find(i=>i.path===\\''+h(item.path)+'\\'))">'+I.download+' Download</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" onclick="B.startRename(\\''+h(item.path)+'\\')">'+I.rename+' Rename <span class="ctx-shortcut">F2</span></div>';
    html+='<div class="ctx-item" onclick="B.showMoveModal([\\''+h(item.path)+'\\'])">'+I.move+' Move to...</div>';
    if(!item.is_folder)html+='<div class="ctx-item" onclick="B.copyFile(\\''+h(item.path)+'\\')">'+I.copy+' Copy</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" onclick="B.showShareModal(\\''+h(item.path)+'\\')">'+I.share+' Share</div>';
    html+='<div class="ctx-item" onclick="B.star(\\''+h(item.path)+'\\','+(item.starred?0:1)+')">'+(!item.starred?I.star:I.starFill)+' '+(item.starred?'Unstar':'Star')+' <span class="ctx-shortcut">S</span></div>';
    html+='<div class="ctx-item" onclick="B.showDetail(\\''+h(item.path)+'\\')">'+I.info+' Details <span class="ctx-shortcut">I</span></div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item ctx-item--danger" onclick="B.trashItems([\\''+h(item.path)+'\\'])">'+I.trash+' Move to trash <span class="ctx-shortcut">Del</span></div>';
    m.innerHTML=html;
  }
  // Position
  const vw=window.innerWidth,vh=window.innerHeight;
  m.style.left=Math.min(x,vw-200)+'px';
  m.style.top=Math.min(y,vh-m.scrollHeight-10)+'px';
  m.classList.add('visible');
}

/* ── Data loading ─────────────────────────────────────────────────── */
async function loadItems(){
  S.selected.clear();
  let data;
  try{
    if(S.searchMode){
      data=await api.get('/drive/search?q='+encodeURIComponent(S.searchQ));
      S.items=data.items||[];
    } else if(S.section==='files'){
      data=await api.get('/folders/'+(S.path||''));
      S.items=data.items||[];
    } else if(S.section==='shared'){
      data=await api.get('/shared');
      S.items=(data.items||[]).map(i=>({...i,is_folder:false}));
    } else if(S.section==='recent'){
      data=await api.get('/drive/recent');
      S.items=data.items||[];
    } else if(S.section==='starred'){
      data=await api.get('/drive/starred');
      S.items=data.items||[];
    } else if(S.section==='trash'){
      data=await api.get('/drive/trash');
      S.items=data.items||[];
    }
  }catch(e){toast('Failed to load files','error');S.items=[]}
  sortItems();
  render();
}

async function loadStats(){
  try{S.stats=await api.get('/drive/stats')}catch{}
  renderSidebar();renderQuota();
}

function sortItems(){
  const {col,asc}=S.sort;
  S.items.sort((a,b)=>{
    // Folders first
    if(a.is_folder!==b.is_folder)return b.is_folder?1:-1;
    let v=0;
    if(col==='name')v=a.name.localeCompare(b.name);
    else if(col==='updated_at')v=(a.updated_at||0)-(b.updated_at||0);
    else if(col==='size')v=(a.size||0)-(b.size||0);
    return asc?v:-v;
  });
}

function render(){
  renderSidebar();renderToolbar();renderBulk();renderItems();renderDetail();renderQuota();
}

/* ── App controller (exposed as window.B) ─────────────────────────── */
const B=window.B={
  nav(path){
    S.path=path;S.section='files';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    history.pushState(null,'','/browse/'+(path||''));
    loadItems();
  },
  setSection(sec){
    S.section=sec;S.path='';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    // Close mobile sidebar
    $('sidebar').classList.remove('open');$('sidebar-backdrop').classList.remove('visible');
    loadItems();
  },
  setView(v){S.view=v;localStorage.setItem('sv',v);renderItems()},
  setSort(col){
    if(S.sort.col===col)S.sort.asc=!S.sort.asc;
    else{S.sort.col=col;S.sort.asc=true}
    localStorage.setItem('sc',col);localStorage.setItem('sa',S.sort.asc?'1':'0');
    sortItems();renderToolbar();renderItems();
  },
  toggleSort(){$('sort-dd')?.classList.toggle('open')},

  // Selection
  clickSel(path,e){
    if(e.ctrlKey||e.metaKey){
      if(S.selected.has(path))S.selected.delete(path);else S.selected.add(path);
    } else if(e.shiftKey&&S.selected.size){
      const paths=S.items.map(i=>i.path);
      const last=[...S.selected].pop();
      const a=paths.indexOf(last),b=paths.indexOf(path);
      const[lo,hi]=[Math.min(a,b),Math.max(a,b)];
      for(let i=lo;i<=hi;i++)S.selected.add(paths[i]);
    } else {
      S.selected.clear();S.selected.add(path);
    }
    // Show detail for single selection
    if(S.selected.size===1){
      S.detail=S.items.find(i=>i.path===[...S.selected][0])||null;
    }
    renderBulk();renderItems();renderDetail();
  },
  toggleSel(path,e){
    if(e.target.checked)S.selected.add(path);else S.selected.delete(path);
    renderBulk();renderItems();
  },
  selectAll(checked){
    if(checked)S.items.forEach(i=>S.selected.add(i.path));else S.selected.clear();
    renderBulk();renderItems();
  },
  clearSel(){S.selected.clear();renderBulk();renderItems()},

  // Context menu
  ctx(e,path){
    e.preventDefault();
    const item=S.items.find(i=>i.path===path)||null;
    if(item&&!S.selected.has(path)){S.selected.clear();S.selected.add(path);renderItems()}
    renderCtx(e.clientX,e.clientY,item);
  },

  // Star
  async star(path,v){
    await api.patch('/drive/star',{path,starred:v});
    const it=S.items.find(i=>i.path===path);
    if(it)it.starred=!!v;
    if(S.detail?.path===path)S.detail.starred=!!v;
    toast(v?'Starred':'Unstarred','success');
    renderItems();renderDetail();loadStats();
  },

  // Rename
  startRename(path){
    hideCtx();
    const item=S.items.find(i=>i.path===path);if(!item)return;
    const newName=prompt('Rename to:',item.name);
    if(!newName||newName===item.name)return;
    B.doRename(path,newName);
  },
  async doRename(path,newName){
    const res=await api.post('/drive/rename',{path,new_name:newName});
    if(res.error){toast(res.error.message||'Rename failed','error');return}
    toast('Renamed to '+newName,'success');loadItems();loadStats();
  },

  // Trash
  async trashItems(paths){
    hideCtx();
    await api.post('/drive/trash',{paths});
    toast(paths.length+' item(s) trashed','success');
    S.selected.clear();loadItems();loadStats();
  },
  async bulkTrash(){B.trashItems([...S.selected])},

  // Restore
  async restoreItems(paths){
    hideCtx();
    await api.post('/drive/restore',{paths});
    toast(paths.length+' item(s) restored','success');
    S.selected.clear();loadItems();loadStats();
  },
  async bulkRestore(){B.restoreItems([...S.selected])},

  // Empty trash
  async emptyTrash(){
    if(!confirm('Permanently delete all items in trash?'))return;
    await api.del('/drive/trash');
    toast('Trash emptied','success');loadItems();loadStats();
  },

  // Download
  downloadFile(item){
    if(!item||item.is_folder)return;
    const a=document.createElement('a');
    a.href='/files/'+item.path;a.download=item.name;a.click();
  },
  bulkDownload(){
    for(const p of S.selected){const it=S.items.find(i=>i.path===p);if(it&&!it.is_folder)B.downloadFile(it)}
  },

  // Star bulk
  async bulkStar(){
    for(const p of S.selected){await api.patch('/drive/star',{path:p,starred:1})}
    toast('Starred '+S.selected.size+' items','success');loadItems();
  },

  // Move
  showMoveModal(paths){
    hideCtx();
    showModal('Move to...','<div class="folder-tree" id="move-tree"><div class="tree-node selected" data-path="" onclick="B.selectMoveTarget(this,\\'\\')">/  (root)</div></div>','<button class="modal-btn modal-btn--secondary" onclick="hideModal()">Cancel</button><button class="modal-btn modal-btn--primary" onclick="B.doMove()">Move</button>');
    B._movePaths=paths;B._moveDest='';
    // Load folders
    api.get('/folders/').then(data=>{
      const tree=$('move-tree');
      if(!tree)return;
      let html='<div class="tree-node selected" data-path="" onclick="B.selectMoveTarget(this,\\'\\')">/  (root)</div>';
      for(const f of (data.items||[]).filter(i=>i.is_folder)){
        html+='<div class="tree-node" data-path="'+h(f.path)+'" onclick="B.selectMoveTarget(this,\\''+h(f.path)+'\\')"><span class="tree-icon">'+I.folder+'</span> '+h(f.name)+'</div>';
      }
      tree.innerHTML=html;
    });
  },
  selectMoveTarget(el,path){
    document.querySelectorAll('.tree-node').forEach(n=>n.classList.remove('selected'));
    el.classList.add('selected');B._moveDest=path;
  },
  async doMove(){
    hideModal();
    const res=await api.post('/drive/move',{paths:B._movePaths,destination:B._moveDest});
    if(res.error){toast(res.error.message||'Move failed','error');return}
    toast('Moved '+B._movePaths.length+' item(s)','success');
    S.selected.clear();loadItems();loadStats();
  },
  bulkMove(){B.showMoveModal([...S.selected])},

  // Copy
  async copyFile(path){
    hideCtx();
    const res=await api.post('/drive/copy',{path});
    if(res.error){toast(res.error.message||'Copy failed','error');return}
    toast('Copied as '+res.name,'success');loadItems();
  },

  // Share
  showShareModal(path){
    hideCtx();
    showModal('Share "'+h(path.split('/').pop())+'"','<div class="share-input-row"><input class="share-input" id="share-actor" placeholder="Actor name (e.g. a/agent-name)"><select class="share-perm-select" id="share-perm"><option value="read">Read</option><option value="write">Write</option></select><button class="modal-btn modal-btn--primary" onclick="B.doShare(\\''+h(path)+'\\')">Share</button></div><div class="share-list" id="share-list">Loading...</div><div style="margin-top:16px"><button class="copy-link-btn" onclick="B.copyLink(\\''+h(path)+'\\')">'+I.link+' Copy file path</button></div>','');
    B._sharePath=path;
    api.get('/shares').then(data=>{
      const list=$('share-list');if(!list)return;
      const given=(data.given||[]).filter(s=>s.path===path||s.object_path===path);
      if(!given.length){list.innerHTML='<div style="font-size:13px;color:var(--text-3)">Not shared with anyone yet</div>';return}
      list.innerHTML=given.map(s=>'<div class="share-entry"><div class="share-avatar">'+h((s.grantee||'?')[0].toUpperCase())+'</div><div class="share-name">'+h(s.grantee)+'</div><div class="share-role">'+h(s.permission)+'</div></div>').join('');
    });
  },
  async doShare(path){
    const actor=$('share-actor')?.value?.trim();
    const perm=$('share-perm')?.value;
    if(!actor){toast('Enter an actor name','error');return}
    const res=await api.post('/shares',{path,grantee:actor,permission:perm});
    if(res.error){toast(res.error.message||'Share failed','error');return}
    toast('Shared with '+actor,'success');
    B.showShareModal(path);// Refresh
  },
  copyLink(path){
    navigator.clipboard.writeText(window.location.origin+'/files/'+path);
    toast('Path copied to clipboard','success');
  },

  // Detail panel
  showDetail(path){
    hideCtx();
    S.detail=S.items.find(i=>i.path===path)||null;
    S.detailOpen=!!S.detail;S.detailTab='details';
    renderDetail();
  },
  closeDetail(){S.detailOpen=false;renderDetail()},
  detailTab(tab){S.detailTab=tab;renderDetail()},
  async saveDesc(path,desc){
    await api.patch('/drive/description',{path,description:desc});
  },

  // New folder
  async newFolder(){
    const name=prompt('Folder name:');if(!name)return;
    const res=await api.post('/folders',{path:S.path+name});
    if(res.error){toast(res.error.message||'Failed','error');return}
    toast('Folder created','success');loadItems();loadStats();
  },

  // Upload
  async uploadFiles(files){
    const panel=$('upload-panel');
    panel.classList.add('visible');
    const list=$('upload-list');
    S.uploading=[];
    for(const f of files){
      const entry={name:f.name,size:f.size,progress:0,status:'uploading',id:Math.random()};
      S.uploading.push(entry);
    }
    renderUploadList();
    $('upload-title').textContent='Uploading '+files.length+' file(s)...';

    for(let i=0;i<files.length;i++){
      const f=files[i];
      const path=S.path+f.name;
      try{
        await fetch('/files/'+path,{
          method:'PUT',
          headers:{'Content-Type':f.type||'application/octet-stream'},
          body:f,
        });
        S.uploading[i].status='done';S.uploading[i].progress=100;
      }catch{
        S.uploading[i].status='error';
      }
      renderUploadList();
    }
    $('upload-title').textContent='Upload complete';
    toast(files.length+' file(s) uploaded','success');
    loadItems();loadStats();
  },

  // Search
  search(q){
    S.searchQ=q;
    if(!q){S.searchMode=false;S.section='files';loadItems();return}
    S.searchMode=true;loadItems();
  },

  // Paste (for clipboard operations)
  async paste(){
    hideCtx();
    if(!S.clipboard)return;
    if(S.clipboard.action==='copy'){
      for(const p of S.clipboard.paths)await api.post('/drive/copy',{path:p,destination:S.path});
    } else {
      await api.post('/drive/move',{paths:S.clipboard.paths,destination:S.path});
    }
    S.clipboard=null;toast('Pasted','success');loadItems();
  },
};

/* ── Upload list render ───────────────────────────────────────────── */
function renderUploadList(){
  const list=$('upload-list');
  list.innerHTML=S.uploading.map(u=>{
    const st=u.status==='done'?'upload-item-status--done':u.status==='error'?'upload-item-status--error':'';
    const fill=u.status==='error'?'upload-item-fill--error':'';
    return '<div class="upload-item"><div class="upload-item-info"><div class="upload-item-name">'+h(u.name)+'</div><div class="upload-item-bar"><div class="upload-item-fill '+fill+'" style="width:'+(u.status==='done'?100:u.status==='error'?100:30)+'%"></div></div></div><span class="upload-item-status '+st+'">'+(u.status==='done'?'✓':u.status==='error'?'✗':'...')+'</span></div>';
  }).join('');
}

/* ── Modal helpers ────────────────────────────────────────────────── */
function showModal(title,body,footer){
  const m=$('modal-overlay');
  m.innerHTML='<div class="modal"><div class="modal-header"><span class="modal-title">'+title+'</span><button class="modal-close" onclick="hideModal()">'+I.x+'</button></div><div class="modal-body">'+body+'</div>'+(footer?'<div class="modal-footer">'+footer+'</div>':'')+'</div>';
  m.classList.add('visible');
}
function hideModal(){$('modal-overlay').classList.remove('visible');$('modal-overlay').innerHTML=''}
window.showModal=showModal;window.hideModal=hideModal;

/* ── Context menu helpers ─────────────────────────────────────────── */
function hideCtx(){$('ctx-menu').classList.remove('visible')}
document.addEventListener('click',()=>hideCtx());
document.addEventListener('click',()=>$('sort-dd')?.classList.remove('open'));

// Empty-area context menu
$('file-content').addEventListener('contextmenu',e=>{
  if(!e.target.closest('.file-row')&&!e.target.closest('.grid-card')){
    e.preventDefault();renderCtx(e.clientX,e.clientY,null);
  }
});

/* ── Keyboard shortcuts ───────────────────────────────────────────── */
document.addEventListener('keydown',e=>{
  const tag=e.target.tagName;
  if(tag==='INPUT'||tag==='TEXTAREA'||tag==='SELECT')return;

  if(e.key==='?'){e.preventDefault();showShortcuts();return}
  if(e.key==='/'||e.key==='f'&&(e.ctrlKey||e.metaKey)){e.preventDefault();$('search-input').focus();return}
  if(e.key==='Escape'){S.selected.clear();S.detailOpen=false;hideCtx();hideModal();render();return}

  const sel=[...S.selected];
  if(e.key==='Delete'||e.key==='Backspace'){if(sel.length&&S.section!=='trash'){B.trashItems(sel)}return}
  if(e.key==='F2'&&sel.length===1){B.startRename(sel[0]);return}
  if(e.key==='s'||e.key==='S'){if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it)B.star(sel[0],it.starred?0:1)}return}
  if(e.key==='i'||e.key==='I'){if(sel.length===1){B.showDetail(sel[0])}else{S.detailOpen=false;renderDetail()}return}
  if(e.key==='a'&&(e.ctrlKey||e.metaKey)){e.preventDefault();B.selectAll(true);return}
  if(e.key==='c'&&(e.ctrlKey||e.metaKey)){if(sel.length){S.clipboard={action:'copy',paths:sel};toast('Copied to clipboard','info')}return}
  if(e.key==='x'&&(e.ctrlKey||e.metaKey)){if(sel.length){S.clipboard={action:'cut',paths:sel};toast('Cut to clipboard','info')}return}
  if(e.key==='v'&&(e.ctrlKey||e.metaKey)){if(S.clipboard)B.paste();return}
  if(e.key==='Enter'){if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it?.is_folder)B.nav(it.path);else if(it)B.downloadFile(it)}return}

  // Arrow navigation
  if(e.key==='ArrowDown'||e.key==='ArrowUp'){
    e.preventDefault();
    const paths=S.items.map(i=>i.path);
    const cur=sel.length?paths.indexOf(sel[sel.length-1]):-1;
    const next=e.key==='ArrowDown'?Math.min(cur+1,paths.length-1):Math.max(cur-1,0);
    S.selected.clear();S.selected.add(paths[next]);
    S.detail=S.items[next];
    renderBulk();renderItems();renderDetail();
    // Scroll into view
    document.querySelector('.file-row.selected,.grid-card.selected')?.scrollIntoView({block:'nearest'});
    return;
  }
});

function showShortcuts(){
  const shortcuts=[
    ['?','Show shortcuts'],['/', 'Search'],['Esc','Deselect / Close'],['↑ ↓','Navigate'],
    ['Enter','Open'],['Del','Trash'],['F2','Rename'],['S','Star/Unstar'],
    ['I','Details panel'],['⌘A','Select all'],['⌘C','Copy'],['⌘X','Cut'],
    ['⌘V','Paste'],['',''],
  ];
  const grid=shortcuts.map(([k,d])=>'<div class="shortcut-item"><span class="shortcut-desc">'+d+'</span><span class="shortcut-keys"><span class="kbd">'+k+'</span></span></div>').join('');
  showModal('Keyboard Shortcuts','<div class="shortcuts-grid">'+grid+'</div>','');
}

/* ── Search ────────────────────────────────────────────────────────── */
let searchTimer=null;
$('search-input').addEventListener('input',e=>{
  clearTimeout(searchTimer);
  searchTimer=setTimeout(()=>B.search(e.target.value),300);
});
$('search-input').addEventListener('keydown',e=>{if(e.key==='Escape'){e.target.value='';B.search('')}});

/* ── Drag and drop ────────────────────────────────────────────────── */
let dragCount=0;
const main=$('main');
main.addEventListener('dragenter',e=>{e.preventDefault();dragCount++;$('drop-overlay').classList.add('active')});
main.addEventListener('dragleave',e=>{e.preventDefault();dragCount--;if(dragCount<=0){$('drop-overlay').classList.remove('active');dragCount=0}});
main.addEventListener('dragover',e=>e.preventDefault());
main.addEventListener('drop',e=>{
  e.preventDefault();$('drop-overlay').classList.remove('active');dragCount=0;
  if(e.dataTransfer?.files?.length)B.uploadFiles(e.dataTransfer.files);
});

/* ── File input ───────────────────────────────────────────────────── */
$('file-input').addEventListener('change',e=>{if(e.target.files.length)B.uploadFiles(e.target.files);e.target.value=''});

/* ── Upload panel close ───────────────────────────────────────────── */
$('upload-close').addEventListener('click',()=>$('upload-panel').classList.remove('visible'));

/* ── Sidebar clicks ───────────────────────────────────────────────── */
$('sidebar-nav').addEventListener('click',e=>{
  const el=e.target.closest('.sidebar-item');
  if(el)B.setSection(el.dataset.section);
});

/* ── Mobile sidebar toggle ────────────────────────────────────────── */
$('mobile-toggle')?.addEventListener('click',()=>{
  $('sidebar').classList.toggle('open');
  $('sidebar-backdrop').classList.toggle('visible');
});
$('sidebar-backdrop').addEventListener('click',()=>{
  $('sidebar').classList.remove('open');
  $('sidebar-backdrop').classList.remove('visible');
});

/* ── Theme toggle ─────────────────────────────────────────────────── */
$('theme-btn').addEventListener('click',()=>{
  const d=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',d?'dark':'light');
});
(function(){const s=localStorage.getItem('theme');
  if(s==='light')document.documentElement.classList.remove('dark');
  else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();

/* ── History ──────────────────────────────────────────────────────── */
window.addEventListener('popstate',()=>{
  const p=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
  S.path=p;S.section='files';S.searchMode=false;loadItems();
});

/* ── Init ─────────────────────────────────────────────────────────── */
const initPath=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
S.path=initPath==='browse'?'':initPath;
loadItems();loadStats();

})();
</script>
</body></html>`;
}
