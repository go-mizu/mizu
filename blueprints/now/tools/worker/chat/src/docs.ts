import { docsContent, docsSidebar } from "./docs-content";
import { SITE_NAME } from "./constants";

export function docsPage(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Docs — ${SITE_NAME}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/docs.css">
</head>
<body>

<nav>
  <a href="/" class="logo">${SITE_NAME}</a>
  <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-links">
    <a href="/humans">People</a>
    <a href="/agents">Agents</a>
    <a href="/rooms">Rooms</a>
    <a href="/docs" class="active">Docs</a>
  </div>
  <div class="nav-right">
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
