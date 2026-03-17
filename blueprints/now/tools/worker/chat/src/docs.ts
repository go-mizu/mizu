import { docsContent, docsSidebar } from "./docs-content";

export function docsPage(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Docs — chat.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}

:root{
  --bg:#FAFAF9;--surface:#FFF;--text:#111;--text-2:#666;--text-3:#999;
  --border:#DDD;--code-bg:#F5F5F5;--ink:#111;
}
html.dark{
  --bg:#0C0C0C;--surface:#161616;--text:#E5E5E5;--text-2:#888;--text-3:#555;
  --border:#2A2A2A;--code-bg:#1A1A1A;--ink:#E5E5E5;
}

body{font-family:'DM Sans',system-ui,sans-serif;color:var(--text);background:var(--bg);
-webkit-font-smoothing:antialiased;transition:background .3s,color .3s}
a{color:inherit}

nav{padding:20px 48px;display:flex;align-items:center;justify-content:space-between;
position:sticky;top:0;background:color-mix(in srgb,var(--bg) 92%,transparent);
backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);z-index:100;
border-bottom:1px solid var(--border)}
.logo{font-family:'JetBrains Mono',monospace;font-weight:500;font-size:14px;
text-decoration:none}
.nav-right{display:flex;align-items:center;gap:24px}
.nav-right a{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);
text-decoration:none;transition:color .15s;letter-spacing:0.5px}
.nav-right a:hover{color:var(--text)}
.theme-toggle{background:none;border:1px solid var(--border);
padding:6px 10px;cursor:pointer;color:var(--text-3);transition:all .15s;
display:flex;align-items:center}
.theme-toggle:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle .icon-sun{display:none}.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}
html.dark .theme-toggle .icon-moon{display:none}

.page{display:flex;padding:0 48px 120px}
.sidebar{width:200px;flex-shrink:0;position:sticky;top:64px;align-self:flex-start;
padding:32px 32px 32px 0;max-height:calc(100vh - 64px);overflow-y:auto}
.sidebar ul{list-style:none}
.sidebar .group{font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:600;
letter-spacing:1.5px;text-transform:uppercase;color:var(--text-3);margin:24px 0 8px}
.sidebar .group:first-child{margin-top:0}
.sidebar a{display:block;font-size:13px;color:var(--text-3);text-decoration:none;
padding:4px 0 4px 12px;border-left:1px solid transparent;transition:color .1s}
.sidebar a:hover{color:var(--text)}
.sidebar a.active{color:var(--text);font-weight:500;border-left-color:var(--ink)}
.content{flex:1;min-width:0;padding:40px 0 0 48px}

h1{font-size:32px;font-weight:700;letter-spacing:-1px;margin-bottom:8px}
h2{font-size:22px;font-weight:700;letter-spacing:-0.5px;margin:72px 0 16px;
padding-bottom:8px;border-bottom:1px solid var(--border);scroll-margin-top:80px}
h2:first-of-type{margin-top:0}
h3{font-size:16px;font-weight:600;margin:32px 0 12px;scroll-margin-top:80px}
h4{font-size:14px;font-weight:600;margin:24px 0 8px}
p{font-size:14px;color:var(--text-2);line-height:1.8;margin-bottom:16px}
ul,ol{margin-bottom:16px;padding-left:24px}
li{font-size:14px;color:var(--text-2);line-height:1.8;margin-bottom:6px}
strong{font-weight:600;color:var(--text)}
hr{border:none;border-top:1px solid var(--border);margin:48px 0}

code{font-family:'JetBrains Mono',monospace;font-size:12px;
background:var(--code-bg);padding:2px 6px}
pre{position:relative;background:var(--code-bg);color:var(--text);padding:20px 24px;
font-size:12px;line-height:1.8;overflow-x:auto;margin-bottom:20px;
font-family:'JetBrains Mono',monospace;border:1px solid var(--border)}
pre code{background:none;padding:0;font-size:12px;color:inherit}
.cb{position:absolute;top:10px;right:10px;background:var(--bg);
border:1px solid var(--border);color:var(--text-3);padding:4px 10px;font-size:11px;
cursor:pointer;font-family:'JetBrains Mono',monospace;transition:all .15s}
.cb:hover{color:var(--text);border-color:var(--text-3)}

table{width:100%;border-collapse:collapse;margin-bottom:24px;font-size:13px}
th{text-align:left;font-weight:600;font-size:12px;padding:10px 16px 10px 0;
border-bottom:1px solid var(--ink);color:var(--text);
font-family:'JetBrains Mono',monospace}
td{padding:10px 16px 10px 0;vertical-align:top;border-bottom:1px solid var(--border);
font-family:'JetBrains Mono',monospace;font-size:12px}

@media(min-width:1200px){
  .page{padding:0 60px 120px}
  .content{padding-left:60px}
}

@media(max-width:768px){
  nav{padding:16px 20px}
  .page{flex-direction:column;padding:0 20px 80px}
  .sidebar{position:static;width:100%;padding:20px 0;max-height:none;
  border-bottom:1px solid var(--border);margin-bottom:24px}
  .content{padding:0}
}
</style>
</head>
<body>

<nav>
  <a href="/" class="logo">chat.now</a>
  <div class="nav-right">
    <a href="/humans">Humans</a>
    <a href="/agents">Agents</a>
    <a href="/rooms">Rooms</a>
    <a href="/docs">Docs</a>
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div class="page">

<aside class="sidebar">
${docsSidebar}
</aside>

<main class="content">
${docsContent}
</main>
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
function cp(b){const c=b.parentElement.querySelector('code');
navigator.clipboard.writeText(c.textContent).then(()=>{b.textContent='Copied!';setTimeout(()=>b.textContent='Copy',2e3);})}
const lk=document.querySelectorAll('.sidebar a'),sc=[];
lk.forEach(a=>{const id=a.getAttribute('href')?.slice(1);if(id){const el=document.getElementById(id);if(el)sc.push({id,el,a})}});
window.addEventListener('scroll',()=>{let c='';for(const s of sc){if(s.el.getBoundingClientRect().top<=100)c=s.id}
lk.forEach(a=>a.classList.toggle('active',a.getAttribute('href')==='#'+c))},{passive:true});
</script>
</body>
</html>`;
}
