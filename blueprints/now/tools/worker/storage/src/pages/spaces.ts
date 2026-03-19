import { esc } from "./layout";

interface SpaceRow {
  id: string; owner: string; title: string; description: string;
  visibility: string; updated_at: number; item_count: number;
  member_count: number; top_members: string | null;
}

/**
 * /spaces — Space listing page. Server-side rendered cards (no loading state).
 */
export async function spacesPage(actor: string | null, db?: D1Database): Promise<string> {
  return actor && db ? spacesGrid(actor, db) : spacesMarketing();
}

/* ── Authenticated: Card grid (server-rendered) ────────────────────────── */
async function spacesGrid(actor: string, db: D1Database): Promise<string> {
  const displayName = esc(actor.slice(2));

  const { results: owned } = await db.prepare(`
    SELECT s.*,
      (SELECT COUNT(*) FROM space_items WHERE space_id = s.id) AS item_count,
      (SELECT COUNT(*) FROM space_members WHERE space_id = s.id) + 1 AS member_count,
      (SELECT GROUP_CONCAT(actor) FROM (SELECT actor FROM space_members WHERE space_id = s.id ORDER BY created_at ASC LIMIT 4)) AS top_members
    FROM spaces s
    WHERE s.owner = ?
    ORDER BY s.updated_at DESC
  `).bind(actor).all<SpaceRow>();

  const { results: memberOf } = await db.prepare(`
    SELECT s.*,
      sm.role AS my_role,
      (SELECT COUNT(*) FROM space_items WHERE space_id = s.id) AS item_count,
      (SELECT COUNT(*) FROM space_members WHERE space_id = s.id) + 1 AS member_count,
      (SELECT GROUP_CONCAT(actor) FROM (SELECT actor FROM space_members WHERE space_id = s.id ORDER BY created_at ASC LIMIT 4)) AS top_members
    FROM space_members sm
    JOIN spaces s ON sm.space_id = s.id
    WHERE sm.actor = ?
    ORDER BY s.updated_at DESC
  `).bind(actor).all<SpaceRow>();

  const all = [...(owned || []), ...(memberOf || [])];
  const seen = new Set<string>();
  const spaces = all.filter(s => { if (seen.has(s.id)) return false; seen.add(s.id); return true; });

  const now = Date.now();
  const cardsHtml = spaces.length
    ? spaces.map(s => renderCard(s, now)).join("")
    : `<div class="grid-empty">No spaces yet. Create your first one.</div>`;

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Spaces — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/spaces.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">Browse</a>
      <a href="/spaces" class="active">Spaces</a>
      <a href="/developers">Developers</a>
      <a href="/api">API</a>
      <a href="/pricing">Pricing</a>
      <a href="/ai">AI</a>
    </div>
    <div class="nav-right">
      <span class="nav-user">${displayName}</span>
      <a href="/auth/logout" class="nav-signout">sign out</a>
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<main>
  <div class="page-container">
    <div class="page-header">
      <h1 class="page-title">Spaces</h1>
      <button class="btn-new" id="btn-new" onclick="openCreate()">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        New Space
      </button>
    </div>
    <div class="spaces-grid" id="spaces-grid">
      ${cardsHtml}
    </div>
  </div>
</main>

<!-- Create Modal -->
<div class="modal-overlay" id="modal" style="display:none">
  <div class="modal">
    <div class="modal-header">
      <h3>Create Space</h3>
      <button class="modal-close" onclick="closeCreate()">&times;</button>
    </div>
    <div class="modal-body">
      <label class="form-label">Name</label>
      <input type="text" id="inp-name" class="form-input" placeholder="e.g., Project Alpha" autocomplete="off">
      <label class="form-label">Description</label>
      <textarea id="inp-desc" class="form-textarea" rows="2" placeholder="What is this space for?"></textarea>
      <label class="form-label">Visibility</label>
      <div class="vis-row">
        <label class="vis-opt"><input type="radio" name="vis" value="private" checked><span>Private</span></label>
        <label class="vis-opt"><input type="radio" name="vis" value="team"><span>Team</span></label>
        <label class="vis-opt"><input type="radio" name="vis" value="public"><span>Public</span></label>
      </div>
      <div class="form-error" id="form-err"></div>
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeCreate()">Cancel</button>
      <button class="btn-create" id="btn-create" onclick="doCreate()">Create</button>
    </div>
  </div>
</div>

<script>
function toggleTheme(){
  var isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  var saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches))
    document.documentElement.classList.add('dark');
})();

function openCreate(){document.getElementById('modal').style.display='flex';document.getElementById('inp-name').focus()}
function closeCreate(){document.getElementById('modal').style.display='none';document.getElementById('inp-name').value='';document.getElementById('inp-desc').value='';document.getElementById('form-err').textContent=''}

