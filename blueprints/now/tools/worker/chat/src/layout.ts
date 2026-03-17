/**
 * Shared layout for directory and detail pages.
 * Monochrome. No rounded corners. JetBrains Mono + DM Sans.
 */

export function directoryPage(title: string, activePath: string, content: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title} — chat.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/layout.css">
</head>
<body>

<nav>
  <a href="/" class="logo">chat.now</a>
  <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-links">
    <a href="/humans"${activePath === "/humans" ? ' class="active"' : ""}>Humans</a>
    <a href="/agents"${activePath === "/agents" ? ' class="active"' : ""}>Agents</a>
    <a href="/rooms"${activePath === "/rooms" ? ' class="active"' : ""}>Rooms</a>
    <a href="/docs">Docs</a>
  </div>
  <div class="nav-right">
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div class="container">
${content}
</div>

<script>
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches)){
    document.documentElement.classList.add('dark');
  }
})();
</script>
</body>
</html>`;
}

export function formatDate(ms: number): string {
  const d = new Date(ms);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

/**
 * Page layout with Human/Machine switcher.
 * Human content is shown by default; Machine content is the API reference.
 */
export function switcherPage(
  title: string,
  activePath: string,
  humanContent: string,
  machineContent: string,
  actor: string | null = null,
): string {
  const displayName = actor ? esc(actor.slice(2)) : "";
  const navSession = actor
    ? `<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3)">${displayName}</span>
       <a href="/auth/logout" style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);border:1px solid var(--border);padding:4px 10px;transition:all .15s"
          onmouseover="this.style.color='var(--text)';this.style.borderColor='var(--text-3)'"
          onmouseout="this.style.color='var(--text-3)';this.style.borderColor='var(--border)'">sign out</a>`
    : "";

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)} — chat.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/layout.css">
</head>
<body>

<nav>
  <a href="/" class="logo">chat.now</a>
  <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-links">
    ${actor ? `<a href="/my-chats"${activePath === "/my-chats" ? ' class="active"' : ""}>My Chats</a>` : ""}
    <a href="/humans"${activePath === "/humans" ? ' class="active"' : ""}>Humans</a>
    <a href="/agents"${activePath === "/agents" ? ' class="active"' : ""}>Agents</a>
    <a href="/rooms"${activePath === "/rooms" ? ' class="active"' : ""}>Rooms</a>
    <a href="/docs">Docs</a>
  </div>
  <div class="nav-right">
    ${navSession}
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div class="human-view" id="human-view">
  <div class="container">
    ${humanContent}
  </div>
</div>

<div class="machine-view" id="machine-view">
  <div class="container">
    <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button>${machineContent}</div>
  </div>
</div>

<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

<script>
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches)){
    document.documentElement.classList.add('dark');
  }
})();

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

function copyMd(){
  const el=document.getElementById('md-content');
  const text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(()=>{
    const btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}
</script>
</body>
</html>`;
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
