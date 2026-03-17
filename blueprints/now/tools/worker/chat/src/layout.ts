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
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}

:root{
  --bg:#FAFAF9;--surface:#FFF;--surface-hover:#F5F5F5;
  --text:#111;--text-2:#666;--text-3:#999;
  --border:#DDD;--code-bg:#F5F5F5;--ink:#111;
}
html.dark{
  --bg:#0C0C0C;--surface:#161616;--surface-hover:#1E1E1E;
  --text:#E5E5E5;--text-2:#888;--text-3:#555;
  --border:#2A2A2A;--code-bg:#1A1A1A;--ink:#E5E5E5;
}

body{font-family:'DM Sans',system-ui,sans-serif;
color:var(--text);background:var(--bg);-webkit-font-smoothing:antialiased;
transition:background .3s,color .3s}
a{color:inherit;text-decoration:none}

nav{padding:20px 48px;display:flex;align-items:center;justify-content:space-between;
position:sticky;top:0;background:color-mix(in srgb,var(--bg) 92%,transparent);
backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);z-index:100}
.logo{font-family:'JetBrains Mono',monospace;font-weight:500;font-size:14px}
.nav-links{display:flex;align-items:center;gap:32px}
.nav-links a{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);
transition:color .15s;letter-spacing:0.5px}
.nav-links a:hover{color:var(--text)}
.nav-links a.active{color:var(--text);font-weight:600}
.nav-right{display:flex;align-items:center;gap:12px}
.theme-toggle{background:none;border:1px solid var(--border);
padding:6px 10px;cursor:pointer;color:var(--text-3);transition:all .15s;
display:flex;align-items:center}
.theme-toggle:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle .icon-sun{display:none}.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}
html.dark .theme-toggle .icon-moon{display:none}

.container{max-width:800px;margin:0 auto;padding:48px 48px 100px}

/* Page header */
.page-header{margin-bottom:40px}
.page-title{font-size:32px;font-weight:700;letter-spacing:-1px;margin-bottom:8px}
.page-desc{font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:12px}
.page-guide{font-family:'JetBrains Mono',monospace;font-size:12px;
color:var(--text-3);line-height:1.8}
.page-guide strong{color:var(--text-2);font-weight:500}
.page-count{font-family:'JetBrains Mono',monospace;font-size:12px;
color:var(--text-3);margin-bottom:16px}
.page-count span{color:var(--text)}

/* Directory list */
.directory{display:flex;flex-direction:column}
.entry{display:flex;align-items:center;gap:16px;padding:14px 0;
border-bottom:1px solid var(--border);transition:background .15s}
.entry:first-child{border-top:1px solid var(--border)}
.entry:hover{background:var(--surface-hover);margin:0 -16px;padding:14px 16px}
.entry-avatar{width:40px;height:40px;flex-shrink:0;overflow:hidden}
.entry-avatar svg{display:block;width:40px;height:40px;filter:grayscale(1) contrast(1.1)}
html.dark .entry-avatar svg{filter:grayscale(1) brightness(1.3)}
.entry-info{flex:1;min-width:0}
.entry-name{font-size:14px;font-weight:600;letter-spacing:-0.2px}
.entry-meta{font-family:'JetBrains Mono',monospace;font-size:11px;
color:var(--text-3);margin-top:3px}
.entry-arrow{color:var(--text-3);font-size:16px;flex-shrink:0;
transition:transform .15s;font-family:'JetBrains Mono',monospace}
.entry:hover .entry-arrow{transform:translateX(3px);color:var(--text-2)}

.empty{text-align:center;padding:80px 20px;color:var(--text-3);font-size:14px}
.empty a{color:var(--text);text-decoration:underline;text-underline-offset:3px}

.mobile-toggle{display:none;background:none;border:none;color:var(--text-3);
cursor:pointer;padding:4px}

@media(max-width:640px){
  nav{padding:16px 20px}
  .nav-links{display:none;position:absolute;top:54px;left:0;right:0;
    flex-direction:column;background:var(--bg);
    padding:16px 20px;gap:16px;z-index:100;border-bottom:1px solid var(--border)}
  .nav-links.open{display:flex}
  .mobile-toggle{display:block}
  .container{padding:28px 20px 60px}
  .page-title{font-size:24px}
  .entry{padding:12px 0;gap:12px}
  .entry-avatar{width:36px;height:36px}
  .entry-avatar svg{width:36px;height:36px}
}
</style>
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
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}

:root{
  --bg:#FAFAF9;--surface:#FFF;--surface-hover:#F5F5F5;
  --text:#111;--text-2:#666;--text-3:#999;
  --border:#DDD;--code-bg:#F5F5F5;--ink:#111;--err:#B91C1C;--ok:#15803D;
}
html.dark{
  --bg:#0C0C0C;--surface:#161616;--surface-hover:#1E1E1E;
  --text:#E5E5E5;--text-2:#888;--text-3:#555;
  --border:#2A2A2A;--code-bg:#1A1A1A;--ink:#E5E5E5;--err:#FCA5A5;--ok:#86EFAC;
}

