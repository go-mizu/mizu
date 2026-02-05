var Qe=Object.defineProperty;var Xe=(e,t,s)=>t in e?Qe(e,t,{enumerable:!0,configurable:!0,writable:!0,value:s}):e[t]=s;var ie=(e,t,s)=>Xe(e,typeof t!="symbol"?t+"":t,s);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const n of document.querySelectorAll('link[rel="modulepreload"]'))a(n);new MutationObserver(n=>{for(const r of n)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&a(i)}).observe(document,{childList:!0,subtree:!0});function s(n){const r={};return n.integrity&&(r.integrity=n.integrity),n.referrerPolicy&&(r.referrerPolicy=n.referrerPolicy),n.crossOrigin==="use-credentials"?r.credentials="include":n.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function a(n){if(n.ep)return;n.ep=!0;const r=s(n);fetch(n.href,r)}})();class et{constructor(){ie(this,"routes",[]);ie(this,"currentPath","");ie(this,"notFoundRenderer",null)}addRoute(t,s){const a=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:a,renderer:s})}setNotFound(t){this.notFoundRenderer=t}navigate(t,s=!1){t!==this.currentPath&&(s?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const s=t.target.closest("a[data-link]");if(s){t.preventDefault();const a=s.getAttribute("href");a&&this.navigate(a)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),s=t.pathname,a=st(t.search);this.currentPath=s+t.search;for(const n of this.routes){const r=tt(n.segments,s);if(r!==null){n.renderer(r,a);return}}this.notFoundRenderer&&this.notFoundRenderer({},a)}}function tt(e,t){const s=t.split("/").filter(Boolean);if(e.length===0&&s.length===0)return{};if(e.length!==s.length)return null;const a={};for(let n=0;n<e.length;n++){const r=e[n],i=s[n];if(r.startsWith(":"))a[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return a}function st(e){const t={};return new URLSearchParams(e).forEach((a,n)=>{t[n]=a}),t}const pe="/api";async function v(e,t){let s=`${pe}${e}`;if(t){const n=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&n.set(i,o)});const r=n.toString();r&&(s+=`?${r}`)}const a=await fetch(s);if(!a.ok)throw new Error(`API error: ${a.status} ${a.statusText}`);return a.json()}async function O(e,t){const s=await fetch(`${pe}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function Ie(e,t){const s=await fetch(`${pe}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function J(e,t){const s=await fetch(`${pe}${e}`,{method:"DELETE",headers:t?{"Content-Type":"application/json"}:void 0,body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}function ve(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.time_range&&(s.time_range=t.time_range),t.region&&(s.region=t.region),t.language&&(s.language=t.language),t.safe_search&&(s.safe_search=t.safe_search),t.site&&(s.site=t.site),t.exclude_site&&(s.exclude_site=t.exclude_site),t.lens&&(s.lens=t.lens),t.verbatim&&(s.verbatim="1")),s}const w={search(e,t){return v("/search",ve(e,t))},searchImages(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(s.size=t.size),t.color&&t.color!=="any"&&(s.color=t.color),t.type&&t.type!=="any"&&(s.type=t.type),t.aspect&&t.aspect!=="any"&&(s.aspect=t.aspect),t.time&&t.time!=="any"&&(s.time=t.time),t.rights&&t.rights!=="any"&&(s.rights=t.rights),t.filetype&&t.filetype!=="any"&&(s.filetype=t.filetype),t.safe&&(s.safe=t.safe)),v("/search/images",s)},reverseImageSearch(e){return O("/search/images/reverse",{url:e})},reverseImageSearchByUpload(e){return O("/search/images/reverse",{image_data:e})},searchVideos(e,t){return v("/search/videos",ve(e,t))},searchNews(e,t){return v("/search/news",ve(e,t))},searchMusic(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),v(`/search/music?${s}`)},searchScience(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),t!=null&&t.per_page&&s.set("per_page",String(t.per_page)),v(`/search/science?${s}`)},searchMaps(e){const t=new URLSearchParams({q:e});return v(`/search/maps?${t}`)},searchCode(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),t!=null&&t.per_page&&s.set("per_page",String(t.per_page)),v(`/search/code?${s}`)},searchSocial(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),v(`/search/social?${s}`)},suggest(e){return v("/suggest",{q:e})},trending(){return v("/suggest/trending")},calculate(e){return v("/instant/calculate",{q:e})},convert(e){return v("/instant/convert",{q:e})},currency(e){return v("/instant/currency",{q:e})},weather(e){return v("/instant/weather",{q:e})},define(e){return v("/instant/define",{q:e})},time(e){return v("/instant/time",{q:e})},knowledge(e){return v(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return v("/preferences")},setPreference(e,t){return O("/preferences",{domain:e,action:t})},deletePreference(e){return J(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return v("/lenses")},createLens(e){return O("/lenses",e)},deleteLens(e){return J(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return v("/history")},clearHistory(){return J("/history")},deleteHistoryItem(e){return J(`/history/${encodeURIComponent(e)}`)},getSettings(){return v("/settings")},updateSettings(e){return Ie("/settings",e)},getBangs(){return v("/bangs")},parseBang(e){return v("/bangs/parse",{q:e})},getRelated(e){return v("/related",{q:e})},newsHome(){return v("/news/home")},newsCategory(e,t=1){return v(`/news/category/${e}`,{page:String(t)})},newsSearch(e,t){const s={q:e};return t!=null&&t.page&&(s.page=String(t.page)),t!=null&&t.time&&(s.time=t.time),t!=null&&t.source&&(s.source=t.source),v("/news/search",s)},newsStory(e){return v(`/news/story/${e}`)},newsLocal(e){const t={};return e&&(t.city=e.city,e.state&&(t.state=e.state),t.country=e.country),v("/news/local",t)},newsFollowing(){return v("/news/following")},newsPreferences(){return v("/news/preferences")},updateNewsPreferences(e){return Ie("/news/preferences",e)},followNews(e,t){return O("/news/follow",{type:e,id:t})},unfollowNews(e,t){return J("/news/follow",{type:e,id:t})},hideNewsSource(e){return O("/news/hide",{source:e})},setNewsLocation(e){return O("/news/location",e)},recordNewsRead(e,t){return O("/news/read",{article:e,duration:t})}};async function at(){try{const e=await fetch("/api/suggest/trending");if(!e.ok)return[];const t=await e.json();return Array.isArray(t)?t.map(s=>typeof s=="string"?s:s.text):t.suggestions||[]}catch{return[]}}function nt(e){let t={...e};const s=new Set;return{get(){return t},set(a){t={...t,...a},s.forEach(n=>n(t))},subscribe(a){return s.add(a),()=>{s.delete(a)}}}}const Ve="mizu_search_state";function rt(){try{const e=localStorage.getItem(Ve);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const G=nt(rt());G.subscribe(e=>{try{localStorage.setItem(Ve,JSON.stringify(e))}catch{}});function H(e){const t=G.get(),s=[e,...t.recentSearches.filter(a=>a!==e)].slice(0,20);G.set({recentSearches:s})}const De='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',it='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',ot='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',lt='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',ct='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',dt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function E(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",s=e.initialValue?pt(e.initialValue):"",a=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${De}</span>
        <input
          id="search-input"
          type="text"
          value="${s}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${a}" type="button" aria-label="Clear">
          ${it}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button id="voice-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${ot}
        </button>
        <button id="camera-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${lt}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function _(e){const t=document.getElementById("search-input"),s=document.getElementById("search-clear-btn"),a=document.getElementById("autocomplete-dropdown"),n=document.getElementById("search-box-wrapper");if(!t||!s||!a||!n)return;let r=null,i=[],o=-1,l=!1;function c(g){if(i=g,o=-1,g.length===0){d();return}l=!0,a.innerHTML=g.map((h,x)=>`
        <div class="autocomplete-item ${x===o?"active":""}" data-index="${x}">
          <span class="suggestion-icon">${h.icon}</span>
          ${h.prefix?`<span class="bang-trigger">${Ee(h.prefix)}</span>`:""}
          <span>${Ee(h.text)}</span>
        </div>
      `).join(""),a.classList.remove("hidden"),a.classList.add("has-items"),a.querySelectorAll(".autocomplete-item").forEach(h=>{h.addEventListener("mousedown",x=>{x.preventDefault();const I=parseInt(h.dataset.index||"0");p(I)}),h.addEventListener("mouseenter",()=>{const x=parseInt(h.dataset.index||"0");u(x)})})}function d(){l=!1,a.classList.add("hidden"),a.classList.remove("has-items"),a.innerHTML="",i=[],o=-1}function u(g){o=g,a.querySelectorAll(".autocomplete-item").forEach((h,x)=>{h.classList.toggle("active",x===g)})}function p(g){const h=i[g];h&&(h.type==="bang"&&h.prefix?(t.value=h.prefix+" ",t.focus(),$(h.prefix+" ")):(t.value=h.text,d(),f(h.text)))}function f(g){const h=g.trim();h&&(d(),e(h))}async function $(g){const h=g.trim();if(!h){b();return}if(h.startsWith("!"))try{const I=(await w.getBangs()).filter(U=>U.trigger.startsWith(h)||U.name.toLowerCase().includes(h.slice(1).toLowerCase())).slice(0,8);if(I.length>0){c(I.map(U=>({text:U.name,type:"bang",icon:dt,prefix:U.trigger})));return}}catch{}try{const x=await w.suggest(h);if(t.value.trim()!==h)return;const I=x.map(U=>({text:U.text,type:"suggestion",icon:De}));I.length===0?b(h):c(I)}catch{b(h)}}function b(g){let x=G.get().recentSearches;if(g&&(x=x.filter(I=>I.toLowerCase().includes(g.toLowerCase()))),x.length===0){d();return}c(x.slice(0,8).map(I=>({text:I,type:"recent",icon:ct})))}t.addEventListener("input",()=>{const g=t.value;s.classList.toggle("hidden",g.length===0),r&&clearTimeout(r),r=setTimeout(()=>$(g),150)}),t.addEventListener("focus",()=>{t.value.trim()?$(t.value):b()}),t.addEventListener("keydown",g=>{if(!l){if(g.key==="Enter"){f(t.value);return}if(g.key==="ArrowDown"){$(t.value);return}return}switch(g.key){case"ArrowDown":g.preventDefault(),u(Math.min(o+1,i.length-1));break;case"ArrowUp":g.preventDefault(),u(Math.max(o-1,-1));break;case"Enter":g.preventDefault(),o>=0?p(o):f(t.value);break;case"Escape":d();break;case"Tab":d();break}}),t.addEventListener("blur",()=>{setTimeout(()=>d(),200)}),s.addEventListener("click",()=>{t.value="",s.classList.add("hidden"),t.focus(),b()});const C=document.getElementById("voice-search-btn");C&&ut(C,t,g=>{t.value=g,s.classList.remove("hidden"),f(g)});const M=document.getElementById("camera-search-btn");M&&M.addEventListener("click",()=>{const g=document.getElementById("reverse-modal");g?g.classList.remove("hidden"):window.dispatchEvent(new CustomEvent("router:navigate",{detail:{path:"/images?reverse=1"}}))})}function ut(e,t,s){const a=window.SpeechRecognition||window.webkitSpeechRecognition;if(!a){e.style.display="none";return}let n=!1,r=null;e.addEventListener("click",()=>{n?o():i()});function i(){r=new a,r.continuous=!1,r.interimResults=!0,r.lang="en-US",r.onstart=()=>{n=!0,e.classList.add("listening"),e.style.color="#ea4335"},r.onresult=l=>{const c=Array.from(l.results).map(d=>d[0].transcript).join("");t.value=c,l.results[0].isFinal&&(o(),s(c))},r.onerror=l=>{console.error("Speech recognition error:",l.error),o(),l.error==="not-allowed"&&alert("Microphone access denied. Please allow microphone access to use voice search.")},r.onend=()=>{o()};try{r.start()}catch(l){console.error("Failed to start speech recognition:",l),o()}}function o(){if(n=!1,e.classList.remove("listening"),e.style.color="",r){try{r.stop()}catch{}r=null}}}function Ee(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function pt(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ht=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],gt=[{label:"Calculator",icon:yt(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:xt(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:wt(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:bt(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:$t(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:kt(),query:"define serendipity",color:"bg-red/10 text-red"}];function vt(){return`
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
          ${E({size:"lg",autofocus:!0})}
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

        <!-- Trending Searches -->
        <div class="trending-section mb-8 hidden" id="trending-container">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Trending Searches</p>
          <div class="trending-chips flex flex-wrap justify-center gap-2" id="trending-chips">
            <!-- Populated by JS -->
          </div>
        </div>

        <!-- Bang Shortcuts -->
        <div class="flex flex-wrap justify-center gap-2 mb-8">
          ${ht.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${de(e.trigger)}</span>
              <span class="text-tertiary ml-1">${de(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${gt.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${ft(e.query)}">
                ${e.icon}
                <span>${de(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${Ct()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${St()}
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
  `}function mt(e){_(n=>{e.navigate(`/search?q=${encodeURIComponent(n)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var i;const n=document.getElementById("search-input"),r=(i=n==null?void 0:n.value)==null?void 0:i.trim();r&&e.navigate(`/search?q=${encodeURIComponent(r)}`)});const s=document.getElementById("home-lucky-btn");s==null||s.addEventListener("click",()=>{var i;const n=document.getElementById("search-input"),r=(i=n==null?void 0:n.value)==null?void 0:i.trim();r&&e.navigate(`/search?q=${encodeURIComponent(r)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(n=>{n.addEventListener("click",()=>{const r=n.dataset.bang||"",i=document.getElementById("search-input");i&&(i.value=r+" ",i.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(n=>{n.addEventListener("click",()=>{const r=n.dataset.query||"";r&&e.navigate(`/search?q=${encodeURIComponent(r)}`)})});async function a(){const n=await at(),r=document.getElementById("trending-container"),i=document.getElementById("trending-chips");r&&i&&n.length>0&&(i.innerHTML=n.slice(0,10).map(o=>`
        <a href="/search?q=${encodeURIComponent(o)}" data-link
           class="trending-chip px-3 py-1.5 bg-secondary hover:bg-accent rounded-full text-xs font-medium text-secondary-foreground hover:text-accent-foreground transition-colors cursor-pointer no-underline">
          ${Lt()}
          ${de(o)}
        </a>
      `).join(""),r.classList.remove("hidden"))}a()}function de(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ft(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function yt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function xt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function wt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function bt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function $t(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function kt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function Ct(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function St(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}function Lt(){return'<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="inline-block mr-1"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>'}const It='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',Et='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',_t='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',Mt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>',Bt='<svg class="favicon-fallback" style="display:none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>';function At(e,t){const s=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,a=Rt(e.url),n=e.published?Ot(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${q(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="sitelinks-grid">
        ${e.sitelinks.slice(0,4).map(u=>`
          <div class="sitelink">
            <a href="${q(u.url)}" target="_blank" rel="noopener">${B(u.title)}</a>
          </div>
        `).join("")}
       </div>`:"",l=e.metadata||{},c=typeof l.rating=="number"?l.rating:null,d=c!==null?`
    <div class="rich-snippet">
      <span class="rating-stars">${"★".repeat(Math.round(c))}${"☆".repeat(5-Math.round(c))}</span>
      <span class="rating-value">${c.toFixed(1)}</span>
      ${l.reviewCount?`<span class="rating-count">(${Tt(l.reviewCount)} reviews)</span>`:""}
      ${l.price?`<span class="price">${B(String(l.price))}</span>`:""}
    </div>
  `:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${q(e.domain)}">
      <div class="result-url">
        <div class="favicon">
          <img src="${q(s)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';" />
          ${Bt}
        </div>
        <div>
          <span class="text-sm">${B(e.domain)}</span>
          <span class="breadcrumbs">${a}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${q(e.url)}" target="_blank" rel="noopener">${B(e.title)}</a>
          </div>
          ${d}
          <div class="result-snippet">
            ${n?`<span class="result-date">${B(n)} — </span>`:""}${r}
          </div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${It}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function Tt(e){return e>=1e6?(e/1e6).toFixed(1).replace(/\.0$/,"")+"M":e>=1e3?(e/1e3).toFixed(1).replace(/\.0$/,"")+"K":e.toLocaleString()}function Ht(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const s=e.dataset.menuIndex,a=document.getElementById(`domain-menu-${s}`),n=e.closest(".search-result"),r=(n==null?void 0:n.dataset.domain)||"";if(!a)return;if(!a.classList.contains("hidden")){a.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),a.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${q(r)}">
          ${Et}
          <span>Boost ${B(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${q(r)}">
          ${_t}
          <span>Lower ${B(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${q(r)}">
          ${Mt}
          <span>Block ${B(r)}</span>
        </button>
      `,a.classList.remove("hidden"),a.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const l=o.dataset.action||"",c=o.dataset.domain||"";try{await w.setPreference(c,l),a.classList.add("hidden"),Nt(`${l.charAt(0).toUpperCase()+l.slice(1)}ed ${c}`)}catch(d){console.error("Failed to set preference:",d)}})});const i=o=>{!a.contains(o.target)&&o.target!==e&&(a.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function Nt(e){const t=document.getElementById("toast");t&&t.remove();const s=document.createElement("div");s.id="toast",s.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",s.textContent=e,document.body.appendChild(s),setTimeout(()=>{s.style.opacity="0",setTimeout(()=>s.remove(),300)},2e3)}function Rt(e){try{const s=new URL(e).pathname.split("/").filter(Boolean);return s.length===0?"":" > "+s.map(a=>B(decodeURIComponent(a))).join(" > ")}catch{return""}}function Ot(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60*24));return n===0?"Today":n===1?"1 day ago":n<7?`${n} days ago`:n<30?`${Math.floor(n/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function B(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function q(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const jt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',qt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',Ft='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',Pt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',Ut='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',zt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',Vt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',Dt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function Gt(e){switch(e.type){case"calculator":return Kt(e);case"unit_conversion":return Wt(e);case"currency":return Yt(e);case"weather":return Jt(e);case"definition":return Zt(e);case"time":return Qt(e);default:return Xt(e)}}function Kt(e){const t=e.data||{},s=t.expression||e.query||"",a=t.formatted||t.result||e.result||"";return`
    <div class="instant-card calculator">
      <div class="flex items-center gap-2 text-tertiary">
        ${jt}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${m(String(a))}</div>
      <div class="instant-sub">${m(s)}</div>
    </div>
  `}function Wt(e){const t=e.data||{},s=t.from_value??"",a=t.from_unit??"",n=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card conversion">
      <div class="flex items-center gap-2 text-tertiary">
        ${qt}
        <span class="instant-type">Unit Conversion${i?` - ${m(i)}`:""}</span>
      </div>
      <div class="instant-result">${m(String(n))} ${m(r)}</div>
      <div class="instant-sub">${m(String(s))} ${m(a)}</div>
    </div>
  `}function Yt(e){const t=e.data||{},s=t.from_value??"",a=t.from_currency??"",n=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card currency">
      <div class="flex items-center gap-2 text-tertiary">
        ${Ft}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${m(String(n))} ${m(r)}</div>
      ${i?`<div class="currency-rate">1 ${m(a)} = ${m(String(i))} ${m(r)}</div>`:""}
      <div class="currency-updated">${m(String(s))} ${m(a)}</div>
    </div>
  `}function Jt(e){const t=e.data||{},s=t.location||"",a=t.temperature??"",n=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=Pt;n.includes("cloud")||n.includes("overcast")?o=Ut:(n.includes("rain")||n.includes("drizzle")||n.includes("storm"))&&(o=zt);const l=[];return r&&l.push(`Humidity: ${m(r)}`),i&&l.push(`Wind: ${m(i)}`),`
    <div class="instant-card weather">
      <div class="weather-main">
        <div class="weather-icon">${o}</div>
        <div class="weather-temp">${m(String(a))}<sup>°</sup></div>
      </div>
      <div class="weather-details">
        <div class="weather-condition">${m(t.condition||"")}</div>
        <div class="weather-location">${m(s)}</div>
        ${l.length>0?`<div class="weather-meta">${l.join(" · ")}</div>`:""}
      </div>
    </div>
  `}function Zt(e){const t=e.data||{},s=t.word||e.query||"",a=t.phonetic||"",n=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[],o=t.example||"";return`
    <div class="instant-card definition">
      <div class="flex items-center gap-2 text-tertiary">
        ${Vt}
        <span class="instant-type">Definition</span>
      </div>
      <div class="word">
        <span>${m(s)}</span>
        <button class="pronunciation-btn" title="Listen to pronunciation" aria-label="Listen to pronunciation">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/></svg>
        </button>
      </div>
      ${a?`<div class="phonetic">${m(a)}</div>`:""}
      ${n?`<div class="part-of-speech">${m(n)}</div>`:""}
      ${r.length>0?r.map((c,d)=>`<div class="definition-text">${d+1}. ${m(c)}</div>`).join(""):""}
      ${o?`<div class="definition-example">"${m(o)}"</div>`:""}
      ${i.length>0?`<div class="mt-3 text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(c=>m(c)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function Qt(e){const t=e.data||{},s=t.location||"",a=t.time||"",n=t.date||"",r=t.timezone||"";return`
    <div class="instant-card time">
      <div class="flex items-center gap-2 text-tertiary">
        ${Dt}
        <span class="instant-type">Time</span>
      </div>
      <div class="time-display">${m(a)}</div>
      <div class="time-location">${m(s)}</div>
      <div class="time-date">${m(n)}</div>
      ${r?`<div class="time-timezone">${m(r)}</div>`:""}
    </div>
  `}function Xt(e){return`
    <div class="instant-card">
      <div class="instant-type">${m(e.type)}</div>
      <div class="instant-result">${m(e.result)}</div>
    </div>
  `}function m(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const es='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function ts(e){const t=e.image?`<img class="kp-image" src="${me(e.image)}" alt="${me(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",s=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${z(r.label)}</td>
                <td class="fact-value">${z(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",a=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${me(r.url)}" target="_blank" rel="noopener">
              ${es}
              <span>${z(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",n=e.source?`<div class="kp-source">Source: ${z(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${z(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${z(e.subtitle)}</div>`:""}
      <div class="kp-description">${z(e.description)}</div>
      ${s}
      ${a}
      ${n}
    </div>
  `}function z(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function me(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ss='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',as='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function ns(e){const{currentPage:t,hasMore:s,totalResults:a,perPage:n}=e,r=Math.min(Math.ceil(a/n),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const l=[];for(let p=i;p<=o;p++)l.push(p);const c=rs(t),d=t<=1?"disabled":"",u=!s&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${c}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${d}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${ss}
          </button>
          ${l.map(p=>`
            <button class="pagination-btn ${p===t?"active":""}" data-page="${p}">
              ${p}
            </button>
          `).join("")}
          <button class="pagination-btn ${u}" data-page="${t+1}" ${!s&&t>=r?"disabled":""} aria-label="Next page">
            ${as}
          </button>
        </div>
      </div>
    </div>
  `}function rs(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],s=["M","i","z","u"],a=Math.min(e-1,6);let n=[s[0]];for(let r=0;r<1+a;r++)n.push("i");n.push("z");for(let r=0;r<1+a;r++)n.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${n.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function is(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(s=>{s.addEventListener("click",()=>{const a=parseInt(s.dataset.page||"1");isNaN(a)||s.disabled||(e(a),window.scrollTo({top:0,behavior:"smooth"}))})})}const os='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',ls='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',cs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',ds='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>',us='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2"/><path d="M8.5 2h7"/><path d="M7 16h10"/></svg>',ps='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',hs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',gs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',vs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',ms='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>';function N(e){const{query:t,active:s}=e,a=encodeURIComponent(t),n=[{id:"all",label:"All",icon:os,href:`/search?q=${a}`},{id:"images",label:"Images",icon:ls,href:`/images?q=${a}`},{id:"videos",label:"Videos",icon:cs,href:`/videos?q=${a}`},{id:"news",label:"News",icon:ds,href:`/news?q=${a}`}],r=[{id:"science",label:"Science",icon:us,href:`/science?q=${a}`},{id:"code",label:"Code",icon:ps,href:`/code?q=${a}`},{id:"music",label:"Music",icon:hs,href:`/music?q=${a}`},{id:"social",label:"Social",icon:gs,href:`/social?q=${a}`},{id:"maps",label:"Maps",icon:vs,href:`/maps?q=${a}`}],i=r.find(o=>o.id===s);return`
    <div class="search-tabs-container">
      <div class="search-tabs" id="search-tabs">
        ${n.map(o=>`
          <a class="search-tab ${o.id===s?"active":""}" href="${o.href}" data-link data-tab="${o.id}">
            ${o.icon}
            <span>${o.label}</span>
          </a>
        `).join("")}
        <div class="search-tab-more">
          <button class="search-tab ${i?"active":""}" id="more-tabs-btn" type="button">
            ${i?i.icon:ms}
            <span>${i?i.label:"More"}</span>
          </button>
          <div class="more-tabs-dropdown hidden" id="more-tabs-dropdown">
            ${r.map(o=>`
              <a class="more-tab-item ${o.id===s?"active":""}" href="${o.href}" data-link>
                ${o.icon}
                <span>${o.label}</span>
              </a>
            `).join("")}
          </div>
        </div>
      </div>
    </div>
  `}function R(){const e=document.getElementById("more-tabs-btn"),t=document.getElementById("more-tabs-dropdown");e&&t&&(e.addEventListener("click",s=>{s.stopPropagation(),t.classList.toggle("hidden")}),document.addEventListener("click",()=>{t.classList.add("hidden")}))}const fs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>';function ys(e){return!e||e.length===0?"":`
    <div class="paa-container">
      <h3 class="paa-title">People also ask</h3>
      <div class="paa-list">
        ${e.map((t,s)=>`
          <div class="paa-item" data-index="${s}">
            <button class="paa-question" aria-expanded="false">
              <span>${fe(t.question)}</span>
              <span class="paa-chevron">${fs}</span>
            </button>
            <div class="paa-answer hidden">
              ${t.answer?`<p class="paa-answer-text">${fe(t.answer)}</p>`:'<p class="paa-loading">Loading...</p>'}
              ${t.source&&t.url?`
                <a href="${ws(t.url)}" target="_blank" class="paa-source">
                  ${fe(t.source)}
                </a>
              `:""}
            </div>
          </div>
        `).join("")}
      </div>
    </div>
  `}function xs(){const e=document.querySelector(".paa-container");e&&e.querySelectorAll(".paa-item").forEach(t=>{const s=t.querySelector(".paa-question"),a=t.querySelector(".paa-answer");s==null||s.addEventListener("click",()=>{var r;const n=s.getAttribute("aria-expanded")==="true";e.querySelectorAll(".paa-item").forEach(i=>{var o,l,c;i!==t&&((o=i.querySelector(".paa-question"))==null||o.setAttribute("aria-expanded","false"),(l=i.querySelector(".paa-answer"))==null||l.classList.add("hidden"),(c=i.querySelector(".paa-chevron"))==null||c.classList.remove("rotated"))}),s.setAttribute("aria-expanded",String(!n)),a==null||a.classList.toggle("hidden",n),(r=t.querySelector(".paa-chevron"))==null||r.classList.toggle("rotated",!n)})})}function fe(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ws(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const _e='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',bs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',$s='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',ks='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21c0 1 0 1 1 1z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3c0 1 0 1 1 1z"/></svg>',Cs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',Me='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',be=[{value:"",label:"Any time"},{value:"hour",label:"Past hour"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}],$e=[{value:"",label:"Any region"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"jp",label:"Japan"},{value:"in",label:"India"},{value:"br",label:"Brazil"}];function Ss(e={}){var u,p;const{timeRange:t="",region:s="",verbatim:a=!1,site:n=""}=e,r=((u=be.find(f=>f.value===t))==null?void 0:u.label)||"Any time",i=((p=$e.find(f=>f.value===s))==null?void 0:p.label)||"Any region",o=t!=="",l=s!=="",c=n!=="",d=o||l||a||c;return`
    <div class="search-tools" id="search-tools">
      <div class="search-tools-row">
        <!-- Time Filter -->
        <div class="search-tool-dropdown" data-tool="time">
          <button class="search-tool-btn ${o?"active":""}" type="button">
            ${bs}
            <span class="search-tool-label">${Z(r)}</span>
            ${_e}
          </button>
          <div class="search-tool-menu hidden">
            ${be.map(f=>`
              <button class="search-tool-option ${f.value===t?"selected":""}" data-value="${f.value}">
                ${Z(f.label)}
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Region Filter -->
        <div class="search-tool-dropdown" data-tool="region">
          <button class="search-tool-btn ${l?"active":""}" type="button">
            ${$s}
            <span class="search-tool-label">${Z(i)}</span>
            ${_e}
          </button>
          <div class="search-tool-menu hidden">
            ${$e.map(f=>`
              <button class="search-tool-option ${f.value===s?"selected":""}" data-value="${f.value}">
                ${Z(f.label)}
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Verbatim Toggle -->
        <button class="search-tool-toggle ${a?"active":""}" data-tool="verbatim" type="button">
          ${ks}
          <span>Verbatim</span>
        </button>

        <!-- Site Search -->
        <div class="search-tool-site" data-tool="site">
          <div class="search-tool-site-input ${c?"has-value":""}">
            ${Cs}
            <input
              type="text"
              id="site-filter-input"
              placeholder="Filter by site..."
              value="${Z(n)}"
              autocomplete="off"
              spellcheck="false"
            />
            ${c?`
              <button class="search-tool-site-clear" type="button" aria-label="Clear site filter">
                ${Me}
              </button>
            `:""}
          </div>
        </div>

        <!-- Clear All Filters -->
        ${d?`
          <button class="search-tool-clear" id="clear-all-filters" type="button">
            ${Me}
            <span>Clear filters</span>
          </button>
        `:""}
      </div>
    </div>
  `}function Ls(e){var l,c;const t=document.getElementById("search-tools");if(!t)return;const s={timeRange:"",region:"",verbatim:!1,site:""},a=t.querySelector('[data-tool="time"]'),n=t.querySelector('[data-tool="region"]'),r=t.querySelector('[data-tool="verbatim"]'),i=t.querySelector("#site-filter-input");if(a){const d=a.querySelector(".search-tool-option.selected");s.timeRange=(d==null?void 0:d.dataset.value)||""}if(n){const d=n.querySelector(".search-tool-option.selected");s.region=(d==null?void 0:d.dataset.value)||""}r!=null&&r.classList.contains("active")&&(s.verbatim=!0),i&&(s.site=i.value),t.querySelectorAll(".search-tool-dropdown").forEach(d=>{const u=d.querySelector(".search-tool-btn"),p=d.querySelector(".search-tool-menu"),f=d.dataset.tool;u==null||u.addEventListener("click",$=>{$.stopPropagation(),t.querySelectorAll(".search-tool-menu").forEach(b=>{b!==p&&b.classList.add("hidden")}),p==null||p.classList.toggle("hidden")}),p==null||p.querySelectorAll(".search-tool-option").forEach($=>{$.addEventListener("click",()=>{var g,h;const b=$.dataset.value||"";p.querySelectorAll(".search-tool-option").forEach(x=>{x.classList.toggle("selected",x===$)});const C=b!=="";u==null||u.classList.toggle("active",C);const M=u==null?void 0:u.querySelector(".search-tool-label");M&&(f==="time"?(M.textContent=((g=be.find(x=>x.value===b))==null?void 0:g.label)||"Any time",s.timeRange=b):f==="region"&&(M.textContent=((h=$e.find(x=>x.value===b))==null?void 0:h.label)||"Any region",s.region=b)),p==null||p.classList.add("hidden"),e({...s})})})}),r==null||r.addEventListener("click",()=>{s.verbatim=!s.verbatim,r.classList.toggle("active",s.verbatim),e({...s})});let o;i==null||i.addEventListener("input",()=>{clearTimeout(o),o=setTimeout(()=>{s.site=i.value.trim(),ye(t,s.site),e({...s})},500)}),i==null||i.addEventListener("keydown",d=>{d.key==="Enter"&&(d.preventDefault(),clearTimeout(o),s.site=i.value.trim(),ye(t,s.site),e({...s}))}),(l=t.querySelector(".search-tool-site-clear"))==null||l.addEventListener("click",()=>{s.site="",i&&(i.value=""),ye(t,""),e({...s})}),(c=t.querySelector("#clear-all-filters"))==null||c.addEventListener("click",()=>{s.timeRange="",s.region="",s.verbatim=!1,s.site="",e({...s})}),document.addEventListener("click",()=>{t.querySelectorAll(".search-tool-menu").forEach(d=>d.classList.add("hidden"))})}function ye(e,t){const s=e.querySelector(".search-tool-site-input");s==null||s.classList.toggle("has-value",t!=="")}function Z(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Is='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Es(e,t={}){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Is}
          </a>
        </div>
        <div class="search-tabs-row">
          ${N({query:e,active:"all"})}
        </div>
        <!-- Search Tools Bar -->
        ${Ss(t)}
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="search-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function _s(e,t,s){const a=parseInt(s.page||"1"),n={timeRange:s.time_range||"",region:s.region||"",verbatim:s.verbatim==="1",site:s.site||""},r=G.get().settings;_(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),R(),Ls(i=>{const o=Ge(t,i);e.navigate(o)}),t&&H(t),Ms(e,t,a,n,r.results_per_page)}function Ge(e,t,s){const a=new URLSearchParams;return a.set("q",e),s&&s>1&&a.set("page",String(s)),t.timeRange&&a.set("time_range",t.timeRange),t.region&&a.set("region",t.region),t.verbatim&&a.set("verbatim","1"),t.site&&a.set("site",t.site),`/search?${a.toString()}`}async function Ms(e,t,s,a,n){const r=document.getElementById("search-content");if(!r||!t)return;let i=t;a.site&&(i=`site:${a.site} ${t}`);try{const o=await w.search(i,{page:s,per_page:n,time_range:a.timeRange||void 0,region:a.region||void 0,verbatim:a.verbatim||void 0});if(o.redirect){window.location.href=o.redirect;return}Bs(r,e,o,t,s,a)}catch(o){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${Q(String(o))}</p>
      </div>
    `}}function Bs(e,t,s,a,n,r){var b;const i=s.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(s.corrected_query)}" data-link class="text-link font-medium">${Q(s.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(a)}&exact=1" data-link class="text-link">${Q(a)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${As(s.total_results)} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,l=s.instant_answer?Gt(s.instant_answer):"",c=((b=s.related_searches)==null?void 0:b.slice(0,4).map(C=>({question:C,answer:void 0})))||[],d=c.length>0?ys(c):"",u=s.results.length>0?s.results.map((C,M)=>At(C,M)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${Q(a)}</strong>"</div>`,p=s.related_searches&&s.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="related-searches-pills">
          ${s.related_searches.map(C=>`
            <a href="/search?q=${encodeURIComponent(C)}" data-link class="related-pill">${Q(C)}</a>
          `).join("")}
        </div>
      </div>
    `:"",f=ns({currentPage:n,hasMore:s.has_more,totalResults:s.total_results,perPage:s.per_page}),$=s.knowledge_panel?ts(s.knowledge_panel):"";e.innerHTML=`
    <div class="search-results-layout">
      <div class="search-results-main">
        ${i}
        ${o}
        ${l}
        ${d}
        ${u}
        ${p}
        ${f}
      </div>
      ${$?`<aside class="search-results-sidebar">${$}</aside>`:""}
    </div>
  `,Ht(),xs(),is(C=>{const M=Ge(a,r,C);t.navigate(M)})}function As(e){return e.toLocaleString("en-US")}function Q(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Ts='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Hs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Be='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Ae='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',Ns='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',Ke='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',Rs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>',Os='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>';let ae="",A={},ne=1,Y=!1,P=!0,T=[],X=!1,ke=[],oe=null;function js(e){return`
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 shadow-sm">
        <div class="flex items-center gap-4 px-4 py-2">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[600px] flex items-center gap-2">
            ${E({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${Hs}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Ts}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1">
          ${N({query:e,active:"images"})}
          <button id="tools-btn" class="tools-btn ml-4">
            ${Ns}
            <span>Tools</span>
            ${Ke}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${qs()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${Rs}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${Os}</button>
        </div>
      </div>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 px-2">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-overlay"></div>
          <div class="preview-container">
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${Be}</button>
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
            <button id="reverse-modal-close" class="modal-close">${Be}</button>
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
  `}function qs(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${Ke}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"?" active":""}" data-value="${s}">
                ${ue(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function ue(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function Fs(e,t,s){if(ae=t,A={},ne=1,T=[],P=!0,X=!1,ke=[],_(a=>{e.navigate(`/images?q=${encodeURIComponent(a)}`)}),R(),t&&H(t),Ps(),Us(),Ds(),Ws(),zs(e),(s==null?void 0:s.reverse)==="1"){const a=document.getElementById("reverse-modal");a&&a.classList.remove("hidden")}Ce(t,A)}function Ps(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{X=!X,t.classList.toggle("hidden",!X),e.classList.toggle("active",X)})}function Us(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(a=>{a.addEventListener("click",n=>{n.stopPropagation();const r=a.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i==null||i.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(a=>{a.addEventListener("click",()=>{const n=a.closest(".filter-dropdown"),r=n==null?void 0:n.dataset.dropdown,i=a.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);!r||!i||!o||(n.querySelectorAll(".filter-option").forEach(l=>l.classList.remove("active")),a.classList.add("active"),i==="any"?(delete A[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=ue(r,"any").replace("Any ","")):(A[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=ue(r,i)),n.classList.add("hidden"),Te(),ne=1,T=[],P=!0,Ce(ae,A))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(a=>a.classList.add("hidden"))});const s=document.getElementById("clear-filters");s&&s.addEventListener("click",()=>{A={},ne=1,T=[],P=!0,t.querySelectorAll(".filter-chip").forEach(a=>{const n=a.dataset.filter;a.classList.remove("has-value"),a.querySelector(".filter-chip-label").textContent=ue(n,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(a=>{a.querySelectorAll(".filter-option").forEach((n,r)=>{n.classList.toggle("active",r===0)})}),Te(),Ce(ae,A)})}function Te(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(A).length===0)}function zs(e){const t=document.getElementById("related-searches");if(!t)return;t.addEventListener("click",r=>{const i=r.target.closest(".related-chip");if(i){const o=i.getAttribute("data-query");o&&e.navigate(`/images?q=${encodeURIComponent(o)}`)}});const s=t.querySelector(".related-scroll-left"),a=t.querySelector(".related-scroll-right"),n=t.querySelector(".related-searches-list");s&&a&&n&&(s.addEventListener("click",()=>{n.scrollBy({left:-200,behavior:"smooth"})}),a.addEventListener("click",()=>{n.scrollBy({left:200,behavior:"smooth"})}),n.addEventListener("scroll",()=>{We()}))}function We(){const e=document.getElementById("related-searches");if(!e)return;const t=e.querySelector(".related-searches-list"),s=e.querySelector(".related-scroll-left"),a=e.querySelector(".related-scroll-right");!t||!s||!a||(s.classList.toggle("hidden",t.scrollLeft<=0),a.classList.toggle("hidden",t.scrollLeft>=t.scrollWidth-t.clientWidth-10))}function Vs(e){const t=document.getElementById("related-searches");if(!t)return;if(!e||e.length===0){t.classList.add("hidden");return}const s=t.querySelector(".related-searches-list");s&&(s.innerHTML=e.map(a=>`
    <button class="related-chip" data-query="${F(a)}">
      <span class="related-chip-text">${K(a)}</span>
    </button>
  `).join(""),t.classList.remove("hidden"),setTimeout(We,50))}function Ds(e){const t=document.getElementById("reverse-search-btn"),s=document.getElementById("reverse-modal"),a=document.getElementById("reverse-modal-close"),n=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!s||(t.addEventListener("click",()=>s.classList.remove("hidden")),a==null||a.addEventListener("click",()=>s.classList.add("hidden")),s.addEventListener("click",l=>{l.target===s&&s.classList.add("hidden")}),n&&(n.addEventListener("dragover",l=>{l.preventDefault(),n.classList.add("drag-over")}),n.addEventListener("dragleave",()=>n.classList.remove("drag-over")),n.addEventListener("drop",l=>{var d;l.preventDefault(),n.classList.remove("drag-over");const c=(d=l.dataTransfer)==null?void 0:d.files;c&&c[0]&&(He(c[0]),s.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(He(r.files[0]),s.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const l=i.value.trim();l&&(Ne(l),s.classList.add("hidden"))}),i.addEventListener("keydown",l=>{if(l.key==="Enter"){const c=i.value.trim();c&&(Ne(c),s.classList.add("hidden"))}})))}async function He(e,t){const s=document.getElementById("images-content");if(s){if(!e.type.startsWith("image/")){alert("Please select an image file");return}if(e.size>10*1024*1024){alert("Image must be smaller than 10MB");return}s.innerHTML=`
    <div class="flex flex-col items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="mt-3 text-secondary">Uploading and searching...</span>
      <div class="w-48 mt-4 h-1 bg-border rounded-full overflow-hidden">
        <div id="upload-progress" class="h-full bg-blue transition-all duration-300" style="width: 0%"></div>
      </div>
    </div>
  `;try{const a=await Gs(e),n=document.getElementById("upload-progress");n&&(n.style.width="50%");const r=await w.reverseImageSearchByUpload(a);n&&(n.style.width="100%"),Ks(s,a,r)}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${K(String(a))}</p>
      </div>
    `}}}function Gs(e){return new Promise((t,s)=>{const a=new FileReader;a.onload=()=>{const r=a.result.split(",")[1];t(r)},a.onerror=s,a.readAsDataURL(e)})}function Ks(e,t,s){const n=!t.startsWith("http")?`data:image/jpeg;base64,${t}`:t;e.innerHTML=`
    <div class="reverse-results">
      <div class="query-image-section">
        <h3>Search image</h3>
        <img src="${n}" alt="Query image" class="query-image" />
      </div>
      ${s.similar_images.length>0?`
        <div class="similar-images-section">
          <h3>Similar images (${s.similar_images.length})</h3>
          <div class="image-grid">
            ${s.similar_images.map((r,i)=>ge(r,i)).join("")}
          </div>
        </div>
      `:'<div class="py-8 text-secondary">No similar images found.</div>'}
    </div>
  `,e.querySelectorAll(".image-card").forEach(r=>{r.addEventListener("click",()=>{const i=parseInt(r.dataset.imageIndex||"0",10);he(s.similar_images[i])})})}async function Ne(e,t){const s=document.getElementById("images-content");if(s){s.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const a=await w.reverseImageSearch(e);s.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${F(e)}" alt="Query image" class="query-image" />
        </div>
        ${a.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${a.similar_images.length})</h3>
            <div class="image-grid">
              ${a.similar_images.map((n,r)=>ge(n,r)).join("")}
            </div>
          </div>
        `:'<div class="py-8 text-secondary">No similar images found.</div>'}
      </div>
    `,s.querySelectorAll(".image-card").forEach(n=>{n.addEventListener("click",()=>{const r=parseInt(n.dataset.imageIndex||"0",10);he(a.similar_images[r])})})}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${K(String(a))}</p>
      </div>
    `}}}function Ws(){const e=document.getElementById("preview-panel"),t=document.getElementById("preview-close"),s=e==null?void 0:e.querySelector(".preview-overlay");t==null||t.addEventListener("click",xe),s==null||s.addEventListener("click",xe),document.addEventListener("keydown",a=>{a.key==="Escape"&&xe()})}function he(e){const t=document.getElementById("preview-panel"),s=document.getElementById("preview-image"),a=document.getElementById("preview-details");if(!t||!s||!a)return;s.src=e.url,s.alt=e.title;const n=e.width&&e.height&&e.width>0&&e.height>0;a.innerHTML=`
    <div class="preview-header">
      <img src="${F(e.thumbnail_url||e.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${K(e.title||"Untitled")}</h3>
        <a href="${F(e.source_url)}" target="_blank" class="preview-domain">${K(e.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${n?`<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${e.width} × ${e.height}</span></div>`:""}
      ${e.format?`<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${e.format.toUpperCase()}</span></div>`:""}
    </div>
    <div class="preview-actions">
      <a href="${F(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${Ae}
      </a>
      <a href="${F(e.url)}" target="_blank" class="preview-btn">
        View full image ${Ae}
      </a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function xe(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function Ys(){oe&&oe.disconnect();const e=document.getElementById("images-content");if(!e)return;const t=document.getElementById("scroll-sentinel");t&&t.remove();const s=document.createElement("div");s.id="scroll-sentinel",s.className="scroll-sentinel",e.appendChild(s),oe=new IntersectionObserver(a=>{a[0].isIntersecting&&!Y&&P&&ae&&Js()},{rootMargin:"400px"}),oe.observe(s)}async function Js(){if(Y||!P)return;Y=!0,ne++;const e=document.getElementById("scroll-sentinel");e&&(e.innerHTML='<div class="loading-more"><div class="spinner-sm"></div></div>');try{const t=await w.searchImages(ae,{...A,page:ne}),s=t.results;P=t.has_more,T=[...T,...s];const a=document.querySelector(".image-grid");if(a&&s.length>0){const n=T.length-s.length,r=s.map((i,o)=>ge(i,n+o)).join("");a.insertAdjacentHTML("beforeend",r),a.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);he(T[o])})})}e&&(e.innerHTML=P?"":'<div class="no-more-results">No more images</div>')}catch{e&&(e.innerHTML="")}finally{Y=!1}}async function Ce(e,t){var a;const s=document.getElementById("images-content");if(!(!s||!e)){Y=!0,s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const n=await w.searchImages(e,{...t,page:1,per_page:50}),r=n.results;if(P=n.has_more,T=r,ke=(a=n.related_searches)!=null&&a.length?n.related_searches:Zs(e),Vs(ke),r.length===0){s.innerHTML=`<div class="py-8 text-secondary">No image results found for "<strong>${K(e)}</strong>"</div>`;return}s.innerHTML=`<div class="image-grid">${r.map((i,o)=>ge(i,o)).join("")}</div>`,s.querySelectorAll(".image-card").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);he(T[o])})}),Ys()}catch(n){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${K(String(n))}</p>
      </div>
    `}finally{Y=!1}}}function ge(e,t){return`
    <div class="image-card" data-image-index="${t}">
      <div class="image-card-img">
        <div class="image-placeholder" style="background: #e0e0e0;"></div>
        <img
          src="${F(e.thumbnail_url||e.url)}"
          alt="${F(e.title)}"
          loading="lazy"
          class="image-lazy"
          onload="this.classList.add('loaded'); this.previousElementSibling.style.display='none';"
          onerror="this.closest('.image-card').style.display='none'"
        />
      </div>
    </div>
  `}function K(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function F(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Zs(e){const t=e.toLowerCase().trim().split(/\s+/).filter(i=>i.length>1);if(t.length===0)return[];const s=[],a=["wallpaper","hd","4k","aesthetic","cute","beautiful","background","art","photography","design","illustration","vintage","modern","minimalist","colorful","dark","light"],n={cat:["kitten","cats playing","black cat","tabby cat","cat meme"],dog:["puppy","dogs playing","golden retriever","german shepherd","dog meme"],nature:["forest","mountains","ocean","sunset nature","flowers"],food:["dessert","healthy food","breakfast","dinner","food photography"],car:["sports car","luxury car","vintage car","car interior","supercar"],house:["modern house","interior design","living room","bedroom design","architecture"],city:["skyline","night city","urban photography","street photography","downtown"]},r=t.slice(0,2).join(" ");for(const i of a)!e.includes(i)&&s.length<4&&s.push(`${r} ${i}`);for(const[i,o]of Object.entries(n))if(t.some(l=>l.includes(i)||i.includes(l))){for(const l of o)!s.includes(l)&&s.length<8&&s.push(l);break}return t.length>=2&&s.length<8&&s.push(t.reverse().join(" ")),s.length<4&&s.push(`${r} images`,`${r} photos`,`best ${r}`),s.slice(0,8)}const Qs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Xs(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Qs}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${N({query:e,active:"videos"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="videos-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function ea(e,t){_(s=>{e.navigate(`/videos?q=${encodeURIComponent(s)}`)}),R(),t&&H(t),ta(t)}async function ta(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const s=await w.searchVideos(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${W(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} video results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        ${a.map(n=>sa(n)).join("")}
      </div>
    `}catch(s){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${W(String(s))}</p>
      </div>
    `}}function sa(e){var r;const t=((r=e.thumbnail)==null?void 0:r.url)||"",s=e.views?aa(e.views):"",a=e.published?na(e.published):"",n=[e.channel,s,a].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${le(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${le(t)}" alt="${le(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${W(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${le(e.url)}" target="_blank" rel="noopener">${W(e.title)}</a>
        </div>
        <div class="video-meta">${W(n)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${W(e.platform)}</div>`:""}
      </div>
    </div>
  `}function aa(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function na(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60*24));return n===0?"Today":n===1?"1 day ago":n<7?`${n} days ago`:n<30?`${Math.floor(n/7)} weeks ago`:n<365?`${Math.floor(n/30)} months ago`:`${Math.floor(n/365)} years ago`}catch{return e}}function W(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function le(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ra='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function ia(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ra}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${N({query:e,active:"news"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="news-content" class="px-4 lg:px-8 py-6">
          <div class="max-w-[900px]">
            <div class="flex items-center justify-center py-16">
              <div class="spinner"></div>
            </div>
          </div>
        </div>
      </main>
    </div>
  `}function oa(e,t){_(s=>{e.navigate(`/news?q=${encodeURIComponent(s)}`)}),R(),t&&H(t),la(t)}async function la(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const s=await w.searchNews(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No news results found for "<strong>${ee(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-6">
          About ${s.total_results.toLocaleString()} news results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-4">
          ${a.map(n=>ca(n)).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load news results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${ee(String(s))}</p>
        </div>
      </div>
    `}}function ca(e){var a;const t=((a=e.thumbnail)==null?void 0:a.url)||"",s=e.published_date?da(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${ee(e.source||e.domain)}
          ${s?` · ${ee(s)}`:""}
        </div>
        <div class="news-title">
          <a href="${Re(e.url)}" target="_blank" rel="noopener">${ee(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${Re(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function da(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<24?`${n}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function ee(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Re(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ua='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Ye='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>',pa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',ha='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>',Oe='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',ga='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',va='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>',ma='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>',fa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>',ya='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',xa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>',wa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',ba='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',Je=[{id:"top",label:"Top Stories",icon:ba},{id:"world",label:"World",icon:ga},{id:"nation",label:"U.S.",icon:Ye},{id:"business",label:"Business",icon:va},{id:"technology",label:"Technology",icon:ma},{id:"entertainment",label:"Entertainment",icon:fa},{id:"sports",label:"Sports",icon:ya},{id:"science",label:"Science",icon:xa},{id:"health",label:"Health",icon:wa}];function $a(){const e=new Date().toLocaleDateString("en-US",{weekday:"long",month:"long",day:"numeric"});return`
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
            ${Ye}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${pa}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${ha}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${Je.map(t=>`
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
            ${E({size:"sm"})}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${Oe}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${ua}
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
                ${Oe}
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
  `}function ka(e,t){_(n=>{e.navigate(`/news?q=${encodeURIComponent(n)}`)});const s=document.getElementById("menu-toggle"),a=document.querySelector(".news-sidebar");s&&a&&s.addEventListener("click",()=>{a.classList.toggle("open")}),Ca()}async function Ca(e){const t=document.getElementById("news-loading"),s=document.getElementById("top-stories-section"),a=document.getElementById("for-you-section"),n=document.getElementById("local-section");try{const r=await w.newsHome();if(t&&(t.style.display="none"),s&&r.topStories.length>0){s.style.display="block";const o=document.getElementById("top-stories-grid");o&&(o.innerHTML=Sa(r.topStories))}if(a&&r.forYou.length>0){a.style.display="block";const o=document.getElementById("for-you-list");o&&(o.innerHTML=r.forYou.slice(0,10).map(l=>_a(l)).join(""))}if(n&&r.localNews.length>0){n.style.display="block";const o=document.getElementById("local-news-scroll");o&&(o.innerHTML=r.localNews.map(l=>Ze(l)).join(""))}const i=document.getElementById("category-sections");if(i&&r.categories){const o=Object.entries(r.categories).filter(([l,c])=>c&&c.length>0).map(([l,c])=>Ma(l,c)).join("");i.innerHTML=o}Ba()}catch(r){t&&(t.innerHTML=`
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `),console.error("Failed to load news:",r)}}function Sa(e){if(e.length===0)return"";const t=e[0],s=e.slice(1,3),a=e.slice(3,9);return`
    <div class="news-featured-row">
      ${La(t)}
      <div class="news-secondary-col">
        ${s.map(n=>Ia(n)).join("")}
      </div>
    </div>
    <div class="news-grid-row">
      ${a.map(n=>Ea(n)).join("")}
    </div>
  `}function La(e){const t=re(e.publishedAt);return`
    <article class="news-card news-card-featured">
      ${e.imageUrl?`<img class="news-card-image" src="${L(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${L(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${S(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${L(e.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${e.id}')">${S(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${S(e.snippet)}</p>
        ${e.clusterId?`<a href="/news-home?story=${e.clusterId}" data-link class="news-full-coverage">Full coverage</a>`:""}
      </div>
    </article>
  `}function Ia(e){const t=re(e.publishedAt);return`
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${L(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${S(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${L(e.url)}" target="_blank" rel="noopener">${S(e.title)}</a>
        </h3>
      </div>
      ${e.imageUrl?`<img class="news-card-thumb" src="${L(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function Ea(e){const t=re(e.publishedAt);return`
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${S(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${L(e.url)}" target="_blank" rel="noopener">${S(e.title)}</a>
        </h3>
      </div>
    </article>
  `}function Ze(e){const t=re(e.publishedAt);return`
    <article class="news-card news-card-compact">
      ${e.imageUrl?`<img class="news-card-thumb-sm" src="${L(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:'<div class="news-card-thumb-placeholder"></div>'}
      <div class="news-card-content">
        <span class="news-source-name">${S(e.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${L(e.url)}" target="_blank" rel="noopener">${S(e.title)}</a>
        </h4>
        <span class="news-time">${t}</span>
      </div>
    </article>
  `}function _a(e){const t=re(e.publishedAt);return`
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${L(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${S(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${L(e.url)}" target="_blank" rel="noopener">${S(e.title)}</a>
        </h3>
        <p class="news-list-snippet">${S(e.snippet)}</p>
      </div>
      ${e.imageUrl?`<img class="news-list-thumb" src="${L(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function Ma(e,t){const s=Je.find(a=>a.id===e);return s?`
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${s.icon}
          <span>${s.label}</span>
        </h2>
        <a href="/news-home?category=${e}" data-link class="news-text-btn">More ${s.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${t.slice(0,5).map(a=>Ze(a)).join("")}
      </div>
    </section>
  `:""}function re(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<24?`${n}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric"})}catch{return""}}function Ba(){document.querySelectorAll(".news-card a, .news-list-item a").forEach(e=>{e.addEventListener("click",function(){const t=this.closest(".news-card, .news-list-item");if(t){const s=t.getAttribute("data-article-id");s&&w.recordNewsRead({id:s,url:this.href,title:this.textContent||"",snippet:"",source:"",sourceUrl:"",publishedAt:"",category:"top",engines:[],score:1}).catch(()=>{})}})})}function S(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function L(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Aa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Ta='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M12 18v-6"/><path d="m9 15 3 3 3-3"/></svg>',Ha='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3"/></svg>';function Na(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Aa}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${N({query:e,active:"science"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="science-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Ra(e,t){_(s=>{e.navigate(`/science?q=${encodeURIComponent(s)}`)}),R(),t&&H(t),Oa(t)}async function Oa(e){const t=document.getElementById("science-content");if(!(!t||!e))try{const s=await w.searchScience(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No academic results found for "<strong>${V(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-4">
          About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-6">
          ${a.map(ja).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load academic results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${V(String(s))}</p>
        </div>
      </div>
    `}}function ja(e){var l,c,d,u,p,f;const t=e,s=((l=t.metadata)==null?void 0:l.authors)||"",a=((c=t.metadata)==null?void 0:c.year)||"",n=(d=t.metadata)==null?void 0:d.citations,r=((u=t.metadata)==null?void 0:u.doi)||"",i=((p=t.metadata)==null?void 0:p.pdf_url)||"",o=((f=t.metadata)==null?void 0:f.source)||qa(e.url);return`
    <article class="paper-card bg-white border border-border rounded-xl p-5 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 bg-blue/10 text-blue rounded-full font-medium">${V(o)}</span>
        ${a?`<span class="text-xs text-tertiary">${V(a)}</span>`:""}
      </div>
      <h3 class="text-lg font-medium text-primary mb-2">
        <a href="${we(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${V(e.title)}</a>
      </h3>
      ${s?`<p class="text-sm text-secondary mb-2">${V(s)}</p>`:""}
      <p class="text-sm text-snippet line-clamp-3 mb-3">${e.snippet||""}</p>
      <div class="flex items-center gap-4 text-xs">
        ${n!==void 0?`<span class="flex items-center gap-1 text-tertiary">${Ha} ${n} citations</span>`:""}
        ${r?`<a href="https://doi.org/${we(r)}" target="_blank" rel="noopener" class="text-tertiary hover:text-blue">DOI: ${V(r)}</a>`:""}
        ${i?`<a href="${we(i)}" target="_blank" rel="noopener" class="flex items-center gap-1 text-blue hover:underline">${Ta} PDF</a>`:""}
      </div>
    </article>
  `}function qa(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function V(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function we(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Fa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Pa='<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',Ua='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><path d="M18 9v1a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V9"/><path d="M12 12v3"/></svg>',za='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>';function Va(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${Fa}
          </a>
        </div>
        <div class="px-4 lg:px-8">
          ${N({query:e,active:"code"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="code-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Da(e,t){_(s=>e.navigate(`/code?q=${encodeURIComponent(s)}`)),R(),t&&H(t),Ga(t)}async function Ga(e){const t=document.getElementById("code-content");if(!(!t||!e))try{const s=await w.searchCode(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No code results found for "${te(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="max-w-[900px] space-y-4">
        ${a.map(Ka).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${te(String(s))}</div>`}}function Ka(e){var l,c,d,u,p,f,$;const t=((l=e.metadata)==null?void 0:l.source)||Ya(e.url),s=(c=e.metadata)==null?void 0:c.stars,a=(d=e.metadata)==null?void 0:d.forks,n=(u=e.metadata)==null?void 0:u.downloads,r=((p=e.metadata)==null?void 0:p.language)||"",i=(f=e.metadata)==null?void 0:f.votes,o=($=e.metadata)==null?void 0:$.answers;return`
    <article class="code-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Wa(t)}">${te(t)}</span>
        ${r?`<span class="text-xs px-2 py-0.5 bg-surface text-secondary rounded-full">${te(r)}</span>`:""}
      </div>
      <h3 class="text-base font-medium text-primary mb-1">
        <a href="${Ja(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${te(e.title)}</a>
      </h3>
      <p class="text-sm text-snippet line-clamp-2 mb-3">${e.content||""}</p>
      <div class="flex items-center gap-4 text-xs text-tertiary">
        ${s!==void 0?`<span class="flex items-center gap-1">${Pa} <span class="text-yellow-500">${ce(s)}</span></span>`:""}
        ${a!==void 0?`<span class="flex items-center gap-1">${Ua} ${ce(a)}</span>`:""}
        ${n!==void 0?`<span class="flex items-center gap-1">${za} ${ce(n)}</span>`:""}
        ${i!==void 0?`<span class="flex items-center gap-1">▲ ${ce(i)}</span>`:""}
        ${o!==void 0?`<span class="flex items-center gap-1">${o} answers</span>`:""}
      </div>
    </article>
  `}function Wa(e){return e.includes("github")?"bg-gray-900 text-white":e.includes("gitlab")?"bg-orange-500 text-white":e.includes("stackoverflow")?"bg-orange-400 text-white":e.includes("npm")?"bg-red-500 text-white":e.includes("pypi")?"bg-blue-500 text-white":e.includes("crates")?"bg-orange-600 text-white":"bg-surface text-secondary"}function ce(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function Ya(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function te(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ja(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Za='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Qa='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',Xa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>';function en(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${Za}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${N({query:e,active:"music"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="music-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function tn(e,t){_(s=>e.navigate(`/music?q=${encodeURIComponent(s)}`)),R(),t&&H(t),sn(t)}async function sn(e){const t=document.getElementById("music-content");if(!(!t||!e))try{const s=await w.searchMusic(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No music results found for "${j(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        ${a.map(an).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${j(String(s))}</div>`}}function an(e){var o,l,c,d,u;const t=((o=e.metadata)==null?void 0:o.source)||rn(e.url),s=((l=e.metadata)==null?void 0:l.artist)||"",a=((c=e.metadata)==null?void 0:c.album)||"",n=((d=e.metadata)==null?void 0:d.duration)||"",r=((u=e.thumbnail)==null?void 0:u.url)||"",i=t.toLowerCase().includes("genius");return`
    <article class="music-card bg-white border border-border rounded-xl overflow-hidden hover:shadow-md transition-shadow">
      <a href="${je(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="relative aspect-square bg-surface">
          ${r?`<img src="${je(r)}" alt="" class="w-full h-full object-cover" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center text-border">${Xa}</div>`}
          <div class="absolute inset-0 bg-black/40 opacity-0 hover:opacity-100 transition-opacity flex items-center justify-center">
            <span class="w-12 h-12 rounded-full bg-white flex items-center justify-center text-primary">${Qa}</span>
          </div>
        </div>
        <div class="p-3">
          <span class="text-xs px-2 py-0.5 rounded-full font-medium ${nn(t)}">${j(t)}</span>
          <h3 class="text-sm font-medium text-primary mt-2 line-clamp-2">${j(e.title)}</h3>
          ${s?`<p class="text-xs text-secondary mt-1">${j(s)}</p>`:""}
          ${a?`<p class="text-xs text-tertiary">${j(a)}</p>`:""}
          ${n?`<p class="text-xs text-tertiary mt-1">${j(n)}</p>`:""}
          ${i&&e.snippet?`<p class="text-xs text-snippet mt-2 line-clamp-2 italic">"${j(e.snippet.slice(0,100))}..."</p>`:""}
        </div>
      </a>
    </article>
  `}function nn(e){return e.toLowerCase().includes("soundcloud")?"bg-orange-500 text-white":e.toLowerCase().includes("bandcamp")?"bg-teal-500 text-white":e.toLowerCase().includes("genius")?"bg-yellow-400 text-black":"bg-surface text-secondary"}function rn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function j(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function je(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const on='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',ln='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m18 15-6-6-6 6"/></svg>',cn='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';function dn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${on}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${N({query:e,active:"social"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="social-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function un(e,t){_(s=>e.navigate(`/social?q=${encodeURIComponent(s)}`)),R(),t&&H(t),pn(t)}async function pn(e){const t=document.getElementById("social-content");if(!(!t||!e))try{const s=await w.searchSocial(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No social results found for "${D(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="max-w-[800px] space-y-4">
        ${a.map(hn).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${D(String(s))}</div>`}}function hn(e){var d;const t=e.metadata||{},s=t.source||mn(e.url),a=t.upvotes||t.score||0,n=t.comments||0,r=t.author||"",i=t.subreddit||"",o=t.published||"",l=((d=e.thumbnail)==null?void 0:d.url)||"",c=e.snippet||"";return`
    <article class="social-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3">
        <!-- Upvote column -->
        <div class="flex flex-col items-center text-tertiary text-sm">
          ${ln}
          <span class="font-medium ${a>0?"text-orange-500":""}">${qe(a)}</span>
        </div>
        <!-- Content -->
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-medium ${gn(s)}">${D(s)}</span>
            ${i?`<span class="text-xs text-blue">r/${D(i)}</span>`:""}
            ${r?`<span class="text-xs text-tertiary">by ${D(r)}</span>`:""}
            ${o?`<span class="text-xs text-tertiary">${vn(o)}</span>`:""}
          </div>
          <h3 class="text-base font-medium text-primary mb-1">
            <a href="${Fe(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${D(e.title)}</a>
          </h3>
          ${c?`<p class="text-sm text-snippet line-clamp-3 mb-2">${D(c)}</p>`:""}
          <div class="flex items-center gap-4 text-xs text-tertiary">
            <span class="flex items-center gap-1">${cn} ${qe(n)} comments</span>
          </div>
        </div>
        <!-- Thumbnail if available -->
        ${l?`
          <img src="${Fe(l)}" alt="" class="w-20 h-20 rounded-lg object-cover flex-shrink-0" loading="lazy" onerror="this.style.display='none'" />
        `:""}
      </div>
    </article>
  `}function gn(e){const t=e.toLowerCase();return t.includes("reddit")?"bg-orange-500 text-white":t.includes("hacker")||t.includes("hn")?"bg-orange-600 text-white":t.includes("mastodon")?"bg-purple-500 text-white":t.includes("lemmy")?"bg-green-500 text-white":"bg-surface text-secondary"}function qe(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function vn(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"just now":n<24?`${n}h ago`:r<7?`${r}d ago`:r<30?`${Math.floor(r/7)}w ago`:`${Math.floor(r/30)}mo ago`}catch{return""}}function mn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function D(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Fe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const fn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Pe='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function yn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${fn}
          </a>
        </div>
        <div class="px-4 lg:px-8">
          ${N({query:e,active:"maps"})}
        </div>
      </header>
      <main class="flex-1 flex flex-col lg:flex-row">
        <!-- Map area -->
        <div id="map-container" class="h-[300px] lg:h-auto lg:flex-1 bg-surface relative">
          <iframe id="map-iframe" class="w-full h-full border-0" src="" title="Map"></iframe>
        </div>
        <!-- Results sidebar -->
        <div id="maps-content" class="lg:w-[400px] lg:border-l border-border overflow-y-auto">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function xn(e,t){_(s=>e.navigate(`/maps?q=${encodeURIComponent(s)}`)),R(),t&&H(t),wn(t)}async function wn(e){var a,n;const t=document.getElementById("maps-content"),s=document.getElementById("map-iframe");if(!(!t||!e)){if(s){const r="https://www.openstreetmap.org/export/embed.html?bbox=-180,-90,180,90&layer=mapnik&marker=0,0";s.src=r}try{const i=(await w.searchMaps(e)).results;if(i.length===0){t.innerHTML=`<div class="p-4 text-secondary">No locations found for "${se(e)}"</div>`;return}const o=i[0],l=((a=o.metadata)==null?void 0:a.lat)||0,c=((n=o.metadata)==null?void 0:n.lon)||0;if(s&&l&&c){const d=`${c-.1},${l-.1},${c+.1},${l+.1}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${d}&layer=mapnik&marker=${l},${c}`}t.innerHTML=`
      <div class="p-4">
        <div class="text-xs text-tertiary mb-4">${i.length} locations found</div>
        <div class="space-y-3">
          ${i.map((d,u)=>bn(d,u)).join("")}
        </div>
      </div>
    `,t.querySelectorAll(".location-card").forEach(d=>{d.addEventListener("click",()=>{const u=d.dataset.lat,p=d.dataset.lon;if(u&&p&&s){const f=`${parseFloat(p)-.05},${parseFloat(u)-.05},${parseFloat(p)+.05},${parseFloat(u)+.05}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${f}&layer=mapnik&marker=${u},${p}`}})})}catch(r){t.innerHTML=`<div class="p-4 text-red text-sm">Failed to load results. ${se(String(r))}</div>`}}}function bn(e,t){var i,o,l;const s=((i=e.metadata)==null?void 0:i.lat)||0,a=((o=e.metadata)==null?void 0:o.lon)||0,n=((l=e.metadata)==null?void 0:l.type)||"place",r=e.content||"";return`
    <article class="location-card bg-white border border-border rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow"
             data-lat="${s}" data-lon="${a}">
      <div class="flex items-start gap-3">
        <span class="flex-shrink-0 w-8 h-8 rounded-full bg-red-500 text-white flex items-center justify-center text-sm font-medium">
          ${t+1}
        </span>
        <div class="flex-1 min-w-0">
          <h3 class="font-medium text-primary text-sm">${se(e.title)}</h3>
          <p class="text-xs text-tertiary mt-0.5 capitalize">${se(n)}</p>
          ${r?`<p class="text-xs text-secondary mt-1 line-clamp-2">${se(r)}</p>`:""}
          <p class="text-xs text-tertiary mt-1">${s.toFixed(5)}, ${a.toFixed(5)}</p>
          <div class="flex items-center gap-2 mt-2">
            <a href="${$n(e.url)}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              View on OSM ${Pe}
            </a>
            <a href="https://www.google.com/maps?q=${s},${a}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              Google Maps ${Pe}
            </a>
          </div>
        </div>
      </div>
    </article>
  `}function se(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function $n(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const kn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',Cn=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],Sn=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function Ln(){const e=G.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${kn}
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
              ${Cn.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${Ue(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${Sn.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${Ue(t.label)}</option>`).join("")}
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
  `}function In(e){const t=document.getElementById("settings-form"),s=document.getElementById("settings-status");t&&t.addEventListener("submit",async a=>{a.preventDefault();const n=new FormData(t),r={safe_search:n.get("safe_search")||"moderate",results_per_page:parseInt(n.get("results_per_page"))||10,region:n.get("region")||"auto",language:n.get("language")||"en",theme:n.get("theme")||"light",open_in_new_tab:n.has("open_in_new_tab"),show_thumbnails:n.has("show_thumbnails")};G.set({settings:r});try{await w.updateSettings(r)}catch{}s&&(s.classList.remove("hidden"),setTimeout(()=>{s.classList.add("hidden")},2e3))})}function Ue(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const En='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',_n='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',Mn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Bn='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function An(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${En}
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
  `}function Tn(e){const t=document.getElementById("clear-all-btn");Hn(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await w.clearHistory(),Le(),t.classList.add("hidden")}catch(s){console.error("Failed to clear history:",s)}})}async function Hn(e){const t=document.getElementById("history-content"),s=document.getElementById("clear-all-btn");if(t)try{const a=await w.getHistory();if(a.length===0){Le();return}s&&s.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${a.map(n=>Nn(n)).join("")}
      </div>
    `,Rn(e)}catch(a){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${Se(String(a))}</p>
      </div>
    `}}function Nn(e){const t=On(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${ze(e.id)}">
      <span class="text-light flex-shrink-0">${Mn}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${Se(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${Se(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${ze(e.id)}" aria-label="Delete">
        ${_n}
      </button>
    </div>
  `}function Rn(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async s=>{s.preventDefault(),s.stopPropagation();const a=t.dataset.deleteId||"",n=t.closest(".history-item");try{await w.deleteHistoryItem(a),n&&n.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){Le();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function Le(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${Bn}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function On(e){try{const t=new Date(e),s=new Date,a=s.getTime()-t.getTime(),n=Math.floor(a/(1e3*60)),r=Math.floor(a/(1e3*60*60)),i=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<60?`${n}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==s.getFullYear()?"numeric":void 0})}catch{return e}}function Se(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ze(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const k=document.getElementById("app");if(!k)throw new Error("App container not found");const y=new et;y.addRoute("",(e,t)=>{k.innerHTML=vt(),mt(y)});y.addRoute("search",(e,t)=>{const s=t.q||"",a={timeRange:t.time_range||"",region:t.region||"",verbatim:t.verbatim==="1",site:t.site||""};k.innerHTML=Es(s,a),_s(y,s,t)});y.addRoute("images",(e,t)=>{const s=t.q||"";k.innerHTML=js(s),Fs(y,s,t)});y.addRoute("videos",(e,t)=>{const s=t.q||"";k.innerHTML=Xs(s),ea(y,s)});y.addRoute("news",(e,t)=>{const s=t.q||"";k.innerHTML=ia(s),oa(y,s)});y.addRoute("news-home",(e,t)=>{k.innerHTML=$a(),ka(y)});y.addRoute("science",(e,t)=>{const s=t.q||"";k.innerHTML=Na(s),Ra(y,s)});y.addRoute("code",(e,t)=>{const s=t.q||"";k.innerHTML=Va(s),Da(y,s)});y.addRoute("music",(e,t)=>{const s=t.q||"";k.innerHTML=en(s),tn(y,s)});y.addRoute("social",(e,t)=>{const s=t.q||"";k.innerHTML=dn(s),un(y,s)});y.addRoute("maps",(e,t)=>{const s=t.q||"";k.innerHTML=yn(s),xn(y,s)});y.addRoute("settings",(e,t)=>{k.innerHTML=Ln(),In()});y.addRoute("history",(e,t)=>{k.innerHTML=An(),Tn(y)});y.setNotFound((e,t)=>{k.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;y.navigate(t.detail.path)});y.start();function jn(){document.addEventListener("keydown",e=>{var a;const t=e.target,s=t.tagName==="INPUT"||t.tagName==="TEXTAREA"||t.isContentEditable;if(e.key==="/"&&!s){e.preventDefault();const n=document.getElementById("search-input");n&&(n.focus(),n.select())}e.key==="Escape"&&(document.querySelectorAll(".modal:not(.hidden), .preview-panel:not(.hidden), .lightbox:not(.hidden)").forEach(i=>{i.classList.add("hidden")}),document.querySelectorAll(".autocomplete-dropdown:not(.hidden), .filter-dropdown:not(.hidden), .filter-pill-dropdown:not(.hidden), .more-tabs-dropdown:not(.hidden), .time-filter-dropdown:not(.hidden), .search-tool-menu:not(.hidden)").forEach(i=>{i.classList.add("hidden")}),document.body.style.overflow="",((a=document.activeElement)==null?void 0:a.id)==="search-input"&&document.activeElement.blur()),e.key==="?"&&!s&&qn()})}function qn(){var t;let e=document.getElementById("keyboard-shortcuts-help");if(e){e.classList.toggle("hidden");return}e=document.createElement("div"),e.id="keyboard-shortcuts-help",e.className="modal",e.innerHTML=`
    <div class="modal-content" style="max-width: 400px;">
      <div class="modal-header">
        <h2>Keyboard Shortcuts</h2>
        <button class="modal-close" id="shortcuts-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
        </button>
      </div>
      <div class="modal-body">
        <div class="shortcuts-list">
          <div class="shortcut-item">
            <kbd>/</kbd>
            <span>Focus search box</span>
          </div>
          <div class="shortcut-item">
            <kbd>Escape</kbd>
            <span>Close modal / unfocus</span>
          </div>
          <div class="shortcut-item">
            <kbd>?</kbd>
            <span>Show this help</span>
          </div>
          <div class="shortcut-item">
            <kbd>&#8593;</kbd> <kbd>&#8595;</kbd>
            <span>Navigate suggestions</span>
          </div>
          <div class="shortcut-item">
            <kbd>Enter</kbd>
            <span>Select / submit</span>
          </div>
        </div>
      </div>
    </div>
  `,document.body.appendChild(e),e.addEventListener("click",s=>{s.target===e&&e.classList.add("hidden")}),(t=document.getElementById("shortcuts-close"))==null||t.addEventListener("click",()=>{e.classList.add("hidden")})}jn();"serviceWorker"in navigator&&window.addEventListener("load",()=>{navigator.serviceWorker.register("/sw.js").then(e=>{console.log("Service Worker registered:",e.scope)}).catch(e=>{console.log("Service Worker registration failed:",e)})});
