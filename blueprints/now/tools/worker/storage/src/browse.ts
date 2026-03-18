import type { Context } from "hono";
import type { Env, Variables, ObjectRow } from "./types";
import { getSessionActor } from "./session";
import { esc, formatSize, relativeTime } from "./layout";
import { wildcardPath } from "./path";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

export async function browsePage(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) {
    return c.redirect("/", 302);
  }

  let prefix = wildcardPath(c, "/browse/");
  if (prefix && !prefix.endsWith("/")) prefix += "/";

  // List objects at this prefix
  const { results } = await c.env.DB.prepare(`
    SELECT id, path, name, is_folder, content_type, size, created_at, updated_at
    FROM objects
    WHERE owner = ? AND path LIKE ? AND path != ?
    ORDER BY is_folder DESC, name ASC
  `)
    .bind(actor, prefix + "%", prefix)
    .all<ObjectRow>();

  // Filter to direct children only
  const items = (results || []).filter((obj) => {
    const rest = obj.path.slice(prefix.length);
    if (obj.is_folder) {
      return rest.replace(/\/$/, "").indexOf("/") === -1;
    }
    return rest.indexOf("/") === -1;
  });

  // Storage stats
  const stats = await c.env.DB.prepare(
    "SELECT COUNT(*) as count, COALESCE(SUM(size),0) as total FROM objects WHERE owner = ? AND is_folder = 0",
  )
    .bind(actor)
    .first<{ count: number; total: number }>();

  const fileCount = stats?.count || 0;
  const totalSize = stats?.total || 0;
  const displayName = esc(actor.slice(2));

  // Breadcrumb
  const parts = prefix.replace(/\/$/, "").split("/").filter(Boolean);
  let breadcrumb = `<a href="/browse" class="crumb">~</a>`;
  let accumulated = "";
  for (const part of parts) {
    accumulated += part + "/";
    breadcrumb += ` <span class="crumb-sep">/</span> <a href="/browse/${accumulated}" class="crumb">${esc(part)}</a>`;
  }

  // File rows
  let rows = "";
  if (items.length === 0) {
    rows = `<div class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>
      <div>No files yet. Upload something or create a folder.</div>
    </div>`;
  } else {
    rows = `<div class="file-list">`;
    for (const item of items) {
      if (item.is_folder) {
        rows += `<a href="/browse/${esc(item.path)}" class="file-row file-row--folder">
  <span class="file-icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></span>
  <span class="file-name">${esc(item.name)}/</span>
  <span class="file-size">&mdash;</span>
  <span class="file-time">${relativeTime(item.updated_at)}</span>
  <span class="file-actions"></span>
</a>`;
      } else {
        rows += `<div class="file-row">
  <span class="file-icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></span>
  <span class="file-name">${esc(item.name)}</span>
  <span class="file-size">${formatSize(item.size)}</span>
  <span class="file-time">${relativeTime(item.updated_at)}</span>
  <span class="file-actions">
    <a href="#" onclick="downloadFile('${esc(item.path)}');return false" title="Download"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></a>
    <a href="#" onclick="deleteFile('${esc(item.path)}');return false" title="Delete"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg></a>
  </span>
</div>`;
      }
    }
    rows += `</div>`;
  }

  return c.html(`<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${prefix || "~"} — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">
</head>
<body>

<nav>
  <a href="/" class="logo">
    <span class="logo-dot"></span> storage.now
  </a>
  <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-links">
    <a href="/browse" class="active">browse</a>
    <a href="/docs">docs</a>
    <a href="/pricing">pricing</a>
  </div>
  <div class="nav-right">
    <span class="nav-user">${displayName}</span>
    <a href="/auth/logout" class="nav-signout">sign out</a>
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div class="browse-container">
  <div class="browse-header">
    <div class="breadcrumb">${breadcrumb}</div>
    <div class="browse-stats">${fileCount} file${fileCount !== 1 ? "s" : ""} · ${formatSize(totalSize)}</div>
  </div>

  <div class="browse-toolbar">
    <button class="tool-btn" onclick="document.getElementById('upload-input').click()">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
      Upload
    </button>
    <input type="file" id="upload-input" multiple style="display:none" onchange="uploadFiles(this.files)">
    <button class="tool-btn" onclick="promptFolder()">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/></svg>
      New folder
    </button>
  </div>

  <div class="drop-zone" id="drop-zone">
    <div class="drop-label">Drop files here to upload</div>
  </div>

  ${rows}
</div>

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

const currentPrefix = ${JSON.stringify(prefix)};

function getCookie(name) {
  const m = document.cookie.match('(?:^|;\\\\s*)' + name + '=([^;]+)');
  return m ? m[1] : '';
}

async function uploadFiles(files) {
  for (const file of files) {
    const path = currentPrefix + file.name;
    await fetch('/files/' + path, {
      method: 'PUT',
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
      body: file,
    });
  }
  location.reload();
}

async function downloadFile(path) {
  const res = await fetch('/files/' + path);
  if (!res.ok) { alert('Download failed'); return; }
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = path.split('/').pop();
  a.click();
  URL.revokeObjectURL(a.href);
}

async function deleteFile(path) {
  if (!confirm('Delete ' + path + '?')) return;
  await fetch('/files/' + path, { method: 'DELETE' });
  location.reload();
}

function promptFolder() {
  const name = prompt('Folder name:');
  if (!name) return;
  fetch('/folders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: currentPrefix + name }),
  }).then(() => location.reload());
}

// Drag and drop
const dz = document.getElementById('drop-zone');
const bc = document.querySelector('.browse-container');
let dragCount = 0;

bc.addEventListener('dragenter', (e) => {
  e.preventDefault();
  dragCount++;
  dz.classList.add('active');
});
bc.addEventListener('dragleave', (e) => {
  e.preventDefault();
  dragCount--;
  if (dragCount <= 0) { dz.classList.remove('active'); dragCount = 0; }
});
bc.addEventListener('dragover', (e) => e.preventDefault());
bc.addEventListener('drop', (e) => {
  e.preventDefault();
  dz.classList.remove('active');
  dragCount = 0;
  if (e.dataTransfer.files.length) uploadFiles(e.dataTransfer.files);
});
</script>
</body>
</html>`);
}