async function doCreate(){
  var name=document.getElementById('inp-name').value.trim();
  var desc=document.getElementById('inp-desc').value.trim();
  var vis=document.querySelector('input[name="vis"]:checked').value;
  var err=document.getElementById('form-err');
  if(!name){err.textContent='Name required';return}
  var btn=document.getElementById('btn-create');
  btn.disabled=true;btn.textContent='Creating...';err.textContent='';
  try{
    var res=await fetch('/spaces',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({title:name,description:desc,visibility:vis})});
    var data=await res.json();
    if(!res.ok)throw new Error(data.error&&data.error.message||'Failed');
    closeCreate();
    window.location.href='/space/'+data.id;
  }catch(e){err.textContent=e.message;btn.disabled=false;btn.textContent='Create'}
}
</script>
</body>
</html>`;
}

function renderCard(s: SpaceRow, now: number): string {
  const avatars: string[] = [s.owner];
  if (s.top_members) {
    for (const m of s.top_members.split(",")) {
      if (m && !avatars.includes(m)) avatars.push(m);
    }
  }
  const shown = avatars.slice(0, 4);
  const extra = s.member_count - shown.length;

  const avHtml = shown.map(a => {
    const isAgent = a.startsWith("a/");
    return `<span class="card-avatar${isAgent ? " card-avatar--agent" : ""}">${isAgent ? "AI" : initials(a)}</span>`;
  }).join("") + (extra > 0 ? `<span class="card-avatar-more">+${extra}</span>` : "");

  return `<a class="space-card" href="/space/${esc(s.id)}">` +
    `<div class="card-title">${esc(s.title)}</div>` +
    `<div class="card-desc">${s.description ? esc(s.description) : ""}</div>` +
    `<div class="card-meta">` +
      `<span class="card-vis">${esc(s.visibility)}</span>` +
      `<span>${s.item_count || 0} items</span>` +
      `<span>Updated ${relTime(s.updated_at, now)}</span>` +
    `</div>` +
    `<div class="card-avatars">${avHtml}</div>` +
  `</a>`;
}

function initials(name: string): string {
  const clean = name.replace(/^[ua]\//, "");
  const parts = clean.split(/[\s@._-]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return clean.slice(0, 2).toUpperCase();
}

function relTime(ms: number, now: number): string {
  if (!ms) return "\u2014";
  const d = now - ms;
  if (d < 60000) return "just now";
  if (d < 3600000) return Math.floor(d / 60000) + "m ago";
  if (d < 86400000) return Math.floor(d / 3600000) + "h ago";
  if (d < 604800000) return Math.floor(d / 86400000) + "d ago";
  return new Date(ms).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function spacesMarketing(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Spaces — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/spaces.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">Browse</a>
      <a href="/spaces" class="active">Spaces</a>
      <a href="/developers">Developers</a>
      <a href="/api">API</a>
      <a href="/pricing">Pricing</a>
      <a href="/ai">AI</a>
    </div>
    <div class="nav-right">
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<main class="marketing">
  <section class="mkt-hero section">
    <h1 class="mkt-title">Organize by purpose,<br>not by folder.</h1>
    <p class="mkt-sub">Spaces brings files, people, and AI agents into collaborative workspaces. Group content around how you work.</p>
    <div class="mkt-cta">
      <input type="email" id="mkt-email" placeholder="you@example.com" autocomplete="email">
      <button id="mkt-btn" onclick="mktSignIn()">Get started</button>
    </div>
    <div class="mkt-note" id="mkt-err">Magic link &middot; No password &middot; Free</div>
  </section>

  <section class="mkt-grid section">
    <div class="mkt-cell">
      <div class="mkt-cell-label">Organize</div>
      <p>Group files into sections. Add notes, URLs, and references. Structure content your way.</p>
    </div>
    <div class="mkt-cell">
      <div class="mkt-cell-label">Collaborate</div>
      <p>Share with team members. Add AI agents. Everyone works in the same place.</p>
    </div>
    <div class="mkt-cell">
      <div class="mkt-cell-label">Discover</div>
      <p>Search across spaces, files, and people. Find anything instantly.</p>
    </div>
  </section>
</main>

<script>
function toggleTheme(){
  var isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  var saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches))
    document.documentElement.classList.add('dark');
})();

async function mktSignIn(){
  var input=document.getElementById('mkt-email');
  var btn=document.getElementById('mkt-btn');
  var note=document.getElementById('mkt-err');
  var email=input.value.trim();
  if(!email||!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)){note.textContent='Valid email required';note.style.color='#EF4444';return}
  btn.disabled=true;btn.textContent='Sending...';
  try{
    var res=await fetch('/auth/magic-link',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:email})});
    var data=await res.json();
    if(!res.ok)throw new Error(data.error&&data.error.message||'Failed');
    if(data.magic_link)window.location.href=data.magic_link;
    else note.textContent='Check your inbox';
  }catch(e){note.textContent=e.message;note.style.color='#EF4444';btn.disabled=false;btn.textContent='Get started'}
}
document.getElementById('mkt-email')&&document.getElementById('mkt-email').addEventListener('keydown',function(e){if(e.key==='Enter')mktSignIn()});

(function(){
  var els=document.querySelectorAll('.section');
  var obs=new IntersectionObserver(function(entries){
    entries.forEach(function(e){if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}});
  },{threshold:0.05});
  els.forEach(function(s){obs.observe(s)});
})();
</script>
</body>
</html>`;
}
