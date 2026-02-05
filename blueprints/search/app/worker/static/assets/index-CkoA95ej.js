var be=Object.defineProperty;var we=(e,t,n)=>t in e?be(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n;var D=(e,t,n)=>we(e,typeof t!="symbol"?t+"":t,n);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const s of document.querySelectorAll('link[rel="modulepreload"]'))a(s);new MutationObserver(s=>{for(const r of s)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&a(i)}).observe(document,{childList:!0,subtree:!0});function n(s){const r={};return s.integrity&&(r.integrity=s.integrity),s.referrerPolicy&&(r.referrerPolicy=s.referrerPolicy),s.crossOrigin==="use-credentials"?r.credentials="include":s.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function a(s){if(s.ep)return;s.ep=!0;const r=n(s);fetch(s.href,r)}})();class $e{constructor(){D(this,"routes",[]);D(this,"currentPath","");D(this,"notFoundRenderer",null)}addRoute(t,n){const a=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:a,renderer:n})}setNotFound(t){this.notFoundRenderer=t}navigate(t,n=!1){t!==this.currentPath&&(n?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const n=t.target.closest("a[data-link]");if(n){t.preventDefault();const a=n.getAttribute("href");a&&this.navigate(a)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),n=t.pathname,a=Ie(t.search);this.currentPath=n+t.search;for(const s of this.routes){const r=ke(s.segments,n);if(r!==null){s.renderer(r,a);return}}this.notFoundRenderer&&this.notFoundRenderer({},a)}}function ke(e,t){const n=t.split("/").filter(Boolean);if(e.length===0&&n.length===0)return{};if(e.length!==n.length)return null;const a={};for(let s=0;s<e.length;s++){const r=e[s],i=n[s];if(r.startsWith(":"))a[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return a}function Ie(e){const t={};return new URLSearchParams(e).forEach((a,s)=>{t[s]=a}),t}const K="/api";async function h(e,t){let n=`${K}${e}`;if(t){const s=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&s.set(i,o)});const r=s.toString();r&&(n+=`?${r}`)}const a=await fetch(n);if(!a.ok)throw new Error(`API error: ${a.status} ${a.statusText}`);return a.json()}async function J(e,t){const n=await fetch(`${K}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function Ee(e,t){const n=await fetch(`${K}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function V(e){const t=await fetch(`${K}${e}`,{method:"DELETE"});if(!t.ok)throw new Error(`API error: ${t.status} ${t.statusText}`);return t.json()}function Q(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.time_range&&(n.time_range=t.time_range),t.region&&(n.region=t.region),t.language&&(n.language=t.language),t.safe_search&&(n.safe_search=t.safe_search),t.site&&(n.site=t.site),t.exclude_site&&(n.exclude_site=t.exclude_site),t.lens&&(n.lens=t.lens)),n}const y={search(e,t){return h("/search",Q(e,t))},searchImages(e,t){const n={q:e};return t&&(t.page!==void 0&&(n.page=String(t.page)),t.per_page!==void 0&&(n.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(n.size=t.size),t.color&&t.color!=="any"&&(n.color=t.color),t.type&&t.type!=="any"&&(n.type=t.type),t.aspect&&t.aspect!=="any"&&(n.aspect=t.aspect),t.time&&t.time!=="any"&&(n.time=t.time),t.rights&&t.rights!=="any"&&(n.rights=t.rights),t.filetype&&t.filetype!=="any"&&(n.filetype=t.filetype),t.safe&&(n.safe=t.safe)),h("/search/images",n)},reverseImageSearch(e){return J("/search/images/reverse",{url:e})},searchVideos(e,t){return h("/search/videos",Q(e,t))},searchNews(e,t){return h("/search/news",Q(e,t))},suggest(e){return h("/suggest",{q:e})},trending(){return h("/suggest/trending")},calculate(e){return h("/instant/calculate",{q:e})},convert(e){return h("/instant/convert",{q:e})},currency(e){return h("/instant/currency",{q:e})},weather(e){return h("/instant/weather",{q:e})},define(e){return h("/instant/define",{q:e})},time(e){return h("/instant/time",{q:e})},knowledge(e){return h(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return h("/preferences")},setPreference(e,t){return J("/preferences",{domain:e,action:t})},deletePreference(e){return V(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return h("/lenses")},createLens(e){return J("/lenses",e)},deleteLens(e){return V(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return h("/history")},clearHistory(){return V("/history")},deleteHistoryItem(e){return V(`/history/${encodeURIComponent(e)}`)},getSettings(){return h("/settings")},updateSettings(e){return Ee("/settings",e)},getBangs(){return h("/bangs")},parseBang(e){return h("/bangs/parse",{q:e})},getRelated(e){return h("/related",{q:e})}};function Ce(e){let t={...e};const n=new Set;return{get(){return t},set(a){t={...t,...a},n.forEach(s=>s(t))},subscribe(a){return n.add(a),()=>{n.delete(a)}}}}const fe="mizu_search_state";function _e(){try{const e=localStorage.getItem(fe);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const H=Ce(_e());H.subscribe(e=>{try{localStorage.setItem(fe,JSON.stringify(e))}catch{}});function Y(e){const t=H.get(),n=[e,...t.recentSearches.filter(a=>a!==e)].slice(0,20);H.set({recentSearches:n})}const ye='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Le='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Se='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',Be='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Me='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',He='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function U(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",n=e.initialValue?Ae(e.initialValue):"",a=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${ye}</span>
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
          ${Le}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${Se}
        </button>
        <button class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${Be}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function z(e){const t=document.getElementById("search-input"),n=document.getElementById("search-clear-btn"),a=document.getElementById("autocomplete-dropdown"),s=document.getElementById("search-box-wrapper");if(!t||!n||!a||!s)return;let r=null,i=[],o=-1,d=!1;function p(u){if(i=u,o=-1,u.length===0){g();return}d=!0,a.innerHTML=u.map((l,m)=>`
        <div class="autocomplete-item ${m===o?"active":""}" data-index="${m}">
          <span class="suggestion-icon">${l.icon}</span>
          ${l.prefix?`<span class="bang-trigger">${ie(l.prefix)}</span>`:""}
          <span>${ie(l.text)}</span>
        </div>
      `).join(""),a.classList.remove("hidden"),a.classList.add("has-items"),a.querySelectorAll(".autocomplete-item").forEach(l=>{l.addEventListener("mousedown",m=>{m.preventDefault();const x=parseInt(l.dataset.index||"0");f(x)}),l.addEventListener("mouseenter",()=>{const m=parseInt(l.dataset.index||"0");L(m)})})}function g(){d=!1,a.classList.add("hidden"),a.classList.remove("has-items"),a.innerHTML="",i=[],o=-1}function L(u){o=u,a.querySelectorAll(".autocomplete-item").forEach((l,m)=>{l.classList.toggle("active",m===u)})}function f(u){const l=i[u];l&&(l.type==="bang"&&l.prefix?(t.value=l.prefix+" ",t.focus(),w(l.prefix+" ")):(t.value=l.text,g(),b(l.text)))}function b(u){const l=u.trim();l&&(g(),e(l))}async function w(u){const l=u.trim();if(!l){R();return}if(l.startsWith("!"))try{const x=(await y.getBangs()).filter(S=>S.trigger.startsWith(l)||S.name.toLowerCase().includes(l.slice(1).toLowerCase())).slice(0,8);if(x.length>0){p(x.map(S=>({text:S.name,type:"bang",icon:He,prefix:S.trigger})));return}}catch{}try{const m=await y.suggest(l);if(t.value.trim()!==l)return;const x=m.map(S=>({text:S.text,type:"suggestion",icon:ye}));x.length===0?R(l):p(x)}catch{R(l)}}function R(u){let m=H.get().recentSearches;if(u&&(m=m.filter(x=>x.toLowerCase().includes(u.toLowerCase()))),m.length===0){g();return}p(m.slice(0,8).map(x=>({text:x,type:"recent",icon:Me})))}t.addEventListener("input",()=>{const u=t.value;n.classList.toggle("hidden",u.length===0),r&&clearTimeout(r),r=setTimeout(()=>w(u),150)}),t.addEventListener("focus",()=>{t.value.trim()?w(t.value):R()}),t.addEventListener("keydown",u=>{if(!d){if(u.key==="Enter"){b(t.value);return}if(u.key==="ArrowDown"){w(t.value);return}return}switch(u.key){case"ArrowDown":u.preventDefault(),L(Math.min(o+1,i.length-1));break;case"ArrowUp":u.preventDefault(),L(Math.max(o-1,-1));break;case"Enter":u.preventDefault(),o>=0?f(o):b(t.value);break;case"Escape":g();break;case"Tab":g();break}}),t.addEventListener("blur",()=>{setTimeout(()=>g(),200)}),n.addEventListener("click",()=>{t.value="",n.classList.add("hidden"),t.focus(),R()})}function ie(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ae(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Te=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],Ne=[{label:"Calculator",icon:Oe(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:Pe(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:Fe(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:Ue(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:ze(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:De(),query:"define serendipity",color:"bg-red/10 text-red"}];function Re(){return`
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
          ${U({size:"lg",autofocus:!0})}
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
          ${Te.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${X(e.trigger)}</span>
              <span class="text-tertiary ml-1">${X(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${Ne.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${qe(e.query)}">
                ${e.icon}
                <span>${X(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${Ve()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${Ge()}
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
  `}function je(e){z(a=>{e.navigate(`/search?q=${encodeURIComponent(a)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var r;const a=document.getElementById("search-input"),s=(r=a==null?void 0:a.value)==null?void 0:r.trim();s&&e.navigate(`/search?q=${encodeURIComponent(s)}`)});const n=document.getElementById("home-lucky-btn");n==null||n.addEventListener("click",()=>{var r;const a=document.getElementById("search-input"),s=(r=a==null?void 0:a.value)==null?void 0:r.trim();s&&e.navigate(`/search?q=${encodeURIComponent(s)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(a=>{a.addEventListener("click",()=>{const s=a.dataset.bang||"",r=document.getElementById("search-input");r&&(r.value=s+" ",r.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(a=>{a.addEventListener("click",()=>{const s=a.dataset.query||"";s&&e.navigate(`/search?q=${encodeURIComponent(s)}`)})})}function X(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function qe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Oe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function Pe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function Fe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function Ue(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function ze(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function De(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function Ve(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function Ge(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}const We='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',Ke='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',Ye='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',Ze='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>';function Je(e,t){const n=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,a=et(e.url),s=e.published?tt(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${E(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="result-sitelinks">
        ${e.sitelinks.map(d=>`<a href="${E(d.url)}" target="_blank" rel="noopener">${C(d.title)}</a>`).join("")}
       </div>`:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${E(e.domain)}">
      <div class="result-url">
        <img class="favicon" src="${E(n)}" alt="" width="18" height="18" loading="lazy" onerror="this.style.display='none'" />
        <div>
          <span class="text-sm">${C(e.domain)}</span>
          <span class="breadcrumbs">${a}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${E(e.url)}" target="_blank" rel="noopener">${C(e.title)}</a>
          </div>
          ${s?`<span class="result-date">${C(s)} -- </span>`:""}
          <div class="result-snippet">${r}</div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${We}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function Qe(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const n=e.dataset.menuIndex,a=document.getElementById(`domain-menu-${n}`),s=e.closest(".search-result"),r=(s==null?void 0:s.dataset.domain)||"";if(!a)return;if(!a.classList.contains("hidden")){a.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),a.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${E(r)}">
          ${Ke}
          <span>Boost ${C(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${E(r)}">
          ${Ye}
          <span>Lower ${C(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${E(r)}">
          ${Ze}
          <span>Block ${C(r)}</span>
        </button>
      `,a.classList.remove("hidden"),a.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const d=o.dataset.action||"",p=o.dataset.domain||"";try{await y.setPreference(p,d),a.classList.add("hidden"),Xe(`${d.charAt(0).toUpperCase()+d.slice(1)}ed ${p}`)}catch(g){console.error("Failed to set preference:",g)}})});const i=o=>{!a.contains(o.target)&&o.target!==e&&(a.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function Xe(e){const t=document.getElementById("toast");t&&t.remove();const n=document.createElement("div");n.id="toast",n.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",n.textContent=e,document.body.appendChild(n),setTimeout(()=>{n.style.opacity="0",setTimeout(()=>n.remove(),300)},2e3)}function et(e){try{const n=new URL(e).pathname.split("/").filter(Boolean);return n.length===0?"":" > "+n.map(a=>C(decodeURIComponent(a))).join(" > ")}catch{return""}}function tt(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60*24));return s===0?"Today":s===1?"1 day ago":s<7?`${s} days ago`:s<30?`${Math.floor(s/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function C(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function E(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const nt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',at='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',st='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',rt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',it='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',ot='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',lt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',ct='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function dt(e){switch(e.type){case"calculator":return ut(e);case"unit_conversion":return ht(e);case"currency":return pt(e);case"weather":return gt(e);case"definition":return mt(e);case"time":return vt(e);default:return ft(e)}}function ut(e){const t=e.data||{},n=t.expression||e.query||"",a=t.formatted||t.result||e.result||"";return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${nt}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${c(n)} = ${c(String(a))}</div>
    </div>
  `}function ht(e){const t=e.data||{},n=t.from_value??"",a=t.from_unit??"",s=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${at}
        <span class="instant-type">Unit Conversion${i?` -- ${c(i)}`:""}</span>
      </div>
      <div class="instant-result">${c(String(n))} ${c(a)} = ${c(String(s))} ${c(r)}</div>
      ${t.formatted?`<div class="instant-sub">${c(t.formatted)}</div>`:""}
    </div>
  `}function pt(e){const t=e.data||{},n=t.from_value??"",a=t.from_currency??"",s=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${st}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${c(String(n))} ${c(a)} = ${c(String(s))} ${c(r)}</div>
      ${i?`<div class="instant-sub">1 ${c(a)} = ${c(String(i))} ${c(r)}</div>`:""}
    </div>
  `}function gt(e){const t=e.data||{},n=t.location||"",a=t.temperature??"",s=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=rt;return s.includes("cloud")||s.includes("overcast")?o=it:(s.includes("rain")||s.includes("drizzle")||s.includes("storm"))&&(o=ot),`
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
  `}function mt(e){const t=e.data||{},n=t.word||e.query||"",a=t.phonetic||"",s=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[];return`
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${lt}
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
  `}function vt(e){const t=e.data||{},n=t.location||"",a=t.time||"",s=t.date||"",r=t.timezone||"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ct}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${c(n)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${c(a)}</div>
      <div class="text-sm text-tertiary">${c(s)}</div>
      ${r?`<div class="text-xs text-light mt-1">${c(r)}</div>`:""}
    </div>
  `}function ft(e){return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${c(e.type)}</div>
      <div class="instant-result">${c(e.result)}</div>
    </div>
  `}function c(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const yt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function xt(e){const t=e.image?`<img class="kp-image" src="${ee(e.image)}" alt="${ee(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",n=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${B(r.label)}</td>
                <td class="fact-value">${B(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",a=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${ee(r.url)}" target="_blank" rel="noopener">
              ${yt}
              <span>${B(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",s=e.source?`<div class="kp-source">Source: ${B(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${B(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${B(e.subtitle)}</div>`:""}
      <div class="kp-description">${B(e.description)}</div>
      ${n}
      ${a}
      ${s}
    </div>
  `}function B(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ee(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const bt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',wt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function $t(e){const{currentPage:t,hasMore:n,totalResults:a,perPage:s}=e,r=Math.min(Math.ceil(a/s),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const d=[];for(let f=i;f<=o;f++)d.push(f);const p=kt(t),g=t<=1?"disabled":"",L=!n&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${p}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${g}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${bt}
          </button>
          ${d.map(f=>`
            <button class="pagination-btn ${f===t?"active":""}" data-page="${f}">
              ${f}
            </button>
          `).join("")}
          <button class="pagination-btn ${L}" data-page="${t+1}" ${!n&&t>=r?"disabled":""} aria-label="Next page">
            ${wt}
          </button>
        </div>
      </div>
    </div>
  `}function kt(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],n=["M","i","z","u"],a=Math.min(e-1,6);let s=[n[0]];for(let r=0;r<1+a;r++)s.push("i");s.push("z");for(let r=0;r<1+a;r++)s.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${s.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function It(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(n=>{n.addEventListener("click",()=>{const a=parseInt(n.dataset.page||"1");isNaN(a)||n.disabled||(e(a),window.scrollTo({top:0,behavior:"smooth"}))})})}const Et='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Ct='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',_t='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',Lt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>';function Z(e){const{query:t,active:n}=e,a=encodeURIComponent(t);return`
    <div class="search-tabs" id="search-tabs">
      ${[{id:"all",label:"All",icon:Et,href:`/search?q=${a}`},{id:"images",label:"Images",icon:Ct,href:`/images?q=${a}`},{id:"videos",label:"Videos",icon:_t,href:`/videos?q=${a}`},{id:"news",label:"News",icon:Lt,href:`/news?q=${a}`}].map(r=>`
        <a class="search-tab ${r.id===n?"active":""}" href="${r.href}" data-link data-tab="${r.id}">
          ${r.icon}
          <span>${r.label}</span>
        </a>
      `).join("")}
    </div>
  `}const St='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Bt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',oe=[{value:"",label:"Any time"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}];function Mt(e,t){var s;const n=((s=oe.find(r=>r.value===t))==null?void 0:s.label)||"Any time",a=t!=="";return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${U({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${St}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          <div class="flex items-center gap-2">
            ${Z({query:e,active:"all"})}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${a?"active-filter":""}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${M(n)}</span>
                ${Bt}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${oe.map(r=>`
                  <button class="time-filter-option ${r.value===t?"active":""}" data-time-range="${r.value}">
                    ${M(r.label)}
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
  `}function Ht(e,t,n){const a=parseInt(n.page||"1"),s=n.time_range||"",r=H.get().settings;z(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),At(e,t),t&&Y(t),Tt(e,t,a,s,r.results_per_page)}function At(e,t,n){const a=document.getElementById("time-filter-btn"),s=document.getElementById("time-filter-dropdown");!a||!s||(a.addEventListener("click",r=>{r.stopPropagation(),s.classList.toggle("hidden")}),s.querySelectorAll(".time-filter-option").forEach(r=>{r.addEventListener("click",()=>{const i=r.dataset.timeRange||"";s.classList.add("hidden");let o=`/search?q=${encodeURIComponent(t)}`;i&&(o+=`&time_range=${i}`),e.navigate(o)})}),document.addEventListener("click",r=>{!s.contains(r.target)&&r.target!==a&&s.classList.add("hidden")}))}async function Tt(e,t,n,a,s){const r=document.getElementById("search-content");if(!(!r||!t))try{const i=await y.search(t,{page:n,per_page:s,time_range:a||void 0});if(i.redirect){window.location.href=i.redirect;return}Nt(r,e,i,t,n,a)}catch(i){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${M(String(i))}</p>
      </div>
    `}}function Nt(e,t,n,a,s,r){const i=n.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(n.corrected_query)}" data-link class="text-link font-medium">${M(n.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(a)}&exact=1" data-link class="text-link">${M(a)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${Rt(n.total_results)} results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,d=n.instant_answer?dt(n.instant_answer):"",p=n.results.length>0?n.results.map((b,w)=>Je(b,w)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${M(a)}</strong>"</div>`,g=n.related_searches&&n.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${n.related_searches.map(b=>`
            <a href="/search?q=${encodeURIComponent(b)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${M(b)}
            </a>
          `).join("")}
        </div>
      </div>
    `:"",L=$t({currentPage:s,hasMore:n.has_more,totalResults:n.total_results,perPage:n.per_page}),f=n.knowledge_panel?xt(n.knowledge_panel):"";e.innerHTML=`
    <div class="flex gap-8">
      <div class="flex-1 min-w-0">
        ${i}
        ${o}
        ${d}
        ${p}
        ${g}
        ${L}
      </div>
      ${f?`<aside class="hidden lg:block flex-shrink-0 w-[360px] pt-2">${f}</aside>`:""}
    </div>
  `,Qe(),It(b=>{let w=`/search?q=${encodeURIComponent(a)}&page=${b}`;r&&(w+=`&time_range=${r}`),t.navigate(w)})}function Rt(e){return e.toLocaleString("en-US")}function M(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const jt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',qt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',le='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',ce='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',Ot='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',xe='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>';let O="",$={},P=1,N=!1,_=!0,k=[],j=!1;function Pt(e){return`
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20">
        <div class="flex items-center gap-4 px-4 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px] flex items-center gap-2">
            ${U({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${qt}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${jt}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1 border-b border-border">
          ${Z({query:e,active:"images"})}
          <button id="tools-btn" class="tools-btn ml-4">
            ${Ot}
            <span>Tools</span>
            ${xe}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${Ft()}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 px-4 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-panel-content">
            <button id="preview-close" class="preview-close" aria-label="Close">${le}</button>
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
            <button id="reverse-modal-close" class="modal-close">${le}</button>
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
  `}function Ft(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${xe}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(n=>`
              <button class="filter-option${n==="any"?" active":""}" data-value="${n}">
                ${W(t.id,n)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function W(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function Ut(e,t){O=t,$={},P=1,k=[],_=!0,j=!1,z(n=>{e.navigate(`/images?q=${encodeURIComponent(n)}`)}),t&&Y(t),zt(),Dt(),Vt(),Gt(),Wt(),te(t,$)}function zt(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{j=!j,t.classList.toggle("hidden",!j),e.classList.toggle("active",j)})}function Dt(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(a=>{a.addEventListener("click",s=>{s.stopPropagation();const r=a.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i==null||i.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(a=>{a.addEventListener("click",()=>{const s=a.closest(".filter-dropdown"),r=s==null?void 0:s.dataset.dropdown,i=a.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);!r||!i||!o||(s.querySelectorAll(".filter-option").forEach(d=>d.classList.remove("active")),a.classList.add("active"),i==="any"?(delete $[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=W(r,"any").replace("Any ","")):($[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=W(r,i)),s.classList.add("hidden"),de(),P=1,k=[],_=!0,te(O,$))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(a=>a.classList.add("hidden"))});const n=document.getElementById("clear-filters");n&&n.addEventListener("click",()=>{$={},P=1,k=[],_=!0,t.querySelectorAll(".filter-chip").forEach(a=>{const s=a.dataset.filter;a.classList.remove("has-value"),a.querySelector(".filter-chip-label").textContent=W(s,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(a=>{a.querySelectorAll(".filter-option").forEach((s,r)=>{s.classList.toggle("active",r===0)})}),de(),te(O,$)})}function de(){const e=document.getElementById("clear-filters");if(!e)return;const t=Object.keys($).length>0;e.classList.toggle("hidden",!t)}function Vt(e){const t=document.getElementById("reverse-search-btn"),n=document.getElementById("reverse-modal"),a=document.getElementById("reverse-modal-close"),s=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!n||(t.addEventListener("click",()=>{n.classList.remove("hidden")}),a==null||a.addEventListener("click",()=>{n.classList.add("hidden")}),n.addEventListener("click",d=>{d.target===n&&n.classList.add("hidden")}),s&&(s.addEventListener("dragover",d=>{d.preventDefault(),s.classList.add("drag-over")}),s.addEventListener("dragleave",()=>{s.classList.remove("drag-over")}),s.addEventListener("drop",d=>{var g;d.preventDefault(),s.classList.remove("drag-over");const p=(g=d.dataTransfer)==null?void 0:g.files;p&&p[0]&&(ue(p[0]),n.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(ue(r.files[0]),n.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const d=i.value.trim();d&&(he(d),n.classList.add("hidden"))}),i.addEventListener("keydown",d=>{if(d.key==="Enter"){const p=i.value.trim();p&&(he(p),n.classList.add("hidden"))}})))}async function ue(e,t){alert("Image upload coming soon. Please use the URL option for now.")}async function he(e,t){const n=document.getElementById("images-content");if(n){n.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const a=await y.reverseImageSearch(e);n.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${F(e)}" alt="Query image" class="query-image" />
        </div>

        ${a.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${a.similar_images.length})</h3>
            <div class="image-grid">
              ${a.similar_images.map((s,r)=>se(s,r)).join("")}
            </div>
          </div>
        `:`
          <div class="py-8 text-secondary">No similar images found.</div>
        `}
      </div>
    `,n.querySelectorAll(".image-card").forEach(s=>{s.addEventListener("click",()=>{const r=parseInt(s.dataset.imageIndex||"0",10);ae(a.similar_images[r])})})}catch(a){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${A(String(a))}</p>
      </div>
    `}}}function Gt(){const e=document.getElementById("preview-close");e&&e.addEventListener("click",pe),document.addEventListener("keydown",t=>{t.key==="Escape"&&pe()})}function ae(e){const t=document.getElementById("preview-panel"),n=document.getElementById("preview-image"),a=document.getElementById("preview-details");if(!t||!n||!a)return;n.src=e.url,n.alt=e.title;const r=e.width&&e.height&&e.width>0&&e.height>0?`${e.width} × ${e.height}${e.format?` · ${e.format.toUpperCase()}`:""}`:e.format?e.format.toUpperCase():"";a.innerHTML=`
    <h3 class="preview-title">${A(e.title||"Untitled")}</h3>
    ${r?`<p class="preview-dimensions">${r}</p>`:""}
    <p class="preview-source">${A(e.source_domain)}</p>
    <div class="preview-actions">
      <a href="${F(e.url)}" target="_blank" class="preview-btn">View image ${ce}</a>
      <a href="${F(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">Visit page ${ce}</a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function pe(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function Wt(){const e=document.createElement("div");e.id="scroll-sentinel",e.style.height="1px";const t=new IntersectionObserver(n=>{n[0].isIntersecting&&!N&&_&&O&&Kt()},{rootMargin:"200px"});setTimeout(()=>{const n=document.getElementById("images-content");if(n){const a=document.getElementById("scroll-sentinel");a&&a.remove(),n.appendChild(e),t.observe(e)}},100)}async function Kt(){if(!(N||!_)){N=!0,P++;try{const e=await y.searchImages(O,{...$,page:P}),t=e.results;_=e.has_more,k=[...k,...t];const n=document.querySelector(".image-grid");if(n&&t.length>0){const a=k.length-t.length,s=t.map((i,o)=>se(i,a+o)).join("");n.insertAdjacentHTML("beforeend",s),n.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);ae(k[o])})})}if(!_){const a=document.getElementById("scroll-sentinel");a&&(a.innerHTML='<div class="text-center text-tertiary py-4 text-sm">No more images</div>')}}catch{}finally{N=!1}}}async function te(e,t){const n=document.getElementById("images-content");if(!(!n||!e)){N=!0;try{const a=await y.searchImages(e,{...t,page:1,per_page:40}),s=a.results;if(_=a.has_more,k=s,s.length===0){n.innerHTML=`
        <div class="py-8 text-secondary">No image results found for "<strong>${A(e)}</strong>"</div>
      `;return}n.innerHTML=`
      <div class="image-grid">
        ${s.map((r,i)=>se(r,i)).join("")}
      </div>
    `,n.querySelectorAll(".image-card").forEach(r=>{r.setAttribute("data-initialized","true"),r.addEventListener("click",()=>{const i=parseInt(r.dataset.imageIndex||"0",10);ae(k[i])})})}catch(a){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${A(String(a))}</p>
      </div>
    `}finally{N=!1}}}function se(e,t){const n=e.width&&e.height&&e.width>0&&e.height>0;return`
    <div class="image-card" data-image-index="${t}">
      <img
        src="${F(e.thumbnail_url||e.url)}"
        alt="${F(e.title)}"
        loading="lazy"
        onerror="this.parentElement.style.display='none'"
      />
      <div class="image-info">
        <div class="image-title">${A(e.title||"")}</div>
        <div class="image-source">${A(e.source_domain)}</div>
        ${n?`<div class="image-dimensions">${e.width} × ${e.height}</div>`:""}
      </div>
    </div>
  `}function A(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function F(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Yt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Zt(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${U({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Yt}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${Z({query:e,active:"videos"})}
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
  `}function Jt(e,t){z(n=>{e.navigate(`/videos?q=${encodeURIComponent(n)}`)}),t&&Y(t),Qt(t)}async function Qt(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const n=await y.searchVideos(e),a=n.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${T(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} video results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        ${a.map(s=>Xt(s)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${T(String(n))}</p>
      </div>
    `}}function Xt(e){var r;const t=((r=e.thumbnail)==null?void 0:r.url)||"",n=e.views?en(e.views):"",a=e.published?tn(e.published):"",s=[e.channel,n,a].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${G(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${G(t)}" alt="${G(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${T(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${G(e.url)}" target="_blank" rel="noopener">${T(e.title)}</a>
        </div>
        <div class="video-meta">${T(s)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${T(e.platform)}</div>`:""}
      </div>
    </div>
  `}function en(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function tn(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60*24));return s===0?"Today":s===1?"1 day ago":s<7?`${s} days ago`:s<30?`${Math.floor(s/7)} weeks ago`:s<365?`${Math.floor(s/30)} months ago`:`${Math.floor(s/365)} years ago`}catch{return e}}function T(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function G(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const nn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function an(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 py-3 max-w-[1200px]">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${U({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${nn}
          </a>
        </div>
        <div class="max-w-[1200px] pl-[170px]">
          ${Z({query:e,active:"news"})}
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
  `}function sn(e,t){z(n=>{e.navigate(`/news?q=${encodeURIComponent(n)}`)}),t&&Y(t),rn(t)}async function rn(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const n=await y.searchNews(e),a=n.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No news results found for "<strong>${q(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${n.total_results.toLocaleString()} news results (${(n.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div>
        ${a.map(s=>on(s)).join("")}
      </div>
    `}catch(n){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load news results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${q(String(n))}</p>
      </div>
    `}}function on(e){var a;const t=((a=e.thumbnail)==null?void 0:a.url)||"",n=e.published_date?ln(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${q(e.source||e.domain)}
          ${n?` · ${q(n)}`:""}
        </div>
        <div class="news-title">
          <a href="${ge(e.url)}" target="_blank" rel="noopener">${q(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${ge(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function ln(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),s=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return s<1?"Just now":s<24?`${s}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function q(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ge(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const cn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',dn=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],un=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function hn(){const e=H.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${cn}
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
              ${dn.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${me(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${un.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${me(t.label)}</option>`).join("")}
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
  `}function pn(e){const t=document.getElementById("settings-form"),n=document.getElementById("settings-status");t&&t.addEventListener("submit",async a=>{a.preventDefault();const s=new FormData(t),r={safe_search:s.get("safe_search")||"moderate",results_per_page:parseInt(s.get("results_per_page"))||10,region:s.get("region")||"auto",language:s.get("language")||"en",theme:s.get("theme")||"light",open_in_new_tab:s.has("open_in_new_tab"),show_thumbnails:s.has("show_thumbnails")};H.set({settings:r});try{await y.updateSettings(r)}catch{}n&&(n.classList.remove("hidden"),setTimeout(()=>{n.classList.add("hidden")},2e3))})}function me(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const gn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',mn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',vn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',fn='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function yn(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${gn}
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
  `}function xn(e){const t=document.getElementById("clear-all-btn");bn(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await y.clearHistory(),re(),t.classList.add("hidden")}catch(n){console.error("Failed to clear history:",n)}})}async function bn(e){const t=document.getElementById("history-content"),n=document.getElementById("clear-all-btn");if(t)try{const a=await y.getHistory();if(a.length===0){re();return}n&&n.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${a.map(s=>wn(s)).join("")}
      </div>
    `,$n(e)}catch(a){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${ne(String(a))}</p>
      </div>
    `}}function wn(e){const t=kn(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${ve(e.id)}">
      <span class="text-light flex-shrink-0">${vn}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${ne(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${ne(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${ve(e.id)}" aria-label="Delete">
        ${mn}
      </button>
    </div>
  `}function $n(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async n=>{n.preventDefault(),n.stopPropagation();const a=t.dataset.deleteId||"",s=t.closest(".history-item");try{await y.deleteHistoryItem(a),s&&s.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){re();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function re(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${fn}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function kn(e){try{const t=new Date(e),n=new Date,a=n.getTime()-t.getTime(),s=Math.floor(a/(1e3*60)),r=Math.floor(a/(1e3*60*60)),i=Math.floor(a/(1e3*60*60*24));return s<1?"Just now":s<60?`${s}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==n.getFullYear()?"numeric":void 0})}catch{return e}}function ne(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ve(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const I=document.getElementById("app");if(!I)throw new Error("App container not found");const v=new $e;v.addRoute("",(e,t)=>{I.innerHTML=Re(),je(v)});v.addRoute("search",(e,t)=>{const n=t.q||"",a=t.time_range||"";I.innerHTML=Mt(n,a),Ht(v,n,t)});v.addRoute("images",(e,t)=>{const n=t.q||"";I.innerHTML=Pt(n),Ut(v,n)});v.addRoute("videos",(e,t)=>{const n=t.q||"";I.innerHTML=Zt(n),Jt(v,n)});v.addRoute("news",(e,t)=>{const n=t.q||"";I.innerHTML=an(n),sn(v,n)});v.addRoute("settings",(e,t)=>{I.innerHTML=hn(),pn()});v.addRoute("history",(e,t)=>{I.innerHTML=yn(),xn(v)});v.setNotFound((e,t)=>{I.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;v.navigate(t.detail.path)});v.start();