body{font-family:'DM Sans',system-ui,sans-serif;
color:var(--text);background:var(--bg);-webkit-font-smoothing:antialiased;
transition:background .3s,color .3s;padding-bottom:80px}
a{color:inherit;text-decoration:none}

nav{padding:20px 48px;display:flex;align-items:center;justify-content:space-between;
position:sticky;top:0;background:color-mix(in srgb,var(--bg) 92%,transparent);
backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);z-index:100}
.logo{font-family:'JetBrains Mono',monospace;font-weight:500;font-size:14px}
.nav-links{display:flex;align-items:center;gap:32px}
.nav-links a{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);
transition:color .15s;letter-spacing:0.5px}
.nav-links a:hover{color:var(--text)}
.nav-links a.active{color:var(--text);font-weight:600}
.nav-right{display:flex;align-items:center;gap:12px}
.theme-toggle{background:none;border:1px solid var(--border);
padding:6px 10px;cursor:pointer;color:var(--text-3);transition:all .15s;
display:flex;align-items:center}
.theme-toggle:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle .icon-sun{display:none}.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}
html.dark .theme-toggle .icon-moon{display:none}

.container{max-width:800px;margin:0 auto;padding:48px 48px 100px}

/* Page header */
.page-header{margin-bottom:40px}
.page-title{font-size:32px;font-weight:700;letter-spacing:-1px;margin-bottom:8px}
.page-desc{font-size:14px;color:var(--text-2);line-height:1.7;margin-bottom:12px}
.page-guide{font-family:'JetBrains Mono',monospace;font-size:12px;
color:var(--text-3);line-height:1.8}
.page-guide strong{color:var(--text-2);font-weight:500}
.page-count{font-family:'JetBrains Mono',monospace;font-size:12px;
color:var(--text-3);margin-bottom:16px}
.page-count span{color:var(--text)}

/* Directory list */
.directory{display:flex;flex-direction:column}
.entry{display:flex;align-items:center;gap:16px;padding:14px 0;
border-bottom:1px solid var(--border);transition:background .15s}
.entry:first-child{border-top:1px solid var(--border)}
.entry:hover{background:var(--surface-hover);margin:0 -16px;padding:14px 16px}
.entry-avatar{width:40px;height:40px;flex-shrink:0;overflow:hidden}
.entry-avatar svg{display:block;width:40px;height:40px;filter:grayscale(1) contrast(1.1)}
html.dark .entry-avatar svg{filter:grayscale(1) brightness(1.3)}
.entry-info{flex:1;min-width:0}
.entry-name{font-size:14px;font-weight:600;letter-spacing:-0.2px}
.entry-meta{font-family:'JetBrains Mono',monospace;font-size:11px;
color:var(--text-3);margin-top:3px}
.entry-arrow{color:var(--text-3);font-size:16px;flex-shrink:0;
transition:transform .15s;font-family:'JetBrains Mono',monospace}
.entry:hover .entry-arrow{transform:translateX(3px);color:var(--text-2)}

.empty{text-align:center;padding:80px 20px;color:var(--text-3);font-size:14px}
.empty a{color:var(--text);text-decoration:underline;text-underline-offset:3px}

/* Profile styles */
.profile{max-width:640px}
.profile-header{display:flex;align-items:center;gap:20px;margin-bottom:40px}
.profile-avatar{width:64px;height:64px;flex-shrink:0;overflow:hidden}
.profile-avatar svg{display:block;width:64px;height:64px;filter:grayscale(1) contrast(1.1)}
html.dark .profile-avatar svg{filter:grayscale(1) brightness(1.3)}
.profile-name{font-size:22px;font-weight:700;letter-spacing:-0.5px;margin-bottom:2px}
.profile-meta{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3)}
.profile-badge{display:inline-block;font-family:'JetBrains Mono',monospace;
font-size:11px;font-weight:500;padding:2px 8px;border:1px solid var(--border);
color:var(--text-3);margin-left:8px;vertical-align:middle}

/* Fieldset sections */
fieldset.ps{border:1px solid var(--border);padding:28px 32px;margin-bottom:32px;
background:none}
fieldset.ps legend{font-family:'JetBrains Mono',monospace;font-size:11px;
padding:0 10px;color:var(--text-3);letter-spacing:1px}

/* Chat bubbles */
.chat{display:flex;flex-direction:column;gap:12px}
.chat-msg{display:flex;gap:10px;max-width:85%}
.chat-msg.right{align-self:flex-end;flex-direction:row-reverse}
.chat-av{width:28px;height:28px;flex-shrink:0;overflow:hidden}
.chat-av svg{display:block;width:28px;height:28px;filter:grayscale(1) contrast(1.1)}
html.dark .chat-av svg{filter:grayscale(1) brightness(1.3)}
.chat-av-letter{width:28px;height:28px;background:var(--surface);border:1px solid var(--border);
display:flex;align-items:center;justify-content:center;font-family:'JetBrains Mono',monospace;
font-size:11px;font-weight:600;color:var(--text-3)}
.chat-bubble{padding:10px 14px;font-size:14px;line-height:1.5}
.chat-msg.right .chat-bubble{background:var(--ink);color:var(--bg)}
.chat-msg.left .chat-bubble{background:var(--surface);border:1px solid var(--border);color:var(--text)}
.chat-meta{font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--text-3);
margin-top:4px;padding:0 2px}
.chat-msg.right .chat-meta{text-align:right}

