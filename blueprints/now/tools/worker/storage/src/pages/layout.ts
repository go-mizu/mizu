/**
 * Shared layout for Storage pages.
 * Monochrome. No rounded corners. Inter + JetBrains Mono.
 */

const FONT_LINK = `<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">`;

const THEME_TOGGLE = `<button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
  <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
  <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
</button>`;

const MOBILE_TOGGLE = `<button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
</button>`;

const THEME_SCRIPT = `<script>
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
</script>`;

function navLinks(activePath: string, actor: string | null): string {
  const browseLink = actor
    ? `<a href="/browse"${activePath === "/browse" ? ' class="active"' : ""}>Browse</a>`
    : "";
  return `${browseLink}
    <a href="/developers"${activePath === "/developers" ? ' class="active"' : ""}>Developers</a>
    <a href="/api"${activePath === "/api" ? ' class="active"' : ""}>API</a>
    <a href="/cli"${activePath === "/cli" ? ' class="active"' : ""}>CLI</a>
    <a href="/ai"${activePath === "/ai" ? ' class="active"' : ""}>AI</a>
    <a href="/pricing"${activePath === "/pricing" ? ' class="active"' : ""}>Pricing</a>`;
}

export function directoryPage(
  title: string,
  activePath: string,
  content: string,
  actor: string | null = null,
): string {
  const displayName = actor ? esc(actor.slice(2)) : "";
  const navSession = actor
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)} — Storage</title>
${FONT_LINK}
<link rel="stylesheet" href="/layout.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Storage</a>
    ${MOBILE_TOGGLE}
    <div class="nav-links">
      ${navLinks(activePath, actor)}
    </div>
    <div class="nav-right">
      ${navSession}
      ${THEME_TOGGLE}
    </div>
  </div>
</nav>

<div class="container">
${content}
</div>

${THEME_SCRIPT}
</body>
</html>`;
}

export function immersivePage(
  title: string,
  humanContent: string,
  machineContent: string,
  actor: string | null = null,
  activePath: string = "",
): string {
  const displayName = actor ? esc(actor.slice(2)) : "";
  const navSession = actor
    ? `<span class="nav-user">${displayName}</span>
       <a href="/auth/logout" class="nav-signout">sign out</a>`
    : "";

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)} — Storage</title>
${FONT_LINK}
<link rel="stylesheet" href="/layout.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Storage</a>
    ${MOBILE_TOGGLE}
    <div class="nav-links">
      ${navLinks(activePath, actor)}
    </div>
    <div class="nav-right">
      ${navSession}
      ${THEME_TOGGLE}
    </div>
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
  const text=el.innerText.replace(/^copy\n/,'');
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

export function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function formatDate(ms: number): string {
  const d = new Date(ms);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

export function relativeTime(ms: number): string {
  const diff = Date.now() - ms;
  if (diff < 60_000) return "just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)}d ago`;
  return formatDate(ms);
}

export function esc(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
