var Me=Object.defineProperty;var He=(e,t,n)=>t in e?Me(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var K=(e,t,n)=>He(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))s(a);new MutationObserver(a=>{for(const r of a)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&s(i)}).observe(document,{childList:!0,subtree:!0});function n(a){const r={};return a.integrity&&(r.integrity=a.integrity),a.referrerPolicy&&(r.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?r.credentials="include":a.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function s(a){if(a.ep)return;a.ep=!0;const r=n(a);fetch(a.href,r)}})();class Ae{constructor(){K(this,"routes",[]);K(this,"currentPath","");K(this,"notFoundRenderer",null)}addRoute(t,n){const s=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:s,renderer:n})}setNotFound(t){this.notFoundRenderer=t}navigate(t,n=!1){t!==this.currentPath&&(n?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const n=t.target.closest("a[data-link]");if(n){t.preventDefault();const s=n.getAttribute("href");s&&this.navigate(s)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),n=t.pathname,s=Ne(t.search);this.currentPath=n+t.search;for(const a of this.routes){const r=Te(a.segments,n);if(r!==null){a.renderer(r,s);return}}this.notFoundRenderer&&this.notFoundRenderer({},s)}}function Te(e,t){const n=t.split("/").filter(Boolean);if(e.length===0&&n.length===0)return{};if(e.length!==n.length)return null;const s={};for(let a=0;a<e.length;a++){const r=e[a],i=n[a];if(r.startsWith(":"))s[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return s}function Ne(e){const t={};return new URLSearchParams(e).forEach((s,a)=>{t[a]=s}),t}const X="/api";async function u(e,t){let n=`${X}${e}`;if(t){const a=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&a.set(i,o)});const r=a.toString();r&&(n+=`?${r}`)}const s=await fetch(n);if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function A(e,t){const n=await fetch(`${X}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function he(e,t){const n=await fetch(`${X}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function z(e,t){const n=await fetch(`${X}${e}`,{method:"DELETE",headers:t?{"Content-Type":"application/json"}:void 0,body:t?JSON.stringify(t):void 0});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}function ne(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.time_range&&(n.time_range=t.time_range),t.region&&(n.region=t.region),t.language&&(n.language=t.language),t.safe_search&&(n.safe_search=t.safe_search),t.site&&(n.site=t.site),t.exclude_site&&(n.exclude_site=t.exclude_site),t.lens&&(n.lens=t.lens)),n}const f={search(e,t){return u("/search",ne(e,t))},searchImages(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(n.size=t.size),t.color&&t.color!=="any"&&(n.color=t.color),t.type&&t.type!=="any"&&(n.type=t.type),t.aspect&&t.aspect!=="any"&&(n.aspect=t.aspect),t.time&&t.time!=="any"&&(n.time=t.time),t.rights&&t.rights!=="any"&&(n.rights=t.rights),t.filetype&&t.filetype!=="any"&&(n.filetype=t.filetype),t.safe&&(n.safe=t.safe)),u("/search/images",n)},reverseImageSearch(e){return A("/search/images/reverse",{url:e})},searchVideos(e,t){return u("/search/videos",ne(e,t))},searchNews(e,t){return u("/search/news",ne(e,t))},suggest(e){return u("/suggest",{q:e})},trending(){return u("/suggest/trending")},calculate(e){return u("/instant/calculate",{q:e})},convert(e){return u("/instant/convert",{q:e})},currency(e){return u("/instant/currency",{q:e})},weather(e){return u("/instant/weather",{q:e})},define(e){return u("/instant/define",{q:e})},time(e){return u("/instant/time",{q:e})},knowledge(e){return u(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return u("/preferences")},setPreference(e,t){return A("/preferences",{domain:e,action:t})},deletePreference(e){return z(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return u("/lenses")},createLens(e){return A("/lenses",e)},deleteLens(e){return z(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return u("/history")},clearHistory(){return z("/history")},deleteHistoryItem(e){return z(`/history/${encodeURIComponent(e)}`)},getSettings(){return u("/settings")},updateSettings(e){return he("/settings",e)},getBangs(){return u("/bangs")},parseBang(e){return u("/bangs/parse",{q:e})},getRelated(e){return u("/related",{q:e})},newsHome(){return u("/news/home")},newsCategory(e,t=1){return u(`/news/category/${e}`,{page:String(t)})},newsSearch(e,t){const n={q:e};return t!=null&&t.page&&(n.page=String(t.page)),t!=null&&t.time&&(n.time=t.time),t!=null&&t.source&&(n.source=t.source),u("/news/search",n)},newsStory(e){return u(`/news/story/${e}`)},newsLocal(e){const t={};return e&&(t.city=e.city,e.state&&(t.state=e.state),t.country=e.country),u("/news/local",t)},newsFollowing(){return u("/news/following")},newsPreferences(){return u("/news/preferences")},updateNewsPreferences(e){return he("/news/preferences",e)},followNews(e,t){return A("/news/follow",{type:e,id:t})},unfollowNews(e,t){return z("/news/follow",{type:e,id:t})},hideNewsSource(e){return A("/news/hide",{source:e})},setNewsLocation(e){return A("/news/location",e)},recordNewsRead(e,t){return A("/news/read",{article:e,duration:t})}};function Re(e){let t={...e};const n=new Set;return{get(){return t},set(s){t={...t,...s},n.forEach(a=>a(t))},subscribe(s){return n.add(s),()=>{n.delete(s)}}}}const Ce="mizu_search_state";function Oe(){try{const e=localStorage.getItem(Ce);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const R=Re(Oe());R.subscribe(e=>{try{localStorage.setItem(Ce,JSON.stringify(e))}catch{}});function ee(e){const t=R.get(),n=[e,...t.recentSearches.filter(s=>s!==e)].slice(0,20);R.set({recentSearches:n})}const Ie='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',qe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',je='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',Pe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Fe='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',Ue='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function P(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",n=e.initialValue?ze(e.initialValue):"",s=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${Ie}</span>
        <input
          id="search-input"
          type="text"
          value="${n}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${s}" type="button" aria-label="Clear">
          ${qe}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${je}
        </button>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${Pe}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function F(e){const t=document.getElementById("search-input"),n=document.getElementById("search-clear-btn"),s=document.getElementById("autocomplete-dropdown"),a=document.getElementById("search-box-wrapper");if(!t||!n||!s||!a)return;let r=null,i=[],o=-1,l=!1;function p(h){if(i=h,o=-1,h.length===0){v();return}l=!0,s.innerHTML=h.map((c,m)=>`
        <div class="autocomplete-item ${m===o?"active":""}" data-index="${m}">
          <span class="suggestion-icon">${c.icon}</span>
          ${c.prefix?`<span class="bang-trigger">${pe(c.prefix)}</span>`:""}
          <span>${pe(c.text)}</span>
        </div>
      `).join(""),s.classList.remove("hidden"),s.classList.add("has-items"),s.querySelectorAll(".autocomplete-item").forEach(c=>{c.addEventListener("mousedown",m=>{m.preventDefault();const b=parseInt(c.dataset.index||"0");y(b)}),c.addEventListener("mouseenter",()=>{const m=parseInt(c.dataset.index||"0");M(m)})})}function v(){l=!1,s.classList.add("hidden"),s.classList.remove("has-items"),s.innerHTML="",i=[],o=-1}function M(h){o=h,s.querySelectorAll(".autocomplete-item").forEach((c,m)=>{c.classList.toggle("active",m===h)})}function y(h){const c=i[h];c&&(c.type==="bang"&&c.prefix?(t.value=c.prefix+" ",t.focus(),C(c.prefix+" ")):(t.value=c.text,v(),$(c.text)))}function $(h){const c=h.trim();c&&(v(),e(c))}async function C(h){const c=h.trim();if(!c){U();return}if(c.startsWith("!"))try{const b=(await f.getBangs()).filter(H=>H.trigger.startsWith(c)||H.name.toLowerCase().includes(c.slice(1).toLowerCase())).slice(0,8);if(b.length>0){p(b.map(H=>({text:H.name,type:"bang",icon:Ue,prefix:H.trigger})));return}}catch{}try{const m=await f.suggest(c);if(t.value.trim()!==c)return;const b=m.map(H=>({text:H.text,type:"suggestion",icon:Ie}));b.length===0?U(c):p(b)}catch{U(c)}}function U(h){let m=R.get().recentSearches;if(h&&(m=m.filter(b=>b.toLowerCase().includes(h.toLowerCase()))),m.length===0){v();return}p(m.slice(0,8).map(b=>({text:b,type:"recent",icon:Fe})))}t.addEventListener("input",()=>{const h=t.value;n.classList.toggle("hidden",h.length===0),r&&clearTimeout(r),r=setTimeout(()=>C(h),150)}),t.addEventListener("focus",()=>{t.value.trim()?C(t.value):U()}),t.addEventListener("keydown",h=>{if(!l){if(h.key==="Enter"){$(t.value);return}if(h.key==="ArrowDown"){C(t.value);return}return}switch(h.key){case"ArrowDown":h.preventDefault(),M(Math.min(o+1,i.length-1));break;case"ArrowUp":h.preventDefault(),M(Math.max(o-1,-1));break;case"Enter":h.preventDefault(),o>=0?y(o):$(t.value);break;case"Escape":v();break;case"Tab":v();break}}),t.addEventListener("blur",()=>{setTimeout(()=>v(),200)}),n.addEventListener("click",()=>{t.value="",n.classList.add("hidden"),t.focus(),U()})}function pe(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ze(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Ve=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],De=[{label:"Calculator",icon:Ke(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:Ze(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:Je(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:Qe(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:Xe(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:et(),query:"define serendipity",color:"bg-red/10 text-red"}];function Ge(){return`
    <div class="min-h-screen flex flex-col">
      <div class="flex-1 flex flex-col items-center justify-center px-4 -mt-20">
        <!-- Logo -->
        <div class="mb-8 text-center">
          <h1 class="text-6xl font-semibold mb-2 select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </h1>
          <p class="text-secondary text-lg">Privacy-first search</p>
        </div>

        <!-- Search Box -->
        <div class="w-full max-w-2xl mb-6">
          ${P({size:"lg",autofocus:!0})}
        </div>

        <!-- Search Buttons -->
        <div class="flex gap-3 mb-8">
          <button id="home-search-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            Mizu Search
          </button>
          <button id="home-lucky-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            I'm Feeling Lucky
          </button>
        </div>

        <!-- Bang Shortcuts -->
        <div class="flex flex-wrap justify-center gap-2 mb-8">
          ${Ve.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${se(e.trigger)}</span>
              <span class="text-tertiary ml-1">${se(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${De.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${Ye(e.query)}">
                ${e.icon}
                <span>${se(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${tt()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${nt()}
            News
          </a>
        </div>
      </div>

      <!-- Footer -->
      <footer class="py-4 text-center">
        <div class="text-xs text-light space-x-4">
          <span>Use <strong>!bangs</strong> to search other sites directly</span>
          <span>&middot;</span>
          <a href="/settings" data-link class="hover:text-secondary">Settings</a>
          <span>&middot;</span>
          <a href="/history" data-link class="hover:text-secondary">History</a>
        </div>
      </footer>
    </div>
  `}function We(e){F(s=>{e.navigate(`/search?q=${encodeURIComponent(s)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var r;const s=document.getElementById("search-input"),a=(r=s==null?void 0:s.value)==null?void 0:r.trim();a&&e.navigate(`/search?q=${encodeURIComponent(a)}`)});const n=document.getElementById("home-lucky-btn");n==null||n.addEventListener("click",()=>{var r;const s=document.getElementById("search-input"),a=(r=s==null?void 0:s.value)==null?void 0:r.trim();a&&e.navigate(`/search?q=${encodeURIComponent(a)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(s=>{s.addEventListener("click",()=>{const a=s.dataset.bang||"",r=document.getElementById("search-input");r&&(r.value=a+" ",r.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(s=>{s.addEventListener("click",()=>{const a=s.dataset.query||"";a&&e.navigate(`/search?q=${encodeURIComponent(a)}`)})})}function se(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ye(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Ke(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function Ze(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function Je(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function Qe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function Xe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function et(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function tt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function nt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}const st='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',at='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',rt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',it='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>';function ot(e,t){const n=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,s=dt(e.url),a=e.published?ut(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${L(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="result-sitelinks">
        ${e.sitelinks.map(l=>`<a href="${L(l.url)}" target="_blank" rel="noopener">${S(l.title)}</a>`).join("")}
       </div>`:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${L(e.domain)}">
      <div class="result-url">
        <img class="favicon" src="${L(n)}" alt="" width="18" height="18" loading="lazy" onerror="this.style.display='none'" />
        <div>
          <span class="text-sm">${S(e.domain)}</span>
          <span class="breadcrumbs">${s}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${L(e.url)}" target="_blank" rel="noopener">${S(e.title)}</a>
          </div>
          ${a?`<span class="result-date">${S(a)} -- </span>`:""}
          <div class="result-snippet">${r}</div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${st}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function lt(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const n=e.dataset.menuIndex,s=document.getElementById(`domain-menu-${n}`),a=e.closest(".search-result"),r=(a==null?void 0:a.dataset.domain)||"";if(!s)return;if(!s.classList.contains("hidden")){s.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),s.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${L(r)}">
          ${at}
          <span>Boost ${S(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${L(r)}">
          ${rt}
          <span>Lower ${S(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${L(r)}">
          ${it}
          <span>Block ${S(r)}</span>
        </button>
      `,s.classList.remove("hidden"),s.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const l=o.dataset.action||"",p=o.dataset.domain||"";try{await f.setPreference(p,l),s.classList.add("hidden"),ct(`${l.charAt(0).toUpperCase()+l.slice(1)}ed ${p}`)}catch(v){console.error("Failed to set preference:",v)}})});const i=o=>{!s.contains(o.target)&&o.target!==e&&(s.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function ct(e){const t=document.getElementById("toast");t&&t.remove();const n=document.createElement("div");n.id="toast",n.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",n.textContent=e,document.body.appendChild(n),setTimeout(()=>{n.style.opacity="0",setTimeout(()=>n.remove(),300)},2e3)}function dt(e){try{const n=new URL(e).pathname.split("/").filter(Boolean);return n.length===0?"":" > "+n.map(s=>S(decodeURIComponent(s))).join(" > ")}catch{return""}}function ut(e){try{const t=new Date(e),s=new Date().getTime()-t.getTime(),a=Math.floor(s/(1e3*60*60*24));return a===0?"Today":a===1?"1 day ago":a<7?`${a} days ago`:a<30?`${Math.floor(a/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function S(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function L(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ht='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',pt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',gt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',vt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',mt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',ft='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',yt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',wt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function xt(e){switch(e.type){case"calculator":return bt(e);case"unit_conversion":return $t(e);case"currency":return kt(e);case"weather":return Ct(e);case"definition":return It(e);case"time":return Et(e);default:return Lt(e)}}function bt(e){const t=e.data||{},n=t.expression||e.query||"",s=t.formatted||t.result||e.result||"";return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ht}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${d(n)} = ${d(String(s))}</div>
    </div>
  `}function $t(e){const t=e.data||{},n=t.from_value??"",s=t.from_unit??"",a=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${pt}
        <span class="instant-type">Unit Conversion${i?` -- ${d(i)}`:""}</span>
      </div>
      <div class="instant-result">${d(String(n))} ${d(s)} = ${d(String(a))} ${d(r)}</div>
      ${t.formatted?`<div class="instant-sub">${d(t.formatted)}</div>`:""}
    </div>
  `}function kt(e){const t=e.data||{},n=t.from_value??"",s=t.from_currency??"",a=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${gt}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${d(String(n))} ${d(s)} = ${d(String(a))} ${d(r)}</div>
      ${i?`<div class="instant-sub">1 ${d(s)} = ${d(String(i))} ${d(r)}</div>`:""}
    </div>
  `}function Ct(e){const t=e.data||{},n=t.location||"",s=t.temperature??"",a=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=vt;return a.includes("cloud")||a.includes("overcast")?o=mt:(a.includes("rain")||a.includes("drizzle")||a.includes("storm"))&&(o=ft),`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">Weather</div>
      <div class="flex items-center gap-4 mb-3">
        <div>${o}</div>
        <div>
          <div class="text-2xl font-semibold text-primary">${d(String(s))}&deg;</div>
          <div class="text-secondary capitalize">${d(t.condition||"")}</div>
        </div>
      </div>
      <div class="text-sm font-medium text-primary mb-2">${d(n)}</div>
      <div class="flex gap-6 text-sm text-tertiary">
        ${r?`<span>Humidity: ${d(r)}</span>`:""}
        ${i?`<span>Wind: ${d(i)}</span>`:""}
      </div>
    </div>
  `}function It(e){const t=e.data||{},n=t.word||e.query||"",s=t.phonetic||"",a=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[];return`
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${yt}
        <span class="instant-type">Definition</span>
      </div>
      <div class="flex items-baseline gap-3 mb-1">
        <span class="text-xl font-semibold text-primary">${d(n)}</span>
        ${s?`<span class="text-tertiary text-sm">${d(s)}</span>`:""}
      </div>
      ${a?`<div class="text-sm italic text-secondary mb-2">${d(a)}</div>`:""}
      ${r.length>0?`<ol class="list-decimal list-inside space-y-1 text-sm text-snippet mb-3">
              ${r.map(o=>`<li>${d(o)}</li>`).join("")}
             </ol>`:""}
      ${i.length>0?`<div class="text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(o=>d(o)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function Et(e){const t=e.data||{},n=t.location||"",s=t.time||"",a=t.date||"",r=t.timezone||"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${wt}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${d(n)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${d(s)}</div>
      <div class="text-sm text-tertiary">${d(a)}</div>
      ${r?`<div class="text-xs text-light mt-1">${d(r)}</div>`:""}
    </div>
  `}function Lt(e){return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${d(e.type)}</div>
      <div class="instant-result">${d(e.result)}</div>
    </div>
  `}function d(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const St='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function _t(e){const t=e.image?`<img class="kp-image" src="${ae(e.image)}" alt="${ae(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",n=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${T(r.label)}</td>
                <td class="fact-value">${T(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",s=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${ae(r.url)}" target="_blank" rel="noopener">
              ${St}
              <span>${T(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",a=e.source?`<div class="kp-source">Source: ${T(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${T(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${T(e.subtitle)}</div>`:""}
      <div class="kp-description">${T(e.description)}</div>
      ${n}
      ${s}
      ${a}
    </div>
  `}function T(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ae(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Bt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',Mt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function Ht(e){const{currentPage:t,hasMore:n,totalResults:s,perPage:a}=e,r=Math.min(Math.ceil(s/a),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const l=[];for(let y=i;y<=o;y++)l.push(y);const p=At(t),v=t<=1?"disabled":"",M=!n&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${p}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${v}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${Bt}
          </button>
          ${l.map(y=>`
            <button class="pagination-btn ${y===t?"active":""}" data-page="${y}">
              ${y}
            </button>
          `).join("")}
          <button class="pagination-btn ${M}" data-page="${t+1}" ${!n&&t>=r?"disabled":""} aria-label="Next page">
            ${Mt}
          </button>
        </div>
      </div>
    </div>
  `}function At(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],n=["M","i","z","u"],s=Math.min(e-1,6);let a=[n[0]];for(let r=0;r<1+s;r++)a.push("i");a.push("z");for(let r=0;r<1+s;r++)a.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${a.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function Tt(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(n=>{n.addEventListener("click",()=>{const s=parseInt(n.dataset.page||"1");isNaN(s)||n.disabled||(e(s),window.scrollTo({top:0,behavior:"smooth"}))})})}const Nt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Rt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Ot='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',qt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>';function te(e){const{query:t,active:n}=e,s=encodeURIComponent(t);return`
    <div class="search-tabs" id="search-tabs">
      ${[{id:"all",label:"All",icon:Nt,href:`/search?q=${s}`},{id:"images",label:"Images",icon:Rt,href:`/images?q=${s}`},{id:"videos",label:"Videos",icon:Ot,href:`/videos?q=${s}`},{id:"news",label:"News",icon:qt,href:`/news?q=${s}`}].map(r=>`
        <a class="search-tab ${r.id===n?"active":""}" href="${r.href}" data-link data-tab="${r.id}">
          ${r.icon}
          <span>${r.label}</span>
        </a>
      `).join("")}
    </div>
  `}const jt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Pt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',ge=[{value:"",label:"Any time"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}];function Ft(e,t){var a;const n=((a=ge.find(r=>r.value===t))==null?void 0:a.label)||"Any time",s=t!=="";return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${P({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${jt}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          <div class="flex items-center gap-2">
            ${te({query:e,active:"all"})}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${s?"active-filter":""}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${N(n)}</span>
                ${Pt}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${ge.map(r=>`
                  <button class="time-filter-option ${r.value===t?"active":""}" data-time-range="${r.value}">
                    ${N(r.label)}
                  </button>
                `).join("")}
              </div>
            </div>
          </div>
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="search-content" class="max-w-[1200px] pl-[170px] pr-4 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Ut(e,t,n){const s=parseInt(n.page||"1"),a=n.time_range||"",r=R.get().settings;F(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),zt(e,t),t&&ee(t),Vt(e,t,s,a,r.results_per_page)}function zt(e,t,n){const s=document.getElementById("time-filter-btn"),a=document.getElementById("time-filter-dropdown");!s||!a||(s.addEventListener("click",r=>{r.stopPropagation(),a.classList.toggle("hidden")}),a.querySelectorAll(".time-filter-option").forEach(r=>{r.addEventListener("click",()=>{const i=r.dataset.timeRange||"";a.classList.add("hidden");let o=`/search?q=${encodeURIComponent(t)}`;i&&(o+=`&time_range=${i}`),e.navigate(o)})}),document.addEventListener("click",r=>{!a.contains(r.target)&&r.target!==s&&a.classList.add("hidden")}))}async function Vt(e,t,n,s,a){const r=document.getElementById("search-content");if(!(!r||!t))try{const i=await f.search(t,{page:n,per_page:a,time_range:s||void 0});if(i.redirect){window.location.href=i.redirect;return}Dt(r,e,i,t,n,s)}catch(i){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${N(String(i))}</p>
      </div>
    `}}function Dt(e,t,n,s,a,r){const i=n.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(n.corrected_query)}" data-link class="text-link font-medium">${N(n.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(s)}&exact=1" data-link class="text-link">${N(s)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${Gt(n.total_results)} results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,l=n.instant_answer?xt(n.instant_answer):"",p=n.results.length>0?n.results.map(($,C)=>ot($,C)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${N(s)}</strong>"</div>`,v=n.related_searches&&n.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${n.related_searches.map($=>`
            <a href="/search?q=${encodeURIComponent($)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${N($)}
            </a>
          `).join("")}
        </div>
      </div>
    `:"",M=Ht({currentPage:a,hasMore:n.has_more,totalResults:n.total_results,perPage:n.per_page}),y=n.knowledge_panel?_t(n.knowledge_panel):"";e.innerHTML=`
    <div class="flex gap-8">
      <div class="flex-1 min-w-0">
        ${i}
        ${o}
        ${l}
        ${p}
        ${v}
        ${M}
      </div>
      ${y?`<aside class="hidden lg:block flex-shrink-0 w-[360px] pt-2">${y}</aside>`:""}
    </div>
  `,lt(),Tt($=>{let C=`/search?q=${encodeURIComponent(s)}&page=${$}`;r&&(C+=`&time_range=${r}`),t.navigate(C)})}function Gt(e){return e.toLocaleString("en-US")}function N(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Wt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Yt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',ve='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',me='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',Kt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',Ee='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',Zt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>',Jt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>';let G="",I={},W=1,q=!1,B=!0,E=[],V=!1,ie=[],Z=null;function Qt(e){return`
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 shadow-sm">
        <div class="flex items-center gap-4 px-4 py-2">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[600px] flex items-center gap-2">
            ${P({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${Yt}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Wt}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1">
          ${te({query:e,active:"images"})}
          <button id="tools-btn" class="tools-btn ml-4">
            ${Kt}
            <span>Tools</span>
            ${Ee}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${Xt()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${Zt}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${Jt}</button>
        </div>
      </div>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 p-3">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-overlay"></div>
          <div class="preview-container">
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${ve}</button>
            <div class="preview-main">
              <div class="preview-image-wrap">
                <img id="preview-image" src="" alt="" />
              </div>
              <div class="preview-sidebar">
                <div id="preview-details" class="preview-info"></div>
              </div>
            </div>
          </div>
        </div>
      </main>

      <!-- Reverse image search modal -->
      <div id="reverse-modal" class="modal hidden">
        <div class="modal-content">
          <div class="modal-header">
            <h2>Search by image</h2>
            <button id="reverse-modal-close" class="modal-close">${ve}</button>
          </div>
          <div class="modal-body">
            <div id="drop-zone" class="drop-zone">
              <p>Drag and drop an image here or</p>
              <label class="file-upload-btn">
                Upload a file
                <input type="file" id="image-upload" accept="image/*" hidden />
              </label>
            </div>
            <div class="url-input-section">
              <p>Or paste an image URL:</p>
              <div class="url-input-container">
                <input type="text" id="image-url-input" placeholder="https://example.com/image.jpg" />
                <button id="url-search-btn">Search</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  `}function Xt(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${Ee}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(n=>`
              <button class="filter-option${n==="any"?" active":""}" data-value="${n}">
                ${Q(t.id,n)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function Q(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function en(e,t){G=t,I={},W=1,E=[],B=!0,V=!1,ie=[],F(n=>{e.navigate(`/images?q=${encodeURIComponent(n)}`)}),t&&ee(t),tn(),nn(),rn(),on(),sn(e),oe(t,I)}function tn(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{V=!V,t.classList.toggle("hidden",!V),e.classList.toggle("active",V)})}function nn(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(s=>{s.addEventListener("click",a=>{a.stopPropagation();const r=s.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i==null||i.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(s=>{s.addEventListener("click",()=>{const a=s.closest(".filter-dropdown"),r=a==null?void 0:a.dataset.dropdown,i=s.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);!r||!i||!o||(a.querySelectorAll(".filter-option").forEach(l=>l.classList.remove("active")),s.classList.add("active"),i==="any"?(delete I[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=Q(r,"any").replace("Any ","")):(I[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=Q(r,i)),a.classList.add("hidden"),fe(),W=1,E=[],B=!0,oe(G,I))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(s=>s.classList.add("hidden"))});const n=document.getElementById("clear-filters");n&&n.addEventListener("click",()=>{I={},W=1,E=[],B=!0,t.querySelectorAll(".filter-chip").forEach(s=>{const a=s.dataset.filter;s.classList.remove("has-value"),s.querySelector(".filter-chip-label").textContent=Q(a,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(s=>{s.querySelectorAll(".filter-option").forEach((a,r)=>{a.classList.toggle("active",r===0)})}),fe(),oe(G,I)})}function fe(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(I).length===0)}function sn(e){const t=document.getElementById("related-searches");if(!t)return;t.addEventListener("click",r=>{const i=r.target.closest(".related-chip");if(i){const o=i.getAttribute("data-query");o&&e.navigate(`/images?q=${encodeURIComponent(o)}`)}});const n=t.querySelector(".related-scroll-left"),s=t.querySelector(".related-scroll-right"),a=t.querySelector(".related-searches-list");n&&s&&a&&(n.addEventListener("click",()=>{a.scrollBy({left:-200,behavior:"smooth"})}),s.addEventListener("click",()=>{a.scrollBy({left:200,behavior:"smooth"})}),a.addEventListener("scroll",()=>{Le()}))}function Le(){const e=document.getElementById("related-searches");if(!e)return;const t=e.querySelector(".related-searches-list"),n=e.querySelector(".related-scroll-left"),s=e.querySelector(".related-scroll-right");!t||!n||!s||(n.classList.toggle("hidden",t.scrollLeft<=0),s.classList.toggle("hidden",t.scrollLeft>=t.scrollWidth-t.clientWidth-10))}function an(e){const t=document.getElementById("related-searches");if(!t)return;if(!e||e.length===0){t.classList.add("hidden");return}const n=t.querySelector(".related-searches-list");n&&(n.innerHTML=e.map(s=>`
    <button class="related-chip" data-query="${_(s)}">
      <span class="related-chip-text">${j(s)}</span>
    </button>
  `).join(""),t.classList.remove("hidden"),setTimeout(Le,50))}function rn(e){const t=document.getElementById("reverse-search-btn"),n=document.getElementById("reverse-modal"),s=document.getElementById("reverse-modal-close"),a=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!n||(t.addEventListener("click",()=>n.classList.remove("hidden")),s==null||s.addEventListener("click",()=>n.classList.add("hidden")),n.addEventListener("click",l=>{l.target===n&&n.classList.add("hidden")}),a&&(a.addEventListener("dragover",l=>{l.preventDefault(),a.classList.add("drag-over")}),a.addEventListener("dragleave",()=>a.classList.remove("drag-over")),a.addEventListener("drop",l=>{var v;l.preventDefault(),a.classList.remove("drag-over");const p=(v=l.dataTransfer)==null?void 0:v.files;p&&p[0]&&(ye(p[0]),n.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(ye(r.files[0]),n.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const l=i.value.trim();l&&(we(l),n.classList.add("hidden"))}),i.addEventListener("keydown",l=>{if(l.key==="Enter"){const p=i.value.trim();p&&(we(p),n.classList.add("hidden"))}})))}async function ye(e,t){alert("Image upload coming soon. Please use the URL option for now.")}async function we(e,t){const n=document.getElementById("images-content");if(n){n.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const s=await f.reverseImageSearch(e);n.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${_(e)}" alt="Query image" class="query-image" />
        </div>
        ${s.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${s.similar_images.length})</h3>
            <div class="image-grid">
              ${s.similar_images.map((a,r)=>de(a,r)).join("")}
            </div>
          </div>
        `:'<div class="py-8 text-secondary">No similar images found.</div>'}
      </div>
    `,n.querySelectorAll(".image-card").forEach(a=>{a.addEventListener("click",()=>{const r=parseInt(a.dataset.imageIndex||"0",10);ce(s.similar_images[r])})})}catch(s){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${j(String(s))}</p>
      </div>
    `}}}function on(){const e=document.getElementById("preview-panel"),t=document.getElementById("preview-close"),n=e==null?void 0:e.querySelector(".preview-overlay");t==null||t.addEventListener("click",re),n==null||n.addEventListener("click",re),document.addEventListener("keydown",s=>{s.key==="Escape"&&re()})}function ce(e){const t=document.getElementById("preview-panel"),n=document.getElementById("preview-image"),s=document.getElementById("preview-details");if(!t||!n||!s)return;n.src=e.url,n.alt=e.title;const a=e.width&&e.height&&e.width>0&&e.height>0;s.innerHTML=`
    <div class="preview-header">
      <img src="${_(e.thumbnail_url||e.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${j(e.title||"Untitled")}</h3>
        <a href="${_(e.source_url)}" target="_blank" class="preview-domain">${j(e.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${a?`<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${e.width} × ${e.height}</span></div>`:""}
      ${e.format?`<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${e.format.toUpperCase()}</span></div>`:""}
    </div>
    <div class="preview-actions">
      <a href="${_(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${me}
      </a>
      <a href="${_(e.url)}" target="_blank" class="preview-btn">
        View full image ${me}
      </a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function re(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function ln(){Z&&Z.disconnect();const e=document.getElementById("images-content");if(!e)return;const t=document.getElementById("scroll-sentinel");t&&t.remove();const n=document.createElement("div");n.id="scroll-sentinel",n.className="scroll-sentinel",e.appendChild(n),Z=new IntersectionObserver(s=>{s[0].isIntersecting&&!q&&B&&G&&cn()},{rootMargin:"400px"}),Z.observe(n)}async function cn(){if(q||!B)return;q=!0,W++;const e=document.getElementById("scroll-sentinel");e&&(e.innerHTML='<div class="loading-more"><div class="spinner-sm"></div></div>');try{const t=await f.searchImages(G,{...I,page:W}),n=t.results;B=t.has_more,E=[...E,...n];const s=document.querySelector(".image-grid");if(s&&n.length>0){const a=E.length-n.length,r=n.map((i,o)=>de(i,a+o)).join("");s.insertAdjacentHTML("beforeend",r),s.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);ce(E[o])})})}e&&(e.innerHTML=B?"":'<div class="no-more-results">No more images</div>')}catch{e&&(e.innerHTML="")}finally{q=!1}}async function oe(e,t){var s;const n=document.getElementById("images-content");if(!(!n||!e)){q=!0,n.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const a=await f.searchImages(e,{...t,page:1,per_page:50}),r=a.results;if(B=a.has_more,E=r,ie=(s=a.related_searches)!=null&&s.length?a.related_searches:dn(e),an(ie),r.length===0){n.innerHTML=`<div class="py-8 text-secondary">No image results found for "<strong>${j(e)}</strong>"</div>`;return}n.innerHTML=`<div class="image-grid">${r.map((i,o)=>de(i,o)).join("")}</div>`,n.querySelectorAll(".image-card").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);ce(E[o])})}),ln()}catch(a){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${j(String(a))}</p>
      </div>
    `}finally{q=!1}}}function de(e,t){return`
    <div class="image-card" data-image-index="${t}">
      <div class="image-card-img">
        <img
          src="${_(e.thumbnail_url||e.url)}"
          alt="${_(e.title)}"
          loading="lazy"
          onerror="this.closest('.image-card').style.display='none'"
        />
      </div>
    </div>
  `}function j(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function _(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function dn(e){const t=e.toLowerCase().trim().split(/\s+/).filter(i=>i.length>1);if(t.length===0)return[];const n=[],s=["wallpaper","hd","4k","aesthetic","cute","beautiful","background","art","photography","design","illustration","vintage","modern","minimalist","colorful","dark","light"],a={cat:["kitten","cats playing","black cat","tabby cat","cat meme"],dog:["puppy","dogs playing","golden retriever","german shepherd","dog meme"],nature:["forest","mountains","ocean","sunset nature","flowers"],food:["dessert","healthy food","breakfast","dinner","food photography"],car:["sports car","luxury car","vintage car","car interior","supercar"],house:["modern house","interior design","living room","bedroom design","architecture"],city:["skyline","night city","urban photography","street photography","downtown"]},r=t.slice(0,2).join(" ");for(const i of s)!e.includes(i)&&n.length<4&&n.push(`${r} ${i}`);for(const[i,o]of Object.entries(a))if(t.some(l=>l.includes(i)||i.includes(l))){for(const l of o)!n.includes(l)&&n.length<8&&n.push(l);break}return t.length>=2&&n.length<8&&n.push(t.reverse().join(" ")),n.length<4&&n.push(`${r} images`,`${r} photos`,`best ${r}`),n.slice(0,8)}const un='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function hn(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${P({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${un}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${te({query:e,active:"videos"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="videos-content" class="max-w-[1200px] mx-auto px-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function pn(e,t){F(n=>{e.navigate(`/videos?q=${encodeURIComponent(n)}`)}),t&&ee(t),gn(t)}async function gn(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const n=await f.searchVideos(e),s=n.results;if(s.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${O(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} video results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        ${s.map(a=>vn(a)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${O(String(n))}</p>
      </div>
    `}}function vn(e){var r;const t=((r=e.thumbnail)==null?void 0:r.url)||"",n=e.views?mn(e.views):"",s=e.published?fn(e.published):"",a=[e.channel,n,s].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${J(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${J(t)}" alt="${J(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${O(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${J(e.url)}" target="_blank" rel="noopener">${O(e.title)}</a>
        </div>
        <div class="video-meta">${O(a)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${O(e.platform)}</div>`:""}
      </div>
    </div>
  `}function mn(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function fn(e){try{const t=new Date(e),s=new Date().getTime()-t.getTime(),a=Math.floor(s/(1e3*60*60*24));return a===0?"Today":a===1?"1 day ago":a<7?`${a} days ago`:a<30?`${Math.floor(a/7)} weeks ago`:a<365?`${Math.floor(a/30)} months ago`:`${Math.floor(a/365)} years ago`}catch{return e}}function O(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function J(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const yn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function wn(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${P({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${yn}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${te({query:e,active:"news"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="news-content" class="max-w-[800px] pl-[170px] pr-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function xn(e,t){F(n=>{e.navigate(`/news?q=${encodeURIComponent(n)}`)}),t&&ee(t),bn(t)}async function bn(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const n=await f.searchNews(e),s=n.results;if(s.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No news results found for "<strong>${D(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} news results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div>
        ${s.map(a=>$n(a)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load news results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${D(String(n))}</p>
      </div>
    `}}function $n(e){var s;const t=((s=e.thumbnail)==null?void 0:s.url)||"",n=e.published_date?kn(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${D(e.source||e.domain)}
          ${n?` · ${D(n)}`:""}
        </div>
        <div class="news-title">
          <a href="${xe(e.url)}" target="_blank" rel="noopener">${D(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${xe(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function kn(e){try{const t=new Date(e),s=new Date().getTime()-t.getTime(),a=Math.floor(s/(1e3*60*60)),r=Math.floor(s/(1e3*60*60*24));return a<1?"Just now":a<24?`${a}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function D(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function xe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Cn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Se='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>',In='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',En='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>',be='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',Ln='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',Sn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>',_n='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>',Bn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>',Mn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',Hn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>',An='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',Tn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',_e=[{id:"top",label:"Top Stories",icon:Tn},{id:"world",label:"World",icon:Ln},{id:"nation",label:"U.S.",icon:Se},{id:"business",label:"Business",icon:Sn},{id:"technology",label:"Technology",icon:_n},{id:"entertainment",label:"Entertainment",icon:Bn},{id:"sports",label:"Sports",icon:Mn},{id:"science",label:"Science",icon:Hn},{id:"health",label:"Health",icon:An}];function Nn(){const e=new Date().toLocaleDateString("en-US",{weekday:"long",month:"long",day:"numeric"});return`
    <div class="news-layout">
      <!-- Sidebar Navigation -->
      <nav class="news-sidebar">
        <div class="news-sidebar-header">
          <a href="/" data-link class="news-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
            <span class="news-logo-suffix">News</span>
          </a>
        </div>

        <div class="news-nav-section">
          <a href="/news-home" data-link class="news-nav-item active">
            ${Se}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${In}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${En}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${_e.map(t=>`
            <a href="/news-home?category=${t.id}" data-link class="news-nav-item" data-category="${t.id}">
              ${t.icon}
              <span>${t.label}</span>
            </a>
          `).join("")}
        </div>
      </nav>

      <!-- Main Content -->
      <main class="news-main">
        <!-- Header -->
        <header class="news-header">
          <div class="news-header-left">
            <button class="news-menu-btn" id="menu-toggle">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="3" y1="12" x2="21" y2="12"></line>
                <line x1="3" y1="6" x2="21" y2="6"></line>
                <line x1="3" y1="18" x2="21" y2="18"></line>
              </svg>
            </button>
            <a href="/" data-link class="news-logo-mobile">
              <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
              <span class="news-logo-suffix">News</span>
            </a>
          </div>

          <div class="news-search-container">
            ${P({size:"sm"})}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${be}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${Cn}
            </a>
          </div>
        </header>

        <!-- Content Area -->
        <div class="news-content" id="news-content">
          <div class="news-briefing">
            <h1 class="news-briefing-title">Your briefing</h1>
            <p class="news-briefing-date">${e}</p>
          </div>

          <!-- Loading State -->
          <div class="news-loading" id="news-loading">
            <div class="spinner"></div>
            <p>Loading your news...</p>
          </div>

          <!-- Top Stories Section -->
          <section class="news-section" id="top-stories-section" style="display: none;">
            <h2 class="news-section-title">Top stories</h2>
            <div class="news-grid" id="top-stories-grid"></div>
          </section>

          <!-- For You Section -->
          <section class="news-section" id="for-you-section" style="display: none;">
            <h2 class="news-section-title">For you</h2>
            <div class="news-list" id="for-you-list"></div>
          </section>

          <!-- Local News Section -->
          <section class="news-section" id="local-section" style="display: none;">
            <div class="news-section-header">
              <h2 class="news-section-title">
                ${be}
                <span id="local-title">Local news</span>
              </h2>
              <button class="news-text-btn" id="change-location-btn">Change location</button>
            </div>
            <div class="news-horizontal-scroll" id="local-news-scroll"></div>
          </section>

          <!-- Category Sections -->
          <div id="category-sections"></div>
        </div>
      </main>
    </div>
  `}function Rn(e,t){F(a=>{e.navigate(`/news?q=${encodeURIComponent(a)}`)});const n=document.getElementById("menu-toggle"),s=document.querySelector(".news-sidebar");n&&s&&n.addEventListener("click",()=>{s.classList.toggle("open")}),On()}async function On(e){const t=document.getElementById("news-loading"),n=document.getElementById("top-stories-section"),s=document.getElementById("for-you-section"),a=document.getElementById("local-section");try{const r=await f.newsHome();if(t&&(t.style.display="none"),n&&r.topStories.length>0){n.style.display="block";const o=document.getElementById("top-stories-grid");o&&(o.innerHTML=qn(r.topStories))}if(s&&r.forYou.length>0){s.style.display="block";const o=document.getElementById("for-you-list");o&&(o.innerHTML=r.forYou.slice(0,10).map(l=>Un(l)).join(""))}if(a&&r.localNews.length>0){a.style.display="block";const o=document.getElementById("local-news-scroll");o&&(o.innerHTML=r.localNews.map(l=>Be(l)).join(""))}const i=document.getElementById("category-sections");if(i&&r.categories){const o=Object.entries(r.categories).filter(([l,p])=>p&&p.length>0).map(([l,p])=>zn(l,p)).join("");i.innerHTML=o}Vn()}catch(r){t&&(t.innerHTML=`
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `),console.error("Failed to load news:",r)}}function qn(e){if(e.length===0)return"";const t=e[0],n=e.slice(1,3),s=e.slice(3,9);return`
    <div class="news-featured-row">
      ${jn(t)}
      <div class="news-secondary-col">
        ${n.map(a=>Pn(a)).join("")}
      </div>
    </div>
    <div class="news-grid-row">
      ${s.map(a=>Fn(a)).join("")}
    </div>
  `}function jn(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-featured">
      ${e.imageUrl?`<img class="news-card-image" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${e.id}')">${w(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${w(e.snippet)}</p>
        ${e.clusterId?`<a href="/news-home?story=${e.clusterId}" data-link class="news-full-coverage">Full coverage</a>`:""}
      </div>
    </article>
  `}function Pn(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
      </div>
      ${e.imageUrl?`<img class="news-card-thumb" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function Fn(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
      </div>
    </article>
  `}function Be(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-compact">
      ${e.imageUrl?`<img class="news-card-thumb-sm" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:'<div class="news-card-thumb-placeholder"></div>'}
      <div class="news-card-content">
        <span class="news-source-name">${w(e.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h4>
        <span class="news-time">${t}</span>
      </div>
    </article>
  `}function Un(e){const t=Y(e.publishedAt);return`
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
        <p class="news-list-snippet">${w(e.snippet)}</p>
      </div>
      ${e.imageUrl?`<img class="news-list-thumb" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function zn(e,t){const n=_e.find(s=>s.id===e);return n?`
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${n.icon}
          <span>${n.label}</span>
        </h2>
        <a href="/news-home?category=${e}" data-link class="news-text-btn">More ${n.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${t.slice(0,5).map(s=>Be(s)).join("")}
      </div>
    </section>
  `:""}function Y(e){try{const t=new Date(e),s=new Date().getTime()-t.getTime(),a=Math.floor(s/(1e3*60*60)),r=Math.floor(s/(1e3*60*60*24));return a<1?"Just now":a<24?`${a}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric"})}catch{return""}}function Vn(){document.querySelectorAll(".news-card a, .news-list-item a").forEach(e=>{e.addEventListener("click",function(){const t=this.closest(".news-card, .news-list-item");if(t){const n=t.getAttribute("data-article-id");n&&f.recordNewsRead({id:n,url:this.href,title:this.textContent||"",snippet:"",source:"",sourceUrl:"",publishedAt:"",category:"top",engines:[],score:1}).catch(()=>{})}})})}function w(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function x(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Dn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',Gn=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],Wn=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function Yn(){const e=R.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${Dn}
          </a>
          <h1 class="text-xl font-semibold text-primary">Settings</h1>
        </div>
      </header>

      <!-- Form -->
      <main class="max-w-[700px] mx-auto px-4 py-8">
        <form id="settings-form">
          <!-- Safe Search -->
          <div class="settings-section">
            <h3>Safe Search</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="safe_search" value="off" ${e.safe_search==="off"?"checked":""} />
                <span>Off</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="moderate" ${e.safe_search==="moderate"?"checked":""} />
                <span>Moderate</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="strict" ${e.safe_search==="strict"?"checked":""} />
                <span>Strict</span>
              </label>
            </div>
          </div>

          <!-- Results per page -->
          <div class="settings-section">
            <h3>Results per page</h3>
            <select name="results_per_page" class="settings-select">
              ${[10,20,30,50].map(t=>`<option value="${t}" ${e.results_per_page===t?"selected":""}>${t}</option>`).join("")}
            </select>
          </div>

          <!-- Region -->
          <div class="settings-section">
            <h3>Region</h3>
            <select name="region" class="settings-select">
              ${Gn.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${$e(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${Wn.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${$e(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Theme -->
          <div class="settings-section">
            <h3>Theme</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="theme" value="light" ${e.theme==="light"?"checked":""} />
                <span>Light</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="dark" ${e.theme==="dark"?"checked":""} />
                <span>Dark</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="system" ${e.theme==="system"?"checked":""} />
                <span>System</span>
              </label>
            </div>
          </div>

          <!-- Open in new tab -->
          <div class="settings-section">
            <h3>Behavior</h3>
            <label class="settings-label">
              <input type="checkbox" name="open_in_new_tab" ${e.open_in_new_tab?"checked":""} />
              <span>Open results in new tab</span>
            </label>
            <label class="settings-label">
              <input type="checkbox" name="show_thumbnails" ${e.show_thumbnails?"checked":""} />
              <span>Show thumbnails in results</span>
            </label>
          </div>

          <!-- Save Button -->
          <div class="flex items-center gap-4 pt-4">
            <button type="submit" id="settings-save-btn"
                    class="px-6 py-2.5 bg-blue text-white rounded-lg font-medium text-sm hover:bg-blue/90 transition-colors cursor-pointer">
              Save settings
            </button>
            <span id="settings-status" class="text-sm text-green hidden">Settings saved!</span>
          </div>
        </form>
      </main>
    </div>
  `}function Kn(e){const t=document.getElementById("settings-form"),n=document.getElementById("settings-status");t&&t.addEventListener("submit",async s=>{s.preventDefault();const a=new FormData(t),r={safe_search:a.get("safe_search")||"moderate",results_per_page:parseInt(a.get("results_per_page"))||10,region:a.get("region")||"auto",language:a.get("language")||"en",theme:a.get("theme")||"light",open_in_new_tab:a.has("open_in_new_tab"),show_thumbnails:a.has("show_thumbnails")};R.set({settings:r});try{await f.updateSettings(r)}catch{}n&&(n.classList.remove("hidden"),setTimeout(()=>{n.classList.add("hidden")},2e3))})}function $e(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Zn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',Jn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',Qn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Xn='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function es(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${Zn}
            </a>
            <h1 class="text-xl font-semibold text-primary">Search History</h1>
          </div>
          <button id="clear-all-btn" class="text-sm text-red hover:text-red/80 font-medium cursor-pointer hidden">
            Clear all
          </button>
        </div>
      </header>

      <!-- Content -->
      <main class="max-w-[700px] mx-auto px-4 py-6">
        <div id="history-content">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function ts(e){const t=document.getElementById("clear-all-btn");ns(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await f.clearHistory(),ue(),t.classList.add("hidden")}catch(n){console.error("Failed to clear history:",n)}})}async function ns(e){const t=document.getElementById("history-content"),n=document.getElementById("clear-all-btn");if(t)try{const s=await f.getHistory();if(s.length===0){ue();return}n&&n.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${s.map(a=>ss(a)).join("")}
      </div>
    `,as(e)}catch(s){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${le(String(s))}</p>
      </div>
    `}}function ss(e){const t=rs(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${ke(e.id)}">
      <span class="text-light flex-shrink-0">${Qn}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${le(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${le(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${ke(e.id)}" aria-label="Delete">
        ${Jn}
      </button>
    </div>
  `}function as(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async n=>{n.preventDefault(),n.stopPropagation();const s=t.dataset.deleteId||"",a=t.closest(".history-item");try{await f.deleteHistoryItem(s),a&&a.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){ue();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function ue(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${Xn}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function rs(e){try{const t=new Date(e),n=new Date,s=n.getTime()-t.getTime(),a=Math.floor(s/(1e3*60)),r=Math.floor(s/(1e3*60*60)),i=Math.floor(s/(1e3*60*60*24));return a<1?"Just now":a<60?`${a}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==n.getFullYear()?"numeric":void 0})}catch{return e}}function le(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ke(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const k=document.getElementById("app");if(!k)throw new Error("App container not found");const g=new Ae;g.addRoute("",(e,t)=>{k.innerHTML=Ge(),We(g)});g.addRoute("search",(e,t)=>{const n=t.q||"",s=t.time_range||"";k.innerHTML=Ft(n,s),Ut(g,n,t)});g.addRoute("images",(e,t)=>{const n=t.q||"";k.innerHTML=Qt(n),en(g,n)});g.addRoute("videos",(e,t)=>{const n=t.q||"";k.innerHTML=hn(n),pn(g,n)});g.addRoute("news",(e,t)=>{const n=t.q||"";k.innerHTML=wn(n),xn(g,n)});g.addRoute("news-home",(e,t)=>{k.innerHTML=Nn(),Rn(g)});g.addRoute("settings",(e,t)=>{k.innerHTML=Yn(),Kn()});g.addRoute("history",(e,t)=>{k.innerHTML=es(),ts(g)});g.setNotFound((e,t)=>{k.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;g.navigate(t.detail.path)});g.start();
