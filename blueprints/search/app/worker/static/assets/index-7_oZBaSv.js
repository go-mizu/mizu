var fe=Object.defineProperty;var xe=(e,t,n)=>t in e?fe(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var z=(e,t,n)=>xe(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const s of document.querySelectorAll('link[rel="modulepreload"]'))a(s);new MutationObserver(s=>{for(const r of s)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&a(i)}).observe(document,{childList:!0,subtree:!0});function n(s){const r={};return s.integrity&&(r.integrity=s.integrity),s.referrerPolicy&&(r.referrerPolicy=s.referrerPolicy),s.crossOrigin==="use-credentials"?r.credentials="include":s.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function a(s){if(s.ep)return;s.ep=!0;const r=n(s);fetch(s.href,r)}})();class ye{constructor(){z(this,"routes",[]);z(this,"currentPath","");z(this,"notFoundRenderer",null)}addRoute(t,n){const a=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:a,renderer:n})}setNotFound(t){this.notFoundRenderer=t}navigate(t,n=!1){t!==this.currentPath&&(n?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const n=t.target.closest("a[data-link]");if(n){t.preventDefault();const a=n.getAttribute("href");a&&this.navigate(a)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),n=t.pathname,a=we(t.search);this.currentPath=n+t.search;for(const s of this.routes){const r=be(s.segments,n);if(r!==null){s.renderer(r,a);return}}this.notFoundRenderer&&this.notFoundRenderer({},a)}}function be(e,t){const n=t.split("/").filter(Boolean);if(e.length===0&&n.length===0)return{};if(e.length!==n.length)return null;const a={};for(let s=0;s<e.length;s++){const r=e[s],i=n[s];if(r.startsWith(":"))a[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return a}function we(e){const t={};return new URLSearchParams(e).forEach((a,s)=>{t[s]=a}),t}const G="/api";async function p(e,t){let n=`${G}${e}`;if(t){const s=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&s.set(i,o)});const r=s.toString();r&&(n+=`?${r}`)}const a=await fetch(n);if(!a.ok)throw new Error(`API error: ${a.status} ${a.statusText}`);return a.json()}async function Y(e,t){const n=await fetch(`${G}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function $e(e,t){const n=await fetch(`${G}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function D(e){const t=await fetch(`${G}${e}`,{method:"DELETE"});if(!t.ok)throw new Error(`API error: ${t.status} ${t.statusText}`);return t.json()}function Z(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.time_range&&(n.time_range=t.time_range),t.region&&(n.region=t.region),t.language&&(n.language=t.language),t.safe_search&&(n.safe_search=t.safe_search),t.site&&(n.site=t.site),t.exclude_site&&(n.exclude_site=t.exclude_site),t.lens&&(n.lens=t.lens)),n}const x={search(e,t){return p("/search",Z(e,t))},searchImages(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(n.size=t.size),t.color&&t.color!=="any"&&(n.color=t.color),t.type&&t.type!=="any"&&(n.type=t.type),t.aspect&&t.aspect!=="any"&&(n.aspect=t.aspect),t.time&&t.time!=="any"&&(n.time=t.time),t.rights&&t.rights!=="any"&&(n.rights=t.rights),t.filetype&&t.filetype!=="any"&&(n.filetype=t.filetype),t.safe&&(n.safe=t.safe)),p("/search/images",n)},reverseImageSearch(e){return Y("/search/images/reverse",{url:e})},searchVideos(e,t){return p("/search/videos",Z(e,t))},searchNews(e,t){return p("/search/news",Z(e,t))},suggest(e){return p("/suggest",{q:e})},trending(){return p("/suggest/trending")},calculate(e){return p("/instant/calculate",{q:e})},convert(e){return p("/instant/convert",{q:e})},currency(e){return p("/instant/currency",{q:e})},weather(e){return p("/instant/weather",{q:e})},define(e){return p("/instant/define",{q:e})},time(e){return p("/instant/time",{q:e})},knowledge(e){return p(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return p("/preferences")},setPreference(e,t){return Y("/preferences",{domain:e,action:t})},deletePreference(e){return D(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return p("/lenses")},createLens(e){return Y("/lenses",e)},deleteLens(e){return D(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return p("/history")},clearHistory(){return D("/history")},deleteHistoryItem(e){return D(`/history/${encodeURIComponent(e)}`)},getSettings(){return p("/settings")},updateSettings(e){return $e("/settings",e)},getBangs(){return p("/bangs")},parseBang(e){return p("/bangs/parse",{q:e})},getRelated(e){return p("/related",{q:e})}};function ke(e){let t={...e};const n=new Set;return{get(){return t},set(a){t={...t,...a},n.forEach(s=>s(t))},subscribe(a){return n.add(a),()=>{n.delete(a)}}}}const me="mizu_search_state";function Ie(){try{const e=localStorage.getItem(me);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const A=ke(Ie());A.subscribe(e=>{try{localStorage.setItem(me,JSON.stringify(e))}catch{}});function W(e){const t=A.get(),n=[e,...t.recentSearches.filter(a=>a!==e)].slice(0,20);A.set({recentSearches:n})}const ve='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',_e='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Ce='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',Ee='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Le='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',Se='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function F(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",n=e.initialValue?Be(e.initialValue):"",a=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${ve}</span>
        <input
          id="search-input"
          type="text"
          value="${n}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${a}" type="button" aria-label="Clear">
          ${_e}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${Ce}
        </button>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${Ee}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function U(e){const t=document.getElementById("search-input"),n=document.getElementById("search-clear-btn"),a=document.getElementById("autocomplete-dropdown"),s=document.getElementById("search-box-wrapper");if(!t||!n||!a||!s)return;let r=null,i=[],o=-1,d=!1;function h(u){if(i=u,o=-1,u.length===0){g();return}d=!0,a.innerHTML=u.map((l,m)=>`
        <div class="autocomplete-item ${m===o?"active":""}" data-index="${m}">
          <span class="suggestion-icon">${l.icon}</span>
          ${l.prefix?`<span class="bang-trigger">${se(l.prefix)}</span>`:""}
          <span>${se(l.text)}</span>
        </div>
      `).join(""),a.classList.remove("hidden"),a.classList.add("has-items"),a.querySelectorAll(".autocomplete-item").forEach(l=>{l.addEventListener("mousedown",m=>{m.preventDefault();const y=parseInt(l.dataset.index||"0");f(y)}),l.addEventListener("mouseenter",()=>{const m=parseInt(l.dataset.index||"0");S(m)})})}function g(){d=!1,a.classList.add("hidden"),a.classList.remove("has-items"),a.innerHTML="",i=[],o=-1}function S(u){o=u,a.querySelectorAll(".autocomplete-item").forEach((l,m)=>{l.classList.toggle("active",m===u)})}function f(u){const l=i[u];l&&(l.type==="bang"&&l.prefix?(t.value=l.prefix+" ",t.focus(),w(l.prefix+" ")):(t.value=l.text,g(),b(l.text)))}function b(u){const l=u.trim();l&&(g(),e(l))}async function w(u){const l=u.trim();if(!l){j();return}if(l.startsWith("!"))try{const y=(await x.getBangs()).filter(B=>B.trigger.startsWith(l)||B.name.toLowerCase().includes(l.slice(1).toLowerCase())).slice(0,8);if(y.length>0){h(y.map(B=>({text:B.name,type:"bang",icon:Se,prefix:B.trigger})));return}}catch{}try{const m=await x.suggest(l);if(t.value.trim()!==l)return;const y=m.map(B=>({text:B.text,type:"suggestion",icon:ve}));y.length===0?j(l):h(y)}catch{j(l)}}function j(u){let m=A.get().recentSearches;if(u&&(m=m.filter(y=>y.toLowerCase().includes(u.toLowerCase()))),m.length===0){g();return}h(m.slice(0,8).map(y=>({text:y,type:"recent",icon:Le})))}t.addEventListener("input",()=>{const u=t.value;n.classList.toggle("hidden",u.length===0),r&&clearTimeout(r),r=setTimeout(()=>w(u),150)}),t.addEventListener("focus",()=>{t.value.trim()?w(t.value):j()}),t.addEventListener("keydown",u=>{if(!d){if(u.key==="Enter"){b(t.value);return}if(u.key==="ArrowDown"){w(t.value);return}return}switch(u.key){case"ArrowDown":u.preventDefault(),S(Math.min(o+1,i.length-1));break;case"ArrowUp":u.preventDefault(),S(Math.max(o-1,-1));break;case"Enter":u.preventDefault(),o>=0?f(o):b(t.value);break;case"Escape":g();break;case"Tab":g();break}}),t.addEventListener("blur",()=>{setTimeout(()=>g(),200)}),n.addEventListener("click",()=>{t.value="",n.classList.add("hidden"),t.focus(),j()})}function se(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Be(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Me=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],He=[{label:"Calculator",icon:Ne(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:je(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:Oe(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:qe(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:Pe(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:Fe(),query:"define serendipity",color:"bg-red/10 text-red"}];function Te(){return`
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
          ${F({size:"lg",autofocus:!0})}
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
          ${Me.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${J(e.trigger)}</span>
              <span class="text-tertiary ml-1">${J(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${He.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${Re(e.query)}">
                ${e.icon}
                <span>${J(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${Ue()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${ze()}
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
  `}function Ae(e){U(a=>{e.navigate(`/search?q=${encodeURIComponent(a)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var r;const a=document.getElementById("search-input"),s=(r=a==null?void 0:a.value)==null?void 0:r.trim();s&&e.navigate(`/search?q=${encodeURIComponent(s)}`)});const n=document.getElementById("home-lucky-btn");n==null||n.addEventListener("click",()=>{var r;const a=document.getElementById("search-input"),s=(r=a==null?void 0:a.value)==null?void 0:r.trim();s&&e.navigate(`/search?q=${encodeURIComponent(s)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(a=>{a.addEventListener("click",()=>{const s=a.dataset.bang||"",r=document.getElementById("search-input");r&&(r.value=s+" ",r.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(a=>{a.addEventListener("click",()=>{const s=a.dataset.query||"";s&&e.navigate(`/search?q=${encodeURIComponent(s)}`)})})}function J(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Re(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Ne(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function je(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function Oe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function qe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function Pe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function Fe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function Ue(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function ze(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}const De='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',Ve='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',Ge='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',We='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>';function Ke(e,t){const n=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,a=Je(e.url),s=e.published?Qe(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${_(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="result-sitelinks">
        ${e.sitelinks.map(d=>`<a href="${_(d.url)}" target="_blank" rel="noopener">${C(d.title)}</a>`).join("")}
       </div>`:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${_(e.domain)}">
      <div class="result-url">
        <img class="favicon" src="${_(n)}" alt="" width="18" height="18" loading="lazy" onerror="this.style.display='none'" />
        <div>
          <span class="text-sm">${C(e.domain)}</span>
          <span class="breadcrumbs">${a}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${_(e.url)}" target="_blank" rel="noopener">${C(e.title)}</a>
          </div>
          ${s?`<span class="result-date">${C(s)} -- </span>`:""}
          <div class="result-snippet">${r}</div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${De}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function Ye(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const n=e.dataset.menuIndex,a=document.getElementById(`domain-menu-${n}`),s=e.closest(".search-result"),r=(s==null?void 0:s.dataset.domain)||"";if(!a)return;if(!a.classList.contains("hidden")){a.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),a.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${_(r)}">
          ${Ve}
          <span>Boost ${C(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${_(r)}">
          ${Ge}
          <span>Lower ${C(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${_(r)}">
          ${We}
          <span>Block ${C(r)}</span>
        </button>
      `,a.classList.remove("hidden"),a.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const d=o.dataset.action||"",h=o.dataset.domain||"";try{await x.setPreference(h,d),a.classList.add("hidden"),Ze(`${d.charAt(0).toUpperCase()+d.slice(1)}ed ${h}`)}catch(g){console.error("Failed to set preference:",g)}})});const i=o=>{!a.contains(o.target)&&o.target!==e&&(a.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function Ze(e){const t=document.getElementById("toast");t&&t.remove();const n=document.createElement("div");n.id="toast",n.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",n.textContent=e,document.body.appendChild(n),setTimeout(()=>{n.style.opacity="0",setTimeout(()=>n.remove(),300)},2e3)}function Je(e){try{const n=new URL(e).pathname.split("/").filter(Boolean);return n.length===0?"":" > "+n.map(a=>C(decodeURIComponent(a))).join(" > ")}catch{return""}}function Qe(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60*24));return s===0?"Today":s===1?"1 day ago":s<7?`${s} days ago`:s<30?`${Math.floor(s/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function C(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function _(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Xe='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',et='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',tt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',nt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',at='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',st='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',rt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',it='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function ot(e){switch(e.type){case"calculator":return lt(e);case"unit_conversion":return ct(e);case"currency":return dt(e);case"weather":return ut(e);case"definition":return pt(e);case"time":return ht(e);default:return gt(e)}}function lt(e){const t=e.data||{},n=t.expression||e.query||"",a=t.formatted||t.result||e.result||"";return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${Xe}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${c(n)} = ${c(String(a))}</div>
    </div>
  `}function ct(e){const t=e.data||{},n=t.from_value??"",a=t.from_unit??"",s=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${et}
        <span class="instant-type">Unit Conversion${i?` -- ${c(i)}`:""}</span>
      </div>
      <div class="instant-result">${c(String(n))} ${c(a)} = ${c(String(s))} ${c(r)}</div>
      ${t.formatted?`<div class="instant-sub">${c(t.formatted)}</div>`:""}
    </div>
  `}function dt(e){const t=e.data||{},n=t.from_value??"",a=t.from_currency??"",s=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${tt}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${c(String(n))} ${c(a)} = ${c(String(s))} ${c(r)}</div>
      ${i?`<div class="instant-sub">1 ${c(a)} = ${c(String(i))} ${c(r)}</div>`:""}
    </div>
  `}function ut(e){const t=e.data||{},n=t.location||"",a=t.temperature??"",s=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=nt;return s.includes("cloud")||s.includes("overcast")?o=at:(s.includes("rain")||s.includes("drizzle")||s.includes("storm"))&&(o=st),`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">Weather</div>
      <div class="flex items-center gap-4 mb-3">
        <div>${o}</div>
        <div>
          <div class="text-2xl font-semibold text-primary">${c(String(a))}&deg;</div>
          <div class="text-secondary capitalize">${c(t.condition||"")}</div>
        </div>
      </div>
      <div class="text-sm font-medium text-primary mb-2">${c(n)}</div>
      <div class="flex gap-6 text-sm text-tertiary">
        ${r?`<span>Humidity: ${c(r)}</span>`:""}
        ${i?`<span>Wind: ${c(i)}</span>`:""}
      </div>
    </div>
  `}function pt(e){const t=e.data||{},n=t.word||e.query||"",a=t.phonetic||"",s=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[];return`
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${rt}
        <span class="instant-type">Definition</span>
      </div>
      <div class="flex items-baseline gap-3 mb-1">
        <span class="text-xl font-semibold text-primary">${c(n)}</span>
        ${a?`<span class="text-tertiary text-sm">${c(a)}</span>`:""}
      </div>
      ${s?`<div class="text-sm italic text-secondary mb-2">${c(s)}</div>`:""}
      ${r.length>0?`<ol class="list-decimal list-inside space-y-1 text-sm text-snippet mb-3">
              ${r.map(o=>`<li>${c(o)}</li>`).join("")}
             </ol>`:""}
      ${i.length>0?`<div class="text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(o=>c(o)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function ht(e){const t=e.data||{},n=t.location||"",a=t.time||"",s=t.date||"",r=t.timezone||"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${it}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${c(n)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${c(a)}</div>
      <div class="text-sm text-tertiary">${c(s)}</div>
      ${r?`<div class="text-xs text-light mt-1">${c(r)}</div>`:""}
    </div>
  `}function gt(e){return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${c(e.type)}</div>
      <div class="instant-result">${c(e.result)}</div>
    </div>
  `}function c(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const mt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function vt(e){const t=e.image?`<img class="kp-image" src="${Q(e.image)}" alt="${Q(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",n=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${M(r.label)}</td>
                <td class="fact-value">${M(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",a=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${Q(r.url)}" target="_blank" rel="noopener">
              ${mt}
              <span>${M(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",s=e.source?`<div class="kp-source">Source: ${M(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${M(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${M(e.subtitle)}</div>`:""}
      <div class="kp-description">${M(e.description)}</div>
      ${n}
      ${a}
      ${s}
    </div>
  `}function M(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Q(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ft='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',xt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function yt(e){const{currentPage:t,hasMore:n,totalResults:a,perPage:s}=e,r=Math.min(Math.ceil(a/s),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const d=[];for(let f=i;f<=o;f++)d.push(f);const h=bt(t),g=t<=1?"disabled":"",S=!n&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${h}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${g}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${ft}
          </button>
          ${d.map(f=>`
            <button class="pagination-btn ${f===t?"active":""}" data-page="${f}">
              ${f}
            </button>
          `).join("")}
          <button class="pagination-btn ${S}" data-page="${t+1}" ${!n&&t>=r?"disabled":""} aria-label="Next page">
            ${xt}
          </button>
        </div>
      </div>
    </div>
  `}function bt(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],n=["M","i","z","u"],a=Math.min(e-1,6);let s=[n[0]];for(let r=0;r<1+a;r++)s.push("i");s.push("z");for(let r=0;r<1+a;r++)s.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${s.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function wt(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(n=>{n.addEventListener("click",()=>{const a=parseInt(n.dataset.page||"1");isNaN(a)||n.disabled||(e(a),window.scrollTo({top:0,behavior:"smooth"}))})})}const $t='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',kt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',It='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',_t='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>';function K(e){const{query:t,active:n}=e,a=encodeURIComponent(t);return`
    <div class="search-tabs" id="search-tabs">
      ${[{id:"all",label:"All",icon:$t,href:`/search?q=${a}`},{id:"images",label:"Images",icon:kt,href:`/images?q=${a}`},{id:"videos",label:"Videos",icon:It,href:`/videos?q=${a}`},{id:"news",label:"News",icon:_t,href:`/news?q=${a}`}].map(r=>`
        <a class="search-tab ${r.id===n?"active":""}" href="${r.href}" data-link data-tab="${r.id}">
          ${r.icon}
          <span>${r.label}</span>
        </a>
      `).join("")}
    </div>
  `}const Ct='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Et='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',re=[{value:"",label:"Any time"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}];function Lt(e,t){var s;const n=((s=re.find(r=>r.value===t))==null?void 0:s.label)||"Any time",a=t!=="";return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Ct}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          <div class="flex items-center gap-2">
            ${K({query:e,active:"all"})}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${a?"active-filter":""}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${H(n)}</span>
                ${Et}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${re.map(r=>`
                  <button class="time-filter-option ${r.value===t?"active":""}" data-time-range="${r.value}">
                    ${H(r.label)}
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
  `}function St(e,t,n){const a=parseInt(n.page||"1"),s=n.time_range||"",r=A.get().settings;U(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),Bt(e,t),t&&W(t),Mt(e,t,a,s,r.results_per_page)}function Bt(e,t,n){const a=document.getElementById("time-filter-btn"),s=document.getElementById("time-filter-dropdown");!a||!s||(a.addEventListener("click",r=>{r.stopPropagation(),s.classList.toggle("hidden")}),s.querySelectorAll(".time-filter-option").forEach(r=>{r.addEventListener("click",()=>{const i=r.dataset.timeRange||"";s.classList.add("hidden");let o=`/search?q=${encodeURIComponent(t)}`;i&&(o+=`&time_range=${i}`),e.navigate(o)})}),document.addEventListener("click",r=>{!s.contains(r.target)&&r.target!==a&&s.classList.add("hidden")}))}async function Mt(e,t,n,a,s){const r=document.getElementById("search-content");if(!(!r||!t))try{const i=await x.search(t,{page:n,per_page:s,time_range:a||void 0});if(i.redirect){window.location.href=i.redirect;return}Ht(r,e,i,t,n,a)}catch(i){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${H(String(i))}</p>
      </div>
    `}}function Ht(e,t,n,a,s,r){const i=n.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(n.corrected_query)}" data-link class="text-link font-medium">${H(n.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(a)}&exact=1" data-link class="text-link">${H(a)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${Tt(n.total_results)} results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,d=n.instant_answer?ot(n.instant_answer):"",h=n.results.length>0?n.results.map((b,w)=>Ke(b,w)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${H(a)}</strong>"</div>`,g=n.related_searches&&n.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${n.related_searches.map(b=>`
            <a href="/search?q=${encodeURIComponent(b)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${H(b)}
            </a>
          `).join("")}
        </div>
      </div>
    `:"",S=yt({currentPage:s,hasMore:n.has_more,totalResults:n.total_results,perPage:n.per_page}),f=n.knowledge_panel?vt(n.knowledge_panel):"";e.innerHTML=`
    <div class="flex gap-8">
      <div class="flex-1 min-w-0">
        ${i}
        ${o}
        ${d}
        ${h}
        ${g}
        ${S}
      </div>
      ${f?`<aside class="hidden lg:block flex-shrink-0 w-[360px] pt-2">${f}</aside>`:""}
    </div>
  `,Ye(),wt(b=>{let w=`/search?q=${encodeURIComponent(a)}&page=${b}`;r&&(w+=`&time_range=${r}`),t.navigate(w)})}function Tt(e){return e.toLocaleString("en-US")}function H(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const At='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Rt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',ie='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',oe='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';let q="",$={},P=1,N=!1,E=!0,k=[];function Nt(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1400px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px] flex items-center gap-2">
            ${F({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${Rt}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${At}
          </a>
        </div>
        <div class="max-w-[1400px] pl-[170px]">
          ${K({query:e,active:"images"})}
        </div>
        <!-- Filter toolbar -->
        <div id="filter-toolbar" class="max-w-[1400px] px-4 py-2 flex flex-wrap gap-2 items-center border-t border-border/50">
          ${jt()}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 max-w-[1400px] mx-auto px-4 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-panel-content">
            <button id="preview-close" class="preview-close" aria-label="Close">${ie}</button>
            <div id="preview-image-container" class="preview-image-container">
              <img id="preview-image" src="" alt="" />
            </div>
            <div id="preview-details" class="preview-details"></div>
          </div>
        </div>
      </main>

      <!-- Reverse image search modal -->
      <div id="reverse-modal" class="modal hidden">
        <div class="modal-content">
          <div class="modal-header">
            <h2>Search by image</h2>
            <button id="reverse-modal-close" class="modal-close">${ie}</button>
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
  `}function jt(){return[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Rights",options:["any","creative_commons","commercial"]}].map(t=>`
    <select id="filter-${t.id}" class="filter-select" data-filter="${t.id}">
      ${t.options.map(n=>`<option value="${n}">${Ot(t.id,n)}</option>`).join("")}
    </select>
  `).join("")+`
    <button id="clear-filters" class="filter-clear hidden">Clear filters</button>
  `}function Ot(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function qt(e,t){q=t,$={},P=1,k=[],E=!0,U(n=>{e.navigate(`/images?q=${encodeURIComponent(n)}`)}),t&&W(t),Pt(),Ft(),Ut(),zt(),X(t,$)}function Pt(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-select").forEach(a=>{a.addEventListener("change",()=>{const s=a.dataset.filter,r=a.value;r==="any"?delete $[s]:$[s]=r,P=1,k=[],E=!0,le(),X(q,$)})});const n=document.getElementById("clear-filters");n&&n.addEventListener("click",()=>{$={},P=1,k=[],E=!0,t.querySelectorAll(".filter-select").forEach(a=>{a.value="any"}),le(),X(q,$)})}function le(){const e=document.getElementById("clear-filters");if(!e)return;const t=Object.keys($).length>0;e.classList.toggle("hidden",!t)}function Ft(e){const t=document.getElementById("reverse-search-btn"),n=document.getElementById("reverse-modal"),a=document.getElementById("reverse-modal-close"),s=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!n||(t.addEventListener("click",()=>{n.classList.remove("hidden")}),a==null||a.addEventListener("click",()=>{n.classList.add("hidden")}),n.addEventListener("click",d=>{d.target===n&&n.classList.add("hidden")}),s&&(s.addEventListener("dragover",d=>{d.preventDefault(),s.classList.add("drag-over")}),s.addEventListener("dragleave",()=>{s.classList.remove("drag-over")}),s.addEventListener("drop",d=>{var g;d.preventDefault(),s.classList.remove("drag-over");const h=(g=d.dataTransfer)==null?void 0:g.files;h&&h[0]&&(ce(h[0]),n.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(ce(r.files[0]),n.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const d=i.value.trim();d&&(de(d),n.classList.add("hidden"))}),i.addEventListener("keydown",d=>{if(d.key==="Enter"){const h=i.value.trim();h&&(de(h),n.classList.add("hidden"))}})))}async function ce(e,t){alert("Image upload coming soon. Please use the URL option for now.")}async function de(e,t){const n=document.getElementById("images-content");if(n){n.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const a=await x.reverseImageSearch(e);n.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${T(e)}" alt="Query image" class="query-image" />
        </div>

        ${a.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${a.similar_images.length})</h3>
            <div class="image-grid">
              ${a.similar_images.map((s,r)=>ne(s,r)).join("")}
            </div>
          </div>
        `:`
          <div class="py-8 text-secondary">No similar images found.</div>
        `}
      </div>
    `,n.querySelectorAll(".image-card").forEach(s=>{s.addEventListener("click",()=>{const r=parseInt(s.dataset.imageIndex||"0",10);te(a.similar_images[r])})})}catch(a){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${L(String(a))}</p>
      </div>
    `}}}function Ut(){const e=document.getElementById("preview-close");e&&e.addEventListener("click",ue),document.addEventListener("keydown",t=>{t.key==="Escape"&&ue()})}function te(e){const t=document.getElementById("preview-panel"),n=document.getElementById("preview-image"),a=document.getElementById("preview-details");!t||!n||!a||(n.src=e.url,n.alt=e.title,a.innerHTML=`
    <h3 class="preview-title">${L(e.title||"Untitled")}</h3>
    <p class="preview-dimensions">${e.width} x ${e.height} ${e.format?`- ${e.format.toUpperCase()}`:""}</p>
    <p class="preview-source">${L(e.source_domain)}</p>
    <div class="preview-actions">
      <a href="${T(e.url)}" target="_blank" class="preview-btn">View image ${oe}</a>
      <a href="${T(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">Visit page ${oe}</a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden")}function ue(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function zt(){const e=document.createElement("div");e.id="scroll-sentinel",e.style.height="1px";const t=new IntersectionObserver(n=>{n[0].isIntersecting&&!N&&E&&q&&Dt()},{rootMargin:"200px"});setTimeout(()=>{const n=document.getElementById("images-content");if(n){const a=document.getElementById("scroll-sentinel");a&&a.remove(),n.appendChild(e),t.observe(e)}},100)}async function Dt(){if(!(N||!E)){N=!0,P++;try{const e=await x.searchImages(q,{...$,page:P}),t=e.results;E=e.has_more,k=[...k,...t];const n=document.querySelector(".image-grid");if(n&&t.length>0){const a=k.length-t.length,s=t.map((i,o)=>ne(i,a+o)).join("");n.insertAdjacentHTML("beforeend",s),n.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);te(k[o])})})}if(!E){const a=document.getElementById("scroll-sentinel");a&&(a.innerHTML='<div class="text-center text-tertiary py-4 text-sm">No more images</div>')}}catch{}finally{N=!1}}}async function X(e,t){const n=document.getElementById("images-content");if(!(!n||!e)){N=!0;try{const a=await x.searchImages(e,{...t,page:1,per_page:30}),s=a.results;if(E=a.has_more,k=s,s.length===0){n.innerHTML=`
        <div class="py-8 text-secondary">No image results found for "<strong>${L(e)}</strong>"</div>
      `;return}const r=Object.entries(t).filter(([i,o])=>o&&o!=="any").map(([i,o])=>`${i}: ${o}`).join(", ");n.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${a.total_results.toLocaleString()} image results (${(a.search_time_ms/1e3).toFixed(2)} seconds)
        ${r?`<span class="ml-2 text-blue">Filters: ${L(r)}</span>`:""}
      </div>
      <div class="image-grid">
        ${s.map((i,o)=>ne(i,o)).join("")}
      </div>
    `,n.querySelectorAll(".image-card").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);te(k[o])})})}catch(a){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${L(String(a))}</p>
      </div>
    `}finally{N=!1}}}function ne(e,t){return`
    <div class="image-card" data-image-index="${t}" data-full-url="${T(e.url)}" data-source-url="${T(e.source_url)}">
      <img
        src="${T(e.thumbnail_url||e.url)}"
        alt="${T(e.title)}"
        loading="lazy"
        onerror="this.parentElement.style.display='none'"
      />
      <div class="image-info">
        <div class="image-title">${L(e.title||"")}</div>
        <div class="image-source">${L(e.source_domain)}</div>
        ${e.width&&e.height?`<div class="image-dimensions">${e.width} x ${e.height}</div>`:""}
      </div>
    </div>
  `}function L(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function T(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Vt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Gt(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Vt}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${K({query:e,active:"videos"})}
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
  `}function Wt(e,t){U(n=>{e.navigate(`/videos?q=${encodeURIComponent(n)}`)}),t&&W(t),Kt(t)}async function Kt(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const n=await x.searchVideos(e),a=n.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${R(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} video results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        ${a.map(s=>Yt(s)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${R(String(n))}</p>
      </div>
    `}}function Yt(e){var r;const t=((r=e.thumbnail)==null?void 0:r.url)||"",n=e.views?Zt(e.views):"",a=e.published?Jt(e.published):"",s=[e.channel,n,a].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${V(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${V(t)}" alt="${V(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${R(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${V(e.url)}" target="_blank" rel="noopener">${R(e.title)}</a>
        </div>
        <div class="video-meta">${R(s)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${R(e.platform)}</div>`:""}
      </div>
    </div>
  `}function Zt(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function Jt(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60*24));return s===0?"Today":s===1?"1 day ago":s<7?`${s} days ago`:s<30?`${Math.floor(s/7)} weeks ago`:s<365?`${Math.floor(s/30)} months ago`:`${Math.floor(s/365)} years ago`}catch{return e}}function R(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function V(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Qt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Xt(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Qt}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${K({query:e,active:"news"})}
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
  `}function en(e,t){U(n=>{e.navigate(`/news?q=${encodeURIComponent(n)}`)}),t&&W(t),tn(t)}async function tn(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const n=await x.searchNews(e),a=n.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No news results found for "<strong>${O(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} news results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div>
        ${a.map(s=>nn(s)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load news results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${O(String(n))}</p>
      </div>
    `}}function nn(e){var a;const t=((a=e.thumbnail)==null?void 0:a.url)||"",n=e.published_date?an(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${O(e.source||e.domain)}
          ${n?` · ${O(n)}`:""}
        </div>
        <div class="news-title">
          <a href="${pe(e.url)}" target="_blank" rel="noopener">${O(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${pe(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function an(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return s<1?"Just now":s<24?`${s}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function O(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function pe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const sn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',rn=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],on=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function ln(){const e=A.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${sn}
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
              ${rn.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${he(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${on.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${he(t.label)}</option>`).join("")}
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
  `}function cn(e){const t=document.getElementById("settings-form"),n=document.getElementById("settings-status");t&&t.addEventListener("submit",async a=>{a.preventDefault();const s=new FormData(t),r={safe_search:s.get("safe_search")||"moderate",results_per_page:parseInt(s.get("results_per_page"))||10,region:s.get("region")||"auto",language:s.get("language")||"en",theme:s.get("theme")||"light",open_in_new_tab:s.has("open_in_new_tab"),show_thumbnails:s.has("show_thumbnails")};A.set({settings:r});try{await x.updateSettings(r)}catch{}n&&(n.classList.remove("hidden"),setTimeout(()=>{n.classList.add("hidden")},2e3))})}function he(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const dn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',un='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',pn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',hn='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function gn(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${dn}
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
  `}function mn(e){const t=document.getElementById("clear-all-btn");vn(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await x.clearHistory(),ae(),t.classList.add("hidden")}catch(n){console.error("Failed to clear history:",n)}})}async function vn(e){const t=document.getElementById("history-content"),n=document.getElementById("clear-all-btn");if(t)try{const a=await x.getHistory();if(a.length===0){ae();return}n&&n.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${a.map(s=>fn(s)).join("")}
      </div>
    `,xn(e)}catch(a){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${ee(String(a))}</p>
      </div>
    `}}function fn(e){const t=yn(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${ge(e.id)}">
      <span class="text-light flex-shrink-0">${pn}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${ee(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${ee(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${ge(e.id)}" aria-label="Delete">
        ${un}
      </button>
    </div>
  `}function xn(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async n=>{n.preventDefault(),n.stopPropagation();const a=t.dataset.deleteId||"",s=t.closest(".history-item");try{await x.deleteHistoryItem(a),s&&s.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){ae();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function ae(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${hn}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function yn(e){try{const t=new Date(e),n=new Date,a=n.getTime()-t.getTime(),s=Math.floor(a/(1e3*60)),r=Math.floor(a/(1e3*60*60)),i=Math.floor(a/(1e3*60*60*24));return s<1?"Just now":s<60?`${s}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==n.getFullYear()?"numeric":void 0})}catch{return e}}function ee(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ge(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const I=document.getElementById("app");if(!I)throw new Error("App container not found");const v=new ye;v.addRoute("",(e,t)=>{I.innerHTML=Te(),Ae(v)});v.addRoute("search",(e,t)=>{const n=t.q||"",a=t.time_range||"";I.innerHTML=Lt(n,a),St(v,n,t)});v.addRoute("images",(e,t)=>{const n=t.q||"";I.innerHTML=Nt(n),qt(v,n)});v.addRoute("videos",(e,t)=>{const n=t.q||"";I.innerHTML=Gt(n),Wt(v,n)});v.addRoute("news",(e,t)=>{const n=t.q||"";I.innerHTML=Xt(n),en(v,n)});v.addRoute("settings",(e,t)=>{I.innerHTML=ln(),cn()});v.addRoute("history",(e,t)=>{I.innerHTML=gn(),mn(v)});v.setNotFound((e,t)=>{I.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;v.navigate(t.detail.path)});v.start();