/* Message form */
.msg-form{display:flex;gap:0;margin-top:16px}
.msg-form textarea{flex:1;font-family:'DM Sans',system-ui,sans-serif;font-size:14px;
padding:12px 14px;border:1px solid var(--border);background:var(--bg);
color:var(--text);resize:none;height:44px;outline:none;transition:border-color .15s;line-height:1.4}
.msg-form textarea:focus{border-color:var(--text-3)}
.msg-form textarea::placeholder{color:var(--text-3)}
.msg-form button{font-family:'JetBrains Mono',monospace;font-size:12px;
padding:12px 20px;border:1px solid var(--ink);border-left:none;
background:var(--ink);color:var(--bg);cursor:pointer;transition:opacity .15s;white-space:nowrap}
.msg-form button:hover:not(:disabled){opacity:0.85}
.msg-form button:disabled{opacity:0.4;cursor:not-allowed}
.msg-sent{font-family:'JetBrains Mono',monospace;font-size:11px;
color:var(--ok);margin-top:8px;display:none}
.msg-sent.show{display:block}
.msg-error{font-family:'JetBrains Mono',monospace;font-size:11px;
color:var(--err);margin-top:8px;min-height:16px}
.signin-prompt{font-size:13px;color:var(--text-3);margin-top:16px;
padding:14px 0;border-top:1px solid var(--border)}
.signin-prompt a{text-decoration:underline;text-underline-offset:3px;color:var(--text)}

/* Machine view markdown */
.machine-view{display:none}
.machine-view.active{display:block}
.human-view.hidden{display:none}
.md{background:var(--surface);padding:32px 40px;border:1px solid var(--border);
font-family:'JetBrains Mono',monospace;font-size:13px;line-height:2;
color:var(--text-2);white-space:pre-wrap;word-break:break-word;position:relative}
.md .h1{color:var(--text);font-weight:700;font-size:16px}
.md .h2{color:var(--text);font-weight:600;font-size:14px}
.md .h3{color:var(--text-3);font-weight:500}
.md .dim{color:var(--text-3)}
.md-copy{position:absolute;top:12px;right:12px;
font-family:'JetBrains Mono',monospace;font-size:11px;
padding:4px 12px;border:1px solid var(--border);background:var(--bg);
color:var(--text-3);cursor:pointer;transition:all .15s}
.md-copy:hover{color:var(--text);border-color:var(--text-3)}

/* Floating mode switch */
.mode-switch{position:fixed;bottom:24px;left:50%;transform:translateX(-50%);
display:flex;background:var(--bg);border:1px solid var(--border);z-index:200;
box-shadow:0 4px 24px rgba(0,0,0,0.08)}
html.dark .mode-switch{box-shadow:0 4px 24px rgba(0,0,0,0.4)}
.mode-switch button{font-family:'JetBrains Mono',monospace;font-size:11px;
letter-spacing:1px;padding:10px 20px;border:none;background:none;
color:var(--text-3);cursor:pointer;display:flex;align-items:center;gap:8px;
transition:color .15s}
.mode-switch button.active{color:var(--text)}
.mode-switch button+button{border-left:1px solid var(--border)}
.dot{width:6px;height:6px;border:1px solid var(--text-3);display:inline-block}
.mode-switch button.active .dot{background:var(--text);border-color:var(--text)}

.mobile-toggle{display:none;background:none;border:none;color:var(--text-3);
cursor:pointer;padding:4px}

@media(max-width:640px){
  nav{padding:16px 20px}
  .nav-links{display:none;position:absolute;top:54px;left:0;right:0;
    flex-direction:column;background:var(--bg);
    padding:16px 20px;gap:16px;z-index:100;border-bottom:1px solid var(--border)}
  .nav-links.open{display:flex}
  .mobile-toggle{display:block}
  .container{padding:28px 20px 60px}
  .page-title{font-size:24px}
  .entry{padding:12px 0;gap:12px}
  .entry-avatar{width:36px;height:36px}
  .entry-avatar svg{width:36px;height:36px}
  .profile-header{flex-direction:column;text-align:center}
  .profile-avatar{width:56px;height:56px}
  .profile-avatar svg{width:56px;height:56px}
  fieldset.ps{padding:20px 16px}
  .chat-msg{max-width:92%}
  .md{padding:24px 20px;font-size:12px}
  .mode-switch{bottom:16px}
  .mode-switch button{padding:8px 14px;font-size:10px}
  .msg-form{flex-direction:column}
  .msg-form button{border-left:1px solid var(--ink);border-top:none}
}
</style>
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
